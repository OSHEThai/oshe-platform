package recordsaudit

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

var (
	// ErrDuplicateRecord indicates a record already exists with the same identifier in the tenant scope.
	ErrDuplicateRecord = errors.New("record identifier already exists in tenant scope")
	// ErrDuplicateVersion indicates an attempt to overwrite an existing accepted version snapshot.
	ErrDuplicateVersion = errors.New("record version already exists; accepted versions cannot be overwritten")
	// ErrRecordNotFound indicates that the requested record does not exist.
	ErrRecordNotFound = errors.New("record not found")
	// ErrCrossTenantAccess indicates that caller tenant scope does not match the resource tenant.
	ErrCrossTenantAccess = errors.New("cross-tenant access denied: caller tenant does not match record tenant")
	// ErrInvalidCorrelationID indicates that a correlation identifier is missing or lacks the corr_ prefix.
	ErrInvalidCorrelationID = errors.New("invalid correlation identifier: must have prefix corr_")
	// ErrInvalidCausationID indicates that a causation identifier is missing or lacks the caus_ prefix.
	ErrInvalidCausationID = errors.New("invalid causation identifier: must have prefix caus_")
	// ErrBlankRecordType indicates that the record type identifier is blank.
	ErrBlankRecordType = errors.New("record type cannot be blank")
	// ErrBlankVersion indicates that the version identifier is blank.
	ErrBlankVersion = errors.New("version identifier cannot be blank")
	// ErrBlankActor indicates that the actor identifier is blank.
	ErrBlankActor = errors.New("actor identifier cannot be blank")
)

// AuditEventType defines canonical categories of audit entries.
type AuditEventType string

const (
	// AuditDeclared indicates the initial declaration of an authoritative record.
	AuditDeclared AuditEventType = "RECORD_DECLARED"
	// AuditVersioned indicates the declaration of a new immutable version snapshot.
	AuditVersioned AuditEventType = "RECORD_VERSIONED"
	// AuditStateChanged indicates a validated lifecycle state transition.
	AuditStateChanged AuditEventType = "STATE_CHANGED"
	// AuditAccessDenied indicates a rejected or unauthorized access attempt.
	AuditAccessDenied AuditEventType = "ACCESS_DENIED"
)

// AuditEntry is an immutable, append-only record capturing operational and security causality.
type AuditEntry struct {
	SequenceNumber int64                `json:"sequence_number"`
	EventType      AuditEventType       `json:"event_type"`
	TenantID       string               `json:"tenant_id"`
	RecordID       string               `json:"record_id"`
	ActorID        string               `json:"actor_id"`
	CorrelationID  string               `json:"correlation_id"`
	CausationID    string               `json:"causation_id"`
	PriorState     RecordLifecycleState `json:"prior_state,omitempty"`
	CurrentState   RecordLifecycleState `json:"current_state,omitempty"`
	Version        string               `json:"version,omitempty"`
	Details        string               `json:"details,omitempty"`
	Timestamp      time.Time            `json:"timestamp"`
}

// RecordSnapshot represents an immutable historical version snapshot of an accepted record.
type RecordSnapshot struct {
	RecordID      string               `json:"record_id"`
	TenantID      string               `json:"tenant_id"`
	Version       string               `json:"version"`
	PayloadDigest string               `json:"payload_digest"`
	DeclaredBy    string               `json:"declared_by"`
	DeclaredAt    time.Time            `json:"declared_at"`
	State         RecordLifecycleState `json:"state"`
}

// DeclaredRecord represents the current state of an authoritative record in tenant scope.
type DeclaredRecord struct {
	RecordID       string               `json:"record_id"`
	TenantID       string               `json:"tenant_id"`
	RecordType     string               `json:"record_type"`
	CurrentVersion string               `json:"current_version"`
	State          RecordLifecycleState `json:"state"`
	CreatedAt      time.Time            `json:"created_at"`
	UpdatedAt      time.Time            `json:"updated_at"`
}

// RecordStore provides an in-memory, thread-safe store for record declarations,
// version snapshots, and append-only audit journals with strict tenant isolation.
type RecordStore struct {
	mu         sync.RWMutex
	records    map[string]DeclaredRecord  // key: tenantID + ":" + recordID
	snapshots  map[string][]RecordSnapshot // key: tenantID + ":" + recordID
	auditLog   []AuditEntry
	seqCounter int64
}

// NewRecordStore initializes an empty in-memory record and audit store.
func NewRecordStore() *RecordStore {
	return &RecordStore{
		records:   make(map[string]DeclaredRecord),
		snapshots: make(map[string][]RecordSnapshot),
		auditLog:  make([]AuditEntry, 0),
	}
}

func makeRecordKey(tenantID, recordID string) string {
	return fmt.Sprintf("%s:%s", strings.TrimSpace(tenantID), strings.TrimSpace(recordID))
}

func validateTrackingIDs(correlationID, causationID string) error {
	tCorr := strings.TrimSpace(correlationID)
	if tCorr == "" || !strings.HasPrefix(tCorr, "corr_") {
		return ErrInvalidCorrelationID
	}
	tCaus := strings.TrimSpace(causationID)
	if tCaus == "" || !strings.HasPrefix(tCaus, "caus_") {
		return ErrInvalidCausationID
	}
	return nil
}

// DeclareRecord creates an authoritative record in the ACCEPTED state with an initial immutable snapshot.
func (s *RecordStore) DeclareRecord(tenantID, recordID, recordType, version, payloadDigest, actorID, corrID, causID string) (*DeclaredRecord, error) {
	tTenant := strings.TrimSpace(tenantID)
	if tTenant == "" {
		return nil, ErrBlankTenantID
	}
	tRecordID := strings.TrimSpace(recordID)
	if tRecordID == "" {
		return nil, ErrBlankID
	}
	tType := strings.TrimSpace(recordType)
	if tType == "" {
		return nil, ErrBlankRecordType
	}
	tVersion := strings.TrimSpace(version)
	if tVersion == "" {
		return nil, ErrBlankVersion
	}
	tActor := strings.TrimSpace(actorID)
	if tActor == "" {
		return nil, ErrBlankActor
	}
	if err := ValidateDigest(payloadDigest); err != nil {
		return nil, err
	}
	if err := validateTrackingIDs(corrID, causID); err != nil {
		return nil, err
	}

	key := makeRecordKey(tTenant, tRecordID)

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.records[key]; exists {
		return nil, ErrDuplicateRecord
	}

	now := time.Now().UTC()

	rec := DeclaredRecord{
		RecordID:       tRecordID,
		TenantID:       tTenant,
		RecordType:     tType,
		CurrentVersion: tVersion,
		State:          StateAccepted,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	snap := RecordSnapshot{
		RecordID:      tRecordID,
		TenantID:      tTenant,
		Version:       tVersion,
		PayloadDigest: strings.TrimSpace(payloadDigest),
		DeclaredBy:    tActor,
		DeclaredAt:    now,
		State:         StateAccepted,
	}

	s.seqCounter++
	audit := AuditEntry{
		SequenceNumber: s.seqCounter,
		EventType:      AuditDeclared,
		TenantID:       tTenant,
		RecordID:       tRecordID,
		ActorID:        tActor,
		CorrelationID:  strings.TrimSpace(corrID),
		CausationID:    strings.TrimSpace(causID),
		PriorState:     "",
		CurrentState:   StateAccepted,
		Version:        tVersion,
		Details:        fmt.Sprintf("Declared record type %s version %s", tType, tVersion),
		Timestamp:      now,
	}

	s.records[key] = rec
	s.snapshots[key] = []RecordSnapshot{snap}
	s.auditLog = append(s.auditLog, audit)

	return &rec, nil
}

// CreateNewVersion appends a new immutable version snapshot to an existing record.
// Prevents overwriting or silently replacing existing accepted versions.
func (s *RecordStore) CreateNewVersion(tenantID, recordID, newVersion, payloadDigest, actorID, corrID, causID string) (*RecordSnapshot, error) {
	tTenant := strings.TrimSpace(tenantID)
	if tTenant == "" {
		return nil, ErrBlankTenantID
	}
	tRecordID := strings.TrimSpace(recordID)
	if tRecordID == "" {
		return nil, ErrBlankID
	}
	tVersion := strings.TrimSpace(newVersion)
	if tVersion == "" {
		return nil, ErrBlankVersion
	}
	tActor := strings.TrimSpace(actorID)
	if tActor == "" {
		return nil, ErrBlankActor
	}
	if err := ValidateDigest(payloadDigest); err != nil {
		return nil, err
	}
	if err := validateTrackingIDs(corrID, causID); err != nil {
		return nil, err
	}

	key := makeRecordKey(tTenant, tRecordID)

	s.mu.Lock()
	defer s.mu.Unlock()

	rec, exists := s.records[key]
	if !exists {
		return nil, ErrRecordNotFound
	}
	if rec.State == StateArchived {
		return nil, ErrRecordArchived
	}

	// Verify newVersion does not already exist
	history := s.snapshots[key]
	for _, snap := range history {
		if snap.Version == tVersion {
			return nil, ErrDuplicateVersion
		}
	}

	now := time.Now().UTC()

	snap := RecordSnapshot{
		RecordID:      tRecordID,
		TenantID:      tTenant,
		Version:       tVersion,
		PayloadDigest: strings.TrimSpace(payloadDigest),
		DeclaredBy:    tActor,
		DeclaredAt:    now,
		State:         rec.State,
	}

	rec.CurrentVersion = tVersion
	rec.UpdatedAt = now

	s.seqCounter++
	audit := AuditEntry{
		SequenceNumber: s.seqCounter,
		EventType:      AuditVersioned,
		TenantID:       tTenant,
		RecordID:       tRecordID,
		ActorID:        tActor,
		CorrelationID:  strings.TrimSpace(corrID),
		CausationID:    strings.TrimSpace(causID),
		PriorState:     rec.State,
		CurrentState:   rec.State,
		Version:        tVersion,
		Details:        fmt.Sprintf("Created version snapshot %s", tVersion),
		Timestamp:      now,
	}

	s.records[key] = rec
	s.snapshots[key] = append(s.snapshots[key], snap)
	s.auditLog = append(s.auditLog, audit)

	return &snap, nil
}

// TransitionRecordState changes the lifecycle state of a record and records an append-only audit event.
func (s *RecordStore) TransitionRecordState(tenantID, recordID string, targetState RecordLifecycleState, actorID, corrID, causID string) error {
	tTenant := strings.TrimSpace(tenantID)
	if tTenant == "" {
		return ErrBlankTenantID
	}
	tRecordID := strings.TrimSpace(recordID)
	if tRecordID == "" {
		return ErrBlankID
	}
	tActor := strings.TrimSpace(actorID)
	if tActor == "" {
		return ErrBlankActor
	}
	if err := validateTrackingIDs(corrID, causID); err != nil {
		return err
	}

	key := makeRecordKey(tTenant, tRecordID)

	s.mu.Lock()
	defer s.mu.Unlock()

	rec, exists := s.records[key]
	if !exists {
		return ErrRecordNotFound
	}
	if rec.State == targetState {
		return nil // Idempotent
	}
	if rec.State == StateArchived {
		return ErrRecordArchived
	}

	// Validate permitted transitions
	priorState := rec.State
	switch priorState {
	case StateDraft:
		if targetState != StateAccepted && targetState != StateArchived {
			return ErrInvalidLifecycleTransition
		}
	case StateAccepted:
		if targetState != StateArchived {
			return ErrInvalidLifecycleTransition
		}
	default:
		return ErrInvalidLifecycleTransition
	}

	now := time.Now().UTC()
	rec.State = targetState
	rec.UpdatedAt = now

	s.seqCounter++
	audit := AuditEntry{
		SequenceNumber: s.seqCounter,
		EventType:      AuditStateChanged,
		TenantID:       tTenant,
		RecordID:       tRecordID,
		ActorID:        tActor,
		CorrelationID:  strings.TrimSpace(corrID),
		CausationID:    strings.TrimSpace(causID),
		PriorState:     priorState,
		CurrentState:   targetState,
		Version:        rec.CurrentVersion,
		Details:        fmt.Sprintf("Transitioned state from %s to %s", priorState, targetState),
		Timestamp:      now,
	}

	s.records[key] = rec
	s.auditLog = append(s.auditLog, audit)

	return nil
}

// RecordAccessDenied appends an immutable ACCESS_DENIED audit entry for security monitoring.
func (s *RecordStore) RecordAccessDenied(tenantID, recordID, actorID, corrID, causID, reason string) error {
	tTenant := strings.TrimSpace(tenantID)
	if tTenant == "" {
		return ErrBlankTenantID
	}
	tRecordID := strings.TrimSpace(recordID)
	if tRecordID == "" {
		return ErrBlankID
	}
	tActor := strings.TrimSpace(actorID)
	if tActor == "" {
		return ErrBlankActor
	}
	if err := validateTrackingIDs(corrID, causID); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	s.seqCounter++

	audit := AuditEntry{
		SequenceNumber: s.seqCounter,
		EventType:      AuditAccessDenied,
		TenantID:       tTenant,
		RecordID:       tRecordID,
		ActorID:        tActor,
		CorrelationID:  strings.TrimSpace(corrID),
		CausationID:    strings.TrimSpace(causID),
		Details:        strings.TrimSpace(reason),
		Timestamp:      now,
	}

	s.auditLog = append(s.auditLog, audit)
	return nil
}

// GetAuditTrail returns an immutable, tenant-scoped copy of audit entries for a record.
// Fails closed if caller tenant does not match the record's tenant scope.
func (s *RecordStore) GetAuditTrail(callerTenantID, recordID string) ([]AuditEntry, error) {
	tCaller := strings.TrimSpace(callerTenantID)
	if tCaller == "" {
		return nil, ErrBlankTenantID
	}
	tRecordID := strings.TrimSpace(recordID)
	if tRecordID == "" {
		return nil, ErrBlankID
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	key := makeRecordKey(tCaller, tRecordID)
	if _, exists := s.records[key]; !exists {
		// Check if record exists under a different tenant to provide strict cross-tenant denial
		for rKey, rec := range s.records {
			if rec.RecordID == tRecordID && rec.TenantID != tCaller {
				_ = rKey
				return nil, ErrCrossTenantAccess
			}
		}
		return nil, ErrRecordNotFound
	}

	var results []AuditEntry
	for _, entry := range s.auditLog {
		if entry.TenantID == tCaller && entry.RecordID == tRecordID {
			results = append(results, entry)
		}
	}

	return results, nil
}

// GetSnapshots returns all historical snapshots for an accepted record within tenant scope.
func (s *RecordStore) GetSnapshots(callerTenantID, recordID string) ([]RecordSnapshot, error) {
	tCaller := strings.TrimSpace(callerTenantID)
	if tCaller == "" {
		return nil, ErrBlankTenantID
	}
	tRecordID := strings.TrimSpace(recordID)
	if tRecordID == "" {
		return nil, ErrBlankID
	}

	key := makeRecordKey(tCaller, tRecordID)

	s.mu.RLock()
	defer s.mu.RUnlock()

	rec, exists := s.records[key]
	if !exists {
		for _, r := range s.records {
			if r.RecordID == tRecordID && r.TenantID != tCaller {
				return nil, ErrCrossTenantAccess
			}
		}
		return nil, ErrRecordNotFound
	}

	history := s.snapshots[key]
	out := make([]RecordSnapshot, len(history))
	copy(out, history)
	_ = rec
	return out, nil
}

// GetRecord returns an immutable copy of a declared record within caller tenant scope.
func (s *RecordStore) GetRecord(callerTenantID, recordID string) (DeclaredRecord, error) {
	tCaller := strings.TrimSpace(callerTenantID)
	if tCaller == "" {
		return DeclaredRecord{}, ErrBlankTenantID
	}
	tRecordID := strings.TrimSpace(recordID)
	if tRecordID == "" {
		return DeclaredRecord{}, ErrBlankID
	}

	key := makeRecordKey(tCaller, tRecordID)

	s.mu.RLock()
	defer s.mu.RUnlock()

	rec, exists := s.records[key]
	if !exists {
		for _, r := range s.records {
			if r.RecordID == tRecordID && r.TenantID != tCaller {
				return DeclaredRecord{}, ErrCrossTenantAccess
			}
		}
		return DeclaredRecord{}, ErrRecordNotFound
	}

	return rec, nil
}
