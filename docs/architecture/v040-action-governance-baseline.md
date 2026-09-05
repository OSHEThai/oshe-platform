---
document_id: ARC-V040-ACTGOV-001
title: v0.4.0 OSHE Inspect Corrective Action Ownership, Assignment, Reassignment, Extension, Escalation, Evidence Governance, and Responsibility Baseline
document_type: architecture_specification
document_version: 1.0.0
lifecycle_status: DRAFT
status: APPROVED_FOR_LOCAL_DEVELOPMENT
date: "2026-09-05"
author_role: Engineering Lead
author_pane: w9:p23
governing_issue: "GitHub Issue #133"
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
  binding_due_date_policy: HUMAN_OWNED_UNSELECTED
  binding_extension_policy: HUMAN_OWNED_UNSELECTED
  binding_escalation_policy: HUMAN_OWNED_UNSELECTED
  binding_evidence_verification_policy: HUMAN_OWNED_UNSELECTED
credit_boundary: ACTION_GOVERNANCE_SPECIFICATION_ONLY_NO_EXTERNAL_DELIVERY_OR_LIVE_OPERATIONAL_ENACTMENT
---

# v0.4.0 OSHE Inspect Corrective Action Ownership, Assignment, Reassignment, Extension, Escalation, Evidence Governance, and Responsibility Baseline

## 1. Executive Summary & Governance Posture

### 1.1 Authority Baseline & Purpose
This architectural specification establishes the authoritative **Corrective Action (CAPA) Ownership, Assignment, Reassignment, Extension, Escalation, Evidence Governance, and Responsibility History Baseline** for **Milestone v0.4.0 - OSHE Inspect Private Alpha** within `OSHEThai/oshe-platform`. It fulfills the deliverable requirements of **GitHub Issue #133 (`[V040-I022] Corrective Action Ownership, Assignment, and Evidence Governance Prework`)** under Roadmap Topic `V040-T05` and standing owner foundation decision `HDEC-V040-FOUNDATION-054`.

The primary objective is to define a bounded, deterministic, fail-closed governance model within the **Workflow and Action Module (`MOD-WFA`)** governing:
- Visible, unbroken ownership and reassignment history preserving complete inspector and owner custody.
- Least-privilege role boundaries and strict segregation of duties (`SOD-03`, `SOD-04`).
- Generic request and approval workflows for due-date extensions and operational escalations.
- Rigorous evidence submission, independent review, acceptance, and rejection lifecycles.
- Optimistic concurrency control and duplicate prevention.

### 1.2 Retained Unselected Policies & Non-Claims
In strict compliance with `ASN-V040-I022-ACTION-GOVERNANCE-002` and `HDEC-V040-FOUNDATION-054`:
1. **Binding Scoring Policy (`HUMAN_OWNED_UNSELECTED`):** Action resolution scoring models remain provisional under Issue #136 (`V040-I025`).
2. **Finding Closure Policy (`HUMAN_OWNED_UNSELECTED`):** Formal finding closure authorization and residual safety risk acceptance remain human-owned under Issue #134 (`V040-I023`).
3. **Offline Authority Model (`HUMAN_OWNED_UNSELECTED`):** Client-side action cache validity and offline action editing limits remain human-owned under Issue #126 (`V040-I015`).
4. **Binding Due-Date, Extension, Escalation, and Evidence Policies (`HUMAN_OWNED_UNSELECTED`):** The workflows defined herein provide generic governance structures; corporate SLAs, mandatory escalation hierarchies, and formal evidentiary standards remain unselected pending explicit owner decisions.
5. **Retained Human Gate Holds:** Foundation gates `H040-007` (Technical Release Authorization), `H040-008` (Real Participant / Private-Alpha UAT Onboarding), `H040-009` (Binding Support Ownership), `H040-010` (External Environment/Route Activation), and `H040-011` (Final Milestone Outcome) remain strictly on **`HOLD`**.

---

## 2. Corrective Action Entity Model & State Machine (`MOD-WFA`)

Under `modules/workflow-action/action_governance.go`, the `GovernedAction` entity manages the complete lifecycle of a corrective action item:

```
┌────────────────────────────────────────────────────────────────────────┐
│                             GovernedAction                             │
│  - action_id: String (act_syn_*)                                       │
│  - tenant_id: String (ten_syn_*)                                       │
│  - finding_id: String (fnd_syn_*)                                      │
│  - title: String                                                       │
│  - state: ActionState (ASSIGNED, IN_PROGRESS, IN_REVIEW, ...)          │
│  - current_owner: String (usr_syn_capa_*)                              │
│  - current_owner_role: String ("CAPA_OWNER")                           │
│  - due_date: Timestamp (UTC ISO 8601)                                  │
│  - state_version: Int64 (Monotonic concurrency version)                │
│  - ownership_history: []OwnershipRecord                                │
│  - extension_requests: []ExtensionRequest                              │
│  - escalation_requests: []EscalationRequest                            │
│  - evidence_list: []GovernedEvidence                                   │
│  - required_evidence_count: Int                                        │
└────────────────────────────────────────────────────────────────────────┘
```

### 2.1 Action Lifecycle States

| State | Mutability | Evidence Submission | State Semantics & Boundary |
| :--- | :--- | :--- | :--- |
| **`ASSIGNED`** | Editable | Allowed | Action created and assigned to an active CAPA Owner. Work has not yet commenced. |
| **`IN_PROGRESS`** | Editable | Allowed | CAPA Owner has actively commenced remediation tasks. |
| **`IN_REVIEW`** | Read-Only | Frozen | Remediation completed; submitted for formal verification by an Independent Reviewer. |
| **`REJECTED`** | Editable | Allowed | Submitted evidence or remediation rejected by Reviewer. Returned to owner for rework. |
| **`CLOSED`** | Read-Only | Disallowed | Terminal state. Verified and signed off by authorized Reviewer. Double-closure prevented. |
| **`REOPENED`** | Editable | Allowed | Reopened by supervisory authority following finding re-inspection failure. |
| **`OVERDUE`** | Editable | Allowed | Current evaluation time exceeds `due_date` without approved extension. |

---

## 3. Synthetic Visible Ownership, Reassignment, and Revocation Lineage

In accordance with foundation gate `H040-004`, responsibility for a corrective action is tracked through an unbroken, append-only custody trail:

### 3.1 Ownership Lineage Invariant
1. **Never Erased or Overwritten:** Reassigning or revoking action ownership **never overwrites or purges** prior ownership records.
2. **Attributed Reassignment:** Every reassignment requires an explicit caller subject, authorized supervisory role, and mandatory justification reason (`reason`).
3. **State Version Advance:** Reassignment advances `state_version` monotonically, preventing concurrent lost updates.

### 3.2 Revocation Protocol
- If an assigned owner is rotated off-site or contract terminated, an authorized supervisor executes `RevokeOwner`.
- The current owner record is marked `is_active = false` with `revoked_at` and `revocation_reason`.
- The action transitions to an unassigned state (`current_owner = ""`), preserving all accumulated evidence and history until a new owner is designated.

---

## 4. Least Privilege & Segregation of Duties (`SOD-03` / `SOD-04`)

Under `ARC-V040-DOMAIN-001`, strict role separation governs action operations:

1. **CAPA Owner:** Authorized to commence work, submit remedial evidence, and submit requests for due-date extension or escalation.
2. **Independent Reviewer:** Authorized to inspect evidence, accept/reject evidence items, review extension requests, and close or reopen actions.
3. **Supervisor / Manager:** Authorized to assign, reassign, or revoke action ownership, and acknowledge operational escalations.
4. **The Self-Approval Prohibition Invariant (`ErrSelfApprovalProhibited`):**
   - A CAPA Owner **cannot review or approve their own evidence submissions**.
   - A CAPA Owner **cannot approve their own due-date extension requests**.
   - A CAPA Owner **cannot self-acknowledge operational escalations**.
   - Any attempt at self-approval or self-review immediately fails closed.

---

## 5. Generic Due-Date Extension & Escalation Framework

### 5.1 Due-Date Extension Lifecycle
```
┌─────────────┐       Owner Request        ┌─────────────┐       Reviewer Review       ┌──────────────┐
│   ACTIVE    │ ─────────────────────────> │   PENDING   │ ──────────────────────────> │   APPROVED   │ (DueDate Advanced)
└─────────────┘                            └──────┬──────┘                             └──────────────┘
                                                  │                                    ┌──────────────┐
                                                  └──────────────────────────────────> │   REJECTED   │ (DueDate Retained)
                                                                                       └──────────────┘
```

1. **Request Rules:**
   - Must specify non-blank `request_id`, valid `reason`, and a `requested_due_date` strictly after the current `due_date`.
   - Duplicate request IDs fail closed with `ErrDuplicateRequestID`.
2. **Approval Outcome:**
   - If approved by an independent reviewer, `action.due_date` is updated to `requested_due_date`.
   - If rejected, `action.due_date` remains strictly unchanged, and rejection notes are captured.

### 5.2 Operational Escalation Handling
1. **Trigger:** An owner or inspector can escalate an action experiencing blockers (e.g. hazardous material leak, supply chain outage).
2. **Acknowledgment:** A supervisory role acknowledges the escalation, recording formal resolution notes and mitigation measures.

---

## 6. Evidence Governance: Submit, Review, Accept, and Reject

Evidence submitted to substantiate corrective action remediation is governed by rigorous verification states:

### 6.1 Evidence Review States
- **`SUBMITTED`:** Evidence uploaded by CAPA Owner with SHA-256 digest and media type. Awaits formal inspection.
- **`ACCEPTED`:** Verified by Independent Reviewer. Increments `AcceptedEvidenceCount()`.
- **`REJECTED`:** Denied by Independent Reviewer with mandatory rejection notes (e.g. photo illegible, incorrect equipment unit). Requires corrective rework.

### 6.2 Action Closure Dependency
An action cannot transition to `CLOSED` until:
$$\text{AcceptedEvidenceCount()} \ge \text{required\_evidence\_count}$$
Unreviewed or rejected evidence cannot satisfy closure criteria.

---

## 7. Optimistic Concurrency & Duplicate Prevention

To eliminate race conditions in multi-user and offline-reconnecting environments:

1. **State Versioning:** Every mutation requires passing `expectedVersion`. If `expectedVersion != action.state_version`, the operation is aborted with `ErrConcurrentModification`.
2. **Unique Constraints:**
   - `action_id`: strictly unique across the engine.
   - `evidence_id`: strictly unique per action (`ErrDuplicateEvidenceID`).
   - `request_id`: strictly unique per action (`ErrDuplicateRequestID`).

---

## 8. Synthetic Operations Fixture Matrix

The following synthetic YAML fixture illustrates complete action governance scenarios:

```yaml
schema_version: "1.0.0"
fixture_id: "fix_syn_action_governance_v1"

scenarios:
  # Scenario 1: Ownership Reassignment with Preserved Lineage
  - scenario_id: "scen_gov_reassign_01"
    action_id: "act_syn_fire_barrier_01"
    initial_owner: "usr_capa_owner_01"
    reassigned_owner: "usr_capa_owner_02"
    reassigned_by: "usr_supervisor_01"
    reason: "Primary contractor rotated off-site"
    history_records_count: 2
    prior_owner_active: false
    new_owner_active: true

  # Scenario 2: Approved Due-Date Extension
  - scenario_id: "scen_gov_extension_02"
    action_id: "act_syn_scaffold_tag_03"
    requested_by: "usr_capa_owner_01"
    reviewed_by: "usr_reviewer_01"
    status: "APPROVED"
    initial_due_date: "2026-09-06T10:00:00Z"
    updated_due_date: "2026-09-08T10:00:00Z"
    self_approval_prevented: true

  # Scenario 3: Evidence Submission and Formal Acceptance
  - scenario_id: "scen_gov_evidence_03"
    action_id: "act_syn_extinguisher_07"
    evidence_id: "evd_capa_recharge_photo_01"
    submitted_by: "usr_capa_owner_01"
    reviewed_by: "usr_reviewer_01"
    status: "ACCEPTED"
    accepted_evidence_count: 1
    required_evidence_count: 1

  # Scenario 4: Optimistic Concurrency Conflict Rejection
  - scenario_id: "scen_gov_concurrency_04"
    action_id: "act_syn_concurrency_08"
    initial_version: 1
    mutation_1_version: 1 # Succeeds, advances version to 2
    mutation_2_version: 1 # Fails with ErrConcurrentModification
    expected_error: "ErrConcurrentModification"
```

---

## 9. Governance Boundaries, Prohibitions & Operational Non-Claims

In strict compliance with `HDEC-V040-FOUNDATION-054` and `ASN-V040-I022-ACTION-GOVERNANCE-002`:

1. **100% Synthetic Data Policy (`H040-003`):** All users, actions, findings, and evidence references are synthetic models. Zero real employee records or corporate data are used.
2. **Segregation of Duties Enforced (`SOD-03`, `SOD-04`):** Self-approval of extensions, escalations, or evidence review is programmatically blocked.
3. **Preservation of Gate `H040-010` (HOLD):** Zero external notification routes, email/SMS services, or cloud providers are activated.
4. **No Participant Onboarding (`H040-008` HOLD):** Zero real plant personnel or contractors are recruited or onboarded.
5. **Specification-Only Credit:** Delivery of this baseline confers documentation and architectural baseline credit only; zero operational release, deployment, or residual-risk acceptance is claimed.
