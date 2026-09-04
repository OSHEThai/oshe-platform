from __future__ import annotations

import copy
import json
import pathlib
import subprocess
import sys
import unittest
import yaml

from database.validate_migration_manifest import validate_manifest_and_ordering

ROOT = pathlib.Path(__file__).resolve().parents[1]
MANIFEST_PATH = ROOT / "database" / "migration-manifest.yaml"
ORDERING_PATH = ROOT / "database" / "schema-ordering.json"
VALIDATOR_SCRIPT_PATH = ROOT / "database" / "validate_migration_manifest.py"


class DatabaseMigrationManifestValidationTests(unittest.TestCase):
    def setUp(self) -> None:
        self.assertTrue(MANIFEST_PATH.is_file(), f"missing {MANIFEST_PATH}")
        self.assertTrue(ORDERING_PATH.is_file(), f"missing {ORDERING_PATH}")

        self.manifest = yaml.safe_load(MANIFEST_PATH.read_text(encoding="utf-8"))
        self.ordering = json.loads(ORDERING_PATH.read_text(encoding="utf-8"))

    def test_canonical_manifest_and_ordering_pass(self) -> None:
        """Verifies that repository's canonical manifest and schema ordering pass cleanly with zero errors."""
        errors = validate_manifest_and_ordering(self.manifest, self.ordering)
        self.assertEqual(errors, [], f"canonical validation had unexpected errors: {errors}")

    def test_fail_closed_on_duplicate_schema_namespace(self) -> None:
        """Verifies rejection if two modules share the same schema namespace."""
        bad_manifest = copy.deepcopy(self.manifest)
        bad_manifest["modules"][1]["schema_namespace"] = "org"  # duplicate of MOD-ORG

        errors = validate_manifest_and_ordering(bad_manifest, self.ordering)
        self.assertTrue(any("duplicate schema_namespace" in e for e in errors), f"expected duplicate namespace error, got: {errors}")

    def test_fail_closed_on_duplicate_table_prefix(self) -> None:
        """Verifies rejection if two modules share the same table prefix."""
        bad_manifest = copy.deepcopy(self.manifest)
        bad_manifest["modules"][1]["table_prefix"] = "org_"  # duplicate of MOD-ORG

        errors = validate_manifest_and_ordering(bad_manifest, self.ordering)
        self.assertTrue(any("duplicate table_prefix" in e for e in errors), f"expected duplicate prefix error, got: {errors}")

    def test_fail_closed_on_order_index_inversion(self) -> None:
        """Verifies rejection if order_index does not match sequence index."""
        bad_manifest = copy.deepcopy(self.manifest)
        bad_manifest["modules"][0]["order_index"] = 2
        bad_manifest["modules"][1]["order_index"] = 1

        errors = validate_manifest_and_ordering(bad_manifest, self.ordering)
        self.assertTrue(any("order_index mismatch" in e for e in errors), f"expected order_index mismatch error, got: {errors}")

    def test_fail_closed_on_cyclic_dependency(self) -> None:
        """Verifies detection and fail-closed rejection of cyclic dependencies in the DAG."""
        bad_manifest = copy.deepcopy(self.manifest)
        # Introduce cycle: MOD-ORG depends on MOD-CTR (while MOD-CTR depends on MOD-ORG)
        bad_manifest["modules"][0]["dependency_module_ids"] = ["MOD-CTR"]

        errors = validate_manifest_and_ordering(bad_manifest, self.ordering)
        self.assertTrue(any("cyclic dependency" in e or "dependency ordering violation" in e for e in errors), f"expected cycle error, got: {errors}")

    def test_fail_closed_on_dependency_ordering_violation(self) -> None:
        """Verifies rejection if a module depends on another module that appears later in the DAG."""
        bad_manifest = copy.deepcopy(self.manifest)
        # MOD-IAM (index 2) depends on MOD-WFA (index 7)
        bad_manifest["modules"][1]["dependency_module_ids"].append("MOD-WFA")

        errors = validate_manifest_and_ordering(bad_manifest, self.ordering)
        self.assertTrue(any("dependency ordering violation" in e for e in errors), f"expected ordering violation error, got: {errors}")

    def test_fail_closed_on_missing_required_phase(self) -> None:
        """Verifies rejection if a module omits required phase M0, M1, or M2."""
        bad_manifest = copy.deepcopy(self.manifest)
        bad_manifest["modules"][4]["phases_supported"] = ["M0", "M1"]  # omitted M2

        errors = validate_manifest_and_ordering(bad_manifest, self.ordering)
        self.assertTrue(any("missing mandatory phases" in e for e in errors), f"expected missing phase error, got: {errors}")

    def test_fail_closed_on_destructive_phase_m3(self) -> None:
        """Verifies rejection if a module declares destructive phase M3 without owner decision."""
        bad_manifest = copy.deepcopy(self.manifest)
        bad_manifest["modules"][3]["phases_supported"].append("M3")

        errors = validate_manifest_and_ordering(bad_manifest, self.ordering)
        self.assertTrue(any("destructive phase 'M3'" in e for e in errors), f"expected M3 prohibition error, got: {errors}")

    def test_fail_closed_on_forbidden_invariant_override(self) -> None:
        """Verifies rejection if architectural rules attempt to permit forbidden patterns."""
        bad_manifest = copy.deepcopy(self.manifest)
        bad_manifest["rules"]["shared_tables"] = "ALLOWED"

        errors = validate_manifest_and_ordering(bad_manifest, self.ordering)
        self.assertTrue(any("shared_tables must be 'PROHIBITED'" in e for e in errors), f"expected rule error, got: {errors}")

    def test_fail_closed_on_runtime_execution_tamper(self) -> None:
        """Verifies rejection if provisional/runtime invariants are tampered with."""
        bad_ordering = copy.deepcopy(self.ordering)
        bad_ordering["runtime_execution"] = True

        bad_manifest = copy.deepcopy(self.manifest)
        bad_manifest["execution_state"] = "EXECUTED_LIVE"

        errors = validate_manifest_and_ordering(bad_manifest, bad_ordering)
        self.assertTrue(any("runtime_execution must be False" in e for e in errors), f"expected runtime_execution error, got: {errors}")
        self.assertTrue(any("execution_state must be 'NOT_RUNTIME_EXECUTED'" in e for e in errors), f"expected execution_state error, got: {errors}")

    def test_fail_closed_on_unknown_dependency_module(self) -> None:
        """Verifies rejection if a module declares an unknown dependency."""
        bad_manifest = copy.deepcopy(self.manifest)
        bad_manifest["modules"][0]["dependency_module_ids"] = ["MOD-EXTERNAL-NONEXISTENT"]

        errors = validate_manifest_and_ordering(bad_manifest, self.ordering)
        self.assertTrue(any("unknown module 'MOD-EXTERNAL-NONEXISTENT'" in e for e in errors), f"expected unknown dependency error, got: {errors}")

    def test_cli_execution_success(self) -> None:
        """Verifies that running the validator script directly via CLI exits with code 0."""
        result = subprocess.run(
            [sys.executable, str(VALIDATOR_SCRIPT_PATH)],
            capture_output=True,
            text=True,
            check=False,
        )
        self.assertEqual(result.returncode, 0, f"CLI validator failed: {result.stderr}")
        self.assertIn("validation PASSED", result.stdout)


if __name__ == "__main__":
    unittest.main()
