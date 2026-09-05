// Package localidentity provides local identity, authorization, and directory services for OSHE Platform.
//
// PROVISIONAL GOVERNANCE DECLARATION (H030-004, Issue #95):
// Under approved Sole Human Owner decision H030-004, this file implements local
// in-memory external-access condition models, sponsor-change protocols, explicit renewal
// controls, expiry/deactivation lifecycles, and stale-session invalidation mechanics.
//
// Strict Access Condition Invariants:
// 1. Online / Local Access Only: Offline access and trusted-device verification are strictly barred.
//    Any attempt to assert or require a trusted device fails closed (ErrTrustedDeviceProhibited).
// 2. Short Synthetic Validity: Initial external access windows are bounded to a maximum of 14 days.
// 3. Explicit Renewal Controls: Access extension requires active sponsor approval and is capped at 7 days.
// 4. Stale-Session Invalidation: Sponsor changes, renewals, and deactivations increment the condition's
//    generation counter, immediately rendering existing session tokens stale and denied (CategorySessionStale).
// 5. Append-Only Audit Ledger: Every access transition is recorded with full attribution and zero hard deletion.
// 6. Zero External Enactment: Operates purely in-memory on local synthetic fixtures.
package localidentity

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// MaxExternalAccessDuration defines the ceiling on initial external access validity (14 days).
const MaxExternalAccessDuration = 14 * 24 * time.Hour

// MaxRenewalExtension defines the maximum duration added per single renewal event (7 days).
const MaxRenewalExtension = 7 * 24 * time.Hour

// AccessConditionStatus classifies the lifecycle state of an access condition.
type AccessConditionStatus string

const (
	AccessConditionActive    AccessConditionStatus = "ACTIVE"
	AccessConditionSuspended AccessConditionStatus = "SUSPENDED"
	AccessConditionExpired   AccessConditionStatus = "EXPIRED"
	AccessConditionRevoked   AccessConditionStatus = "REVOKED"
)

var (
	// ErrBlankConditionID indicates missing condition identifier.
	ErrBlankConditionID = errors.New("condition ID must not be blank")
	// ErrBlankProjectID indicates missing project identifier.
	ErrBlankProjectID = errors.New("project ID must not be blank")
	// ErrTrustedDeviceProhibited indicates an illegal attempt to claim or require trusted-device authentication.
	ErrTrustedDeviceProhibited = errors.New("trusted-device or offline access is prohibited under H030-004: online/local access only")
	// ErrAccessConditionInactive indicates the condition is not currently in ACTIVE status.
	ErrAccessConditionInactive = errors.New("access condition is not active")
	// ErrAccessConditionExpired indicates the access validity window has elapsed.
	ErrAccessConditionExpired = errors.New("access condition has expired")
	// ErrAccessConditionRevoked indicates the access condition has been revoked.
	ErrAccessConditionRevoked = errors.New("access condition is revoked")
	// ErrAccessConditionSuspended indicates the access condition is temporarily suspended.
	ErrAccessConditionSuspended = errors.New("access condition is suspended")
	// ErrRenewalDurationExceeded indicates the requested renewal extension exceeds 7 days.
	ErrRenewalDurationExceeded = errors.New("renewal extension exceeds maximum allowed 7 days")
	// ErrStaleSession indicates caller session generation is older than current condition generation.
	ErrStaleSession = errors.New("session is stale: access condition generation has been incremented")
	// ErrAccessConditionNotFound indicates the requested access condition does not exist.
	ErrAccessConditionNotFound = errors.New("access condition record not found")
	// ErrDuplicateConditionID indicates a condition with the same ID already exists in the tenant.
	ErrDuplicateConditionID = errors.New("condition ID already registered for tenant")
	// ErrDurationExceeded indicates initial validity exceeds 14 days.
	ErrDurationExceeded = errors.New("initial access duration exceeds maximum allowed 14 days")
	// ErrSponsorUnchanged indicates an attempt to reassign to the identical current sponsor.
	ErrSponsorUnchanged = errors.New("new sponsor is identical to existing sponsor")
)

// AccessConditionRecord models a local, time-bounded, project/site access condition for an external user.
type AccessConditionRecord struct {
	conditionID     string
	tenantID        string
	subject         string
	projectID       string
	siteID          string
	sponsorID       string
	validFrom       time.Time
	validTo         time.Time
	status          AccessConditionStatus
	generation      int64
	renewalCount    int
	lastRenewedAt   time.Time
	revocationReason string
	revokedAt       time.Time
	createdAt       time.Time
	updatedAt       time.Time
}

// ConditionID returns the unique condition identifier.
func (c AccessConditionRecord) ConditionID() string { return c.conditionID }

// TenantID returns the authoritative tenant identifier.
func (c AccessConditionRecord) TenantID() string { return c.tenantID }

// Subject returns the external user subject identifier (usr_*).
func (c AccessConditionRecord) Subject() string { return c.subject }

// ProjectID returns the bounded project identifier (prj_*).
func (c AccessConditionRecord) ProjectID() string { return c.projectID }

// SiteID returns the bounded site identifier (ste_*) or empty if not site-restricted.
func (c AccessConditionRecord) SiteID() string { return c.siteID }

// SponsorID returns the mandatory internal sponsor manager identifier (usr_*).
func (c AccessConditionRecord) SponsorID() string { return c.sponsorID }

// ValidFrom returns the start of the validity window.
func (c AccessConditionRecord) ValidFrom() time.Time { return c.validFrom }

// ValidTo returns the expiration timestamp of the validity window.
func (c AccessConditionRecord) ValidTo() time.Time { return c.validTo }

// Status returns the lifecycle state of the condition.
func (c AccessConditionRecord) Status() AccessConditionStatus { return c.status }

// Generation returns the condition's decision generation counter.
func (c AccessConditionRecord) Generation() int64 { return c.generation }

// RenewalCount returns the number of times access has been renewed.
func (c AccessConditionRecord) RenewalCount() int { return c.renewalCount }

// LastRenewedAt returns the timestamp of the last renewal.
func (c AccessConditionRecord) LastRenewedAt() time.Time { return c.lastRenewedAt }

// RevocationReason returns recorded reason for revocation.
func (c AccessConditionRecord) RevocationReason() string { return c.revocationReason }

// RevokedAt returns timestamp of revocation.
func (c AccessConditionRecord) RevokedAt() time.Time { return c.revokedAt }

// CreatedAt returns creation timestamp.
func (c AccessConditionRecord) CreatedAt() time.Time { return c.createdAt }

// UpdatedAt returns update timestamp.
func (c AccessConditionRecord) UpdatedAt() time.Time { return c.updatedAt }

// IsActive returns true if the status is ACTIVE.
func (c AccessConditionRecord) IsActive() bool { return c.status == AccessConditionActive }

// IsValidAt checks if the condition is active and time t falls within [validFrom, validTo].
func (c AccessConditionRecord) IsValidAt(t time.Time) bool {
	if c.status != AccessConditionActive {
		return false
	}
	return !t.Before(c.validFrom) && !t.After(c.validTo)
}

// EffectiveStatus computes the operational status taking temporal expiry into account.
func (c AccessConditionRecord) EffectiveStatus(t time.Time) AccessConditionStatus {
	if c.status == AccessConditionRevoked {
		return AccessConditionRevoked
	}
	if c.status == AccessConditionSuspended {
		return AccessConditionSuspended
	}
	if t.After(c.validTo) || t.Before(c.validFrom) {
		return AccessConditionExpired
	}
	return AccessConditionActive
}

// ScopeMatches evaluates whether a target project and site are permitted under this condition.
func (c AccessConditionRecord) ScopeMatches(targetProject, targetSite string) bool {
	tProj := strings.TrimSpace(targetProject)
	if tProj == "" || tProj != c.projectID {
		return false
	}
	if c.siteID != "" {
		tSite := strings.TrimSpace(targetSite)
		if tSite == "" || tSite != c.siteID {
			return false
		}
	}
	return true
}

// NewAccessConditionRecord constructs and validates a new AccessConditionRecord.
// Enforces:
// 1. Mandatory online-only access: trustedDeviceRequired or allowOffline must be FALSE.
// 2. Short synthetic validity: initial duration cannot exceed 14 days.
// 3. Valid internal sponsor (usr_*).
func NewAccessConditionRecord(
	conditionID, tenantID, subject, projectID, siteID, sponsorID string,
	trustedDeviceRequired, allowOffline bool,
	validFrom, validTo time.Time,
) (AccessConditionRecord, error) {
	if trustedDeviceRequired || allowOffline {
		return AccessConditionRecord{}, ErrTrustedDeviceProhibited
	}

	trimmedID := strings.TrimSpace(conditionID)
	if trimmedID == "" {
		return AccessConditionRecord{}, ErrBlankConditionID
	}
	trimmedTenant := strings.TrimSpace(tenantID)
	if trimmedTenant == "" {
		return AccessConditionRecord{}, ErrBlankTenantID
	}
	if err := ValidateSubject(subject); err != nil {
		return AccessConditionRecord{}, err
	}
	trimmedProject := strings.TrimSpace(projectID)
	if trimmedProject == "" {
		return AccessConditionRecord{}, ErrBlankProjectID
	}
	if err := ValidateInternalSponsor(sponsorID); err != nil {
		return AccessConditionRecord{}, err
	}

	if validTo.Before(validFrom) || validTo.Equal(validFrom) {
		return AccessConditionRecord{}, ErrInvalidTimeWindow
	}
	duration := validTo.Sub(validFrom)
	if duration > MaxExternalAccessDuration {
		return AccessConditionRecord{}, fmt.Errorf("%w: duration %s exceeds %s", ErrDurationExceeded, duration, MaxExternalAccessDuration)
	}

	now := time.Now().UTC()
	return AccessConditionRecord{
		conditionID:   trimmedID,
		tenantID:      trimmedTenant,
		subject:       strings.TrimSpace(subject),
		projectID:     trimmedProject,
		siteID:        strings.TrimSpace(siteID),
		sponsorID:     strings.TrimSpace(sponsorID),
		validFrom:     validFrom.UTC(),
		validTo:       validTo.UTC(),
		status:        AccessConditionActive,
		generation:    1,
		renewalCount:  0,
		createdAt:     now,
		updatedAt:     now,
	}, nil
}

// AccessConditionAuditRecord models an immutable, append-only historical audit entry.
type AccessConditionAuditRecord struct {
	RecordID        string                `json:"record_id"`
	TenantID        string                `json:"tenant_id"`
	ConditionID     string                `json:"condition_id"`
	Subject         string                `json:"subject"`
	Transition      string                `json:"transition"`
	PriorSponsorID  string                `json:"prior_sponsor_id,omitempty"`
	NewSponsorID    string                `json:"new_sponsor_id,omitempty"`
	PriorGeneration int64                 `json:"prior_generation"`
	NewGeneration   int64                 `json:"new_generation"`
	NewValidTo      time.Time             `json:"new_valid_to,omitempty"`
	NewStatus       AccessConditionStatus `json:"new_status"`
	ActorSubject    string                `json:"actor_subject"`
	Reason          string                `json:"reason"`
	RecordedAt      time.Time             `json:"recorded_at"`
}

// AccessConditionLedger provides a thread-safe, in-memory append-only audit trail for access conditions.
type AccessConditionLedger struct {
	mu      sync.RWMutex
	records []AccessConditionAuditRecord
}

// NewAccessConditionLedger initializes an empty in-memory ledger.
func NewAccessConditionLedger() *AccessConditionLedger {
	return &AccessConditionLedger{
		records: make([]AccessConditionAuditRecord, 0),
	}
}

// AppendRecord appends an audit entry to the ledger.
func (l *AccessConditionLedger) AppendRecord(record AccessConditionAuditRecord) error {
	if record.TenantID == "" || record.ConditionID == "" {
		return ErrBlankConditionID
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.records = append(l.records, record)
	return nil
}

// GetConditionHistory returns the complete audit history for a condition under a tenant.
func (l *AccessConditionLedger) GetConditionHistory(tenantID, conditionID string) ([]AccessConditionAuditRecord, error) {
	tTenant := strings.TrimSpace(tenantID)
	if tTenant == "" {
		return nil, ErrBlankTenantID
	}
	tCond := strings.TrimSpace(conditionID)
	if tCond == "" {
		return nil, ErrBlankConditionID
	}

	l.mu.RLock()
	defer l.mu.RUnlock()

	var results []AccessConditionAuditRecord
	for _, rec := range l.records {
		if rec.TenantID == tTenant && rec.ConditionID == tCond {
			results = append(results, rec)
		}
	}
	return results, nil
}

// AccessConditionRegistry manages active and historical access conditions in memory.
type AccessConditionRegistry struct {
	mu         sync.RWMutex
	conditions map[string]AccessConditionRecord // key: tenantID + ":" + conditionID
	ledger     *AccessConditionLedger
}

// NewAccessConditionRegistry initializes an in-memory access condition registry.
func NewAccessConditionRegistry(ledger *AccessConditionLedger) *AccessConditionRegistry {
	if ledger == nil {
		ledger = NewAccessConditionLedger()
	}
	return &AccessConditionRegistry{
		conditions: make(map[string]AccessConditionRecord),
		ledger:     ledger,
	}
}

func makeConditionKey(tenantID, conditionID string) string {
	return fmt.Sprintf("%s:%s", strings.TrimSpace(tenantID), strings.TrimSpace(conditionID))
}

// CreateCondition registers a new access condition and logs an initial audit record.
func (r *AccessConditionRegistry) CreateCondition(cond AccessConditionRecord, actorSubject, reason string, at time.Time) error {
	if cond.TenantID() == "" || cond.ConditionID() == "" {
		return ErrBlankConditionID
	}

	key := makeConditionKey(cond.TenantID(), cond.ConditionID())

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.conditions[key]; exists {
		return ErrDuplicateConditionID
	}

	r.conditions[key] = cond

	audit := AccessConditionAuditRecord{
		RecordID:        fmt.Sprintf("hcnd_%s_%d", cond.ConditionID(), at.UTC().UnixNano()),
		TenantID:        cond.TenantID(),
		ConditionID:     cond.ConditionID(),
		Subject:         cond.Subject(),
		Transition:      "CONDITION_CREATED",
		PriorGeneration: 0,
		NewGeneration:   cond.Generation(),
		NewValidTo:      cond.ValidTo(),
		NewStatus:       cond.Status(),
		ActorSubject:    strings.TrimSpace(actorSubject),
		Reason:          strings.TrimSpace(reason),
		RecordedAt:      at.UTC(),
	}
	return r.ledger.AppendRecord(audit)
}

// GetCondition retrieves an access condition by tenant and ID.
func (r *AccessConditionRegistry) GetCondition(tenantID, conditionID string) (AccessConditionRecord, error) {
	key := makeConditionKey(tenantID, conditionID)

	r.mu.RLock()
	defer r.mu.RUnlock()

	c, exists := r.conditions[key]
	if !exists {
		return AccessConditionRecord{}, ErrAccessConditionNotFound
	}
	return c, nil
}

// ChangeSponsor transfers internal sponsorship to a new manager, increments the generation counter,
// and immediately renders existing caller sessions stale.
func (r *AccessConditionRegistry) ChangeSponsor(tenantID, conditionID, newSponsorID, actorSubject, reason string, at time.Time) (AccessConditionRecord, error) {
	key := makeConditionKey(tenantID, conditionID)

	r.mu.Lock()
	defer r.mu.Unlock()

	current, exists := r.conditions[key]
	if !exists {
		return AccessConditionRecord{}, ErrAccessConditionNotFound
	}
	if !current.IsActive() {
		return AccessConditionRecord{}, ErrAccessConditionInactive
	}
	if !current.IsValidAt(at) {
		return AccessConditionRecord{}, ErrAccessConditionExpired
	}

	trimmedNewSponsor := strings.TrimSpace(newSponsorID)
	if err := ValidateInternalSponsor(trimmedNewSponsor); err != nil {
		return AccessConditionRecord{}, err
	}
	if trimmedNewSponsor == current.SponsorID() {
		return AccessConditionRecord{}, ErrSponsorUnchanged
	}

	priorSponsor := current.SponsorID()
	priorGen := current.Generation()
	newGen := priorGen + 1

	updated := current
	updated.sponsorID = trimmedNewSponsor
	updated.generation = newGen
	updated.updatedAt = at.UTC()

	r.conditions[key] = updated

	audit := AccessConditionAuditRecord{
		RecordID:        fmt.Sprintf("hcnd_%s_%d", current.ConditionID(), at.UTC().UnixNano()),
		TenantID:        current.TenantID(),
		ConditionID:     current.ConditionID(),
		Subject:         current.Subject(),
		Transition:      "SPONSOR_CHANGED",
		PriorSponsorID:  priorSponsor,
		NewSponsorID:    trimmedNewSponsor,
		PriorGeneration: priorGen,
		NewGeneration:   newGen,
		NewStatus:       updated.Status(),
		ActorSubject:    strings.TrimSpace(actorSubject),
		Reason:          strings.TrimSpace(reason),
		RecordedAt:      at.UTC(),
	}
	if err := r.ledger.AppendRecord(audit); err != nil {
		return AccessConditionRecord{}, err
	}

	return updated, nil
}

// RenewAccess extends the validity window of an active condition by up to 7 days,
// increments the generation counter, and records the renewal event.
func (r *AccessConditionRegistry) RenewAccess(tenantID, conditionID string, extension time.Duration, actorSubject, reason string, at time.Time) (AccessConditionRecord, error) {
	key := makeConditionKey(tenantID, conditionID)

	r.mu.Lock()
	defer r.mu.Unlock()

	current, exists := r.conditions[key]
	if !exists {
		return AccessConditionRecord{}, ErrAccessConditionNotFound
	}
	if current.Status() == AccessConditionRevoked {
		return AccessConditionRecord{}, ErrAccessConditionRevoked
	}
	if current.Status() == AccessConditionSuspended {
		return AccessConditionRecord{}, ErrAccessConditionSuspended
	}
	if extension <= 0 || extension > MaxRenewalExtension {
		return AccessConditionRecord{}, ErrRenewalDurationExceeded
	}

	priorGen := current.Generation()
	newGen := priorGen + 1
	newValidTo := current.ValidTo().Add(extension)

	updated := current
	updated.validTo = newValidTo
	updated.status = AccessConditionActive
	updated.generation = newGen
	updated.renewalCount++
	updated.lastRenewedAt = at.UTC()
	updated.updatedAt = at.UTC()

	r.conditions[key] = updated

	audit := AccessConditionAuditRecord{
		RecordID:        fmt.Sprintf("hcnd_%s_%d", current.ConditionID(), at.UTC().UnixNano()),
		TenantID:        current.TenantID(),
		ConditionID:     current.ConditionID(),
		Subject:         current.Subject(),
		Transition:      "ACCESS_RENEWED",
		PriorGeneration: priorGen,
		NewGeneration:   newGen,
		NewValidTo:      newValidTo,
		NewStatus:       AccessConditionActive,
		ActorSubject:    strings.TrimSpace(actorSubject),
		Reason:          strings.TrimSpace(reason),
		RecordedAt:      at.UTC(),
	}
	if err := r.ledger.AppendRecord(audit); err != nil {
		return AccessConditionRecord{}, err
	}

	return updated, nil
}

// DeactivateAccess revokes an access condition in memory, increments the generation counter,
// and captures the deactivation audit event.
func (r *AccessConditionRegistry) DeactivateAccess(tenantID, conditionID, actorSubject, reason string, at time.Time) (AccessConditionRecord, error) {
	key := makeConditionKey(tenantID, conditionID)

	r.mu.Lock()
	defer r.mu.Unlock()

	current, exists := r.conditions[key]
	if !exists {
		return AccessConditionRecord{}, ErrAccessConditionNotFound
	}
	if current.Status() == AccessConditionRevoked {
		return AccessConditionRecord{}, ErrAccessConditionRevoked
	}

	trimmedReason := strings.TrimSpace(reason)
	if trimmedReason == "" {
		return AccessConditionRecord{}, errors.New("deactivation reason must not be blank")
	}

	priorGen := current.Generation()
	newGen := priorGen + 1

	updated := current
	updated.status = AccessConditionRevoked
	updated.generation = newGen
	updated.revokedAt = at.UTC()
	updated.revocationReason = trimmedReason
	updated.updatedAt = at.UTC()

	r.conditions[key] = updated

	audit := AccessConditionAuditRecord{
		RecordID:        fmt.Sprintf("hcnd_%s_%d", current.ConditionID(), at.UTC().UnixNano()),
		TenantID:        current.TenantID(),
		ConditionID:     current.ConditionID(),
		Subject:         current.Subject(),
		Transition:      "ACCESS_DEACTIVATED",
		PriorGeneration: priorGen,
		NewGeneration:   newGen,
		NewStatus:       AccessConditionRevoked,
		ActorSubject:    strings.TrimSpace(actorSubject),
		Reason:          trimmedReason,
		RecordedAt:      at.UTC(),
	}
	if err := r.ledger.AppendRecord(audit); err != nil {
		return AccessConditionRecord{}, err
	}

	return updated, nil
}

// EvaluateConditionAccess evaluates an external user's request against active access conditions and session token generation.
// Returns:
// - allowed: true if condition is active, within validity window, scope matches, and session token generation is current.
// - denialCategory: explainable non-leaking diagnostic category if denied.
func (r *AccessConditionRegistry) EvaluateConditionAccess(tenantID, conditionID, targetProject, targetSite string, sessionTokenGen int64, at time.Time) (bool, DiagnosticDenialCategory) {
	key := makeConditionKey(tenantID, conditionID)

	r.mu.RLock()
	cond, exists := r.conditions[key]
	r.mu.RUnlock()

	if !exists {
		return false, CategoryDefaultDeny
	}

	// 1. Check status and temporal validity
	if cond.Status() == AccessConditionRevoked || cond.Status() == AccessConditionSuspended {
		return false, CategoryIdentityInactive
	}
	if !cond.IsValidAt(at) {
		return false, CategoryIdentityInactive
	}

	// 2. Check scope match
	if !cond.ScopeMatches(targetProject, targetSite) {
		return false, CategoryDefaultDeny
	}

	// 3. Stale-session check: if session generation is older than condition generation, deny as stale
	if sessionTokenGen < cond.Generation() {
		return false, CategorySessionStale
	}

	return true, CategoryNone
}
