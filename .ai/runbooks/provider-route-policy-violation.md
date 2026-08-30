# Provider Route Policy Violation

## Trigger

An agent used or attempted to use an unknown, disabled, expired, over-limit, data-incompatible, or otherwise unapproved provider route.

## Procedure

1. Suspend the assignment and disable further use of the affected route.
2. Record route, model/runtime, CLI/adapter, session, task, data classes, tools, timestamps, and possible disclosures.
3. Preserve safe logs and configuration digests without reproducing sensitive content.
4. Assess exposure, revoke credentials through the human-controlled process when needed, and notify the Sole Human Owner.
5. Requalify the exact route; do not substitute a provider, model, data class, or tool scope automatically.

## Resume Criteria

Resume only with a newly validated assignment and an exact approved route. A policy violation cannot be closed by retrying successfully.

## Required Record

Incident reference, affected artifacts, containment, disclosure assessment, credential action, validation evidence, unresolved risk, and human decision.
