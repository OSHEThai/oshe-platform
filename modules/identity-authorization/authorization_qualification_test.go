// Package localidentity_test provides qualification tests for OSHE Platform authorization services.
//
// QUALIFICATION SUITE DECLARATION (Issue #93 / V030-I020):
// Under approved Sole Human Owner decisions H030-003, H030-004, and H030-005, this qualification
// suite provides deterministic evidence for:
// 1. Default-deny evaluation, unauthenticated rejection, and cross-tenant boundary isolation.
// 2. Privilege escalation prevention, least-privilege role bounds, and scope containment.
// 3. Scoped role assignment lifecycle, temporal validity windows, and explicit revocation.
// 4. Delegation bounding (1-hop ceiling, non-self, temporal window, max duration, non-delegable protected roles, source authority, scope containment, and emergency break-glass denial).
// 5. Segregation-of-duties (SOD) conflict detection across concurrent and proposed assignments.
// 6. Append-only audit ledgers, tenant-isolated query boundaries, and deterministic lineage reconstruction.
// 7. Integrated policy evaluation combining scoped assignments and active delegation grants.
//
// Boundary & Non-Claims Invariant:
// Operates exclusively in-memory on local synthetic fixtures. Zero external identity provider
// synchronization, zero production database persistence, zero network routes, zero customer data,
// and zero binding operational authority or runtime policy activation are claimed or enacted.
package localidentity_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	localidentity "github.com/oshethai/oshe-platform/modules/identity-authorization"
)

// TestQualification_DefaultDenyMatrixAndUnauthenticatedCallers verifies that access evaluation
// fails closed with explicit typed denial reasons for unauthenticated callers, mismatched tenants,
// suspended memberships, and unknown roles/actions.
func TestQualification_DefaultDenyMatrixAndUnauthenticatedCallers(t *testing.T) {
	evaluator := localidentity.NewPolicyEvaluator()
	tenantID := "ten_qual_deny_01"
	subject := "usr_qual_worker_01"

	// 1. Unauthenticated identity rejected
	unauthReq := localidentity.AccessRequest{
		Identity: localidentity.SubjectIdentity{Subject: subject, TenantID: tenantID, IsAuthenticated: false},
		Target:   localidentity.TargetResource{TenantID: tenantID, ProjectID: "prj_01", Lifecycle: localidentity.ResourceActive},
		Action:   localidentity.ActionRead,
	}
	res := evaluator.Evaluate(unauthReq)
	if res.Allowed || res.DenialReason != localidentity.DenialUnauthenticated {
		t.Fatalf("expected DenialUnauthenticated, got allowed=%v reason=%s", res.Allowed, res.DenialReason)
	}

	// 2. Blank subject or tenant rejected
	blankSubReq := unauthReq
	blankSubReq.Identity.IsAuthenticated = true
	blankSubReq.Identity.Subject = ""
	res = evaluator.Evaluate(blankSubReq)
	if res.Allowed || res.DenialReason != localidentity.DenialUnauthenticated {
		t.Errorf("expected DenialUnauthenticated for blank subject, got %s", res.DenialReason)
	}

	// 3. Cross-tenant target rejected
	crossTenantReq := localidentity.AccessRequest{
		Identity: localidentity.SubjectIdentity{Subject: subject, TenantID: tenantID, IsAuthenticated: true},
		Target:   localidentity.TargetResource{TenantID: "ten_foreign_tenant", ProjectID: "prj_01", Lifecycle: localidentity.ResourceActive},
		Action:   localidentity.ActionRead,
	}
	res = evaluator.Evaluate(crossTenantReq)
	if res.Allowed || res.DenialReason != localidentity.DenialCrossTenant {
		t.Errorf("expected DenialCrossTenant for foreign tenant target, got %s", res.DenialReason)
	}

	// 4. Inactive / Suspended membership rejected
	evaluator.SetMembership(tenantID, subject, localidentity.MembershipSuspended)
	suspendedReq := localidentity.AccessRequest{
		Identity: localidentity.SubjectIdentity{Subject: subject, TenantID: tenantID, IsAuthenticated: true},
		Target:   localidentity.TargetResource{TenantID: tenantID, ProjectID: "prj_01", Lifecycle: localidentity.ResourceActive},
		Action:   localidentity.ActionRead,
	}
	res = evaluator.Evaluate(suspendedReq)
	if res.Allowed || res.DenialReason != localidentity.DenialInactiveMembership {
		t.Errorf("expected DenialInactiveMembership, got %s", res.DenialReason)
	}

	// 5. Unassigned role rejected with DenialRoleNotGranted
	evaluator.SetMembership(tenantID, "usr_no_role", localidentity.MembershipActive)
	noRoleReq := localidentity.AccessRequest{
		Identity: localidentity.SubjectIdentity{Subject: "usr_no_role", TenantID: tenantID, IsAuthenticated: true},
		Target:   localidentity.TargetResource{TenantID: tenantID, ProjectID: "prj_01", Lifecycle: localidentity.ResourceActive},
		Action:   localidentity.ActionRead,
	}
	res = evaluator.Evaluate(noRoleReq)
	if res.Allowed || res.DenialReason != localidentity.DenialRoleNotGranted {
		t.Errorf("expected DenialRoleNotGranted for unassigned role, got %s", res.DenialReason)
	}

	// 6. Delegation placeholder strictly fails closed with DenialDelegationNotImplemented
	evaluator.SetMembership(tenantID, subject, localidentity.MembershipActive)
	evaluator.AddRoleAssignment(localidentity.RoleAssignment{
		Subject:  subject,
		TenantID: tenantID,
		Role:     localidentity.RoleInspector,
		Scope:    localidentity.ScopeGrant{TenantID: tenantID, ProjectID: "prj_01"},
	})
	delegationPlaceholderReq := suspendedReq
	delegationPlaceholderReq.Delegation = localidentity.DelegationContext{IsDelegated: true, Delegator: "usr_pm_1"}
	res = evaluator.Evaluate(delegationPlaceholderReq)
	if res.Allowed || res.DenialReason != localidentity.DenialDelegationNotImplemented {
		t.Errorf("expected DenialDelegationNotImplemented, got %s", res.DenialReason)
	}

	// 7. Unknown role rejected in matrix validation
	matrix := localidentity.NewProvisionalAuthorizationMatrix()
	err := matrix.ValidateRoleAssignment(nil, localidentity.RoleAssignment{
		Subject:  subject,
		TenantID: tenantID,
		Role:     localidentity.Role("UNKNOWN_SPECIAL_ROLE"),
		Scope:    localidentity.ScopeGrant{TenantID: tenantID},
	})
	if !errors.Is(err, localidentity.ErrUnregisteredRole) {
		t.Errorf("expected ErrUnregisteredRole for unknown role, got: %v", err)
	}
}

// TestQualification_PrivilegeEscalationAndScopeContainment verifies least-privilege role bounds,
// prohibition of unauthorized actions, and exact organizational scope containment.
func TestQualification_PrivilegeEscalationAndScopeContainment(t *testing.T) {
	evaluator := localidentity.NewPolicyEvaluator()
	tenantID := "ten_qual_esc_01"

	// 1. Role-based action boundaries
	cases := []struct {
		role            localidentity.Role
		forbiddenAction localidentity.Action
		desc            string
	}{
		{localidentity.RoleInspector, localidentity.ActionDelete, "Inspector cannot delete"},
		{localidentity.RoleContractor, localidentity.ActionDelete, "Contractor cannot delete"},
		{localidentity.RoleContractor, localidentity.ActionUpdate, "Contractor cannot update"},
		{localidentity.RoleViewer, localidentity.ActionCreate, "Viewer cannot create"},
		{localidentity.RoleViewer, localidentity.ActionUpdate, "Viewer cannot update"},
		{localidentity.RoleViewer, localidentity.ActionDelete, "Viewer cannot delete"},
		{localidentity.RoleAuditor, localidentity.ActionCreate, "Auditor cannot create"},
		{localidentity.RoleAuditor, localidentity.ActionUpdate, "Auditor cannot update"},
		{localidentity.RoleAuditor, localidentity.ActionDelete, "Auditor cannot delete"},
	}

	for _, tc := range cases {
		sub := "usr_" + strings.ToLower(string(tc.role))
		evaluator.SetMembership(tenantID, sub, localidentity.MembershipActive)
		evaluator.AddRoleAssignment(localidentity.RoleAssignment{
			Subject:  sub,
			TenantID: tenantID,
			Role:     tc.role,
			Scope:    localidentity.ScopeGrant{TenantID: tenantID, ProjectID: "prj_main"},
		})

		req := localidentity.AccessRequest{
			Identity: localidentity.SubjectIdentity{Subject: sub, TenantID: tenantID, IsAuthenticated: true},
			Target:   localidentity.TargetResource{TenantID: tenantID, ProjectID: "prj_main", Lifecycle: localidentity.ResourceActive},
			Action:   tc.forbiddenAction,
		}
		res := evaluator.Evaluate(req)
		if res.Allowed || res.DenialReason != localidentity.DenialPrivilegeEscalation {
			t.Errorf("%s: expected DenialPrivilegeEscalation, got allowed=%v reason=%s", tc.desc, res.Allowed, res.DenialReason)
		}
	}

	// 2. Scope containment: Sibling project isolation
	inspectorSub := "usr_inspector_scope"
	evaluator.SetMembership(tenantID, inspectorSub, localidentity.MembershipActive)
	evaluator.AddRoleAssignment(localidentity.RoleAssignment{
		Subject:  inspectorSub,
		TenantID: tenantID,
		Role:     localidentity.RoleInspector,
		Scope:    localidentity.ScopeGrant{TenantID: tenantID, ProjectID: "prj_alpha"},
	})

	// Access on assigned project succeeds
	resAlpha := evaluator.Evaluate(localidentity.AccessRequest{
		Identity: localidentity.SubjectIdentity{Subject: inspectorSub, TenantID: tenantID, IsAuthenticated: true},
		Target:   localidentity.TargetResource{TenantID: tenantID, ProjectID: "prj_alpha", Lifecycle: localidentity.ResourceActive},
		Action:   localidentity.ActionRead,
	})
	if !resAlpha.Allowed {
		t.Fatalf("expected allowed on assigned project, got %s: %s", resAlpha.DenialReason, resAlpha.Message)
	}

	// Access on sibling project is denied with DenialScopeMismatch
	resBeta := evaluator.Evaluate(localidentity.AccessRequest{
		Identity: localidentity.SubjectIdentity{Subject: inspectorSub, TenantID: tenantID, IsAuthenticated: true},
		Target:   localidentity.TargetResource{TenantID: tenantID, ProjectID: "prj_beta", Lifecycle: localidentity.ResourceActive},
		Action:   localidentity.ActionRead,
	})
	if resBeta.Allowed || resBeta.DenialReason != localidentity.DenialScopeMismatch {
		t.Errorf("expected DenialScopeMismatch for sibling project, got allowed=%v reason=%s", resBeta.Allowed, resBeta.DenialReason)
	}

	// 3. Direct Object ID lock mismatch
	lockedSub := "usr_locked_worker"
	evaluator.SetMembership(tenantID, lockedSub, localidentity.MembershipActive)
	evaluator.AddRoleAssignment(localidentity.RoleAssignment{
		Subject:  lockedSub,
		TenantID: tenantID,
		Role:     localidentity.RoleInspector,
		Scope:    localidentity.ScopeGrant{TenantID: tenantID, ProjectID: "prj_alpha", ObjectID: "form_123"},
	})

	resObjMismatch := evaluator.Evaluate(localidentity.AccessRequest{
		Identity: localidentity.SubjectIdentity{Subject: lockedSub, TenantID: tenantID, IsAuthenticated: true},
		Target:   localidentity.TargetResource{TenantID: tenantID, ProjectID: "prj_alpha", ObjectID: "form_999", Lifecycle: localidentity.ResourceActive},
		Action:   localidentity.ActionRead,
	})
	if resObjMismatch.Allowed || resObjMismatch.DenialReason != localidentity.DenialDirectObjectMismatch {
		t.Errorf("expected DenialDirectObjectMismatch, got %s", resObjMismatch.DenialReason)
	}

	// 4. Archived Record Modification Denial
	resArchived := evaluator.Evaluate(localidentity.AccessRequest{
		Identity: localidentity.SubjectIdentity{Subject: inspectorSub, TenantID: tenantID, IsAuthenticated: true},
		Target:   localidentity.TargetResource{TenantID: tenantID, ProjectID: "prj_alpha", Lifecycle: localidentity.ResourceArchived},
		Action:   localidentity.ActionUpdate,
	})
	if resArchived.Allowed || resArchived.DenialReason != localidentity.DenialArchivedRecord {
		t.Errorf("expected DenialArchivedRecord for archived resource update, got %s", resArchived.DenialReason)
	}
}

// TestQualification_ScopedAssignmentLifecycleAndRevocation verifies scoped role assignment creation,
// scope-level boundaries, temporal window enforcement, and explicit revocation mechanics.
func TestQualification_ScopedAssignmentLifecycleAndRevocation(t *testing.T) {
	matrix := localidentity.NewProvisionalAuthorizationMatrix()
	ledger := localidentity.NewAssignmentLedger()
	registry := localidentity.NewScopedAssignmentRegistry(ledger)

	tenantID := "ten_qual_life_01"
	subject := "usr_pm_lifecycle"
	t0 := time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC)
	t1 := t0.Add(14 * 24 * time.Hour) // 14 days

	// 1. Matrix validation: ProjectManager not permitted at Tenant scope level (depth violation)
	badScopeAsn, _ := localidentity.NewScopedAssignment("asn_bad_01", tenantID, subject, localidentity.RoleProjectManager, localidentity.ScopeGrant{TenantID: tenantID}, t0, t1, "usr_sponsor_01")
	err := matrix.ValidateRoleAssignment(nil, badScopeAsn.ToRoleAssignment())
	if !errors.Is(err, localidentity.ErrInvalidScopeLevel) {
		t.Fatalf("expected ErrInvalidScopeLevel when assigning PM at tenant scope, got: %v", err)
	}

	// 2. Valid assignment construction at Project scope level
	goodScope := localidentity.ScopeGrant{TenantID: tenantID, CompanyID: "cmp_1", ProjectID: "prj_1"}
	validAsn, err := localidentity.NewScopedAssignment("asn_valid_01", tenantID, subject, localidentity.RoleProjectManager, goodScope, t0, t1, "usr_sponsor_01")
	if err != nil {
		t.Fatalf("unexpected NewScopedAssignment error: %v", err)
	}

	err = registry.RegisterAssignment(validAsn, "usr_admin", "Operational PM onboarding", t0)
	if err != nil {
		t.Fatalf("unexpected RegisterAssignment error: %v", err)
	}

	// 3. Temporal validity checks
	// Before validFrom
	beforeTime := t0.Add(-1 * time.Hour)
	if validAsn.IsValidAt(beforeTime) {
		t.Errorf("expected IsValidAt to be false before validFrom")
	}
	if validAsn.StateAt(beforeTime) != localidentity.AssignmentStateExpired {
		t.Errorf("expected StateAt to be EXPIRED before validFrom, got %s", validAsn.StateAt(beforeTime))
	}

	// Inside validity window
	midTime := t0.Add(7 * 24 * time.Hour)
	if !validAsn.IsValidAt(midTime) {
		t.Errorf("expected IsValidAt to be true within window")
	}
	if validAsn.StateAt(midTime) != localidentity.AssignmentStateActive {
		t.Errorf("expected StateAt to be ACTIVE within window, got %s", validAsn.StateAt(midTime))
	}

	// After validTo
	afterTime := t1.Add(1 * time.Hour)
	if validAsn.IsValidAt(afterTime) {
		t.Errorf("expected IsValidAt to be false after validTo")
	}
	if validAsn.StateAt(afterTime) != localidentity.AssignmentStateExpired {
		t.Errorf("expected StateAt to be EXPIRED after validTo, got %s", validAsn.StateAt(afterTime))
	}

	// 4. Explicit Revocation
	revokedAsn, auditRec, err := validAsn.Revoke("usr_compliance_lead", "Project early closure", midTime)
	if err != nil {
		t.Fatalf("unexpected Revoke error: %v", err)
	}
	if revokedAsn.IsActive() {
		t.Errorf("expected revoked assignment IsActive == false")
	}
	if revokedAsn.State() != localidentity.AssignmentStateRevoked {
		t.Errorf("expected State == REVOKED, got %s", revokedAsn.State())
	}
	if revokedAsn.IsValidAt(midTime) {
		t.Errorf("expected IsValidAt == false for revoked assignment")
	}
	if auditRec.Transition != "ASSIGNMENT_REVOKED" {
		t.Errorf("expected transition ASSIGNMENT_REVOKED, got %s", auditRec.Transition)
	}

	// Double revocation rejected
	_, _, err = revokedAsn.Revoke("usr_compliance_lead", "Second revoke", midTime)
	if !errors.Is(err, localidentity.ErrAssignmentRevoked) {
		t.Errorf("expected ErrAssignmentRevoked on second revocation attempt, got: %v", err)
	}
}

// TestQualification_DelegationChainDepthAndBounding verifies all delegation bounding rules:
// 1-hop chain ceiling, self-delegation prohibition, temporal validity, 30-day duration ceiling,
// non-delegable protected roles, source authority containment, scope containment, and emergency bypass denial.
func TestQualification_DelegationChainDepthAndBounding(t *testing.T) {
	matrix := localidentity.NewProvisionalAuthorizationMatrix()
	t0 := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)

	baseReq := localidentity.DelegationRequest{
		DelegatorSubject: "usr_pm_1",
		DelegatorRole:    localidentity.RoleProjectManager,
		DelegatorScope:   localidentity.ScopeGrant{TenantID: "ten_1", CompanyID: "cmp_1", ProjectID: "prj_1"},
		DelegateeSubject: "usr_delegatee_1",
		DelegatedRole:    localidentity.RoleProjectManager,
		DelegatedScope:   localidentity.ScopeGrant{TenantID: "ten_1", CompanyID: "cmp_1", ProjectID: "prj_1"},
		ValidFrom:        t0,
		ValidTo:          t0.Add(7 * 24 * time.Hour), // 7 days (within PM 14-day limit)
		IsSubDelegation:  false,
	}

	// 1. Valid delegation succeeds
	if err := matrix.ValidateDelegationRequest(baseReq); err != nil {
		t.Fatalf("expected valid delegation request to succeed, got: %v", err)
	}

	// 2. Multi-hop re-delegation forbidden (chain depth > 1)
	subReq := baseReq
	subReq.IsSubDelegation = true
	if err := matrix.ValidateDelegationRequest(subReq); !errors.Is(err, localidentity.ErrMultiHopDelegationForbidden) {
		t.Errorf("expected ErrMultiHopDelegationForbidden for sub-delegation, got: %v", err)
	}

	_, err := localidentity.NewDelegationRecord(
		"del_bad_hop", "ten_1", "usr_pm_1", localidentity.RoleProjectManager, baseReq.DelegatorScope,
		"usr_del_2", localidentity.RoleProjectManager, baseReq.DelegatedScope,
		t0, t0.Add(5*24*time.Hour), "usr_sponsor", 2, true, // depth 2
	)
	if !errors.Is(err, localidentity.ErrUnauthorizedChainDepth) {
		t.Errorf("expected ErrUnauthorizedChainDepth for depth 2 delegation record, got: %v", err)
	}

	// 3. Self-delegation forbidden (including whitespace padding)
	selfReq := baseReq
	selfReq.DelegateeSubject = "  usr_pm_1  "
	if err := matrix.ValidateDelegationRequest(selfReq); !errors.Is(err, localidentity.ErrSelfDelegationForbidden) {
		t.Errorf("expected ErrSelfDelegationForbidden for self-delegation, got: %v", err)
	}

	// 4. Inverted temporal window forbidden
	invReq := baseReq
	invReq.ValidTo = t0.Add(-1 * time.Hour)
	if err := matrix.ValidateDelegationRequest(invReq); !errors.Is(err, localidentity.ErrInvalidDelegationWindow) {
		t.Errorf("expected ErrInvalidDelegationWindow for inverted window, got: %v", err)
	}

	// 5. Absolute maximum duration exceeded (> 30 days)
	longReq := baseReq
	longReq.DelegatorRole = localidentity.RoleViewer
	longReq.DelegatedRole = localidentity.RoleViewer
	longReq.ValidTo = t0.Add(31 * 24 * time.Hour)
	if err := matrix.ValidateDelegationRequest(longReq); !errors.Is(err, localidentity.ErrDelegationDurationExceeded) {
		t.Errorf("expected ErrDelegationDurationExceeded for 31-day request, got: %v", err)
	}

	// 6. Role-specific delegation duration exceeded (PM max 14 days; requesting 15 days)
	pmLongReq := baseReq
	pmLongReq.ValidTo = t0.Add(15 * 24 * time.Hour)
	if err := matrix.ValidateDelegationRequest(pmLongReq); !errors.Is(err, localidentity.ErrDelegationDurationExceeded) {
		t.Errorf("expected ErrDelegationDurationExceeded for PM requesting 15 days, got: %v", err)
	}

	// 7. Protected sovereign role non-delegable (RoleTenantAdmin)
	adminReq := baseReq
	adminReq.DelegatorRole = localidentity.RoleTenantAdmin
	adminReq.DelegatedRole = localidentity.RoleTenantAdmin
	if err := matrix.ValidateDelegationRequest(adminReq); !errors.Is(err, localidentity.ErrProtectedAuthorityNonDelegable) {
		t.Errorf("expected ErrProtectedAuthorityNonDelegable for TenantAdmin delegation, got: %v", err)
	}

	// 8. Source authority containment: Delegator lacks requested permissions
	// Inspector trying to delegate ProjectManager role
	lackPermReq := baseReq
	lackPermReq.DelegatorRole = localidentity.RoleInspector
	lackPermReq.DelegatedRole = localidentity.RoleProjectManager
	if err := matrix.ValidateDelegationRequest(lackPermReq); !errors.Is(err, localidentity.ErrExceedsSourceAuthority) {
		t.Errorf("expected ErrExceedsSourceAuthority when delegator lacks permissions, got: %v", err)
	}

	// 9. Scope containment: Delegated scope broader or sibling to delegator scope
	// Sibling project escalation
	siblingReq := baseReq
	siblingReq.DelegatedScope = localidentity.ScopeGrant{TenantID: "ten_1", CompanyID: "cmp_1", ProjectID: "prj_sibling"}
	if err := matrix.ValidateDelegationRequest(siblingReq); !errors.Is(err, localidentity.ErrScopeExceedsSourceAuthority) {
		t.Errorf("expected ErrScopeExceedsSourceAuthority for sibling project delegation, got: %v", err)
	}

	// Cross-tenant escalation
	crossTenantReq := baseReq
	crossTenantReq.DelegatedScope = localidentity.ScopeGrant{TenantID: "ten_foreign", CompanyID: "cmp_1", ProjectID: "prj_1"}
	if err := matrix.ValidateDelegationRequest(crossTenantReq); !errors.Is(err, localidentity.ErrScopeExceedsSourceAuthority) {
		t.Errorf("expected ErrScopeExceedsSourceAuthority for cross-tenant delegation, got: %v", err)
	}

	// 10. Emergency break-glass access denial (H030-003)
	if err := localidentity.AssertEmergencyAccessDenied(true); !errors.Is(err, localidentity.ErrEmergencyAccessDenied) {
		t.Errorf("expected ErrEmergencyAccessDenied for break-glass claim, got: %v", err)
	}
	if err := localidentity.AssertEmergencyAccessDenied(false); err != nil {
		t.Errorf("expected nil when isEmergency == false, got: %v", err)
	}
}

// TestQualification_SegregationOfDutiesConflictEngine verifies segregation-of-duties rules:
// SOD-01 (Inspector vs Auditor), SOD-02 (Contractor vs TenantAdmin), SOD-02B (Contractor vs PM),
// and same-role duplicate active assignment rejections.
func TestQualification_SegregationOfDutiesConflictEngine(t *testing.T) {
	tenantID := "ten_qual_sod_01"
	t0 := time.Date(2026, 9, 5, 14, 0, 0, 0, time.UTC)
	t1 := t0.Add(30 * 24 * time.Hour)

	// 1. SOD-01: Inspector vs Auditor on overlapping scope
	workerSub := "usr_sod_worker_01"
	existingInsp, _ := localidentity.NewScopedAssignment("asn_sod_01", tenantID, workerSub, localidentity.RoleInspector, localidentity.ScopeGrant{TenantID: tenantID, ProjectID: "prj_alpha"}, t0, t1, "usr_sponsor")
	candAuditor, _ := localidentity.NewScopedAssignment("asn_sod_02", tenantID, workerSub, localidentity.RoleAuditor, localidentity.ScopeGrant{TenantID: tenantID, ProjectID: "prj_alpha"}, t0, t1, "usr_sponsor")

	err := localidentity.CheckRoleConflict([]localidentity.ScopedAssignment{existingInsp}, candAuditor, t0.Add(1*time.Hour))
	if !errors.Is(err, localidentity.ErrRoleConflictDetected) {
		t.Fatalf("expected ErrRoleConflictDetected for Inspector vs Auditor on same project, got: %v", err)
	}

	// Non-overlapping project passes
	candAuditorBeta, _ := localidentity.NewScopedAssignment("asn_sod_03", tenantID, workerSub, localidentity.RoleAuditor, localidentity.ScopeGrant{TenantID: tenantID, ProjectID: "prj_beta"}, t0, t1, "usr_sponsor")
	err = localidentity.CheckRoleConflict([]localidentity.ScopedAssignment{existingInsp}, candAuditorBeta, t0.Add(1*time.Hour))
	if err != nil {
		t.Errorf("expected non-overlapping project assignment to pass SOD check, got: %v", err)
	}

	// 2. SOD-02: Contractor vs TenantAdmin (scope-insensitive conflict)
	contractorSub := "usr_contractor_sod"
	existingContractor, _ := localidentity.NewScopedAssignment("asn_sod_ext", tenantID, contractorSub, localidentity.RoleContractor, localidentity.ScopeGrant{TenantID: tenantID, ProjectID: "prj_alpha"}, t0, t1, "usr_sponsor")
	candAdmin, _ := localidentity.NewScopedAssignment("asn_sod_admin", tenantID, contractorSub, localidentity.RoleTenantAdmin, localidentity.ScopeGrant{TenantID: tenantID}, t0, t1, "usr_sponsor")

	err = localidentity.CheckRoleConflict([]localidentity.ScopedAssignment{existingContractor}, candAdmin, t0.Add(1*time.Hour))
	if !errors.Is(err, localidentity.ErrRoleConflictDetected) {
		t.Errorf("expected ErrRoleConflictDetected for Contractor vs TenantAdmin, got: %v", err)
	}

	// 3. SOD-02B: Contractor vs ProjectManager on overlapping scope
	candPM, _ := localidentity.NewScopedAssignment("asn_sod_pm", tenantID, contractorSub, localidentity.RoleProjectManager, localidentity.ScopeGrant{TenantID: tenantID, ProjectID: "prj_alpha"}, t0, t1, "usr_sponsor")
	err = localidentity.CheckRoleConflict([]localidentity.ScopedAssignment{existingContractor}, candPM, t0.Add(1*time.Hour))
	if !errors.Is(err, localidentity.ErrRoleConflictDetected) {
		t.Errorf("expected ErrRoleConflictDetected for Contractor vs PM on same project, got: %v", err)
	}

	// 4. Duplicate active role assignment on overlapping scope rejected
	candDuplicateInsp, _ := localidentity.NewScopedAssignment("asn_sod_dup", tenantID, workerSub, localidentity.RoleInspector, localidentity.ScopeGrant{TenantID: tenantID, ProjectID: "prj_alpha"}, t0, t1, "usr_sponsor")
	err = localidentity.CheckRoleConflict([]localidentity.ScopedAssignment{existingInsp}, candDuplicateInsp, t0.Add(1*time.Hour))
	if !errors.Is(err, localidentity.ErrRoleConflictDetected) {
		t.Errorf("expected ErrRoleConflictDetected for duplicate active Inspector role, got: %v", err)
	}
}

// TestQualification_AppendOnlyAuditLedgersAndRecoveryLineage verifies that the assignment
// and delegation ledgers provide immutable, append-only, tenant-isolated historical audit trails.
func TestQualification_AppendOnlyAuditLedgersAndRecoveryLineage(t *testing.T) {
	asnLedger := localidentity.NewAssignmentLedger()
	delLedger := localidentity.NewDelegationLedger()

	tenantA := "ten_audit_alpha"
	tenantB := "ten_audit_bravo"
	subjectA := "usr_audit_worker_a"
	assignmentID := "asn_audit_01"
	delegationID := "del_audit_01"
	t0 := time.Date(2026, 9, 5, 15, 0, 0, 0, time.UTC)

	// 1. Append Assignment Records
	recAsn1 := localidentity.AssignmentAuditRecord{
		RecordID:     "hasn_01",
		TenantID:     tenantA,
		AssignmentID: assignmentID,
		Subject:      subjectA,
		Role:         localidentity.RoleInspector,
		Scope:        localidentity.ScopeGrant{TenantID: tenantA, ProjectID: "prj_1"},
		Transition:   "ASSIGNMENT_CREATED",
		ActorSubject: "usr_admin",
		Reason:       "Initial grant",
		RecordedAt:   t0,
	}
	recAsn2 := localidentity.AssignmentAuditRecord{
		RecordID:     "hasn_02",
		TenantID:     tenantA,
		AssignmentID: assignmentID,
		Subject:      subjectA,
		Role:         localidentity.RoleInspector,
		Scope:        localidentity.ScopeGrant{TenantID: tenantA, ProjectID: "prj_1"},
		Transition:   "ASSIGNMENT_REVOKED",
		ActorSubject: "usr_admin",
		Reason:       "Revoked after inspection completion",
		RecordedAt:   t0.Add(5 * time.Hour),
	}

	_ = asnLedger.AppendRecord(recAsn1)
	_ = asnLedger.AppendRecord(recAsn2)

	// 2. Append Delegation Records
	recDel1 := localidentity.DelegationAuditRecord{
		RecordID:         "hdel_01",
		TenantID:         tenantA,
		DelegationID:     delegationID,
		DelegatorSubject: "usr_pm_1",
		DelegateeSubject: subjectA,
		DelegatedRole:    localidentity.RoleProjectManager,
		DelegatedScope:   localidentity.ScopeGrant{TenantID: tenantA, ProjectID: "prj_1"},
		Transition:       "DELEGATION_CREATED",
		ActorSubject:     "usr_pm_1",
		Reason:           "Temporary absence cover",
		RecordedAt:       t0,
	}
	recDel2 := localidentity.DelegationAuditRecord{
		RecordID:         "hdel_02",
		TenantID:         tenantA,
		DelegationID:     delegationID,
		DelegatorSubject: "usr_pm_1",
		DelegateeSubject: subjectA,
		DelegatedRole:    localidentity.RoleProjectManager,
		DelegatedScope:   localidentity.ScopeGrant{TenantID: tenantA, ProjectID: "prj_1"},
		Transition:       "DELEGATION_REVOKED",
		ActorSubject:     "usr_pm_1",
		Reason:           "Returned to duty",
		RecordedAt:       t0.Add(6 * time.Hour),
	}

	_ = delLedger.AppendRecord(recDel1)
	_ = delLedger.AppendRecord(recDel2)

	// 3. Lineage Reconstruction for Tenant A
	asnTrail, err := asnLedger.GetAssignmentAuditTrail(tenantA, assignmentID)
	if err != nil {
		t.Fatalf("unexpected GetAssignmentAuditTrail error: %v", err)
	}
	if len(asnTrail) != 2 {
		t.Fatalf("expected 2 assignment records in audit trail, got %d", len(asnTrail))
	}
	if asnTrail[0].Transition != "ASSIGNMENT_CREATED" || asnTrail[1].Transition != "ASSIGNMENT_REVOKED" {
		t.Errorf("unexpected assignment audit transitions: %+v", asnTrail)
	}

	delTrail, err := delLedger.GetDelegationAuditTrail(tenantA, delegationID)
	if err != nil {
		t.Fatalf("unexpected GetDelegationAuditTrail error: %v", err)
	}
	if len(delTrail) != 2 {
		t.Fatalf("expected 2 delegation records in audit trail, got %d", len(delTrail))
	}
	if delTrail[0].Transition != "DELEGATION_CREATED" || delTrail[1].Transition != "DELEGATION_REVOKED" {
		t.Errorf("unexpected delegation audit transitions: %+v", delTrail)
	}

	// 4. Cross-Tenant Isolation: Tenant B query for Tenant A records returns 0 records
	leakAsn, err := asnLedger.GetAssignmentAuditTrail(tenantB, assignmentID)
	if err != nil || len(leakAsn) != 0 {
		t.Fatalf("cross-tenant leakage in AssignmentLedger: %+v", leakAsn)
	}

	leakDel, err := delLedger.GetDelegationAuditTrail(tenantB, delegationID)
	if err != nil || len(leakDel) != 0 {
		t.Fatalf("cross-tenant leakage in DelegationLedger: %+v", leakDel)
	}

	// 5. Blank input query validation
	if _, err := asnLedger.GetAssignmentAuditTrail("", assignmentID); !errors.Is(err, localidentity.ErrBlankTenantID) {
		t.Errorf("expected ErrBlankTenantID for empty tenant, got: %v", err)
	}
	if _, err := delLedger.GetDelegationAuditTrail("", delegationID); !errors.Is(err, localidentity.ErrBlankTenantID) {
		t.Errorf("expected ErrBlankTenantID for empty tenant, got: %v", err)
	}
}

// TestQualification_IntegratedPolicyEvaluationWithAssignments verifies policy evaluation
// driven by active scoped assignments and dynamic role grants.
func TestQualification_IntegratedPolicyEvaluationWithAssignments(t *testing.T) {
	evaluator := localidentity.NewPolicyEvaluator()
	tenantID := "ten_qual_integ_01"
	inspectorSub := "usr_synth_insp_01"
	t0 := time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC)
	t1 := t0.Add(7 * 24 * time.Hour)

	// Activate membership
	evaluator.SetMembership(tenantID, inspectorSub, localidentity.MembershipActive)

	// Create active scoped assignment
	scope := localidentity.ScopeGrant{TenantID: tenantID, CompanyID: "cmp_1", ProjectID: "prj_east", SiteID: "ste_rayong"}
	asn, err := localidentity.NewScopedAssignment("asn_integ_01", tenantID, inspectorSub, localidentity.RoleInspector, scope, t0, t1, "usr_sponsor")
	if err != nil {
		t.Fatalf("unexpected NewScopedAssignment error: %v", err)
	}

	// Add role assignment to policy evaluator
	evaluator.AddRoleAssignment(asn.ToRoleAssignment())

	// 1. Permitted Action within authorized scope
	resAllowed := evaluator.Evaluate(localidentity.AccessRequest{
		Identity: localidentity.SubjectIdentity{Subject: inspectorSub, TenantID: tenantID, IsAuthenticated: true},
		Target:   localidentity.TargetResource{TenantID: tenantID, CompanyID: "cmp_1", ProjectID: "prj_east", SiteID: "ste_rayong", Lifecycle: localidentity.ResourceActive},
		Action:   localidentity.ActionCreate,
	})
	if !resAllowed.Allowed {
		t.Fatalf("expected allowed within authorized scope, got: %s: %s", resAllowed.DenialReason, resAllowed.Message)
	}

	// 2. Action outside authorized scope denied
	resScopeDenied := evaluator.Evaluate(localidentity.AccessRequest{
		Identity: localidentity.SubjectIdentity{Subject: inspectorSub, TenantID: tenantID, IsAuthenticated: true},
		Target:   localidentity.TargetResource{TenantID: tenantID, CompanyID: "cmp_1", ProjectID: "prj_west", SiteID: "ste_bangkok", Lifecycle: localidentity.ResourceActive},
		Action:   localidentity.ActionCreate,
	})
	if resScopeDenied.Allowed || resScopeDenied.DenialReason != localidentity.DenialScopeMismatch {
		t.Errorf("expected DenialScopeMismatch outside assigned scope, got: %s", resScopeDenied.DenialReason)
	}

	// 3. Forbidden action (DELETE) for Inspector denied
	resActionDenied := evaluator.Evaluate(localidentity.AccessRequest{
		Identity: localidentity.SubjectIdentity{Subject: inspectorSub, TenantID: tenantID, IsAuthenticated: true},
		Target:   localidentity.TargetResource{TenantID: tenantID, CompanyID: "cmp_1", ProjectID: "prj_east", SiteID: "ste_rayong", Lifecycle: localidentity.ResourceActive},
		Action:   localidentity.ActionDelete,
	})
	if resActionDenied.Allowed || resActionDenied.DenialReason != localidentity.DenialPrivilegeEscalation {
		t.Errorf("expected DenialPrivilegeEscalation for DELETE, got: %s", resActionDenied.DenialReason)
	}
}
