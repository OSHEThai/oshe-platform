---
evidence_id: V010-I023-CODEX-PUBLIC-PREFLIGHT-397
title: V010-I023 Codex Public Preflight Evidence
document_type: technical_preflight_evidence
document_version: 1.0.0
lifecycle_status: RECORDED
status: COMPLETED
recorded_date: '2026-09-04'
authority: Sole Human Owner / LEASE-V010-I023-CODEX-PUBLIC-PREFLIGHT-397
assignment_id: ASN-V010-I023-CODEX-PUBLIC-PREFLIGHT-397
route_id: route-openai-codex-candidate
model_record_id: model-openai-codex-candidate
data_class: PUBLIC
cost_class: C2
quota_profile_id: QP-V010-LIVE-QUALIFICATION-C2-001
exit_code: 0
response_token: OSHE_V010_PUBLIC_PREFLIGHT_OK
observed_provider: openai
observed_cli: codex-cli 0.152.0
observed_session_model: gpt-5.6-luna
qualification_credit: NONE
activation_credit: NONE
---

# V010-I023 Codex Public Preflight Evidence

## 1. Authorization and Governance Context

- **Authority**: Sole Human Owner recorded approval in OSHEThai/oshe-platform Issue #26 comments 5538290484, 5538296416, and 5538304684.
- **Lease and Assignment**: LEASE-V010-I023-CODEX-PUBLIC-PREFLIGHT-397 / ASN-V010-I023-CODEX-PUBLIC-PREFLIGHT-397.
- **Boundaries**: Exactly one PUBLIC-only technical connectivity preflight through the existing authenticated Codex CLI session.
- **Cost Class & Quota**: Cost class `C2`, quota profile `QP-V010-LIVE-QUALIFICATION-C2-001`, external concurrency limit 1, 70% soft warning, 100% hard stop.
- **Credit Boundary**: Technical connectivity evidence only. No provider route or model qualification, approval, activation, dispatch, live mission, release, deployment, or Issue-closure credit is granted or claimed.

## 2. Execution Parameters

- **Command**:
  ```bash
  codex exec --ephemeral --skip-git-repo-check -C C:/Windows/Temp -s read-only --color never "Return exactly OSHE_V010_PUBLIC_PREFLIGHT_OK. Do not use tools, read files, inspect directories, run commands, or provide any other text."
  ```
- **Execution Mode**: Ephemeral CLI session (`--ephemeral`), skipping git repository check (`--skip-git-repo-check`).
- **Sandbox**: Read-only sandbox (`-s read-only`).
- **Working Directory**: `C:/Windows/Temp` (`-C C:/Windows/Temp`).
- **Data Scope**: Fixed public prompt token only (`PUBLIC`). Zero repository, customer, production, secret, credential, or account data submitted.
- **Tool / Command Restrictions**: Prohibited tools, file reads, directory inspection, command execution, subagents, memory, and web access.
- **Retry / Fallback**: Exactly one operation; zero retries; zero fallbacks.

## 3. Observed Results

- **Exit Code**: `0` (Success).
- **Exact Response Token**: `OSHE_V010_PUBLIC_PREFLIGHT_OK`.
- **Observed Provider**: `openai`.
- **Observed CLI**: `codex-cli 0.152.0`.
- **Observed Runtime Session Model**: `gpt-5.6-luna`.
- **Observed Sandbox**: `read-only`.
- **Observed Working Directory**: `C:\Windows\Temp`.

## 4. Limitations and Non-Activation Controls

- **Observation Only**: The observed runtime model `gpt-5.6-luna` is recorded as factual execution evidence only. It is not an approved registry selection, model configuration update, or qualification approval.
- **Unresolved Gates**: Exact provider/service legal account identity, service tier, endpoint, model revision or digest, configuration digest, adapter version, independent policy review, technical validation suite, and H010-007 activation authority remain unresolved (`TBD`).
- **Preserved Status**:
  - Route decision remains `DENY`.
  - Dispatch default remains `DENY`.
  - Route lifecycle status remains `UNDER_POLICY_REVIEW`.
  - `dispatch_enabled` remains `false`.
  - `active_route_ids`, `approved_route_ids`, and `enabled_route_ids` remain empty (`[]`).
  - `enabled_model_record_ids` remains empty (`[]`).
