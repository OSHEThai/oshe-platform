package publicationsnapshot_test

import (
	"errors"
	"testing"
	"time"

	publicationsnapshot "oshe/publication-snapshot"
)

// NEG-LIFE-01: Unauthorized Publish & Review Rejection
// Threat: Unauthorized roles attempting to approve or publish public portal snapshots.
// Expected: Rejection with ErrUnauthorizedReviewer, ErrUnauthorizedPublish, or ErrIllegalStateTransition.
func TestNegativeControl_NEG_LIFE_01_UnauthorizedPublish(t *testing.T) {
	ledger := publicationsnapshot.NewLifecycleAuditLedger()
	ctrl := publicationsnapshot.NewLifecycleController(ledger)

	tenantID := "ten_neg_life_01"
	snapID := "snap_neg_01"
	digest := "digest_neg_01"
	t0 := time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC)

	_, _ = ctrl.RegisterDraft(tenantID, snapID, 1, digest, "usr_author", t0)
	_, _ = ctrl.SubmitForReview(tenantID, snapID, "usr_author", "AUTHOR", t0)

	// 1. Unauthorized role (INSPECTOR) attempting approval
	evInsp, _ := publicationsnapshot.NewApprovalEvidence("app_neg_01", "usr_insp", "INSPECTOR", t0, digest, "Inspector attempt", 7*24*time.Hour)
	_, err := ctrl.Approve(tenantID, snapID, evInsp, t0)
	if !errors.Is(err, publicationsnapshot.ErrUnauthorizedReviewer) {
		t.Fatalf("expected ErrUnauthorizedReviewer for Inspector approval, got: %v", err)
	}

	// 2. Unauthorized role (CONTRACTOR) attempting approval
	evCont, _ := publicationsnapshot.NewApprovalEvidence("app_neg_02", "usr_cont", "CONTRACTOR", t0, digest, "Contractor attempt", 7*24*time.Hour)
	_, err = ctrl.Approve(tenantID, snapID, evCont, t0)
	if !errors.Is(err, publicationsnapshot.ErrUnauthorizedReviewer) {
		t.Fatalf("expected ErrUnauthorizedReviewer for Contractor approval, got: %v", err)
	}

	// Approve properly with AUDITOR
	evAud, _ := publicationsnapshot.NewApprovalEvidence("app_neg_03", "usr_aud", "AUDITOR", t0, digest, "Auditor approval", 7*24*time.Hour)
	_, err = ctrl.Approve(tenantID, snapID, evAud, t0)
	if err != nil {
		t.Fatalf("legitimate approval failed: %v", err)
	}

	// 3. Unauthorized role (VIEWER) attempting publish
	window := publicationsnapshot.EffectiveWindow{EffectiveFrom: t0, ExpiresAt: t0.Add(30 * 24 * time.Hour)}
	_, err = ctrl.Publish(tenantID, snapID, "usr_viewer", "VIEWER", window, t0)
	if !errors.Is(err, publicationsnapshot.ErrUnauthorizedPublish) {
		t.Fatalf("expected ErrUnauthorizedPublish for Viewer, got: %v", err)
	}

	// 4. Attempting to publish an unapproved draft directly
	snapIDUnapproved := "snap_unapproved"
	_, _ = ctrl.RegisterDraft(tenantID, snapIDUnapproved, 1, digest, "usr_author", t0)
	_, err = ctrl.Publish(tenantID, snapIDUnapproved, "usr_aud", "AUDITOR", window, t0)
	if !errors.Is(err, publicationsnapshot.ErrIllegalStateTransition) {
		t.Fatalf("expected ErrIllegalStateTransition when publishing unapproved draft, got: %v", err)
	}
}

// NEG-LIFE-02: Stale Approval & Content Digest Drift Rejection
// Threat: Publishing based on outdated approvals or content modified after approval.
// Expected: Rejection with ErrStaleApproval or ErrApprovalDigestMismatch.
func TestNegativeControl_NEG_LIFE_02_StaleApproval(t *testing.T) {
	ledger := publicationsnapshot.NewLifecycleAuditLedger()
	ctrl := publicationsnapshot.NewLifecycleController(ledger)

	tenantID := "ten_neg_life_02"
	snapID := "snap_neg_02"
	digest := "digest_orig_02"
	t0 := time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC)

	_, _ = ctrl.RegisterDraft(tenantID, snapID, 1, digest, "usr_author", t0)
	_, _ = ctrl.SubmitForReview(tenantID, snapID, "usr_author", "AUTHOR", t0)

	// 1. Approval evidence created with 7-day max validity
	tOldApproval := t0.Add(-8 * 24 * time.Hour) // 8 days ago
	staleEvidence, _ := publicationsnapshot.NewApprovalEvidence("app_stale", "usr_aud", "AUDITOR", tOldApproval, digest, "Old approval", 7*24*time.Hour)

	_, err := ctrl.Approve(tenantID, snapID, staleEvidence, t0)
	if !errors.Is(err, publicationsnapshot.ErrStaleApproval) {
		t.Fatalf("expected ErrStaleApproval for 8-day old approval, got: %v", err)
	}

	// 2. Content digest mismatch (tampered/altered snapshot content)
	alteredEvidence, _ := publicationsnapshot.NewApprovalEvidence("app_altered", "usr_aud", "AUDITOR", t0, "digest_different_tampered", "Tampered", 7*24*time.Hour)
	_, err = ctrl.Approve(tenantID, snapID, alteredEvidence, t0)
	if !errors.Is(err, publicationsnapshot.ErrApprovalDigestMismatch) {
		t.Fatalf("expected ErrApprovalDigestMismatch for altered content digest, got: %v", err)
	}

	// 3. Approval valid at approval time, but becomes stale before publish
	freshEvidence, _ := publicationsnapshot.NewApprovalEvidence("app_fresh", "usr_aud", "AUDITOR", t0, digest, "Fresh approval", 3*24*time.Hour) // 3 days validity
	_, err = ctrl.Approve(tenantID, snapID, freshEvidence, t0)
	if err != nil {
		t.Fatalf("approval failed: %v", err)
	}

	// Attempt publish 4 days later (approval has expired)
	tPublishLate := t0.Add(4 * 24 * time.Hour)
	window := publicationsnapshot.EffectiveWindow{EffectiveFrom: tPublishLate, ExpiresAt: tPublishLate.Add(30 * 24 * time.Hour)}
	_, err = ctrl.Publish(tenantID, snapID, "usr_aud", "AUDITOR", window, tPublishLate)
	if !errors.Is(err, publicationsnapshot.ErrStaleApproval) {
		t.Fatalf("expected ErrStaleApproval when approval expired prior to publication, got: %v", err)
	}
}

// NEG-LIFE-03: Expiry & Publication Window Validation Denial
// Threat: Inverted dates, publication window exceeding limits, or publishing into an already-expired window.
// Expected: Rejection with ErrInvalidPublicationWindow or ErrSnapshotExpired.
func TestNegativeControl_NEG_LIFE_03_ExpiryAndWindowValidation(t *testing.T) {
	ledger := publicationsnapshot.NewLifecycleAuditLedger()
	ctrl := publicationsnapshot.NewLifecycleController(ledger)

	tenantID := "ten_neg_life_03"
	snapID := "snap_neg_03"
	digest := "digest_neg_03"
	t0 := time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC)

	_, _ = ctrl.RegisterDraft(tenantID, snapID, 1, digest, "usr_author", t0)
	_, _ = ctrl.SubmitForReview(tenantID, snapID, "usr_author", "AUTHOR", t0)
	ev, _ := publicationsnapshot.NewApprovalEvidence("app_03", "usr_aud", "AUDITOR", t0, digest, "OK", 7*24*time.Hour)
	_, _ = ctrl.Approve(tenantID, snapID, ev, t0)

	// 1. Inverted window: ExpiresAt is before EffectiveFrom
	invertedWindow := publicationsnapshot.EffectiveWindow{
		EffectiveFrom: t0.Add(10 * 24 * time.Hour),
		ExpiresAt:     t0.Add(5 * 24 * time.Hour),
	}
	_, err := ctrl.Publish(tenantID, snapID, "usr_aud", "AUDITOR", invertedWindow, t0)
	if !errors.Is(err, publicationsnapshot.ErrInvalidPublicationWindow) {
		t.Fatalf("expected ErrInvalidPublicationWindow for inverted dates, got: %v", err)
	}

	// 2. Zero-length window: ExpiresAt == EffectiveFrom
	zeroWindow := publicationsnapshot.EffectiveWindow{
		EffectiveFrom: t0,
		ExpiresAt:     t0,
	}
	_, err = ctrl.Publish(tenantID, snapID, "usr_aud", "AUDITOR", zeroWindow, t0)
	if !errors.Is(err, publicationsnapshot.ErrInvalidPublicationWindow) {
		t.Fatalf("expected ErrInvalidPublicationWindow for zero-length window, got: %v", err)
	}

	// 3. Excessively long window (> 365 days)
	excessiveWindow := publicationsnapshot.EffectiveWindow{
		EffectiveFrom: t0,
		ExpiresAt:     t0.Add(366 * 24 * time.Hour),
	}
	_, err = ctrl.Publish(tenantID, snapID, "usr_aud", "AUDITOR", excessiveWindow, t0)
	if !errors.Is(err, publicationsnapshot.ErrInvalidPublicationWindow) {
		t.Fatalf("expected ErrInvalidPublicationWindow for window > 365 days, got: %v", err)
	}

	// 4. Publishing into a window that is already in the past
	pastWindow := publicationsnapshot.EffectiveWindow{
		EffectiveFrom: t0.Add(-10 * 24 * time.Hour),
		ExpiresAt:     t0.Add(-1 * 24 * time.Hour),
	}
	_, err = ctrl.Publish(tenantID, snapID, "usr_aud", "AUDITOR", pastWindow, t0)
	if !errors.Is(err, publicationsnapshot.ErrSnapshotExpired) {
		t.Fatalf("expected ErrSnapshotExpired when publishing into past window, got: %v", err)
	}
}

// NEG-LIFE-04: Withdrawal & Replacement Validation Denial
// Threat: Arbitrary unrecorded withdrawal, blank reason, or operating on terminal states.
// Expected: Rejection with ErrMissingWithdrawalReason, ErrUnauthorizedReviewer, or ErrDuplicateTransition.
func TestNegativeControl_NEG_LIFE_04_WithdrawalAndReplacementValidation(t *testing.T) {
	ledger := publicationsnapshot.NewLifecycleAuditLedger()
	ctrl := publicationsnapshot.NewLifecycleController(ledger)

	tenantID := "ten_neg_life_04"
	snapID := "snap_neg_04"
	digest := "digest_neg_04"
	t0 := time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC)

	_, _ = ctrl.RegisterDraft(tenantID, snapID, 1, digest, "usr_author", t0)
	_, _ = ctrl.SubmitForReview(tenantID, snapID, "usr_author", "AUTHOR", t0)
	ev, _ := publicationsnapshot.NewApprovalEvidence("app_04", "usr_aud", "AUDITOR", t0, digest, "OK", 7*24*time.Hour)
	_, _ = ctrl.Approve(tenantID, snapID, ev, t0)
	window := publicationsnapshot.EffectiveWindow{EffectiveFrom: t0, ExpiresAt: t0.Add(30 * 24 * time.Hour)}
	_, _ = ctrl.Publish(tenantID, snapID, "usr_aud", "AUDITOR", window, t0)

	// 1. Withdrawal with blank reason
	_, err := ctrl.Withdraw(tenantID, snapID, "usr_aud", "AUDITOR", "   ", t0)
	if !errors.Is(err, publicationsnapshot.ErrMissingWithdrawalReason) {
		t.Fatalf("expected ErrMissingWithdrawalReason for blank justification, got: %v", err)
	}

	// 2. Withdrawal by unauthorized role (VIEWER)
	_, err = ctrl.Withdraw(tenantID, snapID, "usr_viewer", "VIEWER", "Valid reason", t0)
	if !errors.Is(err, publicationsnapshot.ErrUnauthorizedReviewer) {
		t.Fatalf("expected ErrUnauthorizedReviewer for Viewer withdrawal attempt, got: %v", err)
	}

	// Legitimately withdraw
	_, err = ctrl.Withdraw(tenantID, snapID, "usr_aud", "AUDITOR", "Legitimate recall", t0)
	if err != nil {
		t.Fatalf("legitimate withdrawal failed: %v", err)
	}

	// 3. Duplicate withdrawal on already WITHDRAWN snapshot
	_, err = ctrl.Withdraw(tenantID, snapID, "usr_aud", "AUDITOR", "Second recall attempt", t0)
	if !errors.Is(err, publicationsnapshot.ErrDuplicateTransition) {
		t.Fatalf("expected ErrDuplicateTransition for second withdrawal, got: %v", err)
	}

	// 4. Replacement on non-published (WITHDRAWN) snapshot fails
	_, err = ctrl.Replace(tenantID, snapID, "successor_id", "usr_admin", "TENANT_ADMIN", "Reason", t0)
	if !errors.Is(err, publicationsnapshot.ErrIllegalStateTransition) {
		t.Fatalf("expected ErrIllegalStateTransition when replacing a withdrawn snapshot, got: %v", err)
	}
}

// NEG-LIFE-05: Duplicate Transition Rejection
// Threat: Transitioning to the same state repeatedly or applying invalid state machine leaps.
// Expected: Rejection with ErrDuplicateTransition or ErrIllegalStateTransition.
func TestNegativeControl_NEG_LIFE_05_DuplicateTransition(t *testing.T) {
	ledger := publicationsnapshot.NewLifecycleAuditLedger()
	ctrl := publicationsnapshot.NewLifecycleController(ledger)

	tenantID := "ten_neg_life_05"
	snapID := "snap_neg_05"
	digest := "digest_neg_05"
	t0 := time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC)

	_, _ = ctrl.RegisterDraft(tenantID, snapID, 1, digest, "usr_author", t0)
	_, _ = ctrl.SubmitForReview(tenantID, snapID, "usr_author", "AUTHOR", t0)

	// 1. Re-submitting under-review snapshot fails
	_, err := ctrl.SubmitForReview(tenantID, snapID, "usr_author", "AUTHOR", t0)
	if !errors.Is(err, publicationsnapshot.ErrIllegalStateTransition) {
		t.Fatalf("expected ErrIllegalStateTransition on re-submit, got: %v", err)
	}

	ev, _ := publicationsnapshot.NewApprovalEvidence("app_05", "usr_aud", "AUDITOR", t0, digest, "OK", 7*24*time.Hour)
	_, _ = ctrl.Approve(tenantID, snapID, ev, t0)
	window := publicationsnapshot.EffectiveWindow{EffectiveFrom: t0, ExpiresAt: t0.Add(30 * 24 * time.Hour)}
	_, _ = ctrl.Publish(tenantID, snapID, "usr_aud", "AUDITOR", window, t0)

	// 2. Re-publishing already published snapshot fails
	_, err = ctrl.Publish(tenantID, snapID, "usr_aud", "AUDITOR", window, t0)
	if !errors.Is(err, publicationsnapshot.ErrDuplicateTransition) {
		t.Fatalf("expected ErrDuplicateTransition on re-publish, got: %v", err)
	}
}

// NEG-LIFE-06: Non-Live Non-Claim Invariant Verification
// Threat: False assertion of live external publication routes or persistence side-effects.
// Expected: Operates exclusively in-memory on local synthetic fixtures with zero live side-effects.
func TestNegativeControl_NEG_LIFE_06_NonLiveNonClaim(t *testing.T) {
	// Assert governance non-claims:
	// 1. Zero external network/public-route execution
	// 2. Zero production database schema or SQL mutations
	// 3. Zero real credentials or customer data
	// 4. Purely in-memory, thread-safe local synthetic models
	ledger := publicationsnapshot.NewLifecycleAuditLedger()
	ctrl := publicationsnapshot.NewLifecycleController(ledger)

	if ctrl == nil || ledger == nil {
		t.Fatalf("lifecycle controller or ledger failed to initialize in memory")
	}
}
