// Package orgtenancy provides organizational hierarchy and tenancy models for OSHE Platform.
//
// QUALIFICATION SUITE DECLARATION (Issue #85 / V030-I012):
// Under approved Sole Human Owner decision H030-002 and Milestone v0.3.0 boundaries,
// this qualification suite verifies deterministic contractor boundary negative controls:
// 1. Contractor nesting ceiling (depth 1 max, sub-subcontractor depth 2+ strictly rejected).
// 2. Sibling and unrelated-party isolation (lateral contractor and cross-project scope denial).
// 3. Expired and closed participation denials (strict temporal windows and closed-state denial).
// 4. Internal company and admin escalation denial (strict rejection of administrative authority).
// 5. Project-closure denial and cascade (in-memory cascade and operation denial on closed projects).
// 6. Cross-tenant historical-access isolation (append-only ledger tenant boundary isolation).
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

// TestQualification_ContractorNestingCeiling verifies that contractor-to-subcontractor
// nesting is capped strictly at depth 1 (Primary Contractor -> Subcontractor). Any attempt
// to nest further (sub-subcontractor at depth 2+) is deterministically rejected with ErrNestingDepthExceeded.
func TestQualification_ContractorNestingCeiling(t *testing.T) {
	tenantID := "ten_qual_synth_nesting_01"
	partyPrime, err := NewParty(tenantID, "prt_prime_01", "Prime Industrial Services", PartyTypeContractor)
	if err != nil {
		t.Fatalf("unexpected NewParty error: %v", err)
	}
	partySub, err := NewParty(tenantID, "prt_sub_01", "Specialized Piping Subcontractor", PartyTypeSubcontractor)
	if err != nil {
		t.Fatalf("unexpected NewParty error: %v", err)
	}
	partySubSub, err := NewParty(tenantID, "prt_subsub_01", "Unauthorized Deep Contractor", PartyTypeSubcontractor)
	if err != nil {
		t.Fatalf("unexpected NewParty error: %v", err)
	}

	baseTime := time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC)
	primeFrom := baseTime
	primeTo := baseTime.Add(60 * 24 * time.Hour)

	// 1. Primary contractor participation (depth 0)
	primePP, err := NewProjectParticipation(
		partyPrime,
		"ptp_prime_01",
		"prj_qual_nesting",
		"ste_site_01",
		"usr_sponsor_lead_01",
		ParticipationRoleContractorWorker,
		primeFrom,
		primeTo,
	)
	if err != nil {
		t.Fatalf("unexpected NewProjectParticipation error: %v", err)
	}

	if primePP.NestingDepth() != 0 {
		t.Fatalf("expected prime nesting depth 0, got %d", primePP.NestingDepth())
	}
	if primePP.IsSubcontractor() {
		t.Fatalf("expected prime IsSubcontractor == false")
	}
	if primePP.ParentParticipationID() != "" {
		t.Fatalf("expected prime ParentParticipationID to be empty, got %q", primePP.ParentParticipationID())
	}

	// 2. Subcontractor participation under prime (depth 1) - permitted within ceiling
	subFrom := baseTime.Add(1 * 24 * time.Hour)
	subTo := baseTime.Add(30 * 24 * time.Hour)
	subPP, err := NewNestedSubcontractorParticipation(
		primePP,
		partySub,
		"ptp_sub_01",
		"ste_site_01",
		"usr_sponsor_sub_01",
		ParticipationRoleSubcontractorLead,
		subFrom,
		subTo,
	)
	if err != nil {
		t.Fatalf("unexpected NewNestedSubcontractorParticipation error: %v", err)
	}

	if subPP.NestingDepth() != 1 {
		t.Fatalf("expected subcontractor nesting depth 1, got %d", subPP.NestingDepth())
	}
	if !subPP.IsSubcontractor() {
		t.Fatalf("expected subcontractor IsSubcontractor == true")
	}
	if subPP.ParentParticipationID() != primePP.ParticipationID() {
		t.Fatalf("expected ParentParticipationID %q, got %q", primePP.ParticipationID(), subPP.ParentParticipationID())
	}

	// 3. Negative control: Attempt sub-subcontractor under subPP (depth 2) - MUST fail with ErrNestingDepthExceeded
	_, err = NewNestedSubcontractorParticipation(
		subPP,
		partySubSub,
		"ptp_subsub_01",
		"ste_site_01",
		"usr_sponsor_subsub",
		ParticipationRoleSubcontractorLead,
		subFrom,
		subTo,
	)
	if !errors.Is(err, ErrNestingDepthExceeded) {
		t.Fatalf("expected ErrNestingDepthExceeded for depth 2 nesting attempt, got %v", err)
	}

	// Verify constant invariant
	if MaxContractorNestingDepth != 1 {
		t.Fatalf("expected MaxContractorNestingDepth == 1, got %d", MaxContractorNestingDepth)
	}

	// 4. Temporal containment negative control: Subcontractor validity window cannot exceed parent window
	// Case A: validFrom precedes parent validFrom
	invalidEarlyFrom := primeFrom.Add(-1 * time.Hour)
	_, err = NewNestedSubcontractorParticipation(
		primePP,
		partySub,
		"ptp_sub_invalid_early",
		"ste_site_01",
		"usr_sponsor_sub_01",
		ParticipationRoleSubcontractorLead,
		invalidEarlyFrom,
		subTo,
	)
	if !errors.Is(err, ErrValidityWindowExceedsParent) {
		t.Errorf("expected ErrValidityWindowExceedsParent when sub starts before parent, got %v", err)
	}

	// Case B: validTo exceeds parent validTo
	invalidLateTo := primeTo.Add(1 * time.Hour)
	_, err = NewNestedSubcontractorParticipation(
		primePP,
		partySub,
		"ptp_sub_invalid_late",
		"ste_site_01",
		"usr_sponsor_sub_01",
		ParticipationRoleSubcontractorLead,
		subFrom,
		invalidLateTo,
	)
	if !errors.Is(err, ErrValidityWindowExceedsParent) {
		t.Errorf("expected ErrValidityWindowExceedsParent when sub ends after parent, got %v", err)
	}

	// 5. Site scope containment: Subcontractor cannot expand to a different site when parent is site-restricted
	_, err = NewNestedSubcontractorParticipation(
		primePP,
		partySub,
		"ptp_sub_invalid_site",
		"ste_other_site_99",
		"usr_sponsor_sub_01",
		ParticipationRoleSubcontractorLead,
		subFrom,
		subTo,
	)
	if !errors.Is(err, ErrScopeMismatch) {
		t.Errorf("expected ErrScopeMismatch when sub targets foreign site, got %v", err)
	}

	// 6. Inactive parent contractor denial: Nested subcontractor creation under archived parent contractor is rejected
	archivedPrime := primePP.Archive()
	_, err = NewNestedSubcontractorParticipation(
		archivedPrime,
		partySub,
		"ptp_sub_archived_parent",
		"ste_site_01",
		"usr_sponsor_sub_01",
		ParticipationRoleSubcontractorLead,
		subFrom,
		subTo,
	)
	if !errors.Is(err, ErrParentNotActive) {
		t.Errorf("expected ErrParentNotActive when parent is archived, got %v", err)
	}

	// 7. Closed parent contractor denial: Nested subcontractor creation under closed parent contractor is rejected
	closedPrime, err := primePP.Close()
	if err != nil {
		t.Fatalf("unexpected Close error: %v", err)
	}
	_, err = NewNestedSubcontractorParticipation(
		closedPrime,
		partySub,
		"ptp_sub_closed_parent",
		"ste_site_01",
		"usr_sponsor_sub_01",
		ParticipationRoleSubcontractorLead,
		subFrom,
		subTo,
	)
	if !errors.Is(err, ErrParentClosed) {
		t.Errorf("expected ErrParentClosed when parent is closed, got %v", err)
	}
}

// TestQualification_SiblingAndUnrelatedPartyIsolation verifies that contractors and
// subcontractors have zero lateral authority over sibling contractors, sibling projects,
// or cross-tenant scopes.
func TestQualification_SiblingAndUnrelatedPartyIsolation(t *testing.T) {
	tenantA := "ten_qual_synth_iso_a"
	tenantB := "ten_qual_synth_iso_b"

	partyPrime1, _ := NewParty(tenantA, "prt_prime_01", "Civil Works Prime", PartyTypeContractor)
	partyPrime2, _ := NewParty(tenantA, "prt_prime_02", "Electrical Prime", PartyTypeContractor)
	partySub1, _ := NewParty(tenantA, "prt_sub_01", "HVAC Subcontractor", PartyTypeSubcontractor)
	partySub2, _ := NewParty(tenantA, "prt_sub_02", "Ducting Subcontractor", PartyTypeSubcontractor)

	partyForeign, _ := NewParty(tenantB, "prt_foreign_01", "Foreign Tenant Contractor", PartyTypeContractor)

	baseTime := time.Date(2026, 9, 5, 8, 0, 0, 0, time.UTC)
	from := baseTime
	to := baseTime.Add(30 * 24 * time.Hour)

	// Prime 1 in Project A
	ppPrime1, err := NewProjectParticipation(partyPrime1, "ptp_prime_01", "prj_iso_01", "ste_01", "usr_sponsor_01", ParticipationRoleContractorWorker, from, to)
	if err != nil {
		t.Fatalf("unexpected NewProjectParticipation error: %v", err)
	}

	// Prime 2 in same Project A (sibling contractor)
	ppPrime2, err := NewProjectParticipation(partyPrime2, "ptp_prime_02", "prj_iso_01", "ste_01", "usr_sponsor_02", ParticipationRoleContractorWorker, from, to)
	if err != nil {
		t.Fatalf("unexpected NewProjectParticipation error: %v", err)
	}

	// Prime 1 in Project B (different project, same contractor)
	ppPrime1ProjB, err := NewProjectParticipation(partyPrime1, "ptp_prime_01_projb", "prj_iso_02", "ste_02", "usr_sponsor_01", ParticipationRoleContractorWorker, from, to)
	if err != nil {
		t.Fatalf("unexpected NewProjectParticipation error: %v", err)
	}

	// Subcontractor 1 under Prime 1
	ppSub1, err := NewNestedSubcontractorParticipation(ppPrime1, partySub1, "ptp_sub_01", "ste_01", "usr_sponsor_03", ParticipationRoleSubcontractorLead, from, to)
	if err != nil {
		t.Fatalf("unexpected NewNestedSubcontractorParticipation error: %v", err)
	}

	// Subcontractor 2 under Prime 1 (sibling sub under same prime)
	ppSub2, err := NewNestedSubcontractorParticipation(ppPrime1, partySub2, "ptp_sub_02", "ste_01", "usr_sponsor_04", ParticipationRoleSubcontractorLead, from, to)
	if err != nil {
		t.Fatalf("unexpected NewNestedSubcontractorParticipation error: %v", err)
	}

	// Foreign contractor in Tenant B
	ppForeign, err := NewProjectParticipation(partyForeign, "ptp_foreign_01", "prj_foreign_01", "ste_foreign", "usr_sponsor_foreign", ParticipationRoleContractorWorker, from, to)
	if err != nil {
		t.Fatalf("unexpected NewProjectParticipation error: %v", err)
	}

	// 1. Lateral contractor denial: Prime 1 accessing sibling Prime 2 in same project
	if err := ppPrime1.ValidateNoSiblingAccess(ppPrime2); !errors.Is(err, ErrSiblingAccessDenied) {
		t.Errorf("expected ErrSiblingAccessDenied between sibling contractors in same project, got %v", err)
	}
	if err := ppPrime2.ValidateNoSiblingAccess(ppPrime1); !errors.Is(err, ErrSiblingAccessDenied) {
		t.Errorf("expected ErrSiblingAccessDenied in reverse direction, got %v", err)
	}

	// 2. Cross-project contractor denial: Prime 1 accessing its own participation in different project
	if err := ppPrime1.ValidateNoSiblingAccess(ppPrime1ProjB); !errors.Is(err, ErrSiblingAccessDenied) {
		t.Errorf("expected ErrSiblingAccessDenied across distinct projects, got %v", err)
	}

	// 3. Sibling subcontractor denial: Sub 1 accessing sibling Sub 2 under same prime
	if err := ppSub1.ValidateNoSiblingAccess(ppSub2); !errors.Is(err, ErrSiblingAccessDenied) {
		t.Errorf("expected ErrSiblingAccessDenied between sibling subcontractors, got %v", err)
	}
	if err := ppSub2.ValidateNoSiblingAccess(ppSub1); !errors.Is(err, ErrSiblingAccessDenied) {
		t.Errorf("expected ErrSiblingAccessDenied in reverse sibling direction, got %v", err)
	}

	// 4. Lateral/upward subcontractor denial: Child Sub 1 accessing parent Prime 1
	if err := ppSub1.ValidateNoSiblingAccess(ppPrime1); !errors.Is(err, ErrSiblingAccessDenied) {
		t.Errorf("expected ErrSiblingAccessDenied for child attempting lateral/upward access to parent, got %v", err)
	}

	// 5. Downward parent relationship: Parent Prime 1 accessing its child Sub 1 is permitted
	if err := ppPrime1.ValidateNoSiblingAccess(ppSub1); err != nil {
		t.Errorf("expected nil for downward parent-to-child check, got %v", err)
	}

	// 6. Identity / Self access is permitted
	if err := ppPrime1.ValidateNoSiblingAccess(ppPrime1); err != nil {
		t.Errorf("expected nil for self-access check, got %v", err)
	}
	if err := ppSub1.ValidateNoSiblingAccess(ppSub1); err != nil {
		t.Errorf("expected nil for self-access check on subcontractor, got %v", err)
	}

	// 7. Cross-tenant linkage rejection: Prime 1 accessing foreign tenant participation
	if err := ppPrime1.ValidateNoSiblingAccess(ppForeign); !errors.Is(err, ErrCrossTenantLinkage) {
		t.Errorf("expected ErrCrossTenantLinkage for foreign tenant target, got %v", err)
	}
}

// TestQualification_ExpiredAndClosedParticipationDenials verifies that expired or closed
// participations strictly deny operational validation, sponsor reassignment, and child nesting.
func TestQualification_ExpiredAndClosedParticipationDenials(t *testing.T) {
	tenantID := "ten_qual_synth_exp_01"
	party, err := NewParty(tenantID, "prt_exp_01", "Temporal Boundary Testing Ltd", PartyTypeContractor)
	if err != nil {
		t.Fatalf("unexpected NewParty: %v", err)
	}

	t0 := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	t1 := time.Date(2026, 9, 30, 23, 59, 59, 0, time.UTC)

	pp, err := NewProjectParticipation(
		party,
		"ptp_temporal_01",
		"prj_temporal_01",
		"ste_temporal_01",
		"usr_sponsor_temporal",
		ParticipationRoleContractorWorker,
		t0,
		t1,
	)
	if err != nil {
		t.Fatalf("unexpected NewProjectParticipation: %v", err)
	}

	// 1. Temporal boundary validations via ValidateScopeAt
	// Before validity window
	beforeWindow := t0.Add(-1 * time.Second)
	if err := pp.ValidateScopeAt("prj_temporal_01", "ste_temporal_01", beforeWindow); !errors.Is(err, ErrParticipationNotYetValid) {
		t.Errorf("expected ErrParticipationNotYetValid before window, got %v", err)
	}
	if pp.IsValidAt(beforeWindow) {
		t.Errorf("expected IsValidAt to be false before window")
	}

	// Exactly at start of validity window
	if err := pp.ValidateScopeAt("prj_temporal_01", "ste_temporal_01", t0); err != nil {
		t.Errorf("expected valid at t0, got %v", err)
	}
	if !pp.IsValidAt(t0) {
		t.Errorf("expected IsValidAt to be true at t0")
	}

	// Inside validity window
	midWindow := time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)
	if err := pp.ValidateScopeAt("prj_temporal_01", "ste_temporal_01", midWindow); err != nil {
		t.Errorf("expected valid at midWindow, got %v", err)
	}
	if !pp.IsValidAt(midWindow) {
		t.Errorf("expected IsValidAt to be true at midWindow")
	}

	// Exactly at end of validity window
	if err := pp.ValidateScopeAt("prj_temporal_01", "ste_temporal_01", t1); err != nil {
		t.Errorf("expected valid at t1, got %v", err)
	}
	if !pp.IsValidAt(t1) {
		t.Errorf("expected IsValidAt to be true at t1")
	}

	// After validity window
	afterWindow := t1.Add(1 * time.Second)
	if err := pp.ValidateScopeAt("prj_temporal_01", "ste_temporal_01", afterWindow); !errors.Is(err, ErrParticipationExpired) {
		t.Errorf("expected ErrParticipationExpired after window, got %v", err)
	}
	if pp.IsValidAt(afterWindow) {
		t.Errorf("expected IsValidAt to be false after window")
	}

	// 2. ReassignSponsor on expired participation must fail with ErrParticipationExpired
	_, _, err = ReassignSponsor(pp, "usr_new_sponsor_01", "usr_actor_01", "reassignment after expiry", afterWindow)
	if !errors.Is(err, ErrParticipationExpired) {
		t.Errorf("expected ErrParticipationExpired on ReassignSponsor after window, got %v", err)
	}

	// 3. Closure state denials
	closedPP, closeRec, err := CloseParticipation(pp, "usr_admin_actor", "Scope contract concluded")
	if err != nil {
		t.Fatalf("unexpected CloseParticipation error: %v", err)
	}
	if !closedPP.IsClosed() {
		t.Errorf("expected closedPP.IsClosed() == true")
	}
	if closedPP.IsActive() {
		t.Errorf("expected closedPP.IsActive() == false")
	}
	if closeRec.Transition != "PARTICIPATION_CLOSE" || closeRec.NewState != StateClosed {
		t.Errorf("unexpected close audit record: %+v", closeRec)
	}

	// Negative control: Close on already closed participation
	_, _, err = CloseParticipation(closedPP, "usr_admin_actor", "duplicate close")
	if !errors.Is(err, ErrParticipationClosed) {
		t.Errorf("expected ErrParticipationClosed when closing already closed participation, got %v", err)
	}

	// Negative control: ReassignSponsor on closed participation
	_, _, err = ReassignSponsor(closedPP, "usr_new_sponsor_02", "usr_actor_02", "reassignment on closed", midWindow)
	if !errors.Is(err, ErrParticipationClosed) {
		t.Errorf("expected ErrParticipationClosed on ReassignSponsor for closed participation, got %v", err)
	}

	// Negative control: Operational scope access on closed participation fails closed
	if err := closedPP.ValidateScopeAt("prj_temporal_01", "ste_temporal_01", midWindow); !errors.Is(err, ErrEntityArchived) {
		t.Errorf("expected ErrEntityArchived (not active) for closed participation scope validation, got %v", err)
	}

	// Negative control: Subcontractor nesting under closed parent fails with ErrParentClosed
	subParty, _ := NewParty(tenantID, "prt_sub_temp", "Sub Contractor", PartyTypeSubcontractor)
	_, err = NewNestedSubcontractorParticipation(
		closedPP,
		subParty,
		"ptp_sub_under_closed",
		"ste_temporal_01",
		"usr_sponsor_sub",
		ParticipationRoleSubcontractorLead,
		midWindow,
		t1,
	)
	if !errors.Is(err, ErrParentClosed) {
		t.Errorf("expected ErrParentClosed when nesting under closed parent, got %v", err)
	}

	// Negative control: Close on archived participation fails with ErrEntityArchived
	archivedPP := pp.Archive()
	_, _, err = CloseParticipation(archivedPP, "usr_admin_actor", "closing archived")
	if !errors.Is(err, ErrEntityArchived) {
		t.Errorf("expected ErrEntityArchived when closing archived participation, got %v", err)
	}
}

// TestQualification_InternalCompanyAndAdminEscalationDenials verifies that contractors and
// subcontractors are strictly prohibited from holding or escalating into internal corporate,
// company-level, business-unit, or tenant administrative roles.
func TestQualification_InternalCompanyAndAdminEscalationDenials(t *testing.T) {
	tenantID := "ten_qual_synth_esc_01"
	party, err := NewParty(tenantID, "prt_esc_01", "Third Party Compliance Co", PartyTypeContractor)
	if err != nil {
		t.Fatalf("unexpected NewParty error: %v", err)
	}

	baseTime := time.Date(2026, 9, 5, 9, 0, 0, 0, time.UTC)
	from := baseTime
	to := baseTime.Add(30 * 24 * time.Hour)

	// 1. Exhaustive matrix of forbidden administrative escalation keywords
	escalationRoles := []ParticipationRole{
		"ADMIN",
		"TENANT_ADMIN",
		"SUPER_ADMIN",
		"SUPERUSER",
		"COMPANY_ADMIN",
		"COMPANY_MANAGER",
		"BUSINESS_UNIT_LEAD",
		"BUSINESS_UNIT_ADMIN",
		"TENANT_OWNER",
		"site_safety_admin",
		"company_director",
		"Super_Inspector",
	}

	for _, role := range escalationRoles {
		// A. Direct function assertion
		if err := AssertNoInternalAuthority(role); !errors.Is(err, ErrElevationForbidden) {
			t.Errorf("role %q: expected ErrElevationForbidden from AssertNoInternalAuthority, got %v", role, err)
		}

		// B. Rejection at primary participation construction
		_, err := NewProjectParticipation(party, "ptp_esc_test", "prj_esc_01", "ste_01", "usr_sponsor_01", role, from, to)
		if !errors.Is(err, ErrElevationForbidden) {
			t.Errorf("role %q: expected ErrElevationForbidden from NewProjectParticipation, got %v", role, err)
		}
	}

	// 2. Approved non-administrative contractor roles must succeed
	approvedRoles := []ParticipationRole{
		ParticipationRoleContractorWorker,
		ParticipationRoleSiteSafetyLead,
		ParticipationRoleClientAuditor,
		ParticipationRoleConsultant,
		ParticipationRoleSubcontractorLead,
	}

	for _, role := range approvedRoles {
		if err := AssertNoInternalAuthority(role); err != nil {
			t.Errorf("approved role %q: unexpected AssertNoInternalAuthority error: %v", role, err)
		}

		pp, err := NewProjectParticipation(party, fmt.Sprintf("ptp_ok_%s", role), "prj_esc_01", "ste_01", "usr_sponsor_01", role, from, to)
		if err != nil {
			t.Errorf("approved role %q: unexpected NewProjectParticipation error: %v", role, err)
		}
		if pp.Role() != role {
			t.Errorf("approved role %q: role mismatch, got %q", role, pp.Role())
		}
	}

	// 3. Subcontractor constructor escalation rejection
	validParent, err := NewProjectParticipation(party, "ptp_valid_parent", "prj_esc_01", "ste_01", "usr_sponsor_01", ParticipationRoleContractorWorker, from, to)
	if err != nil {
		t.Fatalf("unexpected NewProjectParticipation: %v", err)
	}

	subParty, _ := NewParty(tenantID, "prt_sub_esc", "Subcontractor Esc", PartyTypeSubcontractor)
	for _, role := range escalationRoles {
		_, err := NewNestedSubcontractorParticipation(validParent, subParty, "ptp_sub_esc_fail", "ste_01", "usr_sponsor_01", role, from, to)
		if !errors.Is(err, ErrElevationForbidden) {
			t.Errorf("subcontractor role %q: expected ErrElevationForbidden, got %v", role, err)
		}
	}

	// 4. Sponsor ID identity verification: must be internal user identity (usr_*), never contractor ID
	invalidSponsors := []string{
		"prt_contractor_01",
		"cmp_company_01",
		"ste_site_01",
		"prj_project_01",
		"bnu_bu_01",
		"external_party_id",
		"",
		"   ",
	}

	for _, badSponsor := range invalidSponsors {
		err := ValidateSponsorID(badSponsor)
		if strings.TrimSpace(badSponsor) == "" {
			if !errors.Is(err, ErrBlankSponsorID) {
				t.Errorf("sponsor %q: expected ErrBlankSponsorID, got %v", badSponsor, err)
			}
		} else {
			if !errors.Is(err, ErrInvalidSponsorID) {
				t.Errorf("sponsor %q: expected ErrInvalidSponsorID, got %v", badSponsor, err)
			}
		}
	}

	// Valid sponsors must begin with approved user prefixes
	validSponsors := []string{
		"usr_manager_01",
		"usr-lead-engineer",
		"user-compliance-officer",
	}
	for _, goodSponsor := range validSponsors {
		if err := ValidateSponsorID(goodSponsor); err != nil {
			t.Errorf("valid sponsor %q: unexpected ValidateSponsorID error: %v", goodSponsor, err)
		}
	}
}

// TestQualification_ProjectClosureCascadeAndDenials verifies that project closure cascades
// strictly to in-scope contractor and subcontractor participations while preserving
// out-of-scope entities, and operational checks fail closed with ErrProjectClosedCascade.
func TestQualification_ProjectClosureCascadeAndDenials(t *testing.T) {
	tenantA := "ten_qual_synth_casc_a"
	tenantB := "ten_qual_synth_casc_b"

	companyA, err := NewCompany(tenantA, "cmp_casc_01", "Cascade Energy Corp")
	if err != nil {
		t.Fatalf("unexpected NewCompany: %v", err)
	}

	projA, err := NewProject(companyA, "prj_casc_alpha", "Project Alpha Site Build")
	if err != nil {
		t.Fatalf("unexpected NewProject: %v", err)
	}

	projB, err := NewProject(companyA, "prj_casc_beta", "Project Beta Expansion")
	if err != nil {
		t.Fatalf("unexpected NewProject: %v", err)
	}

	companyB, err := NewCompany(tenantB, "cmp_casc_b", "Foreign Tenant Corp")
	if err != nil {
		t.Fatalf("unexpected NewCompany: %v", err)
	}
	projForeign, err := NewProject(companyB, "prj_casc_foreign", "Foreign Project")
	if err != nil {
		t.Fatalf("unexpected NewProject: %v", err)
	}

	baseTime := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	from := baseTime
	to := baseTime.Add(60 * 24 * time.Hour)

	partyPrimeA, _ := NewParty(tenantA, "prt_prime_casc_a", "Prime A", PartyTypeContractor)
	partySubA, _ := NewParty(tenantA, "prt_sub_casc_a", "Sub A", PartyTypeSubcontractor)
	partyArchivedA, _ := NewParty(tenantA, "prt_arch_casc_a", "Archived A", PartyTypeContractor)
	partyPrimeB, _ := NewParty(tenantA, "prt_prime_casc_b", "Prime B", PartyTypeContractor)
	partyForeign, _ := NewParty(tenantB, "prt_prime_casc_f", "Foreign Prime", PartyTypeContractor)

	// Participations under projA
	ppPrimeA, err := NewProjectParticipation(partyPrimeA, "ptp_casc_prime_a", projA.ProjectID(), "", "usr_sponsor_a", ParticipationRoleContractorWorker, from, to)
	if err != nil {
		t.Fatalf("unexpected NewProjectParticipation: %v", err)
	}
	ppSubA, err := NewNestedSubcontractorParticipation(ppPrimeA, partySubA, "ptp_casc_sub_a", "", "usr_sponsor_sub_a", ParticipationRoleSubcontractorLead, from, to)
	if err != nil {
		t.Fatalf("unexpected NewNestedSubcontractorParticipation: %v", err)
	}
	ppArchivedA, err := NewProjectParticipation(partyArchivedA, "ptp_casc_arch_a", projA.ProjectID(), "", "usr_sponsor_a", ParticipationRoleContractorWorker, from, to)
	if err != nil {
		t.Fatalf("unexpected NewProjectParticipation: %v", err)
	}
	ppArchivedA = ppArchivedA.Archive() // Pre-archived

	// Participation under projB (unrelated project)
	ppPrimeB, err := NewProjectParticipation(partyPrimeB, "ptp_casc_prime_b", projB.ProjectID(), "", "usr_sponsor_b", ParticipationRoleContractorWorker, from, to)
	if err != nil {
		t.Fatalf("unexpected NewProjectParticipation: %v", err)
	}

	// Participation under foreign tenant
	ppForeign, err := NewProjectParticipation(partyForeign, "ptp_casc_foreign", projForeign.ProjectID(), "", "usr_sponsor_f", ParticipationRoleContractorWorker, from, to)
	if err != nil {
		t.Fatalf("unexpected NewProjectParticipation: %v", err)
	}

	// 1. Negative control: CascadeProjectClosure on an active (non-closed) project MUST fail
	_, _, err = CascadeProjectClosure(projA, []ProjectParticipation{ppPrimeA}, "usr_auditor", "premature cascade")
	if err == nil || !strings.Contains(err.Error(), "cannot cascade closure from non-closed project") {
		t.Fatalf("expected error cascading from non-closed project, got %v", err)
	}

	// 2. Close projA in memory
	closedProjA, err := projA.Close()
	if err != nil {
		t.Fatalf("unexpected projA.Close: %v", err)
	}
	if !closedProjA.IsClosed() {
		t.Fatalf("expected closedProjA.IsClosed() == true")
	}

	// 3. Execute CascadeProjectClosure on the closed project
	inputParticipations := []ProjectParticipation{ppPrimeA, ppSubA, ppArchivedA, ppPrimeB, ppForeign}
	updatedList, auditRecords, err := CascadeProjectClosure(closedProjA, inputParticipations, "usr_closure_agent", "Project delivery accepted and closed")
	if err != nil {
		t.Fatalf("unexpected CascadeProjectClosure error: %v", err)
	}

	// Verify updatedList length matches input
	if len(updatedList) != len(inputParticipations) {
		t.Fatalf("expected updatedList length %d, got %d", len(inputParticipations), len(updatedList))
	}

	// Map updated participations by ID
	updatedMap := make(map[string]ProjectParticipation)
	for _, p := range updatedList {
		updatedMap[p.ParticipationID()] = p
	}

	// A. ppPrimeA must now be StateClosed
	if p, ok := updatedMap["ptp_casc_prime_a"]; !ok || p.State() != StateClosed {
		t.Errorf("expected ptp_casc_prime_a to be StateClosed, got %v (found=%v)", p.State(), ok)
	}

	// B. ppSubA must now be StateClosed
	if p, ok := updatedMap["ptp_casc_sub_a"]; !ok || p.State() != StateClosed {
		t.Errorf("expected ptp_casc_sub_a to be StateClosed, got %v (found=%v)", p.State(), ok)
	}

	// C. ppArchivedA must remain StateArchived (cascade does not resurrect or overwrite archived)
	if p, ok := updatedMap["ptp_casc_arch_a"]; !ok || p.State() != StateArchived {
		t.Errorf("expected ptp_casc_arch_a to retain StateArchived, got %v (found=%v)", p.State(), ok)
	}

	// D. ppPrimeB must remain StateActive (different project)
	if p, ok := updatedMap["ptp_casc_prime_b"]; !ok || p.State() != StateActive {
		t.Errorf("expected ptp_casc_prime_b to remain StateActive, got %v (found=%v)", p.State(), ok)
	}

	// E. ppForeign must remain StateActive (different tenant)
	if p, ok := updatedMap["ptp_casc_foreign"]; !ok || p.State() != StateActive {
		t.Errorf("expected ptp_casc_foreign to remain StateActive, got %v (found=%v)", p.State(), ok)
	}

	// Audit records: exactly 2 transitions (ppPrimeA and ppSubA)
	if len(auditRecords) != 2 {
		t.Fatalf("expected exactly 2 audit records for cascade, got %d", len(auditRecords))
	}
	for _, rec := range auditRecords {
		if rec.Transition != "PROJECT_CLOSURE_CASCADE" {
			t.Errorf("expected transition PROJECT_CLOSURE_CASCADE, got %s", rec.Transition)
		}
		if rec.NewState != StateClosed {
			t.Errorf("expected new_state CLOSED, got %s", rec.NewState)
		}
		if rec.ProjectID != projA.ProjectID() {
			t.Errorf("expected project ID %s, got %s", projA.ProjectID(), rec.ProjectID)
		}
	}

	// 4. ValidateParticipationAgainstProject operational negative controls
	// Active participation against closed project -> MUST fail with ErrProjectClosedCascade
	if err := ValidateParticipationAgainstProject(ppPrimeA, closedProjA, baseTime.Add(1*time.Hour)); !errors.Is(err, ErrProjectClosedCascade) {
		t.Errorf("expected ErrProjectClosedCascade against closed project, got %v", err)
	}

	// Archived project negative control -> MUST fail with ErrParentArchived
	archivedProjA := projA.Archive()
	if err := ValidateParticipationAgainstProject(ppPrimeA, archivedProjA, baseTime.Add(1*time.Hour)); !errors.Is(err, ErrParentArchived) {
		t.Errorf("expected ErrParentArchived against archived project, got %v", err)
	}

	// Cross-tenant mismatch -> MUST fail with ErrCrossTenantLinkage
	if err := ValidateParticipationAgainstProject(ppForeign, projA, baseTime.Add(1*time.Hour)); !errors.Is(err, ErrCrossTenantLinkage) {
		t.Errorf("expected ErrCrossTenantLinkage for mismatched tenant, got %v", err)
	}

	// Project ID scope mismatch -> MUST fail with ErrScopeMismatch
	if err := ValidateParticipationAgainstProject(ppPrimeA, projB, baseTime.Add(1*time.Hour)); !errors.Is(err, ErrScopeMismatch) {
		t.Errorf("expected ErrScopeMismatch for mismatched project ID, got %v", err)
	}

	// Active matching project and active participation -> succeeds
	if err := ValidateParticipationAgainstProject(ppPrimeA, projA, baseTime.Add(1*time.Hour)); err != nil {
		t.Errorf("expected nil for valid active project and participation, got %v", err)
	}
}

// TestQualification_CrossTenantHistoricalAccessIsolation verifies that the in-memory
// append-only PartyLifecycleLedger strictly enforces tenant boundary isolation on all
// party, participation, and sponsor reassignment historical audit queries.
func TestQualification_CrossTenantHistoricalAccessIsolation(t *testing.T) {
	ledger := NewPartyLifecycleLedger()

	tenantAlpha := "ten_qual_synth_hist_alpha"
	tenantBravo := "ten_qual_synth_hist_bravo"

	baseTime := time.Date(2026, 9, 5, 14, 0, 0, 0, time.UTC)

	// 1. Populate audit events for Tenant Alpha
	partyRecAlpha := HistoricalPartyLifecycleRecord{
		RecordID:      "hprt_alpha_01",
		TenantID:      tenantAlpha,
		PartyID:       "prt_alpha_01",
		PreviousState: StateActive,
		NewState:      StateArchived,
		Transition:    "PARTY_DEACTIVATE",
		ActorSubject:  "usr_alpha_actor",
		Reason:        "Alpha party contract end",
		RecordedAt:    baseTime,
	}
	if err := ledger.AppendPartyRecord(partyRecAlpha); err != nil {
		t.Fatalf("unexpected AppendPartyRecord error: %v", err)
	}

	partRecAlpha := HistoricalParticipationLifecycleRecord{
		RecordID:        "hptp_alpha_01",
		TenantID:        tenantAlpha,
		ParticipationID: "ptp_alpha_01",
		PartyID:         "prt_alpha_01",
		ProjectID:       "prj_alpha_01",
		SponsorID:       "usr_alpha_sponsor_1",
		PreviousState:   StateActive,
		NewState:        StateClosed,
		Transition:      "PARTICIPATION_CLOSE",
		ActorSubject:    "usr_alpha_actor",
		Reason:          "Alpha milestone closed",
		RecordedAt:      baseTime.Add(1 * time.Hour),
	}
	if err := ledger.AppendParticipationRecord(partRecAlpha); err != nil {
		t.Fatalf("unexpected AppendParticipationRecord error: %v", err)
	}

	sponsorRecAlpha := SponsorReassignmentRecord{
		RecordID:        "sps_alpha_01",
		TenantID:        tenantAlpha,
		ParticipationID: "ptp_alpha_01",
		PriorSponsorID:  "usr_alpha_sponsor_1",
		NewSponsorID:    "usr_alpha_sponsor_2",
		ActorSubject:    "usr_alpha_actor",
		Reason:          "Sponsor manager rotation",
		ReassignedAt:    baseTime.Add(2 * time.Hour),
	}
	if err := ledger.AppendSponsorReassignment(sponsorRecAlpha); err != nil {
		t.Fatalf("unexpected AppendSponsorReassignment error: %v", err)
	}

	// 2. Populate audit events for Tenant Bravo
	partyRecBravo := HistoricalPartyLifecycleRecord{
		RecordID:      "hprt_bravo_01",
		TenantID:      tenantBravo,
		PartyID:       "prt_bravo_01",
		PreviousState: StateActive,
		NewState:      StateArchived,
		Transition:    "PARTY_DEACTIVATE",
		ActorSubject:  "usr_bravo_actor",
		Reason:        "Bravo party decommissioned",
		RecordedAt:    baseTime,
	}
	if err := ledger.AppendPartyRecord(partyRecBravo); err != nil {
		t.Fatalf("unexpected AppendPartyRecord error: %v", err)
	}

	partRecBravo := HistoricalParticipationLifecycleRecord{
		RecordID:        "hptp_bravo_01",
		TenantID:        tenantBravo,
		ParticipationID: "ptp_bravo_01",
		PartyID:         "prt_bravo_01",
		ProjectID:       "prj_bravo_01",
		SponsorID:       "usr_bravo_sponsor_1",
		PreviousState:   StateActive,
		NewState:        StateClosed,
		Transition:      "PARTICIPATION_CLOSE",
		ActorSubject:    "usr_bravo_actor",
		Reason:          "Bravo project done",
		RecordedAt:      baseTime.Add(1 * time.Hour),
	}
	if err := ledger.AppendParticipationRecord(partRecBravo); err != nil {
		t.Fatalf("unexpected AppendParticipationRecord error: %v", err)
	}

	// 3. Query verification: Tenant Alpha queries its own records
	partyAlphaHistory, err := ledger.GetPartyHistory(tenantAlpha, "prt_alpha_01")
	if err != nil {
		t.Fatalf("unexpected GetPartyHistory error: %v", err)
	}
	if len(partyAlphaHistory) != 1 || partyAlphaHistory[0].RecordID != "hprt_alpha_01" {
		t.Errorf("expected 1 record hprt_alpha_01, got %+v", partyAlphaHistory)
	}

	partAlphaHistory, err := ledger.GetParticipationHistory(tenantAlpha, "ptp_alpha_01")
	if err != nil {
		t.Fatalf("unexpected GetParticipationHistory error: %v", err)
	}
	if len(partAlphaHistory) != 1 || partAlphaHistory[0].RecordID != "hptp_alpha_01" {
		t.Errorf("expected 1 record hptp_alpha_01, got %+v", partAlphaHistory)
	}

	spsAlphaHistory, err := ledger.GetSponsorHistory(tenantAlpha, "ptp_alpha_01")
	if err != nil {
		t.Fatalf("unexpected GetSponsorHistory error: %v", err)
	}
	if len(spsAlphaHistory) != 1 || spsAlphaHistory[0].PriorSponsorID != "usr_alpha_sponsor_1" || spsAlphaHistory[0].NewSponsorID != "usr_alpha_sponsor_2" {
		t.Errorf("expected sponsor reassignment record preserving attribution, got %+v", spsAlphaHistory)
	}

	// 4. Negative controls: Cross-tenant query isolation
	// Tenant Bravo queries Alpha's party ID -> MUST return 0 records (no cross-tenant leakage)
	leakedParty, err := ledger.GetPartyHistory(tenantBravo, "prt_alpha_01")
	if err != nil {
		t.Fatalf("unexpected error on cross-tenant query: %v", err)
	}
	if len(leakedParty) != 0 {
		t.Fatalf("SECURITY VIOLATION: Tenant Bravo accessed Tenant Alpha's party history: %+v", leakedParty)
	}

	// Tenant Bravo queries Alpha's participation ID -> MUST return 0 records
	leakedPart, err := ledger.GetParticipationHistory(tenantBravo, "ptp_alpha_01")
	if err != nil {
		t.Fatalf("unexpected error on cross-tenant query: %v", err)
	}
	if len(leakedPart) != 0 {
		t.Fatalf("SECURITY VIOLATION: Tenant Bravo accessed Tenant Alpha's participation history: %+v", leakedPart)
	}

	// Tenant Bravo queries Alpha's sponsor history -> MUST return 0 records
	leakedSponsor, err := ledger.GetSponsorHistory(tenantBravo, "ptp_alpha_01")
	if err != nil {
		t.Fatalf("unexpected error on cross-tenant query: %v", err)
	}
	if len(leakedSponsor) != 0 {
		t.Fatalf("SECURITY VIOLATION: Tenant Bravo accessed Tenant Alpha's sponsor history: %+v", leakedSponsor)
	}

	// Tenant Alpha queries Bravo's party ID -> MUST return 0 records
	leakedBravoParty, err := ledger.GetPartyHistory(tenantAlpha, "prt_bravo_01")
	if err != nil {
		t.Fatalf("unexpected error on cross-tenant query: %v", err)
	}
	if len(leakedBravoParty) != 0 {
		t.Fatalf("SECURITY VIOLATION: Tenant Alpha accessed Tenant Bravo's party history: %+v", leakedBravoParty)
	}

	// 5. Query validation controls: Blank or whitespace tenant / ID queries must fail closed
	if _, err := ledger.GetPartyHistory("", "prt_01"); !errors.Is(err, ErrBlankTenantID) {
		t.Errorf("expected ErrBlankTenantID for empty tenant, got %v", err)
	}
	if _, err := ledger.GetPartyHistory("   ", "prt_01"); !errors.Is(err, ErrBlankTenantID) {
		t.Errorf("expected ErrBlankTenantID for whitespace tenant, got %v", err)
	}
	if _, err := ledger.GetPartyHistory(tenantAlpha, ""); !errors.Is(err, ErrBlankID) {
		t.Errorf("expected ErrBlankID for empty party ID, got %v", err)
	}
	if _, err := ledger.GetPartyHistory(tenantAlpha, "   "); !errors.Is(err, ErrBlankID) {
		t.Errorf("expected ErrBlankID for whitespace party ID, got %v", err)
	}

	if _, err := ledger.GetParticipationHistory("", "ptp_01"); !errors.Is(err, ErrBlankTenantID) {
		t.Errorf("expected ErrBlankTenantID for empty tenant in participation query, got %v", err)
	}
	if _, err := ledger.GetParticipationHistory(tenantAlpha, ""); !errors.Is(err, ErrBlankID) {
		t.Errorf("expected ErrBlankID for empty participation ID, got %v", err)
	}

	if _, err := ledger.GetSponsorHistory("", "ptp_01"); !errors.Is(err, ErrBlankTenantID) {
		t.Errorf("expected ErrBlankTenantID for empty tenant in sponsor query, got %v", err)
	}
	if _, err := ledger.GetSponsorHistory(tenantAlpha, ""); !errors.Is(err, ErrBlankID) {
		t.Errorf("expected ErrBlankID for empty participation ID in sponsor query, got %v", err)
	}

	// 6. Append-time validation controls
	if err := ledger.AppendPartyRecord(HistoricalPartyLifecycleRecord{TenantID: "", PartyID: "prt_01"}); !errors.Is(err, ErrBlankID) {
		t.Errorf("expected ErrBlankID for blank tenant in AppendPartyRecord, got %v", err)
	}
	if err := ledger.AppendParticipationRecord(HistoricalParticipationLifecycleRecord{TenantID: "", ParticipationID: "ptp_01"}); !errors.Is(err, ErrBlankID) {
		t.Errorf("expected ErrBlankID for blank tenant in AppendParticipationRecord, got %v", err)
	}
	if err := ledger.AppendSponsorReassignment(SponsorReassignmentRecord{TenantID: "", ParticipationID: "ptp_01"}); !errors.Is(err, ErrBlankID) {
		t.Errorf("expected ErrBlankID for blank tenant in AppendSponsorReassignment, got %v", err)
	}

	// 7. Immutability & Zero hard deletion: Prior records are permanently preserved
	sponsorRecAlpha2 := SponsorReassignmentRecord{
		RecordID:        "sps_alpha_02",
		TenantID:        tenantAlpha,
		ParticipationID: "ptp_alpha_01",
		PriorSponsorID:  "usr_alpha_sponsor_2",
		NewSponsorID:    "usr_alpha_sponsor_3",
		ActorSubject:    "usr_alpha_actor",
		Reason:          "Second sponsor rotation",
		ReassignedAt:    baseTime.Add(3 * time.Hour),
	}
	if err := ledger.AppendSponsorReassignment(sponsorRecAlpha2); err != nil {
		t.Fatalf("unexpected AppendSponsorReassignment error: %v", err)
	}

	history, err := ledger.GetSponsorHistory(tenantAlpha, "ptp_alpha_01")
	if err != nil {
		t.Fatalf("unexpected GetSponsorHistory error: %v", err)
	}
	if len(history) != 2 {
		t.Fatalf("expected 2 historical sponsor records preserved without deletion, got %d", len(history))
	}
	if history[0].PriorSponsorID != "usr_alpha_sponsor_1" || history[0].NewSponsorID != "usr_alpha_sponsor_2" {
		t.Errorf("first sponsor record corrupted: %+v", history[0])
	}
	if history[1].PriorSponsorID != "usr_alpha_sponsor_2" || history[1].NewSponsorID != "usr_alpha_sponsor_3" {
		t.Errorf("second sponsor record corrupted: %+v", history[1])
	}
}
