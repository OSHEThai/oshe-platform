package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// savedRunScript captures the production seam so tests can restore it.
var savedRunScript = runScript

// setFakeRunner replaces the execution seam for the duration of the test.
func setFakeRunner(t *testing.T, started bool, code int, err error) {
	t.Helper()
	runScript = func(ctx context.Context, path string) runResult {
		return runResult{Started: started, ExitCode: code, Err: err}
	}
	t.Cleanup(func() { runScript = savedRunScript })
}

func realRoot(t *testing.T) string {
	t.Helper()
	root, err := resolveGovernedRoot(".")
	if err != nil {
		t.Fatalf("resolve governed root from test cwd: %v", err)
	}
	return root
}

// decodeDiagnostic asserts exactly one newline-delimited JSON object and decodes it.
func decodeDiagnostic(t *testing.T, b []byte) map[string]any {
	t.Helper()
	trimmed := bytes.TrimSpace(b)
	if len(trimmed) == 0 {
		t.Fatalf("expected exactly one diagnostic line, got empty output")
	}
	if bytes.Contains(trimmed, []byte("\n")) {
		t.Fatalf("expected exactly one diagnostic line, got %q", string(b))
	}
	var d map[string]any
	if err := json.Unmarshal(trimmed, &d); err != nil {
		t.Fatalf("diagnostic not decodable JSON: %v", err)
	}
	return d
}

func TestHelpExitsZero(t *testing.T) {
	var out, errb bytes.Buffer
	code := run([]string{"help"}, realRoot(t), &out, &errb)
	if code != ExitSuccess {
		t.Fatalf("help exit = %d, want %d", code, ExitSuccess)
	}
	if out.Len() == 0 {
		t.Fatal("help produced no output")
	}
}

func TestNoArgsExitsUsage(t *testing.T) {
	var out, errb bytes.Buffer
	code := run(nil, realRoot(t), &out, &errb)
	if code != ExitUsage {
		t.Fatalf("no-args exit = %d, want %d", code, ExitUsage)
	}
	decodeDiagnostic(t, errb.Bytes())
}

func TestUnknownCommandExitsUsage(t *testing.T) {
	var out, errb bytes.Buffer
	code := run([]string{"frobnicate"}, realRoot(t), &out, &errb)
	if code != ExitUsage {
		t.Fatalf("unknown-command exit = %d, want %d", code, ExitUsage)
	}
}

func TestRunnerlessCommandExitsEnvironment(t *testing.T) {
	for _, cmd := range []string{"build", "lint", "unit", "schema", "integration", "verify"} {
		var out, errb bytes.Buffer
		code := run([]string{cmd}, realRoot(t), &out, &errb)
		if code != ExitEnvironment {
			t.Fatalf("%s exit = %d, want %d", cmd, code, ExitEnvironment)
		}
		d := decodeDiagnostic(t, errb.Bytes())
		if d["event"] != eventEnvironment {
			t.Errorf("%s event = %v, want %q", cmd, d["event"], eventEnvironment)
		}
	}
}

func TestBootstrapSuccess(t *testing.T) {
	setFakeRunner(t, true, 0, nil)
	var out, errb bytes.Buffer
	code := run([]string{"bootstrap"}, realRoot(t), &out, &errb)
	if code != ExitSuccess {
		t.Fatalf("bootstrap exit = %d, want %d", code, ExitSuccess)
	}
	if errb.Len() != 0 {
		t.Fatalf("success emitted diagnostics: %q", errb.String())
	}
}

func TestFabricatedRootRejectedExitPrecondition(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "deploy", "local"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "deploy", "local", "bootstrap.ps1"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "deploy", "local", "compose.dev.yaml"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	// Forged manifest: identity strings appear only in comments; the actual
	// fields differ. Exact parsing must reject it (never substring/comment match).
	forged := "# organization: OSHEThai\n# name: oshe-platform\nrepository:\n  organization: example\n  name: example-platform\n"
	if err := os.WriteFile(filepath.Join(dir, "repo-manifest.yaml"), []byte(forged), 0o644); err != nil {
		t.Fatal(err)
	}

	invoked := false
	runScript = func(ctx context.Context, path string) runResult {
		invoked = true
		return runResult{Started: true, ExitCode: 0}
	}
	t.Cleanup(func() { runScript = savedRunScript })

	var out, errb bytes.Buffer
	code := run([]string{"bootstrap"}, dir, &out, &errb)
	if code != ExitPrecondition {
		t.Fatalf("fabricated-root exit = %d, want %d", code, ExitPrecondition)
	}
	if invoked {
		t.Fatal("fabricated root executed its same-named script")
	}
	d := decodeDiagnostic(t, errb.Bytes())
	if d["event"] != eventPrecondition {
		t.Errorf("event = %v, want %q", d["event"], eventPrecondition)
	}
}

func TestMissingExecutableExitsEnvironment(t *testing.T) {
	setFakeRunner(t, false, 0, &exec.Error{Name: "pwsh", Err: exec.ErrNotFound})
	var out, errb bytes.Buffer
	code := run([]string{"bootstrap"}, realRoot(t), &out, &errb)
	if code != ExitEnvironment {
		t.Fatalf("missing-executable exit = %d, want %d", code, ExitEnvironment)
	}
	d := decodeDiagnostic(t, errb.Bytes())
	if d["event"] != eventRunnerMissing {
		t.Errorf("event = %v, want %q", d["event"], eventRunnerMissing)
	}
}

func TestResetMissingExecutableExitsEnvironment(t *testing.T) {
	setFakeRunner(t, false, 0, &exec.Error{Name: "pwsh", Err: exec.ErrNotFound})
	var out, errb bytes.Buffer
	code := run([]string{"reset"}, realRoot(t), &out, &errb)
	if code != ExitEnvironment {
		t.Fatalf("reset missing-executable exit = %d, want %d", code, ExitEnvironment)
	}
	d := decodeDiagnostic(t, errb.Bytes())
	if d["event"] != eventRunnerMissing {
		t.Errorf("event = %v, want %q", d["event"], eventRunnerMissing)
	}
	if d["command"] != "reset" {
		t.Errorf("command = %v, want %q", d["command"], "reset")
	}
	if d["exit_code"] != float64(ExitEnvironment) {
		t.Errorf("exit_code = %v, want %d", d["exit_code"], ExitEnvironment)
	}
}

func TestStartedNonzeroExitsWrappedFailure(t *testing.T) {
	setFakeRunner(t, true, 7, nil)
	var out, errb bytes.Buffer
	code := run([]string{"bootstrap"}, realRoot(t), &out, &errb)
	if code != ExitWrappedToolFailure {
		t.Fatalf("started-nonzero exit = %d, want %d", code, ExitWrappedToolFailure)
	}
	d := decodeDiagnostic(t, errb.Bytes())
	if d["event"] != eventRunnerFailure {
		t.Errorf("event = %v, want %q", d["event"], eventRunnerFailure)
	}
}

func TestResetFailureEmitsExactlyOneDiagnostic(t *testing.T) {
	setFakeRunner(t, true, 1, nil)
	var out, errb bytes.Buffer
	code := run([]string{"reset"}, realRoot(t), &out, &errb)
	if code != ExitWrappedToolFailure {
		t.Fatalf("reset exit = %d, want %d", code, ExitWrappedToolFailure)
	}
	// decodeDiagnostic asserts exactly one line and stable schema.
	d := decodeDiagnostic(t, errb.Bytes())
	if d["event"] != eventRunnerFailure {
		t.Errorf("event = %v, want %q", d["event"], eventRunnerFailure)
	}
}

func TestEnvCannotManufactureSuccess(t *testing.T) {
	t.Setenv("DEV_WRAPPER_TEST", "1")
	setFakeRunner(t, false, 0, &exec.Error{Name: "pwsh", Err: exec.ErrNotFound})
	var out, errb bytes.Buffer
	code := run([]string{"bootstrap"}, realRoot(t), &out, &errb)
	if code == ExitSuccess {
		t.Fatal("environment input manufactured success")
	}
	if code != ExitEnvironment {
		t.Fatalf("exit = %d, want %d", code, ExitEnvironment)
	}
}

func TestExecutableAnchorFailureFailsClosed(t *testing.T) {
	saved := executablePath
	executablePath = func() (string, error) { return "", errors.New("cannot resolve executable") }
	t.Cleanup(func() { executablePath = saved })

	if _, err := executableAnchor(); err == nil {
		t.Fatal("executable anchor failure returned no error (cwd fallback)")
	}
}

func TestClassifyLocalFailureExitsPrecondition(t *testing.T) {
	savedClassify := classifyLocal
	classifyLocal = func(root string) error { return errors.New("simulated classification failure") }
	t.Cleanup(func() { classifyLocal = savedClassify })

	var out, errb bytes.Buffer
	code := run([]string{"bootstrap"}, realRoot(t), &out, &errb)
	if code != ExitPrecondition {
		t.Fatalf("classifyLocal failure exit = %d, want %d", code, ExitPrecondition)
	}
	d := decodeDiagnostic(t, errb.Bytes())
	if d["event"] != eventPrecondition {
		t.Errorf("event = %v, want %q", d["event"], eventPrecondition)
	}
}
