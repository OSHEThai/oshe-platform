---
document_id: ADR-0001
title: 'Initial Core Architecture: Modular Monolith'
document_type: architecture_decision_record
document_version: 1.0.0
lifecycle_status: APPROVED
maturity: BASELINE
implementation_status: IMPLEMENTED
review_status: APPROVED
owner: Human Product and Release Authority
reviewers:
- Architecture Owner
- Security Lead
- Test Lead
applicable_releases:
- v0.1.0
effective_date: '2026-08-12'
last_reviewed_date: '2026-08-12'
next_review_trigger: A module requires independent scaling that cannot be achieved
  by process or resource separation.; A regulatory, medical, security, or customer
  isolation boundary requires a separate runtime or database.; An independently staffed
  team owns a separate release and support lifecycle.; Reliability evidence shows
  a shared failure domain is no longer acceptable.
source_of_truth: GOOGLE_DRIVE
classification: INTERNAL
change_risk: R2
related_decisions:
- DEC-0001
related_issues:
- V010-I001
supersedes: []
superseded_by: null
---

# ADR-0001 — Initial Core Architecture: Modular Monolith

## Status

**Accepted on 2026-08-12.**

## Context

The product must support a broad OSHE domain model, shared capability engines, standalone product compositions, SaaS, self-hosted deployment, and site-edge operation. Development is initially performed by one human owner using a controlled multi-agent engineering workflow. Atomic cross-module changes, coherent permissions, shared contracts, and low operational overhead are more important at this stage than independent service deployment.

## Decision

1. Use a modular monolith as the initial authoritative business-application architecture.
2. Keep business domains in explicit modules with owned schemas, aggregates, application services, events, and public contracts.
3. Use shared building blocks only for genuinely cross-cutting platform concerns.
4. Prohibit direct updates to another module's tables or internal state.
5. Permit separate deployable processes for web applications, API hosting, background workers, sync gateways, reporting, AI gateways, and extension hosts while preserving one modular source model.
6. Extract a module into an independently deployed service only after a documented RFC demonstrates a distinct scale, security, ownership, availability, or release-lifecycle need.

## Alternatives Considered

- Microservices from the first release.
- An unrestricted monolith with shared tables and internal calls.
- One repository or deployable service per business module.

## Rationale

- Minimizes coordination and operational cost for a solo human owner.
- Supports atomic refactoring and contract evolution while the domain model is still forming.
- Makes module-boundary and tenant-isolation tests practical before distribution complexity is introduced.
- Retains a controlled extraction path through versioned contracts and domain events.

## Positive Consequences

- Simpler local development, CI, debugging, migration, and release management.
- One place to enforce architecture, permission, audit, and record-integrity rules.
- Lower risk of premature distributed consistency failures.

## Negative Consequences and Trade-offs

- Requires strict architecture tests to prevent erosion into a tightly coupled monolith.
- Some modules may later require extraction work.
- Deployment scaling is initially coarser than a fully decomposed service architecture.

## Mandatory Constraints

- Module-owned data and internal types remain private.
- Cross-module interaction uses versioned application contracts, references, assertions, snapshots, or events.
- Safety-critical state transitions remain authoritative within the owning module.
- Architecture tests enforce dependency direction and prohibited references.

## Validation

- Architecture, policy, schema, and implementation work must reference this ADR where applicable.
- CI and architecture tests will progressively encode the mandatory constraints.
- Material exceptions require an RFC, impact assessment, compensating controls, and human approval.

## Review Triggers

- A module requires independent scaling that cannot be achieved by process or resource separation.
- A regulatory, medical, security, or customer isolation boundary requires a separate runtime or database.
- An independently staffed team owns a separate release and support lifecycle.
- Reliability evidence shows a shared failure domain is no longer acceptable.
