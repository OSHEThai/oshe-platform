# Backup or Restore Failure

## Trigger

A backup, restore, reconstruction, integrity, provider-exit, or continuity exercise fails or produces inconclusive evidence.

## Procedure

1. Stop destructive retries and preserve the failed exercise state and logs.
2. Verify source backup identity, checksum, encryption access, dependencies, ordering, and target isolation.
3. Classify data loss, corruption, missing configuration, credential, tooling, or procedure cause.
4. Repair and repeat only in an approved non-production environment unless a human emergency decision authorizes otherwise.
5. Reassess recovery objectives and continuity claims.

## Resume Criteria

Integrity and post-restore checks pass for the exact scenario, or the gap is explicitly accepted by the Sole Human Owner within authority.

## Required Record

Asset and backup identity, environment, commands, failure, containment, retest, achieved objectives, unresolved loss, and human decision.
