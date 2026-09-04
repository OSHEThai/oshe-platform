---
document_id: PDR-GOOGLE-GEMINI-001
title: Provider Data Policy Review - Google Gemini
document_type: provider_data_policy_review
document_version: 1.0.0
lifecycle_status: UNDER_REVIEW
maturity: DETAILED
implementation_status: IMPLEMENTED_FOR_POLICY_REVIEW
review_status: REVIEW_REQUIRED
owner: Security Privacy and Product Safety Agent
approver: Sole Human Owner / Human Product and Release Authority
applicable_routes:
- route-google-gemini-candidate
effective_date: null
source_of_truth: LOCAL_REPOSITORY
classification: INTERNAL
change_risk: R3
current_route_decision: DENY
owner_authority_ref: H010-004 / Issue-26-comment-5536206761
terms_status: OWNER_ACCEPTED_PENDING_EXACT_IDENTITY
---

# Provider Data Policy Review - Google Gemini

## Current status

- Sole Human Owner H010-004: Gemini selected as secondary; terms accepted for the applicable route; `PUBLIC` only
- Current route decision: `DENY` pending exact identity, independent review, and activation evidence
- Dispatch: prohibited

No route is approved. Unknown or conflicting facts fail closed.

## Exact route identity required

Record provider/service legal identity, account or local-host profile, service tier, region or host, endpoint, exact model ID, revision/digest, CLI/runtime, adapter, authentication mode, configuration digest, and intended use.

## Authoritative evidence required

Reviewed official source (read on 2026-09-04): [Gemini CLI Terms of Service and Privacy Notice](https://google-gemini.github.io/gemini-cli/docs/tos-privacy.html). The applicable terms and data handling depend on the sign-in/authentication method; the exact account and authentication method remain unresolved.

Record exact sources and effective versions for terms, data processing, security/privacy, retention, training or service-improvement use, regions, subprocessors, support access, logging, deletion, export, encryption, incident handling, administrative settings, local package provenance, licensing, host trust, and update behavior.

## Data handling assessment

Assess prompts, outputs, code, files, attachments, metadata, tool calls, telemetry, transcripts, caches, backups, support or abuse monitoring, human access, derived data, and deletion/export behavior.

## Route scope decision

Record permitted roles, purposes, task families, data classes, tools, network profile, sandbox, credentials, quotas, concurrency, cost class, failover, evaluation, limitations, expiry, and review trigger.

## Required tests and evidence

Include identity readback, configuration digest, data-class denial, hidden-tool or subagent denial, invalid-task rejection, quota and hard-stop behavior, failure recovery, output-contract validation, independent review, and evidence bundle.

## Approval

The Sole Human Owner recorded H010-004 approval in Issue #26 comment 5536206761: Gemini is secondary, terms are accepted for the applicable route, and `PUBLIC` is the only permitted proposed data class. `current_route_decision` remains `DENY` until exact route identity and all technical/evidence gates are complete.

## Evidence status as of 2026-08-18

- Exact account/service identity: `TBD`
- Exact model and revision/digest: `TBD`
- CLI/runtime and adapter: `TBD`
- Data-policy sources reviewed: none
- Approved roles: none
- Allowed data classes: none
- Evaluation: not started
- Dispatch: prohibited
