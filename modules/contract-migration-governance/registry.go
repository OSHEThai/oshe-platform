// Package governance provides a dependency-free owned contract registry
// and deterministic compatibility gate foundation for v0.2.0 API, event,
// and schema contracts under milestone topic V020-T07.
package governance

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
)

var (
	// ErrEmptyContractName indicates that the contract name is empty or whitespace-only.
	ErrEmptyContractName = errors.New("contract name cannot be empty")
	// ErrInvalidContractKind indicates that the contract kind is unrecognized.
	ErrInvalidContractKind = errors.New("invalid contract kind: must be API, EVENT, or SCHEMA")
	// ErrEmptyOwner indicates that the owning module identifier is empty or whitespace-only.
	ErrEmptyOwner = errors.New("contract owner cannot be empty")
	// ErrInvalidDigest indicates that the content digest is not a valid 64-character lowercase SHA-256 hex string.
	ErrInvalidDigest = errors.New("contract digest must be a valid 64-character lowercase SHA-256 hex string")
	// ErrVersionConflict indicates that a registration attempt used an existing version with a conflicting owner or digest.
	ErrVersionConflict = errors.New("contract version conflict: owner or digest mismatch for existing version")
	// ErrVersionRegression indicates an attempt to register a version lower than the latest registered version.
	ErrVersionRegression = errors.New("contract version regression: registered version must be strictly greater than latest version")
	// ErrContractNotFound indicates that no contract matching the name and kind was found in the registry.
	ErrContractNotFound = errors.New("contract not found in registry")
)

// ContractKind defines the category of versioned contract.
type ContractKind string

const (
	// KindAPI represents an HTTP/REST or RPC public endpoint contract.
	KindAPI ContractKind = "API"
	// KindEvent represents an asynchronous message or outbox domain event contract.
	KindEvent ContractKind = "EVENT"
	// KindSchema represents a data model or payload JSON schema contract.
	KindSchema ContractKind = "SCHEMA"
)

// Validate checks whether the contract kind is one of the approved kinds.
func (k ContractKind) Validate() error {
	switch k {
	case KindAPI, KindEvent, KindSchema:
		return nil
	default:
		return ErrInvalidContractKind
	}
}

// SemVer represents a canonical semantic version (Major.Minor.Patch).
type SemVer struct {
	Major int
	Minor int
	Patch int
}

// String returns the dot-separated string representation (e.g., "1.2.3").
func (v SemVer) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// ParseSemVer parses a string in "Major.Minor.Patch" format into a SemVer struct.
func ParseSemVer(s string) (SemVer, error) {
	parts := strings.Split(strings.TrimSpace(s), ".")
	if len(parts) != 3 {
		return SemVer{}, fmt.Errorf("invalid semver format %q: expected major.minor.patch", s)
	}
	maj, err := strconv.Atoi(parts[0])
	if err != nil || maj < 0 {
		return SemVer{}, fmt.Errorf("invalid major version %q", parts[0])
	}
	min, err := strconv.Atoi(parts[1])
	if err != nil || min < 0 {
		return SemVer{}, fmt.Errorf("invalid minor version %q", parts[1])
	}
	patch, err := strconv.Atoi(parts[2])
	if err != nil || patch < 0 {
		return SemVer{}, fmt.Errorf("invalid patch version %q", parts[2])
	}
	return SemVer{Major: maj, Minor: min, Patch: patch}, nil
}

// Compare returns -1 if v < other, 0 if v == other, and 1 if v > other.
func (v SemVer) Compare(other SemVer) int {
	if v.Major != other.Major {
		if v.Major < other.Major {
			return -1
		}
		return 1
	}
	if v.Minor != other.Minor {
		if v.Minor < other.Minor {
			return -1
		}
		return 1
	}
	if v.Patch != other.Patch {
		if v.Patch < other.Patch {
			return -1
		}
		return 1
	}
	return 0
}

// ContractRecord captures an immutable registered contract revision.
type ContractRecord struct {
	Name               string       `json:"name"`
	Kind               ContractKind `json:"kind"`
	Owner              string       `json:"owner"`
	Version            SemVer       `json:"version"`
	Digest             string       `json:"digest"`
	BackwardCompatible bool         `json:"backward_compatible"`
}

// Validate ensures all mandatory fields of a ContractRecord are present and well-formed.
func (c ContractRecord) Validate() error {
	if strings.TrimSpace(c.Name) == "" {
		return ErrEmptyContractName
	}
	if err := c.Kind.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(c.Owner) == "" {
		return ErrEmptyOwner
	}
	d := strings.TrimSpace(c.Digest)
	if len(d) != 64 {
		return ErrInvalidDigest
	}
	raw, err := hex.DecodeString(d)
	if err != nil || len(raw) != 32 {
		return ErrInvalidDigest
	}
	// Check lowercase
	if strings.ToLower(d) != d {
		return ErrInvalidDigest
	}
	return nil
}

// CompatibilityOutcome records the deterministic result of evaluating contract compatibility.
type CompatibilityOutcome struct {
	Decision             string `json:"decision"` // "COMPATIBLE" or "MIGRATION_REQUIRED"
	RequiresMigration    bool   `json:"requires_migration"`
	MigrationRequirement string `json:"migration_requirement,omitempty"`
	Reason               string `json:"reason"`
}

// ContractRegistry provides thread-safe in-memory management of versioned contracts.
type ContractRegistry struct {
	mu      sync.RWMutex
	records map[string][]ContractRecord // key: name + "\x00" + kind
}

// NewContractRegistry constructs an initialized in-memory contract registry.
func NewContractRegistry() *ContractRegistry {
	return &ContractRegistry{
		records: make(map[string][]ContractRecord),
	}
}

func contractKey(name string, kind ContractKind) string {
	return fmt.Sprintf("%s\x00%s", strings.TrimSpace(name), string(kind))
}

// Register adds a contract record to the registry.
// Invariants:
// - Re-registering an identical record (same name, kind, owner, version, digest, backward_compatible) is an idempotent no-op.
// - Registering an existing version with a different owner or digest returns ErrVersionConflict.
// - Registering a version lower than the latest version returns ErrVersionRegression.
// - Registering a version strictly greater than latest version succeeds.
func (r *ContractRegistry) Register(rec ContractRecord) error {
	if err := rec.Validate(); err != nil {
		return err
	}

	normRec := ContractRecord{
		Name:               strings.TrimSpace(rec.Name),
		Kind:               rec.Kind,
		Owner:              strings.TrimSpace(rec.Owner),
		Version:            rec.Version,
		Digest:             strings.TrimSpace(rec.Digest),
		BackwardCompatible: rec.BackwardCompatible,
	}

	key := contractKey(normRec.Name, normRec.Kind)

	r.mu.Lock()
	defer r.mu.Unlock()

	history, exists := r.records[key]
	if !exists || len(history) == 0 {
		r.records[key] = []ContractRecord{normRec}
		return nil
	}

	// Check if this exact version is already present in history
	for _, existing := range history {
		if existing.Version.Compare(normRec.Version) == 0 {
			if existing.Owner == normRec.Owner &&
				existing.Digest == normRec.Digest &&
				existing.BackwardCompatible == normRec.BackwardCompatible {
				return nil // Idempotent success
			}
			return fmt.Errorf("%w: version %s exists with owner %q / digest %s",
				ErrVersionConflict, existing.Version, existing.Owner, existing.Digest)
		}
	}

	// Verify monotonic version progression against the latest recorded version
	latest := history[len(history)-1]
	if normRec.Version.Compare(latest.Version) <= 0 {
		return fmt.Errorf("%w: candidate %s is not greater than latest %s",
			ErrVersionRegression, normRec.Version, latest.Version)
	}

	r.records[key] = append(r.records[key], normRec)
	return nil
}

// EvaluateCompatibility evaluates whether a candidate record is backward-compatible with an existing record.
// A candidate is COMPATIBLE if:
// 1. It is identical to the existing record, OR
// 2. It is a monotonic minor or patch addition with identical Major version and explicitly marked BackwardCompatible=true.
// Otherwise, it returns MIGRATION_REQUIRED with a descriptive non-empty migration requirement.
func (r *ContractRegistry) EvaluateCompatibility(existing, candidate ContractRecord) CompatibilityOutcome {
	if existing.Name != candidate.Name || existing.Kind != candidate.Kind {
		return CompatibilityOutcome{
			Decision:             "MIGRATION_REQUIRED",
			RequiresMigration:    true,
			MigrationRequirement: fmt.Sprintf("contract identity mismatch between %s:%s and %s:%s", existing.Kind, existing.Name, candidate.Kind, candidate.Name),
			Reason:               "cannot evaluate compatibility across different contracts or kinds",
		}
	}

	if existing.Owner != candidate.Owner {
		return CompatibilityOutcome{
			Decision:             "MIGRATION_REQUIRED",
			RequiresMigration:    true,
			MigrationRequirement: fmt.Sprintf("ownership transfer from %q to %q requires governance migration approval", existing.Owner, candidate.Owner),
			Reason:               "contract owner transfer requires explicit migration review",
		}
	}

	// Identical version and digest
	if existing.Version.Compare(candidate.Version) == 0 && existing.Digest == candidate.Digest {
		return CompatibilityOutcome{
			Decision:          "COMPATIBLE",
			RequiresMigration: false,
			Reason:            "identical contract version and content digest",
		}
	}

	// Version regression
	if candidate.Version.Compare(existing.Version) < 0 {
		return CompatibilityOutcome{
			Decision:             "MIGRATION_REQUIRED",
			RequiresMigration:    true,
			MigrationRequirement: fmt.Sprintf("candidate version %s is a regression from %s", candidate.Version, existing.Version),
			Reason:               "version regression is not permitted without recovery migration",
		}
	}

	// Breaking change: Major version increment
	if candidate.Version.Major > existing.Version.Major {
		return CompatibilityOutcome{
			Decision:             "MIGRATION_REQUIRED",
			RequiresMigration:    true,
			MigrationRequirement: fmt.Sprintf("major version transition from %s to %s requires breaking-change migration plan", existing.Version, candidate.Version),
			Reason:               "major version increment introduces non-backward-compatible changes",
		}
	}

	// Same major version: check if candidate is marked backward-compatible
	if candidate.Version.Major == existing.Version.Major {
		if candidate.BackwardCompatible {
			return CompatibilityOutcome{
				Decision:          "COMPATIBLE",
				RequiresMigration: false,
				Reason:            fmt.Sprintf("monotonic version upgrade from %s to %s explicitly marked backward-compatible", existing.Version, candidate.Version),
			}
		}
		return CompatibilityOutcome{
			Decision:             "MIGRATION_REQUIRED",
			RequiresMigration:    true,
			MigrationRequirement: fmt.Sprintf("minor/patch progression from %s to %s lacking backward-compatibility certification requires consumer migration", existing.Version, candidate.Version),
			Reason:               "non-backward-compatible minor or patch modification",
		}
	}

	return CompatibilityOutcome{
		Decision:             "MIGRATION_REQUIRED",
		RequiresMigration:    true,
		MigrationRequirement: "unclassified contract state change requires manual migration review",
		Reason:               "unhandled compatibility state",
	}
}

// GetLatest returns the highest version registered for a given contract.
func (r *ContractRegistry) GetLatest(name string, kind ContractKind) (ContractRecord, error) {
	key := contractKey(name, kind)
	r.mu.RLock()
	defer r.mu.RUnlock()

	history, found := r.records[key]
	if !found || len(history) == 0 {
		return ContractRecord{}, ErrContractNotFound
	}
	return history[len(history)-1], nil
}

// GetVersion returns a specific registered version for a contract.
func (r *ContractRegistry) GetVersion(name string, kind ContractKind, version SemVer) (ContractRecord, error) {
	key := contractKey(name, kind)
	r.mu.RLock()
	defer r.mu.RUnlock()

	history, found := r.records[key]
	if !found {
		return ContractRecord{}, ErrContractNotFound
	}
	for _, rec := range history {
		if rec.Version.Compare(version) == 0 {
			return rec, nil
		}
	}
	return ContractRecord{}, ErrContractNotFound
}

// List returns all registered versions in chronological/monotonic order for a contract.
func (r *ContractRegistry) List(name string, kind ContractKind) []ContractRecord {
	key := contractKey(name, kind)
	r.mu.RLock()
	defer r.mu.RUnlock()

	history, found := r.records[key]
	if !found {
		return []ContractRecord{}
	}
	res := make([]ContractRecord, len(history))
	copy(res, history)
	return res
}
