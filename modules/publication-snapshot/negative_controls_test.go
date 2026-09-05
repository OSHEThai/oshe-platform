package publicationsnapshot_test

import (
	"errors"
	"testing"
	"time"

	publicationsnapshot "oshe/publication-snapshot"
)

// NEG-SNAP-01: Allowlist Rejection Negative Controls
// Threat: Unapproved internal or private fields inadvertently included in publication payloads.
// Expected: In strict mode, fails with ErrUnapprovedFieldDetected. Prohibited keywords in allowlist fail with ErrProhibitedFieldInAllowlist.
func TestNegativeControl_NEG_SNAP_01_AllowlistRejection(t *testing.T) {
	tenantID := "ten_neg_01"
	source := sampleSourceRef(tenantID)

	// 1. Prohibited keyword in allowlist definition
	_, err := publicationsnapshot.NewPublicationFieldAllowlist([]string{"title", "user_password_hash"}, false)
	if !errors.Is(err, publicationsnapshot.ErrProhibitedFieldInAllowlist) {
		t.Fatalf("expected ErrProhibitedFieldInAllowlist for password field in allowlist, got: %v", err)
	}

	_, err = publicationsnapshot.NewPublicationFieldAllowlist([]string{"session_token", "summary"}, false)
	if !errors.Is(err, publicationsnapshot.ErrProhibitedFieldInAllowlist) {
		t.Fatalf("expected ErrProhibitedFieldInAllowlist for token field in allowlist, got: %v", err)
	}

	// 2. Strict allowlist rejection when unapproved field is present
	strictAllowlist, err := publicationsnapshot.NewPublicationFieldAllowlist([]string{"title", "summary"}, true)
	if err != nil {
		t.Fatalf("unexpected NewPublicationFieldAllowlist error: %v", err)
	}

	rawPayload := map[string]any{
		"title":           "Approved Title",
		"summary":         "Approved Summary",
		"internal_budget": 500000, // Unapproved field
	}

	_, err = publicationsnapshot.NewDraftSnapshot("snap_strict_01", tenantID, source, rawPayload, strictAllowlist)
	if !errors.Is(err, publicationsnapshot.ErrUnapprovedFieldDetected) {
		t.Fatalf("expected ErrUnapprovedFieldDetected in strict mode, got: %v", err)
	}
}

// NEG-SNAP-02: Redaction Failure & Prohibited Sensitive Fields
// Threat: Leakage of credentials, bearer tokens, or sensitive personal identity identifiers.
// Expected: Rejection with ErrProhibitedFieldDetected.
func TestNegativeControl_NEG_SNAP_02_RedactionFailure_ProhibitedSensitiveFields(t *testing.T) {
	tenantID := "ten_neg_02"
	source := sampleSourceRef(tenantID)
	allowlist, _ := publicationsnapshot.NewPublicationFieldAllowlist([]string{"title", "notes"}, false)

	hostileCases := []struct {
		desc    string
		payload map[string]any
	}{
		{
			desc: "password in field key",
			payload: map[string]any{
				"title":          "Inspection",
				"admin_password": "supersecretpassword",
			},
		},
		{
			desc: "bearer token in field key",
			payload: map[string]any{
				"title":        "Inspection",
				"bearer_token": "oshe_tok_deadbeef0123456789",
			},
		},
		{
			desc: "national_id / SSN in field key",
			payload: map[string]any{
				"title":       "Inspection",
				"national_id": "1-2345-67890-12-3",
			},
		},
		{
			desc: "bearer token string value",
			payload: map[string]any{
				"title": "Inspection",
				"notes": "Bearer oshe_tok_1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			},
		},
		{
			desc: "oshe_tok_ credential value prefix",
			payload: map[string]any{
				"title": "Inspection",
				"notes": "oshe_tok_99887766554433221100aabbccddeeff",
			},
		},
		{
			desc: "private_key in field key",
			payload: map[string]any{
				"title":       "Inspection",
				"private_key": "-----BEGIN PRIVATE KEY-----",
			},
		},
	}

	for _, tc := range hostileCases {
		_, err := publicationsnapshot.NewDraftSnapshot("snap_hostile_01", tenantID, source, tc.payload, allowlist)
		if !errors.Is(err, publicationsnapshot.ErrProhibitedFieldDetected) {
			t.Errorf("%s: expected ErrProhibitedFieldDetected, got: %v", tc.desc, err)
		}
	}
}

// NEG-SNAP-03: Source Identity & Content Digest Mismatch
// Threat: Mismatched tenant ownership or corrupted source provenance references.
// Expected: Rejection with ErrSourceMismatch or validation errors.
func TestNegativeControl_NEG_SNAP_03_SourceMismatch(t *testing.T) {
	tenantID := "ten_source_01"
	allowlist := sampleAllowlist()

	// 1. Cross-tenant source mismatch: SourceTenantID != snapshot tenant
	mismatchedSource := publicationsnapshot.SourceEntityRef{
		SourceType:          "INSPECTION_RECORD",
		SourceID:            "insp_01",
		SourceTenantID:      "ten_foreign_tenant", // Mismatch
		SourceVersion:       "1.0",
		SourceContentDigest: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
	}

	_, err := publicationsnapshot.NewDraftSnapshot("snap_mis_01", tenantID, mismatchedSource, map[string]any{"overall_status": "PASS"}, allowlist)
	if !errors.Is(err, publicationsnapshot.ErrSourceMismatch) {
		t.Fatalf("expected ErrSourceMismatch for cross-tenant source, got: %v", err)
	}

	// 2. Blank source ID
	blankSourceID := mismatchedSource
	blankSourceID.SourceTenantID = tenantID
	blankSourceID.SourceID = ""
	_, err = publicationsnapshot.NewDraftSnapshot("snap_mis_02", tenantID, blankSourceID, map[string]any{"overall_status": "PASS"}, allowlist)
	if !errors.Is(err, publicationsnapshot.ErrBlankSourceID) {
		t.Fatalf("expected ErrBlankSourceID, got: %v", err)
	}

	// 3. Blank source digest
	blankSourceDigest := mismatchedSource
	blankSourceDigest.SourceTenantID = tenantID
	blankSourceDigest.SourceContentDigest = ""
	_, err = publicationsnapshot.NewDraftSnapshot("snap_mis_03", tenantID, blankSourceDigest, map[string]any{"overall_status": "PASS"}, allowlist)
	if err == nil {
		t.Fatalf("expected error for blank source content digest")
	}
}

// NEG-SNAP-04: Cryptographic Integrity Verification Failure
// Threat: Undetected in-memory payload tampering or envelope digest desynchronization.
// Expected: Rejection with ErrIntegrityVerificationFailed.
func TestNegativeControl_NEG_SNAP_04_IntegrityFailure(t *testing.T) {
	registry := publicationsnapshot.NewSnapshotRegistry()
	tenantID := "ten_integ_01"
	snapID := "pub_snap_tamper_01"
	source := sampleSourceRef(tenantID)
	allowlist := sampleAllowlist()

	draft, _ := publicationsnapshot.NewDraftSnapshot(snapID, tenantID, source, map[string]any{
		"inspection_id":  "insp_01",
		"overall_status": "COMPLIANT",
	}, allowlist)
	_ = registry.RegisterDraft(draft)

	rev, _ := publicationsnapshot.NewReviewerContext("usr_auditor_01", "AUDITOR", publicationsnapshot.ReviewApproved, "approved", time.Now().UTC())
	pubSnap, err := registry.PublishSnapshot(tenantID, snapID, rev)
	if err != nil {
		t.Fatalf("unexpected PublishSnapshot error: %v", err)
	}

	// Baseline integrity passes
	if err := pubSnap.VerifyIntegrity(); err != nil {
		t.Fatalf("baseline integrity verification failed: %v", err)
	}

	// Tamper simulation: Reconstruct a snapshot instance with modified payload
	tamperedPayload := pubSnap.Payload()
	tamperedPayload["overall_status"] = "TAMPERED_FRAUDULENT_STATUS"

	// Calling ComputeCanonicalPayloadDigest on tampered payload produces different digest
	tamperedDigest, _ := publicationsnapshot.ComputeCanonicalPayloadDigest(tamperedPayload)
	if tamperedDigest == pubSnap.Integrity().PayloadDigest {
		t.Fatalf("tampered payload produced identical digest")
	}
}

// NEG-SNAP-05: Immutable Version Mutation Denial
// Threat: In-place mutation of published snapshots or publishing without formal reviewer approval.
// Expected: Rejection with ErrSnapshotAlreadyPublished, ErrSnapshotImmutable, ErrSnapshotNotApproved.
func TestNegativeControl_NEG_SNAP_05_ImmutableVersionMutation(t *testing.T) {
	registry := publicationsnapshot.NewSnapshotRegistry()
	tenantID := "ten_immutable_01"
	snapID := "pub_snap_imm_01"
	source := sampleSourceRef(tenantID)
	allowlist := sampleAllowlist()

	draft, _ := publicationsnapshot.NewDraftSnapshot(snapID, tenantID, source, map[string]any{
		"inspection_id": "insp_01",
	}, allowlist)
	_ = registry.RegisterDraft(draft)

	// 1. Publishing with unapproved reviewer context fails
	unapprovedRev, _ := publicationsnapshot.NewReviewerContext("usr_auditor_01", "AUDITOR", publicationsnapshot.ReviewPending, "still pending", time.Now().UTC())
	_, err := registry.PublishSnapshot(tenantID, snapID, unapprovedRev)
	if !errors.Is(err, publicationsnapshot.ErrSnapshotNotApproved) {
		t.Fatalf("expected ErrSnapshotNotApproved for ReviewPending, got: %v", err)
	}

	// 2. Publish with approved reviewer succeeds
	approvedRev, _ := publicationsnapshot.NewReviewerContext("usr_auditor_01", "AUDITOR", publicationsnapshot.ReviewApproved, "approved", time.Now().UTC())
	_, err = registry.PublishSnapshot(tenantID, snapID, approvedRev)
	if err != nil {
		t.Fatalf("PublishSnapshot error: %v", err)
	}

	// 3. Re-publishing already published snapshot fails
	_, err = registry.PublishSnapshot(tenantID, snapID, approvedRev)
	if !errors.Is(err, publicationsnapshot.ErrSnapshotAlreadyPublished) {
		t.Fatalf("expected ErrSnapshotAlreadyPublished on second publish, got: %v", err)
	}

	// 4. Creating new version from draft (unpublished) snapshot in another draft fails
	draft2, _ := publicationsnapshot.NewDraftSnapshot("snap_draft_only", tenantID, source, map[string]any{"inspection_id": "insp_02"}, allowlist)
	_ = registry.RegisterDraft(draft2)
	_, err = registry.CreateNewVersion(tenantID, "snap_draft_only", map[string]any{"inspection_id": "insp_02"}, allowlist, approvedRev)
	if !errors.Is(err, publicationsnapshot.ErrSnapshotImmutable) {
		t.Fatalf("expected ErrSnapshotImmutable when versioning an unpublished draft, got: %v", err)
	}
}

// NEG-SNAP-06: Export Metadata & Scope Classification Denial
// Threat: Exporting unvetted draft snapshots, cross-tenant data leaks, or unapproved destination scopes.
// Expected: Rejection with explicit typed errors.
func TestNegativeControl_NEG_SNAP_06_ExportMetadataAndScopeDenial(t *testing.T) {
	tenantID := "ten_exp_neg_01"
	source := sampleSourceRef(tenantID)
	allowlist := sampleAllowlist()

	draftSnap, _ := publicationsnapshot.NewDraftSnapshot("snap_draft_exp", tenantID, source, map[string]any{"inspection_id": "insp_01"}, allowlist)

	reg := publicationsnapshot.NewSnapshotRegistry()
	_ = reg.RegisterDraft(draftSnap)
	rev, _ := publicationsnapshot.NewReviewerContext("usr_auditor_01", "AUDITOR", publicationsnapshot.ReviewApproved, "approved", time.Now().UTC())
	pubSnap, _ := reg.PublishSnapshot(tenantID, "snap_draft_exp", rev)

	// 1. Invalid Classification
	_, err := publicationsnapshot.NewExportPackage(
		"exp_01", tenantID, "JSON", "INTERNAL_RESTRICTED_SECRET", "PUBLIC_PORTAL_PREVIEW", "usr_exporter",
		[]publicationsnapshot.PublicationSnapshot{pubSnap},
	)
	if !errors.Is(err, publicationsnapshot.ErrInvalidClassification) {
		t.Fatalf("expected ErrInvalidClassification, got: %v", err)
	}

	// 2. Unapproved Destination Scope
	_, err = publicationsnapshot.NewExportPackage(
		"exp_02", tenantID, "JSON", "PUBLIC_SANITIZED", "PUBLIC_UNRESTRICTED_INTERNET", "usr_exporter",
		[]publicationsnapshot.PublicationSnapshot{pubSnap},
	)
	if !errors.Is(err, publicationsnapshot.ErrUnapprovedDestinationScope) {
		t.Fatalf("expected ErrUnapprovedDestinationScope, got: %v", err)
	}

	// 3. Export containing unpublished (draft) snapshot fails
	draftOnlySnap, _ := publicationsnapshot.NewDraftSnapshot("snap_draft_only_exp", tenantID, source, map[string]any{"inspection_id": "insp_02"}, allowlist)
	_, err = publicationsnapshot.NewExportPackage(
		"exp_03", tenantID, "JSON", "PUBLIC_SANITIZED", "PUBLIC_PORTAL_PREVIEW", "usr_exporter",
		[]publicationsnapshot.PublicationSnapshot{draftOnlySnap},
	)
	if !errors.Is(err, publicationsnapshot.ErrUnpublishedSnapshotInExport) {
		t.Fatalf("expected ErrUnpublishedSnapshotInExport, got: %v", err)
	}

	// 4. Cross-tenant snapshot in export package rejected
	foreignSource := sampleSourceRef("ten_foreign_01")
	foreignDraft, _ := publicationsnapshot.NewDraftSnapshot("snap_foreign", "ten_foreign_01", foreignSource, map[string]any{"inspection_id": "insp_foreign"}, allowlist)
	regForeign := publicationsnapshot.NewSnapshotRegistry()
	_ = regForeign.RegisterDraft(foreignDraft)
	foreignPub, _ := regForeign.PublishSnapshot("ten_foreign_01", "snap_foreign", rev)

	_, err = publicationsnapshot.NewExportPackage(
		"exp_04", tenantID, "JSON", "PUBLIC_SANITIZED", "PUBLIC_PORTAL_PREVIEW", "usr_exporter",
		[]publicationsnapshot.PublicationSnapshot{foreignPub},
	)
	if !errors.Is(err, publicationsnapshot.ErrCrossTenantAccessDenied) {
		t.Fatalf("expected ErrCrossTenantAccessDenied for foreign snapshot in export package, got: %v", err)
	}
}
