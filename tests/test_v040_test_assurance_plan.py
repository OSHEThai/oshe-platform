from __future__ import annotations

import pathlib
import re
import unittest
from typing import Any, Dict, List, Optional

ROOT = pathlib.Path(__file__).resolve().parents[1]
ASSURANCE_PLAN_PATH = ROOT / "docs" / "qualification" / "v040-test-assurance-evidence-plan.md"

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


class V040TestAssurancePlanTests(unittest.TestCase):
    def setUp(self) -> None:
        self.assertTrue(ASSURANCE_PLAN_PATH.is_file(), f"Missing assurance plan at {ASSURANCE_PLAN_PATH}")
        self.content = ASSURANCE_PLAN_PATH.read_text(encoding="utf-8")
        self.frontmatter = parse_simple_frontmatter(self.content)

    def test_plan_document_exists_and_frontmatter_valid(self) -> None:
        """Asserts required governed frontmatter fields, identity, and authority linkages."""
        self.assertEqual(self.frontmatter.get("document_id"), "DOC-PLAN-V040-001")
        self.assertEqual(self.frontmatter.get("document_type"), "quality_assurance_plan")
        self.assertEqual(self.frontmatter.get("lifecycle_status"), "DRAFT")
        self.assertEqual(self.frontmatter.get("governing_issue"), "GitHub Issue #115")
        self.assertEqual(self.frontmatter.get("authority_source"), "HDEC-V040-FOUNDATION-054")
        self.assertEqual(self.frontmatter.get("credit_boundary"), "PLANNING_ONLY_NO_EXECUTION_OR_RELEASE_CREDIT")

        approved_gates = self.frontmatter.get("governing_gates_approved", [])
        for gate in ["H040-001", "H040-002", "H040-003", "H040-004", "H040-005", "H040-006"]:
            self.assertIn(gate, approved_gates, f"Missing approved gate {gate}")

        retained_holds = self.frontmatter.get("retained_holds", [])
        for hold in ["H040-007", "H040-008", "H040-009", "H040-010", "H040-011"]:
            self.assertIn(hold, retained_holds, f"Missing retained hold {hold}")

    def test_technical_test_framework_coverage(self) -> None:
        """Asserts technical testing covers four roles, responsive platforms, server authority, and pilot checklist."""
        # Roles
        self.assertIn("Checklist Author", self.content)
        self.assertIn("Field Inspector", self.content)
        self.assertIn("CAPA Owner", self.content)
        self.assertIn("Independent Reviewer", self.content)

        # Platforms & localization
        self.assertIn("Chrome/Edge", self.content)
        self.assertIn("Android Chrome", self.content)
        self.assertIn("en-US", self.content)
        self.assertIn("th-TH", self.content)
        self.assertIn("Asia/Bangkok", self.content)

        # Concurrency & authority
        self.assertIn("Server Authority", self.content)
        self.assertIn("No Last-Write-Wins", self.content)
        self.assertIn("Quarantine on Conflict", self.content)

        # Pilot checklist
        self.assertIn("synthetic non-regulatory pilot checklist", self.content)
        self.assertIn("Unknown", self.content)
        self.assertIn("Not Applicable", self.content)

    def test_synthetic_vs_real_user_evidence_separation(self) -> None:
        """Asserts explicit separation of synthetic technical evidence from real-user empirical evidence."""
        self.assertIn("Strict Separation of Synthetic and Real-User Evidence", self.content)
        self.assertIn("Non-Substitution Invariant", self.content)
        self.assertIn(
            "Under no circumstances may simulated agent runs, automated scripts, or mock data substitute for",
            self.content,
        )
        self.assertIn("Real-User UAT & Private-Alpha Protocol (Gated under `H040-008` HOLD)", self.content)

    def test_retained_holds_and_prohibitions(self) -> None:
        """Verifies explicit hold status of H040-008 through H040-011 and prohibitions."""
        for hold_id in ["H040-008", "H040-009", "H040-010", "H040-011"]:
            self.assertIn(hold_id, self.content)

        self.assertIn("HOLD", self.content)
        self.assertIn("Zero External Public Routes", self.content)
        self.assertIn("Zero CDN Edge Deployment", self.content)
        self.assertIn("Zero Production Database Deployment", self.content)
        self.assertIn("Zero Real Customer or Personal Data", self.content)
        self.assertIn("Zero Provider, Credential, or Account Mutations", self.content)

    def test_support_recovery_and_defect_policy(self) -> None:
        """Asserts operational recovery, conflict quarantine resolution, and P1-P4 defect severity taxonomy."""
        self.assertIn("Support, Failure Modes, and Operational Recovery", self.content)
        self.assertIn("Manual Fallback Pathways", self.content)
        self.assertIn("Quarantine Resolution", self.content)
        self.assertIn("Disaster Recovery & Lineage Rebuild", self.content)

        # Defect policy
        self.assertIn("Defect Severity Policy & Triage Rules", self.content)
        self.assertIn("P1 - Critical / Blocker", self.content)
        self.assertIn("Zero P1 defects permitted", self.content)
        self.assertIn("P2 - High", self.content)
        self.assertIn("P3 - Medium", self.content)
        self.assertIn("P4 - Low", self.content)

    def test_evidence_mapping_scorecard(self) -> None:
        """Asserts presence of comprehensive evidence mapping scorecard linking gates to tests and blockers."""
        self.assertIn("Comprehensive Evidence Mapping & Assurance Scorecard", self.content)
        self.assertIn("EVD-01", self.content)
        self.assertIn("EVD-02", self.content)
        self.assertIn("EVD-03", self.content)
        self.assertIn("EVD-04", self.content)
        self.assertIn("EVD-05", self.content)
        self.assertIn("EVD-06", self.content)
        self.assertIn("EVD-07", self.content)
        self.assertIn("EVD-08", self.content)
        self.assertIn("EVD-09", self.content)
        self.assertIn("Release Blocker?", self.content)


if __name__ == "__main__":
    unittest.main()
