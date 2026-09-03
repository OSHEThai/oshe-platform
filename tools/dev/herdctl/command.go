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
