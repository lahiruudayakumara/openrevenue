package domain

import (
	"testing"
	"time"

	foundation "github.com/opencorex-org/openrevenue/pkg/domain"
	"github.com/opencorex-org/openrevenue/pkg/id"
)

func TestReversalSwapsDebitAndCredit(t *testing.T) {
	tenant, _ := foundation.NewTenantID("revenue")
	jurisdiction, _ := foundation.NewJurisdiction("LK")
	actor, _ := foundation.NewActor(foundation.ActorUser, "officer")
	correlation, _ := foundation.NewCorrelationID("test-reversal")
	scope, _ := foundation.NewContext(tenant, jurisdiction, actor, correlation)
	currency, _ := foundation.NewCurrency("XCR", 2)
	amount, _ := foundation.NewMoney(2500, currency)
	now := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	e, err := NewEntry(scope, AssessmentDebit, id.New[TaxpayerTag](), id.New[RegistrationTag](), id.New[PeriodTag](), amount, "ASSESSMENT", "a", now)
	if err != nil {
		t.Fatal(err)
	}
	r, err := NewReversal(scope, e, now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if r.Credit.Minor() != 2500 {
		t.Fatalf("credit = %d", r.Credit.Minor())
	}
	if r.ReversalOf == nil || e.ID != *r.ReversalOf {
		t.Fatal("reversal link missing")
	}
}

func TestAssessmentAndPaymentPostingsAreBalancedAndTraceable(t *testing.T) {
	tenant, _ := foundation.NewTenantID("revenue")
	jurisdiction, _ := foundation.NewJurisdiction("LK")
	actor, _ := foundation.NewActor(foundation.ActorUser, "officer")
	correlation, _ := foundation.NewCorrelationID("balanced-posting")
	scope, _ := foundation.NewContext(tenant, jurisdiction, actor, correlation)
	causation, _ := foundation.NewCorrelationID("source-command")
	scope, _ = scope.WithCausationID(causation)
	currency, _ := foundation.NewCurrency("XCR", 2)
	amount, _ := foundation.NewMoney(10_00, currency)
	taxpayerID := id.New[TaxpayerTag]()
	registrationID := id.New[RegistrationTag]()
	periodID := id.New[PeriodTag]()
	now := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)

	postings := []Posting{}
	assessment, err := NewAssessmentPosting(
		scope, taxpayerID, registrationID, periodID, amount, "assessment-1", now,
	)
	if err != nil {
		t.Fatal(err)
	}
	postings = append(postings, assessment)
	receipt, err := NewPaymentReceiptPosting(scope, taxpayerID, amount, "payment-1", now)
	if err != nil {
		t.Fatal(err)
	}
	postings = append(postings, receipt)
	allocation, err := NewPaymentAllocationPosting(
		scope, taxpayerID, registrationID, periodID, amount, "payment-1", now,
	)
	if err != nil {
		t.Fatal(err)
	}
	postings = append(postings, allocation)

	for _, posting := range postings {
		if err := posting.ValidateBalanced(); err != nil {
			t.Fatalf("posting is not balanced: %v", err)
		}
		for _, entry := range posting.Entries {
			if entry.CorrelationID != "balanced-posting" ||
				entry.CausationID != "source-command" ||
				entry.CreatedBy != "officer" ||
				entry.TaxpayerID != taxpayerID {
				t.Fatalf("entry traceability = %#v", entry)
			}
		}
	}
	reversal, err := NewReversalPosting(scope, assessment, now.Add(time.Hour))
	if err != nil || reversal.ValidateBalanced() != nil {
		t.Fatalf("reversal = %#v, %v", reversal, err)
	}
}
