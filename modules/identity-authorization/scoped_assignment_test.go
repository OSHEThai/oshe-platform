package localidentity

import (
	"errors"
	"testing"
	"time"
)

func TestScopedAssignment_CreationAndAccessors(t *testing.T) {
	tenantID := "ten_alpha"
	subject := "usr_synth_somchai_01"
	role := RoleInspector
	scope := ScopeGrant{
		TenantID:  tenantID,
		CompanyID: "cmp_main",
		ProjectID: "prj_refinery",
		SiteID:    "ste_01",
		AreaID:    "ara_boiler_1",
	}
	baseTime := time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC)
	from := baseTime
	to := baseTime.Add(30 * 24 * time.Hour)
	approval := "usr_sponsor_lead"

	asn, err := NewScopedAssignment("asn_01", tenantID, subject, role, scope, from, to, approval)
	if err != nil {
		t.Fatalf("unexpected NewScopedAssignment error: %v", err)
	}

	if asn.AssignmentID() != "asn_01" {
		t.Errorf("assignmentID mismatch: %s", asn.AssignmentID())
	}
	if asn.TenantID() != tenantID {
		t.Errorf("tenantID mismatch: %s", asn.TenantID())
	}
	if asn.Subject() != subject {
		t.Errorf("subject mismatch: %s", asn.Subject())
	}
	if asn.Role() != role {
		t.Errorf("role mismatch: %s", asn.Role())
	}
	if asn.Scope().ProjectID != "prj_refinery" {
		t.Errorf("scope project mismatch: %s", asn.Scope().ProjectID)
	}
	if asn.ValidFrom() != from || asn.ValidTo() != to {
		t.Errorf("validity window mismatch")
	}
	if asn.ApprovalSource() != approval {
		t.Errorf("approval source mismatch: %s", asn.ApprovalSource())
	}
	if !asn.IsActive() || asn.State() != AssignmentStateActive {
		t.Errorf("expected active assignment state")
	}
}

func TestScopedAssignment_ValidationRejections(t *testing.T) {
	tenantID := "ten_alpha"
	subject := "usr_synth_01"
	role := RoleInspector
	scope := ScopeGrant{TenantID: tenantID, ProjectID: "prj_01"}
	from := time.Now()
	to := from.Add(24 * time.Hour)
	approval := "usr_approver"

	// 1. Blank assignment ID
	if _, err := NewScopedAssignment("", tenantID, subject, role, scope, from, to, approval); !errors.Is(err, ErrBlankAssignmentID) {
		t.Errorf("expected ErrBlankAssignmentID, got %v", err)
	}

	// 2. Blank tenant ID
	if _, err := NewScopedAssignment("asn_01", "", subject, role, scope, from, to, approval); !errors.Is(err, ErrBlankTenantID) {
		t.Errorf("expected ErrBlankTenantID, got %v", err)
	}

	// 3. Blank / non-synthetic subject
	if _, err := NewScopedAssignment("asn_01", tenantID, "", role, scope, from, to, approval); !errors.Is(err, ErrBlankSubject) {
		t.Errorf("expected ErrBlankSubject, got %v", err)
	}
	if _, err := NewScopedAssignment("asn_01", tenantID, "ext_vendor", role, scope, from, to, approval); !errors.Is(err, ErrInvalidSubjectFormat) {
		t.Errorf("expected ErrInvalidSubjectFormat, got %v", err)
	}

	// 4. Unknown role
	if _, err := NewScopedAssignment("asn_01", tenantID, subject, Role("SUPER_ADMIN"), scope, from, to, approval); !errors.Is(err, ErrUnknownRole) {
		t.Errorf("expected ErrUnknownRole, got %v", err)
	}

	// 5. Blank approval source
	if _, err := NewScopedAssignment("asn_01", tenantID, subject, role, scope, from, to, ""); !errors.Is(err, ErrBlankApprovalSource) {
		t.Errorf("expected ErrBlankApprovalSource, got %v", err)
	}

	// 6. Inverted / equal validity window
	if _, err := NewScopedAssignment("asn_01", tenantID, subject, role, scope, to, from, approval); !errors.Is(err, ErrInvalidTimeWindow) {
		t.Errorf("expected ErrInvalidTimeWindow for inverted dates, got %v", err)
	}
	if _, err := NewScopedAssignment("asn_01", tenantID, subject, role, scope, from, from, approval); !errors.Is(err, ErrInvalidTimeWindow) {
		t.Errorf("expected ErrInvalidTimeWindow for equal dates, got %v", err)
	}

	// 7. Tenant mismatch between scope and assignment
	badScope := ScopeGrant{TenantID: "ten_other", ProjectID: "prj_01"}
	if _, err := NewScopedAssignment("asn_01", tenantID, subject, role, badScope, from, to, approval); err == nil {
		t.Errorf("expected error for scope tenant mismatch")
	}
}

func TestScopedAssignment_TemporalValidity(t *testing.T) {
	tenantID := "ten_alpha"
	subject := "usr_synth_01"
	baseTime := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	from := baseTime
	to := baseTime.Add(10 * 24 * time.Hour)

	asn, _ := NewScopedAssignment("asn_01", tenantID, subject, RoleViewer, ScopeGrant{TenantID: tenantID}, from, to, "usr_admin")

	before := baseTime.Add(-1 * time.Hour)
	during := baseTime.Add(5 * 24 * time.Hour)
	after := to.Add(1 * time.Hour)

	if !asn.IsValidAt(during) {
		t.Errorf("expected valid during validity window")
	}
	if asn.IsValidAt(before) {
		t.Errorf("expected invalid before validity window")
	}
	if asn.IsValidAt(after) {
		t.Errorf("expected invalid after validity window")
	}

	if asn.StateAt(during) != AssignmentStateActive {
		t.Errorf("expected active state during window")
	}
	if asn.StateAt(after) != AssignmentStateExpired {
		t.Errorf("expected expired state after window")
	}
}

func TestScopedAssignment_RevocationMechanics(t *testing.T) {
	tenantID := "ten_alpha"
	subject := "usr_synth_01"
	from := time.Now().Add(-1 * time.Hour)
	to := time.Now().Add(24 * time.Hour)
	now := time.Now()

	asn, _ := NewScopedAssignment("asn_01", tenantID, subject, RoleProjectManager, ScopeGrant{TenantID: tenantID, ProjectID: "prj_01"}, from, to, "usr_admin")

	// Revoke assignment
	revoked, audit, err := asn.Revoke("usr_compliance_lead", "Security investigation revocation", now)
	if err != nil {
		t.Fatalf("unexpected Revoke error: %v", err)
	}

	if revoked.IsActive() || revoked.State() != AssignmentStateRevoked {
		t.Errorf("expected revoked state, got %v", revoked.State())
	}
	if revoked.IsValidAt(now) {
		t.Errorf("revoked assignment must not be valid")
	}
	if revoked.RevokedBy() != "usr_compliance_lead" || revoked.RevocationReason() != "Security investigation revocation" {
		t.Errorf("revocation attribution mismatch: by=%s, reason=%s", revoked.RevokedBy(), revoked.RevocationReason())
	}

	// Verify audit record
	if audit.Transition != "ASSIGNMENT_REVOKED" || audit.ActorSubject != "usr_compliance_lead" {
		t.Errorf("audit record mismatch: %+v", audit)
	}

	// Re-revocation fails closed
	_, _, err = revoked.Revoke("usr_admin", "Duplicate", now)
	if !errors.Is(err, ErrAssignmentRevoked) {
		t.Errorf("expected ErrAssignmentRevoked on re-revocation, got %v", err)
	}
}

func TestScopedAssignment_RoleConflictDetection(t *testing.T) {
	tenantID := "ten_alpha"
	subject := "usr_synth_01"
	from := time.Now().Add(-1 * time.Hour)
	to := time.Now().Add(48 * time.Hour)
	now := time.Now()

	scopeProjectA := ScopeGrant{TenantID: tenantID, ProjectID: "prj_alpha"}
	scopeProjectB := ScopeGrant{TenantID: tenantID, ProjectID: "prj_bravo"}

	// Existing assignment: Inspector on Project Alpha
	existingAsn, _ := NewScopedAssignment("asn_ex_01", tenantID, subject, RoleInspector, scopeProjectA, from, to, "usr_admin")
	existingList := []ScopedAssignment{existingAsn}

	// 1. Conflict: Assigning Auditor on same project scope (Inspector vs Auditor)
	candAuditorSameScope, _ := NewScopedAssignment("asn_cand_01", tenantID, subject, RoleAuditor, scopeProjectA, from, to, "usr_admin")
	err := CheckRoleConflict(existingList, candAuditorSameScope, now)
	if !errors.Is(err, ErrRoleConflictDetected) {
		t.Errorf("expected ErrRoleConflictDetected for Inspector + Auditor on same scope, got %v", err)
	}

	// 2. Conflict: Duplicate active role on overlapping scope
	candDuplicateRole, _ := NewScopedAssignment("asn_cand_02", tenantID, subject, RoleInspector, scopeProjectA, from, to, "usr_admin")
	err = CheckRoleConflict(existingList, candDuplicateRole, now)
	if !errors.Is(err, ErrRoleConflictDetected) {
		t.Errorf("expected ErrRoleConflictDetected for duplicate role on same scope, got %v", err)
	}

	// 3. No conflict: Auditor on DIFFERENT non-overlapping project scope (Project Bravo)
	candAuditorDiffScope, _ := NewScopedAssignment("asn_cand_03", tenantID, subject, RoleAuditor, scopeProjectB, from, to, "usr_admin")
	err = CheckRoleConflict(existingList, candAuditorDiffScope, now)
	if err != nil {
		t.Errorf("expected no conflict for Auditor on distinct project scope, got %v", err)
	}

	// 4. Conflict: PM vs Auditor on same project
	pmAsn, _ := NewScopedAssignment("asn_pm", tenantID, subject, RoleProjectManager, scopeProjectA, from, to, "usr_admin")
	pmList := []ScopedAssignment{pmAsn}
	err = CheckRoleConflict(pmList, candAuditorSameScope, now)
	if !errors.Is(err, ErrRoleConflictDetected) {
		t.Errorf("expected ErrRoleConflictDetected for PM + Auditor on same scope, got %v", err)
	}

	// 5. Conflict: Contractor cannot hold TenantAdmin or PM
	contractorAsn, _ := NewScopedAssignment("asn_con", tenantID, subject, RoleContractor, scopeProjectA, from, to, "usr_admin")
	conList := []ScopedAssignment{contractorAsn}
	candAdmin, _ := NewScopedAssignment("asn_admin", tenantID, subject, RoleTenantAdmin, scopeProjectA, from, to, "usr_admin")
	err = CheckRoleConflict(conList, candAdmin, now)
	if !errors.Is(err, ErrRoleConflictDetected) {
		t.Errorf("expected ErrRoleConflictDetected for Contractor + TenantAdmin, got %v", err)
	}
}

func TestScopedAssignment_RegistryAndAuditLedger(t *testing.T) {
	ledger := NewAssignmentLedger()
	registry := NewScopedAssignmentRegistry(ledger)

	tenantID := "ten_alpha"
	subject := "usr_synth_somchai"
	from := time.Now().Add(-1 * time.Hour)
	to := time.Now().Add(48 * time.Hour)
	now := time.Now()

	scope1 := ScopeGrant{TenantID: tenantID, ProjectID: "prj_01"}
	scope2 := ScopeGrant{TenantID: tenantID, ProjectID: "prj_02"}

	asn1, _ := NewScopedAssignment("asn_reg_01", tenantID, subject, RoleInspector, scope1, from, to, "usr_lead_01")
	asn2, _ := NewScopedAssignment("asn_reg_02", tenantID, subject, RoleProjectManager, scope2, from, to, "usr_lead_02")

	// 1. Register assignments
	if err := registry.RegisterAssignment(asn1, "usr_lead_01", "Assigned for Phase 1", now); err != nil {
		t.Fatalf("RegisterAssignment asn1 failed: %v", err)
	}
	if err := registry.RegisterAssignment(asn2, "usr_lead_02", "Assigned for Phase 2", now); err != nil {
		t.Fatalf("RegisterAssignment asn2 failed: %v", err)
	}

	// Duplicate assignment ID rejected
	if err := registry.RegisterAssignment(asn1, "usr_lead_01", "Duplicate", now); !errors.Is(err, ErrDuplicateAssignmentID) {
		t.Errorf("expected ErrDuplicateAssignmentID, got %v", err)
	}

	// 2. ListActiveAssignments
	activeList, err := registry.ListActiveAssignments(tenantID, subject, now)
	if err != nil || len(activeList) != 2 {
		t.Errorf("expected 2 active assignments, got %d (err: %v)", len(activeList), err)
	}

	// 3. ListAssignmentsByScope
	scopeQuery := ScopeGrant{TenantID: tenantID, ProjectID: "prj_01"}
	scopedList, err := registry.ListAssignmentsByScope(tenantID, scopeQuery, now)
	if err != nil || len(scopedList) != 1 || scopedList[0].AssignmentID() != "asn_reg_01" {
		t.Errorf("expected only asn_reg_01 in scope query, got %v", scopedList)
	}

	// 4. Revoke assignment
	revoked, err := registry.RevokeAssignment(tenantID, "asn_reg_01", "usr_admin", "Safety re-alignment", now)
	if err != nil || revoked.State() != AssignmentStateRevoked {
		t.Fatalf("RevokeAssignment failed: %v", err)
	}

	// Active list now returns 1
	activeAfterRevoke, _ := registry.ListActiveAssignments(tenantID, subject, now)
	if len(activeAfterRevoke) != 1 || activeAfterRevoke[0].AssignmentID() != "asn_reg_02" {
		t.Errorf("expected only asn_reg_02 active after revoke, got %d", len(activeAfterRevoke))
	}

	// 5. Audit trail verification: asn_reg_01 has 2 entries (CREATE and REVOKE)
	trail, err := ledger.GetAssignmentAuditTrail(tenantID, "asn_reg_01")
	if err != nil || len(trail) != 2 {
		t.Fatalf("expected 2 audit records for asn_reg_01, got %d (err: %v)", len(trail), err)
	}
	if trail[0].Transition != "ASSIGNMENT_CREATED" || trail[1].Transition != "ASSIGNMENT_REVOKED" {
		t.Errorf("audit transitions mismatch: %+v", trail)
	}

	// 6. Cross-tenant isolation verification
	foreignTrail, err := ledger.GetAssignmentAuditTrail("ten_other", "asn_reg_01")
	if err != nil || len(foreignTrail) != 0 {
		t.Errorf("cross-tenant leakage: other tenant retrieved audit trail")
	}
}

func TestScopedAssignment_EvaluateScopedAccess_EndToEnd(t *testing.T) {
	evaluator := NewPolicyEvaluator()
	registry := NewScopedAssignmentRegistry(nil)

	tenantID := "ten_alpha"
	subject := "usr_synth_operator"

	// Set active membership
	evaluator.SetMembership(tenantID, subject, MembershipActive)

	from := time.Now().Add(-1 * time.Hour)
	to := time.Now().Add(24 * time.Hour)
	now := time.Now()

	// Assign Inspector to prj_plant
	asn, _ := NewScopedAssignment("asn_eval_01", tenantID, subject, RoleInspector, ScopeGrant{TenantID: tenantID, ProjectID: "prj_plant"}, from, to, "usr_admin")
	_ = registry.RegisterAssignment(asn, "usr_admin", "Initial", now)

	// Target 1: READ in prj_plant -> should be ALLOWED (Inspector permits Read)
	req1 := AccessRequest{
		Identity: SubjectIdentity{Subject: subject, TenantID: tenantID, IsAuthenticated: true},
		Target:   TargetResource{TenantID: tenantID, ProjectID: "prj_plant", Lifecycle: ResourceActive},
		Action:   ActionRead,
	}
	res1 := EvaluateScopedAccess(registry, evaluator, req1, now)
	if !res1.Allowed {
		t.Errorf("expected access allowed for Inspector Read on prj_plant, got denial: %s", res1.DenialReason)
	}

	// Target 2: CREATE in prj_plant -> should be ALLOWED (Inspector permits Create)
	req2 := AccessRequest{
		Identity: SubjectIdentity{Subject: subject, TenantID: tenantID, IsAuthenticated: true},
		Target:   TargetResource{TenantID: tenantID, ProjectID: "prj_plant", Lifecycle: ResourceActive},
		Action:   ActionCreate,
	}
	res2 := EvaluateScopedAccess(registry, evaluator, req2, now)
	if !res2.Allowed {
		t.Errorf("expected access allowed for Inspector Create on prj_plant, got denial: %s", res2.DenialReason)
	}

	// Target 3: DELETE in prj_plant -> should be DENIED with DenialPrivilegeEscalation (Inspector cannot Delete)
	req3 := AccessRequest{
		Identity: SubjectIdentity{Subject: subject, TenantID: tenantID, IsAuthenticated: true},
		Target:   TargetResource{TenantID: tenantID, ProjectID: "prj_plant", Lifecycle: ResourceActive},
		Action:   ActionDelete,
	}
	res3 := EvaluateScopedAccess(registry, evaluator, req3, now)
	if res3.Allowed || res3.DenialReason != DenialPrivilegeEscalation {
		t.Errorf("expected DenialPrivilegeEscalation for Inspector Delete, got allowed=%v reason=%s", res3.Allowed, res3.DenialReason)
	}

	// Target 4: READ in unrelated prj_other -> should be DENIED with DenialScopeMismatch
	req4 := AccessRequest{
		Identity: SubjectIdentity{Subject: subject, TenantID: tenantID, IsAuthenticated: true},
		Target:   TargetResource{TenantID: tenantID, ProjectID: "prj_other", Lifecycle: ResourceActive},
		Action:   ActionRead,
	}
	res4 := EvaluateScopedAccess(registry, evaluator, req4, now)
	if res4.Allowed || res4.DenialReason != DenialScopeMismatch {
		t.Errorf("expected DenialScopeMismatch for unrelated project, got allowed=%v reason=%s", res4.Allowed, res4.DenialReason)
	}

	// Target 5: After revocation -> should be DENIED with DenialRoleNotGranted
	_, _ = registry.RevokeAssignment(tenantID, "asn_eval_01", "usr_admin", "Revoked", now)
	res5 := EvaluateScopedAccess(registry, evaluator, req1, now)
	if res5.Allowed || res5.DenialReason != DenialRoleNotGranted {
		t.Errorf("expected DenialRoleNotGranted after revocation, got allowed=%v reason=%s", res5.Allowed, res5.DenialReason)
	}
}
