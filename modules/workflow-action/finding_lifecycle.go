package workflowaction

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// TargetKindFinding defines the target kind for finding transitions in business rules.
const TargetKindFinding = "FINDING"

// FindingState represents the explicit lifecycle state of a finding.
type FindingState string

const (
	FindingStateOpen        FindingState = "OPEN"
	FindingStateUnderReview FindingState = "UNDER_REVIEW"
	FindingStateRemediated  FindingState = "REMEDIATED"
	FindingStateClosed      FindingState = "CLOSED"
)

// FindingSeverity represents owner-controlled severity inputs.
// Autonomous AI classification is strictly prohibited.
type FindingSeverity string

const (
	SeverityLow      FindingSeverity = "LOW"
	SeverityMedium   FindingSeverity = "MEDIUM"
	SeverityHigh     FindingSeverity = "HIGH"
	SeverityCritical FindingSeverity = "CRITICAL"
)

var (
	ErrBlankFindingID                     = errors.New("finding ID cannot be blank")
	ErrDuplicateFindingID                 = errors.New("duplicate finding ID")
	ErrFindingNotFound                    = errors.New("finding not found")
	ErrMissingSourceContext               = errors.New("finding must link to an active inspection execution, question, and response")
	ErrRuleNotFound                       = errors.New("finding evaluation rule not registered in rule catalog")
	ErrIncompatibleRuleVersion            = errors.New("rule version mismatch: pinned matrix version required")
	ErrMissingImmediateControl            = errors.New("immediate-control note is mandatory for critical or high-severity findings")
	ErrAutonomousClassificationProhibited = errors.New("autonomous AI classification of severity or critical status is prohibited")
	ErrUnauthorizedSeverityDowngrade      = errors.New("unauthorized severity downgrade: requires supervisory authority and rationale")
	ErrUnauthorizedCriticalDowngrade      = errors.New("unauthorized removal of critical flag: requires supervisory authority and rationale")
	ErrMissingClassificationRationale     = errors.New("classification rationale cannot be blank")
	ErrAutonomousClosureProhibited        = errors.New("autonomous AI closure of finding is strictly prohibited")
	ErrUnauthorizedClosure                = errors.New("unauthorized finding closure: actor lacks required human closure authority")
	ErrSilentHidingProhibited             = errors.New("suppression or silent hiding of concerning response finding is prohibited")
	ErrInvalidFindingStateTransition      = errors.New("invalid finding state transition")
	ErrFindingAlreadyClosed               = errors.New("finding is already closed and cannot be mutated")
	ErrInvalidRecurrenceLink              = errors.New("recurrence link target does not exist or tenant mismatch")
)

// FindingRule defines a registered deterministic rule for generating findings from failed checklist responses.
type FindingRule struct {
	RuleID                   string          `json:"rule_id"`
	RuleVersion              string          `json:"rule_version"`
	QuestionPattern          string          `json:"question_pattern"`
	TriggerCondition         string          `json:"trigger_condition"`
	DefaultSeverityInput     FindingSeverity `json:"default_severity_input"`
	MandatoryCritical        bool            `json:"mandatory_critical"`
	RequiresImmediateControl bool            `json:"requires_immediate_control"`
	RequiresEvidence         bool            `json:"requires_evidence"`
}

// FindingRuleCatalog manages the approved, version-pinned finding generation rules.
type FindingRuleCatalog struct {
	mu    sync.RWMutex
	rules map[string]FindingRule // rule_id -> rule
}

// NewFindingRuleCatalog initializes a rule catalog with default alpha rules.
func NewFindingRuleCatalog() *FindingRuleCatalog {
	c := &FindingRuleCatalog{
		rules: make(map[string]FindingRule),
	}
	// Register default baseline rules
	c.RegisterRule(FindingRule{
		RuleID:                   "RULE-FND-FIRE-EXIT-BLOCKED",
		RuleVersion:              CurrentRuleMatrixVersion,
		QuestionPattern:          "qst_fire_exit_*",
		TriggerCondition:         "RESPONSE_CONCERNING_OR_FAILED",
		DefaultSeverityInput:     SeverityCritical,
		MandatoryCritical:        true,
		RequiresImmediateControl: true,
		RequiresEvidence:         true,
	})
	c.RegisterRule(FindingRule{
		RuleID:                   "RULE-FND-EXTINGUISHER-DEFECT",
		RuleVersion:              CurrentRuleMatrixVersion,
		QuestionPattern:          "qst_extinguisher_*",
		TriggerCondition:         "RESPONSE_FAILED",
		DefaultSeverityInput:     SeverityHigh,
		MandatoryCritical:        false,
		RequiresImmediateControl: true,
		RequiresEvidence:         true,
	})
	c.RegisterRule(FindingRule{
		RuleID:                   "RULE-FND-PPE-NONCOMPLIANCE",
		RuleVersion:              CurrentRuleMatrixVersion,
		QuestionPattern:          "qst_ppe_*",
		TriggerCondition:         "RESPONSE_FAILED",
		DefaultSeverityInput:     SeverityMedium,
		MandatoryCritical:        false,
		RequiresImmediateControl: false,
		RequiresEvidence:         false,
	})
	return c
}

// RegisterRule adds an authorized rule to the catalog.
func (c *FindingRuleCatalog) RegisterRule(rule FindingRule) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.rules[rule.RuleID] = rule
}

// GetRule retrieves a rule by ID.
func (c *FindingRuleCatalog) GetRule(ruleID string) (FindingRule, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	r, ok := c.rules[ruleID]
	return r, ok
}

// FindingAuditEntry records an append-only audit event in the finding lifecycle.
type FindingAuditEntry struct {
	Sequence  int64        `json:"sequence"`
	Timestamp time.Time    `json:"timestamp"`
	Actor     string       `json:"actor"`
	ActorRole string       `json:"actor_role"`
	Action    string       `json:"action"`
	FromState FindingState `json:"from_state,omitempty"`
	ToState   FindingState `json:"to_state,omitempty"`
	Details   string       `json:"details"`
}

// FindingRecord represents an immutable-identity safety finding.
type FindingRecord struct {
	FindingID        string              `json:"finding_id"`
	TenantID         string              `json:"tenant_id"`
	ExecutionID      string              `json:"execution_id"`
	QuestionID       string              `json:"question_id"`
	ResponseID       string              `json:"response_id"`
	RecurrenceID     string              `json:"recurrence_id,omitempty"`
	RuleID           string              `json:"rule_id"`
	RuleVersion      string              `json:"rule_version"`
	Title            string              `json:"title"`
	Description      string              `json:"description"`
	Severity         FindingSeverity     `json:"severity"`
	CriticalFlag     bool                `json:"critical_flag"`
	ImmediateControl string              `json:"immediate_control"`
	State            FindingState        `json:"state"`
	EvidenceIDs      []string            `json:"evidence_ids,omitempty"`
	CreatedAt        time.Time           `json:"created_at"`
	CreatedBy        string              `json:"created_by"`
	ClosedAt         *time.Time          `json:"closed_at,omitempty"`
	ClosedBy         string              `json:"closed_by,omitempty"`
	ClosureRationale string              `json:"closure_rationale,omitempty"`
	History          []FindingAuditEntry `json:"history"`
}

// FindingCreationRequest defines all parameters to deterministically generate a finding.
type FindingCreationRequest struct {
	TenantID         string          `json:"tenant_id"`
	FindingID        string          `json:"finding_id"`
	ExecutionID      string          `json:"execution_id"`
	QuestionID       string          `json:"question_id"`
	ResponseID       string          `json:"response_id"`
	RecurrenceID     string          `json:"recurrence_id,omitempty"`
	RuleID           string          `json:"rule_id"`
	RuleVersion      string          `json:"rule_version"`
	Title            string          `json:"title"`
	Description      string          `json:"description"`
	SeverityInput    FindingSeverity `json:"severity_input"`
	CriticalFlag     bool            `json:"critical_flag"`
	ImmediateControl string          `json:"immediate_control"`
	Actor            string          `json:"actor"`
	ActorRole        string          `json:"actor_role"`
	EvidenceIDs      []string        `json:"evidence_ids,omitempty"`
}

// FindingManager provides thread-safe operations on finding records.
type FindingManager struct {
	mu       sync.RWMutex
	findings map[string]map[string]*FindingRecord // tenantID -> findingID -> record
	catalog  *FindingRuleCatalog
}

// NewFindingManager initializes a new FindingManager with the given rule catalog.
func NewFindingManager(catalog *FindingRuleCatalog) *FindingManager {
	if catalog == nil {
		catalog = NewFindingRuleCatalog()
	}
	return &FindingManager{
		findings: make(map[string]map[string]*FindingRecord),
		catalog:  catalog,
	}
}

func severityRank(s FindingSeverity) int {
	switch s {
	case SeverityLow:
		return 1
	case SeverityMedium:
		return 2
	case SeverityHigh:
		return 3
	case SeverityCritical:
		return 4
	default:
		return 0
	}
}

// isSupervisorRole checks if the actor role has supervisory classification authority.
func isSupervisorRole(role string) bool {
	norm := strings.ToUpper(strings.TrimSpace(role))
	switch norm {
	case "SAFETY_SUPERVISOR", "SUPERVISOR", "SAFETY_LEAD", "QUALITY_LEAD", "TENANT_ADMIN", "OWNER":
		return true
	default:
		return false
	}
}

// isHumanClosureAuthorized checks if actor role has human final-closure authority.
func isHumanClosureAuthorized(role string) bool {
	norm := strings.ToUpper(strings.TrimSpace(role))
	switch norm {
	case "SAFETY_SUPERVISOR", "SUPERVISOR", "SAFETY_LEAD", "QUALITY_LEAD", "OWNER":
		return true
	default:
		return false
	}
}

// CreateFinding creates an authoritative finding enforcing rule presence, version pinning, and fail-closed guards.
func (m *FindingManager) CreateFinding(req FindingCreationRequest) (*FindingRecord, error) {
	tTenant := strings.TrimSpace(req.TenantID)
	if tTenant == "" {
		return nil, ErrBlankTenantID
	}
	tFinding := strings.TrimSpace(req.FindingID)
	if tFinding == "" {
		return nil, ErrBlankFindingID
	}
	if strings.TrimSpace(req.ExecutionID) == "" || strings.TrimSpace(req.QuestionID) == "" || strings.TrimSpace(req.ResponseID) == "" {
		return nil, ErrMissingSourceContext
	}
	tRule := strings.TrimSpace(req.RuleID)
	if tRule == "" {
		return nil, ErrRuleNotFound
	}

	// Verify rule registration in catalog
	rule, exists := m.catalog.GetRule(tRule)
	if !exists {
		return nil, ErrRuleNotFound
	}

	// Verify rule version compatibility
	if req.RuleVersion != rule.RuleVersion || req.RuleVersion != CurrentRuleMatrixVersion {
		return nil, ErrIncompatibleRuleVersion
	}

	// Prohibit autonomous AI classification of severity or critical flag
	actorNorm := strings.ToUpper(strings.TrimSpace(req.Actor))
	roleNorm := strings.ToUpper(strings.TrimSpace(req.ActorRole))
	if strings.Contains(actorNorm, "AI_AGENT") || strings.Contains(roleNorm, "AI_CORE") {
		if req.SeverityInput == "" && !rule.MandatoryCritical {
			return nil, ErrAutonomousClassificationProhibited
		}
	}

	// Resolve severity: defaults to rule catalog default if blank
	severity := req.SeverityInput
	if severity == "" {
		severity = rule.DefaultSeverityInput
	}
	if severityRank(severity) == 0 {
		return nil, errors.New("invalid severity level")
	}

	// Resolve critical flag: rule mandatory critical cannot be unset
	critical := req.CriticalFlag
	if rule.MandatoryCritical {
		critical = true
	}

	// Check immediate control note requirement
	immediateCtrl := strings.TrimSpace(req.ImmediateControl)
	if (critical || severity == SeverityCritical || rule.RequiresImmediateControl) && immediateCtrl == "" {
		return nil, ErrMissingImmediateControl
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if tenantMap, exists := m.findings[tTenant]; exists {
		if _, exists := tenantMap[tFinding]; exists {
			return nil, ErrDuplicateFindingID
		}
	} else {
		m.findings[tTenant] = make(map[string]*FindingRecord)
	}

	// Validate recurrence link if specified
	tRecurrence := strings.TrimSpace(req.RecurrenceID)
	if tRecurrence != "" {
		if tenantMap, exists := m.findings[tTenant]; exists {
			if _, exists := tenantMap[tRecurrence]; !exists {
				return nil, ErrInvalidRecurrenceLink
			}
		} else {
			return nil, ErrInvalidRecurrenceLink
		}
	}

	now := time.Now().UTC()
	audit := FindingAuditEntry{
		Sequence:  1,
		Timestamp: now,
		Actor:     req.Actor,
		ActorRole: req.ActorRole,
		Action:    "CREATED",
		FromState: "",
		ToState:   FindingStateOpen,
		Details:   fmt.Sprintf("Finding created under rule %s v%s with severity %s (critical=%t)", rule.RuleID, rule.RuleVersion, severity, critical),
	}

	rec := &FindingRecord{
		FindingID:        tFinding,
		TenantID:         tTenant,
		ExecutionID:      req.ExecutionID,
		QuestionID:       req.QuestionID,
		ResponseID:       req.ResponseID,
		RecurrenceID:     tRecurrence,
		RuleID:           rule.RuleID,
		RuleVersion:      rule.RuleVersion,
		Title:            req.Title,
		Description:      req.Description,
		Severity:         severity,
		CriticalFlag:     critical,
		ImmediateControl: immediateCtrl,
		State:            FindingStateOpen,
		EvidenceIDs:      req.EvidenceIDs,
		CreatedAt:        now,
		CreatedBy:        req.Actor,
		History:          []FindingAuditEntry{audit},
	}

	m.findings[tTenant][tFinding] = rec
	return rec, nil
}

// GetFinding retrieves a finding record within tenant scope.
func (m *FindingManager) GetFinding(tenantID, findingID string) (*FindingRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tenantMap, exists := m.findings[tenantID]
	if !exists {
		return nil, ErrFindingNotFound
	}
	rec, exists := tenantMap[findingID]
	if !exists {
		return nil, ErrFindingNotFound
	}

	copyRec := *rec
	return &copyRec, nil
}

// UpdateSeverity updates finding severity, enforcing downgrade authorization and rationale.
func (m *FindingManager) UpdateSeverity(tenantID, findingID string, newSeverity FindingSeverity, actor, actorRole, rationale string) error {
	tTenant := strings.TrimSpace(tenantID)
	tFinding := strings.TrimSpace(findingID)
	tRationale := strings.TrimSpace(rationale)

	m.mu.Lock()
	defer m.mu.Unlock()

	tenantMap, exists := m.findings[tTenant]
	if !exists {
		return ErrFindingNotFound
	}
	rec, exists := tenantMap[tFinding]
	if !exists {
		return ErrFindingNotFound
	}

	if rec.State == FindingStateClosed {
		return ErrFindingAlreadyClosed
	}

	newRank := severityRank(newSeverity)
	if newRank == 0 {
		return errors.New("invalid new severity level")
	}
	oldRank := severityRank(rec.Severity)

	// Downgrade check
	if newRank < oldRank {
		if !isSupervisorRole(actorRole) {
			return ErrUnauthorizedSeverityDowngrade
		}
		if tRationale == "" {
			return ErrMissingClassificationRationale
		}
	}

	// Prohibit AI autonomous downgrade or change
	if strings.Contains(strings.ToUpper(actor), "AI_AGENT") || strings.Contains(strings.ToUpper(actorRole), "AI_CORE") {
		return ErrAutonomousClassificationProhibited
	}

	now := time.Now().UTC()
	audit := FindingAuditEntry{
		Sequence:  int64(len(rec.History) + 1),
		Timestamp: now,
		Actor:     actor,
		ActorRole: actorRole,
		Action:    "SEVERITY_UPDATED",
		Details:   fmt.Sprintf("Severity transitioned from %s to %s. Rationale: %s", rec.Severity, newSeverity, tRationale),
	}

	rec.Severity = newSeverity
	rec.History = append(rec.History, audit)
	return nil
}

// UpdateCriticalFlag updates the critical status of a finding with downgrade protection.
func (m *FindingManager) UpdateCriticalFlag(tenantID, findingID string, critical bool, actor, actorRole, rationale string) error {
	tTenant := strings.TrimSpace(tenantID)
	tFinding := strings.TrimSpace(findingID)
	tRationale := strings.TrimSpace(rationale)

	m.mu.Lock()
	defer m.mu.Unlock()

	tenantMap, exists := m.findings[tTenant]
	if !exists {
		return ErrFindingNotFound
	}
	rec, exists := tenantMap[tFinding]
	if !exists {
		return ErrFindingNotFound
	}

	if rec.State == FindingStateClosed {
		return ErrFindingAlreadyClosed
	}

	// Check if rule mandates critical permanently
	if rule, ok := m.catalog.GetRule(rec.RuleID); ok && rule.MandatoryCritical && !critical {
		return errors.New("cannot remove critical flag: rule specifies mandatory critical")
	}

	// Downgrade check: critical -> non-critical
	if rec.CriticalFlag && !critical {
		if !isSupervisorRole(actorRole) {
			return ErrUnauthorizedCriticalDowngrade
		}
		if tRationale == "" {
			return ErrMissingClassificationRationale
		}
	}

	// Prohibit AI autonomous downgrade
	if strings.Contains(strings.ToUpper(actor), "AI_AGENT") || strings.Contains(strings.ToUpper(actorRole), "AI_CORE") {
		return ErrAutonomousClassificationProhibited
	}

	now := time.Now().UTC()
	audit := FindingAuditEntry{
		Sequence:  int64(len(rec.History) + 1),
		Timestamp: now,
		Actor:     actor,
		ActorRole: actorRole,
		Action:    "CRITICAL_FLAG_UPDATED",
		Details:   fmt.Sprintf("Critical flag transitioned from %t to %t. Rationale: %s", rec.CriticalFlag, critical, tRationale),
	}

	rec.CriticalFlag = critical
	rec.History = append(rec.History, audit)
	return nil
}

// TransitionState transitions finding lifecycle state.
func (m *FindingManager) TransitionState(tenantID, findingID string, toState FindingState, actor, actorRole, rationale string) error {
	tTenant := strings.TrimSpace(tenantID)
	tFinding := strings.TrimSpace(findingID)

	m.mu.Lock()
	defer m.mu.Unlock()

	tenantMap, exists := m.findings[tTenant]
	if !exists {
		return ErrFindingNotFound
	}
	rec, exists := tenantMap[tFinding]
	if !exists {
		return ErrFindingNotFound
	}

	if rec.State == FindingStateClosed {
		return ErrFindingAlreadyClosed
	}

	// Prohibit autonomous AI closure
	if toState == FindingStateClosed {
		if strings.Contains(strings.ToUpper(actor), "AI_AGENT") || strings.Contains(strings.ToUpper(actorRole), "AI_CORE") {
			return ErrAutonomousClosureProhibited
		}
		if !isHumanClosureAuthorized(actorRole) {
			return ErrUnauthorizedClosure
		}
		if strings.TrimSpace(rationale) == "" {
			return errors.New("closure rationale cannot be blank")
		}
	}

	// Permitted linear transitions
	switch rec.State {
	case FindingStateOpen:
		if toState != FindingStateUnderReview && toState != FindingStateRemediated && toState != FindingStateClosed {
			return ErrInvalidFindingStateTransition
		}
	case FindingStateUnderReview:
		if toState != FindingStateRemediated && toState != FindingStateClosed && toState != FindingStateOpen {
			return ErrInvalidFindingStateTransition
		}
	case FindingStateRemediated:
		if toState != FindingStateClosed && toState != FindingStateUnderReview {
			return ErrInvalidFindingStateTransition
		}
	default:
		return ErrInvalidFindingStateTransition
	}

	now := time.Now().UTC()
	audit := FindingAuditEntry{
		Sequence:  int64(len(rec.History) + 1),
		Timestamp: now,
		Actor:     actor,
		ActorRole: actorRole,
		Action:    "STATE_TRANSITION",
		FromState: rec.State,
		ToState:   toState,
		Details:   fmt.Sprintf("Transitioned from %s to %s. Details: %s", rec.State, toState, rationale),
	}

	rec.State = toState
	if toState == FindingStateClosed {
		rec.ClosedAt = &now
		rec.ClosedBy = actor
		rec.ClosureRationale = rationale
	}
	rec.History = append(rec.History, audit)
	return nil
}
