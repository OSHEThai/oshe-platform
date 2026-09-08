#!/usr/bin/env python3
"""Bounded local verifier for audit provenance tuples declared in RFC.

Enforces:
1. Machine-readable parseability of declared (commit, path, blob, normalized SHA-256) tuples.
2. Resolution of commits, paths, and blobs via argument-vector Git calls against local history.
3. Exact equality between resolved blob and declared blob.
4. Exact equality between normalized blob content SHA-256 and declared hash.
5. Fail-closed nonzero exit if any reference, commit history, or digest mismatches.
"""

from __future__ import annotations

import argparse
import hashlib
import pathlib
import re
import subprocess
import sys
from dataclasses import dataclass

ROOT = pathlib.Path(__file__).resolve().parents[1]
DEFAULT_RFC_PATH = ROOT / "docs" / "rfc" / "audit-20260908-baseline-contract-reconciliation.md"


@dataclass(frozen=True)
class ProvenanceTuple:
    target: str
    commit: str
    path: str
    blob: str
    normalized_sha256: str


def parse_rfc_tuples(content: str) -> list[ProvenanceTuple]:
    """Parses declared provenance tuples from RFC markdown content."""
    tuples: list[ProvenanceTuple] = []

    # 1. Parse markdown table rows: | target | commit | path | blob | normalized_sha256 |
    table_pattern = re.compile(
        r"^\s*\|\s*(\S+)\s*\|\s*([0-9a-fA-F]{40})\s*\|\s*(\S+)\s*\|\s*([0-9a-fA-F]{40})\s*\|\s*([0-9a-fA-F]{64})\s*\|\s*$",
        re.MULTILINE,
    )
    for m in table_pattern.finditer(content):
        target, commit, path, blob, norm_hash = m.groups()
        if target.lower() in ("target", "---", ":---", "---:"):
            continue
        tuples.append(
            ProvenanceTuple(
                target=target.strip(),
                commit=commit.strip().lower(),
                path=path.strip(),
                blob=blob.strip().lower(),
                normalized_sha256=norm_hash.strip().lower(),
            )
        )

    if not tuples:
        # 2. Fallback to bullet list declaration if table is not present
        bullet_pattern = re.compile(
            r"commit\s+([0-9a-fA-F]{40}).*?path\s+(\S+).*?blob\s+([0-9a-fA-F]{40}).*?normalized(?:\s+SHA-256)?(?:\s+is)?\s+([0-9a-fA-F]{64})",
            re.IGNORECASE | re.DOTALL,
        )
        for m in bullet_pattern.finditer(content):
            commit, path, blob, norm_hash = m.groups()
            clean_path = path.strip().rstrip(",")
            target = "compose" if "compose" in clean_path else "seed"
            tuples.append(
                ProvenanceTuple(
                    target=target,
                    commit=commit.strip().lower(),
                    path=clean_path,
                    blob=blob.strip().lower(),
                    normalized_sha256=norm_hash.strip().lower(),
                )
            )

    if not tuples:
        raise ValueError("No valid provenance tuples found in RFC content")

    return tuples


def verify_tuple(
    t: ProvenanceTuple,
    repo_root: pathlib.Path,
    git_runner=None,
) -> list[str]:
    """Verifies a single provenance tuple against git history.

    Returns a list of error strings; empty list indicates full compliance.
    """
    errors: list[str] = []
    if git_runner is None:
        def default_runner(cmd: list[str]) -> subprocess.CompletedProcess:
            return subprocess.run(cmd, cwd=str(repo_root), capture_output=True)
        git_runner = default_runner

    # 1. Verify commit exists in local git history
    res = git_runner(["git", "rev-parse", "--verify", f"{t.commit}^{{commit}}"])
    if res.returncode != 0:
        errors.append(f"[{t.target}] commit {t.commit} missing from local git history")
        return errors

    # 2. Resolve blob from commit:path
    res = git_runner(["git", "rev-parse", "--verify", f"{t.commit}:{t.path}"])
    if res.returncode != 0:
        errors.append(f"[{t.target}] failed to resolve path {t.path} at commit {t.commit}")
        return errors
    resolved_blob = res.stdout.strip().decode("ascii", errors="replace").lower()
    if resolved_blob != t.blob.lower():
        errors.append(f"[{t.target}] blob mismatch for {t.path}: expected {t.blob}, got {resolved_blob}")
        return errors

    # 3. Verify blob object type
    res = git_runner(["git", "cat-file", "-t", t.blob])
    if res.returncode != 0 or res.stdout.strip().decode("ascii", errors="replace") != "blob":
        errors.append(f"[{t.target}] object {t.blob} is not a valid blob")
        return errors

    # 4. Verify normalized content SHA-256
    res = git_runner(["git", "cat-file", "-p", t.blob])
    if res.returncode != 0:
        errors.append(f"[{t.target}] failed to read content for blob {t.blob}")
        return errors
    content = res.stdout
    normalized = content.replace(b"\r\n", b"\n")
    digest = hashlib.sha256(normalized).hexdigest().lower()
    if digest != t.normalized_sha256.lower():
        errors.append(
            f"[{t.target}] normalized SHA-256 mismatch for {t.path}: expected {t.normalized_sha256}, got {digest}"
        )

    return errors


def verify_audit_provenance(
    rfc_path: pathlib.Path | None = None,
    repo_root: pathlib.Path | None = None,
    git_runner=None,
) -> tuple[bool, list[str]]:
    """Verifies all provenance tuples in the specified RFC."""
    rfc_path = rfc_path or DEFAULT_RFC_PATH
    repo_root = repo_root or ROOT

    if not rfc_path.is_file():
        return False, [f"RFC file not found: {rfc_path}"]

    try:
        content = rfc_path.read_text(encoding="utf-8")
        tuples = parse_rfc_tuples(content)
    except Exception as exc:
        return False, [f"Failed to parse RFC provenance tuples: {exc}"]

    all_errors: list[str] = []
    for t in tuples:
        errs = verify_tuple(t, repo_root=repo_root, git_runner=git_runner)
        all_errors.extend(errs)

    return len(all_errors) == 0, all_errors


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="Verify audit provenance tuples declared in RFC")
    parser.add_argument("--rfc", type=pathlib.Path, default=DEFAULT_RFC_PATH, help="Path to RFC markdown file")
    parser.add_argument("--root", type=pathlib.Path, default=ROOT, help="Path to repository root")
    args = parser.parse_args(argv)

    success, errors = verify_audit_provenance(rfc_path=args.rfc, repo_root=args.root)
    if success:
        print(f"[PROVENANCE-PASS] All declared tuples in {args.rfc.name} verified against Git history.")
        return 0

    print(f"[PROVENANCE-FAIL] Provenance verification failed with {len(errors)} error(s):", file=sys.stderr)
    for err in errors:
        print(f"  - {err}", file=sys.stderr)
    return 1


if __name__ == "__main__":
    sys.exit(main())
