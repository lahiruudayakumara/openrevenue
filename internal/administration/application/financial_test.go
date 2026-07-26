package application_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	app "github.com/opencorex-org/openrevenue/internal/administration/application"
	filing "github.com/opencorex-org/openrevenue/internal/filing/domain"
	foundation "github.com/opencorex-org/openrevenue/pkg/domain"
)

func financialSlice(t *testing.T) (*app.Service, foundation.Context, app.Taxpayer, app.Assessment) {
	t.Helper()
	s := app.New(nil)
	requestScope := scope(t, "revenue", "LK", "cashier")
	taxpayerValue, err := s.CreateTaxpayer(requestScope, "Fictional Taxpayer", "demo-600", "request-600")
	if err != nil {
		t.Fatal(err)
	}
	registrationValue, _ := s.Register(requestScope, taxpayerValue.ID.String(), "SAMPLE_INCOME")
	registrationValue, _ = s.ApproveRegistration(requestScope, registrationValue.ID.String())
	taxReturn, _ := s.DraftReturn(
		requestScope, taxpayerValue.ID.String(), registrationValue.ID.String(),
		"FY-DEMO-2026", []filing.Line{{Code: "GROSS", AmountMinor: 1_000_00}},
	)
	_, _ = s.ValidateReturn(requestScope, taxReturn.ID.String())
	_, _ = s.CalculateReturn(requestScope, taxReturn.ID.String())
	assessment, err := s.SubmitAndAssess(context.Background(), requestScope, taxReturn.ID.String())
	if err != nil {
		t.Fatal(err)
	}
	replayed, err := s.SubmitAndAssess(context.Background(), requestScope, taxReturn.ID.String())
	if err != nil || replayed.ID != assessment.ID {
		t.Fatalf("assessment posting was not idempotent: %#v, %v", replayed, err)
	}
	return s, requestScope, taxpayerValue, assessment
}

func money(t *testing.T, minor int64) foundation.Money {
	t.Helper()
	currency, _ := foundation.NewCurrency("XCR", 2)
	value, err := foundation.NewMoney(minor, currency)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func TestPaymentAllocationScenarios(t *testing.T) {
	tests := []struct {
		name            string
		paymentMinor    int64
		assessmentID    bool
		wantAllocated   int64
		wantUnapplied   int64
		wantOutstanding int64
		wantNetDue      int64
	}{
		{"partial", 50_00, true, 50_00, 0, 50_00, 50_00},
		{"exact", 100_00, true, 100_00, 0, 0, 0},
		{"overpayment", 150_00, true, 100_00, 50_00, 0, -50_00},
		{"unapplied credit", 100_00, false, 0, 100_00, 100_00, 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			s, requestScope, taxpayerValue, assessment := financialSlice(t)
			assessmentID := ""
			if test.assessmentID {
				assessmentID = assessment.ID.String()
			}
			payment, err := s.RecordPayment(
				requestScope, taxpayerValue.ID.String(), assessmentID, money(t, test.paymentMinor),
			)
			if err != nil {
				t.Fatal(err)
			}
			if payment.Allocated.Minor() != test.wantAllocated ||
				payment.Unapplied.Minor() != test.wantUnapplied {
				t.Fatalf("payment = %#v", payment)
			}
			updatedAssessment, _ := s.GetAssessment(requestScope, assessment.ID.String())
			if updatedAssessment.Outstanding.Minor() != test.wantOutstanding {
				t.Fatalf("outstanding = %d", updatedAssessment.Outstanding.Minor())
			}
			balance, err := s.LedgerBalance(requestScope, taxpayerValue.ID.String())
			if err != nil || balance.NetDueMinor != test.wantNetDue {
				t.Fatalf("balance = %#v, %v", balance, err)
			}
		})
	}
}

func TestConcurrentAllocationCannotDoubleSpendPayment(t *testing.T) {
	s, requestScope, taxpayerValue, assessment := financialSlice(t)
	payment, err := s.RecordPayment(
		requestScope, taxpayerValue.ID.String(), "", money(t, 100_00),
	)
	if err != nil {
		t.Fatal(err)
	}
	var wait sync.WaitGroup
	errorsSeen := make(chan error, 2)
	allocationAmount := money(t, 100_00)
	for range 2 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			_, allocationErr := s.AllocatePayment(
				requestScope, payment.ID.String(), assessment.ID.String(),
				allocationAmount, payment.Version,
			)
			errorsSeen <- allocationErr
		}()
	}
	wait.Wait()
	close(errorsSeen)
	successes, conflicts := 0, 0
	for allocationErr := range errorsSeen {
		switch {
		case allocationErr == nil:
			successes++
		case errors.Is(allocationErr, app.ErrConflict):
			conflicts++
		default:
			t.Fatalf("unexpected allocation error: %v", allocationErr)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("successes = %d conflicts = %d", successes, conflicts)
	}
	stored, _ := s.GetPayment(requestScope, payment.ID.String())
	if stored.Allocated.Minor() != 100_00 || !stored.Unapplied.IsZero() {
		t.Fatalf("payment was double spent: %#v", stored)
	}
}

func TestPaymentCanBeAllocatedInMultipleVersionedSteps(t *testing.T) {
	s, requestScope, taxpayerValue, assessment := financialSlice(t)
	payment, _ := s.RecordPayment(requestScope, taxpayerValue.ID.String(), "", money(t, 100_00))
	first, err := s.AllocatePayment(
		requestScope, payment.ID.String(), assessment.ID.String(), money(t, 40_00), 1,
	)
	if err != nil {
		t.Fatal(err)
	}
	second, err := s.AllocatePayment(
		requestScope, payment.ID.String(), assessment.ID.String(), money(t, 60_00), first.Version,
	)
	if err != nil {
		t.Fatal(err)
	}
	if second.Version != 3 || second.Allocated.Minor() != 100_00 ||
		len(second.Allocations) != 2 ||
		second.Allocations[0].PostingID == second.Allocations[1].PostingID {
		t.Fatalf("versioned allocations = %#v", second)
	}
}

func TestPaymentReceiptIsIdempotent(t *testing.T) {
	s, requestScope, taxpayerValue, assessment := financialSlice(t)
	first, err := s.RecordPaymentIdempotent(
		requestScope, taxpayerValue.ID.String(), assessment.ID.String(),
		money(t, 100_00), "bank-reference-1",
	)
	if err != nil {
		t.Fatal(err)
	}
	replayed, err := s.RecordPaymentIdempotent(
		requestScope, taxpayerValue.ID.String(), assessment.ID.String(),
		money(t, 100_00), "bank-reference-1",
	)
	if err != nil || replayed.ID != first.ID {
		t.Fatalf("replayed payment = %#v, %v", replayed, err)
	}
	if _, err = s.RecordPaymentIdempotent(
		requestScope, taxpayerValue.ID.String(), "", money(t, 50_00), "bank-reference-1",
	); !errors.Is(err, app.ErrConflict) {
		t.Fatalf("idempotency conflict = %v", err)
	}
}

func TestPaymentAllocationAuthorizationFailsClosed(t *testing.T) {
	s := app.NewWithDependencies(
		nil, foundation.SystemClock{},
		denyingAuthorizer{permission: "payment:allocate"},
	)
	requestScope := scope(t, "revenue", "LK", "cashier")
	taxpayerValue, _ := s.CreateTaxpayer(requestScope, "Fictional Taxpayer", "demo-700", "request-700")
	registrationValue, _ := s.Register(requestScope, taxpayerValue.ID.String(), "SAMPLE_INCOME")
	registrationValue, _ = s.ApproveRegistration(requestScope, registrationValue.ID.String())
	taxReturn, _ := s.DraftReturn(
		requestScope, taxpayerValue.ID.String(), registrationValue.ID.String(),
		"FY-DEMO-2026", []filing.Line{{Code: "GROSS", AmountMinor: 1_000_00}},
	)
	_, _ = s.ValidateReturn(requestScope, taxReturn.ID.String())
	_, _ = s.CalculateReturn(requestScope, taxReturn.ID.String())
	assessment, _ := s.SubmitAndAssess(context.Background(), requestScope, taxReturn.ID.String())
	payment, _ := s.RecordPayment(requestScope, taxpayerValue.ID.String(), "", money(t, 100_00))

	if _, err := s.AllocatePayment(
		requestScope, payment.ID.String(), assessment.ID.String(),
		money(t, 100_00), payment.Version,
	); !errors.Is(err, app.ErrForbidden) {
		t.Fatalf("authorization error = %v", err)
	}
}

func TestLedgerPostingReversalIsBalancedAndSingleUse(t *testing.T) {
	s, requestScope, taxpayerValue, assessment := financialSlice(t)
	reversal, err := s.ReversePosting(requestScope, assessment.PostingID.String())
	if err != nil || reversal.ValidateBalanced() != nil {
		t.Fatalf("reversal = %#v, %v", reversal, err)
	}
	if _, err = s.ReversePosting(requestScope, assessment.PostingID.String()); !errors.Is(err, app.ErrConflict) {
		t.Fatalf("second reversal error = %v", err)
	}
	balance, _ := s.LedgerBalance(requestScope, taxpayerValue.ID.String())
	if balance.ReceivableMinor != 0 {
		t.Fatalf("reversal did not correct balance: %#v", balance)
	}
}
