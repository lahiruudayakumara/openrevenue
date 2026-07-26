package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"

	"github.com/go-chi/chi/v5"
	app "github.com/opencorex-org/openrevenue/internal/administration/application"
	filing "github.com/opencorex-org/openrevenue/internal/filing/domain"
	foundation "github.com/opencorex-org/openrevenue/pkg/domain"
	mw "github.com/opencorex-org/openrevenue/pkg/middleware"
	"github.com/opencorex-org/openrevenue/pkg/problem"
)

type Handler struct{ s *app.Service }

var (
	taxpayerCreateSuccesses         atomic.Uint64
	taxRegistrationSubmitSuccesses  atomic.Uint64
	taxRegistrationApproveSuccesses atomic.Uint64
	verticalSliceFailures           atomic.Uint64
)

func Router(s *app.Service) http.Handler {
	h := &Handler{s: s}
	r := chi.NewRouter()
	r.Use(mw.Correlation, mw.Security)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		write(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	r.Get("/ready", func(w http.ResponseWriter, r *http.Request) {
		write(w, http.StatusOK, map[string]string{"status": "ready"})
	})
	r.Get("/metrics", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		_, _ = fmt.Fprintf(
			w,
			"# HELP openrevenue_up Whether the API process is running.\n"+
				"# TYPE openrevenue_up gauge\nopenrevenue_up 1\n"+
				"# HELP openrevenue_domain_context_rejections_total Requests rejected before application dispatch because tenant, jurisdiction, actor, or correlation context was invalid.\n"+
				"# TYPE openrevenue_domain_context_rejections_total counter\n"+
				"openrevenue_domain_context_rejections_total %d\n"+
				"# HELP openrevenue_taxpayer_create_success_total Successfully created or replayed taxpayers.\n"+
				"# TYPE openrevenue_taxpayer_create_success_total counter\n"+
				"openrevenue_taxpayer_create_success_total %d\n"+
				"# HELP openrevenue_tax_registration_submit_success_total Successfully submitted tax registrations.\n"+
				"# TYPE openrevenue_tax_registration_submit_success_total counter\n"+
				"openrevenue_tax_registration_submit_success_total %d\n"+
				"# HELP openrevenue_tax_registration_approve_success_total Successfully approved tax registrations.\n"+
				"# TYPE openrevenue_tax_registration_approve_success_total counter\n"+
				"openrevenue_tax_registration_approve_success_total %d\n"+
				"# HELP openrevenue_registration_vertical_slice_failures_total Safe taxpayer and registration request failures.\n"+
				"# TYPE openrevenue_registration_vertical_slice_failures_total counter\n"+
				"openrevenue_registration_vertical_slice_failures_total %d\n",
			mw.RejectedDomainContexts(),
			taxpayerCreateSuccesses.Load(),
			taxRegistrationSubmitSuccesses.Load(),
			taxRegistrationApproveSuccesses.Load(),
			verticalSliceFailures.Load(),
		)
	})
	r.Group(func(r chi.Router) {
		r.Use(mw.Authenticate, mw.RequireDomainContext)
		r.Route("/api/v1", func(r chi.Router) {
			r.Post("/taxpayers", h.createTaxpayer)
			r.Get("/taxpayers/{taxpayerID}", h.getTaxpayer)
			r.Post("/taxpayers/{taxpayerID}/tax-registrations", h.register)
			r.Get("/tax-registrations/{registrationID}", h.getRegistration)
			r.Post("/tax-registrations/{registrationID}/approve", h.approveRegistration)
			r.Post("/returns", h.draft)
			r.Post("/returns/{returnID}/validate", h.validate)
			r.Post("/returns/{returnID}/submit", h.submit)
			r.Post("/payments", h.payment)
			r.Get("/taxpayers/{taxpayerID}/ledger", h.ledger)
			r.Get("/admin/audit-events", h.audits)
		})
	})
	return http.MaxBytesHandler(r, 1<<20)
}
func write(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	d := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		problem.Write(w, r, 400, "Invalid request", nil)
		return false
	}
	return true
}

func writeApplicationError(w http.ResponseWriter, r *http.Request, title string, err error) {
	switch {
	case errors.Is(err, app.ErrNotFound):
		problem.Write(w, r, http.StatusNotFound, "Resource not found", nil)
	case errors.Is(err, app.ErrForbidden):
		problem.Write(w, r, http.StatusForbidden, "Operation forbidden", nil)
	case errors.Is(err, app.ErrConflict), errors.Is(err, app.ErrInvalidTransition):
		problem.Write(w, r, http.StatusConflict, title, nil)
	default:
		problem.Write(w, r, http.StatusUnprocessableEntity, title, nil)
	}
}
func requestContext(r *http.Request) foundation.Context {
	scope, ok := mw.DomainContext(r.Context())
	if !ok {
		panic("domain context middleware is not installed")
	}
	return scope
}
func (h *Handler) createTaxpayer(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name       string `json:"name"`
		Identifier string `json:"identifier"`
	}
	if !decode(w, r, &in) {
		return
	}
	key := r.Header.Get("Idempotency-Key")
	if key == "" {
		problem.Write(w, r, 400, "Idempotency key required", nil)
		return
	}
	v, err := h.s.CreateTaxpayer(requestContext(r), in.Name, in.Identifier, key)
	if err != nil {
		verticalSliceFailures.Add(1)
		writeApplicationError(w, r, "Taxpayer creation failed", err)
		return
	}
	taxpayerCreateSuccesses.Add(1)
	write(w, 201, v)
}

func (h *Handler) getTaxpayer(w http.ResponseWriter, r *http.Request) {
	v, err := h.s.GetTaxpayer(requestContext(r), chi.URLParam(r, "taxpayerID"))
	if err != nil {
		writeApplicationError(w, r, "Taxpayer query failed", err)
		return
	}
	write(w, http.StatusOK, v)
}

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	var in struct {
		TaxType string `json:"taxType"`
	}
	if !decode(w, r, &in) {
		return
	}
	v, err := h.s.Register(requestContext(r), chi.URLParam(r, "taxpayerID"), in.TaxType)
	if err != nil {
		verticalSliceFailures.Add(1)
		writeApplicationError(w, r, "Registration submission failed", err)
		return
	}
	taxRegistrationSubmitSuccesses.Add(1)
	write(w, 201, v)
}

func (h *Handler) getRegistration(w http.ResponseWriter, r *http.Request) {
	v, err := h.s.GetRegistration(requestContext(r), chi.URLParam(r, "registrationID"))
	if err != nil {
		writeApplicationError(w, r, "Registration query failed", err)
		return
	}
	write(w, http.StatusOK, v)
}

func (h *Handler) approveRegistration(w http.ResponseWriter, r *http.Request) {
	v, err := h.s.ApproveRegistration(requestContext(r), chi.URLParam(r, "registrationID"))
	if err != nil {
		verticalSliceFailures.Add(1)
		writeApplicationError(w, r, "Registration approval failed", err)
		return
	}
	taxRegistrationApproveSuccesses.Add(1)
	write(w, http.StatusOK, v)
}
func (h *Handler) draft(w http.ResponseWriter, r *http.Request) {
	var in struct {
		TaxpayerID     string        `json:"taxpayerId"`
		RegistrationID string        `json:"registrationId"`
		PeriodID       string        `json:"periodId"`
		Lines          []filing.Line `json:"lines"`
	}
	if !decode(w, r, &in) {
		return
	}
	v, err := h.s.DraftReturn(requestContext(r), in.TaxpayerID, in.RegistrationID, in.PeriodID, in.Lines)
	if err != nil {
		problem.Write(w, r, 422, "Draft failed", err)
		return
	}
	write(w, 201, v)
}
func (h *Handler) validate(w http.ResponseWriter, r *http.Request) {
	v, err := h.s.ValidateReturn(requestContext(r), chi.URLParam(r, "returnID"))
	if err != nil {
		problem.Write(w, r, 422, "Validation failed", err)
		return
	}
	write(w, 200, v)
}
func (h *Handler) submit(w http.ResponseWriter, r *http.Request) {
	v, err := h.s.SubmitAndAssess(r.Context(), requestContext(r), chi.URLParam(r, "returnID"))
	if err != nil {
		problem.Write(w, r, 422, "Submission failed", err)
		return
	}
	write(w, 201, v)
}
func (h *Handler) payment(w http.ResponseWriter, r *http.Request) {
	var in struct {
		TaxpayerID   string `json:"taxpayerId"`
		AssessmentID string `json:"assessmentId"`
		AmountMinor  int64  `json:"amountMinor"`
		Currency     string `json:"currency"`
	}
	if !decode(w, r, &in) {
		return
	}
	currency, err := foundation.NewCurrency(in.Currency, 2)
	if err != nil {
		problem.Write(w, r, 422, "Payment failed", err)
		return
	}
	amount, err := foundation.NewMoney(in.AmountMinor, currency)
	if err != nil {
		problem.Write(w, r, 422, "Payment failed", err)
		return
	}
	v, err := h.s.RecordPayment(requestContext(r), in.TaxpayerID, in.AssessmentID, amount)
	if err != nil {
		problem.Write(w, r, 422, "Payment failed", err)
		return
	}
	write(w, 201, v)
}
func (h *Handler) ledger(w http.ResponseWriter, r *http.Request) {
	scope := requestContext(r)
	entries, err := h.s.Ledger(scope, chi.URLParam(r, "taxpayerID"))
	if err != nil {
		problem.Write(w, r, 422, "Ledger query failed", err)
		return
	}
	asOf, err := h.s.CurrentTime(scope)
	if err != nil {
		problem.Write(w, r, 422, "Ledger query failed", err)
		return
	}
	write(w, 200, map[string]any{"entries": entries, "asOf": asOf})
}
func (h *Handler) audits(w http.ResponseWriter, r *http.Request) {
	events, err := h.s.Audits(requestContext(r))
	if err != nil {
		problem.Write(w, r, 422, "Audit query failed", err)
		return
	}
	write(w, 200, map[string]any{"events": events})
}
