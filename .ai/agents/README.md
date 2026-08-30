# Specialist Agent Profiles

Specialist profiles narrow a canonical role assignment for repeatable work. They are not authority roles, cannot be dispatched without a registered canonical role and approved provider route, and cannot spawn subagents.

- `repo-domain-analyst` — Repository mapping, domain boundaries, and dependency analysis; default write mode: read-only.
- `backend-dotnet-worker` — .NET domain, application, API, and infrastructure implementation; default write mode: scoped backend paths.
- `frontend-pwa-worker` — React, PWA, accessibility, forms, and offline user experience; default write mode: scoped frontend paths.
- `database-migration-worker` — PostgreSQL migrations, backfill, reconciliation, and rollback; default write mode: migration and database paths.
- `api-event-contract-worker` — OpenAPI, event, webhook, and SDK contracts; default write mode: contract paths.
- `edge-offline-sync-worker` — Outbox, retry, ordering, conflicts, cache freshness, and site edge; default write mode: sync paths.
- `unit-property-test-worker` — Unit, property, state-machine, and boundary tests; default write mode: test paths.
- `integration-e2e-worker` — Integration, browser, offline, restore, and critical journey tests; default write mode: test paths.
- `security-isolation-worker` — Authorization, tenant isolation, secret, file, and abuse tests; default write mode: security test paths.
- `docs-content-pack-worker` — Documentation, content pack drafts, changelog, and migration guides; default write mode: documentation and content paths.
- `devops-platform-specialist` — CI, deployment definitions, infrastructure controls, and supply-chain assurance; default write mode: scoped workflow and deployment paths.
- `observability-reliability-specialist` — Telemetry, SLOs, performance, capacity, resilience, and recovery evidence; default write mode: scoped observability and test paths.
- `accessibility-localization-specialist` — Accessible user experience and Thai-English localization assurance; default write mode: scoped UI, resource, test, and documentation paths.
- `content-governance-specialist` — Controlled content, provenance, citations, versions, and publication evidence; default write mode: documentation and content-control paths.
- `incident-continuity-specialist` — Incident preparation, backup, restore, reconstruction, and continuity exercises; default write mode: runbook, test, and evidence paths.
- `privacy-data-governance-specialist` — Privacy impact, minimization, classification, retention, and data-lineage review; default write mode: read-only or scoped governance paths.
- `github-manager` — Evidence-gated full GitHub operation and administration under ADR-0006; default write mode: exact passing GitHub operation gate only.

The machine-readable source is `registry.yaml`; individual profiles are under `profiles/`.
