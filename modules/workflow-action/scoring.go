// Package workflowaction coordinates operational inspection lifecycles, finding tracking, and corrective actions.
//
// PROVISIONAL GOVERNANCE DECLARATION (Issue #137 / V040-I026 / HDEC-V040-SCORING-058):
// Under approved Sole Human Owner decision HDEC-V040-SCORING-058, this file implements
// the deterministic operational scoring engine for Milestone v0.4.0 OSHE Inspect Private Alpha.
//
// Strict Scoring Engine Invariants:
//  1. Model 2 Weighted Normalized Scoring (MODEL_2_WEIGHTED): Section scores are computed as
//     earned points over active effective denominator and multiplied by normalized section weights.
//  2. Not Applicable (NA) Denominator Exclusions: NA responses subtract available points from
//     the denominator with zero negative score penalty.
//  3. Unknown Quarantine (U1_QUARANTINE_DENOMINATOR): UNKNOWN responses quarantine points from
//     the denominator and block passing determinations pending supervisory resolution.
//  4. Fixed-Point Basis Points & Round Half Up (R1_ROUND_HALF_UP): Calculated scores map to integer
//     basis points (1% = 100 bps; 80.00% = 8000 bps; 100.00% = 10000 bps) with half-up rounding.
//  5. Critical Fail Priority Flag (CF1_PRIORITY_FLAG): Critical fail triggers lock the compliance outcome
//     to NON_COMPLIANT_CRITICAL while continuing to report numerical score without masking.
//  6. Three Mandatory Pass Predicates: Passing requires (1) no critical-fail condition, (2) no
//     unresolved quarantined UNKNOWN responses, and (3) score >= 80.00% (8000 bps).
//  7. Dynamic Section Weight Redistribution: If all questions in a section evaluate to NA or UNKNOWN,
//     the section weight is reallocated proportionally across active sections.
//  8. Pinned Version Traceability: All evaluations record template version, rule matrix version,
//     and formula version.
package workflowaction

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
)

// Scoring Model and Policy Enumerations approved under HDEC-V040-SCORING-058.
const (
	ScoringModelSelected        = "MODEL_2_WEIGHTED"
	UnknownHandlingSelected     = "U1_QUARANTINE_DENOMINATOR"
	RoundingRuleSelected        = "R1_ROUND_HALF_UP"
	CriticalFailPolicySelected  = "CF1_PRIORITY_FLAG"
	FormulaVersionSelected      = "v0.4.0-HDEC-058"
	PassingThresholdBasisPoints = int64(8000)  // 80.00%
	MaximumBasisPoints          = int64(10000) // 100.00%
)

// ComplianceOutcome represents the definitive evaluation outcome of an inspection.
type ComplianceOutcome string

const (
	OutcomePass                                ComplianceOutcome = "PASS"
	OutcomeFail                                ComplianceOutcome = "FAIL"
	OutcomeNonCompliantCritical                ComplianceOutcome = "NON_COMPLIANT_CRITICAL"
	OutcomeProvisionalPendingUnknownResolution ComplianceOutcome = "PROVISIONAL_PENDING_UNKNOWN_RESOLUTION"
)

var (
	ErrBlankExecutionID  = errors.New("execution ID cannot be blank")
	ErrBlankTemplateID   = errors.New("template ID cannot be blank")
	ErrBlankVersion      = errors.New("template version and rule matrix version cannot be blank")
	ErrEmptySections     = errors.New("scoring evaluation requires at least one section")
	ErrInvalidWeightsSum = errors.New("sum of section weights must equal 1.0 (100%)")
)

// ScoredQuestion represents an individual inspection response evaluated for scoring.
type ScoredQuestion struct {
	QuestionID      string  `json:"question_id"`
	SectionID       string  `json:"section_id"`
	QuestionType    string  `json:"question_type"` // e.g., PASS_FAIL_NA_UNKNOWN, SINGLE_CHOICE, NUMERIC_MEASUREMENT, TEXT_NOTE, EVIDENCE_ATTACHMENT
	MaxPoints       float64 `json:"max_points"`
	EarnedPoints    float64 `json:"earned_points"`
	ResponseState   string  `json:"response_state"` // PASS, FAIL, NA, UNKNOWN, or option ID
	IsCriticalFail  bool    `json:"is_critical_fail"`
	IsExcluded      bool    `json:"is_excluded"` // true for non-scored types (TEXT_NOTE, EVIDENCE_ATTACHMENT) or conditional branch exclusions
	ExclusionReason string  `json:"exclusion_reason,omitempty"`
	Notes           string  `json:"notes,omitempty"`
}

// ScoredSection represents a category or section containing scored questions and an assigned weight.
type ScoredSection struct {
	SectionID    string           `json:"section_id"`
	SectionTitle string           `json:"section_title"`
	Weight       float64          `json:"weight"` // Normalized section weight (sum over all sections = 1.0)
	Questions    []ScoredQuestion `json:"questions"`
}

// ScoringEvaluationRequest encapsulates all contextual inputs for a deterministic scoring evaluation.
type ScoringEvaluationRequest struct {
	ExecutionID       string          `json:"execution_id"`
	TenantID          string          `json:"tenant_id"`
	TemplateID        string          `json:"template_id"`
	TemplateVersion   string          `json:"template_version"`
	RuleMatrixVersion string          `json:"rule_matrix_version"`
	Sections          []ScoredSection `json:"sections"`
	EvaluatedAt       time.Time       `json:"evaluated_at"`
}

// SectionScoreResult contains the granular evaluation math for a single section.
type SectionScoreResult struct {
	SectionID                 string  `json:"section_id"`
	SectionTitle              string  `json:"section_title"`
	OriginalWeight            float64 `json:"original_weight"`
	EffectiveWeight           float64 `json:"effective_weight"`
	TotalMaxPoints            float64 `json:"total_max_points"`
	EarnedPoints              float64 `json:"earned_points"`
	NAPoints                  float64 `json:"na_points"`
	QuarantinedUnknownPoints  float64 `json:"quarantined_unknown_points"`
	EffectiveDenominator      float64 `json:"effective_denominator"`
	SectionScoreRatio         float64 `json:"section_score_ratio"`
	WeightedContributionScore float64 `json:"weighted_contribution_score"`
	IsActive                  bool    `json:"is_active"` // false if all items are NA or UNKNOWN
}

// PassPredicateResult details the evaluation of the three mandatory pass predicates.
type PassPredicateResult struct {
	NoCriticalFailPresent          bool `json:"no_critical_fail_present"`
	NoUnresolvedUnknownQuarantined bool `json:"no_unresolved_unknown_quarantined"`
	ScoreThresholdSatisfied        bool `json:"score_threshold_satisfied"` // >= 80.00% (8000 bps)
	AllPredicatesSatisfied         bool `json:"all_predicates_satisfied"`
}

// ScoringEvaluationResult is the comprehensive, deterministic output of the scoring engine.
type ScoringEvaluationResult struct {
	ExecutionID                 string               `json:"execution_id"`
	TenantID                    string               `json:"tenant_id"`
	TemplateID                  string               `json:"template_id"`
	TemplateVersion             string               `json:"template_version"`
	RuleMatrixVersion           string               `json:"rule_matrix_version"`
	FormulaVersion              string               `json:"formula_version"`
	ScoringModel                string               `json:"scoring_model"`
	UnknownHandling             string               `json:"unknown_handling"`
	RoundingRule                string               `json:"rounding_rule"`
	CriticalFailPolicy          string               `json:"critical_fail_policy"`
	RawScorePercent             float64              `json:"raw_score_percent"`
	BasisPoints                 int64                `json:"basis_points"`
	RoundedScorePercent         float64              `json:"rounded_score_percent"`
	DisplayScore                string               `json:"display_score"`
	HasCriticalFail             bool                 `json:"has_critical_fail"`
	CriticalFailQuestions       []string             `json:"critical_fail_questions"`
	HasQuarantinedUnknown       bool                 `json:"has_quarantined_unknown"`
	QuarantinedUnknownQuestions []string             `json:"quarantined_unknown_questions"`
	PassPredicates              PassPredicateResult  `json:"pass_predicates"`
	Outcome                     ComplianceOutcome    `json:"outcome"`
	SectionResults              []SectionScoreResult `json:"section_results"`
	ActiveSectionsCount         int                  `json:"active_sections_count"`
	TotalQuestionsEvaluated     int                  `json:"total_questions_evaluated"`
	TotalQuestionsExcluded      int                  `json:"total_questions_excluded"`
	TraceabilityKey             string               `json:"traceability_key"`
	EvaluatedAt                 time.Time            `json:"evaluated_at"`
}

// DeterministicScoringEngine coordinates operational compliance scoring.
type DeterministicScoringEngine struct {
	clock Clock
}

// NewDeterministicScoringEngine constructs an engine with an injectable clock.
func NewDeterministicScoringEngine(clock Clock) *DeterministicScoringEngine {
	if clock == nil {
		clock = time.Now
	}
	return &DeterministicScoringEngine{clock: clock}
}

// Evaluate executes the deterministic scoring calculation according to HDEC-V040-SCORING-058.
func (e *DeterministicScoringEngine) Evaluate(req ScoringEvaluationRequest) (ScoringEvaluationResult, error) {
	if strings.TrimSpace(req.ExecutionID) == "" {
		return ScoringEvaluationResult{}, ErrBlankExecutionID
	}
	if strings.TrimSpace(req.TenantID) == "" {
		return ScoringEvaluationResult{}, ErrBlankTenantID
	}
	if strings.TrimSpace(req.TemplateID) == "" {
		return ScoringEvaluationResult{}, ErrBlankTemplateID
	}
	if strings.TrimSpace(req.TemplateVersion) == "" || strings.TrimSpace(req.RuleMatrixVersion) == "" {
		return ScoringEvaluationResult{}, ErrBlankVersion
	}
	if len(req.Sections) == 0 {
		return ScoringEvaluationResult{}, ErrEmptySections
	}

	// Validate sum of section weights equals 1.0 (allow minor float epsilon 0.001)
	var weightSum float64
	for _, sec := range req.Sections {
		weightSum += sec.Weight
	}
	if math.Abs(weightSum-1.0) > 0.001 {
		return ScoringEvaluationResult{}, ErrInvalidWeightsSum
	}

	evaluatedAt := req.EvaluatedAt
	if evaluatedAt.IsZero() {
		evaluatedAt = e.clock().UTC()
	}

	var hasCriticalFail bool
	var criticalQuestions []string
	var hasUnknown bool
	var unknownQuestions []string

	totalQuestions := 0
	excludedQuestions := 0

	// First pass: evaluate section raw totals and identify active sections
	sectionIntermediates := make([]SectionScoreResult, len(req.Sections))
	var activeWeightSum float64

	for i, sec := range req.Sections {
		res := SectionScoreResult{
			SectionID:       sec.SectionID,
			SectionTitle:    sec.SectionTitle,
			OriginalWeight:  sec.Weight,
			EffectiveWeight: sec.Weight,
		}

		for _, q := range sec.Questions {
			totalQuestions++

			// Track critical fails
			if q.IsCriticalFail {
				hasCriticalFail = true
				criticalQuestions = append(criticalQuestions, q.QuestionID)
			}

			// Non-scored types (TEXT_NOTE, EVIDENCE_ATTACHMENT) or conditional exclusions are excluded from denominator
			if q.IsExcluded || q.QuestionType == "TEXT_NOTE" || q.QuestionType == "EVIDENCE_ATTACHMENT" {
				excludedQuestions++
				continue
			}

			res.TotalMaxPoints += q.MaxPoints

			switch strings.ToUpper(strings.TrimSpace(q.ResponseState)) {
			case "NA", "NOT_APPLICABLE":
				// Excluded from denominator with zero negative impact
				res.NAPoints += q.MaxPoints

			case "UNKNOWN":
				// Quarantined from denominator under U1_QUARANTINE_DENOMINATOR
				hasUnknown = true
				unknownQuestions = append(unknownQuestions, q.QuestionID)
				res.QuarantinedUnknownPoints += q.MaxPoints

			default:
				// Regular scored responses (PASS, FAIL, SINGLE_CHOICE, MULTI_CHOICE, NUMERIC_MEASUREMENT)
				res.EarnedPoints += q.EarnedPoints
			}
		}

		// Effective Denominator = TotalMax - NA - QuarantinedUnknown
		res.EffectiveDenominator = res.TotalMaxPoints - res.NAPoints - res.QuarantinedUnknownPoints
		if res.EffectiveDenominator > 0 {
			res.IsActive = true
			res.SectionScoreRatio = res.EarnedPoints / res.EffectiveDenominator
			activeWeightSum += sec.Weight
		} else {
			// All items were NA or UNKNOWN: section is inactive for scoring
			res.IsActive = false
			res.SectionScoreRatio = 0.0
		}

		sectionIntermediates[i] = res
	}

	// Second pass: Dynamic Section Weight Redistribution & Weighted Sum (MODEL_2_WEIGHTED)
	var rawWeightedScore float64
	activeCount := 0

	for i := range sectionIntermediates {
		sec := &sectionIntermediates[i]
		if sec.IsActive {
			activeCount++
			if activeWeightSum > 0 {
				// Re-allocate weight proportionally across active sections
				sec.EffectiveWeight = sec.OriginalWeight / activeWeightSum
			} else {
				sec.EffectiveWeight = 0.0
			}
			sec.WeightedContributionScore = sec.EffectiveWeight * sec.SectionScoreRatio
			rawWeightedScore += sec.WeightedContributionScore
		} else {
			sec.EffectiveWeight = 0.0
			sec.WeightedContributionScore = 0.0
		}
	}

	// Calculate raw percentage: 0.0 to 100.0
	rawScorePercent := rawWeightedScore * 100.0
	if rawScorePercent > 100.0 {
		rawScorePercent = 100.0
	}
	if rawScorePercent < 0.0 {
		rawScorePercent = 0.0
	}

	// Rounding Arithmetic: R1_ROUND_HALF_UP via integer basis points
	// Basis Points: 0 to 10000. 1% = 100 bps.
	// Add tiny epsilon (1e-9) to handle binary float64 precision boundaries (e.g. 80.005 -> 8000.5)
	rawBps := rawWeightedScore * float64(MaximumBasisPoints)
	basisPoints := int64(math.Floor(rawBps + 0.5 + 1e-9))
	if basisPoints > MaximumBasisPoints {
		basisPoints = MaximumBasisPoints
	}
	if basisPoints < 0 {
		basisPoints = 0
	}

	// Rounded percentage: basisPoints / 100.0
	roundedPercent := float64(basisPoints) / 100.0

	// Format display score: formatted to one decimal place (80.0%)
	displayScore := fmt.Sprintf("%.1f%%", roundedPercent)

	// Evaluate the Three Mandatory Pass Predicates
	pred1 := !hasCriticalFail
	pred2 := !hasUnknown
	pred3 := basisPoints >= PassingThresholdBasisPoints
	allPredsPass := pred1 && pred2 && pred3

	// Determine Compliance Outcome
	var outcome ComplianceOutcome
	if hasCriticalFail {
		// CF1_PRIORITY_FLAG: Critical fail locks outcome to NON_COMPLIANT_CRITICAL
		outcome = OutcomeNonCompliantCritical
	} else if hasUnknown {
		// U1_QUARANTINE_DENOMINATOR: Unresolved UNKNOWN locks outcome to PROVISIONAL
		outcome = OutcomeProvisionalPendingUnknownResolution
	} else if allPredsPass {
		outcome = OutcomePass
	} else {
		outcome = OutcomeFail
	}

	traceKey := fmt.Sprintf("trace_%s_%s_%s_%d",
		req.TemplateID,
		req.TemplateVersion,
		req.RuleMatrixVersion,
		evaluatedAt.Unix(),
	)

	return ScoringEvaluationResult{
		ExecutionID:                 req.ExecutionID,
		TenantID:                    req.TenantID,
		TemplateID:                  req.TemplateID,
		TemplateVersion:             req.TemplateVersion,
		RuleMatrixVersion:           req.RuleMatrixVersion,
		FormulaVersion:              FormulaVersionSelected,
		ScoringModel:                ScoringModelSelected,
		UnknownHandling:             UnknownHandlingSelected,
		RoundingRule:                RoundingRuleSelected,
		CriticalFailPolicy:          CriticalFailPolicySelected,
		RawScorePercent:             rawScorePercent,
		BasisPoints:                 basisPoints,
		RoundedScorePercent:         roundedPercent,
		DisplayScore:                displayScore,
		HasCriticalFail:             hasCriticalFail,
		CriticalFailQuestions:       criticalQuestions,
		HasQuarantinedUnknown:       hasUnknown,
		QuarantinedUnknownQuestions: unknownQuestions,
		PassPredicates: PassPredicateResult{
			NoCriticalFailPresent:          pred1,
			NoUnresolvedUnknownQuarantined: pred2,
			ScoreThresholdSatisfied:        pred3,
			AllPredicatesSatisfied:         allPredsPass,
		},
		Outcome:                 outcome,
		SectionResults:          sectionIntermediates,
		ActiveSectionsCount:     activeCount,
		TotalQuestionsEvaluated: totalQuestions,
		TotalQuestionsExcluded:  excludedQuestions,
		TraceabilityKey:         traceKey,
		EvaluatedAt:             evaluatedAt,
	}, nil
}
