// Package application defines transactional assessment persistence boundaries.
package application

import (
	"context"

	foundation "github.com/opencorex-org/openrevenue/pkg/domain"
)

type AssessmentRecord struct {
	ID               string
	ReturnID         string
	AmountMinor      int64
	OutstandingMinor int64
	Currency         string
	PostingID        string
}

type Repository interface {
	CreateIfReturnAbsent(context.Context, foundation.Context, AssessmentRecord) (AssessmentRecord, bool, error)
	FindByID(context.Context, foundation.Context, string) (AssessmentRecord, error)
	UpdateOutstanding(context.Context, foundation.Context, string, int64) error
}
