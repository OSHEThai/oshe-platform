from __future__ import annotations

import copy
import json
import unittest
from datetime import datetime
from pathlib import Path
from typing import Any

import yaml
from jsonschema import Draft202012Validator


REPO_ROOT = Path(__file__).resolve().parents[1]

REQUIRED_DELEGATED_TASK_GUARDS = {
    "visible-parent-assignment",
    "narrower-or-equal-authority",
    "non-overlapping-write-scope",
    "explicit-output-contract",
    "independent-review-when-required",
}

REQUIRED_SPECIALIST_RULES = {
    "profile-scope-must-be-narrower-than-role-and-assignment",
    "hidden-delegation-is-prohibited",
    "specialist-profile-may-not-spawn",
    "independent-review-requirements-remain-active",
}

REQUIRED_PERMISSION_PROHIBITIONS = {
    "self-approve",
    "spawn-hidden-subagent",
    "use-unregistered-session",
    "bypass-required-review",
}


def load_yaml(relative_path: str) -> dict[str, Any]:
    value = yaml.safe_load((REPO_ROOT / relative_path).read_text(encoding="utf-8"))
    if not isinstance(value, dict):
        raise AssertionError(f"{relative_path}: expected a YAML object")
    return value


def load_json(relative_path: str) -> dict[str, Any]:
    value = json.loads((REPO_ROOT / relative_path).read_text(encoding="utf-8"))
    if not isinstance(value, dict):
        raise AssertionError(f"{relative_path}: expected a JSON object")
    return value


def require_equal(key: str, actual: Any, expected: Any) -> None:
    if actual != expected:
        raise AssertionError(f"{key}: expected {expected!r}, got {actual!r}")


def require_guards(key: str, actual: Any, required: set[str]) -> None:
    actual_set = set(actual or [])
    missing = sorted(required - actual_set)
    if missing:
        raise AssertionError(f"{key}: missing {', '.join(missing)}")


def assert_declared_delegation_contract(
    role_registry: dict[str, Any],
    delegation_policy: dict[str, Any],
    specialist_registry: dict[str, Any],
    permissions: dict[str, Any],
) -> None:
    role_controls = role_registry.get("global_controls") or {}
    require_equal("roles.global_controls.workers_may_spawn", role_controls.get("workers_may_spawn"), False)
    require_equal(
        "roles.global_controls.provider_native_hidden_subagents",
        role_controls.get("provider_native_hidden_subagents"),
        "PROHIBITED",
    )
    require_equal(
        "roles.global_controls.unregistered_sessions",
        role_controls.get("unregistered_sessions"),
        "PROHIBITED",
    )
    require_equal(
        "roles.global_controls.maximum_visible_delegation_depth",
        role_controls.get("maximum_visible_delegation_depth"),
        2,
    )

    require_equal("delegation.hidden_delegation", delegation_policy.get("hidden_delegation"), "PROHIBITED")
    require_equal(
        "delegation.provider_native_hidden_subagents",
        delegation_policy.get("provider_native_hidden_subagents"),
        "PROHIBITED",
    )
    require_equal(
        "delegation.unregistered_sessions",
        delegation_policy.get("unregistered_sessions"),
        "PROHIBITED",
    )
    require_equal(
        "delegation.specialist_profiles_may_spawn",
        delegation_policy.get("specialist_profiles_may_spawn"),
        False,
    )
    require_equal(
        "delegation.maximum_visible_delegation_depth",
        delegation_policy.get("maximum_visible_delegation_depth"),
        role_controls.get("maximum_visible_delegation_depth"),
    )
    require_guards(
        "delegation.delegated_task_requires",
        delegation_policy.get("delegated_task_requires"),
        REQUIRED_DELEGATED_TASK_GUARDS,
    )
    require_guards(
        "agents.rules",
        specialist_registry.get("rules"),
        REQUIRED_SPECIALIST_RULES,
    )
    require_guards(
        "permissions.global_agent_prohibitions",
        permissions.get("global_agent_prohibitions"),
        REQUIRED_PERMISSION_PROHIBITIONS,
    )


def assert_assignment_profile_parent(
    assignment: dict[str, Any], specialist_registry: dict[str, Any]
) -> None:
    role = assignment.get("role") or {}
    profile_id = role.get("specialist_profile")
    role_id = role.get("role_id")
    profiles = [
        profile
        for profile in specialist_registry.get("profiles") or []
        if profile.get("profile_id") == profile_id
    ]
    if len(profiles) != 1:
        raise AssertionError(f"specialist_profile: expected one registry entry for {profile_id!r}")
    if role_id not in (profiles[0].get("parent_roles") or []):
        raise AssertionError(
            f"specialist_profile.parent_roles: {profile_id!r} does not permit {role_id!r}"
        )


def assert_no_same_assignment_self_review(assignment: dict[str, Any]) -> None:
    assignment_id = assignment.get("assignment_id")
    reviewer_assignment_id = (assignment.get("review") or {}).get("reviewer_assignment_id")
    if reviewer_assignment_id is not None and reviewer_assignment_id == assignment_id:
        raise AssertionError(
            f"reviewer_assignment_id: self-review prohibited for {assignment_id!r}"
        )


def assert_no_unapproved_fallback(routing_policy: dict[str, Any]) -> None:
    require_equal("routing.routing_status", routing_policy.get("routing_status"), "NO_APPROVED_ROUTES")
    for role_id, route in sorted((routing_policy.get("role_routes") or {}).items()):
        require_equal(f"routing.{role_id}.primary_route_id", route.get("primary_route_id"), None)
        require_equal(f"routing.{role_id}.fallback_route_ids", route.get("fallback_route_ids"), [])
        require_equal(f"routing.{role_id}.dispatch_enabled", route.get("dispatch_enabled"), False)


def parse_timestamp(value: Any, lease_id: str, field: str) -> datetime:
    if not isinstance(value, str):
        raise AssertionError(f"{lease_id}.{field}: ACTIVE lease requires an ISO timestamp")
    return datetime.fromisoformat(value.replace("Z", "+00:00"))


def normalize_literal_path(value: str) -> str:
    return value.replace("\\", "/")


def assert_no_exact_active_write_path_collisions(leases: list[dict[str, Any]]) -> None:
    active = [lease for lease in leases if lease.get("state") == "ACTIVE"]
    for index, left in enumerate(active):
        left_id = str(left.get("lease_id"))
        left_start = parse_timestamp(left.get("activated_at"), left_id, "activated_at")
        left_end = parse_timestamp(left.get("expires_at"), left_id, "expires_at")
        if left_start >= left_end:
            raise AssertionError(f"{left_id}.timing: activated_at must precede expires_at")
        for right in active[index + 1 :]:
            right_id = str(right.get("lease_id"))
            right_start = parse_timestamp(right.get("activated_at"), right_id, "activated_at")
            right_end = parse_timestamp(right.get("expires_at"), right_id, "expires_at")
            if right_start >= right_end:
                raise AssertionError(f"{right_id}.timing: activated_at must precede expires_at")
            intervals_overlap = left_start < right_end and right_start < left_end
            same_lease_domain = (
                left.get("repository"),
                left.get("worktree"),
                left.get("branch"),
            ) == (
                right.get("repository"),
                right.get("worktree"),
                right.get("branch"),
            )
            duplicate_paths = {
                normalize_literal_path(path) for path in left.get("allowed_paths") or []
            } & {
                normalize_literal_path(path) for path in right.get("allowed_paths") or []
            }
            if intervals_overlap and same_lease_domain and duplicate_paths:
                duplicate = sorted(duplicate_paths)[0]
                raise AssertionError(
                    f"allowed_paths: exact active collision for {duplicate!r} between {left_id} and {right_id}"
                )


class PermissionDelegationContractTests(unittest.TestCase):
    def setUp(self) -> None:
        self.roles = load_yaml(".ai/roles/registry.yaml")
        self.delegation = load_yaml(".ai/policies/delegation.yaml")
        self.agents = load_yaml(".ai/agents/registry.yaml")
        self.permissions = load_yaml(".ai/policies/permissions.yaml")
        self.routing = load_yaml(".ai/policies/provider-routing.yaml")
        self.assignment = load_yaml(".ai/examples/agent-assignment.example.yaml")
        self.session = load_yaml(".ai/examples/agent-session.example.yaml")
        self.lease = load_yaml(".ai/examples/write-lease.example.yaml")
        self.session_schema = load_json(".ai/schemas/agent-session.schema.json")
        self.write_lease_schema = load_json(".ai/schemas/write-lease.schema.json")

    def assert_contract(self, **replacements: dict[str, Any]) -> None:
        assert_declared_delegation_contract(
            replacements.get("roles", self.roles),
            replacements.get("delegation", self.delegation),
            replacements.get("agents", self.agents),
            replacements.get("permissions", self.permissions),
        )

    def make_active_lease(
        self, lease_id: str, assignment_id: str, session_id: str, allowed_path: str
    ) -> dict[str, Any]:
        lease = copy.deepcopy(self.lease)
        lease.update(
            {
                "lease_id": lease_id,
                "assignment_id": assignment_id,
                "agent_session_id": session_id,
                "allowed_paths": [allowed_path],
                "activated_at": "2026-08-30T19:00:00+00:00",
                "expires_at": "2026-08-30T20:00:00+00:00",
                "state": "ACTIVE",
            }
        )
        return lease

    def assert_lease_schema_valid(self, lease: dict[str, Any]) -> None:
        errors = list(Draft202012Validator(self.write_lease_schema).iter_errors(lease))
        self.assertEqual(errors, [], "\n".join(error.message for error in errors))

    def test_current_declared_delegation_contract_passes(self) -> None:
        self.assert_contract()
        assert_assignment_profile_parent(self.assignment, self.agents)
        assert_no_same_assignment_self_review(self.assignment)
        assert_no_unapproved_fallback(self.routing)

    def test_workers_may_spawn_true_is_rejected(self) -> None:
        hostile = copy.deepcopy(self.roles)
        hostile["global_controls"]["workers_may_spawn"] = True
        with self.assertRaisesRegex(AssertionError, "workers_may_spawn"):
            self.assert_contract(roles=hostile)
        self.assertIs(self.roles["global_controls"]["workers_may_spawn"], False)

    def test_provider_native_hidden_subagents_allowed_is_rejected(self) -> None:
        cases = (
            ("roles", self.roles, ("global_controls", "provider_native_hidden_subagents")),
            ("delegation", self.delegation, ("provider_native_hidden_subagents",)),
        )
        for document, original, key_path in cases:
            with self.subTest(document=document):
                hostile = copy.deepcopy(original)
                target = hostile
                for key in key_path[:-1]:
                    target = target[key]
                target[key_path[-1]] = "ALLOWED"
                with self.assertRaisesRegex(AssertionError, "provider_native_hidden_subagents"):
                    self.assert_contract(**{document: hostile})

    def test_hidden_delegation_allowed_is_rejected(self) -> None:
        hostile = copy.deepcopy(self.delegation)
        hostile["hidden_delegation"] = "ALLOWED"
        with self.assertRaisesRegex(AssertionError, "hidden_delegation"):
            self.assert_contract(delegation=hostile)

    def test_unregistered_sessions_allowed_is_rejected(self) -> None:
        cases = (
            ("roles", self.roles, ("global_controls", "unregistered_sessions")),
            ("delegation", self.delegation, ("unregistered_sessions",)),
        )
        for document, original, key_path in cases:
            with self.subTest(document=document):
                hostile = copy.deepcopy(original)
                target = hostile
                for key in key_path[:-1]:
                    target = target[key]
                target[key_path[-1]] = "ALLOWED"
                with self.assertRaisesRegex(AssertionError, "unregistered_sessions"):
                    self.assert_contract(**{document: hostile})

    def test_specialist_profiles_may_spawn_true_is_rejected(self) -> None:
        hostile = copy.deepcopy(self.delegation)
        hostile["specialist_profiles_may_spawn"] = True
        with self.assertRaisesRegex(AssertionError, "specialist_profiles_may_spawn"):
            self.assert_contract(delegation=hostile)

    def test_visible_delegation_depth_mismatch_is_rejected(self) -> None:
        hostile = copy.deepcopy(self.delegation)
        hostile["maximum_visible_delegation_depth"] = 3
        with self.assertRaisesRegex(AssertionError, "maximum_visible_delegation_depth"):
            self.assert_contract(delegation=hostile)

    def test_required_delegation_guard_removal_is_rejected(self) -> None:
        for guard in sorted(REQUIRED_DELEGATED_TASK_GUARDS):
            with self.subTest(guard=guard):
                hostile = copy.deepcopy(self.delegation)
                hostile["delegated_task_requires"].remove(guard)
                with self.assertRaisesRegex(AssertionError, guard):
                    self.assert_contract(delegation=hostile)

    def test_specialist_registry_guard_removal_is_rejected(self) -> None:
        for guard in sorted(REQUIRED_SPECIALIST_RULES):
            with self.subTest(guard=guard):
                hostile = copy.deepcopy(self.agents)
                hostile["rules"].remove(guard)
                with self.assertRaisesRegex(AssertionError, guard):
                    self.assert_contract(agents=hostile)

    def test_permissions_hidden_spawn_prohibition_removal_is_rejected(self) -> None:
        hostile = copy.deepcopy(self.permissions)
        hostile["global_agent_prohibitions"].remove("spawn-hidden-subagent")
        with self.assertRaisesRegex(AssertionError, "spawn-hidden-subagent"):
            self.assert_contract(permissions=hostile)

    def test_session_schema_rejects_hidden_delegation_true(self) -> None:
        hostile = copy.deepcopy(self.session)
        hostile["hidden_delegation"] = True
        errors = list(Draft202012Validator(self.session_schema).iter_errors(hostile))
        self.assertEqual([list(error.absolute_path) for error in errors], [["hidden_delegation"]])

    def test_session_schema_rejects_invisible_session(self) -> None:
        hostile = copy.deepcopy(self.session)
        hostile["visible_to_control_plane"] = False
        errors = list(Draft202012Validator(self.session_schema).iter_errors(hostile))
        self.assertEqual([list(error.absolute_path) for error in errors], [["visible_to_control_plane"]])

    def test_specialist_profile_parent_role_mismatch_is_rejected(self) -> None:
        assert_assignment_profile_parent(self.assignment, self.agents)
        hostile = copy.deepcopy(self.assignment)
        hostile["role"]["role_id"] = "engineering-agent"
        with self.assertRaisesRegex(AssertionError, "specialist_profile.parent_roles"):
            assert_assignment_profile_parent(hostile, self.agents)

    def test_same_assignment_self_review_is_rejected(self) -> None:
        hostile = copy.deepcopy(self.assignment)
        hostile["review"]["reviewer_assignment_id"] = hostile["assignment_id"]
        with self.assertRaisesRegex(AssertionError, "reviewer_assignment_id"):
            assert_no_same_assignment_self_review(hostile)

    def test_unapproved_fallback_route_is_rejected(self) -> None:
        assert_no_unapproved_fallback(self.routing)
        hostile = copy.deepcopy(self.routing)
        hostile["role_routes"]["engineering-agent"]["fallback_route_ids"] = [
            "route-openai-codex-candidate"
        ]
        with self.assertRaisesRegex(AssertionError, "engineering-agent.fallback_route_ids"):
            assert_no_unapproved_fallback(hostile)

    def test_exact_duplicate_active_write_path_is_rejected(self) -> None:
        leases = [
            self.make_active_lease("LEASE-I014-TEST-A", "ASN-I014-TEST-A", "SESSION-I014-TEST-A", "tests/**"),
            self.make_active_lease("LEASE-I014-TEST-B", "ASN-I014-TEST-B", "SESSION-I014-TEST-B", "tests/**"),
        ]
        for lease in leases:
            self.assert_lease_schema_valid(lease)
        with self.assertRaisesRegex(AssertionError, "exact active collision"):
            assert_no_exact_active_write_path_collisions(leases)

    def test_distinct_literal_active_write_paths_are_not_false_positives(self) -> None:
        leases = [
            self.make_active_lease("LEASE-I014-TEST-A", "ASN-I014-TEST-A", "SESSION-I014-TEST-A", "tests/unit/**"),
            self.make_active_lease(
                "LEASE-I014-TEST-B",
                "ASN-I014-TEST-B",
                "SESSION-I014-TEST-B",
                "tests/integration/**",
            ),
        ]
        for lease in leases:
            self.assert_lease_schema_valid(lease)
        assert_no_exact_active_write_path_collisions(leases)


if __name__ == "__main__":
    unittest.main()
