// Package localidentity provides local identity, authorization, and directory services for OSHE Platform.
//
// PROVISIONAL GOVERNANCE DECLARATION (H030-003, H030-004, H030-005):
// Under approved Sole Human Owner decision H030-005 (Privacy, Data Minimization, and Directory Visibility),
// this file implements local in-memory scoped directory visibility, search, attribute-exposure,
// and read-boundary controls.
//
// Invariants Enforced (NFR-V030-PRIV-001 / NEG-V030-04):
// 1. Exact-Scope Partitioning (NFR-V030-PRIV-001): Directory discovery is strictly partitioned
//    to the caller's authorized project and organization boundary.
// 2. Anti-Enumeration (NEG-V030-04): Queries targeting cross-project or unassigned contexts
//    return empty slices ([]MinimizedDirectoryProfile{}) without leaking project existence or errors.
// 3. Data Minimization: Exposed profiles contain ONLY sanitized operational attributes. Personal
//    email, phone numbers, national identifiers, passwords, bearer tokens, and security credentials
//    are strictly excluded.
// 4. Role-Bounded Read Controls: Unauthenticated callers or callers lacking PermDirectoryRead
//    are denied access. External contractors (RoleContractor) are bounded strictly to their
//    assigned project and site.
// 5. Zero External Provider: Operates purely in-memory with synthetic test fixtures. Zero external
//    identity provider, network route, or production runtime is accessed or claimed.
package localidentity

import (
	"errors"
	"fmt"
	"strings"
)

const (
	// DefaultSearchLimit specifies the default maximum number of directory entries returned.
	DefaultSearchLimit = 50
	// MaxSearchLimit specifies the hard ceiling on directory query pagination to prevent mass harvesting.
	MaxSearchLimit = 100
)

var (
	// ErrUnauthenticatedCaller indicates an anonymous or unauthenticated caller attempted directory access.
	ErrUnauthenticatedCaller = errors.New("unauthenticated caller cannot access directory")
	// ErrDirectoryReadPermissionDenied indicates caller lacks the required directory read permission.
	ErrDirectoryReadPermissionDenied = errors.New("caller lacks iam:directory:read permission")
	// ErrDataMinimizationViolation indicates a directory profile contains forbidden credentials or sensitive PII.
	ErrDataMinimizationViolation = errors.New("profile violates data minimization constraints")
)

// ViewerContext encapsulates the authenticated caller's identity, role, and operational scope.
type ViewerContext struct {
	Identity SubjectIdentity
	Role     Role
	Scope    ScopeGrant
}

// Validate asserts that the caller context has valid authentication claims and tenant consistency.
func (v ViewerContext) Validate() error {
	if !v.Identity.IsAuthenticated {
		return ErrUnauthenticatedCaller
	}
	if strings.TrimSpace(v.Identity.Subject) == "" {
		return ErrBlankSubject
	}
	if strings.TrimSpace(v.Identity.TenantID) == "" {
		return ErrBlankTenantID
	}
	if v.Scope.TenantID != "" && v.Scope.TenantID != v.Identity.TenantID {
		return ErrSubjectTenantMismatch
	}
	return nil
}

// MinimizedDirectoryProfile represents a privacy-preserving, data-minimized projection of a directory profile.
// It exposes only safe operational attributes and strictly excludes personal contact info, credentials, and tokens.
type MinimizedDirectoryProfile struct {
	ProfileID     string        `json:"profile_id"`
	Subject       string        `json:"subject"`
	TenantID      string        `json:"tenant_id"`
	CompanyID     string        `json:"company_id"`
	ProjectID     string        `json:"project_id,omitempty"`
	SiteID        string        `json:"site_id,omitempty"`
	DisplayName   string        `json:"display_name"`
	JobTitle      string        `json:"job_title"`
	Department    string        `json:"department,omitempty"`
	AssignedAreas []string      `json:"assigned_areas,omitempty"`
	Status        ProfileStatus `json:"status"`
}

// MinimizeProfile transforms a full DirectoryProfile into a sanitized MinimizedDirectoryProfile.
func MinimizeProfile(p DirectoryProfile) MinimizedDirectoryProfile {
	return MinimizedDirectoryProfile{
		ProfileID:     p.ProfileID(),
		Subject:       p.Subject(),
		TenantID:      p.TenantID(),
		CompanyID:     p.CompanyID(),
		ProjectID:     p.ProjectID(),
		SiteID:        p.SiteID(),
		DisplayName:   p.DisplayName(),
		JobTitle:      p.JobTitle(),
		Department:    p.Department(),
		AssignedAreas: p.AssignedAreas(),
		Status:        p.Status(),
	}
}

// AssertDataMinimization verifies that a minimized profile contains only permitted operational attributes
// and contains zero forbidden sensitive fields (credentials, raw PII, etc.).
func AssertDataMinimization(p MinimizedDirectoryProfile) error {
	forbiddenSubstrings := []string{
		"password", "secret", "token", "oshe_tok_", "bearer", "session_id",
		"@gmail.com", "@yahoo.com", "@hotmail.com",
		"+66", "081-", "089-",
		"ssn", "national_id", "salary", "credit_card",
	}

	corpus := strings.ToLower(fmt.Sprintf("%s %s %s %s %s",
		p.DisplayName, p.JobTitle, p.Department, p.ProfileID, p.Subject))

	for _, f := range forbiddenSubstrings {
		if strings.Contains(corpus, f) {
			return fmt.Errorf("%w: profile contains sensitive pattern %q", ErrDataMinimizationViolation, f)
		}
	}
	return nil
}

// DirectorySearchFilter defines caller-specified criteria for searching the scoped directory.
type DirectorySearchFilter struct {
	Query           string // Case-insensitive text search matching DisplayName, JobTitle, or Department
	CompanyID       string // Optional company filter (must be within caller's scope)
	ProjectID       string // Optional project filter (must be within caller's scope)
	SiteID          string // Optional site filter (must be within caller's scope)
	IncludeInactive bool   // Request inactive profiles (permitted only to TenantAdmin and Auditor)
	Limit           int    // Max results (defaults to DefaultSearchLimit, bounded by MaxSearchLimit)
	Offset          int    // Offset for pagination
}

// DirectoryVisibilityService provides scoped, privacy-partitioned directory access,
// anti-enumeration protection (NEG-V030-04), and data minimization enforcement.
type DirectoryVisibilityService struct {
	registry *DirectoryRegistry
	matrix   AuthorizationMatrix
}

// NewDirectoryVisibilityService constructs a DirectoryVisibilityService bound to a registry and matrix.
func NewDirectoryVisibilityService(registry *DirectoryRegistry, matrix AuthorizationMatrix) *DirectoryVisibilityService {
	return &DirectoryVisibilityService{
		registry: registry,
		matrix:   matrix,
	}
}

// SearchDirectory executes scoped directory search partitioned to caller's authorized boundary.
//
// Invariants Enforced:
// 1. Authentication & Permission: Caller must be authenticated and possess PermDirectoryRead.
// 2. Tenant Isolation: Query is strictly locked to caller's TenantID.
// 3. Exact Scope Partitioning (NFR-V030-PRIV-001):
//   - If caller is project-scoped (RoleInspector, RoleProjectManager, RoleContractor, etc.),
//     queries are automatically partitioned to caller's assigned ProjectID.
//   - If caller explicitly requests a project outside their scope, ANTI-ENUMERATION (NEG-V030-04)
//     returns an empty slice ([]MinimizedDirectoryProfile{}) without leaking project existence.
//
// 4. Inactive Profile Shielding: Inactive profiles are omitted unless caller holds TenantAdmin or Auditor.
// 5. Anti-Harvesting: Query limits are clamped between 1 and MaxSearchLimit (100).
func (s *DirectoryVisibilityService) SearchDirectory(viewer ViewerContext, filter DirectorySearchFilter) ([]MinimizedDirectoryProfile, error) {
	if err := viewer.Validate(); err != nil {
		return nil, err
	}

	// 1. PermDirectoryRead verification
	if !s.matrix.RoleHasPermission(viewer.Role, PermDirectoryRead) {
		return nil, ErrDirectoryReadPermissionDenied
	}

	// 2. Anti-Harvesting Bounds
	limit := filter.Limit
	if limit <= 0 {
		limit = DefaultSearchLimit
	} else if limit > MaxSearchLimit {
		limit = MaxSearchLimit
	}

	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	// 3. Resolve viewer scope depth
	viewerScopeLevel := ResolveScopeLevel(viewer.Scope)

	// 4. Anti-enumeration and scope partitioning
	targetProjectID := strings.TrimSpace(filter.ProjectID)
	targetCompanyID := strings.TrimSpace(filter.CompanyID)
	targetSiteID := strings.TrimSpace(filter.SiteID)

	if viewerScopeLevel == ScopeLevelProject || viewerScopeLevel == ScopeLevelSite || viewerScopeLevel == ScopeLevelArea {
		if viewer.Scope.ProjectID != "" {
			// Anti-enumeration: if caller explicitly requested a different project, return empty results
			if targetProjectID != "" && targetProjectID != viewer.Scope.ProjectID {
				return []MinimizedDirectoryProfile{}, nil
			}
			// Automatically partition to caller's assigned project
			targetProjectID = viewer.Scope.ProjectID
		}
		if viewerScopeLevel == ScopeLevelSite && viewer.Scope.SiteID != "" {
			if targetSiteID != "" && targetSiteID != viewer.Scope.SiteID {
				return []MinimizedDirectoryProfile{}, nil
			}
			targetSiteID = viewer.Scope.SiteID
		}
	} else if viewerScopeLevel == ScopeLevelCompany {
		if viewer.Scope.CompanyID != "" {
			if targetCompanyID != "" && targetCompanyID != viewer.Scope.CompanyID {
				return []MinimizedDirectoryProfile{}, nil
			}
			targetCompanyID = viewer.Scope.CompanyID
		}
	}

	// External contractor boundary: contractors can only see profiles in their assigned project/site
	if viewer.Role == RoleContractor {
		if viewer.Scope.ProjectID == "" {
			return []MinimizedDirectoryProfile{}, nil
		}
		if targetProjectID != "" && targetProjectID != viewer.Scope.ProjectID {
			return []MinimizedDirectoryProfile{}, nil
		}
		targetProjectID = viewer.Scope.ProjectID
	}

	// 5. Inactive profile shielding
	canViewInactive := (viewer.Role == RoleTenantAdmin || viewer.Role == RoleAuditor) && filter.IncludeInactive

	// 6. Execute registry query
	regQuery := DirectoryQuery{
		TenantID:        viewer.Identity.TenantID,
		CompanyID:       targetCompanyID,
		ProjectID:       targetProjectID,
		SiteID:          targetSiteID,
		IncludeInactive: canViewInactive,
	}

	rawProfiles, err := s.registry.SearchDirectory(regQuery)
	if err != nil {
		return nil, err
	}

	// 7. Filter by text query and boundary checks
	textQuery := strings.ToLower(strings.TrimSpace(filter.Query))
	var matched []MinimizedDirectoryProfile

	for _, p := range rawProfiles {
		// Project-level isolation: if caller is project-scoped, reject any profile outside caller's project
		if viewer.Scope.ProjectID != "" && p.ProjectID() != viewer.Scope.ProjectID {
			continue
		}
		// Site-level isolation: if caller is site-scoped, reject non-matching site
		if viewerScopeLevel == ScopeLevelSite && viewer.Scope.SiteID != "" && p.SiteID() != "" && p.SiteID() != viewer.Scope.SiteID {
			continue
		}

		if textQuery != "" {
			nameMatch := strings.Contains(strings.ToLower(p.DisplayName()), textQuery)
			titleMatch := strings.Contains(strings.ToLower(p.JobTitle()), textQuery)
			deptMatch := strings.Contains(strings.ToLower(p.Department()), textQuery)
			if !nameMatch && !titleMatch && !deptMatch {
				continue
			}
		}

		minProfile := MinimizeProfile(p)
		matched = append(matched, minProfile)
	}

	if matched == nil {
		return []MinimizedDirectoryProfile{}, nil
	}

	// 8. Pagination offset and limit
	if offset >= len(matched) {
		return []MinimizedDirectoryProfile{}, nil
	}
	end := offset + limit
	if end > len(matched) {
		end = len(matched)
	}

	return matched[offset:end], nil
}

// GetVisibleProfile retrieves a single profile by ID, verifying that it is visible within caller's scope.
// If the profile does not exist, belongs to another project, or is inactive without permission,
// it returns ErrProfileNotFound to prevent object existence enumeration.
func (s *DirectoryVisibilityService) GetVisibleProfile(viewer ViewerContext, profileID string) (MinimizedDirectoryProfile, error) {
	if err := viewer.Validate(); err != nil {
		return MinimizedDirectoryProfile{}, err
	}

	if !s.matrix.RoleHasPermission(viewer.Role, PermDirectoryRead) {
		return MinimizedDirectoryProfile{}, ErrDirectoryReadPermissionDenied
	}

	trimmedProfID := strings.TrimSpace(profileID)
	if trimmedProfID == "" {
		return MinimizedDirectoryProfile{}, ErrBlankProfileID
	}

	profile, err := s.registry.GetProfile(viewer.Identity.TenantID, trimmedProfID)
	if err != nil {
		return MinimizedDirectoryProfile{}, ErrProfileNotFound
	}

	// Inactive check: only TenantAdmin and Auditor may view inactive profiles
	if !profile.IsActive() {
		if viewer.Role != RoleTenantAdmin && viewer.Role != RoleAuditor {
			return MinimizedDirectoryProfile{}, ErrProfileNotFound
		}
	}

	// Scope visibility check
	viewerScopeLevel := ResolveScopeLevel(viewer.Scope)
	if viewerScopeLevel == ScopeLevelProject || viewerScopeLevel == ScopeLevelSite || viewerScopeLevel == ScopeLevelArea {
		if viewer.Scope.ProjectID != "" && profile.ProjectID() != viewer.Scope.ProjectID {
			return MinimizedDirectoryProfile{}, ErrProfileNotFound
		}
		if viewerScopeLevel == ScopeLevelSite && viewer.Scope.SiteID != "" && profile.SiteID() != "" && profile.SiteID() != viewer.Scope.SiteID {
			return MinimizedDirectoryProfile{}, ErrProfileNotFound
		}
	} else if viewerScopeLevel == ScopeLevelCompany {
		if viewer.Scope.CompanyID != "" && profile.CompanyID() != viewer.Scope.CompanyID {
			return MinimizedDirectoryProfile{}, ErrProfileNotFound
		}
	}

	if viewer.Role == RoleContractor {
		if viewer.Scope.ProjectID != "" && profile.ProjectID() != viewer.Scope.ProjectID {
			return MinimizedDirectoryProfile{}, ErrProfileNotFound
		}
	}

	return MinimizeProfile(profile), nil
}

// ListVisibleProfilesBySubject retrieves all profiles for a singular subject that are visible
// within the caller's authorized scope. Cross-project profiles for the same subject are omitted.
func (s *DirectoryVisibilityService) ListVisibleProfilesBySubject(viewer ViewerContext, subject string) ([]MinimizedDirectoryProfile, error) {
	if err := viewer.Validate(); err != nil {
		return nil, err
	}

	if !s.matrix.RoleHasPermission(viewer.Role, PermDirectoryRead) {
		return nil, ErrDirectoryReadPermissionDenied
	}

	canViewInactive := viewer.Role == RoleTenantAdmin || viewer.Role == RoleAuditor
	profiles, err := s.registry.ListProfilesBySubject(viewer.Identity.TenantID, subject, canViewInactive)
	if err != nil {
		return nil, err
	}

	viewerScopeLevel := ResolveScopeLevel(viewer.Scope)
	var visible []MinimizedDirectoryProfile

	for _, p := range profiles {
		if viewerScopeLevel == ScopeLevelProject || viewerScopeLevel == ScopeLevelSite || viewerScopeLevel == ScopeLevelArea {
			if viewer.Scope.ProjectID != "" && p.ProjectID() != viewer.Scope.ProjectID {
				continue
			}
		} else if viewerScopeLevel == ScopeLevelCompany {
			if viewer.Scope.CompanyID != "" && p.CompanyID() != viewer.Scope.CompanyID {
				continue
			}
		}

		if viewer.Role == RoleContractor {
			if viewer.Scope.ProjectID != "" && p.ProjectID() != viewer.Scope.ProjectID {
				continue
			}
		}

		visible = append(visible, MinimizeProfile(p))
	}

	if visible == nil {
		return []MinimizedDirectoryProfile{}, nil
	}
	return visible, nil
}
