from __future__ import annotations
import pathlib
import unittest

ROOT = pathlib.Path(__file__).resolve().parents[1]
DOC = ROOT / "docs" / "architecture" / "v050-supported-alpha-profile-baseline.md"

class V050SupportedAlphaProfileBaselineTests(unittest.TestCase):
    def setUp(self) -> None:
        self.assertTrue(DOC.is_file())
        self.content = DOC.read_text(encoding="utf-8")
    def test_identity_and_boundary(self) -> None:
        for value in ("ARC-V050-PROF-001", "V050-I003 / GitHub Issue #154 (closed planning record)", "HDEC-V050-E003-EXECUTION-ACTIVATION-014", "PROFILE_SPECIFICATION_ONLY_NO_PARTICIPANT_ACTIVITY_RUNTIME_QUALIFICATION_OR_RELEASE_CREDIT"):
            self.assertIn(value, self.content)
    def test_synthetic_and_profile_constraints(self) -> None:
        for value in ("no participant", "synthetic identifiers", "raw credentials", "biometric values", "cross-tenant exports", "English (`en-US`)", "Thai (`th-TH`)", "Asia/Bangkok", "keyboard accessibility"):
            self.assertIn(value, self.content)
    def test_holds_config_and_nonclaims(self) -> None:
        for n in range(3, 9): self.assertIn(f"V050-CFG-{n:03d}", self.content)
        for n in range(9, 14): self.assertIn(f"H050-{n:03d}", self.content)
        for value in ("[NON-BINDING_PROPOSED]", "[TBD]", "remain HOLD", "AI has no autonomous protected decision authority"): self.assertIn(value, self.content)
    def test_readme_registrations(self) -> None:
        self.assertIn("v050-supported-alpha-profile-baseline.md", (ROOT / "docs" / "architecture" / "README.md").read_text(encoding="utf-8"))
        self.assertIn("test_v050_supported_alpha_profile_baseline.py", (ROOT / "tests" / "release" / "README.md").read_text(encoding="utf-8"))

if __name__ == "__main__": unittest.main()
