package main

import (
	"os"
	"testing"
)

func TestSkillSyncController(t *testing.T) {
	c := &SkillSyncController{Root: t.TempDir()}
	id := "skill-1"

	// Validate
	if err := c.Validate(id); err != nil {
		t.Fatal(err)
	}

	// Activate
	if err := c.Activate(id, "1.0.0"); err != nil {
		t.Fatal(err)
	}

	// Verify file exists
	if _, err := os.Stat(c.manifestPath(id)); err != nil {
		t.Fatal("manifest not created")
	}

	// Cleanup
	if err := c.Cleanup(id); err != nil {
		t.Fatal(err)
	}

	// Verify file deleted
	if _, err := os.Stat(c.manifestPath(id)); err == nil {
		t.Fatal("manifest not deleted")
	}
}
