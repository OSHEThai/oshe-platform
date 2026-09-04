#!/usr/bin/env python3
"""Generate and verify deterministic, offline release-integrity metadata.

The output is deliberately unsigned.  Production signing identities, hosted
attestation services, credentials, and publication are outside this tool's
authority.
"""
from __future__ import annotations

import argparse
import hashlib
import json
import pathlib
import subprocess
import tempfile
from typing import Any


EXCLUDED_PARTS = {".git", ".local-ci", "__pycache__", "artifacts", "build", "dist", "node_modules"}
INPUT_PATHS = (
    ".ai/requirements-validation.txt",
    "tools/dev/go.mod",
    "toolchain.lock.yaml",
    "repo-manifest.yaml",
    "LICENSE-POLICY.md",
    "modules/module-registry.yaml",
    "contracts/api/go.mod",
    "packages/identifiers/go.mod",
    "schemas/api/error-envelope.schema.json",
    "modules/organization-tenancy/go.mod",
    "modules/identity-authorization/go.mod",
    "modules/files-evidence/go.mod",
    "modules/records-audit/go.mod",
    "modules/configuration-checklist/go.mod",
    "modules/workflow-action/go.mod",
    "modules/events-outbox-jobs/go.mod",
    "modules/reporting-localization/go.mod",
    "modules/contract-migration-governance/go.mod",
)
ARTIFACT_NAMES = ("sbom.spdx.json", "provenance.json", "platform-bom.json", "signature-envelope.json")


def canonical_bytes(value: Any) -> bytes:
    return (json.dumps(value, indent=2, sort_keys=True, ensure_ascii=True) + "\n").encode("utf-8")


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


def git_commit(root: pathlib.Path) -> str:
    completed = subprocess.run(
        ["git", "rev-parse", "HEAD"], cwd=root, check=False, capture_output=True, text=True
    )
    return completed.stdout.strip() if completed.returncode == 0 and completed.stdout.strip() else "UNAVAILABLE"


def component_records(root: pathlib.Path) -> list[dict[str, str]]:
    records: list[dict[str, str]] = []
    for relative in INPUT_PATHS:
        path = root / relative
        if not path.is_file():
            raise ValueError(f"required supply-chain input is missing: {relative}")
        records.append({"name": relative, "sha256": sha256_bytes(path.read_bytes())})
    return records


def release_name(root: pathlib.Path) -> str:
    for line in (root / "repo-manifest.yaml").read_text(encoding="utf-8").splitlines():
        if line.startswith("  release:"):
            return line.split(":", 1)[1].strip()
    return "UNDECLARED"


def bundle(root: pathlib.Path) -> dict[str, bytes]:
    components = component_records(root)
    digest = repository_digest(root)
    commit = git_commit(root)
    sbom = {
        "SPDXID": "SPDXRef-DOCUMENT",
        "dataLicense": "CC0-1.0",
        "documentNamespace": f"urn:oshethai:sbom:sha256:{digest}",
        "name": "OSHE Platform offline SBOM",
        "packages": [
            {"SPDXID": f"SPDXRef-{index:02d}", "checksums": [{"algorithm": "SHA256", "checksumValue": item["sha256"]}], "name": item["name"]}
            for index, item in enumerate(components, start=1)
        ],
        "spdxVersion": "SPDX-2.3",
    }
    provenance = {
        "_type": "https://in-toto.io/Statement/v1",
        "predicate": {
            "buildDefinition": {"buildType": "https://oshethai.example/build/offline-supply-chain/v1", "externalParameters": {"network": "disabled"}, "resolvedDependencies": components},
            "runDetails": {"builder": {"id": "oshethai/local-offline"}, "metadata": {"invocationId": "deterministic-local"}},
        },
        "predicateType": "https://slsa.dev/provenance/v1",
        "subject": [{"digest": {"sha256": digest}, "name": f"git:{commit}"}],
    }
    platform_bom = {
        "components": components,
        "format": "OSHE-PLATFORM-BOM-1",
        "release": release_name(root),
        "repository_digest": {"sha256": digest},
        "source_commit": commit,
    }
    envelope = {
        "algorithm": "NONE",
        "kind": "UNSIGNED_TEST_ONLY",
        "payload_sha256": sha256_bytes(canonical_bytes(provenance)),
        "production_ready": False,
        "signature": None,
    }
    artifacts = {
        "sbom.spdx.json": canonical_bytes(sbom),
        "provenance.json": canonical_bytes(provenance),
        "platform-bom.json": canonical_bytes(platform_bom),
        "signature-envelope.json": canonical_bytes(envelope),
    }
    manifest = "".join(f"{sha256_bytes(artifacts[name])}  {name}\n" for name in ARTIFACT_NAMES).encode("utf-8")
    artifacts["SHA256SUMS"] = manifest
    return artifacts


def write_bundle(root: pathlib.Path, output: pathlib.Path) -> None:
    if any((output / name).exists() for name in (*ARTIFACT_NAMES, "SHA256SUMS")):
        raise FileExistsError("refusing to replace an existing supply-chain bundle")
    output.mkdir(parents=True, exist_ok=True)
    for name, content in bundle(root).items():
        (output / name).write_bytes(content)


def verify_bundle(root: pathlib.Path, output: pathlib.Path) -> bool:
    expected = bundle(root)
    return all((output / name).is_file() and (output / name).read_bytes() == content for name, content in expected.items())


def main() -> int:
    parser = argparse.ArgumentParser(description="Generate offline, deterministic supply-chain metadata.")
    parser.add_argument("--emit", action="store_true", help="write a bundle to --output-dir")
    parser.add_argument("--verify", action="store_true", help="generate and verify an ephemeral bundle without repository writes")
    parser.add_argument("--output-dir", default=".ci/artifacts/supply-chain")
    args = parser.parse_args()
    if args.emit == args.verify:
        parser.error("choose exactly one of --emit or --verify")
    root = pathlib.Path.cwd().resolve()
    if args.emit:
        output = (root / args.output_dir).resolve()
        if root not in output.parents:
            parser.error("output directory must remain inside the repository")
        try:
            write_bundle(root, output)
        except FileExistsError as exc:
            print(f"ERROR: {exc}")
            return 1
        print(f"Supply-chain bundle written to {output}")
        return 0
    with tempfile.TemporaryDirectory() as temporary:
        output = pathlib.Path(temporary)
        write_bundle(root, output)
        if not verify_bundle(root, output):
            print("ERROR: supply-chain bundle verification failed")
            return 1
    print("Supply-chain verification passed")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
