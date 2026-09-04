package evidence

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
)

var (
	// ErrBlankStorageKey indicates that the storage object key is empty.
	ErrBlankStorageKey = errors.New("storage object key cannot be blank")
	// ErrIntegrityMismatch indicates that the uploaded payload digest does not match expected digest.
	ErrIntegrityMismatch = errors.New("integrity verification failed: payload SHA-256 digest does not match expected hash")
	// ErrObjectNotFound indicates that the requested object does not exist in the caller tenant scope.
	ErrObjectNotFound = errors.New("object not found in tenant storage scope")
	// ErrStorageCapacityExceeded indicates that payload size exceeds maximum single-file upload capacity.
	ErrStorageCapacityExceeded = errors.New("payload exceeds maximum allowable storage capacity")
)

// ScopedStorageAdapter defines a provider-neutral, tenant-scoped storage interface.
type ScopedStorageAdapter interface {
	PutObject(tenantID, storageKey string, r io.Reader, expectedDigest string) error
	GetObject(tenantID, storageKey string) (io.ReadCloser, error)
	DeleteObject(tenantID, storageKey string) error
	ObjectExists(tenantID, storageKey string) (bool, error)
}

// MemoryStorageAdapter provides an in-memory, thread-safe implementation of ScopedStorageAdapter.
// Objects are partitioned strictly by tenant ID to guarantee cross-tenant data isolation.
type MemoryStorageAdapter struct {
	mu      sync.RWMutex
	objects map[string]map[string][]byte // tenantID -> storageKey -> payload
}

// NewMemoryStorageAdapter initializes an empty in-memory scoped storage adapter.
func NewMemoryStorageAdapter() *MemoryStorageAdapter {
	return &MemoryStorageAdapter{
		objects: make(map[string]map[string][]byte),
	}
}

// PutObject stores a payload within the tenant's partition after verifying content SHA-256 digest.
// Fails closed if tenantID or storageKey is blank, size exceeds MaxFileSizeBytes, or digest mismatches.
func (a *MemoryStorageAdapter) PutObject(tenantID, storageKey string, r io.Reader, expectedDigest string) error {
	tTenant := strings.TrimSpace(tenantID)
	if tTenant == "" {
		return ErrEmptyTenantID
	}
	tKey := strings.TrimSpace(storageKey)
	if tKey == "" {
		return ErrBlankStorageKey
	}
	if r == nil {
		return errors.New("reader cannot be nil")
	}
	normDigest := strings.ToLower(strings.TrimSpace(expectedDigest))
	if err := ValidateDigest(normDigest); err != nil {
		return err
	}

	// Read stream up to limit + 1 byte to detect overflow
	limitedReader := io.LimitReader(r, MaxFileSizeBytes+1)
	hasher := sha256.New()
	var buf bytes.Buffer
	tee := io.TeeReader(limitedReader, hasher)

	n, err := buf.ReadFrom(tee)
	if err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("failed to read payload stream: %w", err)
	}
	if n > MaxFileSizeBytes {
		return ErrStorageCapacityExceeded
	}
	if n <= 0 {
		return fmt.Errorf("%w: payload must contain at least 1 byte", ErrInvalidSize)
	}

	calculatedDigest := hex.EncodeToString(hasher.Sum(nil))
	if calculatedDigest != normDigest {
		return fmt.Errorf("%w: expected %s, calculated %s", ErrIntegrityMismatch, normDigest, calculatedDigest)
	}

	payload := buf.Bytes()

	a.mu.Lock()
	defer a.mu.Unlock()

	if _, exists := a.objects[tTenant]; !exists {
		a.objects[tTenant] = make(map[string][]byte)
	}
	a.objects[tTenant][tKey] = payload

	return nil
}

// GetObject retrieves a tenant-scoped object stream.
// Fails closed with ErrObjectNotFound if tenant does not exist or object is absent.
func (a *MemoryStorageAdapter) GetObject(tenantID, storageKey string) (io.ReadCloser, error) {
	tTenant := strings.TrimSpace(tenantID)
	if tTenant == "" {
		return nil, ErrEmptyTenantID
	}
	tKey := strings.TrimSpace(storageKey)
	if tKey == "" {
		return nil, ErrBlankStorageKey
	}

	a.mu.RLock()
	defer a.mu.RUnlock()

	tenantStore, exists := a.objects[tTenant]
	if !exists {
		return nil, ErrObjectNotFound
	}
	payload, exists := tenantStore[tKey]
	if !exists {
		return nil, ErrObjectNotFound
	}

	// Return a copy wrapped in a ReadCloser
	copied := make([]byte, len(payload))
	copy(copied, payload)
	return io.NopCloser(bytes.NewReader(copied)), nil
}

// DeleteObject removes an object within caller tenant scope.
func (a *MemoryStorageAdapter) DeleteObject(tenantID, storageKey string) error {
	tTenant := strings.TrimSpace(tenantID)
	if tTenant == "" {
		return ErrEmptyTenantID
	}
	tKey := strings.TrimSpace(storageKey)
	if tKey == "" {
		return ErrBlankStorageKey
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	tenantStore, exists := a.objects[tTenant]
	if !exists {
		return ErrObjectNotFound
	}
	if _, exists := tenantStore[tKey]; !exists {
		return ErrObjectNotFound
	}

	delete(tenantStore, tKey)
	return nil
}

// ObjectExists checks whether an object exists strictly within caller tenant scope.
func (a *MemoryStorageAdapter) ObjectExists(tenantID, storageKey string) (bool, error) {
	tTenant := strings.TrimSpace(tenantID)
	if tTenant == "" {
		return false, ErrEmptyTenantID
	}
	tKey := strings.TrimSpace(storageKey)
	if tKey == "" {
		return false, ErrBlankStorageKey
	}

	a.mu.RLock()
	defer a.mu.RUnlock()

	tenantStore, exists := a.objects[tTenant]
	if !exists {
		return false, nil
	}
	_, exists = tenantStore[tKey]
	return exists, nil
}
