// Package workforcefoundation provides a synthetic, in-memory Workforce
// foundation. It does not authenticate, authorize, persist, or synchronize
// identities; trusted identity is an immutable opaque usr_ reference only.
package workforcefoundation

import (
	"errors"
	"fmt"
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

// entityKey prevents delimiter injection and key collision across tenants.
type entityKey struct {
	tenantID string
	id       string
}

func makeKey(tenantID, id string) entityKey {
	return entityKey{
		tenantID: strings.TrimSpace(tenantID),
		id:       strings.TrimSpace(id),
	}
}

// Registry is a thread-safe local fixture store. It has zero database or
// external identity-provider effects.
type Registry struct {
	mu          sync.RWMutex
	people      map[entityKey]Person
	employments map[entityKey]Employment
	history     []HistoryRecord
}

func NewRegistry() *Registry {
	return &Registry{
		people:      make(map[entityKey]Person),
		employments: make(map[entityKey]Employment),
	}
}

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
	pKey := makeKey(person.TenantID, person.ID)
	if _, exists := r.people[pKey]; exists { return ErrDuplicatePerson }
	r.people[pKey] = person
	r.history = append(r.history, HistoryRecord{TenantID: person.TenantID, EntityKind: "PERSON_REGISTERED", EntityID: person.ID, ActorRef: strings.TrimSpace(actorRef), OccurredAt: at.UTC()})
	return nil
}

// GetPerson default-denies absent and cross-tenant lookup with one non-leaking error.
func (r *Registry) GetPerson(tenantID, personID string) (Person, error) {
	if r == nil { return Person{}, ErrPersonNotFound }
	r.mu.RLock()
	defer r.mu.RUnlock()
	person, ok := r.people[makeKey(tenantID, personID)]
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
	r.mu.Lock()
	defer r.mu.Unlock()
	personKey := makeKey(employment.TenantID, employment.PersonID)
	if _, exists := r.people[personKey]; !exists {
		for k := range r.people {
			if k.id == employment.PersonID && k.tenantID != employment.TenantID {
				return fmt.Errorf("%w: %w", ErrTenantMismatch, ErrPersonNotFound)
			}
		}
		return ErrPersonNotFound
	}
	empKey := makeKey(employment.TenantID, employment.ID)
	if _, exists := r.employments[empKey]; exists { return ErrDuplicateEmployment }
	r.employments[empKey] = employment
	r.history = append(r.history, HistoryRecord{TenantID: employment.TenantID, EntityKind: "EMPLOYMENT_REGISTERED", EntityID: employment.ID, ActorRef: strings.TrimSpace(actorRef), OccurredAt: at.UTC()})
	return nil
}

// History returns a copy of only the requested tenant's append-only attribution.
func (r *Registry) History(tenantID string) []HistoryRecord {
	if r == nil { return nil }
	r.mu.RLock()
	defer r.mu.RUnlock()
	tID := strings.TrimSpace(tenantID)
	result := make([]HistoryRecord, 0)
	for _, record := range r.history {
		if record.TenantID == tID {
			result = append(result, record)
		}
	}
	return result
}
