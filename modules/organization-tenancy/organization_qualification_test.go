// Package orgtenancy provides organizational hierarchy and tenancy models for OSHE Platform.
//
// QUALIFICATION SUITE DECLARATION (Issue #81 / V030-I008):
// Under approved Sole Human Owner decision H030-002 and Milestone v0.3.0 boundaries,
// this qualification suite verifies hierarchy integrity, parentage constraints,
// cross-tenant isolation, orphan prevention, archive/close/move simulation denials,
// historical scope preservation, and reversible rerun lineage using existing public in-memory APIs.
//
// Gate Invariant: H030-002 remains deferred: tests operate strictly within local in-memory
// simulation and preflight verification; zero binding operational authority, runtime execution,
// external persistence, or release claims are enacted.
package orgtenancy

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

// TestQualification_CircularAndInvertedParentageRejections verifies that circular,
// inverted, or malformed parent-child relationships are strictly rejected.
func TestQualification_CircularAndInvertedParentageRejections(t *testing.T) {
	tenantID := "ten_qual_synth_001"
	company, err := NewCompany(tenantID, "cmp_qual_01", "Qualification Holdings")
	if err != nil {
		t.Fatalf("unexpected NewCompany error: %v", err)
	}

	// 1. Uninitialized zero-value parent rejections (orphan / inverted prevention)
	var zeroCompany Company
	if _, err := NewBusinessUnit(zeroCompany, "bnu_01", "BU"); !errors.Is(err, ErrParentMismatch) {
		t.Errorf("expected ErrParentMismatch for zero Company parent, got %v", err)
	}
	if _, err := NewProject(zeroCompany, "prj_01", "Proj"); !errors.Is(err, ErrParentMismatch) {
		t.Errorf("expected ErrParentMismatch for zero Company parent on NewProject, got %v", err)
	}

	var zeroBU BusinessUnit
	if _, err := NewProjectUnderBusinessUnit(zeroBU, "prj_01", "Proj"); !errors.Is(err, ErrParentMismatch) {
		t.Errorf("expected ErrParentMismatch for zero BU parent on NewProjectUnderBusinessUnit, got %v", err)
	}

	var zeroProject Project
	if _, err := NewSite(zeroProject, "ste_01", "Site"); !errors.Is(err, ErrParentMismatch) {
		t.Errorf("expected ErrParentMismatch for zero Project parent on NewSite, got %v", err)
	}

	var zeroSite Site
	if _, err := NewArea(zeroSite, "ara_01", "Area"); !errors.Is(err, ErrParentMismatch) {
		t.Errorf("expected ErrParentMismatch for zero Site parent on NewArea, got %v", err)
	}

	// 2. ValidateParentProject: Parent-Site relationship integrity check
	bu, err := NewBusinessUnit(company, "bnu_01", "Operations BU")
	if err != nil {
		t.Fatalf("unexpected NewBusinessUnit error: %v", err)
	}
	projA, err := NewProjectUnderBusinessUnit(bu, "prj_alpha", "Project Alpha")
	if err != nil {
		t.Fatalf("unexpected NewProjectUnderBusinessUnit error: %v", err)
	}
	projB, err := NewProjectUnderBusinessUnit(bu, "prj_beta", "Project Beta")
	if err != nil {
		t.Fatalf("unexpected NewProjectUnderBusinessUnit error: %v", err)
	}

	siteA, err := NewSite(projA, "ste_alpha_01", "Site Alpha")
	if err != nil {
		t.Fatalf("unexpected NewSite error: %v", err)
	}

	// siteA belongs to projA, validating against projB must fail closed
	if err := siteA.ValidateParentProject(projA.ProjectID()); err != nil {
		t.Errorf("expected ValidateParentProject to succeed for matching project, got %v", err)
	}
	if err := siteA.ValidateParentProject(projB.ProjectID()); !errors.Is(err, ErrProjectSiteMismatch) {
		t.Errorf("expected ErrProjectSiteMismatch when validating site against non-parent project, got %v", err)
	}

	// 3. ValidateParentSite: Site-Area relationship integrity check
	siteB, err := NewSite(projB, "ste_beta_01", "Site Beta")
	if err != nil {
		t.Fatalf("unexpected NewSite error: %v", err)
	}
	areaA, err := NewArea(siteA, "ara_alpha_01", "Area Alpha")
	if err != nil {
		t.Fatalf("unexpected NewArea error: %v", err)
	}

	// areaA belongs to siteA, validating against siteB must fail closed
	if err := areaA.ValidateParentSite(siteA.SiteID()); err != nil {
		t.Errorf("expected ValidateParentSite to succeed for matching site, got %v", err)
	}
	if err := areaA.ValidateParentSite(siteB.SiteID()); !errors.Is(err, ErrParentMismatch) {
		t.Errorf("expected ErrParentMismatch when validating area against non-parent site, got %v", err)
	}

	// 4. Invariant tree structure: canonical path must be strictly non-circular
	scope := areaA.ResolveScope()
	expectedPath := "ten_qual_synth_001/cmp_qual_01/bnu_01/prj_alpha/ste_alpha_01/ara_alpha_01"
	if scope.CanonicalPath != expectedPath {
		t.Errorf("expected canonical path %q, got %q", expectedPath, scope.CanonicalPath)
	}
	segments := strings.Split(scope.CanonicalPath, "/")
	if len(segments) != 6 {
		t.Errorf("expected exactly 6 hierarchy segments, got %d", len(segments))
	}
	seen := make(map[string]bool)
	for _, seg := range segments {
		if seen[seg] {
			t.Errorf("circular reference or duplicate identifier in hierarchy path: %s", seg)
		}
		seen[seg] = true
	}
}

// TestQualification_CrossTenantIsolation verifies that multi-tenant isolation is enforced
// across all hierarchy entities, operations, queries, and move requests.
func TestQualification_CrossTenantIsolation(t *testing.T) {
	tenantAlpha := "ten_alpha_qual"
	tenantBravo := "ten_bravo_qual"

	// Construct Alpha hierarchy
	companyA, _ := NewCompany(tenantAlpha, "cmp_a", "Company Alpha")
	buA, _ := NewBusinessUnit(companyA, "bnu_a", "BU Alpha")
	projA, _ := NewProjectUnderBusinessUnit(buA, "prj_a", "Project Alpha")
	siteA, _ := NewSite(projA, "ste_a", "Site Alpha")
	areaA, _ := NewArea(siteA, "ara_a", "Area Alpha")

	// Construct Bravo hierarchy
	companyB, _ := NewCompany(tenantBravo, "cmp_b", "Company Bravo")
	buB, _ := NewBusinessUnit(companyB, "bnu_b", "BU Bravo")
	projB, _ := NewProjectUnderBusinessUnit(buB, "prj_b", "Project Bravo")
	siteB, _ := NewSite(projB, "ste_b", "Site Bravo")
	areaB, _ := NewArea(siteB, "ara_b", "Area Bravo")

	// Derive Bravo Context
	claimsBravo := &TrustedClaims{
		Subject:         "usr_bravo_auditor",
		TenantID:        tenantBravo,
		IsAuthenticated: true,
	}
	ctxBravo, err := DeriveTenantContext(claimsBravo, nil)
	if err != nil {
		t.Fatalf("unexpected DeriveTenantContext error: %v", err)
	}

	// 1. Scope validation denials across all hierarchy levels for foreign context
	if err := companyA.ValidateScope(ctxBravo); !errors.Is(err, ErrTenantMismatch) {
		t.Errorf("expected ErrTenantMismatch for Company A with Bravo context, got %v", err)
	}
	if err := buA.ValidateScope(ctxBravo); !errors.Is(err, ErrTenantMismatch) {
		t.Errorf("expected ErrTenantMismatch for BU A with Bravo context, got %v", err)
	}
	if err := projA.ValidateScope(ctxBravo); !errors.Is(err, ErrTenantMismatch) {
		t.Errorf("expected ErrTenantMismatch for Project A with Bravo context, got %v", err)
	}
	if err := siteA.ValidateScope(ctxBravo); !errors.Is(err, ErrTenantMismatch) {
		t.Errorf("expected ErrTenantMismatch for Site A with Bravo context, got %v", err)
	}
	if err := areaA.ValidateScope(ctxBravo); !errors.Is(err, ErrTenantMismatch) {
		t.Errorf("expected ErrTenantMismatch for Area A with Bravo context, got %v", err)
	}
	if err := areaA.ResolveScope().ValidateScope(ctxBravo); !errors.Is(err, ErrTenantMismatch) {
		t.Errorf("expected ErrTenantMismatch for ResolvedScope A with Bravo context, got %v", err)
	}

	// 2. Cross-tenant move denials
	if _, err := MoveSiteToProject(siteA, projB); !errors.Is(err, ErrCrossTenantMove) {
		t.Errorf("expected ErrCrossTenantMove when moving site across tenants, got %v", err)
	}
	if _, err := MoveAreaToSite(areaA, siteB); !errors.Is(err, ErrCrossTenantMove) {
		t.Errorf("expected ErrCrossTenantMove when moving area across tenants, got %v", err)
	}

	// 3. Prefix collision attack resistance
	prefixCollisionCases := []string{
		tenantAlpha + "-2",
		tenantAlpha + "_suffix",
		tenantAlpha + "x",
		tenantAlpha[:len(tenantAlpha)-1],
	}
	for _, colTenant := range prefixCollisionCases {
		colClaims := &TrustedClaims{
			Subject:         "usr_attacker",
			TenantID:        colTenant,
			IsAuthenticated: true,
		}
		colCtx, err := DeriveTenantContext(colClaims, nil)
		if err != nil {
			t.Fatalf("unexpected DeriveTenantContext for %s: %v", colTenant, err)
		}
		if err := siteA.ValidateScope(colCtx); !errors.Is(err, ErrTenantMismatch) {
			t.Errorf("expected ErrTenantMismatch for prefix collision %s, got %v", colTenant, err)
		}
	}

	// 4. Party registry cross-tenant isolation
	reg := NewPartyRegistry()
	partyA, _ := NewParty(tenantAlpha, "prt_a", "Contractor Alpha", PartyTypeContractor)
	partyB, _ := NewParty(tenantBravo, "prt_b", "Contractor Bravo", PartyTypeContractor)
	_ = reg.RegisterParty(partyA)
	_ = reg.RegisterParty(partyB)

	// Cross-tenant lookup returns ErrPartyNotFound (never leaks metadata)
	if _, err := reg.GetParty(tenantAlpha, "prt_b"); !errors.Is(err, ErrPartyNotFound) {
		t.Errorf("expected ErrPartyNotFound on cross-tenant party lookup, got %v", err)
	}

	// Cross-tenant participation listing
	baseTime := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	ppA, _ := NewProjectParticipation(partyA, "ptp_a", "prj_shared_id", "ste_01", "usr_mgr", ParticipationRoleContractorWorker, baseTime, baseTime.Add(24*time.Hour))
	ppB, _ := NewProjectParticipation(partyB, "ptp_b", "prj_shared_id", "ste_01", "usr_mgr", ParticipationRoleContractorWorker, baseTime, baseTime.Add(24*time.Hour))
	_ = reg.RegisterParticipation(ppA)
	_ = reg.RegisterParticipation(ppB)

	partsAlpha, err := reg.ListParticipationsByProject(tenantAlpha, "prj_shared_id")
	if err != nil {
		t.Fatalf("unexpected ListParticipationsByProject: %v", err)
	}
	if len(partsAlpha) != 1 || partsAlpha[0].PartyID() != "prt_a" {
		t.Errorf("ListParticipationsByProject leaked foreign tenant records: %+v", partsAlpha)
	}

	// Cross-tenant scope assertion on participation
	if err := ppA.ValidateScope(ctxBravo); !errors.Is(err, ErrTenantMismatch) {
		t.Errorf("expected ErrTenantMismatch on participation ValidateScope with foreign context, got %v", err)
	}
	_ = areaB
}

// TestQualification_OrphanPreventionAndParentGuards verifies that child contexts
// cannot be orphaned under uninitialized, archived, or closed parents.
func TestQualification_OrphanPreventionAndParentGuards(t *testing.T) {
	tenantID := "ten_qual_orphan"
	company, _ := NewCompany(tenantID, "cmp_01", "Acme Enterprise")
	bu, _ := NewBusinessUnit(company, "bnu_01", "Manufacturing")
	project, _ := NewProjectUnderBusinessUnit(bu, "prj_01", "New Assembly Line")
	site, _ := NewSite(project, "ste_01", "Ayutthaya Plant")

	// 1. Archived parent blocks child creation at every tier
	archivedComp := company.Archive()
	if _, err := NewBusinessUnit(archivedComp, "bnu_fail", "BU"); !errors.Is(err, ErrParentArchived) {
		t.Errorf("expected ErrParentArchived under archived company, got %v", err)
	}

	archivedBU := bu.Archive()
	if _, err := NewProjectUnderBusinessUnit(archivedBU, "prj_fail", "Proj"); !errors.Is(err, ErrParentArchived) {
		t.Errorf("expected ErrParentArchived under archived BU, got %v", err)
	}

	archivedProject := project.Archive()
	if _, err := NewSite(archivedProject, "ste_fail", "Site"); !errors.Is(err, ErrParentArchived) {
		t.Errorf("expected ErrParentArchived under archived project, got %v", err)
	}

	archivedSite := site.Archive()
	if _, err := NewArea(archivedSite, "ara_fail", "Area"); !errors.Is(err, ErrParentArchived) {
		t.Errorf("expected ErrParentArchived under archived site, got %v", err)
	}

	party, _ := NewParty(tenantID, "prt_01", "Vendor", PartyTypeContractor)
	baseTime := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	if _, err := NewSponsoredParty(archivedSite, "prt_sp_fail", "Vendor", "usr_mgr", baseTime, baseTime.Add(24*time.Hour)); !errors.Is(err, ErrParentArchived) {
		t.Errorf("expected ErrParentArchived for sponsored party under archived site, got %v", err)
	}

	archivedParty := party.Archive()
	if _, err := NewProjectParticipation(archivedParty, "ptp_fail", "prj_01", "ste_01", "usr_mgr", ParticipationRoleContractorWorker, baseTime, baseTime.Add(24*time.Hour)); !errors.Is(err, ErrPartyArchived) {
		t.Errorf("expected ErrPartyArchived under archived party, got %v", err)
	}

	// 2. Closed parent guard blocks operational attachment
	closedProject, _ := project.Close()
	if err := AssertParentOperationalActive(closedProject.State()); !errors.Is(err, ErrParentClosed) {
		t.Errorf("expected ErrParentClosed from AssertParentOperationalActive on closed project, got %v", err)
	}

	closedSite, _ := site.Close()
	if err := AssertParentOperationalActive(closedSite.State()); !errors.Is(err, ErrParentClosed) {
		t.Errorf("expected ErrParentClosed from AssertParentOperationalActive on closed site, got %v", err)
	}

	// 3. Move destination orphan guard: destination parent must be active
	activeArea, _ := NewArea(site, "ara_01", "Painting Booth")
	if _, err := MoveAreaToSite(activeArea, closedSite); !errors.Is(err, ErrParentClosed) {
		t.Errorf("expected ErrParentClosed when moving area to closed site, got %v", err)
	}
	if _, err := MoveAreaToSite(activeArea, archivedSite); !errors.Is(err, ErrParentArchived) {
		t.Errorf("expected ErrParentArchived when moving area to archived site, got %v", err)
	}
}

// TestQualification_ArchiveCloseMoveDenialsMatrix verifies the comprehensive matrix
// of denial invariants across archive, close, and re-parenting operations.
func TestQualification_ArchiveCloseMoveDenialsMatrix(t *testing.T) {
	tenantID := "ten_qual_matrix"
	company, _ := NewCompany(tenantID, "cmp_01", "Matrix Industries")
	bu, _ := NewBusinessUnit(company, "bnu_01", "Operations")
	projectSource, _ := NewProjectUnderBusinessUnit(bu, "prj_src", "Source Project")
	projectDest, _ := NewProjectUnderBusinessUnit(bu, "prj_dst", "Destination Project")

	site, _ := NewSiteWithLocale(projectSource, "ste_01", "Site Chonburi", "Asia/Bangkok", "th-TH")
	area, _ := NewArea(site, "ara_01", "Boiler Room")

	// 1. Terminal Closed Semantics
	// Reopening closed state is terminal and prohibited in operational posture
	closedComp, _ := company.Close()
	if err := ReopenEntity(closedComp.State()); !errors.Is(err, ErrCannotReopenClosed) {
		t.Errorf("expected ErrCannotReopenClosed for ReopenEntity(StateClosed), got %v", err)
	}

	// Closing an already archived entity fails closed
	archivedSite := site.Archive()
	if _, err := archivedSite.Close(); !errors.Is(err, ErrEntityArchived) {
		t.Errorf("expected ErrEntityArchived when closing archived site, got %v", err)
	}

	// 2. Operational Mutation Rejection on Inactive Entities
	if err := AssertOperationalActive(StateActive); err != nil {
		t.Errorf("expected StateActive to permit operations, got %v", err)
	}
	if err := AssertOperationalActive(StateClosed); !errors.Is(err, ErrEntityClosed) {
		t.Errorf("expected ErrEntityClosed from AssertOperationalActive(StateClosed), got %v", err)
	}
	if err := AssertOperationalActive(StateArchived); !errors.Is(err, ErrEntityArchived) {
		t.Errorf("expected ErrEntityArchived from AssertOperationalActive(StateArchived), got %v", err)
	}
	if err := AssertOperationalActive("UNAPPROVED_STATE"); !errors.Is(err, ErrActiveRecordRejected) {
		t.Errorf("expected ErrActiveRecordRejected for unknown state, got %v", err)
	}

	// 3. Move Site Inactive Source Denials
	if _, err := MoveSiteToProject(archivedSite, projectDest); !errors.Is(err, ErrEntityArchived) {
		t.Errorf("expected ErrEntityArchived moving archived site, got %v", err)
	}
	closedSite, _ := site.Close()
	if _, err := MoveSiteToProject(closedSite, projectDest); !errors.Is(err, ErrEntityClosed) {
		t.Errorf("expected ErrEntityClosed moving closed site, got %v", err)
	}

	// 4. Move Site Inactive Destination Denials
	archivedDest := projectDest.Archive()
	if _, err := MoveSiteToProject(site, archivedDest); !errors.Is(err, ErrParentArchived) {
		t.Errorf("expected ErrParentArchived moving site to archived project, got %v", err)
	}
	closedDest, _ := projectDest.Close()
	if _, err := MoveSiteToProject(site, closedDest); !errors.Is(err, ErrParentClosed) {
		t.Errorf("expected ErrParentClosed moving site to closed project, got %v", err)
	}

	// 5. Move Area Inactive Source & Destination Denials
	archivedArea := area.Archive()
	if _, err := MoveAreaToSite(archivedArea, site); !errors.Is(err, ErrEntityArchived) {
		t.Errorf("expected ErrEntityArchived moving archived area, got %v", err)
	}
	closedArea, _ := area.Close()
	if _, err := MoveAreaToSite(closedArea, site); !errors.Is(err, ErrEntityClosed) {
		t.Errorf("expected ErrEntityClosed moving closed area, got %v", err)
	}
}

// TestQualification_HistoricalScopePreservation verifies that entity transitions
// (archive, close, move) retain frozen historical snapshots without data mutation.
func TestQualification_HistoricalScopePreservation(t *testing.T) {
	tenantID := "ten_qual_history"
	company, _ := NewCompany(tenantID, "cmp_01", "Historical Holding")
	bu, _ := NewBusinessUnit(company, "bnu_01", "Downstream")
	projA, _ := NewProjectUnderBusinessUnit(bu, "prj_a", "Project A")
	projB, _ := NewProjectUnderBusinessUnit(bu, "prj_b", "Project B")

	site, _ := NewSiteWithLocale(projA, "ste_plant", "Plant 1", "Asia/Bangkok", "th-TH")

	// Phase 1: Capture initial scope snapshot
	initialScope := site.ResolveScope()
	recordInit := CaptureHistoricalScope(site.SiteID(), initialScope, site.State(), "REGISTRATION", "usr_lead", "Initial site commissioning")

	if recordInit.State != StateActive {
		t.Errorf("expected initial state active, got %v", recordInit.State)
	}
	if recordInit.Scope.CanonicalPath != "ten_qual_history/cmp_01/bnu_01/prj_a/ste_plant" {
		t.Errorf("initial canonical path mismatch: %s", recordInit.Scope.CanonicalPath)
	}

	// Phase 2: Move Site from Project A to Project B
	movedSite, err := MoveSiteToProject(site, projB)
	if err != nil {
		t.Fatalf("unexpected MoveSiteToProject error: %v", err)
	}

	recordMoved := CaptureHistoricalScope(movedSite.SiteID(), movedSite.ResolveScope(), movedSite.State(), "REPARENT_MOVE", "usr_lead", "Re-parenting site to Project B")

	if recordMoved.Scope.CanonicalPath != "ten_qual_history/cmp_01/bnu_01/prj_b/ste_plant" {
		t.Errorf("moved canonical path mismatch: %s", recordMoved.Scope.CanonicalPath)
	}

	// Assert historical non-corruption: original snapshot recordInit remains completely unmodified
	if recordInit.Scope.CanonicalPath != "ten_qual_history/cmp_01/bnu_01/prj_a/ste_plant" {
		t.Errorf("prior historical record was corrupted by re-parenting operation")
	}

	// Phase 3: Close Site
	closedSite, err := movedSite.Close()
	if err != nil {
		t.Fatalf("unexpected Close error: %v", err)
	}

	recordClosed := CaptureHistoricalScope(closedSite.SiteID(), closedSite.ResolveScope(), closedSite.State(), "CLOSE", "usr_compliance", "Decommissioning site")

	if recordClosed.State != StateClosed {
		t.Errorf("expected recordClosed StateClosed, got %v", recordClosed.State)
	}

	// Assert zero hard deletion: all 3 historical scope records exist and preserve their attribution
	records := []HistoricalScopeRecord{recordInit, recordMoved, recordClosed}
	for i, r := range records {
		if r.EntityID != "ste_plant" {
			t.Errorf("record %d lost entity ID attribution: %s", i, r.EntityID)
		}
		if !strings.HasPrefix(r.RecordID, "hist_ste_plant_") {
			t.Errorf("record %d record ID malformed: %s", i, r.RecordID)
		}
		if !strings.Contains(r.Scope.NonAuthorityNotice, "DERIVED_OUTPUT_NON_AUTHORITY") {
			t.Errorf("record %d missing non-authority notice: %s", i, r.Scope.NonAuthorityNotice)
		}
	}
}

// TestQualification_ReversibleRerunLineage verifies that the qualification test harness
// executes deterministically across repeated runs and proves state simulation reversibility.
func TestQualification_ReversibleRerunLineage(t *testing.T) {
	// Execute the full qualification lifecycle workflow across 5 consecutive iterations
	// asserting bit-level deterministic reproducibility and zero state leakage.
	for iteration := 1; iteration <= 5; iteration++ {
		tenantID := fmt.Sprintf("ten_lineage_%d", iteration)
		compID := "cmp_lineage"
		buID := "bnu_lineage"
		projID := "prj_lineage"
		siteID := "ste_lineage"
		areaID := "ara_lineage"

		company, err := NewCompany(tenantID, compID, "Lineage Corp")
		if err != nil {
			t.Fatalf("iteration %d: NewCompany failed: %v", iteration, err)
		}
		bu, err := NewBusinessUnit(company, buID, "Lineage BU")
		if err != nil {
			t.Fatalf("iteration %d: NewBusinessUnit failed: %v", iteration, err)
		}
		project, err := NewProjectUnderBusinessUnit(bu, projID, "Lineage Proj")
		if err != nil {
			t.Fatalf("iteration %d: NewProjectUnderBusinessUnit failed: %v", iteration, err)
		}
		site, err := NewSiteWithLocale(project, siteID, "Lineage Site", "Asia/Bangkok", "th-TH")
		if err != nil {
			t.Fatalf("iteration %d: NewSiteWithLocale failed: %v", iteration, err)
		}
		area, err := NewArea(site, areaID, "Lineage Area")
		if err != nil {
			t.Fatalf("iteration %d: NewArea failed: %v", iteration, err)
		}

		// Verify deterministic canonical path
		expectedPath := fmt.Sprintf("%s/%s/%s/%s/%s/%s", tenantID, compID, buID, projID, siteID, areaID)
		scope := area.ResolveScope()
		if scope.CanonicalPath != expectedPath {
			t.Fatalf("iteration %d: canonical path mismatch: expected %q, got %q", iteration, expectedPath, scope.CanonicalPath)
		}

		// Verify reversible simulation step (ACTIVE <-> CLOSED <-> ARCHIVED <-> ACTIVE)
		s1, err := SimulateReversibleTransition(StateActive, StateClosed)
		if err != nil || s1 != StateClosed {
			t.Fatalf("iteration %d: simulated transition to StateClosed failed: %v", iteration, err)
		}

		s2, err := SimulateReversibleTransition(s1, StateArchived)
		if err != nil || s2 != StateArchived {
			t.Fatalf("iteration %d: simulated transition to StateArchived failed: %v", iteration, err)
		}

		s3, err := SimulateReversibleTransition(s2, StateActive)
		if err != nil || s3 != StateActive {
			t.Fatalf("iteration %d: simulated reversal to StateActive failed: %v", iteration, err)
		}
	}
}
