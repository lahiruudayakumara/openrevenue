package domain

import "testing"

func TestTaxpayerIdentifierNormalizationAndScope(t *testing.T) {
	tenant, _ := NewTenantID("revenue")
	jurisdiction, _ := NewJurisdiction("LK")
	identifier, err := NewTaxpayerIdentifier(
		tenant,
		jurisdiction,
		"TIN",
		" ab-12 34 ",
		UpperAlphanumericNormalizer{},
	)
	if err != nil {
		t.Fatal(err)
	}
	if identifier.String() != "AB1234" {
		t.Fatalf("normalized identifier = %q", identifier.String())
	}
	if identifier.Tenant() != tenant || identifier.Jurisdiction() != jurisdiction {
		t.Fatal("identifier lost its isolation scope")
	}
}

func TestTaxpayerIdentifierRejectsInvalidInput(t *testing.T) {
	tenant, _ := NewTenantID("revenue")
	jurisdiction, _ := NewJurisdiction("LK")
	tests := []struct {
		name       string
		scheme     string
		value      string
		normalizer IdentifierNormalizer
	}{
		{"missing normalizer", "TIN", "123", nil},
		{"invalid scheme", "tin", "123", UpperAlphanumericNormalizer{}},
		{"empty normalized value", "TIN", " - ", UpperAlphanumericNormalizer{}},
		{"unsupported character", "TIN", "ABC/123", UpperAlphanumericNormalizer{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := NewTaxpayerIdentifier(tenant, jurisdiction, tt.scheme, tt.value, tt.normalizer); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}
