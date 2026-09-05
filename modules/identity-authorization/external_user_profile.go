// Package localidentity provides local identity, authorization, and directory services for OSHE Platform.
//
// PROVISIONAL GOVERNANCE DECLARATION (H030-004, H030-005, Issue #94):
// Under approved Sole Human Owner decisions H030-004 and H030-005, this file implements
// synthetic local external-user profiles, sponsor-required enrollment, temporal validity,
// minimum profile invariants, and company-administration denial controls.
//
// Strict Prework Invariants:
// 1. Synthetic Local Users Only: All identities are purely synthetic fixtures (usr_ext_synth_*).
//    Zero real user/customer data is processed or stored.
// 2. No External IdP Integration: Operates strictly in-memory. Zero OIDC, SAML, LDAP, or cloud IdP sync.
// 3. Mandatory Internal Sponsor: Every external user MUST be enrolled under an authorized internal manager (usr_*).
//    External self-sponsorship and chain delegation are strictly rejected.
// 4. Company Administration Denial: External users are categorically barred from holding internal Company,
//    Business Unit, or Tenant administrative roles (TenantAdmin, ProjectManager).
// 5. Profile Minimization: Exposes only sanitized operational display attributes and opaque synthetic contact references.
//    Strictly omits passwords, tokens, national IDs, personal emails, personal phone numbers, and private PII.
// 6. Zero Final User-Model Binding: This implementation is an in-memory prework simulation candidate
//    pending formal sovereign architecture gates (H030-007, H030-008).
package localidentity

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// ExternalUserType classifies approved external-user categories under Issue #94.
type ExternalUserType string

const (
	ExternalTypeTemporaryWorker   ExternalUserType = "TEMPORARY_WORKER"
	ExternalTypeSiteLocalWorker   ExternalUserType = "SITE_LOCAL_WORKER"
	ExternalTypeContractorWorker  ExternalUserType = "CONTRACTOR_WORKER"
	ExternalTypeClientInspector   ExternalUserType = "CLIENT_INSPECTOR"
	ExternalTypeAuditor           ExternalUserType = "EXTERNAL_AUDITOR"
	ExternalTypePartnerSpecialist ExternalUserType = "PARTNER_SPECIALIST"
)

// KnownExternalUserTypes represents the authoritative set of recognized external user types.
var KnownExternalUserTypes = map[ExternalUserType]bool{
	ExternalTypeTemporaryWorker:   true,
	ExternalTypeSiteLocalWorker:   true,
	ExternalTypeContractorWorker:  true,
	ExternalTypeClientInspector:   true,
	ExternalTypeAuditor:           true,
	ExternalTypePartnerSpecialist: true,
}

// EnrollmentStatus represents the operational enrollment status of an external user.
type EnrollmentStatus string

const (
	EnrollmentStatusActive  EnrollmentStatus = "ACTIVE"
	EnrollmentStatusRevoked EnrollmentStatus = "REVOKED"
	EnrollmentStatusExpired EnrollmentStatus = "EXPIRED"
)

var (
	// ErrInvalidExternalUserType indicates an unapproved external user type.
	ErrInvalidExternalUserType = errors.New("invalid or unrecognized external user type")
	// ErrMissingInternalSponsor indicates missing or blank internal sponsor manager.
	ErrMissingInternalSponsor = errors.New("mandatory internal sponsor manager is required")
	// ErrInvalidInternalSponsor indicates sponsor is not an authorized internal user (e.g. external user self-sponsoring).
	ErrInvalidInternalSponsor = errors.New("sponsor must be an authorized internal user identity (usr_ prefix without ext_ designation)")
	// ErrCompanyAdminDenied indicates an illegal attempt to assign company administration to an external user.
	ErrCompanyAdminDenied = errors.New("external user cannot hold internal company administration or project manager roles")
	// ErrPIIDetected indicates personal identifiable information (email, phone, national ID) was detected in profile.
	ErrPIIDetected = errors.New("profile minimization violation: personal PII detected in external user profile")
	// ErrEnrollmentRevoked indicates the external user enrollment has been revoked.
	ErrEnrollmentRevoked = errors.New("external user enrollment is revoked")
	// ErrEnrollmentExpired indicates the external user enrollment validity window has elapsed.
	ErrEnrollmentExpired = errors.New("external user enrollment has expired")
	// ErrInvalidTimeWindow indicates valid_to is not strictly after valid_from.
	ErrInvalidTimeWindow = errors.New("valid_to must be strictly after valid_from")
	// ErrExternalUserNotFound indicates the external user record does not exist.
	ErrExternalUserNotFound = errors.New("external user profile not found")
	// ErrDuplicateExternalUser indicates the synthetic subject is already enrolled in the tenant.
	ErrDuplicateExternalUser = errors.New("external user subject already enrolled for tenant")
	// ErrScopeNotGranted indicates an attempted action falls outside the external user's bounded scopes.
	ErrScopeNotGranted = errors.New("requested operation is outside the external user's bounded scope grant")
)

// ValidateExternalUserType verifies that the user type is one of the recognized types.
func ValidateExternalUserType(t ExternalUserType) error {
	if !KnownExternalUserTypes[t] {
		return ErrInvalidExternalUserType
	}
	return nil
}

// ValidateInternalSponsor asserts that the sponsor is non-blank, references an internal user (usr_*),
// and is NOT an external user itself (prohibiting chain or self-sponsorship).
func ValidateInternalSponsor(sponsorID string) error {
	trimmed := strings.TrimSpace(sponsorID)
	if trimmed == "" {
		return ErrMissingInternalSponsor
	}
	if !strings.HasPrefix(trimmed, "usr_") && !strings.HasPrefix(trimmed, "usr-") {
		return ErrInvalidInternalSponsor
	}
	// Prohibit external users from acting as internal sponsors
	if strings.Contains(trimmed, "_ext_") || strings.Contains(trimmed, "-ext-") || strings.HasPrefix(trimmed, "usr_ext") {
		return fmt.Errorf("%w: external identity %q cannot sponsor other external users", ErrInvalidInternalSponsor, trimmed)
	}
	return nil
}

// AssertNoCompanyAdministration asserts that external users cannot be granted internal administrative roles.
func AssertNoCompanyAdministration(userType ExternalUserType, role Role) error {
	if role == RoleTenantAdmin || role == RoleProjectManager {
		return fmt.Errorf("%w: external user type %s cannot be assigned role %s", ErrCompanyAdminDenied, userType, role)
	}
	return nil
}

// ValidateProfileMinimization asserts that display name and contact references do not contain
// real email addresses, phone numbers, or national identification patterns.
func ValidateProfileMinimization(displayName, contactReference string) error {
	combined := strings.ToLower(displayName + " " + contactReference)
	if strings.Contains(combined, "@") {
		return fmt.Errorf("%w: raw email addresses are prohibited in external profile fields", ErrPIIDetected)
	}
	if strings.Contains(combined, "+66") || strings.Contains(combined, "08") || strings.Contains(combined, "phone") {
		return fmt.Errorf("%w: raw phone numbers are prohibited in external profile fields", ErrPIIDetected)
	}
	if strings.Contains(combined, "id:") || strings.Contains(combined, "citizen") || strings.Contains(combined, "passport") {
		return fmt.Errorf("%w: national/government identity numbers are prohibited", ErrPIIDetected)
	}
	return nil
}

// ExternalUserProfile models an in-memory, synthetic external user profile.
type ExternalUserProfile struct {
	subject          string
	tenantID         string
	companyID        string
	userType         ExternalUserType
	sponsorID        string
	organizationName string
	displayName      string
	contactReference string
	validFrom        time.Time
	validTo          time.Time
	status           EnrollmentStatus
	assignedScopes   []ScopeGrant
	createdAt        time.Time
	updatedAt        time.Time
}

// Subject returns the synthetic external user identifier (usr_ext_*).
func (p ExternalUserProfile) Subject() string { return p.subject }

// TenantID returns the authoritative tenant identifier.
func (p ExternalUserProfile) TenantID() string { return p.tenantID }

// CompanyID returns the sponsoring company identifier (cmp_*).
func (p ExternalUserProfile) CompanyID() string { return p.companyID }

// UserType returns the approved external user type.
func (p ExternalUserProfile) UserType() ExternalUserType { return p.userType }

// SponsorID returns the mandatory internal sponsor manager identifier (usr_*).
func (p ExternalUserProfile) SponsorID() string { return p.sponsorID }

// OrganizationName returns the external employer or partner organization name.
func (p ExternalUserProfile) OrganizationName() string { return p.organizationName }

// DisplayName returns the sanitized, minimized operational display name.
func (p ExternalUserProfile) DisplayName() string { return p.displayName }

// ContactReference returns the opaque synthetic contact reference (e.g. ref_synth_*).
func (p ExternalUserProfile) ContactReference() string { return p.contactReference }

// ValidFrom returns the start of the enrollment validity window.
func (p ExternalUserProfile) ValidFrom() time.Time { return p.validFrom }

// ValidTo returns the expiration timestamp of the enrollment validity window.
func (p ExternalUserProfile) ValidTo() time.Time { return p.validTo }

// Status returns the enrollment status (ACTIVE, REVOKED, EXPIRED).
func (p ExternalUserProfile) Status() EnrollmentStatus { return p.status }

// AssignedScopes returns an immutable copy of the explicitly granted operational scopes.
func (p ExternalUserProfile) AssignedScopes() []ScopeGrant {
	if p.assignedScopes == nil {
		return []ScopeGrant{}
	}
	out := make([]ScopeGrant, len(p.assignedScopes))
	copy(out, p.assignedScopes)
	return out
}

// CreatedAt returns profile creation timestamp.
func (p ExternalUserProfile) CreatedAt() time.Time { return p.createdAt }

// UpdatedAt returns profile last update timestamp.
func (p ExternalUserProfile) UpdatedAt() time.Time { return p.updatedAt }

// IsActive returns true if the enrollment is currently in ACTIVE status.
func (p ExternalUserProfile) IsActive() bool { return p.status == EnrollmentStatusActive }

// IsValidAt checks if the enrollment is active and current time falls within [validFrom, validTo].
func (p ExternalUserProfile) IsValidAt(t time.Time) bool {
	if p.status != EnrollmentStatusActive {
		return false
	}
	return !t.Before(p.validFrom) && !t.After(p.validTo)
}

// EffectiveStatus returns the effective status accounting for temporal expiration.
func (p ExternalUserProfile) EffectiveStatus(t time.Time) EnrollmentStatus {
	if p.status == EnrollmentStatusRevoked {
		return EnrollmentStatusRevoked
	}
	if t.After(p.validTo) || t.Before(p.validFrom) {
		return EnrollmentStatusExpired
	}
	return EnrollmentStatusActive
}

// IsScopePermitted evaluates whether target falls within any of the user's assigned scopes.
func (p ExternalUserProfile) IsScopePermitted(target TargetResource) bool {
	for _, scope := range p.assignedScopes {
		if scopeMatches(scope, target) {
			return true
		}
	}
	return false
}

// Revoke transitions enrollment to REVOKED status in memory.
func (p ExternalUserProfile) Revoke(at time.Time) (ExternalUserProfile, ExternalUserAuditRecord, error) {
	if p.status == EnrollmentStatusRevoked {
		return p, ExternalUserAuditRecord{}, ErrEnrollmentRevoked
	}

	updated := p
	updated.status = EnrollmentStatusRevoked
	updated.updatedAt = at.UTC()

	record := ExternalUserAuditRecord{
		RecordID:     fmt.Sprintf("hext_%s_%d", p.subject, at.UTC().UnixNano()),
		TenantID:     p.tenantID,
		Subject:      p.subject,
		UserType:     p.userType,
		SponsorID:    p.sponsorID,
		Transition:   "EXTERNAL_USER_REVOKED",
		ActorSubject: p.sponsorID,
		Reason:       "Sponsor enrollment revocation",
		RecordedAt:   at.UTC(),
	}

	return updated, record, nil
}

// NewExternalUserProfile constructs and validates a new synthetic ExternalUserProfile.
func NewExternalUserProfile(
	subject, tenantID, companyID string,
	userType ExternalUserType,
	sponsorID, organizationName, displayName, contactReference string,
	validFrom, validTo time.Time,
	scopes []ScopeGrant,
) (ExternalUserProfile, error) {
	trimmedSub := strings.TrimSpace(subject)
	if trimmedSub == "" {
		return ExternalUserProfile{}, ErrBlankSubject
	}
	if !strings.HasPrefix(trimmedSub, "usr_") && !strings.HasPrefix(trimmedSub, "usr-") {
		return ExternalUserProfile{}, ErrInvalidSubjectFormat
	}
	trimmedTenant := strings.TrimSpace(tenantID)
	if trimmedTenant == "" {
		return ExternalUserProfile{}, ErrBlankTenantID
	}
	trimmedCompany := strings.TrimSpace(companyID)
	if trimmedCompany == "" {
		return ExternalUserProfile{}, ErrBlankCompanyID
	}
	if err := ValidateExternalUserType(userType); err != nil {
		return ExternalUserProfile{}, err
	}
	if err := ValidateInternalSponsor(sponsorID); err != nil {
		return ExternalUserProfile{}, err
	}
	trimmedOrg := strings.TrimSpace(organizationName)
	if trimmedOrg == "" {
		return ExternalUserProfile{}, errors.New("organization name must not be blank")
	}
	sanitizedName := SanitizeDisplayName(displayName)
	if sanitizedName == "" {
		return ExternalUserProfile{}, ErrBlankDisplayName
	}
	trimmedContact := strings.TrimSpace(contactReference)
	if trimmedContact == "" {
		return ExternalUserProfile{}, errors.New("contact reference must not be blank")
	}

	if err := ValidateProfileMinimization(sanitizedName, trimmedContact); err != nil {
		return ExternalUserProfile{}, err
	}

	if validTo.Before(validFrom) || validTo.Equal(validFrom) {
		return ExternalUserProfile{}, ErrInvalidTimeWindow
	}

	// Clean scopes and enforce tenant match
	var cleanScopes []ScopeGrant
	for _, s := range scopes {
		if s.TenantID != "" && s.TenantID != trimmedTenant {
			return ExternalUserProfile{}, ErrTenantMismatch
		}
		s.TenantID = trimmedTenant
		cleanScopes = append(cleanScopes, s)
	}
	if cleanScopes == nil {
		cleanScopes = []ScopeGrant{}
	}

	now := time.Now().UTC()
	return ExternalUserProfile{
		subject:          trimmedSub,
		tenantID:         trimmedTenant,
		companyID:        trimmedCompany,
		userType:         userType,
		sponsorID:        strings.TrimSpace(sponsorID),
		organizationName: trimmedOrg,
		displayName:      sanitizedName,
		contactReference: trimmedContact,
		validFrom:        validFrom.UTC(),
		validTo:          validTo.UTC(),
		status:           EnrollmentStatusActive,
		assignedScopes:   cleanScopes,
		createdAt:        now,
		updatedAt:        now,
	}, nil
}

// ExternalUserAuditRecord models an immutable audit event for external user enrollment.
type ExternalUserAuditRecord struct {
	RecordID     string           `json:"record_id"`
	TenantID     string           `json:"tenant_id"`
	Subject      string           `json:"subject"`
	UserType     ExternalUserType `json:"user_type"`
	SponsorID    string           `json:"sponsor_id"`
	Transition   string           `json:"transition"`
	ActorSubject string           `json:"actor_subject"`
	Reason       string           `json:"reason"`
	RecordedAt   time.Time        `json:"recorded_at"`
}

// ExternalUserLedger provides an in-memory, thread-safe, append-only audit trail for external user events.
type ExternalUserLedger struct {
	mu      sync.RWMutex
	records []ExternalUserAuditRecord
}

// NewExternalUserLedger initializes an empty in-memory ledger.
func NewExternalUserLedger() *ExternalUserLedger {
	return &ExternalUserLedger{
		records: make([]ExternalUserAuditRecord, 0),
	}
}

// AppendRecord appends an immutable audit record to the ledger.
func (l *ExternalUserLedger) AppendRecord(record ExternalUserAuditRecord) error {
	if record.TenantID == "" || record.Subject == "" {
		return ErrBlankSubject
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.records = append(l.records, record)
	return nil
}

// GetAuditTrail retrieves the complete audit history for a subject strictly within tenant boundaries.
func (l *ExternalUserLedger) GetAuditTrail(tenantID, subject string) ([]ExternalUserAuditRecord, error) {
	tTenant := strings.TrimSpace(tenantID)
	if tTenant == "" {
		return nil, ErrBlankTenantID
	}
	tSub := strings.TrimSpace(subject)
	if tSub == "" {
		return nil, ErrBlankSubject
	}

	l.mu.RLock()
	defer l.mu.RUnlock()

	var results []ExternalUserAuditRecord
	for _, rec := range l.records {
		if rec.TenantID == tTenant && rec.Subject == tSub {
			results = append(results, rec)
		}
	}
	return results, nil
}

// ExternalUserRegistry provides a thread-safe, in-memory store for enrolled external user profiles.
type ExternalUserRegistry struct {
	mu       sync.RWMutex
	profiles map[string]ExternalUserProfile // key: "tenantID:subject"
	ledger   *ExternalUserLedger
}

// NewExternalUserRegistry constructs an empty in-memory registry.
func NewExternalUserRegistry(ledger *ExternalUserLedger) *ExternalUserRegistry {
	if ledger == nil {
		ledger = NewExternalUserLedger()
	}
	return &ExternalUserRegistry{
		profiles: make(map[string]ExternalUserProfile),
		ledger:   ledger,
	}
}

func makeExternalUserKey(tenantID, subject string) string {
	return fmt.Sprintf("%s:%s", strings.TrimSpace(tenantID), strings.TrimSpace(subject))
}

// EnrollExternalUser registers a new external user profile and appends an audit record.
func (r *ExternalUserRegistry) EnrollExternalUser(profile ExternalUserProfile, actorSubject, reason string, at time.Time) error {
	if profile.TenantID() == "" || profile.Subject() == "" {
		return ErrBlankSubject
	}

	key := makeExternalUserKey(profile.TenantID(), profile.Subject())

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.profiles[key]; exists {
		return ErrDuplicateExternalUser
	}

	r.profiles[key] = profile

	record := ExternalUserAuditRecord{
		RecordID:     fmt.Sprintf("hext_%s_%d", profile.Subject(), at.UTC().UnixNano()),
		TenantID:     profile.TenantID(),
		Subject:      profile.Subject(),
		UserType:     profile.UserType(),
		SponsorID:    profile.SponsorID(),
		Transition:   "EXTERNAL_USER_ENROLLED",
		ActorSubject: strings.TrimSpace(actorSubject),
		Reason:       strings.TrimSpace(reason),
		RecordedAt:   at.UTC(),
	}
	return r.ledger.AppendRecord(record)
}

// RevokeEnrollment marks an external user enrollment as REVOKED and captures an audit record.
func (r *ExternalUserRegistry) RevokeEnrollment(tenantID, subject, actorSubject, reason string, at time.Time) (ExternalUserProfile, error) {
	key := makeExternalUserKey(tenantID, subject)

	r.mu.Lock()
	defer r.mu.Unlock()

	current, exists := r.profiles[key]
	if !exists {
		return ExternalUserProfile{}, ErrExternalUserNotFound
	}

	revoked, audit, err := current.Revoke(at)
	if err != nil {
		return ExternalUserProfile{}, err
	}
	audit.ActorSubject = strings.TrimSpace(actorSubject)
	audit.Reason = strings.TrimSpace(reason)

	r.profiles[key] = revoked
	if err := r.ledger.AppendRecord(audit); err != nil {
		return ExternalUserProfile{}, err
	}

	return revoked, nil
}

// GetExternalUser retrieves an external user profile by tenant and subject.
func (r *ExternalUserRegistry) GetExternalUser(tenantID, subject string) (ExternalUserProfile, error) {
	key := makeExternalUserKey(tenantID, subject)

	r.mu.RLock()
	defer r.mu.RUnlock()

	p, exists := r.profiles[key]
	if !exists {
		return ExternalUserProfile{}, ErrExternalUserNotFound
	}
	return p, nil
}

// ListActiveExternalUsers returns all active, time-valid external users in a tenant at time 'at'.
func (r *ExternalUserRegistry) ListActiveExternalUsers(tenantID string, at time.Time) ([]ExternalUserProfile, error) {
	tTenant := strings.TrimSpace(tenantID)
	if tTenant == "" {
		return nil, ErrBlankTenantID
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	var results []ExternalUserProfile
	for _, p := range r.profiles {
		if p.TenantID() == tTenant && p.IsValidAt(at) {
			results = append(results, p)
		}
	}
	return results, nil
}

// ListBySponsor returns all external users enrolled under a specific internal sponsor manager.
func (r *ExternalUserRegistry) ListBySponsor(tenantID, sponsorID string) ([]ExternalUserProfile, error) {
	tTenant := strings.TrimSpace(tenantID)
	if tTenant == "" {
		return nil, ErrBlankTenantID
	}
	tSponsor := strings.TrimSpace(sponsorID)
	if tSponsor == "" {
		return nil, ErrMissingInternalSponsor
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	var results []ExternalUserProfile
	for _, p := range r.profiles {
		if p.TenantID() == tTenant && p.SponsorID() == tSponsor {
			results = append(results, p)
		}
	}
	return results, nil
}
