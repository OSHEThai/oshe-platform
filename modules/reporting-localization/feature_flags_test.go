package reporting_test

import (
	"errors"
	"fmt"
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

func TestFeatureFlag_RolloutPercentageBounds(t *testing.T) {
	registry := reporting.NewFeatureFlagRegistry(nil)

	// 1. Valid bounds: 0%, 50%, 100%
	validPercentages := []int{0, 50, 100}
	for _, pct := range validPercentages {
		flag := defaultFlag(fmt.Sprintf("flag_valid_pct_%d", pct))
		flag.Rollout.Percentage = pct
		if err := registry.RegisterFlag(flag); err != nil {
			t.Fatalf("expected valid percentage %d to register, got: %v", pct, err)
		}
	}

	// 2. Invalid bounds on registration: < 0 or > 100
	invalidPercentages := []int{-1, -50, 101, 200}
	for _, pct := range invalidPercentages {
		flag := defaultFlag(fmt.Sprintf("flag_invalid_pct_%d", pct))
		flag.Rollout.Percentage = pct
		err := registry.RegisterFlag(flag)
		if !errors.Is(err, reporting.ErrInvalidRolloutPercentage) {
			t.Fatalf("expected ErrInvalidRolloutPercentage for percentage %d, got: %v", pct, err)
		}
	}

	// 3. SetRolloutPercentage updates within valid bounds
	f := defaultFlag("flag_pct_update")
	_ = registry.RegisterFlag(f)

	validUpdates := []int{0, 25, 75, 100}
	for _, pct := range validUpdates {
		if err := registry.SetRolloutPercentage("flag_pct_update", pct); err != nil {
			t.Fatalf("expected valid SetRolloutPercentage %d, got: %v", pct, err)
		}
	}

	// 4. SetRolloutPercentage rejects invalid bounds
	for _, pct := range []int{-1, 101} {
		err := registry.SetRolloutPercentage("flag_pct_update", pct)
		if !errors.Is(err, reporting.ErrInvalidRolloutPercentage) {
			t.Fatalf("expected ErrInvalidRolloutPercentage on update with %d, got: %v", pct, err)
		}
	}

	// 5. SetRolloutPercentage fails on non-existent flag
	err := registry.SetRolloutPercentage("nonexistent_flag", 50)
	if !errors.Is(err, reporting.ErrFlagNotFound) {
		t.Fatalf("expected ErrFlagNotFound, got: %v", err)
	}
}

func TestFeatureFlag_DeterministicCohortAssignment(t *testing.T) {
	flagID := "flag_analytics_v2"
	subjectID := "synthetic_user_001"

	// 1. Same flag and subject evaluated 100 times must return identical bucket (strictly deterministic)
	baseBucket := reporting.ComputeCohortBucket(flagID, subjectID)
	if baseBucket < 0 || baseBucket >= 100 {
		t.Fatalf("expected bucket in [0, 99], got: %d", baseBucket)
	}

	for range 100 {
		b := reporting.ComputeCohortBucket(flagID, subjectID)
		if b != baseBucket {
			t.Fatalf("non-deterministic cohort bucket observed: got %d, expected %d", b, baseBucket)
		}
	}

	// 2. Distinct synthetic subjects produce varied distribution in [0, 99]
	bucketsSeen := make(map[int]bool)
	for i := range 50 {
		subj := fmt.Sprintf("synthetic_user_%03d", i)
		bucket := reporting.ComputeCohortBucket(flagID, subj)
		if bucket < 0 || bucket >= 100 {
			t.Fatalf("bucket out of bounds [0, 99] for subject %s: %d", subj, bucket)
		}
		bucketsSeen[bucket] = true
	}
	if len(bucketsSeen) < 10 {
		t.Errorf("expected diverse bucket distribution across 50 subjects, got only %d distinct buckets", len(bucketsSeen))
	}

	// 3. Different flag IDs with identical subject yield different buckets (independent flag cohorts)
	differentFlagBucket := reporting.ComputeCohortBucket("flag_export_csv", subjectID)
	if differentFlagBucket == baseBucket {
		altBucket := reporting.ComputeCohortBucket("flag_pdf_summary", subjectID)
		if altBucket == baseBucket {
			t.Errorf("expected flag independence in cohort hashing for subject %s", subjectID)
		}
	}
}

func TestFeatureFlag_CohortRolloutEvaluation(t *testing.T) {
	registry := reporting.NewFeatureFlagRegistry(nil)

	// 1. 0% Rollout -> Excludes all subjects
	flagZero := defaultFlag("flag_zero_pct")
	flagZero.Enabled = true
	flagZero.Rollout.Percentage = 0
	_ = registry.RegisterFlag(flagZero)

	resZero := registry.Evaluate("flag_zero_pct", authorizedContext("ten_alpha"))
	if resZero.Exposed {
		t.Errorf("expected 0%% rollout to exclude subject")
	}
	if !strings.Contains(resZero.Reason, "0%") {
		t.Errorf("expected 0%% reason, got: %s", resZero.Reason)
	}

	// 2. 100% Rollout -> Exposes subject when all other gates pass
	flagHundred := defaultFlag("flag_hundred_pct")
	flagHundred.Enabled = true
	flagHundred.Rollout.Percentage = 100
	_ = registry.RegisterFlag(flagHundred)

	resHundred := registry.Evaluate("flag_hundred_pct", authorizedContext("ten_alpha"))
	if !resHundred.Exposed {
		t.Errorf("expected 100%% rollout to expose subject, reason: %s", resHundred.Reason)
	}

	// 3. Fractional 50% Rollout: Find qualifying and non-qualifying synthetic subjects
	flagFifty := defaultFlag("flag_fifty_pct")
	flagFifty.Enabled = true
	flagFifty.Rollout.Percentage = 50
	_ = registry.RegisterFlag(flagFifty)

	var includedSubject, excludedSubject string
	var incBucket, excBucket int

	for i := range 100 {
		candidate := fmt.Sprintf("synthetic_subject_%03d", i)
		b := reporting.ComputeCohortBucket("flag_fifty_pct", candidate)
		if b < 50 && includedSubject == "" {
			includedSubject = candidate
			incBucket = b
		}
		if b >= 50 && excludedSubject == "" {
			excludedSubject = candidate
			excBucket = b
		}
		if includedSubject != "" && excludedSubject != "" {
			break
		}
	}

	if includedSubject == "" || excludedSubject == "" {
		t.Fatalf("failed to find synthetic subjects for 50%% boundary testing")
	}

	// Qualifying subject (< 50) -> Exposed: true
	ctxIncluded := authorizedContext("ten_alpha")
	ctxIncluded.SubjectID = includedSubject
	resInc := registry.Evaluate("flag_fifty_pct", ctxIncluded)
	if !resInc.Exposed {
		t.Errorf("expected subject %s (bucket %d) to be exposed under 50%% rollout, reason: %s", includedSubject, incBucket, resInc.Reason)
	}

	// Non-qualifying subject (>= 50) -> Exposed: false
	ctxExcluded := authorizedContext("ten_alpha")
	ctxExcluded.SubjectID = excludedSubject
	resExc := registry.Evaluate("flag_fifty_pct", ctxExcluded)
	if resExc.Exposed {
		t.Errorf("expected subject %s (bucket %d) to be excluded under 50%% rollout", excludedSubject, excBucket)
	}
	if !strings.Contains(resExc.Reason, "outside rollout percentage") {
		t.Errorf("expected outside rollout percentage reason, got: %s", resExc.Reason)
	}

	// 4. Missing subject ID on fractional rollout -> fail closed
	ctxBlankSubject := authorizedContext("ten_alpha")
	ctxBlankSubject.SubjectID = "   "
	resBlank := registry.Evaluate("flag_fifty_pct", ctxBlankSubject)
	if resBlank.Exposed {
		t.Errorf("expected blank subject ID to be excluded on fractional rollout")
	}
	if !strings.Contains(resBlank.Reason, "subject identifier required") {
		t.Errorf("expected subject identifier required reason, got: %s", resBlank.Reason)
	}
}

func TestFeatureFlag_CohortPreservesPriorGates(t *testing.T) {
	baseTime := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	registry := reporting.NewFeatureFlagRegistry(func() time.Time { return baseTime })

	flag := defaultFlag("flag_gates_check")
	flag.Enabled = true
	flag.Rollout.Percentage = 100
	flag.Rollout.AllowedTenants = []string{"ten_alpha"}
	flag.Rollout.AllowedRoles = []string{"SAFETY_OFFICER"}
	flag.Rollout.EffectiveFrom = baseTime.Add(1 * time.Hour)
	flag.Rollout.EffectiveTo = baseTime.Add(4 * time.Hour)
	_ = registry.RegisterFlag(flag)

	// 1. Unauthorized caller is denied regardless of cohort eligibility
	ctxUnauth := authorizedContext("ten_alpha")
	ctxUnauth.IsAuthorized = false
	ctxUnauth.EvaluationTime = baseTime.Add(2 * time.Hour)
	resUnauth := registry.Evaluate("flag_gates_check", ctxUnauth)
	if resUnauth.Exposed {
		t.Errorf("expected unauthorized caller to be denied")
	}
	if !strings.Contains(resUnauth.Reason, "cannot bypass security controls") {
		t.Errorf("expected auth denial reason, got: %s", resUnauth.Reason)
	}

	// 2. Disabled flag is denied regardless of cohort eligibility
	_ = registry.SetFlagState("flag_gates_check", false)
	ctxValid := authorizedContext("ten_alpha")
	ctxValid.EvaluationTime = baseTime.Add(2 * time.Hour)
	resDisabled := registry.Evaluate("flag_gates_check", ctxValid)
	if resDisabled.Exposed {
		t.Errorf("expected disabled flag to be denied")
	}
	if !strings.Contains(resDisabled.Reason, "safe fallback active") {
		t.Errorf("expected safe fallback reason, got: %s", resDisabled.Reason)
	}
	_ = registry.SetFlagState("flag_gates_check", true)

	// 3. Temporal window expiry is denied regardless of cohort eligibility
	ctxExpired := authorizedContext("ten_alpha")
	ctxExpired.EvaluationTime = baseTime.Add(5 * time.Hour)
	resExpired := registry.Evaluate("flag_gates_check", ctxExpired)
	if resExpired.Exposed {
		t.Errorf("expected expired window to be denied")
	}
	if !strings.Contains(resExpired.Reason, "stale configuration") {
		t.Errorf("expected stale configuration reason, got: %s", resExpired.Reason)
	}

	// 4. Tenant mismatch is denied regardless of cohort eligibility
	ctxTenantMismatch := authorizedContext("ten_bravo")
	ctxTenantMismatch.EvaluationTime = baseTime.Add(2 * time.Hour)
	resTenant := registry.Evaluate("flag_gates_check", ctxTenantMismatch)
	if resTenant.Exposed {
		t.Errorf("expected tenant mismatch to be denied")
	}
	if !strings.Contains(resTenant.Reason, "not included in rollout allowlist") {
		t.Errorf("expected tenant mismatch reason, got: %s", resTenant.Reason)
	}

	// 5. Role mismatch is denied regardless of cohort eligibility
	ctxRoleMismatch := authorizedContext("ten_alpha")
	ctxRoleMismatch.CallerRoles = []string{"VIEWER_ONLY"}
	ctxRoleMismatch.EvaluationTime = baseTime.Add(2 * time.Hour)
	resRole := registry.Evaluate("flag_gates_check", ctxRoleMismatch)
	if resRole.Exposed {
		t.Errorf("expected role mismatch to be denied")
	}
	if !strings.Contains(resRole.Reason, "roles do not match") {
		t.Errorf("expected role mismatch reason, got: %s", resRole.Reason)
	}
}
