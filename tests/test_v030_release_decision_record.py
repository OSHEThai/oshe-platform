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

    def test_decision_record_structure_and_frontmatter(self) -> None:
        """Asserts required governed frontmatter fields and approval state."""
        self.assertEqual(self.frontmatter.get("document_id"), "ARC-V030-DECREC-001")
        self.assertEqual(self.frontmatter.get("document_type"), "architecture_decision_record")
        self.assertEqual(self.frontmatter.get("lifecycle_status"), "APPROVED")
        self.assertEqual(self.frontmatter.get("status"), "APPROVED_BY_SOLE_HUMAN_OWNER")
        self.assertEqual(self.frontmatter.get("governing_issue"), "GitHub Issue #111")
        self.assertEqual(self.frontmatter.get("governing_decision"), "HDEC-V030-RELEASE-053")
        self.assertEqual(self.frontmatter.get("governing_gate"), "H030-008")
        self.assertEqual(self.frontmatter.get("selected_option"), "OPTION_1_FULL_APPROVAL_AND_MILESTONE_CLOSURE")

        retained = self.frontmatter.get("retained_human_gates", [])
        self.assertIn("H030-007", retained, "H030-007 must remain retained on HOLD")

        credit = self.frontmatter.get("credit_boundary", "")
        self.assertIn("NO_TAGGING_OR_RELEASE_WITHOUT_SUCCESSOR", credit)

    def test_granted_authorizations(self) -> None:
        """Verifies that Option 1 granted authorizations are explicitly documented."""
        self.assertIn("Milestone v0.3.0 Closure", self.content)
        self.assertIn("Git Release Tagging Authority", self.content)
        self.assertIn("v0.3.0", self.content)
        self.assertIn("GitHub Release Publication Authority", self.content)
        self.assertIn("Milestone v0.4.0 Entry Authority", self.content)

    def test_retained_holds_and_prohibitions(self) -> None:
        """Verifies that H030-007 HOLD and operational prohibitions are strictly enforced."""
        self.assertIn("H030-007", self.content)
        self.assertIn("HOLD", self.content)
        self.assertIn("No Production Deployment", self.content)
        self.assertIn("No Cryptographic Signing with Production Identities", self.content)
        self.assertIn("No Provider Route Dispatch", self.content)
        self.assertIn("No Release / Tagging Action in Present Task", self.content)

    def test_pre_decision_packet_preservation(self) -> None:
        """Verifies that the pre-decision proposal packet exists and remains unfilled."""
        self.assertTrue(DECISION_PACKET_PATH.is_file())
        packet_text = DECISION_PACKET_PATH.read_text(encoding="utf-8")
        self.assertIn("ARC-V030-DECPKT-001", packet_text)
        self.assertIn("selected_option: UNFILLED", packet_text)
        self.assertIn("decided_by: UNFILLED", packet_text)

    def test_readme_navigation_cross_references(self) -> None:
        """Asserts architecture and release README files cross-reference the decision record."""
        self.assertTrue(ARCH_README_PATH.is_file())
        arch_readme_text = ARCH_README_PATH.read_text(encoding="utf-8")
        self.assertIn("v030-release-decision-record.md", arch_readme_text)
        self.assertIn("HDEC-V030-RELEASE-053", arch_readme_text)

        self.assertTrue(RELEASE_README_PATH.is_file())
        release_readme_text = RELEASE_README_PATH.read_text(encoding="utf-8")
        self.assertIn("test_v030_release_decision_record.py", release_readme_text)
        self.assertIn("HDEC-V030-RELEASE-053", release_readme_text)


if __name__ == "__main__":
    unittest.main()
