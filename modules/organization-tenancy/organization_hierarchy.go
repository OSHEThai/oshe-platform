package orgtenancy

import (
	"errors"
	"strings"
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

// Project represents a project entity strictly bounded to a company and tenant.
type Project struct {
	tenantID  string
	companyID string
	projectID string
	name      string
	state     LifecycleState
}

// TenantID returns the authoritative tenant identifier.
func (p Project) TenantID() string { return p.tenantID }

// CompanyID returns the parent company identifier.
func (p Project) CompanyID() string { return p.companyID }

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
	tenantID  string
	companyID string
	projectID string
	siteID    string
	name      string
	state     LifecycleState
}

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
		tenantID:  project.TenantID(),
		companyID: project.CompanyID(),
		projectID: project.ProjectID(),
		siteID:    trimmedSite,
		name:      trimmedName,
		state:     StateActive,
	}, nil
}

// Area represents a localized inspection area strictly bounded to a site, project, company, and tenant.
type Area struct {
	tenantID  string
	companyID string
	projectID string
	siteID    string
	areaID    string
	name      string
	state     LifecycleState
}

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
		tenantID:  site.TenantID(),
		companyID: site.CompanyID(),
		projectID: site.ProjectID(),
		siteID:    site.SiteID(),
		areaID:    trimmedArea,
		name:      trimmedName,
		state:     StateActive,
	}, nil
}
