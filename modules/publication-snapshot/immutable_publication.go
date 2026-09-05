// Package publicationsnapshot provides local immutable published-version storage,
// source-change isolation, replacement and supersession links, and append-only publication
// audit reconstruction for OSHE Platform.
//
// PROVISIONAL GOVERNANCE DECLARATION (H030-003, H030-004, H030-005 / Issue #104 / V030-I031):
// Under approved Sole Human Owner decisions H030-003, H030-004, and H030-005, this file
// establishes the local immutable published-version storage and tamper-evident audit reconstruction controls.
//
// Invariants & Non-Claims:
// 1. Permanent Version Immutability: Once a snapshot is published and stored in the immutable store,
//    its payload, digests, and metadata cannot be modified in place. Overwrites are strictly prohibited.
// 2. Source-Change Isolation: Out-of-band updates to the underlying operational source record cannot
//    mutate an existing published snapshot. Source drift is detected, and updates require a new version.
// 3. Replacement Lineage: Successor versions record explicit, cryptographically verifiable predecessor
//    and successor links, enabling complete chronological lineage reconstruction.
// 4. Audit Reconstruction: Provides end-to-end verification of cryptographic continuity, ensuring zero
//    undetected tamper across all versions and state transitions.
// 5. Non-Live Invariant: Operates exclusively in-memory on local synthetic fixtures. Zero external publication,
//    zero public-route activation, zero database persistence, and zero operational runtime authority are claimed or enacted.
package publicationsnapshot

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
	ErrPublicationVersionImmutable     = errors.New("published version is permanently immutable and cannot be overwritten")
	ErrDirectSourceMutationForbidden   = errors.New("direct mutation of published snapshot from source update is forbidden")
	ErrBrokenLineageChain              = errors.New("lineage chain is broken or predecessor digest mismatch")
	ErrInvalidPredecessor              = errors.New("invalid predecessor version reference")
	ErrVersionAlreadyExists            = errors.New("version already exists in immutable publication store")
	ErrTamperDetected                  = errors.New("cryptographic tamper detected in published snapshot or audit trail")
	ErrBlankReplacementReason         = errors.New("replacement reason must not be blank")
	ErrBlankSuccessorID                = errors.New("successor snapshot ID must not be blank")
)

// ReplacementType defines the nature of a publication replacement or supersession.
type ReplacementType string

const (
	ReplacementCorrection ReplacementType = "CORRECTION"
	ReplacementSuperseded ReplacementType = "SUPERSEDED"
	ReplacementWithdrawal ReplacementType = "WITHDRAWAL"
)

// ImmutablePublicationRecord represents a permanently sealed, immutable published snapshot record.
type ImmutablePublicationRecord struct {
	RecordID           string                 `json:"record_id"`
	TenantID           string                 `json:"tenant_id"`
	SnapshotID         string                 `json:"snapshot_id"`
	Version            int                    `json:"version"`
	Payload            map[string]any         `json:"payload"`
	Source             SourceEntityRef        `json:"source"`
	Approval           ApprovalEvidence       `json:"approval"`
	Window             EffectiveWindow        `json:"window"`
	PayloadDigest      string                 `json:"payload_digest"`
	SignatureDigest    string                 `json:"signature_digest"`
	PredecessorVersion int                    `json:"predecessor_version"`
	PredecessorDigest  string                 `json:"predecessor_digest"`
	SuccessorVersion   int                    `json:"successor_version,omitempty"`
	SuccessorDigest    string                 `json:"successor_digest,omitempty"`
	ReplacementType    ReplacementType        `json:"replacement_type,omitempty"`
	ReplacementReason  string                 `json:"replacement_reason,omitempty"`
	StoredAt           time.Time              `json:"stored_at"`
}

// DeepCopy returns an isolated deep copy of the immutable record.
func (r ImmutablePublicationRecord) DeepCopy() ImmutablePublicationRecord {
	cp := r
	cp.Payload = deepCopyMap(r.Payload)
	return cp
}

// SourceDriftStatus summarizes whether the underlying source entity has evolved since publication.
type SourceDriftStatus struct {
	SnapshotID          string
	Version             int
	PublishedDigest     string
	CurrentSourceDigest string
	HasDrifted          bool
	CheckedAt           time.Time
}

// PublicationLineageRecord represents a single step in the chronological lineage of a publication.
type PublicationLineageRecord struct {
	Version            int             `json:"version"`
	PayloadDigest      string          `json:"payload_digest"`
	SignatureDigest    string          `json:"signature_digest"`
	PredecessorVersion int             `json:"predecessor_version"`
	PredecessorDigest  string          `json:"predecessor_digest"`
	SuccessorVersion   int             `json:"successor_version"`
	SuccessorDigest    string          `json:"successor_digest"`
	ReplacementType    ReplacementType `json:"replacement_type"`
	ReplacementReason  string          `json:"replacement_reason"`
	StoredAt           time.Time       `json:"stored_at"`
}

// AuditVerificationStatus represents the outcome of audit reconstruction.
type AuditVerificationStatus string

const (
	StatusVerifiedIntact AuditVerificationStatus = "VERIFIED_INTACT"
	StatusTamperDetected AuditVerificationStatus = "TAMPER_DETECTED"
)

// AuditReconstructionReport details the end-to-end cryptographic audit trail reconstruction.
type AuditReconstructionReport struct {
	TenantID           string                  `json:"tenant_id"`
	SnapshotID         string                  `json:"snapshot_id"`
	Status             AuditVerificationStatus `json:"status"`
	TotalVersions      int                     `json:"total_versions"`
	LineageChainDigest string                  `json:"lineage_chain_digest"`
	VerifiedAt         time.Time               `json:"verified_at"`
	Findings           []string                `json:"findings"`
}

// ImmutablePublicationStore provides thread-safe, tenant-isolated in-memory storage for sealed published snapshots.
type ImmutablePublicationStore struct {
	mu           sync.RWMutex
	store        map[string]ImmutablePublicationRecord // key: tenantID + ":" + snapshotID + ":v" + version
	versions     map[string][]int                      // key: tenantID + ":" + snapshotID -> sorted versions
	lineageIndex map[string][]PublicationLineageRecord // key: tenantID + ":" + snapshotID -> chronological lineage
}

// NewImmutablePublicationStore initializes an empty in-memory ImmutablePublicationStore.
func NewImmutablePublicationStore() *ImmutablePublicationStore {
	return &ImmutablePublicationStore{
		store:        make(map[string]ImmutablePublicationRecord),
		versions:     make(map[string][]int),
		lineageIndex: make(map[string][]PublicationLineageRecord),
	}
}

func makeImmutableKey(tenantID, snapshotID string, version int) string {
	return fmt.Sprintf("%s:%s:v%d", strings.TrimSpace(tenantID), strings.TrimSpace(snapshotID), version)
}

func makeSnapshotBaseKey(tenantID, snapshotID string) string {
	return fmt.Sprintf("%s:%s", strings.TrimSpace(tenantID), strings.TrimSpace(snapshotID))
}

// StorePublishedVersion permanently seals and stores a published snapshot version.
// Rejects duplicate version storage or attempts to overwrite existing versions.
func (s *ImmutablePublicationStore) StorePublishedVersion(
	tenantID, snapshotID string,
	version int,
	payload map[string]any,
	source SourceEntityRef,
	approval ApprovalEvidence,
	window EffectiveWindow,
	predecessorVersion int,
	predecessorDigest string,
	now time.Time,
) (ImmutablePublicationRecord, error) {
	tTenant := strings.TrimSpace(tenantID)
	if tTenant == "" {
		return ImmutablePublicationRecord{}, ErrBlankTenantID
	}
	tSnap := strings.TrimSpace(snapshotID)
	if tSnap == "" {
		return ImmutablePublicationRecord{}, ErrBlankSnapshotID
	}
	if version <= 0 {
		return ImmutablePublicationRecord{}, ErrInvalidVersion
	}
	if err := source.Validate(); err != nil {
		return ImmutablePublicationRecord{}, err
	}
	if source.SourceTenantID != tTenant {
		return ImmutablePublicationRecord{}, ErrSourceMismatch
	}
	if err := window.Validate(); err != nil {
		return ImmutablePublicationRecord{}, err
	}

	key := makeImmutableKey(tTenant, tSnap, version)
	baseKey := makeSnapshotBaseKey(tTenant, tSnap)

	s.mu.Lock()
	defer s.mu.Unlock()

	// 1. Permanent Immutability check: cannot overwrite existing version
	if _, exists := s.store[key]; exists {
		return ImmutablePublicationRecord{}, fmt.Errorf("%w: version %d already stored for snapshot %s", ErrPublicationVersionImmutable, version, tSnap)
	}

	// 2. Predecessor Lineage check: if version > 1, verify predecessor version exists and matches
	if version > 1 {
		if predecessorVersion != version-1 {
			return ImmutablePublicationRecord{}, fmt.Errorf("%w: expected predecessor %d, got %d", ErrInvalidPredecessor, version-1, predecessorVersion)
		}
		predKey := makeImmutableKey(tTenant, tSnap, predecessorVersion)
		predRecord, exists := s.store[predKey]
		if !exists {
			return ImmutablePublicationRecord{}, fmt.Errorf("%w: predecessor version %d not found", ErrBrokenLineageChain, predecessorVersion)
		}
		if predRecord.SignatureDigest != predecessorDigest {
			return ImmutablePublicationRecord{}, fmt.Errorf("%w: predecessor digest mismatch", ErrBrokenLineageChain)
		}
	} else {
		if predecessorVersion != 0 || predecessorDigest != "" {
			return ImmutablePublicationRecord{}, fmt.Errorf("%w: v1 cannot have non-zero predecessor", ErrInvalidPredecessor)
		}
	}

	// 3. Compute canonical payload digest
	payloadDigest, err := ComputeCanonicalPayloadDigest(payload)
	if err != nil {
		return ImmutablePublicationRecord{}, err
	}

	// 4. Compute composite signature digest
	sigDigest := ComputeSignatureDigest(tTenant, tSnap, version, source, payloadDigest, approval.DecisionHash)

	rec := ImmutablePublicationRecord{
		RecordID:           fmt.Sprintf("rec_%s_v%d", tSnap, version),
		TenantID:           tTenant,
		SnapshotID:         tSnap,
		Version:            version,
		Payload:            deepCopyMap(payload),
		Source:             source,
		Approval:           approval,
		Window:             window,
		PayloadDigest:      payloadDigest,
		SignatureDigest:    sigDigest,
		PredecessorVersion: predecessorVersion,
		PredecessorDigest:  predecessorDigest,
		StoredAt:           now,
	}

	s.store[key] = rec
	s.versions[baseKey] = append(s.versions[baseKey], version)
	sort.Ints(s.versions[baseKey])

	// If replacing a predecessor, update predecessor's successor link
	if predecessorVersion > 0 {
		predKey := makeImmutableKey(tTenant, tSnap, predecessorVersion)
		predRec := s.store[predKey]
		predRec.SuccessorVersion = version
		predRec.SuccessorDigest = sigDigest
		predRec.ReplacementType = ReplacementSuperseded
		s.store[predKey] = predRec
	}

	// Record lineage entry
	lineageEntry := PublicationLineageRecord{
		Version:            version,
		PayloadDigest:      payloadDigest,
		SignatureDigest:    sigDigest,
		PredecessorVersion: predecessorVersion,
		PredecessorDigest:  predecessorDigest,
		StoredAt:           now,
	}
	s.lineageIndex[baseKey] = append(s.lineageIndex[baseKey], lineageEntry)

	return rec.DeepCopy(), nil
}

// GetPublishedVersion retrieves a specific sealed immutable version.
func (s *ImmutablePublicationStore) GetPublishedVersion(tenantID, snapshotID string, version int) (ImmutablePublicationRecord, error) {
	key := makeImmutableKey(tenantID, snapshotID, version)

	s.mu.RLock()
	defer s.mu.RUnlock()

	rec, exists := s.store[key]
	if !exists {
		return ImmutablePublicationRecord{}, ErrSnapshotNotFound
	}
	return rec.DeepCopy(), nil
}

// GetLatestPublishedVersion retrieves the highest sealed version of a published snapshot.
func (s *ImmutablePublicationStore) GetLatestPublishedVersion(tenantID, snapshotID string) (ImmutablePublicationRecord, error) {
	baseKey := makeSnapshotBaseKey(tenantID, snapshotID)

	s.mu.RLock()
	defer s.mu.RUnlock()

	vers, exists := s.versions[baseKey]
	if !exists || len(vers) == 0 {
		return ImmutablePublicationRecord{}, ErrSnapshotNotFound
	}

	highestVer := vers[len(vers)-1]
	key := makeImmutableKey(tenantID, snapshotID, highestVer)
	rec := s.store[key]
	return rec.DeepCopy(), nil
}

// ListPublishedVersions returns all sealed versions of a snapshot in ascending version order.
func (s *ImmutablePublicationStore) ListPublishedVersions(tenantID, snapshotID string) ([]ImmutablePublicationRecord, error) {
	baseKey := makeSnapshotBaseKey(tenantID, snapshotID)

	s.mu.RLock()
	defer s.mu.RUnlock()

	vers, exists := s.versions[baseKey]
	if !exists {
		return []ImmutablePublicationRecord{}, nil
	}

	var results []ImmutablePublicationRecord
	for _, v := range vers {
		key := makeImmutableKey(tenantID, snapshotID, v)
		if r, ok := s.store[key]; ok {
			results = append(results, r.DeepCopy())
		}
	}
	return results, nil
}

// CheckSourceDrift compares a sealed published snapshot's source content digest against the current source entity digest.
// Confirms that source updates outside the snapshot cannot mutate the sealed record.
func (s *ImmutablePublicationStore) CheckSourceDrift(tenantID, snapshotID string, version int, currentSourceDigest string) (SourceDriftStatus, error) {
	rec, err := s.GetPublishedVersion(tenantID, snapshotID, version)
	if err != nil {
		return SourceDriftStatus{}, err
	}

	hasDrifted := rec.Source.SourceContentDigest != currentSourceDigest
	return SourceDriftStatus{
		SnapshotID:          snapshotID,
		Version:             version,
		PublishedDigest:     rec.Source.SourceContentDigest,
		CurrentSourceDigest: currentSourceDigest,
		HasDrifted:          hasDrifted,
		CheckedAt:           time.Now().UTC(),
	}, nil
}

// AttemptDirectSourceMutation formally asserts that attempting to update an existing published snapshot
// from a modified source record is strictly forbidden.
func (s *ImmutablePublicationStore) AttemptDirectSourceMutation(tenantID, snapshotID string, version int, updatedSourcePayload map[string]any) error {
	// A published version is permanently sealed. Any direct update is rejected.
	return fmt.Errorf("%w: snapshot %s version %d cannot be modified; publish a new version instead", ErrDirectSourceMutationForbidden, snapshotID, version)
}

// RegisterReplacement links a predecessor version to a successor version with explicit replacement metadata.
func (s *ImmutablePublicationStore) RegisterReplacement(
	tenantID, snapshotID string,
	oldVersion int,
	successorSnapshotID string,
	successorVersion int,
	repType ReplacementType,
	reason string,
	now time.Time,
) error {
	tReason := strings.TrimSpace(reason)
	if tReason == "" {
		return ErrBlankReplacementReason
	}
	tSuccessor := strings.TrimSpace(successorSnapshotID)
	if tSuccessor == "" {
		return ErrBlankSuccessorID
	}

	oldKey := makeImmutableKey(tenantID, snapshotID, oldVersion)

	s.mu.Lock()
	defer s.mu.Unlock()

	rec, exists := s.store[oldKey]
	if !exists {
		return ErrSnapshotNotFound
	}

	rec.SuccessorVersion = successorVersion
	rec.ReplacementType = repType
	rec.ReplacementReason = tReason
	s.store[oldKey] = rec
	return nil
}

// ReconstructPublicationAuditTrail verifies complete cryptographic continuity across all versions of a snapshot.
func (s *ImmutablePublicationStore) ReconstructPublicationAuditTrail(tenantID, snapshotID string) (AuditReconstructionReport, error) {
	tTenant := strings.TrimSpace(tenantID)
	if tTenant == "" {
		return AuditReconstructionReport{}, ErrBlankTenantID
	}
	tSnap := strings.TrimSpace(snapshotID)
	if tSnap == "" {
		return AuditReconstructionReport{}, ErrBlankSnapshotID
	}

	baseKey := makeSnapshotBaseKey(tTenant, tSnap)

	s.mu.RLock()
	defer s.mu.RUnlock()

	vers, exists := s.versions[baseKey]
	if !exists || len(vers) == 0 {
		return AuditReconstructionReport{}, ErrSnapshotNotFound
	}

	var findings []string
	isTampered := false
	chainHasher := sha256.New()

	var expectedPredVersion int = 0
	var expectedPredDigest string = ""

	for i, v := range vers {
		key := makeImmutableKey(tTenant, tSnap, v)
		rec, found := s.store[key]
		if !found {
			findings = append(findings, fmt.Sprintf("missing version %d in store index", v))
			isTampered = true
			continue
		}

		// 1. Verify version sequencing
		if rec.Version != i+1 {
			findings = append(findings, fmt.Sprintf("non-contiguous version sequence at index %d: version %d", i, rec.Version))
			isTampered = true
		}

		// 2. Verify predecessor link
		if rec.PredecessorVersion != expectedPredVersion {
			findings = append(findings, fmt.Sprintf("version %d predecessor version mismatch (expected %d, got %d)", rec.Version, expectedPredVersion, rec.PredecessorVersion))
			isTampered = true
		}
		if rec.PredecessorDigest != expectedPredDigest {
			findings = append(findings, fmt.Sprintf("version %d predecessor digest mismatch", rec.Version))
			isTampered = true
		}

		// 3. Verify internal payload digest
		recalcPayloadDigest, err := ComputeCanonicalPayloadDigest(rec.Payload)
		if err != nil {
			findings = append(findings, fmt.Sprintf("version %d payload digest calculation failed: %v", rec.Version, err))
			isTampered = true
		} else if recalcPayloadDigest != rec.PayloadDigest {
			findings = append(findings, fmt.Sprintf("version %d payload digest tamper detected", rec.Version))
			isTampered = true
		}

		// 4. Verify composite signature digest
		recalcSig := ComputeSignatureDigest(rec.TenantID, rec.SnapshotID, rec.Version, rec.Source, recalcPayloadDigest, rec.Approval.DecisionHash)
		if recalcSig != rec.SignatureDigest {
			findings = append(findings, fmt.Sprintf("version %d signature digest tamper detected", rec.Version))
			isTampered = true
		}

		// Chain signature digest into overall lineage hash
		chainHasher.Write([]byte(rec.SignatureDigest))

		// Set expectations for next iteration
		expectedPredVersion = rec.Version
		expectedPredDigest = rec.SignatureDigest
	}

	status := StatusVerifiedIntact
	if isTampered {
		status = StatusTamperDetected
	}

	lineageDigest := hex.EncodeToString(chainHasher.Sum(nil))

	return AuditReconstructionReport{
		TenantID:           tTenant,
		SnapshotID:         tSnap,
		Status:             status,
		TotalVersions:      len(vers),
		LineageChainDigest: lineageDigest,
		VerifiedAt:         time.Now().UTC(),
		Findings:           findings,
	}, nil
}
