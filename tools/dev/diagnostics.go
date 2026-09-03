package main

import (
	"encoding/json"
	"fmt"
	"io"
)

// Diagnostic is the stable machine-readable wrapper record. Each failure emits
// exactly one newline-delimited JSON object on the governed stderr stream.
type Diagnostic struct {
	Event    string `json:"event"`
	Command  string `json:"command"`
	ExitCode int    `json:"exit_code"`
	Detail   string `json:"detail"`
}

// Diagnostic event names (stable).
const (
	eventRunnerMissing = "runner_missing"
	eventRunnerFailure = "runner_failure"
	eventPrecondition  = "precondition"
	eventUsage         = "usage"
	eventEnvironment   = "environment"
)

// emitDiagnostic writes exactly one newline-delimited JSON object.
func emitDiagnostic(w io.Writer, d Diagnostic) {
	b, err := json.Marshal(d)
	if err != nil {
		fmt.Fprintf(w, "{\"event\":\"internal_error\",\"command\":\"%s\",\"exit_code\":5,\"detail\":\"diagnostic marshal failed\"}\n", d.Command)
		return
	}
	fmt.Fprintf(w, "%s\n", b)
}
