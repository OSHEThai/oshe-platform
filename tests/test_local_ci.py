from __future__ import annotations

import json
import os
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

    def test_repository_configuration_includes_go_test_suite_check(self) -> None:
        root = self.make_repo([{"id": "pass", "command": ["python", "-c", "print('ok')"]}])
        (root / "tools").mkdir()
        (root / "tools" / "run_go_tests.py").write_text("print('go tests ok')", encoding="utf-8")
        completed = self.run_ci(root, "--mode", "incremental", "--no-checkpoint")
        self.assertEqual(completed.returncode, 0, completed.stdout + completed.stderr)
        self.assertIn("RUN  go-test-suite: python tools/run_go_tests.py", completed.stdout)
        self.assertIn("PASS go-test-suite", completed.stdout)

    def test_go_checkpoint_invalidated_when_go_toolchain_changes_or_disappears(self) -> None:
        root = self.make_repo([
            {"id": "python-pass", "command": ["python", "-c", "print('py-ok')"]},
            {"id": "go-pass", "command": ["python", "tools/run_go_tests.py"]},
        ])
        (root / "tools").mkdir()
        (root / "tools" / "run_go_tests.py").write_text("print('go-ok')", encoding="utf-8")

        # Hermetic mock bin directory for Go executable
        mock_bin = tempfile.TemporaryDirectory()
        self.addCleanup(mock_bin.cleanup)
        mock_bin_path = pathlib.Path(mock_bin.name)
        mock_go_script = mock_bin_path / ("go.bat" if sys.platform == "win32" else "go")
        mock_go_script.write_text("@echo go version go1.26.0 mock/arch\n", encoding="utf-8")
        mock_go_script.chmod(0o755)

        custom_env = os.environ.copy()
        custom_env["PATH"] = f"{mock_bin_path}{os.pathsep}{custom_env.get('PATH', '')}"

        # Helper to run CI with custom env
        def run_with_env(*args: str) -> subprocess.CompletedProcess[str]:
            return subprocess.run(
                [sys.executable, str(RUNNER), *args],
                cwd=root,
                check=False,
                capture_output=True,
                text=True,
                env=custom_env,
            )

        # 1. First run: both checks execute and checkpoint
        first = run_with_env("--mode", "incremental")
        self.assertEqual(first.returncode, 0, first.stdout + first.stderr)
        self.assertIn("PASS python-pass", first.stdout)
        self.assertIn("PASS go-pass", first.stdout)

        # 2. Second run: unchanged toolchain, both checks must SKIP
        second = run_with_env("--mode", "incremental")
        self.assertEqual(second.returncode, 0, second.stdout + second.stderr)
        self.assertIn("SKIP python-pass: unchanged passing checkpoint", second.stdout)
        self.assertIn("SKIP go-pass: unchanged passing checkpoint", second.stdout)

        # 3. Third run: Go toolchain changes version -> Go check MUST invalidate and re-run, Python check unaffected (skips)
        mock_go_script.write_text("@echo go version go1.26.1-upgraded mock/arch\n", encoding="utf-8")
        third = run_with_env("--mode", "incremental")
        self.assertEqual(third.returncode, 0, third.stdout + third.stderr)
        self.assertIn("SKIP python-pass: unchanged passing checkpoint", third.stdout)
        self.assertNotIn("SKIP go-pass", third.stdout)
        self.assertIn("RUN  go-pass: python tools/run_go_tests.py", third.stdout)

        # 4. Fourth run: Go toolchain disappears completely -> Go check MUST invalidate and re-run, Python check unaffected (skips)
        no_go_env = os.environ.copy()
        no_go_env["PATH"] = os.pathsep.join([
            p for p in os.environ.get("PATH", "").split(os.pathsep)
            if "go" not in p.lower() and p
        ])
        fourth = subprocess.run(
            [sys.executable, str(RUNNER), "--mode", "incremental"],
            cwd=root,
            check=False,
            capture_output=True,
            text=True,
            env=no_go_env,
        )
        self.assertIn("SKIP python-pass: unchanged passing checkpoint", fourth.stdout)
        self.assertNotIn("SKIP go-pass", fourth.stdout)
        self.assertIn("RUN  go-pass: python tools/run_go_tests.py", fourth.stdout)


    def test_unverifiable_or_timed_out_go_identity_strictly_bypasses_cache(self) -> None:
        root = self.make_repo([
            {"id": "python-pass", "command": ["python", "-c", "print('py-ok')"]},
            {"id": "go-pass", "command": ["python", "tools/run_go_tests.py"]},
        ])
        (root / "tools").mkdir()
        (root / "tools" / "run_go_tests.py").write_text("print('go-pass-ran')", encoding="utf-8")

        # Mock bin directory with an unverifiable/failing Go probe
        mock_bin = tempfile.TemporaryDirectory()
        self.addCleanup(mock_bin.cleanup)
        mock_bin_path = pathlib.Path(mock_bin.name)
        mock_go_script = mock_bin_path / ("go.bat" if sys.platform == "win32" else "go")
        mock_go_script.write_text("@echo probe-failure >&2 & exit /b 1\n", encoding="utf-8")
        mock_go_script.chmod(0o755)

        custom_env = os.environ.copy()
        custom_env["PATH"] = f"{mock_bin_path}{os.pathsep}{custom_env.get('PATH', '')}"

        def run_with_env(*args: str) -> subprocess.CompletedProcess[str]:
            return subprocess.run(
                [sys.executable, str(RUNNER), *args],
                cwd=root,
                check=False,
                capture_output=True,
                text=True,
                env=custom_env,
            )

        # 1. First run: both checks execute and finish
        first = run_with_env("--mode", "incremental")
        self.assertEqual(first.returncode, 0, first.stdout + first.stderr)
        self.assertIn("RUN  python-pass: python -c print('py-ok')", first.stdout)
        self.assertIn("RUN  go-pass: python tools/run_go_tests.py", first.stdout)
        self.assertIn("PASS python-pass", first.stdout)
        self.assertIn("PASS go-pass", first.stdout)

        # 2. Second consecutive run with EXACT SAME unverifiable Go probe:
        # python-pass MUST skip (unchanged toolchain), but go-pass MUST strictly bypass cache and re-run
        second = run_with_env("--mode", "incremental")
        self.assertEqual(second.returncode, 0, second.stdout + second.stderr)
        self.assertIn("SKIP python-pass: unchanged passing checkpoint", second.stdout)
        self.assertNotIn("SKIP go-pass", second.stdout)
        self.assertIn("RUN  go-pass: python tools/run_go_tests.py", second.stdout)

        # 3. Third consecutive run: still strictly bypasses cache for go-pass
        third = run_with_env("--mode", "incremental")
        self.assertEqual(third.returncode, 0, third.stdout + third.stderr)
        self.assertIn("SKIP python-pass: unchanged passing checkpoint", third.stdout)
        self.assertNotIn("SKIP go-pass", third.stdout)
        self.assertIn("RUN  go-pass: python tools/run_go_tests.py", third.stdout)
if __name__ == "__main__":
    unittest.main()
