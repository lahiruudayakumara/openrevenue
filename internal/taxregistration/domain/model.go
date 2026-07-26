// Package domain owns tax-registration lifecycle rules.
package domain

import (
	"errors"
	"regexp"
	"strings"
	"time"

	foundation "github.com/opencorex-org/openrevenue/pkg/domain"
	"github.com/opencorex-org/openrevenue/pkg/id"
)

type Tag struct{}
type ID = id.ID[Tag]

type Status string

const (
	StatusSubmitted Status = "SUBMITTED"
	StatusApproved  Status = "APPROVED"
)

var taxTypePattern = regexp.MustCompile(`^[A-Z][A-Z0-9_]{1,63}$`)

type Registration struct {
	ID           ID         `json:"id"`
	TenantID     string     `json:"tenantId"`
	Jurisdiction string     `json:"jurisdiction"`
	TaxpayerID   string     `json:"taxpayerId"`
	TaxType      string     `json:"taxType"`
	Status       Status     `json:"status"`
	SubmittedAt  time.Time  `json:"submittedAt"`
	ApprovedAt   *time.Time `json:"approvedAt,omitempty"`
	ApprovedBy   string     `json:"approvedBy,omitempty"`
}

func Submit(scope foundation.Context, taxpayerID, taxType string, now time.Time) (Registration, error) {
	if err := scope.Validate(); err != nil {
		return Registration{}, err
	}
	taxpayerID = strings.TrimSpace(taxpayerID)
	taxType = strings.TrimSpace(taxType)
	if taxpayerID == "" {
		return Registration{}, errors.New("taxpayer id is required")
	}
	if !taxTypePattern.MatchString(taxType) {
		return Registration{}, errors.New("tax type must be a 2-64 character uppercase code")
	}
	if now.IsZero() {
		return Registration{}, errors.New("registration submission time is required")
	}
	return Registration{
		ID: id.New[Tag](), TenantID: scope.Tenant().String(),
		Jurisdiction: scope.Jurisdiction().String(), TaxpayerID: taxpayerID,
		TaxType: taxType, Status: StatusSubmitted, SubmittedAt: now.UTC(),
	}, nil
}

func (r *Registration) Approve(scope foundation.Context, now time.Time) error {
	if err := scope.Validate(); err != nil {
		return err
	}
	if r.TenantID != scope.Tenant().String() || r.Jurisdiction != scope.Jurisdiction().String() {
		return errors.New("registration is outside the request scope")
	}
	if r.Status != StatusSubmitted {
		return errors.New("only a submitted registration can be approved")
	}
	if now.IsZero() {
		return errors.New("registration approval time is required")
	}
	approvedAt := now.UTC()
	r.Status = StatusApproved
	r.ApprovedAt = &approvedAt
	r.ApprovedBy = scope.Actor().ID()
	return nil
}
