#!/usr/bin/env python3
"""Local validator for module-owned database migration manifest and schema ordering.

Enforces:
1. Unique module/schema/prefix identity across all 9 canonical modules.
2. Topological dependency ordering (DAG) and order_index consistency.
3. Supported phases M0-M2 and prohibition of destructive M3 without owner decision.
4. Forbidden cross-module writes, shared tables, and private joins.
5. Strict provisional, non-runtime, and H020-005 gate invariants.
"""

from __future__ import annotations

import argparse
import json
import pathlib
import sys
import yaml

ROOT = pathlib.Path(__file__).resolve().parents[1]
DEFAULT_MANIFEST_PATH = ROOT / "database" / "migration-manifest.yaml"
DEFAULT_ORDERING_PATH = ROOT / "database" / "schema-ordering.json"

EXPECTED_MODULE_IDS = [
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

MANDATORY_PHASES = {"M0", "M1", "M2"}


def validate_manifest_and_ordering(manifest: dict, ordering: dict) -> list[str]:
    """Validates migration manifest and schema ordering dictionaries against architectural invariants.

    Returns a list of error strings. Empty list indicates full compliance.
    """
    errors: list[str] = []

    # 1. Document Schema & Identity
    if manifest.get("schema_version") != "1.0.0":
        errors.append(f"manifest schema_version must be '1.0.0', got {manifest.get('schema_version')!r}")
    if manifest.get("manifest_id") != "DB-MIG-MANIFEST-001":
        errors.append(f"manifest_id must be 'DB-MIG-MANIFEST-001', got {manifest.get('manifest_id')!r}")
    if manifest.get("lifecycle_status") != "PROVISIONAL":
        errors.append(f"manifest lifecycle_status must be 'PROVISIONAL', got {manifest.get('lifecycle_status')!r}")
    if manifest.get("status") != "PROVISIONAL_PENDING_H020_005":
        errors.append(f"manifest status must be 'PROVISIONAL_PENDING_H020_005', got {manifest.get('status')!r}")
    if manifest.get("human_gate") != "H020-005":
        errors.append(f"manifest human_gate must be 'H020-005', got {manifest.get('human_gate')!r}")
    if manifest.get("execution_state") != "NOT_RUNTIME_EXECUTED":
        errors.append(f"manifest execution_state must be 'NOT_RUNTIME_EXECUTED', got {manifest.get('execution_state')!r}")

    if ordering.get("schema_version") != "1.0.0":
        errors.append(f"ordering schema_version must be '1.0.0', got {ordering.get('schema_version')!r}")
    if ordering.get("ordering_id") != "DB-SCHEMA-ORDERING-001":
        errors.append(f"ordering_id must be 'DB-SCHEMA-ORDERING-001', got {ordering.get('ordering_id')!r}")
    if ordering.get("lifecycle_status") != "PROVISIONAL":
        errors.append(f"ordering lifecycle_status must be 'PROVISIONAL', got {ordering.get('lifecycle_status')!r}")
    if ordering.get("status") != "PROVISIONAL_PENDING_H020_005":
        errors.append(f"ordering status must be 'PROVISIONAL_PENDING_H020_005', got {ordering.get('status')!r}")
    if ordering.get("human_gate") != "H020-005":
        errors.append(f"ordering human_gate must be 'H020-005', got {ordering.get('human_gate')!r}")
    if ordering.get("runtime_execution") is not False:
        errors.append(f"ordering runtime_execution must be False, got {ordering.get('runtime_execution')!r}")

    # 2. Invariants and Forbidden Architectural Rules
    rules = manifest.get("rules", {})
    if rules.get("cross_module_direct_writes") != "PROHIBITED":
        errors.append("rule cross_module_direct_writes must be 'PROHIBITED'")
    if rules.get("shared_tables") != "PROHIBITED":
        errors.append("rule shared_tables must be 'PROHIBITED'")
    if rules.get("private_table_joins") != "PROHIBITED":
        errors.append("rule private_table_joins must be 'PROHIBITED'")

    ordering_invariants = ordering.get("invariants", {})
    if ordering_invariants.get("cross_module_direct_writes") != "PROHIBITED":
        errors.append("ordering invariant cross_module_direct_writes must be 'PROHIBITED'")
    if ordering_invariants.get("shared_tables") != "PROHIBITED":
        errors.append("ordering invariant shared_tables must be 'PROHIBITED'")
    if ordering_invariants.get("private_table_joins") != "PROHIBITED":
        errors.append("ordering invariant private_table_joins must be 'PROHIBITED'")
    if ordering_invariants.get("destructive_migrations_m3") != "PROHIBITED_WITHOUT_OWNER_DECISION":
        errors.append("ordering invariant destructive_migrations_m3 must be 'PROHIBITED_WITHOUT_OWNER_DECISION'")

    # 3. Module Identity and Uniqueness
    modules = manifest.get("modules", [])
    if not isinstance(modules, list):
        errors.append("manifest 'modules' must be a list")
        return errors

    module_ids = [m.get("module_id") for m in modules]
    if module_ids != EXPECTED_MODULE_IDS:
        errors.append(f"manifest modules sequence mismatch: expected {EXPECTED_MODULE_IDS}, got {module_ids}")

    namespaces: list[str] = []
    prefixes: list[str] = []
    order_indices: list[int] = []
    deps_by_mod: dict[str, list[str]] = {}

    for i, mod in enumerate(modules, start=1):
        mid = mod.get("module_id")
        ns = mod.get("schema_namespace")
        pfx = mod.get("table_prefix")
        idx = mod.get("order_index")
        phases = mod.get("phases_supported", [])
        deps = mod.get("dependency_module_ids", [])

        if not mid or not isinstance(mid, str):
            errors.append(f"module item {i} missing valid 'module_id'")
            continue

        if ns:
            namespaces.append(ns)
        else:
            errors.append(f"module {mid} missing 'schema_namespace'")

        if pfx:
            prefixes.append(pfx)
        else:
            errors.append(f"module {mid} missing 'table_prefix'")

        if idx is not None:
            order_indices.append(idx)
            if idx != i:
                errors.append(f"module {mid} order_index mismatch: expected {i}, got {idx}")
        else:
            errors.append(f"module {mid} missing 'order_index'")

        # Validate phases supported
        phases_set = set(phases)
        if not MANDATORY_PHASES.issubset(phases_set):
            errors.append(f"module {mid} missing mandatory phases {MANDATORY_PHASES - phases_set}")
        if "M3" in phases_set:
            errors.append(f"module {mid} includes destructive phase 'M3' which is prohibited without owner decision")

        deps_by_mod[mid] = deps

    if len(namespaces) != len(set(namespaces)):
        errors.append("duplicate schema_namespace found in manifest modules")
    if len(prefixes) != len(set(prefixes)):
        errors.append("duplicate table_prefix found in manifest modules")
    if len(order_indices) != len(set(order_indices)):
        errors.append("duplicate order_index found in manifest modules")

    # 4. Topological Ordering & Dependency Precedence (DAG validation)
    topological = ordering.get("topological_ordering", [])
    if topological != EXPECTED_MODULE_IDS:
        errors.append(f"topological_ordering mismatch: expected {EXPECTED_MODULE_IDS}, got {topological}")

    seen_modules: set[str] = set()
    for mod_id in topological:
        deps = deps_by_mod.get(mod_id, [])
        for dep in deps:
            if dep not in EXPECTED_MODULE_IDS:
                errors.append(f"module {mod_id} depends on unknown module {dep!r}")
            elif dep not in seen_modules:
                errors.append(f"dependency ordering violation: {mod_id} depends on {dep} which has not appeared earlier in the topological DAG")
        seen_modules.add(mod_id)

    # Detect cycles via depth-first search
    def has_cycle(current: str, visiting: set[str], visited: set[str]) -> bool:
        visiting.add(current)
        for neighbor in deps_by_mod.get(current, []):
            if neighbor in visiting:
                return True
            if neighbor not in visited:
                if has_cycle(neighbor, visiting, visited):
                    return True
        visiting.remove(current)
        visited.add(current)
        return False

    visited_mods: set[str] = set()
    for mod_id in deps_by_mod:
        if mod_id not in visited_mods:
            if has_cycle(mod_id, set(), visited_mods):
                errors.append(f"cyclic dependency detected involving module {mod_id}")
                break

    return errors


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="Validate module database migration manifest and schema ordering.")
    parser.add_argument("--manifest", type=pathlib.Path, default=DEFAULT_MANIFEST_PATH, help="Path to migration-manifest.yaml")
    parser.add_argument("--ordering", type=pathlib.Path, default=DEFAULT_ORDERING_PATH, help="Path to schema-ordering.json")

    args = parser.parse_args(argv)

    if not args.manifest.is_file():
        print(f"ERROR: migration manifest not found at {args.manifest}", file=sys.stderr)
        return 1
    if not args.ordering.is_file():
        print(f"ERROR: schema ordering not found at {args.ordering}", file=sys.stderr)
        return 1

    try:
        manifest_data = yaml.safe_load(args.manifest.read_text(encoding="utf-8"))
    except Exception as exc:
        print(f"ERROR: failed to parse YAML manifest at {args.manifest}: {exc}", file=sys.stderr)
        return 1

    try:
        ordering_data = json.loads(args.ordering.read_text(encoding="utf-8"))
    except Exception as exc:
        print(f"ERROR: failed to parse JSON ordering at {args.ordering}: {exc}", file=sys.stderr)
        return 1

    validation_errors = validate_manifest_and_ordering(manifest_data, ordering_data)
    if validation_errors:
        print(f"Migration manifest validation FAILED with {len(validation_errors)} error(s):", file=sys.stderr)
        for err in validation_errors:
            print(f"  - {err}", file=sys.stderr)
        return 1

    print("Migration manifest and schema ordering validation PASSED.")
    print("All 9 canonical modules, topological DAG ordering, and forbidden architectural invariants verified.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
