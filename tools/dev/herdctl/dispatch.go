package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type DispatchResult struct {
	MissionID   string `json:"mission_id"`
	Disposition string `json:"disposition"`
	Error       string `json:"error,omitempty"`
}

type LocalDispatcher struct {
	Store *Store
}

func (d *LocalDispatcher) dispatchFile(id string) string {
	return filepath.Join(d.Store.missionDir(id), "mock_dispatch.json")
}

func (d *LocalDispatcher) resultFile(id string) string {
	return filepath.Join(d.Store.missionDir(id), "mock_result.json")
}

func (d *LocalDispatcher) Dispatch(id string) error {
	path := d.dispatchFile(id)
	if _, err := os.Stat(path); err == nil {
		// Duplicate replay: if already dispatched, it is idempotent.
		return nil
	}
	return atomicWrite(path, []byte(`{"status":"dispatched"}`))
}

func (d *LocalDispatcher) Monitor(id string) (*DispatchResult, error) {
	data, err := os.ReadFile(d.resultFile(id))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.New("pending")
		}
		return nil, err
	}

	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var res DispatchResult
	if err := dec.Decode(&res); err != nil {
		return nil, fmt.Errorf("invalid result: %v", err)
	}

	// Canonical validation
	canonical, err := json.Marshal(res)
	if err != nil || !bytes.Equal(canonical, data) {
		return nil, errors.New("result is not canonical")
	}
	if res.MissionID != id {
		return nil, errors.New("mission id mismatch")
	}
	return &res, nil
}

func (d *LocalDispatcher) Timeout(id string) error {
	return d.writeResult(id, "TIMEOUT", "execution timed out")
}

func (d *LocalDispatcher) Cancel(id string) error {
	return d.writeResult(id, "CANCELLED", "execution cancelled")
}

func (d *LocalDispatcher) Restart(id string) error {
	os.Remove(d.resultFile(id))
	os.Remove(d.dispatchFile(id))
	return d.Dispatch(id)
}

func (d *LocalDispatcher) writeResult(id, disposition, errStr string) error {
	res := DispatchResult{
		MissionID:   id,
		Disposition: disposition,
	}
	if errStr != "" {
		res.Error = errStr
	}
	b, err := json.Marshal(res)
	if err != nil {
		return err
	}
	return atomicWrite(d.resultFile(id), b)
}

// MockComplete is used by tests to inject a passing result.
func (d *LocalDispatcher) MockComplete(id string) error {
	return d.writeResult(id, "PASS", "")
}
