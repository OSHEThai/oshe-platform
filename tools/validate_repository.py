#!/usr/bin/env python3
from __future__ import annotations

import argparse
import dataclasses
import fnmatch
import json
import pathlib
import re
import subprocess
import sys
import time

PLACEHOLDERS = tuple(
    "__" + name + "__"
    for name in (
        "GITHUB_ORG",
        "PRIMARY_GITHUB_OWNER",
        "RECOVERY_GITHUB_OWNER",
        "SECURITY_CONTACT_EMAIL",
    )
)

COMMON_REQUIRED = [
    "AGENTS.md",
    "CLAUDE.md",
    "GEMINI.md",
    "QWEN.md",
    ".github/CODEOWNERS",
    ".github/PULL_REQUEST_TEMPLATE.md",
    ".github/ISSUE_TEMPLATE/config.yml",
    ".github/ISSUE_TEMPLATE/bug.yml",
    ".github/ISSUE_TEMPLATE/change.yml",
    ".github/workflows/foundation.yml",
    ".editorconfig",
    ".gitattributes",
    ".gitignore",
    "README.md",
    "SECURITY.md",
    "CONTRIBUTING.md",
    "repo-manifest.yaml",
    ".ci/local-ci.json",
    "tools/run_local_ci.py",
]

REPO_REQUIRED = {
    "platform": [
        "apps/README.md",
        "products/README.md",
        "modules/README.md",
        "modules/module-registry.yaml",
        "modules/organization-tenancy/README.md",
        "modules/identity-authorization/README.md",
        "modules/files-evidence/README.md",
        "modules/records-audit/README.md",
        "modules/configuration-checklist/README.md",
        "modules/workflow-action/README.md",
        "modules/events-outbox-jobs/README.md",
        "modules/reporting-localization/README.md",
        "modules/contract-migration-governance/README.md",
        "packages/README.md",
        "contracts/README.md",
        "schemas/README.md",
        "database/README.md",
        "tests/README.md",
        "tests/security/README.md",
        "tests/release/README.md",
        "deploy/README.md",
        "docs/adr/README.md",
        "docs/rfc/README.md",
        ".ai/README.md",
    ],
    "content": [
        "packs/README.md",
        "packs/common/README.md",
        "packs/capability/README.md",
        "packs/industry/README.md",
        "packs/jurisdiction/thailand/README.md",
        "packs/standards/README.md",
        "forms/README.md",
        "checklists/README.md",
        "signage/README.md",
        "translations/README.md",
        "schemas/pack.schema.json",
        "tests/README.md",
    ],
}

SECRET_PATTERNS = [
    re.compile(r"ghp_[A-Za-z0-9]{20,}"),
    re.compile(r"github_pat_[A-Za-z0-9_]{20,}"),
    re.compile(r"-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----"),
    re.compile(r"(?i)(password|secret|token)\s*[:=]\s*[\"'][^\"']{8,}[\"']"),
    re.compile(
        r"(?m)^[ \t]*(?:export[ \t]+)?(?:[A-Z][A-Z0-9_]*_)?"
        r"(?:PASSWORD|SECRET|TOKEN)(?:_(?:VALUE|KEY))?[ \t]*=[ \t]*"
        r"(?![\"']|\$\{|<)[^\s#]{8,}[ \t]*(?:#.*)?$"
    ),
]

RIGHTS_METADATA = "RIGHTS-METADATA.json"
RIGHTS_REQUIRED = ("LICENSE", "DCO-1.1.txt", "NOTICE.md", RIGHTS_METADATA)
RIGHTS_IGNORED_PARTS = {".git", ".local-ci", "__pycache__", ".pytest_cache"}
RIGHTS_LICENSES = {
    "platform": {
        "OSHE_AUTHORED_ENGINEERING": {"MPL-2.0"},
        "PUBLIC_CONTRACT": {"Apache-2.0"},
        "PUBLIC_SCHEMA": {"Apache-2.0"},
        "SDK": {"Apache-2.0"},
        "INTEGRATION_EXAMPLE": {"Apache-2.0"},
        "CONFORMANCE_KIT": {"Apache-2.0"},
    },
    "content": {
        "CODE_TOOL_SCHEMA_TEST_AUTOMATION": {"Apache-2.0"},
        "OSHE_AUTHORED_PRACTICAL_CONTENT": {"CC-BY-SA-4.0"},
        "OSHE_AUTHORED_METADATA_OR_MAPPING": {"CC-BY-4.0"},
    },
}


def validate_rights_metadata(root: pathlib.Path, repo_kind: str) -> list[str]:
    """Require every repository file to have deterministic, fail-closed rights metadata."""
    errors: list[str] = []
    for relative_path in RIGHTS_REQUIRED:
        if not (root / relative_path).is_file():
            errors.append(f"missing licensing control: {relative_path}")
    metadata_path = root / RIGHTS_METADATA
    if not metadata_path.is_file():
        return errors
    try:
        metadata = json.loads(metadata_path.read_text(encoding="utf-8"))
    except json.JSONDecodeError as exc:
        return errors + [f"invalid rights metadata: {exc}"]
    if metadata.get("schema_version") != "1.0.0":
        errors.append("rights metadata must declare schema_version 1.0.0")
    if metadata.get("licensor") != "OSHEThai":
        errors.append("rights metadata licensor must be OSHEThai")
    expected_root_license = {"platform": "MPL-2.0", "content": "Apache-2.0"}[repo_kind]
    if metadata.get("root_license") != expected_root_license:
        errors.append(f"rights metadata root license must be {expected_root_license}")
    rules = metadata.get("rules")
    if not isinstance(rules, list) or not rules:
        return errors + ["rights metadata must contain ordered rules"]
    for path in root.rglob("*"):
        if not path.is_file():
            continue
        relative = path.relative_to(root).as_posix()
        if any(part in RIGHTS_IGNORED_PARTS for part in path.relative_to(root).parts) or path.suffix == ".pyc":
            continue
        matched = next((rule for rule in rules if isinstance(rule, dict) and fnmatch.fnmatchcase(relative, rule.get("path", ""))), None)
        if matched is None:
            errors.append(f"missing rights metadata for {relative}")
            continue
        classification = matched.get("classification")
        license_id = matched.get("license")
        if classification == "THIRD_PARTY_STANDARD_TEXT":
            if not isinstance(matched.get("source"), str) or not matched["source"].strip():
                errors.append(f"third-party standard text lacks provenance: {relative}")
            continue
        allowed = RIGHTS_LICENSES[repo_kind].get(classification)
        if allowed is None or license_id not in allowed:
            errors.append(f"invalid rights classification or license for {relative}")
        if matched.get("copyright") != "OSHEThai":
            errors.append(f"OSHE-authored rights lack OSHEThai attribution: {relative}")
    for license_id in sorted({license_id for values in RIGHTS_LICENSES[repo_kind].values() for license_id in values}):
        if not (root / "LICENSES" / f"{license_id}.txt").is_file():
            errors.append(f"missing standard license text: {license_id}")
    if not (root / "LICENSE").read_bytes().strip():
        errors.append("root LICENSE is empty")
    return errors


SECRET_ALLOWED_VALUES: dict[str, tuple[str, ...]] = {
    "deploy/local/.env.example": ("oshe_dev_synthetic_only",),
}


def secret_scan_text(relative_path: str, text: str) -> str:
    allowed = SECRET_ALLOWED_VALUES.get(relative_path)
    if not allowed:
        return text
    for value in allowed:
        text = text.replace(value, "")
    return text



OUTER_YAML_PATHS = (
    ".github/ISSUE_TEMPLATE/architecture.yml",
    ".github/ISSUE_TEMPLATE/bug.yml",
    ".github/ISSUE_TEMPLATE/change.yml",
    ".github/ISSUE_TEMPLATE/config.yml",
    ".github/ISSUE_TEMPLATE/documentation.yml",
    ".github/workflows/foundation.yml",
    "modules/module-registry.yaml",
    "repo-manifest.yaml",
)


@dataclasses.dataclass(frozen=True)
class YamlResourceLimits:
    max_file_bytes: int = 65536
    max_composed_nodes: int = 4096
    max_depth: int = 16
    max_aggregate_scalar_characters: int = 65536
    max_anchors: int = 0
    max_aliases: int = 0
    max_expanded_visits: int = 4096
    max_process_count: int = 1
    wall_clock_seconds: int = 10
    memory_mib: int = 128


@dataclasses.dataclass
class YamlMetrics:
    file_bytes: int = 0
    composed_nodes: int = 0
    depth: int = 0
    aggregate_scalar_characters: int = 0
    anchors: int = 0
    aliases: int = 0
    expanded_visits: int = 0
    process_count: int = 1
    wall_clock_seconds: int = 0
    memory_mib: int = 0


class YamlLimitError(ValueError):
    """A selected Secure Full-44 limit was exceeded before YAML construction."""


def _require_nonnegative_metric(name: str, value: int) -> None:
    if isinstance(value, bool) or not isinstance(value, int) or value < 0:
        raise YamlLimitError(f"invalid {name}: {value!r}")


def _check_limit(name: str, value: int, maximum: int) -> None:
    _require_nonnegative_metric(name, value)
    if value > maximum:
        raise YamlLimitError(f"{name} exceeds selected limit {maximum}: {value}")


def enforce_yaml_limits(metrics: YamlMetrics, limits: YamlResourceLimits = YamlResourceLimits()) -> None:
    """Fail closed on every approved numeric limit without constructing YAML."""
    for name, value in dataclasses.asdict(limits).items():
        _require_nonnegative_metric(f"limit {name}", value)
        if value > (2**63 - 1):
            raise YamlLimitError(f"limit {name} exceeds signed 64-bit range")

    checks = (
        ("file bytes", metrics.file_bytes, limits.max_file_bytes),
        ("composed nodes", metrics.composed_nodes, limits.max_composed_nodes),
        ("depth", metrics.depth, limits.max_depth),
        (
            "aggregate scalar characters",
            metrics.aggregate_scalar_characters,
            limits.max_aggregate_scalar_characters,
        ),
        ("anchors", metrics.anchors, limits.max_anchors),
        ("aliases", metrics.aliases, limits.max_aliases),
        ("expanded visits", metrics.expanded_visits, limits.max_expanded_visits),
        ("parser processes", metrics.process_count, limits.max_process_count),
        ("parser wall-clock seconds", metrics.wall_clock_seconds, limits.wall_clock_seconds),
        ("parser memory MiB", metrics.memory_mib, limits.memory_mib),
    )
    for name, value, maximum in checks:
        _check_limit(name, value, maximum)


def _strip_yaml_comment_and_count_markers(line: str) -> tuple[str, int, int]:
    """Return comment-free text and unquoted anchor/alias token counts.

    This is intentionally a bounded preflight scanner, not a YAML parser. It
    rejects anchors and aliases before any downstream YAML construction.
    """
    quote: str | None = None
    escaped = False
    github_expression_depth = 0
    anchors = 0
    aliases = 0
    result: list[str] = []
    for index, char in enumerate(line):
        if quote == '"' and escaped:
            escaped = False
            result.append(char)
            continue
        if quote == '"' and char == "\\":
            escaped = True
            result.append(char)
            continue
        if char in ("'", '"'):
            if quote is None:
                quote = char
            elif quote == char:
                quote = None
            result.append(char)
            continue
        if quote is None and line.startswith("${{", index):
            github_expression_depth += 1
        elif quote is None and github_expression_depth and line.startswith("}}", index):
            github_expression_depth -= 1
        if (
            quote is None
            and github_expression_depth == 0
            and char == "#"
            and (index == 0 or line[index - 1].isspace())
        ):
            break
        if quote is None and github_expression_depth == 0 and char in ("&", "*"):
            next_char = line[index + 1] if index + 1 < len(line) else ""
            if next_char and not next_char.isspace():
                if char == "&":
                    anchors += 1
                else:
                    aliases += 1
        result.append(char)
    return "".join(result), anchors, aliases


def _scalar_segments(text: str) -> list[str]:
    """Extract bounded scalar-like segments for preconstruction accounting."""
    stripped = text.strip()
    if not stripped or stripped.startswith("#"):
        return []
    if stripped.startswith("- "):
        stripped = stripped[2:].lstrip()
    if ":" in stripped:
        key, value = stripped.split(":", 1)
        return [segment.strip() for segment in (key, value) if segment.strip()]
    return [stripped]


def analyze_yaml_preconstruction(text: str, limits: YamlResourceLimits = YamlResourceLimits()) -> YamlMetrics:
    """Account for an outer YAML document using bounded, parser-free scanning."""
    encoded_size = len(text.encode("utf-8"))
    metrics = YamlMetrics(
        file_bytes=encoded_size,
        process_count=1,
        memory_mib=(encoded_size + (1024 * 1024) - 1) // (1024 * 1024),
    )
    enforce_yaml_limits(metrics, limits)
    started = time.monotonic()

    for raw_line in text.splitlines():
        if time.monotonic() - started > limits.wall_clock_seconds:
            raise YamlLimitError("parser wall-clock seconds exceeds selected limit")
        logical_line, anchors, aliases = _strip_yaml_comment_and_count_markers(raw_line)
        stripped = logical_line.strip()
        if not stripped:
            continue
        leading_spaces = len(logical_line) - len(logical_line.lstrip(" "))
        if logical_line.startswith("\t"):
            raise YamlLimitError("tab indentation is not permitted in outer YAML")
        metrics.depth = max(metrics.depth, leading_spaces // 2)
        segments = _scalar_segments(logical_line)
        metrics.composed_nodes += max(1, len(segments))
        metrics.aggregate_scalar_characters += sum(len(segment) for segment in segments)
        metrics.anchors += anchors
        metrics.aliases += aliases
        metrics.expanded_visits += aliases
        metrics.wall_clock_seconds = int(time.monotonic() - started)
        enforce_yaml_limits(metrics, limits)

    return metrics


def validate_outer_yaml_file(path: pathlib.Path, limits: YamlResourceLimits = YamlResourceLimits()) -> None:
    """Read at most the selected byte limit plus one byte before decoding."""
    file_size = path.stat().st_size
    _check_limit("file bytes", file_size, limits.max_file_bytes)
    with path.open("rb") as handle:
        data = handle.read(limits.max_file_bytes + 1)
    _check_limit("file bytes", len(data), limits.max_file_bytes)
    try:
        text = data.decode("utf-8")
    except UnicodeDecodeError as exc:
        raise YamlLimitError(f"outer YAML is not UTF-8: {exc}") from exc
    analyze_yaml_preconstruction(text, limits)


def validate_outer_yaml_corpus(root: pathlib.Path) -> list[str]:
    """Validate only the eight declared outer YAML inputs, never delegated .ai YAML."""
    errors: list[str] = []
    for relative_path in OUTER_YAML_PATHS:
        path = root / relative_path
        if not path.is_file():
            errors.append(f"missing required outer YAML path: {relative_path}")
            continue
        try:
            validate_outer_yaml_file(path)
        except (OSError, YamlLimitError) as exc:
            errors.append(f"outer YAML preflight failed for {relative_path}: {exc}")
    return errors

def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--repo-kind", choices=("platform", "content"), required=True)
    args = parser.parse_args()

    root = pathlib.Path.cwd()
    errors: list[str] = []

    for rel in COMMON_REQUIRED + REPO_REQUIRED[args.repo_kind]:
        if not (root / rel).exists():
            errors.append(f"missing required path: {rel}")

    errors.extend(validate_rights_metadata(root, args.repo_kind))

    for path in root.rglob("*"):
        if not path.is_file() or ".git" in path.parts:
            continue
        try:
            text = path.read_text(encoding="utf-8")
        except UnicodeDecodeError:
            continue

        relative = path.relative_to(root)
        scan_text = secret_scan_text(relative.as_posix(), text)

        for pattern in SECRET_PATTERNS:
            if pattern.search(scan_text):
                errors.append(f"possible secret in {relative}")

        if path.suffix == ".json":
            try:
                json.loads(text)
            except json.JSONDecodeError as exc:
                errors.append(f"invalid JSON {path.relative_to(root)}: {exc}")

    manifest = root / "repo-manifest.yaml"
    if manifest.exists() and "repository:" not in manifest.read_text(encoding="utf-8"):
        errors.append("repo-manifest.yaml lacks repository key")

    if args.repo_kind == "platform":
        errors.extend(validate_outer_yaml_corpus(root))

    if args.repo_kind == "platform":
        ai_validator = root / ".ai" / "tools" / "validate_agent_os.py"
        if not ai_validator.is_file():
            errors.append("missing AI Agent OS validator: .ai/tools/validate_agent_os.py")
        else:
            completed = subprocess.run(
                [sys.executable, str(ai_validator)],
                cwd=root,
                check=False,
                text=True,
            )
            if completed.returncode != 0:
                errors.append("AI Agent OS validation failed")

    if errors:
        for error in errors:
            print(f"ERROR: {error}", file=sys.stderr)
        return 1

    unresolved = []
    for path in root.rglob("*"):
        if path.is_file() and ".git" not in path.parts:
            try:
                text = path.read_text(encoding="utf-8")
            except UnicodeDecodeError:
                continue
            if any(value in text for value in PLACEHOLDERS):
                unresolved.append(str(path.relative_to(root)))

    if unresolved:
        print("NOTICE: bootstrap placeholders remain in:")
        for item in unresolved:
            print(f"  - {item}")
        print("They must be replaced before pushing to GitHub.")

    print(f"Foundation validation passed for {args.repo_kind}.")
    return 0

if __name__ == "__main__":
    raise SystemExit(main())
