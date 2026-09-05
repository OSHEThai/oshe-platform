// Package orgtenancy provides organizational hierarchy and tenancy models for OSHE Platform.
//
// PROVISIONAL GOVERNANCE DECLARATION (H030-002 Deferred Gate):
// Under approved Sole Human Owner decision H030-002 and Milestone v0.3.0 boundaries,
// the lifecycle state-machine models, closure mechanisms, move operations, and historical
// attribution fixtures in this file provide local-simulation, preflight validation, and
// test harness capabilities only.
//
// Zero binding operational authority, runtime execution, database persistence mutation,
// or final operational completion is claimed or enacted. Binding authority transitions
// remain strictly deferred pending successor owner gates (H030-007, H030-008).
// All close, archive, and move operations operate as in-memory, side-effect-free,
// immutable projections with preflight validation semantics.
package orgtenancy

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// StateClosed represents the closed lifecycle state within the local v0.3 simulation harness.
// A closed entity remains queryable for historical audit and compliance records,
// but strictly rejects staging of new active operational records, inspections, or child entities.
const StateClosed LifecycleState = "CLOSED"

var (
	// ErrEntityClosed indicates the target entity is closed and unavailable for active operations.
	ErrEntityClosed = errors.New("entity is closed")
	// ErrParentClosed indicates the parent entity is closed, preventing child attachment or mutation.
	ErrParentClosed = errors.New("parent entity is closed")
	// ErrCannotReopenClosed indicates that operational policy in v0.3 treats closed as non-reactivatable without simulation harness.
	ErrCannotReopenClosed = errors.New("closed state is terminal for v0.3 operational policy: reopening is prohibited")
	// ErrCrossTenantMove indicates an illegal attempt to move or re-parent an entity across tenant boundaries.
	ErrCrossTenantMove = errors.New("cannot move entity across tenant boundaries")
	// ErrInvalidMoveTarget indicates the destination parent is not in an active operational state.
	ErrInvalidMoveTarget = errors.New("invalid move target: destination must be active")
	// ErrActiveRecordRejected indicates an operational mutation was rejected because the entity is not active.
	ErrActiveRecordRejected = errors.New("active operational record rejected: entity is not active")
)

// IsClosed returns true if the company is in CLOSED state.
func (c Company) IsClosed() bool { return c.state == StateClosed }

// Close returns an in-memory immutable copy of the company in CLOSED state (preflight simulation semantics).
func (c Company) Close() (Company, error) {
	if c.state == StateArchived {
		return Company{}, ErrEntityArchived
	}
	c.state = StateClosed
	return c, nil
}

// IsClosed returns true if the business unit is in CLOSED state.
func (b BusinessUnit) IsClosed() bool { return b.state == StateClosed }

// Close returns an in-memory immutable copy of the business unit in CLOSED state (preflight simulation semantics).
func (b BusinessUnit) Close() (BusinessUnit, error) {
	if b.state == StateArchived {
		return BusinessUnit{}, ErrEntityArchived
	}
	b.state = StateClosed
	return b, nil
}

// IsClosed returns true if the project is in CLOSED state.
func (p Project) IsClosed() bool { return p.state == StateClosed }

// Close returns an in-memory immutable copy of the project in CLOSED state (preflight simulation semantics).
func (p Project) Close() (Project, error) {
	if p.state == StateArchived {
		return Project{}, ErrEntityArchived
	}
	p.state = StateClosed
	return p, nil
}

// IsClosed returns true if the site is in CLOSED state.
func (s Site) IsClosed() bool { return s.state == StateClosed }

// Close returns an in-memory immutable copy of the site in CLOSED state (preflight simulation semantics).
func (s Site) Close() (Site, error) {
	if s.state == StateArchived {
		return Site{}, ErrEntityArchived
	}
	s.state = StateClosed
	return s, nil
}

// IsClosed returns true if the area is in CLOSED state.
func (a Area) IsClosed() bool { return a.state == StateClosed }

// Close returns an in-memory immutable copy of the area in CLOSED state (preflight simulation semantics).
func (a Area) Close() (Area, error) {
	if a.state == StateArchived {
		return Area{}, ErrEntityArchived
	}
	a.state = StateClosed
	return a, nil
}

// IsClosed returns true if the sponsored party is in CLOSED state.
func (sp SponsoredParty) IsClosed() bool { return sp.state == StateClosed }

// Close returns an in-memory immutable copy of the sponsored party in CLOSED state (preflight simulation semantics).
func (sp SponsoredParty) Close() (SponsoredParty, error) {
	if sp.state == StateArchived {
		return SponsoredParty{}, ErrEntityArchived
	}
	sp.state = StateClosed
	return sp, nil
}

// AssertOperationalActive validates that an entity's lifecycle state permits active operational mutations.
// Fails closed if the entity is closed or archived.
func AssertOperationalActive(state LifecycleState) error {
	switch state {
	case StateActive:
		return nil
	case StateClosed:
		return fmt.Errorf("%w: entity state is %s", ErrEntityClosed, state)
	case StateArchived:
		return fmt.Errorf("%w: entity state is %s", ErrEntityArchived, state)
	default:
		return fmt.Errorf("%w: unknown or unapproved lifecycle state %q", ErrActiveRecordRejected, state)
	}
}

// AssertParentOperationalActive validates that a parent entity is active before allowing child attachment.
func AssertParentOperationalActive(parentState LifecycleState) error {
	switch parentState {
	case StateActive:
		return nil
	case StateClosed:
		return ErrParentClosed
	case StateArchived:
		return ErrParentArchived
	default:
		return ErrParentMismatch
	}
}

// ReopenEntity checks operational policy: in standard operational posture, reopening CLOSED is prohibited.
func ReopenEntity(state LifecycleState) error {
	if state == StateClosed {
		return ErrCannotReopenClosed
	}
	return errors.New("entity is not closed")
}

// SimulateReversibleTransition provides a local simulation harness method allowing
// test fixtures to simulate forward and reverse lifecycle state transitions
// (e.g. ACTIVE <-> CLOSED <-> ARCHIVED) in memory without enacting external authority.
// This directly implements H030-002 local lifecycle simulation requirements.
func SimulateReversibleTransition(currentState, targetState LifecycleState) (LifecycleState, error) {
	switch targetState {
	case StateActive, StateClosed, StateArchived:
		return targetState, nil
	default:
		return currentState, fmt.Errorf("unrecognized target simulation state: %s", targetState)
	}
}

// MoveSiteToProject simulates safely re-parenting a Site to a new Project in memory.
// Preflight Invariants enforced:
// 1. Same-tenant move only: destination project must belong to the exact same tenant (fails with ErrCrossTenantMove).
// 2. Active destination only: destination project must be StateActive (rejects archived with ErrParentArchived, closed with ErrParentClosed).
// 3. Source active only: source site must be StateActive (cannot move archived or closed sites).
// 4. Attribution preservation: siteID, name, timeZone, locale, and state are preserved exactly without hard deletion.
// 5. Pure preflight simulation: zero external database mutation or runtime executor invocation.
func MoveSiteToProject(site Site, destProject Project) (Site, error) {
	if site.TenantID() == "" || site.SiteID() == "" {
		return Site{}, ErrBlankID
	}
	if destProject.TenantID() == "" || destProject.ProjectID() == "" {
		return Site{}, ErrBlankID
	}

	// 1. Cross-tenant check
	if site.TenantID() != destProject.TenantID() {
		return Site{}, ErrCrossTenantMove
	}

	// 2. Source state check
	if site.State() == StateArchived {
		return Site{}, ErrEntityArchived
	}
	if site.State() == StateClosed {
		return Site{}, ErrEntityClosed
	}
	if !site.IsActive() {
		return Site{}, ErrActiveRecordRejected
	}

	// 3. Destination state check
	if err := AssertParentOperationalActive(destProject.State()); err != nil {
		return Site{}, err
	}

	// 4. Construct re-parented site preserving identity, locale, and attribution
	return Site{
		tenantID:       destProject.TenantID(),
		companyID:      destProject.CompanyID(),
		businessUnitID: destProject.BusinessUnitID(),
		projectID:      destProject.ProjectID(),
		siteID:         site.SiteID(),
		name:           site.Name(),
		timeZone:       site.TimeZone(),
		locale:         site.Locale(),
		state:          site.State(),
	}, nil
}

// MoveAreaToSite simulates safely re-parenting an Area to a new Site within the same tenant in memory.
// Preflight Invariants enforced:
// 1. Same-tenant move only (fails with ErrCrossTenantMove).
// 2. Active destination only (destination site must be StateActive).
// 3. Source active only (cannot move archived or closed area).
// 4. Attribution preservation: areaID, name, timeZone, locale, and state are preserved.
func MoveAreaToSite(area Area, destSite Site) (Area, error) {
	if area.TenantID() == "" || area.AreaID() == "" {
		return Area{}, ErrBlankID
	}
	if destSite.TenantID() == "" || destSite.SiteID() == "" {
		return Area{}, ErrBlankID
	}

	// 1. Cross-tenant check
	if area.TenantID() != destSite.TenantID() {
		return Area{}, ErrCrossTenantMove
	}

	// 2. Source state check
	if area.State() == StateArchived {
		return Area{}, ErrEntityArchived
	}
	if area.State() == StateClosed {
		return Area{}, ErrEntityClosed
	}
	if !area.IsActive() {
		return Area{}, ErrActiveRecordRejected
	}

	// 3. Destination state check
	if err := AssertParentOperationalActive(destSite.State()); err != nil {
		return Area{}, err
	}

	// 4. Construct re-parented area
	return Area{
		tenantID:       destSite.TenantID(),
		companyID:      destSite.CompanyID(),
		businessUnitID: destSite.BusinessUnitID(),
		projectID:      destSite.ProjectID(),
		siteID:         destSite.SiteID(),
		areaID:         area.AreaID(),
		name:           area.Name(),
		timeZone:       area.TimeZone(),
		locale:         area.Locale(),
		state:          area.State(),
	}, nil
}

// HistoricalScopeRecord encapsulates a frozen, immutable snapshot of an entity's resolved scope
// at a specific historical lifecycle event (such as Close, Archive, or Move).
// This guarantees historical attribution preservation without hard deletion.
type HistoricalScopeRecord struct {
	RecordID      string         `json:"record_id"`
	RecordedAt    time.Time      `json:"recorded_at"`
	EntityID      string         `json:"entity_id"`
	Scope         ResolvedScope  `json:"scope"`
	State         LifecycleState `json:"state"`
	Transition    string         `json:"transition"`
	ActorSubject  string         `json:"actor_subject"`
	Reason        string         `json:"reason"`
}

// CaptureHistoricalScope creates an immutable historical scope record for an entity transition.
// It is an in-memory audit projection preserving attribution across local simulations.
func CaptureHistoricalScope(entityID string, scope ResolvedScope, state LifecycleState, transition, actorSubject, reason string) HistoricalScopeRecord {
	trimmedEntityID := strings.TrimSpace(entityID)
	trimmedTransition := strings.TrimSpace(transition)
	trimmedActor := strings.TrimSpace(actorSubject)
	trimmedReason := strings.TrimSpace(reason)

	return HistoricalScopeRecord{
		RecordID:     fmt.Sprintf("hist_%s_%d", trimmedEntityID, time.Now().UTC().UnixNano()),
		RecordedAt:   time.Now().UTC(),
		EntityID:     trimmedEntityID,
		Scope:        scope,
		State:        state,
		Transition:   trimmedTransition,
		ActorSubject: trimmedActor,
		Reason:       trimmedReason,
	}
}
