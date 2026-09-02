package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

const goodEnv = "POSTGRES_DB=oshe_dev\nPOSTGRES_USER=oshe_dev\nPOSTGRES_PASSWORD=oshe_dev_synthetic_only\nMEILI_MASTER_KEY=oshe_dev_synthetic_only\n"

// canonicalFile reads the sealed canonical static file from the repository.
func canonicalFile(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "deploy", "local", name))
	if err != nil {
		t.Fatalf("read canonical %s: %v", name, err)
	}
	return string(data)
}

func writeLocalFiles(t *testing.T, dir, compose, env, seed string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, "deploy", "local"), 0o755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(dir, "deploy", "local", "compose.dev.yaml"), []byte(compose), 0o644)
	os.WriteFile(filepath.Join(dir, "deploy", "local", ".env.example"), []byte(env), 0o644)
	os.WriteFile(filepath.Join(dir, "deploy", "local", "seed-synthetic.ps1"), []byte(seed), 0o644)
}

func TestClassifyLocalPass(t *testing.T) {
	dir := t.TempDir()
	writeLocalFiles(t, dir, canonicalFile(t, "compose.dev.yaml"), goodEnv, canonicalFile(t, "seed-synthetic.ps1"))
	if err := classifyLocal(dir); err != nil {
		t.Fatalf("classifyLocal = %v, want nil", err)
	}
}

// I012-SEC-001-R1: compose tampering (unknown/personal/credential) fails the sealed digest.

func TestClassifyLocalRejectsComposeTampering(t *testing.T) {
	canonical := canonicalFile(t, "compose.dev.yaml")
	cases := map[string]string{
		"unknown_service": canonical + "  mongodb:\n    image: mongo:latest\n",
		"personal_data":   canonical + "# customer@example.com\n",
		"credential":      canonical + "    password: REAL_SECRET\n",
	}
	for name, compose := range cases {
		dir := t.TempDir()
		writeLocalFiles(t, dir, compose, goodEnv, canonicalFile(t, "seed-synthetic.ps1"))
		if err := classifyLocal(dir); !errors.Is(err, errNotCanonical) {
			t.Errorf("%s: classifyLocal = %v, want %v", name, err, errNotCanonical)
		}
	}
}

// I012-SEC-001-R1 + SEC-003-R1: seed tampering (credential and system-qualified aliases).

func TestClassifyLocalRejectsSeedTampering(t *testing.T) {
	canonical := canonicalFile(t, "seed-synthetic.ps1")
	cases := map[string]string{
		"credential":       canonical + "$p = 'REAL_SECRET'\n",
		"datetime_now":     canonical + "$t = [System.DateTime]::Now\n",
		"guid_newguid":     canonical + "$g = [System.Guid]::NewGuid()\n",
		"env_machinename":  canonical + "$n = [System.Environment]::MachineName\n",
	}
	for name, seed := range cases {
		dir := t.TempDir()
		writeLocalFiles(t, dir, canonicalFile(t, "compose.dev.yaml"), goodEnv, seed)
		if err := classifyLocal(dir); !errors.Is(err, errNotCanonical) {
			t.Errorf("%s: classifyLocal = %v, want %v", name, err, errNotCanonical)
		}
	}
}

// Env allowlist tampering remains rejected.

func TestClassifyLocalRejectsEnvTampering(t *testing.T) {
	cases := map[string]string{
		"unknown_key":   "AWS_ACCESS_KEY_ID=AKIA1234\n",
		"non_synthetic": "POSTGRES_PASSWORD=REAL_SECRET\n",
		"duplicate":     goodEnv + "POSTGRES_DB=oshe_dev\n",
		"malformed":     "NOT_A_KEY_VALUE\n",
	}
	for name, env := range cases {
		dir := t.TempDir()
		writeLocalFiles(t, dir, canonicalFile(t, "compose.dev.yaml"), env, canonicalFile(t, "seed-synthetic.ps1"))
		if err := classifyLocal(dir); !errors.Is(err, errSecretDetected) {
			t.Errorf("%s: classifyLocal = %v, want %v", name, err, errSecretDetected)
		}
	}
}

// Runner unreachable for a tampered (unsafe) root.

func TestRunDoesNotInvokeRunnerForUnsafeRoot(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "repo-manifest.yaml"), []byte("repository:\n  organization: OSHEThai\n  name: oshe-platform\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	tampered := canonicalFile(t, "compose.dev.yaml") + "# production_endpoint\n"
	writeLocalFiles(t, dir, tampered, goodEnv, canonicalFile(t, "seed-synthetic.ps1"))

	invoked := false
	runScript = func(ctx context.Context, path string) runResult {
		invoked = true
		return runResult{Started: true, ExitCode: 0}
	}
	t.Cleanup(func() { runScript = savedRunScript })

	var out, errb bytes.Buffer
	code := run([]string{"bootstrap"}, dir, &out, &errb)
	if code != ExitPrecondition {
		t.Fatalf("run exit = %d, want %d", code, ExitPrecondition)
	}
	if invoked {
		t.Fatal("runner invoked for tampered/unsafe root")
	}
	decodeDiagnostic(t, errb.Bytes())
}

func TestClassifyLocalRejectsMissingFile(t *testing.T) {
	dir := t.TempDir()
	if err := classifyLocal(dir); err == nil {
		t.Fatal("classifyLocal = nil, want error (uncertain)")
	}
}
