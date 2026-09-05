import os
import re
import unittest
import yaml

class TestV040ScoringCalculationBaseline(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.doc_path = os.path.join(
            os.path.dirname(os.path.dirname(os.path.abspath(__file__))),
            "docs",
            "architecture",
            "v040-scoring-calculation-baseline.md"
        )
        if not os.path.exists(cls.doc_path):
            raise FileNotFoundError(f"Scoring calculation baseline not found at {cls.doc_path}")

        with open(cls.doc_path, "r", encoding="utf-8") as f:
            cls.content = f.read()

        # Extract frontmatter
        fm_match = re.match(r"^---\s*\n(.*?)\n---\s*\n", cls.content, re.DOTALL)
        if not fm_match:
            raise ValueError("Frontmatter not found in scoring calculation baseline")
        cls.frontmatter = yaml.safe_load(fm_match.group(1))

        # Extract synthetic scoring fixture YAML
        yaml_blocks = re.findall(r"```yaml\s*\n(.*?)\n```", cls.content, re.DOTALL)
        cls.fixture = None
        for block in yaml_blocks:
            try:
                parsed = yaml.safe_load(block)
                if isinstance(parsed, dict) and "fixture_id" in parsed and "scenarios" in parsed:
                    cls.fixture = parsed
                    break
            except Exception:
                continue

    def test_document_metadata_and_frontmatter(self):
        """Validates document metadata, identifiers, and governance gates in YAML frontmatter."""
        fm = self.frontmatter
        self.assertEqual(fm.get("document_id"), "ARC-V040-SCORING-001")
        self.assertEqual(fm.get("governing_issue"), "GitHub Issue #137")
        self.assertEqual(fm.get("authority_source"), "HDEC-V040-FOUNDATION-054")
        self.assertEqual(fm.get("milestone"), "v0.4.0 - OSHE Inspect Private Alpha")
        self.assertEqual(fm.get("lifecycle_status"), "DRAFT")
        self.assertEqual(fm.get("status"), "APPROVED_FOR_LOCAL_DEVELOPMENT")

        # Approved scoring rules under HDEC-V040-SCORING-058
        rules = fm.get("approved_scoring_rules", {})
        self.assertEqual(rules.get("selected_scoring_model"), "MODEL_2_WEIGHTED")
        self.assertEqual(rules.get("selected_unknown_handling"), "U1_QUARANTINE_DENOMINATOR")
        self.assertEqual(rules.get("selected_rounding_rule"), "R1_ROUND_HALF_UP")
        self.assertEqual(rules.get("selected_critical_fail_policy"), "CF1_PRIORITY_FLAG")
        self.assertEqual(rules.get("selected_passing_threshold_percent"), 80)

        # Pass predicates
        predicates = fm.get("pass_predicates", [])
        self.assertEqual(len(predicates), 3)

        # Foundation gates H040-001 through H040-006 must be listed
        approved_gates = fm.get("approved_foundation_gates", [])
        for g in ["H040-001", "H040-002", "H040-003", "H040-004", "H040-005", "H040-006"]:
            self.assertIn(g, approved_gates, f"Missing approved gate {g}")

        # Retained holds H040-007 through H040-011 must be listed
        retained_holds = fm.get("retained_holds", [])
        for h in ["H040-007", "H040-008", "H040-009", "H040-010", "H040-011"]:
            self.assertIn(h, retained_holds, f"Missing retained hold {h}")

        # Unselected human-owned policies
        unselected = fm.get("retained_unselected_policies", {})
        self.assertEqual(unselected.get("finding_closure_policy"), "HUMAN_OWNED_UNSELECTED")
        self.assertEqual(unselected.get("offline_authority"), "HUMAN_OWNED_UNSELECTED")

    def test_approved_scoring_rules_documentation(self):
        """Validates that all five owner-approved scoring rules are documented."""
        self.assertIn("MODEL_2_WEIGHTED", self.content)
        self.assertIn("U1_QUARANTINE_DENOMINATOR", self.content)
        self.assertIn("R1_ROUND_HALF_UP", self.content)
        self.assertIn("CF1_PRIORITY_FLAG", self.content)
        self.assertIn("80.00%", self.content)

    def test_mandatory_pass_predicates_documentation(self):
        """Validates that all three mandatory pass predicates are documented."""
        self.assertIn("No critical-fail condition is present", self.content)
        self.assertIn("No UNKNOWN response remains unresolved or quarantined", self.content)
        self.assertIn("Score is greater than or equal to 80.00 percent", self.content)

    def test_mathematical_formulation_and_weight_redistribution(self):
        """Validates Model 2 mathematical formula and dynamic weight redistribution."""
        self.assertIn("EffectiveDenominator", self.content)
        self.assertIn("Dynamic Section Weight Redistribution", self.content)
        self.assertIn("Zero-Division Defense", self.content)

    def test_exclusions_and_na_semantics(self):
        """Validates non-scored exclusions and NA denominator subtraction."""
        self.assertIn("Non-Scored Question Types", self.content)
        self.assertIn("TEXT_NOTE", self.content)
        self.assertIn("EVIDENCE_ATTACHMENT", self.content)
        self.assertIn("zero negative score impact", self.content)

    def test_basis_points_rounding_arithmetic(self):
        """Validates integer basis points and half-up rounding boundary proofs."""
        self.assertIn("10000", self.content)
        self.assertIn("8000", self.content)
        self.assertIn("79.995%", self.content)
        self.assertIn("79.994%", self.content)

    def test_traceability_and_non_authority_projection(self):
        """Validates pinned version traceability and non-authoritative reporting projection."""
        self.assertIn("v0.4.0-HDEC-058", self.content)
        self.assertIn("TraceabilityKey", self.content)
        self.assertIn("DERIVED_OUTPUT_NON_AUTHORITY", self.content)
        self.assertIn("MOD-REP", self.content)
        self.assertIn("MOD-WFA", self.content)

    def test_synthetic_operations_fixtures(self):
        """Validates synthetic scoring fixture YAML structure and scenarios."""
        self.assertIsNotNone(self.fixture, "Synthetic scoring fixture YAML block not found")
        fix = self.fixture

        self.assertEqual(fix.get("fixture_id"), "fix_syn_scoring_evaluation_v1")
        scenarios = fix.get("scenarios", [])
        self.assertEqual(len(scenarios), 4, "Fixture must define exactly 4 scenarios")

        # Scenario 1: Standard Passing Evaluation
        s1 = scenarios[0]
        self.assertEqual(s1.get("scenario_id"), "scen_score_pass_01")
        self.assertEqual(s1.get("outcome"), "PASS")
        self.assertEqual(s1.get("basis_points"), 8400)
        self.assertTrue(s1.get("pass_predicates", {}).get("all_predicates_satisfied"))

        # Scenario 2: Critical Fail Priority Flag
        s2 = scenarios[1]
        self.assertEqual(s2.get("scenario_id"), "scen_score_crit_02")
        self.assertEqual(s2.get("outcome"), "NON_COMPLIANT_CRITICAL")
        self.assertFalse(s2.get("score_masked"))

        # Scenario 3: Unknown Quarantine
        s3 = scenarios[2]
        self.assertEqual(s3.get("scenario_id"), "scen_score_unknown_03")
        self.assertEqual(s3.get("outcome"), "PROVISIONAL_PENDING_UNKNOWN_RESOLUTION")

        # Scenario 4: R1 Round Half Up Boundary
        s4 = scenarios[3]
        self.assertEqual(s4.get("scenario_id"), "scen_score_round_up_04")
        self.assertEqual(s4.get("expected_basis_points"), 8000)
        self.assertEqual(s4.get("outcome"), "PASS")

if __name__ == "__main__":
    unittest.main()
