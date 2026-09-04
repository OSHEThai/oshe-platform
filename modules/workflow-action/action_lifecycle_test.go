package workflowaction_test

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	workflowaction "github.com/oshethai/oshe-platform/modules/workflow-action"
)

const (
	validEvidenceDigest1 = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	validEvidenceDigest2 = "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
)


func defaultActionReq(id string) workflowaction.CreateActionRequest {
	return workflowaction.CreateActionRequest{
		ID:                    id,
		TenantID:              "ten_alpha",
		Title:                 "Repair Guardrail on Platform 3",
		Owner:                 "user_owner_alice",
		Reviewer:              "user_reviewer_bob",
		DueDate:               time.Now().Add(24 * time.Hour),
		RequiredEvidenceCount: 2,
		Creator:               "user_reviewer_bob",
	}
}

func TestUnauthorizedAction(t *testing.T) {
	mgr := workflowaction.NewActionManager(nil)
	_, err := mgr.CreateAction(defaultActionReq("act_unauth"))
	if err != nil {
		t.Fatalf("create action failed: %v", err)
	}

	intruder := "user_intruder_charlie"

	// 1. Non-owner attempts to start work
	if err := mgr.StartWork("act_unauth", intruder); !errors.Is(err, workflowaction.ErrUnauthorizedAction) {
		t.Errorf("expected ErrUnauthorizedAction on StartWork by intruder, got: %v", err)
	}

	// 2. Non-owner attempts to attach evidence
	ev := workflowaction.EvidenceAttachment{
		EvidenceID: "evd_001",
		TenantID:   "ten_alpha",
		Digest:     validEvidenceDigest1,
	}
	if err := mgr.AttachEvidence("act_unauth", intruder, ev); !errors.Is(err, workflowaction.ErrUnauthorizedAction) {
		t.Errorf("expected ErrUnauthorizedAction on AttachEvidence by intruder, got: %v", err)
	}

	// 3. Non-owner attempts to submit for review
	if err := mgr.SubmitForReview("act_unauth", intruder, "done"); !errors.Is(err, workflowaction.ErrUnauthorizedAction) {
		t.Errorf("expected ErrUnauthorizedAction on SubmitForReview by intruder, got: %v", err)
	}

	// Valid owner attaches 2 pieces of evidence and submits
	_ = mgr.AttachEvidence("act_unauth", "user_owner_alice", ev)
	ev2 := workflowaction.EvidenceAttachment{
		EvidenceID: "evd_002",
		TenantID:   "ten_alpha",
		Digest:     validEvidenceDigest2,
	}
	_ = mgr.AttachEvidence("act_unauth", "user_owner_alice", ev2)
	_ = mgr.SubmitForReview("act_unauth", "user_owner_alice", "ready for review")

	// 4. Non-reviewer attempts to reject
	if err := mgr.RejectReview("act_unauth", intruder, "rejected"); !errors.Is(err, workflowaction.ErrUnauthorizedAction) {
		t.Errorf("expected ErrUnauthorizedAction on RejectReview by intruder, got: %v", err)
	}

	// 5. Non-reviewer attempts to close
	if err := mgr.CloseAction("act_unauth", intruder, "closed"); !errors.Is(err, workflowaction.ErrUnauthorizedAction) {
		t.Errorf("expected ErrUnauthorizedAction on CloseAction by intruder, got: %v", err)
	}

	// Valid reviewer closes action
	if err := mgr.CloseAction("act_unauth", "user_reviewer_bob", "all good"); err != nil {
		t.Fatalf("close failed: %v", err)
	}

	// 6. Non-reviewer attempts to reopen
	if err := mgr.ReopenAction("act_unauth", intruder, "reopen"); !errors.Is(err, workflowaction.ErrUnauthorizedAction) {
		t.Errorf("expected ErrUnauthorizedAction on ReopenAction by intruder, got: %v", err)
	}
}

func TestInsufficientEvidence(t *testing.T) {
	mgr := workflowaction.NewActionManager(nil)
	_, _ = mgr.CreateAction(defaultActionReq("act_insufficient"))

	// Owner attempts to submit with 0 attachments (requires 2)
	err := mgr.SubmitForReview("act_insufficient", "user_owner_alice", "ready")
	if !errors.Is(err, workflowaction.ErrInsufficientEvidence) {
		t.Fatalf("expected ErrInsufficientEvidence with 0 evidence, got: %v", err)
	}

	// Attach 1 piece of evidence
	ev1 := workflowaction.EvidenceAttachment{
		EvidenceID: "evd_single",
		TenantID:   "ten_alpha",
		Digest:     validEvidenceDigest1,
	}
	_ = mgr.AttachEvidence("act_insufficient", "user_owner_alice", ev1)

	// Owner attempts to submit with 1 attachment (requires 2)
	err = mgr.SubmitForReview("act_insufficient", "user_owner_alice", "ready")
	if !errors.Is(err, workflowaction.ErrInsufficientEvidence) {
		t.Fatalf("expected ErrInsufficientEvidence with 1 evidence, got: %v", err)
	}

	// Attach 2nd piece of evidence
	ev2 := workflowaction.EvidenceAttachment{
		EvidenceID: "evd_second",
		TenantID:   "ten_alpha",
		Digest:     validEvidenceDigest2,
	}
	_ = mgr.AttachEvidence("act_insufficient", "user_owner_alice", ev2)

	// Now submission succeeds
	if err := mgr.SubmitForReview("act_insufficient", "user_owner_alice", "ready now"); err != nil {
		t.Fatalf("expected successful submission, got: %v", err)
	}

	snap, _ := mgr.GetAction("act_insufficient")
	if snap.State != workflowaction.ActionStateInReview {
		t.Errorf("expected state IN_REVIEW, got: %s", snap.State)
	}
}

func TestReviewRejectionAndResubmission(t *testing.T) {
	mgr := workflowaction.NewActionManager(nil)
	_, _ = mgr.CreateAction(defaultActionReq("act_rejection"))

	_ = mgr.AttachEvidence("act_rejection", "user_owner_alice", workflowaction.EvidenceAttachment{
		EvidenceID: "evd_1",
		TenantID:   "ten_alpha",
		Digest:     validEvidenceDigest1,
	})
	_ = mgr.AttachEvidence("act_rejection", "user_owner_alice", workflowaction.EvidenceAttachment{
		EvidenceID: "evd_2",
		TenantID:   "ten_alpha",
		Digest:     validEvidenceDigest2,
	})
	_ = mgr.SubmitForReview("act_rejection", "user_owner_alice", "submitted")

	// Blank rejection reason rejected
	if err := mgr.RejectReview("act_rejection", "user_reviewer_bob", "   "); !errors.Is(err, workflowaction.ErrBlankReason) {
		t.Errorf("expected ErrBlankReason on empty rejection notes, got: %v", err)
	}

	// Reviewer rejects with substantive reason
	if err := mgr.RejectReview("act_rejection", "user_reviewer_bob", "Photo 2 is blurry; retake inspection photo"); err != nil {
		t.Fatalf("unexpected error on valid rejection: %v", err)
	}

	snap, _ := mgr.GetAction("act_rejection")
	if snap.State != workflowaction.ActionStateRejected {
		t.Errorf("expected state REJECTED, got: %s", snap.State)
	}

	// Owner restarts work and attaches corrective evidence
	if err := mgr.StartWork("act_rejection", "user_owner_alice"); err != nil {
		t.Fatalf("start work after rejection failed: %v", err)
	}
	_ = mgr.AttachEvidence("act_rejection", "user_owner_alice", workflowaction.EvidenceAttachment{
		EvidenceID: "evd_3_retake",
		TenantID:   "ten_alpha",
		Digest:     validEvidenceDigest1,
	})

	// Resubmit
	if err := mgr.SubmitForReview("act_rejection", "user_owner_alice", "retake attached"); err != nil {
		t.Fatalf("resubmission failed: %v", err)
	}

	// Reviewer closes
	if err := mgr.CloseAction("act_rejection", "user_reviewer_bob", "verified clear photo"); err != nil {
		t.Fatalf("closure failed: %v", err)
	}

	snapClosed, _ := mgr.GetAction("act_rejection")
	if snapClosed.State != workflowaction.ActionStateClosed {
		t.Errorf("expected state CLOSED, got: %s", snapClosed.State)
	}
}

func TestReopenFlow(t *testing.T) {
	mgr := workflowaction.NewActionManager(nil)
	_, _ = mgr.CreateAction(defaultActionReq("act_reopen"))

	_ = mgr.AttachEvidence("act_reopen", "user_owner_alice", workflowaction.EvidenceAttachment{
		EvidenceID: "evd_1",
		TenantID:   "ten_alpha",
		Digest:     validEvidenceDigest1,
	})
	_ = mgr.AttachEvidence("act_reopen", "user_owner_alice", workflowaction.EvidenceAttachment{
		EvidenceID: "evd_2",
		TenantID:   "ten_alpha",
		Digest:     validEvidenceDigest2,
	})
	_ = mgr.SubmitForReview("act_reopen", "user_owner_alice", "done")
	_ = mgr.CloseAction("act_reopen", "user_reviewer_bob", "accepted")

	// Attempting to reopen with blank reason
	if err := mgr.ReopenAction("act_reopen", "user_reviewer_bob", ""); !errors.Is(err, workflowaction.ErrBlankReason) {
		t.Errorf("expected ErrBlankReason, got: %v", err)
	}

	// Valid reopen
	if err := mgr.ReopenAction("act_reopen", "user_reviewer_bob", "New safety hazard identified post-repair"); err != nil {
		t.Fatalf("reopen failed: %v", err)
	}

	snapReopened, _ := mgr.GetAction("act_reopen")
	if snapReopened.State != workflowaction.ActionStateReopened {
		t.Errorf("expected state REOPENED, got: %s", snapReopened.State)
	}
	if snapReopened.ClosedAt != nil || snapReopened.ClosedBy != "" {
		t.Errorf("reopened action must clear closure markers")
	}

	// Reopening an unclosed action fails closed
	if err := mgr.ReopenAction("act_reopen", "user_reviewer_bob", "again"); !errors.Is(err, workflowaction.ErrInvalidPrecedingState) {
		t.Errorf("expected ErrInvalidPrecedingState reopening non-closed action, got: %v", err)
	}
}

func TestOverdueHandling(t *testing.T) {
	clock, advance := newTestClock(t)
	baseTime := clock()

	mgr := workflowaction.NewActionManager(clock)
	req := defaultActionReq("act_overdue")
	req.DueDate = baseTime.Add(1 * time.Hour)
	_, _ = mgr.CreateAction(req)

	// At T + 30m, not overdue
	advance(30 * time.Minute)
	overdue, err := mgr.CheckOverdue("act_overdue")
	if err != nil || overdue {
		t.Errorf("expected not overdue at 30m, got overdue=%v, err=%v", overdue, err)
	}

	// At T + 2h, overdue
	advance(90 * time.Minute)
	overdue, err = mgr.CheckOverdue("act_overdue")
	if err != nil || !overdue {
		t.Errorf("expected overdue at 2h, got overdue=%v, err=%v", overdue, err)
	}

	snap, _ := mgr.GetAction("act_overdue")
	if snap.State != workflowaction.ActionStateOverdue {
		t.Errorf("expected state OVERDUE, got %s", snap.State)
	}
}

func TestCrossTenantDenial(t *testing.T) {
	mgr := workflowaction.NewActionManager(nil)
	req := defaultActionReq("act_cross")
	req.TenantID = "ten_alpha"
	_, _ = mgr.CreateAction(req)

	// Attempt to attach evidence belonging to ten_bravo
	evCross := workflowaction.EvidenceAttachment{
		EvidenceID: "evd_bravo",
		TenantID:   "ten_bravo",
		Digest:     validEvidenceDigest1,
	}

	err := mgr.AttachEvidence("act_cross", "user_owner_alice", evCross)
	if !errors.Is(err, workflowaction.ErrCrossTenantDenied) {
		t.Fatalf("expected ErrCrossTenantDenied, got: %v", err)
	}
}

func TestDuplicateAndConcurrentClosureDenial(t *testing.T) {
	mgr := workflowaction.NewActionManager(nil)
	_, _ = mgr.CreateAction(defaultActionReq("act_closure"))

	_ = mgr.AttachEvidence("act_closure", "user_owner_alice", workflowaction.EvidenceAttachment{
		EvidenceID: "evd_1",
		TenantID:   "ten_alpha",
		Digest:     validEvidenceDigest1,
	})
	_ = mgr.AttachEvidence("act_closure", "user_owner_alice", workflowaction.EvidenceAttachment{
		EvidenceID: "evd_2",
		TenantID:   "ten_alpha",
		Digest:     validEvidenceDigest2,
	})
	_ = mgr.SubmitForReview("act_closure", "user_owner_alice", "ready")

	// Concurrent closures
	const concurrency = 20
	var wg sync.WaitGroup
	var successCount int64
	var closedErrCount int64

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := mgr.CloseAction("act_closure", "user_reviewer_bob", "closing")
			if err == nil {
				atomic.AddInt64(&successCount, 1)
			} else if errors.Is(err, workflowaction.ErrActionClosed) {
				atomic.AddInt64(&closedErrCount, 1)
			}
		}()
	}

	wg.Wait()

	if successCount != 1 {
		t.Fatalf("expected exactly 1 successful closure, got %d", successCount)
	}
	if closedErrCount != concurrency-1 {
		t.Fatalf("expected %d closures rejected with ErrActionClosed, got %d", concurrency-1, closedErrCount)
	}

	// Subsequent sequential call must also return ErrActionClosed
	if err := mgr.CloseAction("act_closure", "user_reviewer_bob", "re-closing"); !errors.Is(err, workflowaction.ErrActionClosed) {
		t.Errorf("expected ErrActionClosed on subsequent close call, got: %v", err)
	}
}

func TestCompleteHistoryRequirement(t *testing.T) {
	mgr := workflowaction.NewActionManager(nil)
	_, _ = mgr.CreateAction(defaultActionReq("act_history"))

	_ = mgr.StartWork("act_history", "user_owner_alice")
	_ = mgr.AttachEvidence("act_history", "user_owner_alice", workflowaction.EvidenceAttachment{
		EvidenceID: "evd_1",
		TenantID:   "ten_alpha",
		Digest:     validEvidenceDigest1,
	})
	_ = mgr.AttachEvidence("act_history", "user_owner_alice", workflowaction.EvidenceAttachment{
		EvidenceID: "evd_2",
		TenantID:   "ten_alpha",
		Digest:     validEvidenceDigest2,
	})
	_ = mgr.SubmitForReview("act_history", "user_owner_alice", "review please")
	_ = mgr.RejectReview("act_history", "user_reviewer_bob", "need clearer photo")
	_ = mgr.StartWork("act_history", "user_owner_alice")
	_ = mgr.AttachEvidence("act_history", "user_owner_alice", workflowaction.EvidenceAttachment{
		EvidenceID: "evd_3",
		TenantID:   "ten_alpha",
		Digest:     validEvidenceDigest2,
	})
	_ = mgr.SubmitForReview("act_history", "user_owner_alice", "resubmitted")
	_ = mgr.CloseAction("act_history", "user_reviewer_bob", "verified and closed")
	_ = mgr.ReopenAction("act_history", "user_reviewer_bob", "reopened for inspection")

	snap, _ := mgr.GetAction("act_history")
	history := snap.History

	if len(history) < 10 {
		t.Fatalf("expected at least 10 history entries, got %d", len(history))
	}

	for i, entry := range history {
		expectedSeq := int64(i + 1)
		if entry.Sequence != expectedSeq {
			t.Errorf("entry %d: expected sequence %d, got %d", i, expectedSeq, entry.Sequence)
		}
		if entry.Timestamp.IsZero() {
			t.Errorf("entry %d: timestamp is zero", i)
		}
		if entry.Actor == "" {
			t.Errorf("entry %d: actor is blank", i)
		}
	}
}
