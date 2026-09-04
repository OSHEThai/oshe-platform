---
evidence_id: V010-I023-PUBLIC-TECHNICAL-RESPONSE-R2
title: V010-I023 PUBLIC Technical Response Evidence R2
document_type: technical_response_evidence
document_version: 1.0.0
lifecycle_status: RECORDED
status: COMPLETED
recorded_date: '2026-09-04'
authority: Sole Human Owner / HDEC-V010-I023-PUBLIC-TECHNICAL-QUALIFICATION-050
assignment_id: ASN-V010-I023-PUBLIC-TECHNICAL-QUALIFICATION-466
lease_id: LEASE-V010-I023-PUBLIC-TECHNICAL-QUALIFICATION-466
route_ids:
  - route-openai-codex-candidate
  - route-google-antigravity-qwen3.6-35b-a3b-candidate
data_class: PUBLIC
cost_class: C2
qualification_credit: TECHNICAL_RESPONSE_EVIDENCE_ONLY
activation_credit: NONE
---

# V010-I023 PUBLIC Technical Response Evidence R2

## Scope

The Sole Human Owner authorized exactly two sequential, PUBLIC-only response checks under HDEC-V010-I023-PUBLIC-TECHNICAL-QUALIFICATION-050. The active lease permitted exactly four commands: two local version inspections and one ephemeral response operation for each candidate route. No source, repository, customer, production, credential, account, payment, subscription, model, or CLI-setting data was supplied or changed.

## Observed commands and results

| Order | Command class | Result |
| --- | --- | --- |
| 1 | `codex --version` from `C:\Windows\Temp` | Exit `0`; `codex-cli 0.152.0`. |
| 2 | `agy --version` from `C:\Windows\Temp` | Exit `0`; `1.1.26`. |
| 3 | Ephemeral Codex response with `read-only` sandbox | Exit `0`; returned exactly `{"case_id":"V010_I023_CODEX_TECH_QUAL_R2","classification":"PUBLIC","status":"OK"}`. The CLI identified provider `openai`, runtime model `gpt-5.6-luna`, working directory `C:\Windows\Temp`, and read-only sandbox. |
| 4 | Sandboxed Antigravity response with slash-command expansion disabled | Exit `0`; returned exactly `{"case_id":"V010_I023_ANTIGRAVITY_TECH_QUAL_R2","classification":"PUBLIC","status":"OK"}`. The CLI warned that `--mode plan has no effect while slash command expansion is disabled`. |

The two response operations were run one at a time. No retry or fallback was performed.

## Evidence interpretation

This is reproducible technical-response evidence: both existing authenticated sessions returned the required PUBLIC JSON object under the recorded restrictions. The Codex observation is factual runtime identity evidence for that one execution. The Antigravity response proves response connectivity and JSON adherence only; its plan-mode warning means this evidence does not prove plan-mode enforcement.

## Non-activation boundary

This record does not resolve account profile, service tier, endpoint, model revision or digest, configuration digest, adapter identity, actual quota consumption, full evaluation, or all policy and activation prerequisites. It does not approve, enable, configure, activate, dispatch, or qualify either route, and it does not support a live mission, release, deployment, signing, or Issue #26 closure by itself.
