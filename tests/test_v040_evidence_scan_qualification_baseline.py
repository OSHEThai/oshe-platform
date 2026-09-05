from __future__ import annotations

import pathlib
import re
import unittest
import yaml
from typing import Any, Dict, List, Optional

ROOT = pathlib.Path(__file__).resolve().parents[1]
QUALIFICATION_BASELINE_PATH = (
    ROOT / "docs" / "qualification" / "v040-evidence-scan-qualification-baseline.md"
)

FRONTMATTER_PATTERN = re.compile(r"^---\s*\n(.*?)\n---\s*\n", re.DOTALL)


def parse_simple_frontmatter(content: str) -> Dict[str, Any]:
    match = FRONTMATTER_PATTERN.match(content)
    if not match:
        raise ValueError("Missing YAML frontmatter")
    lines = match.group(1).splitlines()
    data: Dict[str, Any] = {}
    current_key: Optional[str] = None

    for line in lines:
        if not line.strip() or line.strip().startswith("#"):
            continue
        stripped = line.strip()
        indent = len(line) - len(line.lstrip())

        # List items
        if stripped.startswith("- "):
            val = stripped[2:].strip().strip("\"'")
            if current_key and isinstance(data.get(current_key), list):
                data[current_key].append(val)
            continue

        # Sub-dictionary items
        if indent > 0 and ":" in stripped and current_key:
            k, v = stripped.split(":", 1)
            k = k.strip()
            v = v.strip().strip("\"'")
            if isinstance(data.get(current_key), dict):
                data[current_key][k] = v
            continue

        # Top-level key
        if ":" in stripped:
            key, val = stripped.split(":", 1)
            key = key.strip()
            val = val.strip().strip("\"'")
            if val == "":
                if key == "retained_unselected_policies":
                    data[key] = {}
                else:
                    data[key] = []
                current_key = key
            else:
                data[key] = val
                current_key = None
    return data


class V040EvidenceScanQualificationBaselineTests(unittest.TestCase):
    def setUp(self) -> None:
        self.assertTrue(
            QUALIFICATION_BASELINE_PATH.is_file(),
            f"Missing evidence scan qualification baseline at {QUALIFICATION_BASELINE_PATH}",
        )
        self.content = QUALIFICATION_BASELINE_PATH.read_text(encoding="utf-8")
        self.frontmatter = parse_simple_frontmatter(self.content)

    def test_document_metadata_and_governance(self) -> None:
        """Asserts required governed frontmatter fields, status, and authority link."""
        self.assertEqual(self.frontmatter.get("document_id"), "QLF-V040-EVDSCAN-001")
        self.assertEqual(self.frontmatter.get("document_type"), "qualification_specification")
        self.assertEqual(self.frontmatter.get("lifecycle_status"), "DRAFT")
        self.assertEqual(self.frontmatter.get("status"), "APPROVED_FOR_LOCAL_DEVELOPMENT")
        self.assertEqual(self.frontmatter.get("authority_source"), "HDEC-V040-FOUNDATION-054")
        self.assertEqual(self.frontmatter.get("governing_issue"), "GitHub Issue #131")
        self.assertEqual(self.frontmatter.get("milestone"), "v0.4.0 - OSHE Inspect Private Alpha")
        self.assertEqual(
            self.frontmatter.get("credit_boundary"),
            "TECHNICAL_QUALIFICATION_ONLY_NO_USER_OR_DEVICE_EVIDENCE",
        )

        approved = self.frontmatter.get("approved_foundation_gates", [])
        for g in ["H040-001", "H040-002", "H040-003", "H040-004", "H040-005", "H040-006"]:
            self.assertIn(g, approved, f"{g} must be present in approved_foundation_gates")

        deferred = self.frontmatter.get("retained_holds", [])
        for h in ["H040-007", "H040-008", "H040-009", "H040-010", "H040-011"]:
            self.assertIn(h, deferred, f"{h} must be present in retained_holds")

    def test_retained_unselected_policies_explicitly_marked(self) -> None:
        """Asserts all five unselected policies are marked HUMAN_OWNED_UNSELECTED."""
        unselected = self.frontmatter.get("retained_unselected_policies", {})
        self.assertEqual(unselected.get("offline_authority"), "HUMAN_OWNED_UNSELECTED")
        self.assertEqual(unselected.get("binding_scoring_policy"), "HUMAN_OWNED_UNSELECTED")
        self.assertEqual(unselected.get("finding_closure_policy"), "HUMAN_OWNED_UNSELECTED")
        self.assertEqual(unselected.get("evidence_retention_policy"), "HUMAN_OWNED_UNSELECTED")
        self.assertEqual(unselected.get("external_storage_provider_policy"), "HUMAN_OWNED_UNSELECTED")

        self.assertIn("HUMAN_OWNED_UNSELECTED", self.content)

    def test_parent_entity_association_and_orphan_rejection(self) -> None:
        """Asserts explicit parent entity binding, orphan rejection, and context scoping."""
        self.assertIn("CHECKLIST_RESPONSE", self.content)
        self.assertIn("FINDING", self.content)
        self.assertIn("ACTION_ITEM", self.content)
        self.assertIn("ERR_ORPHAN_EVIDENCE_PROHIBITED", self.content)
        self.assertIn("ERR_PARENT_CONTEXT_MISMATCH", self.content)

    def test_privacy_metadata_minimization_and_exif_stripping(self) -> None:
        """Asserts mandatory EXIF stripping and metadata minimization."""
        self.assertIn("Mandatory EXIF & Geotag Stripping", self.content)
        self.assertIn("GPS Latitude", self.content)
        self.assertIn("Device serial number", self.content)
        self.assertIn("10 MiB", self.content)

    def test_immutable_originals_and_derived_objects(self) -> None:
        """Asserts write-once originals and derived object classification."""
        self.assertIn("ORIGINAL_ACCEPTED", self.content)
        self.assertIn("ERR_ORIGINAL_IMMUTABLE", self.content)
        self.assertIn("DERIVED_OBJECT", self.content)
        self.assertIn("ERR_DERIVED_CANNOT_REPLACE_ORIGINAL", self.content)

    def test_cryptographic_integrity_and_duplicate_upload(self) -> None:
        """Asserts SHA-256 digest validation, tamper detection, and duplicate handling."""
        self.assertIn("Continuous Digest Verification", self.content)
        self.assertIn("ERR_DIGEST_TAMPER_DETECTED", self.content)
        self.assertIn("ACK_IDEMPOTENT_DUPLICATE", self.content)
        self.assertIn("ERR_IDEMPOTENCY_CONFLICT", self.content)

    def test_interruption_resumption_and_export_manifests(self) -> None:
        """Asserts chunked upload resumption and export manifest verification."""
        self.assertIn("Chunked Upload Interruption Resilience", self.content)
        self.assertIn("manifest.json", self.content)
        self.assertIn("ERR_EXPORT_MANIFEST_INVALID", self.content)

    def test_barcode_and_qr_security_defenses(self) -> None:
        """Asserts cardinal scan security invariant, injection defense, and non-enumeration."""
        self.assertIn("A Scan is Never Authority", self.content)
        self.assertIn("Zero Inherent Authorization", self.content)
        self.assertIn("ERR_SCAN_PAYLOAD_MALICIOUS", self.content)
        self.assertIn("ERR_SCAN_RESOURCE_NOT_FOUND", self.content)

    def test_non_substitution_invariant_and_retained_holds(self) -> None:
        """Asserts non-substitution invariant and exclusion of empirical user evidence claims."""
        self.assertIn("Non-Substitution Invariant", self.content)
        self.assertIn("cannot substitute for, replace, or claim the status of real participant", self.content)
        self.assertIn("H040-008", self.content)

    def test_synthetic_scenarios_fixture_parsing(self) -> None:
        """Validates the synthetic qualification scenarios YAML fixture block."""
        yaml_blocks = re.findall(r"```yaml\s*\n(.*?)\n```", self.content, re.DOTALL)
        found_fixture = False
        for block in yaml_blocks:
            try:
                parsed = yaml.safe_load(block)
                if isinstance(parsed, dict) and parsed.get("fixture_id") == "fix_syn_evidence_scan_qualification_v1":
                    scenarios = parsed.get("scenarios", [])
                    self.assertGreaterEqual(len(scenarios), 8, "Must define at least 8 qualification scenarios")
                    scenario_ids = [s.get("id") for s in scenarios]
                    for expected_id in [
                        "QLF-EVD-01",
                        "QLF-EVD-02",
                        "QLF-EVD-03",
                        "QLF-EVD-04",
                        "QLF-EVD-05",
                        "QLF-EVD-06",
                        "QLF-EVD-07",
                        "QLF-EVD-08",
                    ]:
                        self.assertIn(expected_id, scenario_ids, f"Scenario {expected_id} missing in fixture")
                    found_fixture = True
                    break
            except Exception:
                continue
        self.assertTrue(found_fixture, "Synthetic evidence/scan qualification scenarios fixture YAML block not found")


if __name__ == "__main__":
    unittest.main()
