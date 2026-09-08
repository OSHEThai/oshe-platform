package incidentintakefoundation

import (
	"errors"
	"testing"
	"time"
)

var instant = time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)

func TestSubmitIsUnclassifiedAndRequiresHumanTriage(t *testing.T) {
	r := NewRegistry()
	got, err := r.Submit("int_alpha", "ten_alpha", "ref_reporter_alpha", "sub_submitter_alpha", instant)
	if err != nil {
		t.Fatal(err)
	}
	if got.State != Unclassified || !got.HumanTriageRequired {
		t.Fatalf("unsafe intake state: %+v", got)
	}
	// A caller can only mutate its value copy; the registry retains the fixed state.
	got.State, got.HumanTriageRequired = "ROUTED", false
	stored, err := r.Get("ten_alpha", "int_alpha")
	if err != nil || stored.State != Unclassified || !stored.HumanTriageRequired {
		t.Fatalf("stored intake must remain unclassified and human-triage-required: %+v err=%v", stored, err)
	}
}

func TestSubmitDefaultDeniesDuplicateAndCrossTenantLookup(t *testing.T) {
	r := NewRegistry()
	if _, err := r.Submit("int_alpha", "ten_alpha", "ref_reporter_alpha", "sub_submitter_alpha", instant); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Submit("int_alpha", "ten_alpha", "ref_reporter_alpha", "sub_submitter_alpha", instant); !errors.Is(err, ErrDuplicateIntake) {
		t.Fatalf("want duplicate denial, got %v", err)
	}
	if _, err := r.Get("ten_bravo", "int_alpha"); !errors.Is(err, ErrIntakeNotFound) {
		t.Fatalf("cross-tenant lookup must not leak: %v", err)
	}
}

func TestSubmitRejectsBlankAndNonOpaqueSyntheticReferences(t *testing.T) {
	r := NewRegistry()
	if _, err := r.Submit("int_alpha", "ten_alpha", "reporter@example.test", "sub_submitter_alpha", instant); !errors.Is(err, ErrInvalidReference) {
		t.Fatalf("want invalid reporter reference, got %v", err)
	}
	if _, err := r.Submit("int_alpha", "ten_alpha", "ref_reporter_alpha", "", instant); !errors.Is(err, ErrBlankField) {
		t.Fatalf("want blank actor denial, got %v", err)
	}
}

func TestHistoryIsTenantScopedAppendOnlySubmissionAttribution(t *testing.T) {
	r := NewRegistry()
	if _, err := r.Submit("int_alpha", "ten_alpha", "ref_reporter_alpha", "sub_submitter_alpha", instant); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Submit("int_bravo", "ten_bravo", "ref_reporter_bravo", "sub_submitter_bravo", instant); err != nil {
		t.Fatal(err)
	}
	history := r.History("ten_alpha")
	if len(history) != 1 || history[0].IntakeID != "int_alpha" || history[0].ActorRef != "sub_submitter_alpha" {
		t.Fatalf("unexpected tenant-scoped history: %+v", history)
	}
}
