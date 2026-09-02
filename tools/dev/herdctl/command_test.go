package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func runCmd(t *testing.T, s *Store, args ...string) (int, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	code := run(args, s.Root, s.Clock, &out, &errb)
	return code, out.String(), errb.String()
}

func writeMissionFile(t *testing.T, m *Mission) string {
	t.Helper()
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "mission.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestUnknownCommand(t *testing.T) {
	s := newStore(t)
	code, _, errb := runCmd(t, s, "frobnicate")
	if code == 0 {
		t.Fatal("unknown command returned 0")
	}
	var d map[string]any
	if err := json.Unmarshal([]byte(errb), &d); err != nil {
		t.Fatalf("diagnostic not JSON: %v", err)
	}
	if d["command"] != "frobnicate" {
		t.Errorf("command = %v, want frobnicate", d["command"])
	}
}

func TestUnknownRootFlagRejected(t *testing.T) {
	s := newStore(t)
	code, _, _ := runCmd(t, s, "--root", "/tmp/x", "status", "--id", "M-1")
	if code == 0 {
		t.Fatal("root-selection flag accepted")
	}
}

func TestInvalidIDTraversal(t *testing.T) {
	s := newStore(t)
	for _, id := range []string{"../etc/passwd", "a/b", "a\\b", ".", "..", "MISSION-001.json"} {
		code, _, _ := runCmd(t, s, "status", "--id", id)
		if code == 0 {
			t.Errorf("traversal/malformed id %q accepted", id)
		}
	}
}

func TestDuplicateFlagRejected(t *testing.T) {
	s := newStore(t)
	code, _, _ := runCmd(t, s, "status", "--id", "M-1", "--id", "M-2")
	if code == 0 {
		t.Fatal("duplicate --id accepted")
	}
}

func TestMissingIDRejected(t *testing.T) {
	s := newStore(t)
	code, _, _ := runCmd(t, s, "status")
	if code == 0 {
		t.Fatal("missing --id accepted")
	}
}

func TestUnexpectedPositionalRejected(t *testing.T) {
	s := newStore(t)
	code, _, _ := runCmd(t, s, "status", "--id", "M-1", "extra")
	if code == 0 {
		t.Fatal("extra positional accepted")
	}
}

func TestCreateThenStatus(t *testing.T) {
	s := newStore(t)
	path := writeMissionFile(t, testMission("MISSION-CLI"))
	code, out, errb := runCmd(t, s, "create", "--file", path)
	if code != 0 {
		t.Fatalf("create exit %d: %s", code, errb)
	}
	var created map[string]any
	if err := json.Unmarshal([]byte(out), &created); err != nil {
		t.Fatalf("create output not JSON: %v", err)
	}
	if created["state"] != "CREATED" {
		t.Errorf("created state = %v, want CREATED", created["state"])
	}
	code, out, errb = runCmd(t, s, "status", "--id", "MISSION-CLI")
	if code != 0 {
		t.Fatalf("status exit %d: %s", code, errb)
	}
	var st map[string]any
	if err := json.Unmarshal([]byte(out), &st); err != nil {
		t.Fatalf("status output not JSON: %v", err)
	}
	if st["state"] != "CREATED" {
		t.Errorf("state = %v, want CREATED", st["state"])
	}
}

func TestDuplicateCreateRejected(t *testing.T) {
	s := newStore(t)
	path := writeMissionFile(t, testMission("MISSION-DUP"))
	if code, _, _ := runCmd(t, s, "create", "--file", path); code != 0 {
		t.Fatal("first create failed")
	}
	if code, _, _ := runCmd(t, s, "create", "--file", path); code == 0 {
		t.Fatal("duplicate create accepted")
	}
}

func TestFullCLILifecycle(t *testing.T) {
	s := newStore(t)
	id := "MISSION-CLIFULL"
	path := writeMissionFile(t, testMission(id))
	if code, _, errb := runCmd(t, s, "create", "--file", path); code != 0 {
		t.Fatalf("create: %s", errb)
	}
	for _, cmd := range []string{"validate", "start", "pause", "resume", "cancel", "archive"} {
		code, out, errb := runCmd(t, s, cmd, "--id", id)
		if code != 0 {
			t.Fatalf("%s exit %d: %s", cmd, code, errb)
		}
		var r map[string]any
		if err := json.Unmarshal([]byte(out), &r); err != nil {
			t.Fatalf("%s output not JSON: %v", cmd, err)
		}
	}
	code, out, _ := runCmd(t, s, "status", "--id", id)
	if code != 0 {
		t.Fatal("status failed")
	}
	var st map[string]any
	json.Unmarshal([]byte(out), &st)
	if st["state"] != "ARCHIVED" {
		t.Errorf("final state = %v, want ARCHIVED", st["state"])
	}
}

func TestHelpIsJSON(t *testing.T) {
	s := newStore(t)
	code, out, _ := runCmd(t, s, "help")
	if code != 0 {
		t.Fatal("help exit nonzero")
	}
	var h map[string]any
	if err := json.Unmarshal([]byte(out), &h); err != nil {
		t.Fatalf("help output not JSON: %v", err)
	}
}

// CODE-006 diagnostic redaction.

func TestDiagnosticRedaction(t *testing.T) {
	s := newStore(t)
	m := testMission("MISSION-SECRET")
	m.DataClassification = "SECRET-SENTINEL-12345"
	path := writeMissionFile(t, m)
	code, _, errb := runCmd(t, s, "create", "--file", path)
	if code == 0 {
		t.Fatal("expected failure for invalid classification")
	}
	if strings.Contains(errb, "SECRET-SENTINEL-12345") {
		t.Fatal("diagnostic leaked mission content")
	}
}

func TestMissionRejectsTrailingDelimiter(t *testing.T) {
	data := []byte(`{"contract_type":"mission","contract_version":"1.0.0","id":"M-1","title":"t","goal":"g","non_goals":["n"],"risk_class":"R2","base_commit":"0123456789abcdef0123456789abcdef01234567","human_decisions":[]}}`)
	if _, err := decodeMission(data); err == nil {
		t.Fatal("trailing closing delimiter accepted")
	}
}

func TestDiagnosticRedactsUnknownField(t *testing.T) {
	s := newStore(t)
	m := testMission("MISSION-SECRET2")
	data, _ := json.Marshal(m)
	var raw map[string]any
	json.Unmarshal(data, &raw)
	raw["SECRET-SENTINEL-12345"] = "x"
	tampered, _ := json.Marshal(raw)
	path := filepath.Join(t.TempDir(), "mission.json")
	os.WriteFile(path, tampered, 0o644)
	code, _, errb := runCmd(t, s, "create", "--file", path)
	if code == 0 {
		t.Fatal("expected failure for unknown field")
	}
	if strings.Contains(errb, "SECRET-SENTINEL-12345") {
		t.Fatal("diagnostic leaked attacker-controlled field name")
	}
}
