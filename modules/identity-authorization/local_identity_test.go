package localidentity_test

import (
	"crypto/sha256"
	"errors"
	"strings"
	"testing"
	"time"

	localidentity "github.com/oshethai/oshe-platform/modules/identity-authorization"
)

type mockClock struct {
	now time.Time
}

func (c *mockClock) Now() time.Time {
	return c.now
}

func (c *mockClock) Advance(d time.Duration) {
	c.now = c.now.Add(d)
}

func newTestClock(t *testing.T) *mockClock {
	t.Helper()
	return &mockClock{
		now: time.Date(2026, 9, 4, 18, 0, 0, 0, time.UTC),
	}
}

func TestIdentity_ConstructionValidation(t *testing.T) {
	// Valid Active
	id, err := localidentity.NewIdentity("user-lead-01", "tenant-alpha", localidentity.IdentityActive)
	if err != nil {
		t.Fatalf("unexpected NewIdentity error: %v", err)
	}
	if id.Subject() != "user-lead-01" || id.TenantID() != "tenant-alpha" || id.State() != localidentity.IdentityActive || !id.IsActive() {
		t.Errorf("identity fields mismatch: %+v", id)
	}

	// Valid Disabled
	idDis, err := localidentity.NewIdentity("user-disabled", "tenant-alpha", localidentity.IdentityDisabled)
	if err != nil {
		t.Fatalf("unexpected NewIdentity error: %v", err)
	}
	if idDis.IsActive() {
		t.Errorf("expected disabled identity to report IsActive == false")
	}

	// Blank Subject
	if _, err := localidentity.NewIdentity("", "tenant-alpha", localidentity.IdentityActive); !errors.Is(err, localidentity.ErrBlankSubject) {
		t.Errorf("expected ErrBlankSubject for empty subject, got %v", err)
	}
	if _, err := localidentity.NewIdentity("   \t", "tenant-alpha", localidentity.IdentityActive); !errors.Is(err, localidentity.ErrBlankSubject) {
		t.Errorf("expected ErrBlankSubject for whitespace subject, got %v", err)
	}

	// Blank Tenant ID
	if _, err := localidentity.NewIdentity("user-01", "", localidentity.IdentityActive); !errors.Is(err, localidentity.ErrBlankTenantID) {
		t.Errorf("expected ErrBlankTenantID for empty tenant ID, got %v", err)
	}
	if _, err := localidentity.NewIdentity("user-01", "  \n ", localidentity.IdentityActive); !errors.Is(err, localidentity.ErrBlankTenantID) {
		t.Errorf("expected ErrBlankTenantID for whitespace tenant ID, got %v", err)
	}

	// Invalid State
	if _, err := localidentity.NewIdentity("user-01", "tenant-01", "UNKNOWN_STATE"); !errors.Is(err, localidentity.ErrInvalidState) {
		t.Errorf("expected ErrInvalidState, got %v", err)
	}
}

func TestIdentityManager_RegistrationAndState(t *testing.T) {
	clock := newTestClock(t)
	mgr := localidentity.NewIdentityManager(clock.Now)

	id, _ := localidentity.NewIdentity("user-01", "tenant-01", localidentity.IdentityActive)
	if err := mgr.RegisterIdentity(id); err != nil {
		t.Fatalf("unexpected RegisterIdentity error: %v", err)
	}

	// Duplicate registration
	if err := mgr.RegisterIdentity(id); !errors.Is(err, localidentity.ErrIdentityExists) {
		t.Errorf("expected ErrIdentityExists for duplicate, got %v", err)
	}

	// Disable identity
	if err := mgr.SetIdentityState("user-01", localidentity.IdentityDisabled); err != nil {
		t.Errorf("unexpected SetIdentityState error: %v", err)
	}

	// Non-existent subject
	if err := mgr.SetIdentityState("nonexistent", localidentity.IdentityActive); !errors.Is(err, localidentity.ErrIdentityNotFound) {
		t.Errorf("expected ErrIdentityNotFound, got %v", err)
	}

	// Invalid state transition
	if err := mgr.SetIdentityState("user-01", "INVALID"); !errors.Is(err, localidentity.ErrInvalidState) {
		t.Errorf("expected ErrInvalidState, got %v", err)
	}
}

func TestToken_IssuanceAndDigestSecurity(t *testing.T) {
	clock := newTestClock(t)
	mgr := localidentity.NewIdentityManager(clock.Now)

	id, _ := localidentity.NewIdentity("user-01", "tenant-alpha", localidentity.IdentityActive)
	_ = mgr.RegisterIdentity(id)

	rawToken, digest, err := mgr.IssueSession("user-01", 15*time.Minute)
	if err != nil {
		t.Fatalf("unexpected IssueSession error: %v", err)
	}

	// Format validation
	if !strings.HasPrefix(rawToken, localidentity.TokenPrefix) {
		t.Errorf("token %q does not start with prefix %q", rawToken, localidentity.TokenPrefix)
	}
	expectedLen := len(localidentity.TokenPrefix) + 64 // 32 bytes hex encoded = 64 chars
	if len(rawToken) != expectedLen {
		t.Errorf("token length = %d, want %d", len(rawToken), expectedLen)
	}

	// Digest correctness
	computedDigest := sha256.Sum256([]byte(rawToken))
	if digest != computedDigest {
		t.Errorf("returned digest does not match SHA-256 of raw token")
	}

	// Issue session for disabled identity should fail
	disID, _ := localidentity.NewIdentity("user-dis", "tenant-alpha", localidentity.IdentityDisabled)
	_ = mgr.RegisterIdentity(disID)
	if _, _, err := mgr.IssueSession("user-dis", 15*time.Minute); !errors.Is(err, localidentity.ErrIdentityDisabled) {
		t.Errorf("expected ErrIdentityDisabled when issuing session for disabled identity, got %v", err)
	}

	// Issue session for non-existent identity should fail
	if _, _, err := mgr.IssueSession("nonexistent", 15*time.Minute); !errors.Is(err, localidentity.ErrIdentityNotFound) {
		t.Errorf("expected ErrIdentityNotFound, got %v", err)
	}
}

func TestValidateSession_AnonymousOrMalformed(t *testing.T) {
	clock := newTestClock(t)
	mgr := localidentity.NewIdentityManager(clock.Now)

	malformedTokens := []struct {
		name  string
		token string
	}{
		{"empty string", ""},
		{"whitespace only", "   \t\n  "},
		{"missing prefix", "d23441a48e516b6c34aea4fa41551a30e30af8031234567890abcdef12345678"},
		{"wrong prefix", "bearer_tok_d23441a48e516b6c34aea4fa41551a30e30af8031234567890abcdef12345678"},
		{"truncated hex", "oshe_tok_d23441a48e516b6c"},
		{"invalid hex characters", "oshe_tok_zzzz41a48e516b6c34aea4fa41551a30e30af8031234567890abcdef12345678"},
		{"too long hex", "oshe_tok_d23441a48e516b6c34aea4fa41551a30e30af8031234567890abcdef12345678aaaa"},
	}

	for _, tc := range malformedTokens {
		t.Run(tc.name, func(t *testing.T) {
			_, err := mgr.ValidateSession(tc.token, "")
			if !errors.Is(err, localidentity.ErrMalformedToken) {
				t.Fatalf("expected ErrMalformedToken for %q, got %v", tc.name, err)
			}
		})
	}
}

func TestValidateSession_UnknownToken(t *testing.T) {
	clock := newTestClock(t)
	mgr := localidentity.NewIdentityManager(clock.Now)

	// Generate a syntactically valid token that was never registered in the manager
	unregisteredToken, _, _ := localidentity.GenerateOpaqueToken()

	_, err := mgr.ValidateSession(unregisteredToken, "")
	if !errors.Is(err, localidentity.ErrTokenNotFound) {
		t.Fatalf("expected ErrTokenNotFound, got %v", err)
	}
}

func TestValidateSession_ExpiredToken_InjectableClock(t *testing.T) {
	clock := newTestClock(t)
	mgr := localidentity.NewIdentityManager(clock.Now)

	id, _ := localidentity.NewIdentity("user-01", "tenant-alpha", localidentity.IdentityActive)
	_ = mgr.RegisterIdentity(id)

	ttl := 10 * time.Minute
	rawToken, _, err := mgr.IssueSession("user-01", ttl)
	if err != nil {
		t.Fatalf("unexpected IssueSession error: %v", err)
	}

	// Valid before expiry
	clock.Advance(5 * time.Minute)
	valID, err := mgr.ValidateSession(rawToken, "tenant-alpha")
	if err != nil {
		t.Fatalf("unexpected error at T+5m: %v", err)
	}
	if !valID.IsAuthenticated() {
		t.Errorf("expected IsAuthenticated == true")
	}

	// Exactly at expiration time
	clock.Advance(5 * time.Minute) // now T+10m
	_, err = mgr.ValidateSession(rawToken, "tenant-alpha")
	if !errors.Is(err, localidentity.ErrTokenExpired) {
		t.Fatalf("expected ErrTokenExpired at exact expiry, got %v", err)
	}

	// Past expiration time
	clock.Advance(1 * time.Minute) // now T+11m
	_, err = mgr.ValidateSession(rawToken, "tenant-alpha")
	if !errors.Is(err, localidentity.ErrTokenExpired) {
		t.Fatalf("expected ErrTokenExpired past expiry, got %v", err)
	}
}

func TestValidateSession_RevokedToken(t *testing.T) {
	clock := newTestClock(t)
	mgr := localidentity.NewIdentityManager(clock.Now)

	id, _ := localidentity.NewIdentity("user-01", "tenant-alpha", localidentity.IdentityActive)
	_ = mgr.RegisterIdentity(id)

	rawToken, _, _ := mgr.IssueSession("user-01", 30*time.Minute)

	// Valid initially
	if _, err := mgr.ValidateSession(rawToken, "tenant-alpha"); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}

	// Revoke by token
	if err := mgr.RevokeToken(rawToken); err != nil {
		t.Fatalf("unexpected RevokeToken error: %v", err)
	}

	// Validation must now fail closed
	if _, err := mgr.ValidateSession(rawToken, "tenant-alpha"); !errors.Is(err, localidentity.ErrTokenRevoked) {
		t.Fatalf("expected ErrTokenRevoked after revocation, got %v", err)
	}

	// Revoking nonexistent session fails closed
	var fakeDigest [32]byte
	if err := mgr.RevokeSession(fakeDigest); !errors.Is(err, localidentity.ErrTokenNotFound) {
		t.Errorf("expected ErrTokenNotFound for unknown digest revocation, got %v", err)
	}

	// Re-verifying digest method works
	rawToken2, dig2, _ := mgr.IssueSession("user-01", 30*time.Minute)
	if err := mgr.RevokeSession(dig2); err != nil {
		t.Fatalf("unexpected RevokeSession error: %v", err)
	}
	if _, err := mgr.ValidateSession(rawToken2, "tenant-alpha"); !errors.Is(err, localidentity.ErrTokenRevoked) {
		t.Fatalf("expected ErrTokenRevoked after digest revocation, got %v", err)
	}
}

func TestValidateSession_DisabledIdentity(t *testing.T) {
	clock := newTestClock(t)
	mgr := localidentity.NewIdentityManager(clock.Now)

	id, _ := localidentity.NewIdentity("user-01", "tenant-alpha", localidentity.IdentityActive)
	_ = mgr.RegisterIdentity(id)

	rawToken, _, _ := mgr.IssueSession("user-01", 30*time.Minute)

	// Initially valid
	if _, err := mgr.ValidateSession(rawToken, "tenant-alpha"); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}

	// Disable identity
	_ = mgr.SetIdentityState("user-01", localidentity.IdentityDisabled)

	// Must fail closed with ErrIdentityDisabled
	if _, err := mgr.ValidateSession(rawToken, "tenant-alpha"); !errors.Is(err, localidentity.ErrIdentityDisabled) {
		t.Fatalf("expected ErrIdentityDisabled after disabling identity, got %v", err)
	}

	// Re-enable identity -> validation passes again
	_ = mgr.SetIdentityState("user-01", localidentity.IdentityActive)
	if _, err := mgr.ValidateSession(rawToken, "tenant-alpha"); err != nil {
		t.Fatalf("expected validation success after re-enabling identity, got %v", err)
	}
}

func TestValidateSession_MismatchedTenant(t *testing.T) {
	clock := newTestClock(t)
	mgr := localidentity.NewIdentityManager(clock.Now)

	idA, _ := localidentity.NewIdentity("user-tenant-a", "tenant-alpha-001", localidentity.IdentityActive)
	_ = mgr.RegisterIdentity(idA)

	tokenA, _, _ := mgr.IssueSession("user-tenant-a", 30*time.Minute)

	// Validate against matching tenant -> OK
	if _, err := mgr.ValidateSession(tokenA, "tenant-alpha-001"); err != nil {
		t.Fatalf("expected success for matching tenant, got %v", err)
	}

	// Validate against mismatched tenant -> ErrTenantMismatch
	if _, err := mgr.ValidateSession(tokenA, "tenant-bravo-999"); !errors.Is(err, localidentity.ErrTenantMismatch) {
		t.Fatalf("expected ErrTenantMismatch when validating against mismatched tenant, got %v", err)
	}

	// Validate with empty expectedTenantID -> returns tenant without constraint
	valID, err := mgr.ValidateSession(tokenA, "")
	if err != nil {
		t.Fatalf("unexpected error when expectedTenantID is empty: %v", err)
	}
	if valID.TenantID() != "tenant-alpha-001" {
		t.Errorf("TenantID = %q, want tenant-alpha-001", valID.TenantID())
	}
}

func TestValidateSession_ValidAccepted(t *testing.T) {
	clock := newTestClock(t)
	mgr := localidentity.NewIdentityManager(clock.Now)

	subject := "auditor-01"
	tenantID := "tenant-acme-corp"
	id, _ := localidentity.NewIdentity(subject, tenantID, localidentity.IdentityActive)
	_ = mgr.RegisterIdentity(id)

	rawToken, digest, err := mgr.IssueSession(subject, 1*time.Hour)
	if err != nil {
		t.Fatalf("unexpected IssueSession error: %v", err)
	}

	clock.Advance(15 * time.Minute)

	valID, err := mgr.ValidateSession(rawToken, tenantID)
	if err != nil {
		t.Fatalf("unexpected ValidateSession error: %v", err)
	}

	if !valID.IsAuthenticated() {
		t.Errorf("IsAuthenticated() = false, want true")
	}
	if valID.Subject() != subject {
		t.Errorf("Subject() = %q, want %q", valID.Subject(), subject)
	}
	if valID.TenantID() != tenantID {
		t.Errorf("TenantID() = %q, want %q", valID.TenantID(), tenantID)
	}
	if valID.TokenDigest() != digest {
		t.Errorf("TokenDigest() mismatch")
	}
	if !valID.ValidatedAt().Equal(clock.Now()) {
		t.Errorf("ValidatedAt() = %v, want %v", valID.ValidatedAt(), clock.Now())
	}
}
