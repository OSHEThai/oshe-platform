package recordsaudit_test

import (
	"errors"
	"testing"

	recordsaudit "github.com/oshethai/oshe-platform/modules/records-audit"
)

const (
	sampleDigestA = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	sampleDigestB = "fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210"
	validCorrID   = "corr_0123456789abcdef0123456789abcdef"
	validCausID   = "caus_fedcba9876543210fedcba9876543210"
)

func TestRecordStore_DeclareRecord_Valid(t *testing.T) {
	store := recordsaudit.NewRecordStore()
	rec, err := store.DeclareRecord(
		"ten_001",
		"rec_insp_101",
		"INSPECTION_RECORD",
		"v1",
		sampleDigestA,
		"user_auditor_alice",
		validCorrID,
		validCausID,
	)
	if err != nil {
		t.Fatalf("unexpected declaration error: %v", err)
	}

	if rec.RecordID != "rec_insp_101" {
		t.Errorf("expected RecordID 'rec_insp_101', got %q", rec.RecordID)
	}
	if rec.TenantID != "ten_001" {
		t.Errorf("expected TenantID 'ten_001', got %q", rec.TenantID)
	}
	if rec.State != recordsaudit.StateAccepted {
		t.Errorf("expected StateAccepted, got %q", rec.State)
	}
	if rec.CurrentVersion != "v1" {
		t.Errorf("expected CurrentVersion 'v1', got %q", rec.CurrentVersion)
	}

	// Verify snapshot created
	snaps, err := store.GetSnapshots("ten_001", "rec_insp_101")
	if err != nil {
		t.Fatalf("failed to retrieve snapshots: %v", err)
	}
	if len(snaps) != 1 {
		t.Fatalf("expected 1 snapshot, got %d", len(snaps))
	}
	if snaps[0].Version != "v1" || snaps[0].PayloadDigest != sampleDigestA {
		t.Errorf("snapshot mismatch: %+v", snaps[0])
	}

	// Verify audit trail contains RECORD_DECLARED
	audit, err := store.GetAuditTrail("ten_001", "rec_insp_101")
	if err != nil {
		t.Fatalf("failed to retrieve audit trail: %v", err)
	}
	if len(audit) != 1 {
		t.Fatalf("expected 1 audit entry, got %d", len(audit))
	}
	if audit[0].EventType != recordsaudit.AuditDeclared {
		t.Errorf("expected AuditDeclared event, got %q", audit[0].EventType)
	}
	if audit[0].SequenceNumber != 1 {
		t.Errorf("expected sequence number 1, got %d", audit[0].SequenceNumber)
	}
}

func TestRecordStore_DuplicateRecordDenial(t *testing.T) {
	store := recordsaudit.NewRecordStore()
	_, err := store.DeclareRecord("ten_001", "rec_duplicate", "REPORT", "v1", sampleDigestA, "alice", validCorrID, validCausID)
	if err != nil {
		t.Fatalf("initial declaration failed: %v", err)
	}

	_, err = store.DeclareRecord("ten_001", "rec_duplicate", "REPORT", "v1", sampleDigestB, "bob", validCorrID, validCausID)
	if err == nil {
		t.Fatal("expected duplicate record error, got nil")
	}
	if !errors.Is(err, recordsaudit.ErrDuplicateRecord) {
		t.Errorf("expected ErrDuplicateRecord, got: %v", err)
	}
}

func TestRecordStore_VersionPreservationAndSnapshotImmutability(t *testing.T) {
	store := recordsaudit.NewRecordStore()
	_, err := store.DeclareRecord("ten_001", "rec_versioned", "REPORT", "v1", sampleDigestA, "alice", validCorrID, validCausID)
	if err != nil {
		t.Fatalf("initial declaration failed: %v", err)
	}

	// Create new version v2
	snapV2, err := store.CreateNewVersion("ten_001", "rec_versioned", "v2", sampleDigestB, "bob", validCorrID, validCausID)
	if err != nil {
		t.Fatalf("failed to create version v2: %v", err)
	}
	if snapV2.Version != "v2" {
		t.Errorf("expected version v2, got %q", snapV2.Version)
	}

	// Overwriting v1 or v2 must be rejected
	_, err = store.CreateNewVersion("ten_001", "rec_versioned", "v1", sampleDigestA, "charlie", validCorrID, validCausID)
	if err == nil {
		t.Fatal("expected error overwriting existing accepted version v1, got nil")
	}
	if !errors.Is(err, recordsaudit.ErrDuplicateVersion) {
		t.Errorf("expected ErrDuplicateVersion, got: %v", err)
	}

	// Verify all historical snapshots are preserved in immutable sequence
	snaps, err := store.GetSnapshots("ten_001", "rec_versioned")
	if err != nil {
		t.Fatalf("failed to retrieve snapshots: %v", err)
	}
	if len(snaps) != 2 {
		t.Fatalf("expected exactly 2 snapshots, got %d", len(snaps))
	}
	if snaps[0].Version != "v1" || snaps[0].PayloadDigest != sampleDigestA {
		t.Errorf("historical version v1 was mutated: %+v", snaps[0])
	}
	if snaps[1].Version != "v2" || snaps[1].PayloadDigest != sampleDigestB {
		t.Errorf("version v2 mismatch: %+v", snaps[1])
	}
}

func TestRecordStore_AppendOnlyAuditAndMonotonicOrdering(t *testing.T) {
	store := recordsaudit.NewRecordStore()
	_, _ = store.DeclareRecord("ten_001", "rec_audit_seq", "TYPE_A", "v1", sampleDigestA, "alice", validCorrID, validCausID)
	_, _ = store.CreateNewVersion("ten_001", "rec_audit_seq", "v2", sampleDigestB, "bob", validCorrID, validCausID)
	_ = store.TransitionRecordState("ten_001", "rec_audit_seq", recordsaudit.StateArchived, "admin", validCorrID, validCausID)

	audit, err := store.GetAuditTrail("ten_001", "rec_audit_seq")
	if err != nil {
		t.Fatalf("failed to get audit trail: %v", err)
	}
	if len(audit) != 3 {
		t.Fatalf("expected 3 audit entries, got %d", len(audit))
	}

	// Assert monotonic sequence numbering
	for i := range len(audit) {
		expectedSeq := int64(i + 1)
		if audit[i].SequenceNumber != expectedSeq {
			t.Errorf("audit entry %d: expected sequence %d, got %d", i, expectedSeq, audit[i].SequenceNumber)
		}
	}

	// Assert event types
	if audit[0].EventType != recordsaudit.AuditDeclared {
		t.Errorf("entry 0: expected AuditDeclared, got %s", audit[0].EventType)
	}
	if audit[1].EventType != recordsaudit.AuditVersioned {
		t.Errorf("entry 1: expected AuditVersioned, got %s", audit[1].EventType)
	}
	if audit[2].EventType != recordsaudit.AuditStateChanged {
		t.Errorf("entry 2: expected AuditStateChanged, got %s", audit[2].EventType)
	}
}

func TestRecordStore_AuditCompleteness(t *testing.T) {
	store := recordsaudit.NewRecordStore()
	_, _ = store.DeclareRecord("ten_001", "rec_complete", "INSPECTION", "v1", sampleDigestA, "officer_1", validCorrID, validCausID)

	audit, err := store.GetAuditTrail("ten_001", "rec_complete")
	if err != nil || len(audit) == 0 {
		t.Fatalf("failed to retrieve audit: %v", err)
	}

	entry := audit[0]
	if entry.ActorID != "officer_1" {
		t.Errorf("expected actor 'officer_1', got %q", entry.ActorID)
	}
	if entry.CorrelationID != validCorrID {
		t.Errorf("expected correlation %q, got %q", validCorrID, entry.CorrelationID)
	}
	if entry.CausationID != validCausID {
		t.Errorf("expected causation %q, got %q", validCausID, entry.CausationID)
	}
	if entry.Timestamp.IsZero() {
		t.Error("expected non-zero timestamp")
	}
	if entry.CurrentState != recordsaudit.StateAccepted {
		t.Errorf("expected state ACCEPTED, got %q", entry.CurrentState)
	}
}

func TestRecordStore_UnauthorizedAuditAccessAndCrossTenantDenial(t *testing.T) {
	store := recordsaudit.NewRecordStore()
	_, err := store.DeclareRecord("ten_alpha", "rec_secret_1", "CONFIDENTIAL", "v1", sampleDigestA, "alice", validCorrID, validCausID)
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	// Tenant Bravo attempts to access Tenant Alpha's audit trail
	_, err = store.GetAuditTrail("ten_bravo", "rec_secret_1")
	if err == nil {
		t.Fatal("security violation: tenant_bravo read tenant_alpha's audit trail")
	}
	if !errors.Is(err, recordsaudit.ErrCrossTenantAccess) {
		t.Errorf("expected ErrCrossTenantAccess, got: %v", err)
	}

	// Tenant Bravo attempts to access Tenant Alpha's record
	_, err = store.GetRecord("ten_bravo", "rec_secret_1")
	if !errors.Is(err, recordsaudit.ErrCrossTenantAccess) {
		t.Errorf("expected ErrCrossTenantAccess on GetRecord, got: %v", err)
	}

	// Tenant Bravo attempts to access Tenant Alpha's snapshots
	_, err = store.GetSnapshots("ten_bravo", "rec_secret_1")
	if !errors.Is(err, recordsaudit.ErrCrossTenantAccess) {
		t.Errorf("expected ErrCrossTenantAccess on GetSnapshots, got: %v", err)
	}

	// Empty tenant ID fails closed
	if _, err = store.GetAuditTrail("", "rec_secret_1"); !errors.Is(err, recordsaudit.ErrBlankTenantID) {
		t.Errorf("expected ErrBlankTenantID, got: %v", err)
	}
}

func TestRecordStore_InvalidLifecycleTransitions(t *testing.T) {
	store := recordsaudit.NewRecordStore()
	_, err := store.DeclareRecord("ten_001", "rec_archived", "TYPE", "v1", sampleDigestA, "alice", validCorrID, validCausID)
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	// Transition to ARCHIVED
	err = store.TransitionRecordState("ten_001", "rec_archived", recordsaudit.StateArchived, "admin", validCorrID, validCausID)
	if err != nil {
		t.Fatalf("archive transition failed: %v", err)
	}

	// Attempting to transition from ARCHIVED back to ACCEPTED must fail
	err = store.TransitionRecordState("ten_001", "rec_archived", recordsaudit.StateAccepted, "admin", validCorrID, validCausID)
	if !errors.Is(err, recordsaudit.ErrRecordArchived) {
		t.Errorf("expected ErrRecordArchived, got: %v", err)
	}

	// Attempting to add a new version to an archived record must fail
	_, err = store.CreateNewVersion("ten_001", "rec_archived", "v2", sampleDigestB, "admin", validCorrID, validCausID)
	if !errors.Is(err, recordsaudit.ErrRecordArchived) {
		t.Errorf("expected ErrRecordArchived on versioning archived record, got: %v", err)
	}
}

func TestRecordStore_CorrelationAndCausationValidation(t *testing.T) {
	store := recordsaudit.NewRecordStore()

	// Invalid correlation ID prefix
	_, err := store.DeclareRecord("ten_1", "rec_1", "TYPE", "v1", sampleDigestA, "alice", "invalid_corr", validCausID)
	if !errors.Is(err, recordsaudit.ErrInvalidCorrelationID) {
		t.Errorf("expected ErrInvalidCorrelationID for 'invalid_corr', got: %v", err)
	}

	// Invalid causation ID prefix
	_, err = store.DeclareRecord("ten_1", "rec_1", "TYPE", "v1", sampleDigestA, "alice", validCorrID, "invalid_caus")
	if !errors.Is(err, recordsaudit.ErrInvalidCausationID) {
		t.Errorf("expected ErrInvalidCausationID for 'invalid_caus', got: %v", err)
	}

	// Empty tracking IDs
	_, err = store.DeclareRecord("ten_1", "rec_1", "TYPE", "v1", sampleDigestA, "alice", "", validCausID)
	if !errors.Is(err, recordsaudit.ErrInvalidCorrelationID) {
		t.Errorf("expected ErrInvalidCorrelationID for empty correlation ID, got: %v", err)
	}
}

func TestRecordStore_AccessDeniedAuditLogging(t *testing.T) {
	store := recordsaudit.NewRecordStore()
	_, _ = store.DeclareRecord("ten_001", "rec_deny_test", "TYPE", "v1", sampleDigestA, "alice", validCorrID, validCausID)

	err := store.RecordAccessDenied("ten_001", "rec_deny_test", "unauthorized_user_bob", validCorrID, validCausID, "missing required role")
	if err != nil {
		t.Fatalf("failed to record access denied event: %v", err)
	}

	audit, err := store.GetAuditTrail("ten_001", "rec_deny_test")
	if err != nil {
		t.Fatalf("failed to retrieve audit: %v", err)
	}
	if len(audit) != 2 {
		t.Fatalf("expected 2 audit entries, got %d", len(audit))
	}
	if audit[1].EventType != recordsaudit.AuditAccessDenied {
		t.Errorf("expected AuditAccessDenied event, got %q", audit[1].EventType)
	}
	if audit[1].ActorID != "unauthorized_user_bob" {
		t.Errorf("expected actor bob, got %q", audit[1].ActorID)
	}
}
