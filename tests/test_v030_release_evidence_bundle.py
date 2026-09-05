from __future__ import annotations

import pathlib
import re
import unittest

ROOT = pathlib.Path(__file__).resolve().parents[1]

DOCS_DIR = ROOT / "docs" / "architecture"
TESTS_DIR = ROOT / "tests"

EVIDENCE_BUNDLE_PATH = DOCS_DIR / "v030-release-evidence-bundle.md"
EVIDENCE_GATES_PATH = DOCS_DIR / "v030-evidence-gates.md"
EXTERNAL_ROUTE_PACKET_PATH = DOCS_DIR / "v030-external-route-activation-packet.md"
ORG_ISOLATION_ASSURE_PATH = DOCS_DIR / "v030-organization-party-portal-isolation-assurance.md"
THREAT_SAFETY_ASSURE_PATH = DOCS_DIR / "v030-threat-privacy-safety-assurance.md"
WALKING_SKELETON_TEST_PATH = TESTS_DIR / "test_v030_walking_skeleton_integration_harness.py"
ARCH_README_PATH = DOCS_DIR / "README.md"
RELEASE_README_PATH = TESTS_DIR / "release" / "README.md"


class TestV030ReleaseEvidenceBundle(unittest.TestCase):
    """Deterministic validation test suite for Milestone v0.3.0 Release Evidence Bundle."""

    def test_evidence_bundle_existence_and_metadata(self) -> None:
        """Assert release evidence bundle exists, is non-empty, and carries expected metadata."""
        self.assertTrue(EVIDENCE_BUNDLE_PATH.is_file(), f"Missing {EVIDENCE_BUNDLE_PATH}")
        content = EVIDENCE_BUNDLE_PATH.read_text(encoding="utf-8")
        self.assertGreater(len(content), 1000, "Evidence bundle document too short")
        self.assertIn("`REL-V030-EVD-001`", content)
        self.assertIn("`V030-I037`", content)
        self.assertIn("Issue #110", content)
        self.assertIn("v0.3.0 - Organization Identity and Portal Alpha", content)

    def test_reconciled_architecture_documents_exist_and_contain_ids(self) -> None:
        """Assert all 5 reconciled architecture documents exist and contain authoritative IDs."""
        expected_docs = [
            (EVIDENCE_BUNDLE_PATH, "REL-V030-EVD-001"),
            (EVIDENCE_GATES_PATH, "ARC-V030-EVGATES-001"),
            (EXTERNAL_ROUTE_PACKET_PATH, "ARC-V030-EXTROUTE-001"),
            (ORG_ISOLATION_ASSURE_PATH, "DOC-ASSURE-V030-001"),
            (THREAT_SAFETY_ASSURE_PATH, "DOC-ASSURE-V030-002"),
        ]
        for path, doc_id in expected_docs:
            self.assertTrue(path.is_file(), f"Document missing: {path}")
            content = path.read_text(encoding="utf-8")
            self.assertGreater(len(content), 500, f"Document too small: {path}")
            self.assertIn(doc_id, content, f"Missing document ID {doc_id} in {path}")

    def test_walking_skeleton_harness_exists(self) -> None:
        """Assert walking skeleton integration harness exists and contains test cases."""
        self.assertTrue(WALKING_SKELETON_TEST_PATH.is_file(), f"Missing {WALKING_SKELETON_TEST_PATH}")
        content = WALKING_SKELETON_TEST_PATH.read_text(encoding="utf-8")
        self.assertIn("class V030WalkingSkeletonIntegrationHarnessTests", content)
        self.assertIn("SyntheticV030IntegrationContext", content)

    def test_owner_held_human_gates_preserved(self) -> None:
        """Assert H030-007 and H030-008 are explicitly recorded as HOLD across evidence documents."""
        bundle_content = EVIDENCE_BUNDLE_PATH.read_text(encoding="utf-8")
        gates_content = EVIDENCE_GATES_PATH.read_text(encoding="utf-8")
        ext_route_content = EXTERNAL_ROUTE_PACKET_PATH.read_text(encoding="utf-8")

        for content in [bundle_content, gates_content, ext_route_content]:
            self.assertIn("H030-007", content)
            self.assertIn("HOLD", content)

        self.assertIn("H030-008", bundle_content)
        self.assertIn("H030-008", gates_content)

        # Assert non-claims
        self.assertIn("Zero Release Recommendation", bundle_content)
        self.assertIn("Zero Release Approval", bundle_content)
        self.assertIn("Zero Live Public Route", bundle_content)

    def test_predecessor_non_adoption_attestation(self) -> None:
        """Assert predecessor attempt PR #1057 is explicitly marked unadopted / no credit."""
        content = EVIDENCE_BUNDLE_PATH.read_text(encoding="utf-8")
        self.assertIn("PR #1057", content)
        self.assertIn("CLOSED_UNADOPTED", content)
        self.assertIn("HELD_NO_CREDIT", content)

    def test_evidence_gates_matrix_traceability(self) -> None:
        """Assert all 8 capability gate identifiers are reconciled in the bundle."""
        gate_ids = [
            "GATE-ORG-01",
            "GATE-ID-01",
            "GATE-AUTH-01",
            "GATE-PORTAL-01",
            "GATE-PUB-01",
            "GATE-CTR-01",
            "GATE-A11Y-01",
            "GATE-REL-01",
        ]
        bundle_content = EVIDENCE_BUNDLE_PATH.read_text(encoding="utf-8")
        gates_content = EVIDENCE_GATES_PATH.read_text(encoding="utf-8")

        for gid in gate_ids:
            self.assertIn(gid, gates_content, f"Missing {gid} in evidence gates document")
            self.assertIn(gid, bundle_content, f"Missing {gid} in release evidence bundle")

    def test_negative_controls_traceability(self) -> None:
        """Assert negative control prefixes are represented in evidence gates."""
        gates_content = EVIDENCE_GATES_PATH.read_text(encoding="utf-8")
        for prefix in ["NEG-ORG", "NEG-IAM", "NEG-AUTH", "NEG-PRT", "NEG-PUB", "NEG-EXT"]:
            self.assertIn(prefix, gates_content, f"Missing negative control {prefix}")

    def test_architecture_readme_index_reconciliation(self) -> None:
        """Assert architecture README indexes the release evidence bundle."""
        self.assertTrue(ARCH_README_PATH.is_file(), f"Missing {ARCH_README_PATH}")
        content = ARCH_README_PATH.read_text(encoding="utf-8")
        self.assertIn("v030-release-evidence-bundle.md", content)
        self.assertIn("REL-V030-EVD-001", content)

    def test_release_readme_updated(self) -> None:
        """Assert release test README is updated and points to the evidence bundle test."""
        self.assertTrue(RELEASE_README_PATH.is_file(), f"Missing {RELEASE_README_PATH}")
        content = RELEASE_README_PATH.read_text(encoding="utf-8")
        self.assertIn("test_v030_release_evidence_bundle.py", content)


if __name__ == "__main__":
    unittest.main()
