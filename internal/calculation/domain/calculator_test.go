package domain_test

import (
	"reflect"
	"testing"

	calculation "github.com/opencorex-org/openrevenue/internal/calculation/domain"
	foundation "github.com/opencorex-org/openrevenue/pkg/domain"
)

func TestFlatRateCalculationIsDeterministicAndExplainable(t *testing.T) {
	currency, _ := foundation.NewCurrency("XCR", 2)
	calculator := calculation.FlatRateCalculator{
		RuleVersion: "fictional-flat-rate-v1", RateBPS: 1_000,
	}
	input := calculation.Input{
		RuleVersion: "fictional-flat-rate-v1", Currency: currency,
		Lines:     []calculation.InputLine{{Code: "GROSS", AmountMinor: 10_005}},
		InputHash: "stable-input",
	}
	first, err := calculator.Calculate(input)
	if err != nil {
		t.Fatal(err)
	}
	second, err := calculator.Calculate(input)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("results differ: %#v != %#v", first, second)
	}
	if first.Amount.Minor() != 1_001 {
		t.Fatalf("rounded amount = %d", first.Amount.Minor())
	}
	if len(first.Steps) != 1 ||
		first.Steps[0].Rounding != "HALF_AWAY_FROM_ZERO_TO_MINOR_UNIT" ||
		first.ResultHash == "" {
		t.Fatalf("result is not explainable: %#v", first)
	}
}

func TestCalculationRejectsIncompatibleRule(t *testing.T) {
	currency, _ := foundation.NewCurrency("XCR", 2)
	_, err := (calculation.FlatRateCalculator{
		RuleVersion: "fictional-flat-rate-v1", RateBPS: 1_000,
	}).Calculate(calculation.Input{
		RuleVersion: "fictional-flat-rate-v2", Currency: currency,
		InputHash: "stable-input",
	})
	if err == nil {
		t.Fatal("incompatible rule was accepted")
	}
}
