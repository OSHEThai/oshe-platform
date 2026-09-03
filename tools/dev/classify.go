package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Classification sentinel errors (fail closed).
var (
	errTargetNotIdentified     = errors.New("local disposable target not positively identified")
	errSecretDetected          = errors.New("secret, credential, or unclassified data detected")
	errNotCanonical            = errors.New("governed compose or seed is not the sealed canonical content")
	errClassificationUncertain = errors.New("data classification uncertain")
)

// Sealed canonical SHA-256 digests. Compose and seed are accepted only by full
// digest equality; no substring or regex acceptance is used for the final decision.
const (
	canonicalComposeSHA256 = "3e10f7e5768c4eba31ef6d14b56a031c279f76c18ac04c2ffa4ff7c00e45c000"
	canonicalSeedSHA256    = "f9d68daa5c0cc6dbe693d30497d46fff3400484c3b4fcbd1d9846f2152414d77"
)

// Closed env-key allowlist: key -> sole permitted synthetic value.
var envAllowlist = map[string]string{
	"POSTGRES_DB":       "oshe_dev",
	"POSTGRES_USER":     "oshe_dev",
	"POSTGRES_PASSWORD": "oshe_dev_synthetic_only",
	"MEILI_MASTER_KEY":  "oshe_dev_synthetic_only",
}

// classifyLocal runs the fail-closed data-classification guard over the governed
// root. It returns nil only when compose and seed are byte-identical to the
// sealed canonical content, the env matches the closed allowlist, and the target
// identity is unambiguous.
func classifyLocal(root string) error {
	compose, err := readLocalFile(root, "compose.dev.yaml")
	if err != nil {
		return err
	}
	env, err := readLocalFile(root, ".env.example")
	if err != nil {
		return err
	}
	seed, err := readLocalFile(root, "seed-synthetic.ps1")
	if err != nil {
		return err
	}

	if sha256Hex(compose) != canonicalComposeSHA256 {
		return errNotCanonical
	}
	if sha256Hex(seed) != canonicalSeedSHA256 {
		return errNotCanonical
	}
	if err := checkEnvAllowlist(env); err != nil {
		return err
	}
	if err := checkTargetIdentity(compose); err != nil {
		return err
	}
	return nil
}

// readLocalFile reads a governed deploy/local file; any failure fails closed.
func readLocalFile(root, name string) (string, error) {
	data, err := os.ReadFile(filepath.Join(root, "deploy", "local", name))
	if err != nil {
		return "", fmt.Errorf("%w: %s: %v", errClassificationUncertain, name, err)
	}
	return string(data), nil
}

// sha256Hex returns the hex-encoded SHA-256 of s.
func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// checkTargetIdentity requires exactly one canonical top-level compose name.
func checkTargetIdentity(compose string) error {
	names := regexp.MustCompile(`(?m)^name:\s*(\S+)\s*$`).FindAllStringSubmatch(compose, -1)
	if len(names) != 1 || names[0][1] != "oshe-local" {
		return errTargetNotIdentified
	}
	return nil
}

// checkEnvAllowlist parses env against the closed allowlist, rejecting unknown,
// malformed, duplicate, or non-synthetic entries.
func checkEnvAllowlist(env string) error {
	seen := map[string]bool{}
	for _, line := range strings.Split(env, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		kv := strings.SplitN(line, "=", 2)
		if len(kv) != 2 {
			return fmt.Errorf("%w: malformed env line", errSecretDetected)
		}
		key := strings.TrimSpace(kv[0])
		value := strings.TrimSpace(kv[1])
		allowed, ok := envAllowlist[key]
		if !ok {
			return fmt.Errorf("%w: unknown env key %q", errSecretDetected, key)
		}
		if seen[key] {
			return fmt.Errorf("%w: duplicate env key %q", errSecretDetected, key)
		}
		seen[key] = true
		if value != allowed {
			return fmt.Errorf("%w: non-synthetic value for %q", errSecretDetected, key)
		}
	}
	return nil
}
