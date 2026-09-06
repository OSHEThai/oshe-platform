#!/usr/bin/env python3
"""
test_v040_controlled_reporting_qualification_baseline.py
Qualification baseline automated verification suite for V040-I030 / Issue #141.

Verifies:
1. Architecture qualification baseline document existence and governed frontmatter metadata.
2. Complete coverage of core reporting qualification invariants.
3. Scenario matrix completeness (SYN-REP-01 .. SYN-REP-12).
4. Foundation holds retention (H040-007 .. H040-011) on HOLD with zero authority grant.
5. Go qualification test coverage in modules/reporting-localization/reporting_qualification_test.go.
6. Synthetic alpha boundary assertions (zero customer/production data).
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


class TestV040ControlledReportingQualificationBaseline(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.repo_root = get_repo_root()
        cls.baseline_path = cls.repo_root / "docs" / "architecture" / "v040-controlled-reporting-qualification-baseline.md"
        cls.go_test_path = cls.repo_root / "modules" / "reporting-localization" / "reporting_qualification_test.go"

    def test_01_baseline_document_exists(self):
        self.assertTrue(
            self.baseline_path.is_file(),
            f"Qualification baseline document missing at: {self.baseline_path}",
        )
        content = self.baseline_path.read_text(encoding="utf-8")
        self.assertGreater(len(content), 1000, "Baseline document is unexpectedly short")

    def test_02_frontmatter_metadata(self):
        content = self.baseline_path.read_text(encoding="utf-8")
        frontmatter_match = re.search(r"^---\s*\n(.*?)\n---", content, re.DOTALL)
        self.assertIsNotNone(frontmatter_match, "YAML frontmatter block not found in baseline document")
        fm_text = frontmatter_match.group(1)

        self.assertIn("document_id: QLF-V040-REPORTING-001", fm_text)
        self.assertIn("governing_issue: 141", fm_text)
        self.assertIn("assignment_id: ASN-V040-I030-HOLD-MAPPING-CORRECTION-009", fm_text)
        self.assertIn("lease_id: LEASE-V040-I030-HOLD-MAPPING-CORRECTION-009", fm_text)
        self.assertIn("status: APPROVED", fm_text)
        self.assertIn("lifecycle: APPROVED", fm_text)
        self.assertIn("target_milestone: v0.4.0", fm_text)
        self.assertIn("synthetic_scenario_id: fix_syn_controlled_reporting_qualification_v1", fm_text)

        # Governing decisions
        self.assertIn("HDEC-V040-FOUNDATION-054", fm_text)
        self.assertIn("HDEC-V040-SCORING-058", fm_text)
        self.assertIn("HDEC-V040-I030-BOUNDED-REPOSITORY-DISCOVERY-001", fm_text)

        # Retained holds H040-007 .. H040-011
        for hold_id in ["H040-007", "H040-008", "H040-009", "H040-010", "H040-011"]:
            self.assertIn(hold_id, fm_text, f"Retained hold {hold_id} missing from frontmatter")

    def test_03_core_invariants_documented(self):
        content = self.baseline_path.read_text(encoding="utf-8")
        required_invariants = [
            "DERIVED_OUTPUT_NON_AUTHORITY",
            "ErrUnauthorizedReader",
            "ErrUnsupportedFilter",
            "ErrReportTampered",
            "ErrCrossTenantRecord",
            "ErrMissingNonAuthority",
            "SourceDataDigest",
            "RenderedDigest",
            "FreshnessDisposition",
        ]
        for inv in required_invariants:
            self.assertIn(inv, content, f"Required invariant marker missing from document: {inv}")

    def test_04_scenario_matrix_completeness(self):
        content = self.baseline_path.read_text(encoding="utf-8")
        for i in range(1, 13):
            scenario_id = f"SYN-REP-{i:02d}"
            self.assertIn(scenario_id, content, f"Scenario {scenario_id} missing from matrix table")

    def test_05_retained_foundation_holds_table(self):
        content = self.baseline_path.read_text(encoding="utf-8")
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

        # Verify every row remains HOLD and explicitly states no activation/authorization is granted
        for hold_id, _ in holds:
            row_match = re.search(rf"\|\s*\*\*{hold_id}\*\*\s*\|\s*([^|]+)\|\s*\*\*HOLD\*\*\s*\|\s*([^|\n]+)\|", content)
            self.assertIsNotNone(row_match, f"Row for {hold_id} must have status HOLD")
            enforcement = row_match.group(2)
            self.assertIn(
                "no activation/authorization is granted",
                enforcement,
                f"Row for {hold_id} must say 'no activation/authorization is granted'",
            )

    def test_06_go_qualification_test_coverage(self):
        self.assertTrue(self.go_test_path.is_file(), f"Go qualification test file missing at {self.go_test_path}")
        content = self.go_test_path.read_text(encoding="utf-8")

        expected_test_funcs = [
            "TestQualification_SYN_REP_01_MetricCalculationDeterminism",
            "TestQualification_SYN_REP_02_ScopedFiltersAndRejection",
            "TestQualification_SYN_REP_03_FreshnessBoundaries",
            "TestQualification_SYN_REP_04_CrossTenantReaderAuthorization",
            "TestQualification_SYN_REP_05_EmptyRecordExportRobustness",
            "TestQualification_SYN_REP_06_LargeRecordExportCompleteness",
            "TestQualification_SYN_REP_07_LongTextContentPreservation",
            "TestQualification_SYN_REP_08_TemplateVersionAndContextPinning",
            "TestQualification_SYN_REP_09_ExportManifestCryptographicHashing",
            "TestQualification_SYN_REP_10_TamperDetectionViaDigest",
            "TestQualification_SYN_REP_11_CrossTenantRecordExportFailure",
            "TestQualification_SYN_REP_12_NonAuthorityDeclarationEnforcement",
        ]
        for fn in expected_test_funcs:
            self.assertIn(fn, content, f"Go test function {fn} missing from {self.go_test_path.name}")

    def test_07_synthetic_boundary_integrity(self):
        forbidden_patterns = [
            r"prod[.-]api\.",
            r"customer[.-]secret",
            r"live[.-]bearer[.-]token",
            r"BEGIN (?:RSA )?PRIVATE KEY",
        ]
        for path in [self.baseline_path, self.go_test_path, Path(__file__)]:
            text = path.read_text(encoding="utf-8")
            for pat in forbidden_patterns:
                self.assertIsNone(
                    re.search(pat, text, re.IGNORECASE),
                    f"Forbidden customer/production pattern '{pat}' detected in {path.name}",
                )


if __name__ == "__main__":
    unittest.main()
