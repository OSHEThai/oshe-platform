# Repository Boundary Violation

## Trigger

An assignment writes outside its repository, module ownership, worktree, branch, or leased path scope.

## Procedure

1. Stop the assignment and preserve the full diff and Git state for every affected repository.
2. Identify the first out-of-scope action, affected artifacts, possible data exposure, and concurrent writers.
3. Separate authorized work from contamination without discarding evidence or unrelated user changes.
4. Restore clean ownership through explicit repository-specific tasks and non-overlapping leases.
5. Re-run validation and independent review for affected boundaries.

## Resume Criteria

Every remaining change has an authorized repository, role assignment, branch/worktree, and path lease; contamination is resolved and recorded.

## Required Record

Repositories, commits, branches, worktrees, changed paths, cause, containment, recovery, validation, and decisions needed.
