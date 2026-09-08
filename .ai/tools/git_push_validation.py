"""Narrow, procedural audit-branch publisher; never an arbitrary Git runner.

The host, installed Git/GCM and reviewed evidence authors remain trusted. This
is not an OS sandbox or an atomic remote compare-and-swap implementation.
"""
from __future__ import annotations

import hashlib
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
from datetime import datetime
from typing import Any

REMOTE = "https://github.com/OSHEThai/oshe-platform.git"
BRANCHES = frozenset("fix/audit-20260908-" + name for name in (
    "push-gate", "integration", "tenant-stores", "ci-coverage",
    "baseline-contracts", "current-docs",
))
OID = re.compile(r"[0-9a-f]{40}\Z")
SAFE_OVERRIDES = (
    "core.fsmonitor=false", "filter.lfs.process=", "filter.lfs.clean=",
    "filter.lfs.smudge=", "filter.lfs.required=false",
    "http.followRedirects=false", "push.gpgSign=false",
)


def digest(value: Any) -> str:
    return "sha256:" + hashlib.sha256(json.dumps(
        value, sort_keys=True, separators=(",", ":"), ensure_ascii=False,
    ).encode()).hexdigest()


def _require(condition: bool, message: str) -> None:
    if not condition:
        raise ValueError(message)


def safe_environment() -> dict[str, str]:
    env = dict(os.environ)
    for key in env:
        upper = key.upper()
        if upper.startswith("GIT_") and upper != "GIT_PAGER":
            raise ValueError("ambient Git environment override")
        if upper.startswith("GCM_") or upper in {
            "GH_TOKEN", "GITHUB_TOKEN", "GH_ENTERPRISE_TOKEN", "GITHUB_ENTERPRISE_TOKEN",
        }:
            raise ValueError("ambient credential selector")
        if upper in {
            "SSH_ASKPASS", "SSH_AUTH_SOCK", "SSH_ASKPASS_REQUIRE",
            "HTTPS_PROXY", "HTTP_PROXY", "ALL_PROXY", "CURL_CA_BUNDLE",
            "SSL_CERT_FILE", "SSL_CERT_DIR", "LD_PRELOAD", "DYLD_INSERT_LIBRARIES",
        }:
            raise ValueError("ambient transport/executable override")
    env.pop("GIT_PAGER", None)
    env.update(GIT_TERMINAL_PROMPT="0", GCM_INTERACTIVE="never",
               GIT_NO_REPLACE_OBJECTS="1", GIT_ATTR_NOSYSTEM="1")
    return env


def command_for(spec: dict[str, Any]) -> list[str]:
    prefix = [spec["git_executable"], "--no-optional-locks"]
    for override in SAFE_OVERRIDES:
        prefix += ["-c", override]
    return prefix + [
        "-C", spec["worktree"], "push", "--porcelain", "--no-verify",
        "--no-follow-tags", "--recurse-submodules=no", "--",
        spec["remote_url"],
        f"{spec['source_commit']}:refs/heads/{spec['destination_branch']}",
    ]


def classify_push(record: dict[str, Any]) -> tuple[str | None, str | None]:
    try:
        spec = record["execution"]["push"]
        review = record["independent_review"]
        valid = (
            record["scope"]["organization"] == "OSHEThai"
            and record["scope"]["repository"] == "oshe-platform"
            and spec["remote_url"] == REMOTE
            and spec["source_branch"] in BRANCHES
            and spec["destination_branch"] in BRANCHES
            and OID.fullmatch(spec["source_commit"]) is not None
            and (spec["expected_remote_oid"] is None
                 or OID.fullmatch(spec["expected_remote_oid"]) is not None)
            and spec["expected_remote_oid"] != spec["source_commit"]
            and record["scope"]["target"] == "refs/heads/" + spec["destination_branch"]
            and record["scope"]["exact_commit_or_configuration_identity"] == spec["source_commit"]
            and record["scope"]["expected_post_state"] == spec["source_commit"]
            and review["disposition"] == "PASS"
            and review["reviewer_assignment_id"] not in {None, record["assignment_id"]}
            and review["reviewer_role_id"] == "security-privacy-product-safety-agent"
            and review["review_ref"] == spec["review_evidence"]["path"]
            and record["execution"]["command"] == command_for(spec)
        )
        return ("branch-push", "BRANCH_PR") if valid else (None, None)
    except (KeyError, TypeError, ValueError):
        return None, None


def _git(spec: dict[str, Any], args: list[str], env: dict[str, str],
         *, allow_failure: bool = False) -> subprocess.CompletedProcess[bytes]:
    prefix = command_for(spec)
    # Reuse only the fixed prefix; never allow a gate to supply diagnostic args.
    prefix = prefix[:prefix.index("push")]
    result = subprocess.run(prefix + args, env=env, capture_output=True,
                            check=False, timeout=120)
    if not allow_failure:
        _require(result.returncode == 0, "Git diagnostic failed")
    return result


def _config(spec: dict[str, Any], env: dict[str, str]) -> str:
    raw = _git(spec, ["config", "--null", "--list", "--includes"], env).stdout
    origin = []
    for entry in raw.split(b"\0"):
        if not entry:
            continue
        key_bytes, _, value_bytes = entry.partition(b"\n")
        key = key_bytes.decode("utf-8", errors="strict").lower()
        value = value_bytes.decode("utf-8", errors="strict")
        bad = (
            key.startswith(("url.", "include.", "includeif.", "protocol."))
            or key in {"core.hookspath", "core.sshcommand", "core.gitproxy",
                       "core.worktree", "core.attributesfile", "core.askpass", "extensions.worktreeconfig"}
            or key.startswith("remote.") and key.endswith((".pushurl", ".push", ".mirror", ".vcs", ".receivepack", ".uploadpack"))
            or key.startswith("push.") and not (key == "push.gpgsign" and value == "false")
            or key.startswith("filter.") and not key.startswith("filter.lfs.")
            or key.startswith("http.") and key not in {"http.sslbackend", "http.sslverify", "http.followredirects"}
            or key == "http.sslverify" and value.lower() not in {"true", "1", "yes"}
            or key == "http.followredirects" and value != "false"
            or key == "core.fsmonitor" and value.lower() not in {"false", "0", "no"}
            or key.startswith("credential.") and not (
                key == "credential.helper" and value in {"manager", "manager-core"}
            )
        )
        _require(not bad, "unsupported Git configuration")
        if key == "remote.origin.url":
            origin.append(value)
    _require(origin == [REMOTE], "origin URL must be exact and unambiguous")
    return "sha256:" + hashlib.sha256(raw).hexdigest()


def _evidence(spec: dict[str, Any], record: dict[str, Any]) -> None:
    for field in ("review_evidence", "test_evidence"):
        binding = spec[field]
        path = Path(binding["path"])
        _require(path.is_absolute() and path.resolve() == path, "noncanonical evidence path")
        raw = path.read_bytes()
        _require(hashlib.sha256(raw).hexdigest() == binding["sha256"], "evidence hash changed")
        data = json.loads(raw)
        _require(data.get("disposition") == "PASS"
                 and data.get("target_commit") == spec["source_commit"], "evidence not PASS for exact commit")
        if field == "review_evidence":
            _require(data.get("reviewer_assignment_id") == record["independent_review"]["reviewer_assignment_id"], "review attribution mismatch")
            _require(data.get("producer_assignment_id") not in {None, data.get("reviewer_assignment_id")}, "self review")
        else:
            _require(data.get("failed") == 0 and type(data.get("failed")) is int
                     and type(data.get("passed")) is int and data["passed"] > 0
                     and data.get("skipped") == 0 and type(data.get("skipped")) is int,
                     "test evidence missing passing non-skipped checks")


def remote_oid(spec: dict[str, Any], env: dict[str, str]) -> str | None:
    ref = "refs/heads/" + spec["destination_branch"]
    out = _git(spec, ["ls-remote", "--refs", "--", spec["remote_url"], ref], env).stdout.decode()
    lines = out.splitlines()
    if not lines:
        return None
    _require(len(lines) == 1, "ambiguous remote response")
    values = lines[0].split("\t")
    _require(len(values) == 2 and values[1] == ref and OID.fullmatch(values[0]) is not None,
             "invalid remote response")
    return values[0]


def snapshot(record: dict[str, Any]) -> dict[str, Any]:
    """Read-only preflight, also usable to construct the expected state digest."""
    _require(classify_push(record) == ("branch-push", "BRANCH_PR"), "invalid push grammar")
    spec = record["execution"]["push"]
    env = safe_environment()
    worktree = Path(spec["worktree"])
    binary = Path(spec["git_executable"])
    found = shutil.which("git")
    _require(worktree.is_absolute() and worktree.resolve() == worktree, "noncanonical worktree")
    _require(binary.is_absolute() and binary.resolve() == binary and found is not None
             and Path(found).resolve() == binary, "untrusted Git executable path")
    _require(hashlib.sha256(binary.read_bytes()).hexdigest() == spec["git_executable_sha256"], "Git executable changed")
    config_digest = _config(spec, env)
    root = _git(spec, ["rev-parse", "--show-toplevel"], env).stdout.decode().strip()
    _require(Path(root).resolve() == worktree, "wrong repository root")
    branch = _git(spec, ["symbolic-ref", "--quiet", "HEAD"], env).stdout.decode().strip()
    _require(branch == "refs/heads/" + spec["source_branch"], "wrong or detached branch")
    head = _git(spec, ["rev-parse", "--verify", "HEAD^{commit}"], env).stdout.decode().strip()
    _require(head == spec["source_commit"], "source HEAD changed")
    common = Path(_git(spec, ["rev-parse", "--path-format=absolute", "--git-common-dir"], env).stdout.decode().strip()).resolve()
    for relative in ("objects/info/alternates", "info/grafts", "hooks/pre-push"):
        _require(not (common / relative).exists(), "unsafe object/hook indirection")
    _require(not _git(spec, ["for-each-ref", "--format=%(refname)", "refs/replace"], env).stdout.strip(), "replace refs are unsupported")
    _require(not _git(spec, ["status", "--porcelain=v1", "--untracked-files=all"], env).stdout.strip(), "dirty worktree")
    _evidence(spec, record)
    oid = remote_oid(spec, env)
    _require(oid == spec["expected_remote_oid"], "remote prestate changed")
    if oid is not None:
        _require(_git(spec, ["merge-base", "--is-ancestor", oid, head], env,
                      allow_failure=True).returncode == 0, "not a proven fast-forward")
    return {
        "worktree": str(worktree), "common_dir": str(common), "source_branch": spec["source_branch"],
        "source_commit": head, "remote_url": spec["remote_url"],
        "destination_branch": spec["destination_branch"], "remote_oid": oid,
        "config_digest": config_digest, "git_executable_sha256": spec["git_executable_sha256"],
        "review_evidence": spec["review_evidence"], "test_evidence": spec["test_evidence"],
    }


def execute_push(record: dict[str, Any], *, dry_run: bool = False) -> int:
    from evaluate_github_operation import evaluate
    _require(not evaluate(record, datetime.now().astimezone()), "operation gate denied")
    state = snapshot(record)
    _require(digest(state) == record["scope"]["expected_pre_state_digest"], "prestate digest mismatch")
    if dry_run:
        print("BOUNDED_GIT_PUSH_DRY_RUN_PASS")
        return 0
    gate_id = record["gate_id"]
    _require(re.fullmatch(r"GHG-[A-Z0-9-]+", gate_id) is not None, "invalid gate identity")
    directory = Path(state["common_dir"]) / "audit-push-gates"
    directory.mkdir(exist_ok=True)
    _require(directory.resolve() == directory, "receipt directory indirection")
    # O_EXCL: an attempted gate is consumed even if validation/transport later fails.
    marker = directory / (gate_id + ".used")
    with marker.open("xb") as stream:
        stream.write(digest(record).encode())
    _require(snapshot(record) == state, "preflight state changed")
    _require(not evaluate(record, datetime.now().astimezone()), "operation gate expired or changed")
    spec = record["execution"]["push"]
    env = safe_environment()
    completed = run_push(spec, env)
    print(f"BOUNDED_GIT_PUSH_RECEIPT gate_id={gate_id} exit_code={completed.returncode}")
    print(f"stdout_sha256={hashlib.sha256(completed.stdout).hexdigest()}")
    print(f"stderr_sha256={hashlib.sha256(completed.stderr).hexdigest()}")
    if completed.returncode != 0:
        return completed.returncode
    _require(remote_oid(spec, env) == spec["source_commit"], "remote readback mismatch")
    print("BOUNDED_GIT_PUSH_READBACK_PASS")
    return 0


def run_push(spec: dict[str, Any], env: dict[str, str]) -> subprocess.CompletedProcess[bytes]:
    """Single transport seam for hermetic tests; no user-selectable transport."""
    return subprocess.run(command_for(spec), env=env, capture_output=True,
                          check=False, timeout=120)
