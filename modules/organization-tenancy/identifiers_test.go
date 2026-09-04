package orgtenancy_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	orgtenancy "github.com/oshethai/oshe-platform/modules/organization-tenancy"
)

const (
	validSHA256 = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	diffSHA256  = "a591a6d40bf420404a011733cfb7b190d62c65bf0bcda32b57b277d9ad9f146e"
)

func TestCanonicalID_Validation(t *testing.T) {
	// Valid IDs
	validCases := []struct {
		id     string
		prefix string
	}{
		{"ten_12345678", orgtenancy.PrefixTenant},
		{"cmp_alpha_corp", orgtenancy.PrefixCompany},
		{"ste_bangkok_01", orgtenancy.PrefixSite},
		{"usr_somchai_01", orgtenancy.PrefixUser},
		{"corr_9876543210ab", orgtenancy.PrefixCorrelation},
		{"caus_fedcba098765", orgtenancy.PrefixCausation},
		{"idem_req_abc12345", orgtenancy.PrefixIdempotency},
	}

	for _, tc := range validCases {
		if err := orgtenancy.ValidateCanonicalID(tc.id, tc.prefix); err != nil {
			t.Errorf("expected valid ID for %s, got: %v", tc.id, err)
		}
	}

	// Invalid IDs
	invalidCases := []struct {
		name        string
		id          string
		prefix      string
		expectedErr error
	}{
		{"blank", "   ", orgtenancy.PrefixTenant, orgtenancy.ErrBlankIdentifier},
		{"no underscore", "ten12345678", orgtenancy.PrefixTenant, orgtenancy.ErrMalformedIdentifier},
		{"wrong prefix", "cmp_12345678", orgtenancy.PrefixTenant, orgtenancy.ErrPrefixMismatch},
		{"token too short", "ten_short", orgtenancy.PrefixTenant, orgtenancy.ErrMalformedIdentifier},
		{"uppercase characters", "ten_UPPERCASE12", orgtenancy.PrefixTenant, orgtenancy.ErrInvalidCharacters},
		{"illegal special chars", "ten_abc@def!", orgtenancy.PrefixTenant, orgtenancy.ErrInvalidCharacters},
		{"trailing underscore", "ten_", orgtenancy.PrefixTenant, orgtenancy.ErrMalformedIdentifier},
	}

	for _, tc := range invalidCases {
		t.Run(tc.name, func(t *testing.T) {
			err := orgtenancy.ValidateCanonicalID(tc.id, tc.prefix)
			if err == nil {
				t.Fatalf("expected error for %s, got nil", tc.name)
			}
			if !errors.Is(err, tc.expectedErr) {
				t.Errorf("expected error %v, got: %v", tc.expectedErr, err)
			}
		})
	}
}

func TestCanonicalID_GenerationAndCollisionResistance(t *testing.T) {
	seen := make(map[string]bool)
	const count = 500

	for i := 0; i < count; i++ {
		id, err := orgtenancy.GenerateCanonicalID(orgtenancy.PrefixTenant)
		if err != nil {
			t.Fatalf("GenerateCanonicalID failed: %v", err)
		}
		if err := orgtenancy.ValidateCanonicalID(id, orgtenancy.PrefixTenant); err != nil {
			t.Fatalf("generated ID %q failed validation: %v", id, err)
		}
		if seen[id] {
			t.Fatalf("collision detected on generated ID: %s", id)
		}
		seen[id] = true
	}
}

func TestTrackingContext_LifecycleAndSerialization(t *testing.T) {
	ctx, err := orgtenancy.GenerateTrackingContext()
	if err != nil {
		t.Fatalf("GenerateTrackingContext failed: %v", err)
	}

	if err := orgtenancy.ValidateCanonicalID(ctx.CorrelationID, orgtenancy.PrefixCorrelation); err != nil {
		t.Errorf("invalid generated correlation ID: %v", err)
	}
	if err := orgtenancy.ValidateCanonicalID(ctx.CausationID, orgtenancy.PrefixCausation); err != nil {
		t.Errorf("invalid generated causation ID: %v", err)
	}

	// JSON round trip
	b, err := json.Marshal(ctx)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded orgtenancy.TrackingContext
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if decoded.CorrelationID != ctx.CorrelationID || decoded.CausationID != ctx.CausationID {
		t.Errorf("decoded context does not match original: %+v vs %+v", decoded, ctx)
	}
}

func TestIdempotencyStore_IdempotentCreatesAndCollisionDetection(t *testing.T) {
	store := orgtenancy.NewIdempotencyStore()
	tenantID := "ten_alpha"
	key := "idem_order_001"

	createCallCount := 0
	createFn := func() (string, error) {
		createCallCount++
		return "res_inspection_12345", nil
	}

	// 1. Initial creation
	resID1, isReplay, err := store.EvaluateOrCreate(tenantID, key, validSHA256, createFn)
	if err != nil {
		t.Fatalf("initial create failed: %v", err)
	}
	if isReplay {
		t.Errorf("expected isReplay = false for first call")
	}
	if resID1 != "res_inspection_12345" || createCallCount != 1 {
		t.Errorf("unexpected create result: resID=%s, count=%d", resID1, createCallCount)
	}

	// 2. Duplicate create with identical payload (safe idempotent replay)
	resID2, isReplay, err := store.EvaluateOrCreate(tenantID, key, validSHA256, createFn)
	if err != nil {
		t.Fatalf("duplicate create failed: %v", err)
	}
	if !isReplay {
		t.Errorf("expected isReplay = true for identical replay")
	}
	if resID2 != resID1 || createCallCount != 1 {
		t.Errorf("createFn should NOT be called again: resID=%s, count=%d", resID2, createCallCount)
	}

	// 3. Duplicate create with differing payload (collision conflict rejection)
	_, _, err = store.EvaluateOrCreate(tenantID, key, diffSHA256, createFn)
	if !errors.Is(err, orgtenancy.ErrIdempotencyConflict) {
		t.Fatalf("expected ErrIdempotencyConflict on payload mismatch, got: %v", err)
	}

	// 4. Multi-tenant isolation: same key in different tenant succeeds independently
	resTenantBravo, isReplayBravo, err := store.EvaluateOrCreate("ten_bravo", key, validSHA256, func() (string, error) {
		return "res_inspection_bravo", nil
	})
	if err != nil {
		t.Fatalf("cross-tenant create failed: %v", err)
	}
	if isReplayBravo {
		t.Errorf("expected isReplay = false for different tenant")
	}
	if resTenantBravo != "res_inspection_bravo" {
		t.Errorf("unexpected tenant bravo resource ID: %s", resTenantBravo)
	}
}

func TestExternalRefRegistry_RegistrationAndResolution(t *testing.T) {
	reg := orgtenancy.NewExternalRefRegistry()
	tenantID := "ten_alpha"
	system := "SAP_PM"
	extID := "NOTIF-2026-9901"
	internalID := "ins_scaffold_98765432"

	// 1. Initial registration
	if err := reg.RegisterExternalRef(tenantID, system, extID, internalID); err != nil {
		t.Fatalf("RegisterExternalRef failed: %v", err)
	}

	// 2. Forward resolution
	resolvedInternal, err := reg.ResolveExternalRef(tenantID, system, extID)
	if err != nil {
		t.Fatalf("ResolveExternalRef failed: %v", err)
	}
	if resolvedInternal != internalID {
		t.Errorf("expected internal ID %s, got %s", internalID, resolvedInternal)
	}

	// 3. Reverse resolution
	resolvedExt, err := reg.ResolveInternal(tenantID, system, internalID)
	if err != nil {
		t.Fatalf("ResolveInternal failed: %v", err)
	}
	if resolvedExt != extID {
		t.Errorf("expected external ID %s, got %s", extID, resolvedExt)
	}

	// 4. Idempotent re-registration
	if err := reg.RegisterExternalRef(tenantID, system, extID, internalID); err != nil {
		t.Fatalf("identical re-registration should succeed cleanly: %v", err)
	}

	// 5. Conflicting external ID mapping (external ID already mapped to different internal ID)
	err = reg.RegisterExternalRef(tenantID, system, extID, "ins_other_11112222")
	if !errors.Is(err, orgtenancy.ErrDuplicateExternalRef) {
		t.Fatalf("expected ErrDuplicateExternalRef, got: %v", err)
	}

	// 6. Conflicting internal ID mapping (internal entity already mapped to different external ID in same system)
	err = reg.RegisterExternalRef(tenantID, system, "NOTIF-DIFFERENT-002", internalID)
	if !errors.Is(err, orgtenancy.ErrConflictingExternalRef) {
		t.Fatalf("expected ErrConflictingExternalRef, got: %v", err)
	}

	// 7. Tenant isolation on resolution
	_, err = reg.ResolveExternalRef("ten_bravo", system, extID)
	if !errors.Is(err, orgtenancy.ErrExternalRefNotFound) {
		t.Errorf("expected ErrExternalRefNotFound for cross-tenant query, got: %v", err)
	}
}

func TestExternalRefRegistry_AntiEnumerationBoundaries(t *testing.T) {
	reg := orgtenancy.NewExternalRefRegistry()
	tenantID := "ten_alpha"
	system := "ERP"
	internalID := "ins_00112233"

	tests := []struct {
		name        string
		extID       string
		expectedErr error
	}{
		{"too short", "ab", orgtenancy.ErrExternalIDTooShort},
		{"too long", strings.Repeat("x", 129), orgtenancy.ErrExternalIDTooLong},
		{"wildcard asterisk", "NOTIF*", orgtenancy.ErrExternalIDEnumeration},
		{"wildcard question", "NOTIF?", orgtenancy.ErrExternalIDEnumeration},
		{"wildcard percent", "%notif%", orgtenancy.ErrExternalIDEnumeration},
		{"injection chars", "notif;DROP", orgtenancy.ErrExternalIDEnumeration},
		{"trivial repeat digits", "0000", orgtenancy.ErrExternalIDEnumeration},
		{"trivial repeat letters", "aaaa", orgtenancy.ErrExternalIDEnumeration},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := reg.RegisterExternalRef(tenantID, system, tt.extID, internalID)
			if err == nil {
				t.Fatalf("expected error for %s, got nil", tt.name)
			}
			if !errors.Is(err, tt.expectedErr) {
				t.Errorf("expected %v, got: %v", tt.expectedErr, err)
			}
		})
	}
}
