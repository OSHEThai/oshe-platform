---
document_id: TPL-PDR-001
title: Provider Data Policy Review Template
document_type: review_template
document_version: 1.0.0
lifecycle_status: APPROVED
maturity: OPERATIONAL
implementation_status: IMPLEMENTED_FOR_PLANNING_CONTROL
review_status: APPROVED
owner: Security Privacy and Product Safety Agent
reviewers:
- Research and Legal Content Agent
- Independent Review and Challenge Agent
approval_authority: Sole Human Owner / Human Product and Release Authority
applicable_releases:
- v0.1.0 onward
effective_date: '2026-08-18'
source_of_truth: GOOGLE_DRIVE
classification: INTERNAL
change_risk: R3
related_decisions:
- ADR-0005
---

# Provider Data Policy Review Template

## 1. Route identity

Record the `route_id`, provider/service/model label, and intended role/task/data scope. Do not collect or require account or organization profile, service tier, region or host, endpoint or runtime source, model revision/digest, authentication reference, or configuration digest. CLI and adapter observations may be recorded when available, but are not prerequisites.

An unknown provider/service/model label is recorded as `UNKNOWN` and the route remains denied.

## 2. Evidence sources

Use authoritative provider terms, data-processing settings, security and privacy documentation, subprocessors, support documentation, incident terms, and controlled observations. Marketing summaries are supplemental only.

For every source record title, URL or controlled reference, effective or accessed date, authority, applicable provider/service/model label, known limitations, and preserved snapshot where permitted.

## 3. Data-handling assessment

Assess separately:

- input, prompt, attachment, code, metadata and tool-call handling;
- output, transcript and derived-data handling;
- retention periods and account controls;
- training, service improvement and abuse-monitoring use;
- human support or reviewer access;
- cache, telemetry, logs, backups and deletion;
- export and portability;
- encryption in transit and at rest;
- region, residency and subprocessors;
- incident notification and audit evidence;
- local-runtime host, storage, package provenance, network and update behavior where applicable.

## 4. Data-class decisions

Record a separate `ALLOW`, `ALLOW_WITH_LIMITS`, `DENY`, or `UNKNOWN` decision for `PUBLIC`, `INTERNAL`, `RESTRICTED`, `CONFIDENTIAL_PERSONAL`, and `HIGHLY_RESTRICTED`.

An unknown or contradictory policy fact keeps the affected data class denied. Approval for one data class does not imply approval for another.

## 5. Scope and controls

Record approved roles, task families, risk ceiling, assurance level, tools, network, sandbox, filesystem scope, write lease, retention, evidence, quota, concurrency, cost class, fallback route, timeout, and stop conditions.

Provider-native memory, tools, web access, background agents, code execution or subagents remain disabled unless explicitly registered, bounded, tested and evidenced.

## 6. Findings and limitations

Record unresolved questions, contradictions, assumptions, accepted limitations, corrective actions, expiry and requalification triggers. Do not convert absence of evidence into permission.

## 7. Independent review and decision

The Security Privacy and Product Safety Agent prepares the review. The Independent Review and Challenge Agent performs independent review. Qualified human legal or privacy review is added when terms, regulated data or claims require it. The Sole Human Owner records the final route and data-class decision.

## 8. Required output

Produce a schema-valid provider-policy review record, evidence references, data-class matrix, limitations, review status, approval reference, and updates to `provider-policy-review-register.yaml`, `ai-service-route-registry.yaml`, and `model-registry.yaml`.
