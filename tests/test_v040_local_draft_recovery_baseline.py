import os
import re
import unittest
import yaml

class TestV040LocalDraftRecoveryBaseline(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.doc_path = os.path.join(
            os.path.dirname(os.path.dirname(os.path.abspath(__file__))),
            "docs",
            "architecture",
            "v040-local-draft-recovery-baseline.md"
        )
        if not os.path.exists(cls.doc_path):
            raise FileNotFoundError(f"Local draft recovery baseline not found at {cls.doc_path}")

        with open(cls.doc_path, "r", encoding="utf-8") as f:
            cls.content = f.read()

        # Extract frontmatter
        fm_match = re.match(r"^---\s*\n(.*?)\n---\s*\n", cls.content, re.DOTALL)
        if not fm_match:
            raise ValueError("Frontmatter not found in local draft recovery baseline")
        cls.frontmatter = yaml.safe_load(fm_match.group(1))

        # Extract synthetic operations fixture YAML
        yaml_blocks = re.findall(r"```yaml\s*\n(.*?)\n```", cls.content, re.DOTALL)
        cls.fixture = None
        for block in yaml_blocks:
            try:
                parsed = yaml.safe_load(block)
                if isinstance(parsed, dict) and "fixture_id" in parsed and "scenarios" in parsed:
                    cls.fixture = parsed
                    break
            except Exception:
                continue

    def test_document_metadata_and_frontmatter(self):
        """Validates document metadata, identifiers, and governance gates in YAML frontmatter."""
        fm = self.frontmatter
        self.assertEqual(fm.get("document_id"), "ARC-V040-RECOV-001")
        self.assertEqual(fm.get("governing_issue"), "GitHub Issue #125")
        self.assertEqual(fm.get("authority_source"), "HDEC-V040-FOUNDATION-054")
        self.assertEqual(fm.get("milestone"), "v0.4.0 - OSHE Inspect Private Alpha")
        self.assertEqual(fm.get("lifecycle_status"), "DRAFT")
        self.assertEqual(fm.get("status"), "APPROVED_FOR_LOCAL_DEVELOPMENT")

        # Consumed prework and pending integration
        consumed = fm.get("consumed_prework_artifacts", [])
        self.assertTrue(any("ARC-V040-EVD-001" in item for item in consumed))

        pending = fm.get("pending_integrations", [])
        self.assertIn("PENDING_INTEGRATION_ISSUE_128_V040_I017", pending)

        # Foundation gates H040-001 through H040-006 must be listed
        approved_gates = fm.get("approved_foundation_gates", [])
        for g in ["H040-001", "H040-002", "H040-003", "H040-004", "H040-005", "H040-006"]:
            self.assertIn(g, approved_gates, f"Missing approved gate {g}")

        # Retained holds H040-007 through H040-011 must be listed
        retained_holds = fm.get("retained_holds", [])
        for h in ["H040-007", "H040-008", "H040-009", "H040-010", "H040-011"]:
            self.assertIn(h, retained_holds, f"Missing retained hold {h}")

        # Unselected human-owned policies
        unselected = fm.get("retained_unselected_policies", {})
        self.assertEqual(unselected.get("binding_scoring_policy"), "HUMAN_OWNED_UNSELECTED")
        self.assertEqual(unselected.get("finding_closure_policy"), "HUMAN_OWNED_UNSELECTED")
        self.assertEqual(unselected.get("offline_authority"), "HUMAN_OWNED_UNSELECTED")
        self.assertEqual(unselected.get("evidence_retention_policy"), "HUMAN_OWNED_UNSELECTED")
        self.assertEqual(unselected.get("external_storage_provider_policy"), "HUMAN_OWNED_UNSELECTED")

    def test_local_draft_storage_architecture(self):
        """Validates IndexedDB object stores for local drafts and write-ahead journaling."""
        self.assertIn("oshe_offline_db", self.content)
        self.assertIn("work_packages", self.content)
        self.assertIn("response_drafts", self.content)
        self.assertIn("evidence_queue", self.content)
        self.assertIn("evidence_blobs", self.content)
        self.assertIn("mutation_journal", self.content)

    def test_sync_state_machine_enumeration(self):
        """Validates synchronization states for locally authored items."""
        states = [
            "DRAFT_LOCAL",
            "QUEUED_FOR_SYNC",
            "SYNCING",
            "SYNC_ACKNOWLEDGED",
            "SYNC_FAILED_RETRYABLE",
            "SYNC_QUARANTINED",
        ]
        for st in states:
            self.assertIn(st, self.content, f"Missing sync state {st}")

    def test_interruption_recovery_domains(self):
        """Validates interruption recovery across network drops, tab crashes, and device shutdowns."""
        self.assertIn("Network Disconnection & Flapping", self.content)
        self.assertIn("Browser Tab Closure", self.content)
        self.assertIn("NFR-REC-01", self.content)
        self.assertIn("100% fidelity", self.content)
        self.assertIn("Device Power Loss", self.content)

    def test_idempotent_retries_and_conflict_quarantine(self):
        """Validates mutation idempotency key composition and server-side quarantine."""
        self.assertIn("mutation_key", self.content)
        self.assertIn("client_seq_num", self.content)
        self.assertIn("Idempotent Acknowledgment", self.content)
        self.assertIn("Conflict Quarantine Trigger", self.content)
        self.assertIn("H040-005", self.content)

    def test_low_storage_defenses_and_quota_ceilings(self):
        """Validates low storage detection, photo lock, and zero silent eviction."""
        self.assertIn("navigator.storage.estimate", self.content)
        self.assertIn("ErrLocalStorageQuotaExceeded", self.content)
        self.assertIn("Zero Silent Eviction", self.content)
        self.assertIn("Draft Provenance Preservation Invariant", self.content)

    def test_visible_user_guidance_and_ui_indicators(self):
        """Validates global header badges and item-level sync markers."""
        self.assertIn("Global Header Status Banner", self.content)
        self.assertIn("บันทึกข้อมูลเรียบร้อย", self.content)
        self.assertIn("Item-Level Sync Status Markers", self.content)
        self.assertIn("PENDING_LOCAL", self.content)

    def test_issue_128_open_preservation(self):
        """Validates that Issue #128 is explicitly preserved as open pending integration."""
        self.assertIn("PENDING_INTEGRATION_ISSUE_128_V040_I017", self.content)
        self.assertIn("preserved as OPEN", self.content)
        self.assertIn("Issue #128", self.content)

    def test_synthetic_operations_fixtures(self):
        """Validates synthetic operations fixture YAML structure and scenarios."""
        self.assertIsNotNone(self.fixture, "Synthetic operations fixture YAML block not found")
        fix = self.fixture

        self.assertEqual(fix.get("fixture_id"), "fix_syn_local_draft_recovery_v1")
        scenarios = fix.get("scenarios", [])
        self.assertEqual(len(scenarios), 5, "Fixture must define exactly 5 operational scenarios")

        # Scenario 1: Tab Crash
        s1 = scenarios[0]
        self.assertEqual(s1.get("scenario_id"), "scen_rec_tab_crash_01")
        self.assertEqual(s1.get("recovery_outcome"), "100_PERCENT_RESTORED")

        # Scenario 2: Network Drop
        s2 = scenarios[1]
        self.assertEqual(s2.get("scenario_id"), "scen_rec_network_drop_02")
        self.assertEqual(s2.get("post_drop_sync_state"), "SYNC_FAILED_RETRYABLE")

        # Scenario 3: Low Storage
        s3 = scenarios[2]
        self.assertEqual(s3.get("scenario_id"), "scen_rec_low_storage_03")
        self.assertEqual(s3.get("expected_result"), "FAIL_CLOSED_LOCK_PHOTO")
        self.assertFalse(s3.get("drafts_purged"))

        # Scenario 4: Idempotent Retry
        s4 = scenarios[3]
        self.assertEqual(s4.get("scenario_id"), "scen_rec_idempotent_retry_04")
        self.assertFalse(s4.get("duplicate_created"))
        self.assertEqual(s4.get("final_sync_state"), "SYNC_ACKNOWLEDGED")

        # Scenario 5: Conflict Quarantine
        s5 = scenarios[4]
        self.assertEqual(s5.get("scenario_id"), "scen_rec_conflict_quarantine_05")
        self.assertEqual(s5.get("client_sync_state"), "SYNC_QUARANTINED")
        self.assertFalse(s5.get("data_discarded"))

if __name__ == "__main__":
    unittest.main()
