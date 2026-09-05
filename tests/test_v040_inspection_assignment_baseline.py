import os
import re
import unittest
import yaml


class TestV040InspectionAssignmentBaseline(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.doc_path = os.path.join(
            os.path.dirname(os.path.dirname(os.path.abspath(__file__))),
            "docs",
            "architecture",
            "v040-inspection-assignment-baseline.md",
        )
        if not os.path.exists(cls.doc_path):
            raise FileNotFoundError(f"Inspection assignment baseline not found at {cls.doc_path}")

        with open(cls.doc_path, "r", encoding="utf-8") as f:
            cls.content = f.read()

        # Extract frontmatter
        fm_match = re.match(r"^---\s*\n(.*?)\n---\s*\n", cls.content, re.DOTALL)
        if not fm_match:
            raise ValueError("Frontmatter not found in inspection assignment baseline")
        cls.frontmatter = yaml.safe_load(fm_match.group(1))

        # Extract synthetic multi-assignment fixture YAML
        yaml_blocks = re.findall(r"```yaml\s*\n(.*?)\n```", cls.content, re.DOTALL)
        cls.fixture = None
        for block in yaml_blocks:
            try:
                parsed = yaml.safe_load(block)
                if isinstance(parsed, dict) and "fixture_id" in parsed and "assignments" in parsed:
                    cls.fixture = parsed
                    break
            except Exception:
                continue

    def test_document_metadata_and_frontmatter(self):
        """Validates document metadata, identifiers, and governance gates in YAML frontmatter."""
        fm = self.frontmatter
        self.assertEqual(fm.get("document_id"), "ARC-V040-ASGN-001")
        self.assertEqual(fm.get("governing_issue"), "GitHub Issue #121")
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

    def test_four_foundation_roles_and_inspector_eligibility(self):
        """Validates that all four roles are documented and only Inspector is eligible for assignment."""
        self.assertIn("Checklist Author", self.content)
        self.assertIn("Inspector", self.content)
        self.assertIn("CAPA Owner", self.content)
        self.assertIn("Independent Reviewer", self.content)
        self.assertIn("Sole role eligible for inspection execution assignment", self.content)

    def test_supported_and_unsupported_device_platforms(self):
        """Validates supported browser engines and explicit rejection of unsupported platforms."""
        self.assertIn("Desktop Google Chrome", self.content)
        self.assertIn("Desktop Microsoft Edge", self.content)
        self.assertIn("Mobile Android Chrome", self.content)
        self.assertIn("Apple iOS", self.content)
        self.assertIn("Mozilla Firefox", self.content)

    def test_denial_reasons_register(self):
        """Validates that all seven fail-closed denial codes and error identifiers are documented."""
        expected_denials = [
            ("DENIAL_EXPIRED_MEMBERSHIP", "ErrExpiredTenantMembership"),
            ("DENIAL_WRONG_SCOPE", "ErrUnauthorizedAssignmentScope"),
            ("DENIAL_REVOKED_ROLE", "ErrRevokedInspectorRole"),
            ("DENIAL_STALE_SESSION", "ErrStaleUserSession"),
            ("DENIAL_UNSUPPORTED_DEVICE", "ErrUnsupportedDevicePlatform"),
            ("DENIAL_DUPLICATE_ASSIGNMENT", "ErrDuplicateInspectionAssignment"),
            ("DENIAL_DOWNLOADED_AMBIGUITY", "ErrDownloadedWorkConflict"),
        ]
        for code, err in expected_denials:
            self.assertIn(code, self.content, f"Missing denial code {code}")
            self.assertIn(err, self.content, f"Missing error identifier {err}")

    def test_downloaded_work_authority_and_preservation(self):
        """Validates preservation of prior responsibility and supervisory override on reassignment."""
        self.assertIn("Prohibition Against Erasing Prior Responsibility", self.content)
        self.assertIn("override_downloaded_work", self.content)
        self.assertIn("REVOKED_SUPERSEDED", self.content)
        self.assertIn("ErrExecutionCancelledWhileDownloaded", self.content)
        self.assertIn("QUARANTINED", self.content)

    def test_assignment_lifecycle_states(self):
        """Validates all assignment lifecycle states."""
        states = ["UNASSIGNED", "ASSIGNED", "DOWNLOADED", "IN_PROGRESS", "REASSIGNED", "CANCELLED"]
        for st in states:
            self.assertIn(st, self.content, f"Missing assignment lifecycle state {st}")

    def test_append_only_audit_events(self):
        """Validates the five required append-only audit events emitted to MOD-REC."""
        events = [
            "assignment.created",
            "assignment.downloaded",
            "assignment.reassigned",
            "assignment.cancelled",
            "assignment.denied",
        ]
        for evt in events:
            self.assertIn(evt, self.content, f"Missing audit event {evt}")

    def test_synthetic_fixture_structure(self):
        """Validates synthetic multi-assignment fixture YAML structure."""
        self.assertIsNotNone(self.fixture, "Synthetic multi-assignment fixture YAML block not found")
        fix = self.fixture

        self.assertEqual(fix.get("fixture_id"), "fix_syn_inspection_assignments_v1")
        assignments = fix.get("assignments", [])
        self.assertEqual(len(assignments), 3, "Fixture must define exactly 3 assignment scenarios")

        # Scenario 1: Clean Download
        s1 = assignments[0]
        self.assertEqual(s1.get("assignment_id"), "asg_syn_clean_01")
        self.assertEqual(s1.get("status"), "DOWNLOADED")
        self.assertTrue(s1.get("download_state", {}).get("is_downloaded"))

        # Scenario 2: Reassigned with Override
        s2 = assignments[1]
        self.assertEqual(s2.get("assignment_id"), "asg_syn_reassigned_02")
        self.assertEqual(s2.get("status"), "REASSIGNED")
        self.assertTrue(s2.get("reassignment_record", {}).get("override_downloaded_work"))

        # Scenario 3: Cancelled preserving download
        s3 = assignments[2]
        self.assertEqual(s3.get("assignment_id"), "asg_syn_cancelled_03")
        self.assertEqual(s3.get("status"), "CANCELLED")
        self.assertTrue(s3.get("download_state", {}).get("is_downloaded"))

        # Denial scenarios
        denials = fix.get("denial_scenarios", [])
        self.assertGreaterEqual(len(denials), 5, "Fixture must define at least 5 denial scenarios")

    def test_governance_prohibitions_and_non_claims(self):
        """Validates retained holds and anti-scope non-claims."""
        for hold_id in ["H040-007", "H040-008", "H040-009", "H040-010", "H040-011"]:
            self.assertIn(hold_id, self.content)

        self.assertIn("100% Synthetic Data Policy", self.content)
        self.assertIn("Default-Deny Authority Invariant", self.content)
        self.assertIn("No External Route or Notification Activation", self.content)
        self.assertIn("No Real Participant Onboarding", self.content)


if __name__ == "__main__":
    unittest.main()
