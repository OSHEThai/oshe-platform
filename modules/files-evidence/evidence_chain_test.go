package evidence_test

import (
	"errors"
	"testing"

	evidence "github.com/oshethai/oshe-platform/modules/files-evidence"
)

func TestEvidenceChain_RegisterAndCommitOriginal_Valid(t *testing.T) {
	storage := evidence.NewMemoryStorageAdapter()
	mgr := evidence.NewEvidenceIntegrityManager(storage)

	tenantID := "ten_alpha"
	evidenceID := "evd_syn_original_001"
	payload := []byte("synthetic-original-photo-bytes-fire-extinguisher")
	digest := evidence.ComputeSHA256Digest(payload)

	rec, err := mgr.RegisterOriginal(
		tenantID, evidenceID, "extinguisher_gauge.jpg", "image/jpeg",
		int64(len(payload)), digest, "INSPECTION_RESPONSE", "rsp_syn_01", "inspector_alice",
	)
	if err != nil {
		t.Fatalf("RegisterOriginal failed: %v", err)
	}
	if rec.ObjectType != evidence.ObjectTypeOriginal {
		t.Errorf("expected ObjectTypeOriginal, got %q", rec.ObjectType)
	}
	if rec.Committed {
		t.Errorf("expected uncommitted state initially")
	}

	// Commit original
	err = mgr.CommitOriginal(tenantID, evidenceID, payload, "inspector_alice")
	if err != nil {
		t.Fatalf("CommitOriginal failed: %v", err)
	}

	// Verify committed record
	committedRec, err := mgr.GetRecord(tenantID, evidenceID)
	if err != nil {
		t.Fatalf("GetRecord failed: %v", err)
	}
	if !committedRec.Committed {
		t.Errorf("expected record to be committed")
	}
	if committedRec.State != evidence.StateCompleted {
		t.Errorf("expected StateCompleted, got %q", committedRec.State)
	}

	// Verify custody chain
	chain, err := mgr.GetCustodyChain(tenantID, evidenceID)
	if err != nil {
		t.Fatalf("GetCustodyChain failed: %v", err)
	}
	if len(chain) < 3 {
		t.Fatalf("expected at least 3 custody events, got %d", len(chain))
	}
	if chain[0].EventType != evidence.EventCaptureLocal {
		t.Errorf("expected first event CAPTURE_LOCAL, got %q", chain[0].EventType)
	}
	if chain[len(chain)-1].EventType != evidence.EventOriginalCommitted {
		t.Errorf("expected last event ORIGINAL_COMMITTED, got %q", chain[len(chain)-1].EventType)
	}
}

func TestEvidenceChain_OriginalImmutability(t *testing.T) {
	storage := evidence.NewMemoryStorageAdapter()
	mgr := evidence.NewEvidenceIntegrityManager(storage)

	tenantID := "ten_alpha"
	evidenceID := "evd_syn_original_immut"
	payload := []byte("immutable-original-evidence-payload")
	digest := evidence.ComputeSHA256Digest(payload)

	_, err := mgr.RegisterOriginal(
		tenantID, evidenceID, "original.png", "image/png",
		int64(len(payload)), digest, "SAFETY_FINDING", "fnd_syn_01", "inspector_bob",
	)
	if err != nil {
		t.Fatalf("RegisterOriginal failed: %v", err)
	}

	if err := mgr.CommitOriginal(tenantID, evidenceID, payload, "inspector_bob"); err != nil {
		t.Fatalf("CommitOriginal failed: %v", err)
	}

	// Attempt 1: Re-committing committed original must fail
	err = mgr.CommitOriginal(tenantID, evidenceID, payload, "malicious_actor")
	if !errors.Is(err, evidence.ErrOriginalImmutable) {
		t.Errorf("expected ErrOriginalImmutable on re-commit, got %v", err)
	}

	// Attempt 2: Re-registering existing committed original must fail
	_, err = mgr.RegisterOriginal(
		tenantID, evidenceID, "replaced.png", "image/png",
		int64(len(payload)), digest, "SAFETY_FINDING", "fnd_syn_01", "malicious_actor",
	)
	if !errors.Is(err, evidence.ErrOriginalImmutable) {
		t.Errorf("expected ErrOriginalImmutable on re-register, got %v", err)
	}
}

func TestEvidenceChain_RegisterDerived_ValidAndCommitted(t *testing.T) {
	storage := evidence.NewMemoryStorageAdapter()
	mgr := evidence.NewEvidenceIntegrityManager(storage)

	tenantID := "ten_alpha"
	parentID := "evd_syn_parent_001"
	parentPayload := []byte("original-high-res-photo-payload")
	parentDigest := evidence.ComputeSHA256Digest(parentPayload)

	_, err := mgr.RegisterOriginal(
		tenantID, parentID, "highres.jpg", "image/jpeg",
		int64(len(parentPayload)), parentDigest, "INSPECTION_RESPONSE", "rsp_syn_01", "inspector_alice",
	)
	if err != nil {
		t.Fatalf("RegisterOriginal failed: %v", err)
	}
	if err := mgr.CommitOriginal(tenantID, parentID, parentPayload, "inspector_alice"); err != nil {
		t.Fatalf("CommitOriginal failed: %v", err)
	}

	// Register derived thumbnail
	derivedID := "evd_syn_thumb_001"
	derivedPayload := []byte("derived-thumbnail-64x64-preview-bytes")
	derivedDigest := evidence.ComputeSHA256Digest(derivedPayload)

	derivedRec, err := mgr.RegisterDerived(
		tenantID, derivedID, parentID, evidence.DerivationThumbnailPreview,
		"thumb_highres.jpg", "image/jpeg", int64(len(derivedPayload)), derivedDigest, "image_pipeline",
	)
	if err != nil {
		t.Fatalf("RegisterDerived failed: %v", err)
	}

	if derivedRec.ObjectType != evidence.ObjectTypeDerived {
		t.Errorf("expected ObjectTypeDerived, got %q", derivedRec.ObjectType)
	}
	if derivedRec.ParentEvidenceID != parentID {
		t.Errorf("expected parent %q, got %q", parentID, derivedRec.ParentEvidenceID)
	}
	if derivedRec.DerivationType != evidence.DerivationThumbnailPreview {
		t.Errorf("expected DerivationThumbnailPreview, got %q", derivedRec.DerivationType)
	}

	// Commit derived object
	if err := mgr.CommitDerived(tenantID, derivedID, derivedPayload, "image_pipeline"); err != nil {
		t.Fatalf("CommitDerived failed: %v", err)
	}

	rec, err := mgr.GetRecord(tenantID, derivedID)
	if err != nil || !rec.Committed {
		t.Fatalf("expected committed derived record, got rec=%v, err=%v", rec, err)
	}
}

func TestEvidenceChain_DerivedValidationFailures(t *testing.T) {
	storage := evidence.NewMemoryStorageAdapter()
	mgr := evidence.NewEvidenceIntegrityManager(storage)

	tenantID := "ten_alpha"
	parentID := "evd_syn_parent_002"
	parentPayload := []byte("original-payload-for-failure-tests")
	parentDigest := evidence.ComputeSHA256Digest(parentPayload)

	// Attempt derived before parent is registered
	_, err := mgr.RegisterDerived(
		tenantID, "evd_syn_thumb_fail", parentID, evidence.DerivationThumbnailPreview,
		"thumb.jpg", "image/jpeg", 100, "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", "pipeline",
	)
	if !errors.Is(err, evidence.ErrMissingParentEvidence) {
		t.Errorf("expected ErrMissingParentEvidence, got %v", err)
	}

	// Register parent but do not commit it
	_, err = mgr.RegisterOriginal(
		tenantID, parentID, "parent.jpg", "image/jpeg",
		int64(len(parentPayload)), parentDigest, "CAPA_ACTION", "act_syn_01", "inspector_bob",
	)
	if err != nil {
		t.Fatalf("RegisterOriginal failed: %v", err)
	}

	// Attempt derived before parent is committed
	_, err = mgr.RegisterDerived(
		tenantID, "evd_syn_thumb_uncommitted", parentID, evidence.DerivationThumbnailPreview,
		"thumb.jpg", "image/jpeg", 100, "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", "pipeline",
	)
	if !errors.Is(err, evidence.ErrRecordNotCommitted) {
		t.Errorf("expected ErrRecordNotCommitted, got %v", err)
	}

	// Now commit parent
	if err := mgr.CommitOriginal(tenantID, parentID, parentPayload, "inspector_bob"); err != nil {
		t.Fatalf("CommitOriginal failed: %v", err)
	}

	// Blank parent ID must fail
	_, err = mgr.RegisterDerived(
		tenantID, "evd_syn_thumb_blank", "", evidence.DerivationThumbnailPreview,
		"thumb.jpg", "image/jpeg", 100, "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", "pipeline",
	)
	if !errors.Is(err, evidence.ErrEmptyParentID) {
		t.Errorf("expected ErrEmptyParentID, got %v", err)
	}

	// Invalid derivation type must fail
	_, err = mgr.RegisterDerived(
		tenantID, "evd_syn_thumb_invalid_type", parentID, evidence.DerivationNone,
		"thumb.jpg", "image/jpeg", 100, "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", "pipeline",
	)
	if !errors.Is(err, evidence.ErrInvalidDerivationType) {
		t.Errorf("expected ErrInvalidDerivationType, got %v", err)
	}

	// Valid derived object
	thumbID := "evd_syn_thumb_valid"
	thumbPayload := []byte("valid-thumbnail-bytes")
	thumbDigest := evidence.ComputeSHA256Digest(thumbPayload)
	_, err = mgr.RegisterDerived(
		tenantID, thumbID, parentID, evidence.DerivationThumbnailPreview,
		"thumb.jpg", "image/jpeg", int64(len(thumbPayload)), thumbDigest, "pipeline",
	)
	if err != nil {
		t.Fatalf("RegisterDerived failed: %v", err)
	}
	if err := mgr.CommitDerived(tenantID, thumbID, thumbPayload, "pipeline"); err != nil {
		t.Fatalf("CommitDerived failed: %v", err)
	}

	// Attempt nested derivation (derived from derived) must fail closed
	_, err = mgr.RegisterDerived(
		tenantID, "evd_syn_nested_deriv", thumbID, evidence.DerivationCompressedRendition,
		"compressed_thumb.jpg", "image/jpeg", 50, "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", "pipeline",
	)
	if !errors.Is(err, evidence.ErrNestedDerivationProhibited) {
		t.Errorf("expected ErrNestedDerivationProhibited, got %v", err)
	}
}

func TestEvidenceChain_TransferInterruptionAndResume(t *testing.T) {
	storage := evidence.NewMemoryStorageAdapter()
	mgr := evidence.NewEvidenceIntegrityManager(storage)

	tenantID := "ten_alpha"
	evidenceID := "evd_syn_interrupted_001"
	payload := []byte("synthetic-payload-for-interruption-test")
	digest := evidence.ComputeSHA256Digest(payload)

	_, err := mgr.RegisterOriginal(
		tenantID, evidenceID, "flapping_connection.jpg", "image/jpeg",
		int64(len(payload)), digest, "INSPECTION_RESPONSE", "rsp_syn_02", "inspector_alice",
	)
	if err != nil {
		t.Fatalf("RegisterOriginal failed: %v", err)
	}

	// Simulate transfer interruption
	err = mgr.RecordTransferInterrupted(tenantID, evidenceID, 1024, int64(len(payload)), "TCP connection reset by peer", "network_monitor")
	if err != nil {
		t.Fatalf("RecordTransferInterrupted failed: %v", err)
	}

	rec, _ := mgr.GetRecord(tenantID, evidenceID)
	if rec.State != evidence.StateFailed {
		t.Errorf("expected StateFailed after interruption, got %q", rec.State)
	}

	// Resume transfer
	err = mgr.RecordTransferResumed(tenantID, evidenceID, "inspector_alice")
	if err != nil {
		t.Fatalf("RecordTransferResumed failed: %v", err)
	}

	rec, _ = mgr.GetRecord(tenantID, evidenceID)
	if rec.State != evidence.StateTransferring {
		t.Errorf("expected StateTransferring after resume, got %q", rec.State)
	}

	// Commit payload after resume
	err = mgr.CommitOriginal(tenantID, evidenceID, payload, "inspector_alice")
	if err != nil {
		t.Fatalf("CommitOriginal after resume failed: %v", err)
	}

	chain, _ := mgr.GetCustodyChain(tenantID, evidenceID)
	hasInterrupted := false
	hasResumed := false
	for _, ev := range chain {
		if ev.EventType == evidence.EventTransferInterrupted {
			hasInterrupted = true
		}
		if ev.EventType == evidence.EventTransferResumed {
			hasResumed = true
		}
	}
	if !hasInterrupted || !hasResumed {
		t.Errorf("custody chain must record interruption and resumption events")
	}
}

func TestEvidenceChain_DuplicateHandling(t *testing.T) {
	storage := evidence.NewMemoryStorageAdapter()
	mgr := evidence.NewEvidenceIntegrityManager(storage)

	tenantID := "ten_alpha"
	evidenceID := "evd_syn_duplicate_001"
	payload := []byte("original-payload-for-duplicate-test")
	digest := evidence.ComputeSHA256Digest(payload)

	_, err := mgr.RegisterOriginal(
		tenantID, evidenceID, "duplicate_test.png", "image/png",
		int64(len(payload)), digest, "SAFETY_FINDING", "fnd_syn_02", "inspector_alice",
	)
	if err != nil {
		t.Fatalf("RegisterOriginal failed: %v", err)
	}
	if err := mgr.CommitOriginal(tenantID, evidenceID, payload, "inspector_alice"); err != nil {
		t.Fatalf("CommitOriginal failed: %v", err)
	}

	// Idempotent duplicate: identical digest
	isDup, err := mgr.HandleDuplicateUpload(tenantID, evidenceID, digest, payload, "client_retry_agent")
	if err != nil {
		t.Fatalf("HandleDuplicateUpload failed on identical payload: %v", err)
	}
	if !isDup {
		t.Errorf("expected HandleDuplicateUpload=true for idempotent duplicate")
	}

	// Conflicting duplicate: different payload
	conflictingPayload := []byte("different-payload-violating-hash")
	conflictingDigest := evidence.ComputeSHA256Digest(conflictingPayload)
	_, err = mgr.HandleDuplicateUpload(tenantID, evidenceID, conflictingDigest, conflictingPayload, "attacker")
	if !errors.Is(err, evidence.ErrDuplicateEvidenceConflict) {
		t.Errorf("expected ErrDuplicateEvidenceConflict on conflicting digest, got %v", err)
	}
}

func TestEvidenceChain_TamperDetection(t *testing.T) {
	storage := evidence.NewMemoryStorageAdapter()
	mgr := evidence.NewEvidenceIntegrityManager(storage)

	tenantID := "ten_alpha"
	evidenceID := "evd_syn_tamper_001"
	legitPayload := []byte("authentic-inspection-evidence-photo")
	legitDigest := evidence.ComputeSHA256Digest(legitPayload)

	_, err := mgr.RegisterOriginal(
		tenantID, evidenceID, "legit.jpg", "image/jpeg",
		int64(len(legitPayload)), legitDigest, "INSPECTION_RESPONSE", "rsp_syn_03", "inspector_bob",
	)
	if err != nil {
		t.Fatalf("RegisterOriginal failed: %v", err)
	}
	if err := mgr.CommitOriginal(tenantID, evidenceID, legitPayload, "inspector_bob"); err != nil {
		t.Fatalf("CommitOriginal failed: %v", err)
	}

	// Legitimate verification
	valid, err := mgr.VerifyTamper(tenantID, evidenceID, legitPayload, "auditor_carol")
	if err != nil || !valid {
		t.Fatalf("VerifyTamper failed on legitimate payload: %v", err)
	}

	// Tampered payload
	tamperedPayload := []byte("tampered-corrupted-inspection-photo-bytes")
	valid, err = mgr.VerifyTamper(tenantID, evidenceID, tamperedPayload, "auditor_carol")
	if !errors.Is(err, evidence.ErrTamperDetected) || valid {
		t.Fatalf("expected ErrTamperDetected, got valid=%v, err=%v", valid, err)
	}

	// Verify tamper event was logged in chain of custody
	chain, _ := mgr.GetCustodyChain(tenantID, evidenceID)
	foundTamperEvent := false
	for _, ev := range chain {
		if ev.EventType == evidence.EventTamperDetected {
			foundTamperEvent = true
			break
		}
	}
	if !foundTamperEvent {
		t.Errorf("expected EventTamperDetected in custody chain")
	}
}

func TestEvidenceChain_ExportManifestGenerationAndVerification(t *testing.T) {
	storage := evidence.NewMemoryStorageAdapter()
	mgr := evidence.NewEvidenceIntegrityManager(storage)

	tenantID := "ten_alpha"
	p1 := []byte("evidence-item-one-clean-data")
	p2 := []byte("evidence-item-two-clean-data")
	d1 := evidence.ComputeSHA256Digest(p1)
	d2 := evidence.ComputeSHA256Digest(p2)

	_, _ = mgr.RegisterOriginal(tenantID, "evd_01", "one.jpg", "image/jpeg", int64(len(p1)), d1, "INSPECTION_RESPONSE", "rsp_01", "user1")
	_ = mgr.CommitOriginal(tenantID, "evd_01", p1, "user1")

	_, _ = mgr.RegisterOriginal(tenantID, "evd_02", "two.png", "image/png", int64(len(p2)), d2, "SAFETY_FINDING", "fnd_01", "user2")
	_ = mgr.CommitOriginal(tenantID, "evd_02", p2, "user2")

	// Generate export manifest
	manifest, err := mgr.GenerateExportManifest(tenantID, "exp_pkg_001", []string{"evd_01", "evd_02"}, "export_operator")
	if err != nil {
		t.Fatalf("GenerateExportManifest failed: %v", err)
	}
	if manifest.ExportID != "exp_pkg_001" {
		t.Errorf("expected ExportID exp_pkg_001, got %q", manifest.ExportID)
	}
	if len(manifest.Items) != 2 {
		t.Fatalf("expected 2 items in manifest, got %d", len(manifest.Items))
	}
	if manifest.RootDigest == "" {
		t.Errorf("manifest root digest cannot be empty")
	}

	// Verify export manifest with intact payloads
	payloads := map[string][]byte{
		"evd_01": p1,
		"evd_02": p2,
	}
	if err := evidence.VerifyExportManifest(manifest, payloads); err != nil {
		t.Fatalf("VerifyExportManifest failed on intact package: %v", err)
	}

	// Tamper payload in package
	tamperedPayloads := map[string][]byte{
		"evd_01": []byte("corrupted-item-one-bytes"),
		"evd_02": p2,
	}
	if err := evidence.VerifyExportManifest(manifest, tamperedPayloads); !errors.Is(err, evidence.ErrExportTampered) {
		t.Errorf("expected ErrExportTampered on tampered payload, got %v", err)
	}

	// Tamper root digest
	tamperedManifest := *manifest
	tamperedManifest.RootDigest = "0000000000000000000000000000000000000000000000000000000000000000"
	if err := evidence.VerifyExportManifest(&tamperedManifest, payloads); !errors.Is(err, evidence.ErrExportTampered) {
		t.Errorf("expected ErrExportTampered on tampered root digest, got %v", err)
	}
}

func TestEvidenceChain_TenantIsolation(t *testing.T) {
	storage := evidence.NewMemoryStorageAdapter()
	mgr := evidence.NewEvidenceIntegrityManager(storage)

	tenantA := "ten_alpha"
	tenantB := "ten_bravo"
	payload := []byte("tenant-isolated-evidence-data")
	digest := evidence.ComputeSHA256Digest(payload)

	_, err := mgr.RegisterOriginal(
		tenantA, "evd_isolated_01", "iso.jpg", "image/jpeg",
		int64(len(payload)), digest, "INSPECTION_RESPONSE", "rsp_01", "alice",
	)
	if err != nil {
		t.Fatalf("RegisterOriginal failed: %v", err)
	}
	_ = mgr.CommitOriginal(tenantA, "evd_isolated_01", payload, "alice")

	// Cross-tenant retrieval must fail with ErrObjectNotFound
	_, err = mgr.GetRecord(tenantB, "evd_isolated_01")
	if !errors.Is(err, evidence.ErrObjectNotFound) {
		t.Errorf("expected ErrObjectNotFound for cross-tenant retrieval, got %v", err)
	}

	// Cross-tenant derivation must fail with ErrMissingParentEvidence
	_, err = mgr.RegisterDerived(
		tenantB, "evd_thumb_cross", "evd_isolated_01", evidence.DerivationThumbnailPreview,
		"cross.jpg", "image/jpeg", 50, "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", "bob",
	)
	if !errors.Is(err, evidence.ErrMissingParentEvidence) {
		t.Errorf("expected ErrMissingParentEvidence for cross-tenant derivation, got %v", err)
	}
}

func TestComputeSHA256Digest(t *testing.T) {
	emptyHash := evidence.ComputeSHA256Digest([]byte{})
	expectedEmpty := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	if emptyHash != expectedEmpty {
		t.Errorf("expected %s, got %s", expectedEmpty, emptyHash)
	}
}
