package evidence_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"sync"
	"testing"

	evidence "github.com/oshethai/oshe-platform/modules/files-evidence"
)

func computeHash(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func TestMemoryStorageAdapter_PutAndGet(t *testing.T) {
	adapter := evidence.NewMemoryStorageAdapter()
	tenantID := "ten_alpha"
	storageKey := "inspections/photo_001.png"
	payload := []byte("synthetic-binary-payload-data-for-testing")
	digest := computeHash(payload)

	// PutObject
	err := adapter.PutObject(tenantID, storageKey, bytes.NewReader(payload), digest)
	if err != nil {
		t.Fatalf("failed to put object: %v", err)
	}

	// ObjectExists
	exists, err := adapter.ObjectExists(tenantID, storageKey)
	if err != nil {
		t.Fatalf("ObjectExists failed: %v", err)
	}
	if !exists {
		t.Error("expected object to exist")
	}

	// GetObject
	rc, err := adapter.GetObject(tenantID, storageKey)
	if err != nil {
		t.Fatalf("failed to get object: %v", err)
	}
	defer rc.Close()

	readBack, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("failed to read back object: %v", err)
	}
	if !bytes.Equal(readBack, payload) {
		t.Fatalf("payload mismatch: expected %q, got %q", string(payload), string(readBack))
	}
}

func TestMemoryStorageAdapter_PutDigestMismatch(t *testing.T) {
	adapter := evidence.NewMemoryStorageAdapter()
	tenantID := "ten_alpha"
	storageKey := "inspections/tampered.png"
	payload := []byte("legitimate-file-content")
	wrongDigest := "0000000000000000000000000000000000000000000000000000000000000000"

	err := adapter.PutObject(tenantID, storageKey, bytes.NewReader(payload), wrongDigest)
	if err == nil {
		t.Fatal("expected ErrIntegrityMismatch for tampered/wrong digest, got nil")
	}
	if !errors.Is(err, evidence.ErrIntegrityMismatch) {
		t.Errorf("expected ErrIntegrityMismatch, got: %v", err)
	}

	// Object must not be stored
	exists, _ := adapter.ObjectExists(tenantID, storageKey)
	if exists {
		t.Error("tampered object must not exist in storage adapter")
	}
}

func TestMemoryStorageAdapter_CrossTenantIsolation(t *testing.T) {
	adapter := evidence.NewMemoryStorageAdapter()
	tenantAlpha := "ten_hospital_a"
	tenantBeta := "ten_hospital_b"
	key := "confidential_record.pdf"
	payload := []byte("private-patient-safety-report")
	digest := computeHash(payload)

	// Tenant Alpha stores record
	if err := adapter.PutObject(tenantAlpha, key, bytes.NewReader(payload), digest); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	// Tenant Beta attempting to get Tenant Alpha's record must fail closed
	_, err := adapter.GetObject(tenantBeta, key)
	if err == nil {
		t.Fatal("security violation: tenant Beta retrieved tenant Alpha's object")
	}
	if !errors.Is(err, evidence.ErrObjectNotFound) {
		t.Errorf("expected ErrObjectNotFound for cross-tenant retrieval, got: %v", err)
	}

	// Tenant Beta checking existence must report false
	exists, err := adapter.ObjectExists(tenantBeta, key)
	if err != nil {
		t.Fatalf("unexpected error checking existence: %v", err)
	}
	if exists {
		t.Error("tenant Beta should not see existence of tenant Alpha's object")
	}

	// Tenant Beta attempting delete must fail closed
	err = adapter.DeleteObject(tenantBeta, key)
	if !errors.Is(err, evidence.ErrObjectNotFound) {
		t.Errorf("expected ErrObjectNotFound for cross-tenant delete, got: %v", err)
	}
}

func TestMemoryStorageAdapter_Delete(t *testing.T) {
	adapter := evidence.NewMemoryStorageAdapter()
	tenantID := "ten_1"
	key := "temp.png"
	payload := []byte("temp-data")
	digest := computeHash(payload)

	_ = adapter.PutObject(tenantID, key, bytes.NewReader(payload), digest)

	// Delete within tenant scope
	if err := adapter.DeleteObject(tenantID, key); err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	// Verify deleted
	exists, _ := adapter.ObjectExists(tenantID, key)
	if exists {
		t.Error("deleted object still exists")
	}

	_, err := adapter.GetObject(tenantID, key)
	if !errors.Is(err, evidence.ErrObjectNotFound) {
		t.Errorf("expected ErrObjectNotFound, got: %v", err)
	}
}

func TestMemoryStorageAdapter_SizeCapacityCheck(t *testing.T) {
	adapter := evidence.NewMemoryStorageAdapter()
	tenantID := "ten_1"
	key := "empty.png"
	emptyPayload := []byte("")
	emptyDigest := computeHash(emptyPayload)

	// Empty payload (0 bytes) must fail closed
	err := adapter.PutObject(tenantID, key, bytes.NewReader(emptyPayload), emptyDigest)
	if err == nil {
		t.Fatal("expected error for empty 0-byte payload, got nil")
	}
	if !errors.Is(err, evidence.ErrInvalidSize) {
		t.Errorf("expected ErrInvalidSize, got: %v", err)
	}
}

func TestMemoryStorageAdapter_Concurrency(t *testing.T) {
	adapter := evidence.NewMemoryStorageAdapter()
	const workers = 25
	var wg sync.WaitGroup

	for i := range workers {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			tenant := fmt.Sprintf("ten_worker_%d", workerID%4)
			key := fmt.Sprintf("files/obj_%d.dat", workerID)
			payload := []byte(fmt.Sprintf("worker-payload-content-%d", workerID))
			digest := computeHash(payload)

			if err := adapter.PutObject(tenant, key, bytes.NewReader(payload), digest); err != nil {
				t.Errorf("worker %d PutObject failed: %v", workerID, err)
				return
			}

			exists, err := adapter.ObjectExists(tenant, key)
			if err != nil || !exists {
				t.Errorf("worker %d ObjectExists failed: %v", workerID, err)
				return
			}

			rc, err := adapter.GetObject(tenant, key)
			if err != nil {
				t.Errorf("worker %d GetObject failed: %v", workerID, err)
				return
			}
			defer rc.Close()

			data, _ := io.ReadAll(rc)
			if !bytes.Equal(data, payload) {
				t.Errorf("worker %d payload mismatch", workerID)
			}
		}(i)
	}

	wg.Wait()
}
