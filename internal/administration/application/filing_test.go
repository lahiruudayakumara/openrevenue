package application_test

import (
	"errors"
	"testing"

	app "github.com/opencorex-org/openrevenue/internal/administration/application"
	filing "github.com/opencorex-org/openrevenue/internal/filing/domain"
	foundation "github.com/opencorex-org/openrevenue/pkg/domain"
)

func TestReturnCalculationAuthorizationFailsClosed(t *testing.T) {
	s := app.NewWithDependencies(
		nil,
		foundation.SystemClock{},
		denyingAuthorizer{permission: "return:calculate"},
	)
	requestScope := scope(t, "revenue", "LK", "officer")
	taxpayerValue, _ := s.CreateTaxpayer(requestScope, "Fictional Taxpayer", "demo-500", "request-500")
	registrationValue, _ := s.Register(requestScope, taxpayerValue.ID.String(), "SAMPLE_INCOME")
	registrationValue, _ = s.ApproveRegistration(requestScope, registrationValue.ID.String())
	taxReturn, _ := s.DraftReturn(
		requestScope, taxpayerValue.ID.String(), registrationValue.ID.String(),
		"FY-DEMO-2026", []filing.Line{{Code: "GROSS", AmountMinor: 100_00}},
	)
	_, _ = s.ValidateReturn(requestScope, taxReturn.ID.String())

	if _, err := s.CalculateReturn(requestScope, taxReturn.ID.String()); !errors.Is(err, app.ErrForbidden) {
		t.Fatalf("authorization error = %v", err)
	}
}
