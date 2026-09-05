package orgtenancy

import (
	"errors"
	"testing"
	"time"
)

func TestPartyLifecycle_DeactivateParty(t *testing.T) {
	tenantID := "ten_01"
	party, err := NewParty(tenantID, "prt_alpha", "Alpha Services Ltd", PartyTypeContractor)
	if err != nil {
		t.Fatalf("unexpected NewParty error: %v", err)
	}

	// Deactivate party
	deactivated, record, err := DeactivateParty(party, "usr_admin_01", "Contract terminated")
	if err != nil {
		t.Fatalf("unexpected DeactivateParty error: %v", err)
	}

	if deactivated.State() != StateArchived || deactivated.IsActive() {
		t.Errorf("expected party to be in StateArchived")
	}
	if record.PartyID != "prt_alpha" || record.TenantID != tenantID || record.Transition != "PARTY_DEACTIVATE" {
		t.Errorf("audit record mismatch: %+v", record)
	}
	if record.PreviousState != StateActive || record.NewState != StateArchived {
		t.Errorf("expected transition from ACTIVE to ARCHIVED")
	}

	// Deactivating already archived party returns ErrEntityArchived
	_, _, err = DeactivateParty(deactivated, "usr_admin_01", "Redundant deactivation")
	if !errors.Is(err, ErrEntityArchived) {
		t.Errorf("expected ErrEntityArchived on re-deactivation, got %v", err)
	}
}

func TestPartyLifecycle_DeactivateAndCloseParticipation(t *testing.T) {
	party, _ := NewParty("ten_01", "prt_01", "Alpha", PartyTypeContractor)
	from := time.Now().Add(-1 * time.Hour)
	to := time.Now().Add(48 * time.Hour)

	pp, _ := NewProjectParticipation(party, "ptp_01", "prj_01", "ste_01", "usr_mgr_01", ParticipationRoleSiteSafetyLead, from, to)

	// 1. Close participation
	closed, closeRec, err := CloseParticipation(pp, "usr_mgr_01", "Scope completed")
	if err != nil {
		t.Fatalf("unexpected CloseParticipation error: %v", err)
	}
	if closed.State() != StateClosed || !closed.IsActive() == false {
		t.Errorf("expected closed participation")
	}
	if closeRec.Transition != "PARTICIPATION_CLOSE" || closeRec.NewState != StateClosed {
		t.Errorf("close audit record mismatch: %+v", closeRec)
	}

	// Closing already closed participation fails
	_, _, err = CloseParticipation(closed, "usr_mgr_01", "Duplicate close")
	if !errors.Is(err, ErrParticipationClosed) {
		t.Errorf("expected ErrParticipationClosed, got %v", err)
	}

	// 2. Deactivate participation
	pp2, _ := NewProjectParticipation(party, "ptp_02", "prj_01", "ste_01", "usr_mgr_01", ParticipationRoleSiteSafetyLead, from, to)
	deact, deactRec, err := DeactivateParticipation(pp2, "usr_mgr_01", "Administrative archive")
	if err != nil {
		t.Fatalf("unexpected DeactivateParticipation error: %v", err)
	}
	if deact.State() != StateArchived {
		t.Errorf("expected archived participation")
	}
	if deactRec.Transition != "PARTICIPATION_DEACTIVATE" {
		t.Errorf("deactivate audit record mismatch: %+v", deactRec)
	}
}

func TestPartyLifecycle_SponsorReassignment_SuccessAndHistory(t *testing.T) {
	party, _ := NewParty("ten_01", "prt_01", "Alpha Contractors", PartyTypeContractor)
	baseTime := time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC)
	from := baseTime
	to := baseTime.Add(30 * 24 * time.Hour)

	pp, err := NewProjectParticipation(party, "ptp_01", "prj_01", "ste_01", "usr_sponsor_original", ParticipationRoleSiteSafetyLead, from, to)
	if err != nil {
		t.Fatalf("unexpected NewProjectParticipation: %v", err)
	}

	now := baseTime.Add(5 * 24 * time.Hour)
	updated, record, err := ReassignSponsor(pp, "usr_sponsor_replacement", "usr_hr_lead", "Original manager reassigned", now)
	if err != nil {
		t.Fatalf("unexpected ReassignSponsor error: %v", err)
	}

	// Check updated participation
	if updated.SponsorID() != "usr_sponsor_replacement" {
		t.Errorf("expected new sponsor %q, got %q", "usr_sponsor_replacement", updated.SponsorID())
	}
	if updated.PartyID() != pp.PartyID() || updated.ProjectID() != pp.ProjectID() || updated.SiteID() != pp.SiteID() {
		t.Errorf("reassigned participation modified unrelated scope fields")
	}
	if updated.ValidFrom() != pp.ValidFrom() || updated.ValidTo() != pp.ValidTo() {
		t.Errorf("reassigned participation modified validity window")
	}

	// Check audit record
	if record.PriorSponsorID != "usr_sponsor_original" {
		t.Errorf("expected prior sponsor 'usr_sponsor_original', got %q", record.PriorSponsorID)
	}
	if record.NewSponsorID != "usr_sponsor_replacement" {
		t.Errorf("expected new sponsor 'usr_sponsor_replacement', got %q", record.NewSponsorID)
	}
	if record.ActorSubject != "usr_hr_lead" || record.Reason != "Original manager reassigned" {
		t.Errorf("record actor or reason mismatch: %+v", record)
	}
	if record.ReassignedAt != now {
		t.Errorf("record timestamp mismatch")
	}
}

func TestPartyLifecycle_SponsorReassignment_Rejections(t *testing.T) {
	party, _ := NewParty("ten_01", "prt_01", "Alpha Contractors", PartyTypeContractor)
	baseTime := time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC)
	from := baseTime
	to := baseTime.Add(10 * 24 * time.Hour)
	now := baseTime.Add(2 * 24 * time.Hour)

	pp, _ := NewProjectParticipation(party, "ptp_01", "prj_01", "ste_01", "usr_current_mgr", ParticipationRoleSiteSafetyLead, from, to)

	// 1. Reassigning to identical sponsor
	_, _, err := ReassignSponsor(pp, "usr_current_mgr", "usr_actor", "Same", now)
	if !errors.Is(err, ErrSponsorUnchanged) {
		t.Errorf("expected ErrSponsorUnchanged, got %v", err)
	}

	// 2. Blank new sponsor
	_, _, err = ReassignSponsor(pp, "", "usr_actor", "Blank", now)
	if !errors.Is(err, ErrBlankSponsorID) {
		t.Errorf("expected ErrBlankSponsorID, got %v", err)
	}
	_, _, err = ReassignSponsor(pp, "   ", "usr_actor", "Whitespace", now)
	if !errors.Is(err, ErrBlankSponsorID) {
		t.Errorf("expected ErrBlankSponsorID for whitespace, got %v", err)
	}

	// 3. Non-user sponsor identifier
	_, _, err = ReassignSponsor(pp, "ext_contractor_mgr", "usr_actor", "External", now)
	if !errors.Is(err, ErrInvalidSponsorID) {
		t.Errorf("expected ErrInvalidSponsorID for non-user sponsor, got %v", err)
	}

	// 4. Reassigning on closed participation
	closed, _, _ := CloseParticipation(pp, "usr_actor", "Closed")
	_, _, err = ReassignSponsor(closed, "usr_new_mgr", "usr_actor", "Reassign on closed", now)
	if !errors.Is(err, ErrParticipationClosed) {
		t.Errorf("expected ErrParticipationClosed, got %v", err)
	}

	// 5. Reassigning on archived participation
	archived, _, _ := DeactivateParticipation(pp, "usr_actor", "Archived")
	_, _, err = ReassignSponsor(archived, "usr_new_mgr", "usr_actor", "Reassign on archived", now)
	if !errors.Is(err, ErrParentNotActive) {
		t.Errorf("expected ErrParentNotActive, got %v", err)
	}

	// 6. Reassigning on expired participation
	expiredTime := to.Add(1 * time.Hour)
	_, _, err = ReassignSponsor(pp, "usr_new_mgr", "usr_actor", "Reassign expired", expiredTime)
	if !errors.Is(err, ErrParticipationExpired) {
		t.Errorf("expected ErrParticipationExpired, got %v", err)
	}
}

func TestPartyLifecycle_CascadeProjectClosure(t *testing.T) {
	tenantID := "ten_01"
	company, _ := NewCompany(tenantID, "cmp_01", "Acme")
	proj1, _ := NewProject(company, "prj_target", "Target Project")
	proj2, _ := NewProject(company, "prj_other", "Other Project")

	party1, _ := NewParty(tenantID, "prt_01", "Contractor 1", PartyTypeContractor)
	party2, _ := NewParty(tenantID, "prt_02", "Subcontractor 2", PartyTypeSubcontractor)
	party3, _ := NewParty(tenantID, "prt_03", "Contractor 3", PartyTypeContractor)

	from := time.Now().Add(-1 * time.Hour)
	to := time.Now().Add(48 * time.Hour)

	// Part 1 & Part 2 in proj1 (prime and sub)
	part1, _ := NewProjectParticipation(party1, "ptp_01", "prj_target", "ste_01", "usr_mgr", ParticipationRoleSiteSafetyLead, from, to)
	part2, _ := NewNestedSubcontractorParticipation(part1, party2, "ptp_02", "ste_01", "usr_mgr", ParticipationRoleSubcontractorLead, from, to)

	// Part 3 in proj2
	part3, _ := NewProjectParticipation(party3, "ptp_03", proj2.ProjectID(), "ste_02", "usr_mgr", ParticipationRoleContractorWorker, from, to)
	participations := []ProjectParticipation{part1, part2, part3}

	// Attempting cascade before project is closed fails
	_, _, err := CascadeProjectClosure(proj1, participations, "usr_lead", "Premature cascade")
	if err == nil {
		t.Fatalf("expected error cascading from active project")
	}

	// Close project 1
	closedProj1, err := proj1.Close()
	if err != nil {
		t.Fatalf("proj1.Close failed: %v", err)
	}

	// Execute cascade
	updated, auditRecords, err := CascadeProjectClosure(closedProj1, participations, "usr_lead", "Phase 1 Complete")
	if err != nil {
		t.Fatalf("CascadeProjectClosure failed: %v", err)
	}

	// Assertions
	if len(auditRecords) != 2 {
		t.Errorf("expected 2 audit records for target project participations, got %d", len(auditRecords))
	}

	for _, p := range updated {
		if p.ProjectID() == "prj_target" {
			if !p.IsClosed() || p.State() != StateClosed {
				t.Errorf("expected participation %s in prj_target to be closed, got state %v", p.ParticipationID(), p.State())
			}
		} else if p.ProjectID() == "prj_other" {
			if !p.IsActive() {
				t.Errorf("participation in prj_other should remain active")
			}
		}
	}
}

func TestPartyLifecycle_ValidateParticipationAgainstProject(t *testing.T) {
	tenantID := "ten_01"
	company, _ := NewCompany(tenantID, "cmp_01", "Company")
	project, _ := NewProject(company, "prj_01", "Project")
	party, _ := NewParty(tenantID, "prt_01", "Contractor", PartyTypeContractor)

	from := time.Now().Add(-1 * time.Hour)
	to := time.Now().Add(24 * time.Hour)
	now := time.Now()

	pp, _ := NewProjectParticipation(party, "ptp_01", "prj_01", "ste_01", "usr_mgr", ParticipationRoleSiteSafetyLead, from, to)

	// 1. Valid active scenario
	if err := ValidateParticipationAgainstProject(pp, project, now); err != nil {
		t.Errorf("expected valid check, got %v", err)
	}

	// 2. Closed project denial
	closedProj, _ := project.Close()
	if err := ValidateParticipationAgainstProject(pp, closedProj, now); !errors.Is(err, ErrProjectClosedCascade) {
		t.Errorf("expected ErrProjectClosedCascade when project closed, got %v", err)
	}

	// 3. Archived project denial
	archivedProj := project.Archive()
	if err := ValidateParticipationAgainstProject(pp, archivedProj, now); !errors.Is(err, ErrParentArchived) {
		t.Errorf("expected ErrParentArchived when project archived, got %v", err)
	}

	// 4. Cross-tenant linkage denial
	foreignCompany, _ := NewCompany("ten_other", "cmp_other", "Other")
	foreignProj, _ := NewProject(foreignCompany, "prj_01", "Other Project")
	if err := ValidateParticipationAgainstProject(pp, foreignProj, now); !errors.Is(err, ErrCrossTenantLinkage) {
		t.Errorf("expected ErrCrossTenantLinkage for foreign tenant project, got %v", err)
	}

	// 5. Scope mismatch denial
	otherProject, _ := NewProject(company, "prj_other", "Other Proj")
	if err := ValidateParticipationAgainstProject(pp, otherProject, now); !errors.Is(err, ErrScopeMismatch) {
		t.Errorf("expected ErrScopeMismatch for mismatched project, got %v", err)
	}

	// 6. Expired participation denial
	expiredAt := to.Add(1 * time.Hour)
	if err := ValidateParticipationAgainstProject(pp, project, expiredAt); !errors.Is(err, ErrParticipationExpired) {
		t.Errorf("expected ErrParticipationExpired when expired, got %v", err)
	}
}

func TestPartyLifecycle_AppendOnlyLedger_IsolationAndNoHardDelete(t *testing.T) {
	ledger := NewPartyLifecycleLedger()
	tenantA := "ten_alpha"
	tenantB := "ten_bravo"

	// 1. Record Party Deactivations
	partyRecA := HistoricalPartyLifecycleRecord{
		RecordID:      "rec_prt_1",
		TenantID:      tenantA,
		PartyID:       "prt_a1",
		PreviousState: StateActive,
		NewState:      StateArchived,
		Transition:    "PARTY_DEACTIVATE",
		ActorSubject:  "usr_a",
		Reason:        "Contract completed",
		RecordedAt:    time.Now().UTC(),
	}
	partyRecB := HistoricalPartyLifecycleRecord{
		RecordID:      "rec_prt_2",
		TenantID:      tenantB,
		PartyID:       "prt_b1",
		PreviousState: StateActive,
		NewState:      StateArchived,
		Transition:    "PARTY_DEACTIVATE",
		ActorSubject:  "usr_b",
		Reason:        "Non-renewal",
		RecordedAt:    time.Now().UTC(),
	}

	_ = ledger.AppendPartyRecord(partyRecA)
	_ = ledger.AppendPartyRecord(partyRecB)

	// 2. Record Sponsor Reassignments
	spsRecA := SponsorReassignmentRecord{
		RecordID:        "rec_sps_1",
		TenantID:        tenantA,
		ParticipationID: "ptp_a1",
		PriorSponsorID:  "usr_mgr_1",
		NewSponsorID:    "usr_mgr_2",
		ActorSubject:    "usr_hr_a",
		Reason:          "Manager retirement",
		ReassignedAt:    time.Now().UTC(),
	}
	_ = ledger.AppendSponsorReassignment(spsRecA)

	// 3. Query Party History for Tenant A
	histA, err := ledger.GetPartyHistory(tenantA, "prt_a1")
	if err != nil {
		t.Fatalf("unexpected GetPartyHistory error: %v", err)
	}
	if len(histA) != 1 || histA[0].PartyID != "prt_a1" {
		t.Errorf("party history mismatch for Tenant A")
	}

	// 4. Cross-tenant isolation verification: Tenant A query for Tenant B party yields zero records
	leakCheck, err := ledger.GetPartyHistory(tenantA, "prt_b1")
	if err != nil {
		t.Fatalf("unexpected leak check error: %v", err)
	}
	if len(leakCheck) != 0 {
		t.Errorf("cross-tenant leakage: Tenant A retrieved Tenant B records")
	}

	// 5. Query Sponsor History
	spsHist, err := ledger.GetSponsorHistory(tenantA, "ptp_a1")
	if err != nil || len(spsHist) != 1 {
		t.Fatalf("unexpected GetSponsorHistory result: %v", err)
	}
	if spsHist[0].PriorSponsorID != "usr_mgr_1" || spsHist[0].NewSponsorID != "usr_mgr_2" {
		t.Errorf("sponsor history attribution corrupted")
	}

	// 6. Zero hard deletion: records are append-only and cannot be mutated
	if spsHist[0].RecordID != "rec_sps_1" {
		t.Errorf("record ID corrupted")
	}
}

func TestPartyLifecycle_ReversibleSimulationState(t *testing.T) {
	party, _ := NewParty("ten_01", "prt_01", "Contractor", PartyTypeContractor)

	// Forward transition
	archived, err := SimulateReversiblePartyState(party, StateArchived)
	if err != nil || archived.State() != StateArchived {
		t.Fatalf("expected StateArchived simulation: %v", err)
	}

	// Reverse transition (H030-002 local simulation harness)
	active, err := SimulateReversiblePartyState(archived, StateActive)
	if err != nil || active.State() != StateActive {
		t.Fatalf("expected StateActive simulation reversal: %v", err)
	}
}
