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
owner_authority_ref: H010-004 / Issue-26-comment-5536206761
terms_status: OWNER_ACCEPTED_PENDING_EXACT_IDENTITY
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
- Sole Human Owner H010-004: Codex selected as primary; terms accepted for the applicable route; `PUBLIC` only
- Current route decision: `DENY` pending exact identity, independent review, and activation evidence
- Dispatch: prohibited

No route is approved. Unknown or conflicting facts fail closed.

## Exact route identity required

Record provider/service legal identity, account or local-host profile, service tier, region or host, endpoint, exact model ID, revision/digest, CLI/runtime, adapter, authentication mode, configuration digest, and intended use.

The selected-candidate record and connector identifier are intake evidence only. Every exact provider-identity field remains unresolved.

## Authoritative evidence required

Record exact sources and effective versions for terms, data processing, security/privacy, retention, training or service-improvement use, regions, subprocessors, support access, logging, deletion, export, encryption, incident handling, administrative settings, local package provenance, licensing, host trust, and update behavior.

Reviewed official sources (read on 2026-09-04):

- [Using Codex with your ChatGPT plan](https://help.openai.com/en/articles/11369540-using-codex-with-your-chatgpt-plan): the applicable terms/privacy basis depends on the sign-in plan or agreement.
- [How your data is used to improve model performance](https://help.openai.com/en/articles/5722486-how-your-data-is-used-to-improve-model-performance): data controls apply to Codex tasks.

The exact sign-in plan, account identity, model, and runtime configuration remain unresolved. The owner acceptance is therefore bounded to `PUBLIC` repository metadata and cannot enable dispatch.

## Data handling assessment

Assess prompts, outputs, code, files, attachments, metadata, tool calls, telemetry, transcripts, caches, backups, support or abuse monitoring, human access, derived data, and deletion/export behavior.

The proposed scope is limited to `PUBLIC` repository metadata. It does not permit secrets, customer data, production data, repository contents beyond public metadata, or any provider submission while the route remains denied.

## Route scope decision

Record permitted roles, purposes, task families, data classes, tools, network profile, sandbox, credentials, quotas, concurrency, cost class, failover, evaluation, limitations, expiry, and review trigger.

No role, task family, data class, tool, network profile, credential, quota, failover, or runtime use is approved. `PUBLIC` repository metadata is a proposal for review, not an allowed data class.

## Required tests and evidence

Include identity readback, configuration digest, data-class denial, hidden-tool or subagent denial, invalid-task rejection, quota and hard-stop behavior, failure recovery, output-contract validation, independent review, and evidence bundle.

A single ephemeral, read-only PUBLIC-only connectivity preflight was executed under Sole Human Owner authorization (LEASE-V010-I023-CODEX-PUBLIC-PREFLIGHT-397) and exited 0 with token `OSHE_V010_PUBLIC_PREFLIGHT_OK` (recorded in [V010-I023-CODEX-PUBLIC-PREFLIGHT-397](../evidence/V010-I023-CODEX-PUBLIC-PREFLIGHT-397.md)). The observed runtime session model was `gpt-5.6-luna`. Full route qualification tests, independent review, and activation gates remain incomplete.

## Approval

The Sole Human Owner recorded H010-004 approval in Issue #26 comment 5536206761: Codex is primary, terms are accepted for the applicable route, and `PUBLIC` is the only permitted proposed data class. `current_route_decision` remains `DENY` until exact route identity and all technical/evidence gates are complete.

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
- Technical preflight evidence: [V010-I023-CODEX-PUBLIC-PREFLIGHT-397](../evidence/V010-I023-CODEX-PUBLIC-PREFLIGHT-397.md) (ephemeral read-only exit 0; observed session model gpt-5.6-luna; no qualification or activation credit)
