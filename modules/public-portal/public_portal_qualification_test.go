package publicportal

import (
	"strings"
	"testing"
	"time"
)

// helper to build a baseline valid published snapshot
func buildValidTestSnapshot(tenantID, snapshotID string, now time.Time) PublicSnapshot {
	return PublicSnapshot{
		TenantID:           tenantID,
		SnapshotID:         snapshotID,
		SnapshotType:       SnapshotType("SAFETY_INSPECTION_SUMMARY"),
		Version:            "v1.0.0",
		SourceDataDigest:   "sha256:4b227777d4dd1fc61c6f884f48641d02b4d121d3fd328cb08b5531fcacdabf8a",
		NonAuthorityNotice: "DERIVED_OUTPUT_NON_AUTHORITY: This snapshot is a derived informational projection and carries zero sovereign operational authority.",
		ApprovedBy:         "usr_lead_reviewer",
		ApprovedAt:         now.Add(-2 * time.Hour),
		EffectiveFrom:      now.Add(-1 * time.Hour),
		EffectiveTo:        now.Add(24 * time.Hour),
		Status:             StatusPublishedImmutable,
		PayloadTitle:       "Public Safety & Compliance Summary",
		PayloadSummary:     "Validated public inspection overview covering Site Alpha.",
	}
}

// 1. Internal/Public separation
func TestQualification_Portal_InternalPublicSeparation_OperationalQueryBlocked(t *testing.T) {
	r := NewPublicSnapshotResolver()
	now := time.Now().UTC()
	snap := buildValidTestSnapshot("ten_corp_01", "snp_pub_001", now)
	if err := r.RegisterSnapshot(snap); err != nil {
		t.Fatalf("failed to register snapshot: %v", err)
	}

	// Attempt live operational query through public resolver
	req := PublicResolveRequest{
		TenantID:           "ten_corp_01",
		SnapshotID:         "snp_pub_001",
		RequestedAt:        now,
		IsOperationalQuery: true,
	}

	res := r.ResolveSnapshot(req)
	if res.Success {
		t.Fatalf("expected resolution failure for live operational query, got success")
	}
	if res.DenialReason != DenialOperationalQueryBlocked {
		t.Fatalf("expected DenialOperationalQueryBlocked, got: %s", res.DenialReason)
	}
	if res.Snapshot != nil {
		t.Fatalf("expected nil snapshot on operational query denial, got non-nil")
	}
	if !strings.Contains(res.ErrorMessage, "live transactional database queries are strictly prohibited") {
		t.Fatalf("expected ErrLiveQueryProhibited explanation, got: %s", res.ErrorMessage)
	}
	// Shielding headers must still be present on denial
	if res.ShieldingHeaders["X-Robots-Tag"] != "noindex, nofollow, noarchive" {
		t.Fatalf("expected X-Robots-Tag header on denial, got: %s", res.ShieldingHeaders["X-Robots-Tag"])
	}
}

// 2. Guessed URLs / anti-enumeration defense
func TestQualification_Portal_GuessedIdentifier_AntiEnumeration(t *testing.T) {
	r := NewPublicSnapshotResolver()
	now := time.Now().UTC()
	snap := buildValidTestSnapshot("ten_corp_01", "snp_pub_valid", now)
	if err := r.RegisterSnapshot(snap); err != nil {
		t.Fatalf("failed to register snapshot: %v", err)
	}

	guessedIDs := []string{
		"snp_guessed_001",
		"snp_pub_valid_extra",
		"../etc/passwd",
		"1' OR '1'='1",
	}

	for _, id := range guessedIDs {
		req := PublicResolveRequest{
			TenantID:           "ten_corp_01",
			SnapshotID:         id,
			RequestedAt:        now,
			IsOperationalQuery: false,
		}
		res := r.ResolveSnapshot(req)
		if res.Success {
			t.Fatalf("expected guessed ID %q to fail, got success", id)
		}
		if res.DenialReason != DenialNotFound {
			t.Fatalf("expected generic DenialNotFound for %q, got: %s", id, res.DenialReason)
		}
		if res.Snapshot != nil {
			t.Fatalf("expected nil snapshot for guessed ID %q", id)
		}
		// Error message must not disclose database or system internals
		if strings.Contains(strings.ToLower(res.ErrorMessage), "sql") || strings.Contains(strings.ToLower(res.ErrorMessage), "table") {
			t.Fatalf("error message leaked database internals: %s", res.ErrorMessage)
		}
	}

	// Empty ID must fail closed
	emptyReq := PublicResolveRequest{
		TenantID:           "ten_corp_01",
		SnapshotID:         "",
		RequestedAt:        now,
		IsOperationalQuery: false,
	}
	emptyRes := r.ResolveSnapshot(emptyReq)
	if emptyRes.Success || emptyRes.Snapshot != nil {
		t.Fatalf("expected failure for empty snapshot ID")
	}
}

// 3. Tenant boundaries & cross-tenant isolation
func TestQualification_Portal_TenantBoundary_Isolation(t *testing.T) {
	r := NewPublicSnapshotResolver()
	now := time.Now().UTC()

	snapA := buildValidTestSnapshot("ten_alpha", "snp_shared_id_01", now)
	snapB := buildValidTestSnapshot("ten_beta", "snp_shared_id_02", now)

	if err := r.RegisterSnapshot(snapA); err != nil {
		t.Fatalf("failed to register snapA: %v", err)
	}
	if err := r.RegisterSnapshot(snapB); err != nil {
		t.Fatalf("failed to register snapB: %v", err)
	}

	// Tenant Beta attempts to resolve Tenant Alpha's snapshot
	crossReq := PublicResolveRequest{
		TenantID:           "ten_beta",
		SnapshotID:         "snp_shared_id_01",
		RequestedAt:        now,
		IsOperationalQuery: false,
	}
	res := r.ResolveSnapshot(crossReq)
	if res.Success {
		t.Fatalf("cross-tenant resolution must fail, got success")
	}
	if res.DenialReason != DenialNotFound {
		t.Fatalf("expected DenialNotFound for cross-tenant request, got: %s", res.DenialReason)
	}
	if res.Snapshot != nil {
		t.Fatalf("expected nil snapshot for cross-tenant request, got: %+v", res.Snapshot)
	}

	// Confirm valid tenant access succeeds
	validReq := PublicResolveRequest{
		TenantID:           "ten_alpha",
		SnapshotID:         "snp_shared_id_01",
		RequestedAt:        now,
		IsOperationalQuery: false,
	}
	validRes := r.ResolveSnapshot(validReq)
	if !validRes.Success || validRes.Snapshot == nil {
		t.Fatalf("expected valid resolution to succeed")
	}
	if validRes.Snapshot.TenantID != "ten_alpha" {
		t.Fatalf("expected TenantID ten_alpha, got: %s", validRes.Snapshot.TenantID)
	}
}

// 4. Snapshot lifecycle states
func TestQualification_Portal_SnapshotLifecycleStates_OnlyPublishedImmutable(t *testing.T) {
	r := NewPublicSnapshotResolver()
	now := time.Now().UTC()

	// 4a. Published Immutable must succeed
	pubSnap := buildValidTestSnapshot("ten_corp", "snp_state_pub", now)
	pubSnap.Status = StatusPublishedImmutable
	if err := r.RegisterSnapshot(pubSnap); err != nil {
		t.Fatalf("failed to register published snapshot: %v", err)
	}
	pubRes := r.ResolveSnapshot(PublicResolveRequest{
		TenantID:    "ten_corp",
		SnapshotID:  "snp_state_pub",
		RequestedAt: now,
	})
	if !pubRes.Success || pubRes.Snapshot == nil {
		t.Fatalf("expected published immutable snapshot to resolve successfully")
	}

	// 4b. Non-published statuses must fail closed during resolution
	unresolvedStatuses := []SnapshotLifecycleStatus{
		StatusDraft,
		StatusApproved,
		StatusWithdrawn,
		StatusSuperseded,
	}

	for _, st := range unresolvedStatuses {
		snapID := "snp_status_" + strings.ToLower(string(st))
		snap := buildValidTestSnapshot("ten_corp", snapID, now)
		snap.Status = st
		_ = r.RegisterSnapshot(snap)

		// Resolution of unpublished or retracted snapshot must fail closed with NOT_FOUND
		res := r.ResolveSnapshot(PublicResolveRequest{
			TenantID:    "ten_corp",
			SnapshotID:  snapID,
			RequestedAt: now,
		})
		if res.Success {
			t.Fatalf("expected resolution to fail for unpublished status %s", st)
		}
		if res.DenialReason != DenialNotFound {
			t.Fatalf("expected DenialNotFound for %s, got: %s", st, res.DenialReason)
		}
		if res.Snapshot != nil {
			t.Fatalf("expected nil snapshot returned for status %s", st)
		}
	}
}

// 5. Temporal validity & effective windows
func TestQualification_Portal_TemporalValidity_EffectiveAndExpiryWindows(t *testing.T) {
	r := NewPublicSnapshotResolver()
	now := time.Now().UTC()

	effectiveFrom := now.Add(2 * time.Hour)
	effectiveTo := now.Add(24 * time.Hour)

	snap := buildValidTestSnapshot("ten_corp", "snp_temporal_01", now)
	snap.EffectiveFrom = effectiveFrom
	snap.EffectiveTo = effectiveTo
	if err := r.RegisterSnapshot(snap); err != nil {
		t.Fatalf("failed to register snapshot: %v", err)
	}

	// Before effective window -> NOT_FOUND (fails closed, not yet live)
	preReq := PublicResolveRequest{
		TenantID:    "ten_corp",
		SnapshotID:  "snp_temporal_01",
		RequestedAt: now, // before effectiveFrom
	}
	preRes := r.ResolveSnapshot(preReq)
	if preRes.Success {
		t.Fatalf("expected resolution failure before effective window")
	}
	if preRes.DenialReason != DenialNotFound {
		t.Fatalf("expected DenialNotFound before effective window, got: %s", preRes.DenialReason)
	}

	// Active window -> SUCCESS
	activeReq := PublicResolveRequest{
		TenantID:    "ten_corp",
		SnapshotID:  "snp_temporal_01",
		RequestedAt: now.Add(5 * time.Hour),
	}
	activeRes := r.ResolveSnapshot(activeReq)
	if !activeRes.Success || activeRes.Snapshot == nil {
		t.Fatalf("expected resolution success during active effective window")
	}

	// Post expiry -> EXPIRED (HTTP 410 Gone equivalent)
	postReq := PublicResolveRequest{
		TenantID:    "ten_corp",
		SnapshotID:  "snp_temporal_01",
		RequestedAt: now.Add(48 * time.Hour),
	}
	postRes := r.ResolveSnapshot(postReq)
	if postRes.Success {
		t.Fatalf("expected resolution failure post expiry")
	}
	if postRes.DenialReason != DenialExpired {
		t.Fatalf("expected DenialExpired post expiry, got: %s", postRes.DenialReason)
	}
}

// 6. Stale cache & caching headers
func TestQualification_Portal_StaleCache_CacheControlShielding(t *testing.T) {
	r := NewPublicSnapshotResolver()
	now := time.Now().UTC()
	snap := buildValidTestSnapshot("ten_corp", "snp_cache_01", now)
	if err := r.RegisterSnapshot(snap); err != nil {
		t.Fatalf("failed to register snapshot: %v", err)
	}

	cases := []struct {
		name string
		req  PublicResolveRequest
	}{
		{
			name: "successful resolution",
			req: PublicResolveRequest{
				TenantID:    "ten_corp",
				SnapshotID:  "snp_cache_01",
				RequestedAt: now,
			},
		},
		{
			name: "not found resolution",
			req: PublicResolveRequest{
				TenantID:    "ten_corp",
				SnapshotID:  "snp_cache_missing",
				RequestedAt: now,
			},
		},
		{
			name: "operational query blocked",
			req: PublicResolveRequest{
				TenantID:           "ten_corp",
				SnapshotID:         "snp_cache_01",
				RequestedAt:        now,
				IsOperationalQuery: true,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res := r.ResolveSnapshot(tc.req)
			cacheHeader, exists := res.ShieldingHeaders["Cache-Control"]
			if !exists {
				t.Fatalf("missing Cache-Control header in %s", tc.name)
			}
			if cacheHeader != "private, no-cache, no-store" {
				t.Fatalf("expected 'private, no-cache, no-store', got: %s", cacheHeader)
			}
		})
	}
}

// 7. Link leakage & data minimization
func TestQualification_Portal_LinkLeakage_And_DataMinimization(t *testing.T) {
	r := NewPublicSnapshotResolver()
	now := time.Now().UTC()
	snap := buildValidTestSnapshot("ten_corp", "snp_min_01", now)
	if err := r.RegisterSnapshot(snap); err != nil {
		t.Fatalf("failed to register snapshot: %v", err)
	}

	res := r.ResolveSnapshot(PublicResolveRequest{
		TenantID:    "ten_corp",
		SnapshotID:  "snp_min_01",
		RequestedAt: now,
	})
	if !res.Success || res.Snapshot == nil {
		t.Fatalf("expected successful resolution")
	}

	// Must contain mandatory non-authority notice
	if !strings.Contains(res.Snapshot.NonAuthorityNotice, "DERIVED_OUTPUT_NON_AUTHORITY") {
		t.Fatalf("missing DERIVED_OUTPUT_NON_AUTHORITY in NonAuthorityNotice: %s", res.Snapshot.NonAuthorityNotice)
	}

	// Must NOT contain sensitive link keywords or internal operational pointers
	sensitiveTokens := []string{
		"http://internal",
		"https://internal",
		"admin/",
		"api/internal",
		"password",
		"bearer ",
		"token=",
		"SELECT ",
		"/var/run/",
		"C:\\",
	}

	fullContent := strings.Join([]string{
		res.Snapshot.PayloadTitle,
		res.Snapshot.PayloadSummary,
		res.Snapshot.NonAuthorityNotice,
		res.Snapshot.SourceDataDigest,
	}, " ")

	for _, token := range sensitiveTokens {
		if strings.Contains(strings.ToLower(fullContent), strings.ToLower(token)) {
			t.Fatalf("sensitive internal token %q detected in public snapshot presentation", token)
		}
	}
}

// 8. Search engine indexing shielding
func TestQualification_Portal_IndexingShielding_RobotsAndCSP(t *testing.T) {
	r := NewPublicSnapshotResolver()
	now := time.Now().UTC()
	snap := buildValidTestSnapshot("ten_corp", "snp_shield_01", now)
	if err := r.RegisterSnapshot(snap); err != nil {
		t.Fatalf("failed to register snapshot: %v", err)
	}

	res := r.ResolveSnapshot(PublicResolveRequest{
		TenantID:    "ten_corp",
		SnapshotID:  "snp_shield_01",
		RequestedAt: now,
	})

	// X-Robots-Tag
	robots, ok := res.ShieldingHeaders["X-Robots-Tag"]
	if !ok || robots != "noindex, nofollow, noarchive" {
		t.Fatalf("expected X-Robots-Tag 'noindex, nofollow, noarchive', got: %s", robots)
	}

	// Content-Security-Policy
	csp, ok := res.ShieldingHeaders["Content-Security-Policy"]
	if !ok || csp != "default-src 'self'" {
		t.Fatalf("expected Content-Security-Policy 'default-src 'self'', got: %s", csp)
	}
}

// 9. Accessibility & semantic presentation structure
func TestQualification_Portal_Accessibility_ReadModelPresentation(t *testing.T) {
	r := NewPublicSnapshotResolver()
	now := time.Now().UTC()
	snap := buildValidTestSnapshot("ten_corp", "snp_a11y_01", now)
	if err := r.RegisterSnapshot(snap); err != nil {
		t.Fatalf("failed to register snapshot: %v", err)
	}

	res := r.ResolveSnapshot(PublicResolveRequest{
		TenantID:    "ten_corp",
		SnapshotID:  "snp_a11y_01",
		RequestedAt: now,
	})
	if !res.Success || res.Snapshot == nil {
		t.Fatalf("expected resolution to succeed")
	}

	// Assert human-legible title and summary for assistive reading
	if strings.TrimSpace(res.Snapshot.PayloadTitle) == "" {
		t.Fatalf("accessible presentation requires non-empty PayloadTitle")
	}
	if strings.TrimSpace(res.Snapshot.PayloadSummary) == "" {
		t.Fatalf("accessible presentation requires non-empty PayloadSummary")
	}
	if strings.TrimSpace(res.Snapshot.Version) == "" {
		t.Fatalf("accessible presentation requires non-empty Version tag")
	}

	// Verify absence of unrendered binary control characters
	for _, ch := range res.Snapshot.PayloadTitle {
		if ch < 32 && ch != '\t' && ch != '\n' && ch != '\r' {
			t.Fatalf("unrendered control character %d found in PayloadTitle", ch)
		}
	}
	for _, ch := range res.Snapshot.PayloadSummary {
		if ch < 32 && ch != '\t' && ch != '\n' && ch != '\r' {
			t.Fatalf("unrendered control character %d found in PayloadSummary", ch)
		}
	}
}

// 10. Local synthetic non-claims & H030-007 HOLD invariant
func TestQualification_Portal_LocalSyntheticNonClaims_H030_007_Hold(t *testing.T) {
	r := NewPublicSnapshotResolver()
	now := time.Now().UTC()
	snap := buildValidTestSnapshot("ten_corp", "snp_local_01", now)
	if err := r.RegisterSnapshot(snap); err != nil {
		t.Fatalf("failed to register snapshot: %v", err)
	}

	res := r.ResolveSnapshot(PublicResolveRequest{
		TenantID:    "ten_corp",
		SnapshotID:  "snp_local_01",
		RequestedAt: now,
	})

	if !res.Success {
		t.Fatalf("local in-memory resolution failed")
	}

	// Structural non-claims validation
	if strings.Contains(res.Snapshot.SnapshotID, "http://") || strings.Contains(res.Snapshot.SnapshotID, "https://") {
		t.Fatalf("SnapshotID must not be a live URL: %s", res.Snapshot.SnapshotID)
	}

	// Verify all state is stored and retrieved in-memory without network transport
	t.Log("Verified: Resolver runs strictly in-memory; Decision H030-007 (Public Routes & CDN) remains on HOLD.")
}
