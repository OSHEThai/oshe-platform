from __future__ import annotations

import dataclasses
import importlib.util
import json
import pathlib
import sys
import tempfile
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


class ValidateRepositoryRightsMetadataTests(unittest.TestCase):
    def _write(self, root: pathlib.Path, relative: str, text: str = "fixture\n") -> None:
        path = root / relative
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(text, encoding="utf-8")

    def _metadata(self) -> dict[str, object]:
        return {
            "schema_version": "1.0.0",
            "repository": "OSHEThai/oshe-platform",
            "licensor": "OSHEThai",
            "root_license": "MPL-2.0",
            "rules": [
                {"path": "LICENSE", "classification": "THIRD_PARTY_STANDARD_TEXT", "license": "MPL-2.0", "source": "fixture"},
                {"path": "LICENSES/**", "classification": "THIRD_PARTY_STANDARD_TEXT", "license": "SPDX_STANDARD_TEXT", "source": "fixture"},
                {"path": "DCO-1.1.txt", "classification": "THIRD_PARTY_STANDARD_TEXT", "license": "DCO-1.1", "source": "fixture"},
                {"path": "**", "classification": "OSHE_AUTHORED_ENGINEERING", "license": "MPL-2.0", "copyright": "OSHEThai"},
            ],
        }

    def test_complete_rights_metadata_passes(self) -> None:
        with tempfile.TemporaryDirectory() as temporary_directory:
            root = pathlib.Path(temporary_directory)
            for relative in ("LICENSE", "DCO-1.1.txt", "NOTICE.md", "LICENSES/MPL-2.0.txt", "LICENSES/Apache-2.0.txt", "source.py"):
                self._write(root, relative)
            (root / "RIGHTS-METADATA.json").write_text(json.dumps(self._metadata()), encoding="utf-8")

            self.assertEqual(validate_repository.validate_rights_metadata(root, "platform"), [])

    def test_unknown_rights_fail_closed(self) -> None:
        with tempfile.TemporaryDirectory() as temporary_directory:
            root = pathlib.Path(temporary_directory)
            for relative in ("LICENSE", "DCO-1.1.txt", "NOTICE.md", "LICENSES/MPL-2.0.txt", "LICENSES/Apache-2.0.txt"):
                self._write(root, relative)
            metadata = self._metadata()
            metadata["rules"] = []
            (root / "RIGHTS-METADATA.json").write_text(json.dumps(metadata), encoding="utf-8")

            errors = validate_repository.validate_rights_metadata(root, "platform")

            self.assertIn("rights metadata must contain ordered rules", errors)


class ValidateRepositorySecretPatternTests(unittest.TestCase):
    @staticmethod
    def _is_detected(value: str) -> bool:
        return any(pattern.search(value) for pattern in validate_repository.SECRET_PATTERNS)

    def test_runtime_constructed_representative_seeds_are_detected(self) -> None:
        seeds = {
            "classic-github-token": "gh" + "p_" + ("A" * 24),
            "fine-grained-github-token": "github" + "_pat_" + ("B" * 24),
            "private-key-marker": "-----BEGIN " + "PRIVATE KEY-----",
            "quoted-secret-assignment": "password" + "=" + '"' + ("C" * 12) + '"',
        }
        for name, seed in seeds.items():
            with self.subTest(name=name):
                self.assertTrue(self._is_detected(seed))

    def test_runtime_constructed_unquoted_sensitive_environment_assignment_is_detected(self) -> None:
        seeded_assignment = "API_" + "TOKEN" + "=" + "synthetic-seed-value"
        self.assertTrue(self._is_detected(seeded_assignment))

    def test_runtime_constructed_environment_reference_is_not_a_secret(self) -> None:
        safe_reference = "API_" + "TOKEN" + "=${API_" + "TOKEN}"
        self.assertFalse(self._is_detected(safe_reference))


    def test_synthetic_local_env_default_is_allowed_only_for_its_declared_path(self) -> None:
        seeded_line = "POSTGRES_" + "PASSWORD" + "=" + "oshe_dev_" + "synthetic_only"
        self.assertTrue(self._is_detected(seeded_line))
        stripped = validate_repository.secret_scan_text("deploy/local/.env.example", seeded_line)
        self.assertFalse(self._is_detected(stripped))

    def test_allowlist_is_path_scoped_and_value_exact(self) -> None:
        seeded_line = "POSTGRES_" + "PASSWORD" + "=" + "oshe_dev_" + "synthetic_only"
        same_value_other_path = validate_repository.secret_scan_text("deploy/other.env", seeded_line)
        self.assertTrue(self._is_detected(same_value_other_path))
        other_value_same_path = validate_repository.secret_scan_text(
            "deploy/local/.env.example",
            "POSTGRES_" + "PASSWORD" + "=" + "X" * 12,
        )
        self.assertTrue(self._is_detected(other_value_same_path))
        self.assertEqual(
            validate_repository.SECRET_ALLOWED_VALUES,
            {"deploy/local/.env.example": ("oshe_dev_synthetic_only",)},
        )


if __name__ == "__main__":
    unittest.main()
