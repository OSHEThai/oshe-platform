package publicationsnapshot_test

import (
	"testing"
	"time"

	publicationsnapshot "oshe/publication-snapshot"
)

func sampleSourceRef(tenantID string) publicationsnapshot.SourceEntityRef {
	return publicationsnapshot.SourceEntityRef{
		SourceType:          "INSPECTION_RECORD",
		SourceID:            "insp_synth_001",
		SourceTenantID:      tenantID,
		SourceVersion:       "1.0",
		SourceContentDigest: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
	}
}

func sampleAllowlist() publicationsnapshot.PublicationFieldAllowlist {
	al, _ := publicationsnapshot.NewPublicationFieldAllowlist([]string{
		"inspection_id",
		"project_code",
		"site_name",
		"inspection_date",
		"overall_status",
		"findings_count",
		"public_summary",
	}, false)
	return al
}

func TestSnapshot_DraftCreationAndAccessors(t *testing.T) {
	tenantID := "ten_synth_alpha"
	snapID := "pub_snap_001"
	source := sampleSourceRef(tenantID)
	allowlist := sampleAllowlist()

	rawPayload := map[string]any{
		"inspection_id":   "insp_synth_001",
		"project_code":    "prj_expansion",
		"site_name":       "Rayong Facility",
		"inspection_date": "2026-09-05",
		"overall_status":  "COMPLIANT",
		"findings_count":  3,
		"public_summary":  "Semi-annual scheduled safety inspection completed with minor findings.",
		"internal_notes":  "CONFIDENTIAL: Internal inspector notes should be redacted by default.",
		"db_row_id":       994821,
	}

	snap, err := publicationsnapshot.NewDraftSnapshot(snapID, tenantID, source, rawPayload, allowlist)
	if err != nil {
		t.Fatalf("unexpected NewDraftSnapshot error: %v", err)
	}

	// Verify accessors
	if snap.SnapshotID() != snapID {
		t.Errorf("snapshotID mismatch: %s", snap.SnapshotID())
	}
	if snap.TenantID() != tenantID {
		t.Errorf("tenantID mismatch: %s", snap.TenantID())
	}
	if snap.Version() != 1 {
		t.Errorf("version mismatch: %d, expected 1", snap.Version())
	}
	if snap.Status() != publicationsnapshot.StatusDraft {
		t.Errorf("expected StatusDraft, got %s", snap.Status())
	}
	if snap.IsPublished() {
		t.Errorf("expected IsPublished == false for draft")
	}
	if snap.IsImmutable() {
		t.Errorf("expected IsImmutable == false for draft")
	}

	// Verify allowlist stripping: unapproved fields removed
	payload := snap.Payload()
	if _, exists := payload["internal_notes"]; exists {
		t.Errorf("unapproved field 'internal_notes' was not stripped from payload")
	}
	if _, exists := payload["db_row_id"]; exists {
		t.Errorf("unapproved field 'db_row_id' was not stripped from payload")
	}
	if payload["overall_status"] != "COMPLIANT" {
		t.Errorf("approved field 'overall_status' mismatch: %v", payload["overall_status"])
	}

	// Verify integrity digests present
	if snap.Integrity().PayloadDigest == "" || snap.Integrity().SignatureDigest == "" {
		t.Errorf("missing integrity digests in snapshot")
	}
}

func TestSnapshot_CanonicalPayloadDigestDeterminism(t *testing.T) {
	payload1 := map[string]any{
		"alpha": "value_a",
		"beta":  "value_b",
		"gamma": "value_c",
	}
	payload2 := map[string]any{
		"gamma": "value_c",
		"alpha": "value_a",
		"beta":  "value_b",
	}

	d1, err1 := publicationsnapshot.ComputeCanonicalPayloadDigest(payload1)
	if err1 != nil {
		t.Fatalf("digest1 error: %v", err1)
	}
	d2, err2 := publicationsnapshot.ComputeCanonicalPayloadDigest(payload2)
	if err2 != nil {
		t.Fatalf("digest2 error: %v", err2)
	}

	if d1 != d2 {
		t.Fatalf("canonical digests differ for identical payloads: %s != %s", d1, d2)
	}
}

func TestSnapshot_ReviewerApprovalAndPublish(t *testing.T) {
	registry := publicationsnapshot.NewSnapshotRegistry()
	tenantID := "ten_synth_alpha"
	snapID := "pub_snap_pub_01"
	source := sampleSourceRef(tenantID)
	allowlist := sampleAllowlist()

	raw := map[string]any{
		"inspection_id":   "insp_synth_001",
		"project_code":    "prj_expansion",
		"overall_status":  "COMPLIANT",
		"inspection_date": "2026-09-05",
	}

	draft, err := publicationsnapshot.NewDraftSnapshot(snapID, tenantID, source, raw, allowlist)
	if err != nil {
		t.Fatalf("NewDraftSnapshot error: %v", err)
	}

	if err := registry.RegisterDraft(draft); err != nil {
		t.Fatalf("RegisterDraft error: %v", err)
	}

	// Create valid reviewer context
	now := time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC)
	reviewer, err := publicationsnapshot.NewReviewerContext("usr_synth_auditor_01", "AUDITOR", publicationsnapshot.ReviewApproved, "Approved for external preview", now)
	if err != nil {
		t.Fatalf("NewReviewerContext error: %v", err)
	}

	pubSnap, err := registry.PublishSnapshot(tenantID, snapID, reviewer)
	if err != nil {
		t.Fatalf("PublishSnapshot error: %v", err)
	}

	if !pubSnap.IsPublished() || pubSnap.Status() != publicationsnapshot.StatusPublished {
		t.Errorf("expected StatusPublished, got %s", pubSnap.Status())
	}
	if !pubSnap.IsImmutable() {
		t.Errorf("expected IsImmutable == true for published snapshot")
	}
	if pubSnap.Reviewer().ApprovalStatus != publicationsnapshot.ReviewApproved {
		t.Errorf("expected ReviewApproved, got %s", pubSnap.Reviewer().ApprovalStatus)
	}

	// Verify integrity
	if err := pubSnap.VerifyIntegrity(); err != nil {
		t.Fatalf("VerifyIntegrity failed on published snapshot: %v", err)
	}
	if err := registry.VerifySnapshotIntegrity(tenantID, snapID); err != nil {
		t.Fatalf("registry VerifySnapshotIntegrity failed: %v", err)
	}
}

func TestSnapshot_CreateNewVersionAndSupercede(t *testing.T) {
	registry := publicationsnapshot.NewSnapshotRegistry()
	tenantID := "ten_synth_alpha"
	snapID := "pub_snap_v_01"
	source := sampleSourceRef(tenantID)
	allowlist := sampleAllowlist()

	draft, _ := publicationsnapshot.NewDraftSnapshot(snapID, tenantID, source, map[string]any{
		"inspection_id":  "insp_synth_001",
		"overall_status": "PENDING_REMEDIATION",
	}, allowlist)
	_ = registry.RegisterDraft(draft)

	t0 := time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC)
	rev1, _ := publicationsnapshot.NewReviewerContext("usr_auditor_1", "AUDITOR", publicationsnapshot.ReviewApproved, "v1 approval", t0)
	v1, err := registry.PublishSnapshot(tenantID, snapID, rev1)
	if err != nil {
		t.Fatalf("PublishSnapshot v1 error: %v", err)
	}
	if v1.Version() != 1 {
		t.Errorf("expected v1 version 1, got %d", v1.Version())
	}

	// Create new version with updated status
	t1 := t0.Add(24 * time.Hour)
	rev2, _ := publicationsnapshot.NewReviewerContext("usr_auditor_2", "AUDITOR", publicationsnapshot.ReviewApproved, "v2 remediated approval", t1)
	v2, err := registry.CreateNewVersion(tenantID, snapID, map[string]any{
		"inspection_id":  "insp_synth_001",
		"overall_status": "COMPLIANT_REMEDIATED",
	}, allowlist, rev2)
	if err != nil {
		t.Fatalf("CreateNewVersion error: %v", err)
	}

	if v2.Version() != 2 {
		t.Errorf("expected version 2, got %d", v2.Version())
	}
	if v2.Status() != publicationsnapshot.StatusPublished {
		t.Errorf("expected v2 to be published, got %s", v2.Status())
	}

	// Verify v1 is now SUPERSEDED
	historicV1, err := registry.GetSnapshotVersion(tenantID, snapID, 1)
	if err != nil {
		t.Fatalf("failed to retrieve historic v1: %v", err)
	}
	if historicV1.Status() != publicationsnapshot.StatusSuperseded {
		t.Errorf("expected historic v1 to be SUPERSEDED, got %s", historicV1.Status())
	}

	// Verify GetSnapshot returns v2
	current, err := registry.GetSnapshot(tenantID, snapID)
	if err != nil {
		t.Fatalf("GetSnapshot error: %v", err)
	}
	if current.Version() != 2 {
		t.Errorf("expected GetSnapshot to return current v2, got v%d", current.Version())
	}
}

func TestSnapshot_Withdrawal(t *testing.T) {
	registry := publicationsnapshot.NewSnapshotRegistry()
	tenantID := "ten_synth_alpha"
	snapID := "pub_snap_withdraw_01"
	source := sampleSourceRef(tenantID)
	allowlist := sampleAllowlist()

	draft, _ := publicationsnapshot.NewDraftSnapshot(snapID, tenantID, source, map[string]any{
		"inspection_id": "insp_01",
	}, allowlist)
	_ = registry.RegisterDraft(draft)

	t0 := time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC)
	rev, _ := publicationsnapshot.NewReviewerContext("usr_auditor_1", "AUDITOR", publicationsnapshot.ReviewApproved, "initial approval", t0)
	_, _ = registry.PublishSnapshot(tenantID, snapID, rev)

	// Withdraw snapshot
	withdrawnRev, _ := publicationsnapshot.NewReviewerContext("usr_admin_1", "TENANT_ADMIN", publicationsnapshot.ReviewWithdrawn, "Administrative retraction due to source correction", t0.Add(2*time.Hour))
	withdrawnSnap, err := registry.WithdrawSnapshot(tenantID, snapID, "Source data recall", withdrawnRev)
	if err != nil {
		t.Fatalf("WithdrawSnapshot error: %v", err)
	}

	if withdrawnSnap.Status() != publicationsnapshot.StatusWithdrawn {
		t.Errorf("expected StatusWithdrawn, got %s", withdrawnSnap.Status())
	}
	if !withdrawnSnap.IsImmutable() {
		t.Errorf("expected withdrawn snapshot to be immutable")
	}
}

func TestSnapshot_ExportPackageGeneration(t *testing.T) {
	tenantID := "ten_synth_alpha"
	source := sampleSourceRef(tenantID)
	allowlist := sampleAllowlist()

	draft1, _ := publicationsnapshot.NewDraftSnapshot("snap_exp_1", tenantID, source, map[string]any{
		"inspection_id":  "insp_01",
		"overall_status": "COMPLIANT",
	}, allowlist)
	draft2, _ := publicationsnapshot.NewDraftSnapshot("snap_exp_2", tenantID, source, map[string]any{
		"inspection_id":  "insp_02",
		"overall_status": "COMPLIANT",
	}, allowlist)

	reg := publicationsnapshot.NewSnapshotRegistry()
	_ = reg.RegisterDraft(draft1)
	_ = reg.RegisterDraft(draft2)

	rev, _ := publicationsnapshot.NewReviewerContext("usr_auditor_01", "AUDITOR", publicationsnapshot.ReviewApproved, "approved", time.Now().UTC())
	pub1, _ := reg.PublishSnapshot(tenantID, "snap_exp_1", rev)
	pub2, _ := reg.PublishSnapshot(tenantID, "snap_exp_2", rev)

	// Build export package
	pkg, err := publicationsnapshot.NewExportPackage(
		"exp_pkg_20260905_01",
		tenantID,
		"JSON",
		"PUBLIC_SANITIZED",
		"PUBLIC_PORTAL_PREVIEW",
		"usr_synth_exporter_01",
		[]publicationsnapshot.PublicationSnapshot{pub1, pub2},
	)
	if err != nil {
		t.Fatalf("NewExportPackage error: %v", err)
	}

	if pkg.RecordCount != 2 {
		t.Errorf("expected RecordCount 2, got %d", pkg.RecordCount)
	}
	if pkg.Classification != "PUBLIC_SANITIZED" {
		t.Errorf("expected classification PUBLIC_SANITIZED, got %s", pkg.Classification)
	}
	if pkg.DestinationScope != "PUBLIC_PORTAL_PREVIEW" {
		t.Errorf("expected destination PUBLIC_PORTAL_PREVIEW, got %s", pkg.DestinationScope)
	}
	if pkg.IntegrityChecksum == "" {
		t.Errorf("missing package integrity checksum")
	}
}
