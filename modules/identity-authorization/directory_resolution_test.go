package localidentity

import (
	"errors"
	"testing"
)

func TestDirectoryResolution_DuplicateCollisionRejection(t *testing.T) {
	resolver := NewDirectoryResolver(nil, nil)
	tenantID := "ten_alpha"

	p1, err := NewDirectoryProfile("prof_dup_01", "usr_synth_01", tenantID, "cmp_a", "prj_01", "ste_01", "Alice Somchai", "Lead", "Eng", nil)
	if err != nil {
		t.Fatalf("unexpected NewDirectoryProfile error: %v", err)
	}

	// First registration succeeds
	if err := resolver.RegisterProfile(p1, "usr_admin", "Initial registration"); err != nil {
		t.Fatalf("unexpected RegisterProfile error: %v", err)
	}

	// Attempting to register another profile with identical profileID in same tenant fails closed
	p2, _ := NewDirectoryProfile("prof_dup_01", "usr_synth_02", tenantID, "cmp_a", "prj_01", "ste_01", "Bob Prasert", "Lead", "Eng", nil)
	err = resolver.RegisterProfile(p2, "usr_admin", "Collision attempt")
	if !errors.Is(err, ErrDuplicateIdentifierCollision) {
		t.Errorf("expected ErrDuplicateIdentifierCollision on duplicate profile ID, got %v", err)
	}
}

func TestDirectoryResolution_ExplicitFalseMergeProhibition(t *testing.T) {
	tenantID := "ten_alpha"
	subjectA := "usr_synth_worker_01"
	subjectB := "usr_synth_worker_02" // Distinct worker sharing same display name and title

	pA, _ := NewDirectoryProfile("prof_a", subjectA, tenantID, "cmp_a", "prj_01", "ste_01", "Kamon Srisuk", "Safety Inspector", "EHS", []string{"ara_1"})

	// 1. AssertNoFalseMerge checks that an existing profile cannot be bound to a different subject
	if err := AssertNoFalseMerge(pA, subjectA); err != nil {
		t.Errorf("expected matching subject to pass, got %v", err)
	}
	if err := AssertNoFalseMerge(pA, subjectB); !errors.Is(err, ErrFalseMergeProhibited) {
		t.Errorf("expected ErrFalseMergeProhibited when attempting to bind profile to subject B, got %v", err)
	}

	// 2. AssertDistinctSubjects ensures two distinct subjects are never considered identical
	if err := AssertDistinctSubjects(subjectA, subjectA); err != nil {
		t.Errorf("expected identical subject to pass, got %v", err)
	}
	if err := AssertDistinctSubjects(subjectA, subjectB); !errors.Is(err, ErrFalseMergeProhibited) {
		t.Errorf("expected ErrFalseMergeProhibited between distinct subjects, got %v", err)
	}
}

func TestDirectoryResolution_NonStructuralUpdatesAndStructuralImmutability(t *testing.T) {
	resolver := NewDirectoryResolver(nil, nil)
	tenantID := "ten_alpha"
	subject := "usr_synth_01"

	p, _ := NewDirectoryProfile("prof_mod_01", subject, tenantID, "cmp_a", "prj_01", "ste_01", "Somchai P.", "Junior Safety Officer", "EHS", []string{"ara_1"})
	_ = resolver.RegisterProfile(p, "usr_admin", "Onboarding")

	// 1. Valid non-structural update: promotion and area expansion
	newTitle := "Senior Safety Coordinator"
	newDept := "Operations & Safety"
	newAreas := []string{"ara_1", "ara_2", "ara_3"}
	update := ProfileNonStructuralUpdate{
		JobTitle:      &newTitle,
		Department:    &newDept,
		AssignedAreas: newAreas,
	}

	updated, err := resolver.UpdateProfileAttributes(tenantID, "prof_mod_01", update, "usr_hr_lead", "Annual promotion")
	if err != nil {
		t.Fatalf("unexpected UpdateProfileAttributes error: %v", err)
	}

	// Verify updated non-structural fields
	if updated.JobTitle() != newTitle {
		t.Errorf("jobTitle mismatch: %s", updated.JobTitle())
	}
	if updated.Department() != newDept {
		t.Errorf("department mismatch: %s", updated.Department())
	}
	if len(updated.AssignedAreas()) != 3 {
		t.Errorf("expected 3 assigned areas, got %d", len(updated.AssignedAreas()))
	}

	// 2. Structural identity immutability assertion:
	// Verify ProfileID, Subject, TenantID, CompanyID, ProjectID, SiteID are 100% unchanged
	if updated.ProfileID() != p.ProfileID() || updated.Subject() != p.Subject() ||
		updated.TenantID() != p.TenantID() || updated.CompanyID() != p.CompanyID() ||
		updated.ProjectID() != p.ProjectID() || updated.SiteID() != p.SiteID() {
		t.Errorf("structural identity corruption detected during non-structural update")
	}

	// 3. Update with zero changes returns ErrNoNonStructuralChanges
	_, err = resolver.UpdateProfileAttributes(tenantID, "prof_mod_01", ProfileNonStructuralUpdate{}, "usr_hr", "Empty update")
	if !errors.Is(err, ErrNoNonStructuralChanges) {
		t.Errorf("expected ErrNoNonStructuralChanges for no-op update, got %v", err)
	}
}

func TestDirectoryResolution_InactivationAndActiveFiltering(t *testing.T) {
	resolver := NewDirectoryResolver(nil, nil)
	tenantID := "ten_alpha"

	p1, _ := NewDirectoryProfile("prof_active", "usr_01", tenantID, "cmp_a", "prj_target", "", "Active Worker", "Lead", "Eng", nil)
	p2, _ := NewDirectoryProfile("prof_term", "usr_02", tenantID, "cmp_a", "prj_target", "", "Departing Worker", "Lead", "Eng", nil)

	_ = resolver.RegisterProfile(p1, "usr_admin", "Register")
	_ = resolver.RegisterProfile(p2, "usr_admin", "Register")

	// 1. Inactivate p2
	inactivated, err := resolver.InactivateProfile(tenantID, "prof_term", "usr_admin", "Assignment ended")
	if err != nil {
		t.Fatalf("unexpected InactivateProfile error: %v", err)
	}
	if inactivated.IsActive() || inactivated.Status() != ProfileStatusInactive {
		t.Errorf("expected profile to be inactive")
	}

	// 2. Updating an inactive profile is rejected
	newTitle := "New Title"
	_, err = resolver.UpdateProfileAttributes(tenantID, "prof_term", ProfileNonStructuralUpdate{JobTitle: &newTitle}, "usr_admin", "Update attempt")
	if !errors.Is(err, ErrProfileInactive) {
		t.Errorf("expected ErrProfileInactive when updating inactive profile, got %v", err)
	}

	// 3. SearchActiveDirectory filters out inactive profiles by default
	activeList, err := resolver.SearchActiveDirectory(DirectoryQuery{TenantID: tenantID, ProjectID: "prj_target", IncludeInactive: false})
	if err != nil {
		t.Fatalf("SearchActiveDirectory error: %v", err)
	}
	if len(activeList) != 1 || activeList[0].ProfileID() != "prof_active" {
		t.Errorf("expected only active profile in active search, got %d results", len(activeList))
	}

	// 4. Querying with IncludeInactive reveals inactive profiles
	allList, err := resolver.SearchActiveDirectory(DirectoryQuery{TenantID: tenantID, ProjectID: "prj_target", IncludeInactive: true})
	if err != nil {
		t.Fatalf("SearchActiveDirectory include inactive error: %v", err)
	}
	if len(allList) != 2 {
		t.Errorf("expected 2 profiles when including inactive, got %d", len(allList))
	}
}

func TestDirectoryResolution_AppendOnlyHistory_IsolationAndNoHardDelete(t *testing.T) {
	resolver := NewDirectoryResolver(nil, nil)
	tenantA := "ten_alpha"
	tenantB := "ten_bravo"
	subjectA := "usr_synth_alpha_01"

	pA, _ := NewDirectoryProfile("prof_audit_01", subjectA, tenantA, "cmp_a", "prj_01", "", "Alpha User", "Officer", "EHS", []string{"ara_1"})
	_ = resolver.RegisterProfile(pA, "usr_admin", "Initial Onboarding")

	// Step 1: Update title
	title := "Senior Officer"
	_, _ = resolver.UpdateProfileAttributes(tenantA, "prof_audit_01", ProfileNonStructuralUpdate{JobTitle: &title}, "usr_hr", "Promotion")

	// Step 2: Inactivate
	_, _ = resolver.InactivateProfile(tenantA, "prof_audit_01", "usr_mgr", "Project completion")

	// Query Profile History for Tenant A
	history, err := resolver.GetProfileHistory(tenantA, "prof_audit_01")
	if err != nil {
		t.Fatalf("unexpected GetProfileHistory error: %v", err)
	}
	if len(history) != 3 {
		t.Fatalf("expected 3 history records (register, update, inactivate), got %d", len(history))
	}

	// Assert transitions
	if history[0].Transition != "PROFILE_INITIAL_REGISTER" {
		t.Errorf("record 0 transition mismatch: %s", history[0].Transition)
	}
	if history[1].Transition != "PROFILE_UPDATE_ATTRIBUTES" {
		t.Errorf("record 1 transition mismatch: %s", history[1].Transition)
	}
	if history[2].Transition != "PROFILE_INACTIVATE" {
		t.Errorf("record 2 transition mismatch: %s", history[2].Transition)
	}

	// Query Subject History
	subjHistory, err := resolver.GetSubjectHistory(tenantA, subjectA)
	if err != nil || len(subjHistory) != 3 {
		t.Errorf("subject history mismatch: %v, len=%d", err, len(subjHistory))
	}

	// Tenant boundary isolation: Tenant B query for Tenant A profile history returns 0 records
	leakCheck, err := resolver.GetProfileHistory(tenantB, "prof_audit_01")
	if err != nil {
		t.Fatalf("unexpected leak check error: %v", err)
	}
	if len(leakCheck) != 0 {
		t.Errorf("cross-tenant leakage: Tenant B accessed Tenant A profile history")
	}

	// Zero hard deletion assertion: history is append-only and past entries remain intact
	if history[0].ActorSubject != "usr_admin" || history[1].ActorSubject != "usr_hr" || history[2].ActorSubject != "usr_mgr" {
		t.Errorf("historical actor attribution corrupted")
	}
}

func TestDirectoryResolution_NoAuthOrSessionMutation(t *testing.T) {
	// Assert that directory resolution operations never alter session revocation registries or access policies
	resolver := NewDirectoryResolver(nil, nil)
	p, _ := NewDirectoryProfile("prof_auth_check", "usr_01", "ten_01", "cmp_01", "prj_01", "", "User", "Title", "Dept", nil)
	_ = resolver.RegisterProfile(p, "usr_admin", "Register")

	// Updating or inactivating directory profile conveys ZERO role mutation or token revocation
	inactivated, _ := resolver.InactivateProfile("ten_01", "prof_auth_check", "usr_admin", "Inactivate")
	if err := AssertNoAuthorizationBypass(inactivated); err != nil {
		t.Errorf("expected clean authorization separation, got %v", err)
	}
}
