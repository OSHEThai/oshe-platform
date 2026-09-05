// Package portal provides synthetic internal portal models, role-scoped navigation,
// bounded work queues, and safe authorized-content access for OSHE Platform.
//
// PROVISIONAL GOVERNANCE DECLARATION (H030-006, Issue #98):
// Under approved Sole Human Owner decisions, this package implements in-memory,
// dependency-free local models for internal portal audience resolution, role-scoped navigation,
// bounded work queues, and non-leaking empty/denied states.
//
// Strict Scoping Invariants:
// 1. Role-Scoped Navigation: Callers only see navigation items explicitly authorized for their active role.
//    Contractors, Inspectors, and Auditors cannot discover administrative surfaces.
// 2. Exact Project/Site Scope Bounding: Work queues and content views are partitioned strictly to the caller's
//    authorized tenant and project. Unrelated projects or sibling companies are inaccessible.
// 3. Non-Leaking Diagnostics: Cross-scope or unauthorized requests return non-leaking empty results
//    or generic denied status, preventing project existence reconnaissance.
// 4. Accessibility Compliance: All navigation elements mandate descriptive, non-empty text labels and ARIA labels.
// 5. Zero Runtime/UI Deployment: Operates strictly on synthetic in-memory fixtures. No live UI, service,
//    identity provider, or persistent database is connected or claimed.
package portal

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// PortalRole classifies internal portal user roles.
type PortalRole string

const (
	RoleTenantAdmin    PortalRole = "TENANT_ADMIN"
	RoleProjectManager PortalRole = "PROJECT_MANAGER"
	RoleInspector      PortalRole = "INSPECTOR"
	RoleAuditor        PortalRole = "AUDITOR"
	RoleViewer         PortalRole = "VIEWER"
	RoleContractor     PortalRole = "CONTRACTOR"
)

var (
	// ErrUnauthenticatedViewer indicates viewer is not authenticated.
	ErrUnauthenticatedViewer = errors.New("unauthenticated portal viewer")
	// ErrBlankTenantID indicates missing tenant identifier.
	ErrBlankTenantID = errors.New("tenant ID must not be blank")
	// ErrBlankSubject indicates missing viewer subject.
	ErrBlankSubject = errors.New("subject must not be blank")
	// ErrAccessDenied indicates generic non-leaking access denial.
	ErrAccessDenied = errors.New("portal access denied")
	// ErrScopeMismatch indicates viewer scope does not match requested resource.
	ErrScopeMismatch = errors.New("viewer scope does not match requested target")
	// ErrInvalidRole indicates an unapproved portal role.
	ErrInvalidRole = errors.New("invalid or unrecognized portal role")
	// ErrEmptyLabel indicates missing accessible label.
	ErrEmptyLabel = errors.New("accessibility violation: label must not be empty")
	// ErrEmptyAriaLabel indicates missing accessibility ARIA label.
	ErrEmptyAriaLabel = errors.New("accessibility violation: aria-label must not be empty")
)

// KnownPortalRoles catalogs the recognized internal portal roles.
var KnownPortalRoles = map[PortalRole]bool{
	RoleTenantAdmin:    true,
	RoleProjectManager: true,
	RoleInspector:      true,
	RoleAuditor:        true,
	RoleViewer:         true,
	RoleContractor:     true,
}

// PortalViewer encapsulates the caller context for portal presentation.
type PortalViewer struct {
	Subject         string
	TenantID        string
	CompanyID       string
	ProjectID       string
	SiteID          string
	Role            PortalRole
	IsAuthenticated bool
}

// Validate asserts that the viewer context is authenticated and valid.
func (v PortalViewer) Validate() error {
	if !v.IsAuthenticated {
		return ErrUnauthenticatedViewer
	}
	if strings.TrimSpace(v.Subject) == "" {
		return ErrBlankSubject
	}
	if strings.TrimSpace(v.TenantID) == "" {
		return ErrBlankTenantID
	}
	if !KnownPortalRoles[v.Role] {
		return ErrInvalidRole
	}
	return nil
}

// NavigationItem represents an accessible, role-guarded portal navigation link.
type NavigationItem struct {
	ID            string       `json:"id"`
	Label         string       `json:"label"`
	AriaLabel     string       `json:"aria_label"`
	Path          string       `json:"path"`
	RequiredRoles []PortalRole `json:"required_roles"`
	ScopeLevel    string       `json:"scope_level"`
}

// ValidateAccessibility ensures the item meets WCAG / a11y non-empty label requirements.
func (item NavigationItem) ValidateAccessibility() error {
	if strings.TrimSpace(item.Label) == "" {
		return ErrEmptyLabel
	}
	if strings.TrimSpace(item.AriaLabel) == "" {
		return ErrEmptyAriaLabel
	}
	return nil
}

// StandardNavigationCatalog defines the authoritative set of internal portal navigation items.
var StandardNavigationCatalog = []NavigationItem{
	{
		ID:            "nav_tenant_admin",
		Label:         "Tenant Administration",
		AriaLabel:     "Manage tenant-wide policies and organization hierarchy",
		Path:          "/portal/admin",
		RequiredRoles: []PortalRole{RoleTenantAdmin},
		ScopeLevel:    "TENANT",
	},
	{
		ID:            "nav_audit_exports",
		Label:         "Compliance Audit Exports",
		AriaLabel:     "Export sealed immutable audit packages and compliance reports",
		Path:          "/portal/audit/exports",
		RequiredRoles: []PortalRole{RoleTenantAdmin, RoleAuditor},
		ScopeLevel:    "TENANT",
	},
	{
		ID:            "nav_project_queue",
		Label:         "Project Oversight Queue",
		AriaLabel:     "Review pending inspections and approve site safety findings",
		Path:          "/portal/project/queue",
		RequiredRoles: []PortalRole{RoleTenantAdmin, RoleProjectManager},
		ScopeLevel:    "PROJECT",
	},
	{
		ID:            "nav_contractor_mgmt",
		Label:         "Contractor Oversight",
		AriaLabel:     "Review contractor task allocations and site safety compliance",
		Path:          "/portal/project/contractors",
		RequiredRoles: []PortalRole{RoleTenantAdmin, RoleProjectManager},
		ScopeLevel:    "PROJECT",
	},
	{
		ID:            "nav_field_inspections",
		Label:         "Field Inspections",
		AriaLabel:     "Conduct and log safety inspections in assigned areas",
		Path:          "/portal/inspections",
		RequiredRoles: []PortalRole{RoleTenantAdmin, RoleProjectManager, RoleInspector},
		ScopeLevel:    "SITE",
	},
	{
		ID:            "nav_my_tasks",
		Label:         "Work Queue",
		AriaLabel:     "View and execute assigned safety tasks and corrective actions",
		Path:          "/portal/workqueue",
		RequiredRoles: []PortalRole{RoleTenantAdmin, RoleProjectManager, RoleInspector, RoleContractor},
		ScopeLevel:    "PROJECT",
	},
	{
		ID:            "nav_contractor_remediation",
		Label:         "Assigned Findings",
		AriaLabel:     "View safety findings requiring contractor corrective action",
		Path:          "/portal/findings/assigned",
		RequiredRoles: []PortalRole{RoleContractor},
		ScopeLevel:    "SITE",
	},
	{
		ID:            "nav_read_reports",
		Label:         "Safety Overview",
		AriaLabel:     "Read-only access to published inspection summaries and safety metrics",
		Path:          "/portal/reports",
		RequiredRoles: []PortalRole{RoleTenantAdmin, RoleProjectManager, RoleInspector, RoleAuditor, RoleViewer},
		ScopeLevel:    "PROJECT",
	},
}

// NavigationMenu represents the resolved, role-contained navigation menu.
type NavigationMenu struct {
	ViewerRole PortalRole       `json:"viewer_role"`
	TenantID   string           `json:"tenant_id"`
	ProjectID  string           `json:"project_id,omitempty"`
	Items      []NavigationItem `json:"items"`
}

// ResolveNavigation filters the navigation catalog strictly to items permitted for the viewer's active role.
func ResolveNavigation(viewer PortalViewer) (NavigationMenu, error) {
	if err := viewer.Validate(); err != nil {
		return NavigationMenu{}, err
	}

	var allowedItems []NavigationItem
	for _, item := range StandardNavigationCatalog {
		if err := item.ValidateAccessibility(); err != nil {
			return NavigationMenu{}, err
		}

		isRoleAllowed := false
		for _, r := range item.RequiredRoles {
			if r == viewer.Role {
				isRoleAllowed = true
				break
			}
		}

		if isRoleAllowed {
			allowedItems = append(allowedItems, item)
		}
	}

	if allowedItems == nil {
		allowedItems = []NavigationItem{}
	}

	return NavigationMenu{
		ViewerRole: viewer.Role,
		TenantID:   viewer.TenantID,
		ProjectID:  viewer.ProjectID,
		Items:      allowedItems,
	}, nil
}

// WorkQueueItem represents a pending operational task bounded to a specific tenant and project.
type WorkQueueItem struct {
	QueueID       string    `json:"queue_id"`
	TenantID      string    `json:"tenant_id"`
	ProjectID     string    `json:"project_id"`
	SiteID        string    `json:"site_id,omitempty"`
	TaskType      string    `json:"task_type"`
	Title         string    `json:"title"`
	AriaLabel     string    `json:"aria_label"`
	AssignedRole  PortalRole `json:"assigned_role"`
	AssignedTo    string    `json:"assigned_to,omitempty"`
	Priority      string    `json:"priority"`
	CreatedAt     time.Time `json:"created_at"`
}

// WorkQueue represents the resolved work queue presented to a viewer.
type WorkQueue struct {
	TenantID         string          `json:"tenant_id"`
	ProjectID        string          `json:"project_id"`
	TotalCount       int             `json:"total_count"`
	Items            []WorkQueueItem `json:"items"`
	EmptyStateNotice string          `json:"empty_state_notice"`
}

// ResolveWorkQueue filters candidate work items strictly matching the viewer's tenant, project, and role.
// Returns a non-leaking empty state when no items exist for the viewer's scope.
func ResolveWorkQueue(viewer PortalViewer, allItems []WorkQueueItem) (WorkQueue, error) {
	if err := viewer.Validate(); err != nil {
		return WorkQueue{}, err
	}

	trimmedProj := strings.TrimSpace(viewer.ProjectID)
	// Project-scoped roles require an active project scope
	if trimmedProj == "" && viewer.Role != RoleTenantAdmin {
		return WorkQueue{}, ErrScopeMismatch
	}

	var matchingItems []WorkQueueItem
	for _, item := range allItems {
		// 1. Strict Tenant Isolation
		if item.TenantID != viewer.TenantID {
			continue
		}

		// 2. Strict Project Isolation (TenantAdmin can observe all within tenant if ProjectID is empty)
		if trimmedProj != "" && item.ProjectID != trimmedProj {
			continue
		}

		// 3. Site containment if viewer is site-restricted
		if strings.TrimSpace(viewer.SiteID) != "" && item.SiteID != "" && item.SiteID != strings.TrimSpace(viewer.SiteID) {
			continue
		}

		// 4. Role relevance check
		if viewer.Role != RoleTenantAdmin {
			if item.AssignedRole != viewer.Role && item.AssignedTo != viewer.Subject {
				continue
			}
		}

		matchingItems = append(matchingItems, item)
	}

	if matchingItems == nil {
		matchingItems = []WorkQueueItem{}
	}

	notice := ""
	if len(matchingItems) == 0 {
		notice = "No work items pending for current operational scope."
	}

	return WorkQueue{
		TenantID:         viewer.TenantID,
		ProjectID:        viewer.ProjectID,
		TotalCount:       len(matchingItems),
		Items:            matchingItems,
		EmptyStateNotice: notice,
	}, nil
}

// TargetContent models an internal resource request.
type TargetContent struct {
	ContentID     string
	TenantID      string
	ProjectID     string
	SiteID        string
	RequiredRoles []PortalRole
	Title         string
	Body          string
}

// PortalContentView represents the sanitized, authorized content presentation.
type PortalContentView struct {
	ContentID string `json:"content_id"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	Scope     string `json:"scope"`
}

// ViewContent evaluates whether a viewer is authorized to view a specific content item.
// Invariant: Non-leaking denial. Cross-tenant, cross-project, or unauthorized role queries fail closed with ErrAccessDenied.
func ViewContent(viewer PortalViewer, content TargetContent) (PortalContentView, error) {
	if err := viewer.Validate(); err != nil {
		return PortalContentView{}, err
	}

	// 1. Strict Tenant Match
	if viewer.TenantID != content.TenantID {
		return PortalContentView{}, ErrAccessDenied
	}

	// 2. Strict Project Match
	if viewer.Role != RoleTenantAdmin {
		if strings.TrimSpace(viewer.ProjectID) == "" || viewer.ProjectID != content.ProjectID {
			return PortalContentView{}, ErrAccessDenied
		}
	}

	// 3. Site match if viewer is site-restricted
	if strings.TrimSpace(viewer.SiteID) != "" && content.SiteID != "" && viewer.SiteID != content.SiteID {
		return PortalContentView{}, ErrAccessDenied
	}

	// 4. Role Authorization
	roleAllowed := false
	if viewer.Role == RoleTenantAdmin {
		roleAllowed = true
	} else {
		for _, r := range content.RequiredRoles {
			if r == viewer.Role {
				roleAllowed = true
				break
			}
		}
	}

	if !roleAllowed {
		return PortalContentView{}, ErrAccessDenied
	}

	scopeDesc := fmt.Sprintf("%s/%s", content.TenantID, content.ProjectID)
	if content.SiteID != "" {
		scopeDesc = fmt.Sprintf("%s/%s", scopeDesc, content.SiteID)
	}

	return PortalContentView{
		ContentID: content.ContentID,
		Title:     content.Title,
		Body:      content.Body,
		Scope:     scopeDesc,
	}, nil
}
