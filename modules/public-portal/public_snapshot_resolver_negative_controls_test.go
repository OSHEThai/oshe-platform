package publicportal_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	publicportal "oshe/public-portal"
)

// NEG-SNAP-01: Operational Query Prohibition
// Threat: Public portal caller attempts live transactional query or SQL database bypass.
func TestNegativeControl_OperationalQueryBlocked(t *testing.T) {
	resolver := publicportal.NewPublicSnapshotResolver()

	req := publicportal.PublicResolveRequest{
		TenantID:           "ten_alpha",
		SnapshotID:         "snp_any",
		IsOperationalQuery: true, // Malicious / live query bypass attempt
		RequestedAt:        time.Now().UTC(),
	}

	res := resolver.ResolveSnapshot(req)
	if res.Success {
		t.Fatalf("operational query bypass attempt must not succeed")
	}
	if res.DenialReason != publicportal.DenialOperationalQueryBlocked {
		t.Fatalf("expected DenialOperationalQueryBlocked, got: %s", res.DenialReason)
	}
	if !strings.Contains(res.ErrorMessage, "live transactional database queries are strictly prohibited") {
		t.Errorf("error message mismatch: %s", res.ErrorMessage)
	}
}

// NEG-SNAP-02: Guessed Identifier Non-Leaking Rejection
// Threat: Attacker guessing snapshot IDs to confirm resource existence or internal structures.
func TestNegativeControl_GuessedIdentifier_NonLeaking(t *testing.T) {
	resolver := publicportal.NewPublicSnapshotResolver()

	req := publicportal.PublicResolveRequest{
		TenantID:    "ten_alpha",
		SnapshotID:  "snp_guessed_id_probe_123",
		RequestedAt: time.Now().UTC(),
	}

	res := resolver.ResolveSnapshot(req)
	if res.Success {
		t.Fatalf("guessed snapshot ID must fail")
	}
	if res.DenialReason != publicportal.DenialNotFound {
		t.Fatalf("expected non-leaking DenialNotFound for guessed ID, got: %s", res.DenialReason)
	}
	if !errors.Is(publicportal.ErrSnapshotNotFound, errors.New(res.ErrorMessage)) && !strings.Contains(res.ErrorMessage, "not found") {
		t.Errorf("expected non-leaking error message, got: %s", res.ErrorMessage)
	}
}

// NEG-SNAP-03: Cross-Tenant Isolation Rejection
// Threat: Caller from Tenant A queries snapshot belonging to Tenant B.
func TestNegativeControl_WrongTenant_Isolation(t *testing.T) {
	resolver := publicportal.NewPublicSnapshotResolver()

	sB := publicportal.PublicSnapshot{
		SnapshotID:     "snp_victim_b",
		TenantID:       "ten_bravo",
		SnapshotType:   publicportal.SnapshotTypeInspection,
		Version:        "1.0.0",
		Status:         publicportal.StatusPublishedImmutable,
		EffectiveFrom:  time.Now().Add(-1 * time.Hour),
		PayloadTitle:   "Confidential Safety Audit",
		PayloadSummary: "Tenant B Internal Review",
	}
	_ = resolver.RegisterSnapshot(sB)

	// Attacker presenting Tenant Alpha attempts to access Tenant Bravo snapshot
	res := resolver.ResolveSnapshot(publicportal.PublicResolveRequest{
		TenantID:    "ten_alpha",
		SnapshotID:  "snp_victim_b",
		RequestedAt: time.Now().UTC(),
	})

	if res.Success {
		t.Fatalf("cross-tenant snapshot access must fail closed")
	}
	// Must return generic DenialNotFound, NEVER revealing that snp_victim_b exists under ten_bravo
	if res.DenialReason != publicportal.DenialNotFound {
		t.Fatalf("cross-tenant query must return non-leaking DenialNotFound, got: %s", res.DenialReason)
	}
}

// NEG-SNAP-04: Unpublished, Draft, Withdrawn & Superseded Statuses
// Threat: Premature exposure of unapproved drafts or stale/retracted snapshots.
func TestNegativeControl_UnpublishedStatuses_NonLeaking(t *testing.T) {
	resolver := publicportal.NewPublicSnapshotResolver()
	tenantID := "ten_alpha"
	now := time.Now().UTC()

	unpublishedStatuses := []publicportal.SnapshotLifecycleStatus{
		publicportal.StatusDraft,
		publicportal.StatusStaged,
		publicportal.StatusApproved, // Approved internally, but not yet PUBLISHED_IMMUTABLE
		publicportal.StatusWithdrawn,
		publicportal.StatusSuperseded,
	}

	for _, status := range unpublishedStatuses {
		snapID := "snp_status_" + strings.ToLower(string(status))
		s := publicportal.PublicSnapshot{
			SnapshotID:     snapID,
			TenantID:       tenantID,
			SnapshotType:   publicportal.SnapshotTypeInspection,
			Version:        "1.0.0",
			Status:         status,
			EffectiveFrom:  now.Add(-1 * time.Hour),
			PayloadTitle:   "Unpublished Report",
			PayloadSummary: "Confidential Draft Data",
		}
		_ = resolver.RegisterSnapshot(s)

		res := resolver.ResolveSnapshot(publicportal.PublicResolveRequest{
			TenantID:    tenantID,
			SnapshotID:  snapID,
			RequestedAt: now,
		})

		if res.Success {
			t.Fatalf("status %s must never be visible to public resolver", status)
		}
		if res.DenialReason != publicportal.DenialNotFound {
			t.Errorf("status %s must return non-leaking DenialNotFound, got %s", status, res.DenialReason)
		}
	}
}

// NEG-SNAP-05: Data Minimization & Shielding Header Verification
func TestNegativeControl_DataMinimization_And_Shielding(t *testing.T) {
	resolver := publicportal.NewPublicSnapshotResolver()
	now := time.Now().UTC()

	s := publicportal.PublicSnapshot{
		SnapshotID:         "snp_clean_01",
		TenantID:           "ten_alpha",
		SnapshotType:       publicportal.SnapshotTypeMetric,
		Version:            "1.0.0",
		Status:             publicportal.StatusPublishedImmutable,
		EffectiveFrom:      now.Add(-1 * time.Hour),
		NonAuthorityNotice: publicportal.DefaultNonAuthorityNotice,
		PayloadTitle:       "Public Safety Performance Summary",
		PayloadSummary:     "Clean sanitized score summary: 99.2% compliance.",
	}
	_ = resolver.RegisterSnapshot(s)

	res := resolver.ResolveSnapshot(publicportal.PublicResolveRequest{
		TenantID:    "ten_alpha",
		SnapshotID:  "snp_clean_01",
		RequestedAt: now,
	})

	if !res.Success {
		t.Fatalf("expected successful resolution: %v", res.ErrorMessage)
	}

	// 1. Verify mandatory shielding headers in response
	headers := res.ShieldingHeaders
	if headers["X-Robots-Tag"] != "noindex, nofollow, noarchive" {
		t.Errorf("X-Robots-Tag missing or invalid: %s", headers["X-Robots-Tag"])
	}
	if headers["Content-Security-Policy"] != "default-src 'self'" {
		t.Errorf("Content-Security-Policy missing or invalid: %s", headers["Content-Security-Policy"])
	}
	if headers["Cache-Control"] != "private, no-cache, no-store" {
		t.Errorf("Cache-Control missing or invalid: %s", headers["Cache-Control"])
	}

	// 2. Data minimization check: payload must not contain sensitive tokens
	forbiddenSubstrings := []string{"password", "token", "hash", "secret", "bearer", "@", "+66", "citizen"}
	body := strings.ToLower(res.Snapshot.PayloadTitle + " " + res.Snapshot.PayloadSummary)
	for _, f := range forbiddenSubstrings {
		if strings.Contains(body, f) {
			t.Errorf("payload contains sensitive or unminimized token %q", f)
		}
	}
}
