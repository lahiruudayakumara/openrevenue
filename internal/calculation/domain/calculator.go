// Package domain defines the deterministic calculation boundary.
package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"math"

	foundation "github.com/opencorex-org/openrevenue/pkg/domain"
)

type InputLine struct {
	Code        string
	AmountMinor int64
}

type Input struct {
	RuleVersion string
	Currency    foundation.Currency
	Lines       []InputLine
	InputHash   string
}

type Step struct {
	Code        string `json:"code"`
	BasisMinor  int64  `json:"basisMinor"`
	RateBPS     int64  `json:"rateBasisPoints"`
	ResultMinor int64  `json:"resultMinor"`
	Rounding    string `json:"rounding"`
}

type Result struct {
	RuleVersion string           `json:"ruleVersion"`
	InputHash   string           `json:"inputHash"`
	Amount      foundation.Money `json:"amount"`
	Steps       []Step           `json:"steps"`
	ResultHash  string           `json:"resultHash"`
}

func (r Result) Validate() error {
	if r.RuleVersion == "" || r.InputHash == "" || r.ResultHash == "" || len(r.Steps) == 0 {
		return errors.New("calculation result is incomplete")
	}
	return r.Amount.Validate()
}

type Calculator interface {
	Calculate(Input) (Result, error)
}

// FlatRateCalculator is a fictional example calculator. It uses integer
// arithmetic and rounds half away from zero to the currency's minor unit.
type FlatRateCalculator struct {
	RuleVersion string
	RateBPS     int64
}

func (c FlatRateCalculator) Calculate(input Input) (Result, error) {
	if c.RuleVersion == "" || input.RuleVersion != c.RuleVersion || c.RateBPS < 0 || c.RateBPS > 10_000 {
		return Result{}, errors.New("calculation rule is incompatible")
	}
	if err := input.Currency.Validate(); err != nil {
		return Result{}, err
	}
	var basis int64
	for _, line := range input.Lines {
		if line.AmountMinor < 0 || basis > math.MaxInt64-line.AmountMinor {
			return Result{}, errors.New("calculation input is invalid or overflows")
		}
		basis += line.AmountMinor
	}
	if c.RateBPS != 0 && basis > math.MaxInt64/c.RateBPS {
		return Result{}, errors.New("calculation overflows")
	}
	product := basis * c.RateBPS
	if product > math.MaxInt64-5_000 {
		return Result{}, errors.New("calculation rounding overflows")
	}
	amountMinor := (product + 5_000) / 10_000
	amount, err := foundation.NewMoney(amountMinor, input.Currency)
	if err != nil {
		return Result{}, err
	}
	step := Step{
		Code: "FICTIONAL_FLAT_RATE", BasisMinor: basis, RateBPS: c.RateBPS,
		ResultMinor: amountMinor, Rounding: "HALF_AWAY_FROM_ZERO_TO_MINOR_UNIT",
	}
	sum := sha256.Sum256([]byte(input.RuleVersion + "\x00" + input.InputHash + "\x00" + amount.String()))
	return Result{
		RuleVersion: input.RuleVersion, InputHash: input.InputHash, Amount: amount,
		Steps: []Step{step}, ResultHash: hex.EncodeToString(sum[:]),
	}, nil
}
