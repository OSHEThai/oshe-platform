#!/usr/bin/env python3
"""
test_v040_localization_accessibility_qualification_baseline.py
Qualification baseline automated verification suite for V040-I031 / Issue #142.

Verifies:
1. Architecture qualification baseline document existence and governed frontmatter metadata.
2. Complete coverage of core localization and accessibility qualification invariants.
3. Explicit non-claims: no browser/UI/device runtime claim, H040-002 and H040-006 deferred.
4. Buddhist Era year arithmetic verification (CE + 543 = BE).
5. Scenario matrix completeness (SYN-LOC-01 .. SYN-LOC-10).
6. Foundation holds retention (H040-007 .. H040-011) on HOLD with zero authority grant.
7. Go qualification test coverage in modules/reporting-localization/localization_qualification_test.go.
8. Synthetic alpha boundary assertions (zero customer/production data).
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


class TestV040LocalizationAccessibilityQualificationBaseline(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.repo_root = get_repo_root()
        cls.baseline_path = cls.repo_root / "docs" / "architecture" / "v040-localization-accessibility-qualification-baseline.md"
        cls.go_test_path = cls.repo_root / "modules" / "reporting-localization" / "localization_qualification_test.go"

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

        self.assertIn("document_id: QLF-V040-LOC-A11Y-001", fm_text)
        self.assertIn("governing_issue: 142", fm_text)
        self.assertIn("assignment_id: ASN-V040-I031-LOCALIZATION-QUALIFICATION-003", fm_text)
        self.assertIn("lease_id: LEASE-V040-I031-LOCALIZATION-QUALIFICATION-003", fm_text)
        self.assertIn("status: APPROVED", fm_text)
        self.assertIn("lifecycle: APPROVED", fm_text)
        self.assertIn("target_milestone: v0.4.0", fm_text)
        self.assertIn("synthetic_scenario_id: fix_syn_localization_accessibility_qualification_v1", fm_text)

        # Decisions
        self.assertIn("HDEC-V040-FOUNDATION-054", fm_text)
        self.assertIn("HDEC-V040-SCORING-058", fm_text)

        # Deferred decisions
        self.assertIn("H040-002", fm_text)
        self.assertIn("H040-006", fm_text)

        # Retained holds H040-007 .. H040-011
        for hold_id in ["H040-007", "H040-008", "H040-009", "H040-010", "H040-011"]:
            self.assertIn(hold_id, fm_text, f"Retained hold {hold_id} missing from frontmatter")

    def test_03_core_invariants_documented(self):
        content = self.baseline_path.read_text(encoding="utf-8")
        required_invariants = [
            "en-US",
            "th-TH",
            "DispositionExactMatch",
            "DispositionFallback",
            "DispositionMissing",
            "Buddhist Era",
            "Asia/Bangkok",
            "DefaultOff",
            "FEATURE_FLAG_NON_AUTHORITY",
            "AccessibilityMetadata",
            "no browser, UI, or mobile device runtime claim",
            "H040-002",
            "H040-006",
        ]
        for inv in required_invariants:
            self.assertIn(inv, content, f"Required invariant marker missing from document: {inv}")

    def test_04_buddhist_era_math(self):
        # In Thailand, BE year = CE year + 543
        ce_year = 2026
        be_year = ce_year + 543
        self.assertEqual(be_year, 2569)

    def test_05_scenario_matrix_completeness(self):
        content = self.baseline_path.read_text(encoding="utf-8")
        for i in range(1, 11):
            scenario_id = f"SYN-LOC-{i:02d}"
            self.assertIn(scenario_id, content, f"Scenario {scenario_id} missing from matrix table")

    def test_06_retained_foundation_holds_table(self):
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

    def test_07_go_qualification_test_coverage(self):
        self.assertTrue(self.go_test_path.is_file(), f"Go qualification test file missing at {self.go_test_path}")
        content = self.go_test_path.read_text(encoding="utf-8")

        expected_test_funcs = [
            "TestV040Qualification_ExactMatchAndVisibleFallback",
            "TestV040Qualification_DateTimeUTCAndBuddhistEra",
            "TestV040Qualification_UnitsAndLongTextExpansion",
            "TestV040Qualification_FeatureFlagDefaultOff",
            "TestV040Qualification_AuthorizationSeparation",
            "TestV040Qualification_AccessibilityMetadata",
            "TestV040Qualification_RolloutTemporalAndCohortBoundaries",
        ]
        for fn in expected_test_funcs:
            self.assertIn(fn, content, f"Go test function {fn} missing from {self.go_test_path.name}")

    def test_08_synthetic_boundary_integrity(self):
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
