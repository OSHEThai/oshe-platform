package localidentity_test

import (
	"testing"

	localidentity "github.com/oshethai/oshe-platform/modules/identity-authorization"
)

func newBaseEvaluator() *localidentity.PolicyEvaluator {
	eval := localidentity.NewPolicyEvaluator()
	eval.SetMembership("tenant-alpha", "user-lead", localidentity.MembershipActive)
	eval.SetMembership("tenant-alpha", "user-inspector", localidentity.MembershipActive)
	eval.SetMembership("tenant-alpha", "user-pm", localidentity.MembershipActive)
	eval.SetMembership("tenant-alpha", "user-auditor", localidentity.MembershipActive)
	eval.SetMembership("tenant-alpha", "user-viewer", localidentity.MembershipActive)
	eval.SetMembership("tenant-alpha", "user-contractor", localidentity.MembershipActive)
	eval.SetMembership("tenant-alpha", "user-support", localidentity.MembershipActive)
	return eval
}

func TestPolicy_PrivilegeEscalation(t *testing.T) {
	eval := newBaseEvaluator()
	eval.AddRoleAssignment(localidentity.RoleAssignment{
		Subject:  "user-inspector",
		TenantID: "tenant-alpha",
		Role:     localidentity.RoleInspector,
		Scope: localidentity.ScopeGrant{
			TenantID:  "tenant-alpha",
			ProjectID: "proj-alpha",
		},
	})

	// Inspector attempts DELETE (privilege escalation)
	req := localidentity.AccessRequest{
		Identity: localidentity.SubjectIdentity{
			Subject:         "user-inspector",
			TenantID:        "tenant-alpha",
			IsAuthenticated: true,
		},
		Target: localidentity.TargetResource{
			TenantID:  "tenant-alpha",
			ProjectID: "proj-alpha",
			Lifecycle: localidentity.ResourceActive,
		},
		Action: localidentity.ActionDelete,
	}

	res := eval.Evaluate(req)
	if res.Allowed {
		t.Fatalf("expected privilege escalation denial for Inspector DELETE")
	}
	if res.DenialReason != localidentity.DenialPrivilegeEscalation {
		t.Errorf("expected DenialPrivilegeEscalation, got: %s", res.DenialReason)
	}

	// Viewer attempts CREATE (privilege escalation)
	eval.AddRoleAssignment(localidentity.RoleAssignment{
		Subject:  "user-viewer",
		TenantID: "tenant-alpha",
		Role:     localidentity.RoleViewer,
		Scope: localidentity.ScopeGrant{
			TenantID:  "tenant-alpha",
			ProjectID: "proj-alpha",
		},
	})
	reqViewer := localidentity.AccessRequest{
		Identity: localidentity.SubjectIdentity{
			Subject:         "user-viewer",
			TenantID:        "tenant-alpha",
			IsAuthenticated: true,
		},
		Target: localidentity.TargetResource{
			TenantID:  "tenant-alpha",
			ProjectID: "proj-alpha",
			Lifecycle: localidentity.ResourceActive,
		},
		Action: localidentity.ActionCreate,
	}
	resViewer := eval.Evaluate(reqViewer)
	if resViewer.Allowed || resViewer.DenialReason != localidentity.DenialPrivilegeEscalation {
		t.Errorf("expected DenialPrivilegeEscalation for Viewer CREATE, got: %s", resViewer.DenialReason)
	}
}

func TestPolicy_InactiveMembership(t *testing.T) {
	eval := localidentity.NewPolicyEvaluator()

	// Suspended membership
	eval.SetMembership("tenant-alpha", "user-suspended", localidentity.MembershipSuspended)
	eval.AddRoleAssignment(localidentity.RoleAssignment{
		Subject:  "user-suspended",
		TenantID: "tenant-alpha",
		Role:     localidentity.RoleTenantAdmin,
		Scope:    localidentity.ScopeGrant{TenantID: "tenant-alpha"},
	})

	req := localidentity.AccessRequest{
		Identity: localidentity.SubjectIdentity{
			Subject:         "user-suspended",
			TenantID:        "tenant-alpha",
			IsAuthenticated: true,
		},
		Target: localidentity.TargetResource{
			TenantID:  "tenant-alpha",
			Lifecycle: localidentity.ResourceActive,
		},
		Action: localidentity.ActionRead,
	}

	res := eval.Evaluate(req)
	if res.Allowed || res.DenialReason != localidentity.DenialInactiveMembership {
		t.Fatalf("expected DenialInactiveMembership for suspended member, got: %s", res.DenialReason)
	}

	// Inactive membership
	eval.SetMembership("tenant-alpha", "user-inactive", localidentity.MembershipInactive)
	eval.AddRoleAssignment(localidentity.RoleAssignment{
		Subject:  "user-inactive",
		TenantID: "tenant-alpha",
		Role:     localidentity.RoleTenantAdmin,
		Scope:    localidentity.ScopeGrant{TenantID: "tenant-alpha"},
	})
	req.Identity.Subject = "user-inactive"
	resInactive := eval.Evaluate(req)
	if resInactive.Allowed || resInactive.DenialReason != localidentity.DenialInactiveMembership {
		t.Fatalf("expected DenialInactiveMembership for inactive member, got: %s", resInactive.DenialReason)
	}

	// Unknown non-member
	req.Identity.Subject = "user-nonexistent"
	resNonMember := eval.Evaluate(req)
	if resNonMember.Allowed || resNonMember.DenialReason != localidentity.DenialInactiveMembership {
		t.Fatalf("expected DenialInactiveMembership for non-member, got: %s", resNonMember.DenialReason)
	}
}

func TestPolicy_MissingRole(t *testing.T) {
	eval := localidentity.NewPolicyEvaluator()
	eval.SetMembership("tenant-alpha", "user-norole", localidentity.MembershipActive)

	req := localidentity.AccessRequest{
		Identity: localidentity.SubjectIdentity{
			Subject:         "user-norole",
			TenantID:        "tenant-alpha",
			IsAuthenticated: true,
		},
		Target: localidentity.TargetResource{
			TenantID:  "tenant-alpha",
			ProjectID: "proj-alpha",
			Lifecycle: localidentity.ResourceActive,
		},
		Action: localidentity.ActionRead,
	}

	res := eval.Evaluate(req)
	if res.Allowed || res.DenialReason != localidentity.DenialRoleNotGranted {
		t.Fatalf("expected DenialRoleNotGranted when user has zero role assignments, got: %s", res.DenialReason)
	}
}

func TestPolicy_ScopeMismatch_SiblingProjectAndCompanyWide(t *testing.T) {
	eval := newBaseEvaluator()

	// Grant on proj-alpha only
	eval.AddRoleAssignment(localidentity.RoleAssignment{
		Subject:  "user-pm",
		TenantID: "tenant-alpha",
		Role:     localidentity.RoleProjectManager,
		Scope: localidentity.ScopeGrant{
			TenantID:  "tenant-alpha",
			CompanyID: "comp-main",
			ProjectID: "proj-alpha",
		},
	})

	// Access attempt on sibling project proj-bravo
	reqSibling := localidentity.AccessRequest{
		Identity: localidentity.SubjectIdentity{
			Subject:         "user-pm",
			TenantID:        "tenant-alpha",
			IsAuthenticated: true,
		},
		Target: localidentity.TargetResource{
			TenantID:  "tenant-alpha",
			CompanyID: "comp-main",
			ProjectID: "proj-bravo", // sibling project!
			Lifecycle: localidentity.ResourceActive,
		},
		Action: localidentity.ActionRead,
	}

	res := eval.Evaluate(reqSibling)
	if res.Allowed {
		t.Fatalf("expected scope mismatch denial on sibling project access")
	}
	if res.DenialReason != localidentity.DenialScopeMismatch {
		t.Errorf("expected DenialScopeMismatch, got: %s", res.DenialReason)
	}

	// Implicit company-wide grant attempt for project manager (empty project ID)
	eval.AddRoleAssignment(localidentity.RoleAssignment{
		Subject:  "user-inspector",
		TenantID: "tenant-alpha",
		Role:     localidentity.RoleInspector,
		Scope: localidentity.ScopeGrant{
			TenantID:  "tenant-alpha",
			CompanyID: "comp-main",
			ProjectID: "", // implicit company-wide attempt
		},
	})
	reqCompanyWide := localidentity.AccessRequest{
		Identity: localidentity.SubjectIdentity{
			Subject:         "user-inspector",
			TenantID:        "tenant-alpha",
			IsAuthenticated: true,
		},
		Target: localidentity.TargetResource{
			TenantID:  "tenant-alpha",
			CompanyID: "comp-main",
			ProjectID: "proj-alpha",
			Lifecycle: localidentity.ResourceActive,
		},
		Action: localidentity.ActionRead,
	}
	resCompanyWide := eval.Evaluate(reqCompanyWide)
	if resCompanyWide.Allowed || resCompanyWide.DenialReason != localidentity.DenialScopeMismatch {
		t.Errorf("expected DenialScopeMismatch for implicit company-wide project role, got: %s", resCompanyWide.DenialReason)
	}

	// Unbounded contractor grant (must have project or site scope)
	eval.AddRoleAssignment(localidentity.RoleAssignment{
		Subject:  "user-contractor",
		TenantID: "tenant-alpha",
		Role:     localidentity.RoleContractor,
		Scope: localidentity.ScopeGrant{
			TenantID: "tenant-alpha",
		},
	})
	reqContractor := localidentity.AccessRequest{
		Identity: localidentity.SubjectIdentity{
			Subject:         "user-contractor",
			TenantID:        "tenant-alpha",
			IsAuthenticated: true,
		},
		Target: localidentity.TargetResource{
			TenantID:  "tenant-alpha",
			ProjectID: "proj-alpha",
			Lifecycle: localidentity.ResourceActive,
		},
		Action: localidentity.ActionRead,
	}
	resContractor := eval.Evaluate(reqContractor)
	if resContractor.Allowed || resContractor.DenialReason != localidentity.DenialScopeMismatch {
		t.Errorf("expected DenialScopeMismatch for unbounded contractor, got: %s", resContractor.DenialReason)
	}
}

func TestPolicy_DirectObjectMismatch(t *testing.T) {
	eval := newBaseEvaluator()
	eval.AddRoleAssignment(localidentity.RoleAssignment{
		Subject:  "user-inspector",
		TenantID: "tenant-alpha",
		Role:     localidentity.RoleInspector,
		Scope: localidentity.ScopeGrant{
			TenantID:  "tenant-alpha",
			ProjectID: "proj-alpha",
			ObjectID:  "evidence-record-001",
		},
	})

	// Access to another object in same project
	req := localidentity.AccessRequest{
		Identity: localidentity.SubjectIdentity{
			Subject:         "user-inspector",
			TenantID:        "tenant-alpha",
			IsAuthenticated: true,
		},
		Target: localidentity.TargetResource{
			TenantID:  "tenant-alpha",
			ProjectID: "proj-alpha",
			ObjectID:  "evidence-record-002",
			Lifecycle: localidentity.ResourceActive,
		},
		Action: localidentity.ActionRead,
	}

	res := eval.Evaluate(req)
	if res.Allowed || res.DenialReason != localidentity.DenialDirectObjectMismatch {
		t.Fatalf("expected DenialDirectObjectMismatch, got: %s", res.DenialReason)
	}

	// Access to the locked object succeeds
	req.Target.ObjectID = "evidence-record-001"
	resOK := eval.Evaluate(req)
	if !resOK.Allowed {
		t.Fatalf("expected direct object match to allow, got denial: %s", resOK.DenialReason)
	}
}

func TestPolicy_ArchivedRecordDenial(t *testing.T) {
	eval := newBaseEvaluator()

	// Inspector on proj-alpha
	eval.AddRoleAssignment(localidentity.RoleAssignment{
		Subject:  "user-inspector",
		TenantID: "tenant-alpha",
		Role:     localidentity.RoleInspector,
		Scope: localidentity.ScopeGrant{
			TenantID:  "tenant-alpha",
			ProjectID: "proj-alpha",
		},
	})

	// Mutation on archived record
	reqMutate := localidentity.AccessRequest{
		Identity: localidentity.SubjectIdentity{
			Subject:         "user-inspector",
			TenantID:        "tenant-alpha",
			IsAuthenticated: true,
		},
		Target: localidentity.TargetResource{
			TenantID:  "tenant-alpha",
			ProjectID: "proj-alpha",
			Lifecycle: localidentity.ResourceArchived,
		},
		Action: localidentity.ActionUpdate,
	}

	resMutate := eval.Evaluate(reqMutate)
	if resMutate.Allowed || resMutate.DenialReason != localidentity.DenialArchivedRecord {
		t.Fatalf("expected DenialArchivedRecord on update, got: %s", resMutate.DenialReason)
	}

	// Read on archived record by non-auditor
	reqRead := reqMutate
	reqRead.Action = localidentity.ActionRead
	resRead := eval.Evaluate(reqRead)
	if resRead.Allowed || resRead.DenialReason != localidentity.DenialArchivedRecord {
		t.Fatalf("expected DenialArchivedRecord on read by inspector, got: %s", resRead.DenialReason)
	}

	// Read on archived record by Auditor is permitted
	eval.AddRoleAssignment(localidentity.RoleAssignment{
		Subject:  "user-auditor",
		TenantID: "tenant-alpha",
		Role:     localidentity.RoleAuditor,
		Scope: localidentity.ScopeGrant{
			TenantID:  "tenant-alpha",
			ProjectID: "proj-alpha",
		},
	})
	reqAuditor := localidentity.AccessRequest{
		Identity: localidentity.SubjectIdentity{
			Subject:         "user-auditor",
			TenantID:        "tenant-alpha",
			IsAuthenticated: true,
		},
		Target: localidentity.TargetResource{
			TenantID:  "tenant-alpha",
			ProjectID: "proj-alpha",
			Lifecycle: localidentity.ResourceArchived,
		},
		Action: localidentity.ActionRead,
	}
	resAuditor := eval.Evaluate(reqAuditor)
	if !resAuditor.Allowed {
		t.Fatalf("expected Auditor to read archived record, got: %s", resAuditor.DenialReason)
	}
}

func TestPolicy_EntitlementSeparation(t *testing.T) {
	eval := newBaseEvaluator()

	// 1. Entitlement without Authorization: Tenant has entitlement, but user has no role
	eval.SetEntitlement("tenant-alpha", "FEATURE_DEEP_INSPECT", true)
	reqNoRole := localidentity.AccessRequest{
		Identity: localidentity.SubjectIdentity{
			Subject:         "user-norole",
			TenantID:        "tenant-alpha",
			IsAuthenticated: true,
		},
		Target: localidentity.TargetResource{
			TenantID:  "tenant-alpha",
			ProjectID: "proj-alpha",
			Lifecycle: localidentity.ResourceActive,
		},
		Action:              localidentity.ActionRead,
		RequiredEntitlement: "FEATURE_DEEP_INSPECT",
	}
	eval.SetMembership("tenant-alpha", "user-norole", localidentity.MembershipActive)
	resNoRole := eval.Evaluate(reqNoRole)
	if resNoRole.Allowed || resNoRole.DenialReason != localidentity.DenialRoleNotGranted {
		t.Fatalf("entitlement must not bypass authorization; expected DenialRoleNotGranted, got: %s", resNoRole.DenialReason)
	}

	// 2. Missing Entitlement with Valid Role: User has PM role, but tenant lacks required entitlement
	eval.AddRoleAssignment(localidentity.RoleAssignment{
		Subject:  "user-pm",
		TenantID: "tenant-alpha",
		Role:     localidentity.RoleProjectManager,
		Scope: localidentity.ScopeGrant{
			TenantID:  "tenant-alpha",
			ProjectID: "proj-alpha",
		},
	})
	reqMissingEnt := localidentity.AccessRequest{
		Identity: localidentity.SubjectIdentity{
			Subject:         "user-pm",
			TenantID:        "tenant-alpha",
			IsAuthenticated: true,
		},
		Target: localidentity.TargetResource{
			TenantID:  "tenant-alpha",
			ProjectID: "proj-alpha",
			Lifecycle: localidentity.ResourceActive,
		},
		Action:              localidentity.ActionRead,
		RequiredEntitlement: "FEATURE_ADVANCED_ANALYTICS",
	}
	resMissingEnt := eval.Evaluate(reqMissingEnt)
	if resMissingEnt.Allowed || resMissingEnt.DenialReason != localidentity.DenialMissingEntitlement {
		t.Fatalf("expected DenialMissingEntitlement, got: %s", resMissingEnt.DenialReason)
	}

	// 3. Enabling entitlement permits access
	eval.SetEntitlement("tenant-alpha", "FEATURE_ADVANCED_ANALYTICS", true)
	resActiveEnt := eval.Evaluate(reqMissingEnt)
	if !resActiveEnt.Allowed {
		t.Fatalf("expected allowed access when entitlement is active, got denial: %s", resActiveEnt.DenialReason)
	}
}

func TestPolicy_CrossTenantDenial(t *testing.T) {
	eval := newBaseEvaluator()
	eval.AddRoleAssignment(localidentity.RoleAssignment{
		Subject:  "user-lead",
		TenantID: "tenant-alpha",
		Role:     localidentity.RoleTenantAdmin,
		Scope:    localidentity.ScopeGrant{TenantID: "tenant-alpha"},
	})

	reqCross := localidentity.AccessRequest{
		Identity: localidentity.SubjectIdentity{
			Subject:         "user-lead",
			TenantID:        "tenant-alpha",
			IsAuthenticated: true,
		},
		Target: localidentity.TargetResource{
			TenantID:  "tenant-beta", // cross-tenant!
			ProjectID: "proj-beta",
			Lifecycle: localidentity.ResourceActive,
		},
		Action: localidentity.ActionRead,
	}

	res := eval.Evaluate(reqCross)
	if res.Allowed || res.DenialReason != localidentity.DenialCrossTenant {
		t.Fatalf("expected DenialCrossTenant, got: %s", res.DenialReason)
	}
}

func TestPolicy_UnknownRole(t *testing.T) {
	eval := newBaseEvaluator()
	eval.AddRoleAssignment(localidentity.RoleAssignment{
		Subject:  "user-lead",
		TenantID: "tenant-alpha",
		Role:     localidentity.Role("SUPER_GLOBAL_ADMIN"),
		Scope:    localidentity.ScopeGrant{TenantID: "tenant-alpha"},
	})

	req := localidentity.AccessRequest{
		Identity: localidentity.SubjectIdentity{
			Subject:         "user-lead",
			TenantID:        "tenant-alpha",
			IsAuthenticated: true,
		},
		Target: localidentity.TargetResource{
			TenantID:  "tenant-alpha",
			ProjectID: "proj-alpha",
			Lifecycle: localidentity.ResourceActive,
		},
		Action: localidentity.ActionRead,
	}

	res := eval.Evaluate(req)
	if res.Allowed || res.DenialReason != localidentity.DenialUnknownRole {
		t.Fatalf("expected DenialUnknownRole for unrecognized role, got: %s", res.DenialReason)
	}
}

func TestPolicy_DelegationPlaceholderDenial(t *testing.T) {
	eval := newBaseEvaluator()
	eval.AddRoleAssignment(localidentity.RoleAssignment{
		Subject:  "user-lead",
		TenantID: "tenant-alpha",
		Role:     localidentity.RoleTenantAdmin,
		Scope:    localidentity.ScopeGrant{TenantID: "tenant-alpha"},
	})

	req := localidentity.AccessRequest{
		Identity: localidentity.SubjectIdentity{
			Subject:         "user-lead",
			TenantID:        "tenant-alpha",
			IsAuthenticated: true,
		},
		Target: localidentity.TargetResource{
			TenantID:  "tenant-alpha",
			Lifecycle: localidentity.ResourceActive,
		},
		Action: localidentity.ActionRead,
		Delegation: localidentity.DelegationContext{
			IsDelegated: true,
			Delegator:   "user-owner",
		},
	}

	res := eval.Evaluate(req)
	if res.Allowed || res.DenialReason != localidentity.DenialDelegationNotImplemented {
		t.Fatalf("expected DenialDelegationNotImplemented for delegation placeholder, got: %s", res.DenialReason)
	}
}

func TestPolicy_UnauthenticatedCaller(t *testing.T) {
	eval := newBaseEvaluator()

	reqUnauth := localidentity.AccessRequest{
		Identity: localidentity.SubjectIdentity{
			Subject:         "user-lead",
			TenantID:        "tenant-alpha",
			IsAuthenticated: false,
		},
		Target: localidentity.TargetResource{
			TenantID:  "tenant-alpha",
			Lifecycle: localidentity.ResourceActive,
		},
		Action: localidentity.ActionRead,
	}

	res := eval.Evaluate(reqUnauth)
	if res.Allowed || res.DenialReason != localidentity.DenialUnauthenticated {
		t.Fatalf("expected DenialUnauthenticated, got: %s", res.DenialReason)
	}
}
