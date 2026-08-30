# Mission Start

## Objective

Validate mission, resolve versions, create integration branch, allocate slots, and dispatch only ready tasks.

## Required Record

Create a run incident or handoff record with timestamps, actors, affected mission and tasks, actions, evidence, and closure decision.

## Procedure

1. Suspend the affected assignment and prevent new writes or dispatch where applicable.
2. Record exact mission, task, assignment, session, route, repository, worktree, branch, and timestamps.
3. Preserve the minimum safe evidence before cleanup, retry, reroute, or recovery.
4. Apply only the bounded action authorized by the applicable policy and assignment.
5. Re-run affected deterministic checks and obtain required independent review.
6. Record unresolved risk and obtain the Sole Human Owner decision when a protected action or resumption gate is involved.

## Stop and Resume Rules

Stop when authority, data classification, credential safety, evidence integrity, rollback, or scope is unclear. Resume only from a verified checkpoint with a valid assignment, route, tool profile, write lease when needed, and recorded approval.
