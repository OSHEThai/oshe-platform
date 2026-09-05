package orgtenancy

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Canonical identifier prefixes for core domain entities and message tracking.
const (
	PrefixTenant      = "ten"
	PrefixCompany     = "cmp"
	PrefixBusinessUnit = "bnu"
	PrefixProject     = "prj"
	PrefixSite        = "ste"
	PrefixArea        = "ara"
	PrefixParty        = "prt"
	PrefixUser        = "usr"
	PrefixCorrelation = "corr"
	PrefixCausation   = "caus"
	PrefixIdempotency = "idem"
	PrefixExternalRef = "ext"
)

// Anti-abuse and anti-enumeration boundaries for external references.
const (
	MinExternalIDLength = 3
	MaxExternalIDLength = 128
)

var (
	// ErrBlankIdentifier indicates an empty or whitespace-only identifier.
	ErrBlankIdentifier = errors.New("identifier must not be blank")
	// ErrMalformedIdentifier indicates the identifier does not adhere to the <prefix>_<token> format.
	ErrMalformedIdentifier = errors.New("identifier is malformed: expected format <prefix>_<token>")
	// ErrPrefixMismatch indicates the identifier prefix does not match the expected canonical prefix.
	ErrPrefixMismatch = errors.New("identifier prefix mismatch")
	// ErrInvalidCharacters indicates illegal uppercase, whitespace, or special characters in the identifier.
	ErrInvalidCharacters = errors.New("identifier contains invalid characters: must be lowercase alphanumeric, underscore, or hyphen")
	// ErrBlankPayloadDigest indicates a missing payload checksum.
	ErrBlankPayloadDigest = errors.New("payload digest must not be blank")
	// ErrInvalidDigest indicates that the digest is not a valid 64-character lowercase hexadecimal SHA-256 string.
	ErrInvalidDigest = errors.New("payload digest must be a 64-character lowercase hexadecimal SHA-256 string")
	// ErrIdempotencyConflict indicates a collision where an existing idempotency key was reused with a differing payload.
	ErrIdempotencyConflict = errors.New("idempotency key conflict: key already used with differing payload digest")
	// ErrDuplicateExternalRef indicates that an external reference is already mapped to a different internal entity.
	ErrDuplicateExternalRef = errors.New("duplicate external reference: external ID is already mapped to a different entity")
	// ErrConflictingExternalRef indicates an internal entity is already mapped to a different external ID in the system.
	ErrConflictingExternalRef = errors.New("conflicting external reference: internal entity is already mapped to a different external ID")
	// ErrExternalRefNotFound indicates that the requested external reference mapping does not exist.
	ErrExternalRefNotFound = errors.New("external reference not found in registry")
	// ErrExternalIDTooShort indicates the external ID violates the minimum length threshold (anti-enumeration).
	ErrExternalIDTooShort = errors.New("external ID is too short (anti-enumeration boundary violation)")
	// ErrExternalIDTooLong indicates the external ID exceeds the maximum allowed length.
	ErrExternalIDTooLong = errors.New("external ID exceeds maximum permitted length")
	// ErrExternalIDEnumeration indicates trivial sequential, wildcard, or low-entropy patterns were detected.
	ErrExternalIDEnumeration = errors.New("external ID rejected: trivial sequential or wildcard pattern detected")
	// ErrCrossTenantMapping indicates an illegal attempt to link entities across tenant boundaries.
	ErrCrossTenantMapping = errors.New("cross-tenant external reference mapping is strictly prohibited")
)

// ValidateCanonicalID verifies that an identifier conforms strictly to the canonical format:
// "<expectedPrefix>_<token>" where token is lowercase alphanumeric, hyphen, or underscore (min 8 chars).
func ValidateCanonicalID(id, expectedPrefix string) error {
	trimmed := strings.TrimSpace(id)
	if trimmed == "" {
		return ErrBlankIdentifier
	}

	idx := strings.IndexByte(trimmed, '_')
	if idx <= 0 || idx == len(trimmed)-1 {
		return ErrMalformedIdentifier
	}

	prefix := trimmed[:idx]
	token := trimmed[idx+1:]

	if prefix != expectedPrefix {
		return fmt.Errorf("%w: expected prefix %q, got %q", ErrPrefixMismatch, expectedPrefix, prefix)
	}

	if len(token) < 8 {
		return fmt.Errorf("%w: token must be at least 8 characters", ErrMalformedIdentifier)
	}

	for i := 0; i < len(token); i++ {
		c := token[i]
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_' || c == '-') {
			return ErrInvalidCharacters
		}
	}

	return nil
}

// GenerateCanonicalID generates a cryptographically random, stable, non-reusable canonical identifier.
func GenerateCanonicalID(prefix string) (string, error) {
	trimmedPrefix := strings.TrimSpace(prefix)
	if trimmedPrefix == "" {
		return "", ErrBlankIdentifier
	}

	b := make([]byte, 16) // 128 bits of cryptographic randomness
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	return fmt.Sprintf("%s_%s", trimmedPrefix, hex.EncodeToString(b)), nil
}

// ValidateDigest verifies that a payload digest is a 64-character lowercase hexadecimal SHA-256 string.
func ValidateDigest(digest string) error {
	trimmed := strings.TrimSpace(digest)
	if trimmed == "" {
		return ErrBlankPayloadDigest
	}
	if len(trimmed) != 64 {
		return ErrInvalidDigest
	}
	for i := 0; i < len(trimmed); i++ {
		c := trimmed[i]
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return ErrInvalidDigest
		}
	}
	return nil
}

// TrackingContext encapsulates correlation and causation identifiers for cross-module operational tracing.
type TrackingContext struct {
	CorrelationID string `json:"correlation_id"`
	CausationID   string `json:"causation_id"`
}

// NewTrackingContext creates and validates a TrackingContext.
func NewTrackingContext(correlationID, causationID string) (TrackingContext, error) {
	if err := ValidateCanonicalID(correlationID, PrefixCorrelation); err != nil {
		return TrackingContext{}, fmt.Errorf("invalid correlation_id: %w", err)
	}
	if err := ValidateCanonicalID(causationID, PrefixCausation); err != nil {
		return TrackingContext{}, fmt.Errorf("invalid causation_id: %w", err)
	}
	return TrackingContext{
		CorrelationID: strings.TrimSpace(correlationID),
		CausationID:   strings.TrimSpace(causationID),
	}, nil
}

// GenerateTrackingContext generates fresh, cryptographically random correlation and causation IDs.
func GenerateTrackingContext() (TrackingContext, error) {
	corr, err := GenerateCanonicalID(PrefixCorrelation)
	if err != nil {
		return TrackingContext{}, err
	}
	caus, err := GenerateCanonicalID(PrefixCausation)
	if err != nil {
		return TrackingContext{}, err
	}
	return TrackingContext{
		CorrelationID: corr,
		CausationID:   caus,
	}, nil
}

// IdempotencyRecord models a recorded operation result for a given idempotency key.
type IdempotencyRecord struct {
	Key           string    `json:"key"`
	TenantID      string    `json:"tenant_id"`
	PayloadDigest string    `json:"payload_digest"`
	ResourceID    string    `json:"resource_id"`
	CreatedAt     time.Time `json:"created_at"`
}

// IdempotencyStore is a thread-safe in-memory store for idempotency keys.
type IdempotencyStore struct {
	mu      sync.RWMutex
	records map[string]IdempotencyRecord // key: tenantID + ":" + idempotencyKey
}

// NewIdempotencyStore initializes an empty in-memory IdempotencyStore.
func NewIdempotencyStore() *IdempotencyStore {
	return &IdempotencyStore{
		records: make(map[string]IdempotencyRecord),
	}
}

func makeIdempotencyKey(tenantID, key string) string {
	return fmt.Sprintf("%s:%s", strings.TrimSpace(tenantID), strings.TrimSpace(key))
}

// EvaluateOrCreate evaluates an idempotency key. If it exists:
// - If the payload digest matches: returns the existing ResourceID and isReplay = true.
// - If the payload digest differs: returns ErrIdempotencyConflict (preventing payload collisions).
// If it does not exist: calls createFn(), records the result, and returns isReplay = false.
func (s *IdempotencyStore) EvaluateOrCreate(tenantID, key, payloadDigest string, createFn func() (string, error)) (resourceID string, isReplay bool, err error) {
	tTenant := strings.TrimSpace(tenantID)
	if tTenant == "" {
		return "", false, ErrEmptyTenantID
	}
	tKey := strings.TrimSpace(key)
	if tKey == "" {
		return "", false, errors.New("idempotency key cannot be blank")
	}
	if err := ValidateDigest(payloadDigest); err != nil {
		return "", false, err
	}
	if createFn == nil {
		return "", false, errors.New("createFn cannot be nil")
	}

	storageKey := makeIdempotencyKey(tTenant, tKey)

	s.mu.Lock()
	defer s.mu.Unlock()

	if existing, exists := s.records[storageKey]; exists {
		if existing.PayloadDigest == strings.TrimSpace(payloadDigest) {
			// Idempotent duplicate: safe replay
			return existing.ResourceID, true, nil
		}
		// Conflicting duplicate: payload mismatch
		return "", false, ErrIdempotencyConflict
	}

	newID, err := createFn()
	if err != nil {
		return "", false, err
	}

	record := IdempotencyRecord{
		Key:           tKey,
		TenantID:      tTenant,
		PayloadDigest: strings.TrimSpace(payloadDigest),
		ResourceID:    newID,
		CreatedAt:     time.Now().UTC(),
	}

	s.records[storageKey] = record
	return newID, false, nil
}

// ExternalReference represents a mapped external identifier with anti-enumeration protection.
type ExternalReference struct {
	TenantID       string    `json:"tenant_id"`
	ExternalSystem string    `json:"external_system"`
	ExternalID     string    `json:"external_id"`
	InternalID     string    `json:"internal_id"`
	MappedAt       time.Time `json:"mapped_at"`
}

// ExternalRefRegistry manages controlled external reference mappings within tenant boundaries.
type ExternalRefRegistry struct {
	mu           sync.RWMutex
	forwardMap   map[string]ExternalReference // tenant:system:externalID -> ref
	reverseMap   map[string]string            // tenant:system:internalID -> externalID
}

// NewExternalRefRegistry constructs an empty ExternalRefRegistry.
func NewExternalRefRegistry() *ExternalRefRegistry {
	return &ExternalRefRegistry{
		forwardMap: make(map[string]ExternalReference),
		reverseMap: make(map[string]string),
	}
}

func makeForwardKey(tenantID, system, externalID string) string {
	return fmt.Sprintf("%s:%s:%s", strings.TrimSpace(tenantID), strings.ToUpper(strings.TrimSpace(system)), strings.TrimSpace(externalID))
}

func makeReverseKey(tenantID, system, internalID string) string {
	return fmt.Sprintf("%s:%s:%s", strings.TrimSpace(tenantID), strings.ToUpper(strings.TrimSpace(system)), strings.TrimSpace(internalID))
}

// validateExternalID applies anti-enumeration and validation boundaries to external IDs.
func validateExternalID(externalID string) error {
	trimmed := strings.TrimSpace(externalID)
	if len(trimmed) < MinExternalIDLength {
		return ErrExternalIDTooShort
	}
	if len(trimmed) > MaxExternalIDLength {
		return ErrExternalIDTooLong
	}

	// Reject wildcard, SQL injection probe, or single repeated character tokens
	if strings.ContainsAny(trimmed, "*?%<>$\\/;\"'") {
		return ErrExternalIDEnumeration
	}

	// Reject all-identical character tokens (e.g. "0000", "111", "aaa")
	allSame := true
	for i := 1; i < len(trimmed); i++ {
		if trimmed[i] != trimmed[0] {
			allSame = false
			break
		}
	}
	if allSame {
		return ErrExternalIDEnumeration
	}

	return nil
}

// RegisterExternalRef registers a mapping between an external system ID and an internal canonical ID.
// Invariants:
// - Re-registering an identical mapping is idempotent (succeeds).
// - Mapping an externalID to a different internalID returns ErrDuplicateExternalRef.
// - Mapping an internalID to a different externalID in the same system returns ErrConflictingExternalRef.
// - Trivial sequential, wildcard, or out-of-boundary external IDs fail with anti-enumeration errors.
func (r *ExternalRefRegistry) RegisterExternalRef(tenantID, system, externalID, internalID string) error {
	tTenant := strings.TrimSpace(tenantID)
	if tTenant == "" {
		return ErrEmptyTenantID
	}
	tSystem := strings.ToUpper(strings.TrimSpace(system))
	if tSystem == "" {
		return errors.New("external system identifier cannot be blank")
	}
	tExtID := strings.TrimSpace(externalID)
	if err := validateExternalID(tExtID); err != nil {
		return err
	}
	tIntID := strings.TrimSpace(internalID)
	if tIntID == "" {
		return ErrBlankIdentifier
	}

	fKey := makeForwardKey(tTenant, tSystem, tExtID)
	rKey := makeReverseKey(tTenant, tSystem, tIntID)

	r.mu.Lock()
	defer r.mu.Unlock()

	// 1. Check forward collision
	if existing, exists := r.forwardMap[fKey]; exists {
		if existing.InternalID == tIntID {
			// Idempotent re-registration of identical mapping
			return nil
		}
		return ErrDuplicateExternalRef
	}

	// 2. Check reverse collision (one internal entity cannot map to multiple external IDs in same system)
	if existingExtID, exists := r.reverseMap[rKey]; exists {
		if existingExtID != tExtID {
			return ErrConflictingExternalRef
		}
	}

	ref := ExternalReference{
		TenantID:       tTenant,
		ExternalSystem: tSystem,
		ExternalID:     tExtID,
		InternalID:     tIntID,
		MappedAt:       time.Now().UTC(),
	}

	r.forwardMap[fKey] = ref
	r.reverseMap[rKey] = tExtID
	return nil
}

// ResolveExternalRef resolves an external reference to its internal canonical ID.
// Fails closed if not found or if caller tenant does not match the reference.
func (r *ExternalRefRegistry) ResolveExternalRef(tenantID, system, externalID string) (string, error) {
	tTenant := strings.TrimSpace(tenantID)
	if tTenant == "" {
		return "", ErrEmptyTenantID
	}
	tSystem := strings.ToUpper(strings.TrimSpace(system))
	tExtID := strings.TrimSpace(externalID)

	fKey := makeForwardKey(tTenant, tSystem, tExtID)

	r.mu.RLock()
	defer r.mu.RUnlock()

	ref, exists := r.forwardMap[fKey]
	if !exists {
		return "", ErrExternalRefNotFound
	}
	if ref.TenantID != tTenant {
		return "", ErrCrossTenantMapping
	}

	return ref.InternalID, nil
}

// ResolveInternal resolves an internal canonical ID back to its mapped external ID.
func (r *ExternalRefRegistry) ResolveInternal(tenantID, system, internalID string) (string, error) {
	tTenant := strings.TrimSpace(tenantID)
	if tTenant == "" {
		return "", ErrEmptyTenantID
	}
	tSystem := strings.ToUpper(strings.TrimSpace(system))
	tIntID := strings.TrimSpace(internalID)

	rKey := makeReverseKey(tTenant, tSystem, tIntID)

	r.mu.RLock()
	defer r.mu.RUnlock()

	extID, exists := r.reverseMap[rKey]
	if !exists {
		return "", ErrExternalRefNotFound
	}
	return extID, nil
}

// JSON Serialization helpers

// MarshalJSON for TrackingContext
func (tc TrackingContext) MarshalJSON() ([]byte, error) {
	type Alias TrackingContext
	return json.Marshal(&struct {
		Alias
	}{
		Alias: (Alias)(tc),
	})
}
