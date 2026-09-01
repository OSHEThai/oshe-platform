---
document_id: ADR-0006
title: Evidence-Gated Full GitHub Operator Authority
document_type: architecture_decision_record
document_version: 1.1.0
lifecycle_status: APPROVED
maturity: BASELINE
implementation_status: IMPLEMENTED
review_status: REVIEW_REQUIRED
owner: Sole Human Owner / Human Product and Release Authority
reviewers:
- Project Management Agent
- Security Privacy and Product Safety Agent
- Independent Review and Challenge Agent
- Release and Evidence Agent
applicable_releases:
- v0.1.0
effective_date: '2026-08-30'
last_reviewed_date: '2026-08-30'
owner_instruction_date: '2026-09-01'
source_of_truth: GITHUB_AFTER_CONTROLLED_TRANSFER
classification: INTERNAL
change_risk: R4
related_decisions:
- ADR-0005
supersedes:
- ADR-0005 clauses 5 and 12 for governed GitHub operations only
superseded_by: null
---

# ADR-0006 — Evidence-Gated Full GitHub Operator Authority

## Status

Approved by the Sole Human Owner on 30-08-2026. This is standing conditional authorization: a separately assigned `release-evidence-agent` using the `github-manager` specialist profile may execute the approved GitHub operation without another per-operation human confirmation when the applicable evidence gate passes.

On 01-09-2026, the Sole Human Owner narrowed this authority by prohibiting repository deletion. That prohibition is effective immediately and does not weaken or remove the evidence gates for any other GitHub action. This amendment records the owner instruction; independent review remains pending, and no Independent Review `PASS` is claimed.

## Context

ADR-0005 initially deferred GitHub execution and retained protected GitHub actions for the Sole Human Owner. The repositories, issues, milestones, labels, workflows, and initial protection controls now exist, and the Sole Human Owner has directed that the GitHub Manager receive full GitHub authority when work evidence is complete.

A broad token or provider capability alone is not sufficient authority. The project needs a deterministic gate that binds each GitHub write to an assignment, exact target, verified evidence, review requirements, recovery plan, audit record, and the standing human decision in this ADR.

## Decision

1. `github-manager` remains a legacy alias and becomes an approved specialist profile under `release-evidence-agent`.
2. The assigned role receives full operational GitHub authority within the allowlisted `OSHEThai` organization and repositories when the operation gate passes.
3. Full GitHub authority includes:
   - repositories, branches, commits, pull requests, reviews, merge queues, and merges;
   - issues, discussions, comments, labels, milestones, projects, assignments, and relationships;
   - Actions workflows, runs, artifacts, caches, variables, environments, and GitHub deployment records;
   - tags, signatures, releases, release assets, attestations, provenance, and security metadata;
   - repository settings, rulesets, branch protections, webhooks, integrations, collaborators, teams, deploy keys, secrets, variables, visibility, archival, and transfer;
   - organization-scoped metadata and controls explicitly included in the assignment.
4. Repository deletion is prohibited. The `repository-delete` action is always denied, regardless of action class, assignment, credential capability, evidence completeness, independent review, recovery evidence, or any other gate input.
5. Every other write requires a valid assignment and a `github-operation-gate` record evaluated by the repository gate tool.
6. Metadata-only operations require the common evidence gate. Merge, release, administrative, credential, visibility, transfer, destructive, and deployment-triggering operations additionally require a distinct Independent Review and Challenge Agent disposition of `PASS`.
7. The agent may evaluate objective evidence and execute the operation, but it may not produce and independently review the same material change.
8. A passing gate authorizes only the exact action, organization, repository, target, expected pre-state, expected post-state, commit identities, and validity window recorded in the gate.
9. Missing, stale, contradictory, skipped, unknown, quarantined, or inconclusive evidence fails closed.
10. Destructive actions other than repository deletion require a verified backup or reconstruction path, target re-resolution immediately before execution, and post-operation validation.
11. Rulesets or protections may be changed when the exact administrative gate passes; they may never be weakened merely to bypass a failed check or review.
12. Secrets may be created, rotated, or deleted through approved references. Secret values may not be read back, printed, logged, committed, or placed in evidence.
13. GitHub authority does not grant non-GitHub production access, customer-data access, legal or safety approval, financial or ownership authority, or external deployment authority. A workflow that causes such an external effect requires the corresponding separate authorization.

## Required Evidence Gate

Every write operation must prove:

- registered assignment, visible session, canonical role, specialist profile, and approved provider route;
- allowlisted organization and repository;
- exact action class, target, pre-state, expected post-state, and validity window;
- exact base/head/tag/workflow/configuration identities where applicable;
- bounded scope and write lease where repository content changes;
- required checks passed on the exact applicable commit;
- required producing and independent reviews completed;
- no unresolved blocking finding, conflict, or protection violation;
- rollback, backup, recovery, or reconstruction evidence proportionate to impact;
- approved credential profile without exposed secret values;
- audit capture and post-operation verification plan;
- applicable data, legal, safety, release, and deployment boundaries remain satisfied.

No evidence gate may authorize `repository-delete`. The evaluator rejects that action before considering action class, review disposition, or evidence completeness.

## Retained Sole Human Owner Authority

The Sole Human Owner remains the accountable project and organization owner and retains:

- account ownership, billing, legal ownership, succession, and recovery-owner decisions;
- product direction, protected scope, residual-risk acceptance, and exceptions;
- customer-data, production-data, legal, medical, certification, compliance, and final safety decisions;
- non-GitHub production deployment and external-system authority unless separately delegated;
- authority to suspend or revoke this standing authorization at any time.

## Failure and Recovery

If a gate fails, the action is denied. If execution diverges from the recorded expected result, the agent stops related operations, preserves evidence, applies only the approved recovery action, invokes the GitHub operation failure runbook, and escalates unresolved impact.

A `repository-delete` request is denied unconditionally and must not proceed to execution or recovery planning as an authorized operation.

## Validation

- The role registry resolves `github-manager` to `release-evidence-agent` plus the approved specialist profile.
- The permission and tool registries expose `github-full-control` only behind the evidence gate.
- The operation schema and evaluator reject incomplete evidence and high-impact operations without independent review.
- The policy declares exactly `repository-delete` as prohibited, excludes it from allowed examples, and the evaluator rejects the canonical deletion aliases plus NFKC, case, whitespace, underscore, and hyphen variants even when every other destructive-operation condition is satisfied.
- Static validation proves the policy is internally consistent; runtime provider, credential, and adapter enforcement remain separate qualification requirements.

## Review Triggers

- Organization ownership, recovery, billing, or legal responsibility changes.
- GitHub permission, token, App, ruleset, Actions, environment, deployment, or audit behavior changes materially.
- An unauthorized, destructive, security, privacy, release, or evidence-integrity incident occurs.
- The evidence gate is bypassed, produces false approval, or cannot bind the exact operation.
- The Sole Human Owner suspends or narrows standing authorization.
