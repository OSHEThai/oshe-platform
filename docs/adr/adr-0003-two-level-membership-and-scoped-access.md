---
document_id: ADR-0003
title: Two-Level Membership and Scoped Access Model
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
next_review_trigger: A federation requirement cannot be represented through memberships,
  assertions, and data-sharing contracts.; A customer or regulator requires physically
  separate identity infrastructure.; Policy complexity exceeds defined performance
  or administration thresholds.
source_of_truth: GOOGLE_DRIVE
classification: INTERNAL
change_risk: R2
related_decisions:
- DEC-0003
related_issues:
- V010-I001
supersedes: []
superseded_by: null
---

# ADR-0003 — Two-Level Membership and Scoped Access Model

## Status

**Accepted on 2026-08-12.**

## Context

Companies can own multiple projects and sites, while project workers, contractors, consultants, and external participants must remain isolated from sibling projects and company administration. A single person may participate in multiple scopes, but authentication identity must not be confused with organizational authority.

## Decision

1. Use one trusted authentication identity that may hold multiple explicitly scoped memberships.
2. Maintain distinct company-level and project/site-level directory and membership records.
3. Allow company-level users to access only projects or sites for which they have an active membership and role assignment.
4. Prevent project/site users from entering company administration unless a separate company-level membership is granted.
5. Calculate effective authorization as the intersection of identity, entitlement, membership, role, scope, validity, delegation, data classification, and record-specific compartment.
6. Support company super administrators, company scoped administrators, project/site administrators, contractor administrators, functional roles, job positions, and legal appointments as separate concepts.

## Alternatives Considered

- One flat global user role model.
- Completely separate identity systems for company and project/site access.
- Implicit access to all company projects for every company employee.

## Rationale

- Meets the project and contractor isolation requirement.
- Avoids duplicate credentials while preserving separate authority.
- Supports external stakeholders and temporary project memberships.
- Allows explicit revocation, expiry, delegation, and access review.

## Positive Consequences

- Clear company, project, site, area, contractor, and case boundaries.
- Consistent authorization across APIs, data, files, search, exports, jobs, portals, and AI retrieval.
- Better support for federated multi-party projects.

## Negative Consequences and Trade-offs

- Authorization policies and tests are more complex than simple RBAC.
- Membership lifecycle and context selection require careful UX.
- Migration from flat directories requires explicit mapping.

## Mandatory Constraints

- Client-provided tenant, company, project, site, or contractor identifiers are never trusted as authority.
- Absence or ambiguity of required scope fails closed.
- Administrative roles do not automatically grant functional approval authority.
- Public content is served from approved publication snapshots.

## Validation

- Architecture, policy, schema, and implementation work must reference this ADR where applicable.
- CI and architecture tests will progressively encode the mandatory constraints.
- Material exceptions require an RFC, impact assessment, compensating controls, and human approval.

## Review Triggers

- A federation requirement cannot be represented through memberships, assertions, and data-sharing contracts.
- A customer or regulator requires physically separate identity infrastructure.
- Policy complexity exceeds defined performance or administration thresholds.
