#!/usr/bin/env python3
"""Validate and rehearse a deterministic local mock mission without dispatching agents."""
from __future__ import annotations

import argparse
import copy
import json
import pathlib
from typing import Any


REQUIRED_ROLES = {"PM", "PM Secretary"}


def canonical_bytes(value: Any) -> bytes:
    return (json.dumps(value, indent=2, sort_keys=True, ensure_ascii=True) + "\n").encode("utf-8")


def load_mission(path: pathlib.Path) -> dict[str, Any]:
    mission = json.loads(path.read_text(encoding="utf-8"))
    validate_mission(mission)
    return mission


def validate_mission(mission: dict[str, Any]) -> None:
    if mission.get("schema_version") != "1.0.0" or not isinstance(mission.get("mission_id"), str):
        raise ValueError("mission requires schema_version 1.0.0 and mission_id")
    roles = mission.get("visible_roles")
    tasks = mission.get("tasks")
    if not isinstance(roles, list) or not REQUIRED_ROLES.issubset(set(roles)):
        raise ValueError("mission must include PM and PM Secretary visible roles")
    if not isinstance(tasks, list) or not tasks:
        raise ValueError("mission requires non-empty tasks")
    ids: set[str] = set()
    for task in tasks:
        if not isinstance(task, dict):
            raise ValueError("task must be an object")
        task_id = task.get("task_id")
        owner = task.get("owner_role")
        if not isinstance(task_id, str) or not task_id or task_id in ids:
            raise ValueError("task_id must be unique and non-empty")
        ids.add(task_id)
        if owner not in roles or "Worker" in str(owner):
            raise ValueError(f"task {task_id} has an untracked or retired owner role")
        if task.get("hidden_agent") is not False:
            raise ValueError(f"task {task_id} permits hidden authority")
        if task.get("result_contract") != "STRUCTURED_JSON":
            raise ValueError(f"task {task_id} lacks a structured result contract")
        if task.get("provider_mode") != "SYNTHETIC_OFFLINE" or task.get("route_id") is not None:
            raise ValueError(f"task {task_id} selects a live provider route")
        if owner.endswith("Lead") and task.get("report_to") != "PM Secretary":
            raise ValueError(f"task {task_id} Lead must report to PM Secretary")
        if owner == "PM Secretary" and task.get("report_to") != "PM":
            raise ValueError(f"task {task_id} PM Secretary must report to PM")
        if owner == "PM" and task.get("report_to") is not None:
            raise ValueError(f"task {task_id} PM does not report within the mock hierarchy")
        paths = task.get("write_paths")
        if not isinstance(paths, list) or not paths or not all(isinstance(path, str) and path for path in paths):
            raise ValueError(f"task {task_id} requires non-empty exact write paths")
    for task in tasks:
        deps = task.get("depends_on")
        if not isinstance(deps, list) or any(dep not in ids or dep == task["task_id"] for dep in deps):
            raise ValueError(f"task {task['task_id']} has invalid dependencies")


def assert_disjoint_writes(tasks: list[dict[str, Any]]) -> None:
    paths: set[str] = set()
    for task in tasks:
        for path in task["write_paths"]:
            if path in paths:
                raise ValueError(f"concurrent tasks overlap write path: {path}")
            paths.add(path)


def rehearse(mission: dict[str, Any]) -> dict[str, Any]:
    validate_mission(mission)
    remaining = {task["task_id"]: task for task in mission["tasks"]}
    completed: set[str] = set()
    events: list[dict[str, Any]] = []
    batch = 0
    while remaining:
        ready = sorted((task for task in remaining.values() if set(task["depends_on"]).issubset(completed)), key=lambda task: task["task_id"])
        if not ready:
            raise ValueError("mission task graph contains a dependency cycle")
        assert_disjoint_writes(ready)
        batch += 1
        for task in ready:
            events.append({"batch": batch, "event": "TASK_COMPLETED", "owner_role": task["owner_role"], "report_to": task["report_to"], "task_id": task["task_id"]})
            completed.add(task["task_id"])
            del remaining[task["task_id"]]
    return {
        "mission_id": mission["mission_id"],
        "provider_routes_enabled": 0,
        "qualification": "MOCK_ONLY_NOT_LIVE",
        "result_contract": "STRUCTURED_JSON",
        "status": "COMPLETED",
        "trace": events,
    }


def write_trace(path: pathlib.Path, trace: dict[str, Any]) -> None:
    if path.exists():
        raise FileExistsError("refusing to replace an existing mission rehearsal trace")
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_bytes(canonical_bytes(trace))


def verify(mission: dict[str, Any]) -> None:
    first, second = rehearse(mission), rehearse(copy.deepcopy(mission))
    if first != second or first["status"] != "COMPLETED" or first["provider_routes_enabled"] != 0:
        raise ValueError("mock rehearsal is not deterministic and offline")
    if not any(event["owner_role"].endswith("Lead") and event["report_to"] == "PM Secretary" for event in first["trace"]):
        raise ValueError("mock rehearsal lacks Lead to PM Secretary reporting")


def main() -> int:
    parser = argparse.ArgumentParser(description="Rehearse a deterministic offline mission DAG.")
    parser.add_argument("--verify", action="store_true")
    parser.add_argument("--emit", action="store_true")
    parser.add_argument("--mission-file", default="tests/fixtures/mission_rehearsal/offline_mission.json")
    parser.add_argument("--output", default="artifacts/mission-rehearsal/trace.json")
    args = parser.parse_args()
    if args.verify == args.emit:
        parser.error("choose exactly one of --verify or --emit")
    root = pathlib.Path.cwd().resolve()
    try:
        mission_file = (root / args.mission_file).resolve()
        if root not in mission_file.parents:
            parser.error("mission file must remain inside the repository")
        mission = load_mission(mission_file)
        if args.verify:
            verify(mission)
            print("Mission rehearsal verification passed")
            return 0
        output = (root / args.output).resolve()
        if root not in output.parents:
            parser.error("output must remain inside the repository")
        write_trace(output, rehearse(mission))
        print(f"Mission rehearsal trace written to {output}")
        return 0
    except (FileExistsError, ValueError, json.JSONDecodeError) as exc:
        print(f"ERROR: {exc}")
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
