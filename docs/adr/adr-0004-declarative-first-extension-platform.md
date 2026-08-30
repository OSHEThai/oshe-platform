---
document_id: ADR-0004
title: Declarative-First Extension Platform
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
next_review_trigger: Declarative engines cannot express a recurring validated requirement.;
  A new runtime is required for a supported industry or hardware integration.; Marketplace
  risk, support, or compatibility evidence requires a policy revision.
source_of_truth: GOOGLE_DRIVE
classification: INTERNAL
change_risk: R2
related_decisions:
- DEC-0004
related_issues:
- V010-I001
supersedes: []
superseded_by: null
---

# ADR-0004 — Declarative-First Extension Platform

## Status

**Accepted on 2026-08-12.**

## Context

Customers, partners, industries, and small organizations require different forms, checklists, workflows, reports, connectors, and specialized applications. Loading arbitrary third-party code into the core process or allowing direct database access would undermine tenant isolation, safety controls, upgrades, and support.

## Decision

1. Use a declarative-first extension platform for forms, checklists, workflows, rules, reports, dashboards, notifications, signs, translations, content packs, and mini applications.
2. Add controlled remote services, sandboxed UI, WebAssembly functions, and isolated container services only when declarative capabilities are insufficient.
3. Require every extension to declare compatibility, permissions, data classes, external transfer, network destinations, runtime, support, lifecycle, and risk class.
4. Calculate extension access as the intersection of extension permission, company approval, project/site enablement, user permission, and data classification.
5. Prohibit direct access to core databases, message brokers, object-storage credentials, caches, search indexes, and production secrets.
6. Prohibit third-party extensions from performing protected final actions such as final PTW approval, LOTO release, medical fitness decisions, legal compliance declarations, or critical-investigation closure.

## Alternatives Considered

- In-process DLL or package plugins.
- Customer source-code forks.
- Arbitrary containers inside pooled SaaS data cells.
- Direct database integration.

## Rationale

- Allows broad customization without compromising the core trust boundary.
- Supports standalone products, enterprise composition, self-hosted deployment, and site edge.
- Enables a marketplace with risk-based trust and review.
- Preserves upgrade, rollback, uninstall, and revocation controls.

## Positive Consequences

- Most customer needs can be delivered without code forks.
- Permissions, compatibility, and data use are reviewable before installation.
- Official, partner, community, and private catalogs can share one contract.

## Negative Consequences and Trade-offs

- Extension APIs and studios require substantial platform investment.
- Some specialized use cases must remain external or isolated.
- Publisher review and vulnerability response add operational overhead.

## Mandatory Constraints

- Declarative content is preferred whenever it can meet the requirement.
- Code extensions run outside the core process with resource and network limits.
- Published artifacts are signed and include an SBOM or content bill of materials as applicable.
- Permission expansion and new external destinations require renewed consent.

## Validation

- Architecture, policy, schema, and implementation work must reference this ADR where applicable.
- CI and architecture tests will progressively encode the mandatory constraints.
- Material exceptions require an RFC, impact assessment, compensating controls, and human approval.

## Review Triggers

- Declarative engines cannot express a recurring validated requirement.
- A new runtime is required for a supported industry or hardware integration.
- Marketplace risk, support, or compatibility evidence requires a policy revision.
