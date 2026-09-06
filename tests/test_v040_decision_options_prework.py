#!/usr/bin/env python3
"""
test_v040_decision_options_prework.py
Automated validation suite for v0.4.0 Decision Options Prework Packet Template.
Governed by Issue #151 / ASN-V040-I040-DECISION-PACKET-PREWORK-002 / LEASE-V040-I040-DECISION-PACKET-PREWORK-002.

Verifies:
1. Document existence and non-trivial length.
2. Frontmatter metadata conforms to governance requirements (document_id, governing_gate, retained_holds).
3. Explicit "NO DECISION RECORDED" statement; zero option recommended or chosen; zero risk accepted.
4. Frozen scope, version, configuration, and limitations fields present.
5. Evidence-input ledger references Issue #150 and marks real inputs MISSING / PENDING.
6. Five mutually exclusive unselected options (CONTINUE, PIVOT, EXTEND, HOLD, STOP).
7. Residual-risk and owner signature placeholders present.
8. Retained foundation holds H040-007 through H040-011 strictly on HOLD / BLOCKED with zero authority grant.
"""

import os
import re
import unittest
from pathlib import Path


def get_repo_root() -> Path:
    current = Path(__file__).resolve().parent
    while current != current.parent:
        if (current / "docs").is_dir() and (current / "modules").is_dir():
            return current
        current = current.parent
    return Path(__file__).resolve().parent.parent


class TestV040DecisionOptionsPrework(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.repo_root = get_repo_root()
        cls.doc_path = (
            cls.repo_root
            / "docs"
            / "architecture"
            / "v040-decision-options-prework.md"
        )

    def test_01_document_exists(self):
        self.assertTrue(
            self.doc_path.is_file(),
            f"Decision options prework template missing at: {self.doc_path}",
        )
        content = self.doc_path.read_text(encoding="utf-8")
        self.assertGreater(
            len(content), 2000, "Template document content is unexpectedly short"
        )

    def test_02_frontmatter_metadata(self):
        content = self.doc_path.read_text(encoding="utf-8")
        frontmatter_match = re.search(r"^---\s*\n(.*?)\n---", content, re.DOTALL)
        self.assertIsNotNone(
            frontmatter_match, "YAML frontmatter block not found in template"
        )
        fm_text = frontmatter_match.group(1)

        self.assertIn("document_id: DOC-PLAN-V040-DEC-001", fm_text)
        self.assertIn("governing_issue: \"GitHub Issue #151\"", fm_text)
        self.assertIn("governing_gate: H040-011", fm_text)
        self.assertIn(
            "assignment_id: ASN-V040-I040-DECISION-PACKET-PREWORK-002", fm_text
        )
        self.assertIn(
            "lease_id: LEASE-V040-I040-DECISION-PACKET-PREWORK-002", fm_text
        )
        self.assertIn(
            "credit_boundary: PLANNING_ONLY_TEMPLATE_PREPARATION_NO_DECISION_RECORDED", fm_text
        )

        # Retained holds in frontmatter
        for hold_id in ["H040-007", "H040-008", "H040-009", "H040-010", "H040-011"]:
            self.assertIn(
                hold_id, fm_text, f"Retained hold {hold_id} missing from frontmatter"
            )

    def test_03_no_decision_recorded_statement(self):
        content = self.doc_path.read_text(encoding="utf-8")

        # Must explicitly declare NO DECISION RECORDED
        self.assertIn("NO DECISION RECORDED", content)
        clean_text = content.lower().replace("*", "")
        self.assertIn("makes no decision", clean_text)
        self.assertIn("recommends no outcome option", clean_text)
        self.assertIn("accepts no residual risk", clean_text)
        self.assertIn("authorizes no next action", clean_text)

    def test_04_frozen_parameters_present(self):
        content = self.doc_path.read_text(encoding="utf-8")

        self.assertIn("Frozen Scope", content)
        self.assertIn("Frozen Version", content)
        self.assertIn("Frozen Configuration", content)
        self.assertIn("Frozen Limitations", content)

        # Key frozen parameters
        self.assertIn("single-tenant OSHE Inspect vertical slice", content)
        self.assertIn("v0.4.0 - OSHE Inspect Private Alpha", content)
        self.assertIn("ten_synthetic_alpha", content)

    def test_05_evidence_input_ledger_with_missing_inputs(self):
        content = self.doc_path.read_text(encoding="utf-8")

        # Must reference Issue #150
        self.assertIn("Issue #150", content)

        # Real inputs must be marked MISSING / PENDING
        self.assertIn("MISSING / PENDING", content)
        self.assertIn("Real Human User & Usability Feedback", content)
        self.assertIn("Real UAT Field Walkthrough Trials", content)
        self.assertIn("Operational Support & Helpdesk Ownership", content)
        self.assertIn("Physical Mobile Hardware Compatibility", content)
        self.assertIn("Live Network & Cellular Field Performance", content)
        self.assertIn("Cloud Runtime Infrastructure Execution", content)
        self.assertIn("Disaster Recovery & Backup Restoration", content)

    def test_06_five_unselected_options(self):
        content = self.doc_path.read_text(encoding="utf-8")

        expected_options = [
            "Option 1: CONTINUE",
            "Option 2: PIVOT",
            "Option 3: EXTEND",
            "Option 4: HOLD",
            "Option 5: STOP",
        ]
        for opt in expected_options:
            self.assertIn(opt, content, f"Option '{opt}' missing from decision structure")

        # All options unselected
        self.assertIn("[UNSELECTED - NONE CHOSEN]", content)

    def test_07_placeholders_and_retained_holds(self):
        content = self.doc_path.read_text(encoding="utf-8")

        # Placeholders
        self.assertIn("[RESIDUAL_RISK_PLACEHOLDER - NO RESIDUAL RISK ACCEPTED / PENDING SOLE HUMAN OWNER]", content)
        self.assertIn("[OWNER_DECISION_PLACEHOLDER - NO DECISION RECORDED / PENDING SOLE HUMAN OWNER]", content)
        self.assertIn("[UNFILLED - SOLE HUMAN OWNER ONLY]", content)

        # Retained holds table
        for hold_id in ["H040-007", "H040-008", "H040-009", "H040-010", "H040-011"]:
            self.assertIn(hold_id, content, f"Hold {hold_id} missing from body")

        self.assertIn("`H040-007` | Technical release authorization | **HOLD / BLOCKED**", content)
        self.assertIn("`H040-008` | Real participant, private-alpha, and UAT engagement | **HOLD / BLOCKED**", content)
        self.assertIn("`H040-009` | Binding support and manual-fallback operational ownership | **HOLD / BLOCKED**", content)
        self.assertIn("`H040-010` | External environment, device, account, route, storage, and notification activation | **HOLD / BLOCKED**", content)
        self.assertIn("`H040-011` | Final outcome, residual-risk acceptance, and v0.5.0 entry decision | **HOLD / BLOCKED**", content)

        self.assertIn("Zero authority is granted to alter, bypass, or lift any hold", content)


if __name__ == "__main__":
    unittest.main()
