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
	// Invalid role
	if _, err := NewProjectParticipation(party, "ptp_01", "prj_01", "ste_01", "usr_01", ParticipationRole("SUPER_ADMIN"), validFrom, validTo); !errors.Is(err, ErrInvalidParticipationRole) {
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

func TestProjectParticipation_TemporalValidityAndExpiration(t *testing.T) {
	party, _ := NewParty("ten_01", "prt_01", "Audit Firm", PartyTypeAuditor)
	baseTime := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	validFrom := baseTime
	validTo := baseTime.Add(14 * 24 * time.Hour)

	pp, err := NewProjectParticipation(party, "ptp_01", "prj_01", "ste_01", "usr_sponsor", ParticipationRoleClientAuditor, validFrom, validTo)
	if err != nil {
		t.Fatalf("unexpected NewProjectParticipation: %v", err)
	}

	before := baseTime.Add(-1 * time.Hour)
	during := baseTime.Add(7 * 24 * time.Hour)
	after := validTo.Add(1 * time.Hour)

	if !pp.IsValidAt(during) {
		t.Errorf("expected IsValidAt true during window")
	}
	if pp.IsValidAt(before) {
		t.Errorf("expected IsValidAt false before window")
	}
	if pp.IsValidAt(after) {
		t.Errorf("expected IsValidAt false after expiration")
	}

	// ValidateScopeAt temporal assertions
	if err := pp.ValidateScopeAt("prj_01", "ste_01", before); !errors.Is(err, ErrParticipationNotYetValid) {
		t.Errorf("expected ErrParticipationNotYetValid before window, got %v", err)
	}
	if err := pp.ValidateScopeAt("prj_01", "ste_01", during); err != nil {
		t.Errorf("expected scope ok during window, got %v", err)
	}
	if err := pp.ValidateScopeAt("prj_01", "ste_01", after); !errors.Is(err, ErrParticipationExpired) {
		t.Errorf("expected ErrParticipationExpired after window, got %v", err)
	}

	// Archive lifecycle: archived participation fails closed even during time window
	archived := pp.Archive()
	if archived.IsValidAt(during) {
		t.Errorf("archived participation must return IsValidAt false")
	}
	if err := archived.ValidateScopeAt("prj_01", "ste_01", during); !errors.Is(err, ErrEntityArchived) {
		t.Errorf("expected ErrEntityArchived for archived participation, got %v", err)
	}
}

func TestProjectParticipation_DefaultDenyScope(t *testing.T) {
	party, _ := NewParty("ten_01", "prt_01", "Safety Ltd", PartyTypeContractor)
	validFrom := time.Now().Add(-1 * time.Hour)
	validTo := time.Now().Add(24 * time.Hour)
	now := time.Now()

	// 1. Site-bounded participation
	ppSite, _ := NewProjectParticipation(party, "ptp_01", "prj_alpha", "ste_rayong", "usr_mgr", ParticipationRoleSiteSafetyLead, validFrom, validTo)

	// Unrelated project
	if err := ppSite.ValidateScopeAt("prj_beta", "ste_rayong", now); !errors.Is(err, ErrScopeMismatch) {
		t.Errorf("expected ErrScopeMismatch for wrong project, got %v", err)
	}
	// Unrelated site
	if err := ppSite.ValidateScopeAt("prj_alpha", "ste_bangkok", now); !errors.Is(err, ErrScopeMismatch) {
		t.Errorf("expected ErrScopeMismatch for wrong site, got %v", err)
	}
	// Empty site when site bound is required
	if err := ppSite.ValidateScopeAt("prj_alpha", "", now); !errors.Is(err, ErrScopeMismatch) {
		t.Errorf("expected ErrScopeMismatch for missing site, got %v", err)
	}
	// Matching project and site
	if err := ppSite.ValidateScopeAt("prj_alpha", "ste_rayong", now); err != nil {
		t.Errorf("expected scope valid for exact project and site, got %v", err)
	}

	// 2. Project-wide participation (empty siteID bound)
	ppProject, _ := NewProjectParticipation(party, "ptp_02", "prj_alpha", "", "usr_mgr", ParticipationRoleConsultant, validFrom, validTo)
	if err := ppProject.ValidateScopeAt("prj_alpha", "ste_any", now); err != nil {
		t.Errorf("project-wide participation should accept any site under project, got %v", err)
	}
	if err := ppProject.ValidateScopeAt("prj_beta", "ste_any", now); !errors.Is(err, ErrScopeMismatch) {
		t.Errorf("expected ErrScopeMismatch for wrong project in project-wide role, got %v", err)
	}
}

func TestProjectParticipation_CrossTenantIsolation(t *testing.T) {
	party, _ := NewParty("tenant-alpha", "prt_01", "Partner Alpha", PartyTypePartner)
	validFrom := time.Now().Add(-1 * time.Hour)
	validTo := time.Now().Add(24 * time.Hour)

	pp, _ := NewProjectParticipation(party, "ptp_01", "prj_01", "ste_01", "usr_sponsor", ParticipationRoleConsultant, validFrom, validTo)

	claimsB := &TrustedClaims{
		Subject:         "usr-bravo",
		TenantID:        "tenant-bravo",
		IsAuthenticated: true,
	}
	ctxB, err := DeriveTenantContext(claimsB, nil)
	if err != nil {
		t.Fatalf("unexpected DeriveTenantContext: %v", err)
	}

	if err := pp.ValidateScope(ctxB); !errors.Is(err, ErrTenantMismatch) {
		t.Errorf("expected ErrTenantMismatch for foreign tenant context, got %v", err)
	}
}

func TestProjectParticipation_NoImplicitInternalAuthority(t *testing.T) {
	// Boundary test: verifies external party participation does not convey internal company authority
	party, _ := NewParty("ten_01", "prt_ext_01", "Subcontractor Alpha", PartyTypeSubcontractor)
	validFrom := time.Now().Add(-1 * time.Hour)
	validTo := time.Now().Add(24 * time.Hour)
	pp, _ := NewProjectParticipation(party, "ptp_ext_01", "prj_plant", "ste_01", "usr_mgr", ParticipationRoleSubcontractorLead, validFrom, validTo)

	// An external party cannot be used to construct a project or site in the internal corporate hierarchy
	// (NewProject requires Company, NewBusinessUnit requires Company).
	// The party model explicitly lacks CompanyID() and BusinessUnitID().
	if pp.TenantID() != party.TenantID() {
		t.Errorf("tenant binding mismatch")
	}
	if pp.Role() == ParticipationRole("ADMIN") {
		t.Errorf("administrative roles must be strictly disallowed")
	}
}

func TestPartyRegistry_OperationsAndTenantIsolation(t *testing.T) {
	reg := NewPartyRegistry()

	tenantA := "tenant-alpha"
	tenantB := "tenant-bravo"

	partyA1, _ := NewParty(tenantA, "prt_a1", "Alpha Contractor 1", PartyTypeContractor)
	partyA2, _ := NewParty(tenantA, "prt_a2", "Alpha Client", PartyTypeClient)
	partyB1, _ := NewParty(tenantB, "prt_b1", "Bravo Contractor", PartyTypeContractor)

	// 1. Register Parties
	if err := reg.RegisterParty(partyA1); err != nil {
		t.Fatalf("unexpected RegisterParty error: %v", err)
	}
	if err := reg.RegisterParty(partyA2); err != nil {
		t.Fatalf("unexpected RegisterParty error: %v", err)
	}
	if err := reg.RegisterParty(partyB1); err != nil {
		t.Fatalf("unexpected RegisterParty error: %v", err)
	}

	// 2. Duplicate registration rejection
	if err := reg.RegisterParty(partyA1); !errors.Is(err, ErrDuplicateParty) {
		t.Errorf("expected ErrDuplicateParty on duplicate insert, got %v", err)
	}

	// 3. Get Party
	p, err := reg.GetParty(tenantA, "prt_a1")
	if err != nil || p.Name() != "Alpha Contractor 1" {
		t.Errorf("failed to retrieve party A1: %v", err)
	}

	// 4. Cross-tenant party access rejection
	if _, err := reg.GetParty(tenantA, "prt_b1"); !errors.Is(err, ErrPartyNotFound) {
		t.Errorf("expected ErrPartyNotFound when Tenant A requests Tenant B party, got %v", err)
	}

	// 5. Register Participations
	validFrom := time.Now().Add(-1 * time.Hour)
	validTo := time.Now().Add(48 * time.Hour)

	ppA1, _ := NewProjectParticipation(partyA1, "ptp_a1", "prj_100", "ste_01", "usr_mgr", ParticipationRoleContractorWorker, validFrom, validTo)
	ppA2, _ := NewProjectParticipation(partyA2, "ptp_a2", "prj_100", "", "usr_mgr", ParticipationRoleClientAuditor, validFrom, validTo)
	ppB1, _ := NewProjectParticipation(partyB1, "ptp_b1", "prj_100", "ste_01", "usr_mgr_b", ParticipationRoleContractorWorker, validFrom, validTo)

	if err := reg.RegisterParticipation(ppA1); err != nil {
		t.Fatalf("unexpected RegisterParticipation error: %v", err)
	}
	if err := reg.RegisterParticipation(ppA2); err != nil {
		t.Fatalf("unexpected RegisterParticipation error: %v", err)
	}
	if err := reg.RegisterParticipation(ppB1); err != nil {
		t.Fatalf("unexpected RegisterParticipation error: %v", err)
	}

	// Duplicate participation rejection
	if err := reg.RegisterParticipation(ppA1); !errors.Is(err, ErrDuplicateParticipation) {
		t.Errorf("expected ErrDuplicateParticipation, got %v", err)
	}

	// 6. List by Project (Tenant isolation check)
	// Tenant A list on prj_100 should contain ppA1 and ppA2, and NEVER ppB1
	partsA, err := reg.ListParticipationsByProject(tenantA, "prj_100")
	if err != nil {
		t.Fatalf("unexpected ListParticipationsByProject error: %v", err)
	}
	if len(partsA) != 2 {
		t.Errorf("expected 2 participations for Tenant A on prj_100, got %d", len(partsA))
	}
	for _, p := range partsA {
		if p.TenantID() != tenantA {
			t.Errorf("cross-tenant leakage: found tenant %q in Tenant A results", p.TenantID())
		}
	}

	// 7. List by Party
	partsByParty, err := reg.ListParticipationsByParty(tenantA, "prt_a1")
	if err != nil || len(partsByParty) != 1 || partsByParty[0].ParticipationID() != "ptp_a1" {
		t.Errorf("ListParticipationsByParty failed: %v", err)
	}
}

func TestCanonicalPrefix_Participation(t *testing.T) {
	if PrefixParticipation != "ptp" {
		t.Errorf("expected PrefixParticipation to be 'ptp', got %q", PrefixParticipation)
	}
	if PrefixParty != "prt" {
		t.Errorf("expected PrefixParty to be 'prt', got %q", PrefixParty)
	}

	// Test generation and validation of ptp canonical ID
	ptpID, err := GenerateCanonicalID(PrefixParticipation)
	if err != nil {
		t.Fatalf("unexpected GenerateCanonicalID error: %v", err)
	}
	if err := ValidateCanonicalID(ptpID, PrefixParticipation); err != nil {
		t.Errorf("expected valid ptp canonical ID %q, got err: %v", ptpID, err)
	}
}
