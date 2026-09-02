package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
)

type State string

const (
	StateAbsent    State = "ABSENT"
	StateCreated   State = "CREATED"
	StateValidated State = "VALIDATED"
	StateRunning   State = "RUNNING"
	StatePaused    State = "PAUSED"
	StateCancelled State = "CANCELLED"
	StateArchived  State = "ARCHIVED"
)

type Mission struct {
	ContractType       string         `json:"contract_type"`
	ContractVersion    string         `json:"contract_version"`
	ID                 string         `json:"id"`
	Title              string         `json:"title"`
	Goal               string         `json:"goal"`
	NonGoals           []string       `json:"non_goals"`
	RiskClass          string         `json:"risk_class"`
	BaseCommit         string         `json:"base_commit"`
	IntegrationBranch  string         `json:"integration_branch,omitempty"`
	HumanDecisions     []string       `json:"human_decisions"`
	DataClassification string         `json:"data_classification,omitempty"`
	Extensions         map[string]any `json:"extensions,omitempty"`
}

var (
	idPattern  = regexp.MustCompile(`^[A-Z0-9-]+$`)
	commitPat  = regexp.MustCompile(`^(?:[0-9a-f]{40}|[0-9a-f]{64})$`)
	riskValues = map[string]bool{"R0": true, "R1": true, "R2": true, "R3": true, "R4": true, "RX": true}
)

// decodeSingle rejects unknown fields and requires a single value ending in EOF.
func decodeSingle(dec *json.Decoder, v any) error {
	if err := dec.Decode(v); err != nil {
		return err
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		return fmt.Errorf("trailing data")
	}
	return nil
}

func decodeMission(data []byte) (*Mission, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var m Mission
	if err := decodeSingle(dec, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

func (m *Mission) canonicalBytes() ([]byte, error) {
	return json.Marshal(m)
}

// validateMission validates the mission document against the 1.0.0 contract.
// Error messages are redacted and never include mission field values.
func validateMission(m *Mission) error {
	if m.ContractType != "mission" {
		return fmt.Errorf("contract_type must be mission")
	}
	if m.ContractVersion != "1.0.0" {
		return fmt.Errorf("contract_version must be 1.0.0")
	}
	if !idPattern.MatchString(m.ID) {
		return fmt.Errorf("id does not match the mission identifier grammar")
	}
	if m.Title == "" {
		return fmt.Errorf("title must be non-empty")
	}
	if m.Goal == "" {
		return fmt.Errorf("goal must be non-empty")
	}
	if len(m.NonGoals) == 0 || !uniqueNonEmpty(m.NonGoals) {
		return fmt.Errorf("non_goals must be a non-empty list of unique non-empty strings")
	}
	if !riskValues[m.RiskClass] {
		return fmt.Errorf("risk_class must be one of R0,R1,R2,R3,R4,RX")
	}
	if !commitPat.MatchString(m.BaseCommit) {
		return fmt.Errorf("base_commit must be 40 or 64 hex characters")
	}
	if !uniqueNonEmpty(m.HumanDecisions) {
		return fmt.Errorf("human_decisions must be unique non-empty strings")
	}
	switch m.DataClassification {
	case "", "PUBLIC", "INTERNAL":
	case "RESTRICTED", "CONFIDENTIAL_PERSONAL", "HIGHLY_RESTRICTED":
		return fmt.Errorf("data_classification is not permitted in this bounded CLI")
	default:
		return fmt.Errorf("data_classification has an invalid value")
	}
	if m.Extensions != nil && len(m.Extensions) != 0 {
		return fmt.Errorf("extensions must be empty")
	}
	return nil
}

func uniqueNonEmpty(list []string) bool {
	seen := map[string]bool{}
	for _, s := range list {
		if s == "" || seen[s] {
			return false
		}
		seen[s] = true
	}
	return true
}

var transitionTable = map[string]map[State]State{
	"validate": {StateCreated: StateValidated},
	"start":    {StateValidated: StateRunning},
	"pause":    {StateRunning: StatePaused},
	"resume":   {StatePaused: StateRunning},
	"archive":  {StateCancelled: StateArchived},
}

func transitionFor(command string, from State) (State, bool) {
	if command == "cancel" {
		switch from {
		case StateCreated, StateValidated, StateRunning, StatePaused:
			return StateCancelled, true
		default:
			return "", false
		}
	}
	to, ok := transitionTable[command][from]
	return to, ok
}
