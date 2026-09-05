from __future__ import annotations

import pathlib
import re
import unittest
from typing import Any, Dict, List, Optional

ROOT = pathlib.Path(__file__).resolve().parents[1]
BASELINE_DOC_PATH = ROOT / "docs" / "architecture" / "v040-domain-data-authority-baseline.md"

FRONTMATTER_PATTERN = re.compile(r"^---\s*\n(.*?)\n---\s*\n", re.DOTALL)


def parse_simple_frontmatter(content: str) -> Dict[str, Any]:
    match = FRONTMATTER_PATTERN.match(content)
    if not match:
        raise ValueError("Missing YAML frontmatter")
    lines = match.group(1).splitlines()
    data: Dict[str, Any] = {}
    current_key: Optional[str] = None
    sub_key: Optional[str] = None

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

        # Sub-dictionary items (indented key: value)
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
                # Could be list or dict
                if key == "retained_unselected_policies":
                    data[key] = {}
                else:
                    data[key] = []
                current_key = key
            else:
                data[key] = val
                current_key = None
    return data


class V040DomainDataAuthorityBaselineTests(unittest.TestCase):
    def setUp(self) -> None:
        self.assertTrue(BASELINE_DOC_PATH.is_file(), f"Missing baseline document at {BASELINE_DOC_PATH}")
        self.content = BASELINE_DOC_PATH.read_text(encoding="utf-8")
        self.frontmatter = parse_simple_frontmatter(self.content)

    def test_document_metadata_and_governance(self) -> None:
        """Asserts required governed frontmatter fields, status, and authority link."""
        self.assertEqual(self.frontmatter.get("document_id"), "ARC-V040-DOMAIN-001")
        self.assertEqual(self.frontmatter.get("document_type"), "architecture_specification")
        self.assertEqual(self.frontmatter.get("lifecycle_status"), "DRAFT")
        self.assertEqual(self.frontmatter.get("status"), "APPROVED_FOR_LOCAL_ALPHA_DEVELOPMENT")
        self.assertEqual(self.frontmatter.get("authority_source"), "HDEC-V040-FOUNDATION-054")
        self.assertEqual(self.frontmatter.get("governing_issue"), "GitHub Issue #113")
        self.assertEqual(self.frontmatter.get("milestone"), "v0.4.0 - OSHE Inspect Private Alpha")

        approved = self.frontmatter.get("approved_foundation_gates", [])
        for g in ["H040-001", "H040-002", "H040-003", "H040-004", "H040-005", "H040-006"]:
            self.assertIn(g, approved, f"{g} must be present in approved_foundation_gates")

        deferred = self.frontmatter.get("deferred_human_gates", [])
        for h in ["H040-007", "H040-008", "H040-009", "H040-010", "H040-011"]:
            self.assertIn(h, deferred, f"{h} must be present in deferred_human_gates")

    def test_retained_unselected_policies_explicitly_marked(self) -> None:
        """Asserts binding scoring, finding closure, and offline authority are retained as unselected."""
        unselected = self.frontmatter.get("retained_unselected_policies", {})
        self.assertEqual(unselected.get("binding_scoring_policy"), "HUMAN_OWNED_UNSELECTED")
        self.assertEqual(unselected.get("finding_closure_policy"), "HUMAN_OWNED_UNSELECTED")
        self.assertEqual(unselected.get("offline_authority"), "HUMAN_OWNED_UNSELECTED")

        # Confirm documented in body text
        self.assertIn("Binding Scoring Policy", self.content)
        self.assertIn("Finding Closure Policy", self.content)
        self.assertIn("Offline Authority Model", self.content)
        self.assertIn("HUMAN_OWNED_UNSELECTED", self.content)

    def test_explicit_module_ownership(self) -> None:
        """Asserts all 7 core modules and domain entity boundaries are documented."""
        modules = ["MOD-CFG", "MOD-WFA", "MOD-EVD", "MOD-REP", "MOD-REC", "MOD-IAM", "MOD-ORG"]
        for m in modules:
            self.assertIn(m, self.content, f"Module {m} must be present in ownership matrix")

        # Default-deny and four roles
        self.assertIn("Default-Deny", self.content)
        roles = ["Checklist Author", "Inspector", "CAPA Owner", "Independent Reviewer"]
        for r in roles:
            self.assertIn(r, self.content, f"Role {r} must be documented")

        # Zero AI safety autonomy
        self.assertIn("AI has zero autonomous safety decision authority", self.content)

    def test_deterministic_workflow_states(self) -> None:
        """Asserts explicit lifecycle state machines and protected transitions."""
        # Checklist states
        for st in ["DRAFT", "UNDER_REVIEW", "APPROVED", "PUBLISHED", "RETIRED", "SUPERSEDED"]:
            self.assertIn(st, self.content, f"Checklist state {st} missing")

        # Inspection states
        for st in ["SCHEDULED", "ASSIGNED", "IN_PROGRESS", "COMPLETED", "FINALIZED", "REJECTED"]:
            self.assertIn(st, self.content, f"Inspection state {st} missing")

        # Finding & CAPA states
        for st in ["IDENTIFIED", "ACTION_ASSIGNED", "EVIDENCE_SUBMITTED", "VERIFIED_CLOSED", "REOPENED"]:
            self.assertIn(st, self.content, f"Finding state {st} missing")

        self.assertIn("Protected Transitions", self.content)

    def test_server_authority_and_conflict_quarantine(self) -> None:
        """Asserts server authority, zero last-write-wins, and conflict quarantine."""
        self.assertIn("Server Authority for Protected State", self.content)
        self.assertIn("zero last-write-wins", self.content)
        self.assertIn("QUARANTINED", self.content)
        self.assertIn("manual reconciliation", self.content)

    def test_public_contracts_and_domain_events(self) -> None:
        """Asserts public contract mapping and domain event specifications."""
        self.assertIn("contracts/api/checklist/v1", self.content)
        self.assertIn("contracts/api/inspection/v1", self.content)
        self.assertIn("contracts/api/finding/v1", self.content)
        self.assertIn("contracts/api/action/v1", self.content)
        self.assertIn("contracts/api/evidence/v1", self.content)

        self.assertIn("ChecklistPublishedEvent", self.content)
        self.assertIn("InspectionCompletedEvent", self.content)
        self.assertIn("FindingIdentifiedEvent", self.content)
        self.assertIn("ConflictQuarantinedEvent", self.content)

    def test_retained_holds_and_prohibitions(self) -> None:
        """Asserts all five retained holds and operational prohibitions."""
        for h in ["H040-007", "H040-008", "H040-009", "H040-010", "H040-011"]:
            self.assertIn(h, self.content)
        self.assertIn("HOLD", self.content)

        self.assertIn("Zero public routes", self.content)
        self.assertIn("Zero live cloud deployments", self.content)
        self.assertIn("Zero customer data", self.content)


if __name__ == "__main__":
    unittest.main()
