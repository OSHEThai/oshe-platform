// Package portal provides synthetic internal portal models, role-scoped navigation,
// bounded work queues, and safe authorized-content access for OSHE Platform.
//
// PROVISIONAL GOVERNANCE DECLARATION (Issue #140 / V040-I029):
// Under approved Sole Human Owner decision HDEC-V040-FOUNDATION-054, this file implements
// the synthetic standalone OSHE Inspect product composition, role-based navigation,
// bounded work queues, and explicit separation of feature entitlement from record authorization.
//
// Strict Product Composition Invariants:
// 1. Standalone Product Identity: OSHE Inspect is composed as a focused, self-contained
//    application surface with explicit manifest, role profiles, and operational work queues.
// 2. Default-Deny Role Navigation: Navigation items are filtered strictly to the viewer's
//    active authenticated role. Unauthenticated, unentitled, or unknown roles receive zero exposure.
// 3. Entitlement Never Substitutes for Record Authorization: Possessing an Inspect product
//    entitlement (feature license) is a prerequisite for module access, but NEVER grants
//    access to individual records or queues. Tenant, project, scope, and role permissions
//    must be independently satisfied on every record lookup.
// 4. Non-Leaking Empty and Denied States: Unauthorized or out-of-scope queries return generic
//    denials or sanitized empty views, preventing project or asset enumeration.
// 5. Accessibility Compliance: All composed navigation and action controls mandate non-empty
//    accessible labels (WCAG / a11y).
// 6. Zero External Enactment: Operates purely in-memory on synthetic local fixtures.
package portal

import (
	"errors"
	"fmt"
	"strings"
)

// Extended Inspect-specific portal roles.
const (
	RoleChecklistAuthor     PortalRole = "CHECKLIST_AUTHOR"
	RoleCAPAOwner           PortalRole = "CAPA_OWNER"
	RoleIndependentReviewer PortalRole = "INDEPENDENT_REVIEWER"
	RoleSupervisor          PortalRole = "SUPERVISOR"
)

func init() {
	KnownPortalRoles[RoleChecklistAuthor] = true
	KnownPortalRoles[RoleCAPAOwner] = true
	KnownPortalRoles[RoleIndependentReviewer] = true
	KnownPortalRoles[RoleSupervisor] = true
}

var (
	ErrEntitlementRequired = errors.New("viewer tenant lacks active product entitlement for OSHE Inspect")
	ErrRecordAccessDenied  = errors.New("access denied: record authorization failed (scope or role mismatch)")
	ErrBlankRecordID       = errors.New("target record ID cannot be blank")
)

// InspectProductManifest defines the explicit product identity, version, and capability envelope.
type InspectProductManifest struct {
	ProductID        string       `json:"product_id"`
	ProductName      string       `json:"product_name"`
	ProductVersion   string       `json:"product_version"`
	SupportedModes   []string     `json:"supported_modes"`
	SupportedRoles   []PortalRole `json:"supported_roles"`
	CoreCapabilities []string     `json:"core_capabilities"`
	NoticeWatermark  string       `json:"notice_watermark"`
}

// StandaloneInspectManifest returns the authoritative product manifest for OSHE Inspect.
func StandaloneInspectManifest() InspectProductManifest {
	return InspectProductManifest{
		ProductID:      "oshe-inspect",
		ProductName:    "OSHE Inspect",
		ProductVersion: "v0.4.0-alpha",
		SupportedModes: []string{
			"STANDALONE_WEB",
			"OFFLINE_RESPONSIVE",
		},
		SupportedRoles: []PortalRole{
			RoleInspector,
			RoleChecklistAuthor,
			RoleCAPAOwner,
			RoleIndependentReviewer,
			RoleSupervisor,
			RoleTenantAdmin,
			RoleProjectManager,
		},
		CoreCapabilities: []string{
			"CHECKLIST_EXECUTION",
			"OFFLINE_LOCAL_SYNC",
			"EVIDENCE_CAPTURE_PREVIEW",
			"FINDING_IDENTIFICATION",
			"CAPA_GOVERNANCE",
			"AUDIT_PRESERVATION",
		},
		NoticeWatermark: "SYNTHETIC_STANDALONE_INSPECT_ALPHA",
	}
}

// FeatureEntitlement models license or tenant-level feature grants.
// INVARIANT: Entitlement defines commercial/feature boundary, NOT authorization truth.
type FeatureEntitlement struct {
	TenantID        string `json:"tenant_id"`
	InspectEnabled  bool   `json:"inspect_enabled"`
	AdvancedReports bool   `json:"advanced_reports"`
	OfflineEnabled  bool   `json:"offline_enabled"`
}

// TargetRecord models an operational domain record being evaluated for user access.
type TargetRecord struct {
	RecordID        string              `json:"record_id"`
	TenantID        string              `json:"tenant_id"`
	ProjectID       string              `json:"project_id"`
	SiteID          string              `json:"site_id"`
	RecordType      string              `json:"record_type"` // e.g., "INSPECTION", "FINDING", "ACTION", "TEMPLATE"
	AllowedRoles    map[PortalRole]bool `json:"allowed_roles"`
	AssignedSubject string              `json:"assigned_subject,omitempty"`
}

// InspectWorkItem represents an actionable task in an Inspect work queue.
type InspectWorkItem struct {
	ItemID          string     `json:"item_id"`
	TenantID        string     `json:"tenant_id"`
	ProjectID       string     `json:"project_id"`
	SiteID          string     `json:"site_id"`
	ItemType        string     `json:"item_type"` // "INSPECTION_EXECUTION", "FINDING_REVIEW", "CAPA_ACTION", "TEMPLATE_REVIEW"
	Title           string     `json:"title"`
	TargetRole      PortalRole `json:"target_role"`
	AssignedSubject string     `json:"assigned_subject,omitempty"`
	Priority        string     `json:"priority"` // "HIGH", "MEDIUM", "LOW"
	DueStatus       string     `json:"due_status"`
}

// InspectWorkQueue aggregates role-filtered pending work items and empty-state guidance.
type InspectWorkQueue struct {
	TenantID       string            `json:"tenant_id"`
	ProjectID      string            `json:"project_id"`
	ViewerRole     PortalRole        `json:"viewer_role"`
	Items          []InspectWorkItem `json:"items"`
	TotalCount     int               `json:"total_count"`
	IsEmpty        bool              `json:"is_empty"`
	EmptyStateCode string            `json:"empty_state_code,omitempty"`
	EmptyMessage   string            `json:"empty_message,omitempty"`
}

// InspectNavigationCatalog defines the authoritative navigation catalog for OSHE Inspect.
var InspectNavigationCatalog = []NavigationItem{
	// Inspector Journey
	{
		ID:            "inspect-nav-assigned",
		Label:         "My Assigned Inspections",
		Path:          "/inspect/assigned",
		AriaLabel:     "View active field inspections assigned to your account",
		RequiredRoles: []PortalRole{RoleInspector},
		ScopeLevel:    "PROJECT",
	},
	{
		ID:            "inspect-nav-offline",
		Label:         "Offline Work Packages",
		Path:          "/inspect/offline",
		AriaLabel:     "Manage downloaded offline work packages and local synchronization queue",
		RequiredRoles: []PortalRole{RoleInspector},
		ScopeLevel:    "PROJECT",
	},

	// Checklist Author Journey
	{
		ID:            "inspect-nav-templates",
		Label:         "Checklist Templates",
		Path:          "/inspect/templates",
		AriaLabel:     "Author, edit, and manage checklist template versions",
		RequiredRoles: []PortalRole{RoleChecklistAuthor},
		ScopeLevel:    "PROJECT",
	},
	{
		ID:            "inspect-nav-questions",
		Label:         "Question Bank",
		Path:          "/inspect/questions",
		AriaLabel:     "Configure conditional logic, rules, and question definitions",
		RequiredRoles: []PortalRole{RoleChecklistAuthor},
		ScopeLevel:    "PROJECT",
	},

	// CAPA Owner Journey
	{
		ID:            "inspect-nav-capa",
		Label:         "Corrective Actions",
		Path:          "/inspect/actions",
		AriaLabel:     "View and remediate assigned safety corrective actions",
		RequiredRoles: []PortalRole{RoleCAPAOwner},
		ScopeLevel:    "PROJECT",
	},
	{
		ID:            "inspect-nav-evidence",
		Label:         "Evidence Submissions",
		Path:          "/inspect/evidence-submit",
		AriaLabel:     "Upload and track verification evidence for open actions",
		RequiredRoles: []PortalRole{RoleCAPAOwner},
		ScopeLevel:    "PROJECT",
	},

	// Independent Reviewer Journey
	{
		ID:            "inspect-nav-review-queue",
		Label:         "Review Queue",
		Path:          "/inspect/reviews",
		AriaLabel:     "Perform independent verification of completed inspections and findings",
		RequiredRoles: []PortalRole{RoleIndependentReviewer},
		ScopeLevel:    "PROJECT",
	},
	{
		ID:            "inspect-nav-evidence-verify",
		Label:         "Evidence Verification",
		Path:          "/inspect/evidence-verify",
		AriaLabel:     "Inspect, accept, or reject remedial evidence submissions",
		RequiredRoles: []PortalRole{RoleIndependentReviewer},
		ScopeLevel:    "PROJECT",
	},

	// Supervisor / Admin Journey
	{
		ID:            "inspect-nav-schedules",
		Label:         "Inspection Schedules",
		Path:          "/inspect/schedules",
		AriaLabel:     "Configure periodic recurrence rules and assignment dispatch",
		RequiredRoles: []PortalRole{RoleSupervisor},
		ScopeLevel:    "PROJECT",
	},
	{
		ID:            "inspect-nav-diagnostics",
		Label:         "Operational Diagnostics",
		Path:          "/inspect/diagnostics",
		AriaLabel:     "View operational health, sync conflicts, and schedule diagnostics",
		RequiredRoles: []PortalRole{RoleSupervisor},
		ScopeLevel:    "TENANT",
	},
}

// InspectCompositionEngine coordinates standalone Inspect portal resolution.
type InspectCompositionEngine struct{}

// NewInspectCompositionEngine constructs an InspectCompositionEngine.
func NewInspectCompositionEngine() *InspectCompositionEngine {
	return &InspectCompositionEngine{}
}

// ResolveNavigation filters the Inspect navigation catalog strictly to items permitted for the viewer's role.
// Enforces default-deny and requires active feature entitlement.
func (e *InspectCompositionEngine) ResolveNavigation(
	viewer PortalViewer,
	entitlement FeatureEntitlement,
) (NavigationMenu, error) {
	if err := viewer.Validate(); err != nil {
		return NavigationMenu{}, err
	}

	// 1. Entitlement Pre-Check: Tenant must have Inspect product enabled
	if !entitlement.InspectEnabled || entitlement.TenantID != viewer.TenantID {
		return NavigationMenu{}, ErrEntitlementRequired
	}

	// 2. Default-Deny Role Filter
	allowedItems := make([]NavigationItem, 0)
	for _, item := range InspectNavigationCatalog {
		if err := item.ValidateAccessibility(); err != nil {
			return NavigationMenu{}, fmt.Errorf("navigation item %s failed a11y: %w", item.ID, err)
		}

		isRoleAllowed := false
		for _, r := range item.RequiredRoles {
			if r == viewer.Role || (viewer.Role == RoleTenantAdmin && r == RoleSupervisor) {
				isRoleAllowed = true
				break
			}
		}

		if isRoleAllowed {
			allowedItems = append(allowedItems, item)
		}
	}

	return NavigationMenu{
		ViewerRole: viewer.Role,
		TenantID:   viewer.TenantID,
		ProjectID:  viewer.ProjectID,
		Items:      allowedItems,
	}, nil
}

// ResolveWorkQueue resolves pending operational items strictly matching viewer tenant, project, and role.
// Returns non-leaking empty states when zero items match.
func (e *InspectCompositionEngine) ResolveWorkQueue(
	viewer PortalViewer,
	entitlement FeatureEntitlement,
	allItems []InspectWorkItem,
) (InspectWorkQueue, error) {
	if err := viewer.Validate(); err != nil {
		return InspectWorkQueue{}, err
	}

	// Entitlement check
	if !entitlement.InspectEnabled || entitlement.TenantID != viewer.TenantID {
		return InspectWorkQueue{}, ErrEntitlementRequired
	}

	matched := make([]InspectWorkItem, 0)
	for _, item := range allItems {
		// Strict isolation: tenant and project must match
		if item.TenantID != viewer.TenantID || item.ProjectID != viewer.ProjectID {
			continue
		}

		// Role match
		if item.TargetRole != viewer.Role && viewer.Role != RoleTenantAdmin {
			continue
		}

		// If assigned to a specific user, verify identity
		if item.AssignedSubject != "" && item.AssignedSubject != viewer.Subject && viewer.Role != RoleTenantAdmin && viewer.Role != RoleSupervisor {
			continue
		}

		matched = append(matched, item)
	}

	if len(matched) == 0 {
		return InspectWorkQueue{
			TenantID:       viewer.TenantID,
			ProjectID:      viewer.ProjectID,
			ViewerRole:     viewer.Role,
			Items:          matched,
			TotalCount:     0,
			IsEmpty:        true,
			EmptyStateCode: "NO_ACTIVE_TASKS",
			EmptyMessage:   "No pending tasks in your work queue for this project.",
		}, nil
	}

	return InspectWorkQueue{
		TenantID:       viewer.TenantID,
		ProjectID:      viewer.ProjectID,
		ViewerRole:     viewer.Role,
		Items:          matched,
		TotalCount:     len(matched),
		IsEmpty:        false,
		EmptyStateCode: "",
		EmptyMessage:   "",
	}, nil
}

// EvaluateRecordAccess evaluates whether a viewer is authorized to access an individual domain record.
// CRITICAL INVARIANT: Entitlement never substitutes for record authorization.
func (e *InspectCompositionEngine) EvaluateRecordAccess(
	viewer PortalViewer,
	entitlement FeatureEntitlement,
	record TargetRecord,
) (bool, error) {
	if err := viewer.Validate(); err != nil {
		return false, err
	}

	if strings.TrimSpace(record.RecordID) == "" {
		return false, ErrBlankRecordID
	}

	// Step 1: Feature Entitlement Prerequisite
	// Lacking entitlement fails closed immediately
	if !entitlement.InspectEnabled || entitlement.TenantID != viewer.TenantID {
		return false, ErrEntitlementRequired
	}

	// Step 2: Record-Level Authorization Verification
	// Having an entitlement NEVER bypasses record-level scope and permission checks!

	// 2a. Tenant boundary isolation
	if record.TenantID != viewer.TenantID {
		return false, ErrRecordAccessDenied
	}

	// 2b. Project scope confinement
	if record.ProjectID != "" && record.ProjectID != viewer.ProjectID {
		return false, ErrRecordAccessDenied
	}

	// 2c. Role capability check
	if len(record.AllowedRoles) > 0 {
		if !record.AllowedRoles[viewer.Role] && viewer.Role != RoleTenantAdmin {
			return false, ErrRecordAccessDenied
		}
	}

	// 2d. Assigned subject check (if record is privatized to an assignee)
	if record.AssignedSubject != "" && record.AssignedSubject != viewer.Subject {
		if viewer.Role != RoleTenantAdmin && viewer.Role != RoleSupervisor && viewer.Role != RoleIndependentReviewer {
			return false, ErrRecordAccessDenied
		}
	}

	return true, nil
}
