package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	app "github.com/opencorex-org/openrevenue/internal/administration/application"
)

func TestOperationalAndAuthenticationBoundaries(t *testing.T) {
	router := Router(app.New(nil))
	health := httptest.NewRecorder()
	router.ServeHTTP(health, httptest.NewRequest(http.MethodGet, "/health", nil))
	if health.Code != http.StatusOK {
		t.Fatalf("health status = %d", health.Code)
	}
	if health.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("secure headers missing")
	}

	protected := httptest.NewRecorder()
	router.ServeHTTP(protected, httptest.NewRequest(http.MethodGet, "/api/v1/admin/audit-events", nil))
	if protected.Code != http.StatusUnauthorized {
		t.Fatalf("protected status = %d", protected.Code)
	}

	missingScopeRequest := httptest.NewRequest(http.MethodGet, "/api/v1/admin/audit-events", nil)
	missingScopeRequest.Header.Set("Authorization", "Bearer synthetic")
	missingScope := httptest.NewRecorder()
	router.ServeHTTP(missingScope, missingScopeRequest)
	if missingScope.Code != http.StatusBadRequest {
		t.Fatalf("missing scope status = %d", missingScope.Code)
	}

	validRequest := httptest.NewRequest(http.MethodGet, "/api/v1/admin/audit-events", nil)
	validRequest.Header.Set("Authorization", "Bearer synthetic")
	validRequest.Header.Set("X-Tenant-ID", "revenue")
	validRequest.Header.Set("X-Jurisdiction-Code", "LK")
	validRequest.Header.Set("X-Actor-ID", "auditor")
	validRequest.Header.Set("X-Correlation-ID", "api-test")
	valid := httptest.NewRecorder()
	router.ServeHTTP(valid, validRequest)
	if valid.Code != http.StatusOK {
		t.Fatalf("valid context status = %d body = %s", valid.Code, valid.Body)
	}

	metrics := httptest.NewRecorder()
	router.ServeHTTP(metrics, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if !strings.Contains(metrics.Body.String(), "openrevenue_domain_context_rejections_total 1") {
		t.Fatalf("domain-context metric missing: %s", metrics.Body)
	}
}
