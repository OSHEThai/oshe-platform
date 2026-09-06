#!/usr/bin/env python3
"""
Baseline governance validation for V040-I027:
Fail-Closed Priority Hierarchy, Unknown-Quarantine Handling, Deferred Override Boundary,
Autonomous AI Boundary, and Append-Only Audit Protection.
"""

import os
import unittest
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parent.parent


class TestV040FailClosedGovernanceBaseline(unittest.TestCase):
    def setUp(self):
        self.doc_path = REPO_ROOT / "docs" / "architecture" / "v040-fail-closed-governance-baseline.md"
        self.gov_source = REPO_ROOT / "modules" / "workflow-action" / "fail_closed_governance.go"
        self.gov_test = REPO_ROOT / "modules" / "workflow-action" / "fail_closed_governance_test.go"
        self.rules_source = REPO_ROOT / "modules" / "workflow-action" / "business_rules.go"

    def test_governance_baseline_document_exists_and_valid(self):
        self.assertTrue(self.doc_path.is_file(), f"Document missing: {self.doc_path}")
        content = self.doc_path.read_text(encoding="utf-8")

        required_markers = [
            "document_id: ARC-V040-FCGOV-001",
            "HDEC-V040-FOUNDATION-054",
            "HDEC-V040-SCORING-058",
            "H040-004",
            "H040-007",
            "H040-011",
            "issue: 138",
            "packet: V040-I027",
            "Strict Fail-Closed Priority Hierarchy",
            "Fail-Closed Unknown Quarantine",
            "Deferred Manual Override Boundary",
            "Autonomous AI Boundaries",
            "Append-Only Audit Protection",
            "SYN-FC-01",
            "SYN-FC-04",
            "SYN-FC-05",
        ]
        for marker in required_markers:
            self.assertIn(marker, content, f"Missing marker {marker} in {self.doc_path}")

    def test_business_rules_has_required_denial_codes(self):
        self.assertTrue(self.rules_source.is_file(), f"Source missing: {self.rules_source}")
        content = self.rules_source.read_text(encoding="utf-8")

        required_symbols = [
            "DenialCriticalFailActive",
            "DenialUnknownQuarantined",
            "DenialManualOverrideDeferred",
            "DenialAutonomousAIBoundary",
            "ErrCriticalFailActive",
            "ErrUnknownQuarantined",
            "ErrManualOverrideDeferred",
            "ErrAutonomousAIBoundary",
            "RuleResultQuarantined",
            "TargetKindInspection",
        ]
        for sym in required_symbols:
            self.assertIn(sym, content, f"Missing symbol {sym} in {self.rules_source}")

    def test_governor_source_has_required_invariants(self):
        self.assertTrue(self.gov_source.is_file(), f"Source missing: {self.gov_source}")
        content = self.gov_source.read_text(encoding="utf-8")

        required_gov_symbols = [
            "type FailClosedGovernor struct",
            "type FailClosedAuditEntry struct",
            "type FailClosedTransitionRequest struct",
            "type FailClosedGovernanceResult struct",
            "PriorityOverrideDenial",
            "PriorityAIBoundaryDenial",
            "PriorityCriticalFail",
            "PriorityUnknownQuarantine",
            "PriorityScoreThreshold",
            "RegisterCriticalFinding",
            "RegisterQuarantinedQuestion",
            "ResolveUnknownQuestion",
            "ClearCriticalFinding",
            "Qualify(req FailClosedTransitionRequest)",
            "AuditHistory",
            "TotalAuditCount",
        ]
        for sym in required_gov_symbols:
            self.assertIn(sym, content, f"Missing governor symbol {sym} in {self.gov_source}")

    def test_governor_tests_exist_and_cover_scenarios(self):
        self.assertTrue(self.gov_test.is_file(), f"Test missing: {self.gov_test}")
        content = self.gov_test.read_text(encoding="utf-8")

        required_tests = [
            "TestFailClosed_CriticalFailPriorityBlocksConclusiveTransitions",
            "TestFailClosed_UnknownResponseQuarantinesTransition",
            "TestFailClosed_CriticalFailDominatesUnknownQuarantine",
            "TestFailClosed_ManualOverrideAttemptDeniedUnderDeferredH040",
            "TestFailClosed_AutonomousAIBoundaryEnforced",
            "TestFailClosed_AuthorizedHumanResolutionPath",
            "TestFailClosed_AppendOnlyAuditLedgerSequenceAndIntegrity",
        ]
        for t in required_tests:
            self.assertIn(t, content, f"Missing test function {t} in {self.gov_test}")


if __name__ == "__main__":
    unittest.main()
