// Package localidentity provides local identity, authorization, and directory services for OSHE Platform.
//
// PROVISIONAL GOVERNANCE DECLARATION (H030-003, H030-004, H030-005, Issue #88):
// Under approved Sole Human Owner decisions H030-003, H030-004, and H030-005, this file
// establishes the local directory resolution engine, profile lifecycle transitions,
// false-merge prevention protocols, and append-only context history ledger.
//
// Core Invariants:
// 1. Duplicate Identifier Collision Rejection: Profiles with duplicate IDs fail registration closed.
// 2. Explicit False-Merge Prohibition: Distinct subjects (usr_*) can NEVER be merged, aliased,
//    or consolidated, even if they share display names or designations. A profile's subject is permanently immutable.
// 3. Structural Identity Immutability: ProfileID, Subject, TenantID, CompanyID, ProjectID, and SiteID
//    cannot be modified in-place; only sanitized non-structural attributes may be updated.
// 4. Safe Inactivation & Active Default Filtering: Inactive profiles are omitted from active directory discovery.
// 5. Append-Only Context History: All updates, status changes, and inactivations emit immutable audit records.
// 6. Zero Auth/Credential Mutation: Complete separation from roles, grants, session tokens, and credentials.
// 7. Strict Tenant Boundary Isolation: Cross-tenant history and profile queries fail closed with zero data leakage.
package localidentity

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"
)

var (
	// ErrDuplicateIdentifierCollision indicates collision when registering profiles with identical identifiers.
	ErrDuplicateIdentifierCollision = errors.New("duplicate identifier collision detected")
	// ErrFalseMergeProhibited indicates an illegal attempt to merge, alias, or consolidate two distinct subjects.
	ErrFalseMergeProhibited = errors.New("false-merge prohibited: distinct subjects cannot be merged, aliased, or consolidated")
	// ErrStructuralIdentityImmutable indicates an attempt to mutate structural identifiers (ProfileID, Subject, TenantID, CompanyID, ProjectID, SiteID).
	ErrStructuralIdentityImmutable = errors.New("structural identity is immutable: subject, tenant, company, and project cannot be changed")
	// ErrProfileInactive indicates an operation was rejected because the profile is currently inactive.
	ErrProfileInactive = errors.New("directory profile is inactive")
	// ErrNoNonStructuralChanges indicates that an update request contains identical values to current state.
	ErrNoNonStructuralChanges = errors.New("no non-structural changes detected")
)

// AssertNoFalseMerge validates that an operation does not attempt to bind, alias, or merge
// an existing profile with a different singular identity subject.
func AssertNoFalseMerge(existingProfile DirectoryProfile, candidateSubject string) error {
	trimmedCandidate := strings.TrimSpace(candidateSubject)
	if trimmedCandidate == "" {
		return ErrBlankSubject
	}
	if existingProfile.Subject() != trimmedCandidate {
		return fmt.Errorf("%w: profile %q is bound to subject %q; cannot bind or merge with subject %q",
			ErrFalseMergeProhibited, existingProfile.ProfileID(), existingProfile.Subject(), trimmedCandidate)
	}
	return nil
}

// AssertDistinctSubjects explicitly enforces that two subjects must be treated as independent,
// non-mergeable identities even if their display names or job titles are identical.
func AssertDistinctSubjects(subjectA, subjectB string) error {
	trimmedA := strings.TrimSpace(subjectA)
	trimmedB := strings.TrimSpace(subjectB)
	if trimmedA == "" || trimmedB == "" {
		return ErrBlankSubject
	}
	if trimmedA != trimmedB {
		// They are distinct subjects; any attempt to assert identity equivalence or merge them fails
		return fmt.Errorf("%w: subject %q and subject %q are distinct independent identities",
			ErrFalseMergeProhibited, trimmedA, trimmedB)
	}
	return nil
}

// ProfileNonStructuralUpdate encapsulates mutable non-structural operational attributes.
// Structural fields (ProfileID, Subject, TenantID, CompanyID, ProjectID, SiteID) are strictly excluded.
type ProfileNonStructuralUpdate struct {
	DisplayName   *string
	JobTitle      *string
	Department    *string
	AssignedAreas []string
}

// DirectoryProfileHistoryRecord captures an immutable, append-only historical audit record
// of a directory profile modification, status change, or inactivation.
type DirectoryProfileHistoryRecord struct {
	RecordID      string            `json:"record_id"`
	TenantID      string            `json:"tenant_id"`
	ProfileID     string            `json:"profile_id"`
	Subject       string            `json:"subject"`
	CompanyID     string            `json:"company_id"`
	ProjectID     string            `json:"project_id,omitempty"`
	SiteID        string            `json:"site_id,omitempty"`
	PreviousState ProfileStatus     `json:"previous_state"`
	NewState      ProfileStatus     `json:"new_state"`
	Transition    string            `json:"transition"`
	ChangedFields map[string]string `json:"changed_fields,omitempty"`
	ActorSubject  string            `json:"actor_subject"`
	Reason        string            `json:"reason"`
	RecordedAt    time.Time         `json:"recorded_at"`
}

// UpdateProfileNonStructural applies sanitized non-structural updates to an active profile in memory.
// Returns the updated profile and an immutable audit record.
func UpdateProfileNonStructural(current DirectoryProfile, update ProfileNonStructuralUpdate, actorSubject, reason string) (DirectoryProfile, DirectoryProfileHistoryRecord, error) {
	if !current.IsActive() {
		return current, DirectoryProfileHistoryRecord{}, ErrProfileInactive
	}

	changedFields := make(map[string]string)
	updated := current

	if update.DisplayName != nil {
		sanitized := SanitizeDisplayName(*update.DisplayName)
		if sanitized == "" {
			return current, DirectoryProfileHistoryRecord{}, ErrBlankDisplayName
		}
		if sanitized != current.DisplayName() {
			changedFields["display_name"] = fmt.Sprintf("%s -> %s", current.DisplayName(), sanitized)
			updated.displayName = sanitized
		}
	}

	if update.JobTitle != nil {
		trimmedTitle := strings.TrimSpace(*update.JobTitle)
		if trimmedTitle != current.JobTitle() {
			changedFields["job_title"] = fmt.Sprintf("%s -> %s", current.JobTitle(), trimmedTitle)
			updated.jobTitle = trimmedTitle
		}
	}

	if update.Department != nil {
		trimmedDept := strings.TrimSpace(*update.Department)
		if trimmedDept != current.Department() {
			changedFields["department"] = fmt.Sprintf("%s -> %s", current.Department(), trimmedDept)
			updated.department = trimmedDept
		}
	}

	if update.AssignedAreas != nil {
		var cleanAreas []string
		for _, a := range update.AssignedAreas {
			t := strings.TrimSpace(a)
			if t != "" {
				cleanAreas = append(cleanAreas, t)
			}
		}
		if cleanAreas == nil {
			cleanAreas = []string{}
		}
		if !reflect.DeepEqual(cleanAreas, current.AssignedAreas()) {
			changedFields["assigned_areas"] = fmt.Sprintf("%v -> %v", current.AssignedAreas(), cleanAreas)
			updated.assignedAreas = cleanAreas
		}
	}

	if len(changedFields) == 0 {
		return current, DirectoryProfileHistoryRecord{}, ErrNoNonStructuralChanges
	}

	now := time.Now().UTC()
	updated.updatedAt = now

	record := DirectoryProfileHistoryRecord{
		RecordID:      fmt.Sprintf("hprof_%s_%d", current.ProfileID(), now.UnixNano()),
		TenantID:      current.TenantID(),
		ProfileID:     current.ProfileID(),
		Subject:       current.Subject(),
		CompanyID:     current.CompanyID(),
		ProjectID:     current.ProjectID(),
		SiteID:        current.SiteID(),
		PreviousState: current.Status(),
		NewState:      updated.Status(),
		Transition:    "PROFILE_UPDATE_ATTRIBUTES",
		ChangedFields: changedFields,
		ActorSubject:  strings.TrimSpace(actorSubject),
		Reason:        strings.TrimSpace(reason),
		RecordedAt:    now,
	}

	return updated, record, nil
}

// InactivateProfileWithHistory transitions an active profile to INACTIVE status in memory,
// capturing an immutable audit record.
func InactivateProfileWithHistory(profile DirectoryProfile, actorSubject, reason string) (DirectoryProfile, DirectoryProfileHistoryRecord, error) {
	if !profile.IsActive() {
		return profile, DirectoryProfileHistoryRecord{}, errors.New("profile is already inactive")
	}

	now := time.Now().UTC()
	inactivated := profile.Deactivate()

	record := DirectoryProfileHistoryRecord{
		RecordID:      fmt.Sprintf("hprof_%s_%d", profile.ProfileID(), now.UnixNano()),
		TenantID:      profile.TenantID(),
		ProfileID:     profile.ProfileID(),
		Subject:       profile.Subject(),
		CompanyID:     profile.CompanyID(),
		ProjectID:     profile.ProjectID(),
		SiteID:        profile.SiteID(),
		PreviousState: profile.Status(),
		NewState:      ProfileStatusInactive,
		Transition:    "PROFILE_INACTIVATE",
		ActorSubject:  strings.TrimSpace(actorSubject),
		Reason:        strings.TrimSpace(reason),
		RecordedAt:    now,
	}

	return inactivated, record, nil
}

// ActivateProfileWithHistory reactivates an inactive profile in memory, capturing an audit record.
func ActivateProfileWithHistory(profile DirectoryProfile, actorSubject, reason string) (DirectoryProfile, DirectoryProfileHistoryRecord, error) {
	if profile.IsActive() {
		return profile, DirectoryProfileHistoryRecord{}, errors.New("profile is already active")
	}

	now := time.Now().UTC()
	activated := profile.Activate()

	record := DirectoryProfileHistoryRecord{
		RecordID:      fmt.Sprintf("hprof_%s_%d", profile.ProfileID(), now.UnixNano()),
		TenantID:      profile.TenantID(),
		ProfileID:     profile.ProfileID(),
		Subject:       profile.Subject(),
		CompanyID:     profile.CompanyID(),
		ProjectID:     profile.ProjectID(),
		SiteID:        profile.SiteID(),
		PreviousState: profile.Status(),
		NewState:      ProfileStatusActive,
		Transition:    "PROFILE_ACTIVATE",
		ActorSubject:  strings.TrimSpace(actorSubject),
		Reason:        strings.TrimSpace(reason),
		RecordedAt:    now,
	}

	return activated, record, nil
}

// DirectoryResolutionLedger provides a thread-safe, in-memory append-only audit trail
// for directory profile lifecycle events and non-structural mutations.
type DirectoryResolutionLedger struct {
	mu      sync.RWMutex
	records []DirectoryProfileHistoryRecord
}

// NewDirectoryResolutionLedger initializes an empty in-memory ledger.
func NewDirectoryResolutionLedger() *DirectoryResolutionLedger {
	return &DirectoryResolutionLedger{
		records: make([]DirectoryProfileHistoryRecord, 0),
	}
}

// AppendRecord appends an immutable historical record to the ledger.
func (l *DirectoryResolutionLedger) AppendRecord(record DirectoryProfileHistoryRecord) error {
	if record.TenantID == "" || record.ProfileID == "" {
		return ErrBlankProfileID
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.records = append(l.records, record)
	return nil
}

// GetProfileHistory retrieves the audit trail for a profile strictly within tenant boundaries.
func (l *DirectoryResolutionLedger) GetProfileHistory(tenantID, profileID string) ([]DirectoryProfileHistoryRecord, error) {
	tTenant := strings.TrimSpace(tenantID)
	if tTenant == "" {
		return nil, ErrBlankTenantID
	}
	tProfile := strings.TrimSpace(profileID)
	if tProfile == "" {
		return nil, ErrBlankProfileID
	}

	l.mu.RLock()
	defer l.mu.RUnlock()

	var results []DirectoryProfileHistoryRecord
	for _, rec := range l.records {
		if rec.TenantID == tTenant && rec.ProfileID == tProfile {
			results = append(results, rec)
		}
	}
	return results, nil
}

// GetSubjectHistory retrieves all profile history entries for a singular subject within a tenant.
func (l *DirectoryResolutionLedger) GetSubjectHistory(tenantID, subject string) ([]DirectoryProfileHistoryRecord, error) {
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

	var results []DirectoryProfileHistoryRecord
	for _, rec := range l.records {
		if rec.TenantID == tTenant && rec.Subject == tSub {
			results = append(results, rec)
		}
	}
	return results, nil
}

// DirectoryResolver coordinates profile registration, non-structural updates, inactivation,
// exact-scope queries, and append-only audit tracking in memory.
type DirectoryResolver struct {
	registry *DirectoryRegistry
	ledger   *DirectoryResolutionLedger
}

// NewDirectoryResolver initializes a new DirectoryResolver.
func NewDirectoryResolver(registry *DirectoryRegistry, ledger *DirectoryResolutionLedger) *DirectoryResolver {
	if registry == nil {
		registry = NewDirectoryRegistry()
	}
	if ledger == nil {
		ledger = NewDirectoryResolutionLedger()
	}
	return &DirectoryResolver{
		registry: registry,
		ledger:   ledger,
	}
}

// RegisterProfile stores a new directory profile, asserting duplicate collision avoidance
// and capturing an initial audit record.
func (r *DirectoryResolver) RegisterProfile(profile DirectoryProfile, actorSubject, reason string) error {
	// Attempt registration in registry; fails with ErrDuplicateProfileID if collision occurs
	if err := r.registry.RegisterProfile(profile); err != nil {
		if errors.Is(err, ErrDuplicateProfileID) {
			return fmt.Errorf("%w: %v", ErrDuplicateIdentifierCollision, err)
		}
		return err
	}

	// Capture initial registration history record
	now := time.Now().UTC()
	initRecord := DirectoryProfileHistoryRecord{
		RecordID:      fmt.Sprintf("hprof_%s_%d", profile.ProfileID(), now.UnixNano()),
		TenantID:      profile.TenantID(),
		ProfileID:     profile.ProfileID(),
		Subject:       profile.Subject(),
		CompanyID:     profile.CompanyID(),
		ProjectID:     profile.ProjectID(),
		SiteID:        profile.SiteID(),
		PreviousState: "",
		NewState:      profile.Status(),
		Transition:    "PROFILE_INITIAL_REGISTER",
		ActorSubject:  strings.TrimSpace(actorSubject),
		Reason:        strings.TrimSpace(reason),
		RecordedAt:    now,
	}
	return r.ledger.AppendRecord(initRecord)
}

// ResolveProfile retrieves a single profile by tenant and profile ID.
func (r *DirectoryResolver) ResolveProfile(tenantID, profileID string) (DirectoryProfile, error) {
	return r.registry.GetProfile(tenantID, profileID)
}

// ResolveProfilesBySubject retrieves all profiles for a singular subject under a tenant.
func (r *DirectoryResolver) ResolveProfilesBySubject(tenantID, subject string, includeInactive bool) ([]DirectoryProfile, error) {
	return r.registry.ListProfilesBySubject(tenantID, subject, includeInactive)
}

// SearchActiveDirectory executes an active, exact-scope directory query with cross-scope isolation.
func (r *DirectoryResolver) SearchActiveDirectory(query DirectoryQuery) ([]DirectoryProfile, error) {
	return r.registry.SearchDirectory(query)
}

// UpdateProfileAttributes applies non-structural updates to a profile and logs the audit record.
func (r *DirectoryResolver) UpdateProfileAttributes(tenantID, profileID string, update ProfileNonStructuralUpdate, actorSubject, reason string) (DirectoryProfile, error) {
	current, err := r.registry.GetProfile(tenantID, profileID)
	if err != nil {
		return DirectoryProfile{}, err
	}

	updated, record, err := UpdateProfileNonStructural(current, update, actorSubject, reason)
	if err != nil {
		return DirectoryProfile{}, err
	}

	// Store updated profile in registry
	pKey := makeProfileKey(tenantID, profileID)
	r.registry.mu.Lock()
	r.registry.profiles[pKey] = updated
	r.registry.mu.Unlock()

	// Append history record
	if err := r.ledger.AppendRecord(record); err != nil {
		return DirectoryProfile{}, err
	}

	return updated, nil
}

// InactivateProfile sets an active profile to INACTIVE status and logs the audit record.
func (r *DirectoryResolver) InactivateProfile(tenantID, profileID, actorSubject, reason string) (DirectoryProfile, error) {
	current, err := r.registry.GetProfile(tenantID, profileID)
	if err != nil {
		return DirectoryProfile{}, err
	}

	inactivated, record, err := InactivateProfileWithHistory(current, actorSubject, reason)
	if err != nil {
		return DirectoryProfile{}, err
	}

	pKey := makeProfileKey(tenantID, profileID)
	r.registry.mu.Lock()
	r.registry.profiles[pKey] = inactivated
	r.registry.mu.Unlock()

	if err := r.ledger.AppendRecord(record); err != nil {
		return DirectoryProfile{}, err
	}

	return inactivated, nil
}

// GetProfileHistory retrieves the audit history for a profile.
func (r *DirectoryResolver) GetProfileHistory(tenantID, profileID string) ([]DirectoryProfileHistoryRecord, error) {
	return r.ledger.GetProfileHistory(tenantID, profileID)
}

// GetSubjectHistory retrieves all audit history for a singular subject across its profiles.
func (r *DirectoryResolver) GetSubjectHistory(tenantID, subject string) ([]DirectoryProfileHistoryRecord, error) {
	return r.ledger.GetSubjectHistory(tenantID, subject)
}
