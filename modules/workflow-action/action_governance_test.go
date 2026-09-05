package workflowaction_test

import (
	"testing"
	"time"

	workflowaction "github.com/oshethai/oshe-platform/modules/workflow-action"
)

func setupGovernanceEngine() (*workflowaction.ActionGovernanceEngine, workflowaction.Clock, time.Time) {
	t0 := time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC)
	clock := func() time.Time { return t0 }
	engine := workflowaction.NewActionGovernanceEngine(clock)
	return engine, clock, t0
}

func TestOwnership_VisibleHistoryAndReassignment(t *testing.T) {
	engine, _, t0 := setupGovernanceEngine()

	actID := "act_syn_fire_barrier_01"
	err := engine.RegisterAction(workflowaction.GovernedAction{
		ActionID:              actID,
		TenantID:              "ten_synthetic_alpha",
		FindingID:             "fnd_syn_fire_barrier_01",
		Title:                 "Repair compromised fire barrier sealant",
		State:                 workflowaction.ActionStateAssigned,
		CurrentOwner:          "usr_capa_owner_01",
		CurrentOwnerRole:      "CAPA_OWNER",
		DueDate:               t0.Add(72 * time.Hour),
		RequiredEvidenceCount: 2,
	})
	if err != nil {
		t.Fatalf("failed to register action: %v", err)
	}

	act, err := engine.GetAction(actID)
	if err != nil {
		t.Fatalf("failed to get action: %v", err)
	}
	if len(act.OwnershipHistory) != 1 {
		t.Fatalf("expected 1 initial ownership record, got %d", len(act.OwnershipHistory))
	}
	if act.OwnershipHistory[0].OwnerSubject != "usr_capa_owner_01" || !act.OwnershipHistory[0].IsActive {
		t.Errorf("initial ownership record malformed: %+v", act.OwnershipHistory[0])
	}

	// Reassign to successor
	reassignedAct, err := engine.ReassignOwner(
		actID,
		"usr_capa_owner_02",
		"CAPA_OWNER",
		"usr_supervisor_01",
		"Primary contractor rotated off-site",
		act.StateVersion,
	)
	if err != nil {
		t.Fatalf("reassignment failed: %v", err)
	}

	if reassignedAct.CurrentOwner != "usr_capa_owner_02" {
		t.Errorf("expected current owner usr_capa_owner_02, got %s", reassignedAct.CurrentOwner)
	}
	if len(reassignedAct.OwnershipHistory) != 2 {
		t.Fatalf("expected 2 ownership records, got %d", len(reassignedAct.OwnershipHistory))
	}

	// Prior owner record must be deactivated and preserved
	prior := reassignedAct.OwnershipHistory[0]
	if prior.IsActive {
		t.Errorf("expected prior owner to be inactive")
	}
	if prior.OwnerSubject != "usr_capa_owner_01" {
		t.Errorf("prior owner subject corrupted: %s", prior.OwnerSubject)
	}
	if prior.RevocationReason == "" {
		t.Errorf("expected non-blank revocation reason on prior owner")
	}

	// New active owner record
	current := reassignedAct.OwnershipHistory[1]
	if !current.IsActive || current.OwnerSubject != "usr_capa_owner_02" {
		t.Errorf("current owner record malformed: %+v", current)
	}
}

func TestOwnership_Revocation(t *testing.T) {
	engine, _, t0 := setupGovernanceEngine()

	actID := "act_syn_spill_kit_02"
	_ = engine.RegisterAction(workflowaction.GovernedAction{
		ActionID:         actID,
		TenantID:         "ten_synthetic_alpha",
		FindingID:        "fnd_syn_spill_kit_02",
		Title:            "Restock chemical spill kit neutralizer",
		State:            workflowaction.ActionStateAssigned,
		CurrentOwner:     "usr_capa_owner_01",
		CurrentOwnerRole: "CAPA_OWNER",
		DueDate:          t0.Add(48 * time.Hour),
	})

	act, _ := engine.GetAction(actID)
	revokedAct, err := engine.RevokeOwner(actID, "usr_supervisor_01", "Assigned contractor contract terminated", act.StateVersion)
	if err != nil {
		t.Fatalf("revocation failed: %v", err)
	}

	if revokedAct.CurrentOwner != "" {
		t.Errorf("expected empty current owner after revocation, got %s", revokedAct.CurrentOwner)
	}
	if len(revokedAct.OwnershipHistory) != 1 || revokedAct.OwnershipHistory[0].IsActive {
		t.Errorf("expected deactivated ownership record in history")
	}

	// Re-revoking without owner must fail closed
	_, err = engine.RevokeOwner(actID, "usr_supervisor_01", "Second attempt", revokedAct.StateVersion)
	if err != workflowaction.ErrNoActiveOwner {
		t.Errorf("expected ErrNoActiveOwner, got %v", err)
	}
}

func TestExtension_RequestAndApprovalWorkflow(t *testing.T) {
	engine, _, t0 := setupGovernanceEngine()

	actID := "act_syn_scaffold_tag_03"
	initialDue := t0.Add(24 * time.Hour)
	requestedDue := t0.Add(72 * time.Hour)

	_ = engine.RegisterAction(workflowaction.GovernedAction{
		ActionID:         actID,
		TenantID:         "ten_synthetic_alpha",
		FindingID:        "fnd_syn_scaffold_tag_03",
		Title:            "Replace weathered green inspection tag on mobile scaffold",
		State:            workflowaction.ActionStateInProgress,
		CurrentOwner:     "usr_capa_owner_01",
		CurrentOwnerRole: "CAPA_OWNER",
		DueDate:          initialDue,
	})

	act, _ := engine.GetAction(actID)

	// Owner requests extension
	extReq := workflowaction.ExtensionRequest{
		RequestID:        "ext_req_01",
		ActionID:         actID,
		RequestedBy:      "usr_capa_owner_01",
		RequestedDueDate: requestedDue,
		Reason:           "Delayed delivery of replacement durable tag supplies from vendor",
	}

	withReq, err := engine.RequestExtension(extReq, act.StateVersion)
	if err != nil {
		t.Fatalf("extension request failed: %v", err)
	}
	if len(withReq.ExtensionRequests) != 1 {
		t.Fatalf("expected 1 extension request, got %d", len(withReq.ExtensionRequests))
	}
	if withReq.ExtensionRequests[0].Status != workflowaction.ExtensionStatusPending {
		t.Errorf("expected status PENDING, got %s", withReq.ExtensionRequests[0].Status)
	}

	// Independent Reviewer approves extension
	approvedAct, err := engine.ReviewExtension(
		actID,
		"ext_req_01",
		"usr_reviewer_01",
		true,
		"Vendor delay verified; approved 48h extension",
		withReq.StateVersion,
	)
	if err != nil {
		t.Fatalf("extension review failed: %v", err)
	}

	if approvedAct.DueDate != requestedDue {
		t.Errorf("expected updated due date %v, got %v", requestedDue, approvedAct.DueDate)
	}
	if approvedAct.ExtensionRequests[0].Status != workflowaction.ExtensionStatusApproved {
		t.Errorf("expected status APPROVED, got %s", approvedAct.ExtensionRequests[0].Status)
	}
	if approvedAct.ExtensionRequests[0].ReviewedBy != "usr_reviewer_01" {
		t.Errorf("expected reviewer usr_reviewer_01, got %s", approvedAct.ExtensionRequests[0].ReviewedBy)
	}
}

func TestExtension_SelfApprovalProhibition(t *testing.T) {
	engine, _, t0 := setupGovernanceEngine()

	actID := "act_syn_guardrail_04"
	_ = engine.RegisterAction(workflowaction.GovernedAction{
		ActionID:         actID,
		TenantID:         "ten_synthetic_alpha",
		FindingID:        "fnd_syn_guardrail_04",
		Title:            "Secure loose intermediate guardrail clamp",
		State:            workflowaction.ActionStateInProgress,
		CurrentOwner:     "usr_capa_owner_01",
		CurrentOwnerRole: "CAPA_OWNER",
		DueDate:          t0.Add(24 * time.Hour),
	})

	act, _ := engine.GetAction(actID)
	withReq, _ := engine.RequestExtension(workflowaction.ExtensionRequest{
		RequestID:        "ext_req_sod_01",
		ActionID:         actID,
		RequestedBy:      "usr_capa_owner_01",
		RequestedDueDate: t0.Add(48 * time.Hour),
		Reason:           "Fabrication queue delay",
	}, act.StateVersion)

	// Owner attempts self-approval -> MUST FAIL
	_, err := engine.ReviewExtension(
		actID,
		"ext_req_sod_01",
		"usr_capa_owner_01", // same as owner / requester
		true,
		"Self-approving my own extension",
		withReq.StateVersion,
	)
	if err != workflowaction.ErrSelfApprovalProhibited {
		t.Fatalf("expected ErrSelfApprovalProhibited on self-approval, got %v", err)
	}
}

func TestExtension_RejectionWorkflow(t *testing.T) {
	engine, _, t0 := setupGovernanceEngine()

	actID := "act_syn_loto_05"
	originalDue := t0.Add(24 * time.Hour)

	_ = engine.RegisterAction(workflowaction.GovernedAction{
		ActionID:         actID,
		TenantID:         "ten_synthetic_alpha",
		FindingID:        "fnd_syn_loto_05",
		Title:            "Install lockout hasp on main feeder breaker",
		State:            workflowaction.ActionStateInProgress,
		CurrentOwner:     "usr_capa_owner_01",
		CurrentOwnerRole: "CAPA_OWNER",
		DueDate:          originalDue,
	})

	act, _ := engine.GetAction(actID)
	withReq, _ := engine.RequestExtension(workflowaction.ExtensionRequest{
		RequestID:        "ext_req_reject_01",
		ActionID:         actID,
		RequestedBy:      "usr_capa_owner_01",
		RequestedDueDate: t0.Add(96 * time.Hour),
		Reason:           "High maintenance backlog",
	}, act.StateVersion)

	// Reviewer rejects extension
	rejectedAct, err := engine.ReviewExtension(
		actID,
		"ext_req_reject_01",
		"usr_reviewer_02",
		false,
		"Critical electrical hazard; extension rejected. Must isolate immediately.",
		withReq.StateVersion,
	)
	if err != nil {
		t.Fatalf("rejection review failed: %v", err)
	}

	if rejectedAct.DueDate != originalDue {
		t.Errorf("due date was altered on rejection: expected %v, got %v", originalDue, rejectedAct.DueDate)
	}
	if rejectedAct.ExtensionRequests[0].Status != workflowaction.ExtensionStatusRejected {
		t.Errorf("expected status REJECTED, got %s", rejectedAct.ExtensionRequests[0].Status)
	}
}

func TestEscalation_RequestAndAcknowledgement(t *testing.T) {
	engine, _, t0 := setupGovernanceEngine()

	actID := "act_syn_chemical_leak_06"
	_ = engine.RegisterAction(workflowaction.GovernedAction{
		ActionID:         actID,
		TenantID:         "ten_synthetic_alpha",
		FindingID:        "fnd_syn_chemical_leak_06",
		Title:            "Contain and repair sulfuric acid metering pump seal leak",
		State:            workflowaction.ActionStateInProgress,
		CurrentOwner:     "usr_capa_owner_01",
		CurrentOwnerRole: "CAPA_OWNER",
		DueDate:          t0.Add(12 * time.Hour),
	})

	act, _ := engine.GetAction(actID)

	// Trigger escalation
	withEsc, err := engine.RequestEscalation(workflowaction.EscalationRequest{
		RequestID:   "esc_req_01",
		ActionID:    actID,
		EscalatedBy: "usr_capa_owner_01",
		TargetLevel: "LEVEL_1_PLANT_SAFETY_DIRECTOR",
		Reason:      "Corrosion has compromised secondary containment sump; immediate intervention required.",
	}, act.StateVersion)
	if err != nil {
		t.Fatalf("escalation request failed: %v", err)
	}

	// Self-acknowledgement prohibited
	_, err = engine.AcknowledgeEscalation(actID, "esc_req_01", "usr_capa_owner_01", "Self ack", withEsc.StateVersion)
	if err != workflowaction.ErrSelfApprovalProhibited {
		t.Errorf("expected ErrSelfApprovalProhibited on self-acknowledgement, got %v", err)
	}

	// Plant supervisor acknowledges
	ackedAct, err := engine.AcknowledgeEscalation(
		actID,
		"esc_req_01",
		"usr_plant_director_01",
		"Emergency containment team dispatched to unit.",
		withEsc.StateVersion,
	)
	if err != nil {
		t.Fatalf("acknowledgement failed: %v", err)
	}

	if ackedAct.EscalationRequests[0].Status != workflowaction.EscalationStatusAcknowledged {
		t.Errorf("expected status ACKNOWLEDGED, got %s", ackedAct.EscalationRequests[0].Status)
	}
}

func TestEvidence_SubmissionAndReviewWorkflow(t *testing.T) {
	engine, _, t0 := setupGovernanceEngine()

	actID := "act_syn_extinguisher_07"
	_ = engine.RegisterAction(workflowaction.GovernedAction{
		ActionID:              actID,
		TenantID:              "ten_synthetic_alpha",
		FindingID:             "fnd_syn_extinguisher_07",
		Title:                 "Recharge discharged dry chemical fire extinguisher Unit 14",
		State:                 workflowaction.ActionStateInProgress,
		CurrentOwner:          "usr_capa_owner_01",
		CurrentOwnerRole:      "CAPA_OWNER",
		DueDate:               t0.Add(24 * time.Hour),
		RequiredEvidenceCount: 1,
	})

	act, _ := engine.GetAction(actID)

	// Owner submits evidence photo
	withEv, err := engine.SubmitEvidence(actID, workflowaction.GovernedEvidence{
		EvidenceID:  "evd_capa_recharge_photo_01",
		Digest:      "a1b2c3d4e5f60718293a4b5c6d7e8f90123456789abcdef0123456789abcdef0",
		MediaType:   "image/jpeg",
		SubmittedBy: "usr_capa_owner_01",
		Description: "Photograph of recharged pressure gauge in green zone and new inspection tag",
	}, act.StateVersion)
	if err != nil {
		t.Fatalf("evidence submission failed: %v", err)
	}

	if len(withEv.EvidenceList) != 1 || withEv.EvidenceList[0].Status != workflowaction.EvidenceStatusSubmitted {
		t.Fatalf("expected 1 SUBMITTED evidence item")
	}

	// Submitter cannot review own evidence
	_, err = engine.ReviewEvidence(actID, "evd_capa_recharge_photo_01", "usr_capa_owner_01", true, "Self ok", withEv.StateVersion)
	if err != workflowaction.ErrSelfApprovalProhibited {
		t.Errorf("expected ErrSelfApprovalProhibited on self-review, got %v", err)
	}

	// Reviewer accepts evidence
	acceptedAct, err := engine.ReviewEvidence(
		actID,
		"evd_capa_recharge_photo_01",
		"usr_reviewer_01",
		true,
		"Verified pressure gauge and collar tag validity",
		withEv.StateVersion,
	)
	if err != nil {
		t.Fatalf("evidence review acceptance failed: %v", err)
	}

	if acceptedAct.AcceptedEvidenceCount() != 1 {
		t.Errorf("expected 1 accepted evidence count, got %d", acceptedAct.AcceptedEvidenceCount())
	}
	if acceptedAct.EvidenceList[0].Status != workflowaction.EvidenceStatusAccepted {
		t.Errorf("expected status ACCEPTED, got %s", acceptedAct.EvidenceList[0].Status)
	}
}

func TestConcurrency_OptimisticLocking(t *testing.T) {
	engine, _, t0 := setupGovernanceEngine()

	actID := "act_syn_concurrency_08"
	_ = engine.RegisterAction(workflowaction.GovernedAction{
		ActionID:         actID,
		TenantID:         "ten_synthetic_alpha",
		FindingID:        "fnd_syn_concurrency_08",
		Title:            "Clear obstructed emergency eye wash station",
		State:            workflowaction.ActionStateInProgress,
		CurrentOwner:     "usr_capa_owner_01",
		CurrentOwnerRole: "CAPA_OWNER",
		DueDate:          t0.Add(24 * time.Hour),
		StateVersion:     1,
	})

	// Mutation 1 succeeds with expected version 1
	updated1, err := engine.ReassignOwner(actID, "usr_capa_owner_02", "CAPA_OWNER", "usr_sup_01", "Rotation", 1)
	if err != nil {
		t.Fatalf("mutation 1 failed: %v", err)
	}
	if updated1.StateVersion != 2 {
		t.Errorf("expected version 2, got %d", updated1.StateVersion)
	}

	// Mutation 2 with stale version 1 MUST FAIL with ErrConcurrentModification
	_, err = engine.ReassignOwner(actID, "usr_capa_owner_03", "CAPA_OWNER", "usr_sup_01", "Conflicting mutation", 1)
	if err != workflowaction.ErrConcurrentModification {
		t.Fatalf("expected ErrConcurrentModification on stale version, got %v", err)
	}
}

func TestDuplicatePrevention(t *testing.T) {
	engine, _, t0 := setupGovernanceEngine()

	actID := "act_syn_duplicate_09"
	_ = engine.RegisterAction(workflowaction.GovernedAction{
		ActionID:         actID,
		TenantID:         "ten_synthetic_alpha",
		FindingID:        "fnd_syn_duplicate_09",
		Title:            "Replace damaged safety signage",
		State:            workflowaction.ActionStateInProgress,
		CurrentOwner:     "usr_capa_owner_01",
		CurrentOwnerRole: "CAPA_OWNER",
		DueDate:          t0.Add(24 * time.Hour),
	})

	// Duplicate Action Registration
	err := engine.RegisterAction(workflowaction.GovernedAction{
		ActionID: actID,
		TenantID: "ten_synthetic_alpha",
	})
	if err != workflowaction.ErrDuplicateActionID {
		t.Errorf("expected ErrDuplicateActionID, got %v", err)
	}

	act, _ := engine.GetAction(actID)

	// First evidence
	withEv, err := engine.SubmitEvidence(actID, workflowaction.GovernedEvidence{
		EvidenceID:  "evd_dup_01",
		Digest:      "hash123",
		MediaType:   "image/jpeg",
		SubmittedBy: "usr_capa_owner_01",
	}, act.StateVersion)
	if err != nil {
		t.Fatalf("first evidence submission failed: %v", err)
	}

	// Duplicate evidence ID must fail closed
	_, err = engine.SubmitEvidence(actID, workflowaction.GovernedEvidence{
		EvidenceID:  "evd_dup_01",
		Digest:      "hash456",
		MediaType:   "image/png",
		SubmittedBy: "usr_capa_owner_01",
	}, withEv.StateVersion)
	if err != workflowaction.ErrDuplicateEvidenceID {
		t.Errorf("expected ErrDuplicateEvidenceID, got %v", err)
	}
}
