package main

import (
	"context"
	"io"
	"os"
	"path/filepath"
)

// executablePath is a seam for os.Executable; same-package tests may substitute
// it to exercise the fail-closed path. It is not selectable by callers.
var executablePath = os.Executable

func main() {
	anchor, err := executableAnchor()
	if err != nil {
		emitDiagnostic(os.Stderr, Diagnostic{Event: eventEnvironment, Command: "", ExitCode: ExitEnvironment, Detail: "cannot resolve executable location: " + err.Error()})
		os.Exit(ExitEnvironment)
	}
	os.Exit(run(os.Args[1:], anchor, os.Stdout, os.Stderr))
}

// executableAnchor returns the wrapper executable's directory without any
// caller-cwd fallback; a resolution failure fails closed with an error.
func executableAnchor() (string, error) {
	exe, err := executablePath()
	if err != nil {
		return "", err
	}
	return filepath.Dir(exe), nil
}

// run dispatches the fixed command against the governed repository root and
// returns the process exit code. It is also the in-process test entry point.
func run(args []string, anchor string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		emitDiagnostic(stderr, Diagnostic{Event: eventUsage, Command: "", ExitCode: ExitUsage, Detail: "no command"})
		return ExitUsage
	}
	command := args[0]
	if command == "help" || command == "-h" || command == "--help" {
		printHelp(stdout)
		return ExitSuccess
	}
	if runnerlessCommands[command] {
		emitDiagnostic(stderr, Diagnostic{Event: eventEnvironment, Command: command, ExitCode: ExitEnvironment, Detail: "runner not available in structure-only foundation"})
		return ExitEnvironment
	}
	script, ok := fixedCommands[command]
	if !ok {
		emitDiagnostic(stderr, Diagnostic{Event: eventUsage, Command: command, ExitCode: ExitUsage, Detail: "unknown command"})
		return ExitUsage
	}
	root, err := resolveGovernedRoot(anchor)
	if err != nil {
		emitDiagnostic(stderr, Diagnostic{Event: eventPrecondition, Command: command, ExitCode: ExitPrecondition, Detail: err.Error()})
		return ExitPrecondition
	}
	if command == "bootstrap" || command == "reset" || command == "report" {
		if cerr := classifyLocal(root); cerr != nil {
			emitDiagnostic(stderr, Diagnostic{Event: eventPrecondition, Command: command, ExitCode: ExitPrecondition, Detail: cerr.Error()})
			return ExitPrecondition
		}
	}
	absScript := filepath.Join(root, "deploy", "local", script)
	return executeRunner(context.Background(), command, absScript, stderr)
}

func printHelp(w io.Writer) {
	io.WriteString(w, "usage: dev <command>\n\n")
	io.WriteString(w, "commands:\n")
	io.WriteString(w, "  bootstrap    start the local stack and seed synthetic data\n")
	io.WriteString(w, "  reset        teardown and rebuild the local stack (local volumes only)\n")
	io.WriteString(w, "  diagnose     check the PostgreSQL authority boundary\n")
	io.WriteString(w, "  report       emit the allowlisted environment report\n")
	io.WriteString(w, "  build        (runner not available)\n")
	io.WriteString(w, "  lint         (runner not available)\n")
	io.WriteString(w, "  unit         (runner not available)\n")
	io.WriteString(w, "  schema       (runner not available)\n")
	io.WriteString(w, "  integration  (runner not available)\n")
	io.WriteString(w, "  verify       (runner not available)\n")
	io.WriteString(w, "  help         show this help\n")
}
