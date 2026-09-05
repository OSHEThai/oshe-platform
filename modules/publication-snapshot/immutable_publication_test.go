package publicationsnapshot_test

import (
	"errors"
	"testing"
	"time"

	publicationsnapshot "oshe/publication-snapshot"
)

func TestImmutableStore_StoreAndRetrievePublishedVersion(t *testing.T) {
	store := publicationsnapshot.NewImmutablePublicationStore()
	tenantID := "ten_imm_01"
	snapID := "snap_imm_001"
	t0 := time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC)

	source := publicationsnapshot.SourceEntityRef{
		SourceType:          "SAFETY_INSPECTION",
		SourceID:            "insp_001",
		SourceTenantID:      tenantID,
		SourceVersion:       "1.0",
		SourceContentDigest: "digest_source_001",
	}

	evidence, _ := publicationsnapshot.NewApprovalEvidence("app_01", "usr_auditor_01", "AUDITOR", t0, "digest_source_001", "Approved", 7*24*time.Hour)
	window := publicationsnapshot.EffectiveWindow{
		EffectiveFrom: t0,
		ExpiresAt:     t0.Add(30 * 24 * time.Hour),
	}

	payload := map[string]any{
		"inspection_id":  "insp_001",
		"overall_status": "COMPLIANT",
		"project_code":   "PRJ-ALPHA",
	}

	rec, err := store.StorePublishedVersion(tenantID, snapID, 1, payload, source, evidence, window, 0, "", t0)
	if err != nil {
		t.Fatalf("StorePublishedVersion failed: %v", err)
	}

	if rec.Version != 1 || rec.SnapshotID != snapID || rec.TenantID != tenantID {
		t.Errorf("record metadata mismatch: %+v", rec)
	}

	// Retrieve by exact version
	retrieved, err := store.GetPublishedVersion(tenantID, snapID, 1)
	if err != nil {
		t.Fatalf("GetPublishedVersion failed: %v", err)
	}
	if retrieved.Payload["overall_status"] != "COMPLIANT" {
		t.Errorf("payload content mismatch: %v", retrieved.Payload)
	}

	// Retrieve by latest version
	latest, err := store.GetLatestPublishedVersion(tenantID, snapID)
	if err != nil {
		t.Fatalf("GetLatestPublishedVersion failed: %v", err)
	}
	if latest.Version != 1 {
		t.Errorf("latest version mismatch: %d", latest.Version)
	}

	// Defensive deep-copy verification: mutating retrieved payload does not affect store
	retrieved.Payload["overall_status"] = "MUTATED_BY_CALLER"
	checkAgain, _ := store.GetPublishedVersion(tenantID, snapID, 1)
	if checkAgain.Payload["overall_status"] == "MUTATED_BY_CALLER" {
		t.Fatalf("defensive deep-copy failed: caller mutated stored payload!")
	}
}

func TestImmutableStore_SourceUpdateIsolation(t *testing.T) {
	store := publicationsnapshot.NewImmutablePublicationStore()
	tenantID := "ten_imm_02"
	snapID := "snap_imm_002"
	t0 := time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC)

	originalSourceDigest := "initial_source_digest_111"
	source := publicationsnapshot.SourceEntityRef{
		SourceType:          "AUDIT_RECORD",
		SourceID:            "audit_002",
		SourceTenantID:      tenantID,
		SourceVersion:       "1.0",
		SourceContentDigest: originalSourceDigest,
	}

	evidence, _ := publicationsnapshot.NewApprovalEvidence("app_02", "usr_auditor_01", "AUDITOR", t0, originalSourceDigest, "Approved", 7*24*time.Hour)
	window := publicationsnapshot.EffectiveWindow{EffectiveFrom: t0, ExpiresAt: t0.Add(30 * 24 * time.Hour)}
	payload := map[string]any{"audit_id": "audit_002", "score": 95}

	rec, err := store.StorePublishedVersion(tenantID, snapID, 1, payload, source, evidence, window, 0, "", t0)
	if err != nil {
		t.Fatalf("StorePublishedVersion failed: %v", err)
	}

	// 1. Operational source evolves: new findings added in operational DB
	evolvedSourceDigest := "evolved_source_digest_999"

	// 2. Check source drift
	drift, err := store.CheckSourceDrift(tenantID, snapID, 1, evolvedSourceDigest)
	if err != nil {
		t.Fatalf("CheckSourceDrift failed: %v", err)
	}
	if !drift.HasDrifted {
		t.Errorf("expected drift to be true when operational digest changes")
	}
	if drift.PublishedDigest != originalSourceDigest {
		t.Errorf("expected published digest to match original: %s != %s", drift.PublishedDigest, originalSourceDigest)
	}

	// 3. Confirm published snapshot record in store remains 100% untouched
	storedRec, err := store.GetPublishedVersion(tenantID, snapID, 1)
	if err != nil {
		t.Fatalf("GetPublishedVersion failed: %v", err)
	}
	if storedRec.Source.SourceContentDigest != originalSourceDigest {
		t.Fatalf("published snapshot source content digest was corrupted by source update!")
	}
	if storedRec.SignatureDigest != rec.SignatureDigest {
		t.Fatalf("published snapshot signature digest was corrupted by source update!")
	}

	// 4. Attempting direct mutation from source is rejected
	err = store.AttemptDirectSourceMutation(tenantID, snapID, 1, map[string]any{"audit_id": "audit_002", "score": 80})
	if !errors.Is(err, publicationsnapshot.ErrDirectSourceMutationForbidden) {
		t.Fatalf("expected ErrDirectSourceMutationForbidden, got: %v", err)
	}
}

func TestImmutableStore_ReplacementAndLineageChain(t *testing.T) {
	store := publicationsnapshot.NewImmutablePublicationStore()
	tenantID := "ten_imm_03"
	snapID := "snap_imm_003"
	t0 := time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC)

	source1 := publicationsnapshot.SourceEntityRef{
		SourceType:          "CHECKLIST_RECORD",
		SourceID:            "chk_001",
		SourceTenantID:      tenantID,
		SourceVersion:       "1.0",
		SourceContentDigest: "digest_v1",
	}
	ev1, _ := publicationsnapshot.NewApprovalEvidence("app_03_1", "usr_auditor", "AUDITOR", t0, "digest_v1", "v1 ok", 7*24*time.Hour)
	win1 := publicationsnapshot.EffectiveWindow{EffectiveFrom: t0, ExpiresAt: t0.Add(30 * 24 * time.Hour)}

	// Store v1
	v1, err := store.StorePublishedVersion(tenantID, snapID, 1, map[string]any{"status": "INITIAL"}, source1, ev1, win1, 0, "", t0)
	if err != nil {
		t.Fatalf("Store v1 failed: %v", err)
	}

	// Store v2 (superseding v1)
	t1 := t0.Add(24 * time.Hour)
	source2 := source1
	source2.SourceVersion = "2.0"
	source2.SourceContentDigest = "digest_v2"
	ev2, _ := publicationsnapshot.NewApprovalEvidence("app_03_2", "usr_auditor", "AUDITOR", t1, "digest_v2", "v2 ok", 7*24*time.Hour)
	win2 := publicationsnapshot.EffectiveWindow{EffectiveFrom: t1, ExpiresAt: t1.Add(30 * 24 * time.Hour)}

	v2, err := store.StorePublishedVersion(tenantID, snapID, 2, map[string]any{"status": "UPDATED"}, source2, ev2, win2, 1, v1.SignatureDigest, t1)
	if err != nil {
		t.Fatalf("Store v2 failed: %v", err)
	}

	// Store v3 (superseding v2)
	t2 := t1.Add(24 * time.Hour)
	source3 := source1
	source3.SourceVersion = "3.0"
	source3.SourceContentDigest = "digest_v3"
	ev3, _ := publicationsnapshot.NewApprovalEvidence("app_03_3", "usr_auditor", "AUDITOR", t2, "digest_v3", "v3 ok", 7*24*time.Hour)
	win3 := publicationsnapshot.EffectiveWindow{EffectiveFrom: t2, ExpiresAt: t2.Add(30 * 24 * time.Hour)}

	v3, err := store.StorePublishedVersion(tenantID, snapID, 3, map[string]any{"status": "FINAL"}, source3, ev3, win3, 2, v2.SignatureDigest, t2)
	if err != nil {
		t.Fatalf("Store v3 failed: %v", err)
	}

	// Verify ListPublishedVersions
	versions, err := store.ListPublishedVersions(tenantID, snapID)
	if err != nil {
		t.Fatalf("ListPublishedVersions failed: %v", err)
	}
	if len(versions) != 3 {
		t.Fatalf("expected 3 versions, got %d", len(versions))
	}
	if versions[0].Version != 1 || versions[1].Version != 2 || versions[2].Version != 3 {
		t.Errorf("version sequence mismatch: %+v", versions)
	}

	// Verify v1 successor link updated
	updatedV1, _ := store.GetPublishedVersion(tenantID, snapID, 1)
	if updatedV1.SuccessorVersion != 2 || updatedV1.SuccessorDigest != v2.SignatureDigest {
		t.Errorf("v1 successor link mismatch: %+v", updatedV1)
	}

	// Verify v2 links
	if v2.PredecessorVersion != 1 || v2.PredecessorDigest != v1.SignatureDigest {
		t.Errorf("v2 predecessor link mismatch: %+v", v2)
	}

	// Verify v3 is latest
	latest, _ := store.GetLatestPublishedVersion(tenantID, snapID)
	if latest.Version != 3 || latest.Payload["status"] != "FINAL" {
		t.Errorf("latest version mismatch: %+v", latest)
	}
	_ = v3
}

func TestImmutableStore_AuditReconstruction_Intact(t *testing.T) {
	store := publicationsnapshot.NewImmutablePublicationStore()
	tenantID := "ten_imm_04"
	snapID := "snap_imm_004"
	t0 := time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC)

	source1 := publicationsnapshot.SourceEntityRef{
		SourceType:          "INCIDENT_RECORD",
		SourceID:            "inc_001",
		SourceTenantID:      tenantID,
		SourceVersion:       "1.0",
		SourceContentDigest: "digest_inc_1",
	}
	ev1, _ := publicationsnapshot.NewApprovalEvidence("app_04_1", "usr_auditor", "AUDITOR", t0, "digest_inc_1", "v1 approved", 7*24*time.Hour)
	win1 := publicationsnapshot.EffectiveWindow{EffectiveFrom: t0, ExpiresAt: t0.Add(30 * 24 * time.Hour)}
	v1, _ := store.StorePublishedVersion(tenantID, snapID, 1, map[string]any{"incident": "reported"}, source1, ev1, win1, 0, "", t0)

	t1 := t0.Add(12 * time.Hour)
	source2 := source1
	source2.SourceVersion = "2.0"
	source2.SourceContentDigest = "digest_inc_2"
	ev2, _ := publicationsnapshot.NewApprovalEvidence("app_04_2", "usr_auditor", "AUDITOR", t1, "digest_inc_2", "v2 approved", 7*24*time.Hour)
	win2 := publicationsnapshot.EffectiveWindow{EffectiveFrom: t1, ExpiresAt: t1.Add(30 * 24 * time.Hour)}
	_, _ = store.StorePublishedVersion(tenantID, snapID, 2, map[string]any{"incident": "resolved"}, source2, ev2, win2, 1, v1.SignatureDigest, t1)

	// Reconstruct audit trail
	report, err := store.ReconstructPublicationAuditTrail(tenantID, snapID)
	if err != nil {
		t.Fatalf("ReconstructPublicationAuditTrail failed: %v", err)
	}

	if report.Status != publicationsnapshot.StatusVerifiedIntact {
		t.Fatalf("expected VERIFIED_INTACT, got: %s (findings: %v)", report.Status, report.Findings)
	}
	if report.TotalVersions != 2 {
		t.Errorf("expected 2 versions in audit report, got %d", report.TotalVersions)
	}
	if report.LineageChainDigest == "" {
		t.Errorf("missing lineage chain digest")
	}
	if len(report.Findings) != 0 {
		t.Errorf("expected 0 findings for intact audit, got: %v", report.Findings)
	}
}

func TestImmutableStore_TenantIsolation(t *testing.T) {
	store := publicationsnapshot.NewImmutablePublicationStore()
	tenantA := "ten_iso_a"
	tenantB := "ten_iso_b"
	snapID := "snap_iso_001"
	t0 := time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC)

	sourceA := publicationsnapshot.SourceEntityRef{
		SourceType:          "RECORD_A",
		SourceID:            "rec_a",
		SourceTenantID:      tenantA,
		SourceVersion:       "1.0",
		SourceContentDigest: "digest_a",
	}
	evA, _ := publicationsnapshot.NewApprovalEvidence("app_a", "usr_admin", "TENANT_ADMIN", t0, "digest_a", "ok", 7*24*time.Hour)
	winA := publicationsnapshot.EffectiveWindow{EffectiveFrom: t0, ExpiresAt: t0.Add(30 * 24 * time.Hour)}
	_, _ = store.StorePublishedVersion(tenantA, snapID, 1, map[string]any{"data": "tenant_a_confidential"}, sourceA, evA, winA, 0, "", t0)

	// Tenant B querying for Tenant A snapshot gets ErrSnapshotNotFound (no cross-tenant leakage)
	_, err := store.GetPublishedVersion(tenantB, snapID, 1)
	if !errors.Is(err, publicationsnapshot.ErrSnapshotNotFound) {
		t.Errorf("expected ErrSnapshotNotFound on cross-tenant lookup, got: %v", err)
	}

	// Tenant B audit reconstruction on Tenant A snapshot gets ErrSnapshotNotFound
	_, err = store.ReconstructPublicationAuditTrail(tenantB, snapID)
	if !errors.Is(err, publicationsnapshot.ErrSnapshotNotFound) {
		t.Errorf("expected ErrSnapshotNotFound for cross-tenant audit reconstruction, got: %v", err)
	}
}
