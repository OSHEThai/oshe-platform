from __future__ import annotations

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


REPO_ROOT = Path(__file__).resolve().parents[1]


class AgentOsValidatorTests(unittest.TestCase):
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

    def make_approved_provider_fixture(self, root: Path) -> str:
        route_path = root / ".ai" / "provider-routes" / "ai-service-route-registry.yaml"
        routes = yaml.safe_load(route_path.read_text(encoding="utf-8"))
        route = routes["routes"][0]
        route_id = route["provider_route_id"]
        routes["active_route_ids"] = [route_id]
        routes["approved_route_ids"] = [route_id]
        routes["enabled_route_ids"] = [route_id]
        route["approved_roles"] = ["project-management-agent"]
        route["allowed_data_classes"] = ["PUBLIC"]
        route["data_policy_review"] = {
            "review_id": "PDR-QWEN-LOCAL-001",
            "status": "APPROVED",
            "allowed_data_classes": ["PUBLIC"],
        }
        route["lifecycle"]["status"] = "APPROVED"
        route["lifecycle"]["dispatch_enabled"] = True
        route_path.write_text(yaml.safe_dump(routes, sort_keys=False), encoding="utf-8")

        model_path = root / ".ai" / "policies" / "model-registry.yaml"
        models = yaml.safe_load(model_path.read_text(encoding="utf-8"))
        model = models["models"][0]
        model_id = model["model_record_id"]
        models["approved_model_refs"] = [model_id]
        models["enabled_model_record_ids"] = [model_id]
        model["approval_status"] = "APPROVED"
        model["dispatch_enabled"] = True
        model["approved_roles"] = ["project-management-agent"]
        model["allowed_data_classes"] = ["PUBLIC"]
        model_path.write_text(yaml.safe_dump(models, sort_keys=False), encoding="utf-8")

        routing_path = root / ".ai" / "policies" / "provider-routing.yaml"
        routing = yaml.safe_load(routing_path.read_text(encoding="utf-8"))
        routing["routing_status"] = "APPROVED_ROUTES_CONFIGURED"
        role_route = routing["role_routes"]["project-management-agent"]
        role_route["primary_route_id"] = route_id
        role_route["selection_state"] = "APPROVED"
        role_route["dispatch_enabled"] = True
        routing_path.write_text(yaml.safe_dump(routing, sort_keys=False), encoding="utf-8")

        review_path = root / ".ai" / "provider-routes" / "provider-policy-review-register.yaml"
        reviews = yaml.safe_load(review_path.read_text(encoding="utf-8"))
        review = next(item for item in reviews["reviews"] if item["review_id"] == "PDR-QWEN-LOCAL-001")
        review["status"] = "APPROVED"
        review["route_decision"] = "APPROVED"
        review_path.write_text(yaml.safe_dump(reviews, sort_keys=False), encoding="utf-8")
        return route_id

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
        self.assertIn("Dispatch-enabled route is absent from enabled IDs", result.stdout)

    def test_coherent_approved_provider_fixture_passes(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            self.make_fixture(root)
            self.make_approved_provider_fixture(root)
            result = self.run_validator(root)

        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertIn("provider_routes_enabled=1", result.stdout)

    def test_approved_route_requires_enabled_model(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            self.make_fixture(root)
            self.make_approved_provider_fixture(root)
            model_path = root / ".ai" / "policies" / "model-registry.yaml"
            models = yaml.safe_load(model_path.read_text(encoding="utf-8"))
            models["enabled_model_record_ids"] = []
            model_path.write_text(yaml.safe_dump(models, sort_keys=False), encoding="utf-8")
            result = self.run_validator(root)

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("Approved and enabled model ID sets must match", result.stdout)

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
