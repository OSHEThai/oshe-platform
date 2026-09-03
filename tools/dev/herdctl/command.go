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
	case "int-verify", "int-prepare", "int-review-approve", "int-review-remediate", "int-handoff", "int-draft-pr", "int-status":
		return cmdIntegrationLifecycle(args, cmd, store, stdout, stderr)
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

func cmdIntegrationLifecycle(args []string, cmd string, store *Store, stdout, stderr io.Writer) int {
	id, err := parseIDFlag(args)
	if err != nil {
		return fail(stderr, cmd, catUsage)
	}
	c := &IntegrationController{Store: store}
	var res *IntegrationStatus
	switch cmd {
	case "int-verify":
		// Mock dependency list for CLI
		err = c.VerifyDependencies(id, []string{"mock-dep"})
	case "int-prepare":
		err = c.PrepareCandidate(id)
	case "int-review-approve":
		err = c.SubmitReview(id, true)
	case "int-review-remediate":
		err = c.SubmitReview(id, false)
	case "int-handoff":
		err = c.Handoff(id)
	case "int-draft-pr":
		err = c.RequestDraftPR(id)
	case "int-status":
		res, err = c.Status(id)
	}
	if err != nil {
		return fail(stderr, cmd, errorCategory(err))
	}
	if cmd == "int-status" {
		emitJSON(stdout, res)
	} else {
		emitJSON(stdout, map[string]any{"event": cmd, "mission_id": id})
	}
	return 0
}

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

func cmdAdapterStub(args []string, stdout, stderr io.Writer) int {
	if len(args) < 3 {
		return 1
	}
	adapterName := args[1]
	payload := args[2]

	var adapter ProviderAdapter
	switch adapterName {
	case "local-stub":
		adapter = &LocalStubAdapter{}
	case "offline-mock":
		adapter = &OfflineMockAdapter{}
	default:
		fmt.Fprintln(stderr, "unknown adapter")
		return 1
	}

	res, err := adapter.Dispatch(payload)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}

	fmt.Fprintln(stdout, res)
	return 0
}

var globalQuota = NewQuotaController(100, 2)

func cmdQuotaSim(args []string, stdout, stderr io.Writer) int {
	if len(args) < 2 {
		return 1
	}
	action := args[1]
	switch action {
	case "acquire":
		if len(args) < 3 {
			return 1
		}
		cost := 10 // default cost for simulation
		if args[2] == "high" {
			cost = 150
		}
		if err := globalQuota.Acquire(cost); err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		fmt.Fprintln(stdout, "acquired")
	case "release":
		globalQuota.Release()
		fmt.Fprintln(stdout, "released")
	case "stop":
		globalQuota.SetStopBehavior(true)
		fmt.Fprintln(stdout, "stopped")
	case "reset":
		globalQuota.Reset()
		fmt.Fprintln(stdout, "reset")
	default:
		return 1
	}
	return 0
}
