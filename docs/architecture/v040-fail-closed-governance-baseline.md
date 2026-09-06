---
document_id: ARC-V040-FCGOV-001
title: V0.4.0 Fail-Closed Governance and Safety Boundary Baseline
status: DRAFT
lifecycle: DRAFT
version: 0.1.0
milestone: v0.4.0
release: v0.4.0 - OSHE Inspect Private Alpha
owner: Sole Human Owner
authority:
  - HDEC-V040-FOUNDATION-054
  - HDEC-V040-SCORING-058
gates:
  - H040-004
  - H040-007
  - H040-008
  - H040-009
  - H040-010
  - H040-011
issue: 138
packet: V040-I027
created_at: 2026-09-06T07:08:00+07:00
updated_at: 2026-09-06T07:08:00+07:00
---

# V0.4.0 Fail-Closed Governance and Safety Boundary Baseline

## 1. Context & Authority

Under approved Sole Human Owner decisions **HDEC-V040-FOUNDATION-054** (Architecture Baseline & Gate Controls) and **HDEC-V040-SCORING-058** (Deterministic Operational Scoring Engine), this specification establishes authoritative runtime boundaries, priority enforcement rules, deferred override protocols, and autonomous AI restrictions for Milestone `v0.4.0 OSHE Inspect Private Alpha`.

All logic specified herein operates exclusively in-memory on local synthetic data fixtures. No production credentials, external network routing, cloud resources, or customer records are used. Gates `H040-007` through `H040-011` remain on strict `HOLD`.

---

## 2. Core Governance Invariants

### Invariant 1: Strict Fail-Closed Priority Hierarchy

When qualifying state transitions toward conclusive or protected states (`APPROVED`, `CLOSED`, `REMEDIATED`, `PUBLISHED`, `SCORED_PASS`), safety evaluations enforce the following non-negotiable deterministic precedence:

$$\text{Manual Override Attempt (Denied)} \succ \text{AI Boundary Violation (Blocked)} \succ \text{Critical Fail (CF1)} \succ \text{Unknown Quarantine (U1)} \succ \text{Score Threshold (8000 bps)} \succ \text{Standard Business Rules}$$

1. **Manual Override Attempt**: Unconditionally evaluated first; always fails closed with `DenialManualOverrideDeferred` under Gate `H040-004`.
2. **Autonomous AI Boundary**: AI actors attempting protected transitions are blocked with `DenialAutonomousAIBoundary`.
3. **Critical Fail Priority (CF1)**: Any active, unresolved critical failure unconditionally locks the transition outcome to `DENIED` with `DenialCriticalFailActive` (`CRITICAL_FAIL_PRIORITY`), regardless of high numerical scores (e.g., 99.00%) or unreviewed unknowns.
4. **Unknown Quarantine (U1)**: Any unresolved `UNKNOWN` response quarantines transitions to conclusive states with `RuleResultQuarantined` and `DenialUnknownQuarantined` (`UNKNOWN_QUARANTINE`).
5. **Score Threshold**: Scored inspection transitions require reaching or exceeding the passing threshold of $80.00\%$ ($8000$ basis points). Scores below threshold fail closed with `DenialMissingCondition` (`SCORE_THRESHOLD`).
6. **Standard RuleMatrix Predicates**: Underlying business rules, actor roles, required preconditions, and evidence attachment counts are qualified only after all higher-priority invariants pass.

### Invariant 2: Fail-Closed Unknown Quarantine

An `UNKNOWN` checklist response represents an unverified physical hazard or missing inspection data. In accordance with `U1_QUARANTINE_DENOMINATOR` under `HDEC-V040-SCORING-058`:
- Unresolved `UNKNOWN` items quarantine the target entity.
- The entity cannot transition to conclusive passing states (`APPROVED`, `CLOSED`, `SCORED_PASS`).
- Resolution requires explicit, authorized human supervisory action (`ResolveUnknownQuestion`) with documented physical re-inspection rationale.
- Autonomous AI actors are prohibited from resolving quarantined items.

### Invariant 3: Deferred Manual Override Boundary (Gate H040-004)

Under Gate `H040-004`, manual override capability is a **HUMAN_OWNED_UNSELECTED** authority boundary:
- No runtime bypass, exception grant, or forced approval mechanism exists or may be implemented.
- Any transition request setting `IsOverrideAttempt = true` is unconditionally rejected with denial code `DenialManualOverrideDeferred`.
- Every override attempt is permanently recorded in the append-only audit ledger with actor identity, role, timestamp, requested transition, and provided rationale.
- Zero administrative, executive, or supervisory authority can bypass this gate prior to formal human owner ratification.

### Invariant 4: Autonomous AI Boundaries

Autonomous AI agents, automated bots, LLM workflows, and system engines are strictly prohibited from:
- Clearing verified critical failures.
- Resolving quarantined `UNKNOWN` inspection responses.
- Exercising manual overrides.
- Authorizing transitions into protected states (`APPROVED`, `CLOSED`, `REMEDIATED`).

Any attempt by an AI actor role (`AI`, `AI_AGENT`, `ENGINEERING_AGENT`, `SYSTEM_AGENT`, `AUTONOMOUS_AGENT`, `LLM`) immediately fails closed with `DenialAutonomousAIBoundary`.

### Invariant 5: Append-Only Audit Protection

All qualification evaluations, denials, override attempts, critical registrations, and supervisory resolutions are recorded in a tamper-evident, append-only in-memory ledger:
- Strictly monotonic sequence numbering ($1, 2, 3, \dots$).
- Canonical UTC timestamps.
- Explicit recording of tenant ID, target ID, target kind, actor ID, actor role, action code, denial code, explanation, and correlation ID.
- Complete audit history retrieval per tenant and target entity.

---

## 3. Synthetic Verification Scenarios

| Scenario ID | Condition | Expected Result | Priority Applied | Denial Code |
| :--- | :--- | :--- | :--- | :--- |
| **SYN-FC-01** | Score 95%, active critical fail | `DENIED` | `CRITICAL_FAIL_PRIORITY` | `CRITICAL_FAIL_ACTIVE` |
| **SYN-FC-02** | Score 92%, unresolved UNKNOWN | `QUARANTINED` | `UNKNOWN_QUARANTINE` | `UNKNOWN_RESPONSE_QUARANTINED` |
| **SYN-FC-03** | Active critical fail AND unresolved UNKNOWN | `DENIED` | `CRITICAL_FAIL_PRIORITY` | `CRITICAL_FAIL_ACTIVE` |
| **SYN-FC-04** | Manual override attempt (any actor) | `DENIED` | `MANUAL_OVERRIDE_DENIED` | `MANUAL_OVERRIDE_DEFERRED` |
| **SYN-FC-05** | Autonomous AI attempting protected transition | `DENIED` | `AI_BOUNDARY_DENIED` | `AUTONOMOUS_AI_BOUNDARY_EXCEEDED` |
| **SYN-FC-06** | Human supervisor resolves UNKNOWN, score 85% | `PERMITTED` | `STANDARD_PERMITTED` | `NONE` |
| **SYN-FC-07** | Monotonic sequence audit verification | All entries recorded | N/A | Monotonic sequence 1..N |
