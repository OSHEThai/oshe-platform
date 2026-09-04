package workflowaction_test

import (
	"errors"
	"testing"

	workflowaction "github.com/oshethai/oshe-platform/modules/workflow-action"
)

func TestQualification_RequiredResponse(t *testing.T) {
	matrix := workflowaction.DefaultRuleMatrix()
	catalog := workflowaction.DefaultTransitionCatalog()

	// 1. Template transition qualification: Draft -> InReview
	tmplReq := workflowaction.TransitionQualificationRequest{
		RuleID:          "RULE-TMPL-SUBMIT",
		TargetKind:      workflowaction.TargetKindTemplate,
		TargetID:        "tmpl-checklist-001",
		TenantID:        "ten-alpha",
		FromState:       "Draft",
		ToState:         "InReview",
		Actor:           "author-user",
		ActorRole:       "AUTHOR",
		InputConditions: map[string]bool{"questions_non_empty": true},
		MatrixVersion:   workflowaction.CurrentRuleMatrixVersion,
	}

	tmplRes := catalog.Qualify(matrix, tmplReq)
	if !tmplRes.Allowed {
		t.Fatalf("expected template qualification to be allowed, got denied: %s (%s)", tmplRes.DenialReason, tmplRes.Explanation)
	}
	if tmplRes.DenialReason != workflowaction.DenialNone {
		t.Errorf("expected DenialNone, got %s", tmplRes.DenialReason)
	}
	if tmplRes.TraceabilityKey != "OSHE-BR-TMPL-01" {
		t.Errorf("expected traceability OSHE-BR-TMPL-01, got %s", tmplRes.TraceabilityKey)
	}
	if tmplRes.Explanation == "" {
		t.Error("expected non-empty explanation on allowed transition")
	}

	// 2. Action transition qualification: IN_REVIEW -> CLOSED
	actReq := workflowaction.TransitionQualificationRequest{
		RuleID:          "RULE-ACT-CLOSE",
		TargetKind:      workflowaction.TargetKindAction,
		TargetID:        "act-hazard-001",
		TenantID:        "ten-alpha",
		FromState:       string(workflowaction.ActionStateInReview),
		ToState:         string(workflowaction.ActionStateClosed),
		Actor:           "reviewer-user",
		ActorRole:       "REVIEWER",
		EvidenceIDs:     []string{"ev-doc-001"},
		InputConditions: map[string]bool{"inspection_verified": true},
		MatrixVersion:   workflowaction.CurrentRuleMatrixVersion,
	}

	actRes := catalog.Qualify(matrix, actReq)
	if !actRes.Allowed {
		t.Fatalf("expected action closure qualification to be allowed, got denied: %s (%s)", actRes.DenialReason, actRes.Explanation)
	}
	if actRes.DenialReason != workflowaction.DenialNone {
		t.Errorf("expected DenialNone, got %s", actRes.DenialReason)
	}
	if actRes.TraceabilityKey != "OSHE-BR-ACT-04" {
		t.Errorf("expected traceability OSHE-BR-ACT-04, got %s", actRes.TraceabilityKey)
	}
}

func TestQualification_InvalidOrMissingCondition(t *testing.T) {
	matrix := workflowaction.DefaultRuleMatrix()
	catalog := workflowaction.DefaultTransitionCatalog()

	// Missing condition "inspection_verified"
	req := workflowaction.TransitionQualificationRequest{
		RuleID:          "RULE-ACT-CLOSE",
		TargetKind:      workflowaction.TargetKindAction,
		TargetID:        "act-001",
		TenantID:        "ten-alpha",
		FromState:       string(workflowaction.ActionStateInReview),
		ToState:         string(workflowaction.ActionStateClosed),
		Actor:           "reviewer-user",
		ActorRole:       "REVIEWER",
		EvidenceIDs:     []string{"ev-001"},
		InputConditions: map[string]bool{}, // missing inspection_verified
		MatrixVersion:   workflowaction.CurrentRuleMatrixVersion,
	}

	res := catalog.Qualify(matrix, req)
	if res.Allowed {
		t.Fatal("expected qualification denial for missing condition")
	}
	if res.DenialReason != workflowaction.DenialMissingCondition {
		t.Errorf("expected DenialMissingCondition, got %s", res.DenialReason)
	}

	// Condition explicitly false
	req.InputConditions = map[string]bool{"inspection_verified": false}
	resFalse := catalog.Qualify(matrix, req)
	if resFalse.Allowed || resFalse.DenialReason != workflowaction.DenialMissingCondition {
		t.Errorf("expected DenialMissingCondition for false condition, got %s", resFalse.DenialReason)
	}
}

func TestQualification_TransitionValidation(t *testing.T) {
	matrix := workflowaction.DefaultRuleMatrix()
	catalog := workflowaction.DefaultTransitionCatalog()

	// Unregistered transition jump: DRAFT -> CLOSED directly
	req := workflowaction.TransitionQualificationRequest{
		TargetKind:      workflowaction.TargetKindWorkflow,
		TargetID:        "wf-001",
		TenantID:        "ten-alpha",
		FromState:       string(workflowaction.StateDraft),
		ToState:         string(workflowaction.StateClosed), // invalid transition jump!
		Actor:           "supervisor-user",
		ActorRole:       "SUPERVISOR",
		MatrixVersion:   workflowaction.CurrentRuleMatrixVersion,
	}

	res := catalog.Qualify(matrix, req)
	if res.Allowed {
		t.Fatal("expected qualification denial for invalid transition jump")
	}
	if res.DenialReason != workflowaction.DenialInvalidTransition {
		t.Errorf("expected DenialInvalidTransition, got %s", res.DenialReason)
	}

	// Wrong rule specified for transition
	reqMismatch := workflowaction.TransitionQualificationRequest{
		RuleID:          "RULE-WF-CLOSE", // wrong rule for Draft -> InProgress
		TargetKind:      workflowaction.TargetKindWorkflow,
		TargetID:        "wf-001",
		TenantID:        "ten-alpha",
		FromState:       string(workflowaction.StateDraft),
		ToState:         string(workflowaction.StateInProgress),
		Actor:           "operator-user",
		ActorRole:       "OPERATOR",
		MatrixVersion:   workflowaction.CurrentRuleMatrixVersion,
	}

	resMismatch := catalog.Qualify(matrix, reqMismatch)
	if resMismatch.Allowed || resMismatch.DenialReason != workflowaction.DenialInvalidTransition {
		t.Errorf("expected DenialInvalidTransition for mismatched rule, got %s", resMismatch.DenialReason)
	}
}

func TestQualification_AuthorizationFailure(t *testing.T) {
	matrix := workflowaction.DefaultRuleMatrix()
	catalog := workflowaction.DefaultTransitionCatalog()

	// Actor role CONTRACTOR attempts RULE-ACT-CLOSE which requires REVIEWER
	req := workflowaction.TransitionQualificationRequest{
		RuleID:          "RULE-ACT-CLOSE",
		TargetKind:      workflowaction.TargetKindAction,
		TargetID:        "act-001",
		TenantID:        "ten-alpha",
		FromState:       string(workflowaction.ActionStateInReview),
		ToState:         string(workflowaction.ActionStateClosed),
		Actor:           "contractor-user",
		ActorRole:       "CONTRACTOR", // unauthorized role
		EvidenceIDs:     []string{"ev-001"},
		InputConditions: map[string]bool{"inspection_verified": true},
		MatrixVersion:   workflowaction.CurrentRuleMatrixVersion,
	}

	res := catalog.Qualify(matrix, req)
	if res.Allowed {
		t.Fatal("expected authorization denial for contractor attempting closure")
	}
	if res.DenialReason != workflowaction.DenialUnauthorizedActor {
		t.Errorf("expected DenialUnauthorizedActor, got %s", res.DenialReason)
	}
}

func TestQualification_MissingEvidence(t *testing.T) {
	matrix := workflowaction.DefaultRuleMatrix()
	catalog := workflowaction.DefaultTransitionCatalog()

	// RULE-ACT-SUBMIT requires minimum 1 evidence
	req := workflowaction.TransitionQualificationRequest{
		RuleID:          "RULE-ACT-SUBMIT",
		TargetKind:      workflowaction.TargetKindAction,
		TargetID:        "act-001",
		TenantID:        "ten-alpha",
		FromState:       string(workflowaction.ActionStateInProgress),
		ToState:         string(workflowaction.ActionStateInReview),
		Actor:           "owner-user",
		ActorRole:       "OWNER",
		EvidenceIDs:     []string{}, // zero evidence provided!
		InputConditions: map[string]bool{"remediation_complete": true},
		MatrixVersion:   workflowaction.CurrentRuleMatrixVersion,
	}

	res := catalog.Qualify(matrix, req)
	if res.Allowed {
		t.Fatal("expected qualification denial for missing evidence")
	}
	if res.DenialReason != workflowaction.DenialMissingEvidence {
		t.Errorf("expected DenialMissingEvidence, got %s", res.DenialReason)
	}
}

func TestQualification_ArchivedTargetDenial(t *testing.T) {
	matrix := workflowaction.DefaultRuleMatrix()
	catalog := workflowaction.DefaultTransitionCatalog()

	// Target flagged IsArchived
	reqArchived := workflowaction.TransitionQualificationRequest{
		RuleID:          "RULE-WF-START",
		TargetKind:      workflowaction.TargetKindWorkflow,
		TargetID:        "wf-001",
		TenantID:        "ten-alpha",
		FromState:       string(workflowaction.StateDraft),
		ToState:         string(workflowaction.StateInProgress),
		Actor:           "operator-user",
		ActorRole:       "OPERATOR",
		IsArchived:      true, // archived target
		MatrixVersion:   workflowaction.CurrentRuleMatrixVersion,
	}

	resArchived := catalog.Qualify(matrix, reqArchived)
	if resArchived.Allowed {
		t.Fatal("expected qualification denial on archived target")
	}
	if resArchived.DenialReason != workflowaction.DenialArchivedTarget {
		t.Errorf("expected DenialArchivedTarget, got %s", resArchived.DenialReason)
	}

	// FromState is ARCHIVED
	reqFromArchived := reqArchived
	reqFromArchived.IsArchived = false
	reqFromArchived.FromState = string(workflowaction.StateArchived)

	resFromArchived := catalog.Qualify(matrix, reqFromArchived)
	if resFromArchived.Allowed || resFromArchived.DenialReason != workflowaction.DenialArchivedTarget {
		t.Errorf("expected DenialArchivedTarget when FromState is ARCHIVED, got %s", resFromArchived.DenialReason)
	}

	// FromState is Retired (template)
	reqRetired := reqArchived
	reqRetired.IsArchived = false
	reqRetired.TargetKind = workflowaction.TargetKindTemplate
	reqRetired.FromState = "Retired"
	resRetired := catalog.Qualify(matrix, reqRetired)
	if resRetired.Allowed || resRetired.DenialReason != workflowaction.DenialArchivedTarget {
		t.Errorf("expected DenialArchivedTarget when FromState is Retired, got %s", resRetired.DenialReason)
	}
}

func TestQualification_VersionCompatibility(t *testing.T) {
	matrix := workflowaction.DefaultRuleMatrix()
	catalog := workflowaction.DefaultTransitionCatalog()

	// Incompatible version in request
	req := workflowaction.TransitionQualificationRequest{
		RuleID:          "RULE-WF-START",
		TargetKind:      workflowaction.TargetKindWorkflow,
		TargetID:        "wf-001",
		TenantID:        "ten-alpha",
		FromState:       string(workflowaction.StateDraft),
		ToState:         string(workflowaction.StateInProgress),
		Actor:           "operator-user",
		ActorRole:       "OPERATOR",
		MatrixVersion:   "2.0.0", // incompatible!
	}

	res := catalog.Qualify(matrix, req)
	if res.Allowed {
		t.Fatal("expected qualification denial for incompatible matrix version")
	}
	if res.DenialReason != workflowaction.DenialIncompatibleVersion {
		t.Errorf("expected DenialIncompatibleVersion, got %s", res.DenialReason)
	}

	// Incompatible version in matrix instance
	badMatrix := workflowaction.NewRuleMatrix("0.9.0")
	_ = badMatrix.RegisterRule(workflowaction.BusinessRule{
		RuleID:    "RULE-WF-START",
		OwnerRole: "OPERATOR",
	})
	req.MatrixVersion = "0.9.0"
	resBadMatrix := catalog.Qualify(badMatrix, req)
	if resBadMatrix.Allowed || resBadMatrix.DenialReason != workflowaction.DenialIncompatibleVersion {
		t.Errorf("expected DenialIncompatibleVersion for bad matrix version, got %s", resBadMatrix.DenialReason)
	}
}

func TestQualification_UnregisteredRule(t *testing.T) {
	matrix := workflowaction.DefaultRuleMatrix()
	catalog := workflowaction.DefaultTransitionCatalog()

	req := workflowaction.TransitionQualificationRequest{
		RuleID:        "RULE-UNKNOWN-999",
		TargetKind:    workflowaction.TargetKindWorkflow,
		TargetID:      "wf-001",
		TenantID:      "ten-alpha",
		FromState:     string(workflowaction.StateDraft),
		ToState:       string(workflowaction.StateInProgress),
		Actor:         "operator-user",
		ActorRole:     "OPERATOR",
		MatrixVersion: workflowaction.CurrentRuleMatrixVersion,
	}

	res := catalog.Qualify(matrix, req)
	if res.Allowed {
		t.Fatal("expected qualification denial for unregistered rule")
	}
	if res.DenialReason != workflowaction.DenialUnregisteredRule {
		t.Errorf("expected DenialUnregisteredRule, got %s", res.DenialReason)
	}
}

func TestQualification_BlankIdentifiers(t *testing.T) {
	matrix := workflowaction.DefaultRuleMatrix()
	catalog := workflowaction.DefaultTransitionCatalog()

	baseReq := workflowaction.TransitionQualificationRequest{
		RuleID:        "RULE-WF-START",
		TargetKind:    workflowaction.TargetKindWorkflow,
		TargetID:      "wf-001",
		TenantID:      "ten-alpha",
		FromState:     string(workflowaction.StateDraft),
		ToState:       string(workflowaction.StateInProgress),
		Actor:         "operator-user",
		ActorRole:     "OPERATOR",
		MatrixVersion: workflowaction.CurrentRuleMatrixVersion,
	}

	// Blank TargetID
	blankTarget := baseReq
	blankTarget.TargetID = "  "
	res := catalog.Qualify(matrix, blankTarget)
	if res.Allowed || res.DenialReason != workflowaction.DenialBlankIdentifier {
		t.Errorf("expected DenialBlankIdentifier for blank TargetID, got %s", res.DenialReason)
	}

	// Blank TenantID
	blankTenant := baseReq
	blankTenant.TenantID = ""
	res = catalog.Qualify(matrix, blankTenant)
	if res.Allowed || res.DenialReason != workflowaction.DenialBlankIdentifier {
		t.Errorf("expected DenialBlankIdentifier for blank TenantID, got %s", res.DenialReason)
	}

	// Blank Actor
	blankActor := baseReq
	blankActor.Actor = ""
	res = catalog.Qualify(matrix, blankActor)
	if res.Allowed || res.DenialReason != workflowaction.DenialBlankIdentifier {
		t.Errorf("expected DenialBlankIdentifier for blank Actor, got %s", res.DenialReason)
	}
}

func TestQualification_RollbackAndNonMutationOnDenial(t *testing.T) {
	matrix := workflowaction.DefaultRuleMatrix()
	catalog := workflowaction.DefaultTransitionCatalog()
	engine := workflowaction.NewEngine(nil)

	// Create a workflow in StateDraft with Revision 1
	inst, err := engine.CreateWorkflow("wf-qual-test")
	if err != nil {
		t.Fatalf("failed to create workflow: %v", err)
	}
	if inst.CurrentState != workflowaction.StateDraft || inst.Revision != 1 {
		t.Fatalf("unexpected initial state: %+v", inst)
	}

	// Set up an AuthPredicate driven by business rule qualification
	authorizer := func(workflowID string, from, to workflowaction.State) bool {
		qualReq := workflowaction.TransitionQualificationRequest{
			TargetKind:      workflowaction.TargetKindWorkflow,
			TargetID:        workflowID,
			TenantID:        "ten-alpha",
			FromState:       string(from),
			ToState:         string(to),
			Actor:           "unauthorized-actor",
			ActorRole:       "VIEWER", // unauthorized for RULE-WF-START (requires OPERATOR)
			MatrixVersion:   workflowaction.CurrentRuleMatrixVersion,
		}
		result := catalog.Qualify(matrix, qualReq)
		return result.Allowed
	}

	// Attempt transition
	_, err = engine.Transition(workflowaction.TransitionRequest{
		WorkflowID:       "wf-qual-test",
		RequestID:        "req-001",
		CorrelationID:    "corr-001",
		SourceState:      workflowaction.StateDraft,
		TargetState:      workflowaction.StateInProgress,
		ExpectedRevision: 1,
		Authorizer:       authorizer,
	})

	if !errors.Is(err, workflowaction.ErrAuthorizationDenied) {
		t.Fatalf("expected ErrAuthorizationDenied, got %v", err)
	}

	// Verify non-mutation: state must remain StateDraft, revision 1, zero audit entries
	verifyInst, err := engine.GetWorkflow("wf-qual-test")
	if err != nil {
		t.Fatalf("failed to retrieve workflow: %v", err)
	}
	if verifyInst.CurrentState != workflowaction.StateDraft {
		t.Errorf("expected state to remain StateDraft on qualification denial, got %s", verifyInst.CurrentState)
	}
	if verifyInst.Revision != 1 {
		t.Errorf("expected revision to remain 1, got %d", verifyInst.Revision)
	}

	history := engine.AuditHistory("wf-qual-test")
	if len(history) != 0 {
		t.Errorf("expected 0 audit log entries after denied transition, got %d", len(history))
	}
}

func TestQualification_RuleTraceability(t *testing.T) {
	matrix := workflowaction.DefaultRuleMatrix()
	catalog := workflowaction.DefaultTransitionCatalog()

	expectedTraceability := map[string]string{
		"RULE-TMPL-SUBMIT":  "OSHE-BR-TMPL-01",
		"RULE-TMPL-APPROVE": "OSHE-BR-TMPL-02",
		"RULE-TMPL-PUBLISH": "OSHE-BR-TMPL-03",
		"RULE-TMPL-RETIRE":  "OSHE-BR-TMPL-04",
		"RULE-WF-START":     "OSHE-BR-WF-01",
		"RULE-WF-SUBMIT":    "OSHE-BR-WF-02",
		"RULE-WF-APPROVE":   "OSHE-BR-WF-03",
		"RULE-WF-CLOSE":     "OSHE-BR-WF-04",
		"RULE-ACT-START":    "OSHE-BR-ACT-01",
		"RULE-ACT-SUBMIT":   "OSHE-BR-ACT-02",
		"RULE-ACT-REJECT":   "OSHE-BR-ACT-03",
		"RULE-ACT-CLOSE":    "OSHE-BR-ACT-04",
		"RULE-ACT-REOPEN":   "OSHE-BR-ACT-05",
	}

	for ruleID, expectedKey := range expectedTraceability {
		rule, err := matrix.GetRule(ruleID)
		if err != nil {
			t.Errorf("failed to retrieve rule %s: %v", ruleID, err)
			continue
		}
		if rule.TraceabilityKey != expectedKey {
			t.Errorf("rule %s traceability mismatch: expected %s, got %s", ruleID, expectedKey, rule.TraceabilityKey)
		}
	}

	// Verify Qualify returns correct TraceabilityKey on denial where rule is matched
	deniedReq := workflowaction.TransitionQualificationRequest{
		RuleID:          "RULE-ACT-CLOSE",
		TargetKind:      workflowaction.TargetKindAction,
		TargetID:        "act-001",
		TenantID:        "ten-alpha",
		FromState:       string(workflowaction.ActionStateInReview),
		ToState:         string(workflowaction.ActionStateClosed),
		Actor:           "someone",
		ActorRole:       "CONTRACTOR",
		MatrixVersion:   workflowaction.CurrentRuleMatrixVersion,
	}
	deniedRes := catalog.Qualify(matrix, deniedReq)
	if deniedRes.TraceabilityKey != "OSHE-BR-ACT-04" {
		t.Errorf("expected traceability OSHE-BR-ACT-04 on denial, got %s", deniedRes.TraceabilityKey)
	}
}

func TestRuleMatrix_RegistrationValidation(t *testing.T) {
	matrix := workflowaction.NewRuleMatrix("1.0.0")

	// Blank Rule ID
	err := matrix.RegisterRule(workflowaction.BusinessRule{RuleID: "   "})
	if !errors.Is(err, workflowaction.ErrBlankRuleID) {
		t.Errorf("expected ErrBlankRuleID, got %v", err)
	}

	// Valid rule registration
	err = matrix.RegisterRule(workflowaction.BusinessRule{
		RuleID:          "RULE-CUSTOM-001",
		OwnerRole:       "SAFETY_OFFICER",
		TraceabilityKey: "CUSTOM-01",
	})
	if err != nil {
		t.Fatalf("failed to register valid rule: %v", err)
	}

	// Duplicate rule registration
	err = matrix.RegisterRule(workflowaction.BusinessRule{
		RuleID: "RULE-CUSTOM-001",
	})
	if !errors.Is(err, workflowaction.ErrDuplicateRuleID) {
		t.Errorf("expected ErrDuplicateRuleID, got %v", err)
	}

	// Retrieved rule defaults
	r, err := matrix.GetRule("RULE-CUSTOM-001")
	if err != nil {
		t.Fatalf("failed to get registered rule: %v", err)
	}
	if r.DeterministicResult != workflowaction.RuleResultPermitted {
		t.Errorf("expected default RuleResultPermitted, got %s", r.DeterministicResult)
	}
	if r.FailureBehavior != workflowaction.FailureFailClosed {
		t.Errorf("expected default FailureFailClosed, got %s", r.FailureBehavior)
	}
}
