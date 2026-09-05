package workflowaction_test

import (
	"testing"
	"time"

	workflowaction "github.com/oshethai/oshe-platform/modules/workflow-action"
)

func setupScoringEngine() (*workflowaction.DeterministicScoringEngine, time.Time) {
	t0 := time.Date(2026, 9, 5, 23, 30, 0, 0, time.UTC)
	clock := func() time.Time { return t0 }
	engine := workflowaction.NewDeterministicScoringEngine(clock)
	return engine, t0
}

func TestScoring_Model2WeightedCalculation(t *testing.T) {
	engine, t0 := setupScoringEngine()

	// Section 1: Weight 0.60, Max 100, Earned 80 -> Ratio 0.80 -> Contrib 0.48
	// Section 2: Weight 0.40, Max 50,  Earned 45 -> Ratio 0.90 -> Contrib 0.36
	// Total: 0.48 + 0.36 = 0.84 (84.00% / 8400 bps)
	req := workflowaction.ScoringEvaluationRequest{
		ExecutionID:       "ins_syn_weighted_01",
		TenantID:          "ten_synthetic_alpha",
		TemplateID:        "chk_syn_pilot_plant_safety_v1",
		TemplateVersion:   "1.1.0",
		RuleMatrixVersion: "1.0.0",
		EvaluatedAt:       t0,
		Sections: []workflowaction.ScoredSection{
			{
				SectionID:    "sec_electrical",
				SectionTitle: "Electrical Systems",
				Weight:       0.60,
				Questions: []workflowaction.ScoredQuestion{
					{
						QuestionID:    "q_elec_01",
						SectionID:     "sec_electrical",
						QuestionType:  "PASS_FAIL_NA_UNKNOWN",
						MaxPoints:     100.0,
						EarnedPoints:  80.0,
						ResponseState: "PASS",
					},
				},
			},
			{
				SectionID:    "sec_mechanical",
				SectionTitle: "Mechanical Equipment",
				Weight:       0.40,
				Questions: []workflowaction.ScoredQuestion{
					{
						QuestionID:    "q_mech_01",
						SectionID:     "sec_mechanical",
						QuestionType:  "PASS_FAIL_NA_UNKNOWN",
						MaxPoints:     50.0,
						EarnedPoints:  45.0,
						ResponseState: "PASS",
					},
				},
			},
		},
	}

	res, err := engine.Evaluate(req)
	if err != nil {
		t.Fatalf("unexpected evaluation error: %v", err)
	}

	if res.BasisPoints != 8400 {
		t.Errorf("expected 8400 basis points, got %d", res.BasisPoints)
	}
	if res.RoundedScorePercent != 84.00 {
		t.Errorf("expected 84.00 percent, got %.2f", res.RoundedScorePercent)
	}
	if res.Outcome != workflowaction.OutcomePass {
		t.Errorf("expected PASS outcome, got %s", res.Outcome)
	}
	if !res.PassPredicates.AllPredicatesSatisfied {
		t.Errorf("expected all pass predicates satisfied")
	}
}

func TestScoring_ExclusionsAndNA_ZeroPenalty(t *testing.T) {
	engine, t0 := setupScoringEngine()

	// Section: Weight 1.0, Max 100 pts across 3 questions:
	// Q1: Scored, 50 pts, Earned 50 pts (PASS)
	// Q2: NA, 50 pts -> Denominator subtracted by 50! Effective Denominator = 50.
	// Q3: Non-scored TEXT_NOTE (excluded from denominator)
	// Ratio: 50 / (100 - 50) = 50 / 50 = 1.0 (100.00%)
	req := workflowaction.ScoringEvaluationRequest{
		ExecutionID:       "ins_syn_na_02",
		TenantID:          "ten_synthetic_alpha",
		TemplateID:        "chk_syn_pilot_plant_safety_v1",
		TemplateVersion:   "1.1.0",
		RuleMatrixVersion: "1.0.0",
		EvaluatedAt:       t0,
		Sections: []workflowaction.ScoredSection{
			{
				SectionID:    "sec_boiler",
				SectionTitle: "Boiler Safety",
				Weight:       1.0,
				Questions: []workflowaction.ScoredQuestion{
					{
						QuestionID:    "q_b1",
						SectionID:     "sec_boiler",
						QuestionType:  "PASS_FAIL_NA_UNKNOWN",
						MaxPoints:     50.0,
						EarnedPoints:  50.0,
						ResponseState: "PASS",
					},
					{
						QuestionID:    "q_b2_na",
						SectionID:     "sec_boiler",
						QuestionType:  "PASS_FAIL_NA_UNKNOWN",
						MaxPoints:     50.0,
						EarnedPoints:  0.0,
						ResponseState: "NA",
						Notes:         "Boiler model does not utilize auxiliary economizer",
					},
					{
						QuestionID:   "q_b3_note",
						SectionID:    "sec_boiler",
						QuestionType: "TEXT_NOTE",
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
		t.Fatalf("unexpected error: %v", err)
	}

	if res.BasisPoints != 10000 {
		t.Errorf("expected 10000 basis points (100%%) with NA exclusion, got %d", res.BasisPoints)
	}
	secRes := res.SectionResults[0]
	if secRes.NAPoints != 50.0 {
		t.Errorf("expected 50 NA points subtracted, got %.1f", secRes.NAPoints)
	}
	if secRes.EffectiveDenominator != 50.0 {
		t.Errorf("expected effective denominator 50.0, got %.1f", secRes.EffectiveDenominator)
	}
}

func TestScoring_UnknownQuarantine_BlocksPass(t *testing.T) {
	engine, t0 := setupScoringEngine()

	// High score (90%), but contains an unresolved UNKNOWN
	req := workflowaction.ScoringEvaluationRequest{
		ExecutionID:       "ins_syn_unknown_03",
		TenantID:          "ten_synthetic_alpha",
		TemplateID:        "chk_syn_pilot_plant_safety_v1",
		TemplateVersion:   "1.1.0",
		RuleMatrixVersion: "1.0.0",
		EvaluatedAt:       t0,
		Sections: []workflowaction.ScoredSection{
			{
				SectionID:    "sec_pressure",
				SectionTitle: "Pressure Vessels",
				Weight:       1.0,
				Questions: []workflowaction.ScoredQuestion{
					{
						QuestionID:    "q_pv_01",
						SectionID:     "sec_pressure",
						QuestionType:  "PASS_FAIL_NA_UNKNOWN",
						MaxPoints:     100.0,
						EarnedPoints:  90.0,
						ResponseState: "PASS",
					},
					{
						QuestionID:    "q_pv_02_unknown",
						SectionID:     "sec_pressure",
						QuestionType:  "PASS_FAIL_NA_UNKNOWN",
						MaxPoints:     20.0,
						EarnedPoints:  0.0,
						ResponseState: "UNKNOWN",
						Notes:         "Gauge tag obstructed by scaffolding; verification pending",
					},
				},
			},
		},
	}

	res, err := engine.Evaluate(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !res.HasQuarantinedUnknown {
		t.Errorf("expected HasQuarantinedUnknown to be true")
	}
	if res.PassPredicates.NoUnresolvedUnknownQuarantined {
		t.Errorf("expected pass predicate NoUnresolvedUnknownQuarantined to fail (be false)")
	}
	if res.PassPredicates.AllPredicatesSatisfied {
		t.Errorf("expected AllPredicatesSatisfied to be false")
	}
	if res.Outcome != workflowaction.OutcomeProvisionalPendingUnknownResolution {
		t.Errorf("expected outcome PROVISIONAL_PENDING_UNKNOWN_RESOLUTION, got %s", res.Outcome)
	}
}

func TestScoring_RoundingBoundaries_IntegerBasisPoints(t *testing.T) {
	engine, t0 := setupScoringEngine()

	cases := []struct {
		name        string
		earnedPts   float64
		maxPts      float64
		expectedBps int64
		expectedPct float64
		shouldPass  bool
	}{
		{
			name:        "boundary_round_up_passes_80",
			earnedPts:   79.995,
			maxPts:      100.0,
			expectedBps: 8000,
			expectedPct: 80.00,
			shouldPass:  true, // 79.995% rounds half-up to 8000 bps (80.00%) -> PASS
		},
		{
			name:        "boundary_round_down_fails_80",
			earnedPts:   79.994,
			maxPts:      100.0,
			expectedBps: 7999,
			expectedPct: 79.99,
			shouldPass:  false, // 79.994% rounds down to 7999 bps (79.99%) -> FAIL
		},
		{
			name:        "exact_half_rounds_up",
			earnedPts:   80.005,
			maxPts:      100.0,
			expectedBps: 8001,
			expectedPct: 80.01,
			shouldPass:  true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := workflowaction.ScoringEvaluationRequest{
				ExecutionID:       "ins_syn_round_" + tc.name,
				TenantID:          "ten_synthetic_alpha",
				TemplateID:        "chk_syn_pilot_plant_safety_v1",
				TemplateVersion:   "1.1.0",
				RuleMatrixVersion: "1.0.0",
				EvaluatedAt:       t0,
				Sections: []workflowaction.ScoredSection{
					{
						SectionID:    "sec_main",
						SectionTitle: "Main Section",
						Weight:       1.0,
						Questions: []workflowaction.ScoredQuestion{
							{
								QuestionID:    "q1",
								SectionID:     "sec_main",
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
				t.Fatalf("unexpected error: %v", err)
			}

			if res.BasisPoints != tc.expectedBps {
				t.Errorf("expected %d bps, got %d", tc.expectedBps, res.BasisPoints)
			}
			if res.RoundedScorePercent != tc.expectedPct {
				t.Errorf("expected %.2f percent, got %.2f", tc.expectedPct, res.RoundedScorePercent)
			}

			if tc.shouldPass && res.Outcome != workflowaction.OutcomePass {
				t.Errorf("expected PASS outcome, got %s", res.Outcome)
			}
			if !tc.shouldPass && res.Outcome != workflowaction.OutcomeFail {
				t.Errorf("expected FAIL outcome, got %s", res.Outcome)
			}
		})
	}
}

func TestScoring_CriticalFailPriorityFlag(t *testing.T) {
	engine, t0 := setupScoringEngine()

	// Very high score (95%), but contains a Critical Fail
	req := workflowaction.ScoringEvaluationRequest{
		ExecutionID:       "ins_syn_crit_05",
		TenantID:          "ten_synthetic_alpha",
		TemplateID:        "chk_syn_pilot_plant_safety_v1",
		TemplateVersion:   "1.1.0",
		RuleMatrixVersion: "1.0.0",
		EvaluatedAt:       t0,
		Sections: []workflowaction.ScoredSection{
			{
				SectionID:    "sec_confined",
				SectionTitle: "Confined Space",
				Weight:       1.0,
				Questions: []workflowaction.ScoredQuestion{
					{
						QuestionID:    "q_normal",
						SectionID:     "sec_confined",
						QuestionType:  "PASS_FAIL_NA_UNKNOWN",
						MaxPoints:     100.0,
						EarnedPoints:  95.0,
						ResponseState: "PASS",
					},
					{
						QuestionID:     "q_crit",
						SectionID:      "sec_confined",
						QuestionType:   "PASS_FAIL_NA_UNKNOWN",
						MaxPoints:      0.0, // Zero points, but critical flag
						EarnedPoints:   0.0,
						ResponseState:  "FAIL",
						IsCriticalFail: true,
						Notes:          "Uncalibrated atmospheric gas monitor used in toxic zone",
					},
				},
			},
		},
	}

	res, err := engine.Evaluate(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Score is NOT masked (numerical percentage reflects earned points)
	if res.BasisPoints != 9500 {
		t.Errorf("expected score not to be masked, got %d bps", res.BasisPoints)
	}

	// Priority flag locks outcome to NON_COMPLIANT_CRITICAL
	if !res.HasCriticalFail {
		t.Errorf("expected HasCriticalFail to be true")
	}
	if res.PassPredicates.NoCriticalFailPresent {
		t.Errorf("expected NoCriticalFailPresent predicate to fail (be false)")
	}
	if res.Outcome != workflowaction.OutcomeNonCompliantCritical {
		t.Errorf("expected NON_COMPLIANT_CRITICAL outcome, got %s", res.Outcome)
	}
}

func TestScoring_PredicateMatrix_AllThreeRequired(t *testing.T) {
	engine, t0 := setupScoringEngine()

	matrixCases := []struct {
		name          string
		scorePct      float64
		hasCritical   bool
		hasUnknown    bool
		expectedPred1 bool
		expectedPred2 bool
		expectedPred3 bool
		expectedAll   bool
		expectedOut   workflowaction.ComplianceOutcome
	}{
		{
			name:          "all_pass",
			scorePct:      85.0,
			hasCritical:   false,
			hasUnknown:    false,
			expectedPred1: true,
			expectedPred2: true,
			expectedPred3: true,
			expectedAll:   true,
			expectedOut:   workflowaction.OutcomePass,
		},
		{
			name:          "critical_fail_overrides_high_score",
			scorePct:      95.0,
			hasCritical:   true,
			hasUnknown:    false,
			expectedPred1: false,
			expectedPred2: true,
			expectedPred3: true,
			expectedAll:   false,
			expectedOut:   workflowaction.OutcomeNonCompliantCritical,
		},
		{
			name:          "unknown_blocks_high_score",
			scorePct:      90.0,
			hasCritical:   false,
			hasUnknown:    true,
			expectedPred1: true,
			expectedPred2: false,
			expectedPred3: true,
			expectedAll:   false,
			expectedOut:   workflowaction.OutcomeProvisionalPendingUnknownResolution,
		},
		{
			name:          "low_score_fails_cleanly",
			scorePct:      75.0,
			hasCritical:   false,
			hasUnknown:    false,
			expectedPred1: true,
			expectedPred2: true,
			expectedPred3: false,
			expectedAll:   false,
			expectedOut:   workflowaction.OutcomeFail,
		},
	}

	for _, tc := range matrixCases {
		t.Run(tc.name, func(t *testing.T) {
			qList := []workflowaction.ScoredQuestion{
				{
					QuestionID:    "q_base",
					SectionID:     "sec_main",
					QuestionType:  "NUMERIC_MEASUREMENT",
					MaxPoints:     100.0,
					EarnedPoints:  tc.scorePct,
					ResponseState: "PASS",
				},
			}
			if tc.hasCritical {
				qList = append(qList, workflowaction.ScoredQuestion{
					QuestionID:     "q_crit",
					SectionID:      "sec_main",
					QuestionType:   "PASS_FAIL_NA_UNKNOWN",
					IsCriticalFail: true,
				})
			}
			if tc.hasUnknown {
				qList = append(qList, workflowaction.ScoredQuestion{
					QuestionID:    "q_unk",
					SectionID:     "sec_main",
					QuestionType:  "PASS_FAIL_NA_UNKNOWN",
					MaxPoints:     10.0,
					ResponseState: "UNKNOWN",
				})
			}

			req := workflowaction.ScoringEvaluationRequest{
				ExecutionID:       "ins_matrix_" + tc.name,
				TenantID:          "ten_synthetic_alpha",
				TemplateID:        "chk_syn_pilot_plant_safety_v1",
				TemplateVersion:   "1.1.0",
				RuleMatrixVersion: "1.0.0",
				EvaluatedAt:       t0,
				Sections: []workflowaction.ScoredSection{
					{
						SectionID:    "sec_main",
						SectionTitle: "Main Section",
						Weight:       1.0,
						Questions:    qList,
					},
				},
			}

			res, err := engine.Evaluate(req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if res.PassPredicates.NoCriticalFailPresent != tc.expectedPred1 {
				t.Errorf("expected pred1 %v, got %v", tc.expectedPred1, res.PassPredicates.NoCriticalFailPresent)
			}
			if res.PassPredicates.NoUnresolvedUnknownQuarantined != tc.expectedPred2 {
				t.Errorf("expected pred2 %v, got %v", tc.expectedPred2, res.PassPredicates.NoUnresolvedUnknownQuarantined)
			}
			if res.PassPredicates.ScoreThresholdSatisfied != tc.expectedPred3 {
				t.Errorf("expected pred3 %v, got %v", tc.expectedPred3, res.PassPredicates.ScoreThresholdSatisfied)
			}
			if res.PassPredicates.AllPredicatesSatisfied != tc.expectedAll {
				t.Errorf("expected allPreds %v, got %v", tc.expectedAll, res.PassPredicates.AllPredicatesSatisfied)
			}
			if res.Outcome != tc.expectedOut {
				t.Errorf("expected outcome %s, got %s", tc.expectedOut, res.Outcome)
			}
		})
	}
}

func TestScoring_DynamicSectionWeightRedistribution(t *testing.T) {
	engine, t0 := setupScoringEngine()

	// 3 Sections:
	// Sec 1: Weight 0.50, Earned 100/100 (Ratio 1.0)
	// Sec 2: Weight 0.30, Earned 50/100  (Ratio 0.5)
	// Sec 3: Weight 0.20, ALL NA! (Inactive)
	// Re-distribution:
	// Active weights: Sec 1 + Sec 2 = 0.80
	// Sec 1 Effective Weight = 0.50 / 0.80 = 0.625 -> Contribution = 0.625 * 1.0 = 0.625
	// Sec 2 Effective Weight = 0.30 / 0.80 = 0.375 -> Contribution = 0.375 * 0.5 = 0.1875
	// Total Score = 0.625 + 0.1875 = 0.8125 -> 81.25% (8125 bps)
	req := workflowaction.ScoringEvaluationRequest{
		ExecutionID:       "ins_syn_redistribute_06",
		TenantID:          "ten_synthetic_alpha",
		TemplateID:        "chk_syn_pilot_plant_safety_v1",
		TemplateVersion:   "1.1.0",
		RuleMatrixVersion: "1.0.0",
		EvaluatedAt:       t0,
		Sections: []workflowaction.ScoredSection{
			{
				SectionID:    "sec_1",
				SectionTitle: "Section 1",
				Weight:       0.50,
				Questions: []workflowaction.ScoredQuestion{
					{
						QuestionID:    "q_1",
						SectionID:     "sec_1",
						QuestionType:  "PASS_FAIL_NA_UNKNOWN",
						MaxPoints:     100.0,
						EarnedPoints:  100.0,
						ResponseState: "PASS",
					},
				},
			},
			{
				SectionID:    "sec_2",
				SectionTitle: "Section 2",
				Weight:       0.30,
				Questions: []workflowaction.ScoredQuestion{
					{
						QuestionID:    "q_2",
						SectionID:     "sec_2",
						QuestionType:  "PASS_FAIL_NA_UNKNOWN",
						MaxPoints:     100.0,
						EarnedPoints:  50.0,
						ResponseState: "PASS",
					},
				},
			},
			{
				SectionID:    "sec_3",
				SectionTitle: "Section 3 (All NA)",
				Weight:       0.20,
				Questions: []workflowaction.ScoredQuestion{
					{
						QuestionID:    "q_3_na",
						SectionID:     "sec_3",
						QuestionType:  "PASS_FAIL_NA_UNKNOWN",
						MaxPoints:     100.0,
						EarnedPoints:  0.0,
						ResponseState: "NA",
						Notes:         "All equipment inactive",
					},
				},
			},
		},
	}

	res, err := engine.Evaluate(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.ActiveSectionsCount != 2 {
		t.Errorf("expected 2 active sections, got %d", res.ActiveSectionsCount)
	}
	if res.BasisPoints != 8125 {
		t.Errorf("expected 8125 basis points (81.25%%), got %d", res.BasisPoints)
	}
	if res.SectionResults[2].IsActive {
		t.Errorf("expected section 3 to be marked inactive")
	}
	if res.SectionResults[2].EffectiveWeight != 0.0 {
		t.Errorf("expected section 3 effective weight 0.0, got %f", res.SectionResults[2].EffectiveWeight)
	}
}

func TestScoring_AllSectionsInactiveEdgeCase(t *testing.T) {
	engine, t0 := setupScoringEngine()

	// Edge case: All items in all sections are NA
	req := workflowaction.ScoringEvaluationRequest{
		ExecutionID:       "ins_syn_all_na_07",
		TenantID:          "ten_synthetic_alpha",
		TemplateID:        "chk_syn_pilot_plant_safety_v1",
		TemplateVersion:   "1.1.0",
		RuleMatrixVersion: "1.0.0",
		EvaluatedAt:       t0,
		Sections: []workflowaction.ScoredSection{
			{
				SectionID:    "sec_empty",
				SectionTitle: "Empty Section",
				Weight:       1.0,
				Questions: []workflowaction.ScoredQuestion{
					{
						QuestionID:    "q_all_na",
						SectionID:     "sec_empty",
						QuestionType:  "PASS_FAIL_NA_UNKNOWN",
						MaxPoints:     100.0,
						EarnedPoints:  0.0,
						ResponseState: "NA",
					},
				},
			},
		},
	}

	res, err := engine.Evaluate(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.ActiveSectionsCount != 0 {
		t.Errorf("expected 0 active sections, got %d", res.ActiveSectionsCount)
	}
	if res.BasisPoints != 0 {
		t.Errorf("expected 0 basis points for all-NA inspection, got %d", res.BasisPoints)
	}
}

func TestScoring_ValidationErrors(t *testing.T) {
	engine, t0 := setupScoringEngine()

	validReq := workflowaction.ScoringEvaluationRequest{
		ExecutionID:       "ins_val_01",
		TenantID:          "ten_alpha",
		TemplateID:        "chk_alpha",
		TemplateVersion:   "1.0.0",
		RuleMatrixVersion: "1.0.0",
		EvaluatedAt:       t0,
		Sections: []workflowaction.ScoredSection{
			{
				SectionID: "sec_1",
				Weight:    1.0,
				Questions: []workflowaction.ScoredQuestion{
					{QuestionID: "q1", MaxPoints: 10, EarnedPoints: 10, ResponseState: "PASS"},
				},
			},
		},
	}

	// 1. Blank execution ID
	r1 := validReq
	r1.ExecutionID = ""
	if _, err := engine.Evaluate(r1); err != workflowaction.ErrBlankExecutionID {
		t.Errorf("expected ErrBlankExecutionID, got %v", err)
	}

	// 2. Blank tenant ID
	r2 := validReq
	r2.TenantID = ""
	if _, err := engine.Evaluate(r2); err != workflowaction.ErrBlankTenantID {
		t.Errorf("expected ErrBlankTenantID, got %v", err)
	}

	// 3. Blank template ID
	r3 := validReq
	r3.TemplateID = ""
	if _, err := engine.Evaluate(r3); err != workflowaction.ErrBlankTemplateID {
		t.Errorf("expected ErrBlankTemplateID, got %v", err)
	}

	// 4. Blank version
	r4 := validReq
	r4.TemplateVersion = ""
	if _, err := engine.Evaluate(r4); err != workflowaction.ErrBlankVersion {
		t.Errorf("expected ErrBlankVersion, got %v", err)
	}

	// 5. Empty sections
	r5 := validReq
	r5.Sections = nil
	if _, err := engine.Evaluate(r5); err != workflowaction.ErrEmptySections {
		t.Errorf("expected ErrEmptySections, got %v", err)
	}

	// 6. Invalid weights sum
	r6 := validReq
	r6.Sections = []workflowaction.ScoredSection{
		{SectionID: "s1", Weight: 0.50}, // sum = 0.50 != 1.0
	}
	if _, err := engine.Evaluate(r6); err != workflowaction.ErrInvalidWeightsSum {
		t.Errorf("expected ErrInvalidWeightsSum, got %v", err)
	}
}
