package main

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestExitCodeConstants(t *testing.T) {
	cases := []struct {
		name string
		got  int
		want int
	}{
		{"ExitSuccess", ExitSuccess, 0},
		{"ExitWrappedToolFailure", ExitWrappedToolFailure, 1},
		{"ExitUsage", ExitUsage, 2},
		{"ExitPrecondition", ExitPrecondition, 3},
		{"ExitContractFailure", ExitContractFailure, 4},
		{"ExitEnvironment", ExitEnvironment, 5},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s = %d, want %d", c.name, c.got, c.want)
		}
	}
}

func TestDiagnosticStableSchema(t *testing.T) {
	var buf bytes.Buffer
	emitDiagnostic(&buf, Diagnostic{Event: eventRunnerFailure, Command: "reset", ExitCode: ExitWrappedToolFailure, Detail: "exit code 7"})

	line := buf.String()
	if line == "" || line[len(line)-1] != '\n' {
		t.Fatalf("expected a newline-terminated diagnostic, got %q", line)
	}

	var d map[string]any
	if err := json.Unmarshal([]byte(line), &d); err != nil {
		t.Fatalf("diagnostic is not decodable JSON: %v", err)
	}
	for _, k := range []string{"event", "command", "exit_code", "detail"} {
		if _, ok := d[k]; !ok {
			t.Errorf("diagnostic missing stable field %q", k)
		}
	}
	if d["event"] != eventRunnerFailure {
		t.Errorf("event = %v, want %q", d["event"], eventRunnerFailure)
	}
}
