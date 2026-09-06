#!/usr/bin/env python3
"""
test_v040_scoring_safety_qualification_baseline.py
Qualification baseline automated verification suite for V040-I028 / Issue #139.

Verifies:
1. Architecture qualification baseline document existence and governed frontmatter metadata.
2. Complete coverage of core safety qualification invariants.
3. Deterministic basis-point rounding arithmetic (R1_ROUND_HALF_UP).
4. Scenario matrix completeness (SYN-SCORING-01 .. SYN-SCORING-14).
5. Foundation holds retention (H040-007 .. H040-011) on HOLD with zero authority grant.
6. Synthetic alpha boundary assertions (zero customer/production data).
"""

import math
import os
import re
import unittest
from pathlib import Path


def get_repo_root() -> Path:
    # Anchor to the worktree root
    current = Path(__file__).resolve().parent
    while current != current.parent:
        if (current / "docs").is_dir() and (current / "modules").is_dir():
            return current
        current = current.parent
    # Fallback to parent of tests directory
    return Path(__file__).resolve().parent.parent


class TestV040ScoringSafetyQualificationBaseline(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.repo_root = get_repo_root()
        cls.baseline_path = cls.repo_root / "docs" / "architecture" / "v040-scoring-safety-qualification-baseline.md"
        cls.go_test_path = cls.repo_root / "modules" / "workflow-action" / "scoring_qualification_test.go"

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

        self.assertIn("document_id: QLF-V040-SCORING-SAFETY-001", fm_text)
        self.assertIn("governing_issue: 139", fm_text)
        self.assertIn("assignment_id: ASN-V040-I028-SCORING-SAFETY-QUALIFICATION-002", fm_text)
        self.assertIn("lease_id: LEASE-V040-I028-SCORING-SAFETY-QUALIFICATION-002", fm_text)
        self.assertIn("status: APPROVED", fm_text)
        self.assertIn("lifecycle: APPROVED", fm_text)
        self.assertIn("target_milestone: v0.4.0", fm_text)
        self.assertIn("synthetic_scenario_id: fix_syn_scoring_safety_qualification_v1", fm_text)

        # Decisions
        self.assertIn("HDEC-V040-FOUNDATION-054", fm_text)
        self.assertIn("HDEC-V040-SCORING-058", fm_text)

        # Retained holds H040-007 .. H040-011
        for hold_id in ["H040-007", "H040-008", "H040-009", "H040-010", "H040-011"]:
            self.assertIn(hold_id, fm_text, f"Retained hold {hold_id} missing from frontmatter")

    def test_03_core_invariants_documented(self):
        content = self.baseline_path.read_text(encoding="utf-8")
        required_invariants = [
            "MODEL_2_WEIGHTED",
            "U1_QUARANTINE_DENOMINATOR",
            "R1_ROUND_HALF_UP",
            "CF1_PRIORITY_FLAG",
            "8000",
            "critical > UNKNOWN > score",
            "H040-004",
            "DERIVED_OUTPUT_NON_AUTHORITY",
            "PROVISIONAL_PENDING_UNKNOWN_RESOLUTION",
            "NON_COMPLIANT_CRITICAL",
        ]
        for inv in required_invariants:
            self.assertIn(inv, content, f"Required invariant marker missing from document: {inv}")

    def test_04_deterministic_rounding_math(self):
        # Basis points calculation formula:
        # basis_points = floor(raw_percent * 100 + 0.5 + 1e-9)
        def calc_bps(percent: float) -> int:
            return int(math.floor(percent * 100.0 + 0.5 + 1e-9))

        # 79.995% rounds half-up to 8000 bps (80.00%) -> PASS
        bps_up = calc_bps(79.995)
        self.assertEqual(bps_up, 8000)
        self.assertTrue(bps_up >= 8000, "79.995% must satisfy 8000 bps threshold")

        # 79.994% rounds down to 7999 bps (79.99%) -> FAIL
        bps_down = calc_bps(79.994)
        self.assertEqual(bps_down, 7999)
        self.assertFalse(bps_down >= 8000, "79.994% must not satisfy 8000 bps threshold")

        # Exact threshold
        bps_exact = calc_bps(80.000)
        self.assertEqual(bps_exact, 8000)

        # 80.005% -> 8001 bps
        bps_plus = calc_bps(80.005)
        self.assertEqual(bps_plus, 8001)

    def test_05_scenario_matrix_completeness(self):
        content = self.baseline_path.read_text(encoding="utf-8")
        for i in range(1, 15):
            scenario_id = f"SYN-SCORING-{i:02d}"
            self.assertIn(scenario_id, content, f"Scenario {scenario_id} missing from matrix table")

    def test_06_retained_foundation_holds_table(self):
        content = self.baseline_path.read_text(encoding="utf-8")
        holds = [
            ("H040-007", "External Production Deployment"),
            ("H040-008", "Live Third-Party Integrations"),
            ("H040-009", "Commercial Licensing & Payment Gateways"),
            ("H040-010", "Automated Destructive Maintenance"),
            ("H040-011", "Autonomous Human Decision Delegation"),
        ]
        for hold_id, subject in holds:
            self.assertIn(hold_id, content, f"Hold {hold_id} missing")
            self.assertIn(subject, content, f"Hold subject '{subject}' missing")

    def test_07_go_qualification_test_coverage(self):
        self.assertTrue(self.go_test_path.is_file(), f"Go qualification test file missing at {self.go_test_path}")
        content = self.go_test_path.read_text(encoding="utf-8")

        expected_test_funcs = [
            "TestQualification_BoundaryRoundingExactBasisPoints",
            "TestQualification_NAAndExclusions_DenominatorSubtraction",
            "TestQualification_UnknownQuarantine_ProvisionalOutcome",
            "TestQualification_RuleVersionTraceability",
            "TestQualification_CriticalFailPriorityHierarchy",
            "TestQualification_DeferredManualOverrideDenial",
            "TestQualification_AutonomousAIBoundaryDenial",
            "TestQualification_AuthorizedHumanResolutionPath",
            "TestQualification_DynamicWeightRedistribution",
            "TestQualification_AuditLedgerMonotonicity",
        ]
        for fn in expected_test_funcs:
            self.assertIn(fn, content, f"Go test function {fn} missing from {self.go_test_path.name}")

    def test_08_synthetic_boundary_integrity(self):
        # Assert no customer or production data markers exist in baseline or tests
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
