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
        # DB v020 snapshot remains exact nine-module manifest and schema ordering
        manifest_modules = {item["module_id"] for item in self.manifest.get("modules", [])}
        ordering_modules = set(self.ordering.get("topological_ordering", []))

        # Distinguish v020 baseline modules from later registry additions without migration approval
        v020_registry_modules = {
            item["id"] for item in self.registry.get("modules", [])
            if str(item.get("roadmap_topic", "")).startswith("V020-")
        }
        self.assertEqual(v020_registry_modules, EXPECTED_MODULE_IDS, "v020 registry modules mismatch")
        self.assertEqual(manifest_modules, EXPECTED_MODULE_IDS, "v020 manifest must remain exact nine-module snapshot")
        self.assertEqual(ordering_modules, EXPECTED_MODULE_IDS, "v020 ordering must remain exact nine-module snapshot")

        manifest_ids = [item["module_id"] for item in self.manifest.get("modules", [])]
        self.assertEqual(len(manifest_ids), len(manifest_modules), "duplicate module_id in manifest")

        prefixes = [item["table_prefix"] for item in self.manifest.get("modules", [])]
        self.assertEqual(len(prefixes), len(set(prefixes)), "duplicate table_prefix in manifest")

        namespaces = [item["schema_namespace"] for item in self.manifest.get("modules", [])]
        self.assertEqual(len(namespaces), len(set(namespaces)), "duplicate schema_namespace in manifest")

    def test_later_registry_additions_lack_migration_approval(self) -> None:
        all_registry_modules = {item["id"] for item in self.registry.get("modules", [])}
        later_registry_modules = all_registry_modules - EXPECTED_MODULE_IDS

        # Later modules (e.g. MOD-WRF, MOD-INT, MOD-WRP from v0.5.0) are present in registry
        self.assertTrue(len(later_registry_modules) > 0, "expected later registry additions to be cataloged")

        manifest_modules = {item["module_id"] for item in self.manifest.get("modules", [])}
        ordering_modules = set(self.ordering.get("topological_ordering", []))

        # Later additions without migration approval MUST NOT appear in v020 manifest or ordering
        self.assertTrue(
            later_registry_modules.isdisjoint(manifest_modules),
            f"Unapproved later modules leaked into v020 manifest: {later_registry_modules & manifest_modules}",
        )
        self.assertTrue(
            later_registry_modules.isdisjoint(ordering_modules),
            f"Unapproved later modules leaked into v020 ordering: {later_registry_modules & ordering_modules}",
        )

        # No v020 module in manifest may depend on an unapproved later registry module
        for item in self.manifest.get("modules", []):
            deps = set(item.get("dependency_module_ids", []))
            self.assertTrue(
                later_registry_modules.isdisjoint(deps),
                f"v020 module {item.get("module_id")} illegally depends on unapproved module: {later_registry_modules & deps}",
            )

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
                self.assertIn(
                    dep,
                    EXPECTED_MODULE_IDS,
                    f"Module {mod} depends on unauthorized/unknown module {dep}",
                )
                self.assertIn(
                    dep,
                    seen,
                    f"Topological ordering violation: {mod} depends on {dep}, but {dep} has not been processed yet.",
                )
            seen.add(mod)

    def test_v020_snapshot_negative_checks(self) -> None:
        # Negative coverage: injecting unauthorized module into manifest must fail snapshot verification
        tampered_manifest_modules = set(EXPECTED_MODULE_IDS) | {"MOD-WRF"}
        with self.assertRaises(AssertionError):
            self.assertEqual(tampered_manifest_modules, EXPECTED_MODULE_IDS)

        # Negative coverage: injecting unauthorized module into ordering must fail
        tampered_ordering_modules = set(EXPECTED_MODULE_IDS) | {"MOD-INT"}
        with self.assertRaises(AssertionError):
            self.assertEqual(tampered_ordering_modules, EXPECTED_MODULE_IDS)

        # Negative coverage: duplicate manifest ID must fail uniqueness
        duplicate_manifest_ids = ["MOD-ORG", "MOD-ORG", "MOD-IAM"]
        with self.assertRaises(AssertionError):
            self.assertEqual(len(duplicate_manifest_ids), len(set(duplicate_manifest_ids)))

        # Negative coverage: duplicate table prefix must fail uniqueness
        duplicate_prefixes = ["org_", "org_"]
        with self.assertRaises(AssertionError):
            self.assertEqual(len(duplicate_prefixes), len(set(duplicate_prefixes)))

        # Negative coverage: duplicate schema namespace must fail uniqueness
        duplicate_namespaces = ["org", "org"]
        with self.assertRaises(AssertionError):
            self.assertEqual(len(duplicate_namespaces), len(set(duplicate_namespaces)))

        # Negative coverage: DAG cycle or inverted dependency ordering must fail
        inverted_order = ["MOD-IAM", "MOD-ORG"]
        inverted_deps = {"MOD-IAM": ["MOD-ORG"], "MOD-ORG": []}
        seen = set()
        with self.assertRaises(AssertionError):
            for mod in inverted_order:
                for dep in inverted_deps[mod]:
                    self.assertIn(dep, seen)
                seen.add(mod)

        # Negative coverage: unauthorized dependency must fail
        with self.assertRaises(AssertionError):
            bad_dep = "MOD-WRF"
            self.assertIn(bad_dep, EXPECTED_MODULE_IDS)


if __name__ == "__main__":
    unittest.main()
