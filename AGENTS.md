# OSHE Platform Agent Contract

## Mission

Develop the OSHE product family while preserving tenant, company, project, site, contractor, privacy, record-integrity, and safety boundaries.

## Required Workflow

1. Read the mission and task packet.
2. Validate the assignment against `.ai/roles/registry.yaml`, `.ai/agents/registry.yaml`, the active skill bundle, risk class, provider route, tool profile, and allowed paths.
3. State assumptions before changing code.
4. Change only leased paths.
5. Run all applicable local checks as one non-fail-fast incremental batch and fix the complete failure set together.
6. Reuse a passing checkpoint only when command, toolchain, repository input, and base commit remain unchanged.
7. Open or update the pull request only after local CI passes; run Full CI only for Milestone closure, locally first and then on GitHub.
8. Produce the required structured result and clean unreferenced worktrees, branches, caches, logs, downloads, and failed outputs.
9. Stop and escalate when authority, requirements, data classification, or rollback is unclear.

## Non-Negotiable Rules

- Never trust tenant, company, project, site, or contractor scope supplied only by a client.
- Never read or write another module's tables directly.
- Never use last-write-wins for safety-critical, medical, legal, or restricted investigation state.
- Never make a destructive migration without an approved migration and recovery plan.
- Never let AI approve PTW, release LOTO, decide fitness, declare legal compliance, close a critical investigation, or publish official legal content.
- Never access production secrets or customer data from a general agent workspace.
- Never spawn hidden subagents or delegate beyond the visible Herdr task graph.
- Never treat a role card, specialist profile, provider name, model alias, or passing check as permission to dispatch.
- Never approve residual risk, customer-data use, legal or safety claims, account ownership, billing, or non-GitHub production deployment without the Sole Human Owner.
- Full GitHub operations are permitted only to the assigned Release and Evidence Agent using the `github-manager` specialist profile after the exact ADR-0006 operation gate passes; no other role inherits this authority.
- Issue-scoped work links its Issue. Directly authorized work outside the prepared Issue set still requires a pull request, which becomes its primary audit record.
- Delete merged head branches and safely delete closed-unmerged or abandoned branches after proving that no active work or recovery/evidence reference needs them.

## Standard Commands

- Validation prerequisites: `python -m pip install --disable-pip-version-check -r .ai/requirements-validation.txt`
- Foundation validation: `python tools/validate_repository.py --repo-kind platform`
- AI control validation: `python .ai/tools/validate_agent_os.py`
- AI validator regression tests: `python -m unittest tests.test_validate_agent_os`
- Local incremental CI: `python tools/run_local_ci.py --mode incremental`
- Milestone-close Full CI only: `python tools/run_local_ci.py --mode full --milestone-close "<milestone>"`
- Unresolved marker scan: `rg -n "TODO|TBC|PLACEHOLDER" . --glob '!Plan/**'`
- Product build and test commands are added only when implementation introduces the applicable runner. Do not invent a successful build for the current structure-only foundation.

## Required Output

Write a result matching `.ai/schemas/result.schema.json`, including base commit, result commit, changed files, commands, tests, assumptions, risks, and decisions needed.

## Authority and Runtime Status

- ADR-0005 defines 12 canonical role authorities; ADR-0006 grants the Release and Evidence Agent evidence-gated full GitHub authority through the `github-manager` specialist profile; ADR-0007 governs local-first CI, pull requests, branch lifecycle, and cleanup.
- Provider and model routes default to deny. A route is usable only after its provider/model label, data policy, quota, tool scope, qualification evidence, expiry, and human approval are recorded. Do not collect account, tier, location, endpoint/runtime, revision/digest, authentication-reference, or configuration-digest metadata as a route-use prerequisite.
- Runtime enforcement is not yet implemented. The repository controls are preparation and validation artifacts, not evidence of an operational dispatcher.
