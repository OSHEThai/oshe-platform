package main

import (
	"encoding/json"
	"fmt"
	"io"
)

func cmdSkillSync(args []string, cmd string, c *SkillSyncController, stdout, stderr io.Writer) int {
	if len(args) < 2 {
		return 1
	}
	id := args[1]
	var err error
	switch cmd {
	case "sync-validate":
		err = c.Validate(id)
	case "sync-activate":
		err = c.Activate(id, "1.0.0")
	case "sync-cleanup":
		err = c.Cleanup(id)
	default:
		return 1
	}

	if err != nil {
		b, _ := json.Marshal(map[string]any{"event": "error", "command": cmd, "error": err.Error()})
		fmt.Fprintln(stderr, string(b))
		return 1
	}

	b, _ := json.Marshal(map[string]any{"event": cmd, "skill_id": id})
	fmt.Fprintln(stdout, string(b))
	return 0
}
