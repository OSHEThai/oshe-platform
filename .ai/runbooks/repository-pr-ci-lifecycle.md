# Repository PR and CI Lifecycle

## Start

1. Resolve the governing Issue, or record that the work was directly authorized outside the prepared Issue set.
2. Create a short-lived branch or controlled worktree from the exact current `main` commit.
3. Record allowed paths, risk, required reviewers, and rollback or recovery needs.

## Local incremental CI

1. Run `python tools/run_local_ci.py --mode incremental`.
2. Let every applicable check finish; do not stop at the first failure.
3. Correct the complete reported failure set together.
4. Rerun. A passing checkpoint may be skipped only when the runner confirms the command, toolchain, input digest, and base commit are unchanged.
5. Record the exact commit and local CI summary in the pull request.

## Pull request and GitHub CI

1. Issue-scoped work links the governing Issue. Directly authorized out-of-Issue work opens a pull request without inventing a placeholder Issue.
2. Open or update the pull request only after local incremental CI passes.
3. Wait for GitHub checks on the exact head commit. A local pass is not a GitHub pass.
4. Resolve required review findings and re-run invalidated checks.
5. Before merge, re-read the exact head, base, mergeability, checks, review state, scope, and ADR-0006 operation gate.

## Full CI

Use Full CI only to close a Milestone:

1. Run `python tools/run_local_ci.py --mode full --milestone-close "<milestone>"` on the exact candidate commit.
2. After the local Full CI pass, dispatch GitHub Full CI for that same commit and Milestone.
3. Do not use checkpoints to skip Full CI checks.

## Cleanup

1. Merge with squash after all gates pass and delete the remote head branch.
2. For a closed-unmerged or abandoned branch, confirm it has no open pull request, active assignment or worktree, unrecovered commit, release, or evidence reference before deletion.
3. Remove disposable local worktrees, merged branches, caches, temporary logs, downloaded artifacts, and failed outputs.
4. Preserve required evidence and valid checkpoint metadata.
5. Re-read remote branches, local worktrees, local branches, and repository status to prove cleanup completed.

## Stop conditions

Stop on a stale base, moved head, missing local evidence, failed or pending GitHub check, unresolved review, ambiguous branch ownership, unrecovered commit, or failed GitHub operation gate.
