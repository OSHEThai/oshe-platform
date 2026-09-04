---
document_id: PDR-QWEN-LOCAL-001
title: Provider Data Policy Review - Qwen Local
document_type: provider_data_policy_review
document_version: 1.0.0
lifecycle_status: DRAFT
maturity: DETAILED
implementation_status: NOT_STARTED
review_status: REVIEW_REQUIRED
owner: Security Privacy and Product Safety Agent
approver: Sole Human Owner / Human Product and Release Authority
applicable_routes:
- route-qwen-local-candidate
effective_date: null
source_of_truth: GOOGLE_DRIVE
classification: INTERNAL
change_risk: R3
current_route_decision: DENY
---

# Provider Data Policy Review - Qwen Local

## Current status

No route is approved. Unknown or conflicting facts fail closed.

## Route label and scope

Record the provider/service/model label and intended use. Do not collect or require account or local-host profile, service tier, region or host, endpoint or runtime source, model revision/digest, authentication reference, or configuration digest. CLI and adapter observations are optional.

## Authoritative evidence required

Record exact sources and effective versions for terms, data processing, security/privacy, retention, training or service-improvement use, regions, subprocessors, support access, logging, deletion, export, encryption, incident handling, administrative settings, local package provenance, licensing, host trust, and update behavior.

## Data handling assessment

Assess prompts, outputs, code, files, attachments, metadata, tool calls, telemetry, transcripts, caches, backups, support or abuse monitoring, human access, derived data, and deletion/export behavior.

## Route scope decision

Record permitted roles, purposes, task families, data classes, tools, network profile, sandbox, credentials, quotas, concurrency, cost class, failover, evaluation, limitations, expiry, and review trigger.

## Required tests and evidence

Include data-class denial, hidden-tool or subagent denial, invalid-task rejection, quota and hard-stop behavior, failure recovery, output-contract validation, independent review, and evidence bundle.

## Approval

Only the Sole Human Owner may change `current_route_decision` from `DENY`. Approval applies only to the documented provider/model label and frozen scope.

## Evidence status as of 2026-08-18

- Account, tier, location, endpoint/runtime, revision/digest, authentication, and configuration metadata: intentionally not collected
- Provider/model label: recorded from the candidate route
- CLI and adapter observations: optional
- Data-policy sources reviewed: none
- Approved roles: none
- Allowed data classes: none
- Evaluation: not started
- Dispatch: prohibited
