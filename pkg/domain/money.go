package domain

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"regexp"
	"strconv"
)

var currencyPattern = regexp.MustCompile(`^[A-Z]{3}$`)

var (
	ErrCurrencyMismatch = errors.New("money currencies do not match")
	ErrMoneyOverflow    = errors.New("money operation overflow")
)

type Currency struct {
	code       string
	minorUnits uint8
}

func NewCurrency(code string, minorUnits uint8) (Currency, error) {
	if !currencyPattern.MatchString(code) {
		return Currency{}, invalid("currency", "must be a three-letter uppercase code")
	}
	if minorUnits > 6 {
		return Currency{}, invalid("currency minor units", "must be between 0 and 6")
	}
	return Currency{code: code, minorUnits: minorUnits}, nil
}

func (c Currency) Code() string      { return c.code }
func (c Currency) MinorUnits() uint8 { return c.minorUnits }
func (c Currency) Validate() error {
	_, err := NewCurrency(c.code, c.minorUnits)
	return err
}

type Money struct {
	minor    int64
	currency Currency
}

func NewMoney(minor int64, currency Currency) (Money, error) {
	if err := currency.Validate(); err != nil {
		return Money{}, err
	}
	return Money{minor: minor, currency: currency}, nil
}

func (m Money) Minor() int64       { return m.minor }
func (m Money) Currency() Currency { return m.currency }
func (m Money) IsZero() bool       { return m.minor == 0 }
func (m Money) Validate() error    { return m.currency.Validate() }

func (m Money) Add(other Money) (Money, error) {
	if err := m.Validate(); err != nil {
		return Money{}, err
	}
	if err := other.Validate(); err != nil {
		return Money{}, err
	}
	if m.currency != other.currency {
		return Money{}, ErrCurrencyMismatch
	}
	if (other.minor > 0 && m.minor > math.MaxInt64-other.minor) ||
		(other.minor < 0 && m.minor < math.MinInt64-other.minor) {
		return Money{}, ErrMoneyOverflow
	}
	return Money{minor: m.minor + other.minor, currency: m.currency}, nil
}

func (m Money) Subtract(other Money) (Money, error) {
	if err := m.Validate(); err != nil {
		return Money{}, err
	}
	if err := other.Validate(); err != nil {
		return Money{}, err
	}
	if m.currency != other.currency {
		return Money{}, ErrCurrencyMismatch
	}
	if (other.minor > 0 && m.minor < math.MinInt64+other.minor) ||
		(other.minor < 0 && m.minor > math.MaxInt64+other.minor) {
		return Money{}, ErrMoneyOverflow
	}
	return Money{minor: m.minor - other.minor, currency: m.currency}, nil
}

func (m Money) Negate() (Money, error) {
	if err := m.Validate(); err != nil {
		return Money{}, err
	}
	if m.minor == math.MinInt64 {
		return Money{}, ErrMoneyOverflow
	}
	return Money{minor: -m.minor, currency: m.currency}, nil
}

func (m Money) Multiply(multiplier int64) (Money, error) {
	if err := m.Validate(); err != nil {
		return Money{}, err
	}
	product := new(big.Int).Mul(big.NewInt(m.minor), big.NewInt(multiplier))
	if !product.IsInt64() {
		return Money{}, ErrMoneyOverflow
	}
	return Money{minor: product.Int64(), currency: m.currency}, nil
}

func (m Money) String() string {
	sign := ""
	magnitude := uint64(m.minor)
	if m.minor < 0 {
		sign = "-"
		magnitude = uint64(-(m.minor + 1)) + 1
	}
	scale := uint64(1)
	for range m.currency.minorUnits {
		scale *= 10
	}
	if m.currency.minorUnits == 0 {
		return fmt.Sprintf("%s %s%d", m.currency.code, sign, magnitude)
	}
	return fmt.Sprintf(
		"%s %s%d.%0*d",
		m.currency.code,
		sign,
		magnitude/scale,
		int(m.currency.minorUnits),
		magnitude%scale,
	)
}

func (m Money) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Minor      int64  `json:"minor"`
		Currency   string `json:"currency"`
		MinorUnits uint8  `json:"minorUnits"`
	}{Minor: m.minor, Currency: m.currency.code, MinorUnits: m.currency.minorUnits})
}

func (m Money) MarshalText() ([]byte, error) {
	return []byte(strconv.FormatInt(m.minor, 10) + " " + m.currency.code), nil
}
