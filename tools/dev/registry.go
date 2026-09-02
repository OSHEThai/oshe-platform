package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Fixed command registry: command name -> governed local script. Commands not
// listed here (build, lint, unit, schema, integration, verify) have no runner
// yet in the structure-only foundation and resolve to ExitEnvironment(5).
var fixedCommands = map[string]string{
	"bootstrap": "bootstrap.ps1",
	"reset":     "teardown-rebuild.ps1",
	"diagnose":  "check-postgres-authority.ps1",
	"report":    "report-environment.ps1",
}

var runnerlessCommands = map[string]bool{
	"build": true, "lint": true, "unit": true,
	"schema": true, "integration": true, "verify": true,
}

// Exact governed-repository identity (compared field-for-field, never by
// substring or comment text).
const (
	governedOrg  = "OSHEThai"
	governedName = "oshe-platform"
)

var errNotGoverned = errors.New("governed repository root not found or identity mismatch")

// repoIdentity is the exact parsed governed identity.
type repoIdentity struct {
	Organization string
	Name         string
}

// parseRepoIdentity extracts the exact organization and name fields from a
// repo-manifest.yaml. Comment lines and inline comments are ignored; values are
// compared exactly by the caller, never substring-matched.
func parseRepoIdentity(data []byte) repoIdentity {
	var id repoIdentity
	inRepository := false
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimRight(raw, "\r")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " \t"))
		if !inRepository {
			if indent == 0 && trimmed == "repository:" {
				inRepository = true
			}
			continue
		}
		if indent == 0 {
			inRepository = false
			continue
		}
		key, value, ok := splitKV(trimmed)
		if !ok {
			continue
		}
		switch key {
		case "organization":
			id.Organization = value
		case "name":
			id.Name = value
		}
	}
	return id
}

// splitKV parses a "key: value" pair, stripping inline comments and quotes.
func splitKV(trimmed string) (key, value string, ok bool) {
	idx := strings.Index(trimmed, ":")
	if idx < 0 {
		return "", "", false
	}
	key = strings.TrimSpace(trimmed[:idx])
	value = strings.TrimSpace(trimmed[idx+1:])
	if ci := strings.Index(value, " #"); ci >= 0 {
		value = strings.TrimSpace(value[:ci])
	}
	value = strings.Trim(value, `"'`)
	return key, value, true
}

// runResult reports the outcome of launching a governed script.
type runResult struct {
	Started  bool  // interpreter was found and launched
	ExitCode int   // child exit code, meaningful only when Started
	Err      error // non-nil only when the interpreter could not be launched
}

// runScript is the in-process execution seam. Production launches pwsh against
// the absolute governed script path. Tests substitute a deterministic fake; no
// environment or argument input can activate any substitution.
var runScript = func(ctx context.Context, scriptPath string) runResult {
	cmd := exec.CommandContext(ctx, "pwsh", "-NoProfile", "-File", scriptPath)
	if e := cmd.Run(); e != nil {
		var execErr *exec.Error
		if errors.As(e, &execErr) {
			return runResult{Started: false, Err: execErr}
		}
		var exitErr *exec.ExitError
		if errors.As(e, &exitErr) {
			return runResult{Started: true, ExitCode: exitErr.ExitCode()}
		}
		return runResult{Started: false, Err: e}
	}
	return runResult{Started: true, ExitCode: 0}
}

// resolveGovernedRoot resolves the governed repository root by walking up from
// anchor until repo-manifest.yaml with the exact pinned identity is found, then
// requiring the governed local-stack compose file. The anchor is independent of
// the caller working directory.
func resolveGovernedRoot(anchor string) (string, error) {
	abs, err := filepath.Abs(anchor)
	if err != nil {
		return "", err
	}
	for dir := abs; ; dir = filepath.Dir(dir) {
		data, readErr := os.ReadFile(filepath.Join(dir, "repo-manifest.yaml"))
		if readErr == nil {
			id := parseRepoIdentity(data)
			if id.Organization == governedOrg && id.Name == governedName {
				if _, statErr := os.Stat(filepath.Join(dir, "deploy", "local", "compose.dev.yaml")); statErr == nil {
					return dir, nil
				}
			}
			return "", errNotGoverned
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errNotGoverned
		}
	}
}

// executeRunner runs a governed script and maps the outcome to the exit-code
// contract, emitting exactly one diagnostic on failure.
func executeRunner(ctx context.Context, command, scriptPath string, stderr io.Writer) int {
	res := runScript(ctx, scriptPath)
	if res.Err != nil {
		emitDiagnostic(stderr, Diagnostic{Event: eventRunnerMissing, Command: command, ExitCode: ExitEnvironment, Detail: res.Err.Error()})
		return ExitEnvironment
	}
	if res.Started && res.ExitCode != 0 {
		emitDiagnostic(stderr, Diagnostic{Event: eventRunnerFailure, Command: command, ExitCode: ExitWrappedToolFailure, Detail: fmt.Sprintf("exit code %d", res.ExitCode)})
		return ExitWrappedToolFailure
	}
	return ExitSuccess
}
