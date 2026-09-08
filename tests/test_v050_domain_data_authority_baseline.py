from __future__ import annotations

import pathlib
import unittest


ROOT = pathlib.Path(__file__).resolve().parents[1]
DOC = ROOT / "docs" / "architecture" / "v050-domain-data-authority-baseline.md"


class V050DomainDataAuthorityBaselineTests(unittest.TestCase):
    def setUp(self) -> None:
        self.assertTrue(DOC.is_file(), f"Missing baseline document: {DOC}")
        self.content = DOC.read_text(encoding="utf-8")

    def test_identity_and_credit_boundary(self) -> None:
        self.assertIn("document_id: ARC-V050-DOMAIN-001", self.content)
        self.assertIn("V050-I002 / GitHub Issue #153 (closed planning record)", self.content)
        self.assertIn("HDEC-V050-E002-EXECUTION-ACTIVATION-008", self.content)
        self.assertIn("ARCHITECTURE_BASELINE_ONLY_NO_MODULE_CONTRACT_MIGRATION_RUNTIME_OR_RELEASE_CREDIT", self.content)

    def test_synthetic_only_and_retained_holds(self) -> None:
        self.assertIn("No real workforce, employment, reporter, witness", self.content)
        for config in range(3, 9):
            self.assertIn(f"V050-CFG-00{config}", self.content)
        for gate in range(9, 14):
            self.assertIn(f"H050-{gate:03d}", self.content)
        self.assertIn("remain HOLD", self.content)

    def test_ownership_and_default_deny(self) -> None:
        for phrase in ("Incident may consume a", "may not write Workforce state", "Workforce may\nnot write Incident state", "Cross-domain direct writes\nare forbidden", "**default-deny**", "cross-tenant"):
            self.assertIn(phrase, self.content)

    def test_protected_state_and_ai_boundary(self) -> None:
        for phrase in ("no last-write-wins behavior", "append-only\nhistorical context", "AI has no autonomous\nprotected decision authority", "UNCLASSIFIED", "TRIAGE_PROPOSED", "UNDER_REVIEW", "SUPERSEDED"):
            self.assertIn(phrase, self.content)

    def test_readme_registrations(self) -> None:
        self.assertIn("v050-domain-data-authority-baseline.md", (ROOT / "docs" / "architecture" / "README.md").read_text(encoding="utf-8"))
        self.assertIn("test_v050_domain_data_authority_baseline.py", (ROOT / "tests" / "release" / "README.md").read_text(encoding="utf-8"))


if __name__ == "__main__":
    unittest.main()
