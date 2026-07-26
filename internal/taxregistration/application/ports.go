// Package application defines tax-registration use-case ports independently of storage.
package application

import (
	"context"

	registration "github.com/opencorex-org/openrevenue/internal/taxregistration/domain"
	foundation "github.com/opencorex-org/openrevenue/pkg/domain"
)

type Repository interface {
	Create(context.Context, foundation.Context, registration.Registration) error
	FindByID(context.Context, foundation.Context, registration.ID) (registration.Registration, error)
	Update(context.Context, foundation.Context, registration.Registration) error
}

type Authorizer interface {
	Authorize(context.Context, foundation.Context, string, string) error
}
