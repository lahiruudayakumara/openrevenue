package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	app "github.com/opencorex-org/openrevenue/internal/administration/application"
	filing "github.com/opencorex-org/openrevenue/internal/filing/domain"
	foundation "github.com/opencorex-org/openrevenue/pkg/domain"
)

type notifier struct{ sent int }

func (n *notifier) Send(context.Context, app.Notification) error { n.sent++; return nil }

func scope(t *testing.T, tenantValue, jurisdictionValue, actorID string) foundation.Context {
	t.Helper()
	tenant, err := foundation.NewTenantID(tenantValue)
	if err != nil {
		t.Fatal(err)
	}
	jurisdiction, err := foundation.NewJurisdiction(jurisdictionValue)
	if err != nil {
		t.Fatal(err)
	}
	actor, err := foundation.NewActor(foundation.ActorUser, actorID)
	if err != nil {
		t.Fatal(err)
	}
	correlation, err := foundation.NewCorrelationID("vertical-slice")
	if err != nil {
		t.Fatal(err)
	}
	value, err := foundation.NewContext(tenant, jurisdiction, actor, correlation)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func TestVerticalSliceUsesCanonicalContextMoneyAndClock(t *testing.T) {
	n := &notifier{}
	now := time.Date(2026, 7, 1, 10, 30, 0, 0, time.FixedZone("test", 5*60*60))
	clock, _ := foundation.NewFixedClock(now)
	s := app.NewWithClock(n, clock)
	admin := scope(t, "revenue", "LK", "admin")
	officer := scope(t, "revenue", "LK", "officer")
	taxpayerActor := scope(t, "revenue", "LK", "taxpayer")
	cashier := scope(t, "revenue", "LK", "cashier")

	taxpayer, err := s.CreateTaxpayer(admin, "Demo Cooperative", "demo-001", "key")
	if err != nil {
		t.Fatal(err)
	}
	if taxpayer.Identifier != "DEMO001" {
		t.Fatalf("identifier = %q", taxpayer.Identifier)
	}
	registration, err := s.Register(officer, taxpayer.ID.String(), "SAMPLE_INCOME")
	if err != nil {
		t.Fatal(err)
	}
	registration, err = s.ApproveRegistration(officer, registration.ID.String())
	if err != nil {
		t.Fatal(err)
	}
	taxReturn, err := s.DraftReturn(
		taxpayerActor,
		taxpayer.ID.String(),
		registration.ID.String(),
		"2026",
		[]filing.Line{{Code: "GROSS", AmountMinor: 100_00}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.ValidateReturn(taxpayerActor, taxReturn.ID.String()); err != nil {
		t.Fatal(err)
	}
	assessment, err := s.SubmitAndAssess(context.Background(), taxpayerActor, taxReturn.ID.String())
	if err != nil {
		t.Fatal(err)
	}
	if assessment.Amount.Minor() != 10_00 {
		t.Fatalf("assessment = %d", assessment.Amount.Minor())
	}
	paymentAmount, _ := foundation.NewMoney(10_00, assessment.Amount.Currency())
	if _, err = s.RecordPayment(cashier, taxpayer.ID.String(), assessment.ID.String(), paymentAmount); err != nil {
		t.Fatal(err)
	}
	entries, err := s.Ledger(admin, taxpayer.ID.String())
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("entries = %d", len(entries))
	}
	if entries[0].PostedAt != clock.Now() || entries[1].PostedAt != clock.Now() {
		t.Fatal("ledger did not use the injected clock")
	}
	events, err := s.Audits(admin)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) < 10 {
		t.Fatalf("audits = %d", len(events))
	}
	if n.sent != 1 {
		t.Fatalf("notifications = %d", n.sent)
	}
}

func TestTenantAndJurisdictionIsolationIsMandatory(t *testing.T) {
	s := app.New(nil)
	tenantOne := scope(t, "tenant-one", "LK", "admin")
	tenantTwo := scope(t, "tenant-two", "LK", "admin")
	otherJurisdiction := scope(t, "tenant-one", "IN", "admin")
	taxpayer, err := s.CreateTaxpayer(tenantOne, "Synthetic Taxpayer", "SYN-001", "same-key")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Register(tenantTwo, taxpayer.ID.String(), "SAMPLE"); err == nil {
		t.Fatal("cross-tenant taxpayer access was allowed")
	}
	if _, err := s.Register(otherJurisdiction, taxpayer.ID.String(), "SAMPLE"); err == nil {
		t.Fatal("cross-jurisdiction taxpayer access was allowed")
	}
	other, err := s.CreateTaxpayer(tenantTwo, "Other Synthetic Taxpayer", "SYN-001", "same-key")
	if err != nil {
		t.Fatal(err)
	}
	if other.ID == taxpayer.ID {
		t.Fatal("idempotency result leaked across tenants")
	}
}

func TestApplicationBoundaryRejectsMissingContext(t *testing.T) {
	s := app.New(nil)
	if _, err := s.CreateTaxpayer(foundation.Context{}, "Name", "ID-1", "key"); err == nil {
		t.Fatal("zero domain context was accepted")
	}
	if _, err := s.Audits(foundation.Context{}); err == nil {
		t.Fatal("zero domain context was accepted by query boundary")
	}
}

func TestPaymentRejectsCurrencyMixing(t *testing.T) {
	n := &notifier{}
	s := app.New(n)
	requestScope := scope(t, "revenue", "LK", "officer")
	taxpayer, _ := s.CreateTaxpayer(requestScope, "Synthetic Taxpayer", "SYN-002", "key")
	registration, _ := s.Register(requestScope, taxpayer.ID.String(), "SAMPLE")
	registration, _ = s.ApproveRegistration(requestScope, registration.ID.String())
	taxReturn, _ := s.DraftReturn(
		requestScope,
		taxpayer.ID.String(),
		registration.ID.String(),
		"2026",
		[]filing.Line{{Code: "GROSS", AmountMinor: 100_00}},
	)
	_, _ = s.ValidateReturn(requestScope, taxReturn.ID.String())
	assessment, _ := s.SubmitAndAssess(context.Background(), requestScope, taxReturn.ID.String())
	usd, _ := foundation.NewCurrency("USD", 2)
	amount, _ := foundation.NewMoney(assessment.Amount.Minor(), usd)
	_, err := s.RecordPayment(requestScope, taxpayer.ID.String(), assessment.ID.String(), amount)
	if !errors.Is(err, foundation.ErrCurrencyMismatch) {
		t.Fatalf("error = %v", err)
	}
}
