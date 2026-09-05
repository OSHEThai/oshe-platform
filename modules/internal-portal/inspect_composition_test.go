package portal

import (
	"testing"
)

func setupTestContext() (PortalViewer, FeatureEntitlement) {
	viewer := PortalViewer{
		TenantID:        "ten_synthetic_alpha",
		ProjectID:       "prj_plant_safety_01",
		Subject:         "usr_syn_inspector_01",
		Role:            RoleInspector,
		IsAuthenticated: true,
	}

	entitlement := FeatureEntitlement{
		TenantID:        "ten_synthetic_alpha",
		InspectEnabled:  true,
		AdvancedReports: true,
		OfflineEnabled:  true,
	}

	return viewer, entitlement
}

func TestInspectManifest(t *testing.T) {
	manifest := StandaloneInspectManifest()

	if manifest.ProductID != "oshe-inspect" {
		t.Errorf("expected product ID oshe-inspect, got %s", manifest.ProductID)
	}
	if manifest.ProductName != "OSHE Inspect" {
		t.Errorf("expected product name OSHE Inspect, got %s", manifest.ProductName)
	}
	if manifest.ProductVersion != "v0.4.0-alpha" {
		t.Errorf("expected product version v0.4.0-alpha, got %s", manifest.ProductVersion)
	}
	if len(manifest.SupportedModes) < 2 {
		t.Errorf("expected at least 2 supported modes, got %d", len(manifest.SupportedModes))
	}
	if len(manifest.SupportedRoles) < 4 {
		t.Errorf("expected at least 4 supported roles, got %d", len(manifest.SupportedRoles))
	}
	if manifest.NoticeWatermark == "" {
		t.Errorf("expected non-empty notice watermark")
	}
}

func TestInspectNavigation_DefaultDenyAndRoleScoping(t *testing.T) {
	engine := NewInspectCompositionEngine()
	viewer, entitlement := setupTestContext()

	// 1. Unauthenticated viewer -> MUST FAIL
	unauthViewer := viewer
	unauthViewer.IsAuthenticated = false
	_, err := engine.ResolveNavigation(unauthViewer, entitlement)
	if err != ErrUnauthenticatedViewer {
		t.Fatalf("expected ErrUnauthenticatedViewer, got %v", err)
	}

	// 2. Unentitled tenant -> MUST FAIL
	unentitled := entitlement
	unentitled.InspectEnabled = false
	_, err = engine.ResolveNavigation(viewer, unentitled)
	if err != ErrEntitlementRequired {
		t.Fatalf("expected ErrEntitlementRequired, got %v", err)
	}

	// 3. Inspector Journey Navigation
	inspectorMenu, err := engine.ResolveNavigation(viewer, entitlement)
	if err != nil {
		t.Fatalf("failed to resolve inspector navigation: %v", err)
	}
	if len(inspectorMenu.Items) != 2 {
		t.Fatalf("expected exactly 2 inspector navigation items, got %d", len(inspectorMenu.Items))
	}
	for _, item := range inspectorMenu.Items {
		if !containsRole(item.RequiredRoles, RoleInspector) {
			t.Errorf("leaked non-inspector item %s to inspector", item.ID)
		}
	}

	// 4. Checklist Author Journey Navigation
	authorViewer := viewer
	authorViewer.Role = RoleChecklistAuthor
	authorViewer.Subject = "usr_syn_author_01"
	authorMenu, err := engine.ResolveNavigation(authorViewer, entitlement)
	if err != nil {
		t.Fatalf("failed to resolve author navigation: %v", err)
	}
	if len(authorMenu.Items) != 2 {
		t.Fatalf("expected exactly 2 author navigation items, got %d", len(authorMenu.Items))
	}
	for _, item := range authorMenu.Items {
		if !containsRole(item.RequiredRoles, RoleChecklistAuthor) {
			t.Errorf("leaked non-author item %s to author", item.ID)
		}
	}

	// 5. CAPA Owner Journey Navigation
	capaViewer := viewer
	capaViewer.Role = RoleCAPAOwner
	capaViewer.Subject = "usr_syn_capa_01"
	capaMenu, err := engine.ResolveNavigation(capaViewer, entitlement)
	if err != nil {
		t.Fatalf("failed to resolve CAPA navigation: %v", err)
	}
	if len(capaMenu.Items) != 2 {
		t.Fatalf("expected exactly 2 CAPA navigation items, got %d", len(capaMenu.Items))
	}
	for _, item := range capaMenu.Items {
		if !containsRole(item.RequiredRoles, RoleCAPAOwner) {
			t.Errorf("leaked non-CAPA item %s to CAPA owner", item.ID)
		}
	}

	// 6. Independent Reviewer Journey Navigation
	reviewerViewer := viewer
	reviewerViewer.Role = RoleIndependentReviewer
	reviewerViewer.Subject = "usr_syn_reviewer_01"
	reviewerMenu, err := engine.ResolveNavigation(reviewerViewer, entitlement)
	if err != nil {
		t.Fatalf("failed to resolve reviewer navigation: %v", err)
	}
	if len(reviewerMenu.Items) != 2 {
		t.Fatalf("expected exactly 2 reviewer navigation items, got %d", len(reviewerMenu.Items))
	}
	for _, item := range reviewerMenu.Items {
		if !containsRole(item.RequiredRoles, RoleIndependentReviewer) {
			t.Errorf("leaked non-reviewer item %s to reviewer", item.ID)
		}
	}

	// 7. Unknown Role -> Fails closed with ErrInvalidRole
	unknownViewer := viewer
	unknownViewer.Role = PortalRole("UNKNOWN_ROLE")
	_, err = engine.ResolveNavigation(unknownViewer, entitlement)
	if err != ErrInvalidRole {
		t.Fatalf("expected ErrInvalidRole for unknown role, got %v", err)
	}

	// 8. Contractor Role -> Allowed role but 0 items in Inspect catalog (Default-Deny)
	contractorViewer := viewer
	contractorViewer.Role = RoleContractor
	contractorMenu, err := engine.ResolveNavigation(contractorViewer, entitlement)
	if err != nil {
		t.Fatalf("failed to resolve contractor navigation: %v", err)
	}
	if len(contractorMenu.Items) != 0 {
		t.Errorf("expected 0 items for contractor role in Inspect, got %d", len(contractorMenu.Items))
	}
}

func TestInspectWorkQueue_ResolutionAndPartitioning(t *testing.T) {
	engine := NewInspectCompositionEngine()
	viewer, entitlement := setupTestContext()

	candidateItems := []InspectWorkItem{
		// 1. Matching inspector task
		{
			ItemID:          "item_01",
			TenantID:        "ten_synthetic_alpha",
			ProjectID:       "prj_plant_safety_01",
			SiteID:          "ste_rayong",
			ItemType:        "INSPECTION_EXECUTION",
			Title:           "Weekly Electrical Safety Walk",
			TargetRole:      RoleInspector,
			AssignedSubject: "usr_syn_inspector_01",
			Priority:        "HIGH",
			DueStatus:       "ON_TIME",
		},
		// 2. Mismatched tenant task
		{
			ItemID:     "item_02_other_tenant",
			TenantID:   "ten_foreign_beta",
			ProjectID:  "prj_plant_safety_01",
			TargetRole: RoleInspector,
		},
		// 3. Mismatched project task
		{
			ItemID:     "item_03_other_project",
			TenantID:   "ten_synthetic_alpha",
			ProjectID:  "prj_other_project",
			TargetRole: RoleInspector,
		},
		// 4. Mismatched role task (CAPA Action)
		{
			ItemID:     "item_04_capa_role",
			TenantID:   "ten_synthetic_alpha",
			ProjectID:  "prj_plant_safety_01",
			TargetRole: RoleCAPAOwner,
		},
		// 5. Assigned to a different inspector
		{
			ItemID:          "item_05_other_inspector",
			TenantID:        "ten_synthetic_alpha",
			ProjectID:       "prj_plant_safety_01",
			TargetRole:      RoleInspector,
			AssignedSubject: "usr_syn_inspector_02",
		},
	}

	queue, err := engine.ResolveWorkQueue(viewer, entitlement, candidateItems)
	if err != nil {
		t.Fatalf("work queue resolution failed: %v", err)
	}

	if queue.TotalCount != 1 {
		t.Fatalf("expected exactly 1 matching item, got %d", queue.TotalCount)
	}
	if queue.Items[0].ItemID != "item_01" {
		t.Errorf("unexpected item in queue: %s", queue.Items[0].ItemID)
	}
	if queue.IsEmpty {
		t.Errorf("queue should not be empty")
	}

	// Empty Queue Scenario -> Sanitized Non-Leaking Response
	emptyViewer := viewer
	emptyViewer.ProjectID = "prj_empty_project"
	emptyQueue, err := engine.ResolveWorkQueue(emptyViewer, entitlement, candidateItems)
	if err != nil {
		t.Fatalf("empty queue resolution failed: %v", err)
	}

	if !emptyQueue.IsEmpty || emptyQueue.TotalCount != 0 {
		t.Errorf("expected empty queue, got %d items", emptyQueue.TotalCount)
	}
	if emptyQueue.EmptyStateCode != "NO_ACTIVE_TASKS" {
		t.Errorf("expected empty state code NO_ACTIVE_TASKS, got %s", emptyQueue.EmptyStateCode)
	}
}

func TestEntitlementSeparatedFromRecordAuthorization(t *testing.T) {
	engine := NewInspectCompositionEngine()
	viewer, entitlement := setupTestContext()

	baseRecord := TargetRecord{
		RecordID:   "ins_syn_clean_01",
		TenantID:   "ten_synthetic_alpha",
		ProjectID:  "prj_plant_safety_01",
		SiteID:     "ste_rayong",
		RecordType: "INSPECTION",
		AllowedRoles: map[PortalRole]bool{
			RoleInspector:          true,
			RoleSupervisor:         true,
			RoleIndependentReviewer: true,
		},
		AssignedSubject: "usr_syn_inspector_01",
	}

	// Case 1: Fully Authorized (Entitled + Matching Tenant + Matching Project + Matching Role + Matching Assignee)
	allowed, err := engine.EvaluateRecordAccess(viewer, entitlement, baseRecord)
	if err != nil || !allowed {
		t.Fatalf("expected access to be allowed, got allowed=%v, err=%v", allowed, err)
	}

	// Case 2: INVARIANT PROOF - Entitled user, but CROSS-TENANT record
	// Proves entitlement NEVER grants cross-tenant access
	crossTenantRecord := baseRecord
	crossTenantRecord.TenantID = "ten_foreign_beta"
	allowed, err = engine.EvaluateRecordAccess(viewer, entitlement, crossTenantRecord)
	if allowed || err != ErrRecordAccessDenied {
		t.Fatalf("security violation: entitlement granted access to cross-tenant record! allowed=%v, err=%v", allowed, err)
	}

	// Case 3: INVARIANT PROOF - Entitled user, but CROSS-PROJECT record
	// Proves entitlement NEVER grants cross-project access
	crossProjectRecord := baseRecord
	crossProjectRecord.ProjectID = "prj_unauthorized_expansion"
	allowed, err = engine.EvaluateRecordAccess(viewer, entitlement, crossProjectRecord)
	if allowed || err != ErrRecordAccessDenied {
		t.Fatalf("security violation: entitlement granted access to cross-project record! allowed=%v, err=%v", allowed, err)
	}

	// Case 4: INVARIANT PROOF - Entitled user, but ROLE UNAUTHORIZED on record
	// Proves entitlement NEVER bypasses role authorization boundaries
	unauthorizedRoleRecord := baseRecord
	unauthorizedRoleRecord.AllowedRoles = map[PortalRole]bool{
		RoleCAPAOwner: true, // Only CAPA Owner can access this record
	}
	allowed, err = engine.EvaluateRecordAccess(viewer, entitlement, unauthorizedRoleRecord)
	if allowed || err != ErrRecordAccessDenied {
		t.Fatalf("security violation: entitlement bypassed role permission check! allowed=%v, err=%v", allowed, err)
	}

	// Case 5: INVARIANT PROOF - Entitled user, but RECORD PRIVATIZED to another assignee
	otherAssigneeRecord := baseRecord
	otherAssigneeRecord.AssignedSubject = "usr_syn_inspector_02" // different inspector
	allowed, err = engine.EvaluateRecordAccess(viewer, entitlement, otherAssigneeRecord)
	if allowed || err != ErrRecordAccessDenied {
		t.Fatalf("security violation: entitlement bypassed assignee custody! allowed=%v, err=%v", allowed, err)
	}

	// Case 6: Lacking Feature Entitlement -> Fails closed before record evaluation
	unentitled := entitlement
	unentitled.InspectEnabled = false
	allowed, err = engine.EvaluateRecordAccess(viewer, unentitled, baseRecord)
	if allowed || err != ErrEntitlementRequired {
		t.Fatalf("expected ErrEntitlementRequired when feature is disabled, got allowed=%v, err=%v", allowed, err)
	}
}

func TestAccessibility_NavigationCompliance(t *testing.T) {
	for _, item := range InspectNavigationCatalog {
		if err := item.ValidateAccessibility(); err != nil {
			t.Errorf("navigation item %s failed accessibility: %v", item.ID, err)
		}
		if item.Path == "" || item.Label == "" || item.AriaLabel == "" {
			t.Errorf("navigation item %s has blank fields: %+v", item.ID, item)
		}
	}
}

func containsRole(roles []PortalRole, target PortalRole) bool {
	for _, r := range roles {
		if r == target {
			return true
		}
	}
	return false
}
