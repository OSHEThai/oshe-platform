from __future__ import annotations

import pathlib
import re
import unittest
from typing import Any, Dict, List, Optional

ROOT = pathlib.Path(__file__).resolve().parents[1]
BASELINE_PATH = ROOT / "docs" / "security" / "v040-threat-privacy-hazard-baseline.md"

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


class V040SecurityBaselineTests(unittest.TestCase):
    def setUp(self) -> None:
        self.assertTrue(BASELINE_PATH.is_file(), f"Missing security baseline at {BASELINE_PATH}")
        self.content = BASELINE_PATH.read_text(encoding="utf-8")
        self.frontmatter = parse_simple_frontmatter(self.content)

    def test_baseline_document_existence_and_metadata(self) -> None:
        """Asserts required governed frontmatter fields and authority references."""
        self.assertEqual(self.frontmatter.get("document_id"), "SEC-V040-BASE-001")
        self.assertEqual(self.frontmatter.get("document_type"), "security_and_safety_baseline")
        self.assertEqual(self.frontmatter.get("lifecycle_status"), "DRAFT")
        self.assertEqual(self.frontmatter.get("status"), "APPROVED_FOR_LOCAL_ALPHA_DEVELOPMENT")
        self.assertEqual(self.frontmatter.get("governing_issue"), "GitHub Issue #145")
        self.assertEqual(self.frontmatter.get("governing_decision"), "HDEC-V040-FOUNDATION-054")
        self.assertEqual(self.frontmatter.get("milestone"), "v0.4.0 - OSHE Inspect Private Alpha")
        self.assertIn("NO_RELEASE_FALLBACK_OWNERSHIP_OR_RESIDUAL_RISK_ACCEPTANCE", self.frontmatter.get("credit_boundary", ""))

    def test_approved_foundation_gates_represented(self) -> None:
        """Asserts approved foundation gates H040-001 through H040-006 are covered."""
        approved = self.frontmatter.get("approved_foundation_gates", [])
        for gate in ["H040-001", "H040-002", "H040-003", "H040-004", "H040-005", "H040-006"]:
            self.assertIn(gate, approved, f"{gate} must be in approved_foundation_gates frontmatter")
            self.assertIn(gate, self.content, f"{gate} must be documented in body")

    def test_retained_holds_and_prohibitions_preserved(self) -> None:
        """Verifies strict retention of H040-007 through H040-011 holds and operational non-claims."""
        retained = self.frontmatter.get("retained_holds", [])
        for gate in ["H040-007", "H040-008", "H040-009", "H040-010", "H040-011"]:
            self.assertIn(gate, retained, f"{gate} must be in retained_holds frontmatter")
            self.assertIn(gate, self.content, f"{gate} must be documented in body")

        # Specifically assert H040-009 and H040-011 holds
        self.assertIn("H040-009", self.content)
        self.assertIn("H040-011", self.content)
        self.assertIn("HOLD", self.content)

        # Assert non-claims
        self.assertIn("Zero Release Recommendation or Approval", self.content)
        self.assertIn("Zero Real-User Authorization", self.content)
        self.assertIn("Zero Residual-Risk Acceptance", self.content)
        self.assertIn("Zero Operational Support Ownership", self.content)

    def test_threat_model_eleven_scenarios_coverage(self) -> None:
        """Asserts all 11 required threat and abuse scenarios from Issue #145 are modeled with mitigations."""
        required_threats = [
            "Tenant Leakage",
            "Privilege Escalation",
            "Malicious Evidence / QR",
            "Sync Replay & Race",
            "Stale Authority",
            "Data Loss",
            "Evidence Tampering",
            "Misleading Pass",
            "Lost Finding",
            "Unauthorized Closure",
            "Support / Fallback Failure",
        ]
        for threat in required_threats:
            self.assertIn(threat, self.content, f"Missing threat scenario: {threat}")

    def test_product_safety_hazard_classifications(self) -> None:
        """Asserts S0, S1, S2, S3 hazard classifications and S0/S1 critical controls."""
        for severity in ["S0", "S1", "S2", "S3"]:
            self.assertIn(severity, self.content, f"Missing severity level: {severity}")

        # S0/S1 critical controls
        self.assertIn("Catastrophic", self.content)
        self.assertIn("Critical", self.content)
        self.assertIn("Fail-Safe & Stop Behavior", self.content)
        self.assertIn("UNSAFE_FAILED", self.content)
        self.assertIn("Release-Blocking Treatment", self.content)

    def test_critical_functions_and_server_authority(self) -> None:
        """Asserts critical functions register, server authority, and conflict quarantine."""
        self.assertIn("FN-01", self.content)
        self.assertIn("FN-02", self.content)
        self.assertIn("FN-03", self.content)
        self.assertIn("FN-04", self.content)
        self.assertIn("FN-05", self.content)
        self.assertIn("FN-06", self.content)

        # Server authority & quarantine
        self.assertIn("Server Authority", self.content)
        self.assertIn("Conflict Quarantine", self.content)
        self.assertIn("QUARANTINED", self.content)
        self.assertIn("last-write-wins", self.content)

    def test_privacy_and_synthetic_data_controls(self) -> None:
        """Asserts synthetic data mandate and browser storage privacy controls."""
        self.assertIn("Synthetic Data Mandate", self.content)
        self.assertIn("H040-003", self.content)
        self.assertIn("usr_syn_", self.content)
        self.assertIn("ins_syn_", self.content)
        self.assertIn("Zero Raw Credentials", self.content)
        self.assertIn("Session Purge", self.content)

    def test_four_authorized_roles_and_ai_non_authority(self) -> None:
        """Asserts four authorized roles and AI non-authority invariant."""
        for role in ["Checklist Author", "Inspector", "CAPA Owner", "Independent Reviewer"]:
            self.assertIn(role, self.content, f"Missing role definition: {role}")

        self.assertIn("AI Non-Authority Invariant", self.content)
        self.assertIn("zero autonomous safety authority", self.content)


if __name__ == "__main__":
    unittest.main()
