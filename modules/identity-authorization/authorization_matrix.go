// Package localidentity provides identity, authorization, and directory services for OSHE Platform.
//
// PROVISIONAL GOVERNANCE DECLARATION (Issue #90 / V030-I017 / H030-003):
// Under approved Sole Human Owner decision H030-003, this file implements a provisional,
// local-only authorization matrix, least-privilege role/permission catalog, separation-of-duty (SOD)
// conflict rules, source-authority validation, and delegation limits.
//
// Invariant: This baseline is an AI prework candidate. It does NOT bind or select a final
// authority model, role catalog, or protected authority owner. All bindings remain strictly
// provisional, in-memory, and local-only. Zero external identity provider, customer,
// production, runtime, public route, or release effect is claimed or enacted.
package localidentity

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// MaxDelegationDuration defines the maximum allowed duration for any provisional delegation (30 days).
const MaxDelegationDuration = 30 * 24 * time.Hour

// ScopeLevel classifies the organizational hierarchy depth of an authorization scope.
type ScopeLevel string

const (
	ScopeLevelTenant  ScopeLevel = "TENANT"
	ScopeLevelCompany ScopeLevel = "COMPANY"
	ScopeLevelProject ScopeLevel = "PROJECT"
	ScopeLevelSite    ScopeLevel = "SITE"
	ScopeLevelArea    ScopeLevel = "AREA"
)

// ScopeLevelRank returns the numerical hierarchy rank of a scope level (higher = broader scope).
func ScopeLevelRank(level ScopeLevel) int {
	switch level {
	case ScopeLevelTenant:
		return 5
	case ScopeLevelCompany:
		return 4
	case ScopeLevelProject:
		return 3
	case ScopeLevelSite:
		return 2
	case ScopeLevelArea:
		return 1
	default:
		return 0
	}
}

// Permission represents an authoritative, fine-grained action capability on a resource.
type Permission string

const (
	// Organization & Tenancy Permissions (MOD-ORG)
	PermOrgTenantRead     Permission = "org:tenant:read"
	PermOrgTenantManage   Permission = "org:tenant:manage"
	PermOrgCompanyRead    Permission = "org:company:read"
	PermOrgCompanyManage  Permission = "org:company:manage"
	PermOrgProjectRead    Permission = "org:project:read"
	PermOrgProjectManage  Permission = "org:project:manage"
	PermOrgSiteRead       Permission = "org:site:read"
	PermOrgSiteManage     Permission = "org:site:manage"
	PermOrgAreaRead       Permission = "org:area:read"
	PermOrgAreaManage     Permission = "org:area:manage"
	PermOrgContractorLink Permission = "org:contractor:link"

	// Identity & Directory Permissions (MOD-IAM)
	PermIdentityUserRead      Permission = "iam:user:read"
	PermIdentityUserManage    Permission = "iam:user:manage"
	PermIdentityRoleAssign    Permission = "iam:role:assign"
	PermIdentitySessionRevoke Permission = "iam:session:revoke"
	PermDirectoryRead         Permission = "iam:directory:read"
	PermDirectoryManage       Permission = "iam:directory:manage"

	// Inspections & Workflow Permissions (MOD-WFA)
	PermInspectionRead    Permission = "wfa:inspection:read"
	PermInspectionCreate  Permission = "wfa:inspection:create"
	PermInspectionSubmit  Permission = "wfa:inspection:submit"
	PermInspectionReview  Permission = "wfa:inspection:review"
	PermInspectionApprove Permission = "wfa:inspection:approve"
	PermFindingCreate     Permission = "wfa:finding:create"
	PermFindingRemediate  Permission = "wfa:finding:remediate"
	PermFindingVerify     Permission = "wfa:finding:verify"

	// Records & Audit Permissions (MOD-REC)
	PermAuditRead       Permission = "rec:audit:read"
	PermAuditExport     Permission = "rec:audit:export"
	PermRecordArchive   Permission = "rec:record:archive"
	PermLegalHoldManage Permission = "rec:legal_hold:manage"

	// Portal & Reporting Permissions (MOD-REP)
	PermPortalSnapshotStage    Permission = "rep:snapshot:stage"
	PermPortalSnapshotPublish  Permission = "rep:snapshot:publish"
	PermPortalSnapshotWithdraw Permission = "rep:snapshot:withdraw"

	// Delegations Permissions (MOD-IAM)
	PermDelegationGrant  Permission = "iam:delegation:grant"
	PermDelegationRevoke Permission = "iam:delegation:revoke"
)

var (
	// ErrSODConflict indicates a separation-of-duties violation between mutually exclusive roles or scopes.
	ErrSODConflict = errors.New("separation of duties conflict detected")
	// ErrExceedsSourceAuthority indicates that a role grant or delegation exceeds the delegator's authority.
	ErrExceedsSourceAuthority = errors.New("operation exceeds delegator or granter source authority")
	// ErrScopeExceedsSourceAuthority indicates a delegated scope is broader than the delegator's scope.
	ErrScopeExceedsSourceAuthority = errors.New("delegated scope exceeds delegator scope boundary")
	// ErrProtectedAuthorityNonDelegable indicates an attempt to delegate a protected sovereign authority.
	ErrProtectedAuthorityNonDelegable = errors.New("protected sovereign authority cannot be delegated")
	// ErrMultiHopDelegationForbidden indicates an attempt to re-delegate an already delegated authority.
	ErrMultiHopDelegationForbidden = errors.New("multi-hop or re-delegation is strictly prohibited")
	// ErrSelfDelegationForbidden indicates an attempt by a subject to delegate authority to themselves.
	ErrSelfDelegationForbidden = errors.New("self-delegation is strictly prohibited")
	// ErrDelegationDurationExceeded indicates a delegation window exceeds the maximum permitted duration.
	ErrDelegationDurationExceeded = errors.New("delegation window exceeds maximum permitted duration (30 days)")
	// ErrInvalidScopeLevel indicates a role is being assigned at an unauthorized or unsupported scope level.
	ErrInvalidScopeLevel = errors.New("role cannot be granted at the requested scope level")
	// ErrPermissionNotGranted indicates an action is not authorized by the assigned role matrix.
	ErrPermissionNotGranted = errors.New("requested permission is not granted to role in matrix")
	// ErrUnregisteredRole indicates an unrecognized or unconfigured role in the matrix.
	ErrUnregisteredRole = errors.New("role is not registered in authorization matrix")
	// ErrInvalidDelegationWindow indicates malformed or inverted temporal validity timestamps.
	ErrInvalidDelegationWindow = errors.New("delegation valid_to must be strictly after valid_from")
)

// PermissionDefinition details the governance and ownership metadata for a discrete permission.
type PermissionDefinition struct {
	ID          Permission
	OwnerModule string
	MinScope    ScopeLevel
	Description string
	IsProtected bool
}

// RoleDefinition defines the least-privilege permission envelope and scope bounds for a role.
type RoleDefinition struct {
	Role              Role
	AllowedScopes     []ScopeLevel
	Permissions       map[Permission]bool
	Description       string
	IsProtectedRole   bool
	MaxDelegationDays int
}

// ConflictRuleID defines canonical identifiers for separation-of-duty rules.
type ConflictRuleID string

const (
	SODInspectorVsAuditor         ConflictRuleID = "SOD-01-INSPECTOR-AUDITOR"
	SODContractorVsAdmin          ConflictRuleID = "SOD-02-CONTRACTOR-ADMIN"
	SODInspectionSubmitVsApprove  ConflictRuleID = "SOD-03-SUBMIT-APPROVE"
	SODFindingRemediateVsVerify   ConflictRuleID = "SOD-04-REMEDIATE-VERIFY"
	SODDelegationSelfApproval     ConflictRuleID = "SOD-05-DELEGATION-SELF-GRANT"
)

// ConflictRule codifies a separation-of-duties invariant between two roles.
type ConflictRule struct {
	ID               ConflictRuleID
	Name             string
	Description      string
	ConflictingRoles [2]Role
	ScopeSensitive   bool // true if conflict applies when scopes overlap
}

// DelegationRequest defines the parameters of a proposed authority delegation.
type DelegationRequest struct {
	DelegatorSubject string
	DelegatorRole    Role
	DelegatorScope   ScopeGrant
	DelegateeSubject string
	DelegatedRole    Role
	DelegatedScope   ScopeGrant
	ValidFrom        time.Time
	ValidTo          time.Time
	IsSubDelegation  bool
}

// AuthorizationMatrix provides authoritative role/permission definitions, conflict evaluation,
// and delegation bounding for provisional local authorization.
type AuthorizationMatrix struct {
	roles       map[Role]RoleDefinition
	permissions map[Permission]PermissionDefinition
	conflicts   []ConflictRule
}

// NewProvisionalAuthorizationMatrix constructs and initializes the provisional v0.3.0 authorization matrix.
func NewProvisionalAuthorizationMatrix() AuthorizationMatrix {
	m := AuthorizationMatrix{
		roles:       make(map[Role]RoleDefinition),
		permissions: make(map[Permission]PermissionDefinition),
		conflicts:   make([]ConflictRule, 0),
	}

	m.registerPermissions()
	m.registerRoles()
	m.registerConflictRules()

	return m
}

func (m *AuthorizationMatrix) registerPermissions() {
	perms := []PermissionDefinition{
		// MOD-ORG
		{ID: PermOrgTenantRead, OwnerModule: "MOD-ORG", MinScope: ScopeLevelTenant, Description: "Read tenant metadata", IsProtected: false},
		{ID: PermOrgTenantManage, OwnerModule: "MOD-ORG", MinScope: ScopeLevelTenant, Description: "Configure tenant and root policies", IsProtected: true},
		{ID: PermOrgCompanyRead, OwnerModule: "MOD-ORG", MinScope: ScopeLevelCompany, Description: "Read company hierarchy", IsProtected: false},
		{ID: PermOrgCompanyManage, OwnerModule: "MOD-ORG", MinScope: ScopeLevelCompany, Description: "Manage company settings", IsProtected: false},
		{ID: PermOrgProjectRead, OwnerModule: "MOD-ORG", MinScope: ScopeLevelProject, Description: "Read project information", IsProtected: false},
		{ID: PermOrgProjectManage, OwnerModule: "MOD-ORG", MinScope: ScopeLevelProject, Description: "Manage project boundaries", IsProtected: false},
		{ID: PermOrgSiteRead, OwnerModule: "MOD-ORG", MinScope: ScopeLevelSite, Description: "Read site configurations", IsProtected: false},
		{ID: PermOrgSiteManage, OwnerModule: "MOD-ORG", MinScope: ScopeLevelSite, Description: "Manage site entities", IsProtected: false},
		{ID: PermOrgAreaRead, OwnerModule: "MOD-ORG", MinScope: ScopeLevelArea, Description: "Read area designations", IsProtected: false},
		{ID: PermOrgAreaManage, OwnerModule: "MOD-ORG", MinScope: ScopeLevelArea, Description: "Manage area zones", IsProtected: false},
		{ID: PermOrgContractorLink, OwnerModule: "MOD-ORG", MinScope: ScopeLevelProject, Description: "Sponsor and link external contractors", IsProtected: false},

		// MOD-IAM
		{ID: PermIdentityUserRead, OwnerModule: "MOD-IAM", MinScope: ScopeLevelTenant, Description: "Read user accounts", IsProtected: false},
		{ID: PermIdentityUserManage, OwnerModule: "MOD-IAM", MinScope: ScopeLevelTenant, Description: "Manage user lifecycles", IsProtected: true},
		{ID: PermIdentityRoleAssign, OwnerModule: "MOD-IAM", MinScope: ScopeLevelTenant, Description: "Assign roles and scope grants", IsProtected: true},
		{ID: PermIdentitySessionRevoke, OwnerModule: "MOD-IAM", MinScope: ScopeLevelTenant, Description: "Revoke active session tokens", IsProtected: true},
		{ID: PermDirectoryRead, OwnerModule: "MOD-IAM", MinScope: ScopeLevelProject, Description: "Query scoped directory profiles", IsProtected: false},
		{ID: PermDirectoryManage, OwnerModule: "MOD-IAM", MinScope: ScopeLevelCompany, Description: "Manage directory profile entries", IsProtected: false},

		// MOD-WFA
		{ID: PermInspectionRead, OwnerModule: "MOD-WFA", MinScope: ScopeLevelProject, Description: "Read inspection forms and logs", IsProtected: false},
		{ID: PermInspectionCreate, OwnerModule: "MOD-WFA", MinScope: ScopeLevelProject, Description: "Create and execute inspections", IsProtected: false},
		{ID: PermInspectionSubmit, OwnerModule: "MOD-WFA", MinScope: ScopeLevelProject, Description: "Submit completed inspections for review", IsProtected: false},
		{ID: PermInspectionReview, OwnerModule: "MOD-WFA", MinScope: ScopeLevelProject, Description: "Review inspection submissions", IsProtected: false},
		{ID: PermInspectionApprove, OwnerModule: "MOD-WFA", MinScope: ScopeLevelProject, Description: "Formally approve inspection results", IsProtected: false},
		{ID: PermFindingCreate, OwnerModule: "MOD-WFA", MinScope: ScopeLevelSite, Description: "Log safety findings", IsProtected: false},
		{ID: PermFindingRemediate, OwnerModule: "MOD-WFA", MinScope: ScopeLevelSite, Description: "Submit remediation actions for findings", IsProtected: false},
		{ID: PermFindingVerify, OwnerModule: "MOD-WFA", MinScope: ScopeLevelSite, Description: "Verify and close remediated findings", IsProtected: false},

		// MOD-REC
		{ID: PermAuditRead, OwnerModule: "MOD-REC", MinScope: ScopeLevelCompany, Description: "Read append-only audit journals", IsProtected: false},
		{ID: PermAuditExport, OwnerModule: "MOD-REC", MinScope: ScopeLevelTenant, Description: "Export compliance audit packages", IsProtected: true},
		{ID: PermRecordArchive, OwnerModule: "MOD-REC", MinScope: ScopeLevelProject, Description: "Transition records to archived state", IsProtected: false},
		{ID: PermLegalHoldManage, OwnerModule: "MOD-REC", MinScope: ScopeLevelTenant, Description: "Manage legal hold freezes", IsProtected: true},

		// MOD-REP
		{ID: PermPortalSnapshotStage, OwnerModule: "MOD-REP", MinScope: ScopeLevelProject, Description: "Stage static portal snapshots", IsProtected: false},
		{ID: PermPortalSnapshotPublish, OwnerModule: "MOD-REP", MinScope: ScopeLevelTenant, Description: "Approve and publish portal snapshots", IsProtected: true},
		{ID: PermPortalSnapshotWithdraw, OwnerModule: "MOD-REP", MinScope: ScopeLevelTenant, Description: "Withdraw published portal snapshots", IsProtected: true},

		// MOD-IAM Delegations
		{ID: PermDelegationGrant, OwnerModule: "MOD-IAM", MinScope: ScopeLevelProject, Description: "Grant time-bounded role delegation", IsProtected: true},
		{ID: PermDelegationRevoke, OwnerModule: "MOD-IAM", MinScope: ScopeLevelProject, Description: "Revoke active delegation grants", IsProtected: false},
	}

	for _, p := range perms {
		m.permissions[p.ID] = p
	}
}

func (m *AuthorizationMatrix) registerRoles() {
	// RoleTenantAdmin: Tenant sovereign authority; non-delegable
	m.roles[RoleTenantAdmin] = RoleDefinition{
		Role:          RoleTenantAdmin,
		AllowedScopes: []ScopeLevel{ScopeLevelTenant},
		Permissions: map[Permission]bool{
			PermOrgTenantRead: true, PermOrgTenantManage: true, PermOrgCompanyRead: true, PermOrgCompanyManage: true,
			PermOrgProjectRead: true, PermOrgProjectManage: true, PermOrgSiteRead: true, PermOrgSiteManage: true,
			PermOrgAreaRead: true, PermOrgAreaManage: true, PermOrgContractorLink: true,
			PermIdentityUserRead: true, PermIdentityUserManage: true, PermIdentityRoleAssign: true, PermIdentitySessionRevoke: true,
			PermDirectoryRead: true, PermDirectoryManage: true,
			PermInspectionRead: true, PermInspectionCreate: true, PermInspectionSubmit: true, PermInspectionReview: true, PermInspectionApprove: true,
			PermFindingCreate: true, PermFindingRemediate: true, PermFindingVerify: true,
			PermAuditRead: true, PermAuditExport: true, PermRecordArchive: true, PermLegalHoldManage: true,
			PermPortalSnapshotStage: true, PermPortalSnapshotPublish: true, PermPortalSnapshotWithdraw: true,
			PermDelegationGrant: true, PermDelegationRevoke: true,
		},
		Description:       "Authoritative tenant administrator with sovereign configuration rights",
		IsProtectedRole:   true,
		MaxDelegationDays: 0, // Cannot be delegated
	}

	// RoleProjectManager: Operational leadership over project and sites
	m.roles[RoleProjectManager] = RoleDefinition{
		Role:          RoleProjectManager,
		AllowedScopes: []ScopeLevel{ScopeLevelProject, ScopeLevelSite},
		Permissions: map[Permission]bool{
			PermOrgProjectRead: true, PermOrgProjectManage: true, PermOrgSiteRead: true, PermOrgSiteManage: true,
			PermOrgAreaRead: true, PermOrgAreaManage: true, PermOrgContractorLink: true,
			PermDirectoryRead: true,
			PermInspectionRead: true, PermInspectionCreate: true, PermInspectionSubmit: true, PermInspectionReview: true, PermInspectionApprove: true,
			PermFindingCreate: true, PermFindingRemediate: true, PermFindingVerify: true,
			PermRecordArchive: true,
			PermPortalSnapshotStage: true,
			PermDelegationGrant: true, PermDelegationRevoke: true,
		},
		Description:       "Operational project manager governing inspections and site teams",
		IsProtectedRole:   false,
		MaxDelegationDays: 14,
	}

	// RoleInspector: Field evaluation and inspection execution
	m.roles[RoleInspector] = RoleDefinition{
		Role:          RoleInspector,
		AllowedScopes: []ScopeLevel{ScopeLevelProject, ScopeLevelSite, ScopeLevelArea},
		Permissions: map[Permission]bool{
			PermOrgProjectRead: true, PermOrgSiteRead: true, PermOrgAreaRead: true,
			PermDirectoryRead:    true,
			PermInspectionRead:   true, PermInspectionCreate: true, PermInspectionSubmit: true,
			PermFindingCreate:    true, PermFindingRemediate: true,
		},
		Description:       "Field safety inspector logging observations and findings",
		IsProtectedRole:   false,
		MaxDelegationDays: 7,
	}

	// RoleAuditor: Independent compliance oversight; strictly read and export
	m.roles[RoleAuditor] = RoleDefinition{
		Role:          RoleAuditor,
		AllowedScopes: []ScopeLevel{ScopeLevelTenant, ScopeLevelCompany, ScopeLevelProject, ScopeLevelSite},
		Permissions: map[Permission]bool{
			PermOrgTenantRead: true, PermOrgCompanyRead: true, PermOrgProjectRead: true, PermOrgSiteRead: true, PermOrgAreaRead: true,
			PermDirectoryRead:  true,
			PermInspectionRead: true,
			PermAuditRead:      true, PermAuditExport: true,
		},
		Description:       "Independent oversight auditor with zero operational create/update rights",
		IsProtectedRole:   false,
		MaxDelegationDays: 14,
	}

	// RoleViewer: Passive read-only observer
	m.roles[RoleViewer] = RoleDefinition{
		Role:          RoleViewer,
		AllowedScopes: []ScopeLevel{ScopeLevelTenant, ScopeLevelCompany, ScopeLevelProject, ScopeLevelSite, ScopeLevelArea},
		Permissions: map[Permission]bool{
			PermOrgProjectRead: true, PermOrgSiteRead: true, PermOrgAreaRead: true,
			PermDirectoryRead:  true,
			PermInspectionRead: true,
		},
		Description:       "Passive read-only participant for status and dashboard reviews",
		IsProtectedRole:   false,
		MaxDelegationDays: 30,
	}

	// RoleContractor: External bounded partner; submit inspection responses only
	m.roles[RoleContractor] = RoleDefinition{
		Role:          RoleContractor,
		AllowedScopes: []ScopeLevel{ScopeLevelProject, ScopeLevelSite, ScopeLevelArea},
		Permissions: map[Permission]bool{
			PermOrgSiteRead:      true, PermOrgAreaRead: true,
			PermDirectoryRead:    true,
			PermInspectionRead:   true, PermInspectionCreate: true,
			PermFindingRemediate: true,
		},
		Description:       "External contractor bounded strictly to assigned project and site tasks",
		IsProtectedRole:   false,
		MaxDelegationDays: 7,
	}

	// RoleSupport: Internal technical support; diagnostics only
	m.roles[RoleSupport] = RoleDefinition{
		Role:          RoleSupport,
		AllowedScopes: []ScopeLevel{ScopeLevelProject, ScopeLevelSite},
		Permissions: map[Permission]bool{
			PermOrgProjectRead: true, PermOrgSiteRead: true,
			PermDirectoryRead:  true,
			PermInspectionRead: true,
		},
		Description:       "Technical support role for diagnostic review",
		IsProtectedRole:   false,
		MaxDelegationDays: 3,
	}
}

func (m *AuthorizationMatrix) registerConflictRules() {
	m.conflicts = []ConflictRule{
		{
			ID:               SODInspectorVsAuditor,
			Name:             "Inspector vs. Auditor Separation",
			Description:      "An Inspector cannot concurrently hold Auditor privileges within overlapping project/company scopes",
			ConflictingRoles: [2]Role{RoleInspector, RoleAuditor},
			ScopeSensitive:   true,
		},
		{
			ID:               SODContractorVsAdmin,
			Name:             "Contractor vs. Tenant Admin Separation",
			Description:      "An external contractor cannot hold administrative or project management authority",
			ConflictingRoles: [2]Role{RoleContractor, RoleTenantAdmin},
			ScopeSensitive:   false,
		},
		{
			ID:               ConflictRuleID("SOD-02B-CONTRACTOR-PM"),
			Name:             "Contractor vs. Project Manager Separation",
			Description:      "An external contractor cannot concurrently hold Project Manager authority",
			ConflictingRoles: [2]Role{RoleContractor, RoleProjectManager},
			ScopeSensitive:   true,
		},
		{
			ID:               SODInspectionSubmitVsApprove,
			Name:             "Inspection Submitter vs. Approver Separation",
			Description:      "Inspection creation and formal approval require separate role functions",
			ConflictingRoles: [2]Role{RoleContractor, RoleProjectManager},
			ScopeSensitive:   true,
		},
		{
			ID:               SODDelegationSelfApproval,
			Name:             "Self-Delegation Prohibition",
			Description:      "A delegator cannot delegate permissions to themselves",
			ConflictingRoles: [2]Role{},
			ScopeSensitive:   false,
		},
	}
}

// GetRoleDefinition returns the definition and permission set for a given role.
func (m *AuthorizationMatrix) GetRoleDefinition(r Role) (RoleDefinition, bool) {
	def, exists := m.roles[r]
	return def, exists
}

// GetPermissionDefinition returns metadata for a permission.
func (m *AuthorizationMatrix) GetPermissionDefinition(p Permission) (PermissionDefinition, bool) {
	def, exists := m.permissions[p]
	return def, exists
}

// RoleHasPermission checks if a role contains a specific permission.
func (m *AuthorizationMatrix) RoleHasPermission(r Role, p Permission) bool {
	def, exists := m.roles[r]
	if !exists {
		return false
	}
	return def.Permissions[p]
}

// ProtectedPermissions returns all permissions classified as sovereign/protected.
func (m *AuthorizationMatrix) ProtectedPermissions() []PermissionDefinition {
	var out []PermissionDefinition
	for _, p := range m.permissions {
		if p.IsProtected {
			out = append(out, p)
		}
	}
	return out
}

// ConflictRules returns all active separation-of-duty rules.
func (m *AuthorizationMatrix) ConflictRules() []ConflictRule {
	return m.conflicts
}

// ResolveScopeLevel determines the most specific ScopeLevel described by a ScopeGrant.
func ResolveScopeLevel(grant ScopeGrant) ScopeLevel {
	if grant.AreaID != "" {
		return ScopeLevelArea
	}
	if grant.SiteID != "" {
		return ScopeLevelSite
	}
	if grant.ProjectID != "" {
		return ScopeLevelProject
	}
	if grant.CompanyID != "" {
		return ScopeLevelCompany
	}
	return ScopeLevelTenant
}

// CheckScopesOverlap determines if two scope grants overlap in hierarchy.
func CheckScopesOverlap(a, b ScopeGrant) bool {
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
	return true
}

// ScopeContains verifies that outer scope strictly contains or equals inner scope.
func ScopeContains(outer, inner ScopeGrant) bool {
	if outer.TenantID != "" && outer.TenantID != inner.TenantID {
		return false
	}
	if outer.CompanyID != "" && outer.CompanyID != inner.CompanyID {
		return false
	}
	if outer.ProjectID != "" && outer.ProjectID != inner.ProjectID {
		return false
	}
	if outer.SiteID != "" && outer.SiteID != inner.SiteID {
		return false
	}
	if outer.AreaID != "" && outer.AreaID != inner.AreaID {
		return false
	}
	return true
}

// ValidateRoleAssignment validates a proposed role assignment against scope rules and SOD conflicts.
func (m *AuthorizationMatrix) ValidateRoleAssignment(existing []RoleAssignment, assignment RoleAssignment) error {
	roleDef, exists := m.roles[assignment.Role]
	if !exists {
		return fmt.Errorf("%w: %s", ErrUnregisteredRole, assignment.Role)
	}

	scopeLvl := ResolveScopeLevel(assignment.Scope)
	var scopeAllowed bool
	for _, allowed := range roleDef.AllowedScopes {
		if allowed == scopeLvl {
			scopeAllowed = true
			break
		}
	}
	if !scopeAllowed {
		return fmt.Errorf("%w: role %s not allowed at scope level %s", ErrInvalidScopeLevel, assignment.Role, scopeLvl)
	}

	// SOD conflict evaluation
	for _, r := range m.conflicts {
		for _, ex := range existing {
			if ex.Subject != assignment.Subject {
				continue
			}

			// Check conflicting pair
			isPair := (ex.Role == r.ConflictingRoles[0] && assignment.Role == r.ConflictingRoles[1]) ||
				(ex.Role == r.ConflictingRoles[1] && assignment.Role == r.ConflictingRoles[0])

			if isPair {
				if !r.ScopeSensitive || CheckScopesOverlap(ex.Scope, assignment.Scope) {
					return fmt.Errorf("%w: rule %s (%s vs %s)", ErrSODConflict, r.ID, r.ConflictingRoles[0], r.ConflictingRoles[1])
				}
			}
		}
	}

	return nil
}

// ValidateDelegationRequest rigorously asserts all source-authority, non-elevation, and duration invariants.
func (m *AuthorizationMatrix) ValidateDelegationRequest(req DelegationRequest) error {
	// 1. Multi-hop re-delegation prohibition
	if req.IsSubDelegation {
		return ErrMultiHopDelegationForbidden
	}

	// 2. Self-delegation prohibition
	if strings.TrimSpace(req.DelegatorSubject) == strings.TrimSpace(req.DelegateeSubject) {
		return ErrSelfDelegationForbidden
	}

	// 3. Temporal validity checks
	if req.ValidTo.Before(req.ValidFrom) || req.ValidTo.Equal(req.ValidFrom) {
		return ErrInvalidDelegationWindow
	}
	duration := req.ValidTo.Sub(req.ValidFrom)
	if duration > MaxDelegationDuration {
		return fmt.Errorf("%w: duration %s exceeds %s", ErrDelegationDurationExceeded, duration, MaxDelegationDuration)
	}

	// 4. Role registration and protected authority checks
	delegatorDef, exists := m.roles[req.DelegatorRole]
	if !exists {
		return fmt.Errorf("%w: delegator role %s", ErrUnregisteredRole, req.DelegatorRole)
	}
	if delegatorDef.IsProtectedRole {
		return fmt.Errorf("%w: delegator role %s is protected and cannot be delegated", ErrProtectedAuthorityNonDelegable, req.DelegatorRole)
	}

	delegatedDef, exists := m.roles[req.DelegatedRole]
	if !exists {
		return fmt.Errorf("%w: delegated role %s", ErrUnregisteredRole, req.DelegatedRole)
	}
	if delegatedDef.IsProtectedRole {
		return fmt.Errorf("%w: delegated role %s is protected", ErrProtectedAuthorityNonDelegable, req.DelegatedRole)
	}

	// Check max delegation days configured on role
	if delegatorDef.MaxDelegationDays > 0 {
		maxRoleDuration := time.Duration(delegatorDef.MaxDelegationDays) * 24 * time.Hour
		if duration > maxRoleDuration {
			return fmt.Errorf("%w: role %s limits delegation to %d days", ErrDelegationDurationExceeded, req.DelegatorRole, delegatorDef.MaxDelegationDays)
		}
	}

	// 5. Source Authority Containment: Delegator must possess all permissions granted to delegatee
	for perm := range delegatedDef.Permissions {
		if !delegatorDef.Permissions[perm] {
			return fmt.Errorf("%w: delegator lacks permission %s required by delegated role %s", ErrExceedsSourceAuthority, perm, req.DelegatedRole)
		}
	}

	// 6. Scope Containment: Delegator scope must strictly contain or equal delegated scope
	if !ScopeContains(req.DelegatorScope, req.DelegatedScope) {
		return fmt.Errorf("%w: delegated scope exceeds delegator scope", ErrScopeExceedsSourceAuthority)
	}

	return nil
}
