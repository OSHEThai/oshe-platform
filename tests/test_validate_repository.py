from __future__ import annotations

import dataclasses
import importlib.util
import pathlib
import sys
import unittest


MODULE_PATH = pathlib.Path(__file__).resolve().parents[1] / "tools" / "validate_repository.py"
SPEC = importlib.util.spec_from_file_location("validate_repository", MODULE_PATH)
assert SPEC is not None and SPEC.loader is not None
validate_repository = importlib.util.module_from_spec(SPEC)
sys.modules["validate_repository"] = validate_repository
SPEC.loader.exec_module(validate_repository)


class ValidateRepositoryYamlLimitTests(unittest.TestCase):
    def setUp(self) -> None:
        self.limits = validate_repository.YamlResourceLimits()
        self.metric_limits = {
            "file_bytes": self.limits.max_file_bytes,
            "composed_nodes": self.limits.max_composed_nodes,
            "depth": self.limits.max_depth,
            "aggregate_scalar_characters": self.limits.max_aggregate_scalar_characters,
            "anchors": self.limits.max_anchors,
            "aliases": self.limits.max_aliases,
            "expanded_visits": self.limits.max_expanded_visits,
            "process_count": self.limits.max_process_count,
            "wall_clock_seconds": self.limits.wall_clock_seconds,
            "memory_mib": self.limits.memory_mib,
        }

    def _metrics(self, **changes: int) -> object:
        return dataclasses.replace(validate_repository.YamlMetrics(), **changes)

    def test_approved_limit_values_pass(self) -> None:
        for metric, limit in self.metric_limits.items():
            with self.subTest(metric=metric):
                validate_repository.enforce_yaml_limits(self._metrics(**{metric: limit}))

    def test_boundary_plus_one_values_fail_closed(self) -> None:
        for metric, limit in self.metric_limits.items():
            with self.subTest(metric=metric):
                with self.assertRaises(validate_repository.YamlLimitError):
                    validate_repository.enforce_yaml_limits(self._metrics(**{metric: limit + 1}))

    def test_zero_values_pass(self) -> None:
        for metric in self.metric_limits:
            with self.subTest(metric=metric):
                validate_repository.enforce_yaml_limits(self._metrics(**{metric: 0}))

    def test_negative_values_fail_closed(self) -> None:
        for metric in self.metric_limits:
            with self.subTest(metric=metric):
                with self.assertRaises(validate_repository.YamlLimitError):
                    validate_repository.enforce_yaml_limits(self._metrics(**{metric: -1}))

    def test_overflow_values_fail_closed(self) -> None:
        overflow = 2**63
        for metric in self.metric_limits:
            with self.subTest(metric=metric):
                with self.assertRaises(validate_repository.YamlLimitError):
                    validate_repository.enforce_yaml_limits(self._metrics(**{metric: overflow}))

    def test_anchor_and_alias_limits_reject_before_yaml_construction(self) -> None:
        with self.assertRaises(validate_repository.YamlLimitError):
            validate_repository.analyze_yaml_preconstruction("key: &named value\n")
        with self.assertRaises(validate_repository.YamlLimitError):
            validate_repository.analyze_yaml_preconstruction("key: *named\n")

    def test_github_actions_boolean_and_is_not_a_yaml_anchor(self) -> None:
        metrics = validate_repository.analyze_yaml_preconstruction(
            "REQUESTED_CI_MODE: ${{ github.event_name == 'workflow_dispatch' && inputs.ci_mode || 'incremental' }}\n"
        )
        self.assertEqual(metrics.anchors, 0)
        self.assertEqual(metrics.aliases, 0)

    def test_outer_corpus_is_exact_and_excludes_delegated_ai_yaml(self) -> None:
        self.assertEqual(len(validate_repository.OUTER_YAML_PATHS), 8)
        self.assertFalse(any(path.startswith(".ai/") for path in validate_repository.OUTER_YAML_PATHS))


if __name__ == "__main__":
    unittest.main()
