package reporting_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	reporting "github.com/oshethai/oshe-platform/modules/reporting-localization"
)

func defaultRenderer() (*reporting.ReportRenderer, reporting.ReportTemplate) {
	renderer := reporting.NewReportRenderer(nil)
	tmpl := reporting.ReportTemplate{
		TemplateID:      "tmpl_inspection_export",
		VersionID:       "1.0.0",
		Title:           "Complete Safety Inspection Export",
		Description:     "Authoritative scoped export template for safety findings",
		SupportedFormat: reporting.FormatTextTable,
		Sections:        []string{"header", "records", "audit"},
		Limitations:     []string{"unverified_offline_sync_excluded", "derived_non_authority"},
	}
	_ = renderer.RegisterTemplate(tmpl)
	return renderer, tmpl
}

func sampleRecords(tenantID string) []reporting.ReportRecord {
	now := time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC)
	return []reporting.ReportRecord{
		{
			RecordID:   "rec_003",
			TenantID:   tenantID,
			RecordType: "inspection_finding",
			Version:    "1.0.0",
			State:      "CLOSED",
			Content:    "Scaffold handrail repaired and verified.",
			CreatedAt:  now,
		},
		{
			RecordID:   "rec_001",
			TenantID:   tenantID,
			RecordType: "inspection_finding",
			Version:    "1.0.0",
			State:      "OPEN",
			Content:    "Missing fire extinguisher on level 2.",
			CreatedAt:  now,
		},
		{
			RecordID:   "rec_002",
			TenantID:   tenantID,
			RecordType: "inspection_finding",
			Version:    "1.0.0",
			State:      "IN_REVIEW",
			Content:    "Electrical hazard near water pump.",
			CreatedAt:  now,
		},
	}
}

func TestReportRenderer_ReportFixtureAndManifest(t *testing.T) {
	renderer, tmpl := defaultRenderer()
	tenantID := "ten_export_alpha"

	req := reporting.CompleteRecordExportRequest{
		TenantID:     tenantID,
		TemplateID:   tmpl.TemplateID,
		RequestedBy:  "compliance_officer_alice",
		Records:      sampleRecords(tenantID),
		GenerationAt: time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC),
	}

	report, err := renderer.RenderCompleteRecordExport(req)
	if err != nil {
		t.Fatalf("RenderCompleteRecordExport failed: %v", err)
	}

	manifest := report.Manifest
	if manifest.TenantID != tenantID {
		t.Errorf("expected tenant %s, got %s", tenantID, manifest.TenantID)
	}
	if manifest.RecordCount != 3 {
		t.Errorf("expected record count 3, got %d", manifest.RecordCount)
	}
	if manifest.TemplateID != tmpl.TemplateID || manifest.TemplateVersionID != tmpl.VersionID {
		t.Errorf("template version mismatch: %+v", manifest)
	}
	if len(manifest.SourceDataDigest) != 64 || len(manifest.RenderedDigest) != 64 {
		t.Errorf("invalid digest lengths: source=%s, rendered=%s", manifest.SourceDataDigest, manifest.RenderedDigest)
	}
	if !strings.Contains(manifest.NonAuthorityNotice, "DERIVED_OUTPUT_NON_AUTHORITY") {
		t.Errorf("missing non-authority notice in manifest: %s", manifest.NonAuthorityNotice)
	}
	if len(manifest.Limitations) != 2 {
		t.Errorf("expected 2 limitations, got %d", len(manifest.Limitations))
	}

	// Verify report integrity
	if err := reporting.VerifyReportIntegrity(report); err != nil {
		t.Fatalf("VerifyReportIntegrity failed: %v", err)
	}
}

func TestReportRenderer_ExportReproducibility(t *testing.T) {
	renderer, tmpl := defaultRenderer()
	tenantID := "ten_reproduce"
	genTime := time.Date(2026, 9, 5, 15, 30, 0, 0, time.UTC)

	req := reporting.CompleteRecordExportRequest{
		TenantID:     tenantID,
		TemplateID:   tmpl.TemplateID,
		RequestedBy:  "auditor_bob",
		Records:      sampleRecords(tenantID),
		GenerationAt: genTime,
	}

	report1, err := renderer.RenderCompleteRecordExport(req)
	if err != nil {
		t.Fatalf("render 1 failed: %v", err)
	}

	report2, err := renderer.RenderCompleteRecordExport(req)
	if err != nil {
		t.Fatalf("render 2 failed: %v", err)
	}

	if report1.RenderedOutput != report2.RenderedOutput {
		t.Errorf("rendered output is not reproducible byte-for-byte")
	}
	if report1.Manifest.SourceDataDigest != report2.Manifest.SourceDataDigest {
		t.Errorf("source digests differ: %s vs %s", report1.Manifest.SourceDataDigest, report2.Manifest.SourceDataDigest)
	}
	if report1.Manifest.RenderedDigest != report2.Manifest.RenderedDigest {
		t.Errorf("rendered digests differ: %s vs %s", report1.Manifest.RenderedDigest, report2.Manifest.RenderedDigest)
	}
}

func TestReportRenderer_LongTextContent(t *testing.T) {
	renderer, tmpl := defaultRenderer()
	tenantID := "ten_longtext"

	// 25,000 characters of substantive technical text
	longBody := strings.Repeat("Detailed safety hazard finding and remediation instructions.\n", 400)
	records := []reporting.ReportRecord{
		{
			RecordID:   "rec_long_001",
			TenantID:   tenantID,
			RecordType: "incident_report",
			Version:    "1.0.0",
			State:      "INVESTIGATING",
			Content:    longBody,
			CreatedAt:  time.Now().UTC(),
		},
	}

	req := reporting.CompleteRecordExportRequest{
		TenantID:     tenantID,
		TemplateID:   tmpl.TemplateID,
		RequestedBy:  "investigator_lead",
		Records:      records,
		GenerationAt: time.Now().UTC(),
	}

	report, err := renderer.RenderCompleteRecordExport(req)
	if err != nil {
		t.Fatalf("long text export failed: %v", err)
	}

	if !strings.Contains(report.RenderedOutput, longBody) {
		t.Fatalf("rendered output truncated or corrupted long text")
	}

	if err := reporting.VerifyReportIntegrity(report); err != nil {
		t.Fatalf("integrity verification failed on long text: %v", err)
	}
}

func TestReportRenderer_EmptyData(t *testing.T) {
	renderer, tmpl := defaultRenderer()
	tenantID := "ten_empty"

	req := reporting.CompleteRecordExportRequest{
		TenantID:     tenantID,
		TemplateID:   tmpl.TemplateID,
		RequestedBy:  "auditor_empty",
		Records:      []reporting.ReportRecord{},
		GenerationAt: time.Now().UTC(),
	}

	report, err := renderer.RenderCompleteRecordExport(req)
	if err != nil {
		t.Fatalf("empty data render failed: %v", err)
	}

	if report.Manifest.RecordCount != 0 {
		t.Errorf("expected record count 0, got %d", report.Manifest.RecordCount)
	}
	if len(report.Manifest.SourceDataDigest) != 64 || len(report.Manifest.RenderedDigest) != 64 {
		t.Errorf("expected valid digests on empty report")
	}
	if err := reporting.VerifyReportIntegrity(report); err != nil {
		t.Fatalf("integrity failed on empty report: %v", err)
	}
}

func TestReportRenderer_LargeData(t *testing.T) {
	renderer, tmpl := defaultRenderer()
	tenantID := "ten_large"

	const recordCount = 200
	records := make([]reporting.ReportRecord, recordCount)
	now := time.Now().UTC()

	// Intentionally add in reverse order to test deterministic sorting
	for i := 0; i < recordCount; i++ {
		records[i] = reporting.ReportRecord{
			RecordID:   fmt.Sprintf("rec_%04d", recordCount-i),
			TenantID:   tenantID,
			RecordType: "bulk_inspection",
			Version:    "1.0.0",
			State:      "COMPLETED",
			Content:    fmt.Sprintf("Inspection item observation #%d", recordCount-i),
			CreatedAt:  now,
		}
	}

	req := reporting.CompleteRecordExportRequest{
		TenantID:     tenantID,
		TemplateID:   tmpl.TemplateID,
		RequestedBy:  "batch_processor",
		Records:      records,
		GenerationAt: now,
	}

	report, err := renderer.RenderCompleteRecordExport(req)
	if err != nil {
		t.Fatalf("large dataset render failed: %v", err)
	}

	if report.Manifest.RecordCount != recordCount {
		t.Errorf("expected %d records, got %d", recordCount, report.Manifest.RecordCount)
	}

	if err := reporting.VerifyReportIntegrity(report); err != nil {
		t.Fatalf("integrity check failed on large dataset: %v", err)
	}
}

func TestReportRenderer_TamperDetection(t *testing.T) {
	renderer, tmpl := defaultRenderer()
	tenantID := "ten_tamper"

	req := reporting.CompleteRecordExportRequest{
		TenantID:     tenantID,
		TemplateID:   tmpl.TemplateID,
		RequestedBy:  "compliance_qa",
		Records:      sampleRecords(tenantID),
		GenerationAt: time.Now().UTC(),
	}

	report, _ := renderer.RenderCompleteRecordExport(req)

	// Tamper with output
	tamperedReport := report
	tamperedReport.RenderedOutput = report.RenderedOutput + "\n[TAMPERED_INJECTED_RECORD]"

	err := reporting.VerifyReportIntegrity(tamperedReport)
	if !errors.Is(err, reporting.ErrReportTampered) {
		t.Fatalf("expected ErrReportTampered on tampered output, got: %v", err)
	}
}

func TestReportRenderer_TenantValidation(t *testing.T) {
	renderer, tmpl := defaultRenderer()

	// 1. Blank tenant
	_, err := renderer.RenderCompleteRecordExport(reporting.CompleteRecordExportRequest{
		TenantID:    "",
		TemplateID:  tmpl.TemplateID,
		RequestedBy: "user1",
	})
	if !errors.Is(err, reporting.ErrBlankTenantID) {
		t.Errorf("expected ErrBlankTenantID, got: %v", err)
	}

	// 2. Blank requester
	_, err = renderer.RenderCompleteRecordExport(reporting.CompleteRecordExportRequest{
		TenantID:    "ten_alpha",
		TemplateID:  tmpl.TemplateID,
		RequestedBy: "",
	})
	if !errors.Is(err, reporting.ErrBlankReaderID) {
		t.Errorf("expected ErrBlankReaderID, got: %v", err)
	}

	// 3. Cross-tenant record injection
	badRecords := []reporting.ReportRecord{
		{
			RecordID: "rec_foreign",
			TenantID: "ten_bravo", // foreign tenant
			Content:  "Secret data",
		},
	}
	_, err = renderer.RenderCompleteRecordExport(reporting.CompleteRecordExportRequest{
		TenantID:    "ten_alpha",
		TemplateID:  tmpl.TemplateID,
		RequestedBy: "user1",
		Records:     badRecords,
	})
	if !errors.Is(err, reporting.ErrCrossTenantRecord) {
		t.Errorf("expected ErrCrossTenantRecord on cross-tenant record injection, got: %v", err)
	}

	// 4. Missing template
	_, err = renderer.RenderCompleteRecordExport(reporting.CompleteRecordExportRequest{
		TenantID:    "ten_alpha",
		TemplateID:  "nonexistent_template",
		RequestedBy: "user1",
	})
	if !errors.Is(err, reporting.ErrTemplateNotFound) {
		t.Errorf("expected ErrTemplateNotFound, got: %v", err)
	}
}
