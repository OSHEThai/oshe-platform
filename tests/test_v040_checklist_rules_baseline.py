import os
import re
import unittest
import yaml

class TestV040ChecklistRulesBaseline(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.doc_path = os.path.join(
            os.path.dirname(os.path.dirname(os.path.abspath(__file__))),
            "docs",
            "architecture",
            "v040-checklist-rules-baseline.md"
        )
        if not os.path.exists(cls.doc_path):
            raise FileNotFoundError(f"Checklist rules baseline not found at {cls.doc_path}")

        with open(cls.doc_path, "r", encoding="utf-8") as f:
            cls.content = f.read()

        # Extract frontmatter
        fm_match = re.match(r"^---\s*\n(.*?)\n---\s*\n", cls.content, re.DOTALL)
        if not fm_match:
            raise ValueError("Frontmatter not found in checklist rules baseline")
        cls.frontmatter = yaml.safe_load(fm_match.group(1))

        # Extract synthetic rules fixture YAML
        yaml_blocks = re.findall(r"```yaml\s*\n(.*?)\n```", cls.content, re.DOTALL)
        cls.fixture = None
        for block in yaml_blocks:
            try:
                parsed = yaml.safe_load(block)
                if isinstance(parsed, dict) and "fixture_id" in parsed:
                    cls.fixture = parsed
                    break
            except Exception:
                continue

    def test_document_metadata_and_frontmatter(self):
        """Validates document metadata, identifiers, and governance gates in YAML frontmatter."""
        fm = self.frontmatter
        self.assertEqual(fm.get("document_id"), "ARC-V040-CHKRULE-001")
        self.assertEqual(fm.get("governing_issue"), "GitHub Issue #117")
        self.assertEqual(fm.get("authority_source"), "HDEC-V040-FOUNDATION-054")
        self.assertEqual(fm.get("milestone"), "v0.4.0 - OSHE Inspect Private Alpha")
        self.assertEqual(fm.get("lifecycle_status"), "DRAFT")
        self.assertEqual(fm.get("status"), "APPROVED_FOR_LOCAL_DEVELOPMENT")

        # Approved foundation gates H040-001 through H040-006
        approved_gates = fm.get("approved_foundation_gates", [])
        for g in ["H040-001", "H040-002", "H040-003", "H040-004", "H040-005", "H040-006"]:
            self.assertIn(g, approved_gates, f"Missing approved gate {g}")

        # Retained holds H040-007 through H040-011
        retained_holds = fm.get("retained_holds", [])
        for h in ["H040-007", "H040-008", "H040-009", "H040-010", "H040-011"]:
            self.assertIn(h, retained_holds, f"Missing retained hold {h}")

        # Unselected human-owned policies
        unselected = fm.get("retained_unselected_policies", {})
        self.assertEqual(unselected.get("binding_scoring_policy"), "HUMAN_OWNED_UNSELECTED")
        self.assertEqual(unselected.get("finding_closure_policy"), "HUMAN_OWNED_UNSELECTED")
        self.assertEqual(unselected.get("offline_authority"), "HUMAN_OWNED_UNSELECTED")

    def test_deterministic_rule_model_predicates_and_operators(self):
        """Validates defined predicates, operators, and topological DAG evaluation."""
        self.assertIn("METADATA_MATCH", self.content)
        self.assertIn("PRIOR_ANSWER_MATCH", self.content)

        for op in ["EQUALS", "NOT_EQUALS", "IN", "NOT_IN", "GREATER_THAN", "LESS_THAN"]:
            self.assertIn(op, self.content, f"Missing operator definition for {op}")

        # Topological ordering & DAG requirements
        self.assertIn("Topological Ordering", self.content)
        self.assertIn("Acyclic Dependency DAG", self.content)

    def test_response_validation_rules_all_six_types(self):
        """Validates response validation rules across all six supported question types."""
        expected_types = [
            "PASS_FAIL_NA_UNKNOWN",
            "SINGLE_CHOICE",
            "MULTI_CHOICE",
            "NUMERIC_MEASUREMENT",
            "TEXT_NOTE",
            "EVIDENCE_ATTACHMENT",
        ]
        for q_type in expected_types:
            self.assertIn(q_type, self.content, f"Missing validation rule for {q_type}")

        # Mandatory justification for NA and UNKNOWN
        self.assertIn("Mandatory Justification for `NA`", self.content)
        self.assertIn("Mandatory Justification for `UNKNOWN`", self.content)

    def test_evidence_policies_and_submission_blocking(self):
        """Validates evidence policies (OPTIONAL, MANDATORY_ON_FAIL, MANDATORY_ALWAYS) and submission blocking."""
        self.assertIn("MANDATORY_ON_FAIL", self.content)
        self.assertIn("MANDATORY_ALWAYS", self.content)
        self.assertIn("OPTIONAL", self.content)
        self.assertIn("ErrMandatoryEvidenceMissing", self.content)

    def test_question_and_section_exclusion_rules(self):
        """Validates exclusion rules, denominator exclusion, and zero negative scoring impact."""
        self.assertIn("EXCLUDED_BY_RULE", self.content)
        self.assertIn("Exclusion from Denominators", self.content)

    def test_scoring_reference_provisional_contract(self):
        """Validates that scoring is declared non-binding, provisional, and HUMAN_OWNED_UNSELECTED."""
        self.assertIn("HUMAN_OWNED_UNSELECTED", self.content)
        self.assertIn("PROVISIONAL_WEIGHTED_SCORE_V1", self.content)
        self.assertIn("Zero Legal or Regulatory Claims", self.content)

    def test_safe_failure_modes_and_fail_closed_defaults(self):
        """Validates fail-closed failure modes: missing metadata defaults to SHOW, conflict quarantine."""
        self.assertIn("Defaults to **`SHOW`**", self.content)
        self.assertIn("H040-005", self.content)
        self.assertIn("H040-004", self.content)

    def test_synthetic_rules_fixture(self):
        """Validates the synthetic conditional rules fixture YAML structure."""
        self.assertIsNotNone(self.fixture, "Synthetic rules fixture YAML block not found")
        fix = self.fixture

        self.assertEqual(fix.get("fixture_id"), "fix_syn_rules_pilot_v1")
        self.assertEqual(fix.get("template_id"), "chk_syn_pilot_plant_safety_v1")
        self.assertEqual(fix.get("template_version"), "1.0.0")

        rules = fix.get("rules", [])
        self.assertGreaterEqual(len(rules), 3, "Fixture must define at least 3 rules")

        observed_actions = set()
        for r in rules:
            self.assertTrue(r.get("rule_id", "").startswith("rule_syn_"))
            self.assertIn(r.get("predicate_type"), ["METADATA_MATCH", "PRIOR_ANSWER_MATCH"])
            self.assertIn("en-US", r.get("description", {}))
            self.assertIn("th-TH", r.get("description", {}))
            observed_actions.add(r.get("action"))

        self.assertIn("SHOW", observed_actions)
        self.assertIn("TRIGGER_FINDING", observed_actions)

    def test_template_versioning_and_immutability(self):
        """Validates immutability boundaries and active inspection execution pinning."""
        self.assertIn("PUBLISHED_IMMUTABLE", self.content)
        self.assertIn("Execution Pinning", self.content)

if __name__ == "__main__":
    unittest.main()
