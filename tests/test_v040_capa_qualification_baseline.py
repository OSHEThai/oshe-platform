from __future__ import annotations

import pathlib
import re
import unittest
import yaml
from typing import Any, Dict, List, Optional

ROOT = pathlib.Path(__file__).resolve().parents[1]
QUALIFICATION_BASELINE_PATH = (
    ROOT / "docs" / "qualification" / "v040-capa-qualification-baseline.md"
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


class V040CAPAQualificationBaselineTests(unittest.TestCase):
    def setUp(self) -> None:
        self.assertTrue(
            QUALIFICATION_BASELINE_PATH.is_file(),
            f"Missing CAPA qualification baseline at {QUALIFICATION_BASELINE_PATH}",
        )
        self.content = QUALIFICATION_BASELINE_PATH.read_text(encoding="utf-8")
        self.frontmatter = parse_simple_frontmatter(self.content)

    def test_document_metadata_and_governance(self) -> None:
        """Asserts required governed frontmatter fields, status, and authority link."""
        self.assertEqual(self.frontmatter.get("document_id"), "QLF-V040-CAPA-001")
        self.assertEqual(self.frontmatter.get("document_type"), "qualification_specification")
        self.assertEqual(self.frontmatter.get("lifecycle_status"), "DRAFT")
        self.assertEqual(self.frontmatter.get("status"), "APPROVED_FOR_LOCAL_DEVELOPMENT")
        self.assertEqual(self.frontmatter.get("authority_source"), "HDEC-V040-FOUNDATION-054")
        self.assertEqual(self.frontmatter.get("governing_issue"), "GitHub Issue #135")
        self.assertEqual(self.frontmatter.get("milestone"), "v0.4.0 - OSHE Inspect Private Alpha")
        self.assertEqual(
            self.frontmatter.get("credit_boundary"),
            "TECHNICAL_QUALIFICATION_ONLY_NO_REAL_OWNER_OR_RELEASE_CREDIT",
        )

        approved = self.frontmatter.get("approved_foundation_gates", [])
        for g in ["H040-001", "H040-002", "H040-003", "H040-004", "H040-005", "H040-006"]:
            self.assertIn(g, approved, f"{g} must be present in approved_foundation_gates")

        deferred = self.frontmatter.get("retained_holds", [])
        for h in ["H040-007", "H040-008", "H040-009", "H040-010", "H040-011"]:
            self.assertIn(h, deferred, f"{h} must be present in retained_holds")

    def test_retained_unselected_policies_explicitly_marked(self) -> None:
        """Asserts finding closure and related policies are marked HUMAN_OWNED_UNSELECTED."""
        unselected = self.frontmatter.get("retained_unselected_policies", {})
        self.assertEqual(unselected.get("finding_closure_policy"), "HUMAN_OWNED_UNSELECTED")
        self.assertEqual(unselected.get("binding_scoring_policy"), "HUMAN_OWNED_UNSELECTED")
        self.assertEqual(unselected.get("offline_authority"), "HUMAN_OWNED_UNSELECTED")
        self.assertEqual(unselected.get("binding_due_date_policy"), "HUMAN_OWNED_UNSELECTED")
        self.assertEqual(unselected.get("binding_extension_policy"), "HUMAN_OWNED_UNSELECTED")
        self.assertEqual(unselected.get("binding_escalation_policy"), "HUMAN_OWNED_UNSELECTED")
        self.assertEqual(unselected.get("independent_review_policy"), "HUMAN_OWNED_UNSELECTED")
        self.assertEqual(unselected.get("reinspection_criteria_policy"), "HUMAN_OWNED_UNSELECTED")

        self.assertIn("HUMAN_OWNED_UNSELECTED", self.content)

    def test_finding_creation_severity_and_immediate_controls(self) -> None:
        """Asserts finding severity catalog, critical flag, and immediate control rules."""
        for sev in ["LOW", "MEDIUM", "HIGH", "CRITICAL"]:
            self.assertIn(sev, self.content, f"Severity {sev} missing")

        self.assertIn("critical_flag", self.content)
        self.assertIn("ErrImmediateControlRequired", self.content)
        self.assertIn("recurrence_id", self.content)

    def test_action_ownership_and_missing_owner_rejection(self) -> None:
        """Asserts action ownership requirement, missing owner rejection, and custody history."""
        self.assertIn("assigned_owner_id", self.content)
        self.assertIn("ErrMissingActionOwner", self.content)
        self.assertIn("Unbroken Custody & Reassignment Audit", self.content)

    def test_due_dates_extensions_and_escalation(self) -> None:
        """Asserts due date constraints, extension rejection, and overdue escalation."""
        self.assertIn("ErrInvalidExtensionRequest", self.content)
        self.assertIn("ErrExtensionPastMaximumWindow", self.content)
        self.assertIn("ErrActionOverdueEscalation", self.content)

    def test_evidence_submission_and_independent_review(self) -> None:
        """Asserts mandatory remediation evidence, self-review prohibition, and rejection."""
        self.assertIn("ErrRemediationEvidenceMissing", self.content)
        self.assertIn("ErrSelfReviewForbidden", self.content)
        self.assertIn("ErrRemediationEvidenceRejected", self.content)

    def test_reinspection_concurrency_and_anti_lww(self) -> None:
        """Asserts optimistic concurrency control, stale-state conflict, and anti-LWW."""
        self.assertIn("StateVersion", self.content)
        self.assertIn("ErrStaleStateConflict", self.content)
        self.assertIn("Zero Last-Write-Wins", self.content)

    def test_unauthorized_stale_and_offline_closure_denial(self) -> None:
        """Asserts offline closure prohibition, unauthorized closure rejection, and stale closure."""
        self.assertIn("ErrOfflineClosureForbidden", self.content)
        self.assertIn("ErrUnauthorizedClosure", self.content)
        self.assertIn("ErrStaleVersionAtClosure", self.content)

    def test_reopening_workflow_and_hazard_tracking(self) -> None:
        """Asserts supervisory reopening and child reinspection staging."""
        self.assertIn("Supervisory Reopening Protocol", self.content)
        self.assertIn("StatusReopenedPendingReinspection", self.content)

    def test_non_substitution_invariant_and_retained_holds(self) -> None:
        """Asserts non-substitution invariant and exclusion of empirical owner evidence claims."""
        self.assertIn("Non-Substitution Invariant", self.content)
        self.assertIn("cannot substitute for, replace, or claim the status of empirical real-world owner", self.content)
        self.assertIn("H040-008", self.content)

    def test_synthetic_scenarios_fixture_parsing(self) -> None:
        """Validates the synthetic qualification scenarios YAML fixture block."""
        yaml_blocks = re.findall(r"```yaml\s*\n(.*?)\n```", self.content, re.DOTALL)
        found_fixture = False
        for block in yaml_blocks:
            try:
                parsed = yaml.safe_load(block)
                if isinstance(parsed, dict) and parsed.get("fixture_id") == "fix_syn_capa_qualification_v1":
                    scenarios = parsed.get("scenarios", [])
                    self.assertGreaterEqual(len(scenarios), 8, "Must define at least 8 qualification scenarios")
                    scenario_ids = [s.get("id") for s in scenarios]
                    for expected_id in [
                        "QLF-CAPA-01",
                        "QLF-CAPA-02",
                        "QLF-CAPA-03",
                        "QLF-CAPA-04",
                        "QLF-CAPA-05",
                        "QLF-CAPA-06",
                        "QLF-CAPA-07",
                        "QLF-CAPA-08",
                    ]:
                        self.assertIn(expected_id, scenario_ids, f"Scenario {expected_id} missing in fixture")
                    found_fixture = True
                    break
            except Exception:
                continue
        self.assertTrue(found_fixture, "Synthetic CAPA qualification scenarios fixture YAML block not found")


if __name__ == "__main__":
    unittest.main()
