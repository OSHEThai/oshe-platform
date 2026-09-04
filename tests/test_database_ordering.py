from __future__ import annotations

import json
import pathlib
import unittest
import yaml

ROOT = pathlib.Path(__file__).resolve().parents[1]
MODULE_REGISTRY_PATH = ROOT / "modules" / "module-registry.yaml"
MIGRATION_MANIFEST_PATH = ROOT / "database" / "migration-manifest.yaml"
SCHEMA_ORDERING_PATH = ROOT / "database" / "schema-ordering.json"

EXPECTED_MODULE_IDS = {
    "MOD-ORG",
    "MOD-IAM",
    "MOD-EVD",
    "MOD-REC",
    "MOD-CFG",
    "MOD-WFA",
    "MOD-EVT",
    "MOD-REP",
    "MOD-CTR",
}


class DatabaseOrderingStaticTests(unittest.TestCase):
    def setUp(self) -> None:
        self.assertTrue(MODULE_REGISTRY_PATH.is_file(), "missing module-registry.yaml")
        self.assertTrue(MIGRATION_MANIFEST_PATH.is_file(), "missing migration-manifest.yaml")
        self.assertTrue(SCHEMA_ORDERING_PATH.is_file(), "missing schema-ordering.json")

        self.registry = yaml.safe_load(MODULE_REGISTRY_PATH.read_text(encoding="utf-8"))
        self.manifest = yaml.safe_load(MIGRATION_MANIFEST_PATH.read_text(encoding="utf-8"))
        self.ordering = json.loads(SCHEMA_ORDERING_PATH.read_text(encoding="utf-8"))

    def test_canonical_module_coverage_and_uniqueness(self) -> None:
        registry_modules = {item["id"] for item in self.registry.get("modules", [])}
        manifest_modules = {item["module_id"] for item in self.manifest.get("modules", [])}
        ordering_modules = set(self.ordering.get("topological_ordering", []))

        self.assertEqual(registry_modules, EXPECTED_MODULE_IDS)
        self.assertEqual(manifest_modules, EXPECTED_MODULE_IDS)
        self.assertEqual(ordering_modules, EXPECTED_MODULE_IDS)

        manifest_ids = [item["module_id"] for item in self.manifest.get("modules", [])]
        self.assertEqual(len(manifest_ids), len(manifest_modules), "duplicate module_id in manifest")

        prefixes = [item["table_prefix"] for item in self.manifest.get("modules", [])]
        self.assertEqual(len(prefixes), len(set(prefixes)), "duplicate table_prefix in manifest")

        namespaces = [item["schema_namespace"] for item in self.manifest.get("modules", [])]
        self.assertEqual(len(namespaces), len(set(namespaces)), "duplicate schema_namespace in manifest")

    def test_prohibitions_and_invariants_declared(self) -> None:
        rules = self.manifest.get("rules", {})
        self.assertEqual(rules.get("cross_module_direct_writes"), "PROHIBITED")
        self.assertEqual(rules.get("shared_tables"), "PROHIBITED")
        self.assertEqual(rules.get("private_table_joins"), "PROHIBITED")

        invariants = self.ordering.get("invariants", {})
        self.assertEqual(invariants.get("cross_module_direct_writes"), "PROHIBITED")
        self.assertEqual(invariants.get("shared_tables"), "PROHIBITED")
        self.assertEqual(invariants.get("private_table_joins"), "PROHIBITED")
        self.assertEqual(invariants.get("destructive_migrations_m3"), "PROHIBITED_WITHOUT_OWNER_DECISION")

    def test_provisional_and_non_runtime_execution_status(self) -> None:
        self.assertEqual(self.manifest.get("lifecycle_status"), "PROVISIONAL")
        self.assertEqual(self.manifest.get("status"), "PROVISIONAL_PENDING_H020_005")
        self.assertEqual(self.manifest.get("human_gate"), "H020-005")
        self.assertEqual(self.manifest.get("execution_state"), "NOT_RUNTIME_EXECUTED")

        self.assertEqual(self.ordering.get("lifecycle_status"), "PROVISIONAL")
        self.assertEqual(self.ordering.get("status"), "PROVISIONAL_PENDING_H020_005")
        self.assertEqual(self.ordering.get("human_gate"), "H020-005")
        self.assertFalse(self.ordering.get("runtime_execution"))

    def test_dependency_dag_is_acyclic_and_order_is_topologically_valid(self) -> None:
        deps = {}
        for item in self.manifest.get("modules", []):
            deps[item["module_id"]] = list(item.get("dependency_module_ids", []))

        order = self.ordering.get("topological_ordering", [])
        self.assertEqual(len(order), len(EXPECTED_MODULE_IDS))

        seen = set()
        for mod in order:
            for dep in deps[mod]:
                if dep not in EXPECTED_MODULE_IDS:
                    continue
                self.assertIn(
                    dep,
                    seen,
                    f"Topological ordering violation: {mod} depends on {dep}, but {dep} has not been processed yet.",
                )
            seen.add(mod)


if __name__ == "__main__":
    unittest.main()
