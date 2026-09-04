package recordsaudit

import (
	"errors"
	"strings"
	"sync"
	"time"
)

// ObjectKind discriminates between an original immutable evidence object
// and a derived derivative object.
type ObjectKind string

const (
	KindOriginal ObjectKind = "ORIGINAL"
	KindDerived  ObjectKind = "DERIVED"
)

// DerivedKind defines the specific transformation category of a derived object.
type DerivedKind string

const (
	DerivedAnnotation     DerivedKind = "ANNOTATION"
	DerivedRedaction      DerivedKind = "REDACTION"
	DerivedThumbnail      DerivedKind = "THUMBNAIL"
	DerivedTransformation DerivedKind = "TRANSFORMATION"
)

// RecordLifecycleState represents the discrete lifecycle progression of an object record.
type RecordLifecycleState string

const (
	StateDraft    RecordLifecycleState = "DRAFT"
	StateAccepted RecordLifecycleState = "ACCEPTED"
	StateArchived RecordLifecycleState = "ARCHIVED"
)

var (
	ErrBlankID                    = errors.New("object identifier must not be blank")
	ErrBlankTenantID              = errors.New("tenant identifier must not be blank")
	ErrBlankParentID              = errors.New("parent original identifier must not be blank")
	ErrInvalidDigest              = errors.New("digest must be a 64-character lowercase hexadecimal SHA-256 string")
	ErrInvalidSizeBytes           = errors.New("size bytes must be greater than zero")
	ErrBlankMediaType             = errors.New("media type must not be blank")
	ErrInvalidDerivedKind         = errors.New("invalid derived object kind; must be ANNOTATION, REDACTION, THUMBNAIL, or TRANSFORMATION")
	ErrUnknownParent              = errors.New("parent original record not found")
	ErrParentNotAccepted          = errors.New("parent original record must be in ACCEPTED state before derived objects can link to it")
	ErrCrossTenantLinkage         = errors.New("cross-tenant linkage denied: derived object tenant does not match parent tenant")
	ErrOriginalOverwriteDenied    = errors.New("overwriting an accepted original record is strictly prohibited")
	ErrOriginalAsDerivedDenied    = errors.New("cannot register an original object as a derived object")
	ErrDigestMismatch             = errors.New("digest mismatch: calculated or provided content hash does not match recorded digest")
	ErrInvalidLifecycleTransition = errors.New("invalid lifecycle state transition")
	ErrRecordArchived             = errors.New("record is archived and cannot be mutated")
	ErrDuplicateObjectID          = errors.New("object identifier already exists in registry")
)

// ValidateDigest ensures the digest is a 64-character lowercase SHA-256 hexadecimal string.
func ValidateDigest(digest string) error {
	if len(digest) != 64 {
		return ErrInvalidDigest
	}
	for i := 0; i < len(digest); i++ {
		c := digest[i]
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return ErrInvalidDigest
		}
	}
	return nil
}

// OriginalRecord represents an authoritative, immutable original evidence record.
type OriginalRecord struct {
	objectID     string
	tenantID     string
	mediaType    string
	sizeBytes    int64
	sha256Digest string
	state        RecordLifecycleState
	createdAt    time.Time
}

// NewOriginalRecord creates and validates an OriginalRecord in the DRAFT state.
func NewOriginalRecord(objectID, tenantID, mediaType string, sizeBytes int64, sha256Digest string, createdAt time.Time) (OriginalRecord, error) {
	if strings.TrimSpace(objectID) == "" {
		return OriginalRecord{}, ErrBlankID
	}
	if strings.TrimSpace(tenantID) == "" {
		return OriginalRecord{}, ErrBlankTenantID
	}
	if strings.TrimSpace(mediaType) == "" {
		return OriginalRecord{}, ErrBlankMediaType
	}
	if sizeBytes <= 0 {
		return OriginalRecord{}, ErrInvalidSizeBytes
	}
	if err := ValidateDigest(sha256Digest); err != nil {
		return OriginalRecord{}, err
	}
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}

	return OriginalRecord{
		objectID:     strings.TrimSpace(objectID),
		tenantID:     strings.TrimSpace(tenantID),
		mediaType:    strings.TrimSpace(mediaType),
		sizeBytes:    sizeBytes,
		sha256Digest: sha256Digest,
		state:        StateDraft,
		createdAt:    createdAt,
	}, nil
}

func (r OriginalRecord) ObjectID() string            { return r.objectID }
func (r OriginalRecord) TenantID() string            { return r.tenantID }
func (r OriginalRecord) MediaType() string           { return r.mediaType }
func (r OriginalRecord) SizeBytes() int64            { return r.sizeBytes }
func (r OriginalRecord) SHA256Digest() string        { return r.sha256Digest }
func (r OriginalRecord) State() RecordLifecycleState { return r.state }
func (r OriginalRecord) CreatedAt() time.Time        { return r.createdAt }
func (r OriginalRecord) Kind() ObjectKind            { return KindOriginal }

// DerivedRecord represents a separately identified synthetic derived object
// (such as an annotation, redaction, thumbnail, or transformation) linked
// strictly to exactly one accepted original record.
type DerivedRecord struct {
	objectID     string
	tenantID     string
	parentID     string
	derivedKind  DerivedKind
	mediaType    string
	sizeBytes    int64
	sha256Digest string
	state        RecordLifecycleState
	createdAt    time.Time
}

// NewDerivedRecord constructs and validates a DerivedRecord in DRAFT state.
func NewDerivedRecord(objectID, tenantID, parentID string, kind DerivedKind, mediaType string, sizeBytes int64, sha256Digest string, createdAt time.Time) (DerivedRecord, error) {
	if strings.TrimSpace(objectID) == "" {
		return DerivedRecord{}, ErrBlankID
	}
	if strings.TrimSpace(tenantID) == "" {
		return DerivedRecord{}, ErrBlankTenantID
	}
	if strings.TrimSpace(parentID) == "" {
		return DerivedRecord{}, ErrBlankParentID
	}
	switch kind {
	case DerivedAnnotation, DerivedRedaction, DerivedThumbnail, DerivedTransformation:
	default:
		return DerivedRecord{}, ErrInvalidDerivedKind
	}
	if strings.TrimSpace(mediaType) == "" {
		return DerivedRecord{}, ErrBlankMediaType
	}
	if sizeBytes <= 0 {
		return DerivedRecord{}, ErrInvalidSizeBytes
	}
	if err := ValidateDigest(sha256Digest); err != nil {
		return DerivedRecord{}, err
	}
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}

	return DerivedRecord{
		objectID:     strings.TrimSpace(objectID),
		tenantID:     strings.TrimSpace(tenantID),
		parentID:     strings.TrimSpace(parentID),
		derivedKind:  kind,
		mediaType:    strings.TrimSpace(mediaType),
		sizeBytes:    sizeBytes,
		sha256Digest: sha256Digest,
		state:        StateDraft,
		createdAt:    createdAt,
	}, nil
}

func (r DerivedRecord) ObjectID() string            { return r.objectID }
func (r DerivedRecord) TenantID() string            { return r.tenantID }
func (r DerivedRecord) ParentID() string            { return r.parentID }
func (r DerivedRecord) DerivedKind() DerivedKind    { return r.derivedKind }
func (r DerivedRecord) MediaType() string           { return r.mediaType }
func (r DerivedRecord) SizeBytes() int64            { return r.sizeBytes }
func (r DerivedRecord) SHA256Digest() string        { return r.sha256Digest }
func (r DerivedRecord) State() RecordLifecycleState { return r.state }
func (r DerivedRecord) CreatedAt() time.Time        { return r.createdAt }
func (r DerivedRecord) Kind() ObjectKind            { return KindDerived }

// IntegrityLinkage represents verified integrity and parent-child provenance
// between a derived object and its accepted original.
type IntegrityLinkage struct {
	Derived  DerivedRecord  `json:"derived"`
	Original OriginalRecord `json:"original"`
}

// IntegrityRegistry provides an in-memory, thread-safe registry that governs
// the lifecycle, immutability, and derived-linkage validation for records.
type IntegrityRegistry struct {
	mu        sync.RWMutex
	originals map[string]OriginalRecord
	derived   map[string]DerivedRecord
}

// NewIntegrityRegistry initializes a new empty registry.
func NewIntegrityRegistry() *IntegrityRegistry {
	return &IntegrityRegistry{
		originals: make(map[string]OriginalRecord),
		derived:   make(map[string]DerivedRecord),
	}
}

// RegisterOriginal registers an original record.
// If the objectID already exists as an accepted original, overwrite is denied.
// If the objectID already exists as draft, it cannot be registered twice.
func (reg *IntegrityRegistry) RegisterOriginal(rec OriginalRecord) error {
	reg.mu.Lock()
	defer reg.mu.Unlock()

	if rec.objectID == "" {
		return ErrBlankID
	}

	if existing, exists := reg.originals[rec.objectID]; exists {
		if existing.state == StateAccepted {
			return ErrOriginalOverwriteDenied
		}
		return ErrDuplicateObjectID
	}

	if _, exists := reg.derived[rec.objectID]; exists {
		return ErrDuplicateObjectID
	}

	reg.originals[rec.objectID] = rec
	return nil
}

// AcceptOriginal transitions an original record from DRAFT to ACCEPTED.
// Once ACCEPTED, the original becomes permanently immutable against overwrite or mutation.
func (reg *IntegrityRegistry) AcceptOriginal(objectID string) error {
	reg.mu.Lock()
	defer reg.mu.Unlock()

	rec, exists := reg.originals[objectID]
	if !exists {
		return ErrUnknownParent
	}

	if rec.state == StateAccepted {
		return nil // idempotent acceptance
	}
	if rec.state == StateArchived {
		return ErrRecordArchived
	}
	if rec.state != StateDraft {
		return ErrInvalidLifecycleTransition
	}

	rec.state = StateAccepted
	reg.originals[objectID] = rec
	return nil
}

// ArchiveOriginal transitions an accepted original to ARCHIVED state.
func (reg *IntegrityRegistry) ArchiveOriginal(objectID string) error {
	reg.mu.Lock()
	defer reg.mu.Unlock()

	rec, exists := reg.originals[objectID]
	if !exists {
		return ErrUnknownParent
	}
	if rec.state == StateArchived {
		return nil // idempotent
	}
	if rec.state != StateAccepted {
		return ErrInvalidLifecycleTransition
	}

	rec.state = StateArchived
	reg.originals[objectID] = rec
	return nil
}

// RegisterDerived registers a derived object linked to an original record.
// It fails closed if:
// - objectID already exists as an original (ErrOriginalAsDerivedDenied) or derived (ErrDuplicateObjectID)
// - parent does not exist (ErrUnknownParent)
// - parent is not in ACCEPTED state (ErrParentNotAccepted)
// - derived object tenant does not match parent original tenant (ErrCrossTenantLinkage)
func (reg *IntegrityRegistry) RegisterDerived(rec DerivedRecord) error {
	reg.mu.Lock()
	defer reg.mu.Unlock()

	if rec.objectID == "" {
		return ErrBlankID
	}

	// Cannot treat original as derived
	if _, exists := reg.originals[rec.objectID]; exists {
		return ErrOriginalAsDerivedDenied
	}
	if _, exists := reg.derived[rec.objectID]; exists {
		return ErrDuplicateObjectID
	}

	parent, parentExists := reg.originals[rec.parentID]
	if !parentExists {
		return ErrUnknownParent
	}

	if parent.state != StateAccepted {
		return ErrParentNotAccepted
	}

	if parent.tenantID != rec.tenantID {
		return ErrCrossTenantLinkage
	}

	reg.derived[rec.objectID] = rec
	return nil
}

// AcceptDerived transitions a derived record from DRAFT to ACCEPTED.
func (reg *IntegrityRegistry) AcceptDerived(objectID string) error {
	reg.mu.Lock()
	defer reg.mu.Unlock()

	rec, exists := reg.derived[objectID]
	if !exists {
		return ErrUnknownParent
	}

	if rec.state == StateAccepted {
		return nil
	}
	if rec.state == StateArchived {
		return ErrRecordArchived
	}
	if rec.state != StateDraft {
		return ErrInvalidLifecycleTransition
	}

	rec.state = StateAccepted
	reg.derived[objectID] = rec
	return nil
}

// VerifyIntegrityLinkage validates tenant scope, digest integrity, and parent existence
// before returning the verified IntegrityLinkage.
// It fails closed for:
// - Unknown derived object (ErrUnknownParent)
// - Tenant mismatch between caller scope and derived/parent record (ErrCrossTenantLinkage)
// - Digest mismatch between expected/calculated digest and derived record (ErrDigestMismatch)
// - Unknown parent original (ErrUnknownParent)
// - Unaccepted parent original (ErrParentNotAccepted)
func (reg *IntegrityRegistry) VerifyIntegrityLinkage(derivedID, callerTenantID, expectedDigest string) (IntegrityLinkage, error) {
	reg.mu.RLock()
	defer reg.mu.RUnlock()

	if strings.TrimSpace(callerTenantID) == "" {
		return IntegrityLinkage{}, ErrBlankTenantID
	}

	derived, exists := reg.derived[derivedID]
	if !exists {
		return IntegrityLinkage{}, ErrUnknownParent
	}

	if derived.tenantID != strings.TrimSpace(callerTenantID) {
		return IntegrityLinkage{}, ErrCrossTenantLinkage
	}

	if err := ValidateDigest(expectedDigest); err != nil {
		return IntegrityLinkage{}, err
	}

	if derived.sha256Digest != expectedDigest {
		return IntegrityLinkage{}, ErrDigestMismatch
	}

	parent, parentExists := reg.originals[derived.parentID]
	if !parentExists {
		return IntegrityLinkage{}, ErrUnknownParent
	}

	if parent.tenantID != derived.tenantID {
		return IntegrityLinkage{}, ErrCrossTenantLinkage
	}

	if parent.state != StateAccepted {
		return IntegrityLinkage{}, ErrParentNotAccepted
	}

	return IntegrityLinkage{
		Derived:  derived,
		Original: parent,
	}, nil
}
