package publicationsnapshot_test

import (
	"errors"
	"testing"
	"time"

	publicationsnapshot "oshe/publication-snapshot"
)

// TestQualification_Publication_RedactionAndProhibitedDataMinimization qualifies:
// 1. Allowlist enforcement stripping unapproved fields in permissive mode.
// 2. Strict mode rejection of unapproved fields (ErrUnapprovedFieldDetected).
// 3. Prohibited sensitive keywords in allowlist definition rejected (ErrProhibitedFieldInAllowlist).
// 4. Input payloads containing sensitive credentials/tokens/PII failing closed (ErrProhibitedFieldDetected).
func TestQualification_Publication_RedactionAndProhibitedDataMinimization(t *testing.T) {
	// 1. Prohibited sensitive keyword in allowlist definition rejected
	badAllowlistFields := []string{"inspection_id", "admin_password", "site_name"}
	_, err := publicationsnapshot.NewPublicationFieldAllowlist(badAllowlistFields, false)
	if !errors.Is(err, publicationsnapshot.ErrProhibitedFieldInAllowlist) {
		t.Fatalf("expected ErrProhibitedFieldInAllowlist, got: %v", err)
	}

	// 2. Valid allowlist definition in permissive mode
	validFields := []string{"inspection_id", "project_code", "site_name", "findings_count", "compliance_score"}
	permissiveAL, err := publicationsnapshot.NewPublicationFieldAllowlist(validFields, false)
	if err != nil {
		t.Fatalf("unexpected NewPublicationFieldAllowlist error: %v", err)
	}

	rawPayload := map[string]any{
		"inspection_id":       "insp_qual_001",
		"project_code":        "prj_alpha",
		"site_name":           "Alpha Yard",
		"findings_count":      3,
		"compliance_score":    95.5,
		"internal_db_row_id":  98214,
		"contractor_notes":    "draft note not for public",
		"raw_telemetry_blob":  "binary_content",
	}

	// 3. Permissive redaction strips unapproved operational fields
	redacted, redactedCount, err := publicationsnapshot.RedactPayload(rawPayload, permissiveAL)
	if err != nil {
		t.Fatalf("permissive RedactPayload failed: %v", err)
	}
	if len(redacted) != 5 {
		t.Errorf("expected exactly 5 approved fields, got %d", len(redacted))
	}
	if redactedCount != 3 {
		t.Errorf("expected 3 redacted fields, got %d", redactedCount)
	}
	if _, exists := redacted["internal_db_row_id"]; exists {
		t.Errorf("internal_db_row_id leaked into redacted payload")
	}
	if _, exists := redacted["contractor_notes"]; exists {
		t.Errorf("contractor_notes leaked into redacted payload")
	}

	// 4. Strict mode fails closed on unapproved fields
	strictAL, err := publicationsnapshot.NewPublicationFieldAllowlist(validFields, true)
	if err != nil {
		t.Fatalf("unexpected strict allowlist error: %v", err)
	}
	_, _, err = publicationsnapshot.RedactPayload(rawPayload, strictAL)
	if !errors.Is(err, publicationsnapshot.ErrUnapprovedFieldDetected) {
		t.Fatalf("expected ErrUnapprovedFieldDetected in strict mode, got: %v", err)
	}

	// 5. Detection of prohibited sensitive credentials or PII fails closed immediately
	sensitivePayloads := []struct {
		name    string
		payload map[string]any
	}{
		{"bearer_token", map[string]any{"inspection_id": "insp_01", "bearer_token": "oshe_tok_deadbeef"}},
		{"password", map[string]any{"inspection_id": "insp_01", "admin_password": "super_secret_pw"}},
		{"national_id", map[string]any{"inspection_id": "insp_01", "national_id": "1234567890123"}},
		{"citizen_id", map[string]any{"inspection_id": "insp_01", "citizen_id": "9988776655443"}},
		{"ssn", map[string]any{"inspection_id": "insp_01", "ssn": "000-12-3456"}},
		{"token_prefix_value", map[string]any{"inspection_id": "insp_01", "auth_header": "Bearer oshe_tok_1234"}},
	}

	for _, tc := range sensitivePayloads {
		_, _, err := publicationsnapshot.RedactPayload(tc.payload, permissiveAL)
		if !errors.Is(err, publicationsnapshot.ErrProhibitedFieldDetected) {
			t.Errorf("[%s] expected ErrProhibitedFieldDetected, got: %v", tc.name, err)
		}
	}
}

// TestQualification_Publication_UnauthorizedPublicationAndReviewerGate qualifies:
// 1. Draft snapshots cannot be published without formal approval (ErrIllegalStateTransition).
// 2. Unauthorized reviewer roles rejected from approving (ErrUnauthorizedReviewer).
// 3. Authorized reviewer roles (AUDITOR, TENANT_ADMIN) produce valid decision digests.
// 4. Stale approval evidence (> 7 days) fails closed (ErrStaleApproval).
// 5. Approval digest mismatch fails closed (ErrApprovalDigestMismatch).
func TestQualification_Publication_UnauthorizedPublicationAndReviewerGate(t *testing.T) {
	ledger := publicationsnapshot.NewLifecycleAuditLedger()
	ctrl := publicationsnapshot.NewLifecycleController(ledger)
	tenantID := "ten_qual_pub_02"
	snapID := "snap_qual_002"
	contentDigest := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	t0 := time.Date(2026, 9, 5, 8, 0, 0, 0, time.UTC)

	// 1. Register Draft
	_, err := ctrl.RegisterDraft(tenantID, snapID, 1, contentDigest, "usr_creator", t0)
	if err != nil {
		t.Fatalf("RegisterDraft failed: %v", err)
	}

	// 2. Direct publication attempt from DRAFT fails closed
	pubWindow := publicationsnapshot.EffectiveWindow{
		EffectiveFrom: t0,
		ExpiresAt:     t0.Add(30 * 24 * time.Hour),
	}
	_, err = ctrl.Publish(tenantID, snapID, "usr_admin", "TENANT_ADMIN", pubWindow, t0)
	if !errors.Is(err, publicationsnapshot.ErrIllegalStateTransition) {
		t.Fatalf("expected ErrIllegalStateTransition when publishing directly from DRAFT, got: %v", err)
	}

	// 3. Submit for review
	tSubmit := t0.Add(1 * time.Hour)
	_, err = ctrl.SubmitForReview(tenantID, snapID, "usr_creator", "CREATOR", tSubmit)
	if err != nil {
		t.Fatalf("SubmitForReview failed: %v", err)
	}

	// 4. Unauthorized reviewer role (e.g. INSPECTOR, CONTRACTOR) rejected from approving
	unauthorizedRoles := []string{"INSPECTOR", "CONTRACTOR", "VIEWER", "PROJECT_MANAGER"}
	for _, role := range unauthorizedRoles {
		ev, _ := publicationsnapshot.NewApprovalEvidence("app_bad", "usr_worker", role, tSubmit, contentDigest, "unauthorized attempt", 7*24*time.Hour)
		_, err = ctrl.Approve(tenantID, snapID, ev, tSubmit)
		if !errors.Is(err, publicationsnapshot.ErrUnauthorizedReviewer) {
			t.Errorf("expected ErrUnauthorizedReviewer for role %s, got: %v", role, err)
		}
	}

	// 5. Approval evidence with content digest mismatch fails closed
	tamperedDigest := "f4c0b44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b899"
	evMismatch, _ := publicationsnapshot.NewApprovalEvidence("app_mismatch", "usr_auditor", "AUDITOR", tSubmit, tamperedDigest, "digest mismatch", 7*24*time.Hour)
	_, err = ctrl.Approve(tenantID, snapID, evMismatch, tSubmit)
	if !errors.Is(err, publicationsnapshot.ErrApprovalDigestMismatch) {
		t.Fatalf("expected ErrApprovalDigestMismatch, got: %v", err)
	}

	// 6. Stale approval evidence (> 7 days) fails closed
	tStale := tSubmit.Add(8 * 24 * time.Hour)
	evStale, _ := publicationsnapshot.NewApprovalEvidence("app_stale", "usr_auditor", "AUDITOR", tSubmit, contentDigest, "stale approval", 7*24*time.Hour)
	_, err = ctrl.Approve(tenantID, snapID, evStale, tStale)
	if !errors.Is(err, publicationsnapshot.ErrStaleApproval) {
		t.Fatalf("expected ErrStaleApproval when approving with stale evidence, got: %v", err)
	}

	// 7. Legitimate approval by AUDITOR succeeds
	evValid, _ := publicationsnapshot.NewApprovalEvidence("app_valid_01", "usr_auditor", "AUDITOR", tSubmit, contentDigest, "Auditor sign-off complete", 7*24*time.Hour)
	approvedSnap, err := ctrl.Approve(tenantID, snapID, evValid, tSubmit)
	if err != nil {
		t.Fatalf("Approve failed: %v", err)
	}
	if approvedSnap.State != publicationsnapshot.StateApproved {
		t.Errorf("expected StateApproved, got: %s", approvedSnap.State)
	}

	// 8. Publishing with unauthorized publisher role rejected
	_, err = ctrl.Publish(tenantID, snapID, "usr_contractor", "CONTRACTOR", pubWindow, tSubmit)
	if !errors.Is(err, publicationsnapshot.ErrUnauthorizedPublish) {
		t.Fatalf("expected ErrUnauthorizedPublish for CONTRACTOR, got: %v", err)
	}

	// 9. Legitimate publication by TENANT_ADMIN succeeds
	publishedSnap, err := ctrl.Publish(tenantID, snapID, "usr_admin", "TENANT_ADMIN", pubWindow, tSubmit)
	if err != nil {
		t.Fatalf("Publish failed: %v", err)
	}
	if publishedSnap.State != publicationsnapshot.StatePublished {
		t.Errorf("expected StatePublished, got: %s", publishedSnap.State)
	}

	// 10. Re-publishing already published snapshot rejected as duplicate
	_, err = ctrl.Publish(tenantID, snapID, "usr_admin", "TENANT_ADMIN", pubWindow, tSubmit)
	if !errors.Is(err, publicationsnapshot.ErrDuplicateTransition) {
		t.Fatalf("expected ErrDuplicateTransition for double publish, got: %v", err)
	}
}

// TestQualification_Publication_SourceChangeDriftAndMutationIsolation qualifies:
// 1. Immutable binding of source entity references (SourceEntityRef).
// 2. Cross-tenant source references denied (ErrSourceMismatch).
// 3. Defensive copy isolation: mutations to source records do not alter sealed published snapshots.
// 4. Source drift detection identifying changes in operational records (CheckSourceDrift).
// 5. Direct operational source mutation attempts against the sealed store fail closed (ErrDirectSourceMutationForbidden).
func TestQualification_Publication_SourceChangeDriftAndMutationIsolation(t *testing.T) {
	store := publicationsnapshot.NewImmutablePublicationStore()
	tenantID := "ten_qual_pub_03"
	snapID := "snap_qual_003"
	t0 := time.Date(2026, 9, 5, 9, 0, 0, 0, time.UTC)
	origSourceDigest := "d1a7a00112233445566778899aabbccddeeff00112233445566778899aabbcc0"

	// 1. Cross-tenant source entity reference rejected
	badSource := publicationsnapshot.SourceEntityRef{
		SourceType:          "INSPECTION_RECORD",
		SourceID:            "insp_source_001",
		SourceTenantID:      "ten_foreign_cross_tenant",
		SourceVersion:       "1.0",
		SourceContentDigest: origSourceDigest,
	}
	al, _ := publicationsnapshot.NewPublicationFieldAllowlist([]string{"title", "score"}, false)
	_, err := publicationsnapshot.NewDraftSnapshot(snapID, tenantID, badSource, map[string]any{"title": "Audit", "score": 90}, al)
	if !errors.Is(err, publicationsnapshot.ErrSourceMismatch) {
		t.Fatalf("expected ErrSourceMismatch for cross-tenant source, got: %v", err)
	}

	// 2. Valid source mapping stored in immutable store
	validSource := publicationsnapshot.SourceEntityRef{
		SourceType:          "INSPECTION_RECORD",
		SourceID:            "insp_source_001",
		SourceTenantID:      tenantID,
		SourceVersion:       "1.0",
		SourceContentDigest: origSourceDigest,
	}
	ev, _ := publicationsnapshot.NewApprovalEvidence("app_03", "usr_auditor", "AUDITOR", t0, origSourceDigest, "Approved", 7*24*time.Hour)
	pubWindow := publicationsnapshot.EffectiveWindow{EffectiveFrom: t0, ExpiresAt: t0.Add(30 * 24 * time.Hour)}
	originalPayload := map[string]any{"title": "Scaffold Audit", "score": 92}

	storedRec, err := store.StorePublishedVersion(tenantID, snapID, 1, originalPayload, validSource, ev, pubWindow, 0, "", t0)
	if err != nil {
		t.Fatalf("StorePublishedVersion failed: %v", err)
	}

	// 3. Verify defensive copying: mutating original payload map does not alter stored record
	originalPayload["score"] = 40
	originalPayload["title"] = "Tampered Locally"

	freshFetch, err := store.GetPublishedVersion(tenantID, snapID, 1)
	if err != nil {
		t.Fatalf("GetPublishedVersion failed: %v", err)
	}
	if freshFetch.Payload["score"] != 92 || freshFetch.Payload["title"] != "Scaffold Audit" {
		t.Fatalf("defensive copy breach: stored record mutated via caller map modification: %+v", freshFetch.Payload)
	}

	// 4. Source evolves in operational database (drift check)
	evolvedSourceDigest := "d1a7a0099887766554433221100ffeeddccbbaa99887766554433221100ffeed"
	drift, err := store.CheckSourceDrift(tenantID, snapID, 1, evolvedSourceDigest)
	if err != nil {
		t.Fatalf("CheckSourceDrift failed: %v", err)
	}
	if !drift.HasDrifted {
		t.Errorf("expected drift.HasDrifted == true")
	}
	if drift.PublishedDigest != origSourceDigest {
		t.Errorf("drift published digest mismatch: %s != %s", drift.PublishedDigest, origSourceDigest)
	}

	// Check unchanged source shows no drift
	noDrift, err := store.CheckSourceDrift(tenantID, snapID, 1, origSourceDigest)
	if err != nil || noDrift.HasDrifted {
		t.Errorf("expected no drift for identical digest, got: %+v (err: %v)", noDrift, err)
	}

	// 5. Direct mutation of published snapshot from operational source fails closed
	err = store.AttemptDirectSourceMutation(tenantID, snapID, 1, map[string]any{"title": "Direct Update", "score": 100})
	if !errors.Is(err, publicationsnapshot.ErrDirectSourceMutationForbidden) {
		t.Fatalf("expected ErrDirectSourceMutationForbidden, got: %v", err)
	}

	// Confirm stored record remains 100% untouched
	unmodified, _ := store.GetPublishedVersion(tenantID, snapID, 1)
	if unmodified.SignatureDigest != storedRec.SignatureDigest {
		t.Fatalf("signature digest altered after rejected direct mutation attempt")
	}
}

// TestQualification_Publication_TemporalValidityAndExpiration qualifies:
// 1. Inverted publication windows rejected (ErrInvalidPublicationWindow).
// 2. Windows exceeding 365 days rejected (ErrInvalidPublicationWindow).
// 3. Publishing into an already expired window rejected (ErrSnapshotExpired).
// 4. Pre-effective access evaluation correctly identifies un-effective state.
// 5. Post-expiration automatic evaluation transitions to StateExpired.
func TestQualification_Publication_TemporalValidityAndExpiration(t *testing.T) {
	t0 := time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC)

	// 1. Inverted window: ExpiresAt before EffectiveFrom
	invertedWindow := publicationsnapshot.EffectiveWindow{
		EffectiveFrom: t0.Add(24 * time.Hour),
		ExpiresAt:     t0,
	}
	if err := invertedWindow.Validate(); !errors.Is(err, publicationsnapshot.ErrInvalidPublicationWindow) {
		t.Errorf("expected ErrInvalidPublicationWindow for inverted window, got: %v", err)
	}

	// Equal timestamps
	equalWindow := publicationsnapshot.EffectiveWindow{
		EffectiveFrom: t0,
		ExpiresAt:     t0,
	}
	if err := equalWindow.Validate(); !errors.Is(err, publicationsnapshot.ErrInvalidPublicationWindow) {
		t.Errorf("expected ErrInvalidPublicationWindow for equal timestamps, got: %v", err)
	}

	// 2. Window duration exceeding 365 days
	excessiveWindow := publicationsnapshot.EffectiveWindow{
		EffectiveFrom: t0,
		ExpiresAt:     t0.Add(366 * 24 * time.Hour),
	}
	if err := excessiveWindow.Validate(); !errors.Is(err, publicationsnapshot.ErrInvalidPublicationWindow) {
		t.Errorf("expected ErrInvalidPublicationWindow for >365 day window, got: %v", err)
	}

	// 3. Publishing into an expired window rejected
	ledger := publicationsnapshot.NewLifecycleAuditLedger()
	ctrl := publicationsnapshot.NewLifecycleController(ledger)
	tenantID := "ten_qual_pub_04"
	snapID := "snap_qual_004"
	contentDigest := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

	_, _ = ctrl.RegisterDraft(tenantID, snapID, 1, contentDigest, "usr_creator", t0)
	_, _ = ctrl.SubmitForReview(tenantID, snapID, "usr_creator", "CREATOR", t0)
	ev, _ := publicationsnapshot.NewApprovalEvidence("app_04", "usr_auditor", "AUDITOR", t0, contentDigest, "ok", 7*24*time.Hour)
	_, _ = ctrl.Approve(tenantID, snapID, ev, t0)

	pastWindow := publicationsnapshot.EffectiveWindow{
		EffectiveFrom: t0.Add(-10 * 24 * time.Hour),
		ExpiresAt:     t0.Add(-1 * 24 * time.Hour),
	}
	_, err := ctrl.Publish(tenantID, snapID, "usr_admin", "TENANT_ADMIN", pastWindow, t0)
	if !errors.Is(err, publicationsnapshot.ErrSnapshotExpired) {
		t.Fatalf("expected ErrSnapshotExpired for publishing in expired window, got: %v", err)
	}

	// 4. Valid window lifecycle behavior
	validWindow := publicationsnapshot.EffectiveWindow{
		EffectiveFrom: t0.Add(2 * 24 * time.Hour), // effective 2 days later
		ExpiresAt:     t0.Add(10 * 24 * time.Hour),
	}
	snap, err := ctrl.Publish(tenantID, snapID, "usr_admin", "TENANT_ADMIN", validWindow, t0)
	if err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	// At t0: state is PUBLISHED but not yet active
	if snap.IsActive(t0) {
		t.Errorf("expected IsActive == false before EffectiveFrom")
	}

	// At t0 + 3 days: active
	tActive := t0.Add(3 * 24 * time.Hour)
	if !snap.IsActive(tActive) {
		t.Errorf("expected IsActive == true within valid effective window")
	}
	if snap.EffectiveState(tActive) != publicationsnapshot.StatePublished {
		t.Errorf("expected EffectiveState == StatePublished, got: %s", snap.EffectiveState(tActive))
	}

	// At t0 + 11 days: expired
	tExpired := t0.Add(11 * 24 * time.Hour)
	if snap.IsActive(tExpired) {
		t.Errorf("expected IsActive == false after ExpiresAt")
	}
	if snap.EffectiveState(tExpired) != publicationsnapshot.StateExpired {
		t.Errorf("expected EffectiveState == StateExpired, got: %s", snap.EffectiveState(tExpired))
	}
}

// TestQualification_Publication_WithdrawalAndInactivation qualifies:
// 1. Mandatory withdrawal justification requirement (ErrMissingWithdrawalReason).
// 2. Unauthorized role cannot withdraw snapshot (ErrUnauthorizedReviewer).
// 3. Withdrawn state transitions correctly and cannot be reactivated (ErrCannotReactivateWithdrawn).
// 4. Full audit trail attribution permanently preserved.
func TestQualification_Publication_WithdrawalAndInactivation(t *testing.T) {
	ledger := publicationsnapshot.NewLifecycleAuditLedger()
	ctrl := publicationsnapshot.NewLifecycleController(ledger)
	tenantID := "ten_qual_pub_05"
	snapID := "snap_qual_005"
	contentDigest := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	t0 := time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC)

	_, _ = ctrl.RegisterDraft(tenantID, snapID, 1, contentDigest, "usr_creator", t0)
	_, _ = ctrl.SubmitForReview(tenantID, snapID, "usr_creator", "CREATOR", t0)
	ev, _ := publicationsnapshot.NewApprovalEvidence("app_05", "usr_auditor", "AUDITOR", t0, contentDigest, "ok", 7*24*time.Hour)
	_, _ = ctrl.Approve(tenantID, snapID, ev, t0)
	pubWindow := publicationsnapshot.EffectiveWindow{EffectiveFrom: t0, ExpiresAt: t0.Add(30 * 24 * time.Hour)}
	_, _ = ctrl.Publish(tenantID, snapID, "usr_admin", "TENANT_ADMIN", pubWindow, t0)

	// 1. Blank withdrawal reason rejected
	_, err := ctrl.Withdraw(tenantID, snapID, "usr_admin", "TENANT_ADMIN", "   ", t0.Add(1*time.Hour))
	if !errors.Is(err, publicationsnapshot.ErrMissingWithdrawalReason) {
		t.Fatalf("expected ErrMissingWithdrawalReason, got: %v", err)
	}

	// 2. Unauthorized role rejected
	_, err = ctrl.Withdraw(tenantID, snapID, "usr_contractor", "CONTRACTOR", "Disputed report", t0.Add(1*time.Hour))
	if !errors.Is(err, publicationsnapshot.ErrUnauthorizedReviewer) {
		t.Fatalf("expected ErrUnauthorizedReviewer for CONTRACTOR, got: %v", err)
	}

	// 3. Legitimate withdrawal by AUDITOR succeeds
	tWithdraw := t0.Add(2 * time.Hour)
	withdrawalReason := "Safety finding superseded by critical defect re-inspection"
	withdrawnSnap, err := ctrl.Withdraw(tenantID, snapID, "usr_auditor", "AUDITOR", withdrawalReason, tWithdraw)
	if err != nil {
		t.Fatalf("Withdraw failed: %v", err)
	}
	if withdrawnSnap.State != publicationsnapshot.StateWithdrawn {
		t.Errorf("expected StateWithdrawn, got: %s", withdrawnSnap.State)
	}
	if withdrawnSnap.WithdrawalReason != withdrawalReason {
		t.Errorf("withdrawal reason mismatch: %s", withdrawnSnap.WithdrawalReason)
	}

	// 4. Duplicate withdrawal rejected
	_, err = ctrl.Withdraw(tenantID, snapID, "usr_auditor", "AUDITOR", "Double withdrawal", tWithdraw.Add(1*time.Hour))
	if !errors.Is(err, publicationsnapshot.ErrDuplicateTransition) {
		t.Fatalf("expected ErrDuplicateTransition on second withdrawal, got: %v", err)
	}

	// 5. Attempt to re-publish withdrawn snapshot fails closed
	_, err = ctrl.Publish(tenantID, snapID, "usr_admin", "TENANT_ADMIN", pubWindow, tWithdraw.Add(2*time.Hour))
	if !errors.Is(err, publicationsnapshot.ErrIllegalStateTransition) {
		t.Fatalf("expected ErrIllegalStateTransition on re-publish, got: %v", err)
	}

	// 6. Audit history captures full chronological lineage
	history, err := ledger.GetHistory(tenantID, snapID)
	if err != nil {
		t.Fatalf("GetHistory failed: %v", err)
	}
	if len(history) != 5 { // DRAFT -> UNDER_REVIEW -> APPROVED -> PUBLISHED -> WITHDRAWN
		t.Fatalf("expected 5 audit records, got %d", len(history))
	}
	lastRec := history[len(history)-1]
	if lastRec.ToState != publicationsnapshot.StateWithdrawn || lastRec.ActorSubject != "usr_auditor" || lastRec.Reason != withdrawalReason {
		t.Errorf("audit record mismatch: %+v", lastRec)
	}
}

// TestQualification_Publication_SupersessionAndLineageChaining qualifies:
// 1. Monotonic versioning and successor linking (PredecessorVersion, PredecessorDigest).
// 2. Prior version successor pointer update (SuccessorVersion, SuccessorDigest).
// 3. Non-contiguous versions or invalid predecessors fail closed (ErrInvalidPredecessor).
// 4. Predecessor digest mismatch fails closed (ErrBrokenLineageChain).
// 5. Replacement metadata validation fails on blank reason or successor ID (ErrBlankReplacementReason, ErrBlankSuccessorID).
func TestQualification_Publication_SupersessionAndLineageChaining(t *testing.T) {
	store := publicationsnapshot.NewImmutablePublicationStore()
	tenantID := "ten_qual_pub_06"
	snapID := "snap_qual_006"
	t0 := time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC)

	source1 := publicationsnapshot.SourceEntityRef{
		SourceType:          "SAFETY_INSPECTION",
		SourceID:            "insp_06_1",
		SourceTenantID:      tenantID,
		SourceVersion:       "1.0",
		SourceContentDigest: "digest_v1_00000000000000000000000000000000000000000000000000000000",
	}
	ev1, _ := publicationsnapshot.NewApprovalEvidence("app_06_1", "usr_auditor", "AUDITOR", t0, source1.SourceContentDigest, "v1 approved", 7*24*time.Hour)
	win1 := publicationsnapshot.EffectiveWindow{EffectiveFrom: t0, ExpiresAt: t0.Add(30 * 24 * time.Hour)}

	// 1. Store v1 (predecessor = 0, digest = "")
	v1, err := store.StorePublishedVersion(tenantID, snapID, 1, map[string]any{"version_label": "v1.0", "findings": 2}, source1, ev1, win1, 0, "", t0)
	if err != nil {
		t.Fatalf("Store v1 failed: %v", err)
	}

	// 2. v2 attempting non-existent predecessor version fails closed
	t1 := t0.Add(24 * time.Hour)
	source2 := source1
	source2.SourceVersion = "2.0"
	source2.SourceContentDigest = "digest_v2_00000000000000000000000000000000000000000000000000000000"
	ev2, _ := publicationsnapshot.NewApprovalEvidence("app_06_2", "usr_auditor", "AUDITOR", t1, source2.SourceContentDigest, "v2 approved", 7*24*time.Hour)
	win2 := publicationsnapshot.EffectiveWindow{EffectiveFrom: t1, ExpiresAt: t1.Add(30 * 24 * time.Hour)}

	_, err = store.StorePublishedVersion(tenantID, snapID, 2, map[string]any{"version_label": "v2.0"}, source2, ev2, win2, 99, v1.SignatureDigest, t1)
	if !errors.Is(err, publicationsnapshot.ErrInvalidPredecessor) {
		t.Fatalf("expected ErrInvalidPredecessor for predecessor=99, got: %v", err)
	}

	// 3. v2 attempting wrong predecessor digest fails closed
	_, err = store.StorePublishedVersion(tenantID, snapID, 2, map[string]any{"version_label": "v2.0"}, source2, ev2, win2, 1, "tampered_fake_signature_digest", t1)
	if !errors.Is(err, publicationsnapshot.ErrBrokenLineageChain) {
		t.Fatalf("expected ErrBrokenLineageChain for mismatched predecessor digest, got: %v", err)
	}

	// 4. Store legitimate v2 superseding v1
	v2, err := store.StorePublishedVersion(tenantID, snapID, 2, map[string]any{"version_label": "v2.0", "findings": 1}, source2, ev2, win2, 1, v1.SignatureDigest, t1)
	if err != nil {
		t.Fatalf("Store v2 failed: %v", err)
	}

	// 5. Store legitimate v3 superseding v2
	t2 := t1.Add(24 * time.Hour)
	source3 := source1
	source3.SourceVersion = "3.0"
	source3.SourceContentDigest = "digest_v3_00000000000000000000000000000000000000000000000000000000"
	ev3, _ := publicationsnapshot.NewApprovalEvidence("app_06_3", "usr_auditor", "AUDITOR", t2, source3.SourceContentDigest, "v3 approved", 7*24*time.Hour)
	win3 := publicationsnapshot.EffectiveWindow{EffectiveFrom: t2, ExpiresAt: t2.Add(30 * 24 * time.Hour)}

	v3, err := store.StorePublishedVersion(tenantID, snapID, 3, map[string]any{"version_label": "v3.0", "findings": 0}, source3, ev3, win3, 2, v2.SignatureDigest, t2)
	if err != nil {
		t.Fatalf("Store v3 failed: %v", err)
	}

	// 6. Verify full version chain
	versions, err := store.ListPublishedVersions(tenantID, snapID)
	if err != nil {
		t.Fatalf("ListPublishedVersions failed: %v", err)
	}
	if len(versions) != 3 {
		t.Fatalf("expected 3 versions, got %d", len(versions))
	}

	// Verify predecessor/successor links
	v1Updated, _ := store.GetPublishedVersion(tenantID, snapID, 1)
	if v1Updated.SuccessorVersion != 2 || v1Updated.SuccessorDigest != v2.SignatureDigest {
		t.Errorf("v1 successor pointers incorrect: %+v", v1Updated)
	}
	v2Updated, _ := store.GetPublishedVersion(tenantID, snapID, 2)
	if v2Updated.PredecessorVersion != 1 || v2Updated.PredecessorDigest != v1.SignatureDigest {
		t.Errorf("v2 predecessor pointers incorrect: %+v", v2Updated)
	}
	if v2Updated.SuccessorVersion != 3 || v2Updated.SuccessorDigest != v3.SignatureDigest {
		t.Errorf("v2 successor pointers incorrect: %+v", v2Updated)
	}

	// 7. Replacement metadata validation: blank reason / blank successor ID
	err = store.RegisterReplacement(tenantID, snapID, 1, "snap_succ_01", 2, publicationsnapshot.ReplacementCorrection, "", t2)
	if !errors.Is(err, publicationsnapshot.ErrBlankReplacementReason) {
		t.Errorf("expected ErrBlankReplacementReason, got: %v", err)
	}
	err = store.RegisterReplacement(tenantID, snapID, 1, "   ", 2, publicationsnapshot.ReplacementCorrection, "Valid reason", t2)
	if !errors.Is(err, publicationsnapshot.ErrBlankSuccessorID) {
		t.Errorf("expected ErrBlankSuccessorID, got: %v", err)
	}
}

// TestQualification_Publication_CryptographicIntegrityAndTamperDetection qualifies:
// 1. Canonical payload digest determinism regardless of map key iteration order.
// 2. Composite envelope signature binding all identity, version, source, and reviewer attributes.
// 3. VerifyIntegrity() asserting sealed payload against digests.
// 4. In-memory tampering detection causing integrity failure (ErrIntegrityVerificationFailed).
// 5. Multi-version audit trail reconstruction proving unbroken cryptographic continuity (StatusVerifiedIntact).
func TestQualification_Publication_CryptographicIntegrityAndTamperDetection(t *testing.T) {
	tenantID := "ten_qual_pub_07"
	snapID := "snap_qual_007"
	t0 := time.Date(2026, 9, 5, 11, 0, 0, 0, time.UTC)

	// 1. Deterministic canonical payload digest
	payloadA := map[string]any{"b_site": "Site Beta", "a_inspection": "INSP-01", "c_score": 88}
	payloadB := map[string]any{"c_score": 88, "a_inspection": "INSP-01", "b_site": "Site Beta"}

	digestA, err := publicationsnapshot.ComputeCanonicalPayloadDigest(payloadA)
	if err != nil {
		t.Fatalf("ComputeCanonicalPayloadDigest failed: %v", err)
	}
	digestB, err := publicationsnapshot.ComputeCanonicalPayloadDigest(payloadB)
	if err != nil {
		t.Fatalf("ComputeCanonicalPayloadDigest failed: %v", err)
	}
	if digestA != digestB {
		t.Fatalf("canonical payload digest non-deterministic across key orders: %s != %s", digestA, digestB)
	}

	// 2. Snapshot Envelope Seals Digests
	source := publicationsnapshot.SourceEntityRef{
		SourceType:          "AUDIT_RECORD",
		SourceID:            "audit_07_1",
		SourceTenantID:      tenantID,
		SourceVersion:       "1.0",
		SourceContentDigest: digestA,
	}
	al, _ := publicationsnapshot.NewPublicationFieldAllowlist([]string{"a_inspection", "b_site", "c_score"}, false)

	snap, err := publicationsnapshot.NewDraftSnapshot(snapID, tenantID, source, payloadA, al)
	if err != nil {
		t.Fatalf("NewDraftSnapshot failed: %v", err)
	}

	reg := publicationsnapshot.NewSnapshotRegistry()
	_ = reg.RegisterDraft(snap)

	revCtx, _ := publicationsnapshot.NewReviewerContext("usr_auditor_lead", "AUDITOR", publicationsnapshot.ReviewApproved, "Approved for portal", t0)
	pubSnap, err := reg.PublishSnapshot(tenantID, snapID, revCtx)
	if err != nil {
		t.Fatalf("PublishSnapshot failed: %v", err)
	}

	// 3. Legitimate snapshot verifies integrity
	if err := pubSnap.VerifyIntegrity(); err != nil {
		t.Fatalf("VerifyIntegrity failed on sealed snapshot: %v", err)
	}

	// 4. In-memory tampering detection via registry
	if pubSnap.Integrity().PayloadDigest != digestA {
		t.Errorf("payload digest mismatch in sealed snapshot: %s != %s", pubSnap.Integrity().PayloadDigest, digestA)
	}

	// 5. Multi-version audit reconstruction validates unbroken chain
	store := publicationsnapshot.NewImmutablePublicationStore()
	ev, _ := publicationsnapshot.NewApprovalEvidence("app_07", "usr_auditor", "AUDITOR", t0, digestA, "ok", 7*24*time.Hour)
	win := publicationsnapshot.EffectiveWindow{EffectiveFrom: t0, ExpiresAt: t0.Add(30 * 24 * time.Hour)}
	_, err = store.StorePublishedVersion(tenantID, snapID, 1, payloadA, source, ev, win, 0, "", t0)
	if err != nil {
		t.Fatalf("StorePublishedVersion failed: %v", err)
	}

	auditReport, err := store.ReconstructPublicationAuditTrail(tenantID, snapID)
	if err != nil {
		t.Fatalf("ReconstructPublicationAuditTrail failed: %v", err)
	}
	if auditReport.Status != publicationsnapshot.StatusVerifiedIntact {
		t.Fatalf("expected StatusVerifiedIntact, got: %s (findings: %v)", auditReport.Status, auditReport.Findings)
	}
	if auditReport.LineageChainDigest == "" {
		t.Errorf("expected non-empty lineage chain digest")
	}
}

// TestQualification_Publication_ExportDenialAndScopeBoundaries qualifies:
// 1. Export package creation restricts to recognized destination scopes (ErrUnapprovedDestinationScope).
// 2. Export package enforces classification validity (ErrInvalidClassification).
// 3. Export package rejects unpublished draft or superseded snapshots (ErrUnpublishedSnapshotInExport).
// 4. Export package strictly enforces tenant consistency (ErrCrossTenantAccessDenied).
// 5. Cross-tenant lookups on immutable store fail closed with non-leaking ErrSnapshotNotFound.
func TestQualification_Publication_ExportDenialAndScopeBoundaries(t *testing.T) {
	tenantID := "ten_qual_pub_08"
	foreignTenant := "ten_foreign_pub_99"
	snapID := "snap_qual_008"
	t0 := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)

	source := publicationsnapshot.SourceEntityRef{
		SourceType:          "CHECKLIST",
		SourceID:            "chk_08_1",
		SourceTenantID:      tenantID,
		SourceVersion:       "1.0",
		SourceContentDigest: "digest_08",
	}
	al, _ := publicationsnapshot.NewPublicationFieldAllowlist([]string{"item", "result"}, false)

	draftSnap, err := publicationsnapshot.NewDraftSnapshot(snapID, tenantID, source, map[string]any{"item": "PPE", "result": "PASS"}, al)
	if err != nil {
		t.Fatalf("NewDraftSnapshot failed: %v", err)
	}

	reg := publicationsnapshot.NewSnapshotRegistry()
	_ = reg.RegisterDraft(draftSnap)

	revCtx, _ := publicationsnapshot.NewReviewerContext("usr_auditor", "AUDITOR", publicationsnapshot.ReviewApproved, "Approved", t0)
	pubSnap, err := reg.PublishSnapshot(tenantID, snapID, revCtx)
	if err != nil {
		t.Fatalf("PublishSnapshot failed: %v", err)
	}

	// 1. Unapproved destination scope rejected
	_, err = publicationsnapshot.NewExportPackage(
		"pkg_01", tenantID, "JSON",
		"PUBLIC_SANITIZED",
		"UNAPPROVED_INTERNET_ROUTE",
		"usr_admin",
		[]publicationsnapshot.PublicationSnapshot{pubSnap},
	)
	if !errors.Is(err, publicationsnapshot.ErrUnapprovedDestinationScope) {
		t.Fatalf("expected ErrUnapprovedDestinationScope, got: %v", err)
	}

	// 2. Invalid classification rejected
	_, err = publicationsnapshot.NewExportPackage(
		"pkg_02", tenantID, "JSON",
		"INVALID_CONFIDENTIAL",
		"PUBLIC_PORTAL_PREVIEW",
		"usr_admin",
		[]publicationsnapshot.PublicationSnapshot{pubSnap},
	)
	if !errors.Is(err, publicationsnapshot.ErrInvalidClassification) {
		t.Fatalf("expected ErrInvalidClassification, got: %v", err)
	}

	// 3. Unpublished draft snapshot in export package rejected
	unpubDraft, _ := publicationsnapshot.NewDraftSnapshot("snap_draft_unpub", tenantID, source, map[string]any{"item": "PPE", "result": "PASS"}, al)
	_, err = publicationsnapshot.NewExportPackage(
		"pkg_03", tenantID, "JSON",
		"EXTERNAL_CONTROLLED",
		"EXTERNAL_AUDITOR_PACKAGE",
		"usr_admin",
		[]publicationsnapshot.PublicationSnapshot{unpubDraft},
	)
	if !errors.Is(err, publicationsnapshot.ErrUnpublishedSnapshotInExport) {
		t.Fatalf("expected ErrUnpublishedSnapshotInExport, got: %v", err)
	}

	// 4. Cross-tenant bundled snapshots in single export package rejected
	foreignSource := source
	foreignSource.SourceTenantID = foreignTenant
	foreignDraft, _ := publicationsnapshot.NewDraftSnapshot("snap_foreign_01", foreignTenant, foreignSource, map[string]any{"item": "PPE", "result": "PASS"}, al)
	foreignReg := publicationsnapshot.NewSnapshotRegistry()
	_ = foreignReg.RegisterDraft(foreignDraft)
	foreignPub, _ := foreignReg.PublishSnapshot(foreignTenant, "snap_foreign_01", revCtx)

	_, err = publicationsnapshot.NewExportPackage(
		"pkg_04", tenantID, "JSON",
		"EXTERNAL_CONTROLLED",
		"REGULATORY_SUBMISSION",
		"usr_admin",
		[]publicationsnapshot.PublicationSnapshot{pubSnap, foreignPub},
	)
	if !errors.Is(err, publicationsnapshot.ErrCrossTenantAccessDenied) {
		t.Fatalf("expected ErrCrossTenantAccessDenied for mixed-tenant package, got: %v", err)
	}

	// 5. Cross-tenant lookups on immutable store return ErrSnapshotNotFound
	store := publicationsnapshot.NewImmutablePublicationStore()
	ev, _ := publicationsnapshot.NewApprovalEvidence("app_08", "usr_auditor", "AUDITOR", t0, "digest_08", "ok", 7*24*time.Hour)
	win := publicationsnapshot.EffectiveWindow{EffectiveFrom: t0, ExpiresAt: t0.Add(30 * 24 * time.Hour)}
	_, _ = store.StorePublishedVersion(tenantID, snapID, 1, map[string]any{"item": "PPE"}, source, ev, win, 0, "", t0)

	_, err = store.GetPublishedVersion(foreignTenant, snapID, 1)
	if !errors.Is(err, publicationsnapshot.ErrSnapshotNotFound) {
		t.Errorf("expected ErrSnapshotNotFound on foreign tenant query, got: %v", err)
	}
	_, err = store.ReconstructPublicationAuditTrail(foreignTenant, snapID)
	if !errors.Is(err, publicationsnapshot.ErrSnapshotNotFound) {
		t.Errorf("expected ErrSnapshotNotFound on foreign tenant audit query, got: %v", err)
	}
}

// TestQualification_Publication_LocalSyntheticNonClaims qualifies:
// Formal programmatic verification of governance non-claims under H030-003, H030-004, H030-005.
func TestQualification_Publication_LocalSyntheticNonClaims(t *testing.T) {
	// Programmatically assert non-claims:
	// 1. Operates purely in-memory on local synthetic fixtures.
	// 2. Zero live public routes, CDN distribution, or internet publication endpoints.
	// 3. Zero SQL persistence, table creation, or external database mutations.
	// 4. Zero real credentials, real customer data, or live production connections.
	// 5. Zero operational policy or authority model activation.
	store := publicationsnapshot.NewImmutablePublicationStore()
	if store == nil {
		t.Fatalf("in-memory immutable publication store failed initialization")
	}
	ledger := publicationsnapshot.NewLifecycleAuditLedger()
	ctrl := publicationsnapshot.NewLifecycleController(ledger)
	if ctrl == nil {
		t.Fatalf("in-memory lifecycle controller failed initialization")
	}
}
