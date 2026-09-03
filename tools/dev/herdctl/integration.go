package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

type IntegrationStatus struct {
	MissionID    string   `json:"mission_id"`
	Dependencies []string `json:"dependencies_met"`
	Candidate    bool     `json:"candidate_prepared"`
	ReviewState  string   `json:"review_state"` // "PENDING", "REMEDIATION_REQUIRED", "APPROVED"
	Handoff      bool     `json:"handoff_complete"`
	DraftPR      bool     `json:"draft_pr_requested"`
}

type IntegrationController struct {
	Store *Store
}

func (c *IntegrationController) statusFile(id string) string {
	return filepath.Join(c.Store.missionDir(id), "integration_status.json")
}

func (c *IntegrationController) Status(id string) (*IntegrationStatus, error) {
	data, err := os.ReadFile(c.statusFile(id))
	if err != nil {
		if os.IsNotExist(err) {
			return &IntegrationStatus{MissionID: id, ReviewState: "PENDING"}, nil
		}
		return nil, err
	}
	var st IntegrationStatus
	if err := json.Unmarshal(data, &st); err != nil {
		return nil, err
	}
	return &st, nil
}

func (c *IntegrationController) saveStatus(st *IntegrationStatus) error {
	b, err := json.Marshal(st)
	if err != nil {
		return err
	}
	return atomicWrite(c.statusFile(st.MissionID), b)
}

func (c *IntegrationController) VerifyDependencies(id string, deps []string) error {
	st, err := c.Status(id)
	if err != nil {
		return err
	}
	st.Dependencies = deps
	return c.saveStatus(st)
}

func (c *IntegrationController) PrepareCandidate(id string) error {
	st, err := c.Status(id)
	if err != nil {
		return err
	}
	st.Candidate = true
	return c.saveStatus(st)
}

func (c *IntegrationController) SubmitReview(id string, approved bool) error {
	st, err := c.Status(id)
	if err != nil {
		return err
	}
	if !st.Candidate {
		return errors.New("cannot review before candidate is prepared")
	}
	if approved {
		st.ReviewState = "APPROVED"
	} else {
		st.ReviewState = "REMEDIATION_REQUIRED"
		// Loop: require preparing the candidate again.
		st.Candidate = false
	}
	return c.saveStatus(st)
}

func (c *IntegrationController) Handoff(id string) error {
	st, err := c.Status(id)
	if err != nil {
		return err
	}
	if st.ReviewState != "APPROVED" {
		return errors.New("cannot handoff without approved review")
	}
	st.Handoff = true
	return c.saveStatus(st)
}

func (c *IntegrationController) RequestDraftPR(id string) error {
	st, err := c.Status(id)
	if err != nil {
		return err
	}
	if !st.Handoff {
		return errors.New("cannot request draft PR before handoff is complete")
	}
	st.DraftPR = true
	return c.saveStatus(st)
}
