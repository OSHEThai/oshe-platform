from __future__ import annotations

import pathlib
import unittest


ROOT = pathlib.Path(__file__).resolve().parents[1]
DOC = ROOT / "docs" / "architecture" / "v050-test-assurance-uat-support-recovery-evidence-plan.md"


class V050TestAssurancePlanTests(unittest.TestCase):
    def setUp(self) -> None:
        self.assertTrue(DOC.is_file())
        self.content = DOC.read_text(encoding="utf-8")

    def test_identity_and_plan_boundary(self) -> None:
        for value in (
            "ARC-V050-ASSURE-001",
            "V050-I004 / GitHub Issue #155 (closed planning record)",
            "HDEC-V050-E004-EXECUTION-ACTIVATION-020",
            "DETERMINISTIC_PLAN_ONLY_NO_PARTICIPANT_ACTIVITY_RUNTIME_QUALIFICATION_OR_RELEASE_CREDIT",
        ):
            self.assertIn(value, self.content)

    def test_evidence_and_assurance_constraints(self) -> None:
        for value in (
            "A test design is not test-execution evidence.",
            "A synthetic run is not human-validation evidence.",
            "No participant",
            "cross-tenant",
            "no last-write-wins",
            "quarantined",
            "raw credential",
            "[TBD]",
            "[NON-BINDING_PROPOSED]",
        ):
            self.assertIn(value, self.content)

    def test_config_holds_and_human_boundary(self) -> None:
        for n in range(3, 9):
            self.assertIn(f"V050-CFG-{n:03d}", self.content)
        for n in range(9, 14):
            self.assertIn(f"H050-{n:03d}", self.content)
        for value in (
            "remain UNSELECTED",
            "remain HOLD or NOT_DUE",
            "must not select, recruit, invite, onboard, or observe participants",
            "Under no circumstances may simulated",
            "AI has no autonomous protected decision authority",
        ):
            self.assertIn(value, self.content)

    def test_support_recovery_and_evidence_map(self) -> None:
        for value in (
            "default-deny intake",
            "manual fallback proposal",
            "restore proposal",
            "P1",
            "P4",
            "EVD-V050-01",
            "EVD-V050-08",
            "cannot earn participant, runtime, qualification, release, milestone-closure, or v0.6.0 credit",
        ):
            self.assertIn(value, self.content)

    def test_readme_registrations(self) -> None:
        self.assertIn(
            "v050-test-assurance-uat-support-recovery-evidence-plan.md",
            (ROOT / "docs" / "architecture" / "README.md").read_text(encoding="utf-8"),
        )
        self.assertIn(
            "test_v050_test_assurance_plan.py",
            (ROOT / "tests" / "release" / "README.md").read_text(encoding="utf-8"),
        )


if __name__ == "__main__":
    unittest.main()
