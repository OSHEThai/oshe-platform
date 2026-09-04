package api_test

import (
	"errors"
	"testing"

	"github.com/oshethai/oshe-platform/contracts/api"
)

const (
	sampleDigest = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
)

func TestRequestEnvelope_ValidConstruction(t *testing.T) {
	env, err := api.NewRequestEnvelope("ten_alpha", "corr_0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatalf("unexpected envelope construction error: %v", err)
	}

	env.CausationID = "caus_0123456789abcdef0123456789abcdef"
	env.Pagination = &api.PaginationParams{
		Limit:  20,
		Cursor: "eyJvZmZzZXQiOjIwfQ==",
	}
	env.Concurrency = &api.ConcurrencyToken{
		ExpectedRevision: 3,
		ETag:             "W/\"rev-3\"",
	}
	env.Idempotency = &api.IdempotencyParams{
		Key:           "idemp-key-001",
		PayloadDigest: sampleDigest,
	}

	if err := env.Validate(); err != nil {
		t.Fatalf("expected valid envelope, got error: %v", err)
	}

	if env.Version != api.CurrentContractVersion {
		t.Errorf("expected version %s, got %s", api.CurrentContractVersion, env.Version)
	}
	if env.TenantID != "ten_alpha" {
		t.Errorf("expected tenant_id 'ten_alpha', got %s", env.TenantID)
	}
}

func TestRequestEnvelope_Denial_BlankAndInvalidTenant(t *testing.T) {
	cases := []struct {
		name     string
		tenantID string
		wantErr  error
	}{
		{"empty", "", api.ErrBlankTenantScope},
		{"whitespace", "   \t", api.ErrBlankTenantScope},
		{"missing_prefix", "alpha_tenant", api.ErrInvalidTenantScope},
		{"wrong_prefix_org", "org_alpha", api.ErrInvalidTenantScope},
		{"wrong_prefix_usr", "usr_alpha", api.ErrInvalidTenantScope},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := api.NewRequestEnvelope(tc.tenantID, "corr_0123456789abcdef0123456789abcdef")
			if err == nil {
				t.Fatalf("expected error for tenant %q, got nil", tc.tenantID)
			}
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("expected error %v, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestRequestEnvelope_Denial_UnsupportedVersion(t *testing.T) {
	env, _ := api.NewRequestEnvelope("ten_alpha", "corr_0123456789abcdef0123456789abcdef")

	invalidVersions := []string{"", "v2", "1.0.0", "beta"}
	for _, v := range invalidVersions {
		t.Run(v, func(t *testing.T) {
			env.Version = v
			err := env.Validate()
			if err == nil {
				t.Fatalf("expected error for version %q, got nil", v)
			}
			if !errors.Is(err, api.ErrUnsupportedVersion) {
				t.Errorf("expected ErrUnsupportedVersion, got %v", err)
			}
		})
	}
}

func TestRequestEnvelope_Denial_BlankAndInvalidCorrelation(t *testing.T) {
	cases := []struct {
		name    string
		corrID  string
		wantErr error
	}{
		{"empty", "", api.ErrBlankCorrelationID},
		{"whitespace", "   ", api.ErrBlankCorrelationID},
		{"missing_prefix", "0123456789abcdef0123456789abcdef", api.ErrInvalidCorrelationID},
		{"wrong_prefix_caus", "caus_0123456789abcdef0123456789abcdef", api.ErrInvalidCorrelationID},
		{"wrong_prefix_ten", "ten_0123456789abcdef0123456789abcdef", api.ErrInvalidCorrelationID},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := api.NewRequestEnvelope("ten_alpha", tc.corrID)
			if err == nil {
				t.Fatalf("expected error for corrID %q, got nil", tc.corrID)
			}
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("expected error %v, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestPagination_StableBoundariesAndMalformedDenial(t *testing.T) {
	// Valid limits
	for _, limit := range []int{1, 20, 50, api.MaxPaginationLimit} {
		p := &api.PaginationParams{Limit: limit, Cursor: "valid_cursor_token"}
		if err := p.Validate(); err != nil {
			t.Errorf("expected limit %d to be valid, got %v", limit, err)
		}
	}

	// Invalid limits
	for _, invalidLimit := range []int{-10, 0, api.MaxPaginationLimit + 1, 500} {
		p := &api.PaginationParams{Limit: invalidLimit}
		if err := p.Validate(); !errors.Is(err, api.ErrInvalidPagination) {
			t.Errorf("expected ErrInvalidPagination for limit %d, got %v", invalidLimit, err)
		}
	}

	// Malformed cursor with whitespace
	p := &api.PaginationParams{Limit: 20, Cursor: "bad cursor with spaces"}
	if err := p.Validate(); !errors.Is(err, api.ErrInvalidPagination) {
		t.Errorf("expected ErrInvalidPagination for cursor with whitespace, got %v", err)
	}
}

func TestConcurrency_EvaluationAndStaleTokenBehavior(t *testing.T) {
	// Matching revision
	matchingToken := &api.ConcurrencyToken{ExpectedRevision: 5, ETag: "v5"}
	if err := api.EvaluateConcurrency(matchingToken, 5); err != nil {
		t.Errorf("expected matching revision to pass, got %v", err)
	}

	// Stale revision (older)
	staleToken := &api.ConcurrencyToken{ExpectedRevision: 4, ETag: "v4"}
	err := api.EvaluateConcurrency(staleToken, 5)
	if err == nil {
		t.Fatal("expected error on stale revision, got nil")
	}
	if !errors.Is(err, api.ErrStaleConcurrencyToken) {
		t.Errorf("expected ErrStaleConcurrencyToken, got %v", err)
	}

	// Nil token
	if err := api.EvaluateConcurrency(nil, 5); err == nil {
		t.Error("expected error for nil concurrency token, got nil")
	}
}

func TestIdempotency_DuplicateAndConflictingReplay(t *testing.T) {
	key := "idemp-test-999"
	digest := sampleDigest

	// Identical replay
	isReplay, err := api.EvaluateIdempotencyReplay(key, digest, key, digest)
	if err != nil {
		t.Fatalf("expected identical replay to succeed without error, got: %v", err)
	}
	if !isReplay {
		t.Error("expected isReplay to be true for identical payload")
	}

	// Conflicting replay with different digest
	differentDigest := "fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210"
	isReplayConflict, err := api.EvaluateIdempotencyReplay(key, digest, key, differentDigest)
	if err == nil {
		t.Fatal("expected conflict error for different payload digest, got nil")
	}
	if !errors.Is(err, api.ErrDuplicateIdempotencyKey) {
		t.Errorf("expected ErrDuplicateIdempotencyKey, got %v", err)
	}
	if isReplayConflict {
		t.Error("expected isReplay to be false on conflict")
	}
}

func TestScope_ContradictoryMetadataAndMismatchedTenant(t *testing.T) {
	// Matching tenant scope
	if err := api.ValidateScopeMatch("ten_alpha", "ten_alpha"); err != nil {
		t.Errorf("expected matching tenant scope to pass, got %v", err)
	}

	// Cross-tenant mismatch
	err := api.ValidateScopeMatch("ten_alpha", "ten_beta")
	if err == nil {
		t.Fatal("expected error on mismatched tenant scope, got nil")
	}
	if !errors.Is(err, api.ErrContradictoryMetadata) {
		t.Errorf("expected ErrContradictoryMetadata, got %v", err)
	}

	// Blank tenant inputs
	if err := api.ValidateScopeMatch("", "ten_beta"); !errors.Is(err, api.ErrContradictoryMetadata) {
		t.Errorf("expected ErrContradictoryMetadata for blank caller tenant, got %v", err)
	}
}

func TestProvisionalGovernance_NonRuntimeDeclaration(t *testing.T) {
	if api.ProvisionalStatus != "PROVISIONAL_PENDING_H020_005" {
		t.Errorf("expected ProvisionalStatus 'PROVISIONAL_PENDING_H020_005', got %s", api.ProvisionalStatus)
	}
	if api.CurrentContractVersion != "v1" {
		t.Errorf("expected CurrentContractVersion 'v1', got %s", api.CurrentContractVersion)
	}
}
