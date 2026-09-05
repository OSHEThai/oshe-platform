// Package localidentity_test provides the integrated qualification test suite for external user lifecycles,
// access conditions, bounded contractor administration, auditor read-only boundaries, and preserved attribution.
//
// QUALIFICATION SUITE DECLARATION (Issue #97 / V030-I024):
// Under approved Sole Human Owner decisions H030-003, H030-004, and H030-005, this qualification
// suite establishes deterministic local synthetic evidence for:
// 1. External user enrollment temporal validity, expiration, and explicit revocation mechanics.
// 2. Access condition lifecycles, sponsor-change protocols, 7-day renewal limits, and generation-based stale-session invalidation.
// 3. Multi-project participation isolation: a worker active across two projects cannot access unauthorized sibling projects.
// 4. Bounded contractor administration: external contractors are strictly barred from administrative/management roles and actions.
// 5. Auditor read-only boundaries: compliance auditors cannot execute mutating operations across projects.
// 6. Post-deactivation historical actor attribution preservation: historical actions remain permanently intact and queryable with zero deletion.
// 7. Mandatory online-only access (trusted-device prohibition) and profile data minimization (PII rejection).
//
// Boundary & Non-Claims Invariant:
// Operates exclusively in-memory on local synthetic fixtures (usr_*, prj_*, ten_*). Zero external
// identity provider synchronization, zero database persistence, zero network routes, zero customer data,
// and zero binding operational authority or runtime policy activation are claimed or enacted.
package localidentity_test

import (
	"errors"
	"testing"
	"time"

	localidentity "github.com/oshethai/oshe-platform/modules/identity-authorization"
)

// TestQualification_ExternalUser_TemporalExpiryAndRevocation verifies enrollment start/end validity windows,
// automatic temporal expiration, explicit revocation mechanics, and time-window input validation.
func TestQualification_ExternalUser_TemporalExpiryAndRevocation(t *testing.T) {
	tenantID := "ten_qual_ext_01"
	subject := "usr_ext_synth_temporal"
	sponsorID := "usr_sponsor_mgr"
	t0 := time.Date(2026, 9, 5, 8, 0, 0, 0, time.UTC)
	t1 := t0.Add(7 * 24 * time.Hour) // 7 days

	// 1. Inverted time window rejected
	_, err := localidentity.NewExternalUserProfile(
		subject, tenantID, "cmp_contractor_01", localidentity.ExternalTypeContractorWorker,
		sponsorID, "Vendor Corp", "Somchai S", "ref_synth_001",
		t1, t0, nil,
	)
	if !errors.Is(err, localidentity.ErrInvalidTimeWindow) {
		t.Fatalf("expected ErrInvalidTimeWindow for inverted window, got: %v", err)
	}

	// 2. Valid profile construction
	profile, err := localidentity.NewExternalUserProfile(
		subject, tenantID, "cmp_contractor_01", localidentity.ExternalTypeContractorWorker,
		sponsorID, "Vendor Corp", "Somchai S", "ref_synth_001",
		t0, t1, nil,
	)
	if err != nil {
		t.Fatalf("unexpected NewExternalUserProfile error: %v", err)
	}

	// 3. Temporal validity before validFrom
	tBefore := t0.Add(-1 * time.Hour)
	if profile.IsValidAt(tBefore) {
		t.Errorf("expected IsValidAt == false before validFrom")
	}
	if profile.EffectiveStatus(tBefore) != localidentity.EnrollmentStatusExpired {
		t.Errorf("expected EffectiveStatus == EXPIRED before validFrom, got: %s", profile.EffectiveStatus(tBefore))
	}

	// 4. Temporal validity within window
	tMid := t0.Add(3 * 24 * time.Hour)
	if !profile.IsValidAt(tMid) {
		t.Errorf("expected IsValidAt == true within window")
	}
	if profile.EffectiveStatus(tMid) != localidentity.EnrollmentStatusActive {
		t.Errorf("expected EffectiveStatus == ACTIVE within window, got: %s", profile.EffectiveStatus(tMid))
	}

	// 5. Temporal validity after validTo
	tAfter := t1.Add(1 * time.Hour)
	if profile.IsValidAt(tAfter) {
		t.Errorf("expected IsValidAt == false after validTo")
	}
	if profile.EffectiveStatus(tAfter) != localidentity.EnrollmentStatusExpired {
		t.Errorf("expected EffectiveStatus == EXPIRED after validTo, got: %s", profile.EffectiveStatus(tAfter))
	}

	// 6. Explicit Revocation
	revokedProfile, auditRec, err := profile.Revoke(tMid)
	if err != nil {
		t.Fatalf("unexpected Revoke error: %v", err)
	}
	if revokedProfile.IsActive() {
		t.Errorf("expected revoked profile IsActive == false")
	}
	if revokedProfile.Status() != localidentity.EnrollmentStatusRevoked {
		t.Errorf("expected Status == REVOKED, got: %s", revokedProfile.Status())
	}
	if revokedProfile.EffectiveStatus(tMid) != localidentity.EnrollmentStatusRevoked {
		t.Errorf("expected EffectiveStatus == REVOKED, got: %s", revokedProfile.EffectiveStatus(tMid))
	}
	if auditRec.Transition != "EXTERNAL_USER_REVOKED" {
		t.Errorf("expected transition EXTERNAL_USER_REVOKED, got: %s", auditRec.Transition)
	}

	// 7. Double revocation rejected
	_, _, err = revokedProfile.Revoke(tMid)
	if !errors.Is(err, localidentity.ErrEnrollmentRevoked) {
		t.Errorf("expected ErrEnrollmentRevoked on double revoke, got: %v", err)
	}
}

// TestQualification_ExternalUser_RenewalAndSponsorChangeGeneration verifies access condition lifecycles,
// 7-day renewal extension ceilings, sponsor-change protocols, and generation-based stale-session invalidation.
func TestQualification_ExternalUser_RenewalAndSponsorChangeGeneration(t *testing.T) {
	ledger := localidentity.NewAccessConditionLedger()
	registry := localidentity.NewAccessConditionRegistry(ledger)

	tenantID := "ten_qual_ext_02"
	subject := "usr_ext_synth_renewal"
	projectID := "prj_qual_site_a"
	sponsor1 := "usr_sponsor_mgr_1"
	sponsor2 := "usr_sponsor_mgr_2"
	t0 := time.Date(2026, 9, 5, 9, 0, 0, 0, time.UTC)
	t1 := t0.Add(10 * 24 * time.Hour) // Initial 10 days (within 14-day max)

	// 1. Create condition with initial Generation = 1
	cond, err := localidentity.NewAccessConditionRecord(
		"cnd_qual_01", tenantID, subject, projectID, "", sponsor1,
		false, false, t0, t1,
	)
	if err != nil {
		t.Fatalf("unexpected NewAccessConditionRecord error: %v", err)
	}
	if err := registry.CreateCondition(cond, sponsor1, "Initial project onboarding", t0); err != nil {
		t.Fatalf("failed to create condition: %v", err)
	}

	// 2. Caller presenting session token generation = 1 is allowed
	tNow := t0.Add(1 * time.Hour)
	allowed, denial := registry.EvaluateConditionAccess(tenantID, "cnd_qual_01", projectID, "", 1, tNow)
	if !allowed || denial != localidentity.CategoryNone {
		t.Fatalf("expected initial session allowed, got allowed=%v denial=%s", allowed, denial)
	}

	// 3. Renewal extension > 7 days rejected
	_, err = registry.RenewAccess(tenantID, "cnd_qual_01", 8*24*time.Hour, sponsor1, "Attempt 8-day renewal", tNow)
	if !errors.Is(err, localidentity.ErrRenewalDurationExceeded) {
		t.Fatalf("expected ErrRenewalDurationExceeded for 8-day renewal, got: %v", err)
	}

	// 4. Valid 5-day renewal increments generation to 2
	tRenew := t0.Add(5 * 24 * time.Hour)
	renewedCond, err := registry.RenewAccess(tenantID, "cnd_qual_01", 5*24*time.Hour, sponsor1, "5-day approved extension", tRenew)
	if err != nil {
		t.Fatalf("unexpected RenewAccess error: %v", err)
	}
	if renewedCond.Generation() != 2 {
		t.Errorf("expected condition Generation == 2, got: %d", renewedCond.Generation())
	}
	if renewedCond.RenewalCount() != 1 {
		t.Errorf("expected RenewalCount == 1, got: %d", renewedCond.RenewalCount())
	}

	// 5. Stale-session check: Token with generation 1 is now DENIED as stale
	allowedOld, denialOld := registry.EvaluateConditionAccess(tenantID, "cnd_qual_01", projectID, "", 1, tRenew)
	if allowedOld || denialOld != localidentity.CategorySessionStale {
		t.Errorf("expected generation 1 token denied as CategorySessionStale after renewal, got: allowed=%v denial=%s", allowedOld, denialOld)
	}

	// Token with generation 2 is ALLOWED
	allowedNew, denialNew := registry.EvaluateConditionAccess(tenantID, "cnd_qual_01", projectID, "", 2, tRenew)
	if !allowedNew || denialNew != localidentity.CategoryNone {
		t.Errorf("expected generation 2 token allowed after renewal, got: allowed=%v denial=%s", allowedNew, denialNew)
	}

	// 6. Sponsor Change: transfer to sponsor2 increments generation to 3
	tSponsorChange := tRenew.Add(1 * time.Hour)
	updatedCond, err := registry.ChangeSponsor(tenantID, "cnd_qual_01", sponsor2, sponsor1, "Manager rotation", tSponsorChange)
	if err != nil {
		t.Fatalf("unexpected ChangeSponsor error: %v", err)
	}
	if updatedCond.Generation() != 3 {
		t.Errorf("expected Generation == 3 after sponsor change, got: %d", updatedCond.Generation())
	}
	if updatedCond.SponsorID() != sponsor2 {
		t.Errorf("expected SponsorID == %s, got: %s", sponsor2, updatedCond.SponsorID())
	}

	// Reassigning to same sponsor rejected
	_, err = registry.ChangeSponsor(tenantID, "cnd_qual_01", sponsor2, sponsor2, "No-op", tSponsorChange)
	if !errors.Is(err, localidentity.ErrSponsorUnchanged) {
		t.Errorf("expected ErrSponsorUnchanged when reassigned to same sponsor, got: %v", err)
	}

	// External user acting as sponsor rejected
	_, err = registry.ChangeSponsor(tenantID, "cnd_qual_01", "usr_ext_other_contractor", sponsor2, "Illegal delegation", tSponsorChange)
	if !errors.Is(err, localidentity.ErrInvalidInternalSponsor) {
		t.Errorf("expected ErrInvalidInternalSponsor for external user sponsor, got: %v", err)
	}

	// 7. Stale-session check: Token with generation 2 is now DENIED as stale
	allowedOld2, denialOld2 := registry.EvaluateConditionAccess(tenantID, "cnd_qual_01", projectID, "", 2, tSponsorChange)
	if allowedOld2 || denialOld2 != localidentity.CategorySessionStale {
		t.Errorf("expected generation 2 token denied as CategorySessionStale after sponsor change, got: allowed=%v denial=%s", allowedOld2, denialOld2)
	}

	// Token with generation 3 is ALLOWED
	allowedNew3, denialNew3 := registry.EvaluateConditionAccess(tenantID, "cnd_qual_01", projectID, "", 3, tSponsorChange)
	if !allowedNew3 || denialNew3 != localidentity.CategoryNone {
		t.Errorf("expected generation 3 token allowed after sponsor change, got: allowed=%v denial=%s", allowedNew3, denialNew3)
	}

	// 8. Audit ledger captures complete transition history
	history, err := ledger.GetConditionHistory(tenantID, "cnd_qual_01")
	if err != nil {
		t.Fatalf("unexpected GetConditionHistory error: %v", err)
	}
	if len(history) != 3 {
		t.Fatalf("expected 3 audit records (CREATED, RENEWED, SPONSOR_CHANGED), got %d", len(history))
	}
	if history[0].Transition != "CONDITION_CREATED" || history[1].Transition != "ACCESS_RENEWED" || history[2].Transition != "SPONSOR_CHANGED" {
		t.Errorf("unexpected audit transitions: %+v", history)
	}
}

// TestQualification_ExternalUser_TwoProjectIsolationAndAntiLeakage verifies concurrent multi-project
// participation, strict cross-project access denial, and anti-leakage boundaries.
func TestQualification_ExternalUser_TwoProjectIsolationAndAntiLeakage(t *testing.T) {
	ledger := localidentity.NewAttributionLedger()
	registry := localidentity.NewMultiProjectParticipationRegistry(ledger)

	tenantID := "ten_qual_ext_03"
	subject := "usr_ext_synth_twoproject"
	sponsorID := "usr_sponsor_lead"
	t0 := time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC)
	t1 := t0.Add(14 * 24 * time.Hour)
	now := t0.Add(1 * time.Hour)

	// 1. Assign worker to prj_alpha (RoleInspector)
	scopeAlpha := localidentity.ScopeGrant{TenantID: tenantID, ProjectID: "prj_alpha"}
	partAlpha, err := localidentity.NewProjectParticipation(
		"part_ext_alpha", tenantID, subject, "prj_alpha",
		localidentity.RoleInspector, scopeAlpha, sponsorID,
		t0, t1, false,
	)
	if err != nil {
		t.Fatalf("unexpected NewProjectParticipation error for prj_alpha: %v", err)
	}
	_ = registry.AssignParticipation(partAlpha)

	// 2. Assign same worker to prj_beta (RoleContractor)
	scopeBeta := localidentity.ScopeGrant{TenantID: tenantID, ProjectID: "prj_beta"}
	partBeta, err := localidentity.NewProjectParticipation(
		"part_ext_beta", tenantID, subject, "prj_beta",
		localidentity.RoleContractor, scopeBeta, sponsorID,
		t0, t1, true,
	)
	if err != nil {
		t.Fatalf("unexpected NewProjectParticipation error for prj_beta: %v", err)
	}
	_ = registry.AssignParticipation(partBeta)

	// 3. Verify active participations count is exactly 2
	activeParts, err := registry.ListActiveParticipationsBySubject(tenantID, subject, now)
	if err != nil || len(activeParts) != 2 {
		t.Fatalf("expected 2 active project participations, got len=%d err=%v", len(activeParts), err)
	}

	// 4. Authorized access on prj_alpha succeeds
	resAlpha := registry.EvaluateMultiProjectAccess(localidentity.AccessRequest{
		Identity: localidentity.SubjectIdentity{Subject: subject, TenantID: tenantID, IsAuthenticated: true},
		Target:   localidentity.TargetResource{TenantID: tenantID, ProjectID: "prj_alpha", Lifecycle: localidentity.ResourceActive},
		Action:   localidentity.ActionRead,
	}, now)
	if !resAlpha.Allowed {
		t.Errorf("expected read allowed on assigned project prj_alpha, got denied: %s", resAlpha.DenialReason)
	}

	// 5. Authorized access on prj_beta succeeds
	resBeta := registry.EvaluateMultiProjectAccess(localidentity.AccessRequest{
		Identity: localidentity.SubjectIdentity{Subject: subject, TenantID: tenantID, IsAuthenticated: true},
		Target:   localidentity.TargetResource{TenantID: tenantID, ProjectID: "prj_beta", Lifecycle: localidentity.ResourceActive},
		Action:   localidentity.ActionRead,
	}, now)
	if !resBeta.Allowed {
		t.Errorf("expected read allowed on assigned project prj_beta, got denied: %s", resBeta.DenialReason)
	}

	// 6. Unauthorized access on sibling prj_gamma fails closed with DenialScopeMismatch
	resGamma := registry.EvaluateMultiProjectAccess(localidentity.AccessRequest{
		Identity: localidentity.SubjectIdentity{Subject: subject, TenantID: tenantID, IsAuthenticated: true},
		Target:   localidentity.TargetResource{TenantID: tenantID, ProjectID: "prj_gamma", Lifecycle: localidentity.ResourceActive},
		Action:   localidentity.ActionRead,
	}, now)
	if resGamma.Allowed || resGamma.DenialReason != localidentity.DenialScopeMismatch {
		t.Errorf("expected DenialScopeMismatch on unassigned prj_gamma, got allowed=%v reason=%s", resGamma.Allowed, resGamma.DenialReason)
	}

	// 7. Cross-tenant access to foreign tenant fails closed with DenialCrossTenant
	resCrossTenant := registry.EvaluateMultiProjectAccess(localidentity.AccessRequest{
		Identity: localidentity.SubjectIdentity{Subject: subject, TenantID: tenantID, IsAuthenticated: true},
		Target:   localidentity.TargetResource{TenantID: "ten_foreign_corp", ProjectID: "prj_alpha", Lifecycle: localidentity.ResourceActive},
		Action:   localidentity.ActionRead,
	}, now)
	if resCrossTenant.Allowed || resCrossTenant.DenialReason != localidentity.DenialCrossTenant {
		t.Errorf("expected DenialCrossTenant for cross-tenant target, got allowed=%v reason=%s", resCrossTenant.Allowed, resCrossTenant.DenialReason)
	}
}

// TestQualification_ExternalUser_ContractorAndAuditorBoundaries verifies bounded contractor administration
// and auditor read-only restrictions.
func TestQualification_ExternalUser_ContractorAndAuditorBoundaries(t *testing.T) {
	tenantID := "ten_qual_ext_04"
	contractorSubject := "usr_ext_synth_contractor"
	auditorSubject := "usr_ext_synth_auditor"
	sponsorID := "usr_sponsor_lead"
	t0 := time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC)
	t1 := t0.Add(14 * 24 * time.Hour)
	scope := localidentity.ScopeGrant{TenantID: tenantID, ProjectID: "prj_bounds"}

	// 1. External contractor cannot hold TenantAdmin or ProjectManager in profile
	for _, adminRole := range []localidentity.Role{localidentity.RoleTenantAdmin, localidentity.RoleProjectManager} {
		if err := localidentity.AssertNoCompanyAdministration(localidentity.ExternalTypeContractorWorker, adminRole); !errors.Is(err, localidentity.ErrCompanyAdminDenied) {
			t.Errorf("expected ErrCompanyAdminDenied for contractor holding %s, got: %v", adminRole, err)
		}
	}

	// 2. Contractor cannot be assigned TenantAdmin or ProjectManager in participation
	for _, adminRole := range []localidentity.Role{localidentity.RoleTenantAdmin, localidentity.RoleProjectManager} {
		_, err := localidentity.NewProjectParticipation(
			"part_bad_admin", tenantID, contractorSubject, "prj_bounds",
			adminRole, scope, sponsorID, t0, t1, true,
		)
		if !errors.Is(err, localidentity.ErrContractorAdminProhibited) {
			t.Errorf("expected ErrContractorAdminProhibited for contractor participation with %s, got: %v", adminRole, err)
		}
	}

	// 3. Contractor cannot delete resources
	if err := localidentity.AssertContractorAdminBounds(localidentity.RoleContractor, true, localidentity.ActionDelete, ""); !errors.Is(err, localidentity.ErrContractorAdminProhibited) {
		t.Errorf("expected ErrContractorAdminProhibited for contractor ActionDelete, got: %v", err)
	}

	// 4. Contractor cannot exercise administrative permissions
	adminPerms := []localidentity.Permission{
		localidentity.PermOrgTenantManage,
		localidentity.PermOrgProjectManage,
		localidentity.PermIdentityUserManage,
		localidentity.PermIdentityRoleAssign,
		localidentity.PermInspectionApprove,
		localidentity.PermAuditExport,
	}
	for _, perm := range adminPerms {
		if err := localidentity.AssertContractorAdminBounds(localidentity.RoleContractor, true, localidentity.ActionRead, perm); !errors.Is(err, localidentity.ErrContractorAdminProhibited) {
			t.Errorf("expected ErrContractorAdminProhibited for contractor exercising %s, got: %v", perm, err)
		}
	}

	// 5. Auditor is strictly read-only: mutating actions fail closed
	mutatingActions := []localidentity.Action{
		localidentity.ActionCreate,
		localidentity.ActionUpdate,
		localidentity.ActionDelete,
	}
	for _, act := range mutatingActions {
		if err := localidentity.AssertAuditorReadOnly(localidentity.RoleAuditor, act, ""); !errors.Is(err, localidentity.ErrAuditorReadOnlyViolation) {
			t.Errorf("expected ErrAuditorReadOnlyViolation for auditor %s, got: %v", act, err)
		}
	}

	// 6. Auditor attempting mutating permissions fails closed
	mutatingPerms := []localidentity.Permission{
		localidentity.PermInspectionCreate,
		localidentity.PermFindingCreate,
		localidentity.PermFindingRemediate,
		localidentity.PermRecordArchive,
	}
	for _, perm := range mutatingPerms {
		if err := localidentity.AssertAuditorReadOnly(localidentity.RoleAuditor, localidentity.ActionRead, perm); !errors.Is(err, localidentity.ErrAuditorReadOnlyViolation) {
			t.Errorf("expected ErrAuditorReadOnlyViolation for auditor exercising %s, got: %v", perm, err)
		}
	}

	// Verify auditor profile assignment works with read-only capabilities
	auditorProfile, err := localidentity.NewExternalUserProfile(
		auditorSubject, tenantID, "cmp_audit_firm", localidentity.ExternalTypeAuditor,
		sponsorID, "Compliance Auditor Firm", "Priya Inspector", "ref_synth_aud_01",
		t0, t1, []localidentity.ScopeGrant{scope},
	)
	if err != nil {
		t.Fatalf("unexpected NewExternalUserProfile error for auditor: %v", err)
	}
	if auditorProfile.UserType() != localidentity.ExternalTypeAuditor {
		t.Errorf("expected UserType == EXTERNAL_AUDITOR, got: %s", auditorProfile.UserType())
	}
}

// TestQualification_ExternalUser_DeactivationAndHistoricalAttribution verifies that deactivation
// terminates operational access immediately, while historical attribution records remain permanently intact.
func TestQualification_ExternalUser_DeactivationAndHistoricalAttribution(t *testing.T) {
	ledger := localidentity.NewAttributionLedger()
	registry := localidentity.NewMultiProjectParticipationRegistry(ledger)

	tenantID := "ten_qual_ext_05"
	subject := "usr_ext_synth_attribution"
	projectID := "prj_qual_history"
	sponsorID := "usr_sponsor_lead"
	t0 := time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC)
	t1 := t0.Add(14 * 24 * time.Hour)

	// 1. Register participation
	scope := localidentity.ScopeGrant{TenantID: tenantID, ProjectID: projectID}
	part, err := localidentity.NewProjectParticipation(
		"part_hist_01", tenantID, subject, projectID,
		localidentity.RoleInspector, scope, sponsorID,
		t0, t1, false,
	)
	if err != nil {
		t.Fatalf("unexpected NewProjectParticipation error: %v", err)
	}
	_ = registry.AssignParticipation(part)

	// 2. Log operational actions while active
	rec1 := localidentity.HistoricalAttributionRecord{
		RecordID:    "attr_qual_001",
		TenantID:    tenantID,
		ProjectID:   projectID,
		Subject:     subject,
		DisplayName: "Kallaya Sorn",
		RoleAtEvent: localidentity.RoleInspector,
		ActionType:  "INSPECTION_SUBMITTED",
		ResourceID:  "insp_scaffold_100",
		Details:     map[string]string{"checklist": "SCAFFOLD-CHECK", "score": "PASS"},
		RecordedAt:  t0.Add(2 * time.Hour),
	}
	if err := registry.RecordAttribution(rec1); err != nil {
		t.Fatalf("unexpected RecordAttribution error: %v", err)
	}

	rec2 := localidentity.HistoricalAttributionRecord{
		RecordID:    "attr_qual_002",
		TenantID:    tenantID,
		ProjectID:   projectID,
		Subject:     subject,
		DisplayName: "Kallaya Sorn",
		RoleAtEvent: localidentity.RoleInspector,
		ActionType:  "FINDING_LOGGED",
		ResourceID:  "fnd_hazard_200",
		Details:     map[string]string{"severity": "CRITICAL", "item": "Missing handrail"},
		RecordedAt:  t0.Add(4 * time.Hour),
	}
	if err := registry.RecordAttribution(rec2); err != nil {
		t.Fatalf("unexpected RecordAttribution error: %v", err)
	}

	// 3. Deactivate participation
	tDeact := t0.Add(24 * time.Hour)
	deactPart, err := registry.DeactivateParticipation(tenantID, subject, projectID, sponsorID, "Project completed", tDeact)
	if err != nil {
		t.Fatalf("unexpected DeactivateParticipation error: %v", err)
	}
	if deactPart.IsActive() {
		t.Errorf("expected deactivated participation to be inactive")
	}

	// 4. Operational request fails closed with DenialInactiveMembership
	resPostDeact := registry.EvaluateMultiProjectAccess(localidentity.AccessRequest{
		Identity: localidentity.SubjectIdentity{Subject: subject, TenantID: tenantID, IsAuthenticated: true},
		Target:   localidentity.TargetResource{TenantID: tenantID, ProjectID: projectID, Lifecycle: localidentity.ResourceActive},
		Action:   localidentity.ActionRead,
	}, tDeact.Add(1*time.Hour))
	if resPostDeact.Allowed || resPostDeact.DenialReason != localidentity.DenialInactiveMembership {
		t.Fatalf("expected DenialInactiveMembership after deactivation, got allowed=%v reason=%s", resPostDeact.Allowed, resPostDeact.DenialReason)
	}

	// 5. Historical records remain fully preserved and queryable
	trail, err := registry.GetAttributionTrail(tenantID, projectID, "insp_scaffold_100")
	if err != nil || len(trail) != 1 {
		t.Fatalf("attribution trail lost after deactivation: err=%v len=%d", err, len(trail))
	}
	if trail[0].Subject != subject || trail[0].DisplayName != "Kallaya Sorn" || trail[0].RoleAtEvent != localidentity.RoleInspector {
		t.Errorf("attribution record corrupted: %+v", trail[0])
	}

	// 6. Ledger rejects record overwrite (immutability)
	tamperedRec := rec1
	tamperedRec.DisplayName = "Altered Whistleblower"
	err = ledger.RecordAttribution(tamperedRec)
	if !errors.Is(err, localidentity.ErrAttributionImmutable) {
		t.Errorf("expected ErrAttributionImmutable on overwrite attempt, got: %v", err)
	}

	// 7. Cross-tenant isolation: Foreign tenant receives 0 records
	leakTrail, err := registry.GetAttributionTrail("ten_foreign_corp", projectID, "insp_scaffold_100")
	if err != nil || len(leakTrail) != 0 {
		t.Fatalf("cross-tenant attribution leakage detected: %+v", leakTrail)
	}
}

// TestQualification_ExternalUser_OnlineOnlyAndProfileMinimization verifies mandatory online-only access
// (trusted-device prohibition) and PII rejection in external user profiles.
func TestQualification_ExternalUser_OnlineOnlyAndProfileMinimization(t *testing.T) {
	tenantID := "ten_qual_ext_06"
	t0 := time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC)
	t1 := t0.Add(7 * 24 * time.Hour)

	// 1. Mandatory online-only: trusted device or offline access fails closed
	_, err := localidentity.NewAccessConditionRecord(
		"cnd_bad_device", tenantID, "usr_ext_01", "prj_01", "", "usr_sponsor_lead",
		true, false, t0, t1, // trustedDeviceRequired = true
	)
	if !errors.Is(err, localidentity.ErrTrustedDeviceProhibited) {
		t.Errorf("expected ErrTrustedDeviceProhibited for trustedDeviceRequired=true, got: %v", err)
	}

	_, err = localidentity.NewAccessConditionRecord(
		"cnd_bad_offline", tenantID, "usr_ext_01", "prj_01", "", "usr_sponsor_lead",
		false, true, t0, t1, // allowOffline = true
	)
	if !errors.Is(err, localidentity.ErrTrustedDeviceProhibited) {
		t.Errorf("expected ErrTrustedDeviceProhibited for allowOffline=true, got: %v", err)
	}

	// 2. Profile Data Minimization: PII payloads rejected
	piiCases := []struct {
		name       string
		contactRef string
		desc       string
	}{
		{"Somchai somchai@contractor.com", "ref_synth_01", "email in name"},
		{"Somchai", "contact: worker@vendor.com", "email in contact reference"},
		{"Somchai Phone: 0812345678", "ref_synth_01", "phone in name"},
		{"Somchai", "+66891234567", "phone in contact reference"},
		{"Somchai Citizen ID: 1234567890123", "ref_synth_01", "national ID in name"},
		{"Somchai", "passport: AA1234567", "passport in contact reference"},
	}

	for _, tc := range piiCases {
		_, err := localidentity.NewExternalUserProfile(
			"usr_ext_pii", tenantID, "cmp_01", localidentity.ExternalTypeTemporaryWorker,
			"usr_sponsor_lead", "Vendor Co", tc.name, tc.contactRef,
			t0, t1, nil,
		)
		if !errors.Is(err, localidentity.ErrPIIDetected) {
			t.Errorf("%s: expected ErrPIIDetected, got: %v", tc.desc, err)
		}
	}
}
