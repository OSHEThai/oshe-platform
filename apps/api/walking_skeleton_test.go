package api_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/oshethai/oshe-platform/apps/api"
	orgtenancy "github.com/oshethai/oshe-platform/modules/organization-tenancy"
)

type mockClaimsConfig struct {
	subject  string
	tenantID string
	roles    []string
	err      error
}

func newMockResolver(cfg mockClaimsConfig) api.ClaimsResolver {
	return func(r *http.Request) (*orgtenancy.TrustedClaims, error) {
		if cfg.err != nil {
			return nil, cfg.err
		}
		if cfg.tenantID == "" {
			return nil, errors.New("unauthenticated")
		}
		return &orgtenancy.TrustedClaims{
			Subject:         cfg.subject,
			TenantID:        cfg.tenantID,
			IsAuthenticated: true,
		}, nil
	}
}

func TestWalkingSkeleton_CompleteSyntheticLifecycle(t *testing.T) {
	store := api.NewWalkingSkeletonStore(nil)
	tenantID := "ten_safety_alpha"
	inspector := "inspector_somchai"
	reviewer := "lead_alice"

	// 1. Setup server for inspector
	resolverInspector := newMockResolver(mockClaimsConfig{
		subject:  inspector,
		tenantID: tenantID,
		roles:    []string{"INSPECTOR"},
	})
	server := api.NewWalkingSkeletonServer(store, resolverInspector)
	handler := server.Handler()

	// -------------------------------------------------------------
	// Step 1: Publish Template
	// -------------------------------------------------------------
	publishBody := `{
		"template_id": "tmpl_scaffold_v1",
		"version_id": "v1.0.0",
		"title": "Scaffolding Safety Inspection Checklist",
		"questions": [
			{"id": "q1", "text": "Are base plates secure?", "type": "BOOLEAN", "required": true},
			{"id": "q2", "text": "Are guardrails in place?", "type": "BOOLEAN", "required": true}
		]
	}`
	reqPub := httptest.NewRequest("POST", "/api/v1/templates/publish", bytes.NewBufferString(publishBody))
	wPub := httptest.NewRecorder()
	handler.ServeHTTP(wPub, reqPub)

	if wPub.Code != http.StatusCreated {
		t.Fatalf("publish template failed: status=%d, body=%s", wPub.Code, wPub.Body.String())
	}

	var pubResp api.PublishedTemplate
	_ = json.NewDecoder(wPub.Body).Decode(&pubResp)
	if pubResp.TemplateID != "tmpl_scaffold_v1" || pubResp.TenantID != tenantID {
		t.Errorf("unexpected published template: %+v", pubResp)
	}

	// -------------------------------------------------------------
	// Step 2: Instantiate Checklist
	// -------------------------------------------------------------
	instBody := `{"instance_id": "inst_inspect_001", "template_id": "tmpl_scaffold_v1"}`
	reqInst := httptest.NewRequest("POST", "/api/v1/checklists/instantiate", bytes.NewBufferString(instBody))
	wInst := httptest.NewRecorder()
	handler.ServeHTTP(wInst, reqInst)

	if wInst.Code != http.StatusCreated {
		t.Fatalf("instantiate checklist failed: status=%d, body=%s", wInst.Code, wInst.Body.String())
	}

	// -------------------------------------------------------------
	// Step 3: Complete Checklist (with finding)
	// -------------------------------------------------------------
	completeBody := `{
		"instance_id": "inst_inspect_001",
		"answers": {
			"q1": "true",
			"q2": "false"
		}
	}`
	reqComp := httptest.NewRequest("POST", "/api/v1/checklists/complete", bytes.NewBufferString(completeBody))
	wComp := httptest.NewRecorder()
	handler.ServeHTTP(wComp, reqComp)

	if wComp.Code != http.StatusOK {
		t.Fatalf("complete checklist failed: status=%d, body=%s", wComp.Code, wComp.Body.String())
	}

	// -------------------------------------------------------------
	// Step 4: Upload Evidence with SHA-256 Digest
	// -------------------------------------------------------------
	payload := "binary_photo_content_scaffold_defect_bytes"
	h := sha256.Sum256([]byte(payload))
	computedDigest := hex.EncodeToString(h[:])

	evUploadBody := map[string]any{
		"evidence_id":     "evd_scaffold_photo_01",
		"instance_id":     "inst_inspect_001",
		"filename":        "scaffold_crack.jpg",
		"media_type":      "image/jpeg",
		"payload":         payload,
		"asserted_digest": computedDigest,
	}
	evBytes, _ := json.Marshal(evUploadBody)
	reqEv := httptest.NewRequest("POST", "/api/v1/evidence/upload", bytes.NewBuffer(evBytes))
	wEv := httptest.NewRecorder()
	handler.ServeHTTP(wEv, reqEv)

	if wEv.Code != http.StatusCreated {
		t.Fatalf("upload evidence failed: status=%d, body=%s", wEv.Code, wEv.Body.String())
	}

	// -------------------------------------------------------------
	// Step 5: Create Corrective Action
	// -------------------------------------------------------------
	actionBody := map[string]any{
		"action_id":    "act_fix_scaffold_01",
		"instance_id":  "inst_inspect_001",
		"title":        "Install replacement mid-rail on scaffold",
		"owner":        inspector,
		"reviewer":     reviewer,
		"evidence_ids": []string{"evd_scaffold_photo_01"},
	}
	actBytes, _ := json.Marshal(actionBody)
	reqAct := httptest.NewRequest("POST", "/api/v1/actions", bytes.NewBuffer(actBytes))
	wAct := httptest.NewRecorder()
	handler.ServeHTTP(wAct, reqAct)

	if wAct.Code != http.StatusCreated {
		t.Fatalf("create action failed: status=%d, body=%s", wAct.Code, wAct.Body.String())
	}

	// -------------------------------------------------------------
	// Step 6: Close Action as Authorized Reviewer
	// -------------------------------------------------------------
	resolverReviewer := newMockResolver(mockClaimsConfig{
		subject:  reviewer,
		tenantID: tenantID,
		roles:    []string{"COMPLIANCE_LEAD"},
	})
	serverReviewer := api.NewWalkingSkeletonServer(store, resolverReviewer)
	handlerReviewer := serverReviewer.Handler()

	closeBody := `{"action_id": "act_fix_scaffold_01"}`
	reqClose := httptest.NewRequest("POST", "/api/v1/actions/close", bytes.NewBufferString(closeBody))
	wClose := httptest.NewRecorder()
	handlerReviewer.ServeHTTP(wClose, reqClose)

	if wClose.Code != http.StatusOK {
		t.Fatalf("close action failed: status=%d, body=%s", wClose.Code, wClose.Body.String())
	}

	// -------------------------------------------------------------
	// Step 7: Get Report Summary
	// -------------------------------------------------------------
	reqRep := httptest.NewRequest("GET", "/api/v1/reports/summary", nil)
	wRep := httptest.NewRecorder()
	handler.ServeHTTP(wRep, reqRep)

	if wRep.Code != http.StatusOK {
		t.Fatalf("get report summary failed: status=%d, body=%s", wRep.Code, wRep.Body.String())
	}

	var repSummary api.ReportSummary
	_ = json.NewDecoder(wRep.Body).Decode(&repSummary)
	if repSummary.InspectionsCount != 1 || repSummary.ClosedActionsCount != 1 || repSummary.OpenActionsCount != 0 {
		t.Errorf("unexpected report summary counts: %+v", repSummary)
	}
	if !strings.Contains(repSummary.NonAuthorityNotice, "DERIVED_OUTPUT_NON_AUTHORITY") {
		t.Errorf("missing non-authority disclaimer: %s", repSummary.NonAuthorityNotice)
	}

	// -------------------------------------------------------------
	// Step 8: Get Export Records Manifest
	// -------------------------------------------------------------
	reqExp := httptest.NewRequest("GET", "/api/v1/exports/records", nil)
	wExp := httptest.NewRecorder()
	handler.ServeHTTP(wExp, reqExp)

	if wExp.Code != http.StatusOK {
		t.Fatalf("get export records failed: status=%d, body=%s", wExp.Code, wExp.Body.String())
	}

	var expManifest api.ExportManifest
	_ = json.NewDecoder(wExp.Body).Decode(&expManifest)
	if expManifest.RecordCount != 1 || len(expManifest.SourceDigest) != 64 {
		t.Errorf("unexpected export manifest: %+v", expManifest)
	}

	// -------------------------------------------------------------
	// Step 9: Reconstruct Audit Trail
	// -------------------------------------------------------------
	reqAud := httptest.NewRequest("GET", "/api/v1/audit/trail", nil)
	wAud := httptest.NewRecorder()
	handler.ServeHTTP(wAud, reqAud)

	if wAud.Code != http.StatusOK {
		t.Fatalf("get audit trail failed: status=%d, body=%s", wAud.Code, wAud.Body.String())
	}

	var auditTrail []api.AuditEventData
	_ = json.NewDecoder(wAud.Body).Decode(&auditTrail)
	if len(auditTrail) < 6 {
		t.Fatalf("expected at least 6 audit entries, got %d", len(auditTrail))
	}

	// Verify monotonic sequence
	for i, entry := range auditTrail {
		expectedSeq := int64(i + 1)
		if entry.Sequence != expectedSeq {
			t.Errorf("audit entry %d: expected sequence %d, got %d", i, expectedSeq, entry.Sequence)
		}
		if entry.TenantID != tenantID {
			t.Errorf("audit entry %d: wrong tenant %s", i, entry.TenantID)
		}
	}
}

func TestWalkingSkeleton_DefaultDeny_Unauthenticated(t *testing.T) {
	store := api.NewWalkingSkeletonStore(nil)
	resolverUnauth := newMockResolver(mockClaimsConfig{
		err: errors.New("missing bearer authorization header"),
	})
	server := api.NewWalkingSkeletonServer(store, resolverUnauth)
	handler := server.Handler()

	req := httptest.NewRequest("GET", "/api/v1/reports/summary", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for unauthenticated request, got %d", w.Code)
	}
}

func TestWalkingSkeleton_DefaultDeny_HeaderInjection(t *testing.T) {
	store := api.NewWalkingSkeletonStore(nil)
	resolver := newMockResolver(mockClaimsConfig{
		subject:  "user_alice",
		tenantID: "ten_valid",
	})
	server := api.NewWalkingSkeletonServer(store, resolver)
	handler := server.Handler()

	req := httptest.NewRequest("GET", "/api/v1/reports/summary", nil)
	req.Header.Set("X-Tenant-ID", "ten_malicious_override")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for untrusted header injection, got %d", w.Code)
	}
}

func TestWalkingSkeleton_CrossTenantIsolation(t *testing.T) {
	store := api.NewWalkingSkeletonStore(nil)

	// Populate tenant Alpha resource
	resolverAlpha := newMockResolver(mockClaimsConfig{
		subject:  "user_alpha",
		tenantID: "ten_alpha",
	})
	serverAlpha := api.NewWalkingSkeletonServer(store, resolverAlpha)
	handlerAlpha := serverAlpha.Handler()

	// Alpha publishes template and instantiates checklist
	pubBody := `{"template_id": "tmpl_alpha", "version_id": "v1", "title": "Alpha Tmpl", "questions": [{"id": "q1", "text": "Q1", "type": "BOOLEAN"}]}`
	reqPub := httptest.NewRequest("POST", "/api/v1/templates/publish", bytes.NewBufferString(pubBody))
	wPub := httptest.NewRecorder()
	handlerAlpha.ServeHTTP(wPub, reqPub)
	if wPub.Code != http.StatusCreated {
		t.Fatalf("alpha publish failed: %d", wPub.Code)
	}

	instBody := `{"instance_id": "inst_alpha_01", "template_id": "tmpl_alpha"}`
	reqInst := httptest.NewRequest("POST", "/api/v1/checklists/instantiate", bytes.NewBufferString(instBody))
	wInst := httptest.NewRecorder()
	handlerAlpha.ServeHTTP(wInst, reqInst)
	if wInst.Code != http.StatusCreated {
		t.Fatalf("alpha instantiate failed: %d", wInst.Code)
	}

	// Tenant Bravo requests
	resolverBravo := newMockResolver(mockClaimsConfig{
		subject:  "user_bravo",
		tenantID: "ten_bravo",
	})
	serverBravo := api.NewWalkingSkeletonServer(store, resolverBravo)
	handlerBravo := serverBravo.Handler()

	// Bravo attempts to upload evidence targeting Alpha's checklist instance -> 404
	evBody := `{"evidence_id": "evd_bravo", "instance_id": "inst_alpha_01", "filename": "test.jpg", "payload": "abc"}`
	reqEv := httptest.NewRequest("POST", "/api/v1/evidence/upload", bytes.NewBufferString(evBody))
	wEv := httptest.NewRecorder()
	handlerBravo.ServeHTTP(wEv, reqEv)

	if wEv.Code != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found targeting cross-tenant instance, got %d", wEv.Code)
	}

	// Bravo checks audit trail -> 0 events
	reqAud := httptest.NewRequest("GET", "/api/v1/audit/trail", nil)
	wAud := httptest.NewRecorder()
	handlerBravo.ServeHTTP(wAud, reqAud)

	var bravoEvents []api.AuditEventData
	_ = json.NewDecoder(wAud.Body).Decode(&bravoEvents)
	if len(bravoEvents) != 0 {
		t.Fatalf("cross-tenant audit data leaked: expected 0, got %d", len(bravoEvents))
	}
}

func TestWalkingSkeleton_EvidenceDigestMismatch(t *testing.T) {
	store := api.NewWalkingSkeletonStore(nil)
	resolver := newMockResolver(mockClaimsConfig{
		subject:  "inspector",
		tenantID: "ten_alpha",
	})
	server := api.NewWalkingSkeletonServer(store, resolver)
	handler := server.Handler()

	// Publish and instantiate
	pubBody := `{"template_id": "tmpl_1", "version_id": "v1", "title": "T1", "questions": [{"id": "q1", "text": "Q1", "type": "BOOLEAN"}]}`
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("POST", "/api/v1/templates/publish", bytes.NewBufferString(pubBody)))

	instBody := `{"instance_id": "inst_1", "template_id": "tmpl_1"}`
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("POST", "/api/v1/checklists/instantiate", bytes.NewBufferString(instBody)))

	// Upload with mismatched digest
	evMismatch := `{
		"evidence_id": "evd_bad",
		"instance_id": "inst_1",
		"filename": "defect.jpg",
		"payload": "correct_bytes",
		"asserted_digest": "0000000000000000000000000000000000000000000000000000000000000000"
	}`
	reqEv := httptest.NewRequest("POST", "/api/v1/evidence/upload", bytes.NewBufferString(evMismatch))
	wEv := httptest.NewRecorder()
	handler.ServeHTTP(wEv, reqEv)

	if wEv.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request on digest mismatch, got %d", wEv.Code)
	}
}

func TestWalkingSkeleton_ActionClosureUnauthorized(t *testing.T) {
	store := api.NewWalkingSkeletonStore(nil)

	// Creator creates action with designated reviewer
	resolverCreator := newMockResolver(mockClaimsConfig{
		subject:  "creator_bob",
		tenantID: "ten_alpha",
	})
	handlerCreator := api.NewWalkingSkeletonServer(store, resolverCreator).Handler()

	actBody := `{
		"action_id": "act_security",
		"title": "Fix lock",
		"owner": "creator_bob",
		"reviewer": "designated_reviewer_charlie"
	}`
	reqAct := httptest.NewRequest("POST", "/api/v1/actions", bytes.NewBufferString(actBody))
	handlerCreator.ServeHTTP(httptest.NewRecorder(), reqAct)

	// Creator attempts to self-close action -> 403 Forbidden
	closeBody := `{"action_id": "act_security"}`
	reqClose := httptest.NewRequest("POST", "/api/v1/actions/close", bytes.NewBufferString(closeBody))
	wClose := httptest.NewRecorder()
	handlerCreator.ServeHTTP(wClose, reqClose)

	if wClose.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for unauthorized self-closure, got %d", wClose.Code)
	}
}
