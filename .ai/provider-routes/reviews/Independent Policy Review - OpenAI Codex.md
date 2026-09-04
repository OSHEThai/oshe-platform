---
document_id: IPR-OPENAI-CODEX-001
title: Independent Policy Review - OpenAI Codex
document_type: independent_policy_review
document_version: 1.0.0
lifecycle_status: COMPLETED
review_disposition: DENY_MAINTAINED
reviewer: Security, Privacy and Product Safety Lead
reviewer_pane: w9:p13
applicable_route: route-openai-codex-candidate
reviewed_date: '2026-09-04'
---

# Independent Policy Review - OpenAI Codex

## Scope and sources

This review covers only the current OpenAI Codex candidate and the proposed `PUBLIC` repository-metadata boundary. It relies on the PM-read official sources:

- https://help.openai.com/en/articles/11369540-codex-in-chatgpt
- https://help.openai.com/en/articles/5722486-chatgpt-privacy-policies

## Finding

For individual/consumer Codex use, submitted content may be used to improve models depending on applicable data controls. This finding does not approve a route, data class, dispatch, credential, account, model, CLI, or configuration.

## Disposition

`DENY_MAINTAINED`. Keep the candidate default-deny and disabled. The proposed `PUBLIC` scope remains a boundary for review, not an enabled data-class permission.

## Unresolved gates

- Exact account, legal service, tier, endpoint, and region identity.
- Model revision/digest and configuration digest.
- Technical validation and evaluation evidence.
- Conditions of H010-007; it is conditionally owner-approved and cannot take effect until all prerequisites pass.

## Credit boundary

This is an independent policy review only. It performs and authorizes no provider invocation, configuration, qualification, activation, dispatch, mission, release, deployment, or Issue closure.
