package workforcefoundation

import (
	"errors"
	"testing"
	"time"
)

var instant = time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)

func person(tenant, id, subject string) Person { return Person{ID: id, TenantID: tenant, TrustedIdentityRef: subject} }

func TestPersonIdentityIsOpaqueTenantScopedAndImmutable(t *testing.T) {
	r := NewRegistry()
	p := person("ten_alpha", "per_alpha", "usr_identity_alpha")
	if err := r.RegisterPerson(p, "usr_owner_alpha", instant); err != nil { t.Fatal(err) }
	got, err := r.GetPerson("ten_alpha", "per_alpha")
	if err != nil || got.TrustedIdentityRef != "usr_identity_alpha" { t.Fatalf("got=%+v err=%v", got, err) }
	if err := r.RegisterPerson(person("ten_alpha", "per_alpha", "usr_other"), "usr_owner_alpha", instant); !errors.Is(err, ErrDuplicatePerson) { t.Fatalf("want duplicate, got %v", err) }
	if _, err := r.GetPerson("ten_bravo", "per_alpha"); !errors.Is(err, ErrPersonNotFound) { t.Fatalf("cross-tenant lookup must not leak: %v", err) }
}

func TestPersonRejectsAuthenticationTruthAndBlankReferences(t *testing.T) {
	r := NewRegistry()
	if err := r.RegisterPerson(person("ten_alpha", "per_alpha", "token_secret"), "usr_owner", instant); !errors.Is(err, ErrInvalidIdentityRef) { t.Fatalf("want invalid identity ref, got %v", err) }
	if err := r.RegisterPerson(person("ten_alpha", "per_alpha", "usr_identity"), "", instant); !errors.Is(err, ErrBlankField) { t.Fatalf("want blank actor, got %v", err) }
}

func TestEmploymentRequiresSameTenantPersonAndExplicitOwner(t *testing.T) {
	r := NewRegistry()
	if err := r.RegisterPerson(person("ten_alpha", "per_alpha", "usr_identity_alpha"), "usr_owner_alpha", instant); err != nil { t.Fatal(err) }
	e := Employment{ID: "emp_alpha", TenantID: "ten_alpha", PersonID: "per_alpha", CompanyID: "cmp_alpha", OwnerSubjectRef: "usr_owner_alpha", ValidFrom: instant, ValidTo: instant.Add(24*time.Hour)}
	if err := r.RegisterEmployment(e, "usr_owner_alpha", instant); err != nil { t.Fatal(err) }
	foreign := e; foreign.ID, foreign.TenantID = "emp_bravo", "ten_bravo"
	if err := r.RegisterEmployment(foreign, "usr_owner_bravo", instant); !errors.Is(err, ErrPersonNotFound) { t.Fatalf("want non-leaking foreign-person denial, got %v", err) }
	if got := r.History("ten_alpha"); len(got) != 2 || got[0].EntityKind != "PERSON_REGISTERED" || got[1].EntityKind != "EMPLOYMENT_REGISTERED" { t.Fatalf("unexpected history: %+v", got) }
}

func TestEmploymentRejectsImplicitMembershipAndInvalidWindow(t *testing.T) {
	r := NewRegistry()
	e := Employment{ID: "emp_alpha", TenantID: "ten_alpha", PersonID: "per_missing", CompanyID: "cmp_alpha", OwnerSubjectRef: "usr_owner_alpha", ValidFrom: instant, ValidTo: instant.Add(time.Hour)}
	if err := r.RegisterEmployment(e, "usr_owner_alpha", instant); !errors.Is(err, ErrPersonNotFound) { t.Fatalf("want explicit-person requirement, got %v", err) }
	if err := r.RegisterPerson(person("ten_alpha", "per_alpha", "usr_identity_alpha"), "usr_owner_alpha", instant); err != nil { t.Fatal(err) }
	e.PersonID, e.ValidTo = "per_alpha", instant
	if err := r.RegisterEmployment(e, "usr_owner_alpha", instant); !errors.Is(err, ErrInvalidWindow) { t.Fatalf("want invalid window, got %v", err) }
}
