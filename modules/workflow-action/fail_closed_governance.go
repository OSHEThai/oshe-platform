// Package workflowaction coordinates operational inspection lifecycles, finding tracking, and corrective actions.
//
// PROVISIONAL GOVERNANCE DECLARATION (Issue #138 / V040-I027):
// Under approved Sole Human Owner decisions HDEC-V040-FOUNDATION-054 and HDEC-V040-SCORING-058,
// this file implements synthetic fail-closed priority hierarchy, unknown-quarantine handling,
// deferred manual override boundaries, autonomous AI boundaries, and append-only audit protection.
//
// Strict Fail-Closed Invariants:
//  1. Strict Priority Hierarchy: Unresolved critical failure (CF1) dominates unknown quarantine (U1)
//     and score thresholds, unconditionally blocking conclusive/passing transitions.
//  2. Fail-Closed Unknown Quarantine: Unresolved UNKNOWN responses quarantine transitions to conclusive states.
//  3. Deferred Manual Override Boundary: Under Gate H040-004, manual override authority is deferred human-owned.
//     Any override attempt is unconditionally denied and recorded in the audit ledger; zero authority is granted.
//  4. Autonomous AI Boundaries: AI agents are strictly prohibited from clearing critical flags, resolving
//     quarantines, executing overrides, or authorizing transitions to protected states.
//  5. Append-Only Audit Protection: Every qualification check, denial, override attempt, and quarantine
//     event is immutably recorded in the append-only ledger with monotonic sequence numbers.
package workflowaction

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// Priority constants defining the deterministic safety evaluation order.
const (
	PriorityOverrideDenial    = "MANUAL_OVERRIDE_DENIED"
	PriorityAIBoundaryDenial  = "AI_BOUNDARY_DENIED"
	PriorityCriticalFail      = "CRITICAL_FAIL_PRIORITY"
	PriorityUnknownQuarantine = "UNKNOWN_QUARANTINE"
	PriorityScoreThreshold    = "SCORE_THRESHOLD"
	PriorityStandardPermitted = "STANDARD_PERMITTED"
)

// FailClosedAuditEntry records an immutable, append-only governance evaluation or override attempt.
type FailClosedAuditEntry struct {
	Sequence              int64      `json:"sequence"`
	Timestamp             time.Time  `json:"timestamp"`
	TenantID              string     `json:"tenant_id"`
	TargetID              string     `json:"target_id"`
	TargetKind            string     `json:"target_kind"`
	Actor                 string     `json:"actor"`
	ActorRole             string     `json:"actor_role"`
	Action                string     `json:"action"`
	DenialCode            DenialCode `json:"denial_code"`
	Reason                string     `json:"reason"`
	RuleVersion           string     `json:"rule_version"`
	CorrelationID         string     `json:"correlation_id"`
	HasCriticalFail       bool       `json:"has_critical_fail"`
	HasQuarantinedUnknown bool       `json:"has_quarantined_unknown"`
	IsOverrideAttempt     bool       `json:"is_override_attempt"`
}

// FailClosedTransitionRequest provides full context for a fail-closed transition qualification.
type FailClosedTransitionRequest struct {
	TargetKind             string   `json:"target_kind"`
	TenantID               string   `json:"tenant_id"`
	TargetID               string   `json:"target_id"`
	CurrentState           string   `json:"current_state"`
	TargetState            string   `json:"target_state"`
	Actor                  string   `json:"actor"`
	ActorRole              string   `json:"actor_role"`
	RuleVersion            string   `json:"rule_version"`
	ConditionsMet          []string `json:"conditions_met"`
	EvidenceIDs            []string `json:"evidence_ids"`
	HasCriticalFail        bool     `json:"has_critical_fail"`
	CriticalFindingIDs     []string `json:"critical_finding_ids"`
	HasQuarantinedUnknown  bool     `json:"has_quarantined_unknown"`
	QuarantinedQuestionIDs []string `json:"quarantined_question_ids"`
	BasisPoints            int64    `json:"basis_points"`
	IsOverrideAttempt      bool     `json:"is_override_attempt"`
	OverrideRationale      string   `json:"override_rationale"`
	CorrelationID          string   `json:"correlation_id"`
}

// FailClosedGovernanceResult represents the deterministic result of a fail-closed qualification check.
type FailClosedGovernanceResult struct {
	Permitted       bool       `json:"permitted"`
	Result          RuleResult `json:"result"`
	DenialCode      DenialCode `json:"denial_code"`
	DenialReason    string     `json:"denial_reason"`
	PriorityApplied string     `json:"priority_applied"`
	AuditSequence   int64      `json:"audit_sequence"`
	EvaluatedAt     time.Time  `json:"evaluated_at"`
}

func isProtectedTargetState(state string) bool {
	norm := strings.ToUpper(strings.TrimSpace(state))
	switch norm {
	case "APPROVED", "CLOSED", "REMEDIATED", "PUBLISHED", "SCORED_PASS":
		return true
	default:
		return false
	}
}

// FailClosedGovernor coordinates deterministic fail-closed rule qualification, priority enforcement,
// unknown-quarantine state management, deferred override denial, and append-only audit logging.
type FailClosedGovernor struct {
	mu                 sync.RWMutex
	clock              Clock
	ruleMatrix         *RuleMatrix
	transitionCatalog  *TransitionCatalog
	auditLedger        []FailClosedAuditEntry
	quarantinedTargets map[string]map[string]map[string]bool // tenantID -> targetID -> questionID -> true
	criticalTargets    map[string]map[string]map[string]bool // tenantID -> targetID -> findingID -> true
}

// NewFailClosedGovernor constructs a new fail-closed governor instance.
func NewFailClosedGovernor(clock Clock, matrix *RuleMatrix, catalog *TransitionCatalog) *FailClosedGovernor {
	if clock == nil {
		clock = time.Now
	}
	if matrix == nil {
		matrix = DefaultRuleMatrix()
	}
	if catalog == nil {
		catalog = DefaultTransitionCatalog()
	}
	return &FailClosedGovernor{
		clock:              clock,
		ruleMatrix:         matrix,
		transitionCatalog:  catalog,
		auditLedger:        make([]FailClosedAuditEntry, 0),
		quarantinedTargets: make(map[string]map[string]map[string]bool),
		criticalTargets:    make(map[string]map[string]map[string]bool),
	}
}

// RegisterCriticalFinding associates an active critical failure with a target entity.
func (g *FailClosedGovernor) RegisterCriticalFinding(tenantID, targetID, findingID string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	tenantID = strings.TrimSpace(tenantID)
	targetID = strings.TrimSpace(targetID)
	findingID = strings.TrimSpace(findingID)
	if tenantID == "" || targetID == "" || findingID == "" {
		return
	}

	if _, ok := g.criticalTargets[tenantID]; !ok {
		g.criticalTargets[tenantID] = make(map[string]map[string]bool)
	}
	if _, ok := g.criticalTargets[tenantID][targetID]; !ok {
		g.criticalTargets[tenantID][targetID] = make(map[string]bool)
	}
	g.criticalTargets[tenantID][targetID][findingID] = true
}

// HasActiveCriticalFinding checks if a target entity has any unresolved critical failures.
func (g *FailClosedGovernor) HasActiveCriticalFinding(tenantID, targetID string) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.hasActiveCritical(tenantID, targetID)
}

func (g *FailClosedGovernor) hasActiveCritical(tenantID, targetID string) bool {
	tenantID = strings.TrimSpace(tenantID)
	targetID = strings.TrimSpace(targetID)
	if targets, ok := g.criticalTargets[tenantID]; ok {
		if findings, ok := targets[targetID]; ok {
			return len(findings) > 0
		}
	}
	return false
}

// RegisterQuarantinedQuestion associates an unreviewed UNKNOWN response with a target entity.
func (g *FailClosedGovernor) RegisterQuarantinedQuestion(tenantID, targetID, questionID string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	tenantID = strings.TrimSpace(tenantID)
	targetID = strings.TrimSpace(targetID)
	questionID = strings.TrimSpace(questionID)
	if tenantID == "" || targetID == "" || questionID == "" {
		return
	}

	if _, ok := g.quarantinedTargets[tenantID]; !ok {
		g.quarantinedTargets[tenantID] = make(map[string]map[string]bool)
	}
	if _, ok := g.quarantinedTargets[tenantID][targetID]; !ok {
		g.quarantinedTargets[tenantID][targetID] = make(map[string]bool)
	}
	g.quarantinedTargets[tenantID][targetID][questionID] = true
}

// HasActiveQuarantine checks if a target entity is under quarantine due to unresolved UNKNOWN responses.
func (g *FailClosedGovernor) HasActiveQuarantine(tenantID, targetID string) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.hasActiveQuarantine(tenantID, targetID)
}

func (g *FailClosedGovernor) hasActiveQuarantine(tenantID, targetID string) bool {
	tenantID = strings.TrimSpace(tenantID)
	targetID = strings.TrimSpace(targetID)
	if targets, ok := g.quarantinedTargets[tenantID]; ok {
		if questions, ok := targets[targetID]; ok {
			return len(questions) > 0
		}
	}
	return false
}

// ResolveUnknownQuestion allows an authorized human supervisor to resolve an UNKNOWN quarantine.
// Autonomous AI roles are strictly prohibited from resolving quarantines.
func (g *FailClosedGovernor) ResolveUnknownQuestion(
	tenantID, targetID, questionID, supervisor, supervisorRole, rationale string,
) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	tenantID = strings.TrimSpace(tenantID)
	targetID = strings.TrimSpace(targetID)
	questionID = strings.TrimSpace(questionID)
	rationale = strings.TrimSpace(rationale)

	if isAutonomousAgentRole(supervisorRole) || strings.Contains(strings.ToUpper(supervisor), "AI_AGENT") {
		return ErrAutonomousAIBoundary
	}
	if !isSupervisorRole(supervisorRole) {
		return ErrUnauthorizedActorClass
	}
	if rationale == "" {
		return ErrMissingClassificationRationale
	}

	now := g.clock().UTC()
	seq := int64(len(g.auditLedger) + 1)

	if targets, ok := g.quarantinedTargets[tenantID]; ok {
		if questions, ok := targets[targetID]; ok {
			delete(questions, questionID)
			if len(questions) == 0 {
				delete(targets, targetID)
			}
		}
	}

	g.auditLedger = append(g.auditLedger, FailClosedAuditEntry{
		Sequence:      seq,
		Timestamp:     now,
		TenantID:      tenantID,
		TargetID:      targetID,
		TargetKind:    TargetKindInspection,
		Actor:         supervisor,
		ActorRole:     supervisorRole,
		Action:        "UNKNOWN_RESOLVED",
		DenialCode:    DenialNone,
		Reason:        fmt.Sprintf("Quarantined question %s resolved by supervisor: %s", questionID, rationale),
		RuleVersion:   g.ruleMatrix.Version(),
		CorrelationID: fmt.Sprintf("res_unk_%d", now.UnixNano()),
	})

	return nil
}

// ClearCriticalFinding allows an authorized human supervisor to clear a verified critical finding.
func (g *FailClosedGovernor) ClearCriticalFinding(
	tenantID, targetID, findingID, supervisor, supervisorRole, rationale string,
) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	tenantID = strings.TrimSpace(tenantID)
	targetID = strings.TrimSpace(targetID)
	findingID = strings.TrimSpace(findingID)
	rationale = strings.TrimSpace(rationale)

	if isAutonomousAgentRole(supervisorRole) || strings.Contains(strings.ToUpper(supervisor), "AI_AGENT") {
		return ErrAutonomousAIBoundary
	}
	if !isSupervisorRole(supervisorRole) {
		return ErrUnauthorizedActorClass
	}
	if rationale == "" {
		return ErrMissingClassificationRationale
	}

	now := g.clock().UTC()
	seq := int64(len(g.auditLedger) + 1)

	if targets, ok := g.criticalTargets[tenantID]; ok {
		if findings, ok := targets[targetID]; ok {
			delete(findings, findingID)
			if len(findings) == 0 {
				delete(targets, targetID)
			}
		}
	}

	g.auditLedger = append(g.auditLedger, FailClosedAuditEntry{
		Sequence:      seq,
		Timestamp:     now,
		TenantID:      tenantID,
		TargetID:      targetID,
		TargetKind:    TargetKindFinding,
		Actor:         supervisor,
		ActorRole:     supervisorRole,
		Action:        "CRITICAL_CLEARED",
		DenialCode:    DenialNone,
		Reason:        fmt.Sprintf("Critical finding %s cleared by supervisor: %s", findingID, rationale),
		RuleVersion:   g.ruleMatrix.Version(),
		CorrelationID: fmt.Sprintf("clr_crit_%d", now.UnixNano()),
	})

	return nil
}

// Qualify evaluates an operational state transition through the full fail-closed priority hierarchy.
func (g *FailClosedGovernor) Qualify(req FailClosedTransitionRequest) FailClosedGovernanceResult {
	g.mu.Lock()
	defer g.mu.Unlock()

	now := g.clock().UTC()
	seq := int64(len(g.auditLedger) + 1)

	tenantID := strings.TrimSpace(req.TenantID)
	targetID := strings.TrimSpace(req.TargetID)
	ruleVer := strings.TrimSpace(req.RuleVersion)
	if ruleVer == "" {
		ruleVer = g.ruleMatrix.Version()
	}

	// -------------------------------------------------------------------------
	// Step 1: Manual Override Detection & Unconditional Denial (H040-004)
	// -------------------------------------------------------------------------
	if req.IsOverrideAttempt {
		reason := "manual override authority is deferred human-owned (H040-004): override attempt denied"
		if req.OverrideRationale != "" {
			reason = fmt.Sprintf("%s. Provided rationale: %s", reason, req.OverrideRationale)
		}

		g.auditLedger = append(g.auditLedger, FailClosedAuditEntry{
			Sequence:              seq,
			Timestamp:             now,
			TenantID:              tenantID,
			TargetID:              targetID,
			TargetKind:            req.TargetKind,
			Actor:                 req.Actor,
			ActorRole:             req.ActorRole,
			Action:                "OVERRIDE_ATTEMPT_DENIED",
			DenialCode:            DenialManualOverrideDeferred,
			Reason:                reason,
			RuleVersion:           ruleVer,
			CorrelationID:         req.CorrelationID,
			HasCriticalFail:       req.HasCriticalFail || len(req.CriticalFindingIDs) > 0,
			HasQuarantinedUnknown: req.HasQuarantinedUnknown || len(req.QuarantinedQuestionIDs) > 0,
			IsOverrideAttempt:     true,
		})

		return FailClosedGovernanceResult{
			Permitted:       false,
			Result:          RuleResultDenied,
			DenialCode:      DenialManualOverrideDeferred,
			DenialReason:    reason,
			PriorityApplied: PriorityOverrideDenial,
			AuditSequence:   seq,
			EvaluatedAt:     now,
		}
	}

	// -------------------------------------------------------------------------
	// Step 2: Autonomous AI Boundary Enforcement
	// -------------------------------------------------------------------------
	if isAutonomousAgentRole(req.ActorRole) && isProtectedTargetState(req.TargetState) {
		reason := fmt.Sprintf("autonomous AI actor %q (%s) is prohibited from authorizing transition to protected state %s",
			req.Actor, req.ActorRole, req.TargetState)

		g.auditLedger = append(g.auditLedger, FailClosedAuditEntry{
			Sequence:              seq,
			Timestamp:             now,
			TenantID:              tenantID,
			TargetID:              targetID,
			TargetKind:            req.TargetKind,
			Actor:                 req.Actor,
			ActorRole:             req.ActorRole,
			Action:                "AI_BOUNDARY_BLOCKED",
			DenialCode:            DenialAutonomousAIBoundary,
			Reason:                reason,
			RuleVersion:           ruleVer,
			CorrelationID:         req.CorrelationID,
			HasCriticalFail:       req.HasCriticalFail,
			HasQuarantinedUnknown: req.HasQuarantinedUnknown,
		})

		return FailClosedGovernanceResult{
			Permitted:       false,
			Result:          RuleResultDenied,
			DenialCode:      DenialAutonomousAIBoundary,
			DenialReason:    reason,
			PriorityApplied: PriorityAIBoundaryDenial,
			AuditSequence:   seq,
			EvaluatedAt:     now,
		}
	}

	// -------------------------------------------------------------------------
	// Step 3: Critical-Fail Priority Evaluation (CF1 Dominance)
	// -------------------------------------------------------------------------
	hasActiveCritical := req.HasCriticalFail ||
		len(req.CriticalFindingIDs) > 0 ||
		g.hasActiveCritical(tenantID, targetID)

	if hasActiveCritical && isProtectedTargetState(req.TargetState) {
		findingList := ""
		if len(req.CriticalFindingIDs) > 0 {
			findingList = fmt.Sprintf(" (findings: %s)", strings.Join(req.CriticalFindingIDs, ", "))
		}
		reason := fmt.Sprintf("transition blocked under critical-fail priority: unresolved critical failure active%s", findingList)

		g.auditLedger = append(g.auditLedger, FailClosedAuditEntry{
			Sequence:              seq,
			Timestamp:             now,
			TenantID:              tenantID,
			TargetID:              targetID,
			TargetKind:            req.TargetKind,
			Actor:                 req.Actor,
			ActorRole:             req.ActorRole,
			Action:                "CRITICAL_FAIL_BLOCKED",
			DenialCode:            DenialCriticalFailActive,
			Reason:                reason,
			RuleVersion:           ruleVer,
			CorrelationID:         req.CorrelationID,
			HasCriticalFail:       true,
			HasQuarantinedUnknown: req.HasQuarantinedUnknown,
		})

		return FailClosedGovernanceResult{
			Permitted:       false,
			Result:          RuleResultDenied,
			DenialCode:      DenialCriticalFailActive,
			DenialReason:    reason,
			PriorityApplied: PriorityCriticalFail,
			AuditSequence:   seq,
			EvaluatedAt:     now,
		}
	}

	// -------------------------------------------------------------------------
	// Step 4: Unknown-Quarantine Evaluation (U1 Denominator Quarantine)
	// -------------------------------------------------------------------------
	hasActiveQuarantine := req.HasQuarantinedUnknown ||
		len(req.QuarantinedQuestionIDs) > 0 ||
		g.hasActiveQuarantine(tenantID, targetID)

	if hasActiveQuarantine && isProtectedTargetState(req.TargetState) {
		questionList := ""
		if len(req.QuarantinedQuestionIDs) > 0 {
			questionList = fmt.Sprintf(" (questions: %s)", strings.Join(req.QuarantinedQuestionIDs, ", "))
		}
		reason := fmt.Sprintf("transition quarantined: unreviewed UNKNOWN responses pending human supervisory resolution%s", questionList)

		g.auditLedger = append(g.auditLedger, FailClosedAuditEntry{
			Sequence:              seq,
			Timestamp:             now,
			TenantID:              tenantID,
			TargetID:              targetID,
			TargetKind:            req.TargetKind,
			Actor:                 req.Actor,
			ActorRole:             req.ActorRole,
			Action:                "UNKNOWN_QUARANTINE_RECORDED",
			DenialCode:            DenialUnknownQuarantined,
			Reason:                reason,
			RuleVersion:           ruleVer,
			CorrelationID:         req.CorrelationID,
			HasCriticalFail:       false,
			HasQuarantinedUnknown: true,
		})

		return FailClosedGovernanceResult{
			Permitted:       false,
			Result:          RuleResultQuarantined,
			DenialCode:      DenialUnknownQuarantined,
			DenialReason:    reason,
			PriorityApplied: PriorityUnknownQuarantine,
			AuditSequence:   seq,
			EvaluatedAt:     now,
		}
	}

	// -------------------------------------------------------------------------
	// Step 5: Score Threshold Check (if scored inspection)
	// -------------------------------------------------------------------------
	if req.BasisPoints > 0 && req.BasisPoints < PassingThresholdBasisPoints && isProtectedTargetState(req.TargetState) {
		reason := fmt.Sprintf("score below passing threshold: required %d bps (80.00%%), got %d bps", PassingThresholdBasisPoints, req.BasisPoints)

		g.auditLedger = append(g.auditLedger, FailClosedAuditEntry{
			Sequence:              seq,
			Timestamp:             now,
			TenantID:              tenantID,
			TargetID:              targetID,
			TargetKind:            req.TargetKind,
			Actor:                 req.Actor,
			ActorRole:             req.ActorRole,
			Action:                "SCORE_BELOW_THRESHOLD",
			DenialCode:            DenialMissingCondition,
			Reason:                reason,
			RuleVersion:           ruleVer,
			CorrelationID:         req.CorrelationID,
			HasCriticalFail:       false,
			HasQuarantinedUnknown: false,
		})

		return FailClosedGovernanceResult{
			Permitted:       false,
			Result:          RuleResultDenied,
			DenialCode:      DenialMissingCondition,
			DenialReason:    reason,
			PriorityApplied: PriorityScoreThreshold,
			AuditSequence:   seq,
			EvaluatedAt:     now,
		}
	}

	// -------------------------------------------------------------------------
	// Step 6: Underlying TransitionCatalog / RuleMatrix Qualification
	// -------------------------------------------------------------------------
	condMap := make(map[string]bool)
	for _, c := range req.ConditionsMet {
		condMap[c] = true
	}

	catReq := TransitionQualificationRequest{
		TargetKind:      req.TargetKind,
		TenantID:        tenantID,
		TargetID:        targetID,
		FromState:       req.CurrentState,
		ToState:         req.TargetState,
		Actor:           req.Actor,
		ActorRole:       req.ActorRole,
		EvidenceIDs:     req.EvidenceIDs,
		InputConditions: condMap,
		MatrixVersion:   ruleVer,
	}

	qualRes := g.transitionCatalog.Qualify(g.ruleMatrix, catReq)

	var permitted bool
	var result RuleResult
	var denialCode DenialCode
	var reason string
	var priorityApplied string
	var actionCode string

	if qualRes.Allowed {
		permitted = true
		result = RuleResultPermitted
		denialCode = DenialNone
		reason = qualRes.Explanation
		priorityApplied = PriorityStandardPermitted
		actionCode = "TRANSITION_PERMITTED"
	} else {
		permitted = false
		result = RuleResultDenied
		denialCode = qualRes.DenialReason
		reason = qualRes.Explanation
		priorityApplied = string(qualRes.DenialReason)
		actionCode = "TRANSITION_DENIED"
	}

	g.auditLedger = append(g.auditLedger, FailClosedAuditEntry{
		Sequence:      seq,
		Timestamp:     now,
		TenantID:      tenantID,
		TargetID:      targetID,
		TargetKind:    req.TargetKind,
		Actor:         req.Actor,
		ActorRole:     req.ActorRole,
		Action:        actionCode,
		DenialCode:    denialCode,
		Reason:        reason,
		RuleVersion:   ruleVer,
		CorrelationID: req.CorrelationID,
	})

	return FailClosedGovernanceResult{
		Permitted:       permitted,
		Result:          result,
		DenialCode:      denialCode,
		DenialReason:    reason,
		PriorityApplied: priorityApplied,
		AuditSequence:   seq,
		EvaluatedAt:     now,
	}
}

// AuditHistory returns an append-only audit trail for a specific target entity.
func (g *FailClosedGovernor) AuditHistory(tenantID, targetID string) []FailClosedAuditEntry {
	g.mu.RLock()
	defer g.mu.RUnlock()

	tenantID = strings.TrimSpace(tenantID)
	targetID = strings.TrimSpace(targetID)

	var history []FailClosedAuditEntry
	for _, entry := range g.auditLedger {
		if (tenantID == "" || entry.TenantID == tenantID) &&
			(targetID == "" || entry.TargetID == targetID) {
			history = append(history, entry)
		}
	}
	return history
}

// TotalAuditCount returns the total number of entries committed to the audit ledger.
func (g *FailClosedGovernor) TotalAuditCount() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return len(g.auditLedger)
}
