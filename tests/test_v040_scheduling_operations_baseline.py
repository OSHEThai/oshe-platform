import os
import re
import unittest
import yaml

class TestV040SchedulingOperationsBaseline(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.doc_path = os.path.join(
            os.path.dirname(os.path.dirname(os.path.abspath(__file__))),
            "docs",
            "architecture",
            "v040-scheduling-operations-baseline.md"
        )
        if not os.path.exists(cls.doc_path):
            raise FileNotFoundError(f"Scheduling operations baseline not found at {cls.doc_path}")

        with open(cls.doc_path, "r", encoding="utf-8") as f:
            cls.content = f.read()

        # Extract frontmatter
        fm_match = re.match(r"^---\s*\n(.*?)\n---\s*\n", cls.content, re.DOTALL)
        if not fm_match:
            raise ValueError("Frontmatter not found in scheduling operations baseline")
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
        self.assertEqual(fm.get("document_id"), "ARC-V040-SCHEDOPS-001")
        self.assertEqual(fm.get("governing_issue"), "GitHub Issue #122")
        self.assertEqual(fm.get("authority_source"), "HDEC-V040-FOUNDATION-054")
        self.assertEqual(fm.get("milestone"), "v0.4.0 - OSHE Inspect Private Alpha")
        self.assertEqual(fm.get("lifecycle_status"), "DRAFT")
        self.assertEqual(fm.get("status"), "APPROVED_FOR_LOCAL_DEVELOPMENT")

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

    def test_module_boundaries_and_advisory_decoupling(self):
        """Validates that notification delivery is advisory and decoupled from business state."""
        self.assertIn("MOD-WFA", self.content)
        self.assertIn("MOD-EVT", self.content)
        self.assertIn("Advisory Decoupling Invariant", self.content)
        self.assertIn("Zero Direct Business State Mutation", self.content)

    def test_append_only_responsibility_history(self):
        """Validates append-only responsibility tracking across assignments, downloads, reassignments, and cancellations."""
        self.assertIn("Append-Only Responsibility History", self.content)
        self.assertIn("Initial Assignment", self.content)
        self.assertIn("Field Download", self.content)
        self.assertIn("Supervisory Reassignment", self.content)
        self.assertIn("Administrative Cancellation", self.content)
        self.assertIn("never erased or overwritten", self.content)

    def test_schedule_operational_diagnostics_taxonomy(self):
        """Validates diagnostic event codes and visible failure reporting."""
        codes = [
            "DIAG_CLOCK_SKEW",
            "DIAG_DUPLICATE_DISPATCH",
            "DIAG_NOTIFICATION_FAILED",
            "DIAG_NOTIFICATION_QUARANTINED",
            "DIAG_SCHEDULE_ALTERED",
            "DIAG_SCHEDULE_CANCELLED",
        ]
        for code in codes:
            self.assertIn(code, self.content, f"Missing diagnostic code {code}")
        self.assertIn("Visible Failure Surface", self.content)

    def test_notification_request_lifecycle_states(self):
        """Validates notification request states and supervisory replay."""
        states = ["StatusPending", "StatusDelivered", "StatusFailed", "StatusQuarantined"]
        for st in states:
            self.assertIn(st, self.content, f"Missing notification status {st}")
        self.assertIn("Controlled Replay Protocol", self.content)
        self.assertIn("ReplayNotification", self.content)

    def test_delayed_and_duplicate_reminders_handling(self):
        """Validates reminder idempotency keys and stale pre-due reminder degradation."""
        self.assertIn("reminder_key", self.content)
        self.assertIn("ErrDuplicateNotificationRequest", self.content)
        self.assertIn("Graceful Degradation for Delayed Reminders", self.content)
        self.assertIn("DIAG_STALE_REMINDER_SUPPRESSED", self.content)

    def test_clock_skew_and_deterministic_fixtures(self):
        """Validates deterministic clock abstraction and clock skew defense."""
        self.assertIn("Clock Interface", self.content)
        self.assertIn("Clock Skew Defense", self.content)
        self.assertIn("ErrClockSkewDetected", self.content)

    def test_local_sink_exclusivity_and_h040_010_hold(self):
        """Validates that only local notification sinks are permitted and external channels are blocked."""
        self.assertIn("ChannelLocalSink", self.content)
        self.assertIn("ChannelInternalLog", self.content)
        self.assertIn("ChannelAuditJournal", self.content)
        self.assertIn("ErrExternalChannelProhibited", self.content)
        self.assertIn("H040-010", self.content)

    def test_synthetic_operations_fixtures(self):
        """Validates synthetic operations fixture YAML structure and scenarios."""
        self.assertIsNotNone(self.fixture, "Synthetic operations fixture YAML block not found")
        fix = self.fixture

        self.assertEqual(fix.get("fixture_id"), "fix_syn_scheduling_operations_v1")
        scenarios = fix.get("scenarios", [])
        self.assertEqual(len(scenarios), 5, "Fixture must define exactly 5 operational scenarios")

        # Scenario 1: Clean Local Sink Delivery
        s1 = scenarios[0]
        self.assertEqual(s1.get("scenario_id"), "scen_syn_clean_delivery_01")
        self.assertEqual(s1.get("status"), "DELIVERED")
        self.assertEqual(s1.get("business_state_impact"), "NONE_ADVISORY_ONLY")

        # Scenario 2: Sink Failure & Quarantine
        s2 = scenarios[1]
        self.assertEqual(s2.get("scenario_id"), "scen_syn_sink_failure_quarantine_02")
        self.assertEqual(s2.get("status"), "QUARANTINED")
        self.assertEqual(s2.get("attempts"), 3)

        # Scenario 3: Supervisory Replay
        s3 = scenarios[2]
        self.assertEqual(s3.get("scenario_id"), "scen_syn_supervisory_replay_03")
        self.assertEqual(s3.get("status_after_replay"), "DELIVERED")

        # Scenario 4: Clock Skew Detection
        s4 = scenarios[3]
        self.assertEqual(s4.get("scenario_id"), "scen_syn_clock_skew_04")
        self.assertEqual(s4.get("evaluation_result"), "REJECTED_FAIL_CLOSED")
        self.assertEqual(s4.get("diagnostic_code"), "DIAG_CLOCK_SKEW")

        # Scenario 5: Delayed Reminder Degradation
        s5 = scenarios[4]
        self.assertEqual(s5.get("scenario_id"), "scen_syn_delayed_reminder_05")
        self.assertEqual(s5.get("diagnostic_code"), "DIAG_STALE_REMINDER_SUPPRESSED")

if __name__ == "__main__":
    unittest.main()
