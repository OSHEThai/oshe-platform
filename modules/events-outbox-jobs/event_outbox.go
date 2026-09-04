package eventoutbox

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	// Canonical identifier prefixes
	PrefixEvent       = "evt"
	PrefixTenant      = "ten"
	PrefixCorrelation = "corr"
	PrefixCausation   = "caus"

	// Current canonical versions
	CurrentEnvelopeVersion = "1.0.0"
	CurrentSchemaVersion   = "1.0.0"
)

var (
	ErrEmptyID                    = errors.New("identifier cannot be empty")
	ErrMalformedID                = errors.New("identifier is malformed")
	ErrPrefixMismatch             = errors.New("identifier prefix mismatch")
	ErrBlankField                 = errors.New("required field cannot be blank")
	ErrInvalidDigest              = errors.New("payload digest must be a 64-character lowercase hexadecimal SHA-256 string")
	ErrUnsupportedEnvelopeVersion = errors.New("unsupported envelope version")
	ErrIncompatibleSchemaVersion  = errors.New("incompatible schema version; only current schema version is accepted")
	ErrDuplicateEventID           = errors.New("duplicate event identifier")
	ErrCrossTenantAssociation     = errors.New("cross-tenant association denied")
	ErrTxClosed                   = errors.New("transaction is already closed")
)

// ValidatePrefixedID ensures an identifier has the format "<expectedPrefix>_<payload>"
// where payload is non-empty and contains only valid lowercase alphanumeric ASCII characters.
func ValidatePrefixedID(raw, expectedPrefix string) error {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ErrEmptyID
	}

	idx := strings.IndexByte(trimmed, '_')
	if idx <= 0 || idx == len(trimmed)-1 {
		return ErrMalformedID
	}

	prefix := trimmed[:idx]
	payload := trimmed[idx+1:]

	if prefix != expectedPrefix {
		return fmt.Errorf("%w: expected %q, got %q", ErrPrefixMismatch, expectedPrefix, prefix)
	}

	for i := 0; i < len(payload); i++ {
		c := payload[i]
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_' || c == '-') {
			return ErrMalformedID
		}
	}
	return nil
}

// ValidatePayloadDigest verifies that the digest is a 64-character lowercase hexadecimal string.
func ValidatePayloadDigest(digest string) error {
	trimmed := strings.TrimSpace(digest)
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

// ValidateCompatibility enforces that the envelope and schema versions match current versions exactly.
func ValidateCompatibility(envelopeVersion, schemaVersion string) error {
	if strings.TrimSpace(envelopeVersion) != CurrentEnvelopeVersion {
		return fmt.Errorf("%w: expected %s, got %s", ErrUnsupportedEnvelopeVersion, CurrentEnvelopeVersion, envelopeVersion)
	}
	if strings.TrimSpace(schemaVersion) != CurrentSchemaVersion {
		return fmt.Errorf("%w: expected %s, got %s", ErrIncompatibleSchemaVersion, CurrentSchemaVersion, schemaVersion)
	}
	return nil
}

// EventEnvelope represents a provider-neutral, immutable event record.
type EventEnvelope struct {
	EventID         string    `json:"event_id"`
	TenantID        string    `json:"tenant_id"`
	Producer        string    `json:"producer"`
	EventType       string    `json:"event_type"`
	EnvelopeVersion string    `json:"envelope_version"`
	SchemaVersion   string    `json:"schema_version"`
	CorrelationID   string    `json:"correlation_id"`
	CausationID     string    `json:"causation_id"`
	PayloadDigest   string    `json:"payload_digest"`
	SequenceNumber  int64     `json:"sequence_number"`
	Timestamp       time.Time `json:"timestamp"`
}

// EventInput represents client-supplied event facts staged within a transaction.
type EventInput struct {
	EventID         string
	TenantID        string // Optional: if provided, must strictly match transaction tenant
	Producer        string
	EventType       string
	EnvelopeVersion string // If empty, defaults to CurrentEnvelopeVersion
	SchemaVersion   string
	CorrelationID   string
	CausationID     string
	PayloadDigest   string
	Timestamp       time.Time // If zero, defaults to time.Now().UTC()
}

// Outbox represents the central thread-safe transactional outbox.
type Outbox struct {
	mu         sync.RWMutex
	committed  []EventEnvelope
	eventIDs   map[string]bool
	seqCounter int64
}

// NewOutbox initializes an empty transactional outbox.
func NewOutbox() *Outbox {
	return &Outbox{
		committed: make([]EventEnvelope, 0),
		eventIDs:  make(map[string]bool),
	}
}

// BeginTx starts a new transactional staging scope bound to a specific tenant ID.
func (o *Outbox) BeginTx(tenantID string) (*OutboxTx, error) {
	if err := ValidatePrefixedID(tenantID, PrefixTenant); err != nil {
		return nil, fmt.Errorf("invalid tenant ID for transaction: %w", err)
	}

	return &OutboxTx{
		outbox:   o,
		tenantID: strings.TrimSpace(tenantID),
		staged:   make([]EventEnvelope, 0),
		stagedIDs: make(map[string]bool),
	}, nil
}

// CommittedEvents returns a snapshot of all committed events, sorted deterministically
// by sequence number.
func (o *Outbox) CommittedEvents() []EventEnvelope {
	o.mu.RLock()
	defer o.mu.RUnlock()

	result := make([]EventEnvelope, len(o.committed))
	copy(result, o.committed)

	sort.Slice(result, func(i, j int) bool {
		return result[i].SequenceNumber < result[j].SequenceNumber
	})

	return result
}

// CommittedEventsForTenant returns committed events scoped to a specific tenant,
// ordered deterministically by sequence number.
func (o *Outbox) CommittedEventsForTenant(tenantID string) ([]EventEnvelope, error) {
	if err := ValidatePrefixedID(tenantID, PrefixTenant); err != nil {
		return nil, err
	}

	o.mu.RLock()
	defer o.mu.RUnlock()

	var result []EventEnvelope
	for _, ev := range o.committed {
		if ev.TenantID == tenantID {
			result = append(result, ev)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].SequenceNumber < result[j].SequenceNumber
	})

	return result, nil
}

// OutboxTx represents an active transaction that stages events in memory.
// Staged events are completely invisible to outbox consumers until Commit is called.
type OutboxTx struct {
	mu        sync.Mutex
	outbox    *Outbox
	tenantID  string
	staged    []EventEnvelope
	stagedIDs map[string]bool
	closed    bool
}

// TenantID returns the transaction's bound tenant identifier.
func (tx *OutboxTx) TenantID() string {
	return tx.tenantID
}

// Stage validates and stages an event within the transaction.
// Fails closed if:
// - transaction is already committed or rolled back
// - tenantID conflicts with the transaction tenant
// - event ID is malformed or already staged/committed
// - required fields are blank
// - versions are incompatible
// - correlation/causation IDs are malformed
// - payload digest is invalid
func (tx *OutboxTx) Stage(input EventInput) error {
	tx.mu.Lock()
	defer tx.mu.Unlock()

	if tx.closed {
		return ErrTxClosed
	}

	// 1. Cross-tenant check
	if input.TenantID != "" && strings.TrimSpace(input.TenantID) != tx.tenantID {
		return ErrCrossTenantAssociation
	}

	// 2. Validate EventID
	if err := ValidatePrefixedID(input.EventID, PrefixEvent); err != nil {
		return err
	}

	eventID := strings.TrimSpace(input.EventID)

	// Check duplicate in currently staged events
	if tx.stagedIDs[eventID] {
		return ErrDuplicateEventID
	}

	// Check duplicate in previously committed events
	tx.outbox.mu.RLock()
	alreadyCommitted := tx.outbox.eventIDs[eventID]
	tx.outbox.mu.RUnlock()
	if alreadyCommitted {
		return ErrDuplicateEventID
	}

	// 3. Validate Producer and EventType
	producer := strings.TrimSpace(input.Producer)
	if producer == "" {
		return fmt.Errorf("%w: producer", ErrBlankField)
	}
	eventType := strings.TrimSpace(input.EventType)
	if eventType == "" {
		return fmt.Errorf("%w: event_type", ErrBlankField)
	}

	// 4. Envelope and Schema version validation
	envVer := strings.TrimSpace(input.EnvelopeVersion)
	if envVer == "" {
		envVer = CurrentEnvelopeVersion
	}
	if err := ValidateCompatibility(envVer, input.SchemaVersion); err != nil {
		return err
	}

	// 5. Correlation and Causation IDs
	if err := ValidatePrefixedID(input.CorrelationID, PrefixCorrelation); err != nil {
		return err
	}
	if err := ValidatePrefixedID(input.CausationID, PrefixCausation); err != nil {
		return err
	}

	// 6. Payload Digest
	if err := ValidatePayloadDigest(input.PayloadDigest); err != nil {
		return err
	}

	ts := input.Timestamp
	if ts.IsZero() {
		ts = time.Now().UTC()
	}

	envelope := EventEnvelope{
		EventID:         eventID,
		TenantID:        tx.tenantID,
		Producer:        producer,
		EventType:       eventType,
		EnvelopeVersion: envVer,
		SchemaVersion:   strings.TrimSpace(input.SchemaVersion),
		CorrelationID:   strings.TrimSpace(input.CorrelationID),
		CausationID:     strings.TrimSpace(input.CausationID),
		PayloadDigest:   strings.TrimSpace(input.PayloadDigest),
		Timestamp:       ts,
	}

	tx.staged = append(tx.staged, envelope)
	tx.stagedIDs[eventID] = true

	return nil
}

// Commit publishes all staged events to the central Outbox in atomic order,
// assigning sequential sequence numbers.
func (tx *OutboxTx) Commit() error {
	tx.mu.Lock()
	defer tx.mu.Unlock()

	if tx.closed {
		return ErrTxClosed
	}

	tx.outbox.mu.Lock()
	defer tx.outbox.mu.Unlock()

	// Final verification of duplicate IDs under outbox lock
	for _, env := range tx.staged {
		if tx.outbox.eventIDs[env.EventID] {
			return ErrDuplicateEventID
		}
	}

	// Publish staged events with sequential numbering
	for _, env := range tx.staged {
		tx.outbox.seqCounter++
		env.SequenceNumber = tx.outbox.seqCounter
		tx.outbox.committed = append(tx.outbox.committed, env)
		tx.outbox.eventIDs[env.EventID] = true
	}

	tx.closed = true
	tx.staged = nil
	tx.stagedIDs = nil

	return nil
}

// Rollback discards all staged events. No event from this transaction will ever
// be observable in the Outbox.
func (tx *OutboxTx) Rollback() error {
	tx.mu.Lock()
	defer tx.mu.Unlock()

	if tx.closed {
		return ErrTxClosed
	}

	tx.closed = true
	tx.staged = nil
	tx.stagedIDs = nil

	return nil
}
