---
name: github-branch-pr-operations
description: >
  Operate branches, pull requests, reviews, merge queues, merges, and recovery after exact commit and ADR-0006 evidence gates pass.
license: Proprietary
metadata:
  oshe-version: "0.1.0"
---

# GitHub Branch and Pull Request Operations

## Objective

Publish and integrate the exact reviewed history without scope drift, missing checks, or protection bypass.

## Required Inputs

- exact repository, base branch, head branch, base SHA, head SHA, PR, merge method, and expected result;
- passing BRANCH_PR or MERGE operation gate;
- required local checkpoint or Full CI evidence, GitHub checks, and review evidence; independent-review PASS for MERGE;
- governing Issue, or an explicit record that directly authorized work is outside the prepared Issue set.

## Procedure

1. Run the applicable local CI batch first. Collect every check result without fail-fast behavior and accept checkpoint skips only for unchanged evidence keys.
2. Link Issue-scoped work. For directly authorized work outside the prepared Issue set, use the pull request as the primary audit record instead of inventing a placeholder Issue.
3. Fetch and verify remote base/head identities, protection state, conversations, checks, and review disposition.
4. Rehearse conflicts non-mutatively where applicable and confirm the passing gate matches current state.
5. Evaluate the gate immediately before push, ready-state, queue, merge, deletion, or recovery action.
6. Execute only the exact recorded operation and never weaken a rule to obtain success.
7. Read back branch, PR, merge commit, linked issues, checks, and protection state.
8. Delete a merged head branch. Delete a closed-unmerged or abandoned branch only after proving it has no active PR, worktree, unrecovered commit, release, or evidence reference.
9. Remove unreferenced local worktrees, branches, caches, logs, downloads, and failed outputs while preserving evidence and valid checkpoints.

## Required Output

Exact SHAs, Issue or direct-authorization basis, PR URL, local CI/checkpoint summary, GitHub checks, merge or branch-operation receipt, cleanup readback, reviews, resulting state, recovery evidence, and unresolved findings.

## Stop Conditions

- base/head, diff, review, checks, target protection, or merge method differs from the gate;
- an unresolved material finding or conversation remains;
- recovery or branch restoration is unavailable for a destructive operation;
- Full CI is requested for anything other than Milestone closure, or GitHub CI is requested before the applicable local pass.

## Evaluation Cases

- accept a squash merge of the exact reviewed head with all gates passing;
- accept checkpoint skips only when command, toolchain, repository input, and base commit are unchanged;
- reject stale-head merge, force-push without DESTRUCTIVE gate, protection bypass, or ordinary-work Full CI.
