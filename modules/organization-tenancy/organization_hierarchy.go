package orgtenancy

import (
	"errors"
	"strings"
	"time"
)

// LifecycleState represents the operational lifecycle state of a hierarchy entity.
type LifecycleState string

const (
	StateActive   LifecycleState = "ACTIVE"
	StateArchived LifecycleState = "ARCHIVED"
)

var (
	// ErrBlankID indicates an entity identifier is empty or whitespace only.
	ErrBlankID = errors.New("identifier must not be blank")
	// ErrBlankTenantID indicates a tenant identifier is empty or whitespace only.
	ErrBlankTenantID = errors.New("tenant ID must not be blank")
	// ErrBlankName indicates an entity name is empty or whitespace only.
	ErrBlankName = errors.New("name must not be blank")
	// ErrCrossTenantLinkage indicates a child entity attempted to link to a parent belonging to another tenant.
	ErrCrossTenantLinkage = errors.New("child tenant ID does not match parent tenant ID")
	// ErrParentArchived indicates an attempt to create or attach a child under an archived parent entity.
	ErrParentArchived = errors.New("cannot attach child context to an archived parent")
	// ErrParentMismatch indicates parent identifier inconsistency within the hierarchy.
	ErrParentMismatch = errors.New("parent entity relationship mismatch")
	// ErrEntityArchived indicates the entity is archived and not available for active operations.
	ErrEntityArchived = errors.New("entity is archived")
	// ErrBlankSponsorID indicates an internal sponsor identifier is empty or whitespace only.
	ErrBlankSponsorID = errors.New("internal sponsor identifier must not be blank")
	// ErrInvalidTimeWindow indicates that valid_to is not strictly after valid_from.
	ErrInvalidTimeWindow = errors.New("valid_to must be strictly after valid_from")
	// ErrSponsorshipExpired indicates the sponsored party relationship has expired.
	ErrSponsorshipExpired = errors.New("sponsored party relationship has expired")
)

// Company represents a company entity strictly bounded to a tenant.
type Company struct {
	tenantID  string
	companyID string
	name      string
	state     LifecycleState
}

// TenantID returns the authoritative tenant identifier.
func (c Company) TenantID() string { return c.tenantID }

// CompanyID returns the authoritative company identifier.
func (c Company) CompanyID() string { return c.companyID }

// Name returns the company display name.
func (c Company) Name() string { return c.name }

// State returns the operational lifecycle state.
func (c Company) State() LifecycleState { return c.state }

// IsActive returns true if the company is in ACTIVE state.
func (c Company) IsActive() bool { return c.state == StateActive }

// Archive returns an immutable copy of the company in ARCHIVED state.
func (c Company) Archive() Company {
	c.state = StateArchived
	return c
}

// ValidateScope confirms the company matches the trusted tenant context without granting extra permissions.
func (c Company) ValidateScope(ctx TenantContext) error {
	return ctx.AuthorizeTenantScope(c.tenantID)
}

// NewCompany constructs and validates a new Company under the specified tenant.
func NewCompany(tenantID, companyID, name string) (Company, error) {
	trimmedTenant := strings.TrimSpace(tenantID)
	if trimmedTenant == "" {
		return Company{}, ErrBlankTenantID
	}
	trimmedCompany := strings.TrimSpace(companyID)
	if trimmedCompany == "" {
		return Company{}, ErrBlankID
	}
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return Company{}, ErrBlankName
	}

	return Company{
		tenantID:  trimmedTenant,
		companyID: trimmedCompany,
		name:      trimmedName,
		state:     StateActive,
	}, nil
}

// BusinessUnit represents a business unit entity strictly bounded to a company and tenant.
type BusinessUnit struct {
	tenantID       string
	companyID      string
	businessUnitID string
	name           string
	state          LifecycleState
}

// TenantID returns the authoritative tenant identifier.
func (b BusinessUnit) TenantID() string { return b.tenantID }

// CompanyID returns the parent company identifier.
func (b BusinessUnit) CompanyID() string { return b.companyID }

// BusinessUnitID returns the authoritative business unit identifier.
func (b BusinessUnit) BusinessUnitID() string { return b.businessUnitID }

// Name returns the business unit display name.
func (b BusinessUnit) Name() string { return b.name }

// State returns the operational lifecycle state.
func (b BusinessUnit) State() LifecycleState { return b.state }

// IsActive returns true if the business unit is in ACTIVE state.
func (b BusinessUnit) IsActive() bool { return b.state == StateActive }

// Archive returns an immutable copy of the business unit in ARCHIVED state.
func (b BusinessUnit) Archive() BusinessUnit {
	b.state = StateArchived
	return b
}

// ValidateScope confirms the business unit matches the trusted tenant context without granting extra permissions.
func (b BusinessUnit) ValidateScope(ctx TenantContext) error {
	return ctx.AuthorizeTenantScope(b.tenantID)
}

// NewBusinessUnit constructs and validates a new BusinessUnit under the specified parent Company.
func NewBusinessUnit(company Company, businessUnitID, name string) (BusinessUnit, error) {
	if company.TenantID() == "" || company.CompanyID() == "" {
		return BusinessUnit{}, ErrParentMismatch
	}
	if !company.IsActive() {
		return BusinessUnit{}, ErrParentArchived
	}
	trimmedBU := strings.TrimSpace(businessUnitID)
	if trimmedBU == "" {
		return BusinessUnit{}, ErrBlankID
	}
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return BusinessUnit{}, ErrBlankName
	}

	return BusinessUnit{
		tenantID:       company.TenantID(),
		companyID:      company.CompanyID(),
		businessUnitID: trimmedBU,
		name:           trimmedName,
		state:          StateActive,
	}, nil
}

// Project represents a project entity strictly bounded to a business unit (or company) and tenant.
type Project struct {
	tenantID       string
	companyID      string
	businessUnitID string
	projectID      string
	name           string
	state          LifecycleState
}

// TenantID returns the authoritative tenant identifier.
func (p Project) TenantID() string { return p.tenantID }

// CompanyID returns the parent company identifier.
func (p Project) CompanyID() string { return p.companyID }

// BusinessUnitID returns the parent business unit identifier if bound.
func (p Project) BusinessUnitID() string { return p.businessUnitID }

// ProjectID returns the authoritative project identifier.
func (p Project) ProjectID() string { return p.projectID }

// Name returns the project display name.
func (p Project) Name() string { return p.name }

// State returns the operational lifecycle state.
func (p Project) State() LifecycleState { return p.state }

// IsActive returns true if the project is in ACTIVE state.
func (p Project) IsActive() bool { return p.state == StateActive }

// Archive returns an immutable copy of the project in ARCHIVED state.
func (p Project) Archive() Project {
	p.state = StateArchived
	return p
}

// ValidateScope confirms the project matches the trusted tenant context without granting extra permissions.
func (p Project) ValidateScope(ctx TenantContext) error {
	return ctx.AuthorizeTenantScope(p.tenantID)
}

// NewProjectUnderBusinessUnit constructs and validates a new Project under a parent BusinessUnit.
func NewProjectUnderBusinessUnit(bu BusinessUnit, projectID, name string) (Project, error) {
	if bu.TenantID() == "" || bu.CompanyID() == "" || bu.BusinessUnitID() == "" {
		return Project{}, ErrParentMismatch
	}
	if !bu.IsActive() {
		return Project{}, ErrParentArchived
	}
	trimmedProject := strings.TrimSpace(projectID)
	if trimmedProject == "" {
		return Project{}, ErrBlankID
	}
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return Project{}, ErrBlankName
	}

	return Project{
		tenantID:       bu.TenantID(),
		companyID:      bu.CompanyID(),
		businessUnitID: bu.BusinessUnitID(),
		projectID:      trimmedProject,
		name:           trimmedName,
		state:          StateActive,
	}, nil
}


// NewProject constructs and validates a new Project under the specified parent Company.
func NewProject(company Company, projectID, name string) (Project, error) {
	if company.TenantID() == "" || company.CompanyID() == "" {
		return Project{}, ErrParentMismatch
	}
	if !company.IsActive() {
		return Project{}, ErrParentArchived
	}
	trimmedProject := strings.TrimSpace(projectID)
	if trimmedProject == "" {
		return Project{}, ErrBlankID
	}
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return Project{}, ErrBlankName
	}

	return Project{
		tenantID:  company.TenantID(),
		companyID: company.CompanyID(),
		projectID: trimmedProject,
		name:      trimmedName,
		state:     StateActive,
	}, nil
}

// Site represents a physical or operational site strictly bounded to a project, company, and tenant.
type Site struct {
	tenantID       string
	companyID      string
	businessUnitID string
	projectID      string
	siteID         string
	name           string
	state          LifecycleState
}

// BusinessUnitID returns the parent business unit identifier.
func (s Site) BusinessUnitID() string { return s.businessUnitID }

// TenantID returns the authoritative tenant identifier.
func (s Site) TenantID() string { return s.tenantID }

// CompanyID returns the parent company identifier.
func (s Site) CompanyID() string { return s.companyID }

// ProjectID returns the parent project identifier.
func (s Site) ProjectID() string { return s.projectID }

// SiteID returns the authoritative site identifier.
func (s Site) SiteID() string { return s.siteID }

// Name returns the site display name.
func (s Site) Name() string { return s.name }

// State returns the operational lifecycle state.
func (s Site) State() LifecycleState { return s.state }

// IsActive returns true if the site is in ACTIVE state.
func (s Site) IsActive() bool { return s.state == StateActive }

// Archive returns an immutable copy of the site in ARCHIVED state.
func (s Site) Archive() Site {
	s.state = StateArchived
	return s
}

// ValidateScope confirms the site matches the trusted tenant context without granting extra permissions.
func (s Site) ValidateScope(ctx TenantContext) error {
	return ctx.AuthorizeTenantScope(s.tenantID)
}

// NewSite constructs and validates a new Site under the specified parent Project.
func NewSite(project Project, siteID, name string) (Site, error) {
	if project.TenantID() == "" || project.CompanyID() == "" || project.ProjectID() == "" {
		return Site{}, ErrParentMismatch
	}
	if !project.IsActive() {
		return Site{}, ErrParentArchived
	}
	trimmedSite := strings.TrimSpace(siteID)
	if trimmedSite == "" {
		return Site{}, ErrBlankID
	}
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return Site{}, ErrBlankName
	}

	return Site{
		tenantID:       project.TenantID(),
		companyID:      project.CompanyID(),
		businessUnitID: project.BusinessUnitID(),
		projectID:      project.ProjectID(),
		siteID:         trimmedSite,
		name:           trimmedName,
		state:          StateActive,
	}, nil
}

// Area represents a localized inspection area strictly bounded to a site, project, business unit, company, and tenant.
type Area struct {
	tenantID       string
	companyID      string
	businessUnitID string
	projectID      string
	siteID         string
	areaID         string
	name           string
	state          LifecycleState
}

// BusinessUnitID returns the business unit identifier.
func (a Area) BusinessUnitID() string { return a.businessUnitID }

// TenantID returns the authoritative tenant identifier.
func (a Area) TenantID() string { return a.tenantID }

// CompanyID returns the company identifier.
func (a Area) CompanyID() string { return a.companyID }

// ProjectID returns the project identifier.
func (a Area) ProjectID() string { return a.projectID }

// SiteID returns the parent site identifier.
func (a Area) SiteID() string { return a.siteID }

// AreaID returns the authoritative area identifier.
func (a Area) AreaID() string { return a.areaID }

// Name returns the area display name.
func (a Area) Name() string { return a.name }

// State returns the operational lifecycle state.
func (a Area) State() LifecycleState { return a.state }

// IsActive returns true if the area is in ACTIVE state.
func (a Area) IsActive() bool { return a.state == StateActive }

// Archive returns an immutable copy of the area in ARCHIVED state.
func (a Area) Archive() Area {
	a.state = StateArchived
	return a
}

// ValidateScope confirms the area matches the trusted tenant context without granting extra permissions.
func (a Area) ValidateScope(ctx TenantContext) error {
	return ctx.AuthorizeTenantScope(a.tenantID)
}

// NewArea constructs and validates a new Area under the specified parent Site.
func NewArea(site Site, areaID, name string) (Area, error) {
	if site.TenantID() == "" || site.CompanyID() == "" || site.ProjectID() == "" || site.SiteID() == "" {
		return Area{}, ErrParentMismatch
	}
	if !site.IsActive() {
		return Area{}, ErrParentArchived
	}
	trimmedArea := strings.TrimSpace(areaID)
	if trimmedArea == "" {
		return Area{}, ErrBlankID
	}
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return Area{}, ErrBlankName
	}

	return Area{
		tenantID:       site.TenantID(),
		companyID:      site.CompanyID(),
		businessUnitID: site.BusinessUnitID(),
		projectID:      site.ProjectID(),
		siteID:         site.SiteID(),
		areaID:         trimmedArea,
		name:           trimmedName,
		state:          StateActive,
	}, nil
}

// SponsoredParty represents a third-party contractor relationship bounded to an internal sponsor and site/project.
type SponsoredParty struct {
	tenantID    string
	partyID     string
	companyName string
	sponsorID   string
	projectID   string
	siteID      string
	validFrom   time.Time
	validTo     time.Time
	state       LifecycleState
}

// TenantID returns the authoritative tenant identifier.
func (sp SponsoredParty) TenantID() string { return sp.tenantID }

// PartyID returns the contractor party identifier.
func (sp SponsoredParty) PartyID() string { return sp.partyID }

// CompanyName returns the external contractor company name.
func (sp SponsoredParty) CompanyName() string { return sp.companyName }

// SponsorID returns the internal sponsor subject identifier.
func (sp SponsoredParty) SponsorID() string { return sp.sponsorID }

// ProjectID returns the bounded project identifier.
func (sp SponsoredParty) ProjectID() string { return sp.projectID }

// SiteID returns the bounded site identifier.
func (sp SponsoredParty) SiteID() string { return sp.siteID }

// ValidFrom returns the start of the validity window.
func (sp SponsoredParty) ValidFrom() time.Time { return sp.validFrom }

// ValidTo returns the end of the validity window.
func (sp SponsoredParty) ValidTo() time.Time { return sp.validTo }

// State returns the operational lifecycle state.
func (sp SponsoredParty) State() LifecycleState { return sp.state }

// IsActive returns true if the sponsored party is in ACTIVE state.
func (sp SponsoredParty) IsActive() bool { return sp.state == StateActive }

// Archive marks the sponsored party as archived preserving history (no hard deletion).
func (sp SponsoredParty) Archive() SponsoredParty {
	sp.state = StateArchived
	return sp
}

// ValidateScope confirms the sponsored party matches the trusted tenant context.
func (sp SponsoredParty) ValidateScope(ctx TenantContext) error {
	return ctx.AuthorizeTenantScope(sp.tenantID)
}

// IsValidAt checks if the sponsored party is active and within its effective validity window.
func (sp SponsoredParty) IsValidAt(t time.Time) bool {
	if !sp.IsActive() {
		return false
	}
	return !t.Before(sp.validFrom) && !t.After(sp.validTo)
}

// NewSponsoredParty constructs and validates a new SponsoredParty under a bounded Site.
func NewSponsoredParty(site Site, partyID, companyName, sponsorID string, validFrom, validTo time.Time) (SponsoredParty, error) {
	if site.TenantID() == "" || site.SiteID() == "" || site.ProjectID() == "" {
		return SponsoredParty{}, ErrParentMismatch
	}
	if !site.IsActive() {
		return SponsoredParty{}, ErrParentArchived
	}
	trimmedPartyID := strings.TrimSpace(partyID)
	if trimmedPartyID == "" {
		return SponsoredParty{}, ErrBlankID
	}
	trimmedCompany := strings.TrimSpace(companyName)
	if trimmedCompany == "" {
		return SponsoredParty{}, ErrBlankName
	}
	trimmedSponsor := strings.TrimSpace(sponsorID)
	if trimmedSponsor == "" {
		return SponsoredParty{}, ErrBlankSponsorID
	}
	if validTo.Before(validFrom) || validTo.Equal(validFrom) {
		return SponsoredParty{}, ErrInvalidTimeWindow
	}

	return SponsoredParty{
		tenantID:    site.TenantID(),
		partyID:     trimmedPartyID,
		companyName: trimmedCompany,
		sponsorID:   trimmedSponsor,
		projectID:   site.ProjectID(),
		siteID:      site.SiteID(),
		validFrom:   validFrom,
		validTo:     validTo,
		state:       StateActive,
	}, nil
}
