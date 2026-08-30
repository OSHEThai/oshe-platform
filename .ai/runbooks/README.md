# Runbook Index

- `mission-start.md` — Validate mission, resolve versions, create integration branch, allocate slots, and dispatch only ready tasks.
- `blocked-agent.md` — Read the structured blocker, classify missing authority versus technical issue, answer or escalate, then record the decision.
- `provider-quota-exhaustion.md` — Freeze new work on the provider, preserve active state, reroute eligible tasks without weakening gates, and notify the Project Management Agent and Sole Human Owner when a human decision is required.
- `stuck-or-crashed-agent.md` — Capture pane output, inspect worktree and local commits, quarantine partial changes, restart from verified checkpoint, and reconcile.
- `write-lease-violation.md` — Stop the agent, capture the diff, revoke the lease, restore clean scope, review for contamination, and record an incident.
- `integration-conflict.md` — Classify semantic versus mechanical conflict, assign one accountable integrator, rerun affected checks, and document resolution.
- `security-or-secret-exposure.md` — Stop all related agents, revoke credentials, preserve minimal forensic evidence, sanitize worktrees and transcripts, and escalate to human authority.
- `human-handoff.md` — Summarize scope, evidence, unresolved findings, decisions, risks, rollback, and recommendation without burying critical information.
- `provider-route-policy-violation.md` — Disable the affected route, contain possible data exposure, preserve evidence, and require requalification.
- `validation-or-schema-failure.md` — Stop integration, isolate the invalid artifact, reproduce the failure, and repair without weakening the contract.
- `independent-review-disagreement.md` — Preserve both positions, prohibit producer self-resolution, and escalate material disagreement.
- `backup-restore-failure.md` — Preserve restore evidence, prevent destructive retries, reassess recovery objectives, and obtain the recovery decision.
- `repository-boundary-violation.md` — Stop cross-repository or cross-module writes, inspect contamination, and re-establish separate leases.
- `github-operation-failure.md` — Stop partial or divergent GitHub operations, preserve the exact gate and audit evidence, contain credentials, and apply only approved recovery.
- `repository-pr-ci-lifecycle.md` — Run local-first checkpointed CI, open the correct PR record, gate GitHub CI and merge, then delete unused branches and disposable outputs.

## AI Service Route-Control References

- [`../provider-routes/`](../provider-routes/) — planning-control route registry, reviews, usage controls, and qualification evidence.
- Unqualified, unknown, suspended, expired, or over-limit routes fail closed; no provider substitution or data-class change is inferred.
- The no-eligible-route outcome is task suspension and human escalation, not improvised dispatch.

Every runbook is fail-closed. A completed runbook record is evidence of response activity, not automatic proof of recovery, approval, runtime readiness, or release qualification.
