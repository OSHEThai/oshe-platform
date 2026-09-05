// Package localidentity provides local identity, authorization, and directory services for OSHE Platform.
//
// PROVISIONAL GOVERNANCE DECLARATION (H030-003, H030-004, H030-005, Issue #91):
// Under approved Sole Human Owner decision H030-003, this file implements local
// in-memory scoped role assignments, temporal validity windows, revocation mechanics,
// segregation-of-duties conflict detection, and an append-only audit ledger against
// the provisional authorization matrix.
//
// Zero external identity provider integration, database persistence, network execution,
// or binding administrative authority is claimed or enacted. All assignments operate
// as synthetic in-memory fixtures.
package localidentity

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// AssignmentState represents the operational lifecycle state of a scoped role assignment.
type AssignmentState string

const (
	AssignmentStateActive  AssignmentState = "ACTIVE"
	AssignmentStateRevoked AssignmentState = "REVOKED"
	AssignmentStateExpired AssignmentState = "EXPIRED"
)

var (
	// ErrBlankAssignmentID indicates missing assignment identifier.
	ErrBlankAssignmentID = errors.New("assignment ID must not be blank")
	// ErrBlankApprovalSource indicates missing approval source or internal sponsor.
	ErrBlankApprovalSource = errors.New("approval source must not be blank")
	// ErrInvalidTimeWindow indicates valid_to is not strictly after valid_from.
	ErrInvalidTimeWindow = errors.New("valid_to must be strictly after valid_from")
	// ErrRoleConflictDetected indicates a segregation-of-duties conflict or duplicate active role.
	ErrRoleConflictDetected = errors.New("role assignment conflict detected: segregation of duties violation")
	// ErrAssignmentRevoked indicates the role assignment has been explicitly revoked.
	ErrAssignmentRevoked = errors.New("role assignment has been revoked")
	// ErrAssignmentExpired indicates the role assignment temporal window has elapsed.
	ErrAssignmentExpired = errors.New("role assignment has expired")
	// ErrAssignmentNotFound indicates the requested scoped assignment does not exist.
	ErrAssignmentNotFound = errors.New("scoped assignment not found")
	// ErrDuplicateAssignmentID indicates an assignment with the same ID already exists in the tenant.
	ErrDuplicateAssignmentID = errors.New("assignment ID already registered for tenant")
	// ErrUnknownRole indicates the role is not recognized in the system matrix.
	ErrUnknownRole = errors.New("role is not recognized in provisional matrix")
)

// ScopedAssignment encapsulates an authoritative, time-bounded, auditable role grant
// tied strictly to an organizational hierarchy scope.
type ScopedAssignment struct {
	assignmentID     string
	tenantID         string
	subject          string
	role             Role
	scope            ScopeGrant
	validFrom        time.Time
	validTo          time.Time
	approvalSource   string
	state            AssignmentState
	revokedBy        string
	revokedAt        time.Time
	revocationReason string
	createdAt        time.Time
	updatedAt        time.Time
}

// AssignmentID returns the authoritative canonical assignment identifier.
func (s ScopedAssignment) AssignmentID() string { return s.assignmentID }

// TenantID returns the authoritative tenant identifier.
func (s ScopedAssignment) TenantID() string { return s.tenantID }

// Subject returns the singular trusted synthetic identity subject (usr_*).
func (s ScopedAssignment) Subject() string { return s.subject }

// Role returns the assigned discrete security role.
func (s ScopedAssignment) Role() Role { return s.role }

// Scope returns the bounded hierarchy scope grant.
func (s ScopedAssignment) Scope() ScopeGrant { return s.scope }

// ValidFrom returns the start of the validity window.
func (s ScopedAssignment) ValidFrom() time.Time { return s.validFrom }

// ValidTo returns the end of the validity window.
func (s ScopedAssignment) ValidTo() time.Time { return s.validTo }

// ApprovalSource returns the internal sponsor/approver subject ID or authority reference.
func (s ScopedAssignment) ApprovalSource() string { return s.approvalSource }

// State returns the operational lifecycle state.
func (s ScopedAssignment) State() AssignmentState { return s.state }

// RevokedBy returns the actor who revoked the assignment if revoked.
func (s ScopedAssignment) RevokedBy() string { return s.revokedBy }

// RevokedAt returns the revocation timestamp if revoked.
func (s ScopedAssignment) RevokedAt() time.Time { return s.revokedAt }

// RevocationReason returns the recorded rationale for revocation.
func (s ScopedAssignment) RevocationReason() string { return s.revocationReason }

// CreatedAt returns creation timestamp.
func (s ScopedAssignment) CreatedAt() time.Time { return s.createdAt }

// UpdatedAt returns last update timestamp.
func (s ScopedAssignment) UpdatedAt() time.Time { return s.updatedAt }

// IsActive returns true if the assignment is in ACTIVE state.
func (s ScopedAssignment) IsActive() bool { return s.state == AssignmentStateActive }

// IsValidAt evaluates whether the assignment is active and the timestamp falls strictly within [validFrom, validTo].
func (s ScopedAssignment) IsValidAt(t time.Time) bool {
	if s.state != AssignmentStateActive {
		return false
	}
	return !t.Before(s.validFrom) && !t.After(s.validTo)
}

// StateAt returns the effective lifecycle state at a given timestamp.
func (s ScopedAssignment) StateAt(t time.Time) AssignmentState {
	if s.state == AssignmentStateRevoked {
		return AssignmentStateRevoked
	}
	if t.After(s.validTo) || t.Before(s.validFrom) {
		return AssignmentStateExpired
	}
	return AssignmentStateActive
}

// ToRoleAssignment converts the scoped assignment to an access policy RoleAssignment.
func (s ScopedAssignment) ToRoleAssignment() RoleAssignment {
	return RoleAssignment{
		Subject:  s.subject,
		TenantID: s.tenantID,
		Role:     s.role,
		Scope:    s.scope,
	}
}

// Revoke transitions an active assignment to REVOKED state in memory and emits an audit record.
func (s ScopedAssignment) Revoke(revokedBy, reason string, at time.Time) (ScopedAssignment, AssignmentAuditRecord, error) {
	if s.state == AssignmentStateRevoked {
		return s, AssignmentAuditRecord{}, ErrAssignmentRevoked
	}

	trimmedRevoker := strings.TrimSpace(revokedBy)
	if trimmedRevoker == "" {
		return s, AssignmentAuditRecord{}, errors.New("revokedBy must not be blank")
	}
	trimmedReason := strings.TrimSpace(reason)
	if trimmedReason == "" {
		return s, AssignmentAuditRecord{}, errors.New("revocation reason must not be blank")
	}

	updated := s
	updated.state = AssignmentStateRevoked
	updated.revokedBy = trimmedRevoker
	updated.revocationReason = trimmedReason
	updated.revokedAt = at.UTC()
	updated.updatedAt = at.UTC()

	audit := AssignmentAuditRecord{
		RecordID:     fmt.Sprintf("hasn_%s_%d", s.assignmentID, at.UTC().UnixNano()),
		TenantID:     s.tenantID,
		AssignmentID: s.assignmentID,
		Subject:      s.subject,
		Role:         s.role,
		Scope:        s.scope,
		Transition:   "ASSIGNMENT_REVOKED",
		ActorSubject: trimmedRevoker,
		Reason:       trimmedReason,
		RecordedAt:   at.UTC(),
	}

	return updated, audit, nil
}

// NewScopedAssignment constructs and validates a new in-memory ScopedAssignment.
func NewScopedAssignment(assignmentID, tenantID, subject string, role Role, scope ScopeGrant, validFrom, validTo time.Time, approvalSource string) (ScopedAssignment, error) {
	trimmedID := strings.TrimSpace(assignmentID)
	if trimmedID == "" {
		return ScopedAssignment{}, ErrBlankAssignmentID
	}
	trimmedTenant := strings.TrimSpace(tenantID)
	if trimmedTenant == "" {
		return ScopedAssignment{}, ErrBlankTenantID
	}
	if err := ValidateSubject(subject); err != nil {
		return ScopedAssignment{}, err
	}
	if !KnownRoles[role] {
		return ScopedAssignment{}, ErrUnknownRole
	}
	trimmedApproval := strings.TrimSpace(approvalSource)
	if trimmedApproval == "" {
		return ScopedAssignment{}, ErrBlankApprovalSource
	}
	if validTo.Before(validFrom) || validTo.Equal(validFrom) {
		return ScopedAssignment{}, ErrInvalidTimeWindow
	}

	// Enforce tenant boundary match between scope and assignment
	if scope.TenantID != "" && scope.TenantID != trimmedTenant {
		return ScopedAssignment{}, errors.New("scope tenant must match assignment tenant")
	}
	scope.TenantID = trimmedTenant

	now := time.Now().UTC()
	return ScopedAssignment{
		assignmentID:   trimmedID,
		tenantID:       trimmedTenant,
		subject:        strings.TrimSpace(subject),
		role:           role,
		scope:          scope,
		validFrom:      validFrom.UTC(),
		validTo:        validTo.UTC(),
		approvalSource: trimmedApproval,
		state:          AssignmentStateActive,
		createdAt:      now,
		updatedAt:      now,
	}, nil
}

// scopesOverlap checks if two ScopeGrants have overlapping hierarchy coverage.
func scopesOverlap(a, b ScopeGrant) bool {
	if a.TenantID != "" && b.TenantID != "" && a.TenantID != b.TenantID {
		return false
	}
	if a.CompanyID != "" && b.CompanyID != "" && a.CompanyID != b.CompanyID {
		return false
	}
	if a.ProjectID != "" && b.ProjectID != "" && a.ProjectID != b.ProjectID {
		return false
	}
	if a.SiteID != "" && b.SiteID != "" && a.SiteID != b.SiteID {
		return false
	}
	if a.AreaID != "" && b.AreaID != "" && a.AreaID != b.AreaID {
		return false
	}
	if a.ObjectID != "" && b.ObjectID != "" && a.ObjectID != b.ObjectID {
		return false
	}
	return true
}

// CheckRoleConflict evaluates whether candidate conflicts with any existing active assignments
// held by the same subject on overlapping scopes (segregation-of-duties enforcement).
func CheckRoleConflict(existing []ScopedAssignment, candidate ScopedAssignment, at time.Time) error {
	candRole := candidate.Role()
	candScope := candidate.Scope()

	for _, ex := range existing {
		if ex.AssignmentID() == candidate.AssignmentID() {
			continue
		}
		if ex.TenantID() != candidate.TenantID() || ex.Subject() != candidate.Subject() {
			continue
		}
		if !ex.IsValidAt(at) {
			continue // Expired or revoked assignments do not conflict
		}

		if scopesOverlap(ex.Scope(), candScope) {
			exRole := ex.Role()

			// 1. Same-role duplicate active assignment on overlapping scope
			if exRole == candRole {
				return fmt.Errorf("%w: duplicate active role %s already granted on overlapping scope", ErrRoleConflictDetected, candRole)
			}

			// 2. Segregation of Duties: Inspector vs Auditor
			if (candRole == RoleInspector && exRole == RoleAuditor) || (candRole == RoleAuditor && exRole == RoleInspector) {
				return fmt.Errorf("%w: cannot assign both INSPECTOR and AUDITOR on overlapping scope", ErrRoleConflictDetected)
			}

			// 3. Segregation of Duties: ProjectManager vs Auditor
			if (candRole == RoleProjectManager && exRole == RoleAuditor) || (candRole == RoleAuditor && exRole == RoleProjectManager) {
				return fmt.Errorf("%w: cannot assign both PROJECT_MANAGER and AUDITOR on overlapping scope", ErrRoleConflictDetected)
			}

			// 4. Segregation of Duties: Contractor vs Administrative/Management Roles
			if candRole == RoleContractor && (exRole == RoleTenantAdmin || exRole == RoleProjectManager) {
				return fmt.Errorf("%w: contractor cannot hold administrative role %s", ErrRoleConflictDetected, exRole)
			}
			if (candRole == RoleTenantAdmin || candRole == RoleProjectManager) && exRole == RoleContractor {
				return fmt.Errorf("%w: cannot assign administrative role %s to subject holding CONTRACTOR", ErrRoleConflictDetected, candRole)
			}
		}
	}
	return nil
}

// AssignmentAuditRecord captures an immutable historical record of an assignment event.
type AssignmentAuditRecord struct {
	RecordID     string     `json:"record_id"`
	TenantID     string     `json:"tenant_id"`
	AssignmentID string     `json:"assignment_id"`
	Subject      string     `json:"subject"`
	Role         Role       `json:"role"`
	Scope        ScopeGrant `json:"scope"`
	Transition   string     `json:"transition"`
	ActorSubject string     `json:"actor_subject"`
	Reason       string     `json:"reason"`
	RecordedAt   time.Time  `json:"recorded_at"`
}

// AssignmentLedger provides a thread-safe in-memory append-only audit trail for role assignment events.
type AssignmentLedger struct {
	mu      sync.RWMutex
	records []AssignmentAuditRecord
}

// NewAssignmentLedger initializes an empty in-memory ledger.
func NewAssignmentLedger() *AssignmentLedger {
	return &AssignmentLedger{
		records: make([]AssignmentAuditRecord, 0),
	}
}

// AppendRecord appends an audit record to the ledger.
func (l *AssignmentLedger) AppendRecord(record AssignmentAuditRecord) error {
	if record.TenantID == "" || record.AssignmentID == "" {
		return ErrBlankAssignmentID
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.records = append(l.records, record)
	return nil
}

// GetAssignmentAuditTrail retrieves the audit history for an assignment under a tenant.
func (l *AssignmentLedger) GetAssignmentAuditTrail(tenantID, assignmentID string) ([]AssignmentAuditRecord, error) {
	tTenant := strings.TrimSpace(tenantID)
	if tTenant == "" {
		return nil, ErrBlankTenantID
	}
	tAsn := strings.TrimSpace(assignmentID)
	if tAsn == "" {
		return nil, ErrBlankAssignmentID
	}

	l.mu.RLock()
	defer l.mu.RUnlock()

	var results []AssignmentAuditRecord
	for _, rec := range l.records {
		if rec.TenantID == tTenant && rec.AssignmentID == tAsn {
			results = append(results, rec)
		}
	}
	return results, nil
}

// GetSubjectAuditTrail retrieves the audit history for a subject across all assignments under a tenant.
func (l *AssignmentLedger) GetSubjectAuditTrail(tenantID, subject string) ([]AssignmentAuditRecord, error) {
	tTenant := strings.TrimSpace(tenantID)
	if tTenant == "" {
		return nil, ErrBlankTenantID
	}
	tSub := strings.TrimSpace(subject)
	if tSub == "" {
		return nil, ErrBlankSubject
	}

	l.mu.RLock()
	defer l.mu.RUnlock()

	var results []AssignmentAuditRecord
	for _, rec := range l.records {
		if rec.TenantID == tTenant && rec.Subject == tSub {
			results = append(results, rec)
		}
	}
	return results, nil
}

// ScopedAssignmentRegistry provides a thread-safe, in-memory store for scoped role assignments.
type ScopedAssignmentRegistry struct {
	mu          sync.RWMutex
	assignments map[string]ScopedAssignment // key: tenantID + ":" + assignmentID
	ledger      *AssignmentLedger
}

// NewScopedAssignmentRegistry constructs an empty in-memory registry.
func NewScopedAssignmentRegistry(ledger *AssignmentLedger) *ScopedAssignmentRegistry {
	if ledger == nil {
		ledger = NewAssignmentLedger()
	}
	return &ScopedAssignmentRegistry{
		assignments: make(map[string]ScopedAssignment),
		ledger:      ledger,
	}
}

func makeAssignmentKey(tenantID, assignmentID string) string {
	return fmt.Sprintf("%s:%s", strings.TrimSpace(tenantID), strings.TrimSpace(assignmentID))
}

// RegisterAssignment stores a new scoped assignment, validating segregation-of-duties conflicts
// and appending an initial audit record.
func (r *ScopedAssignmentRegistry) RegisterAssignment(assignment ScopedAssignment, actorSubject, reason string, at time.Time) error {
	if assignment.TenantID() == "" || assignment.AssignmentID() == "" {
		return ErrBlankAssignmentID
	}

	key := makeAssignmentKey(assignment.TenantID(), assignment.AssignmentID())

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.assignments[key]; exists {
		return ErrDuplicateAssignmentID
	}

	// Fetch existing assignments for this subject in the tenant to check conflicts
	var existing []ScopedAssignment
	for _, a := range r.assignments {
		if a.TenantID() == assignment.TenantID() && a.Subject() == assignment.Subject() {
			existing = append(existing, a)
		}
	}

	if err := CheckRoleConflict(existing, assignment, at); err != nil {
		return err
	}

	r.assignments[key] = assignment

	// Append creation audit record
	initAudit := AssignmentAuditRecord{
		RecordID:     fmt.Sprintf("hasn_%s_%d", assignment.AssignmentID(), at.UTC().UnixNano()),
		TenantID:     assignment.TenantID(),
		AssignmentID: assignment.AssignmentID(),
		Subject:      assignment.Subject(),
		Role:         assignment.Role(),
		Scope:        assignment.Scope(),
		Transition:   "ASSIGNMENT_CREATED",
		ActorSubject: strings.TrimSpace(actorSubject),
		Reason:       strings.TrimSpace(reason),
		RecordedAt:   at.UTC(),
	}
	return r.ledger.AppendRecord(initAudit)
}

// RevokeAssignment explicitly revokes an assignment and appends the revocation audit record.
func (r *ScopedAssignmentRegistry) RevokeAssignment(tenantID, assignmentID, actorSubject, reason string, at time.Time) (ScopedAssignment, error) {
	key := makeAssignmentKey(tenantID, assignmentID)

	r.mu.Lock()
	defer r.mu.Unlock()

	current, exists := r.assignments[key]
	if !exists {
		return ScopedAssignment{}, ErrAssignmentNotFound
	}

	revoked, audit, err := current.Revoke(actorSubject, reason, at)
	if err != nil {
		return ScopedAssignment{}, err
	}

	r.assignments[key] = revoked
	if err := r.ledger.AppendRecord(audit); err != nil {
		return ScopedAssignment{}, err
	}

	return revoked, nil
}

// GetAssignment retrieves a single scoped assignment by tenant and ID.
func (r *ScopedAssignmentRegistry) GetAssignment(tenantID, assignmentID string) (ScopedAssignment, error) {
	key := makeAssignmentKey(tenantID, assignmentID)

	r.mu.RLock()
	defer r.mu.RUnlock()

	a, exists := r.assignments[key]
	if !exists {
		return ScopedAssignment{}, ErrAssignmentNotFound
	}
	return a, nil
}

// ListActiveAssignments returns all active, time-valid assignments for a subject in a tenant at time 'at'.
func (r *ScopedAssignmentRegistry) ListActiveAssignments(tenantID, subject string, at time.Time) ([]ScopedAssignment, error) {
	tTenant := strings.TrimSpace(tenantID)
	if tTenant == "" {
		return nil, ErrBlankTenantID
	}
	tSub := strings.TrimSpace(subject)
	if tSub == "" {
		return nil, ErrBlankSubject
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	var results []ScopedAssignment
	for _, a := range r.assignments {
		if a.TenantID() == tTenant && a.Subject() == tSub && a.IsValidAt(at) {
			results = append(results, a)
		}
	}
	return results, nil
}

// ListAssignmentsByScope returns all active, time-valid assignments overlapping with target scope at time 'at'.
func (r *ScopedAssignmentRegistry) ListAssignmentsByScope(tenantID string, scope ScopeGrant, at time.Time) ([]ScopedAssignment, error) {
	tTenant := strings.TrimSpace(tenantID)
	if tTenant == "" {
		return nil, ErrBlankTenantID
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	var results []ScopedAssignment
	for _, a := range r.assignments {
		if a.TenantID() == tTenant && a.IsValidAt(at) && scopesOverlap(a.Scope(), scope) {
			results = append(results, a)
		}
	}
	return results, nil
}

// PopulateEvaluator populates all active, time-valid scoped assignments into a PolicyEvaluator.
func (r *ScopedAssignmentRegistry) PopulateEvaluator(tenantID string, evaluator *PolicyEvaluator, at time.Time) error {
	tTenant := strings.TrimSpace(tenantID)
	if tTenant == "" {
		return ErrBlankTenantID
	}
	if evaluator == nil {
		return errors.New("evaluator cannot be nil")
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, a := range r.assignments {
		if a.TenantID() == tTenant && a.IsValidAt(at) {
			evaluator.AddRoleAssignment(a.ToRoleAssignment())
		}
	}
	return nil
}

// EvaluateScopedAccess provides an end-to-end evaluation entry point asserting that caller
// holds an active, valid scoped assignment permitting the requested action at time 'at'.
func EvaluateScopedAccess(registry *ScopedAssignmentRegistry, evaluator *PolicyEvaluator, req AccessRequest, at time.Time) EvaluationResult {
	if registry == nil || evaluator == nil {
		return Deny(DenialDefaultDeny, "uninitialized authorization evaluators")
	}

	// 1. Basic identity assertions
	if !req.Identity.IsAuthenticated || strings.TrimSpace(req.Identity.Subject) == "" || strings.TrimSpace(req.Identity.TenantID) == "" {
		return Deny(DenialUnauthenticated, "unauthenticated caller identity")
	}

	// 2. Cross-tenant denial
	if req.Identity.TenantID != req.Target.TenantID {
		return Deny(DenialCrossTenant, "cross-tenant access prohibited")
	}

	// 3. Find active valid scoped assignments for this subject at timestamp 'at'
	activeAssignments, err := registry.ListActiveAssignments(req.Identity.TenantID, req.Identity.Subject, at)
	if err != nil || len(activeAssignments) == 0 {
		return Deny(DenialRoleNotGranted, "no active scoped role assignments found for subject")
	}

	// 4. Construct isolated evaluation instance that inherits memberships and entitlements from evaluator
	// but derives active role assignments strictly from the ScopedAssignmentRegistry at time 'at'
	scopedEval := NewPolicyEvaluator()

	evaluator.mu.RLock()
	for k, v := range evaluator.memberships {
		scopedEval.memberships[k] = v
	}
	for k, v := range evaluator.entitlements {
		scopedEval.entitlements[k] = make(map[string]bool)
		for ek, ev := range v {
			scopedEval.entitlements[k][ek] = ev
		}
	}
	evaluator.mu.RUnlock()

	for _, a := range activeAssignments {
		scopedEval.AddRoleAssignment(a.ToRoleAssignment())
	}

	return scopedEval.Evaluate(req)
}
