package reporting

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

var (
	ErrTemplateNotFound  = errors.New("report template not found in catalog")
	ErrDuplicateTemplate = errors.New("duplicate report template ID")
	ErrReportTampered    = errors.New("report integrity verification failed: rendered digest does not match manifest")
	ErrCrossTenantRecord = errors.New("cross-tenant record in export request: all records must match tenant scope")
)

// ExportFormat specifies the output serialization format.
type ExportFormat string

const (
	FormatJSON      ExportFormat = "JSON"
	FormatTextTable ExportFormat = "TEXT_TABLE"
)

// ReportTemplate defines the formatting structure, sections, and declared limitations.
type ReportTemplate struct {
	TemplateID      string       `json:"template_id"`
	VersionID       string       `json:"version_id"`
	Title           string       `json:"title"`
	Description     string       `json:"description"`
	SupportedFormat ExportFormat `json:"supported_format"`
	Sections        []string     `json:"sections"`
	Limitations     []string     `json:"limitations"`
}

// ReportRecord represents an authoritative domain record supplied for export rendering.
type ReportRecord struct {
	RecordID   string            `json:"record_id"`
	TenantID   string            `json:"tenant_id"`
	RecordType string            `json:"record_type"`
	Version    string            `json:"version"`
	State      string            `json:"state"`
	Content    string            `json:"content"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	CreatedAt  time.Time         `json:"created_at"`
}

// CompleteRecordExportRequest contains all parameters required to render a complete-record export.
type CompleteRecordExportRequest struct {
	TenantID     string            `json:"tenant_id"`
	TemplateID   string            `json:"template_id"`
	RequestedBy  string            `json:"requested_by"`
	Records      []ReportRecord    `json:"records"`
	Filters      map[string]string `json:"filters,omitempty"`
	GenerationAt time.Time         `json:"generation_at"`
}

// GenerationManifest captures the provenance, data digest, and integrity links of a rendered report.
type GenerationManifest struct {
	ManifestID         string    `json:"manifest_id"`
	TenantID           string    `json:"tenant_id"`
	TemplateID         string    `json:"template_id"`
	TemplateVersionID  string    `json:"template_version_id"`
	GeneratedAt        time.Time `json:"generated_at"`
	GeneratedBy        string    `json:"generated_by"`
	RecordCount        int       `json:"record_count"`
	SourceDataDigest   string    `json:"source_data_digest"`
	RenderedDigest     string    `json:"rendered_digest"`
	NonAuthorityNotice string    `json:"non_authority_notice"`
	Limitations        []string  `json:"limitations"`
}

// RenderedReport packages the generation manifest alongside the rendered content string.
type RenderedReport struct {
	Manifest       GenerationManifest `json:"manifest"`
	RenderedOutput string             `json:"rendered_output"`
}

// ReportRenderer coordinates deterministic versioned report generation and manifest hashing.
type ReportRenderer struct {
	mu        sync.RWMutex
	templates map[string]ReportTemplate
	clock     func() time.Time
}

// NewReportRenderer constructs a new ReportRenderer.
func NewReportRenderer(clock func() time.Time) *ReportRenderer {
	if clock == nil {
		clock = time.Now
	}
	return &ReportRenderer{
		templates: make(map[string]ReportTemplate),
		clock:     clock,
	}
}

// RegisterTemplate registers an export template in the renderer.
func (r *ReportRenderer) RegisterTemplate(tmpl ReportTemplate) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	id := strings.TrimSpace(tmpl.TemplateID)
	if id == "" {
		return errors.New("template ID cannot be blank")
	}
	if _, exists := r.templates[id]; exists {
		return ErrDuplicateTemplate
	}

	limitCopy := make([]string, len(tmpl.Limitations))
	copy(limitCopy, tmpl.Limitations)

	secCopy := make([]string, len(tmpl.Sections))
	copy(secCopy, tmpl.Sections)

	r.templates[id] = ReportTemplate{
		TemplateID:      id,
		VersionID:       strings.TrimSpace(tmpl.VersionID),
		Title:           strings.TrimSpace(tmpl.Title),
		Description:     strings.TrimSpace(tmpl.Description),
		SupportedFormat: tmpl.SupportedFormat,
		Sections:        secCopy,
		Limitations:     limitCopy,
	}
	return nil
}

// calculateSourceDigest computes a deterministic SHA-256 digest over sorted records.
func calculateSourceDigest(records []ReportRecord) string {
	h := sha256.New()
	for _, rec := range records {
		h.Write([]byte(rec.RecordID))
		h.Write([]byte(rec.TenantID))
		h.Write([]byte(rec.RecordType))
		h.Write([]byte(rec.Version))
		h.Write([]byte(rec.State))
		h.Write([]byte(rec.Content))
	}
	return hex.EncodeToString(h.Sum(nil))
}

// RenderCompleteRecordExport renders a complete-record export with generation manifest.
func (r *ReportRenderer) RenderCompleteRecordExport(req CompleteRecordExportRequest) (RenderedReport, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tenantID := strings.TrimSpace(req.TenantID)
	if tenantID == "" {
		return RenderedReport{}, ErrBlankTenantID
	}
	requestedBy := strings.TrimSpace(req.RequestedBy)
	if requestedBy == "" {
		return RenderedReport{}, ErrBlankReaderID
	}

	tmplID := strings.TrimSpace(req.TemplateID)
	tmpl, exists := r.templates[tmplID]
	if !exists {
		return RenderedReport{}, ErrTemplateNotFound
	}

	// Filter and validate records strictly within tenant scope
	var filtered []ReportRecord
	for _, rec := range req.Records {
		if rec.TenantID != "" && rec.TenantID != tenantID {
			return RenderedReport{}, ErrCrossTenantRecord
		}

		// Apply optional filters
		matches := true
		for k, expectedVal := range req.Filters {
			if rec.Metadata == nil || rec.Metadata[k] != expectedVal {
				matches = false
				break
			}
		}
		if matches {
			filtered = append(filtered, rec)
		}
	}

	// Deterministic sort by RecordID
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].RecordID < filtered[j].RecordID
	})

	now := req.GenerationAt
	if now.IsZero() {
		now = r.clock().UTC()
	}

	sourceDigest := calculateSourceDigest(filtered)

	// Render output according to template format
	var output string
	switch tmpl.SupportedFormat {
	case FormatTextTable:
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("=== %s (v%s) ===\n", tmpl.Title, tmpl.VersionID))
		sb.WriteString(fmt.Sprintf("Tenant: %s | GeneratedBy: %s | Records: %d\n\n", tenantID, requestedBy, len(filtered)))
		if len(filtered) == 0 {
			sb.WriteString("(no records match the requested scope)\n")
		} else {
			for _, rec := range filtered {
				sb.WriteString(fmt.Sprintf("[%s] Type: %s | Ver: %s | State: %s\n", rec.RecordID, rec.RecordType, rec.Version, rec.State))
				sb.WriteString(fmt.Sprintf("Content: %s\n\n", rec.Content))
			}
		}
		sb.WriteString("=== END OF REPORT ===\n")
		output = sb.String()
	default:
		// Default JSON output
		type jsonExportPayload struct {
			Title       string         `json:"title"`
			Version     string         `json:"version"`
			TenantID    string         `json:"tenant_id"`
			RecordCount int            `json:"record_count"`
			Records     []ReportRecord `json:"records"`
		}
		b, err := json.Marshal(jsonExportPayload{
			Title:       tmpl.Title,
			Version:     tmpl.VersionID,
			TenantID:    tenantID,
			RecordCount: len(filtered),
			Records:     filtered,
		})
		if err != nil {
			return RenderedReport{}, err
		}
		output = string(b)
	}

	// Compute rendered output digest
	hRender := sha256.Sum256([]byte(output))
	renderedDigest := hex.EncodeToString(hRender[:])

	limits := make([]string, len(tmpl.Limitations))
	copy(limits, tmpl.Limitations)

	manifest := GenerationManifest{
		ManifestID:         fmt.Sprintf("rep_%s_%d", tenantID, now.UnixNano()),
		TenantID:           tenantID,
		TemplateID:         tmpl.TemplateID,
		TemplateVersionID:  tmpl.VersionID,
		GeneratedAt:        now,
		GeneratedBy:        requestedBy,
		RecordCount:        len(filtered),
		SourceDataDigest:   sourceDigest,
		RenderedDigest:     renderedDigest,
		NonAuthorityNotice: DefaultNonAuthorityNotice,
		Limitations:        limits,
	}

	return RenderedReport{
		Manifest:       manifest,
		RenderedOutput: output,
	}, nil
}

// VerifyReportIntegrity recomputes the SHA-256 digest of RenderedOutput and matches against Manifest.
func VerifyReportIntegrity(report RenderedReport) error {
	h := sha256.Sum256([]byte(report.RenderedOutput))
	computed := hex.EncodeToString(h[:])

	if computed != report.Manifest.RenderedDigest {
		return ErrReportTampered
	}
	return nil
}
