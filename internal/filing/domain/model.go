package domain

import (
	"errors"
	"time"

	foundation "github.com/opencorex-org/openrevenue/pkg/domain"
	"github.com/opencorex-org/openrevenue/pkg/id"
)

type ReturnTag struct{}
type TaxReturnID = id.ID[ReturnTag]
type Status string

const (
	Draft     Status = "DRAFT"
	Validated Status = "VALIDATED"
	Submitted Status = "SUBMITTED"
)

type Line struct {
	Code        string `json:"code"`
	AmountMinor int64  `json:"amountMinor"`
}
type TaxReturn struct {
	ID             TaxReturnID `json:"id"`
	TenantID       string      `json:"tenantId"`
	Jurisdiction   string      `json:"jurisdiction"`
	TaxpayerID     string      `json:"taxpayerId"`
	RegistrationID string      `json:"registrationId"`
	PeriodID       string      `json:"periodId"`
	FormVersion    string      `json:"formVersion"`
	RuleVersion    string      `json:"ruleVersion"`
	Lines          []Line      `json:"lines"`
	Status         Status      `json:"status"`
	SubmittedAt    *time.Time  `json:"submittedAt,omitempty"`
}

func New(scope foundation.Context, taxpayer, registration, period string, lines []Line) (TaxReturn, error) {
	if err := scope.Validate(); err != nil {
		return TaxReturn{}, err
	}
	if taxpayer == "" || registration == "" || period == "" {
		return TaxReturn{}, errors.New("taxpayer, registration, and period are required")
	}
	return TaxReturn{
		ID: id.New[ReturnTag](), TenantID: scope.Tenant().String(),
		Jurisdiction: scope.Jurisdiction().String(), TaxpayerID: taxpayer,
		RegistrationID: registration, PeriodID: period,
		FormVersion: "sample-income-v1", RuleVersion: "fictional-flat-rate-v1",
		Lines: lines, Status: Draft,
	}, nil
}
func (r *TaxReturn) Validate() error {
	if r.Status != Draft {
		return errors.New("only drafts can be validated")
	}
	if len(r.Lines) == 0 {
		return errors.New("at least one line is required")
	}
	for _, l := range r.Lines {
		if l.AmountMinor < 0 {
			return errors.New("line amount cannot be negative")
		}
	}
	r.Status = Validated
	return nil
}
func (r *TaxReturn) Submit(now time.Time) error {
	if r.Status != Validated {
		return errors.New("return must be validated")
	}
	if now.IsZero() {
		return errors.New("submission time is required")
	}
	now = now.UTC()
	r.Status = Submitted
	r.SubmittedAt = &now
	return nil
}
