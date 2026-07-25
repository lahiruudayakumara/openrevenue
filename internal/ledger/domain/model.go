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

type TaxpayerID = id.ID[TaxpayerTag]
type RegistrationID = id.ID[RegistrationTag]
type PeriodID = id.ID[PeriodTag]
type EntryID = id.ID[EntryTag]

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

type Entry struct {
	ID                EntryID           `json:"id"`
	TenantID          string            `json:"tenantId"`
	Jurisdiction      string            `json:"jurisdiction"`
	TaxpayerID        TaxpayerID        `json:"taxpayerId"`
	TaxRegistrationID RegistrationID    `json:"taxRegistrationId"`
	TaxPeriodID       PeriodID          `json:"taxPeriodId"`
	Type              EntryType         `json:"entryType"`
	Debit             Money             `json:"debitAmount"`
	Credit            Money             `json:"creditAmount"`
	ReferenceType     string            `json:"referenceType"`
	ReferenceID       string            `json:"referenceId"`
	EffectiveDate     time.Time         `json:"effectiveDate"`
	PostedAt          time.Time         `json:"postedAt"`
	ReversalOf        *EntryID          `json:"reversalOf,omitempty"`
	CreatedBy         string            `json:"createdBy"`
	Metadata          map[string]string `json:"metadata,omitempty"`
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
		ID: id.New[EntryTag](), TenantID: scope.Tenant().String(),
		Jurisdiction: scope.Jurisdiction().String(), TaxpayerID: taxpayer,
		TaxRegistrationID: registration, TaxPeriodID: period, Type: t,
		ReferenceType: refType, ReferenceID: refID, EffectiveDate: now,
		PostedAt: now, CreatedBy: scope.Actor().ID(),
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
		ID: id.New[EntryTag](), TenantID: original.TenantID,
		Jurisdiction: original.Jurisdiction, TaxpayerID: original.TaxpayerID,
		TaxRegistrationID: original.TaxRegistrationID, TaxPeriodID: original.TaxPeriodID,
		Type: Reversal, Debit: original.Credit, Credit: original.Debit,
		ReferenceType: "LEDGER_ENTRY", ReferenceID: original.ID.String(),
		EffectiveDate: now, PostedAt: now, CreatedBy: scope.Actor().ID(),
		ReversalOf: &original.ID,
	}, nil
}
