import os
import re
import unittest
import yaml


class TestV040EvidenceCapturePrework(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.doc_path = os.path.join(
            os.path.dirname(os.path.dirname(os.path.abspath(__file__))),
            "docs",
            "architecture",
            "v040-evidence-capture-prework.md",
        )
        if not os.path.exists(cls.doc_path):
            raise FileNotFoundError(f"Evidence capture prework doc not found at {cls.doc_path}")

        with open(cls.doc_path, "r", encoding="utf-8") as f:
            cls.content = f.read()

        # Extract frontmatter
        fm_match = re.match(r"^---\s*\n(.*?)\n---\s*\n", cls.content, re.DOTALL)
        if not fm_match:
            raise ValueError("Frontmatter not found in evidence capture prework doc")
        cls.frontmatter = yaml.safe_load(fm_match.group(1))

        # Extract synthetic multi-scenario fixture YAML
        yaml_blocks = re.findall(r"```yaml\s*\n(.*?)\n```", cls.content, re.DOTALL)
        cls.fixture = None
        for block in yaml_blocks:
            try:
                parsed = yaml.safe_load(block)
                if isinstance(parsed, dict) and "fixture_id" in parsed and "attachments" in parsed:
                    cls.fixture = parsed
                    break
            except Exception:
                continue

    def test_document_metadata_and_frontmatter(self):
        """Validates document metadata, identifiers, and governance gates in YAML frontmatter."""
        fm = self.frontmatter
        self.assertEqual(fm.get("document_id"), "ARC-V040-EVD-001")
        self.assertEqual(fm.get("governing_issue"), "GitHub Issue #128")
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

        # Pending dependency V040-I014
        pending_deps = fm.get("pending_dependencies", [])
        self.assertIn("PENDING_DEPENDENCY_V040_I014", pending_deps)
        self.assertIn("PENDING_DEPENDENCY_V040_I014", self.content)

        # Unselected human-owned policies
        unselected = fm.get("retained_unselected_policies", {})
        self.assertEqual(unselected.get("binding_scoring_policy"), "HUMAN_OWNED_UNSELECTED")
        self.assertEqual(unselected.get("finding_closure_policy"), "HUMAN_OWNED_UNSELECTED")
        self.assertEqual(unselected.get("offline_authority"), "HUMAN_OWNED_UNSELECTED")
        self.assertEqual(unselected.get("evidence_retention_policy"), "HUMAN_OWNED_UNSELECTED")
        self.assertEqual(unselected.get("external_storage_provider_policy"), "HUMAN_OWNED_UNSELECTED")

    def test_explicit_association_targets(self):
        """Validates that all three association targets and orphan prohibition are documented."""
        self.assertIn("INSPECTION_RESPONSE", self.content)
        self.assertIn("SAFETY_FINDING", self.content)
        self.assertIn("CAPA_ACTION", self.content)
        self.assertIn("Prohibition of Orphan Evidence", self.content)

    def test_metadata_minimization_and_exif_stripping(self):
        """Validates EXIF stripping and data minimization under H040-003."""
        self.assertIn("EXIF", self.content)
        self.assertIn("GPS", self.content)
        self.assertIn("camera serial number", self.content)
        self.assertIn("Sanitized Allowed Technical Metadata", self.content)
        self.assertIn("pixel_width", self.content)
        self.assertIn("pixel_height", self.content)

    def test_supported_media_formats_and_limits(self):
        """Validates supported image/PDF types, 10 MiB limit, and rejection of video/audio/executables."""
        self.assertIn("image/jpeg", self.content)
        self.assertIn("image/png", self.content)
        self.assertIn("image/webp", self.content)
        self.assertIn("application/pdf", self.content)
        self.assertIn("10 MiB", self.content)
        self.assertIn("video/*", self.content)
        self.assertIn("audio/*", self.content)

    def test_client_local_queue_and_preview(self):
        """Validates IndexedDB local queueing, SHA-256 computation upon capture, and preview lifecycle."""
        self.assertIn("IndexedDB", self.content)
        self.assertIn("evidence_blobs", self.content)
        self.assertIn("evidence_queue", self.content)
        self.assertIn("Immediate SHA-256 Computation", self.content)
        self.assertIn("URL.createObjectURL", self.content)
        self.assertIn("Deterministic Cleanup", self.content)

    def test_fail_closed_denial_reasons(self):
        """Validates fail-closed denial codes and error identifiers."""
        expected_denials = [
            ("DENIAL_UNSUPPORTED_MEDIA_TYPE", "ErrInvalidMediaType"),
            ("DENIAL_SIZE_EXCEEDED", "ErrInvalidSize"),
            ("DENIAL_MISSING_ASSOCIATION", "ErrMissingAssociationTarget"),
            ("DENIAL_WRONG_SCOPE", "ErrTenantMismatch"),
            ("DENIAL_METADATA_VIOLATION", "ErrInvalidFilename"),
            ("DENIAL_DIGEST_MISMATCH", "ErrIntegrityMismatch"),
            ("DENIAL_PREVIEW_UNAUTHORIZED", "ErrUnauthorizedPreviewAccess"),
        ]
        for code, err in expected_denials:
            self.assertIn(code, self.content, f"Missing denial code {code}")
            self.assertIn(err, self.content, f"Missing error identifier {err}")

    def test_synthetic_fixture_structure(self):
        """Validates synthetic multi-scenario fixture YAML structure."""
        self.assertIsNotNone(self.fixture, "Synthetic multi-scenario fixture YAML block not found")
        fix = self.fixture

        self.assertEqual(fix.get("fixture_id"), "fix_syn_evidence_capture_v1")
        attachments = fix.get("attachments", [])
        self.assertEqual(len(attachments), 3, "Fixture must define exactly 3 attachment scenarios")

        # Attachment 1: Response
        a1 = attachments[0]
        self.assertEqual(a1.get("association_type"), "INSPECTION_RESPONSE")
        self.assertEqual(a1.get("media_type"), "image/jpeg")
        self.assertEqual(a1.get("source_context"), "CAMERA_DIRECT")

        # Attachment 2: Finding
        a2 = attachments[1]
        self.assertEqual(a2.get("association_type"), "SAFETY_FINDING")
        self.assertEqual(a2.get("media_type"), "image/png")

        # Attachment 3: CAPA
        a3 = attachments[2]
        self.assertEqual(a3.get("association_type"), "CAPA_ACTION")
        self.assertEqual(a3.get("media_type"), "image/webp")

        # Denial scenarios
        denials = fix.get("denial_scenarios", [])
        self.assertGreaterEqual(len(denials), 4, "Fixture must define at least 4 denial scenarios")

    def test_governance_prohibitions_and_non_claims(self):
        """Validates retained holds and anti-scope non-claims."""
        for hold_id in ["H040-007", "H040-008", "H040-009", "H040-010", "H040-011"]:
            self.assertIn(hold_id, self.content)

        self.assertIn("100% Synthetic Data Policy", self.content)
        self.assertIn("No External Route or Cloud Bucket Activation", self.content)
        self.assertIn("Issue Closure Prohibition", self.content)


if __name__ == "__main__":
    unittest.main()
