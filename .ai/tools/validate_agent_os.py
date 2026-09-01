#!/usr/bin/env python3
"""Validate the repository-ready OSHE AI Agent OS control package."""

from __future__ import annotations

import json
import re
import sys
import unicodedata
import urllib.parse
from pathlib import Path
from typing import Any

import yaml
from jsonschema import Draft202012Validator, FormatChecker


REPO_ROOT = Path(__file__).resolve().parents[2]
AI_ROOT = REPO_ROOT / ".ai"
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

EXPECTED_ROLES = {
    "project-management-agent",
    "product-planning-agent",
    "architecture-agent",
    "engineering-agent",
    "data-integration-agent",
    "security-privacy-product-safety-agent",
    "test-quality-agent",
    "documentation-configuration-agent",
    "research-legal-content-agent",
    "independent-review-challenge-agent",
    "release-evidence-agent",
    "implementation-customer-success-planning-agent",
}

EXPECTED_MINIMUM_COUNTS = {
    "roles": 12,
    "profiles": 17,
    "skills": 41,
    "policies_including_registry": 18,
    "runbooks": 15,
    "schemas": 20,
}

REQUIRED_SKILL_SECTIONS = (
    "## Objective",
    "## Required Inputs",
    "## Procedure",
    "## Required Output",
    "## Stop Conditions",
    "## Evaluation Cases",
)

REQUIRED_LAYOUT = (
    "AGENTS.md",
    "CLAUDE.md",
    "GEMINI.md",
    "QWEN.md",
    ".ai/README.md",
    ".ai/context/prompt-envelope.md",
    ".ai/roles/registry.yaml",
    ".ai/agents/registry.yaml",
    ".ai/bundles/role-bundles.yaml",
    ".ai/skill-registry.yaml",
    ".ai/policies/policy-registry.yaml",
    ".ai/provider-routes/ai-service-route-registry.yaml",
    ".ai/readiness.yaml",
    ".ai/preparation-handoff.md",
    ".ai/policies/github-operations.yaml",
    ".ai/policies/github-credential-profiles.yaml",
    ".ai/policies/repository-workflow-and-ci.yaml",
    ".ai/schemas/github-operation-gate.schema.json",
    ".ai/tools/evaluate_github_operation.py",
    ".ai/tools/execute_github_operation.py",
    ".ci/local-ci.json",
    "tools/run_local_ci.py",
    "tests/test_local_ci.py",
    "docs/adr/adr-0006-evidence-gated-full-github-operator-authority.md",
    "docs/adr/adr-0007-local-first-ci-and-repository-lifecycle.md",
)


def normalize_action(value: str) -> str:
    normalized = unicodedata.normalize("NFKC", value).casefold().strip()
    return ACTION_SEPARATOR_PATTERN.sub("-", normalized).strip("-")


class Validation:
    def __init__(self) -> None:
        self.errors: list[str] = []
        self.checked_files: set[Path] = set()

    def error(self, message: str) -> None:
        self.errors.append(message)

    def load_yaml(self, path: Path) -> Any:
        self.checked_files.add(path)
        try:
            with path.open("r", encoding="utf-8-sig") as stream:
                return yaml.safe_load(stream)
        except Exception as exc:  # pragma: no cover - diagnostic boundary
            self.error(f"YAML parse failed: {path.relative_to(REPO_ROOT)}: {exc}")
            return None

    def load_json(self, path: Path) -> Any:
        self.checked_files.add(path)
        try:
            with path.open("r", encoding="utf-8-sig") as stream:
                return json.load(stream)
        except Exception as exc:  # pragma: no cover - diagnostic boundary
            self.error(f"JSON parse failed: {path.relative_to(REPO_ROOT)}: {exc}")
            return None

    def frontmatter(self, path: Path) -> dict[str, Any]:
        self.checked_files.add(path)
        try:
            text = path.read_text(encoding="utf-8-sig")
        except Exception as exc:
            self.error(f"Markdown read failed: {path.relative_to(REPO_ROOT)}: {exc}")
            return {}
        lines = text.splitlines()
        if not lines or lines[0].strip() != "---":
            self.error(f"Missing YAML frontmatter: {path.relative_to(REPO_ROOT)}")
            return {}
        try:
            end = lines.index("---", 1)
            data = yaml.safe_load("\n".join(lines[1:end]))
        except Exception as exc:
            self.error(f"Frontmatter parse failed: {path.relative_to(REPO_ROOT)}: {exc}")
            return {}
        if not isinstance(data, dict):
            self.error(f"Frontmatter is not an object: {path.relative_to(REPO_ROOT)}")
            return {}
        return data

    def resolve_repo_path(self, value: str, context: str) -> Path | None:
        candidate = REPO_ROOT / value
        try:
            candidate.resolve().relative_to(REPO_ROOT.resolve())
        except ValueError:
            self.error(f"Reference escapes repository in {context}: {value}")
            return None
        if not candidate.exists():
            self.error(f"Missing referenced path in {context}: {value}")
            return None
        return candidate


def unique_values(validation: Validation, values: list[str], context: str) -> set[str]:
    result = set(values)
    if len(result) != len(values):
        validation.error(f"Duplicate values in {context}")
    return result


def validate_required_layout(validation: Validation) -> None:
    for value in REQUIRED_LAYOUT:
        if not (REPO_ROOT / value).is_file():
            validation.error(f"Missing required AI Agent OS path: {value}")
    for provider in ("codex", "claude", "gemini", "deepseek", "qwen"):
        if not (AI_ROOT / "provider-notes" / f"{provider}.md").is_file():
            validation.error(f"Missing provider note: {provider}.md")


def validate_parse_all(validation: Validation) -> None:
    for path in sorted(AI_ROOT.rglob("*")):
        if not path.is_file():
            continue
        if any(part in {"runs", "state", "transcripts"} for part in path.parts):
            validation.error(f"Runtime artifact is present under tracked controls: {path.relative_to(REPO_ROOT)}")
        if path.suffix.lower() in {".yaml", ".yml"}:
            validation.load_yaml(path)
        elif path.suffix.lower() == ".json":
            validation.load_json(path)


def validate_local_markdown_links(validation: Validation) -> None:
    link_pattern = re.compile(r"\[[^\]]*\]\(([^)]+)\)")
    for path in sorted(AI_ROOT.rglob("*.md")):
        text = path.read_text(encoding="utf-8-sig")
        for match in link_pattern.finditer(text):
            target = match.group(1).split("#", 1)[0]
            if not target or "://" in target or target.startswith("mailto:"):
                continue
            resolved = (path.parent / urllib.parse.unquote(target)).resolve()
            try:
                resolved.relative_to(REPO_ROOT.resolve())
            except ValueError:
                validation.error(f"Markdown link escapes repository: {path.relative_to(REPO_ROOT)} -> {target}")
                continue
            if not resolved.exists():
                validation.error(f"Broken local Markdown link: {path.relative_to(REPO_ROOT)} -> {target}")


def validate_json_schemas(validation: Validation) -> None:
    schemas = sorted((AI_ROOT / "schemas").glob("*.schema.json"))
    if len(schemas) < EXPECTED_MINIMUM_COUNTS["schemas"]:
        validation.error(f"Expected at least {EXPECTED_MINIMUM_COUNTS['schemas']} schemas, found {len(schemas)}")
    for path in schemas:
        schema = validation.load_json(path)
        if schema is None:
            continue
        try:
            Draft202012Validator.check_schema(schema)
        except Exception as exc:
            validation.error(f"Invalid JSON Schema {path.name}: {exc}")


def validate_roles(validation: Validation) -> tuple[set[str], dict[str, Any]]:
    registry = validation.load_yaml(AI_ROOT / "roles" / "registry.yaml") or {}
    roles = registry.get("roles", [])
    role_ids = unique_values(validation, [item.get("role_id") for item in roles if isinstance(item, dict)], "role registry")
    if role_ids != EXPECTED_ROLES:
        validation.error(f"Canonical role set mismatch: expected {sorted(EXPECTED_ROLES)}, found {sorted(role_ids)}")
    if len(roles) != EXPECTED_MINIMUM_COUNTS["roles"]:
        validation.error(f"Expected 12 canonical role entries, found {len(roles)}")

    role_schema = validation.load_json(AI_ROOT / "schemas" / "role-card.schema.json")
    validator = Draft202012Validator(role_schema, format_checker=FormatChecker()) if role_schema else None
    for item in roles:
        if not isinstance(item, dict):
            validation.error("Non-object role entry")
            continue
        role_id = item.get("role_id", "<missing>")
        path = validation.resolve_repo_path(str(item.get("role_card", "")), f"role {role_id}")
        if path is None:
            continue
        metadata = validation.frontmatter(path)
        if metadata.get("role_id") != role_id:
            validation.error(f"Role-card ID mismatch for {role_id}: {path.relative_to(REPO_ROOT)}")
        if validator:
            for error in sorted(validator.iter_errors(metadata), key=lambda item: list(item.path)):
                validation.error(f"Role-card schema failure {role_id}: {error.message}")
        if item.get("skill_bundle_id") != role_id:
            validation.error(f"Role {role_id} must resolve to its canonical bundle ID")

    if registry.get("global_controls", {}).get("provider_native_hidden_subagents") != "PROHIBITED":
        validation.error("Role registry must prohibit provider-native hidden subagents")
    if registry.get("global_controls", {}).get("unregistered_sessions") != "PROHIBITED":
        validation.error("Role registry must prohibit unregistered sessions")
    if registry.get("global_controls", {}).get("github_execution") != "STANDING_CONDITIONAL_FULL_AUTHORITY_ADR_0006":
        validation.error("Role registry does not activate ADR-0006 standing conditional GitHub authority")
    return role_ids, registry


def validate_skills_and_bundles(validation: Validation, role_ids: set[str]) -> set[str]:
    registry = validation.load_yaml(AI_ROOT / "skill-registry.yaml") or {}
    skills = registry.get("skills", [])
    skill_ids = unique_values(validation, [item.get("name") for item in skills if isinstance(item, dict)], "skill registry")
    if len(skills) < EXPECTED_MINIMUM_COUNTS["skills"]:
        validation.error(f"Expected at least {EXPECTED_MINIMUM_COUNTS['skills']} skills, found {len(skills)}")
    for item in skills:
        if not isinstance(item, dict):
            validation.error("Non-object skill entry")
            continue
        skill_id = item.get("name", "<missing>")
        source = validation.resolve_repo_path(str(item.get("source", "")), f"skill {skill_id}")
        if source is None:
            continue
        skill_file = source / "SKILL.md"
        if not skill_file.is_file():
            validation.error(f"Missing SKILL.md for {skill_id}")
            continue
        metadata = validation.frontmatter(skill_file)
        if metadata.get("name") != skill_id:
            validation.error(f"Skill metadata name mismatch for {skill_id}")
        text = skill_file.read_text(encoding="utf-8-sig")
        for heading in REQUIRED_SKILL_SECTIONS:
            if heading not in text:
                validation.error(f"Skill {skill_id} is missing section {heading}")

    bundles_doc = validation.load_yaml(AI_ROOT / "bundles" / "role-bundles.yaml") or {}
    bundles = bundles_doc.get("bundles", {})
    if set(bundles) != role_ids:
        validation.error("Role bundle keys must exactly match canonical role IDs")
    for role_id, bundle in bundles.items():
        if not isinstance(bundle, list) or not bundle:
            validation.error(f"Role bundle is empty or invalid: {role_id}")
            continue
        unique_values(validation, bundle, f"bundle {role_id}")
        unknown = set(bundle) - skill_ids
        if unknown:
            validation.error(f"Bundle {role_id} references unknown skills: {sorted(unknown)}")
    return skill_ids


def validate_profiles_and_permissions(
    validation: Validation, role_ids: set[str], skill_ids: set[str]
) -> None:
    tools_doc = validation.load_yaml(AI_ROOT / "policies" / "tool-profiles.yaml") or {}
    tool_profiles = set((tools_doc.get("profiles") or {}).keys())
    profiles_doc = validation.load_yaml(AI_ROOT / "agents" / "registry.yaml") or {}
    profiles = profiles_doc.get("profiles", [])
    profile_ids = unique_values(validation, [item.get("profile_id") for item in profiles if isinstance(item, dict)], "specialist profile registry")
    if len(profiles) < EXPECTED_MINIMUM_COUNTS["profiles"]:
        validation.error(f"Expected at least {EXPECTED_MINIMUM_COUNTS['profiles']} specialist profiles, found {len(profiles)}")
    profile_schema = validation.load_json(AI_ROOT / "schemas" / "specialist-profile.schema.json")
    profile_validator = Draft202012Validator(profile_schema, format_checker=FormatChecker()) if profile_schema else None
    for item in profiles:
        if not isinstance(item, dict):
            validation.error("Non-object specialist profile entry")
            continue
        profile_id = item.get("profile_id", "<missing>")
        path = validation.resolve_repo_path(str(item.get("path", "")), f"specialist profile {profile_id}")
        if path:
            metadata = validation.frontmatter(path)
            if metadata.get("profile_id") != profile_id:
                validation.error(f"Specialist profile metadata mismatch for {profile_id}")
            if profile_validator:
                for error in sorted(profile_validator.iter_errors(metadata), key=lambda value: list(value.path)):
                    validation.error(f"Specialist profile schema failure {profile_id}: {error.message}")
        unknown_roles = set(item.get("parent_roles") or []) - role_ids
        if unknown_roles:
            validation.error(f"Profile {profile_id} references unknown roles: {sorted(unknown_roles)}")
        unknown_skills = set(item.get("required_skills") or []) - skill_ids
        if unknown_skills:
            validation.error(f"Profile {profile_id} references unknown skills: {sorted(unknown_skills)}")
        if item.get("default_tool_profile") not in tool_profiles:
            validation.error(f"Profile {profile_id} references an unknown tool profile")

    permissions = validation.load_yaml(AI_ROOT / "policies" / "permissions.yaml") or {}
    permission_roles = set((permissions.get("roles") or {}).keys())
    if permission_roles != role_ids:
        validation.error("Permission role keys must exactly match canonical role IDs")
    if permissions.get("default_policy") != "DENY":
        validation.error("Permission default policy must be DENY")
    for role_id, envelope in (permissions.get("roles") or {}).items():
        if envelope.get("default_tool_profile") not in tool_profiles:
            validation.error(f"Permission envelope {role_id} references an unknown tool profile")
    if not profile_ids:
        validation.error("Specialist profile registry is empty")


def validate_policies(validation: Validation) -> None:
    policy_files = sorted((AI_ROOT / "policies").glob("*.yaml"))
    if len(policy_files) < EXPECTED_MINIMUM_COUNTS["policies_including_registry"]:
        validation.error(f"Expected at least {EXPECTED_MINIMUM_COUNTS['policies_including_registry']} policy files, found {len(policy_files)}")
    registry = validation.load_yaml(AI_ROOT / "policies" / "policy-registry.yaml") or {}
    entries = registry.get("policies", [])
    unique_values(validation, [item.get("id") for item in entries if isinstance(item, dict)], "policy registry")
    for item in entries:
        if isinstance(item, dict):
            validation.resolve_repo_path(str(item.get("path", "")), f"policy {item.get('id')}")


def validate_github_authority(validation: Validation) -> None:
    policy = validation.load_yaml(AI_ROOT / "policies" / "github-operations.yaml") or {}
    permissions = validation.load_yaml(AI_ROOT / "policies" / "permissions.yaml") or {}
    tools = validation.load_yaml(AI_ROOT / "policies" / "tool-profiles.yaml") or {}
    profiles = validation.load_yaml(AI_ROOT / "agents" / "registry.yaml") or {}
    credentials = validation.load_yaml(AI_ROOT / "policies" / "github-credential-profiles.yaml") or {}

    if policy.get("lifecycle_status") != "APPROVED":
        validation.error("GitHub operations policy must record the Sole Human Owner approval")
    if policy.get("authorization_mode") != "STANDING_CONDITIONAL_AUTHORIZATION":
        validation.error("GitHub authority must use standing conditional authorization")
    if policy.get("authorized_role_id") != "release-evidence-agent" or policy.get("required_specialist_profile_id") != "github-manager":
        validation.error("Full GitHub authority must bind release-evidence-agent to github-manager")
    if policy.get("default_write_decision") != "DENY":
        validation.error("GitHub operations must deny writes by default")
    if policy.get("organization_allowlist") != ["OSHEThai"]:
        validation.error("GitHub organization allowlist must be exactly OSHEThai")

    if policy.get("prohibited_actions") != ["repository-delete"]:
        validation.error("GitHub prohibited actions must be exactly repository-delete")
    allowed_examples = {
        normalize_action(str(example))
        for action_class in (policy.get("action_classes") or {}).values()
        if isinstance(action_class, dict)
        for example in (action_class.get("examples") or [])
    }
    if ALWAYS_PROHIBITED_ACTION_ALIASES & allowed_examples:
        validation.error("repository-delete must not appear in allowed GitHub action examples")

    required_high_impact = {"MERGE", "RELEASE", "REPOSITORY_ADMIN", "CREDENTIAL", "SECURITY", "DESTRUCTIVE", "DEPLOYMENT_TRIGGER"}
    if set(policy.get("high_impact_action_classes") or []) != required_high_impact:
        validation.error("GitHub high-impact action set is incomplete or unexpectedly widened")
    for action_class in required_high_impact:
        if not policy.get("action_classes", {}).get(action_class, {}).get("independent_review_required"):
            validation.error(f"GitHub high-impact action lacks independent review: {action_class}")

    release_permissions = permissions.get("roles", {}).get("release-evidence-agent", {})
    if "GITHUB_FULL_CONTROL" not in set(release_permissions.get("allowed_modes") or []):
        validation.error("Release and Evidence Agent lacks GITHUB_FULL_CONTROL mode")
    if "github-full-control" not in set(release_permissions.get("conditional_tool_profiles") or []):
        validation.error("Release and Evidence Agent lacks github-full-control tool profile")
    if release_permissions.get("github_operation_gate_required") is not True:
        validation.error("Release and Evidence Agent must require a GitHub operation gate")
    if "github-full-control" not in (tools.get("profiles") or {}):
        validation.error("github-full-control tool profile is missing")

    github_profiles = [item for item in profiles.get("profiles", []) if item.get("profile_id") == "github-manager"]
    if len(github_profiles) != 1 or github_profiles[0].get("parent_roles") != ["release-evidence-agent"]:
        validation.error("github-manager specialist profile must resolve only under release-evidence-agent")

    if credentials.get("default_policy") != "DENY_UNLESS_EXACT_PROFILE_APPROVED":
        validation.error("GitHub credential profiles must fail closed")
    if credentials.get("secret_values_in_registry") != "PROHIBITED":
        validation.error("GitHub credential registry must prohibit secret values")

    example = validation.load_yaml(AI_ROOT / "examples" / "github-operation-gate.example.yaml") or {}
    if not example.get("unresolved_blockers"):
        validation.error("GitHub operation example must remain a denied non-operational example")
    if any(item.get("satisfied") is True for item in (example.get("evidence") or {}).values()):
        validation.error("GitHub operation example must not claim satisfied operational evidence")


def validate_repository_workflow(validation: Validation) -> None:
    policy = validation.load_yaml(AI_ROOT / "policies" / "repository-workflow-and-ci.yaml") or {}
    if policy.get("lifecycle_status") != "APPROVED" or policy.get("approved_by") != "Sole Human Owner":
        validation.error("Repository workflow and CI policy must record Sole Human Owner approval")
    pull_requests = policy.get("pull_requests", {})
    if pull_requests.get("protected_branch_delivery") != "PULL_REQUEST_REQUIRED":
        validation.error("Protected branch delivery must require a pull request")
    if pull_requests.get("directly_authorized_out_of_issue_work") != "OPEN_PULL_REQUEST_AS_PRIMARY_AUDIT_RECORD":
        validation.error("Directly authorized out-of-Issue work must use a pull request audit record")
    if pull_requests.get("direct_push_to_main") != "DENY":
        validation.error("Direct push to main must remain denied")

    ci = policy.get("ci", {})
    if ci.get("primary_environment") != "LOCAL":
        validation.error("CI primary environment must be LOCAL")
    incremental = ci.get("incremental", {})
    if incremental.get("execution") != "RUN_ALL_APPLICABLE_CHECKS_IN_ONE_BATCH":
        validation.error("Incremental CI must batch all applicable checks")
    if incremental.get("failure_behavior") != "COLLECT_ALL_FAILURES_WITHOUT_FAIL_FAST":
        validation.error("Incremental CI must collect every failure")
    expected_checkpoint_fields = {"CHECK_COMMAND_DIGEST", "TOOLCHAIN_IDENTITY", "REPOSITORY_INPUT_DIGEST", "BASE_COMMIT"}
    actual_checkpoint_fields = set(incremental.get("checkpoint", {}).get("required_match_fields") or [])
    if actual_checkpoint_fields != expected_checkpoint_fields:
        validation.error("CI checkpoint identity fields are incomplete or unexpectedly widened")
    if ci.get("github", {}).get("prerequisite") != "APPLICABLE_LOCAL_INCREMENTAL_CI_PASS":
        validation.error("GitHub CI must require the applicable local CI pass")
    if ci.get("full", {}).get("allowed_trigger") != "MILESTONE_CLOSE_ONLY":
        validation.error("Full CI must be limited to Milestone closure")
    if ci.get("full", {}).get("checkpoint_skip") != "DENY":
        validation.error("Full CI must not skip checks from checkpoints")

    branch_lifecycle = policy.get("branch_lifecycle", {})
    if branch_lifecycle.get("delete_head_branch_after_merge") is not True:
        validation.error("Merged head branches must be deleted")
    if branch_lifecycle.get("delete_closed_unmerged_or_abandoned_branch") is not True:
        validation.error("Closed-unmerged or abandoned branches must be cleaned after safety checks")
    if branch_lifecycle.get("postcondition") != "BRANCH_ABSENCE_VERIFIED":
        validation.error("Branch cleanup must require absence readback")

    config = validation.load_json(REPO_ROOT / ".ci" / "local-ci.json") or {}
    checks = config.get("checks") or []
    check_ids = [item.get("id") for item in checks if isinstance(item, dict)]
    unique_values(validation, check_ids, "local CI checks")
    if set(check_ids) != {"agent-os-regression", "foundation-validation"}:
        validation.error("Platform local CI must include agent regression and foundation validation")
    for item in checks:
        if not isinstance(item, dict) or not isinstance(item.get("command"), list) or not item.get("command"):
            validation.error("Every local CI check must have a non-empty command list")


def validate_provider_routes_fail_closed(validation: Validation, role_ids: set[str]) -> None:
    routing = validation.load_yaml(AI_ROOT / "policies" / "provider-routing.yaml") or {}
    if routing.get("routing_status") != "NO_APPROVED_ROUTES":
        validation.error("Canonical provider routing must remain NO_APPROVED_ROUTES")
    if set((routing.get("role_routes") or {}).keys()) != role_ids:
        validation.error("Provider routing role keys must exactly match canonical roles")
    for role_id, route in (routing.get("role_routes") or {}).items():
        if route.get("dispatch_enabled") is not False:
            validation.error(f"Provider routing unexpectedly enables {role_id}")
        if route.get("primary_route_id") is not None or route.get("fallback_route_ids"):
            validation.error(f"Provider routing assigns an unapproved route to {role_id}")

    models = validation.load_yaml(AI_ROOT / "policies" / "model-registry.yaml") or {}
    if models.get("dispatch_default") != "DENY":
        validation.error("Model registry dispatch default must be DENY")
    if models.get("approved_model_refs") or models.get("enabled_model_record_ids"):
        validation.error("Model registry must not approve or enable a model in this preparation")
    for model in models.get("models", []):
        if model.get("dispatch_enabled") is not False:
            validation.error(f"Model is unexpectedly enabled: {model.get('model_record_id')}")

    routes = validation.load_yaml(AI_ROOT / "provider-routes" / "ai-service-route-registry.yaml") or {}
    if routes.get("dispatch_default") != "DENY" or routes.get("default_dispatch_policy") != "DENY_UNLESS_EXACT_ROUTE_APPROVED":
        validation.error("Provider route registry must fail closed")
    if routes.get("active_route_ids") or routes.get("approved_route_ids") or routes.get("enabled_route_ids"):
        validation.error("Provider route registry must have no active, approved, or enabled routes")
    if routes.get("global_controls", {}).get("hidden_subagents") != "PROHIBITED":
        validation.error("Provider route registry must prohibit hidden subagents")
    for route in routes.get("routes", []):
        route_id = route.get("provider_route_id")
        if route.get("approved_roles") or route.get("allowed_data_classes"):
            validation.error(f"Candidate route has authority before approval: {route_id}")
        if route.get("lifecycle", {}).get("dispatch_enabled") is not False:
            validation.error(f"Candidate route is unexpectedly enabled: {route_id}")

    reviews = validation.load_yaml(AI_ROOT / "provider-routes" / "provider-policy-review-register.yaml") or {}
    for review in reviews.get("reviews", []):
        if review.get("route_decision") != "DENY":
            validation.error(f"Provider review is not fail-closed: {review.get('review_id')}")
        document = AI_ROOT / "provider-routes" / "reviews" / str(review.get("document", ""))
        if not document.is_file():
            validation.error(f"Provider review document is missing: {review.get('document')}")


def validate_examples(validation: Validation) -> None:
    mappings = {
        "agent-assignment.example.yaml": "agent-assignment.schema.json",
        "provider-data-policy-review.example.yaml": "provider-data-policy-review.schema.json",
        "provider-route-intake.example.yaml": "ai-service-route-registry.schema.json",
        "result-contract.example.yaml": "result.schema.json",
        "task-packet.example.yaml": "task.schema.json",
        "agent-session.example.yaml": "agent-session.schema.json",
        "write-lease.example.yaml": "write-lease.schema.json",
        "evidence-record.example.yaml": "evidence-record.schema.json",
        "incident-record.example.yaml": "incident-record.schema.json",
        "github-operation-gate.example.yaml": "github-operation-gate.schema.json",
    }
    for example_name, schema_name in mappings.items():
        instance = validation.load_yaml(AI_ROOT / "examples" / example_name)
        schema = validation.load_json(AI_ROOT / "schemas" / schema_name)
        if instance is None or schema is None:
            continue
        validator = Draft202012Validator(schema, format_checker=FormatChecker())
        for error in sorted(validator.iter_errors(instance), key=lambda item: list(item.path)):
            location = "/".join(str(part) for part in error.path) or "<root>"
            validation.error(f"Example {example_name} fails {schema_name} at {location}: {error.message}")


def validate_readiness_and_runbooks(validation: Validation) -> None:
    readiness = validation.load_yaml(AI_ROOT / "readiness.yaml") or {}
    if readiness.get("state") != "PREPARED_FOR_REVIEW_NOT_RUNTIME_READY":
        validation.error("Readiness must not claim runtime readiness")
    if not readiness.get("runtime_blockers"):
        validation.error("Readiness must list runtime blockers")
    if readiness.get("dispatch_rule") != "DENY_UNTIL_EVERY_APPLICABLE_RUNTIME_BLOCKER_IS_RESOLVED_AND_HUMAN_APPROVAL_IS_RECORDED":
        validation.error("Readiness dispatch rule must fail closed")
    runbooks = [path for path in (AI_ROOT / "runbooks").glob("*.md") if path.name != "README.md"]
    if len(runbooks) < EXPECTED_MINIMUM_COUNTS["runbooks"]:
        validation.error(f"Expected at least {EXPECTED_MINIMUM_COUNTS['runbooks']} operational runbooks, found {len(runbooks)}")


def main() -> int:
    validation = Validation()
    if not AI_ROOT.is_dir():
        print("ERROR: .ai directory is missing", file=sys.stderr)
        return 1

    validate_required_layout(validation)
    validate_parse_all(validation)
    validate_local_markdown_links(validation)
    validate_json_schemas(validation)
    role_ids, _ = validate_roles(validation)
    skill_ids = validate_skills_and_bundles(validation, role_ids)
    validate_profiles_and_permissions(validation, role_ids, skill_ids)
    validate_policies(validation)
    validate_github_authority(validation)
    validate_repository_workflow(validation)
    validate_provider_routes_fail_closed(validation, role_ids)
    validate_examples(validation)
    validate_readiness_and_runbooks(validation)

    if validation.errors:
        print(f"AI Agent OS validation FAILED with {len(validation.errors)} error(s):")
        for message in validation.errors:
            print(f"- {message}")
        return 1

    print("AI Agent OS validation PASSED")
    print(f"roles={len(role_ids)} profiles>=17 skills={len(skill_ids)} policies>=18 runbooks>=15 schemas>=20")
    print(f"parsed_or_checked_files={len(validation.checked_files)} provider_routes_enabled=0")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
