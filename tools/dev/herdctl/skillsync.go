package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

type SkillManifest struct {
	ID        string   `json:"id"`
	Version   string   `json:"version"`
	Activated bool     `json:"activated"`
	Artifacts []string `json:"artifacts"`
}

type SkillSyncController struct {
	Root string
}

func (c *SkillSyncController) manifestPath(id string) string {
	return filepath.Join(c.Root, id+".json")
}

func (c *SkillSyncController) Validate(id string) error {
	if id == "" {
		return errors.New("invalid skill id")
	}
	return nil
}

func (c *SkillSyncController) Activate(id, version string) error {
	if err := c.Validate(id); err != nil {
		return err
	}
	manifest := SkillManifest{
		ID:        id,
		Version:   version,
		Activated: true,
		Artifacts: []string{"bundle.zip"},
	}
	b, err := json.Marshal(manifest)
	if err != nil {
		return err
	}
	path := c.manifestPath(id)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

func (c *SkillSyncController) Cleanup(id string) error {
	path := c.manifestPath(id)
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return nil
}
