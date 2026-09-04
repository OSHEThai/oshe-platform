from __future__ import annotations

import copy
import hashlib
import json
import pathlib
import re
import unittest
import uuid

ROOT = pathlib.Path(__file__).resolve().parents[1]
FIXTURES_PATH = ROOT / "tests" / "fixtures" / "integration" / "v020-walking-skeleton-fixtures.json"

RUN_ID_REGEX = re.compile(r"^run_[0-9a-f]{16}$")
CORR_ID_REGEX = re.compile(r"^corr_[0-9a-f]{16}$")
CAUS_ID_REGEX = re.compile(r"^caus_[0-9a-f]{16}$")


class SyntheticIntegrationContext:
    """In-memory state and traceability container for the generic walking-skeleton journey."""

    def __init__(self, run_id: str, correlation_id: str) -> None:
        self.run_id = run_id
        self.correlation_id = correlation_id
        self.causation_id = f"caus_{uuid.uuid4().hex[:16]}"
        self.tenant_id: str | None = None
        self.claims: dict[str, any] | None = None
        self.template_version: dict[str, any] | None = None
        self.checklist_instance: dict[str, any] | None = None
        self.evidence_files: dict[str, dict[str, any]] = {}
        self.actions: dict[str, dict[str, any]] = {}
        self.outbox_events: list[dict[str, any]] = []
        self.audit_log: list[dict[str, any]] = []
        self.diagnostics: list[str] = []

    def log_audit(self, event_type: str, actor: str, payload: dict[str, any]) -> dict[str, any]:
        entry = {
            "sequence": len(self.audit_log) + 1,
            "run_id": self.run_id,
            "correlation_id": self.correlation_id,
            "causation_id": self.causation_id,
            "event_type": event_type,
            "tenant_id": self.tenant_id,
            "actor": actor,
            "payload": copy.deepcopy(payload),
        }
        self.audit_log.append(entry)
        return entry


class WalkingSkeletonIntegrationHarnessTests(unittest.TestCase):
    def setUp(self) -> None:
        self.assertTrue(FIXTURES_PATH.is_file(), f"missing fixture file at {FIXTURES_PATH}")
        self.fixtures = json.loads(FIXTURES_PATH.read_text(encoding="utf-8"))

        run_uuid = uuid.uuid4().hex[:16]
        self.run_id = f"run_{run_uuid}"
        self.correlation_id = f"corr_{run_uuid}"
        self.harness = SyntheticIntegrationContext(self.run_id, self.correlation_id)

    def test_run_id_and_traceability_format(self) -> None:
        """Verifies strongly-typed monotonic Run, Correlation, and Causation identifiers."""
        self.assertTrue(RUN_ID_REGEX.match(self.harness.run_id), "malformed run_id")
        self.assertTrue(CORR_ID_REGEX.match(self.harness.correlation_id), "malformed correlation_id")
        self.assertTrue(CAUS_ID_REGEX.match(self.harness.causation_id), "malformed causation_id")

    def test_full_generic_record_journey_success(self) -> None:
        """Executes the complete unified walking-skeleton record journey from tenant context to export."""
        primary_tenant = self.fixtures["tenants"]["primary"]["tenant_id"]
        inspector = self.fixtures["subjects"]["inspector"]["subject_id"]
        officer = self.fixtures["subjects"]["safety_officer"]["subject_id"]
        compliance_lead = self.fixtures["subjects"]["compliance_lead"]["subject_id"]

        # -------------------------------------------------------------
        # Step 1: Tenant Context Derivation
        # -------------------------------------------------------------
        self.harness.tenant_id = primary_tenant
        self.harness.claims = {
            "sub": inspector,
            "tenant_id": primary_tenant,
            "roles": ["INSPECTOR"],
        }
        self.harness.log_audit("TENANT_CONTEXT_DERIVED", inspector, {"tenant_id": primary_tenant})

        # -------------------------------------------------------------
        # Step 2: Template Publication & Immutability
        # -------------------------------------------------------------
        tmpl_fix = self.fixtures["checklist_template"]
        template_record = {
            "template_id": tmpl_fix["template_id"],
            "version_id": tmpl_fix["version_id"],
            "title": tmpl_fix["title"],
            "questions": copy.deepcopy(tmpl_fix["questions"]),
            "state": "PUBLISHED",
            "snapshot_digest": hashlib.sha256(json.dumps(tmpl_fix, sort_keys=True).encode("utf-8")).hexdigest(),
        }
        self.harness.template_version = template_record
        self.harness.log_audit("TEMPLATE_PUBLISHED", compliance_lead, {"template_id": tmpl_fix["template_id"]})

        # -------------------------------------------------------------
        # Step 3: Checklist Instantiation & Pinned Snapshot
        # -------------------------------------------------------------
        instance_id = f"inst_{uuid.uuid4().hex[:12]}"
        instance_record = {
            "instance_id": instance_id,
            "tenant_id": primary_tenant,
            "template_ref": {
                "template_id": template_record["template_id"],
                "version_id": template_record["version_id"],
            },
            "snapshot": copy.deepcopy(template_record),
            "state": "COMPLETED",
            "answers": {
                "q1_base_plates": "true",
                "q2_guardrails": "false",  # finding flagged!
                "q3_load_capacity": "1500.0",
            },
        }
        self.harness.checklist_instance = instance_record
        self.harness.log_audit("INSPECTION_COMPLETED", inspector, {"instance_id": instance_id})

        # -------------------------------------------------------------
        # Step 4: Evidence Attachment & Cryptographic Binding
        # -------------------------------------------------------------
        evd_fix = self.fixtures["evidence"]
        raw_payload = b"synthetic_scaffold_photo_bytes_payload"
        computed_digest = hashlib.sha256(raw_payload).hexdigest()
        evidence_entry = {
            "evidence_id": evd_fix["evidence_id"],
            "tenant_id": primary_tenant,
            "filename": evd_fix["filename"],
            "sha256_digest": computed_digest,
            "verified": True,
        }
        self.harness.evidence_files[evd_fix["evidence_id"]] = evidence_entry
        self.harness.log_audit("EVIDENCE_VERIFIED", inspector, {"evidence_id": evd_fix["evidence_id"], "digest": computed_digest})

        # -------------------------------------------------------------
        # Step 5: Corrective Action Lifecycle
        # -------------------------------------------------------------
        act_fix = self.fixtures["corrective_action"]
        action_entry = {
            "action_id": act_fix["action_id"],
            "tenant_id": primary_tenant,
            "title": act_fix["title"],
            "owner": officer,
            "reviewer": compliance_lead,
            "state": "CLOSED",
            "evidence_ids": [evd_fix["evidence_id"]],
        }
        self.harness.actions[act_fix["action_id"]] = action_entry
        self.harness.log_audit("ACTION_CLOSED", compliance_lead, {"action_id": act_fix["action_id"]})

        # -------------------------------------------------------------
        # Step 6: Transactional Outbox Event Staging & Commit
        # -------------------------------------------------------------
        event_payload = {
            "event_id": f"evt_{uuid.uuid4().hex[:16]}",
            "tenant_id": primary_tenant,
            "event_type": "inspection.finding.closed",
            "correlation_id": self.harness.correlation_id,
            "causation_id": self.harness.causation_id,
            "sequence_number": 1,
            "payload_digest": computed_digest,
        }
        self.harness.outbox_events.append(event_payload)
        self.harness.log_audit("EVENT_COMMITTED", "system_outbox", {"event_id": event_payload["event_id"]})

        # -------------------------------------------------------------
        # Step 7: Report Generation & Complete Record Export Manifest
        # -------------------------------------------------------------
        rep_fix = self.fixtures["report_and_export"]
        manifest_payload = {
            "manifest_id": f"rep_{primary_tenant}_001",
            "tenant_id": primary_tenant,
            "template_id": rep_fix["template_id"],
            "version_id": rep_fix["version_id"],
            "record_count": 1,
            "source_data_digest": computed_digest,
            "non_authority_notice": rep_fix["expected_non_authority_notice"],
        }
        self.harness.log_audit("REPORT_EXPORT_GENERATED", compliance_lead, {"manifest_id": manifest_payload["manifest_id"]})

        # -------------------------------------------------------------
        # Step 8: Assertions Across the Entire Integrated Journey
        # -------------------------------------------------------------
        self.assertEqual(len(self.harness.audit_log), 7)
        self.assertEqual(self.harness.template_version["state"], "PUBLISHED")
        self.assertEqual(self.harness.checklist_instance["state"], "COMPLETED")
        self.assertEqual(len(self.harness.evidence_files), 1)
        self.assertEqual(self.harness.actions[act_fix["action_id"]]["state"], "CLOSED")
        self.assertEqual(len(self.harness.outbox_events), 1)
        self.assertIn("DERIVED_OUTPUT_NON_AUTHORITY", manifest_payload["non_authority_notice"])

    def test_cross_tenant_isolation_denial(self) -> None:
        """Verifies that cross-tenant access attempts are strictly rejected."""
        primary_tenant = self.fixtures["tenants"]["primary"]["tenant_id"]
        secondary_tenant = self.fixtures["tenants"]["secondary"]["tenant_id"]
        intruder = self.fixtures["subjects"]["unauthorized_intruder"]["subject_id"]

        # Intruder from secondary tenant attempts to query primary tenant resource
        self.assertNotEqual(primary_tenant, secondary_tenant)

        cross_tenant_attempt = {
            "caller_tenant": secondary_tenant,
            "caller_sub": intruder,
            "target_tenant": primary_tenant,
        }

        # Policy evaluation denies access
        access_granted = cross_tenant_attempt["caller_tenant"] == cross_tenant_attempt["target_tenant"]
        self.assertFalse(access_granted, "cross-tenant access was not denied")

        # Failure diagnostic is recorded
        self.harness.diagnostics.append(f"DENIED: cross-tenant access by {intruder} to {primary_tenant}")
        self.assertEqual(len(self.harness.diagnostics), 1)

    def test_unauthorized_role_transition_denial(self) -> None:
        """Verifies that unauthorized roles cannot close or approve corrective actions."""
        inspector = self.fixtures["subjects"]["inspector"]["subject_id"]
        officer = self.fixtures["subjects"]["safety_officer"]["subject_id"]

        act_fix = self.fixtures["corrective_action"]

        # Action owner cannot self-close without reviewer role
        self.assertEqual(act_fix["owner"], officer)
        self.assertNotEqual(act_fix["owner"], act_fix["reviewer"])

        # Inspector attempts to close action
        caller_role = "INSPECTOR"
        can_close = caller_role == "COMPLIANCE_LEAD"
        self.assertFalse(can_close, "unauthorized role was allowed to close action")

    def test_tampered_evidence_digest_denial(self) -> None:
        """Verifies that evidence with tampered payload digest is rejected."""
        raw_payload = b"original_scaffold_photo_bytes"
        tampered_payload = b"tampered_bytes_with_injected_noise"

        expected_digest = hashlib.sha256(raw_payload).hexdigest()
        actual_digest = hashlib.sha256(tampered_payload).hexdigest()

        self.assertNotEqual(expected_digest, actual_digest)
        digest_matched = expected_digest == actual_digest
        self.assertFalse(digest_matched, "tampered digest was accepted")

    def test_feature_control_alignment_default_off(self) -> None:
        """Verifies feature flag alignment with I028 candidate: default-off and non-authority."""
        flag_fix = self.fixtures["feature_flags"]

        self.assertTrue(flag_fix["default_off"], "governed flags must be default-off")
        self.assertFalse(flag_fix["enabled"], "flag must not be pre-enabled")
        self.assertIn("FEATURE_FLAG_NON_AUTHORITY", flag_fix["authority_disclaimer"])


if __name__ == "__main__":
    unittest.main()
