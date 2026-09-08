"""Bounded publisher tests. All transport targets are temporary local repos."""
from __future__ import annotations

import copy
from datetime import datetime, timedelta, timezone
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import unittest
from unittest import mock

import yaml

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / ".ai" / "tools"))
import evaluate_github_operation as gate
import git_push_validation as push


class BoundedPushTests(unittest.TestCase):
    def setUp(self) -> None:
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.base = Path(self.temp.name).resolve()
        self.wt = self.base / "worktree"
        self.bare = self.base / "remote.git"
        self.git = str(Path(shutil.which("git") or "git").resolve())
        minimal = {k: v for k, v in os.environ.items() if k.upper() in {
            "PATH", "SYSTEMROOT", "WINDIR", "TEMP", "TMP", "PATHEXT",
        }}
        self.env_patch = mock.patch.dict(os.environ, minimal, clear=True)
        self.env_patch.start()
        self.addCleanup(self.env_patch.stop)
        self.git_env = dict(minimal, GIT_CONFIG_NOSYSTEM="1", GIT_CONFIG_GLOBAL=os.devnull,
                            GIT_TERMINAL_PROMPT="0")
        self.local(["init", "--bare", str(self.bare)])
        self.local(["init", "-b", "fix/audit-20260908-push-gate", str(self.wt)])
        for key, value in (("user.name", "Synthetic"), ("user.email", "synthetic@example.invalid"),
                           ("remote.origin.url", push.REMOTE)):
            self.local(["-C", str(self.wt), "config", key, value])
        self.local(["-C", str(self.wt), "commit", "--allow-empty", "-m", "base"])
        self.old = self.local(["-C", str(self.wt), "rev-parse", "HEAD"]).strip().decode()
        self.local(["-C", str(self.wt), "push", str(self.bare),
                    "HEAD:refs/heads/fix/audit-20260908-tenant-stores"])
        self.local(["-C", str(self.wt), "commit", "--allow-empty", "-m", "candidate"])
        self.head = self.local(["-C", str(self.wt), "rev-parse", "HEAD"]).strip().decode()
        self.ai = self.base / ".ai"
        for folder in ("policies", "schemas"):
            shutil.copytree(ROOT / ".ai" / folder, self.ai / folder)
        now = datetime.now(timezone.utc)
        policy_path = self.ai / "policies/github-operations.yaml"
        policy = yaml.safe_load(policy_path.read_text())
        route = policy["direct_git_push_routes"][0]
        route["activated_at"] = (now - timedelta(minutes=1)).isoformat()
        route["expires_at"] = (now + timedelta(hours=1)).isoformat()
        policy_path.write_text(yaml.safe_dump(policy), encoding="utf-8")
        credentials_path = self.ai / "policies/github-credential-profiles.yaml"
        credentials = yaml.safe_load(credentials_path.read_text())
        for item in credentials["profiles"]:
            if item["credential_profile_id"] == route["credential_profile_id"]:
                item["expires_at"] = (now + timedelta(hours=1)).isoformat()
        credentials_path.write_text(yaml.safe_dump(credentials), encoding="utf-8")
        self.ai_patch = mock.patch.object(gate, "AI_ROOT", self.ai)
        self.ai_patch.start()
        self.addCleanup(self.ai_patch.stop)
        self.record = {
            "schema_version": "1.0.0", "gate_id": "GHG-SYNTHETIC-PUSH-001",
            "assignment_id": "ASN-SYNTHETIC-RELEASE", "session_id": "SESSION-SYNTHETIC",
            "human_authority_ref": "ADR-0006",
            "actor": {"role_id": "release-evidence-agent", "specialist_profile_id": "github-manager",
                      "execution_route_kind": "DIRECT_GIT_PUSH", "execution_route_id": route["execution_route_id"],
                      "provider_route_id": None, "credential_profile_id": route["credential_profile_id"]},
            "scope": {"organization": "OSHEThai", "repository": "oshe-platform", "action": "branch-push",
                      "action_class": "BRANCH_PR", "target": "refs/heads/fix/audit-20260908-tenant-stores",
                      "expected_pre_state_digest": "sha256:" + "0" * 64,
                      "expected_post_state": self.head, "exact_commit_or_configuration_identity": self.head},
            "evidence": {k: {"satisfied": True, "refs": ["synthetic:" + k]} for k in policy["required_evidence_flags"]},
            "unresolved_blockers": [], "requested_at": (now - timedelta(seconds=1)).isoformat(),
            "expires_at": (now + timedelta(minutes=10)).isoformat(), "external_authority_ref": "synthetic-owner-approval",
            "independent_review": {"reviewer_role_id": "security-privacy-product-safety-agent",
                                   "reviewer_assignment_id": "ASN-SYNTHETIC-SECURITY", "disposition": "PASS",
                                   "review_ref": str(self.base / "review.json")},
        }
        self.record["evidence"]["approved_execution_route_validated"] = {"satisfied": True, "refs": ["synthetic:route"]}
        spec = {
            "worktree": str(self.wt), "source_branch": "fix/audit-20260908-push-gate",
            "source_commit": self.head, "destination_branch": "fix/audit-20260908-tenant-stores",
            "remote_url": push.REMOTE, "expected_remote_oid": self.old,
            "git_executable": self.git, "git_executable_sha256": hashlib.sha256(Path(self.git).read_bytes()).hexdigest(),
        }
        for field, name, extra in (
            ("review_evidence", "review.json", {"reviewer_assignment_id": "ASN-SYNTHETIC-SECURITY", "producer_assignment_id": "ASN-SYNTHETIC-PRODUCER"}),
            ("test_evidence", "tests.json", {"passed": 1, "failed": 0, "skipped": 0}),
        ):
            path = self.base / name
            path.write_text(json.dumps(dict(disposition="PASS", target_commit=self.head, **extra)), encoding="utf-8")
            spec[field] = {"path": str(path), "sha256": hashlib.sha256(path.read_bytes()).hexdigest()}
        self.record["execution"] = {"push": spec, "command": push.command_for(spec)}
        self.refresh_command()
        original_git = push._git

        def isolated_git(spec, args, env, **kwargs):
            return original_git(spec, args, dict(env, GIT_CONFIG_NOSYSTEM="1", GIT_CONFIG_GLOBAL=os.devnull), **kwargs)

        self.original_git = original_git

        self.git_patch = mock.patch.object(push, "_git", side_effect=isolated_git)
        self.git_patch.start()
        self.addCleanup(self.git_patch.stop)
        self.remote_patch = mock.patch.object(push, "remote_oid", side_effect=self.local_remote_oid)
        self.remote_patch.start()
        self.addCleanup(self.remote_patch.stop)
        self.transport_patch = mock.patch.object(push, "run_push", side_effect=self.local_push)
        self.transport = self.transport_patch.start()
        self.addCleanup(self.transport_patch.stop)
        self.record["scope"]["expected_pre_state_digest"] = push.digest(push.snapshot(self.record))

    def local(self, args):
        result = subprocess.run([self.git, "-c", "core.hooksPath=" + str(self.base / "no-hooks"),
                                 "-c", "commit.gpgSign=false", *args], env=self.git_env,
                                capture_output=True, check=True)
        return result.stdout

    def refresh_command(self):
        self.record["execution"]["command"] = push.command_for(self.record["execution"]["push"])
        self.record["execution"]["command_digest"] = gate.command_digest(self.record["execution"]["command"])

    def local_remote_oid(self, spec, env):
        out = self.local(["ls-remote", "--refs", str(self.bare), "refs/heads/" + spec["destination_branch"]]).decode()
        return out.split("\t")[0] if out.strip() else None

    def local_push(self, spec, env):
        command = push.command_for(spec)
        command[command.index(push.REMOTE)] = str(self.bare)
        return subprocess.run(command, env=self.git_env, capture_output=True, check=False)

    def test_valid_gate_and_real_local_transfer_readback(self):
        self.assertEqual([], gate.evaluate(self.record, datetime.now(timezone.utc)))
        self.assertEqual(0, push.execute_push(self.record, dry_run=True))
        self.transport.assert_not_called()
        self.assertEqual(0, push.execute_push(self.record))
        self.assertEqual(self.head, self.local_remote_oid(self.record["execution"]["push"], {}))
        self.assertEqual(1, self.transport.call_count)

    def test_new_exact_branch_creation(self):
        spec = self.record["execution"]["push"]
        spec["destination_branch"] = "fix/audit-20260908-integration"
        spec["expected_remote_oid"] = None
        self.record["scope"]["target"] = "refs/heads/" + spec["destination_branch"]
        self.refresh_command()
        self.record["scope"]["expected_pre_state_digest"] = push.digest(push.snapshot(self.record))
        self.assertEqual(0, push.execute_push(self.record))

    def test_extra_options_and_refspecs_denied(self):
        for value in ("--force", "--force-with-lease", "--delete", "--mirror", "--all", "--tags", "--prune", "+HEAD:refs/heads/main", ":refs/heads/main", "--receive-pack=evil"):
            with self.subTest(value=value):
                record = copy.deepcopy(self.record)
                record["execution"]["command"].append(value)
                record["execution"]["command_digest"] = gate.command_digest(record["execution"]["command"])
                self.assertTrue(gate.evaluate(record, datetime.now(timezone.utc)))

    def test_wrong_scope_commit_remote_branch_and_review_denied(self):
        for key, value in (("remote_url", "https://example.invalid/repo.git"), ("destination_branch", "main"),
                           ("source_branch", "main"), ("source_commit", "HEAD"), ("expected_remote_oid", "+HEAD")):
            with self.subTest(key=key):
                record = copy.deepcopy(self.record)
                record["execution"]["push"][key] = value
                self.assertIsNone(push.classify_push(record)[0])
        self.record["independent_review"]["reviewer_assignment_id"] = self.record["assignment_id"]
        self.assertIsNone(push.classify_push(self.record)[0])

    def test_unsafe_config_rejected(self):
        for key, value in (("remote.origin.pushurl", "https://example.invalid/repo"),
                           ("url.https://example.invalid/.insteadOf", "https://github.com/"),
                           ("push.followTags", "true"), ("core.hooksPath", "hooks"),
                           ("core.fsmonitor", "evil"), ("filter.evil.clean", "evil"),
                           ("credential.helper", "!evil"), ("http.sslVerify", "false")):
            with self.subTest(key=key):
                self.local(["-C", str(self.wt), "config", key, value])
                with self.assertRaises(ValueError):
                    push.snapshot(self.record)
                self.local(["-C", str(self.wt), "config", "--unset-all", key])
        self.transport.assert_not_called()

    def test_environment_override_denied(self):
        for key in ("GIT_CONFIG_COUNT", "GIT_EXEC_PATH", "GIT_DIR", "GIT_WORK_TREE", "GIT_SSH_COMMAND", "HTTPS_PROXY", "SSH_ASKPASS"):
            with self.subTest(key=key), mock.patch.dict(os.environ, {key: "unsafe"}):
                with self.assertRaises(ValueError):
                    push.snapshot(self.record)

    def test_credential_selectors_never_reach_transport(self):
        for key, value in (
            ("core.askPass", "arbitrary-helper-command"),
            ("credential.username", "alternate-identity"),
            ("credential.provider", "alternate"),
            ("credential.credentialStore", "plaintext"),
            ("credential.useHttpPath", "true"),
            ("credential.https://github.com.username", "alternate-identity"),
            ("credential.https://github.com.helper", "manager"),
        ):
            with self.subTest(key=key):
                self.local(["-C", str(self.wt), "config", key, value])
                with mock.patch.object(push, "remote_oid") as remote:
                    with self.assertRaisesRegex(ValueError, "configuration"):
                        push.execute_push(self.record)
                    remote.assert_not_called()
                self.transport.assert_not_called()
                self.local(["-C", str(self.wt), "config", "--unset-all", key])

    def test_credential_environment_never_reaches_git(self):
        for key in ("GCM_CREDENTIAL_STORE", "GCM_PROVIDER", "GCM_CONFIG_HOME",
                    "GCM_INTERACTIVE", "GCM_FUTURE_SELECTOR", "GH_TOKEN", "GITHUB_TOKEN"):
            with self.subTest(key=key), mock.patch.dict(os.environ, {key: "synthetic"}):
                with mock.patch.object(push, "_git") as diagnostic:
                    with self.assertRaisesRegex(ValueError, "credential selector"):
                        push.execute_push(self.record)
                    diagnostic.assert_not_called()
                self.transport.assert_not_called()

    def test_alternate_home_credential_config_denied(self):
        home_dir = self.base / "synthetic-home"
        home_dir.mkdir()
        (home_dir / ".gitconfig").write_text(
            "[core]\n askPass = arbitrary-helper-command\n"
            "[credential]\n username = alternate-identity\n", encoding="utf-8")
        isolated_env = dict(self.git_env, HOME=str(home_dir), XDG_CONFIG_HOME=str(home_dir))
        isolated_env.pop("GIT_CONFIG_GLOBAL")
        def home_git(spec, args, env, **kwargs):
            return self.original_git(spec, args, isolated_env, **kwargs)

        with mock.patch.object(push, "_git", side_effect=home_git):
            with mock.patch.object(push, "remote_oid") as remote:
                with self.assertRaisesRegex(ValueError, "configuration"):
                    push.execute_push(self.record)
                remote.assert_not_called()
        self.transport.assert_not_called()

    def test_stale_remote_and_non_fast_forward_denied(self):
        with mock.patch.object(push, "remote_oid", return_value="f" * 40):
            with self.assertRaises(ValueError):
                push.execute_push(self.record)
        self.record["execution"]["push"]["expected_remote_oid"] = "f" * 40
        with mock.patch.object(push, "remote_oid", return_value="f" * 40):
            with self.assertRaises(ValueError):
                push.snapshot(self.record)
        self.transport.assert_not_called()

    def test_changed_config_and_dirty_worktree_denied(self):
        self.local(["-C", str(self.wt), "config", "user.name", "changed"])
        with self.assertRaises(ValueError):
            push.execute_push(self.record)
        (self.wt / "untracked.txt").write_text("synthetic")
        with self.assertRaises(ValueError):
            push.snapshot(self.record)
        self.transport.assert_not_called()

    def test_evidence_tampering_and_stale_head_denied(self):
        path = Path(self.record["execution"]["push"]["review_evidence"]["path"])
        path.write_text("{}")
        with self.assertRaises(ValueError):
            push.snapshot(self.record)
        self.local(["-C", str(self.wt), "commit", "--allow-empty", "-m", "unreviewed"])
        with self.assertRaises(ValueError):
            push.snapshot(self.record)

    def test_failed_gate_consumed_and_not_replayed(self):
        self.transport.side_effect = None
        self.transport.return_value = subprocess.CompletedProcess([], 1, b"", b"synthetic failure")
        self.assertEqual(1, push.execute_push(self.record))
        with self.assertRaises(FileExistsError):
            push.execute_push(self.record)
        self.assertEqual(1, self.transport.call_count)

    def test_second_preflight_change_and_bad_post_readback(self):
        original = push.snapshot(self.record)
        with mock.patch.object(push, "snapshot", side_effect=[original, dict(original, config_digest="changed")]):
            with self.assertRaises(ValueError):
                push.execute_push(self.record)
        self.transport.assert_not_called()
        self.record["gate_id"] = "GHG-SYNTHETIC-PUSH-002"
        self.transport.side_effect = None
        self.transport.return_value = subprocess.CompletedProcess([], 0, b"", b"")
        with self.assertRaises(ValueError):
            push.execute_push(self.record)

    def test_expired_gate_and_missing_checks_denied(self):
        self.record["expires_at"] = (datetime.now(timezone.utc) - timedelta(seconds=1)).isoformat()
        with self.assertRaises(ValueError):
            push.execute_push(self.record)
        self.record["evidence"]["required_checks_passed"]["satisfied"] = False
        self.assertTrue(gate.evaluate(self.record, datetime.now(timezone.utc)))
        self.transport.assert_not_called()

    def test_real_divergent_history_denied(self):
        self.local(["-C", str(self.wt), "checkout", "--detach", self.old])
        self.local(["-C", str(self.wt), "commit", "--allow-empty", "-m", "divergent"])
        divergent = self.local(["-C", str(self.wt), "rev-parse", "HEAD"]).decode().strip()
        self.local(["-C", str(self.wt), "checkout", self.record["execution"]["push"]["source_branch"]])
        self.record["execution"]["push"]["expected_remote_oid"] = divergent
        with mock.patch.object(push, "remote_oid", return_value=divergent):
            with self.assertRaisesRegex(ValueError, "fast-forward"):
                push.snapshot(self.record)
        self.transport.assert_not_called()

    def test_detached_head_and_changed_binary_denied(self):
        spec = self.record["execution"]["push"]
        saved = spec["git_executable_sha256"]
        spec["git_executable_sha256"] = "0" * 64
        with self.assertRaisesRegex(ValueError, "executable changed"):
            push.snapshot(self.record)
        spec["git_executable_sha256"] = saved
        self.local(["-C", str(self.wt), "checkout", "--detach", self.head])
        with self.assertRaises(ValueError):
            push.snapshot(self.record)
        self.transport.assert_not_called()


if __name__ == "__main__":
    unittest.main()
