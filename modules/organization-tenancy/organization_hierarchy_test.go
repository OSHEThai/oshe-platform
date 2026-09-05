package orgtenancy

import (
	"errors"
	"testing"
	"time"
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

func TestHierarchy_SixLevelConstructionWithBusinessUnit(t *testing.T) {
	tenantID := "ten_01j9876543210zyxwvutsrqpon"
	companyID := "cmp_01j9876543210zyxwvutsrqpon"
	buID := "bnu_01j9876543210zyxwvutsrqpon"
	projectID := "prj_01j9876543210zyxwvutsrqpon"
	siteID := "ste_01j9876543210zyxwvutsrqpon"
	areaID := "ara_01j9876543210zyxwvutsrqpon"

	company, err := NewCompany(tenantID, companyID, "Acme Holdings")
	if err != nil {
		t.Fatalf("unexpected NewCompany error: %v", err)
	}

	bu, err := NewBusinessUnit(company, buID, "Heavy Industry BU")
	if err != nil {
		t.Fatalf("unexpected NewBusinessUnit error: %v", err)
	}
	if bu.TenantID() != tenantID || bu.CompanyID() != companyID || bu.BusinessUnitID() != buID || bu.Name() != "Heavy Industry BU" {
		t.Errorf("business unit field mismatch: %+v", bu)
	}
	if !bu.IsActive() || bu.State() != StateActive {
		t.Errorf("expected business unit active, got %v", bu.State())
	}

	project, err := NewProjectUnderBusinessUnit(bu, projectID, "Refinery Modernization")
	if err != nil {
		t.Fatalf("unexpected NewProjectUnderBusinessUnit error: %v", err)
	}
	if project.TenantID() != tenantID || project.CompanyID() != companyID || project.BusinessUnitID() != buID || project.ProjectID() != projectID {
		t.Errorf("project field mismatch: %+v", project)
	}
	if !project.IsActive() {
		t.Errorf("expected project active")
	}

	site, err := NewSite(project, siteID, "Map Ta Phut Facility")
	if err != nil {
		t.Fatalf("unexpected NewSite error: %v", err)
	}
	if site.TenantID() != tenantID || site.CompanyID() != companyID || site.BusinessUnitID() != buID || site.ProjectID() != projectID || site.SiteID() != siteID {
		t.Errorf("site field mismatch: %+v", site)
	}
	if !site.IsActive() {
		t.Errorf("expected site active")
	}

	area, err := NewArea(site, areaID, "Cracker Unit 4")
	if err != nil {
		t.Fatalf("unexpected NewArea error: %v", err)
	}
	if area.TenantID() != tenantID || area.CompanyID() != companyID || area.BusinessUnitID() != buID || area.ProjectID() != projectID || area.SiteID() != siteID || area.AreaID() != areaID {
		t.Errorf("area field mismatch: %+v", area)
	}
	if !area.IsActive() {
		t.Errorf("expected area active")
	}

	claims := &TrustedClaims{
		Subject:         "usr_01j9876543210zyxwvutsrqpon",
		TenantID:        tenantID,
		IsAuthenticated: true,
	}
	ctx, err := DeriveTenantContext(claims, nil)
	if err != nil {
		t.Fatalf("unexpected DeriveTenantContext error: %v", err)
	}
	if err := bu.ValidateScope(ctx); err != nil {
		t.Errorf("expected BU in scope, got %v", err)
	}
	if err := project.ValidateScope(ctx); err != nil {
		t.Errorf("expected Project in scope, got %v", err)
	}
	if err := site.ValidateScope(ctx); err != nil {
		t.Errorf("expected Site in scope, got %v", err)
	}
	if err := area.ValidateScope(ctx); err != nil {
		t.Errorf("expected Area in scope, got %v", err)
	}
}

func TestHierarchy_BusinessUnitValidationAndRejections(t *testing.T) {
	company, err := NewCompany("ten-01", "cmp-01", "Acme")
	if err != nil {
		t.Fatalf("unexpected NewCompany error: %v", err)
	}

	// Blank ID rejection
	if _, err := NewBusinessUnit(company, "", "BU Name"); !errors.Is(err, ErrBlankID) {
		t.Errorf("expected ErrBlankID, got %v", err)
	}
	if _, err := NewBusinessUnit(company, "   ", "BU Name"); !errors.Is(err, ErrBlankID) {
		t.Errorf("expected ErrBlankID, got %v", err)
	}

	// Blank name rejection
	if _, err := NewBusinessUnit(company, "bu-01", ""); !errors.Is(err, ErrBlankName) {
		t.Errorf("expected ErrBlankName, got %v", err)
	}
	if _, err := NewBusinessUnit(company, "bu-01", "   "); !errors.Is(err, ErrBlankName) {
		t.Errorf("expected ErrBlankName, got %v", err)
	}

	// Uninitialized company parent rejection
	if _, err := NewBusinessUnit(Company{}, "bu-01", "BU Name"); !errors.Is(err, ErrParentMismatch) {
		t.Errorf("expected ErrParentMismatch, got %v", err)
	}

	// Archived company parent rejection
	archivedComp := company.Archive()
	if _, err := NewBusinessUnit(archivedComp, "bu-01", "BU Name"); !errors.Is(err, ErrParentArchived) {
		t.Errorf("expected ErrParentArchived, got %v", err)
	}

	// Valid BU creation
	bu, err := NewBusinessUnit(company, "bu-01", "BU Active")
	if err != nil {
		t.Fatalf("unexpected NewBusinessUnit error: %v", err)
	}

	// Archive BU and verify project rejection
	archivedBU := bu.Archive()
	if !archivedBU.State().StateArchived() || archivedBU.IsActive() {
		t.Errorf("expected archived BU state")
	}
	if _, err := NewProjectUnderBusinessUnit(archivedBU, "prj-01", "Project"); !errors.Is(err, ErrParentArchived) {
		t.Errorf("expected ErrParentArchived under archived BU, got %v", err)
	}

	// Uninitialized BU parent rejection
	if _, err := NewProjectUnderBusinessUnit(BusinessUnit{}, "prj-01", "Project"); !errors.Is(err, ErrParentMismatch) {
		t.Errorf("expected ErrParentMismatch under empty BU, got %v", err)
	}
}

func TestHierarchy_SponsoredPartyLifecycleAndValidation(t *testing.T) {
	company, _ := NewCompany("ten-01", "cmp-01", "Acme")
	bu, _ := NewBusinessUnit(company, "bu-01", "Ops BU")
	project, _ := NewProjectUnderBusinessUnit(bu, "prj-01", "Plant Expansion")
	site, _ := NewSite(project, "ste-01", "Rayong Site")

	baseTime := time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC)
	validFrom := baseTime
	validTo := baseTime.Add(30 * 24 * time.Hour)

	// 1. Valid construction
	sp, err := NewSponsoredParty(site, "prt_contractor_01", "Apex Inspection Services Ltd", "usr_sponsor_mgr", validFrom, validTo)
	if err != nil {
		t.Fatalf("unexpected NewSponsoredParty error: %v", err)
	}
	if sp.TenantID() != "ten-01" || sp.PartyID() != "prt_contractor_01" || sp.CompanyName() != "Apex Inspection Services Ltd" ||
		sp.SponsorID() != "usr_sponsor_mgr" || sp.ProjectID() != "prj-01" || sp.SiteID() != "ste-01" {
		t.Errorf("sponsored party field mismatch: %+v", sp)
	}
	if !sp.IsActive() || sp.State() != StateActive {
		t.Errorf("expected sponsored party active")
	}

	// 2. Validity window checks
	during := baseTime.Add(5 * 24 * time.Hour)
	before := baseTime.Add(-1 * time.Hour)
	after := validTo.Add(1 * time.Hour)

	if !sp.IsValidAt(during) {
		t.Errorf("expected valid during validity window")
	}
	if sp.IsValidAt(before) {
		t.Errorf("expected invalid before validity window")
	}
	if sp.IsValidAt(after) {
		t.Errorf("expected invalid after validity window")
	}

	// 3. Archive lifecycle (no hard deletion, preserves historical record)
	archivedSP := sp.Archive()
	if archivedSP.IsActive() || archivedSP.State() != StateArchived {
		t.Errorf("expected archived state")
	}
	if archivedSP.IsValidAt(during) {
		t.Errorf("expected archived sponsored party to be invalid even within time window")
	}
	if archivedSP.PartyID() != "prt_contractor_01" || archivedSP.CompanyName() != "Apex Inspection Services Ltd" {
		t.Errorf("expected archived party to retain identification history")
	}

	// 4. Rejections
	// Blank party ID
	if _, err := NewSponsoredParty(site, "", "Apex", "usr_01", validFrom, validTo); !errors.Is(err, ErrBlankID) {
		t.Errorf("expected ErrBlankID, got %v", err)
	}
	// Blank company name
	if _, err := NewSponsoredParty(site, "prt_01", "", "usr_01", validFrom, validTo); !errors.Is(err, ErrBlankName) {
		t.Errorf("expected ErrBlankName, got %v", err)
	}
	// Blank sponsor ID
	if _, err := NewSponsoredParty(site, "prt_01", "Apex", "", validFrom, validTo); !errors.Is(err, ErrBlankSponsorID) {
		t.Errorf("expected ErrBlankSponsorID, got %v", err)
	}
	if _, err := NewSponsoredParty(site, "prt_01", "Apex", "   ", validFrom, validTo); !errors.Is(err, ErrBlankSponsorID) {
		t.Errorf("expected ErrBlankSponsorID, got %v", err)
	}
	// Invalid time window (validTo before or equal to validFrom)
	if _, err := NewSponsoredParty(site, "prt_01", "Apex", "usr_01", validTo, validFrom); !errors.Is(err, ErrInvalidTimeWindow) {
		t.Errorf("expected ErrInvalidTimeWindow, got %v", err)
	}
	if _, err := NewSponsoredParty(site, "prt_01", "Apex", "usr_01", validFrom, validFrom); !errors.Is(err, ErrInvalidTimeWindow) {
		t.Errorf("expected ErrInvalidTimeWindow for equal times, got %v", err)
	}
	// Archived site parent rejection
	archivedSite := site.Archive()
	if _, err := NewSponsoredParty(archivedSite, "prt_01", "Apex", "usr_01", validFrom, validTo); !errors.Is(err, ErrParentArchived) {
		t.Errorf("expected ErrParentArchived under archived site, got %v", err)
	}
	// Uninitialized site parent rejection
	if _, err := NewSponsoredParty(Site{}, "prt_01", "Apex", "usr_01", validFrom, validTo); !errors.Is(err, ErrParentMismatch) {
		t.Errorf("expected ErrParentMismatch under empty site, got %v", err)
	}
}

func TestHierarchy_CrossTenantDenialIncludingBusinessUnitAndSponsoredParty(t *testing.T) {
	tenantA := "tenant-alpha"
	tenantB := "tenant-bravo"

	companyA, _ := NewCompany(tenantA, "cmp-a", "Tenant A Corp")
	buA, _ := NewBusinessUnit(companyA, "bu-a", "Tenant A BU")
	projectA, _ := NewProjectUnderBusinessUnit(buA, "prj-a", "Tenant A Project")
	siteA, _ := NewSite(projectA, "site-a", "Tenant A Site")

	validFrom := time.Now().Add(-1 * time.Hour)
	validTo := time.Now().Add(24 * time.Hour)
	spA, err := NewSponsoredParty(siteA, "prt-contractor-a", "Contractor A", "usr-sponsor-a", validFrom, validTo)
	if err != nil {
		t.Fatalf("unexpected NewSponsoredParty error: %v", err)
	}

	claimsB := &TrustedClaims{
		Subject:         "user-b",
		TenantID:        tenantB,
		IsAuthenticated: true,
	}
	ctxB, err := DeriveTenantContext(claimsB, nil)
	if err != nil {
		t.Fatalf("unexpected DeriveTenantContext error: %v", err)
	}

	if err := buA.ValidateScope(ctxB); !errors.Is(err, ErrTenantMismatch) {
		t.Errorf("expected ErrTenantMismatch for BU A with Tenant B context, got %v", err)
	}
	if err := spA.ValidateScope(ctxB); !errors.Is(err, ErrTenantMismatch) {
		t.Errorf("expected ErrTenantMismatch for SponsoredParty A with Tenant B context, got %v", err)
	}
}

func TestHierarchy_CanonicalPrefixesVerification(t *testing.T) {
	if PrefixBusinessUnit != "bnu" {
		t.Errorf("expected PrefixBusinessUnit to be 'bnu', got %q", PrefixBusinessUnit)
	}
	if PrefixParty != "prt" {
		t.Errorf("expected PrefixParty to be 'prt', got %q", PrefixParty)
	}

	buID, err := GenerateCanonicalID(PrefixBusinessUnit)
	if err != nil {
		t.Fatalf("unexpected GenerateCanonicalID for bnu error: %v", err)
	}
	if err := ValidateCanonicalID(buID, PrefixBusinessUnit); err != nil {
		t.Errorf("expected valid canonical BU ID %q, got err: %v", buID, err)
	}

	partyID, err := GenerateCanonicalID(PrefixParty)
	if err != nil {
		t.Fatalf("unexpected GenerateCanonicalID for prt error: %v", err)
	}
	if err := ValidateCanonicalID(partyID, PrefixParty); err != nil {
		t.Errorf("expected valid canonical party ID %q, got err: %v", partyID, err)
	}
}
