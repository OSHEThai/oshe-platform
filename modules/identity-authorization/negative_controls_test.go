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
