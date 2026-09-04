package governance_test

import (
	"errors"
	"sync"
	"testing"

	governance "github.com/oshethai/oshe-platform/modules/contract-migration-governance"
)

func TestCompatibilityRegistry_InitialRegistration(t *testing.T) {
	reg := governance.NewCompatibilityRegistry()

	apiEntry := governance.CompatibilityEntry{
		Name:        "api.request_envelope",
		Kind:        governance.KindAPI,
		OwnerModule: "MOD-CTR",
		Version:     mustSemVer(t, "1.0.0"),
		Digest:      digestA,
		Direction:   governance.DirectionBackward,
		Description: "Initial API request envelope specification",
	}

	if err := reg.Register("MOD-CTR", apiEntry); err != nil {
		t.Fatalf("expected initial registration to succeed, got: %v", err)
	}

	// Read back latest
	latest, err := reg.GetLatest("api.request_envelope", governance.KindAPI)
	if err != nil {
		t.Fatalf("failed to retrieve latest entry: %v", err)
	}
	if latest.Name != "api.request_envelope" {
		t.Errorf("expected name api.request_envelope, got %s", latest.Name)
	}
	if latest.OwnerModule != "MOD-CTR" {
		t.Errorf("expected owner MOD-CTR, got %s", latest.OwnerModule)
	}
	if latest.Version.String() != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %s", latest.Version)
	}
	if latest.Direction != governance.DirectionBackward {
		t.Errorf("expected direction BACKWARD, got %s", latest.Direction)
	}

	// Verify owner lookup
	owner, err := reg.GetOwner("api.request_envelope", governance.KindAPI)
	if err != nil {
		t.Fatalf("failed to get owner: %v", err)
	}
	if owner != "MOD-CTR" {
		t.Errorf("expected owner MOD-CTR, got %s", owner)
	}
}

func TestCompatibilityRegistry_CompatibleEvolution(t *testing.T) {
	reg := governance.NewCompatibilityRegistry()

	v100 := governance.CompatibilityEntry{
		Name:        "evt.inspection_completed",
		Kind:        governance.KindEvent,
		OwnerModule: "MOD-EVT",
		Version:     mustSemVer(t, "1.0.0"),
		Digest:      digestA,
		Direction:   governance.DirectionBackward,
		Description: "Baseline inspection completed event",
	}
	if err := reg.Register("MOD-EVT", v100); err != nil {
		t.Fatalf("registration of 1.0.0 failed: %v", err)
	}

	// Compatible minor upgrade (1.1.0) with DirectionBackward
	v110 := v100
	v110.Version = mustSemVer(t, "1.1.0")
	v110.Digest = digestB
	v110.Direction = governance.DirectionBackward
	v110.Description = "Minor addition of non-breaking metadata"
	if err := reg.Register("MOD-EVT", v110); err != nil {
		t.Fatalf("expected compatible minor bump to succeed, got: %v", err)
	}

	// Compatible patch upgrade (1.1.1) with DirectionBidirectional
	v111 := v110
	v111.Version = mustSemVer(t, "1.1.1")
	v111.Digest = digestA
	v111.Direction = governance.DirectionBidirectional
	v111.Description = "Patch bugfix in validation"
	if err := reg.Register("MOD-EVT", v111); err != nil {
		t.Fatalf("expected compatible patch bump to succeed, got: %v", err)
	}

	history := reg.ListHistory("evt.inspection_completed", governance.KindEvent)
	if len(history) != 3 {
		t.Fatalf("expected 3 entries in history, got %d", len(history))
	}
	if history[2].Version.String() != "1.1.1" {
		t.Errorf("expected latest version 1.1.1, got %s", history[2].Version)
	}
}

func TestCompatibilityRegistry_IncompatibleEvolution_DeniedWithoutMigrationPath(t *testing.T) {
	reg := governance.NewCompatibilityRegistry()

	base := governance.CompatibilityEntry{
		Name:        "schema.finding_record",
		Kind:        governance.KindSchema,
		OwnerModule: "MOD-REC",
		Version:     mustSemVer(t, "1.0.0"),
		Digest:      digestA,
		Direction:   governance.DirectionBackward,
	}
	if err := reg.Register("MOD-REC", base); err != nil {
		t.Fatalf("initial registration failed: %v", err)
	}

	// 1. Major bump without migration path must fail
	majorBump := base
	majorBump.Version = mustSemVer(t, "2.0.0")
	majorBump.Digest = digestB
	majorBump.Direction = governance.DirectionBreaking
	err := reg.Register("MOD-REC", majorBump)
	if err == nil {
		t.Fatal("expected ErrIncompatibleWithoutMigrationPath for major bump, got nil")
	}
	if !errors.Is(err, governance.ErrIncompatibleWithoutMigrationPath) {
		t.Errorf("expected ErrIncompatibleWithoutMigrationPath, got: %v", err)
	}

	// 2. Minor bump marked DirectionBreaking without migration path must fail
	breakingMinor := base
	breakingMinor.Version = mustSemVer(t, "1.1.0")
	breakingMinor.Digest = digestB
	breakingMinor.Direction = governance.DirectionBreaking
	err = reg.Register("MOD-REC", breakingMinor)
	if err == nil {
		t.Fatal("expected ErrIncompatibleWithoutMigrationPath for breaking minor, got nil")
	}
	if !errors.Is(err, governance.ErrIncompatibleWithoutMigrationPath) {
		t.Errorf("expected ErrIncompatibleWithoutMigrationPath, got: %v", err)
	}

	// 3. Minor bump marked DirectionForward (forward-only lacks backward compatibility) must fail without migration path
	forwardOnly := base
	forwardOnly.Version = mustSemVer(t, "1.1.0")
	forwardOnly.Digest = digestB
	forwardOnly.Direction = governance.DirectionForward
	err = reg.Register("MOD-REC", forwardOnly)
	if err == nil {
		t.Fatal("expected ErrIncompatibleWithoutMigrationPath for forward-only minor, got nil")
	}
	if !errors.Is(err, governance.ErrIncompatibleWithoutMigrationPath) {
		t.Errorf("expected ErrIncompatibleWithoutMigrationPath, got: %v", err)
	}
}

func TestCompatibilityRegistry_MigrationPathAcceptance(t *testing.T) {
	reg := governance.NewCompatibilityRegistry()

	base := governance.CompatibilityEntry{
		Name:        "schema.checklist_template",
		Kind:        governance.KindSchema,
		OwnerModule: "MOD-CFG",
		Version:     mustSemVer(t, "1.2.0"),
		Digest:      digestA,
		Direction:   governance.DirectionBackward,
	}
	if err := reg.Register("MOD-CFG", base); err != nil {
		t.Fatalf("initial registration failed: %v", err)
	}

	// Major bump with valid MigrationPath must be accepted
	v200 := base
	v200.Version = mustSemVer(t, "2.0.0")
	v200.Digest = digestB
	v200.Direction = governance.DirectionBreaking
	v200.MigrationPath = &governance.MigrationPath{
		MigrationID:  "MIG-CFG-001",
		FromVersion:  mustSemVer(t, "1.2.0"),
		ToVersion:    mustSemVer(t, "2.0.0"),
		Strategy:     "DUAL_WRITE",
		Description:  "Migrate checklist section schema with dual-write translation layer",
		Instructions: []string{"Step 1: enable dual-read adapter", "Step 2: migrate legacy records"},
	}

	if err := reg.Register("MOD-CFG", v200); err != nil {
		t.Fatalf("expected major bump with valid migration path to succeed, got: %v", err)
	}

	latest, err := reg.GetLatest("schema.checklist_template", governance.KindSchema)
	if err != nil {
		t.Fatalf("failed to get latest after migration: %v", err)
	}
	if latest.Version.String() != "2.0.0" {
		t.Errorf("expected version 2.0.0, got %s", latest.Version)
	}
	if latest.MigrationPath == nil || latest.MigrationPath.MigrationID != "MIG-CFG-001" {
		t.Errorf("expected MigrationPath with ID MIG-CFG-001, got %+v", latest.MigrationPath)
	}

	// Mismatched from_version in MigrationPath must be rejected
	v300 := v200
	v300.Version = mustSemVer(t, "3.0.0")
	v300.Digest = digestA
	v300.MigrationPath = &governance.MigrationPath{
		MigrationID: "MIG-CFG-002",
		FromVersion: mustSemVer(t, "1.0.0"), // Incorrect: latest is 2.0.0
		ToVersion:   mustSemVer(t, "3.0.0"),
		Strategy:    "ADAPTER",
		Description: "Stale from-version migration",
	}
	err = reg.Register("MOD-CFG", v300)
	if err == nil {
		t.Fatal("expected ErrInvalidMigrationPath for mismatched from_version, got nil")
	}
	if !errors.Is(err, governance.ErrInvalidMigrationPath) {
		t.Errorf("expected ErrInvalidMigrationPath, got: %v", err)
	}
}

func TestCompatibilityRegistry_DuplicateAndVersionDenials(t *testing.T) {
	reg := governance.NewCompatibilityRegistry()

	base := governance.CompatibilityEntry{
		Name:        "api.access_control",
		Kind:        governance.KindAPI,
		OwnerModule: "MOD-IAM",
		Version:     mustSemVer(t, "1.0.0"),
		Digest:      digestA,
		Direction:   governance.DirectionBackward,
	}
	if err := reg.Register("MOD-IAM", base); err != nil {
		t.Fatalf("initial registration failed: %v", err)
	}

	// 1. Idempotent identical re-registration
	if err := reg.Register("MOD-IAM", base); err != nil {
		t.Fatalf("expected identical re-registration to succeed, got: %v", err)
	}

	// 2. Conflicting digest at same version
	conflictDigest := base
	conflictDigest.Digest = digestB
	err := reg.Register("MOD-IAM", conflictDigest)
	if err == nil || !errors.Is(err, governance.ErrVersionConflict) {
		t.Errorf("expected ErrVersionConflict for changed digest at same version, got: %v", err)
	}

	// 3. Version regression (e.g. 1.1.0 registered then attempting 1.0.5)
	v110 := base
	v110.Version = mustSemVer(t, "1.1.0")
	v110.Digest = digestB
	if err := reg.Register("MOD-IAM", v110); err != nil {
		t.Fatalf("register 1.1.0 failed: %v", err)
	}

	regression := base
	regression.Version = mustSemVer(t, "1.0.5")
	err = reg.Register("MOD-IAM", regression)
	if err == nil || !errors.Is(err, governance.ErrVersionRegression) {
		t.Errorf("expected ErrVersionRegression, got: %v", err)
	}

	// 4. Duplicate ownership denial: another module attempts to register the same contract
	err = reg.Register("MOD-ORG", governance.CompatibilityEntry{
		Name:        "api.access_control",
		Kind:        governance.KindAPI,
		OwnerModule: "MOD-ORG",
		Version:     mustSemVer(t, "1.2.0"),
		Digest:      digestA,
		Direction:   governance.DirectionBackward,
	})
	if err == nil || !errors.Is(err, governance.ErrDuplicateOwnership) {
		t.Errorf("expected ErrDuplicateOwnership when MOD-ORG claims MOD-IAM contract, got: %v", err)
	}
}

func TestCompatibilityRegistry_BlankMetadataAndValidation(t *testing.T) {
	reg := governance.NewCompatibilityRegistry()

	validEntry := governance.CompatibilityEntry{
		Name:        "api.metadata_test",
		Kind:        governance.KindAPI,
		OwnerModule: "MOD-CTR",
		Version:     mustSemVer(t, "1.0.0"),
		Digest:      digestA,
		Direction:   governance.DirectionBackward,
	}

	// Blank caller module
	if err := reg.Register("  ", validEntry); err == nil || !errors.Is(err, governance.ErrBlankMetadata) {
		t.Errorf("expected ErrBlankMetadata for blank caller module, got: %v", err)
	}

	// Blank contract name
	blankName := validEntry
	blankName.Name = "   "
	if err := reg.Register("MOD-CTR", blankName); err == nil || !errors.Is(err, governance.ErrBlankMetadata) {
		t.Errorf("expected ErrBlankMetadata for blank contract name, got: %v", err)
	}

	// Blank owner module
	blankOwner := validEntry
	blankOwner.OwnerModule = "   "
	if err := reg.Register("MOD-CTR", blankOwner); err == nil || !errors.Is(err, governance.ErrBlankMetadata) {
		t.Errorf("expected ErrBlankMetadata for blank owner module, got: %v", err)
	}

	// Invalid digest: uppercase hex
	badDigest := validEntry
	badDigest.Digest = "0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF"
	if err := reg.Register("MOD-CTR", badDigest); err == nil || !errors.Is(err, governance.ErrInvalidDigest) {
		t.Errorf("expected ErrInvalidDigest for uppercase hex, got: %v", err)
	}

	// Invalid digest: wrong length
	shortDigest := validEntry
	shortDigest.Digest = "0123456789abcdef"
	if err := reg.Register("MOD-CTR", shortDigest); err == nil || !errors.Is(err, governance.ErrInvalidDigest) {
		t.Errorf("expected ErrInvalidDigest for short hex, got: %v", err)
	}

	// Invalid direction
	badDirection := validEntry
	badDirection.Direction = "UNKNOWN_DIRECTION"
	if err := reg.Register("MOD-CTR", badDirection); err == nil || !errors.Is(err, governance.ErrInvalidCompatibilityDirection) {
		t.Errorf("expected ErrInvalidCompatibilityDirection, got: %v", err)
	}
}

func TestCompatibilityRegistry_UnknownDependencies(t *testing.T) {
	reg := governance.NewCompatibilityRegistry()

	// 1. Registering an entry with unknown dependency must fail
	entryWithUnknownDep := governance.CompatibilityEntry{
		Name:        "api.session_action",
		Kind:        governance.KindAPI,
		OwnerModule: "MOD-WFA",
		Version:     mustSemVer(t, "1.0.0"),
		Digest:      digestA,
		Direction:   governance.DirectionBackward,
		Dependencies: []governance.DependencyRef{
			{
				Name:    "schema.non_existent",
				Kind:    governance.KindSchema,
				Version: mustSemVer(t, "1.0.0"),
			},
		},
	}

	err := reg.Register("MOD-WFA", entryWithUnknownDep)
	if err == nil || !errors.Is(err, governance.ErrUnknownDependency) {
		t.Fatalf("expected ErrUnknownDependency for unregistered contract, got: %v", err)
	}

	// 2. Register the dependency contract at version 1.0.0
	depContract := governance.CompatibilityEntry{
		Name:        "schema.non_existent",
		Kind:        governance.KindSchema,
		OwnerModule: "MOD-CTR",
		Version:     mustSemVer(t, "1.0.0"),
		Digest:      digestB,
		Direction:   governance.DirectionBackward,
	}
	if err := reg.Register("MOD-CTR", depContract); err != nil {
		t.Fatalf("failed to register dependency contract: %v", err)
	}

	// 3. Dependency version mismatch: entry requires 2.0.0 but only 1.0.0 exists
	entryWithWrongDepVer := entryWithUnknownDep
	entryWithWrongDepVer.Dependencies = []governance.DependencyRef{
		{
			Name:    "schema.non_existent",
			Kind:    governance.KindSchema,
			Version: mustSemVer(t, "2.0.0"),
		},
	}
	err = reg.Register("MOD-WFA", entryWithWrongDepVer)
	if err == nil || !errors.Is(err, governance.ErrUnknownDependency) {
		t.Fatalf("expected ErrUnknownDependency for version mismatch, got: %v", err)
	}

	// 4. Registering with existing, matching dependency must now succeed
	if err := reg.Register("MOD-WFA", entryWithUnknownDep); err != nil {
		t.Fatalf("expected registration to succeed after registering dependency, got: %v", err)
	}
}

func TestCompatibilityRegistry_OwnershipIsolationAndDefensiveCopy(t *testing.T) {
	reg := governance.NewCompatibilityRegistry()

	entry := governance.CompatibilityEntry{
		Name:        "api.isolated_contract",
		Kind:        governance.KindAPI,
		OwnerModule: "MOD-CTR",
		Version:     mustSemVer(t, "1.0.0"),
		Digest:      digestA,
		Direction:   governance.DirectionBackward,
		MigrationPath: &governance.MigrationPath{
			MigrationID:  "MIG-ISO-001",
			FromVersion:  mustSemVer(t, "0.9.0"),
			ToVersion:    mustSemVer(t, "1.0.0"),
			Strategy:     "ADAPTER",
			Description:  "Adapter from prototype",
			Instructions: []string{"original instruction"},
		},
	}

	// Cross-module caller attempt must fail
	err := reg.Register("MOD-WFA", entry)
	if err == nil || !errors.Is(err, governance.ErrCrossModuleMutationDenied) {
		t.Fatalf("expected ErrCrossModuleMutationDenied when MOD-WFA calls for MOD-CTR entry, got: %v", err)
	}

	// Authorized caller succeeds
	if err := reg.Register("MOD-CTR", entry); err != nil {
		t.Fatalf("authorized registration failed: %v", err)
	}

	// Defensive copy test: retrieve and mutate fields externally
	retrieved, err := reg.GetLatest("api.isolated_contract", governance.KindAPI)
	if err != nil {
		t.Fatalf("failed to get latest: %v", err)
	}

	retrieved.MigrationPath.Instructions[0] = "mutated instruction"
	retrieved.Digest = digestB

	// Verify registry internal state remained unmodified
	readAgain, err := reg.GetLatest("api.isolated_contract", governance.KindAPI)
	if err != nil {
		t.Fatalf("failed to read again: %v", err)
	}
	if readAgain.Digest != digestA {
		t.Errorf("registry internal digest was mutated! expected %s, got %s", digestA, readAgain.Digest)
	}
	if readAgain.MigrationPath.Instructions[0] != "original instruction" {
		t.Errorf("registry internal instructions slice was mutated! expected 'original instruction', got %q",
			readAgain.MigrationPath.Instructions[0])
	}
}

func TestCompatibilityRegistry_EvaluateEvolution(t *testing.T) {
	reg := governance.NewCompatibilityRegistry()

	base := governance.CompatibilityEntry{
		Name:        "schema.evolution_test",
		Kind:        governance.KindSchema,
		OwnerModule: "MOD-CTR",
		Version:     mustSemVer(t, "1.0.0"),
		Digest:      digestA,
		Direction:   governance.DirectionBackward,
	}

	// Compatible minor
	candMinor := base
	candMinor.Version = mustSemVer(t, "1.1.0")
	candMinor.Digest = digestB
	res, err := reg.EvaluateEvolution(base, candMinor)
	if err != nil || res.Decision != "COMPATIBLE" || res.RequiresMigration {
		t.Errorf("expected COMPATIBLE, got %+v, err: %v", res, err)
	}

	// Incompatible major without migration path
	candMajor := base
	candMajor.Version = mustSemVer(t, "2.0.0")
	candMajor.Digest = digestB
	candMajor.Direction = governance.DirectionBreaking
	res, err = reg.EvaluateEvolution(base, candMajor)
	if err != nil || res.Decision != "INCOMPATIBLE" || !res.RequiresMigration {
		t.Errorf("expected INCOMPATIBLE requiring migration, got %+v, err: %v", res, err)
	}

	// Incompatible major with migration path
	candMajorWithMig := candMajor
	candMajorWithMig.MigrationPath = &governance.MigrationPath{
		MigrationID: "MIG-EVO-001",
		FromVersion: mustSemVer(t, "1.0.0"),
		ToVersion:   mustSemVer(t, "2.0.0"),
		Strategy:    "UPCAST",
		Description: "Upcast schema v1 to v2",
	}
	res, err = reg.EvaluateEvolution(base, candMajorWithMig)
	if err != nil || res.Decision != "MIGRATION_REQUIRED" || !res.RequiresMigration || res.MigrationPath == nil {
		t.Errorf("expected MIGRATION_REQUIRED with migration path, got %+v, err: %v", res, err)
	}
}

func TestCompatibilityRegistry_ConcurrentOperations(t *testing.T) {
	reg := governance.NewCompatibilityRegistry()

	var wg sync.WaitGroup
	workers := 10

	// Register base contract
	base := governance.CompatibilityEntry{
		Name:        "api.concurrency_target",
		Kind:        governance.KindAPI,
		OwnerModule: "MOD-CTR",
		Version:     mustSemVer(t, "1.0.0"),
		Digest:      digestA,
		Direction:   governance.DirectionBackward,
	}
	if err := reg.Register("MOD-CTR", base); err != nil {
		t.Fatalf("failed to register concurrency base: %v", err)
	}

	// Concurrent readers
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				_, err := reg.GetLatest("api.concurrency_target", governance.KindAPI)
				if err != nil {
					t.Errorf("concurrent read error: %v", err)
				}
			}
		}()
	}

	wg.Wait()
}
