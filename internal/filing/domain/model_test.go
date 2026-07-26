package domain_test

import (
	"reflect"
	"testing"
	"time"

	calculation "github.com/opencorex-org/openrevenue/internal/calculation/domain"
	filing "github.com/opencorex-org/openrevenue/internal/filing/domain"
	foundation "github.com/opencorex-org/openrevenue/pkg/domain"
)

func filingScope(t *testing.T) foundation.Context {
	t.Helper()
	tenant, _ := foundation.NewTenantID("revenue")
	jurisdiction, _ := foundation.NewJurisdiction("LK")
	actor, _ := foundation.NewActor(foundation.ActorUser, "taxpayer")
	correlation, _ := foundation.NewCorrelationID("filing-test")
	value, err := foundation.NewContext(tenant, jurisdiction, actor, correlation)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func TestValidationUsesStableFieldCodes(t *testing.T) {
	value, err := filing.NewVersioned(
		filingScope(t), "taxpayer-1", "registration-1", "FY-DEMO-2026",
		filing.DefaultFormVersion, filing.DefaultRuleVersion,
		[]filing.Line{
			{Code: "UNKNOWN", AmountMinor: 1},
			{Code: "GROSS", AmountMinor: -1},
			{Code: "GROSS", AmountMinor: 2},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	result := value.ValidateAgainst(filing.DefaultForm())
	want := []filing.ValidationIssue{
		{Field: "GROSS", Code: "AMOUNT_NEGATIVE"},
		{Field: "GROSS", Code: "FIELD_DUPLICATE"},
		{Field: "UNKNOWN", Code: "FIELD_NOT_IN_FORM"},
	}
	if result.Valid || !reflect.DeepEqual(result.Issues, want) {
		t.Fatalf("validation = %#v", result)
	}
}

func TestSubmissionFreezesVersionsAndPayload(t *testing.T) {
	lines := []filing.Line{{Code: "GROSS", AmountMinor: 100_00}}
	value, _ := filing.New(filingScope(t), "taxpayer-1", "registration-1", "FY-DEMO-2026", lines)
	lines[0].AmountMinor = 999_00
	if value.Lines[0].AmountMinor != 100_00 {
		t.Fatal("draft retained caller-owned payload")
	}
	result := value.ValidateAgainst(filing.DefaultForm())
	currency, _ := foundation.NewCurrency("XCR", 2)
	calculated, _ := (calculation.FlatRateCalculator{
		RuleVersion: filing.DefaultRuleVersion, RateBPS: 1_000,
	}).Calculate(calculation.Input{
		RuleVersion: filing.DefaultRuleVersion, Currency: currency,
		Lines:     []calculation.InputLine{{Code: "GROSS", AmountMinor: 100_00}},
		InputHash: result.PayloadHash,
	})
	if err := value.RecordCalculation(calculated); err != nil {
		t.Fatal(err)
	}
	if err := value.Submit(time.Now(), "sample-income-v2", filing.DefaultRuleVersion); err == nil {
		t.Fatal("stale form version was accepted")
	}
	if err := value.Submit(time.Now(), filing.DefaultFormVersion, filing.DefaultRuleVersion); err != nil {
		t.Fatal(err)
	}
	if value.FrozenPayloadHash == "" || value.Status != filing.Submitted {
		t.Fatalf("submitted return = %#v", value)
	}
}

func TestAmendmentPreservesOriginalHistory(t *testing.T) {
	value, _ := filing.New(filingScope(t), "taxpayer-1", "registration-1", "FY-DEMO-2026", []filing.Line{{Code: "GROSS", AmountMinor: 100}})
	validation := value.ValidateAgainst(filing.DefaultForm())
	currency, _ := foundation.NewCurrency("XCR", 2)
	result, _ := (calculation.FlatRateCalculator{
		RuleVersion: filing.DefaultRuleVersion, RateBPS: 1_000,
	}).Calculate(calculation.Input{
		RuleVersion: filing.DefaultRuleVersion, Currency: currency,
		Lines:     []calculation.InputLine{{Code: "GROSS", AmountMinor: 100}},
		InputHash: validation.PayloadHash,
	})
	_ = value.RecordCalculation(result)
	_ = value.Submit(time.Now(), filing.DefaultFormVersion, filing.DefaultRuleVersion)

	amendment, err := value.Amend(filingScope(t))
	if err != nil {
		t.Fatal(err)
	}
	if amendment.ID == value.ID || amendment.Revision != 2 ||
		amendment.OriginalReturnID != value.ID.String() ||
		amendment.SupersedesID != value.ID.String() ||
		amendment.Status != filing.Draft {
		t.Fatalf("amendment = %#v", amendment)
	}
	if value.Status != filing.Submitted || value.Revision != 1 {
		t.Fatalf("original was mutated: %#v", value)
	}
}
