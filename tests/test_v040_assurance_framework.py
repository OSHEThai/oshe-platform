from __future__ import annotations

import pathlib
import re
import unittest
from typing import Any, Dict, List, Optional

ROOT = pathlib.Path(__file__).resolve().parents[1]
FRAMEWORK_DOC_PATH = ROOT / "docs" / "security" / "v040-system-context-assurance-framework.md"

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

        # List items
        if stripped.startswith("- "):
            val = stripped[2:].strip().strip("\"'")
            if current_key and isinstance(data.get(current_key), list):
                data[current_key].append(val)
            continue

        # Top-level key
        if ":" in stripped:
            key, val = stripped.split(":", 1)
            key = key.strip()
            val = val.strip().strip("\"'")
            if val == "":
                data[key] = []
                current_key = key
            else:
                data[key] = val
                current_key = None
    return data


class V040AssuranceFrameworkTests(unittest.TestCase):
    def setUp(self) -> None:
        self.assertTrue(FRAMEWORK_DOC_PATH.is_file(), f"Missing framework doc at {FRAMEWORK_DOC_PATH}")
        self.content = FRAMEWORK_DOC_PATH.read_text(encoding="utf-8")
        self.frontmatter = parse_simple_frontmatter(self.content)

    def test_frontmatter_governance_and_metadata(self) -> None:
        """Asserts required governed frontmatter fields, status, and authority link."""
        self.assertEqual(self.frontmatter.get("document_id"), "SEC-V040-ASR-001")
        self.assertEqual(self.frontmatter.get("document_type"), "security_and_safety_assurance_framework")
        self.assertEqual(self.frontmatter.get("lifecycle_status"), "DRAFT")
        self.assertEqual(self.frontmatter.get("status"), "APPROVED_FOR_LOCAL_ALPHA_DEVELOPMENT")
        self.assertEqual(self.frontmatter.get("governing_decision"), "HDEC-V040-FOUNDATION-054")
        self.assertEqual(self.frontmatter.get("governing_issue"), "GitHub Issue #144")
        self.assertEqual(self.frontmatter.get("milestone"), "v0.4.0 - OSHE Inspect Private Alpha")
        self.assertEqual(self.frontmatter.get("pending_dependency_status"), "PENDING_DEPENDENCY")

    def test_approved_foundation_gates_and_retained_holds(self) -> None:
        """Asserts approved foundation gates H040-001..006 and retained holds H040-007..011."""
        approved = self.frontmatter.get("approved_foundation_gates", [])
        for g in ["H040-001", "H040-002", "H040-003", "H040-004", "H040-005", "H040-006"]:
            self.assertIn(g, approved, f"{g} must be present in approved_foundation_gates")

        retained = self.frontmatter.get("retained_holds", [])
        for h in ["H040-007", "H040-008", "H040-009", "H040-010", "H040-011"]:
            self.assertIn(h, retained, f"{h} must be present in retained_holds")

    def test_unresolved_feature_dependencies_explicitly_pending(self) -> None:
        """Asserts all five required feature dependencies are present and marked PENDING_DEPENDENCY."""
        deps = self.frontmatter.get("unresolved_dependencies", [])
        required_deps = ["V040-I013", "V040-I018", "V040-I021", "V040-I027", "V040-I029"]
        for d in required_deps:
            self.assertIn(d, deps, f"{d} must be in unresolved_dependencies frontmatter list")

        # Confirm documented in body text with PENDING_DEPENDENCY
        for d in required_deps:
            pattern = re.compile(rf"{d}.*?PENDING_DEPENDENCY", re.DOTALL)
            self.assertTrue(pattern.search(self.content), f"Dependency {d} must be marked PENDING_DEPENDENCY in body")

    def test_trust_boundaries_represented(self) -> None:
        """Asserts all six trust boundaries TB-01 through TB-06 are documented."""
        boundaries = ["TB-01", "TB-02", "TB-03", "TB-04", "TB-05", "TB-06"]
        for b in boundaries:
            self.assertIn(b, self.content, f"Trust boundary {b} missing from framework document")

    def test_operational_data_flows_represented(self) -> None:
        """Asserts operational data flows DF-01 through DF-07 are documented."""
        flows = ["DF-01", "DF-02", "DF-03", "DF-04", "DF-05", "DF-06", "DF-07"]
        for f in flows:
            self.assertIn(f, self.content, f"Data flow {f} missing from framework document")

    def test_safety_claims_and_arguments_represented(self) -> None:
        """Asserts top-level safety claims G1 through G5 are documented."""
        goals = ["Goal G1", "Goal G2", "Goal G3", "Goal G4", "Goal G5"]
        for g in goals:
            self.assertIn(g, self.content, f"Safety goal {g} missing from assurance framework")

    def test_negative_test_mapping(self) -> None:
        """Asserts negative tests NEG-TEST-01 through NEG-TEST-10 are cataloged."""
        neg_tests = [f"NEG-TEST-{i:02d}" for i in range(1, 11)]
        for nt in neg_tests:
            self.assertIn(nt, self.content, f"Negative test {nt} missing from framework document")

    def test_accountable_owner_placeholders(self) -> None:
        """Asserts accountable owner placeholders are present for each boundary."""
        owners = [
            "[OWNER-CLIENT-LEAD]",
            "[OWNER-INFRA-HOLD]",
            "[OWNER-SEC-LEAD]",
            "[OWNER-ARCH-LEAD]",
            "[OWNER-DATA-LEAD]",
            "[OWNER-QA-LEAD]",
        ]
        for o in owners:
            self.assertIn(o, self.content, f"Accountable owner placeholder {o} missing")

    def test_segregation_of_duties_and_ai_non_autonomy(self) -> None:
        """Asserts SOD rule for CAPA closure and AI non-authority invariant."""
        self.assertIn("SOD-CAPA-01", self.content)
        self.assertIn("zero autonomous safety decision authority", self.content)

    def test_operational_prohibitions_and_non_claims(self) -> None:
        """Asserts simulated-user non-claim, release non-authorization, and issue non-closure."""
        self.assertIn("Zero Simulated-User Assurance Claims", self.content)
        self.assertIn("No Automated Issue Closure", self.content)
        self.assertIn("Zero Public Route", self.content)


if __name__ == "__main__":
    unittest.main()
