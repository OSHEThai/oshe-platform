---
document_id: ADR-0002
title: Shared Codebase and Deployment Profiles
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
next_review_trigger: A profile requires behavior that cannot be represented safely
  through supported composition and adapters.; Evidence demonstrates that a shared
  release train creates unacceptable operational or regulatory risk.; A deployment
  profile reaches end of support.
source_of_truth: GOOGLE_DRIVE
classification: INTERNAL
change_risk: R2
related_decisions:
- DEC-0002
related_issues:
- V010-I001
supersedes: []
superseded_by: null
---

# ADR-0002 — Shared Codebase and Deployment Profiles

## Status

**Accepted on 2026-08-12.**

## Context

The platform must operate as SaaS, self-hosted software, site-edge nodes, hybrid installations, and air-gapped bundles. Customer-specific forks would make security updates, legal-content updates, compatibility, support, and migration unmanageable.

## Decision

1. Maintain a shared codebase, contract set, and signed artifact family for all supported deployment profiles.
2. Represent products and deployment modes through composition manifests, configuration profiles, packs, entitlements, and runtime adapters rather than customer forks.
3. Build an artifact once and promote the same digest through test, staging, canary, and production channels.
4. Use compatibility matrices and a platform BOM to identify tested combinations of application, schema, product, edge, pack, extension, and configuration versions.
5. Treat a site-edge node as a scoped operational authority that synchronizes versioned events and immutable files, not as a raw database replica.

## Alternatives Considered

- Separate SaaS, self-hosted, and edge codebases.
- Customer-specific branches or forks.
- Database replication as the primary site-edge model.

## Rationale

- Keeps fixes and security controls consistent across deployment profiles.
- Supports customer migration between SaaS and self-hosted modes.
- Preserves one extension and pack ecosystem.
- Avoids divergent domain behavior and unsupported customer variants.

## Positive Consequences

- Consistent contracts, migrations, tests, and support documentation.
- Lower release and vulnerability-management overhead.
- Clear upgrade and rollback paths.

## Negative Consequences and Trade-offs

- Deployment abstractions and compatibility tests require early investment.
- The shared codebase must tolerate different infrastructure capabilities.
- Air-gapped update and revocation processes add packaging complexity.

## Mandatory Constraints

- No production deployment uses an untracked `latest` artifact.
- Environment-specific behavior is configured through validated profiles.
- Customer changes use supported overlays, packs, connectors, or extensions.
- Offline and site-edge behavior declares authority, freshness, conflict, and recovery rules.

## Validation

- Architecture, policy, schema, and implementation work must reference this ADR where applicable.
- CI and architecture tests will progressively encode the mandatory constraints.
- Material exceptions require an RFC, impact assessment, compensating controls, and human approval.

## Review Triggers

- A profile requires behavior that cannot be represented safely through supported composition and adapters.
- Evidence demonstrates that a shared release train creates unacceptable operational or regulatory risk.
- A deployment profile reaches end of support.
