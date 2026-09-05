from __future__ import annotations

import pathlib
import re
import unittest
from typing import Any, Dict, List, Optional

ROOT = pathlib.Path(__file__).resolve().parents[1]
DECISION_PACKET_PATH = ROOT / "docs" / "architecture" / "v030-human-release-decision-packet.md"
ARCH_README_PATH = ROOT / "docs" / "architecture" / "README.md"
RELEASE_README_PATH = ROOT / "tests" / "release" / "README.md"

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


class V030ReleaseDecisionPacketTests(unittest.TestCase):
    def setUp(self) -> None:
        self.assertTrue(DECISION_PACKET_PATH.is_file(), f"Missing decision packet at {DECISION_PACKET_PATH}")
        self.content = DECISION_PACKET_PATH.read_text(encoding="utf-8")
        self.frontmatter = parse_simple_frontmatter(self.content)

    def test_decision_packet_structure_and_frontmatter(self) -> None:
        """Asserts required governed frontmatter fields and operational boundaries."""
        self.assertEqual(self.frontmatter.get("document_id"), "ARC-V030-DECPKT-001")
        self.assertEqual(self.frontmatter.get("document_type"), "architecture_decision_packet")
        self.assertEqual(self.frontmatter.get("lifecycle_status"), "DRAFT")
        self.assertEqual(self.frontmatter.get("status"), "HELD_PENDING_SOLE_HUMAN_OWNER_DECISION_H030_008")
        self.assertEqual(self.frontmatter.get("governing_issue"), "GitHub Issue #111")
        self.assertEqual(self.frontmatter.get("governing_decision"), "HDEC-V030-ENTRY-AND-POLICY-052")
        self.assertEqual(self.frontmatter.get("milestone"), "v0.3.0 - Organization Identity and Portal Alpha")

        deferred = self.frontmatter.get("deferred_human_gates", [])
        self.assertIn("H030-007", deferred, "H030-007 must remain in deferred_human_gates")
        self.assertIn("H030-008", deferred, "H030-008 must remain in deferred_human_gates")

        credit = self.frontmatter.get("credit_boundary", "")
        self.assertIn("NO_RELEASE_OR_TAGGING_CREDIT", credit)

    def test_four_formal_decision_options(self) -> None:
        """Asserts all four structured options are documented with impacts, prerequisites, and risks."""
        self.assertIn("Option 1: Full Approval & Milestone v0.3.0 Closure", self.content)
        self.assertIn("Option 2: Conditional Approval with Deferred Ingress", self.content)
        self.assertIn("Option 3: Extended Hold Pending Staging / Persistence Spike", self.content)
        self.assertIn("Option 4: Formal Rejection / Scope Remand", self.content)

        # Confirm analysis criteria
        self.assertIn("Prerequisites:", self.content)
        self.assertIn("System & Operational Impacts:", self.content)
        self.assertIn("Risks:", self.content)
        self.assertIn("Comparative Analysis & Tradeoff Matrix", self.content)

    def test_unfilled_sole_human_owner_decision_record(self) -> None:
        """Asserts the decision record template is strictly UNFILLED and pending human execution."""
        self.assertIn("HDEC-V030-RELEASE-053", self.content)
        self.assertIn("selected_option: UNFILLED", self.content)
        self.assertIn("decided_by: UNFILLED", self.content)
        self.assertIn("decided_at: UNFILLED", self.content)
        self.assertIn("signature_or_auth_ref: UNFILLED", self.content)
        self.assertIn("PENDING_HUMAN_EXECUTION", self.content)

        # Non-claims assertion: Zero automated filling
        self.assertNotIn("selected_option: OPTION_1", self.content)
        self.assertNotIn("selected_option: OPTION_2", self.content)
        self.assertNotIn("selected_option: OPTION_3", self.content)
        self.assertNotIn("selected_option: OPTION_4", self.content)

    def test_deferred_human_gate_and_non_claims_invariants(self) -> None:
        """Verifies explicit hold status of H030-008 and operational non-claims."""
        self.assertIn("H030-008", self.content)
        self.assertIn("HOLD", self.content)
        self.assertIn("Zero Release or Tagging Action", self.content)
        self.assertIn("HDEC-V030-ENTRY-AND-POLICY-052", self.content)

    def test_evidence_basis_and_prerequisite_checklist(self) -> None:
        """Asserts evidence reference and prerequisite gating checklist."""
        self.assertIn("ARC-V030-RELEVD-001", self.content)
        self.assertIn("Prerequisite Conditions Checklist", self.content)
        self.assertIn("Residual Risk and Technical Debt Assessment", self.content)
        self.assertIn("In-Memory Data Models", self.content)
        self.assertIn("Mock Clock Dependency", self.content)
        self.assertIn("Public Route Packet Hold", self.content)

    def test_readme_navigation_cross_references(self) -> None:
        """Asserts architecture and release README files cross-reference the decision packet."""
        self.assertTrue(ARCH_README_PATH.is_file())
        arch_readme_text = ARCH_README_PATH.read_text(encoding="utf-8")
        self.assertIn("v030-human-release-decision-packet.md", arch_readme_text)
        self.assertIn("ARC-V030-DECPKT-001", arch_readme_text)

        self.assertTrue(RELEASE_README_PATH.is_file())
        release_readme_text = RELEASE_README_PATH.read_text(encoding="utf-8")
        self.assertIn("test_v030_release_decision_packet.py", release_readme_text)
        self.assertIn("ARC-V030-DECPKT-001", release_readme_text)


if __name__ == "__main__":
    unittest.main()
