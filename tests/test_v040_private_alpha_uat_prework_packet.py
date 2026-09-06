#!/usr/bin/env python3
"""
test_v040_private_alpha_uat_prework_packet.py
Automated validation suite for v0.4.0 Private-Alpha / UAT Protocol and Evidence-Template Prework Packet.
Governed by Issue #149 / ASN-V040-I038-PRIVATE-ALPHA-PREWORK-001.

Verifies:
1. Document existence and non-trivial length.
2. Frontmatter metadata conforms to governance requirements (document_id, human_gates, retained_holds).
3. Strict blocked/hold status for real participants, sessions, accounts/devices/environments, and consent.
4. Human gates H040-007, H040-008, and H040-010 are strictly retained on HOLD / BLOCKED.
5. All six required sections are present:
   - Proposed Role and Eligibility Matrix
   - Controlled Synthetic Test Scenario Matrix
   - Session ID & Evidence Capture Template
   - Onboarding, Support, and Consent Placeholders
   - Stop and Closeout Checklist
   - Explicit Missing-Evidence Ledger
6. Placeholders and template schemas conform to safety boundaries.
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


class TestV040PrivateAlphaUatPreworkPacket(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.repo_root = get_repo_root()
        cls.doc_path = (
            cls.repo_root
            / "docs"
            / "architecture"
            / "v040-private-alpha-uat-prework-packet.md"
        )

    def test_01_document_exists(self):
        self.assertTrue(
            self.doc_path.is_file(),
            f"Private-alpha UAT prework packet missing at: {self.doc_path}",
        )
        content = self.doc_path.read_text(encoding="utf-8")
        self.assertGreater(
            len(content), 2000, "Packet content is unexpectedly short"
        )

    def test_02_frontmatter_metadata(self):
        content = self.doc_path.read_text(encoding="utf-8")
        frontmatter_match = re.search(r"^---\s*\n(.*?)\n---", content, re.DOTALL)
        self.assertIsNotNone(
            frontmatter_match, "YAML frontmatter block not found in packet"
        )
        fm_text = frontmatter_match.group(1)

        self.assertIn("document_id: DOC-PLAN-V040-UAT-001", fm_text)
        self.assertIn("governing_issue: \"GitHub Issue #149\"", fm_text)
        self.assertIn(
            "assignment_id: ASN-V040-I038-PRIVATE-ALPHA-PREWORK-001", fm_text
        )
        self.assertIn(
            "lease_id: LEASE-V040-I038-PRIVATE-ALPHA-PREWORK-001", fm_text
        )
        self.assertIn(
            "credit_boundary: PLANNING_ONLY_NO_EXECUTION_OR_RELEASE_CREDIT", fm_text
        )

        # Human gates in frontmatter
        for gate in ["H040-007", "H040-008", "H040-010"]:
            self.assertIn(gate, fm_text, f"Human gate {gate} missing from frontmatter")

        # Retained holds in frontmatter
        for hold_id in ["H040-007", "H040-008", "H040-009", "H040-010", "H040-011"]:
            self.assertIn(
                hold_id, fm_text, f"Retained hold {hold_id} missing from frontmatter"
            )

    def test_03_governance_holds_and_blocked_actions(self):
        content = self.doc_path.read_text(encoding="utf-8")
        content_lower = content.lower()

        # Blocked action declarations
        self.assertIn("zero real participants", content_lower)
        self.assertIn("zero real sessions", content_lower)
        self.assertIn("zero consent acceptance", content_lower)
        self.assertIn(
            "zero authority is granted to alter, bypass, or lift any hold",
            content_lower,
        )

        # Human gates H040-007, H040-008, H040-010 on HOLD / BLOCKED
        self.assertIn("`H040-007` | Technical release authorization | **HOLD / BLOCKED**", content)
        self.assertIn("`H040-008` | Real participant, private-alpha, and UAT engagement | **HOLD / BLOCKED**", content)
        self.assertIn("`H040-009` | Binding support and manual-fallback operational ownership | **HOLD / BLOCKED**", content)
        self.assertIn("`H040-010` | External environment, device, account, route, storage, and notification activation | **HOLD / BLOCKED**", content)
        self.assertIn("`H040-011` | Final outcome, residual-risk acceptance, and v0.5.0 entry decision | **HOLD / BLOCKED**", content)

    def test_04_core_sections_presence(self):
        content = self.doc_path.read_text(encoding="utf-8")

        required_sections = [
            "Proposed Role and Eligibility Matrix",
            "Controlled Synthetic Test Scenario Matrix",
            "Session ID & Evidence Capture Template",
            "Onboarding, Support, and Consent Placeholders",
            "Stop and Closeout Checklist",
            "Explicit Missing-Evidence Ledger",
        ]
        for section in required_sections:
            self.assertIn(
                section, content, f"Required section '{section}' missing from document"
            )

    def test_05_placeholders_and_template_schema(self):
        content = self.doc_path.read_text(encoding="utf-8")

        # Placeholders
        self.assertIn("[CONSENT_PLACEHOLDER - NOT ACCEPTED / BLOCKED]", content)
        self.assertIn("[SUPPORT_PLACEHOLDER - NO LIVE RUNBOOK / BLOCKED]", content)
        self.assertIn("[ONBOARDING_PLACEHOLDER - NO REAL ACCOUNTS / BLOCKED]", content)

        # Session ID template pattern
        self.assertIn("SESS-V040-SYN-", content)

    def test_06_synthetic_scenarios_coverage(self):
        content = self.doc_path.read_text(encoding="utf-8")

        # Five synthetic scenarios
        for scn in [
            "SCN-SYN-UAT-01",
            "SCN-SYN-UAT-02",
            "SCN-SYN-UAT-03",
            "SCN-SYN-UAT-04",
            "SCN-SYN-UAT-05",
        ]:
            self.assertIn(scn, content, f"Synthetic scenario {scn} missing from matrix")

    def test_07_missing_evidence_ledger_completeness(self):
        content = self.doc_path.read_text(encoding="utf-8")

        required_missing_items = [
            "Real Human User Usability",
            "Physical Mobile Device Compatibility",
            "Live Network & Bandwidth Performance",
            "Production Cloud Infrastructure Runtime",
            "Disaster Recovery & Cold Backup Restore",
            "Operational Support Ownership & SLAs",
            "Sovereign Human Gate Approvals",
        ]
        for item in required_missing_items:
            self.assertIn(
                item, content, f"Missing evidence item '{item}' not documented in ledger"
            )


if __name__ == "__main__":
    unittest.main()
