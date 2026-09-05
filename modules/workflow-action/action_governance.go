// Package workflowaction coordinates operational inspection lifecycles, finding tracking, and corrective actions.
//
// PROVISIONAL GOVERNANCE DECLARATION (Issue #133 / V040-I022):
// Under approved Sole Human Owner decision HDEC-V040-FOUNDATION-054, this file implements
// synthetic action ownership, due dates, extension requests, escalation handling,
// evidence submission/review/rejection, and visible responsibility history.
//
// Strict Action Governance Invariants:
// 1. Synthetic Visible Ownership & History: Every action preserves an unbroken, append-only
//    trail of ownership assignments, reassignments, and revocations. Prior owners are never erased.
// 2. Segregation of Duties (SOD): A CAPA owner cannot review or approve their own evidence,
//    due-date extensions, or escalation items (ErrSelfApprovalProhibited).
// 3. Generic Extension & Escalation Boundaries: Due-date extension and escalation workflows operate
//    generically without pre-selecting final binding corporate policy (HUMAN_OWNED_UNSELECTED).
// 4. Evidence Governance: Submitted evidence remains pending review until accepted or rejected by
//    an authorized reviewer. Rejected evidence requires corrective rework.
// 5. Concurrency & Duplicate Protection: Mutations require matching StateVersion to prevent lost
//    updates, and duplicate request/evidence IDs fail closed.
// 6. Zero External Enactment: Operates purely in-memory on local synthetic fixtures.
package workflowaction

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// ExtensionStatus represents the state of a due-date extension request.
type ExtensionStatus string

const (
	ExtensionStatusPending  ExtensionStatus = "PENDING"
	ExtensionStatusApproved ExtensionStatus = "APPROVED"
	ExtensionStatusRejected ExtensionStatus = "REJECTED"
)

// EscalationStatus represents the lifecycle state of a corrective action escalation.
type EscalationStatus string

const (
	EscalationStatusPending      EscalationStatus = "PENDING"
	EscalationStatusAcknowledged EscalationStatus = "ACKNOWLEDGED"
	EscalationStatusResolved     EscalationStatus = "RESOLVED"
)

// EvidenceReviewStatus represents the verification outcome of a submitted evidence item.
type EvidenceReviewStatus string

const (
	EvidenceStatusSubmitted EvidenceReviewStatus = "SUBMITTED"
	EvidenceStatusAccepted  EvidenceReviewStatus = "ACCEPTED"
	EvidenceStatusRejected  EvidenceReviewStatus = "REJECTED"
)

var (
	ErrUnauthorizedActionCaller = errors.New("caller is unauthorized to perform action operation")
	ErrSelfApprovalProhibited   = errors.New("segregation of duties: self-approval or self-review is strictly prohibited")
	ErrConcurrentModification   = errors.New("optimistic concurrency conflict: action state version mismatch")
	ErrDuplicateRequestID       = errors.New("request ID already exists")
	ErrInvalidDueDateExtension  = errors.New("extension due date must be after current due date")
	ErrActionTerminalState      = errors.New("action is in terminal closed state and cannot be modified")
	ErrNoActiveOwner            = errors.New("action has no active assigned owner")
	ErrEvidenceNotFound         = errors.New("evidence item not found on action")
	ErrExtensionNotFound        = errors.New("extension request not found on action")
	ErrEscalationNotFound       = errors.New("escalation request not found on action")
)

// OwnershipRecord is an immutable, append-only record documenting action custody.
type OwnershipRecord struct {
	OwnerSubject     string    `json:"owner_subject"`
	OwnerRole        string    `json:"owner_role"`
	AssignedBy       string    `json:"assigned_by"`
	AssignedAt       time.Time `json:"assigned_at"`
	RevocationReason string    `json:"revocation_reason,omitempty"`
	RevokedAt        time.Time `json:"revoked_at,omitempty"`
	IsActive         bool      `json:"is_active"`
}

// ExtensionRequest tracks an explicit proposal to extend an action's due date.
type ExtensionRequest struct {
	RequestID        string          `json:"request_id"`
	ActionID         string          `json:"action_id"`
	RequestedBy      string          `json:"requested_by"`
	RequestedAt      time.Time       `json:"requested_at"`
	CurrentDueDate   time.Time       `json:"current_due_date"`
	RequestedDueDate time.Time       `json:"requested_due_date"`
	Reason           string          `json:"reason"`
	Status           ExtensionStatus `json:"status"`
	ReviewedBy       string          `json:"reviewed_by,omitempty"`
	ReviewedAt       time.Time       `json:"reviewed_at,omitempty"`
	ReviewNotes      string          `json:"review_notes,omitempty"`
}

// EscalationRequest tracks the escalation of an overdue or high-severity corrective action.
type EscalationRequest struct {
	RequestID        string           `json:"request_id"`
	ActionID         string           `json:"action_id"`
	EscalatedBy      string           `json:"escalated_by"`
	EscalatedAt      time.Time        `json:"escalated_at"`
	TargetLevel      string           `json:"target_level"`
	Reason           string           `json:"reason"`
	Status           EscalationStatus `json:"status"`
	AcknowledgedBy   string           `json:"acknowledged_by,omitempty"`
	AcknowledgedAt   time.Time        `json:"acknowledged_at,omitempty"`
	ResolutionNotes  string           `json:"resolution_notes,omitempty"`
}

// GovernedEvidence tracks evidence submission, metadata, and formal review disposition.
type GovernedEvidence struct {
	EvidenceID     string               `json:"evidence_id"`
	ActionID       string               `json:"action_id"`
	Digest         string               `json:"digest"`
	MediaType      string               `json:"media_type"`
	SubmittedBy    string               `json:"submitted_by"`
	SubmittedAt    time.Time            `json:"submitted_at"`
	Description    string               `json:"description"`
	Status         EvidenceReviewStatus `json:"status"`
	ReviewedBy     string               `json:"reviewed_by,omitempty"`
	ReviewedAt     time.Time            `json:"reviewed_at,omitempty"`
	ReviewNotes    string               `json:"review_notes,omitempty"`
}

// GovernedAction represents an action entity under full governance controls.
type GovernedAction struct {
	ActionID               string               `json:"action_id"`
	TenantID               string               `json:"tenant_id"`
	FindingID              string               `json:"finding_id"`
	Title                  string               `json:"title"`
	State                  ActionState          `json:"state"`
	CurrentOwner           string               `json:"current_owner"`
	CurrentOwnerRole       string               `json:"current_owner_role"`
	DueDate                time.Time            `json:"due_date"`
	StateVersion           int64                `json:"state_version"`
	OwnershipHistory       []OwnershipRecord    `json:"ownership_history"`
	ExtensionRequests      []ExtensionRequest   `json:"extension_requests"`
	EscalationRequests     []EscalationRequest  `json:"escalation_requests"`
	EvidenceList           []GovernedEvidence   `json:"evidence_list"`
	RequiredEvidenceCount  int                  `json:"required_evidence_count"`
	CreatedAt              time.Time            `json:"created_at"`
	UpdatedAt              time.Time            `json:"updated_at"`
}

// ActionGovernanceEngine coordinates thread-safe, audited action governance.
type ActionGovernanceEngine struct {
	mu      sync.RWMutex
	clock   Clock
	actions map[string]*GovernedAction
}

// NewActionGovernanceEngine constructs a new ActionGovernanceEngine.
func NewActionGovernanceEngine(clock Clock) *ActionGovernanceEngine {
	if clock == nil {
		clock = time.Now
	}
	return &ActionGovernanceEngine{
		clock:   clock,
		actions: make(map[string]*GovernedAction),
	}
}

// RegisterAction adds an initial governed action to the engine.
func (e *ActionGovernanceEngine) RegisterAction(act GovernedAction) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	id := strings.TrimSpace(act.ActionID)
	if id == "" {
		return ErrBlankActionID
	}
	if _, exists := e.actions[id]; exists {
		return ErrDuplicateActionID
	}

	now := e.clock().UTC()
	if act.CreatedAt.IsZero() {
		act.CreatedAt = now
	}
	act.UpdatedAt = now
	if act.StateVersion == 0 {
		act.StateVersion = 1
	}

	// Initialize initial ownership record if owner is provided
	if act.CurrentOwner != "" && len(act.OwnershipHistory) == 0 {
		act.OwnershipHistory = append(act.OwnershipHistory, OwnershipRecord{
			OwnerSubject: act.CurrentOwner,
			OwnerRole:    act.CurrentOwnerRole,
			AssignedBy:   "SYSTEM_INITIAL",
			AssignedAt:   now,
			IsActive:     true,
		})
	}

	copyAct := act
	e.actions[id] = &copyAct
	return nil
}

// GetAction returns a snapshot copy of a governed action.
func (e *ActionGovernanceEngine) GetAction(actionID string) (GovernedAction, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	act, ok := e.actions[actionID]
	if !ok {
		return GovernedAction{}, ErrActionNotFound
	}
	return *act, nil
}

// ReassignOwner transfers action custody to a new owner, preserving prior ownership history.
func (e *ActionGovernanceEngine) ReassignOwner(
	actionID, newOwner, newRole, callerSubject, reason string,
	expectedVersion int64,
) (GovernedAction, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	act, ok := e.actions[actionID]
	if !ok {
		return GovernedAction{}, ErrActionNotFound
	}
	if act.State == ActionStateClosed {
		return GovernedAction{}, ErrActionTerminalState
	}
	if act.StateVersion != expectedVersion {
		return GovernedAction{}, ErrConcurrentModification
	}
	if strings.TrimSpace(newOwner) == "" || strings.TrimSpace(reason) == "" {
		return GovernedAction{}, ErrBlankReason
	}

	now := e.clock().UTC()

	// Deactivate current active ownership record
	for i := range act.OwnershipHistory {
		if act.OwnershipHistory[i].IsActive {
			act.OwnershipHistory[i].IsActive = false
			act.OwnershipHistory[i].RevokedAt = now
			act.OwnershipHistory[i].RevocationReason = fmt.Sprintf("REASSIGNED: %s", reason)
		}
	}

	// Append new ownership record
	act.OwnershipHistory = append(act.OwnershipHistory, OwnershipRecord{
		OwnerSubject: newOwner,
		OwnerRole:    newRole,
		AssignedBy:   callerSubject,
		AssignedAt:   now,
		IsActive:     true,
	})

	act.CurrentOwner = newOwner
	act.CurrentOwnerRole = newRole
	act.StateVersion++
	act.UpdatedAt = now

	return *act, nil
}

// RevokeOwner revokes the current owner's assignment without assigning an immediate successor.
func (e *ActionGovernanceEngine) RevokeOwner(
	actionID, callerSubject, reason string,
	expectedVersion int64,
) (GovernedAction, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	act, ok := e.actions[actionID]
	if !ok {
		return GovernedAction{}, ErrActionNotFound
	}
	if act.State == ActionStateClosed {
		return GovernedAction{}, ErrActionTerminalState
	}
	if act.StateVersion != expectedVersion {
		return GovernedAction{}, ErrConcurrentModification
	}
	if strings.TrimSpace(reason) == "" {
		return GovernedAction{}, ErrBlankReason
	}
	if act.CurrentOwner == "" {
		return GovernedAction{}, ErrNoActiveOwner
	}

	now := e.clock().UTC()

	for i := range act.OwnershipHistory {
		if act.OwnershipHistory[i].IsActive {
			act.OwnershipHistory[i].IsActive = false
			act.OwnershipHistory[i].RevokedAt = now
			act.OwnershipHistory[i].RevocationReason = fmt.Sprintf("REVOKED_BY_%s: %s", callerSubject, reason)
		}
	}

	act.CurrentOwner = ""
	act.CurrentOwnerRole = ""
	act.StateVersion++
	act.UpdatedAt = now

	return *act, nil
}

// RequestExtension records an extension request from the action owner.
func (e *ActionGovernanceEngine) RequestExtension(
	req ExtensionRequest,
	expectedVersion int64,
) (GovernedAction, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	act, ok := e.actions[req.ActionID]
	if !ok {
		return GovernedAction{}, ErrActionNotFound
	}
	if act.State == ActionStateClosed {
		return GovernedAction{}, ErrActionTerminalState
	}
	if act.StateVersion != expectedVersion {
		return GovernedAction{}, ErrConcurrentModification
	}
	if strings.TrimSpace(req.RequestID) == "" {
		return GovernedAction{}, errors.New("request ID cannot be blank")
	}
	if strings.TrimSpace(req.Reason) == "" {
		return GovernedAction{}, ErrBlankReason
	}
	if !req.RequestedDueDate.After(act.DueDate) {
		return GovernedAction{}, ErrInvalidDueDateExtension
	}

	// Verify request ID uniqueness
	for _, existing := range act.ExtensionRequests {
		if existing.RequestID == req.RequestID {
			return GovernedAction{}, ErrDuplicateRequestID
		}
	}

	now := e.clock().UTC()
	req.RequestedAt = now
	req.CurrentDueDate = act.DueDate
	req.Status = ExtensionStatusPending

	act.ExtensionRequests = append(act.ExtensionRequests, req)
	act.StateVersion++
	act.UpdatedAt = now

	return *act, nil
}

// ReviewExtension reviews a pending extension request. Enforces Segregation of Duties.
func (e *ActionGovernanceEngine) ReviewExtension(
	actionID, requestID, reviewerSubject string,
	approve bool,
	notes string,
	expectedVersion int64,
) (GovernedAction, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	act, ok := e.actions[actionID]
	if !ok {
		return GovernedAction{}, ErrActionNotFound
	}
	if act.State == ActionStateClosed {
		return GovernedAction{}, ErrActionTerminalState
	}
	if act.StateVersion != expectedVersion {
		return GovernedAction{}, ErrConcurrentModification
	}

	targetIdx := -1
	for i, ext := range act.ExtensionRequests {
		if ext.RequestID == requestID {
			targetIdx = i
			break
		}
	}
	if targetIdx == -1 {
		return GovernedAction{}, ErrExtensionNotFound
	}

	ext := &act.ExtensionRequests[targetIdx]
	if ext.Status != ExtensionStatusPending {
		return GovernedAction{}, errors.New("extension request is not in pending status")
	}

	// Segregation of Duties: Owner cannot approve own extension request
	if reviewerSubject == ext.RequestedBy || reviewerSubject == act.CurrentOwner {
		return GovernedAction{}, ErrSelfApprovalProhibited
	}

	now := e.clock().UTC()
	ext.ReviewedBy = reviewerSubject
	ext.ReviewedAt = now
	ext.ReviewNotes = notes

	if approve {
		ext.Status = ExtensionStatusApproved
		act.DueDate = ext.RequestedDueDate
	} else {
		ext.Status = ExtensionStatusRejected
	}

	act.StateVersion++
	act.UpdatedAt = now

	return *act, nil
}

// RequestEscalation records an operational escalation request for the action.
func (e *ActionGovernanceEngine) RequestEscalation(
	req EscalationRequest,
	expectedVersion int64,
) (GovernedAction, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	act, ok := e.actions[req.ActionID]
	if !ok {
		return GovernedAction{}, ErrActionNotFound
	}
	if act.State == ActionStateClosed {
		return GovernedAction{}, ErrActionTerminalState
	}
	if act.StateVersion != expectedVersion {
		return GovernedAction{}, ErrConcurrentModification
	}
	if strings.TrimSpace(req.RequestID) == "" {
		return GovernedAction{}, errors.New("request ID cannot be blank")
	}
	if strings.TrimSpace(req.Reason) == "" {
		return GovernedAction{}, ErrBlankReason
	}

	for _, existing := range act.EscalationRequests {
		if existing.RequestID == req.RequestID {
			return GovernedAction{}, ErrDuplicateRequestID
		}
	}

	now := e.clock().UTC()
	req.EscalatedAt = now
	req.Status = EscalationStatusPending

	act.EscalationRequests = append(act.EscalationRequests, req)
	act.StateVersion++
	act.UpdatedAt = now

	return *act, nil
}

// AcknowledgeEscalation acknowledges an escalation item. Enforces Segregation of Duties.
func (e *ActionGovernanceEngine) AcknowledgeEscalation(
	actionID, requestID, reviewerSubject, notes string,
	expectedVersion int64,
) (GovernedAction, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	act, ok := e.actions[actionID]
	if !ok {
		return GovernedAction{}, ErrActionNotFound
	}
	if act.State == ActionStateClosed {
		return GovernedAction{}, ErrActionTerminalState
	}
	if act.StateVersion != expectedVersion {
		return GovernedAction{}, ErrConcurrentModification
	}

	targetIdx := -1
	for i, esc := range act.EscalationRequests {
		if esc.RequestID == requestID {
			targetIdx = i
			break
		}
	}
	if targetIdx == -1 {
		return GovernedAction{}, ErrEscalationNotFound
	}

	esc := &act.EscalationRequests[targetIdx]
	if esc.Status != EscalationStatusPending {
		return GovernedAction{}, errors.New("escalation request is not in pending status")
	}

	// Segregation of Duties: Escalator cannot self-acknowledge
	if reviewerSubject == esc.EscalatedBy || reviewerSubject == act.CurrentOwner {
		return GovernedAction{}, ErrSelfApprovalProhibited
	}

	now := e.clock().UTC()
	esc.AcknowledgedBy = reviewerSubject
	esc.AcknowledgedAt = now
	esc.ResolutionNotes = notes
	esc.Status = EscalationStatusAcknowledged

	act.StateVersion++
	act.UpdatedAt = now

	return *act, nil
}

// SubmitEvidence attaches a new evidence submission from the action owner.
func (e *ActionGovernanceEngine) SubmitEvidence(
	actionID string,
	ev GovernedEvidence,
	expectedVersion int64,
) (GovernedAction, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	act, ok := e.actions[actionID]
	if !ok {
		return GovernedAction{}, ErrActionNotFound
	}
	if act.State == ActionStateClosed {
		return GovernedAction{}, ErrActionTerminalState
	}
	if act.StateVersion != expectedVersion {
		return GovernedAction{}, ErrConcurrentModification
	}
	if strings.TrimSpace(ev.EvidenceID) == "" {
		return GovernedAction{}, errors.New("evidence ID cannot be blank")
	}
	if strings.TrimSpace(ev.Digest) == "" {
		return GovernedAction{}, errors.New("evidence digest cannot be blank")
	}

	for _, existing := range act.EvidenceList {
		if existing.EvidenceID == ev.EvidenceID {
			return GovernedAction{}, ErrDuplicateEvidenceID
		}
	}

	now := e.clock().UTC()
	ev.ActionID = actionID
	ev.SubmittedAt = now
	ev.Status = EvidenceStatusSubmitted

	act.EvidenceList = append(act.EvidenceList, ev)
	act.StateVersion++
	act.UpdatedAt = now

	return *act, nil
}

// ReviewEvidence evaluates submitted evidence. Enforces Segregation of Duties.
func (e *ActionGovernanceEngine) ReviewEvidence(
	actionID, evidenceID, reviewerSubject string,
	accept bool,
	notes string,
	expectedVersion int64,
) (GovernedAction, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	act, ok := e.actions[actionID]
	if !ok {
		return GovernedAction{}, ErrActionNotFound
	}
	if act.State == ActionStateClosed {
		return GovernedAction{}, ErrActionTerminalState
	}
	if act.StateVersion != expectedVersion {
		return GovernedAction{}, ErrConcurrentModification
	}

	targetIdx := -1
	for i, ev := range act.EvidenceList {
		if ev.EvidenceID == evidenceID {
			targetIdx = i
			break
		}
	}
	if targetIdx == -1 {
		return GovernedAction{}, ErrEvidenceNotFound
	}

	ev := &act.EvidenceList[targetIdx]
	if ev.Status != EvidenceStatusSubmitted {
		return GovernedAction{}, errors.New("evidence item is not in submitted status")
	}

	// Segregation of Duties: Submitter cannot review own evidence
	if reviewerSubject == ev.SubmittedBy || reviewerSubject == act.CurrentOwner {
		return GovernedAction{}, ErrSelfApprovalProhibited
	}

	now := e.clock().UTC()
	ev.ReviewedBy = reviewerSubject
	ev.ReviewedAt = now
	ev.ReviewNotes = notes

	if accept {
		ev.Status = EvidenceStatusAccepted
	} else {
		ev.Status = EvidenceStatusRejected
	}

	act.StateVersion++
	act.UpdatedAt = now

	return *act, nil
}

// AcceptedEvidenceCount returns the number of verified accepted evidence items on an action.
func (a *GovernedAction) AcceptedEvidenceCount() int {
	count := 0
	for _, ev := range a.EvidenceList {
		if ev.Status == EvidenceStatusAccepted {
			count++
		}
	}
	return count
}
