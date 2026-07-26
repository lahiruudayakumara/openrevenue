package application

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	audit "github.com/opencorex-org/openrevenue/internal/audit/domain"
	filing "github.com/opencorex-org/openrevenue/internal/filing/domain"
	event "github.com/opencorex-org/openrevenue/internal/integration/domain"
	ledger "github.com/opencorex-org/openrevenue/internal/ledger/domain"
	taxpayer "github.com/opencorex-org/openrevenue/internal/taxpayer/domain"
	registration "github.com/opencorex-org/openrevenue/internal/taxregistration/domain"
	foundation "github.com/opencorex-org/openrevenue/pkg/domain"
	"github.com/opencorex-org/openrevenue/pkg/id"
)

type AssessmentTag struct{}
type PaymentTag struct{}

type Taxpayer = taxpayer.Taxpayer
type Registration = registration.Registration

var (
	ErrNotFound          = errors.New("resource not found")
	ErrConflict          = errors.New("resource conflict")
	ErrForbidden         = errors.New("operation forbidden")
	ErrInvalidTransition = errors.New("invalid state transition")
)

type Authorizer interface {
	Authorize(foundation.Context, string, string) error
}

type allowAllAuthorizer struct{}

func (allowAllAuthorizer) Authorize(foundation.Context, string, string) error { return nil }

type idempotencyRecord struct {
	Fingerprint [32]byte
	Taxpayer    Taxpayer
}

type Assessment struct {
	ID           id.ID[AssessmentTag] `json:"id"`
	TenantID     string               `json:"tenantId"`
	Jurisdiction string               `json:"jurisdiction"`
	ReturnID     string               `json:"returnId"`
	Amount       ledger.Money         `json:"amount"`
}

type Payment struct {
	ID           id.ID[PaymentTag] `json:"id"`
	TenantID     string            `json:"tenantId"`
	Jurisdiction string            `json:"jurisdiction"`
	TaxpayerID   string            `json:"taxpayerId"`
	Amount       ledger.Money      `json:"amount"`
	AllocatedTo  string            `json:"allocatedTo"`
}

type Notification struct{ To, Subject, Body string }

type Notifier interface {
	Send(context.Context, Notification) error
}

type Service struct {
	mu            sync.RWMutex
	clock         foundation.Clock
	notifier      Notifier
	taxpayers     map[string]Taxpayer
	registrations map[string]Registration
	returns       map[string]filing.TaxReturn
	assessments   map[string]Assessment
	payments      map[string]Payment
	entries       []ledger.Entry
	audits        []audit.Event
	events        []event.Event
	idempotency   map[string]idempotencyRecord
	identifiers   map[string]string
	authorizer    Authorizer
}

func New(notifier Notifier) *Service {
	return NewWithClock(notifier, foundation.SystemClock{})
}

func NewWithClock(notifier Notifier, clock foundation.Clock) *Service {
	return NewWithDependencies(notifier, clock, allowAllAuthorizer{})
}

func NewWithDependencies(notifier Notifier, clock foundation.Clock, authorizer Authorizer) *Service {
	if clock == nil {
		panic("application clock is required")
	}
	if authorizer == nil {
		panic("application authorizer is required")
	}
	return &Service{
		clock: clock, notifier: notifier, taxpayers: map[string]Taxpayer{},
		registrations: map[string]Registration{}, returns: map[string]filing.TaxReturn{},
		assessments: map[string]Assessment{}, payments: map[string]Payment{},
		idempotency: map[string]idempotencyRecord{}, identifiers: map[string]string{},
		authorizer: authorizer,
	}
}

func (s *Service) record(scope foundation.Context, action, kind, resourceID string) error {
	event, err := audit.New(scope, action, kind, resourceID, s.clock.Now())
	if err != nil {
		return err
	}
	s.audits = append(s.audits, event)
	return nil
}

func (s *Service) emit(scope foundation.Context, eventType, kind, resourceID string, data map[string]string) error {
	value, err := event.New(scope, eventType, kind, resourceID, s.clock.Now(), data)
	if err != nil {
		return err
	}
	s.events = append(s.events, value)
	return nil
}

func (s *Service) authorize(scope foundation.Context, permission, resourceID string) error {
	if err := s.authorizer.Authorize(scope, permission, resourceID); err != nil {
		return fmt.Errorf("%w: %v", ErrForbidden, err)
	}
	return nil
}

func (s *Service) CreateTaxpayer(
	scope foundation.Context,
	name string,
	rawIdentifier string,
	idempotencyKey string,
) (Taxpayer, error) {
	if err := scope.Validate(); err != nil {
		return Taxpayer{}, err
	}
	if err := s.authorize(scope, "taxpayer:create", ""); err != nil {
		return Taxpayer{}, err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return Taxpayer{}, errors.New("taxpayer name is required")
	}
	if idempotencyKey == "" || len(idempotencyKey) > 128 {
		return Taxpayer{}, errors.New("idempotency key must contain 1-128 characters")
	}
	identifier, err := foundation.NewTaxpayerIdentifier(
		scope.Tenant(),
		scope.Jurisdiction(),
		"DEMO_ID",
		rawIdentifier,
		foundation.UpperAlphanumericNormalizer{},
	)
	if err != nil {
		return Taxpayer{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	idempotencyKey = scope.IsolationKey("idempotency:" + idempotencyKey)
	fingerprint := sha256.Sum256([]byte(name + "\x00" + identifier.Scheme() + "\x00" + identifier.String()))
	if value, ok := s.idempotency[idempotencyKey]; ok {
		if value.Fingerprint != fingerprint {
			return Taxpayer{}, fmt.Errorf("%w: idempotency key was already used for another request", ErrConflict)
		}
		return value.Taxpayer, nil
	}
	identifierKey := scope.IsolationKey("identifier:" + identifier.Scheme() + ":" + identifier.String())
	if _, exists := s.identifiers[identifierKey]; exists {
		return Taxpayer{}, fmt.Errorf("%w: taxpayer identifier already exists", ErrConflict)
	}
	value, err := taxpayer.New(scope, name, identifier, s.clock.Now())
	if err != nil {
		return Taxpayer{}, err
	}
	s.taxpayers[scope.IsolationKey(value.ID.String())] = value
	s.identifiers[identifierKey] = value.ID.String()
	s.idempotency[idempotencyKey] = idempotencyRecord{Fingerprint: fingerprint, Taxpayer: value}
	if err := s.record(scope, "TaxpayerCreated", "taxpayer", value.ID.String()); err != nil {
		return Taxpayer{}, err
	}
	if err := s.emit(scope, "TaxpayerCreated", "taxpayer", value.ID.String(), map[string]string{
		"identifierScheme": value.IdentifierScheme,
	}); err != nil {
		return Taxpayer{}, err
	}
	return value, nil
}

func (s *Service) Register(
	scope foundation.Context,
	taxpayerID string,
	taxType string,
) (Registration, error) {
	if err := scope.Validate(); err != nil {
		return Registration{}, err
	}
	if err := s.authorize(scope, "tax-registration:submit", taxpayerID); err != nil {
		return Registration{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.taxpayers[scope.IsolationKey(taxpayerID)]; !ok {
		return Registration{}, ErrNotFound
	}
	value, err := registration.Submit(scope, taxpayerID, taxType, s.clock.Now())
	if err != nil {
		return Registration{}, err
	}
	s.registrations[scope.IsolationKey(value.ID.String())] = value
	if err := s.record(scope, "TaxRegistrationSubmitted", "tax_registration", value.ID.String()); err != nil {
		return Registration{}, err
	}
	if err := s.emit(scope, "TaxRegistrationSubmitted", "tax_registration", value.ID.String(), map[string]string{
		"taxpayerId": taxpayerID, "taxType": value.TaxType,
	}); err != nil {
		return Registration{}, err
	}
	return value, nil
}

func (s *Service) ApproveRegistration(scope foundation.Context, registrationID string) (Registration, error) {
	if err := scope.Validate(); err != nil {
		return Registration{}, err
	}
	if err := s.authorize(scope, "tax-registration:approve", registrationID); err != nil {
		return Registration{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	key := scope.IsolationKey(registrationID)
	value, ok := s.registrations[key]
	if !ok {
		return Registration{}, ErrNotFound
	}
	if err := value.Approve(scope, s.clock.Now()); err != nil {
		return Registration{}, fmt.Errorf("%w: %v", ErrInvalidTransition, err)
	}
	s.registrations[key] = value
	if err := s.record(scope, "TaxRegistrationApproved", "tax_registration", registrationID); err != nil {
		return Registration{}, err
	}
	if err := s.emit(scope, "TaxRegistrationApproved", "tax_registration", registrationID, map[string]string{
		"taxpayerId": value.TaxpayerID, "taxType": value.TaxType,
	}); err != nil {
		return Registration{}, err
	}
	return value, nil
}

func (s *Service) GetTaxpayer(scope foundation.Context, taxpayerID string) (Taxpayer, error) {
	if err := scope.Validate(); err != nil {
		return Taxpayer{}, err
	}
	if err := s.authorize(scope, "taxpayer:read", taxpayerID); err != nil {
		return Taxpayer{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	value, ok := s.taxpayers[scope.IsolationKey(taxpayerID)]
	if !ok {
		return Taxpayer{}, ErrNotFound
	}
	return value, nil
}

func (s *Service) GetRegistration(scope foundation.Context, registrationID string) (Registration, error) {
	if err := scope.Validate(); err != nil {
		return Registration{}, err
	}
	if err := s.authorize(scope, "tax-registration:read", registrationID); err != nil {
		return Registration{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	value, ok := s.registrations[scope.IsolationKey(registrationID)]
	if !ok {
		return Registration{}, ErrNotFound
	}
	return value, nil
}

func (s *Service) DraftReturn(
	scope foundation.Context,
	taxpayerID string,
	registrationID string,
	periodID string,
	lines []filing.Line,
) (filing.TaxReturn, error) {
	if err := scope.Validate(); err != nil {
		return filing.TaxReturn{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	taxRegistration, ok := s.registrations[scope.IsolationKey(registrationID)]
	if !ok || taxRegistration.TaxpayerID != taxpayerID || taxRegistration.Status != registration.StatusApproved {
		return filing.TaxReturn{}, errors.New("registration not found")
	}
	taxReturn, err := filing.New(scope, taxpayerID, registrationID, periodID, lines)
	if err != nil {
		return filing.TaxReturn{}, err
	}
	s.returns[scope.IsolationKey(taxReturn.ID.String())] = taxReturn
	if err := s.record(scope, "ReturnCreated", "return", taxReturn.ID.String()); err != nil {
		return filing.TaxReturn{}, err
	}
	return taxReturn, nil
}

func (s *Service) ValidateReturn(scope foundation.Context, returnID string) (filing.TaxReturn, error) {
	if err := scope.Validate(); err != nil {
		return filing.TaxReturn{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	key := scope.IsolationKey(returnID)
	taxReturn, ok := s.returns[key]
	if !ok {
		return taxReturn, errors.New("return not found")
	}
	if err := taxReturn.Validate(); err != nil {
		return taxReturn, err
	}
	s.returns[key] = taxReturn
	if err := s.record(scope, "ReturnValidated", "return", returnID); err != nil {
		return filing.TaxReturn{}, err
	}
	return taxReturn, nil
}

func (s *Service) SubmitAndAssess(
	ctx context.Context,
	scope foundation.Context,
	returnID string,
) (Assessment, error) {
	if err := scope.Validate(); err != nil {
		return Assessment{}, err
	}
	s.mu.Lock()
	key := scope.IsolationKey(returnID)
	taxReturn, ok := s.returns[key]
	if !ok {
		s.mu.Unlock()
		return Assessment{}, errors.New("return not found")
	}
	now := s.clock.Now()
	if err := taxReturn.Submit(now); err != nil {
		s.mu.Unlock()
		return Assessment{}, err
	}
	currency, err := foundation.NewCurrency("XCR", 2)
	if err != nil {
		s.mu.Unlock()
		return Assessment{}, err
	}
	taxable, _ := foundation.NewMoney(0, currency)
	for _, line := range taxReturn.Lines {
		lineAmount, moneyErr := foundation.NewMoney(line.AmountMinor, currency)
		if moneyErr != nil {
			s.mu.Unlock()
			return Assessment{}, moneyErr
		}
		taxable, moneyErr = taxable.Add(lineAmount)
		if moneyErr != nil {
			s.mu.Unlock()
			return Assessment{}, moneyErr
		}
	}
	amount, err := foundation.NewMoney(taxable.Minor()/10, currency)
	if err != nil {
		s.mu.Unlock()
		return Assessment{}, err
	}
	assessment := Assessment{
		ID: id.New[AssessmentTag](), TenantID: scope.Tenant().String(),
		Jurisdiction: scope.Jurisdiction().String(), ReturnID: returnID, Amount: amount,
	}
	entry, err := ledger.NewEntry(
		scope, ledger.AssessmentDebit, ledger.TaxpayerID(taxReturn.TaxpayerID),
		ledger.RegistrationID(taxReturn.RegistrationID), ledger.PeriodID(taxReturn.PeriodID),
		amount, "ASSESSMENT", assessment.ID.String(), now,
	)
	if err != nil {
		s.mu.Unlock()
		return Assessment{}, err
	}
	s.returns[key] = taxReturn
	s.assessments[scope.IsolationKey(assessment.ID.String())] = assessment
	s.entries = append(s.entries, entry)
	for _, event := range []struct{ action, kind, id string }{
		{"ReturnSubmitted", "return", returnID},
		{"AssessmentCreated", "assessment", assessment.ID.String()},
		{"LedgerEntryPosted", "ledger_entry", entry.ID.String()},
	} {
		if err := s.record(scope, event.action, event.kind, event.id); err != nil {
			s.mu.Unlock()
			return Assessment{}, err
		}
	}
	s.mu.Unlock()

	if s.notifier != nil {
		_ = s.notifier.Send(ctx, Notification{
			To: "demo@example.invalid", Subject: "Return submitted",
			Body: "Your fictional sample return was assessed.",
		})
	}
	return assessment, nil
}

func (s *Service) RecordPayment(
	scope foundation.Context,
	taxpayerID string,
	assessmentID string,
	amount ledger.Money,
) (Payment, error) {
	if err := scope.Validate(); err != nil {
		return Payment{}, err
	}
	if err := amount.Validate(); err != nil {
		return Payment{}, err
	}
	if amount.Minor() <= 0 {
		return Payment{}, errors.New("payment amount must be positive")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	assessment, ok := s.assessments[scope.IsolationKey(assessmentID)]
	if !ok {
		return Payment{}, errors.New("assessment not found")
	}
	taxReturn := s.returns[scope.IsolationKey(assessment.ReturnID)]
	if taxReturn.TaxpayerID != taxpayerID {
		return Payment{}, errors.New("assessment does not belong to taxpayer")
	}
	if amount.Currency() != assessment.Amount.Currency() {
		return Payment{}, foundation.ErrCurrencyMismatch
	}
	payment := Payment{
		ID: id.New[PaymentTag](), TenantID: scope.Tenant().String(),
		Jurisdiction: scope.Jurisdiction().String(), TaxpayerID: taxpayerID,
		Amount: amount, AllocatedTo: assessmentID,
	}
	entry, err := ledger.NewEntry(
		scope, ledger.PaymentCredit, ledger.TaxpayerID(taxReturn.TaxpayerID),
		ledger.RegistrationID(taxReturn.RegistrationID), ledger.PeriodID(taxReturn.PeriodID),
		amount, "PAYMENT", payment.ID.String(), s.clock.Now(),
	)
	if err != nil {
		return Payment{}, err
	}
	s.payments[scope.IsolationKey(payment.ID.String())] = payment
	s.entries = append(s.entries, entry)
	for _, event := range []struct{ action, kind, id string }{
		{"PaymentReceived", "payment", payment.ID.String()},
		{"PaymentAllocated", "assessment", assessmentID},
		{"LedgerEntryPosted", "ledger_entry", entry.ID.String()},
	} {
		if err := s.record(scope, event.action, event.kind, event.id); err != nil {
			return Payment{}, err
		}
	}
	return payment, nil
}

func (s *Service) Ledger(scope foundation.Context, taxpayerID string) ([]ledger.Entry, error) {
	if err := scope.Validate(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	entries := make([]ledger.Entry, 0)
	for _, entry := range s.entries {
		if entry.TenantID == scope.Tenant().String() &&
			entry.Jurisdiction == scope.Jurisdiction().String() &&
			entry.TaxpayerID.String() == taxpayerID {
			entries = append(entries, entry)
		}
	}
	sort.SliceStable(entries, func(i, j int) bool {
		return entries[i].PostedAt.Before(entries[j].PostedAt)
	})
	return entries, nil
}

func (s *Service) Audits(scope foundation.Context) ([]audit.Event, error) {
	if err := scope.Validate(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	events := make([]audit.Event, 0)
	for _, event := range s.audits {
		if event.TenantID == scope.Tenant().String() &&
			event.Jurisdiction == scope.Jurisdiction().String() {
			events = append(events, event)
		}
	}
	return events, nil
}

func (s *Service) Events(scope foundation.Context) ([]event.Event, error) {
	if err := scope.Validate(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	events := make([]event.Event, 0)
	for _, value := range s.events {
		if value.Tenant == scope.Tenant().String() &&
			value.Jurisdiction == scope.Jurisdiction().String() {
			events = append(events, value)
		}
	}
	return events, nil
}

func (s *Service) CurrentTime(scope foundation.Context) (time.Time, error) {
	if err := scope.Validate(); err != nil {
		return time.Time{}, err
	}
	return s.clock.Now().UTC(), nil
}
