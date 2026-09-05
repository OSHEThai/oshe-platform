import os
import re
import unittest
import yaml

class TestV040ChecklistLifecycleBaseline(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.doc_path = os.path.join(
            os.path.dirname(os.path.dirname(os.path.abspath(__file__))),
            "docs",
            "architecture",
            "v040-checklist-lifecycle-baseline.md"
        )
        if not os.path.exists(cls.doc_path):
            raise FileNotFoundError(f"Checklist lifecycle baseline not found at {cls.doc_path}")

        with open(cls.doc_path, "r", encoding="utf-8") as f:
            cls.content = f.read()

        # Extract frontmatter
        fm_match = re.match(r"^---\s*\n(.*?)\n---\s*\n", cls.content, re.DOTALL)
        if not fm_match:
            raise ValueError("Frontmatter not found in checklist lifecycle baseline")
        cls.frontmatter = yaml.safe_load(fm_match.group(1))

        # Extract synthetic multi-version lifecycle fixture YAML
        yaml_blocks = re.findall(r"```yaml\s*\n(.*?)\n```", cls.content, re.DOTALL)
        cls.fixture = None
        for block in yaml_blocks:
            try:
                parsed = yaml.safe_load(block)
                if isinstance(parsed, dict) and "fixture_id" in parsed and "iterations" in parsed:
                    cls.fixture = parsed
                    break
            except Exception:
                continue

    def test_document_metadata_and_frontmatter(self):
        """Validates document metadata, identifiers, and governance gates in YAML frontmatter."""
        fm = self.frontmatter
        self.assertEqual(fm.get("document_id"), "ARC-V040-CHKLIFE-001")
        self.assertEqual(fm.get("governing_issue"), "GitHub Issue #118")
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

    def test_lifecycle_states_enumeration(self):
        """Validates that all seven discrete lifecycle states are documented."""
        expected_states = [
            "DRAFT",
            "UNDER_REVIEW",
            "REJECTED",
            "APPROVED",
            "PUBLISHED_IMMUTABLE",
            "RETIRED",
            "SUPERSEDED",
        ]
        for state in expected_states:
            self.assertIn(state, self.content, f"Missing lifecycle state definition for {state}")

    def test_authorized_transition_matrix_and_role_boundaries(self):
        """Validates authorized roles, transition events, and segregation of duties (SOD-03)."""
        self.assertIn("Checklist Author", self.content)
        self.assertIn("Independent Reviewer", self.content)
        self.assertIn("SOD-03", self.content)
        self.assertIn("Self-Approval Prohibited", self.content)
        self.assertIn("ErrUnauthorizedTransition", self.content)

    def test_independent_reviewer_attribution(self):
        """Validates mandatory reviewer attribution fields and structured review findings."""
        self.assertIn("reviewer_subject", self.content)
        self.assertIn("reviewer_role", self.content)
        self.assertIn("review_timestamp", self.content)
        self.assertIn("review_findings", self.content)
        self.assertIn("BLOCKING", self.content)
        self.assertIn("MAJOR", self.content)

    def test_immutable_published_versions(self):
        """Validates cryptographic sealing with content_digest and prohibition of in-place editing."""
        self.assertIn("PUBLISHED_IMMUTABLE", self.content)
        self.assertIn("content_digest", self.content)
        self.assertIn("ErrImmutableVersionMutation", self.content)

    def test_copy_provenance_and_version_linking(self):
        """Validates SemVer progression, predecessor_version, and derived_from_digest."""
        self.assertIn("predecessor_version", self.content)
        self.assertIn("derived_from_digest", self.content)
        self.assertIn("copied_by", self.content)
        self.assertIn("copied_at", self.content)

    def test_retirement_supersession_and_execution_pinning(self):
        """Validates retirement reasons, supersession links, and in-flight inspection pinning."""
        self.assertIn("retired_at", self.content)
        self.assertIn("retirement_reason", self.content)
        self.assertIn("successor_version", self.content)
        self.assertIn("In-Flight Schema Pinning", self.content)
        self.assertIn("Zero Runtime Hot-Swapping", self.content)

    def test_append_only_lifecycle_audit_events(self):
        """Validates append-only audit events emitted to MOD-REC."""
        events = [
            "checklist.submitted",
            "checklist.approved",
            "checklist.rejected",
            "checklist.published",
            "checklist.copied",
            "checklist.retired",
            "checklist.superseded",
        ]
        for evt in events:
            self.assertIn(evt, self.content, f"Missing audit event {evt}")

    def test_synthetic_multi_version_lifecycle_fixtures(self):
        """Validates the synthetic multi-version lifecycle fixture YAML structure."""
        self.assertIsNotNone(self.fixture, "Synthetic multi-version lifecycle fixture YAML block not found")
        fix = self.fixture

        self.assertEqual(fix.get("fixture_id"), "fix_syn_checklist_lifecycle_v1_v2")
        self.assertEqual(fix.get("template_id"), "chk_syn_pilot_plant_safety_v1")

        iterations = fix.get("iterations", [])
        self.assertEqual(len(iterations), 2, "Fixture must define exactly 2 iterations (v1.0.0 and v1.1.0)")

        # Iteration 1
        it1 = iterations[0]
        self.assertEqual(it1.get("version"), "1.0.0")
        self.assertEqual(it1.get("lifecycle_status"), "SUPERSEDED")
        self.assertEqual(it1.get("successor_version"), "1.1.0")
        self.assertTrue(bool(it1.get("content_digest")))
        self.assertEqual(it1.get("review_record", {}).get("review_decision"), "APPROVED")
        self.assertEqual(it1.get("review_record", {}).get("reviewer_role"), "Independent Reviewer")

        # Iteration 2
        it2 = iterations[1]
        self.assertEqual(it2.get("version"), "1.1.0")
        self.assertEqual(it2.get("lifecycle_status"), "PUBLISHED_IMMUTABLE")
        self.assertEqual(it2.get("predecessor_version"), "1.0.0")
        self.assertEqual(it2.get("derived_from_digest"), it1.get("content_digest"))
        self.assertTrue(bool(it2.get("content_digest")))
        self.assertEqual(it2.get("review_record", {}).get("review_decision"), "APPROVED")

if __name__ == "__main__":
    unittest.main()
