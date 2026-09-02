from __future__ import annotations

import copy
import json
import re
import unittest
from datetime import datetime
from pathlib import Path
from typing import Any

import yaml
from jsonschema import Draft202012Validator, FormatChecker


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

REQUIRED_ROLE_CARD_FIELDS = (
    "purpose",
    "authority",
    "prohibited_actions",
    "inputs",
    "outputs",
    "escalation",
)

ROLE_CARD_FIELD_SECTIONS = {
    "purpose": ("Purpose",),
    "authority": ("Allowed Authority",),
    "prohibited_actions": ("Prohibited Actions",),
    "inputs": ("Required Inputs",),
    "outputs": ("Required Outputs",),
    "escalation": ("Human Approval Triggers", "Stop Conditions"),
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


def load_role_card(relative_path: Path) -> tuple[dict[str, Any], dict[str, str]]:
    text = relative_path.read_text(encoding="utf-8")
    if not text.startswith("---\n"):
        raise AssertionError(f"{relative_path}: missing YAML front matter")
    _, front_matter, body = text.split("---", 2)
    metadata = yaml.safe_load(front_matter)
    if not isinstance(metadata, dict):
        raise AssertionError(f"{relative_path}: role-card metadata must be a YAML object")

    headings = list(re.finditer(r"^##\s+(.+?)\s*$", body, re.MULTILINE))
    sections: dict[str, str] = {}
    for index, heading in enumerate(headings):
        start = heading.end()
        end = headings[index + 1].start() if index + 1 < len(headings) else len(body)
        sections[heading.group(1).strip()] = body[start:end].strip()
    return metadata, sections


def role_card_paths() -> list[Path]:
    cards_dir = REPO_ROOT / ".ai" / "roles" / "cards"
    return sorted(path for path in cards_dir.glob("*.md") if path.name != "index.md")


def schema_errors(document: Any, schema: dict[str, Any]) -> list[str]:
    return [
        error.message
        for error in Draft202012Validator(schema, format_checker=FormatChecker()).iter_errors(document)
    ]


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


def _glob_segment_tokens(segment: str) -> list[tuple[str, Any]]:
    tokens: list[tuple[str, Any]] = []
    index = 0
    while index < len(segment):
        char = segment[index]
        if char == "*":
            if not tokens or tokens[-1][0] != "star":
                tokens.append(("star", None))
            index += 1
        elif char == "?":
            tokens.append(("any", None))
            index += 1
        elif char == "[":
            closing = segment.find("]", index + 1)
            if closing == -1:
                tokens.append(("literal", char))
                index += 1
                continue
            expression = segment[index + 1 : closing]
            negated = expression.startswith(("!", "^"))
            if negated:
                expression = expression[1:]
            literals: set[str] = set()
            ranges: list[tuple[str, str]] = []
            cursor = 0
            while cursor < len(expression):
                if cursor + 2 < len(expression) and expression[cursor + 1] == "-":
                    ranges.append((expression[cursor], expression[cursor + 2]))
                    cursor += 3
                else:
                    literals.add(expression[cursor])
                    cursor += 1
            tokens.append(("class", (negated, literals, ranges)))
            index = closing + 1
        else:
            tokens.append(("literal", char))
            index += 1
    return tokens


def _class_matches(spec: tuple[bool, set[str], list[tuple[str, str]]], char: str) -> bool:
    negated, literals, ranges = spec
    matched = char in literals or any(start <= char <= end for start, end in ranges)
    return not matched if negated else matched


def _token_matches_char(token: tuple[str, Any], char: str) -> bool:
    kind, value = token
    if kind == "any":
        return True
    if kind == "literal":
        return value == char
    if kind == "class":
        return _class_matches(value, char)
    return False


def _epsilon_closure(tokens: list[tuple[str, Any]], states: set[int]) -> set[int]:
    closure = set(states)
    stack = list(states)
    while stack:
        index = stack.pop()
        if index < len(tokens) and tokens[index][0] == "star":
            following = index + 1
            if following not in closure:
                closure.add(following)
                stack.append(following)
    return closure


def _advance(tokens: list[tuple[str, Any]], states: set[int], char: str) -> set[int]:
    following: set[int] = set()
    for index in states:
        if index >= len(tokens):
            continue
        if tokens[index][0] == "star":
            following.add(index)
        if _token_matches_char(tokens[index], char):
            following.add(index + 1)
    return following


def _candidate_characters(
    left_tokens: list[tuple[str, Any]], right_tokens: list[tuple[str, Any]]
) -> list[str]:
    characters: set[str] = set()
    for kind, value in left_tokens + right_tokens:
        if kind == "literal":
            characters.add(value)
        elif kind == "class":
            _, literals, ranges = value
            characters.update(literals)
            for start, end in ranges:
                characters.add(start)
                characters.add(end)
    for filler in "0123456789":
        if filler not in characters:
            characters.add(filler)
            break
    return sorted(characters)


def _glob_segment_intersects(left: str, right: str) -> bool:
    left_tokens = _glob_segment_tokens(left)
    right_tokens = _glob_segment_tokens(right)
    left_accept = len(left_tokens)
    right_accept = len(right_tokens)
    characters = _candidate_characters(left_tokens, right_tokens)

    start = (
        frozenset(_epsilon_closure(left_tokens, {0})),
        frozenset(_epsilon_closure(right_tokens, {0})),
    )
    if left_accept in start[0] and right_accept in start[1]:
        return True

    pending = [start]
    visited = {start}
    while pending:
        left_states, right_states = pending.pop()
        for char in characters:
            next_left = frozenset(_epsilon_closure(left_tokens, _advance(left_tokens, left_states, char)))
            next_right = frozenset(_epsilon_closure(right_tokens, _advance(right_tokens, right_states, char)))
            if left_accept in next_left and right_accept in next_right:
                return True
            state = (next_left, next_right)
            if state not in visited:
                visited.add(state)
                pending.append(state)
    return False


def _scope_segments(value: str) -> tuple[str, ...]:
    normalized = normalize_literal_path(value).strip("/")
    return tuple(segment for segment in normalized.split("/") if segment)


def _scope_patterns_overlap(left: str, right: str) -> bool:
    left_segments = _scope_segments(left)
    right_segments = _scope_segments(right)
    pending = [(0, 0)]
    visited: set[tuple[int, int]] = set()
    while pending:
        left_index, right_index = pending.pop()
        state = (left_index, right_index)
        if state in visited:
            continue
        visited.add(state)
        if left_index == len(left_segments) and right_index == len(right_segments):
            return True
        left_globstar = left_index < len(left_segments) and left_segments[left_index] == "**"
        right_globstar = right_index < len(right_segments) and right_segments[right_index] == "**"
        if left_globstar:
            pending.append((left_index + 1, right_index))
            if right_index < len(right_segments) and not right_globstar:
                if _glob_segment_intersects("*", right_segments[right_index]):
                    pending.append((left_index, right_index + 1))
            elif right_globstar:
                pending.append((left_index, right_index + 1))
            continue
        if right_globstar:
            pending.append((left_index, right_index + 1))
            if left_index < len(left_segments) and _glob_segment_intersects(left_segments[left_index], "*"):
                pending.append((left_index + 1, right_index))
            continue
        if left_index < len(left_segments) and right_index < len(right_segments):
            if _glob_segment_intersects(left_segments[left_index], right_segments[right_index]):
                pending.append((left_index + 1, right_index + 1))
    return False


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
            if not intervals_overlap or not same_lease_domain:
                continue
            collisions = [
                (normalize_literal_path(left_path), normalize_literal_path(right_path))
                for left_path in left.get("allowed_paths") or []
                for right_path in right.get("allowed_paths") or []
                if _scope_patterns_overlap(left_path, right_path)
            ]
            if collisions:
                left_path, right_path = sorted(collisions)[0]
                collision_kind = "exact" if left_path == right_path else "overlapping"
                first_id, second_id = sorted((left_id, right_id))
                raise AssertionError(
                    f"allowed_paths: {collision_kind} active collision for "
                    f"{left_path!r} with {right_path!r} between {first_id} and {second_id}"
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
        self.assignment_schema = load_json(".ai/schemas/agent-assignment.schema.json")
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
        errors = schema_errors(lease, self.write_lease_schema)
        self.assertEqual(errors, [], "\n".join(errors))

    def test_current_role_cards_validate_with_required_contract_fields(self) -> None:
        role_card_schema = load_json(".ai/schemas/role-card.schema.json")
        cards = role_card_paths()
        self.assertTrue(cards, "no current role cards discovered")
        for card_path in cards:
            with self.subTest(role_card=card_path.name):
                metadata, sections = load_role_card(card_path)
                self.assertEqual(schema_errors(metadata, role_card_schema), [])
                self.assertEqual(metadata.get("role_id"), card_path.stem)
                for field in REQUIRED_ROLE_CARD_FIELDS:
                    required_sections = ROLE_CARD_FIELD_SECTIONS[field]
                    self.assertTrue(
                        all(sections.get(section, "").strip() for section in required_sections),
                        f"{card_path}: missing required role-card field {field}",
                    )

    def test_current_assignment_session_and_lease_examples_validate(self) -> None:
        examples = (
            ("assignment", self.assignment, self.assignment_schema),
            ("session", self.session, self.session_schema),
            ("write lease", self.lease, self.write_lease_schema),
        )
        for name, document, schema in examples:
            with self.subTest(example=name):
                errors = schema_errors(document, schema)
                self.assertEqual(errors, [], "\n".join(errors))

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

    def test_nested_glob_active_write_paths_are_rejected_in_both_orders(self) -> None:
        pairs = (
            ("tests/**", "tests/unit/**"),
            ("tests/*", "tests/unit/**"),
        )
        for left_path, right_path in pairs:
            for paths in ((left_path, right_path), (right_path, left_path)):
                with self.subTest(paths=paths):
                    leases = [
                        self.make_active_lease(
                            "LEASE-I014-TEST-A", "ASN-I014-TEST-A", "SESSION-I014-TEST-A", paths[0]
                        ),
                        self.make_active_lease(
                            "LEASE-I014-TEST-B", "ASN-I014-TEST-B", "SESSION-I014-TEST-B", paths[1]
                        ),
                    ]
                    for lease in leases:
                        self.assert_lease_schema_valid(lease)
                    with self.assertRaisesRegex(AssertionError, "overlapping active collision"):
                        assert_no_exact_active_write_path_collisions(leases)
    def test_intersecting_glob_active_write_paths_are_rejected_in_both_orders(self) -> None:
        pairs = (
            ("tests/a*", "tests/*b"),
            ("tests/[ab]*", "tests/[bc]*"),
            ("tests/a?", "tests/?b"),
        )
        for left_path, right_path in pairs:
            for paths in ((left_path, right_path), (right_path, left_path)):
                with self.subTest(paths=paths):
                    leases = [
                        self.make_active_lease(
                            "LEASE-I014-TEST-A", "ASN-I014-TEST-A", "SESSION-I014-TEST-A", paths[0]
                        ),
                        self.make_active_lease(
                            "LEASE-I014-TEST-B", "ASN-I014-TEST-B", "SESSION-I014-TEST-B", paths[1]
                        ),
                    ]
                    for lease in leases:
                        self.assert_lease_schema_valid(lease)
                    with self.assertRaisesRegex(AssertionError, "overlapping active collision"):
                        assert_no_exact_active_write_path_collisions(leases)

    def test_assignment_schema_requires_each_required_field(self) -> None:
        for field in self.assignment_schema["required"]:
            with self.subTest(field=field):
                hostile = copy.deepcopy(self.assignment)
                hostile.pop(field, None)
                self.assertTrue(
                    schema_errors(hostile, self.assignment_schema),
                    f"agent-assignment schema should require {field}",
                )

    def test_session_schema_requires_each_required_field(self) -> None:
        for field in self.session_schema["required"]:
            with self.subTest(field=field):
                hostile = copy.deepcopy(self.session)
                hostile.pop(field, None)
                self.assertTrue(
                    schema_errors(hostile, self.session_schema),
                    f"agent-session schema should require {field}",
                )

    def test_write_lease_schema_requires_each_required_field(self) -> None:
        for field in self.write_lease_schema["required"]:
            with self.subTest(field=field):
                hostile = copy.deepcopy(self.lease)
                hostile.pop(field, None)
                self.assertTrue(
                    schema_errors(hostile, self.write_lease_schema),
                    f"write-lease schema should require {field}",
                )

    def test_role_card_schema_requires_each_required_field(self) -> None:
        role_card_schema = load_json(".ai/schemas/role-card.schema.json")
        cards = role_card_paths()
        self.assertTrue(cards, "no current role cards discovered")
        metadata, _ = load_role_card(cards[0])
        for field in role_card_schema["required"]:
            with self.subTest(field=field):
                hostile = copy.deepcopy(metadata)
                hostile.pop(field, None)
                self.assertTrue(
                    schema_errors(hostile, role_card_schema),
                    f"role-card schema should require {field}",
                )

    def test_role_registry_entries_require_role_id_and_resolve_role_cards(self) -> None:
        entries = self.roles.get("roles") or []
        self.assertTrue(entries, "no role registry entries discovered")
        for entry in entries:
            with self.subTest(role_id=entry.get("role_id")):
                role_id = entry.get("role_id")
                self.assertTrue(role_id, "role entry missing role_id")
                card_ref = entry.get("role_card")
                self.assertTrue(card_ref, f"{role_id}: role entry missing role_card")
                card_path = REPO_ROOT / card_ref
                self.assertTrue(card_path.is_file(), f"{role_id}: role_card not found: {card_ref}")
                metadata, _ = load_role_card(card_path)
                self.assertEqual(metadata.get("role_id"), role_id)

    def test_specialist_registry_profiles_require_required_fields(self) -> None:
        profiles = self.agents.get("profiles") or []
        self.assertTrue(profiles, "no specialist profiles discovered")
        for profile in profiles:
            with self.subTest(profile_id=profile.get("profile_id")):
                self.assertTrue(profile.get("profile_id"), "profile missing profile_id")
                self.assertTrue(
                    profile.get("default_tool_profile"), "profile missing default_tool_profile"
                )
                self.assertTrue(profile.get("parent_roles"), "profile missing parent_roles")


if __name__ == "__main__":
    unittest.main()
