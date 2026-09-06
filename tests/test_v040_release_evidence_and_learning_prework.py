#!/usr/bin/env python3
"""
test_v040_release_evidence_and_learning_prework.py
Automated validation suite for v0.4.0 Release Evidence and Learning Prework Scorecard.
Governed by Issue #150 / ASN-V040-I039-EVIDENCE-LEARNING-PREWORK-001.

Verifies:
1. Document existence and non-trivial length.
2. Frontmatter metadata conforms to governance requirements (document_id, human_gates, retained_holds).
3. Scorecard sections presence: source-attribution, technical synthetic results, missing evidence,
   proposed-unmeasured thresholds, defect/risk reconciliation, and recommendations.
4. Explicitly missing evidence catalog (human, UAT, support, device, network, runtime, backup).
5. Proposed-unmeasured thresholds catalog carrying NON-BINDING_PROPOSED and UNMEASURED markers.
6. Formal recommendation records INSUFFICIENT EVIDENCE rather than acceptance.
7. Retained foundation holds H040-007 through H040-011 on HOLD / BLOCKED with zero authority grant.
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


class TestV040ReleaseEvidenceAndLearningPrework(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.repo_root = get_repo_root()
        cls.doc_path = (
            cls.repo_root
            / "docs"
            / "architecture"
            / "v040-release-evidence-and-learning-prework.md"
        )

    def test_01_document_exists(self):
        self.assertTrue(
            self.doc_path.is_file(),
            f"Release evidence and learning prework document missing at: {self.doc_path}",
        )
        content = self.doc_path.read_text(encoding="utf-8")
        self.assertGreater(
            len(content), 2000, "Scorecard document content is unexpectedly short"
        )

    def test_02_frontmatter_metadata(self):
        content = self.doc_path.read_text(encoding="utf-8")
        frontmatter_match = re.search(r"^---\s*\n(.*?)\n---", content, re.DOTALL)
        self.assertIsNotNone(
            frontmatter_match, "YAML frontmatter block not found in document"
        )
        fm_text = frontmatter_match.group(1)

        self.assertIn("document_id: EV-V040-REL-LRN-001", fm_text)
        self.assertIn("governing_issue: \"GitHub Issue #150\"", fm_text)
        self.assertIn(
            "assignment_id: ASN-V040-I039-EVIDENCE-LEARNING-PREWORK-001", fm_text
        )
        self.assertIn(
            "lease_id: LEASE-V040-I039-EVIDENCE-LEARNING-PREWORK-001", fm_text
        )
        self.assertIn(
            "credit_boundary: PLANNING_ONLY_NO_EXECUTION_OR_RELEASE_CREDIT", fm_text
        )

        # Human gates in frontmatter
        for gate in ["H040-007", "H040-008", "H040-009", "H040-010", "H040-011"]:
            self.assertIn(gate, fm_text, f"Human gate {gate} missing from frontmatter")

        # Retained holds in frontmatter
        for hold_id in ["H040-007", "H040-008", "H040-009", "H040-010", "H040-011"]:
            self.assertIn(
                hold_id, fm_text, f"Retained hold {hold_id} missing from frontmatter"
            )

    def test_03_scorecard_sections_presence(self):
        content = self.doc_path.read_text(encoding="utf-8")

        required_sections = [
            "Retained Human Gates & Operational Hold Ledger",
            "Source-Attribution Ledger",
            "Technical Synthetic-Result Inventory",
            "Explicitly Missing Operational Evidence Ledger",
            "Proposed-Unmeasured Thresholds Catalog",
            "Defect, Limitation, and Risk Reconciliation",
            "Learning Synthesis & Recommendations",
        ]
        for section in required_sections:
            self.assertIn(
                section, content, f"Required section '{section}' missing from document"
            )

    def test_04_explicitly_missing_evidence_domains(self):
        content = self.doc_path.read_text(encoding="utf-8")

        required_missing_domains = [
            "Human / User Evidence",
            "UAT Field Trial Evidence",
            "Operational Support Evidence",
            "Physical Mobile Device Evidence",
            "Live Network & Field Evidence",
            "Cloud Runtime Infrastructure",
            "Disaster Recovery & Backup",
        ]
        for domain in required_missing_domains:
            self.assertIn(
                domain, content, f"Missing evidence domain '{domain}' not in ledger"
            )

        # Assert MISSING / UNPROVEN status
        self.assertIn("MISSING / UNPROVEN", content)

    def test_05_proposed_unmeasured_thresholds(self):
        content = self.doc_path.read_text(encoding="utf-8")

        self.assertIn("[NON-BINDING_PROPOSED]", content)
        self.assertIn("[UNMEASURED]", content)

        # Confirm all NFR items represented
        for nfr in ["NFR-PERF-01", "NFR-PERF-02", "NFR-PERF-03", "NFR-SYNC-01", "NFR-CAP-01", "NFR-SLA-01"]:
            self.assertIn(nfr, content, f"Metric '{nfr}' missing from thresholds catalog")

    def test_06_recommendation_asserts_insufficient_evidence(self):
        content = self.doc_path.read_text(encoding="utf-8")
        content_upper = content.upper()

        # Formal recommendation must record insufficient evidence rather than acceptance
        self.assertIn("INSUFFICIENT EVIDENCE", content_upper)
        self.assertIn("DO NOT RELEASE", content_upper)
        self.assertIn("DO NOT ACCEPT RESIDUAL RISK", content_upper)

    def test_07_retained_foundation_holds_table(self):
        content = self.doc_path.read_text(encoding="utf-8")

        for hold_id in ["H040-007", "H040-008", "H040-009", "H040-010", "H040-011"]:
            self.assertIn(hold_id, content, f"Hold {hold_id} missing from body")

        # Table rows check
        self.assertIn("`H040-007` | Technical release authorization | **HOLD / BLOCKED**", content)
        self.assertIn("`H040-008` | Real participant, private-alpha, and UAT engagement | **HOLD / BLOCKED**", content)
        self.assertIn("`H040-009` | Binding support and manual-fallback operational ownership | **HOLD / BLOCKED**", content)
        self.assertIn("`H040-010` | External environment, device, account, route, storage, and notification activation | **HOLD / BLOCKED**", content)
        self.assertIn("`H040-011` | Final outcome, residual-risk acceptance, and v0.5.0 entry decision | **HOLD / BLOCKED**", content)

        # Zero authority grant statement
        self.assertIn(
            "Zero authority is granted to alter, bypass, or lift any hold",
            content,
        )


if __name__ == "__main__":
    unittest.main()
