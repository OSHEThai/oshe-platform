package localidentity

import (
	"errors"
	"testing"
	"time"
)

func TestAccessCondition_CreationAndAccessors(t *testing.T) {
	tenantID := "ten_alpha"
	subject := "usr_ext_synth_worker_01"
	projectID := "prj_plant_01"
	siteID := "ste_rayong"
	sponsorID := "usr_internal_manager"
	baseTime := time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC)
	from := baseTime
	to := baseTime.Add(10 * 24 * time.Hour) // 10 days <= 14 days limit

	cond, err := NewAccessConditionRecord(
		"cnd_01", tenantID, subject, projectID, siteID, sponsorID,
		false, false, from, to,
	)
	if err != nil {
		t.Fatalf("unexpected NewAccessConditionRecord error: %v", err)
	}

	if cond.ConditionID() != "cnd_01" {
		t.Errorf("conditionID mismatch: %s", cond.ConditionID())
	}
	if cond.TenantID() != tenantID {
		t.Errorf("tenantID mismatch: %s", cond.TenantID())
	}
	if cond.Subject() != subject {
		t.Errorf("subject mismatch: %s", cond.Subject())
	}
	if cond.ProjectID() != projectID {
		t.Errorf("projectID mismatch: %s", cond.ProjectID())
	}
	if cond.SiteID() != siteID {
		t.Errorf("siteID mismatch: %s", cond.SiteID())
	}
	if cond.SponsorID() != sponsorID {
		t.Errorf("sponsorID mismatch: %s", cond.SponsorID())
	}
	if cond.Generation() != 1 {
		t.Errorf("expected initial generation 1, got %d", cond.Generation())
	}
	if cond.RenewalCount() != 0 {
		t.Errorf("expected initial renewalCount 0, got %d", cond.RenewalCount())
	}
	if !cond.IsActive() || cond.Status() != AccessConditionActive {
		t.Errorf("expected active status")
	}
}

func TestAccessCondition_ProhibitTrustedDeviceAndOfflineAccess(t *testing.T) {
	tenantID := "ten_alpha"
	subject := "usr_ext_01"
	from := time.Now()
	to := from.Add(7 * 24 * time.Hour)

	// 1. Claiming trusted device required is strictly barred under H030-004
	_, err := NewAccessConditionRecord(
		"cnd_01", tenantID, subject, "prj_01", "ste_01", "usr_mgr",
		true, false, from, to,
	)
	if !errors.Is(err, ErrTrustedDeviceProhibited) {
		t.Errorf("expected ErrTrustedDeviceProhibited when trustedDeviceRequired=true, got %v", err)
	}

	// 2. Claiming allow offline access is strictly barred under H030-004
	_, err = NewAccessConditionRecord(
		"cnd_01", tenantID, subject, "prj_01", "ste_01", "usr_mgr",
		false, true, from, to,
	)
	if !errors.Is(err, ErrTrustedDeviceProhibited) {
		t.Errorf("expected ErrTrustedDeviceProhibited when allowOffline=true, got %v", err)
	}

	// 3. AssertEmergencyAccessDenied rejects break-glass
	if err := AssertEmergencyAccessDenied(true); !errors.Is(err, ErrEmergencyAccessDenied) {
		t.Errorf("expected ErrEmergencyAccessDenied on break-glass attempt, got %v", err)
	}
}

func TestAccessCondition_DurationCeilingsAndRejections(t *testing.T) {
	tenantID := "ten_alpha"
	subject := "usr_ext_01"
	from := time.Now()

	// Duration > 14 days is rejected
	longTo := from.Add(15 * 24 * time.Hour)
	_, err := NewAccessConditionRecord(
		"cnd_01", tenantID, subject, "prj_01", "ste_01", "usr_mgr",
		false, false, from, longTo,
	)
	if !errors.Is(err, ErrDurationExceeded) {
		t.Errorf("expected ErrDurationExceeded for 15 days, got %v", err)
	}

	// Inverted dates rejected
	_, err = NewAccessConditionRecord(
		"cnd_01", tenantID, subject, "prj_01", "ste_01", "usr_mgr",
		false, false, longTo, from,
	)
	if !errors.Is(err, ErrInvalidTimeWindow) {
		t.Errorf("expected ErrInvalidTimeWindow on inverted dates, got %v", err)
	}

	// Blank project ID rejected
	_, err = NewAccessConditionRecord(
		"cnd_01", tenantID, subject, "", "ste_01", "usr_mgr",
		false, false, from, from.Add(7*24*time.Hour),
	)
	if !errors.Is(err, ErrBlankProjectID) {
		t.Errorf("expected ErrBlankProjectID, got %v", err)
	}
}

func TestAccessCondition_TemporalValidityAndScopeMatching(t *testing.T) {
	tenantID := "ten_alpha"
	subject := "usr_ext_01"
	baseTime := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	from := baseTime
	to := baseTime.Add(7 * 24 * time.Hour)

	cond, _ := NewAccessConditionRecord(
		"cnd_01", tenantID, subject, "prj_plant", "ste_site_a", "usr_mgr",
		false, false, from, to,
	)

	before := baseTime.Add(-1 * time.Hour)
	during := baseTime.Add(3 * 24 * time.Hour)
	after := to.Add(1 * time.Hour)

	if !cond.IsValidAt(during) {
		t.Errorf("expected valid during validity window")
	}
	if cond.IsValidAt(before) {
		t.Errorf("expected invalid before validity window")
	}
	if cond.IsValidAt(after) {
		t.Errorf("expected invalid after validity window")
	}

	if cond.EffectiveStatus(during) != AccessConditionActive {
		t.Errorf("expected active status during window")
	}
	if cond.EffectiveStatus(after) != AccessConditionExpired {
		t.Errorf("expected expired status after window")
	}

	// Scope matching
	if !cond.ScopeMatches("prj_plant", "ste_site_a") {
		t.Errorf("expected scope match for matching project and site")
	}
	if cond.ScopeMatches("prj_other", "ste_site_a") {
		t.Errorf("expected scope mismatch for wrong project")
	}
	if cond.ScopeMatches("prj_plant", "ste_site_b") {
		t.Errorf("expected scope mismatch for wrong site")
	}
}

func TestAccessCondition_SponsorChangeProtocol(t *testing.T) {
	registry := NewAccessConditionRegistry(nil)
	tenantID := "ten_alpha"
	from := time.Now().Add(-1 * time.Hour)
	to := from.Add(7 * 24 * time.Hour)
	now := time.Now()

	cond, _ := NewAccessConditionRecord(
		"cnd_sp_01", tenantID, "usr_ext_01", "prj_01", "ste_01", "usr_mgr_initial",
		false, false, from, to,
	)
	_ = registry.CreateCondition(cond, "usr_admin", "Initial", now)

	// 1. Change sponsor to a new internal manager
	updated, err := registry.ChangeSponsor(tenantID, "cnd_sp_01", "usr_mgr_replacement", "usr_admin", "Manager transferred", now)
	if err != nil {
		t.Fatalf("ChangeSponsor error: %v", err)
	}

	if updated.SponsorID() != "usr_mgr_replacement" {
		t.Errorf("expected new sponsor ID 'usr_mgr_replacement', got %s", updated.SponsorID())
	}
	// Generation must be bumped from 1 to 2
	if updated.Generation() != 2 {
		t.Errorf("expected generation 2 after sponsor change, got %d", updated.Generation())
	}

	// 2. Re-assigning to identical sponsor fails closed
	_, err = registry.ChangeSponsor(tenantID, "cnd_sp_01", "usr_mgr_replacement", "usr_admin", "Duplicate", now)
	if !errors.Is(err, ErrSponsorUnchanged) {
		t.Errorf("expected ErrSponsorUnchanged for identical sponsor, got %v", err)
	}

	// 3. Non-user sponsor fails closed
	_, err = registry.ChangeSponsor(tenantID, "cnd_sp_01", "ext_contractor", "usr_admin", "External", now)
	if !errors.Is(err, ErrInvalidInternalSponsor) {
		t.Errorf("expected ErrInvalidInternalSponsor for non-user sponsor, got %v", err)
	}
}

func TestAccessCondition_RenewalProtocol(t *testing.T) {
	registry := NewAccessConditionRegistry(nil)
	tenantID := "ten_alpha"
	baseTime := time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC)
	from := baseTime
	to := baseTime.Add(7 * 24 * time.Hour)
	now := baseTime.Add(2 * 24 * time.Hour)

	cond, _ := NewAccessConditionRecord(
		"cnd_rn_01", tenantID, "usr_ext_01", "prj_01", "ste_01", "usr_mgr",
		false, false, from, to,
	)
	_ = registry.CreateCondition(cond, "usr_admin", "Initial", now)

	// 1. Renewal with 5 days extension (<= 7 days ceiling)
	extension := 5 * 24 * time.Hour
	renewed, err := registry.RenewAccess(tenantID, "cnd_rn_01", extension, "usr_mgr", "Project extension", now)
	if err != nil {
		t.Fatalf("RenewAccess error: %v", err)
	}

	expectedNewValidTo := to.Add(extension)
	if renewed.ValidTo() != expectedNewValidTo {
		t.Errorf("validTo mismatch: expected %v, got %v", expectedNewValidTo, renewed.ValidTo())
	}
	if renewed.Generation() != 2 {
		t.Errorf("expected generation 2 after renewal, got %d", renewed.Generation())
	}
	if renewed.RenewalCount() != 1 {
		t.Errorf("expected renewalCount 1, got %d", renewed.RenewalCount())
	}

	// 2. Renewal extension > 7 days fails closed
	_, err = registry.RenewAccess(tenantID, "cnd_rn_01", 8*24*time.Hour, "usr_mgr", "Too long", now)
	if !errors.Is(err, ErrRenewalDurationExceeded) {
		t.Errorf("expected ErrRenewalDurationExceeded for 8 days, got %v", err)
	}
}

func TestAccessCondition_DeactivationAndStaleSessionDenial(t *testing.T) {
	registry := NewAccessConditionRegistry(nil)
	tenantID := "ten_alpha"
	from := time.Now().Add(-1 * time.Hour)
	to := from.Add(7 * 24 * time.Hour)
	now := time.Now()

	cond, _ := NewAccessConditionRecord(
		"cnd_eval_01", tenantID, "usr_ext_01", "prj_01", "ste_01", "usr_mgr_1",
		false, false, from, to,
	)
	_ = registry.CreateCondition(cond, "usr_admin", "Initial", now)

	// Step 1: Initial evaluation with session token generation == 1 -> ALLOWED
	allowed, cat := registry.EvaluateConditionAccess(tenantID, "cnd_eval_01", "prj_01", "ste_01", 1, now)
	if !allowed || cat != CategoryNone {
		t.Errorf("expected initial access allowed with generation 1, got allowed=%v, cat=%s", allowed, cat)
	}

	// Step 2: Sponsor change bumps condition generation to 2
	_, err := registry.ChangeSponsor(tenantID, "cnd_eval_01", "usr_mgr_2", "usr_admin", "Reassign", now)
	if err != nil {
		t.Fatalf("ChangeSponsor error: %v", err)
	}

	// Step 3: Immediate local authority effect: caller using old session token (gen 1) is DENIED as STALE
	allowedOldToken, catOldToken := registry.EvaluateConditionAccess(tenantID, "cnd_eval_01", "prj_01", "ste_01", 1, now)
	if allowedOldToken || catOldToken != CategorySessionStale {
		t.Errorf("expected CategorySessionStale for old session generation 1, got allowed=%v, cat=%s", allowedOldToken, catOldToken)
	}

	// Step 4: Re-authenticated session token with generation == 2 -> ALLOWED
	allowedNewToken, catNewToken := registry.EvaluateConditionAccess(tenantID, "cnd_eval_01", "prj_01", "ste_01", 2, now)
	if !allowedNewToken || catNewToken != CategoryNone {
		t.Errorf("expected access allowed for refreshed session token generation 2, got allowed=%v, cat=%s", allowedNewToken, catNewToken)
	}

	// Step 5: Deactivation immediately renders condition REVOKED
	_, err = registry.DeactivateAccess(tenantID, "cnd_eval_01", "usr_admin", "Access terminated", now)
	if err != nil {
		t.Fatalf("DeactivateAccess error: %v", err)
	}

	// Evaluation after deactivation fails with CategoryIdentityInactive
	allowedDeact, catDeact := registry.EvaluateConditionAccess(tenantID, "cnd_eval_01", "prj_01", "ste_01", 2, now)
	if allowedDeact || catDeact != CategoryIdentityInactive {
		t.Errorf("expected CategoryIdentityInactive after deactivation, got allowed=%v, cat=%s", allowedDeact, catDeact)
	}
}

func TestAccessCondition_LedgerAndTenantIsolation(t *testing.T) {
	ledger := NewAccessConditionLedger()
	registry := NewAccessConditionRegistry(ledger)

	tenantA := "ten_alpha"
	tenantB := "ten_bravo"
	from := time.Now().Add(-1 * time.Hour)
	to := from.Add(7 * 24 * time.Hour)
	now := time.Now()

	cond, _ := NewAccessConditionRecord(
		"cnd_audit_01", tenantA, "usr_ext_01", "prj_01", "ste_01", "usr_mgr_1",
		false, false, from, to,
	)
	_ = registry.CreateCondition(cond, "usr_admin", "Init", now)

	// Sponsor change
	_, _ = registry.ChangeSponsor(tenantA, "cnd_audit_01", "usr_mgr_2", "usr_admin", "Change sponsor", now)

	// Renewal
	_, _ = registry.RenewAccess(tenantA, "cnd_audit_01", 3*24*time.Hour, "usr_mgr_2", "Extend", now)

	// Deactivation
	_, _ = registry.DeactivateAccess(tenantA, "cnd_audit_01", "usr_admin", "Revoke", now)

	// 1. Audit trail for Tenant A has 4 records
	history, err := ledger.GetConditionHistory(tenantA, "cnd_audit_01")
	if err != nil || len(history) != 4 {
		t.Fatalf("expected 4 audit records, got %d (err: %v)", len(history), err)
	}
	if history[0].Transition != "CONDITION_CREATED" ||
		history[1].Transition != "SPONSOR_CHANGED" ||
		history[2].Transition != "ACCESS_RENEWED" ||
		history[3].Transition != "ACCESS_DEACTIVATED" {
		t.Errorf("audit history transitions mismatch: %+v", history)
	}

	// 2. Tenant boundary isolation: Tenant B query returns 0 records
	leakCheck, err := ledger.GetConditionHistory(tenantB, "cnd_audit_01")
	if err != nil || len(leakCheck) != 0 {
		t.Errorf("cross-tenant leakage: foreign tenant retrieved audit history")
	}
}
