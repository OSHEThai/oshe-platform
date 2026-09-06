package reporting_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	reporting "github.com/oshethai/oshe-platform/modules/reporting-localization"
)

// setupQualificationLocalizationCatalog initializes a catalog populated with synthetic bundles.
func setupQualificationLocalizationCatalog() *reporting.LocalizationCatalog {
	cat := reporting.NewLocalizationCatalog(reporting.LocaleEnUS)

	_ = cat.RegisterBundle(reporting.LocaleEnUS, map[string]string{
		"nav.dashboard":           "Operational Safety Dashboard",
		"checklist.item.scaffold": "Scaffold Stability Verification",
		"finding.severity.high":   "High Hazard Finding",
		"alert.evacuation":        "Emergency Evacuation Order", // Present in EN, omitted in TH to test fallback
		"metric.inspection_score": "Inspection Compliance Score",
	})

	_ = cat.RegisterBundle(reporting.LocaleThTH, map[string]string{
		"nav.dashboard":           "แดชบอร์ดความปลอดภัยในการดำเนินงาน",
		"checklist.item.scaffold": "การตรวจสอบความมั่นคงของนั่งร้าน",
		"finding.severity.high":   "ข้อบกพร่องระดับอันตรายสูง",
		"metric.inspection_score": "คะแนนการปฏิบัติตามข้อกำหนดการตรวจสอบ",
	})

	return cat
}

// defaultQualificationFlag builds a standard feature flag for qualification testing.
func defaultQualificationFlag(id string) reporting.FeatureFlag {
	return reporting.FeatureFlag{
		FlagID:      id,
		Title:       "Dynamic Inspection Hazard Visualizer",
		Description: "Enables synthetic hazard hotspot mapping on field floorplans",
		DefaultOff:  true,
		Enabled:     true,
		Stage:       reporting.StageAlpha,
		Owner:       "field-ux-team",
		Rollout: reporting.RolloutMetadata{
			Percentage:     100,
			AllowedTenants: []string{"ten_qualification_alpha"},
			AllowedRoles:   []string{"SAFETY_INSPECTOR", "SITE_SUPERVISOR"},
		},
		Accessibility: reporting.AccessibilityMetadata{
			KeyboardNavigable: true,
			ScreenReaderLabel: "Toggle synthetic hazard hotspot map overlay",
			AriaRole:          "switch",
			ContrastCertified: true,
		},
	}
}

// TestV040Qualification_ExactMatchAndVisibleFallback validates Invariants 1, 2, and 3:
// Exact match, visible fallback with FallbackUsed=true and MissingNotice, and explicit visible missing indicators.
func TestV040Qualification_ExactMatchAndVisibleFallback(t *testing.T) {
	cat := setupQualificationLocalizationCatalog()

	// 1. Exact match in Thai (th-TH)
	resTh := cat.Resolve(reporting.LocaleThTH, "nav.dashboard")
	if resTh.Disposition != reporting.DispositionExactMatch {
		t.Errorf("expected DispositionExactMatch for th-TH, got: %s", resTh.Disposition)
	}
	if resTh.Text != "แดชบอร์ดความปลอดภัยในการดำเนินงาน" {
		t.Errorf("unexpected Thai text: %s", resTh.Text)
	}
	if resTh.FallbackUsed {
		t.Errorf("expected FallbackUsed=false for exact match")
	}

	// 2. Exact match in English (en-US)
	resEn := cat.Resolve(reporting.LocaleEnUS, "nav.dashboard")
	if resEn.Disposition != reporting.DispositionExactMatch {
		t.Errorf("expected DispositionExactMatch for en-US, got: %s", resEn.Disposition)
	}
	if resEn.Text != "Operational Safety Dashboard" {
		t.Errorf("unexpected English text: %s", resEn.Text)
	}

	// 3. Fallback: key present in en-US but missing in th-TH
	resFallback := cat.Resolve(reporting.LocaleThTH, "alert.evacuation")
	if resFallback.Disposition != reporting.DispositionFallback {
		t.Errorf("expected DispositionFallback, got: %s", resFallback.Disposition)
	}
	if !resFallback.FallbackUsed {
		t.Errorf("expected FallbackUsed=true")
	}
	if resFallback.Text != "Emergency Evacuation Order" {
		t.Errorf("expected English fallback text, got: %s", resFallback.Text)
	}
	if !strings.Contains(resFallback.MissingNotice, "fallen back to en-US") {
		t.Errorf("expected MissingNotice to explain fallback path, got: %s", resFallback.MissingNotice)
	}

	// 4. Missing key in both locales: visible missing indicator
	resMissing := cat.Resolve(reporting.LocaleThTH, "unregistered.synthetic.key")
	if resMissing.Disposition != reporting.DispositionMissing {
		t.Errorf("expected DispositionMissing, got: %s", resMissing.Disposition)
	}
	if resMissing.Text != "[MISSING: unregistered.synthetic.key]" {
		t.Errorf("expected visible [MISSING: key] placeholder, got: %s", resMissing.Text)
	}
	if !strings.Contains(resMissing.MissingNotice, "missing in both") {
		t.Errorf("expected detailed MissingNotice, got: %s", resMissing.MissingNotice)
	}
}

// TestV040Qualification_DateTimeUTCAndBuddhistEra validates Invariant 4:
// Dual time-zone rendering for UTC (CE) and Asia/Bangkok Buddhist Era (BE = CE + 543).
func TestV040Qualification_DateTimeUTCAndBuddhistEra(t *testing.T) {
	// Fixed UTC reference instant: 2026-09-06 02:30:00 UTC
	// In Asia/Bangkok (UTC+7): 2026-09-06 09:30:00
	fixedInstant := time.Date(2026, 9, 6, 2, 30, 0, 0, time.UTC)

	// 1. Thai Buddhist Era formatting in Asia/Bangkok
	thFormatted, err := reporting.FormatDateTime(fixedInstant, reporting.LocaleThTH, "Asia/Bangkok")
	if err != nil {
		t.Fatalf("FormatDateTime th-TH failed: %v", err)
	}
	// Expected BE year: 2026 + 543 = 2569
	expectedTh := "06/09/2569 09:30:00 (Asia/Bangkok)"
	if thFormatted != expectedTh {
		t.Errorf("expected Buddhist Era time %q, got %q", expectedTh, thFormatted)
	}

	// 2. English formatting in UTC
	enFormatted, err := reporting.FormatDateTime(fixedInstant, reporting.LocaleEnUS, "UTC")
	if err != nil {
		t.Fatalf("FormatDateTime en-US failed: %v", err)
	}
	expectedEn := "2026-09-06 02:30:00 (UTC)"
	if enFormatted != expectedEn {
		t.Errorf("expected UTC time %q, got %q", expectedEn, enFormatted)
	}

	// 3. Invalid time zone must fail closed with ErrInvalidTimeZone
	_, err = reporting.FormatDateTime(fixedInstant, reporting.LocaleEnUS, "Invalid/Unsupported_Zone")
	if !errors.Is(err, reporting.ErrInvalidTimeZone) {
		t.Errorf("expected ErrInvalidTimeZone for invalid zone, got: %v", err)
	}
}

// TestV040Qualification_UnitsAndLongTextExpansion validates Invariant 5:
// Localized unit labels, comma-separated numbers, and Thai script expansion tolerance.
func TestV040Qualification_UnitsAndLongTextExpansion(t *testing.T) {
	// 1. Number formatting with comma thousands separators
	val := 9876543.21
	formattedNum := reporting.FormatNumber(val, reporting.LocaleEnUS)
	if formattedNum != "9,876,543.21" {
		t.Errorf("expected '9,876,543.21', got %s", formattedNum)
	}

	// 2. Localized physical units
	if u := reporting.FormatUnit(25.4, "celsius", reporting.LocaleThTH); u != "25.40 °C" {
		t.Errorf("expected '25.40 °C', got %s", u)
	}
	if u := reporting.FormatUnit(120, "meter", reporting.LocaleThTH); u != "120 ม." {
		t.Errorf("expected '120 ม.', got %s", u)
	}
	if u := reporting.FormatUnit(120, "meter", reporting.LocaleEnUS); u != "120 m" {
		t.Errorf("expected '120 m', got %s", u)
	}
	if u := reporting.FormatUnit(85.5, "kg", reporting.LocaleThTH); u != "85.50 กก." {
		t.Errorf("expected '85.50 กก.', got %s", u)
	}
	if u := reporting.FormatUnit(95, "db", reporting.LocaleThTH); u != "95 เดซิเบล" {
		t.Errorf("expected '95 เดซิเบล', got %s", u)
	}

	// 3. Substantive technical long text expansion
	enText := "Mandatory fall protection barrier inspection requires physical anchoring verification and secondary safety lanyard confirmation."
	thText := "การตรวจสอบแผงกั้นป้องกันการตกจากที่สูงจำเป็นต้องได้รับการตรวจสอบจุดยึดทางกายภาพและการยืนยันการใช้งานสายช่วยชีวิตเพื่อความปลอดภัยขั้นที่สอง"

	ratio := reporting.CalculateTextExpansion(enText, thText)
	if ratio < 0.8 || ratio > 2.0 {
		t.Errorf("unexpected Thai-to-English expansion ratio: %f", ratio)
	}

	cat := setupQualificationLocalizationCatalog()
	_ = cat.RegisterTranslation(reporting.LocaleEnUS, "safety.finding.long", enText)
	_ = cat.RegisterTranslation(reporting.LocaleThTH, "safety.finding.long", thText)

	resolvedTh := cat.Resolve(reporting.LocaleThTH, "safety.finding.long")
	if resolvedTh.Text != thText {
		t.Errorf("long Thai text corrupted or truncated during resolution")
	}
}

// TestV040Qualification_FeatureFlagDefaultOff validates Invariant 6:
// All feature flags must declare DefaultOff=true and start disabled by default.
func TestV040Qualification_FeatureFlagDefaultOff(t *testing.T) {
	registry := reporting.NewFeatureFlagRegistry(nil)

	// Valid registration with DefaultOff = true
	flag := defaultQualificationFlag("flag_qual_default_off")
	flag.Enabled = false // Initially disabled
	if err := registry.RegisterFlag(flag); err != nil {
		t.Fatalf("RegisterFlag failed: %v", err)
	}

	ctx := reporting.EvaluationContext{
		TenantID:     "ten_qualification_alpha",
		SubjectID:    "usr_safety_lead_01",
		CallerRoles:  []string{"SAFETY_INSPECTOR"},
		IsAuthorized: true,
	}

	res := registry.Evaluate("flag_qual_default_off", ctx)
	if res.Exposed {
		t.Errorf("expected disabled flag to evaluate to false")
	}
	if !strings.Contains(res.Reason, "safe fallback active") {
		t.Errorf("expected safe fallback active reason, got: %s", res.Reason)
	}

	// Attempting to register flag with DefaultOff = false must fail closed
	badFlag := defaultQualificationFlag("flag_qual_bad_default")
	badFlag.DefaultOff = false
	err := registry.RegisterFlag(badFlag)
	if !errors.Is(err, reporting.ErrMustDefaultOff) {
		t.Fatalf("expected ErrMustDefaultOff when DefaultOff is false, got: %v", err)
	}
}

// TestV040Qualification_AuthorizationSeparation validates Invariant 7:
// Feature flags NEVER bypass authorization or grant security authority.
func TestV040Qualification_AuthorizationSeparation(t *testing.T) {
	registry := reporting.NewFeatureFlagRegistry(nil)

	flag := defaultQualificationFlag("flag_qual_auth_separation")
	flag.Enabled = true // Fully enabled in tenant
	_ = registry.RegisterFlag(flag)

	// Caller is explicitly NOT authorized (ctx.IsAuthorized = false)
	unauthCtx := reporting.EvaluationContext{
		TenantID:     "ten_qualification_alpha",
		SubjectID:    "usr_unauthorized_guest",
		CallerRoles:  []string{"GUEST"},
		IsAuthorized: false, // Unauthorized!
	}

	res := registry.Evaluate("flag_qual_auth_separation", unauthCtx)
	if res.Exposed {
		t.Errorf("CRITICAL SAFETY VIOLATION: feature flag exposed to unauthorized caller")
	}
	if !strings.Contains(res.Reason, "underlying authorization denied") {
		t.Errorf("expected authorization denied reason, got: %s", res.Reason)
	}
	if !strings.Contains(res.AuthorityNote, "FEATURE_FLAG_NON_AUTHORITY") {
		t.Errorf("missing non-authority notice in evaluation result: %s", res.AuthorityNote)
	}
}

// TestV040Qualification_AccessibilityMetadata validates Invariant 8:
// Feature flags carry structured accessibility metadata that propagates to evaluation output.
func TestV040Qualification_AccessibilityMetadata(t *testing.T) {
	registry := reporting.NewFeatureFlagRegistry(nil)

	flag := defaultQualificationFlag("flag_qual_a11y")
	_ = registry.RegisterFlag(flag)

	ctx := reporting.EvaluationContext{
		TenantID:     "ten_qualification_alpha",
		SubjectID:    "usr_inspector_elena",
		CallerRoles:  []string{"SAFETY_INSPECTOR"},
		IsAuthorized: true,
	}

	res := registry.Evaluate("flag_qual_a11y", ctx)
	if !res.Exposed {
		t.Fatalf("expected enabled flag to evaluate to true for authorized user: %s", res.Reason)
	}

	a11y := res.Accessibility
	if !a11y.KeyboardNavigable {
		t.Errorf("expected KeyboardNavigable=true")
	}
	if a11y.ScreenReaderLabel != "Toggle synthetic hazard hotspot map overlay" {
		t.Errorf("unexpected ScreenReaderLabel: %s", a11y.ScreenReaderLabel)
	}
	if a11y.AriaRole != "switch" {
		t.Errorf("expected AriaRole 'switch', got: %s", a11y.AriaRole)
	}
	if !a11y.ContrastCertified {
		t.Errorf("expected ContrastCertified=true")
	}
}

// TestV040Qualification_RolloutTemporalAndCohortBoundaries validates rollout temporal bounds and cohort hashing.
func TestV040Qualification_RolloutTemporalAndCohortBoundaries(t *testing.T) {
	baseTime := time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)
	registry := reporting.NewFeatureFlagRegistry(func() time.Time { return baseTime })

	flag := defaultQualificationFlag("flag_qual_temporal")
	flag.Rollout.EffectiveFrom = baseTime.Add(1 * time.Hour)
	flag.Rollout.EffectiveTo = baseTime.Add(48 * time.Hour)
	_ = registry.RegisterFlag(flag)

	// 1. Before effective window -> denied
	ctxBefore := reporting.EvaluationContext{
		TenantID:       "ten_qualification_alpha",
		SubjectID:      "usr_1",
		CallerRoles:    []string{"SAFETY_INSPECTOR"},
		IsAuthorized:   true,
		EvaluationTime: baseTime, // 1 hour before EffectiveFrom
	}
	resBefore := registry.Evaluate("flag_qual_temporal", ctxBefore)
	if resBefore.Exposed {
		t.Errorf("expected exposure denied before effective window")
	}
	if !strings.Contains(resBefore.Reason, "precedes rollout effective window") {
		t.Errorf("unexpected reason: %s", resBefore.Reason)
	}

	// 2. Deterministic cohort hashing
	bucket1 := reporting.ComputeCohortBucket("flag_cohort_test", "user_alpha")
	bucket2 := reporting.ComputeCohortBucket("flag_cohort_test", "user_alpha")
	if bucket1 != bucket2 {
		t.Errorf("cohort bucket assignment must be deterministic: %d vs %d", bucket1, bucket2)
	}
	if bucket1 < 0 || bucket1 >= 100 {
		t.Errorf("cohort bucket must be in [0, 99], got: %d", bucket1)
	}
}
