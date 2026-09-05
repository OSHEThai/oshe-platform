// Package publicationsnapshot provides deterministic local publication lifecycle controls
// for synthetic snapshots, including authorized review/approval evidence, effective/expiry
// checks, publication, withdrawal, replacement, and supersession with historical preservation.
//
// PROVISIONAL GOVERNANCE DECLARATION (H030-003, H030-004, H030-005 / Issue #103 / V030-I030):
// Under approved Sole Human Owner decisions H030-003, H030-004, and H030-005, this file
// establishes the deterministic local lifecycle state machine and historical preservation controls.
//
// Invariants & Non-Claims:
// 1. Authorized Approval: Publication requires valid, non-stale approval evidence from an authorized role.
// 2. Temporal Validity: Publication enforces non-inverted effective and expiry windows; expired snapshots fail closed.
// 3. Immutability & Replacement: Published snapshots cannot be mutated in place; updates require formal replacement or supersession.
// 4. Withdrawal Attribution: Retraction requires an authorized caller and non-blank justification, permanently retained in audit records.
// 5. Append-Only Audit: All state transitions are captured in an append-only in-memory ledger; zero deletion permitted.
// 6. Non-Live Invariant: Operates strictly in-memory on local synthetic fixtures. Zero external publication,
//    zero public-route activation, zero database persistence, and zero operational runtime authority are claimed or enacted.
package publicationsnapshot

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// LifecycleState represents the operational publication lifecycle state.
type LifecycleState string

const (
	StateDraft        LifecycleState = "DRAFT"
	StateUnderReview  LifecycleState = "UNDER_REVIEW"
	StateApproved     LifecycleState = "APPROVED"
	StatePublished    LifecycleState = "PUBLISHED"
	StateExpired      LifecycleState = "EXPIRED"
	StateWithdrawn    LifecycleState = "WITHDRAWN"
	StateReplaced     LifecycleState = "REPLACED"
	StateSuperseded   LifecycleState = "SUPERSEDED"
)

// DefaultMaxApprovalValidity is the maximum age an approval can have before it becomes stale (7 days).
const DefaultMaxApprovalValidity = 7 * 24 * time.Hour

// MaxPublicationWindowDuration is the maximum allowable publication window duration (365 days).
const MaxPublicationWindowDuration = 365 * 24 * time.Hour

var (
	ErrUnauthorizedPublish       = errors.New("publisher role unauthorized for snapshot publication")
	ErrUnauthorizedReviewer      = errors.New("reviewer role unauthorized for snapshot approval")
	ErrStaleApproval             = errors.New("approval evidence is stale and exceeds maximum validity window")
	ErrApprovalDigestMismatch    = errors.New("snapshot content digest differs from approved evidence digest")
	ErrInvalidPublicationWindow  = errors.New("publication effective window is invalid or inverted")
	ErrNotYetEffective           = errors.New("snapshot publication is not yet effective")
	ErrSnapshotExpired           = errors.New("snapshot publication has expired")
	ErrIllegalStateTransition    = errors.New("illegal publication lifecycle state transition")
	ErrDuplicateTransition       = errors.New("duplicate lifecycle transition rejected")
	ErrMissingWithdrawalReason   = errors.New("withdrawal reason must not be blank")
	ErrMissingReplacementReason  = errors.New("replacement reason must not be blank")
	ErrBlankApprovalID           = errors.New("approval ID must not be blank")
	ErrBlankApproverSubject      = errors.New("approver subject must not be blank")
	ErrSnapshotNotUnderReview    = errors.New("snapshot must be in UNDER_REVIEW state for approval")
	ErrCannotReactivateWithdrawn = errors.New("withdrawn snapshot cannot be republished or transitioned")
)

// ApprovalEvidence encapsulates cryptographically anchored approval evidence for publication.
type ApprovalEvidence struct {
	ApprovalID      string
	ApproverSubject string
	ApproverRole    string
	ApprovedAt      time.Time
	ContentDigest   string
	DecisionHash    string
	ApprovalNotes   string
	MaxApprovalAge  time.Duration
}

// NewApprovalEvidence constructs and validates approval evidence.
func NewApprovalEvidence(approvalID, approverSubject, approverRole string, approvedAt time.Time, contentDigest, notes string, maxAge time.Duration) (ApprovalEvidence, error) {
	tAppID := strings.TrimSpace(approvalID)
	if tAppID == "" {
		return ApprovalEvidence{}, ErrBlankApprovalID
	}
	tSub := strings.TrimSpace(approverSubject)
	if tSub == "" {
		return ApprovalEvidence{}, ErrBlankApproverSubject
	}
	tRole := strings.TrimSpace(approverRole)
	if tRole == "" {
		return ApprovalEvidence{}, errors.New("approver role must not be blank")
	}
	tDigest := strings.TrimSpace(contentDigest)
	if tDigest == "" {
		return ApprovalEvidence{}, errors.New("content digest must not be blank")
	}
	if maxAge <= 0 {
		maxAge = DefaultMaxApprovalValidity
	}

	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%s|%s|%s|%d|%s|%s", tAppID, tSub, tRole, approvedAt.UnixNano(), tDigest, strings.TrimSpace(notes))))
	decisionHash := hex.EncodeToString(h.Sum(nil))

	return ApprovalEvidence{
		ApprovalID:      tAppID,
		ApproverSubject: tSub,
		ApproverRole:    tRole,
		ApprovedAt:      approvedAt,
		ContentDigest:   tDigest,
		DecisionHash:    decisionHash,
		ApprovalNotes:   strings.TrimSpace(notes),
		MaxApprovalAge:  maxAge,
	}, nil
}

// IsStale returns true if the approval timestamp exceeds its maximum validity age relative to now.
func (a ApprovalEvidence) IsStale(now time.Time) bool {
	if a.ApprovedAt.IsZero() {
		return true
	}
	if now.Before(a.ApprovedAt) {
		return false
	}
	return now.Sub(a.ApprovedAt) > a.MaxApprovalAge
}

// EffectiveWindow defines the bounded temporal window during which a snapshot is authorized for publication.
type EffectiveWindow struct {
	EffectiveFrom time.Time
	ExpiresAt     time.Time
}

// Validate asserts the effective window is non-inverted and within duration limits.
func (w EffectiveWindow) Validate() error {
	if w.EffectiveFrom.IsZero() || w.ExpiresAt.IsZero() {
		return fmt.Errorf("%w: effective and expiry timestamps must be non-zero", ErrInvalidPublicationWindow)
	}
	if w.ExpiresAt.Before(w.EffectiveFrom) || w.ExpiresAt.Equal(w.EffectiveFrom) {
		return fmt.Errorf("%w: expiry %v must be after effective date %v", ErrInvalidPublicationWindow, w.ExpiresAt, w.EffectiveFrom)
	}
	if w.ExpiresAt.Sub(w.EffectiveFrom) > MaxPublicationWindowDuration {
		return fmt.Errorf("%w: publication window exceeds maximum allowable 365 days", ErrInvalidPublicationWindow)
	}
	return nil
}

// IsEffective returns true if current time has reached or passed EffectiveFrom.
func (w EffectiveWindow) IsEffective(now time.Time) bool {
	return now.After(w.EffectiveFrom) || now.Equal(w.EffectiveFrom)
}

// IsExpired returns true if current time has reached or passed ExpiresAt.
func (w EffectiveWindow) IsExpired(now time.Time) bool {
	return now.After(w.ExpiresAt) || now.Equal(w.ExpiresAt)
}

// LifecycleManagedSnapshot represents a snapshot whose lifecycle is governed by deterministic controls.
type LifecycleManagedSnapshot struct {
	SnapshotID        string
	TenantID          string
	Version           int
	State             LifecycleState
	Approval          ApprovalEvidence
	Window            EffectiveWindow
	ContentDigest     string
	SuccessorID       string // ID of replacement or superseding snapshot
	WithdrawalReason  string
	WithdrawnBy       string
	ReplacementReason string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// EffectiveState returns the calculated lifecycle state at the given time, accounting for automatic expiration.
func (s LifecycleManagedSnapshot) EffectiveState(now time.Time) LifecycleState {
	if s.State == StatePublished {
		if s.Window.IsExpired(now) {
			return StateExpired
		}
	}
	return s.State
}

// IsActive returns true if the snapshot is currently active, effective, and non-expired.
func (s LifecycleManagedSnapshot) IsActive(now time.Time) bool {
	return s.EffectiveState(now) == StatePublished && s.Window.IsEffective(now)
}

// LifecycleAuditRecord represents an immutable chronological audit entry for a lifecycle transition.
type LifecycleAuditRecord struct {
	RecordID     string         `json:"record_id"`
	TenantID     string         `json:"tenant_id"`
	SnapshotID   string         `json:"snapshot_id"`
	Version      int            `json:"version"`
	FromState    LifecycleState `json:"from_state"`
	ToState      LifecycleState `json:"to_state"`
	ActorSubject string         `json:"actor_subject"`
	ActorRole    string         `json:"actor_role"`
	Timestamp    time.Time      `json:"timestamp"`
	Reason       string         `json:"reason"`
	ApprovalID   string         `json:"approval_id,omitempty"`
	AuditDigest  string         `json:"audit_digest"`
}

// LifecycleAuditLedger provides an append-only in-memory storage for lifecycle audit records.
type LifecycleAuditLedger struct {
	mu      sync.RWMutex
	records []LifecycleAuditRecord
}

// NewLifecycleAuditLedger creates an empty audit ledger.
func NewLifecycleAuditLedger() *LifecycleAuditLedger {
	return &LifecycleAuditLedger{
		records: make([]LifecycleAuditRecord, 0),
	}
}

// AppendRecord appends an immutable transition record to the ledger.
func (l *LifecycleAuditLedger) AppendRecord(rec LifecycleAuditRecord) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Compute tamper-evident record digest
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%s|%s|%s|%d|%s|%s|%s|%s|%d|%s|%s",
		rec.RecordID, rec.TenantID, rec.SnapshotID, rec.Version,
		rec.FromState, rec.ToState, rec.ActorSubject, rec.ActorRole,
		rec.Timestamp.UnixNano(), rec.Reason, rec.ApprovalID)))
	rec.AuditDigest = hex.EncodeToString(h.Sum(nil))

	l.records = append(l.records, rec)
	return nil
}

// GetHistory returns the chronological transition history for a given tenant and snapshot ID.
func (l *LifecycleAuditLedger) GetHistory(tenantID, snapshotID string) ([]LifecycleAuditRecord, error) {
	tTenant := strings.TrimSpace(tenantID)
	if tTenant == "" {
		return nil, ErrBlankTenantID
	}
	tSnap := strings.TrimSpace(snapshotID)
	if tSnap == "" {
		return nil, ErrBlankSnapshotID
	}

	l.mu.RLock()
	defer l.mu.RUnlock()

	var history []LifecycleAuditRecord
	for _, r := range l.records {
		if r.TenantID == tTenant && r.SnapshotID == tSnap {
			history = append(history, r)
		}
	}
	if history == nil {
		return []LifecycleAuditRecord{}, nil
	}
	return history, nil
}

// LifecycleController coordinates publication lifecycle state transitions.
type LifecycleController struct {
	mu                  sync.RWMutex
	ledger              *LifecycleAuditLedger
	snapshots           map[string]LifecycleManagedSnapshot // key: tenantID + ":" + snapshotID
	authorizedRoles     map[string]bool                     // roles permitted to publish/approve
	transitionSeq       int
}

// NewLifecycleController constructs a new in-memory LifecycleController.
func NewLifecycleController(ledger *LifecycleAuditLedger) *LifecycleController {
	if ledger == nil {
		ledger = NewLifecycleAuditLedger()
	}
	return &LifecycleController{
		ledger:    ledger,
		snapshots: make(map[string]LifecycleManagedSnapshot),
		authorizedRoles: map[string]bool{
			"AUDITOR":      true,
			"TENANT_ADMIN": true,
		},
	}
}

func makeLifecycleKey(tenantID, snapshotID string) string {
	return fmt.Sprintf("%s:%s", strings.TrimSpace(tenantID), strings.TrimSpace(snapshotID))
}

// RegisterDraft registers a new snapshot in the DRAFT lifecycle state.
func (c *LifecycleController) RegisterDraft(tenantID, snapshotID string, version int, contentDigest, creatorSubject string, now time.Time) (LifecycleManagedSnapshot, error) {
	tTenant := strings.TrimSpace(tenantID)
	if tTenant == "" {
		return LifecycleManagedSnapshot{}, ErrBlankTenantID
	}
	tSnap := strings.TrimSpace(snapshotID)
	if tSnap == "" {
		return LifecycleManagedSnapshot{}, ErrBlankSnapshotID
	}
	tDigest := strings.TrimSpace(contentDigest)
	if tDigest == "" {
		return LifecycleManagedSnapshot{}, errors.New("content digest must not be blank")
	}
	if version <= 0 {
		return LifecycleManagedSnapshot{}, ErrInvalidVersion
	}

	key := makeLifecycleKey(tTenant, tSnap)

	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.snapshots[key]; exists {
		return LifecycleManagedSnapshot{}, fmt.Errorf("snapshot %s already registered", tSnap)
	}

	snap := LifecycleManagedSnapshot{
		SnapshotID:    tSnap,
		TenantID:      tTenant,
		Version:       version,
		State:         StateDraft,
		ContentDigest: tDigest,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	c.snapshots[key] = snap
	c.transitionSeq++
	_ = c.ledger.AppendRecord(LifecycleAuditRecord{
		RecordID:     fmt.Sprintf("audit_%d", c.transitionSeq),
		TenantID:     tTenant,
		SnapshotID:   tSnap,
		Version:      version,
		FromState:    "",
		ToState:      StateDraft,
		ActorSubject: creatorSubject,
		ActorRole:    "CREATOR",
		Timestamp:    now,
		Reason:       "Initial draft registration",
	})

	return snap, nil
}

// SubmitForReview transitions a snapshot from DRAFT to UNDER_REVIEW.
func (c *LifecycleController) SubmitForReview(tenantID, snapshotID, submitterSubject, submitterRole string, now time.Time) (LifecycleManagedSnapshot, error) {
	key := makeLifecycleKey(tenantID, snapshotID)

	c.mu.Lock()
	defer c.mu.Unlock()

	snap, exists := c.snapshots[key]
	if !exists {
		return LifecycleManagedSnapshot{}, ErrSnapshotNotFound
	}
	if snap.State != StateDraft {
		return LifecycleManagedSnapshot{}, fmt.Errorf("%w: cannot submit for review from state %s", ErrIllegalStateTransition, snap.State)
	}

	snap.State = StateUnderReview
	snap.UpdatedAt = now
	c.snapshots[key] = snap

	c.transitionSeq++
	_ = c.ledger.AppendRecord(LifecycleAuditRecord{
		RecordID:     fmt.Sprintf("audit_%d", c.transitionSeq),
		TenantID:     snap.TenantID,
		SnapshotID:   snap.SnapshotID,
		Version:      snap.Version,
		FromState:    StateDraft,
		ToState:      StateUnderReview,
		ActorSubject: submitterSubject,
		ActorRole:    submitterRole,
		Timestamp:    now,
		Reason:       "Submitted for publication review",
	})

	return snap, nil
}

// Approve records formal reviewer approval and transitions snapshot from UNDER_REVIEW to APPROVED.
func (c *LifecycleController) Approve(tenantID, snapshotID string, evidence ApprovalEvidence, now time.Time) (LifecycleManagedSnapshot, error) {
	key := makeLifecycleKey(tenantID, snapshotID)

	c.mu.Lock()
	defer c.mu.Unlock()

	snap, exists := c.snapshots[key]
	if !exists {
		return LifecycleManagedSnapshot{}, ErrSnapshotNotFound
	}
	if snap.State != StateUnderReview {
		return LifecycleManagedSnapshot{}, fmt.Errorf("%w: snapshot must be in UNDER_REVIEW state, currently %s", ErrSnapshotNotUnderReview, snap.State)
	}

	// Verify reviewer role authorization
	if !c.authorizedRoles[evidence.ApproverRole] {
		return LifecycleManagedSnapshot{}, fmt.Errorf("%w: role %s cannot approve publication", ErrUnauthorizedReviewer, evidence.ApproverRole)
	}

	// Verify approval evidence freshness
	if evidence.IsStale(now) {
		return LifecycleManagedSnapshot{}, fmt.Errorf("%w: approval from %v exceeds maximum validity", ErrStaleApproval, evidence.ApprovedAt)
	}

	// Verify content digest matches approved digest
	if evidence.ContentDigest != snap.ContentDigest {
		return LifecycleManagedSnapshot{}, fmt.Errorf("%w: approved digest %s != snapshot digest %s", ErrApprovalDigestMismatch, evidence.ContentDigest, snap.ContentDigest)
	}

	snap.State = StateApproved
	snap.Approval = evidence
	snap.UpdatedAt = now
	c.snapshots[key] = snap

	c.transitionSeq++
	_ = c.ledger.AppendRecord(LifecycleAuditRecord{
		RecordID:     fmt.Sprintf("audit_%d", c.transitionSeq),
		TenantID:     snap.TenantID,
		SnapshotID:   snap.SnapshotID,
		Version:      snap.Version,
		FromState:    StateUnderReview,
		ToState:      StateApproved,
		ActorSubject: evidence.ApproverSubject,
		ActorRole:    evidence.ApproverRole,
		Timestamp:    now,
		Reason:       evidence.ApprovalNotes,
		ApprovalID:   evidence.ApprovalID,
	})

	return snap, nil
}

// Publish transitions a snapshot from APPROVED to PUBLISHED within an authorized effective window.
func (c *LifecycleController) Publish(tenantID, snapshotID, publisherSubject, publisherRole string, window EffectiveWindow, now time.Time) (LifecycleManagedSnapshot, error) {
	key := makeLifecycleKey(tenantID, snapshotID)

	c.mu.Lock()
	defer c.mu.Unlock()

	snap, exists := c.snapshots[key]
	if !exists {
		return LifecycleManagedSnapshot{}, ErrSnapshotNotFound
	}
	if snap.State == StatePublished {
		return LifecycleManagedSnapshot{}, fmt.Errorf("%w: snapshot is already published", ErrDuplicateTransition)
	}
	if snap.State != StateApproved {
		return LifecycleManagedSnapshot{}, fmt.Errorf("%w: cannot publish from state %s", ErrIllegalStateTransition, snap.State)
	}

	// Role authority check
	if !c.authorizedRoles[publisherRole] {
		return LifecycleManagedSnapshot{}, fmt.Errorf("%w: role %s not authorized to publish", ErrUnauthorizedPublish, publisherRole)
	}

	// Verify approval is still fresh at publish time
	if snap.Approval.IsStale(now) {
		return LifecycleManagedSnapshot{}, fmt.Errorf("%w: approval has become stale before publication", ErrStaleApproval)
	}

	// Validate publication window
	if err := window.Validate(); err != nil {
		return LifecycleManagedSnapshot{}, err
	}
	if window.IsExpired(now) {
		return LifecycleManagedSnapshot{}, fmt.Errorf("%w: publication window is already expired at publication time", ErrSnapshotExpired)
	}

	snap.State = StatePublished
	snap.Window = window
	snap.UpdatedAt = now
	c.snapshots[key] = snap

	c.transitionSeq++
	_ = c.ledger.AppendRecord(LifecycleAuditRecord{
		RecordID:     fmt.Sprintf("audit_%d", c.transitionSeq),
		TenantID:     snap.TenantID,
		SnapshotID:   snap.SnapshotID,
		Version:      snap.Version,
		FromState:    StateApproved,
		ToState:      StatePublished,
		ActorSubject: publisherSubject,
		ActorRole:    publisherRole,
		Timestamp:    now,
		Reason:       "Published within authorized effective window",
		ApprovalID:   snap.Approval.ApprovalID,
	})

	return snap, nil
}

// Withdraw transitions a published or approved snapshot to WITHDRAWN with mandatory justification.
func (c *LifecycleController) Withdraw(tenantID, snapshotID, withdrawerSubject, withdrawerRole, reason string, now time.Time) (LifecycleManagedSnapshot, error) {
	trimmedReason := strings.TrimSpace(reason)
	if trimmedReason == "" {
		return LifecycleManagedSnapshot{}, ErrMissingWithdrawalReason
	}
	if !c.authorizedRoles[withdrawerRole] {
		return LifecycleManagedSnapshot{}, fmt.Errorf("%w: role %s cannot withdraw snapshot", ErrUnauthorizedReviewer, withdrawerRole)
	}

	key := makeLifecycleKey(tenantID, snapshotID)

	c.mu.Lock()
	defer c.mu.Unlock()

	snap, exists := c.snapshots[key]
	if !exists {
		return LifecycleManagedSnapshot{}, ErrSnapshotNotFound
	}
	if snap.State == StateWithdrawn {
		return LifecycleManagedSnapshot{}, fmt.Errorf("%w: snapshot is already withdrawn", ErrDuplicateTransition)
	}
	if snap.State != StatePublished && snap.State != StateApproved {
		return LifecycleManagedSnapshot{}, fmt.Errorf("%w: cannot withdraw snapshot in state %s", ErrIllegalStateTransition, snap.State)
	}

	priorState := snap.State
	snap.State = StateWithdrawn
	snap.WithdrawalReason = trimmedReason
	snap.WithdrawnBy = withdrawerSubject
	snap.UpdatedAt = now
	c.snapshots[key] = snap

	c.transitionSeq++
	_ = c.ledger.AppendRecord(LifecycleAuditRecord{
		RecordID:     fmt.Sprintf("audit_%d", c.transitionSeq),
		TenantID:     snap.TenantID,
		SnapshotID:   snap.SnapshotID,
		Version:      snap.Version,
		FromState:    priorState,
		ToState:      StateWithdrawn,
		ActorSubject: withdrawerSubject,
		ActorRole:    withdrawerRole,
		Timestamp:    now,
		Reason:       trimmedReason,
	})

	return snap, nil
}

// Replace transitions a PUBLISHED snapshot to REPLACED by linking it to a successor snapshot ID.
func (c *LifecycleController) Replace(tenantID, snapshotID, successorSnapshotID, actorSubject, actorRole, reason string, now time.Time) (LifecycleManagedSnapshot, error) {
	trimmedSuccessor := strings.TrimSpace(successorSnapshotID)
	if trimmedSuccessor == "" {
		return LifecycleManagedSnapshot{}, errors.New("successor snapshot ID must not be blank")
	}
	trimmedReason := strings.TrimSpace(reason)
	if trimmedReason == "" {
		return LifecycleManagedSnapshot{}, ErrMissingReplacementReason
	}

	key := makeLifecycleKey(tenantID, snapshotID)

	c.mu.Lock()
	defer c.mu.Unlock()

	snap, exists := c.snapshots[key]
	if !exists {
		return LifecycleManagedSnapshot{}, ErrSnapshotNotFound
	}
	if snap.State != StatePublished {
		return LifecycleManagedSnapshot{}, fmt.Errorf("%w: can only replace PUBLISHED snapshots, currently %s", ErrIllegalStateTransition, snap.State)
	}

	snap.State = StateReplaced
	snap.SuccessorID = trimmedSuccessor
	snap.ReplacementReason = trimmedReason
	snap.UpdatedAt = now
	c.snapshots[key] = snap

	c.transitionSeq++
	_ = c.ledger.AppendRecord(LifecycleAuditRecord{
		RecordID:     fmt.Sprintf("audit_%d", c.transitionSeq),
		TenantID:     snap.TenantID,
		SnapshotID:   snap.SnapshotID,
		Version:      snap.Version,
		FromState:    StatePublished,
		ToState:      StateReplaced,
		ActorSubject: actorSubject,
		ActorRole:    actorRole,
		Timestamp:    now,
		Reason:       trimmedReason,
	})

	return snap, nil
}

// Supersede transitions a PUBLISHED snapshot to SUPERSEDED when a new version of the same record is published.
func (c *LifecycleController) Supersede(tenantID, snapshotID, successorSnapshotID, actorSubject, actorRole string, now time.Time) (LifecycleManagedSnapshot, error) {
	trimmedSuccessor := strings.TrimSpace(successorSnapshotID)
	if trimmedSuccessor == "" {
		return LifecycleManagedSnapshot{}, errors.New("successor snapshot ID must not be blank")
	}

	key := makeLifecycleKey(tenantID, snapshotID)

	c.mu.Lock()
	defer c.mu.Unlock()

	snap, exists := c.snapshots[key]
	if !exists {
		return LifecycleManagedSnapshot{}, ErrSnapshotNotFound
	}
	if snap.State != StatePublished {
		return LifecycleManagedSnapshot{}, fmt.Errorf("%w: can only supersede PUBLISHED snapshots, currently %s", ErrIllegalStateTransition, snap.State)
	}

	snap.State = StateSuperseded
	snap.SuccessorID = trimmedSuccessor
	snap.UpdatedAt = now
	c.snapshots[key] = snap

	c.transitionSeq++
	_ = c.ledger.AppendRecord(LifecycleAuditRecord{
		RecordID:     fmt.Sprintf("audit_%d", c.transitionSeq),
		TenantID:     snap.TenantID,
		SnapshotID:   snap.SnapshotID,
		Version:      snap.Version,
		FromState:    StatePublished,
		ToState:      StateSuperseded,
		ActorSubject: actorSubject,
		ActorRole:    actorRole,
		Timestamp:    now,
		Reason:       fmt.Sprintf("Superseded by version %s", trimmedSuccessor),
	})

	return snap, nil
}

// GetSnapshot retrieves the snapshot and evaluates its effective state at the given time.
func (c *LifecycleController) GetSnapshot(tenantID, snapshotID string, now time.Time) (LifecycleManagedSnapshot, error) {
	key := makeLifecycleKey(tenantID, snapshotID)

	c.mu.RLock()
	defer c.mu.RUnlock()

	snap, exists := c.snapshots[key]
	if !exists {
		return LifecycleManagedSnapshot{}, ErrSnapshotNotFound
	}

	// Return with dynamic effective state
	snap.State = snap.EffectiveState(now)
	return snap, nil
}

// GetActivePublishedSnapshot retrieves a snapshot strictly if it is currently active, effective, and non-expired.
func (c *LifecycleController) GetActivePublishedSnapshot(tenantID, snapshotID string, now time.Time) (LifecycleManagedSnapshot, error) {
	snap, err := c.GetSnapshot(tenantID, snapshotID, now)
	if err != nil {
		return LifecycleManagedSnapshot{}, err
	}

	if snap.State == StateExpired {
		return LifecycleManagedSnapshot{}, ErrSnapshotExpired
	}
	if snap.State == StateWithdrawn {
		return LifecycleManagedSnapshot{}, ErrSnapshotWithdrawn
	}
	if snap.State == StateReplaced || snap.State == StateSuperseded {
		return LifecycleManagedSnapshot{}, fmt.Errorf("%w: snapshot is %s and no longer active", ErrIllegalStateTransition, snap.State)
	}
	if snap.State != StatePublished {
		return LifecycleManagedSnapshot{}, fmt.Errorf("%w: snapshot is in state %s", ErrIllegalStateTransition, snap.State)
	}
	if !snap.Window.IsEffective(now) {
		return LifecycleManagedSnapshot{}, ErrNotYetEffective
	}

	return snap, nil
}
