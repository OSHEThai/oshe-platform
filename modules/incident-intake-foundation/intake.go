// Package incidentintakefoundation provides a synthetic, in-memory incident
// intake boundary. It neither captures report content nor classifies, routes,
// escalates, notifies, or makes safety decisions.
package incidentintakefoundation

import (
	"errors"
	"strings"
	"sync"
	"time"
)

var (
	ErrBlankField       = errors.New("required field is blank")
	ErrInvalidReference = errors.New("reference must use its required opaque synthetic prefix")
	ErrDuplicateIntake  = errors.New("intake already exists in tenant")
	ErrIntakeNotFound   = errors.New("intake not found")
)

// State is deliberately restricted to the only state authorized before
// V050-CFG-006 selects binding classification and triage rules.
type State string

const Unclassified State = "UNCLASSIFIED"

// Intake is content-free metadata. ReporterRef and ActorRef are opaque
// synthetic references only; they are not identities, credentials, grants,
// evidence links, or authorization claims.
type Intake struct {
	ID                  string
	TenantID            string
	ReporterRef         string
	State               State
	HumanTriageRequired bool
}

// SubmissionHistory is append-only attribution for synthetic submissions.
type SubmissionHistory struct {
	TenantID   string
	IntakeID   string
	ActorRef   string
	OccurredAt time.Time
}

// Registry is a thread-safe local fixture store with zero persistent or
// external effects.
type Registry struct {
	mu      sync.RWMutex
	intakes map[string]Intake
	history []SubmissionHistory
}

func NewRegistry() *Registry { return &Registry{intakes: make(map[string]Intake)} }

func key(tenantID, intakeID string) string {
	return strings.TrimSpace(tenantID) + ":" + strings.TrimSpace(intakeID)
}

func require(value string) error {
	if strings.TrimSpace(value) == "" {
		return ErrBlankField
	}
	return nil
}

func hasPrefix(value, prefix string) bool {
	value = strings.TrimSpace(value)
	return strings.HasPrefix(value, prefix) && len(value) > len(prefix)
}

// Submit creates the sole permitted intake state. There is intentionally no
// API to classify, route, acknowledge, attach evidence, or change this state.
func (r *Registry) Submit(intakeID, tenantID, reporterRef, actorRef string, at time.Time) (Intake, error) {
	if r == nil {
		return Intake{}, ErrIntakeNotFound
	}
	for _, value := range []string{intakeID, tenantID, reporterRef, actorRef} {
		if err := require(value); err != nil {
			return Intake{}, err
		}
	}
	if !hasPrefix(intakeID, "int_") || !hasPrefix(tenantID, "ten_") || !hasPrefix(reporterRef, "ref_") || !hasPrefix(actorRef, "sub_") {
		return Intake{}, ErrInvalidReference
	}

	intake := Intake{
		ID:                  strings.TrimSpace(intakeID),
		TenantID:            strings.TrimSpace(tenantID),
		ReporterRef:         strings.TrimSpace(reporterRef),
		State:               Unclassified,
		HumanTriageRequired: true,
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.intakes[key(intake.TenantID, intake.ID)]; exists {
		return Intake{}, ErrDuplicateIntake
	}
	r.intakes[key(intake.TenantID, intake.ID)] = intake
	r.history = append(r.history, SubmissionHistory{
		TenantID:   intake.TenantID,
		IntakeID:   intake.ID,
		ActorRef:   strings.TrimSpace(actorRef),
		OccurredAt: at.UTC(),
	})
	return intake, nil
}

// Get default-denies absent and cross-tenant lookup with one non-leaking error.
func (r *Registry) Get(tenantID, intakeID string) (Intake, error) {
	if r == nil {
		return Intake{}, ErrIntakeNotFound
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	intake, ok := r.intakes[key(tenantID, intakeID)]
	if !ok {
		return Intake{}, ErrIntakeNotFound
	}
	return intake, nil
}

// History returns a copy of only the requested tenant's submission history.
func (r *Registry) History(tenantID string) []SubmissionHistory {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]SubmissionHistory, 0)
	for _, record := range r.history {
		if record.TenantID == strings.TrimSpace(tenantID) {
			result = append(result, record)
		}
	}
	return result
}
