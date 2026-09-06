package workflowaction_test

import (
	"errors"
	"testing"
	"time"

	workflowaction "github.com/oshethai/oshe-platform/modules/workflow-action"
)

func setupGovernorTestEnv() (*workflowaction.FailClosedGovernor, time.Time) {
	t0 := time.Date(2026, 9, 6, 9, 0, 0, 0, time.UTC)
	clock := func() time.Time { return t0 }

	matrix := workflowaction.DefaultRuleMatrix()
	catalog := workflowaction.DefaultTransitionCatalog()
	governor := workflowaction.NewFailClosedGovernor(clock, matrix, catalog)
	return governor, t0
}

func TestFailClosed_CriticalFailPriorityBlocksConclusiveTransitions(t *testing.T) {
	governor, _ := setupGovernorTestEnv()

	// High score (9500 bps = 95.00%), but active critical failure
	req := workflowaction.FailClosedTransitionRequest{
		TargetKind:         workflowaction.TargetKindWorkflow,
		TenantID:           "ten_syn_alpha",
		TargetID:           "ins_syn_high_score_crit",
		CurrentState:       string(workflowaction.StateUnderReview),
		TargetState:        string(workflowaction.StateApproved),
		Actor:              "usr_inspector_01",
		ActorRole:          "SUPERVISOR",
		ConditionsMet:      []string{"review_passed"},
		HasCriticalFail:    true,
		CriticalFindingIDs: []string{"fnd_syn_cracked_boiler_01"},
		BasisPoints:        9500, // 95.00%
	}

	res := governor.Qualify(req)
	if res.Permitted {
		t.Errorf("expected transition to be denied due to active critical fail")
	}
	if res.DenialCode != workflowaction.DenialCriticalFailActive {
		t.Errorf("expected DenialCriticalFailActive, got: %s", res.DenialCode)
	}
	if res.PriorityApplied != workflowaction.PriorityCriticalFail {
		t.Errorf("expected priority %s, got: %s", workflowaction.PriorityCriticalFail, res.PriorityApplied)
	}
	if res.Result != workflowaction.RuleResultDenied {
		t.Errorf("expected RuleResultDenied, got: %v", res.Result)
	}
}

func TestFailClosed_UnknownResponseQuarantinesTransition(t *testing.T) {
	governor, _ := setupGovernorTestEnv()

	// High score (9200 bps = 92.00%), zero criticals, but unreviewed UNKNOWN response
	req := workflowaction.FailClosedTransitionRequest{
		TargetKind:             workflowaction.TargetKindWorkflow,
		TenantID:               "ten_syn_alpha",
		TargetID:               "ins_syn_unknown_quarantine",
		CurrentState:           string(workflowaction.StateUnderReview),
		TargetState:            string(workflowaction.StateApproved),
		Actor:                  "usr_inspector_01",
		ActorRole:              "SUPERVISOR",
		ConditionsMet:          []string{"review_passed"},
		HasCriticalFail:        false,
		HasQuarantinedUnknown:  true,
		QuarantinedQuestionIDs: []string{"qst_syn_radiation_shield_01"},
		BasisPoints:            9200, // 92.00%
	}

	res := governor.Qualify(req)
	if res.Permitted {
		t.Errorf("expected transition to be quarantined due to unresolved UNKNOWN")
	}
	if res.DenialCode != workflowaction.DenialUnknownQuarantined {
		t.Errorf("expected DenialUnknownQuarantined, got: %s", res.DenialCode)
	}
	if res.PriorityApplied != workflowaction.PriorityUnknownQuarantine {
		t.Errorf("expected priority %s, got: %s", workflowaction.PriorityUnknownQuarantine, res.PriorityApplied)
	}
	if res.Result != workflowaction.RuleResultQuarantined {
		t.Errorf("expected RuleResultQuarantined, got: %v", res.Result)
	}
}

func TestFailClosed_CriticalFailDominatesUnknownQuarantine(t *testing.T) {
	governor, _ := setupGovernorTestEnv()

	// Both active critical fail AND unreviewed unknown
	req := workflowaction.FailClosedTransitionRequest{
		TargetKind:             workflowaction.TargetKindWorkflow,
		TenantID:               "ten_syn_alpha",
		TargetID:               "ins_syn_crit_and_unk",
		CurrentState:           string(workflowaction.StateUnderReview),
		TargetState:            string(workflowaction.StateApproved),
		Actor:                  "usr_inspector_01",
		ActorRole:              "SUPERVISOR",
		ConditionsMet:          []string{"review_passed"},
		HasCriticalFail:        true,
		CriticalFindingIDs:     []string{"fnd_syn_ammonia_leak_01"},
		HasQuarantinedUnknown:  true,
		QuarantinedQuestionIDs: []string{"qst_syn_ventilation_02"},
		BasisPoints:            9000,
	}

	res := governor.Qualify(req)
	if res.Permitted {
		t.Errorf("expected transition to be denied")
	}
	// Critical-Fail priority (CF1) must dominate over Unknown quarantine (U1)
	if res.DenialCode != workflowaction.DenialCriticalFailActive {
		t.Errorf("expected Critical-Fail priority (DenialCriticalFailActive), got: %s", res.DenialCode)
	}
	if res.PriorityApplied != workflowaction.PriorityCriticalFail {
		t.Errorf("expected priority %s, got: %s", workflowaction.PriorityCriticalFail, res.PriorityApplied)
	}
	if res.Result != workflowaction.RuleResultDenied {
		t.Errorf("expected RuleResultDenied, got: %v", res.Result)
	}
}

func TestFailClosed_ManualOverrideAttemptDeniedUnderDeferredH040(t *testing.T) {
	governor, _ := setupGovernorTestEnv()

	// Caller (even executive supervisor) attempts manual override to bypass failure
	req := workflowaction.FailClosedTransitionRequest{
		TargetKind:         workflowaction.TargetKindWorkflow,
		TenantID:           "ten_syn_alpha",
		TargetID:           "ins_syn_override_attempt",
		CurrentState:       string(workflowaction.StateUnderReview),
		TargetState:        string(workflowaction.StateApproved),
		Actor:              "usr_executive_admin",
		ActorRole:          "ADMIN",
		ConditionsMet:      []string{"review_passed"},
		IsOverrideAttempt:  true,
		OverrideRationale:  "Urgent facility restart required for scheduled production run",
		HasCriticalFail:    true,
		CriticalFindingIDs: []string{"fnd_syn_fire_sprinkler_disabled"},
	}

	res := governor.Qualify(req)
	if res.Permitted {
		t.Errorf("expected manual override to be unconditionally denied under deferred H040-004")
	}
	if res.DenialCode != workflowaction.DenialManualOverrideDeferred {
		t.Errorf("expected DenialManualOverrideDeferred, got: %s", res.DenialCode)
	}
	if res.PriorityApplied != workflowaction.PriorityOverrideDenial {
		t.Errorf("expected priority %s, got: %s", workflowaction.PriorityOverrideDenial, res.PriorityApplied)
	}

	// Verify immutable audit entry recorded for override denial
	history := governor.AuditHistory("ten_syn_alpha", "ins_syn_override_attempt")
	if len(history) != 1 {
		t.Fatalf("expected 1 audit entry for override attempt, got: %d", len(history))
	}
	entry := history[0]
	if entry.Action != "OVERRIDE_ATTEMPT_DENIED" {
		t.Errorf("expected action OVERRIDE_ATTEMPT_DENIED, got: %s", entry.Action)
	}
	if entry.DenialCode != workflowaction.DenialManualOverrideDeferred {
		t.Errorf("expected DenialManualOverrideDeferred in audit, got: %s", entry.DenialCode)
	}
	if !entry.IsOverrideAttempt {
		t.Errorf("expected IsOverrideAttempt = true in audit")
	}
}

func TestFailClosed_AutonomousAIBoundaryEnforced(t *testing.T) {
	governor, _ := setupGovernorTestEnv()

	aiRoles := []string{"AI", "AI_AGENT", "ENGINEERING_AGENT", "SYSTEM_AGENT", "AUTONOMOUS_AGENT", "LLM"}

	for _, role := range aiRoles {
		req := workflowaction.FailClosedTransitionRequest{
			TargetKind:    workflowaction.TargetKindWorkflow,
			TenantID:      "ten_syn_alpha",
			TargetID:      "ins_syn_ai_bypass",
			CurrentState:  string(workflowaction.StateUnderReview),
			TargetState:   string(workflowaction.StateApproved), // Protected state!
			Actor:         "agent_ai_autonomous_01",
			ActorRole:     role,
			ConditionsMet: []string{"review_passed"},
			BasisPoints:   9500,
		}

		res := governor.Qualify(req)
		if res.Permitted {
			t.Errorf("expected AI role %s to be blocked from protected transition", role)
		}
		if res.DenialCode != workflowaction.DenialAutonomousAIBoundary {
			t.Errorf("expected DenialAutonomousAIBoundary for AI role %s, got: %s", role, res.DenialCode)
		}
		if res.PriorityApplied != workflowaction.PriorityAIBoundaryDenial {
			t.Errorf("expected priority %s, got: %s", workflowaction.PriorityAIBoundaryDenial, res.PriorityApplied)
		}
	}
}

func TestFailClosed_AuthorizedHumanResolutionPath(t *testing.T) {
	governor, _ := setupGovernorTestEnv()

	tenantID := "ten_syn_alpha"
	targetID := "ins_syn_quarantine_res_01"
	questionID := "qst_syn_scaffold_lock_01"

	// 1. Register quarantined question
	governor.RegisterQuarantinedQuestion(tenantID, targetID, questionID)
	if !governor.HasActiveQuarantine(tenantID, targetID) {
		t.Fatalf("expected target to have active quarantine")
	}

	// 2. Unauthorized worker attempts to resolve unknown
	err := governor.ResolveUnknownQuestion(tenantID, targetID, questionID, "usr_worker_bob", "WORKER", "I looked at it, it is fine")
	if !errors.Is(err, workflowaction.ErrUnauthorizedActorClass) {
		t.Errorf("expected ErrUnauthorizedActorClass for non-supervisor, got: %v", err)
	}

	// 3. Autonomous AI attempts to resolve unknown
	err = governor.ResolveUnknownQuestion(tenantID, targetID, questionID, "agent_ai_core", "AI_AGENT", "Documented via CV vision model")
	if !errors.Is(err, workflowaction.ErrAutonomousAIBoundary) {
		t.Errorf("expected ErrAutonomousAIBoundary for AI agent, got: %v", err)
	}

	// 4. Authorized human supervisor resolves unknown with valid rationale
	err = governor.ResolveUnknownQuestion(
		tenantID,
		targetID,
		questionID,
		"usr_supervisor_elena",
		"SAFETY_SUPERVISOR",
		"Re-inspected physically; secondary locking pin verified engaged with photographic proof",
	)
	if err != nil {
		t.Fatalf("supervisor resolution failed: %v", err)
	}

	if governor.HasActiveQuarantine(tenantID, targetID) {
		t.Errorf("expected quarantine to be cleared after supervisor resolution")
	}

	// 5. Subsequent qualification now permits transition (Score = 85.00%, conditions met)
	req := workflowaction.FailClosedTransitionRequest{
		TargetKind:    workflowaction.TargetKindWorkflow,
		TenantID:      tenantID,
		TargetID:      targetID,
		CurrentState:  string(workflowaction.StateUnderReview),
		TargetState:   string(workflowaction.StateApproved),
		Actor:         "usr_supervisor_elena",
		ActorRole:     "SUPERVISOR",
		ConditionsMet: []string{"review_passed"},
		BasisPoints:   8500, // 85.00%
	}

	res := governor.Qualify(req)
	if !res.Permitted {
		t.Errorf("expected transition to be permitted after resolution, got denial: %s (%s)", res.DenialCode, res.DenialReason)
	}
	if res.Result != workflowaction.RuleResultPermitted {
		t.Errorf("expected RuleResultPermitted, got: %v", res.Result)
	}
}

func TestFailClosed_AppendOnlyAuditLedgerSequenceAndIntegrity(t *testing.T) {
	governor, _ := setupGovernorTestEnv()

	tenantID := "ten_syn_alpha"
	targetID := "ins_syn_audit_ledger_01"

	// Action 1: Critical Fail block
	_ = governor.Qualify(workflowaction.FailClosedTransitionRequest{
		TargetKind:         workflowaction.TargetKindWorkflow,
		TenantID:           tenantID,
		TargetID:           targetID,
		CurrentState:       string(workflowaction.StateUnderReview),
		TargetState:        string(workflowaction.StateApproved),
		Actor:              "usr_inspector_01",
		ActorRole:          "SUPERVISOR",
		HasCriticalFail:    true,
		CriticalFindingIDs: []string{"fnd_syn_01"},
	})

	// Action 2: Unknown Quarantine block
	_ = governor.Qualify(workflowaction.FailClosedTransitionRequest{
		TargetKind:             workflowaction.TargetKindWorkflow,
		TenantID:               tenantID,
		TargetID:               targetID,
		CurrentState:           string(workflowaction.StateUnderReview),
		TargetState:            string(workflowaction.StateApproved),
		Actor:                  "usr_inspector_01",
		ActorRole:              "SUPERVISOR",
		HasQuarantinedUnknown:  true,
		QuarantinedQuestionIDs: []string{"qst_syn_01"},
	})

	// Action 3: Manual override denial
	_ = governor.Qualify(workflowaction.FailClosedTransitionRequest{
		TargetKind:        workflowaction.TargetKindWorkflow,
		TenantID:          tenantID,
		TargetID:          targetID,
		CurrentState:      string(workflowaction.StateUnderReview),
		TargetState:       string(workflowaction.StateApproved),
		Actor:             "usr_admin_01",
		ActorRole:         "ADMIN",
		IsOverrideAttempt: true,
	})

	// Action 4: AI boundary denial
	_ = governor.Qualify(workflowaction.FailClosedTransitionRequest{
		TargetKind:   workflowaction.TargetKindWorkflow,
		TenantID:     tenantID,
		TargetID:     targetID,
		CurrentState: string(workflowaction.StateUnderReview),
		TargetState:  string(workflowaction.StateApproved),
		Actor:        "agent_ai",
		ActorRole:    "AI_AGENT",
	})

	history := governor.AuditHistory(tenantID, targetID)
	if len(history) != 4 {
		t.Fatalf("expected 4 audit entries, got: %d", len(history))
	}

	// Verify monotonic sequence numbering
	for i, entry := range history {
		expectedSeq := int64(i + 1)
		if entry.Sequence != expectedSeq {
			t.Errorf("entry %d: expected sequence %d, got %d", i, expectedSeq, entry.Sequence)
		}
		if entry.TenantID != tenantID || entry.TargetID != targetID {
			t.Errorf("entry %d: mismatched tenant/target: %s / %s", i, entry.TenantID, entry.TargetID)
		}
		if entry.Timestamp.IsZero() {
			t.Errorf("entry %d: expected non-zero timestamp", i)
		}
	}

	expectedActions := []string{
		"CRITICAL_FAIL_BLOCKED",
		"UNKNOWN_QUARANTINE_RECORDED",
		"OVERRIDE_ATTEMPT_DENIED",
		"AI_BOUNDARY_BLOCKED",
	}

	for i, exp := range expectedActions {
		if history[i].Action != exp {
			t.Errorf("entry %d: expected action %s, got: %s", i, exp, history[i].Action)
		}
	}

	if governor.TotalAuditCount() != 4 {
		t.Errorf("expected total audit count 4, got: %d", governor.TotalAuditCount())
	}
}
