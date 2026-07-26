package application_test

import (
	"errors"
	"testing"

	app "github.com/opencorex-org/openrevenue/internal/administration/application"
	registration "github.com/opencorex-org/openrevenue/internal/taxregistration/domain"
	foundation "github.com/opencorex-org/openrevenue/pkg/domain"
)

type denyingAuthorizer struct{ permission string }

func (a denyingAuthorizer) Authorize(_ foundation.Context, permission, _ string) error {
	if permission == a.permission {
		return errors.New("denied")
	}
	return nil
}

func TestTaxpayerIdempotencyAndIdentifierUniqueness(t *testing.T) {
	s := app.New(nil)
	requestScope := scope(t, "revenue", "LK", "registrar")

	first, err := s.CreateTaxpayer(requestScope, "Fictional Cooperative", "demo-100", "request-1")
	if err != nil {
		t.Fatal(err)
	}
	replayed, err := s.CreateTaxpayer(requestScope, "Fictional Cooperative", "demo-100", "request-1")
	if err != nil || replayed.ID != first.ID {
		t.Fatalf("idempotent replay = %#v, %v", replayed, err)
	}
	if _, err = s.CreateTaxpayer(requestScope, "Different Name", "demo-101", "request-1"); !errors.Is(err, app.ErrConflict) {
		t.Fatalf("reused key error = %v", err)
	}
	if _, err = s.CreateTaxpayer(requestScope, "Duplicate Identifier", "DEMO 100", "request-2"); !errors.Is(err, app.ErrConflict) {
		t.Fatalf("duplicate identifier error = %v", err)
	}
}

func TestRegistrationLifecycleRetrievalAndEventContext(t *testing.T) {
	s := app.New(nil)
	requestScope := scope(t, "revenue", "LK", "officer")
	causation, _ := foundation.NewCorrelationID("command-11")
	requestScope, _ = requestScope.WithCausationID(causation)
	taxpayerValue, _ := s.CreateTaxpayer(requestScope, "Fictional Taxpayer", "demo-200", "request-1")

	submitted, err := s.Register(requestScope, taxpayerValue.ID.String(), "SAMPLE_INCOME")
	if err != nil {
		t.Fatal(err)
	}
	if submitted.Status != registration.StatusSubmitted {
		t.Fatalf("status = %s", submitted.Status)
	}
	approved, err := s.ApproveRegistration(requestScope, submitted.ID.String())
	if err != nil {
		t.Fatal(err)
	}
	if approved.Status != registration.StatusApproved || approved.ApprovedBy != "officer" {
		t.Fatalf("approved registration = %#v", approved)
	}
	if _, err = s.ApproveRegistration(requestScope, submitted.ID.String()); !errors.Is(err, app.ErrInvalidTransition) {
		t.Fatalf("second approval error = %v", err)
	}
	got, err := s.GetRegistration(requestScope, submitted.ID.String())
	if err != nil || got.Status != registration.StatusApproved {
		t.Fatalf("retrieved registration = %#v, %v", got, err)
	}
	events, err := s.Events(requestScope)
	if err != nil || len(events) != 3 {
		t.Fatalf("events = %#v, %v", events, err)
	}
	for _, value := range events {
		if value.Actor.ID != "officer" || value.Tenant != "revenue" ||
			value.CorrelationID != "vertical-slice" || value.CausationID != "command-11" {
			t.Fatalf("event context = %#v", value)
		}
	}
	audits, _ := s.Audits(requestScope)
	for _, value := range audits {
		if value.CausationID != "command-11" || value.ActorType != "USER" {
			t.Fatalf("audit context = %#v", value)
		}
	}
}

func TestAuthorizationDenialFailsClosed(t *testing.T) {
	s := app.NewWithDependencies(nil, foundation.SystemClock{}, denyingAuthorizer{permission: "tax-registration:approve"})
	requestScope := scope(t, "revenue", "LK", "officer")
	taxpayerValue, _ := s.CreateTaxpayer(requestScope, "Fictional Taxpayer", "demo-300", "request-1")
	submitted, _ := s.Register(requestScope, taxpayerValue.ID.String(), "SAMPLE_INCOME")
	if _, err := s.ApproveRegistration(requestScope, submitted.ID.String()); !errors.Is(err, app.ErrForbidden) {
		t.Fatalf("authorization error = %v", err)
	}
}
