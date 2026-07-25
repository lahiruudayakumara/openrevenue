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
