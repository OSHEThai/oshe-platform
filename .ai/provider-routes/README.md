---
document_id: IDX-AISR-001
title: AI Service Route Registry and Provider Reviews Package Index
document_type: registry_package_index
document_version: 1.0.0
lifecycle_status: APPROVED
maturity: OPERATIONAL
implementation_status: IMPLEMENTED_FOR_PLANNING_CONTROL
review_status: APPROVED
owner: Sole Human Owner / Human Product and Release Authority
operational_owners:
- Security Privacy and Product Safety Agent
- Project Management Agent
reviewers:
- Architecture Agent
- Test and Quality Agent
- Independent Review and Challenge Agent
applicable_releases:
- v0.1.0
- v0.2.0
- v0.3.0
- v0.4.0
effective_date: '2026-08-18'
source_of_truth: GOOGLE_DRIVE
classification: INTERNAL
change_risk: R3
related_decisions:
- DEC-008
related_questions:
- OQ-010
---

# AI Service Route Registry and Provider Reviews Package

## Purpose

Provide the machine-readable planning-control baseline for AI-service identity, provider-route lifecycle, model identity, role routing, provider data-policy review, variable usage control, and dispatch denial before qualification.

## Current decision

All seven route records are non-dispatchable; all models are unqualified; all canonical roles are unassigned; every `dispatch_enabled` value is false; and the default decision is deny. Provider/model labels are planning identifiers only. The registry does not collect account, service-tier, location, endpoint/runtime, model revision/digest, authentication-reference, or configuration-digest metadata.

An interactive AI session used to help prepare documents does not create an approved Herdr route.

## Package artifacts

- `ai-service-route-registry.yaml` — authoritative route-level planning-control registry.
- `provider-policy-review-register.yaml` — review status and current deny decision.
- `ai-service-usage-register.yaml` — empty usage-ledger structure.
- `AI Service Candidate Routing Quota and Failover Matrix.md` — human-readable status matrix.
- `Provider Data Policy Review Template.md` — mandatory route review template.
- `Provider Reviews/` — five candidate-provider review records; all are `DRAFT`, `NOT_STARTED`, and deny dispatch.
- Reference implementation `model-registry.yaml`, `provider-routing.yaml`, `budgets.yaml`, and `data-classification.yaml` — canonical planning-control materialization with no enabled route.
- Schemas and examples — route registry, model registry, provider routing, usage control, usage ledger, and provider policy review.

## Candidate service lanes inherited from the existing plan

- Qwen local runtime
- OpenAI Codex
- Google Gemini
- Anthropic Claude
- DeepSeek specialist API lane
- DeepSeek fast or bulk API lane

These names identify planning families only.

## Route activation gate

A route becomes dispatchable only after an approved provider/model label, applicable policy evidence, bounded role/task/data/tool/network scope, usage and hard-stop controls, technical preflight, negative tests, evaluation, recovery evidence, independent review, and Sole Human Owner approval are complete. Account, tier, location, endpoint/runtime, revision/digest, authentication-reference, and configuration-digest metadata are not activation prerequisites.

## Current limitations

- Registry structure is implemented; runtime enforcement is not.
- No provider policy review is complete.
- No route may process any data through Herdr dispatch.
- No fixed budget exists; development cost is variable usage-based.
- GitHub execution remains deferred by owner.
