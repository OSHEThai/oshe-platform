---
document_id: PDR-OPENAI-CODEX-001
title: Provider Data Policy Review - OpenAI Codex
document_type: provider_data_policy_review
document_version: 1.0.0
lifecycle_status: UNDER_REVIEW
maturity: DETAILED
implementation_status: IMPLEMENTED_FOR_PLANNING_CONTROL
review_status: REVIEW_REQUIRED
owner: Security Privacy and Product Safety Agent
approver: Sole Human Owner / Human Product and Release Authority
applicable_routes:
- route-openai-codex-candidate
effective_date: null
source_of_truth: LOCAL_REPOSITORY
classification: INTERNAL
change_risk: R3
current_route_decision: DENY
---

# Provider Data Policy Review - OpenAI Codex

## Current status

The local repository is authoritative for this review. The Google Drive copy is stale and was not consulted for this update.

- Route lifecycle: `UNDER_POLICY_REVIEW`
- Selected candidate: OpenAI Codex / ChatGPT Codex Connector
- Selection evidence: owner-supplied report of a Chrome-observed ChatGPT Codex Connector installation with identifier `141917362`
- Selection-evidence limitation: not independently verified and does not resolve provider, service, account, model, runtime, adapter, configuration, or data-policy identity
- Proposed data scope: `PUBLIC` repository metadata only
- Explicit exclusions: secrets, customer data, and production data
- Current route decision: `DENY` pending independent review
- Dispatch: prohibited

No route is approved. Unknown or conflicting facts fail closed.

## Exact route identity required

Record provider/service legal identity, account or local-host profile, service tier, region or host, endpoint, exact model ID, revision/digest, CLI/runtime, adapter, authentication mode, configuration digest, and intended use.

The selected-candidate record and connector identifier are intake evidence only. Every exact provider-identity field remains unresolved.

## Authoritative evidence required

Record exact sources and effective versions for terms, data processing, security/privacy, retention, training or service-improvement use, regions, subprocessors, support access, logging, deletion, export, encryption, incident handling, administrative settings, local package provenance, licensing, host trust, and update behavior.

No authoritative provider identity or data-policy source has been reviewed. The owner-supplied Chrome observation establishes candidate selection only.

## Data handling assessment

Assess prompts, outputs, code, files, attachments, metadata, tool calls, telemetry, transcripts, caches, backups, support or abuse monitoring, human access, derived data, and deletion/export behavior.

The proposed scope is limited to `PUBLIC` repository metadata. It does not permit secrets, customer data, production data, repository contents beyond public metadata, or any provider submission while the route remains denied.

## Route scope decision

Record permitted roles, purposes, task families, data classes, tools, network profile, sandbox, credentials, quotas, concurrency, cost class, failover, evaluation, limitations, expiry, and review trigger.

No role, task family, data class, tool, network profile, credential, quota, failover, or runtime use is approved. `PUBLIC` repository metadata is a proposal for review, not an allowed data class.

## Required tests and evidence

Include identity readback, configuration digest, data-class denial, hidden-tool or subagent denial, invalid-task rejection, quota and hard-stop behavior, failure recovery, output-contract validation, independent review, and evidence bundle.

These tests and the independent review have not been completed.

## Approval

Only the Sole Human Owner may change `current_route_decision` from `DENY`. Approval applies only to the exact route identity and frozen scope.

## Evidence status as of 2026-09-01

- Selected candidate: OpenAI Codex / ChatGPT Codex Connector
- Owner-supplied selection evidence: Chrome-observed ChatGPT Codex Connector installation, identifier `141917362`
- Selection evidence independently verified: no
- Exact account/service identity: `TBD`
- Exact model and revision/digest: `TBD`
- CLI/runtime and adapter: `TBD`
- Data-policy sources reviewed: none
- Proposed data scope: `PUBLIC` repository metadata only; not approved
- Secrets, customer data, and production data: prohibited
- Approved roles: none
- Allowed data classes: none
- Independent review: required and not completed
- Evaluation: not started
- Route lifecycle: `UNDER_POLICY_REVIEW`
- Dispatch: prohibited
