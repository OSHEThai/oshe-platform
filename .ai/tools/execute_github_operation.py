#!/usr/bin/env python3
"""Execute one direct-gh operation only after its exact ADR-0006 gate passes."""

from __future__ import annotations

import argparse
import hashlib
import subprocess
import sys
from datetime import datetime
from pathlib import Path

from evaluate_github_operation import evaluate, load_yaml, parse_datetime


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("record", type=Path, help="Direct-gh operation gate YAML or JSON")
    parser.add_argument("--now", help="Override evaluation time for deterministic checks")
    parser.add_argument("--dry-run", action="store_true", help="Validate only; do not invoke gh")
    args = parser.parse_args()

    if args.now and not args.dry_run:
        print("DIRECT_GH_EXECUTION_DENY: --now is permitted only with --dry-run", file=sys.stderr)
        return 1

    try:
        record = load_yaml(args.record)
        now = parse_datetime(args.now) if args.now else datetime.now().astimezone()
        if record.get("actor", {}).get("execution_route_kind") != "DIRECT_GH_CLI":
            raise ValueError("gate is not a DIRECT_GH_CLI operation")
        errors = evaluate(record, now)
    except Exception as exc:  # pragma: no cover - diagnostic boundary
        print(f"DIRECT_GH_EXECUTION_DENY: {exc}", file=sys.stderr)
        return 1

    if errors:
        print(f"DIRECT_GH_EXECUTION_DENY ({len(errors)} reason(s))")
        for error in errors:
            print(f"- {error}")
        return 1

    command = record["execution"]["command"]
    if args.dry_run:
        print("DIRECT_GH_EXECUTION_DRY_RUN_PASS")
        print(f"gate_id={record['gate_id']}")
        return 0

    completed = subprocess.run(command, check=False, capture_output=True)
    print(f"DIRECT_GH_EXECUTION_RECEIPT gate_id={record['gate_id']} exit_code={completed.returncode}")
    print(f"stdout_sha256={hashlib.sha256(completed.stdout).hexdigest()}")
    print(f"stderr_sha256={hashlib.sha256(completed.stderr).hexdigest()}")
    return completed.returncode


if __name__ == "__main__":
    raise SystemExit(main())
