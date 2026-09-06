#!/usr/bin/env python3
"""
test_v040_decision_options_prework.py
Automated guard verification suite for V040-I040 / Issue #151 planning-only decision packet prework.

Verifies:
1. Decision options prework document exists and contains required frontmatter metadata.
2. Mandatory "NO DECISION RECORDED" declaration is present.
3. All five outcome options (Continue, Pivot, Extend, Hold, Stop) are present and remain UNSELECTED.
4. Evidence-input ledger explicitly designates Issue #150 and all real user/support/runtime inputs as MISSING or PENDING.
5. All five foundation holds (H040-007 through H040-011) remain strictly on HOLD with zero activation granted.
6. Residual risk and v0.5 / pilot gap analysis fields are explicitly documented.
7. Sole Human Owner decision record placeholder is strictly UNFILLED.
8. Synthetic alpha boundary assertions (zero customer/production data or credentials).
"""

import os
import re
import unittest
from pathlib import Path


def get_repo_root() -> Path:
    current = Path(__file__).resolve().parent
    while current != current.parent:
        if (current / "docs").is_dir() and (current / "tests").is_dir():
            return current
        current = current.parent
    return Path(__file__).resolve().parent.parent


class TestV040DecisionOptionsPrework(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.repo_root = get_repo_root()
        cls.doc_path = cls.repo_root / "docs" / "architecture" / "v040-decision-options-prework.md"

    def test_01_document_exists(self):
        self.assertTrue(
            self.doc_path.is_file(),
            f"Decision options prework document missing at: {self.doc_path}",
        )
        content = self.doc_path.read_text(encoding="utf-8")
        self.assertGreater(len(content), 1500, "Document content is unexpectedly short")

    def test_02_frontmatter_metadata(self):
        content = self.doc_path.read_text(encoding="utf-8")
        frontmatter_match = re.search(r"^---\s*\n(.*?)\n---", content, re.DOTALL)
        self.assertIsNotNone(frontmatter_match, "YAML frontmatter block not found in document")
        fm_text = frontmatter_match.group(1)

        self.assertIn("document_id: DOC-V040-DECISION-OPTIONS-PREWORK-001", fm_text)
        self.assertIn("governing_issue: 151", fm_text)
        self.assertIn("assignment_id: ASN-V040-I040-DECISION-PACKET-PREWORK-001", fm_text)
        self.assertIn("lease_id: LEASE-V040-I040-DECISION-PACKET-PREWORK-001", fm_text)
        self.assertIn("status: DRAFT", fm_text)
        self.assertIn("lifecycle: DRAFT", fm_text)
        self.assertIn("target_milestone: v0.4.0", fm_text)
        self.assertIn("governing_gate: H040-011", fm_text)

        # Retained holds H040-007 .. H040-011
        for hold_id in ["H040-007", "H040-008", "H040-009", "H040-010", "H040-011"]:
            self.assertIn(hold_id, fm_text, f"Retained hold {hold_id} missing from frontmatter")

    def test_03_no_decision_recorded_declaration(self):
        content = self.doc_path.read_text(encoding="utf-8")
        self.assertIn(
            "NO DECISION RECORDED",
            content,
            "Document must explicitly state 'NO DECISION RECORDED'",
        )

    def test_04_five_unselected_options(self):
        content = self.doc_path.read_text(encoding="utf-8")
        options = [
            "Option 1: Continue",
            "Option 2: Pivot",
            "Option 3: Extend",
            "Option 4: Hold",
            "Option 5: Stop",
        ]
        for opt in options:
            self.assertIn(opt, content, f"Outcome option '{opt}' missing from document")

        # Verify that selected_outcome is UNFILLED
        self.assertIn(
            "selected_outcome: UNFILLED",
            content,
            "Decision template must have 'selected_outcome: UNFILLED'",
        )

    def test_05_evidence_input_ledger_status(self):
        content = self.doc_path.read_text(encoding="utf-8")
        # Issue #150 must be MISSING / PENDING
        self.assertTrue(
            re.search(r"Issue #150.*?MISSING\s*/\s*PENDING", content, re.DOTALL),
            "Issue #150 must be marked MISSING / PENDING in evidence ledger",
        )
        # Real user engagement must be MISSING / PENDING
        self.assertTrue(
            re.search(r"Real User Engagement.*?MISSING\s*/\s*PENDING", content, re.DOTALL),
            "Real user engagement must be marked MISSING / PENDING in evidence ledger",
        )
        # Support ownership must be MISSING / PENDING
        self.assertTrue(
            re.search(r"Support Ownership.*?MISSING\s*/\s*PENDING", content, re.DOTALL),
            "Support ownership must be marked MISSING / PENDING in evidence ledger",
        )
        # Runtime activation must be MISSING / PENDING
        self.assertTrue(
            re.search(r"Runtime Activation.*?MISSING\s*/\s*PENDING", content, re.DOTALL),
            "Runtime activation must be marked MISSING / PENDING in evidence ledger",
        )

    def test_06_retained_foundation_holds_table(self):
        content = self.doc_path.read_text(encoding="utf-8")
        holds = [
            ("H040-007", "Technical release authorization"),
            ("H040-008", "Real participant, private-alpha, and UAT engagement"),
            ("H040-009", "Binding support and manual-fallback operational ownership"),
            ("H040-010", "External environment, device, account, route, storage, and notification activation"),
            ("H040-011", "Final outcome, residual-risk acceptance, and v0.5.0 entry decision"),
        ]
        for hold_id, subject in holds:
            self.assertIn(hold_id, content, f"Hold {hold_id} missing")
            self.assertIn(subject, content, f"Hold subject '{subject}' missing")

        for hold_id, _ in holds:
            row_match = re.search(
                rf"\|\s*\*\*{hold_id}\*\*\s*\|\s*([^|]+)\|\s*\*\*HOLD\*\*\s*\|\s*([^|\n]+)\|",
                content,
            )
            self.assertIsNotNone(row_match, f"Row for {hold_id} must have status HOLD")
            enforcement = row_match.group(2)
            self.assertIn(
                "no activation/authorization is granted",
                enforcement,
                f"Row for {hold_id} must say 'no activation/authorization is granted'",
            )

    def test_07_sole_human_signature_placeholder_unfilled(self):
        content = self.doc_path.read_text(encoding="utf-8")
        unfilled_fields = [
            "decided_by: UNFILLED",
            "decided_at: UNFILLED",
            "signature_or_auth_ref: UNFILLED",
            "status: PENDING_SOLE_HUMAN_OWNER_EXECUTION",
        ]
        for field in unfilled_fields:
            self.assertIn(field, content, f"Placeholder field '{field}' missing or filled")

    def test_08_residual_risk_and_gaps_present(self):
        content = self.doc_path.read_text(encoding="utf-8")
        self.assertIn("Residual Technical Risks (Unaccepted)", content)
        self.assertIn("Milestone v0.5 / Pilot Gap Fields", content)

    def test_09_synthetic_boundary_integrity(self):
        forbidden_patterns = [
            r"prod[.-]api\.",
            r"customer[.-]secret",
            r"live[.-]bearer[.-]token",
            r"BEGIN (?:RSA )?PRIVATE KEY",
        ]
        for path in [self.doc_path, Path(__file__)]:
            text = path.read_text(encoding="utf-8")
            for pat in forbidden_patterns:
                self.assertIsNone(
                    re.search(pat, text, re.IGNORECASE),
                    f"Forbidden customer/production pattern '{pat}' detected in {path.name}",
                )


if __name__ == "__main__":
    unittest.main()
