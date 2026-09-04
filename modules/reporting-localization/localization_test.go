package reporting_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	reporting "github.com/oshethai/oshe-platform/modules/reporting-localization"
)

func setupCatalog() *reporting.LocalizationCatalog {
	cat := reporting.NewLocalizationCatalog(reporting.LocaleEnUS)

	_ = cat.RegisterBundle(reporting.LocaleEnUS, map[string]string{
		"app.title":               "OSHE Safety Platform",
		"inspection.status.open":   "Pending Inspection",
		"inspection.status.closed": "Inspection Complete",
		"alert.hazard.fall":       "Fall Protection Warning",
		"metric.closure_rate":     "Average Closure Rate",
	})

	_ = cat.RegisterBundle(reporting.LocaleThTH, map[string]string{
		"app.title":               "แพลตฟอร์มความปลอดภัย OSHE",
		"inspection.status.open":   "รอการตรวจสอบความปลอดภัย",
		"inspection.status.closed": "ตรวจสอบเสร็จสิ้น",
		// "alert.hazard.fall" is intentionally omitted in Thai to test fallback
		"metric.closure_rate":     "อัตราการปิดข้อบกพร่องเฉลี่ย",
	})

	return cat
}

func TestLocalization_ThaiEnglishFixture(t *testing.T) {
	cat := setupCatalog()

	// 1. Thai resolution
	msgTh := cat.Resolve(reporting.LocaleThTH, "app.title")
	if msgTh.Disposition != reporting.DispositionExactMatch {
		t.Errorf("expected DispositionExactMatch, got %s", msgTh.Disposition)
	}
	if msgTh.Text != "แพลตฟอร์มความปลอดภัย OSHE" {
		t.Errorf("unexpected Thai text: %s", msgTh.Text)
	}
	if msgTh.FallbackUsed {
		t.Errorf("expected FallbackUsed=false for exact match")
	}

	// 2. English resolution
	msgEn := cat.Resolve(reporting.LocaleEnUS, "app.title")
	if msgEn.Disposition != reporting.DispositionExactMatch {
		t.Errorf("expected DispositionExactMatch, got %s", msgEn.Disposition)
	}
	if msgEn.Text != "OSHE Safety Platform" {
		t.Errorf("unexpected English text: %s", msgEn.Text)
	}
}

func TestLocalization_VisibleFallback(t *testing.T) {
	cat := setupCatalog()

	// "alert.hazard.fall" exists in en-US, but missing in th-TH
	msg := cat.Resolve(reporting.LocaleThTH, "alert.hazard.fall")
	if msg.Disposition != reporting.DispositionFallback {
		t.Errorf("expected DispositionFallback, got %s", msg.Disposition)
	}
	if !msg.FallbackUsed {
		t.Errorf("expected FallbackUsed=true")
	}
	if msg.Text != "Fall Protection Warning" {
		t.Errorf("expected English fallback text, got: %s", msg.Text)
	}
	if !strings.Contains(msg.MissingNotice, "fallen back to en-US") {
		t.Errorf("expected missing notice explaining fallback, got: %s", msg.MissingNotice)
	}
}

func TestLocalization_MissingTranslationVisibility(t *testing.T) {
	cat := setupCatalog()

	// Key is missing in both target and fallback
	msg := cat.Resolve(reporting.LocaleThTH, "unregistered.key.identifier")
	if msg.Disposition != reporting.DispositionMissing {
		t.Errorf("expected DispositionMissing, got %s", msg.Disposition)
	}
	if msg.Text != "[MISSING: unregistered.key.identifier]" {
		t.Errorf("expected visible missing placeholder, got: %s", msg.Text)
	}
	if !strings.Contains(msg.MissingNotice, "missing in both") {
		t.Errorf("expected missing notice details, got: %s", msg.MissingNotice)
	}
}

func TestLocalization_TimeZoneAndBuddhistEra(t *testing.T) {
	// Fixed UTC time: 2026-09-05 08:30:00 UTC
	// In Asia/Bangkok (UTC+7): 2026-09-05 15:30:00
	fixedTime := time.Date(2026, 9, 5, 8, 30, 0, 0, time.UTC)

	// 1. Thai Buddhist Era format in Asia/Bangkok
	thFormatted, err := reporting.FormatDateTime(fixedTime, reporting.LocaleThTH, "Asia/Bangkok")
	if err != nil {
		t.Fatalf("FormatDateTime th-TH failed: %v", err)
	}
	// BE year = 2026 + 543 = 2569
	expectedTh := "05/09/2569 15:30:00 (Asia/Bangkok)"
	if thFormatted != expectedTh {
		t.Errorf("expected %q, got %q", expectedTh, thFormatted)
	}

	// 2. English format in UTC
	enFormatted, err := reporting.FormatDateTime(fixedTime, reporting.LocaleEnUS, "UTC")
	if err != nil {
		t.Fatalf("FormatDateTime en-US failed: %v", err)
	}
	expectedEn := "2026-09-05 08:30:00 (UTC)"
	if enFormatted != expectedEn {
		t.Errorf("expected %q, got %q", expectedEn, enFormatted)
	}

	// 3. Invalid time zone check
	_, err = reporting.FormatDateTime(fixedTime, reporting.LocaleEnUS, "Invalid/Nonexistent_Zone")
	if !errors.Is(err, reporting.ErrInvalidTimeZone) {
		t.Errorf("expected ErrInvalidTimeZone, got: %v", err)
	}
}

func TestLocalization_FormatNumberAndUnits(t *testing.T) {
	// Number formatting
	val := 1234567.89
	str := reporting.FormatNumber(val, reporting.LocaleEnUS)
	if str != "1,234,567.89" {
		t.Errorf("expected 1,234,567.89, got %s", str)
	}

	negVal := -9876.0
	negStr := reporting.FormatNumber(negVal, reporting.LocaleEnUS)
	if negStr != "-9,876" {
		t.Errorf("expected -9,876, got %s", negStr)
	}

	// Unit formatting in Thai vs English
	unitMetersTh := reporting.FormatUnit(150, "meter", reporting.LocaleThTH)
	if unitMetersTh != "150 ม." {
		t.Errorf("expected '150 ม.', got %s", unitMetersTh)
	}

	unitMetersEn := reporting.FormatUnit(150, "meter", reporting.LocaleEnUS)
	if unitMetersEn != "150 m" {
		t.Errorf("expected '150 m', got %s", unitMetersEn)
	}

	unitKgTh := reporting.FormatUnit(75.5, "kg", reporting.LocaleThTH)
	if unitKgTh != "75.50 กก." {
		t.Errorf("expected '75.50 กก.', got %s", unitKgTh)
	}

	unitDbTh := reporting.FormatUnit(92, "db", reporting.LocaleThTH)
	if unitDbTh != "92 เดซิเบล" {
		t.Errorf("expected '92 เดซิเบล', got %s", unitDbTh)
	}
}

func TestLocalization_LongTextAndExpansion(t *testing.T) {
	cat := setupCatalog()

	enLong := strings.Repeat("Industrial safety compliance requirement documentation. ", 20)
	thLong := strings.Repeat("เอกสารข้อกำหนดการปฏิบัติตามมาตรฐานความปลอดภัยในโรงงานอุตสาหกรรม ", 20)

	_ = cat.RegisterTranslation(reporting.LocaleEnUS, "long.doc", enLong)
	_ = cat.RegisterTranslation(reporting.LocaleThTH, "long.doc", thLong)

	resolvedTh := cat.Resolve(reporting.LocaleThTH, "long.doc")
	if resolvedTh.Text != thLong {
		t.Errorf("long text resolution corrupted or truncated")
	}

	expansionRatio := reporting.CalculateTextExpansion(enLong, thLong)
	if expansionRatio <= 0.5 || expansionRatio >= 2.5 {
		t.Errorf("unexpected text expansion ratio: %f", expansionRatio)
	}
}

func TestLocalization_InputValidation(t *testing.T) {
	cat := reporting.NewLocalizationCatalog("")

	// Blank locale registration
	err := cat.RegisterTranslation("", "key1", "val1")
	if !errors.Is(err, reporting.ErrBlankLocale) {
		t.Errorf("expected ErrBlankLocale, got: %v", err)
	}

	// Blank key registration
	err = cat.RegisterTranslation(reporting.LocaleEnUS, "", "val1")
	if !errors.Is(err, reporting.ErrBlankKey) {
		t.Errorf("expected ErrBlankKey, got: %v", err)
	}

	// Empty key resolution
	msg := cat.Resolve(reporting.LocaleEnUS, "   ")
	if msg.Disposition != reporting.DispositionMissing || msg.Text != "[EMPTY_KEY]" {
		t.Errorf("expected [EMPTY_KEY] for whitespace key, got %+v", msg)
	}
}
