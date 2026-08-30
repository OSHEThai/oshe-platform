# OSHE Platform

This repository is the authoritative engineering source for platform applications, business modules, contracts, schemas, tests, deployment definitions, agent controls, and release evidence.

## v0.1.0 Foundation

The current repository contains structure and governance only. Product implementation starts in later topics and releases.

## Primary Areas

- `apps/` — deployable applications and processes
- `modules/` — domain-owned implementation modules
- `packages/` — approved reusable packages
- `contracts/` — public API, event, extension, and integration contracts
- `schemas/` — governed machine-readable schemas
- `database/` — migration and data-verification assets
- `tests/` — architecture, integration, permission, security, and release tests
- `deploy/` — deployment-profile definitions
- `.ai/` — canonical agent roles, skills, contracts, and policies from V010-T04
- `docs/` — ADRs, RFCs, architecture, and engineering records

## Non-Negotiables

- No direct update of another module's data.
- No client-trusted tenant or scope authority.
- No last-write-wins for protected records.
- No production secrets or customer data in development.
- No agent merge or production deployment.
