---
document_id: ARC-V040-SCORING-001
title: v0.4.0 OSHE Inspect Deterministic Scoring Calculation Engine, Basis Points, Exclusions, Rounding, and Limitations Baseline
document_type: architecture_specification
document_version: 1.0.0
lifecycle_status: DRAFT
status: APPROVED_FOR_LOCAL_DEVELOPMENT
date: "2026-09-05"
author_role: Engineering Lead
author_pane: w9:p23
governing_issue: "GitHub Issue #137"
authority_source: HDEC-V040-FOUNDATION-054
governing_decisions:
  - HDEC-V040-SCORING-058
  - HDEC-V040-FOUNDATION-054
  - HDEC-V030-ENTRY-AND-POLICY-052
  - ADR-0005
  - ADR-0006
milestone: "v0.4.0 - OSHE Inspect Private Alpha"
approved_scoring_rules:
  selected_scoring_model: MODEL_2_WEIGHTED
  selected_unknown_handling: U1_QUARANTINE_DENOMINATOR
  selected_rounding_rule: R1_ROUND_HALF_UP
  selected_critical_fail_policy: CF1_PRIORITY_FLAG
  selected_passing_threshold_percent: 80
pass_predicates:
  - No critical-fail condition is present.
  - No UNKNOWN response remains unresolved or quarantined.
  - Score is greater than or equal to 80.00 percent after the selected calculation and rounding rule.
approved_foundation_gates:
  - H040-001
  - H040-002
  - H040-003
  - H040-004
  - H040-005
  - H040-006
retained_holds:
  - H040-007
  - H040-008
  - H040-009
  - H040-010
  - H040-011
retained_unselected_policies:
  finding_closure_policy: HUMAN_OWNED_UNSELECTED
  offline_authority: HUMAN_OWNED_UNSELECTED
credit_boundary: SCORING_CALCULATION_ENGINE_SPECIFICATION_ONLY_NO_EXTERNAL_DELIVERY_OR_LIVE_OPERATIONAL_ENACTMENT
---

# v0.4.0 OSHE Inspect Deterministic Scoring Calculation Engine, Basis Points, Exclusions, Rounding, and Limitations Baseline

## 1. Executive Summary & Governance Posture

### 1.1 Authority Baseline & Purpose
This architectural specification establishes the authoritative **Deterministic Operational Scoring Engine, Basis Points Precision, Response Exclusions, Rounding Rules, and Limitations Baseline** for **Milestone v0.4.0 - OSHE Inspect Private Alpha** within `OSHEThai/oshe-platform`. It fulfills the deliverable requirements of **GitHub Issue #137 (`[V040-I026] Deterministic Scoring Calculation Engine`)** under Roadmap Topic `V040-T06` (Sequence 2.0) and standing owner authority decision **`HDEC-V040-SCORING-058`**.

On **2026-09-05 at 23:26:08 UTC**, the **Sole Human Owner** formally approved `HDEC-V040-SCORING-058`, establishing the exact mathematical scoring formulation, unknown handling, rounding rule, critical fail policy, and pass predicates for private alpha execution.

### 1.2 Module Ownership & Architecture Decisions
Under `ARC-V040-DOMAIN-001` and `ASN-V040-I026-SCORING-ENGINE-IMPLEMENTATION-002`:
1. **`MOD-WFA` (Workflow & Action):** Owns operational compliance scoring (`modules/workflow-action/scoring.go`). Evaluates completed inspection responses, applies business rules, and computes authoritative inspection compliance outcomes.
2. **`MOD-CFG` (Configuration & Checklist):** Owns frozen checklist templates, question definitions, section weight configurations, and scoring references (`modules/configuration-checklist`). Provides immutable inputs to the scoring engine.
3. **`MOD-REP` (Reporting & Localization):** Owns non-authoritative metric projections, dashboards, and reporting summaries (`modules/reporting-localization/reporting.go`). Operates strictly as a read-only projection layer with mandatory non-authority disclaimers (`DERIVED_OUTPUT_NON_AUTHORITY`).

### 1.3 Retained Unselected Policies & Non-Claims
In strict compliance with `HDEC-V040-FOUNDATION-054` and `HDEC-V040-SCORING-058`:
1. **Finding Closure Policy (`HUMAN_OWNED_UNSELECTED`):** Formal finding closure and residual safety risk acceptance remain human-owned under Issue #134 (`V040-I023`).
2. **Offline Authority Model (`HUMAN_OWNED_UNSELECTED`):** Operational lease expiration ceilings (24 hours) remain alpha baselines; long-term lease governance remains human-owned under Issue #126 (`V040-I015`).
3. **Non-Regulatory Synthetic Boundary:** Scoring calculations operate exclusively on synthetic non-regulatory alpha fixtures (`chk_syn_pilot_plant_safety_v1`). Zero statutory occupational safety compliance, regulatory certification, or legal liability discharge is claimed.
4. **Retained Human Gate Holds:** Foundation gates `H040-007` (Technical Release Authorization), `H040-008` (Real Participant / Private-Alpha UAT Onboarding), `H040-009` (Binding Support Ownership), `H040-010` (External Environment/Route Activation), and `H040-011` (Final Milestone Outcome) remain strictly on **`HOLD`**.

---

## 2. The Five Owner-Approved Scoring Choices (`HDEC-V040-SCORING-058`)

The scoring engine enforces the five exact policy choices authorized by the Sole Human Owner:

| Policy Area | Approved Selection | Architectural Identifier | Engine Behavior |
| :--- | :--- | :--- | :--- |
| **Scoring Model** | Category/Section Weighted Normalized Scoring | **`MODEL_2_WEIGHTED`** | Calculates compliance independently per section, multiplies by normalized section weights, and re-allocates weight if sections are inactive. |
| **Unknown Handling** | Quarantine UNKNOWN from Denominator | **`U1_QUARANTINE_DENOMINATOR`** | Quarantines available points from denominator and locks inspection to provisional status, blocking pass until resolved. |
| **Rounding Rule** | Round Half Up to Integer Basis Points | **`R1_ROUND_HALF_UP`** | Maps scores to integer basis points ($100.00\% = 10000\text{ bps}$) with half-up rounding at the second decimal place (1 bps). |
| **Critical-Fail Policy** | Priority Flag Without Score Masking | **`CF1_PRIORITY_FLAG`** | Locks compliance outcome to `NON_COMPLIANT_CRITICAL` while preserving numerical score calculation. |
| **Passing Threshold** | 80.00% Compliance Standard | **`80%` ($8000\text{ bps}$)** | Standard passing baseline evaluated after basis points conversion. |

---

## 3. The Three Mandatory Pass Predicates

An inspection achieves an overall passing compliance determination if and only if **all three** of the following predicates evaluate to true:

```
┌────────────────────────────────────────────────────────────────────────┐
│                        MANDATORY PASS PREDICATES                       │
│                                                                        │
│  [Predicate 1: Critical Fail Check]  ──> No critical-fail condition   │
│                 │                        is present.                   │
│                 ▼                                                      │
│  [Predicate 2: Unknown Resolution]   ──> No UNKNOWN response remains   │
│                 │                        unresolved or quarantined.    │
│                 ▼                                                      │
│  [Predicate 3: Threshold Evaluation] ──> Basis points >= 8000 (80.00%) │
│                 │                        after R1 half-up rounding.    │
│                 ▼                                                      │
│     [ALL THREE SATISFIED: PASS] (Otherwise: Non-Passing Outcome)       │
└────────────────────────────────────────────────────────────────────────┘
```

### 3.1 Predicate Evaluation Matrix

| Has Critical Fail? | Has Quarantined Unknown? | Score $\ge 80.00\%$? | All Predicates Met? | Final Compliance Outcome |
| :---: | :---: | :---: | :---: | :--- |
| **No** | **No** | **Yes** | **YES** | **`PASS`** |
| **Yes** | No | Yes ($95.00\%$) | No | **`NON_COMPLIANT_CRITICAL`** |
| **Yes** | Yes | Yes ($90.00\%$) | No | **`NON_COMPLIANT_CRITICAL`** |
| No | **Yes** | Yes ($92.00\%$) | No | **`PROVISIONAL_PENDING_UNKNOWN_RESOLUTION`** |
| No | No | **No** ($79.99\%$) | No | **`FAIL`** |
| **Yes** | No | **No** ($60.00\%$) | No | **`NON_COMPLIANT_CRITICAL`** |

---

## 4. Mathematical Formulation & Normalization (Model 2)

### 4.1 Section-Level Normalized Calculation
For each section $s \in \text{Sections}$:

$$\text{EffectiveDenominator}_s = \sum_{q \in s} \text{MaxPoints}_q - \text{NAPoints}_s - \text{QuarantinedUnknownPoints}_s$$

$$\text{SectionRatio}_s = \begin{cases} \frac{\sum_{q \in s} \text{EarnedPoints}_q}{\text{EffectiveDenominator}_s} & \text{if } \text{EffectiveDenominator}_s > 0 \\ 0.0 & \text{otherwise} \end{cases}$$

### 4.2 Dynamic Section Weight Redistribution
If all questions in a section evaluate to `NA` or `UNKNOWN`, the section has an effective denominator of zero and is marked inactive (`IsActive = false`).

To prevent score dilution, the engine dynamically recalculates effective section weights across active sections:

$$W_s^{\text{eff}} = \frac{W_s^{\text{orig}}}{\sum_{a \in \text{ActiveSections}} W_a^{\text{orig}}}$$

$$\text{WeightedScore}_{\text{raw}} = \sum_{s \in \text{ActiveSections}} \left( W_s^{\text{eff}} \times \text{SectionRatio}_s \right)$$

- **Zero-Division Defense:** If all sections across the entire inspection are inactive (e.g. 100% of questions are NA), $\text{WeightedScore}_{\text{raw}} = 0.0$ and basis points evaluate safely to $0\text{ bps}$ without NaN, panic, or division by zero.

---

## 5. Response Exclusions & Denominator Subtraction Semantics

1. **Non-Scored Question Types (`TEXT_NOTE`, `EVIDENCE_ATTACHMENT`):**
   - Provide qualitative context or proof attachments.
   - Assigned $\text{MaxPoints} = 0.0$ and excluded from all denominator calculations.
2. **Conditional Exclusions (`IsExcluded: true`):**
   - Questions skipped due to conditional branch rules (e.g. scaffolding questions hidden because no scaffolding was used) are completely excluded from scoring.
3. **Not Applicable (`NA`) Responses:**
   - Subtract the question's available points from the section denominator.
   - Cause **zero negative score impact** ($\text{Earned} / (\text{Max} - \text{NA})$).
4. **Unknown (`UNKNOWN`) Responses:**
   - Points are quarantined from the denominator during field calculation.
   - Flags the inspection as provisional and blocks passing compliance until resolved by supervisory review.

---

## 6. Integer Basis Points & Round Half Up Arithmetic (R1)

To prevent IEEE 754 floating-point rounding discrepancies across platforms:

### 6.1 Fixed-Point Integer Basis Points Standard
- $1\% = 100 \text{ basis points (bps)}$.
- $80.00\% = 8000 \text{ bps}$ (Passing Threshold).
- $100.00\% = 10000 \text{ bps}$ (Maximum Basis Points).

### 6.2 Deterministic Rounding Formula (`R1_ROUND_HALF_UP`)
$$\text{RawBPS} = \text{WeightedScore}_{\text{raw}} \times 10000$$

$$\text{BasisPoints} = \left\lfloor \text{RawBPS} + 0.5 + 10^{-9} \right\rfloor$$

- Epsilon ($10^{-9}$) compensates for binary float64 representation of exact decimal halves (e.g. $0.80005 \times 10000 = 8000.5$).
- **Boundary Proofs:**
  - 79.995% ($79.995\% \rightarrow 8000 \text{ bps}$, $80.00\%$, Pass).
  - 79.994% ($79.994\% \rightarrow 7999 \text{ bps}$, $79.99\%$, Fail).
  - 80.005% ($80.005\% \rightarrow 8001 \text{ bps}$, $80.01\%$, Pass).

---

## 7. Traceability, Disclosures & Non-Authoritative Reporting Projection

### 7.1 Pinned Version Traceability Key
Every scoring result generates a deterministic traceability record linking:
- `TemplateID`: Bound checklist template identifier.
- `TemplateVersion`: Pinned immutable template version (e.g. `1.1.0`).
- `RuleMatrixVersion`: Pinned business rule version (e.g. `1.0.0`).
- `FormulaVersion`: Authoritative formula identifier (`v0.4.0-HDEC-058`).
- `TraceabilityKey`: Formatted composite key `trace_{TemplateID}_{TemplateVersion}_{RuleMatrixVersion}_{Timestamp}`.

### 7.2 Non-Authoritative Reporting Projection (`MOD-REP`)
In accordance with `modules/reporting-localization/reporting.go`:
- Metric `metric_inspection_compliance_score` provides reporting projections.
- Mandates explicit disclosure:
  $$\text{"DERIVED_OUTPUT_NON_AUTHORITY: Reports, metrics, and analytics are derived outputs and never constitute operational authority or replace authoritative records."}$$
- Reaffirms that `MOD-WFA` remains the sole operational source of truth.

---

## 8. Synthetic Operations Fixture Matrix

The following synthetic YAML fixture illustrates the complete scoring evaluation scenarios:

```yaml
schema_version: "1.0.0"
fixture_id: "fix_syn_scoring_evaluation_v1"

scenarios:
  # Scenario 1: Standard Passing Weighted Evaluation
  - scenario_id: "scen_score_pass_01"
    execution_id: "ins_syn_pass_01"
    template_id: "chk_syn_pilot_plant_safety_v1"
    template_version: "1.1.0"
    formula_version: "v0.4.0-HDEC-058"
    raw_score_percent: 84.00
    basis_points: 8400
    display_score: "84.0%"
    has_critical_fail: false
    has_quarantined_unknown: false
    pass_predicates:
      no_critical_fail_present: true
      no_unresolved_unknown_quarantined: true
      score_threshold_satisfied: true
      all_predicates_satisfied: true
    outcome: "PASS"

  # Scenario 2: Critical Fail Priority Flag (CF1)
  - scenario_id: "scen_score_crit_02"
    execution_id: "ins_syn_crit_02"
    raw_score_percent: 95.00
    basis_points: 9500
    has_critical_fail: true
    outcome: "NON_COMPLIANT_CRITICAL"
    score_masked: false

  # Scenario 3: Unknown Quarantine (U1)
  - scenario_id: "scen_score_unknown_03"
    execution_id: "ins_syn_unk_03"
    raw_score_percent: 90.00
    basis_points: 9000
    has_quarantined_unknown: true
    outcome: "PROVISIONAL_PENDING_UNKNOWN_RESOLUTION"

  # Scenario 4: R1 Round Half Up Boundary
  - scenario_id: "scen_score_round_up_04"
    raw_input_percent: 79.995
    expected_basis_points: 8000
    expected_rounded_percent: 80.00
    outcome: "PASS"
```

---

## 9. Governance Boundaries, Prohibitions & Operational Non-Claims

In strict compliance with `HDEC-V040-FOUNDATION-054` and `HDEC-V040-SCORING-058`:

1. **100% Synthetic Data Policy (`H040-003`):** All calculations operate on synthetic test fixtures. Zero customer records or real workforce data are referenced.
2. **Preservation of Foundation Holds (`H040-007` - `H040-011` HOLD):** Zero external route activation, cloud deployment, or legal safety certification is claimed.
3. **No Unilateral Policy Selection:** The engine enforces strictly the five owner-approved selections in `HDEC-V040-SCORING-058`.
4. **Specification & Engine Implementation Credit:** Delivery confers operational scoring engine and documentation baseline credit only; zero milestone release or residual-risk acceptance is claimed.
