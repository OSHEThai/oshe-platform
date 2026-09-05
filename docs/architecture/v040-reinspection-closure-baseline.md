---
document_id: ARC-V040-REINSP-001
title: v0.4.0 OSHE Inspect Reinspection Assignment, Independent Review, Rejection, Reopen, Closure, Stale-State Protection, and Recurrence Handling Baseline
document_type: architecture_specification
document_version: 1.0.0
lifecycle_status: DRAFT
status: APPROVED_FOR_LOCAL_DEVELOPMENT
date: "2026-09-06"
author_role: Security Privacy and Product Safety Lead
author_pane: w9:p13
governing_issue: "GitHub Issue #134"
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
integrated_baselines:
  - "ARC-V040-FND-001 (docs/architecture/v040-finding-foundation-baseline.md)"
  - "ARC-V040-ACTGOV-001 (docs/architecture/v040-action-governance-baseline.md)"
  - "ARC-V040-EVD-003 (docs/architecture/v040-evidence-integrity-baseline.md)"
retained_unselected_policies:
  binding_scoring_policy: HUMAN_OWNED_UNSELECTED
  finding_closure_policy: HUMAN_OWNED_UNSELECTED
  offline_authority: HUMAN_OWNED_UNSELECTED
  independent_review_policy: HUMAN_OWNED_UNSELECTED
  reinspection_criteria_policy: HUMAN_OWNED_UNSELECTED
credit_boundary: REINSPECTION_CLOSURE_SPECIFICATION_ONLY_NO_EXTERNAL_DELIVERY_OR_LIVE_OPERATIONAL_ENACTMENT
---

# v0.4.0 OSHE Inspect Reinspection Assignment, Independent Review, Rejection, Reopen, Closure, Stale-State Protection, and Recurrence Handling Baseline

## 1. Executive Summary & Governance Posture

### 1.1 Authority Baseline & Purpose
This architectural specification establishes the authoritative **Reinspection Assignment, Independent Review, Rejection, Reopen, Closure, Stale-State Protection, and Recurrence Handling Baseline** for **Milestone v0.4.0 - OSHE Inspect Private Alpha** within `OSHEThai/oshe-platform`. It fulfills the deliverable requirements of **GitHub Issue #134 (`[V040-I023] Implement Reinspection Assignment, Independent Review, Rejection, Reopen, Closure, Stale-State Protection, and Recurrence Handling`)** under Roadmap Topic `V040-T05` and standing owner foundation decision `HDEC-V040-FOUNDATION-054`.

The primary objective is to define a mathematically conservative, fail-closed reinspection and closure model within the **Workflow and Action Module (`MOD-WFA`)**, guaranteeing:
- **Mandatory Reinspection Prerequisite Gates:** No safety finding or corrective action item can advance to final closure without satisfying explicit prerequisite verification gates, accepted evidentiary proof, and valid state history.
- **Configurable Inspector Independence Boundaries:** Enforced segregation of duties ensuring that reinspection verification satisfies tenant-configured independence rules (`SAME_INSPECTOR_ALLOWED`, `DIFFERENT_INSPECTOR_REQUIRED`, `THIRD_PARTY_REQUIRED`), strictly barring self-certification by corrective action owners.
- **Fail-Closed Stale-State & Concurrency Protection:** Strict optimistic concurrency versioning (`StateVersion`) that rejects out-of-order, race-conditioned, or stale updates, completely prohibiting last-write-wins semantics.
- **Absolute Prohibition of Offline Final Closure:** While preliminary field reinspection notes and observations may be captured in local offline drafts, **final closure is a protected state transition** that strictly fails closed if attempted offline. Final closure requires live authoritative server validation and confirmed tamper-free evidence verification.
- **Deterministic Rejection, Reopening, and Recurrence Lineage:** Unbroken, append-only history tracking all reinspection rejections for deficient remediations, supervisory reopenings of closed findings upon latent defect discovery, and bidirectional recurrence links tracking chronic workplace hazards.

### 1.2 Retained Unselected Policies & Non-Claims
In strict compliance with `ASN-V040-I023-REINSPECTION-001` and `HDEC-V040-FOUNDATION-054`:
1. **Finding Closure Policy (`HUMAN_OWNED_UNSELECTED`):** Regulatory sign-off criteria, statutory verification standards, and residual risk acceptance remain strictly human-owned pending explicit owner determination.
2. **Independent Review Policy (`HUMAN_OWNED_UNSELECTED`):** Specific enterprise risk thresholds triggering mandatory third-party independent audits remain human-owned; the engine provides generic configurable independence boundaries without pre-empting owner policy.
3. **Offline Authority Model (`HUMAN_OWNED_UNSELECTED`):** Client-side offline cache duration and offline conflict resolution policies remain unselected under Issue #126 (`V040-I015`); offline final closure is unconditionally denied.
4. **Zero Autonomous AI Decisions (`H040-004`):** Autonomous AI agents are strictly prohibited from evaluating reinspection adequacy, approving extensions, downgrading deficiencies, or executing final finding closure (`ErrAutonomousClosureProhibited`).
5. **No Real Operational Data (`H040-003`):** 100% synthetic reinspection test fixtures only; zero real plant hazard logs or workforce observations.
6. **Retained Human Gate Holds:** Foundation gates `H040-007` (Technical Release Authorization), `H040-008` (Real Participant / Private-Alpha UAT Onboarding), `H040-009` (Binding Support Ownership), `H040-010` (External Environment/Route Activation), and `H040-011` (Final Milestone Outcome) remain strictly on **`HOLD`**.

---

## 2. Reinspection & Protected Closure Domain Model (`MOD-WFA`)

Under `modules/workflow-action/reinspection_lifecycle.go`, the reinspection workflow coordinates between findings (`FindingRecord`), actions (`GovernedAction`), and independent verifiers:

```
┌────────────────────────────────────────────────────────────────────────┐
│                          ReinspectionOrder                             │
│  - reinspection_id: String (rin_syn_*)                                 │
│  - tenant_id: String (ten_syn_*)                                       │
│  - finding_id: String (fnd_syn_*)                                      │
│  - action_id: String (act_syn_*)                                       │
│  - original_inspector: String (usr_syn_inspector_*)                    │
│  - assigned_reinspector: String (usr_syn_reinspector_*)                │
│  - reinspector_role: String ("INSPECTOR", "THIRD_PARTY_AUDITOR")       │
│  - independence_rule: IndependenceRequirement                          │
│  - state: ReinspectionState (PENDING, ASSIGNED, IN_PROGRESS, ...)      │
│  - state_version: Int64 (Monotonic concurrency version)                │
│  - is_offline: Bool (Strict offline final closure guard)               │
│  - evidence_ids: []String                                              │
│  - recurrence_id: String (rec_syn_*)                                   │
│  - recurrence_count: Int                                               │
│  - history: []ReinspectionHistoryEntry                                 │
└────────────────────────────────────────────────────────────────────────┘
```

### 2.1 Reinspection Lifecycle State Machine

| State | Mutability | Evidentiary Requirement | State Transition Semantics & Boundary |
| :--- | :--- | :--- | :--- |
| **`PENDING_ASSIGNMENT`** | Editable | None | Reinspection order created following action completion. Awaiting verifier assignment. |
| **`ASSIGNED`** | Editable | None | Qualified reinspector designated under applicable independence rules. |
| **`IN_PROGRESS`** | Editable | Accepted evidence | Reinspector actively verifying physical workplace conditions and remediation evidence. |
| **`VERIFIED_SATISFACTORY`** | Read-Only | All evidence accepted | Reinspector confirms corrective controls effectively eliminate the identified hazard. |
| **`REJECTED_DEFICIENT`** | Editable | Defect documentation | Reinspection fails due to incomplete rework or persistent hazard. Action returned for remediation. |
| **`CLOSED`** | Read-Only | Terminal | Protected final closure executed by authorized human supervisor. Double-closure denied. |
| **`REOPENED`** | Editable | Audit findings | Supervisory reopening of closed order following audit or recurrence discovery. |

---

## 3. Prerequisite Gates & Inspector Independence Boundaries

### 3.1 Prerequisite Verification Gates
Before a reinspection order can be marked `VERIFIED_SATISFACTORY` or advance to `CLOSED`, the system validates four mandatory gates:
1. **Action Remediated State:** The associated corrective action must have completed work and be in `IN_REVIEW` or verified remediated status.
2. **Zero Pending or Rejected Evidence:** All linked evidence attachments must possess `ACCEPTED` status under `MOD-EVD`. Any item with `SUBMITTED` (unverified) or `REJECTED` status fails closed with `ErrPendingOrRejectedEvidence`.
3. **Minimum Evidence Quota:** The count of verified accepted evidence items must meet or exceed `RequiredEvidenceCount`.
4. **Physical Verification Note:** Mandatory verification notes detailing the on-site physical inspection must be recorded.

### 3.2 Inspector Independence Boundaries
To prevent conflicts of interest and regulatory non-compliance, reinspection assignment enforces three configurable tiers:
1. **`SAME_INSPECTOR_ALLOWED`:** Routine low-risk tasks where original finding author may verify resolution.
2. **`DIFFERENT_INSPECTOR_REQUIRED`:** High/critical findings requiring an independent second inspector (`AssignedReinspector != OriginalInspector`). Rejection occurs with `ErrInspectorIndependenceViolation` if identical.
3. **`THIRD_PARTY_REQUIRED`:** Statutory or high-consequence findings requiring external credentialed auditors (`reinspector_role == "THIRD_PARTY_AUDITOR"`).

---

## 4. Concurrency, Stale-State, and Offline Final Closure Denial

### 4.1 Optimistic Concurrency Control (`StateVersion`)
Every state change advances `StateVersion` monotonically. Any mutation passing an expected version that does not strictly match the current persisted version fails closed with `ErrConcurrentModification`. This guarantees:
- Reinspection orders cannot be corrupted by stale client submissions.
- Concurrent supervisor and inspector actions do not overwrite each other.
- Last-write-wins is completely prohibited.

### 4.2 Absolute Prohibition of Offline Final Closure
- Field inspectors operating in low-connectivity areas may record draft observations locally.
- However, transitioning an order to `CLOSED` with `is_offline = true` is strictly prohibited and fails closed with `ErrOfflineClosureProhibited`.
- Final closure **must** be submitted to the authoritative backend engine where cryptographic evidence digests and tenant permissions are verified in real time.

---

## 5. Rejection, Reopening, and Recurrence Lineage

### 5.1 Rejection Lifecycle (`REJECTED_DEFICIENT`)
- If reinspection reveals that corrective actions are ineffective or incomplete, the reinspector issues a rejection with mandatory deficiency details.
- The order transitions to `REJECTED_DEFICIENT`.
- If the deficiency indicates recurring chronic failure, `RecurrenceID` is linked and `RecurrenceCount` incremented.
- The associated corrective action is flagged for urgent rework.

### 5.2 Supervisory Reopen Lifecycle (`REOPENED`)
- A finding in `CLOSED` state may be reopened if a subsequent internal audit or customer incident reveals ongoing risk.
- Reopening requires supervisory authority (`isHumanClosureAuthorized`), mandatory non-blank justification reason, and advances versioning.
- Closure metadata (`closed_at`, `closed_by`) is cleared and logged in append-only history.

### 5.3 Bidirectional Recurrence Tracking
- Findings and reinspection orders maintain explicit links to preceding failure occurrences (`recurrence_id`).
- Recurrence counters allow compliance teams to track chronic non-conformances across inspection cycles without losing historical traceability.

---

## 6. Verification and Synthetic Test Boundary

The implementation in `modules/workflow-action/reinspection_lifecycle.go` and `reinspection_lifecycle_test.go` provides 100% synthetic coverage for:
1. Complete happy-path verification and authorized human closure.
2. Fail-closed rejection of unauthorized closure attempts.
3. Fail-closed rejection of autonomous AI closure attempts.
4. Strict rejection of offline final closure attempts.
5. Optimistic concurrency conflict and stale-state rejection.
6. Rejection of deficient remediation and recurrence increments.
7. Supervisory reopening of closed orders with complete audit history.
8. Active authority revocation preventing subsequent mutations.
9. Configurable inspector independence rule enforcement.
10. Prevention of double-closure on terminal records.
