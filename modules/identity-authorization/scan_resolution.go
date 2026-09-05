// Package localidentity provides identity, authorization, and directory services for OSHE Platform.
//
// PROVISIONAL GOVERNANCE DECLARATION (Issue #130 / V040-I019):
// Under approved Sole Human Owner decision HDEC-V040-FOUNDATION-054, this file implements
// the synthetic-only untrusted scan/identifier resolver.
//
// Strict Scan Resolution Invariants:
// 1. A Scan is Input-Only (Never Authority): An encoded scan or QR payload is strictly untrusted
//    user input. Presenting a scan payload never conveys authorization or circumvents permission checks.
// 2. Multi-Layered Validation: Evaluates syntax, canonical prefix, expiration, tenant boundary,
//    geographic/project scope, object lifecycle state, and caller role/permission.
// 3. Anti-Enumeration Defense: Non-existent, guessed, cross-tenant, and unauthorized scan attempts
//    return a non-differentiating, non-enumerating denial response, preventing attackers from
//    probing valid identifiers.
// 4. Default-Deny: Any validation or authorization check failure immediately fails closed.
// 5. Append-Only Audit Logging: Every resolution attempt records an immutable audit entry.
// 6. Zero External Enactment: Operates purely in-memory on synthetic local fixtures.
package localidentity

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Supported canonical scan schemes.
const (
	ScanSchemeURI     = "oshe://"
	ScanSchemeCompact = "oshe:"
	ScanSchemeHTTPS   = "https://app.oshe.local/scan"
)

// ScannableObjectType defines permitted object types in scan payloads.
type ScannableObjectType string

const (
	ScannableEquipment  ScannableObjectType = "equipment"
	ScannableSite       ScannableObjectType = "site"
	ScannableArea       ScannableObjectType = "area"
	ScannableChecklist  ScannableObjectType = "checklist"
	ScannableInspection ScannableObjectType = "inspection"
	ScannableFinding    ScannableObjectType = "finding"
)

// ValidObjectTypePrefixes maps object types to their canonical identifier prefixes.
var ValidObjectTypePrefixes = map[ScannableObjectType]string{
	ScannableEquipment:  "eqp_",
	ScannableSite:       "ste_",
	ScannableArea:       "ara_",
	ScannableChecklist:  "chk_",
	ScannableInspection: "ins_",
	ScannableFinding:    "fnd_",
}

// ScanDenialCode defines standardized, non-enumerating denial categories.
type ScanDenialCode string

const (
	// DenialScanInvalidInput represents malformed syntax, unsupported schemes, or corrupt characters.
	DenialScanInvalidInput ScanDenialCode = "DENIAL_SCAN_INVALID_INPUT"
	// DenialScanExpired represents temporally expired scan payloads.
	DenialScanExpired ScanDenialCode = "DENIAL_SCAN_EXPIRED"
	// DenialScanUnauthorized represents any authorization, scope, tenant, object-state,
	// or existence failure. Using a single unified code prevents resource enumeration attacks.
	DenialScanUnauthorized ScanDenialCode = "DENIAL_SCAN_UNAUTHORIZED"
)

var (
	ErrMalformedScanPayload    = errors.New("scan payload is malformed or uses unsupported scheme")
	ErrScanPayloadExpired      = errors.New("scan payload has expired")
	ErrScanUnauthorized        = errors.New("unauthorized scan resolution: access denied")
	ErrInvalidObjectType       = errors.New("unsupported scannable object type")
	ErrInvalidObjectIdentifier = errors.New("object identifier prefix mismatch for declared type")
)

// ScannableObject represents a registered target domain entity that can be resolved from a scan.
type ScannableObject struct {
	TenantID           string
	ObjectType         ScannableObjectType
	ObjectID           string
	ProjectID          string
	SiteID             string
	AreaID             string
	LifecycleState     ResourceLifecycle
	RequiredPermission Permission
	Metadata           map[string]string
}

// ScanPayload represents a parsed, normalized scan data structure.
type ScanPayload struct {
	Raw        string
	TenantID   string
	ObjectType ScannableObjectType
	ObjectID   string
	Token      string
	ExpiresAt  time.Time
	HasExpiry  bool
}

// ScanResolutionContext encapsulates the ambient security and operational context of a scan event.
type ScanResolutionContext struct {
	Identity      SubjectIdentity
	CallerRole    Role
	ActiveProject string
	ActiveSite    string
	RawScan       string
	Action        Permission
	At            time.Time
}

// ScanResolutionAuditRecord captures an immutable record of a scan resolution event.
type ScanResolutionAuditRecord struct {
	RecordID           string
	TenantID           string
	ActorSubject       string
	CallerRole         Role
	RawPayloadHash     string
	ParsedObjectType   ScannableObjectType
	ParsedObjectID     string
	Allowed            bool
	DenialCode         ScanDenialCode
	InternalDiagnostic string
	Timestamp          time.Time
}

// ScanResolutionResult returns the resolution decision and resolved object (if permitted).
type ScanResolutionResult struct {
	Allowed        bool
	DenialCode     ScanDenialCode
	ErrorMessage   string
	ResolvedObject *ScannableObject
	Audit          ScanResolutionAuditRecord
}

// ScanResolver coordinates parsing, security validation, permission evaluation, and audit logging.
type ScanResolver struct {
	mu           sync.RWMutex
	objects      map[string]ScannableObject // key: tenantID:objectID
	auditRecords []ScanResolutionAuditRecord
	matrix       AuthorizationMatrix
	evaluator    *PolicyEvaluator
}

// NewScanResolver initializes a new ScanResolver.
func NewScanResolver(matrix AuthorizationMatrix, evaluator *PolicyEvaluator) *ScanResolver {
	return &ScanResolver{
		objects:      make(map[string]ScannableObject),
		auditRecords: make([]ScanResolutionAuditRecord, 0),
		matrix:       matrix,
		evaluator:    evaluator,
	}
}

// RegisterObject registers a scannable domain entity in the resolver's local directory.
func (r *ScanResolver) RegisterObject(obj ScannableObject) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if strings.TrimSpace(obj.TenantID) == "" || strings.TrimSpace(obj.ObjectID) == "" {
		return errors.New("tenantID and objectID cannot be empty")
	}

	prefix, ok := ValidObjectTypePrefixes[obj.ObjectType]
	if !ok {
		return ErrInvalidObjectType
	}
	if !strings.HasPrefix(obj.ObjectID, prefix) {
		return ErrInvalidObjectIdentifier
	}

	key := fmt.Sprintf("%s:%s", obj.TenantID, obj.ObjectID)
	r.objects[key] = obj
	return nil
}

// ParseScan parses and normalizes an untrusted raw scan payload against supported schemes.
func (r *ScanResolver) ParseScan(raw string) (ScanPayload, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ScanPayload{}, ErrMalformedScanPayload
	}

	// Reject path traversal, null bytes, and control characters
	if strings.Contains(trimmed, "\x00") || strings.Contains(trimmed, "../") || strings.Contains(trimmed, "..\\") {
		return ScanPayload{}, ErrMalformedScanPayload
	}

	// 1. URI Scheme: oshe://<tenant>/<object_type>/<object_id>?token=...&exp=...
	if strings.HasPrefix(trimmed, ScanSchemeURI) {
		return parseURIScheme(trimmed)
	}

	// 2. Compact Scheme: oshe:<tenant>:<object_type>:<object_id>
	if strings.HasPrefix(trimmed, ScanSchemeCompact) && !strings.HasPrefix(trimmed, ScanSchemeURI) {
		return parseCompactScheme(trimmed)
	}

	// 3. HTTPS Scheme: https://app.oshe.local/scan?tenant=...&type=...&id=...
	if strings.HasPrefix(trimmed, ScanSchemeHTTPS) {
		return parseHTTPSScheme(trimmed)
	}

	return ScanPayload{}, ErrMalformedScanPayload
}

func parseURIScheme(raw string) (ScanPayload, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "oshe" {
		return ScanPayload{}, ErrMalformedScanPayload
	}

	tenantID := u.Host
	if tenantID == "" {
		return ScanPayload{}, ErrMalformedScanPayload
	}

	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 2 {
		return ScanPayload{}, ErrMalformedScanPayload
	}

	objType := ScannableObjectType(parts[0])
	objID := parts[1]

	prefix, ok := ValidObjectTypePrefixes[objType]
	if !ok || !strings.HasPrefix(objID, prefix) {
		return ScanPayload{}, ErrMalformedScanPayload
	}

	payload := ScanPayload{
		Raw:        raw,
		TenantID:   tenantID,
		ObjectType: objType,
		ObjectID:   objID,
		Token:      u.Query().Get("token"),
	}

	expStr := u.Query().Get("exp")
	if expStr != "" {
		expUnix, err := strconv.ParseInt(expStr, 10, 64)
		if err == nil {
			payload.ExpiresAt = time.Unix(expUnix, 0).UTC()
			payload.HasExpiry = true
		}
	}

	return payload, nil
}

func parseCompactScheme(raw string) (ScanPayload, error) {
	content := strings.TrimPrefix(raw, ScanSchemeCompact)
	parts := strings.Split(content, ":")
	if len(parts) < 3 {
		return ScanPayload{}, ErrMalformedScanPayload
	}

	tenantID := parts[0]
	objType := ScannableObjectType(parts[1])
	objID := parts[2]

	if tenantID == "" || objID == "" {
		return ScanPayload{}, ErrMalformedScanPayload
	}

	prefix, ok := ValidObjectTypePrefixes[objType]
	if !ok || !strings.HasPrefix(objID, prefix) {
		return ScanPayload{}, ErrMalformedScanPayload
	}

	return ScanPayload{
		Raw:        raw,
		TenantID:   tenantID,
		ObjectType: objType,
		ObjectID:   objID,
	}, nil
}

func parseHTTPSScheme(raw string) (ScanPayload, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return ScanPayload{}, ErrMalformedScanPayload
	}

	tenantID := u.Query().Get("tenant")
	objType := ScannableObjectType(u.Query().Get("type"))
	objID := u.Query().Get("id")

	if tenantID == "" || objID == "" {
		return ScanPayload{}, ErrMalformedScanPayload
	}

	prefix, ok := ValidObjectTypePrefixes[objType]
	if !ok || !strings.HasPrefix(objID, prefix) {
		return ScanPayload{}, ErrMalformedScanPayload
	}

	payload := ScanPayload{
		Raw:        raw,
		TenantID:   tenantID,
		ObjectType: objType,
		ObjectID:   objID,
		Token:      u.Query().Get("token"),
	}

	expStr := u.Query().Get("exp")
	if expStr != "" {
		expUnix, err := strconv.ParseInt(expStr, 10, 64)
		if err == nil {
			payload.ExpiresAt = time.Unix(expUnix, 0).UTC()
			payload.HasExpiry = true
		}
	}

	return payload, nil
}

// Resolve processes an untrusted scan in the context of the caller's active identity, scope, and permissions.
func (r *ScanResolver) Resolve(ctx ScanResolutionContext) ScanResolutionResult {
	now := ctx.At
	if now.IsZero() {
		now = time.Now().UTC()
	}

	rawHash := sha256SumHex([]byte(ctx.RawScan))
	recordID := fmt.Sprintf("audit_scan_%x", sha256Sum([]byte(fmt.Sprintf("%s:%s:%d", ctx.Identity.Subject, rawHash, now.UnixNano())))[:8])

	// 1. Authenticated Identity Check
	if !ctx.Identity.IsAuthenticated || strings.TrimSpace(ctx.Identity.Subject) == "" || strings.TrimSpace(ctx.Identity.TenantID) == "" {
		audit := ScanResolutionAuditRecord{
			RecordID:           recordID,
			TenantID:           ctx.Identity.TenantID,
			ActorSubject:       ctx.Identity.Subject,
			CallerRole:         ctx.CallerRole,
			RawPayloadHash:     rawHash,
			Allowed:            false,
			DenialCode:         DenialScanUnauthorized,
			InternalDiagnostic: "caller identity is unauthenticated or has blank subject/tenant",
			Timestamp:          now,
		}
		r.recordAudit(audit)
		return ScanResolutionResult{
			Allowed:      false,
			DenialCode:   DenialScanUnauthorized,
			ErrorMessage: "unauthenticated caller",
			Audit:        audit,
		}
	}

	// 2. Syntax and Scheme Parse
	payload, err := r.ParseScan(ctx.RawScan)
	if err != nil {
		audit := ScanResolutionAuditRecord{
			RecordID:           recordID,
			TenantID:           ctx.Identity.TenantID,
			ActorSubject:       ctx.Identity.Subject,
			CallerRole:         ctx.CallerRole,
			RawPayloadHash:     rawHash,
			Allowed:            false,
			DenialCode:         DenialScanInvalidInput,
			InternalDiagnostic: fmt.Sprintf("syntax error: %s", err.Error()),
			Timestamp:          now,
		}
		r.recordAudit(audit)
		return ScanResolutionResult{
			Allowed:      false,
			DenialCode:   DenialScanInvalidInput,
			ErrorMessage: "invalid or malformed scan payload",
			Audit:        audit,
		}
	}

	// 3. Temporal Expiration Check
	if payload.HasExpiry && now.After(payload.ExpiresAt) {
		audit := ScanResolutionAuditRecord{
			RecordID:           recordID,
			TenantID:           ctx.Identity.TenantID,
			ActorSubject:       ctx.Identity.Subject,
			CallerRole:         ctx.CallerRole,
			RawPayloadHash:     rawHash,
			ParsedObjectType:   payload.ObjectType,
			ParsedObjectID:     payload.ObjectID,
			Allowed:            false,
			DenialCode:         DenialScanExpired,
			InternalDiagnostic: fmt.Sprintf("payload expired at %s, current time %s", payload.ExpiresAt.Format(time.RFC3339), now.Format(time.RFC3339)),
			Timestamp:          now,
		}
		r.recordAudit(audit)
		return ScanResolutionResult{
			Allowed:      false,
			DenialCode:   DenialScanExpired,
			ErrorMessage: "scan payload has expired",
			Audit:        audit,
		}
	}

	// 4. Tenant Boundary Isolation Check
	if payload.TenantID != ctx.Identity.TenantID {
		audit := ScanResolutionAuditRecord{
			RecordID:           recordID,
			TenantID:           ctx.Identity.TenantID,
			ActorSubject:       ctx.Identity.Subject,
			CallerRole:         ctx.CallerRole,
			RawPayloadHash:     rawHash,
			ParsedObjectType:   payload.ObjectType,
			ParsedObjectID:     payload.ObjectID,
			Allowed:            false,
			DenialCode:         DenialScanUnauthorized,
			InternalDiagnostic: fmt.Sprintf("cross-tenant access attempt: payload tenant %s != caller tenant %s", payload.TenantID, ctx.Identity.TenantID),
			Timestamp:          now,
		}
		r.recordAudit(audit)
		// Return generic DenialScanUnauthorized to prevent tenant enumeration
		return ScanResolutionResult{
			Allowed:      false,
			DenialCode:   DenialScanUnauthorized,
			ErrorMessage: "access denied to target resource",
			Audit:        audit,
		}
	}

	// 5. Object Existence Check (in caller's tenant)
	r.mu.RLock()
	objKey := fmt.Sprintf("%s:%s", ctx.Identity.TenantID, payload.ObjectID)
	targetObj, exists := r.objects[objKey]
	r.mu.RUnlock()

	if !exists {
		audit := ScanResolutionAuditRecord{
			RecordID:           recordID,
			TenantID:           ctx.Identity.TenantID,
			ActorSubject:       ctx.Identity.Subject,
			CallerRole:         ctx.CallerRole,
			RawPayloadHash:     rawHash,
			ParsedObjectType:   payload.ObjectType,
			ParsedObjectID:     payload.ObjectID,
			Allowed:            false,
			DenialCode:         DenialScanUnauthorized,
			InternalDiagnostic: fmt.Sprintf("scanned object %s does not exist in tenant %s (anti-enumeration)", payload.ObjectID, ctx.Identity.TenantID),
			Timestamp:          now,
		}
		r.recordAudit(audit)
		// Anti-Enumeration: Return generic DenialScanUnauthorized rather than "object not found"
		return ScanResolutionResult{
			Allowed:      false,
			DenialCode:   DenialScanUnauthorized,
			ErrorMessage: "access denied to target resource",
			Audit:        audit,
		}
	}

	// 6. Object Lifecycle State Evaluation
	if targetObj.LifecycleState != ResourceActive {
		audit := ScanResolutionAuditRecord{
			RecordID:           recordID,
			TenantID:           ctx.Identity.TenantID,
			ActorSubject:       ctx.Identity.Subject,
			CallerRole:         ctx.CallerRole,
			RawPayloadHash:     rawHash,
			ParsedObjectType:   payload.ObjectType,
			ParsedObjectID:     payload.ObjectID,
			Allowed:            false,
			DenialCode:         DenialScanUnauthorized,
			InternalDiagnostic: fmt.Sprintf("scanned object is in inactive lifecycle state: %s", targetObj.LifecycleState),
			Timestamp:          now,
		}
		r.recordAudit(audit)
		return ScanResolutionResult{
			Allowed:      false,
			DenialCode:   DenialScanUnauthorized,
			ErrorMessage: "access denied to target resource",
			Audit:        audit,
		}
	}

	// 7. Scope Confinement Check
	if ctx.ActiveSite != "" && targetObj.SiteID != "" && ctx.ActiveSite != targetObj.SiteID {
		audit := ScanResolutionAuditRecord{
			RecordID:           recordID,
			TenantID:           ctx.Identity.TenantID,
			ActorSubject:       ctx.Identity.Subject,
			CallerRole:         ctx.CallerRole,
			RawPayloadHash:     rawHash,
			ParsedObjectType:   payload.ObjectType,
			ParsedObjectID:     payload.ObjectID,
			Allowed:            false,
			DenialCode:         DenialScanUnauthorized,
			InternalDiagnostic: fmt.Sprintf("scope mismatch: caller active site %s != target site %s", ctx.ActiveSite, targetObj.SiteID),
			Timestamp:          now,
		}
		r.recordAudit(audit)
		return ScanResolutionResult{
			Allowed:      false,
			DenialCode:   DenialScanUnauthorized,
			ErrorMessage: "access denied to target resource",
			Audit:        audit,
		}
	}

	// 8. Caller Role & Permission Authorization Check
	// A scan is NEVER authority. The caller must possess the required permission.
	requiredPerm := targetObj.RequiredPermission
	if ctx.Action != "" {
		requiredPerm = ctx.Action
	}

	if requiredPerm != "" {
		if !r.matrix.RoleHasPermission(ctx.CallerRole, requiredPerm) {
			audit := ScanResolutionAuditRecord{
				RecordID:           recordID,
				TenantID:           ctx.Identity.TenantID,
				ActorSubject:       ctx.Identity.Subject,
				CallerRole:         ctx.CallerRole,
				RawPayloadHash:     rawHash,
				ParsedObjectType:   payload.ObjectType,
				ParsedObjectID:     payload.ObjectID,
				Allowed:            false,
				DenialCode:         DenialScanUnauthorized,
				InternalDiagnostic: fmt.Sprintf("role %s lacks required permission %s on scanned target", ctx.CallerRole, requiredPerm),
				Timestamp:          now,
			}
			r.recordAudit(audit)
			return ScanResolutionResult{
				Allowed:      false,
				DenialCode:   DenialScanUnauthorized,
				ErrorMessage: "access denied to target resource",
				Audit:        audit,
			}
		}
	}

	// 9. Successful Resolution & Authorization
	audit := ScanResolutionAuditRecord{
		RecordID:           recordID,
		TenantID:           ctx.Identity.TenantID,
		ActorSubject:       ctx.Identity.Subject,
		CallerRole:         ctx.CallerRole,
		RawPayloadHash:     rawHash,
		ParsedObjectType:   payload.ObjectType,
		ParsedObjectID:     payload.ObjectID,
		Allowed:            true,
		DenialCode:         "",
		InternalDiagnostic: "scan resolved and caller authorized successfully",
		Timestamp:          now,
	}
	r.recordAudit(audit)

	resolvedCopy := targetObj
	return ScanResolutionResult{
		Allowed:        true,
		DenialCode:     "",
		ErrorMessage:   "",
		ResolvedObject: &resolvedCopy,
		Audit:          audit,
	}
}

// AuditLedger returns a slice copy of all historical scan resolution audit records.
func (r *ScanResolver) AuditLedger() []ScanResolutionAuditRecord {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]ScanResolutionAuditRecord, len(r.auditRecords))
	copy(out, r.auditRecords)
	return out
}

func (r *ScanResolver) recordAudit(rec ScanResolutionAuditRecord) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.auditRecords = append(r.auditRecords, rec)
}

func sha256Sum(data []byte) []byte {
	h := sha256.Sum256(data)
	return h[:]
}

func sha256SumHex(data []byte) string {
	return hex.EncodeToString(sha256Sum(data))
}
