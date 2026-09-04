package localidentity_test

import (
	"crypto/sha256"
	"errors"
	"testing"
	"time"

	localidentity "github.com/oshethai/oshe-platform/modules/identity-authorization"
)

func dummyDigest(seed string) [32]byte {
	return sha256.Sum256([]byte(seed))
}

func TestRevocationRegistry_RevokeSpecificSession(t *testing.T) {
	now := time.Date(2026, 9, 4, 18, 0, 0, 0, time.UTC)
	reg := localidentity.NewRevocationRegistry(func() time.Time { return now })

	tokenDig := dummyDigest("token-session-101")
	tenantID := "ten_org_alpha"
	subject := "usr_alice"

	// Prior to revocation: session is valid
	diag := reg.EvaluateSession(tokenDig, tenantID, tenantID, subject, 1)
	if !diag.Allowed {
		t.Fatalf("expected session to be allowed prior to revocation, got: %+v", diag)
	}
	if diag.DenialCategory != localidentity.CategoryNone {
		t.Errorf("expected denial category NONE, got %s", diag.DenialCategory)
	}

	// Revoke session token
	err := reg.RevokeSessionToken(tokenDig, tenantID, subject, "user logged out")
	if err != nil {
		t.Fatalf("unexpected error revoking session: %v", err)
	}

	// After revocation: session is denied with SESSION_REVOKED
	diagAfter := reg.EvaluateSession(tokenDig, tenantID, tenantID, subject, 1)
	if diagAfter.Allowed {
		t.Fatalf("expected revoked session to be denied")
	}
	if diagAfter.DenialCategory != localidentity.CategorySessionRevoked {
		t.Errorf("expected denial category SESSION_REVOKED, got %s", diagAfter.DenialCategory)
	}
}

func TestRevocationRegistry_StaleSessionBySubject(t *testing.T) {
	reg := localidentity.NewRevocationRegistry(nil)
	tokenDig := dummyDigest("token-alice-old")
	tenantID := "ten_org_alpha"
	subject := "usr_alice"

	// Session issued under generation 1
	sessionGen := int64(1)

	// Invalidate all prior sessions for subject
	err := reg.RevokeSubject(tenantID, subject, "credential rotation")
	if err != nil {
		t.Fatalf("unexpected error revoking subject: %v", err)
	}

	// Older session generation must be evaluated as stale
	diag := reg.EvaluateSession(tokenDig, tenantID, tenantID, subject, sessionGen)
	if diag.Allowed {
		t.Fatalf("expected stale session to be denied")
	}
	if diag.DenialCategory != localidentity.CategorySessionStale {
		t.Errorf("expected denial category SESSION_STALE, got %s", diag.DenialCategory)
	}

	// Fresh session generation issued after subject revocation must be valid
	freshGen := diag.DecisionGeneration + 1
	freshDig := dummyDigest("token-alice-new")
	diagFresh := reg.EvaluateSession(freshDig, tenantID, tenantID, subject, freshGen)
	if !diagFresh.Allowed {
		t.Errorf("expected fresh session to be allowed, got: %+v", diagFresh)
	}
}

func TestRevocationRegistry_ChangedPolicyGeneration(t *testing.T) {
	reg := localidentity.NewRevocationRegistry(nil)
	tokenDig := dummyDigest("token-bob")
	tenantID := "ten_org_beta"
	subject := "usr_bob"

	// Bump tenant policy generation (e.g. role redefinition or permissions updated)
	newPolicyGen, err := reg.BumpTenantPolicyGeneration(tenantID, "role permissions updated")
	if err != nil {
		t.Fatalf("unexpected error bumping policy generation: %v", err)
	}

	// Session issued under an older generation than newPolicyGen must be rejected as policy stale
	olderSessionGen := newPolicyGen - 1
	diag := reg.EvaluateSession(tokenDig, tenantID, tenantID, subject, olderSessionGen)
	if diag.Allowed {
		t.Fatalf("expected session issued under older policy generation to be denied")
	}
	if diag.DenialCategory != localidentity.CategoryPolicyStale {
		t.Errorf("expected denial category POLICY_GENERATION_STALE, got %s", diag.DenialCategory)
	}

	// Session aligned with or newer than current policy generation is accepted
	alignedDiag := reg.EvaluateSession(tokenDig, tenantID, tenantID, subject, newPolicyGen)
	if !alignedDiag.Allowed {
		t.Errorf("expected aligned session to be allowed, got: %+v", alignedDiag)
	}
}

func TestRevocationRegistry_CrossTenantDenial(t *testing.T) {
	reg := localidentity.NewRevocationRegistry(nil)
	tokenDig := dummyDigest("token-cross-tenant")

	// Caller tenant does not match target resource tenant
	diag := reg.EvaluateSession(tokenDig, "ten_caller", "ten_target", "usr_charlie", 1)
	if diag.Allowed {
		t.Fatalf("expected cross-tenant access to be denied")
	}
	if diag.DenialCategory != localidentity.CategoryCrossTenantDenied {
		t.Errorf("expected CROSS_TENANT_ACCESS_DENIED, got %s", diag.DenialCategory)
	}

	// Empty tenant context fails closed
	diagEmpty := reg.EvaluateSession(tokenDig, "", "ten_target", "usr_charlie", 1)
	if diagEmpty.Allowed || diagEmpty.DenialCategory != localidentity.CategoryCrossTenantDenied {
		t.Errorf("expected cross tenant denial on empty caller tenant")
	}
}

func TestRevocationRegistry_NonLeakingDiagnostics(t *testing.T) {
	reg := localidentity.NewRevocationRegistry(nil)
	tokenDig := dummyDigest("token-leak-test")

	diag := reg.EvaluateSession(tokenDig, "ten_caller", "ten_other_tenant", "usr_dan", 1)

	// Summary must not leak names or foreign internal details
	if diag.Summary != "tenant context mismatch" {
		t.Errorf("expected non-leaking summary, got %q", diag.Summary)
	}
	if diag.DecisionGeneration <= 0 {
		t.Errorf("expected positive decision generation, got %d", diag.DecisionGeneration)
	}
}

func TestRevocationRegistry_AppendOnlyAuditBehavior(t *testing.T) {
	reg := localidentity.NewRevocationRegistry(nil)
	tenantID := "ten_audit_test"
	subject := "usr_evan"
	tokenDig := dummyDigest("token-evan-audit")

	// Perform operations that trigger audit events
	_ = reg.RevokeSessionToken(tokenDig, tenantID, subject, "compromised token")
	_ = reg.RevokeSubject(tenantID, subject, "terminated employment")
	_, _ = reg.BumpTenantPolicyGeneration(tenantID, "security policy update")

	trail, err := reg.AuditTrail(tenantID)
	if err != nil {
		t.Fatalf("failed to retrieve audit trail: %v", err)
	}

	if len(trail) != 3 {
		t.Fatalf("expected exactly 3 audit entries, got %d", len(trail))
	}

	// Monotonic sequence numbering
	for i := range len(trail) {
		expectedSeq := int64(i + 1)
		if trail[i].SequenceNumber != expectedSeq {
			t.Errorf("entry %d: expected sequence %d, got %d", i, expectedSeq, trail[i].SequenceNumber)
		}
	}

	if trail[0].EventType != localidentity.EventSessionRevoked {
		t.Errorf("entry 0: expected EventSessionRevoked, got %s", trail[0].EventType)
	}
	if trail[1].EventType != localidentity.EventSubjectRevoked {
		t.Errorf("entry 1: expected EventSubjectRevoked, got %s", trail[1].EventType)
	}
	if trail[2].EventType != localidentity.EventPolicyGenerationBump {
		t.Errorf("entry 2: expected EventPolicyGenerationBump, got %s", trail[2].EventType)
	}

	// Foreign tenant cannot see these audit events
	foreignTrail, err := reg.AuditTrail("ten_foreign")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(foreignTrail) != 0 {
		t.Errorf("foreign tenant should see 0 events, got %d", len(foreignTrail))
	}

	// Empty tenant ID fails closed
	if _, err := reg.AuditTrail(""); !errors.Is(err, localidentity.ErrBlankTenantID) {
		t.Errorf("expected ErrBlankTenantID on empty audit query, got: %v", err)
	}
}

func TestRevocationRegistry_ValidCurrentDecision(t *testing.T) {
	reg := localidentity.NewRevocationRegistry(nil)
	tokenDig := dummyDigest("token-active")
	tenantID := "ten_valid"
	subject := "usr_valid"

	diag := reg.EvaluateSession(tokenDig, tenantID, tenantID, subject, 1)
	if !diag.Allowed {
		t.Errorf("expected valid active session to be allowed, got: %+v", diag)
	}
	if diag.DenialCategory != localidentity.CategoryNone {
		t.Errorf("expected denial category NONE, got %s", diag.DenialCategory)
	}
}
