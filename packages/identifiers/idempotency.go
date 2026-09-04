package identifiers

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

var (
	// ErrEmptyIdempotencyKey indicates that the provided idempotency key is empty or whitespace-only.
	ErrEmptyIdempotencyKey = errors.New("idempotency key cannot be empty")
	// ErrIdempotencyConflict indicates that an idempotency key was reused with a conflicting payload digest.
	ErrIdempotencyConflict = errors.New("idempotency key conflict: payload mismatch")
)

// IdempotencyStatus classifies the outcome of checking an idempotency key.
type IdempotencyStatus string

const (
	// StatusFirstUse indicates that the key is newly recorded.
	StatusFirstUse IdempotencyStatus = "FIRST_USE"
	// StatusReplay indicates that the key was previously registered with the identical payload digest.
	StatusReplay IdempotencyStatus = "REPLAY"
	// StatusConflict indicates that the key was previously registered with a different payload digest.
	StatusConflict IdempotencyStatus = "CONFLICT"
)

// IdempotencyRecord captures the immutable registration metadata for a key.
type IdempotencyRecord struct {
	Key          string
	PayloadHash  string
	RegisteredAt time.Time
}

// IdempotencyLedger provides a thread-safe, in-memory ledger for idempotency validation.
type IdempotencyLedger struct {
	mu      sync.RWMutex
	records map[string]IdempotencyRecord
}

// NewIdempotencyLedger constructs an initialized in-memory idempotency ledger.
func NewIdempotencyLedger() *IdempotencyLedger {
	return &IdempotencyLedger{
		records: make(map[string]IdempotencyRecord),
	}
}

// HashPayload computes the deterministic hex-encoded SHA-256 digest of arbitrary payload bytes.
func HashPayload(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

// CheckOrRecord checks a key against the ledger.
// If the key is unseen, it registers the payload digest and returns StatusFirstUse.
// If the key exists with the exact same payload digest, it returns StatusReplay.
// If the key exists with a different payload digest, it returns StatusConflict and ErrIdempotencyConflict.
func (l *IdempotencyLedger) CheckOrRecord(key string, payload []byte) (IdempotencyStatus, *IdempotencyRecord, error) {
	trimmedKey := strings.TrimSpace(key)
	if trimmedKey == "" {
		return "", nil, ErrEmptyIdempotencyKey
	}

	hash := HashPayload(payload)

	l.mu.Lock()
	defer l.mu.Unlock()

	if existing, found := l.records[trimmedKey]; found {
		if existing.PayloadHash == hash {
			return StatusReplay, &existing, nil
		}
		return StatusConflict, &existing, fmt.Errorf("%w: key %q previously recorded with hash %s, got %s",
			ErrIdempotencyConflict, trimmedKey, existing.PayloadHash, hash)
	}

	rec := IdempotencyRecord{
		Key:          trimmedKey,
		PayloadHash:  hash,
		RegisteredAt: time.Now().UTC(),
	}
	l.records[trimmedKey] = rec

	return StatusFirstUse, &rec, nil
}

// Get retrieves the recorded entry for a key if present.
func (l *IdempotencyLedger) Get(key string) (IdempotencyRecord, bool) {
	trimmedKey := strings.TrimSpace(key)
	if trimmedKey == "" {
		return IdempotencyRecord{}, false
	}
	l.mu.RLock()
	defer l.mu.RUnlock()
	rec, found := l.records[trimmedKey]
	return rec, found
}

// Count returns the total number of tracked idempotency keys in memory.
func (l *IdempotencyLedger) Count() int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return len(l.records)
}
