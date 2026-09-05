package localidentity

import (
	"errors"
	"testing"
	"time"
)

func TestExternalUserProfile_CreationAndAccessors(t *testing.T) {
	tenantID := "ten_alpha"
	companyID := "cmp_sponsor"
	sponsorID := "usr_internal_manager"
	baseTime := time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC)
	from := baseTime
	to := baseTime.Add(30 * 24 * time.Hour)

	types := []ExternalUserType{
		ExternalTypeTemporaryWorker,
		ExternalTypeSiteLocalWorker,
		ExternalTypeContractorWorker,
		ExternalTypeClientInspector,
		ExternalTypeAuditor,
		ExternalTypePartnerSpecialist,
	}

	for i, uType := range types {
		subject := "usr_ext_synth_" + string(uType)
		scopes := []ScopeGrant{
			{TenantID: tenantID, CompanyID: companyID, ProjectID: "prj_01"},
		}

		profile, err := NewExternalUserProfile(
			subject, tenantID, companyID,
			uType, sponsorID, "External Partner Corp",
			"External Operator "+string(uType),
			"ref_synth_contact_01",
			from, to, scopes,
		)
		if err != nil {
			t.Fatalf("unexpected NewExternalUserProfile error for %s: %v", uType, err)
		}

		if profile.Subject() != subject {
			t.Errorf("subject mismatch for %s: %s", uType, profile.Subject())
		}
		if profile.TenantID() != tenantID {
			t.Errorf("tenantID mismatch: %s", profile.TenantID())
		}
		if profile.CompanyID() != companyID {
			t.Errorf("companyID mismatch: %s", profile.CompanyID())
		}
		if profile.UserType() != uType {
			t.Errorf("userType mismatch: %s", profile.UserType())
		}
		if profile.SponsorID() != sponsorID {
			t.Errorf("sponsorID mismatch: %s", profile.SponsorID())
		}
		if profile.OrganizationName() != "External Partner Corp" {
			t.Errorf("org mismatch: %s", profile.OrganizationName())
		}
		if profile.ContactReference() != "ref_synth_contact_01" {
			t.Errorf("contactReference mismatch: %s", profile.ContactReference())
		}
		if !profile.IsActive() || profile.Status() != EnrollmentStatusActive {
			t.Errorf("expected active status for %s", uType)
		}
		if len(profile.AssignedScopes()) != 1 {
			t.Errorf("scope count mismatch: %d", len(profile.AssignedScopes()))
		}
		_ = i
	}
}

func TestExternalUserProfile_MandatoryInternalSponsorValidation(t *testing.T) {
	tenantID := "ten_alpha"
	companyID := "cmp_sponsor"
	from := time.Now()
	to := from.Add(24 * time.Hour)

	// 1. Missing sponsor fails closed
	_, err := NewExternalUserProfile(
		"usr_ext_01", tenantID, companyID, ExternalTypeTemporaryWorker,
		"", "Partner Corp", "Worker", "ref_01", from, to, nil,
	)
	if !errors.Is(err, ErrMissingInternalSponsor) {
		t.Errorf("expected ErrMissingInternalSponsor for empty sponsor, got %v", err)
	}

	// 2. Non-user sponsor ID fails closed
	_, err = NewExternalUserProfile(
		"usr_ext_01", tenantID, companyID, ExternalTypeTemporaryWorker,
		"cmp_partner", "Partner Corp", "Worker", "ref_01", from, to, nil,
	)
	if !errors.Is(err, ErrInvalidInternalSponsor) {
		t.Errorf("expected ErrInvalidInternalSponsor for non-user sponsor, got %v", err)
	}

	// 3. External user attempting to sponsor another external user fails closed (anti-chain delegation)
	_, err = NewExternalUserProfile(
		"usr_ext_01", tenantID, companyID, ExternalTypeTemporaryWorker,
		"usr_ext_other_contractor", "Partner Corp", "Worker", "ref_01", from, to, nil,
	)
	if !errors.Is(err, ErrInvalidInternalSponsor) {
		t.Errorf("expected ErrInvalidInternalSponsor for external user acting as sponsor, got %v", err)
	}
}

func TestExternalUserProfile_CompanyAdministrationDenial(t *testing.T) {
	// External users must NEVER be granted TenantAdmin or ProjectManager roles
	for uType := range KnownExternalUserTypes {
		if err := AssertNoCompanyAdministration(uType, RoleTenantAdmin); !errors.Is(err, ErrCompanyAdminDenied) {
			t.Errorf("expected ErrCompanyAdminDenied for %s as TenantAdmin, got %v", uType, err)
		}
		if err := AssertNoCompanyAdministration(uType, RoleProjectManager); !errors.Is(err, ErrCompanyAdminDenied) {
			t.Errorf("expected ErrCompanyAdminDenied for %s as ProjectManager, got %v", uType, err)
		}
		// Non-administrative roles pass check
		if err := AssertNoCompanyAdministration(uType, RoleInspector); err != nil {
			t.Errorf("expected RoleInspector to pass admin denial check for %s: %v", uType, err)
		}
		if err := AssertNoCompanyAdministration(uType, RoleContractor); err != nil {
			t.Errorf("expected RoleContractor to pass admin denial check for %s: %v", uType, err)
		}
	}
}

func TestExternalUserProfile_ProfileMinimization(t *testing.T) {
	tenantID := "ten_alpha"
	companyID := "cmp_sponsor"
	sponsorID := "usr_manager"
	from := time.Now()
	to := from.Add(24 * time.Hour)

	// 1. Raw personal email rejected
	_, err := NewExternalUserProfile(
		"usr_ext_01", tenantID, companyID, ExternalTypeContractorWorker,
		sponsorID, "Vendor", "Somchai Prasert", "somchai@contractor-firm.com", from, to, nil,
	)
	if !errors.Is(err, ErrPIIDetected) {
		t.Errorf("expected ErrPIIDetected for email, got %v", err)
	}

	// 2. Raw phone number rejected
	_, err = NewExternalUserProfile(
		"usr_ext_01", tenantID, companyID, ExternalTypeContractorWorker,
		sponsorID, "Vendor", "Somchai Prasert", "+66812345678", from, to, nil,
	)
	if !errors.Is(err, ErrPIIDetected) {
		t.Errorf("expected ErrPIIDetected for phone, got %v", err)
	}

	// 3. National ID rejected
	_, err = NewExternalUserProfile(
		"usr_ext_01", tenantID, companyID, ExternalTypeContractorWorker,
		sponsorID, "Vendor", "Somchai Citizen ID:12345", "ref_synth_01", from, to, nil,
	)
	if !errors.Is(err, ErrPIIDetected) {
		t.Errorf("expected ErrPIIDetected for citizen ID, got %v", err)
	}
}

func TestExternalUserProfile_TemporalValidityAndExpiration(t *testing.T) {
	tenantID := "ten_alpha"
	companyID := "cmp_sponsor"
	sponsorID := "usr_manager"
	baseTime := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	from := baseTime
	to := baseTime.Add(14 * 24 * time.Hour)

	profile, _ := NewExternalUserProfile(
		"usr_ext_01", tenantID, companyID, ExternalTypeSiteLocalWorker,
		sponsorID, "Vendor", "Worker A", "ref_synth_01", from, to, nil,
	)

	before := baseTime.Add(-1 * time.Hour)
	during := baseTime.Add(7 * 24 * time.Hour)
	after := to.Add(1 * time.Hour)

	if !profile.IsValidAt(during) {
		t.Errorf("expected valid during validity window")
	}
	if profile.IsValidAt(before) {
		t.Errorf("expected invalid before validity window")
	}
	if profile.IsValidAt(after) {
		t.Errorf("expected invalid after validity window")
	}

	if profile.EffectiveStatus(during) != EnrollmentStatusActive {
		t.Errorf("expected active effective status during window")
	}
	if profile.EffectiveStatus(after) != EnrollmentStatusExpired {
		t.Errorf("expected expired effective status after window")
	}

	// Inverted validity window rejected
	_, err := NewExternalUserProfile(
		"usr_ext_01", tenantID, companyID, ExternalTypeSiteLocalWorker,
		sponsorID, "Vendor", "Worker A", "ref_synth_01", to, from, nil,
	)
	if !errors.Is(err, ErrInvalidTimeWindow) {
		t.Errorf("expected ErrInvalidTimeWindow on inverted dates, got %v", err)
	}
}

func TestExternalUserProfile_Revocation(t *testing.T) {
	tenantID := "ten_alpha"
	companyID := "cmp_sponsor"
	sponsorID := "usr_manager"
	from := time.Now().Add(-1 * time.Hour)
	to := time.Now().Add(24 * time.Hour)
	now := time.Now()

	profile, _ := NewExternalUserProfile(
		"usr_ext_01", tenantID, companyID, ExternalTypeClientInspector,
		sponsorID, "Client Corp", "Inspector A", "ref_synth_01", from, to, nil,
	)

	revoked, audit, err := profile.Revoke(now)
	if err != nil {
		t.Fatalf("unexpected Revoke error: %v", err)
	}
	if revoked.IsActive() || revoked.Status() != EnrollmentStatusRevoked {
		t.Errorf("expected revoked status")
	}
	if revoked.IsValidAt(now) {
		t.Errorf("revoked profile must not be valid")
	}
	if audit.Transition != "EXTERNAL_USER_REVOKED" || audit.SponsorID != sponsorID {
		t.Errorf("audit record mismatch: %+v", audit)
	}

	// Re-revoking fails closed
	_, _, err = revoked.Revoke(now)
	if !errors.Is(err, ErrEnrollmentRevoked) {
		t.Errorf("expected ErrEnrollmentRevoked on re-revoke, got %v", err)
	}
}

func TestExternalUserProfile_RegistryAndLedger(t *testing.T) {
	ledger := NewExternalUserLedger()
	registry := NewExternalUserRegistry(ledger)

	tenantID := "ten_alpha"
	companyID := "cmp_sponsor"
	sponsor1 := "usr_lead_01"
	sponsor2 := "usr_lead_02"
	from := time.Now().Add(-1 * time.Hour)
	to := time.Now().Add(48 * time.Hour)
	now := time.Now()

	p1, _ := NewExternalUserProfile("usr_ext_worker_1", tenantID, companyID, ExternalTypeTemporaryWorker, sponsor1, "Firm A", "Worker 1", "ref_01", from, to, nil)
	p2, _ := NewExternalUserProfile("usr_ext_worker_2", tenantID, companyID, ExternalTypeAuditor, sponsor2, "Firm B", "Auditor 2", "ref_02", from, to, nil)

	// 1. Enroll users
	if err := registry.EnrollExternalUser(p1, sponsor1, "Project ramp-up", now); err != nil {
		t.Fatalf("EnrollExternalUser p1 failed: %v", err)
	}
	if err := registry.EnrollExternalUser(p2, sponsor2, "Annual audit", now); err != nil {
		t.Fatalf("EnrollExternalUser p2 failed: %v", err)
	}

	// 2. Duplicate rejection
	if err := registry.EnrollExternalUser(p1, sponsor1, "Duplicate", now); !errors.Is(err, ErrDuplicateExternalUser) {
		t.Errorf("expected ErrDuplicateExternalUser, got %v", err)
	}

	// 3. List by sponsor
	s1List, err := registry.ListBySponsor(tenantID, sponsor1)
	if err != nil || len(s1List) != 1 || s1List[0].Subject() != "usr_ext_worker_1" {
		t.Errorf("ListBySponsor failed for sponsor 1: len=%d, err=%v", len(s1List), err)
	}

	// 4. Revoke enrollment
	revoked, err := registry.RevokeEnrollment(tenantID, "usr_ext_worker_1", sponsor1, "Early contract termination", now)
	if err != nil || revoked.Status() != EnrollmentStatusRevoked {
		t.Fatalf("RevokeEnrollment failed: %v", err)
	}

	// Active list now contains only p2
	activeList, _ := registry.ListActiveExternalUsers(tenantID, now)
	if len(activeList) != 1 || activeList[0].Subject() != "usr_ext_worker_2" {
		t.Errorf("expected only p2 in active list, got %d items", len(activeList))
	}

	// 5. Audit trail verification
	trail, err := ledger.GetAuditTrail(tenantID, "usr_ext_worker_1")
	if err != nil || len(trail) != 2 {
		t.Fatalf("expected 2 audit records for worker 1, got %d (err: %v)", len(trail), err)
	}
	if trail[0].Transition != "EXTERNAL_USER_ENROLLED" || trail[1].Transition != "EXTERNAL_USER_REVOKED" {
		t.Errorf("trail transition mismatch: %+v", trail)
	}

	// 6. Cross-tenant isolation verification
	foreignTrail, err := ledger.GetAuditTrail("ten_other", "usr_ext_worker_1")
	if err != nil || len(foreignTrail) != 0 {
		t.Errorf("cross-tenant leakage: foreign tenant retrieved audit records")
	}
}

func TestExternalUserProfile_ScopeBounding(t *testing.T) {
	tenantID := "ten_alpha"
	companyID := "cmp_sponsor"
	sponsorID := "usr_manager"
	from := time.Now()
	to := from.Add(24 * time.Hour)

	scopes := []ScopeGrant{
		{TenantID: tenantID, ProjectID: "prj_alpha", SiteID: "ste_01"},
	}

	profile, _ := NewExternalUserProfile(
		"usr_ext_01", tenantID, companyID, ExternalTypeContractorWorker,
		sponsorID, "Vendor", "Worker", "ref_synth_01", from, to, scopes,
	)

	// Target 1: Matching project and site -> permitted
	targetMatch := TargetResource{TenantID: tenantID, ProjectID: "prj_alpha", SiteID: "ste_01"}
	if !profile.IsScopePermitted(targetMatch) {
		t.Errorf("expected scope match to be permitted")
	}

	// Target 2: Wrong site in same project -> denied
	targetWrongSite := TargetResource{TenantID: tenantID, ProjectID: "prj_alpha", SiteID: "ste_02"}
	if profile.IsScopePermitted(targetWrongSite) {
		t.Errorf("expected wrong site to be denied")
	}

	// Target 3: Unrelated project -> denied
	targetWrongProject := TargetResource{TenantID: tenantID, ProjectID: "prj_beta", SiteID: "ste_01"}
	if profile.IsScopePermitted(targetWrongProject) {
		t.Errorf("expected wrong project to be denied")
	}
}
