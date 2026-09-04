from __future__ import annotations

import importlib.util
import json
import pathlib
import sys
import tempfile
import unittest


MODULE_PATH = pathlib.Path(__file__).resolve().parents[1] / "tools" / "reference_benchmark.py"
SPEC = importlib.util.spec_from_file_location("reference_benchmark", MODULE_PATH)
assert SPEC is not None and SPEC.loader is not None
benchmark = importlib.util.module_from_spec(SPEC)
sys.modules["reference_benchmark"] = benchmark
SPEC.loader.exec_module(benchmark)


FIXTURES = pathlib.Path(__file__).resolve().parent / "fixtures" / "reference_benchmark" / "synthetic_cases.json"


class ReferenceBenchmarkTests(unittest.TestCase):
    def setUp(self) -> None:
        self.fixtures = benchmark.load_fixtures(FIXTURES)

    def test_baseline_is_deterministic_and_provider_free(self) -> None:
        first = benchmark.scorecard(self.fixtures, "NONE")
        second = benchmark.scorecard(self.fixtures, "NONE")
        self.assertEqual(first, second)
        self.assertEqual(first["summary"], {"completed": 4, "failed": 0, "total": 4})
        self.assertEqual(
            [item["fixture_id"] for item in first["fixtures"]],
            [
                "synthetic-json-echo",
                "synthetic-schema-summary",
                "synthetic-json-validator-valid-record",
                "synthetic-json-validator-missing-required-field",
            ],
        )
        self.assertTrue(all(item["measures"]["route_selection"] == "NONE" for item in first["fixtures"]))

    def test_failure_injections_fail_closed(self) -> None:
        for policy, marker in (("TIMEOUT", "timeout_triggered"), ("MALFORMED_OUTPUT", "parse_error_detected"), ("STEP_LIMIT", "step_limit_exceeded")):
            with self.subTest(policy=policy):
                card = benchmark.scorecard(self.fixtures, policy)
                self.assertEqual(card["summary"]["failed"], len(self.fixtures))
                self.assertTrue(all(item["measures"].get(marker) == "true" for item in card["fixtures"]))

    def test_live_provider_argument_is_rejected(self) -> None:
        with self.assertRaisesRegex(ValueError, "not authorized under H010-007"):
            benchmark.run_fixture(self.fixtures[0], "NONE", provider="candidate-provider")

    def test_invalid_fixture_and_output_collision_fail_closed(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            root = pathlib.Path(temporary)
            invalid = root / "invalid.json"
            invalid.write_text(json.dumps({"schema_version": "1.0.0", "fixtures": [{"fixture_id": "x", "input": {}, "expected_output": {}, "max_turns": 0}]}), encoding="utf-8")
            with self.assertRaises(ValueError):
                benchmark.load_fixtures(invalid)
            output = root / "scorecard.json"
            benchmark.write_scorecard(output, benchmark.scorecard(self.fixtures, "NONE"))
            with self.assertRaises(FileExistsError):
                benchmark.write_scorecard(output, benchmark.scorecard(self.fixtures, "NONE"))


if __name__ == "__main__":
    unittest.main()
