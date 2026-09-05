from __future__ import annotations

import pathlib
import re
import unittest
from typing import Any, Dict, List, Optional

ROOT = pathlib.Path(__file__).resolve().parents[1]
DECISION_PACKET_PATH = ROOT / "docs" / "qualification" / "v040-human-outcome-decision-packet.md"

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


class V040HumanOutcomeDecisionPacketTests(unittest.TestCase):
    def setUp(self) -> None:
        self.assertTrue(DECISION_PACKET_PATH.is_file(), f"Missing decision packet at {DECISION_PACKET_PATH}")
        self.content = DECISION_PACKET_PATH.read_text(encoding="utf-8")
        self.frontmatter = parse_simple_frontmatter(self.content)

    def test_document_structure_and_frontmatter(self) -> None:
        """Asserts required governed frontmatter fields and operational boundaries."""
        self.assertEqual(self.frontmatter.get("document_id"), "DOC-V040-OUTCOME-DECPKT-001")
        self.assertEqual(self.frontmatter.get("document_type"), "qualification_decision_packet")
        self.assertEqual(self.frontmatter.get("lifecycle_status"), "DRAFT")
        self.assertEqual(self.frontmatter.get("status"), "HELD_PENDING_SOLE_HUMAN_OWNER_DECISION_H040_011")
        self.assertEqual(self.frontmatter.get("governing_issue"), "GitHub Issue #151")
        self.assertEqual(self.frontmatter.get("governing_decision"), "HDEC-V040-FOUNDATION-054")
        self.assertEqual(self.frontmatter.get("governing_gate"), "H040-011")
        self.assertEqual(self.frontmatter.get("milestone"), "v0.4.0 - OSHE Inspect Private Alpha")

        retained = self.frontmatter.get("retained_human_gates", [])
        for expected_gate in ["H040-007", "H040-008", "H040-009", "H040-010", "H040-011"]:
            self.assertIn(expected_gate, retained, f"{expected_gate} must remain in retained_human_gates")

        credit = self.frontmatter.get("credit_boundary", "")
        self.assertIn("NO_OUTCOME_SELECTED", credit)

    def test_five_neutral_decision_options_structure(self) -> None:
        """Asserts all five neutral options are documented with conditions, evidence, limitations, risks, and v0.5 impacts."""
        self.assertIn("Option 1: Continue", self.content)
        self.assertIn("Option 2: Pivot", self.content)
        self.assertIn("Option 3: Extend", self.content)
        self.assertIn("Option 4: Hold", self.content)
        self.assertIn("Option 5: Stop", self.content)

        # Confirm required subfield coverage across options
        self.assertIn("Required Conditions:", self.content)
        self.assertIn("Evidence Basis:", self.content)
        self.assertIn("Known Limitations:", self.content)
        self.assertIn("Residual Risks:", self.content)
        self.assertIn("v0.5 Dependency Impacts:", self.content)
        self.assertIn("Comparative Analysis & Tradeoff Matrix", self.content)

    def test_strictly_unfilled_decision_record(self) -> None:
        """Asserts the decision record template is strictly UNFILLED and no outcome is selected."""
        self.assertIn("HDEC-V040-OUTCOME-056", self.content)
        self.assertIn("selected_outcome: UNFILLED", self.content)
        self.assertIn("rationale: UNFILLED", self.content)
        self.assertIn("authorized_v05_planning: UNFILLED", self.content)
        self.assertIn("authorized_pilot_readiness: UNFILLED", self.content)
        self.assertIn("authorized_release_tag: UNFILLED", self.content)
        self.assertIn("decided_by: UNFILLED", self.content)
        self.assertIn("decided_at: UNFILLED", self.content)
        self.assertIn("signature_or_auth_ref: UNFILLED", self.content)
        self.assertIn("PENDING_SOLE_HUMAN_OWNER_EXECUTION", self.content)

        # Verify no outcome has been pre-selected
        self.assertNotIn("selected_outcome: CONTINUE", self.content)
        self.assertNotIn("selected_outcome: PIVOT", self.content)
        self.assertNotIn("selected_outcome: EXTEND", self.content)
        self.assertNotIn("selected_outcome: HOLD", self.content)
        self.assertNotIn("selected_outcome: STOP", self.content)

    def test_non_claims_and_hold_invariants(self) -> None:
        """Verifies explicit hold status of H040-011 and exclusion of pilot/production/customer data/GA claims."""
        self.assertIn("H040-011", self.content)
        self.assertIn("HOLD_PENDING_SOLE_HUMAN_OWNER", self.content)
        self.assertIn("HDEC-V040-FOUNDATION-054", self.content)

        # Prohibitions and exclusions
        self.assertIn("Private-alpha pilot onboarding, real user recruitment, or participant testing", self.content)
        self.assertIn("Production deployment, cloud hosting, or external environment routing", self.content)
        self.assertIn("Customer data ingestion, PII exposure, or live telemetry collection", self.content)
        self.assertIn("Commercial general availability (GA), software release tagging, or cryptographic signing", self.content)
        self.assertIn("Residual risk acceptance or Milestone v0.5.0 entry authorization", self.content)

    def test_residual_risk_ledger_unaccepted(self) -> None:
        """Asserts residual risks and limitations are documented as unaccepted."""
        self.assertIn("Residual Risk & Technical Debt Ledger (Unaccepted)", self.content)
        self.assertIn("none are accepted by this document", self.content)
        self.assertIn("Synthetic-Data Boundary", self.content)
        self.assertIn("Single-Tenant Architecture", self.content)


if __name__ == "__main__":
    unittest.main()
