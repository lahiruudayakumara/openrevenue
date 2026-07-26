// Package domain owns the taxpayer aggregate and its invariants.
package domain

import (
	"errors"
	"strings"
	"time"

	foundation "github.com/opencorex-org/openrevenue/pkg/domain"
	"github.com/opencorex-org/openrevenue/pkg/id"
)

type Tag struct{}
type ID = id.ID[Tag]

type Taxpayer struct {
	ID               ID        `json:"id"`
	TenantID         string    `json:"tenantId"`
	Jurisdiction     string    `json:"jurisdiction"`
	Name             string    `json:"name"`
	Identifier       string    `json:"identifier"`
	IdentifierScheme string    `json:"identifierScheme"`
	CreatedAt        time.Time `json:"createdAt"`
}

func New(scope foundation.Context, name string, identifier foundation.TaxpayerIdentifier, now time.Time) (Taxpayer, error) {
	if err := scope.Validate(); err != nil {
		return Taxpayer{}, err
	}
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 200 {
		return Taxpayer{}, errors.New("taxpayer name must contain 1-200 characters")
	}
	if identifier.Tenant() != scope.Tenant() || identifier.Jurisdiction() != scope.Jurisdiction() {
		return Taxpayer{}, errors.New("taxpayer identifier is outside the request scope")
	}
	if now.IsZero() {
		return Taxpayer{}, errors.New("taxpayer creation time is required")
	}
	return Taxpayer{
		ID: id.New[Tag](), TenantID: scope.Tenant().String(),
		Jurisdiction: scope.Jurisdiction().String(), Name: name,
		Identifier: identifier.String(), IdentifierScheme: identifier.Scheme(),
		CreatedAt: now.UTC(),
	}, nil
}
