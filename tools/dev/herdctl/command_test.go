package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestCmdSkillSync(t *testing.T) {
	c := &SkillSyncController{Root: t.TempDir()}
	id := "skill-cmd-1"

	var out, errb bytes.Buffer

	// Validate
	if code := cmdSkillSync([]string{"sync-validate", id}, "sync-validate", c, &out, &errb); code != 0 {
		t.Fatal("sync-validate failed")
	}

	// Activate
	if code := cmdSkillSync([]string{"sync-activate", id}, "sync-activate", c, &out, &errb); code != 0 {
		t.Fatal("sync-activate failed")
	}

	// Cleanup
	if code := cmdSkillSync([]string{"sync-cleanup", id}, "sync-cleanup", c, &out, &errb); code != 0 {
		t.Fatal("sync-cleanup failed")
	}

	// Invalid Command
	if code := cmdSkillSync([]string{"unknown", id}, "unknown", c, &out, &errb); code == 0 {
		t.Fatal("unknown command should fail")
	}
}

func TestCmdAdapterStub(t *testing.T) {
	var out, errb bytes.Buffer

	// Missing args
	if code := cmdAdapterStub([]string{"adapter-stub"}, &out, &errb); code == 0 {
		t.Fatal("expected failure for missing args")
	}

	// Unknown adapter
	if code := cmdAdapterStub([]string{"adapter-stub", "unknown", "payload"}, &out, &errb); code == 0 {
		t.Fatal("expected failure for unknown adapter")
	}

	// Local stub success
	if code := cmdAdapterStub([]string{"adapter-stub", "local-stub", "ok"}, &out, &errb); code != 0 {
		t.Fatal("expected success for local-stub")
	}
	if !strings.Contains(out.String(), "stub-success") {
		t.Fatal("missing stub-success output")
	}

	// Local stub failure
	if code := cmdAdapterStub([]string{"adapter-stub", "local-stub", "fail"}, &out, &errb); code == 0 {
		t.Fatal("expected failure for local-stub fail payload")
	}

	out.Reset()
	// Offline mock success
	if code := cmdAdapterStub([]string{"adapter-stub", "offline-mock", "ok"}, &out, &errb); code != 0 {
		t.Fatal("expected success for offline-mock")
	}
	if !strings.Contains(out.String(), "mock-success") {
		t.Fatal("missing mock-success output")
	}

	// Offline mock failure
	if code := cmdAdapterStub([]string{"adapter-stub", "offline-mock", "error"}, &out, &errb); code == 0 {
		t.Fatal("expected failure for offline-mock error payload")
	}
}

func TestCmdQuotaSim(t *testing.T) {
	var out, errb bytes.Buffer

	// Reset before test
	cmdQuotaSim([]string{"quota-sim", "reset"}, &out, &errb)
	out.Reset()

	// Acquire normal
	if code := cmdQuotaSim([]string{"quota-sim", "acquire", "normal"}, &out, &errb); code != 0 {
		t.Fatal("expected success")
	}
	if !strings.Contains(out.String(), "acquired") {
		t.Fatal("expected acquired output")
	}

	// Acquire high (exceed quota)
	if code := cmdQuotaSim([]string{"quota-sim", "acquire", "high"}, &out, &errb); code == 0 {
		t.Fatal("expected failure on high cost")
	}

	// Release
	out.Reset()
	if code := cmdQuotaSim([]string{"quota-sim", "release"}, &out, &errb); code != 0 {
		t.Fatal("expected success on release")
	}

	// Stop
	out.Reset()
	cmdQuotaSim([]string{"quota-sim", "stop"}, &out, &errb)
	if code := cmdQuotaSim([]string{"quota-sim", "acquire", "normal"}, &out, &errb); code == 0 {
		t.Fatal("expected failure after stop")
	}
}
