#!/usr/bin/env python3
"""Evaluate an ADR-0006 GitHub operation gate without performing the operation."""

from __future__ import annotations

import argparse
import hashlib
import json
import re
import sys
import unicodedata
from datetime import datetime
from pathlib import Path
from typing import Any

import yaml
from jsonschema import Draft202012Validator, FormatChecker


REPO_ROOT = Path(__file__).resolve().parents[2]
AI_ROOT = REPO_ROOT / ".ai"
REQUIRED_POLICY_PROHIBITED_ACTIONS = frozenset({"repository-delete"})
ALWAYS_PROHIBITED_ACTION_ALIASES = frozenset(
    {
        "repository-delete",
        "delete-repository",
        "repo-delete",
    }
)
ACTION_SEPARATOR_PATTERN = re.compile(
    r"[\s_\-\u2010-\u2015\u2043\u2212\ufe58\ufe63\uff0d]+"
)


def load_yaml(path: Path) -> Any:
    with path.open("r", encoding="utf-8-sig") as stream:
        return yaml.safe_load(stream)


def load_json(path: Path) -> Any:
    with path.open("r", encoding="utf-8-sig") as stream:
        return json.load(stream)


def parse_datetime(value: str) -> datetime:
    parsed = datetime.fromisoformat(value.replace("Z", "+00:00"))
    if parsed.tzinfo is None:
        raise ValueError("timestamp must include a timezone")
    return parsed


def canonical_digest(record: dict[str, Any]) -> str:
    payload = json.dumps(record, sort_keys=True, separators=(",", ":"), ensure_ascii=False)
    return "sha256:" + hashlib.sha256(payload.encode("utf-8")).hexdigest()


def command_digest(command: list[str]) -> str:
    payload = json.dumps(command, separators=(",", ":"), ensure_ascii=False)
    return "sha256:" + hashlib.sha256(payload.encode("utf-8")).hexdigest()


def normalize_action(value: str) -> str:
    normalized = unicodedata.normalize("NFKC", value).casefold().strip()
    return ACTION_SEPARATOR_PATTERN.sub("-", normalized).strip("-")


def classify_direct_gh_command(command: list[str], repository: str) -> tuple[str | None, str | None]:
    """Return (action-id, action-class) for a bounded direct-gh command."""
    if len(command) < 4 or command[0] != "gh":
        return None, None

    repo_values: list[str] = []
    index = 1
    while index < len(command):
        value = command[index]
        if value in {"--repo", "-R"}:
            if index + 1 >= len(command):
                return None, None
            repo_values.append(command[index + 1])
            index += 2
            continue
        if value.startswith("--repo="):
            repo_values.append(value.removeprefix("--repo="))
        index += 1
    if repo_values != [repository]:
        return None, None

    noun, verb = command[1], command[2]
    action_id = normalize_action(f"{noun}-{verb}")
    if noun == "issue" and verb in {"create", "edit", "comment", "close", "reopen", "lock", "unlock", "pin", "unpin"}:
        return action_id, "METADATA"
    if noun == "pr" and verb in {"create", "edit", "comment", "review", "ready", "reopen"}:
        return action_id, "BRANCH_PR"
    if noun == "pr" and verb == "merge":
        return action_id, "MERGE"
    if noun == "workflow" and verb in {"run", "rerun", "cancel"}:
        return action_id, "WORKFLOW"
    if noun == "release" and verb in {"create", "edit", "upload", "delete"}:
        return action_id, "RELEASE"
    if noun == "repo" and verb in {"create", "edit", "archive", "sync", "rename"}:
        return action_id, "REPOSITORY_ADMIN"
    if noun in {"secret", "variable"} and verb in {"set", "delete", "list"}:
        return action_id, "CREDENTIAL"
    return None, None


def evaluate(record: dict[str, Any], now: datetime) -> list[str]:
    errors: list[str] = []
    policy = load_yaml(AI_ROOT / "policies" / "github-operations.yaml")
    credential_registry = load_yaml(AI_ROOT / "policies" / "github-credential-profiles.yaml")

    schema = load_json(AI_ROOT / "schemas" / "github-operation-gate.schema.json")
    validator = Draft202012Validator(schema, format_checker=FormatChecker())

    configured_values = policy.get("prohibited_actions")
    configured_prohibited_actions = {
        normalize_action(value)
        for value in configured_values
        if isinstance(value, str)
    } if isinstance(configured_values, list) else set()
    if configured_prohibited_actions != REQUIRED_POLICY_PROHIBITED_ACTIONS:
        errors.append("GitHub prohibited action policy diverges from the hard-coded safety baseline")

    prohibited_actions = ALWAYS_PROHIBITED_ACTION_ALIASES | configured_prohibited_actions
    scope = record.get("scope")
    action = scope.get("action") if isinstance(scope, dict) else None
    if isinstance(action, str) and normalize_action(action) in prohibited_actions:
        errors.append("GitHub action is always prohibited: repository-delete")

    for error in sorted(validator.iter_errors(record), key=lambda item: list(item.path)):
        location = "/".join(str(part) for part in error.path) or "<root>"
        errors.append(f"schema {location}: {error.message}")
    if errors:
        return errors

    if record["human_authority_ref"] != "ADR-0006":
        errors.append("standing human authority must resolve to ADR-0006")
    if record["actor"]["role_id"] != policy["authorized_role_id"]:
        errors.append("actor role is not authorized by the GitHub operations policy")
    if record["actor"]["specialist_profile_id"] != policy["required_specialist_profile_id"]:
        errors.append("github-manager specialist profile is required")
    if record["scope"]["organization"] not in policy["organization_allowlist"]:
        errors.append("organization is not allowlisted")

    execution_kind = record["actor"].get("execution_route_kind", "AI_PROVIDER")
    if execution_kind == "DIRECT_GH_CLI":
        execution = record["execution"]
        command = execution["command"]
        qualified_repository = f"{record['scope']['organization']}/{record['scope']['repository']}"
        if execution["command_digest"] != command_digest(command):
            errors.append("direct gh command digest does not match the exact command")
        actual_action, actual_action_class = classify_direct_gh_command(command, qualified_repository)
        if actual_action is None or actual_action_class is None:
            errors.append("direct gh command is unclassified, lacks one exact --repo, or is not permitted through the gate executor")
        else:
            if normalize_action(record["scope"]["action"]) != actual_action:
                errors.append("direct gh scope action does not match the exact command")
            if record["scope"]["action_class"] != actual_action_class:
                errors.append("direct gh scope action class does not match the exact command")
        direct_route_id = record["actor"].get("execution_route_id")
        direct_routes = policy.get("direct_gh_execution_routes") or []
        direct_route = next((item for item in direct_routes if item.get("execution_route_id") == direct_route_id), None)
        if not direct_route or direct_route.get("lifecycle_status") != "APPROVED_ACTIVE":
            errors.append("direct gh execution route is not approved and active")
        else:
            if record["scope"]["organization"] not in set(direct_route.get("organization_allowlist") or []):
                errors.append("organization is not allowlisted for direct gh execution route")
            if qualified_repository not in set(direct_route.get("repository_allowlist") or []):
                errors.append("repository is not allowlisted for direct gh execution route")
            if record["actor"]["credential_profile_id"] != direct_route.get("credential_profile_id"):
                errors.append("direct gh execution route credential profile does not match")
            try:
                activated_at = parse_datetime(direct_route["activated_at"])
                expires_at = parse_datetime(direct_route["expires_at"])
                if now < activated_at or now >= expires_at:
                    errors.append("direct gh execution route is inactive or expired")
            except (KeyError, TypeError, ValueError):
                errors.append("direct gh execution route has an invalid expiry")
    else:
        route_registry = load_yaml(AI_ROOT / "provider-routes" / "ai-service-route-registry.yaml")
        route_id = record["actor"]["provider_route_id"]
        approved_routes = set(route_registry.get("approved_route_ids") or [])
        enabled_routes = set(route_registry.get("enabled_route_ids") or [])
        active_routes = set(route_registry.get("active_route_ids") or [])
        route = next((item for item in route_registry.get("routes", []) if item.get("provider_route_id") == route_id), None)
        if route_id not in approved_routes or route_id not in enabled_routes or route_id not in active_routes:
            errors.append("provider route is not approved, enabled, and active")
        elif not route or route.get("lifecycle", {}).get("dispatch_enabled") is not True:
            errors.append("provider route lifecycle is not dispatch-enabled")

    credential_id = record["actor"]["credential_profile_id"]
    credential_profile = next((item for item in credential_registry.get("profiles", []) if item.get("credential_profile_id") == credential_id), None)
    if credential_id not in set(credential_registry.get("approved_profile_ids") or []):
        errors.append("GitHub credential profile is not approved")
    if execution_kind == "DIRECT_GH_CLI":
        if not credential_profile or credential_profile.get("approval_status") != "APPROVED_ACTIVE":
            errors.append("direct gh credential profile is not approved and active")
        else:
            if record["scope"]["organization"] not in set(credential_profile.get("organization_allowlist") or []):
                errors.append("organization is not allowlisted for direct gh credential profile")
            if record["scope"]["repository"] not in set(credential_profile.get("repository_allowlist") or []):
                errors.append("repository is not allowlisted for direct gh credential profile")
            try:
                if now >= parse_datetime(credential_profile["expires_at"]):
                    errors.append("direct gh credential profile is expired")
            except (KeyError, TypeError, ValueError):
                errors.append("direct gh credential profile has an invalid expiry")

    expected_flags = set(policy["required_evidence_flags"])
    expected_flags.add(policy["execution_route_evidence_flags"][execution_kind])
    supplied_flags = set(record["evidence"])
    if supplied_flags != expected_flags:
        errors.append("evidence flag set does not exactly match policy")
    for name in sorted(expected_flags & supplied_flags):
        item = record["evidence"][name]
        if item.get("satisfied") is not True:
            errors.append(f"evidence is not satisfied: {name}")
        if not item.get("refs"):
            errors.append(f"evidence has no references: {name}")

    if record["unresolved_blockers"]:
        errors.append("unresolved blockers are present")

    action_class = record["scope"]["action_class"]
    high_impact = action_class in set(policy["high_impact_action_classes"])
    review = record["independent_review"]
    if high_impact:
        if review.get("reviewer_role_id") != "independent-review-challenge-agent":
            errors.append("high-impact operation requires the independent-review-challenge-agent")
        if not review.get("reviewer_assignment_id") or review.get("reviewer_assignment_id") == record["assignment_id"]:
            errors.append("high-impact operation requires a distinct reviewer assignment")
        if review.get("disposition") != "PASS" or not review.get("review_ref"):
            errors.append("high-impact operation requires an independent review PASS reference")
    elif review.get("disposition") not in {"PASS", "NOT_REQUIRED"}:
        errors.append("metadata or branch operation review disposition is incomplete")

    if action_class == "DEPLOYMENT_TRIGGER" and not record.get("external_authority_ref"):
        errors.append("deployment-triggering operation requires separate external authority")

    requested_at = parse_datetime(record["requested_at"])
    expires_at = parse_datetime(record["expires_at"])
    if expires_at <= requested_at:
        errors.append("gate expiry must be after request time")
    validity_seconds = policy["gate_validity_minutes"] * 60
    if (expires_at - requested_at).total_seconds() > validity_seconds:
        errors.append("gate validity window exceeds policy")
    if now < requested_at or now >= expires_at:
        errors.append("gate is not within its validity window")

    return errors


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("record", type=Path, help="GitHub operation gate YAML or JSON")
    parser.add_argument("--now", help="Override evaluation time for deterministic tests")
    args = parser.parse_args()

    try:
        record = load_yaml(args.record)
        now = parse_datetime(args.now) if args.now else datetime.now().astimezone()
        errors = evaluate(record, now)
    except Exception as exc:  # pragma: no cover - diagnostic boundary
        print(f"GITHUB_OPERATION_GATE_DENY: {exc}", file=sys.stderr)
        return 1

    if errors:
        print(f"GITHUB_OPERATION_GATE_DENY ({len(errors)} reason(s))")
        for error in errors:
            print(f"- {error}")
        return 1

    print("GITHUB_OPERATION_GATE_PASS")
    print(f"gate_id={record['gate_id']}")
    print(f"record_digest={canonical_digest(record)}")
    print(f"scope={record['scope']['organization']}/{record['scope']['repository']}:{record['scope']['action_class']}:{record['scope']['target']}")
    print(f"expires_at={record['expires_at']}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
