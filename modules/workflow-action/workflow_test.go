package workflowaction_test

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	workflowaction "github.com/oshethai/oshe-platform/modules/workflow-action"
)

func newTestClock(t *testing.T) (workflowaction.Clock, func(time.Duration)) {
	t.Helper()
	var current = time.Date(2026, 9, 4, 18, 0, 0, 0, time.UTC)
	clock := func() time.Time {
		return current
	}
	advance := func(d time.Duration) {
		current = current.Add(d)
	}
	return clock, advance
}

func alwaysAllow(workflowID string, from, to workflowaction.State) bool {
	return true
}

func alwaysDeny(workflowID string, from, to workflowaction.State) bool {
	return false
}

func TestWorkflow_Creation(t *testing.T) {
	clock, _ := newTestClock(t)
	engine := workflowaction.NewEngine(clock)

	// Valid creation
	inst, err := engine.CreateWorkflow("wf-001")
	if err != nil {
		t.Fatalf("unexpected CreateWorkflow error: %v", err)
	}
	if inst.ID != "wf-001" || inst.CurrentState != workflowaction.StateDraft || inst.Revision != 1 {
		t.Errorf("unexpected instance attributes: %+v", inst)
	}

	// Blank ID
	if _, err := engine.CreateWorkflow(""); !errors.Is(err, workflowaction.ErrBlankID) {
		t.Errorf("expected ErrBlankID for empty string, got %v", err)
	}
	if _, err := engine.CreateWorkflow("   \t"); !errors.Is(err, workflowaction.ErrBlankID) {
		t.Errorf("expected ErrBlankID for whitespace string, got %v", err)
	}

	// Duplicate ID
	if _, err := engine.CreateWorkflow("wf-001"); !errors.Is(err, workflowaction.ErrWorkflowExists) {
		t.Errorf("expected ErrWorkflowExists for duplicate, got %v", err)
	}

	// GetWorkflow
	got, err := engine.GetWorkflow("wf-001")
	if err != nil {
		t.Fatalf("unexpected GetWorkflow error: %v", err)
	}
	if got.ID != "wf-001" || got.Revision != 1 {
		t.Errorf("retrieved mismatch: %+v", got)
	}

	// Get non-existent
	if _, err := engine.GetWorkflow("nonexistent"); !errors.Is(err, workflowaction.ErrNotFound) {
		t.Errorf("expected ErrNotFound for missing workflow, got %v", err)
	}
}

func TestWorkflow_ValidTransitionsAndAuditTrail(t *testing.T) {
	clock, advance := newTestClock(t)
	engine := workflowaction.NewEngine(clock)

	wfID := "wf-inspect-01"
	_, _ = engine.CreateWorkflow(wfID)

	steps := []struct {
		target   workflowaction.State
		expected int
		reqID    string
		corrID   string
	}{
		{workflowaction.StateInProgress, 1, "req-1", "corr-alpha"},
		{workflowaction.StateUnderReview, 2, "req-2", "corr-beta"},
		{workflowaction.StateApproved, 3, "req-3", "corr-gamma"},
		{workflowaction.StateClosed, 4, "req-4", "corr-delta"},
	}

	for _, step := range steps {
		advance(5 * time.Minute)
		res, err := engine.Transition(workflowaction.TransitionRequest{
			WorkflowID:       wfID,
			RequestID:        step.reqID,
			CorrelationID:    step.corrID,
			TargetState:      step.target,
			ExpectedRevision: step.expected,
			Authorizer:       alwaysAllow,
		})
		if err != nil {
			t.Fatalf("failed transition to %s: %v", step.target, err)
		}
		if res.CurrentState != step.target {
			t.Errorf("CurrentState = %v, want %v", res.CurrentState, step.target)
		}
		if res.Revision != step.expected+1 {
			t.Errorf("Revision = %d, want %d", res.Revision, step.expected+1)
		}
	}

	// Audit trail verification
	history := engine.AuditHistory(wfID)
	if len(history) != 4 {
		t.Fatalf("expected 4 audit entries, got %d", len(history))
	}

	if history[0].PriorState != workflowaction.StateDraft || history[0].CurrentState != workflowaction.StateInProgress {
		t.Errorf("history[0] state mismatch: %+v", history[0])
	}
	if history[0].CorrelationID != "corr-alpha" || history[0].RequestID != "req-1" || history[0].Revision != 2 {
		t.Errorf("history[0] metadata mismatch: %+v", history[0])
	}

	if history[3].PriorState != workflowaction.StateApproved || history[3].CurrentState != workflowaction.StateClosed {
		t.Errorf("history[3] state mismatch: %+v", history[3])
	}
}

func TestWorkflow_InvalidTransition(t *testing.T) {
	clock, _ := newTestClock(t)
	engine := workflowaction.NewEngine(clock)

	wfID := "wf-jump-01"
	_, _ = engine.CreateWorkflow(wfID)

	// DRAFT cannot jump directly to APPROVED or CLOSED
	for _, invalidTarget := range []workflowaction.State{
		workflowaction.StateApproved,
		workflowaction.StateClosed,
		workflowaction.StateUnderReview,
	} {
		_, err := engine.Transition(workflowaction.TransitionRequest{
			WorkflowID:       wfID,
			RequestID:        "req-jump",
			CorrelationID:    "corr-1",
			TargetState:      invalidTarget,
			ExpectedRevision: 1,
			Authorizer:       alwaysAllow,
		})
		if !errors.Is(err, workflowaction.ErrInvalidTransition) {
			t.Fatalf("expected ErrInvalidTransition jumping to %s, got %v", invalidTarget, err)
		}
	}

	// Verify state and revision unchanged
	inst, _ := engine.GetWorkflow(wfID)
	if inst.CurrentState != workflowaction.StateDraft || inst.Revision != 1 {
		t.Errorf("workflow state modified after invalid transition: %+v", inst)
	}
	if len(engine.AuditHistory(wfID)) != 0 {
		t.Errorf("audit history should remain empty on invalid transition")
	}
}

func TestWorkflow_StaleRevision(t *testing.T) {
	clock, _ := newTestClock(t)
	engine := workflowaction.NewEngine(clock)

	wfID := "wf-stale-01"
	_, _ = engine.CreateWorkflow(wfID)

	// Stale revision: expected 2 when current is 1
	_, err := engine.Transition(workflowaction.TransitionRequest{
		WorkflowID:       wfID,
		RequestID:        "req-stale",
		CorrelationID:    "corr-stale",
		TargetState:      workflowaction.StateInProgress,
		ExpectedRevision: 2,
		Authorizer:       alwaysAllow,
	})
	if !errors.Is(err, workflowaction.ErrStaleRevision) {
		t.Fatalf("expected ErrStaleRevision, got %v", err)
	}

	// Stale revision: expected 0 when current is 1
	_, err = engine.Transition(workflowaction.TransitionRequest{
		WorkflowID:       wfID,
		RequestID:        "req-stale-2",
		CorrelationID:    "corr-stale",
		TargetState:      workflowaction.StateInProgress,
		ExpectedRevision: 0,
		Authorizer:       alwaysAllow,
	})
	if !errors.Is(err, workflowaction.ErrStaleRevision) {
		t.Fatalf("expected ErrStaleRevision for 0, got %v", err)
	}
}

func TestWorkflow_AuthorizationDenial(t *testing.T) {
	clock, _ := newTestClock(t)
	engine := workflowaction.NewEngine(clock)

	wfID := "wf-auth-01"
	_, _ = engine.CreateWorkflow(wfID)

	// Nil authorizer
	_, err := engine.Transition(workflowaction.TransitionRequest{
		WorkflowID:       wfID,
		RequestID:        "req-auth-1",
		CorrelationID:    "corr-1",
		TargetState:      workflowaction.StateInProgress,
		ExpectedRevision: 1,
		Authorizer:       nil,
	})
	if !errors.Is(err, workflowaction.ErrAuthorizationDenied) {
		t.Fatalf("expected ErrAuthorizationDenied on nil authorizer, got %v", err)
	}

	// Explicit deny predicate
	_, err = engine.Transition(workflowaction.TransitionRequest{
		WorkflowID:       wfID,
		RequestID:        "req-auth-2",
		CorrelationID:    "corr-1",
		TargetState:      workflowaction.StateInProgress,
		ExpectedRevision: 1,
		Authorizer:       alwaysDeny,
	})
	if !errors.Is(err, workflowaction.ErrAuthorizationDenied) {
		t.Fatalf("expected ErrAuthorizationDenied on deny predicate, got %v", err)
	}

	// State and revision remain unchanged
	inst, _ := engine.GetWorkflow(wfID)
	if inst.CurrentState != workflowaction.StateDraft || inst.Revision != 1 {
		t.Errorf("workflow mutated despite auth denial")
	}
}

func TestWorkflow_DuplicateRetry_Idempotent(t *testing.T) {
	clock, _ := newTestClock(t)
	engine := workflowaction.NewEngine(clock)

	wfID := "wf-idem-01"
	_, _ = engine.CreateWorkflow(wfID)

	req := workflowaction.TransitionRequest{
		WorkflowID:       wfID,
		RequestID:        "idem-req-001",
		CorrelationID:    "corr-idem",
		TargetState:      workflowaction.StateInProgress,
		ExpectedRevision: 1,
		Authorizer:       alwaysAllow,
	}

	// First execution
	res1, err := engine.Transition(req)
	if err != nil {
		t.Fatalf("first transition failed: %v", err)
	}
	if res1.CurrentState != workflowaction.StateInProgress || res1.Revision != 2 {
		t.Fatalf("res1 mismatch: %+v", res1)
	}

	// Duplicate retry: exact match
	res2, err := engine.Transition(req)
	if err != nil {
		t.Fatalf("duplicate retry failed: %v", err)
	}
	if res2.CurrentState != workflowaction.StateInProgress || res2.Revision != 2 {
		t.Errorf("duplicate retry returned mismatched state/rev: %+v", res2)
	}

	// Audit history must contain exactly 1 entry (idempotent, no duplicate audit event)
	history := engine.AuditHistory(wfID)
	if len(history) != 1 {
		t.Errorf("expected 1 audit entry after idempotent retry, got %d", len(history))
	}
}

func TestWorkflow_ConflictingDuplicate(t *testing.T) {
	clock, _ := newTestClock(t)
	engine := workflowaction.NewEngine(clock)

	wfID := "wf-conflict-01"
	_, _ = engine.CreateWorkflow(wfID)

	// Successful initial transition
	_, err := engine.Transition(workflowaction.TransitionRequest{
		WorkflowID:       wfID,
		RequestID:        "shared-req-id",
		CorrelationID:    "corr-1",
		TargetState:      workflowaction.StateInProgress,
		ExpectedRevision: 1,
		Authorizer:       alwaysAllow,
	})
	if err != nil {
		t.Fatalf("first transition failed: %v", err)
	}

	// Conflicting duplicate with same RequestID but different TargetState
	_, err = engine.Transition(workflowaction.TransitionRequest{
		WorkflowID:       wfID,
		RequestID:        "shared-req-id",
		CorrelationID:    "corr-2",
		TargetState:      workflowaction.StateArchived,
		ExpectedRevision: 1,
		Authorizer:       alwaysAllow,
	})
	if !errors.Is(err, workflowaction.ErrConflictingDuplicateRequest) {
		t.Fatalf("expected ErrConflictingDuplicateRequest, got %v", err)
	}
}

func TestWorkflow_TerminalStateMutation(t *testing.T) {
	clock, _ := newTestClock(t)
	engine := workflowaction.NewEngine(clock)

	// 1. Closed workflow
	wfClosed := "wf-term-closed"
	_, _ = engine.CreateWorkflow(wfClosed)
	_, _ = engine.Transition(workflowaction.TransitionRequest{
		WorkflowID: wfClosed, RequestID: "r1", TargetState: workflowaction.StateInProgress, ExpectedRevision: 1, Authorizer: alwaysAllow,
	})
	_, _ = engine.Transition(workflowaction.TransitionRequest{
		WorkflowID: wfClosed, RequestID: "r2", TargetState: workflowaction.StateUnderReview, ExpectedRevision: 2, Authorizer: alwaysAllow,
	})
	_, _ = engine.Transition(workflowaction.TransitionRequest{
		WorkflowID: wfClosed, RequestID: "r3", TargetState: workflowaction.StateApproved, ExpectedRevision: 3, Authorizer: alwaysAllow,
	})
	_, _ = engine.Transition(workflowaction.TransitionRequest{
		WorkflowID: wfClosed, RequestID: "r4", TargetState: workflowaction.StateClosed, ExpectedRevision: 4, Authorizer: alwaysAllow,
	})

	// Attempt mutation from Closed
	_, err := engine.Transition(workflowaction.TransitionRequest{
		WorkflowID:       wfClosed,
		RequestID:        "r5",
		TargetState:      workflowaction.StateArchived,
		ExpectedRevision: 5,
		Authorizer:       alwaysAllow,
	})
	if !errors.Is(err, workflowaction.ErrTerminalState) {
		t.Fatalf("expected ErrTerminalState when mutating CLOSED, got %v", err)
	}

	// 2. Archived workflow
	wfArchived := "wf-term-archived"
	_, _ = engine.CreateWorkflow(wfArchived)
	_, _ = engine.Transition(workflowaction.TransitionRequest{
		WorkflowID: wfArchived, RequestID: "ra1", TargetState: workflowaction.StateArchived, ExpectedRevision: 1, Authorizer: alwaysAllow,
	})

	// Attempt mutation from Archived
	_, err = engine.Transition(workflowaction.TransitionRequest{
		WorkflowID:       wfArchived,
		RequestID:        "ra2",
		TargetState:      workflowaction.StateInProgress,
		ExpectedRevision: 2,
		Authorizer:       alwaysAllow,
	})
	if !errors.Is(err, workflowaction.ErrTerminalState) {
		t.Fatalf("expected ErrTerminalState when mutating ARCHIVED, got %v", err)
	}
}

func TestWorkflow_ConcurrentAttempts(t *testing.T) {
	clock, _ := newTestClock(t)
	engine := workflowaction.NewEngine(clock)

	wfID := "wf-race-01"
	_, _ = engine.CreateWorkflow(wfID)

	var (
		concurrency = 25
		wg          sync.WaitGroup
		successes   int64
		staleErrors int64
	)

	wg.Add(concurrency)
	for i := range concurrency {
		go func(idx int) {
			defer wg.Done()
			_, err := engine.Transition(workflowaction.TransitionRequest{
				WorkflowID:       wfID,
				RequestID:        fmt.Sprintf("race-req-%d", idx),
				CorrelationID:    "race-corr",
				TargetState:      workflowaction.StateInProgress,
				ExpectedRevision: 1,
				Authorizer:       alwaysAllow,
			})
			if err == nil {
				atomic.AddInt64(&successes, 1)
			} else if errors.Is(err, workflowaction.ErrStaleRevision) {
				atomic.AddInt64(&staleErrors, 1)
			}
		}(i)
	}
	wg.Wait()

	// Exactly ONE attempt must win; all other attempts must fail with ErrStaleRevision
	if successes != 1 {
		t.Fatalf("expected exactly 1 success, got %d", successes)
	}
	if staleErrors != int64(concurrency-1) {
		t.Fatalf("expected %d stale errors, got %d", concurrency-1, staleErrors)
	}

	inst, _ := engine.GetWorkflow(wfID)
	if inst.CurrentState != workflowaction.StateInProgress || inst.Revision != 2 {
		t.Errorf("unexpected final state/rev: %+v", inst)
	}

	history := engine.AuditHistory(wfID)
	if len(history) != 1 {
		t.Errorf("expected exactly 1 audit entry, got %d", len(history))
	}
}
