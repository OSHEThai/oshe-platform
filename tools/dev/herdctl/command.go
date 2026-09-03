package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
)

// Stable redacted diagnostic categories.
const (
	catUsage      = "usage"
	catInvalid    = "invalid"
	catDurability = "durability_unsupported"
)

func dispatch(args []string, store *Store, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return fail(stderr, "", catUsage)
	}
	cmd := args[0]
	switch cmd {
	case "help", "-h", "--help":
		return cmdHelp(stdout)
	case "create":
		return cmdCreate(args, store, stdout, stderr)
	case "validate", "start", "pause", "resume", "cancel", "archive":
		return cmdTransition(args, cmd, store, stdout, stderr)
	case "dispatch", "monitor", "timeout", "dispatch-cancel", "restart":
		return cmdDispatchLifecycle(args, cmd, store, stdout, stderr)
	case "status":
		return cmdStatus(args, store, stdout, stderr)
	default:
		return fail(stderr, cmd, catUsage)
	}
}

func cmdHelp(stdout io.Writer) int {
	emitJSON(stdout, map[string]any{
		"event":    "help",
		"commands": []string{"create", "validate", "start", "status", "pause", "resume", "cancel", "archive", "help"},
	})
	return 0
}

func cmdCreate(args []string, store *Store, stdout, stderr io.Writer) int {
	path, err := parseFileFlag(args)
	if err != nil {
		return fail(stderr, "create", catUsage)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fail(stderr, "create", catInvalid)
	}
	m, err := decodeMission(data)
	if err != nil {
		return fail(stderr, "create", catInvalid)
	}
	if err := store.Create(m); err != nil {
		return fail(stderr, "create", errorCategory(err))
	}
	emitJSON(stdout, map[string]any{"event": "created", "mission_id": m.ID, "state": "CREATED"})
	return 0
}

func cmdTransition(args []string, cmd string, store *Store, stdout, stderr io.Writer) int {
	id, err := parseIDFlag(args)
	if err != nil {
		return fail(stderr, cmd, catUsage)
	}
	if err := store.Transition(id, cmd); err != nil {
		return fail(stderr, cmd, errorCategory(err))
	}
	state, err := store.Status(id)
	if err != nil {
		return fail(stderr, cmd, errorCategory(err))
	}
	emitJSON(stdout, map[string]any{"event": "transitioned", "mission_id": id, "state": string(state)})
	return 0
}

func cmdStatus(args []string, store *Store, stdout, stderr io.Writer) int {
	id, err := parseIDFlag(args)
	if err != nil {
		return fail(stderr, "status", catUsage)
	}
	state, err := store.Status(id)
	if err != nil {
		return fail(stderr, "status", errorCategory(err))
	}
	emitJSON(stdout, map[string]any{"event": "status", "mission_id": id, "state": string(state)})
	return 0
}

// errorCategory maps an error to a stable redacted category.
func errorCategory(err error) string {
	if errors.Is(err, errDurabilityUnsupported) {
		return catDurability
	}
	return catInvalid
}

func parseFileFlag(args []string) (string, error) {
	var file string
	seen := false
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--file":
			if i+1 >= len(args) {
				return "", fmt.Errorf("missing value for --file")
			}
			if seen {
				return "", fmt.Errorf("duplicate --file")
			}
			file = args[i+1]
			seen = true
			i++
		default:
			return "", fmt.Errorf("unexpected argument %q", args[i])
		}
	}
	if file == "" {
		return "", fmt.Errorf("--file required")
	}
	return file, nil
}

func parseIDFlag(args []string) (string, error) {
	var id string
	seen := false
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--id":
			if i+1 >= len(args) {
				return "", fmt.Errorf("missing value for --id")
			}
			if seen {
				return "", fmt.Errorf("duplicate --id")
			}
			id = args[i+1]
			seen = true
			i++
		default:
			return "", fmt.Errorf("unexpected argument %q", args[i])
		}
	}
	if id == "" {
		return "", fmt.Errorf("--id required")
	}
	if !idPattern.MatchString(id) {
		return "", fmt.Errorf("invalid mission id")
	}
	return id, nil
}

// fail emits one decodable diagnostic with a stable redacted category.
func fail(stderr io.Writer, cmd, category string) int {
	emitJSON(stderr, map[string]any{"event": "error", "command": cmd, "category": category})
	return 1
}

func emitJSON(w io.Writer, v any) {
	b, _ := json.Marshal(v)
	fmt.Fprintln(w, string(b))
}

func cmdDispatchLifecycle(args []string, cmd string, store *Store, stdout, stderr io.Writer) int {
	id, err := parseIDFlag(args)
	if err != nil {
		return fail(stderr, cmd, catUsage)
	}
	d := &LocalDispatcher{Store: store}
	var res *DispatchResult
	switch cmd {
	case "dispatch":
		err = d.Dispatch(id)
	case "monitor":
		res, err = d.Monitor(id)
	case "timeout":
		err = d.Timeout(id)
	case "dispatch-cancel":
		err = d.Cancel(id)
	case "restart":
		err = d.Restart(id)
	}
	if err != nil {
		return fail(stderr, cmd, errorCategory(err))
	}
	if cmd == "monitor" {
		emitJSON(stdout, res)
	} else {
		emitJSON(stdout, map[string]any{"event": cmd, "mission_id": id})
	}
	return 0
}
