from __future__ import annotations

import pathlib
import re
import unittest
from typing import Any, Dict, List, Optional

ROOT = pathlib.Path(__file__).resolve().parents[1]
DECISION_RECORD_PATH = ROOT / "docs" / "architecture" / "v030-release-decision-record.md"
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


class V030ReleaseDecisionRecordTests(unittest.TestCase):
    def setUp(self) -> None:
        self.assertTrue(DECISION_RECORD_PATH.is_file(), f"Missing decision record at {DECISION_RECORD_PATH}")
        self.content = DECISION_RECORD_PATH.read_text(encoding="utf-8")
        self.frontmatter = parse_simple_frontmatter(self.content)

    def test_decision_record_frontmatter_and_identity(self) -> None:
        """Asserts required governed frontmatter fields and authority identity."""
        self.assertEqual(self.frontmatter.get("document_id"), "ARC-V030-DECREC-001")
        self.assertEqual(self.frontmatter.get("document_type"), "architecture_decision_record")
        self.assertEqual(self.frontmatter.get("lifecycle_status"), "APPROVED")
        self.assertEqual(self.frontmatter.get("authority_source"), "HDEC-V030-RELEASE-053")
        self.assertEqual(self.frontmatter.get("governing_issue"), "GitHub Issue #111")
        self.assertEqual(self.frontmatter.get("governing_gate"), "H030-008")
        self.assertEqual(self.frontmatter.get("decided_by"), "Sole Human Owner")
        self.assertEqual(self.frontmatter.get("selected_option"), "OPTION_1_FULL_APPROVAL_AND_MILESTONE_CLOSURE")

        retained = self.frontmatter.get("retained_holds", [])
        self.assertIn("H030-007", retained, "H030-007 must remain in retained_holds")

        credit = self.frontmatter.get("credit_boundary", "")
        self.assertIn("NO_RELEASE_OR_TAGGING_CREDIT", credit)

    def test_approved_authorizations_documented(self) -> None:
        """Asserts all Sole Human Owner Option 1 authorizations are explicitly documented."""
        self.assertIn("Milestone v0.3.0 Closure", self.content)
        self.assertIn("Annotated Git Tag Creation", self.content)
        self.assertIn("v0.3.0", self.content)
        self.assertIn("GitHub Release Publication", self.content)
        self.assertIn("Milestone v0.4.0 Planning", self.content)
        self.assertIn("HDEC-V030-RELEASE-053", self.content)
        self.assertIn("OPTION_1_FULL_APPROVAL_AND_MILESTONE_CLOSURE", self.content)

    def test_retained_holds_and_prohibitions(self) -> None:
        """Verifies strict retention of H030-007 HOLD and operational non-claims."""
        self.assertIn("H030-007", self.content)
        self.assertIn("HOLD_UNTIL_SUPERSEDED", self.content)
        self.assertIn("Zero External Public Routes", self.content)
        self.assertIn("Zero CDN Edge Deployment", self.content)
        self.assertIn("Zero Production Database Deployment", self.content)
        self.assertIn("Zero Real Customer or Personal Data", self.content)
        self.assertIn("Zero Provider, Credential, or Account Mutations", self.content)
        self.assertIn("Mandatory Local-Synthetic Boundary Statement", self.content)

    def test_decision_packet_preservation(self) -> None:
        """Verifies that the release decision packet is preserved intact."""
        self.assertTrue(DECISION_PACKET_PATH.is_file(), f"Missing decision packet at {DECISION_PACKET_PATH}")
        packet_text = DECISION_PACKET_PATH.read_text(encoding="utf-8")
        self.assertIn("ARC-V030-DECPKT-001", packet_text)
        self.assertIn("selected_option: UNFILLED", packet_text)
        self.assertIn("Option 1: Full Approval & Milestone v0.3.0 Closure", packet_text)

    def test_readme_navigation_cross_references(self) -> None:
        """Asserts architecture and release README files cross-reference the decision record."""
        self.assertTrue(ARCH_README_PATH.is_file())
        arch_readme_text = ARCH_README_PATH.read_text(encoding="utf-8")
        self.assertIn("v030-release-decision-record.md", arch_readme_text)
        self.assertIn("ARC-V030-DECREC-001", arch_readme_text)

        self.assertTrue(RELEASE_README_PATH.is_file())
        release_readme_text = RELEASE_README_PATH.read_text(encoding="utf-8")
        self.assertIn("test_v030_release_decision_record.py", release_readme_text)
        self.assertIn("ARC-V030-DECREC-001", release_readme_text)


if __name__ == "__main__":
    unittest.main()
