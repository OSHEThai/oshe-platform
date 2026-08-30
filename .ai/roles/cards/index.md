---
document_id: IDX-HERDR-ROLE-CARDS-001
title: Canonical ADR-0005 Role Cards Index
document_type: role_card_index
document_version: 1.0.0
lifecycle_status: APPROVED
maturity: OPERATIONAL
implementation_status: IMPLEMENTED
review_status: APPROVED
owner: Sole Human Owner / Human Product and Release Authority
reviewers:
- Project Management Agent
- Independent Review and Challenge Agent
applicable_releases:
- v0.1.0
- v0.2.0
- v0.3.0
- v0.4.0
effective_date: '2026-08-18'
source_of_truth: GOOGLE_DRIVE_UNTIL_CONTROLLED_GITHUB_TRANSFER
classification: INTERNAL
change_risk: R3
related_decisions:
- DEC-008
- ADR-0005
---

# Canonical ADR-0005 Role Cards Index

## Purpose

This folder contains the canonical planning and controlled-dispatch role cards authorized by ADR-0005. Each role is a Herdr Terminal role definition, not a person, provider brand, model alias, or permanent agent identity.

## Canonical Role Cards

1. `project-management-agent.md` — `project-management-agent`
2. `product-planning-agent.md` — `product-planning-agent`
3. `architecture-agent.md` — `architecture-agent`
4. `engineering-agent.md` — `engineering-agent`
5. `data-integration-agent.md` — `data-integration-agent`
6. `security-privacy-product-safety-agent.md` — `security-privacy-product-safety-agent`
7. `test-quality-agent.md` — `test-quality-agent`
8. `documentation-configuration-agent.md` — `documentation-configuration-agent`
9. `research-legal-content-agent.md` — `research-legal-content-agent`
10. `independent-review-challenge-agent.md` — `independent-review-challenge-agent`
11. `release-evidence-agent.md` — `release-evidence-agent`
12. `implementation-customer-success-planning-agent.md` — `implementation-customer-success-planning-agent`


## Authority and Assignment Rules

- `../registry.yaml` is the machine-readable role authority and legacy-alias resolver.
- `../../bundles/role-bundles.yaml` maps each canonical role to the current skill catalog.
- Every runtime or planning assignment must validate against `../../schemas/agent-assignment.schema.json`.
- A role card defines the maximum permitted envelope; the active assignment must be narrower.
- Provider/model/runtime selection is dynamic but must resolve to an approved service route.
- Self-approval, hidden delegation, unregistered sessions, and provider-native hidden subagents are prohibited.
- Final protected decisions remain with the Sole Human Owner.

## Legacy Names and Specialist Profiles

Legacy lead and worker names are aliases only. They resolve through `../registry.yaml` or the non-authoritative specialist registry at `../../agents/registry.yaml`; legacy names are never direct dispatch authority.

## Runtime Status

The registry, role cards, bundles, and schemas are implemented as planning-control artifacts. Static repository validation is available, but Herdr adapter validation and runtime enforcement remain implementation work under v0.1.0.
