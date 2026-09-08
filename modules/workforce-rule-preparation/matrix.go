// Package workforcerulepreparation provides a synthetic, in-memory rule-matrix
// preparation boundary. Every retained case is explicitly non-current and
// requires a human decision; this package cannot determine readiness.
package workforcerulepreparation

import (
	"errors"
	"strings"
	"sync"
	"time"
)

var (
	ErrBlankField                    = errors.New("required field is blank")
	ErrInvalidReference              = errors.New("reference must use its required opaque synthetic prefix")
	ErrDuplicateMatrix               = errors.New("rule matrix already exists in tenant")
	ErrDuplicateVersion              = errors.New("rule matrix version already exists in tenant")
	ErrDuplicateRequirement          = errors.New("requirement appears more than once in matrix")
	ErrIncompleteDispositionCoverage = errors.New("matrix must contain every required non-current disposition")
	ErrAssessmentNotFound            = errors.New("assessment not found")
)

// Disposition captures only planning cases that can never assert a current or
// binding workforce outcome before V050-CFG-004 is selected.
type Disposition string

const (
	Unknown    Disposition = "UNKNOWN"
	Incomplete Disposition = "INCOMPLETE"
	Expired    Disposition = "EXPIRED"
	Disputed   Disposition = "DISPUTED"
	Exception  Disposition = "EXCEPTION"
)

var requiredDispositions = map[Disposition]struct{}{
	Unknown: {}, Incomplete: {}, Expired: {}, Disputed: {}, Exception: {},
}

// Entry is a content-free synthetic requirement reference and its
// non-current disposition. It contains no training, credential, role,
// appointment, date, rule value, evidence, or person data.
type Entry struct {
	RequirementRef string
	Disposition    Disposition
}

// Matrix is immutable once registered. VersionRef identifies a synthetic
// draft version; it is not an approved rule version.
type Matrix struct {
	ID         string
	TenantID   string
	VersionRef string
	Entries    []Entry
}

// Assessment is deliberately incapable of asserting readiness or authority.
type Assessment struct {
	RequirementRef        string
	Disposition           Disposition
	NonBinding            bool
	HumanDecisionRequired bool
}

// HistoryRecord is append-only synthetic attribution for matrix registration.
type HistoryRecord struct {
	TenantID   string
	MatrixID   string
	ActorRef   string
	OccurredAt time.Time
}

type matrixKey struct{ tenantID, matrixID string }
type versionKey struct{ tenantID, versionRef string }
type requirementKey struct{ tenantID, matrixID, requirementRef string }

// Registry is a thread-safe local fixture store with zero persistence or
// external effect.
type Registry struct {
	mu          sync.RWMutex
	matrices    map[matrixKey]Matrix
	versions    map[versionKey]struct{}
	assessments map[requirementKey]Assessment
	history     []HistoryRecord
}

func NewRegistry() *Registry {
	return &Registry{
		matrices:    make(map[matrixKey]Matrix),
		versions:    make(map[versionKey]struct{}),
		assessments: make(map[requirementKey]Assessment),
	}
}

func trim(value string) string { return strings.TrimSpace(value) }

func require(value string) error {
	if trim(value) == "" {
		return ErrBlankField
	}
	return nil
}

func hasPrefix(value, prefix string) bool {
	value = trim(value)
	return strings.HasPrefix(value, prefix) && len(value) > len(prefix)
}

func validDisposition(value Disposition) bool {
	_, ok := requiredDispositions[value]
	return ok
}

func cloneEntries(entries []Entry) []Entry {
	result := make([]Entry, len(entries))
	copy(result, entries)
	return result
}

// Register stores one immutable synthetic draft matrix. It requires all five
// fixed non-current cases and produces no readiness or authority outcome.
func (r *Registry) Register(matrix Matrix, actorRef string, at time.Time) error {
	if r == nil {
		return ErrAssessmentNotFound
	}
	for _, value := range []string{matrix.ID, matrix.TenantID, matrix.VersionRef, actorRef} {
		if err := require(value); err != nil {
			return err
		}
	}
	if !hasPrefix(matrix.ID, "mat_") || !hasPrefix(matrix.TenantID, "ten_") || !hasPrefix(matrix.VersionRef, "ver_") || !hasPrefix(actorRef, "sub_") {
		return ErrInvalidReference
	}
	coverage := make(map[Disposition]struct{})
	seenRequirements := make(map[string]struct{})
	entries := cloneEntries(matrix.Entries)
	for _, entry := range entries {
		if err := require(entry.RequirementRef); err != nil {
			return err
		}
		if !hasPrefix(entry.RequirementRef, "req_") || !validDisposition(entry.Disposition) {
			return ErrInvalidReference
		}
		ref := trim(entry.RequirementRef)
		if _, exists := seenRequirements[ref]; exists {
			return ErrDuplicateRequirement
		}
		seenRequirements[ref] = struct{}{}
		coverage[entry.Disposition] = struct{}{}
	}
	if len(coverage) != len(requiredDispositions) {
		return ErrIncompleteDispositionCoverage
	}

	matrix.ID, matrix.TenantID, matrix.VersionRef, matrix.Entries = trim(matrix.ID), trim(matrix.TenantID), trim(matrix.VersionRef), entries
	key := matrixKey{tenantID: matrix.TenantID, matrixID: matrix.ID}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.matrices[key]; exists {
		return ErrDuplicateMatrix
	}
	version := versionKey{tenantID: matrix.TenantID, versionRef: matrix.VersionRef}
	if _, exists := r.versions[version]; exists {
		return ErrDuplicateVersion
	}
	r.matrices[key] = matrix
	r.versions[version] = struct{}{}
	for _, entry := range matrix.Entries {
		r.assessments[requirementKey{tenantID: matrix.TenantID, matrixID: matrix.ID, requirementRef: trim(entry.RequirementRef)}] = Assessment{
			RequirementRef: trim(entry.RequirementRef), Disposition: entry.Disposition, NonBinding: true, HumanDecisionRequired: true,
		}
	}
	r.history = append(r.history, HistoryRecord{TenantID: matrix.TenantID, MatrixID: matrix.ID, ActorRef: trim(actorRef), OccurredAt: at.UTC()})
	return nil
}

// GetAssessment default-denies absent and cross-tenant access with one
// non-leaking error. Returned values do not assert readiness or authority.
func (r *Registry) GetAssessment(tenantID, matrixID, requirementRef string) (Assessment, error) {
	if r == nil {
		return Assessment{}, ErrAssessmentNotFound
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	assessment, ok := r.assessments[requirementKey{tenantID: trim(tenantID), matrixID: trim(matrixID), requirementRef: trim(requirementRef)}]
	if !ok {
		return Assessment{}, ErrAssessmentNotFound
	}
	return assessment, nil
}

// History returns a tenant-scoped copy of append-only matrix attribution.
func (r *Registry) History(tenantID string) []HistoryRecord {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]HistoryRecord, 0)
	for _, record := range r.history {
		if record.TenantID == trim(tenantID) {
			result = append(result, record)
		}
	}
	return result
}
