from __future__ import annotations

import pathlib
import re
import unittest
from typing import Any, Dict, List, Optional

ROOT = pathlib.Path(__file__).resolve().parents[1]
QUALIFICATION_BASELINE_PATH = ROOT / "docs" / "qualification" / "v040-checklist-qualification-baseline.md"

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
        if stripped.startswith("- "):
            if current_key and isinstance(data.get(current_key), list):
                data[current_key].append(stripped[2:].strip().strip("\"'"))
            continue
        if ":" in line:
            key, val = line.split(":", 1)
            key = key.strip()
            val = val.strip().strip("\"'")
            if val == "":
                data[key] = []
                current_key = key
            else:
                data[key] = val
                current_key = None
    return data


class V040ChecklistQualificationBaselineTests(unittest.TestCase):
    def setUp(self) -> None:
        self.assertTrue(
            QUALIFICATION_BASELINE_PATH.is_file(),
            f"Missing checklist qualification baseline at {QUALIFICATION_BASELINE_PATH}",
        )
        self.content = QUALIFICATION_BASELINE_PATH.read_text(encoding="utf-8")
        self.frontmatter = parse_simple_frontmatter(self.content)

    def test_qualification_document_metadata_and_frontmatter(self) -> None:
        """Asserts required governed frontmatter fields, identifiers, and authority linkages."""
        self.assertEqual(self.frontmatter.get("document_id"), "QLF-V040-CHKL-001")
        self.assertEqual(self.frontmatter.get("document_type"), "qualification_specification")
        self.assertEqual(self.frontmatter.get("lifecycle_status"), "DRAFT")
        self.assertEqual(self.frontmatter.get("status"), "APPROVED_FOR_LOCAL_DEVELOPMENT")
        self.assertEqual(self.frontmatter.get("governing_issue"), "GitHub Issue #119")
        self.assertEqual(self.frontmatter.get("authority_source"), "HDEC-V040-FOUNDATION-054")
        self.assertEqual(
            self.frontmatter.get("credit_boundary"),
            "TECHNICAL_QUALIFICATION_ONLY_NO_USER_EVIDENCE_OR_RELEASE_CREDIT",
        )

        approved_gates = self.frontmatter.get("approved_foundation_gates", [])
        for gate in ["H040-001", "H040-002", "H040-003", "H040-004", "H040-005", "H040-006"]:
            self.assertIn(gate, approved_gates, f"Missing approved gate {gate}")

        retained_holds = self.frontmatter.get("retained_holds", [])
        for hold in ["H040-007", "H040-008", "H040-009", "H040-010", "H040-011"]:
            self.assertIn(hold, retained_holds, f"Missing retained hold {hold}")

    def test_qualification_test_domain_coverage(self) -> None:
        """Asserts technical testing covers all required test domains from QDOM-01 to QDOM-11."""
        domains = [
            "QDOM-01",
            "QDOM-02",
            "QDOM-03",
            "QDOM-04",
            "QDOM-05",
            "QDOM-06",
            "QDOM-07",
            "QDOM-08",
            "QDOM-09",
            "QDOM-10",
            "QDOM-11",
        ]
        for domain in domains:
            self.assertIn(domain, self.content, f"Missing domain {domain}")

        # Core operational topics
        self.assertIn("authoring", self.content.lower())
        self.assertIn("lifecycle", self.content.lower())
        self.assertIn("concurrency", self.content.lower())
        self.assertIn("provenance", self.content.lower())
        self.assertIn("localization", self.content.lower())
        self.assertIn("execution pinning", self.content.lower())

    def test_supported_six_question_types_and_bilingual_localization(self) -> None:
        """Asserts strict validation across the six alpha question types and dual-language localization."""
        expected_types = [
            "PASS_FAIL_NA_UNKNOWN",
            "SINGLE_CHOICE",
            "MULTI_CHOICE",
            "NUMERIC_MEASUREMENT",
            "TEXT_NOTE",
            "EVIDENCE_ATTACHMENT",
        ]
        for q_type in expected_types:
            self.assertIn(q_type, self.content, f"Missing question type {q_type}")

        # Bilingual localization
        self.assertIn("en-US", self.content)
        self.assertIn("th-TH", self.content)
        self.assertIn("Zero Monolingual Content", self.content)

        # UNKNOWN and NA per H040-006
        self.assertIn("UNKNOWN", self.content)
        self.assertIn("NA", self.content)
        self.assertIn("Mandatory non-blank justification note on `NA` and `UNKNOWN`", self.content)

    def test_non_substitution_invariant_and_evidence_separation(self) -> None:
        """Asserts explicit boundary that technical qualification does not substitute for real user evidence."""
        self.assertIn("Non-Substitution Invariant", self.content)
        self.assertIn(
            "Under no circumstances may simulated agent runs, automated test executions, or synthetic test payloads substitute for",
            self.content,
        )
        self.assertIn("Gate `H040-008`", self.content)
        self.assertIn("HOLD", self.content)

    def test_concurrency_provenance_and_execution_pinning(self) -> None:
        """Asserts server authority, no-LWW conflict quarantine, provenance derivation, and execution pinning."""
        # Concurrency
        self.assertIn("Server Sole Authority", self.content)
        self.assertIn("Rejection of Last-Write-Wins", self.content)
        self.assertIn("QUARANTINED_CONFLICT", self.content)

        # Provenance
        self.assertIn("predecessor_version", self.content)
        self.assertIn("derived_from_digest", self.content)
        self.assertIn("SemVer Progression", self.content)
        self.assertIn("content_digest", self.content)

        # Execution Pinning
        self.assertIn("Execution Pinning", self.content)
        self.assertIn("Zero Runtime Hot-Swapping", self.content)
        self.assertIn("Historical Audit Repeatability", self.content)

    def test_negative_controls_and_safe_failure_modes(self) -> None:
        """Asserts deterministic negative controls from NEG-01 through NEG-10."""
        for neg in ["NEG-01", "NEG-02", "NEG-03", "NEG-04", "NEG-05", "NEG-06", "NEG-07", "NEG-08", "NEG-09", "NEG-10"]:
            self.assertIn(neg, self.content, f"Missing negative control {neg}")

        self.assertIn("SOD-03", self.content)
        self.assertIn("Self-Approval Denial", self.content)
        self.assertIn("ErrSelfApprovalProhibited", self.content)
        self.assertIn("ErrImmutableVersionMutation", self.content)

    def test_retained_holds_and_anti_scope_prohibitions(self) -> None:
        """Verifies strict retention of H040-007 through H040-011 holds and explicit anti-scope prohibitions."""
        for hold_id in ["H040-007", "H040-008", "H040-009", "H040-010", "H040-011"]:
            self.assertIn(hold_id, self.content)

        self.assertIn("Zero External Public Routes", self.content)
        self.assertIn("Zero CDN Edge Deployment", self.content)
        self.assertIn("Zero Production Database Deployment", self.content)
        self.assertIn("Zero Real Customer or Personal Data", self.content)
        self.assertIn("Zero Provider, Credential, or Account Mutations", self.content)


if __name__ == "__main__":
    unittest.main()
