// Package orgtenancy provides organizational hierarchy and tenancy models for OSHE Platform.
//
// PROVISIONAL GOVERNANCE DECLARATION (H030-002 Deferred Gate):
// Under approved Sole Human Owner decision H030-002, this file establishes the
// local-simulation and preflight validation boundaries for external party identity,
// project participation, and contractor/subcontractor nesting.
//
// Zero binding operational contractor authority, persistent database mutation,
// external identity provider synchronization, or runtime execution is claimed or enacted.
// All nesting and participation models operate as local, reversible, in-memory representations.
package orgtenancy

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// MaxContractorNestingDepth defines the strict ceiling on contractor-to-subcontractor nesting.
// Under H030-002, only minimal depth 1 (Contractor -> Subcontractor) is permitted.
// Deeper nesting (sub-subcontractors at depth 2+) is strictly prohibited.
const MaxContractorNestingDepth = 1

// PartyType classifies an external party's operational relationship to the tenant.
type PartyType string

const (
	PartyTypeClient        PartyType = "CLIENT"
	PartyTypeContractor    PartyType = "CONTRACTOR"
	PartyTypeSubcontractor PartyType = "SUBCONTRACTOR"
	PartyTypePartner       PartyType = "PARTNER"
	PartyTypeAuditor       PartyType = "AUDITOR"
)

// ParticipationRole defines the authorized, non-administrative role held by a party within a project.
type ParticipationRole string

const (
	ParticipationRoleContractorWorker  ParticipationRole = "CONTRACTOR_WORKER"
	ParticipationRoleSiteSafetyLead    ParticipationRole = "SITE_SAFETY_LEAD"
	ParticipationRoleClientAuditor     ParticipationRole = "CLIENT_AUDITOR"
	ParticipationRoleConsultant        ParticipationRole = "CONSULTANT"
	ParticipationRoleSubcontractorLead ParticipationRole = "SUBCONTRACTOR_LEAD"
)

var (
	// ErrInvalidPartyType indicates an unapproved or blank party type.
	ErrInvalidPartyType = errors.New("invalid or unrecognized party type")
	// ErrInvalidParticipationRole indicates an unapproved or blank participation role.
	ErrInvalidParticipationRole = errors.New("invalid or unrecognized participation role")
	// ErrScopeMismatch indicates an operation attempted outside authorized project or site bounds.
	ErrScopeMismatch = errors.New("operation outside authorized participation project or site scope")
	// ErrParticipationExpired indicates the participation validity window has ended.
	ErrParticipationExpired = errors.New("project participation has expired")
	// ErrParticipationNotYetValid indicates the participation validity window has not yet begun.
	ErrParticipationNotYetValid = errors.New("project participation is not yet active")
	// ErrPartyArchived indicates an attempt to create participation under an archived party.
	ErrPartyArchived = errors.New("cannot create participation under an archived party")
	// ErrPartyNotFound indicates the requested party does not exist.
	ErrPartyNotFound = errors.New("party not found in registry")
	// ErrParticipationNotFound indicates the requested participation does not exist.
	ErrParticipationNotFound = errors.New("participation not found in registry")
	// ErrDuplicateParty indicates a party with the given ID already exists in the tenant.
	ErrDuplicateParty = errors.New("party ID already registered for tenant")
	// ErrDuplicateParticipation indicates a participation with the given ID already exists.
	ErrDuplicateParticipation = errors.New("participation ID already registered for tenant")
	// ErrNestingDepthExceeded indicates an attempt to nest contractors beyond depth 1.
	ErrNestingDepthExceeded = errors.New("unauthorized contractor nesting depth: maximum allowed depth is 1 (subcontractor)")
	// ErrValidityWindowExceedsParent indicates a subcontractor validity window extends outside parent contractor window.
	ErrValidityWindowExceedsParent = errors.New("nested participation validity window cannot exceed parent contractor validity window")
	// ErrElevationForbidden indicates an illegal attempt to grant internal corporate administrative authority to a contractor.
	ErrElevationForbidden = errors.New("contractor/subcontractor elevation to internal administrative authority is strictly prohibited")
	// ErrParentNotActive indicates that the parent contractor participation is not in active state.
	ErrParentNotActive = errors.New("parent contractor participation is not active")
	// ErrSiblingAccessDenied indicates an attempt to access a lateral sibling contractor or project scope.
	ErrSiblingAccessDenied = errors.New("cross-sibling or lateral contractor scope access is strictly denied")
	// ErrInvalidSponsorID indicates that the internal sponsor ID is missing or lacks the required user prefix.
	ErrInvalidSponsorID = errors.New("internal sponsor identifier must be non-blank and reference an internal user identity")
)

// ValidatePartyType verifies that a party type is one of the approved enumerations.
func ValidatePartyType(pt PartyType) error {
	switch pt {
	case PartyTypeClient, PartyTypeContractor, PartyTypeSubcontractor, PartyTypePartner, PartyTypeAuditor:
		return nil
	default:
		return ErrInvalidPartyType
	}
}

// ValidateParticipationRole verifies that a role is one of the approved enumerations.
func ValidateParticipationRole(role ParticipationRole) error {
	switch role {
	case ParticipationRoleContractorWorker, ParticipationRoleSiteSafetyLead, ParticipationRoleClientAuditor,
		ParticipationRoleConsultant, ParticipationRoleSubcontractorLead:
		return nil
	default:
		return ErrInvalidParticipationRole
	}
}

// ValidateSponsorID asserts that the mandatory internal sponsor identifier is non-blank
// and references an internal user identity.
func ValidateSponsorID(sponsorID string) error {
	trimmed := strings.TrimSpace(sponsorID)
	if trimmed == "" {
		return ErrBlankSponsorID
	}
	if !strings.HasPrefix(trimmed, "usr_") && !strings.HasPrefix(trimmed, "usr-") && !strings.HasPrefix(trimmed, "user-") {
		return ErrInvalidSponsorID
	}
	return nil
}

// AssertNoInternalAuthority validates that a contractor or subcontractor role does not confer
// or attempt to escalate into internal company, business unit, or tenant administrative authority.
func AssertNoInternalAuthority(role ParticipationRole) error {
	upper := strings.ToUpper(string(role))
	if strings.Contains(upper, "ADMIN") || strings.Contains(upper, "SUPER") ||
		strings.Contains(upper, "COMPANY") || strings.Contains(upper, "BUSINESS_UNIT") ||
		strings.Contains(upper, "TENANT_OWNER") {
		return ErrElevationForbidden
	}
	return ValidateParticipationRole(role)
}

// Party represents an external client, contractor, subcontractor, or partner entity.
// It is strictly tenant-scoped and grants zero internal organization or company administrative rights.
type Party struct {
	tenantID  string
	partyID   string
	name      string
	partyType PartyType
	state     LifecycleState
}

// TenantID returns the authoritative tenant identifier.
func (p Party) TenantID() string { return p.tenantID }

// PartyID returns the authoritative canonical party identifier (prt_*).
func (p Party) PartyID() string { return p.partyID }

// Name returns the legal/display name of the external party.
func (p Party) Name() string { return p.name }

// PartyType returns the party classification.
func (p Party) PartyType() PartyType { return p.partyType }

// State returns the operational lifecycle state.
func (p Party) State() LifecycleState { return p.state }

// IsActive returns true if the party is active.
func (p Party) IsActive() bool { return p.state == StateActive }

// Archive returns an immutable copy of the party in ARCHIVED state (preserving attribution, no hard deletion).
func (p Party) Archive() Party {
	p.state = StateArchived
	return p
}

// ValidateScope confirms the party matches the trusted tenant context.
func (p Party) ValidateScope(ctx TenantContext) error {
	return ctx.AuthorizeTenantScope(p.tenantID)
}

// NewParty constructs and validates a new external Party.
func NewParty(tenantID, partyID, name string, partyType PartyType) (Party, error) {
	trimmedTenant := strings.TrimSpace(tenantID)
	if trimmedTenant == "" {
		return Party{}, ErrBlankTenantID
	}
	trimmedPartyID := strings.TrimSpace(partyID)
	if trimmedPartyID == "" {
		return Party{}, ErrBlankID
	}
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return Party{}, ErrBlankName
	}
	if err := ValidatePartyType(partyType); err != nil {
		return Party{}, err
	}

	return Party{
		tenantID:  trimmedTenant,
		partyID:   trimmedPartyID,
		name:      trimmedName,
		partyType: partyType,
		state:     StateActive,
	}, nil
}

// ProjectParticipation models an external party's bounded, time-limited participation in a specific project,
// with optional subcontractor nesting under a primary contractor.
type ProjectParticipation struct {
	participationID       string
	tenantID              string
	partyID               string
	projectID             string
	siteID                string // optional bound to specific site within project
	sponsorID             string // internal manager subject (usr_*)
	role                  ParticipationRole
	validFrom             time.Time
	validTo               time.Time
	state                 LifecycleState
	parentParticipationID string // non-empty if nested subcontractor
	nestingDepth          int    // 0 = primary contractor, 1 = subcontractor
}

// ParticipationID returns the authoritative canonical participation identifier (ptp_*).
func (pp ProjectParticipation) ParticipationID() string { return pp.participationID }

// TenantID returns the authoritative tenant identifier.
func (pp ProjectParticipation) TenantID() string { return pp.tenantID }

// PartyID returns the participating external party identifier (prt_*).
func (pp ProjectParticipation) PartyID() string { return pp.partyID }

// ProjectID returns the bounded project identifier (prj_*).
func (pp ProjectParticipation) ProjectID() string { return pp.projectID }

// SiteID returns the bounded site identifier if set (ste_* or empty).
func (pp ProjectParticipation) SiteID() string { return pp.siteID }

// SponsorID returns the internal sponsor manager subject ID (usr_*).
func (pp ProjectParticipation) SponsorID() string { return pp.sponsorID }

// Role returns the participation role.
func (pp ProjectParticipation) Role() ParticipationRole { return pp.role }

// ValidFrom returns the beginning of the validity window.
func (pp ProjectParticipation) ValidFrom() time.Time { return pp.validFrom }

// ValidTo returns the end of the validity window.
func (pp ProjectParticipation) ValidTo() time.Time { return pp.validTo }

// State returns the operational lifecycle state.
func (pp ProjectParticipation) State() LifecycleState { return pp.state }

// IsActive returns true if the participation is in ACTIVE state.
func (pp ProjectParticipation) IsActive() bool { return pp.state == StateActive }

// ParentParticipationID returns the parent contractor's participation ID if nested.
func (pp ProjectParticipation) ParentParticipationID() string { return pp.parentParticipationID }

// NestingDepth returns 0 for direct contractors, 1 for subcontractors.
func (pp ProjectParticipation) NestingDepth() int { return pp.nestingDepth }

// IsSubcontractor returns true if this participation is nested under a primary contractor.
func (pp ProjectParticipation) IsSubcontractor() bool { return pp.parentParticipationID != "" }

// Archive returns an immutable copy of the participation in ARCHIVED state (no hard deletion).
func (pp ProjectParticipation) Archive() ProjectParticipation {
	pp.state = StateArchived
	return pp
}

// Close returns an immutable copy of the participation in CLOSED state (preflight simulation semantics).
func (pp ProjectParticipation) Close() (ProjectParticipation, error) {
	if pp.state == StateArchived {
		return ProjectParticipation{}, ErrEntityArchived
	}
	pp.state = StateClosed
	return pp, nil
}

// ValidateScope confirms the participation matches the trusted tenant context.
func (pp ProjectParticipation) ValidateScope(ctx TenantContext) error {
	return ctx.AuthorizeTenantScope(pp.tenantID)
}

// IsValidAt checks if the participation is active and within its effective temporal window.
func (pp ProjectParticipation) IsValidAt(t time.Time) bool {
	if !pp.IsActive() {
		return false
	}
	return !t.Before(pp.validFrom) && !t.After(pp.validTo)
}

// ValidateScopeAt validates that a requested action is within the participation's authorized
// project, optional site bounds, and active temporal window. Fails closed by default.
func (pp ProjectParticipation) ValidateScopeAt(projectID, siteID string, at time.Time) error {
	if !pp.IsActive() {
		return ErrEntityArchived
	}
	if at.Before(pp.validFrom) {
		return ErrParticipationNotYetValid
	}
	if at.After(pp.validTo) {
		return ErrParticipationExpired
	}

	trimmedProject := strings.TrimSpace(projectID)
	if trimmedProject == "" || trimmedProject != pp.projectID {
		return ErrScopeMismatch
	}

	// If site bound is configured, target must match site exactly
	if pp.siteID != "" {
		trimmedSite := strings.TrimSpace(siteID)
		if trimmedSite == "" || trimmedSite != pp.siteID {
			return ErrScopeMismatch
		}
	}

	return nil
}

// ValidateNestedScope validates that both the nested subcontractor and its parent contractor
// permit the requested operation at the target scope and time. Fails closed if parent is inactive,
// expired, or misaligned.
func (pp ProjectParticipation) ValidateNestedScope(parent ProjectParticipation, projectID, siteID string, at time.Time) error {
	if pp.parentParticipationID != "" {
		if parent.ParticipationID() != pp.parentParticipationID {
			return fmt.Errorf("%w: parent participation mismatch", ErrParentMismatch)
		}
		if err := parent.ValidateScopeAt(projectID, siteID, at); err != nil {
			return fmt.Errorf("%w: parent scope check failed: %v", ErrParentNotActive, err)
		}
	}
	return pp.ValidateScopeAt(projectID, siteID, at)
}

// ValidateNoSiblingAccess verifies that a participation cannot access or claim authority over a sibling contractor.
func (pp ProjectParticipation) ValidateNoSiblingAccess(target ProjectParticipation) error {
	if pp.TenantID() != target.TenantID() {
		return ErrCrossTenantLinkage
	}
	if pp.ParticipationID() == target.ParticipationID() {
		return nil // Same participation
	}
	if target.ParentParticipationID() == pp.ParticipationID() {
		return nil // Downward parent-to-child relationship
	}
	// Lateral sibling access between distinct contractors or across sibling projects is denied
	if pp.ProjectID() != target.ProjectID() || pp.PartyID() != target.PartyID() {
		return ErrSiblingAccessDenied
	}
	return nil
}

// SimulateReversibleParticipationState provides a local simulation harness method allowing
// test fixtures to simulate forward and reverse lifecycle state transitions on a participation
// in memory without enacting external authority (H030-002).
func SimulateReversibleParticipationState(pp ProjectParticipation, targetState LifecycleState) (ProjectParticipation, error) {
	switch targetState {
	case StateActive, StateClosed, StateArchived:
		pp.state = targetState
		return pp, nil
	default:
		return pp, fmt.Errorf("unrecognized target participation state: %s", targetState)
	}
}

// NewProjectParticipation constructs and validates a new primary ProjectParticipation relationship (depth 0).
func NewProjectParticipation(party Party, participationID, projectID, siteID, sponsorID string, role ParticipationRole, validFrom, validTo time.Time) (ProjectParticipation, error) {
	if party.TenantID() == "" || party.PartyID() == "" {
		return ProjectParticipation{}, ErrParentMismatch
	}
	if !party.IsActive() {
		return ProjectParticipation{}, ErrPartyArchived
	}
	trimmedPartID := strings.TrimSpace(participationID)
	if trimmedPartID == "" {
		return ProjectParticipation{}, ErrBlankID
	}
	trimmedProject := strings.TrimSpace(projectID)
	if trimmedProject == "" {
		return ProjectParticipation{}, ErrBlankID
	}
	if err := ValidateSponsorID(sponsorID); err != nil {
		return ProjectParticipation{}, err
	}
	if err := AssertNoInternalAuthority(role); err != nil {
		return ProjectParticipation{}, err
	}
	if validTo.Before(validFrom) || validTo.Equal(validFrom) {
		return ProjectParticipation{}, ErrInvalidTimeWindow
	}

	return ProjectParticipation{
		participationID:       trimmedPartID,
		tenantID:              party.TenantID(),
		partyID:               party.PartyID(),
		projectID:             trimmedProject,
		siteID:                strings.TrimSpace(siteID),
		sponsorID:             strings.TrimSpace(sponsorID),
		role:                  role,
		validFrom:             validFrom,
		validTo:               validTo,
		state:                 StateActive,
		parentParticipationID: "",
		nestingDepth:          0,
	}, nil
}

// NewNestedSubcontractorParticipation constructs a nested subcontractor participation (depth 1)
// bound under a primary contractor participation.
//
// Invariants enforced under H030-002:
// 1. Mandatory Internal Sponsor: sponsorID must be an authorized internal manager (usr_*).
// 2. Strict Depth Ceiling: Parent must have NestingDepth == 0; sub-subcontracting (depth 2+) is rejected with ErrNestingDepthExceeded.
// 3. Parent Active Requirement: Parent must be active (rejects archived/closed parent).
// 4. Temporal Containment: Subcontractor validity window cannot exceed parent validity window (ErrValidityWindowExceedsParent).
// 5. Scope Bounding: Subcontractor inherits parent project and cannot expand beyond parent site bounds.
// 6. Non-Elevation: Subcontractor receives no internal company/BU/tenant authority.
func NewNestedSubcontractorParticipation(parent ProjectParticipation, subParty Party, participationID, siteID, sponsorID string, role ParticipationRole, validFrom, validTo time.Time) (ProjectParticipation, error) {
	if parent.TenantID() == "" || parent.ParticipationID() == "" {
		return ProjectParticipation{}, ErrParentMismatch
	}
	if !parent.IsActive() {
		if parent.State() == StateClosed {
			return ProjectParticipation{}, ErrParentClosed
		}
		return ProjectParticipation{}, ErrParentNotActive
	}
	if parent.NestingDepth() >= MaxContractorNestingDepth {
		return ProjectParticipation{}, ErrNestingDepthExceeded
	}

	if subParty.TenantID() == "" || subParty.PartyID() == "" {
		return ProjectParticipation{}, ErrParentMismatch
	}
	if !subParty.IsActive() {
		return ProjectParticipation{}, ErrPartyArchived
	}
	if subParty.TenantID() != parent.TenantID() {
		return ProjectParticipation{}, ErrCrossTenantLinkage
	}

	trimmedPartID := strings.TrimSpace(participationID)
	if trimmedPartID == "" {
		return ProjectParticipation{}, ErrBlankID
	}

	if err := ValidateSponsorID(sponsorID); err != nil {
		return ProjectParticipation{}, err
	}
	if err := AssertNoInternalAuthority(role); err != nil {
		return ProjectParticipation{}, err
	}

	// Temporal validity checks
	if validTo.Before(validFrom) || validTo.Equal(validFrom) {
		return ProjectParticipation{}, ErrInvalidTimeWindow
	}
	// Subcontractor validity cannot exceed parent validity window
	if validFrom.Before(parent.ValidFrom()) || validTo.After(parent.ValidTo()) {
		return ProjectParticipation{}, ErrValidityWindowExceedsParent
	}

	// Site bounding checks: if parent is site-restricted, subcontractor cannot target a different site
	effectiveSiteID := strings.TrimSpace(siteID)
	if parent.SiteID() != "" {
		if effectiveSiteID != "" && effectiveSiteID != parent.SiteID() {
			return ProjectParticipation{}, fmt.Errorf("%w: subcontractor site %q cannot exceed parent site bound %q", ErrScopeMismatch, effectiveSiteID, parent.SiteID())
		}
		if effectiveSiteID == "" {
			effectiveSiteID = parent.SiteID() // Default to parent site bound
		}
	}

	return ProjectParticipation{
		participationID:       trimmedPartID,
		tenantID:              parent.TenantID(),
		partyID:               subParty.PartyID(),
		projectID:             parent.ProjectID(),
		siteID:                effectiveSiteID,
		sponsorID:             strings.TrimSpace(sponsorID),
		role:                  role,
		validFrom:             validFrom,
		validTo:               validTo,
		state:                 StateActive,
		parentParticipationID: parent.ParticipationID(),
		nestingDepth:          parent.NestingDepth() + 1,
	}, nil
}

// PartyRegistry manages tenant-isolated parties and their project participation bindings in memory.
type PartyRegistry struct {
	mu             sync.RWMutex
	parties        map[string]Party                // key: tenantID + ":" + partyID
	participations map[string]ProjectParticipation // key: tenantID + ":" + participationID
}

// NewPartyRegistry constructs an empty in-memory PartyRegistry.
func NewPartyRegistry() *PartyRegistry {
	return &PartyRegistry{
		parties:        make(map[string]Party),
		participations: make(map[string]ProjectParticipation),
	}
}

func makePartyKey(tenantID, partyID string) string {
	return fmt.Sprintf("%s:%s", strings.TrimSpace(tenantID), strings.TrimSpace(partyID))
}

func makeParticipationKey(tenantID, participationID string) string {
	return fmt.Sprintf("%s:%s", strings.TrimSpace(tenantID), strings.TrimSpace(participationID))
}

// RegisterParty stores a new party. Returns ErrDuplicateParty if ID is already registered for tenant.
func (r *PartyRegistry) RegisterParty(p Party) error {
	if p.TenantID() == "" || p.PartyID() == "" {
		return ErrBlankID
	}
	key := makePartyKey(p.TenantID(), p.PartyID())

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.parties[key]; exists {
		return ErrDuplicateParty
	}
	r.parties[key] = p
	return nil
}

// GetParty retrieves a party by tenant and party ID. Fails closed if not found.
func (r *PartyRegistry) GetParty(tenantID, partyID string) (Party, error) {
	key := makePartyKey(tenantID, partyID)

	r.mu.RLock()
	defer r.mu.RUnlock()

	p, exists := r.parties[key]
	if !exists {
		return Party{}, ErrPartyNotFound
	}
	return p, nil
}

// RegisterParticipation stores a new participation relationship.
func (r *PartyRegistry) RegisterParticipation(pp ProjectParticipation) error {
	if pp.TenantID() == "" || pp.ParticipationID() == "" {
		return ErrBlankID
	}
	key := makeParticipationKey(pp.TenantID(), pp.ParticipationID())

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.participations[key]; exists {
		return ErrDuplicateParticipation
	}
	r.participations[key] = pp
	return nil
}

// GetParticipation retrieves a participation by tenant and participation ID.
func (r *PartyRegistry) GetParticipation(tenantID, participationID string) (ProjectParticipation, error) {
	key := makeParticipationKey(tenantID, participationID)

	r.mu.RLock()
	defer r.mu.RUnlock()

	pp, exists := r.participations[key]
	if !exists {
		return ProjectParticipation{}, ErrParticipationNotFound
	}
	return pp, nil
}

// ListParticipationsByProject returns all participations bounded to a given project under a tenant.
func (r *PartyRegistry) ListParticipationsByProject(tenantID, projectID string) ([]ProjectParticipation, error) {
	tTenant := strings.TrimSpace(tenantID)
	if tTenant == "" {
		return nil, ErrBlankTenantID
	}
	tProject := strings.TrimSpace(projectID)
	if tProject == "" {
		return nil, ErrBlankID
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []ProjectParticipation
	for _, pp := range r.participations {
		if pp.TenantID() == tTenant && pp.ProjectID() == tProject {
			result = append(result, pp)
		}
	}
	return result, nil
}

// ListParticipationsByParty returns all participations for a specific party under a tenant.
func (r *PartyRegistry) ListParticipationsByParty(tenantID, partyID string) ([]ProjectParticipation, error) {
	tTenant := strings.TrimSpace(tenantID)
	if tTenant == "" {
		return nil, ErrBlankTenantID
	}
	tParty := strings.TrimSpace(partyID)
	if tParty == "" {
		return nil, ErrBlankID
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []ProjectParticipation
	for _, pp := range r.participations {
		if pp.TenantID() == tTenant && pp.PartyID() == tParty {
			result = append(result, pp)
		}
	}
	return result, nil
}

// ListSubcontractors returns all nested subcontractor participations under a given prime contractor participation.
func (r *PartyRegistry) ListSubcontractors(tenantID, parentParticipationID string) ([]ProjectParticipation, error) {
	tTenant := strings.TrimSpace(tenantID)
	if tTenant == "" {
		return nil, ErrBlankTenantID
	}
	tParentID := strings.TrimSpace(parentParticipationID)
	if tParentID == "" {
		return nil, ErrBlankID
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []ProjectParticipation
	for _, pp := range r.participations {
		if pp.TenantID() == tTenant && pp.ParentParticipationID() == tParentID {
			result = append(result, pp)
		}
	}
	return result, nil
}
