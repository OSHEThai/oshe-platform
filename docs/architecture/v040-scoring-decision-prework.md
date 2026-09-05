---
document_id: ARC-V040-SCORING-DECPKT-001
title: v0.4.0 OSHE Inspect Versioned Scoring Rule Matrix, Exclusions, Rounding, and Migration Decision Prework (PENDING_OWNER_DECISION)
document_type: architecture_decision_packet
document_version: 1.0.0
lifecycle_status: DRAFT
status: PENDING_OWNER_DECISION
date: "2026-09-05"
author_role: Architecture and Data Lead
author_pane: w9:p22
governing_issue: "GitHub Issue #136"
authority_source: HDEC-V040-FOUNDATION-054
governing_decisions:
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
retained_unselected_policies:
  binding_scoring_policy: HUMAN_OWNED_UNSELECTED
  finding_closure_policy: HUMAN_OWNED_UNSELECTED
  offline_authority: HUMAN_OWNED_UNSELECTED
credit_boundary: SCORING_DECISION_PREWORK_ONLY_NO_BINDING_POLICY_SELECTION
---

# v0.4.0 OSHE Inspect Versioned Scoring Rule Matrix, Exclusions, Rounding, and Migration Decision Prework (PENDING_OWNER_DECISION)

## 1. Executive Summary & Governance Reference

### 1.1 Authority Baseline & Objective
This architectural decision packet establishes the non-binding **Versioned Scoring Rule Matrix, Exclusions, Rounding, and Migration Decision Prework** for **Milestone v0.4.0 - OSHE Inspect Private Alpha** in fulfillment of **GitHub Issue #136 (`[V040-I025] Define Versioned Scoring Rule Matrix for Weights, Answers, Sections, Exclusions, Unknown, Not Applicable, Rounding, and Rule Changes`)** under standing owner foundation decision `HDEC-V040-FOUNDATION-054`.

The primary purpose is to formulate structured, deterministic, and verifiable options for inspection score computation, answer weighting, question exclusions, Unknown/NA semantics, rounding arithmetic, critical-fail handling, and cross-version migration to enable an informed decision by the **Sole Human Owner**.

### 1.2 Reservation of Human Decision Authority (PENDING_OWNER_DECISION)
In strict compliance with `ASN-V040-I025-SCORING-PREWORK-001` and `HDEC-V040-FOUNDATION-054`:
1. **Binding Scoring Policy Remains Unselected:** All scoring models, weights, multipliers, passing percentages, and critical-fail triggers presented in this document are **purely descriptive prework models**. Zero binding scoring policies, certification thresholds, or legal/safety outcomes are selected or enacted.
2. **Designation:** This prework is formally marked **`PENDING_OWNER_DECISION`**.
3. **Issue #136 Remains OPEN:** GitHub Issue #136 remains open pending formal evaluation and execution of the scoring-rule decision gate by the Sole Human Owner.
4. **Retained Human Gate Holds:** Foundation gates `H040-007` (Technical Release Authorization), `H040-008` (Real Participant / Private-Alpha UAT Onboarding), `H040-009` (Binding Support Ownership), `H040-010` (External Environment/Route Activation), and `H040-011` (Final Milestone Outcome) remain strictly on **`HOLD`**.

---

## 2. Approved Foundation Semantics & Scoring Inputs (`H040-006`)

Under approved foundation gate `H040-006`, scoring in Milestone v0.4.0 is governed by strict non-regulatory alpha boundaries:
- **Synthetic Non-Regulatory Pilot Checklist:** Operates on fictionalized synthetic alpha fixtures (`chk_syn_pilot_plant_safety_v1`).
- **Zero Legal or Real Safety Claims:** The scoring engine makes zero claim of statutory occupational safety compliance, regulatory certification, or legal liability discharge.

### 2.1 Supported Question Types & Score Contribution
The scoring model processes responses across the six supported alpha question types established in `ARC-V040-CHKL-001` and `ARC-V040-CHKRULE-001`:

| Question Type | Response States | Scored Contribution Semantics | Denominator Inclusion |
| :--- | :--- | :--- | :--- |
| **`PASS_FAIL_NA_UNKNOWN`** | `PASS` | 100% of question base points earned. | Included in denominator. |
| | `FAIL` | 0% of question base points earned; triggers non-conformance finding. | Included in denominator. |
| | `NA` | Excluded from denominator; requires mandatory explanatory note. | **Excluded from denominator.** |
| | `UNKNOWN` | Evaluated under unselected policy options; requires mandatory justification. | Evaluated per Section 2.3. |
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

### 2.3 Unknown (`UNKNOWN`) Semantics & Alternatives
When an inspector cannot evaluate a safety condition due to physical, environmental, or access barriers:
1. **Mandatory Justification:** The inspector must provide a written justification note.
2. **Automatic Review Task:** Triggers an automatic supervisory follow-up task in `MOD-WFA`.
3. **Three Scoring Disposition Alternatives (Unselected):**
   - **Alternative U1 (Conservative Alpha Default - Recommended):** Quarantine from denominator. The question's points are excluded from the denominator, but the inspection is flagged as `PROVISIONAL_PENDING_UNKNOWN_RESOLUTION`.
   - **Alternative U2 (Strict Compliance / Zero Earned):** Question remains in denominator with 0 earned points until verified by supervisor.
   - **Alternative U3 (Neutral Weight Halving):** Awards 50% points provisionally pending resolution.

---

## 3. Alternative Versioned Scoring Formulations

Three versioned scoring formulas are modeled for human owner evaluation:

### Model 1: Flat Percentage Compliance (Additive Unweighted)
- **Mathematical Definition:**
  $$Score_{flat} = \left( \frac{\sum_{i \in Applicable} PointsEarned_i}{\sum_{i \in Applicable} MaxPoints_i} \right) \times 100$$
- **Tradeoff Profile:**
  - *Advantages:* Simple, transparent, immediately auditable by field inspectors without complex calculation tools.
  - *Disadvantages:* Treats critical safety items (e.g. emergency shutoff access) identically to minor administrative items (e.g. sign posting).

### Model 2: Category/Section Weighted Normalized Scoring (Recommended Prework)
- **Mathematical Definition:**
  $$Score_{weighted} = \sum_{s \in Sections} \left( Weight_s \times \frac{\sum_{q \in s} PointsEarned_q}{\sum_{q \in s} MaxPoints_q} \right)$$
  Where $\sum Weight_s = 1.0$ (normalized 100%).
- **Tradeoff Profile:**
  - *Advantages:* Reflects relative risk across functional domains (e.g. Electrical Safety weighted 40%, Housekeeping weighted 10%). Isolates question density differences between sections.
  - *Disadvantages:* Requires explicit authoring of normalized section weights in checklist templates.

### Model 3: Risk-Deductive Penalty Scoring
- **Mathematical Definition:**
  $$Score_{deductive} = \max\left(0, 100 - \sum_{f \in Findings} Penalty(Severity_f)\right)$$
  Where penalties are tiered: Low = -2, Medium = -5, High = -15, Critical = -30.
- **Tradeoff Profile:**
  - *Advantages:* Aligns with traditional regulatory enforcement audit methods.
  - *Disadvantages:* Can collapse to 0 quickly on complex industrial sites; non-intuitive for positive compliance measurement.

---

## 4. Rounding Arithmetic & Numerical Precision

To guarantee deterministic, bit-for-bit identical calculation between client web applications and central server engines:

1. **Fixed-Point Precision Standard:**
   - Raw floating-point arithmetic (IEEE 754 float64) is normalized to fixed-point integer basis points ($1\% = 100 \text{ bps}$; $100.00\% = 10000 \text{ bps}$) before persistence and comparison.
2. **Rounding Method Alternatives:**
   - **Method R1: Round Half Up (`ROUND_HALF_UP`) - Recommended:** Standard arithmetic rounding at the second decimal place (0.01%). Example: $95.555\% \rightarrow 95.56\%$.
   - **Method R2: Floor / Truncate (`ROUND_DOWN`):** Truncates beyond two decimal places. Example: $95.559\% \rightarrow 95.55\%$. Conservative; never overstates compliance.
3. **Disclosed Display Precision:** Display views format scores to exactly one decimal place ($95.6\%$) while internal audit ledgers preserve exact basis points ($9556$).

---

## 5. Critical-Fail Invariants & AI Safety Boundary (`H040-004`)

### 5.1 Critical-Fail Handling Alternatives
When an inspection response triggers a critical failure condition (e.g., failed emergency brake, exposed high-voltage bus):

- **Alternative CF1: Non-Overriding Priority Flag (Recommended Prework):**
  - The inspection mathematical score continues to evaluate earned points.
  - However, the overall compliance outcome is locked to `NON_COMPLIANT_CRITICAL`.
  - An immediate critical finding is staged in `MOD-WFA` requiring mandatory supervisor notification.
  - *Rationale:* Preserves numerical scoring transparency while preventing safety masking.
- **Alternative CF2: Hard Override / Zero Total Score:**
  - The inspection overall score is immediately clamped to $0.00\%$ regardless of other passing responses.
- **Alternative CF3: Capped Maximum Score:**
  - Overall score is capped at a failing threshold (e.g., maximum $59.00\%$).

### 5.2 AI Safety Boundary Invariant (`H040-004`)
- AI models, neural estimators, and automated algorithms have **ZERO authority** to:
  1. Trigger, downgrade, or dismiss a critical-fail condition.
  2. Compute, alter, or certify an official inspection score.
  3. Authorize finding closure or risk acceptance.
- AI outputs remain strictly auxiliary, supporting artifacts requiring explicit human review.

---

## 6. Template Versioning, Migration & Execution Pinning

Under `ARC-V040-CHKL-001` and `ARC-V040-CHKLIFE-001`:

1. **Execution Pinning Invariant:**
   - Active inspections are permanently pinned to the exact `template_id` and `template_version` instantiated at creation time.
   - When a checklist template's scoring model or weights are modified, a new SemVer version (`major.minor.patch`) is published.
   - Ongoing inspections in `IN_PROGRESS` status **never** hot-swap scoring rules.
2. **Cross-Version Migration Rules:**
   - Pre-existing historical inspection scores are permanently immutable.
   - If an audit replay or re-scoring is explicitly requested by an authorized human reviewer, the recalculated score is recorded as a separate versioned audit projection (`recalculated_score_v2`) linking to the new scoring rule version, preserving the original score intact in `MOD-REC`.

---

## 7. Sole Human Owner Decision Record (PENDING_OWNER_DECISION)

> **MANDATORY GOVERNANCE NOTICE:** The decision record below is strictly reserved for the Sole Human Owner. No automated agent, script, or subagent possesses authority to select options, populate values, or sign this record.

```yaml
# Decision Record: HDEC-V040-SCORING-058 (PENDING)
schema_version: 1.0.0
decision_id: HDEC-V040-SCORING-058
governing_issue: 136
governing_gate: H040-006
status: PENDING_OWNER_DECISION

# Formulation Selection: [ MODEL_1_FLAT | MODEL_2_WEIGHTED | MODEL_3_DEDUCTIVE ]
selected_scoring_model: UNFILLED

# Unknown Response Handling: [ U1_QUARANTINE_DENOMINATOR | U2_ZERO_EARNED | U3_HALF_POINTS ]
selected_unknown_handling: UNFILLED

# Rounding Method: [ R1_ROUND_HALF_UP | R2_ROUND_DOWN ]
selected_rounding_rule: UNFILLED

# Critical Fail Behavior: [ CF1_PRIORITY_FLAG | CF2_HARD_ZERO | CF3_SCORE_CAP ]
selected_critical_fail_policy: UNFILLED

# Passing Score Threshold (Percentage):
selected_passing_threshold: UNFILLED

# Execution Attribution:
decided_by: UNFILLED  # Must be Sole Human Owner
decided_at: UNFILLED  # ISO 8601 UTC Timestamp
signature_or_auth_ref: UNFILLED
```

---

## 8. Operational Prohibitions & Governance Non-Claims

In strict adherence to `HDEC-V040-FOUNDATION-054`:
1. **No Binding Policy Selection:** Zero binding scoring formulas, thresholds, or critical-fail policies are selected by this document.
2. **No Issue Closure:** GitHub Issue #136 remains **OPEN** pending Sole Human Owner execution of `HDEC-V040-SCORING-058`.
3. **No External Route or Deployment Action:** Public network routes, DNS, CDN distribution, cloud infrastructure, and database mutations remain on strict **HOLD** (`H040-007`, `H040-010`).
4. **No Real User Testing:** No real participants or UAT sessions are authorized (`H040-008`).
5. **No Production / Customer Data:** Operates exclusively on synthetic non-regulatory fixtures.
