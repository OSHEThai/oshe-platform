package localidentity

import (
	"errors"
	"testing"
)

func TestDirectoryProfile_CreationAndAccessors(t *testing.T) {
	subject := "usr_synth_somchai_01"
	tenantID := "ten_alpha"
	companyID := "cmp_heavy"
	projectID := "prj_plant_expansion"
	siteID := "ste_rayong"
	name := "Somchai Prasert"
	title := "Site Safety Coordinator"
	dept := "EHS Operations"
	areas := []string{"ara_boiler_1", "ara_turbine_2"}

	profile, err := NewDirectoryProfile("prof_01", subject, tenantID, companyID, projectID, siteID, name, title, dept, areas)
	if err != nil {
		t.Fatalf("unexpected NewDirectoryProfile error: %v", err)
	}

	if profile.ProfileID() != "prof_01" {
		t.Errorf("profileID mismatch: %s", profile.ProfileID())
	}
	if profile.Subject() != subject {
		t.Errorf("subject mismatch: %s", profile.Subject())
	}
	if profile.TenantID() != tenantID {
		t.Errorf("tenantID mismatch: %s", profile.TenantID())
	}
	if profile.CompanyID() != companyID {
		t.Errorf("companyID mismatch: %s", profile.CompanyID())
	}
	if profile.ProjectID() != projectID {
		t.Errorf("projectID mismatch: %s", profile.ProjectID())
	}
	if profile.SiteID() != siteID {
		t.Errorf("siteID mismatch: %s", profile.SiteID())
	}
	if profile.DisplayName() != name {
		t.Errorf("displayName mismatch: %s", profile.DisplayName())
	}
	if profile.JobTitle() != title {
		t.Errorf("jobTitle mismatch: %s", profile.JobTitle())
	}
	if profile.Department() != dept {
		t.Errorf("department mismatch: %s", profile.Department())
	}
	if len(profile.AssignedAreas()) != 2 {
		t.Errorf("expected 2 assigned areas, got %d", len(profile.AssignedAreas()))
	}
	if !profile.IsActive() || profile.Status() != ProfileStatusActive {
		t.Errorf("expected active profile status")
	}
	if profile.IsCompanyOnly() || !profile.IsProjectScoped() {
		t.Errorf("expected project-scoped profile")
	}
}

func TestDirectoryProfile_CompanyOnlyProfile(t *testing.T) {
	subject := "usr_synth_advisor_01"
	profile, err := NewDirectoryProfile("prof_comp_only", subject, "ten_alpha", "cmp_holding", "", "", "Anan Sukjai", "Corporate Safety Director", "Executive EHS", nil)
	if err != nil {
		t.Fatalf("unexpected error creating company-only profile: %v", err)
	}

	if !profile.IsCompanyOnly() {
		t.Errorf("expected IsCompanyOnly to be true")
	}
	if profile.IsProjectScoped() {
		t.Errorf("expected IsProjectScoped to be false for company-wide profile")
	}
	if profile.ProjectID() != "" || profile.SiteID() != "" {
		t.Errorf("projectID and siteID should be empty")
	}
}

func TestDirectoryProfile_ValidationRejections(t *testing.T) {
	// Blank profile ID
	if _, err := NewDirectoryProfile("", "usr_01", "ten_01", "cmp_01", "", "", "Name", "Title", "Dept", nil); !errors.Is(err, ErrBlankProfileID) {
		t.Errorf("expected ErrBlankProfileID, got %v", err)
	}

	// Blank subject
	if _, err := NewDirectoryProfile("prof_01", "", "ten_01", "cmp_01", "", "", "Name", "Title", "Dept", nil); !errors.Is(err, ErrBlankSubject) {
		t.Errorf("expected ErrBlankSubject, got %v", err)
	}

	// Invalid subject format (missing usr_ prefix)
	if _, err := NewDirectoryProfile("prof_01", "ext_vendor_01", "ten_01", "cmp_01", "", "", "Name", "Title", "Dept", nil); !errors.Is(err, ErrInvalidSubjectFormat) {
		t.Errorf("expected ErrInvalidSubjectFormat, got %v", err)
	}

	// Blank tenant ID
	if _, err := NewDirectoryProfile("prof_01", "usr_01", "", "cmp_01", "", "", "Name", "Title", "Dept", nil); !errors.Is(err, ErrBlankTenantID) {
		t.Errorf("expected ErrBlankTenantID, got %v", err)
	}

	// Blank company ID
	if _, err := NewDirectoryProfile("prof_01", "usr_01", "ten_01", "", "", "", "Name", "Title", "Dept", nil); !errors.Is(err, ErrBlankCompanyID) {
		t.Errorf("expected ErrBlankCompanyID, got %v", err)
	}

	// Blank display name
	if _, err := NewDirectoryProfile("prof_01", "usr_01", "ten_01", "cmp_01", "", "", "   ", "Title", "Dept", nil); !errors.Is(err, ErrBlankDisplayName) {
		t.Errorf("expected ErrBlankDisplayName, got %v", err)
	}
}

func TestDirectoryProfile_MultiProfileSingularIdentity(t *testing.T) {
	registry := NewDirectoryRegistry()
	tenantID := "ten_alpha"
	subject := "usr_synth_somchai_01" // Singular trusted subject

	// Profile 1: Company A, Project 1 (Site Safety Officer)
	p1, _ := NewDirectoryProfile("prof_p1", subject, tenantID, "cmp_a", "prj_terminal", "ste_01", "Somchai P.", "Safety Officer", "Site Operations", []string{"ara_gate"})
	// Profile 2: Company A, Project 2 (Lead Auditor)
	p2, _ := NewDirectoryProfile("prof_p2", subject, tenantID, "cmp_a", "prj_refinery", "ste_02", "Somchai Prasert", "Lead Inspector", "Quality & Audit", []string{"ara_cracking"})
	// Profile 3: Company B (Regional Advisor)
	p3, _ := NewDirectoryProfile("prof_p3", subject, tenantID, "cmp_b", "", "", "S. Prasert", "Regional Advisor", "Consultancy", nil)

	if err := registry.RegisterProfile(p1); err != nil {
		t.Fatalf("failed to register p1: %v", err)
	}
	if err := registry.RegisterProfile(p2); err != nil {
		t.Fatalf("failed to register p2: %v", err)
	}
	if err := registry.RegisterProfile(p3); err != nil {
		t.Fatalf("failed to register p3: %v", err)
	}

	// Retrieve all profiles for singular subject
	profiles, err := registry.ListProfilesBySubject(tenantID, subject, false)
	if err != nil {
		t.Fatalf("unexpected ListProfilesBySubject error: %v", err)
	}
	if len(profiles) != 3 {
		t.Fatalf("expected 3 scoped profiles linked to subject, got %d", len(profiles))
	}

	// Verify link integrity: all profiles point to identical singular subject
	for _, p := range profiles {
		if p.Subject() != subject {
			t.Errorf("singular identity violation: profile subject %s != %s", p.Subject(), subject)
		}
	}
}

func TestDirectoryProfile_DeactivationAndInactiveFilter(t *testing.T) {
	registry := NewDirectoryRegistry()
	tenantID := "ten_alpha"
	subject := "usr_synth_01"

	p1, _ := NewDirectoryProfile("prof_act", subject, tenantID, "cmp_01", "prj_01", "", "Active User", "Engineer", "Ops", nil)
	p2, _ := NewDirectoryProfile("prof_deact", subject, tenantID, "cmp_01", "prj_01", "", "Leaving User", "Engineer", "Ops", nil)

	_ = registry.RegisterProfile(p1)
	_ = registry.RegisterProfile(p2)

	// Deactivate p2
	deactProfile, err := registry.DeactivateProfile(tenantID, "prof_deact")
	if err != nil {
		t.Fatalf("unexpected DeactivateProfile error: %v", err)
	}
	if deactProfile.IsActive() || deactProfile.Status() != ProfileStatusInactive {
		t.Errorf("expected profile to be inactive")
	}

	// 1. SearchDirectory without IncludeInactive returns only active profile
	results, err := registry.SearchDirectory(DirectoryQuery{TenantID: tenantID, ProjectID: "prj_01", IncludeInactive: false})
	if err != nil {
		t.Fatalf("SearchDirectory error: %v", err)
	}
	if len(results) != 1 || results[0].ProfileID() != "prof_act" {
		t.Errorf("expected only active profile in active search, got %d results", len(results))
	}

	// 2. SearchDirectory with IncludeInactive returns both
	allResults, err := registry.SearchDirectory(DirectoryQuery{TenantID: tenantID, ProjectID: "prj_01", IncludeInactive: true})
	if err != nil {
		t.Fatalf("SearchDirectory error: %v", err)
	}
	if len(allResults) != 2 {
		t.Errorf("expected 2 profiles when including inactive, got %d", len(allResults))
	}
}

func TestDirectoryProfile_CrossScopeIsolation_EmptyResults(t *testing.T) {
	registry := NewDirectoryRegistry()
	tenantA := "ten_alpha"
	tenantB := "ten_bravo"

	pA1, _ := NewDirectoryProfile("prof_a_prj1", "usr_a1", tenantA, "cmp_a", "prj_one", "", "Alice", "Lead", "Eng", nil)
	pA2, _ := NewDirectoryProfile("prof_a_prj2", "usr_a2", tenantA, "cmp_a", "prj_two", "", "Bob", "Lead", "Eng", nil)
	pB1, _ := NewDirectoryProfile("prof_b_prj1", "usr_b1", tenantB, "cmp_b", "prj_one", "", "Charlie", "Lead", "Eng", nil)

	_ = registry.RegisterProfile(pA1)
	_ = registry.RegisterProfile(pA2)
	_ = registry.RegisterProfile(pB1)

	// 1. Querying prj_one under Tenant A returns only pA1, never pA2 or pB1
	resA1, err := registry.SearchDirectory(DirectoryQuery{TenantID: tenantA, ProjectID: "prj_one"})
	if err != nil {
		t.Fatalf("SearchDirectory error: %v", err)
	}
	if len(resA1) != 1 || resA1[0].ProfileID() != "prof_a_prj1" {
		t.Errorf("expected only prof_a_prj1, got %d results", len(resA1))
	}

	// 2. Querying an unrelated project (prj_nonexistent) returns an empty slice, never errors or leaks
	resEmpty, err := registry.SearchDirectory(DirectoryQuery{TenantID: tenantA, ProjectID: "prj_nonexistent"})
	if err != nil {
		t.Fatalf("expected clean empty result, got error: %v", err)
	}
	if len(resEmpty) != 0 {
		t.Errorf("expected 0 results for unrelated project, got %d", len(resEmpty))
	}

	// 3. Querying by Tenant B returns only Tenant B profiles
	resB, err := registry.SearchDirectory(DirectoryQuery{TenantID: tenantB, ProjectID: "prj_one"})
	if err != nil {
		t.Fatalf("SearchDirectory Tenant B error: %v", err)
	}
	if len(resB) != 1 || resB[0].ProfileID() != "prof_b_prj1" {
		t.Errorf("expected only prof_b_prj1 for Tenant B")
	}
}

func TestDirectoryProfile_DataMinimization_NoAuthFields(t *testing.T) {
	// Assert data minimization invariants: DirectoryProfile struct explicitly lacks:
	// - Passwords / credentials
	// - Session tokens / bearer hashes
	// - Role definitions / permission grants
	// - Private PII (phone numbers, personal emails, national IDs)
	profile, _ := NewDirectoryProfile("prof_min", "usr_01", "ten_01", "cmp_01", "prj_01", "ste_01", "Somchai", "Officer", "EHS", []string{"ara_1"})

	// AssertNoAuthorizationBypass formally confirms zero authorization entitlement
	if err := AssertNoAuthorizationBypass(profile); err != nil {
		t.Errorf("expected AssertNoAuthorizationBypass to pass for clean profile, got %v", err)
	}

	// Sanitize display name verification
	dirtyName := " Somchai Prasert \n\r\t \x00 "
	clean := SanitizeDisplayName(dirtyName)
	if clean != "Somchai Prasert" {
		t.Errorf("expected clean display name %q, got %q", "Somchai Prasert", clean)
	}
}

func TestDirectoryProfile_DuplicateRejection(t *testing.T) {
	registry := NewDirectoryRegistry()
	p1, _ := NewDirectoryProfile("prof_dup", "usr_01", "ten_01", "cmp_01", "prj_01", "", "User One", "Role", "Dept", nil)
	p2, _ := NewDirectoryProfile("prof_dup", "usr_02", "ten_01", "cmp_01", "prj_01", "", "User Two", "Role", "Dept", nil)

	if err := registry.RegisterProfile(p1); err != nil {
		t.Fatalf("first registration failed: %v", err)
	}
	// Duplicate profile ID in same tenant must fail
	if err := registry.RegisterProfile(p2); !errors.Is(err, ErrDuplicateProfileID) {
		t.Errorf("expected ErrDuplicateProfileID on duplicate, got %v", err)
	}
}
