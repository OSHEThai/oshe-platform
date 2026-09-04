#!/usr/bin/env python3
"""Deterministic offline reference-mission benchmark; never dispatches a provider."""
from __future__ import annotations

import argparse
import hashlib
import json
import pathlib
import tempfile
from typing import Any


POLICIES = ("NONE", "TIMEOUT", "MALFORMED_OUTPUT", "STEP_LIMIT")
PUBLIC_RESPONSE_SCHEMA_VERSION = "1.0.0"


def canonical_bytes(value: Any) -> bytes:
    return (json.dumps(value, indent=2, sort_keys=True, ensure_ascii=True) + "\n").encode("utf-8")


def load_fixtures(path: pathlib.Path) -> list[dict[str, Any]]:
    data = json.loads(path.read_text(encoding="utf-8"))
    if data.get("schema_version") != "1.0.0" or not isinstance(data.get("fixtures"), list) or not data["fixtures"]:
        raise ValueError("fixture corpus must contain a non-empty schema_version 1.0.0 fixtures list")
    fixtures = data["fixtures"]
    ids: set[str] = set()
    for fixture in fixtures:
        if not isinstance(fixture, dict):
            raise ValueError("fixture must be an object")
        fixture_id = fixture.get("fixture_id")
        if not isinstance(fixture_id, str) or not fixture_id or fixture_id in ids:
            raise ValueError("fixture_id must be unique and non-empty")
        ids.add(fixture_id)
        if not isinstance(fixture.get("input"), dict) or not isinstance(fixture.get("expected_output"), dict):
            raise ValueError(f"fixture {fixture_id} must have object input and expected_output")
        if not isinstance(fixture.get("max_turns"), int) or fixture["max_turns"] < 1:
            raise ValueError(f"fixture {fixture_id} max_turns must be a positive integer")
    return fixtures


def load_public_response_observations(path: pathlib.Path) -> list[dict[str, Any]]:
    data = json.loads(path.read_text(encoding="utf-8"))
    observations = data.get("observations")
    if data.get("schema_version") != PUBLIC_RESPONSE_SCHEMA_VERSION or not isinstance(observations, list) or not observations:
        raise ValueError("public response observations must contain a non-empty schema_version 1.0.0 observations list")
    route_ids: set[str] = set()
    for observation in observations:
        if not isinstance(observation, dict):
            raise ValueError("public response observation must be an object")
        route_id = observation.get("route_id")
        if not isinstance(route_id, str) or not route_id or route_id in route_ids:
            raise ValueError("public response observation route_id must be unique and non-empty")
        route_ids.add(route_id)
        if not isinstance(observation.get("response"), dict) or not isinstance(observation.get("expected_response"), dict):
            raise ValueError(f"public response observation {route_id} must contain object response and expected_response")
        if not isinstance(observation.get("cli_version"), str) or not observation["cli_version"]:
            raise ValueError(f"public response observation {route_id} must contain cli_version")
        warning = observation.get("warning")
        if warning is not None and not isinstance(warning, str):
            raise ValueError(f"public response observation {route_id} warning must be a string or null")
    return observations


def compare_public_response_observations(observations: list[dict[str, Any]]) -> dict[str, Any]:
    candidates = []
    for observation in observations:
        matched = observation["response"] == observation["expected_response"]
        candidates.append(
            {
                "route_id": observation["route_id"],
                "cli_version": observation["cli_version"],
                "observed_runtime_model": observation.get("observed_runtime_model"),
                "structured_output_match": matched,
                "warning": observation.get("warning"),
            }
        )
    matched_count = sum(candidate["structured_output_match"] for candidate in candidates)
    warning_count = sum(candidate["warning"] is not None for candidate in candidates)
    return {
        "benchmark_id": "V010-I030-PUBLIC-RESPONSE-COMPARISON",
        "execution_mode": "RECORDED_PUBLIC_OBSERVATIONS_NO_DISPATCH",
        "provider_routes_enabled": 0,
        "candidates": candidates,
        "summary": {
            "observed": len(candidates),
            "structured_output_matches": matched_count,
            "structured_output_mismatches": len(candidates) - matched_count,
            "warnings": warning_count,
        },
    }


def run_fixture(fixture: dict[str, Any], policy: str, provider: str | None = None) -> dict[str, Any]:
    if provider is not None:
        raise ValueError("live providers are not authorized under H010-007")
    if policy not in POLICIES:
        raise ValueError(f"unknown failure policy: {policy}")
    fixture_id = fixture["fixture_id"]
    measures: dict[str, str] = {
        "fixture_schema_valid": "true",
        "provider_mode": "SYNTHETIC_OFFLINE",
        "route_selection": "NONE",
    }
    if policy == "NONE":
        measures.update({"expected_output_match": "true", "turns": "1"})
        status = "COMPLETED"
        output = fixture["expected_output"]
    elif policy == "TIMEOUT":
        measures.update({"timeout_triggered": "true", "turns": "0"})
        status, output = "FAILED", None
    elif policy == "MALFORMED_OUTPUT":
        measures.update({"parse_error_detected": "true", "turns": "1"})
        status, output = "FAILED", None
    else:
        measures.update({"step_limit_exceeded": "true", "turns": str(fixture["max_turns"])})
        status, output = "FAILED", None
    return {
        "failure_injection_policy": policy,
        "fixture_id": fixture_id,
        "measures": measures,
        "output": output,
        "status": status,
    }


def scorecard(fixtures: list[dict[str, Any]], policy: str) -> dict[str, Any]:
    results = [run_fixture(fixture, policy) for fixture in fixtures]
    completed = sum(result["status"] == "COMPLETED" for result in results)
    return {
        "benchmark_id": "V010-I030-OFFLINE-REFERENCE",
        "failure_injection_policy": policy,
        "fixtures": results,
        "provider_routes_enabled": 0,
        "summary": {"completed": completed, "failed": len(results) - completed, "total": len(results)},
    }


def verify(fixtures: list[dict[str, Any]]) -> None:
    baseline = scorecard(fixtures, "NONE")
    if baseline["summary"]["completed"] != len(fixtures):
        raise ValueError("baseline did not complete every fixture")
    for result, fixture in zip(baseline["fixtures"], fixtures, strict=True):
        if result["output"] != fixture["expected_output"] or result["measures"]["route_selection"] != "NONE":
            raise ValueError("baseline output or route boundary mismatch")
    for policy, marker in (("TIMEOUT", "timeout_triggered"), ("MALFORMED_OUTPUT", "parse_error_detected"), ("STEP_LIMIT", "step_limit_exceeded")):
        injected = scorecard(fixtures, policy)
        if injected["summary"]["failed"] != len(fixtures) or not all(item["measures"].get(marker) == "true" for item in injected["fixtures"]):
            raise ValueError(f"failure injection did not fail closed: {policy}")


def write_scorecard(output: pathlib.Path, contents: dict[str, Any]) -> None:
    if output.exists():
        raise FileExistsError("refusing to replace an existing benchmark scorecard")
    output.parent.mkdir(parents=True, exist_ok=True)
    output.write_bytes(canonical_bytes(contents))


def main() -> int:
    parser = argparse.ArgumentParser(description="Run deterministic offline reference benchmark fixtures.")
    parser.add_argument("--verify", action="store_true")
    parser.add_argument("--emit", action="store_true")
    parser.add_argument("--policy", choices=POLICIES, default="NONE")
    parser.add_argument("--fixture-file", default="tests/fixtures/reference_benchmark/synthetic_cases.json")
    parser.add_argument("--output", default="artifacts/reference-benchmark/scorecard.json")
    parser.add_argument("--provider")
    args = parser.parse_args()
    if args.verify == args.emit:
        parser.error("choose exactly one of --verify or --emit")
    root = pathlib.Path.cwd().resolve()
    try:
        if args.provider is not None:
            raise ValueError("live providers are not authorized under H010-007")
        fixture_path = (root / args.fixture_file).resolve()
        if root not in fixture_path.parents:
            parser.error("fixture file must remain inside the repository")
        fixtures = load_fixtures(fixture_path)
        if args.verify:
            verify(fixtures)
            print("Reference benchmark verification passed")
            return 0
        output = (root / args.output).resolve()
        if root not in output.parents:
            parser.error("output must remain inside the repository")
        card = scorecard(fixtures, args.policy)
        card["scorecard_sha256"] = hashlib.sha256(canonical_bytes(card)).hexdigest()
        write_scorecard(output, card)
        print(f"Reference benchmark scorecard written to {output}")
        return 0
    except (FileExistsError, ValueError, json.JSONDecodeError) as exc:
        print(f"ERROR: {exc}")
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
