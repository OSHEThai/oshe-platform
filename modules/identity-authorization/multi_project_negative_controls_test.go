package localidentity_test

import (
	"errors"
	"testing"
	"time"

	localidentity "github.com/oshethai/oshe-platform/modules/identity-authorization"
)

// NEG-V030-10: Cross-Project Unauthorized Access Denial (H030-003, H030-005 / Issue #96)
// Threat: Horizontal privilege escalation across projects within a tenant.
// Test Scenario & Hostile Input: An authenticated worker holding active participation exclusively on prj_alpha
// submits read/write access requests targeting sibling projects prj_beta and prj_gamma.
// Expected Behavior: Immediate failure closed with DenialScopeMismatch; zero resource metadata or existence is leaked.
func TestNegativeControl_NEG_V030_10_CrossProjectAccessDenial(t *testing.T) {
	ledger := localidentity.NewAttributionLedger()
	registry := localidentity.NewMultiProjectParticipationRegistry(ledger)

	tenantID := "ten_neg_alpha"
	subject := "usr_hostile_worker"
	t0 := time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC)
	t1 := t0.Add(7 * 24 * time.Hour)
	now := t0.Add(1 * time.Hour)

	// Register participation strictly on prj_alpha
	scopeAlpha := localidentity.ScopeGrant{TenantID: tenantID, ProjectID: "prj_alpha"}
	partAlpha, err := localidentity.NewProjectParticipation(
		"part_neg_01", tenantID, subject, "prj_alpha",
		localidentity.RoleInspector, scopeAlpha, "usr_admin_lead",
		t0, t1, false,
	)
	if err != nil {
		t.Fatalf("failed to create participation fixture: %v", err)
	}
	_ = registry.AssignParticipation(partAlpha)

	// 1. Hostile probe: Access sibling project prj_beta
	reqBeta := localidentity.AccessRequest{
		Identity: localidentity.SubjectIdentity{Subject: subject, TenantID: tenantID, IsAuthenticated: true},
		Target:   localidentity.TargetResource{TenantID: tenantID, ProjectID: "prj_beta", Lifecycle: localidentity.ResourceActive},
		Action:   localidentity.ActionRead,
	}
	resBeta := registry.EvaluateMultiProjectAccess(reqBeta, now)
	if resBeta.Allowed {
		t.Fatalf("CRITICAL SECURITY FAILURE (NEG-V030-10): subject on prj_alpha accessed sibling prj_beta")
	}
	if resBeta.DenialReason != localidentity.DenialScopeMismatch {
		t.Errorf("expected DenialScopeMismatch for sibling project probe, got: %s", resBeta.DenialReason)
	}

	// 2. Hostile probe: Access unassigned project prj_gamma with write action
	reqGamma := localidentity.AccessRequest{
		Identity: localidentity.SubjectIdentity{Subject: subject, TenantID: tenantID, IsAuthenticated: true},
		Target:   localidentity.TargetResource{TenantID: tenantID, ProjectID: "prj_gamma", Lifecycle: localidentity.ResourceActive},
		Action:   localidentity.ActionCreate,
	}
	resGamma := registry.EvaluateMultiProjectAccess(reqGamma, now)
	if resGamma.Allowed {
		t.Fatalf("CRITICAL SECURITY FAILURE (NEG-V030-10): subject on prj_alpha created resource on prj_gamma")
	}
	if resGamma.DenialReason != localidentity.DenialScopeMismatch {
		t.Errorf("expected DenialScopeMismatch for unassigned project write, got: %s", resGamma.DenialReason)
	}
}

// NEG-V030-11: Bounded Contractor Administration Prohibition (H030-003, H030-004 / Issue #96)
// Threat: External contractor elevating to internal administrative or project management authority.
// Test Scenario & Hostile Input: Attempt to grant contractor RoleTenantAdmin or RoleProjectManager,
// or contractor attempting administrative actions (Delete, PermOrgTenantManage, PermInspectionApprove).
// Expected Behavior: Fails closed immediately with ErrContractorAdminProhibited.
func TestNegativeControl_NEG_V030_11_ContractorAdminProhibition(t *testing.T) {
	tenantID := "ten_neg_contract"
	contractorSubject := "usr_hostile_contractor"
	t0 := time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC)
	t1 := t0.Add(7 * 24 * time.Hour)
	scope := localidentity.ScopeGrant{TenantID: tenantID, ProjectID: "prj_contract"}

	// 1. Hostile input: Attempt to register contractor as TenantAdmin
	_, err := localidentity.NewProjectParticipation(
		"part_bad_admin", tenantID, contractorSubject, "prj_contract",
		localidentity.RoleTenantAdmin, scope, "usr_admin_lead",
		t0, t1, true,
	)
	if !errors.Is(err, localidentity.ErrContractorAdminProhibited) {
		t.Fatalf("expected ErrContractorAdminProhibited when assigning TenantAdmin to contractor, got: %v", err)
	}

	// 2. Hostile input: Attempt to register contractor as ProjectManager
	_, err = localidentity.NewProjectParticipation(
		"part_bad_pm", tenantID, contractorSubject, "prj_contract",
		localidentity.RoleProjectManager, scope, "usr_admin_lead",
		t0, t1, true,
	)
	if !errors.Is(err, localidentity.ErrContractorAdminProhibited) {
		t.Fatalf("expected ErrContractorAdminProhibited when assigning ProjectManager to contractor, got: %v", err)
	}

	// 3. Hostile input: Contractor attempting administrative permissions
	adminPerms := []localidentity.Permission{
		localidentity.PermOrgTenantManage,
		localidentity.PermOrgProjectManage,
		localidentity.PermIdentityUserManage,
		localidentity.PermIdentityRoleAssign,
		localidentity.PermInspectionApprove,
		localidentity.PermAuditExport,
		localidentity.PermLegalHoldManage,
		localidentity.PermPortalSnapshotPublish,
		localidentity.PermDelegationGrant,
	}

	for _, perm := range adminPerms {
		err := localidentity.AssertContractorAdminBounds(localidentity.RoleContractor, true, localidentity.ActionRead, perm)
		if !errors.Is(err, localidentity.ErrContractorAdminProhibited) {
			t.Errorf("expected ErrContractorAdminProhibited for contractor attempting %s, got: %v", perm, err)
		}
	}

	// 4. Hostile input: Contractor attempting resource deletion
	err = localidentity.AssertContractorAdminBounds(localidentity.RoleContractor, true, localidentity.ActionDelete, "")
	if !errors.Is(err, localidentity.ErrContractorAdminProhibited) {
		t.Errorf("expected ErrContractorAdminProhibited for contractor attempting ActionDelete, got: %v", err)
	}
}

// NEG-V030-12: Auditor Mutating Operation Denial (H030-003, H030-005 / Issue #96)
// Threat: Compliance auditor attempting to alter, create, or delete operational safety records.
// Test Scenario & Hostile Input: Auditor submits ActionCreate, ActionUpdate, or ActionDelete requests.
// Expected Behavior: Fails closed with ErrAuditorReadOnlyViolation and DenialPrivilegeEscalation.
func TestNegativeControl_NEG_V030_12_AuditorReadOnlyViolation(t *testing.T) {
	ledger := localidentity.NewAttributionLedger()
	registry := localidentity.NewMultiProjectParticipationRegistry(ledger)

	tenantID := "ten_neg_auditor"
	auditorSubject := "usr_hostile_auditor"
	projectID := "prj_audit_target"
	t0 := time.Date(2026, 9, 5, 11, 0, 0, 0, time.UTC)
	t1 := t0.Add(30 * 24 * time.Hour)
	now := t0.Add(1 * time.Hour)

	part, err := localidentity.NewProjectParticipation(
		"part_aud_01", tenantID, auditorSubject, projectID,
		localidentity.RoleAuditor, localidentity.ScopeGrant{TenantID: tenantID, ProjectID: projectID},
		"usr_admin_lead", t0, t1, false,
	)
	if err != nil {
		t.Fatalf("failed to create auditor participation: %v", err)
	}
	_ = registry.AssignParticipation(part)

	mutatingActions := []localidentity.Action{
		localidentity.ActionCreate,
		localidentity.ActionUpdate,
		localidentity.ActionDelete,
	}

	for _, act := range mutatingActions {
		// 1. AssertAuditorReadOnly helper rejects mutating action
		err := localidentity.AssertAuditorReadOnly(localidentity.RoleAuditor, act, "")
		if !errors.Is(err, localidentity.ErrAuditorReadOnlyViolation) {
			t.Errorf("expected ErrAuditorReadOnlyViolation for action %s, got: %v", act, err)
		}

		// 2. Multi-project access evaluation rejects mutating action
		req := localidentity.AccessRequest{
			Identity: localidentity.SubjectIdentity{Subject: auditorSubject, TenantID: tenantID, IsAuthenticated: true},
			Target:   localidentity.TargetResource{TenantID: tenantID, ProjectID: projectID, Lifecycle: localidentity.ResourceActive},
			Action:   act,
		}
		res := registry.EvaluateMultiProjectAccess(req, now)
		if res.Allowed {
			t.Fatalf("CRITICAL SECURITY FAILURE (NEG-V030-12): auditor permitted %s on %s", act, projectID)
		}
		if res.DenialReason != localidentity.DenialPrivilegeEscalation {
			t.Errorf("expected DenialPrivilegeEscalation for auditor %s, got: %s", act, res.DenialReason)
		}
	}
}

// NEG-V030-13: Deactivated Participant Operational Access Denial (H030-004, H030-005 / Issue #96)
// Threat: Former worker attempting operational access after deactivation.
// Test Scenario & Hostile Input: Worker active in project is deactivated; worker then submits access requests.
// Expected Behavior: Access fails closed with DenialInactiveMembership; past attribution records remain intact.
func TestNegativeControl_NEG_V030_13_DeactivatedParticipantOperationDenial(t *testing.T) {
	ledger := localidentity.NewAttributionLedger()
	registry := localidentity.NewMultiProjectParticipationRegistry(ledger)

	tenantID := "ten_neg_deact"
	subject := "usr_departed_worker"
	projectID := "prj_deact_target"
	t0 := time.Date(2026, 9, 5, 11, 0, 0, 0, time.UTC)
	t1 := t0.Add(14 * 24 * time.Hour)

	part, _ := localidentity.NewProjectParticipation(
		"part_deact_01", tenantID, subject, projectID,
		localidentity.RoleInspector, localidentity.ScopeGrant{TenantID: tenantID, ProjectID: projectID},
		"usr_admin_lead", t0, t1, false,
	)
	_ = registry.AssignParticipation(part)

	// Record legitimate action before departure
	_ = registry.RecordAttribution(localidentity.HistoricalAttributionRecord{
		RecordID:    "attr_prior_01",
		TenantID:    tenantID,
		ProjectID:   projectID,
		Subject:     subject,
		DisplayName: "Departed Worker",
		RoleAtEvent: localidentity.RoleInspector,
		ActionType:  "INSPECTION_SUBMITTED",
		ResourceID:  "insp_historic_99",
		RecordedAt:  t0.Add(1 * time.Hour),
	})

	// Deactivate participation
	tDeact := t0.Add(5 * time.Hour)
	_, err := registry.DeactivateParticipation(tenantID, subject, projectID, "usr_admin_lead", "Staff turnover", tDeact)
	if err != nil {
		t.Fatalf("unexpected error deactivating participation: %v", err)
	}

	// Hostile input: Deactivated worker attempts access
	req := localidentity.AccessRequest{
		Identity: localidentity.SubjectIdentity{Subject: subject, TenantID: tenantID, IsAuthenticated: true},
		Target:   localidentity.TargetResource{TenantID: tenantID, ProjectID: projectID, Lifecycle: localidentity.ResourceActive},
		Action:   localidentity.ActionRead,
	}
	res := registry.EvaluateMultiProjectAccess(req, tDeact.Add(1*time.Hour))
	if res.Allowed {
		t.Fatalf("CRITICAL SECURITY FAILURE (NEG-V030-13): deactivated participant permitted access")
	}
	if res.DenialReason != localidentity.DenialInactiveMembership {
		t.Errorf("expected DenialInactiveMembership for deactivated participant, got: %s", res.DenialReason)
	}

	// Invariant: Historical record remains queryable and uncorrupted
	history, err := registry.GetAttributionTrail(tenantID, projectID, "insp_historic_99")
	if err != nil || len(history) != 1 {
		t.Fatalf("historical record lost after deactivation: err=%v len=%d", err, len(history))
	}
	if history[0].Subject != subject || history[0].DisplayName != "Departed Worker" {
		t.Errorf("historical record altered after deactivation: %+v", history[0])
	}
}

// NEG-V030-14: Attribution Ledger Tampering Denial (H030-003 / Issue #96)
// Threat: Revisionism, overwriting, or deletion of past safety findings and inspection logs.
// Test Scenario & Hostile Input: Attempting to overwrite existing attribution record with duplicate record ID.
// Expected Behavior: Rejection with ErrAttributionImmutable; zero hard deletion capability.
func TestNegativeControl_NEG_V030_14_AttributionTamperingDenial(t *testing.T) {
	ledger := localidentity.NewAttributionLedger()
	t0 := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)

	originalRec := localidentity.HistoricalAttributionRecord{
		RecordID:    "attr_critical_01",
		TenantID:    "ten_alpha",
		ProjectID:   "prj_site_1",
		Subject:     "usr_whistleblower",
		DisplayName: "Safety Whistleblower",
		RoleAtEvent: localidentity.RoleInspector,
		ActionType:  "HAZARD_FLAGGED",
		ResourceID:  "fnd_critical_100",
		Details:     map[string]string{"risk": "IMMINENT_DANGER"},
		RecordedAt:  t0,
	}
	if err := ledger.RecordAttribution(originalRec); err != nil {
		t.Fatalf("failed to record initial attribution: %v", err)
	}

	// Hostile input: Attempt to overwrite record with modified finding details and altered subject
	tamperedRec := originalRec
	tamperedRec.Subject = "usr_different_actor"
	tamperedRec.DisplayName = "Altered Name"
	tamperedRec.Details = map[string]string{"risk": "NO_RISK"}

	err := ledger.RecordAttribution(tamperedRec)
	if !errors.Is(err, localidentity.ErrAttributionImmutable) {
		t.Fatalf("CRITICAL INTEGRITY FAILURE (NEG-V030-14): ledger permitted overwrite of historical record, got: %v", err)
	}

	// Verify original record is unchanged
	trail, err := ledger.GetAttributionTrail("ten_alpha", "prj_site_1", "fnd_critical_100")
	if err != nil || len(trail) != 1 {
		t.Fatalf("failed to retrieve original attribution trail: %v", err)
	}
	if trail[0].Subject != "usr_whistleblower" || trail[0].Details["risk"] != "IMMINENT_DANGER" {
		t.Errorf("attribution record was corrupted: %+v", trail[0])
	}
}
