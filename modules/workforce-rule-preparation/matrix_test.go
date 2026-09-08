package workforcerulepreparation

import (
	"errors"
	"testing"
	"time"
)

var instant = time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)

func completeMatrix(tenant, id string) Matrix {
	return Matrix{ID: id, TenantID: tenant, VersionRef: "ver_draft_one", Entries: []Entry{
		{RequirementRef: "req_unknown", Disposition: Unknown},
		{RequirementRef: "req_incomplete", Disposition: Incomplete},
		{RequirementRef: "req_expired", Disposition: Expired},
		{RequirementRef: "req_disputed", Disposition: Disputed},
		{RequirementRef: "req_exception", Disposition: Exception},
	}}
}

func TestMatrixReturnsOnlyNonBindingHumanDecisionRequiredAssessments(t *testing.T) {
	r := NewRegistry()
	if err := r.Register(completeMatrix("ten_alpha", "mat_alpha"), "sub_owner_alpha", instant); err != nil {
		t.Fatal(err)
	}
	got, err := r.GetAssessment("ten_alpha", "mat_alpha", "req_expired")
	if err != nil {
		t.Fatal(err)
	}
	if got.Disposition != Expired || !got.NonBinding || !got.HumanDecisionRequired {
		t.Fatalf("must not assert a binding workforce outcome: %+v", got)
	}
}

func TestMatrixRequiresEveryNonCurrentDispositionAndOpaqueReferences(t *testing.T) {
	r := NewRegistry()
	incomplete := completeMatrix("ten_alpha", "mat_alpha")
	incomplete.Entries = incomplete.Entries[:4]
	if err := r.Register(incomplete, "sub_owner_alpha", instant); !errors.Is(err, ErrIncompleteDispositionCoverage) {
		t.Fatalf("want incomplete coverage, got %v", err)
	}
	invalid := completeMatrix("ten_alpha", "mat_alpha")
	invalid.Entries[0].RequirementRef = "employee@example.test"
	if err := r.Register(invalid, "sub_owner_alpha", instant); !errors.Is(err, ErrInvalidReference) {
		t.Fatalf("want opaque-reference denial, got %v", err)
	}
}

func TestMatrixDefaultDeniesTenantCollisionAndDuplicate(t *testing.T) {
	r := NewRegistry()
	if err := r.Register(completeMatrix("ten_alpha", "mat_bravo:mat_charlie"), "sub_owner_alpha", instant); err != nil {
		t.Fatal(err)
	}
	if _, err := r.GetAssessment("ten_alpha:mat_bravo", "mat_charlie", "req_unknown"); !errors.Is(err, ErrAssessmentNotFound) {
		t.Fatalf("cross-tenant lookup must default-deny: %v", err)
	}
	if err := r.Register(completeMatrix("ten_alpha:mat_bravo", "mat_charlie"), "sub_owner_bravo", instant); err != nil {
		t.Fatalf("former concatenated-key collision must remain isolated: %v", err)
	}
	sameVersionDifferentID := completeMatrix("ten_alpha", "mat_delta")
	if err := r.Register(sameVersionDifferentID, "sub_owner_alpha", instant); !errors.Is(err, ErrDuplicateVersion) {
		t.Fatalf("same tenant version must be rejected even with a new matrix ID: %v", err)
	}
	crossTenantSameVersion := completeMatrix("ten_bravo", "mat_delta")
	if err := r.Register(crossTenantSameVersion, "sub_owner_bravo", instant); err != nil {
		t.Fatalf("version uniqueness must remain tenant-scoped: %v", err)
	}
	if err := r.Register(completeMatrix("ten_alpha", "mat_bravo:mat_charlie"), "sub_owner_alpha", instant); !errors.Is(err, ErrDuplicateMatrix) {
		t.Fatalf("want duplicate denial, got %v", err)
	}
}

func TestMatrixCopiesImmutableContentAndKeepsAppendOnlyHistory(t *testing.T) {
	r := NewRegistry()
	matrix := completeMatrix("ten_alpha", "mat_alpha")
	if err := r.Register(matrix, "sub_owner_alpha", instant); err != nil {
		t.Fatal(err)
	}
	matrix.Entries[0].Disposition = Exception
	got, err := r.GetAssessment("ten_alpha", "mat_alpha", "req_unknown")
	if err != nil || got.Disposition != Unknown {
		t.Fatalf("registered matrix must be immutable from caller mutation: %+v err=%v", got, err)
	}
	history := r.History("ten_alpha")
	if len(history) != 1 || history[0].MatrixID != "mat_alpha" || history[0].ActorRef != "sub_owner_alpha" {
		t.Fatalf("unexpected append-only history: %+v", history)
	}
}
