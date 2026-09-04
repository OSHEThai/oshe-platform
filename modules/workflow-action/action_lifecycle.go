package workflowaction

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// ActionState represents the explicit lifecycle state of an action.
type ActionState string

const (
	ActionStateAssigned   ActionState = "ASSIGNED"
	ActionStateInProgress ActionState = "IN_PROGRESS"
	ActionStateInReview   ActionState = "IN_REVIEW"
	ActionStateRejected   ActionState = "REJECTED"
	ActionStateOverdue    ActionState = "OVERDUE"
	ActionStateClosed     ActionState = "CLOSED"
	ActionStateReopened   ActionState = "REOPENED"
)

var (
	ErrBlankActionID         = errors.New("action ID cannot be blank")
	ErrBlankTenantID         = errors.New("tenant ID cannot be blank")
	ErrBlankOwner            = errors.New("owner cannot be blank")
	ErrBlankReviewer         = errors.New("reviewer cannot be blank")
	ErrBlankTitle            = errors.New("title cannot be blank")
	ErrActionNotFound        = errors.New("action not found")
	ErrDuplicateActionID     = errors.New("duplicate action ID")
	ErrUnauthorizedAction    = errors.New("unauthorized action: actor lacks permission")
	ErrCrossTenantDenied     = errors.New("cross-tenant operation denied")
	ErrInvalidPrecedingState = errors.New("invalid preceding state for transition")
	ErrInsufficientEvidence  = errors.New("insufficient accepted evidence for closure or review submission")
	ErrActionClosed          = errors.New("action is closed and cannot be mutated or double-closed")
	ErrBlankReason           = errors.New("reason cannot be blank")
	ErrDuplicateEvidenceID   = errors.New("duplicate evidence attachment ID")
	ErrInvalidEvidenceDigest = errors.New("invalid evidence digest")
)

// ActionHistoryEntry is an immutable, append-only record of an action transition.
type ActionHistoryEntry struct {
	Sequence    int64       `json:"sequence"`
	FromState   ActionState `json:"from_state"`
	ToState     ActionState `json:"to_state"`
	Actor       string      `json:"actor"`
	Reason      string      `json:"reason"`
	EvidenceIDs []string    `json:"evidence_ids,omitempty"`
	Timestamp   time.Time   `json:"timestamp"`
}

// EvidenceAttachment links verified accepted evidence to an action.
type EvidenceAttachment struct {
	EvidenceID string    `json:"evidence_id"`
	TenantID   string    `json:"tenant_id"`
	Digest     string    `json:"digest"`
	AttachedBy string    `json:"attached_by"`
	AttachedAt time.Time `json:"attached_at"`
}

// ActionSnapshot is an immutable view of an Action.
type ActionSnapshot struct {
	ID                    string               `json:"id"`
	TenantID              string               `json:"tenant_id"`
	Title                 string               `json:"title"`
	Owner                 string               `json:"owner"`
	Reviewer              string               `json:"reviewer"`
	DueDate               time.Time            `json:"due_date"`
	RequiredEvidenceCount int                  `json:"required_evidence_count"`
	State                 ActionState          `json:"state"`
	Evidence              []EvidenceAttachment `json:"evidence"`
	History               []ActionHistoryEntry `json:"history"`
	ClosedAt              *time.Time           `json:"closed_at,omitempty"`
	ClosedBy              string               `json:"closed_by,omitempty"`
}

// action represents the internal mutable state of an action under ActionManager lock.
type action struct {
	id                    string
	tenantID              string
	title                 string
	owner                 string
	reviewer              string
	dueDate               time.Time
	requiredEvidenceCount int
	state                 ActionState
	evidence              []EvidenceAttachment
	evidenceIDs           map[string]bool
	history               []ActionHistoryEntry
	closedAt              *time.Time
	closedBy              string
}

func (a *action) snapshot() ActionSnapshot {
	evCopy := make([]EvidenceAttachment, len(a.evidence))
	copy(evCopy, a.evidence)

	histCopy := make([]ActionHistoryEntry, len(a.history))
	copy(histCopy, a.history)

	var closedAtCopy *time.Time
	if a.closedAt != nil {
		t := *a.closedAt
		closedAtCopy = &t
	}

	return ActionSnapshot{
		ID:                    a.id,
		TenantID:              a.tenantID,
		Title:                 a.title,
		Owner:                 a.owner,
		Reviewer:              a.reviewer,
		DueDate:               a.dueDate,
		RequiredEvidenceCount: a.requiredEvidenceCount,
		State:                 a.state,
		Evidence:              evCopy,
		History:               histCopy,
		ClosedAt:              closedAtCopy,
		ClosedBy:              a.closedBy,
	}
}

// CreateActionRequest holds initial action configuration.
type CreateActionRequest struct {
	ID                    string
	TenantID              string
	Title                 string
	Owner                 string
	Reviewer              string
	DueDate               time.Time
	RequiredEvidenceCount int
	Creator               string
}

// ActionManager coordinates thread-safe, authorized, deterministic action lifecycles.
type ActionManager struct {
	mu      sync.RWMutex
	actions map[string]*action
	clock   Clock
}

// NewActionManager constructs a new ActionManager with an injectable clock.
func NewActionManager(clock Clock) *ActionManager {
	if clock == nil {
		clock = time.Now
	}
	return &ActionManager{
		actions: make(map[string]*action),
		clock:   clock,
	}
}

// CreateAction initializes a new action in ASSIGNED state with chronological sequence 1.
func (m *ActionManager) CreateAction(req CreateActionRequest) (ActionSnapshot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	id := strings.TrimSpace(req.ID)
	if id == "" {
		return ActionSnapshot{}, ErrBlankActionID
	}
	if _, exists := m.actions[id]; exists {
		return ActionSnapshot{}, ErrDuplicateActionID
	}

	tenantID := strings.TrimSpace(req.TenantID)
	if tenantID == "" {
		return ActionSnapshot{}, ErrBlankTenantID
	}

	owner := strings.TrimSpace(req.Owner)
	if owner == "" {
		return ActionSnapshot{}, ErrBlankOwner
	}

	reviewer := strings.TrimSpace(req.Reviewer)
	if reviewer == "" {
		return ActionSnapshot{}, ErrBlankReviewer
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		return ActionSnapshot{}, ErrBlankTitle
	}

	creator := strings.TrimSpace(req.Creator)
	if creator == "" {
		creator = reviewer
	}

	now := m.clock().UTC()
	act := &action{
		id:                    id,
		tenantID:              tenantID,
		title:                 title,
		owner:                 owner,
		reviewer:              reviewer,
		dueDate:               req.DueDate,
		requiredEvidenceCount: req.RequiredEvidenceCount,
		state:                 ActionStateAssigned,
		evidence:              make([]EvidenceAttachment, 0),
		evidenceIDs:           make(map[string]bool),
		history: []ActionHistoryEntry{
			{
				Sequence:  1,
				FromState: "",
				ToState:   ActionStateAssigned,
				Actor:     creator,
				Reason:    "Action created and assigned",
				Timestamp: now,
			},
		},
	}

	m.actions[id] = act
	return act.snapshot(), nil
}

// GetAction retrieves an immutable copy of an action.
func (m *ActionManager) GetAction(id string) (ActionSnapshot, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	act, exists := m.actions[id]
	if !exists {
		return ActionSnapshot{}, ErrActionNotFound
	}
	return act.snapshot(), nil
}

// StartWork transitions action to IN_PROGRESS. Caller must be the assigned owner.
func (m *ActionManager) StartWork(actionID, callerIdentity string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	act, exists := m.actions[actionID]
	if !exists {
		return ErrActionNotFound
	}

	if act.state == ActionStateClosed {
		return ErrActionClosed
	}

	if strings.TrimSpace(callerIdentity) != act.owner {
		return ErrUnauthorizedAction
	}

	switch act.state {
	case ActionStateAssigned, ActionStateRejected, ActionStateReopened, ActionStateOverdue:
	case ActionStateInProgress:
		return nil // idempotent
	default:
		return fmt.Errorf("%w: cannot start work from %s", ErrInvalidPrecedingState, act.state)
	}

	now := m.clock().UTC()
	seq := int64(len(act.history) + 1)
	act.history = append(act.history, ActionHistoryEntry{
		Sequence:  seq,
		FromState: act.state,
		ToState:   ActionStateInProgress,
		Actor:     callerIdentity,
		Reason:    "Work started by owner",
		Timestamp: now,
	})
	act.state = ActionStateInProgress
	return nil
}

// AttachEvidence associates accepted evidence with an action.
// Fails closed if caller is not owner, tenant mismatches, action is closed, or duplicate evidence.
func (m *ActionManager) AttachEvidence(actionID, callerIdentity string, ev EvidenceAttachment) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	act, exists := m.actions[actionID]
	if !exists {
		return ErrActionNotFound
	}

	if act.state == ActionStateClosed {
		return ErrActionClosed
	}

	if strings.TrimSpace(callerIdentity) != act.owner {
		return ErrUnauthorizedAction
	}

	if strings.TrimSpace(ev.TenantID) != act.tenantID {
		return ErrCrossTenantDenied
	}

	evidenceID := strings.TrimSpace(ev.EvidenceID)
	if evidenceID == "" {
		return errors.New("evidence ID cannot be blank")
	}

	if act.evidenceIDs[evidenceID] {
		return ErrDuplicateEvidenceID
	}

	digest := strings.TrimSpace(ev.Digest)
	if len(digest) != 64 {
		return ErrInvalidEvidenceDigest
	}

	now := m.clock().UTC()
	attachment := EvidenceAttachment{
		EvidenceID: evidenceID,
		TenantID:   act.tenantID,
		Digest:     digest,
		AttachedBy: callerIdentity,
		AttachedAt: now,
	}

	act.evidence = append(act.evidence, attachment)
	act.evidenceIDs[evidenceID] = true

	seq := int64(len(act.history) + 1)
	act.history = append(act.history, ActionHistoryEntry{
		Sequence:    seq,
		FromState:   act.state,
		ToState:     act.state,
		Actor:       callerIdentity,
		Reason:      fmt.Sprintf("Attached evidence %s", evidenceID),
		EvidenceIDs: []string{evidenceID},
		Timestamp:   now,
	})

	return nil
}

// SubmitForReview submits the action for review.
// Requires authorized owner and requisite accepted evidence count.
func (m *ActionManager) SubmitForReview(actionID, callerIdentity, notes string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	act, exists := m.actions[actionID]
	if !exists {
		return ErrActionNotFound
	}

	if act.state == ActionStateClosed {
		return ErrActionClosed
	}

	if strings.TrimSpace(callerIdentity) != act.owner {
		return ErrUnauthorizedAction
	}

	switch act.state {
	case ActionStateAssigned, ActionStateInProgress, ActionStateRejected, ActionStateReopened, ActionStateOverdue:
	default:
		return fmt.Errorf("%w: cannot submit for review from %s", ErrInvalidPrecedingState, act.state)
	}

	if len(act.evidence) < act.requiredEvidenceCount {
		return fmt.Errorf("%w: requires %d, got %d", ErrInsufficientEvidence, act.requiredEvidenceCount, len(act.evidence))
	}

	now := m.clock().UTC()
	seq := int64(len(act.history) + 1)
	act.history = append(act.history, ActionHistoryEntry{
		Sequence:  seq,
		FromState: act.state,
		ToState:   ActionStateInReview,
		Actor:     callerIdentity,
		Reason:    notes,
		Timestamp: now,
	})
	act.state = ActionStateInReview
	return nil
}

// RejectReview rejects the action and returns it to REJECTED for corrective rework.
// Requires authorized reviewer identity.
func (m *ActionManager) RejectReview(actionID, callerIdentity, reason string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	act, exists := m.actions[actionID]
	if !exists {
		return ErrActionNotFound
	}

	if act.state == ActionStateClosed {
		return ErrActionClosed
	}

	if strings.TrimSpace(callerIdentity) != act.reviewer {
		return ErrUnauthorizedAction
	}

	if act.state != ActionStateInReview {
		return fmt.Errorf("%w: rejection requires IN_REVIEW state, got %s", ErrInvalidPrecedingState, act.state)
	}

	reason = strings.TrimSpace(reason)
	if reason == "" {
		return ErrBlankReason
	}

	now := m.clock().UTC()
	seq := int64(len(act.history) + 1)
	act.history = append(act.history, ActionHistoryEntry{
		Sequence:  seq,
		FromState: act.state,
		ToState:   ActionStateRejected,
		Actor:     callerIdentity,
		Reason:    reason,
		Timestamp: now,
	})
	act.state = ActionStateRejected
	return nil
}

// CloseAction approves and finalizes an action in the terminal CLOSED state.
// Requires:
// - authorized reviewer identity
// - valid preceding state IN_REVIEW
// - requisite accepted evidence count satisfied
// - double-closure prevention (fails closed if already CLOSED)
func (m *ActionManager) CloseAction(actionID, callerIdentity, notes string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	act, exists := m.actions[actionID]
	if !exists {
		return ErrActionNotFound
	}

	if act.state == ActionStateClosed {
		return ErrActionClosed
	}

	if strings.TrimSpace(callerIdentity) != act.reviewer {
		return ErrUnauthorizedAction
	}

	if act.state != ActionStateInReview {
		return fmt.Errorf("%w: closure requires IN_REVIEW preceding state, got %s", ErrInvalidPrecedingState, act.state)
	}

	if len(act.evidence) < act.requiredEvidenceCount {
		return fmt.Errorf("%w: closure requires %d evidence, got %d", ErrInsufficientEvidence, act.requiredEvidenceCount, len(act.evidence))
	}

	now := m.clock().UTC()
	seq := int64(len(act.history) + 1)
	act.history = append(act.history, ActionHistoryEntry{
		Sequence:  seq,
		FromState: act.state,
		ToState:   ActionStateClosed,
		Actor:     callerIdentity,
		Reason:    notes,
		Timestamp: now,
	})
	act.state = ActionStateClosed
	act.closedAt = &now
	act.closedBy = callerIdentity
	return nil
}

// ReopenAction transitions a CLOSED action to REOPENED for additional corrective actions.
// Requires authorized reviewer/authority identity and non-blank reason.
func (m *ActionManager) ReopenAction(actionID, callerIdentity, reason string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	act, exists := m.actions[actionID]
	if !exists {
		return ErrActionNotFound
	}

	if act.state != ActionStateClosed {
		return fmt.Errorf("%w: reopen requires CLOSED state, got %s", ErrInvalidPrecedingState, act.state)
	}

	if strings.TrimSpace(callerIdentity) != act.reviewer {
		return ErrUnauthorizedAction
	}

	reason = strings.TrimSpace(reason)
	if reason == "" {
		return ErrBlankReason
	}

	now := m.clock().UTC()
	seq := int64(len(act.history) + 1)
	act.history = append(act.history, ActionHistoryEntry{
		Sequence:  seq,
		FromState: act.state,
		ToState:   ActionStateReopened,
		Actor:     callerIdentity,
		Reason:    reason,
		Timestamp: now,
	})
	act.state = ActionStateReopened
	act.closedAt = nil
	act.closedBy = ""
	return nil
}

// CheckOverdue evaluates if an action has passed its due date and marks it OVERDUE.
func (m *ActionManager) CheckOverdue(actionID string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	act, exists := m.actions[actionID]
	if !exists {
		return false, ErrActionNotFound
	}

	if act.state == ActionStateClosed {
		return false, nil
	}

	now := m.clock().UTC()
	if !act.dueDate.IsZero() && now.After(act.dueDate) {
		if act.state != ActionStateOverdue {
			seq := int64(len(act.history) + 1)
			act.history = append(act.history, ActionHistoryEntry{
				Sequence:  seq,
				FromState: act.state,
				ToState:   ActionStateOverdue,
				Actor:     "system_timer",
				Reason:    fmt.Sprintf("Due date %s exceeded at %s", act.dueDate.Format(time.RFC3339), now.Format(time.RFC3339)),
				Timestamp: now,
			})
			act.state = ActionStateOverdue
		}
		return true, nil
	}

	return false, nil
}
