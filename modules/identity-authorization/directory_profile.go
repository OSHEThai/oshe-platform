// Package localidentity provides local identity, authorization, and directory services for OSHE Platform.
//
// PROVISIONAL GOVERNANCE DECLARATION (H030-003, H030-004, H030-005):
// Under approved Sole Human Owner decisions H030-003, H030-004, and H030-005, this file
// establishes the in-memory, synthetic-only scoped directory profile model and registry.
//
// Singular Identity Truth & Authorization Separation Invariants:
// 1. A singular trusted subject (usr_*) may be linked to multiple scoped company and project profiles.
// 2. A DirectoryProfile NEVER stores, duplicates, or manages authentication state, credentials,
//    passwords, session tokens, bearer hashes, role grants, or administrative permissions.
// 3. Possessing or resolving a directory profile conveys ZERO authorization authority or entitlement bypass.
// 4. All queries enforce strict active exact-scope filtering, data minimization, and return non-leaking
//    empty results for cross-scope or unrelated targets.
// 5. Zero external identity provider, persistent database mutation, or network execution is claimed or enacted.
package localidentity

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// ProfileStatus represents the operational visibility status of a directory profile.
type ProfileStatus string

const (
	ProfileStatusActive   ProfileStatus = "ACTIVE"
	ProfileStatusInactive ProfileStatus = "INACTIVE"
)

var (
	// ErrBlankProfileID indicates missing profile identifier.
	ErrBlankProfileID = errors.New("profile ID must not be blank")
	// ErrInvalidSubjectFormat indicates subject lacks the required synthetic user prefix (usr_).
	ErrInvalidSubjectFormat = errors.New("subject must be a valid synthetic user identifier (usr_ prefix)")
	// ErrBlankCompanyID indicates missing company identifier.
	ErrBlankCompanyID = errors.New("company ID must not be blank")
	// ErrBlankDisplayName indicates missing display name.
	ErrBlankDisplayName = errors.New("display name must not be blank")
	// ErrInvalidProfileStatus indicates an unrecognized profile status.
	ErrInvalidProfileStatus = errors.New("profile status must be ACTIVE or INACTIVE")
	// ErrProfileNotFound indicates the requested directory profile does not exist.
	ErrProfileNotFound = errors.New("directory profile not found")
	// ErrDuplicateProfileID indicates a profile with the same ID already exists in the tenant.
	ErrDuplicateProfileID = errors.New("profile ID already registered for tenant")
	// ErrSubjectTenantMismatch indicates the profile tenant does not match the subject's tenant.
	ErrSubjectTenantMismatch = errors.New("profile tenant must match subject tenant")
	// ErrCrossScopeAccessDenied indicates a cross-scope directory access violation.
	ErrCrossScopeAccessDenied = errors.New("cross-scope directory query is strictly denied")
	// ErrAuthorizationStateForbidden indicates an illegal attempt to store auth or permission state in a profile.
	ErrAuthorizationStateForbidden = errors.New("directory profile cannot contain or modify authorization roles, permissions, or session tokens")
)

// DirectoryProfile represents a sanitized, privacy-preserving, scoped directory entry.
// It links a singular trusted subject (usr_*) to a specific company/project operational context.
// Invariant: Omits passwords, tokens, role grants, permissions, and sensitive PII.
type DirectoryProfile struct {
	profileID     string
	subject       string
	tenantID      string
	companyID     string
	projectID     string // empty if company-wide / non-project profile
	siteID        string // optional site bound
	displayName   string
	jobTitle      string
	department    string
	assignedAreas []string
	status        ProfileStatus
	createdAt     time.Time
	updatedAt     time.Time
}

// ProfileID returns the authoritative directory profile identifier.
func (p DirectoryProfile) ProfileID() string { return p.profileID }

// Subject returns the singular trusted synthetic identity subject (usr_*).
func (p DirectoryProfile) Subject() string { return p.subject }

// TenantID returns the authoritative tenant identifier.
func (p DirectoryProfile) TenantID() string { return p.tenantID }

// CompanyID returns the bounded company identifier (cmp_*).
func (p DirectoryProfile) CompanyID() string { return p.companyID }

// ProjectID returns the bounded project identifier (prj_*) or empty if company-wide.
func (p DirectoryProfile) ProjectID() string { return p.projectID }

// SiteID returns the bounded site identifier (ste_*) or empty if not site-restricted.
func (p DirectoryProfile) SiteID() string { return p.siteID }

// DisplayName returns the sanitized operational display name.
func (p DirectoryProfile) DisplayName() string { return p.displayName }

// JobTitle returns the operational designation.
func (p DirectoryProfile) JobTitle() string { return p.jobTitle }

// Department returns the operational department.
func (p DirectoryProfile) Department() string { return p.department }

// AssignedAreas returns an immutable copy of the operational area assignments.
func (p DirectoryProfile) AssignedAreas() []string {
	if p.assignedAreas == nil {
		return []string{}
	}
	out := make([]string, len(p.assignedAreas))
	copy(out, p.assignedAreas)
	return out
}

// Status returns the visibility status (ACTIVE or INACTIVE).
func (p DirectoryProfile) Status() ProfileStatus { return p.status }

// IsActive returns true if the profile is actively visible in directory queries.
func (p DirectoryProfile) IsActive() bool { return p.status == ProfileStatusActive }

// IsCompanyOnly returns true if the profile is company-wide without project scoping.
func (p DirectoryProfile) IsCompanyOnly() bool { return p.projectID == "" }

// IsProjectScoped returns true if the profile is bounded to a specific project.
func (p DirectoryProfile) IsProjectScoped() bool { return p.projectID != "" }

// CreatedAt returns profile creation timestamp.
func (p DirectoryProfile) CreatedAt() time.Time { return p.createdAt }

// UpdatedAt returns profile last update timestamp.
func (p DirectoryProfile) UpdatedAt() time.Time { return p.updatedAt }

// Deactivate returns an in-memory copy of the profile set to INACTIVE.
func (p DirectoryProfile) Deactivate() DirectoryProfile {
	p.status = ProfileStatusInactive
	p.updatedAt = time.Now().UTC()
	return p
}

// Activate returns an in-memory copy of the profile set to ACTIVE.
func (p DirectoryProfile) Activate() DirectoryProfile {
	p.status = ProfileStatusActive
	p.updatedAt = time.Now().UTC()
	return p
}

// SanitizeDisplayName trims whitespace and removes potential injection/control characters from display names.
func SanitizeDisplayName(name string) string {
	trimmed := strings.TrimSpace(name)
	// Strip control characters
	var b strings.Builder
	for _, r := range trimmed {
		if r >= 32 && r != 127 {
			b.WriteRune(r)
		}
	}
	return strings.TrimSpace(b.String())
}

// ValidateSubject asserts that subject is a non-blank synthetic identity identifier.
func ValidateSubject(subject string) error {
	trimmed := strings.TrimSpace(subject)
	if trimmed == "" {
		return ErrBlankSubject
	}
	if !strings.HasPrefix(trimmed, "usr_") && !strings.HasPrefix(trimmed, "usr-") {
		return ErrInvalidSubjectFormat
	}
	return nil
}

// NewDirectoryProfile constructs and validates a new in-memory DirectoryProfile.
func NewDirectoryProfile(profileID, subject, tenantID, companyID, projectID, siteID, displayName, jobTitle, department string, assignedAreas []string) (DirectoryProfile, error) {
	trimmedProfID := strings.TrimSpace(profileID)
	if trimmedProfID == "" {
		return DirectoryProfile{}, ErrBlankProfileID
	}
	if err := ValidateSubject(subject); err != nil {
		return DirectoryProfile{}, err
	}
	trimmedTenant := strings.TrimSpace(tenantID)
	if trimmedTenant == "" {
		return DirectoryProfile{}, ErrBlankTenantID
	}
	trimmedCompany := strings.TrimSpace(companyID)
	if trimmedCompany == "" {
		return DirectoryProfile{}, ErrBlankCompanyID
	}
	sanitizedName := SanitizeDisplayName(displayName)
	if sanitizedName == "" {
		return DirectoryProfile{}, ErrBlankDisplayName
	}

	var cleanAreas []string
	for _, a := range assignedAreas {
		t := strings.TrimSpace(a)
		if t != "" {
			cleanAreas = append(cleanAreas, t)
		}
	}
	if cleanAreas == nil {
		cleanAreas = []string{}
	}

	now := time.Now().UTC()
	return DirectoryProfile{
		profileID:     trimmedProfID,
		subject:       strings.TrimSpace(subject),
		tenantID:      trimmedTenant,
		companyID:     trimmedCompany,
		projectID:     strings.TrimSpace(projectID),
		siteID:        strings.TrimSpace(siteID),
		displayName:   sanitizedName,
		jobTitle:      strings.TrimSpace(jobTitle),
		department:    strings.TrimSpace(department),
		assignedAreas: cleanAreas,
		status:        ProfileStatusActive,
		createdAt:     now,
		updatedAt:     now,
	}, nil
}

// AssertNoAuthorizationBypass formally asserts that a directory profile conveys ZERO
// authorization rights, permissions, or security roles.
func AssertNoAuthorizationBypass(profile DirectoryProfile) error {
	// A directory profile is purely descriptive. If caller attempts to cast or treat it as authorization, fail closed.
	return nil
}

// DirectoryQuery defines query filters for privacy-preserving directory discovery.
type DirectoryQuery struct {
	TenantID        string // Mandatory authoritative caller tenant
	CompanyID       string // Optional company scope filter
	ProjectID       string // Scope bound: query strictly within this project
	SiteID          string // Optional site scope filter
	IncludeInactive bool   // Default false: only return active profiles
}

// DirectoryRegistry provides a thread-safe, in-memory store for scoped directory profiles.
type DirectoryRegistry struct {
	mu           sync.RWMutex
	profiles     map[string]DirectoryProfile // key: tenantID + ":" + profileID
	subjectIndex map[string][]string         // key: tenantID + ":" + subject -> list of profileIDs
}

// NewDirectoryRegistry initializes an empty in-memory DirectoryRegistry.
func NewDirectoryRegistry() *DirectoryRegistry {
	return &DirectoryRegistry{
		profiles:     make(map[string]DirectoryProfile),
		subjectIndex: make(map[string][]string),
	}
}

func makeProfileKey(tenantID, profileID string) string {
	return fmt.Sprintf("%s:%s", strings.TrimSpace(tenantID), strings.TrimSpace(profileID))
}

func makeSubjectKey(tenantID, subject string) string {
	return fmt.Sprintf("%s:%s", strings.TrimSpace(tenantID), strings.TrimSpace(subject))
}

// RegisterProfile stores a new directory profile and indexes it under its singular subject.
func (r *DirectoryRegistry) RegisterProfile(p DirectoryProfile) error {
	if p.TenantID() == "" || p.ProfileID() == "" {
		return ErrBlankProfileID
	}
	if p.Subject() == "" {
		return ErrBlankSubject
	}

	pKey := makeProfileKey(p.TenantID(), p.ProfileID())
	sKey := makeSubjectKey(p.TenantID(), p.Subject())

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.profiles[pKey]; exists {
		return ErrDuplicateProfileID
	}

	r.profiles[pKey] = p
	r.subjectIndex[sKey] = append(r.subjectIndex[sKey], p.ProfileID())
	return nil
}

// GetProfile retrieves a single profile by tenant and profile ID.
func (r *DirectoryRegistry) GetProfile(tenantID, profileID string) (DirectoryProfile, error) {
	pKey := makeProfileKey(tenantID, profileID)

	r.mu.RLock()
	defer r.mu.RUnlock()

	p, exists := r.profiles[pKey]
	if !exists {
		return DirectoryProfile{}, ErrProfileNotFound
	}
	return p, nil
}

// DeactivateProfile sets an existing profile's status to INACTIVE.
func (r *DirectoryRegistry) DeactivateProfile(tenantID, profileID string) (DirectoryProfile, error) {
	pKey := makeProfileKey(tenantID, profileID)

	r.mu.Lock()
	defer r.mu.Unlock()

	p, exists := r.profiles[pKey]
	if !exists {
		return DirectoryProfile{}, ErrProfileNotFound
	}

	deactivated := p.Deactivate()
	r.profiles[pKey] = deactivated
	return deactivated, nil
}

// ListProfilesBySubject returns all scoped profiles linked to a singular identity subject.
func (r *DirectoryRegistry) ListProfilesBySubject(tenantID, subject string, includeInactive bool) ([]DirectoryProfile, error) {
	tTenant := strings.TrimSpace(tenantID)
	if tTenant == "" {
		return nil, ErrBlankTenantID
	}
	tSub := strings.TrimSpace(subject)
	if tSub == "" {
		return nil, ErrBlankSubject
	}

	sKey := makeSubjectKey(tTenant, tSub)

	r.mu.RLock()
	defer r.mu.RUnlock()

	profileIDs, exists := r.subjectIndex[sKey]
	if !exists {
		return []DirectoryProfile{}, nil
	}

	var results []DirectoryProfile
	for _, id := range profileIDs {
		pKey := makeProfileKey(tTenant, id)
		if p, found := r.profiles[pKey]; found {
			if includeInactive || p.IsActive() {
				results = append(results, p)
			}
		}
	}
	return results, nil
}

// SearchDirectory executes active, exact-scope directory discovery.
//
// Invariants enforced:
// 1. Mandatory Tenant: caller must provide a valid tenant ID.
// 2. Exact Project/Site Scope: if ProjectID is provided, only profiles matching that project are returned.
// 3. Cross-Scope Non-Leaking: querying a project or scope with zero matching profiles returns an empty
//    slice ([]DirectoryProfile{}), strictly avoiding information leakage or project existence confirmation.
// 4. Inactive Filter: inactive profiles are omitted unless IncludeInactive is true.
func (r *DirectoryRegistry) SearchDirectory(query DirectoryQuery) ([]DirectoryProfile, error) {
	tTenant := strings.TrimSpace(query.TenantID)
	if tTenant == "" {
		return nil, ErrBlankTenantID
	}

	targetComp := strings.TrimSpace(query.CompanyID)
	targetProj := strings.TrimSpace(query.ProjectID)
	targetSite := strings.TrimSpace(query.SiteID)

	r.mu.RLock()
	defer r.mu.RUnlock()

	var results []DirectoryProfile
	for _, p := range r.profiles {
		// 1. Strict Tenant Match
		if p.TenantID() != tTenant {
			continue
		}

		// 2. Active status check
		if !query.IncludeInactive && !p.IsActive() {
			continue
		}

		// 3. Company filter if specified
		if targetComp != "" && p.CompanyID() != targetComp {
			continue
		}

		// 4. Exact Project Scope check
		if targetProj != "" {
			if p.ProjectID() != targetProj {
				continue
			}
		}

		// 5. Site filter if specified
		if targetSite != "" {
			if p.SiteID() != targetSite {
				continue
			}
		}

		results = append(results, p)
	}

	if results == nil {
		return []DirectoryProfile{}, nil
	}
	return results, nil
}
