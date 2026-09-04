package checklist_test

import (
	"errors"
	"testing"

	checklist "github.com/oshethai/oshe-platform/modules/configuration-checklist"
)

func sampleQuestions() []checklist.Question {
	return []checklist.Question{
		{ID: "q1", Text: "Are emergency exits unobstructed?", Type: "BOOLEAN", Required: true},
		{ID: "q2", Text: "Record observed ambient temperature", Type: "DECIMAL", Required: false},
	}
}

func TestTemplateLifecycle_ValidPublicationFlow(t *testing.T) {
	tv, err := checklist.NewTemplateVersion("tmpl_fire_safety", "ver_001", "Fire Safety Inspection", sampleQuestions())
	if err != nil {
		t.Fatalf("unexpected construction error: %v", err)
	}

	if tv.State != checklist.StateDraft {
		t.Errorf("expected state Draft, got %s", tv.State)
	}

	// Draft -> InReview
	if err := tv.SubmitForReview(); err != nil {
		t.Fatalf("submit for review failed: %v", err)
	}
	if tv.State != checklist.StateInReview {
		t.Errorf("expected state InReview, got %s", tv.State)
	}

	// Record Review
	if err := tv.RecordReview("safety_officer_alice", "All standard questions present"); err != nil {
		t.Fatalf("record review failed: %v", err)
	}
	if tv.Review == nil || tv.Review.ReviewedBy != "safety_officer_alice" {
		t.Errorf("expected review recorded for alice")
	}

	// InReview -> Approved
	if err := tv.Approve("compliance_lead_bob", "Formal approval per protocol"); err != nil {
		t.Fatalf("approve failed: %v", err)
	}
	if tv.State != checklist.StateApproved {
		t.Errorf("expected state Approved, got %s", tv.State)
	}

	// Approved -> Published
	if err := tv.Publish(); err != nil {
		t.Fatalf("publish failed: %v", err)
	}
	if tv.State != checklist.StatePublished {
		t.Errorf("expected state Published, got %s", tv.State)
	}

	// Verify immutable snapshot
	snap := tv.PublishedSnapshot
	if snap == nil {
		t.Fatal("expected non-nil PublishedSnapshot upon publication")
	}
	if snap.TemplateID != tv.TemplateID || snap.VersionID != tv.VersionID {
		t.Errorf("snapshot metadata mismatch")
	}
	if len(snap.Questions) != len(sampleQuestions()) {
		t.Errorf("snapshot questions length mismatch")
	}
	if snap.Approval.ApprovedBy != "compliance_lead_bob" {
		t.Errorf("snapshot approval mismatch")
	}

	// Published -> Retired
	if err := tv.Retire(); err != nil {
		t.Fatalf("retire failed: %v", err)
	}
	if tv.State != checklist.StateRetired {
		t.Errorf("expected state Retired, got %s", tv.State)
	}
}

func TestTemplateLifecycle_MissingReviewRejection(t *testing.T) {
	tv, _ := checklist.NewTemplateVersion("tmpl_01", "v1", "Test", sampleQuestions())
	_ = tv.SubmitForReview()

	// Approve without recording review must fail
	err := tv.Approve("approver_bob", "auth")
	if err == nil {
		t.Fatal("expected error approving without review marker, got nil")
	}
	if !errors.Is(err, checklist.ErrMissingReview) {
		t.Errorf("expected ErrMissingReview, got: %v", err)
	}
}

func TestTemplateLifecycle_MissingApprovalRejection(t *testing.T) {
	tv, _ := checklist.NewTemplateVersion("tmpl_01", "v1", "Test", sampleQuestions())
	_ = tv.SubmitForReview()
	_ = tv.RecordReview("reviewer_alice", "ok")

	// Attempting to publish while InReview (skipping Approved) must fail
	err := tv.Publish()
	if err == nil {
		t.Fatal("expected error publishing unapproved template, got nil")
	}
	if !errors.Is(err, checklist.ErrInvalidLifecycleTransition) {
		t.Errorf("expected ErrInvalidLifecycleTransition, got: %v", err)
	}
}

func TestTemplateLifecycle_InvalidTransitions(t *testing.T) {
	tv, _ := checklist.NewTemplateVersion("tmpl_01", "v1", "Test", sampleQuestions())

	// Direct Draft -> Publish invalid
	if err := tv.Publish(); !errors.Is(err, checklist.ErrInvalidLifecycleTransition) {
		t.Errorf("expected ErrInvalidLifecycleTransition for Draft->Publish, got: %v", err)
	}

	// Direct Draft -> Retire invalid
	if err := tv.Retire(); !errors.Is(err, checklist.ErrInvalidLifecycleTransition) {
		t.Errorf("expected ErrInvalidLifecycleTransition for Draft->Retire, got: %v", err)
	}

	// Progress to Retired
	_ = tv.SubmitForReview()
	_ = tv.RecordReview("alice", "ok")
	_ = tv.Approve("bob", "auth")
	_ = tv.Publish()
	_ = tv.Retire()

	// Retired cannot transition anywhere
	if err := tv.Publish(); !errors.Is(err, checklist.ErrInvalidLifecycleTransition) {
		t.Errorf("expected ErrInvalidLifecycleTransition for Retired->Publish, got: %v", err)
	}
	if err := tv.SubmitForReview(); !errors.Is(err, checklist.ErrInvalidLifecycleTransition) {
		t.Errorf("expected ErrInvalidLifecycleTransition for Retired->SubmitForReview, got: %v", err)
	}
}

func TestTemplateLifecycle_PublishedMutationRejection(t *testing.T) {
	tv, _ := checklist.NewTemplateVersion("tmpl_01", "v1", "Initial Title", sampleQuestions())
	_ = tv.SubmitForReview()
	_ = tv.RecordReview("alice", "ok")
	_ = tv.Approve("bob", "auth")
	_ = tv.Publish()

	// Mutating published template content must fail closed
	err := tv.UpdateContent("Mutated Title", []checklist.Question{
		{ID: "q99", Text: "New unapproved question", Type: "TEXT", Required: false},
	})
	if err == nil {
		t.Fatal("expected ErrImmutableContent mutating published template, got nil")
	}
	if !errors.Is(err, checklist.ErrImmutableContent) {
		t.Errorf("expected ErrImmutableContent, got: %v", err)
	}

	// Retire and ensure mutation still prohibited
	_ = tv.Retire()
	err = tv.UpdateContent("Mutated Retired", sampleQuestions())
	if !errors.Is(err, checklist.ErrImmutableContent) {
		t.Errorf("expected ErrImmutableContent on retired template, got: %v", err)
	}
}

func TestTemplateLifecycle_CopyProvenanceAndSnapshotPreservation(t *testing.T) {
	v1, err := checklist.NewTemplateVersion("tmpl_original", "ver_1", "Original Template", sampleQuestions())
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	_ = v1.SubmitForReview()
	_ = v1.RecordReview("alice", "looks good")
	_ = v1.Approve("bob", "executive")
	_ = v1.Publish()

	// Derive copy v2
	v2, err := v1.Copy("tmpl_original", "ver_2", "Updated Template Version 2")
	if err != nil {
		t.Fatalf("copy failed: %v", err)
	}

	// Verify v2 state is Draft with valid provenance
	if v2.State != checklist.StateDraft {
		t.Errorf("expected v2 state to be Draft, got %s", v2.State)
	}
	if v2.Provenance == nil {
		t.Fatal("expected non-nil provenance on copied template")
	}
	if v2.Provenance.SourceTemplateID != "tmpl_original" || v2.Provenance.SourceVersionID != "ver_1" {
		t.Errorf("provenance mismatch: %+v", v2.Provenance)
	}

	// Mutate v2 in Draft
	newQuestions := []checklist.Question{
		{ID: "q_extra", Text: "Additional safety check", Type: "BOOLEAN", Required: true},
	}
	if err := v2.UpdateContent("Completely Changed V2", newQuestions); err != nil {
		t.Fatalf("v2 draft update failed: %v", err)
	}

	// Assert v1 questions and PublishedSnapshot remain untouched
	if len(v1.Questions) != 2 {
		t.Errorf("v1 questions were mutated by v2 changes: expected 2, got %d", len(v1.Questions))
	}
	if v1.PublishedSnapshot == nil || len(v1.PublishedSnapshot.Questions) != 2 {
		t.Errorf("v1 published snapshot was mutated by v2 changes")
	}
	if v1.Title != "Original Template" {
		t.Errorf("v1 title was mutated: got %q", v1.Title)
	}
}

func TestTemplateLifecycle_RejectToDraft(t *testing.T) {
	tv, _ := checklist.NewTemplateVersion("tmpl_01", "v1", "Title", sampleQuestions())
	_ = tv.SubmitForReview()

	if err := tv.RejectToDraft(); err != nil {
		t.Fatalf("reject to draft failed: %v", err)
	}
	if tv.State != checklist.StateDraft {
		t.Errorf("expected Draft state after rejection, got %s", tv.State)
	}
}

func TestTemplateLifecycle_InstantiateAndStaleVersionRetention(t *testing.T) {
	v1, err := checklist.NewTemplateVersion("tmpl_fire", "ver_1", "Fire Safety V1", sampleQuestions())
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	_ = v1.SubmitForReview()
	_ = v1.RecordReview("alice", "ok")
	_ = v1.Approve("bob", "auth")
	if err := v1.Publish(); err != nil {
		t.Fatalf("publish failed: %v", err)
	}

	// Instantiate checklist against published v1
	inst1, err := v1.Instantiate("inst_001", "ten_alpha")
	if err != nil {
		t.Fatalf("instantiate failed: %v", err)
	}
	if inst1.InstanceID != "inst_001" || inst1.TenantID != "ten_alpha" {
		t.Errorf("instance metadata mismatch: %+v", inst1)
	}
	if inst1.TemplateRef.TemplateID != "tmpl_fire" || inst1.TemplateRef.VersionID != "ver_1" {
		t.Errorf("version reference mismatch: %+v", inst1.TemplateRef)
	}
	if inst1.IsStale("ver_1") {
		t.Errorf("expected inst1 to not be stale against ver_1")
	}

	// Derive v2, update content, and publish v2
	v2, _ := v1.Copy("tmpl_fire", "ver_2", "Fire Safety V2")
	_ = v2.UpdateContent("Fire Safety V2", []checklist.Question{
		{ID: "q_new", Text: "New requirement", Type: "BOOLEAN", Required: true},
	})
	_ = v2.SubmitForReview()
	_ = v2.RecordReview("alice", "ok v2")
	_ = v2.Approve("bob", "auth v2")
	_ = v2.Publish()

	// Retire v1
	if err := v1.Retire(); err != nil {
		t.Fatalf("retire v1 failed: %v", err)
	}

	// Assert prior instance retains exact version used (v1) and its snapshot
	if inst1.TemplateRef.VersionID != "ver_1" {
		t.Errorf("expected inst1 to retain ver_1 reference, got %s", inst1.TemplateRef.VersionID)
	}
	if len(inst1.Snapshot.Questions) != 2 || inst1.Snapshot.Title != "Fire Safety V1" {
		t.Errorf("inst1 snapshot was corrupted: title=%s, questions=%d", inst1.Snapshot.Title, len(inst1.Snapshot.Questions))
	}
	if !inst1.IsStale("ver_2") {
		t.Errorf("expected inst1 to be flagged stale against latest version ver_2")
	}
}

func TestTemplateLifecycle_InstantiateUnpublishedRejection(t *testing.T) {
	tv, _ := checklist.NewTemplateVersion("tmpl_01", "v1", "Title", sampleQuestions())

	// Draft instantiate -> rejected
	_, err := tv.Instantiate("inst_01", "ten_alpha")
	if !errors.Is(err, checklist.ErrNotPublished) {
		t.Fatalf("expected ErrNotPublished on draft instantiate, got: %v", err)
	}

	// InReview instantiate -> rejected
	_ = tv.SubmitForReview()
	_, err = tv.Instantiate("inst_01", "ten_alpha")
	if !errors.Is(err, checklist.ErrNotPublished) {
		t.Fatalf("expected ErrNotPublished on in-review instantiate, got: %v", err)
	}

	// Publish and verify parameter validation
	_ = tv.RecordReview("alice", "ok")
	_ = tv.Approve("bob", "auth")
	_ = tv.Publish()

	// Empty instance ID
	_, err = tv.Instantiate("", "ten_alpha")
	if !errors.Is(err, checklist.ErrEmptyInstanceID) {
		t.Errorf("expected ErrEmptyInstanceID, got: %v", err)
	}

	// Empty tenant ID
	_, err = tv.Instantiate("inst_01", "")
	if !errors.Is(err, checklist.ErrEmptyTenantID) {
		t.Errorf("expected ErrEmptyTenantID, got: %v", err)
	}
}

func TestTemplateLifecycle_InputValidation(t *testing.T) {
	_, err := checklist.NewTemplateVersion("", "v1", "Title", sampleQuestions())
	if !errors.Is(err, checklist.ErrEmptyTemplateID) {
		t.Errorf("expected ErrEmptyTemplateID, got: %v", err)
	}

	_, err = checklist.NewTemplateVersion("tmpl_01", "", "Title", sampleQuestions())
	if !errors.Is(err, checklist.ErrEmptyVersionID) {
		t.Errorf("expected ErrEmptyVersionID, got: %v", err)
	}

	_, err = checklist.NewTemplateVersion("tmpl_01", "v1", "", sampleQuestions())
	if !errors.Is(err, checklist.ErrEmptyTitle) {
		t.Errorf("expected ErrEmptyTitle, got: %v", err)
	}

	_, err = checklist.NewTemplateVersion("tmpl_01", "v1", "Title", nil)
	if !errors.Is(err, checklist.ErrEmptyQuestionList) {
		t.Errorf("expected ErrEmptyQuestionList, got: %v", err)
	}
}
