#!/usr/bin/env python3
"""
test_v040_inspect_integration_recovery_qualification_evidence.py
Automated validation suite for v0.4.0 Inspect Integration and Recovery Qualification Evidence.
Governed by Issue #148 / ASN-V040-I037-INTEGRATION-RECOVERY-001.

Verifies:
1. Integration and recovery qualification evidence document exists and has non-trivial length.
2. Frontmatter metadata conforms to governance requirements (document_id, governing_issue, retained_holds).
3. Explicit non-claims and unproven declarations: live device, live network, backup/restore,
   performance, user, runtime, and H040-008 remain unproven.
4. Retained foundation holds (H040-007 .. H040-011) remain strictly on HOLD with zero authority grant.
5. All six Go module suites are documented with PASS (synthetic/in-process) results.
6. Six integration and recovery domains are mapped to source tests:
   - Scope denial
   - Offline / interruption
   - Evidence / QR abuse
   - Scoring / CAPA / reinspection
   - Export / audit
   - Diagnostics / outbox / recovery
7. Source-to-test mapping points to real existing files in the worktree.
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


class TestV040InspectIntegrationRecoveryQualificationEvidence(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.repo_root = get_repo_root()
        cls.evidence_path = (
            cls.repo_root
            / "docs"
            / "architecture"
            / "v040-inspect-integration-recovery-qualification-evidence.md"
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

        self.assertIn("document_id: EV-V040-INT-REC-001", fm_text)
        self.assertIn("governing_issue: 148", fm_text)
        self.assertIn(
            "assignment_id: ASN-V040-I037-INTEGRATION-RECOVERY-001", fm_text
        )
        self.assertIn(
            "lease_id: LEASE-V040-I037-INTEGRATION-RECOVERY-001", fm_text
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

        # Must explicitly label suite results synthetic/in-process
        self.assertIn("synthetic", content_lower)
        self.assertIn("in-process", content_lower)

        # Explicit non-claims
        self.assertIn("live device evidence unproven", content_lower)
        self.assertIn("live network evidence unproven", content_lower)
        self.assertIn("backup and restore evidence unproven", content_lower)
        self.assertIn("performance evidence unproven", content_lower)
        self.assertIn("user evidence unproven", content_lower)
        self.assertIn("runtime evidence unproven", content_lower)
        self.assertIn("h040-008 remains unproven", content_lower)

        # Zero authority grant statement
        self.assertIn(
            "zero authority is granted to alter, bypass, or lift any hold",
            content_lower,
        )

    def test_04_retained_foundation_holds_table(self):
        content = self.evidence_path.read_text(encoding="utf-8")

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

    def test_05_six_module_suites_pass_results(self):
        content = self.evidence_path.read_text(encoding="utf-8")

        expected_modules = [
            "modules/workflow-action",
            "modules/files-evidence",
            "modules/records-audit",
            "modules/events-outbox-jobs",
            "modules/identity-authorization",
            "modules/reporting-localization",
        ]
        for mod in expected_modules:
            self.assertIn(mod, content, f"Module {mod} not documented in evidence")

        # Verify PASS results documented
        self.assertIn("6 of 6 suites passed. 0 suites failed.", content)

    def test_06_six_integration_recovery_domains_mapped(self):
        content = self.evidence_path.read_text(encoding="utf-8")

        required_domains = [
            "Scope Denial",
            "Offline / Interruption",
            "Evidence / QR Abuse",
            "Scoring / CAPA / Reinspection",
            "Export / Audit",
            "Diagnostics / Outbox / Recovery",
        ]
        for domain in required_domains:
            self.assertIn(
                domain, content, f"Required domain {domain} missing from evidence document"
            )

        # Verify key test functions mapped
        mapped_test_names = [
            "TestNegativeControl_AnonymousAndUnauthenticated",
            "TestOperationalQualification_RollbackNoSilentStateMutation",
            "TestEvidenceChain_OriginalImmutability",
            "TestScanResolution_MalformedInput",
            "TestQualification_BoundaryRoundingExactBasisPoints",
            "TestRecordStore_DeclareRecord_Valid",
            "TestOperationalQualification_DuplicateAndPoisonReplayHandling",
        ]
        for test_fn in mapped_test_names:
            self.assertIn(
                test_fn, content, f"Mapped test {test_fn} missing from evidence document"
            )

    def test_07_source_mapping_files_exist_on_disk(self):
        # Verify that key mapped files referenced in the evidence actually exist on disk
        mapped_files = [
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
            # files-evidence
            self.repo_root / "modules" / "files-evidence" / "evidence_chain.go",
            self.repo_root / "modules" / "files-evidence" / "evidence_chain_test.go",
            self.repo_root / "modules" / "files-evidence" / "file_metadata.go",
            # records-audit
            self.repo_root / "modules" / "records-audit" / "record_audit.go",
            self.repo_root / "modules" / "records-audit" / "immutable_objects.go",
            self.repo_root / "modules" / "records-audit" / "retention_export.go",
            # events-outbox-jobs
            self.repo_root / "modules" / "events-outbox-jobs" / "event_outbox.go",
            self.repo_root / "modules" / "events-outbox-jobs" / "event_dispatcher.go",
            self.repo_root
            / "modules"
            / "events-outbox-jobs"
            / "operational_qualification_test.go",
            # identity-authorization
            self.repo_root / "modules" / "identity-authorization" / "access_policy.go",
            self.repo_root / "modules" / "identity-authorization" / "scan_resolution.go",
            self.repo_root
            / "modules"
            / "identity-authorization"
            / "scan_resolution_test.go",
            # reporting-localization
            self.repo_root / "modules" / "reporting-localization" / "reporting.go",
            self.repo_root
            / "modules"
            / "reporting-localization"
            / "reporting_qualification_test.go",
        ]
        for f in mapped_files:
            self.assertTrue(f.is_file(), f"Mapped source file does not exist on disk: {f}")


if __name__ == "__main__":
    unittest.main()
