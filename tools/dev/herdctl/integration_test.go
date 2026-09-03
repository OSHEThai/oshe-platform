package main

import (
	"testing"
)

func TestIntegrationController_Lifecycle(t *testing.T) {
	s := newStore(t)
	id := "M-INT-1"
	if err := s.Create(testMission(id)); err != nil {
		t.Fatal(err)
	}

	c := &IntegrationController{Store: s}

	// 1. Verify dependencies
	if err := c.VerifyDependencies(id, []string{"DEP-1"}); err != nil {
		t.Fatal(err)
	}

	// 2. Prepare candidate
	if err := c.PrepareCandidate(id); err != nil {
		t.Fatal(err)
	}

	// 3. Review: Remediation Loop
	if err := c.SubmitReview(id, false); err != nil {
		t.Fatal(err)
	}

	// 4. Try handoff (should fail due to remediation required)
	if err := c.Handoff(id); err == nil {
		t.Fatal("expected handoff to fail during remediation")
	}

	// 5. Re-prepare candidate
	if err := c.PrepareCandidate(id); err != nil {
		t.Fatal(err)
	}

	// 6. Review: Approved
	if err := c.SubmitReview(id, true); err != nil {
		t.Fatal(err)
	}

	// 7. Try draft PR (should fail before handoff)
	if err := c.RequestDraftPR(id); err == nil {
		t.Fatal("expected draft PR to fail before handoff")
	}

	// 8. Handoff
	if err := c.Handoff(id); err != nil {
		t.Fatal(err)
	}

	// 9. Draft PR Request
	if err := c.RequestDraftPR(id); err != nil {
		t.Fatal(err)
	}

	// 10. Final validation
	st, err := c.Status(id)
	if err != nil {
		t.Fatal(err)
	}
	if !st.DraftPR || !st.Handoff || st.ReviewState != "APPROVED" || len(st.Dependencies) != 1 {
		t.Fatalf("unexpected state: %+v", st)
	}
}
