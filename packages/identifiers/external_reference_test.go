package identifiers_test

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/oshethai/oshe-platform/packages/identifiers"
)

func TestExternalReference_RegisterAndLookupPublic(t *testing.T) {
	registry := identifiers.NewExternalReferenceRegistry()
	tenantID := "ten_alpha"
	system := "sap_pm"
	externalID := "EQUIP-98765"
	internalID := identifiers.ID("ins_0123456789abcdef0123456789abcdef")

	// Register
	ref, err := registry.Register(tenantID, system, externalID, internalID)
	if err != nil {
		t.Fatalf("failed to register external reference: %v", err)
	}

	if ref.TenantID != tenantID {
		t.Errorf("expected tenant %q, got %q", tenantID, ref.TenantID)
	}
	if ref.System != system {
		t.Errorf("expected system %q, got %q", system, ref.System)
	}
	if ref.ExternalID != externalID {
		t.Errorf("expected external ID %q, got %q", externalID, ref.ExternalID)
	}
	if !strings.HasPrefix(ref.LookupToken, "ref_") {
		t.Errorf("expected lookup token prefix 'ref_', got %q", ref.LookupToken)
	}

	// Invariant: Lookup token must not equal or leak the internal ID string
	if ref.LookupToken == string(internalID) {
		t.Fatal("security invariant violated: lookup token leaked raw internal ID")
	}

	// Public lookup
	publicRef, err := registry.LookupPublic(tenantID, system, externalID)
	if err != nil {
		t.Fatalf("public lookup failed: %v", err)
	}
	if publicRef.LookupToken != ref.LookupToken {
		t.Errorf("expected matching lookup token %q, got %q", ref.LookupToken, publicRef.LookupToken)
	}

	// Internal resolution within correct tenant context
	resolvedID, err := registry.ResolveInternal(tenantID, ref.LookupToken)
	if err != nil {
		t.Fatalf("internal resolution failed: %v", err)
	}
	if resolvedID != internalID {
		t.Errorf("expected resolved ID %q, got %q", internalID, resolvedID)
	}
}

func TestExternalReference_IdempotentReRegistration(t *testing.T) {
	registry := identifiers.NewExternalReferenceRegistry()
	tenantID := "ten_beta"
	system := "maximo"
	externalID := "WO-4421"
	internalID := identifiers.ID("act_abcdef0123456789abcdef0123456789")

	ref1, err := registry.Register(tenantID, system, externalID, internalID)
	if err != nil {
		t.Fatalf("initial registration failed: %v", err)
	}

	// Re-register identical binding
	ref2, err := registry.Register(tenantID, system, externalID, internalID)
	if err != nil {
		t.Fatalf("idempotent re-registration returned error: %v", err)
	}

	if ref1.LookupToken != ref2.LookupToken {
		t.Errorf("expected identical lookup token on re-registration, got %q vs %q", ref1.LookupToken, ref2.LookupToken)
	}
	if registry.Count() != 1 {
		t.Errorf("expected registry count to remain 1, got %d", registry.Count())
	}
}

func TestExternalReference_Conflict(t *testing.T) {
	registry := identifiers.NewExternalReferenceRegistry()
	tenantID := "ten_gamma"
	system := "jira"
	externalID := "SAFETY-101"

	id1 := identifiers.ID("fnd_11111111111111111111111111111111")
	id2 := identifiers.ID("fnd_22222222222222222222222222222222")

	_, err := registry.Register(tenantID, system, externalID, id1)
	if err != nil {
		t.Fatalf("first registration failed: %v", err)
	}

	// Conflicting registration to different internal ID
	_, err = registry.Register(tenantID, system, externalID, id2)
	if err == nil {
		t.Fatal("expected conflict error, got nil")
	}
	if !errors.Is(err, identifiers.ErrReferenceConflict) {
		t.Errorf("expected ErrReferenceConflict, got: %v", err)
	}
}

func TestExternalReference_TenantIsolationAndMismatchedDenial(t *testing.T) {
	registry := identifiers.NewExternalReferenceRegistry()
	tenantA := "ten_org_a"
	tenantB := "ten_org_b"
	system := "oracle_erp"
	externalID := "INV-5501"
	internalID := identifiers.ID("evd_33333333333333333333333333333333")

	refA, err := registry.Register(tenantA, system, externalID, internalID)
	if err != nil {
		t.Fatalf("registration for tenantA failed: %v", err)
	}

	// TenantB attempting to lookup TenantA's external reference must fail closed
	_, err = registry.LookupPublic(tenantB, system, externalID)
	if err == nil {
		t.Fatal("security violation: tenantB looked up tenantA's external reference")
	}
	if !errors.Is(err, identifiers.ErrReferenceNotFound) {
		t.Errorf("expected ErrReferenceNotFound on foreign tenant lookup, got: %v", err)
	}

	// TenantB attempting to resolve TenantA's lookup token must fail closed
	_, err = registry.ResolveInternal(tenantB, refA.LookupToken)
	if err == nil {
		t.Fatal("security violation: tenantB resolved tenantA's lookup token")
	}
	if !errors.Is(err, identifiers.ErrReferenceNotFound) {
		t.Errorf("expected ErrReferenceNotFound on foreign tenant token resolution, got: %v", err)
	}

	// Unknown reference lookup fails closed
	_, err = registry.LookupPublic(tenantA, system, "UNKNOWN_EXTERNAL_ID")
	if !errors.Is(err, identifiers.ErrReferenceNotFound) {
		t.Errorf("expected ErrReferenceNotFound for unknown external reference, got: %v", err)
	}
}

func TestExternalReference_EmptyParameterValidation(t *testing.T) {
	registry := identifiers.NewExternalReferenceRegistry()
	id := identifiers.ID("ins_44444444444444444444444444444444")

	if _, err := registry.Register("", "sys", "ext1", id); !errors.Is(err, identifiers.ErrEmptyTenantID) {
		t.Errorf("expected ErrEmptyTenantID, got: %v", err)
	}
	if _, err := registry.Register("ten", "", "ext1", id); !errors.Is(err, identifiers.ErrEmptySystem) {
		t.Errorf("expected ErrEmptySystem, got: %v", err)
	}
	if _, err := registry.Register("ten", "sys", "", id); !errors.Is(err, identifiers.ErrEmptyExternalID) {
		t.Errorf("expected ErrEmptyExternalID, got: %v", err)
	}
	if _, err := registry.Register("ten", "sys", "ext1", ""); !errors.Is(err, identifiers.ErrEmptyID) {
		t.Errorf("expected ErrEmptyID, got: %v", err)
	}

	if _, err := registry.LookupPublic("", "sys", "ext1"); !errors.Is(err, identifiers.ErrEmptyTenantID) {
		t.Errorf("expected ErrEmptyTenantID on lookup, got: %v", err)
	}
	if _, err := registry.ResolveInternal("", "ref_123"); !errors.Is(err, identifiers.ErrEmptyTenantID) {
		t.Errorf("expected ErrEmptyTenantID on resolve, got: %v", err)
	}
	if _, err := registry.ResolveInternal("ten", ""); !errors.Is(err, identifiers.ErrEmptyLookupToken) {
		t.Errorf("expected ErrEmptyLookupToken on resolve, got: %v", err)
	}
}

func TestExternalReference_ConcurrentOperations(t *testing.T) {
	registry := identifiers.NewExternalReferenceRegistry()
	const workers = 30
	var wg sync.WaitGroup

	for i := range workers {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			tenantID := fmt.Sprintf("ten_worker_%d", workerID%5)
			system := "work_order_sys"
			externalID := fmt.Sprintf("EXT-%d", workerID)
			internalID := identifiers.ID(fmt.Sprintf("ins_%032x", workerID))

			ref, err := registry.Register(tenantID, system, externalID, internalID)
			if err != nil {
				t.Errorf("worker %d register failed: %v", workerID, err)
				return
			}

			// Public lookup
			lookupRef, err := registry.LookupPublic(tenantID, system, externalID)
			if err != nil {
				t.Errorf("worker %d lookup failed: %v", workerID, err)
				return
			}
			if lookupRef.LookupToken != ref.LookupToken {
				t.Errorf("worker %d token mismatch", workerID)
				return
			}

			// Resolve
			resolved, err := registry.ResolveInternal(tenantID, ref.LookupToken)
			if err != nil {
				t.Errorf("worker %d resolve failed: %v", workerID, err)
				return
			}
			if resolved != internalID {
				t.Errorf("worker %d resolved ID mismatch: expected %s, got %s", workerID, internalID, resolved)
			}
		}(i)
	}

	wg.Wait()

	if registry.Count() != workers {
		t.Errorf("expected registry count %d, got %d", workers, registry.Count())
	}
}
