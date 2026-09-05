package portal_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	portal "oshe/internal-portal"
)

// NEG-PORTAL-01: Wrong-Role Navigation Containment
// Threat: Lower-privilege roles (Contractor, Inspector, Auditor) discovering or executing administrative navigation.
func TestNegativeControl_NavigationRoleContainment(t *testing.T) {
	tenantID := "ten_alpha"
	projectID := "prj_01"

	// 1. Contractor cannot see admin or project management links
	contractorViewer := portal.PortalViewer{
		Subject:         "usr_synth_contractor",
		TenantID:        tenantID,
		ProjectID:       projectID,
		Role:            portal.RoleContractor,
		IsAuthenticated: true,
	}
	menu, err := portal.ResolveNavigation(contractorViewer)
	if err != nil {
		t.Fatalf("unexpected ResolveNavigation error: %v", err)
	}

	for _, item := range menu.Items {
		if item.ID == "nav_tenant_admin" || item.ID == "nav_project_queue" || item.ID == "nav_audit_exports" {
			t.Fatalf("navigation containment breach: contractor received administrative nav item %s", item.ID)
		}
		if strings.Contains(strings.ToLower(item.Label), "admin") {
			t.Fatalf("navigation containment breach: contractor received admin label: %s", item.Label)
		}
	}

	// 2. Inspector cannot see tenant admin or auditor export links
	inspectorViewer := portal.PortalViewer{
		Subject:         "usr_synth_inspector",
		TenantID:        tenantID,
		ProjectID:       projectID,
		Role:            portal.RoleInspector,
		IsAuthenticated: true,
	}
	inspMenu, _ := portal.ResolveNavigation(inspectorViewer)
	for _, item := range inspMenu.Items {
		if item.ID == "nav_tenant_admin" || item.ID == "nav_audit_exports" {
			t.Fatalf("navigation containment breach: inspector received %s", item.ID)
		}
	}
}

// NEG-PORTAL-02: Cross-Project Work Queue Isolation & Anti-Enumeration
// Threat: Worker on Project Alpha querying or discovering pending tasks from Project Beta.
func TestNegativeControl_CrossProjectWorkQueueLeakage(t *testing.T) {
	tenantID := "ten_alpha"
	now := time.Now().UTC()

	items := []portal.WorkQueueItem{
		{
			QueueID:      "q_victim_beta",
			TenantID:     tenantID,
			ProjectID:    "prj_beta_secret",
			TaskType:     "CRITICAL_HAZARD_REPAIR",
			Title:        "Confidential Hazard Repair in Project Beta",
			AriaLabel:    "Fix high-pressure valve",
			AssignedRole: portal.RoleInspector,
			Priority:     "CRITICAL",
			CreatedAt:    now,
		},
	}

	// Attacker on Project Alpha queries work queue
	attacker := portal.PortalViewer{
		Subject:         "usr_synth_attacker",
		TenantID:        tenantID,
		ProjectID:       "prj_alpha",
		Role:            portal.RoleInspector,
		IsAuthenticated: true,
	}

	queue, err := portal.ResolveWorkQueue(attacker, items)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Queue must be empty and must NOT leak the existence of Project Beta tasks
	if queue.TotalCount != 0 || len(queue.Items) != 0 {
		t.Fatalf("anti-enumeration breach: attacker on prj_alpha received %d tasks from prj_beta", queue.TotalCount)
	}
	if !strings.Contains(queue.EmptyStateNotice, "No work items") {
		t.Errorf("expected clean non-leaking empty notice, got %q", queue.EmptyStateNotice)
	}
}

// NEG-PORTAL-03: Cross-Tenant Work Queue Isolation
func TestNegativeControl_CrossTenantQueueIsolation(t *testing.T) {
	now := time.Now().UTC()
	items := []portal.WorkQueueItem{
		{
			QueueID:      "q_foreign_01",
			TenantID:     "ten_foreign",
			ProjectID:    "prj_alpha",
			TaskType:     "INSPECTION",
			Title:        "Foreign Task",
			AriaLabel:    "Foreign task",
			AssignedRole: portal.RoleInspector,
			Priority:     "HIGH",
			CreatedAt:    now,
		},
	}

	viewer := portal.PortalViewer{
		Subject:         "usr_synth_01",
		TenantID:        "ten_alpha",
		ProjectID:       "prj_alpha",
		Role:            portal.RoleInspector,
		IsAuthenticated: true,
	}

	queue, err := portal.ResolveWorkQueue(viewer, items)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if queue.TotalCount != 0 || len(queue.Items) != 0 {
		t.Fatalf("cross-tenant leakage: viewer in ten_alpha received tasks from ten_foreign")
	}
}

// NEG-PORTAL-04: Non-Leaking Content Access Denial
// Threat: IDOR / lateral probing of content IDs confirming existence in sibling projects or foreign tenants.
func TestNegativeControl_NonLeakingContentDenial(t *testing.T) {
	content := portal.TargetContent{
		ContentID:     "cnt_confidential_01",
		TenantID:      "ten_alpha",
		ProjectID:     "prj_beta",
		RequiredRoles: []portal.PortalRole{portal.RoleProjectManager},
		Title:         "Confidential Safety Incident Report",
		Body:          "Sensitive incident investigation data",
	}

	// 1. Viewer on different project (prj_alpha) attempting to probe prj_beta content
	viewer := portal.PortalViewer{
		Subject:         "usr_synth_pm_alpha",
		TenantID:        "ten_alpha",
		ProjectID:       "prj_alpha",
		Role:            portal.RoleProjectManager,
		IsAuthenticated: true,
	}

	_, err := portal.ViewContent(viewer, content)
	if !errors.Is(err, portal.ErrAccessDenied) {
		t.Fatalf("expected non-leaking ErrAccessDenied, got %v", err)
	}

	// 2. Unauthenticated viewer
	unauthViewer := viewer
	unauthViewer.IsAuthenticated = false
	_, err = portal.ViewContent(unauthViewer, content)
	if !errors.Is(err, portal.ErrUnauthenticatedViewer) {
		t.Fatalf("expected ErrUnauthenticatedViewer, got %v", err)
	}
}

// NEG-PORTAL-05: Accessibility Label Validation
// Threat: Inaccessible UI navigation links lacking text labels or ARIA labels.
func TestNegativeControl_AccessibilityLabelEnforcement(t *testing.T) {
	// 1. Empty Label rejected
	itemNoLabel := portal.NavigationItem{
		ID:            "nav_bad",
		Label:         "",
		AriaLabel:     "Some aria label",
		Path:          "/portal/bad",
		RequiredRoles: []portal.PortalRole{portal.RoleViewer},
	}
	if !errors.Is(itemNoLabel.ValidateAccessibility(), portal.ErrEmptyLabel) {
		t.Errorf("expected ErrEmptyLabel for blank label, got %v", itemNoLabel.ValidateAccessibility())
	}

	// 2. Empty AriaLabel rejected
	itemNoAria := portal.NavigationItem{
		ID:            "nav_bad",
		Label:         "Good Label",
		AriaLabel:     "   ",
		Path:          "/portal/bad",
		RequiredRoles: []portal.PortalRole{portal.RoleViewer},
	}
	if !errors.Is(itemNoAria.ValidateAccessibility(), portal.ErrEmptyAriaLabel) {
		t.Errorf("expected ErrEmptyAriaLabel for whitespace aria-label, got %v", itemNoAria.ValidateAccessibility())
	}
}
