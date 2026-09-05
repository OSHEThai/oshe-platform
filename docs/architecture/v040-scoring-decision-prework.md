---
document_id: ARC-V040-SCORING-DECPKT-001
title: v0.4.0 OSHE Inspect Versioned Scoring Rule Matrix, Exclusions, Rounding, and Migration Approved Decision Baseline (HDEC-V040-SCORING-058)
document_type: architecture_decision_record
document_version: 1.0.0
lifecycle_status: APPROVED
status: APPROVED_BY_SOLE_HUMAN_OWNER
date: "2026-09-05"
author_role: Architecture and Data Lead
author_pane: w9:p22
governing_issue: "GitHub Issue #136"
authority_source: HDEC-V040-SCORING-058
governing_decisions:
  - HDEC-V040-SCORING-058
  - HDEC-V040-FOUNDATION-054
  - HDEC-V030-ENTRY-AND-POLICY-052
  - ADR-0005
  - ADR-0006
milestone: "v0.4.0 - OSHE Inspect Private Alpha"
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
approved_scoring_decisions:
  decision_id: HDEC-V040-SCORING-058
  selected_scoring_model: MODEL_2_WEIGHTED
  selected_unknown_handling: U1_QUARANTINE_DENOMINATOR
  selected_rounding_rule: R1_ROUND_HALF_UP
  selected_critical_fail_policy: CF1_PRIORITY_FLAG
  selected_passing_threshold_percent: 80
pass_predicates:
  - No critical-fail condition is present.
  - No UNKNOWN response remains unresolved or quarantined.
  - Score is greater than or equal to 80.00 percent after the selected calculation and rounding rule.
retained_unselected_policies:
  finding_closure_policy: HUMAN_OWNED_UNSELECTED
  offline_authority: HUMAN_OWNED_UNSELECTED
credit_boundary: SCORING_DECISION_RECORD_MATERIALIZATION_ONLY
---

# v0.4.0 OSHE Inspect Versioned Scoring Rule Matrix, Exclusions, Rounding, and Migration Approved Decision Baseline (HDEC-V040-SCORING-058)

## 1. Executive Summary & Governance Reference

### 1.1 Authority Baseline & Approved Scoring Decision
This architectural decision record establishes the approved **Versioned Scoring Rule Matrix, Exclusions, Rounding, and Migration Baseline** for **Milestone v0.4.0 - OSHE Inspect Private Alpha** in formal fulfillment of **GitHub Issue #136 (`[V040-I025] Define Versioned Scoring Rule Matrix for Weights, Answers, Sections, Exclusions, Unknown, Not Applicable, Rounding, and Rule Changes`)** under standing owner authority decision **`HDEC-V040-SCORING-058`** and foundation decision `HDEC-V040-FOUNDATION-054`.

On **2026-09-05 at 23:26:08 UTC**, the **Sole Human Owner** formally approved the scoring architecture baseline under `HDEC-V040-SCORING-058`. The decision transitions the scoring policy from pending evaluation to **`APPROVED_BY_SOLE_HUMAN_OWNER`** for private-alpha synthetic execution.

### 1.2 Approved Scoring Choices & Mandatory Pass Predicates
In strict accordance with `HDEC-V040-SCORING-058`, the Sole Human Owner has authorized the following exact scoring selections:
1. **Selected Scoring Model:** **`MODEL_2_WEIGHTED`** (Category/Section Weighted Normalized Scoring).
2. **Selected Unknown Handling:** **`U1_QUARANTINE_DENOMINATOR`** (Quarantine UNKNOWN responses from the denominator pending supervisory verification).
3. **Selected Rounding Rule:** **`R1_ROUND_HALF_UP`** (Fixed-point basis points standard, round half-up at the second decimal place).
4. **Selected Critical-Fail Policy:** **`CF1_PRIORITY_FLAG`** (Priority flag; overall inspection outcome locked to `NON_COMPLIANT_CRITICAL` without score masking).
5. **Selected Passing Threshold:** **`80%`** ($80.00\%$ compliance threshold, equivalent to 8000 basis points).

#### Mandatory Inspection Pass Predicates
An inspection achieves an overall passing compliance status if and only if **all three** of the following predicates are satisfied:
1. **No critical-fail condition is present.**
2. **No UNKNOWN response remains unresolved or quarantined.**
3. **Score is greater than or equal to 80.00 percent after the selected calculation and rounding rule.**

### 1.3 Retained Human Gate Holds
In strict adherence to foundation governance, gates `H040-007` through `H040-011` remain on **`HOLD`**:
- `H040-007` (Technical Release Authorization): `HOLD` pending full integration evidence.
- `H040-008` (Real Participant / Private-Alpha UAT Onboarding): `HOLD` pending owner authorization.
- `H040-009` (Binding Support Ownership): `HOLD`.
- `H040-010` (External Environment/Route Activation): `HOLD`.
- `H040-011` (Final Milestone Outcome): `HOLD`.

---

## 2. Approved Foundation Semantics & Scoring Inputs (`H040-006`)

Under approved foundation gate `H040-006`, scoring in Milestone v0.4.0 is governed by strict non-regulatory alpha boundaries:
- **Synthetic Non-Regulatory Pilot Checklist:** Operates exclusively on fictionalized synthetic alpha fixtures (`chk_syn_pilot_plant_safety_v1`).
- **Zero Legal or Real Safety Claims:** The scoring engine makes zero claim of statutory occupational safety compliance, regulatory certification, or legal liability discharge.

### 2.1 Supported Question Types & Score Contribution
The scoring model processes responses across the six supported alpha question types established in `ARC-V040-CHKL-001` and `ARC-V040-CHKRULE-001`:

| Question Type | Response States | Scored Contribution Semantics | Denominator Inclusion |
| :--- | :--- | :--- | :--- |
| **`PASS_FAIL_NA_UNKNOWN`** | `PASS` | 100% of question base points earned. | Included in denominator. |
| | `FAIL` | 0% of question base points earned; triggers non-conformance finding. | Included in denominator. |
| | `NA` | Excluded from denominator; requires mandatory explanatory note. | **Excluded from denominator.** |
| | `UNKNOWN` | Quarantined from denominator (`U1`); requires mandatory justification. | **Quarantined per U1.** |
| **`SINGLE_CHOICE`** | Option ID | Earned points determined by option-level score weight (0.0 to 1.0 multiplier). | Included in denominator. |
| **`MULTI_CHOICE`** | Array of Option IDs | Evaluated via additive or all-or-nothing option weights; bounded by question max. | Included in denominator. |
| **`NUMERIC_MEASUREMENT`** | IEEE 754 Float | Graded against defined target range (within range = 100%; out of range = 0%). | Included in denominator. |
| **`TEXT_NOTE`** | String (1-1000 chars) | Informational context only; non-scored (weight = 0.0). | Excluded from denominator. |
| **`EVIDENCE_ATTACHMENT`** | Array of evd_syn_* IDs | Verification artifact; non-scored directly (weight = 0.0). | Excluded from denominator. |

### 2.2 Not Applicable (`NA`) Semantics
In accordance with `H040-006` and `ARC-V040-CHKRULE-001`:
1. **Denominator Subtraction:** When an inspector records `NA`, the question's maximum available points are subtracted from the inspection's scoring denominator.
2. **Formula:** $Earned / (MaxAvailable - NAPoints)$.
3. **Zero Negative Impact:** An `NA` response causes zero penalty or downward score distortion.
4. **Mandatory Note:** Requires a non-blank explanatory note justifying non-applicability.

### 2.3 Approved Unknown (`UNKNOWN`) Semantics (`U1_QUARANTINE_DENOMINATOR`)
Under approved choice **`U1_QUARANTINE_DENOMINATOR`**:
1. **Denominator Quarantine:** The maximum points associated with an `UNKNOWN` question are quarantined and subtracted from the active scoring denominator during field calculation.
2. **Provisional Status:** Any inspection containing an unresolved `UNKNOWN` response is flagged as `PROVISIONAL_PENDING_UNKNOWN_RESOLUTION`.
3. **Blocking Pass Predicate:** An inspection cannot achieve a passing evaluation while an `UNKNOWN` response remains unresolved or quarantined.
4. **Supervisory Verification Task:** Automatically stages a follow-up verification task in `MOD-WFA` to clear the barrier and achieve definitive resolution.

---

## 3. Approved Scoring Formulation: Model 2 (Category/Section Weighted Normalized Scoring)

Under approved choice **`MODEL_2_WEIGHTED`**:

### 3.1 Mathematical Definition
$$Score_{weighted} = \sum_{s \in Sections} \left( Weight_s \times \frac{\sum_{q \in s} PointsEarned_q}{\sum_{q \in s} MaxPoints_q - NAPoints_s - UnknownPoints_s} \right)$$
Where:
- $\sum_{s \in Sections} Weight_s = 1.0$ (normalized 100%).
- Each section score is evaluated independently across its active, non-quarantined questions.
- If all questions in a section evaluate to `NA` or `UNKNOWN`, the section weight is dynamically re-allocated proportionally across remaining active sections.

### 3.2 Evaluation of Alternative Models (Historical Context)
- *Model 1 (Flat Percentage Compliance):* Evaluated as alternative; rejected in favor of Model 2 to prevent question density bias.
- *Model 3 (Risk-Deductive Penalty Scoring):* Evaluated as alternative; rejected in favor of Model 2 to ensure positive compliance normalization.

---

## 4. Approved Rounding Arithmetic: R1 (Round Half Up to 2 Decimal Places)

Under approved choice **`R1_ROUND_HALF_UP`**:

1. **Fixed-Point Precision Standard:**
   - Raw floating-point calculations (IEEE 754 float64) are mapped to integer basis points ($1\% = 100 \text{ bps}$; $100.00\% = 10000 \text{ bps}$; passing threshold $80.00\% = 8000 \text{ bps}$).
2. **Deterministic Rounding Rule (`ROUND_HALF_UP`):**
   - Scores are rounded at the second decimal place (0.01% / 1 basis point).
   - Values with fractional remainder $\ge 0.5\text{ bps}$ round upward to the nearest basis point.
   - Example: $79.995\% \rightarrow 80.00\%$ (Pass); $79.994\% \rightarrow 79.99\%$ (Fail).
3. **Display vs. Audit Ledger Precision:**
   - User interfaces display scores formatted to one decimal place ($80.0\%$) with secondary tooltip indicating exact basis points ($80.00\%$).
   - Immutable audit logs in `MOD-REC` preserve exact integer basis points ($8000$).

---

## 5. Approved Critical-Fail Policy: CF1 (Priority Flag Without Score Masking)

Under approved choice **`CF1_PRIORITY_FLAG`**:

### 5.1 Enforcement Mechanics
1. **Priority Flagging:** When any response triggers a critical-fail condition (e.g. `critical_flag: true` on life-safety hazard):
   - The numerical percentage score continues to evaluate earned points according to Model 2.
   - The overall compliance outcome is locked to **`NON_COMPLIANT_CRITICAL`**, regardless of whether the score exceeds $80.00\%$.
2. **First Pass Predicate Violation:** An inspection containing an active critical-fail condition fails the first pass predicate (*No critical-fail condition is present*), resulting in a definitive non-passing inspection determination.
3. **Immediate Notification:** Staged immediately in `MOD-WFA` with mandatory immediate-control documentation.

### 5.2 AI Safety Boundary Invariant (`H040-004`)
- AI models, algorithms, and automated agents have **ZERO authority** to:
  1. Trigger, downgrade, clear, or override a critical-fail condition.
  2. Modify official scoring weights, formulas, or passing thresholds.
  3. Authorize finding closure or risk acceptance.
- AI tools remain strictly auxiliary, supporting fixtures requiring human review.

---

## 6. Template Versioning, Migration & Execution Pinning

Under `ARC-V040-CHKL-001` and `ARC-V040-CHKLIFE-001`:

1. **Execution Pinning Invariant:**
   - Active inspections are permanently pinned to the exact `template_id` and `template_version` instantiated at creation time.
   - Ongoing inspections in `IN_PROGRESS` status **never** hot-swap scoring models, weights, or thresholds.
2. **Cross-Version Migration Rules:**
   - Historical inspection scores are permanently immutable.
   - Recalculations requested for audit replay are recorded as distinct versioned audit projections (`recalculated_score_v2`) in `MOD-REC`, preserving original scores intact.

---

## 7. Sole Human Owner Approved Decision Record

The Sole Human Owner has executed **`HDEC-V040-SCORING-058`**:

```yaml
# Decision Record: HDEC-V040-SCORING-058
schema_version: 1.0.0
decision_id: HDEC-V040-SCORING-058
status: APPROVED_STANDING_UNTIL_SUPERSEDED
authority: Sole Human Owner
governing_issue: 136
governing_gate: H040-006
approved_at_utc: '2026-09-05T23:26:08Z'
scope: Synthetic, non-regulatory v0.4.0 OSHE Inspect alpha scoring only; no release, customer-data, legal, compliance, or residual-risk authority.
selected_scoring_model: MODEL_2_WEIGHTED
selected_unknown_handling: U1_QUARANTINE_DENOMINATOR
selected_rounding_rule: R1_ROUND_HALF_UP
selected_critical_fail_policy: CF1_PRIORITY_FLAG
selected_passing_threshold_percent: 80
pass_predicates:
  - No critical-fail condition is present.
  - No UNKNOWN response remains unresolved or quarantined.
  - Score is greater than or equal to 80.00 percent after the selected calculation and rounding rule.
retained_holds:
  - H040-007
  - H040-008
  - H040-009
  - H040-010
  - H040-011
```

---

## 8. Operational Prohibitions & Governance Non-Claims

In strict adherence to `HDEC-V040-FOUNDATION-054` and `HDEC-V040-SCORING-058`:
1. **Scope of Authority:** Approval applies strictly to the scoring rule baseline for private-alpha synthetic test fixtures. No policy selection beyond `HDEC-V040-SCORING-058` is enacted.
2. **Issue Status:** GitHub Issue #136 remains **OPEN** pending completed independent review and PR merge.
3. **No External Route or Deployment Action:** Public network routes, DNS, CDN distribution, cloud infrastructure, and database mutations remain on strict **`HOLD`** (`H040-007`, `H040-010`).
4. **No Real User Testing:** Real participant recruitment and live UAT remain on strict **`HOLD`** (`H040-008`).
5. **No Production / Customer Data:** Operates exclusively on synthetic non-regulatory fixtures (`H040-003`).
