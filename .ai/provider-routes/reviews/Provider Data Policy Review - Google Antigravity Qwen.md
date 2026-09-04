---
document_id: PDR-ANTIGRAVITY-QWEN-001
title: Provider Data Policy Review - Google Antigravity Qwen
document_type: provider_data_policy_review
document_version: 1.0.1
lifecycle_status: UNDER_REVIEW
maturity: DETAILED
implementation_status: IMPLEMENTED_FOR_POLICY_REVIEW
review_status: REVIEW_REQUIRED
owner: Security Privacy and Product Safety Agent
approver: Sole Human Owner / Human Product and Release Authority
applicable_routes:
- route-google-antigravity-qwen3.6-35b-a3b-candidate
effective_date: null
source_of_truth: LOCAL_REPOSITORY
classification: INTERNAL
change_risk: R3
current_route_decision: DENY
owner_authority_ref: H010-004 / Issue-26-comment-5536206761 / Issue-26-comment-5536683519
terms_status: OWNER_ACCEPTED_PENDING_ROUTE_QUALIFICATION
---

# Provider Data Policy Review - Google Antigravity Qwen

## Current status

- Sole Human Owner correction: Google Antigravity via Gemini CLI is the secondary
  client/service candidate, with `qwen3.6-35b-a3b` selected.
- H010-004 policy scope: `PUBLIC` only, limited to repository metadata; terms are
  owner-accepted for the applicable service pending route qualification.
- Current route decision: `DENY`; dispatch is prohibited. The active-lease independent policy review is completed with `DENY_MAINTAINED`.

No route is approved. Unknown or conflicting facts fail closed.

## Identity and authoritative sources

Local readback recorded Gemini CLI `0.57.0` and the selected model
`qwen3.6-35b-a3b`. It is not route qualification or provider invocation.

Reviewed official sources on 2026-09-04:

- [Google Antigravity CLI installation](https://www.antigravity.google/docs/cli/install/)
- [Google Antigravity Terms](https://antigravity.google/terms)

Before any route activation, confirm applicable policy scope, technical qualification,
evaluation evidence, and the owner activation decision. Do not collect account,
authentication, tier, endpoint/region, model revision/digest, configuration-digest, or
adapter-identity metadata under this review.

## Data and control boundary

The only proposed data class is `PUBLIC`, limited to repository metadata. Secrets,
customer data, production data, and every other data class are excluded. No roles,
tasks, tools, credentials, network access, provider-native tools, memory, web access,
subagents, fallback route, quota, or cost authority is approved.

## Required evidence before reconsideration

The active-lease [Independent Policy Review - Google Antigravity Qwen R2](Independent%20Policy%20Review%20-%20Google%20Antigravity%20Qwen%20R2.md) is complete with `DENY_MAINTAINED`. Data-class denial tests, hidden-tool and subagent denial tests, task rejection, quota/hard-stop behavior, output-contract validation, technical qualification, evaluation evidence, and the required owner activation decision remain required. Until then, retain `DENY`.

## Historical lineage

[Provider Data Policy Review - Google Gemini.md](Provider%20Data%20Policy%20Review%20-%20Google%20Gemini.md)
is retained as a superseded interpretation. It is not an adopted route or approval.
