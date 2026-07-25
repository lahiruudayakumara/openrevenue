package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync/atomic"

	foundation "github.com/opencorex-org/openrevenue/pkg/domain"
	"github.com/opencorex-org/openrevenue/pkg/id"
)

type correlationTag struct{}

type key string

const CorrelationKey key = "correlation-id"
const domainContextKey key = "domain-context"

var rejectedDomainContexts atomic.Uint64

func Security(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		next.ServeHTTP(w, r)
	})
}
func Correlation(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		v := r.Header.Get("X-Correlation-ID")
		if v == "" {
			v = id.New[correlationTag]().String()
		}
		w.Header().Set("X-Correlation-ID", v)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), CorrelationKey, v)))
	})
}
func Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") && r.Header.Get("Authorization") == "" {
			http.Error(w, "authorization required", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
func CorrelationID(ctx context.Context) string { v, _ := ctx.Value(CorrelationKey).(string); return v }

func RequireDomainContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenant, err := foundation.NewTenantID(r.Header.Get("X-Tenant-ID"))
		if err != nil {
			writeContextProblem(w, "A valid X-Tenant-ID header is required.")
			return
		}
		jurisdiction, err := foundation.NewJurisdiction(r.Header.Get("X-Jurisdiction-Code"))
		if err != nil {
			writeContextProblem(w, "A valid X-Jurisdiction-Code header is required.")
			return
		}
		actorKind := foundation.ActorKind(r.Header.Get("X-Actor-Type"))
		if actorKind == "" {
			actorKind = foundation.ActorUser
		}
		actor, err := foundation.NewActor(actorKind, r.Header.Get("X-Actor-ID"))
		if err != nil {
			writeContextProblem(w, "A valid X-Actor-ID and X-Actor-Type are required.")
			return
		}
		correlation, err := foundation.NewCorrelationID(CorrelationID(r.Context()))
		if err != nil {
			writeContextProblem(w, "X-Correlation-ID is invalid.")
			return
		}
		scope, err := foundation.NewContext(tenant, jurisdiction, actor, correlation)
		if err != nil {
			writeContextProblem(w, "The request domain context is invalid.")
			return
		}
		ctx := context.WithValue(r.Context(), domainContextKey, scope)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func DomainContext(ctx context.Context) (foundation.Context, bool) {
	scope, ok := ctx.Value(domainContextKey).(foundation.Context)
	return scope, ok
}

func writeContextProblem(w http.ResponseWriter, detail string) {
	rejectedDomainContexts.Add(1)
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(http.StatusBadRequest)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"type":   "https://openrevenue.org/problems/domain-context",
		"title":  "Invalid domain context",
		"status": http.StatusBadRequest,
		"detail": detail,
	})
}

func RejectedDomainContexts() uint64 {
	return rejectedDomainContexts.Load()
}
