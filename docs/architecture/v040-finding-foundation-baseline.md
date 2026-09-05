---
document_id: ARC-V040-FND-001
title: v0.4.0 OSHE Inspect Finding Foundation, Deterministic Creation, Type, Severity, Critical Flag, Immediate-Control Note, and Recurrence Reference Baseline
document_type: architecture_specification
document_version: 1.0.0
lifecycle_status: DRAFT
status: APPROVED_FOR_LOCAL_DEVELOPMENT
date: "2026-09-06"
author_role: Security Privacy and Product Safety Lead
author_pane: w9:p13
governing_issue: "GitHub Issue #132"
authority_source: HDEC-V040-FOUNDATION-054
governing_decisions:
  - HDEC-V040-FOUNDATION-054
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
integrated_baselines:
  - "ARC-V040-CHKRULE-001 (docs/architecture/v040-checklist-rules-baseline.md)"
  - "ARC-V040-EVD-003 (docs/architecture/v040-evidence-integrity-baseline.md)"
retained_unselected_policies:
  binding_scoring_policy: HUMAN_OWNED_UNSELECTED
  finding_closure_policy: HUMAN_OWNED_UNSELECTED
  offline_authority: HUMAN_OWNED_UNSELECTED
  evidence_retention_policy: HUMAN_OWNED_UNSELECTED
  external_storage_provider_policy: HUMAN_OWNED_UNSELECTED
credit_boundary: FINDING_FOUNDATION_BASELINE_SPECIFICATION_ONLY_NO_BINDING_SEVERITY_OR_CLOSURE_SELECTION
---

# v0.4.0 OSHE Inspect Finding Foundation, Deterministic Creation, Type, Severity, Critical Flag, Immediate-Control Note, and Recurrence Reference Baseline

## 1. Executive Summary & Governance Posture

### 1.1 Authority Baseline & Purpose
This architectural specification establishes the authoritative **Finding Foundation, Deterministic Creation, Type, Severity, Critical Flag, Immediate-Control Note, and Recurrence Reference Baseline** for **Milestone v0.4.0 - OSHE Inspect Private Alpha** within `OSHEThai/oshe-platform`. It fulfills the deliverable requirements of **GitHub Issue #132 (`[V040-I021] Implement Deterministic Finding Creation, Type, Severity, Critical Flag, Immediate-Control Note, and Recurrence Reference`)** under Roadmap Topic `V040-T05` and standing owner foundation decision `HDEC-V040-FOUNDATION-054`.

The primary objective is to define an integrated, conservative, fail-closed finding creation and governance model within the **Workflow and Action Module (`MOD-WFA`)**, guaranteeing:
- **Deterministic Finding Generation:** Automatic, reproducible finding creation from concerning or failed checklist question responses using registered, version-pinned rules.
- **Topological Traceability:** Strict provenance linking every finding to its parent inspection execution (`execution_id`), specific question (`question_id`), raw response (`response_id`), and recurring historical non-conformance (`recurrence_id`).
- **Owner-Controlled Severity & Critical Catalog:** Severity (`LOW`, `MEDIUM`, `HIGH`, `CRITICAL`) and imminent danger (`critical_flag`) represented as configurable, owner-governed rule inputs rather than autonomous AI outputs.
- **Mandatory Immediate Controls:** Enforced documentation of temporary risk mitigations for all critical and high-severity findings prior to record acceptance.
- **Fail-Closed Governance Protections:** Strict prohibition of autonomous AI classification, silent suppression, unauthorized downgrade, or automated closure.

### 1.2 Retained Unselected Policies & Non-Claims
In strict compliance with `ASN-V040-I021-FINDING-FOUNDATION-001` and `HDEC-V040-FOUNDATION-054`:
1. **Finding Closure Policy (`HUMAN_OWNED_UNSELECTED`):** Regulatory sign-off criteria, verification standards, and residual risk acceptance remain strictly human-owned pending explicit owner determination under Issue #134 (`V040-I023`).
2. **Binding Severity & Critical Threshold Policy (`HUMAN_OWNED_UNSELECTED`):** Binding corporate severity thresholds, regulatory reporting triggers, and insurance notification rules remain human-owned pending Gate `H040-004`.
3. **No Real Operational Data (`H040-003`):** 100% synthetic finding test fixtures only; zero real plant hazard logs or workforce observations.
4. **Retained Human Gate Holds:** Foundation gates `H040-007` (Technical Release Authorization), `H040-008` (Real Participant / Private-Alpha UAT Onboarding), `H040-009` (Binding Support Ownership), `H040-010` (External Environment/Route Activation), and `H040-011` (Final Milestone Outcome) remain strictly on **`HOLD`**.

---

## 2. Deterministic Finding Creation Engine & Topological Model

Under `MOD-WFA`, findings cannot be generated arbitrarily or detached from authoritative inspection evidence. Every finding is instantiated through deterministic rules:

### 2.1 Entity Model Specification
```
┌────────────────────────────────────────────────────────────────────────┐
│                             FindingRecord                              │
│  - finding_id: String (fnd_syn_[0-9a-f]{16})                           │
│  - tenant_id: String (ten_syn_*)                                       │
│  - execution_id: String (ins_syn_*)                                    │
│  - question_id: String (qst_syn_*)                                     │
│  - response_id: String (rsp_syn_*)                                     │
│  - recurrence_id: String (fnd_syn_*, optional)                         │
│  - rule_id: String (RULE-FND-*)                                        │
│  - rule_version: String ("1.0.0")                                      │
│  - title: LocalizedString (en-US, th-TH)                               │
│  - description: String                                                 │
│  - severity: Enum (LOW, MEDIUM, HIGH, CRITICAL)                        │
│  - critical_flag: Boolean                                              │
│  - immediate_control: String (mandatory if critical or high)           │
│  - state: Enum (OPEN, UNDER_REVIEW, REMEDIATED, CLOSED)                │
│  - evidence_ids: List[String] (evd_syn_*)                              │
│  - created_at: Timestamp (UTC ISO 8601)                                │
│  - created_by: String (usr_syn_inspector_*)                            │
│  - history: List[FindingAuditEntry]                                    │
└────────────────────────────────────────────────────────────────────────┘
```

### 2.2 Creation Invariants & Fail-Closed Guards
1. **Rule Registration Prerequisite (`ErrRuleNotFound`):** A finding can only be generated by referencing an approved rule in the `FindingRuleCatalog`. Unregistered rules fail closed immediately.
2. **Version Pinning Invariant (`ErrIncompatibleRuleVersion`):** The evaluated rule version must match the active checklist matrix version (`1.0.0`). Stale or deprecated versions fail closed.
3. **Source Context Invariant (`ErrMissingSourceContext`):** Attempting to create a finding without an active `execution_id`, `question_id`, and `response_id` fails closed.
4. **Recurrence Verification (`ErrInvalidRecurrenceLink`):** If a `recurrence_id` is supplied, the target finding must exist within the identical tenant scope.

---

## 3. Severity Catalog & Critical-Condition Model (Owner-Controlled)

In strict accordance with Gate `H040-004`, severity and critical classifications are modeled as owner-controlled inputs:

| Severity Level | Nominal Impact Description | Immediate Control Mandatory | Escalation Authority | Downgrade Authority |
| :--- | :--- | :--- | :--- | :--- |
| **`LOW`** | Minor procedural or housekeeping observation | No | Inspector / Lead | Supervisor / Admin |
| **`MEDIUM`** | Non-immediate hazard; potential regulatory non-compliance | Optional | Inspector / Lead | Supervisor / Admin |
| **`HIGH`** | Major safety hazard; significant potential for injury or loss | **Yes** | Inspector / Lead | Supervisor / Admin |
| **`CRITICAL`** | Imminent danger; life-safety threat; stop-work condition | **Yes** | Inspector / Lead | Supervisor / Admin |

### 3.1 Immediate Control Requirement (`ErrMissingImmediateControl`)
When a finding is marked with `critical_flag: true`, `severity: CRITICAL`, or when required by rule, an `immediate_control` description is strictly mandatory (e.g. `"Hazardous area cordoned with red tape, circuit breaker tagged out, warning notice posted"`). Creation without this field fails closed.

### 3.2 Downgrade Protection Mechanics
1. **Escalation Allowed:** Any assigned inspector may escalate finding severity or raise the critical flag upon discovering worsening site hazards.
2. **Downgrade Guard (`ErrUnauthorizedSeverityDowngrade`):** Lowering severity or unsetting the critical flag is prohibited for frontline inspectors. It requires supervisory role authorization (`SAFETY_SUPERVISOR`, `QUALITY_LEAD`, `OWNER`).
3. **Mandatory Rationale (`ErrMissingClassificationRationale`):** Any authorized downgrade must supply a non-blank audit rationale explaining why the risk was re-assessed.

---

## 4. Human Authority Invariants & Autonomous Prohibitions

To preserve system safety and prevent AI hallucination or silent suppression:

1. **Prohibition of Autonomous AI Classification (`ErrAutonomousClassificationProhibited`):** AI agents may summarize evidence or propose checklist answers, but may never autonomously classify finding severity or set/unset the critical flag without human rule execution.
2. **Prohibition of Silent Hiding (`ErrSilentHidingProhibited`):** Responses marked as failed or concerning must deterministically instantiate a finding. Silent suppression, exclusion, or concealment of non-conformances is strictly blocked.
3. **Prohibition of Autonomous AI Closure (`ErrAutonomousClosureProhibited`):** Automated or AI-driven closure of findings is prohibited. Findings may only transition to `CLOSED` upon verified human supervisory sign-off (`ErrUnauthorizedClosure`).

---

## 5. Finding Lifecycle State Machine & Audit History

```
┌──────────┐      Submit CAPA / Review       ┌────────────────┐      Verification OK      ┌────────────┐
│   OPEN   │ ──────────────────────────────> │  UNDER_REVIEW  │ ────────────────────────> │ REMEDIATED │
└────┬─────┘                                 └───────┬────────┘                           └─────┬──────┘
     │                                               │                                          │
     │ Urgent Fix Verified                           │ Re-work Required                         │
     │                                               ▼                                          │
     │                                       ┌────────────────┐                                 │
     │                                       │      OPEN      │                                 │
     │                                       └────────────────┘                                 │
     │                                                                                          │
     └───────────────────────────────► ┌──────────┐ ◄───────────────────────────────────────────┘
                                       │  CLOSED  │ (Human Supervisor Sign-off Only)
                                       └──────────┘
```

### 5.1 Linear State Progression
- `OPEN`: Newly created finding awaiting remediation or review.
- `UNDER_REVIEW`: Remedial action submitted; pending supervisory verification.
- `REMEDIATED`: Corrective action physically verified with attached evidence.
- `CLOSED`: Formally signed off by human safety authority. Once closed, the record is immutable (`ErrFindingAlreadyClosed`).

### 5.2 Append-Only Audit Ledger
Every state transition, severity change, critical flag modification, and evidence attachment appends an immutable entry to `FindingAuditEntry` recording monotonic sequence, UTC timestamp, actor ID, role, action code, and rationale.

---

## 6. Synthetic Multi-Scenario Fixtures & Verification Matrix

```yaml
schema_version: "1.0.0"
fixture_id: "fix_syn_finding_foundation_v1"

findings:
  # Scenario 1: Critical Imminent Hazard (Blocked Fire Exit)
  - finding_id: "fnd_syn_fire_exit_01"
    tenant_id: "ten_syn_01"
    execution_id: "ins_syn_clean_01"
    question_id: "qst_fire_exit_01"
    response_id: "rsp_syn_blocked_exit_01"
    rule_id: "RULE-FND-FIRE-EXIT-BLOCKED"
    rule_version: "1.0.0"
    title: "Blocked Emergency Exit Corridor"
    description: "Wooden pallets obstruct fire exit B2 in plant warehouse."
    severity: "CRITICAL"
    critical_flag: true
    immediate_control: "Pallets cordoned off with yellow tape, warning sign placed, facility team dispatched to clear."
    state: "OPEN"
    evidence_ids: ["evd_syn_orig_photo_01"]

  # Scenario 2: High Severity Defect (Extinguisher Out of Service)
  - finding_id: "fnd_syn_extinguisher_02"
    tenant_id: "ten_syn_01"
    execution_id: "ins_syn_clean_01"
    question_id: "qst_extinguisher_01"
    response_id: "rsp_syn_extinguisher_press_01"
    rule_id: "RULE-FND-EXTINGUISHER-DEFECT"
    rule_version: "1.0.0"
    title: "Fire Extinguisher Depressurized"
    description: "Pressure gauge indicates needle below operational green zone."
    severity: "HIGH"
    critical_flag: false
    immediate_control: "Unit tagged out of service; temporary backup extinguisher placed."
    state: "OPEN"
    evidence_ids: ["evd_syn_orig_photo_02"]

  # Scenario 3: Medium Recurrence Observation (PPE Non-Compliance)
  - finding_id: "fnd_syn_ppe_recurrence_03"
    tenant_id: "ten_syn_01"
    execution_id: "ins_syn_clean_01"
    question_id: "qst_ppe_01"
    response_id: "rsp_syn_ppe_fail_01"
    recurrence_id: "fnd_syn_ppe_prior_week"
    rule_id: "RULE-FND-PPE-NONCOMPLIANCE"
    rule_version: "1.0.0"
    title: "Recurrence: Missing Eye Protection in Grinding Bay"
    description: "Worker observed operating bench grinder without safety glasses."
    severity: "MEDIUM"
    critical_flag: false
    immediate_control: "Work stopped immediately; proper ANSI Z87.1 glasses issued."
    state: "OPEN"

denial_scenarios:
  - scenario_id: "den_fnd_01_unregistered_rule"
    rule_id: "RULE-UNREGISTERED-HAZARD"
    expected_error: "ErrRuleNotFound"

  - scenario_id: "den_fnd_02_stale_rule_version"
    rule_version: "0.9.0-DEPRECATED"
    expected_error: "ErrIncompatibleRuleVersion"

  - scenario_id: "den_fnd_03_missing_immediate_control"
    critical_flag: true
    immediate_control: ""
    expected_error: "ErrMissingImmediateControl"

  - scenario_id: "den_fnd_04_unauthorized_downgrade"
    actor_role: "INSPECTOR"
    action: "DOWNGRADE_CRITICAL_TO_LOW"
    expected_error: "ErrUnauthorizedSeverityDowngrade"

  - scenario_id: "den_fnd_05_autonomous_ai_closure"
    actor: "AI_AGENT_CORE"
    action: "CLOSE_FINDING"
    expected_error: "ErrAutonomousClosureProhibited"
```

---

## 7. Governance Boundaries, Retained Holds & Non-Claims

In strict compliance with `HDEC-V040-FOUNDATION-054` and `ASN-V040-I021-FINDING-FOUNDATION-001`:

1. **100% Synthetic Data Policy (`H040-003`):** All finding IDs, descriptions, and audit trails are synthetic fixtures. Zero real operational non-conformances or personnel records are utilized.
2. **No Binding Severity or Closure Policy Selection:** Delivery of this baseline establishes architectural contracts and deterministic rule plumbing only; final corporate severity matrices and regulatory closure thresholds remain human-owned.
3. **No External Route or Cloud Bucket Activation (`H040-007` & `H040-010` HOLD):** Zero external API endpoints or cloud storage routes are activated.
4. **Issue Closure Prohibition:** Issue #132 remains open following this draft pull request until formal independent review and merge completion.
5. **Specification-Only Credit:** Delivery confers architectural and unit-test prework credit only; zero operational release, deployment, or residual-risk acceptance is claimed.
