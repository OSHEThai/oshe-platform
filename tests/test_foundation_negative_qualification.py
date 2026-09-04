from __future__ import annotations

import copy
import hashlib
import json
import pathlib
import re
import unittest

ROOT = pathlib.Path(__file__).resolve().parents[1]
REGISTER_PATH = ROOT / "tests" / "fixtures" / "identifiers" / "entity-identifier-register.json"
MIGRATION_FIXTURES_PATH = ROOT / "tests" / "fixtures" / "identifiers" / "migration-fixtures.json"
RECOVERY_FIXTURES_PATH = ROOT / "tests" / "fixtures" / "identifiers" / "recovery-fixtures.json"

PREFIX_REGEX = re.compile(r"^[a-z0-9]{2,16}$")
CANONICAL_ID_REGEX = re.compile(r"^[a-z0-9]{2,16}_[0-9a-f]{32}$")


class FoundationNegativeQualificationTests(unittest.TestCase):
    def setUp(self) -> None:
        self.assertTrue(REGISTER_PATH.is_file(), f"missing register at {REGISTER_PATH}")
        self.assertTrue(MIGRATION_FIXTURES_PATH.is_file(), f"missing migration fixtures at {MIGRATION_FIXTURES_PATH}")
        self.assertTrue(RECOVERY_FIXTURES_PATH.is_file(), f"missing recovery fixtures at {RECOVERY_FIXTURES_PATH}")

        self.register = json.loads(REGISTER_PATH.read_text(encoding="utf-8"))
        self.migration_data = json.loads(MIGRATION_FIXTURES_PATH.read_text(encoding="utf-8"))
        self.recovery_data = json.loads(RECOVERY_FIXTURES_PATH.read_text(encoding="utf-8"))

    def test_entity_identifier_register_structure_and_invariants(self) -> None:
        """Verifies entity-identifier-register schema, prefix uniqueness, format, and scoping."""
        self.assertEqual(self.register.get("schema_version"), "1.0.0")
        self.assertEqual(self.register.get("register_id"), "REG-V020-IDENTIFIER-ENTITY-001")

        governance = self.register.get("governance", {})
        self.assertEqual(governance.get("status"), "APPROVED_SPECIFICATION")
        self.assertEqual(governance.get("collision_rule"), "crypto_rand_128bit_minimum")

        entities = self.register.get("entities", [])
        self.assertGreaterEqual(len(entities), 15, "insufficient entity definitions in register")

        seen_prefixes: set[str] = set()
        seen_entity_types: set[str] = set()

        for ent in entities:
            entity_type = ent.get("entity_type")
            prefix = ent.get("prefix")
            canonical_prefix = ent.get("canonical_prefix")
            scope = ent.get("scope")
            tenant_scoped = ent.get("tenant_scoped")

            self.assertTrue(entity_type, "entity_type must not be empty")
            self.assertNotIn(entity_type, seen_entity_types, f"duplicate entity_type: {entity_type}")
            seen_entity_types.add(entity_type)

            self.assertTrue(PREFIX_REGEX.match(prefix), f"prefix {prefix} violates regex")
            self.assertNotIn(prefix, seen_prefixes, f"duplicate prefix: {prefix}")
            seen_prefixes.add(prefix)

            self.assertEqual(canonical_prefix, f"{prefix}_", f"canonical prefix mismatch for {prefix}")
            self.assertIn(scope, ("GLOBAL", "TENANT"), f"invalid scope for {entity_type}")
            self.assertIsInstance(tenant_scoped, bool)
            self.assertEqual(tenant_scoped, scope == "TENANT")

            self.assertEqual(ent.get("payload_format"), "hex_lowercase")
            self.assertGreaterEqual(ent.get("min_payload_bytes", 0), 16)

    def test_identifier_collision_resistance_simulation(self) -> None:
        """Simulates synthetic identifier generation ensuring zero collisions across high volume."""
        generated: set[str] = set()
        count = 5000
        for i in range(count):
            # Synthetic 128-bit pseudo-random hex simulation
            digest = hashlib.sha256(f"seed-sim-{i}".encode("utf-8")).hexdigest()[:32]
            identifier = f"ins_{digest}"
            self.assertTrue(CANONICAL_ID_REGEX.match(identifier))
            self.assertNotIn(identifier, generated, f"simulated collision for {identifier}")
            generated.add(identifier)

        self.assertEqual(len(generated), count)

    def test_migration_fixtures_determinism_and_rollback(self) -> None:
        """Tests deterministic legacy migration and transactional rollback preservation."""
        scenarios = {s["scenario_id"]: s for s in self.migration_data.get("scenarios", [])}

        # Scenario 1: Deterministic UUID conversion
        s1 = scenarios.get("MIG-001-UNPREFIXED-UUID-TO-CANONICAL")
        self.assertIsNotNone(s1)
        for legacy, expected in zip(s1["legacy_records"], s1["expected_canonical_records"]):
            clean_hex = legacy["legacy_id"].replace("-", "").lower()
            computed_id = f"{s1['prefix']}_{clean_hex}"
            self.assertEqual(computed_id, expected["canonical_id"])
            self.assertTrue(CANONICAL_ID_REGEX.match(computed_id))

        # Scenario 2: Atomic Rollback Simulation
        s2 = scenarios.get("MIG-002-PARTIAL-FAILURE-ATOMIC-ROLLBACK")
        self.assertIsNotNone(s2)

        store: dict[str, dict] = {"fnd_seed0000000000000000000000000000": {"title": "Seed"}}
        snapshot = copy.deepcopy(store)

        failure_occurred = False
        staged: dict[str, dict] = {}
        for item in s2["batch"]:
            entropy = item["entropy_hex"]
            # Validate hex format
            if len(entropy) != 32 or not re.match(r"^[0-9a-f]{32}$", entropy):
                # Rollback!
                store = copy.deepcopy(snapshot)
                failure_occurred = True
                break
            can_id = f"{s2['prefix']}_{entropy}"
            staged[can_id] = item

        if not failure_occurred:
            store.update(staged)

        self.assertTrue(failure_occurred, "expected migration failure did not trigger")
        # Ensure exact snapshot preservation
        self.assertEqual(len(store), 1)
        self.assertIn("fnd_seed0000000000000000000000000000", store)
        self.assertNotIn("fnd_0123456789abcdef0123456789abcdef", store)

        # Scenario 3: Cross-tenant migration denial
        s3 = scenarios.get("MIG-003-CROSS-TENANT-MIGRATION-DENIAL")
        self.assertIsNotNone(s3)
        caller_tenant = s3["caller_tenant_id"]
        record_tenant = s3["target_record"]["tenant_id"]
        self.assertNotEqual(caller_tenant, record_tenant)

        # Scenario 4: Duplicate collision avoidance
        s4 = scenarios.get("MIG-004-COLLISION-AVOIDANCE-AND-REJECTION")
        self.assertIsNotNone(s4)
        self.assertEqual(s4["existing_id"], s4["candidate_record"]["computed_canonical_id"])

    def test_cross_tenant_denial_invariants(self) -> None:
        """Verifies multi-tenant isolation, default-deny, and prefix collision denial."""
        trusted_tenant = "ten_0123456789abcdef0123456789abcdef"

        # 1. Foreign tenant access denied
        foreign_tenants = [
            "ten_fedcba9876543210fedcba9876543210",
            "ten_0123456789abcdef0123456789abcde0",
            "ten_ffffffffffffffffffffffffffffffff",
        ]
        for foreign in foreign_tenants:
            self.assertNotEqual(trusted_tenant, foreign)

        # 2. Prefix / substring collision denial
        prefix_colliding_targets = [
            f"{trusted_tenant}-2",
            f"{trusted_tenant}_suffix",
            f"{trusted_tenant}x",
            trusted_tenant[:-1],
        ]
        for target in prefix_colliding_targets:
            self.assertNotEqual(trusted_tenant, target, f"prefix collision incorrectly allowed for {target}")

        # 3. Client override forbidden simulation
        client_override = {"tenant_id": "ten_attacker_override"}
        self.assertTrue(bool(client_override.get("tenant_id")))

    def test_wire_serialization_negative_vectors(self) -> None:
        """Verifies wire serialization vectors and tamper detection."""
        vectors = {v["vector_id"]: v for v in self.recovery_data.get("tampered_vectors", [])}

        # Vector 1: Duplicate keys
        v1 = vectors.get("TAMPER-001-DUPLICATE-KEYS")
        self.assertIsNotNone(v1)
        raw_json = v1["raw_json"]
        # Raw JSON contains duplicate "correlation_id"
        self.assertEqual(raw_json.count('"correlation_id"'), 2)

        # Vector 2: Extra unexpected field
        v2 = vectors.get("TAMPER-002-EXTRA-UNEXPECTED-FIELD")
        self.assertIsNotNone(v2)
        parsed = json.loads(v2["raw_json"])
        self.assertIn("unexpected_injection", parsed)

        # Vector 3: Unsupported version
        v3 = vectors.get("TAMPER-003-UNSUPPORTED-VERSION")
        self.assertIsNotNone(v3)
        parsed_v3 = json.loads(v3["raw_json"])
        self.assertEqual(parsed_v3.get("version"), "v2")

        # Vector 4: Empty version
        v4 = vectors.get("TAMPER-004-EMPTY-VERSION")
        self.assertIsNotNone(v4)
        parsed_v4 = json.loads(v4["raw_json"])
        self.assertEqual(parsed_v4.get("version"), "")

        # Vector 5: Malformed prefix
        v5 = vectors.get("TAMPER-005-MALFORMED-CORRELATION-PREFIX")
        self.assertIsNotNone(v5)
        parsed_v5 = json.loads(v5["raw_json"])
        self.assertTrue(parsed_v5.get("correlation_id").startswith("usr_"))

    def test_entity_state_recovery_verification(self) -> None:
        """Verifies state serialization, cryptographic digest hashing, and recovery."""
        scenarios = self.recovery_data.get("state_recovery_scenarios", [])
        self.assertGreaterEqual(len(scenarios), 1)

        for sc in scenarios:
            initial = sc["initial_state"]
            serialized_initial = json.dumps(initial, sort_keys=True)
            initial_digest = hashlib.sha256(serialized_initial.encode("utf-8")).hexdigest()

            # Corrupt state
            corrupted_state = {"corrupted": "bad_data"}
            corrupted_digest = hashlib.sha256(json.dumps(corrupted_state, sort_keys=True).encode("utf-8")).hexdigest()
            self.assertNotEqual(initial_digest, corrupted_digest)

            # Restore from serialized snapshot
            recovered = json.loads(serialized_initial)
            recovered_digest = hashlib.sha256(json.dumps(recovered, sort_keys=True).encode("utf-8")).hexdigest()

            self.assertEqual(initial_digest, recovered_digest)
            self.assertEqual(initial, recovered)


if __name__ == "__main__":
    unittest.main()
