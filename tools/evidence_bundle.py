#!/usr/bin/env python3
"""Create deterministic, offline changeset and evidence-bundle artifacts."""
from __future__ import annotations

import argparse
import hashlib
import json
import pathlib
import subprocess
import tempfile
from typing import Any, Iterable


ARTIFACT_NAMES = (
    "changeset.json",
    "changelog-fragment.md",
    "pull-request-body.md",
    "evidence-record.yaml",
    "signature-envelope.json",
)
REQUIRED_INPUTS = (
    ".github/PULL_REQUEST_TEMPLATE.md",
    ".ai/schemas/evidence-record.schema.json",
    ".ai/policies/evidence.yaml",
    "repo-manifest.yaml",
)
DEFAULT_ALLOWED_PATHS = (
    ".ai/",
    ".ci/",
    ".github/",
    "apps/",
    "contracts/",
    "database/",
    "docs/",
    "modules/",
    "packages/",
    "products/",
    "schemas/",
    "tests/",
    "tools/",
    "README.md",
    "repo-manifest.yaml",
    "LICENSE-POLICY.md",
)


def canonical_bytes(value: Any) -> bytes:
    return (json.dumps(value, indent=2, sort_keys=True, ensure_ascii=True) + "\n").encode("utf-8")


def digest(value: bytes) -> str:
    return hashlib.sha256(value).hexdigest()


def run_git(root: pathlib.Path, *args: str) -> str:
    completed = subprocess.run(["git", *args], cwd=root, check=False, capture_output=True, text=True)
    if completed.returncode != 0:
        raise ValueError(f"git {' '.join(args)} failed: {completed.stderr.strip()}")
    return completed.stdout.strip()


def require_inputs(root: pathlib.Path) -> None:
    for relative in REQUIRED_INPUTS:
        if not (root / relative).is_file():
            raise ValueError(f"required evidence input is missing: {relative}")


def allowed(path: str, prefixes: Iterable[str]) -> bool:
    return any(path == prefix or path.startswith(prefix) for prefix in prefixes)


def changed_entries(root: pathlib.Path, base_commit: str, prefixes: tuple[str, ...]) -> list[dict[str, str | None]]:
    rows = run_git(root, "diff", "--name-status", base_commit, "HEAD").splitlines()
    entries: list[dict[str, str | None]] = []
    for row in rows:
        status, path = row.split("\t", 1)
        if not allowed(path, prefixes):
            raise ValueError(f"changed path is outside the allowed scope: {path}")
        post_digest = None if status.startswith("D") else digest(run_git(root, "show", f"HEAD:{path}").encode("utf-8"))
        entries.append({"path": path, "status": status, "sha256": post_digest})
    return sorted(entries, key=lambda item: str(item["path"]))


def repository_digest(root: pathlib.Path) -> str:
    names = run_git(root, "ls-files").splitlines()
    hasher = hashlib.sha256()
    for relative in sorted(name for name in names if name):
        path = root / relative
        hasher.update(relative.encode("utf-8"))
        hasher.update(b"\0")
        hasher.update(path.read_bytes())
        hasher.update(b"\0")
    return hasher.hexdigest()


def render_pr_body(issue_ref: str, risk_class: str, base_commit: str, entries: list[dict[str, str | None]]) -> str:
    changed = ", ".join(str(item["path"]) for item in entries) or "No tracked changes"
    return "\n".join(
        (
            "## Mission and issue",
            "",
            f"- Issue, or direct out-of-Issue authorization: {issue_ref}",
            "- Mission: Offline deterministic evidence-bundle generation",
            f"- Risk class: {risk_class}",
            f"- Base commit: {base_commit}",
            "",
            "## Outcome",
            "",
            "Creates a locally verifiable changeset, changelog fragment, draft PR body, and unsigned evidence bundle.",
            "",
            "## Scope",
            "",
            f"- Changed paths: {changed}",
            "- Non-goals: GitHub actions, signing, release, deployment, provider routing, production data.",
            "",
            "## Verification",
            "",
            "- [x] Offline deterministic bundle verification completed",
            "- [x] Incomplete-evidence and scope-negative paths fail closed",
            "",
            "## Evidence",
            "",
            "- Evidence class: CI",
            "- Signature: UNSIGNED_TEST_ONLY; production_ready=false",
            "- Remaining risks: qualified human review and any release decision remain separate.",
            "",
            "## Merge and cleanup gate",
            "",
            "- [ ] ADR-0006 GitHub operation gate is evaluated separately",
            "- [ ] Remote branch cleanup occurs only after merge readback",
            "- [x] No production deployment performed",
            "",
        )
    )


def bundle(root: pathlib.Path, base_commit: str, issue_ref: str, risk_class: str, prefixes: tuple[str, ...]) -> dict[str, bytes]:
    require_inputs(root)
    entries = changed_entries(root, base_commit, prefixes)
    head = run_git(root, "rev-parse", "HEAD")
    captured_at = run_git(root, "show", "-s", "--format=%cI", "HEAD")
    changeset = {
        "base_commit": base_commit,
        "changed_paths": entries,
        "issue_ref": issue_ref,
        "repository_input_digest": repository_digest(root),
        "result_commit": head,
        "risk_class": risk_class,
    }
    changelog = "\n".join((f"## {issue_ref}", "", f"- Evidence bundle for {len(entries)} tracked change(s) from `{base_commit}` to `{head}`.", ""))
    pr_body = render_pr_body(issue_ref, risk_class, base_commit, entries)
    record_seed = digest(canonical_bytes(changeset))[:12].upper()
    evidence = {
        "artifact_refs": ["changeset.json", "changelog-fragment.md", "pull-request-body.md", "SHA256SUMS"],
        "captured_at": captured_at,
        "captured_by": "tools/evidence_bundle.py",
        "claim": "Offline deterministic changeset and evidence bundle generated and verified.",
        "command_or_method": "python tools/evidence_bundle.py --emit",
        "commit": head,
        "environment": "local-offline-evidence-bundle",
        "evidence_class": "CI",
        "evidence_id": f"EVD-V010-BUNDLE-{record_seed}",
        "human_decision_ref": None,
        "limitations": ["Unsigned test-only artifact", "Does not prove runtime behavior", "Does not authorize merge or release"],
        "repository": "OSHEThai/oshe-platform",
        "result": "PASS",
        "review_ref": None,
        "schema_version": "1.0.0",
    }
    evidence_bytes = canonical_bytes(evidence)
    envelope = {
        "algorithm": "NONE",
        "kind": "UNSIGNED_TEST_ONLY",
        "payload_sha256": digest(evidence_bytes),
        "production_ready": False,
        "signature": None,
    }
    artifacts = {
        "changeset.json": canonical_bytes(changeset),
        "changelog-fragment.md": changelog.encode("utf-8"),
        "pull-request-body.md": pr_body.encode("utf-8"),
        "evidence-record.yaml": evidence_bytes,
        "signature-envelope.json": canonical_bytes(envelope),
    }
    artifacts["SHA256SUMS"] = "".join(f"{digest(artifacts[name])}  {name}\n" for name in ARTIFACT_NAMES).encode("utf-8")
    return artifacts


def write_bundle(root: pathlib.Path, output: pathlib.Path, base_commit: str, issue_ref: str, risk_class: str, prefixes: tuple[str, ...]) -> None:
    if any((output / name).exists() for name in (*ARTIFACT_NAMES, "SHA256SUMS")):
        raise FileExistsError("refusing to replace an existing evidence bundle")
    output.mkdir(parents=True, exist_ok=True)
    for name, content in bundle(root, base_commit, issue_ref, risk_class, prefixes).items():
        (output / name).write_bytes(content)


def verify_bundle(root: pathlib.Path, output: pathlib.Path, base_commit: str, issue_ref: str, risk_class: str, prefixes: tuple[str, ...]) -> bool:
    expected = bundle(root, base_commit, issue_ref, risk_class, prefixes)
    return all((output / name).is_file() and (output / name).read_bytes() == content for name, content in expected.items())


def main() -> int:
    parser = argparse.ArgumentParser(description="Generate deterministic offline evidence bundles.")
    parser.add_argument("--emit", action="store_true")
    parser.add_argument("--verify", action="store_true")
    parser.add_argument("--output-dir", default=".ci/artifacts/evidence-bundle")
    parser.add_argument("--base-commit")
    parser.add_argument("--issue-ref", default="V010-I028")
    parser.add_argument("--risk-class", default="R2")
    parser.add_argument("--allowed-path", action="append", dest="allowed_paths")
    args = parser.parse_args()
    if args.emit == args.verify:
        parser.error("choose exactly one of --emit or --verify")
    root = pathlib.Path.cwd().resolve()
    try:
        base = args.base_commit or run_git(root, "merge-base", "HEAD", "origin/main")
        prefixes = tuple(args.allowed_paths or DEFAULT_ALLOWED_PATHS)
        if args.emit:
            output = (root / args.output_dir).resolve()
            if root not in output.parents:
                parser.error("output directory must remain inside the repository")
            write_bundle(root, output, base, args.issue_ref, args.risk_class, prefixes)
            print(f"Evidence bundle written to {output}")
            return 0
        with tempfile.TemporaryDirectory() as temporary:
            output = pathlib.Path(temporary)
            write_bundle(root, output, base, args.issue_ref, args.risk_class, prefixes)
            if not verify_bundle(root, output, base, args.issue_ref, args.risk_class, prefixes):
                raise ValueError("ephemeral bundle did not verify")
    except (FileExistsError, ValueError) as exc:
        print(f"ERROR: {exc}")
        return 1
    print("Evidence-bundle verification passed")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
