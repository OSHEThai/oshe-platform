package orgtenancy

import (
	"errors"
	"strings"
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

func TestHierarchy_SiteTimeZoneAndLocale(t *testing.T) {
	company, _ := NewCompany("ten-01", "cmp-01", "Acme")
	bu, _ := NewBusinessUnit(company, "bu-01", "Operations")
	project, _ := NewProjectUnderBusinessUnit(bu, "prj-01", "Expansion")

	// 1. Default time zone and locale
	siteDefault, err := NewSite(project, "site-default", "Default Site")
	if err != nil {
		t.Fatalf("NewSite failed: %v", err)
	}
	if siteDefault.TimeZone() != DefaultTimeZone {
		t.Errorf("expected default time zone %s, got %s", DefaultTimeZone, siteDefault.TimeZone())
	}
	if siteDefault.Locale() != DefaultLocale {
		t.Errorf("expected default locale %s, got %s", DefaultLocale, siteDefault.Locale())
	}

	// 2. Explicit valid time zone and locale
	siteExplicit, err := NewSiteWithLocale(project, "site-tokyo", "Tokyo Site", "Asia/Tokyo", "en-US")
	if err != nil {
		t.Fatalf("NewSiteWithLocale failed: %v", err)
	}
	if siteExplicit.TimeZone() != "Asia/Tokyo" {
		t.Errorf("expected Asia/Tokyo, got %s", siteExplicit.TimeZone())
	}
	if siteExplicit.Locale() != "en-US" {
		t.Errorf("expected en-US, got %s", siteExplicit.Locale())
	}

	// 3. Invalid IANA time zone rejected
	_, err = NewSiteWithLocale(project, "site-bad-tz", "Bad TZ", "Mars/Olympus", "th-TH")
	if !errors.Is(err, ErrInvalidTimeZone) {
		t.Errorf("expected ErrInvalidTimeZone for invalid time zone, got: %v", err)
	}

	// 4. Invalid BCP 47 locale rejected
	_, err = NewSiteWithLocale(project, "site-bad-loc", "Bad Locale", "Asia/Bangkok", "invalid_locale_with_special!@#")
	if !errors.Is(err, ErrInvalidLocale) {
		t.Errorf("expected ErrInvalidLocale for invalid locale, got: %v", err)
	}
}

func TestHierarchy_AreaTimeZoneAndLocaleInheritance(t *testing.T) {
	company, _ := NewCompany("ten-01", "cmp-01", "Acme")
	bu, _ := NewBusinessUnit(company, "bu-01", "Operations")
	project, _ := NewProjectUnderBusinessUnit(bu, "prj-01", "Expansion")
	site, _ := NewSiteWithLocale(project, "site-custom", "Custom Site", "Asia/Singapore", "en-US")

	// 1. Inherited from site
	areaInherited, err := NewArea(site, "area-inherited", "Inherited Area")
	if err != nil {
		t.Fatalf("NewArea failed: %v", err)
	}
	if areaInherited.TimeZone() != "Asia/Singapore" {
		t.Errorf("expected inherited time zone Asia/Singapore, got %s", areaInherited.TimeZone())
	}
	if areaInherited.Locale() != "en-US" {
		t.Errorf("expected inherited locale en-US, got %s", areaInherited.Locale())
	}

	// 2. Explicit override on Area
	areaExplicit, err := NewAreaWithLocale(site, "area-explicit", "Explicit Area", "Asia/Bangkok", "th-TH")
	if err != nil {
		t.Fatalf("NewAreaWithLocale failed: %v", err)
	}
	if areaExplicit.TimeZone() != "Asia/Bangkok" {
		t.Errorf("expected Asia/Bangkok, got %s", areaExplicit.TimeZone())
	}
	if areaExplicit.Locale() != "th-TH" {
		t.Errorf("expected th-TH, got %s", areaExplicit.Locale())
	}

	// 3. Invalid time zone on Area rejected
	_, err = NewAreaWithLocale(site, "area-bad-tz", "Bad TZ", "Invalid/Zone", "th-TH")
	if !errors.Is(err, ErrInvalidTimeZone) {
		t.Errorf("expected ErrInvalidTimeZone on area, got: %v", err)
	}

	// 4. Invalid locale on Area rejected
	_, err = NewAreaWithLocale(site, "area-bad-loc", "Bad Loc", "Asia/Bangkok", "bad-loc-@#$")
	if !errors.Is(err, ErrInvalidLocale) {
		t.Errorf("expected ErrInvalidLocale on area, got: %v", err)
	}
}

func TestHierarchy_ProjectSiteAndAreaParentValidation(t *testing.T) {
	company, _ := NewCompany("ten-01", "cmp-01", "Acme")
	projectA, _ := NewProject(company, "prj-a", "Project A")
	siteA, _ := NewSite(projectA, "ste-a", "Site under Project A")
	areaA, _ := NewArea(siteA, "ara-a", "Area under Site A")

	// ValidateParentProject
	if err := siteA.ValidateParentProject("prj-a"); err != nil {
		t.Errorf("expected matching project to validate, got: %v", err)
	}
	if err := siteA.ValidateParentProject("prj-b"); !errors.Is(err, ErrProjectSiteMismatch) {
		t.Errorf("expected ErrProjectSiteMismatch for project B, got: %v", err)
	}

	// ValidateParentSite
	if err := areaA.ValidateParentSite("ste-a"); err != nil {
		t.Errorf("expected matching site to validate, got: %v", err)
	}
	if err := areaA.ValidateParentSite("ste-other"); !errors.Is(err, ErrParentMismatch) {
		t.Errorf("expected ErrParentMismatch for other site, got: %v", err)
	}
}

func TestHierarchy_CanonicalIdentifierEnforcementWithSyntheticCompatibility(t *testing.T) {
	company, _ := NewCompany("ten-01", "cmp-01", "Acme")
	project, _ := NewProject(company, "prj-01", "Project")

	// 1. Valid canonical IDs with prefix
	siteCanonical, err := NewSite(project, "ste_01j9876543210zyxwvutsrqpon", "Canonical Site")
	if err != nil {
		t.Errorf("expected valid canonical site ID to succeed, got: %v", err)
	}
	if siteCanonical.SiteID() != "ste_01j9876543210zyxwvutsrqpon" {
		t.Errorf("site ID mismatch: %s", siteCanonical.SiteID())
	}

	// 2. Malformed canonical ID (missing token or invalid trailing underscore)
	_, err = NewSite(project, "ste_", "Bad Site")
	if !errors.Is(err, ErrMalformedIdentifier) {
		t.Errorf("expected ErrMalformedIdentifier for ste_, got: %v", err)
	}

	// 3. Invalid characters in canonical ID
	_, err = NewSite(project, "ste_BAD@ID!", "Bad Chars Site")
	if !errors.Is(err, ErrInvalidCharacters) {
		t.Errorf("expected ErrInvalidCharacters for ste_BAD@ID!, got: %v", err)
	}

	// 4. Wrong prefix canonical ID
	_, err = NewSite(project, "prj_01j9876543210zyxwvutsrqpon", "Wrong Prefix Site")
	if !errors.Is(err, ErrPrefixMismatch) {
		t.Errorf("expected ErrPrefixMismatch for prj_ prefix on site, got: %v", err)
	}

	// 5. Backward-compatible synthetic slug (no underscore)
	siteSlug, err := NewSite(project, "site-rayong-complex", "Slug Site")
	if err != nil {
		t.Errorf("expected synthetic slug without underscore to succeed, got: %v", err)
	}
	if siteSlug.SiteID() != "site-rayong-complex" {
		t.Errorf("slug site ID mismatch: %s", siteSlug.SiteID())
	}
}

func TestHierarchy_ResolvedScope_NonAuthoritative(t *testing.T) {
	tenantID := "ten-synth-alpha"
	company, err := NewCompany(tenantID, "cmp-01", "Acme")
	if err != nil {
		t.Fatalf("NewCompany failed: %v", err)
	}
	bu, err := NewBusinessUnit(company, "bu-01", "Operations")
	if err != nil {
		t.Fatalf("NewBusinessUnit failed: %v", err)
	}
	project, err := NewProjectUnderBusinessUnit(bu, "prj-01", "Expansion")
	if err != nil {
		t.Fatalf("NewProjectUnderBusinessUnit failed: %v", err)
	}
	site, err := NewSiteWithLocale(project, "ste-01", "Site Rayong", "Asia/Bangkok", "th-TH")
	if err != nil {
		t.Fatalf("NewSiteWithLocale failed: %v", err)
	}
	area, err := NewArea(site, "ara-01", "Cracker 1")
	if err != nil {
		t.Fatalf("NewArea failed: %v", err)
	}

	// 1. Scope resolution produces descriptive canonical path
	scopeArea := area.ResolveScope()
	expectedPath := "ten-synth-alpha/cmp-01/bu-01/prj-01/ste-01/ara-01"
	if scopeArea.CanonicalPath != expectedPath {
		t.Errorf("expected canonical path %q, got %q", expectedPath, scopeArea.CanonicalPath)
	}
	if scopeArea.TimeZone != "Asia/Bangkok" || scopeArea.Locale != "th-TH" {
		t.Errorf("scope time zone or locale mismatch: tz=%s, loc=%s", scopeArea.TimeZone, scopeArea.Locale)
	}
	if !strings.Contains(scopeArea.NonAuthorityNotice, "DERIVED_OUTPUT_NON_AUTHORITY") {
		t.Errorf("expected non-authority notice, got %q", scopeArea.NonAuthorityNotice)
	}

	// 2. ValidateScope only asserts tenant equality
	claimsSameTenant := &TrustedClaims{
		Subject:         "usr-01",
		TenantID:        tenantID,
		IsAuthenticated: true,
	}
	ctxSame, err := DeriveTenantContext(claimsSameTenant, nil)
	if err != nil {
		t.Fatalf("DeriveTenantContext failed: %v", err)
	}
	if err := scopeArea.ValidateScope(ctxSame); err != nil {
		t.Errorf("expected same tenant scope to validate, got: %v", err)
	}

	claimsOtherTenant := &TrustedClaims{
		Subject:         "usr-02",
		TenantID:        "ten-synth-other",
		IsAuthenticated: true,
	}
	ctxOther, err := DeriveTenantContext(claimsOtherTenant, nil)
	if err != nil {
		t.Fatalf("DeriveTenantContext other failed: %v", err)
	}
	if err := scopeArea.ValidateScope(ctxOther); !errors.Is(err, ErrTenantMismatch) {
		t.Errorf("expected ErrTenantMismatch on cross-tenant scope validation, got: %v", err)
	}
}
