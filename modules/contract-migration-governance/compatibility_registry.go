// Package governance provides a deterministic, module-owned compatibility
// registry foundation for API, event, and schema contracts under milestone topic V020-T07.
package governance

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
)

var (
	// ErrBlankMetadata indicates a blank, empty, or whitespace-only required metadata field.
	ErrBlankMetadata = errors.New("blank or whitespace metadata field")
	// ErrDuplicateOwnership indicates that a contract name and kind is already owned by another module.
	ErrDuplicateOwnership = errors.New("duplicate contract ownership: contract already owned by another module")
	// ErrCrossModuleMutationDenied indicates that a caller attempted to register or modify a contract owned by another module.
	ErrCrossModuleMutationDenied = errors.New("cross-module mutation denied: caller does not own target contract")
	// ErrUnknownDependency indicates that an entry declares a dependency on a contract or version that is not registered.
	ErrUnknownDependency = errors.New("unknown dependency: referenced contract version is not registered")
	// ErrIncompatibleWithoutMigrationPath indicates an incompatible backward/forward evolution lacking a required migration path.
	ErrIncompatibleWithoutMigrationPath = errors.New("incompatible contract evolution without explicit migration path")
	// ErrInvalidCompatibilityDirection indicates an unrecognized compatibility direction.
	ErrInvalidCompatibilityDirection = errors.New("invalid compatibility direction: must be BACKWARD, FORWARD, BIDIRECTIONAL, or BREAKING")
	// ErrInvalidMigrationPath indicates an ill-formed migration path specification.
	ErrInvalidMigrationPath = errors.New("invalid migration path specification")
)

// CompatibilityDirection specifies the compatibility guarantee of a contract version relative to its preceding version.
type CompatibilityDirection string

const (
	// DirectionBackward indicates new consumers/producers can read or handle payloads from older versions.
	DirectionBackward CompatibilityDirection = "BACKWARD"
	// DirectionForward indicates older consumers/producers can handle payloads from newer versions.
	DirectionForward CompatibilityDirection = "FORWARD"
	// DirectionBidirectional indicates full mutual compatibility across minor/patch upgrades.
	DirectionBidirectional CompatibilityDirection = "BIDIRECTIONAL"
	// DirectionBreaking indicates an incompatible modification in one or both directions.
	DirectionBreaking CompatibilityDirection = "BREAKING"
)

// Validate ensures the compatibility direction is one of the approved canonical values.
func (d CompatibilityDirection) Validate() error {
	switch d {
	case DirectionBackward, DirectionForward, DirectionBidirectional, DirectionBreaking:
		return nil
	default:
		return fmt.Errorf("%w: got %q", ErrInvalidCompatibilityDirection, d)
	}
}

// DependencyRef defines a typed reference to another registered contract upon which an entry depends.
type DependencyRef struct {
	Name    string       `json:"name"`
	Kind    ContractKind `json:"kind"`
	Version SemVer       `json:"version"`
}

// Validate ensures all mandatory fields of a DependencyRef are non-blank and well-formed.
func (d DependencyRef) Validate() error {
	if strings.TrimSpace(d.Name) == "" {
		return fmt.Errorf("%w: dependency name cannot be blank", ErrBlankMetadata)
	}
	if err := d.Kind.Validate(); err != nil {
		return err
	}
	return nil
}

// MigrationPath defines an explicit, structured procedure for migrating consumers across incompatible contract revisions.
type MigrationPath struct {
	MigrationID  string   `json:"migration_id"`
	FromVersion  SemVer   `json:"from_version"`
	ToVersion    SemVer   `json:"to_version"`
	Strategy     string   `json:"strategy"` // e.g. "DUAL_WRITE", "UPCAST", "DEPRECATION_WINDOW", "ADAPTER"
	Description  string   `json:"description"`
	Instructions []string `json:"instructions,omitempty"`
}

// Validate ensures the migration path contains complete, non-blank metadata and proper version boundaries.
func (m *MigrationPath) Validate() error {
	if m == nil {
		return nil
	}
	if strings.TrimSpace(m.MigrationID) == "" {
		return fmt.Errorf("%w: migration_id cannot be blank", ErrBlankMetadata)
	}
	if strings.TrimSpace(m.Strategy) == "" {
		return fmt.Errorf("%w: migration strategy cannot be blank", ErrBlankMetadata)
	}
	if strings.TrimSpace(m.Description) == "" {
		return fmt.Errorf("%w: migration description cannot be blank", ErrBlankMetadata)
	}
	if m.FromVersion.Compare(m.ToVersion) >= 0 {
		return fmt.Errorf("%w: from_version %s must be strictly less than to_version %s",
			ErrInvalidMigrationPath, m.FromVersion, m.ToVersion)
	}
	return nil
}

// CompatibilityEntry captures an immutable registered contract entry in the compatibility registry.
type CompatibilityEntry struct {
	Name          string                 `json:"name"`
	Kind          ContractKind           `json:"kind"`
	OwnerModule   string                 `json:"owner_module"`
	Version       SemVer                 `json:"version"`
	Digest        string                 `json:"digest"`
	Direction     CompatibilityDirection `json:"direction"`
	Dependencies  []DependencyRef        `json:"dependencies,omitempty"`
	MigrationPath *MigrationPath         `json:"migration_path,omitempty"`
	Description   string                 `json:"description,omitempty"`
}

// Validate ensures all mandatory fields of a CompatibilityEntry are present and well-formed.
func (e CompatibilityEntry) Validate() error {
	if strings.TrimSpace(e.Name) == "" {
		return fmt.Errorf("%w: contract name cannot be blank", ErrBlankMetadata)
	}
	if err := e.Kind.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(e.OwnerModule) == "" {
		return fmt.Errorf("%w: owner module cannot be blank", ErrBlankMetadata)
	}
	d := strings.TrimSpace(e.Digest)
	if len(d) != 64 {
		return ErrInvalidDigest
	}
	raw, err := hex.DecodeString(d)
	if err != nil || len(raw) != 32 || strings.ToLower(d) != d {
		return ErrInvalidDigest
	}
	if err := e.Direction.Validate(); err != nil {
		return err
	}
	for _, dep := range e.Dependencies {
		if err := dep.Validate(); err != nil {
			return err
		}
	}
	if e.MigrationPath != nil {
		if err := e.MigrationPath.Validate(); err != nil {
			return err
		}
		if e.MigrationPath.ToVersion.Compare(e.Version) != 0 {
			return fmt.Errorf("%w: migration to_version %s does not match entry version %s",
				ErrInvalidMigrationPath, e.MigrationPath.ToVersion, e.Version)
		}
	}
	return nil
}

// CompatibilityResult captures the evaluated compatibility relationship between two versions.
type CompatibilityResult struct {
	Decision          string                 `json:"decision"` // "COMPATIBLE", "MIGRATION_REQUIRED", or "INCOMPATIBLE"
	Direction         CompatibilityDirection `json:"direction"`
	RequiresMigration bool                   `json:"requires_migration"`
	MigrationPath     *MigrationPath         `json:"migration_path,omitempty"`
	Reason            string                 `json:"reason"`
}

// CompatibilityRegistry provides thread-safe, module-isolated in-memory management of versioned contracts.
type CompatibilityRegistry struct {
	mu             sync.RWMutex
	contractOwners map[string]string               // key: contractKey(name, kind) -> ownerModule
	entries        map[string][]CompatibilityEntry // key: contractKey(name, kind) -> chronological entries
}

// NewCompatibilityRegistry constructs an initialized in-memory compatibility registry.
func NewCompatibilityRegistry() *CompatibilityRegistry {
	return &CompatibilityRegistry{
		contractOwners: make(map[string]string),
		entries:        make(map[string][]CompatibilityEntry),
	}
}

// cloneEntry produces a deep defensive copy of a CompatibilityEntry.
func cloneEntry(e CompatibilityEntry) CompatibilityEntry {
	cloned := e
	if len(e.Dependencies) > 0 {
		cloned.Dependencies = make([]DependencyRef, len(e.Dependencies))
		copy(cloned.Dependencies, e.Dependencies)
	}
	if e.MigrationPath != nil {
		mp := *e.MigrationPath
		if len(e.MigrationPath.Instructions) > 0 {
			mp.Instructions = make([]string, len(e.MigrationPath.Instructions))
			copy(mp.Instructions, e.MigrationPath.Instructions)
		}
		cloned.MigrationPath = &mp
	}
	return cloned
}

// Register adds a versioned contract entry to the registry under strict module ownership and compatibility checks.
func (r *CompatibilityRegistry) Register(callerModule string, entry CompatibilityEntry) error {
	trimmedCaller := strings.TrimSpace(callerModule)
	if trimmedCaller == "" {
		return fmt.Errorf("%w: caller module cannot be blank", ErrBlankMetadata)
	}

	if err := entry.Validate(); err != nil {
		return err
	}

	trimmedOwner := strings.TrimSpace(entry.OwnerModule)
	if trimmedCaller != trimmedOwner {
		return fmt.Errorf("%w: caller %q cannot register contract on behalf of owner %q",
			ErrCrossModuleMutationDenied, trimmedCaller, trimmedOwner)
	}

	normEntry := CompatibilityEntry{
		Name:        strings.TrimSpace(entry.Name),
		Kind:        entry.Kind,
		OwnerModule: trimmedOwner,
		Version:     entry.Version,
		Digest:      strings.TrimSpace(entry.Digest),
		Direction:   entry.Direction,
		Description: strings.TrimSpace(entry.Description),
	}

	if len(entry.Dependencies) > 0 {
		normEntry.Dependencies = make([]DependencyRef, len(entry.Dependencies))
		for i, dep := range entry.Dependencies {
			normEntry.Dependencies[i] = DependencyRef{
				Name:    strings.TrimSpace(dep.Name),
				Kind:    dep.Kind,
				Version: dep.Version,
			}
		}
	}

	if entry.MigrationPath != nil {
		mp := *entry.MigrationPath
		mp.MigrationID = strings.TrimSpace(mp.MigrationID)
		mp.Strategy = strings.TrimSpace(mp.Strategy)
		mp.Description = strings.TrimSpace(mp.Description)
		if len(entry.MigrationPath.Instructions) > 0 {
			mp.Instructions = make([]string, len(entry.MigrationPath.Instructions))
			for i, inst := range entry.MigrationPath.Instructions {
				mp.Instructions[i] = strings.TrimSpace(inst)
			}
		}
		normEntry.MigrationPath = &mp
	}

	key := contractKey(normEntry.Name, normEntry.Kind)

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check dependencies against currently registered contracts
	for _, dep := range normEntry.Dependencies {
		depKey := contractKey(dep.Name, dep.Kind)
		depHistory, exists := r.entries[depKey]
		if !exists || len(depHistory) == 0 {
			return fmt.Errorf("%w: dependency %s:%s@%s does not exist",
				ErrUnknownDependency, dep.Kind, dep.Name, dep.Version)
		}
		found := false
		for _, exist := range depHistory {
			if exist.Version.Compare(dep.Version) == 0 {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("%w: dependency %s:%s at version %s is not registered",
				ErrUnknownDependency, dep.Kind, dep.Name, dep.Version)
		}
	}

	// Verify ownership isolation
	existingOwner, ownerExists := r.contractOwners[key]
	if ownerExists && existingOwner != normEntry.OwnerModule {
		return fmt.Errorf("%w: contract %s:%s is owned by %q, registration rejected for %q",
			ErrDuplicateOwnership, normEntry.Kind, normEntry.Name, existingOwner, normEntry.OwnerModule)
	}

	history := r.entries[key]
	if len(history) == 0 {
		// First registration: establish ownership and store entry
		r.contractOwners[key] = normEntry.OwnerModule
		r.entries[key] = []CompatibilityEntry{normEntry}
		return nil
	}

	// Check existing history
	latest := history[len(history)-1]

	// Check if this exact version is already registered
	for _, exist := range history {
		if exist.Version.Compare(normEntry.Version) == 0 {
			if exist.OwnerModule == normEntry.OwnerModule &&
				exist.Digest == normEntry.Digest &&
				exist.Direction == normEntry.Direction &&
				exist.Description == normEntry.Description {
				return nil // Idempotent re-registration of identical entry
			}
			return fmt.Errorf("%w: version %s exists with conflicting digest or metadata",
				ErrVersionConflict, normEntry.Version)
		}
	}

	// Check version regression
	if normEntry.Version.Compare(latest.Version) <= 0 {
		return fmt.Errorf("%w: version %s is not greater than latest registered version %s",
			ErrVersionRegression, normEntry.Version, latest.Version)
	}

	// Check compatibility and migration-path requirement
	isBreaking := false
	if normEntry.Version.Major > latest.Version.Major {
		isBreaking = true
	} else if normEntry.Direction == DirectionBreaking || normEntry.Direction == DirectionForward {
		// Major bump, explicit BREAKING, or FORWARD-only (lacks backward compatibility) is breaking
		isBreaking = true
	}

	if isBreaking {
		if normEntry.MigrationPath == nil {
			return fmt.Errorf("%w: evolution from %s to %s requires explicit migration path",
				ErrIncompatibleWithoutMigrationPath, latest.Version, normEntry.Version)
		}
		if normEntry.MigrationPath.FromVersion.Compare(latest.Version) != 0 {
			return fmt.Errorf("%w: migration path from_version %s does not match latest version %s",
				ErrInvalidMigrationPath, normEntry.MigrationPath.FromVersion, latest.Version)
		}
	}

	r.entries[key] = append(r.entries[key], normEntry)
	return nil
}

// EvaluateEvolution analyzes the compatibility transition between an existing record and a candidate record.
func (r *CompatibilityRegistry) EvaluateEvolution(existing, candidate CompatibilityEntry) (CompatibilityResult, error) {
	if existing.Name != candidate.Name || existing.Kind != candidate.Kind {
		return CompatibilityResult{
			Decision:  "INCOMPATIBLE",
			Direction: candidate.Direction,
			Reason:    "cannot evaluate compatibility across different contracts or kinds",
		}, nil
	}

	if existing.OwnerModule != candidate.OwnerModule {
		return CompatibilityResult{
			Decision:  "INCOMPATIBLE",
			Direction: candidate.Direction,
			Reason:    fmt.Sprintf("contract ownership mismatch: %s vs %s", existing.OwnerModule, candidate.OwnerModule),
		}, nil
	}

	// Identical version and digest
	if existing.Version.Compare(candidate.Version) == 0 && existing.Digest == candidate.Digest {
		return CompatibilityResult{
			Decision:          "COMPATIBLE",
			Direction:         candidate.Direction,
			RequiresMigration: false,
			Reason:            "identical contract version and content digest",
		}, nil
	}

	// Version regression
	if candidate.Version.Compare(existing.Version) < 0 {
		return CompatibilityResult{
			Decision:  "INCOMPATIBLE",
			Direction: candidate.Direction,
			Reason:    fmt.Sprintf("version regression from %s to %s", existing.Version, candidate.Version),
		}, ErrVersionRegression
	}

	// Breaking changes
	isBreaking := candidate.Version.Major > existing.Version.Major ||
		candidate.Direction == DirectionBreaking ||
		candidate.Direction == DirectionForward

	if isBreaking {
		if candidate.MigrationPath != nil && candidate.MigrationPath.FromVersion.Compare(existing.Version) == 0 {
			return CompatibilityResult{
				Decision:          "MIGRATION_REQUIRED",
				Direction:         candidate.Direction,
				RequiresMigration: true,
				MigrationPath:     candidate.MigrationPath,
				Reason:            fmt.Sprintf("breaking evolution from %s to %s accepted with registered migration path", existing.Version, candidate.Version),
			}, nil
		}
		return CompatibilityResult{
			Decision:          "INCOMPATIBLE",
			Direction:         candidate.Direction,
			RequiresMigration: true,
			Reason:            fmt.Sprintf("breaking change from %s to %s lacks required migration path", existing.Version, candidate.Version),
		}, nil
	}

	return CompatibilityResult{
		Decision:          "COMPATIBLE",
		Direction:         candidate.Direction,
		RequiresMigration: false,
		Reason:            fmt.Sprintf("compatible evolution from %s to %s (%s)", existing.Version, candidate.Version, candidate.Direction),
	}, nil
}

// GetLatest returns the highest version registered for a contract, defensively copied.
func (r *CompatibilityRegistry) GetLatest(name string, kind ContractKind) (CompatibilityEntry, error) {
	key := contractKey(name, kind)
	r.mu.RLock()
	defer r.mu.RUnlock()

	history, found := r.entries[key]
	if !found || len(history) == 0 {
		return CompatibilityEntry{}, ErrContractNotFound
	}
	return cloneEntry(history[len(history)-1]), nil
}

// GetVersion returns a specific registered version, defensively copied.
func (r *CompatibilityRegistry) GetVersion(name string, kind ContractKind, version SemVer) (CompatibilityEntry, error) {
	key := contractKey(name, kind)
	r.mu.RLock()
	defer r.mu.RUnlock()

	history, found := r.entries[key]
	if !found {
		return CompatibilityEntry{}, ErrContractNotFound
	}
	for _, entry := range history {
		if entry.Version.Compare(version) == 0 {
			return cloneEntry(entry), nil
		}
	}
	return CompatibilityEntry{}, ErrContractNotFound
}

// ListHistory returns all registered versions in chronological order, defensively copied.
func (r *CompatibilityRegistry) ListHistory(name string, kind ContractKind) []CompatibilityEntry {
	key := contractKey(name, kind)
	r.mu.RLock()
	defer r.mu.RUnlock()

	history, found := r.entries[key]
	if !found {
		return []CompatibilityEntry{}
	}
	result := make([]CompatibilityEntry, len(history))
	for i, entry := range history {
		result[i] = cloneEntry(entry)
	}
	return result
}

// GetOwner returns the module owning the specified contract.
func (r *CompatibilityRegistry) GetOwner(name string, kind ContractKind) (string, error) {
	key := contractKey(name, kind)
	r.mu.RLock()
	defer r.mu.RUnlock()

	owner, found := r.contractOwners[key]
	if !found {
		return "", ErrContractNotFound
	}
	return owner, nil
}

// UnmarshalJSON parses a SemVer from either a string like "1.2.3" or an object representation.
func (v *SemVer) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		parsed, err := ParseSemVer(s)
		if err != nil {
			return err
		}
		*v = parsed
		return nil
	}
	type rawSemVer SemVer
	var raw rawSemVer
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	*v = SemVer(raw)
	return nil
}

// MarshalJSON serializes SemVer to its canonical dot-separated string representation.
func (v SemVer) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.String())
}

// Count returns the total number of distinct contract versions registered across all contracts.
func (r *CompatibilityRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	count := 0
	for _, history := range r.entries {
		count += len(history)
	}
	return count
}

// ContractCount returns the total number of distinct contracts registered.
func (r *CompatibilityRegistry) ContractCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.entries)
}

// RegistryValidationItem defines a single item in a JSON validation payload.
type RegistryValidationItem struct {
	CallerModule string             `json:"caller_module,omitempty"`
	Entry        CompatibilityEntry `json:"entry"`
}

// RegistryValidationPayload models the batch JSON format for validating contract registries.
type RegistryValidationPayload struct {
	SchemaVersion string                   `json:"schema_version"`
	Entries       []RegistryValidationItem `json:"entries"`
}

// ValidateRegistryJSON parses a JSON payload and registers each entry into a fresh CompatibilityRegistry.
// It fails closed if JSON is malformed, schema_version is blank, or any entry violates:
// - metadata requirements (blank fields, invalid digest)
// - contract ownership (cross-module mutation, duplicate ownership)
// - dependency resolution (unresolved dependency contract/version)
// - version progression (version regression)
// - compatibility evolution (breaking/major without explicit valid migration path)
func ValidateRegistryJSON(data []byte) (*CompatibilityRegistry, []string) {
	var payload RegistryValidationPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, []string{fmt.Sprintf("JSON parse error: %v", err)}
	}

	if strings.TrimSpace(payload.SchemaVersion) == "" {
		return nil, []string{"schema_version cannot be blank"}
	}

	reg := NewCompatibilityRegistry()
	var errs []string

	for i, item := range payload.Entries {
		caller := strings.TrimSpace(item.CallerModule)
		if caller == "" {
			caller = strings.TrimSpace(item.Entry.OwnerModule)
		}

		if err := reg.Register(caller, item.Entry); err != nil {
			errs = append(errs, fmt.Sprintf("entry[%d] (%s:%s@%s): %v",
				i, item.Entry.Kind, item.Entry.Name, item.Entry.Version, err))
		}
	}

	return reg, errs
}
