package workflowaction_test

import (
	"errors"
	"testing"

	workflowaction "github.com/oshethai/oshe-platform/modules/workflow-action"
)

func TestFindingLifecycle_DeterministicCreation_Valid(t *testing.T) {
	catalog := workflowaction.NewFindingRuleCatalog()
	mgr := workflowaction.NewFindingManager(catalog)

	req := workflowaction.FindingCreationRequest{
		TenantID:         "ten_syn_01",
		FindingID:        "fnd_syn_fire_exit_01",
		ExecutionID:      "ins_syn_clean_01",
		QuestionID:       "qst_fire_exit_01",
		ResponseID:       "rsp_syn_blocked_exit_01",
		RuleID:           "RULE-FND-FIRE-EXIT-BLOCKED",
		RuleVersion:      workflowaction.CurrentRuleMatrixVersion,
		Title:            "Blocked Fire Exit Corridor",
		Description:      "Wooden pallets obstruct doorway B2 in manufacturing sector.",
		SeverityInput:    workflowaction.SeverityCritical,
		CriticalFlag:     true,
		ImmediateControl: "Pallets cordoned off with yellow tape, warning sign placed, facility team dispatched to clear.",
		Actor:            "usr_syn_inspector_alice",
		ActorRole:        "INSPECTOR",
		EvidenceIDs:      []string{"evd_syn_orig_photo_01"},
	}

	finding, err := mgr.CreateFinding(req)
	if err != nil {
		t.Fatalf("CreateFinding failed: %v", err)
	}

	if finding.FindingID != "fnd_syn_fire_exit_01" {
		t.Errorf("expected finding ID fnd_syn_fire_exit_01, got %q", finding.FindingID)
	}
	if finding.Severity != workflowaction.SeverityCritical {
		t.Errorf("expected SeverityCritical, got %q", finding.Severity)
	}
	if !finding.CriticalFlag {
		t.Errorf("expected CriticalFlag=true")
	}
	if finding.State != workflowaction.FindingStateOpen {
		t.Errorf("expected FindingStateOpen, got %q", finding.State)
	}
	if len(finding.History) != 1 {
		t.Errorf("expected 1 audit entry, got %d", len(finding.History))
	}
}

func TestFindingLifecycle_FailClosed_UnregisteredRule(t *testing.T) {
	catalog := workflowaction.NewFindingRuleCatalog()
	mgr := workflowaction.NewFindingManager(catalog)

	req := workflowaction.FindingCreationRequest{
		TenantID:         "ten_syn_01",
		FindingID:        "fnd_syn_invalid_rule_01",
		ExecutionID:      "ins_syn_clean_01",
		QuestionID:       "qst_unregistered_01",
		ResponseID:       "rsp_syn_01",
		RuleID:           "RULE-DOES-NOT-EXIST",
		RuleVersion:      workflowaction.CurrentRuleMatrixVersion,
		Title:            "Invalid Rule Finding",
		Description:      "Attempt to create finding with unapproved rule.",
		SeverityInput:    workflowaction.SeverityLow,
		ImmediateControl: "None",
		Actor:            "usr_syn_inspector_bob",
		ActorRole:        "INSPECTOR",
	}

	_, err := mgr.CreateFinding(req)
	if !errors.Is(err, workflowaction.ErrRuleNotFound) {
		t.Errorf("expected ErrRuleNotFound for unregistered rule, got %v", err)
	}
}

func TestFindingLifecycle_FailClosed_StaleRuleVersion(t *testing.T) {
	catalog := workflowaction.NewFindingRuleCatalog()
	mgr := workflowaction.NewFindingManager(catalog)

	req := workflowaction.FindingCreationRequest{
		TenantID:         "ten_syn_01",
		FindingID:        "fnd_syn_stale_ver_01",
		ExecutionID:      "ins_syn_clean_01",
		QuestionID:       "qst_extinguisher_01",
		ResponseID:       "rsp_syn_01",
		RuleID:           "RULE-FND-EXTINGUISHER-DEFECT",
		RuleVersion:      "0.9.0-DEPRECATED",
		Title:            "Stale Version Finding",
		Description:      "Attempt to evaluate rule with non-pinned version.",
		SeverityInput:    workflowaction.SeverityHigh,
		ImmediateControl: "Tag placed on unit",
		Actor:            "usr_syn_inspector_bob",
		ActorRole:        "INSPECTOR",
	}

	_, err := mgr.CreateFinding(req)
	if !errors.Is(err, workflowaction.ErrIncompatibleRuleVersion) {
		t.Errorf("expected ErrIncompatibleRuleVersion, got %v", err)
	}
}

func TestFindingLifecycle_FailClosed_MissingSourceContext(t *testing.T) {
	catalog := workflowaction.NewFindingRuleCatalog()
	mgr := workflowaction.NewFindingManager(catalog)

	req := workflowaction.FindingCreationRequest{
		TenantID:         "ten_syn_01",
		FindingID:        "fnd_syn_no_source_01",
		ExecutionID:      "", // Missing execution
		QuestionID:       "qst_extinguisher_01",
		ResponseID:       "rsp_syn_01",
		RuleID:           "RULE-FND-EXTINGUISHER-DEFECT",
		RuleVersion:      workflowaction.CurrentRuleMatrixVersion,
		Title:            "Missing Source Finding",
		SeverityInput:    workflowaction.SeverityHigh,
		ImmediateControl: "Tag placed",
		Actor:            "usr_syn_inspector_alice",
		ActorRole:        "INSPECTOR",
	}

	_, err := mgr.CreateFinding(req)
	if !errors.Is(err, workflowaction.ErrMissingSourceContext) {
		t.Errorf("expected ErrMissingSourceContext for empty execution_id, got %v", err)
	}
}

func TestFindingLifecycle_FailClosed_MissingImmediateControlForCritical(t *testing.T) {
	catalog := workflowaction.NewFindingRuleCatalog()
	mgr := workflowaction.NewFindingManager(catalog)

	req := workflowaction.FindingCreationRequest{
		TenantID:         "ten_syn_01",
		FindingID:        "fnd_syn_no_control_01",
		ExecutionID:      "ins_syn_clean_01",
		QuestionID:       "qst_fire_exit_01",
		ResponseID:       "rsp_syn_01",
		RuleID:           "RULE-FND-FIRE-EXIT-BLOCKED",
		RuleVersion:      workflowaction.CurrentRuleMatrixVersion,
		Title:            "Critical Finding Without Immediate Control",
		Description:      "Blocked emergency exit doorway.",
		SeverityInput:    workflowaction.SeverityCritical,
		CriticalFlag:     true,
		ImmediateControl: "", // Empty immediate control
		Actor:            "usr_syn_inspector_alice",
		ActorRole:        "INSPECTOR",
	}

	_, err := mgr.CreateFinding(req)
	if !errors.Is(err, workflowaction.ErrMissingImmediateControl) {
		t.Errorf("expected ErrMissingImmediateControl, got %v", err)
	}
}

func TestFindingLifecycle_RecurrenceLinking(t *testing.T) {
	catalog := workflowaction.NewFindingRuleCatalog()
	mgr := workflowaction.NewFindingManager(catalog)

	// 1. Create primary finding
	req1 := workflowaction.FindingCreationRequest{
		TenantID:         "ten_syn_01",
		FindingID:        "fnd_syn_primary_01",
		ExecutionID:      "ins_syn_week1",
		QuestionID:       "qst_ppe_01",
		ResponseID:       "rsp_syn_01",
		RuleID:           "RULE-FND-PPE-NONCOMPLIANCE",
		RuleVersion:      workflowaction.CurrentRuleMatrixVersion,
		Title:            "Missing Safety Glasses in Grind Shop",
		SeverityInput:    workflowaction.SeverityMedium,
		ImmediateControl: "Worker issued spare eyewear immediately",
		Actor:            "usr_syn_inspector_alice",
		ActorRole:        "INSPECTOR",
	}
	f1, err := mgr.CreateFinding(req1)
	if err != nil {
		t.Fatalf("CreateFinding 1 failed: %v", err)
	}

	// 2. Create recurring finding pointing to f1
	req2 := workflowaction.FindingCreationRequest{
		TenantID:         "ten_syn_01",
		FindingID:        "fnd_syn_recurrence_02",
		ExecutionID:      "ins_syn_week2",
		QuestionID:       "qst_ppe_01",
		ResponseID:       "rsp_syn_02",
		RecurrenceID:     f1.FindingID,
		RuleID:           "RULE-FND-PPE-NONCOMPLIANCE",
		RuleVersion:      workflowaction.CurrentRuleMatrixVersion,
		Title:            "Recurrence: Missing Safety Glasses in Grind Shop",
		SeverityInput:    workflowaction.SeverityMedium,
		ImmediateControl: "Eyewear issued and verbal warning given",
		Actor:            "usr_syn_inspector_bob",
		ActorRole:        "INSPECTOR",
	}
	f2, err := mgr.CreateFinding(req2)
	if err != nil {
		t.Fatalf("CreateFinding 2 with recurrence failed: %v", err)
	}
	if f2.RecurrenceID != f1.FindingID {
		t.Errorf("expected recurrence_id %q, got %q", f1.FindingID, f2.RecurrenceID)
	}

	// 3. Fail closed on invalid recurrence link
	req3 := req2
	req3.FindingID = "fnd_syn_bad_recurrence_03"
	req3.RecurrenceID = "fnd_nonexistent_99"
	_, err = mgr.CreateFinding(req3)
	if !errors.Is(err, workflowaction.ErrInvalidRecurrenceLink) {
		t.Errorf("expected ErrInvalidRecurrenceLink, got %v", err)
	}
}

func TestFindingLifecycle_SeverityEscalationAndDowngrade(t *testing.T) {
	catalog := workflowaction.NewFindingRuleCatalog()
	mgr := workflowaction.NewFindingManager(catalog)

	req := workflowaction.FindingCreationRequest{
		TenantID:         "ten_syn_01",
		FindingID:        "fnd_syn_severity_test_01",
		ExecutionID:      "ins_syn_clean_01",
		QuestionID:       "qst_extinguisher_01",
		ResponseID:       "rsp_syn_01",
		RuleID:           "RULE-FND-EXTINGUISHER-DEFECT",
		RuleVersion:      workflowaction.CurrentRuleMatrixVersion,
		Title:            "Extinguisher Pressure Low",
		SeverityInput:    workflowaction.SeverityMedium,
		ImmediateControl: "Unit tagged for replacement",
		Actor:            "usr_syn_inspector_alice",
		ActorRole:        "INSPECTOR",
	}
	_, err := mgr.CreateFinding(req)
	if err != nil {
		t.Fatalf("CreateFinding failed: %v", err)
	}

	// 1. Escalation: Medium -> High by Inspector is permitted
	err = mgr.UpdateSeverity("ten_syn_01", "fnd_syn_severity_test_01", workflowaction.SeverityHigh, "usr_syn_inspector_alice", "INSPECTOR", "Re-evaluated hazard proximity")
	if err != nil {
		t.Fatalf("escalation failed: %v", err)
	}

	// 2. Downgrade: High -> Medium by ordinary Inspector is denied fail-closed
	err = mgr.UpdateSeverity("ten_syn_01", "fnd_syn_severity_test_01", workflowaction.SeverityMedium, "usr_syn_inspector_alice", "INSPECTOR", "Downgrading hazard")
	if !errors.Is(err, workflowaction.ErrUnauthorizedSeverityDowngrade) {
		t.Errorf("expected ErrUnauthorizedSeverityDowngrade for non-supervisor actor, got %v", err)
	}

	// 3. Downgrade by Supervisor without rationale is denied
	err = mgr.UpdateSeverity("ten_syn_01", "fnd_syn_severity_test_01", workflowaction.SeverityMedium, "usr_syn_super_dan", "SAFETY_SUPERVISOR", "")
	if !errors.Is(err, workflowaction.ErrMissingClassificationRationale) {
		t.Errorf("expected ErrMissingClassificationRationale for empty rationale, got %v", err)
	}

	// 4. Downgrade by Supervisor with rationale succeeds
	err = mgr.UpdateSeverity("ten_syn_01", "fnd_syn_severity_test_01", workflowaction.SeverityMedium, "usr_syn_super_dan", "SAFETY_SUPERVISOR", "Secondary extinguisher available in immediate 5m radius")
	if err != nil {
		t.Fatalf("authorized downgrade failed: %v", err)
	}

	f, _ := mgr.GetFinding("ten_syn_01", "fnd_syn_severity_test_01")
	if f.Severity != workflowaction.SeverityMedium {
		t.Errorf("expected SeverityMedium, got %s", f.Severity)
	}
}

func TestFindingLifecycle_AutonomousProhibitionsAndClosure(t *testing.T) {
	catalog := workflowaction.NewFindingRuleCatalog()
	mgr := workflowaction.NewFindingManager(catalog)

	req := workflowaction.FindingCreationRequest{
		TenantID:         "ten_syn_01",
		FindingID:        "fnd_syn_closure_test_01",
		ExecutionID:      "ins_syn_clean_01",
		QuestionID:       "qst_ppe_01",
		ResponseID:       "rsp_syn_01",
		RuleID:           "RULE-FND-PPE-NONCOMPLIANCE",
		RuleVersion:      workflowaction.CurrentRuleMatrixVersion,
		Title:            "PPE Deficiency",
		SeverityInput:    workflowaction.SeverityLow,
		ImmediateControl: "Corrected on spot",
		Actor:            "usr_syn_inspector_alice",
		ActorRole:        "INSPECTOR",
	}
	_, err := mgr.CreateFinding(req)
	if err != nil {
		t.Fatalf("CreateFinding failed: %v", err)
	}

	// 1. Prohibit AI autonomous severity change
	err = mgr.UpdateSeverity("ten_syn_01", "fnd_syn_closure_test_01", workflowaction.SeverityHigh, "AI_AGENT_AUTONOMOUS", "AI_CORE", "LLM determined high risk")
	if !errors.Is(err, workflowaction.ErrAutonomousClassificationProhibited) {
		t.Errorf("expected ErrAutonomousClassificationProhibited, got %v", err)
	}

	// 2. Prohibit AI autonomous closure
	err = mgr.TransitionState("ten_syn_01", "fnd_syn_closure_test_01", workflowaction.FindingStateClosed, "AI_AGENT_CORE", "AI_CORE", "Closed automatically by AI rule engine")
	if !errors.Is(err, workflowaction.ErrAutonomousClosureProhibited) {
		t.Errorf("expected ErrAutonomousClosureProhibited, got %v", err)
	}

	// 3. Prohibit unauthorized role from closing finding
	err = mgr.TransitionState("ten_syn_01", "fnd_syn_closure_test_01", workflowaction.FindingStateClosed, "usr_syn_worker_charlie", "LINE_WORKER", "Fixed issue myself")
	if !errors.Is(err, workflowaction.ErrUnauthorizedClosure) {
		t.Errorf("expected ErrUnauthorizedClosure, got %v", err)
	}

	// 4. Authorized human supervisor closure succeeds
	err = mgr.TransitionState("ten_syn_01", "fnd_syn_closure_test_01", workflowaction.FindingStateClosed, "usr_syn_super_dan", "SAFETY_SUPERVISOR", "Remediation verified; safety briefing conducted.")
	if err != nil {
		t.Fatalf("authorized closure failed: %v", err)
	}

	f, _ := mgr.GetFinding("ten_syn_01", "fnd_syn_closure_test_01")
	if f.State != workflowaction.FindingStateClosed {
		t.Errorf("expected FindingStateClosed, got %s", f.State)
	}
	if f.ClosedBy != "usr_syn_super_dan" {
		t.Errorf("expected ClosedBy usr_syn_super_dan, got %s", f.ClosedBy)
	}

	// 5. Mutating closed finding fails closed
	err = mgr.UpdateSeverity("ten_syn_01", "fnd_syn_closure_test_01", workflowaction.SeverityHigh, "usr_syn_super_dan", "SAFETY_SUPERVISOR", "reopening")
	if !errors.Is(err, workflowaction.ErrFindingAlreadyClosed) {
		t.Errorf("expected ErrFindingAlreadyClosed on closed mutation, got %v", err)
	}
}
