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
	if err := r.RegisterEmployment(foreign, "usr_owner_bravo", instant); !errors.Is(err, ErrPersonNotFound) {
		t.Fatalf("want non-leaking foreign-person denial ErrPersonNotFound, got %v", err)
	}
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

func TestStructuredKeysPreventAdversarialDelimiterCollision(t *testing.T) {
	r := NewRegistry()
	// Register person in tenant with colon delimiter
	p1 := person("tenant:alpha", "001", "usr_identity_p1")
	if err := r.RegisterPerson(p1, "usr_owner_1", instant); err != nil {
		t.Fatalf("register p1 failed: %v", err)
	}

	// Attempt adversarial collision: tenant="tenant", id="alpha:001"
	// With string concatenation "tenant" + ":" + "alpha:001" == "tenant:alpha:001", this would collide!
	p2 := person("tenant", "alpha:001", "usr_identity_p2")
	if err := r.RegisterPerson(p2, "usr_owner_2", instant); err != nil {
		t.Fatalf("register p2 must succeed as independent structured key, got: %v", err)
	}

	// Verify independent lookups
	got1, err := r.GetPerson("tenant:alpha", "001")
	if err != nil || got1.TrustedIdentityRef != "usr_identity_p1" {
		t.Fatalf("lookup p1 failed or collided: %+v, %v", got1, err)
	}

	got2, err := r.GetPerson("tenant", "alpha:001")
	if err != nil || got2.TrustedIdentityRef != "usr_identity_p2" {
		t.Fatalf("lookup p2 failed or collided: %+v, %v", got2, err)
	}

	// Cross-tenant lookup must fail closed
	if _, err := r.GetPerson("tenant", "001"); !errors.Is(err, ErrPersonNotFound) {
		t.Fatalf("cross-tenant lookup must not find p1 under tenant: %v", err)
	}

	// Cross-tenant employment with delimiter manipulation must be denied
	e := Employment{
		ID:              "emp_001",
		TenantID:        "tenant",
		PersonID:        "001", // exists in tenant:alpha, NOT in tenant
		CompanyID:       "cmp_1",
		OwnerSubjectRef: "usr_owner_2",
		ValidFrom:       instant,
		ValidTo:         instant.Add(24 * time.Hour),
	}
	err = r.RegisterEmployment(e, "usr_owner_2", instant)
	if !errors.Is(err, ErrPersonNotFound) {
		t.Fatalf("cross-tenant employment via delimiter manipulation must be denied with ErrPersonNotFound, got: %v", err)
	}
}

func TestEmploymentNonLeakingDenialWithoutForeignEnumeration(t *testing.T) {
	r := NewRegistry()
	if err := r.RegisterPerson(person("ten_alpha", "per_alpha", "usr_identity_alpha"), "usr_owner_alpha", instant); err != nil {
		t.Fatal(err)
	}

	// 1. Foreign person reference: person exists in ten_alpha, employment attempted in ten_bravo
	foreignEmp := Employment{
		ID:              "emp_foreign",
		TenantID:        "ten_bravo",
		PersonID:        "per_alpha",
		CompanyID:       "cmp_bravo",
		OwnerSubjectRef: "usr_owner_bravo",
		ValidFrom:       instant,
		ValidTo:         instant.Add(24 * time.Hour),
	}
	errForeign := r.RegisterEmployment(foreignEmp, "usr_owner_bravo", instant)

	// 2. Completely nonexistent person reference: person does not exist in any tenant
	missingEmp := Employment{
		ID:              "emp_missing",
		TenantID:        "ten_bravo",
		PersonID:        "per_nonexistent",
		CompanyID:       "cmp_bravo",
		OwnerSubjectRef: "usr_owner_bravo",
		ValidFrom:       instant,
		ValidTo:         instant.Add(24 * time.Hour),
	}
	errMissing := r.RegisterEmployment(missingEmp, "usr_owner_bravo", instant)

	// Both must return identical non-leaking ErrPersonNotFound (zero cross-tenant existence oracle)
	if !errors.Is(errForeign, ErrPersonNotFound) {
		t.Fatalf("expected ErrPersonNotFound for foreign person, got: %v", errForeign)
	}
	if !errors.Is(errMissing, ErrPersonNotFound) {
		t.Fatalf("expected ErrPersonNotFound for missing person, got: %v", errMissing)
	}
	if errForeign != errMissing {
		t.Fatalf("expected identical non-leaking error for foreign and missing person, got foreign=%v, missing=%v", errForeign, errMissing)
	}
}
