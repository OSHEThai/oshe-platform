from __future__ import annotations

import pathlib
import re
import unittest
import yaml
from typing import Any, Dict, List, Optional

ROOT = pathlib.Path(__file__).resolve().parents[1]
QUALIFICATION_BASELINE_PATH = (
    ROOT / "docs" / "qualification" / "v040-offline-qualification-baseline.md"
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


class V040OfflineQualificationBaselineTests(unittest.TestCase):
    def setUp(self) -> None:
        self.assertTrue(
            QUALIFICATION_BASELINE_PATH.is_file(),
            f"Missing offline qualification baseline at {QUALIFICATION_BASELINE_PATH}",
        )
        self.content = QUALIFICATION_BASELINE_PATH.read_text(encoding="utf-8")
        self.frontmatter = parse_simple_frontmatter(self.content)

    def test_document_metadata_and_governance(self) -> None:
        """Asserts required governed frontmatter fields, status, and authority link."""
        self.assertEqual(self.frontmatter.get("document_id"), "QLF-V040-OFFLINE-001")
        self.assertEqual(self.frontmatter.get("document_type"), "qualification_specification")
        self.assertEqual(self.frontmatter.get("lifecycle_status"), "DRAFT")
        self.assertEqual(self.frontmatter.get("status"), "APPROVED_FOR_LOCAL_DEVELOPMENT")
        self.assertEqual(self.frontmatter.get("authority_source"), "HDEC-V040-FOUNDATION-054")
        self.assertEqual(self.frontmatter.get("governing_issue"), "GitHub Issue #127")
        self.assertEqual(self.frontmatter.get("milestone"), "v0.4.0 - OSHE Inspect Private Alpha")
        self.assertEqual(
            self.frontmatter.get("credit_boundary"),
            "TECHNICAL_QUALIFICATION_ONLY_NO_USER_EVIDENCE_OR_RELEASE_CREDIT",
        )

        approved = self.frontmatter.get("approved_foundation_gates", [])
        for g in ["H040-001", "H040-002", "H040-003", "H040-004", "H040-005", "H040-006"]:
            self.assertIn(g, approved, f"{g} must be present in approved_foundation_gates")

        deferred = self.frontmatter.get("retained_holds", [])
        for h in ["H040-007", "H040-008", "H040-009", "H040-010", "H040-011"]:
            self.assertIn(h, deferred, f"{h} must be present in retained_holds")

    def test_retained_unselected_policies_explicitly_marked(self) -> None:
        """Asserts offline authority, scoring, and closure policies are marked HUMAN_OWNED_UNSELECTED."""
        unselected = self.frontmatter.get("retained_unselected_policies", {})
        self.assertEqual(unselected.get("offline_authority"), "HUMAN_OWNED_UNSELECTED")
        self.assertEqual(unselected.get("binding_scoring_policy"), "HUMAN_OWNED_UNSELECTED")
        self.assertEqual(unselected.get("finding_closure_policy"), "HUMAN_OWNED_UNSELECTED")

        self.assertIn("HUMAN_OWNED_UNSELECTED", self.content)

    def test_eight_offline_qualification_dimensions(self) -> None:
        """Asserts all eight offline qualification dimensions are documented."""
        self.assertIn("Dimension 1: Supported Device & Viewport Boundary", self.content)
        self.assertIn("Dimension 2: Reference Data Freshness & 24-Hour Lease Expiration", self.content)
        self.assertIn("Dimension 3: Offline Scope & Authorization Enforcement", self.content)
        self.assertIn("Dimension 4: Protected Local Storage Partitioning", self.content)
        self.assertIn("Dimension 5: Abrupt Interruption & Crash Recovery", self.content)
        self.assertIn("Dimension 6: Synchronization State Machine", self.content)
        self.assertIn("Dimension 7: C0–C5 Conflict Classification & Quarantine", self.content)
        self.assertIn("Dimension 8: Visible Human Reconciliation & User Guidance", self.content)

    def test_supported_and_unsupported_platforms(self) -> None:
        """Asserts supported Chrome/Edge platforms and rejection of unsupported environments."""
        self.assertIn("Android 12+", self.content)
        self.assertIn("Google Chrome Mobile", self.content)
        self.assertIn("DENIAL_UNSUPPORTED_DEVICE", self.content)
        self.assertIn("Apple iOS / iPadOS", self.content)
        self.assertIn("Mozilla Firefox", self.content)

    def test_lease_expiration_and_storage_defenses(self) -> None:
        """Asserts 24-hour lease expiration and low-storage defenses."""
        self.assertIn("ERR_OFFLINE_LEASE_EXPIRED", self.content)
        self.assertIn("24-Hour Maximum Offline Lease Ceiling", self.content)
        self.assertIn("ERR_STORAGE_QUOTA_CRITICAL", self.content)
        self.assertIn("15%", self.content)

    def test_c0_through_c5_conflict_classification_and_anti_lww(self) -> None:
        """Asserts all six conflict classes and server authority anti-LWW rules."""
        for c in ["`C0`", "`C1`", "`C2`", "`C3`", "`C4`", "`C5`"]:
            self.assertIn(c, self.content, f"Conflict class {c} missing")

        self.assertIn("Zero Last-Write-Wins", self.content)
        self.assertIn("Server Authority Invariant", self.content)

    def test_non_substitution_and_non_claims_invariants(self) -> None:
        """Asserts non-substitution invariant and exclusion of empirical user evidence claims."""
        self.assertIn("Non-Substitution Invariant", self.content)
        self.assertIn("cannot substitute for, replace, or claim the status of empirical real-user evidence", self.content)
        self.assertIn("H040-008", self.content)

    def test_synthetic_scenarios_fixture_parsing(self) -> None:
        """Validates the synthetic qualification scenarios YAML fixture block."""
        yaml_blocks = re.findall(r"```yaml\s*\n(.*?)\n```", self.content, re.DOTALL)
        found_fixture = False
        for block in yaml_blocks:
            try:
                parsed = yaml.safe_load(block)
                if isinstance(parsed, dict) and parsed.get("fixture_id") == "fix_syn_offline_qualification_v1":
                    scenarios = parsed.get("scenarios", [])
                    self.assertGreaterEqual(len(scenarios), 8, "Must define at least 8 qualification scenarios")
                    scenario_ids = [s.get("id") for s in scenarios]
                    for expected_id in ["QLF-OFF-01", "QLF-OFF-02", "QLF-OFF-03", "QLF-OFF-04", "QLF-OFF-05", "QLF-OFF-06", "QLF-OFF-07", "QLF-OFF-08"]:
                        self.assertIn(expected_id, scenario_ids, f"Scenario {expected_id} missing in fixture")
                    found_fixture = True
                    break
            except Exception:
                continue
        self.assertTrue(found_fixture, "Synthetic offline qualification scenarios fixture YAML block not found")


if __name__ == "__main__":
    unittest.main()
