package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

type BenchmarkScorecard struct {
	MissionID        string            `json:"mission_id"`
	Fixture          string            `json:"fixture_id"`
	FailureInjection string            `json:"failure_injection_policy"`
	Measures         map[string]string `json:"measures"`
	Status           string            `json:"status"` // "PENDING", "COMPLETED", "FAILED"
}

type BenchmarkController struct {
	Store *Store
}

func (c *BenchmarkController) scorecardFile(id string) string {
	return filepath.Join(c.Store.missionDir(id), "benchmark_scorecard.json")
}

func (c *BenchmarkController) Scorecard(id string) (*BenchmarkScorecard, error) {
	data, err := os.ReadFile(c.scorecardFile(id))
	if err != nil {
		if os.IsNotExist(err) {
			return &BenchmarkScorecard{
				MissionID: id,
				Status:    "PENDING",
				Measures:  make(map[string]string),
			}, nil
		}
		return nil, err
	}
	var sc BenchmarkScorecard
	if err := json.Unmarshal(data, &sc); err != nil {
		return nil, err
	}
	if sc.Measures == nil {
		sc.Measures = make(map[string]string)
	}
	return &sc, nil
}

func (c *BenchmarkController) saveScorecard(sc *BenchmarkScorecard) error {
	b, err := json.Marshal(sc)
	if err != nil {
		return err
	}
	return atomicWrite(c.scorecardFile(sc.MissionID), b)
}

func (c *BenchmarkController) SetFixture(id, fixture string) error {
	sc, err := c.Scorecard(id)
	if err != nil {
		return err
	}
	sc.Fixture = fixture
	return c.saveScorecard(sc)
}

func (c *BenchmarkController) SetFailureInjection(id, policy string) error {
	sc, err := c.Scorecard(id)
	if err != nil {
		return err
	}
	sc.FailureInjection = policy
	return c.saveScorecard(sc)
}

func (c *BenchmarkController) RecordMeasure(id, key, value string) error {
	sc, err := c.Scorecard(id)
	if err != nil {
		return err
	}
	if sc.Status == "COMPLETED" {
		return errors.New("cannot record measure after benchmark completion")
	}
	sc.Measures[key] = value
	return c.saveScorecard(sc)
}

func (c *BenchmarkController) Complete(id string) error {
	sc, err := c.Scorecard(id)
	if err != nil {
		return err
	}
	if sc.Fixture == "" {
		return errors.New("cannot complete without a fixture")
	}
	sc.Status = "COMPLETED"
	return c.saveScorecard(sc)
}

func (c *BenchmarkController) Fail(id string) error {
	sc, err := c.Scorecard(id)
	if err != nil {
		return err
	}
	sc.Status = "FAILED"
	return c.saveScorecard(sc)
}
