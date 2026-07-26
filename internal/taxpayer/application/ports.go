// Package application defines taxpayer use-case ports independently of storage.
package application

import (
	"context"

	taxpayer "github.com/opencorex-org/openrevenue/internal/taxpayer/domain"
	foundation "github.com/opencorex-org/openrevenue/pkg/domain"
)

type Repository interface {
	Create(context.Context, foundation.Context, taxpayer.Taxpayer) error
	FindByID(context.Context, foundation.Context, taxpayer.ID) (taxpayer.Taxpayer, error)
	IdentifierExists(context.Context, foundation.Context, string, string) (bool, error)
}

type IdempotencyStore interface {
	Load(context.Context, foundation.Context, string, string) ([]byte, bool, error)
	Save(context.Context, foundation.Context, string, string, string, []byte) error
}

type Authorizer interface {
	Authorize(context.Context, foundation.Context, string, string) error
}
