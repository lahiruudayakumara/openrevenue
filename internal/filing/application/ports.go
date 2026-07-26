// Package application defines storage and version-catalog boundaries for filing.
package application

import (
	"context"

	filing "github.com/opencorex-org/openrevenue/internal/filing/domain"
	foundation "github.com/opencorex-org/openrevenue/pkg/domain"
)

type Repository interface {
	Create(context.Context, foundation.Context, filing.TaxReturn) error
	FindByID(context.Context, foundation.Context, filing.TaxReturnID) (filing.TaxReturn, error)
	Update(context.Context, foundation.Context, filing.TaxReturn) error
	History(context.Context, foundation.Context, filing.TaxReturnID) ([]filing.TaxReturn, error)
}

type FormCatalog interface {
	FindImmutable(context.Context, foundation.Context, string) (filing.FormDefinition, error)
	CurrentVersions(context.Context, foundation.Context, string) (formVersion, ruleVersion string, err error)
}
