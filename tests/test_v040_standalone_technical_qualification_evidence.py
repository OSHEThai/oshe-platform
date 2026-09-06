#!/usr/bin/env python3
"""
test_v040_standalone_technical_qualification_evidence.py
Automated validation suite for v0.4.0 Standalone Technical Qualification Evidence.
Governed by Issue #143 / ASN-V040-I032-TECHNICAL-QUALIFICATION-002.

Verifies:
1. Technical qualification evidence document exists and has non-trivial length.
2. Frontmatter metadata conforms to governance requirements (document_id, governing_issue, retained_holds).
3. Explicit non-claims and unproven declarations: user, device, browser, runtime, H040-008.
4. Retained foundation holds (H040-007 .. H040-011) remain strictly on HOLD with zero authority grant.
5. All five Go module suites are documented with PASS results.
6. Source-to-test mapping points to real existing files in the worktree.
7. Core technical invariants are documented in the traceability matrix.
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


class TestV040StandaloneTechnicalQualificationEvidence(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.repo_root = get_repo_root()
        cls.evidence_path = (
            cls.repo_root
            / "docs"
            / "architecture"
            / "v040-standalone-technical-qualification-evidence.md"
        )

    def test_01_evidence_document_exists(self):
        self.assertTrue(
            self.evidence_path.is_file(),
            f"Evidence document missing at: {self.evidence_path}",
        )
        content = self.evidence_path.read_text(encoding="utf-8")
        self.assertGreater(
            len(content), 1500, "Evidence document content is unexpectedly short"
        )

    def test_02_frontmatter_metadata(self):
        content = self.evidence_path.read_text(encoding="utf-8")
        frontmatter_match = re.search(r"^---\s*\n(.*?)\n---", content, re.DOTALL)
        self.assertIsNotNone(
            frontmatter_match, "YAML frontmatter block not found in evidence document"
        )
        fm_text = frontmatter_match.group(1)

        self.assertIn("document_id: EV-V040-TECH-001", fm_text)
        self.assertIn("governing_issue: 143", fm_text)
        self.assertIn(
            "assignment_id: ASN-V040-I032-TECHNICAL-QUALIFICATION-002", fm_text
        )
        self.assertIn(
            "lease_id: LEASE-V040-I032-TECHNICAL-QUALIFICATION-002", fm_text
        )
        self.assertIn("status: APPROVED", fm_text)
        self.assertIn("lifecycle: APPROVED", fm_text)
        self.assertIn("target_milestone: v0.4.0", fm_text)

        # Retained holds H040-007 through H040-011 in frontmatter
        for hold_id in ["H040-007", "H040-008", "H040-009", "H040-010", "H040-011"]:
            self.assertIn(
                hold_id, fm_text, f"Retained hold {hold_id} missing from frontmatter"
            )

    def test_03_explicit_unproven_and_non_claims(self):
        content = self.evidence_path.read_text(encoding="utf-8")
        content_lower = content.lower()

        # Must explicitly declare user, device, browser, runtime evidence unproven
        self.assertIn("user evidence remains unproven", content_lower)
        self.assertIn("device evidence remains unproven", content_lower)
        self.assertIn("browser evidence remains unproven", content_lower)
        self.assertIn("runtime evidence remains unproven", content_lower)
        self.assertIn("h040-008 remains unproven", content_lower)

        # Zero authority grant statement
        self.assertIn(
            "zero authority is granted to alter, bypass, or lift any hold",
            content_lower,
        )

    def test_04_retained_foundation_holds_table(self):
        content = self.evidence_path.read_text(encoding="utf-8")

        # All 5 foundation holds must be present and marked HOLD
        for hold_id in ["H040-007", "H040-008", "H040-009", "H040-010", "H040-011"]:
            self.assertIn(hold_id, content, f"Hold {hold_id} missing from document body")

        # Table rows check
        self.assertIn("`H040-007` | Technical release authorization | **HOLD**", content)
        self.assertIn(
            "`H040-008` | Real participant, private-alpha, and UAT engagement | **HOLD**",
            content,
        )
        self.assertIn(
            "`H040-009` | Binding support and manual-fallback operational ownership | **HOLD**",
            content,
        )
        self.assertIn(
            "`H040-010` | External environment, device, account, route, storage, and notification activation | **HOLD**",
            content,
        )
        self.assertIn(
            "`H040-011` | Final outcome, residual-risk acceptance, and v0.5.0 entry decision | **HOLD**",
            content,
        )

    def test_05_five_module_suites_pass_results(self):
        content = self.evidence_path.read_text(encoding="utf-8")

        expected_modules = [
            "modules/internal-portal",
            "modules/identity-authorization",
            "modules/reporting-localization",
            "modules/records-audit",
            "modules/workflow-action",
        ]
        for mod in expected_modules:
            self.assertIn(mod, content, f"Module {mod} not documented in evidence")

        # Verify PASS results documented
        self.assertIn("5 of 5 suites passed. 0 suites failed.", content)

    def test_06_source_mapping_files_exist_on_disk(self):
        # Verify that key mapped files referenced in the evidence actually exist on disk
        mapped_files = [
            # internal-portal
            self.repo_root / "modules" / "internal-portal" / "portal.go",
            self.repo_root / "modules" / "internal-portal" / "portal_test.go",
            self.repo_root / "modules" / "internal-portal" / "inspect_composition.go",
            # identity-authorization
            self.repo_root / "modules" / "identity-authorization" / "access_policy.go",
            self.repo_root
            / "modules"
            / "identity-authorization"
            / "authorization_qualification_test.go",
            # reporting-localization
            self.repo_root / "modules" / "reporting-localization" / "reporting.go",
            self.repo_root
            / "modules"
            / "reporting-localization"
            / "reporting_qualification_test.go",
            self.repo_root
            / "modules"
            / "reporting-localization"
            / "localization_qualification_test.go",
            # records-audit
            self.repo_root / "modules" / "records-audit" / "record_audit.go",
            self.repo_root / "modules" / "records-audit" / "immutable_objects.go",
            # workflow-action
            self.repo_root / "modules" / "workflow-action" / "scoring.go",
            self.repo_root
            / "modules"
            / "workflow-action"
            / "scoring_qualification_test.go",
            self.repo_root
            / "modules"
            / "workflow-action"
            / "fail_closed_governance.go",
        ]
        for f in mapped_files:
            self.assertTrue(f.is_file(), f"Mapped source file does not exist on disk: {f}")

    def test_07_core_invariants_and_tokens_documented(self):
        content = self.evidence_path.read_text(encoding="utf-8")
        required_tokens = [
            "DERIVED_OUTPUT_NON_AUTHORITY",
            "R1_ROUND_HALF_UP",
            "ErrMustDefaultOff",
            "DefaultOff",
            "Buddhist Era",
            "SHA-256",
            "8000 bps",
        ]
        for token in required_tokens:
            self.assertIn(
                token, content, f"Required invariant token missing from document: {token}"
            )


if __name__ == "__main__":
    unittest.main()
