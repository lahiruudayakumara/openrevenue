package domain

import (
	"encoding/json"
	"errors"
	"math"
	"testing"
)

func TestMoneyArithmeticAndFormatting(t *testing.T) {
	xcr, _ := NewCurrency("XCR", 2)
	left, _ := NewMoney(1250, xcr)
	right, _ := NewMoney(-250, xcr)
	sum, err := left.Add(right)
	if err != nil {
		t.Fatal(err)
	}
	if sum.Minor() != 1000 || sum.String() != "XCR 10.00" {
		t.Fatalf("sum = %s (%d)", sum, sum.Minor())
	}
	difference, err := left.Subtract(right)
	if err != nil {
		t.Fatal(err)
	}
	if difference.Minor() != 1500 {
		t.Fatalf("difference = %d", difference.Minor())
	}
}

func TestMoneyRejectsCurrencyMixing(t *testing.T) {
	xcr, _ := NewCurrency("XCR", 2)
	usd, _ := NewCurrency("USD", 2)
	left, _ := NewMoney(100, xcr)
	right, _ := NewMoney(100, usd)
	if _, err := left.Add(right); !errors.Is(err, ErrCurrencyMismatch) {
		t.Fatalf("error = %v", err)
	}
}

func TestMoneyDetectsEveryOverflowDirection(t *testing.T) {
	xcr, _ := NewCurrency("XCR", 2)
	tests := []struct {
		name string
		run  func() error
	}{
		{"positive add", func() error {
			a, _ := NewMoney(math.MaxInt64, xcr)
			b, _ := NewMoney(1, xcr)
			_, err := a.Add(b)
			return err
		}},
		{"negative add", func() error {
			a, _ := NewMoney(math.MinInt64, xcr)
			b, _ := NewMoney(-1, xcr)
			_, err := a.Add(b)
			return err
		}},
		{"positive subtract", func() error {
			a, _ := NewMoney(math.MinInt64, xcr)
			b, _ := NewMoney(1, xcr)
			_, err := a.Subtract(b)
			return err
		}},
		{"negative subtract", func() error {
			a, _ := NewMoney(math.MaxInt64, xcr)
			b, _ := NewMoney(-1, xcr)
			_, err := a.Subtract(b)
			return err
		}},
		{"negate minimum", func() error {
			a, _ := NewMoney(math.MinInt64, xcr)
			_, err := a.Negate()
			return err
		}},
		{"multiply", func() error {
			a, _ := NewMoney(math.MaxInt64, xcr)
			_, err := a.Multiply(2)
			return err
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.run(); !errors.Is(err, ErrMoneyOverflow) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestMoneyJSONIncludesMinorUnitDefinition(t *testing.T) {
	currency, _ := NewCurrency("JPY", 0)
	money, _ := NewMoney(500, currency)
	encoded, err := json.Marshal(money)
	if err != nil {
		t.Fatal(err)
	}
	if string(encoded) != `{"minor":500,"currency":"JPY","minorUnits":0}` {
		t.Fatalf("json = %s", encoded)
	}
}

func TestCurrencyValidation(t *testing.T) {
	for _, test := range []struct {
		code  string
		units uint8
	}{
		{"usd", 2},
		{"US", 2},
		{"USD", 7},
	} {
		if _, err := NewCurrency(test.code, test.units); err == nil {
			t.Fatalf("accepted currency %q/%d", test.code, test.units)
		}
	}
	validCurrency, _ := NewCurrency("USD", 2)
	validMoney, _ := NewMoney(1, validCurrency)
	if _, err := (Money{}).Add(validMoney); err == nil {
		t.Fatal("zero-value money participated in arithmetic")
	}
}
