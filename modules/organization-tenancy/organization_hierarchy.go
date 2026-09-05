package orgtenancy

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// Default locale and time zone standards for physical site operations.
const (
	DefaultTimeZone = "Asia/Bangkok"
	DefaultLocale   = "th-TH"
	FallbackLocale  = "en-US"

	DefaultNonAuthorityScopeNotice = "DERIVED_OUTPUT_NON_AUTHORITY: Resolved hierarchy scopes are descriptive projections only and never grant lateral, upward, or implicit operational authority."
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
	// ErrInvalidTimeZone indicates that the time zone identifier is not a valid IANA time zone.
	ErrInvalidTimeZone = errors.New("invalid IANA time zone identifier")
	// ErrInvalidLocale indicates that the locale identifier is not a valid BCP 47 language tag.
	ErrInvalidLocale = errors.New("invalid BCP 47 locale tag")
	// ErrProjectSiteMismatch indicates a mismatch between a site and its parent project.
	ErrProjectSiteMismatch = errors.New("site does not belong to the specified project")
)

var knownCanonicalPrefixes = map[string]bool{
	PrefixTenant:       true,
	PrefixCompany:      true,
	PrefixBusinessUnit: true,
	PrefixProject:      true,
	PrefixSite:         true,
	PrefixArea:         true,
	PrefixParty:        true,
	PrefixUser:         true,
	PrefixCorrelation:  true,
	PrefixCausation:    true,
	PrefixIdempotency:  true,
	PrefixExternalRef:  true,
}

// ValidateTimeZone validates that tz is a recognized IANA time zone identifier.
func ValidateTimeZone(tz string) error {
	trimmed := strings.TrimSpace(tz)
	if trimmed == "" {
		return nil
	}
	if _, err := time.LoadLocation(trimmed); err != nil {
		return fmt.Errorf("%w: %q", ErrInvalidTimeZone, trimmed)
	}
	return nil
}

// ValidateLocale validates that loc is a well-formed BCP 47 language tag (e.g. th-TH, en-US, th, en).
func ValidateLocale(loc string) error {
	trimmed := strings.TrimSpace(loc)
	if trimmed == "" {
		return nil
	}
	parts := strings.Split(trimmed, "-")
	if len(parts) == 0 || len(parts) > 3 {
		return fmt.Errorf("%w: %q", ErrInvalidLocale, trimmed)
	}
	for _, p := range parts {
		if len(p) == 0 {
			return fmt.Errorf("%w: %q", ErrInvalidLocale, trimmed)
		}
		for i := 0; i < len(p); i++ {
			c := p[i]
			if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')) {
				return fmt.Errorf("%w: %q", ErrInvalidLocale, trimmed)
			}
		}
	}
	return nil
}

// validateEntityID validates an entity identifier.
// If the identifier contains an underscore and has a canonical prefix format, it enforces canonical prefix matching
// and valid lowercase character sets, while preserving backward compatibility for synthetic slugs.
func validateEntityID(id, expectedPrefix string) error {
	trimmed := strings.TrimSpace(id)
	if trimmed == "" {
		return ErrBlankID
	}
	if strings.Contains(trimmed, "_") {
		idx := strings.IndexByte(trimmed, '_')
		if idx <= 0 || idx == len(trimmed)-1 {
			return ErrMalformedIdentifier
		}
		prefix := trimmed[:idx]
		token := trimmed[idx+1:]
		if knownCanonicalPrefixes[prefix] && prefix != expectedPrefix {
			return fmt.Errorf("%w: expected prefix %q, got %q", ErrPrefixMismatch, expectedPrefix, prefix)
		}
		for i := 0; i < len(token); i++ {
			c := token[i]
			if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_' || c == '-') {
				return ErrInvalidCharacters
			}
		}
	}
	return nil
}

func validateTenantID(tenantID string) error {
	trimmed := strings.TrimSpace(tenantID)
	if trimmed == "" {
		return ErrBlankTenantID
	}
	if strings.Contains(trimmed, "_") {
		idx := strings.IndexByte(trimmed, '_')
		if idx <= 0 || idx == len(trimmed)-1 {
			return ErrMalformedIdentifier
		}
		prefix := trimmed[:idx]
		token := trimmed[idx+1:]
		if knownCanonicalPrefixes[prefix] && prefix != PrefixTenant {
			return fmt.Errorf("%w: expected prefix %q, got %q", ErrPrefixMismatch, PrefixTenant, prefix)
		}
		for i := 0; i < len(token); i++ {
			c := token[i]
			if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_' || c == '-') {
				return ErrInvalidCharacters
			}
		}
	}
	return nil
}

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
	if err := validateTenantID(tenantID); err != nil {
		return Company{}, err
	}
	if err := validateEntityID(companyID, PrefixCompany); err != nil {
		return Company{}, err
	}
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return Company{}, ErrBlankName
	}

	return Company{
		tenantID:  strings.TrimSpace(tenantID),
		companyID: strings.TrimSpace(companyID),
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
	if err := validateEntityID(businessUnitID, PrefixBusinessUnit); err != nil {
		return BusinessUnit{}, err
	}
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return BusinessUnit{}, ErrBlankName
	}

	return BusinessUnit{
		tenantID:       company.TenantID(),
		companyID:      company.CompanyID(),
		businessUnitID: strings.TrimSpace(businessUnitID),
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
	if err := validateEntityID(projectID, PrefixProject); err != nil {
		return Project{}, err
	}
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return Project{}, ErrBlankName
	}

	return Project{
		tenantID:       bu.TenantID(),
		companyID:      bu.CompanyID(),
		businessUnitID: bu.BusinessUnitID(),
		projectID:      strings.TrimSpace(projectID),
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
	if err := validateEntityID(projectID, PrefixProject); err != nil {
		return Project{}, err
	}
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return Project{}, ErrBlankName
	}

	return Project{
		tenantID:  company.TenantID(),
		companyID: company.CompanyID(),
		projectID: strings.TrimSpace(projectID),
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
	timeZone       string
	locale         string
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

// TimeZone returns the site IANA time zone identifier (default Asia/Bangkok).
func (s Site) TimeZone() string {
	if s.timeZone == "" {
		return DefaultTimeZone
	}
	return s.timeZone
}

// Locale returns the site BCP 47 locale identifier (default th-TH).
func (s Site) Locale() string {
	if s.locale == "" {
		return DefaultLocale
	}
	return s.locale
}

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

// ValidateParentProject ensures that the site belongs to the expected project.
func (s Site) ValidateParentProject(expectedProjectID string) error {
	if s.projectID != strings.TrimSpace(expectedProjectID) {
		return ErrProjectSiteMismatch
	}
	return nil
}

// NewSite constructs and validates a new Site under the specified parent Project with default time zone and locale.
func NewSite(project Project, siteID, name string) (Site, error) {
	return NewSiteWithLocale(project, siteID, name, DefaultTimeZone, DefaultLocale)
}

// NewSiteWithLocale constructs and validates a new Site with explicit time zone and locale.
func NewSiteWithLocale(project Project, siteID, name, timeZone, locale string) (Site, error) {
	if project.TenantID() == "" || project.CompanyID() == "" || project.ProjectID() == "" {
		return Site{}, ErrParentMismatch
	}
	if !project.IsActive() {
		return Site{}, ErrParentArchived
	}
	if err := validateEntityID(siteID, PrefixSite); err != nil {
		return Site{}, err
	}
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return Site{}, ErrBlankName
	}

	tz := strings.TrimSpace(timeZone)
	if tz == "" {
		tz = DefaultTimeZone
	} else if err := ValidateTimeZone(tz); err != nil {
		return Site{}, err
	}

	loc := strings.TrimSpace(locale)
	if loc == "" {
		loc = DefaultLocale
	} else if err := ValidateLocale(loc); err != nil {
		return Site{}, err
	}

	return Site{
		tenantID:       project.TenantID(),
		companyID:      project.CompanyID(),
		businessUnitID: project.BusinessUnitID(),
		projectID:      project.ProjectID(),
		siteID:         strings.TrimSpace(siteID),
		name:           trimmedName,
		timeZone:       tz,
		locale:         loc,
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
	timeZone       string
	locale         string
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

// TimeZone returns the inherited or configured time zone identifier.
func (a Area) TimeZone() string { return a.timeZone }

// Locale returns the inherited or configured locale identifier.
func (a Area) Locale() string { return a.locale }

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

// ValidateParentSite ensures that the area belongs to the expected site.
func (a Area) ValidateParentSite(expectedSiteID string) error {
	if a.siteID != strings.TrimSpace(expectedSiteID) {
		return ErrParentMismatch
	}
	return nil
}

// NewArea constructs and validates a new Area under the specified parent Site, inheriting time zone and locale from the site.
func NewArea(site Site, areaID, name string) (Area, error) {
	return NewAreaWithLocale(site, areaID, name, "", "")
}

// NewAreaWithLocale constructs and validates a new Area. If timeZone or locale are unspecified/blank,
// they are inherited from the parent site.
func NewAreaWithLocale(site Site, areaID, name, timeZone, locale string) (Area, error) {
	if site.TenantID() == "" || site.CompanyID() == "" || site.ProjectID() == "" || site.SiteID() == "" {
		return Area{}, ErrParentMismatch
	}
	if !site.IsActive() {
		return Area{}, ErrParentArchived
	}
	if err := validateEntityID(areaID, PrefixArea); err != nil {
		return Area{}, err
	}
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return Area{}, ErrBlankName
	}

	tz := strings.TrimSpace(timeZone)
	if tz == "" {
		tz = site.TimeZone()
	} else if err := ValidateTimeZone(tz); err != nil {
		return Area{}, err
	}

	loc := strings.TrimSpace(locale)
	if loc == "" {
		loc = site.Locale()
	} else if err := ValidateLocale(loc); err != nil {
		return Area{}, err
	}

	return Area{
		tenantID:       site.TenantID(),
		companyID:      site.CompanyID(),
		businessUnitID: site.BusinessUnitID(),
		projectID:      site.ProjectID(),
		siteID:         site.SiteID(),
		areaID:         strings.TrimSpace(areaID),
		name:           trimmedName,
		timeZone:       tz,
		locale:         loc,
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
	if err := validateEntityID(trimmedPartyID, PrefixParty); err != nil {
		return SponsoredParty{}, err
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

// ResolvedScope captures an immutable, non-authoritative description of resolved hierarchy scope.
// It is a descriptive projection only and does not confer lateral, upward, or implicit operational authority.
type ResolvedScope struct {
	TenantID           string `json:"tenant_id"`
	CompanyID          string `json:"company_id"`
	BusinessUnitID     string `json:"business_unit_id,omitempty"`
	ProjectID          string `json:"project_id,omitempty"`
	SiteID             string `json:"site_id,omitempty"`
	AreaID             string `json:"area_id,omitempty"`
	TimeZone           string `json:"time_zone,omitempty"`
	Locale             string `json:"locale,omitempty"`
	CanonicalPath      string `json:"canonical_path"`
	NonAuthorityNotice string `json:"non_authority_notice"`
}

// ValidateScope checks that the resolved scope matches the trusted tenant context.
// Invariant: It validates tenant equality only and conveys zero extra authority.
func (rs ResolvedScope) ValidateScope(ctx TenantContext) error {
	return ctx.AuthorizeTenantScope(rs.TenantID)
}

// ResolveScope resolves the hierarchy scope for a Company.
func (c Company) ResolveScope() ResolvedScope {
	return ResolvedScope{
		TenantID:           c.tenantID,
		CompanyID:          c.companyID,
		CanonicalPath:      fmt.Sprintf("%s/%s", c.tenantID, c.companyID),
		NonAuthorityNotice: DefaultNonAuthorityScopeNotice,
	}
}

// ResolveScope resolves the hierarchy scope for a BusinessUnit.
func (b BusinessUnit) ResolveScope() ResolvedScope {
	return ResolvedScope{
		TenantID:           b.tenantID,
		CompanyID:          b.companyID,
		BusinessUnitID:     b.businessUnitID,
		CanonicalPath:      fmt.Sprintf("%s/%s/%s", b.tenantID, b.companyID, b.businessUnitID),
		NonAuthorityNotice: DefaultNonAuthorityScopeNotice,
	}
}

// ResolveScope resolves the hierarchy scope for a Project.
func (p Project) ResolveScope() ResolvedScope {
	path := fmt.Sprintf("%s/%s/%s", p.tenantID, p.companyID, p.projectID)
	if p.businessUnitID != "" {
		path = fmt.Sprintf("%s/%s/%s/%s", p.tenantID, p.companyID, p.businessUnitID, p.projectID)
	}
	return ResolvedScope{
		TenantID:           p.tenantID,
		CompanyID:          p.companyID,
		BusinessUnitID:     p.businessUnitID,
		ProjectID:          p.projectID,
		CanonicalPath:      path,
		NonAuthorityNotice: DefaultNonAuthorityScopeNotice,
	}
}

// ResolveScope resolves the hierarchy scope for a Site.
func (s Site) ResolveScope() ResolvedScope {
	path := fmt.Sprintf("%s/%s/%s/%s", s.tenantID, s.companyID, s.projectID, s.siteID)
	if s.businessUnitID != "" {
		path = fmt.Sprintf("%s/%s/%s/%s/%s", s.tenantID, s.companyID, s.businessUnitID, s.projectID, s.siteID)
	}
	return ResolvedScope{
		TenantID:           s.tenantID,
		CompanyID:          s.companyID,
		BusinessUnitID:     s.businessUnitID,
		ProjectID:          s.projectID,
		SiteID:             s.siteID,
		TimeZone:           s.TimeZone(),
		Locale:             s.Locale(),
		CanonicalPath:      path,
		NonAuthorityNotice: DefaultNonAuthorityScopeNotice,
	}
}

// ResolveScope resolves the hierarchy scope for an Area.
func (a Area) ResolveScope() ResolvedScope {
	path := fmt.Sprintf("%s/%s/%s/%s/%s", a.tenantID, a.companyID, a.projectID, a.siteID, a.areaID)
	if a.businessUnitID != "" {
		path = fmt.Sprintf("%s/%s/%s/%s/%s/%s", a.tenantID, a.companyID, a.businessUnitID, a.projectID, a.siteID, a.areaID)
	}
	return ResolvedScope{
		TenantID:           a.tenantID,
		CompanyID:          a.companyID,
		BusinessUnitID:     a.businessUnitID,
		ProjectID:          a.projectID,
		SiteID:             a.siteID,
		AreaID:             a.areaID,
		TimeZone:           a.TimeZone(),
		Locale:             a.Locale(),
		CanonicalPath:      path,
		NonAuthorityNotice: DefaultNonAuthorityScopeNotice,
	}
}

// ResolveScope resolves the hierarchy scope for a SponsoredParty.
func (sp SponsoredParty) ResolveScope() ResolvedScope {
	return ResolvedScope{
		TenantID:           sp.tenantID,
		ProjectID:          sp.projectID,
		SiteID:             sp.siteID,
		CanonicalPath:      fmt.Sprintf("%s/%s/%s/%s", sp.tenantID, sp.projectID, sp.siteID, sp.partyID),
		NonAuthorityNotice: DefaultNonAuthorityScopeNotice,
	}
}
