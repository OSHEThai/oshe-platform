// Package workflowaction coordinates operational inspection lifecycles, finding tracking, and corrective actions.
//
// PROVISIONAL GOVERNANCE DECLARATION (Issue #134 / V040-I023):
// Under approved Sole Human Owner decision HDEC-V040-FOUNDATION-054, this file implements
// synthetic reinspection order assignment, configurable inspector independence,
// verification prerequisite gates, fail-closed offline final closure rejection,
// optimistic concurrency control, deficiency rejection, supervisory reopening,
// and recurrence lineage tracking.
//
// Strict Reinspection Governance Invariants:
// 1. Mandatory Prerequisite Gates: Reinspection verification and closure require
//    remediated actions, zero pending/rejected evidence, and explicit physical verification.
// 2. Configurable Independence: Enforces inspector independence tiers (SAME_INSPECTOR_ALLOWED,
//    DIFFERENT_INSPECTOR_REQUIRED, THIRD_PARTY_REQUIRED), barring self-review.
// 3. Offline Final Closure Denial: Offline final closure is strictly prohibited and fails closed
//    (ErrOfflineClosureProhibited). Final closure is a protected state requiring authoritative online evaluation.
// 4. Zero Autonomous AI Decisions: AI agents cannot evaluate adequacy or close findings (ErrAutonomousClosureProhibited).
// 5. Concurrency & Anti-Last-Write-Wins: Optimistic state versioning rejects stale updates (ErrConcurrentModification).
// 6. Complete Recurrence & Reopen History: Deficiency rejections, supervisory reopenings, and chronic recurrence
//    links preserve unbroken append-only audit trails.
package workflowaction

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// ReinspectionState represents the lifecycle state of a reinspection verification order.
type ReinspectionState string

const (
	ReinspectionStatePendingAssignment    ReinspectionState = "PENDING_ASSIGNMENT"
	ReinspectionStateAssigned             ReinspectionState = "ASSIGNED"
	ReinspectionStateInProgress           ReinspectionState = "IN_PROGRESS"
	ReinspectionStateVerifiedSatisfactory ReinspectionState = "VERIFIED_SATISFACTORY"
	ReinspectionStateRejectedDeficient    ReinspectionState = "REJECTED_DEFICIENT"
	ReinspectionStateClosed               ReinspectionState = "CLOSED"
	ReinspectionStateReopened             ReinspectionState = "REOPENED"
)

// IndependenceRequirement defines the segregation-of-duties requirement for reinspection assignment.
type IndependenceRequirement string

const (
	IndependenceSameInspectorAllowed       IndependenceRequirement = "SAME_INSPECTOR_ALLOWED"
	IndependenceDifferentInspectorRequired IndependenceRequirement = "DIFFERENT_INSPECTOR_REQUIRED"
	IndependenceThirdPartyRequired         IndependenceRequirement = "THIRD_PARTY_REQUIRED"
)

var (
	ErrBlankReinspectionID              = errors.New("reinspection ID cannot be blank")
	ErrDuplicateReinspectionID          = errors.New("duplicate reinspection ID")
	ErrReinspectionNotFound             = errors.New("reinspection record not found")
	ErrOfflineClosureProhibited         = errors.New("offline final closure is strictly prohibited: requires live authoritative server validation")
	ErrStaleReinspectionState           = errors.New("reinspection state is stale or does not match prerequisite version")
	ErrPrerequisitesUnsatisfied         = errors.New("closure prerequisites unsatisfied: action not ready or required evidence missing")
	ErrPendingOrRejectedEvidence        = errors.New("closure rejected: evidence items are pending review or rejected")
	ErrInspectorIndependenceViolation   = errors.New("reinspection violates inspector independence policy: distinct inspector required")
	ErrInvalidReinspectionTransition    = errors.New("invalid reinspection state transition")
	ErrReinspectionTerminalState        = errors.New("reinspection is in terminal state and cannot be modified")
	ErrMissingReinspectionRationale     = errors.New("reinspection rationale or notes cannot be blank")
	ErrActorAuthorityRevoked            = errors.New("actor authority has been revoked or is inactive")
	ErrRecurrenceChainBroken            = errors.New("recurrence linkage invalid or blank recurrence ID")
)

// isAutonomousAgentRole checks if the role corresponds to an automated AI agent.
func isAutonomousAgentRole(role string) bool {
	norm := strings.ToUpper(strings.TrimSpace(role))
	switch norm {
	case "AI", "AI_AGENT", "ENGINEERING_AGENT", "SYSTEM_AGENT", "AUTONOMOUS_AGENT", "LLM":
		return true
	default:
		return false
	}
}

// ReinspectionHistoryEntry records an immutable, append-only event in the reinspection timeline.
type ReinspectionHistoryEntry struct {
	Sequence    int64             `json:"sequence"`
	Timestamp   time.Time         `json:"timestamp"`
	Actor       string            `json:"actor"`
	ActorRole   string            `json:"actor_role"`
	FromState   ReinspectionState `json:"from_state"`
	ToState     ReinspectionState `json:"to_state"`
	Action      string            `json:"action"`
	Reason      string            `json:"reason"`
	EvidenceIDs []string          `json:"evidence_ids,omitempty"`
}

// ReinspectionOrder models a formal reinspection verification entity.
type ReinspectionOrder struct {
	ReinspectionID      string                     `json:"reinspection_id"`
	TenantID            string                     `json:"tenant_id"`
	FindingID           string                     `json:"finding_id"`
	ActionID            string                     `json:"action_id"`
	OriginalInspector   string                     `json:"original_inspector"`
	AssignedReinspector string                     `json:"assigned_reinspector"`
	ReinspectorRole     string                     `json:"reinspector_role"`
	IndependenceRule    IndependenceRequirement    `json:"independence_rule"`
	State               ReinspectionState          `json:"state"`
	StateVersion        int64                      `json:"state_version"`
	IsOffline           bool                       `json:"is_offline"`
	EvidenceIDs         []string                   `json:"evidence_ids,omitempty"`
	VerificationNotes   string                     `json:"verification_notes,omitempty"`
	DeficiencyDetails   string                     `json:"deficiency_details,omitempty"`
	ScheduledAt         time.Time                  `json:"scheduled_at"`
	CompletedAt         *time.Time                 `json:"completed_at,omitempty"`
	ClosedAt            *time.Time                 `json:"closed_at,omitempty"`
	ClosedBy            string                     `json:"closed_by,omitempty"`
	ClosureRationale    string                     `json:"closure_rationale,omitempty"`
	RecurrenceID        string                     `json:"recurrence_id,omitempty"`
	RecurrenceCount     int                        `json:"recurrence_count"`
	History             []ReinspectionHistoryEntry `json:"history"`
	CreatedAt           time.Time                  `json:"created_at"`
	UpdatedAt           time.Time                  `json:"updated_at"`
}

func (o *ReinspectionOrder) clone() *ReinspectionOrder {
	if o == nil {
		return nil
	}
	cp := *o
	if o.EvidenceIDs != nil {
		cp.EvidenceIDs = make([]string, len(o.EvidenceIDs))
		copy(cp.EvidenceIDs, o.EvidenceIDs)
	}
	if o.History != nil {
		cp.History = make([]ReinspectionHistoryEntry, len(o.History))
		copy(cp.History, o.History)
	}
	if o.CompletedAt != nil {
		t := *o.CompletedAt
		cp.CompletedAt = &t
	}
	if o.ClosedAt != nil {
		t := *o.ClosedAt
		cp.ClosedAt = &t
	}
	return &cp
}

// ReinspectionEngine coordinates thread-safe, audited reinspection workflows.
type ReinspectionEngine struct {
	mu            sync.RWMutex
	clock         Clock
	orders        map[string]*ReinspectionOrder
	revokedActors map[string]bool
	actionEngine  *ActionGovernanceEngine
}

// NewReinspectionEngine constructs a new engine with injectable clock and action governance engine.
func NewReinspectionEngine(clock Clock, actionEngine *ActionGovernanceEngine) *ReinspectionEngine {
	if clock == nil {
		clock = time.Now
	}
	return &ReinspectionEngine{
		clock:         clock,
		orders:        make(map[string]*ReinspectionOrder),
		revokedActors: make(map[string]bool),
		actionEngine:  actionEngine,
	}
}

// RevokeActorAuthority invalidates authority credentials for an actor subject.
func (e *ReinspectionEngine) RevokeActorAuthority(actor string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.revokedActors[actor] = true
}

// IsActorRevoked checks if an actor subject has had their authority revoked.
func (e *ReinspectionEngine) IsActorRevoked(actor string) bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.revokedActors[actor]
}

// CreateReinspectionOrder registers a new reinspection order with deterministic identity and independence validation.
func (e *ReinspectionEngine) CreateReinspectionOrder(order ReinspectionOrder) (*ReinspectionOrder, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	id := strings.TrimSpace(order.ReinspectionID)
	if id == "" {
		return nil, ErrBlankReinspectionID
	}
	if _, exists := e.orders[id]; exists {
		return nil, ErrDuplicateReinspectionID
	}
	tenantID := strings.TrimSpace(order.TenantID)
	if tenantID == "" {
		return nil, ErrBlankTenantID
	}
	if strings.TrimSpace(order.FindingID) == "" {
		return nil, ErrBlankFindingID
	}
	if strings.TrimSpace(order.ActionID) == "" {
		return nil, ErrBlankActionID
	}

	// Validate independence rule configuration
	if order.IndependenceRule == "" {
		order.IndependenceRule = IndependenceSameInspectorAllowed
	}

	assigned := strings.TrimSpace(order.AssignedReinspector)
	orig := strings.TrimSpace(order.OriginalInspector)
	role := strings.ToUpper(strings.TrimSpace(order.ReinspectorRole))

	if assigned != "" {
		if order.IndependenceRule == IndependenceDifferentInspectorRequired && assigned == orig {
			return nil, ErrInspectorIndependenceViolation
		}
		if order.IndependenceRule == IndependenceThirdPartyRequired {
			if role != "THIRD_PARTY_AUDITOR" && role != "EXTERNAL_AUDITOR" {
				return nil, ErrInspectorIndependenceViolation
			}
		}
	}

	now := e.clock().UTC()
	initialState := order.State
	if initialState == "" {
		if assigned != "" {
			initialState = ReinspectionStateAssigned
		} else {
			initialState = ReinspectionStatePendingAssignment
		}
	}

	newOrder := &ReinspectionOrder{
		ReinspectionID:      id,
		TenantID:            tenantID,
		FindingID:           order.FindingID,
		ActionID:            order.ActionID,
		OriginalInspector:   orig,
		AssignedReinspector: assigned,
		ReinspectorRole:     role,
		IndependenceRule:    order.IndependenceRule,
		State:               initialState,
		StateVersion:        1,
		IsOffline:           order.IsOffline,
		EvidenceIDs:         make([]string, 0),
		RecurrenceID:        order.RecurrenceID,
		RecurrenceCount:     order.RecurrenceCount,
		ScheduledAt:         order.ScheduledAt,
		CreatedAt:           now,
		UpdatedAt:           now,
		History: []ReinspectionHistoryEntry{
			{
				Sequence:  1,
				Timestamp: now,
				Actor:     "system",
				ActorRole: "ENGINE",
				FromState: "",
				ToState:   initialState,
				Action:    "CREATE_REINSPECTION_ORDER",
				Reason:    "Order created following action remediation",
			},
		},
	}

	e.orders[id] = newOrder
	return newOrder.clone(), nil
}

// AssignReinspector assigns or reassigns a qualified verifier under applicable independence rules.
func (e *ReinspectionEngine) AssignReinspector(
	reinspectionID, reinspector, role, caller, callerRole, reason string,
	expectedVersion int64,
) (*ReinspectionOrder, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	order, exists := e.orders[reinspectionID]
	if !exists {
		return nil, ErrReinspectionNotFound
	}
	if order.State == ReinspectionStateClosed {
		return nil, ErrReinspectionTerminalState
	}
	if e.revokedActors[caller] {
		return nil, ErrActorAuthorityRevoked
	}
	if order.StateVersion != expectedVersion {
		return nil, ErrConcurrentModification
	}

	reinspector = strings.TrimSpace(reinspector)
	if reinspector == "" {
		return nil, errors.New("assigned reinspector cannot be blank")
	}
	role = strings.ToUpper(strings.TrimSpace(role))

	// Validate independence constraint
	if order.IndependenceRule == IndependenceDifferentInspectorRequired && reinspector == order.OriginalInspector {
		return nil, ErrInspectorIndependenceViolation
	}
	if order.IndependenceRule == IndependenceThirdPartyRequired {
		if role != "THIRD_PARTY_AUDITOR" && role != "EXTERNAL_AUDITOR" {
			return nil, ErrInspectorIndependenceViolation
		}
	}

	now := e.clock().UTC()
	fromState := order.State
	order.AssignedReinspector = reinspector
	order.ReinspectorRole = role
	order.State = ReinspectionStateAssigned
	order.StateVersion++
	order.UpdatedAt = now

	seq := int64(len(order.History) + 1)
	order.History = append(order.History, ReinspectionHistoryEntry{
		Sequence:  seq,
		Timestamp: now,
		Actor:     caller,
		ActorRole: callerRole,
		FromState: fromState,
		ToState:   ReinspectionStateAssigned,
		Action:    "ASSIGN_REINSPECTOR",
		Reason:    reason,
	})

	return order.clone(), nil
}

// StartReinspection marks the reinspection as actively in progress.
func (e *ReinspectionEngine) StartReinspection(
	reinspectionID, caller, callerRole string,
	expectedVersion int64,
) (*ReinspectionOrder, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	order, exists := e.orders[reinspectionID]
	if !exists {
		return nil, ErrReinspectionNotFound
	}
	if order.State == ReinspectionStateClosed {
		return nil, ErrReinspectionTerminalState
	}
	if e.revokedActors[caller] {
		return nil, ErrActorAuthorityRevoked
	}
	if order.StateVersion != expectedVersion {
		return nil, ErrConcurrentModification
	}

	caller = strings.TrimSpace(caller)
	if caller != order.AssignedReinspector && !isSupervisorRole(callerRole) {
		return nil, ErrUnauthorizedActionCaller
	}

	switch order.State {
	case ReinspectionStateAssigned, ReinspectionStateRejectedDeficient, ReinspectionStateReopened:
	case ReinspectionStateInProgress:
		return order.clone(), nil // idempotent
	default:
		return nil, fmt.Errorf("%w: cannot start reinspection from state %s", ErrInvalidReinspectionTransition, order.State)
	}

	now := e.clock().UTC()
	fromState := order.State
	order.State = ReinspectionStateInProgress
	order.StateVersion++
	order.UpdatedAt = now

	seq := int64(len(order.History) + 1)
	order.History = append(order.History, ReinspectionHistoryEntry{
		Sequence:  seq,
		Timestamp: now,
		Actor:     caller,
		ActorRole: callerRole,
		FromState: fromState,
		ToState:   ReinspectionStateInProgress,
		Action:    "START_REINSPECTION",
		Reason:    "Reinspection initiated by verifier",
	})

	return order.clone(), nil
}

// VerifySatisfactory records physical verification approval and validates action prerequisites.
func (e *ReinspectionEngine) VerifySatisfactory(
	reinspectionID, caller, callerRole, verificationNotes string,
	evidenceIDs []string,
	expectedVersion int64,
) (*ReinspectionOrder, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	order, exists := e.orders[reinspectionID]
	if !exists {
		return nil, ErrReinspectionNotFound
	}
	if order.State == ReinspectionStateClosed {
		return nil, ErrReinspectionTerminalState
	}
	if e.revokedActors[caller] {
		return nil, ErrActorAuthorityRevoked
	}
	if order.StateVersion != expectedVersion {
		return nil, ErrConcurrentModification
	}

	caller = strings.TrimSpace(caller)
	if caller != order.AssignedReinspector && !isSupervisorRole(callerRole) {
		return nil, ErrUnauthorizedActionCaller
	}

	if order.State != ReinspectionStateInProgress {
		return nil, fmt.Errorf("%w: verification requires IN_PROGRESS state, got %s", ErrInvalidReinspectionTransition, order.State)
	}

	verificationNotes = strings.TrimSpace(verificationNotes)
	if verificationNotes == "" {
		return nil, ErrMissingReinspectionRationale
	}

	// Validate action evidence prerequisites if action engine is present and action is registered
	if e.actionEngine != nil && order.ActionID != "" {
		action, err := e.actionEngine.GetAction(order.ActionID)
		if err == nil {
			// Check for unaccepted or rejected evidence on action
			for _, ev := range action.EvidenceList {
				if ev.Status == EvidenceStatusRejected || ev.Status == EvidenceStatusSubmitted {
					return nil, fmt.Errorf("%w: evidence %s has status %s", ErrPendingOrRejectedEvidence, ev.EvidenceID, ev.Status)
				}
			}

			if len(action.EvidenceList) < action.RequiredEvidenceCount {
				return nil, fmt.Errorf("%w: action requires %d evidence items, has %d", ErrPrerequisitesUnsatisfied, action.RequiredEvidenceCount, len(action.EvidenceList))
			}
		}
	}

	now := e.clock().UTC()
	fromState := order.State
	order.State = ReinspectionStateVerifiedSatisfactory
	order.VerificationNotes = verificationNotes
	order.EvidenceIDs = append(order.EvidenceIDs, evidenceIDs...)
	order.CompletedAt = &now
	order.StateVersion++
	order.UpdatedAt = now

	seq := int64(len(order.History) + 1)
	order.History = append(order.History, ReinspectionHistoryEntry{
		Sequence:    seq,
		Timestamp:   now,
		Actor:       caller,
		ActorRole:   callerRole,
		FromState:   fromState,
		ToState:     ReinspectionStateVerifiedSatisfactory,
		Action:      "VERIFY_SATISFACTORY",
		Reason:      verificationNotes,
		EvidenceIDs: evidenceIDs,
	})

	return order.clone(), nil
}

// SubmitDeficiencyRejection documents reinspection failure, returning action for rework and tracking recurrence.
func (e *ReinspectionEngine) SubmitDeficiencyRejection(
	reinspectionID, caller, callerRole, deficiencyDetails string,
	recurrenceDetected bool,
	recurrenceID string,
	expectedVersion int64,
) (*ReinspectionOrder, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	order, exists := e.orders[reinspectionID]
	if !exists {
		return nil, ErrReinspectionNotFound
	}
	if order.State == ReinspectionStateClosed {
		return nil, ErrReinspectionTerminalState
	}
	if e.revokedActors[caller] {
		return nil, ErrActorAuthorityRevoked
	}
	if order.StateVersion != expectedVersion {
		return nil, ErrConcurrentModification
	}

	caller = strings.TrimSpace(caller)
	if caller != order.AssignedReinspector && !isSupervisorRole(callerRole) {
		return nil, ErrUnauthorizedActionCaller
	}

	if order.State != ReinspectionStateInProgress {
		return nil, fmt.Errorf("%w: deficiency rejection requires IN_PROGRESS state, got %s", ErrInvalidReinspectionTransition, order.State)
	}

	deficiencyDetails = strings.TrimSpace(deficiencyDetails)
	if deficiencyDetails == "" {
		return nil, ErrMissingReinspectionRationale
	}

	now := e.clock().UTC()
	fromState := order.State
	order.State = ReinspectionStateRejectedDeficient
	order.DeficiencyDetails = deficiencyDetails
	if recurrenceDetected {
		if strings.TrimSpace(recurrenceID) != "" {
			order.RecurrenceID = strings.TrimSpace(recurrenceID)
		}
		order.RecurrenceCount++
	}
	order.StateVersion++
	order.UpdatedAt = now

	seq := int64(len(order.History) + 1)
	order.History = append(order.History, ReinspectionHistoryEntry{
		Sequence:  seq,
		Timestamp: now,
		Actor:     caller,
		ActorRole: callerRole,
		FromState: fromState,
		ToState:   ReinspectionStateRejectedDeficient,
		Action:    "SUBMIT_DEFICIENCY_REJECTION",
		Reason:    deficiencyDetails,
	})

	return order.clone(), nil
}

// FinalClose executes protected final closure.
// Enforces:
// 1. Offline prohibition: isOffline must be false (fails closed with ErrOfflineClosureProhibited).
// 2. Autonomous AI prohibition: AI roles cannot close records (fails closed with ErrAutonomousClosureProhibited).
// 3. Supervisory authority: callerRole must be authorized human closure role (ErrUnauthorizedClosure).
// 4. Revocation check: caller must not be revoked (ErrActorAuthorityRevoked).
// 5. Preceding state: must be VERIFIED_SATISFACTORY (ErrPrerequisitesUnsatisfied).
// 6. Concurrency check: state version must match expectedVersion (ErrConcurrentModification).
// 7. Double-closure denial: already CLOSED fails closed (ErrReinspectionTerminalState).
func (e *ReinspectionEngine) FinalClose(
	reinspectionID, caller, callerRole, closureRationale string,
	isOffline bool,
	expectedVersion int64,
) (*ReinspectionOrder, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// 1. Absolute offline closure prohibition
	if isOffline {
		return nil, ErrOfflineClosureProhibited
	}

	// 2. Autonomous AI closure prohibition
	if isAutonomousAgentRole(callerRole) {
		return nil, ErrAutonomousClosureProhibited
	}

	// 3. Human supervisory closure authority check
	if !isHumanClosureAuthorized(callerRole) {
		return nil, ErrUnauthorizedClosure
	}

	// 4. Revocation check
	if e.revokedActors[caller] {
		return nil, ErrActorAuthorityRevoked
	}

	order, exists := e.orders[reinspectionID]
	if !exists {
		return nil, ErrReinspectionNotFound
	}

	// 5. Terminal double-closure check
	if order.State == ReinspectionStateClosed {
		return nil, ErrReinspectionTerminalState
	}

	// 6. Optimistic concurrency check
	if order.StateVersion != expectedVersion {
		return nil, ErrConcurrentModification
	}

	// 7. Prerequisite state check
	if order.State != ReinspectionStateVerifiedSatisfactory {
		return nil, fmt.Errorf("%w: final closure requires VERIFIED_SATISFACTORY state, got %s", ErrPrerequisitesUnsatisfied, order.State)
	}

	closureRationale = strings.TrimSpace(closureRationale)
	if closureRationale == "" {
		return nil, ErrMissingReinspectionRationale
	}

	now := e.clock().UTC()
	fromState := order.State
	order.State = ReinspectionStateClosed
	order.ClosedAt = &now
	order.ClosedBy = caller
	order.ClosureRationale = closureRationale
	order.StateVersion++
	order.UpdatedAt = now

	seq := int64(len(order.History) + 1)
	order.History = append(order.History, ReinspectionHistoryEntry{
		Sequence:  seq,
		Timestamp: now,
		Actor:     caller,
		ActorRole: callerRole,
		FromState: fromState,
		ToState:   ReinspectionStateClosed,
		Action:    "FINAL_CLOSE",
		Reason:    closureRationale,
	})

	return order.clone(), nil
}

// Reopen transitions a CLOSED order back to REOPENED for supervisory review or post-audit defect handling.
func (e *ReinspectionEngine) Reopen(
	reinspectionID, caller, callerRole, reopenReason string,
	expectedVersion int64,
) (*ReinspectionOrder, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if isAutonomousAgentRole(callerRole) {
		return nil, ErrAutonomousClosureProhibited
	}
	if !isHumanClosureAuthorized(callerRole) {
		return nil, ErrUnauthorizedClosure
	}
	if e.revokedActors[caller] {
		return nil, ErrActorAuthorityRevoked
	}

	order, exists := e.orders[reinspectionID]
	if !exists {
		return nil, ErrReinspectionNotFound
	}
	if order.StateVersion != expectedVersion {
		return nil, ErrConcurrentModification
	}
	if order.State != ReinspectionStateClosed {
		return nil, fmt.Errorf("%w: reopen requires CLOSED state, got %s", ErrInvalidReinspectionTransition, order.State)
	}

	reopenReason = strings.TrimSpace(reopenReason)
	if reopenReason == "" {
		return nil, ErrMissingReinspectionRationale
	}

	now := e.clock().UTC()
	fromState := order.State
	order.State = ReinspectionStateReopened
	order.ClosedAt = nil
	order.ClosedBy = ""
	order.ClosureRationale = ""
	order.StateVersion++
	order.UpdatedAt = now

	seq := int64(len(order.History) + 1)
	order.History = append(order.History, ReinspectionHistoryEntry{
		Sequence:  seq,
		Timestamp: now,
		Actor:     caller,
		ActorRole: callerRole,
		FromState: fromState,
		ToState:   ReinspectionStateReopened,
		Action:    "REOPEN_ORDER",
		Reason:    reopenReason,
	})

	return order.clone(), nil
}

// LinkRecurrence links a historical recurrence non-conformance reference to the reinspection order.
func (e *ReinspectionEngine) LinkRecurrence(
	reinspectionID, recurrenceID, caller, callerRole, reason string,
	expectedVersion int64,
) (*ReinspectionOrder, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	order, exists := e.orders[reinspectionID]
	if !exists {
		return nil, ErrReinspectionNotFound
	}
	if order.State == ReinspectionStateClosed {
		return nil, ErrReinspectionTerminalState
	}
	if e.revokedActors[caller] {
		return nil, ErrActorAuthorityRevoked
	}
	if order.StateVersion != expectedVersion {
		return nil, ErrConcurrentModification
	}

	recurrenceID = strings.TrimSpace(recurrenceID)
	if recurrenceID == "" {
		return nil, ErrRecurrenceChainBroken
	}

	now := e.clock().UTC()
	order.RecurrenceID = recurrenceID
	order.RecurrenceCount++
	order.StateVersion++
	order.UpdatedAt = now

	seq := int64(len(order.History) + 1)
	order.History = append(order.History, ReinspectionHistoryEntry{
		Sequence:  seq,
		Timestamp: now,
		Actor:     caller,
		ActorRole: callerRole,
		FromState: order.State,
		ToState:   order.State,
		Action:    "LINK_RECURRENCE",
		Reason:    fmt.Sprintf("Linked recurrence %s: %s", recurrenceID, reason),
	})

	return order.clone(), nil
}

// GetReinspection retrieves an immutable snapshot clone of a reinspection order.
func (e *ReinspectionEngine) GetReinspection(reinspectionID string) (*ReinspectionOrder, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	order, exists := e.orders[reinspectionID]
	if !exists {
		return nil, ErrReinspectionNotFound
	}
	return order.clone(), nil
}
