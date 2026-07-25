package domain

import (
	"testing"
	"time"
)

func mustDate(t *testing.T, year int, month time.Month, day int) Date {
	t.Helper()
	date, err := NewDate(year, month, day)
	if err != nil {
		t.Fatal(err)
	}
	return date
}

func TestPeriodIsHalfOpen(t *testing.T) {
	start := mustDate(t, 2025, time.April, 1)
	end := mustDate(t, 2026, time.April, 1)
	period, err := NewPeriod(start, end)
	if err != nil {
		t.Fatal(err)
	}
	if !period.Contains(start) {
		t.Fatal("start must be included")
	}
	if period.Contains(end) {
		t.Fatal("end must be excluded")
	}
	if period.Days() != 365 {
		t.Fatalf("period days = %d", period.Days())
	}
}

func TestFinancialYearIsDeterministicAcrossLeapYear(t *testing.T) {
	rule, err := NewFinancialYearRule(time.April, 1)
	if err != nil {
		t.Fatal(err)
	}
	period, err := rule.Containing(mustDate(t, 2024, time.February, 29))
	if err != nil {
		t.Fatal(err)
	}
	if period.Start().String() != "2023-04-01" || period.End().String() != "2024-04-01" {
		t.Fatalf("financial year = %s/%s", period.Start(), period.End())
	}
	if period.Days() != 366 {
		t.Fatalf("leap financial year days = %d", period.Days())
	}
}

func TestInvalidDatesAndPeriodsAreRejected(t *testing.T) {
	if _, err := NewDate(2025, time.February, 29); err == nil {
		t.Fatal("invalid date accepted")
	}
	date := mustDate(t, 2025, time.January, 1)
	if _, err := NewPeriod(date, date); err == nil {
		t.Fatal("empty period accepted")
	}
	if _, err := NewFinancialYearRule(time.February, 29); err == nil {
		t.Fatal("non-annual start accepted")
	}
}

func TestFixedClockNormalizesUTC(t *testing.T) {
	location := time.FixedZone("test", 5*60*60)
	fixed, err := NewFixedClock(time.Date(2025, 7, 1, 12, 0, 0, 0, location))
	if err != nil {
		t.Fatal(err)
	}
	if fixed.Now().Location() != time.UTC || fixed.Now().Hour() != 7 {
		t.Fatalf("fixed time = %s", fixed.Now())
	}
}
