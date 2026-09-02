from __future__ import annotations

import copy
import hashlib
import json
import shutil
import subprocess
import sys
import tempfile
import unittest
from datetime import datetime, timedelta, timezone
from pathlib import Path

import yaml
from jsonschema import Draft202012Validator


REPO_ROOT = Path(__file__).resolve().parents[1]

I015_CONTRACT_FILES = {
    "mission": ("mission.schema.json", "mission.example.yaml"),
    "task": ("task.schema.json", "task-packet.example.yaml"),
    "result": ("result.schema.json", "result-contract.example.yaml"),
    "review": ("review.schema.json", "review.example.yaml"),
    "integration": ("integration.schema.json", "integration.example.yaml"),
    "handoff": ("handoff.schema.json", "handoff.example.yaml"),
}


class AgentOsValidatorTests(unittest.TestCase):
    def load_i015_contract(self, root: Path, contract_type: str) -> tuple[dict[str, object], dict[str, object]]:
        schema_name, example_name = I015_CONTRACT_FILES[contract_type]
        schema = json.loads((root / ".ai" / "schemas" / schema_name).read_text(encoding="utf-8"))
        instance = yaml.safe_load((root / ".ai" / "examples" / example_name).read_text(encoding="utf-8"))
        return schema, instance

    def i015_schema_errors(self, schema: dict[str, object], instance: dict[str, object]) -> list[str]:
        return [error.message for error in Draft202012Validator(schema).iter_errors(instance)]

    def make_fixture(self, destination: Path) -> None:
        shutil.copytree(REPO_ROOT / ".ai", destination / ".ai")
        for name in ("AGENTS.md", "CLAUDE.md", "GEMINI.md", "QWEN.md"):
            shutil.copy2(REPO_ROOT / name, destination / name)
        adr_destination = destination / "docs" / "adr"
        adr_destination.mkdir(parents=True)
        shutil.copy2(
            REPO_ROOT / "docs" / "adr" / "adr-0006-evidence-gated-full-github-operator-authority.md",
            adr_destination / "adr-0006-evidence-gated-full-github-operator-authority.md",
        )
        shutil.copy2(
            REPO_ROOT / "docs" / "adr" / "adr-0007-local-first-ci-and-repository-lifecycle.md",
            adr_destination / "adr-0007-local-first-ci-and-repository-lifecycle.md",
        )
        ci_destination = destination / ".ci"
        ci_destination.mkdir()
        shutil.copy2(REPO_ROOT / ".ci" / "local-ci.json", ci_destination / "local-ci.json")
        tools_destination = destination / "tools"
        tools_destination.mkdir()
        shutil.copy2(REPO_ROOT / "tools" / "run_local_ci.py", tools_destination / "run_local_ci.py")
        tests_destination = destination / "tests"
        tests_destination.mkdir()
        shutil.copy2(REPO_ROOT / "tests" / "test_local_ci.py", tests_destination / "test_local_ci.py")

    def run_validator(self, root: Path) -> subprocess.CompletedProcess[str]:
        return subprocess.run(
            [sys.executable, str(root / ".ai" / "tools" / "validate_agent_os.py")],
            cwd=root,
            check=False,
            capture_output=True,
            text=True,
        )

    def approve_test_route_and_credential(self, root: Path) -> tuple[str, str]:
        route_path = root / ".ai" / "provider-routes" / "ai-service-route-registry.yaml"
        route_registry = yaml.safe_load(route_path.read_text(encoding="utf-8"))
        route = route_registry["routes"][0]
        route_id = route["provider_route_id"]
        route_registry["approved_route_ids"] = [route_id]
        route_registry["enabled_route_ids"] = [route_id]
        route_registry["active_route_ids"] = [route_id]
        route["lifecycle"]["status"] = "APPROVED"
        route["lifecycle"]["dispatch_enabled"] = True
        route_path.write_text(yaml.safe_dump(route_registry, sort_keys=False), encoding="utf-8")

        credential_id = "github-test-credential"
        credential_path = root / ".ai" / "policies" / "github-credential-profiles.yaml"
        credential_registry = yaml.safe_load(credential_path.read_text(encoding="utf-8"))
        credential_registry["approved_profile_ids"] = [credential_id]
        credential_registry["profiles"] = [
            {
                "credential_profile_id": credential_id,
                "approval_status": "APPROVED_FOR_TEST_FIXTURE",
            }
        ]
        credential_path.write_text(yaml.safe_dump(credential_registry, sort_keys=False), encoding="utf-8")
        return route_id, credential_id

    def make_gate_record(
        self,
        root: Path,
        now: datetime,
        action_class: str = "METADATA",
        independent_review: bool = False,
    ) -> dict[str, object]:
        policy = yaml.safe_load((root / ".ai" / "policies" / "github-operations.yaml").read_text(encoding="utf-8"))
        route_id, credential_id = self.approve_test_route_and_credential(root)
        evidence = {
            name: {"satisfied": True, "refs": [f"EVD-{name}"]}
            for name in policy["required_evidence_flags"]
        }
        evidence[policy["execution_route_evidence_flags"]["AI_PROVIDER"]] = {
            "satisfied": True,
            "refs": ["EVD-provider-route"],
        }
        return {
            "schema_version": "1.0.0",
            "gate_id": "GHG-TEST-001",
            "assignment_id": "ASN-TEST-001",
            "session_id": "SESSION-TEST-001",
            "human_authority_ref": "ADR-0006",
            "actor": {
                "role_id": "release-evidence-agent",
                "specialist_profile_id": "github-manager",
                "provider_route_id": route_id,
                "credential_profile_id": credential_id,
            },
            "scope": {
                "organization": "OSHEThai",
                "repository": "oshe-platform",
                "action_class": action_class,
                "action": "test-operation",
                "target": "test-target",
                "expected_pre_state_digest": "sha256:" + "0" * 64,
                "expected_post_state": "Expected test state",
                "exact_commit_or_configuration_identity": "test-identity",
            },
            "evidence": evidence,
            "unresolved_blockers": [],
            "independent_review": {
                "reviewer_role_id": "independent-review-challenge-agent" if independent_review else None,
                "reviewer_assignment_id": "ASN-REVIEW-TEST-001" if independent_review else None,
                "disposition": "PASS" if independent_review else "NOT_REQUIRED",
                "review_ref": "REV-TEST-001" if independent_review else None,
            },
            "requested_at": (now - timedelta(minutes=1)).isoformat(),
            "expires_at": (now + timedelta(minutes=10)).isoformat(),
            "external_authority_ref": None,
        }

    def run_gate(
        self, root: Path, record: dict[str, object], now: datetime
    ) -> subprocess.CompletedProcess[str]:
        record_path = root / "gate.yaml"
        record_path.write_text(yaml.safe_dump(record, sort_keys=False), encoding="utf-8")
        return subprocess.run(
            [
                sys.executable,
                str(root / ".ai" / "tools" / "evaluate_github_operation.py"),
                str(record_path),
                "--now",
                now.isoformat(),
            ],
            cwd=root,
            check=False,
            capture_output=True,
            text=True,
        )

    def run_direct_gh_executor(
        self, root: Path, record: dict[str, object], now: datetime, dry_run: bool = True
    ) -> subprocess.CompletedProcess[str]:
        record_path = root / "direct-gate.yaml"
        record_path.write_text(yaml.safe_dump(record, sort_keys=False), encoding="utf-8")
        return subprocess.run(
            [
                sys.executable,
                str(root / ".ai" / "tools" / "execute_github_operation.py"),
                str(record_path),
                "--now",
                now.isoformat(),
                *( ["--dry-run"] if dry_run else [] ),
            ],
            cwd=root,
            check=False,
            capture_output=True,
            text=True,
        )

    def make_direct_gh_gate_record(self, root: Path, now: datetime) -> dict[str, object]:
        policy_path = root / ".ai" / "policies" / "github-operations.yaml"
        policy = yaml.safe_load(policy_path.read_text(encoding="utf-8"))
        direct_route = policy["direct_gh_execution_routes"][0]
        direct_route["activated_at"] = (now - timedelta(minutes=1)).isoformat()
        direct_route["expires_at"] = (now + timedelta(days=1)).isoformat()
        policy_path.write_text(yaml.safe_dump(policy, sort_keys=False), encoding="utf-8")
        credential_path = root / ".ai" / "policies" / "github-credential-profiles.yaml"
        credentials = yaml.safe_load(credential_path.read_text(encoding="utf-8"))
        profile = next(item for item in credentials["profiles"] if item["credential_profile_id"] == direct_route["credential_profile_id"])
        profile["expires_at"] = (now + timedelta(days=1)).isoformat()
        credential_path.write_text(yaml.safe_dump(credentials, sort_keys=False), encoding="utf-8")
        command = ["gh", "issue", "comment", "--repo", "OSHEThai/oshe-platform", "1", "--body", "test"]
        evidence = {name: {"satisfied": True, "refs": [f"EVD-{name}"]} for name in policy["required_evidence_flags"]}
        evidence[policy["execution_route_evidence_flags"]["DIRECT_GH_CLI"]] = {"satisfied": True, "refs": ["EVD-direct-gh-route"]}
        return {
            "schema_version": "1.0.0",
            "gate_id": "GHG-DIRECT-GH-001",
            "assignment_id": "ASN-DIRECT-GH-001",
            "session_id": "SESSION-DIRECT-GH-001",
            "human_authority_ref": "ADR-0006",
            "actor": {"role_id": "release-evidence-agent", "specialist_profile_id": "github-manager", "execution_route_kind": "DIRECT_GH_CLI", "execution_route_id": direct_route["execution_route_id"], "provider_route_id": None, "credential_profile_id": direct_route["credential_profile_id"]},
            "scope": {"organization": "OSHEThai", "repository": "oshe-platform", "action_class": "METADATA", "action": "issue-comment", "target": "test-target", "expected_pre_state_digest": "sha256:" + "0" * 64, "expected_post_state": "Expected test state", "exact_commit_or_configuration_identity": "test-identity"},
            "execution": {"command": command, "command_digest": "sha256:" + hashlib.sha256(json.dumps(command, separators=(",", ":"), ensure_ascii=False).encode("utf-8")).hexdigest()},
            "evidence": evidence,
            "unresolved_blockers": [],
            "independent_review": {"reviewer_role_id": None, "reviewer_assignment_id": None, "disposition": "NOT_REQUIRED", "review_ref": None},
            "requested_at": (now - timedelta(minutes=1)).isoformat(),
            "expires_at": (now + timedelta(minutes=10)).isoformat(),
            "external_authority_ref": None,
        }

    def test_current_package_passes(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            self.make_fixture(root)
            result = self.run_validator(root)

        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertIn("provider_routes_enabled=0", result.stdout)

    def test_static_provider_note_set_passes(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            self.make_fixture(root)
            result = self.run_validator(root)

        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertNotIn("Provider note", result.stdout)

    def test_provider_notes_reject_missing_required_statements(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            self.make_fixture(root)
            codex_path = root / ".ai" / "provider-notes" / "codex.md"
            codex_text = codex_path.read_text(encoding="utf-8")
            codex_path.write_text(
                codex_text.replace('unsupported_invocation: "FAIL_CLOSED_NO_DISPATCH"\n', "", 1),
                encoding="utf-8",
            )
            qwen_path = root / ".ai" / "provider-notes" / "qwen.md"
            qwen_text = qwen_path.read_text(encoding="utf-8")
            qwen_path.write_text(
                qwen_text.replace("## Output and data boundary", "## Removed section", 1),
                encoding="utf-8",
            )
            result = self.run_validator(root)

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("Provider note codex metadata keys mismatch", result.stdout)
        self.assertIn("Provider note codex field unsupported_invocation", result.stdout)
        self.assertIn("Provider note qwen is missing section ## Output and data boundary", result.stdout)

    def test_provider_notes_reject_forbidden_frontmatter_claims(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            self.make_fixture(root)
            claude_path = root / ".ai" / "provider-notes" / "claude.md"
            claude_text = claude_path.read_text(encoding="utf-8")
            claude_text = claude_text.replace(
                'route_status: "DEFAULT_DENY_NO_APPROVED_ROUTE"',
                'route_status: "ACTIVE"',
                1,
            ).replace(
                'model_alias_selection: "NONE"',
                'model_alias_selection: "latest"',
                1,
            )
            claude_path.write_text(claude_text, encoding="utf-8")
            result = self.run_validator(root)

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("Provider note claude field route_status must be DEFAULT_DENY_NO_APPROVED_ROUTE", result.stdout)
        self.assertIn("Provider note claude field model_alias_selection must be NONE", result.stdout)

    def test_provider_notes_reject_forbidden_body_claims(self) -> None:
        cases = (
            ("route_dispatch", "Dispatch: enabled", "active route or dispatch claim"),
            ("adapter_runtime", "Adapter runtime: enabled", "adapter or runtime activation claim"),
            ("credential", "Approved credential: provider-production", "approved credential claim"),
            ("model_alias", "Model alias: latest", "selected model alias claim"),
            ("retention", "Retention: 30 days", "retention promise claim"),
            ("numeric_budget", "Numeric budget: 4096", "numeric budget claim"),
            ("smoke_test", "Smoke test: PASSED", "smoke-test claim"),
        )
        for case_name, body_claim, expected_claim in cases:
            with self.subTest(case=case_name), tempfile.TemporaryDirectory() as temp_dir:
                root = Path(temp_dir)
                self.make_fixture(root)
                note_path = root / ".ai" / "provider-notes" / "codex.md"
                with note_path.open("a", encoding="utf-8") as stream:
                    stream.write(f"\n{body_claim}\n")
                result = self.run_validator(root)

                self.assertNotEqual(result.returncode, 0)
                self.assertIn(f"Provider note codex contains forbidden {expected_claim}", result.stdout)

    def test_i015_selected_contract_examples_pass(self) -> None:
        for contract_type in I015_CONTRACT_FILES:
            with self.subTest(contract_type=contract_type):
                schema, instance = self.load_i015_contract(REPO_ROOT, contract_type)
                self.assertEqual(self.i015_schema_errors(schema, instance), [])

        registry = yaml.safe_load(
            (REPO_ROOT / ".ai" / "schemas" / "extensions" / "registry.yaml").read_text(encoding="utf-8")
        )
        self.assertEqual(
            registry,
            {
                "schema_version": "1.0.0",
                "registry_status": "EMPTY_NO_REGISTERED_EXTENSIONS",
                "registered_extensions": [],
            },
        )

    def test_i015_version_hard_cutover_has_no_fallback(self) -> None:
        invalid_versions: tuple[object, ...] = (
            None,
            1,
            "v1.0.0",
            "1",
            "1.0",
            "01.0.0",
            "1.0.0-alpha",
            "1.0.0+build",
            " 1.0.0 ",
            "0.9.0",
            "2.0.0",
        )
        for contract_type in I015_CONTRACT_FILES:
            schema, valid = self.load_i015_contract(REPO_ROOT, contract_type)
            missing = copy.deepcopy(valid)
            missing.pop("contract_version")
            with self.subTest(contract_type=contract_type, version="missing"):
                self.assertTrue(self.i015_schema_errors(schema, missing))
            for invalid_version in invalid_versions:
                candidate = copy.deepcopy(valid)
                candidate["contract_version"] = invalid_version
                with self.subTest(contract_type=contract_type, version=invalid_version):
                    self.assertTrue(self.i015_schema_errors(schema, candidate))
            wrong_type = copy.deepcopy(valid)
            wrong_type["contract_type"] = "result" if contract_type != "result" else "task"
            with self.subTest(contract_type=contract_type, discriminator="wrong"):
                self.assertTrue(self.i015_schema_errors(schema, wrong_type))
            external_fallback = copy.deepcopy(valid)
            external_fallback.pop("contract_version")
            external_fallback["metadata"] = {"contract_version": "1.0.0"}
            with self.subTest(contract_type=contract_type, version="external-fallback"):
                self.assertTrue(self.i015_schema_errors(schema, external_fallback))

    def test_i015_required_fields_and_closed_objects_fail_closed(self) -> None:
        for contract_type in I015_CONTRACT_FILES:
            schema, valid = self.load_i015_contract(REPO_ROOT, contract_type)
            for field in schema["required"]:
                candidate = copy.deepcopy(valid)
                candidate.pop(field)
                with self.subTest(contract_type=contract_type, missing=field):
                    self.assertTrue(self.i015_schema_errors(schema, candidate))
            unknown = copy.deepcopy(valid)
            unknown["unknown_core_field"] = True
            with self.subTest(contract_type=contract_type, unknown="top-level"):
                self.assertTrue(self.i015_schema_errors(schema, unknown))
            empty_extensions = copy.deepcopy(valid)
            empty_extensions["extensions"] = {}
            with self.subTest(contract_type=contract_type, extensions="empty"):
                self.assertEqual(self.i015_schema_errors(schema, empty_extensions), [])
            unregistered_extension = copy.deepcopy(valid)
            unregistered_extension["extensions"] = {"org.oshethai.unselected": {}}
            with self.subTest(contract_type=contract_type, extensions="unregistered"):
                self.assertTrue(self.i015_schema_errors(schema, unregistered_extension))

        result_schema, result = self.load_i015_contract(REPO_ROOT, "result")
        for object_path in (("agent",), ("git",), ("tests",), ("tests", "executions", 0)):
            candidate = copy.deepcopy(result)
            target: object = candidate
            for part in object_path:
                target = target[part]  # type: ignore[index]
            target["unknown_nested_field"] = True  # type: ignore[index]
            with self.subTest(contract_type="result", unknown=object_path):
                self.assertTrue(self.i015_schema_errors(result_schema, candidate))

        review_schema, review = self.load_i015_contract(REPO_ROOT, "review")
        review_with_finding = copy.deepcopy(review)
        review_with_finding["verdict"] = "CHANGES_REQUIRED"
        review_with_finding["findings"] = [
            {
                "id": "FINDING-001",
                "severity": "MEDIUM",
                "category": "SCHEMA",
                "issue": "Synthetic finding.",
                "required_fix": "Correct the synthetic instance.",
            }
        ]
        self.assertEqual(self.i015_schema_errors(review_schema, review_with_finding), [])
        for object_path in (("reviewer",), ("findings", 0)):
            candidate = copy.deepcopy(review_with_finding)
            target = candidate
            for part in object_path:
                target = target[part]  # type: ignore[index]
            target["unknown_nested_field"] = True  # type: ignore[index]
            with self.subTest(contract_type="review", unknown=object_path):
                self.assertTrue(self.i015_schema_errors(review_schema, candidate))

        integration_schema, integration = self.load_i015_contract(REPO_ROOT, "integration")
        integration["checks"][0]["unknown_nested_field"] = True
        self.assertTrue(self.i015_schema_errors(integration_schema, integration))

    def test_i015_scalar_array_and_path_constraints_fail_closed(self) -> None:
        cases = (
            ("mission", ("title",), ""),
            ("mission", ("non_goals",), []),
            ("mission", ("human_decisions",), ["DUPLICATE", "DUPLICATE"]),
            ("task", ("allowed_paths",), ["../escape"]),
            ("task", ("forbidden_paths",), ["windows\\path"]),
            ("task", ("required_checks",), []),
            ("result", ("changes",), ["../escape"]),
            ("result", ("git", "base_commit"), "abc123"),
            ("review", ("reviewer", "actor_id"), ""),
            ("integration", ("included_commits",), ["abc123"]),
            ("handoff", ("summary",), ""),
            ("handoff", ("evidence",), ["DUPLICATE", "DUPLICATE"]),
        )
        for contract_type, object_path, value in cases:
            schema, valid = self.load_i015_contract(REPO_ROOT, contract_type)
            candidate = copy.deepcopy(valid)
            target = candidate
            for part in object_path[:-1]:
                target = target[part]  # type: ignore[index]
            target[object_path[-1]] = value  # type: ignore[index]
            with self.subTest(contract_type=contract_type, field=object_path):
                self.assertTrue(self.i015_schema_errors(schema, candidate))

    def test_i015_rcb_material_write_and_typed_no_commit_matrix(self) -> None:
        schema, selected_no_write = self.load_i015_contract(REPO_ROOT, "result")
        self.assertEqual(self.i015_schema_errors(schema, selected_no_write), [])

        material = copy.deepcopy(selected_no_write)
        material["material_write"] = True
        material["changes"] = [".ai/schemas/mission.schema.json"]
        material["git"].pop("no_commit_reason")
        material["git"]["result_commit"] = "23456789abcdef0123456789abcdef0123456789"
        self.assertEqual(self.i015_schema_errors(schema, material), [])

        invalid_material_cases = {
            "missing-result-commit": lambda value: value["git"].pop("result_commit"),
            "commit-plus-reason": lambda value: value["git"].update({"no_commit_reason": "NO_CHANGE_REQUIRED"}),
            "empty-changes": lambda value: value.update({"changes": []}),
            "abbreviated-commit": lambda value: value["git"].update({"result_commit": "abc123"}),
            "all-zero-placeholder": lambda value: value["git"].update({"result_commit": "0" * 40}),
            "all-one-placeholder": lambda value: value["git"].update({"result_commit": "1" * 40}),
        }
        for case_name, mutate in invalid_material_cases.items():
            candidate = copy.deepcopy(material)
            mutate(candidate)
            with self.subTest(material_write=True, case=case_name):
                self.assertTrue(self.i015_schema_errors(schema, candidate))

        valid_no_commit_cases = (
            ("SUBMITTED", "READ_ONLY_TASK"),
            ("BLOCKED", "BLOCKED_BEFORE_MATERIAL_WRITE"),
            ("FAILED", "FAILED_BEFORE_MATERIAL_WRITE"),
            ("SUBMITTED", "TEST_ONLY_NO_MATERIAL_WRITE"),
            ("SUBMITTED", "NO_CHANGE_REQUIRED"),
        )
        for status, reason in valid_no_commit_cases:
            candidate = copy.deepcopy(selected_no_write)
            candidate["status"] = status
            candidate["git"]["no_commit_reason"] = reason
            with self.subTest(material_write=False, reason=reason):
                self.assertEqual(self.i015_schema_errors(schema, candidate), [])

        invalid_no_commit = copy.deepcopy(selected_no_write)
        invalid_no_commit["git"]["no_commit_reason"] = "BLOCKED_BEFORE_MATERIAL_WRITE"
        self.assertTrue(self.i015_schema_errors(schema, invalid_no_commit))
        invalid_no_commit["status"] = "BLOCKED"
        invalid_no_commit["changes"] = [".ai/schemas/mission.schema.json"]
        self.assertTrue(self.i015_schema_errors(schema, invalid_no_commit))

    def test_i015_review_and_integration_readiness_fail_closed(self) -> None:
        review_schema, review = self.load_i015_contract(REPO_ROOT, "review")
        high_finding = {
            "id": "FINDING-HIGH-001",
            "severity": "HIGH",
            "category": "SECURITY",
            "issue": "Synthetic blocking finding.",
            "required_fix": "Resolve before readiness.",
        }
        approved_with_high = copy.deepcopy(review)
        approved_with_high["findings"] = [high_finding]
        self.assertTrue(self.i015_schema_errors(review_schema, approved_with_high))
        changes_without_finding = copy.deepcopy(review)
        changes_without_finding["verdict"] = "CHANGES_REQUIRED"
        self.assertTrue(self.i015_schema_errors(review_schema, changes_without_finding))

        integration_schema, integration = self.load_i015_contract(REPO_ROOT, "integration")
        for outcome in ("FAIL", "SKIPPED", "INCONCLUSIVE"):
            candidate = copy.deepcopy(integration)
            candidate["checks"][0]["outcome"] = outcome
            with self.subTest(ready_for_human=True, outcome=outcome):
                self.assertTrue(self.i015_schema_errors(integration_schema, candidate))

    def test_i015_foa_cross_contract_contradictions_fail_closed(self) -> None:
        cases = (
            ("task-mission", "task-packet.example.yaml", ("mission_id",), "OTHER", "task mission_id"),
            (
                "result-base",
                "result-contract.example.yaml",
                ("git", "base_commit"),
                "2222222222222222222222222222222222222222",
                "result base_commit",
            ),
            (
                "outside-path",
                "result-contract.example.yaml",
                ("changes",),
                ["docs/outside.md"],
                "outside task allowed_paths",
            ),
            (
                "forbidden-path",
                "task-packet.example.yaml",
                ("forbidden_paths",),
                [".ai/schemas/**"],
                "path rules overlap or cannot be proven disjoint",
            ),
            (
                "missing-result-check",
                "result-contract.example.yaml",
                ("tests",),
                {"overall": "NOT_RUN", "executions": []},
                "result tests do not represent every task required_check",
            ),
            (
                "fabricated-integration-commit",
                "integration.example.yaml",
                ("included_commits",),
                ["2222222222222222222222222222222222222222"],
                "included commits must equal evidenced material result commits",
            ),
            (
                "missing-integration-check",
                "integration.example.yaml",
                ("checks", 0, "id"),
                "other-check",
                "integration checks omit a task required_check",
            ),
            (
                "unknown-handoff-decision",
                "handoff.example.yaml",
                ("human_decisions",),
                ["HDEC-UNKNOWN"],
                "handoff references unknown human decisions",
            ),
            (
                "non-submitted-ready",
                "result-contract.example.yaml",
                ("status",),
                "BLOCKED",
                "integration cannot be ready when result status is not SUBMITTED",
            ),
            (
                "unknown-open-finding",
                "integration.example.yaml",
                ("open_findings",),
                ["FINDING-UNKNOWN"],
                "integration references unknown review findings",
            ),
        )
        for case_name, example_name, object_path, value, expected_error in cases:
            with self.subTest(case=case_name), tempfile.TemporaryDirectory() as temp_dir:
                root = Path(temp_dir)
                self.make_fixture(root)
                example_path = root / ".ai" / "examples" / example_name
                instance = yaml.safe_load(example_path.read_text(encoding="utf-8"))
                target = instance
                for part in object_path[:-1]:
                    target = target[part]  # type: ignore[index]
                target[object_path[-1]] = value  # type: ignore[index]
                example_path.write_text(yaml.safe_dump(instance, sort_keys=False), encoding="utf-8")
                result = self.run_validator(root)

                self.assertNotEqual(result.returncode, 0)
                self.assertIn(expected_error, result.stdout)

    def test_i015_security_remediation_negatives_fail_closed(self) -> None:
        result_schema, selected_no_write = self.load_i015_contract(REPO_ROOT, "result")
        integration_schema, integration = self.load_i015_contract(REPO_ROOT, "integration")
        for placeholder in ("0" * 40, "0" * 64):
            result_candidate = copy.deepcopy(selected_no_write)
            result_candidate["material_write"] = True
            result_candidate["changes"] = [".ai/schemas/mission.schema.json"]
            result_candidate["git"].pop("no_commit_reason")
            result_candidate["git"]["result_commit"] = placeholder
            with self.subTest(finding="SEC-I015-003", schema="result", placeholder=len(placeholder)):
                self.assertTrue(self.i015_schema_errors(result_schema, result_candidate))

            integration_candidate = copy.deepcopy(integration)
            integration_candidate["included_commits"] = [placeholder]
            with self.subTest(finding="SEC-I015-003", schema="integration", placeholder=len(placeholder)):
                self.assertTrue(self.i015_schema_errors(integration_schema, integration_candidate))

        fixture_cases = (
            (
                "SEC-I015-003-R1-base-commit-substitution",
                {
                    "result-contract.example.yaml": {
                        "material_write": True,
                        "changes": [".ai/schemas/mission.schema.json"],
                        "git": {
                            "base_commit": "4a4e0bd0def63f442a2f892b56fdac1792d0034f",
                            "branch": "content/v010-i015-contract-suite",
                            "result_commit": "4a4e0bd0def63f442a2f892b56fdac1792d0034f",
                        },
                    },
                    "integration.example.yaml": {
                        "included_commits": ["4a4e0bd0def63f442a2f892b56fdac1792d0034f"]
                    },
                },
                "material result_commit must differ from result base_commit",
            ),
            (
                "SEC-I015-002-omitted-blocking-review-finding",
                {
                    "review.example.yaml": {
                        "verdict": "CHANGES_REQUIRED",
                        "findings": [
                            {
                                "id": "FINDING-HIGH-001",
                                "severity": "HIGH",
                                "category": "SECURITY",
                                "issue": "Synthetic blocking finding.",
                                "required_fix": "Resolve before readiness.",
                            }
                        ],
                    }
                },
                "omits unresolved blocking review findings",
            ),
            (
                "SEC-I015-003-extra-fabricated-commit",
                {
                    "result-contract.example.yaml": {
                        "material_write": True,
                        "changes": [".ai/schemas/mission.schema.json"],
                        "git": {
                            "base_commit": "4a4e0bd0def63f442a2f892b56fdac1792d0034f",
                            "branch": "content/v010-i015-contract-suite",
                            "result_commit": "23456789abcdef0123456789abcdef0123456789",
                        },
                    },
                    "integration.example.yaml": {
                        "included_commits": [
                            "23456789abcdef0123456789abcdef0123456789",
                            "3456789abcdef0123456789abcdef0123456789a",
                        ]
                    },
                },
                "included commits must equal evidenced material result commits",
            ),
            (
                "SEC-I015-005-nonliteral-overlap",
                {
                    "task-packet.example.yaml": {
                        "allowed_paths": [".ai/schemas/**", ".ai/examples/**"],
                        "forbidden_paths": [".ai/schemas/private/**"],
                    }
                },
                "path rules overlap or cannot be proven disjoint",
            ),
        )
        for case_name, file_updates, expected_error in fixture_cases:
            with self.subTest(case=case_name), tempfile.TemporaryDirectory() as temp_dir:
                root = Path(temp_dir)
                self.make_fixture(root)
                for example_name, updates in file_updates.items():
                    example_path = root / ".ai" / "examples" / example_name
                    instance = yaml.safe_load(example_path.read_text(encoding="utf-8"))
                    instance.update(updates)
                    example_path.write_text(yaml.safe_dump(instance, sort_keys=False), encoding="utf-8")
                result = self.run_validator(root)

                self.assertNotEqual(result.returncode, 0)
                self.assertIn(expected_error, result.stdout)

        for outcome in ("FAIL", "SKIPPED", "INCONCLUSIVE"):
            with self.subTest(finding="SEC-I015-001", result_outcome=outcome), tempfile.TemporaryDirectory() as temp_dir:
                root = Path(temp_dir)
                self.make_fixture(root)
                result_path = root / ".ai" / "examples" / "result-contract.example.yaml"
                result_instance = yaml.safe_load(result_path.read_text(encoding="utf-8"))
                result_instance["tests"]["overall"] = "FAIL" if outcome == "FAIL" else "INCONCLUSIVE"
                result_instance["tests"]["executions"][0]["outcome"] = outcome
                result_instance["tests"]["executions"][0]["exit_code"] = 1 if outcome == "FAIL" else None
                result_path.write_text(yaml.safe_dump(result_instance, sort_keys=False), encoding="utf-8")
                validation_result = self.run_validator(root)

                self.assertNotEqual(validation_result.returncode, 0)
                self.assertIn("cannot be ready unless result tests overall is PASS", validation_result.stdout)
                self.assertIn("cannot be ready without PASS result evidence", validation_result.stdout)

        for mode, expected_reason in (
            ("READ_ONLY", "READ_ONLY_TASK"),
            ("TEST_ONLY", "TEST_ONLY_NO_MATERIAL_WRITE"),
        ):
            with self.subTest(finding="SEC-I015-004", mode=mode), tempfile.TemporaryDirectory() as temp_dir:
                root = Path(temp_dir)
                self.make_fixture(root)
                task_path = root / ".ai" / "examples" / "task-packet.example.yaml"
                task_instance = yaml.safe_load(task_path.read_text(encoding="utf-8"))
                task_instance["mode"] = mode
                task_path.write_text(yaml.safe_dump(task_instance, sort_keys=False), encoding="utf-8")
                validation_result = self.run_validator(root)

                self.assertNotEqual(validation_result.returncode, 0)
                self.assertIn("no_commit_reason contradicts task mode or result status", validation_result.stdout)
                self.assertNotEqual(selected_no_write["git"]["no_commit_reason"], expected_reason)

    def test_validator_requires_exact_repository_delete_prohibition(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            self.make_fixture(root)
            policy_path = root / ".ai" / "policies" / "github-operations.yaml"
            policy = yaml.safe_load(policy_path.read_text(encoding="utf-8"))
            policy["prohibited_actions"] = []
            policy_path.write_text(yaml.safe_dump(policy, sort_keys=False), encoding="utf-8")
            result = self.run_validator(root)

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("GitHub prohibited actions must be exactly repository-delete", result.stdout)

    def test_validator_rejects_repository_delete_allowed_example(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            self.make_fixture(root)
            policy_path = root / ".ai" / "policies" / "github-operations.yaml"
            policy = yaml.safe_load(policy_path.read_text(encoding="utf-8"))
            policy["action_classes"]["DESTRUCTIVE"]["examples"].append(
                "  ＤＥＬＥＴＥ _ \u2011 REPOSITORY  "
            )
            policy_path.write_text(yaml.safe_dump(policy, sort_keys=False), encoding="utf-8")
            result = self.run_validator(root)

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("repository-delete must not appear in allowed GitHub action examples", result.stdout)

    def test_unapproved_provider_route_fails_closed(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            self.make_fixture(root)
            route_path = root / ".ai" / "provider-routes" / "ai-service-route-registry.yaml"
            text = route_path.read_text(encoding="utf-8")
            text = text.replace("dispatch_enabled: false", "dispatch_enabled: true", 1)
            route_path.write_text(text, encoding="utf-8")
            result = self.run_validator(root)

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("Candidate route is unexpectedly enabled", result.stdout)

    def test_complete_metadata_gate_passes(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            self.make_fixture(root)
            now = datetime.now(timezone.utc)
            record = self.make_gate_record(root, now)
            result = self.run_gate(root, record, now)

        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertIn("GITHUB_OPERATION_GATE_PASS", result.stdout)

    def test_direct_gh_gate_does_not_require_ai_provider_route(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            self.make_fixture(root)
            now = datetime.now(timezone.utc)
            policy_path = root / ".ai" / "policies" / "github-operations.yaml"
            policy = yaml.safe_load(policy_path.read_text(encoding="utf-8"))
            direct_route = policy["direct_gh_execution_routes"][0]
            direct_route["activated_at"] = (now - timedelta(minutes=1)).isoformat()
            direct_route["expires_at"] = (now + timedelta(days=1)).isoformat()
            policy_path.write_text(yaml.safe_dump(policy, sort_keys=False), encoding="utf-8")

            credential_path = root / ".ai" / "policies" / "github-credential-profiles.yaml"
            credentials = yaml.safe_load(credential_path.read_text(encoding="utf-8"))
            profile = next(item for item in credentials["profiles"] if item["credential_profile_id"] == direct_route["credential_profile_id"])
            profile["expires_at"] = (now + timedelta(days=1)).isoformat()
            credential_path.write_text(yaml.safe_dump(credentials, sort_keys=False), encoding="utf-8")

            evidence = {
                name: {"satisfied": True, "refs": [f"EVD-{name}"]}
                for name in policy["required_evidence_flags"]
            }
            evidence[policy["execution_route_evidence_flags"]["DIRECT_GH_CLI"]] = {
                "satisfied": True,
                "refs": ["EVD-direct-gh-route"],
            }
            record = {
                "schema_version": "1.0.0",
                "gate_id": "GHG-DIRECT-GH-001",
                "assignment_id": "ASN-DIRECT-GH-001",
                "session_id": "SESSION-DIRECT-GH-001",
                "human_authority_ref": "ADR-0006",
                "actor": {
                    "role_id": "release-evidence-agent",
                    "specialist_profile_id": "github-manager",
                    "execution_route_kind": "DIRECT_GH_CLI",
                    "execution_route_id": direct_route["execution_route_id"],
                    "provider_route_id": None,
                    "credential_profile_id": direct_route["credential_profile_id"],
                },
                "scope": {
                    "organization": "OSHEThai",
                    "repository": "oshe-platform",
                    "action_class": "METADATA",
                    "action": "issue-comment",
                    "target": "test-target",
                    "expected_pre_state_digest": "sha256:" + "0" * 64,
                    "expected_post_state": "Expected test state",
                    "exact_commit_or_configuration_identity": "test-identity",
                },
                "execution": {
                    "command": ["gh", "issue", "comment", "--repo", "OSHEThai/oshe-platform", "1", "--body", "test"],
                    "command_digest": "sha256:" + hashlib.sha256(
                        json.dumps(
                            ["gh", "issue", "comment", "--repo", "OSHEThai/oshe-platform", "1", "--body", "test"],
                            separators=(",", ":"),
                            ensure_ascii=False,
                        ).encode("utf-8")
                    ).hexdigest(),
                },
                "evidence": evidence,
                "unresolved_blockers": [],
                "independent_review": {"reviewer_role_id": None, "reviewer_assignment_id": None, "disposition": "NOT_REQUIRED", "review_ref": None},
                "requested_at": (now - timedelta(minutes=1)).isoformat(),
                "expires_at": (now + timedelta(minutes=10)).isoformat(),
                "external_authority_ref": None,
            }
            result = self.run_gate(root, record, now)

        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertIn("GITHUB_OPERATION_GATE_PASS", result.stdout)

    def test_direct_gh_gate_denies_action_class_mismatch(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            self.make_fixture(root)
            now = datetime.now(timezone.utc)
            record = self.make_direct_gh_gate_record(root, now)
            record["scope"]["action_class"] = "MERGE"
            result = self.run_gate(root, record, now)

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("direct gh scope action class does not match the exact command", result.stdout)

    def test_direct_gh_executor_requires_passing_embedded_command_gate(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            self.make_fixture(root)
            now = datetime.now(timezone.utc)
            record = self.make_direct_gh_gate_record(root, now)
            result = self.run_direct_gh_executor(root, record, now)

        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertIn("DIRECT_GH_EXECUTION_DRY_RUN_PASS", result.stdout)

    def test_direct_gh_executor_denies_clock_override_for_execution(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            self.make_fixture(root)
            now = datetime.now(timezone.utc)
            record = self.make_direct_gh_gate_record(root, now)
            result = self.run_direct_gh_executor(root, record, now, dry_run=False)

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("--now is permitted only with --dry-run", result.stderr)

    def test_incomplete_evidence_gate_denies(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            self.make_fixture(root)
            now = datetime.now(timezone.utc)
            record = self.make_gate_record(root, now)
            record["evidence"]["required_checks_passed"]["satisfied"] = False
            result = self.run_gate(root, record, now)

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("evidence is not satisfied: required_checks_passed", result.stdout)

    def test_high_impact_gate_requires_independent_review(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            self.make_fixture(root)
            now = datetime.now(timezone.utc)
            record = self.make_gate_record(root, now, action_class="MERGE")
            result = self.run_gate(root, record, now)

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("high-impact operation requires", result.stdout)

    def test_high_impact_gate_passes_with_independent_review(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            self.make_fixture(root)
            now = datetime.now(timezone.utc)
            record = self.make_gate_record(root, now, action_class="MERGE", independent_review=True)
            result = self.run_gate(root, record, now)

        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertIn("GITHUB_OPERATION_GATE_PASS", result.stdout)

    def test_repository_delete_aliases_and_format_variants_are_always_denied(self) -> None:
        variants = (
            "repository-delete",
            "delete-repository",
            "repo-delete",
            "RePoSiToRy_DeLeTe",
            "  delete   repository  ",
            "ＲＥＰＯＳＩＴＯＲＹ－ＤＥＬＥＴＥ",
            "repo\u2011delete",
        )

        for action in variants:
            with self.subTest(action=action), tempfile.TemporaryDirectory() as temp_dir:
                root = Path(temp_dir)
                self.make_fixture(root)
                now = datetime.now(timezone.utc)
                record = self.make_gate_record(
                    root,
                    now,
                    action_class="DESTRUCTIVE",
                    independent_review=True,
                )
                record["scope"]["action"] = action
                result = self.run_gate(root, record, now)

                self.assertNotEqual(result.returncode, 0)
                self.assertIn("GitHub action is always prohibited: repository-delete", result.stdout)
                self.assertNotIn("GITHUB_OPERATION_GATE_PASS", result.stdout)

    def test_force_push_passes_with_complete_independent_reviewed_gate(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            self.make_fixture(root)
            now = datetime.now(timezone.utc)
            record = self.make_gate_record(
                root,
                now,
                action_class="DESTRUCTIVE",
                independent_review=True,
            )
            record["scope"]["action"] = "force-push"
            result = self.run_gate(root, record, now)

        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertIn("GITHUB_OPERATION_GATE_PASS", result.stdout)

    def test_gate_evaluator_denies_prohibited_action_policy_divergence(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            self.make_fixture(root)
            now = datetime.now(timezone.utc)
            record = self.make_gate_record(
                root,
                now,
                action_class="DESTRUCTIVE",
                independent_review=True,
            )
            record["scope"]["action"] = "force-push"
            policy_path = root / ".ai" / "policies" / "github-operations.yaml"
            policy = yaml.safe_load(policy_path.read_text(encoding="utf-8"))
            policy["prohibited_actions"] = []
            policy_path.write_text(yaml.safe_dump(policy, sort_keys=False), encoding="utf-8")
            result = self.run_gate(root, record, now)

        self.assertNotEqual(result.returncode, 0)
        self.assertIn(
            "GitHub prohibited action policy diverges from the hard-coded safety baseline",
            result.stdout,
        )
        self.assertNotIn("GITHUB_OPERATION_GATE_PASS", result.stdout)


if __name__ == "__main__":
    unittest.main()
