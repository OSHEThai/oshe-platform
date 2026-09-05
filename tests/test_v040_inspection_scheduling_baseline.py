import os
import re
import unittest
import yaml

class TestV040InspectionSchedulingBaseline(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.doc_path = os.path.join(
            os.path.dirname(os.path.dirname(os.path.abspath(__file__))),
            "docs",
            "architecture",
            "v040-inspection-scheduling-baseline.md"
        )
        if not os.path.exists(cls.doc_path):
            raise FileNotFoundError(f"Inspection scheduling baseline not found at {cls.doc_path}")

        with open(cls.doc_path, "r", encoding="utf-8") as f:
            cls.content = f.read()

        # Extract frontmatter
        fm_match = re.match(r"^---\s*\n(.*?)\n---\s*\n", cls.content, re.DOTALL)
        if not fm_match:
            raise ValueError("Frontmatter not found in inspection scheduling baseline")
        cls.frontmatter = yaml.safe_load(fm_match.group(1))

        # Extract synthetic multi-schedule fixture YAML
        yaml_blocks = re.findall(r"```yaml\s*\n(.*?)\n```", cls.content, re.DOTALL)
        cls.fixture = None
        for block in yaml_blocks:
            try:
                parsed = yaml.safe_load(block)
                if isinstance(parsed, dict) and "fixture_id" in parsed and "schedules" in parsed:
                    cls.fixture = parsed
                    break
            except Exception:
                continue

    def test_document_metadata_and_frontmatter(self):
        """Validates document metadata, identifiers, and governance gates in YAML frontmatter."""
        fm = self.frontmatter
        self.assertEqual(fm.get("document_id"), "ARC-V040-SCHED-001")
        self.assertEqual(fm.get("governing_issue"), "GitHub Issue #120")
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

    def test_schedule_lifecycle_states_and_transitions(self):
        """Validates that all five schedule states are defined."""
        expected_states = ["ACTIVE", "PAUSED", "COMPLETED", "CANCELLED", "RETIRED"]
        for state in expected_states:
            self.assertIn(state, self.content, f"Missing schedule state {state}")

    def test_schedule_triggers_enumeration(self):
        """Validates that all four schedule trigger types are defined."""
        triggers = ["CALENDAR_RECURRING", "MANUAL_ADHOC", "EVENT_TRIGGERED", "REINSPECTION_DEFECT"]
        for trg in triggers:
            self.assertIn(trg, self.content, f"Missing schedule trigger {trg}")

    def test_scope_and_checklist_version_binding(self):
        """Validates scope hierarchy and checklist immutable version binding."""
        self.assertIn("tenant_id", self.content)
        self.assertIn("company_id", self.content)
        self.assertIn("site_id", self.content)
        self.assertIn("area_id", self.content)
        self.assertIn("PUBLISHED_IMMUTABLE", self.content)
        self.assertIn("content_digest", self.content)
        self.assertIn("Strict Execution Pinning", self.content)

    def test_recurrence_rules_patterns(self):
        """Validates recurrence patterns: ONCE, DAILY, WEEKLY, MONTHLY."""
        patterns = ["ONCE", "DAILY", "WEEKLY", "MONTHLY"]
        for pat in patterns:
            self.assertIn(pat, self.content, f"Missing recurrence pattern {pat}")
        self.assertIn("Deterministic Next-Occurrence Algorithm", self.content)

    def test_due_windows_and_overdue_progression(self):
        """Validates due window anatomy and compliance status progression."""
        self.assertIn("window_start", self.content)
        self.assertIn("due_date", self.content)
        self.assertIn("grace_period_hours", self.content)
        self.assertIn("grace_until", self.content)
        self.assertIn("ON_TIME", self.content)
        self.assertIn("IN_GRACE_PERIOD", self.content)
        self.assertIn("OVERDUE", self.content)

    def test_canonical_timezone_handling(self):
        """Validates canonical Asia/Bangkok time-zone handling and UTC storage."""
        self.assertIn("Asia/Bangkok", self.content)
        self.assertIn("UTC+07:00", self.content)
        self.assertIn("UTC ISO 8601", self.content)
        self.assertIn("Zero Daylight Saving Time", self.content)

    def test_idempotency_and_deduplication_keys(self):
        """Validates deterministic SHA-256 idempotency key composition."""
        self.assertIn("idempotency_key", self.content)
        self.assertIn("SHA-256", self.content)
        self.assertIn("scheduled_date", self.content)
        self.assertIn("Dispatch Deduplication Invariant", self.content)

    def test_safe_cancellation_pause_and_retirement(self):
        """Validates pause, resume, cancellation, and retirement invariants."""
        self.assertIn("Pausing a Schedule", self.content)
        self.assertIn("Cancelling a Schedule", self.content)
        self.assertIn("Retiring a Schedule", self.content)
        self.assertIn("retirement_reason", self.content)

    def test_synthetic_notification_boundary(self):
        """Validates zero external notification claim and local synthetic notifications."""
        self.assertIn("Zero External Notification Claim", self.content)
        self.assertIn("UNSUPPORTED_PRIVATE_ALPHA", self.content)
        self.assertIn("notification_record", self.content)

    def test_synthetic_multi_schedule_fixtures(self):
        """Validates synthetic multi-schedule fixture YAML structure."""
        self.assertIsNotNone(self.fixture, "Synthetic multi-schedule fixture YAML block not found")
        fix = self.fixture

        self.assertEqual(fix.get("fixture_id"), "fix_syn_inspection_schedules_v1")
        schedules = fix.get("schedules", [])
        self.assertEqual(len(schedules), 3, "Fixture must define exactly 3 schedules")

        # Schedule 1: Weekly Recurrent Site Safety Walk
        s1 = schedules[0]
        self.assertEqual(s1.get("schedule_id"), "sch_syn_weekly_plant_01")
        self.assertEqual(s1.get("trigger_type"), "CALENDAR_RECURRING")
        self.assertEqual(s1.get("status"), "ACTIVE")
        self.assertEqual(s1.get("time_zone"), "Asia/Bangkok")
        self.assertEqual(s1.get("recurrence", {}).get("frequency"), "WEEKLY")
        self.assertEqual(s1.get("recurrence", {}).get("days_of_week"), ["MON"])
        self.assertEqual(s1.get("checklist_binding", {}).get("published_version"), "1.1.0")

        # Schedule 2: Monthly Confined Space
        s2 = schedules[1]
        self.assertEqual(s2.get("schedule_id"), "sch_syn_monthly_confined_01")
        self.assertEqual(s2.get("trigger_type"), "CALENDAR_RECURRING")
        self.assertEqual(s2.get("status"), "ACTIVE")
        self.assertEqual(s2.get("recurrence", {}).get("frequency"), "MONTHLY")
        self.assertEqual(s2.get("recurrence", {}).get("day_of_month"), 15)

        # Schedule 3: Manual Ad-Hoc
        s3 = schedules[2]
        self.assertEqual(s3.get("schedule_id"), "sch_syn_adhoc_maint_01")
        self.assertEqual(s3.get("trigger_type"), "MANUAL_ADHOC")
        self.assertEqual(s3.get("status"), "COMPLETED")
        self.assertEqual(s3.get("recurrence", {}).get("frequency"), "ONCE")

if __name__ == "__main__":
    unittest.main()
