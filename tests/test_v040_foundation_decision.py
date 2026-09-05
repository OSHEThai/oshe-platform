from __future__ import annotations

import pathlib
import re
import unittest
from typing import Any, Dict, List, Optional

ROOT = pathlib.Path(__file__).resolve().parents[1]
FOUNDATION_DECISION_PATH = ROOT / "docs" / "architecture" / "v040-foundation-decision.md"
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


class V040FoundationDecisionTests(unittest.TestCase):
    def setUp(self) -> None:
        self.assertTrue(FOUNDATION_DECISION_PATH.is_file(), f"Missing decision document at {FOUNDATION_DECISION_PATH}")
        self.content = FOUNDATION_DECISION_PATH.read_text(encoding="utf-8")
        self.frontmatter = parse_simple_frontmatter(self.content)

    def test_foundation_decision_metadata(self) -> None:
        """Asserts required governed frontmatter fields and authority identity."""
        self.assertEqual(self.frontmatter.get("document_id"), "ARC-V040-DECREC-001")
        self.assertEqual(self.frontmatter.get("document_type"), "architecture_decision_record")
        self.assertEqual(self.frontmatter.get("lifecycle_status"), "APPROVED")
        self.assertEqual(self.frontmatter.get("status"), "APPROVED_BY_SOLE_HUMAN_OWNER")
        self.assertEqual(self.frontmatter.get("authority_source"), "HDEC-V040-FOUNDATION-054")
        self.assertEqual(self.frontmatter.get("decided_by"), "Sole Human Owner")
        self.assertEqual(self.frontmatter.get("decided_at"), "2026-09-05T14:25:00Z")
        self.assertEqual(self.frontmatter.get("governing_issue"), "GitHub Issue #112")
        self.assertEqual(self.frontmatter.get("milestone"), "v0.4.0 - OSHE Inspect Private Alpha")

        approved = self.frontmatter.get("approved_gates", [])
        for g in ["H040-001", "H040-002", "H040-003", "H040-004", "H040-005", "H040-006"]:
            self.assertIn(g, approved, f"{g} must be present in approved_gates")

        retained = self.frontmatter.get("retained_holds", [])
        for h in ["H040-007", "H040-008", "H040-009", "H040-010", "H040-011"]:
            self.assertIn(h, retained, f"{h} must be present in retained_holds")

    def test_approved_gates_h040_001_through_006_content(self) -> None:
        """Asserts exact decisions for H040-001 through H040-006 are documented."""
        # H040-001: Four roles and exclusions
        self.assertIn("H040-001", self.content)
        self.assertIn("Checklist Author", self.content)
        self.assertIn("Inspector", self.content)
        self.assertIn("CAPA Owner", self.content)
        self.assertIn("Independent Reviewer", self.content)
        self.assertIn("single-tenant OSHE Inspect private-alpha vertical slice", self.content)

        # H040-002: Responsive client & localization
        self.assertIn("H040-002", self.content)
        self.assertIn("Android Chrome", self.content)
        self.assertIn("Asia/Bangkok", self.content)

        # H040-003: Synthetic or redacted data only
        self.assertIn("H040-003", self.content)
        self.assertIn("Synthetic or redacted data only", self.content)

        # H040-004: Default-deny & AI boundary
        self.assertIn("H040-004", self.content)
        self.assertIn("Default-deny", self.content)
        self.assertIn("AI has zero autonomous safety decision authority", self.content)

        # H040-005: Server authority & conflict quarantine
        self.assertIn("H040-005", self.content)
        self.assertIn("Server authority", self.content)
        self.assertIn("zero last-write-wins", self.content)
        self.assertIn("quarantine", self.content)

        # H040-006: Pilot checklist baseline
        self.assertIn("H040-006", self.content)
        self.assertIn("synthetic non-regulatory pilot checklist", self.content)
        self.assertIn("versioned scoring", self.content)
        self.assertIn("Unknown and Not Applicable", self.content)

    def test_retained_holds_h040_007_through_011_documented(self) -> None:
        """Asserts all five retained holds are documented as HOLD."""
        for h in ["H040-007", "H040-008", "H040-009", "H040-010", "H040-011"]:
            self.assertIn(h, self.content)
        self.assertIn("HOLD", self.content)

    def test_operational_prohibitions_and_non_claims(self) -> None:
        """Asserts strict non-production prohibitions are documented."""
        self.assertIn("Zero Public Route or Deployment Action", self.content)
        self.assertIn("Zero Provider, Credential, or Account Mutation", self.content)
        self.assertIn("Zero Real-User Engagement", self.content)
        self.assertIn("Zero Production Data", self.content)
        self.assertIn("No AI Safety Autonomy", self.content)

    def test_readme_cross_references(self) -> None:
        """Asserts architecture and release README files reference the foundation decision."""
        self.assertTrue(ARCH_README_PATH.is_file())
        arch_text = ARCH_README_PATH.read_text(encoding="utf-8")
        self.assertIn("v040-foundation-decision.md", arch_text)
        self.assertIn("ARC-V040-DECREC-001", arch_text)

        self.assertTrue(RELEASE_README_PATH.is_file())
        release_text = RELEASE_README_PATH.read_text(encoding="utf-8")
        self.assertIn("test_v040_foundation_decision.py", release_text)
        self.assertIn("ARC-V040-DECREC-001", release_text)


if __name__ == "__main__":
    unittest.main()
