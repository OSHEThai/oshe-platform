package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const auditSchemaVersion = "1.0.0"

type AuditEvent struct {
	SchemaVersion       string `json:"schema_version"`
	MissionID           string `json:"mission_id"`
	Sequence            int    `json:"sequence"`
	OccurredAtUTC       string `json:"occurred_at_utc"`
	Command             string `json:"command"`
	FromState           string `json:"from_state"`
	ToState             string `json:"to_state"`
	MissionSHA256       string `json:"mission_sha256"`
	PreviousEventSHA256 string `json:"previous_event_sha256"`
	EventSHA256         string `json:"event_sha256"`
}

type Projection struct {
	SchemaVersion string `json:"schema_version"`
	MissionID     string `json:"mission_id"`
	MissionSHA256 string `json:"mission_sha256"`
	State         string `json:"state"`
	Sequence      int    `json:"sequence"`
	JournalTip    string `json:"journal_tip"`
}

type Store struct {
	Root  string
	Clock func() time.Time
}

var errDurabilityUnsupported = errors.New("directory fsync not supported on this platform")

var syncDir = func(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}

func (s *Store) now() time.Time {
	if s.Clock != nil {
		return s.Clock().UTC()
	}
	return time.Now().UTC()
}

func (s *Store) missionDir(id string) string {
	return filepath.Join(s.Root, "missions", id)
}

func (s *Store) auditDir(id string) string {
	return filepath.Join(s.missionDir(id), "audit")
}

func (s *Store) missionPath(id string) string {
	return filepath.Join(s.missionDir(id), "mission.json")
}

func (s *Store) statePath(id string) string {
	return filepath.Join(s.missionDir(id), "state.json")
}

func (e *AuditEvent) computeHash() (string, error) {
	clone := *e
	clone.EventSHA256 = ""
	b, err := json.Marshal(clone)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func tipHash(events []*AuditEvent) string {
	if len(events) == 0 {
		return ""
	}
	return events[len(events)-1].EventSHA256
}

// validateUTCTimestamp requires a canonical RFC3339 UTC timestamp.
func validateUTCTimestamp(s string) error {
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return fmt.Errorf("occurred_at_utc is not a valid RFC3339 timestamp")
	}
	if t.Location() != time.UTC || !strings.HasSuffix(s, "Z") {
		return fmt.Errorf("occurred_at_utc is not canonical UTC")
	}
	return nil
}

func (s *Store) Create(m *Mission) error {
	if err := validateMission(m); err != nil {
		return err
	}
	data, err := m.canonicalBytes()
	if err != nil {
		return err
	}
	hash := sha256Hex(data)

	existing, readErr := os.ReadFile(s.missionPath(m.ID))
	if readErr == nil {
		if sha256Hex(existing) != hash {
			return fmt.Errorf("mission already exists with different content")
		}
		events, _, rerr := s.replay(m.ID, hash)
		if rerr != nil {
			return rerr
		}
		if len(events) > 0 {
			return fmt.Errorf("mission already exists")
		}
		return s.writeGenesis(m.ID, hash)
	}
	if !os.IsNotExist(readErr) {
		return readErr
	}

	if err := os.MkdirAll(s.auditDir(m.ID), 0o755); err != nil {
		return err
	}
	if err := atomicWrite(s.missionPath(m.ID), data); err != nil {
		return err
	}
	return s.writeGenesis(m.ID, hash)
}

func (s *Store) writeGenesis(id, hash string) error {
	event := AuditEvent{
		SchemaVersion:       auditSchemaVersion,
		MissionID:           id,
		Sequence:            1,
		OccurredAtUTC:       s.now().Format(time.RFC3339Nano),
		Command:             "create",
		FromState:           string(StateAbsent),
		ToState:             string(StateCreated),
		MissionSHA256:       hash,
		PreviousEventSHA256: "",
	}
	eventHash, err := event.computeHash()
	if err != nil {
		return err
	}
	event.EventSHA256 = eventHash
	if err := s.appendEvent(id, &event); err != nil {
		return err
	}
	return s.writeProjection(id, hash, StateCreated, &event)
}

func (s *Store) Transition(id, command string) error {
	_, hash, err := s.loadMission(id)
	if err != nil {
		return err
	}
	events, state, err := s.replay(id, hash)
	if err != nil {
		return err
	}
	to, ok := transitionFor(command, state)
	if !ok {
		return fmt.Errorf("invalid transition %q from %q", command, state)
	}
	event := AuditEvent{
		SchemaVersion:       auditSchemaVersion,
		MissionID:           id,
		Sequence:            len(events) + 1,
		OccurredAtUTC:       s.now().Format(time.RFC3339Nano),
		Command:             command,
		FromState:           string(state),
		ToState:             string(to),
		MissionSHA256:       hash,
		PreviousEventSHA256: tipHash(events),
	}
	eventHash, err := event.computeHash()
	if err != nil {
		return err
	}
	event.EventSHA256 = eventHash
	if err := s.appendEvent(id, &event); err != nil {
		return err
	}
	return s.writeProjection(id, hash, to, &event)
}

func (s *Store) Status(id string) (State, error) {
	_, hash, err := s.loadMission(id)
	if err != nil {
		return "", err
	}
	events, state, err := s.replay(id, hash)
	if err != nil {
		return "", err
	}
	if err := s.validateProjection(id, hash, events, state, tipHash(events)); err != nil {
		return "", err
	}
	return state, nil
}

func (s *Store) loadMission(id string) (*Mission, string, error) {
	data, err := os.ReadFile(s.missionPath(id))
	if err != nil {
		return nil, "", fmt.Errorf("mission not found: %v", err)
	}
	m, err := decodeMission(data)
	if err != nil {
		return nil, "", fmt.Errorf("invalid mission document: %v", err)
	}
	return m, sha256Hex(data), nil
}

func decodeAuditEvent(data []byte) (*AuditEvent, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var ev AuditEvent
	if err := decodeSingle(dec, &ev); err != nil {
		return nil, err
	}
	canonical, err := json.Marshal(ev)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(canonical, data) {
		return nil, fmt.Errorf("audit event is not canonical")
	}
	return &ev, nil
}

func decodeProjection(data []byte) (*Projection, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var p Projection
	if err := decodeSingle(dec, &p); err != nil {
		return nil, err
	}
	canonical, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(canonical, data) {
		return nil, fmt.Errorf("projection is not canonical")
	}
	return &p, nil
}

func (s *Store) replay(id, missionHash string) ([]*AuditEvent, State, error) {
	entries, err := os.ReadDir(s.auditDir(id))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, StateAbsent, nil
		}
		return nil, "", err
	}
	var events []*AuditEvent
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || strings.HasPrefix(name, ".tmp-") || strings.HasSuffix(name, ".pending") {
			continue
		}
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		base := strings.TrimSuffix(name, ".json")
		parts := strings.SplitN(base, "-", 2)
		if len(parts) != 2 {
			return nil, "", fmt.Errorf("malformed audit filename")
		}
		fileSeq, err := strconv.Atoi(parts[0])
		if err != nil {
			return nil, "", fmt.Errorf("malformed audit sequence")
		}
		fileHash := parts[1]
		data, err := os.ReadFile(filepath.Join(s.auditDir(id), name))
		if err != nil {
			return nil, "", err
		}
		ev, err := decodeAuditEvent(data)
		if err != nil {
			return nil, "", err
		}
		if fileSeq != ev.Sequence {
			return nil, "", fmt.Errorf("audit filename sequence mismatch")
		}
		if fileHash != ev.EventSHA256 {
			return nil, "", fmt.Errorf("audit filename hash mismatch")
		}
		events = append(events, ev)
	}
	sort.Slice(events, func(i, j int) bool { return events[i].Sequence < events[j].Sequence })

	knownStates := map[string]bool{
		"ABSENT": true, "CREATED": true, "VALIDATED": true, "RUNNING": true,
		"PAUSED": true, "CANCELLED": true, "ARCHIVED": true,
	}
	var tip string
	var prevState string
	for i, ev := range events {
		if ev.Sequence != i+1 {
			return nil, "", fmt.Errorf("audit sequence gap or duplicate")
		}
		if ev.PreviousEventSHA256 != tip {
			return nil, "", fmt.Errorf("audit hash link broken")
		}
		if ev.MissionSHA256 != missionHash {
			return nil, "", fmt.Errorf("mission digest mismatch")
		}
		computed, err := ev.computeHash()
		if err != nil {
			return nil, "", err
		}
		if computed != ev.EventSHA256 {
			return nil, "", fmt.Errorf("event hash mismatch")
		}
		if ev.SchemaVersion != auditSchemaVersion {
			return nil, "", fmt.Errorf("audit schema version mismatch")
		}
		if ev.MissionID != id {
			return nil, "", fmt.Errorf("audit mission id mismatch")
		}
		if err := validateUTCTimestamp(ev.OccurredAtUTC); err != nil {
			return nil, "", err
		}
		if !knownStates[ev.FromState] || !knownStates[ev.ToState] {
			return nil, "", fmt.Errorf("audit contains unknown state")
		}
		if i == 0 {
			if ev.Command != "create" || ev.FromState != string(StateAbsent) || ev.ToState != string(StateCreated) {
				return nil, "", fmt.Errorf("invalid genesis event")
			}
		} else {
			if ev.FromState != prevState {
				return nil, "", fmt.Errorf("consecutive state continuity broken")
			}
			wantTo, ok := transitionFor(ev.Command, State(ev.FromState))
			if !ok || string(wantTo) != ev.ToState {
				return nil, "", fmt.Errorf("invalid lifecycle transition in journal")
			}
		}
		tip = ev.EventSHA256
		prevState = ev.ToState
	}

	state := StateAbsent
	if len(events) > 0 {
		state = State(events[len(events)-1].ToState)
	}
	return events, state, nil
}

func (s *Store) validateProjection(id, missionHash string, events []*AuditEvent, state State, tip string) error {
	data, err := os.ReadFile(s.statePath(id))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	p, err := decodeProjection(data)
	if err != nil {
		return err
	}
	if p.SchemaVersion != auditSchemaVersion {
		return fmt.Errorf("projection schema mismatch")
	}
	if p.MissionID != id {
		return fmt.Errorf("projection mission id mismatch")
	}
	if p.MissionSHA256 != missionHash {
		return fmt.Errorf("projection mission digest mismatch")
	}
	if p.Sequence < 1 {
		return fmt.Errorf("projection sequence is invalid")
	}
	if p.Sequence > len(events) {
		return fmt.Errorf("projection sequence is ahead of the journal")
	}
	if p.Sequence == len(events) {
		if p.State != string(state) || p.JournalTip != tip {
			return fmt.Errorf("projection conflicts with journal at current sequence")
		}
	} else {
		ev := events[p.Sequence-1]
		if p.State != ev.ToState || p.JournalTip != ev.EventSHA256 {
			return fmt.Errorf("stale projection conflicts with journal at its declared sequence")
		}
	}
	return nil
}

func (s *Store) appendEvent(id string, e *AuditEvent) error {
	data, err := json.Marshal(e)
	if err != nil {
		return err
	}
	name := fmt.Sprintf("%d-%s.json", e.Sequence, e.EventSHA256)
	return atomicWrite(filepath.Join(s.auditDir(id), name), data)
}

func (s *Store) writeProjection(id, missionHash string, state State, e *AuditEvent) error {
	p := Projection{
		SchemaVersion: auditSchemaVersion,
		MissionID:     id,
		MissionSHA256: missionHash,
		State:         string(state),
		Sequence:      e.Sequence,
		JournalTip:    e.EventSHA256,
	}
	data, err := json.Marshal(p)
	if err != nil {
		return err
	}
	return atomicWrite(s.statePath(id), data)
}

func atomicWrite(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return err
	}
	if err := syncDir(dir); err != nil {
		return fmt.Errorf("%w: %v", errDurabilityUnsupported, err)
	}
	return nil
}
