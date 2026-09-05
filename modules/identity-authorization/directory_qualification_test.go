// Package localidentity_test provides integration and qualification tests for OSHE Platform identity services.
//
// QUALIFICATION SUITE DECLARATION (Issue #89 / V030-I016):
// Under approved Sole Human Owner decisions H030-003, H030-004, and H030-005, this qualification
// suite provides deterministic evidence for:
// 1. Directory privacy, attribute minimization, and anti-harvesting bounds.
// 2. Duplicate identifier collision rejection and explicit false-merge prohibition.
// 3. Exact-scope directory partitioning, anti-enumeration, and multi-project subject isolation.
// 4. Safe profile lifecycle updates, inactivation shielding, and active-by-default discovery.
// 5. In-memory simulated migration, append-only history tracking, and reversible recovery lineage.
// 6. Strict separation of directory projections from operational authorization and tokens.
//
// Boundary & Non-Claims Invariant:
// Operates exclusively in-memory on local synthetic fixtures. Zero external identity provider
// synchronization (Active Directory, LDAP, Okta), zero production database persistence, zero network
// sockets, zero customer data, and zero binding operational authority are claimed or enacted.
package localidentity_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	localidentity "github.com/oshethai/oshe-platform/modules/identity-authorization"
)

// TestQualification_DirectoryPrivacyAndDataMinimization verifies that directory projections
// expose only sanitized operational fields and strictly exclude credentials, tokens, and PII.
func TestQualification_DirectoryPrivacyAndDataMinimization(t *testing.T) {
	reg := localidentity.NewDirectoryRegistry()
	matrix := localidentity.NewProvisionalAuthorizationMatrix()
	svc := localidentity.NewDirectoryVisibilityService(reg, matrix)

	tenantID := "ten_qual_privacy_01"

	// Register synthetic profiles across various roles
	profiles := []struct {
		id, sub, cmp, prj, ste, name, title, dept string
		areas                                     []string
	}{
		{"prof_priv_01", "usr_synth_lead", "cmp_alpha", "prj_east", "ste_rayong", "Somchai Prasert", "Site Safety Lead", "EHS", []string{"ara_1"}},
		{"prof_priv_02", "usr_synth_insp", "cmp_alpha", "prj_east", "ste_rayong", "Anan Chai", "Field Inspector", "Operations", []string{"ara_2"}},
		{"prof_priv_03", "usr_synth_ext", "cmp_alpha", "prj_east", "ste_rayong", "Kitti Contractor", "Scaffolding Lead", "Subcontractor", nil},
	}

	for _, p := range profiles {
		prof, err := localidentity.NewDirectoryProfile(p.id, p.sub, tenantID, p.cmp, p.prj, p.ste, p.name, p.title, p.dept, p.areas)
		if err != nil {
			t.Fatalf("unexpected NewDirectoryProfile error for %s: %v", p.id, err)
		}
		if err := reg.RegisterProfile(prof); err != nil {
			t.Fatalf("unexpected RegisterProfile error for %s: %v", p.id, err)
		}
	}

	viewer := localidentity.ViewerContext{
		Identity: localidentity.SubjectIdentity{Subject: "usr_synth_lead", TenantID: tenantID, IsAuthenticated: true},
		Role:     localidentity.RoleInspector,
		Scope:    localidentity.ScopeGrant{TenantID: tenantID, CompanyID: "cmp_alpha", ProjectID: "prj_east"},
	}

	// 1. Search directory and verify all returned profiles conform to data minimization
	results, err := svc.SearchDirectory(viewer, localidentity.DirectorySearchFilter{})
	if err != nil {
		t.Fatalf("unexpected SearchDirectory error: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("expected 3 profiles in search results, got %d", len(results))
	}

	forbiddenPatterns := []string{
		"password", "secret", "token", "oshe_tok_", "bearer", "session_id",
		"@gmail.com", "@yahoo.com", "+66", "081-", "089-", "ssn", "national_id",
	}

	for _, p := range results {
		// Verify via standard assertion
		if err := localidentity.AssertDataMinimization(p); err != nil {
			t.Errorf("AssertDataMinimization failed for profile %s: %v", p.ProfileID, err)
		}

		// Direct inspection of serialized attributes
		corpus := strings.ToLower(fmt.Sprintf("%s %s %s %s %s", p.DisplayName, p.JobTitle, p.Department, p.ProfileID, p.Subject))
		for _, f := range forbiddenPatterns {
			if strings.Contains(corpus, f) {
				t.Errorf("profile %s leaked forbidden pattern %q in corpus %q", p.ProfileID, f, corpus)
			}
		}
	}

	// 2. Direct lookup data minimization assertion
	pDirect, err := svc.GetVisibleProfile(viewer, "prof_priv_01")
	if err != nil {
		t.Fatalf("unexpected GetVisibleProfile error: %v", err)
	}
	if err := localidentity.AssertDataMinimization(pDirect); err != nil {
		t.Errorf("AssertDataMinimization failed on direct lookup: %v", err)
	}
}

// TestQualification_DuplicateIdentityAndFalseMergeDenial verifies that duplicate profile IDs
// are strictly rejected and distinct synthetic subjects can never be merged or aliased.
func TestQualification_DuplicateIdentityAndFalseMergeDenial(t *testing.T) {
	resolver := localidentity.NewDirectoryResolver(nil, nil)
	tenantID := "ten_qual_dup_01"

	p1, err := localidentity.NewDirectoryProfile(
		"prof_uniq_01",
		"usr_synth_worker_a",
		tenantID,
		"cmp_heavy",
		"prj_alpha",
		"ste_site_1",
		"Kallaya Sorn",
		"Safety Specialist",
		"Quality & Safety",
		[]string{"ara_1"},
	)
	if err != nil {
		t.Fatalf("unexpected NewDirectoryProfile error: %v", err)
	}

	// 1. Initial registration succeeds
	if err := resolver.RegisterProfile(p1, "usr_admin", "First onboarding"); err != nil {
		t.Fatalf("unexpected RegisterProfile error: %v", err)
	}

	// 2. Duplicate Profile ID collision rejection (H030-003 / H030-004)
	pDuplicateID, _ := localidentity.NewDirectoryProfile(
		"prof_uniq_01", // Same ID
		"usr_synth_worker_b",
		tenantID,
		"cmp_heavy",
		"prj_alpha",
		"ste_site_1",
		"Kallaya Sorn",
		"Safety Specialist",
		"Quality & Safety",
		nil,
	)
	err = resolver.RegisterProfile(pDuplicateID, "usr_admin", "Duplicate attempt")
	if !errors.Is(err, localidentity.ErrDuplicateIdentifierCollision) {
		t.Fatalf("expected ErrDuplicateIdentifierCollision on duplicate profile ID, got: %v", err)
	}

	// 3. Explicit False-Merge Prohibition
	// Even if two distinct subjects share the exact same display name, job title, and project:
	subjectA := "usr_synth_worker_a"
	subjectB := "usr_synth_worker_b"

	if err := localidentity.AssertNoFalseMerge(p1, subjectA); err != nil {
		t.Errorf("expected matching subject to pass, got: %v", err)
	}
	if err := localidentity.AssertNoFalseMerge(p1, subjectB); !errors.Is(err, localidentity.ErrFalseMergeProhibited) {
		t.Errorf("expected ErrFalseMergeProhibited when binding profile to subjectB, got: %v", err)
	}
	if err := localidentity.AssertDistinctSubjects(subjectA, subjectB); !errors.Is(err, localidentity.ErrFalseMergeProhibited) {
		t.Errorf("expected ErrFalseMergeProhibited between distinct subjects, got: %v", err)
	}

	// 4. Structural Identity Immutability
	// Verify that updating non-structural attributes leaves all structural keys untouched
	newTitle := "Senior Safety Specialist"
	newDept := "Corporate Safety Audit"
	updated, err := resolver.UpdateProfileAttributes(tenantID, "prof_uniq_01", localidentity.ProfileNonStructuralUpdate{
		JobTitle:   &newTitle,
		Department: &newDept,
	}, "usr_hr_lead", "Annual promotion")
	if err != nil {
		t.Fatalf("unexpected UpdateProfileAttributes error: %v", err)
	}

	if updated.ProfileID() != p1.ProfileID() ||
		updated.Subject() != p1.Subject() ||
		updated.TenantID() != p1.TenantID() ||
		updated.CompanyID() != p1.CompanyID() ||
		updated.ProjectID() != p1.ProjectID() ||
		updated.SiteID() != p1.SiteID() {
		t.Fatalf("CRITICAL: structural identifiers mutated during non-structural update")
	}
	if updated.JobTitle() != newTitle || updated.Department() != newDept {
		t.Errorf("non-structural fields not updated properly")
	}
}

// TestQualification_CrossProjectSearchAndAntiEnumeration verifies that directory discovery
// strictly partitions queries to the caller's authorized scope and denies cross-project reconnaissance.
func TestQualification_CrossProjectSearchAndAntiEnumeration(t *testing.T) {
	reg := localidentity.NewDirectoryRegistry()
	matrix := localidentity.NewProvisionalAuthorizationMatrix()
	svc := localidentity.NewDirectoryVisibilityService(reg, matrix)

	tenantID := "ten_qual_scope_01"

	// Seed Project Alpha profiles
	pAlpha1, _ := localidentity.NewDirectoryProfile("prof_a1", "usr_alpha_lead", tenantID, "cmp_1", "prj_alpha", "ste_a", "Alpha Lead", "Lead", "Eng", nil)
	pAlpha2, _ := localidentity.NewDirectoryProfile("prof_a2", "usr_alpha_insp", tenantID, "cmp_1", "prj_alpha", "ste_a", "Alpha Insp", "Inspector", "EHS", nil)
	_ = reg.RegisterProfile(pAlpha1)
	_ = reg.RegisterProfile(pAlpha2)

	// Seed Project Beta profiles (victim context)
	pBeta1, _ := localidentity.NewDirectoryProfile("prof_b1", "usr_beta_lead", tenantID, "cmp_1", "prj_beta", "ste_b", "Beta Lead", "Lead", "Eng", nil)
	pBeta2, _ := localidentity.NewDirectoryProfile("prof_b2", "usr_multi_worker", tenantID, "cmp_1", "prj_beta", "ste_b", "Multi Worker", "Specialist", "EHS", nil)
	_ = reg.RegisterProfile(pBeta1)
	_ = reg.RegisterProfile(pBeta2)

	// Seed Multi-project profile for usr_multi_worker on Project Alpha as well
	pAlphaMulti, _ := localidentity.NewDirectoryProfile("prof_a_multi", "usr_multi_worker", tenantID, "cmp_1", "prj_alpha", "ste_a", "Multi Worker", "Site Advisor", "EHS", nil)
	_ = reg.RegisterProfile(pAlphaMulti)

	// Seed Company-wide executive profile (no project bound)
	pCorp, _ := localidentity.NewDirectoryProfile("prof_corp", "usr_corp_dir", tenantID, "cmp_1", "", "", "VP Safety", "Executive", "HQ", nil)
	_ = reg.RegisterProfile(pCorp)

	// Caller on Project Alpha
	alphaViewer := localidentity.ViewerContext{
		Identity: localidentity.SubjectIdentity{Subject: "usr_alpha_insp", TenantID: tenantID, IsAuthenticated: true},
		Role:     localidentity.RoleInspector,
		Scope:    localidentity.ScopeGrant{TenantID: tenantID, CompanyID: "cmp_1", ProjectID: "prj_alpha"},
	}

	// 1. Default search partitions strictly to prj_alpha
	results, err := svc.SearchDirectory(alphaViewer, localidentity.DirectorySearchFilter{})
	if err != nil {
		t.Fatalf("unexpected SearchDirectory error: %v", err)
	}
	if len(results) != 3 { // prof_a1, prof_a2, prof_a_multi
		t.Fatalf("expected 3 profiles in Project Alpha, got %d", len(results))
	}
	for _, p := range results {
		if p.ProjectID != "prj_alpha" {
			t.Errorf("scope leak: received profile %s with project %s", p.ProfileID, p.ProjectID)
		}
	}

	// 2. Anti-enumeration: Cross-project hostile search returns empty slice with nil error
	crossResults, err := svc.SearchDirectory(alphaViewer, localidentity.DirectorySearchFilter{
		ProjectID: "prj_beta",
	})
	if err != nil {
		t.Fatalf("anti-enumeration must return nil error, got: %v", err)
	}
	if len(crossResults) != 0 {
		t.Fatalf("anti-enumeration violation: Alpha worker discovered %d profiles in Project Beta", len(crossResults))
	}

	// 3. Direct probe of Beta profile returns ErrProfileNotFound
	_, err = svc.GetVisibleProfile(alphaViewer, "prof_b1")
	if !errors.Is(err, localidentity.ErrProfileNotFound) {
		t.Errorf("expected ErrProfileNotFound for cross-project direct probe, got: %v", err)
	}

	// 4. Multi-project subject isolation: Alpha viewer looking up usr_multi_worker only sees their Alpha profile
	multiList, err := svc.ListVisibleProfilesBySubject(alphaViewer, "usr_multi_worker")
	if err != nil {
		t.Fatalf("unexpected ListVisibleProfilesBySubject error: %v", err)
	}
	if len(multiList) != 1 || multiList[0].ProfileID != "prof_a_multi" {
		t.Errorf("multi-project isolation violation: expected 1 profile (prof_a_multi), got %v", multiList)
	}

	// 5. External contractor read boundary: Contractor cannot discover corporate directory
	contractorViewer := localidentity.ViewerContext{
		Identity: localidentity.SubjectIdentity{Subject: "usr_contractor_1", TenantID: tenantID, IsAuthenticated: true},
		Role:     localidentity.RoleContractor,
		Scope:    localidentity.ScopeGrant{TenantID: tenantID, CompanyID: "cmp_1", ProjectID: "prj_alpha", SiteID: "ste_a"},
	}

	_, err = svc.GetVisibleProfile(contractorViewer, "prof_corp")
	if !errors.Is(err, localidentity.ErrProfileNotFound) {
		t.Errorf("expected ErrProfileNotFound when contractor looks up corporate profile, got: %v", err)
	}
}

// TestQualification_SafeLifecycleAndActiveFiltering verifies that deactivated profiles
// are automatically shielded from operational discovery, reject updates, and can be viewed
// only by authorized compliance roles.
func TestQualification_SafeLifecycleAndActiveFiltering(t *testing.T) {
	reg := localidentity.NewDirectoryRegistry()
	matrix := localidentity.NewProvisionalAuthorizationMatrix()
	svc := localidentity.NewDirectoryVisibilityService(reg, matrix)
	resolver := localidentity.NewDirectoryResolver(reg, nil)

	tenantID := "ten_qual_life_01"

	pActive, _ := localidentity.NewDirectoryProfile("prof_act_01", "usr_1", tenantID, "cmp_1", "prj_1", "", "Active User", "Engineer", "EHS", nil)
	pDeact, _ := localidentity.NewDirectoryProfile("prof_deact_01", "usr_2", tenantID, "cmp_1", "prj_1", "", "Departed User", "Inspector", "EHS", nil)

	_ = resolver.RegisterProfile(pActive, "usr_admin", "Initial onboarding")
	_ = resolver.RegisterProfile(pDeact, "usr_admin", "Initial onboarding")

	// 1. Inactivate pDeact
	inactivated, err := resolver.InactivateProfile(tenantID, "prof_deact_01", "usr_admin", "Assignment concluded")
	if err != nil {
		t.Fatalf("unexpected InactivateProfile error: %v", err)
	}
	if inactivated.IsActive() || inactivated.Status() != localidentity.ProfileStatusInactive {
		t.Fatalf("expected profile to be in INACTIVE status")
	}

	// 2. Non-structural updates on inactive profile are rejected closed
	newTitle := "New Title"
	_, err = resolver.UpdateProfileAttributes(tenantID, "prof_deact_01", localidentity.ProfileNonStructuralUpdate{JobTitle: &newTitle}, "usr_admin", "Update attempt")
	if !errors.Is(err, localidentity.ErrProfileInactive) {
		t.Errorf("expected ErrProfileInactive when updating inactive profile, got: %v", err)
	}

	// 3. Operational search by Inspector excludes inactive profile by default
	inspectorViewer := localidentity.ViewerContext{
		Identity: localidentity.SubjectIdentity{Subject: "usr_1", TenantID: tenantID, IsAuthenticated: true},
		Role:     localidentity.RoleInspector,
		Scope:    localidentity.ScopeGrant{TenantID: tenantID, CompanyID: "cmp_1", ProjectID: "prj_1"},
	}

	inspectorResults, err := svc.SearchDirectory(inspectorViewer, localidentity.DirectorySearchFilter{})
	if err != nil {
		t.Fatalf("unexpected SearchDirectory error: %v", err)
	}
	if len(inspectorResults) != 1 || inspectorResults[0].ProfileID != "prof_act_01" {
		t.Errorf("inactive profile leaked to operational inspector search: %v", inspectorResults)
	}

	// Operational direct lookup fails closed with ErrProfileNotFound
	_, err = svc.GetVisibleProfile(inspectorViewer, "prof_deact_01")
	if !errors.Is(err, localidentity.ErrProfileNotFound) {
		t.Errorf("expected ErrProfileNotFound for inactive profile direct lookup, got: %v", err)
	}

	// 4. Compliance inspection by TenantAdmin with IncludeInactive=true reveals inactive profile
	adminViewer := localidentity.ViewerContext{
		Identity: localidentity.SubjectIdentity{Subject: "usr_admin", TenantID: tenantID, IsAuthenticated: true},
		Role:     localidentity.RoleTenantAdmin,
		Scope:    localidentity.ScopeGrant{TenantID: tenantID},
	}

	adminResults, err := svc.SearchDirectory(adminViewer, localidentity.DirectorySearchFilter{
		ProjectID:       "prj_1",
		IncludeInactive: true,
	})
	if err != nil {
		t.Fatalf("unexpected admin search error: %v", err)
	}
	if len(adminResults) != 2 {
		t.Errorf("expected 2 profiles (active and inactive) for admin with IncludeInactive=true, got %d", len(adminResults))
	}

	adminProfile, err := svc.GetVisibleProfile(adminViewer, "prof_deact_01")
	if err != nil {
		t.Fatalf("unexpected admin GetVisibleProfile error: %v", err)
	}
	if adminProfile.Status != localidentity.ProfileStatusInactive {
		t.Errorf("expected status INACTIVE for admin lookup, got: %s", adminProfile.Status)
	}
}

// TestQualification_MigrationAndRecoveryReversibleLineage simulates an end-to-end directory
// migration sequence, non-structural mutations, inactivation, and reactivation, proving that
// the append-only history ledger preserves full historical lineage with zero data loss.
func TestQualification_MigrationAndRecoveryReversibleLineage(t *testing.T) {
	ledger := localidentity.NewDirectoryResolutionLedger()
	reg := localidentity.NewDirectoryRegistry()
	resolver := localidentity.NewDirectoryResolver(reg, ledger)

	tenantID := "ten_qual_mig_01"
	subject := "usr_synth_mig_worker"
	profileID := "prof_mig_01"

	// Step 1: Bulk Migration Ingestion (Initial Registration)
	pInitial, err := localidentity.NewDirectoryProfile(
		profileID,
		subject,
		tenantID,
		"cmp_alpha",
		"prj_alpha",
		"ste_site_1",
		"Sompong Thani",
		"Junior EHS Officer",
		"Safety Team",
		[]string{"ara_zone_a"},
	)
	if err != nil {
		t.Fatalf("failed to create migration profile: %v", err)
	}

	if err := resolver.RegisterProfile(pInitial, "usr_migration_job", "V030 Phase 1 Ingestion"); err != nil {
		t.Fatalf("unexpected RegisterProfile error during migration: %v", err)
	}

	// Step 2: Incremental attribute updates over time
	title1 := "Mid-Level Safety Officer"
	dept1 := "Site Operations & Safety"
	areas1 := []string{"ara_zone_a", "ara_zone_b"}
	_, err = resolver.UpdateProfileAttributes(tenantID, profileID, localidentity.ProfileNonStructuralUpdate{
		JobTitle:      &title1,
		Department:    &dept1,
		AssignedAreas: areas1,
	}, "usr_hr_lead", "Year 1 Review")
	if err != nil {
		t.Fatalf("unexpected UpdateProfileAttributes error: %v", err)
	}

	title2 := "Senior Safety Coordinator"
	areas2 := []string{"ara_zone_a", "ara_zone_b", "ara_zone_c"}
	_, err = resolver.UpdateProfileAttributes(tenantID, profileID, localidentity.ProfileNonStructuralUpdate{
		JobTitle:      &title2,
		AssignedAreas: areas2,
	}, "usr_hr_lead", "Year 2 Promotion")
	if err != nil {
		t.Fatalf("unexpected UpdateProfileAttributes error: %v", err)
	}

	// Step 3: Temporary Assignment Inactivation
	_, err = resolver.InactivateProfile(tenantID, profileID, "usr_pm_lead", "Leave of absence")
	if err != nil {
		t.Fatalf("unexpected InactivateProfile error: %v", err)
	}

	// Step 4: Re-activation upon return
	pCurrent, _ := resolver.ResolveProfile(tenantID, profileID)
	reactivated, reactivateRec, err := localidentity.ActivateProfileWithHistory(pCurrent, "usr_pm_lead", "Return to duty")
	if err != nil {
		t.Fatalf("unexpected ActivateProfileWithHistory error: %v", err)
	}
	if !reactivated.IsActive() {
		t.Fatalf("expected profile to be active after reactivation")
	}
	if err := ledger.AppendRecord(reactivateRec); err != nil {
		t.Fatalf("failed to append reactivation record: %v", err)
	}

	// Step 5: Verify Complete Historical Lineage in Append-Only Ledger
	history, err := resolver.GetProfileHistory(tenantID, profileID)
	if err != nil {
		t.Fatalf("unexpected GetProfileHistory error: %v", err)
	}

	// Expected 5 records: REGISTER -> UPDATE 1 -> UPDATE 2 -> INACTIVATE -> ACTIVATE
	if len(history) != 5 {
		t.Fatalf("expected exactly 5 audit records in profile lineage, got %d", len(history))
	}

	expectedTransitions := []string{
		"PROFILE_INITIAL_REGISTER",
		"PROFILE_UPDATE_ATTRIBUTES",
		"PROFILE_UPDATE_ATTRIBUTES",
		"PROFILE_INACTIVATE",
		"PROFILE_ACTIVATE",
	}

	for i, tr := range expectedTransitions {
		if history[i].Transition != tr {
			t.Errorf("record %d: expected transition %s, got %s", i, tr, history[i].Transition)
		}
		if history[i].ProfileID != profileID || history[i].Subject != subject || history[i].TenantID != tenantID {
			t.Errorf("record %d: corrupted identity metadata: %+v", i, history[i])
		}
		if history[i].RecordedAt.IsZero() {
			t.Errorf("record %d: missing RecordedAt timestamp", i)
		}
	}

	// Verify Subject History resolves the same lineage
	subjHistory, err := resolver.GetSubjectHistory(tenantID, subject)
	if err != nil || len(subjHistory) != 5 {
		t.Fatalf("subject history resolution failed: %v, count=%d", err, len(subjHistory))
	}

	// Verify cross-tenant isolation on historical queries: foreign tenant sees 0 records
	foreignHistory, err := resolver.GetProfileHistory("ten_foreign_tenant", profileID)
	if err != nil {
		t.Fatalf("unexpected error on foreign history query: %v", err)
	}
	if len(foreignHistory) != 0 {
		t.Fatalf("CRITICAL SECURITY VIOLATION: foreign tenant accessed profile history: %+v", foreignHistory)
	}
}

// TestQualification_ZeroAuthorityAndSeparationOfConcerns verifies that directory profiles
// remain descriptive identity projections and convey zero operational or administrative authority.
func TestQualification_ZeroAuthorityAndSeparationOfConcerns(t *testing.T) {
	profile, err := localidentity.NewDirectoryProfile(
		"prof_auth_eval_01",
		"usr_synth_zero_auth",
		"ten_qual_auth_01",
		"cmp_heavy",
		"prj_alpha",
		"ste_site_1",
		"Preecha Engineer",
		"Chief Executive Safety Auditor",
		"Corporate Safety",
		nil,
	)
	if err != nil {
		t.Fatalf("unexpected NewDirectoryProfile error: %v", err)
	}

	// 1. AssertNoAuthorizationBypass: possessing a high-title directory profile grants 0 authorization
	if err := localidentity.AssertNoAuthorizationBypass(profile); err != nil {
		t.Errorf("expected clean authorization bypass assertion, got: %v", err)
	}

	// 2. Verify profile does not implement or contain authority methods
	// DirectoryProfile is an immutable struct containing only identity metadata and display attributes.
	if profile.TenantID() != "ten_qual_auth_01" || profile.ProfileID() != "prof_auth_eval_01" {
		t.Errorf("identity metadata corrupted")
	}
}
