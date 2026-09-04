package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	orgtenancy "github.com/oshethai/oshe-platform/modules/organization-tenancy"
)

type tenantCtxKey struct{}

var tenantKey = tenantCtxKey{}

// ClaimsResolver is a server-side function that resolves trusted identity claims for an incoming request.
type ClaimsResolver func(r *http.Request) (*orgtenancy.TrustedClaims, error)

// UntrustedTenantHeaders lists headers clients might attempt to use to supply or override tenant context.
var UntrustedTenantHeaders = []string{
	"X-Tenant-ID",
	"X-Tenant-Id",
	"x-tenant-id",
	"X-Tenant",
	"x-tenant",
	"Tenant-ID",
	"Tenant-Id",
	"tenant-id",
	"X-Org-ID",
	"x-org-id",
	"X-Scope-Tenant",
}

// UntrustedTenantQueryParams lists URL query parameters that untrusted clients might attempt to pass.
var UntrustedTenantQueryParams = []string{
	"tenant_id",
	"tenantId",
	"tenant",
	"org_id",
}

// detectClientOverride inspects headers and query parameters for forbidden tenant override attempts.
func detectClientOverride(r *http.Request) *orgtenancy.ClientOverrideInput {
	for _, h := range UntrustedTenantHeaders {
		if val := strings.TrimSpace(r.Header.Get(h)); val != "" {
			return &orgtenancy.ClientOverrideInput{TenantID: val}
		}
	}
	query := r.URL.Query()
	for _, q := range UntrustedTenantQueryParams {
		if val := strings.TrimSpace(query.Get(q)); val != "" {
			return &orgtenancy.ClientOverrideInput{TenantID: val}
		}
	}
	return nil
}

// writeErrorResponse emits a deterministic JSON error response and the given HTTP status code.
func writeErrorResponse(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"code":    code,
		"message": message,
	})
}

// TenantMiddleware returns an HTTP middleware that extracts and enforces trusted tenant context.
// Invariants:
// - Untrusted client-supplied tenant headers or query parameters trigger immediate 403 Forbidden.
// - Missing or unauthenticated claims trigger 401 Unauthorized.
// - Resolver errors fail closed with 401 Unauthorized.
// - Valid requests have an immutable orgtenancy.TenantContext injected into the request context.
func TenantMiddleware(resolver ClaimsResolver) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Trust Boundary TB-02: Check for forbidden client-supplied tenant override
			override := detectClientOverride(r)
			if override != nil {
				writeErrorResponse(w, http.StatusForbidden, "FORBIDDEN_TENANT_OVERRIDE", "client-supplied tenant override is strictly forbidden")
				return
			}

			if resolver == nil {
				writeErrorResponse(w, http.StatusInternalServerError, "RESOLVER_NOT_CONFIGURED", "trusted claims resolver is not configured")
				return
			}

			claims, err := resolver(r)
			if err != nil {
				writeErrorResponse(w, http.StatusUnauthorized, "CLAIMS_RESOLUTION_FAILED", err.Error())
				return
			}
			if claims == nil {
				writeErrorResponse(w, http.StatusUnauthorized, "MISSING_CLAIMS", "trusted claims required")
				return
			}

			tenantCtx, err := orgtenancy.DeriveTenantContext(claims, nil)
			if err != nil {
				if errors.Is(err, orgtenancy.ErrUnauthenticated) {
					writeErrorResponse(w, http.StatusUnauthorized, "UNAUTHENTICATED", err.Error())
					return
				}
				writeErrorResponse(w, http.StatusForbidden, "TENANT_CONTEXT_DENIED", err.Error())
				return
			}

			// Attach validated immutable tenant context to request
			ctx := context.WithValue(r.Context(), tenantKey, tenantCtx)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// TenantFromContext extracts the trusted TenantContext from context if present.
func TenantFromContext(ctx context.Context) (orgtenancy.TenantContext, bool) {
	if ctx == nil {
		return orgtenancy.TenantContext{}, false
	}
	t, ok := ctx.Value(tenantKey).(orgtenancy.TenantContext)
	return t, ok
}
