package api

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	orgtenancy "github.com/oshethai/oshe-platform/modules/organization-tenancy"
)

const (
	// DefaultNonAuthorityNotice must be embedded in all reports, exports, and analytics.
	DefaultNonAuthorityNotice = "DERIVED_OUTPUT_NON_AUTHORITY: Reports, exports, and analytics are derived outputs and never constitute operational authority or replace authoritative records."
)

var (
	ErrDigestMismatch       = errors.New("integrity verification failed: payload SHA-256 digest does not match expected hash")
	ErrTemplateNotPublished = errors.New("template not found or not in published state")
	ErrCrossTenantAccess    = errors.New("cross-tenant access denied: resource belongs to another tenant")
	ErrActionNotReviewable  = errors.New("action cannot be closed: requires reviewer role and valid preceding state")
)

// Question represents a prompt in a checklist template.
type Question struct {
	ID       string `json:"id"`
	Text     string `json:"text"`
	Type     string `json:"type"`
	Required bool   `json:"required"`
}

// PublishedTemplate models a published immutable checklist template.
type PublishedTemplate struct {
	TemplateID     string     `json:"template_id"`
	TenantID       string     `json:"tenant_id"`
	VersionID      string     `json:"version_id"`
	Title          string     `json:"title"`
	Questions      []Question `json:"questions"`
	SnapshotDigest string     `json:"snapshot_digest"`
	PublishedBy    string     `json:"published_by"`
	PublishedAt    time.Time  `json:"published_at"`
}

// ChecklistInstanceData models an active or completed checklist instance.
type ChecklistInstanceData struct {
	InstanceID  string            `json:"instance_id"`
	TenantID    string            `json:"tenant_id"`
	TemplateID  string            `json:"template_id"`
	VersionID   string            `json:"version_id"`
	InspectorID string            `json:"inspector_id"`
	State       string            `json:"state"` // "IN_PROGRESS", "COMPLETED"
	Answers     map[string]string `json:"answers"`
	CreatedAt   time.Time         `json:"created_at"`
	CompletedAt *time.Time        `json:"completed_at,omitempty"`
}

// EvidenceData represents an attached evidence item.
type EvidenceData struct {
	EvidenceID string    `json:"evidence_id"`
	TenantID   string    `json:"tenant_id"`
	InstanceID string    `json:"instance_id"`
	Filename   string    `json:"filename"`
	MediaType  string    `json:"media_type"`
	SizeBytes  int64     `json:"size_bytes"`
	SHA256     string    `json:"sha256"`
	UploadedBy string    `json:"uploaded_by"`
	UploadedAt time.Time `json:"uploaded_at"`
}

// ActionData models a corrective action derived from an inspection finding.
type ActionData struct {
	ActionID    string     `json:"action_id"`
	TenantID    string     `json:"tenant_id"`
	InstanceID  string     `json:"instance_id"`
	Title       string     `json:"title"`
	Owner       string     `json:"owner"`
	Reviewer    string     `json:"reviewer"`
	State       string     `json:"state"` // "ASSIGNED", "IN_REVIEW", "CLOSED"
	EvidenceIDs []string   `json:"evidence_ids"`
	CreatedAt   time.Time  `json:"created_at"`
	ClosedAt    *time.Time `json:"closed_at,omitempty"`
	ClosedBy    string     `json:"closed_by,omitempty"`
}

// ReportSummary encapsulates derived reporting metrics with mandatory non-authority notice.
type ReportSummary struct {
	ReportID           string    `json:"report_id"`
	TenantID           string    `json:"tenant_id"`
	GeneratedAt        time.Time `json:"generated_at"`
	InspectionsCount   int       `json:"inspections_count"`
	OpenActionsCount   int       `json:"open_actions_count"`
	ClosedActionsCount int       `json:"closed_actions_count"`
	ComplianceRate     float64   `json:"compliance_rate"`
	Freshness          string    `json:"freshness"`
	NonAuthorityNotice string    `json:"non_authority_notice"`
}

// ExportManifest packages complete-record export metadata with cryptographic hash.
type ExportManifest struct {
	ManifestID         string    `json:"manifest_id"`
	TenantID           string    `json:"tenant_id"`
	ExportedAt         time.Time `json:"exported_at"`
	ExportedBy         string    `json:"exported_by"`
	RecordCount        int       `json:"record_count"`
	SourceDigest       string    `json:"source_digest"`
	NonAuthorityNotice string    `json:"non_authority_notice"`
}

// AuditEventData represents an immutable historical record.
type AuditEventData struct {
	Sequence      int64     `json:"sequence"`
	TenantID      string    `json:"tenant_id"`
	EventType     string    `json:"event_type"`
	EntityID      string    `json:"entity_id"`
	ActorID       string    `json:"actor_id"`
	CorrelationID string    `json:"correlation_id"`
	Timestamp     time.Time `json:"timestamp"`
}

// WalkingSkeletonStore is a thread-safe, in-memory store partitioned strictly by tenant.
type WalkingSkeletonStore struct {
	mu          sync.RWMutex
	templates   map[string]PublishedTemplate      // tenantID:templateID
	instances   map[string]ChecklistInstanceData  // tenantID:instanceID
	evidence    map[string]EvidenceData           // tenantID:evidenceID
	actions     map[string]ActionData             // tenantID:actionID
	auditEvents []AuditEventData
	seqCounter  int64
	clock       func() time.Time
}

// NewWalkingSkeletonStore initializes a new empty in-memory store.
func NewWalkingSkeletonStore(clock func() time.Time) *WalkingSkeletonStore {
	if clock == nil {
		clock = time.Now
	}
	return &WalkingSkeletonStore{
		templates: make(map[string]PublishedTemplate),
		instances: make(map[string]ChecklistInstanceData),
		evidence:  make(map[string]EvidenceData),
		actions:   make(map[string]ActionData),
		clock:     clock,
	}
}

func makeScopedKey(tenantID, id string) string {
	return fmt.Sprintf("%s:%s", strings.TrimSpace(tenantID), strings.TrimSpace(id))
}

func (s *WalkingSkeletonStore) appendAudit(tenantID, eventType, entityID, actorID, corrID string) {
	s.seqCounter++
	if corrID == "" {
		corrID = fmt.Sprintf("corr_%d", s.seqCounter)
	}
	entry := AuditEventData{
		Sequence:      s.seqCounter,
		TenantID:      tenantID,
		EventType:     eventType,
		EntityID:      entityID,
		ActorID:       actorID,
		CorrelationID: corrID,
		Timestamp:     s.clock().UTC(),
	}
	s.auditEvents = append(s.auditEvents, entry)
}

// WalkingSkeletonServer wires HTTP endpoints under trusted TenantMiddleware.
type WalkingSkeletonServer struct {
	store    *WalkingSkeletonStore
	resolver ClaimsResolver
	mux      *http.ServeMux
}

// NewWalkingSkeletonServer builds and registers all walking skeleton HTTP handlers.
func NewWalkingSkeletonServer(store *WalkingSkeletonStore, resolver ClaimsResolver) *WalkingSkeletonServer {
	srv := &WalkingSkeletonServer{
		store:    store,
		resolver: resolver,
		mux:      http.NewServeMux(),
	}
	srv.registerRoutes()
	return srv
}

func (srv *WalkingSkeletonServer) registerRoutes() {
	srv.mux.HandleFunc("POST /api/v1/templates/publish", srv.handlePublishTemplate)
	srv.mux.HandleFunc("POST /api/v1/checklists/instantiate", srv.handleInstantiateChecklist)
	srv.mux.HandleFunc("POST /api/v1/checklists/complete", srv.handleCompleteChecklist)
	srv.mux.HandleFunc("POST /api/v1/evidence/upload", srv.handleUploadEvidence)
	srv.mux.HandleFunc("POST /api/v1/actions", srv.handleCreateAction)
	srv.mux.HandleFunc("POST /api/v1/actions/close", srv.handleCloseAction)
	srv.mux.HandleFunc("GET /api/v1/reports/summary", srv.handleGetReportSummary)
	srv.mux.HandleFunc("GET /api/v1/exports/records", srv.handleGetExportRecords)
	srv.mux.HandleFunc("GET /api/v1/audit/trail", srv.handleGetAuditTrail)
}

// Handler returns the HTTP handler wrapped with trusted TenantMiddleware.
func (srv *WalkingSkeletonServer) Handler() http.Handler {
	middleware := TenantMiddleware(srv.resolver)
	return middleware(srv.mux)
}

func (srv *WalkingSkeletonServer) getTenantAndClaims(r *http.Request) (orgtenancy.TenantContext, bool) {
	return TenantFromContext(r.Context())
}

// 1. Publish Template
func (srv *WalkingSkeletonServer) handlePublishTemplate(w http.ResponseWriter, r *http.Request) {
	tenantCtx, ok := srv.getTenantAndClaims(r)
	if !ok {
		writeErrorResponse(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing trusted tenant context")
		return
	}

	var req struct {
		TemplateID string     `json:"template_id"`
		VersionID  string     `json:"version_id"`
		Title      string     `json:"title"`
		Questions  []Question `json:"questions"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON payload")
		return
	}

	tID := strings.TrimSpace(req.TemplateID)
	if tID == "" || len(req.Questions) == 0 {
		writeErrorResponse(w, http.StatusBadRequest, "BAD_REQUEST", "template_id and questions are required")
		return
	}

	srv.store.mu.Lock()
	defer srv.store.mu.Unlock()

	tenantID := tenantCtx.TenantID()
	key := makeScopedKey(tenantID, tID)

	// Calculate snapshot digest
	b, _ := json.Marshal(req.Questions)
	h := sha256.Sum256(b)
	digest := hex.EncodeToString(h[:])

	now := srv.store.clock().UTC()
	tmpl := PublishedTemplate{
		TemplateID:     tID,
		TenantID:       tenantID,
		VersionID:      req.VersionID,
		Title:          req.Title,
		Questions:      req.Questions,
		SnapshotDigest: digest,
		PublishedBy:    tenantCtx.Subject(),
		PublishedAt:    now,
	}

	srv.store.templates[key] = tmpl
	srv.store.appendAudit(tenantID, "TEMPLATE_PUBLISHED", tID, tenantCtx.Subject(), "")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(tmpl)
}

// 2. Instantiate Checklist
func (srv *WalkingSkeletonServer) handleInstantiateChecklist(w http.ResponseWriter, r *http.Request) {
	tenantCtx, ok := srv.getTenantAndClaims(r)
	if !ok {
		writeErrorResponse(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing trusted tenant context")
		return
	}

	var req struct {
		InstanceID string `json:"instance_id"`
		TemplateID string `json:"template_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON payload")
		return
	}

	instID := strings.TrimSpace(req.InstanceID)
	tmplID := strings.TrimSpace(req.TemplateID)
	if instID == "" || tmplID == "" {
		writeErrorResponse(w, http.StatusBadRequest, "BAD_REQUEST", "instance_id and template_id are required")
		return
	}

	srv.store.mu.Lock()
	defer srv.store.mu.Unlock()

	tenantID := tenantCtx.TenantID()
	tmplKey := makeScopedKey(tenantID, tmplID)
	tmpl, exists := srv.store.templates[tmplKey]
	if !exists {
		writeErrorResponse(w, http.StatusNotFound, "NOT_FOUND", "template not found in tenant scope")
		return
	}

	instKey := makeScopedKey(tenantID, instID)
	now := srv.store.clock().UTC()
	instance := ChecklistInstanceData{
		InstanceID:  instID,
		TenantID:    tenantID,
		TemplateID:  tmplID,
		VersionID:   tmpl.VersionID,
		InspectorID: tenantCtx.Subject(),
		State:       "IN_PROGRESS",
		Answers:     make(map[string]string),
		CreatedAt:   now,
	}

	srv.store.instances[instKey] = instance
	srv.store.appendAudit(tenantID, "CHECKLIST_INSTANTIATED", instID, tenantCtx.Subject(), "")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(instance)
}

// 3. Complete Checklist
func (srv *WalkingSkeletonServer) handleCompleteChecklist(w http.ResponseWriter, r *http.Request) {
	tenantCtx, ok := srv.getTenantAndClaims(r)
	if !ok {
		writeErrorResponse(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing trusted tenant context")
		return
	}

	var req struct {
		InstanceID string            `json:"instance_id"`
		Answers    map[string]string `json:"answers"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON payload")
		return
	}

	srv.store.mu.Lock()
	defer srv.store.mu.Unlock()

	tenantID := tenantCtx.TenantID()
	instKey := makeScopedKey(tenantID, req.InstanceID)
	inst, exists := srv.store.instances[instKey]
	if !exists {
		writeErrorResponse(w, http.StatusNotFound, "NOT_FOUND", "instance not found in tenant scope")
		return
	}

	now := srv.store.clock().UTC()
	inst.State = "COMPLETED"
	inst.Answers = req.Answers
	inst.CompletedAt = &now

	srv.store.instances[instKey] = inst
	srv.store.appendAudit(tenantID, "CHECKLIST_COMPLETED", req.InstanceID, tenantCtx.Subject(), "")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(inst)
}

// 4. Upload Evidence (with SHA-256 Digest Verification)
func (srv *WalkingSkeletonServer) handleUploadEvidence(w http.ResponseWriter, r *http.Request) {
	tenantCtx, ok := srv.getTenantAndClaims(r)
	if !ok {
		writeErrorResponse(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing trusted tenant context")
		return
	}

	var req struct {
		EvidenceID     string `json:"evidence_id"`
		InstanceID     string `json:"instance_id"`
		Filename       string `json:"filename"`
		MediaType      string `json:"media_type"`
		Payload        string `json:"payload"`
		AssertedDigest string `json:"asserted_digest"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON payload")
		return
	}

	evID := strings.TrimSpace(req.EvidenceID)
	instID := strings.TrimSpace(req.InstanceID)
	if evID == "" || instID == "" {
		writeErrorResponse(w, http.StatusBadRequest, "BAD_REQUEST", "evidence_id and instance_id are required")
		return
	}

	// Verify SHA-256 digest
	h := sha256.Sum256([]byte(req.Payload))
	computed := hex.EncodeToString(h[:])
	asserted := strings.ToLower(strings.TrimSpace(req.AssertedDigest))

	if asserted != "" && computed != asserted {
		writeErrorResponse(w, http.StatusBadRequest, "DIGEST_MISMATCH", "asserted digest does not match computed payload digest")
		return
	}

	srv.store.mu.Lock()
	defer srv.store.mu.Unlock()

	tenantID := tenantCtx.TenantID()

	// Verify associated instance is within tenant scope
	instKey := makeScopedKey(tenantID, instID)
	if _, exists := srv.store.instances[instKey]; !exists {
		writeErrorResponse(w, http.StatusNotFound, "NOT_FOUND", "associated checklist instance not found in tenant scope")
		return
	}

	now := srv.store.clock().UTC()
	ev := EvidenceData{
		EvidenceID: evID,
		TenantID:   tenantID,
		InstanceID: instID,
		Filename:   req.Filename,
		MediaType:  req.MediaType,
		SizeBytes:  int64(len(req.Payload)),
		SHA256:     computed,
		UploadedBy: tenantCtx.Subject(),
		UploadedAt: now,
	}

	evKey := makeScopedKey(tenantID, evID)
	srv.store.evidence[evKey] = ev
	srv.store.appendAudit(tenantID, "EVIDENCE_UPLOADED", evID, tenantCtx.Subject(), "")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(ev)
}

// 5. Create Action
func (srv *WalkingSkeletonServer) handleCreateAction(w http.ResponseWriter, r *http.Request) {
	tenantCtx, ok := srv.getTenantAndClaims(r)
	if !ok {
		writeErrorResponse(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing trusted tenant context")
		return
	}

	var req struct {
		ActionID    string   `json:"action_id"`
		InstanceID  string   `json:"instance_id"`
		Title       string   `json:"title"`
		Owner       string   `json:"owner"`
		Reviewer    string   `json:"reviewer"`
		EvidenceIDs []string `json:"evidence_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON payload")
		return
	}

	actID := strings.TrimSpace(req.ActionID)
	if actID == "" || req.Owner == "" || req.Reviewer == "" {
		writeErrorResponse(w, http.StatusBadRequest, "BAD_REQUEST", "action_id, owner, and reviewer are required")
		return
	}

	srv.store.mu.Lock()
	defer srv.store.mu.Unlock()

	tenantID := tenantCtx.TenantID()
	actKey := makeScopedKey(tenantID, actID)

	now := srv.store.clock().UTC()
	act := ActionData{
		ActionID:    actID,
		TenantID:    tenantID,
		InstanceID:  req.InstanceID,
		Title:       req.Title,
		Owner:       req.Owner,
		Reviewer:    req.Reviewer,
		State:       "IN_REVIEW",
		EvidenceIDs: req.EvidenceIDs,
		CreatedAt:   now,
	}

	srv.store.actions[actKey] = act
	srv.store.appendAudit(tenantID, "ACTION_CREATED", actID, tenantCtx.Subject(), "")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(act)
}

// 6. Close Action (Requires Reviewer)
func (srv *WalkingSkeletonServer) handleCloseAction(w http.ResponseWriter, r *http.Request) {
	tenantCtx, ok := srv.getTenantAndClaims(r)
	if !ok {
		writeErrorResponse(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing trusted tenant context")
		return
	}

	var req struct {
		ActionID string `json:"action_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON payload")
		return
	}

	srv.store.mu.Lock()
	defer srv.store.mu.Unlock()

	tenantID := tenantCtx.TenantID()
	actKey := makeScopedKey(tenantID, req.ActionID)
	act, exists := srv.store.actions[actKey]
	if !exists {
		writeErrorResponse(w, http.StatusNotFound, "NOT_FOUND", "action not found in tenant scope")
		return
	}

	// Verify caller is reviewer
	caller := tenantCtx.Subject()
	if act.Reviewer != "" && act.Reviewer != caller {
		writeErrorResponse(w, http.StatusForbidden, "FORBIDDEN", "only designated reviewer can close action")
		return
	}

	now := srv.store.clock().UTC()
	act.State = "CLOSED"
	act.ClosedAt = &now
	act.ClosedBy = caller

	srv.store.actions[actKey] = act
	srv.store.appendAudit(tenantID, "ACTION_CLOSED", req.ActionID, caller, "")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(act)
}

// 7. Get Report Summary
func (srv *WalkingSkeletonServer) handleGetReportSummary(w http.ResponseWriter, r *http.Request) {
	tenantCtx, ok := srv.getTenantAndClaims(r)
	if !ok {
		writeErrorResponse(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing trusted tenant context")
		return
	}

	srv.store.mu.RLock()
	defer srv.store.mu.RUnlock()

	tenantID := tenantCtx.TenantID()
	var inspectionsCount, openActions, closedActions int

	for _, inst := range srv.store.instances {
		if inst.TenantID == tenantID {
			inspectionsCount++
		}
	}

	for _, act := range srv.store.actions {
		if act.TenantID == tenantID {
			if act.State == "CLOSED" {
				closedActions++
			} else {
				openActions++
			}
		}
	}

	totalActions := openActions + closedActions
	complianceRate := 100.0
	if totalActions > 0 {
		complianceRate = (float64(closedActions) / float64(totalActions)) * 100.0
	}

	now := srv.store.clock().UTC()
	summary := ReportSummary{
		ReportID:           fmt.Sprintf("rep_%s_%d", tenantID, now.Unix()),
		TenantID:           tenantID,
		GeneratedAt:        now,
		InspectionsCount:   inspectionsCount,
		OpenActionsCount:   openActions,
		ClosedActionsCount: closedActions,
		ComplianceRate:     complianceRate,
		Freshness:          "FRESH",
		NonAuthorityNotice: DefaultNonAuthorityNotice,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(summary)
}

// 8. Get Export Records
func (srv *WalkingSkeletonServer) handleGetExportRecords(w http.ResponseWriter, r *http.Request) {
	tenantCtx, ok := srv.getTenantAndClaims(r)
	if !ok {
		writeErrorResponse(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing trusted tenant context")
		return
	}

	srv.store.mu.RLock()
	defer srv.store.mu.RUnlock()

	tenantID := tenantCtx.TenantID()
	var recordIDs []string

	for _, inst := range srv.store.instances {
		if inst.TenantID == tenantID {
			recordIDs = append(recordIDs, inst.InstanceID)
		}
	}
	sort.Strings(recordIDs)

	h := sha256.New()
	for _, id := range recordIDs {
		h.Write([]byte(id))
	}
	sourceDigest := hex.EncodeToString(h.Sum(nil))

	now := srv.store.clock().UTC()
	manifest := ExportManifest{
		ManifestID:         fmt.Sprintf("exp_%s_%d", tenantID, now.Unix()),
		TenantID:           tenantID,
		ExportedAt:         now,
		ExportedBy:         tenantCtx.Subject(),
		RecordCount:        len(recordIDs),
		SourceDigest:       sourceDigest,
		NonAuthorityNotice: DefaultNonAuthorityNotice,
	}

	srv.store.appendAudit(tenantID, "EXPORT_GENERATED", manifest.ManifestID, tenantCtx.Subject(), "")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(manifest)
}

// 9. Get Audit Trail (Reconstruction)
func (srv *WalkingSkeletonServer) handleGetAuditTrail(w http.ResponseWriter, r *http.Request) {
	tenantCtx, ok := srv.getTenantAndClaims(r)
	if !ok {
		writeErrorResponse(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing trusted tenant context")
		return
	}

	srv.store.mu.RLock()
	defer srv.store.mu.RUnlock()

	tenantID := tenantCtx.TenantID()
	var tenantTrail []AuditEventData

	for _, entry := range srv.store.auditEvents {
		if entry.TenantID == tenantID {
			tenantTrail = append(tenantTrail, entry)
		}
	}

	// Sort chronologically by monotonic sequence
	sort.Slice(tenantTrail, func(i, j int) bool {
		return tenantTrail[i].Sequence < tenantTrail[j].Sequence
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(tenantTrail)
}
