from __future__ import annotations

import pathlib
import re
import unittest
from typing import Any, Dict, Optional


ROOT = pathlib.Path(__file__).resolve().parents[1]
DECISION_PATH = ROOT / "docs" / "architecture" / "v050-foundation-decision.md"
ARCHITECTURE_README_PATH = ROOT / "docs" / "architecture" / "README.md"
RELEASE_README_PATH = ROOT / "tests" / "release" / "README.md"
FRONTMATTER_PATTERN = re.compile(r"^---\s*\n(.*?)\n---\s*\n", re.DOTALL)


def parse_simple_frontmatter(content: str) -> Dict[str, Any]:
    match = FRONTMATTER_PATTERN.match(content)
    if not match:
        raise ValueError("Missing YAML frontmatter")

    data: Dict[str, Any] = {}
    current_key: Optional[str] = None
    for line in match.group(1).splitlines():
        if not line.strip() or line.lstrip().startswith("#"):
            continue
        stripped = line.strip()
        if stripped.startswith("- "):
            if current_key and isinstance(data.get(current_key), list):
                data[current_key].append(stripped[2:].strip().strip("\"'"))
            continue
        if ":" not in line:
            continue
        key, value = line.split(":", 1)
        key = key.strip()
        value = value.strip().strip("\"'")
        if value:
            data[key] = value
            current_key = None
        else:
            data[key] = []
            current_key = key
    return data


class V050FoundationDecisionTests(unittest.TestCase):
    def setUp(self) -> None:
        self.assertTrue(DECISION_PATH.is_file(), f"Missing decision record: {DECISION_PATH}")
        self.content = DECISION_PATH.read_text(encoding="utf-8")
        self.normalized_content = " ".join(self.content.split())
        self.frontmatter = parse_simple_frontmatter(self.content)

    def test_governed_identity_and_credit_boundary(self) -> None:
        self.assertEqual(self.frontmatter.get("document_id"), "ARC-V050-DECREC-001")
        self.assertEqual(self.frontmatter.get("document_type"), "architecture_decision_record")
        self.assertEqual(self.frontmatter.get("lifecycle_status"), "APPROVED")
        self.assertEqual(
            self.frontmatter.get("status"),
            "APPROVED_FOR_SYNTHETIC_FOUNDATION_MATERIALIZATION",
        )
        self.assertEqual(
            self.frontmatter.get("authority_source"),
            "HDEC-V050-WAVE0-EXECUTION-ACTIVATION-001",
        )
        self.assertEqual(self.frontmatter.get("governing_execution_successor"), "V050-E001")
        self.assertEqual(self.frontmatter.get("planning_predecessor"), "GitHub Issue #152 / V050-I001")
        self.assertEqual(self.frontmatter.get("credit_boundary"), "FOUNDATION_BOUNDARY_MATERIALIZATION_ONLY")

    def test_approved_gates_and_retained_holds(self) -> None:
        approved = self.frontmatter.get("approved_gates", [])
        retained = self.frontmatter.get("retained_holds", [])
        for gate in [f"H050-{number:03d}" for number in range(1, 9)]:
            self.assertIn(gate, approved)
            self.assertIn(gate, self.content)
        for gate in [f"H050-{number:03d}" for number in range(9, 14)]:
            self.assertIn(gate, retained)
            self.assertIn(gate, self.content)

    def test_protected_boundary_invariants(self) -> None:
        required_text = [
            "synthetic or redacted fixtures",
            "Default-deny authorization",
            "No report, QR, offline client, dashboard, notification, derived metric, or AI output is",
            "never uses last-write-wins",
            "AI has zero autonomous safety",
            "unclassified and requires human triage",
            "closed automatically",
            "internal-only, anonymized, deny-by-default",
        ]
        for text in required_text:
            self.assertIn(text, self.normalized_content)

    def test_non_claims_prevent_operational_credit(self) -> None:
        required_text = [
            "no real data",
            "participant activity",
            "external service/device/account/route activation",
            "protected merge",
            "release",
            "residual-risk acceptance",
        ]
        for text in required_text:
            self.assertIn(text, self.normalized_content)

    def test_readme_references(self) -> None:
        self.assertTrue(ARCHITECTURE_README_PATH.is_file())
        self.assertTrue(RELEASE_README_PATH.is_file())
        self.assertIn("v050-foundation-decision.md", ARCHITECTURE_README_PATH.read_text(encoding="utf-8"))
        self.assertIn("test_v050_foundation_decision.py", RELEASE_README_PATH.read_text(encoding="utf-8"))


if __name__ == "__main__":
    unittest.main()
