from __future__ import annotations

import pathlib
import re
import unittest
from typing import Any, Dict, List, Optional

ROOT = pathlib.Path(__file__).resolve().parents[1]
PREWORK_DOC_PATH = ROOT / "docs" / "architecture" / "v040-scoring-decision-prework.md"

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
        indent = len(line) - len(line.lstrip())

        # List items
        if stripped.startswith("- "):
            val = stripped[2:].strip().strip("\"'")
            if current_key and isinstance(data.get(current_key), list):
                data[current_key].append(val)
            continue

        # Sub-dictionary items
        if indent > 0 and ":" in stripped and current_key:
            k, v = stripped.split(":", 1)
            k = k.strip()
            v = v.strip().strip("\"'")
            if isinstance(data.get(current_key), dict):
                data[current_key][k] = v
            continue

        # Top-level key
        if ":" in stripped:
            key, val = stripped.split(":", 1)
            key = key.strip()
            val = val.strip().strip("\"'")
            if val == "":
                if key == "retained_unselected_policies":
                    data[key] = {}
                else:
                    data[key] = []
                current_key = key
            else:
                data[key] = val
                current_key = None
    return data


class V040ScoringDecisionPreworkTests(unittest.TestCase):
    def setUp(self) -> None:
        self.assertTrue(PREWORK_DOC_PATH.is_file(), f"Missing prework document at {PREWORK_DOC_PATH}")
        self.content = PREWORK_DOC_PATH.read_text(encoding="utf-8")
        self.frontmatter = parse_simple_frontmatter(self.content)

    def test_document_metadata_and_governance(self) -> None:
        """Asserts required governed frontmatter fields, status, and authority link."""
        self.assertEqual(self.frontmatter.get("document_id"), "ARC-V040-SCORING-DECPKT-001")
        self.assertEqual(self.frontmatter.get("document_type"), "architecture_decision_packet")
        self.assertEqual(self.frontmatter.get("lifecycle_status"), "DRAFT")
        self.assertEqual(self.frontmatter.get("status"), "PENDING_OWNER_DECISION")
        self.assertEqual(self.frontmatter.get("authority_source"), "HDEC-V040-FOUNDATION-054")
        self.assertEqual(self.frontmatter.get("governing_issue"), "GitHub Issue #136")
        self.assertEqual(self.frontmatter.get("milestone"), "v0.4.0 - OSHE Inspect Private Alpha")

        approved = self.frontmatter.get("approved_foundation_gates", [])
        for g in ["H040-001", "H040-002", "H040-003", "H040-004", "H040-005", "H040-006"]:
            self.assertIn(g, approved, f"{g} must be present in approved_foundation_gates")

        deferred = self.frontmatter.get("retained_holds", [])
        for h in ["H040-007", "H040-008", "H040-009", "H040-010", "H040-011"]:
            self.assertIn(h, deferred, f"{h} must be present in retained_holds")

    def test_retained_unselected_policies_explicitly_marked(self) -> None:
        """Asserts binding scoring policy is marked HUMAN_OWNED_UNSELECTED and unselected."""
        unselected = self.frontmatter.get("retained_unselected_policies", {})
        self.assertEqual(unselected.get("binding_scoring_policy"), "HUMAN_OWNED_UNSELECTED")
        self.assertEqual(unselected.get("finding_closure_policy"), "HUMAN_OWNED_UNSELECTED")
        self.assertEqual(unselected.get("offline_authority"), "HUMAN_OWNED_UNSELECTED")

        self.assertIn("HUMAN_OWNED_UNSELECTED", self.content)
        self.assertIn("Binding Scoring Policy Remains Unselected", self.content)
        self.assertIn("PENDING_OWNER_DECISION", self.content)

    def test_na_and_unknown_semantics_modeled(self) -> None:
        """Asserts NA denominator exclusion and Unknown response handling alternatives."""
        # NA semantics
        self.assertIn("Not Applicable (`NA`) Semantics", self.content)
        self.assertIn("Excluded from denominator", self.content)
        self.assertIn("Zero Negative Impact", self.content)

        # Unknown semantics
        self.assertIn("Unknown (`UNKNOWN`) Semantics", self.content)
        self.assertIn("Mandatory Justification", self.content)
        self.assertIn("Alternative U1", self.content)
        self.assertIn("Alternative U2", self.content)
        self.assertIn("Alternative U3", self.content)

    def test_alternative_scoring_formulas_present(self) -> None:
        """Asserts all three scoring formulation alternatives are presented with tradeoffs."""
        self.assertIn("Model 1: Flat Percentage Compliance", self.content)
        self.assertIn("Model 2: Category/Section Weighted Normalized Scoring", self.content)
        self.assertIn("Model 3: Risk-Deductive Penalty Scoring", self.content)
        self.assertIn("Tradeoff Profile", self.content)

    def test_rounding_and_precision_standards(self) -> None:
        """Asserts fixed-point basis points and rounding alternatives."""
        self.assertIn("Fixed-Point Precision Standard", self.content)
        self.assertIn("ROUND_HALF_UP", self.content)
        self.assertIn("ROUND_DOWN", self.content)

    def test_critical_fail_and_ai_safety_boundary(self) -> None:
        """Asserts critical-fail alternatives and AI safety decision authority exclusion."""
        self.assertIn("Critical-Fail Invariants", self.content)
        self.assertIn("Alternative CF1: Non-Overriding Priority Flag", self.content)
        self.assertIn("Alternative CF2: Hard Override", self.content)
        self.assertIn("ZERO authority", self.content)
        self.assertIn("H040-004", self.content)

    def test_unfilled_decision_record_template(self) -> None:
        """Asserts the decision record template is strictly UNFILLED and pending owner decision."""
        self.assertIn("HDEC-V040-SCORING-058", self.content)
        self.assertIn("selected_scoring_model: UNFILLED", self.content)
        self.assertIn("selected_unknown_handling: UNFILLED", self.content)
        self.assertIn("selected_rounding_rule: UNFILLED", self.content)
        self.assertIn("selected_critical_fail_policy: UNFILLED", self.content)
        self.assertIn("selected_passing_threshold: UNFILLED", self.content)
        self.assertIn("decided_by: UNFILLED", self.content)
        self.assertIn("decided_at: UNFILLED", self.content)
        self.assertIn("status: PENDING_OWNER_DECISION", self.content)

    def test_operational_prohibitions_and_retained_holds(self) -> None:
        """Asserts retained holds H040-007 through H040-011 and prohibitions."""
        for h in ["H040-007", "H040-008", "H040-009", "H040-010", "H040-011"]:
            self.assertIn(h, self.content)
        self.assertIn("HOLD", self.content)

        self.assertIn("GitHub Issue #136 remains **OPEN**", self.content)
        self.assertIn("No Binding Policy Selection", self.content)
        self.assertIn("No External Route or Deployment Action", self.content)


if __name__ == "__main__":
    unittest.main()
