package localidentity_test

import (
	"errors"
	"testing"

	localidentity "github.com/oshethai/oshe-platform/modules/identity-authorization"
)

// setupVisibilityFixture constructs a multi-project directory environment for testing.
func setupVisibilityFixture(t *testing.T) (*localidentity.DirectoryVisibilityService, *localidentity.DirectoryRegistry) {
	t.Helper()
	reg := localidentity.NewDirectoryRegistry()
	matrix := localidentity.NewProvisionalAuthorizationMatrix()

	profiles := []struct {
		id, sub, ten, cmp, prj, ste, name, title, dept string
		areas                                          []string
	}{
		// Company cmp_heavy, Project prj_plant
		{"prof_plant_lead", "usr_synth_lead", "ten_alpha", "cmp_heavy", "prj_plant", "ste_rayong", "Somchai Lead", "Plant Safety Lead", "EHS", []string{"ara_1"}},
		{"prof_plant_insp", "usr_synth_insp", "ten_alpha", "cmp_heavy", "prj_plant", "ste_rayong", "Prasert Insp", "Safety Inspector", "Operations", []string{"ara_1", "ara_2"}},
		{"prof_plant_contractor", "usr_synth_ext1", "ten_alpha", "cmp_heavy", "prj_plant", "ste_rayong", "Kitti Contractor", "Welding Specialist", "Contractor Crew", []string{"ara_2"}},

		// Company cmp_heavy, Project prj_warehouse
		{"prof_wh_pm", "usr_synth_wh_pm", "ten_alpha", "cmp_heavy", "prj_warehouse", "ste_bangkok", "Apinya PM", "Project Manager", "Logistics", nil},
		{"prof_wh_insp", "usr_synth_wh_insp", "ten_alpha", "cmp_heavy", "prj_warehouse", "ste_bangkok", "Wichai Inspector", "Civil Inspector", "Quality", nil},

		// Multi-project subject: usr_synth_lead also has a role on prj_warehouse
		{"prof_wh_lead_consult", "usr_synth_lead", "ten_alpha", "cmp_heavy", "prj_warehouse", "ste_bangkok", "Somchai Consultant", "Senior Safety Advisor", "EHS", nil},

		// Company-wide profile (no project bound)
		{"prof_corp_dir", "usr_synth_corp", "ten_alpha", "cmp_heavy", "", "", "Anan Director", "VP Safety", "Executive", nil},

		// Tenant beta profile (cross-tenant isolation fixture)
		{"prof_beta_user", "usr_synth_beta", "ten_beta", "cmp_other", "prj_other", "ste_other", "Somsak Beta", "Site Inspector", "Safety", nil},
	}

	for _, p := range profiles {
		prof, err := localidentity.NewDirectoryProfile(p.id, p.sub, p.ten, p.cmp, p.prj, p.ste, p.name, p.title, p.dept, p.areas)
		if err != nil {
			t.Fatalf("failed to create fixture profile %s: %v", p.id, err)
		}
		if err := reg.RegisterProfile(prof); err != nil {
			t.Fatalf("failed to register fixture profile %s: %v", p.id, err)
		}
	}

	svc := localidentity.NewDirectoryVisibilityService(reg, matrix)
	return svc, reg
}

func TestDirectoryVisibility_ScopePartitioning(t *testing.T) {
	svc, _ := setupVisibilityFixture(t)

	// Caller 1: Inspector on prj_plant
	plantViewer := localidentity.ViewerContext{
		Identity: localidentity.SubjectIdentity{Subject: "usr_synth_insp", TenantID: "ten_alpha", IsAuthenticated: true},
		Role:     localidentity.RoleInspector,
		Scope:    localidentity.ScopeGrant{TenantID: "ten_alpha", CompanyID: "cmp_heavy", ProjectID: "prj_plant"},
	}

	t.Run("Default search partitions strictly to caller project", func(t *testing.T) {
		results, err := svc.SearchDirectory(plantViewer, localidentity.DirectorySearchFilter{})
		if err != nil {
			t.Fatalf("unexpected search error: %v", err)
		}

		if len(results) == 0 {
			t.Fatalf("expected results for plant project, got 0")
		}

		for _, p := range results {
			if p.ProjectID != "prj_plant" {
				t.Errorf("expected project prj_plant, got profile %s with project %s", p.ProfileID, p.ProjectID)
			}
			if p.TenantID != "ten_alpha" {
				t.Errorf("expected tenant ten_alpha, got %s", p.TenantID)
			}
		}
	})

	t.Run("Anti-enumeration: cross-project search returns empty results (NEG-V030-04)", func(t *testing.T) {
		// Plant viewer attempts to search sibling project prj_warehouse
		results, err := svc.SearchDirectory(plantViewer, localidentity.DirectorySearchFilter{
			ProjectID: "prj_warehouse",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Must return empty slice without leaking existence of prj_warehouse
		if len(results) != 0 {
			t.Errorf("anti-enumeration violation: plant worker received %d results for prj_warehouse", len(results))
		}
	})

	t.Run("Cross-project direct profile lookup fails with ErrProfileNotFound", func(t *testing.T) {
		// Plant viewer attempts to fetch profile on prj_warehouse by ID
		_, err := svc.GetVisibleProfile(plantViewer, "prof_wh_pm")
		if err == nil {
			t.Fatalf("expected ErrProfileNotFound for cross-project profile lookup, got nil")
		}
		if !errors.Is(err, localidentity.ErrProfileNotFound) {
			t.Errorf("expected ErrProfileNotFound, got: %v", err)
		}
	})

	t.Run("Multi-project subject visibility isolation", func(t *testing.T) {
		// usr_synth_lead exists on both prj_plant and prj_warehouse
		// plantViewer looking up usr_synth_lead must ONLY see their prj_plant profile
		results, err := svc.ListVisibleProfilesBySubject(plantViewer, "usr_synth_lead")
		if err != nil {
			t.Fatalf("unexpected list error: %v", err)
		}

		if len(results) != 1 {
			t.Fatalf("expected exactly 1 visible profile for subject, got %d", len(results))
		}
		if results[0].ProfileID != "prof_plant_lead" {
			t.Errorf("expected profile prof_plant_lead, got %s", results[0].ProfileID)
		}
		if results[0].ProjectID != "prj_plant" {
			t.Errorf("expected project prj_plant, got %s", results[0].ProjectID)
		}
	})
}

func TestDirectoryVisibility_ContractorBoundaries(t *testing.T) {
	svc, _ := setupVisibilityFixture(t)

	contractorViewer := localidentity.ViewerContext{
		Identity: localidentity.SubjectIdentity{Subject: "usr_synth_ext1", TenantID: "ten_alpha", IsAuthenticated: true},
		Role:     localidentity.RoleContractor,
		Scope:    localidentity.ScopeGrant{TenantID: "ten_alpha", CompanyID: "cmp_heavy", ProjectID: "prj_plant", SiteID: "ste_rayong"},
	}

	t.Run("Contractor search restricted to assigned project", func(t *testing.T) {
		results, err := svc.SearchDirectory(contractorViewer, localidentity.DirectorySearchFilter{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		for _, p := range results {
			if p.ProjectID != "prj_plant" {
				t.Errorf("contractor received profile from unassigned project: %s", p.ProjectID)
			}
		}
	})

	t.Run("Contractor cannot discover corporate directory", func(t *testing.T) {
		// Contractor attempts to fetch company-wide executive profile
		_, err := svc.GetVisibleProfile(contractorViewer, "prof_corp_dir")
		if err == nil {
			t.Fatalf("expected ErrProfileNotFound for corporate profile, got nil")
		}
		if !errors.Is(err, localidentity.ErrProfileNotFound) {
			t.Errorf("expected ErrProfileNotFound, got: %v", err)
		}
	})

	t.Run("Contractor cross-project query returns empty list", func(t *testing.T) {
		results, err := svc.SearchDirectory(contractorViewer, localidentity.DirectorySearchFilter{
			ProjectID: "prj_warehouse",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 0 {
			t.Errorf("expected 0 results for cross-project contractor query, got %d", len(results))
		}
	})
}

func TestDirectoryVisibility_TenantIsolation(t *testing.T) {
	svc, _ := setupVisibilityFixture(t)

	// Caller in tenant alpha attempts to access tenant beta profile
	alphaViewer := localidentity.ViewerContext{
		Identity: localidentity.SubjectIdentity{Subject: "usr_synth_insp", TenantID: "ten_alpha", IsAuthenticated: true},
		Role:     localidentity.RoleTenantAdmin,
		Scope:    localidentity.ScopeGrant{TenantID: "ten_alpha"},
	}

	t.Run("Cross-tenant profile lookup returns ErrProfileNotFound", func(t *testing.T) {
		_, err := svc.GetVisibleProfile(alphaViewer, "prof_beta_user")
		if err == nil {
			t.Fatalf("expected ErrProfileNotFound for cross-tenant profile, got nil")
		}
		if !errors.Is(err, localidentity.ErrProfileNotFound) {
			t.Errorf("expected ErrProfileNotFound, got: %v", err)
		}
	})

	t.Run("Mismatched scope tenant fails validation", func(t *testing.T) {
		spoofedViewer := alphaViewer
		spoofedViewer.Scope.TenantID = "ten_beta" // Spoofed scope tenant

		_, err := svc.SearchDirectory(spoofedViewer, localidentity.DirectorySearchFilter{})
		if err == nil {
			t.Fatalf("expected ErrSubjectTenantMismatch, got nil")
		}
		if !errors.Is(err, localidentity.ErrSubjectTenantMismatch) {
			t.Errorf("expected ErrSubjectTenantMismatch, got: %v", err)
		}
	})
}

func TestDirectoryVisibility_DataMinimization(t *testing.T) {
	svc, _ := setupVisibilityFixture(t)

	viewer := localidentity.ViewerContext{
		Identity: localidentity.SubjectIdentity{Subject: "usr_synth_insp", TenantID: "ten_alpha", IsAuthenticated: true},
		Role:     localidentity.RoleInspector,
		Scope:    localidentity.ScopeGrant{TenantID: "ten_alpha", CompanyID: "cmp_heavy", ProjectID: "prj_plant"},
	}

	results, err := svc.SearchDirectory(viewer, localidentity.DirectorySearchFilter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, p := range results {
		// Verify standard data minimization assertion
		if err := localidentity.AssertDataMinimization(p); err != nil {
			t.Errorf("data minimization assertion failed on profile %s: %v", p.ProfileID, err)
		}
	}

	// Test negative control on AssertDataMinimization
	t.Run("AssertDataMinimization rejects sensitive credentials or PII", func(t *testing.T) {
		leakCases := []struct {
			name    string
			profile localidentity.MinimizedDirectoryProfile
		}{
			{
				name: "password_in_title",
				profile: localidentity.MinimizedDirectoryProfile{
					ProfileID:   "prof_leak1",
					DisplayName: "Test Worker",
					JobTitle:    "Admin PasswordReset",
				},
			},
			{
				name: "token_in_name",
				profile: localidentity.MinimizedDirectoryProfile{
					ProfileID:   "prof_leak2",
					DisplayName: "Bearer Token User",
					JobTitle:    "Lead",
				},
			},
			{
				name: "personal_email_in_dept",
				profile: localidentity.MinimizedDirectoryProfile{
					ProfileID:   "prof_leak3",
					DisplayName: "John Doe",
					JobTitle:    "Inspector",
					Department:  "contact: worker@gmail.com",
				},
			},
			{
				name: "phone_number_in_name",
				profile: localidentity.MinimizedDirectoryProfile{
					ProfileID:   "prof_leak4",
					DisplayName: "John +66812345678",
					JobTitle:    "Inspector",
				},
			},
		}

		for _, tc := range leakCases {
			t.Run(tc.name, func(t *testing.T) {
				err := localidentity.AssertDataMinimization(tc.profile)
				if err == nil {
					t.Fatalf("expected data minimization violation for %s, got nil", tc.name)
				}
				if !errors.Is(err, localidentity.ErrDataMinimizationViolation) {
					t.Errorf("expected ErrDataMinimizationViolation, got: %v", err)
				}
			})
		}
	})
}

func TestDirectoryVisibility_AuthenticationAndPermission(t *testing.T) {
	svc, _ := setupVisibilityFixture(t)

	t.Run("Unauthenticated caller rejected", func(t *testing.T) {
		unauthViewer := localidentity.ViewerContext{
			Identity: localidentity.SubjectIdentity{Subject: "usr_anon", TenantID: "ten_alpha", IsAuthenticated: false},
			Role:     localidentity.RoleInspector,
			Scope:    localidentity.ScopeGrant{TenantID: "ten_alpha"},
		}

		_, err := svc.SearchDirectory(unauthViewer, localidentity.DirectorySearchFilter{})
		if err == nil {
			t.Fatalf("expected ErrUnauthenticatedCaller, got nil")
		}
		if !errors.Is(err, localidentity.ErrUnauthenticatedCaller) {
			t.Errorf("expected ErrUnauthenticatedCaller, got: %v", err)
		}
	})

	t.Run("Missing role permission rejected", func(t *testing.T) {
		unregisteredRoleViewer := localidentity.ViewerContext{
			Identity: localidentity.SubjectIdentity{Subject: "usr_rogue", TenantID: "ten_alpha", IsAuthenticated: true},
			Role:     localidentity.Role("UNKNOWN_ROLE"),
			Scope:    localidentity.ScopeGrant{TenantID: "ten_alpha"},
		}

		_, err := svc.SearchDirectory(unregisteredRoleViewer, localidentity.DirectorySearchFilter{})
		if err == nil {
			t.Fatalf("expected ErrDirectoryReadPermissionDenied, got nil")
		}
		if !errors.Is(err, localidentity.ErrDirectoryReadPermissionDenied) {
			t.Errorf("expected ErrDirectoryReadPermissionDenied, got: %v", err)
		}
	})
}

func TestDirectoryVisibility_InactiveProfiles(t *testing.T) {
	svc, reg := setupVisibilityFixture(t)

	// Deactivate prof_plant_insp
	deactivated, err := reg.DeactivateProfile("ten_alpha", "prof_plant_insp")
	if err != nil {
		t.Fatalf("failed to deactivate profile: %v", err)
	}
	if deactivated.IsActive() {
		t.Fatalf("expected profile to be inactive")
	}

	// 1. Regular Inspector on plant cannot see inactive profile
	inspectorViewer := localidentity.ViewerContext{
		Identity: localidentity.SubjectIdentity{Subject: "usr_synth_lead", TenantID: "ten_alpha", IsAuthenticated: true},
		Role:     localidentity.RoleInspector,
		Scope:    localidentity.ScopeGrant{TenantID: "ten_alpha", CompanyID: "cmp_heavy", ProjectID: "prj_plant"},
	}

	results, err := svc.SearchDirectory(inspectorViewer, localidentity.DirectorySearchFilter{
		IncludeInactive: true, // Requested, but role is Inspector
	})
	if err != nil {
		t.Fatalf("unexpected search error: %v", err)
	}

	for _, p := range results {
		if p.ProfileID == "prof_plant_insp" {
			t.Errorf("inactive profile leaked to standard inspector search")
		}
	}

	// Direct lookup by standard inspector returns ErrProfileNotFound
	_, err = svc.GetVisibleProfile(inspectorViewer, "prof_plant_insp")
	if !errors.Is(err, localidentity.ErrProfileNotFound) {
		t.Errorf("expected ErrProfileNotFound for inactive profile lookup, got: %v", err)
	}

	// 2. TenantAdmin CAN see inactive profile when requested
	adminViewer := localidentity.ViewerContext{
		Identity: localidentity.SubjectIdentity{Subject: "usr_admin", TenantID: "ten_alpha", IsAuthenticated: true},
		Role:     localidentity.RoleTenantAdmin,
		Scope:    localidentity.ScopeGrant{TenantID: "ten_alpha"},
	}

	adminResults, err := svc.SearchDirectory(adminViewer, localidentity.DirectorySearchFilter{
		ProjectID:       "prj_plant",
		IncludeInactive: true,
	})
	if err != nil {
		t.Fatalf("unexpected admin search error: %v", err)
	}

	var foundInactive bool
	for _, p := range adminResults {
		if p.ProfileID == "prof_plant_insp" {
			foundInactive = true
			if p.Status != localidentity.ProfileStatusInactive {
				t.Errorf("expected status INACTIVE, got %s", p.Status)
			}
		}
	}
	if !foundInactive {
		t.Errorf("expected TenantAdmin to see inactive profile with IncludeInactive=true")
	}

	// Direct lookup by TenantAdmin succeeds
	p, err := svc.GetVisibleProfile(adminViewer, "prof_plant_insp")
	if err != nil {
		t.Fatalf("expected TenantAdmin to retrieve inactive profile, got: %v", err)
	}
	if p.Status != localidentity.ProfileStatusInactive {
		t.Errorf("expected status INACTIVE, got %s", p.Status)
	}
}

func TestDirectoryVisibility_PaginationAndAntiHarvesting(t *testing.T) {
	svc, _ := setupVisibilityFixture(t)

	adminViewer := localidentity.ViewerContext{
		Identity: localidentity.SubjectIdentity{Subject: "usr_admin", TenantID: "ten_alpha", IsAuthenticated: true},
		Role:     localidentity.RoleTenantAdmin,
		Scope:    localidentity.ScopeGrant{TenantID: "ten_alpha"},
	}

	// Request with limit 1
	p1, err := svc.SearchDirectory(adminViewer, localidentity.DirectorySearchFilter{
		Limit:  1,
		Offset: 0,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p1) != 1 {
		t.Errorf("expected 1 result with limit=1, got %d", len(p1))
	}

	// Request with offset 1, limit 1
	p2, err := svc.SearchDirectory(adminViewer, localidentity.DirectorySearchFilter{
		Limit:  1,
		Offset: 1,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p2) != 1 {
		t.Errorf("expected 1 result with offset=1, got %d", len(p2))
	}

	if p1[0].ProfileID == p2[0].ProfileID {
		t.Errorf("expected distinct profiles across pages, got identical %s", p1[0].ProfileID)
	}

	// Offset out of range
	pOut, err := svc.SearchDirectory(adminViewer, localidentity.DirectorySearchFilter{
		Offset: 9999,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pOut) != 0 {
		t.Errorf("expected 0 results for out of range offset, got %d", len(pOut))
	}
}

func TestDirectoryVisibility_TextSearch(t *testing.T) {
	svc, _ := setupVisibilityFixture(t)

	plantViewer := localidentity.ViewerContext{
		Identity: localidentity.SubjectIdentity{Subject: "usr_synth_insp", TenantID: "ten_alpha", IsAuthenticated: true},
		Role:     localidentity.RoleInspector,
		Scope:    localidentity.ScopeGrant{TenantID: "ten_alpha", CompanyID: "cmp_heavy", ProjectID: "prj_plant"},
	}

	// Search matching display name "Prasert"
	res, err := svc.SearchDirectory(plantViewer, localidentity.DirectorySearchFilter{
		Query: "prasert",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res) != 1 || res[0].ProfileID != "prof_plant_insp" {
		t.Errorf("expected prof_plant_insp, got %v", res)
	}

	// Search matching job title "welding"
	res, err = svc.SearchDirectory(plantViewer, localidentity.DirectorySearchFilter{
		Query: "welding",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res) != 1 || res[0].ProfileID != "prof_plant_contractor" {
		t.Errorf("expected prof_plant_contractor, got %v", res)
	}

	// Search with non-matching query
	res, err = svc.SearchDirectory(plantViewer, localidentity.DirectorySearchFilter{
		Query: "nonexistent_query_xyz",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res) != 0 {
		t.Errorf("expected 0 results for non-matching query, got %d", len(res))
	}
}
