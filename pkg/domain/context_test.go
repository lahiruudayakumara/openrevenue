package domain

import "testing"

func TestContextValueValidation(t *testing.T) {
	tests := []struct {
		name string
		run  func() error
	}{
		{"empty tenant", func() error { _, err := NewTenantID(""); return err }},
		{"uppercase tenant", func() error { _, err := NewTenantID("Revenue"); return err }},
		{"tenant whitespace", func() error { _, err := NewTenantID(" revenue"); return err }},
		{"lowercase jurisdiction", func() error { _, err := NewJurisdiction("lk"); return err }},
		{"unknown actor kind", func() error { _, err := NewActor("ROBOT", "worker"); return err }},
		{"unsafe actor id", func() error { _, err := NewActor(ActorUser, "user\nadmin"); return err }},
		{"unsafe correlation", func() error { _, err := NewCorrelationID("trace value"); return err }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.run(); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestContextRequiresEveryDimension(t *testing.T) {
	if _, err := NewTenantID("ab"); err != nil {
		t.Fatalf("two-character tenant was rejected: %v", err)
	}
	tenant, _ := NewTenantID("revenue")
	jurisdiction, _ := NewJurisdiction("LK")
	actor, _ := NewActor(ActorUser, "officer-1")
	correlation, _ := NewCorrelationID("request-1")

	context, err := NewContext(tenant, jurisdiction, actor, correlation)
	if err != nil {
		t.Fatal(err)
	}
	if context.IsolationKey("record") == "record" {
		t.Fatal("isolation key is not scoped")
	}

	if _, err := NewContext(TenantID{}, jurisdiction, actor, correlation); err == nil {
		t.Fatal("zero tenant was accepted")
	}
	if _, err := NewContext(tenant, Jurisdiction{}, actor, correlation); err == nil {
		t.Fatal("zero jurisdiction was accepted")
	}
}
