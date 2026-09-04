package orgtenancy

import (
	"errors"
	"testing"
)

func TestDeriveTenantContext_MissingClaims(t *testing.T) {
	_, err := DeriveTenantContext(nil, nil)
	if !errors.Is(err, ErrMissingClaims) {
		t.Fatalf("expected ErrMissingClaims, got %v", err)
	}
}

func TestDeriveTenantContext_Unauthenticated(t *testing.T) {
	claims := &TrustedClaims{
		Subject:         "user-1",
		TenantID:        "tenant-synth-001",
		IsAuthenticated: false,
	}
	_, err := DeriveTenantContext(claims, nil)
	if !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("expected ErrUnauthenticated, got %v", err)
	}
}

func TestDeriveTenantContext_EmptyTenantID(t *testing.T) {
	testCases := []struct {
		name     string
		tenantID string
	}{
		{"empty string", ""},
		{"whitespace only", "   \t\n  "},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			claims := &TrustedClaims{
				Subject:         "user-1",
				TenantID:        tc.tenantID,
				IsAuthenticated: true,
			}
			_, err := DeriveTenantContext(claims, nil)
			if !errors.Is(err, ErrEmptyTenantID) {
				t.Fatalf("expected ErrEmptyTenantID, got %v", err)
			}
		})
	}
}

func TestDeriveTenantContext_OverrideDenial(t *testing.T) {
	claims := &TrustedClaims{
		Subject:         "user-1",
		TenantID:        "tenant-trusted-001",
		IsAuthenticated: true,
	}
	override := &ClientOverrideInput{
		TenantID: "tenant-attacker-override",
	}

	_, err := DeriveTenantContext(claims, override)
	if !errors.Is(err, ErrTenantOverrideForbidden) {
		t.Fatalf("expected ErrTenantOverrideForbidden, got %v", err)
	}

	// Override check must fail even if claims are unauthenticated or missing
	_, err = DeriveTenantContext(nil, override)
	if !errors.Is(err, ErrTenantOverrideForbidden) {
		t.Fatalf("expected ErrTenantOverrideForbidden, got %v", err)
	}
}

func TestDeriveTenantContext_Valid(t *testing.T) {
	claims := &TrustedClaims{
		Subject:         "user-admin-01",
		TenantID:        "tenant-synth-alpha",
		CompanyID:       "company-synth-01",
		SiteID:          "site-synth-01",
		IsAuthenticated: true,
	}

	ctx, err := DeriveTenantContext(claims, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ctx.TenantID() != "tenant-synth-alpha" {
		t.Errorf("TenantID() = %q, want %q", ctx.TenantID(), "tenant-synth-alpha")
	}
	if ctx.CompanyID() != "company-synth-01" {
		t.Errorf("CompanyID() = %q, want %q", ctx.CompanyID(), "company-synth-01")
	}
	if ctx.SiteID() != "site-synth-01" {
		t.Errorf("SiteID() = %q, want %q", ctx.SiteID(), "site-synth-01")
	}
	if ctx.Subject() != "user-admin-01" {
		t.Errorf("Subject() = %q, want %q", ctx.Subject(), "user-admin-01")
	}
}

func TestAuthorizeTenantScope_SameTenant(t *testing.T) {
	claims := &TrustedClaims{
		Subject:         "user-1",
		TenantID:        "tenant-synth-001",
		IsAuthenticated: true,
	}
	ctx, err := DeriveTenantContext(claims, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := ctx.AuthorizeTenantScope("tenant-synth-001"); err != nil {
		t.Errorf("expected authorized same-tenant access, got error: %v", err)
	}
	if !ctx.IsInScope("tenant-synth-001") {
		t.Errorf("IsInScope() = false, want true")
	}
}

func TestAuthorizeTenantScope_CrossTenantDenial(t *testing.T) {
	claimsTenantA := &TrustedClaims{
		Subject:         "user-tenant-a",
		TenantID:        "tenant-alpha",
		IsAuthenticated: true,
	}
	ctxA, err := DeriveTenantContext(claimsTenantA, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Tenant A attempts to access Tenant B resource
	targetTenantB := "tenant-bravo"
	err = ctxA.AuthorizeTenantScope(targetTenantB)
	if !errors.Is(err, ErrTenantMismatch) {
		t.Fatalf("expected ErrTenantMismatch for cross-tenant access, got %v", err)
	}
	if ctxA.IsInScope(targetTenantB) {
		t.Errorf("IsInScope() = true for cross-tenant resource, want false")
	}
}

func TestAuthorizeTenantScope_EmptyTargetDenial(t *testing.T) {
	claims := &TrustedClaims{
		Subject:         "user-1",
		TenantID:        "tenant-alpha",
		IsAuthenticated: true,
	}
	ctx, err := DeriveTenantContext(claims, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, target := range []string{"", "   ", "\t\n"} {
		if err := ctx.AuthorizeTenantScope(target); !errors.Is(err, ErrInvalidTargetTenant) {
			t.Errorf("expected ErrInvalidTargetTenant for empty target %q, got %v", target, err)
		}
		if ctx.IsInScope(target) {
			t.Errorf("IsInScope(%q) = true, want false", target)
		}
	}
}

func TestAuthorizeTenantScope_EmptyContextDenial(t *testing.T) {
	var zeroCtx TenantContext
	if err := zeroCtx.AuthorizeTenantScope("tenant-alpha"); !errors.Is(err, ErrEmptyTenantID) {
		t.Fatalf("expected ErrEmptyTenantID for zero TenantContext, got %v", err)
	}
	if zeroCtx.IsInScope("tenant-alpha") {
		t.Errorf("IsInScope() = true on zero context, want false")
	}
}

func TestAuthorizeTenantScope_PrefixCollisionDenial(t *testing.T) {
	claims := &TrustedClaims{
		Subject:         "user-1",
		TenantID:        "tenant-alpha",
		IsAuthenticated: true,
	}
	ctx, err := DeriveTenantContext(claims, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	subTargets := []string{
		"tenant-alpha-2",
		"tenant-alpha_suffix",
		"tenant-alphax",
		"tenant-alph",
	}

	for _, target := range subTargets {
		t.Run(target, func(t *testing.T) {
			if err := ctx.AuthorizeTenantScope(target); !errors.Is(err, ErrTenantMismatch) {
				t.Fatalf("expected ErrTenantMismatch for prefix collision %q, got %v", target, err)
			}
			if ctx.IsInScope(target) {
				t.Errorf("IsInScope(%q) = true, want false", target)
			}
		})
	}
}
