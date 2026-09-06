package reporting

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

// SYN-REP-01: Metric Query Calculation and Determinism
func TestQualification_SYN_REP_01_MetricCalculationDeterminism(t *testing.T) {
	t0 := time.Date(2026, 9, 6, 8, 0, 0, 0, time.UTC)
	clock, _ := newReportingTestClock(t0)
	catalog := NewReportCatalog(clock)

	metricDef := MetricDefinition{
		MetricID:       "metric_syn_avg_score",
		Title:          "Synthetic Average Compliance Score",
		Owner:          "qa-team",
		Formula:        "AVG(score)",
		DeclaredSource: "MOD-SYN:records_v1",
		Grain:          GrainInstant,
		AllowedFilters: []string{"category", "status"},
		FreshnessBound: 2 * time.Hour,
		Exclusions:     []string{"NA_excluded"},
		Limitations:    []string{"PROJECTION_ONLY"},
		NonAuthority:   true,
	}
	if err := catalog.RegisterMetric(metricDef); err != nil {
		t.Fatalf("unexpected RegisterMetric error: %v", err)
	}

	tenantID := "tenant-alpha"
	readerID := "auditor-01"
	if err := catalog.AuthorizeReader(tenantID, readerID); err != nil {
		t.Fatalf("unexpected AuthorizeReader error: %v", err)
	}

	fixtures := []SyntheticRecord{
		{TenantID: tenantID, Category: "SAFETY", Status: "PASS", Value: 80.0, Timestamp: t0},
		{TenantID: tenantID, Category: "SAFETY", Status: "PASS", Value: 90.0, Timestamp: t0},
		{TenantID: tenantID, Category: "SAFETY", Status: "FAIL", Value: 70.0, Timestamp: t0},
	}
	if err := catalog.LoadFixtures(tenantID, fixtures, t0); err != nil {
		t.Fatalf("unexpected LoadFixtures error: %v", err)
	}

	req := QueryRequest{
		MetricID: "metric_syn_avg_score",
		TenantID: tenantID,
		ReaderID: readerID,
		Filters:  map[string]string{"category": "SAFETY"},
	}

	// Deterministic execution verification (run multiple times)
	for iter := 0; iter < 3; iter++ {
		res, err := catalog.ExecuteQuery(req)
		if err != nil {
			t.Fatalf("iter %d: unexpected ExecuteQuery error: %v", iter, err)
		}
		if res.SampleCount != 3 {
			t.Errorf("iter %d: expected SampleCount 3, got %d", iter, res.SampleCount)
		}
		expectedAvg := (80.0 + 90.0 + 70.0) / 3.0
		if res.CalculatedValue != expectedAvg {
			t.Errorf("iter %d: expected CalculatedValue %f, got %f", iter, expectedAvg, res.CalculatedValue)
		}
		if res.FreshnessDisposition != DispositionFresh {
			t.Errorf("iter %d: expected FRESH, got %v", iter, res.FreshnessDisposition)
		}
		if res.NonAuthorityNotice != DefaultNonAuthorityNotice {
			t.Errorf("iter %d: missing non-authority notice: %q", iter, res.NonAuthorityNotice)
		}
	}
}

// SYN-REP-02: Scoped Filter Application & Undeclared Filter Rejection
func TestQualification_SYN_REP_02_ScopedFiltersAndRejection(t *testing.T) {
	t0 := time.Date(2026, 9, 6, 8, 0, 0, 0, time.UTC)
	clock, _ := newReportingTestClock(t0)
	catalog := NewReportCatalog(clock)

	metricDef := MetricDefinition{
		MetricID:       "metric_filtered_score",
		Title:          "Filtered Score",
		Owner:          "qa-team",
		Formula:        "AVG(score)",
		Grain:          GrainDaily,
		AllowedFilters: []string{"category", "status"},
		NonAuthority:   true,
	}
	if err := catalog.RegisterMetric(metricDef); err != nil {
		t.Fatalf("RegisterMetric error: %v", err)
	}

	tenantID := "tenant-alpha"
	readerID := "auditor-01"
	_ = catalog.AuthorizeReader(tenantID, readerID)

	fixtures := []SyntheticRecord{
		{TenantID: tenantID, Category: "FIRE", Status: "PASS", Value: 100.0, Timestamp: t0},
		{TenantID: tenantID, Category: "FIRE", Status: "FAIL", Value: 50.0, Timestamp: t0},
		{TenantID: tenantID, Category: "ELECTRICAL", Status: "PASS", Value: 90.0, Timestamp: t0},
	}
	_ = catalog.LoadFixtures(tenantID, fixtures, t0)

	// Valid filter query
	res, err := catalog.ExecuteQuery(QueryRequest{
		MetricID: "metric_filtered_score",
		TenantID: tenantID,
		ReaderID: readerID,
		Filters:  map[string]string{"category": "FIRE", "status": "PASS"},
	})
	if err != nil {
		t.Fatalf("unexpected error for valid filter: %v", err)
	}
	if res.SampleCount != 1 || res.CalculatedValue != 100.0 {
		t.Errorf("expected 1 sample with value 100.0, got %d with %f", res.SampleCount, res.CalculatedValue)
	}

	// Undeclared filter query -> must fail closed
	_, err = catalog.ExecuteQuery(QueryRequest{
		MetricID: "metric_filtered_score",
		TenantID: tenantID,
		ReaderID: readerID,
		Filters:  map[string]string{"unregistered_param": "unsafe_injection"},
	})
	if err == nil {
		t.Fatal("expected error for undeclared filter, got nil")
	}
	if !errors.Is(err, ErrUnsupportedFilter) {
		t.Errorf("expected ErrUnsupportedFilter, got: %v", err)
	}
}

// SYN-REP-03: Freshness Boundary Evaluation (FRESH vs STALE vs NOT_FRESH)
func TestQualification_SYN_REP_03_FreshnessBoundaries(t *testing.T) {
	t0 := time.Date(2026, 9, 6, 8, 0, 0, 0, time.UTC)
	clock, advance := newReportingTestClock(t0)
	catalog := NewReportCatalog(clock)

	metricDef := MetricDefinition{
		MetricID:       "metric_freshness_check",
		Title:          "Freshness Check Metric",
		Owner:          "qa-team",
		Formula:        "AVG(score)",
		Grain:          GrainInstant,
		FreshnessBound: 1 * time.Hour,
		NonAuthority:   true,
	}
	_ = catalog.RegisterMetric(metricDef)

	tenantID := "tenant-alpha"
	readerID := "auditor-01"
	_ = catalog.AuthorizeReader(tenantID, readerID)

	// Case 1: Fresh (source updated at t0, evaluated at t0 + 15m)
	_ = catalog.LoadFixtures(tenantID, []SyntheticRecord{{TenantID: tenantID, Value: 10.0}}, t0)
	advance(15 * time.Minute)
	resFresh, err := catalog.ExecuteQuery(QueryRequest{
		MetricID: "metric_freshness_check",
		TenantID: tenantID,
		ReaderID: readerID,
	})
	if err != nil {
		t.Fatalf("ExecuteQuery error: %v", err)
	}
	if resFresh.FreshnessDisposition != DispositionFresh {
		t.Errorf("expected FRESH, got %v", resFresh.FreshnessDisposition)
	}

	// Case 2: Stale (advance past 1h bound to t0 + 75m)
	advance(60 * time.Minute)
	resStale, err := catalog.ExecuteQuery(QueryRequest{
		MetricID: "metric_freshness_check",
		TenantID: tenantID,
		ReaderID: readerID,
	})
	if err != nil {
		t.Fatalf("ExecuteQuery error: %v", err)
	}
	if resStale.FreshnessDisposition != DispositionStale {
		t.Errorf("expected STALE, got %v", resStale.FreshnessDisposition)
	}

	// Case 3: NotFresh (source timestamp zero)
	_ = catalog.LoadFixtures(tenantID, []SyntheticRecord{{TenantID: tenantID, Value: 10.0}}, time.Time{})
	resNotFresh, err := catalog.ExecuteQuery(QueryRequest{
		MetricID: "metric_freshness_check",
		TenantID: tenantID,
		ReaderID: readerID,
	})
	if err != nil {
		t.Fatalf("ExecuteQuery error: %v", err)
	}
	if resNotFresh.FreshnessDisposition != DispositionNotFresh {
		t.Errorf("expected NOT_FRESH, got %v", resNotFresh.FreshnessDisposition)
	}
}

// SYN-REP-04: Cross-Tenant Reader Authorization Boundary
func TestQualification_SYN_REP_04_CrossTenantReaderAuthorization(t *testing.T) {
	clock, _ := newReportingTestClock(time.Now())
	catalog := NewReportCatalog(clock)

	metricDef := defaultMetricDef("metric_auth_test")
	_ = catalog.RegisterMetric(metricDef)

	_ = catalog.AuthorizeReader("tenant-alpha", "reader-alpha")

	// Unauthorized reader on tenant-alpha
	_, err := catalog.ExecuteQuery(QueryRequest{
		MetricID: "metric_auth_test",
		TenantID: "tenant-alpha",
		ReaderID: "reader-rogue",
	})
	if !errors.Is(err, ErrUnauthorizedReader) {
		t.Errorf("expected ErrUnauthorizedReader, got: %v", err)
	}

	// Cross-tenant reader on tenant-beta (where reader-alpha is not authorized)
	_, err = catalog.ExecuteQuery(QueryRequest{
		MetricID: "metric_auth_test",
		TenantID: "tenant-beta",
		ReaderID: "reader-alpha",
	})
	if !errors.Is(err, ErrUnauthorizedReader) {
		t.Errorf("expected ErrUnauthorizedReader for tenant-beta, got: %v", err)
	}

	// Blank tenant
	_, err = catalog.ExecuteQuery(QueryRequest{
		MetricID: "metric_auth_test",
		TenantID: "",
		ReaderID: "reader-alpha",
	})
	if !errors.Is(err, ErrBlankTenantID) {
		t.Errorf("expected ErrBlankTenantID, got: %v", err)
	}

	// Blank reader
	_, err = catalog.ExecuteQuery(QueryRequest{
		MetricID: "metric_auth_test",
		TenantID: "tenant-alpha",
		ReaderID: "",
	})
	if !errors.Is(err, ErrBlankReaderID) {
		t.Errorf("expected ErrBlankReaderID, got: %v", err)
	}
}

// SYN-REP-05: Empty Record Export Rendering Robustness
func TestQualification_SYN_REP_05_EmptyRecordExportRobustness(t *testing.T) {
	renderer := NewReportRenderer(nil)

	jsonTmpl := ReportTemplate{
		TemplateID:      "tmpl-json-empty",
		VersionID:       "1.0.0",
		Title:           "Empty Export JSON",
		SupportedFormat: FormatJSON,
		Limitations:     []string{"SYNTHETIC_TEST_ONLY"},
	}
	textTmpl := ReportTemplate{
		TemplateID:      "tmpl-text-empty",
		VersionID:       "1.0.0",
		Title:           "Empty Export Text",
		SupportedFormat: FormatTextTable,
		Limitations:     []string{"SYNTHETIC_TEST_ONLY"},
	}
	_ = renderer.RegisterTemplate(jsonTmpl)
	_ = renderer.RegisterTemplate(textTmpl)

	// JSON empty test
	resJSON, err := renderer.RenderCompleteRecordExport(CompleteRecordExportRequest{
		TenantID:    "tenant-alpha",
		TemplateID:  "tmpl-json-empty",
		RequestedBy: "qa-user",
		Records:     []ReportRecord{},
	})
	if err != nil {
		t.Fatalf("RenderCompleteRecordExport error: %v", err)
	}
	if resJSON.Manifest.RecordCount != 0 {
		t.Errorf("expected 0 records, got %d", resJSON.Manifest.RecordCount)
	}
	if err := VerifyReportIntegrity(resJSON); err != nil {
		t.Errorf("empty report integrity check failed: %v", err)
	}

	// Text table empty test
	resText, err := renderer.RenderCompleteRecordExport(CompleteRecordExportRequest{
		TenantID:    "tenant-alpha",
		TemplateID:  "tmpl-text-empty",
		RequestedBy: "qa-user",
		Records:     []ReportRecord{},
	})
	if err != nil {
		t.Fatalf("RenderCompleteRecordExport text error: %v", err)
	}
	if !strings.Contains(resText.RenderedOutput, "(no records match the requested scope)") {
		t.Errorf("text export missing empty indicator, got: %s", resText.RenderedOutput)
	}
	if err := VerifyReportIntegrity(resText); err != nil {
		t.Errorf("empty text report integrity check failed: %v", err)
	}
}

// SYN-REP-06: Large Record Export Sorting & Completeness
func TestQualification_SYN_REP_06_LargeRecordExportCompleteness(t *testing.T) {
	renderer := NewReportRenderer(nil)

	tmpl := ReportTemplate{
		TemplateID:      "tmpl-large-export",
		VersionID:       "2.0.0",
		Title:           "Large Record Set Export",
		SupportedFormat: FormatJSON,
	}
	_ = renderer.RegisterTemplate(tmpl)

	// Generate 120 records in descending order
	totalRecords := 120
	var inputRecords []ReportRecord
	for i := totalRecords - 1; i >= 0; i-- {
		inputRecords = append(inputRecords, ReportRecord{
			RecordID:   fmt.Sprintf("rec-%04d", i),
			TenantID:   "tenant-large",
			RecordType: "INSPECTION",
			Version:    "1.0",
			State:      "COMPLETED",
			Content:    fmt.Sprintf("Inspection item content payload %d", i),
		})
	}

	report, err := renderer.RenderCompleteRecordExport(CompleteRecordExportRequest{
		TenantID:    "tenant-large",
		TemplateID:  "tmpl-large-export",
		RequestedBy: "lead-auditor",
		Records:     inputRecords,
	})
	if err != nil {
		t.Fatalf("RenderCompleteRecordExport failed: %v", err)
	}

	if report.Manifest.RecordCount != totalRecords {
		t.Fatalf("expected RecordCount %d, got %d", totalRecords, report.Manifest.RecordCount)
	}

	// Parse JSON output and verify ascending deterministic sorting
	type jsonExportPayload struct {
		Title       string         `json:"title"`
		Version     string         `json:"version"`
		TenantID    string         `json:"tenant_id"`
		RecordCount int            `json:"record_count"`
		Records     []ReportRecord `json:"records"`
	}
	var parsed jsonExportPayload
	if err := json.Unmarshal([]byte(report.RenderedOutput), &parsed); err != nil {
		t.Fatalf("failed to unmarshal rendered JSON: %v", err)
	}

	if len(parsed.Records) != totalRecords {
		t.Fatalf("expected %d records in JSON payload, got %d", totalRecords, len(parsed.Records))
	}

	for idx, rec := range parsed.Records {
		expectedID := fmt.Sprintf("rec-%04d", idx)
		if rec.RecordID != expectedID {
			t.Fatalf("sort invariant broken at index %d: expected %s, got %s", idx, expectedID, rec.RecordID)
		}
	}
}

// SYN-REP-07: Long Text Content Preservation Without Truncation
func TestQualification_SYN_REP_07_LongTextContentPreservation(t *testing.T) {
	renderer := NewReportRenderer(nil)

	tmpl := ReportTemplate{
		TemplateID:      "tmpl-long-text",
		VersionID:       "1.0.0",
		Title:           "Long Text Preservation Template",
		SupportedFormat: FormatJSON,
	}
	_ = renderer.RegisterTemplate(tmpl)

	// Create long payload (> 8KB) including multilingual and special characters
	var sb strings.Builder
	for i := 0; i < 100; i++ {
		sb.WriteString(fmt.Sprintf("Item %d: รายงานการตรวจสอบความปลอดภัย OSHA/TIS-18001; Detailed engineering notes with multiline log;\n", i))
	}
	longContent := sb.String()
	if len(longContent) < 8000 {
		t.Fatalf("test setup failure: expected payload > 8000 bytes, got %d", len(longContent))
	}

	record := ReportRecord{
		RecordID:   "rec-long-01",
		TenantID:   "tenant-long",
		RecordType: "FINDING_DOSSIER",
		Version:    "1.0",
		State:      "OPEN",
		Content:    longContent,
	}

	report, err := renderer.RenderCompleteRecordExport(CompleteRecordExportRequest{
		TenantID:    "tenant-long",
		TemplateID:  "tmpl-long-text",
		RequestedBy: "qa-engineer",
		Records:     []ReportRecord{record},
	})
	if err != nil {
		t.Fatalf("RenderCompleteRecordExport failed: %v", err)
	}

	// Unmarshal JSON and verify exact content preservation
	type jsonExportPayload struct {
		Title       string         `json:"title"`
		Version     string         `json:"version"`
		TenantID    string         `json:"tenant_id"`
		RecordCount int            `json:"record_count"`
		Records     []ReportRecord `json:"records"`
	}
	var parsed jsonExportPayload
	if err := json.Unmarshal([]byte(report.RenderedOutput), &parsed); err != nil {
		t.Fatalf("failed to unmarshal rendered JSON: %v", err)
	}

	if len(parsed.Records) != 1 {
		t.Fatalf("expected 1 record in export, got %d", len(parsed.Records))
	}
	if parsed.Records[0].Content != longContent {
		t.Errorf("rendered output content mismatch: expected length %d, got length %d", len(longContent), len(parsed.Records[0].Content))
	}
	if err := VerifyReportIntegrity(report); err != nil {
		t.Errorf("integrity verification failed for long text export: %v", err)
	}
}

// SYN-REP-08: Template Version and Generation Context Pinning
func TestQualification_SYN_REP_08_TemplateVersionAndContextPinning(t *testing.T) {
	genTime := time.Date(2026, 9, 6, 9, 15, 0, 0, time.UTC)
	renderer := NewReportRenderer(func() time.Time { return genTime })

	tmpl := ReportTemplate{
		TemplateID:      "tmpl-versioned-audit",
		VersionID:       "3.4.2",
		Title:           "Versioned Audit Report",
		SupportedFormat: FormatTextTable,
		Limitations:     []string{"PROJECTION_ONLY", "NON_AUTHORITATIVE"},
	}
	_ = renderer.RegisterTemplate(tmpl)

	report, err := renderer.RenderCompleteRecordExport(CompleteRecordExportRequest{
		TenantID:     "tenant-audit",
		TemplateID:   "tmpl-versioned-audit",
		RequestedBy:  "compliance-lead",
		GenerationAt: genTime,
		Records: []ReportRecord{
			{RecordID: "rec-01", TenantID: "tenant-audit", Content: "Sample text"},
		},
	})
	if err != nil {
		t.Fatalf("RenderCompleteRecordExport error: %v", err)
	}

	manifest := report.Manifest
	if manifest.TemplateVersionID != "3.4.2" {
		t.Errorf("expected TemplateVersionID '3.4.2', got '%s'", manifest.TemplateVersionID)
	}
	if !manifest.GeneratedAt.Equal(genTime) {
		t.Errorf("expected GeneratedAt %v, got %v", genTime, manifest.GeneratedAt)
	}
	if manifest.GeneratedBy != "compliance-lead" {
		t.Errorf("expected GeneratedBy 'compliance-lead', got '%s'", manifest.GeneratedBy)
	}
	if len(manifest.Limitations) != 2 || manifest.Limitations[0] != "PROJECTION_ONLY" {
		t.Errorf("unexpected limitations in manifest: %v", manifest.Limitations)
	}
	if manifest.NonAuthorityNotice != DefaultNonAuthorityNotice {
		t.Errorf("missing standard non-authority notice in manifest: %s", manifest.NonAuthorityNotice)
	}
}

// SYN-REP-09: Export Manifest Cryptographic SHA-256 Hashing
func TestQualification_SYN_REP_09_ExportManifestCryptographicHashing(t *testing.T) {
	renderer := NewReportRenderer(nil)

	tmpl := ReportTemplate{
		TemplateID:      "tmpl-crypto-hash",
		VersionID:       "1.0.0",
		Title:           "Crypto Hash Template",
		SupportedFormat: FormatJSON,
	}
	_ = renderer.RegisterTemplate(tmpl)

	records := []ReportRecord{
		{RecordID: "rec-01", TenantID: "tenant-hash", RecordType: "ACTION", Version: "1.0", State: "OPEN", Content: "Body A"},
		{RecordID: "rec-02", TenantID: "tenant-hash", RecordType: "ACTION", Version: "1.0", State: "CLOSED", Content: "Body B"},
	}

	report, err := renderer.RenderCompleteRecordExport(CompleteRecordExportRequest{
		TenantID:    "tenant-hash",
		TemplateID:  "tmpl-crypto-hash",
		RequestedBy: "security-auditor",
		Records:     records,
	})
	if err != nil {
		t.Fatalf("RenderCompleteRecordExport error: %v", err)
	}

	// Source data digest must be 64-character lowercase hex string
	if len(report.Manifest.SourceDataDigest) != 64 {
		t.Errorf("invalid source data digest length: %d", len(report.Manifest.SourceDataDigest))
	}
	// Rendered digest must match manual SHA-256 calculation of output
	h := sha256.Sum256([]byte(report.RenderedOutput))
	expectedRenderDigest := hex.EncodeToString(h[:])
	if report.Manifest.RenderedDigest != expectedRenderDigest {
		t.Errorf("manifest rendered digest mismatch: expected %s, got %s", expectedRenderDigest, report.Manifest.RenderedDigest)
	}
}

// SYN-REP-10: Tamper Detection via Digest Recomputation
func TestQualification_SYN_REP_10_TamperDetectionViaDigest(t *testing.T) {
	renderer := NewReportRenderer(nil)

	tmpl := ReportTemplate{
		TemplateID:      "tmpl-tamper-check",
		VersionID:       "1.0.0",
		Title:           "Tamper Check Template",
		SupportedFormat: FormatJSON,
	}
	_ = renderer.RegisterTemplate(tmpl)

	report, err := renderer.RenderCompleteRecordExport(CompleteRecordExportRequest{
		TenantID:    "tenant-tamper",
		TemplateID:  "tmpl-tamper-check",
		RequestedBy: "auditor",
		Records: []ReportRecord{
			{RecordID: "rec-tamper-01", TenantID: "tenant-tamper", Content: "Genuine Content"},
		},
	})
	if err != nil {
		t.Fatalf("RenderCompleteRecordExport error: %v", err)
	}

	// Untampered report must verify successfully
	if err := VerifyReportIntegrity(report); err != nil {
		t.Fatalf("expected valid integrity check, got: %v", err)
	}

	// Tampered report: alter a single byte in the rendered output
	tamperedReport := report
	tamperedReport.RenderedOutput = report.RenderedOutput + " "
	errTampered := VerifyReportIntegrity(tamperedReport)
	if errTampered == nil {
		t.Fatal("expected ErrReportTampered on altered output, got nil")
	}
	if !errors.Is(errTampered, ErrReportTampered) {
		t.Errorf("expected ErrReportTampered, got: %v", errTampered)
	}
}

// SYN-REP-11: Cross-Tenant Record Export Boundary Enforcement
func TestQualification_SYN_REP_11_CrossTenantRecordExportFailure(t *testing.T) {
	renderer := NewReportRenderer(nil)

	tmpl := ReportTemplate{
		TemplateID:      "tmpl-isolation",
		VersionID:       "1.0.0",
		Title:           "Isolation Template",
		SupportedFormat: FormatJSON,
	}
	_ = renderer.RegisterTemplate(tmpl)

	// Mix tenant-alpha and tenant-beta records
	mixedRecords := []ReportRecord{
		{RecordID: "rec-01", TenantID: "tenant-alpha", Content: "Alpha Record"},
		{RecordID: "rec-02", TenantID: "tenant-beta", Content: "Injected Beta Record"},
	}

	_, err := renderer.RenderCompleteRecordExport(CompleteRecordExportRequest{
		TenantID:    "tenant-alpha",
		TemplateID:  "tmpl-isolation",
		RequestedBy: "intruder-test",
		Records:     mixedRecords,
	})
	if err == nil {
		t.Fatal("expected ErrCrossTenantRecord on cross-tenant export, got nil")
	}
	if !errors.Is(err, ErrCrossTenantRecord) {
		t.Errorf("expected ErrCrossTenantRecord, got: %v", err)
	}
}

// SYN-REP-12: Non-Authority Declaration Enforcement
func TestQualification_SYN_REP_12_NonAuthorityDeclarationEnforcement(t *testing.T) {
	catalog := NewReportCatalog(nil)

	// Metric with NonAuthority == false must fail registration
	invalidMetric := MetricDefinition{
		MetricID:     "metric_unauthorized_authoritative",
		Title:        "Unlawfully Authoritative Metric",
		Owner:        "rogue-actor",
		Formula:      "AVG(score)",
		NonAuthority: false, // violates non-authority requirement
	}

	err := catalog.RegisterMetric(invalidMetric)
	if err == nil {
		t.Fatal("expected ErrMissingNonAuthority, got nil")
	}
	if !errors.Is(err, ErrMissingNonAuthority) {
		t.Errorf("expected ErrMissingNonAuthority, got: %v", err)
	}

	// Verify standard notice text
	expectedNotice := "DERIVED_OUTPUT_NON_AUTHORITY: Reports, metrics, and analytics are derived outputs and never constitute operational authority or replace authoritative records."
	if DefaultNonAuthorityNotice != expectedNotice {
		t.Errorf("DefaultNonAuthorityNotice discrepancy:\nexpected: %s\ngot:      %s", expectedNotice, DefaultNonAuthorityNotice)
	}
}
