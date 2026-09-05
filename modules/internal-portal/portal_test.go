package portal

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestPortalViewer_Validation(t *testing.T) {
	// 1. Valid viewer
	v := PortalViewer{
		Subject:         "usr_synth_pm_01",
		TenantID:        "ten_alpha",
		CompanyID:       "cmp_main",
		ProjectID:       "prj_alpha",
		Role:            RoleProjectManager,
		IsAuthenticated: true,
	}
	if err := v.Validate(); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}

	// 2. Unauthenticated viewer
	unauth := v
	unauth.IsAuthenticated = false
	if !errors.Is(unauth.Validate(), ErrUnauthenticatedViewer) {
		t.Errorf("expected ErrUnauthenticatedViewer, got %v", unauth.Validate())
	}

	// 3. Blank subject
	noSub := v
	noSub.Subject = ""
	if !errors.Is(noSub.Validate(), ErrBlankSubject) {
		t.Errorf("expected ErrBlankSubject, got %v", noSub.Validate())
	}

	// 4. Blank tenant
	noTenant := v
	noTenant.TenantID = ""
	if !errors.Is(noTenant.Validate(), ErrBlankTenantID) {
		t.Errorf("expected ErrBlankTenantID, got %v", noTenant.Validate())
	}

	// 5. Invalid role
	badRole := v
	badRole.Role = "SUPER_ADMIN"
	if !errors.Is(badRole.Validate(), ErrInvalidRole) {
		t.Errorf("expected ErrInvalidRole, got %v", badRole.Validate())
	}
}

func TestResolveNavigation_RoleContainment(t *testing.T) {
	tenantID := "ten_alpha"
	projectID := "prj_01"

	roles := []struct {
		role          PortalRole
		expectedItems []string
		forbiddenIDs  []string
	}{
		{
			role:          RoleTenantAdmin,
			expectedItems: []string{"nav_tenant_admin", "nav_audit_exports", "nav_project_queue"},
			forbiddenIDs:  []string{},
		},
		{
			role:          RoleProjectManager,
			expectedItems: []string{"nav_project_queue", "nav_contractor_mgmt", "nav_field_inspections", "nav_my_tasks", "nav_read_reports"},
			forbiddenIDs:  []string{"nav_tenant_admin", "nav_audit_exports", "nav_contractor_remediation"},
		},
		{
			role:          RoleInspector,
			expectedItems: []string{"nav_field_inspections", "nav_my_tasks", "nav_read_reports"},
			forbiddenIDs:  []string{"nav_tenant_admin", "nav_audit_exports", "nav_project_queue", "nav_contractor_mgmt", "nav_contractor_remediation"},
		},
		{
			role:          RoleAuditor,
			expectedItems: []string{"nav_audit_exports", "nav_read_reports"},
			forbiddenIDs:  []string{"nav_tenant_admin", "nav_project_queue", "nav_contractor_mgmt", "nav_field_inspections", "nav_my_tasks"},
		},
		{
			role:          RoleContractor,
			expectedItems: []string{"nav_my_tasks", "nav_contractor_remediation"},
			forbiddenIDs:  []string{"nav_tenant_admin", "nav_audit_exports", "nav_project_queue", "nav_contractor_mgmt", "nav_field_inspections", "nav_read_reports"},
		},
		{
			role:          RoleViewer,
			expectedItems: []string{"nav_read_reports"},
			forbiddenIDs:  []string{"nav_tenant_admin", "nav_audit_exports", "nav_project_queue", "nav_contractor_mgmt", "nav_field_inspections", "nav_my_tasks", "nav_contractor_remediation"},
		},
	}

	for _, tc := range roles {
		t.Run(string(tc.role), func(t *testing.T) {
			viewer := PortalViewer{
				Subject:         "usr_synth_test",
				TenantID:        tenantID,
				ProjectID:       projectID,
				Role:            tc.role,
				IsAuthenticated: true,
			}

			menu, err := ResolveNavigation(viewer)
			if err != nil {
				t.Fatalf("ResolveNavigation error: %v", err)
			}

			presentIDs := make(map[string]bool)
			for _, item := range menu.Items {
				presentIDs[item.ID] = true
			}

			for _, exp := range tc.expectedItems {
				if !presentIDs[exp] {
					t.Errorf("role %s missing expected nav item %s", tc.role, exp)
				}
			}

			for _, forb := range tc.forbiddenIDs {
				if presentIDs[forb] {
					t.Errorf("role %s contains forbidden nav item %s (navigation containment leak)", tc.role, forb)
				}
			}
		})
	}
}

func TestResolveNavigation_AccessibilityCompliance(t *testing.T) {
	viewer := PortalViewer{
		Subject:         "usr_synth_pm",
		TenantID:        "ten_alpha",
		ProjectID:       "prj_01",
		Role:            RoleProjectManager,
		IsAuthenticated: true,
	}

	menu, err := ResolveNavigation(viewer)
	if err != nil {
		t.Fatalf("unexpected ResolveNavigation error: %v", err)
	}

	for _, item := range menu.Items {
		if strings.TrimSpace(item.Label) == "" {
			t.Errorf("item %s has blank Label", item.ID)
		}
		if strings.TrimSpace(item.AriaLabel) == "" {
			t.Errorf("item %s has blank AriaLabel", item.ID)
		}
		if !strings.HasPrefix(item.Path, "/portal/") {
			t.Errorf("item %s has invalid path: %s", item.ID, item.Path)
		}
	}
}

func TestResolveWorkQueue_ScopeAndRoleFiltering(t *testing.T) {
	tenantID := "ten_alpha"
	projAlpha := "prj_alpha"
	projBeta := "prj_beta"
	now := time.Now().UTC()

	items := []WorkQueueItem{
		{
			QueueID:      "q_01",
			TenantID:     tenantID,
			ProjectID:    projAlpha,
			TaskType:     "INSPECTION_APPROVAL",
			Title:        "Approve Scaffold Inspection",
			AriaLabel:    "Approve scaffold inspection for Project Alpha",
			AssignedRole: RoleProjectManager,
			Priority:     "HIGH",
			CreatedAt:    now,
		},
		{
			QueueID:      "q_02",
			TenantID:     tenantID,
			ProjectID:    projAlpha,
			TaskType:     "FIELD_INSPECTION",
			Title:        "Inspect Electrical Room B",
			AriaLabel:    "Execute electrical room safety inspection",
			AssignedRole: RoleInspector,
			Priority:     "MEDIUM",
			CreatedAt:    now,
		},
		{
			QueueID:      "q_03",
			TenantID:     tenantID,
			ProjectID:    projAlpha,
			TaskType:     "FINDING_REMEDIATION",
			Title:        "Repair Exposed Cabling",
			AriaLabel:    "Remediate exposed cabling hazard in Area 2",
			AssignedRole: RoleContractor,
			Priority:     "CRITICAL",
			CreatedAt:    now,
		},
		{
			// Item in sibling project Project Beta
			QueueID:      "q_04",
			TenantID:     tenantID,
			ProjectID:    projBeta,
			TaskType:     "INSPECTION_APPROVAL",
			Title:        "Beta Project Approval",
			AriaLabel:    "Approve inspection in Project Beta",
			AssignedRole: RoleProjectManager,
			Priority:     "LOW",
			CreatedAt:    now,
		},
		{
			// Item in foreign tenant
			QueueID:      "q_05",
			TenantID:     "ten_foreign",
			ProjectID:    projAlpha,
			TaskType:     "FIELD_INSPECTION",
			Title:        "Foreign Inspection",
			AriaLabel:    "Foreign tenant task",
			AssignedRole: RoleInspector,
			Priority:     "LOW",
			CreatedAt:    now,
		},
	}

	// 1. PM on Project Alpha sees q_01 only
	viewerPM := PortalViewer{Subject: "usr_pm", TenantID: tenantID, ProjectID: projAlpha, Role: RoleProjectManager, IsAuthenticated: true}
	queuePM, err := ResolveWorkQueue(viewerPM, items)
	if err != nil || queuePM.TotalCount != 1 || queuePM.Items[0].QueueID != "q_01" {
		t.Errorf("PM queue mismatch: count=%d, items=%+v (err: %v)", queuePM.TotalCount, queuePM.Items, err)
	}

	// 2. Inspector on Project Alpha sees q_02 only
	viewerInsp := PortalViewer{Subject: "usr_insp", TenantID: tenantID, ProjectID: projAlpha, Role: RoleInspector, IsAuthenticated: true}
	queueInsp, err := ResolveWorkQueue(viewerInsp, items)
	if err != nil || queueInsp.TotalCount != 1 || queueInsp.Items[0].QueueID != "q_02" {
		t.Errorf("Inspector queue mismatch: count=%d, items=%+v", queueInsp.TotalCount, queueInsp.Items)
	}

	// 3. Contractor on Project Alpha sees q_03 only
	viewerCon := PortalViewer{Subject: "usr_con", TenantID: tenantID, ProjectID: projAlpha, Role: RoleContractor, IsAuthenticated: true}
	queueCon, err := ResolveWorkQueue(viewerCon, items)
	if err != nil || queueCon.TotalCount != 1 || queueCon.Items[0].QueueID != "q_03" {
		t.Errorf("Contractor queue mismatch: count=%d, items=%+v", queueCon.TotalCount, queueCon.Items)
	}
}

func TestResolveWorkQueue_SafeEmptyState(t *testing.T) {
	viewer := PortalViewer{
		Subject:         "usr_pm",
		TenantID:        "ten_alpha",
		ProjectID:       "prj_empty_project",
		Role:            RoleProjectManager,
		IsAuthenticated: true,
	}

	queue, err := ResolveWorkQueue(viewer, []WorkQueueItem{})
	if err != nil {
		t.Fatalf("unexpected error on empty queue: %v", err)
	}
	if queue.TotalCount != 0 || len(queue.Items) != 0 {
		t.Errorf("expected empty items list, got %d", queue.TotalCount)
	}
	if !strings.Contains(queue.EmptyStateNotice, "No work items") {
		t.Errorf("expected helpful empty state notice, got %q", queue.EmptyStateNotice)
	}
}

func TestViewContent_AuthorizedAndScopeMatch(t *testing.T) {
	tenantID := "ten_alpha"
	projectID := "prj_alpha"

	content := TargetContent{
		ContentID:     "cnt_report_01",
		TenantID:      tenantID,
		ProjectID:     projectID,
		SiteID:        "ste_01",
		RequiredRoles: []PortalRole{RoleProjectManager, RoleInspector},
		Title:         "Monthly Site Safety Audit Summary",
		Body:          "All electrical standards met. Zero high findings.",
	}

	// 1. Authorized Inspector on same project
	viewerInsp := PortalViewer{
		Subject:         "usr_insp",
		TenantID:        tenantID,
		ProjectID:       projectID,
		SiteID:          "ste_01",
		Role:            RoleInspector,
		IsAuthenticated: true,
	}

	view, err := ViewContent(viewerInsp, content)
	if err != nil {
		t.Fatalf("unexpected ViewContent error: %v", err)
	}
	if view.ContentID != "cnt_report_01" || view.Title != content.Title {
		t.Errorf("content mismatch: %+v", view)
	}
	if view.Scope != "ten_alpha/prj_alpha/ste_01" {
		t.Errorf("scope mismatch: %s", view.Scope)
	}

	// 2. Unauthorized role (Contractor) on same project -> ErrAccessDenied
	viewerCon := PortalViewer{
		Subject:         "usr_con",
		TenantID:        tenantID,
		ProjectID:       projectID,
		Role:            RoleContractor,
		IsAuthenticated: true,
	}
	_, err = ViewContent(viewerCon, content)
	if !errors.Is(err, ErrAccessDenied) {
		t.Errorf("expected ErrAccessDenied for unauthorized role, got %v", err)
	}

	// 3. Cross-project query (PM on Project Beta requesting Project Alpha content) -> ErrAccessDenied
	viewerPMBeta := PortalViewer{
		Subject:         "usr_pm_beta",
		TenantID:        tenantID,
		ProjectID:       "prj_beta",
		Role:            RoleProjectManager,
		IsAuthenticated: true,
	}
	_, err = ViewContent(viewerPMBeta, content)
	if !errors.Is(err, ErrAccessDenied) {
		t.Errorf("expected ErrAccessDenied for cross-project access, got %v", err)
	}

	// 4. Cross-tenant query -> ErrAccessDenied
	viewerOtherTenant := PortalViewer{
		Subject:         "usr_other",
		TenantID:        "ten_other",
		ProjectID:       projectID,
		Role:            RoleInspector,
		IsAuthenticated: true,
	}
	_, err = ViewContent(viewerOtherTenant, content)
	if !errors.Is(err, ErrAccessDenied) {
		t.Errorf("expected ErrAccessDenied for cross-tenant access, got %v", err)
	}
}
