package localidentity_test

import (
	"errors"
	"testing"
	"time"

	localidentity "github.com/oshethai/oshe-platform/modules/identity-authorization"
)

func TestAuthorizationMatrix_RoleDefinitionsAndHierarchy(t *testing.T) {
	matrix := localidentity.NewProvisionalAuthorizationMatrix()

	expectedRoles := []localidentity.Role{
		localidentity.RoleTenantAdmin,
		localidentity.RoleProjectManager,
		localidentity.RoleInspector,
		localidentity.RoleAuditor,
		localidentity.RoleViewer,
		localidentity.RoleContractor,
		localidentity.RoleSupport,
	}

	for _, r := range expectedRoles {
		def, exists := matrix.GetRoleDefinition(r)
		if !exists {
			t.Fatalf("expected role %s to be registered in matrix", r)
		}
		if def.Role != r {
			t.Errorf("expected role %s, got %s", r, def.Role)
		}
		if len(def.AllowedScopes) == 0 {
			t.Errorf("role %s must define at least one allowed scope", r)
		}
		if len(def.Permissions) == 0 {
			t.Errorf("role %s must define at least one permission", r)
		}
	}

	// Verify TenantAdmin is protected and non-delegable
	adminDef, _ := matrix.GetRoleDefinition(localidentity.RoleTenantAdmin)
	if !adminDef.IsProtectedRole {
		t.Errorf("RoleTenantAdmin must be marked as protected role")
	}
	if adminDef.MaxDelegationDays != 0 {
		t.Errorf("RoleTenantAdmin must have MaxDelegationDays = 0, got %d", adminDef.MaxDelegationDays)
	}

	// Verify non-protected roles have MaxDelegationDays > 0 and IsProtectedRole == false
	pmDef, _ := matrix.GetRoleDefinition(localidentity.RoleProjectManager)
	if pmDef.IsProtectedRole {
		t.Errorf("RoleProjectManager must not be a protected role")
	}
	if pmDef.MaxDelegationDays <= 0 || pmDef.MaxDelegationDays > 30 {
		t.Errorf("RoleProjectManager MaxDelegationDays must be bounded between 1 and 30, got %d", pmDef.MaxDelegationDays)
	}

	// Verify Hierarchy Ranks
	if localidentity.ScopeLevelRank(localidentity.ScopeLevelTenant) <= localidentity.ScopeLevelRank(localidentity.ScopeLevelCompany) {
		t.Errorf("Tenant rank must be higher than Company rank")
	}
	if localidentity.ScopeLevelRank(localidentity.ScopeLevelCompany) <= localidentity.ScopeLevelRank(localidentity.ScopeLevelProject) {
		t.Errorf("Company rank must be higher than Project rank")
	}
	if localidentity.ScopeLevelRank(localidentity.ScopeLevelProject) <= localidentity.ScopeLevelRank(localidentity.ScopeLevelSite) {
		t.Errorf("Project rank must be higher than Site rank")
	}
	if localidentity.ScopeLevelRank(localidentity.ScopeLevelSite) <= localidentity.ScopeLevelRank(localidentity.ScopeLevelArea) {
		t.Errorf("Site rank must be higher than Area rank")
	}
}

func TestAuthorizationMatrix_ScopeResolution(t *testing.T) {
	cases := []struct {
		name     string
		grant    localidentity.ScopeGrant
		expected localidentity.ScopeLevel
	}{
		{
			name:     "tenant_level",
			grant:    localidentity.ScopeGrant{TenantID: "ten_01"},
			expected: localidentity.ScopeLevelTenant,
		},
		{
			name:     "company_level",
			grant:    localidentity.ScopeGrant{TenantID: "ten_01", CompanyID: "comp_01"},
			expected: localidentity.ScopeLevelCompany,
		},
		{
			name:     "project_level",
			grant:    localidentity.ScopeGrant{TenantID: "ten_01", CompanyID: "comp_01", ProjectID: "proj_01"},
			expected: localidentity.ScopeLevelProject,
		},
		{
			name:     "site_level",
			grant:    localidentity.ScopeGrant{TenantID: "ten_01", CompanyID: "comp_01", ProjectID: "proj_01", SiteID: "site_01"},
			expected: localidentity.ScopeLevelSite,
		},
		{
			name:     "area_level",
			grant:    localidentity.ScopeGrant{TenantID: "ten_01", CompanyID: "comp_01", ProjectID: "proj_01", SiteID: "site_01", AreaID: "area_01"},
			expected: localidentity.ScopeLevelArea,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res := localidentity.ResolveScopeLevel(tc.grant)
			if res != tc.expected {
				t.Errorf("expected scope %s, got %s", tc.expected, res)
			}
		})
	}
}

func TestAuthorizationMatrix_ProtectedPermissionsCatalog(t *testing.T) {
	matrix := localidentity.NewProvisionalAuthorizationMatrix()

	protected := matrix.ProtectedPermissions()
	if len(protected) == 0 {
		t.Fatalf("expected protected permissions to be registered")
	}

	protectedMap := make(map[localidentity.Permission]bool)
	for _, p := range protected {
		protectedMap[p.ID] = true
		if !p.IsProtected {
			t.Errorf("permission %s in ProtectedPermissions() has IsProtected=false", p.ID)
		}
	}

	mustBeProtected := []localidentity.Permission{
		localidentity.PermOrgTenantManage,
		localidentity.PermIdentityUserManage,
		localidentity.PermIdentityRoleAssign,
		localidentity.PermIdentitySessionRevoke,
		localidentity.PermAuditExport,
		localidentity.PermLegalHoldManage,
		localidentity.PermPortalSnapshotPublish,
		localidentity.PermPortalSnapshotWithdraw,
		localidentity.PermDelegationGrant,
	}

	for _, perm := range mustBeProtected {
		if !protectedMap[perm] {
			t.Errorf("permission %s must be classified as protected", perm)
		}
	}

	// Ensure normal operational permissions are not protected
	operationalPerms := []localidentity.Permission{
		localidentity.PermInspectionRead,
		localidentity.PermInspectionCreate,
		localidentity.PermInspectionSubmit,
		localidentity.PermFindingCreate,
		localidentity.PermDirectoryRead,
	}

	for _, perm := range operationalPerms {
		if protectedMap[perm] {
			t.Errorf("operational permission %s must NOT be protected", perm)
		}
	}
}

func TestAuthorizationMatrix_LeastPrivilegeRoleProfiles(t *testing.T) {
	matrix := localidentity.NewProvisionalAuthorizationMatrix()

	// 1. Auditor must have zero operational execution/approval permissions
	auditorDisallowed := []localidentity.Permission{
		localidentity.PermInspectionCreate,
		localidentity.PermInspectionSubmit,
		localidentity.PermInspectionReview,
		localidentity.PermInspectionApprove,
		localidentity.PermFindingCreate,
		localidentity.PermFindingRemediate,
		localidentity.PermFindingVerify,
		localidentity.PermOrgProjectManage,
	}
	for _, p := range auditorDisallowed {
		if matrix.RoleHasPermission(localidentity.RoleAuditor, p) {
			t.Errorf("Auditor must NOT have permission %s", p)
		}
	}
	// Auditor must have read and audit export
	if !matrix.RoleHasPermission(localidentity.RoleAuditor, localidentity.PermAuditRead) {
		t.Errorf("Auditor must have PermAuditRead")
	}
	if !matrix.RoleHasPermission(localidentity.RoleAuditor, localidentity.PermAuditExport) {
		t.Errorf("Auditor must have PermAuditExport")
	}

	// 2. Inspector must not have approval rights
	if matrix.RoleHasPermission(localidentity.RoleInspector, localidentity.PermInspectionApprove) {
		t.Errorf("Inspector must NOT have PermInspectionApprove")
	}
	if matrix.RoleHasPermission(localidentity.RoleInspector, localidentity.PermOrgTenantManage) {
		t.Errorf("Inspector must NOT have PermOrgTenantManage")
	}

	// 3. Contractor must have strictly bounded permissions
	contractorDisallowed := []localidentity.Permission{
		localidentity.PermOrgTenantManage,
		localidentity.PermOrgCompanyManage,
		localidentity.PermOrgProjectManage,
		localidentity.PermIdentityRoleAssign,
		localidentity.PermInspectionApprove,
		localidentity.PermAuditExport,
		localidentity.PermLegalHoldManage,
	}
	for _, p := range contractorDisallowed {
		if matrix.RoleHasPermission(localidentity.RoleContractor, p) {
			t.Errorf("Contractor must NOT have permission %s", p)
		}
	}

	// 4. ProjectManager has workflow approval but not sovereign tenant manage
	if !matrix.RoleHasPermission(localidentity.RoleProjectManager, localidentity.PermInspectionApprove) {
		t.Errorf("ProjectManager must have PermInspectionApprove")
	}
	if matrix.RoleHasPermission(localidentity.RoleProjectManager, localidentity.PermOrgTenantManage) {
		t.Errorf("ProjectManager must NOT have PermOrgTenantManage")
	}
	if matrix.RoleHasPermission(localidentity.RoleProjectManager, localidentity.PermIdentityRoleAssign) {
		t.Errorf("ProjectManager must NOT have PermIdentityRoleAssign")
	}
}

func TestAuthorizationMatrix_ValidateRoleAssignment_ScopeBounds(t *testing.T) {
	matrix := localidentity.NewProvisionalAuthorizationMatrix()

	// Valid: ProjectManager at Project scope
	err := matrix.ValidateRoleAssignment(nil, localidentity.RoleAssignment{
		Subject: "usr_pm_1",
		Role:    localidentity.RoleProjectManager,
		Scope:   localidentity.ScopeGrant{TenantID: "ten_1", CompanyID: "comp_1", ProjectID: "proj_1"},
	})
	if err != nil {
		t.Fatalf("expected valid PM assignment at project scope, got: %v", err)
	}

	// Invalid: ProjectManager at Tenant scope (too broad)
	err = matrix.ValidateRoleAssignment(nil, localidentity.RoleAssignment{
		Subject: "usr_pm_1",
		Role:    localidentity.RoleProjectManager,
		Scope:   localidentity.ScopeGrant{TenantID: "ten_1"},
	})
	if err == nil {
		t.Fatalf("expected error assigning PM at tenant scope, got nil")
	}
	if !errors.Is(err, localidentity.ErrInvalidScopeLevel) {
		t.Errorf("expected ErrInvalidScopeLevel, got: %v", err)
	}

	// Invalid: TenantAdmin at Site scope (must be Tenant scope)
	err = matrix.ValidateRoleAssignment(nil, localidentity.RoleAssignment{
		Subject: "usr_admin_1",
		Role:    localidentity.RoleTenantAdmin,
		Scope:   localidentity.ScopeGrant{TenantID: "ten_1", CompanyID: "comp_1", ProjectID: "proj_1", SiteID: "site_1"},
	})
	if err == nil {
		t.Fatalf("expected error assigning TenantAdmin at site scope, got nil")
	}
	if !errors.Is(err, localidentity.ErrInvalidScopeLevel) {
		t.Errorf("expected ErrInvalidScopeLevel, got: %v", err)
	}

	// Invalid: Unregistered role
	err = matrix.ValidateRoleAssignment(nil, localidentity.RoleAssignment{
		Subject: "usr_unknown",
		Role:    localidentity.Role("UNKNOWN_ROLE"),
		Scope:   localidentity.ScopeGrant{TenantID: "ten_1"},
	})
	if err == nil {
		t.Fatalf("expected error for unregistered role, got nil")
	}
	if !errors.Is(err, localidentity.ErrUnregisteredRole) {
		t.Errorf("expected ErrUnregisteredRole, got: %v", err)
	}
}

func TestAuthorizationMatrix_SODConflicts(t *testing.T) {
	matrix := localidentity.NewProvisionalAuthorizationMatrix()

	t.Run("SOD-01: Inspector vs Auditor on overlapping project", func(t *testing.T) {
		existing := []localidentity.RoleAssignment{
			{
				Subject: "usr_worker_1",
				Role:    localidentity.RoleInspector,
				Scope:   localidentity.ScopeGrant{TenantID: "ten_1", ProjectID: "proj_alpha"},
			},
		}

		// Conflict: assigning Auditor to same user on overlapping project
		err := matrix.ValidateRoleAssignment(existing, localidentity.RoleAssignment{
			Subject: "usr_worker_1",
			Role:    localidentity.RoleAuditor,
			Scope:   localidentity.ScopeGrant{TenantID: "ten_1", ProjectID: "proj_alpha"},
		})
		if err == nil {
			t.Fatalf("expected SOD conflict for Inspector vs Auditor on same project, got nil")
		}
		if !errors.Is(err, localidentity.ErrSODConflict) {
			t.Errorf("expected ErrSODConflict, got: %v", err)
		}

		// Non-conflicting: Auditor on a different, non-overlapping project
		err = matrix.ValidateRoleAssignment(existing, localidentity.RoleAssignment{
			Subject: "usr_worker_1",
			Role:    localidentity.RoleAuditor,
			Scope:   localidentity.ScopeGrant{TenantID: "ten_1", ProjectID: "proj_beta"},
		})
		if err != nil {
			t.Errorf("expected non-overlapping project assignment to succeed, got: %v", err)
		}

		// Non-conflicting: Different user
		err = matrix.ValidateRoleAssignment(existing, localidentity.RoleAssignment{
			Subject: "usr_other_user",
			Role:    localidentity.RoleAuditor,
			Scope:   localidentity.ScopeGrant{TenantID: "ten_1", ProjectID: "proj_alpha"},
		})
		if err != nil {
			t.Errorf("expected different user assignment to succeed, got: %v", err)
		}
	})

	t.Run("SOD-02: Contractor vs TenantAdmin (scope-insensitive)", func(t *testing.T) {
		existing := []localidentity.RoleAssignment{
			{
				Subject: "usr_ext_1",
				Role:    localidentity.RoleContractor,
				Scope:   localidentity.ScopeGrant{TenantID: "ten_1", ProjectID: "proj_1"},
			},
		}

		// Attempt to grant TenantAdmin to contractor
		err := matrix.ValidateRoleAssignment(existing, localidentity.RoleAssignment{
			Subject: "usr_ext_1",
			Role:    localidentity.RoleTenantAdmin,
			Scope:   localidentity.ScopeGrant{TenantID: "ten_1"},
		})
		if err == nil {
			t.Fatalf("expected SOD conflict for Contractor vs TenantAdmin, got nil")
		}
		if !errors.Is(err, localidentity.ErrSODConflict) {
			t.Errorf("expected ErrSODConflict, got: %v", err)
		}
	})

	t.Run("SOD-02B: Contractor vs ProjectManager on overlapping project", func(t *testing.T) {
		existing := []localidentity.RoleAssignment{
			{
				Subject: "usr_ext_2",
				Role:    localidentity.RoleContractor,
				Scope:   localidentity.ScopeGrant{TenantID: "ten_1", ProjectID: "proj_1"},
			},
		}

		err := matrix.ValidateRoleAssignment(existing, localidentity.RoleAssignment{
			Subject: "usr_ext_2",
			Role:    localidentity.RoleProjectManager,
			Scope:   localidentity.ScopeGrant{TenantID: "ten_1", ProjectID: "proj_1"},
		})
		if err == nil {
			t.Fatalf("expected SOD conflict for Contractor vs PM on same project, got nil")
		}
		if !errors.Is(err, localidentity.ErrSODConflict) {
			t.Errorf("expected ErrSODConflict, got: %v", err)
		}
	})
}

func TestAuthorizationMatrix_DelegationInvariants(t *testing.T) {
	matrix := localidentity.NewProvisionalAuthorizationMatrix()
	now := time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC)

	baseReq := localidentity.DelegationRequest{
		DelegatorSubject: "usr_pm_1",
		DelegatorRole:    localidentity.RoleProjectManager,
		DelegatorScope:   localidentity.ScopeGrant{TenantID: "ten_1", ProjectID: "proj_1"},
		DelegateeSubject: "usr_delegatee_1",
		DelegatedRole:    localidentity.RoleProjectManager,
		DelegatedScope:   localidentity.ScopeGrant{TenantID: "ten_1", ProjectID: "proj_1"},
		ValidFrom:        now,
		ValidTo:          now.Add(7 * 24 * time.Hour), // 7 days (within 14 days PM limit)
		IsSubDelegation:  false,
	}

	t.Run("Valid delegation succeeds", func(t *testing.T) {
		err := matrix.ValidateDelegationRequest(baseReq)
		if err != nil {
			t.Fatalf("expected valid delegation to succeed, got: %v", err)
		}
	})

	t.Run("Re-delegation / Multi-hop forbidden", func(t *testing.T) {
		req := baseReq
		req.IsSubDelegation = true
		err := matrix.ValidateDelegationRequest(req)
		if err == nil {
			t.Fatalf("expected error for multi-hop delegation, got nil")
		}
		if !errors.Is(err, localidentity.ErrMultiHopDelegationForbidden) {
			t.Errorf("expected ErrMultiHopDelegationForbidden, got: %v", err)
		}
	})

	t.Run("Self-delegation forbidden", func(t *testing.T) {
		req := baseReq
		req.DelegateeSubject = req.DelegatorSubject
		err := matrix.ValidateDelegationRequest(req)
		if err == nil {
			t.Fatalf("expected error for self-delegation, got nil")
		}
		if !errors.Is(err, localidentity.ErrSelfDelegationForbidden) {
			t.Errorf("expected ErrSelfDelegationForbidden, got: %v", err)
		}

		// Also test whitespace trimming
		req.DelegateeSubject = "  " + req.DelegatorSubject + " "
		err = matrix.ValidateDelegationRequest(req)
		if !errors.Is(err, localidentity.ErrSelfDelegationForbidden) {
			t.Errorf("expected ErrSelfDelegationForbidden with padded whitespace, got: %v", err)
		}
	})

	t.Run("Invalid temporal window", func(t *testing.T) {
		// ValidTo before ValidFrom
		req := baseReq
		req.ValidTo = now.Add(-1 * time.Hour)
		err := matrix.ValidateDelegationRequest(req)
		if err == nil {
			t.Fatalf("expected error for inverted delegation window, got nil")
		}
		if !errors.Is(err, localidentity.ErrInvalidDelegationWindow) {
			t.Errorf("expected ErrInvalidDelegationWindow, got: %v", err)
		}

		// ValidTo equal ValidFrom
		req.ValidTo = now
		err = matrix.ValidateDelegationRequest(req)
		if !errors.Is(err, localidentity.ErrInvalidDelegationWindow) {
			t.Errorf("expected ErrInvalidDelegationWindow when equal, got: %v", err)
		}
	})

	t.Run("Absolute maximum duration exceeded (> 30 days)", func(t *testing.T) {
		req := baseReq
		req.DelegatorRole = localidentity.RoleViewer
		req.DelegatedRole = localidentity.RoleViewer
		req.ValidTo = now.Add(31 * 24 * time.Hour)
		err := matrix.ValidateDelegationRequest(req)
		if err == nil {
			t.Fatalf("expected error exceeding 30-day max duration, got nil")
		}
		if !errors.Is(err, localidentity.ErrDelegationDurationExceeded) {
			t.Errorf("expected ErrDelegationDurationExceeded, got: %v", err)
		}
	})

	t.Run("Role-specific maximum delegation days exceeded", func(t *testing.T) {
		// PM is limited to 14 days; requesting 15 days
		req := baseReq
		req.ValidTo = now.Add(15 * 24 * time.Hour)
		err := matrix.ValidateDelegationRequest(req)
		if err == nil {
			t.Fatalf("expected error exceeding PM 14-day limit, got nil")
		}
		if !errors.Is(err, localidentity.ErrDelegationDurationExceeded) {
			t.Errorf("expected ErrDelegationDurationExceeded, got: %v", err)
		}
	})

	t.Run("Protected authority non-delegable: Delegating TenantAdmin role", func(t *testing.T) {
		req := baseReq
		req.DelegatorRole = localidentity.RoleTenantAdmin
		req.DelegatedRole = localidentity.RoleTenantAdmin
		err := matrix.ValidateDelegationRequest(req)
		if err == nil {
			t.Fatalf("expected error delegating protected TenantAdmin, got nil")
		}
		if !errors.Is(err, localidentity.ErrProtectedAuthorityNonDelegable) {
			t.Errorf("expected ErrProtectedAuthorityNonDelegable, got: %v", err)
		}
	})

	t.Run("Source authority containment: Delegator lacks requested permissions", func(t *testing.T) {
		// Inspector trying to delegate ProjectManager role
		req := baseReq
		req.DelegatorRole = localidentity.RoleInspector
		req.DelegatedRole = localidentity.RoleProjectManager
		err := matrix.ValidateDelegationRequest(req)
		if err == nil {
			t.Fatalf("expected error when delegator lacks delegated permissions, got nil")
		}
		if !errors.Is(err, localidentity.ErrExceedsSourceAuthority) {
			t.Errorf("expected ErrExceedsSourceAuthority, got: %v", err)
		}
	})

	t.Run("Scope containment: Delegated scope broader or outside delegator scope", func(t *testing.T) {
		// Delegator has ProjectID = proj_1; attempting to delegate at Company level
		req := baseReq
		req.DelegatorScope = localidentity.ScopeGrant{TenantID: "ten_1", CompanyID: "comp_1", ProjectID: "proj_1"}
		req.DelegatedScope = localidentity.ScopeGrant{TenantID: "ten_1", CompanyID: "comp_1"} // Broader than proj_1
		err := matrix.ValidateDelegationRequest(req)
		if err == nil {
			t.Fatalf("expected error when delegated scope exceeds delegator scope, got nil")
		}
		if !errors.Is(err, localidentity.ErrScopeExceedsSourceAuthority) {
			t.Errorf("expected ErrScopeExceedsSourceAuthority, got: %v", err)
		}

		// Sibling project escalation: Delegator on proj_1, attempting to delegate on proj_2
		req.DelegatedScope = localidentity.ScopeGrant{TenantID: "ten_1", CompanyID: "comp_1", ProjectID: "proj_2"}
		err = matrix.ValidateDelegationRequest(req)
		if err == nil {
			t.Fatalf("expected error when delegating to sibling project, got nil")
		}
		if !errors.Is(err, localidentity.ErrScopeExceedsSourceAuthority) {
			t.Errorf("expected ErrScopeExceedsSourceAuthority, got: %v", err)
		}

		// Cross-tenant escalation: Delegator on ten_1, attempting to delegate on ten_2
		req.DelegatedScope = localidentity.ScopeGrant{TenantID: "ten_2", CompanyID: "comp_1", ProjectID: "proj_1"}
		err = matrix.ValidateDelegationRequest(req)
		if err == nil {
			t.Fatalf("expected error when delegating across tenants, got nil")
		}
		if !errors.Is(err, localidentity.ErrScopeExceedsSourceAuthority) {
			t.Errorf("expected ErrScopeExceedsSourceAuthority, got: %v", err)
		}
	})
}

func TestAuthorizationMatrix_ScopeOverlapAndContains(t *testing.T) {
	// Overlap tests
	t.Run("Scope overlap checks", func(t *testing.T) {
		s1 := localidentity.ScopeGrant{TenantID: "ten_1", ProjectID: "proj_1"}
		s2 := localidentity.ScopeGrant{TenantID: "ten_1", ProjectID: "proj_1", SiteID: "site_a"}
		if !localidentity.CheckScopesOverlap(s1, s2) {
			t.Errorf("expected s1 and s2 to overlap")
		}

		s3 := localidentity.ScopeGrant{TenantID: "ten_1", ProjectID: "proj_2"}
		if localidentity.CheckScopesOverlap(s1, s3) {
			t.Errorf("expected sibling projects not to overlap")
		}

		sCrossTenant := localidentity.ScopeGrant{TenantID: "ten_2", ProjectID: "proj_1"}
		if localidentity.CheckScopesOverlap(s1, sCrossTenant) {
			t.Errorf("expected cross-tenant scopes not to overlap")
		}
	})

	// Contains tests
	t.Run("Scope contains checks", func(t *testing.T) {
		parent := localidentity.ScopeGrant{TenantID: "ten_1", CompanyID: "comp_1"}
		child := localidentity.ScopeGrant{TenantID: "ten_1", CompanyID: "comp_1", ProjectID: "proj_1"}
		otherComp := localidentity.ScopeGrant{TenantID: "ten_1", CompanyID: "comp_2"}

		if !localidentity.ScopeContains(parent, child) {
			t.Errorf("expected parent to contain child")
		}
		if localidentity.ScopeContains(child, parent) {
			t.Errorf("expected child not to contain parent")
		}
		if localidentity.ScopeContains(parent, otherComp) {
			t.Errorf("expected parent not to contain other company")
		}
	})
}
