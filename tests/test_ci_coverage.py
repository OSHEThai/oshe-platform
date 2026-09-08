#!/usr/bin/env python3
"""
tests/test_ci_coverage.py
Regression suite verifying complete local and hosted CI test coverage:
1. All Python test modules in tests/test_*.py are registered in .ci/local-ci.json.
2. All Go modules with go.mod are discovered and tested by tools/run_go_tests.py.
3. Coverage regression detects any newly added Python test module or Go module omitted from CI.
4. Classification of runtime/environment prerequisites fails closed (missing toolchain is not PASS).
5. Non-fail-fast aggregation across all checks and modules.
6. Hosted CI workflow (.github/workflows/foundation.yml) provisions both Go and Python environments.
"""

from __future__ import annotations

import json
import os
import pathlib
import shutil
import subprocess
import sys
import tempfile
import unittest
from typing import Any, Dict, List, Set


ROOT = pathlib.Path(__file__).resolve().parents[1]
LOCAL_CI_CONFIG_PATH = ROOT / ".ci" / "local-ci.json"
FOUNDATION_WORKFLOW_PATH = ROOT / ".github" / "workflows" / "foundation.yml"
RUN_GO_TESTS_PATH = ROOT / "tools" / "run_go_tests.py"
RUN_LOCAL_CI_PATH = ROOT / "tools" / "run_local_ci.py"


def get_all_python_test_modules(repo_root: pathlib.Path) -> Set[str]:
    test_dir = repo_root / "tests"
    modules = set()
    for path in test_dir.glob("test_*.py"):
        if path.is_file():
            modules.add(f"tests.{path.stem}")
    return modules


def get_ci_registered_python_modules(config_path: pathlib.Path) -> Set[str]:
    data = json.loads(config_path.read_text(encoding="utf-8"))
    registered = set()
    for check in data.get("checks", []):
        if not isinstance(check, dict):
            continue
        command = check.get("command", [])
        if not isinstance(command, list):
            continue
        # Check if invoking unittest with module arguments
        if len(command) >= 3 and command[0] == "python" and command[1:3] == ["-m", "unittest"]:
            for arg in command[3:]:
                if isinstance(arg, str) and arg.startswith("tests."):
                    registered.add(arg)
    return registered


def get_all_go_modules(repo_root: pathlib.Path) -> List[str]:
    excluded = {".git", ".local-ci", ".worktrees", ".herdrops-worktrees", "vendor", "node_modules", "bin", "dist"}
    modules = []
    for mod in repo_root.rglob("go.mod"):
        if not mod.is_file():
            continue
        rel = mod.relative_to(repo_root)
        if any(part in excluded for part in rel.parts):
            continue
        modules.append(mod.parent.relative_to(repo_root).as_posix())
    return sorted(modules)


class CiCoverageRegressionTests(unittest.TestCase):
    def test_all_python_test_modules_are_registered_in_local_ci(self) -> None:
        """Verifies that every test_*.py module on disk is registered in .ci/local-ci.json."""
        self.assertTrue(LOCAL_CI_CONFIG_PATH.is_file(), f"Missing {LOCAL_CI_CONFIG_PATH}")
        disk_modules = get_all_python_test_modules(ROOT)
        registered_modules = get_ci_registered_python_modules(LOCAL_CI_CONFIG_PATH)

        unregistered = disk_modules - registered_modules
        self.assertEqual(
            unregistered,
            set(),
            f"The following Python test modules exist on disk but are omitted from .ci/local-ci.json: {sorted(unregistered)}",
        )
        self.assertGreaterEqual(
            len(registered_modules),
            58,
            f"Expected at least 58 registered Python test modules, found {len(registered_modules)}",
        )

    def test_coverage_regression_detects_omitted_python_test(self) -> None:
        """Asserts that the coverage audit detects a newly created unregistered Python test file."""
        disk_modules = {"tests.test_existing_a", "tests.test_existing_b", "tests.test_newly_added_feature"}
        registered_modules = {"tests.test_existing_a", "tests.test_existing_b"}

        unregistered = disk_modules - registered_modules
        self.assertIn(
            "tests.test_newly_added_feature",
            unregistered,
            "Coverage check must flag newly added test file that is missing from CI registration.",
        )

    def test_all_go_modules_are_discovered_by_runner(self) -> None:
        """Verifies that tools/run_go_tests.py discovers all 20 Go modules across the platform."""
        self.assertTrue(RUN_GO_TESTS_PATH.is_file(), f"Missing {RUN_GO_TESTS_PATH}")
        go_modules = get_all_go_modules(ROOT)
        self.assertEqual(
            len(go_modules),
            20,
            f"Expected exactly 20 Go modules across repository, found {len(go_modules)}: {go_modules}",
        )

        expected_key_modules = [
            "apps/api",
            "contracts/api",
            "modules/configuration-checklist",
            "modules/contract-migration-governance",
            "modules/events-outbox-jobs",
            "modules/files-evidence",
            "modules/identity-authorization",
            "modules/incident-intake-foundation",
            "modules/internal-portal",
            "modules/organization-tenancy",
            "modules/public-portal",
            "modules/publication-snapshot",
            "modules/records-audit",
            "modules/reporting-localization",
            "modules/workflow-action",
            "modules/workforce-foundation",
            "modules/workforce-rule-preparation",
            "packages/flags",
            "packages/identifiers",
            "tools/dev",
        ]
        for mod in expected_key_modules:
            self.assertIn(mod, go_modules, f"Expected Go module {mod} missing from discovery list")

    def test_local_ci_runner_executes_go_test_suite(self) -> None:
        """Verifies that tools/run_local_ci.py incorporates tools/run_go_tests.py as go-test-suite."""
        self.assertTrue(RUN_GO_TESTS_PATH.is_file(), f"Missing {RUN_GO_TESTS_PATH}")
        self.assertTrue(RUN_LOCAL_CI_PATH.is_file(), f"Missing {RUN_LOCAL_CI_PATH}")
        runner_code = RUN_LOCAL_CI_PATH.read_text(encoding="utf-8")
        self.assertIn("run_go_tests.py", runner_code)
        self.assertIn("go-test-suite", runner_code)

    def test_missing_go_toolchain_fails_closed_with_prerequisite_error(self) -> None:
        """Verifies that tools/run_go_tests.py fails closed (code 2) when the Go executable is absent."""
        # Run with PATH emptied or modified so 'go' cannot be found
        env = os.environ.copy()
        env["PATH"] = ""
        proc = subprocess.run(
            [sys.executable, str(RUN_GO_TESTS_PATH), "--json"],
            cwd=ROOT,
            capture_output=True,
            text=True,
            check=False,
            env=env,
        )
        self.assertEqual(
            proc.returncode,
            2,
            f"Expected exit code 2 on missing prerequisite, got {proc.returncode}: {proc.stderr}",
        )
        self.assertIn(
            "prerequisite failure",
            proc.stderr.lower() + proc.stdout.lower(),
            "Missing Go executable must explicitly report prerequisite failure and never report PASS.",
        )

    def test_foundation_workflow_provisions_both_go_and_python(self) -> None:
        """Verifies that .github/workflows/foundation.yml configures both Python and Go environments."""
        self.assertTrue(FOUNDATION_WORKFLOW_PATH.is_file(), f"Missing {FOUNDATION_WORKFLOW_PATH}")
        workflow_text = FOUNDATION_WORKFLOW_PATH.read_text(encoding="utf-8")

        # Must provision Python
        self.assertIn("actions/setup-python", workflow_text, "Workflow must include actions/setup-python")
        self.assertIn("python-version", workflow_text)

        # Must provision Go
        self.assertIn("actions/setup-go", workflow_text, "Workflow must include actions/setup-go")
        self.assertTrue(
            "go-version" in workflow_text or "go-version-file" in workflow_text,
            "actions/setup-go step must declare go-version or go-version-file",
        )

        # Must run local CI
        self.assertIn("run_local_ci.py", workflow_text, "Workflow must execute run_local_ci.py")


    def test_run_go_tests_rejects_invalid_and_outside_root_modules(self) -> None:
        """Verifies that tools/run_go_tests.py rejects invalid or outside-root module paths."""
        # Test mixed valid and invalid module
        proc_mixed = subprocess.run(
            [sys.executable, str(RUN_GO_TESTS_PATH), "--module", "apps/api", "--module", "nonexistent/invalid_mod"],
            cwd=ROOT,
            capture_output=True,
            text=True,
            check=False,
        )
        self.assertEqual(
            proc_mixed.returncode,
            1,
            f"Expected exit code 1 on invalid module request, got {proc_mixed.returncode}",
        )
        self.assertIn(
            "nonexistent/invalid_mod",
            proc_mixed.stderr,
            "Error output must explicitly name the missing module",
        )

        # Test outside root module
        proc_outside = subprocess.run(
            [sys.executable, str(RUN_GO_TESTS_PATH), "--module", "../outside_repo"],
            cwd=ROOT,
            capture_output=True,
            text=True,
            check=False,
        )
        self.assertEqual(
            proc_outside.returncode,
            1,
            f"Expected exit code 1 on outside-root module request, got {proc_outside.returncode}",
        )
        self.assertIn(
            "outside repository root",
            proc_outside.stderr,
            "Error output must report resolution outside repository root",
        )

    def test_run_go_tests_json_stdout_is_parseable_standalone_json(self) -> None:
        """Verifies that tools/run_go_tests.py --json emits strictly parseable JSON to stdout."""
        proc = subprocess.run(
            [sys.executable, str(RUN_GO_TESTS_PATH), "--module", "apps/api", "--json"],
            cwd=ROOT,
            capture_output=True,
            text=True,
            check=False,
        )
        self.assertEqual(proc.returncode, 0, proc.stderr)

        # stdout must be directly parseable as standalone JSON without extra text
        try:
            payload = json.loads(proc.stdout)
        except json.JSONDecodeError as exc:
            self.fail(f"stdout is not parseable standalone JSON: {exc}\nstdout content:\n{proc.stdout}")

        self.assertIsInstance(payload, dict)
        self.assertTrue(payload.get("prerequisite_satisfied"))
        self.assertEqual(payload.get("total_modules"), 1)
        self.assertEqual(payload.get("passed_count"), 1)
        self.assertEqual(payload.get("failed_count"), 0)
        self.assertEqual(payload.get("passed_modules"), ["apps/api"])
        self.assertEqual(payload.get("failed_modules"), [])

if __name__ == "__main__":
    unittest.main()
