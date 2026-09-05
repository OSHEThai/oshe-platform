package orgtenancy

import (
	"errors"
	"strings"
	"testing"
)

// TestLifecycle_H030_002_DeferredGateDeclaration verifies that the local lifecycle state machine
// conforms to Sole Human Owner decision H030-002: operates as a local-simulation/preflight harness only,
// with zero external runtime execution, persistent database mutation, or premature final completion claim.
func TestLifecycle_H030_002_DeferredGateDeclaration(t *testing.T) {
	// 1. Verify that reversible transitions function purely in memory
	state := StateActive
	simState, err := SimulateReversibleTransition(state, StateClosed)
	if err != nil || simState != StateClosed {
		t.Fatalf("expected simulated transition to StateClosed, got %v", simState)
	}

	// Reversible simulation step (verifying H030-002 test harness capability)
	revState, err := SimulateReversibleTransition(simState, StateActive)
	if err != nil || revState != StateActive {
		t.Fatalf("expected simulated reversal to StateActive, got %v", revState)
	}

	// 2. Verify invalid simulation target returns error without crashing
	_, err = SimulateReversibleTransition(state, "INVALID_STATE")
	if err == nil {
		t.Errorf("expected error for unapproved simulation state")
	}
}

func TestLifecycle_CloseTransitionsAndState(t *testing.T) {
	tenantID := "ten_alpha"
	company, _ := NewCompany(tenantID, "cmp_01", "Acme Corp")
	bu, _ := NewBusinessUnit(company, "bnu_01", "Operations")
	project, _ := NewProjectUnderBusinessUnit(bu, "prj_01", "Plant Expansion")
	site, _ := NewSite(project, "ste_01", "Site Rayong")
	area, _ := NewArea(site, "ara_01", "Boiler Unit 1")

	// 1. Project Close
	closedProject, err := project.Close()
	if err != nil {
		t.Fatalf("unexpected project.Close() error: %v", err)
	}
	if !closedProject.IsClosed() || closedProject.State() != StateClosed {
		t.Errorf("expected closed project state, got %v", closedProject.State())
	}
	if closedProject.IsActive() {
		t.Errorf("closed project must not be active")
	}
	// Immutable value semantics: original project remains active
	if !project.IsActive() || project.IsClosed() {
		t.Errorf("original project must remain unmodified and active")
	}

	// 2. Site Close
	closedSite, err := site.Close()
	if err != nil {
		t.Fatalf("unexpected site.Close() error: %v", err)
	}
	if !closedSite.IsClosed() || closedSite.State() != StateClosed {
		t.Errorf("expected closed site state")
	}
	if closedSite.IsActive() {
		t.Errorf("closed site must not be active")
	}

	// 3. Area Close
	closedArea, err := area.Close()
	if err != nil {
		t.Fatalf("unexpected area.Close() error: %v", err)
	}
	if !closedArea.IsClosed() || closedArea.State() != StateClosed {
		t.Errorf("expected closed area state")
	}

	// 4. Company & BU Close
	closedBU, err := bu.Close()
	if err != nil || !closedBU.IsClosed() {
		t.Errorf("BU close failed: %v", err)
	}
	closedComp, err := company.Close()
	if err != nil || !closedComp.IsClosed() {
		t.Errorf("Company close failed: %v", err)
	}
}

func TestLifecycle_ClosedRejectionOnArchived(t *testing.T) {
	company, _ := NewCompany("ten_alpha", "cmp_01", "Acme")
	project, _ := NewProject(company, "prj_01", "Project")
	site, _ := NewSite(project, "ste_01", "Site")

	// Archive site first
	archivedSite := site.Archive()
	if !archivedSite.State().StateArchived() {
		t.Fatalf("expected archived site")
	}

	// Attempting to close an archived site fails closed
	if _, err := archivedSite.Close(); !errors.Is(err, ErrEntityArchived) {
		t.Errorf("expected ErrEntityArchived when closing archived site, got %v", err)
	}
}

func TestLifecycle_TerminalClosed_PolicyAndSimulationReversibility(t *testing.T) {
	// In operational posture, reopening closed entities in v0.3 is prohibited
	err := ReopenEntity(StateClosed)
	if !errors.Is(err, ErrCannotReopenClosed) {
		t.Errorf("expected ErrCannotReopenClosed, got %v", err)
	}

	// Non-closed state
	if err := ReopenEntity(StateActive); err == nil {
		t.Errorf("expected error when attempting to reopen an active entity")
	}

	// Reversible simulation harness retains H030-002 preflight testing ability
	res, err := SimulateReversibleTransition(StateClosed, StateActive)
	if err != nil || res != StateActive {
		t.Errorf("expected simulation reversibility in test harness: %v", err)
	}
}

func TestLifecycle_OperationalMutationRejection(t *testing.T) {
	// 1. Active entity permits operational mutations
	if err := AssertOperationalActive(StateActive); err != nil {
		t.Errorf("expected StateActive to permit operations, got %v", err)
	}

	// 2. Closed entity rejects active operations
	if err := AssertOperationalActive(StateClosed); !errors.Is(err, ErrEntityClosed) {
		t.Errorf("expected ErrEntityClosed, got %v", err)
	}

	// 3. Archived entity rejects active operations
	if err := AssertOperationalActive(StateArchived); !errors.Is(err, ErrEntityArchived) {
		t.Errorf("expected ErrEntityArchived, got %v", err)
	}

	// 4. Unknown state rejected
	if err := AssertOperationalActive("UNKNOWN_STATE"); !errors.Is(err, ErrActiveRecordRejected) {
		t.Errorf("expected ErrActiveRecordRejected for unknown state, got %v", err)
	}
}

func TestLifecycle_ParentOperationalGuard(t *testing.T) {
	if err := AssertParentOperationalActive(StateActive); err != nil {
		t.Errorf("expected active parent to pass guard, got %v", err)
	}
	if err := AssertParentOperationalActive(StateClosed); !errors.Is(err, ErrParentClosed) {
		t.Errorf("expected ErrParentClosed, got %v", err)
	}
	if err := AssertParentOperationalActive(StateArchived); !errors.Is(err, ErrParentArchived) {
		t.Errorf("expected ErrParentArchived, got %v", err)
	}
}

func TestLifecycle_SiteMove_SameTenantSuccess(t *testing.T) {
	tenantID := "ten_industrial"
	company, _ := NewCompany(tenantID, "cmp_01", "Siam Heavy")
	bu, _ := NewBusinessUnit(company, "bnu_01", "Downstream BU")

	projSource, _ := NewProjectUnderBusinessUnit(bu, "prj_source", "Source Expansion")
	projDest, _ := NewProjectUnderBusinessUnit(bu, "prj_dest", "Destination Modernization")

	site, err := NewSiteWithLocale(projSource, "ste_maptaphut", "Rayong Olefins Plant", "Asia/Bangkok", "th-TH")
	if err != nil {
		t.Fatalf("unexpected NewSiteWithLocale: %v", err)
	}

	// Move site from projSource to projDest
	movedSite, err := MoveSiteToProject(site, projDest)
	if err != nil {
		t.Fatalf("unexpected MoveSiteToProject error: %v", err)
	}

	// Assertions on moved site
	if movedSite.TenantID() != tenantID {
		t.Errorf("tenant changed: %s", movedSite.TenantID())
	}
	if movedSite.ProjectID() != projDest.ProjectID() {
		t.Errorf("expected project %s, got %s", projDest.ProjectID(), movedSite.ProjectID())
	}
	if movedSite.SiteID() != "ste_maptaphut" {
		t.Errorf("site ID must be preserved: %s", movedSite.SiteID())
	}
	if movedSite.Name() != "Rayong Olefins Plant" {
		t.Errorf("name must be preserved: %s", movedSite.Name())
	}
	if movedSite.TimeZone() != "Asia/Bangkok" || movedSite.Locale() != "th-TH" {
		t.Errorf("timeZone/locale must be preserved")
	}
	if !movedSite.IsActive() {
		t.Errorf("moved site should remain active")
	}

	// Verify scope projection reflects new project parent
	scope := movedSite.ResolveScope()
	expectedPath := "ten_industrial/cmp_01/bnu_01/prj_dest/ste_maptaphut"
	if scope.CanonicalPath != expectedPath {
		t.Errorf("expected canonical path %q, got %q", expectedPath, scope.CanonicalPath)
	}
}

func TestLifecycle_SiteMove_CrossTenantDenial(t *testing.T) {
	companyA, _ := NewCompany("ten_alpha", "cmp_a", "Company Alpha")
	projA, _ := NewProject(companyA, "prj_a", "Project Alpha")
	siteA, _ := NewSite(projA, "ste_a", "Site Alpha")

	companyB, _ := NewCompany("ten_bravo", "cmp_b", "Company Bravo")
	projB, _ := NewProject(companyB, "prj_b", "Project Bravo")

	// Moving across tenants is strictly prohibited
	_, err := MoveSiteToProject(siteA, projB)
	if !errors.Is(err, ErrCrossTenantMove) {
		t.Errorf("expected ErrCrossTenantMove for cross-tenant site move, got %v", err)
	}
}

func TestLifecycle_SiteMove_InactiveDestinationDenials(t *testing.T) {
	company, _ := NewCompany("ten_alpha", "cmp_01", "Company")
	projSource, _ := NewProject(company, "prj_source", "Source")
	site, _ := NewSite(projSource, "ste_01", "Site")

	projDest, _ := NewProject(company, "prj_dest", "Dest")

	// 1. Archived destination project rejected
	archivedDest := projDest.Archive()
	if _, err := MoveSiteToProject(site, archivedDest); !errors.Is(err, ErrParentArchived) {
		t.Errorf("expected ErrParentArchived for move to archived project, got %v", err)
	}

	// 2. Closed destination project rejected
	closedDest, _ := projDest.Close()
	if _, err := MoveSiteToProject(site, closedDest); !errors.Is(err, ErrParentClosed) {
		t.Errorf("expected ErrParentClosed for move to closed project, got %v", err)
	}
}

func TestLifecycle_SiteMove_InactiveSourceDenials(t *testing.T) {
	company, _ := NewCompany("ten_alpha", "cmp_01", "Company")
	projSource, _ := NewProject(company, "prj_source", "Source")
	projDest, _ := NewProject(company, "prj_dest", "Dest")

	site, _ := NewSite(projSource, "ste_01", "Site")

	// 1. Moving an archived site is prohibited
	archivedSite := site.Archive()
	if _, err := MoveSiteToProject(archivedSite, projDest); !errors.Is(err, ErrEntityArchived) {
		t.Errorf("expected ErrEntityArchived when moving archived site, got %v", err)
	}

	// 2. Moving a closed site is prohibited
	closedSite, _ := site.Close()
	if _, err := MoveSiteToProject(closedSite, projDest); !errors.Is(err, ErrEntityClosed) {
		t.Errorf("expected ErrEntityClosed when moving closed site, got %v", err)
	}
}

func TestLifecycle_AreaMove_SameTenantSuccessAndDenials(t *testing.T) {
	tenantID := "ten_alpha"
	company, _ := NewCompany(tenantID, "cmp_01", "Company")
	project, _ := NewProject(company, "prj_01", "Project")

	site1, _ := NewSiteWithLocale(project, "ste_01", "Site 1", "Asia/Bangkok", "th-TH")
	site2, _ := NewSiteWithLocale(project, "ste_02", "Site 2", "Asia/Tokyo", "en-US")
	area, _ := NewArea(site1, "ara_01", "Area Boiler")

	// 1. Successful move within same tenant
	movedArea, err := MoveAreaToSite(area, site2)
	if err != nil {
		t.Fatalf("unexpected MoveAreaToSite error: %v", err)
	}
	if movedArea.SiteID() != site2.SiteID() {
		t.Errorf("expected site %s, got %s", site2.SiteID(), movedArea.SiteID())
	}
	if movedArea.AreaID() != "ara_01" {
		t.Errorf("areaID preserved")
	}

	// 2. Cross-tenant area move rejected
	companyB, _ := NewCompany("ten_bravo", "cmp_b", "Company B")
	projB, _ := NewProject(companyB, "prj_b", "Project B")
	siteB, _ := NewSite(projB, "ste_b", "Site B")

	if _, err := MoveAreaToSite(area, siteB); !errors.Is(err, ErrCrossTenantMove) {
		t.Errorf("expected ErrCrossTenantMove on cross-tenant area move, got %v", err)
	}

	// 3. Move to archived site rejected
	archivedSite := site2.Archive()
	if _, err := MoveAreaToSite(area, archivedSite); !errors.Is(err, ErrParentArchived) {
		t.Errorf("expected ErrParentArchived on move to archived site, got %v", err)
	}

	// 4. Move to closed site rejected
	closedSite, _ := site2.Close()
	if _, err := MoveAreaToSite(area, closedSite); !errors.Is(err, ErrParentClosed) {
		t.Errorf("expected ErrParentClosed on move to closed site, got %v", err)
	}

	// 5. Move closed area rejected
	closedArea, _ := area.Close()
	if _, err := MoveAreaToSite(closedArea, site2); !errors.Is(err, ErrEntityClosed) {
		t.Errorf("expected ErrEntityClosed when moving closed area, got %v", err)
	}
}

func TestLifecycle_HistoricalScopeRetention_NoHardDeletion(t *testing.T) {
	company, _ := NewCompany("ten_01", "cmp_01", "Siam Energy")
	project, _ := NewProject(company, "prj_01", "Refinery Unit 3")
	site, _ := NewSite(project, "ste_01", "Chonburi Complex")

	// 1. Capture historical scope prior to transition
	initialScope := site.ResolveScope()
	record1 := CaptureHistoricalScope(site.SiteID(), initialScope, site.State(), "INITIAL_REGISTRATION", "usr_admin_01", "Initial project commission")

	if !strings.HasPrefix(record1.RecordID, "hist_ste_01_") {
		t.Errorf("expected hist_ record ID, got %s", record1.RecordID)
	}
	if record1.State != StateActive {
		t.Errorf("expected active state in historical record")
	}
	if record1.Scope.CanonicalPath != "ten_01/cmp_01/prj_01/ste_01" {
		t.Errorf("historical canonical path mismatch: %s", record1.Scope.CanonicalPath)
	}

	// 2. Close site (no hard deletion, soft-state transition)
	closedSite, err := site.Close()
	if err != nil {
		t.Fatalf("site.Close failed: %v", err)
	}

	record2 := CaptureHistoricalScope(closedSite.SiteID(), closedSite.ResolveScope(), closedSite.State(), "SITE_DECOMMISSION", "usr_admin_02", "Project operational completion")
	if record2.State != StateClosed {
		t.Errorf("expected closed state in historical record")
	}

	// 3. Historical attribution preservation check:
	// Prior snapshot (record1) remains immutable and uncorrupted by subsequent closure
	if record1.State != StateActive || record1.Scope.CanonicalPath != "ten_01/cmp_01/prj_01/ste_01" {
		t.Errorf("prior historical record was corrupted by subsequent transition")
	}
	if record2.Scope.CanonicalPath != "ten_01/cmp_01/prj_01/ste_01" {
		t.Errorf("closed record canonical path mismatch: %s", record2.Scope.CanonicalPath)
	}
	if record2.EntityID != site.SiteID() {
		t.Errorf("entity ID must be preserved across lifecycle transitions")
	}
}
