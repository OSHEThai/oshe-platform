from __future__ import annotations

import json
import pathlib
import subprocess
import sys
import tempfile
import unittest


ROOT = pathlib.Path(__file__).resolve().parents[1]
RUNNER = ROOT / "tools" / "run_local_ci.py"


class LocalCiTests(unittest.TestCase):
    def make_repo(self, checks: list[dict[str, object]]) -> pathlib.Path:
        temporary = tempfile.TemporaryDirectory()
        self.addCleanup(temporary.cleanup)
        root = pathlib.Path(temporary.name)
        (root / ".ci").mkdir()
        (root / ".ci" / "local-ci.json").write_text(
            json.dumps({"schema_version": "1.0.0", "checks": checks}),
            encoding="utf-8",
        )
        subprocess.run(["git", "init", "-q"], cwd=root, check=True)
        return root

    def run_ci(self, root: pathlib.Path, *args: str) -> subprocess.CompletedProcess[str]:
        return subprocess.run(
            [sys.executable, str(RUNNER), *args],
            cwd=root,
            check=False,
            capture_output=True,
            text=True,
        )

    def test_batch_collects_all_results_without_fail_fast(self) -> None:
        marker = "second-check-ran.txt"
        root = self.make_repo(
            [
                {"id": "first-fails", "command": ["python", "-c", "raise SystemExit(7)"]},
                {
                    "id": "second-runs",
                    "command": ["python", "-c", f"from pathlib import Path; Path('{marker}').write_text('yes')"],
                },
            ]
        )
        completed = self.run_ci(root, "--mode", "incremental", "--no-checkpoint")
        self.assertEqual(completed.returncode, 1, completed.stdout + completed.stderr)
        self.assertTrue((root / marker).is_file())
        self.assertIn("failed=1", completed.stdout)
        self.assertIn("passed=1", completed.stdout)

    def test_unchanged_pass_is_checkpointed_and_skipped(self) -> None:
        root = self.make_repo([{"id": "pass", "command": ["python", "-c", "print('ok')"]}])
        first = self.run_ci(root, "--mode", "incremental")
        second = self.run_ci(root, "--mode", "incremental")
        self.assertEqual(first.returncode, 0, first.stdout + first.stderr)
        self.assertEqual(second.returncode, 0, second.stdout + second.stderr)
        self.assertIn("SKIP pass", second.stdout)

    def test_full_ci_requires_milestone_close(self) -> None:
        root = self.make_repo([{"id": "pass", "command": ["python", "-c", "print('ok')"]}])
        completed = self.run_ci(root, "--mode", "full")
        self.assertNotEqual(completed.returncode, 0)
        self.assertIn("--milestone-close", completed.stderr)

    def test_repository_configuration_includes_supply_chain_verification_check(self) -> None:
        config_path = ROOT / ".ci" / "local-ci.json"
        self.assertTrue(config_path.is_file(), "missing .ci/local-ci.json")
        config = json.loads(config_path.read_text(encoding="utf-8"))
        checks = config.get("checks", [])
        supply_chain_checks = [
            check
            for check in checks
            if isinstance(check, dict)
            and check.get("command") == ["python", "tools/supply_chain.py", "--verify"]
        ]
        self.assertEqual(
            len(supply_chain_checks),
            1,
            "expected exactly one local CI check invoking 'python tools/supply_chain.py --verify'",
        )
        self.assertEqual(supply_chain_checks[0].get("id"), "supply-chain-verification")


if __name__ == "__main__":
    unittest.main()
