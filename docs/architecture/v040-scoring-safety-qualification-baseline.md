---
document_id: QLF-V040-SCORING-SAFETY-001
title: V0.4.0 Deterministic Scoring and Product-Safety Qualification Baseline
governing_issue: 139
assignment_id: ASN-V040-I028-SCORING-SAFETY-QUALIFICATION-002
lease_id: LEASE-V040-I028-SCORING-SAFETY-QUALIFICATION-002
status: APPROVED
lifecycle: APPROVED
author: Test and Quality Lead
version: 1.0.0
date: 2026-09-06
target_milestone: v0.4.0
synthetic_scenario_id: fix_syn_scoring_safety_qualification_v1
governing_decisions:
  - HDEC-V040-FOUNDATION-054
  - HDEC-V040-SCORING-058
retained_holds:
  - H040-007
  - H040-008
  - H040-009
  - H040-010
  - H040-011
---

# V0.4.0 Deterministic Scoring and Product-Safety Qualification Baseline

## 1. Executive Summary & Purpose

This document establishes the governed qualification baseline for deterministic inspection scoring and product-safety boundary enforcement in Milestone `v0.4.0 OSHE Inspect Private Alpha` under Issue #139 (`V040-I028`).

Under approved Sole Human Owner decisions **HDEC-V040-FOUNDATION-054** and **HDEC-V040-SCORING-058**, this qualification baseline defines the synthetic test matrix and automated verification invariants governing:
1. Normalized Model 2 weighted scoring (`MODEL_2_WEIGHTED`).
2. Not Applicable (`NA`) denominator subtraction with zero negative score penalty.
3. Unknown quarantine handling (`U1_QUARANTINE_DENOMINATOR`) requiring human supervisory resolution.
4. Integer basis points representation and round half-up arithmetic (`R1_ROUND_HALF_UP`).
5. Critical-fail priority flag enforcement (`CF1_PRIORITY_FLAG`) with unmasked score reporting.
6. The three mandatory pass predicates and deterministic 80.00% ($8000$ basis points) threshold.
7. Strict priority hierarchy: `critical > UNKNOWN > score`.
8. Deferred manual override boundary under Gate `H040-004` (unconditional denial and append-only audit).
9. Autonomous AI boundary enforcement (strict prohibition against protected state transitions).
10. Derived reporting non-authority designation (`DERIVED_OUTPUT_NON_AUTHORITY`) ensuring strict architectural separation between operational business state (`MOD-WFA`) and reporting projections (`MOD-REP`).

All synthetic qualification fixtures and assertions operate strictly within tenant-scoped synthetic boundaries (`fix_syn_scoring_safety_qualification_v1`). Zero customer data, zero production endpoints, zero authority grants, and zero hold lifts are introduced. Foundation holds `H040-007` through `H040-011` remain strictly on `HOLD`.

---

## 2. Core Safety Qualification Invariants

### Invariant 1: Model 2 Weighted Normalized Scoring (`MODEL_2_WEIGHTED`)
- Section scores are calculated as earned points divided by active effective denominator.
- Total inspection score is the sum of section scores weighted by normalized section weights ($\sum w_i = 1.0$).
- If all questions in a section evaluate to NA or UNKNOWN, that section is marked inactive and its weight is dynamically redistributed proportionally across active sections.

### Invariant 2: Denominator Subtractions for NA and Exclusions
- Not Applicable (`NA` / `NOT_APPLICABLE`) responses subtract the question's maximum points from the section denominator. NA responses carry zero negative penalty.
- Informational and non-scored question types (`TEXT_NOTE`, `EVIDENCE_ATTACHMENT`) and conditionally excluded items are excluded from the denominator.

### Invariant 3: Unknown Quarantine (`U1_QUARANTINE_DENOMINATOR`)
- Questions marked `UNKNOWN` quarantine available points from the effective denominator.
- The presence of any unreviewed `UNKNOWN` response immediately blocks conclusive passing determinations, setting the compliance outcome to `PROVISIONAL_PENDING_UNKNOWN_RESOLUTION`.
- Only an authorized human supervisor may resolve an UNKNOWN quarantine with recorded classification rationale. Autonomous AI agents are strictly prohibited from resolving quarantines.

### Invariant 4: Integer Basis Points & Round Half Up (`R1_ROUND_HALF_UP`)
- All calculated scores map deterministically to integer basis points:
  $$\text{basis\_points} = \lfloor \text{raw\_weighted\_score} \times 10000 + 0.5 + 10^{-9} \rfloor$$
  where $1\% = 100\text{ bps}$, $80.00\% = 8000\text{ bps}$, and $100.00\% = 10000\text{ bps}$.
- Exact boundary behavior:
  - $79.995\% \to 8000\text{ bps}$ ($80.00\%$) $\implies$ Satisfies score threshold.
  - $79.994\% \to 7999\text{ bps}$ ($79.99\%$) $\implies$ Fails score threshold.

### Invariant 5: Critical Fail Priority Flag (`CF1_PRIORITY_FLAG`)
- A critical failure trigger immediately locks the compliance outcome to `NON_COMPLIANT_CRITICAL`.
- Numerical score calculation is reported transparently without masking or suppression, allowing full diagnostic visibility while preventing compliance signoff.

### Invariant 6: Three Mandatory Pass Predicates
Passing compliance (`OutcomePass`) strictly requires all three predicates to evaluate to `true`:
1. `NoCriticalFailPresent`: Zero active critical failure conditions.
2. `NoUnresolvedUnknownQuarantined`: Zero unreviewed quarantined UNKNOWN responses.
3. `ScoreThresholdSatisfied`: Calculated score $\ge 8000\text{ bps}$ ($80.00\%$).

### Invariant 7: Priority Hierarchy (`critical > UNKNOWN > score`)
State transition evaluations through `FailClosedGovernor` enforce the strict priority sequence:
$$\text{Manual Override Denial} \succ \text{AI Boundary Denial} \succ \text{Critical Fail (CF1)} \succ \text{Unknown Quarantine (U1)} \succ \text{Score Threshold}$$
- An unresolved critical failure dominates an UNKNOWN quarantine and a high score.
- An unreviewed UNKNOWN response dominates a high score.

### Invariant 8: Deferred Manual Override Boundary (`H040-004`)
- Under Gate `H040-004`, manual override authority is deferred human-owned.
- Any manual override attempt is unconditionally denied (`DenialManualOverrideDeferred`) and logged immutably in the append-only audit ledger. Zero override authority is granted.

### Invariant 9: Autonomous AI Boundary
- Autonomous AI roles (`AI`, `AI_AGENT`, `ENGINEERING_AGENT`, `SYSTEM_AGENT`, `AUTONOMOUS_AGENT`, `LLM`) are strictly prohibited from authorizing transitions to protected states (`APPROVED`, `CLOSED`, `REMEDIATED`, `PUBLISHED`, `SCORED_PASS`) or clearing critical findings/quarantines.
- All AI bypass attempts fail closed with denial code `DenialAutonomousAIBoundary`.

### Invariant 10: Derived Reporting Non-Authority (`DERIVED_OUTPUT_NON_AUTHORITY`)
- Metrics in `MOD-REP` (such as `metric_inspection_compliance_score`) are non-authoritative derived projections (`NonAuthority: true`).
- All reporting query results include the mandatory header notice:
  `DERIVED_OUTPUT_NON_AUTHORITY: Reports, metrics, and analytics are derived outputs and never constitute operational authority or replace authoritative records.`
- Authoritative operational lifecycle state remains strictly encapsulated in `MOD-WFA`.

---

## 3. Synthetic Qualification Scenario Matrix (`fix_syn_scoring_safety_qualification_v1`)

| Scenario ID | Test Domain | Description | Expected Outcome | Governed Invariant |
|---|---|---|---|---|
| `SYN-SCORING-01` | Model 2 Weighted | Section 1 (0.60 wt, 80/100) + Section 2 (0.40 wt, 45/50) | 8400 bps (84.00%), PASS | Invariant 1 |
| `SYN-SCORING-02` | NA Denominator Subtraction | 50 pts PASS + 50 pts NA -> effective denominator = 50 pts | 10000 bps (100.00%), PASS | Invariant 2 |
| `SYN-SCORING-03` | Non-Scored Exclusions | `TEXT_NOTE` and `EVIDENCE_ATTACHMENT` present | Excluded from denominator, PASS | Invariant 2 |
| `SYN-SCORING-04` | Unknown Quarantine | 90 pts PASS + 20 pts UNKNOWN | 9000 bps, PROVISIONAL outcome | Invariant 3 |
| `SYN-SCORING-05` | Boundary Round-Up | 79.995% earned points | 8000 bps (80.00%), PASS | Invariant 4 |
| `SYN-SCORING-06` | Boundary Round-Down | 79.994% earned points | 7999 bps (79.99%), FAIL | Invariant 4 |
| `SYN-SCORING-07` | Critical Fail Priority | 95.00% score + critical gas monitor fail | 9500 bps (unmasked), NON_COMPLIANT_CRITICAL | Invariant 5 |
| `SYN-SCORING-08` | Predicate Conjunction | Sub-threshold 75.00% score, zero criticals, zero unknowns | 7500 bps, FAIL outcome | Invariant 6 |
| `SYN-SCORING-09` | Priority Dominance | Critical fail active + UNKNOWN active + 9500 bps | CRITICAL_FAIL_PRIORITY denial | Invariant 7 |
| `SYN-SCORING-10` | Manual Override Denial | Admin attempts override under H040-004 | DenialManualOverrideDeferred, audited | Invariant 8 |
| `SYN-SCORING-11` | Autonomous AI Denial | AI agent attempts transition to APPROVED | DenialAutonomousAIBoundary | Invariant 9 |
| `SYN-SCORING-12` | Supervisor Quarantine Resolution | Human safety supervisor clears UNKNOWN with rationale | Quarantine cleared, transition permitted | Invariant 3 |
| `SYN-SCORING-13` | Dynamic Weight Redistribution | Section 3 all NA -> weights reallocated across Sec 1 & Sec 2 | Exact redistributed score, Sec 3 inactive | Invariant 1 |
| `SYN-SCORING-14` | Reporting Non-Authority | Query `metric_inspection_compliance_score` in MOD-REP | DERIVED_OUTPUT_NON_AUTHORITY notice | Invariant 10 |

---

## 4. Retained Foundation Holds & Non-Authority Affirmation

Under **HDEC-V040-FOUNDATION-054**, the following foundation holds remain in active `HOLD` status:

| Hold ID | Governance Subject | Status | Enforcement Constraint |
|---|---|---|---|
| `H040-007` | External Production Deployment | **HOLD** | No public/production traffic or DNS routing |
| `H040-008` | Live Third-Party Integrations | **HOLD** | Only mock/in-memory adapter boundaries |
| `H040-009` | Commercial Licensing & Payment Gateways | **HOLD** | Commercial transactions strictly disabled |
| `H040-010` | Automated Destructive Maintenance | **HOLD** | Automatic deletion or unreviewed purge prohibited |
| `H040-011` | Autonomous Human Decision Delegation | **HOLD** | Human signoff required for protected transitions |

Zero authority is granted to lift any hold. All qualification suites assert synthetic isolation and immutability of audit records.
