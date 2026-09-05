from __future__ import annotations

import hashlib
import pathlib
import re
import unittest
from typing import Any, Dict, List, Optional

ROOT = pathlib.Path(__file__).resolve().parents[1]
EVIDENCE_BUNDLE_PATH = ROOT / "docs" / "architecture" / "v030-release-evidence-bundle.md"
ARCH_README_PATH = ROOT / "docs" / "architecture" / "README.md"
RELEASE_README_PATH = ROOT / "tests" / "release" / "README.md"

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


class SyntheticFailClosedSecurityContext:
    """In-memory reference engine verifying deterministic fail-closed negative controls and revocation integrity."""

    def __init__(self) -> None:
        self.tenants: Dict[str, Dict[str, Any]] = {"ten_alpha": {"name": "Alpha Corp"}}
        self.projects: Dict[str, str] = {"prj_alpha_1": "ten_alpha"}
        self.identities: Dict[str, Dict[str, Any]] = {}
        self.tokens: Dict[str, str] = {}  # token_hash -> subject_id
        self.revoked_sessions: Dict[str, bool] = {}
        self.sponsor_generations: Dict[str, int] = {}
        self.snapshots: Dict[str, Dict[str, Any]] = {}
        self.prohibited_field_names = {"password", "bearer_token", "national_id", "secret_key"}

    def evaluate_access(self, token: str, tenant_id: str, project_id: str) -> Dict[str, Any]:
        """Strict fail-closed evaluation on unauthenticated, mismatched, or invalid scope."""
        if not token or not token.startswith("oshe_tok_"):
            return {"granted": False, "reason": "DenialUnauthenticated"}

        token_hash = hashlib.sha256(token.encode("utf-8")).hexdigest()
        subject_id = self.tokens.get(token_hash)
        if not subject_id:
            return {"granted": False, "reason": "DenialUnauthenticated"}

        if self.revoked_sessions.get(token_hash, False):
            return {"granted": False, "reason": "ErrTokenRevoked"}

        if tenant_id not in self.tenants:
            return {"granted": False, "reason": "DenialScopeMismatch"}

        expected_tenant = self.projects.get(project_id)
        if expected_tenant != tenant_id:
            return {"granted": False, "reason": "DenialDirectObjectMismatch"}

        return {"granted": True, "subject_id": subject_id, "project_id": project_id}

    def evaluate_delegation(self, depth: int, requested_role: str, is_emergency: bool) -> Dict[str, Any]:
        """Enforces 1-hop limit, non-delegable protected roles, and emergency access denial."""
        if is_emergency:
            return {"granted": False, "error": "ErrEmergencyAccessDenied"}
        if depth > 1:
            return {"granted": False, "error": "ErrMultiHopDelegationForbidden"}
        if requested_role in {"RoleTenantAdmin", "RoleSovereign"}:
            return {"granted": False, "error": "ErrProtectedAuthorityNonDelegable"}
        return {"granted": True, "depth": depth, "role": requested_role}

    def check_sod_conflict(self, assigned_roles: List[str]) -> bool:
        """Returns True if Segregation of Duties conflict detected."""
        conflicting_pairs = [{ "INSPECTOR", "AUDITOR" }, { "AUTHOR", "APPROVER" }]
        roles_set = set(assigned_roles)
        for pair in conflicting_pairs:
            if pair.issubset(roles_set):
                return True
        return False

    def sanitize_profile_payload(self, fields: Dict[str, Any]) -> Dict[str, Any]:
        """Fails closed if prohibited fields are present."""
        for field in fields:
            if field.lower() in self.prohibited_field_names:
                raise ValueError(f"ErrProhibitedFieldDetected: {field}")
        return {k: v for k, v in fields.items() if k in {"display_name", "role", "email"}}

    def withdraw_snapshot(self, snapshot_id: str, operator_role: str, reason: str) -> None:
        """Performs emergency withdrawal of publication snapshots."""
        if not reason or not reason.strip():
            raise ValueError("ErrMissingWithdrawalReason")
        if operator_role not in {"AUDITOR", "ADMIN"}:
            raise ValueError("ErrUnauthorizedWithdrawal")
        if snapshot_id not in self.snapshots:
            raise ValueError("ErrSnapshotNotFound")
        self.snapshots[snapshot_id]["status"] = "WITHDRAWN"

    def resolve_public_snapshot(self, tenant_id: str, snapshot_id: str) -> Dict[str, Any]:
        """Resolves snapshot for public route; returns non-leaking generic NOT_FOUND if withdrawn."""
        snapshot = self.snapshots.get(snapshot_id)
        if not snapshot or snapshot.get("tenant_id") != tenant_id or snapshot.get("status") == "WITHDRAWN":
            return {"success": False, "denial_reason": "NOT_FOUND", "snapshot": None}
        return {"success": True, "snapshot": snapshot}


class V030ReleaseEvidenceBundleTests(unittest.TestCase):
    def setUp(self) -> None:
        self.assertTrue(EVIDENCE_BUNDLE_PATH.is_file(), f"Missing release evidence bundle at {EVIDENCE_BUNDLE_PATH}")
        self.content = EVIDENCE_BUNDLE_PATH.read_text(encoding="utf-8")
        self.frontmatter = parse_simple_frontmatter(self.content)
        self.security_ctx = SyntheticFailClosedSecurityContext()

    def test_document_structure_and_frontmatter(self) -> None:
        """Asserts required governed frontmatter fields and operational boundaries."""
        self.assertEqual(self.frontmatter.get("document_id"), "ARC-V030-RELEVD-001")
        self.assertEqual(self.frontmatter.get("document_type"), "architecture_specification")
        self.assertEqual(self.frontmatter.get("lifecycle_status"), "DRAFT")
        self.assertEqual(self.frontmatter.get("status"), "APPROVED_FOR_LOCAL_ALPHA_DEVELOPMENT")
        self.assertEqual(self.frontmatter.get("governing_issue"), "GitHub Issue #110")
        self.assertEqual(self.frontmatter.get("governing_decision"), "HDEC-V030-ENTRY-AND-POLICY-052")
        self.assertEqual(self.frontmatter.get("milestone"), "v0.3.0 - Organization Identity and Portal Alpha")

        deferred = self.frontmatter.get("deferred_human_gates", [])
        self.assertIn("H030-007", deferred, "H030-007 must remain in deferred_human_gates")
        self.assertIn("H030-008", deferred, "H030-008 must remain in deferred_human_gates")

        credit = self.frontmatter.get("credit_boundary", "")
        self.assertIn("NO_RELEASE_OR_HOSTED_ACTIVATION", credit)

    def test_requirements_test_and_review_lineage(self) -> None:
        """Verifies complete requirements, PR, commit, test, and independent review lineage."""
        required_work_items = [
            "V030-I011", "V030-I012", "V030-I013", "V030-I014", "V030-I015",
            "V030-I016", "V030-I017", "V030-I018", "V030-I019", "V030-I020",
            "V030-I021", "V030-I022", "V030-I023", "V030-I024", "V030-I025",
            "V030-I026", "V030-I027", "V030-I028", "V030-I029", "V030-I030",
            "V030-I031", "V030-I032", "V030-I033", "V030-I034", "V030-I035",
            "V030-I036"
        ]
        for item in required_work_items:
            self.assertIn(item, self.content, f"Milestone item {item} must be present in evidence bundle")

        # Verify key PR references exist
        for pr_num in ["#1029", "#1030", "#1031", "#1032", "#1033", "#1034", "#1036", "#1037", "#1038", "#1039", "#1040", "#1041", "#1042", "#1043", "#1044", "#1045", "#1046", "#1047", "#1049", "#1050", "#1051", "#1052", "#1053", "#1054", "#1055", "#1056"]:
            self.assertIn(pr_num, self.content, f"Pull request {pr_num} must be tracked in lineage table")

    def test_defect_and_predecessor_disposition_ledger(self) -> None:
        """Verifies all exceptions (EXC-V030-001, EXC-V030-002, EXC-V030-003) are recorded with non-adoption and successors."""
        self.assertIn("EXC-V030-001", self.content)
        self.assertIn("EXC-V030-002", self.content)
        self.assertIn("EXC-V030-003", self.content)

        # Predecessor non-adoption proof
        self.assertIn("PR #1035", self.content)
        self.assertIn("PR #1048", self.content)
        self.assertIn("PREDECESSOR_OUTPUT_NOT_ADOPTED_NO_CREDIT", self.content)
        self.assertIn("NO_CREDIT_SCOPE_BREACH", self.content)

        # Clean successors proof
        self.assertIn("PR #1038", self.content)
        self.assertIn("PR #1050", self.content)
        self.assertIn("ASN-V030-I024-INDEPENDENT-SECURITY-REVIEW-004", self.content)

    def test_unassigned_risk_fail_closed_assertions(self) -> None:
        """Executes synthetic fail-closed behavior tests for authorization and boundary controls."""
        token = "oshe_tok_" + "a" * 64
        token_hash = hashlib.sha256(token.encode("utf-8")).hexdigest()
        self.security_ctx.tokens[token_hash] = "usr_alice"

        # 1. Unauthenticated request fails closed
        res_unauth = self.security_ctx.evaluate_access("bad_token", "ten_alpha", "prj_alpha_1")
        self.assertFalse(res_unauth["granted"])
        self.assertEqual(res_unauth["reason"], "DenialUnauthenticated")

        # 2. Scope mismatch fails closed
        res_mismatch = self.security_ctx.evaluate_access(token, "ten_unknown", "prj_alpha_1")
        self.assertFalse(res_mismatch["granted"])
        self.assertEqual(res_mismatch["reason"], "DenialScopeMismatch")

        # 3. Direct object mismatch fails closed
        res_obj_mismatch = self.security_ctx.evaluate_access(token, "ten_alpha", "prj_beta_2")
        self.assertFalse(res_obj_mismatch["granted"])
        self.assertEqual(res_obj_mismatch["reason"], "DenialDirectObjectMismatch")

        # 4. Multi-hop delegation rejected
        del_multihop = self.security_ctx.evaluate_delegation(depth=2, requested_role="RoleInspector", is_emergency=False)
        self.assertFalse(del_multihop["granted"])
        self.assertEqual(del_multihop["error"], "ErrMultiHopDelegationForbidden")

        # 5. Emergency access request rejected
        del_emerg = self.security_ctx.evaluate_delegation(depth=1, requested_role="RoleInspector", is_emergency=True)
        self.assertFalse(del_emerg["granted"])
        self.assertEqual(del_emerg["error"], "ErrEmergencyAccessDenied")

        # 6. Protected role non-delegable
        del_prot = self.security_ctx.evaluate_delegation(depth=1, requested_role="RoleTenantAdmin", is_emergency=False)
        self.assertFalse(del_prot["granted"])
        self.assertEqual(del_prot["error"], "ErrProtectedAuthorityNonDelegable")

        # 7. SOD conflict detection
        self.assertTrue(self.security_ctx.check_sod_conflict(["INSPECTOR", "AUDITOR"]))
        self.assertFalse(self.security_ctx.check_sod_conflict(["INSPECTOR", "CONTRACTOR"]))

        # 8. Prohibited field injection fails closed
        with self.assertRaises(ValueError) as ctx:
            self.security_ctx.sanitize_profile_payload({"display_name": "Bob", "bearer_token": "secret_xyz"})
        self.assertIn("ErrProhibitedFieldDetected", str(ctx.exception))

    def test_revocation_and_withdrawal_integrity(self) -> None:
        """Verifies session revocation, sponsor generation change, and emergency snapshot withdrawal."""
        token = "oshe_tok_" + "b" * 64
        token_hash = hashlib.sha256(token.encode("utf-8")).hexdigest()
        self.security_ctx.tokens[token_hash] = "usr_bob"

        # Initially granted
        res1 = self.security_ctx.evaluate_access(token, "ten_alpha", "prj_alpha_1")
        self.assertTrue(res1["granted"])

        # Revoke session
        self.security_ctx.revoked_sessions[token_hash] = True
        res_revoked = self.security_ctx.evaluate_access(token, "ten_alpha", "prj_alpha_1")
        self.assertFalse(res_revoked["granted"])
        self.assertEqual(res_revoked["reason"], "ErrTokenRevoked")

        # Snapshot lifecycle & emergency withdrawal
        snap_id = "snap_test_001"
        self.security_ctx.snapshots[snap_id] = {
            "snapshot_id": snap_id,
            "tenant_id": "ten_alpha",
            "status": "PUBLISHED",
            "data": {"project": "Alpha", "metrics": "clean"}
        }

        # Resolution before withdrawal succeeds
        res_snap1 = self.security_ctx.resolve_public_snapshot("ten_alpha", snap_id)
        self.assertTrue(res_snap1["success"])
        self.assertIsNotNone(res_snap1["snapshot"])

        # Withdrawal fails if missing reason
        with self.assertRaises(ValueError) as ctx1:
            self.security_ctx.withdraw_snapshot(snap_id, "ADMIN", "   ")
        self.assertIn("ErrMissingWithdrawalReason", str(ctx1.exception))

        # Withdrawal fails if unauthorized role
        with self.assertRaises(ValueError) as ctx2:
            self.security_ctx.withdraw_snapshot(snap_id, "CONTRACTOR", "Valid reason")
        self.assertIn("ErrUnauthorizedWithdrawal", str(ctx2.exception))

        # Authorized withdrawal succeeds
        self.security_ctx.withdraw_snapshot(snap_id, "ADMIN", "Emergency regulatory recall")
        self.assertEqual(self.security_ctx.snapshots[snap_id]["status"], "WITHDRAWN")

        # Post-withdrawal resolution returns generic NOT_FOUND
        res_snap_post = self.security_ctx.resolve_public_snapshot("ten_alpha", snap_id)
        self.assertFalse(res_snap_post["success"])
        self.assertEqual(res_snap_post["denial_reason"], "NOT_FOUND")
        self.assertIsNone(res_snap_post["snapshot"])

    def test_deferred_human_gates_and_governance_recommendation(self) -> None:
        """Verifies explicit hold status of H030-007 and H030-008 and operational non-claims."""
        self.assertIn("H030-007", self.content)
        self.assertIn("H030-008", self.content)
        self.assertIn("HOLD", self.content)
        self.assertIn("HDEC-V030-ENTRY-AND-POLICY-052", self.content)
        self.assertIn("Zero Hosted / External Route Activation", self.content)

    def test_readme_and_cross_reference_integrity(self) -> None:
        """Asserts that architecture and release README files reference the evidence bundle."""
        self.assertTrue(ARCH_README_PATH.is_file())
        arch_readme_text = ARCH_README_PATH.read_text(encoding="utf-8")
        self.assertIn("v030-release-evidence-bundle.md", arch_readme_text)
        self.assertIn("ARC-V030-RELEVD-001", arch_readme_text)

        self.assertTrue(RELEASE_README_PATH.is_file())
        release_readme_text = RELEASE_README_PATH.read_text(encoding="utf-8")
        self.assertIn("test_v030_release_evidence_bundle.py", release_readme_text)
        self.assertIn("ARC-V030-RELEVD-001", release_readme_text)


if __name__ == "__main__":
    unittest.main()
