package publicationsnapshot_test

import (
	"testing"
	"time"

	publicationsnapshot "oshe/publication-snapshot"
)

func TestLifecycle_DraftRegistrationAndSubmission(t *testing.T) {
	ledger := publicationsnapshot.NewLifecycleAuditLedger()
	ctrl := publicationsnapshot.NewLifecycleController(ledger)

	tenantID := "ten_life_01"
	snapID := "snap_life_001"
	digest := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	t0 := time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC)

	// 1. Register Draft
	snap, err := ctrl.RegisterDraft(tenantID, snapID, 1, digest, "usr_author_01", t0)
	if err != nil {
		t.Fatalf("RegisterDraft failed: %v", err)
	}
	if snap.State != publicationsnapshot.StateDraft {
		t.Errorf("expected state DRAFT, got %s", snap.State)
	}
	if snap.SnapshotID != snapID || snap.TenantID != tenantID || snap.Version != 1 {
		t.Errorf("snapshot metadata mismatch: %+v", snap)
	}

	// 2. Submit for Review
	t1 := t0.Add(1 * time.Hour)
	submitted, err := ctrl.SubmitForReview(tenantID, snapID, "usr_author_01", "AUTHOR", t1)
	if err != nil {
		t.Fatalf("SubmitForReview failed: %v", err)
	}
	if submitted.State != publicationsnapshot.StateUnderReview {
		t.Errorf("expected state UNDER_REVIEW, got %s", submitted.State)
	}

	// 3. Verify Audit Ledger
	history, err := ledger.GetHistory(tenantID, snapID)
	if err != nil {
		t.Fatalf("GetHistory failed: %v", err)
	}
	if len(history) != 2 {
		t.Fatalf("expected 2 audit records, got %d", len(history))
	}
	if history[0].ToState != publicationsnapshot.StateDraft || history[1].ToState != publicationsnapshot.StateUnderReview {
		t.Errorf("audit history transitions mismatch: %+v", history)
	}
}

func TestLifecycle_ApprovalAndPublish_DeterministicWorkflow(t *testing.T) {
	ledger := publicationsnapshot.NewLifecycleAuditLedger()
	ctrl := publicationsnapshot.NewLifecycleController(ledger)

	tenantID := "ten_life_02"
	snapID := "snap_life_002"
	digest := "a1b2c3d4e5f60123456789abcdef0123456789abcdef0123456789abcdef0123"
	t0 := time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC)

	_, _ = ctrl.RegisterDraft(tenantID, snapID, 1, digest, "usr_author_01", t0)
	_, _ = ctrl.SubmitForReview(tenantID, snapID, "usr_author_01", "AUTHOR", t0.Add(10*time.Minute))

	// 1. Approve
	tApprove := t0.Add(1 * time.Hour)
	evidence, err := publicationsnapshot.NewApprovalEvidence("app_01", "usr_auditor_01", "AUDITOR", tApprove, digest, "Conforms to publication policy", 7*24*time.Hour)
	if err != nil {
		t.Fatalf("NewApprovalEvidence failed: %v", err)
	}

	approved, err := ctrl.Approve(tenantID, snapID, evidence, tApprove)
	if err != nil {
		t.Fatalf("Approve failed: %v", err)
	}
	if approved.State != publicationsnapshot.StateApproved {
		t.Errorf("expected state APPROVED, got %s", approved.State)
	}

	// 2. Publish within valid window
	tPublish := t0.Add(2 * time.Hour)
	window := publicationsnapshot.EffectiveWindow{
		EffectiveFrom: tPublish,
		ExpiresAt:     tPublish.Add(90 * 24 * time.Hour), // 90 days
	}
	published, err := ctrl.Publish(tenantID, snapID, "usr_auditor_01", "AUDITOR", window, tPublish)
	if err != nil {
		t.Fatalf("Publish failed: %v", err)
	}
	if published.State != publicationsnapshot.StatePublished {
		t.Errorf("expected state PUBLISHED, got %s", published.State)
	}
	if !published.IsActive(tPublish) {
		t.Errorf("expected published snapshot to be active at publication time")
	}

	// 3. Retrieve through GetActivePublishedSnapshot
	activeSnap, err := ctrl.GetActivePublishedSnapshot(tenantID, snapID, tPublish.Add(1*time.Hour))
	if err != nil {
		t.Fatalf("GetActivePublishedSnapshot failed: %v", err)
	}
	if activeSnap.SnapshotID != snapID {
		t.Errorf("active snapshot ID mismatch: %s", activeSnap.SnapshotID)
	}
}

func TestLifecycle_EffectiveAndExpiryWindowEvaluation(t *testing.T) {
	ledger := publicationsnapshot.NewLifecycleAuditLedger()
	ctrl := publicationsnapshot.NewLifecycleController(ledger)

	tenantID := "ten_life_03"
	snapID := "snap_life_003"
	digest := "beefdead0123456789abcdef0123456789abcdef0123456789abcdef01234567"
	t0 := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)

	_, _ = ctrl.RegisterDraft(tenantID, snapID, 1, digest, "usr_author", t0)
	_, _ = ctrl.SubmitForReview(tenantID, snapID, "usr_author", "AUTHOR", t0)
	ev, _ := publicationsnapshot.NewApprovalEvidence("app_03", "usr_admin", "TENANT_ADMIN", t0, digest, "Approved", 7*24*time.Hour)
	_, _ = ctrl.Approve(tenantID, snapID, ev, t0)

	// Effective starts tomorrow, expires in 30 days
	effectiveFrom := t0.Add(24 * time.Hour)
	expiresAt := effectiveFrom.Add(30 * 24 * time.Hour)
	window := publicationsnapshot.EffectiveWindow{
		EffectiveFrom: effectiveFrom,
		ExpiresAt:     expiresAt,
	}

	_, err := ctrl.Publish(tenantID, snapID, "usr_admin", "TENANT_ADMIN", window, t0)
	if err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	// 1. Before effective date: Not yet effective
	tBefore := t0.Add(1 * time.Hour)
	snapBefore, err := ctrl.GetSnapshot(tenantID, snapID, tBefore)
	if err != nil {
		t.Fatalf("GetSnapshot before failed: %v", err)
	}
	if snapBefore.IsActive(tBefore) {
		t.Errorf("expected IsActive to be false before EffectiveFrom")
	}
	_, err = ctrl.GetActivePublishedSnapshot(tenantID, snapID, tBefore)
	if err != publicationsnapshot.ErrNotYetEffective {
		t.Errorf("expected ErrNotYetEffective, got: %v", err)
	}

	// 2. During active window: Active
	tMid := effectiveFrom.Add(15 * 24 * time.Hour)
	snapMid, err := ctrl.GetActivePublishedSnapshot(tenantID, snapID, tMid)
	if err != nil {
		t.Fatalf("GetActivePublishedSnapshot during window failed: %v", err)
	}
	if !snapMid.IsActive(tMid) {
		t.Errorf("expected IsActive == true during active window")
	}

	// 3. After expiry date: Expired
	tAfter := expiresAt.Add(1 * time.Hour)
	snapAfter, err := ctrl.GetSnapshot(tenantID, snapID, tAfter)
	if err != nil {
		t.Fatalf("GetSnapshot after expiry failed: %v", err)
	}
	if snapAfter.State != publicationsnapshot.StateExpired {
		t.Errorf("expected EffectiveState to evaluate to EXPIRED, got %s", snapAfter.State)
	}
	if snapAfter.IsActive(tAfter) {
		t.Errorf("expected IsActive == false after expiry")
	}
	_, err = ctrl.GetActivePublishedSnapshot(tenantID, snapID, tAfter)
	if err != publicationsnapshot.ErrSnapshotExpired {
		t.Errorf("expected ErrSnapshotExpired, got: %v", err)
	}
}

func TestLifecycle_Withdrawal_MandatoryAttributionAndAudit(t *testing.T) {
	ledger := publicationsnapshot.NewLifecycleAuditLedger()
	ctrl := publicationsnapshot.NewLifecycleController(ledger)

	tenantID := "ten_life_04"
	snapID := "snap_life_004"
	digest := "digest_life_004"
	t0 := time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC)

	_, _ = ctrl.RegisterDraft(tenantID, snapID, 1, digest, "usr_author", t0)
	_, _ = ctrl.SubmitForReview(tenantID, snapID, "usr_author", "AUTHOR", t0)
	ev, _ := publicationsnapshot.NewApprovalEvidence("app_04", "usr_auditor_01", "AUDITOR", t0, digest, "Approved", 7*24*time.Hour)
	_, _ = ctrl.Approve(tenantID, snapID, ev, t0)
	window := publicationsnapshot.EffectiveWindow{EffectiveFrom: t0, ExpiresAt: t0.Add(30 * 24 * time.Hour)}
	_, _ = ctrl.Publish(tenantID, snapID, "usr_auditor_01", "AUDITOR", window, t0)

	// Withdraw snapshot
	tWithdraw := t0.Add(5 * time.Hour)
	withdrawn, err := ctrl.Withdraw(tenantID, snapID, "usr_auditor_01", "AUDITOR", "Source measurement data revised by external laboratory", tWithdraw)
	if err != nil {
		t.Fatalf("Withdraw failed: %v", err)
	}
	if withdrawn.State != publicationsnapshot.StateWithdrawn {
		t.Errorf("expected state WITHDRAWN, got %s", withdrawn.State)
	}
	if withdrawn.WithdrawnBy != "usr_auditor_01" || withdrawn.WithdrawalReason == "" {
		t.Errorf("missing withdrawal attribution: %+v", withdrawn)
	}

	// Querying active snapshot fails closed
	_, err = ctrl.GetActivePublishedSnapshot(tenantID, snapID, tWithdraw.Add(1*time.Hour))
	if err != publicationsnapshot.ErrSnapshotWithdrawn {
		t.Errorf("expected ErrSnapshotWithdrawn, got: %v", err)
	}

	// Verify audit trail captures WITHDRAWN event
	history, err := ledger.GetHistory(tenantID, snapID)
	if err != nil {
		t.Fatalf("GetHistory failed: %v", err)
	}
	lastRec := history[len(history)-1]
	if lastRec.ToState != publicationsnapshot.StateWithdrawn || lastRec.Reason == "" || lastRec.AuditDigest == "" {
		t.Errorf("withdrawal audit record incomplete: %+v", lastRec)
	}
}

func TestLifecycle_ReplacementAndSupersession(t *testing.T) {
	ledger := publicationsnapshot.NewLifecycleAuditLedger()
	ctrl := publicationsnapshot.NewLifecycleController(ledger)

	tenantID := "ten_life_05"
	snapID1 := "snap_life_orig"
	snapID2 := "snap_life_orig_v2"
	digest := "digest_orig"
	t0 := time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC)

	_, _ = ctrl.RegisterDraft(tenantID, snapID1, 1, digest, "usr_author", t0)
	_, _ = ctrl.SubmitForReview(tenantID, snapID1, "usr_author", "AUTHOR", t0)
	ev, _ := publicationsnapshot.NewApprovalEvidence("app_05", "usr_auditor", "AUDITOR", t0, digest, "Approved", 7*24*time.Hour)
	_, _ = ctrl.Approve(tenantID, snapID1, ev, t0)
	window := publicationsnapshot.EffectiveWindow{EffectiveFrom: t0, ExpiresAt: t0.Add(30 * 24 * time.Hour)}
	_, _ = ctrl.Publish(tenantID, snapID1, "usr_auditor", "AUDITOR", window, t0)

	// 1. Supersede when new version is created
	tSupersede := t0.Add(24 * time.Hour)
	superseded, err := ctrl.Supersede(tenantID, snapID1, snapID2, "usr_auditor", "AUDITOR", tSupersede)
	if err != nil {
		t.Fatalf("Supersede failed: %v", err)
	}
	if superseded.State != publicationsnapshot.StateSuperseded || superseded.SuccessorID != snapID2 {
		t.Errorf("superseded state mismatch: %+v", superseded)
	}

	// 2. Replace workflow test
	snapID3 := "snap_replace_target"
	_, _ = ctrl.RegisterDraft(tenantID, snapID3, 1, digest, "usr_author", t0)
	_, _ = ctrl.SubmitForReview(tenantID, snapID3, "usr_author", "AUTHOR", t0)
	_, _ = ctrl.Approve(tenantID, snapID3, ev, t0)
	_, _ = ctrl.Publish(tenantID, snapID3, "usr_auditor", "AUDITOR", window, t0)

	replaced, err := ctrl.Replace(tenantID, snapID3, "snap_replacement_01", "usr_admin", "TENANT_ADMIN", "Replaced with restructured document standard", tSupersede)
	if err != nil {
		t.Fatalf("Replace failed: %v", err)
	}
	if replaced.State != publicationsnapshot.StateReplaced || replaced.SuccessorID != "snap_replacement_01" {
		t.Errorf("replaced state mismatch: %+v", replaced)
	}
}

func TestLifecycle_AuditLedger_TenantIsolationAndImmutableTrail(t *testing.T) {
	ledger := publicationsnapshot.NewLifecycleAuditLedger()
	ctrl := publicationsnapshot.NewLifecycleController(ledger)

	tenantA := "ten_alpha"
	tenantB := "ten_bravo"
	snapA := "snap_alpha"
	snapB := "snap_bravo"
	t0 := time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC)

	_, _ = ctrl.RegisterDraft(tenantA, snapA, 1, "digest_a", "usr_a", t0)
	_, _ = ctrl.RegisterDraft(tenantB, snapB, 1, "digest_b", "usr_b", t0)

	// Tenant A history only has tenant A record
	histA, err := ledger.GetHistory(tenantA, snapA)
	if err != nil {
		t.Fatalf("GetHistory tenant A failed: %v", err)
	}
	if len(histA) != 1 || histA[0].TenantID != tenantA {
		t.Errorf("tenant A history leaked foreign records: %+v", histA)
	}

	// Querying tenant B snapshot with tenant A credentials returns empty slice (zero leakage)
	crossQuery, err := ledger.GetHistory(tenantA, snapB)
	if err != nil {
		t.Fatalf("cross-query failed: %v", err)
	}
	if len(crossQuery) != 0 {
		t.Errorf("cross-tenant leakage: got %d records for foreign snapshot", len(crossQuery))
	}
}
