package localidentity

import (
	"strings"
	"sync"
)

// Role represents an authoritative discrete security role.
type Role string

const (
	RoleTenantAdmin    Role = "TENANT_ADMIN"
	RoleProjectManager Role = "PROJECT_MANAGER"
	RoleInspector      Role = "INSPECTOR"
	RoleAuditor        Role = "AUDITOR"
	RoleViewer         Role = "VIEWER"
	RoleContractor     Role = "CONTRACTOR"
	RoleSupport        Role = "SUPPORT"
	RoleUnknown        Role = "UNKNOWN"
)

// KnownRoles is the authoritative set of recognized system roles.
var KnownRoles = map[Role]bool{
	RoleTenantAdmin:    true,
	RoleProjectManager: true,
	RoleInspector:      true,
	RoleAuditor:        true,
	RoleViewer:         true,
	RoleContractor:     true,
	RoleSupport:        true,
}

// Action represents an operational verb requested on a target resource.
type Action string

const (
	ActionRead   Action = "READ"
	ActionCreate Action = "CREATE"
	ActionUpdate Action = "UPDATE"
	ActionDelete Action = "DELETE"
	ActionExport Action = "EXPORT"
)

// MembershipStatus defines whether a subject holds active membership in a tenant.
type MembershipStatus string

const (
	MembershipActive    MembershipStatus = "ACTIVE"
	MembershipSuspended MembershipStatus = "SUSPENDED"
	MembershipInactive  MembershipStatus = "INACTIVE"
)

// ResourceLifecycle represents the lifecycle state of a target resource record.
type ResourceLifecycle string

const (
	ResourceActive   ResourceLifecycle = "ACTIVE"
	ResourceArchived ResourceLifecycle = "ARCHIVED"
)

// DenialReason defines stable, explainable, typed denial reasons that never leak
// other-tenant or sibling-project object details.
type DenialReason string

const (
	DenialNone                     DenialReason = "NONE"
	DenialUnauthenticated          DenialReason = "UNAUTHENTICATED_IDENTITY"
	DenialCrossTenant              DenialReason = "CROSS_TENANT_ACCESS_DENIED"
	DenialInactiveMembership       DenialReason = "INACTIVE_MEMBERSHIP"
	DenialUnknownRole              DenialReason = "UNKNOWN_ROLE_DENIED"
	DenialRoleNotGranted           DenialReason = "ROLE_NOT_GRANTED"
	DenialScopeMismatch            DenialReason = "SCOPE_MISMATCH"
	DenialDirectObjectMismatch     DenialReason = "DIRECT_OBJECT_MISMATCH"
	DenialArchivedRecord           DenialReason = "ARCHIVED_RECORD_ACCESS_DENIED"
	DenialDelegationNotImplemented DenialReason = "DELEGATION_NOT_IMPLEMENTED"
	DenialMissingEntitlement       DenialReason = "MISSING_REQUIRED_ENTITLEMENT"
	DenialPrivilegeEscalation      DenialReason = "PRIVILEGE_ESCALATION_DENIED"
	DenialDefaultDeny              DenialReason = "DEFAULT_DENY"
)

// SubjectIdentity represents an authenticated caller identity.
type SubjectIdentity struct {
	Subject         string
	TenantID        string
	IsAuthenticated bool
}

// Membership binds a subject to a tenant with an operational lifecycle status.
type Membership struct {
	Subject  string
	TenantID string
	Status   MembershipStatus
}

// ScopeGrant defines an explicitly bounded organizational hierarchy scope.
type ScopeGrant struct {
	TenantID  string
	CompanyID string
	ProjectID string
	SiteID    string
	AreaID    string
	ObjectID  string // Optional direct object lock
}

// RoleAssignment binds a role and an explicit scope to a subject in a tenant.
type RoleAssignment struct {
	Subject  string
	TenantID string
	Role     Role
	Scope    ScopeGrant
}

// TargetResource describes the target object of an evaluation request.
type TargetResource struct {
	TenantID  string
	CompanyID string
	ProjectID string
	SiteID    string
	AreaID    string
	ObjectID  string
	Lifecycle ResourceLifecycle
}

// DelegationContext holds delegation placeholder data.
type DelegationContext struct {
	IsDelegated bool
	Delegator   string
}

// AccessRequest contains all predicates needed for authoritative policy evaluation.
type AccessRequest struct {
	Identity            SubjectIdentity
	Target              TargetResource
	Action              Action
	Delegation          DelegationContext
	RequiredEntitlement string
}

// EvaluationResult encapsulates the policy decision with typed denial rationale.
type EvaluationResult struct {
	Allowed      bool         `json:"allowed"`
	DenialReason DenialReason `json:"denial_reason"`
	Message      string       `json:"message"`
}

func Allow() EvaluationResult {
	return EvaluationResult{
		Allowed:      true,
		DenialReason: DenialNone,
		Message:      "access granted",
	}
}

func Deny(reason DenialReason, message string) EvaluationResult {
	return EvaluationResult{
		Allowed:      false,
		DenialReason: reason,
		Message:      message,
	}
}

// PolicyEvaluator coordinates in-memory membership, role assignments, entitlements,
// and evaluates access requests using strict default-deny semantics.
type PolicyEvaluator struct {
	mu           sync.RWMutex
	memberships  map[string]MembershipStatus // key: "tenantID:subject"
	assignments  map[string][]RoleAssignment // key: "tenantID:subject"
	entitlements map[string]map[string]bool  // key: tenantID -> set of entitlements
}

// NewPolicyEvaluator constructs an initialized in-memory PolicyEvaluator.
func NewPolicyEvaluator() *PolicyEvaluator {
	return &PolicyEvaluator{
		memberships:  make(map[string]MembershipStatus),
		assignments:  make(map[string][]RoleAssignment),
		entitlements: make(map[string]map[string]bool),
	}
}

// SetMembership sets or updates the membership status for a subject in a tenant.
func (e *PolicyEvaluator) SetMembership(tenantID, subject string, status MembershipStatus) {
	e.mu.Lock()
	defer e.mu.Unlock()
	key := strings.TrimSpace(tenantID) + ":" + strings.TrimSpace(subject)
	e.memberships[key] = status
}

// AddRoleAssignment registers an explicit role and scope grant for a subject.
func (e *PolicyEvaluator) AddRoleAssignment(assignment RoleAssignment) {
	e.mu.Lock()
	defer e.mu.Unlock()
	key := strings.TrimSpace(assignment.TenantID) + ":" + strings.TrimSpace(assignment.Subject)
	e.assignments[key] = append(e.assignments[key], assignment)
}

// SetEntitlement sets the availability of a feature entitlement for a tenant.
func (e *PolicyEvaluator) SetEntitlement(tenantID, entitlement string, enabled bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	t := strings.TrimSpace(tenantID)
	if _, exists := e.entitlements[t]; !exists {
		e.entitlements[t] = make(map[string]bool)
	}
	e.entitlements[t][strings.TrimSpace(entitlement)] = enabled
}

func rolePermitsAction(role Role, action Action) bool {
	switch role {
	case RoleTenantAdmin:
		return true
	case RoleProjectManager:
		return action == ActionRead || action == ActionCreate || action == ActionUpdate || action == ActionExport
	case RoleInspector:
		return action == ActionRead || action == ActionCreate || action == ActionUpdate
	case RoleAuditor:
		return action == ActionRead || action == ActionExport
	case RoleViewer:
		return action == ActionRead
	case RoleContractor:
		return action == ActionRead || action == ActionCreate
	case RoleSupport:
		return action == ActionRead
	default:
		return false
	}
}

func scopeMatches(grant ScopeGrant, target TargetResource) bool {
	if grant.TenantID != "" && grant.TenantID != target.TenantID {
		return false
	}
	if grant.CompanyID != "" && grant.CompanyID != target.CompanyID {
		return false
	}
	if grant.ProjectID != "" && grant.ProjectID != target.ProjectID {
		return false
	}
	if grant.SiteID != "" && grant.SiteID != target.SiteID {
		return false
	}
	if grant.AreaID != "" && grant.AreaID != target.AreaID {
		return false
	}
	return true
}

// Evaluate evaluates an AccessRequest against default-deny policy.
// Denies unless authenticated identity, active membership, explicit role grant,
// explicit scope grant, record lifecycle, and entitlement all pass.
// Delegation placeholders strictly deny.
func (e *PolicyEvaluator) Evaluate(req AccessRequest) EvaluationResult {
	// 1. Authenticated Identity check
	if !req.Identity.IsAuthenticated || strings.TrimSpace(req.Identity.Subject) == "" || strings.TrimSpace(req.Identity.TenantID) == "" {
		return Deny(DenialUnauthenticated, "caller identity is unauthenticated or missing required subject")
	}

	// 2. Cross-Tenant Boundary check
	callerTenant := strings.TrimSpace(req.Identity.TenantID)
	targetTenant := strings.TrimSpace(req.Target.TenantID)
	if targetTenant == "" || callerTenant != targetTenant {
		return Deny(DenialCrossTenant, "tenant boundary mismatch")
	}

	// 3. Delegation Placeholder check (Unimplemented must fail closed)
	if req.Delegation.IsDelegated || strings.TrimSpace(req.Delegation.Delegator) != "" {
		return Deny(DenialDelegationNotImplemented, "delegation placeholder is not implemented and fails closed")
	}

	// 4. Active Membership check
	subject := strings.TrimSpace(req.Identity.Subject)
	memberKey := callerTenant + ":" + subject

	e.mu.RLock()
	memStatus, hasMember := e.memberships[memberKey]
	assignments := e.assignments[memberKey]
	tenantEnts := e.entitlements[callerTenant]
	e.mu.RUnlock()

	if !hasMember || memStatus != MembershipActive {
		return Deny(DenialInactiveMembership, "subject membership is not active in tenant")
	}

	// 5. Entitlement Check (Entitlement does not replace authorization)
	if req.RequiredEntitlement != "" {
		if tenantEnts == nil || !tenantEnts[req.RequiredEntitlement] {
			return Deny(DenialMissingEntitlement, "required tenant entitlement is not active")
		}
	}

	// 6. Record Lifecycle check
	if req.Target.Lifecycle == ResourceArchived {
		if req.Action != ActionRead {
			return Deny(DenialArchivedRecord, "archived record cannot be modified")
		}
	}

	// 7. Role Grants and Scope Evaluation
	if len(assignments) == 0 {
		return Deny(DenialRoleNotGranted, "no role granted to subject in tenant")
	}

	var hasScopeMismatch bool
	var hasDirectObjectMismatch bool
	var hasPrivilegeEscalation bool
	var hasUnknownRole bool

	for _, grant := range assignments {
		// Check for unknown role
		if !KnownRoles[grant.Role] {
			hasUnknownRole = true
			continue
		}

		// Prohibit implicit company-wide or tenant-wide access for project roles, contractors, and support
		if grant.Role == RoleProjectManager || grant.Role == RoleInspector || grant.Role == RoleContractor || grant.Role == RoleSupport {
			if strings.TrimSpace(grant.Scope.ProjectID) == "" && strings.TrimSpace(grant.Scope.SiteID) == "" {
				hasScopeMismatch = true
				continue
			}
		}

		// Scope checking (prohibits sibling-project access)
		if !scopeMatches(grant.Scope, req.Target) {
			hasScopeMismatch = true
			continue
		}

		// Direct object checking
		if grant.Scope.ObjectID != "" && grant.Scope.ObjectID != req.Target.ObjectID {
			hasDirectObjectMismatch = true
			continue
		}

		// Role action checking (privilege)
		if !rolePermitsAction(grant.Role, req.Action) {
			hasPrivilegeEscalation = true
			continue
		}

		// If record is archived, even READ is only permitted for Auditor or TenantAdmin
		if req.Target.Lifecycle == ResourceArchived {
			if grant.Role != RoleAuditor && grant.Role != RoleTenantAdmin {
				return Deny(DenialArchivedRecord, "archived record access requires auditor or admin role")
			}
		}

		// All predicates passed!
		return Allow()
	}

	// Default-deny with most specific explanation
	if hasUnknownRole {
		return Deny(DenialUnknownRole, "unknown or unrecognized role")
	}
	if hasDirectObjectMismatch {
		return Deny(DenialDirectObjectMismatch, "direct object identifier does not match granted scope")
	}
	if hasPrivilegeEscalation {
		return Deny(DenialPrivilegeEscalation, "action exceeds granted role permissions")
	}
	if hasScopeMismatch {
		return Deny(DenialScopeMismatch, "grant scope does not cover target resource scope")
	}

	return Deny(DenialDefaultDeny, "access denied by default policy")
}
