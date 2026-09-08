---
document_id: DOC-ARCH-CI-COVERAGE-001
title: CI Test Coverage Contract and Multi-Module Runner Architecture
document_type: architecture_specification
document_version: 1.0.0
lifecycle_status: APPROVED_FOR_LOCAL_DEVELOPMENT
status: ACTIVE
governing_mission: MISSION-AUDIT-REMEDIATION-20260908
governing_lease: LEASE-AUDIT-20260908-C
governing_assignment: ASN-AUDIT-20260908-C
governing_decisions:
  - HDEC-GITHUB-AUTHORITY-AND-REVIEW-EFFICIENCY-030
  - HDEC-SPEND-ALLOWED-HERDR-DISPATCH-045
  - HDEC-AGENT-TOPOLOGY-LEAN-032
---

# CI Test Coverage Contract and Multi-Module Runner Architecture

## 1. Executive Summary & Purpose

This contract establishes the governed architecture for automated test coverage and execution across `oshe-platform`. Prior to this specification, automated local and hosted continuous integration was restricted to 9 Python regression tests and zero Go module tests, omitting 49 Python test modules and 87 Go test files across 20 Go modules.

This contract mandates:
1. Complete discovery and registration of all 59 Python test modules in `tests/test_*.py`.
2. Multi-module discovery, isolated execution, and aggregation of all 20 Go modules via `tools/run_go_tests.py`.
3. Fail-closed prerequisite classification: missing runtime environments or compilers must fail closed and never report false passes.
4. Non-fail-fast execution: test suites run comprehensively across all units, collecting full diagnostic outcomes without premature aborts.
5. Automated coverage regression enforcement via `tests/test_ci_coverage.py`.
6. Toolchain parity between local execution and GitHub Actions hosted workflows (`.github/workflows/foundation.yml`).

---

## 2. Multi-Module Go Test Runner Architecture (`tools/run_go_tests.py`)

### 2.1 Discovery Contract
`tools/run_go_tests.py` dynamically discovers all `go.mod` files within the repository tree, excluding transient and non-production paths (`.git`, `.worktrees`, `.local-ci`, `vendor`, `node_modules`, `bin`, `dist`).

The canonical 20 Go modules covered by this runner include:
1. `apps/api`
2. `contracts/api`
3. `modules/configuration-checklist`
4. `modules/contract-migration-governance`
5. `modules/events-outbox-jobs`
6. `modules/files-evidence`
7. `modules/identity-authorization`
8. `modules/incident-intake-foundation`
9. `modules/internal-portal`
10. `modules/organization-tenancy`
11. `modules/public-portal`
12. `modules/publication-snapshot`
13. `modules/records-audit`
14. `modules/reporting-localization`
15. `modules/workflow-action`
16. `modules/workforce-foundation`
17. `modules/workforce-rule-preparation`
18. `packages/flags`
19. `packages/identifiers`
20. `tools/dev`

### 2.2 Execution & Aggregation Invariants
- **Non-Fail-Fast:** The runner executes `go test ./...` in every discovered module directory sequentially, ensuring that a failure in one module does not halt execution of remaining modules.
- **Deterministic Timeout:** Each module execution is bounded by a configurable timeout (default: 120 seconds).
- **Strict `--module` Filtering:** When specific modules are requested via `--module`, the runner resolves each path relative to repository root and validates that it contains a `go.mod` file and does not escape repository root. Any invalid, missing, or outside-root module is rejected with exit code `1`; silent omission or partial passes are prohibited.
- **Standalone `--json` Output Contract:** When `--json` is enabled, `tools/run_go_tests.py` emits strictly parseable JSON to standard output (`stdout`). All human-readable progress, toolchain banners, and test execution logs are routed exclusively to standard error (`stderr`), ensuring `json.loads(stdout)` succeeds cleanly in automated callers.
- **Exit Dispositions:**
  - Exit Code `0`: All discovered (or requested) modules executed and passed cleanly.
  - Exit Code `1`: One or more modules failed during test execution, or an invalid module was requested.
  - Exit Code `2`: Environment prerequisite failure (e.g., Go executable not found in `PATH`).
---

## 3. Python Regression Test Registration (`.ci/local-ci.json`)

All 59 Python test modules residing in `tests/test_*.py` are registered as explicit arguments to `python -m unittest` in `.ci/local-ci.json`.

### Invariants:
- **No Silent Skips:** Omission of any test module from the CI configuration is prohibited.
- **Coverage Regression Defense:** `tests/test_ci_coverage.py` automatically checks the disk contents of `tests/` against `.ci/local-ci.json`. Any test module created on disk without registration in CI causes `test_ci_coverage.py` to fail immediately.

---

## 4. Prerequisite Classification & Fail-Closed Rules

Runtime prerequisites are classified honestly:
1. **Missing Go Toolchain:** If `go` is missing from `PATH`, `tools/run_go_tests.py` exits with code `2` and writes an explicit error message to `sys.stderr`. Under no circumstances may a missing toolchain result in a skipped or passing status.
2. **Missing Python Modules:** Missing required Python dependencies (such as PyYAML or jsonschema) must raise an import error that causes the test runner or validator to exit with non-zero exit codes.
3. **Database / Network Dependencies:** Standalone unit and qualification tests must execute strictly against synthetic fixtures and in-memory stubs. Any test requiring external network access or unprovisioned databases must fail closed.

---

## 5. Hosted CI Parity (`.github/workflows/foundation.yml`)

The hosted GitHub Actions workflow `foundation.yml` must maintain exact environment parity with the local CI runner:
1. **Python Environment:** Configured via `actions/setup-python@v6` with Python 3.13.
2. **Go Environment:** Configured via `actions/setup-go@v5` reading version constraints directly from `apps/api/go.mod` (Go 1.26 baseline).
3. **Runner Execution:** Executes `tools/run_local_ci.py`, which triggers both the Python and Go test suites without fail-fast suppression.

---

## 6. Pre-Existing Defect Protocol

Pre-existing test failures identified during CI coverage expansion (e.g., in `tools/dev` or specific contract tests) must be:
1. Accurately and honestly recorded in the CI test output.
2. Preserved without test suppression, weakening of assertion thresholds, or blanket exclusions.
3. Reported to the designated lane owner (e.g., Lane A for workforce contracts, Lane B for registries) for dedicated defect resolution.
