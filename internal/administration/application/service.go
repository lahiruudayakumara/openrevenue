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
	calculation "github.com/opencorex-org/openrevenue/internal/calculation/domain"
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
	calculator    calculation.Calculator
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
		calculator: calculation.FlatRateCalculator{
			RuleVersion: filing.DefaultRuleVersion, RateBPS: 1_000,
		},
	}
}

func (s *Service) SetCalculator(calculator calculation.Calculator) {
	if calculator == nil {
		panic("calculator is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calculator = calculator
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
	return s.DraftReturnVersioned(
		scope, taxpayerID, registrationID, periodID,
		filing.DefaultFormVersion, filing.DefaultRuleVersion, lines,
	)
}

func (s *Service) DraftReturnVersioned(
	scope foundation.Context,
	taxpayerID string,
	registrationID string,
	periodID string,
	formVersion string,
	ruleVersion string,
	lines []filing.Line,
) (filing.TaxReturn, error) {
	if err := scope.Validate(); err != nil {
		return filing.TaxReturn{}, err
	}
	if err := s.authorize(scope, "return:create", registrationID); err != nil {
		return filing.TaxReturn{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	taxRegistration, ok := s.registrations[scope.IsolationKey(registrationID)]
	if !ok || taxRegistration.TaxpayerID != taxpayerID || taxRegistration.Status != registration.StatusApproved {
		return filing.TaxReturn{}, errors.New("registration not found")
	}
	taxReturn, err := filing.NewVersioned(
		scope, taxpayerID, registrationID, periodID, formVersion, ruleVersion, lines,
	)
	if err != nil {
		return filing.TaxReturn{}, err
	}
	s.returns[scope.IsolationKey(taxReturn.ID.String())] = taxReturn
	if err := s.record(scope, "ReturnCreated", "return", taxReturn.ID.String()); err != nil {
		return filing.TaxReturn{}, err
	}
	if err := s.emit(scope, "ReturnCreated", "return", taxReturn.ID.String(), map[string]string{
		"formVersion": taxReturn.FormVersion, "ruleVersion": taxReturn.RuleVersion,
	}); err != nil {
		return filing.TaxReturn{}, err
	}
	return taxReturn, nil
}

func (s *Service) ValidateReturn(scope foundation.Context, returnID string) (filing.TaxReturn, error) {
	if err := scope.Validate(); err != nil {
		return filing.TaxReturn{}, err
	}
	if err := s.authorize(scope, "return:validate", returnID); err != nil {
		return filing.TaxReturn{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	key := scope.IsolationKey(returnID)
	taxReturn, ok := s.returns[key]
	if !ok {
		return taxReturn, errors.New("return not found")
	}
	result := taxReturn.ValidateAgainst(filing.DefaultForm())
	if !result.Valid {
		s.returns[key] = taxReturn
		return taxReturn, nil
	}
	s.returns[key] = taxReturn
	if err := s.record(scope, "ReturnValidated", "return", returnID); err != nil {
		return filing.TaxReturn{}, err
	}
	if err := s.emit(scope, "ReturnValidated", "return", returnID, map[string]string{
		"formVersion": taxReturn.FormVersion, "payloadHash": result.PayloadHash,
	}); err != nil {
		return filing.TaxReturn{}, err
	}
	return taxReturn, nil
}

func (s *Service) CalculateReturn(scope foundation.Context, returnID string) (filing.TaxReturn, error) {
	if err := scope.Validate(); err != nil {
		return filing.TaxReturn{}, err
	}
	if err := s.authorize(scope, "return:calculate", returnID); err != nil {
		return filing.TaxReturn{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	key := scope.IsolationKey(returnID)
	taxReturn, ok := s.returns[key]
	if !ok {
		return filing.TaxReturn{}, ErrNotFound
	}
	if taxReturn.Validation == nil {
		return filing.TaxReturn{}, fmt.Errorf("%w: return has not been validated", ErrInvalidTransition)
	}
	currency, err := foundation.NewCurrency("XCR", 2)
	if err != nil {
		return filing.TaxReturn{}, err
	}
	lines := make([]calculation.InputLine, len(taxReturn.Lines))
	for index, line := range taxReturn.Lines {
		lines[index] = calculation.InputLine{Code: line.Code, AmountMinor: line.AmountMinor}
	}
	result, err := s.calculator.Calculate(calculation.Input{
		RuleVersion: taxReturn.RuleVersion, Currency: currency,
		Lines: lines, InputHash: taxReturn.Validation.PayloadHash,
	})
	if err != nil {
		return filing.TaxReturn{}, err
	}
	if err = taxReturn.RecordCalculation(result); err != nil {
		return filing.TaxReturn{}, fmt.Errorf("%w: %v", ErrInvalidTransition, err)
	}
	s.returns[key] = taxReturn
	if err := s.record(scope, "ReturnCalculated", "return", returnID); err != nil {
		return filing.TaxReturn{}, err
	}
	if err := s.emit(scope, "ReturnCalculated", "return", returnID, map[string]string{
		"ruleVersion": result.RuleVersion, "resultHash": result.ResultHash,
	}); err != nil {
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
	if err := s.authorize(scope, "return:submit", returnID); err != nil {
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
	if err := taxReturn.Submit(now, filing.DefaultFormVersion, filing.DefaultRuleVersion); err != nil {
		s.mu.Unlock()
		return Assessment{}, fmt.Errorf("%w: %v", ErrInvalidTransition, err)
	}
	amount := taxReturn.Calculation.Amount
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
	if err := s.emit(scope, "ReturnSubmitted", "return", returnID, map[string]string{
		"formVersion": taxReturn.FormVersion, "ruleVersion": taxReturn.RuleVersion,
		"payloadHash": taxReturn.FrozenPayloadHash,
	}); err != nil {
		s.mu.Unlock()
		return Assessment{}, err
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

func (s *Service) GetReturn(scope foundation.Context, returnID string) (filing.TaxReturn, error) {
	if err := scope.Validate(); err != nil {
		return filing.TaxReturn{}, err
	}
	if err := s.authorize(scope, "return:read", returnID); err != nil {
		return filing.TaxReturn{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	value, ok := s.returns[scope.IsolationKey(returnID)]
	if !ok {
		return filing.TaxReturn{}, ErrNotFound
	}
	return value, nil
}

func (s *Service) AmendReturn(scope foundation.Context, returnID string) (filing.TaxReturn, error) {
	if err := scope.Validate(); err != nil {
		return filing.TaxReturn{}, err
	}
	if err := s.authorize(scope, "return:amend", returnID); err != nil {
		return filing.TaxReturn{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	original, ok := s.returns[scope.IsolationKey(returnID)]
	if !ok {
		return filing.TaxReturn{}, ErrNotFound
	}
	amendment, err := original.Amend(scope)
	if err != nil {
		return filing.TaxReturn{}, fmt.Errorf("%w: %v", ErrInvalidTransition, err)
	}
	s.returns[scope.IsolationKey(amendment.ID.String())] = amendment
	if err := s.record(scope, "ReturnAmended", "return", amendment.ID.String()); err != nil {
		return filing.TaxReturn{}, err
	}
	if err := s.emit(scope, "ReturnAmended", "return", amendment.ID.String(), map[string]string{
		"originalReturnId": amendment.OriginalReturnID, "supersedesId": amendment.SupersedesID,
	}); err != nil {
		return filing.TaxReturn{}, err
	}
	return amendment, nil
}

func (s *Service) ReturnHistory(scope foundation.Context, returnID string) ([]filing.TaxReturn, error) {
	value, err := s.GetReturn(scope, returnID)
	if err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	history := make([]filing.TaxReturn, 0)
	for _, candidate := range s.returns {
		if candidate.TenantID == scope.Tenant().String() &&
			candidate.Jurisdiction == scope.Jurisdiction().String() &&
			candidate.OriginalReturnID == value.OriginalReturnID {
			history = append(history, candidate)
		}
	}
	sort.Slice(history, func(i, j int) bool { return history[i].Revision < history[j].Revision })
	return history, nil
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
