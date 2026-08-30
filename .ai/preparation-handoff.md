# AI Agent OS Preparation Handoff

Prepared: 30-08-2026
Release topic: V010-T04
State: Prepared for controlled pull-request publication; live commit, push, PR, and merge state require GitHub readback. Runtime dispatch remains disabled.

## Aggregate Staging Purpose

The preparation branch assembles and validates the complete target state so cross-references can be checked before publication. It is not intended to become one oversized pull request.

## Intended Pull Request Slices

1. V010-I013 / GitHub issue #16 — root agent contract, CLI adapters, prompt envelope, readiness, repository boundaries, and authority controls.
2. V010-I014 / GitHub issue #17 — canonical role registry, 12 role cards, 17 specialist profiles, role bundles, permissions, tool profiles, delegation, and write leases.
3. V010-I015 / GitHub issue #18 — schemas, valid examples, static validator, validation dependencies, and CI integration.
4. V010-I016 / GitHub issue #19 — 41-skill catalog, skill registry, operational policies, provider notes, and 15 runbooks.

ADR-0006 and its evidence-gated full GitHub authority are an R4 authority change. ADR-0007 records the Sole Human Owner's directly authorized out-of-Issue pull-request, local-first CI, Milestone-only Full CI, branch cleanup, and workspace cleanup rules. Both decisions require explicit PR disclosure before publication.

Each slice must preserve resolvable references at its exact head or declare a controlled dependency on an earlier slice. Re-run the integrated validator after every split and rebase.

## Static Acceptance

- `python .ai/tools/validate_agent_os.py`
- `python -m unittest tests.test_validate_agent_os`
- `python tools/validate_repository.py --repo-kind platform`
- no enabled provider route or model;
- no missing role, skill, profile, bundle, policy, schema, example, provider-review, or local Markdown reference;
- no runtime state, transcript, credential, customer data, medical data, or security-case data committed.

## Runtime and Publication Gates Still Open

Static preparation does not qualify Herdr dispatch. ADR-0006 provides standing conditional authority for full GitHub operations, but execution still requires an approved provider route, approved GitHub credential profile, visible session, exact operation-gate PASS, and independent-review PASS for high-impact actions. Non-GitHub production, customer-data, legal, safety, residual-risk, organization ownership, billing, and succession decisions remain with the Sole Human Owner.

Ordinary validation follows ADR-0007: one non-fail-fast local incremental batch, state-bound checkpoint reuse, then GitHub CI. Full CI is reserved for Milestone closure and must pass locally before GitHub dispatch. Completed work removes unreferenced branches, worktrees, caches, logs, downloads, and failed outputs while preserving evidence and valid checkpoints.
