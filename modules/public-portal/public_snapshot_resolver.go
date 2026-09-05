// Package publicportal provides local in-memory public snapshot resolution,
// safety shielding, and non-leaking fail-closed access controls for OSHE Platform.
//
// PROVISIONAL GOVERNANCE DECLARATION (H030-006, H030-007, H030-008, Issue #99):
// Under approved Sole Human Owner decisions, this file implements a standalone local
// in-memory public snapshot resolver serving only approved, effective, unexpired,
// unwithdrawn, tenant-scoped sanitized snapshots.
//
// Strict Safety & Shielding Invariants:
// 1. Fail-Closed Non-Leaking Denial: Requests with guessed identifiers, wrong tenants,
//    draft, staged, unapproved, expired, withdrawn, or superseded statuses fail closed
//    with non-leaking denial categories, preventing resource existence reconnaissance.
// 2. Operational Query Prohibition: Any attempt to execute live transactional database
//    queries or bypass snapshot resolution fails closed immediately (ErrLiveQueryProhibited).
// 3. Search Engine & Cache Shielding: Every public response mandates HTTP headers:
//    - X-Robots-Tag: noindex, nofollow, noarchive
//    - Content-Security-Policy: default-src 'self'
//    - Cache-Control: private, no-cache, no-store
// 4. Data Minimization & Non-Authority Disclaimer: Public payloads strictly omit internal
//    database autoincrements, passwords, tokens, and PII, and embed the mandatory
//    DERIVED_OUTPUT_NON_AUTHORITY notice.
// 5. Zero External Route/CDN/Provider Activation: Operates purely in-memory on local fixtures.
//    No live DNS routes, CDN edge caches, or cloud object stores are connected or claimed.
package publicportal

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// DefaultNonAuthorityNotice defines the mandatory disclaimer embedded in public snapshot presentations.
const DefaultNonAuthorityNotice = "DERIVED_OUTPUT_NON_AUTHORITY: This public snapshot is an immutable, sanitized presentation projection derived from approved records. It conveys zero operational authority and cannot be modified."

// SnapshotLifecycleStatus classifies the publication lifecycle state.
type SnapshotLifecycleStatus string

const (
	StatusDraft               SnapshotLifecycleStatus = "DRAFT"
	StatusStaged              SnapshotLifecycleStatus = "STAGED"
	StatusApproved            SnapshotLifecycleStatus = "APPROVED"
	StatusPublishedImmutable  SnapshotLifecycleStatus = "PUBLISHED_IMMUTABLE"
	StatusWithdrawn           SnapshotLifecycleStatus = "WITHDRAWN"
	StatusSuperseded          SnapshotLifecycleStatus = "SUPERSEDED"
)

// SnapshotType classifies the public content category.
type SnapshotType string

const (
	SnapshotTypeInspection SnapshotType = "INSPECTION"
	SnapshotTypeMetric     SnapshotType = "METRIC"
)

// ResolutionDenial defines stable, non-leaking denial reasons.
type ResolutionDenial string

const (
	DenialNone                     ResolutionDenial = "NONE"
	DenialNotFound                 ResolutionDenial = "NOT_FOUND"
	DenialExpired                  ResolutionDenial = "EXPIRED"
	DenialOperationalQueryBlocked  ResolutionDenial = "OPERATIONAL_QUERY_BLOCKED"
	DenialInvalidRequest           ResolutionDenial = "INVALID_REQUEST"
)

var (
	// ErrLiveQueryProhibited indicates an attempt to query operational transactional records directly.
	ErrLiveQueryProhibited = errors.New("live transactional database queries are strictly prohibited through public snapshot resolver")
	// ErrSnapshotNotFound indicates the snapshot is missing, wrong-tenant, draft, or not published.
	ErrSnapshotNotFound = errors.New("public snapshot not found or not published")
	// ErrSnapshotExpired indicates the public snapshot effectiveness window has expired.
	ErrSnapshotExpired = errors.New("public snapshot has expired")
	// ErrBlankTenantID indicates missing tenant identifier.
	ErrBlankTenantID = errors.New("tenant ID must not be blank")
	// ErrBlankSnapshotID indicates missing snapshot identifier.
	ErrBlankSnapshotID = errors.New("snapshot ID must not be blank")
	// ErrDuplicateSnapshotID indicates duplicate registration collision in memory.
	ErrDuplicateSnapshotID = errors.New("snapshot ID already registered for tenant")
)

// PublicSnapshot represents an immutable, sanitized public view snapshot.
type PublicSnapshot struct {
	SnapshotID         string                  `json:"snapshot_id"`
	TenantID           string                  `json:"tenant_id"`
	SnapshotType       SnapshotType            `json:"snapshot_type"`
	Version            string                  `json:"version"`
	SourceDataDigest   string                  `json:"source_data_digest"`
	NonAuthorityNotice string                  `json:"non_authority_notice"`
	ApprovedBy         string                  `json:"approved_by"`
	ApprovedAt         time.Time               `json:"approved_at"`
	EffectiveFrom      time.Time               `json:"effective_from"`
	EffectiveTo        time.Time               `json:"effective_to,omitempty"`
	Status             SnapshotLifecycleStatus `json:"status"`
	PayloadTitle       string                  `json:"payload_title"`
	PayloadSummary     string                  `json:"payload_summary"`
}

// PublicResolveRequest specifies query parameters for public snapshot resolution.
type PublicResolveRequest struct {
	TenantID           string
	SnapshotID         string
	RequestedAt        time.Time
	IsOperationalQuery bool // If true, caller is attempting live transactional table query
}

// PublicResolveResult encapsulates the outcome of a snapshot resolution request.
type PublicResolveResult struct {
	Success          bool              `json:"success"`
	Snapshot         *PublicSnapshot   `json:"snapshot,omitempty"`
	DenialReason     ResolutionDenial  `json:"denial_reason"`
	ShieldingHeaders map[string]string `json:"shielding_headers"`
	ErrorMessage     string            `json:"error_message,omitempty"`
}

// StandardShieldingHeaders returns the mandatory HTTP security headers for public portal responses.
func StandardShieldingHeaders() map[string]string {
	return map[string]string{
		"X-Robots-Tag":            "noindex, nofollow, noarchive",
		"Content-Security-Policy": "default-src 'self'",
		"Cache-Control":           "private, no-cache, no-store",
	}
}

// PublicSnapshotResolver manages and resolves sanitized public snapshots strictly in memory.
type PublicSnapshotResolver struct {
	mu        sync.RWMutex
	snapshots map[string]PublicSnapshot // key: tenantID + ":" + snapshotID
}

// NewPublicSnapshotResolver initializes an empty in-memory resolver.
func NewPublicSnapshotResolver() *PublicSnapshotResolver {
	return &PublicSnapshotResolver{
		snapshots: make(map[string]PublicSnapshot),
	}
}

func makeSnapshotKey(tenantID, snapshotID string) string {
	return fmt.Sprintf("%s:%s", strings.TrimSpace(tenantID), strings.TrimSpace(snapshotID))
}

// RegisterSnapshot stores an approved snapshot in the resolver memory store.
func (r *PublicSnapshotResolver) RegisterSnapshot(s PublicSnapshot) error {
	tTenant := strings.TrimSpace(s.TenantID)
	if tTenant == "" {
		return ErrBlankTenantID
	}
	tID := strings.TrimSpace(s.SnapshotID)
	if tID == "" {
		return ErrBlankSnapshotID
	}

	key := makeSnapshotKey(tTenant, tID)

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.snapshots[key]; exists {
		return ErrDuplicateSnapshotID
	}

	// Ensure notice is embedded
	if strings.TrimSpace(s.NonAuthorityNotice) == "" {
		s.NonAuthorityNotice = DefaultNonAuthorityNotice
	}

	r.snapshots[key] = s
	return nil
}

// ResolveSnapshot evaluates a public resolution request with strict fail-closed, non-leaking behavior.
//
// Invariants enforced:
// 1. Operational queries fail closed immediately with DenialOperationalQueryBlocked.
// 2. Missing, guessed, wrong-tenant, draft, staged, approved-but-unpublished, withdrawn,
//    and superseded snapshots return DenialNotFound, strictly preventing information leakage.
// 3. Expired snapshots return DenialExpired.
// 4. All responses emit mandatory search-engine shielding headers (noindex, nofollow, noarchive).
func (r *PublicSnapshotResolver) ResolveSnapshot(req PublicResolveRequest) PublicResolveResult {
	headers := StandardShieldingHeaders()

	// 1. Operational Query Block
	if req.IsOperationalQuery {
		return PublicResolveResult{
			Success:          false,
			DenialReason:     DenialOperationalQueryBlocked,
			ShieldingHeaders: headers,
			ErrorMessage:     ErrLiveQueryProhibited.Error(),
		}
	}

	tTenant := strings.TrimSpace(req.TenantID)
	tID := strings.TrimSpace(req.SnapshotID)
	if tTenant == "" || tID == "" {
		return PublicResolveResult{
			Success:          false,
			DenialReason:     DenialInvalidRequest,
			ShieldingHeaders: headers,
			ErrorMessage:     "tenant ID and snapshot ID must be provided",
		}
	}

	key := makeSnapshotKey(tTenant, tID)

	r.mu.RLock()
	s, exists := r.snapshots[key]
	r.mu.RUnlock()

	// 2. Guessed ID / Non-existent / Wrong-Tenant isolation: Fail closed with non-leaking DenialNotFound
	if !exists {
		return PublicResolveResult{
			Success:          false,
			DenialReason:     DenialNotFound,
			ShieldingHeaders: headers,
			ErrorMessage:     ErrSnapshotNotFound.Error(),
		}
	}

	// 3. Status Check: Only PUBLISHED_IMMUTABLE is visible to public callers.
	// Draft, Staged, Approved (un-published), Withdrawn, and Superseded return DenialNotFound.
	if s.Status != StatusPublishedImmutable {
		return PublicResolveResult{
			Success:          false,
			DenialReason:     DenialNotFound,
			ShieldingHeaders: headers,
			ErrorMessage:     ErrSnapshotNotFound.Error(),
		}
	}

	// 4. Temporal Validity Check
	reqTime := req.RequestedAt
	if reqTime.IsZero() {
		reqTime = time.Now().UTC()
	}

	if reqTime.Before(s.EffectiveFrom) {
		// Not yet effective -> treat as not found to avoid premature leakage
		return PublicResolveResult{
			Success:          false,
			DenialReason:     DenialNotFound,
			ShieldingHeaders: headers,
			ErrorMessage:     ErrSnapshotNotFound.Error(),
		}
	}

	if !s.EffectiveTo.IsZero() && reqTime.After(s.EffectiveTo) {
		// Expired -> returns DenialExpired (HTTP 410 Gone equivalent)
		return PublicResolveResult{
			Success:          false,
			DenialReason:     DenialExpired,
			ShieldingHeaders: headers,
			ErrorMessage:     ErrSnapshotExpired.Error(),
		}
	}

	// 5. Successful sanitized resolution
	snapshotCopy := s
	return PublicResolveResult{
		Success:          true,
		Snapshot:         &snapshotCopy,
		DenialReason:     DenialNone,
		ShieldingHeaders: headers,
	}
}
