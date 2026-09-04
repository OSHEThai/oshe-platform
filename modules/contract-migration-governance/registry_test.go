package governance_test

import (
	"errors"
	"strings"
	"testing"

	governance "github.com/oshethai/oshe-platform/modules/contract-migration-governance"
)

const (
	digestA = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	digestB = "fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210"
)

func mustSemVer(t *testing.T, s string) governance.SemVer {
	t.Helper()
	v, err := governance.ParseSemVer(s)
	if err != nil {
		t.Fatalf("invalid semver fixture %q: %v", s, err)
	}
	return v
}

func TestRegistry_InitialAndIdenticalRegistration(t *testing.T) {
	registry := governance.NewContractRegistry()
	rec := governance.ContractRecord{
		Name:               "api.error_envelope",
		Kind:               governance.KindAPI,
		Owner:              "MOD-CTR",
		Version:            mustSemVer(t, "1.0.0"),
		Digest:             digestA,
		BackwardCompatible: true,
	}

	// Initial registration
	if err := registry.Register(rec); err != nil {
		t.Fatalf("expected initial registration to succeed, got: %v", err)
	}

	// Idempotent re-registration of identical record
	if err := registry.Register(rec); err != nil {
		t.Fatalf("expected identical re-registration to succeed idempotently, got: %v", err)
	}

	// Verify latest
	latest, err := registry.GetLatest("api.error_envelope", governance.KindAPI)
	if err != nil {
		t.Fatalf("failed to get latest contract: %v", err)
	}
	if latest.Version.String() != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %s", latest.Version)
	}
	if latest.Digest != digestA {
		t.Errorf("expected digest %s, got %s", digestA, latest.Digest)
	}
}

func TestRegistry_DuplicateConflict_DigestOrOwner(t *testing.T) {
	registry := governance.NewContractRegistry()
	base := governance.ContractRecord{
		Name:               "evt.inspection_completed",
		Kind:               governance.KindEvent,
		Owner:              "MOD-WFA",
		Version:            mustSemVer(t, "1.0.0"),
		Digest:             digestA,
		BackwardCompatible: true,
	}

	if err := registry.Register(base); err != nil {
		t.Fatalf("initial registration failed: %v", err)
	}

	// Conflicting digest at same version
	conflictingDigest := base
	conflictingDigest.Digest = digestB
	err := registry.Register(conflictingDigest)
	if err == nil {
		t.Fatal("expected ErrVersionConflict for modified digest at same version, got nil")
	}
	if !errors.Is(err, governance.ErrVersionConflict) {
		t.Errorf("expected ErrVersionConflict, got: %v", err)
	}

	// Conflicting owner at same version
	conflictingOwner := base
	conflictingOwner.Owner = "MOD-ORG"
	err = registry.Register(conflictingOwner)
	if err == nil {
		t.Fatal("expected ErrVersionConflict for modified owner at same version, got nil")
	}
	if !errors.Is(err, governance.ErrVersionConflict) {
		t.Errorf("expected ErrVersionConflict, got: %v", err)
	}
}

func TestRegistry_VersionRegression(t *testing.T) {
	registry := governance.NewContractRegistry()
	recV11 := governance.ContractRecord{
		Name:               "schema.inspection_record",
		Kind:               governance.KindSchema,
		Owner:              "MOD-CTR",
		Version:            mustSemVer(t, "1.1.0"),
		Digest:             digestA,
		BackwardCompatible: true,
	}

	if err := registry.Register(recV11); err != nil {
		t.Fatalf("registration of 1.1.0 failed: %v", err)
	}

	// Attempt to register 1.0.0 after 1.1.0
	recV10 := recV11
	recV10.Version = mustSemVer(t, "1.0.0")
	err := registry.Register(recV10)
	if err == nil {
		t.Fatal("expected ErrVersionRegression when registering 1.0.0 after 1.1.0, got nil")
	}
	if !errors.Is(err, governance.ErrVersionRegression) {
		t.Errorf("expected ErrVersionRegression, got: %v", err)
	}
}

func TestRegistry_MonotonicUpgrade(t *testing.T) {
	registry := governance.NewContractRegistry()
	name := "api.inspection_session"
	kind := governance.KindAPI

	versions := []string{"1.0.0", "1.1.0", "1.1.1", "2.0.0"}
	for _, vStr := range versions {
		rec := governance.ContractRecord{
			Name:               name,
			Kind:               kind,
			Owner:              "MOD-WFA",
			Version:            mustSemVer(t, vStr),
			Digest:             digestA,
			BackwardCompatible: !strings.HasPrefix(vStr, "2."),
		}
		if err := registry.Register(rec); err != nil {
			t.Fatalf("failed to register version %s: %v", vStr, err)
		}
	}

	list := registry.List(name, kind)
	if len(list) != 4 {
		t.Fatalf("expected 4 versions in list, got %d", len(list))
	}
	for i, expected := range versions {
		if list[i].Version.String() != expected {
			t.Errorf("index %d: expected version %s, got %s", i, expected, list[i].Version)
		}
	}
}

func TestEvaluateCompatibility_IdenticalAndCompatible(t *testing.T) {
	registry := governance.NewContractRegistry()
	v10 := governance.ContractRecord{
		Name:               "api.checklist",
		Kind:               governance.KindAPI,
		Owner:              "MOD-CFG",
		Version:            mustSemVer(t, "1.0.0"),
		Digest:             digestA,
		BackwardCompatible: true,
	}

	// Identical record
	outcome := registry.EvaluateCompatibility(v10, v10)
	if outcome.Decision != "COMPATIBLE" || outcome.RequiresMigration {
		t.Errorf("expected identical record to be COMPATIBLE, got %+v", outcome)
	}

	// Backward-compatible minor addition
	v11 := v10
	v11.Version = mustSemVer(t, "1.1.0")
	v11.Digest = digestB
	v11.BackwardCompatible = true

	outcome = registry.EvaluateCompatibility(v10, v11)
	if outcome.Decision != "COMPATIBLE" || outcome.RequiresMigration {
		t.Errorf("expected backward-compatible minor bump to be COMPATIBLE, got %+v", outcome)
	}

	// Backward-compatible patch addition
	v101 := v10
	v101.Version = mustSemVer(t, "1.0.1")
	v101.Digest = digestB
	v101.BackwardCompatible = true

	outcome = registry.EvaluateCompatibility(v10, v101)
	if outcome.Decision != "COMPATIBLE" || outcome.RequiresMigration {
		t.Errorf("expected backward-compatible patch bump to be COMPATIBLE, got %+v", outcome)
	}
}

func TestEvaluateCompatibility_BreakingAndRejections(t *testing.T) {
	registry := governance.NewContractRegistry()
	base := governance.ContractRecord{
		Name:               "evt.finding_created",
		Kind:               governance.KindEvent,
		Owner:              "MOD-WFA",
		Version:            mustSemVer(t, "1.0.0"),
		Digest:             digestA,
		BackwardCompatible: true,
	}

	// Breaking major bump (e.g. 1.0.0 -> 2.0.0)
	majorBump := base
	majorBump.Version = mustSemVer(t, "2.0.0")
	majorBump.Digest = digestB

	outcome := registry.EvaluateCompatibility(base, majorBump)
	if outcome.Decision != "MIGRATION_REQUIRED" || !outcome.RequiresMigration {
		t.Errorf("expected major bump to require migration, got: %+v", outcome)
	}
	if outcome.MigrationRequirement == "" {
		t.Errorf("expected non-empty MigrationRequirement for major bump")
	}

	// Minor bump not marked backward-compatible
	breakingMinor := base
	breakingMinor.Version = mustSemVer(t, "1.2.0")
	breakingMinor.Digest = digestB
	breakingMinor.BackwardCompatible = false

	outcome = registry.EvaluateCompatibility(base, breakingMinor)
	if outcome.Decision != "MIGRATION_REQUIRED" || !outcome.RequiresMigration {
		t.Errorf("expected non-backward-compatible minor bump to require migration, got: %+v", outcome)
	}
	if outcome.MigrationRequirement == "" {
		t.Errorf("expected non-empty MigrationRequirement for breaking minor bump")
	}

	// Mismatched contract name
	diffName := base
	diffName.Name = "evt.other_name"
	outcome = registry.EvaluateCompatibility(base, diffName)
	if outcome.Decision != "MIGRATION_REQUIRED" || outcome.MigrationRequirement == "" {
		t.Errorf("expected mismatched name to require migration with requirement, got: %+v", outcome)
	}

	// Mismatched owner
	diffOwner := base
	diffOwner.Owner = "MOD-ORG"
	outcome = registry.EvaluateCompatibility(base, diffOwner)
	if outcome.Decision != "MIGRATION_REQUIRED" || outcome.MigrationRequirement == "" {
		t.Errorf("expected mismatched owner to require migration with requirement, got: %+v", outcome)
	}

	// Regression
	regression := base
	regression.Version = mustSemVer(t, "0.9.0")
	outcome = registry.EvaluateCompatibility(base, regression)
	if outcome.Decision != "MIGRATION_REQUIRED" || outcome.MigrationRequirement == "" {
		t.Errorf("expected regression to require migration with requirement, got: %+v", outcome)
	}
}

func TestRegistry_LookupNotFound(t *testing.T) {
	registry := governance.NewContractRegistry()

	_, err := registry.GetLatest("non_existent", governance.KindAPI)
	if !errors.Is(err, governance.ErrContractNotFound) {
		t.Errorf("expected ErrContractNotFound, got: %v", err)
	}

	_, err = registry.GetVersion("non_existent", governance.KindAPI, mustSemVer(t, "1.0.0"))
	if !errors.Is(err, governance.ErrContractNotFound) {
		t.Errorf("expected ErrContractNotFound, got: %v", err)
	}

	list := registry.List("non_existent", governance.KindAPI)
	if len(list) != 0 {
		t.Errorf("expected empty list for non-existent contract, got %d", len(list))
	}
}
