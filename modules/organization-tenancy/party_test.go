package orgtenancy

import (
	"errors"
	"testing"
	"time"
)

func TestParty_CreationAndAccessors(t *testing.T) {
	tenantID := "ten_01j9876543210zyxwvutsrqpon"
	partyID := "prt_01j9876543210zyxwvutsrqpon"
	name := "Siam Industrial Safety Consultants Ltd"
	partyType := PartyTypeContractor

	party, err := NewParty(tenantID, partyID, name, partyType)
	if err != nil {
		t.Fatalf("unexpected NewParty error: %v", err)
	}

	if party.TenantID() != tenantID {
		t.Errorf("expected tenant %q, got %q", tenantID, party.TenantID())
	}
	if party.PartyID() != partyID {
		t.Errorf("expected partyID %q, got %q", partyID, party.PartyID())
	}
	if party.Name() != name {
		t.Errorf("expected name %q, got %q", name, party.Name())
	}
	if party.PartyType() != partyType {
		t.Errorf("expected partyType %q, got %q", partyType, party.PartyType())
	}
	if party.State() != StateActive || !party.IsActive() {
		t.Errorf("expected party to be active")
	}
}

func TestParty_Rejections(t *testing.T) {
	// Blank tenant
	if _, err := NewParty("", "prt_01", "Name", PartyTypeClient); !errors.Is(err, ErrBlankTenantID) {
		t.Errorf("expected ErrBlankTenantID, got %v", err)
	}
	if _, err := NewParty("   ", "prt_01", "Name", PartyTypeClient); !errors.Is(err, ErrBlankTenantID) {
		t.Errorf("expected ErrBlankTenantID for whitespace, got %v", err)
	}

	// Blank party ID
	if _, err := NewParty("ten_01", "", "Name", PartyTypeClient); !errors.Is(err, ErrBlankID) {
		t.Errorf("expected ErrBlankID, got %v", err)
	}
	if _, err := NewParty("ten_01", "   ", "Name", PartyTypeClient); !errors.Is(err, ErrBlankID) {
		t.Errorf("expected ErrBlankID for whitespace, got %v", err)
	}

	// Blank name
	if _, err := NewParty("ten_01", "prt_01", "", PartyTypeClient); !errors.Is(err, ErrBlankName) {
		t.Errorf("expected ErrBlankName, got %v", err)
	}
	if _, err := NewParty("ten_01", "prt_01", "   ", PartyTypeClient); !errors.Is(err, ErrBlankName) {
		t.Errorf("expected ErrBlankName for whitespace, got %v", err)
	}

	// Invalid PartyType
	if _, err := NewParty("ten_01", "prt_01", "Name", PartyType("UNKNOWN")); !errors.Is(err, ErrInvalidPartyType) {
		t.Errorf("expected ErrInvalidPartyType, got %v", err)
	}
	if _, err := NewParty("ten_01", "prt_01", "Name", PartyType("")); !errors.Is(err, ErrInvalidPartyType) {
		t.Errorf("expected ErrInvalidPartyType for empty type, got %v", err)
	}
}

func TestParty_ArchiveLifecycle(t *testing.T) {
	party, err := NewParty("ten_01", "prt_01", "Apex Safety", PartyTypeSubcontractor)
	if err != nil {
		t.Fatalf("unexpected NewParty error: %v", err)
	}

	archived := party.Archive()
	if archived.IsActive() || archived.State() != StateArchived {
		t.Errorf("expected party to be archived")
	}
	// Immutable value semantics: original remains active
	if !party.IsActive() {
		t.Errorf("original party must remain unmodified")
	}
	// Historical attribution preserved
	if archived.PartyID() != "prt_01" || archived.Name() != "Apex Safety" {
		t.Errorf("archived party must preserve identity and name")
	}
}

func TestParty_CrossTenantDenial(t *testing.T) {
	party, _ := NewParty("tenant-alpha", "prt_01", "Contractor Alpha", PartyTypeContractor)

	claimsB := &TrustedClaims{
		Subject:         "usr-bravo",
		TenantID:        "tenant-bravo",
		IsAuthenticated: true,
	}
	ctxB, err := DeriveTenantContext(claimsB, nil)
	if err != nil {
		t.Fatalf("unexpected DeriveTenantContext: %v", err)
	}

	if err := party.ValidateScope(ctxB); !errors.Is(err, ErrTenantMismatch) {
		t.Errorf("expected ErrTenantMismatch for foreign tenant context, got %v", err)
	}
}

func TestProjectParticipation_CreationAndScopeValidation(t *testing.T) {
	party, _ := NewParty("ten_01", "prt_01", "Inspect Pro", PartyTypeAuditor)

	baseTime := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	validFrom := baseTime
	validTo := baseTime.Add(60 * 24 * time.Hour)

	pp, err := NewProjectParticipation(party, "ptp_01", "prj_01", "ste_01", "usr_sponsor_mgr", ParticipationRoleClientAuditor, validFrom, validTo)
	if err != nil {
		t.Fatalf("unexpected NewProjectParticipation error: %v", err)
	}

	if pp.ParticipationID() != "ptp_01" || pp.TenantID() != "ten_01" || pp.PartyID() != "prt_01" ||
		pp.ProjectID() != "prj_01" || pp.SiteID() != "ste_01" || pp.SponsorID() != "usr_sponsor_mgr" ||
		pp.Role() != ParticipationRoleClientAuditor {
		t.Errorf("participation fields mismatch: %+v", pp)
	}
	if pp.NestingDepth() != 0 || pp.IsSubcontractor() || pp.ParentParticipationID() != "" {
		t.Errorf("expected depth 0 primary participation")
	}
	if !pp.IsActive() || pp.State() != StateActive {
		t.Errorf("expected participation to be active")
	}

	// Scope validation at active time and matching bounds
	during := baseTime.Add(10 * 24 * time.Hour)
	if err := pp.ValidateScopeAt("prj_01", "ste_01", during); err != nil {
		t.Errorf("expected scope validation success, got %v", err)
	}
}

func TestProjectParticipation_Rejections(t *testing.T) {
	party, _ := NewParty("ten_01", "prt_01", "Inspect Pro", PartyTypeContractor)
	validFrom := time.Now()
	validTo := validFrom.Add(24 * time.Hour)

	// Blank participation ID
	if _, err := NewProjectParticipation(party, "", "prj_01", "ste_01", "usr_01", ParticipationRoleContractorWorker, validFrom, validTo); !errors.Is(err, ErrBlankID) {
		t.Errorf("expected ErrBlankID, got %v", err)
	}
	// Blank project ID
	if _, err := NewProjectParticipation(party, "ptp_01", "", "ste_01", "usr_01", ParticipationRoleContractorWorker, validFrom, validTo); !errors.Is(err, ErrBlankID) {
		t.Errorf("expected ErrBlankID, got %v", err)
	}
	// Blank sponsor ID
	if _, err := NewProjectParticipation(party, "ptp_01", "prj_01", "ste_01", "", ParticipationRoleContractorWorker, validFrom, validTo); !errors.Is(err, ErrBlankSponsorID) {
		t.Errorf("expected ErrBlankSponsorID, got %v", err)
	}
	if _, err := NewProjectParticipation(party, "ptp_01", "prj_01", "ste_01", "   ", ParticipationRoleContractorWorker, validFrom, validTo); !errors.Is(err, ErrBlankSponsorID) {
		t.Errorf("expected ErrBlankSponsorID for whitespace, got %v", err)
	}
	// Invalid sponsor ID (not an internal user)
	if _, err := NewProjectParticipation(party, "ptp_01", "prj_01", "ste_01", "ext_vendor_01", ParticipationRoleContractorWorker, validFrom, validTo); !errors.Is(err, ErrInvalidSponsorID) {
		t.Errorf("expected ErrInvalidSponsorID for non-user sponsor, got %v", err)
	}
	// Invalid role
	if _, err := NewProjectParticipation(party, "ptp_01", "prj_01", "ste_01", "usr_01", ParticipationRole("UNKNOWN_ROLE"), validFrom, validTo); !errors.Is(err, ErrInvalidParticipationRole) {
		t.Errorf("expected ErrInvalidParticipationRole, got %v", err)
	}
	// Invalid time window (validTo before or equal to validFrom)
	if _, err := NewProjectParticipation(party, "ptp_01", "prj_01", "ste_01", "usr_01", ParticipationRoleContractorWorker, validTo, validFrom); !errors.Is(err, ErrInvalidTimeWindow) {
		t.Errorf("expected ErrInvalidTimeWindow, got %v", err)
	}
	if _, err := NewProjectParticipation(party, "ptp_01", "prj_01", "ste_01", "usr_01", ParticipationRoleContractorWorker, validFrom, validFrom); !errors.Is(err, ErrInvalidTimeWindow) {
		t.Errorf("expected ErrInvalidTimeWindow for equal times, got %v", err)
	}
	// Uninitialized parent party
	if _, err := NewProjectParticipation(Party{}, "ptp_01", "prj_01", "ste_01", "usr_01", ParticipationRoleContractorWorker, validFrom, validTo); !errors.Is(err, ErrParentMismatch) {
		t.Errorf("expected ErrParentMismatch, got %v", err)
	}
	// Archived parent party
	archivedParty := party.Archive()
	if _, err := NewProjectParticipation(archivedParty, "ptp_01", "prj_01", "ste_01", "usr_01", ParticipationRoleContractorWorker, validFrom, validTo); !errors.Is(err, ErrPartyArchived) {
		t.Errorf("expected ErrPartyArchived, got %v", err)
	}
}

func TestProjectParticipation_NestedSubcontractorCreation(t *testing.T) {
	tenantID := "ten_01"
	primeParty, _ := NewParty(tenantID, "prt_prime_01", "Prime Builder Corp", PartyTypeContractor)
	subParty, _ := NewParty(tenantID, "prt_sub_01", "Specialized Piping Ltd", PartyTypeSubcontractor)

	baseTime := time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC)
	primeFrom := baseTime
	primeTo := baseTime.Add(90 * 24 * time.Hour)

	// 1. Prime contractor participation (depth 0)
	primePart, err := NewProjectParticipation(primeParty, "ptp_prime_01", "prj_terminal", "ste_01", "usr_sponsor_lead", ParticipationRoleSiteSafetyLead, primeFrom, primeTo)
	if err != nil {
		t.Fatalf("unexpected NewProjectParticipation error: %v", err)
	}

	// 2. Nested subcontractor participation (depth 1)
	subFrom := baseTime.Add(5 * 24 * time.Hour)
	subTo := baseTime.Add(30 * 24 * time.Hour)
	subPart, err := NewNestedSubcontractorParticipation(primePart, subParty, "ptp_sub_01", "ste_01", "usr_sponsor_field", ParticipationRoleSubcontractorLead, subFrom, subTo)
	if err != nil {
		t.Fatalf("unexpected NewNestedSubcontractorParticipation: %v", err)
	}

	// Assertions
	if subPart.ParentParticipationID() != primePart.ParticipationID() {
		t.Errorf("expected parent ID %q, got %q", primePart.ParticipationID(), subPart.ParentParticipationID())
	}
	if subPart.NestingDepth() != 1 || !subPart.IsSubcontractor() {
		t.Errorf("expected depth 1 subcontractor participation")
	}
	if subPart.ProjectID() != primePart.ProjectID() {
		t.Errorf("subcontractor must inherit parent project ID")
	}
	if subPart.SiteID() != "ste_01" {
		t.Errorf("site ID mismatch: %s", subPart.SiteID())
	}
	if subPart.SponsorID() != "usr_sponsor_field" {
		t.Errorf("sponsor ID mismatch: %s", subPart.SponsorID())
	}

	// Validate scope
	during := baseTime.Add(15 * 24 * time.Hour)
	if err := subPart.ValidateNestedScope(primePart, "prj_terminal", "ste_01", during); err != nil {
		t.Errorf("expected nested scope validation success, got %v", err)
	}
}

func TestProjectParticipation_NestingDepthExceeded(t *testing.T) {
	tenantID := "ten_01"
	primeParty, _ := NewParty(tenantID, "prt_prime", "Prime", PartyTypeContractor)
	subParty1, _ := NewParty(tenantID, "prt_sub1", "Sub1", PartyTypeSubcontractor)
	subParty2, _ := NewParty(tenantID, "prt_sub2", "Sub2", PartyTypeSubcontractor)

	from := time.Now().Add(-1 * time.Hour)
	to := time.Now().Add(48 * time.Hour)

	prime, _ := NewProjectParticipation(primeParty, "ptp_prime", "prj_01", "ste_01", "usr_sponsor", ParticipationRoleSiteSafetyLead, from, to)
	sub1, err := NewNestedSubcontractorParticipation(prime, subParty1, "ptp_sub1", "ste_01", "usr_sponsor", ParticipationRoleSubcontractorLead, from, to)
	if err != nil {
		t.Fatalf("unexpected sub1 creation: %v", err)
	}

	// Attempting to nest sub2 under sub1 (depth 2) must fail closed with ErrNestingDepthExceeded
	_, err = NewNestedSubcontractorParticipation(sub1, subParty2, "ptp_sub2", "ste_01", "usr_sponsor", ParticipationRoleContractorWorker, from, to)
	if !errors.Is(err, ErrNestingDepthExceeded) {
		t.Errorf("expected ErrNestingDepthExceeded for sub-subcontractor, got %v", err)
	}
}

func TestProjectParticipation_NoElevationToInternalAuthority(t *testing.T) {
	party, _ := NewParty("ten_01", "prt_01", "Contractor", PartyTypeContractor)
	from := time.Now()
	to := from.Add(24 * time.Hour)

	// Attempting to assign administrative or internal roles
	forbiddenRoles := []ParticipationRole{
		ParticipationRole("ADMIN"),
		ParticipationRole("SUPER_ADMIN"),
		ParticipationRole("COMPANY_ADMIN"),
		ParticipationRole("BUSINESS_UNIT_LEAD"),
		ParticipationRole("TENANT_OWNER"),
	}

	for _, role := range forbiddenRoles {
		_, err := NewProjectParticipation(party, "ptp_01", "prj_01", "ste_01", "usr_01", role, from, to)
		if !errors.Is(err, ErrElevationForbidden) {
			t.Errorf("expected ErrElevationForbidden for role %q, got %v", role, err)
		}
	}
}

func TestProjectParticipation_TemporalContainmentUnderParent(t *testing.T) {
	tenantID := "ten_01"
	primeParty, _ := NewParty(tenantID, "prt_prime", "Prime", PartyTypeContractor)
	subParty, _ := NewParty(tenantID, "prt_sub", "Sub", PartyTypeSubcontractor)

	baseTime := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	primeFrom := baseTime
	primeTo := baseTime.Add(30 * 24 * time.Hour)
	prime, _ := NewProjectParticipation(primeParty, "ptp_prime", "prj_01", "ste_01", "usr_mgr", ParticipationRoleSiteSafetyLead, primeFrom, primeTo)

	// 1. Subcontractor starts before parent
	subStartsEarly := baseTime.Add(-1 * time.Hour)
	subEndsInside := baseTime.Add(10 * 24 * time.Hour)
	_, err := NewNestedSubcontractorParticipation(prime, subParty, "ptp_sub_early", "ste_01", "usr_mgr", ParticipationRoleContractorWorker, subStartsEarly, subEndsInside)
	if !errors.Is(err, ErrValidityWindowExceedsParent) {
		t.Errorf("expected ErrValidityWindowExceedsParent for early start, got %v", err)
	}

	// 2. Subcontractor ends after parent
	subStartsInside := baseTime.Add(1 * time.Hour)
	subEndsLate := primeTo.Add(1 * time.Hour)
	_, err = NewNestedSubcontractorParticipation(prime, subParty, "ptp_sub_late", "ste_01", "usr_mgr", ParticipationRoleContractorWorker, subStartsInside, subEndsLate)
	if !errors.Is(err, ErrValidityWindowExceedsParent) {
		t.Errorf("expected ErrValidityWindowExceedsParent for late end, got %v", err)
	}
}

func TestProjectParticipation_ParentStatusCascade(t *testing.T) {
	tenantID := "ten_01"
	primeParty, _ := NewParty(tenantID, "prt_prime", "Prime", PartyTypeContractor)
	subParty, _ := NewParty(tenantID, "prt_sub", "Sub", PartyTypeSubcontractor)

	from := time.Now().Add(-1 * time.Hour)
	to := time.Now().Add(24 * time.Hour)
	prime, _ := NewProjectParticipation(primeParty, "ptp_prime", "prj_01", "ste_01", "usr_mgr", ParticipationRoleSiteSafetyLead, from, to)

	// 1. Closed parent contractor
	closedPrime, _ := prime.Close()
	_, err := NewNestedSubcontractorParticipation(closedPrime, subParty, "ptp_sub_closed", "ste_01", "usr_mgr", ParticipationRoleContractorWorker, from, to)
	if !errors.Is(err, ErrParentClosed) {
		t.Errorf("expected ErrParentClosed when parent is closed, got %v", err)
	}

	// 2. Archived parent contractor
	archivedPrime := prime.Archive()
	_, err = NewNestedSubcontractorParticipation(archivedPrime, subParty, "ptp_sub_arch", "ste_01", "usr_mgr", ParticipationRoleContractorWorker, from, to)
	if !errors.Is(err, ErrParentNotActive) {
		t.Errorf("expected ErrParentNotActive when parent is archived, got %v", err)
	}

	// 3. Runtime cascade in ValidateNestedScope
	sub, _ := NewNestedSubcontractorParticipation(prime, subParty, "ptp_sub_valid", "ste_01", "usr_mgr", ParticipationRoleContractorWorker, from, to)
	now := time.Now()
	if err := sub.ValidateNestedScope(closedPrime, "prj_01", "ste_01", now); !errors.Is(err, ErrParentNotActive) {
		t.Errorf("expected ErrParentNotActive at runtime when parent closed, got %v", err)
	}
}

func TestProjectParticipation_SiblingAndCrossProjectDenial(t *testing.T) {
	tenantID := "ten_01"
	partyA, _ := NewParty(tenantID, "prt_a", "Contractor A", PartyTypeContractor)
	partyB, _ := NewParty(tenantID, "prt_b", "Contractor B", PartyTypeContractor)

	from := time.Now().Add(-1 * time.Hour)
	to := time.Now().Add(24 * time.Hour)

	ppA, _ := NewProjectParticipation(partyA, "ptp_a", "prj_01", "ste_01", "usr_mgr", ParticipationRoleContractorWorker, from, to)
	ppB, _ := NewProjectParticipation(partyB, "ptp_b", "prj_01", "ste_01", "usr_mgr", ParticipationRoleContractorWorker, from, to)

	// Lateral sibling access between distinct contractors is denied
	if err := ppA.ValidateNoSiblingAccess(ppB); !errors.Is(err, ErrSiblingAccessDenied) {
		t.Errorf("expected ErrSiblingAccessDenied between sibling contractors, got %v", err)
	}

	// Cross-tenant access is denied
	partyForeign, _ := NewParty("ten_other", "prt_other", "Foreign", PartyTypeContractor)
	ppForeign, _ := NewProjectParticipation(partyForeign, "ptp_foreign", "prj_01", "ste_01", "usr_mgr", ParticipationRoleContractorWorker, from, to)
	if err := ppA.ValidateNoSiblingAccess(ppForeign); !errors.Is(err, ErrCrossTenantLinkage) {
		t.Errorf("expected ErrCrossTenantLinkage on foreign tenant participation, got %v", err)
	}
}

func TestProjectParticipation_ReversibleSimulationState(t *testing.T) {
	party, _ := NewParty("ten_01", "prt_01", "Contractor", PartyTypeContractor)
	from := time.Now().Add(-1 * time.Hour)
	to := time.Now().Add(24 * time.Hour)
	pp, _ := NewProjectParticipation(party, "ptp_01", "prj_01", "ste_01", "usr_mgr", ParticipationRoleContractorWorker, from, to)

	// 1. Simulate forward transition to StateClosed
	closed, err := SimulateReversibleParticipationState(pp, StateClosed)
	if err != nil || closed.State() != StateClosed {
		t.Fatalf("expected simulated StateClosed, got %v", closed.State())
	}

	// 2. Simulate reverse transition to StateActive (H030-002 preflight simulation requirement)
	active, err := SimulateReversibleParticipationState(closed, StateActive)
	if err != nil || active.State() != StateActive {
		t.Fatalf("expected simulated reversal to StateActive, got %v", active.State())
	}
}

func TestPartyRegistry_SubcontractorListing(t *testing.T) {
	reg := NewPartyRegistry()
	tenantID := "ten_01"

	primeParty, _ := NewParty(tenantID, "prt_prime", "Prime Corp", PartyTypeContractor)
	subParty1, _ := NewParty(tenantID, "prt_sub1", "Sub 1", PartyTypeSubcontractor)
	subParty2, _ := NewParty(tenantID, "prt_sub2", "Sub 2", PartyTypeSubcontractor)

	_ = reg.RegisterParty(primeParty)
	_ = reg.RegisterParty(subParty1)
	_ = reg.RegisterParty(subParty2)

	from := time.Now().Add(-1 * time.Hour)
	to := time.Now().Add(24 * time.Hour)

	primePart, _ := NewProjectParticipation(primeParty, "ptp_prime_10", "prj_01", "ste_01", "usr_mgr", ParticipationRoleSiteSafetyLead, from, to)
	subPart1, _ := NewNestedSubcontractorParticipation(primePart, subParty1, "ptp_sub_11", "ste_01", "usr_mgr", ParticipationRoleSubcontractorLead, from, to)
	subPart2, _ := NewNestedSubcontractorParticipation(primePart, subParty2, "ptp_sub_12", "ste_01", "usr_mgr", ParticipationRoleContractorWorker, from, to)

	_ = reg.RegisterParticipation(primePart)
	_ = reg.RegisterParticipation(subPart1)
	_ = reg.RegisterParticipation(subPart2)

	// List subcontractors under primePart
	subs, err := reg.ListSubcontractors(tenantID, "ptp_prime_10")
	if err != nil {
		t.Fatalf("unexpected ListSubcontractors error: %v", err)
	}
	if len(subs) != 2 {
		t.Errorf("expected 2 subcontractors under prime, got %d", len(subs))
	}
}
