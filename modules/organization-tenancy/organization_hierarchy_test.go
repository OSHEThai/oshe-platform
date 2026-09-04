package orgtenancy

import (
	"errors"
	"testing"
)

func TestHierarchy_ValidEndToEndConstruction(t *testing.T) {
	tenantID := "tenant-synth-001"

	company, err := NewCompany(tenantID, "comp-01", "Acme Industrial")
	if err != nil {
		t.Fatalf("unexpected NewCompany error: %v", err)
	}
	if company.TenantID() != tenantID || company.CompanyID() != "comp-01" || company.Name() != "Acme Industrial" {
		t.Errorf("company fields mismatch: %+v", company)
	}
	if !company.IsActive() || company.State() != StateActive {
		t.Errorf("expected company active, got %v", company.State())
	}

	project, err := NewProject(company, "proj-alpha", "Bangkok Plant Expansion")
	if err != nil {
		t.Fatalf("unexpected NewProject error: %v", err)
	}
	if project.TenantID() != tenantID || project.CompanyID() != "comp-01" || project.ProjectID() != "proj-alpha" {
		t.Errorf("project fields mismatch: %+v", project)
	}
	if !project.IsActive() {
		t.Errorf("expected project active")
	}

	site, err := NewSite(project, "site-rayong-01", "Rayong East Complex")
	if err != nil {
		t.Fatalf("unexpected NewSite error: %v", err)
	}
	if site.TenantID() != tenantID || site.CompanyID() != "comp-01" || site.ProjectID() != "proj-alpha" || site.SiteID() != "site-rayong-01" {
		t.Errorf("site fields mismatch: %+v", site)
	}
	if !site.IsActive() {
		t.Errorf("expected site active")
	}

	area, err := NewArea(site, "area-boiler-02", "Boiler Room North")
	if err != nil {
		t.Fatalf("unexpected NewArea error: %v", err)
	}
	if area.TenantID() != tenantID || area.CompanyID() != "comp-01" || area.ProjectID() != "proj-alpha" || area.SiteID() != "site-rayong-01" || area.AreaID() != "area-boiler-02" {
		t.Errorf("area fields mismatch: %+v", area)
	}
	if !area.IsActive() {
		t.Errorf("expected area active")
	}
}

func TestHierarchy_BlankIdentifierRejections(t *testing.T) {
	company, _ := NewCompany("tenant-01", "comp-01", "Valid Corp")
	project, _ := NewProject(company, "proj-01", "Valid Project")
	site, _ := NewSite(project, "site-01", "Valid Site")

	// Blank tenant ID
	if _, err := NewCompany("", "comp-01", "Name"); !errors.Is(err, ErrBlankTenantID) {
		t.Errorf("expected ErrBlankTenantID, got %v", err)
	}
	if _, err := NewCompany("   \t", "comp-01", "Name"); !errors.Is(err, ErrBlankTenantID) {
		t.Errorf("expected ErrBlankTenantID for whitespace, got %v", err)
	}

	// Blank company ID & name
	if _, err := NewCompany("tenant-01", "", "Name"); !errors.Is(err, ErrBlankID) {
		t.Errorf("expected ErrBlankID for empty company ID, got %v", err)
	}
	if _, err := NewCompany("tenant-01", "comp-01", "  "); !errors.Is(err, ErrBlankName) {
		t.Errorf("expected ErrBlankName for empty company name, got %v", err)
	}

	// Blank project ID & name
	if _, err := NewProject(company, "", "Name"); !errors.Is(err, ErrBlankID) {
		t.Errorf("expected ErrBlankID for empty project ID, got %v", err)
	}
	if _, err := NewProject(company, "proj-01", "   "); !errors.Is(err, ErrBlankName) {
		t.Errorf("expected ErrBlankName for empty project name, got %v", err)
	}

	// Blank site ID & name
	if _, err := NewSite(project, "", "Name"); !errors.Is(err, ErrBlankID) {
		t.Errorf("expected ErrBlankID for empty site ID, got %v", err)
	}
	if _, err := NewSite(project, "site-01", ""); !errors.Is(err, ErrBlankName) {
		t.Errorf("expected ErrBlankName for empty site name, got %v", err)
	}

	// Blank area ID & name
	if _, err := NewArea(site, "", "Name"); !errors.Is(err, ErrBlankID) {
		t.Errorf("expected ErrBlankID for empty area ID, got %v", err)
	}
	if _, err := NewArea(site, "area-01", "  "); !errors.Is(err, ErrBlankName) {
		t.Errorf("expected ErrBlankName for empty area name, got %v", err)
	}
}

func TestHierarchy_ParentMismatchRejections(t *testing.T) {
	var zeroCompany Company
	if _, err := NewProject(zeroCompany, "proj-01", "Name"); !errors.Is(err, ErrParentMismatch) {
		t.Errorf("expected ErrParentMismatch for zero company parent, got %v", err)
	}

	var zeroProject Project
	if _, err := NewSite(zeroProject, "site-01", "Name"); !errors.Is(err, ErrParentMismatch) {
		t.Errorf("expected ErrParentMismatch for zero project parent, got %v", err)
	}

	var zeroSite Site
	if _, err := NewArea(zeroSite, "area-01", "Name"); !errors.Is(err, ErrParentMismatch) {
		t.Errorf("expected ErrParentMismatch for zero site parent, got %v", err)
	}
}

func TestHierarchy_ArchivedParentRejections(t *testing.T) {
	company, _ := NewCompany("tenant-01", "comp-01", "Acme")
	archivedCompany := company.Archive()

	if _, err := NewProject(archivedCompany, "proj-01", "Name"); !errors.Is(err, ErrParentArchived) {
		t.Fatalf("expected ErrParentArchived when attaching project to archived company, got %v", err)
	}

	project, _ := NewProject(company, "proj-01", "Project")
	archivedProject := project.Archive()

	if _, err := NewSite(archivedProject, "site-01", "Name"); !errors.Is(err, ErrParentArchived) {
		t.Fatalf("expected ErrParentArchived when attaching site to archived project, got %v", err)
	}

	site, _ := NewSite(project, "site-01", "Site")
	archivedSite := site.Archive()

	if _, err := NewArea(archivedSite, "area-01", "Name"); !errors.Is(err, ErrParentArchived) {
		t.Fatalf("expected ErrParentArchived when attaching area to archived site, got %v", err)
	}
}

func TestHierarchy_LifecycleStateTransitions(t *testing.T) {
	company, _ := NewCompany("tenant-01", "comp-01", "Acme")
	if !company.IsActive() || company.State() != StateActive {
		t.Errorf("expected active initially")
	}

	archived := company.Archive()
	if archived.IsActive() || archived.State() != StateArchived {
		t.Errorf("expected archived state, got %v", archived.State())
	}
	// Original remains unchanged (immutable value semantics)
	if !company.IsActive() {
		t.Errorf("original company should remain active")
	}

	// Area lifecycle
	project, _ := NewProject(company, "proj-01", "Proj")
	site, _ := NewSite(project, "site-01", "Site")
	area, _ := NewArea(site, "area-01", "Area")
	archivedArea := area.Archive()
	if !archivedArea.State().StateArchived() && archivedArea.IsActive() {
		t.Errorf("expected area to be archived")
	}
}

// StateArchived helper for readability
func (s LifecycleState) StateArchived() bool {
	return s == StateArchived
}

func TestHierarchy_NoPermissionInheritanceBeyondTenantContext(t *testing.T) {
	// 1. Construct valid hierarchy for Tenant A
	tenantA := "tenant-synth-alpha"
	companyA, _ := NewCompany(tenantA, "comp-a", "Tenant A Corp")
	projectA, _ := NewProject(companyA, "proj-a", "Tenant A Project")
	siteA, _ := NewSite(projectA, "site-a", "Tenant A Site")
	areaA, _ := NewArea(siteA, "area-a", "Tenant A Area")

	// 2. Derive trusted TenantContext for Tenant A
	claimsA := &TrustedClaims{
		Subject:         "user-a",
		TenantID:        tenantA,
		IsAuthenticated: true,
	}
	ctxA, err := DeriveTenantContext(claimsA, nil)
	if err != nil {
		t.Fatalf("unexpected DeriveTenantContext error: %v", err)
	}

	// 3. Verify Tenant A context successfully validates scope for all Tenant A hierarchy entities
	if err := companyA.ValidateScope(ctxA); err != nil {
		t.Errorf("expected Company A in scope for Tenant A, got %v", err)
	}
	if err := projectA.ValidateScope(ctxA); err != nil {
		t.Errorf("expected Project A in scope for Tenant A, got %v", err)
	}
	if err := siteA.ValidateScope(ctxA); err != nil {
		t.Errorf("expected Site A in scope for Tenant A, got %v", err)
	}
	if err := areaA.ValidateScope(ctxA); err != nil {
		t.Errorf("expected Area A in scope for Tenant A, got %v", err)
	}

	// 4. Derive trusted TenantContext for Tenant B (unrelated tenant)
	tenantB := "tenant-synth-bravo"
	claimsB := &TrustedClaims{
		Subject:         "user-b",
		TenantID:        tenantB,
		IsAuthenticated: true,
	}
	ctxB, err := DeriveTenantContext(claimsB, nil)
	if err != nil {
		t.Fatalf("unexpected DeriveTenantContext error: %v", err)
	}

	// 5. Assert that Tenant B context is strictly denied access to Tenant A hierarchy entities
	// Proving hierarchy knowledge/membership grants ZERO permission inheritance or cross-tenant scope bypass
	if err := companyA.ValidateScope(ctxB); !errors.Is(err, ErrTenantMismatch) {
		t.Errorf("expected ErrTenantMismatch for Company A with Tenant B context, got %v", err)
	}
	if err := projectA.ValidateScope(ctxB); !errors.Is(err, ErrTenantMismatch) {
		t.Errorf("expected ErrTenantMismatch for Project A with Tenant B context, got %v", err)
	}
	if err := siteA.ValidateScope(ctxB); !errors.Is(err, ErrTenantMismatch) {
		t.Errorf("expected ErrTenantMismatch for Site A with Tenant B context, got %v", err)
	}
	if err := areaA.ValidateScope(ctxB); !errors.Is(err, ErrTenantMismatch) {
		t.Errorf("expected ErrTenantMismatch for Area A with Tenant B context, got %v", err)
	}
}
