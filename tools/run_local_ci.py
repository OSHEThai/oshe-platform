#!/usr/bin/env python3
from __future__ import annotations

import argparse
import hashlib
import json
import pathlib
import subprocess
import sys
from datetime import datetime, timezone
from typing import Any


EXCLUDED_PARTS = {
    ".git",
    ".local-ci",
    ".worktrees",
    "__pycache__",
    ".pytest_cache",
    ".mypy_cache",
    ".ruff_cache",
    "artifacts",
    "bin",
    "build",
    "coverage",
    "dist",
    "node_modules",
    "obj",
    "TestResults",
}


def sha256_bytes(value: bytes) -> str:
    return hashlib.sha256(value).hexdigest()


def repository_digest(root: pathlib.Path) -> str:
    digest = hashlib.sha256()
    for path in sorted(root.rglob("*")):
        if not path.is_file() or any(part in EXCLUDED_PARTS for part in path.relative_to(root).parts):
            continue
        relative = path.relative_to(root).as_posix()
        digest.update(relative.encode("utf-8"))
        digest.update(b"\0")
        digest.update(path.read_bytes())
        digest.update(b"\0")
    return digest.hexdigest()


def git_value(root: pathlib.Path, *args: str) -> str:
    completed = subprocess.run(
        ["git", *args],
        cwd=root,
        check=False,
        capture_output=True,
        text=True,
    )
    if completed.returncode != 0:
        return "UNAVAILABLE"
    return completed.stdout.strip() or "UNAVAILABLE"


def toolchain_identity() -> str:
    return f"{sys.implementation.name}:{sys.version.split()[0]}:{sys.executable}"


def load_json(path: pathlib.Path) -> dict[str, Any]:
    return json.loads(path.read_text(encoding="utf-8"))


def write_checkpoints(path: pathlib.Path, payload: dict[str, Any]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    temporary = path.with_suffix(".tmp")
    temporary.write_text(json.dumps(payload, indent=2, sort_keys=True) + "\n", encoding="utf-8")
    temporary.replace(path)


def main() -> int:
    parser = argparse.ArgumentParser(description="Run local-first CI without fail-fast behavior.")
    parser.add_argument("--mode", choices=("incremental", "full"), default="incremental")
    parser.add_argument("--milestone-close")
    parser.add_argument("--no-checkpoint", action="store_true")
    parser.add_argument("--config", default=".ci/local-ci.json")
    args = parser.parse_args()

    if args.mode == "full" and not args.milestone_close:
        parser.error("Full CI is permitted only with --milestone-close <milestone>.")
    if args.mode == "incremental" and args.milestone_close:
        parser.error("--milestone-close is valid only for Full CI.")

    root = pathlib.Path.cwd().resolve()
    config_path = (root / args.config).resolve()
    if root not in config_path.parents:
        parser.error("CI configuration must remain inside the repository.")
    config = load_json(config_path)
    checks = config.get("checks")
    if not isinstance(checks, list) or not checks:
        parser.error("CI configuration must contain at least one check.")

    state_path = root / ".local-ci" / "checkpoints.json"
    checkpoints: dict[str, Any] = {"schema_version": "1.0.0", "checks": {}}
    if state_path.is_file():
        checkpoints = load_json(state_path)
    checkpoint_checks = checkpoints.setdefault("checks", {})

    repo_digest = repository_digest(root)
    base_commit = git_value(root, "merge-base", "HEAD", "origin/main")
    toolchain = toolchain_identity()
    failures: list[str] = []
    passed: list[str] = []
    skipped: list[str] = []

    print(f"CI mode: {args.mode}")
    if args.milestone_close:
        print(f"Milestone closure: {args.milestone_close}")
    print(f"Repository input digest: {repo_digest}")
    print(f"Base commit: {base_commit}")

    for check in checks:
        check_id = check.get("id") if isinstance(check, dict) else None
        command = check.get("command") if isinstance(check, dict) else None
        if not isinstance(check_id, str) or not check_id or not isinstance(command, list) or not command:
            failures.append(str(check_id or "INVALID_CHECK"))
            print("FAIL INVALID_CHECK: id and non-empty command are required", file=sys.stderr)
            continue
        if not all(isinstance(item, str) and item for item in command):
            failures.append(check_id)
            print(f"FAIL {check_id}: command entries must be non-empty strings", file=sys.stderr)
            continue

        resolved_command = [sys.executable if command[0] == "python" else command[0], *command[1:]]
        command_digest = sha256_bytes(json.dumps(command, separators=(",", ":")).encode("utf-8"))
        evidence_key = {
            "command_digest": command_digest,
            "toolchain_identity": toolchain,
            "repository_input_digest": repo_digest,
            "base_commit": base_commit,
        }
        previous = checkpoint_checks.get(check_id)
        may_skip = args.mode == "incremental" and not args.no_checkpoint and previous is not None
        if may_skip and all(previous.get(key) == value for key, value in evidence_key.items()):
            skipped.append(check_id)
            print(f"SKIP {check_id}: unchanged passing checkpoint")
            continue

        print(f"RUN  {check_id}: {' '.join(command)}")
        completed = subprocess.run(resolved_command, cwd=root, check=False, text=True)
        if completed.returncode == 0:
            passed.append(check_id)
            checkpoint_checks[check_id] = {
                **evidence_key,
                "passed_at": datetime.now(timezone.utc).isoformat(),
            }
            print(f"PASS {check_id}")
        else:
            failures.append(check_id)
            checkpoint_checks.pop(check_id, None)
            print(f"FAIL {check_id}: exit {completed.returncode}", file=sys.stderr)

    known_ids = {check.get("id") for check in checks if isinstance(check, dict)}
    for stale_id in set(checkpoint_checks) - known_ids:
        checkpoint_checks.pop(stale_id, None)
    write_checkpoints(state_path, checkpoints)

    print(f"CI summary: passed={len(passed)} skipped={len(skipped)} failed={len(failures)}")
    if failures:
        print("Failed checks: " + ", ".join(failures), file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
