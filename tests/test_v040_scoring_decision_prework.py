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
                if key in ["approved_scoring_decisions", "retained_unselected_policies"]:
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
        self.assertTrue(PREWORK_DOC_PATH.is_file(), f"Missing decision document at {PREWORK_DOC_PATH}")
        self.content = PREWORK_DOC_PATH.read_text(encoding="utf-8")
        self.frontmatter = parse_simple_frontmatter(self.content)

    def test_document_metadata_and_approved_authority(self) -> None:
        """Asserts required governed frontmatter fields, status, and authority link."""
        self.assertEqual(self.frontmatter.get("document_id"), "ARC-V040-SCORING-DECPKT-001")
        self.assertEqual(self.frontmatter.get("document_type"), "architecture_decision_record")
        self.assertEqual(self.frontmatter.get("lifecycle_status"), "APPROVED")
        self.assertEqual(self.frontmatter.get("status"), "APPROVED_BY_SOLE_HUMAN_OWNER")
        self.assertEqual(self.frontmatter.get("authority_source"), "HDEC-V040-SCORING-058")
        self.assertEqual(self.frontmatter.get("governing_issue"), "GitHub Issue #136")
        self.assertEqual(self.frontmatter.get("milestone"), "v0.4.0 - OSHE Inspect Private Alpha")

        approved = self.frontmatter.get("approved_foundation_gates", [])
        for g in ["H040-001", "H040-002", "H040-003", "H040-004", "H040-005", "H040-006"]:
            self.assertIn(g, approved, f"{g} must be present in approved_foundation_gates")

        deferred = self.frontmatter.get("retained_holds", [])
        for h in ["H040-007", "H040-008", "H040-009", "H040-010", "H040-011"]:
            self.assertIn(h, deferred, f"{h} must be present in retained_holds")

    def test_approved_scoring_choices(self) -> None:
        """Asserts the five exact approved choices from HDEC-V040-SCORING-058."""
        decisions = self.frontmatter.get("approved_scoring_decisions", {})
        self.assertEqual(decisions.get("selected_scoring_model"), "MODEL_2_WEIGHTED")
        self.assertEqual(decisions.get("selected_unknown_handling"), "U1_QUARANTINE_DENOMINATOR")
        self.assertEqual(decisions.get("selected_rounding_rule"), "R1_ROUND_HALF_UP")
        self.assertEqual(decisions.get("selected_critical_fail_policy"), "CF1_PRIORITY_FLAG")
        self.assertEqual(str(decisions.get("selected_passing_threshold_percent")), "80")

        # Confirm presence in body text
        self.assertIn("MODEL_2_WEIGHTED", self.content)
        self.assertIn("U1_QUARANTINE_DENOMINATOR", self.content)
        self.assertIn("R1_ROUND_HALF_UP", self.content)
        self.assertIn("CF1_PRIORITY_FLAG", self.content)
        self.assertIn("80%", self.content)

    def test_three_pass_predicates_enforced(self) -> None:
        """Asserts all three mandatory pass predicates are defined and documented."""
        predicates = self.frontmatter.get("pass_predicates", [])
        self.assertEqual(len(predicates), 3, "Must define exactly three pass predicates")
        self.assertIn("No critical-fail condition is present.", predicates[0])
        self.assertIn("No UNKNOWN response remains unresolved or quarantined.", predicates[1])
        self.assertIn("Score is greater than or equal to 80.00 percent", predicates[2])

        # Confirm in body text
        self.assertIn("Mandatory Inspection Pass Predicates", self.content)
        self.assertIn("No critical-fail condition is present", self.content)
        self.assertIn("No UNKNOWN response remains unresolved or quarantined", self.content)
        self.assertIn("80.00 percent", self.content)

    def test_populated_decision_record(self) -> None:
        """Asserts the decision record template is populated with HDEC-V040-SCORING-058 approval."""
        self.assertIn("HDEC-V040-SCORING-058", self.content)
        self.assertIn("status: APPROVED_STANDING_UNTIL_SUPERSEDED", self.content)
        self.assertIn("authority: Sole Human Owner", self.content)
        self.assertIn("approved_at_utc: '2026-09-05T23:26:08Z'", self.content)
        self.assertIn("selected_scoring_model: MODEL_2_WEIGHTED", self.content)
        self.assertIn("selected_unknown_handling: U1_QUARANTINE_DENOMINATOR", self.content)
        self.assertIn("selected_rounding_rule: R1_ROUND_HALF_UP", self.content)
        self.assertIn("selected_critical_fail_policy: CF1_PRIORITY_FLAG", self.content)
        self.assertIn("selected_passing_threshold_percent: 80", self.content)

    def test_na_and_unknown_semantics_modeled(self) -> None:
        """Asserts NA denominator exclusion and Unknown response quarantine under U1."""
        self.assertIn("Not Applicable (`NA`) Semantics", self.content)
        self.assertIn("Excluded from denominator", self.content)
        self.assertIn("Zero Negative Impact", self.content)

        self.assertIn("Approved Unknown (`UNKNOWN`) Semantics (`U1_QUARANTINE_DENOMINATOR`)", self.content)
        self.assertIn("Denominator Quarantine", self.content)
        self.assertIn("PROVISIONAL_PENDING_UNKNOWN_RESOLUTION", self.content)

    def test_critical_fail_and_ai_safety_boundary(self) -> None:
        """Asserts CF1 priority flag enforcement and AI safety decision authority exclusion."""
        self.assertIn("CF1 (Priority Flag Without Score Masking)", self.content)
        self.assertIn("NON_COMPLIANT_CRITICAL", self.content)
        self.assertIn("ZERO authority", self.content)
        self.assertIn("H040-004", self.content)

    def test_operational_prohibitions_and_retained_holds(self) -> None:
        """Asserts retained holds H040-007 through H040-011 and non-production prohibitions."""
        for h in ["H040-007", "H040-008", "H040-009", "H040-010", "H040-011"]:
            self.assertIn(h, self.content)
        self.assertIn("HOLD", self.content)

        self.assertIn("GitHub Issue #136 is **CLOSED** following formal acceptance and merge of PR #1075", self.content)
        self.assertIn("No External Route or Deployment Action", self.content)


if __name__ == "__main__":
    unittest.main()
