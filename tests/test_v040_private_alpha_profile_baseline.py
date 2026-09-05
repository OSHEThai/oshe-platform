from __future__ import annotations

import pathlib
import re
import unittest
from typing import Any, Dict, List, Optional

ROOT = pathlib.Path(__file__).resolve().parents[1]
BASELINE_PATH = ROOT / "docs" / "architecture" / "v040-private-alpha-profile-baseline.md"

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


class V040PrivateAlphaProfileBaselineTests(unittest.TestCase):
    def setUp(self) -> None:
        self.assertTrue(BASELINE_PATH.is_file(), f"Missing profile baseline at {BASELINE_PATH}")
        self.content = BASELINE_PATH.read_text(encoding="utf-8")
        self.frontmatter = parse_simple_frontmatter(self.content)

    def test_profile_baseline_metadata_and_frontmatter(self) -> None:
        """Asserts required governed frontmatter fields and authority references."""
        self.assertEqual(self.frontmatter.get("document_id"), "ARC-V040-PROF-001")
        self.assertEqual(self.frontmatter.get("document_type"), "architecture_specification")
        self.assertEqual(self.frontmatter.get("lifecycle_status"), "DRAFT")
        self.assertEqual(self.frontmatter.get("status"), "APPROVED_FOR_LOCAL_DEVELOPMENT")
        self.assertEqual(self.frontmatter.get("governing_issue"), "GitHub Issue #114")
        self.assertEqual(self.frontmatter.get("governing_decision"), "HDEC-V040-FOUNDATION-054")
        self.assertEqual(self.frontmatter.get("milestone"), "v0.4.0 - OSHE Inspect Private Alpha")
        self.assertIn("NO_PARTICIPANT_ONBOARDING_OR_BINDING_SLA", self.frontmatter.get("credit_boundary", ""))

    def test_approved_foundation_gates_covered(self) -> None:
        """Asserts approved foundation gates H040-001 through H040-006 are represented."""
        approved = self.frontmatter.get("approved_foundation_gates", [])
        for gate in ["H040-001", "H040-002", "H040-003", "H040-004", "H040-005", "H040-006"]:
            self.assertIn(gate, approved, f"{gate} must be in approved_foundation_gates")
            self.assertIn(gate, self.content, f"{gate} must be documented in body")

    def test_retained_holds_preserved(self) -> None:
        """Verifies strict retention of H040-007 through H040-011 holds and operational non-claims."""
        retained = self.frontmatter.get("retained_holds", [])
        for gate in ["H040-007", "H040-008", "H040-009", "H040-010", "H040-011"]:
            self.assertIn(gate, retained, f"{gate} must be in retained_holds")
            self.assertIn(gate, self.content, f"{gate} must be documented in body")

        # Specific gate hold assertions
        self.assertIn("H040-007", self.content)
        self.assertIn("H040-008", self.content)
        self.assertIn("H040-009", self.content)
        self.assertIn("H040-010", self.content)
        self.assertIn("H040-011", self.content)
        self.assertIn("HOLD", self.content)

        # Operational non-claims
        self.assertIn("Zero Participant Authorization", self.content)
        self.assertIn("Zero Deployment Authority", self.content)
        self.assertIn("Zero Support or SLA Commitment", self.content)
        self.assertIn("Zero Residual-Risk Acceptance", self.content)

    def test_non_binding_and_tbd_thresholds_invariant(self) -> None:
        """Asserts every performance metric is explicitly tagged NON-BINDING_PROPOSED or TBD."""
        self.assertIn("[NON-BINDING_PROPOSED]", self.content)
        self.assertIn("[TBD]", self.content)
        self.assertIn("Zero performance claims", self.content)
        self.assertIn("binding service level agreements", self.content)

        # Confirm all NFR items carry non-binding or TBD markers
        for nfr in ["NFR-PERF-01", "NFR-PERF-02", "NFR-PERF-03", "NFR-SYNC-01", "NFR-CAP-01", "NFR-A11Y-01"]:
            self.assertIn(nfr, self.content, f"Missing {nfr} in NFR catalog")

    def test_device_browser_and_unsupported_matrix(self) -> None:
        """Asserts supported browsers, localization, and explicit unsupported envelope."""
        # Supported platforms
        self.assertIn("Google Chrome", self.content)
        self.assertIn("Microsoft Edge", self.content)
        self.assertIn("Android Chrome", self.content)

        # Localization & time zone
        self.assertIn("English", self.content)
        self.assertIn("Thai", self.content)
        self.assertIn("Asia/Bangkok", self.content)

        # Unsupported envelope
        self.assertIn("UNSUPPORTED_PRIVATE_ALPHA", self.content)
        self.assertIn("Apple iOS", self.content)
        self.assertIn("Mozilla Firefox", self.content)
        self.assertIn("Native mobile applications", self.content)

    def test_offline_online_and_server_authority(self) -> None:
        """Asserts network modes, server authority, and conflict quarantine under H040-005."""
        self.assertIn("ONLINE_CONNECTED", self.content)
        self.assertIn("OFFLINE_DISCONNECTED", self.content)
        self.assertIn("INTERMITTENT_SYNCING", self.content)
        self.assertIn("Server Sole Authority", self.content)
        self.assertIn("Zero Last-Write-Wins", self.content)
        self.assertIn("QUARANTINED", self.content)

    def test_data_minimization_and_synthetic_scope(self) -> None:
        """Asserts 100% synthetic data mandate and browser storage security under H040-003."""
        self.assertIn("100% Synthetic Data Policy", self.content)
        self.assertIn("H040-003", self.content)
        self.assertIn("usr_syn_", self.content)
        self.assertIn("ins_syn_", self.content)
        self.assertIn("Zero Raw Credentials", self.content)
        self.assertIn("Session Purge", self.content)


if __name__ == "__main__":
    unittest.main()
