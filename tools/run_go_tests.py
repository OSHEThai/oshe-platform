#!/usr/bin/env python3
"""
tools/run_go_tests.py
Deterministic, non-fail-fast Go test runner for all workspace Go modules.
Discovers and executes 'go test ./...' across all go.mod locations.
Fails closed when the Go toolchain or required runtime prerequisites are missing.
"""

from __future__ import annotations

import argparse
import json
import os
import pathlib
import shutil
import subprocess
import sys
import time
from typing import Any, Dict, List, Optional, Tuple


EXCLUDED_PARTS = {
    ".git",
    ".local-ci",
    ".worktrees",
    ".herdrops-worktrees",
    "vendor",
    "node_modules",
    "bin",
    "dist",
}


def find_repo_root(start: Optional[pathlib.Path] = None) -> pathlib.Path:
    current = (start or pathlib.Path.cwd()).resolve()
    while current != current.parent:
        if (current / "apps").is_dir() and (current / "modules").is_dir():
            return current
        current = current.parent
    return (start or pathlib.Path.cwd()).resolve()


def check_go_toolchain() -> Tuple[bool, str]:
    go_path = shutil.which("go")
    if not go_path:
        return False, "Go executable ('go') not found in PATH."
    try:
        proc = subprocess.run(
            [go_path, "version"],
            capture_output=True,
            text=True,
            check=False,
            timeout=10,
        )
        if proc.returncode != 0:
            return False, f"go version failed with exit {proc.returncode}: {proc.stderr.strip()}"
        return True, proc.stdout.strip()
    except Exception as exc:
        return False, f"Failed to execute go version: {exc}"


def discover_go_modules(root: pathlib.Path) -> List[pathlib.Path]:
    modules: List[pathlib.Path] = []
    for mod in root.rglob("go.mod"):
        if not mod.is_file():
            continue
        rel = mod.relative_to(root)
        if any(part in EXCLUDED_PARTS for part in rel.parts):
            continue
        modules.append(mod.parent)
    return sorted(modules)


def run_module_tests(
    module_dir: pathlib.Path,
    timeout: int = 120,
    verbose: bool = False,
) -> Dict[str, Any]:
    start_time = time.perf_counter()
    try:
        cmd = ["go", "test", "./..."]
        if verbose:
            cmd.append("-v")
        proc = subprocess.run(
            cmd,
            cwd=module_dir,
            capture_output=True,
            text=True,
            check=False,
            timeout=timeout,
        )
        elapsed = time.perf_counter() - start_time
        return {
            "returncode": proc.returncode,
            "stdout": proc.stdout,
            "stderr": proc.stderr,
            "elapsed_seconds": round(elapsed, 3),
            "timed_out": False,
        }
    except subprocess.TimeoutExpired as exc:
        elapsed = time.perf_counter() - start_time
        return {
            "returncode": -1,
            "stdout": exc.stdout or "" if isinstance(exc.stdout, str) else "",
            "stderr": f"Test timed out after {timeout} seconds",
            "elapsed_seconds": round(elapsed, 3),
            "timed_out": True,
        }
    except Exception as exc:
        elapsed = time.perf_counter() - start_time
        return {
            "returncode": -2,
            "stdout": "",
            "stderr": f"Execution error: {exc}",
            "elapsed_seconds": round(elapsed, 3),
            "timed_out": False,
        }


def main() -> int:
    parser = argparse.ArgumentParser(
        description="Run Go tests across all discovered modules without fail-fast behavior."
    )
    parser.add_argument(
        "--root",
        type=pathlib.Path,
        default=None,
        help="Repository root directory (defaults to auto-detected repository root)",
    )
    parser.add_argument(
        "--timeout",
        type=int,
        default=120,
        help="Per-module test execution timeout in seconds (default: 120)",
    )
    parser.add_argument(
        "--json",
        action="store_true",
        help="Output full test execution report as JSON",
    )
    parser.add_argument(
        "-v",
        "--verbose",
        action="store_true",
        help="Pass -v to 'go test'",
    )
    parser.add_argument(
        "--module",
        action="append",
        dest="specific_modules",
        help="Test only specific module paths relative to root (can be specified multiple times)",
    )
    args = parser.parse_args()

    root = find_repo_root(args.root)

    # 1. Prerequisite check - fail closed if Go is missing
    go_available, toolchain_info = check_go_toolchain()
    if not go_available:
        msg = f"ERROR: Go toolchain prerequisite failure: {toolchain_info}"
        print(msg, file=sys.stderr)
        if args.json:
            print(
                json.dumps(
                    {
                        "prerequisite_satisfied": False,
                        "error": toolchain_info,
                        "modules_tested": 0,
                        "passed": 0,
                        "failed": 0,
                    },
                    indent=2,
                )
            )
        return 2

    # 2. Discover modules
    if args.specific_modules:
        modules = [
            (root / m).resolve()
            for m in args.specific_modules
            if (root / m / "go.mod").is_file()
        ]
    else:
        modules = discover_go_modules(root)

    if not modules:
        msg = "ERROR: No Go modules with go.mod found in repository."
        print(msg, file=sys.stderr)
        return 1

    print(f"Toolchain: {toolchain_info}")
    print(f"Discovered {len(modules)} Go modules in {root.as_posix()}:")

    passed_modules: List[str] = []
    failed_modules: List[Dict[str, Any]] = []

    # 3. Non-fail-fast execution across all modules
    for mod in modules:
        rel_path = mod.relative_to(root).as_posix()
        sys.stdout.write(f"  RUN   {rel_path} ... ")
        sys.stdout.flush()

        res = run_module_tests(mod, timeout=args.timeout, verbose=args.verbose)
        if res["returncode"] == 0:
            sys.stdout.write(f"PASS ({res['elapsed_seconds']}s)\n")
            sys.stdout.flush()
            passed_modules.append(rel_path)
        else:
            sys.stdout.write(f"FAIL (exit {res['returncode']}, {res['elapsed_seconds']}s)\n")
            sys.stdout.flush()
            # Print brief error context
            output_snippet = (res["stderr"] or res["stdout"]).strip()
            if output_snippet:
                lines = output_snippet.splitlines()
                for line in lines[:8]:
                    print(f"    {line}", file=sys.stderr)
                if len(lines) > 8:
                    print(f"    ... [{len(lines) - 8} lines omitted]", file=sys.stderr)
            failed_modules.append(
                {
                    "module": rel_path,
                    "returncode": res["returncode"],
                    "elapsed_seconds": res["elapsed_seconds"],
                    "stdout": res["stdout"],
                    "stderr": res["stderr"],
                    "timed_out": res["timed_out"],
                }
            )

    # 4. Summary and exit disposition
    print("\n" + "=" * 60)
    print(
        f"Go Test Summary: {len(passed_modules)} passed, {len(failed_modules)} failed out of {len(modules)} modules"
    )
    print("=" * 60)

    if args.json:
        report = {
            "prerequisite_satisfied": True,
            "toolchain": toolchain_info,
            "total_modules": len(modules),
            "passed_count": len(passed_modules),
            "failed_count": len(failed_modules),
            "passed_modules": passed_modules,
            "failed_modules": [
                {
                    "module": f["module"],
                    "returncode": f["returncode"],
                    "elapsed_seconds": f["elapsed_seconds"],
                    "snippet": (f["stderr"] or f["stdout"]).strip()[:300],
                }
                for f in failed_modules
            ],
        }
        print(json.dumps(report, indent=2))

    if failed_modules:
        print("\nFailed Go modules:", file=sys.stderr)
        for f in failed_modules:
            print(f"  - {f['module']} (exit {f['returncode']})", file=sys.stderr)
        return 1

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
