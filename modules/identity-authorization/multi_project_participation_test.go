package localidentity_test

import (
	"errors"
	"testing"
	"time"

	localidentity "github.com/oshethai/oshe-platform/modules/identity-authorization"
)

func TestMultiProject_OneUserTwoProjectsParticipation(t *testing.T) {
	ledger := localidentity.NewAttributionLedger()
	registry := localidentity.NewMultiProjectParticipationRegistry(ledger)

	tenantID := "ten_synth_alpha"
	subject := "usr_synth_worker_01"
	t0 := time.Date(2026, 9, 5, 8, 0, 0, 0, time.UTC)
	t1 := t0.Add(30 * 24 * time.Hour)
	now := t0.Add(2 * time.Hour)

	// 1. Assign worker to prj_alpha as RoleInspector
	scopeAlpha := localidentity.ScopeGrant{TenantID: tenantID, ProjectID: "prj_alpha"}
	partAlpha, err := localidentity.NewProjectParticipation(
		"part_alpha_01", tenantID, subject, "prj_alpha",
		localidentity.RoleInspector, scopeAlpha, "usr_admin_lead",
		t0, t1, false,
	)
	if err != nil {
		t.Fatalf("unexpected NewProjectParticipation error for prj_alpha: %v", err)
	}
	if err := registry.AssignParticipation(partAlpha); err != nil {
		t.Fatalf("failed to assign participation in prj_alpha: %v", err)
	}

	// 2. Assign same worker to prj_beta as RoleContractor
	scopeBeta := localidentity.ScopeGrant{TenantID: tenantID, ProjectID: "prj_beta"}
	partBeta, err := localidentity.NewProjectParticipation(
		"part_beta_01", tenantID, subject, "prj_beta",
		localidentity.RoleContractor, scopeBeta, "usr_admin_lead",
		t0, t1, true,
	)
	if err != nil {
		t.Fatalf("unexpected NewProjectParticipation error for prj_beta: %v", err)
	}
	if err := registry.AssignParticipation(partBeta); err != nil {
		t.Fatalf("failed to assign participation in prj_beta: %v", err)
	}

	// 3. Verify multiple project listings for subject
	parts, err := registry.ListActiveParticipationsBySubject(tenantID, subject, now)
	if err != nil {
		t.Fatalf("unexpected ListActiveParticipationsBySubject error: %v", err)
	}
	if len(parts) != 2 {
		t.Fatalf("expected 2 active project participations for worker, got %d", len(parts))
	}

	// 4. Access Evaluation on prj_alpha (Inspector role)
	reqAlphaRead := localidentity.AccessRequest{
		Identity: localidentity.SubjectIdentity{Subject: subject, TenantID: tenantID, IsAuthenticated: true},
		Target:   localidentity.TargetResource{TenantID: tenantID, ProjectID: "prj_alpha", Lifecycle: localidentity.ResourceActive},
		Action:   localidentity.ActionRead,
	}
	resAlphaRead := registry.EvaluateMultiProjectAccess(reqAlphaRead, now)
	if !resAlphaRead.Allowed {
		t.Errorf("expected read permitted on prj_alpha, got denied: %s", resAlphaRead.DenialReason)
	}

	reqAlphaCreate := reqAlphaRead
	reqAlphaCreate.Action = localidentity.ActionCreate
	resAlphaCreate := registry.EvaluateMultiProjectAccess(reqAlphaCreate, now)
	if !resAlphaCreate.Allowed {
		t.Errorf("expected create permitted for Inspector on prj_alpha, got denied: %s", resAlphaCreate.DenialReason)
	}

	reqAlphaDelete := reqAlphaRead
	reqAlphaDelete.Action = localidentity.ActionDelete
	resAlphaDelete := registry.EvaluateMultiProjectAccess(reqAlphaDelete, now)
	if resAlphaDelete.Allowed || resAlphaDelete.DenialReason != localidentity.DenialPrivilegeEscalation {
		t.Errorf("expected delete denied (DenialPrivilegeEscalation) for Inspector on prj_alpha, got: allowed=%v reason=%s", resAlphaDelete.Allowed, resAlphaDelete.DenialReason)
	}

	// 5. Access Evaluation on prj_beta (Contractor role)
	reqBetaRead := localidentity.AccessRequest{
		Identity: localidentity.SubjectIdentity{Subject: subject, TenantID: tenantID, IsAuthenticated: true},
		Target:   localidentity.TargetResource{TenantID: tenantID, ProjectID: "prj_beta", Lifecycle: localidentity.ResourceActive},
		Action:   localidentity.ActionRead,
	}
	resBetaRead := registry.EvaluateMultiProjectAccess(reqBetaRead, now)
	if !resBetaRead.Allowed {
		t.Errorf("expected read permitted on prj_beta, got denied: %s", resBetaRead.DenialReason)
	}

	reqBetaUpdate := reqBetaRead
	reqBetaUpdate.Action = localidentity.ActionUpdate
	resBetaUpdate := registry.EvaluateMultiProjectAccess(reqBetaUpdate, now)
	if resBetaUpdate.Allowed || resBetaUpdate.DenialReason != localidentity.DenialPrivilegeEscalation {
		t.Errorf("expected update denied (DenialPrivilegeEscalation) for Contractor on prj_beta, got: allowed=%v reason=%s", resBetaUpdate.Allowed, resBetaUpdate.DenialReason)
	}

	reqBetaDelete := reqBetaRead
	reqBetaDelete.Action = localidentity.ActionDelete
	resBetaDelete := registry.EvaluateMultiProjectAccess(reqBetaDelete, now)
	if resBetaDelete.Allowed || resBetaDelete.DenialReason != localidentity.DenialPrivilegeEscalation {
		t.Errorf("expected delete denied (DenialPrivilegeEscalation) for Contractor on prj_beta, got: allowed=%v reason=%s", resBetaDelete.Allowed, resBetaDelete.DenialReason)
	}

	// 6. Cross-project Access Evaluation on unassigned prj_gamma
	reqGammaRead := localidentity.AccessRequest{
		Identity: localidentity.SubjectIdentity{Subject: subject, TenantID: tenantID, IsAuthenticated: true},
		Target:   localidentity.TargetResource{TenantID: tenantID, ProjectID: "prj_gamma", Lifecycle: localidentity.ResourceActive},
		Action:   localidentity.ActionRead,
	}
	resGammaRead := registry.EvaluateMultiProjectAccess(reqGammaRead, now)
	if resGammaRead.Allowed || resGammaRead.DenialReason != localidentity.DenialScopeMismatch {
		t.Errorf("expected cross-project access to prj_gamma denied with DenialScopeMismatch, got allowed=%v reason=%s", resGammaRead.Allowed, resGammaRead.DenialReason)
	}
}

func TestMultiProject_ContractorAdministrationBounds(t *testing.T) {
	tenantID := "ten_synth_alpha"
	subject := "usr_synth_contractor_01"
	t0 := time.Date(2026, 9, 5, 8, 0, 0, 0, time.UTC)
	t1 := t0.Add(14 * 24 * time.Hour)
	scope := localidentity.ScopeGrant{TenantID: tenantID, ProjectID: "prj_contract"}

	// 1. Attempting to assign TenantAdmin to contractor fails
	_, err := localidentity.NewProjectParticipation(
		"part_bad_admin", tenantID, subject, "prj_contract",
		localidentity.RoleTenantAdmin, scope, "usr_admin_lead",
		t0, t1, true,
	)
	if !errors.Is(err, localidentity.ErrContractorAdminProhibited) {
		t.Fatalf("expected ErrContractorAdminProhibited when assigning TenantAdmin to contractor, got: %v", err)
	}

	// 2. Attempting to assign ProjectManager to contractor fails
	_, err = localidentity.NewProjectParticipation(
		"part_bad_pm", tenantID, subject, "prj_contract",
		localidentity.RoleProjectManager, scope, "usr_admin_lead",
		t0, t1, true,
	)
	if !errors.Is(err, localidentity.ErrContractorAdminProhibited) {
		t.Fatalf("expected ErrContractorAdminProhibited when assigning ProjectManager to contractor, got: %v", err)
	}

	// 3. AssertContractorAdminBounds rejects administrative actions and roles
	if err := localidentity.AssertContractorAdminBounds(localidentity.RoleContractor, true, localidentity.ActionDelete, ""); !errors.Is(err, localidentity.ErrContractorAdminProhibited) {
		t.Errorf("expected ErrContractorAdminProhibited for contractor delete action, got: %v", err)
	}

	if err := localidentity.AssertContractorAdminBounds(localidentity.RoleContractor, true, localidentity.ActionRead, localidentity.PermOrgTenantManage); !errors.Is(err, localidentity.ErrContractorAdminProhibited) {
		t.Errorf("expected ErrContractorAdminProhibited for PermOrgTenantManage, got: %v", err)
	}

	if err := localidentity.AssertContractorAdminBounds(localidentity.RoleContractor, true, localidentity.ActionRead, localidentity.PermInspectionApprove); !errors.Is(err, localidentity.ErrContractorAdminProhibited) {
		t.Errorf("expected ErrContractorAdminProhibited for PermInspectionApprove, got: %v", err)
	}

	// Permitted contractor action passes
	if err := localidentity.AssertContractorAdminBounds(localidentity.RoleContractor, true, localidentity.ActionCreate, localidentity.PermInspectionCreate); err != nil {
		t.Errorf("expected nil for valid contractor operational action, got: %v", err)
	}
}

func TestMultiProject_AuditorReadOnlyBehavior(t *testing.T) {
	ledger := localidentity.NewAttributionLedger()
	registry := localidentity.NewMultiProjectParticipationRegistry(ledger)

	tenantID := "ten_synth_alpha"
	auditorSubject := "usr_synth_auditor_01"
	t0 := time.Date(2026, 9, 5, 9, 0, 0, 0, time.UTC)
	t1 := t0.Add(30 * 24 * time.Hour)
	now := t0.Add(1 * time.Hour)

	// Assign Auditor across two projects
	partAlpha, _ := localidentity.NewProjectParticipation(
		"part_aud_alpha", tenantID, auditorSubject, "prj_alpha",
		localidentity.RoleAuditor, localidentity.ScopeGrant{TenantID: tenantID, ProjectID: "prj_alpha"},
		"usr_admin_lead", t0, t1, false,
	)
	_ = registry.AssignParticipation(partAlpha)

	partBeta, _ := localidentity.NewProjectParticipation(
		"part_aud_beta", tenantID, auditorSubject, "prj_beta",
		localidentity.RoleAuditor, localidentity.ScopeGrant{TenantID: tenantID, ProjectID: "prj_beta"},
		"usr_admin_lead", t0, t1, false,
	)
	_ = registry.AssignParticipation(partBeta)

	// 1. Read operations succeed across both projects
	for _, prj := range []string{"prj_alpha", "prj_beta"} {
		reqRead := localidentity.AccessRequest{
			Identity: localidentity.SubjectIdentity{Subject: auditorSubject, TenantID: tenantID, IsAuthenticated: true},
			Target:   localidentity.TargetResource{TenantID: tenantID, ProjectID: prj, Lifecycle: localidentity.ResourceActive},
			Action:   localidentity.ActionRead,
		}
		resRead := registry.EvaluateMultiProjectAccess(reqRead, now)
		if !resRead.Allowed {
			t.Errorf("expected read permitted for Auditor on %s, got denied: %s", prj, resRead.DenialReason)
		}
	}

	// 2. Mutating operations fail closed with DenialPrivilegeEscalation
	mutatingActions := []localidentity.Action{
		localidentity.ActionCreate,
		localidentity.ActionUpdate,
		localidentity.ActionDelete,
	}

	for _, act := range mutatingActions {
		reqMutate := localidentity.AccessRequest{
			Identity: localidentity.SubjectIdentity{Subject: auditorSubject, TenantID: tenantID, IsAuthenticated: true},
			Target:   localidentity.TargetResource{TenantID: tenantID, ProjectID: "prj_alpha", Lifecycle: localidentity.ResourceActive},
			Action:   act,
		}
		resMutate := registry.EvaluateMultiProjectAccess(reqMutate, now)
		if resMutate.Allowed || resMutate.DenialReason != localidentity.DenialPrivilegeEscalation {
			t.Errorf("expected %s denied with DenialPrivilegeEscalation for Auditor, got allowed=%v reason=%s", act, resMutate.Allowed, resMutate.DenialReason)
		}
	}

	// 3. AssertAuditorReadOnly helper checks
	for _, act := range mutatingActions {
		if err := localidentity.AssertAuditorReadOnly(localidentity.RoleAuditor, act, ""); !errors.Is(err, localidentity.ErrAuditorReadOnlyViolation) {
			t.Errorf("expected ErrAuditorReadOnlyViolation for action %s, got: %v", act, err)
		}
	}

	mutatingPerms := []localidentity.Permission{
		localidentity.PermInspectionCreate,
		localidentity.PermFindingCreate,
		localidentity.PermRecordArchive,
	}
	for _, perm := range mutatingPerms {
		if err := localidentity.AssertAuditorReadOnly(localidentity.RoleAuditor, localidentity.ActionRead, perm); !errors.Is(err, localidentity.ErrAuditorReadOnlyViolation) {
			t.Errorf("expected ErrAuditorReadOnlyViolation for permission %s, got: %v", perm, err)
		}
	}
}

func TestMultiProject_DeactivatedUserHistoryPreservation(t *testing.T) {
	ledger := localidentity.NewAttributionLedger()
	registry := localidentity.NewMultiProjectParticipationRegistry(ledger)

	tenantID := "ten_synth_alpha"
	subject := "usr_synth_field_worker"
	projectID := "prj_construction_main"
	t0 := time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC)
	t1 := t0.Add(14 * 24 * time.Hour)

	// 1. Assign participation
	scope := localidentity.ScopeGrant{TenantID: tenantID, ProjectID: projectID}
	part, err := localidentity.NewProjectParticipation(
		"part_field_01", tenantID, subject, projectID,
		localidentity.RoleInspector, scope, "usr_admin_lead",
		t0, t1, false,
	)
	if err != nil {
		t.Fatalf("unexpected NewProjectParticipation error: %v", err)
	}
	_ = registry.AssignParticipation(part)

	// 2. Worker performs operational actions during active participation
	tAction1 := t0.Add(2 * time.Hour)
	rec1 := localidentity.HistoricalAttributionRecord{
		RecordID:    "attr_insp_001",
		TenantID:    tenantID,
		ProjectID:   projectID,
		Subject:     subject,
		DisplayName: "Kallaya Sorn",
		RoleAtEvent: localidentity.RoleInspector,
		ActionType:  "INSPECTION_SUBMITTED",
		ResourceID:  "insp_safety_101",
		Details:     map[string]string{"checklist": "CHK-CONF-SPACE", "findings_count": "1"},
		RecordedAt:  tAction1,
	}
	if err := registry.RecordAttribution(rec1); err != nil {
		t.Fatalf("unexpected RecordAttribution error for rec1: %v", err)
	}

	tAction2 := t0.Add(4 * time.Hour)
	rec2 := localidentity.HistoricalAttributionRecord{
		RecordID:    "attr_fnd_002",
		TenantID:    tenantID,
		ProjectID:   projectID,
		Subject:     subject,
		DisplayName: "Kallaya Sorn",
		RoleAtEvent: localidentity.RoleInspector,
		ActionType:  "FINDING_LOGGED",
		ResourceID:  "fnd_hazard_202",
		Details:     map[string]string{"severity": "HIGH", "location": "Area-B"},
		RecordedAt:  tAction2,
	}
	if err := registry.RecordAttribution(rec2); err != nil {
		t.Fatalf("unexpected RecordAttribution error for rec2: %v", err)
	}

	// 3. Deactivate worker participation
	tDeact := t0.Add(24 * time.Hour)
	deactPart, err := registry.DeactivateParticipation(tenantID, subject, projectID, "usr_admin_lead", "Contract assignment completed", tDeact)
	if err != nil {
		t.Fatalf("unexpected DeactivateParticipation error: %v", err)
	}
	if deactPart.IsActive() {
		t.Errorf("expected deactivated participation to have IsActive == false")
	}
	if deactPart.Status() != localidentity.ParticipationDeactivated {
		t.Errorf("expected status DEACTIVATED, got %s", deactPart.Status())
	}

	// 4. Subsequent operational access fails closed
	tPostDeact := tDeact.Add(1 * time.Hour)
	reqPostDeact := localidentity.AccessRequest{
		Identity: localidentity.SubjectIdentity{Subject: subject, TenantID: tenantID, IsAuthenticated: true},
		Target:   localidentity.TargetResource{TenantID: tenantID, ProjectID: projectID, Lifecycle: localidentity.ResourceActive},
		Action:   localidentity.ActionRead,
	}
	resPostDeact := registry.EvaluateMultiProjectAccess(reqPostDeact, tPostDeact)
	if resPostDeact.Allowed || resPostDeact.DenialReason != localidentity.DenialInactiveMembership {
		t.Fatalf("expected deactivated user access to fail closed with DenialInactiveMembership, got allowed=%v reason=%s", resPostDeact.Allowed, resPostDeact.DenialReason)
	}

	// 5. Historical attribution records remain fully preserved and queryable
	trailInsp, err := registry.GetAttributionTrail(tenantID, projectID, "insp_safety_101")
	if err != nil {
		t.Fatalf("unexpected GetAttributionTrail error: %v", err)
	}
	if len(trailInsp) != 1 {
		t.Fatalf("expected 1 attribution record for inspection, got %d", len(trailInsp))
	}
	if trailInsp[0].Subject != subject || trailInsp[0].DisplayName != "Kallaya Sorn" || trailInsp[0].RoleAtEvent != localidentity.RoleInspector {
		t.Errorf("attribution record corrupted after deactivation: %+v", trailInsp[0])
	}

	// 6. Query entire subject attribution history
	subHistory, err := registry.GetSubjectAttributionHistory(tenantID, subject)
	if err != nil {
		t.Fatalf("unexpected GetSubjectAttributionHistory error: %v", err)
	}
	if len(subHistory) != 2 {
		t.Fatalf("expected 2 historical records in subject history, got %d", len(subHistory))
	}
}

func TestMultiProject_TenantBoundaryIsolation(t *testing.T) {
	ledger := localidentity.NewAttributionLedger()
	registry := localidentity.NewMultiProjectParticipationRegistry(ledger)

	tenantA := "ten_alpha"
	tenantB := "ten_beta"
	subjectA := "usr_synth_worker_a"
	t0 := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)

	// Record in Tenant A
	recA := localidentity.HistoricalAttributionRecord{
		RecordID:    "attr_a_01",
		TenantID:    tenantA,
		ProjectID:   "prj_a",
		Subject:     subjectA,
		DisplayName: "Worker Alpha",
		RoleAtEvent: localidentity.RoleInspector,
		ActionType:  "INSPECTION_SUBMITTED",
		ResourceID:  "res_a_100",
		RecordedAt:  t0,
	}
	_ = registry.RecordAttribution(recA)

	// 1. Tenant B querying Tenant A's trail returns empty slice
	trailB, err := registry.GetAttributionTrail(tenantB, "prj_a", "res_a_100")
	if err != nil {
		t.Fatalf("unexpected error querying attribution trail: %v", err)
	}
	if len(trailB) != 0 {
		t.Fatalf("cross-tenant attribution leakage: Tenant B received records: %+v", trailB)
	}

	// 2. Tenant B querying Tenant A's subject history returns empty slice
	subHistB, err := registry.GetSubjectAttributionHistory(tenantB, subjectA)
	if err != nil {
		t.Fatalf("unexpected error querying subject history: %v", err)
	}
	if len(subHistB) != 0 {
		t.Fatalf("cross-tenant subject leakage: Tenant B received records: %+v", subHistB)
	}

	// 3. Multi-project access evaluation denies cross-tenant target
	partA, _ := localidentity.NewProjectParticipation(
		"part_a_1", tenantA, subjectA, "prj_a",
		localidentity.RoleInspector, localidentity.ScopeGrant{TenantID: tenantA, ProjectID: "prj_a"},
		"usr_admin_lead", t0, t0.Add(24*time.Hour), false,
	)
	_ = registry.AssignParticipation(partA)

	reqCrossTenant := localidentity.AccessRequest{
		Identity: localidentity.SubjectIdentity{Subject: subjectA, TenantID: tenantA, IsAuthenticated: true},
		Target:   localidentity.TargetResource{TenantID: tenantB, ProjectID: "prj_a", Lifecycle: localidentity.ResourceActive},
		Action:   localidentity.ActionRead,
	}
	resCrossTenant := registry.EvaluateMultiProjectAccess(reqCrossTenant, t0.Add(1*time.Hour))
	if resCrossTenant.Allowed || resCrossTenant.DenialReason != localidentity.DenialCrossTenant {
		t.Fatalf("expected DenialCrossTenant for cross-tenant target, got allowed=%v reason=%s", resCrossTenant.Allowed, resCrossTenant.DenialReason)
	}
}

func TestMultiProject_AttributionLedgerImmutability(t *testing.T) {
	ledger := localidentity.NewAttributionLedger()
	t0 := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)

	rec1 := localidentity.HistoricalAttributionRecord{
		RecordID:    "attr_immutable_01",
		TenantID:    "ten_alpha",
		ProjectID:   "prj_alpha",
		Subject:     "usr_worker_01",
		DisplayName: "Original Worker",
		RoleAtEvent: localidentity.RoleInspector,
		ActionType:  "INSPECTION_SUBMITTED",
		ResourceID:  "insp_001",
		RecordedAt:  t0,
	}
	if err := ledger.RecordAttribution(rec1); err != nil {
		t.Fatalf("initial RecordAttribution failed: %v", err)
	}

	// Attempting to overwrite with duplicate RecordID fails
	recDup := rec1
	recDup.DisplayName = "Tampered Worker Name"
	err := ledger.RecordAttribution(recDup)
	if !errors.Is(err, localidentity.ErrAttributionImmutable) {
		t.Fatalf("expected ErrAttributionImmutable on duplicate attribution record, got: %v", err)
	}

	// Blank input validation
	blankRec := rec1
	blankRec.RecordID = ""
	if err := ledger.RecordAttribution(blankRec); !errors.Is(err, localidentity.ErrBlankAttributionID) {
		t.Errorf("expected ErrBlankAttributionID for empty record ID, got: %v", err)
	}
}
