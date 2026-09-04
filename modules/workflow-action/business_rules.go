package workflowaction

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

const (
	// CurrentRuleMatrixVersion defines the canonical version for the business rule matrix.
	CurrentRuleMatrixVersion = "1.0.0"

	// Target kinds
	TargetKindTemplate = "TEMPLATE"
	TargetKindWorkflow = "WORKFLOW"
	TargetKindAction   = "ACTION"
)

// RuleResult defines the deterministic output result of a business rule.
type RuleResult string

const (
	RuleResultPermitted       RuleResult = "PERMITTED"
	RuleResultRequiresApproval RuleResult = "REQUIRES_APPROVAL"
	RuleResultDenied          RuleResult = "DENIED"
)

// FailureBehavior defines how a failed rule evaluation must behave.
type FailureBehavior string

const (
	FailureFailClosed FailureBehavior = "FAIL_CLOSED"
	FailureReject     FailureBehavior = "REJECT"
)

// DenialCode defines stable, typed denial reasons for qualification failures.
type DenialCode string

const (
	DenialNone                DenialCode = "NONE"
	DenialUnregisteredRule    DenialCode = "UNREGISTERED_RULE"
	DenialIncompatibleVersion DenialCode = "INCOMPATIBLE_MATRIX_VERSION"
	DenialMissingCondition    DenialCode = "MISSING_REQUIRED_CONDITION"
	DenialMissingEvidence     DenialCode = "MISSING_REQUIRED_EVIDENCE"
	DenialUnauthorizedActor   DenialCode = "UNAUTHORIZED_ACTOR_CLASS"
	DenialArchivedTarget      DenialCode = "ARCHIVED_TARGET_MUTATION_DENIED"
	DenialInvalidTransition   DenialCode = "INVALID_CATALOG_TRANSITION"
	DenialBlankIdentifier     DenialCode = "BLANK_IDENTIFIER"
)

var (
	ErrBlankRuleID               = errors.New("rule ID cannot be blank")
	ErrDuplicateRuleID           = errors.New("duplicate business rule ID")
	ErrUnregisteredRule          = errors.New("business rule is not registered in matrix")
	ErrIncompatibleMatrixVersion = errors.New("incompatible business rule matrix version")
	ErrMissingCondition          = errors.New("required input condition is missing or unfulfilled")
	ErrMissingEvidence           = errors.New("required evidence is missing")
	ErrUnauthorizedActorClass    = errors.New("actor class or role is not authorized for rule")
	ErrArchivedTargetMutation    = errors.New("archived target cannot undergo transition")
	ErrInvalidCatalogTransition  = errors.New("transition is not permitted by transition catalog")
	ErrBlankIdentifier           = errors.New("required identifier cannot be blank")
)

// EvidenceRequirement specifies evidence constraints for a business rule.
type EvidenceRequirement struct {
	Required bool `json:"required"`
	MinCount int  `json:"min_count"`
}

// BusinessRule defines an authoritative, versioned business rule in the matrix.
type BusinessRule struct {
	RuleID              string              `json:"rule_id"`
	OwnerRole           string              `json:"owner_role"`
	RequiredConditions  []string            `json:"required_conditions"`
	DeterministicResult RuleResult          `json:"deterministic_result"`
	FailureBehavior     FailureBehavior     `json:"failure_behavior"`
	Evidence            EvidenceRequirement `json:"evidence"`
	TraceabilityKey     string              `json:"traceability_key"`
	Description         string              `json:"description"`
}

// RuleMatrix is an in-memory, thread-safe, versioned repository of business rules.
type RuleMatrix struct {
	mu      sync.RWMutex
	version string
	rules   map[string]BusinessRule
}

// NewRuleMatrix initializes a RuleMatrix with canonical version 1.0.0.
func NewRuleMatrix(version string) *RuleMatrix {
	v := strings.TrimSpace(version)
	if v == "" {
		v = CurrentRuleMatrixVersion
	}
	return &RuleMatrix{
		version: v,
		rules:   make(map[string]BusinessRule),
	}
}

// Version returns the matrix schema/version.
func (m *RuleMatrix) Version() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.version
}

// RegisterRule registers a rule in the matrix. Fails on blank ID or duplicate ID.
func (m *RuleMatrix) RegisterRule(rule BusinessRule) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	id := strings.TrimSpace(rule.RuleID)
	if id == "" {
		return ErrBlankRuleID
	}
	if _, exists := m.rules[id]; exists {
		return ErrDuplicateRuleID
	}

	r := rule
	r.RuleID = id
	r.OwnerRole = strings.TrimSpace(rule.OwnerRole)
	r.TraceabilityKey = strings.TrimSpace(rule.TraceabilityKey)
	if r.DeterministicResult == "" {
		r.DeterministicResult = RuleResultPermitted
	}
	if r.FailureBehavior == "" {
		r.FailureBehavior = FailureFailClosed
	}

	m.rules[id] = r
	return nil
}

// GetRule retrieves a rule by ID.
func (m *RuleMatrix) GetRule(ruleID string) (BusinessRule, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	rule, exists := m.rules[strings.TrimSpace(ruleID)]
	if !exists {
		return BusinessRule{}, ErrUnregisteredRule
	}
	return rule, nil
}

// TransitionKey identifies a discrete transition within a target domain.
type TransitionKey struct {
	TargetKind string
	FromState  string
	ToState    string
}

// TransitionCatalog maps domain transitions to required business rules.
type TransitionCatalog struct {
	mu          sync.RWMutex
	transitions map[TransitionKey]string // maps TransitionKey -> RuleID
}

// NewTransitionCatalog initializes an empty transition catalog.
func NewTransitionCatalog() *TransitionCatalog {
	return &TransitionCatalog{
		transitions: make(map[TransitionKey]string),
	}
}

// RegisterTransition binds a target transition to a specific business rule ID.
func (c *TransitionCatalog) RegisterTransition(targetKind, fromState, toState, ruleID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	tk := strings.ToUpper(strings.TrimSpace(targetKind))
	from := strings.TrimSpace(fromState)
	to := strings.TrimSpace(toState)
	rid := strings.TrimSpace(ruleID)

	if tk == "" || from == "" || to == "" || rid == "" {
		return ErrBlankIdentifier
	}

	key := TransitionKey{TargetKind: tk, FromState: from, ToState: to}
	c.transitions[key] = rid
	return nil
}

// LookupRule returns the rule ID bound to the transition.
func (c *TransitionCatalog) LookupRule(targetKind, fromState, toState string) (string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := TransitionKey{
		TargetKind: strings.ToUpper(strings.TrimSpace(targetKind)),
		FromState:  strings.TrimSpace(fromState),
		ToState:    strings.TrimSpace(toState),
	}
	rid, exists := c.transitions[key]
	if !exists {
		return "", ErrInvalidCatalogTransition
	}
	return rid, nil
}

// TransitionQualificationRequest captures all contextual parameters for qualifying a transition.
type TransitionQualificationRequest struct {
	RuleID          string          `json:"rule_id"`
	TargetKind      string          `json:"target_kind"`
	TargetID        string          `json:"target_id"`
	TenantID        string          `json:"tenant_id"`
	FromState       string          `json:"from_state"`
	ToState         string          `json:"to_state"`
	Actor           string          `json:"actor"`
	ActorRole       string          `json:"actor_role"`
	EvidenceIDs     []string        `json:"evidence_ids"`
	InputConditions map[string]bool `json:"input_conditions"`
	MatrixVersion   string          `json:"matrix_version"`
	IsArchived      bool            `json:"is_archived"`
}

// TransitionQualificationResult encapsulates the deterministic decision with stable explanation.
type TransitionQualificationResult struct {
	Allowed         bool       `json:"allowed"`
	RuleID          string     `json:"rule_id"`
	TraceabilityKey string     `json:"traceability_key"`
	DenialReason    DenialCode `json:"denial_reason"`
	Explanation     string     `json:"explanation"`
	EvaluatedAt     time.Time  `json:"evaluated_at"`
}

func qualifyAllow(rule BusinessRule, explanation string, evaluatedAt time.Time) TransitionQualificationResult {
	return TransitionQualificationResult{
		Allowed:         true,
		RuleID:          rule.RuleID,
		TraceabilityKey: rule.TraceabilityKey,
		DenialReason:    DenialNone,
		Explanation:     explanation,
		EvaluatedAt:     evaluatedAt,
	}
}

func qualifyDeny(ruleID, traceKey string, code DenialCode, explanation string, evaluatedAt time.Time) TransitionQualificationResult {
	return TransitionQualificationResult{
		Allowed:         false,
		RuleID:          ruleID,
		TraceabilityKey: traceKey,
		DenialReason:    code,
		Explanation:     explanation,
		EvaluatedAt:     evaluatedAt,
	}
}

// Qualify evaluates a transition request against the RuleMatrix and TransitionCatalog.
// It fails closed if:
// 1. Target, Tenant, or Actor is blank
// 2. Matrix version is incompatible with CurrentRuleMatrixVersion
// 3. Target is in archived / immutable state
// 4. Rule ID is unregistered in the matrix
// 5. Transition is not permitted in the TransitionCatalog for this Rule ID
// 6. ActorRole does not match Rule's OwnerRole
// 7. Any required condition in Rule.RequiredConditions is false or absent
// 8. Required evidence count is not met
func (c *TransitionCatalog) Qualify(matrix *RuleMatrix, req TransitionQualificationRequest) TransitionQualificationResult {
	now := time.Now().UTC()

	// 1. Blank identifier check
	if strings.TrimSpace(req.TargetID) == "" {
		return qualifyDeny(req.RuleID, "", DenialBlankIdentifier, "target ID cannot be blank", now)
	}
	if strings.TrimSpace(req.TenantID) == "" {
		return qualifyDeny(req.RuleID, "", DenialBlankIdentifier, "tenant ID cannot be blank", now)
	}
	if strings.TrimSpace(req.Actor) == "" {
		return qualifyDeny(req.RuleID, "", DenialBlankIdentifier, "actor cannot be blank", now)
	}

	// 2. Matrix Version compatibility
	reqVer := strings.TrimSpace(req.MatrixVersion)
	if reqVer == "" {
		reqVer = matrix.Version()
	}
	if reqVer != CurrentRuleMatrixVersion || matrix.Version() != CurrentRuleMatrixVersion {
		return qualifyDeny(req.RuleID, "", DenialIncompatibleVersion,
			fmt.Sprintf("incompatible matrix version %q: expected %s", reqVer, CurrentRuleMatrixVersion), now)
	}

	// 3. Archived target check (fails closed)
	if req.IsArchived || strings.EqualFold(req.FromState, "ARCHIVED") || strings.EqualFold(req.FromState, "RETIRED") {
		return qualifyDeny(req.RuleID, "", DenialArchivedTarget,
			"target is in an archived/retired state and cannot be mutated", now)
	}

	// 4. Rule ID check
	ruleID := strings.TrimSpace(req.RuleID)
	if ruleID == "" {
		// Attempt to resolve from catalog if ruleID omitted
		catalogRule, err := c.LookupRule(req.TargetKind, req.FromState, req.ToState)
		if err != nil {
			return qualifyDeny("", "", DenialInvalidTransition,
				fmt.Sprintf("no registered transition from %s to %s for target kind %s", req.FromState, req.ToState, req.TargetKind), now)
		}
		ruleID = catalogRule
	}

	rule, err := matrix.GetRule(ruleID)
	if err != nil {
		return qualifyDeny(ruleID, "", DenialUnregisteredRule,
			fmt.Sprintf("business rule %q is not registered in matrix", ruleID), now)
	}

	// 5. Transition catalog validation
	catalogRule, err := c.LookupRule(req.TargetKind, req.FromState, req.ToState)
	if err != nil || catalogRule != rule.RuleID {
		return qualifyDeny(rule.RuleID, rule.TraceabilityKey, DenialInvalidTransition,
			fmt.Sprintf("transition from %s to %s for %s is not permitted under rule %s", req.FromState, req.ToState, req.TargetKind, rule.RuleID), now)
	}

	// 6. Authorization / Actor Class check
	if rule.OwnerRole != "" && !strings.EqualFold(strings.TrimSpace(req.ActorRole), rule.OwnerRole) {
		return qualifyDeny(rule.RuleID, rule.TraceabilityKey, DenialUnauthorizedActor,
			fmt.Sprintf("actor role %q is not authorized for rule %s (required: %s)", req.ActorRole, rule.RuleID, rule.OwnerRole), now)
	}

	// 7. Input Conditions check
	for _, cond := range rule.RequiredConditions {
		trimmedCond := strings.TrimSpace(cond)
		if trimmedCond == "" {
			continue
		}
		if req.InputConditions == nil || !req.InputConditions[trimmedCond] {
			return qualifyDeny(rule.RuleID, rule.TraceabilityKey, DenialMissingCondition,
				fmt.Sprintf("required input condition %q is missing or unfulfilled", trimmedCond), now)
		}
	}

	// 8. Evidence Requirement check
	if rule.Evidence.Required {
		if len(req.EvidenceIDs) < rule.Evidence.MinCount {
			return qualifyDeny(rule.RuleID, rule.TraceabilityKey, DenialMissingEvidence,
				fmt.Sprintf("insufficient evidence: required %d, provided %d", rule.Evidence.MinCount, len(req.EvidenceIDs)), now)
		}
	}

	// All predicates passed!
	return qualifyAllow(rule, fmt.Sprintf("transition qualified and approved under rule %s", rule.RuleID), now)
}

// DefaultRuleMatrix constructs a pre-populated canonical RuleMatrix for standard OSHE transitions.
func DefaultRuleMatrix() *RuleMatrix {
	m := NewRuleMatrix(CurrentRuleMatrixVersion)

	// Template rules
	_ = m.RegisterRule(BusinessRule{
		RuleID:              "RULE-TMPL-SUBMIT",
		OwnerRole:           "AUTHOR",
		RequiredConditions:  []string{"questions_non_empty"},
		DeterministicResult: RuleResultPermitted,
		FailureBehavior:     FailureFailClosed,
		TraceabilityKey:     "OSHE-BR-TMPL-01",
		Description:         "Template submission from Draft to InReview",
	})
	_ = m.RegisterRule(BusinessRule{
		RuleID:              "RULE-TMPL-APPROVE",
		OwnerRole:           "REVIEWER",
		RequiredConditions:  []string{"review_recorded"},
		DeterministicResult: RuleResultPermitted,
		FailureBehavior:     FailureFailClosed,
		TraceabilityKey:     "OSHE-BR-TMPL-02",
		Description:         "Template approval from InReview to Approved",
	})
	_ = m.RegisterRule(BusinessRule{
		RuleID:              "RULE-TMPL-PUBLISH",
		OwnerRole:           "ADMIN",
		RequiredConditions:  []string{"approval_recorded"},
		DeterministicResult: RuleResultPermitted,
		FailureBehavior:     FailureFailClosed,
		TraceabilityKey:     "OSHE-BR-TMPL-03",
		Description:         "Template publication from Approved to Published",
	})
	_ = m.RegisterRule(BusinessRule{
		RuleID:              "RULE-TMPL-RETIRE",
		OwnerRole:           "ADMIN",
		RequiredConditions:  []string{"superseded_or_deprecated"},
		DeterministicResult: RuleResultPermitted,
		FailureBehavior:     FailureFailClosed,
		TraceabilityKey:     "OSHE-BR-TMPL-04",
		Description:         "Template retirement from Published to Retired",
	})

	// Workflow rules
	_ = m.RegisterRule(BusinessRule{
		RuleID:              "RULE-WF-START",
		OwnerRole:           "OPERATOR",
		RequiredConditions:  nil,
		DeterministicResult: RuleResultPermitted,
		FailureBehavior:     FailureFailClosed,
		TraceabilityKey:     "OSHE-BR-WF-01",
		Description:         "Workflow start from Draft to InProgress",
	})
	_ = m.RegisterRule(BusinessRule{
		RuleID:              "RULE-WF-SUBMIT",
		OwnerRole:           "OPERATOR",
		RequiredConditions:  []string{"draft_complete"},
		DeterministicResult: RuleResultPermitted,
		FailureBehavior:     FailureFailClosed,
		TraceabilityKey:     "OSHE-BR-WF-02",
		Description:         "Workflow submission from InProgress to UnderReview",
	})
	_ = m.RegisterRule(BusinessRule{
		RuleID:              "RULE-WF-APPROVE",
		OwnerRole:           "SUPERVISOR",
		RequiredConditions:  []string{"review_passed"},
		DeterministicResult: RuleResultPermitted,
		FailureBehavior:     FailureFailClosed,
		TraceabilityKey:     "OSHE-BR-WF-03",
		Description:         "Workflow approval from UnderReview to Approved",
	})
	_ = m.RegisterRule(BusinessRule{
		RuleID:              "RULE-WF-CLOSE",
		OwnerRole:           "SUPERVISOR",
		RequiredConditions:  []string{"signoff_complete"},
		DeterministicResult: RuleResultPermitted,
		FailureBehavior:     FailureFailClosed,
		TraceabilityKey:     "OSHE-BR-WF-04",
		Description:         "Workflow closure from Approved to Closed",
	})

	// Action rules
	_ = m.RegisterRule(BusinessRule{
		RuleID:              "RULE-ACT-START",
		OwnerRole:           "OWNER",
		RequiredConditions:  nil,
		DeterministicResult: RuleResultPermitted,
		FailureBehavior:     FailureFailClosed,
		TraceabilityKey:     "OSHE-BR-ACT-01",
		Description:         "Action work started from Assigned to InProgress",
	})
	_ = m.RegisterRule(BusinessRule{
		RuleID:              "RULE-ACT-SUBMIT",
		OwnerRole:           "OWNER",
		RequiredConditions:  []string{"remediation_complete"},
		DeterministicResult: RuleResultPermitted,
		FailureBehavior:     FailureFailClosed,
		Evidence:            EvidenceRequirement{Required: true, MinCount: 1},
		TraceabilityKey:     "OSHE-BR-ACT-02",
		Description:         "Action submitted for review with required evidence",
	})
	_ = m.RegisterRule(BusinessRule{
		RuleID:              "RULE-ACT-REJECT",
		OwnerRole:           "REVIEWER",
		RequiredConditions:  []string{"rejection_reason_provided"},
		DeterministicResult: RuleResultPermitted,
		FailureBehavior:     FailureFailClosed,
		TraceabilityKey:     "OSHE-BR-ACT-03",
		Description:         "Action review rejected back to Rejected",
	})
	_ = m.RegisterRule(BusinessRule{
		RuleID:              "RULE-ACT-CLOSE",
		OwnerRole:           "REVIEWER",
		RequiredConditions:  []string{"inspection_verified"},
		DeterministicResult: RuleResultPermitted,
		FailureBehavior:     FailureFailClosed,
		Evidence:            EvidenceRequirement{Required: true, MinCount: 1},
		TraceabilityKey:     "OSHE-BR-ACT-04",
		Description:         "Action approved and closed with evidence",
	})
	_ = m.RegisterRule(BusinessRule{
		RuleID:              "RULE-ACT-REOPEN",
		OwnerRole:           "REVIEWER",
		RequiredConditions:  []string{"reopen_justification"},
		DeterministicResult: RuleResultPermitted,
		FailureBehavior:     FailureFailClosed,
		TraceabilityKey:     "OSHE-BR-ACT-05",
		Description:         "Action reopened from Closed state",
	})

	return m
}

// DefaultTransitionCatalog constructs a pre-populated TransitionCatalog for standard OSHE transitions.
func DefaultTransitionCatalog() *TransitionCatalog {
	c := NewTransitionCatalog()

	// Templates (Draft -> InReview -> Approved -> Published -> Retired)
	_ = c.RegisterTransition(TargetKindTemplate, "Draft", "InReview", "RULE-TMPL-SUBMIT")
	_ = c.RegisterTransition(TargetKindTemplate, "InReview", "Approved", "RULE-TMPL-APPROVE")
	_ = c.RegisterTransition(TargetKindTemplate, "Approved", "Published", "RULE-TMPL-PUBLISH")
	_ = c.RegisterTransition(TargetKindTemplate, "Published", "Retired", "RULE-TMPL-RETIRE")

	// Workflows (DRAFT -> IN_PROGRESS -> UNDER_REVIEW -> APPROVED -> CLOSED)
	_ = c.RegisterTransition(TargetKindWorkflow, string(StateDraft), string(StateInProgress), "RULE-WF-START")
	_ = c.RegisterTransition(TargetKindWorkflow, string(StateInProgress), string(StateUnderReview), "RULE-WF-SUBMIT")
	_ = c.RegisterTransition(TargetKindWorkflow, string(StateUnderReview), string(StateApproved), "RULE-WF-APPROVE")
	_ = c.RegisterTransition(TargetKindWorkflow, string(StateApproved), string(StateClosed), "RULE-WF-CLOSE")

	// Actions (ASSIGNED -> IN_PROGRESS -> IN_REVIEW -> CLOSED, plus REJECTED and REOPENED)
	_ = c.RegisterTransition(TargetKindAction, string(ActionStateAssigned), string(ActionStateInProgress), "RULE-ACT-START")
	_ = c.RegisterTransition(TargetKindAction, string(ActionStateInProgress), string(ActionStateInReview), "RULE-ACT-SUBMIT")
	_ = c.RegisterTransition(TargetKindAction, string(ActionStateInReview), string(ActionStateRejected), "RULE-ACT-REJECT")
	_ = c.RegisterTransition(TargetKindAction, string(ActionStateInReview), string(ActionStateClosed), "RULE-ACT-CLOSE")
	_ = c.RegisterTransition(TargetKindAction, string(ActionStateClosed), string(ActionStateReopened), "RULE-ACT-REOPEN")

	return c
}
