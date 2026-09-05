from __future__ import annotations

import pathlib
import re
import unittest
import yaml
from typing import Any, Dict, List, Optional

ROOT = pathlib.Path(__file__).resolve().parents[1]
BASELINE_DOC_PATH = ROOT / "docs" / "architecture" / "v040-offline-conflict-baseline.md"

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


class V040OfflineConflictBaselineTests(unittest.TestCase):
    def setUp(self) -> None:
        self.assertTrue(BASELINE_DOC_PATH.is_file(), f"Missing baseline document at {BASELINE_DOC_PATH}")
        self.content = BASELINE_DOC_PATH.read_text(encoding="utf-8")
        self.frontmatter = parse_simple_frontmatter(self.content)

    def test_document_metadata_and_governance(self) -> None:
        """Asserts required governed frontmatter fields, status, and authority link."""
        self.assertEqual(self.frontmatter.get("document_id"), "ARC-V040-CONFLICT-001")
        self.assertEqual(self.frontmatter.get("document_type"), "architecture_specification")
        self.assertEqual(self.frontmatter.get("lifecycle_status"), "DRAFT")
        self.assertEqual(self.frontmatter.get("status"), "APPROVED_FOR_LOCAL_DEVELOPMENT")
        self.assertEqual(self.frontmatter.get("authority_source"), "HDEC-V040-FOUNDATION-054")
        self.assertEqual(self.frontmatter.get("governing_issue"), "GitHub Issue #126")
        self.assertEqual(self.frontmatter.get("milestone"), "v0.4.0 - OSHE Inspect Private Alpha")

        approved = self.frontmatter.get("approved_foundation_gates", [])
        for g in ["H040-001", "H040-002", "H040-003", "H040-004", "H040-005", "H040-006"]:
            self.assertIn(g, approved, f"{g} must be present in approved_foundation_gates")

        deferred = self.frontmatter.get("retained_holds", [])
        for h in ["H040-007", "H040-008", "H040-009", "H040-010", "H040-011"]:
            self.assertIn(h, deferred, f"{h} must be present in retained_holds")

    def test_retained_unselected_policies_explicitly_marked(self) -> None:
        """Asserts offline authority and scoring/closure policies are marked HUMAN_OWNED_UNSELECTED."""
        unselected = self.frontmatter.get("retained_unselected_policies", {})
        self.assertEqual(unselected.get("offline_authority"), "HUMAN_OWNED_UNSELECTED")
        self.assertEqual(unselected.get("binding_scoring_policy"), "HUMAN_OWNED_UNSELECTED")
        self.assertEqual(unselected.get("finding_closure_policy"), "HUMAN_OWNED_UNSELECTED")

        self.assertIn("HUMAN_OWNED_UNSELECTED", self.content)
        self.assertIn("Offline Authority Model", self.content)

    def test_c0_through_c5_classification_catalog(self) -> None:
        """Asserts all six conflict classes (C0 through C5) are fully defined and cataloged."""
        classes = ["`C0`", "`C1`", "`C2`", "`C3`", "`C4`", "`C5`"]
        for c in classes:
            self.assertIn(c, self.content, f"Conflict class {c} must be cataloged")

        self.assertIn("Idempotent / Benign Duplicate", self.content)
        self.assertIn("Non-Overlapping Additive Merge", self.content)
        self.assertIn("Stale Base-Version Concurrent Edit", self.content)
        self.assertIn("Competing Workflow State Transition", self.content)
        self.assertIn("Revoked / Expired Authority at Sync", self.content)
        self.assertIn("Cryptographic Integrity / Uncertainty", self.content)

    def test_server_authority_and_anti_lww_invariants(self) -> None:
        """Asserts server authority, prohibition of last-write-wins, and untrusted client timestamps."""
        self.assertIn("Server Authority (`H040-005`)", self.content)
        self.assertIn("Categorical Prohibition of Last-Write-Wins (LWW)", self.content)
        self.assertIn("Untrusted Client Time", self.content)
        self.assertIn("never use client-supplied timestamps", self.content)

    def test_protected_state_rejection(self) -> None:
        """Asserts protected state immutability and fail-closed rejection."""
        self.assertIn("ERR_PROTECTED_STATE_IMMUTABLE", self.content)
        self.assertIn("Protected-State Immutability", self.content)

    def test_conflict_quarantine_and_manual_reconciliation(self) -> None:
        """Asserts conflict quarantine schema and the four manual reconciliation actions."""
        self.assertIn("ConflictRecord", self.content)
        self.assertIn("PENDING_MANUAL_RECONCILIATION", self.content)

        # The four authorized actions
        self.assertIn("RESOLVE_ACCEPT_SERVER", self.content)
        self.assertIn("RESOLVE_OVERWRITE_WITH_CLIENT", self.content)
        self.assertIn("RESOLVE_MANUAL_MERGE", self.content)
        self.assertIn("RESOLVE_SPLIT_NEW_FINDING", self.content)
        self.assertIn("Mandatory Justification", self.content)

    def test_synthetic_scenarios_fixture_parsing(self) -> None:
        """Validates the synthetic conflict scenarios YAML fixture block."""
        yaml_blocks = re.findall(r"```yaml\s*\n(.*?)\n```", self.content, re.DOTALL)
        found_fixture = False
        for block in yaml_blocks:
            try:
                parsed = yaml.safe_load(block)
                if isinstance(parsed, dict) and parsed.get("fixture_id") == "fix_syn_conflict_scenarios_v1":
                    scenarios = parsed.get("scenarios", [])
                    self.assertGreaterEqual(len(scenarios), 5, "Must define scenarios for conflict catalog")
                    scenario_classes = {s.get("conflict_class") for s in scenarios}
                    self.assertIn("C0", scenario_classes)
                    self.assertIn("C2", scenario_classes)
                    self.assertIn("C3", scenario_classes)
                    self.assertIn("C4", scenario_classes)
                    self.assertIn("C5", scenario_classes)
                    found_fixture = True
                    break
            except Exception:
                continue
        self.assertTrue(found_fixture, "Synthetic conflict scenarios fixture YAML block not found")

    def test_operational_prohibitions_and_retained_holds(self) -> None:
        """Asserts retained holds H040-007 through H040-011 and non-production prohibitions."""
        for h in ["H040-007", "H040-008", "H040-009", "H040-010", "H040-011"]:
            self.assertIn(h, self.content)
        self.assertIn("HOLD", self.content)

        self.assertIn("Zero Last-Write-Wins", self.content)
        self.assertIn("Synthetic-Only Data Policy", self.content)


if __name__ == "__main__":
    unittest.main()
