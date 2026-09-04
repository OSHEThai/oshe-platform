package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/oshethai/oshe-platform/apps/api"
	orgtenancy "github.com/oshethai/oshe-platform/modules/organization-tenancy"
)

func okHandler(t *testing.T, expectedTenant string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		tenantCtx, ok := api.TenantFromContext(ctx)
		if !ok {
			t.Errorf("expected TenantContext in request context, but none found")
			http.Error(w, "no tenant context", http.StatusInternalServerError)
			return
		}
		if tenantCtx.TenantID() != expectedTenant {
			t.Errorf("tenant ID mismatch: got %q, want %q", tenantCtx.TenantID(), expectedTenant)
			http.Error(w, "tenant mismatch", http.StatusForbidden)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
}

func parseResponse(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode JSON response: %v, body: %s", err, rec.Body.String())
	}
	return payload
}

func TestTenantMiddleware_MissingClaims(t *testing.T) {
	resolver := func(r *http.Request) (*orgtenancy.TrustedClaims, error) {
		return nil, nil
	}

	handlerCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
	})

	mw := api.TenantMiddleware(resolver)(next)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/inspections", nil)
	rec := httptest.NewRecorder()

	mw.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401 Unauthorized, got %d", rec.Code)
	}
	if handlerCalled {
		t.Errorf("downstream handler should not have been called")
	}

	payload := parseResponse(t, rec)
	if payload["code"] != "MISSING_CLAIMS" {
		t.Errorf("code = %v, want MISSING_CLAIMS", payload["code"])
	}
}

func TestTenantMiddleware_UnauthenticatedClaims(t *testing.T) {
	resolver := func(r *http.Request) (*orgtenancy.TrustedClaims, error) {
		return &orgtenancy.TrustedClaims{
			Subject:         "anonymous",
			TenantID:        "tenant-synth-001",
			IsAuthenticated: false,
		}, nil
	}

	handlerCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
	})

	mw := api.TenantMiddleware(resolver)(next)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/inspections", nil)
	rec := httptest.NewRecorder()

	mw.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401 Unauthorized, got %d", rec.Code)
	}
	if handlerCalled {
		t.Errorf("downstream handler should not have been called")
	}

	payload := parseResponse(t, rec)
	if payload["code"] != "UNAUTHENTICATED" {
		t.Errorf("code = %v, want UNAUTHENTICATED", payload["code"])
	}
}

func TestTenantMiddleware_ResolverError(t *testing.T) {
	resolver := func(r *http.Request) (*orgtenancy.TrustedClaims, error) {
		return nil, errors.New("signature verification failed")
	}

	handlerCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
	})

	mw := api.TenantMiddleware(resolver)(next)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/inspections", nil)
	rec := httptest.NewRecorder()

	mw.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401 Unauthorized, got %d", rec.Code)
	}
	if handlerCalled {
		t.Errorf("downstream handler should not have been called")
	}

	payload := parseResponse(t, rec)
	if payload["code"] != "CLAIMS_RESOLUTION_FAILED" {
		t.Errorf("code = %v, want CLAIMS_RESOLUTION_FAILED", payload["code"])
	}
}

func TestTenantMiddleware_HeaderOverrideRejection(t *testing.T) {
	headersToTest := []string{
		"X-Tenant-ID",
		"X-Tenant-Id",
		"x-tenant-id",
		"X-Tenant",
		"Tenant-ID",
		"X-Org-ID",
		"X-Scope-Tenant",
	}

	for _, header := range headersToTest {
		t.Run(header, func(t *testing.T) {
			// Even with valid server-side claims, client-supplied header MUST trigger immediate 403
			resolver := func(r *http.Request) (*orgtenancy.TrustedClaims, error) {
				return &orgtenancy.TrustedClaims{
					Subject:         "user-1",
					TenantID:        "tenant-trusted-alpha",
					IsAuthenticated: true,
				}, nil
			}

			handlerCalled := false
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				handlerCalled = true
			})

			mw := api.TenantMiddleware(resolver)(next)
			req := httptest.NewRequest(http.MethodGet, "/api/v1/inspections", nil)
			req.Header.Set(header, "attacker-supplied-tenant-id")
			rec := httptest.NewRecorder()

			mw.ServeHTTP(rec, req)

			if rec.Code != http.StatusForbidden {
				t.Errorf("header %q: expected status 403 Forbidden, got %d", header, rec.Code)
			}
			if handlerCalled {
				t.Errorf("header %q: downstream handler should not have been called", header)
			}

			payload := parseResponse(t, rec)
			if payload["code"] != "FORBIDDEN_TENANT_OVERRIDE" {
				t.Errorf("header %q: code = %v, want FORBIDDEN_TENANT_OVERRIDE", header, payload["code"])
			}
		})
	}
}

func TestTenantMiddleware_QueryOverrideRejection(t *testing.T) {
	queryParams := []string{
		"tenant_id",
		"tenantId",
		"tenant",
		"org_id",
	}

	for _, param := range queryParams {
		t.Run(param, func(t *testing.T) {
			resolver := func(r *http.Request) (*orgtenancy.TrustedClaims, error) {
				return &orgtenancy.TrustedClaims{
					Subject:         "user-1",
					TenantID:        "tenant-trusted-alpha",
					IsAuthenticated: true,
				}, nil
			}

			handlerCalled := false
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				handlerCalled = true
			})

			mw := api.TenantMiddleware(resolver)(next)
			req := httptest.NewRequest(http.MethodGet, "/api/v1/inspections?"+param+"=injected-tenant", nil)
			rec := httptest.NewRecorder()

			mw.ServeHTTP(rec, req)

			if rec.Code != http.StatusForbidden {
				t.Errorf("query param %q: expected status 403 Forbidden, got %d", param, rec.Code)
			}
			if handlerCalled {
				t.Errorf("query param %q: downstream handler should not have been called", param)
			}

			payload := parseResponse(t, rec)
			if payload["code"] != "FORBIDDEN_TENANT_OVERRIDE" {
				t.Errorf("query param %q: code = %v, want FORBIDDEN_TENANT_OVERRIDE", param, payload["code"])
			}
		})
	}
}

func TestTenantMiddleware_ValidAuthenticatedRequest(t *testing.T) {
	trustedTenant := "tenant-synth-corp"
	resolver := func(r *http.Request) (*orgtenancy.TrustedClaims, error) {
		return &orgtenancy.TrustedClaims{
			Subject:         "user-lead-01",
			TenantID:        trustedTenant,
			CompanyID:       "comp-01",
			SiteID:          "site-01",
			IsAuthenticated: true,
		}, nil
	}

	handler := okHandler(t, trustedTenant)
	mw := api.TenantMiddleware(resolver)(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/checklists", nil)
	rec := httptest.NewRecorder()

	mw.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 OK, got %d", rec.Code)
	}
}

func TestTenantMiddleware_CrossTenantContextBehavior(t *testing.T) {
	// Request originates with valid claims for Tenant A
	tenantA := "tenant-synth-alpha"
	resolver := func(r *http.Request) (*orgtenancy.TrustedClaims, error) {
		return &orgtenancy.TrustedClaims{
			Subject:         "user-a",
			TenantID:        tenantA,
			IsAuthenticated: true,
		}, nil
	}

	// Downstream handler simulates reading a resource bound to Tenant B
	crossTenantHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantCtx, ok := api.TenantFromContext(r.Context())
		if !ok {
			http.Error(w, "missing context", http.StatusInternalServerError)
			return
		}

		// Simulated resource belonging to Tenant B
		resourceTenant := "tenant-synth-bravo"
		if err := tenantCtx.AuthorizeTenantScope(resourceTenant); err != nil {
			// Cross-tenant access denied
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"code":    "CROSS_TENANT_ACCESS_DENIED",
				"message": err.Error(),
			})
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	mw := api.TenantMiddleware(resolver)(crossTenantHandler)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/resources/res-123", nil)
	rec := httptest.NewRecorder()

	mw.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden for cross-tenant resource, got %d", rec.Code)
	}

	payload := parseResponse(t, rec)
	if payload["code"] != "CROSS_TENANT_ACCESS_DENIED" {
		t.Errorf("code = %v, want CROSS_TENANT_ACCESS_DENIED", payload["code"])
	}
}

func TestTenantFromContext_NilOrEmptyContext(t *testing.T) {
	ctx, ok := api.TenantFromContext(nil)
	if ok || ctx.TenantID() != "" {
		t.Errorf("TenantFromContext(nil) = (%v, %v), want zero, false", ctx, ok)
	}

	ctx, ok = api.TenantFromContext(context.Background())
	if ok || ctx.TenantID() != "" {
		t.Errorf("TenantFromContext(background) = (%v, %v), want zero, false", ctx, ok)
	}
}
