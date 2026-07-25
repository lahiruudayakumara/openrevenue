package domain

import (
	"fmt"
	"time"
)

type Date struct {
	year  int
	month time.Month
	day   int
}

func NewDate(year int, month time.Month, day int) (Date, error) {
	candidate := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	if year < 1 || candidate.Year() != year || candidate.Month() != month || candidate.Day() != day {
		return Date{}, invalid("date", "is not a valid Gregorian calendar date")
	}
	return Date{year: year, month: month, day: day}, nil
}

func DateFromTime(value time.Time) Date {
	year, month, day := value.UTC().Date()
	return Date{year: year, month: month, day: day}
}

func (d Date) Year() int         { return d.year }
func (d Date) Month() time.Month { return d.month }
func (d Date) Day() int          { return d.day }
func (d Date) Time() time.Time {
	return time.Date(d.year, d.month, d.day, 0, 0, 0, 0, time.UTC)
}
func (d Date) String() string { return fmt.Sprintf("%04d-%02d-%02d", d.year, d.month, d.day) }
func (d Date) Validate() error {
	_, err := NewDate(d.year, d.month, d.day)
	return err
}
func (d Date) Before(other Date) bool {
	return d.Time().Before(other.Time())
}

// Period is a half-open interval: Start is included and End is excluded.
type Period struct {
	start Date
	end   Date
}

func NewPeriod(start, end Date) (Period, error) {
	if err := start.Validate(); err != nil {
		return Period{}, err
	}
	if err := end.Validate(); err != nil {
		return Period{}, err
	}
	if !start.Before(end) {
		return Period{}, invalid("period", "end must be after start")
	}
	return Period{start: start, end: end}, nil
}

func (p Period) Start() Date { return p.start }
func (p Period) End() Date   { return p.end }
func (p Period) Validate() error {
	_, err := NewPeriod(p.start, p.end)
	return err
}
func (p Period) Contains(date Date) bool {
	if p.Validate() != nil || date.Validate() != nil {
		return false
	}
	return !date.Before(p.start) && date.Before(p.end)
}
func (p Period) Days() int {
	if p.Validate() != nil {
		return 0
	}
	return int(p.end.Time().Sub(p.start.Time()).Hours() / 24)
}

// FinancialYearRule defines a deterministic annual calendar. February 29 is
// rejected because it does not exist in every financial year.
type FinancialYearRule struct {
	startMonth time.Month
	startDay   int
}

func NewFinancialYearRule(startMonth time.Month, startDay int) (FinancialYearRule, error) {
	if _, err := NewDate(2001, startMonth, startDay); err != nil {
		return FinancialYearRule{}, invalid("financial year start", "must exist in every Gregorian year")
	}
	return FinancialYearRule{startMonth: startMonth, startDay: startDay}, nil
}

func (r FinancialYearRule) ForStartYear(year int) (Period, error) {
	start, err := NewDate(year, r.startMonth, r.startDay)
	if err != nil {
		return Period{}, err
	}
	end, err := NewDate(year+1, r.startMonth, r.startDay)
	if err != nil {
		return Period{}, err
	}
	return NewPeriod(start, end)
}

func (r FinancialYearRule) Containing(date Date) (Period, error) {
	year := date.Year()
	start, err := NewDate(year, r.startMonth, r.startDay)
	if err != nil {
		return Period{}, err
	}
	if date.Before(start) {
		year--
	}
	return r.ForStartYear(year)
}
