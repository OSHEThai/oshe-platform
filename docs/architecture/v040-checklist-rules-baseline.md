---
document_id: ARC-V040-CHKRULE-001
title: v0.4.0 OSHE Inspect Checklist Conditional Rules, Response Validation, Evidence Policies, Exclusions, and Scoring-Reference Baseline
document_type: architecture_specification
document_version: 1.0.0
lifecycle_status: DRAFT
status: APPROVED_FOR_LOCAL_DEVELOPMENT
date: "2026-09-05"
author_role: Engineering Lead
author_pane: w9:p23
governing_issue: "GitHub Issue #117"
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
credit_boundary: CHECKLIST_RULES_SPECIFICATION_ONLY_NO_BINDING_BUSINESS_LOGIC_OR_SCORING_SELECTION
---

# v0.4.0 OSHE Inspect Checklist Conditional Rules, Response Validation, Evidence Policies, Exclusions, and Scoring-Reference Baseline

## 1. Executive Summary & Governance Reference

### 1.1 Authority Baseline & Purpose
This architectural specification establishes the authoritative **Checklist Conditional Rules, Response Validation, Evidence Policies, Question Exclusions, Safe Failures, and Scoring-Reference Baseline** for **Milestone v0.4.0 - OSHE Inspect Private Alpha** within `OSHEThai/oshe-platform`. It fulfills the requirements and deliverable specifications of **GitHub Issue #117 (`[V040-I006] Checklist Conditional Rules, Response Validation, and Exclusions`)** under Roadmap Topic `V040-T01` and standing owner foundation decision `HDEC-V040-FOUNDATION-054`.

The primary purpose is to define an integrated, deterministic, dependency-free specification governing how checklist questions and sections evaluate display visibility, enforce answer boundaries across the six supported alpha question types, mandate evidence capture, handle exclusions without scoring bias, and fail closed during configuration or evaluation anomalies.

### 1.2 Retained Unselected Policies & Non-Claims
In strict compliance with `ASN-V040-I006-CHECKLIST-RULES-001` and `HDEC-V040-FOUNDATION-054`:
1. **Binding Scoring Policy (`HUMAN_OWNED_UNSELECTED`):** All scoring parameters, weights, category multipliers, and compliance thresholds referenced in this document are descriptive, provisional reference models. Binding scoring algorithms and certification thresholds remain strictly human-owned pending explicit owner determination under Issue #136 (`V040-I025`).
2. **Finding Closure Policy (`HUMAN_OWNED_UNSELECTED`):** Regulatory sign-off criteria, corrective action verification standards, and residual risk acceptance remain human-owned under Issue #134 (`V040-I023`).
3. **Offline Authority Model (`HUMAN_OWNED_UNSELECTED`):** Client-side lease durations, offline download expiration limits, and conflict priority policies require explicit human owner decision under Issue #126 (`V040-I015`).
4. **Prohibition of Generalized No-Code Scripting:** Dynamic client-side code execution (`eval()`), arbitrary arithmetic expressions, unconstrained recursive nesting, and live external API lookup questions are categorically prohibited.
5. **Retained Human Gate Holds:** Foundation gates `H040-007` (Technical Release Authorization), `H040-008` (Real Participant / Private-Alpha UAT Onboarding), `H040-009` (Binding Support Ownership), `H040-010` (External Environment/Route Activation), and `H040-011` (Final Milestone Outcome) remain strictly on **`HOLD`**.

---

## 2. Deterministic Rule Model & Evaluation Engine

Under the checklist model baseline (`ARC-V040-CHKL-001`), the **Configuration and Checklist Module (`MOD-CFG`)** defines and owns the rule evaluation engine that governs question visibility and validation.

### 2.1 Rule Types & Entity Definitions
The rule model defines three distinct, strongly-typed rule categories:

```
+----------------------------------------------------------------------------------------------------+
| APPLICABILITY RULE ENTITY (app_syn_*)                                                              |
|                                                                                                    |
|  - rule_id: String (app_syn_[a-z0-9_]{8,32})           - target_entity: "SECTION" | "QUESTION"     |
|  - predicate_type: PredicateType                       - target_id: String (sec_syn_* | qst_syn_*)  |
|  - target_field: String (context attribute name)       - action: "SHOW" | "HIDE"                   |
|  - operator: RuleOperator                              - default_fallback: "SHOW"                  |
|  - comparison_values: []String                         - description: LocalizedStringMap           |
+----------------------------------------------------------------------------------------------------+
```

1. **`ApplicabilityRule`:** Governs whether an entire section or individual question is in-scope for an inspection based on static site/area/work metadata.
2. **`DisplayConditionRule`:** Evaluates dynamic visibility of downstream questions based on an inspector's prior response to an upstream question in the same inspection.
3. **`TriggerFindingRule`:** Evaluates an inspector's response to determine if an automatic non-conformance `Finding` must be staged in `MOD-WFA`.

### 2.2 Supported Predicates & Operators
In accordance with private-alpha boundary constraints, rule predicates and comparison operators are strictly restricted to the following closed catalog:

| Category | Identifier | Behavioral Semantics & Supported Types |
| :--- | :--- | :--- |
| **Predicate** | **`METADATA_MATCH`** | Evaluates static contextual attributes provided at inspection initialization (`Site.facility_type`, `Area.hazard_classification`, `Inspection.work_type`). |
| **Predicate** | **`PRIOR_ANSWER_MATCH`** | Evaluates the captured response value of an upstream question within the same checklist execution session. |
| **Operator** | **`EQUALS`** | Exact string or numeric equality. |
| **Operator** | **`NOT_EQUALS`** | String or numeric inequality. |
| **Operator** | **`IN`** | Target value exists within a declared set of allowed string values. |
| **Operator** | **`NOT_IN`** | Target value does not exist within a declared set of string values. |
| **Operator** | **`GREATER_THAN`** | Numerical value strictly exceeds threshold (`NUMERIC_MEASUREMENT` only). |
| **Operator** | **`LESS_THAN`** | Numerical value strictly falls below threshold (`NUMERIC_MEASUREMENT` only). |

### 2.3 Acyclic Evaluation & Topological Ordering Invariants
To prevent infinite evaluation loops, race conditions, and non-deterministic UI rendering:
1. **Strict Topological Ordering:** Rules may only depend on questions with a lower `display_order` than the target question. Forward dependencies (e.g. Question 2 depending on Question 5) are strictly prohibited and rejected during template validation.
2. **Acyclic Dependency DAG:** Circular dependencies between rules (Rule A triggers Rule B which triggers Rule A) are mathematically impossible under the forward-only topological ordering rule.
3. **Pure Functional Evaluation:** Rule evaluation is idempotent, side-effect-free, and deterministic. Given identical context metadata and prior answers, the rule engine must produce identical visibility states.

---

## 3. Response Validation & Answer Constraints

The checklist engine enforces strict, fail-closed validation on all incoming responses across the six supported alpha question types:

```
┌──────────────────────────────────────────────────────────────────────────────────────────────────┐
│                               RESPONSE VALIDATION BOUNDARIES                                     │
├──────────────────────────┬───────────────────────────────┬───────────────────────────────────────┤
│ Question Type            │ Valid Response Formats        │ Enforced Invariants                   │
├──────────────────────────┼───────────────────────────────┼───────────────────────────────────────┤
│ PASS_FAIL_NA_UNKNOWN     │ "PASS", "FAIL", "NA",         │ NA/UNKNOWN mandate explanatory note;  │
│                          │ "UNKNOWN"                     │ FAIL immediately triggers finding.    │
├──────────────────────────┼───────────────────────────────┼───────────────────────────────────────┤
│ SINGLE_CHOICE            │ Single string option_id       │ Must match active schema option_id;   │
│                          │                               │ triggers finding if option is flagged.│
├──────────────────────────┼───────────────────────────────┼───────────────────────────────────────┤
│ MULTI_CHOICE             │ Array of unique option_ids    │ min_selections <= count <= max;       │
│                          │                               │ duplicates rejected.                  │
├──────────────────────────┼───────────────────────────────┼───────────────────────────────────────┤
│ NUMERIC_MEASUREMENT      │ IEEE 754 float number         │ Mandatory unit label; range bounds    │
│                          │                               │ checked; out-of-bounds flags finding. │
├──────────────────────────┼───────────────────────────────┼───────────────────────────────────────┤
│ TEXT_NOTE                │ UTF-8 string (1 - 1000 chars) │ Whitespace trimmed; control chars     │
│                          │                               │ stripped; PII patterns rejected.      │
├──────────────────────────┼───────────────────────────────┼───────────────────────────────────────┤
│ EVIDENCE_ATTACHMENT      │ Array of evd_syn_* IDs        │ min_attachments <= count; MIME types  │
│                          │                               │ restricted to JPEG/PNG/PDF.           │
└──────────────────────────┴───────────────────────────────┴───────────────────────────────────────┘
```

### 3.1 Non-Applicable (`NA`) & Unknown (`UNKNOWN`) Validation (`H040-006`)
In strict adherence to foundation gate `H040-006`:
1. **Mandatory Justification for `NA`:** An inspector selecting `NA` must provide a written explanatory note (`min_length: 5`, `max_length: 500` characters). The system validates that the justification is non-blank and non-whitespace.
2. **Mandatory Justification for `UNKNOWN`:** An inspector selecting `UNKNOWN` must provide a written justification describing why the condition could not be evaluated (e.g., physical obstruction, extreme weather, locked enclosure). `UNKNOWN` responses automatically trigger a supervisory review task in `MOD-WFA` to schedule follow-up verification.

---

## 4. Evidence Attachment Policies & Enforcement

Evidence requirements are governed by deterministic question-level policies that prevent premature inspection submission:

### 4.1 Policy Definitions

| Policy Identifier | Inspection Submission Constraint | Validation Invariant |
| :--- | :--- | :--- |
| **`OPTIONAL`** | Inspection may be submitted with zero attachments. | Attachments voluntarily provided must satisfy file type and checksum integrity checks. |
| **`MANDATORY_ON_FAIL`** | Submission is blocked if response is non-compliant (`FAIL`, out-of-bounds number, or hazard option) and attachment count is zero. | At least one valid evidence reference (`evd_syn_*`) must be attached to the question response before inspection status can transition to `COMPLETED`. |
| **`MANDATORY_ALWAYS`** | Submission is blocked until at least one attachment is provided, regardless of compliance outcome. | At least one valid evidence reference (`evd_syn_*`) must be present before `COMPLETED` submission. |

### 4.2 Missing Evidence State Handling
1. If an inspector attempts to submit an inspection (`SubmitInspectionRequest`) with unsatisfied evidence obligations, the state machine rejects the transition with `ErrMandatoryEvidenceMissing`.
2. The inspection remains in `IN_PROGRESS` status and returns an explicit list of unsatisfied question codes to guide inspector remediation.

---

## 5. Question & Section Exclusion Rules

When an applicability rule evaluates to `HIDE` or `EXCLUDE`, the checklist engine enforces strict exclusion boundaries:

### 5.1 Exclusion Lifecycle & Scoring Isolation
1. **Assignment of `EXCLUDED_BY_RULE`:** Questions or sections whose applicability predicates evaluate to `HIDE` are marked with status `EXCLUDED_BY_RULE`.
2. **Exclusion from Denominators:** Excluded questions **do not count** toward the inspection's total question count and are completely excluded from scoring denominators. An excluded question causes zero negative scoring impact.
3. **Response Sanitization & Quarantining:** If a question was previously answered and subsequent context changes trigger an applicability rule to hide it, the existing answer is marked `INACTIVE_DUE_TO_EXCLUSION` and quarantined to prevent phantom non-compliance findings.

---

## 6. Provisional Scoring-Reference Contract (`HUMAN_OWNED_UNSELECTED`)

In accordance with `HDEC-V040-FOUNDATION-054` and `ARC-V040-DOMAIN-001`, scoring references in the checklist rules baseline operate strictly as **provisional descriptive references**:

```yaml
provisional_scoring_reference:
  contract_version: "1.0.0"
  governance_status: "HUMAN_OWNED_UNSELECTED"
  scoring_model_reference: "PROVISIONAL_WEIGHTED_SCORE_V1"
  formula_type: "NORMALIZED_WEIGHTED_AVERAGE"
  critical_fail_policy: "FLAG_PRIORITY_REVIEW_ONLY" # No autonomous regulatory failure
  non_conformance_weights:
    LOW: 1.0
    MEDIUM: 2.5
    HIGH: 5.0
    CRITICAL: 10.0
```

### 6.1 Non-Binding Declarations
- **Zero Legal or Regulatory Claims:** No regulatory compliance certificate, legal safety rating, or statutory pass/fail threshold is asserted by this model.
- **Human Authority Reservation:** Final binding scoring policies, passing percentages, and critical-fail triggers require explicit human owner decision under Issue #136 (`V040-I025`).

---

## 7. Safe Failure Modes & Fail-Closed Defaults

The checklist rule engine enforces fail-closed behavior across all evaluation failure modes:

### 7.1 Failure Mode Matrix

| Failure Scenario | Evaluation Posture | Safety Justification & Preventive Control |
| :--- | :--- | :--- |
| **Missing Context Metadata** | Defaults to **`SHOW`** | If static facility/area metadata is missing, the rule engine defaults to displaying the question. Safety questions are never silently hidden due to metadata omissions. |
| **Malformed Rule Predicate** | Rejects Template / Fails Closed | Templates with syntax errors, unregistered operators, or invalid question references are rejected during `DRAFT` validation. |
| **Out-of-Bounds Response** | Rejects Input (`ErrOutOfBounds`) | Numeric measurements exceeding absolute physical limits or text exceeding 1000 chars are rejected before answer commitment. |
| **Concurrency Sync Conflict** | `QUARANTINED` (`H040-005`) | Conflicting concurrent updates to an inspection response are quarantined for manual human reconciliation; zero Last-Write-Wins. |
| **Unauthorized Action Attempt** | `DEFAULT_DENY` (`H040-004`) | Any mutation attempted without the required role (`Checklist Author`, `Inspector`) fails closed with `ErrUnauthorizedAction`. |

---

## 8. Template Versioning & Immutability Boundaries

1. **Published Immutability (`PUBLISHED_IMMUTABLE`):**
   - Once published, a checklist template's sections, questions, applicability rules, and scoring references are permanently immutable.
   - Any modification requires creating a new SemVer version (`major.minor.patch`) linking to the prior version via `predecessor_version`.
2. **Execution Pinning:**
   - Active inspections are permanently pinned to the exact checklist template version instantiated at inspection creation.
   - Runtime hot-swapping or background mutation of active inspection rules is strictly prohibited.

---

## 9. Synthetic Conditional Rules Fixture

The following synthetic YAML fixture demonstrates the complete, bounded conditional rule model:

```yaml
schema_version: "1.0.0"
fixture_id: "fix_syn_rules_pilot_v1"
template_id: "chk_syn_pilot_plant_safety_v1"
template_version: "1.0.0"

rules:
  - rule_id: "rule_syn_confined_space_applicability"
    target_entity: "SECTION"
    target_id: "sec_syn_confined_space_02"
    predicate_type: "METADATA_MATCH"
    target_field: "area_hazard_classification"
    operator: "IN"
    comparison_values:
      - "CONFINED_SPACE"
      - "HAZARDOUS_ATMOSPHERE"
    action: "SHOW"
    default_fallback: "SHOW"
    description:
      en-US: "Show confined space section only in areas with confined space hazard classification."
      th-TH: "แสดงส่วนการตรวจสอบพื้นที่อับอากาศเฉพาะในพื้นที่ที่มีการจำแนกประเภทอันตรายพื้นที่อับอากาศ"

  - rule_id: "rule_syn_ground_resistance_finding"
    target_entity: "QUESTION"
    target_id: "qst_syn_ground_resistance_01"
    predicate_type: "METADATA_MATCH"
    target_field: "numeric_value"
    operator: "GREATER_THAN"
    comparison_values:
      - "5.0"
    action: "TRIGGER_FINDING"
    default_fallback: "NONE"
    finding_spec:
      severity: "HIGH"
      provisional_criticality: "CRITICAL_FAIL_TRIGGER"
      title:
        en-US: "Ground Loop Resistance Exceeds Safe Baseline (> 5.0 ohms)"
        th-TH: "ค่าความต้านทานการต่อลงดินสูงเกินเกณฑ์ปลอดภัย (> 5.0 โอห์ม)"
    description:
      en-US: "Trigger automatic high-severity finding if ground resistance exceeds 5.0 ohms."
      th-TH: "สร้างข้อบกพร่องระดับสูงอัตโนมัติหากค่าความต้านทานดินเกิน 5.0 โอห์ม"

  - rule_id: "rule_syn_scaffold_tag_conditional_display"
    target_entity: "QUESTION"
    target_id: "qst_syn_scaffold_inspector_tag_03"
    predicate_type: "PRIOR_ANSWER_MATCH"
    target_field: "qst_syn_scaffold_brakes_01"
    operator: "EQUALS"
    comparison_values:
      - "PASS"
    action: "SHOW"
    default_fallback: "HIDE"
    description:
      en-US: "Prompt for scaffold inspection green tag only if wheel brakes pass."
      th-TH: "แสดงคำถามเรื่องป้ายเขียวตรวจผ่านนั่งร้านเฉพาะเมื่อระบบเบรกล้อผ่านการตรวจสอบ"
```

---

## 10. Governance Boundaries & Non-Claims

In strict compliance with `HDEC-V040-FOUNDATION-054`:

1. **Synthetic-Only Data Policy (`H040-003`):** All rule IDs (`rule_syn_*`), questions, sections, and synthetic test payloads are fictionalized alpha fixtures. Zero customer data, real corporate records, or actual employee PII is included.
2. **Default-Deny Authority (`H040-004`):** Rule configuration and state transitions evaluate under strict default-deny.
3. **No Generalized No-Code Platform:** As declared in Section 4 of `ARC-V040-CHKL-001`, this rule model rejects dynamic form builder engines and unconstrained script execution.
4. **Non-Binding Scoring References:** All weights, thresholds, and severity ratings are descriptive and provisional (`HUMAN_OWNED_UNSELECTED`).
5. **No Deployment or Release Claim:** Gates `H040-007` through `H040-011` remain on `HOLD`. Zero public routes, internet endpoints, CDN activations, or production software deployments are claimed or authorized.
