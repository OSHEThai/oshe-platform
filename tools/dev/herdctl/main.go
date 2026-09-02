package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

func main() {
	root, err := defaultRoot()
	if err != nil {
		emitJSON(os.Stderr, map[string]any{"event": "error", "command": "", "detail": "cannot resolve user-state root"})
		os.Exit(1)
	}
	os.Exit(run(os.Args[1:], root, time.Now, os.Stdout, os.Stderr))
}

// defaultRoot resolves the fixed platform user-state store root, failing closed
// when the platform configuration directory is unavailable.
func defaultRoot() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil || base == "" {
		return "", fmt.Errorf("user config directory unavailable")
	}
	return filepath.Join(base, "oshethai", "herdctl", "v1"), nil
}

// run constructs the store and dispatches. It is the in-process test entry point
// and the only root-injection path (tests pass an isolated root directly).
func run(args []string, root string, clock func() time.Time, stdout, stderr io.Writer) int {
	store := &Store{Root: root, Clock: clock}
	return dispatch(args, store, stdout, stderr)
}
