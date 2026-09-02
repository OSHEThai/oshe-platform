package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func testClock() func() time.Time {
	return func() time.Time {
		return time.Date(2026, 9, 2, 20, 0, 0, 0, time.UTC)
	}
}

func testMission(id string) *Mission {
	return &Mission{
		ContractType:       "mission",
		ContractVersion:    "1.0.0",
		ID:                 id,
		Title:              "Synthetic Mission",
		Goal:               "Validate herdctl state",
		NonGoals:           []string{"production data", "network access"},
		RiskClass:          "R2",
		BaseCommit:         "0123456789abcdef0123456789abcdef01234567",
		HumanDecisions:     []string{"none"},
		DataClassification: "INTERNAL",
	}
}

func newStore(t *testing.T) *Store {
	t.Helper()
	return &Store{Root: t.TempDir(), Clock: testClock()}
}

// TestMain substitutes the directory-fsync seam with a supported-platform model
// for the normal store tests. The unsupported branch is exercised separately.
func TestMain(m *testing.M) {
	orig := syncDir
	syncDir = func(string) error { return nil }
	code := m.Run()
	syncDir = orig
	os.Exit(code)
}

func fullLifecycle(t *testing.T, s *Store, id string) {
	t.Helper()
	if err := s.Create(testMission(id)); err != nil {
		t.Fatalf("create: %v", err)
	}
	for _, cmd := range []string{"validate", "start", "pause", "resume", "cancel", "archive"} {
		if err := s.Transition(id, cmd); err != nil {
			t.Fatalf("%s: %v", cmd, err)
		}
	}
}

func auditSnapshot(t *testing.T, dir, id string) string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(dir, "missions", id, "audit"))
	if err != nil {
		return ""
	}
	var parts []string
	for _, e := range entries {
		if e.IsDir() || len(e.Name()) == 0 || e.Name()[0] == '.' {
			continue
		}
		data, _ := os.ReadFile(filepath.Join(dir, "missions", id, "audit", e.Name()))
		parts = append(parts, e.Name()+"="+string(data))
	}
	return strings.Join(parts, "\n")
}

// rewriteEvent reads an audit event by sequence prefix, mutates it, recomputes
// its hash, and rewrites it under its new hash name.
func rewriteEvent(t *testing.T, s *Store, id, seqPrefix string, mutate func(*AuditEvent)) {
	t.Helper()
	auditDir := filepath.Join(s.Root, "missions", id, "audit")
	entries, _ := os.ReadDir(auditDir)
	for _, e := range entries {
		if e.IsDir() || !strings.HasPrefix(e.Name(), seqPrefix) {
			continue
		}
		p := filepath.Join(auditDir, e.Name())
		data, _ := os.ReadFile(p)
		var ev AuditEvent
		if err := json.Unmarshal(data, &ev); err != nil {
			t.Fatal(err)
		}
		mutate(&ev)
		hash, err := ev.computeHash()
		if err != nil {
			t.Fatal(err)
		}
		ev.EventSHA256 = hash
		newData, _ := json.Marshal(ev)
		if err := os.Remove(p); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(auditDir, fmt.Sprintf("%d-%s.json", ev.Sequence, hash)), newData, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
}

func TestFullLifecycleAndStatus(t *testing.T) {
	s := newStore(t)
	fullLifecycle(t, s, "MISSION-001")
	state, err := s.Status("MISSION-001")
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if state != StateArchived {
		t.Fatalf("state = %q, want ARCHIVED", state)
	}
}

func TestRestartReplayAfterEveryTransition(t *testing.T) {
	dir := t.TempDir()
	s := &Store{Root: dir, Clock: testClock()}
	id := "MISSION-REPLAY"
	if err := s.Create(testMission(id)); err != nil {
		t.Fatal(err)
	}
	expected := []struct {
		cmd string
		to  State
	}{
		{"validate", StateValidated},
		{"start", StateRunning},
		{"pause", StatePaused},
		{"resume", StateRunning},
		{"cancel", StateCancelled},
		{"archive", StateArchived},
	}
	for i, step := range expected {
		if err := s.Transition(id, step.cmd); err != nil {
			t.Fatalf("%s: %v", step.cmd, err)
		}
		s2 := &Store{Root: dir, Clock: testClock()}
		state, err := s2.Status(id)
		if err != nil {
			t.Fatalf("step %d (%s) replay: %v", i, step.cmd, err)
		}
		if state != step.to {
			t.Fatalf("step %d (%s) replayed state = %q, want %q", i, step.cmd, state, step.to)
		}
	}
}

func TestForbiddenTransitionsWriteNoBytes(t *testing.T) {
	drive := []struct {
		state State
		cmds  []string
	}{
		{StateCreated, nil},
		{StateValidated, []string{"validate"}},
		{StateRunning, []string{"validate", "start"}},
		{StatePaused, []string{"validate", "start", "pause"}},
		{StateCancelled, []string{"cancel"}},
		{StateArchived, []string{"cancel", "archive"}},
	}
	allCmds := []string{"validate", "start", "pause", "resume", "cancel", "archive"}
	for _, sc := range drive {
		dir := t.TempDir()
		s := &Store{Root: dir, Clock: testClock()}
		id := "MISSION-FORBIDDEN"
		if err := s.Create(testMission(id)); err != nil {
			t.Fatal(err)
		}
		for _, c := range sc.cmds {
			if err := s.Transition(id, c); err != nil {
				t.Fatalf("drive %s: %v", c, err)
			}
		}
		before := auditSnapshot(t, dir, id)
		for _, cmd := range allCmds {
			if _, ok := transitionFor(cmd, sc.state); ok {
				continue
			}
			if err := s.Transition(id, cmd); err == nil {
				t.Errorf("%s from %s: forbidden transition succeeded", cmd, sc.state)
			}
			if after := auditSnapshot(t, dir, id); after != before {
				t.Errorf("%s from %s: forbidden transition wrote bytes", cmd, sc.state)
			}
		}
	}
}

// CODE-001 hostile replay cases.

func TestReplayRejectsUnknownField(t *testing.T) {
	s := newStore(t)
	id := "MISSION-UNKNOWN"
	if err := s.Create(testMission(id)); err != nil {
		t.Fatal(err)
	}
	auditDir := filepath.Join(s.Root, "missions", id, "audit")
	entries, _ := os.ReadDir(auditDir)
	for _, e := range entries {
		if e.IsDir() || e.Name()[0] == '.' {
			continue
		}
		p := filepath.Join(auditDir, e.Name())
		data, _ := os.ReadFile(p)
		var m map[string]any
		json.Unmarshal(data, &m)
		m["extra"] = "x"
		tampered, _ := json.Marshal(m)
		os.WriteFile(p, tampered, 0o644)
		break
	}
	if _, err := s.Status(id); err == nil {
		t.Fatal("unknown field in event not rejected")
	}
}

func TestReplayRejectsNoncanonicalEvent(t *testing.T) {
	s := newStore(t)
	id := "MISSION-NONCANON"
	if err := s.Create(testMission(id)); err != nil {
		t.Fatal(err)
	}
	auditDir := filepath.Join(s.Root, "missions", id, "audit")
	entries, _ := os.ReadDir(auditDir)
	for _, e := range entries {
		if e.IsDir() || e.Name()[0] == '.' {
			continue
		}
		p := filepath.Join(auditDir, e.Name())
		data, _ := os.ReadFile(p)
		var m map[string]any
		json.Unmarshal(data, &m)
		noncanonical, _ := json.Marshal(m) // sorted keys, non-canonical order
		os.WriteFile(p, noncanonical, 0o644)
		break
	}
	if _, err := s.Status(id); err == nil {
		t.Fatal("noncanonical event not rejected")
	}
}

func TestReplayRejectsWrongMissionID(t *testing.T) {
	s := newStore(t)
	id := "MISSION-WRONGID"
	if err := s.Create(testMission(id)); err != nil {
		t.Fatal(err)
	}
	rewriteEvent(t, s, id, "1-", func(ev *AuditEvent) { ev.MissionID = "OTHER-MISSION" })
	if _, err := s.Status(id); err == nil {
		t.Fatal("wrong mission id not rejected")
	}
}

func TestReplayRejectsInvalidGenesis(t *testing.T) {
	s := newStore(t)
	id := "MISSION-BADGENESIS"
	if err := s.Create(testMission(id)); err != nil {
		t.Fatal(err)
	}
	rewriteEvent(t, s, id, "1-", func(ev *AuditEvent) { ev.Command = "validate" })
	if _, err := s.Status(id); err == nil {
		t.Fatal("invalid genesis not rejected")
	}
}

func TestReplayRejectsImpossibleTransition(t *testing.T) {
	s := newStore(t)
	id := "MISSION-BADTRANS"
	if err := s.Create(testMission(id)); err != nil {
		t.Fatal(err)
	}
	if err := s.Transition(id, "validate"); err != nil {
		t.Fatal(err)
	}
	rewriteEvent(t, s, id, "2-", func(ev *AuditEvent) { ev.ToState = string(StateArchived) })
	if _, err := s.Status(id); err == nil {
		t.Fatal("impossible transition not rejected")
	}
}

// CODE-002 projection validation.

func TestCurrentProjectionMismatchFailsClosed(t *testing.T) {
	dir := t.TempDir()
	s := &Store{Root: dir, Clock: testClock()}
	id := "MISSION-PROJMISMATCH"
	if err := s.Create(testMission(id)); err != nil {
		t.Fatal(err)
	}
	sp := filepath.Join(dir, "missions", id, "state.json")
	data, _ := os.ReadFile(sp)
	var p Projection
	json.Unmarshal(data, &p)
	p.State = "RUNNING"
	tampered, _ := json.Marshal(p)
	os.WriteFile(sp, tampered, 0o644)
	if _, err := s.Status(id); err == nil {
		t.Fatal("current-sequence projection mismatch not failed closed")
	}
}

// CODE-003 resumable create.

func TestCreateRecoveryAfterCrash(t *testing.T) {
	dir := t.TempDir()
	m := testMission("MISSION-CRASH")
	data, _ := m.canonicalBytes()
	os.MkdirAll(filepath.Join(dir, "missions", "MISSION-CRASH", "audit"), 0o755)
	os.WriteFile(filepath.Join(dir, "missions", "MISSION-CRASH", "mission.json"), data, 0o644)
	s := &Store{Root: dir, Clock: testClock()}
	if err := s.Create(m); err != nil {
		t.Fatalf("resumable create failed: %v", err)
	}
	if state, _ := s.Status("MISSION-CRASH"); state != StateCreated {
		t.Fatalf("state = %q, want CREATED", state)
	}
}

// CODE-004 directory fsync surfaced.

func TestDirSyncUnsupportedSurfaced(t *testing.T) {
	orig := syncDir
	syncDir = func(string) error { return errors.New("flush unsupported") }
	defer func() { syncDir = orig }()
	s := newStore(t)
	err := s.Create(testMission("MISSION-DIRSYNC"))
	if !errors.Is(err, errDurabilityUnsupported) {
		t.Fatalf("Create error = %v, want durability-unsupported", err)
	}
}

func TestTamperMissionFailsClosed(t *testing.T) {
	dir := t.TempDir()
	s := &Store{Root: dir, Clock: testClock()}
	if err := s.Create(testMission("MISSION-TAMPER")); err != nil {
		t.Fatal(err)
	}
	if err := s.Transition("MISSION-TAMPER", "validate"); err != nil {
		t.Fatal(err)
	}
	mp := filepath.Join(dir, "missions", "MISSION-TAMPER", "mission.json")
	data, _ := os.ReadFile(mp)
	os.WriteFile(mp, append(data, '\n'), 0o644)
	if _, err := s.Status("MISSION-TAMPER"); err == nil {
		t.Fatal("tampered mission did not fail closed")
	}
}

func TestEventGapFailsClosed(t *testing.T) {
	dir := t.TempDir()
	s := &Store{Root: dir, Clock: testClock()}
	if err := s.Create(testMission("MISSION-GAP")); err != nil {
		t.Fatal(err)
	}
	if err := s.Transition("MISSION-GAP", "validate"); err != nil {
		t.Fatal(err)
	}
	auditDir := filepath.Join(dir, "missions", "MISSION-GAP", "audit")
	entries, _ := os.ReadDir(auditDir)
	for _, e := range entries {
		if !e.IsDir() && strings.HasPrefix(e.Name(), "1-") {
			os.Remove(filepath.Join(auditDir, e.Name()))
		}
	}
	if _, err := s.Status("MISSION-GAP"); err == nil {
		t.Fatal("audit gap did not fail closed")
	}
}

func TestStaleProjectionRecovered(t *testing.T) {
	dir := t.TempDir()
	s := &Store{Root: dir, Clock: testClock()}
	id := "MISSION-STALE"
	if err := s.Create(testMission(id)); err != nil {
		t.Fatal(err)
	}
	sp := filepath.Join(dir, "missions", id, "state.json")
	early, _ := os.ReadFile(sp)
	for _, cmd := range []string{"validate", "start", "pause", "resume", "cancel", "archive"} {
		if err := s.Transition(id, cmd); err != nil {
			t.Fatal(err)
		}
	}
	os.WriteFile(sp, early, 0o644)
	s2 := &Store{Root: dir, Clock: testClock()}
	if state, err := s2.Status(id); err != nil || state != StateArchived {
		t.Fatalf("stale projection: state=%q err=%v", state, err)
	}
}

func TestMissingProjectionRecovered(t *testing.T) {
	dir := t.TempDir()
	s := &Store{Root: dir, Clock: testClock()}
	id := "MISSION-MISSING"
	fullLifecycle(t, s, id)
	os.Remove(filepath.Join(dir, "missions", id, "state.json"))
	s2 := &Store{Root: dir, Clock: testClock()}
	if state, err := s2.Status(id); err != nil || state != StateArchived {
		t.Fatalf("missing projection: state=%q err=%v", state, err)
	}
}

func TestPendingCrashDataIgnored(t *testing.T) {
	dir := t.TempDir()
	s := &Store{Root: dir, Clock: testClock()}
	if err := s.Create(testMission("MISSION-PENDING")); err != nil {
		t.Fatal(err)
	}
	auditDir := filepath.Join(dir, "missions", "MISSION-PENDING", "audit")
	os.WriteFile(filepath.Join(auditDir, ".tmp-crash"), []byte("garbage"), 0o644)
	os.WriteFile(filepath.Join(auditDir, "99-deadbeef.json.pending"), []byte("garbage"), 0o644)
	if state, err := s.Status("MISSION-PENDING"); err != nil || state != StateCreated {
		t.Fatalf("pending data affected replay: state=%q err=%v", state, err)
	}
}

func TestArchiveRetainsData(t *testing.T) {
	dir := t.TempDir()
	s := &Store{Root: dir, Clock: testClock()}
	fullLifecycle(t, s, "MISSION-ARCHIVE")
	if _, err := os.Stat(filepath.Join(dir, "missions", "MISSION-ARCHIVE", "mission.json")); err != nil {
		t.Fatalf("mission deleted after archive: %v", err)
	}
	entries, _ := os.ReadDir(filepath.Join(dir, "missions", "MISSION-ARCHIVE", "audit"))
	if len(entries) == 0 {
		t.Fatal("audit deleted after archive")
	}
}

func TestStatusReadOnly(t *testing.T) {
	dir := t.TempDir()
	s := &Store{Root: dir, Clock: testClock()}
	id := "MISSION-RO"
	if err := s.Create(testMission(id)); err != nil {
		t.Fatal(err)
	}
	before := auditSnapshot(t, dir, id)
	if _, err := s.Status(id); err != nil {
		t.Fatal(err)
	}
	if after := auditSnapshot(t, dir, id); after != before {
		t.Fatal("status mutated the audit journal")
	}
}

func TestAuditExcludesPayload(t *testing.T) {
	dir := t.TempDir()
	s := &Store{Root: dir, Clock: testClock()}
	if err := s.Create(testMission("MISSION-AUDIT")); err != nil {
		t.Fatal(err)
	}
	entries, _ := os.ReadDir(filepath.Join(dir, "missions", "MISSION-AUDIT", "audit"))
	for _, e := range entries {
		data, _ := os.ReadFile(filepath.Join(dir, "missions", "MISSION-AUDIT", "audit", e.Name()))
		if strings.Contains(string(data), "Synthetic Mission") || strings.Contains(string(data), "Validate herdctl state") {
			t.Fatalf("audit event %s contains mission payload", e.Name())
		}
	}
}

func missionHash(t *testing.T, id string) string {
	t.Helper()
	b, err := testMission(id).canonicalBytes()
	if err != nil {
		t.Fatal(err)
	}
	return sha256Hex(b)
}

func writeProjection(t *testing.T, dir, id string, p *Projection) {
	t.Helper()
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "missions", id, "state.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
}

// CODE-001-R2 consecutive state continuity and UTC timestamp.

func TestReplayRejectsDisconnectedTransition(t *testing.T) {
	s := newStore(t)
	id := "MISSION-DISCONN"
	if err := s.Create(testMission(id)); err != nil {
		t.Fatal(err)
	}
	if err := s.Transition(id, "validate"); err != nil {
		t.Fatal(err)
	}
	// Valid start/VALIDATED/RUNNING triple, but the prior event ended at CREATED.
	rewriteEvent(t, s, id, "2-", func(ev *AuditEvent) {
		ev.Command = "start"
		ev.FromState = string(StateValidated)
		ev.ToState = string(StateRunning)
	})
	if _, err := s.Status(id); err == nil {
		t.Fatal("disconnected transition not rejected")
	}
}

func TestReplayRejectsNonUTCTimestamp(t *testing.T) {
	s := newStore(t)
	id := "MISSION-BADTS"
	if err := s.Create(testMission(id)); err != nil {
		t.Fatal(err)
	}
	rewriteEvent(t, s, id, "1-", func(ev *AuditEvent) {
		ev.OccurredAtUTC = "2026-09-02T20:00:00+07:00"
	})
	if _, err := s.Status(id); err == nil {
		t.Fatal("non-UTC timestamp not rejected")
	}
}

// CODE-002-R2 strict canonical projection and proof-at-declared-sequence.

func TestProjectionRejectsUnknownField(t *testing.T) {
	s := newStore(t)
	id := "MISSION-PROJUNKNOWN"
	if err := s.Create(testMission(id)); err != nil {
		t.Fatal(err)
	}
	sp := filepath.Join(s.Root, "missions", id, "state.json")
	data, _ := os.ReadFile(sp)
	var m map[string]any
	json.Unmarshal(data, &m)
	m["extra"] = "x"
	tampered, _ := json.Marshal(m)
	os.WriteFile(sp, tampered, 0o644)
	if _, err := s.Status(id); err == nil {
		t.Fatal("projection unknown field not rejected")
	}
}

func TestProjectionRejectsTrailingValue(t *testing.T) {
	s := newStore(t)
	id := "MISSION-PROJTRAIL"
	if err := s.Create(testMission(id)); err != nil {
		t.Fatal(err)
	}
	sp := filepath.Join(s.Root, "missions", id, "state.json")
	data, _ := os.ReadFile(sp)
	os.WriteFile(sp, append(data, []byte("{}\n")...), 0o644)
	if _, err := s.Status(id); err == nil {
		t.Fatal("projection trailing value not rejected")
	}
}

func TestProjectionRejectsDuplicateField(t *testing.T) {
	s := newStore(t)
	id := "MISSION-PROJDUP"
	if err := s.Create(testMission(id)); err != nil {
		t.Fatal(err)
	}
	sp := filepath.Join(s.Root, "missions", id, "state.json")
	dup := `{"schema_version":"1.0.0","mission_id":"MISSION-PROJDUP","mission_sha256":"x","state":"CREATED","state":"RUNNING","sequence":1,"journal_tip":"x"}`
	os.WriteFile(sp, []byte(dup), 0o644)
	if _, err := s.Status(id); err == nil {
		t.Fatal("projection duplicate field not rejected")
	}
}

func TestProjectionRejectsSequenceDowngrade(t *testing.T) {
	dir := t.TempDir()
	s := &Store{Root: dir, Clock: testClock()}
	id := "MISSION-DOWNGRADE"
	fullLifecycle(t, s, id)
	writeProjection(t, dir, id, &Projection{
		SchemaVersion: auditSchemaVersion,
		MissionID:     id,
		MissionSHA256: missionHash(t, id),
		State:         "ARCHIVED",
		Sequence:      1,
		JournalTip:    "deadbeef",
	})
	if _, err := s.Status(id); err == nil {
		t.Fatal("sequence-downgraded projection not rejected")
	}
}

func TestProjectionRejectsStaleStateMismatch(t *testing.T) {
	dir := t.TempDir()
	s := &Store{Root: dir, Clock: testClock()}
	id := "MISSION-STALESTATE"
	fullLifecycle(t, s, id)
	writeProjection(t, dir, id, &Projection{
		SchemaVersion: auditSchemaVersion,
		MissionID:     id,
		MissionSHA256: missionHash(t, id),
		State:         "PAUSED",
		Sequence:      3,
		JournalTip:    "deadbeef",
	})
	if _, err := s.Status(id); err == nil {
		t.Fatal("stale projection state mismatch not rejected")
	}
}

func TestProjectionRejectsStaleTipMismatch(t *testing.T) {
	dir := t.TempDir()
	s := &Store{Root: dir, Clock: testClock()}
	id := "MISSION-STALETIP"
	fullLifecycle(t, s, id)
	writeProjection(t, dir, id, &Projection{
		SchemaVersion: auditSchemaVersion,
		MissionID:     id,
		MissionSHA256: missionHash(t, id),
		State:         "RUNNING",
		Sequence:      3,
		JournalTip:    "deadbeef",
	})
	if _, err := s.Status(id); err == nil {
		t.Fatal("stale projection tip mismatch not rejected")
	}
}

func TestProjectionRejectsNegativeSequence(t *testing.T) {
	dir := t.TempDir()
	s := &Store{Root: dir, Clock: testClock()}
	id := "MISSION-NEGSEQ"
	if err := s.Create(testMission(id)); err != nil {
		t.Fatal(err)
	}
	writeProjection(t, dir, id, &Projection{
		SchemaVersion: auditSchemaVersion,
		MissionID:     id,
		MissionSHA256: missionHash(t, id),
		State:         "CREATED",
		Sequence:      -1,
		JournalTip:    "deadbeef",
	})
	if _, err := s.Status(id); err == nil {
		t.Fatal("negative projection sequence not rejected")
	}
}

func TestProjectionRejectsInvalidSequence(t *testing.T) {
	dir := t.TempDir()
	s := &Store{Root: dir, Clock: testClock()}
	id := "MISSION-BADSEQ"
	if err := s.Create(testMission(id)); err != nil {
		t.Fatal(err)
	}
	writeProjection(t, dir, id, &Projection{
		SchemaVersion: auditSchemaVersion,
		MissionID:     id,
		MissionSHA256: missionHash(t, id),
		State:         "CREATED",
		Sequence:      999,
		JournalTip:    "deadbeef",
	})
	if _, err := s.Status(id); err == nil {
		t.Fatal("ahead-of-journal projection sequence not rejected")
	}
}
