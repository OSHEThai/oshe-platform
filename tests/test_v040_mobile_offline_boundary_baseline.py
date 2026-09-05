import os
import re
import unittest
import yaml


class TestV040MobileOfflineBoundaryBaseline(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.doc_path = os.path.join(
            os.path.dirname(os.path.dirname(os.path.abspath(__file__))),
            "docs",
            "architecture",
            "v040-mobile-offline-boundary-baseline.md",
        )
        if not os.path.exists(cls.doc_path):
            raise FileNotFoundError(f"Mobile offline boundary baseline not found at {cls.doc_path}")

        with open(cls.doc_path, "r", encoding="utf-8") as f:
            cls.content = f.read()

        # Extract frontmatter
        fm_match = re.match(r"^---\s*\n(.*?)\n---\s*\n", cls.content, re.DOTALL)
        if not fm_match:
            raise ValueError("Frontmatter not found in mobile offline boundary baseline")
        cls.frontmatter = yaml.safe_load(fm_match.group(1))

        # Extract synthetic multi-scenario fixture YAML
        yaml_blocks = re.findall(r"```yaml\s*\n(.*?)\n```", cls.content, re.DOTALL)
        cls.fixture = None
        for block in yaml_blocks:
            try:
                parsed = yaml.safe_load(block)
                if isinstance(parsed, dict) and "fixture_id" in parsed and "packages" in parsed:
                    cls.fixture = parsed
                    break
            except Exception:
                continue

    def test_document_metadata_and_frontmatter(self):
        """Validates document metadata, identifiers, and governance gates in YAML frontmatter."""
        fm = self.frontmatter
        self.assertEqual(fm.get("document_id"), "ARC-V040-MOBOFF-001")
        self.assertEqual(fm.get("governing_issue"), "GitHub Issue #124")
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

    def test_supported_client_platform_matrix(self):
        """Validates supported browser engines and explicit rejection of unsupported platforms."""
        self.assertIn("Google Chrome", self.content)
        self.assertIn("Microsoft Edge", self.content)
        self.assertIn("Google Chrome Mobile", self.content)
        self.assertIn("Apple iOS / iPadOS", self.content)
        self.assertIn("Mozilla Firefox", self.content)
        self.assertIn("Embedded Webviews", self.content)
        self.assertIn("Native Applications", self.content)

    def test_protected_local_storage_and_data_minimization(self):
        """Validates IndexedDB storage, SessionStorage, localStorage prohibition, and purge on logout."""
        self.assertIn("IndexedDB", self.content)
        self.assertIn("oshe_offline_db", self.content)
        self.assertIn("SessionStorage", self.content)
        self.assertIn("window.localStorage", self.content)
        self.assertIn("strictly prohibited", self.content)
        self.assertIn("Deterministic Session Purge", self.content)
        self.assertIn("50 MB", self.content)

    def test_reference_data_age_and_offline_lease_expiry(self):
        """Validates 24-hour reference data age ceiling, 24-hour lease TTL, edit lock, and draft preservation."""
        self.assertIn("MAX_REFERENCE_DATA_AGE_HOURS = 24", self.content)
        self.assertIn("MAX_OFFLINE_LEASE_HOURS = 24", self.content)
        self.assertIn("OFFLINE_LEASE_EXPIRED", self.content)
        self.assertIn("Fail-Closed Edit Lock", self.content)
        self.assertIn("Draft Preservation Invariant", self.content)

    def test_online_only_protected_states_matrix(self):
        """Validates separation of offline-permissible data capture from online-only state transitions."""
        self.assertIn("Download Work Package", self.content)
        self.assertIn("Submit / Finalize Inspection", self.content)
        self.assertIn("Independent Review / Verification", self.content)
        self.assertIn("Finding / CAPA Closure", self.content)
        self.assertIn("Reassign / Cancel Work", self.content)
        self.assertIn("Online Only", self.content)

    def test_server_authority_and_conflict_quarantine(self):
        """Validates server sole authority, rejection of LWW, and conflict quarantine."""
        self.assertIn("Server Sole Authority Invariant", self.content)
        self.assertIn("Rejection of Last-Write-Wins", self.content)
        self.assertIn("QUARANTINED", self.content)
        self.assertIn("ConflictRecord", self.content)

    def test_fail_closed_denial_reasons(self):
        """Validates fail-closed denial codes and error identifiers."""
        expected_denials = [
            ("DENIAL_UNSUPPORTED_DEVICE", "ErrUnsupportedDevicePlatform"),
            ("DENIAL_EXPIRED_MEMBERSHIP", "ErrExpiredTenantMembership"),
            ("DENIAL_WRONG_SCOPE", "ErrUnauthorizedAssignmentScope"),
            ("DENIAL_STALE_CHECKLIST", "ErrStaleChecklistTemplate"),
            ("DENIAL_STALE_REFERENCE_DATA", "ErrStaleReferenceData"),
            ("DENIAL_STORAGE_EXPIRY", "ErrOfflineLeaseExpired"),
            ("DENIAL_ONLINE_ONLY_STATE", "ErrOnlineOnlyStateTransition"),
        ]
        for code, err in expected_denials:
            self.assertIn(code, self.content, f"Missing denial code {code}")
            self.assertIn(err, self.content, f"Missing error identifier {err}")

    def test_synthetic_fixture_structure(self):
        """Validates synthetic multi-scenario fixture YAML structure."""
        self.assertIsNotNone(self.fixture, "Synthetic multi-scenario fixture YAML block not found")
        fix = self.fixture

        self.assertEqual(fix.get("fixture_id"), "fix_syn_mobile_offline_packages_v1")
        packages = fix.get("packages", [])
        self.assertEqual(len(packages), 2, "Fixture must define exactly 2 package scenarios")

        # Package 1: Active Supported Mobile
        p1 = packages[0]
        self.assertEqual(p1.get("package_id"), "wpk_syn_mobile_active_01")
        self.assertEqual(p1.get("status"), "ACTIVE_OFFLINE")
        self.assertEqual(p1.get("client_platform"), "Android Chrome Mobile v128")
        self.assertEqual(p1.get("limits", {}).get("max_offline_hours"), 24)

        # Package 2: Expired Offline Lease
        p2 = packages[1]
        self.assertEqual(p2.get("package_id"), "wpk_syn_mobile_expired_02")
        self.assertEqual(p2.get("status"), "OFFLINE_LEASE_EXPIRED")
        self.assertTrue(p2.get("edit_locked"))
        self.assertTrue(p2.get("draft_preserved"))

        # Denial scenarios
        denials = fix.get("denial_scenarios", [])
        self.assertGreaterEqual(len(denials), 3, "Fixture must define at least 3 denial scenarios")

    def test_governance_prohibitions_and_non_claims(self):
        """Validates retained holds and anti-scope non-claims."""
        for hold_id in ["H040-007", "H040-008", "H040-009", "H040-010", "H040-011"]:
            self.assertIn(hold_id, self.content)

        self.assertIn("100% Synthetic Data Policy", self.content)
        self.assertIn("Server Authority Invariant", self.content)
        self.assertIn("No External Route, Notification, or MDM Activation", self.content)
        self.assertIn("No Real Participant Onboarding", self.content)


if __name__ == "__main__":
    unittest.main()
