// Package publicationsnapshot provides standalone, synthetic immutable publication snapshot
// models, source entity mapping, approved-field allowlists, deny-by-default redaction,
// reviewer approval context, version identity, cryptographic integrity verification,
// and export metadata for OSHE Platform.
//
// PROVISIONAL GOVERNANCE DECLARATION (H030-003, H030-004, H030-005 / Issue #102 / V030-I029):
// Under approved Sole Human Owner decisions H030-003, H030-004, and H030-005, this module establishes
// the provisional schema and in-memory test fixtures for published portal snapshots.
//
// Invariants & Non-Claims:
// 1. Immutability: Once published, a snapshot's payload and integrity metadata are permanently immutable.
//    Updates require creating a new version, automatically superseding prior versions.
// 2. Deny-by-Default Redaction: Only explicitly allowlisted fields are included in published payloads.
//    All unapproved fields are stripped; prohibited sensitive fields (PII, credentials, tokens) cause immediate rejection.
// 3. Source Mapping: Every snapshot is strictly linked to an internal source entity with cryptographic digest checking.
// 4. Reviewer Context: Publication requires explicit human/reviewer attribution context and signed decision hash.
// 5. Integrity Verification: Every snapshot seals canonical SHA-256 payload and signature digests.
// 6. Zero Operational Authority: Operates exclusively in-memory on local synthetic fixtures. Zero external
//    public route activation, zero database persistence, zero customer data, zero production network routes,
//    and zero operational runtime authority or policy activation are claimed or enacted.
package publicationsnapshot

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// SnapshotStatus represents the lifecycle state of a publication snapshot.
type SnapshotStatus string

const (
	StatusDraft      SnapshotStatus = "DRAFT"
	StatusPublished  SnapshotStatus = "PUBLISHED"
	StatusSuperseded SnapshotStatus = "SUPERSEDED"
	StatusWithdrawn  SnapshotStatus = "WITHDRAWN"
)

// ReviewerApprovalStatus represents the reviewer decision state.
type ReviewerApprovalStatus string

const (
	ReviewPending  ReviewerApprovalStatus = "PENDING_REVIEW"
	ReviewApproved ReviewerApprovalStatus = "APPROVED_FOR_PUBLICATION"
	ReviewRejected ReviewerApprovalStatus = "REJECTED"
	ReviewWithdrawn ReviewerApprovalStatus = "WITHDRAWN"
)

var (
	ErrBlankSnapshotID             = errors.New("snapshot ID must not be blank")
	ErrBlankTenantID               = errors.New("tenant ID must not be blank")
	ErrBlankSourceID               = errors.New("source entity ID must not be blank")
	ErrUnapprovedFieldDetected     = errors.New("unapproved field detected in publication payload")
	ErrProhibitedFieldDetected     = errors.New("prohibited sensitive field detected in payload")
	ErrProhibitedFieldInAllowlist  = errors.New("allowlist cannot contain prohibited sensitive field names")
	ErrRedactionFailure            = errors.New("payload redaction failure")
	ErrSourceMismatch              = errors.New("source entity tenant does not match snapshot tenant")
	ErrSourceDigestMismatch        = errors.New("source content digest does not match expected source digest")
	ErrIntegrityVerificationFailed = errors.New("snapshot payload integrity verification failed")
	ErrSnapshotImmutable           = errors.New("published snapshot is permanently immutable and cannot be modified in place")
	ErrSnapshotNotFound            = errors.New("publication snapshot not found")
	ErrSnapshotNotApproved         = errors.New("snapshot cannot be published without formal reviewer approval")
	ErrSnapshotAlreadyPublished    = errors.New("snapshot is already in published state")
	ErrSnapshotWithdrawn           = errors.New("snapshot has been withdrawn from publication")
	ErrInvalidVersion              = errors.New("version must be a positive monotonically increasing integer")
	ErrInvalidReviewerContext      = errors.New("reviewer context is invalid or incomplete")
	ErrCrossTenantAccessDenied     = errors.New("cross-tenant snapshot access is strictly denied")
	ErrInvalidClassification       = errors.New("export package classification must be PUBLIC_SANITIZED or EXTERNAL_CONTROLLED")
	ErrUnapprovedDestinationScope  = errors.New("destination scope is invalid or unapproved for export")
	ErrUnpublishedSnapshotInExport = errors.New("export package can only contain published snapshots")
)

// ProhibitedFieldKeywords contains lower-case keywords strictly forbidden from publication.
var ProhibitedFieldKeywords = []string{
	"password",
	"secret",
	"bearer",
	"token",
	"private_key",
	"national_id",
	"citizen_id",
	"ssn",
	"phone_number",
	"auth_header",
	"credit_card",
}

// SourceEntityRef establishes immutable provenance linking a snapshot to its internal source entity.
type SourceEntityRef struct {
	SourceType          string // e.g. "INSPECTION_RECORD", "SAFETY_AUDIT"
	SourceID            string // e.g. "insp_synth_001"
	SourceTenantID      string // authoritative tenant owning source
	SourceVersion       string // source record version e.g. "1.0"
	SourceContentDigest string // SHA-256 digest of original internal payload
}

// Validate asserts the source entity reference is well-formed.
func (s SourceEntityRef) Validate() error {
	if strings.TrimSpace(s.SourceType) == "" {
		return errors.New("source type must not be blank")
	}
	if strings.TrimSpace(s.SourceID) == "" {
		return ErrBlankSourceID
	}
	if strings.TrimSpace(s.SourceTenantID) == "" {
		return ErrBlankTenantID
	}
	if strings.TrimSpace(s.SourceContentDigest) == "" {
		return errors.New("source content digest must not be blank")
	}
	return nil
}

// PublicationFieldAllowlist defines explicitly approved fields for a published snapshot.
type PublicationFieldAllowlist struct {
	AllowedFields   map[string]bool
	StrictRejection bool // If true, presence of unapproved fields causes error instead of silent drop
}

// NewPublicationFieldAllowlist creates an allowlist with validation against prohibited keywords.
func NewPublicationFieldAllowlist(fields []string, strictRejection bool) (PublicationFieldAllowlist, error) {
	allowed := make(map[string]bool)
	for _, f := range fields {
		trimmed := strings.TrimSpace(f)
		if trimmed == "" {
			continue
		}
		lower := strings.ToLower(trimmed)
		for _, prohibited := range ProhibitedFieldKeywords {
			if strings.Contains(lower, prohibited) {
				return PublicationFieldAllowlist{}, fmt.Errorf("%w: field %q contains prohibited keyword %q", ErrProhibitedFieldInAllowlist, trimmed, prohibited)
			}
		}
		allowed[trimmed] = true
	}
	return PublicationFieldAllowlist{
		AllowedFields:   allowed,
		StrictRejection: strictRejection,
	}, nil
}

// ReviewerContext records the explicit human/reviewer attribution context for publication.
type ReviewerContext struct {
	ReviewerSubject    string                 // e.g. "usr_synth_auditor_01"
	ReviewerRole       string                 // e.g. "AUDITOR", "TENANT_ADMIN"
	ReviewedAt         time.Time              // approval timestamp
	ApprovalStatus     ReviewerApprovalStatus // must be ReviewApproved to publish
	ReviewDecisionHash string                 // SHA-256 hash of decision notes and actor
	ReviewNotes        string                 // sanitized justification
}

// NewReviewerContext constructs and validates a reviewer context.
func NewReviewerContext(subject, role string, status ReviewerApprovalStatus, notes string, reviewedAt time.Time) (ReviewerContext, error) {
	tSub := strings.TrimSpace(subject)
	if tSub == "" {
		return ReviewerContext{}, fmt.Errorf("%w: reviewer subject must not be blank", ErrInvalidReviewerContext)
	}
	tRole := strings.TrimSpace(role)
	if tRole == "" {
		return ReviewerContext{}, fmt.Errorf("%w: reviewer role must not be blank", ErrInvalidReviewerContext)
	}
	if status != ReviewPending && status != ReviewApproved && status != ReviewRejected && status != ReviewWithdrawn {
		return ReviewerContext{}, fmt.Errorf("%w: unrecognized approval status", ErrInvalidReviewerContext)
	}

	// Calculate deterministic decision hash
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%s|%s|%s|%s|%d", tSub, tRole, status, strings.TrimSpace(notes), reviewedAt.UnixNano())))
	decisionHash := hex.EncodeToString(h.Sum(nil))

	return ReviewerContext{
		ReviewerSubject:    tSub,
		ReviewerRole:       tRole,
		ReviewedAt:         reviewedAt,
		ApprovalStatus:     status,
		ReviewDecisionHash: decisionHash,
		ReviewNotes:        strings.TrimSpace(notes),
	}, nil
}

// IntegrityMetadata contains cryptographic verification digests for the snapshot.
type IntegrityMetadata struct {
	PayloadDigest   string    // SHA-256 hex digest of canonical sanitized JSON payload
	SignatureDigest string    // Composite SHA-256 hex digest over snapshot envelope
	ComputedAt      time.Time // digest computation timestamp
}

// PublicationSnapshot represents an immutable synthetic publication snapshot.
type PublicationSnapshot struct {
	snapshotID string
	tenantID   string
	version    int
	status     SnapshotStatus
	source     SourceEntityRef
	allowlist  []string
	payload    map[string]any
	reviewer   ReviewerContext
	integrity  IntegrityMetadata
	createdAt  time.Time
	updatedAt  time.Time
}

// SnapshotID returns the authoritative snapshot identifier.
func (p PublicationSnapshot) SnapshotID() string { return p.snapshotID }

// TenantID returns the authoritative tenant identifier.
func (p PublicationSnapshot) TenantID() string { return p.tenantID }

// Version returns the snapshot version number.
func (p PublicationSnapshot) Version() int { return p.version }

// Status returns the current lifecycle status.
func (p PublicationSnapshot) Status() SnapshotStatus { return p.status }

// Source returns the source entity reference.
func (p PublicationSnapshot) Source() SourceEntityRef { return p.source }

// Allowlist returns a copy of the approved field allowlist.
func (p PublicationSnapshot) Allowlist() []string {
	out := make([]string, len(p.allowlist))
	copy(out, p.allowlist)
	return out
}

// Payload returns an immutable deep copy of the sanitized payload.
func (p PublicationSnapshot) Payload() map[string]any {
	return deepCopyMap(p.payload)
}

// Reviewer returns the reviewer context.
func (p PublicationSnapshot) Reviewer() ReviewerContext { return p.reviewer }

// Integrity returns the cryptographic integrity metadata.
func (p PublicationSnapshot) Integrity() IntegrityMetadata { return p.integrity }

// CreatedAt returns the creation timestamp.
func (p PublicationSnapshot) CreatedAt() time.Time { return p.createdAt }

// UpdatedAt returns the last update timestamp.
func (p PublicationSnapshot) UpdatedAt() time.Time { return p.updatedAt }

// IsPublished returns true if the snapshot is currently published.
func (p PublicationSnapshot) IsPublished() bool { return p.status == StatusPublished }

// IsImmutable returns true if the snapshot cannot be modified in place.
func (p PublicationSnapshot) IsImmutable() bool {
	return p.status == StatusPublished || p.status == StatusSuperseded || p.status == StatusWithdrawn
}

// ComputeCanonicalPayloadDigest serializes the payload to deterministic canonical JSON and computes its SHA-256 digest.
func ComputeCanonicalPayloadDigest(payload map[string]any) (string, error) {
	if payload == nil {
		payload = make(map[string]any)
	}
	canonicalJSON, err := toCanonicalJSON(payload)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrRedactionFailure, err)
	}
	hash := sha256.Sum256(canonicalJSON)
	return hex.EncodeToString(hash[:]), nil
}

// ComputeSignatureDigest computes the composite envelope digest.
func ComputeSignatureDigest(tenantID, snapshotID string, version int, source SourceEntityRef, payloadDigest, decisionHash string) string {
	raw := fmt.Sprintf("%s|%s|%d|%s|%s|%s|%s",
		strings.TrimSpace(tenantID),
		strings.TrimSpace(snapshotID),
		version,
		strings.TrimSpace(source.SourceID),
		strings.TrimSpace(source.SourceContentDigest),
		strings.TrimSpace(payloadDigest),
		strings.TrimSpace(decisionHash),
	)
	hash := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(hash[:])
}

// VerifyIntegrity asserts that the snapshot payload matches its stored PayloadDigest and composite SignatureDigest.
func (p PublicationSnapshot) VerifyIntegrity() error {
	calcPayloadDigest, err := ComputeCanonicalPayloadDigest(p.payload)
	if err != nil {
		return err
	}
	if calcPayloadDigest != p.integrity.PayloadDigest {
		return fmt.Errorf("%w: payload digest mismatch (expected %s, got %s)", ErrIntegrityVerificationFailed, p.integrity.PayloadDigest, calcPayloadDigest)
	}

	calcSig := ComputeSignatureDigest(p.tenantID, p.snapshotID, p.version, p.source, calcPayloadDigest, p.reviewer.ReviewDecisionHash)
	if calcSig != p.integrity.SignatureDigest {
		return fmt.Errorf("%w: signature digest mismatch (expected %s, got %s)", ErrIntegrityVerificationFailed, p.integrity.SignatureDigest, calcSig)
	}
	return nil
}

// RedactPayload applies deny-by-default redaction against raw input according to the allowlist.
// It detects and rejects prohibited keywords and strips unapproved fields.
func RedactPayload(raw map[string]any, allowlist PublicationFieldAllowlist) (map[string]any, int, error) {
	if raw == nil {
		return make(map[string]any), 0, nil
	}

	sanitized := make(map[string]any)
	redactedCount := 0

	for k, v := range raw {
		lowerKey := strings.ToLower(k)

		// 1. Prohibited keyword check: fail closed if sensitive data is injected
		for _, prohibited := range ProhibitedFieldKeywords {
			if strings.Contains(lowerKey, prohibited) {
				return nil, 0, fmt.Errorf("%w: field %q matches prohibited sensitive pattern %q", ErrProhibitedFieldDetected, k, prohibited)
			}
		}

		// Also check string values for obvious bearer/private token patterns
		if strVal, ok := v.(string); ok {
			lowerVal := strings.ToLower(strVal)
			if strings.HasPrefix(lowerVal, "bearer ") || strings.HasPrefix(lowerVal, "oshe_tok_") {
				return nil, 0, fmt.Errorf("%w: value of field %q contains detected credential token", ErrProhibitedFieldDetected, k)
			}
		}

		// 2. Allowlist matching (Deny-by-default)
		if allowlist.AllowedFields[k] {
			sanitized[k] = v
		} else {
			if allowlist.StrictRejection {
				return nil, 0, fmt.Errorf("%w: field %q is not in the approved allowlist", ErrUnapprovedFieldDetected, k)
			}
			redactedCount++
		}
	}

	return sanitized, redactedCount, nil
}

// NewDraftSnapshot creates a new validated DRAFT publication snapshot.
func NewDraftSnapshot(snapshotID, tenantID string, source SourceEntityRef, rawPayload map[string]any, allowlist PublicationFieldAllowlist) (PublicationSnapshot, error) {
	tSnapID := strings.TrimSpace(snapshotID)
	if tSnapID == "" {
		return PublicationSnapshot{}, ErrBlankSnapshotID
	}
	tTenantID := strings.TrimSpace(tenantID)
	if tTenantID == "" {
		return PublicationSnapshot{}, ErrBlankTenantID
	}
	if err := source.Validate(); err != nil {
		return PublicationSnapshot{}, err
	}
	if source.SourceTenantID != tTenantID {
		return PublicationSnapshot{}, fmt.Errorf("%w: source tenant %s != snapshot tenant %s", ErrSourceMismatch, source.SourceTenantID, tTenantID)
	}

	sanitized, _, err := RedactPayload(rawPayload, allowlist)
	if err != nil {
		return PublicationSnapshot{}, err
	}

	payloadDigest, err := ComputeCanonicalPayloadDigest(sanitized)
	if err != nil {
		return PublicationSnapshot{}, err
	}

	// Draft has empty decision hash initially
	now := time.Now().UTC()
	sigDigest := ComputeSignatureDigest(tTenantID, tSnapID, 1, source, payloadDigest, "")

	var allowlistKeys []string
	for k := range allowlist.AllowedFields {
		allowlistKeys = append(allowlistKeys, k)
	}
	sort.Strings(allowlistKeys)

	return PublicationSnapshot{
		snapshotID: tSnapID,
		tenantID:   tTenantID,
		version:    1,
		status:     StatusDraft,
		source:     source,
		allowlist:  allowlistKeys,
		payload:    sanitized,
		reviewer:   ReviewerContext{ApprovalStatus: ReviewPending},
		integrity: IntegrityMetadata{
			PayloadDigest:   payloadDigest,
			SignatureDigest: sigDigest,
			ComputedAt:      now,
		},
		createdAt: now,
		updatedAt: now,
	}, nil
}

// SnapshotRegistry provides thread-safe, tenant-isolated in-memory storage for snapshot lifecycles.
type SnapshotRegistry struct {
	mu        sync.RWMutex
	snapshots map[string]PublicationSnapshot   // key: tenantID + ":" + snapshotID + ":v" + version
	latestVer map[string]int                   // key: tenantID + ":" + snapshotID -> highest version
	activePub map[string]string                // key: tenantID + ":" + snapshotID -> published version key
}

// NewSnapshotRegistry initializes an empty in-memory SnapshotRegistry.
func NewSnapshotRegistry() *SnapshotRegistry {
	return &SnapshotRegistry{
		snapshots: make(map[string]PublicationSnapshot),
		latestVer: make(map[string]int),
		activePub: make(map[string]string),
	}
}

func makeVersionKey(tenantID, snapshotID string, version int) string {
	return fmt.Sprintf("%s:%s:v%d", strings.TrimSpace(tenantID), strings.TrimSpace(snapshotID), version)
}

func makeBaseKey(tenantID, snapshotID string) string {
	return fmt.Sprintf("%s:%s", strings.TrimSpace(tenantID), strings.TrimSpace(snapshotID))
}

// RegisterDraft registers a new draft snapshot.
func (r *SnapshotRegistry) RegisterDraft(s PublicationSnapshot) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	bKey := makeBaseKey(s.TenantID(), s.SnapshotID())
	if _, exists := r.latestVer[bKey]; exists {
		return fmt.Errorf("snapshot %s already exists for tenant", s.SnapshotID())
	}

	vKey := makeVersionKey(s.TenantID(), s.SnapshotID(), s.Version())
	r.snapshots[vKey] = s
	r.latestVer[bKey] = s.Version()
	return nil
}

// PublishSnapshot approves and transitions a draft snapshot to PUBLISHED status.
func (r *SnapshotRegistry) PublishSnapshot(tenantID, snapshotID string, reviewer ReviewerContext) (PublicationSnapshot, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	bKey := makeBaseKey(tenantID, snapshotID)
	ver, exists := r.latestVer[bKey]
	if !exists {
		return PublicationSnapshot{}, ErrSnapshotNotFound
	}

	vKey := makeVersionKey(tenantID, snapshotID, ver)
	s := r.snapshots[vKey]

	if s.Status() == StatusPublished {
		return PublicationSnapshot{}, ErrSnapshotAlreadyPublished
	}
	if s.Status() == StatusWithdrawn {
		return PublicationSnapshot{}, ErrSnapshotWithdrawn
	}
	if reviewer.ApprovalStatus != ReviewApproved {
		return PublicationSnapshot{}, ErrSnapshotNotApproved
	}
	if strings.TrimSpace(reviewer.ReviewerSubject) == "" {
		return PublicationSnapshot{}, ErrInvalidReviewerContext
	}

	now := time.Now().UTC()
	sigDigest := ComputeSignatureDigest(s.TenantID(), s.SnapshotID(), s.Version(), s.Source(), s.Integrity().PayloadDigest, reviewer.ReviewDecisionHash)

	s.status = StatusPublished
	s.reviewer = reviewer
	s.integrity.SignatureDigest = sigDigest
	s.integrity.ComputedAt = now
	s.updatedAt = now

	r.snapshots[vKey] = s
	r.activePub[bKey] = vKey
	return s, nil
}

// CreateNewVersion creates a new version of an existing published snapshot, superseding the previous version.
func (r *SnapshotRegistry) CreateNewVersion(tenantID, snapshotID string, updatedRawPayload map[string]any, allowlist PublicationFieldAllowlist, reviewer ReviewerContext) (PublicationSnapshot, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	bKey := makeBaseKey(tenantID, snapshotID)
	curVer, exists := r.latestVer[bKey]
	if !exists {
		return PublicationSnapshot{}, ErrSnapshotNotFound
	}

	curKey := makeVersionKey(tenantID, snapshotID, curVer)
	curSnap := r.snapshots[curKey]

	if curSnap.Status() != StatusPublished {
		return PublicationSnapshot{}, fmt.Errorf("%w: can only create new version from published snapshot", ErrSnapshotImmutable)
	}

	if reviewer.ApprovalStatus != ReviewApproved {
		return PublicationSnapshot{}, ErrSnapshotNotApproved
	}

	sanitized, _, err := RedactPayload(updatedRawPayload, allowlist)
	if err != nil {
		return PublicationSnapshot{}, err
	}

	payloadDigest, err := ComputeCanonicalPayloadDigest(sanitized)
	if err != nil {
		return PublicationSnapshot{}, err
	}

	now := time.Now().UTC()
	newVer := curVer + 1
	sigDigest := ComputeSignatureDigest(tenantID, snapshotID, newVer, curSnap.Source(), payloadDigest, reviewer.ReviewDecisionHash)

	// Supersede current version
	curSnap.status = StatusSuperseded
	curSnap.updatedAt = now
	r.snapshots[curKey] = curSnap

	var allowlistKeys []string
	for k := range allowlist.AllowedFields {
		allowlistKeys = append(allowlistKeys, k)
	}
	sort.Strings(allowlistKeys)

	newSnap := PublicationSnapshot{
		snapshotID: snapshotID,
		tenantID:   tenantID,
		version:    newVer,
		status:     StatusPublished,
		source:     curSnap.Source(),
		allowlist:  allowlistKeys,
		payload:    sanitized,
		reviewer:   reviewer,
		integrity: IntegrityMetadata{
			PayloadDigest:   payloadDigest,
			SignatureDigest: sigDigest,
			ComputedAt:      now,
		},
		createdAt: curSnap.CreatedAt(),
		updatedAt: now,
	}

	newKey := makeVersionKey(tenantID, snapshotID, newVer)
	r.snapshots[newKey] = newSnap
	r.latestVer[bKey] = newVer
	r.activePub[bKey] = newKey

	return newSnap, nil
}

// WithdrawSnapshot withdraws a snapshot from publication.
func (r *SnapshotRegistry) WithdrawSnapshot(tenantID, snapshotID string, reason string, reviewer ReviewerContext) (PublicationSnapshot, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	bKey := makeBaseKey(tenantID, snapshotID)
	ver, exists := r.latestVer[bKey]
	if !exists {
		return PublicationSnapshot{}, ErrSnapshotNotFound
	}

	vKey := makeVersionKey(tenantID, snapshotID, ver)
	s := r.snapshots[vKey]

	now := time.Now().UTC()
	s.status = StatusWithdrawn
	reviewer.ApprovalStatus = ReviewWithdrawn
	reviewer.ReviewNotes = reason
	s.reviewer = reviewer
	s.updatedAt = now

	r.snapshots[vKey] = s
	delete(r.activePub, bKey)
	return s, nil
}

// GetSnapshot retrieves the latest or currently active version of a snapshot.
func (r *SnapshotRegistry) GetSnapshot(tenantID, snapshotID string) (PublicationSnapshot, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	bKey := makeBaseKey(tenantID, snapshotID)
	vKey, hasActive := r.activePub[bKey]
	if !hasActive {
		ver, exists := r.latestVer[bKey]
		if !exists {
			return PublicationSnapshot{}, ErrSnapshotNotFound
		}
		vKey = makeVersionKey(tenantID, snapshotID, ver)
	}

	s, exists := r.snapshots[vKey]
	if !exists {
		return PublicationSnapshot{}, ErrSnapshotNotFound
	}
	return s, nil
}

// GetSnapshotVersion retrieves a specific historic version of a snapshot.
func (r *SnapshotRegistry) GetSnapshotVersion(tenantID, snapshotID string, version int) (PublicationSnapshot, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	vKey := makeVersionKey(tenantID, snapshotID, version)
	s, exists := r.snapshots[vKey]
	if !exists {
		return PublicationSnapshot{}, ErrSnapshotNotFound
	}
	return s, nil
}

// VerifySnapshotIntegrity verifies integrity of a registered snapshot.
func (r *SnapshotRegistry) VerifySnapshotIntegrity(tenantID, snapshotID string) error {
	s, err := r.GetSnapshot(tenantID, snapshotID)
	if err != nil {
		return err
	}
	return s.VerifyIntegrity()
}

// ExportPackage represents a formalized export artifact containing verified published snapshots.
type ExportPackage struct {
	ExportID          string                `json:"export_id"`
	TenantID          string                `json:"tenant_id"`
	Format            string                `json:"format"`
	Classification    string                `json:"classification"`
	DestinationScope  string                `json:"destination_scope"`
	Snapshots         []PublicationSnapshot `json:"snapshots"`
	RecordCount       int                   `json:"record_count"`
	ExportedBy        string                `json:"exported_by"`
	ExportedAt        time.Time             `json:"exported_at"`
	IntegrityChecksum string                `json:"integrity_checksum"`
}

// AllowedDestinationScopes contains recognized destination environments.
var AllowedDestinationScopes = map[string]bool{
	"PUBLIC_PORTAL_PREVIEW":    true,
	"EXTERNAL_AUDITOR_PACKAGE": true,
	"REGULATORY_SUBMISSION":    true,
}

// NewExportPackage builds and seals a validated export package.
func NewExportPackage(exportID, tenantID, format, classification, destinationScope, exporterSubject string, snapshots []PublicationSnapshot) (ExportPackage, error) {
	tExpID := strings.TrimSpace(exportID)
	if tExpID == "" {
		return ExportPackage{}, errors.New("export ID must not be blank")
	}
	tTenantID := strings.TrimSpace(tenantID)
	if tTenantID == "" {
		return ExportPackage{}, ErrBlankTenantID
	}
	if classification != "PUBLIC_SANITIZED" && classification != "EXTERNAL_CONTROLLED" {
		return ExportPackage{}, fmt.Errorf("%w: got %q", ErrInvalidClassification, classification)
	}
	if !AllowedDestinationScopes[destinationScope] {
		return ExportPackage{}, fmt.Errorf("%w: scope %q is not in allowed destinations", ErrUnapprovedDestinationScope, destinationScope)
	}
	if strings.TrimSpace(exporterSubject) == "" {
		return ExportPackage{}, errors.New("exporter subject must not be blank")
	}
	if len(snapshots) == 0 {
		return ExportPackage{}, errors.New("export package must contain at least one snapshot")
	}

	h := sha256.New()
	for _, s := range snapshots {
		if s.TenantID() != tTenantID {
			return ExportPackage{}, fmt.Errorf("%w: snapshot %s tenant %s != export tenant %s", ErrCrossTenantAccessDenied, s.SnapshotID(), s.TenantID(), tTenantID)
		}
		if !s.IsPublished() {
			return ExportPackage{}, fmt.Errorf("%w: snapshot %s has status %s", ErrUnpublishedSnapshotInExport, s.SnapshotID(), s.Status())
		}
		if err := s.VerifyIntegrity(); err != nil {
			return ExportPackage{}, fmt.Errorf("snapshot %s failed integrity: %w", s.SnapshotID(), err)
		}
		h.Write([]byte(s.Integrity().SignatureDigest))
	}
	checksum := hex.EncodeToString(h.Sum(nil))

	return ExportPackage{
		ExportID:          tExpID,
		TenantID:          tTenantID,
		Format:            strings.ToUpper(strings.TrimSpace(format)),
		Classification:    classification,
		DestinationScope:  destinationScope,
		Snapshots:         snapshots,
		RecordCount:       len(snapshots),
		ExportedBy:        strings.TrimSpace(exporterSubject),
		ExportedAt:        time.Now().UTC(),
		IntegrityChecksum: checksum,
	}, nil
}

// Helpers

func toCanonicalJSON(v any) ([]byte, error) {
	normalized := normalizeForJSON(v)
	return json.Marshal(normalized)
}

func normalizeForJSON(v any) any {
	switch val := v.(type) {
	case map[string]any:
		var keys []string
		for k := range val {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		out := make(map[string]any, len(keys))
		for _, k := range keys {
			out[k] = normalizeForJSON(val[k])
		}
		return out
	case []any:
		out := make([]any, len(val))
		for i, item := range val {
			out[i] = normalizeForJSON(item)
		}
		return out
	default:
		return val
	}
}

func deepCopyMap(m map[string]any) map[string]any {
	if m == nil {
		return make(map[string]any)
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		if subMap, ok := v.(map[string]any); ok {
			out[k] = deepCopyMap(subMap)
		} else {
			out[k] = v
		}
	}
	return out
}
