package workflowaction

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// State represents an explicit lifecycle state in the workflow engine.
type State string

const (
	StateDraft       State = "DRAFT"
	StateInProgress  State = "IN_PROGRESS"
	StateUnderReview State = "UNDER_REVIEW"
	StateApproved    State = "APPROVED"
	StateClosed      State = "CLOSED"
	StateArchived    State = "ARCHIVED"
)

// TerminalStates defines the set of states from which no further transitions are allowed.
var TerminalStates = map[State]bool{
	StateClosed:   true,
	StateArchived: true,
}

// AllowedTransitions maps each non-terminal state to its strictly permitted next states.
// Implicit transitions or last-write-wins jumps are impossible.
var AllowedTransitions = map[State][]State{
	StateDraft:       {StateInProgress, StateArchived},
	StateInProgress:  {StateUnderReview, StateArchived},
	StateUnderReview: {StateApproved, StateInProgress, StateArchived},
	StateApproved:    {StateClosed, StateArchived},
	StateClosed:      {},
	StateArchived:    {},
}

var (
	ErrNotFound                    = errors.New("workflow instance not found")
	ErrWorkflowExists              = errors.New("workflow instance already exists")
	ErrBlankID                     = errors.New("identifier must not be blank")
	ErrBlankRequestID              = errors.New("request ID must not be blank")
	ErrTerminalState               = errors.New("workflow is in a terminal state and cannot be mutated")
	ErrInvalidTransition           = errors.New("transition is not permitted by state machine")
	ErrStaleRevision               = errors.New("optimistic concurrency conflict: stale revision")
	ErrAuthorizationDenied         = errors.New("caller authorization denied by predicate")
	ErrConflictingDuplicateRequest = errors.New("idempotency violation: conflicting request with same request ID")
	ErrInvalidRollback             = errors.New("invalid rollback: target revision must be strictly earlier and valid")
)

// AuthPredicate is a caller-provided function asserting whether a specific transition is authorized.
// Per lease instructions, no authorization decision is inferred by the engine.
type AuthPredicate func(workflowID string, from, to State) bool

// TransitionRequest holds all parameters for a deterministic state transition.
type TransitionRequest struct {
	WorkflowID       string
	RequestID        string
	CorrelationID    string
	SourceState      State
	TargetState      State
	ExpectedRevision int
	Authorizer       AuthPredicate
}

// AuditEntry is an immutable, append-only record of a completed state transition.
type AuditEntry struct {
	WorkflowID    string    `json:"workflow_id"`
	CorrelationID string    `json:"correlation_id"`
	RequestID     string    `json:"request_id"`
	PriorState    State     `json:"prior_state"`
	CurrentState  State     `json:"current_state"`
	Revision      int       `json:"revision"`
	Timestamp     time.Time `json:"timestamp"`
}

// WorkflowInstance is an immutable snapshot of an active or terminal workflow.
type WorkflowInstance struct {
	ID           string    `json:"id"`
	CurrentState State     `json:"current_state"`
	Revision     int       `json:"revision"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type idempotencyRecord struct {
	workflowID       string
	requestID        string
	sourceState      State
	targetState      State
	expectedRevision int
	resultRevision   int
	resultingState   State
	updatedAt        time.Time
}

// Clock is an injectable time provider for deterministic testing.
type Clock func() time.Time

// Engine provides an in-memory deterministic workflow state-machine with optimistic concurrency control.
type Engine struct {
	mu          sync.RWMutex
	clock       Clock
	instances   map[string]WorkflowInstance
	auditLog    []AuditEntry
	idempotency map[string]idempotencyRecord // key: workflowID + ":" + requestID
}

// NewEngine constructs an Engine instance.
func NewEngine(clock Clock) *Engine {
	if clock == nil {
		clock = time.Now
	}
	return &Engine{
		clock:       clock,
		instances:   make(map[string]WorkflowInstance),
		auditLog:    make([]AuditEntry, 0),
		idempotency: make(map[string]idempotencyRecord),
	}
}

// CreateWorkflow initializes a new workflow instance in StateDraft with Revision 1.
func (e *Engine) CreateWorkflow(id string) (WorkflowInstance, error) {
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return WorkflowInstance{}, ErrBlankID
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.instances[trimmedID]; exists {
		return WorkflowInstance{}, ErrWorkflowExists
	}

	now := e.clock()
	inst := WorkflowInstance{
		ID:           trimmedID,
		CurrentState: StateDraft,
		Revision:     1,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	e.instances[trimmedID] = inst
	return inst, nil
}

// GetWorkflow retrieves an immutable copy of a workflow instance.
func (e *Engine) GetWorkflow(id string) (WorkflowInstance, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	inst, exists := e.instances[strings.TrimSpace(id)]
	if !exists {
		return WorkflowInstance{}, ErrNotFound
	}
	return inst, nil
}

// Transition performs an optimistic-concurrency, authorized, idempotent state transition.
func (e *Engine) Transition(req TransitionRequest) (WorkflowInstance, error) {
	workflowID := strings.TrimSpace(req.WorkflowID)
	if workflowID == "" {
		return WorkflowInstance{}, ErrBlankID
	}
	requestID := strings.TrimSpace(req.RequestID)
	if requestID == "" {
		return WorkflowInstance{}, ErrBlankRequestID
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	inst, exists := e.instances[workflowID]
	if !exists {
		return WorkflowInstance{}, ErrNotFound
	}

	// 1. Idempotent retry evaluation
	idempotencyKey := workflowID + ":" + requestID
	if recorded, found := e.idempotency[idempotencyKey]; found {
		if (req.SourceState == "" || req.SourceState == recorded.sourceState) &&
			recorded.targetState == req.TargetState &&
			recorded.expectedRevision == req.ExpectedRevision {
			// Exact match duplicate retry: return current instance idempotently without double-audit
			return inst, nil
		}
		// Idempotency violation: conflicting request with same request ID
		return WorkflowInstance{}, ErrConflictingDuplicateRequest
	}

	if req.SourceState != "" && req.SourceState != inst.CurrentState {
		return WorkflowInstance{}, ErrInvalidTransition
	}

	// 2. Terminal state check: terminal states cannot be mutated
	if TerminalStates[inst.CurrentState] {
		return WorkflowInstance{}, ErrTerminalState
	}

	// 3. Optimistic concurrency check: caller-provided revision must match current revision
	if req.ExpectedRevision != inst.Revision {
		return WorkflowInstance{}, ErrStaleRevision
	}

	// 4. Allowed transition validation
	allowedNext := AllowedTransitions[inst.CurrentState]
	permitted := false
	for _, next := range allowedNext {
		if next == req.TargetState {
			permitted = true
			break
		}
	}
	if !permitted {
		return WorkflowInstance{}, ErrInvalidTransition
	}

	// 5. Explicit authorization check: caller-provided predicate must explicitly permit transition
	if req.Authorizer == nil || !req.Authorizer(workflowID, inst.CurrentState, req.TargetState) {
		return WorkflowInstance{}, ErrAuthorizationDenied
	}

	// 6. State transition application
	now := e.clock()
	priorState := inst.CurrentState
	inst.CurrentState = req.TargetState
	inst.Revision++
	inst.UpdatedAt = now
	e.instances[workflowID] = inst

	// 7. Append-only audit record retention
	e.auditLog = append(e.auditLog, AuditEntry{
		WorkflowID:    workflowID,
		CorrelationID: strings.TrimSpace(req.CorrelationID),
		RequestID:     requestID,
		PriorState:    priorState,
		CurrentState:  inst.CurrentState,
		Revision:      inst.Revision,
		Timestamp:     now,
	})

	// 8. Idempotency registration
	e.idempotency[idempotencyKey] = idempotencyRecord{
		workflowID:       workflowID,
		requestID:        requestID,
		sourceState:      priorState,
		targetState:      req.TargetState,
		expectedRevision: req.ExpectedRevision,
		resultRevision:   inst.Revision,
		resultingState:   inst.CurrentState,
		updatedAt:        now,
	}

	return inst, nil
}

// AuditHistory returns an append-only audit trail for a workflow instance.
func (e *Engine) AuditHistory(workflowID string) []AuditEntry {
	trimmedID := strings.TrimSpace(workflowID)
	e.mu.RLock()
	defer e.mu.RUnlock()

	var history []AuditEntry
	for _, entry := range e.auditLog {
		if entry.WorkflowID == trimmedID {
			history = append(history, entry)
		}
	}
	return history
}

// Checkpoint captures an immutable snapshot of a workflow instance for rollback boundaries.
func (e *Engine) Checkpoint(workflowID string) (WorkflowInstance, error) {
	return e.GetWorkflow(workflowID)
}

// Rollback restores an active workflow instance to a previously captured checkpoint.
// Fails closed if:
// - workflow does not exist (ErrNotFound)
// - workflow is in a terminal state (ErrTerminalState: CLOSED, ARCHIVED cannot be rolled back)
// - checkpoint ID does not match workflow ID (ErrInvalidRollback)
// - checkpoint revision is not strictly earlier than current revision (ErrInvalidRollback)
// Monotonically increments revision to invalidate pending stale transitions and appends an audit entry.
func (e *Engine) Rollback(workflowID string, checkpoint WorkflowInstance, reason string) (WorkflowInstance, error) {
	wID := strings.TrimSpace(workflowID)
	if wID == "" {
		return WorkflowInstance{}, ErrBlankID
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	inst, exists := e.instances[wID]
	if !exists {
		return WorkflowInstance{}, ErrNotFound
	}

	if TerminalStates[inst.CurrentState] {
		return WorkflowInstance{}, ErrTerminalState
	}

	if checkpoint.ID != inst.ID {
		return WorkflowInstance{}, fmt.Errorf("%w: checkpoint ID %q does not match workflow ID %q", ErrInvalidRollback, checkpoint.ID, inst.ID)
	}

	if checkpoint.Revision >= inst.Revision || checkpoint.Revision < 1 {
		return WorkflowInstance{}, fmt.Errorf("%w: checkpoint revision %d must be strictly earlier than current revision %d", ErrInvalidRollback, checkpoint.Revision, inst.Revision)
	}

	now := e.clock()
	priorState := inst.CurrentState
	inst.CurrentState = checkpoint.CurrentState
	inst.Revision++
	inst.UpdatedAt = now
	e.instances[wID] = inst

	e.auditLog = append(e.auditLog, AuditEntry{
		WorkflowID:    wID,
		CorrelationID: "rollback",
		RequestID:     fmt.Sprintf("rollback-rev-%d-to-%d", checkpoint.Revision, inst.Revision),
		PriorState:    priorState,
		CurrentState:  inst.CurrentState,
		Revision:      inst.Revision,
		Timestamp:     now,
	})

	return inst, nil
}
