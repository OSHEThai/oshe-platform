package identifiers

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
)

var (
	// ErrEmptyTenantID indicates that a tenant identifier is blank or empty.
	ErrEmptyTenantID = errors.New("tenant_id cannot be empty")
	// ErrEmptySystem indicates that the external system identifier is blank or empty.
	ErrEmptySystem = errors.New("external system name cannot be empty")
	// ErrEmptyExternalID indicates that the external identifier is blank or empty.
	ErrEmptyExternalID = errors.New("external_id cannot be empty")
	// ErrEmptyLookupToken indicates that the lookup token is blank or empty.
	ErrEmptyLookupToken = errors.New("lookup_token cannot be empty")
	// ErrReferenceNotFound indicates that no external reference mapping exists.
	ErrReferenceNotFound = errors.New("external reference not found")
	// ErrTenantMismatch indicates that an access attempt occurred with a mismatched tenant context.
	ErrTenantMismatch = errors.New("tenant context mismatch: access denied")
	// ErrReferenceConflict indicates that the external reference was previously bound to a different internal ID.
	ErrReferenceConflict = errors.New("external reference conflict: already bound to a different internal ID")
)

// OpaqueReference represents the public return of an external reference lookup,
// intentionally concealing the raw internal identifier behind a non-enumerable lookup token.
type OpaqueReference struct {
	TenantID    string `json:"tenant_id"`
	System      string `json:"system"`
	ExternalID  string `json:"external_id"`
	LookupToken string `json:"lookup_token"`
}

type referenceEntry struct {
	tenantID    string
	system      string
	externalID  string
	internalID  ID
	lookupToken string
}

// ExternalReferenceRegistry manages tenant-isolated mappings between external third-party
// references and internal canonical identifiers in a thread-safe, fail-closed manner.
type ExternalReferenceRegistry struct {
	mu            sync.RWMutex
	byExtKey      map[string]referenceEntry // key: tenantID + "\x00" + system + "\x00" + externalID
	byLookupToken map[string]referenceEntry // key: tenantID + "\x00" + lookupToken
}

// NewExternalReferenceRegistry initializes an empty external reference registry.
func NewExternalReferenceRegistry() *ExternalReferenceRegistry {
	return &ExternalReferenceRegistry{
		byExtKey:      make(map[string]referenceEntry),
		byLookupToken: make(map[string]referenceEntry),
	}
}

func makeExtKey(tenantID, system, externalID string) string {
	return fmt.Sprintf("%s\x00%s\x00%s", tenantID, system, externalID)
}

func makeTokenKey(tenantID, lookupToken string) string {
	return fmt.Sprintf("%s\x00%s", tenantID, lookupToken)
}

func generateLookupToken() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("failed to generate lookup token entropy: %w", err)
	}
	return "ref_" + hex.EncodeToString(raw[:]), nil
}

// Register establishes a binding between an external reference and an internal canonical ID.
// Re-registering the identical mapping is idempotent. Re-registering with a different internal ID returns ErrReferenceConflict.
func (r *ExternalReferenceRegistry) Register(tenantID, system, externalID string, internalID ID) (OpaqueReference, error) {
	tTenant := strings.TrimSpace(tenantID)
	if tTenant == "" {
		return OpaqueReference{}, ErrEmptyTenantID
	}
	tSystem := strings.TrimSpace(system)
	if tSystem == "" {
		return OpaqueReference{}, ErrEmptySystem
	}
	tExtID := strings.TrimSpace(externalID)
	if tExtID == "" {
		return OpaqueReference{}, ErrEmptyExternalID
	}
	if strings.TrimSpace(string(internalID)) == "" {
		return OpaqueReference{}, ErrEmptyID
	}

	extKey := makeExtKey(tTenant, tSystem, tExtID)

	r.mu.Lock()
	defer r.mu.Unlock()

	if existing, found := r.byExtKey[extKey]; found {
		if existing.internalID == internalID {
			return OpaqueReference{
				TenantID:    existing.tenantID,
				System:      existing.system,
				ExternalID:  existing.externalID,
				LookupToken: existing.lookupToken,
			}, nil
		}
		return OpaqueReference{}, fmt.Errorf("%w: system %q ID %q already bound to %q",
			ErrReferenceConflict, tSystem, tExtID, existing.internalID)
	}

	token, err := generateLookupToken()
	if err != nil {
		return OpaqueReference{}, err
	}

	entry := referenceEntry{
		tenantID:    tTenant,
		system:      tSystem,
		externalID:  tExtID,
		internalID:  internalID,
		lookupToken: token,
	}

	r.byExtKey[extKey] = entry
	r.byLookupToken[makeTokenKey(tTenant, token)] = entry

	return OpaqueReference{
		TenantID:    tTenant,
		System:      tSystem,
		ExternalID:  tExtID,
		LookupToken: token,
	}, nil
}

// LookupPublic retrieves the public opaque reference handle for a given external identifier.
// Fails closed if the mapping is unknown or belongs to a different tenant.
// Crucially, this returns only the OpaqueReference and never reveals the internal ID.
func (r *ExternalReferenceRegistry) LookupPublic(tenantID, system, externalID string) (OpaqueReference, error) {
	tTenant := strings.TrimSpace(tenantID)
	if tTenant == "" {
		return OpaqueReference{}, ErrEmptyTenantID
	}
	tSystem := strings.TrimSpace(system)
	if tSystem == "" {
		return OpaqueReference{}, ErrEmptySystem
	}
	tExtID := strings.TrimSpace(externalID)
	if tExtID == "" {
		return OpaqueReference{}, ErrEmptyExternalID
	}

	extKey := makeExtKey(tTenant, tSystem, tExtID)

	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, found := r.byExtKey[extKey]
	if !found {
		return OpaqueReference{}, ErrReferenceNotFound
	}
	if entry.tenantID != tTenant {
		return OpaqueReference{}, ErrTenantMismatch
	}

	return OpaqueReference{
		TenantID:    entry.tenantID,
		System:      entry.system,
		ExternalID:  entry.externalID,
		LookupToken: entry.lookupToken,
	}, nil
}

// ResolveInternal safely resolves an opaque lookup token back to the internal canonical ID
// strictly within the authorized tenant context. Fails closed if token is invalid or belongs to another tenant.
func (r *ExternalReferenceRegistry) ResolveInternal(tenantID, lookupToken string) (ID, error) {
	tTenant := strings.TrimSpace(tenantID)
	if tTenant == "" {
		return "", ErrEmptyTenantID
	}
	tToken := strings.TrimSpace(lookupToken)
	if tToken == "" {
		return "", ErrEmptyLookupToken
	}

	tokenKey := makeTokenKey(tTenant, tToken)

	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, found := r.byLookupToken[tokenKey]
	if !found {
		return "", ErrReferenceNotFound
	}
	if entry.tenantID != tTenant {
		return "", ErrTenantMismatch
	}

	return entry.internalID, nil
}

// Count returns the number of registered reference mappings.
func (r *ExternalReferenceRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.byExtKey)
}
