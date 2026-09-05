package reporting

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// MetricGrain represents the temporal or aggregation grain of a metric.
type MetricGrain string

const (
	GrainInstant   MetricGrain = "INSTANT"
	GrainDaily     MetricGrain = "DAILY"
	GrainWeekly    MetricGrain = "WEEKLY"
	GrainMonthly   MetricGrain = "MONTHLY"
	GrainAggregate MetricGrain = "AGGREGATE"
)

// FreshnessDisposition describes whether a query result is considered fresh
// according to the metric's declared freshness boundary.
type FreshnessDisposition string

const (
	DispositionFresh    FreshnessDisposition = "FRESH"
	DispositionStale    FreshnessDisposition = "STALE"
	DispositionNotFresh FreshnessDisposition = "NOT_FRESH"
)

const (
	// DefaultNonAuthorityNotice is the standard required disclaimer for all derived reporting outputs.
	DefaultNonAuthorityNotice = "DERIVED_OUTPUT_NON_AUTHORITY: Reports, metrics, and analytics are derived outputs and never constitute operational authority or replace authoritative records."
)

var (
	ErrBlankTenantID       = errors.New("tenant ID cannot be blank")
	ErrBlankReaderID       = errors.New("reader ID cannot be blank")
	ErrBlankMetricID       = errors.New("metric ID cannot be blank")
	ErrUnknownMetric       = errors.New("metric not found in catalog")
	ErrDuplicateMetric     = errors.New("duplicate metric definition: cannot overwrite existing metric")
	ErrUnauthorizedReader  = errors.New("unauthorized reader identity for tenant")
	ErrUnsupportedFilter   = errors.New("unsupported filter for metric")
	ErrMissingNonAuthority = errors.New("metric must explicitly declare non-authority designation")
)

// MetricDefinition specifies the authoritative catalog contract for a metric.
type MetricDefinition struct {
	MetricID       string        `json:"metric_id"`
	Title          string        `json:"title"`
	Owner          string        `json:"owner"`
	Formula        string        `json:"formula"`
	DeclaredSource string        `json:"declared_source"`
	Grain          MetricGrain   `json:"grain"`
	AllowedFilters []string      `json:"allowed_filters"`
	FreshnessBound time.Duration `json:"freshness_bound"`
	Exclusions     []string      `json:"exclusions"`
	Limitations    []string      `json:"limitations"`
	NonAuthority   bool          `json:"non_authority"`
}

// InspectionComplianceScoreMetricID defines the canonical reporting projection metric for inspection compliance scores.
const InspectionComplianceScoreMetricID = "metric_inspection_compliance_score"

// NewInspectionComplianceScoreMetricDefinition returns the non-authoritative metric definition for compliance scores under HDEC-V040-SCORING-058.
func NewInspectionComplianceScoreMetricDefinition() MetricDefinition {
	return MetricDefinition{
		MetricID:       InspectionComplianceScoreMetricID,
		Title:          "Inspection Compliance Score Projection",
		Owner:          "compliance-analytics-team",
		Formula:        "MODEL_2_WEIGHTED (Normalized Section Compliance)",
		DeclaredSource: "MOD-WFA:inspections_v1",
		Grain:          GrainInstant,
		AllowedFilters: []string{"category", "status", "template_id"},
		FreshnessBound: 1 * time.Hour,
		Exclusions:     []string{"NA_excluded_from_denominator", "draft_inspections_excluded"},
		Limitations:    []string{"NON_AUTHORITATIVE_PROJECTION: operational source of truth remains MOD-WFA", "UNKNOWN_quarantined_from_denominator"},
		NonAuthority:   true,
	}
}

// QueryRequest contains caller parameters for evaluating a metric.
type QueryRequest struct {
	MetricID string            `json:"metric_id"`
	TenantID string            `json:"tenant_id"`
	ReaderID string            `json:"reader_id"`
	Filters  map[string]string `json:"filters"`
}

// QueryResult encapsulates the evaluated metric value, diagnostic metadata, and non-authority notices.
type QueryResult struct {
	MetricID             string               `json:"metric_id"`
	TenantID             string               `json:"tenant_id"`
	ReaderID             string               `json:"reader_id"`
	CalculatedValue      float64              `json:"calculated_value"`
	SampleCount          int                  `json:"sample_count"`
	EvaluatedAt          time.Time            `json:"evaluated_at"`
	SourceLastUpdatedAt  time.Time            `json:"source_last_updated_at"`
	FreshnessDisposition FreshnessDisposition `json:"freshness_disposition"`
	Exclusions           []string             `json:"exclusions"`
	Limitations          []string             `json:"limitations"`
	NonAuthorityNotice   string               `json:"non_authority_notice"`
}

// SyntheticRecord models source fixture data strictly within tenant scope.
type SyntheticRecord struct {
	TenantID  string            `json:"tenant_id"`
	Category  string            `json:"category"`
	Status    string            `json:"status"`
	Value     float64           `json:"value"`
	Tags      map[string]string `json:"tags"`
	Timestamp time.Time         `json:"timestamp"`
}

// ReportCatalog provides in-memory metric governance and deterministic query evaluation.
type ReportCatalog struct {
	mu          sync.RWMutex
	clock       func() time.Time
	metrics     map[string]MetricDefinition
	readers     map[string]map[string]bool   // tenantID -> readerID -> authorized
	fixtures    map[string][]SyntheticRecord // tenantID -> records
	sourceTimes map[string]time.Time         // tenantID -> last updated
}

// NewReportCatalog constructs a new in-memory catalog with an injectable clock.
func NewReportCatalog(clock func() time.Time) *ReportCatalog {
	if clock == nil {
		clock = time.Now
	}
	return &ReportCatalog{
		clock:       clock,
		metrics:     make(map[string]MetricDefinition),
		readers:     make(map[string]map[string]bool),
		fixtures:    make(map[string][]SyntheticRecord),
		sourceTimes: make(map[string]time.Time),
	}
}

// RegisterMetric registers an immutable metric definition.
// Fails closed if duplicate, missing IDs, or NonAuthority == false.
func (c *ReportCatalog) RegisterMetric(def MetricDefinition) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	id := strings.TrimSpace(def.MetricID)
	if id == "" {
		return ErrBlankMetricID
	}
	if _, exists := c.metrics[id]; exists {
		return ErrDuplicateMetric
	}

	if !def.NonAuthority {
		return ErrMissingNonAuthority
	}

	filtersCopy := make([]string, len(def.AllowedFilters))
	copy(filtersCopy, def.AllowedFilters)

	exclCopy := make([]string, len(def.Exclusions))
	copy(exclCopy, def.Exclusions)

	limitCopy := make([]string, len(def.Limitations))
	copy(limitCopy, def.Limitations)

	c.metrics[id] = MetricDefinition{
		MetricID:       id,
		Title:          strings.TrimSpace(def.Title),
		Owner:          strings.TrimSpace(def.Owner),
		Formula:        strings.TrimSpace(def.Formula),
		DeclaredSource: strings.TrimSpace(def.DeclaredSource),
		Grain:          def.Grain,
		AllowedFilters: filtersCopy,
		FreshnessBound: def.FreshnessBound,
		Exclusions:     exclCopy,
		Limitations:    limitCopy,
		NonAuthority:   true,
	}

	return nil
}

// AuthorizeReader grants read access for a reader identity within a tenant.
func (c *ReportCatalog) AuthorizeReader(tenantID, readerID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	tID := strings.TrimSpace(tenantID)
	if tID == "" {
		return ErrBlankTenantID
	}
	rID := strings.TrimSpace(readerID)
	if rID == "" {
		return ErrBlankReaderID
	}

	if c.readers[tID] == nil {
		c.readers[tID] = make(map[string]bool)
	}
	c.readers[tID][rID] = true
	return nil
}

// LoadFixtures populates synthetic fixture records strictly scoped to tenantID.
func (c *ReportCatalog) LoadFixtures(tenantID string, records []SyntheticRecord, sourceLastUpdated time.Time) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	tID := strings.TrimSpace(tenantID)
	if tID == "" {
		return ErrBlankTenantID
	}

	copied := make([]SyntheticRecord, len(records))
	for i, r := range records {
		copied[i] = r
		copied[i].TenantID = tID // enforce scope
	}

	c.fixtures[tID] = copied
	c.sourceTimes[tID] = sourceLastUpdated.UTC()
	return nil
}

// ExecuteQuery executes a deterministic query against synthetic fixtures.
// Fails closed on:
// - blank tenant ID
// - blank reader ID
// - unauthorized reader
// - unknown metric ID
// - undeclared filter parameters
func (c *ReportCatalog) ExecuteQuery(req QueryRequest) (QueryResult, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	tenantID := strings.TrimSpace(req.TenantID)
	if tenantID == "" {
		return QueryResult{}, ErrBlankTenantID
	}

	readerID := strings.TrimSpace(req.ReaderID)
	if readerID == "" {
		return QueryResult{}, ErrBlankReaderID
	}

	// 1. Reader authorization check
	tenantReaders, tenantFound := c.readers[tenantID]
	if !tenantFound || !tenantReaders[readerID] {
		return QueryResult{}, ErrUnauthorizedReader
	}

	// 2. Metric definition check
	metricID := strings.TrimSpace(req.MetricID)
	def, metricExists := c.metrics[metricID]
	if !metricExists {
		return QueryResult{}, ErrUnknownMetric
	}

	// 3. Allowed filters check
	allowedMap := make(map[string]bool)
	for _, f := range def.AllowedFilters {
		allowedMap[f] = true
	}
	for k := range req.Filters {
		if !allowedMap[k] {
			return QueryResult{}, fmt.Errorf("%w: %q is not in allowed filters %v", ErrUnsupportedFilter, k, def.AllowedFilters)
		}
	}

	// 4. Fetch tenant-scoped fixtures only (prevent cross-tenant leakage)
	records := c.fixtures[tenantID]
	sourceUpdated := c.sourceTimes[tenantID]
	now := c.clock().UTC()

	// 5. Evaluate matching records
	var sum float64
	var count int

	for _, rec := range records {
		if rec.TenantID != tenantID {
			continue // defensive isolation
		}

		// Apply filters
		matches := true
		for k, expectedVal := range req.Filters {
			switch k {
			case "category":
				if rec.Category != expectedVal {
					matches = false
				}
			case "status":
				if rec.Status != expectedVal {
					matches = false
				}
			default:
				if rec.Tags == nil || rec.Tags[k] != expectedVal {
					matches = false
				}
			}
			if !matches {
				break
			}
		}

		if matches {
			sum += rec.Value
			count++
		}
	}

	// Calculate average or sum based on formula
	var resultVal float64
	if count > 0 {
		if strings.Contains(strings.ToLower(def.Formula), "avg") {
			resultVal = sum / float64(count)
		} else {
			resultVal = sum
		}
	}

	// 6. Evaluate freshness disposition
	disposition := DispositionFresh
	if def.FreshnessBound > 0 && !sourceUpdated.IsZero() {
		age := now.Sub(sourceUpdated)
		if age > def.FreshnessBound {
			disposition = DispositionStale
		}
	} else if sourceUpdated.IsZero() {
		disposition = DispositionNotFresh
	}

	excl := make([]string, len(def.Exclusions))
	copy(excl, def.Exclusions)

	limit := make([]string, len(def.Limitations))
	copy(limit, def.Limitations)

	return QueryResult{
		MetricID:             metricID,
		TenantID:             tenantID,
		ReaderID:             readerID,
		CalculatedValue:      resultVal,
		SampleCount:          count,
		EvaluatedAt:          now,
		SourceLastUpdatedAt:  sourceUpdated,
		FreshnessDisposition: disposition,
		Exclusions:           excl,
		Limitations:          limit,
		NonAuthorityNotice:   DefaultNonAuthorityNotice,
	}, nil
}
