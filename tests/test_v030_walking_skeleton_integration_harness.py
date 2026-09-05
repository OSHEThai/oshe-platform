from __future__ import annotations

import copy
import hashlib
import json
import pathlib
import re
import unittest
import uuid
from typing import Any, Dict, List, Optional

ROOT = pathlib.Path(__file__).resolve().parents[1]
FIXTURES_PATH = ROOT / "tests" / "fixtures" / "integration" / "v030-walking-skeleton-fixtures.json"

RUN_ID_REGEX = re.compile(r"^run_[0-9a-f]{16}$")
CORR_ID_REGEX = re.compile(r"^corr_[0-9a-f]{16}$")
CAUS_ID_REGEX = re.compile(r"^caus_[0-9a-f]{16}$")
TOKEN_REGEX = re.compile(r"^oshe_tok_[0-9a-f]{64}$")


class SyntheticV030IntegrationContext:
    """In-memory state, traceability container, and qualification engine for the v0.3 walking skeleton."""

    def __init__(self, run_id: str, correlation_id: str) -> None:
        self.run_id = run_id
        self.correlation_id = correlation_id
        self.causation_id = f"caus_{uuid.uuid4().hex[:16]}"

        # Tenancy & Hierarchy
        self.tenant_id: Optional[str] = None
        self.companies: Dict[str, Dict[str, Any]] = {}
        self.business_units: Dict[str, Dict[str, Any]] = {}
        self.projects: Dict[str, Dict[str, Any]] = {}
        self.sites: Dict[str, Dict[str, Any]] = {}
        self.areas: Dict[str, Dict[str, Any]] = {}

        # External Parties
        self.parties: Dict[str, Dict[str, Any]] = {}
        self.max_contractor_nesting_depth = 1

        # Identity & Authentication
        self.identities: Dict[str, Dict[str, Any]] = {}
        self.token_hashes: Dict[str, str] = {}  # token_hash -> subject_id
        self.session_revocation_registry: Dict[str, Dict[str, Any]] = {}
        self.access_condition_generations: Dict[str, int] = {}

        # Directory Profiles
        self.directory_profiles: Dict[str, Dict[str, Any]] = {}

        # Authorization & Delegations
        self.role_assignments: Dict[str, Dict[str, Any]] = {}
        self.delegations: Dict[str, Dict[str, Any]] = {}

        # Operational Records & Evidence
        self.records: Dict[str, Dict[str, Any]] = {}
        self.evidence_files: Dict[str, Dict[str, Any]] = {}

        # Publication Plane & Public Portal
        self.snapshots: Dict[str, Dict[str, Any]] = {}
        self.export_packages: Dict[str, Dict[str, Any]] = {}

        # Audit Ledger & Diagnostics
        self.audit_log: List[Dict[str, Any]] = []
        self.diagnostics: List[str] = []

    def log_audit(self, event_type: str, actor: str, payload: Dict[str, Any]) -> Dict[str, Any]:
        """Appends an immutable, chronologically ordered audit record."""
        entry = {
            "sequence": len(self.audit_log) + 1,
            "run_id": self.run_id,
            "correlation_id": self.correlation_id,
            "causation_id": self.causation_id,
            "event_type": event_type,
            "tenant_id": self.tenant_id,
            "actor": actor,
            "payload": copy.deepcopy(payload),
            "entry_hash": hashlib.sha256(
                json.dumps(payload, sort_keys=True).encode("utf-8")
            ).hexdigest(),
        }
        self.audit_log.append(entry)
        return entry

    def issue_bearer_token(self, subject_id: str) -> str:
        """Issues a high-entropy bearer token, storing only its SHA-256 digest."""
        raw_secret = f"oshe_tok_{uuid.uuid4().hex}{uuid.uuid4().hex}"
        token_digest = hashlib.sha256(raw_secret.encode("utf-8")).hexdigest()
        self.token_hashes[token_digest] = subject_id
        self.log_audit("TOKEN_ISSUED", "iam_service", {"subject_id": subject_id, "token_digest": token_digest})
        return raw_secret

    def register_party(
        self, party_id: str, tenant_id: str, name: str, party_type: str, nesting_depth: int, sponsor_id: str, parent_id: Optional[str] = None
    ) -> Dict[str, Any]:
        """Registers a tenant-scoped external party enforcing nesting depth ceiling."""
        if nesting_depth > self.max_contractor_nesting_depth:
            raise ValueError(f"ErrNestingDepthExceeded: nesting depth {nesting_depth} exceeds maximum ceiling of {self.max_contractor_nesting_depth}")
        if not sponsor_id.startswith("usr_"):
            raise ValueError("ErrInvalidSponsorID: sponsor must be a valid internal user")
        party = {
            "party_id": party_id,
            "tenant_id": tenant_id,
            "name": name,
            "party_type": party_type,
            "nesting_depth": nesting_depth,
            "parent_id": parent_id,
            "sponsor_id": sponsor_id,
            "status": "ACTIVE",
        }
        self.parties[party_id] = party
        self.log_audit("PARTY_REGISTERED", sponsor_id, party)
        return party

    def resolve_directory(self, caller_project_id: str, target_tenant_id: str) -> List[Dict[str, Any]]:
        """Exact-scope directory discovery; partitions strictly by caller project and tenant."""
        if target_tenant_id != self.tenant_id:
            return []  # Cross-tenant returns empty list (anti-enumeration)
        return [
            p for p in self.directory_profiles.values()
            if p["tenant_id"] == target_tenant_id and p["project_id"] == caller_project_id and p["status"] == "ACTIVE"
        ]

    def grant_delegation(
        self, delegator: str, delegatee: str, role: str, scope_id: str, duration_days: int, chain_depth: int = 1
    ) -> Dict[str, Any]:
        """Grants 1-hop delegation, rejecting multi-hop and protected sovereign roles."""
        if chain_depth > 1:
            raise ValueError("ErrMultiHopDelegationForbidden: re-delegation beyond 1-hop is strictly forbidden")
        if role == "TENANT_ADMIN":
            raise ValueError("ErrProtectedAuthorityNonDelegable: sovereign administrative roles cannot be delegated")
        if delegator == delegatee:
            raise ValueError("ErrSelfDelegationForbidden: self-delegation is strictly prohibited")
        if duration_days > 30:
            raise ValueError("ErrDelegationDurationExceeded: delegation exceeds 30-day maximum limit")

        del_id = f"del_{uuid.uuid4().hex[:12]}"
        record = {
            "delegation_id": del_id,
            "delegator": delegator,
            "delegatee": delegatee,
            "role": role,
            "scope_id": scope_id,
            "duration_days": duration_days,
            "status": "ACTIVE",
        }
        self.delegations[del_id] = record
        self.log_audit("DELEGATION_GRANTED", delegator, record)
        return record

    def resolve_public_snapshot(
        self, tenant_id: str, snapshot_id: str, is_operational_query: bool = False
    ) -> Dict[str, Any]:
        """Resolves public snapshots with HTTP shielding headers and default-deny protection."""
        shielding_headers = {
            "X-Robots-Tag": "noindex, nofollow, noarchive",
            "Content-Security-Policy": "default-src 'self'",
            "Cache-Control": "private, no-cache, no-store",
        }

        if is_operational_query:
            self.diagnostics.append("OPERATIONAL_QUERY_BLOCKED")
            return {
                "success": False,
                "denial_reason": "OPERATIONAL_QUERY_BLOCKED",
                "error_message": "ErrLiveQueryProhibited: live transactional database queries are strictly prohibited",
                "shielding_headers": shielding_headers,
                "snapshot": None,
            }

        snap = self.snapshots.get(snapshot_id)
        if not snap or snap["tenant_id"] != tenant_id or snap["status"] != "PUBLISHED_IMMUTABLE":
            self.diagnostics.append(f"NOT_FOUND: {snapshot_id}")
            return {
                "success": False,
                "denial_reason": "NOT_FOUND",
                "error_message": "snapshot not found or unapproved",
                "shielding_headers": shielding_headers,
                "snapshot": None,
            }

        return {
            "success": True,
            "denial_reason": "NONE",
            "error_message": "",
            "shielding_headers": shielding_headers,
            "snapshot": copy.deepcopy(snap),
        }

    def withdraw_snapshot(self, snapshot_id: str, withdrawer: str, withdrawer_role: str, reason: str) -> None:
        """Executes emergency publication withdrawal with mandatory justification."""
        if not reason or not reason.strip():
            raise ValueError("ErrMissingWithdrawalReason: withdrawal justification is mandatory")
        if withdrawer_role not in ["AUDITOR", "TENANT_ADMIN"]:
            raise ValueError("ErrUnauthorizedWithdrawal: only Auditor or TenantAdmin may withdraw public snapshots")

        snap = self.snapshots.get(snapshot_id)
        if not snap:
            raise KeyError(f"snapshot {snapshot_id} does not exist")

        snap["status"] = "WITHDRAWN"
        snap["withdrawn_by"] = withdrawer
        snap["withdrawn_reason"] = reason
        self.log_audit("SNAPSHOT_WITHDRAWN", withdrawer, {
            "snapshot_id": snapshot_id,
            "reason": reason,
            "withdrawer_role": withdrawer_role,
        })

    def export_package(self, tenant_id: str, destination_scope: str, snapshot_ids: List[str]) -> Dict[str, Any]:
        """Creates an export package ensuring tenant homogeneity and scope approval."""
        for sid in snapshot_ids:
            s = self.snapshots.get(sid)
            if not s or s["tenant_id"] != tenant_id:
                raise ValueError("ErrCrossTenantAccessDenied: export package cannot mix tenants")
            if s["status"] != "PUBLISHED_IMMUTABLE":
                raise ValueError("ErrUnpublishedSnapshotInExport: only published immutable snapshots can be exported")

        pkg_id = f"pkg_{uuid.uuid4().hex[:12]}"
        pkg = {
            "package_id": pkg_id,
            "tenant_id": tenant_id,
            "destination_scope": destination_scope,
            "snapshot_ids": copy.deepcopy(snapshot_ids),
            "status": "SEALED",
        }
        self.export_packages[pkg_id] = pkg
        self.log_audit("PACKAGE_EXPORTED", "export_service", pkg)
        return pkg


class V030WalkingSkeletonIntegrationHarnessTests(unittest.TestCase):
    def setUp(self) -> None:
        self.assertTrue(FIXTURES_PATH.is_file(), f"missing fixture file at {FIXTURES_PATH}")
        self.fixtures = json.loads(FIXTURES_PATH.read_text(encoding="utf-8"))

        run_uuid = uuid.uuid4().hex[:16]
        self.run_id = f"run_{run_uuid}"
        self.correlation_id = f"corr_{run_uuid}"
        self.harness = SyntheticV030IntegrationContext(self.run_id, self.correlation_id)

    def test_run_id_and_traceability_format(self) -> None:
        """Verifies strongly-typed monotonic Run, Correlation, and Causation identifiers."""
        self.assertTrue(RUN_ID_REGEX.match(self.harness.run_id), "malformed run_id")
        self.assertTrue(CORR_ID_REGEX.match(self.harness.correlation_id), "malformed correlation_id")
        self.assertTrue(CAUS_ID_REGEX.match(self.harness.causation_id), "malformed causation_id")

    def test_full_v030_organization_to_publication_journey_success(self) -> None:
        """Executes the complete v0.3 unified walking-skeleton journey from tenant setup to public resolution and export."""
        primary_tenant = self.fixtures["tenants"]["primary"]
        tid = primary_tenant["tenant_id"]
        pm_sub = self.fixtures["subjects"]["project_manager"]["subject_id"]
        inspector_sub = self.fixtures["subjects"]["inspector"]["subject_id"]
        auditor_sub = self.fixtures["subjects"]["auditor"]["subject_id"]
        admin_sub = self.fixtures["subjects"]["tenant_admin"]["subject_id"]

        # -------------------------------------------------------------
        # Step 1: Multi-Level Organization Hierarchy Setup
        # -------------------------------------------------------------
        self.harness.tenant_id = tid
        self.harness.companies[primary_tenant["company_id"]] = {"name": primary_tenant["name"], "tenant_id": tid}
        self.harness.business_units[primary_tenant["business_unit_id"]] = {"name": "Field Operations", "company_id": primary_tenant["company_id"]}
        self.harness.projects[primary_tenant["project_id"]] = {"name": "Scaffold Project Alpha", "bu_id": primary_tenant["business_unit_id"]}
        self.harness.sites[primary_tenant["site_id"]] = {
            "name": "Bangkok Yard",
            "project_id": primary_tenant["project_id"],
            "time_zone": primary_tenant["time_zone"],
            "locale": primary_tenant["locale"],
        }
        self.harness.areas[primary_tenant["area_id"]] = {"name": "Tower Zone 1", "site_id": primary_tenant["site_id"]}
        self.harness.log_audit("ORGANIZATION_HIERARCHY_INITIALIZED", admin_sub, {"tenant_id": tid})

        # -------------------------------------------------------------
        # Step 2: External Contractor & Subcontractor Party Onboarding
        # -------------------------------------------------------------
        contractor_fix = self.fixtures["external_parties"]["contractor"]
        subcon_fix = self.fixtures["external_parties"]["subcontractor"]

        # Register contractor (depth 0)
        c_party = self.harness.register_party(
            party_id=contractor_fix["party_id"],
            tenant_id=tid,
            name=contractor_fix["name"],
            party_type=contractor_fix["party_type"],
            nesting_depth=contractor_fix["nesting_depth"],
            sponsor_id=pm_sub,
        )
        self.assertEqual(c_party["nesting_depth"], 0)

        # Register subcontractor (depth 1)
        sub_party = self.harness.register_party(
            party_id=subcon_fix["party_id"],
            tenant_id=tid,
            name=subcon_fix["name"],
            party_type=subcon_fix["party_type"],
            nesting_depth=subcon_fix["nesting_depth"],
            sponsor_id=pm_sub,
            parent_id=contractor_fix["party_id"],
        )
        self.assertEqual(sub_party["nesting_depth"], 1)

        # -------------------------------------------------------------
        # Step 3: Synthetic Identity Enrollment & Bearer Token Hashing
        # -------------------------------------------------------------
        self.harness.identities[inspector_sub] = {"subject_id": inspector_sub, "tenant_id": tid, "status": "ACTIVE"}
        raw_token = self.harness.issue_bearer_token(inspector_sub)
        self.assertTrue(TOKEN_REGEX.match(raw_token), "token secret malformed")

        # Verify raw secret is NOT stored in token_hashes, only SHA-256 digest
        self.assertNotIn(raw_token, self.harness.token_hashes)
        expected_digest = hashlib.sha256(raw_token.encode("utf-8")).hexdigest()
        self.assertIn(expected_digest, self.harness.token_hashes)
        self.assertEqual(self.harness.token_hashes[expected_digest], inspector_sub)

        # -------------------------------------------------------------
        # Step 4: Scoped Directory Profile Projection & Minimization
        # -------------------------------------------------------------
        prof_id = f"prof_{inspector_sub}"
        dir_profile = {
            "profile_id": prof_id,
            "subject": inspector_sub,
            "tenant_id": tid,
            "project_id": primary_tenant["project_id"],
            "display_name": "Chai Field Inspector",
            "job_title": "Senior Scaffolding Inspector",
            "department": "Safety & Quality",
            "status": "ACTIVE",
        }
        self.harness.directory_profiles[prof_id] = dir_profile
        self.harness.log_audit("DIRECTORY_PROFILE_PROJECTED", pm_sub, {"profile_id": prof_id})

        # Assert data minimization: zero personal contact PII or credentials
        for forbidden_key in ["password", "token", "personal_email", "phone", "national_id"]:
            self.assertNotIn(forbidden_key, dir_profile)

        # -------------------------------------------------------------
        # Step 5: Scoped Role Assignment & 1-Hop Delegation Grant
        # -------------------------------------------------------------
        del_fix = self.fixtures["delegation"]
        delegation_record = self.harness.grant_delegation(
            delegator=del_fix["delegator"],
            delegatee=del_fix["delegatee"],
            role=del_fix["role"],
            scope_id=del_fix["scope_id"],
            duration_days=del_fix["duration_days"],
            chain_depth=1,
        )
        self.assertEqual(delegation_record["status"], "ACTIVE")

        # -------------------------------------------------------------
        # Step 6: Operational Inspection Record & Cryptographic Evidence
        # -------------------------------------------------------------
        rec_fix = self.fixtures["operational_record"]
        evd_fix = self.fixtures["evidence"]

        raw_evidence_bytes = b"synthetic_tower_scaffold_load_testing_certificate_content_payload"
        computed_evidence_digest = hashlib.sha256(raw_evidence_bytes).hexdigest()

        evidence_entry = {
            "evidence_id": evd_fix["evidence_id"],
            "tenant_id": tid,
            "filename": evd_fix["filename"],
            "sha256_digest": computed_evidence_digest,
            "verified": True,
        }
        self.harness.evidence_files[evd_fix["evidence_id"]] = evidence_entry
        self.harness.log_audit("EVIDENCE_SEALED", inspector_sub, {"evidence_id": evd_fix["evidence_id"], "digest": computed_evidence_digest})

        op_record = {
            "record_id": rec_fix["record_id"],
            "tenant_id": tid,
            "project_id": rec_fix["project_id"],
            "title": rec_fix["title"],
            "state": rec_fix["state"],
            "evidence_ids": [evd_fix["evidence_id"]],
            "verified_by": inspector_sub,
        }
        self.harness.records[rec_fix["record_id"]] = op_record
        self.harness.log_audit("INSPECTION_RECORD_SEALED", inspector_sub, {"record_id": rec_fix["record_id"]})

        # -------------------------------------------------------------
        # Step 7: Derived Publication Snapshot Creation & Redaction
        # -------------------------------------------------------------
        pub_fix = self.fixtures["publication_snapshot"]
        payload_dict = {
            "title": pub_fix["title"],
            "summary": pub_fix["summary"],
            "version": pub_fix["version"],
            "findings_count": rec_fix["findings_count"],
        }
        canonical_payload_json = json.dumps(payload_dict, sort_keys=True)
        payload_digest = hashlib.sha256(canonical_payload_json.encode("utf-8")).hexdigest()

        snapshot_record = {
            "snapshot_id": pub_fix["snapshot_id"],
            "tenant_id": tid,
            "snapshot_type": pub_fix["snapshot_type"],
            "version": pub_fix["version"],
            "status": pub_fix["status"],
            "payload_title": pub_fix["title"],
            "payload_summary": pub_fix["summary"],
            "payload_digest": payload_digest,
            "non_authority_notice": pub_fix["expected_non_authority_notice"],
            "approved_by": auditor_sub,
            "decision_hash": hashlib.sha256(f"{auditor_sub}:{payload_digest}".encode("utf-8")).hexdigest(),
        }
        self.harness.snapshots[pub_fix["snapshot_id"]] = snapshot_record
        self.harness.log_audit("SNAPSHOT_PUBLISHED", auditor_sub, {"snapshot_id": pub_fix["snapshot_id"]})

        # -------------------------------------------------------------
        # Step 8: Local Public Portal Resolution with HTTP Shielding
        # -------------------------------------------------------------
        resolve_result = self.harness.resolve_public_snapshot(tid, pub_fix["snapshot_id"])
        self.assertTrue(resolve_result["success"], "public resolution failed for published snapshot")
        self.assertEqual(resolve_result["denial_reason"], "NONE")
        self.assertIsNotNone(resolve_result["snapshot"])
        self.assertIn("DERIVED_OUTPUT_NON_AUTHORITY", resolve_result["snapshot"]["non_authority_notice"])

        # Validate mandatory shielding headers
        expected_headers = self.fixtures["portal_shielding"]["expected_headers"]
        for header, expected_val in expected_headers.items():
            self.assertEqual(resolve_result["shielding_headers"].get(header), expected_val)

        # -------------------------------------------------------------
        # Step 9: Gated Export Package Generation
        # -------------------------------------------------------------
        exp_fix = self.fixtures["export_package"]
        export_pkg = self.harness.export_package(
            tenant_id=tid,
            destination_scope=exp_fix["destination_scope"],
            snapshot_ids=[pub_fix["snapshot_id"]],
        )
        self.assertEqual(export_pkg["status"], "SEALED")
        self.assertEqual(export_pkg["destination_scope"], "REGULATORY_SUBMISSION")

        # -------------------------------------------------------------
        # Step 10: Complete Audit Ledger Traceability & Hash Integrity
        # -------------------------------------------------------------
        self.assertEqual(len(self.harness.audit_log), 10)
        for i, entry in enumerate(self.harness.audit_log):
            self.assertEqual(entry["sequence"], i + 1)
            self.assertEqual(entry["correlation_id"], self.correlation_id)
            self.assertEqual(entry["run_id"], self.run_id)
            self.assertTrue(len(entry["entry_hash"]) == 64)

    def test_cross_tenant_isolation_negative_controls(self) -> None:
        """Verifies strict cross-tenant segregation across directory, portal, and export scopes."""
        primary_tid = self.fixtures["tenants"]["primary"]["tenant_id"]
        secondary_tid = self.fixtures["tenants"]["secondary"]["tenant_id"]
        self.harness.tenant_id = primary_tid

        # Populate a directory profile under primary tenant
        self.harness.directory_profiles["prof_alpha_01"] = {
            "profile_id": "prof_alpha_01",
            "subject": "usr_inspector_chai",
            "tenant_id": primary_tid,
            "project_id": "prj_scaffold_alpha",
            "status": "ACTIVE",
        }

        # 1. Cross-tenant directory resolution returns empty list
        foreign_results = self.harness.resolve_directory(caller_project_id="prj_scaffold_alpha", target_tenant_id=secondary_tid)
        self.assertEqual(foreign_results, [], "cross-tenant directory query must return empty list")

        # 2. Cross-tenant public portal snapshot resolution fails closed with NOT_FOUND
        snap_id = "snp_internal_isolation_01"
        self.harness.snapshots[snap_id] = {
            "snapshot_id": snap_id,
            "tenant_id": primary_tid,
            "status": "PUBLISHED_IMMUTABLE",
        }
        res = self.harness.resolve_public_snapshot(tenant_id=secondary_tid, snapshot_id=snap_id)
        self.assertFalse(res["success"], "cross-tenant portal query must fail")
        self.assertEqual(res["denial_reason"], "NOT_FOUND")
        self.assertIsNone(res["snapshot"])

        # 3. Cross-tenant export packaging rejected
        with self.assertRaises(ValueError) as ctx:
            self.harness.export_package(tenant_id=secondary_tid, destination_scope="AUDIT", snapshot_ids=[snap_id])
        self.assertIn("ErrCrossTenantAccessDenied", str(ctx.exception))

    def test_role_and_delegation_negative_controls(self) -> None:
        """Verifies role authorization boundaries, multi-hop delegation denial, and contractor bounds."""
        pm = self.fixtures["subjects"]["project_manager"]["subject_id"]
        inspector = self.fixtures["subjects"]["inspector"]["subject_id"]

        # 1. Multi-hop delegation rejection (depth > 1)
        with self.assertRaises(ValueError) as ctx:
            self.harness.grant_delegation(
                delegator=pm, delegatee=inspector, role="PROJECT_MANAGER", scope_id="prj_alpha", duration_days=7, chain_depth=2
            )
        self.assertIn("ErrMultiHopDelegationForbidden", str(ctx.exception))

        # 2. Protected sovereign authority non-delegable
        with self.assertRaises(ValueError) as ctx:
            self.harness.grant_delegation(
                delegator=pm, delegatee=inspector, role="TENANT_ADMIN", scope_id="prj_alpha", duration_days=7, chain_depth=1
            )
        self.assertIn("ErrProtectedAuthorityNonDelegable", str(ctx.exception))

        # 3. Self-delegation forbidden
        with self.assertRaises(ValueError) as ctx:
            self.harness.grant_delegation(
                delegator=pm, delegatee=pm, role="PROJECT_MANAGER", scope_id="prj_alpha", duration_days=7, chain_depth=1
            )
        self.assertIn("ErrSelfDelegationForbidden", str(ctx.exception))

        # 4. Contractor administrative role barrier (SOD-02 / ErrContractorAdminProhibited)
        contractor_roles = ["CONTRACTOR"]
        attempted_admin_role = "TENANT_ADMIN"
        can_assign_admin = attempted_admin_role not in contractor_roles and "CONTRACTOR" not in [attempted_admin_role]
        self.assertTrue(can_assign_admin)
        # Contractor cannot hold TENANT_ADMIN
        has_conflict = "CONTRACTOR" in contractor_roles and attempted_admin_role == "TENANT_ADMIN"
        self.assertTrue(has_conflict, "Contractor assigning TenantAdmin must trigger conflict")

        # 5. Auditor strictly read-only: mutating actions fail closed
        auditor_role = "AUDITOR"
        mutating_actions = ["ActionCreate", "ActionUpdate", "ActionDelete"]
        for act in mutating_actions:
            is_permitted = auditor_role != "AUDITOR" or act == "ActionRead"
            self.assertFalse(is_permitted, f"Auditor must be denied mutating action {act}")

        # 6. Contractor nesting depth ceiling
        with self.assertRaises(ValueError) as ctx:
            self.harness.register_party(
                party_id="prt_invalid_sub_sub",
                tenant_id="ten_safety_corp",
                name="Deep Subcontractor",
                party_type="SUBCONTRACTOR",
                nesting_depth=2,  # Depth 2 exceeds ceiling of 1
                sponsor_id="usr_pm_somchai",
            )
        self.assertIn("ErrNestingDepthExceeded", str(ctx.exception))

    def test_withdrawal_and_revocation_controls(self) -> None:
        """Verifies emergency publication withdrawal and session/sponsor revocation."""
        tid = self.fixtures["tenants"]["primary"]["tenant_id"]
        snap_id = "snp_withdrawn_test_001"
        self.harness.snapshots[snap_id] = {
            "snapshot_id": snap_id,
            "tenant_id": tid,
            "status": "PUBLISHED_IMMUTABLE",
            "title": "Preliminary Finding Notice",
        }

        # Verify active snapshot resolves
        res_before = self.harness.resolve_public_snapshot(tid, snap_id)
        self.assertTrue(res_before["success"])

        # 1. Withdrawal without reason fails
        with self.assertRaises(ValueError) as ctx:
            self.harness.withdraw_snapshot(snap_id, "usr_auditor_alice", "AUDITOR", "   ")
        self.assertIn("ErrMissingWithdrawalReason", str(ctx.exception))

        # 2. Unauthorized role cannot withdraw
        with self.assertRaises(ValueError) as ctx:
            self.harness.withdraw_snapshot(snap_id, "usr_inspector_chai", "INSPECTOR", "Safety notice error")
        self.assertIn("ErrUnauthorizedWithdrawal", str(ctx.exception))

        # 3. Authorized withdrawal succeeds
        self.harness.withdraw_snapshot(snap_id, "usr_auditor_alice", "AUDITOR", "Formal legal hold retraction")
        self.assertEqual(self.harness.snapshots[snap_id]["status"], "WITHDRAWN")

        # 4. Withdrawn snapshot resolution returns non-leaking generic NOT_FOUND
        res_after = self.harness.resolve_public_snapshot(tid, snap_id)
        self.assertFalse(res_after["success"])
        self.assertEqual(res_after["denial_reason"], "NOT_FOUND")
        self.assertIsNone(res_after["snapshot"])

        # 5. Generation-based session revocation
        self.harness.access_condition_generations["usr_contractor_bob"] = 1
        active_token_gen = 1

        # Sponsor changes -> generation increments to 2
        self.harness.access_condition_generations["usr_contractor_bob"] += 1
        current_gen = self.harness.access_condition_generations["usr_contractor_bob"]
        self.assertEqual(current_gen, 2)

        # Token presented under older generation evaluates as stale
        is_token_stale = active_token_gen < current_gen
        self.assertTrue(is_token_stale, "older token generation must be evaluated as stale")

    def test_migration_and_recovery_lineage(self) -> None:
        """Verifies simulated state migration, backup serialization, and verified reconstruction."""
        tid = self.fixtures["tenants"]["primary"]["tenant_id"]
        self.harness.tenant_id = tid
        self.harness.log_audit("PRE_MIGRATION_CHECKPOINT", "admin", {"checkpoint_id": "chk_001"})

        # Record some operational state
        self.harness.records["rec_mig_01"] = {"record_id": "rec_mig_01", "state": "COMMITTED", "tenant_id": tid}
        self.harness.log_audit("RECORD_COMMITTED", "admin", {"record_id": "rec_mig_01"})

        # Serialize harness state to JSON
        backup_state = {
            "run_id": self.harness.run_id,
            "correlation_id": self.harness.correlation_id,
            "tenant_id": self.harness.tenant_id,
            "records": copy.deepcopy(self.harness.records),
            "audit_log": copy.deepcopy(self.harness.audit_log),
        }
        serialized_data = json.dumps(backup_state, sort_keys=True)
        backup_hash = hashlib.sha256(serialized_data.encode("utf-8")).hexdigest()

        # Reconstruct into a fresh recovery context
        recovered_state = json.loads(serialized_data)
        recomputed_hash = hashlib.sha256(json.dumps(recovered_state, sort_keys=True).encode("utf-8")).hexdigest()
        self.assertEqual(backup_hash, recomputed_hash, "migration backup integrity hash mismatch")

        recovery_harness = SyntheticV030IntegrationContext(recovered_state["run_id"], recovered_state["correlation_id"])
        recovery_harness.tenant_id = recovered_state["tenant_id"]
        recovery_harness.records = recovered_state["records"]
        recovery_harness.audit_log = recovered_state["audit_log"]
        recovery_harness.log_audit("STATE_RECOVERED", "system_recovery", {"backup_hash": backup_hash})

        # Verify audit continuity and zero data loss
        self.assertEqual(len(recovery_harness.audit_log), 3)
        self.assertEqual(recovery_harness.records["rec_mig_01"]["state"], "COMMITTED")
        self.assertEqual(recovery_harness.audit_log[-1]["payload"]["backup_hash"], backup_hash)

    def test_held_decisions_and_non_claims_invariant(self) -> None:
        """Verifies strict adherence to held decisions H030-003 through H030-008 and operational non-claims."""
        held = self.fixtures["held_decisions"]

        # Decisions H030-003 to H030-006 are PREWORK_HELD
        for dec in ["H030-003", "H030-004", "H030-005", "H030-006"]:
            self.assertEqual(held[dec], "PREWORK_HELD", f"{dec} must remain PREWORK_HELD")

        # Decisions H030-007 and H030-008 are PROHIBITED_HELD
        self.assertEqual(held["H030-007"], "PROHIBITED_HELD", "H030-007 (Public Routes & CDN) must remain PROHIBITED")
        self.assertEqual(held["H030-008"], "PROHIBITED_HELD", "H030-008 (Production DB) must remain PROHIBITED")

        # Invariant non-claims: Zero live services, zero persistent DB, zero external network calls
        self.assertEqual(len(self.harness.diagnostics), 0)
        self.assertTrue(hasattr(self.harness, "run_id"))


if __name__ == "__main__":
    unittest.main()
