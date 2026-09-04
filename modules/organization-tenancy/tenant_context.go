package orgtenancy

import (
	"errors"
	"strings"
)

var (
	// ErrMissingClaims indicates that nil trusted claims were provided.
	ErrMissingClaims = errors.New("trusted claims are required")
	// ErrUnauthenticated indicates that the caller claims are not authenticated.
	ErrUnauthenticated = errors.New("claims must be authenticated")
	// ErrEmptyTenantID indicates that the authoritative tenant ID in claims is empty.
	ErrEmptyTenantID = errors.New("authoritative tenant ID must not be empty")
	// ErrTenantOverrideForbidden indicates a client attempted to supply a tenant override parameter.
	ErrTenantOverrideForbidden = errors.New("client-supplied tenant override is forbidden")
	// ErrTenantMismatch indicates that the target resource tenant does not match the trusted context.
	ErrTenantMismatch = errors.New("target tenant ID does not match trusted tenant context")
	// ErrInvalidTargetTenant indicates that the requested target tenant ID is empty or invalid.
	ErrInvalidTargetTenant = errors.New("target tenant ID must not be empty")
)

// TrustedClaims represents verified identity, tenant, and membership claims issued strictly server-side.
type TrustedClaims struct {
	Subject         string
	TenantID        string
	CompanyID       string
	SiteID          string
	IsAuthenticated bool
}

// ClientOverrideInput represents any untrusted client-supplied tenant or scope override parameter.
type ClientOverrideInput struct {
	TenantID string
}

// TenantContext is an immutable, trusted tenant scope derived strictly from server-side claims.
type TenantContext struct {
	tenantID  string
	companyID string
	siteID    string
	subject   string
}

// TenantID returns the authoritative tenant identifier.
func (c TenantContext) TenantID() string {
	return c.tenantID
}

// CompanyID returns the authoritative company identifier if present.
func (c TenantContext) CompanyID() string {
	return c.companyID
}

// SiteID returns the authoritative site identifier if present.
func (c TenantContext) SiteID() string {
	return c.siteID
}

// Subject returns the authoritative identity subject.
func (c TenantContext) Subject() string {
	return c.subject
}

// DeriveTenantContext constructs an immutable TenantContext strictly from server-side TrustedClaims.
// Trust Boundary TB-02: It rejects missing claims, unauthenticated claims, empty tenant IDs,
// and any client-supplied tenant override input.
func DeriveTenantContext(claims *TrustedClaims, clientOverride *ClientOverrideInput) (TenantContext, error) {
	if clientOverride != nil && strings.TrimSpace(clientOverride.TenantID) != "" {
		return TenantContext{}, ErrTenantOverrideForbidden
	}
	if claims == nil {
		return TenantContext{}, ErrMissingClaims
	}
	if !claims.IsAuthenticated {
		return TenantContext{}, ErrUnauthenticated
	}
	tenantID := strings.TrimSpace(claims.TenantID)
	if tenantID == "" {
		return TenantContext{}, ErrEmptyTenantID
	}

	return TenantContext{
		tenantID:  tenantID,
		companyID: strings.TrimSpace(claims.CompanyID),
		siteID:    strings.TrimSpace(claims.SiteID),
		subject:   strings.TrimSpace(claims.Subject),
	}, nil
}

// AuthorizeTenantScope evaluates whether a target resource's tenant ID matches the trusted tenant context.
// It fails closed on blank context, blank target, or mismatched tenant IDs.
func (c TenantContext) AuthorizeTenantScope(targetTenantID string) error {
	trimmedTarget := strings.TrimSpace(targetTenantID)
	if trimmedTarget == "" {
		return ErrInvalidTargetTenant
	}
	if c.tenantID == "" {
		return ErrEmptyTenantID
	}
	if c.tenantID != trimmedTarget {
		return ErrTenantMismatch
	}
	return nil
}

// IsInScope is a boolean predicate helper calling AuthorizeTenantScope.
func (c TenantContext) IsInScope(targetTenantID string) bool {
	return c.AuthorizeTenantScope(targetTenantID) == nil
}
