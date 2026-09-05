import os
import re
import unittest
import yaml

class TestV040ChecklistModelBaseline(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.doc_path = os.path.join(
            os.path.dirname(os.path.dirname(os.path.abspath(__file__))),
            "docs",
            "architecture",
            "v040-checklist-model-baseline.md"
        )
        if not os.path.exists(cls.doc_path):
            raise FileNotFoundError(f"Checklist model baseline not found at {cls.doc_path}")

        with open(cls.doc_path, "r", encoding="utf-8") as f:
            cls.content = f.read()

        # Extract frontmatter
        fm_match = re.match(r"^---\s*\n(.*?)\n---\s*\n", cls.content, re.DOTALL)
        if not fm_match:
            raise ValueError("Frontmatter not found in checklist model baseline")
        cls.frontmatter = yaml.safe_load(fm_match.group(1))

        # Extract synthetic pilot checklist fixture YAML
        yaml_blocks = re.findall(r"```yaml\s*\n(.*?)\n```", cls.content, re.DOTALL)
        cls.fixture = None
        for block in yaml_blocks:
            try:
                parsed = yaml.safe_load(block)
                if isinstance(parsed, dict) and "checklist_id" in parsed:
                    cls.fixture = parsed
                    break
            except Exception:
                continue

    def test_document_existence_and_frontmatter(self):
        """Validates document metadata, identifiers, and governance gates in YAML frontmatter."""
        fm = self.frontmatter
        self.assertEqual(fm.get("document_id"), "ARC-V040-CHKL-001")
        self.assertEqual(fm.get("governing_issue"), "GitHub Issue #116")
        self.assertEqual(fm.get("authority_source"), "HDEC-V040-FOUNDATION-054")
        self.assertEqual(fm.get("milestone"), "v0.4.0 - OSHE Inspect Private Alpha")
        self.assertEqual(fm.get("lifecycle_status"), "DRAFT")
        self.assertEqual(fm.get("status"), "APPROVED_FOR_LOCAL_DEVELOPMENT")

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
        self.assertEqual(unselected.get("binding_scoring_policy"), "HUMAN_OWNED_UNSELECTED")
        self.assertEqual(unselected.get("finding_closure_policy"), "HUMAN_OWNED_UNSELECTED")
        self.assertEqual(unselected.get("offline_authority"), "HUMAN_OWNED_UNSELECTED")

    def test_supported_alpha_question_types(self):
        """Validates that all six supported alpha question types are documented."""
        expected_types = [
            "PASS_FAIL_NA_UNKNOWN",
            "SINGLE_CHOICE",
            "MULTI_CHOICE",
            "NUMERIC_MEASUREMENT",
            "TEXT_NOTE",
            "EVIDENCE_ATTACHMENT",
        ]
        for q_type in expected_types:
            self.assertIn(q_type, self.content, f"Missing question type definition for {q_type}")

        # H040-006 explicit UNKNOWN and NA compliance
        self.assertIn("UNKNOWN", self.content, "Missing UNKNOWN response handling per H040-006")
        self.assertIn("PASS_FAIL_NA_UNKNOWN", self.content)

    def test_unsupported_generalized_form_declarations(self):
        """Validates explicit declarations excluding no-code form builders and dynamic code execution."""
        self.assertIn("UNSUPPORTED_PRIVATE_ALPHA", self.content)
        self.assertIn("No Generalized No-Code Form Builder Platform", self.content)
        self.assertIn("No Dynamic Client-Side Code Execution", self.content)
        self.assertIn("No Unconstrained Recursive Sub-Forms", self.content)
        self.assertIn("No External Live API Lookup Questions", self.content)

    def test_synthetic_pilot_checklist_fixture(self):
        """Validates that the synthetic pilot plant checklist fixture is well-formed and valid."""
        self.assertIsNotNone(self.fixture, "Synthetic pilot checklist fixture YAML block not found")
        fix = self.fixture

        # Core metadata
        self.assertTrue(fix.get("checklist_id", "").startswith("chk_syn_"))
        self.assertTrue(fix.get("tenant_id", "").startswith("ten_"))
        self.assertEqual(fix.get("version"), "1.0.0")
        self.assertEqual(fix.get("lifecycle_status"), "PUBLISHED_IMMUTABLE")
        self.assertEqual(fix.get("owner_role"), "Checklist Author")
        self.assertTrue(fix.get("is_pilot"))

        sections = fix.get("sections", [])
        self.assertGreaterEqual(len(sections), 3, "Pilot checklist must contain at least 3 sections")

        # Inventory all question types in fixture
        observed_types = set()
        question_count = 0

        for sec in sections:
            self.assertTrue(sec.get("section_id", "").startswith("sec_syn_"))
            self.assertIn("en-US", sec.get("title", {}))
            self.assertIn("th-TH", sec.get("title", {}))

            for q in sec.get("questions", []):
                question_count += 1
                self.assertTrue(q.get("question_id", "").startswith("qst_syn_"))
                self.assertTrue(bool(q.get("code")))
                q_type = q.get("question_type")
                observed_types.add(q_type)
                self.assertIn("en-US", q.get("prompt", {}))
                self.assertIn("th-TH", q.get("prompt", {}))

                # If numeric, must have unit
                if q_type == "NUMERIC_MEASUREMENT":
                    self.assertIn("numeric_spec", q)
                    self.assertTrue(bool(q["numeric_spec"].get("unit")))

                # If choice, must have options with localized labels
                if q_type in ("SINGLE_CHOICE", "MULTI_CHOICE"):
                    options = q.get("options", [])
                    self.assertGreaterEqual(len(options), 2)
                    for opt in options:
                        self.assertIn("en-US", opt.get("label", {}))
                        self.assertIn("th-TH", opt.get("label", {}))

        self.assertGreaterEqual(question_count, 6, "Pilot checklist must contain at least 6 questions")
        # Assert comprehensive question type coverage in fixture
        for exp in ["PASS_FAIL_NA_UNKNOWN", "SINGLE_CHOICE", "MULTI_CHOICE", "NUMERIC_MEASUREMENT", "TEXT_NOTE", "EVIDENCE_ATTACHMENT"]:
            self.assertIn(exp, observed_types, f"Question type {exp} not exercised in synthetic fixture")

    def test_bilingual_localization_coverage(self):
        """Validates that English (en-US) and Thai (th-TH) translations exist for all checklist elements."""
        fix = self.fixture
        self.assertIsNotNone(fix)

        # Title and Description
        self.assertTrue(bool(fix["title"].get("en-US")))
        self.assertTrue(bool(fix["title"].get("th-TH")))
        self.assertTrue(bool(fix["description"].get("en-US")))
        self.assertTrue(bool(fix["description"].get("th-TH")))

        for sec in fix["sections"]:
            self.assertTrue(bool(sec["title"].get("en-US")))
            self.assertTrue(bool(sec["title"].get("th-TH")))
            for q in sec["questions"]:
                self.assertTrue(bool(q["prompt"].get("en-US")))
                self.assertTrue(bool(q["prompt"].get("th-TH")))
                if "guidance" in q:
                    self.assertTrue(bool(q["guidance"].get("en-US")))
                    self.assertTrue(bool(q["guidance"].get("th-TH")))

    def test_governance_boundaries_and_non_claims(self):
        """Validates non-binding scoring declarations, default-deny, and synthetic-only constraints."""
        # Non-binding scoring declaration
        self.assertIn("HUMAN_OWNED_UNSELECTED", self.content)
        self.assertIn("PROVISIONAL_WEIGHTED_SCORE_V1", self.content)

        # Default-deny and role governance
        self.assertIn("H040-004", self.content)
        self.assertIn("Checklist Author", self.content)
        self.assertIn("Independent Reviewer", self.content)

        # Synthetic data mandate
        self.assertIn("H040-003", self.content)
        self.assertIn("chk_syn_", self.content)

        # Prohibitions on live deployment
        self.assertIn("H040-007", self.content)
        self.assertIn("H040-008", self.content)
        self.assertIn("HOLD", self.content)

if __name__ == "__main__":
    unittest.main()
