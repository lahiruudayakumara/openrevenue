// Package application defines append-only ledger transaction boundaries.
package application

import (
	"context"

	ledger "github.com/opencorex-org/openrevenue/internal/ledger/domain"
	foundation "github.com/opencorex-org/openrevenue/pkg/domain"
)

type Repository interface {
	AppendIdempotent(context.Context, foundation.Context, ledger.Posting) (ledger.Posting, bool, error)
	FindPosting(context.Context, foundation.Context, ledger.PostingID) (ledger.Posting, error)
	AppendReversal(context.Context, foundation.Context, ledger.Posting) error
	EntriesForTaxpayer(context.Context, foundation.Context, ledger.TaxpayerID) ([]ledger.Entry, error)
}

// Transaction coordinates assessment/payment state, ledger postings, audit
// facts, and outbox events in one persistence transaction.
type Transaction interface {
	Within(context.Context, func(context.Context) error) error
}
