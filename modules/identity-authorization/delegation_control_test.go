package localidentity

import (
	"errors"
	"testing"
	"time"
)

func TestDelegationRecord_CreationAndAccessors(t *testing.T) {
	tenantID := "ten_alpha"
	delegator := "usr_synth_pm_01"
	delegatee := "usr_synth_lead_02"
	baseTime := time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC)
	from := baseTime
	to := baseTime.Add(7 * 24 * time.Hour)

	delegatorScope := ScopeGrant{TenantID: tenantID, CompanyID: "cmp_main", ProjectID: "prj_01"}
	delegatedScope := ScopeGrant{TenantID: tenantID, CompanyID: "cmp_main", ProjectID: "prj_01", SiteID: "ste_01"}

	d, err := NewDelegationRecord(
		"del_01", tenantID,
		delegator, RoleProjectManager, delegatorScope,
		delegatee, RoleInspector, delegatedScope,
		from, to, "HDEC-DELEGATION-APPROVAL",
		1, false,
	)
	if err != nil {
		t.Fatalf("unexpected NewDelegationRecord error: %v", err)
	}

	if d.DelegationID() != "del_01" {
		t.Errorf("delegationID mismatch: %s", d.DelegationID())
	}
	if d.TenantID() != tenantID {
		t.Errorf("tenantID mismatch: %s", d.TenantID())
	}
	if d.DelegatorSubject() != delegator {
		t.Errorf("delegator mismatch: %s", d.DelegatorSubject())
	}
	if d.DelegateeSubject() != delegatee {
		t.Errorf("delegatee mismatch: %s", d.DelegateeSubject())
	}
	if d.DelegatorRole() != RoleProjectManager || d.DelegatedRole() != RoleInspector {
		t.Errorf("roles mismatch: delegator=%s, delegated=%s", d.DelegatorRole(), d.DelegatedRole())
	}
	if d.ChainDepth() != 1 {
		t.Errorf("chain depth mismatch: %d", d.ChainDepth())
	}
	if !d.IsActive() || d.State() != DelegationStateActive {
		t.Errorf("expected active delegation state")
	}
}

func TestDelegationRecord_ValidationRejections(t *testing.T) {
	tenantID := "ten_alpha"
	delegator := "usr_synth_pm"
	delegatee := "usr_synth_sub"
	from := time.Now()
	to := from.Add(7 * 24 * time.Hour)
	scope := ScopeGrant{TenantID: tenantID, ProjectID: "prj_01"}

	// 1. Blank delegation ID
	if _, err := NewDelegationRecord("", tenantID, delegator, RoleProjectManager, scope, delegatee, RoleInspector, scope, from, to, "Appr", 1, false); !errors.Is(err, ErrBlankDelegationID) {
		t.Errorf("expected ErrBlankDelegationID, got %v", err)
	}

	// 2. Blank tenant ID
	if _, err := NewDelegationRecord("del_01", "", delegator, RoleProjectManager, scope, delegatee, RoleInspector, scope, from, to, "Appr", 1, false); !errors.Is(err, ErrBlankTenantID) {
		t.Errorf("expected ErrBlankTenantID, got %v", err)
	}

	// 3. Self-delegation forbidden
	if _, err := NewDelegationRecord("del_01", tenantID, delegator, RoleProjectManager, scope, delegator, RoleInspector, scope, from, to, "Appr", 1, false); !errors.Is(err, ErrSelfDelegationForbidden) {
		t.Errorf("expected ErrSelfDelegationForbidden, got %v", err)
	}

	// 4. Inverted / equal dates
	if _, err := NewDelegationRecord("del_01", tenantID, delegator, RoleProjectManager, scope, delegatee, RoleInspector, scope, to, from, "Appr", 1, false); !errors.Is(err, ErrInvalidDelegationWindow) {
		t.Errorf("expected ErrInvalidDelegationWindow on inverted dates, got %v", err)
	}

	// 5. Exceeds max delegation duration (>30 days)
	longTo := from.Add(31 * 24 * time.Hour)
	if _, err := NewDelegationRecord("del_01", tenantID, delegator, RoleProjectManager, scope, delegatee, RoleInspector, scope, from, longTo, "Appr", 1, false); !errors.Is(err, ErrDelegationDurationExceeded) {
		t.Errorf("expected ErrDelegationDurationExceeded for >30 days, got %v", err)
	}

	// 6. Chain depth > 1 rejected
	if _, err := NewDelegationRecord("del_01", tenantID, delegator, RoleProjectManager, scope, delegatee, RoleInspector, scope, from, to, "Appr", 2, false); !errors.Is(err, ErrUnauthorizedChainDepth) {
		t.Errorf("expected ErrUnauthorizedChainDepth for depth 2, got %v", err)
	}

	// 7. Sub-delegation flag rejected
	if _, err := NewDelegationRecord("del_01", tenantID, delegator, RoleProjectManager, scope, delegatee, RoleInspector, scope, from, to, "Appr", 1, true); !errors.Is(err, ErrUnauthorizedChainDepth) {
		t.Errorf("expected ErrUnauthorizedChainDepth for isSubDelegation=true, got %v", err)
	}
}

func TestDelegationControl_EmergencyAccessDenial(t *testing.T) {
	// AssertEmergencyAccessDenied must strictly fail closed if emergency access flag is set
	if err := AssertEmergencyAccessDenied(true); !errors.Is(err, ErrEmergencyAccessDenied) {
		t.Errorf("expected ErrEmergencyAccessDenied for emergency access, got %v", err)
	}
	if err := AssertEmergencyAccessDenied(false); err != nil {
		t.Errorf("expected nil error for standard access, got %v", err)
	}
}

func TestDelegationControl_ProtectedAuthorityNonDelegable(t *testing.T) {
	matrix := NewProvisionalAuthorizationMatrix()
	tenantID := "ten_alpha"
	from := time.Now()
	to := from.Add(7 * 24 * time.Hour)
	scope := ScopeGrant{TenantID: tenantID}

	// 1. Delegating TenantAdmin role is prohibited
	recAdmin, _ := NewDelegationRecord("del_adm", tenantID, "usr_admin", RoleTenantAdmin, scope, "usr_sub", RoleTenantAdmin, scope, from, to, "Appr", 1, false)
	err := ValidateDelegationAuthority(recAdmin, &matrix)
	if !errors.Is(err, ErrProtectedAuthorityNonDelegable) {
		t.Errorf("expected ErrProtectedAuthorityNonDelegable when delegating TenantAdmin, got %v", err)
	}

	// 2. Delegator role cannot be TenantAdmin (sovereign authority cannot delegate)
	recFromAdmin, _ := NewDelegationRecord("del_from_adm", tenantID, "usr_admin", RoleTenantAdmin, scope, "usr_sub", RoleInspector, scope, from, to, "Appr", 1, false)
	err = ValidateDelegationAuthority(recFromAdmin, &matrix)
	if !errors.Is(err, ErrProtectedAuthorityNonDelegable) {
		t.Errorf("expected ErrProtectedAuthorityNonDelegable for delegator TenantAdmin, got %v", err)
	}
}

func TestDelegationControl_SourceAuthorityContainment(t *testing.T) {
	matrix := NewProvisionalAuthorizationMatrix()
	tenantID := "ten_alpha"
	from := time.Now()
	to := from.Add(7 * 24 * time.Hour)

	// 1. Permission containment: RoleInspector lacks permissions of RoleProjectManager (cannot escalate)
	scope := ScopeGrant{TenantID: tenantID, ProjectID: "prj_01"}
	recEscalate, _ := NewDelegationRecord("del_esc", tenantID, "usr_insp", RoleInspector, scope, "usr_sub", RoleProjectManager, scope, from, to, "Appr", 1, false)
	err := ValidateDelegationAuthority(recEscalate, &matrix)
	if !errors.Is(err, ErrExceedsSourceAuthority) {
		t.Errorf("expected ErrExceedsSourceAuthority when Inspector attempts to delegate PM role, got %v", err)
	}

	// 2. Scope containment: Delegator on Project A cannot delegate Project B
	scopeA := ScopeGrant{TenantID: tenantID, ProjectID: "prj_alpha"}
	scopeB := ScopeGrant{TenantID: tenantID, ProjectID: "prj_bravo"}
	recCrossScope, _ := NewDelegationRecord("del_cross", tenantID, "usr_pm", RoleProjectManager, scopeA, "usr_sub", RoleInspector, scopeB, from, to, "Appr", 1, false)
	err = ValidateDelegationAuthority(recCrossScope, &matrix)
	if !errors.Is(err, ErrScopeExceedsSourceAuthority) {
		t.Errorf("expected ErrScopeExceedsSourceAuthority when delegating scope outside delegator scope, got %v", err)
	}
}

func TestDelegationControl_TemporalValidityAndRevocation(t *testing.T) {
	tenantID := "ten_alpha"
	baseTime := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	from := baseTime
	to := baseTime.Add(7 * 24 * time.Hour)
	now := baseTime.Add(2 * 24 * time.Hour)
	scope := ScopeGrant{TenantID: tenantID, ProjectID: "prj_01"}

	rec, _ := NewDelegationRecord("del_01", tenantID, "usr_pm", RoleProjectManager, scope, "usr_sub", RoleInspector, scope, from, to, "Appr", 1, false)

	// Temporal validity checks
	if !rec.IsValidAt(now) {
		t.Errorf("expected valid during window")
	}
	if rec.IsValidAt(baseTime.Add(-1 * time.Hour)) {
		t.Errorf("expected invalid before window")
	}
	if rec.IsValidAt(to.Add(1 * time.Hour)) {
		t.Errorf("expected invalid after window")
	}

	// Revoke delegation
	revoked, audit, err := rec.Revoke("usr_compliance", "Supervisor returned early", now)
	if err != nil {
		t.Fatalf("Revoke error: %v", err)
	}
	if revoked.IsActive() || revoked.State() != DelegationStateRevoked {
		t.Errorf("expected revoked state")
	}
	if revoked.IsValidAt(now) {
		t.Errorf("revoked delegation must not be valid")
	}
	if audit.Transition != "DELEGATION_REVOKED" || audit.ActorSubject != "usr_compliance" {
		t.Errorf("audit record mismatch: %+v", audit)
	}

	// Re-revoking fails closed
	_, _, err = revoked.Revoke("usr_compliance", "Duplicate", now)
	if !errors.Is(err, ErrDelegationRevoked) {
		t.Errorf("expected ErrDelegationRevoked on re-revocation, got %v", err)
	}
}

func TestDelegationControl_RegistryAndLedger(t *testing.T) {
	ledger := NewDelegationLedger()
	registry := NewDelegationRegistry(ledger, nil)

	tenantID := "ten_alpha"
	from := time.Now().Add(-1 * time.Hour)
	to := time.Now().Add(5 * 24 * time.Hour)
	now := time.Now()
	scope := ScopeGrant{TenantID: tenantID, ProjectID: "prj_01"}

	rec, _ := NewDelegationRecord("del_reg_01", tenantID, "usr_pm", RoleProjectManager, scope, "usr_sub", RoleInspector, scope, from, to, "Appr", 1, false)

	// 1. Create delegation
	if err := registry.CreateDelegation(rec, "usr_pm", "Delegating inspection coverage", now); err != nil {
		t.Fatalf("CreateDelegation error: %v", err)
	}

	// 2. Duplicate registration rejected
	if err := registry.CreateDelegation(rec, "usr_pm", "Duplicate", now); !errors.Is(err, ErrDuplicateDelegationID) {
		t.Errorf("expected ErrDuplicateDelegationID, got %v", err)
	}

	// 3. List active delegations for delegatee
	list, err := registry.ListActiveDelegationsForDelegatee(tenantID, "usr_sub", now)
	if err != nil || len(list) != 1 || list[0].DelegationID() != "del_reg_01" {
		t.Errorf("ListActiveDelegationsForDelegatee failed: len=%d, err=%v", len(list), err)
	}

	// 4. Revoke delegation
	revoked, err := registry.RevokeDelegation(tenantID, "del_reg_01", "usr_pm", "End of coverage", now)
	if err != nil || revoked.State() != DelegationStateRevoked {
		t.Fatalf("RevokeDelegation failed: %v", err)
	}

	// Active list returns 0 after revocation
	listAfter, _ := registry.ListActiveDelegationsForDelegatee(tenantID, "usr_sub", now)
	if len(listAfter) != 0 {
		t.Errorf("expected 0 active delegations after revocation, got %d", len(listAfter))
	}

	// 5. Audit trail verification
	trail, err := ledger.GetDelegationAuditTrail(tenantID, "del_reg_01")
	if err != nil || len(trail) != 2 {
		t.Fatalf("expected 2 audit records, got %d (err: %v)", len(trail), err)
	}
	if trail[0].Transition != "DELEGATION_CREATED" || trail[1].Transition != "DELEGATION_REVOKED" {
		t.Errorf("audit transitions mismatch: %+v", trail)
	}

	// 6. Cross-tenant isolation verification
	foreignTrail, err := ledger.GetDelegationAuditTrail("ten_other", "del_reg_01")
	if err != nil || len(foreignTrail) != 0 {
		t.Errorf("cross-tenant leakage: foreign tenant retrieved delegation audit records")
	}
}

func TestDelegationControl_EvaluateDelegatedAccess_EndToEnd(t *testing.T) {
	evaluator := NewPolicyEvaluator()
	registry := NewDelegationRegistry(nil, nil)

	tenantID := "ten_alpha"
	delegator := "usr_synth_pm"
	delegatee := "usr_synth_lead"

	// Set active memberships
	evaluator.SetMembership(tenantID, delegator, MembershipActive)
	evaluator.SetMembership(tenantID, delegatee, MembershipActive)

	from := time.Now().Add(-1 * time.Hour)
	to := time.Now().Add(5 * 24 * time.Hour)
	now := time.Now()

	// Delegator PM grants Inspector authority on Project Alpha to delegatee
	scope := ScopeGrant{TenantID: tenantID, ProjectID: "prj_alpha"}
	rec, _ := NewDelegationRecord("del_eval", tenantID, delegator, RoleProjectManager, scope, delegatee, RoleInspector, scope, from, to, "ApprovalRef", 1, false)
	_ = registry.CreateDelegation(rec, delegator, "Coverage", now)

	// Target 1: Delegatee requests READ on prj_alpha -> ALLOWED
	req1 := AccessRequest{
		Identity: SubjectIdentity{Subject: delegatee, TenantID: tenantID, IsAuthenticated: true},
		Target:   TargetResource{TenantID: tenantID, ProjectID: "prj_alpha", Lifecycle: ResourceActive},
		Action:   ActionRead,
	}
	res1 := EvaluateDelegatedAccess(registry, evaluator, req1, now)
	if !res1.Allowed {
		t.Errorf("expected delegated access allowed for Read on prj_alpha, got denial: %s", res1.DenialReason)
	}

	// Target 2: Delegatee requests CREATE on prj_alpha -> ALLOWED (Inspector permits Create)
	req2 := AccessRequest{
		Identity: SubjectIdentity{Subject: delegatee, TenantID: tenantID, IsAuthenticated: true},
		Target:   TargetResource{TenantID: tenantID, ProjectID: "prj_alpha", Lifecycle: ResourceActive},
		Action:   ActionCreate,
	}
	res2 := EvaluateDelegatedAccess(registry, evaluator, req2, now)
	if !res2.Allowed {
		t.Errorf("expected delegated access allowed for Create on prj_alpha, got denial: %s", res2.DenialReason)
	}

	// Target 3: Delegatee requests DELETE on prj_alpha -> DENIED with DenialPrivilegeEscalation
	req3 := AccessRequest{
		Identity: SubjectIdentity{Subject: delegatee, TenantID: tenantID, IsAuthenticated: true},
		Target:   TargetResource{TenantID: tenantID, ProjectID: "prj_alpha", Lifecycle: ResourceActive},
		Action:   ActionDelete,
	}
	res3 := EvaluateDelegatedAccess(registry, evaluator, req3, now)
	if res3.Allowed || res3.DenialReason != DenialPrivilegeEscalation {
		t.Errorf("expected DenialPrivilegeEscalation for delegated Delete, got allowed=%v reason=%s", res3.Allowed, res3.DenialReason)
	}

	// Target 4: Delegatee requests READ on unrelated prj_other -> DENIED with DenialScopeMismatch
	req4 := AccessRequest{
		Identity: SubjectIdentity{Subject: delegatee, TenantID: tenantID, IsAuthenticated: true},
		Target:   TargetResource{TenantID: tenantID, ProjectID: "prj_other", Lifecycle: ResourceActive},
		Action:   ActionRead,
	}
	res4 := EvaluateDelegatedAccess(registry, evaluator, req4, now)
	if res4.Allowed || res4.DenialReason != DenialScopeMismatch {
		t.Errorf("expected DenialScopeMismatch for unrelated project, got allowed=%v reason=%s", res4.Allowed, res4.DenialReason)
	}

	// Target 5: After revocation -> DENIED with DenialRoleNotGranted
	_, _ = registry.RevokeDelegation(tenantID, "del_eval", delegator, "Revoked early", now)
	res5 := EvaluateDelegatedAccess(registry, evaluator, req1, now)
	if res5.Allowed || res5.DenialReason != DenialRoleNotGranted {
		t.Errorf("expected DenialRoleNotGranted after delegation revoked, got allowed=%v reason=%s", res5.Allowed, res5.DenialReason)
	}
}
