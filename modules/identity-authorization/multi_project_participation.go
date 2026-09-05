// Package localidentity provides local identity, authorization, and directory services for OSHE Platform.
//
// PROVISIONAL GOVERNANCE DECLARATION (H030-003, H030-004, H030-005, Issue #96):
// Under approved Sole Human Owner decisions H030-003, H030-004, and H030-005, this file implements
// local synthetic multi-project participation, bounded contractor administration, auditor read-only
// enforcement, and preserved historical actor attribution after deactivation.
//
// Strict Prework Invariants:
// 1. Synthetic Local Identity Only: Operates strictly on synthetic identities (usr_*). Zero real user
//    or customer data is processed, persisted, or exposed.
// 2. Exact Project Scope & No Cross-Project Leakage: A subject's participation in one project conveys
//    zero operational authority in sibling projects. Cross-project queries and requests fail closed.
// 3. Bounded Contractor Administration: External contractors are categorically barred from holding
//    administrative or project management roles (RoleTenantAdmin, RoleProjectManager).
// 4. Auditor Read-Only Enforcement: Compliance auditors hold strictly read-only capabilities across
//    projects; all mutating actions (ActionCreate, ActionUpdate, ActionDelete) are strictly denied.
// 5. Preserved Historical Attribution: Deactivation of a participant immediately terminates future
//    operational access, but all past actions, signatures, and findings remain permanently attributed
//    in an append-only historical audit trail with zero deletion.
// 6. Zero External Enactment: Operates purely in-memory. No database migrations, external identity
//    provider synchronization, network routes, or runtime policy activation are claimed or enacted.
package localidentity

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// ParticipationStatus classifies the operational lifecycle of a project participation binding.
type ParticipationStatus string

const (
	ParticipationActive      ParticipationStatus = "ACTIVE"
	ParticipationDeactivated ParticipationStatus = "DEACTIVATED"
	ParticipationRevoked     ParticipationStatus = "REVOKED"
	ParticipationExpired     ParticipationStatus = "EXPIRED"
)

var (
	// ErrBlankParticipationID indicates missing participation identifier.
	ErrBlankParticipationID = errors.New("participation ID must not be blank")
	// ErrParticipationNotFound indicates the requested participation record does not exist.
	ErrParticipationNotFound = errors.New("project participation record not found")
	// ErrDuplicateParticipation indicates an active participation already exists for subject in project.
	ErrDuplicateParticipation = errors.New("active participation already exists for subject in project")
	// ErrParticipationInactive indicates the participation is not currently active.
	ErrParticipationInactive = errors.New("project participation is not active")
	// ErrParticipationDeactivated indicates the participation has been deactivated.
	ErrParticipationDeactivated = errors.New("project participation has been deactivated")
	// ErrParticipationExpired indicates the participation temporal validity window has elapsed.
	ErrParticipationExpired = errors.New("project participation temporal window has elapsed")
	// ErrCrossProjectAccessDenied indicates access denied due to lack of participation in target project.
	ErrCrossProjectAccessDenied = errors.New("cross-project access denied: subject lacks active participation in target project")
	// ErrContractorAdminProhibited indicates illegal attempt to assign administrative role to a contractor.
	ErrContractorAdminProhibited = errors.New("bounded contractor administration violation: contractor cannot hold administrative or project management authority")
	// ErrAuditorReadOnlyViolation indicates illegal attempt by an auditor to execute mutating operations.
	ErrAuditorReadOnlyViolation = errors.New("auditor read-only violation: mutating operations are prohibited for auditor role")
	// ErrAttributionImmutable indicates historical attribution records cannot be overwritten or altered.
	ErrAttributionImmutable = errors.New("historical attribution records are immutable and cannot be altered or deleted")
	// ErrBlankResourceID indicates missing resource identifier in attribution record.
	ErrBlankResourceID = errors.New("resource ID must not be blank")
	// ErrBlankActionType indicates missing action type in attribution record.
	ErrBlankActionType = errors.New("action type must not be blank")
	// ErrBlankAttributionID indicates missing attribution record identifier.
	ErrBlankAttributionID = errors.New("attribution record ID must not be blank")
)

// ProjectParticipation binds a synthetic subject (usr_*) to a specific project (prj_*) with a discrete
// security role, bounded scope grant, temporal validity, and internal approver attribution.
type ProjectParticipation struct {
	participationID    string
	tenantID           string
	subject            string
	projectID          string
	role               Role
	scope              ScopeGrant
	assignedBy         string
	validFrom          time.Time
	validTo            time.Time
	status             ParticipationStatus
	isContractor       bool
	createdAt          time.Time
	updatedAt          time.Time
	deactivatedAt      time.Time
	deactivationReason string
	deactivatedBy      string
}

// ParticipationID returns the canonical participation identifier.
func (p ProjectParticipation) ParticipationID() string { return p.participationID }

// TenantID returns the authoritative tenant identifier.
func (p ProjectParticipation) TenantID() string { return p.tenantID }

// Subject returns the synthetic subject identifier (usr_*).
func (p ProjectParticipation) Subject() string { return p.subject }

// ProjectID returns the bounded project identifier (prj_*).
func (p ProjectParticipation) ProjectID() string { return p.projectID }

// Role returns the assigned security role in this project context.
func (p ProjectParticipation) Role() Role { return p.role }

// Scope returns the bounded hierarchy scope grant.
func (p ProjectParticipation) Scope() ScopeGrant { return p.scope }

// AssignedBy returns the internal sponsor/approver identifier.
func (p ProjectParticipation) AssignedBy() string { return p.assignedBy }

// ValidFrom returns the start of the validity window.
func (p ProjectParticipation) ValidFrom() time.Time { return p.validFrom }

// ValidTo returns the expiration timestamp of the validity window.
func (p ProjectParticipation) ValidTo() time.Time { return p.validTo }

// Status returns the current lifecycle status.
func (p ProjectParticipation) Status() ParticipationStatus { return p.status }

// IsContractor returns true if the participant is classified as an external contractor.
func (p ProjectParticipation) IsContractor() bool { return p.isContractor }

// CreatedAt returns creation timestamp.
func (p ProjectParticipation) CreatedAt() time.Time { return p.createdAt }

// UpdatedAt returns last update timestamp.
func (p ProjectParticipation) UpdatedAt() time.Time { return p.updatedAt }

// DeactivatedAt returns deactivation timestamp if deactivated.
func (p ProjectParticipation) DeactivatedAt() time.Time { return p.deactivatedAt }

// DeactivationReason returns recorded rationale for deactivation.
func (p ProjectParticipation) DeactivationReason() string { return p.deactivationReason }

// DeactivatedBy returns actor who executed deactivation.
func (p ProjectParticipation) DeactivatedBy() string { return p.deactivatedBy }

// IsActive returns true if the status is ACTIVE.
func (p ProjectParticipation) IsActive() bool { return p.status == ParticipationActive }

// IsValidAt checks if status is ACTIVE and current time falls within [validFrom, validTo].
func (p ProjectParticipation) IsValidAt(t time.Time) bool {
	if p.status != ParticipationActive {
		return false
	}
	return !t.Before(p.validFrom) && !t.After(p.validTo)
}

// EffectiveStatus computes operational status taking temporal expiry into account.
func (p ProjectParticipation) EffectiveStatus(t time.Time) ParticipationStatus {
	if p.status == ParticipationDeactivated {
		return ParticipationDeactivated
	}
	if p.status == ParticipationRevoked {
		return ParticipationRevoked
	}
	if t.After(p.validTo) || t.Before(p.validFrom) {
		return ParticipationExpired
	}
	return ParticipationActive
}

// ToRoleAssignment converts the project participation into an access policy RoleAssignment.
func (p ProjectParticipation) ToRoleAssignment() RoleAssignment {
	return RoleAssignment{
		Subject:  p.subject,
		TenantID: p.tenantID,
		Role:     p.role,
		Scope:    p.scope,
	}
}

// Deactivate transitions an active participation to DEACTIVATED in memory.
func (p ProjectParticipation) Deactivate(deactivatedBy, reason string, at time.Time) (ProjectParticipation, error) {
	if p.status == ParticipationDeactivated {
		return p, ErrParticipationDeactivated
	}
	trimmedActor := strings.TrimSpace(deactivatedBy)
	if trimmedActor == "" {
		return p, ErrBlankApprovalSource
	}
	trimmedReason := strings.TrimSpace(reason)
	if trimmedReason == "" {
		return p, errors.New("deactivation reason must not be blank")
	}

	updated := p
	updated.status = ParticipationDeactivated
	updated.deactivatedBy = trimmedActor
	updated.deactivationReason = trimmedReason
	updated.deactivatedAt = at.UTC()
	updated.updatedAt = at.UTC()
	return updated, nil
}

// NewProjectParticipation constructs and validates a new ProjectParticipation record.
func NewProjectParticipation(
	participationID, tenantID, subject, projectID string,
	role Role,
	scope ScopeGrant,
	assignedBy string,
	validFrom, validTo time.Time,
	isContractor bool,
) (ProjectParticipation, error) {
	trimmedID := strings.TrimSpace(participationID)
	if trimmedID == "" {
		return ProjectParticipation{}, ErrBlankParticipationID
	}
	trimmedTenant := strings.TrimSpace(tenantID)
	if trimmedTenant == "" {
		return ProjectParticipation{}, ErrBlankTenantID
	}
	if err := ValidateSubject(subject); err != nil {
		return ProjectParticipation{}, err
	}
	trimmedProject := strings.TrimSpace(projectID)
	if trimmedProject == "" {
		return ProjectParticipation{}, ErrBlankProjectID
	}
	if err := ValidateInternalSponsor(assignedBy); err != nil {
		return ProjectParticipation{}, err
	}

	if validTo.Before(validFrom) || validTo.Equal(validFrom) {
		return ProjectParticipation{}, ErrInvalidTimeWindow
	}

	// Bounded Contractor Administration Check: Contractors can NEVER hold administrative roles
	if isContractor || role == RoleContractor {
		if role == RoleTenantAdmin || role == RoleProjectManager {
			return ProjectParticipation{}, fmt.Errorf("%w: contractor cannot be assigned %s", ErrContractorAdminProhibited, role)
		}
	}

	// Scope Containment: Ensure scope is bounded to target tenant and project
	if scope.ProjectID != "" && scope.ProjectID != trimmedProject {
		return ProjectParticipation{}, fmt.Errorf("%w: scope project %s does not match participation project %s", ErrCrossProjectAccessDenied, scope.ProjectID, trimmedProject)
	}
	scope.TenantID = trimmedTenant
	scope.ProjectID = trimmedProject

	now := time.Now().UTC()
	return ProjectParticipation{
		participationID: trimmedID,
		tenantID:        trimmedTenant,
		subject:         strings.TrimSpace(subject),
		projectID:       trimmedProject,
		role:            role,
		scope:           scope,
		assignedBy:      strings.TrimSpace(assignedBy),
		validFrom:       validFrom.UTC(),
		validTo:         validTo.UTC(),
		status:          ParticipationActive,
		isContractor:    isContractor,
		createdAt:       now,
		updatedAt:       now,
	}, nil
}

// AssertContractorAdminBounds validates that an actor classified as a contractor cannot execute administrative actions.
func AssertContractorAdminBounds(role Role, isContractor bool, action Action, permission Permission) error {
	if !isContractor && role != RoleContractor {
		return nil
	}
	if role == RoleTenantAdmin || role == RoleProjectManager {
		return fmt.Errorf("%w: contractor cannot hold role %s", ErrContractorAdminProhibited, role)
	}
	if action == ActionDelete {
		return fmt.Errorf("%w: contractor cannot execute action %s", ErrContractorAdminProhibited, action)
	}

	adminPerms := map[Permission]bool{
		PermOrgTenantManage:        true,
		PermOrgProjectManage:       true,
		PermIdentityUserManage:     true,
		PermIdentityRoleAssign:     true,
		PermIdentitySessionRevoke:  true,
		PermInspectionApprove:      true,
		PermAuditExport:            true,
		PermLegalHoldManage:        true,
		PermPortalSnapshotPublish:  true,
		PermPortalSnapshotWithdraw: true,
		PermDelegationGrant:        true,
	}
	if adminPerms[permission] {
		return fmt.Errorf("%w: contractor cannot exercise permission %s", ErrContractorAdminProhibited, permission)
	}
	return nil
}

// AssertAuditorReadOnly validates that an actor holding the Auditor role cannot perform mutating operations.
func AssertAuditorReadOnly(role Role, action Action, permission Permission) error {
	if role != RoleAuditor {
		return nil
	}
	if action == ActionCreate || action == ActionUpdate || action == ActionDelete {
		return fmt.Errorf("%w: auditor cannot execute mutating action %s", ErrAuditorReadOnlyViolation, action)
	}

	mutatingPerms := map[Permission]bool{
		PermInspectionCreate:   true,
		PermInspectionSubmit:   true,
		PermInspectionReview:   true,
		PermInspectionApprove:  true,
		PermFindingCreate:      true,
		PermFindingRemediate:   true,
		PermFindingVerify:      true,
		PermRecordArchive:      true,
		PermOrgTenantManage:    true,
		PermOrgProjectManage:   true,
		PermIdentityUserManage: true,
		PermIdentityRoleAssign: true,
		PermDelegationGrant:    true,
	}
	if mutatingPerms[permission] {
		return fmt.Errorf("%w: auditor cannot exercise mutating permission %s", ErrAuditorReadOnlyViolation, permission)
	}
	return nil
}

// HistoricalAttributionRecord models an immutable, append-only historical audit entry
// for an operational action executed by an actor within a project, preserved permanently even after deactivation.
type HistoricalAttributionRecord struct {
	RecordID    string            `json:"record_id"`
	TenantID    string            `json:"tenant_id"`
	ProjectID   string            `json:"project_id"`
	Subject     string            `json:"subject"`
	DisplayName string            `json:"display_name"`
	RoleAtEvent Role              `json:"role_at_event"`
	ActionType  string            `json:"action_type"`
	ResourceID  string            `json:"resource_id"`
	Details     map[string]string `json:"details,omitempty"`
	RecordedAt  time.Time         `json:"recorded_at"`
}

// AttributionLedger provides an in-memory, thread-safe, append-only historical audit trail for actor attribution.
type AttributionLedger struct {
	mu      sync.RWMutex
	records []HistoricalAttributionRecord
}

// NewAttributionLedger initializes an empty in-memory ledger.
func NewAttributionLedger() *AttributionLedger {
	return &AttributionLedger{
		records: make([]HistoricalAttributionRecord, 0),
	}
}

// RecordAttribution appends an immutable attribution entry to the ledger.
func (l *AttributionLedger) RecordAttribution(record HistoricalAttributionRecord) error {
	if strings.TrimSpace(record.RecordID) == "" {
		return ErrBlankAttributionID
	}
	if strings.TrimSpace(record.TenantID) == "" {
		return ErrBlankTenantID
	}
	if strings.TrimSpace(record.ProjectID) == "" {
		return ErrBlankProjectID
	}
	if strings.TrimSpace(record.Subject) == "" {
		return ErrBlankSubject
	}
	if strings.TrimSpace(record.ActionType) == "" {
		return ErrBlankActionType
	}
	if strings.TrimSpace(record.ResourceID) == "" {
		return ErrBlankResourceID
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// Ensure record ID uniqueness to preserve immutability
	for _, r := range l.records {
		if r.TenantID == record.TenantID && r.RecordID == record.RecordID {
			return ErrAttributionImmutable
		}
	}

	copiedDetails := make(map[string]string)
	for k, v := range record.Details {
		copiedDetails[k] = v
	}
	record.Details = copiedDetails
	if record.RecordedAt.IsZero() {
		record.RecordedAt = time.Now().UTC()
	} else {
		record.RecordedAt = record.RecordedAt.UTC()
	}

	l.records = append(l.records, record)
	return nil
}

// GetAttributionTrail retrieves all attribution events for a specific resource under a tenant and project.
func (l *AttributionLedger) GetAttributionTrail(tenantID, projectID, resourceID string) ([]HistoricalAttributionRecord, error) {
	tTenant := strings.TrimSpace(tenantID)
	if tTenant == "" {
		return nil, ErrBlankTenantID
	}
	tProject := strings.TrimSpace(projectID)
	if tProject == "" {
		return nil, ErrBlankProjectID
	}
	tResource := strings.TrimSpace(resourceID)
	if tResource == "" {
		return nil, ErrBlankResourceID
	}

	l.mu.RLock()
	defer l.mu.RUnlock()

	var results []HistoricalAttributionRecord
	for _, r := range l.records {
		if r.TenantID == tTenant && r.ProjectID == tProject && r.ResourceID == tResource {
			results = append(results, r)
		}
	}
	return results, nil
}

// GetSubjectAttributionHistory retrieves all attribution events for a subject under a tenant.
func (l *AttributionLedger) GetSubjectAttributionHistory(tenantID, subject string) ([]HistoricalAttributionRecord, error) {
	tTenant := strings.TrimSpace(tenantID)
	if tTenant == "" {
		return nil, ErrBlankTenantID
	}
	tSubject := strings.TrimSpace(subject)
	if tSubject == "" {
		return nil, ErrBlankSubject
	}

	l.mu.RLock()
	defer l.mu.RUnlock()

	var results []HistoricalAttributionRecord
	for _, r := range l.records {
		if r.TenantID == tTenant && r.Subject == tSubject {
			results = append(results, r)
		}
	}
	return results, nil
}

// GetProjectAttributionHistory retrieves all attribution events for a project under a tenant.
func (l *AttributionLedger) GetProjectAttributionHistory(tenantID, projectID string) ([]HistoricalAttributionRecord, error) {
	tTenant := strings.TrimSpace(tenantID)
	if tTenant == "" {
		return nil, ErrBlankTenantID
	}
	tProject := strings.TrimSpace(projectID)
	if tProject == "" {
		return nil, ErrBlankProjectID
	}

	l.mu.RLock()
	defer l.mu.RUnlock()

	var results []HistoricalAttributionRecord
	for _, r := range l.records {
		if r.TenantID == tTenant && r.ProjectID == tProject {
			results = append(results, r)
		}
	}
	return results, nil
}

// MultiProjectParticipationRegistry provides a thread-safe, in-memory store for project participation bindings.
type MultiProjectParticipationRegistry struct {
	mu             sync.RWMutex
	participations map[string]ProjectParticipation // key: "tenantID:subject:projectID"
	ledger         *AttributionLedger
}

// NewMultiProjectParticipationRegistry initializes an in-memory participation registry.
func NewMultiProjectParticipationRegistry(ledger *AttributionLedger) *MultiProjectParticipationRegistry {
	if ledger == nil {
		ledger = NewAttributionLedger()
	}
	return &MultiProjectParticipationRegistry{
		participations: make(map[string]ProjectParticipation),
		ledger:         ledger,
	}
}

func makeParticipationKey(tenantID, subject, projectID string) string {
	return fmt.Sprintf("%s:%s:%s", strings.TrimSpace(tenantID), strings.TrimSpace(subject), strings.TrimSpace(projectID))
}

// AssignParticipation stores a new participation binding in memory.
func (r *MultiProjectParticipationRegistry) AssignParticipation(p ProjectParticipation) error {
	if p.TenantID() == "" || p.Subject() == "" || p.ProjectID() == "" {
		return ErrBlankParticipationID
	}

	key := makeParticipationKey(p.TenantID(), p.Subject(), p.ProjectID())

	r.mu.Lock()
	defer r.mu.Unlock()

	if existing, exists := r.participations[key]; exists && existing.IsActive() {
		return ErrDuplicateParticipation
	}

	r.participations[key] = p
	return nil
}

// DeactivateParticipation marks an active participation binding as DEACTIVATED in memory.
func (r *MultiProjectParticipationRegistry) DeactivateParticipation(tenantID, subject, projectID, deactivatedBy, reason string, at time.Time) (ProjectParticipation, error) {
	key := makeParticipationKey(tenantID, subject, projectID)

	r.mu.Lock()
	defer r.mu.Unlock()

	current, exists := r.participations[key]
	if !exists {
		return ProjectParticipation{}, ErrParticipationNotFound
	}

	updated, err := current.Deactivate(deactivatedBy, reason, at)
	if err != nil {
		return ProjectParticipation{}, err
	}

	r.participations[key] = updated
	return updated, nil
}

// GetParticipation retrieves a project participation binding by tenant, subject, and project.
func (r *MultiProjectParticipationRegistry) GetParticipation(tenantID, subject, projectID string) (ProjectParticipation, error) {
	key := makeParticipationKey(tenantID, subject, projectID)

	r.mu.RLock()
	defer r.mu.RUnlock()

	p, exists := r.participations[key]
	if !exists {
		return ProjectParticipation{}, ErrParticipationNotFound
	}
	return p, nil
}

// ListParticipationsBySubject returns all participation bindings for a subject under a tenant.
func (r *MultiProjectParticipationRegistry) ListParticipationsBySubject(tenantID, subject string) ([]ProjectParticipation, error) {
	tTenant := strings.TrimSpace(tenantID)
	if tTenant == "" {
		return nil, ErrBlankTenantID
	}
	tSubject := strings.TrimSpace(subject)
	if tSubject == "" {
		return nil, ErrBlankSubject
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	var results []ProjectParticipation
	for _, p := range r.participations {
		if p.TenantID() == tTenant && p.Subject() == tSubject {
			results = append(results, p)
		}
	}
	return results, nil
}

// ListActiveParticipationsBySubject returns all active, time-valid participation bindings for a subject at time 'at'.
func (r *MultiProjectParticipationRegistry) ListActiveParticipationsBySubject(tenantID, subject string, at time.Time) ([]ProjectParticipation, error) {
	all, err := r.ListParticipationsBySubject(tenantID, subject)
	if err != nil {
		return nil, err
	}
	var active []ProjectParticipation
	for _, p := range all {
		if p.IsValidAt(at) {
			active = append(active, p)
		}
	}
	return active, nil
}

// ListParticipationsByProject returns all participation bindings within a project under a tenant.
func (r *MultiProjectParticipationRegistry) ListParticipationsByProject(tenantID, projectID string) ([]ProjectParticipation, error) {
	tTenant := strings.TrimSpace(tenantID)
	if tTenant == "" {
		return nil, ErrBlankTenantID
	}
	tProject := strings.TrimSpace(projectID)
	if tProject == "" {
		return nil, ErrBlankProjectID
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	var results []ProjectParticipation
	for _, p := range r.participations {
		if p.TenantID() == tTenant && p.ProjectID() == tProject {
			results = append(results, p)
		}
	}
	return results, nil
}

// AttributionLedger returns the underlying historical attribution ledger.
func (r *MultiProjectParticipationRegistry) AttributionLedger() *AttributionLedger {
	return r.ledger
}

// RecordAttribution delegates recording of historical attribution to the underlying ledger.
func (r *MultiProjectParticipationRegistry) RecordAttribution(record HistoricalAttributionRecord) error {
	return r.ledger.RecordAttribution(record)
}

// GetAttributionTrail retrieves attribution history for a resource from the ledger.
func (r *MultiProjectParticipationRegistry) GetAttributionTrail(tenantID, projectID, resourceID string) ([]HistoricalAttributionRecord, error) {
	return r.ledger.GetAttributionTrail(tenantID, projectID, resourceID)
}

// GetSubjectAttributionHistory retrieves attribution history for a subject from the ledger.
func (r *MultiProjectParticipationRegistry) GetSubjectAttributionHistory(tenantID, subject string) ([]HistoricalAttributionRecord, error) {
	return r.ledger.GetSubjectAttributionHistory(tenantID, subject)
}

// GetProjectAttributionHistory retrieves attribution history for a project from the ledger.
func (r *MultiProjectParticipationRegistry) GetProjectAttributionHistory(tenantID, projectID string) ([]HistoricalAttributionRecord, error) {
	return r.ledger.GetProjectAttributionHistory(tenantID, projectID)
}

// EvaluateMultiProjectAccess evaluates an access request in the context of multi-project participation.
// Enforces:
// 1. Identity authentication and cross-tenant isolation.
// 2. Project participation existence: subject must hold active participation in target project.
// 3. Deactivation enforcement: deactivated or revoked participations fail closed.
// 4. Auditor read-only boundary: mutating actions (Create, Update, Delete) fail closed with DenialPrivilegeEscalation.
// 5. Contractor administration bounds: Delete and admin actions fail closed.
// 6. Scope containment: site and object locks must match target resource.
// 7. Role permissions: evaluates request using underlying PolicyEvaluator.
func (r *MultiProjectParticipationRegistry) EvaluateMultiProjectAccess(
	req AccessRequest,
	at time.Time,
) EvaluationResult {
	// 1. Basic Identity assertions
	if !req.Identity.IsAuthenticated || strings.TrimSpace(req.Identity.Subject) == "" || strings.TrimSpace(req.Identity.TenantID) == "" {
		return Deny(DenialUnauthenticated, "unauthenticated caller identity")
	}

	// 2. Cross-Tenant Denial
	if req.Target.TenantID != "" && req.Identity.TenantID != req.Target.TenantID {
		return Deny(DenialCrossTenant, "cross-tenant access prohibited")
	}

	// 3. Target Project Required
	targetProject := strings.TrimSpace(req.Target.ProjectID)
	if targetProject == "" {
		return Deny(DenialScopeMismatch, "target project must be specified in multi-project evaluation")
	}

	// 4. Retrieve participation for target project
	key := makeParticipationKey(req.Identity.TenantID, req.Identity.Subject, targetProject)
	r.mu.RLock()
	participation, exists := r.participations[key]
	r.mu.RUnlock()

	if !exists {
		return Deny(DenialScopeMismatch, "subject has no participation binding in target project: cross-project access denied")
	}

	// 5. Status & Temporal Validity
	if participation.Status() == ParticipationDeactivated {
		return Deny(DenialInactiveMembership, "participation in target project has been deactivated")
	}
	if participation.Status() == ParticipationRevoked {
		return Deny(DenialInactiveMembership, "participation in target project has been revoked")
	}
	if !participation.IsValidAt(at) {
		return Deny(DenialInactiveMembership, "participation temporal window has elapsed or is not yet active")
	}

	// 6. Archived Record Immutability
	if req.Target.Lifecycle == ResourceArchived && (req.Action == ActionUpdate || req.Action == ActionDelete) {
		return Deny(DenialArchivedRecord, "cannot modify archived resource")
	}

	// 7. Auditor Read-Only Boundary
	if participation.Role() == RoleAuditor {
		if req.Action == ActionCreate || req.Action == ActionUpdate || req.Action == ActionDelete {
			return Deny(DenialPrivilegeEscalation, "auditor role is strictly read-only: mutating operations prohibited")
		}
	}

	// 8. Contractor Bounds: cannot Delete
	if (participation.IsContractor() || participation.Role() == RoleContractor) && req.Action == ActionDelete {
		return Deny(DenialPrivilegeEscalation, "contractor role cannot delete resources")
	}

	// 9. Bounded Site Scope Match
	if participation.Scope().SiteID != "" && req.Target.SiteID != "" && participation.Scope().SiteID != req.Target.SiteID {
		return Deny(DenialScopeMismatch, "target site is outside bounded participation site scope")
	}

	// 10. Direct Object Lock Match
	if participation.Scope().ObjectID != "" && req.Target.ObjectID != "" && participation.Scope().ObjectID != req.Target.ObjectID {
		return Deny(DenialDirectObjectMismatch, "target object is outside direct object lock")
	}

	// 11. Role Permission Evaluation using PolicyEvaluator
	scopedEval := NewPolicyEvaluator()
	scopedEval.SetMembership(req.Identity.TenantID, req.Identity.Subject, MembershipActive)
	scopedEval.AddRoleAssignment(participation.ToRoleAssignment())

	return scopedEval.Evaluate(req)
}
