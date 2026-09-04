package recordsaudit_test

import (
	"errors"
	"testing"
	"time"

	recordsaudit "github.com/oshethai/oshe-platform/modules/records-audit"
)

const (
	validDigest1 = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	validDigest2 = "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
)

func setupStoreWithRecord(t *testing.T, tenantID, recordID, rType string) (*recordsaudit.RecordStore, *recordsaudit.RetentionManager) {
	t.Helper()
	store := recordsaudit.NewRecordStore()
	mgr := recordsaudit.NewRetentionManager(store)

	_, err := store.DeclareRecord(tenantID, recordID, rType, "1.0.0", validDigest1, "actor_admin", "corr_1", "caus_1")
	if err != nil {
		t.Fatalf("failed to declare initial record: %v", err)
	}

	return store, mgr
}

func TestRetention_LegalHoldBlocksDeletion(t *testing.T) {
	_, mgr := setupStoreWithRecord(t, "ten_alpha", "rec_001", "safety_inspection")
	now := time.Now().UTC()

	// Initially can delete
	canDel, err := mgr.CanDelete("ten_alpha", "rec_001", now)
	if err != nil || !canDel {
		t.Fatalf("expected CanDelete=true initially, got %v, err=%v", canDel, err)
	}

	// Place legal hold
	hold := recordsaudit.LegalHold{
		HoldID:   "hold_incident_404",
		TenantID: "ten_alpha",
		RecordID: "rec_001",
		Reason:   "Pending regulatory litigation",
		PlacedBy: "legal_officer_dan",
		PlacedAt: now,
	}
	if err := mgr.PlaceLegalHold(hold); err != nil {
		t.Fatalf("PlaceLegalHold failed: %v", err)
	}

	// Deletion must be blocked by active hold
	canDel, err = mgr.CanDelete("ten_alpha", "rec_001", now)
	if !errors.Is(err, recordsaudit.ErrActiveLegalHold) || canDel {
		t.Fatalf("expected ErrActiveLegalHold, got canDel=%v, err=%v", canDel, err)
	}

	// Release hold
	if err := mgr.ReleaseLegalHold("ten_alpha", "hold_incident_404", "legal_officer_dan", "case settled"); err != nil {
		t.Fatalf("ReleaseLegalHold failed: %v", err)
	}

	// Now deletion is unblocked
	canDel, err = mgr.CanDelete("ten_alpha", "rec_001", now)
	if err != nil || !canDel {
		t.Fatalf("expected CanDelete=true after hold release, got %v, err=%v", canDel, err)
	}
}

func TestRetention_RetentionPeriodActiveBlocksDeletion(t *testing.T) {
	_, mgr := setupStoreWithRecord(t, "ten_alpha", "rec_ret", "audit_log")
	createdAt := time.Now().UTC()

	// Policy: 7 days retention
	policy := recordsaudit.RetentionPolicy{
		PolicyID:        "pol_7d",
		TenantID:        "ten_alpha",
		RecordType:      "audit_log",
		RetentionPeriod: 7 * 24 * time.Hour,
		CreatedAt:       createdAt,
	}
	if err := mgr.RegisterRetentionPolicy(policy); err != nil {
		t.Fatalf("RegisterRetentionPolicy failed: %v", err)
	}

	// Check at +2 days: blocked by retention
	canDel, err := mgr.CanDelete("ten_alpha", "rec_ret", createdAt.Add(2*24*time.Hour))
	if !errors.Is(err, recordsaudit.ErrRetentionPeriodActive) || canDel {
		t.Fatalf("expected ErrRetentionPeriodActive at 2 days, got canDel=%v, err=%v", canDel, err)
	}

	// Check at +8 days: retention expired, deletion permitted
	canDel, err = mgr.CanDelete("ten_alpha", "rec_ret", createdAt.Add(8*24*time.Hour))
	if err != nil || !canDel {
		t.Fatalf("expected deletion allowed after 8 days, got canDel=%v, err=%v", canDel, err)
	}
}

func TestRetention_CrossTenantHoldDenial(t *testing.T) {
	_, mgr := setupStoreWithRecord(t, "ten_alpha", "rec_alpha_01", "report")

	// Attempt to place hold under ten_bravo on ten_alpha record -> ErrNotFound
	hold := recordsaudit.LegalHold{
		HoldID:   "hold_cross",
		TenantID: "ten_bravo",
		RecordID: "rec_alpha_01",
		Reason:   "Cross-tenant attempt",
	}
	err := mgr.PlaceLegalHold(hold)
	if !errors.Is(err, recordsaudit.ErrRecordNotFound) {
		t.Fatalf("expected ErrNotFound placing hold across tenant boundaries, got: %v", err)
	}

	// Place valid hold under ten_alpha
	validHold := recordsaudit.LegalHold{
		HoldID:   "hold_valid",
		TenantID: "ten_alpha",
		RecordID: "rec_alpha_01",
		Reason:   "Valid hold",
	}
	_ = mgr.PlaceLegalHold(validHold)

	// Attempt release from ten_bravo -> ErrCrossTenantAccess
	err = mgr.ReleaseLegalHold("ten_bravo", "hold_valid", "actor_bravo", "unauthorized release")
	if !errors.Is(err, recordsaudit.ErrCrossTenantAccess) {
		t.Fatalf("expected ErrCrossTenantAccess releasing hold from wrong tenant, got: %v", err)
	}
}

func TestExportPackage_CompletenessAndTenantIsolation(t *testing.T) {
	store := recordsaudit.NewRecordStore()
	mgr := recordsaudit.NewRetentionManager(store)

	// Declare 2 records in ten_alpha
	_, _ = store.DeclareRecord("ten_alpha", "rec_a1", "inspection", "1.0.0", validDigest1, "admin", "corr_1", "caus_1")
	_, _ = store.CreateNewVersion("ten_alpha", "rec_a1", "1.1.0", validDigest2, "admin", "corr_2", "caus_2")

	_, _ = store.DeclareRecord("ten_alpha", "rec_a2", "certificate", "1.0.0", validDigest1, "admin", "corr_3", "caus_3")

	// Declare 1 record in ten_bravo
	_, _ = store.DeclareRecord("ten_bravo", "rec_b1", "inspection", "1.0.0", validDigest1, "admin", "corr_4", "caus_4")

	// Export ten_alpha
	pkg, err := mgr.GenerateExportPackage("ten_alpha", "auditor_eve")
	if err != nil {
		t.Fatalf("GenerateExportPackage failed: %v", err)
	}

	if pkg.ItemCount != 2 {
		t.Fatalf("expected 2 items for ten_alpha, got %d", pkg.ItemCount)
	}
	if len(pkg.PackageDigest) != 64 {
		t.Fatalf("expected 64-char package digest, got %q", pkg.PackageDigest)
	}

	// Assert record a1 has both snapshots and audit entries
	item0 := pkg.Items[0]
	if item0.RecordID != "rec_a1" || len(item0.Snapshots) != 2 {
		t.Errorf("item 0 mismatch: %+v", item0)
	}

	// Assert ten_bravo is completely excluded
	for _, item := range pkg.Items {
		if item.RecordID == "rec_b1" {
			t.Errorf("ten_bravo record leaked into ten_alpha export: %+v", item)
		}
	}

	// Verify export integrity
	if err := recordsaudit.VerifyExportIntegrity(pkg); err != nil {
		t.Fatalf("VerifyExportIntegrity failed: %v", err)
	}
}

func TestExportPackage_TamperDetection(t *testing.T) {
	store := recordsaudit.NewRecordStore()
	mgr := recordsaudit.NewRetentionManager(store)

	_, _ = store.DeclareRecord("ten_alpha", "rec_orig", "doc", "1.0.0", validDigest1, "admin", "corr_1", "caus_1")
	pkg, _ := mgr.GenerateExportPackage("ten_alpha", "admin")

	// 1. Bit-level tamper: change payload digest of an item
	tamperedPkg := pkg
	tamperedPkg.Items[0].PayloadDigest = "0000000000000000000000000000000000000000000000000000000000000000"
	err := recordsaudit.VerifyExportIntegrity(tamperedPkg)
	if !errors.Is(err, recordsaudit.ErrExportTampered) {
		t.Fatalf("expected ErrExportTampered on corrupted item digest, got: %v", err)
	}

	// 2. Count tamper
	tamperedPkg2 := pkg
	tamperedPkg2.ItemCount = 99
	err = recordsaudit.VerifyExportIntegrity(tamperedPkg2)
	if !errors.Is(err, recordsaudit.ErrExportTampered) {
		t.Fatalf("expected ErrExportTampered on item count tampering, got: %v", err)
	}
}

func TestSyntheticBackup_AndRestore(t *testing.T) {
	store := recordsaudit.NewRecordStore()
	mgr := recordsaudit.NewRetentionManager(store)

	// Populate data
	_, _ = store.DeclareRecord("ten_alpha", "rec_bk1", "typeA", "1.0.0", validDigest1, "admin", "corr_1", "caus_1")
	_, _ = store.CreateNewVersion("ten_alpha", "rec_bk1", "2.0.0", validDigest2, "admin", "corr_2", "caus_2")

	_ = mgr.RegisterRetentionPolicy(recordsaudit.RetentionPolicy{
		PolicyID:        "p1",
		TenantID:        "ten_alpha",
		RecordType:      "typeA",
		RetentionPeriod: 24 * time.Hour,
	})

	_ = mgr.PlaceLegalHold(recordsaudit.LegalHold{
		HoldID:   "h1",
		TenantID: "ten_alpha",
		RecordID: "rec_bk1",
		Reason:   "audit hold",
		Active:   true,
	})

	// Create synthetic backup
	backup, err := mgr.CreateSyntheticBackup()
	if err != nil {
		t.Fatalf("CreateSyntheticBackup failed: %v", err)
	}
	if len(backup.ArchiveDigest) != 64 {
		t.Fatalf("expected 64-char archive digest, got %q", backup.ArchiveDigest)
	}

	// Create clean second manager and restore
	store2 := recordsaudit.NewRecordStore()
	mgr2 := recordsaudit.NewRetentionManager(store2)

	if err := mgr2.RestoreSyntheticBackup(backup); err != nil {
		t.Fatalf("RestoreSyntheticBackup failed: %v", err)
	}

	// Verify restored record and version snapshots
	rec, err := store2.GetRecord("ten_alpha", "rec_bk1")
	if err != nil || rec.CurrentVersion != "2.0.0" {
		t.Fatalf("restored record mismatch: %+v, err=%v", rec, err)
	}

	snaps, err := store2.GetSnapshots("ten_alpha", "rec_bk1")
	if err != nil || len(snaps) != 2 {
		t.Fatalf("expected 2 restored snapshots, got %d, err=%v", len(snaps), err)
	}

	// Verify legal hold is active on restored manager
	canDel, err := mgr2.CanDelete("ten_alpha", "rec_bk1", time.Now().UTC())
	if !errors.Is(err, recordsaudit.ErrActiveLegalHold) {
		t.Fatalf("expected active legal hold to be restored, got canDel=%v, err=%v", canDel, err)
	}
}

func TestSyntheticBackup_TamperRejection(t *testing.T) {
	store := recordsaudit.NewRecordStore()
	mgr := recordsaudit.NewRetentionManager(store)
	_, _ = store.DeclareRecord("ten_alpha", "rec_tamper", "type", "1.0.0", validDigest1, "admin", "corr_1", "caus_1")

	backup, _ := mgr.CreateSyntheticBackup()

	// Corrupt archive digest
	tamperedBackup := backup
	tamperedBackup.ArchiveDigest = "1111111111111111111111111111111111111111111111111111111111111111"

	store2 := recordsaudit.NewRecordStore()
	mgr2 := recordsaudit.NewRetentionManager(store2)

	err := mgr2.RestoreSyntheticBackup(tamperedBackup)
	if !errors.Is(err, recordsaudit.ErrBackupTampered) {
		t.Fatalf("expected ErrBackupTampered on corrupted backup restore, got: %v", err)
	}
}

func TestRecord_TamperDetection(t *testing.T) {
	_, mgr := setupStoreWithRecord(t, "ten_alpha", "rec_tamper_check", "inspection")

	// Matching expected digest passes
	if err := mgr.VerifyRecordIntegrity("ten_alpha", "rec_tamper_check", validDigest1); err != nil {
		t.Fatalf("expected valid integrity check, got: %v", err)
	}

	// Corrupted expected digest fails closed with ErrTamperDetected
	if err := mgr.VerifyRecordIntegrity("ten_alpha", "rec_tamper_check", validDigest2); !errors.Is(err, recordsaudit.ErrTamperDetected) {
		t.Fatalf("expected ErrTamperDetected on digest mismatch, got: %v", err)
	}
}
