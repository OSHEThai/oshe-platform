// Package orgtenancy provides organizational hierarchy and tenancy models for OSHE Platform.
//
// PROVISIONAL GOVERNANCE DECLARATION (H030-002 Deferred Gate):
// Under approved Sole Human Owner decision H030-002, this file establishes the
// local in-memory reversible party lifecycle state machine, sponsor reassignment simulation,
// project-closure cascade, and append-only historical attribution ledger.
//
// Zero binding operational authority, persistent database mutation, external provider
// integration, or runtime execution is claimed or enacted. Binding authority transitions
// remain strictly deferred pending successor owner gates (H030-007, H030-008).
// All operations operate as in-memory, side-effect-free, reversible simulation fixtures.
package orgtenancy

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

var (
	// ErrSponsorUnavailable indicates that the proposed internal sponsor is inactive, invalid, or unauthorized.
	ErrSponsorUnavailable = errors.New("proposed sponsor is unavailable or unauthorized")
	// ErrCrossTenantSponsor indicates an illegal attempt to assign a sponsor belonging to another tenant.
	ErrCrossTenantSponsor = errors.New("cannot assign sponsor belonging to another tenant")
	// ErrSponsorUnchanged indicates an attempt to reassign to the identical current sponsor.
	ErrSponsorUnchanged = errors.New("new sponsor is identical to existing sponsor")
	// ErrPartyDeactivated indicates that an operational action was attempted on a deactivated/archived party.
	ErrPartyDeactivated = errors.New("party is deactivated")
	// ErrParticipationClosed indicates that the participation relationship is closed.
	ErrParticipationClosed = errors.New("participation relationship is closed")
	// ErrProjectClosedCascade indicates that participation operations are denied because the parent project is closed.
	ErrProjectClosedCascade = errors.New("participation operation denied: parent project is closed")
)

// HistoricalPartyLifecycleRecord captures an immutable audit entry for a party lifecycle transition.
type HistoricalPartyLifecycleRecord struct {
	RecordID      string         `json:"record_id"`
	TenantID      string         `json:"tenant_id"`
	PartyID       string         `json:"party_id"`
	PreviousState LifecycleState `json:"previous_state"`
	NewState      LifecycleState `json:"new_state"`
	Transition    string         `json:"transition"`
	ActorSubject  string         `json:"actor_subject"`
	Reason        string         `json:"reason"`
	RecordedAt    time.Time      `json:"recorded_at"`
}

// HistoricalParticipationLifecycleRecord captures an immutable audit entry for a participation state transition.
type HistoricalParticipationLifecycleRecord struct {
	RecordID        string         `json:"record_id"`
	TenantID        string         `json:"tenant_id"`
	ParticipationID string         `json:"participation_id"`
	PartyID         string         `json:"party_id"`
	ProjectID       string         `json:"project_id"`
	SiteID          string         `json:"site_id,omitempty"`
	SponsorID       string         `json:"sponsor_id"`
	PreviousState   LifecycleState `json:"previous_state"`
	NewState        LifecycleState `json:"new_state"`
	Transition      string         `json:"transition"`
	ActorSubject    string         `json:"actor_subject"`
	Reason          string         `json:"reason"`
	RecordedAt      time.Time      `json:"recorded_at"`
}

// SponsorReassignmentRecord encapsulates an immutable audit entry for an internal sponsor reassignment event.
// Guarantees prior sponsor attribution is permanently preserved without hard deletion.
type SponsorReassignmentRecord struct {

	RecordID        string    `json:"record_id"`
	TenantID        string    `json:"tenant_id"`
	ParticipationID string    `json:"participation_id"`
	PriorSponsorID  string    `json:"prior_sponsor_id"`
	NewSponsorID    string    `json:"new_sponsor_id"`
	ActorSubject    string    `json:"actor_subject"`
	Reason          string    `json:"reason"`
	ReassignedAt    time.Time `json:"reassigned_at"`
}

// IsClosed returns true if the project participation is in CLOSED state.
func (pp ProjectParticipation) IsClosed() bool { return pp.state == StateClosed }

// DeactivateParty transitions a party from StateActive to StateArchived in memory.
// Returns the deactivated party and an immutable historical lifecycle record.
func DeactivateParty(party Party, actorSubject, reason string) (Party, HistoricalPartyLifecycleRecord, error) {
	if party.TenantID() == "" || party.PartyID() == "" {
		return Party{}, HistoricalPartyLifecycleRecord{}, ErrBlankID
	}
	if party.State() == StateArchived {
		return party, HistoricalPartyLifecycleRecord{}, ErrEntityArchived
	}

	prevState := party.State()
	deactivated := party.Archive()

	record := HistoricalPartyLifecycleRecord{
		RecordID:      fmt.Sprintf("hprt_%s_%d", party.PartyID(), time.Now().UTC().UnixNano()),
		TenantID:      party.TenantID(),
		PartyID:       party.PartyID(),
		PreviousState: prevState,
		NewState:      deactivated.State(),
		Transition:    "PARTY_DEACTIVATE",
		ActorSubject:  strings.TrimSpace(actorSubject),
		Reason:        strings.TrimSpace(reason),
		RecordedAt:    time.Now().UTC(),
	}

	return deactivated, record, nil
}

// DeactivateParticipation transitions a project participation from StateActive to StateArchived in memory.
func DeactivateParticipation(pp ProjectParticipation, actorSubject, reason string) (ProjectParticipation, HistoricalParticipationLifecycleRecord, error) {
	if pp.TenantID() == "" || pp.ParticipationID() == "" {
		return ProjectParticipation{}, HistoricalParticipationLifecycleRecord{}, ErrBlankID
	}
	if pp.State() == StateArchived {
		return pp, HistoricalParticipationLifecycleRecord{}, ErrEntityArchived
	}

	prevState := pp.State()
	deactivated := pp.Archive()

	record := HistoricalParticipationLifecycleRecord{
		RecordID:        fmt.Sprintf("hptp_%s_%d", pp.ParticipationID(), time.Now().UTC().UnixNano()),
		TenantID:        pp.TenantID(),
		ParticipationID: pp.ParticipationID(),
		PartyID:         pp.PartyID(),
		ProjectID:       pp.ProjectID(),
		SiteID:          pp.SiteID(),
		SponsorID:       pp.SponsorID(),
		PreviousState:   prevState,
		NewState:        deactivated.State(),
		Transition:      "PARTICIPATION_DEACTIVATE",
		ActorSubject:    strings.TrimSpace(actorSubject),
		Reason:          strings.TrimSpace(reason),
		RecordedAt:      time.Now().UTC(),
	}

	return deactivated, record, nil
}

// CloseParticipation transitions a project participation to StateClosed in memory.
func CloseParticipation(pp ProjectParticipation, actorSubject, reason string) (ProjectParticipation, HistoricalParticipationLifecycleRecord, error) {
	if pp.TenantID() == "" || pp.ParticipationID() == "" {
		return ProjectParticipation{}, HistoricalParticipationLifecycleRecord{}, ErrBlankID
	}
	if pp.State() == StateArchived {
		return ProjectParticipation{}, HistoricalParticipationLifecycleRecord{}, ErrEntityArchived
	}
	if pp.State() == StateClosed {
		return pp, HistoricalParticipationLifecycleRecord{}, ErrParticipationClosed
	}

	prevState := pp.State()
	closed, err := pp.Close()
	if err != nil {
		return ProjectParticipation{}, HistoricalParticipationLifecycleRecord{}, err
	}

	record := HistoricalParticipationLifecycleRecord{
		RecordID:        fmt.Sprintf("hptp_%s_%d", pp.ParticipationID(), time.Now().UTC().UnixNano()),
		TenantID:        pp.TenantID(),
		ParticipationID: pp.ParticipationID(),
		PartyID:         pp.PartyID(),
		ProjectID:       pp.ProjectID(),
		SiteID:          pp.SiteID(),
		SponsorID:       pp.SponsorID(),
		PreviousState:   prevState,
		NewState:        closed.State(),
		Transition:      "PARTICIPATION_CLOSE",
		ActorSubject:    strings.TrimSpace(actorSubject),
		Reason:          strings.TrimSpace(reason),
		RecordedAt:      time.Now().UTC(),
	}

	return closed, record, nil
}

// ReassignSponsor updates the internal sponsor manager for an active project participation.
//
// Invariants enforced under H030-002:
// 1. Participation must be in StateActive and not expired at timestamp 'at'.
// 2. New sponsor ID must be non-blank and reference an internal user (usr_*).
// 3. New sponsor ID cannot be identical to the current sponsor ID (ErrSponsorUnchanged).
// 4. Returns an immutable SponsorReassignmentRecord preserving prior sponsor attribution.
// 5. Preserves all other participation attributes (party, project, site, role, valid window, nesting).
func ReassignSponsor(pp ProjectParticipation, newSponsorID, actorSubject, reason string, at time.Time) (ProjectParticipation, SponsorReassignmentRecord, error) {
	if pp.TenantID() == "" || pp.ParticipationID() == "" {
		return ProjectParticipation{}, SponsorReassignmentRecord{}, ErrBlankID
	}
	if !pp.IsActive() {
		if pp.State() == StateClosed {
			return ProjectParticipation{}, SponsorReassignmentRecord{}, ErrParticipationClosed
		}
		return ProjectParticipation{}, SponsorReassignmentRecord{}, ErrParentNotActive
	}
	if !pp.IsValidAt(at) {
		return ProjectParticipation{}, SponsorReassignmentRecord{}, ErrParticipationExpired
	}

	trimmedSponsor := strings.TrimSpace(newSponsorID)
	if err := ValidateSponsorID(trimmedSponsor); err != nil {
		return ProjectParticipation{}, SponsorReassignmentRecord{}, err
	}

	if trimmedSponsor == pp.SponsorID() {
		return ProjectParticipation{}, SponsorReassignmentRecord{}, ErrSponsorUnchanged
	}

	priorSponsor := pp.SponsorID()

	// Construct updated participation copy with new sponsor
	updated := ProjectParticipation{
		participationID:       pp.ParticipationID(),
		tenantID:              pp.TenantID(),
		partyID:               pp.PartyID(),
		projectID:             pp.ProjectID(),
		siteID:                pp.SiteID(),
		sponsorID:             trimmedSponsor,
		role:                  pp.Role(),
		validFrom:             pp.ValidFrom(),
		validTo:               pp.ValidTo(),
		state:                 pp.State(),
		parentParticipationID: pp.ParentParticipationID(),
		nestingDepth:          pp.NestingDepth(),
	}

	record := SponsorReassignmentRecord{
		RecordID:        fmt.Sprintf("sps_%s_%d", pp.ParticipationID(), time.Now().UTC().UnixNano()),
		TenantID:        pp.TenantID(),
		ParticipationID: pp.ParticipationID(),
		PriorSponsorID:  priorSponsor,
		NewSponsorID:    trimmedSponsor,
		ActorSubject:    strings.TrimSpace(actorSubject),
		Reason:          strings.TrimSpace(reason),
		ReassignedAt:    at.UTC(),
	}

	return updated, record, nil
}

// CascadeProjectClosure safely cascades project closure to all bounded contractor and subcontractor
// participations in memory.
//
// Invariants enforced:
// 1. Project must be in StateClosed.
// 2. All active participations in the project are transitioned to StateClosed.
// 3. Participations already archived remain StateArchived.
// 4. Returns updated participations and immutable audit records for each transitioned entity.
func CascadeProjectClosure(project Project, participations []ProjectParticipation, actorSubject, reason string) ([]ProjectParticipation, []HistoricalParticipationLifecycleRecord, error) {
	if project.TenantID() == "" || project.ProjectID() == "" {
		return nil, nil, ErrBlankID
	}
	if !project.IsClosed() {
		return nil, nil, errors.New("cannot cascade closure from non-closed project")
	}

	var updatedList []ProjectParticipation
	var auditRecords []HistoricalParticipationLifecycleRecord

	for _, pp := range participations {
		// Only process participations matching the project and tenant
		if pp.TenantID() != project.TenantID() || pp.ProjectID() != project.ProjectID() {
			updatedList = append(updatedList, pp)
			continue
		}

		if pp.IsActive() {
			prevState := pp.State()
			closed, err := pp.Close()
			if err != nil {
				return nil, nil, err
			}

			record := HistoricalParticipationLifecycleRecord{
				RecordID:        fmt.Sprintf("casc_%s_%d", pp.ParticipationID(), time.Now().UTC().UnixNano()),
				TenantID:        pp.TenantID(),
				ParticipationID: pp.ParticipationID(),
				PartyID:         pp.PartyID(),
				ProjectID:       pp.ProjectID(),
				SiteID:          pp.SiteID(),
				SponsorID:       pp.SponsorID(),
				PreviousState:   prevState,
				NewState:        closed.State(),
				Transition:      "PROJECT_CLOSURE_CASCADE",
				ActorSubject:    strings.TrimSpace(actorSubject),
				Reason:          strings.TrimSpace(reason),
				RecordedAt:      time.Now().UTC(),
			}

			updatedList = append(updatedList, closed)
			auditRecords = append(auditRecords, record)
		} else {
			// Already closed or archived participations retain state
			updatedList = append(updatedList, pp)
		}
	}

	return updatedList, auditRecords, nil
}

// ValidateParticipationAgainstProject evaluates whether an operational action on a participation
// is permitted given the parent project's current state. Fails closed if the project is closed or archived.
func ValidateParticipationAgainstProject(pp ProjectParticipation, project Project, at time.Time) error {
	if pp.TenantID() != project.TenantID() {
		return ErrCrossTenantLinkage
	}
	if pp.ProjectID() != project.ProjectID() {
		return ErrScopeMismatch
	}

	// Project state check
	if project.IsClosed() {
		return ErrProjectClosedCascade
	}
	if project.State() == StateArchived {
		return ErrParentArchived
	}
	if !project.IsActive() {
		return ErrParentNotActive
	}

	// Participation validity check
	if !pp.IsActive() {
		if pp.State() == StateClosed {
			return ErrParticipationClosed
		}
		return ErrParentNotActive
	}
	if !pp.IsValidAt(at) {
		return ErrParticipationExpired
	}

	return nil
}

// SimulateReversiblePartyState allows test fixtures to exercise forward and reverse state transitions
// on a Party in memory without enacting external authority (H030-002).
func SimulateReversiblePartyState(p Party, targetState LifecycleState) (Party, error) {
	switch targetState {
	case StateActive, StateClosed, StateArchived:
		p.state = targetState
		return p, nil
	default:
		return p, fmt.Errorf("unrecognized target party state: %s", targetState)
	}
}

// PartyLifecycleLedger provides a thread-safe in-memory append-only audit ledger for party
// lifecycle events and sponsor reassignments. Enforces tenant boundary isolation on all queries.
type PartyLifecycleLedger struct {
	mu             sync.RWMutex
	partyRecords   []HistoricalPartyLifecycleRecord
	partRecords    []HistoricalParticipationLifecycleRecord
	sponsorRecords []SponsorReassignmentRecord
}

// NewPartyLifecycleLedger initializes an empty in-memory PartyLifecycleLedger.
func NewPartyLifecycleLedger() *PartyLifecycleLedger {
	return &PartyLifecycleLedger{
		partyRecords:   make([]HistoricalPartyLifecycleRecord, 0),
		partRecords:    make([]HistoricalParticipationLifecycleRecord, 0),
		sponsorRecords: make([]SponsorReassignmentRecord, 0),
	}
}

// AppendPartyRecord records a party lifecycle transition.
func (l *PartyLifecycleLedger) AppendPartyRecord(record HistoricalPartyLifecycleRecord) error {
	if record.TenantID == "" || record.PartyID == "" {
		return ErrBlankID
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.partyRecords = append(l.partyRecords, record)
	return nil
}

// AppendParticipationRecord records a participation lifecycle transition.
func (l *PartyLifecycleLedger) AppendParticipationRecord(record HistoricalParticipationLifecycleRecord) error {
	if record.TenantID == "" || record.ParticipationID == "" {
		return ErrBlankID
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.partRecords = append(l.partRecords, record)
	return nil
}

// AppendSponsorReassignment records an internal sponsor reassignment event.
func (l *PartyLifecycleLedger) AppendSponsorReassignment(record SponsorReassignmentRecord) error {
	if record.TenantID == "" || record.ParticipationID == "" {
		return ErrBlankID
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.sponsorRecords = append(l.sponsorRecords, record)
	return nil
}

// GetPartyHistory returns the append-only lifecycle history for a party under a tenant.
func (l *PartyLifecycleLedger) GetPartyHistory(tenantID, partyID string) ([]HistoricalPartyLifecycleRecord, error) {
	tTenant := strings.TrimSpace(tenantID)
	if tTenant == "" {
		return nil, ErrBlankTenantID
	}
	tParty := strings.TrimSpace(partyID)
	if tParty == "" {
		return nil, ErrBlankID
	}

	l.mu.RLock()
	defer l.mu.RUnlock()

	var results []HistoricalPartyLifecycleRecord
	for _, rec := range l.partyRecords {
		if rec.TenantID == tTenant && rec.PartyID == tParty {
			results = append(results, rec)
		}
	}
	return results, nil
}

// GetParticipationHistory returns the append-only lifecycle history for a participation under a tenant.
func (l *PartyLifecycleLedger) GetParticipationHistory(tenantID, participationID string) ([]HistoricalParticipationLifecycleRecord, error) {
	tTenant := strings.TrimSpace(tenantID)
	if tTenant == "" {
		return nil, ErrBlankTenantID
	}
	tPart := strings.TrimSpace(participationID)
	if tPart == "" {
		return nil, ErrBlankID
	}

	l.mu.RLock()
	defer l.mu.RUnlock()

	var results []HistoricalParticipationLifecycleRecord
	for _, rec := range l.partRecords {
		if rec.TenantID == tTenant && rec.ParticipationID == tPart {
			results = append(results, rec)
		}
	}
	return results, nil
}

// GetSponsorHistory returns all past and current sponsor reassignment records for a participation.
func (l *PartyLifecycleLedger) GetSponsorHistory(tenantID, participationID string) ([]SponsorReassignmentRecord, error) {
	tTenant := strings.TrimSpace(tenantID)
	if tTenant == "" {
		return nil, ErrBlankTenantID
	}
	tPart := strings.TrimSpace(participationID)
	if tPart == "" {
		return nil, ErrBlankID
	}

	l.mu.RLock()
	defer l.mu.RUnlock()

	var results []SponsorReassignmentRecord
	for _, rec := range l.sponsorRecords {
		if rec.TenantID == tTenant && rec.ParticipationID == tPart {
			results = append(results, rec)
		}
	}
	return results, nil
}
