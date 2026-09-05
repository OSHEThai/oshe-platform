package publicportal

import (
	"errors"
	"testing"
	"time"
)

func TestPublicSnapshot_CreationAndAccessors(t *testing.T) {
	baseTime := time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC)
	from := baseTime
	to := baseTime.Add(30 * 24 * time.Hour)

	s := PublicSnapshot{
		SnapshotID:         "snp_alpha_01",
		TenantID:           "ten_alpha",
		SnapshotType:       SnapshotTypeInspection,
		Version:            "1.0.0",
		SourceDataDigest:   "a1b2c3d4e5f6071829304a5b6c7d8e9f0123456789abcdef0123456789abcdef",
		NonAuthorityNotice: DefaultNonAuthorityNotice,
		ApprovedBy:         "usr_compliance_lead",
		ApprovedAt:         baseTime,
		EffectiveFrom:      from,
		EffectiveTo:        to,
		Status:             StatusPublishedImmutable,
		PayloadTitle:       "Monthly Site Safety Report Summary",
		PayloadSummary:     "All safety checkpoints passed with zero critical findings.",
	}

	resolver := NewPublicSnapshotResolver()
	if err := resolver.RegisterSnapshot(s); err != nil {
		t.Fatalf("unexpected RegisterSnapshot error: %v", err)
	}

	// Resolve snapshot
	res := resolver.ResolveSnapshot(PublicResolveRequest{
		TenantID:    "ten_alpha",
		SnapshotID:  "snp_alpha_01",
		RequestedAt: baseTime.Add(5 * 24 * time.Hour),
	})

	if !res.Success {
		t.Fatalf("expected successful resolution, got failure: %s (%s)", res.DenialReason, res.ErrorMessage)
	}
	if res.Snapshot == nil || res.Snapshot.SnapshotID != "snp_alpha_01" {
		t.Fatalf("resolved snapshot mismatch")
	}
	if res.Snapshot.Status != StatusPublishedImmutable {
		t.Errorf("expected StatusPublishedImmutable")
	}

	// Verify shielding headers
	headers := res.ShieldingHeaders
	if headers["X-Robots-Tag"] != "noindex, nofollow, noarchive" {
		t.Errorf("missing or incorrect X-Robots-Tag: %s", headers["X-Robots-Tag"])
	}
	if headers["Cache-Control"] != "private, no-cache, no-store" {
		t.Errorf("missing or incorrect Cache-Control: %s", headers["Cache-Control"])
	}
}

func TestResolveSnapshot_EffectiveWindowChecks(t *testing.T) {
	resolver := NewPublicSnapshotResolver()
	baseTime := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	from := baseTime
	to := baseTime.Add(7 * 24 * time.Hour)

	s := PublicSnapshot{
		SnapshotID:       "snp_temporal",
		TenantID:         "ten_alpha",
		SnapshotType:     SnapshotTypeMetric,
		Version:          "1.0.0",
		Status:           StatusPublishedImmutable,
		EffectiveFrom:    from,
		EffectiveTo:      to,
		PayloadTitle:     "Quarterly Metrics",
		PayloadSummary:   "Safety compliance 98.4%",
	}
	_ = resolver.RegisterSnapshot(s)

	// 1. Before effective window -> returns DenialNotFound
	before := baseTime.Add(-1 * time.Hour)
	resBefore := resolver.ResolveSnapshot(PublicResolveRequest{
		TenantID:    "ten_alpha",
		SnapshotID:  "snp_temporal",
		RequestedAt: before,
	})
	if resBefore.Success || resBefore.DenialReason != DenialNotFound {
		t.Errorf("expected DenialNotFound before effectiveFrom, got success=%v reason=%s", resBefore.Success, resBefore.DenialReason)
	}

	// 2. During window -> success
	during := baseTime.Add(3 * 24 * time.Hour)
	resDuring := resolver.ResolveSnapshot(PublicResolveRequest{
		TenantID:    "ten_alpha",
		SnapshotID:  "snp_temporal",
		RequestedAt: during,
	})
	if !resDuring.Success {
		t.Errorf("expected success during window")
	}

	// 3. After effective window -> returns DenialExpired
	after := to.Add(1 * time.Hour)
	resAfter := resolver.ResolveSnapshot(PublicResolveRequest{
		TenantID:    "ten_alpha",
		SnapshotID:  "snp_temporal",
		RequestedAt: after,
	})
	if resAfter.Success || resAfter.DenialReason != DenialExpired {
		t.Errorf("expected DenialExpired after effectiveTo, got success=%v reason=%s", resAfter.Success, resAfter.DenialReason)
	}
}

func TestResolveSnapshot_RegistrationRejections(t *testing.T) {
	resolver := NewPublicSnapshotResolver()

	// Blank tenant ID
	s1 := PublicSnapshot{SnapshotID: "snp_01"}
	if err := resolver.RegisterSnapshot(s1); !errors.Is(err, ErrBlankTenantID) {
		t.Errorf("expected ErrBlankTenantID, got %v", err)
	}

	// Blank snapshot ID
	s2 := PublicSnapshot{TenantID: "ten_01"}
	if err := resolver.RegisterSnapshot(s2); !errors.Is(err, ErrBlankSnapshotID) {
		t.Errorf("expected ErrBlankSnapshotID, got %v", err)
	}

	// Duplicate registration rejected
	sValid := PublicSnapshot{
		SnapshotID: "snp_dup",
		TenantID:   "ten_01",
		Status:     StatusPublishedImmutable,
	}
	if err := resolver.RegisterSnapshot(sValid); err != nil {
		t.Fatalf("first registration failed: %v", err)
	}
	if err := resolver.RegisterSnapshot(sValid); !errors.Is(err, ErrDuplicateSnapshotID) {
		t.Errorf("expected ErrDuplicateSnapshotID on duplicate, got %v", err)
	}
}
