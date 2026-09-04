package reporting

import (
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

func newReportingTestClock(initial time.Time) (func() time.Time, func(d time.Duration)) {
	curr := initial
	var mu sync.Mutex
	clock := func() time.Time {
		mu.Lock()
		defer mu.Unlock()
		return curr
	}
	advance := func(d time.Duration) {
		mu.Lock()
		defer mu.Unlock()
		curr = curr.Add(d)
	}
	return clock, advance
}

func defaultMetricDef(id string) MetricDefinition {
	return MetricDefinition{
		MetricID:       id,
		Title:          "Average Inspection Score",
		Owner:          "compliance-analytics-team",
		Formula:        "AVG(inspection_score)",
		DeclaredSource: "MOD-INS:inspections_v1",
		Grain:          GrainDaily,
		AllowedFilters: []string{"category", "status"},
		FreshnessBound: 1 * time.Hour,
		Exclusions:     []string{"drafts_excluded", "test_inspections_excluded"},
		Limitations:    []string{"historical_lookback_bounded_to_90_days"},
		NonAuthority:   true,
	}
}

func TestReproducibleFixtureComparison(t *testing.T) {
	c := NewReportCatalog(nil)
	def := defaultMetricDef("metric_avg_score")
	if err := c.RegisterMetric(def); err != nil {
		t.Fatalf("RegisterMetric failed: %v", err)
	}

	_ = c.AuthorizeReader("ten_alpha", "reader_alice")

	fixtures := []SyntheticRecord{
		{Category: "fire_safety", Status: "completed", Value: 80.0},
		{Category: "fire_safety", Status: "completed", Value: 90.0},
		{Category: "electrical", Status: "completed", Value: 70.0},
	}
	_ = c.LoadFixtures("ten_alpha", fixtures, time.Now().UTC())

	req := QueryRequest{
		MetricID: "metric_avg_score",
		TenantID: "ten_alpha",
		ReaderID: "reader_alice",
		Filters:  map[string]string{"category": "fire_safety"},
	}

	res1, err := c.ExecuteQuery(req)
	if err != nil {
		t.Fatalf("query 1 failed: %v", err)
	}

	res2, err := c.ExecuteQuery(req)
	if err != nil {
		t.Fatalf("query 2 failed: %v", err)
	}

	// Exact reproducible calculation
	if res1.CalculatedValue != 85.0 || res2.CalculatedValue != 85.0 {
		t.Errorf("expected reproducible value 85.0, got res1=%f, res2=%f", res1.CalculatedValue, res2.CalculatedValue)
	}
	if res1.SampleCount != 2 || res2.SampleCount != 2 {
		t.Errorf("expected sample count 2, got res1=%d, res2=%d", res1.SampleCount, res2.SampleCount)
	}
}

func TestCrossTenantDenial(t *testing.T) {
	c := NewReportCatalog(nil)
	_ = c.RegisterMetric(defaultMetricDef("metric_isolation"))

	_ = c.AuthorizeReader("ten_alpha", "reader_alpha")
	_ = c.AuthorizeReader("ten_bravo", "reader_bravo")

	_ = c.LoadFixtures("ten_alpha", []SyntheticRecord{
		{Category: "general", Status: "ok", Value: 100.0},
	}, time.Now().UTC())

	_ = c.LoadFixtures("ten_bravo", []SyntheticRecord{
		{Category: "general", Status: "ok", Value: 500.0},
	}, time.Now().UTC())

	// Reader alpha queries tenant alpha -> gets only alpha's 100.0
	resAlpha, err := c.ExecuteQuery(QueryRequest{
		MetricID: "metric_isolation",
		TenantID: "ten_alpha",
		ReaderID: "reader_alpha",
	})
	if err != nil {
		t.Fatalf("query alpha failed: %v", err)
	}
	if resAlpha.CalculatedValue != 100.0 || resAlpha.SampleCount != 1 {
		t.Errorf("expected 100.0 from tenant alpha, got %f", resAlpha.CalculatedValue)
	}

	// Reader alpha attempts to query tenant bravo -> denied
	_, err = c.ExecuteQuery(QueryRequest{
		MetricID: "metric_isolation",
		TenantID: "ten_bravo",
		ReaderID: "reader_alpha",
	})
	if !errors.Is(err, ErrUnauthorizedReader) {
		t.Fatalf("expected ErrUnauthorizedReader querying other tenant, got: %v", err)
	}
}

func TestUnauthorizedQueryDenial(t *testing.T) {
	c := NewReportCatalog(nil)
	_ = c.RegisterMetric(defaultMetricDef("metric_auth"))
	_ = c.AuthorizeReader("ten_alpha", "authorized_reader")

	// Blank tenant
	_, err := c.ExecuteQuery(QueryRequest{
		MetricID: "metric_auth",
		TenantID: "",
		ReaderID: "authorized_reader",
	})
	if !errors.Is(err, ErrBlankTenantID) {
		t.Errorf("expected ErrBlankTenantID, got: %v", err)
	}

	// Blank reader
	_, err = c.ExecuteQuery(QueryRequest{
		MetricID: "metric_auth",
		TenantID: "ten_alpha",
		ReaderID: "",
	})
	if !errors.Is(err, ErrBlankReaderID) {
		t.Errorf("expected ErrBlankReaderID, got: %v", err)
	}

	// Unknown reader
	_, err = c.ExecuteQuery(QueryRequest{
		MetricID: "metric_auth",
		TenantID: "ten_alpha",
		ReaderID: "intruder",
	})
	if !errors.Is(err, ErrUnauthorizedReader) {
		t.Errorf("expected ErrUnauthorizedReader, got: %v", err)
	}
}

func TestAllowedFilterEnforcement(t *testing.T) {
	c := NewReportCatalog(nil)
	_ = c.RegisterMetric(defaultMetricDef("metric_filters"))
	_ = c.AuthorizeReader("ten_alpha", "reader_alice")
	_ = c.LoadFixtures("ten_alpha", []SyntheticRecord{{Value: 10}}, time.Now().UTC())

	// Valid filter: category
	_, err := c.ExecuteQuery(QueryRequest{
		MetricID: "metric_filters",
		TenantID: "ten_alpha",
		ReaderID: "reader_alice",
		Filters:  map[string]string{"category": "safety"},
	})
	if err != nil {
		t.Fatalf("expected success on allowed filter, got: %v", err)
	}

	// Invalid filter: severity (not in allowed filters)
	_, err = c.ExecuteQuery(QueryRequest{
		MetricID: "metric_filters",
		TenantID: "ten_alpha",
		ReaderID: "reader_alice",
		Filters:  map[string]string{"severity": "critical"},
	})
	if !errors.Is(err, ErrUnsupportedFilter) {
		t.Fatalf("expected ErrUnsupportedFilter, got: %v", err)
	}
}

func TestStaleMetricVisibility(t *testing.T) {
	baseTime := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	clock, advance := newReportingTestClock(baseTime)

	c := NewReportCatalog(clock)
	def := defaultMetricDef("metric_stale")
	def.FreshnessBound = 1 * time.Hour
	_ = c.RegisterMetric(def)
	_ = c.AuthorizeReader("ten_alpha", "reader_alice")

	// Source was updated 10 minutes before baseTime -> FRESH
	_ = c.LoadFixtures("ten_alpha", []SyntheticRecord{{Value: 50}}, baseTime.Add(-10*time.Minute))

	resFresh, err := c.ExecuteQuery(QueryRequest{
		MetricID: "metric_stale",
		TenantID: "ten_alpha",
		ReaderID: "reader_alice",
	})
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if resFresh.FreshnessDisposition != DispositionFresh {
		t.Errorf("expected DispositionFresh, got: %s", resFresh.FreshnessDisposition)
	}

	// Advance clock by 2 hours -> now data is STALE (> 1h freshness bound)
	advance(2 * time.Hour)

	resStale, err := c.ExecuteQuery(QueryRequest{
		MetricID: "metric_stale",
		TenantID: "ten_alpha",
		ReaderID: "reader_alice",
	})
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if resStale.FreshnessDisposition != DispositionStale {
		t.Errorf("expected DispositionStale, got: %s", resStale.FreshnessDisposition)
	}
}

func TestExclusionsAndLimitationsPropagation(t *testing.T) {
	c := NewReportCatalog(nil)
	def := defaultMetricDef("metric_props")
	_ = c.RegisterMetric(def)
	_ = c.AuthorizeReader("ten_alpha", "reader_alice")
	_ = c.LoadFixtures("ten_alpha", []SyntheticRecord{{Value: 10}}, time.Now().UTC())

	res, err := c.ExecuteQuery(QueryRequest{
		MetricID: "metric_props",
		TenantID: "ten_alpha",
		ReaderID: "reader_alice",
	})
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}

	if len(res.Exclusions) != 2 || res.Exclusions[0] != "drafts_excluded" {
		t.Errorf("unexpected exclusions: %v", res.Exclusions)
	}
	if len(res.Limitations) != 1 || res.Limitations[0] != "historical_lookback_bounded_to_90_days" {
		t.Errorf("unexpected limitations: %v", res.Limitations)
	}
}

func TestDerivedResultNonAuthority(t *testing.T) {
	c := NewReportCatalog(nil)

	// Attempting to register metric without non-authority designation fails closed
	badDef := defaultMetricDef("bad_metric")
	badDef.NonAuthority = false
	err := c.RegisterMetric(badDef)
	if !errors.Is(err, ErrMissingNonAuthority) {
		t.Fatalf("expected ErrMissingNonAuthority, got: %v", err)
	}

	// Valid metric query returns explicit non-authority notice
	goodDef := defaultMetricDef("good_metric")
	_ = c.RegisterMetric(goodDef)
	_ = c.AuthorizeReader("ten_alpha", "reader_alice")
	_ = c.LoadFixtures("ten_alpha", []SyntheticRecord{{Value: 10}}, time.Now().UTC())

	res, err := c.ExecuteQuery(QueryRequest{
		MetricID: "good_metric",
		TenantID: "ten_alpha",
		ReaderID: "reader_alice",
	})
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}

	if !strings.HasPrefix(res.NonAuthorityNotice, "DERIVED_OUTPUT_NON_AUTHORITY") {
		t.Errorf("expected NonAuthorityNotice to start with DERIVED_OUTPUT_NON_AUTHORITY, got: %s", res.NonAuthorityNotice)
	}
}

func TestWalkingSkeleton_InspectionSummaryReport(t *testing.T) {
	evalTime := time.Date(2026, 9, 5, 14, 0, 0, 0, time.UTC)
	catalog := NewReportCatalog(func() time.Time { return evalTime })

	metric := MetricDefinition{
		MetricID:       "metric_ws_inspection_closure_rate",
		Title:          "Walking Skeleton Inspection Closure Rate",
		Owner:          "MOD-REP",
		Formula:        "avg(closure_rate)",
		DeclaredSource: "MOD-WFA:inspections_walking_skeleton",
		Grain:          GrainDaily,
		AllowedFilters: []string{"checklist_version", "status", "site_id"},
		FreshnessBound: 2 * time.Hour,
		Exclusions:     []string{"draft_inspections_excluded", "unverified_findings_excluded"},
		Limitations:    []string{"internal_engineering_alpha_only", "no_public_service_commitment"},
		NonAuthority:   true,
	}

	if err := catalog.RegisterMetric(metric); err != nil {
		t.Fatalf("failed to register walking skeleton metric: %v", err)
	}

	tenantID := "ten_walking_skeleton_alpha"
	readerID := "reader_qa_inspector"
	if err := catalog.AuthorizeReader(tenantID, readerID); err != nil {
		t.Fatalf("failed to authorize reader: %v", err)
	}

	sourceTime := evalTime.Add(-30 * time.Minute)
	records := []SyntheticRecord{
		{
			TenantID:  tenantID,
			Category:  "safety",
			Status:    "CLOSED",
			Value:     100.0,
			Tags:      map[string]string{"checklist_version": "v1.0", "site_id": "site_east_1"},
			Timestamp: sourceTime,
		},
		{
			TenantID:  tenantID,
			Category:  "safety",
			Status:    "CLOSED",
			Value:     80.0,
			Tags:      map[string]string{"checklist_version": "v1.0", "site_id": "site_east_1"},
			Timestamp: sourceTime,
		},
		{
			TenantID:  tenantID,
			Category:  "safety",
			Status:    "OPEN",
			Value:     50.0,
			Tags:      map[string]string{"checklist_version": "v1.0", "site_id": "site_east_1"},
			Timestamp: sourceTime,
		},
	}
	if err := catalog.LoadFixtures(tenantID, records, sourceTime); err != nil {
		t.Fatalf("failed to load walking skeleton fixtures: %v", err)
	}

	res, err := catalog.ExecuteQuery(QueryRequest{
		MetricID: metric.MetricID,
		TenantID: tenantID,
		ReaderID: readerID,
		Filters:  map[string]string{"status": "CLOSED", "checklist_version": "v1.0"},
	})
	if err != nil {
		t.Fatalf("walking skeleton query failed: %v", err)
	}

	if res.CalculatedValue != 90.0 {
		t.Errorf("expected average closure rate 90.0, got %f", res.CalculatedValue)
	}
	if res.SampleCount != 2 {
		t.Errorf("expected sample count 2, got %d", res.SampleCount)
	}
	if res.FreshnessDisposition != DispositionFresh {
		t.Errorf("expected FRESH freshness disposition, got %s", res.FreshnessDisposition)
	}
	if len(res.Exclusions) != 2 || len(res.Limitations) != 2 {
		t.Errorf("expected exclusions and limitations to be propagated, got %v / %v", res.Exclusions, res.Limitations)
	}
	if !strings.Contains(res.NonAuthorityNotice, "DERIVED_OUTPUT_NON_AUTHORITY") {
		t.Errorf("expected NonAuthorityNotice on query result, got %s", res.NonAuthorityNotice)
	}
}

func TestWalkingSkeleton_CrossTenantQueryIsolation(t *testing.T) {
	catalog := NewReportCatalog(nil)
	metric := defaultMetricDef("metric_ws_isolation")
	_ = catalog.RegisterMetric(metric)

	tenantA := "ten_ws_tenant_a"
	tenantB := "ten_ws_tenant_b"
	readerA := "reader_tenant_a"
	_ = catalog.AuthorizeReader(tenantA, readerA)

	_ = catalog.LoadFixtures(tenantA, []SyntheticRecord{
		{Category: "fire", Status: "CLOSED", Value: 100.0},
	}, time.Now().UTC())

	_, err := catalog.ExecuteQuery(QueryRequest{
		MetricID: metric.MetricID,
		TenantID: tenantB,
		ReaderID: readerA,
	})
	if !errors.Is(err, ErrUnauthorizedReader) {
		t.Fatalf("expected ErrUnauthorizedReader for foreign tenant query, got: %v", err)
	}
}

func TestWalkingSkeleton_StaleMetricAndDisallowedFilterDenial(t *testing.T) {
	baseTime := time.Date(2026, 9, 5, 18, 0, 0, 0, time.UTC)
	clock, advance := newReportingTestClock(baseTime)

	catalog := NewReportCatalog(clock)
	metric := defaultMetricDef("metric_ws_stale")
	metric.FreshnessBound = 1 * time.Hour
	_ = catalog.RegisterMetric(metric)

	tenantID := "ten_ws_stale"
	readerID := "reader_ws_stale"
	_ = catalog.AuthorizeReader(tenantID, readerID)

	sourceTime := baseTime.Add(-10 * time.Minute)
	_ = catalog.LoadFixtures(tenantID, []SyntheticRecord{{Category: "safety", Value: 75.0}}, sourceTime)

	advance(3 * time.Hour)

	res, err := catalog.ExecuteQuery(QueryRequest{
		MetricID: metric.MetricID,
		TenantID: tenantID,
		ReaderID: readerID,
	})
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if res.FreshnessDisposition != DispositionStale {
		t.Errorf("expected DispositionStale, got: %s", res.FreshnessDisposition)
	}

	_, err = catalog.ExecuteQuery(QueryRequest{
		MetricID: metric.MetricID,
		TenantID: tenantID,
		ReaderID: readerID,
		Filters:  map[string]string{"unregistered_dimension": "inject"},
	})
	if !errors.Is(err, ErrUnsupportedFilter) {
		t.Errorf("expected ErrUnsupportedFilter, got: %v", err)
	}
}
