package evidence

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

var (
	// ErrDerivedOriginalConfusion indicates a derived artifact attempted to masquerade as or overwrite an original.
	ErrDerivedOriginalConfusion = errors.New("derived artifact cannot replace, overwrite, or masquerade as original evidence")
	// ErrOriginalImmutable indicates an attempt to modify an accepted original evidence record.
	ErrOriginalImmutable = errors.New("accepted original evidence is immutable and cannot be modified or overwritten")
	// ErrMissingParentEvidence indicates that a derived object references a non-existent parent original.
	ErrMissingParentEvidence = errors.New("derived object must specify a valid existing original evidence parent")
	// ErrDuplicateEvidenceConflict indicates an attempt to register an evidence ID with conflicting digest.
	ErrDuplicateEvidenceConflict = errors.New("evidence ID already exists with conflicting content digest")
	// ErrTamperDetected indicates payload digest does not match the committed cryptographic hash.
	ErrTamperDetected = errors.New("evidence integrity verification failed: payload digest does not match committed hash")
	// ErrExportTampered indicates export manifest verification failed due to item digest mismatch or root corruption.
	ErrExportTampered = errors.New("export manifest verification failed: item digest or manifest root mismatch")
	// ErrTransferInterrupted indicates transfer was interrupted before payload completion.
	ErrTransferInterrupted = errors.New("evidence transfer was interrupted before completion")
	// ErrInvalidDerivationType indicates an unsupported or empty derivation type.
	ErrInvalidDerivationType = errors.New("invalid or unsupported derivation type")
	// ErrEmptyParentID indicates parent_evidence_id was blank for a derived object.
	ErrEmptyParentID = errors.New("parent_evidence_id cannot be empty for derived object")
	// ErrNestedDerivationProhibited indicates a derived object attempted to derive from another derived object.
	ErrNestedDerivationProhibited = errors.New("nested derivation prohibited: parent evidence must be an original")
	// ErrRecordNotCommitted indicates an operation requires a committed original but record is uncommitted.
	ErrRecordNotCommitted = errors.New("evidence record must be committed before derivation or export")
	// ErrBlankExportID indicates that the export identifier is empty.
	ErrBlankExportID = errors.New("export_id cannot be blank")
	// ErrEmptyExportItems indicates that the export manifest has no items.
	ErrEmptyExportItems = errors.New("export manifest must contain at least one evidence item")
)

// EvidenceObjectType distinguishes authoritative originals from downstream derived artifacts.
type EvidenceObjectType string

const (
	ObjectTypeOriginal EvidenceObjectType = "ORIGINAL"
	ObjectTypeDerived  EvidenceObjectType = "DERIVED"
)

// DerivationType defines permitted classifications for derived artifacts.
type DerivationType string

const (
	DerivationNone                DerivationType = "NONE"
	DerivationThumbnailPreview    DerivationType = "THUMBNAIL_PREVIEW"
	DerivationCompressedRendition DerivationType = "COMPRESSED_RENDITION"
	DerivationWatermarkedExport   DerivationType = "WATERMARKED_EXPORT"
	DerivationRedactedView        DerivationType = "REDACTED_VIEW"
)

// CustodyEventType describes deterministic lifecycle events in the chain of custody.
type CustodyEventType string

const (
	EventCaptureLocal        CustodyEventType = "CAPTURE_LOCAL"
	EventQueueLocal          CustodyEventType = "QUEUE_LOCAL"
	EventTransferStart       CustodyEventType = "TRANSFER_START"
	EventTransferInterrupted CustodyEventType = "TRANSFER_INTERRUPTED"
	EventTransferResumed     CustodyEventType = "TRANSFER_RESUMED"
	EventIntegrityVerified   CustodyEventType = "INTEGRITY_VERIFIED"
	EventOriginalCommitted   CustodyEventType = "ORIGINAL_COMMITTED"
	EventDerivedGenerated    CustodyEventType = "DERIVED_GENERATED"
	EventPreviewRendered     CustodyEventType = "PREVIEW_RENDERED"
	EventExportPackaged      CustodyEventType = "EXPORT_PACKAGED"
	EventTamperDetected      CustodyEventType = "TAMPER_DETECTED"
)

// EvidenceRecord represents an authoritative original or derived evidence object.
type EvidenceRecord struct {
	EvidenceID       string               `json:"evidence_id"`
	TenantID         string               `json:"tenant_id"`
	ObjectType       EvidenceObjectType   `json:"object_type"`
	ParentEvidenceID string               `json:"parent_evidence_id,omitempty"`
	DerivationType   DerivationType       `json:"derivation_type,omitempty"`
	AssociationType  string               `json:"association_type"`
	AssociationID    string               `json:"association_id"`
	OriginalName     string               `json:"original_name"`
	MediaType        string               `json:"media_type"`
	SizeBytes        int64                `json:"size_bytes"`
	SHA256Digest     string               `json:"sha256_digest"`
	State            UploadLifecycleState `json:"state"`
	Committed        bool                 `json:"committed"`
	CreatedAt        time.Time            `json:"created_at"`
	CreatedBy        string               `json:"created_by"`
}

// CustodyEvent records an immutable audit log entry for chain of custody tracking.
type CustodyEvent struct {
	EventID             string           `json:"event_id"`
	TenantID            string           `json:"tenant_id"`
	EvidenceID          string           `json:"evidence_id"`
	EventType           CustodyEventType `json:"event_type"`
	ActorID             string           `json:"actor_id"`
	Timestamp           time.Time        `json:"timestamp"`
	PayloadDigest       string           `json:"payload_digest"`
	Details             string           `json:"details"`
	PreviousEventDigest string           `json:"previous_event_digest"`
	EventDigest         string           `json:"event_digest"`
}

// ExportManifestItem represents a single verified file entry inside an export package.
type ExportManifestItem struct {
	EvidenceID       string             `json:"evidence_id"`
	ObjectType       EvidenceObjectType `json:"object_type"`
	ParentEvidenceID string             `json:"parent_evidence_id,omitempty"`
	DerivationType   DerivationType     `json:"derivation_type,omitempty"`
	MediaType        string             `json:"media_type"`
	SizeBytes        int64              `json:"size_bytes"`
	SHA256Digest     string             `json:"sha256_digest"`
}

// ExportManifest contains the complete manifest and cryptographic root digest for an evidence export.
type ExportManifest struct {
	ExportID       string               `json:"export_id"`
	TenantID       string               `json:"tenant_id"`
	ExporterID     string               `json:"exporter_id"`
	CreatedAt      time.Time            `json:"created_at"`
	Items          []ExportManifestItem `json:"items"`
	RootDigest     string               `json:"root_digest"`
	TotalSizeBytes int64                `json:"total_size_bytes"`
}

// EvidenceIntegrityManager provides thread-safe lifecycle management, verification,
// duplicate handling, tamper detection, and export manifest generation for evidence objects.
type EvidenceIntegrityManager struct {
	mu           sync.RWMutex
	records      map[string]map[string]*EvidenceRecord // tenantID -> evidenceID -> record
	custodyChain map[string]map[string][]CustodyEvent  // tenantID -> evidenceID -> events
	storage      ScopedStorageAdapter
}

// NewEvidenceIntegrityManager initializes a new EvidenceIntegrityManager.
func NewEvidenceIntegrityManager(storage ScopedStorageAdapter) *EvidenceIntegrityManager {
	return &EvidenceIntegrityManager{
		records:      make(map[string]map[string]*EvidenceRecord),
		custodyChain: make(map[string]map[string][]CustodyEvent),
		storage:      storage,
	}
}

// ComputeSHA256Digest returns the 64-character lowercase hex SHA-256 digest of payload.
func ComputeSHA256Digest(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// computeEventDigest calculates the tamper-evident hash for a custody event.
func computeEventDigest(prevDigest, eventID, tenantID, evidenceID, eventType, actorID, digest, details string, ts time.Time) string {
	h := sha256.New()
	entry := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s|%s|%d",
		prevDigest, eventID, tenantID, evidenceID, eventType, actorID, digest, details, ts.UnixNano())
	h.Write([]byte(entry))
	return hex.EncodeToString(h.Sum(nil))
}

// recordEventLocked appends a new custody event to the chain under active lock.
func (m *EvidenceIntegrityManager) recordEventLocked(tenantID, evidenceID string, eventType CustodyEventType, actorID, digest, details string) CustodyEvent {
	if _, ok := m.custodyChain[tenantID]; !ok {
		m.custodyChain[tenantID] = make(map[string][]CustodyEvent)
	}
	events := m.custodyChain[tenantID][evidenceID]

	prevDigest := "0000000000000000000000000000000000000000000000000000000000000000"
	if len(events) > 0 {
		prevDigest = events[len(events)-1].EventDigest
	}

	now := time.Now().UTC()
	eventID := fmt.Sprintf("evt_%s_%d", evidenceID, len(events)+1)
	eventHash := computeEventDigest(prevDigest, eventID, tenantID, evidenceID, string(eventType), actorID, digest, details, now)

	event := CustodyEvent{
		EventID:             eventID,
		TenantID:            tenantID,
		EvidenceID:          evidenceID,
		EventType:           eventType,
		ActorID:             actorID,
		Timestamp:           now,
		PayloadDigest:       digest,
		Details:             details,
		PreviousEventDigest: prevDigest,
		EventDigest:         eventHash,
	}

	m.custodyChain[tenantID][evidenceID] = append(events, event)
	return event
}

// RegisterOriginal initializes an authoritative original evidence record in INITIALIZED state.
func (m *EvidenceIntegrityManager) RegisterOriginal(
	tenantID, evidenceID, filename, mediaType string,
	sizeBytes int64, expectedDigest, associationType, associationID, createdBy string,
) (*EvidenceRecord, error) {
	tTenant := strings.TrimSpace(tenantID)
	if tTenant == "" {
		return nil, ErrEmptyTenantID
	}
	tEvidence := strings.TrimSpace(evidenceID)
	if tEvidence == "" {
		return nil, ErrEmptyFileID
	}
	if err := ValidateFilename(filename); err != nil {
		return nil, err
	}
	if !SupportedMediaTypes[mediaType] {
		return nil, ErrInvalidMediaType
	}
	if sizeBytes <= 0 || sizeBytes > MaxFileSizeBytes {
		return nil, ErrInvalidSize
	}
	if len(expectedDigest) != 64 {
		return nil, ErrInvalidDigest
	}
	if strings.TrimSpace(associationType) == "" || strings.TrimSpace(associationID) == "" {
		return nil, errors.New("association_type and association_id are required")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if tenantMap, exists := m.records[tTenant]; exists {
		if rec, exists := tenantMap[tEvidence]; exists {
			if rec.Committed {
				return nil, ErrOriginalImmutable
			}
			return nil, ErrDuplicateEvidenceConflict
		}
	} else {
		m.records[tTenant] = make(map[string]*EvidenceRecord)
	}

	rec := &EvidenceRecord{
		EvidenceID:      tEvidence,
		TenantID:        tTenant,
		ObjectType:      ObjectTypeOriginal,
		DerivationType:  DerivationNone,
		AssociationType: associationType,
		AssociationID:   associationID,
		OriginalName:    filename,
		MediaType:       mediaType,
		SizeBytes:       sizeBytes,
		SHA256Digest:    expectedDigest,
		State:           StateInitialized,
		Committed:       false,
		CreatedAt:       time.Now().UTC(),
		CreatedBy:       createdBy,
	}

	m.records[tTenant][tEvidence] = rec
	m.recordEventLocked(tTenant, tEvidence, EventCaptureLocal, createdBy, expectedDigest, "original evidence registered")
	return rec, nil
}

// CommitOriginal verifies payload integrity, persists to storage adapter, and marks original immutable.
func (m *EvidenceIntegrityManager) CommitOriginal(tenantID, evidenceID string, payload []byte, actorID string) error {
	tTenant := strings.TrimSpace(tenantID)
	tEvidence := strings.TrimSpace(evidenceID)

	m.mu.Lock()
	defer m.mu.Unlock()

	tenantMap, exists := m.records[tTenant]
	if !exists {
		return ErrObjectNotFound
	}
	rec, exists := tenantMap[tEvidence]
	if !exists {
		return ErrObjectNotFound
	}

	if rec.ObjectType != ObjectTypeOriginal {
		return ErrDerivedOriginalConfusion
	}
	if rec.Committed {
		return ErrOriginalImmutable
	}

	// Verify digest
	actualDigest := ComputeSHA256Digest(payload)
	if actualDigest != rec.SHA256Digest {
		m.recordEventLocked(tTenant, tEvidence, EventTamperDetected, actorID, actualDigest,
			fmt.Sprintf("integrity verification failed: expected %s, got %s", rec.SHA256Digest, actualDigest))
		return ErrTamperDetected
	}

	// Verify size
	if int64(len(payload)) != rec.SizeBytes {
		return ErrInvalidSize
	}

	m.recordEventLocked(tTenant, tEvidence, EventIntegrityVerified, actorID, actualDigest, "payload SHA-256 verified successfully")

	rec.State = StateCompleted
	rec.Committed = true
	m.recordEventLocked(tTenant, tEvidence, EventOriginalCommitted, actorID, actualDigest, "original committed as authoritative and immutable")
	return nil
}

// RegisterDerived registers a downstream derived artifact tied strictly to an accepted original parent.
func (m *EvidenceIntegrityManager) RegisterDerived(
	tenantID, derivedID, parentEvidenceID string,
	derivType DerivationType, filename, mediaType string,
	sizeBytes int64, payloadDigest, createdBy string,
) (*EvidenceRecord, error) {
	tTenant := strings.TrimSpace(tenantID)
	if tTenant == "" {
		return nil, ErrEmptyTenantID
	}
	tDerived := strings.TrimSpace(derivedID)
	if tDerived == "" {
		return nil, ErrEmptyFileID
	}
	tParent := strings.TrimSpace(parentEvidenceID)
	if tParent == "" {
		return nil, ErrEmptyParentID
	}
	if derivType == DerivationNone || derivType == "" {
		return nil, ErrInvalidDerivationType
	}
	if err := ValidateFilename(filename); err != nil {
		return nil, err
	}
	if !SupportedMediaTypes[mediaType] {
		return nil, ErrInvalidMediaType
	}
	if sizeBytes <= 0 || sizeBytes > MaxFileSizeBytes {
		return nil, ErrInvalidSize
	}
	if len(payloadDigest) != 64 {
		return nil, ErrInvalidDigest
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	tenantMap, exists := m.records[tTenant]
	if !exists {
		return nil, ErrMissingParentEvidence
	}
	parentRec, exists := tenantMap[tParent]
	if !exists {
		return nil, ErrMissingParentEvidence
	}

	// Derived objects cannot derive from another derived object
	if parentRec.ObjectType != ObjectTypeOriginal {
		return nil, ErrNestedDerivationProhibited
	}
	// Parent original must be committed
	if !parentRec.Committed {
		return nil, ErrRecordNotCommitted
	}

	// Derived ID cannot collide with existing ID
	if _, exists := tenantMap[tDerived]; exists {
		return nil, ErrDuplicateEvidenceConflict
	}

	derivedRec := &EvidenceRecord{
		EvidenceID:       tDerived,
		TenantID:         tTenant,
		ObjectType:       ObjectTypeDerived,
		ParentEvidenceID: tParent,
		DerivationType:   derivType,
		AssociationType:  parentRec.AssociationType,
		AssociationID:    parentRec.AssociationID,
		OriginalName:     filename,
		MediaType:        mediaType,
		SizeBytes:        sizeBytes,
		SHA256Digest:     payloadDigest,
		State:            StateInitialized,
		Committed:        false,
		CreatedAt:        time.Now().UTC(),
		CreatedBy:        createdBy,
	}

	tenantMap[tDerived] = derivedRec
	m.recordEventLocked(tTenant, tDerived, EventDerivedGenerated, createdBy, payloadDigest,
		fmt.Sprintf("derived artifact %s created from parent original %s", derivType, tParent))
	return derivedRec, nil
}

// CommitDerived verifies derived artifact digest and commits it as completed.
func (m *EvidenceIntegrityManager) CommitDerived(tenantID, derivedID string, payload []byte, actorID string) error {
	tTenant := strings.TrimSpace(tenantID)
	tDerived := strings.TrimSpace(derivedID)

	m.mu.Lock()
	defer m.mu.Unlock()

	tenantMap, exists := m.records[tTenant]
	if !exists {
		return ErrObjectNotFound
	}
	rec, exists := tenantMap[tDerived]
	if !exists {
		return ErrObjectNotFound
	}

	if rec.ObjectType != ObjectTypeDerived {
		return ErrDerivedOriginalConfusion
	}
	if rec.Committed {
		return ErrCompletedStateImmutable
	}

	actualDigest := ComputeSHA256Digest(payload)
	if actualDigest != rec.SHA256Digest {
		m.recordEventLocked(tTenant, tDerived, EventTamperDetected, actorID, actualDigest, "derived payload digest mismatch")
		return ErrTamperDetected
	}

	rec.State = StateCompleted
	rec.Committed = true
	m.recordEventLocked(tTenant, tDerived, EventIntegrityVerified, actorID, actualDigest, "derived artifact payload verified and committed")
	return nil
}

// RecordTransferInterrupted records an in-flight network/upload interruption and updates state.
func (m *EvidenceIntegrityManager) RecordTransferInterrupted(tenantID, evidenceID string, bytesTransferred, totalBytes int64, reason, actorID string) error {
	tTenant := strings.TrimSpace(tenantID)
	tEvidence := strings.TrimSpace(evidenceID)

	m.mu.Lock()
	defer m.mu.Unlock()

	tenantMap, exists := m.records[tTenant]
	if !exists {
		return ErrObjectNotFound
	}
	rec, exists := tenantMap[tEvidence]
	if !exists {
		return ErrObjectNotFound
	}

	if rec.Committed {
		return ErrOriginalImmutable
	}

	rec.State = StateFailed
	m.recordEventLocked(tTenant, tEvidence, EventTransferInterrupted, actorID, rec.SHA256Digest,
		fmt.Sprintf("transfer interrupted at %d/%d bytes: %s", bytesTransferred, totalBytes, reason))
	return nil
}

// RecordTransferResumed records resumption of an interrupted upload and resets state to TRANSFERRING.
func (m *EvidenceIntegrityManager) RecordTransferResumed(tenantID, evidenceID, actorID string) error {
	tTenant := strings.TrimSpace(tenantID)
	tEvidence := strings.TrimSpace(evidenceID)

	m.mu.Lock()
	defer m.mu.Unlock()

	tenantMap, exists := m.records[tTenant]
	if !exists {
		return ErrObjectNotFound
	}
	rec, exists := tenantMap[tEvidence]
	if !exists {
		return ErrObjectNotFound
	}

	if rec.Committed {
		return ErrOriginalImmutable
	}

	rec.State = StateTransferring
	m.recordEventLocked(tTenant, tEvidence, EventTransferResumed, actorID, rec.SHA256Digest, "transfer resumed")
	return nil
}

// HandleDuplicateUpload inspects an incoming upload for duplicate submission.
// Returns (true, nil) if identical payload is already committed (idempotent duplicate accepted).
// Returns (false, ErrDuplicateEvidenceConflict) if evidence ID exists with a conflicting digest.
func (m *EvidenceIntegrityManager) HandleDuplicateUpload(tenantID, evidenceID, expectedDigest string, payload []byte, actorID string) (bool, error) {
	tTenant := strings.TrimSpace(tenantID)
	tEvidence := strings.TrimSpace(evidenceID)

	m.mu.Lock()
	defer m.mu.Unlock()

	tenantMap, exists := m.records[tTenant]
	if !exists {
		return false, nil
	}
	rec, exists := tenantMap[tEvidence]
	if !exists {
		return false, nil
	}

	actualDigest := ComputeSHA256Digest(payload)
	if rec.Committed {
		if rec.SHA256Digest == actualDigest && rec.SHA256Digest == expectedDigest {
			m.recordEventLocked(tTenant, tEvidence, EventIntegrityVerified, actorID, actualDigest,
				"idempotent duplicate upload recognized; original preserved unchanged")
			return true, nil
		}
		return false, ErrDuplicateEvidenceConflict
	}

	return false, nil
}

// VerifyTamper checks a candidate byte stream against the committed record digest.
// Fails closed with ErrTamperDetected and appends an audit event if mismatch is detected.
func (m *EvidenceIntegrityManager) VerifyTamper(tenantID, evidenceID string, candidatePayload []byte, actorID string) (bool, error) {
	tTenant := strings.TrimSpace(tenantID)
	tEvidence := strings.TrimSpace(evidenceID)

	m.mu.Lock()
	defer m.mu.Unlock()

	tenantMap, exists := m.records[tTenant]
	if !exists {
		return false, ErrObjectNotFound
	}
	rec, exists := tenantMap[tEvidence]
	if !exists {
		return false, ErrObjectNotFound
	}

	actualDigest := ComputeSHA256Digest(candidatePayload)
	if actualDigest != rec.SHA256Digest {
		m.recordEventLocked(tTenant, tEvidence, EventTamperDetected, actorID, actualDigest,
			fmt.Sprintf("tamper detected: candidate digest %s != committed %s", actualDigest, rec.SHA256Digest))
		return false, ErrTamperDetected
	}

	return true, nil
}

// GetRecord retrieves an evidence record within tenant scope.
func (m *EvidenceIntegrityManager) GetRecord(tenantID, evidenceID string) (*EvidenceRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tenantMap, exists := m.records[tenantID]
	if !exists {
		return nil, ErrObjectNotFound
	}
	rec, exists := tenantMap[evidenceID]
	if !exists {
		return nil, ErrObjectNotFound
	}

	// Return defensive copy
	copyRec := *rec
	return &copyRec, nil
}

// GetCustodyChain retrieves the complete chain of custody for an evidence item.
func (m *EvidenceIntegrityManager) GetCustodyChain(tenantID, evidenceID string) ([]CustodyEvent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tenantMap, exists := m.custodyChain[tenantID]
	if !exists {
		return nil, ErrObjectNotFound
	}
	events, exists := tenantMap[evidenceID]
	if !exists {
		return nil, ErrObjectNotFound
	}

	result := make([]CustodyEvent, len(events))
	copy(result, events)
	return result, nil
}

// computeRootDigest computes the deterministic root SHA-256 digest across sorted manifest items.
func computeRootDigest(items []ExportManifestItem) string {
	sortedItems := make([]ExportManifestItem, len(items))
	copy(sortedItems, items)
	sort.Slice(sortedItems, func(i, j int) bool {
		return sortedItems[i].EvidenceID < sortedItems[j].EvidenceID
	})

	h := sha256.New()
	for _, it := range sortedItems {
		line := fmt.Sprintf("%s:%s:%s:%s:%d\n", it.EvidenceID, it.ObjectType, it.ParentEvidenceID, it.SHA256Digest, it.SizeBytes)
		h.Write([]byte(line))
	}
	return hex.EncodeToString(h.Sum(nil))
}

// GenerateExportManifest produces a verified export manifest for the specified evidence IDs.
func (m *EvidenceIntegrityManager) GenerateExportManifest(tenantID, exportID string, evidenceIDs []string, exporterID string) (*ExportManifest, error) {
	tTenant := strings.TrimSpace(tenantID)
	if tTenant == "" {
		return nil, ErrEmptyTenantID
	}
	tExport := strings.TrimSpace(exportID)
	if tExport == "" {
		return nil, ErrBlankExportID
	}
	if len(evidenceIDs) == 0 {
		return nil, ErrEmptyExportItems
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	tenantMap, exists := m.records[tTenant]
	if !exists {
		return nil, ErrObjectNotFound
	}

	var items []ExportManifestItem
	var totalBytes int64

	for _, eid := range evidenceIDs {
		rec, exists := tenantMap[eid]
		if !exists {
			return nil, fmt.Errorf("item %s not found in tenant scope: %w", eid, ErrObjectNotFound)
		}
		if !rec.Committed {
			return nil, fmt.Errorf("item %s is not committed: %w", eid, ErrRecordNotCommitted)
		}

		item := ExportManifestItem{
			EvidenceID:       rec.EvidenceID,
			ObjectType:       rec.ObjectType,
			ParentEvidenceID: rec.ParentEvidenceID,
			DerivationType:   rec.DerivationType,
			MediaType:        rec.MediaType,
			SizeBytes:        rec.SizeBytes,
			SHA256Digest:     rec.SHA256Digest,
		}
		items = append(items, item)
		totalBytes += rec.SizeBytes

		m.recordEventLocked(tTenant, eid, EventExportPackaged, exporterID, rec.SHA256Digest,
			fmt.Sprintf("included in export package %s", tExport))
	}

	rootDigest := computeRootDigest(items)

	manifest := &ExportManifest{
		ExportID:       tExport,
		TenantID:       tTenant,
		ExporterID:     exporterID,
		CreatedAt:      time.Now().UTC(),
		Items:          items,
		RootDigest:     rootDigest,
		TotalSizeBytes: totalBytes,
	}

	return manifest, nil
}

// VerifyExportManifest verifies that each payload in the package matches the manifest item digest,
// and confirms the manifest root digest integrity.
func VerifyExportManifest(manifest *ExportManifest, payloads map[string][]byte) error {
	if manifest == nil {
		return errors.New("manifest cannot be nil")
	}
	if len(manifest.Items) == 0 {
		return ErrEmptyExportItems
	}

	// Verify root digest
	expectedRoot := computeRootDigest(manifest.Items)
	if manifest.RootDigest != expectedRoot {
		return ErrExportTampered
	}

	// Verify each individual item payload
	for _, it := range manifest.Items {
		payload, exists := payloads[it.EvidenceID]
		if !exists {
			return fmt.Errorf("missing payload for item %s: %w", it.EvidenceID, ErrExportTampered)
		}
		if int64(len(payload)) != it.SizeBytes {
			return fmt.Errorf("size mismatch for item %s: %w", it.EvidenceID, ErrExportTampered)
		}
		actualDigest := ComputeSHA256Digest(payload)
		if actualDigest != it.SHA256Digest {
			return fmt.Errorf("digest mismatch for item %s: %w", it.EvidenceID, ErrExportTampered)
		}
	}

	return nil
}
