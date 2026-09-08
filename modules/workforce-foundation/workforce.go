// Package workforcefoundation provides a synthetic, in-memory Workforce
// foundation. It does not authenticate, authorize, persist, or synchronize
// identities; trusted identity is an immutable opaque usr_ reference only.
package workforcefoundation

import (
	"errors"
	"strings"
	"sync"
	"time"
)

var (
	ErrBlankField          = errors.New("required field is blank")
	ErrInvalidIdentityRef  = errors.New("trusted identity reference must be an opaque usr_ reference")
	ErrDuplicatePerson     = errors.New("person already exists in tenant")
	ErrDuplicateEmployment = errors.New("employment already exists in tenant")
	ErrPersonNotFound      = errors.New("person not found")
	ErrTenantMismatch      = errors.New("cross-tenant relationship is denied")
	ErrInvalidWindow       = errors.New("employment validity window is invalid")
)

// Person stores only a tenant-scoped, synthetic person identity and an opaque
// trusted-identity reference. It deliberately contains no credentials, grants,
// authentication claims, contact details, or implicit membership.
type Person struct {
	ID                 string
	TenantID           string
	TrustedIdentityRef string
}

// Employment is an explicit descriptive relationship. Its OwnerSubjectRef is
// attribution only; it grants neither operational authority nor membership.
type Employment struct {
	ID              string
	TenantID        string
	PersonID        string
	CompanyID       string
	OwnerSubjectRef string
	ValidFrom       time.Time
	ValidTo         time.Time
}

// HistoryRecord is append-only attribution for registration events.
type HistoryRecord struct {
	TenantID   string
	EntityKind string
	EntityID   string
	ActorRef   string
	OccurredAt time.Time
}

// Registry is a thread-safe local fixture store. It has zero database or
// external identity-provider effects.
type Registry struct {
	mu          sync.RWMutex
	people      map[string]Person
	employments map[string]Employment
	history     []HistoryRecord
}

func NewRegistry() *Registry {
	return &Registry{people: make(map[string]Person), employments: make(map[string]Employment)}
}

func key(tenantID, id string) string { return strings.TrimSpace(tenantID) + ":" + strings.TrimSpace(id) }

func require(value string) error {
	if strings.TrimSpace(value) == "" {
		return ErrBlankField
	}
	return nil
}

func validSubjectRef(value string) bool {
	value = strings.TrimSpace(value)
	return strings.HasPrefix(value, "usr_") && len(value) > len("usr_")
}

// RegisterPerson records a new immutable person-to-trusted-identity reference.
func (r *Registry) RegisterPerson(person Person, actorRef string, at time.Time) error {
	if r == nil { return errors.New("registry is nil") }
	for _, value := range []string{person.ID, person.TenantID, actorRef} {
		if err := require(value); err != nil { return err }
	}
	if !validSubjectRef(person.TrustedIdentityRef) || !validSubjectRef(actorRef) { return ErrInvalidIdentityRef }
	person.ID, person.TenantID, person.TrustedIdentityRef = strings.TrimSpace(person.ID), strings.TrimSpace(person.TenantID), strings.TrimSpace(person.TrustedIdentityRef)
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.people[key(person.TenantID, person.ID)]; exists { return ErrDuplicatePerson }
	r.people[key(person.TenantID, person.ID)] = person
	r.history = append(r.history, HistoryRecord{TenantID: person.TenantID, EntityKind: "PERSON_REGISTERED", EntityID: person.ID, ActorRef: strings.TrimSpace(actorRef), OccurredAt: at.UTC()})
	return nil
}

// GetPerson default-denies absent and cross-tenant lookup with one non-leaking error.
func (r *Registry) GetPerson(tenantID, personID string) (Person, error) {
	if r == nil { return Person{}, ErrPersonNotFound }
	r.mu.RLock(); defer r.mu.RUnlock()
	person, ok := r.people[key(tenantID, personID)]
	if !ok { return Person{}, ErrPersonNotFound }
	return person, nil
}

// RegisterEmployment requires an existing same-tenant person and records
// explicit ownership attribution without creating a role, grant, or membership.
func (r *Registry) RegisterEmployment(employment Employment, actorRef string, at time.Time) error {
	if r == nil { return errors.New("registry is nil") }
	for _, value := range []string{employment.ID, employment.TenantID, employment.PersonID, employment.CompanyID, employment.OwnerSubjectRef, actorRef} {
		if err := require(value); err != nil { return err }
	}
	if !validSubjectRef(employment.OwnerSubjectRef) || !validSubjectRef(actorRef) { return ErrInvalidIdentityRef }
	if !employment.ValidTo.After(employment.ValidFrom) { return ErrInvalidWindow }
	employment.ID, employment.TenantID, employment.PersonID, employment.CompanyID, employment.OwnerSubjectRef = strings.TrimSpace(employment.ID), strings.TrimSpace(employment.TenantID), strings.TrimSpace(employment.PersonID), strings.TrimSpace(employment.CompanyID), strings.TrimSpace(employment.OwnerSubjectRef)
	r.mu.Lock(); defer r.mu.Unlock()
	if _, exists := r.people[key(employment.TenantID, employment.PersonID)]; !exists { return ErrPersonNotFound }
	if _, exists := r.employments[key(employment.TenantID, employment.ID)]; exists { return ErrDuplicateEmployment }
	r.employments[key(employment.TenantID, employment.ID)] = employment
	r.history = append(r.history, HistoryRecord{TenantID: employment.TenantID, EntityKind: "EMPLOYMENT_REGISTERED", EntityID: employment.ID, ActorRef: strings.TrimSpace(actorRef), OccurredAt: at.UTC()})
	return nil
}

// History returns a copy of only the requested tenant's append-only attribution.
func (r *Registry) History(tenantID string) []HistoryRecord {
	if r == nil { return nil }
	r.mu.RLock(); defer r.mu.RUnlock()
	result := make([]HistoryRecord, 0)
	for _, record := range r.history { if record.TenantID == strings.TrimSpace(tenantID) { result = append(result, record) } }
	return result
}
