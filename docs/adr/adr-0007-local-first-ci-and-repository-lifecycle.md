---
id: ADR-0007
title: Local-First CI and Repository Lifecycle
lifecycle_status: APPROVED
decision_date: 30-08-2026
effective_date: 30-08-2026
approved_by: Sole Human Owner
supersedes: []
---

# ADR-0007: Local-First CI and Repository Lifecycle

## Context

The repository needs a fast default validation loop, an auditable route for work that was not first represented by an Issue, and deterministic cleanup of branches and disposable artifacts. Running every possible check repeatedly is slow and consumes local and GitHub capacity without improving evidence when neither inputs nor the check definition changed.

## Decision

1. Every change reaches `main` through a pull request. Direct pushes to protected `main` remain prohibited.
2. Work already represented by an Issue must link that Issue. Work directly authorized by the Sole Human Owner that is outside the prepared Issue set must open a pull request as its primary audit record; a synthetic Issue is not required first.
3. Local CI is the primary validation environment. Before a pull request is opened or updated for review, run all applicable local checks in one non-fail-fast batch so the complete failure set can be corrected together.
4. A passing check may be checkpointed and skipped on a later incremental run only when its command, toolchain identity, repository input digest, and base commit are unchanged. Changed or unverifiable inputs invalidate the checkpoint.
5. GitHub CI runs only after the applicable local incremental CI batch passes. GitHub remains the protected-branch verification environment, not the first diagnostic loop.
6. Full CI is authorized only for Milestone closure. It must first pass locally against the exact candidate commit, then pass on GitHub against the same commit. Incremental CI is used for ordinary Issue and pull-request work.
7. A pull request may merge only when the exact-head local evidence, required GitHub checks, review requirements, mergeability, scope, and operation gate all pass.
8. GitHub head branches are deleted automatically after merge. A branch for a closed-unmerged pull request or abandoned task is deleted after confirming that no active pull request, worktree, unrecovered commit, release, or evidence reference still needs it.
9. At task completion, remove disposable worktrees, local branches, caches, temporary logs, downloaded artifacts, and failed build outputs that are no longer referenced. Preserve required evidence records and valid CI checkpoint metadata.
10. Cleanup must resolve exact targets immediately before deletion, remain inside the governed repository or workspace, and be followed by readback verification.

## Consequences

- Contributors receive the whole actionable failure set from one local pass.
- Unchanged passing checks are not repeated without evidence value.
- GitHub Actions usage is focused on protected integration evidence.
- Full CI cost is reserved for Milestone closure.
- Directly authorized unplanned work remains reviewable through a pull request.
- Merged and abandoned branches do not accumulate indefinitely.

## Verification

- `.ai/policies/repository-workflow-and-ci.yaml` records the executable policy.
- `tools/run_local_ci.py` aggregates all configured check results and maintains state-bound checkpoints.
- `.github/workflows/foundation.yml` invokes the same runner without trusting local checkpoint files.
- `.github/PULL_REQUEST_TEMPLATE.md` requires local CI and cleanup evidence.
