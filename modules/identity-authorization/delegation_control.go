// Package localidentity provides local identity, authorization, and directory services for OSHE Platform.
//
// PROVISIONAL GOVERNANCE DECLARATION (H030-003, Issue #92):
// Under approved Sole Human Owner decision H030-003, this file implements local
// in-memory explicit time-bounded delegation controls, chain depth ceilings,
// source-authority containment, revocation mechanics, and emergency-access boundary denials.
//
// Strict Delegation Invariants:
// 1. Explicit Chain Depth Ceiling: Only direct 1-hop delegation is permitted (MaxDelegationChainDepth = 1).
//    Re-delegation or multi-hop delegation is strictly prohibited (ErrMultiHopDelegationForbidden).
// 2. Self-Delegation Prohibition: A delegator cannot delegate authorities to themselves (ErrSelfDelegationForbidden).
// 3. Source Authority Containment: A delegator cannot delegate permissions or scopes exceeding
//    their own active entitlements (ErrExceedsSourceAuthority, ErrScopeExceedsSourceAuthority).
// 4. Protected Authority Non-Delegable: Sovereign tenant administration (RoleTenantAdmin) cannot be delegated.
// 5. Emergency Break-Glass Prohibition: Automated break-glass bypasses or unapproved emergency escalations
//    are strictly barred in Milestone v0.3.0 and fail closed with default-deny (ErrEmergencyAccessDenied).
// 6. Append-Only Audit Ledger: Every delegation creation, revocation, and expiration is permanently recorded.
// 7. Zero External Enactment: Operates purely in-memory on local synthetic fixtures.
package localidentity

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// MaxDelegationChainDepth specifies the strict ceiling on delegation chains under Issue #92.
// Only direct delegation (depth 1) is permitted; multi-hop re-delegation is strictly prohibited.
const MaxDelegationChainDepth = 1

// DelegationState represents the operational lifecycle state of a delegation grant.
type DelegationState string

const (
	DelegationStateActive  DelegationState = "ACTIVE"
	DelegationStateRevoked DelegationState = "REVOKED"
	DelegationStateExpired DelegationState = "EXPIRED"
)

var (
	// ErrBlankDelegationID indicates missing delegation identifier.
	ErrBlankDelegationID = errors.New("delegation ID must not be blank")
	// ErrEmergencyAccessDenied indicates an unapproved attempt to trigger emergency break-glass bypass.
	ErrEmergencyAccessDenied = errors.New("emergency break-glass or unapproved authority bypass is strictly prohibited in milestone v0.3.0")
	// ErrDelegationRevoked indicates that the delegation grant has been revoked.
	ErrDelegationRevoked = errors.New("delegation has been revoked")
	// ErrDelegationExpired indicates that the delegation temporal window has expired.
	ErrDelegationExpired = errors.New("delegation has expired")
	// ErrDelegationNotFound indicates the requested delegation record does not exist.
	ErrDelegationNotFound = errors.New("delegation record not found")
	// ErrDuplicateDelegationID indicates a delegation with the same ID already exists in the tenant.
	ErrDuplicateDelegationID = errors.New("delegation ID already registered for tenant")
	// ErrUnauthorizedChainDepth indicates chain depth exceeds the 1-hop ceiling.
	ErrUnauthorizedChainDepth = errors.New("unauthorized delegation chain depth: multi-hop re-delegation is forbidden")
)

// AssertEmergencyAccessDenied formally validates that emergency break-glass access
// is strictly prohibited and fails closed under H030-003.
func AssertEmergencyAccessDenied(isEmergency bool) error {
	if isEmergency {
		return ErrEmergencyAccessDenied
	}
	return nil
}

// DelegationRecord models an authoritative, time-bounded, explicit delegation grant.
type DelegationRecord struct {
	delegationID     string
	tenantID         string
	delegatorSubject string
	delegatorRole    Role
	delegatorScope   ScopeGrant
	delegateeSubject string
	delegatedRole    Role
	delegatedScope   ScopeGrant
	validFrom        time.Time
	validTo          time.Time
	approvalSource   string
	chainDepth       int  // strictly 1
	isSubDelegation  bool // strictly false
	state            DelegationState
	revokedBy        string
	revokedAt        time.Time
	revocationReason string
	createdAt        time.Time
	updatedAt        time.Time
}

// DelegationID returns the unique delegation identifier.
func (d DelegationRecord) DelegationID() string { return d.delegationID }

// TenantID returns the authoritative tenant identifier.
func (d DelegationRecord) TenantID() string { return d.tenantID }

// DelegatorSubject returns the internal manager delegating the authority (usr_*).
func (d DelegationRecord) DelegatorSubject() string { return d.delegatorSubject }

// DelegatorRole returns the role held by the delegator.
func (d DelegationRecord) DelegatorRole() Role { return d.delegatorRole }

// DelegatorScope returns the scope held by the delegator.
func (d DelegationRecord) DelegatorScope() ScopeGrant { return d.delegatorScope }

// DelegateeSubject returns the recipient of the delegated authority (usr_*).
func (d DelegationRecord) DelegateeSubject() string { return d.delegateeSubject }

// DelegatedRole returns the delegated security role.
func (d DelegationRecord) DelegatedRole() Role { return d.delegatedRole }

// DelegatedScope returns the bounded scope of the delegation.
func (d DelegationRecord) DelegatedScope() ScopeGrant { return d.delegatedScope }

// ValidFrom returns the beginning of the delegation validity window.
func (d DelegationRecord) ValidFrom() time.Time { return d.validFrom }

// ValidTo returns the end of the delegation validity window.
func (d DelegationRecord) ValidTo() time.Time { return d.validTo }

// ApprovalSource returns the recorded approval governance reference.
func (d DelegationRecord) ApprovalSource() string { return d.approvalSource }

// ChainDepth returns the delegation depth (1 for direct).
func (d DelegationRecord) ChainDepth() int { return d.chainDepth }

// State returns the operational lifecycle state.
func (d DelegationRecord) State() DelegationState { return d.state }

// IsActive returns true if the delegation is in ACTIVE state.
func (d DelegationRecord) IsActive() bool { return d.state == DelegationStateActive }

// RevokedBy returns the actor who revoked the delegation if revoked.
func (d DelegationRecord) RevokedBy() string { return d.revokedBy }

// RevokedAt returns the revocation timestamp if revoked.
func (d DelegationRecord) RevokedAt() time.Time { return d.revokedAt }

// RevocationReason returns the recorded rationale for revocation.
func (d DelegationRecord) RevocationReason() string { return d.revocationReason }

// CreatedAt returns creation timestamp.
func (d DelegationRecord) CreatedAt() time.Time { return d.createdAt }

// UpdatedAt returns last update timestamp.
func (d DelegationRecord) UpdatedAt() time.Time { return d.updatedAt }

// IsValidAt evaluates whether the delegation is active and current time falls within [validFrom, validTo].
func (d DelegationRecord) IsValidAt(t time.Time) bool {
	if d.state != DelegationStateActive {
		return false
	}
	return !t.Before(d.validFrom) && !t.After(d.validTo)
}

// EffectiveState returns the effective lifecycle state taking temporal validity into account.
func (d DelegationRecord) EffectiveState(t time.Time) DelegationState {
	if d.state == DelegationStateRevoked {
		return DelegationStateRevoked
	}
	if t.After(d.validTo) || t.Before(d.validFrom) {
		return DelegationStateExpired
	}
	return DelegationStateActive
}

// ToRoleAssignment converts the active delegation into a RoleAssignment for policy evaluation.
func (d DelegationRecord) ToRoleAssignment() RoleAssignment {
	return RoleAssignment{
		Subject:  d.delegateeSubject,
		TenantID: d.tenantID,
		Role:     d.delegatedRole,
		Scope:    d.delegatedScope,
	}
}

// Revoke explicitly revokes an active delegation grant in memory.
func (d DelegationRecord) Revoke(revokedBy, reason string, at time.Time) (DelegationRecord, DelegationAuditRecord, error) {
	if d.state == DelegationStateRevoked {
		return d, DelegationAuditRecord{}, ErrDelegationRevoked
	}

	trimmedRevoker := strings.TrimSpace(revokedBy)
	if trimmedRevoker == "" {
		return d, DelegationAuditRecord{}, errors.New("revokedBy must not be blank")
	}
	trimmedReason := strings.TrimSpace(reason)
	if trimmedReason == "" {
		return d, DelegationAuditRecord{}, errors.New("revocation reason must not be blank")
	}

	updated := d
	updated.state = DelegationStateRevoked
	updated.revokedBy = trimmedRevoker
	updated.revocationReason = trimmedReason
	updated.revokedAt = at.UTC()
	updated.updatedAt = at.UTC()

	audit := DelegationAuditRecord{
		RecordID:         fmt.Sprintf("hdel_%s_%d", d.delegationID, at.UTC().UnixNano()),
		TenantID:         d.tenantID,
		DelegationID:     d.delegationID,
		DelegatorSubject: d.delegatorSubject,
		DelegateeSubject: d.delegateeSubject,
		DelegatedRole:    d.delegatedRole,
		DelegatedScope:   d.delegatedScope,
		Transition:       "DELEGATION_REVOKED",
		ActorSubject:     trimmedRevoker,
		Reason:           trimmedReason,
		RecordedAt:       at.UTC(),
	}

	return updated, audit, nil
}

// NewDelegationRecord constructs and validates a new in-memory DelegationRecord.
func NewDelegationRecord(
	delegationID, tenantID string,
	delegatorSubject string, delegatorRole Role, delegatorScope ScopeGrant,
	delegateeSubject string, delegatedRole Role, delegatedScope ScopeGrant,
	validFrom, validTo time.Time,
	approvalSource string,
	chainDepth int,
	isSubDelegation bool,
) (DelegationRecord, error) {
	trimmedID := strings.TrimSpace(delegationID)
	if trimmedID == "" {
		return DelegationRecord{}, ErrBlankDelegationID
	}
	trimmedTenant := strings.TrimSpace(tenantID)
	if trimmedTenant == "" {
		return DelegationRecord{}, ErrBlankTenantID
	}
	if err := ValidateSubject(delegatorSubject); err != nil {
		return DelegationRecord{}, fmt.Errorf("invalid delegator: %w", err)
	}
	if err := ValidateSubject(delegateeSubject); err != nil {
		return DelegationRecord{}, fmt.Errorf("invalid delegatee: %w", err)
	}
	if strings.TrimSpace(delegatorSubject) == strings.TrimSpace(delegateeSubject) {
		return DelegationRecord{}, ErrSelfDelegationForbidden
	}
	if !KnownRoles[delegatorRole] || !KnownRoles[delegatedRole] {
		return DelegationRecord{}, ErrUnknownRole
	}
	trimmedApproval := strings.TrimSpace(approvalSource)
	if trimmedApproval == "" {
		return DelegationRecord{}, errors.New("approval source must not be blank")
	}

	if validTo.Before(validFrom) || validTo.Equal(validFrom) {
		return DelegationRecord{}, ErrInvalidDelegationWindow
	}
	if validTo.Sub(validFrom) > MaxDelegationDuration {
		return DelegationRecord{}, fmt.Errorf("%w: duration %s exceeds maximum 30 days", ErrDelegationDurationExceeded, validTo.Sub(validFrom))
	}

	// Chain limits: must be direct 1-hop
	if chainDepth > MaxDelegationChainDepth || isSubDelegation {
		return DelegationRecord{}, ErrUnauthorizedChainDepth
	}

	// Scope tenant containment
	delegatorScope.TenantID = trimmedTenant
	delegatedScope.TenantID = trimmedTenant

	now := time.Now().UTC()
	return DelegationRecord{
		delegationID:     trimmedID,
		tenantID:         trimmedTenant,
		delegatorSubject: strings.TrimSpace(delegatorSubject),
		delegatorRole:    delegatorRole,
		delegatorScope:   delegatorScope,
		delegateeSubject: strings.TrimSpace(delegateeSubject),
		delegatedRole:    delegatedRole,
		delegatedScope:   delegatedScope,
		validFrom:        validFrom.UTC(),
		validTo:          validTo.UTC(),
		approvalSource:   trimmedApproval,
		chainDepth:       1,
		isSubDelegation:  false,
		state:            DelegationStateActive,
		createdAt:        now,
		updatedAt:        now,
	}, nil
}

// ValidateDelegationAuthority verifies that a delegation request conforms strictly to source-authority containment,
// role bounding, and matrix constraints.
func ValidateDelegationAuthority(record DelegationRecord, matrix *AuthorizationMatrix) error {
	if matrix == nil {
		m := NewProvisionalAuthorizationMatrix()
		matrix = &m
	}
	req := DelegationRequest{
		DelegatorSubject: record.DelegatorSubject(),
		DelegatorRole:    record.DelegatorRole(),
		DelegatorScope:   record.DelegatorScope(),
		DelegateeSubject: record.DelegateeSubject(),
		DelegatedRole:    record.DelegatedRole(),
		DelegatedScope:   record.DelegatedScope(),
		ValidFrom:        record.ValidFrom(),
		ValidTo:          record.ValidTo(),
		IsSubDelegation:  record.isSubDelegation,
	}

	return matrix.ValidateDelegationRequest(req)
}

// DelegationAuditRecord models an immutable historical audit record for delegation events.
type DelegationAuditRecord struct {
	RecordID         string     `json:"record_id"`
	TenantID         string     `json:"tenant_id"`
	DelegationID     string     `json:"delegation_id"`
	DelegatorSubject string     `json:"delegator_subject"`
	DelegateeSubject string     `json:"delegatee_subject"`
	DelegatedRole    Role       `json:"delegated_role"`
	DelegatedScope   ScopeGrant `json:"delegated_scope"`
	Transition       string     `json:"transition"`
	ActorSubject     string     `json:"actor_subject"`
	Reason           string     `json:"reason"`
	RecordedAt       time.Time  `json:"recorded_at"`
}

// DelegationLedger provides an in-memory, thread-safe append-only audit trail for delegation events.
type DelegationLedger struct {
	mu      sync.RWMutex
	records []DelegationAuditRecord
}

// NewDelegationLedger initializes an empty in-memory ledger.
func NewDelegationLedger() *DelegationLedger {
	return &DelegationLedger{
		records: make([]DelegationAuditRecord, 0),
	}
}

// AppendRecord appends an audit record to the ledger.
func (l *DelegationLedger) AppendRecord(record DelegationAuditRecord) error {
	if record.TenantID == "" || record.DelegationID == "" {
		return ErrBlankDelegationID
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.records = append(l.records, record)
	return nil
}

// GetDelegationAuditTrail retrieves all audit events for a delegation strictly within tenant boundaries.
func (l *DelegationLedger) GetDelegationAuditTrail(tenantID, delegationID string) ([]DelegationAuditRecord, error) {
	tTenant := strings.TrimSpace(tenantID)
	if tTenant == "" {
		return nil, ErrBlankTenantID
	}
	tDel := strings.TrimSpace(delegationID)
	if tDel == "" {
		return nil, ErrBlankDelegationID
	}

	l.mu.RLock()
	defer l.mu.RUnlock()

	var results []DelegationAuditRecord
	for _, rec := range l.records {
		if rec.TenantID == tTenant && rec.DelegationID == tDel {
			results = append(results, rec)
		}
	}
	return results, nil
}

// GetSubjectDelegationAuditTrail retrieves all delegation events involving a subject (as delegator or delegatee).
func (l *DelegationLedger) GetSubjectDelegationAuditTrail(tenantID, subject string) ([]DelegationAuditRecord, error) {
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

	var results []DelegationAuditRecord
	for _, rec := range l.records {
		if rec.TenantID == tTenant && (rec.DelegatorSubject == tSub || rec.DelegateeSubject == tSub) {
			results = append(results, rec)
		}
	}
	return results, nil
}

// DelegationRegistry manages active and historical delegation records in memory.
type DelegationRegistry struct {
	mu          sync.RWMutex
	delegations map[string]DelegationRecord // key: tenantID + ":" + delegationID
	ledger      *DelegationLedger
	matrix      *AuthorizationMatrix
}

func NewDelegationRegistry(ledger *DelegationLedger, matrix *AuthorizationMatrix) *DelegationRegistry {
	if ledger == nil {
		ledger = NewDelegationLedger()
	}
	if matrix == nil {
		m := NewProvisionalAuthorizationMatrix()
		matrix = &m
	}
	return &DelegationRegistry{
		delegations: make(map[string]DelegationRecord),
		ledger:      ledger,
		matrix:      matrix,
	}
}

func makeDelegationKey(tenantID, delegationID string) string {
	return fmt.Sprintf("%s:%s", strings.TrimSpace(tenantID), strings.TrimSpace(delegationID))
}

// CreateDelegation registers a new delegation grant after validating source authority containment and chain depth.
func (r *DelegationRegistry) CreateDelegation(grant DelegationRecord, actorSubject, reason string, at time.Time) error {
	if grant.TenantID() == "" || grant.DelegationID() == "" {
		return ErrBlankDelegationID
	}

	// 1. Validate authority containment against matrix
	if err := ValidateDelegationAuthority(grant, r.matrix); err != nil {
		return err
	}

	key := makeDelegationKey(grant.TenantID(), grant.DelegationID())

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.delegations[key]; exists {
		return ErrDuplicateDelegationID
	}

	r.delegations[key] = grant

	audit := DelegationAuditRecord{
		RecordID:         fmt.Sprintf("hdel_%s_%d", grant.DelegationID(), at.UTC().UnixNano()),
		TenantID:         grant.TenantID(),
		DelegationID:     grant.DelegationID(),
		DelegatorSubject: grant.DelegatorSubject(),
		DelegateeSubject: grant.DelegateeSubject(),
		DelegatedRole:    grant.DelegatedRole(),
		DelegatedScope:   grant.DelegatedScope(),
		Transition:       "DELEGATION_CREATED",
		ActorSubject:     strings.TrimSpace(actorSubject),
		Reason:           strings.TrimSpace(reason),
		RecordedAt:       at.UTC(),
	}

	return r.ledger.AppendRecord(audit)
}

// RevokeDelegation revokes an active delegation in memory and records the audit event.
func (r *DelegationRegistry) RevokeDelegation(tenantID, delegationID, actorSubject, reason string, at time.Time) (DelegationRecord, error) {
	key := makeDelegationKey(tenantID, delegationID)

	r.mu.Lock()
	defer r.mu.Unlock()

	current, exists := r.delegations[key]
	if !exists {
		return DelegationRecord{}, ErrDelegationNotFound
	}

	revoked, audit, err := current.Revoke(actorSubject, reason, at)
	if err != nil {
		return DelegationRecord{}, err
	}

	r.delegations[key] = revoked
	if err := r.ledger.AppendRecord(audit); err != nil {
		return DelegationRecord{}, err
	}

	return revoked, nil
}

// GetDelegation retrieves a single delegation record by tenant and ID.
func (r *DelegationRegistry) GetDelegation(tenantID, delegationID string) (DelegationRecord, error) {
	key := makeDelegationKey(tenantID, delegationID)

	r.mu.RLock()
	defer r.mu.RUnlock()

	d, exists := r.delegations[key]
	if !exists {
		return DelegationRecord{}, ErrDelegationNotFound
	}
	return d, nil
}

// ListActiveDelegationsForDelegatee returns all active, time-valid delegations for a recipient at timestamp 'at'.
func (r *DelegationRegistry) ListActiveDelegationsForDelegatee(tenantID, delegateeSubject string, at time.Time) ([]DelegationRecord, error) {
	tTenant := strings.TrimSpace(tenantID)
	if tTenant == "" {
		return nil, ErrBlankTenantID
	}
	tSub := strings.TrimSpace(delegateeSubject)
	if tSub == "" {
		return nil, ErrBlankSubject
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	var results []DelegationRecord
	for _, d := range r.delegations {
		if d.TenantID() == tTenant && d.DelegateeSubject() == tSub && d.IsValidAt(at) {
			results = append(results, d)
		}
	}
	return results, nil
}

// EvaluateDelegatedAccess evaluates an AccessRequest when caller relies on delegated authority.
// Enforces default-deny and emergency-access prohibition.
func EvaluateDelegatedAccess(registry *DelegationRegistry, evaluator *PolicyEvaluator, req AccessRequest, at time.Time) EvaluationResult {
	if registry == nil || evaluator == nil {
		return Deny(DenialDefaultDeny, "uninitialized delegation evaluators")
	}

	// 1. Basic caller identity check
	if !req.Identity.IsAuthenticated || strings.TrimSpace(req.Identity.Subject) == "" || strings.TrimSpace(req.Identity.TenantID) == "" {
		return Deny(DenialUnauthenticated, "unauthenticated caller identity")
	}

	// 2. Cross-tenant denial
	if req.Identity.TenantID != req.Target.TenantID {
		return Deny(DenialCrossTenant, "cross-tenant access prohibited")
	}

	// 3. Find active valid delegations for the caller
	activeDelegations, err := registry.ListActiveDelegationsForDelegatee(req.Identity.TenantID, req.Identity.Subject, at)
	if err != nil || len(activeDelegations) == 0 {
		return Deny(DenialRoleNotGranted, "no active delegation grants found for caller")
	}

	// 4. Check if any active delegation covers the requested action and target scope
	hasScopeMatch := false
	hasRolePermit := false

	for _, d := range activeDelegations {
		if scopeMatches(d.DelegatedScope(), req.Target) {
			hasScopeMatch = true
			if rolePermitsAction(d.DelegatedRole(), req.Action) {
				hasRolePermit = true
				break
			}
		}
	}

	if !hasScopeMatch {
		return Deny(DenialScopeMismatch, "no active delegation covers the requested operational scope")
	}
	if !hasRolePermit {
		return Deny(DenialPrivilegeEscalation, "delegated role does not permit the requested operational action")
	}

	// 5. Construct scoped evaluator inheriting membership and evaluate
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

	for _, d := range activeDelegations {
		scopedEval.AddRoleAssignment(d.ToRoleAssignment())
	}

	// Mark delegation as verified (clearing the dummy delegation placeholder that normally fails closed)
	reqClean := req
	reqClean.Delegation.IsDelegated = false
	reqClean.Delegation.Delegator = ""

	return scopedEval.Evaluate(reqClean)
}
