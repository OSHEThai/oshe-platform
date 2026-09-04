from __future__ import annotations

import copy
import hashlib
import json
import pathlib
import unittest
import yaml

ROOT = pathlib.Path(__file__).resolve().parents[1]
MIGRATION_MANIFEST_PATH = ROOT / "database" / "migration-manifest.yaml"
SCHEMA_ORDERING_PATH = ROOT / "database" / "schema-ordering.json"

TOPOLOGICAL_ORDER = [
    "MOD-ORG",
    "MOD-IAM",
    "MOD-REC",
    "MOD-EVD",
    "MOD-CFG",
    "MOD-EVT",
    "MOD-WFA",
    "MOD-REP",
    "MOD-CTR",
]


class SyntheticDatabase:
    """In-memory synthetic state model simulating database schema and table states."""

    def __init__(self) -> None:
        self.applied_modules: dict[str, dict[str, any]] = {}
        self.tables: dict[str, list[dict[str, any]]] = {}
        self.current_phase: str = "M0"
        self.disposition: str = "CLEAN_BASELINE"

    def apply_module_migration(self, module_id: str, phase: str, version: str) -> None:
        self.applied_modules[module_id] = {
            "phase": phase,
            "version": version,
            "applied": True,
        }
        self.current_phase = phase

    def create_table(self, table_name: str, records: list[dict[str, any]] | None = None) -> None:
        self.tables[table_name] = records or []

    def snapshot(self) -> dict[str, any]:
        return {
            "applied_modules": copy.deepcopy(self.applied_modules),
            "tables": copy.deepcopy(self.tables),
            "current_phase": self.current_phase,
            "disposition": self.disposition,
        }

    def restore(self, snap: dict[str, any]) -> None:
        self.applied_modules = copy.deepcopy(snap["applied_modules"])
        self.tables = copy.deepcopy(snap["tables"])
        self.current_phase = snap["current_phase"]
        self.disposition = snap["disposition"]

    def digest(self) -> str:
        serialized = json.dumps(
            {"modules": self.applied_modules, "tables": self.tables},
            sort_keys=True,
        )
        return hashlib.sha256(serialized.encode("utf-8")).hexdigest()


class SyntheticRecoveryQualificationTests(unittest.TestCase):
    def setUp(self) -> None:
        self.assertTrue(MIGRATION_MANIFEST_PATH.is_file(), "missing migration-manifest.yaml")
        self.assertTrue(SCHEMA_ORDERING_PATH.is_file(), "missing schema-ordering.json")

        self.manifest = yaml.safe_load(MIGRATION_MANIFEST_PATH.read_text(encoding="utf-8"))
        self.ordering = json.loads(SCHEMA_ORDERING_PATH.read_text(encoding="utf-8"))
        self.db = SyntheticDatabase()

    def test_governing_gate_and_prohibitions(self) -> None:
        """Verifies H020-005 deferral and non-execution invariants."""
        self.assertEqual(self.manifest.get("human_gate"), "H020-005")
        self.assertEqual(self.manifest.get("execution_state"), "NOT_RUNTIME_EXECUTED")
        self.assertEqual(self.manifest.get("lifecycle_status"), "PROVISIONAL")
        self.assertFalse(self.ordering.get("runtime_execution"))

        rules = self.manifest.get("rules", {})
        self.assertEqual(rules.get("cross_module_direct_writes"), "PROHIBITED")
        self.assertEqual(rules.get("shared_tables"), "PROHIBITED")
        self.assertEqual(rules.get("private_table_joins"), "PROHIBITED")

    def test_synthetic_upgrade_flow(self) -> None:
        """Scenario 1: Upgrade (M0 to M1 forward migration)."""
        # Execute forward migration in strict topological order
        for mod in TOPOLOGICAL_ORDER:
            self.db.apply_module_migration(mod, phase="M1", version="0.2.0")
            self.db.create_table(f"{mod.lower().replace('-', '_')}_records", [{"id": 1, "version": "0.2.0"}])

        self.db.disposition = "UPGRADE_SUCCESS"
        self.assertEqual(len(self.db.applied_modules), len(TOPOLOGICAL_ORDER))
        self.assertEqual(self.db.disposition, "UPGRADE_SUCCESS")
        self.assertEqual(self.db.current_phase, "M1")

    def test_synthetic_partial_failure_preservation(self) -> None:
        """Scenario 2: Partial-failure preservation without silent corruption."""
        # Migrate first 4 modules successfully
        for mod in TOPOLOGICAL_ORDER[:4]:
            self.db.apply_module_migration(mod, phase="M1", version="0.2.0")

        pre_failure_snapshot = self.db.snapshot()

        # Step 5 (MOD-CFG) encounters simulated constraint violation
        failed_module = TOPOLOGICAL_ORDER[4]
        failure_occurred = False
        try:
            # Simulate step failure
            raise RuntimeError(f"simulated migration failure in {failed_module}: syntax error")
        except RuntimeError:
            failure_occurred = True
            # Revert to snapshot
            self.db.restore(pre_failure_snapshot)
            self.db.disposition = "PARTIAL_FAILURE_PRESERVED"

        self.assertTrue(failure_occurred)
        self.assertEqual(self.db.disposition, "PARTIAL_FAILURE_PRESERVED")
        # Ensure only the first 4 modules remain committed
        self.assertEqual(list(self.db.applied_modules.keys()), TOPOLOGICAL_ORDER[:4])
        self.assertNotIn(failed_module, self.db.applied_modules)

    def test_synthetic_backfill_verification(self) -> None:
        """Scenario 3: Non-destructive backfill of legacy synthetic data."""
        # Initial legacy records with missing sha256_digest
        legacy_records = [
            {"id": "rec_001", "tenant_id": "ten_alpha", "payload": "doc1"},
            {"id": "rec_002", "tenant_id": "ten_alpha", "payload": "doc2"},
        ]
        self.db.create_table("rec_audit_events", legacy_records)

        # Apply backfill migration: compute digests non-destructively
        for row in self.db.tables["rec_audit_events"]:
            digest = hashlib.sha256(row["payload"].encode("utf-8")).hexdigest()
            row["sha256_digest"] = digest

        self.db.disposition = "BACKFILL_VERIFIED"
        self.assertEqual(self.db.disposition, "BACKFILL_VERIFIED")

        for row in self.db.tables["rec_audit_events"]:
            self.assertIn("sha256_digest", row)
            self.assertEqual(len(row["sha256_digest"]), 64)

    def test_synthetic_rollback_flow(self) -> None:
        """Scenario 4: Non-destructive rollback to clean checkpoint."""
        baseline_snap = self.db.snapshot()

        # Uncommitted / speculative changes
        self.db.apply_module_migration("MOD-EXPERIMENTAL", phase="M1", version="0.2.1-draft")
        self.db.create_table("temp_uncommitted_table", [{"data": "dirty"}])

        # Rollback operation
        self.db.restore(baseline_snap)
        self.db.disposition = "ROLLBACK_SUCCESS"

        self.assertEqual(self.db.disposition, "ROLLBACK_SUCCESS")
        self.assertNotIn("MOD-EXPERIMENTAL", self.db.applied_modules)
        self.assertNotIn("temp_uncommitted_table", self.db.tables)

    def test_synthetic_forward_fix_flow(self) -> None:
        """Scenario 5: Non-destructive forward-fix rather than table drops."""
        # Migrate initial set
        for mod in TOPOLOGICAL_ORDER[:4]:
            self.db.apply_module_migration(mod, phase="M1", version="0.2.0")

        # Step 5 fails initially, then forward-fix patch is applied
        fixed_step = TOPOLOGICAL_ORDER[4]
        # Forward-fix migration script replaces problematic constraint definition
        self.db.apply_module_migration(fixed_step, phase="M2", version="0.2.0-patch1")

        # Remaining steps complete
        for mod in TOPOLOGICAL_ORDER[5:]:
            self.db.apply_module_migration(mod, phase="M1", version="0.2.0")

        self.db.disposition = "FORWARD_FIX_SUCCESS"
        self.assertEqual(self.db.disposition, "FORWARD_FIX_SUCCESS")
        self.assertEqual(len(self.db.applied_modules), len(TOPOLOGICAL_ORDER))
        self.assertEqual(self.db.applied_modules[fixed_step]["phase"], "M2")

    def test_synthetic_backup_and_restore_verification(self) -> None:
        """Scenarios 6 & 7: Synthetic backup creation and exact restore verification."""
        for mod in TOPOLOGICAL_ORDER:
            self.db.apply_module_migration(mod, phase="M1", version="0.2.0")
            self.db.create_table(
                f"{mod.lower().replace('-', '_')}_data",
                [{"id": "001", "tenant_id": "ten_alpha", "content": f"data_{mod}"}],
            )

        # 1. Backup
        backup_snapshot = self.db.snapshot()
        backup_digest = self.db.digest()
        self.db.disposition = "BACKUP_COMPLETED"
        self.assertEqual(self.db.disposition, "BACKUP_COMPLETED")
        self.assertEqual(len(backup_digest), 64)

        # Corrupt current state
        self.db.tables.clear()
        self.db.applied_modules.clear()
        self.assertNotEqual(self.db.digest(), backup_digest)

        # 2. Restore
        self.db.restore(backup_snapshot)
        self.db.disposition = "RESTORE_VERIFIED"

        self.assertEqual(self.db.disposition, "RESTORE_VERIFIED")
        self.assertEqual(self.db.digest(), backup_digest)
        self.assertEqual(len(self.db.applied_modules), len(TOPOLOGICAL_ORDER))


if __name__ == "__main__":
    unittest.main()
