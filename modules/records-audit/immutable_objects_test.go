package recordsaudit_test

import (
	"errors"
	"testing"
	"time"

	recordsaudit "github.com/oshethai/oshe-platform/modules/records-audit"
)

const (
	validDigestOriginal = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	validDigestDerived  = "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
	validDigestDerived2 = "cb8379ac2098aa165029e3938a51da0bcecfc008fd6795f401178647f96c5b34"
	tamperedDigest      = "0000000000000000000000000000000000000000000000000000000000000000"
)

func TestOriginal_CreationAndAcceptance(t *testing.T) {
	now := time.Now().UTC()
	orig, err := recordsaudit.NewOriginalRecord("orig_001", "ten_alpha", "image/png", 1024, validDigestOriginal, now)
	if err != nil {
		t.Fatalf("unexpected construction error: %v", err)
	}

	if orig.ObjectID() != "orig_001" {
		t.Errorf("expected objectID orig_001, got %s", orig.ObjectID())
	}
	if orig.TenantID() != "ten_alpha" {
		t.Errorf("expected tenantID ten_alpha, got %s", orig.TenantID())
	}
	if orig.State() != recordsaudit.StateDraft {
		t.Errorf("expected state DRAFT, got %s", orig.State())
	}
	if orig.Kind() != recordsaudit.KindOriginal {
		t.Errorf("expected kind ORIGINAL, got %s", orig.Kind())
	}

	reg := recordsaudit.NewIntegrityRegistry()
	if err := reg.RegisterOriginal("ten_alpha", orig); err != nil {
		t.Fatalf("unexpected registration error: %v", err)
	}

	if err := reg.AcceptOriginal("ten_alpha", "orig_001"); err != nil {
		t.Fatalf("unexpected acceptance error: %v", err)
	}

	// Idempotent acceptance
	if err := reg.AcceptOriginal("ten_alpha", "orig_001"); err != nil {
		t.Fatalf("unexpected error on idempotent acceptance: %v", err)
	}
}

func TestOriginal_ImmutableOverwriteDenial(t *testing.T) {
	reg := recordsaudit.NewIntegrityRegistry()
	orig, err := recordsaudit.NewOriginalRecord("orig_002", "ten_alpha", "image/png", 2048, validDigestOriginal, time.Time{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := reg.RegisterOriginal("ten_alpha", orig); err != nil {
		t.Fatalf("unexpected registration error: %v", err)
	}
	if err := reg.AcceptOriginal("ten_alpha", "orig_002"); err != nil {
		t.Fatalf("unexpected accept error: %v", err)
	}

	// Attempt to overwrite accepted original with a modified record reusing the same objectID
	tamperedOrig, err := recordsaudit.NewOriginalRecord("orig_002", "ten_alpha", "image/png", 4096, validDigestDerived, time.Time{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = reg.RegisterOriginal("ten_alpha", tamperedOrig)
	if !errors.Is(err, recordsaudit.ErrOriginalOverwriteDenied) {
		t.Fatalf("expected ErrOriginalOverwriteDenied, got: %v", err)
	}
}

func TestOriginal_InputValidation(t *testing.T) {
	_, err := recordsaudit.NewOriginalRecord("", "ten_alpha", "image/png", 100, validDigestOriginal, time.Time{})
	if !errors.Is(err, recordsaudit.ErrBlankID) {
		t.Errorf("expected ErrBlankID, got %v", err)
	}

	_, err = recordsaudit.NewOriginalRecord("orig_valid", "", "image/png", 100, validDigestOriginal, time.Time{})
	if !errors.Is(err, recordsaudit.ErrBlankTenantID) {
		t.Errorf("expected ErrBlankTenantID, got %v", err)
	}

	_, err = recordsaudit.NewOriginalRecord("orig_valid", "ten_alpha", "", 100, validDigestOriginal, time.Time{})
	if !errors.Is(err, recordsaudit.ErrBlankMediaType) {
		t.Errorf("expected ErrBlankMediaType, got %v", err)
	}

	_, err = recordsaudit.NewOriginalRecord("orig_valid", "ten_alpha", "image/png", 0, validDigestOriginal, time.Time{})
	if !errors.Is(err, recordsaudit.ErrInvalidSizeBytes) {
		t.Errorf("expected ErrInvalidSizeBytes for zero size, got %v", err)
	}

	_, err = recordsaudit.NewOriginalRecord("orig_valid", "ten_alpha", "image/png", -10, validDigestOriginal, time.Time{})
	if !errors.Is(err, recordsaudit.ErrInvalidSizeBytes) {
		t.Errorf("expected ErrInvalidSizeBytes for negative size, got %v", err)
	}

	// Digest errors: uppercase, too short, too long, non-hex
	badDigests := []string{
		"E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855", // uppercase
		"shortdigest",
		"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855aa", // 66 chars
		"g3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",   // 'g' non-hex
	}
	for _, bd := range badDigests {
		_, err = recordsaudit.NewOriginalRecord("orig_valid", "ten_alpha", "image/png", 100, bd, time.Time{})
		if !errors.Is(err, recordsaudit.ErrInvalidDigest) {
			t.Errorf("for bad digest %q, expected ErrInvalidDigest, got %v", bd, err)
		}
	}
}

func TestDuplicateMediaIdentityHandling(t *testing.T) {
	reg := recordsaudit.NewIntegrityRegistry()
	orig, _ := recordsaudit.NewOriginalRecord("dup_001", "ten_alpha", "image/png", 100, validDigestOriginal, time.Time{})
	if err := reg.RegisterOriginal("ten_alpha", orig); err != nil {
		t.Fatalf("first register failed: %v", err)
	}

	// Re-registering draft with same ID
	orig2, _ := recordsaudit.NewOriginalRecord("dup_001", "ten_alpha", "image/png", 200, validDigestOriginal, time.Time{})
	if err := reg.RegisterOriginal("ten_alpha", orig2); !errors.Is(err, recordsaudit.ErrDuplicateObjectID) {
		t.Fatalf("expected ErrDuplicateObjectID, got %v", err)
	}
}

func TestDerived_LinkageAndIntegrity(t *testing.T) {
	reg := recordsaudit.NewIntegrityRegistry()
	orig, _ := recordsaudit.NewOriginalRecord("orig_parent", "ten_alpha", "image/png", 1024, validDigestOriginal, time.Time{})
	_ = reg.RegisterOriginal("ten_alpha", orig)
	_ = reg.AcceptOriginal("ten_alpha", "orig_parent")

	derivedKinds := []recordsaudit.DerivedKind{
		recordsaudit.DerivedAnnotation,
		recordsaudit.DerivedRedaction,
		recordsaudit.DerivedThumbnail,
		recordsaudit.DerivedTransformation,
	}

	for i, kind := range derivedKinds {
		derivedID := "derived_" + string(kind)
		der, err := recordsaudit.NewDerivedRecord(derivedID, "ten_alpha", "orig_parent", kind, "application/json", int64(100+i), validDigestDerived, time.Time{})
		if err != nil {
			t.Fatalf("failed to create derived record for %s: %v", kind, err)
		}

		if err := reg.RegisterDerived("ten_alpha", der); err != nil {
			t.Fatalf("failed to register derived record for %s: %v", kind, err)
		}

		if err := reg.AcceptDerived("ten_alpha", derivedID); err != nil {
			t.Fatalf("failed to accept derived record for %s: %v", kind, err)
		}

		linkage, err := reg.VerifyIntegrityLinkage(derivedID, "ten_alpha", validDigestDerived)
		if err != nil {
			t.Fatalf("failed to verify integrity linkage for %s: %v", kind, err)
		}

		if linkage.Derived.ObjectID() != derivedID {
			t.Errorf("expected derived ID %s, got %s", derivedID, linkage.Derived.ObjectID())
		}
		if linkage.Original.ObjectID() != "orig_parent" {
			t.Errorf("expected original parent orig_parent, got %s", linkage.Original.ObjectID())
		}
		if linkage.Derived.DerivedKind() != kind {
			t.Errorf("expected kind %s, got %s", kind, linkage.Derived.DerivedKind())
		}
	}
}

func TestDerived_UnknownAndUnacceptedParent(t *testing.T) {
	reg := recordsaudit.NewIntegrityRegistry()

	// 1. Unknown parent
	der, _ := recordsaudit.NewDerivedRecord("der_no_parent", "ten_alpha", "nonexistent_parent", recordsaudit.DerivedThumbnail, "image/png", 100, validDigestDerived, time.Time{})
	if err := reg.RegisterDerived("ten_alpha", der); !errors.Is(err, recordsaudit.ErrUnknownParent) {
		t.Fatalf("expected ErrUnknownParent, got %v", err)
	}

	// 2. Parent exists but is DRAFT (not ACCEPTED)
	origDraft, _ := recordsaudit.NewOriginalRecord("orig_draft", "ten_alpha", "image/png", 500, validDigestOriginal, time.Time{})
	_ = reg.RegisterOriginal("ten_alpha", origDraft)

	der2, _ := recordsaudit.NewDerivedRecord("der_draft_parent", "ten_alpha", "orig_draft", recordsaudit.DerivedThumbnail, "image/png", 100, validDigestDerived, time.Time{})
	if err := reg.RegisterDerived("ten_alpha", der2); !errors.Is(err, recordsaudit.ErrParentNotAccepted) {
		t.Fatalf("expected ErrParentNotAccepted, got %v", err)
	}
}

func TestDerived_CrossTenantDenial(t *testing.T) {
	reg := recordsaudit.NewIntegrityRegistry()
	orig, _ := recordsaudit.NewOriginalRecord("orig_tenant_a", "tenant_a", "image/png", 1024, validDigestOriginal, time.Time{})
	_ = reg.RegisterOriginal("tenant_a", orig)
	_ = reg.AcceptOriginal("tenant_a", "orig_tenant_a")

	// Attempt to register derived object in tenant_b linked to unshared tenant_a original -> uniform ErrUnknownParent
	derCross, _ := recordsaudit.NewDerivedRecord("der_tenant_b", "tenant_b", "orig_tenant_a", recordsaudit.DerivedRedaction, "image/png", 100, validDigestDerived, time.Time{})
	err := reg.RegisterDerived("tenant_b", derCross)
	if !errors.Is(err, recordsaudit.ErrUnknownParent) {
		t.Fatalf("expected ErrUnknownParent on RegisterDerived without foreign enumeration, got %v", err)
	}

	// Mismatched trusted tenant on derived registration fails closed with ErrCrossTenantLinkage
	if err := reg.RegisterDerived("tenant_a", derCross); !errors.Is(err, recordsaudit.ErrCrossTenantLinkage) {
		t.Fatalf("expected ErrCrossTenantLinkage on mismatched trusted tenant scope, got %v", err)
	}

	// Valid derived registered under tenant_a
	derValid, _ := recordsaudit.NewDerivedRecord("der_tenant_a", "tenant_a", "orig_tenant_a", recordsaudit.DerivedRedaction, "image/png", 100, validDigestDerived, time.Time{})
	_ = reg.RegisterDerived("tenant_a", derValid)

	// Caller attempts to verify linkage under tenant_b scope -> uniform ErrUnknownParent without leakage
	_, err = reg.VerifyIntegrityLinkage("der_tenant_a", "tenant_b", validDigestDerived)
	if !errors.Is(err, recordsaudit.ErrUnknownParent) {
		t.Fatalf("expected ErrUnknownParent on VerifyIntegrityLinkage with unshared foreign record, got %v", err)
	}
}

func TestDerived_OriginalAsDerivedConfusionDenial(t *testing.T) {
	reg := recordsaudit.NewIntegrityRegistry()
	orig, _ := recordsaudit.NewOriginalRecord("entity_001", "ten_alpha", "image/png", 1024, validDigestOriginal, time.Time{})
	_ = reg.RegisterOriginal("ten_alpha", orig)

	// Attempting to register entity_001 as derived
	der, _ := recordsaudit.NewDerivedRecord("entity_001", "ten_alpha", "orig_other", recordsaudit.DerivedAnnotation, "application/json", 100, validDigestDerived, time.Time{})
	err := reg.RegisterDerived("ten_alpha", der)
	if !errors.Is(err, recordsaudit.ErrOriginalAsDerivedDenied) {
		t.Fatalf("expected ErrOriginalAsDerivedDenied, got %v", err)
	}
}

func TestVerifyIntegrityLinkage_TamperDigestFailure(t *testing.T) {
	reg := recordsaudit.NewIntegrityRegistry()
	orig, _ := recordsaudit.NewOriginalRecord("orig_verify", "ten_alpha", "image/png", 1024, validDigestOriginal, time.Time{})
	_ = reg.RegisterOriginal("ten_alpha", orig)
	_ = reg.AcceptOriginal("ten_alpha", "orig_verify")

	der, _ := recordsaudit.NewDerivedRecord("der_verify", "ten_alpha", "orig_verify", recordsaudit.DerivedThumbnail, "image/png", 200, validDigestDerived, time.Time{})
	_ = reg.RegisterDerived("ten_alpha", der)

	// Verify with tampered digest
	_, err := reg.VerifyIntegrityLinkage("der_verify", "ten_alpha", tamperedDigest)
	if !errors.Is(err, recordsaudit.ErrDigestMismatch) {
		t.Fatalf("expected ErrDigestMismatch, got %v", err)
	}

	// Verify with invalid digest shape
	_, err = reg.VerifyIntegrityLinkage("der_verify", "ten_alpha", "invalid-digest")
	if !errors.Is(err, recordsaudit.ErrInvalidDigest) {
		t.Fatalf("expected ErrInvalidDigest, got %v", err)
	}

	// Verify with blank caller tenant
	_, err = reg.VerifyIntegrityLinkage("der_verify", "", validDigestDerived)
	if !errors.Is(err, recordsaudit.ErrBlankTenantID) {
		t.Fatalf("expected ErrBlankTenantID, got %v", err)
	}
}

func TestLifecycle_InvalidTransitions(t *testing.T) {
	reg := recordsaudit.NewIntegrityRegistry()
	orig, _ := recordsaudit.NewOriginalRecord("orig_life", "ten_alpha", "image/png", 1024, validDigestOriginal, time.Time{})
	_ = reg.RegisterOriginal("ten_alpha", orig)

	// Direct archive from DRAFT must fail
	err := reg.ArchiveOriginal("ten_alpha", "orig_life")
	if !errors.Is(err, recordsaudit.ErrInvalidLifecycleTransition) {
		t.Fatalf("expected ErrInvalidLifecycleTransition from DRAFT->ARCHIVED, got %v", err)
	}

	_ = reg.AcceptOriginal("ten_alpha", "orig_life")
	if err := reg.ArchiveOriginal("ten_alpha", "orig_life"); err != nil {
		t.Fatalf("expected successful archive from ACCEPTED, got %v", err)
	}

	// Attempt to re-accept archived record
	err = reg.AcceptOriginal("ten_alpha", "orig_life")
	if !errors.Is(err, recordsaudit.ErrRecordArchived) {
		t.Fatalf("expected ErrRecordArchived, got %v", err)
	}
}

func TestIntegrityRegistry_TenantIsolationAndCollisionSafety(t *testing.T) {
	reg := recordsaudit.NewIntegrityRegistry()
	sharedID := "evidence_media_shared_001"

	// Tenant Alpha registers original
	origA, err := recordsaudit.NewOriginalRecord(sharedID, "ten_alpha", "image/png", 1024, validDigestOriginal, time.Time{})
	if err != nil {
		t.Fatalf("setup origA failed: %v", err)
	}
	if err := reg.RegisterOriginal("ten_alpha", origA); err != nil {
		t.Fatalf("register origA failed: %v", err)
	}

	// Tenant Bravo registers same object ID -> MUST SUCCEED (collision safe)
	origB, err := recordsaudit.NewOriginalRecord(sharedID, "ten_bravo", "image/png", 2048, validDigestDerived, time.Time{})
	if err != nil {
		t.Fatalf("setup origB failed: %v", err)
	}
	if err := reg.RegisterOriginal("ten_bravo", origB); err != nil {
		t.Fatalf("register origB with same ID in ten_bravo must succeed, got: %v", err)
	}

	// Tenant Bravo accepts its own original; Tenant Alpha's original remains in DRAFT
	if err := reg.AcceptOriginal("ten_bravo", sharedID); err != nil {
		t.Fatalf("accept origB failed: %v", err)
	}

	// Cross-tenant accept fails closed with ErrUnknownParent
	if err := reg.AcceptOriginal("ten_charlie", sharedID); !errors.Is(err, recordsaudit.ErrUnknownParent) {
		t.Fatalf("expected ErrUnknownParent for cross-tenant accept, got: %v", err)
	}

	// Cross-tenant archive fails closed with ErrUnknownParent
	if err := reg.ArchiveOriginal("ten_charlie", sharedID); !errors.Is(err, recordsaudit.ErrUnknownParent) {
		t.Fatalf("expected ErrUnknownParent for cross-tenant archive, got: %v", err)
	}
}
