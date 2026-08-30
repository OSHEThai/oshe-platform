#!/usr/bin/env python3
"""Evaluate an ADR-0006 GitHub operation gate without performing the operation."""

from __future__ import annotations

import argparse
import hashlib
import json
import sys
from datetime import datetime
from pathlib import Path
from typing import Any

import yaml
from jsonschema import Draft202012Validator, FormatChecker


REPO_ROOT = Path(__file__).resolve().parents[2]
AI_ROOT = REPO_ROOT / ".ai"


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


def evaluate(record: dict[str, Any], now: datetime) -> list[str]:
    errors: list[str] = []
    policy = load_yaml(AI_ROOT / "policies" / "github-operations.yaml")
    route_registry = load_yaml(AI_ROOT / "provider-routes" / "ai-service-route-registry.yaml")
    credential_registry = load_yaml(AI_ROOT / "policies" / "github-credential-profiles.yaml")

    schema = load_json(AI_ROOT / "schemas" / "github-operation-gate.schema.json")
    validator = Draft202012Validator(schema, format_checker=FormatChecker())
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
    if credential_id not in set(credential_registry.get("approved_profile_ids") or []):
        errors.append("GitHub credential profile is not approved")

    expected_flags = set(policy["required_evidence_flags"])
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
    if now < requested_at or now > expires_at:
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
