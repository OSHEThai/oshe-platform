---
document_id: MPT-ROUTE-001
title: AI Service Candidate Routing, Quota, and Failover Matrix
document_type: provider_route_matrix
document_version: 1.0.0
lifecycle_status: APPROVED
maturity: OPERATIONAL
implementation_status: IMPLEMENTED_FOR_PLANNING_CONTROL
review_status: APPROVED
owner: Project Management Agent
approver: Sole Human Owner / Human Product and Release Authority
effective_date: '2026-08-18'
source_of_truth: GOOGLE_DRIVE
classification: INTERNAL
change_risk: R3
---

# AI Service Candidate Routing, Quota, and Failover Matrix

## Control statement

This matrix records candidate service lanes only. It does not authorize dispatch. Exact routes remain disabled until provider policy, identity, technical validation, evaluation, usage controls, independent review, and Sole Human Owner approval are complete.

## Candidate service lanes

| Route ID | Candidate family | Current state | Dispatch | Allowed data |
|---|---|---|---|---|
| `route-qwen-local-candidate` | Qwen local runtime | `INTAKE_INCOMPLETE` | Disabled | None |
| `route-openai-codex-candidate` | OpenAI Codex | `INTAKE_INCOMPLETE` | Disabled | None |
| `route-google-gemini-candidate` | Google Gemini | `INTAKE_INCOMPLETE` | Disabled | None |
| `route-anthropic-claude-candidate` | Anthropic Claude | `INTAKE_INCOMPLETE` | Disabled | None |
| `route-deepseek-pro-api-candidate` | DeepSeek specialist API lane | `INTAKE_INCOMPLETE` | Disabled | None |
| `route-deepseek-flash-api-candidate` | DeepSeek fast or bulk API lane | `INTAKE_INCOMPLETE` | Disabled | None |

Candidate labels are planning families, not exact provider routes or approved products.

## Canonical role routing

All twelve canonical roles are currently `UNASSIGNED` in `provider-routing.yaml`. No primary or fallback route is active. Role-to-route assignment must resolve the exact mission role, task family, data class, risk, assurance, tools, network, quota, review, and evidence requirements.

## Effort classes

- `E0` — bounded research, inventory, comparison, or documentation update.
- `E1` — small mission with one primary deliverable and deterministic verification.
- `E2` — multi-artifact or small implementation mission with moderate dependencies.
- `E3` — topic-level package requiring architecture, implementation, assurance, testing, and evidence.
- `E4` — release-level program with multiple dependent topics and a final human gate.

## Variable usage and quota control

No fixed monetary budget, provider quota, or percentage threshold is approved. Each exact route must define its own source of quota data, cost class, concurrency, warning condition, hard stop, usage evidence, and human escalation.

Unknown pricing or quota remains `UNKNOWN`. It is never replaced by an invented currency value or threshold.

## Failover

Failover is prohibited until the fallback route is independently qualified for equivalent or stricter scope. Failover may not change data class, risk, assurance, role, task contract, tools, path authority, review, evidence, quota, or human approval requirements.
