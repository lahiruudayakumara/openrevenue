package domain_test

import (
	"testing"
	"time"

	registration "github.com/opencorex-org/openrevenue/internal/taxregistration/domain"
	foundation "github.com/opencorex-org/openrevenue/pkg/domain"
)

func testScope(t *testing.T) foundation.Context {
	t.Helper()
	tenant, _ := foundation.NewTenantID("revenue")
	jurisdiction, _ := foundation.NewJurisdiction("LK")
	actor, _ := foundation.NewActor(foundation.ActorUser, "officer")
	correlation, _ := foundation.NewCorrelationID("registration-test")
	value, err := foundation.NewContext(tenant, jurisdiction, actor, correlation)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func TestRegistrationStateMachine(t *testing.T) {
	now := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	value, err := registration.Submit(testScope(t), "taxpayer-1", "SAMPLE_INCOME", now)
	if err != nil {
		t.Fatal(err)
	}
	if value.Status != registration.StatusSubmitted {
		t.Fatalf("status = %s", value.Status)
	}
	if err = value.Approve(testScope(t), now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err = value.Approve(testScope(t), now.Add(2*time.Hour)); err == nil {
		t.Fatal("second approval was accepted")
	}
}
