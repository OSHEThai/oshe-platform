package flags

import (
	"errors"
	"testing"
)

func TestRegistry_DefaultOffAndScope(t *testing.T) {
	registry, err := NewRegistry([]Definition{{Key: "inspection.preview", DefaultEnabled: false, Stage: Synthetic, AllowedTenants: []string{"ten_demo"}, AllowedRoles: []string{"inspector"}}})
	if err != nil {
		t.Fatal(err)
	}
	subject := Subject{Tenant: "ten_demo", Roles: []string{"inspector"}}
	if registry.Evaluate("inspection.preview", Synthetic, subject) {
		t.Fatal("default-off flag must not enable")
	}
	if registry.Evaluate("missing.flag", Synthetic, subject) {
		t.Fatal("missing flag must fail closed")
	}
	if registry.Evaluate("inspection.preview", Environment("PRODUCTION"), subject) {
		t.Fatal("non-synthetic evaluation must fail closed")
	}
}

func TestRegistry_RejectsInvalidAndDuplicateDefinitions(t *testing.T) {
	if _, err := NewRegistry([]Definition{{Key: "BAD", Stage: Synthetic}}); !errors.Is(err, ErrInvalidKey) {
		t.Fatalf("expected invalid key, got %v", err)
	}
	definition := Definition{Key: "safe.rollout", Stage: Synthetic}
	if _, err := NewRegistry([]Definition{definition, definition}); !errors.Is(err, ErrDuplicateKey) {
		t.Fatalf("expected duplicate key, got %v", err)
	}
	if _, err := NewRegistry([]Definition{{Key: "unsafe.rollout", DefaultEnabled: true, Stage: Synthetic}}); !errors.Is(err, ErrInvalidDefinition) {
		t.Fatalf("expected invalid default, got %v", err)
	}
}

func TestLoadJSONAndAuthorizationBoundary(t *testing.T) {
	registry, err := LoadJSON([]byte(`[{"key":"review.experiment","default_enabled":false,"stage":"SYNTHETIC_ONLY","allowed_tenants":["ten_demo"],"allowed_roles":["reviewer"]}]`))
	if err != nil {
		t.Fatal(err)
	}
	if registry.Evaluate("review.experiment", Synthetic, Subject{Tenant: "ten_other", Roles: []string{"reviewer"}}) {
		t.Fatal("cross-tenant exposure must fail closed")
	}
	if registry.Evaluate("review.experiment", Synthetic, Subject{Tenant: "ten_demo", Roles: []string{"viewer"}}) {
		t.Fatal("unauthorized role exposure must fail closed")
	}
	if _, err := LoadJSON([]byte(`not-json`)); err == nil {
		t.Fatal("malformed registry must fail closed")
	}
}
