package reporting_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	reporting "github.com/oshethai/oshe-platform/modules/reporting-localization"
)

func defaultFlag(id string) reporting.FeatureFlag {
	return reporting.FeatureFlag{
		FlagID:      id,
		Title:       "Advanced Safety Trend Analytics",
		Description: "Enables multi-factor workplace hazard trend visualizer",
		DefaultOff:  true,
		Enabled:     false,
		Stage:       reporting.StageAlpha,
		Owner:       "analytics-team",
		Rollout: reporting.RolloutMetadata{
			Percentage:     100,
			AllowedTenants: []string{"ten_alpha"},
			AllowedRoles:   []string{"SAFETY_OFFICER", "COMPLIANCE_LEAD"},
		},
		Accessibility: reporting.AccessibilityMetadata{
			KeyboardNavigable: true,
			ScreenReaderLabel: "Open advanced hazard trend visualizer",
			AriaRole:          "button",
			ContrastCertified: true,
		},
	}
}

func authorizedContext(tenantID string) reporting.EvaluationContext {
	return reporting.EvaluationContext{
		TenantID:       tenantID,
		SubjectID:      "user_alice",
		CallerRoles:    []string{"SAFETY_OFFICER"},
		IsAuthorized:   true,
		EvaluationTime: time.Now().UTC(),
	}
}

func TestFeatureFlag_DefaultOffInvariant(t *testing.T) {
	registry := reporting.NewFeatureFlagRegistry(nil)

	// Valid default-off registration
	flag := defaultFlag("flag_default_off")
	if err := registry.RegisterFlag(flag); err != nil {
		t.Fatalf("RegisterFlag failed: %v", err)
	}

	// Evaluation before explicit enable must be false
	res := registry.Evaluate("flag_default_off", authorizedContext("ten_alpha"))
	if res.Exposed {
		t.Errorf("expected flag to evaluate to false under default-off invariant")
	}

	// Attempting to register flag with DefaultOff = false must fail closed
	badFlag := defaultFlag("flag_bad")
	badFlag.DefaultOff = false
	err := registry.RegisterFlag(badFlag)
	if !errors.Is(err, reporting.ErrMustDefaultOff) {
		t.Fatalf("expected ErrMustDefaultOff when DefaultOff is false, got: %v", err)
	}
}

func TestFeatureFlag_EnableAndDisableSafeFallback(t *testing.T) {
	registry := reporting.NewFeatureFlagRegistry(nil)
	flag := defaultFlag("flag_toggle")
	_ = registry.RegisterFlag(flag)

	// Initially disabled -> safe fallback active
	res1 := registry.Evaluate("flag_toggle", authorizedContext("ten_alpha"))
	if res1.Exposed {
		t.Fatalf("expected initial evaluation to be disabled")
	}
	if !strings.Contains(res1.Reason, "safe fallback active") {
		t.Errorf("expected safe fallback reason, got: %s", res1.Reason)
	}

	// Enable flag
	if err := registry.SetFlagState("flag_toggle", true); err != nil {
		t.Fatalf("SetFlagState enable failed: %v", err)
	}

	res2 := registry.Evaluate("flag_toggle", authorizedContext("ten_alpha"))
	if !res2.Exposed {
		t.Fatalf("expected enabled flag to evaluate to true, reason: %s", res2.Reason)
	}

	// Disable flag -> safe fallback restores immediately without dirty state
	if err := registry.SetFlagState("flag_toggle", false); err != nil {
		t.Fatalf("SetFlagState disable failed: %v", err)
	}

	res3 := registry.Evaluate("flag_toggle", authorizedContext("ten_alpha"))
	if res3.Exposed {
		t.Fatalf("expected disabled flag to evaluate to false")
	}
}

func TestFeatureFlag_UnauthorizedCannotBypassControls(t *testing.T) {
	registry := reporting.NewFeatureFlagRegistry(nil)
	flag := defaultFlag("flag_auth_check")
	flag.Enabled = true
	_ = registry.RegisterFlag(flag)

	// Caller is unauthorized (IsAuthorized = false)
	unauthCtx := authorizedContext("ten_alpha")
	unauthCtx.IsAuthorized = false

	res := registry.Evaluate("flag_auth_check", unauthCtx)
	if res.Exposed {
		t.Fatalf("critical invariant breach: feature flag granted access to unauthorized caller!")
	}
	if !strings.Contains(res.Reason, "cannot bypass security controls") {
		t.Errorf("expected authorization denial reason, got: %s", res.Reason)
	}
	if !strings.Contains(res.AuthorityNote, "FEATURE_FLAG_NON_AUTHORITY") {
		t.Errorf("expected non-authority notice, got: %s", res.AuthorityNote)
	}
}

func TestFeatureFlag_StaleConfigTimeBoundary(t *testing.T) {
	baseTime := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	registry := reporting.NewFeatureFlagRegistry(func() time.Time { return baseTime })

	flag := defaultFlag("flag_stale_window")
	flag.Enabled = true
	flag.Rollout.EffectiveFrom = baseTime.Add(1 * time.Hour)
	flag.Rollout.EffectiveTo = baseTime.Add(4 * time.Hour)
	_ = registry.RegisterFlag(flag)

	// 1. Before effective window
	ctxEarly := authorizedContext("ten_alpha")
	ctxEarly.EvaluationTime = baseTime.Add(30 * time.Minute)
	resEarly := registry.Evaluate("flag_stale_window", ctxEarly)
	if resEarly.Exposed {
		t.Errorf("expected false before effective window")
	}

	// 2. Inside effective window
	ctxActive := authorizedContext("ten_alpha")
	ctxActive.EvaluationTime = baseTime.Add(2 * time.Hour)
	resActive := registry.Evaluate("flag_stale_window", ctxActive)
	if !resActive.Exposed {
		t.Errorf("expected true within effective window, reason: %s", resActive.Reason)
	}

	// 3. Past effective window (stale config)
	ctxStale := authorizedContext("ten_alpha")
	ctxStale.EvaluationTime = baseTime.Add(5 * time.Hour)
	resStale := registry.Evaluate("flag_stale_window", ctxStale)
	if resStale.Exposed {
		t.Errorf("expected false after effective window expired")
	}
	if !strings.Contains(resStale.Reason, "stale configuration") {
		t.Errorf("expected stale configuration reason, got: %s", resStale.Reason)
	}
}

func TestFeatureFlag_TenantIsolation(t *testing.T) {
	registry := reporting.NewFeatureFlagRegistry(nil)
	flag := defaultFlag("flag_tenant_scope")
	flag.Enabled = true
	flag.Rollout.AllowedTenants = []string{"ten_alpha"}
	_ = registry.RegisterFlag(flag)

	// Tenant Alpha is allowed
	resAlpha := registry.Evaluate("flag_tenant_scope", authorizedContext("ten_alpha"))
	if !resAlpha.Exposed {
		t.Errorf("expected ten_alpha to be exposed")
	}

	// Tenant Bravo is not allowed
	resBravo := registry.Evaluate("flag_tenant_scope", authorizedContext("ten_bravo"))
	if resBravo.Exposed {
		t.Errorf("expected ten_bravo to be rejected")
	}
	if !strings.Contains(resBravo.Reason, "not included in rollout allowlist") {
		t.Errorf("expected tenant allowlist rejection reason, got: %s", resBravo.Reason)
	}
}

func TestFeatureFlag_RoleAllowlist(t *testing.T) {
	registry := reporting.NewFeatureFlagRegistry(nil)
	flag := defaultFlag("flag_role_scope")
	flag.Enabled = true
	flag.Rollout.AllowedRoles = []string{"SAFETY_OFFICER", "COMPLIANCE_LEAD"}
	_ = registry.RegisterFlag(flag)

	// Caller with matching role
	ctxOfficer := authorizedContext("ten_alpha")
	ctxOfficer.CallerRoles = []string{"SAFETY_OFFICER"}
	resOfficer := registry.Evaluate("flag_role_scope", ctxOfficer)
	if !resOfficer.Exposed {
		t.Errorf("expected SAFETY_OFFICER to have access")
	}

	// Caller with non-matching role
	ctxGuest := authorizedContext("ten_alpha")
	ctxGuest.CallerRoles = []string{"GUEST_OBSERVER"}
	resGuest := registry.Evaluate("flag_role_scope", ctxGuest)
	if resGuest.Exposed {
		t.Errorf("expected GUEST_OBSERVER to be rejected")
	}
	if !strings.Contains(resGuest.Reason, "roles do not match") {
		t.Errorf("expected role mismatch reason, got: %s", resGuest.Reason)
	}
}

func TestFeatureFlag_AccessibilityQualificationFixtures(t *testing.T) {
	registry := reporting.NewFeatureFlagRegistry(nil)
	flag := defaultFlag("flag_a11y")
	flag.Enabled = true
	_ = registry.RegisterFlag(flag)

	res := registry.Evaluate("flag_a11y", authorizedContext("ten_alpha"))
	a11y := res.Accessibility

	if !a11y.KeyboardNavigable {
		t.Errorf("expected keyboard navigation to be certified")
	}
	if a11y.ScreenReaderLabel == "" {
		t.Errorf("expected non-empty screen reader label")
	}
	if a11y.AriaRole != "button" {
		t.Errorf("expected AriaRole 'button', got %s", a11y.AriaRole)
	}
	if !a11y.ContrastCertified {
		t.Errorf("expected contrast ratio certification")
	}
}

func TestFeatureFlag_InputValidation(t *testing.T) {
	registry := reporting.NewFeatureFlagRegistry(nil)

	// 1. Blank FlagID
	err := registry.RegisterFlag(reporting.FeatureFlag{
		FlagID:     "   ",
		DefaultOff: true,
	})
	if !errors.Is(err, reporting.ErrBlankFlagID) {
		t.Errorf("expected ErrBlankFlagID, got: %v", err)
	}

	// 2. Duplicate registration
	f := defaultFlag("flag_dup")
	_ = registry.RegisterFlag(f)
	err = registry.RegisterFlag(f)
	if !errors.Is(err, reporting.ErrDuplicateFlagID) {
		t.Errorf("expected ErrDuplicateFlagID, got: %v", err)
	}

	// 3. SetFlagState on non-existent flag
	err = registry.SetFlagState("nonexistent_flag", true)
	if !errors.Is(err, reporting.ErrFlagNotFound) {
		t.Errorf("expected ErrFlagNotFound, got: %v", err)
	}

	// 4. Evaluate non-existent flag
	res := registry.Evaluate("nonexistent_flag", authorizedContext("ten_alpha"))
	if res.Exposed {
		t.Errorf("expected false for missing flag")
	}
	if !strings.Contains(res.Reason, "flag not found") {
		t.Errorf("expected missing flag reason, got: %s", res.Reason)
	}
}
