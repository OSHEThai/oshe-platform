---
document_id: IPR-ANTIGRAVITY-QWEN-002
title: Independent Policy Review - Google Antigravity Qwen R2
document_type: independent_policy_review
document_version: 1.0.0
lifecycle_status: COMPLETED
review_disposition: DENY_MAINTAINED
reviewer: Security, Privacy and Product Safety Lead
reviewer_pane: w9:p13
applicable_route: route-google-antigravity-qwen3.6-35b-a3b-candidate
reviewed_date: '2026-09-04'
predecessor_document_id: IPR-ANTIGRAVITY-QWEN-001
---

# Independent Policy Review - Google Antigravity Qwen R2

## Scope and sources

This active-lease successor review covers only the Google Antigravity via Gemini CLI / `qwen3.6-35b-a3b` candidate and the proposed `PUBLIC` repository-metadata boundary. It relies on PM-read official sources:

- https://antigravity.google/terms
- https://antigravity.google/docs/faq/
- https://antigravity.google/docs/cli/install/

## Findings

The Antigravity terms state that user interaction data and related metadata are recorded and stored, and that interactions may be used to evaluate, develop, and improve services. They identify third-party tool access as a terms risk. Exact client/runtime identity must be reconciled before provider submission. These findings do not approve a route, data class, dispatch, credential, account, model, CLI, or configuration.

## Disposition

`DENY_MAINTAINED`. Keep the candidate default-deny and disabled. The proposed `PUBLIC` scope remains a review boundary, not an enabled data-class permission.

## Unresolved gates

- Exact account, legal service, tier, endpoint, model revision, and configuration identity.
- Technical validation and evaluation evidence.
- Conditions of H010-007; it is conditionally owner-approved and cannot take effect until all prerequisites pass.

## Credit boundary

This active-lease independent policy review performs and authorizes no provider invocation, configuration, qualification, activation, dispatch, mission, release, deployment, or Issue closure.
