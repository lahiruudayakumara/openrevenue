// Package application defines payment receipt and concurrency boundaries.
package application

import (
	"context"

	foundation "github.com/opencorex-org/openrevenue/pkg/domain"
)

type PaymentRecord struct {
	ID             string
	TaxpayerID     string
	AmountMinor    int64
	AllocatedMinor int64
	UnappliedMinor int64
	Currency       string
	Version        uint64
}

type Repository interface {
	CreateIdempotent(context.Context, foundation.Context, string, string, PaymentRecord) (PaymentRecord, bool, error)
	FindByID(context.Context, foundation.Context, string) (PaymentRecord, error)
	AllocateIfVersion(
		context.Context,
		foundation.Context,
		string,
		uint64,
		int64,
	) (PaymentRecord, bool, error)
}
