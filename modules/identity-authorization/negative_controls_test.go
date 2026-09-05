package localidentity_test

import (
	"crypto/sha256"
	"errors"
	"strings"
	"testing"
	"time"

	localidentity "github.com/oshethai/oshe-platform/modules/identity-authorization"
)

// Helper constructing a standard active policy evaluator fixture
func setupEvaluator() (*localidentity.PolicyEvaluator, string, string) {
	eval := localidentity.NewPolicyEvaluator()
	tenantID := "ten_synthetic_alpha"
	subject := "usr_subject_001"

	eval.SetMembership(tenantID, subject, localidentity.MembershipActive)
	return eval, tenantID, subject
}

func TestNegativeControl_AnonymousAndUnauthenticated(t *testing.T) {
	eval, tenantID, _ := setupEvaluator()

	cases := []struct {
		name string
		id   localidentity.SubjectIdentity
	}{
		{
			name: "unauthenticated_flag_false",
			id:   localidentity.SubjectIdentity{Subject: "usr_1", TenantID: tenantID, IsAuthenticated: false},
		},
		{
			name: "blank_subject",
			id:   localidentity.SubjectIdentity{Subject: "", TenantID: tenantID, IsAuthenticated: true},
		},
		{
			name: "blank_tenant",
			id:   localidentity.SubjectIdentity{Subject: "usr_1", TenantID: "", IsAuthenticated: true},
		},
	}

	target := localidentity.TargetResource{TenantID: tenantID, Lifecycle: localidentity.ResourceActive}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res := eval.Evaluate(localidentity.AccessRequest{
				Identity: tc.id,
				Target:   target,
				Action:   localidentity.ActionRead,
			})
			if res.Allowed {
				t.Fatalf("expected anonymous/unauthenticated access to be denied")
			}
			if res.DenialReason != localidentity.DenialUnauthenticated {
				t.Errorf("expected DenialUnauthenticated, got %s", res.DenialReason)
			}
		})
	}
}

func TestNegativeControl_MalformedAndInvalidTokens(t *testing.T) {
	mgr := localidentity.NewIdentityManager(nil)
	id, _ := localidentity.NewIdentity("usr_1", "ten_alpha", localidentity.IdentityActive)
	_ = mgr.RegisterIdentity(id)

	malformedTokens := []struct {
		name  string
		token string
	}{
		{"empty", ""},
		{"whitespace", "   \t\n"},
		{"missing_prefix", "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"},
		{"wrong_prefix", "bear_1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"},
		{"truncated_hex", "oshe_tok_1234"},
		{"invalid_hex_chars", "oshe_tok_zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz"},
	}

	for _, tc := range malformedTokens {
		t.Run(tc.name, func(t *testing.T) {
			val, err := mgr.ValidateSession(tc.token, "ten_alpha")
			if err == nil {
				t.Fatalf("expected error validating malformed token %q, got nil", tc.token)
			}
			if !errors.Is(err, localidentity.ErrMalformedToken) {
				t.Errorf("expected ErrMalformedToken, got: %v", err)
			}
			if val.IsAuthenticated() {
				t.Error("malformed token must never produce authenticated identity")
			}
		})
	}
}

func TestNegativeControl_ExpiredAndRevokedSessions(t *testing.T) {
	mockTime := time.Date(2026, 9, 4, 18, 0, 0, 0, time.UTC)
	clock := func() time.Time { return mockTime }
	mgr := localidentity.NewIdentityManager(clock)

	id, _ := localidentity.NewIdentity("usr_exp", "ten_alpha", localidentity.IdentityActive)
	_ = mgr.RegisterIdentity(id)

	token, digest, err := mgr.IssueSession("usr_exp", 10*time.Minute)
	if err != nil {
		t.Fatalf("setup session failed: %v", err)
	}

	// 1. Advance clock past expiry
	mockTime = mockTime.Add(15 * time.Minute)
	val, err := mgr.ValidateSession(token, "ten_alpha")
	if err == nil {
		t.Fatal("expected expired token to fail validation, got nil")
	}
	if !errors.Is(err, localidentity.ErrTokenExpired) {
		t.Errorf("expected ErrTokenExpired, got: %v", err)
	}
	if val.IsAuthenticated() {
		t.Error("expired token must never yield authenticated identity")
	}

	// Reset clock, issue new token, and explicitly revoke
	mockTime = time.Date(2026, 9, 4, 18, 0, 0, 0, time.UTC)
	token2, digest2, _ := mgr.IssueSession("usr_exp", 10*time.Minute)
	if err := mgr.RevokeSession(digest2); err != nil {
		t.Fatalf("failed to revoke session: %v", err)
	}

	val2, err := mgr.ValidateSession(token2, "ten_alpha")
	if err == nil {
		t.Fatal("expected revoked token to fail validation, got nil")
	}
	if !errors.Is(err, localidentity.ErrTokenRevoked) {
		t.Errorf("expected ErrTokenRevoked, got: %v", err)
	}
	if val2.IsAuthenticated() {
		t.Error("revoked token must never yield authenticated identity")
	}

	_ = digest
}

func TestNegativeControl_StaleSessionAndPolicyInvalidation(t *testing.T) {
	reg := localidentity.NewRevocationRegistry(nil)
	tokenDigest := sha256.Sum256([]byte("token-stale-test"))
	tenantID := "ten_alpha"

	// 1. Subject revocation makes older sessions stale
	subjectA := "usr_subject_stale"
	_ = reg.RevokeSubject(tenantID, subjectA, "user password changed")

	diag := reg.EvaluateSession(tokenDigest, tenantID, tenantID, subjectA, 1)
	if diag.Allowed {
		t.Fatal("expected session issued before subject revocation to be denied")
	}
	if diag.DenialCategory != localidentity.CategorySessionStale {
		t.Errorf("expected CategorySessionStale, got: %s", diag.DenialCategory)
	}

	// 2. Policy generation bump makes older sessions stale for subject without prior subject revocation
	subjectB := "usr_subject_policy"
	newGen, _ := reg.BumpTenantPolicyGeneration(tenantID, "role reassignments")
	diagPolicy := reg.EvaluateSession(tokenDigest, tenantID, tenantID, subjectB, newGen-1)
	if diagPolicy.Allowed {
		t.Fatal("expected session issued before policy generation bump to be denied")
	}
	if diagPolicy.DenialCategory != localidentity.CategoryPolicyStale {
		t.Errorf("expected CategoryPolicyStale, got: %s", diagPolicy.DenialCategory)
	}
}

func TestNegativeControl_DisabledIdentity(t *testing.T) {
	mgr := localidentity.NewIdentityManager(nil)
	id, _ := localidentity.NewIdentity("usr_disabled", "ten_alpha", localidentity.IdentityDisabled)
	_ = mgr.RegisterIdentity(id)

	// Attempting to issue session for disabled identity must fail
	_, _, err := mgr.IssueSession("usr_disabled", 10*time.Minute)
	if err == nil {
		t.Fatal("expected disabled identity to fail session issuance")
	}
	if !errors.Is(err, localidentity.ErrIdentityDisabled) {
		t.Errorf("expected ErrIdentityDisabled, got: %v", err)
	}

	// Active identity issued session, then transitioned to disabled
	idActive, _ := localidentity.NewIdentity("usr_to_disable", "ten_alpha", localidentity.IdentityActive)
	_ = mgr.RegisterIdentity(idActive)
	token, _, _ := mgr.IssueSession("usr_to_disable", 10*time.Minute)

	// Transition to disabled
	_ = mgr.SetIdentityState("usr_to_disable", localidentity.IdentityDisabled)

	val, err := mgr.ValidateSession(token, "ten_alpha")
	if err == nil {
		t.Fatal("expected session for disabled identity to fail validation")
	}
	if !errors.Is(err, localidentity.ErrIdentityDisabled) {
		t.Errorf("expected ErrIdentityDisabled, got: %v", err)
	}
	if val.IsAuthenticated() {
		t.Error("disabled identity must never be authenticated")
	}
}

func TestNegativeControl_CrossTenantMismatch(t *testing.T) {
	eval, tenantAlpha, subject := setupEvaluator()
	eval.AddRoleAssignment(localidentity.RoleAssignment{
		Subject:  subject,
		TenantID: tenantAlpha,
		Role:     localidentity.RoleTenantAdmin,
		Scope:    localidentity.ScopeGrant{TenantID: tenantAlpha},
	})

	// TenantAdmin of Alpha attempts to access resource belonging to TenantBeta
	targetBeta := localidentity.TargetResource{
		TenantID:  "ten_synthetic_beta",
		CompanyID: "comp_beta",
		ProjectID: "proj_beta",
		Lifecycle: localidentity.ResourceActive,
	}

	res := eval.Evaluate(localidentity.AccessRequest{
		Identity: localidentity.SubjectIdentity{Subject: subject, TenantID: tenantAlpha, IsAuthenticated: true},
		Target:   targetBeta,
		Action:   localidentity.ActionRead,
	})

	if res.Allowed {
		t.Fatal("CRITICAL SECURITY VIOLATION: Cross-tenant access was permitted")
	}
	if res.DenialReason != localidentity.DenialCrossTenant {
		t.Errorf("expected DenialCrossTenant, got: %s", res.DenialReason)
	}
}

func TestNegativeControl_ProjectAndAreaScopeMismatch(t *testing.T) {
	eval, tenantID, subject := setupEvaluator()

	// Grant Inspector role scoped strictly to Project Alpha
	eval.AddRoleAssignment(localidentity.RoleAssignment{
		Subject:  subject,
		TenantID: tenantID,
		Role:     localidentity.RoleInspector,
		Scope: localidentity.ScopeGrant{
			TenantID:  tenantID,
			ProjectID: "proj_alpha",
			SiteID:    "site_1",
		},
	})

	// Attempting to access Project Beta
	targetProjectBeta := localidentity.TargetResource{
		TenantID:  tenantID,
		ProjectID: "proj_beta",
		SiteID:    "site_1",
		Lifecycle: localidentity.ResourceActive,
	}

	res := eval.Evaluate(localidentity.AccessRequest{
		Identity: localidentity.SubjectIdentity{Subject: subject, TenantID: tenantID, IsAuthenticated: true},
		Target:   targetProjectBeta,
		Action:   localidentity.ActionRead,
	})

	if res.Allowed {
		t.Fatal("Scope mismatch violation: user accessed sibling project")
	}
	if res.DenialReason != localidentity.DenialScopeMismatch {
		t.Errorf("expected DenialScopeMismatch, got: %s", res.DenialReason)
	}

	// Attempting to access Site 2 within Project Alpha
	targetSite2 := localidentity.TargetResource{
		TenantID:  tenantID,
		ProjectID: "proj_alpha",
		SiteID:    "site_2",
		Lifecycle: localidentity.ResourceActive,
	}

	resSite2 := eval.Evaluate(localidentity.AccessRequest{
		Identity: localidentity.SubjectIdentity{Subject: subject, TenantID: tenantID, IsAuthenticated: true},
		Target:   targetSite2,
		Action:   localidentity.ActionRead,
	})

	if resSite2.Allowed {
		t.Fatal("Scope mismatch violation: user accessed sibling site")
	}
	if resSite2.DenialReason != localidentity.DenialScopeMismatch {
		t.Errorf("expected DenialScopeMismatch, got: %s", resSite2.DenialReason)
	}
}

func TestNegativeControl_DirectObjectIDORMismatch(t *testing.T) {
	eval, tenantID, subject := setupEvaluator()

	// Grant access locked to a specific object ID under project scope
	eval.AddRoleAssignment(localidentity.RoleAssignment{
		Subject:  subject,
		TenantID: tenantID,
		Role:     localidentity.RoleInspector,
		Scope: localidentity.ScopeGrant{
			TenantID:  tenantID,
			ProjectID: "proj_scoped",
			ObjectID:  "ins_authorized_100",
		},
	})

	// Caller attempts IDOR against sibling object under same project scope
	targetOtherObject := localidentity.TargetResource{
		TenantID:  tenantID,
		ProjectID: "proj_scoped",
		ObjectID:  "ins_unauthorized_200",
		Lifecycle: localidentity.ResourceActive,
	}

	res := eval.Evaluate(localidentity.AccessRequest{
		Identity: localidentity.SubjectIdentity{Subject: subject, TenantID: tenantID, IsAuthenticated: true},
		Target:   targetOtherObject,
		Action:   localidentity.ActionRead,
	})

	if res.Allowed {
		t.Fatal("IDOR violation: user accessed object outside direct object lock")
	}
	if res.DenialReason != localidentity.DenialDirectObjectMismatch {
		t.Errorf("expected DenialDirectObjectMismatch, got: %s", res.DenialReason)
	}
}

func TestNegativeControl_PrivilegeEscalation(t *testing.T) {
	eval, tenantID, subject := setupEvaluator()

	// Assign VIEWER role
	eval.AddRoleAssignment(localidentity.RoleAssignment{
		Subject:  subject,
		TenantID: tenantID,
		Role:     localidentity.RoleViewer,
		Scope:    localidentity.ScopeGrant{TenantID: tenantID},
	})

	target := localidentity.TargetResource{TenantID: tenantID, Lifecycle: localidentity.ResourceActive}

	// Viewer attempting CREATE, UPDATE, DELETE must fail closed with DenialPrivilegeEscalation
	prohibitedActions := []localidentity.Action{
		localidentity.ActionCreate,
		localidentity.ActionUpdate,
		localidentity.ActionDelete,
	}

	for _, action := range prohibitedActions {
		t.Run(string(action), func(t *testing.T) {
			res := eval.Evaluate(localidentity.AccessRequest{
				Identity: localidentity.SubjectIdentity{Subject: subject, TenantID: tenantID, IsAuthenticated: true},
				Target:   target,
				Action:   action,
			})
			if res.Allowed {
				t.Fatalf("privilege escalation violation: Viewer permitted %s", action)
			}
			if res.DenialReason != localidentity.DenialPrivilegeEscalation {
				t.Errorf("expected DenialPrivilegeEscalation, got: %s", res.DenialReason)
			}
		})
	}
}

func TestNegativeControl_SupportAccessDenial(t *testing.T) {
	eval, tenantID, subject := setupEvaluator()

	// Assign SUPPORT role under scoped project
	eval.AddRoleAssignment(localidentity.RoleAssignment{
		Subject:  subject,
		TenantID: tenantID,
		Role:     localidentity.RoleSupport,
		Scope: localidentity.ScopeGrant{
			TenantID:  tenantID,
			ProjectID: "proj_support",
		},
	})

	target := localidentity.TargetResource{
		TenantID:  tenantID,
		ProjectID: "proj_support",
		Lifecycle: localidentity.ResourceActive,
	}

	// Support role is read-only; attempts to modify must be denied with DenialPrivilegeEscalation
	writeActions := []localidentity.Action{
		localidentity.ActionCreate,
		localidentity.ActionUpdate,
		localidentity.ActionDelete,
		localidentity.ActionExport,
	}

	for _, action := range writeActions {
		t.Run(string(action), func(t *testing.T) {
			res := eval.Evaluate(localidentity.AccessRequest{
				Identity: localidentity.SubjectIdentity{Subject: subject, TenantID: tenantID, IsAuthenticated: true},
				Target:   target,
				Action:   action,
			})
			if res.Allowed {
				t.Fatalf("support authorization boundary violation: Support permitted %s", action)
			}
			if res.DenialReason != localidentity.DenialPrivilegeEscalation {
				t.Errorf("expected DenialPrivilegeEscalation, got: %s", res.DenialReason)
			}
		})
	}
}

func TestNegativeControl_ContractorBoundary(t *testing.T) {
	eval, tenantID, subject := setupEvaluator()

	// Assign CONTRACTOR role under scoped project
	eval.AddRoleAssignment(localidentity.RoleAssignment{
		Subject:  subject,
		TenantID: tenantID,
		Role:     localidentity.RoleContractor,
		Scope: localidentity.ScopeGrant{
			TenantID:  tenantID,
			ProjectID: "proj_contractor",
		},
	})

	target := localidentity.TargetResource{
		TenantID:  tenantID,
		ProjectID: "proj_contractor",
		Lifecycle: localidentity.ResourceActive,
	}

	// Contractor cannot DELETE or EXPORT
	for _, action := range []localidentity.Action{localidentity.ActionDelete, localidentity.ActionExport} {
		t.Run(string(action), func(t *testing.T) {
			res := eval.Evaluate(localidentity.AccessRequest{
				Identity: localidentity.SubjectIdentity{Subject: subject, TenantID: tenantID, IsAuthenticated: true},
				Target:   target,
				Action:   action,
			})
			if res.Allowed {
				t.Fatalf("contractor boundary violation: Contractor permitted %s", action)
			}
			if res.DenialReason != localidentity.DenialPrivilegeEscalation {
				t.Errorf("expected DenialPrivilegeEscalation, got: %s", res.DenialReason)
			}
		})
	}
}

func TestNegativeControl_ArchivedRecordModification(t *testing.T) {
	eval, tenantID, subject := setupEvaluator()

	// Assign TENANT_ADMIN role (which normally can update)
	eval.AddRoleAssignment(localidentity.RoleAssignment{
		Subject:  subject,
		TenantID: tenantID,
		Role:     localidentity.RoleTenantAdmin,
		Scope:    localidentity.ScopeGrant{TenantID: tenantID},
	})

	// Target record is ARCHIVED
	targetArchived := localidentity.TargetResource{
		TenantID:  tenantID,
		Lifecycle: localidentity.ResourceArchived,
	}

	for _, action := range []localidentity.Action{localidentity.ActionUpdate, localidentity.ActionDelete} {
		t.Run(string(action), func(t *testing.T) {
			res := eval.Evaluate(localidentity.AccessRequest{
				Identity: localidentity.SubjectIdentity{Subject: subject, TenantID: tenantID, IsAuthenticated: true},
				Target:   targetArchived,
				Action:   action,
			})
			if res.Allowed {
				t.Fatalf("archived record immutability violation: Admin permitted %s on archived record", action)
			}
			if res.DenialReason != localidentity.DenialArchivedRecord {
				t.Errorf("expected DenialArchivedRecord, got: %s", res.DenialReason)
			}
		})
	}
}

func TestNegativeControl_NonLeakingDiagnostics(t *testing.T) {
	reg := localidentity.NewRevocationRegistry(nil)
	tokenDigest := sha256.Sum256([]byte("token-leak-test"))

	// Cross-tenant evaluation
	diag := reg.EvaluateSession(tokenDigest, "ten_caller_1", "ten_victim_2", "usr_attacker", 1)

	// Diagnostic summary must be generic and omit any target tenant name or victim subject details
	forbiddenSubstrings := []string{
		"victim",
		"ten_victim_2",
		"attacker",
		"secret",
		"role",
		"admin",
	}

	for _, f := range forbiddenSubstrings {
		if strings.Contains(strings.ToLower(diag.Summary), f) {
			t.Errorf("diagnostic summary leaks sensitive metadata %q: %q", f, diag.Summary)
		}
	}
}

// NEG-V030-04: Cross-Project Directory Enumeration Rejection (H030-005 / NFR-V030-PRIV-001)
// Threat: Unauthorized employee directory harvesting across projects.
// Test Scenario & Hostile Input: Authenticated worker on prj_alpha submits directory query targeting prj_beta.
// Expected Behavior: Query returns empty list ([]), anti-enumeration prevents existence discovery.
func TestNegativeControl_NEG_V030_04_CrossProjectDirectoryEnumeration(t *testing.T) {
	reg := localidentity.NewDirectoryRegistry()
	matrix := localidentity.NewProvisionalAuthorizationMatrix()

	// Seed profile in victim project prj_beta
	victimProfile, err := localidentity.NewDirectoryProfile(
		"prof_victim_01",
		"usr_synth_victim",
		"ten_alpha",
		"cmp_contracting",
		"prj_beta",
		"ste_site_b",
		"Victim Worker",
		"Senior Safety Officer",
		"Safety Operations",
		[]string{"ara_confined_space"},
	)
	if err != nil {
		t.Fatalf("failed to create fixture profile: %v", err)
	}
	if err := reg.RegisterProfile(victimProfile); err != nil {
		t.Fatalf("failed to register fixture profile: %v", err)
	}

	svc := localidentity.NewDirectoryVisibilityService(reg, matrix)

	// Attacker worker is bounded strictly to prj_alpha
	attackerViewer := localidentity.ViewerContext{
		Identity: localidentity.SubjectIdentity{
			Subject:         "usr_synth_attacker",
			TenantID:        "ten_alpha",
			IsAuthenticated: true,
		},
		Role:  localidentity.RoleInspector,
		Scope: localidentity.ScopeGrant{TenantID: "ten_alpha", CompanyID: "cmp_contracting", ProjectID: "prj_alpha"},
	}

	// 1. Hostile query: Attacker attempts to query prj_beta directory
	results, err := svc.SearchDirectory(attackerViewer, localidentity.DirectorySearchFilter{
		ProjectID: "prj_beta",
	})
	if err != nil {
		t.Fatalf("anti-enumeration requires nil error (HTTP 200 equivalent), got error: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("anti-enumeration failure (NEG-V030-04): attacker on prj_alpha received %d profiles from prj_beta", len(results))
	}

	// 2. Hostile query: Attacker omits project filter attempting to enumerate tenant-wide profiles
	unfilteredResults, err := svc.SearchDirectory(attackerViewer, localidentity.DirectorySearchFilter{})
	if err != nil {
		t.Fatalf("unexpected search error: %v", err)
	}
	for _, p := range unfilteredResults {
		if p.ProjectID != "prj_alpha" {
			t.Fatalf("exact-scope partition failure: attacker discovered profile %s in project %s", p.ProfileID, p.ProjectID)
		}
	}

	// 3. Hostile direct lookup: Attacker probes direct profile ID in prj_beta
	_, err = svc.GetVisibleProfile(attackerViewer, "prof_victim_01")
	if err == nil {
		t.Fatalf("expected ErrProfileNotFound for hostile direct profile probe, got nil")
	}
	if !errors.Is(err, localidentity.ErrProfileNotFound) {
		t.Errorf("expected non-leaking ErrProfileNotFound, got: %v", err)
	}
}

// Data Minimization Negative Control: Verifies no sensitive PII or credentials in directory outputs
func TestNegativeControl_DirectoryDataMinimization(t *testing.T) {
	reg := localidentity.NewDirectoryRegistry()
	matrix := localidentity.NewProvisionalAuthorizationMatrix()

	safeProfile, err := localidentity.NewDirectoryProfile(
		"prof_safe_01",
		"usr_synth_safe",
		"ten_alpha",
		"cmp_main",
		"prj_alpha",
		"ste_site_a",
		"Kallaya Sorn",
		"Safety Lead",
		"EHS Operations",
		[]string{"ara_1"},
	)
	if err != nil {
		t.Fatalf("failed to create safe profile: %v", err)
	}
	_ = reg.RegisterProfile(safeProfile)

	svc := localidentity.NewDirectoryVisibilityService(reg, matrix)
	viewer := localidentity.ViewerContext{
		Identity: localidentity.SubjectIdentity{Subject: "usr_synth_safe", TenantID: "ten_alpha", IsAuthenticated: true},
		Role:     localidentity.RoleInspector,
		Scope:    localidentity.ScopeGrant{TenantID: "ten_alpha", CompanyID: "cmp_main", ProjectID: "prj_alpha"},
	}

	p, err := svc.GetVisibleProfile(viewer, "prof_safe_01")
	if err != nil {
		t.Fatalf("failed to get visible profile: %v", err)
	}

	// Assert data minimization passes
	if err := localidentity.AssertDataMinimization(p); err != nil {
		t.Fatalf("data minimization check failed: %v", err)
	}

	// Verify absence of sensitive keys in struct representation
	forbiddenKeys := []string{"password", "token", "hash", "secret", "email", "phone", "ssn"}
	rep := p.DisplayName + " " + p.JobTitle + " " + p.Department
	for _, k := range forbiddenKeys {
		if strings.Contains(strings.ToLower(rep), k) {
			t.Errorf("found forbidden pattern %q in directory representation", k)
		}
	}
}

// NEG-V030-05: Duplicate Directory Identifier Collision & False-Merge Rejection (H030-003, H030-004)
// Threat: Identity aliasing and duplicate profile hijacking across synthetic workers.
// Test Scenario: Register profile with duplicate ID, attempt to alias two distinct synthetic subjects.
// Expected Behavior: Rejection with ErrDuplicateIdentifierCollision and ErrFalseMergeProhibited.
func TestNegativeControl_NEG_V030_05_DuplicateCollisionAndFalseMerge(t *testing.T) {
	resolver := localidentity.NewDirectoryResolver(nil, nil)
	tenantID := "ten_neg_dup_01"

	p1, err := localidentity.NewDirectoryProfile("prof_neg_01", "usr_worker_1", tenantID, "cmp_1", "prj_1", "", "Worker One", "Officer", "EHS", nil)
	if err != nil {
		t.Fatalf("failed to create profile: %v", err)
	}
	if err := resolver.RegisterProfile(p1, "usr_admin", "Register initial"); err != nil {
		t.Fatalf("failed to register profile: %v", err)
	}

	// Hostile input: duplicate profile ID registration
	pDup, _ := localidentity.NewDirectoryProfile("prof_neg_01", "usr_worker_2", tenantID, "cmp_1", "prj_1", "", "Worker Two", "Officer", "EHS", nil)
	err = resolver.RegisterProfile(pDup, "usr_admin", "Hostile duplicate registration")
	if !errors.Is(err, localidentity.ErrDuplicateIdentifierCollision) {
		t.Fatalf("expected ErrDuplicateIdentifierCollision, got: %v", err)
	}

	// Hostile input: false-merge attempt
	err = localidentity.AssertNoFalseMerge(p1, "usr_worker_2")
	if !errors.Is(err, localidentity.ErrFalseMergeProhibited) {
		t.Fatalf("expected ErrFalseMergeProhibited on false-merge, got: %v", err)
	}
}

// NEG-V030-06: Inactive Profile Attribute Mutation Denial (H030-005)
// Threat: Modifying decommissioned or departed worker directory profiles.
// Test Scenario: Inactivate active profile, then attempt non-structural attribute update.
// Expected Behavior: Update fails closed with ErrProfileInactive.
func TestNegativeControl_NEG_V030_06_InactiveProfileMutationDenial(t *testing.T) {
	resolver := localidentity.NewDirectoryResolver(nil, nil)
	tenantID := "ten_neg_inact_01"

	p, _ := localidentity.NewDirectoryProfile("prof_neg_inact_01", "usr_worker_inact", tenantID, "cmp_1", "prj_1", "", "Departing Worker", "Lead", "Eng", nil)
	_ = resolver.RegisterProfile(p, "usr_admin", "Register")

	_, err := resolver.InactivateProfile(tenantID, "prof_neg_inact_01", "usr_admin", "Departure")
	if err != nil {
		t.Fatalf("failed to inactivate profile: %v", err)
	}

	// Hostile input: attempt attribute update on inactive profile
	newTitle := "Unauthorized Title Mutation"
	_, err = resolver.UpdateProfileAttributes(tenantID, "prof_neg_inact_01", localidentity.ProfileNonStructuralUpdate{
		JobTitle: &newTitle,
	}, "usr_admin", "Hostile mutation")
	if !errors.Is(err, localidentity.ErrProfileInactive) {
		t.Fatalf("expected ErrProfileInactive when mutating inactive profile, got: %v", err)
	}
}

func TestNegativeControl_ExternalUser_MissingSponsor(t *testing.T) {
	tenantID := "ten_neg_ext"
	from := time.Now()
	to := from.Add(24 * time.Hour)

	// Blank sponsor rejected
	_, err := localidentity.NewExternalUserProfile(
		"usr_ext_01", tenantID, "cmp_01", localidentity.ExternalTypeTemporaryWorker,
		"", "Firm", "Worker", "ref_01", from, to, nil,
	)
	if !errors.Is(err, localidentity.ErrMissingInternalSponsor) {
		t.Errorf("expected ErrMissingInternalSponsor, got %v", err)
	}

	// External user acting as sponsor rejected (anti-chain sponsorship)
	_, err = localidentity.NewExternalUserProfile(
		"usr_ext_02", tenantID, "cmp_01", localidentity.ExternalTypeTemporaryWorker,
		"usr_ext_other_contractor", "Firm", "Worker", "ref_01", from, to, nil,
	)
	if !errors.Is(err, localidentity.ErrInvalidInternalSponsor) {
		t.Errorf("expected ErrInvalidInternalSponsor, got %v", err)
	}
}

func TestNegativeControl_ExternalUser_CompanyAdminDenial(t *testing.T) {
	// Negative control: external user cannot be assigned TenantAdmin or ProjectManager
	for uType := range localidentity.KnownExternalUserTypes {
		err := localidentity.AssertNoCompanyAdministration(uType, localidentity.RoleTenantAdmin)
		if !errors.Is(err, localidentity.ErrCompanyAdminDenied) {
			t.Errorf("expected ErrCompanyAdminDenied for %s as TenantAdmin, got %v", uType, err)
		}
		err = localidentity.AssertNoCompanyAdministration(uType, localidentity.RoleProjectManager)
		if !errors.Is(err, localidentity.ErrCompanyAdminDenied) {
			t.Errorf("expected ErrCompanyAdminDenied for %s as ProjectManager, got %v", uType, err)
		}
	}
}

func TestNegativeControl_ExternalUser_ExpiredEnrollment(t *testing.T) {
	tenantID := "ten_neg_ext"
	baseTime := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	from := baseTime
	to := baseTime.Add(24 * time.Hour) // expired on Sept 2

	profile, _ := localidentity.NewExternalUserProfile(
		"usr_ext_exp", tenantID, "cmp_01", localidentity.ExternalTypeContractorWorker,
		"usr_manager", "Vendor Firm", "Worker Exp", "ref_synth_01", from, to, nil,
	)

	// Check expired evaluation at Sept 5
	now := time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC)
	if profile.IsValidAt(now) {
		t.Errorf("expected IsValidAt false for expired external user")
	}
	if profile.EffectiveStatus(now) != localidentity.EnrollmentStatusExpired {
		t.Errorf("expected EnrollmentStatusExpired, got %v", profile.EffectiveStatus(now))
	}
}

func TestNegativeControl_ExternalUser_ProfileMinimization(t *testing.T) {
	tenantID := "ten_neg_ext"
	from := time.Now()
	to := from.Add(24 * time.Hour)

	// PII in display name or contact reference fails closed
	piiPayloads := []struct {
		name       string
		contactRef string
	}{
		{"Somchai somchai@email.com", "ref_01"},
		{"Somchai", "contact: user@vendor.com"},
		{"Somchai Phone: 0812345678", "ref_01"},
		{"Somchai", "+66891234567"},
		{"Somchai Citizen ID: 1234567890123", "ref_01"},
	}

	for _, tc := range piiPayloads {
		_, err := localidentity.NewExternalUserProfile(
			"usr_ext_pii", tenantID, "cmp_01", localidentity.ExternalTypeTemporaryWorker,
			"usr_manager", "Firm", tc.name, tc.contactRef, from, to, nil,
		)
		if !errors.Is(err, localidentity.ErrPIIDetected) {
			t.Errorf("expected ErrPIIDetected for %v, got %v", tc, err)
		}
	}
}

func TestNegativeControl_ScopedAssignment_Expired(t *testing.T) {
	eval := localidentity.NewPolicyEvaluator()
	reg := localidentity.NewScopedAssignmentRegistry(nil)
	tenantID := "ten_neg_01"
	subject := "usr_synth_exp"

	eval.SetMembership(tenantID, subject, localidentity.MembershipActive)

	baseTime := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	from := baseTime
	to := baseTime.Add(24 * time.Hour) // expired on Sept 2

	asn, _ := localidentity.NewScopedAssignment("asn_exp", tenantID, subject, localidentity.RoleInspector,
		localidentity.ScopeGrant{TenantID: tenantID, ProjectID: "prj_01"}, from, to, "usr_admin")
	_ = reg.RegisterAssignment(asn, "usr_admin", "Test", from)

	// Evaluate access after expiration (Sept 5)
	now := time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC)
	req := localidentity.AccessRequest{
		Identity: localidentity.SubjectIdentity{Subject: subject, TenantID: tenantID, IsAuthenticated: true},
		Target:   localidentity.TargetResource{TenantID: tenantID, ProjectID: "prj_01", Lifecycle: localidentity.ResourceActive},
		Action:   localidentity.ActionRead,
	}

	res := localidentity.EvaluateScopedAccess(reg, eval, req, now)
	if res.Allowed || res.DenialReason != localidentity.DenialRoleNotGranted {
		t.Errorf("expected DenialRoleNotGranted for expired assignment, got allowed=%v reason=%s", res.Allowed, res.DenialReason)
	}
}

func TestNegativeControl_ScopedAssignment_Revoked(t *testing.T) {
	eval := localidentity.NewPolicyEvaluator()
	reg := localidentity.NewScopedAssignmentRegistry(nil)
	tenantID := "ten_neg_01"
	subject := "usr_synth_rev"

	eval.SetMembership(tenantID, subject, localidentity.MembershipActive)

	from := time.Now().Add(-1 * time.Hour)
	to := time.Now().Add(24 * time.Hour)
	now := time.Now()

	asn, _ := localidentity.NewScopedAssignment("asn_rev", tenantID, subject, localidentity.RoleInspector,
		localidentity.ScopeGrant{TenantID: tenantID, ProjectID: "prj_01"}, from, to, "usr_admin")
	_ = reg.RegisterAssignment(asn, "usr_admin", "Test", now)

	// Explicitly revoke assignment
	_, err := reg.RevokeAssignment(tenantID, "asn_rev", "usr_admin", "Breach", now)
	if err != nil {
		t.Fatalf("RevokeAssignment error: %v", err)
	}

	req := localidentity.AccessRequest{
		Identity: localidentity.SubjectIdentity{Subject: subject, TenantID: tenantID, IsAuthenticated: true},
		Target:   localidentity.TargetResource{TenantID: tenantID, ProjectID: "prj_01", Lifecycle: localidentity.ResourceActive},
		Action:   localidentity.ActionRead,
	}

	res := localidentity.EvaluateScopedAccess(reg, eval, req, now)
	if res.Allowed || res.DenialReason != localidentity.DenialRoleNotGranted {
		t.Errorf("expected DenialRoleNotGranted for revoked assignment, got allowed=%v reason=%s", res.Allowed, res.DenialReason)
	}
}

func TestNegativeControl_ScopedAssignment_RoleConflict(t *testing.T) {
	reg := localidentity.NewScopedAssignmentRegistry(nil)
	tenantID := "ten_neg_01"
	subject := "usr_synth_conflict"

	from := time.Now().Add(-1 * time.Hour)
	to := time.Now().Add(24 * time.Hour)
	now := time.Now()

	scope := localidentity.ScopeGrant{TenantID: tenantID, ProjectID: "prj_01"}

	// Assign Inspector
	asn1, _ := localidentity.NewScopedAssignment("asn_c1", tenantID, subject, localidentity.RoleInspector, scope, from, to, "usr_admin")
	_ = reg.RegisterAssignment(asn1, "usr_admin", "Init", now)

	// Attempting to assign Auditor to the same subject on overlapping scope must fail with role conflict
	asn2, _ := localidentity.NewScopedAssignment("asn_c2", tenantID, subject, localidentity.RoleAuditor, scope, from, to, "usr_admin")
	err := reg.RegisterAssignment(asn2, "usr_admin", "Conflict attempt", now)
	if !errors.Is(err, localidentity.ErrRoleConflictDetected) {
		t.Errorf("expected ErrRoleConflictDetected for Inspector + Auditor, got %v", err)
	}
}

func TestNegativeControl_ScopedAssignment_ScopeMismatch(t *testing.T) {
	eval := localidentity.NewPolicyEvaluator()
	reg := localidentity.NewScopedAssignmentRegistry(nil)
	tenantID := "ten_neg_01"
	subject := "usr_synth_scopetest"

	eval.SetMembership(tenantID, subject, localidentity.MembershipActive)

	from := time.Now().Add(-1 * time.Hour)
	to := time.Now().Add(24 * time.Hour)
	now := time.Now()

	// Assign Inspector strictly to Project Alpha
	asn, _ := localidentity.NewScopedAssignment("asn_scope", tenantID, subject, localidentity.RoleInspector,
		localidentity.ScopeGrant{TenantID: tenantID, ProjectID: "prj_alpha"}, from, to, "usr_admin")
	_ = reg.RegisterAssignment(asn, "usr_admin", "Test", now)

	// Attempt to access Project Bravo
	req := localidentity.AccessRequest{
		Identity: localidentity.SubjectIdentity{Subject: subject, TenantID: tenantID, IsAuthenticated: true},
		Target:   localidentity.TargetResource{TenantID: tenantID, ProjectID: "prj_bravo", Lifecycle: localidentity.ResourceActive},
		Action:   localidentity.ActionRead,
	}

	res := localidentity.EvaluateScopedAccess(reg, eval, req, now)
	if res.Allowed || res.DenialReason != localidentity.DenialScopeMismatch {
		t.Errorf("expected DenialScopeMismatch for out-of-scope project, got allowed=%v reason=%s", res.Allowed, res.DenialReason)
	}
}

func TestNegativeControl_Delegation_EmergencyAccessDenial(t *testing.T) {
	// Negative control: emergency break-glass is strictly rejected under H030-003
	if err := localidentity.AssertEmergencyAccessDenied(true); !errors.Is(err, localidentity.ErrEmergencyAccessDenied) {
		t.Errorf("expected ErrEmergencyAccessDenied, got %v", err)
	}
}

func TestNegativeControl_Delegation_MultiHopChainDenial(t *testing.T) {
	tenantID := "ten_neg_del"
	from := time.Now()
	to := from.Add(7 * 24 * time.Hour)
	scope := localidentity.ScopeGrant{TenantID: tenantID, ProjectID: "prj_01"}

	// Chain depth > 1 rejected
	_, err := localidentity.NewDelegationRecord("del_hop", tenantID, "usr_pm", localidentity.RoleProjectManager, scope, "usr_sub", localidentity.RoleInspector, scope, from, to, "Appr", 2, false)
	if !errors.Is(err, localidentity.ErrUnauthorizedChainDepth) {
		t.Errorf("expected ErrUnauthorizedChainDepth for chain depth 2, got %v", err)
	}

	// Sub-delegation flag rejected
	_, err = localidentity.NewDelegationRecord("del_sub", tenantID, "usr_pm", localidentity.RoleProjectManager, scope, "usr_sub", localidentity.RoleInspector, scope, from, to, "Appr", 1, true)
	if !errors.Is(err, localidentity.ErrUnauthorizedChainDepth) {
		t.Errorf("expected ErrUnauthorizedChainDepth for isSubDelegation=true, got %v", err)
	}
}

func TestNegativeControl_Delegation_SelfDelegationDenial(t *testing.T) {
	tenantID := "ten_neg_del"
	from := time.Now()
	to := from.Add(7 * 24 * time.Hour)
	scope := localidentity.ScopeGrant{TenantID: tenantID, ProjectID: "prj_01"}

	// Self-delegation rejected
	_, err := localidentity.NewDelegationRecord("del_self", tenantID, "usr_same", localidentity.RoleProjectManager, scope, "usr_same", localidentity.RoleInspector, scope, from, to, "Appr", 1, false)
	if !errors.Is(err, localidentity.ErrSelfDelegationForbidden) {
		t.Errorf("expected ErrSelfDelegationForbidden, got %v", err)
	}
}

func TestNegativeControl_Delegation_ProtectedAuthorityDenial(t *testing.T) {
	matrix := localidentity.NewProvisionalAuthorizationMatrix()
	tenantID := "ten_neg_del"
	from := time.Now()
	to := from.Add(7 * 24 * time.Hour)
	scope := localidentity.ScopeGrant{TenantID: tenantID}

	// Attempting to delegate TenantAdmin is rejected
	rec, _ := localidentity.NewDelegationRecord("del_admin", tenantID, "usr_adm", localidentity.RoleTenantAdmin, scope, "usr_sub", localidentity.RoleTenantAdmin, scope, from, to, "Appr", 1, false)
	err := localidentity.ValidateDelegationAuthority(rec, &matrix)
	if !errors.Is(err, localidentity.ErrProtectedAuthorityNonDelegable) {
		t.Errorf("expected ErrProtectedAuthorityNonDelegable, got %v", err)
	}
}

func TestNegativeControl_Delegation_ExpiredDenial(t *testing.T) {
	eval := localidentity.NewPolicyEvaluator()
	reg := localidentity.NewDelegationRegistry(nil, nil)
	tenantID := "ten_neg_del"
	delegator := "usr_pm"
	delegatee := "usr_lead"

	eval.SetMembership(tenantID, delegator, localidentity.MembershipActive)
	eval.SetMembership(tenantID, delegatee, localidentity.MembershipActive)

	baseTime := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	from := baseTime
	to := baseTime.Add(24 * time.Hour) // expired Sept 2

	scope := localidentity.ScopeGrant{TenantID: tenantID, ProjectID: "prj_01"}
	rec, _ := localidentity.NewDelegationRecord("del_exp", tenantID, delegator, localidentity.RoleProjectManager, scope, delegatee, localidentity.RoleInspector, scope, from, to, "Appr", 1, false)
	_ = reg.CreateDelegation(rec, delegator, "Test", from)

	// Evaluate access after expiration (Sept 5)
	now := time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC)
	req := localidentity.AccessRequest{
		Identity: localidentity.SubjectIdentity{Subject: delegatee, TenantID: tenantID, IsAuthenticated: true},
		Target:   localidentity.TargetResource{TenantID: tenantID, ProjectID: "prj_01", Lifecycle: localidentity.ResourceActive},
		Action:   localidentity.ActionRead,
	}

	res := localidentity.EvaluateDelegatedAccess(reg, eval, req, now)
	if res.Allowed || res.DenialReason != localidentity.DenialRoleNotGranted {
		t.Errorf("expected DenialRoleNotGranted for expired delegation, got allowed=%v reason=%s", res.Allowed, res.DenialReason)
	}
}
