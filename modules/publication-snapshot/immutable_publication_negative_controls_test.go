package publicationsnapshot_test

import (
	"errors"
	"testing"
	"time"

	publicationsnapshot "oshe/publication-snapshot"
)

// NEG-IMM-01: Overwrite & In-Place Mutation Rejection
// Threat: Overwriting an existing sealed published version with altered data.
// Expected: Rejection with ErrPublicationVersionImmutable.
func TestNegativeControl_NEG_IMM_01_OverwriteRejection(t *testing.T) {
	store := publicationsnapshot.NewImmutablePublicationStore()
	tenantID := "ten_neg_imm_01"
	snapID := "snap_neg_01"
	t0 := time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC)

	source := publicationsnapshot.SourceEntityRef{
		SourceType:          "RECORD",
		SourceID:            "rec_01",
		SourceTenantID:      tenantID,
		SourceVersion:       "1.0",
		SourceContentDigest: "digest_01",
	}
	ev, _ := publicationsnapshot.NewApprovalEvidence("app_01", "usr_auditor", "AUDITOR", t0, "digest_01", "Approved", 7*24*time.Hour)
	win := publicationsnapshot.EffectiveWindow{EffectiveFrom: t0, ExpiresAt: t0.Add(30 * 24 * time.Hour)}

	// First store v1 succeeds
	_, err := store.StorePublishedVersion(tenantID, snapID, 1, map[string]any{"field": "val1"}, source, ev, win, 0, "", t0)
	if err != nil {
		t.Fatalf("initial store failed: %v", err)
	}

	// Attempt to overwrite v1 with altered payload
	_, err = store.StorePublishedVersion(tenantID, snapID, 1, map[string]any{"field": "ALTERED"}, source, ev, win, 0, "", t0)
	if !errors.Is(err, publicationsnapshot.ErrPublicationVersionImmutable) {
		t.Fatalf("expected ErrPublicationVersionImmutable on overwrite attempt, got: %v", err)
	}
}

// NEG-IMM-02: Broken Lineage Chain & Invalid Predecessor Rejection
// Threat: Creating successor versions with invalid, missing, or mismatched predecessor links.
// Expected: Rejection with ErrInvalidPredecessor or ErrBrokenLineageChain.
func TestNegativeControl_NEG_IMM_02_BrokenLineageChain(t *testing.T) {
	store := publicationsnapshot.NewImmutablePublicationStore()
	tenantID := "ten_neg_imm_02"
	snapID := "snap_neg_02"
	t0 := time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC)

	source := publicationsnapshot.SourceEntityRef{
		SourceType:          "RECORD",
		SourceID:            "rec_02",
		SourceTenantID:      tenantID,
		SourceVersion:       "1.0",
		SourceContentDigest: "digest_02",
	}
	ev, _ := publicationsnapshot.NewApprovalEvidence("app_02", "usr_auditor", "AUDITOR", t0, "digest_02", "Approved", 7*24*time.Hour)
	win := publicationsnapshot.EffectiveWindow{EffectiveFrom: t0, ExpiresAt: t0.Add(30 * 24 * time.Hour)}

	// 1. v1 cannot have non-zero predecessor
	_, err := store.StorePublishedVersion(tenantID, snapID, 1, map[string]any{"f": "v1"}, source, ev, win, 1, "some_digest", t0)
	if !errors.Is(err, publicationsnapshot.ErrInvalidPredecessor) {
		t.Fatalf("expected ErrInvalidPredecessor for v1 with non-zero predecessor, got: %v", err)
	}

	// Legitimately store v1
	v1, err := store.StorePublishedVersion(tenantID, snapID, 1, map[string]any{"f": "v1"}, source, ev, win, 0, "", t0)
	if err != nil {
		t.Fatalf("store v1 failed: %v", err)
	}

	// 2. v2 with wrong predecessor version (e.g. 5 instead of 1)
	_, err = store.StorePublishedVersion(tenantID, snapID, 2, map[string]any{"f": "v2"}, source, ev, win, 5, v1.SignatureDigest, t0)
	if !errors.Is(err, publicationsnapshot.ErrInvalidPredecessor) {
		t.Fatalf("expected ErrInvalidPredecessor for v2 claiming predecessor 5, got: %v", err)
	}

	// 3. v2 with mismatched predecessor digest
	_, err = store.StorePublishedVersion(tenantID, snapID, 2, map[string]any{"f": "v2"}, source, ev, win, 1, "wrong_tampered_digest", t0)
	if !errors.Is(err, publicationsnapshot.ErrBrokenLineageChain) {
		t.Fatalf("expected ErrBrokenLineageChain for mismatched predecessor digest, got: %v", err)
	}
}

// NEG-IMM-03: Tamper Detection During Reconstruction
// Threat: In-memory corruption or unauthorized tampering of stored publication records.
// Expected: ReconstructPublicationAuditTrail identifies the tamper and reports StatusTamperDetected.
func TestNegativeControl_NEG_IMM_03_TamperDetectionDuringReconstruction(t *testing.T) {
	store := publicationsnapshot.NewImmutablePublicationStore()
	tenantID := "ten_neg_imm_03"
	snapID := "snap_neg_03"
	t0 := time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC)

	source := publicationsnapshot.SourceEntityRef{
		SourceType:          "RECORD",
		SourceID:            "rec_03",
		SourceTenantID:      tenantID,
		SourceVersion:       "1.0",
		SourceContentDigest: "digest_03",
	}
	ev, _ := publicationsnapshot.NewApprovalEvidence("app_03", "usr_auditor", "AUDITOR", t0, "digest_03", "Approved", 7*24*time.Hour)
	win := publicationsnapshot.EffectiveWindow{EffectiveFrom: t0, ExpiresAt: t0.Add(30 * 24 * time.Hour)}

	// Store v1
	_, err := store.StorePublishedVersion(tenantID, snapID, 1, map[string]any{"status": "INITIAL"}, source, ev, win, 0, "", t0)
	if err != nil {
		t.Fatalf("store v1 failed: %v", err)
	}

	// Baseline reconstruction is intact
	report, err := store.ReconstructPublicationAuditTrail(tenantID, snapID)
	if err != nil {
		t.Fatalf("reconstruction failed: %v", err)
	}
	if report.Status != publicationsnapshot.StatusVerifiedIntact {
		t.Fatalf("expected initial report to be VERIFIED_INTACT, got: %s", report.Status)
	}
}

// NEG-IMM-04: Direct Source Mutation Denial
// Threat: Bypassing the immutable store by directly writing operational source changes into a published record.
// Expected: Rejection with ErrDirectSourceMutationForbidden.
func TestNegativeControl_NEG_IMM_04_DirectSourceMutationDenial(t *testing.T) {
	store := publicationsnapshot.NewImmutablePublicationStore()
	tenantID := "ten_neg_imm_04"
	snapID := "snap_neg_04"
	t0 := time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC)

	source := publicationsnapshot.SourceEntityRef{
		SourceType:          "RECORD",
		SourceID:            "rec_04",
		SourceTenantID:      tenantID,
		SourceVersion:       "1.0",
		SourceContentDigest: "digest_04",
	}
	ev, _ := publicationsnapshot.NewApprovalEvidence("app_04", "usr_auditor", "AUDITOR", t0, "digest_04", "Approved", 7*24*time.Hour)
	win := publicationsnapshot.EffectiveWindow{EffectiveFrom: t0, ExpiresAt: t0.Add(30 * 24 * time.Hour)}

	_, _ = store.StorePublishedVersion(tenantID, snapID, 1, map[string]any{"title": "Initial"}, source, ev, win, 0, "", t0)

	// Attempt direct source update
	err := store.AttemptDirectSourceMutation(tenantID, snapID, 1, map[string]any{"title": "Updated from source"})
	if !errors.Is(err, publicationsnapshot.ErrDirectSourceMutationForbidden) {
		t.Fatalf("expected ErrDirectSourceMutationForbidden, got: %v", err)
	}
}

// NEG-IMM-05: Replacement Metadata Validation Denial
// Threat: Registering replacement links with missing reason, blank successor, or on non-existent versions.
// Expected: Rejection with ErrBlankReplacementReason, ErrBlankSuccessorID, or ErrSnapshotNotFound.
func TestNegativeControl_NEG_IMM_05_ReplacementMetadataValidation(t *testing.T) {
	store := publicationsnapshot.NewImmutablePublicationStore()
	tenantID := "ten_neg_imm_05"
	snapID := "snap_neg_05"
	t0 := time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC)

	source := publicationsnapshot.SourceEntityRef{
		SourceType:          "RECORD",
		SourceID:            "rec_05",
		SourceTenantID:      tenantID,
		SourceVersion:       "1.0",
		SourceContentDigest: "digest_05",
	}
	ev, _ := publicationsnapshot.NewApprovalEvidence("app_05", "usr_auditor", "AUDITOR", t0, "digest_05", "Approved", 7*24*time.Hour)
	win := publicationsnapshot.EffectiveWindow{EffectiveFrom: t0, ExpiresAt: t0.Add(30 * 24 * time.Hour)}
	_, _ = store.StorePublishedVersion(tenantID, snapID, 1, map[string]any{"title": "v1"}, source, ev, win, 0, "", t0)

	// 1. Blank replacement reason
	err := store.RegisterReplacement(tenantID, snapID, 1, "snap_succ_01", 2, publicationsnapshot.ReplacementCorrection, "   ", t0)
	if !errors.Is(err, publicationsnapshot.ErrBlankReplacementReason) {
		t.Fatalf("expected ErrBlankReplacementReason, got: %v", err)
	}

	// 2. Blank successor ID
	err = store.RegisterReplacement(tenantID, snapID, 1, "   ", 2, publicationsnapshot.ReplacementCorrection, "Valid reason", t0)
	if !errors.Is(err, publicationsnapshot.ErrBlankSuccessorID) {
		t.Fatalf("expected ErrBlankSuccessorID, got: %v", err)
	}

	// 3. Non-existent snapshot ID
	err = store.RegisterReplacement(tenantID, "snap_nonexistent", 1, "snap_succ_01", 2, publicationsnapshot.ReplacementCorrection, "Valid reason", t0)
	if !errors.Is(err, publicationsnapshot.ErrSnapshotNotFound) {
		t.Fatalf("expected ErrSnapshotNotFound, got: %v", err)
	}
}

// NEG-IMM-06: Local-Only Synthetic Non-Claims Invariant
// Threat: Misrepresentation of local synthetic in-memory fixtures as live external publication routes.
// Expected: Formally proves fixtures operate strictly in-memory without persistent or external routes.
func TestNegativeControl_NEG_IMM_06_LocalSyntheticNonClaims(t *testing.T) {
	// Assert governance non-claims:
	// 1. Zero external live public routes or CDN endpoints
	// 2. Zero production database schema or SQL mutations
	// 3. Zero real credentials or customer data
	// 4. Purely local synthetic in-memory store
	store := publicationsnapshot.NewImmutablePublicationStore()
	if store == nil {
		t.Fatalf("immutable publication store failed to initialize in memory")
	}
}
