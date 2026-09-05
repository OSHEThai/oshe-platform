package orgtenancy

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

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

// ProjectParticipation models an external party's bounded, time-limited participation in a specific project.
type ProjectParticipation struct {
	participationID string
	tenantID        string
	partyID         string
	projectID       string
	siteID          string // optional bound to specific site within project
	sponsorID       string // internal manager subject (usr_*)
	role            ParticipationRole
	validFrom       time.Time
	validTo         time.Time
	state           LifecycleState
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

// Archive returns an immutable copy of the participation in ARCHIVED state (no hard deletion).
func (pp ProjectParticipation) Archive() ProjectParticipation {
	pp.state = StateArchived
	return pp
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

// NewProjectParticipation constructs and validates a new ProjectParticipation relationship.
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
	trimmedSponsor := strings.TrimSpace(sponsorID)
	if trimmedSponsor == "" {
		return ProjectParticipation{}, ErrBlankSponsorID
	}
	if err := ValidateParticipationRole(role); err != nil {
		return ProjectParticipation{}, err
	}
	if validTo.Before(validFrom) || validTo.Equal(validFrom) {
		return ProjectParticipation{}, ErrInvalidTimeWindow
	}

	return ProjectParticipation{
		participationID: trimmedPartID,
		tenantID:        party.TenantID(),
		partyID:         party.PartyID(),
		projectID:       trimmedProject,
		siteID:          strings.TrimSpace(siteID),
		sponsorID:       trimmedSponsor,
		role:            role,
		validFrom:       validFrom,
		validTo:         validTo,
		state:           StateActive,
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
