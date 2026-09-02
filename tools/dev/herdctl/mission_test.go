package main

import "testing"

func TestExhaustiveTransitionMatrix(t *testing.T) {
	commands := []string{"validate", "start", "pause", "resume", "cancel", "archive"}
	states := []State{StateAbsent, StateCreated, StateValidated, StateRunning, StatePaused, StateCancelled, StateArchived}

	allowed := map[string]map[State]State{
		"validate": {StateCreated: StateValidated},
		"start":    {StateValidated: StateRunning},
		"pause":    {StateRunning: StatePaused},
		"resume":   {StatePaused: StateRunning},
		"cancel": {
			StateCreated:   StateCancelled,
			StateValidated: StateCancelled,
			StateRunning:   StateCancelled,
			StatePaused:    StateCancelled,
		},
		"archive": {StateCancelled: StateArchived},
	}

	for _, cmd := range commands {
		for _, from := range states {
			wantTo, wantOK := allowed[cmd][from]
			to, ok := transitionFor(cmd, from)
			if ok != wantOK {
				t.Errorf("%s from %s: ok=%v want %v", cmd, from, ok, wantOK)
			}
			if ok && to != wantTo {
				t.Errorf("%s from %s: to=%q want %q", cmd, from, to, wantTo)
			}
		}
	}
}

func TestValidateMission(t *testing.T) {
	good := Mission{
		ContractType:       "mission",
		ContractVersion:    "1.0.0",
		ID:                 "MISSION-001",
		Title:              "Test Mission",
		Goal:               "Test goal",
		NonGoals:           []string{"n1", "n2"},
		RiskClass:          "R2",
		BaseCommit:         "0123456789abcdef0123456789abcdef01234567",
		HumanDecisions:     []string{"d1"},
		DataClassification: "INTERNAL",
	}
	if err := validateMission(&good); err != nil {
		t.Fatalf("valid mission rejected: %v", err)
	}

	bad := []struct {
		name   string
		mutate func(*Mission)
	}{
		{"contract_type", func(m *Mission) { m.ContractType = "other" }},
		{"contract_version", func(m *Mission) { m.ContractVersion = "2.0.0" }},
		{"id_lowercase", func(m *Mission) { m.ID = "mission-001" }},
		{"id_traversal", func(m *Mission) { m.ID = "../MISSION-001" }},
		{"empty_title", func(m *Mission) { m.Title = "" }},
		{"empty_goal", func(m *Mission) { m.Goal = "" }},
		{"empty_non_goals", func(m *Mission) { m.NonGoals = nil }},
		{"dup_non_goals", func(m *Mission) { m.NonGoals = []string{"a", "a"} }},
		{"bad_risk", func(m *Mission) { m.RiskClass = "R9" }},
		{"bad_commit_short", func(m *Mission) { m.BaseCommit = "xyz" }},
		{"bad_commit_len", func(m *Mission) { m.BaseCommit = "0123456789abcdef01234567" }},
		{"high_classification", func(m *Mission) { m.DataClassification = "RESTRICTED" }},
		{"confidential_classification", func(m *Mission) { m.DataClassification = "CONFIDENTIAL_PERSONAL" }},
		{"invalid_classification", func(m *Mission) { m.DataClassification = "BOGUS" }},
		{"nonempty_extensions", func(m *Mission) { m.Extensions = map[string]any{"x": 1} }},
		{"dup_human_decisions", func(m *Mission) { m.HumanDecisions = []string{"a", "a"} }},
	}
	for _, c := range bad {
		m := good
		c.mutate(&m)
		if err := validateMission(&m); err == nil {
			t.Errorf("%s: expected validation failure", c.name)
		}
	}
}

func TestDecodeMissionRejectsTrailingValue(t *testing.T) {
	data := []byte(`{"contract_type":"mission","contract_version":"1.0.0","id":"M-1","title":"t","goal":"g","non_goals":["n"],"risk_class":"R2","base_commit":"0123456789abcdef0123456789abcdef01234567","human_decisions":[]}{"x":1}`)
	if _, err := decodeMission(data); err == nil {
		t.Fatal("trailing value accepted")
	}
}
