from __future__ import annotations

import pathlib
import re
import unittest
from typing import Any, Dict, List, Optional

ROOT = pathlib.Path(__file__).resolve().parents[1]
QUALIFICATION_BASELINE_PATH = ROOT / "docs" / "qualification" / "v040-scheduling-qualification-baseline.md"

FRONTMATTER_PATTERN = re.compile(r"^---\s*\n(.*?)\n---\s*\n", re.DOTALL)


def parse_simple_frontmatter(content: str) -> Dict[str, Any]:
    match = FRONTMATTER_PATTERN.match(content)
    if not match:
        raise ValueError("Missing YAML frontmatter")
    lines = match.group(1).splitlines()
    data: Dict[str, Any] = {}
    current_key: Optional[str] = None
    for line in lines:
        if not line.strip() or line.strip().startswith("#"):
            continue
        stripped = line.strip()
        if stripped.startswith("- "):
            if current_key and isinstance(data.get(current_key), list):
                data[current_key].append(stripped[2:].strip().strip("\"'"))
            continue
        if ":" in line:
            key, val = line.split(":", 1)
            key = key.strip()
            val = val.strip().strip("\"'")
            if val == "":
                data[key] = []
                current_key = key
            else:
                data[key] = val
                current_key = None
    return data


class V040SchedulingQualificationBaselineTests(unittest.TestCase):
    def setUp(self) -> None:
        self.assertTrue(
            QUALIFICATION_BASELINE_PATH.is_file(),
            f"Missing scheduling qualification baseline at {QUALIFICATION_BASELINE_PATH}",
        )
        self.content = QUALIFICATION_BASELINE_PATH.read_text(encoding="utf-8")
        self.frontmatter = parse_simple_frontmatter(self.content)

    def test_qualification_document_metadata_and_frontmatter(self) -> None:
        """Asserts required governed frontmatter fields, identifiers, and authority linkages."""
        self.assertEqual(self.frontmatter.get("document_id"), "QLF-V040-SCHED-001")
        self.assertEqual(self.frontmatter.get("document_type"), "qualification_specification")
        self.assertEqual(self.frontmatter.get("lifecycle_status"), "DRAFT")
        self.assertEqual(self.frontmatter.get("status"), "APPROVED_FOR_LOCAL_DEVELOPMENT")
        self.assertEqual(self.frontmatter.get("governing_issue"), "GitHub Issue #123")
        self.assertEqual(self.frontmatter.get("authority_source"), "HDEC-V040-FOUNDATION-054")
        self.assertEqual(
            self.frontmatter.get("credit_boundary"),
            "TECHNICAL_QUALIFICATION_ONLY_NO_USER_EVIDENCE_OR_RELEASE_CREDIT",
        )

        approved_gates = self.frontmatter.get("approved_foundation_gates", [])
        for gate in ["H040-001", "H040-002", "H040-003", "H040-004", "H040-005", "H040-006"]:
            self.assertIn(gate, approved_gates, f"Missing approved gate {gate}")

        retained_holds = self.frontmatter.get("retained_holds", [])
        for hold in ["H040-007", "H040-008", "H040-009", "H040-010", "H040-011"]:
            self.assertIn(hold, retained_holds, f"Missing retained hold {hold}")

    def test_qualification_test_domain_coverage(self) -> None:
        """Asserts technical testing covers all required test domains from QSCHED-01 to QSCHED-10."""
        domains = [
            "QSCHED-01",
            "QSCHED-02",
            "QSCHED-03",
            "QSCHED-04",
            "QSCHED-05",
            "QSCHED-06",
            "QSCHED-07",
            "QSCHED-08",
            "QSCHED-09",
            "QSCHED-10",
        ]
        for domain in domains:
            self.assertIn(domain, self.content, f"Missing domain {domain}")

        # Core operational topics
        self.assertIn("scheduling", self.content.lower())
        self.assertIn("assignment", self.content.lower())
        self.assertIn("recurrence", self.content.lower())
        self.assertIn("time zone", self.content.lower())
        self.assertIn("reassignment", self.content.lower())
        self.assertIn("cancellation", self.content.lower())
        self.assertIn("notification failure", self.content.lower())

    def test_non_substitution_invariant_and_user_evidence_boundary(self) -> None:
        """Asserts explicit boundary that technical qualification does not substitute for real user evidence."""
        self.assertIn("Non-Substitution Invariant", self.content)
        self.assertIn(
            "Under no circumstances may simulated agent runs, automated scheduler test scripts, or synthetic worker payloads substitute for",
            self.content,
        )
        self.assertIn("Gate `H040-008`", self.content)
        self.assertIn("HOLD", self.content)

    def test_recurrence_time_zone_and_due_windows(self) -> None:
        """Asserts canonical Asia/Bangkok time zone, recurrence frequencies, and due-window compliance progression."""
        # Time Zone
        self.assertIn("Asia/Bangkok", self.content)
        self.assertIn("UTC+07:00", self.content)
        self.assertIn("Zero Daylight Saving Time (DST) Ambiguity", self.content)

        # Frequencies
        for freq in ["ONCE", "DAILY", "WEEKLY", "MONTHLY"]:
            self.assertIn(freq, self.content, f"Missing frequency {freq}")

        # Due Window & Progression
        self.assertIn("window_start", self.content)
        self.assertIn("due_date", self.content)
        self.assertIn("grace_until", self.content)
        self.assertIn("ON_TIME", self.content)
        self.assertIn("IN_GRACE_PERIOD", self.content)
        self.assertIn("OVERDUE", self.content)

    def test_idempotency_custody_and_advisory_decoupling(self) -> None:
        """Asserts cryptographic idempotency deduplication, downloaded work custody, and advisory notification decoupling."""
        # Idempotency
        self.assertIn("idempotency_key", self.content)
        self.assertIn("SHA-256", self.content)
        self.assertIn("Zero Duplicate Dispatch", self.content)

        # Custody & Reassignment
        self.assertIn("Prohibition Against Erasing Prior Responsibility", self.content)
        self.assertIn("override_downloaded_work", self.content)
        self.assertIn("QUARANTINED_CONFLICT", self.content)

        # Advisory Decoupling
        self.assertIn("Advisory Decoupling Invariant", self.content)
        self.assertIn("never alters or mutates", self.content)
        self.assertIn("MOD-WFA", self.content)
        self.assertIn("MOD-EVT", self.content)

    def test_diagnostic_events_and_negative_controls(self) -> None:
        """Asserts diagnostic events and negative controls from NSCHED-01 through NSCHED-08."""
        # Diagnostic Events
        for diag in [
            "DIAG_CLOCK_SKEW",
            "DIAG_DUPLICATE_DISPATCH",
            "DIAG_NOTIFICATION_FAILED",
            "DIAG_NOTIFICATION_QUARANTINED",
            "DIAG_SCHEDULE_ALTERED",
            "DIAG_SCHEDULE_CANCELLED",
        ]:
            self.assertIn(diag, self.content, f"Missing diagnostic event {diag}")

        # Negative Controls
        for neg in [
            "NSCHED-01",
            "NSCHED-02",
            "NSCHED-03",
            "NSCHED-04",
            "NSCHED-05",
            "NSCHED-06",
            "NSCHED-07",
            "NSCHED-08",
        ]:
            self.assertIn(neg, self.content, f"Missing negative control {neg}")

        # Explicit Error Codes
        for err in [
            "ErrExpiredTenantMembership",
            "ErrUnauthorizedAssignmentScope",
            "ErrRevokedInspectorRole",
            "ErrUnsupportedDevicePlatform",
            "ErrDuplicateInspectionAssignment",
            "ErrDownloadedWorkConflict",
            "ErrClockSkewDetected",
            "ErrInvalidChecklistVersionState",
        ]:
            self.assertIn(err, self.content, f"Missing error code {err}")

    def test_retained_holds_and_anti_scope_prohibitions(self) -> None:
        """Verifies strict retention of H040-007 through H040-011 holds and explicit anti-scope prohibitions."""
        for hold_id in ["H040-007", "H040-008", "H040-009", "H040-010", "H040-011"]:
            self.assertIn(hold_id, self.content)

        self.assertIn("Zero External Public Routes", self.content)
        self.assertIn("Zero CDN Edge Deployment", self.content)
        self.assertIn("Zero Production Database Deployment", self.content)
        self.assertIn("Zero Real Customer or Personal Data", self.content)
        self.assertIn("Zero Provider, Credential, or Account Mutations", self.content)
        self.assertIn("Zero External Notification Delivery", self.content)


if __name__ == "__main__":
    unittest.main()
