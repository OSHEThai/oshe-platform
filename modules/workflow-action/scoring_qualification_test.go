package workflowaction_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	workflowaction "github.com/oshethai/oshe-platform/modules/workflow-action"
)

// setupQualificationEnv initializes deterministic scoring and fail-closed governance test environments.
func setupQualificationEnv() (*workflowaction.DeterministicScoringEngine, *workflowaction.FailClosedGovernor, time.Time) {
	t0 := time.Date(2026, 9, 6, 10, 0, 0, 0, time.UTC)
	clock := func() time.Time { return t0 }

	engine := workflowaction.NewDeterministicScoringEngine(clock)
	matrix := workflowaction.DefaultRuleMatrix()
	catalog := workflowaction.DefaultTransitionCatalog()
	governor := workflowaction.NewFailClosedGovernor(clock, matrix, catalog)

	return engine, governor, t0
}

// TestQualification_BoundaryRoundingExactBasisPoints verifies R1_ROUND_HALF_UP arithmetic
// at the 80.00% (8000 bps) threshold boundary.
func TestQualification_BoundaryRoundingExactBasisPoints(t *testing.T) {
	engine, _, t0 := setupQualificationEnv()

	cases := []struct {
		name                string
		earnedPts           float64
		maxPts              float64
		expectedBasisPoints int64
		expectedPercent     float64
		expectedDisplay     string
		expectedThreshold   bool
		expectedOutcome     workflowaction.ComplianceOutcome
	}{
		{
			name:                "boundary_round_up_passes_80",
			earnedPts:           79.995,
			maxPts:              100.0,
			expectedBasisPoints: 8000,
			expectedPercent:     80.00,
			expectedDisplay:     "80.0%",
			expectedThreshold:   true,
			expectedOutcome:     workflowaction.OutcomePass,
		},
		{
			name:                "boundary_round_down_fails_80",
			earnedPts:           79.994,
			maxPts:              100.0,
			expectedBasisPoints: 7999,
			expectedPercent:     79.99,
			expectedDisplay:     "80.0%",
			expectedThreshold:   false,
			expectedOutcome:     workflowaction.OutcomeFail,
		},
		{
			name:                "exact_threshold_passes",
			earnedPts:           80.000,
			maxPts:              100.0,
			expectedBasisPoints: 8000,
			expectedPercent:     80.00,
			expectedDisplay:     "80.0%",
			expectedThreshold:   true,
			expectedOutcome:     workflowaction.OutcomePass,
		},
		{
			name:                "sub_threshold_clearly_fails",
			earnedPts:           79.900,
			maxPts:              100.0,
			expectedBasisPoints: 7990,
			expectedPercent:     79.90,
			expectedDisplay:     "79.9%",
			expectedThreshold:   false,
			expectedOutcome:     workflowaction.OutcomeFail,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := workflowaction.ScoringEvaluationRequest{
				ExecutionID:       "exec_syn_round_" + tc.name,
				TenantID:          "ten_syn_alpha",
				TemplateID:        "chk_syn_alpha_v1",
				TemplateVersion:   "1.0.0",
				RuleMatrixVersion: "1.0.0",
				EvaluatedAt:       t0,
				Sections: []workflowaction.ScoredSection{
					{
						SectionID:    "sec_qualification",
						SectionTitle: "Qualification Section",
						Weight:       1.0,
						Questions: []workflowaction.ScoredQuestion{
							{
								QuestionID:    "q_score_bound",
								SectionID:     "sec_qualification",
								QuestionType:  "NUMERIC_MEASUREMENT",
								MaxPoints:     tc.maxPts,
								EarnedPoints:  tc.earnedPts,
								ResponseState: "GRADED",
							},
						},
					},
				},
			}

			res, err := engine.Evaluate(req)
			if err != nil {
				t.Fatalf("unexpected scoring error: %v", err)
			}

			if res.BasisPoints != tc.expectedBasisPoints {
				t.Errorf("expected %d bps, got %d", tc.expectedBasisPoints, res.BasisPoints)
			}
			if res.RoundedScorePercent != tc.expectedPercent {
				t.Errorf("expected %.2f%%, got %.2f%%", tc.expectedPercent, res.RoundedScorePercent)
			}
			if res.DisplayScore != tc.expectedDisplay {
				t.Errorf("expected display %s, got %s", tc.expectedDisplay, res.DisplayScore)
			}
			if res.PassPredicates.ScoreThresholdSatisfied != tc.expectedThreshold {
				t.Errorf("expected score threshold predicate %v, got %v", tc.expectedThreshold, res.PassPredicates.ScoreThresholdSatisfied)
			}
			if res.Outcome != tc.expectedOutcome {
				t.Errorf("expected outcome %s, got %s", tc.expectedOutcome, res.Outcome)
			}
		})
	}
}

// TestQualification_NAAndExclusions_DenominatorSubtraction verifies that NA responses and
// non-scored types (TEXT_NOTE, EVIDENCE_ATTACHMENT) subtract points from the denominator with zero negative score impact.
func TestQualification_NAAndExclusions_DenominatorSubtraction(t *testing.T) {
	engine, _, t0 := setupQualificationEnv()

	req := workflowaction.ScoringEvaluationRequest{
		ExecutionID:       "exec_syn_na_exclusion_01",
		TenantID:          "ten_syn_alpha",
		TemplateID:        "chk_syn_alpha_v1",
		TemplateVersion:   "1.0.0",
		RuleMatrixVersion: "1.0.0",
		EvaluatedAt:       t0,
		Sections: []workflowaction.ScoredSection{
			{
				SectionID:    "sec_pressurized_tanks",
				SectionTitle: "Pressurized Tanks",
				Weight:       1.0,
				Questions: []workflowaction.ScoredQuestion{
					{
						QuestionID:    "q_tank_valves",
						SectionID:     "sec_pressurized_tanks",
						QuestionType:  "PASS_FAIL_NA_UNKNOWN",
						MaxPoints:     40.0,
						EarnedPoints:  40.0,
						ResponseState: "PASS",
					},
					{
						QuestionID:    "q_cryo_subcooler",
						SectionID:     "sec_pressurized_tanks",
						QuestionType:  "PASS_FAIL_NA_UNKNOWN",
						MaxPoints:     60.0,
						EarnedPoints:  0.0,
						ResponseState: "NA",
						Notes:         "Facility does not operate cryogenic subcoolers",
					},
					{
						QuestionID:   "q_general_notes",
						SectionID:    "sec_pressurized_tanks",
						QuestionType: "TEXT_NOTE",
						MaxPoints:    0.0,
						EarnedPoints: 0.0,
						IsExcluded:   true,
					},
					{
						QuestionID:   "q_photo_evidence",
						SectionID:    "sec_pressurized_tanks",
						QuestionType: "EVIDENCE_ATTACHMENT",
						MaxPoints:    0.0,
						EarnedPoints: 0.0,
						IsExcluded:   true,
					},
				},
			},
		},
	}

	res, err := engine.Evaluate(req)
	if err != nil {
		t.Fatalf("scoring evaluation failed: %v", err)
	}

	sec := res.SectionResults[0]
	if sec.NAPoints != 60.0 {
		t.Errorf("expected 60.0 NA points subtracted, got %.1f", sec.NAPoints)
	}
	if sec.EffectiveDenominator != 40.0 {
		t.Errorf("expected effective denominator 40.0, got %.1f", sec.EffectiveDenominator)
	}
	if res.BasisPoints != 10000 {
		t.Errorf("expected 10000 bps (100.00%%) with NA exclusion, got %d", res.BasisPoints)
	}
	if res.Outcome != workflowaction.OutcomePass {
		t.Errorf("expected PASS outcome, got %s", res.Outcome)
	}
	if res.TotalQuestionsExcluded != 2 {
		t.Errorf("expected 2 excluded non-scored questions, got %d", res.TotalQuestionsExcluded)
	}
}

// TestQualification_UnknownQuarantine_ProvisionalOutcome verifies U1_QUARANTINE_DENOMINATOR:
// unreviewed UNKNOWN responses quarantine points from the denominator, lock outcome to PROVISIONAL,
// and block conclusive transition in FailClosedGovernor.
func TestQualification_UnknownQuarantine_ProvisionalOutcome(t *testing.T) {
	engine, governor, t0 := setupQualificationEnv()

	req := workflowaction.ScoringEvaluationRequest{
		ExecutionID:       "exec_syn_unk_quarantine_01",
		TenantID:          "ten_syn_alpha",
		TemplateID:        "chk_syn_alpha_v1",
		TemplateVersion:   "1.0.0",
		RuleMatrixVersion: "1.0.0",
		EvaluatedAt:       t0,
		Sections: []workflowaction.ScoredSection{
			{
				SectionID:    "sec_hvac",
				SectionTitle: "HVAC and Air Scrubbers",
				Weight:       1.0,
				Questions: []workflowaction.ScoredQuestion{
					{
						QuestionID:    "q_filter_diff_pressure",
						SectionID:     "sec_hvac",
						QuestionType:  "PASS_FAIL_NA_UNKNOWN",
						MaxPoints:     100.0,
						EarnedPoints:  95.0,
						ResponseState: "PASS",
					},
					{
						QuestionID:    "q_exhaust_damper_unknown",
						SectionID:     "sec_hvac",
						QuestionType:  "PASS_FAIL_NA_UNKNOWN",
						MaxPoints:     25.0,
						EarnedPoints:  0.0,
						ResponseState: "UNKNOWN",
						Notes:         "Exhaust damper sensor obscured by duct insulation",
					},
				},
			},
		},
	}

	res, err := engine.Evaluate(req)
	if err != nil {
		t.Fatalf("scoring evaluation failed: %v", err)
	}

	if !res.HasQuarantinedUnknown {
		t.Errorf("expected HasQuarantinedUnknown to be true")
	}
	if len(res.QuarantinedUnknownQuestions) != 1 || res.QuarantinedUnknownQuestions[0] != "q_exhaust_damper_unknown" {
		t.Errorf("unexpected quarantined questions list: %v", res.QuarantinedUnknownQuestions)
	}
	if res.PassPredicates.NoUnresolvedUnknownQuarantined {
		t.Errorf("expected NoUnresolvedUnknownQuarantined predicate to be false")
	}
	if res.PassPredicates.AllPredicatesSatisfied {
		t.Errorf("expected AllPredicatesSatisfied to be false")
	}
	if res.Outcome != workflowaction.OutcomeProvisionalPendingUnknownResolution {
		t.Errorf("expected PROVISIONAL_PENDING_UNKNOWN_RESOLUTION, got %s", res.Outcome)
	}

	// Verify fail-closed governor blocks conclusive transition
	transReq := workflowaction.FailClosedTransitionRequest{
		TargetKind:             workflowaction.TargetKindWorkflow,
		TenantID:               "ten_syn_alpha",
		TargetID:               "exec_syn_unk_quarantine_01",
		CurrentState:           string(workflowaction.StateUnderReview),
		TargetState:            string(workflowaction.StateApproved),
		Actor:                  "usr_inspector_01",
		ActorRole:              "SUPERVISOR",
		ConditionsMet:          []string{"review_passed"},
		HasQuarantinedUnknown:  true,
		QuarantinedQuestionIDs: res.QuarantinedUnknownQuestions,
		BasisPoints:            res.BasisPoints,
	}

	govRes := governor.Qualify(transReq)
	if govRes.Permitted {
		t.Errorf("expected governor transition to be blocked")
	}
	if govRes.Result != workflowaction.RuleResultQuarantined {
		t.Errorf("expected RuleResultQuarantined, got %v", govRes.Result)
	}
	if govRes.DenialCode != workflowaction.DenialUnknownQuarantined {
		t.Errorf("expected DenialUnknownQuarantined, got %s", govRes.DenialCode)
	}
	if govRes.PriorityApplied != workflowaction.PriorityUnknownQuarantine {
		t.Errorf("expected priority %s, got %s", workflowaction.PriorityUnknownQuarantine, govRes.PriorityApplied)
	}
}

// TestQualification_RuleVersionTraceability verifies that evaluation results record complete version
// traceability and that governor enforces matrix version compatibility.
func TestQualification_RuleVersionTraceability(t *testing.T) {
	engine, governor, t0 := setupQualificationEnv()

	req := workflowaction.ScoringEvaluationRequest{
		ExecutionID:       "exec_syn_trace_01",
		TenantID:          "ten_syn_alpha",
		TemplateID:        "chk_syn_alpha_v1",
		TemplateVersion:   "2.3.1",
		RuleMatrixVersion: "1.0.0",
		EvaluatedAt:       t0,
		Sections: []workflowaction.ScoredSection{
			{
				SectionID:    "sec_main",
				SectionTitle: "Main",
				Weight:       1.0,
				Questions: []workflowaction.ScoredQuestion{
					{
						QuestionID:    "q_trace",
						SectionID:     "sec_main",
						QuestionType:  "PASS_FAIL_NA_UNKNOWN",
						MaxPoints:     10.0,
						EarnedPoints:  10.0,
						ResponseState: "PASS",
					},
				},
			},
		},
	}

	res, err := engine.Evaluate(req)
	if err != nil {
		t.Fatalf("scoring evaluation failed: %v", err)
	}

	if res.FormulaVersion != workflowaction.FormulaVersionSelected {
		t.Errorf("expected formula version %s, got %s", workflowaction.FormulaVersionSelected, res.FormulaVersion)
	}
	if res.ScoringModel != workflowaction.ScoringModelSelected {
		t.Errorf("expected model %s, got %s", workflowaction.ScoringModelSelected, res.ScoringModel)
	}
	if res.UnknownHandling != workflowaction.UnknownHandlingSelected {
		t.Errorf("expected unknown handling %s, got %s", workflowaction.UnknownHandlingSelected, res.UnknownHandling)
	}
	if res.RoundingRule != workflowaction.RoundingRuleSelected {
		t.Errorf("expected rounding rule %s, got %s", workflowaction.RoundingRuleSelected, res.RoundingRule)
	}
	if res.CriticalFailPolicy != workflowaction.CriticalFailPolicySelected {
		t.Errorf("expected critical policy %s, got %s", workflowaction.CriticalFailPolicySelected, res.CriticalFailPolicy)
	}
	if !strings.Contains(res.TraceabilityKey, "chk_syn_alpha_v1") ||
		!strings.Contains(res.TraceabilityKey, "2.3.1") ||
		!strings.Contains(res.TraceabilityKey, "1.0.0") {
		t.Errorf("traceability key missing version tokens: %s", res.TraceabilityKey)
	}

	// Governor check with incompatible rule matrix version must fail closed
	badReq := workflowaction.FailClosedTransitionRequest{
		TargetKind:    workflowaction.TargetKindWorkflow,
		TenantID:      "ten_syn_alpha",
		TargetID:      "ins_bad_version",
		CurrentState:  string(workflowaction.StateUnderReview),
		TargetState:   string(workflowaction.StateApproved),
		Actor:         "usr_supervisor",
		ActorRole:     "SUPERVISOR",
		RuleVersion:   "9.9.9", // Incompatible version!
		ConditionsMet: []string{"review_passed"},
		BasisPoints:   9500,
	}

	badRes := governor.Qualify(badReq)
	if badRes.Permitted {
		t.Errorf("expected governor to reject incompatible matrix version")
	}
	if badRes.DenialCode != workflowaction.DenialIncompatibleVersion {
		t.Errorf("expected DenialIncompatibleVersion, got %s", badRes.DenialCode)
	}
}

// TestQualification_CriticalFailPriorityHierarchy verifies the priority order:
// Critical Fail > Unknown Quarantine > Score Threshold.
func TestQualification_CriticalFailPriorityHierarchy(t *testing.T) {
	engine, governor, t0 := setupQualificationEnv()

	// High score (9500 bps), but has BOTH critical fail and unknown quarantine
	req := workflowaction.ScoringEvaluationRequest{
		ExecutionID:       "exec_syn_hierarchy_01",
		TenantID:          "ten_syn_alpha",
		TemplateID:        "chk_syn_alpha_v1",
		TemplateVersion:   "1.0.0",
		RuleMatrixVersion: "1.0.0",
		EvaluatedAt:       t0,
		Sections: []workflowaction.ScoredSection{
			{
				SectionID:    "sec_hazardous",
				SectionTitle: "Hazardous Operations",
				Weight:       1.0,
				Questions: []workflowaction.ScoredQuestion{
					{
						QuestionID:    "q_gas_norm",
						SectionID:     "sec_hazardous",
						QuestionType:  "PASS_FAIL_NA_UNKNOWN",
						MaxPoints:     100.0,
						EarnedPoints:  95.0,
						ResponseState: "PASS",
					},
					{
						QuestionID:     "q_gas_crit",
						SectionID:      "sec_hazardous",
						QuestionType:   "PASS_FAIL_NA_UNKNOWN",
						MaxPoints:      0.0,
						EarnedPoints:   0.0,
						ResponseState:  "FAIL",
						IsCriticalFail: true,
						Notes:          "LFL sensor disabled in fueling zone",
					},
					{
						QuestionID:    "q_gas_unk",
						SectionID:     "sec_hazardous",
						QuestionType:  "PASS_FAIL_NA_UNKNOWN",
						MaxPoints:     10.0,
						EarnedPoints:  0.0,
						ResponseState: "UNKNOWN",
					},
				},
			},
		},
	}

	res, err := engine.Evaluate(req)
	if err != nil {
		t.Fatalf("scoring evaluation failed: %v", err)
	}

	// Score must NOT be masked
	if res.BasisPoints != 9500 {
		t.Errorf("expected unmasked score 9500 bps, got %d", res.BasisPoints)
	}
	// CF1 priority flag dominates outcome over UNKNOWN
	if res.Outcome != workflowaction.OutcomeNonCompliantCritical {
		t.Errorf("expected NON_COMPLIANT_CRITICAL outcome, got %s", res.Outcome)
	}

	// Fail-closed governor priority hierarchy verification
	transReq := workflowaction.FailClosedTransitionRequest{
		TargetKind:             workflowaction.TargetKindWorkflow,
		TenantID:               "ten_syn_alpha",
		TargetID:               "exec_syn_hierarchy_01",
		CurrentState:           string(workflowaction.StateUnderReview),
		TargetState:            string(workflowaction.StateApproved),
		Actor:                  "usr_supervisor_01",
		ActorRole:              "SUPERVISOR",
		ConditionsMet:          []string{"review_passed"},
		HasCriticalFail:        true,
		CriticalFindingIDs:     res.CriticalFailQuestions,
		HasQuarantinedUnknown:  true,
		QuarantinedQuestionIDs: res.QuarantinedUnknownQuestions,
		BasisPoints:            res.BasisPoints,
	}

	govRes := governor.Qualify(transReq)
	if govRes.Permitted {
		t.Errorf("expected transition to be denied")
	}
	// Critical-Fail priority (CF1) must dominate over Unknown quarantine
	if govRes.PriorityApplied != workflowaction.PriorityCriticalFail {
		t.Errorf("expected PriorityCriticalFail, got %s", govRes.PriorityApplied)
	}
	if govRes.DenialCode != workflowaction.DenialCriticalFailActive {
		t.Errorf("expected DenialCriticalFailActive, got %s", govRes.DenialCode)
	}
}

// TestQualification_DeferredManualOverrideDenial verifies that any manual override attempt
// is unconditionally denied under deferred Gate H040-004 and logged to audit.
func TestQualification_DeferredManualOverrideDenial(t *testing.T) {
	_, governor, _ := setupQualificationEnv()

	tenantID := "ten_syn_alpha"
	targetID := "ins_override_denied_01"

	req := workflowaction.FailClosedTransitionRequest{
		TargetKind:         workflowaction.TargetKindWorkflow,
		TenantID:           tenantID,
		TargetID:           targetID,
		CurrentState:       string(workflowaction.StateUnderReview),
		TargetState:        string(workflowaction.StateApproved),
		Actor:              "usr_executive_director",
		ActorRole:          "ADMIN",
		ConditionsMet:      []string{"review_passed"},
		IsOverrideAttempt:  true,
		OverrideRationale:  "Critical production launch override requested by plant VP",
		HasCriticalFail:    true,
		CriticalFindingIDs: []string{"fnd_critical_pressure_leak"},
		BasisPoints:        9500,
	}

	res := governor.Qualify(req)
	if res.Permitted {
		t.Errorf("expected override to be denied unconditionally")
	}
	if res.DenialCode != workflowaction.DenialManualOverrideDeferred {
		t.Errorf("expected DenialManualOverrideDeferred, got %s", res.DenialCode)
	}
	if res.PriorityApplied != workflowaction.PriorityOverrideDenial {
		t.Errorf("expected PriorityOverrideDenial, got %s", res.PriorityApplied)
	}

	// Verify immutable audit ledger entry
	history := governor.AuditHistory(tenantID, targetID)
	if len(history) != 1 {
		t.Fatalf("expected 1 audit entry, got %d", len(history))
	}
	entry := history[0]
	if entry.Action != "OVERRIDE_ATTEMPT_DENIED" {
		t.Errorf("expected action OVERRIDE_ATTEMPT_DENIED, got %s", entry.Action)
	}
	if !entry.IsOverrideAttempt {
		t.Errorf("expected IsOverrideAttempt = true")
	}
	if !strings.Contains(entry.Reason, "H040-004") {
		t.Errorf("expected audit reason to cite H040-004, got %s", entry.Reason)
	}
}

// TestQualification_AutonomousAIBoundaryDenial verifies that autonomous AI agents are strictly
// prohibited from authorizing transitions to protected states or clearing quarantines/criticals.
func TestQualification_AutonomousAIBoundaryDenial(t *testing.T) {
	_, governor, _ := setupQualificationEnv()

	aiRoles := []string{"AI", "AI_AGENT", "ENGINEERING_AGENT", "SYSTEM_AGENT", "AUTONOMOUS_AGENT", "LLM"}

	for _, role := range aiRoles {
		req := workflowaction.FailClosedTransitionRequest{
			TargetKind:    workflowaction.TargetKindWorkflow,
			TenantID:      "ten_syn_alpha",
			TargetID:      "ins_ai_boundary_" + role,
			CurrentState:  string(workflowaction.StateUnderReview),
			TargetState:   string(workflowaction.StateApproved),
			Actor:         "agent_autonomous_evaluator",
			ActorRole:     role,
			ConditionsMet: []string{"review_passed"},
			BasisPoints:   9500,
		}

		res := governor.Qualify(req)
		if res.Permitted {
			t.Errorf("expected AI role %s to be denied authorization to protected state", role)
		}
		if res.DenialCode != workflowaction.DenialAutonomousAIBoundary {
			t.Errorf("expected DenialAutonomousAIBoundary for role %s, got %s", role, res.DenialCode)
		}
		if res.PriorityApplied != workflowaction.PriorityAIBoundaryDenial {
			t.Errorf("expected PriorityAIBoundaryDenial, got %s", res.PriorityApplied)
		}
	}

	// AI attempting to resolve unknown quarantine must fail closed
	err := governor.ResolveUnknownQuestion(
		"ten_syn_alpha",
		"ins_target_ai",
		"q_some_unk",
		"agent_ai_system",
		"AI_AGENT",
		"Resolved by machine vision model",
	)
	if !errors.Is(err, workflowaction.ErrAutonomousAIBoundary) {
		t.Errorf("expected ErrAutonomousAIBoundary, got %v", err)
	}

	// AI attempting to clear critical finding must fail closed
	err = governor.ClearCriticalFinding(
		"ten_syn_alpha",
		"ins_target_ai",
		"fnd_some_crit",
		"agent_ai_system",
		"AI_AGENT",
		"Cleared by acoustic sensor analysis",
	)
	if !errors.Is(err, workflowaction.ErrAutonomousAIBoundary) {
		t.Errorf("expected ErrAutonomousAIBoundary, got %v", err)
	}
}

// TestQualification_AuthorizedHumanResolutionPath verifies that only an authorized human supervisor
// can resolve an UNKNOWN quarantine and enable subsequent state transition.
func TestQualification_AuthorizedHumanResolutionPath(t *testing.T) {
	_, governor, _ := setupQualificationEnv()

	tenantID := "ten_syn_alpha"
	targetID := "ins_human_resolution_01"
	questionID := "q_confined_space_air_test"

	governor.RegisterQuarantinedQuestion(tenantID, targetID, questionID)
	if !governor.HasActiveQuarantine(tenantID, targetID) {
		t.Fatalf("expected active quarantine")
	}

	// Supervisor resolves with valid rationale
	err := governor.ResolveUnknownQuestion(
		tenantID,
		targetID,
		questionID,
		"usr_safety_supervisor_kane",
		"SAFETY_SUPERVISOR",
		"Direct physical re-testing confirmed atmospheric oxygen at 20.9% and LEL at 0%",
	)
	if err != nil {
		t.Fatalf("unexpected supervisor resolution error: %v", err)
	}

	if governor.HasActiveQuarantine(tenantID, targetID) {
		t.Errorf("expected quarantine to be cleared")
	}

	// Transition now qualifies successfully
	transReq := workflowaction.FailClosedTransitionRequest{
		TargetKind:    workflowaction.TargetKindWorkflow,
		TenantID:      tenantID,
		TargetID:      targetID,
		CurrentState:  string(workflowaction.StateUnderReview),
		TargetState:   string(workflowaction.StateApproved),
		Actor:         "usr_safety_supervisor_kane",
		ActorRole:     "SUPERVISOR",
		ConditionsMet: []string{"review_passed"},
		BasisPoints:   8800, // 88.00%
	}

	res := governor.Qualify(transReq)
	if !res.Permitted {
		t.Errorf("expected transition to be permitted after resolution, got %s (%s)", res.DenialCode, res.DenialReason)
	}
}

// TestQualification_DynamicWeightRedistribution verifies that inactive sections (all NA or UNKNOWN)
// have their weights reallocated proportionally across active sections.
func TestQualification_DynamicWeightRedistribution(t *testing.T) {
	engine, _, t0 := setupQualificationEnv()

	// Section 1: weight 0.60, 90/100 -> Ratio 0.90
	// Section 2: weight 0.40, all NA -> Inactive
	// Section 1 effective weight = 0.60 / 0.60 = 1.0
	// Total score = 1.0 * 0.90 = 0.90 -> 90.00% (9000 bps)
	req := workflowaction.ScoringEvaluationRequest{
		ExecutionID:       "exec_syn_redistribute_01",
		TenantID:          "ten_syn_alpha",
		TemplateID:        "chk_syn_alpha_v1",
		TemplateVersion:   "1.0.0",
		RuleMatrixVersion: "1.0.0",
		EvaluatedAt:       t0,
		Sections: []workflowaction.ScoredSection{
			{
				SectionID:    "sec_active",
				SectionTitle: "Active Section",
				Weight:       0.60,
				Questions: []workflowaction.ScoredQuestion{
					{
						QuestionID:    "q_act",
						SectionID:     "sec_active",
						QuestionType:  "PASS_FAIL_NA_UNKNOWN",
						MaxPoints:     100.0,
						EarnedPoints:  90.0,
						ResponseState: "PASS",
					},
				},
			},
			{
				SectionID:    "sec_inactive",
				SectionTitle: "Inactive Section",
				Weight:       0.40,
				Questions: []workflowaction.ScoredQuestion{
					{
						QuestionID:    "q_inact_na",
						SectionID:     "sec_inactive",
						QuestionType:  "PASS_FAIL_NA_UNKNOWN",
						MaxPoints:     50.0,
						EarnedPoints:  0.0,
						ResponseState: "NA",
					},
				},
			},
		},
	}

	res, err := engine.Evaluate(req)
	if err != nil {
		t.Fatalf("scoring evaluation failed: %v", err)
	}

	if res.ActiveSectionsCount != 1 {
		t.Errorf("expected 1 active section, got %d", res.ActiveSectionsCount)
	}
	if res.BasisPoints != 9000 {
		t.Errorf("expected 9000 bps (90.00%%), got %d", res.BasisPoints)
	}
	if res.SectionResults[1].IsActive {
		t.Errorf("expected section 2 to be inactive")
	}
	if res.SectionResults[1].EffectiveWeight != 0.0 {
		t.Errorf("expected section 2 effective weight 0.0, got %f", res.SectionResults[1].EffectiveWeight)
	}
}

// TestQualification_AuditLedgerMonotonicity verifies monotonic sequence numbering and immutable audit preservation.
func TestQualification_AuditLedgerMonotonicity(t *testing.T) {
	_, governor, _ := setupQualificationEnv()

	tenantID := "ten_syn_audit_mono"
	targetID := "ins_target_audit_01"

	// Trigger 3 sequential qualifications
	_ = governor.Qualify(workflowaction.FailClosedTransitionRequest{
		TargetKind:         workflowaction.TargetKindWorkflow,
		TenantID:           tenantID,
		TargetID:           targetID,
		CurrentState:       string(workflowaction.StateUnderReview),
		TargetState:        string(workflowaction.StateApproved),
		Actor:              "usr_01",
		ActorRole:          "SUPERVISOR",
		HasCriticalFail:    true,
		CriticalFindingIDs: []string{"fnd_01"},
	})

	_ = governor.Qualify(workflowaction.FailClosedTransitionRequest{
		TargetKind:        workflowaction.TargetKindWorkflow,
		TenantID:          tenantID,
		TargetID:          targetID,
		CurrentState:      string(workflowaction.StateUnderReview),
		TargetState:       string(workflowaction.StateApproved),
		Actor:             "usr_02",
		ActorRole:         "ADMIN",
		IsOverrideAttempt: true,
	})

	_ = governor.Qualify(workflowaction.FailClosedTransitionRequest{
		TargetKind:   workflowaction.TargetKindWorkflow,
		TenantID:     tenantID,
		TargetID:     targetID,
		CurrentState: string(workflowaction.StateUnderReview),
		TargetState:  string(workflowaction.StateApproved),
		Actor:        "agent_01",
		ActorRole:    "AI_AGENT",
	})

	history := governor.AuditHistory(tenantID, targetID)
	if len(history) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(history))
	}

	for i, entry := range history {
		expectedSeq := int64(i + 1)
		if entry.Sequence != expectedSeq {
			t.Errorf("entry %d: expected sequence %d, got %d", i, expectedSeq, entry.Sequence)
		}
		if entry.Timestamp.IsZero() {
			t.Errorf("entry %d: timestamp should not be zero", i)
		}
	}
}
