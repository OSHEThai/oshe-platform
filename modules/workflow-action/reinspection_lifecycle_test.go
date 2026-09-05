package workflowaction_test

import (
	"errors"
	"testing"
	"time"

	workflowaction "github.com/oshethai/oshe-platform/modules/workflow-action"
)

func setupReinspectionTestEnv() (*workflowaction.ReinspectionEngine, *workflowaction.ActionGovernanceEngine, time.Time) {
	t0 := time.Date(2026, 9, 6, 8, 0, 0, 0, time.UTC)
	clock := func() time.Time { return t0 }

	actionEngine := workflowaction.NewActionGovernanceEngine(clock)
	reinspectionEngine := workflowaction.NewReinspectionEngine(clock, actionEngine)
	return reinspectionEngine, actionEngine, t0
}

func TestReinspection_HappyPath_CompleteVerificationAndClosure(t *testing.T) {
	reinspectionEngine, actionEngine, t0 := setupReinspectionTestEnv()

	// 1. Setup governed action with 2 accepted evidence items
	actID := "act_syn_scaffold_01"
	err := actionEngine.RegisterAction(workflowaction.GovernedAction{
		ActionID:              actID,
		TenantID:              "ten_syn_alpha",
		FindingID:             "fnd_syn_scaffold_01",
		Title:                 "Secure loose scaffolding platform",
		State:                 workflowaction.ActionStateInReview,
		CurrentOwner:          "usr_capa_owner_alice",
		CurrentOwnerRole:      "CAPA_OWNER",
		DueDate:               t0.Add(48 * time.Hour),
		RequiredEvidenceCount: 2,
	})
	if err != nil {
		t.Fatalf("failed to register action: %v", err)
	}

	act, _ := actionEngine.GetAction(actID)
	act, err = actionEngine.SubmitEvidence(
		actID,
		workflowaction.GovernedEvidence{
			EvidenceID:  "evd_syn_scaffold_clamp_01",
			Digest:      "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad",
			MediaType:   "image/jpeg",
			SubmittedBy: "usr_capa_owner_alice",
			Description: "Installed galvanized locking clamps",
		},
		act.StateVersion,
	)
	if err != nil {
		t.Fatalf("submit evidence 1 failed: %v", err)
	}

	act, err = actionEngine.ReviewEvidence(
		actID,
		"evd_syn_scaffold_clamp_01",
		"usr_reviewer_bob",
		true, // accept
		"Clamps verified according to engineering spec",
		act.StateVersion,
	)
	if err != nil {
		t.Fatalf("review evidence 1 failed: %v", err)
	}

	act, err = actionEngine.SubmitEvidence(
		actID,
		workflowaction.GovernedEvidence{
			EvidenceID:  "evd_syn_scaffold_tag_02",
			Digest:      "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			MediaType:   "image/jpeg",
			SubmittedBy: "usr_capa_owner_alice",
			Description: "Green inspection tag placed on ladder access",
		},
		act.StateVersion,
	)
	if err != nil {
		t.Fatalf("submit evidence 2 failed: %v", err)
	}

	act, err = actionEngine.ReviewEvidence(
		actID,
		"evd_syn_scaffold_tag_02",
		"usr_reviewer_bob",
		true, // accept
		"Tag signed by certified safety supervisor",
		act.StateVersion,
	)
	if err != nil {
		t.Fatalf("review evidence 2 failed: %v", err)
	}

	// 2. Create reinspection order with independent inspector requirement
	orderID := "rin_syn_scaffold_01"
	order, err := reinspectionEngine.CreateReinspectionOrder(workflowaction.ReinspectionOrder{
		ReinspectionID:      orderID,
		TenantID:            "ten_syn_alpha",
		FindingID:           "fnd_syn_scaffold_01",
		ActionID:            actID,
		OriginalInspector:   "usr_inspector_charlie",
		AssignedReinspector: "usr_reinspector_dave",
		ReinspectorRole:     "INSPECTOR",
		IndependenceRule:    workflowaction.IndependenceDifferentInspectorRequired,
		ScheduledAt:         t0.Add(2 * time.Hour),
	})
	if err != nil {
		t.Fatalf("create reinspection order failed: %v", err)
	}

	if order.State != workflowaction.ReinspectionStateAssigned {
		t.Errorf("expected ASSIGNED, got %s", order.State)
	}
	if order.StateVersion != 1 {
		t.Errorf("expected StateVersion 1, got %d", order.StateVersion)
	}

	// 3. Start reinspection
	order, err = reinspectionEngine.StartReinspection(orderID, "usr_reinspector_dave", "INSPECTOR", order.StateVersion)
	if err != nil {
		t.Fatalf("start reinspection failed: %v", err)
	}
	if order.State != workflowaction.ReinspectionStateInProgress {
		t.Errorf("expected IN_PROGRESS, got %s", order.State)
	}

	// 4. Verify satisfactory with physical verification notes
	order, err = reinspectionEngine.VerifySatisfactory(
		orderID,
		"usr_reinspector_dave",
		"INSPECTOR",
		"Physical walk-through confirms all loose planks replaced and dual locking clamps affixed",
		[]string{"evd_syn_scaffold_clamp_01", "evd_syn_scaffold_tag_02"},
		order.StateVersion,
	)
	if err != nil {
		t.Fatalf("verify satisfactory failed: %v", err)
	}
	if order.State != workflowaction.ReinspectionStateVerifiedSatisfactory {
		t.Errorf("expected VERIFIED_SATISFACTORY, got %s", order.State)
	}

	// 5. Final closure by authorized human supervisor online
	order, err = reinspectionEngine.FinalClose(
		orderID,
		"usr_supervisor_elena",
		"SAFETY_SUPERVISOR",
		"Remediation verified on site; risk downgraded to acceptable residual risk",
		false, // online
		order.StateVersion,
	)
	if err != nil {
		t.Fatalf("final close failed: %v", err)
	}
	if order.State != workflowaction.ReinspectionStateClosed {
		t.Errorf("expected CLOSED, got %s", order.State)
	}
	if order.ClosedBy != "usr_supervisor_elena" {
		t.Errorf("expected ClosedBy usr_supervisor_elena, got %s", order.ClosedBy)
	}
	if order.ClosedAt == nil {
		t.Errorf("expected non-nil ClosedAt")
	}
	if len(order.History) != 4 {
		t.Errorf("expected 4 history entries, got %d", len(order.History))
	}
}

func TestReinspection_FailClosed_UnauthorizedClosure(t *testing.T) {
	reinspectionEngine, _, _ := setupReinspectionTestEnv()

	orderID := "rin_syn_unauth_01"
	order, err := reinspectionEngine.CreateReinspectionOrder(workflowaction.ReinspectionOrder{
		ReinspectionID:      orderID,
		TenantID:            "ten_syn_alpha",
		FindingID:           "fnd_syn_unauth_01",
		ActionID:            "act_syn_unauth_01",
		OriginalInspector:   "usr_inspector_01",
		AssignedReinspector: "usr_reinspector_02",
		ReinspectorRole:     "INSPECTOR",
		IndependenceRule:    workflowaction.IndependenceSameInspectorAllowed,
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	order, err = reinspectionEngine.StartReinspection(orderID, "usr_reinspector_02", "INSPECTOR", order.StateVersion)
	if err != nil {
		t.Fatalf("start failed: %v", err)
	}

	order, err = reinspectionEngine.VerifySatisfactory(
		orderID,
		"usr_reinspector_02",
		"INSPECTOR",
		"Physical verification satisfactory",
		nil,
		order.StateVersion,
	)
	if err != nil {
		t.Fatalf("verify failed: %v", err)
	}

	// 1. Regular inspector attempts closure -> fails with ErrUnauthorizedClosure
	_, err = reinspectionEngine.FinalClose(
		orderID,
		"usr_reinspector_02",
		"INSPECTOR",
		"Attempted closure by non-supervisor",
		false,
		order.StateVersion,
	)
	if !errors.Is(err, workflowaction.ErrUnauthorizedClosure) {
		t.Errorf("expected ErrUnauthorizedClosure for non-supervisor, got: %v", err)
	}

	// 2. Autonomous AI role attempts closure -> fails with ErrAutonomousClosureProhibited
	aiRoles := []string{"AI", "AI_AGENT", "ENGINEERING_AGENT", "SYSTEM_AGENT", "AUTONOMOUS_AGENT"}
	for _, role := range aiRoles {
		_, err = reinspectionEngine.FinalClose(
			orderID,
			"agent_ai_core_01",
			role,
			"Automated closure attempt by AI",
			false,
			order.StateVersion,
		)
		if !errors.Is(err, workflowaction.ErrAutonomousClosureProhibited) {
			t.Errorf("expected ErrAutonomousClosureProhibited for AI role %s, got: %v", role, err)
		}
	}
}

func TestReinspection_FailClosed_OfflineFinalClosureDenied(t *testing.T) {
	reinspectionEngine, _, _ := setupReinspectionTestEnv()

	orderID := "rin_syn_offline_01"
	order, err := reinspectionEngine.CreateReinspectionOrder(workflowaction.ReinspectionOrder{
		ReinspectionID:      orderID,
		TenantID:            "ten_syn_alpha",
		FindingID:           "fnd_syn_offline_01",
		ActionID:            "act_syn_offline_01",
		OriginalInspector:   "usr_inspector_01",
		AssignedReinspector: "usr_reinspector_02",
		ReinspectorRole:     "INSPECTOR",
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	order, _ = reinspectionEngine.StartReinspection(orderID, "usr_reinspector_02", "INSPECTOR", order.StateVersion)
	order, _ = reinspectionEngine.VerifySatisfactory(orderID, "usr_reinspector_02", "INSPECTOR", "Looks good", nil, order.StateVersion)

	// Attempting final closure with isOffline = true
	_, err = reinspectionEngine.FinalClose(
		orderID,
		"usr_supervisor_01",
		"SAFETY_SUPERVISOR",
		"Offline closure attempt in basement warehouse",
		true, // OFFLINE
		order.StateVersion,
	)
	if !errors.Is(err, workflowaction.ErrOfflineClosureProhibited) {
		t.Errorf("expected ErrOfflineClosureProhibited when isOffline=true, got: %v", err)
	}

	// Persistence verify: order must remain in VERIFIED_SATISFACTORY, not CLOSED
	persisted, _ := reinspectionEngine.GetReinspection(orderID)
	if persisted.State != workflowaction.ReinspectionStateVerifiedSatisfactory {
		t.Errorf("order state corrupted after failed offline closure: %s", persisted.State)
	}
}

func TestReinspection_FailClosed_StaleAndConcurrentUpdates(t *testing.T) {
	reinspectionEngine, _, _ := setupReinspectionTestEnv()

	orderID := "rin_syn_concur_01"
	order, err := reinspectionEngine.CreateReinspectionOrder(workflowaction.ReinspectionOrder{
		ReinspectionID:      orderID,
		TenantID:            "ten_syn_alpha",
		FindingID:           "fnd_syn_concur_01",
		ActionID:            "act_syn_concur_01",
		OriginalInspector:   "usr_inspector_01",
		AssignedReinspector: "usr_reinspector_02",
		ReinspectorRole:     "INSPECTOR",
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	// Advance version through start
	order, _ = reinspectionEngine.StartReinspection(orderID, "usr_reinspector_02", "INSPECTOR", order.StateVersion)

	// Attempt verification with stale version 1 (current is 2)
	_, err = reinspectionEngine.VerifySatisfactory(
		orderID,
		"usr_reinspector_02",
		"INSPECTOR",
		"Notes",
		nil,
		1, // Stale version
	)
	if !errors.Is(err, workflowaction.ErrConcurrentModification) {
		t.Errorf("expected ErrConcurrentModification on stale version, got: %v", err)
	}

	// Valid version succeeds
	order, err = reinspectionEngine.VerifySatisfactory(
		orderID,
		"usr_reinspector_02",
		"INSPECTOR",
		"Valid notes",
		nil,
		order.StateVersion,
	)
	if err != nil {
		t.Fatalf("verification with valid version failed: %v", err)
	}

	// Closure with stale version fails
	_, err = reinspectionEngine.FinalClose(
		orderID,
		"usr_supervisor_01",
		"SAFETY_SUPERVISOR",
		"Closed",
		false,
		order.StateVersion-1, // Stale version
	)
	if !errors.Is(err, workflowaction.ErrConcurrentModification) {
		t.Errorf("expected ErrConcurrentModification on stale closure version, got: %v", err)
	}
}

func TestReinspection_FailClosed_RejectedEvidenceAndDeficiency(t *testing.T) {
	reinspectionEngine, actionEngine, t0 := setupReinspectionTestEnv()

	actID := "act_syn_defect_01"
	_ = actionEngine.RegisterAction(workflowaction.GovernedAction{
		ActionID:              actID,
		TenantID:              "ten_syn_alpha",
		FindingID:             "fnd_syn_defect_01",
		Title:                 "Seal leaking hydraulic pump valve",
		State:                 workflowaction.ActionStateInReview,
		CurrentOwner:          "usr_owner_01",
		CurrentOwnerRole:      "CAPA_OWNER",
		DueDate:               t0.Add(24 * time.Hour),
		RequiredEvidenceCount: 1,
	})

	act, _ := actionEngine.GetAction(actID)
	act, _ = actionEngine.SubmitEvidence(
		actID,
		workflowaction.GovernedEvidence{
			EvidenceID:  "evd_syn_pump_01",
			Digest:      "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			SubmittedBy: "usr_owner_01",
		},
		act.StateVersion,
	)

	// Reviewer rejects evidence
	_, _ = actionEngine.ReviewEvidence(
		actID,
		"evd_syn_pump_01",
		"usr_reviewer_01",
		false, // rejected
		"Photo is blurry and does not clearly show pump seal serial number",
		act.StateVersion,
	)

	orderID := "rin_syn_defect_01"
	order, _ := reinspectionEngine.CreateReinspectionOrder(workflowaction.ReinspectionOrder{
		ReinspectionID:      orderID,
		TenantID:            "ten_syn_alpha",
		FindingID:           "fnd_syn_defect_01",
		ActionID:            actID,
		OriginalInspector:   "usr_inspector_01",
		AssignedReinspector: "usr_reinspector_02",
		ReinspectorRole:     "INSPECTOR",
	})

	order, _ = reinspectionEngine.StartReinspection(orderID, "usr_reinspector_02", "INSPECTOR", order.StateVersion)

	// 1. VerifySatisfactory fails closed because action has rejected evidence
	_, err := reinspectionEngine.VerifySatisfactory(
		orderID,
		"usr_reinspector_02",
		"INSPECTOR",
		"Attempting to verify with rejected evidence",
		[]string{"evd_syn_pump_01"},
		order.StateVersion,
	)
	if !errors.Is(err, workflowaction.ErrPendingOrRejectedEvidence) {
		t.Errorf("expected ErrPendingOrRejectedEvidence, got: %v", err)
	}

	// 2. Reinspector rejects as deficient with recurrence flag
	order, err = reinspectionEngine.SubmitDeficiencyRejection(
		orderID,
		"usr_reinspector_02",
		"INSPECTOR",
		"Valve continues to weep hydraulic fluid at 15 psi pressure check",
		true,
		"rec_syn_pump_leak_02",
		order.StateVersion,
	)
	if err != nil {
		t.Fatalf("submit deficiency rejection failed: %v", err)
	}
	if order.State != workflowaction.ReinspectionStateRejectedDeficient {
		t.Errorf("expected REJECTED_DEFICIENT, got: %s", order.State)
	}
	if order.RecurrenceID != "rec_syn_pump_leak_02" || order.RecurrenceCount != 1 {
		t.Errorf("recurrence linkage malformed: %+v", order)
	}

	// 3. Attempting final closure while deficient fails closed
	_, err = reinspectionEngine.FinalClose(
		orderID,
		"usr_supervisor_01",
		"SAFETY_SUPERVISOR",
		"Closing deficient action",
		false,
		order.StateVersion,
	)
	if !errors.Is(err, workflowaction.ErrPrerequisitesUnsatisfied) {
		t.Errorf("expected ErrPrerequisitesUnsatisfied on closing deficient order, got: %v", err)
	}
}

func TestReinspection_ReopenLifecycle(t *testing.T) {
	reinspectionEngine, _, _ := setupReinspectionTestEnv()

	orderID := "rin_syn_reopen_01"
	order, _ := reinspectionEngine.CreateReinspectionOrder(workflowaction.ReinspectionOrder{
		ReinspectionID:      orderID,
		TenantID:            "ten_syn_alpha",
		FindingID:           "fnd_syn_reopen_01",
		ActionID:            "act_syn_reopen_01",
		OriginalInspector:   "usr_inspector_01",
		AssignedReinspector: "usr_reinspector_02",
		ReinspectorRole:     "INSPECTOR",
	})
	order, _ = reinspectionEngine.StartReinspection(orderID, "usr_reinspector_02", "INSPECTOR", order.StateVersion)
	order, _ = reinspectionEngine.VerifySatisfactory(orderID, "usr_reinspector_02", "INSPECTOR", "Satisfactory fix", nil, order.StateVersion)
	order, err := reinspectionEngine.FinalClose(orderID, "usr_supervisor_01", "SAFETY_SUPERVISOR", "Closed safely", false, order.StateVersion)
	if err != nil {
		t.Fatalf("initial close failed: %v", err)
	}

	// 1. Unauthorized actor attempts reopen
	_, err = reinspectionEngine.Reopen(orderID, "usr_worker_01", "WORKER", "Unauthorized reopen", order.StateVersion)
	if !errors.Is(err, workflowaction.ErrUnauthorizedClosure) {
		t.Errorf("expected ErrUnauthorizedClosure on reopen by unauthorized role, got: %v", err)
	}

	// 2. Autonomous AI attempts reopen
	_, err = reinspectionEngine.Reopen(orderID, "agent_ai", "AI_AGENT", "AI reopen", order.StateVersion)
	if !errors.Is(err, workflowaction.ErrAutonomousClosureProhibited) {
		t.Errorf("expected ErrAutonomousClosureProhibited on reopen by AI, got: %v", err)
	}

	// 3. Authorized supervisor reopens successfully
	order, err = reinspectionEngine.Reopen(
		orderID,
		"usr_supervisor_01",
		"SAFETY_SUPERVISOR",
		"Post-audit internal review identified secondary crack adjacent to repair weld",
		order.StateVersion,
	)
	if err != nil {
		t.Fatalf("reopen failed: %v", err)
	}

	if order.State != workflowaction.ReinspectionStateReopened {
		t.Errorf("expected REOPENED, got: %s", order.State)
	}
	if order.ClosedAt != nil || order.ClosedBy != "" {
		t.Errorf("expected cleared closure metadata on reopen")
	}

	// Can restart reinspection from REOPENED state
	order, err = reinspectionEngine.StartReinspection(orderID, "usr_reinspector_02", "INSPECTOR", order.StateVersion)
	if err != nil {
		t.Fatalf("restart reinspection after reopen failed: %v", err)
	}
	if order.State != workflowaction.ReinspectionStateInProgress {
		t.Errorf("expected IN_PROGRESS after restart, got %s", order.State)
	}
}

func TestReinspection_FailClosed_SourceRevocation(t *testing.T) {
	reinspectionEngine, _, _ := setupReinspectionTestEnv()

	orderID := "rin_syn_revoke_01"
	order, _ := reinspectionEngine.CreateReinspectionOrder(workflowaction.ReinspectionOrder{
		ReinspectionID:      orderID,
		TenantID:            "ten_syn_alpha",
		FindingID:           "fnd_syn_revoke_01",
		ActionID:            "act_syn_revoke_01",
		OriginalInspector:   "usr_inspector_01",
		AssignedReinspector: "usr_reinspector_02",
		ReinspectorRole:     "INSPECTOR",
	})
	order, _ = reinspectionEngine.StartReinspection(orderID, "usr_reinspector_02", "INSPECTOR", order.StateVersion)
	order, _ = reinspectionEngine.VerifySatisfactory(orderID, "usr_reinspector_02", "INSPECTOR", "Satisfactory", nil, order.StateVersion)

	// Revoke authority of supervisor
	reinspectionEngine.RevokeActorAuthority("usr_supervisor_corrupted")

	// Revoked supervisor attempts final closure -> fails with ErrActorAuthorityRevoked
	_, err := reinspectionEngine.FinalClose(
		orderID,
		"usr_supervisor_corrupted",
		"SAFETY_SUPERVISOR",
		"Closing after revocation",
		false,
		order.StateVersion,
	)
	if !errors.Is(err, workflowaction.ErrActorAuthorityRevoked) {
		t.Errorf("expected ErrActorAuthorityRevoked for revoked supervisor, got: %v", err)
	}

	// Revoke authority of reinspector
	reinspectionEngine.RevokeActorAuthority("usr_reinspector_02")
	_, err = reinspectionEngine.StartReinspection(orderID, "usr_reinspector_02", "INSPECTOR", order.StateVersion)
	if !errors.Is(err, workflowaction.ErrActorAuthorityRevoked) {
		t.Errorf("expected ErrActorAuthorityRevoked for revoked reinspector, got: %v", err)
	}
}

func TestReinspection_InspectorIndependenceEnforcement(t *testing.T) {
	reinspectionEngine, _, _ := setupReinspectionTestEnv()

	// 1. DIFFERENT_INSPECTOR_REQUIRED fails closed if same inspector assigned
	_, err := reinspectionEngine.CreateReinspectionOrder(workflowaction.ReinspectionOrder{
		ReinspectionID:      "rin_syn_diff_fail",
		TenantID:            "ten_syn_alpha",
		FindingID:           "fnd_syn_01",
		ActionID:            "act_syn_01",
		OriginalInspector:   "usr_inspector_alice",
		AssignedReinspector: "usr_inspector_alice", // Same inspector!
		ReinspectorRole:     "INSPECTOR",
		IndependenceRule:    workflowaction.IndependenceDifferentInspectorRequired,
	})
	if !errors.Is(err, workflowaction.ErrInspectorIndependenceViolation) {
		t.Errorf("expected ErrInspectorIndependenceViolation when assigning original inspector, got: %v", err)
	}

	// Succeeded when distinct inspector assigned
	order, err := reinspectionEngine.CreateReinspectionOrder(workflowaction.ReinspectionOrder{
		ReinspectionID:      "rin_syn_diff_ok",
		TenantID:            "ten_syn_alpha",
		FindingID:           "fnd_syn_01",
		ActionID:            "act_syn_01",
		OriginalInspector:   "usr_inspector_alice",
		AssignedReinspector: "usr_inspector_bob",
		ReinspectorRole:     "INSPECTOR",
		IndependenceRule:    workflowaction.IndependenceDifferentInspectorRequired,
	})
	if err != nil {
		t.Fatalf("expected success with distinct inspector, got: %v", err)
	}

	// Attempting reassignment to original inspector fails
	_, err = reinspectionEngine.AssignReinspector(
		"rin_syn_diff_ok",
		"usr_inspector_alice",
		"INSPECTOR",
		"usr_lead_01",
		"SAFETY_LEAD",
		"Reassigning to original inspector",
		order.StateVersion,
	)
	if !errors.Is(err, workflowaction.ErrInspectorIndependenceViolation) {
		t.Errorf("expected ErrInspectorIndependenceViolation on reassignment, got: %v", err)
	}

	// 2. THIRD_PARTY_REQUIRED fails closed if internal inspector role assigned
	_, err = reinspectionEngine.CreateReinspectionOrder(workflowaction.ReinspectionOrder{
		ReinspectionID:      "rin_syn_3p_fail",
		TenantID:            "ten_syn_alpha",
		FindingID:           "fnd_syn_02",
		ActionID:            "act_syn_02",
		OriginalInspector:   "usr_inspector_alice",
		AssignedReinspector: "usr_inspector_bob",
		ReinspectorRole:     "INSPECTOR", // Not THIRD_PARTY_AUDITOR
		IndependenceRule:    workflowaction.IndependenceThirdPartyRequired,
	})
	if !errors.Is(err, workflowaction.ErrInspectorIndependenceViolation) {
		t.Errorf("expected ErrInspectorIndependenceViolation for non-third-party auditor, got: %v", err)
	}

	// Succeeded when role is THIRD_PARTY_AUDITOR
	_, err = reinspectionEngine.CreateReinspectionOrder(workflowaction.ReinspectionOrder{
		ReinspectionID:      "rin_syn_3p_ok",
		TenantID:            "ten_syn_alpha",
		FindingID:           "fnd_syn_02",
		ActionID:            "act_syn_02",
		OriginalInspector:   "usr_inspector_alice",
		AssignedReinspector: "usr_auditor_external",
		ReinspectorRole:     "THIRD_PARTY_AUDITOR",
		IndependenceRule:    workflowaction.IndependenceThirdPartyRequired,
	})
	if err != nil {
		t.Fatalf("expected success for third-party auditor, got: %v", err)
	}
}

func TestReinspection_RecurrenceHandling(t *testing.T) {
	reinspectionEngine, _, _ := setupReinspectionTestEnv()

	orderID := "rin_syn_recur_01"
	order, err := reinspectionEngine.CreateReinspectionOrder(workflowaction.ReinspectionOrder{
		ReinspectionID:      orderID,
		TenantID:            "ten_syn_alpha",
		FindingID:           "fnd_syn_recur_01",
		ActionID:            "act_syn_recur_01",
		OriginalInspector:   "usr_inspector_01",
		AssignedReinspector: "usr_reinspector_02",
		ReinspectorRole:     "INSPECTOR",
		RecurrenceID:        "rec_syn_combustible_dust_01",
		RecurrenceCount:     1,
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	// Link next recurrence in series
	order, err = reinspectionEngine.LinkRecurrence(
		orderID,
		"rec_syn_combustible_dust_02",
		"usr_lead_01",
		"SAFETY_LEAD",
		"Secondary ventilation duct inspection revealed recurrence",
		order.StateVersion,
	)
	if err != nil {
		t.Fatalf("link recurrence failed: %v", err)
	}

	if order.RecurrenceID != "rec_syn_combustible_dust_02" {
		t.Errorf("expected recurrence ID rec_syn_combustible_dust_02, got %s", order.RecurrenceID)
	}
	if order.RecurrenceCount != 2 {
		t.Errorf("expected RecurrenceCount 2, got %d", order.RecurrenceCount)
	}

	// Blank recurrence fails closed
	_, err = reinspectionEngine.LinkRecurrence(
		orderID,
		"",
		"usr_lead_01",
		"SAFETY_LEAD",
		"Blank recurrence attempt",
		order.StateVersion,
	)
	if !errors.Is(err, workflowaction.ErrRecurrenceChainBroken) {
		t.Errorf("expected ErrRecurrenceChainBroken on blank recurrence ID, got: %v", err)
	}
}

func TestReinspection_DoubleClosurePrevention(t *testing.T) {
	reinspectionEngine, _, _ := setupReinspectionTestEnv()

	orderID := "rin_syn_double_close_01"
	order, _ := reinspectionEngine.CreateReinspectionOrder(workflowaction.ReinspectionOrder{
		ReinspectionID:      orderID,
		TenantID:            "ten_syn_alpha",
		FindingID:           "fnd_syn_close_01",
		ActionID:            "act_syn_close_01",
		OriginalInspector:   "usr_inspector_01",
		AssignedReinspector: "usr_reinspector_02",
		ReinspectorRole:     "INSPECTOR",
	})
	order, _ = reinspectionEngine.StartReinspection(orderID, "usr_reinspector_02", "INSPECTOR", order.StateVersion)
	order, _ = reinspectionEngine.VerifySatisfactory(orderID, "usr_reinspector_02", "INSPECTOR", "Satisfactory", nil, order.StateVersion)
	order, err := reinspectionEngine.FinalClose(orderID, "usr_supervisor_01", "SAFETY_SUPERVISOR", "Closed safely", false, order.StateVersion)
	if err != nil {
		t.Fatalf("initial close failed: %v", err)
	}

	// Second closure attempt must fail closed with ErrReinspectionTerminalState
	_, err = reinspectionEngine.FinalClose(
		orderID,
		"usr_supervisor_01",
		"SAFETY_SUPERVISOR",
		"Double-closure attempt",
		false,
		order.StateVersion,
	)
	if !errors.Is(err, workflowaction.ErrReinspectionTerminalState) {
		t.Errorf("expected ErrReinspectionTerminalState on double closure, got: %v", err)
	}
}
