package domain

import (
	"errors"
	"time"

	foundation "github.com/opencorex-org/openrevenue/pkg/domain"
	"github.com/opencorex-org/openrevenue/pkg/id"
)

type TaxpayerTag struct{}
type RegistrationTag struct{}
type PeriodTag struct{}
type EntryTag struct{}
type PostingTag struct{}

type TaxpayerID = id.ID[TaxpayerTag]
type RegistrationID = id.ID[RegistrationTag]
type PeriodID = id.ID[PeriodTag]
type EntryID = id.ID[EntryTag]
type PostingID = id.ID[PostingTag]

type Money = foundation.Money

type EntryType string

const (
	AssessmentDebit  EntryType = "ASSESSMENT_DEBIT"
	PaymentCredit    EntryType = "PAYMENT_CREDIT"
	PenaltyDebit     EntryType = "PENALTY_DEBIT"
	InterestDebit    EntryType = "INTEREST_DEBIT"
	RefundDebit      EntryType = "REFUND_DEBIT"
	AdjustmentDebit  EntryType = "ADJUSTMENT_DEBIT"
	AdjustmentCredit EntryType = "ADJUSTMENT_CREDIT"
	Reversal         EntryType = "REVERSAL"
)

type Account string

const (
	TaxpayerReceivable Account = "TAXPAYER_RECEIVABLE"
	RevenueControl     Account = "REVENUE_CONTROL"
	CashControl        Account = "CASH_CONTROL"
	UnappliedCash      Account = "UNAPPLIED_CASH"
)

type Entry struct {
	ID                EntryID           `json:"id"`
	PostingID         PostingID         `json:"postingId"`
	TenantID          string            `json:"tenantId"`
	Jurisdiction      string            `json:"jurisdiction"`
	TaxpayerID        TaxpayerID        `json:"taxpayerId"`
	TaxRegistrationID RegistrationID    `json:"taxRegistrationId"`
	TaxPeriodID       PeriodID          `json:"taxPeriodId"`
	Type              EntryType         `json:"entryType"`
	Account           Account           `json:"account"`
	Debit             Money             `json:"debitAmount"`
	Credit            Money             `json:"creditAmount"`
	ReferenceType     string            `json:"referenceType"`
	ReferenceID       string            `json:"referenceId"`
	EffectiveDate     time.Time         `json:"effectiveDate"`
	PostedAt          time.Time         `json:"postedAt"`
	ReversalOf        *EntryID          `json:"reversalOf,omitempty"`
	CreatedBy         string            `json:"createdBy"`
	CorrelationID     string            `json:"correlationId"`
	CausationID       string            `json:"causationId"`
	Metadata          map[string]string `json:"metadata,omitempty"`
}

type Posting struct {
	ID            PostingID  `json:"id"`
	TenantID      string     `json:"tenantId"`
	Jurisdiction  string     `json:"jurisdiction"`
	ReferenceType string     `json:"referenceType"`
	ReferenceID   string     `json:"referenceId"`
	Entries       []Entry    `json:"entries"`
	PostedAt      time.Time  `json:"postedAt"`
	ReversalOf    *PostingID `json:"reversalOf,omitempty"`
}

func (p Posting) ValidateBalanced() error {
	if len(p.Entries) < 2 {
		return errors.New("ledger posting requires at least two entries")
	}
	currency := p.Entries[0].Debit.Currency()
	if p.Entries[0].Debit.IsZero() {
		currency = p.Entries[0].Credit.Currency()
	}
	var debits, credits Money
	var err error
	debits, _ = foundation.NewMoney(0, currency)
	credits, _ = foundation.NewMoney(0, currency)
	for _, entry := range p.Entries {
		if entry.PostingID != p.ID || entry.TenantID != p.TenantID ||
			entry.Jurisdiction != p.Jurisdiction {
			return errors.New("ledger posting entry context does not match")
		}
		debits, err = debits.Add(entry.Debit)
		if err != nil {
			return err
		}
		credits, err = credits.Add(entry.Credit)
		if err != nil {
			return err
		}
	}
	if debits.Minor() != credits.Minor() || debits.Minor() <= 0 {
		return errors.New("ledger posting is not balanced")
	}
	return nil
}

func NewEntry(
	scope foundation.Context,
	t EntryType,
	taxpayer TaxpayerID,
	registration RegistrationID,
	period PeriodID,
	amount Money,
	refType string,
	refID string,
	now time.Time,
) (Entry, error) {
	if err := scope.Validate(); err != nil {
		return Entry{}, err
	}
	if err := amount.Validate(); err != nil {
		return Entry{}, err
	}
	if amount.Minor() <= 0 {
		return Entry{}, errors.New("ledger amount must be positive")
	}
	if now.IsZero() {
		return Entry{}, errors.New("posting time is required")
	}
	zero, err := foundation.NewMoney(0, amount.Currency())
	if err != nil {
		return Entry{}, err
	}
	now = now.UTC()
	e := Entry{
		ID: id.New[EntryTag](), PostingID: id.New[PostingTag](),
		TenantID:     scope.Tenant().String(),
		Jurisdiction: scope.Jurisdiction().String(), TaxpayerID: taxpayer,
		TaxRegistrationID: registration, TaxPeriodID: period, Type: t,
		Account:       TaxpayerReceivable,
		ReferenceType: refType, ReferenceID: refID, EffectiveDate: now,
		PostedAt: now, CreatedBy: scope.Actor().ID(),
		CorrelationID: scope.CorrelationID().String(),
		CausationID:   scope.CausationID().String(),
	}
	if t == PaymentCredit || t == AdjustmentCredit {
		e.Credit = amount
		e.Debit = zero
	} else {
		e.Debit = amount
		e.Credit = zero
	}
	return e, nil
}

func NewAssessmentPosting(
	scope foundation.Context,
	taxpayer TaxpayerID,
	registration RegistrationID,
	period PeriodID,
	amount Money,
	refID string,
	now time.Time,
) (Posting, error) {
	return newBalancedPosting(
		scope, AssessmentDebit, taxpayer, registration, period, amount,
		"ASSESSMENT", refID, TaxpayerReceivable, RevenueControl, now,
	)
}

func NewPaymentAllocationPosting(
	scope foundation.Context,
	taxpayer TaxpayerID,
	registration RegistrationID,
	period PeriodID,
	amount Money,
	paymentID string,
	now time.Time,
) (Posting, error) {
	return newBalancedPosting(
		scope, PaymentCredit, taxpayer, registration, period, amount,
		"PAYMENT_ALLOCATION", paymentID, UnappliedCash, TaxpayerReceivable, now,
	)
}

func NewPaymentReceiptPosting(
	scope foundation.Context,
	taxpayer TaxpayerID,
	amount Money,
	paymentID string,
	now time.Time,
) (Posting, error) {
	return newBalancedPosting(
		scope, PaymentCredit, taxpayer, "", "", amount,
		"PAYMENT_RECEIPT", paymentID, CashControl, UnappliedCash, now,
	)
}

func newBalancedPosting(
	scope foundation.Context,
	entryType EntryType,
	taxpayer TaxpayerID,
	registration RegistrationID,
	period PeriodID,
	amount Money,
	refType, refID string,
	debitAccount, creditAccount Account,
	now time.Time,
) (Posting, error) {
	base, err := NewEntry(scope, entryType, taxpayer, registration, period, amount, refType, refID, now)
	if err != nil {
		return Posting{}, err
	}
	postingID := base.PostingID
	zero, _ := foundation.NewMoney(0, amount.Currency())
	debit := base
	debit.ID = id.New[EntryTag]()
	debit.PostingID = postingID
	debit.Account = debitAccount
	debit.Debit, debit.Credit = amount, zero
	credit := base
	credit.ID = id.New[EntryTag]()
	credit.PostingID = postingID
	credit.Account = creditAccount
	credit.Debit, credit.Credit = zero, amount
	posting := Posting{
		ID: postingID, TenantID: scope.Tenant().String(),
		Jurisdiction:  scope.Jurisdiction().String(),
		ReferenceType: refType, ReferenceID: refID,
		Entries: []Entry{debit, credit}, PostedAt: now.UTC(),
	}
	return posting, posting.ValidateBalanced()
}

func NewReversal(scope foundation.Context, original Entry, now time.Time) (Entry, error) {
	if err := scope.Validate(); err != nil {
		return Entry{}, err
	}
	if original.TenantID != scope.Tenant().String() ||
		original.Jurisdiction != scope.Jurisdiction().String() {
		return Entry{}, errors.New("ledger entry is outside the requested tenant and jurisdiction")
	}
	if now.IsZero() {
		return Entry{}, errors.New("reversal time is required")
	}
	now = now.UTC()
	return Entry{
		ID: id.New[EntryTag](), PostingID: id.New[PostingTag](), TenantID: original.TenantID,
		Jurisdiction: original.Jurisdiction, TaxpayerID: original.TaxpayerID,
		TaxRegistrationID: original.TaxRegistrationID, TaxPeriodID: original.TaxPeriodID,
		Type: Reversal, Debit: original.Credit, Credit: original.Debit,
		ReferenceType: "LEDGER_ENTRY", ReferenceID: original.ID.String(),
		Account: original.Account, EffectiveDate: now, PostedAt: now,
		CreatedBy:     scope.Actor().ID(),
		CorrelationID: scope.CorrelationID().String(),
		CausationID:   scope.CausationID().String(),
		ReversalOf:    &original.ID,
	}, nil
}

func NewReversalPosting(scope foundation.Context, original Posting, now time.Time) (Posting, error) {
	if err := scope.Validate(); err != nil {
		return Posting{}, err
	}
	if original.TenantID != scope.Tenant().String() ||
		original.Jurisdiction != scope.Jurisdiction().String() {
		return Posting{}, errors.New("ledger posting is outside the request scope")
	}
	if now.IsZero() {
		return Posting{}, errors.New("reversal time is required")
	}
	postingID := id.New[PostingTag]()
	entries := make([]Entry, len(original.Entries))
	for index, source := range original.Entries {
		entryID := source.ID
		entries[index] = source
		entries[index].ID = id.New[EntryTag]()
		entries[index].PostingID = postingID
		entries[index].Type = Reversal
		entries[index].Debit, entries[index].Credit = source.Credit, source.Debit
		entries[index].ReferenceType = "LEDGER_POSTING"
		entries[index].ReferenceID = original.ID.String()
		entries[index].ReversalOf = &entryID
		entries[index].EffectiveDate = now.UTC()
		entries[index].PostedAt = now.UTC()
		entries[index].CreatedBy = scope.Actor().ID()
		entries[index].CorrelationID = scope.CorrelationID().String()
		entries[index].CausationID = scope.CausationID().String()
	}
	reversalOf := original.ID
	posting := Posting{
		ID: postingID, TenantID: original.TenantID, Jurisdiction: original.Jurisdiction,
		ReferenceType: "LEDGER_POSTING", ReferenceID: original.ID.String(),
		Entries: entries, PostedAt: now.UTC(), ReversalOf: &reversalOf,
	}
	return posting, posting.ValidateBalanced()
}
