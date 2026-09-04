import json
import subprocess
import sys
import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
CLI = ROOT / "json_validator.py"
DAG = ROOT / "mission-dag.json"


class LocalRehearsalTests(unittest.TestCase):
    def test_dag_rehearses_failure_and_recovery_deterministically(self):
        dag = json.loads(DAG.read_text(encoding="utf-8"))
        steps = dag["steps"]
        self.assertEqual(
            [step["step_id"] for step in steps],
            ["preflight-valid-record", "inject-missing-status", "recovery-valid-record"],
        )

        for step in steps:
            result = subprocess.run(
                [
                    sys.executable,
                    str(CLI),
                    "--input",
                    str(ROOT / step["input"]),
                    "--schema",
                    str(ROOT / step["schema"]),
                ],
                check=False,
                capture_output=True,
                text=True,
            )
            self.assertEqual(result.stderr, "")
            self.assertEqual(result.returncode, step["expected_exit_code"])
            self.assertEqual(json.loads(result.stdout), step["expected_output"])


if __name__ == "__main__":
    unittest.main()
