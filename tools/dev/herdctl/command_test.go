package main

import (
	"bytes"
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
