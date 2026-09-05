---
document_id: QLF-V040-CAPA-001
title: v0.4.0 OSHE Inspect Findings, Severity, Ownership, Evidence, Escalation, Reinspection, Reopen, Closure, Concurrency, and Audit Qualification Baseline
document_type: qualification_specification
document_version: 1.0.0
lifecycle_status: DRAFT
status: APPROVED_FOR_LOCAL_DEVELOPMENT
date: "2026-09-06"
author_role: Architecture and Data Lead
author_pane: w9:p22
governing_issue: "GitHub Issue #135"
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
  independent_review_policy: HUMAN_OWNED_UNSELECTED
  reinspection_criteria_policy: HUMAN_OWNED_UNSELECTED
credit_boundary: TECHNICAL_QUALIFICATION_ONLY_NO_REAL_OWNER_OR_RELEASE_CREDIT
---

# v0.4.0 OSHE Inspect Findings, Severity, Ownership, Evidence, Escalation, Reinspection, Reopen, Closure, Concurrency, and Audit Qualification Baseline

## 1. Executive Summary & Governance Authority

### 1.1 Authority Baseline & Purpose
This qualification specification establishes the authoritative, deterministic **Findings, Severity Classification, Action Ownership, Evidence Verification, Escalation, Reinspection, Reopen, Closure Authority, Concurrency Control, and Audit History Qualification Baseline** for **Milestone v0.4.0 - OSHE Inspect Private Alpha** within `OSHEThai/oshe-platform`. It fulfills the qualification scope and acceptance criteria of **GitHub Issue #135 (`[V040-I024] Qualify Findings, Severity, Ownership, Evidence, Escalation, Reinspection, Reopen, Closure, Concurrency, and Audit`)** under Roadmap Topic `V040-T05` and standing owner foundation decision `HDEC-V040-FOUNDATION-054`.

The primary objective is to define an integrated, dependency-free technical verification harness qualifying the end-to-end corrective action (CAPA) lifecycle established across:
- **`ARC-V040-FND-001` (Issue #132):** Finding foundation, severity classification, critical flags, and recurrence reference.
- **`ARC-V040-ACTGOV-001` (Issue #133):** Corrective action ownership, assignment, extensions, escalations, and evidence governance.
- **`ARC-V040-REINSP-001` (Issue #134):** Reinspection assignment, independent review, rejection, reopening, stale-state protection, and closure.

### 1.2 Non-Substitution Invariant: Technical & Synthetic Scope Only
In strict compliance with `ASN-V040-I024-CAPA-QUALIFICATION-001` and `HDEC-V040-FOUNDATION-054`:
- **Synthetic Technical Qualification Only:** This baseline evaluates deterministic state machine transitions, optimistic concurrency locks, cryptographic hash chains, role boundary enforcements, and audit logging using local synthetic fixtures (`fnd_*`, `act_*`, `evd_*`, `usr_*`) and mock clocks.
- **Non-Substitution Invariant:** Automated test harnesses and synthetic simulations **cannot substitute for, replace, or claim the status of empirical real-world owner accountability, supervisor judgment, or real CAPA management**. Gate `H040-008` (Real Participant / Private-Alpha UAT Authorization) remains strictly on **`HOLD`** pending separate owner screening and authorization.

### 1.3 Retained Unselected Policies & Non-Claims
1. **Finding Closure Policy (`HUMAN_OWNED_UNSELECTED`):** Regulatory sign-off criteria, statutory verification standards, and residual safety risk acceptance remain strictly human-owned under Issue #134 (`V040-I023`).
2. **Binding Scoring Policy (`HUMAN_OWNED_UNSELECTED`):** Action resolution scoring models remain provisional under Issue #136 (`V040-I025`).
3. **Offline Authority Model (`HUMAN_OWNED_UNSELECTED`):** Offline lease parameters remain human-owned under Issue #126 (`V040-I015`).
4. **Binding Extension & Escalation Policies (`HUMAN_OWNED_UNSELECTED`):** Corporate SLA limits and escalation paths remain unselected and human-owned.
5. **No Real Operational Data (`H040-003`):** 100% synthetic finding test fixtures only; zero real plant hazard logs or workforce observations.
6. **Retained Human Gate Holds:** Foundation gates `H040-007` (Technical Release Authorization), `H040-008` (Real Participant UAT), `H040-009` (Binding Support Ownership), `H040-010` (External Environment Activation), and `H040-011` (Final Milestone Outcome) remain strictly on **`HOLD`**.

---

## 2. Dimension 1: Finding Creation, Severity Classification & Recurrence Lineage

In accordance with `ARC-V040-FND-001`:

1. **Deterministic Finding Creation:** Findings (`fnd_syn_*`) are created from concerning or failed checklist question responses using registered, version-pinned rules in `MOD-WFA`.
2. **Severity Catalog & Critical Flags:**
   - Severity is constrained to four discrete categories: `LOW`, `MEDIUM`, `HIGH`, `CRITICAL`.
   - Findings involving imminent danger mandate the boolean flag `critical_flag: true`.
3. **Mandatory Immediate Controls:**
   - For all `CRITICAL` findings and `HIGH` severity hazards, an `immediate_control_note` is mandatory before record commitment.
   - Omission of immediate controls fails closed with `ErrImmediateControlRequired`.
4. **Bidirectional Recurrence Lineage:**
   - Recurring non-conformances explicitly reference their predecessor hazard via `recurrence_id`.
   - Tracks recurrence count and historical hazard lineage across inspection cycles.

---

## 3. Dimension 2: Action Ownership Assignment & Missing Owner Rejection

Under `ARC-V040-ACTGOV-001`:

1. **Mandatory Named Responsible Owner:** Every corrective action (`act_syn_*`) must be assigned to an active, authorized user subject (`assigned_owner_id`).
2. **Missing Owner Rejection (Fail-Closed):**
   - Action creation or assignment attempts without an active, validated owner fail closed with `ErrMissingActionOwner`.
3. **Unbroken Custody & Reassignment Audit:**
   - Reassignment transitions capture `from_owner`, `to_owner`, `reassigned_by`, and `reassignment_reason` in append-only audit journals (`MOD-REC`).

---

## 4. Dimension 3: Due Dates, Invalid Extension Rejection & Escalation

In accordance with `ARC-V040-ACTGOV-001`:

1. **Valid Due Date Enforcement:** Target remediation dates must be strictly future-dated relative to action creation time (`due_date > created_at`).
2. **Invalid Extension Rejection (Fail-Closed):**
   - Extension requests must carry explicit justification notes and be authorized by an authorized supervisory role.
   - Retroactive extension requests (`new_due_date <= current_due_date`) fail closed with `ErrInvalidExtensionRequest`.
   - Extensions exceeding maximum allowed extension windows fail closed with `ErrExtensionPastMaximumWindow`.
3. **Automatic Escalation on Overdue Expiration:**
   - Actions passing their active due date without approved extension or evidence submission automatically transition to `ESCALATED` status (`ErrActionOverdueEscalation`).

---

## 5. Dimension 4: Remediation Evidence Submission & Independent Review Rejection

Under `ARC-V040-ACTGOV-001` and `ARC-V040-REINSP-001`:

1. **Mandatory Evidentiary Proof:**
   - Action owners advancing status to `EVIDENCE_SUBMITTED` must provide valid evidence references (`evd_syn_*`) satisfying cryptographic checksums (`ARC-V040-EVD-003`).
   - Submissions without attachments fail closed with `ErrRemediationEvidenceMissing`.
2. **Segregation of Duties (SOD) & Self-Review Bar:**
   - Action owners are **strictly prohibited** from reviewing, verifying, or approving their own corrective actions (`ErrSelfReviewForbidden`).
   - Review authority is restricted to an independent `Independent Reviewer` or authorized supervisor.
3. **Deficient Remediation Rejection:**
   - If submitted evidence is judged inadequate, the reviewer rejects the submission with mandatory feedback (`ErrRemediationEvidenceRejected`).
   - Action transitions to `REWORK_REQUIRED` / `IN_PROGRESS`, locking premature closure.

---

## 6. Dimension 5: Reinspection Assignment & Stale-State Protection

In accordance with `ARC-V040-REINSP-001`:

1. **Configurable Reinspection Independence:**
   - Reinspections enforce tenant-configured independence rules (`SAME_INSPECTOR_ALLOWED`, `DIFFERENT_INSPECTOR_REQUIRED`, `THIRD_PARTY_REQUIRED`).
2. **Optimistic Concurrency & Stale-State Defense (`H040-005`):**
   - Every finding, action, and reinspection record maintains a monotonic `StateVersion`.
   - Updates submitting a stale or mismatched version token fail closed with `ErrStaleStateConflict`.
   - **Zero Last-Write-Wins (LWW):** Conflicting concurrent edits can never overwrite server state.

---

## 7. Dimension 6: Unauthorized, Stale, and Offline Closure Denial

Under `H040-004`, `H040-005`, and `ARC-V040-REINSP-001`:

1. **Protected State Transition:** Finding and action closure is a protected, high-assurance state transition.
2. **Absolute Prohibition of Offline Final Closure:**
   - While preliminary field notes may be drafted offline, **final closure attempted in an offline state strictly fails closed with `ErrOfflineClosureForbidden`**.
   - Closure requires live server-authoritative validation, unexpired credentials, and verified evidence integrity.
3. **Unauthorized Role Closure Rejection:**
   - Closure attempted by unauthorized roles (e.g. `Inspector`, `Checklist Author`, or unassigned user) fails closed with `ErrUnauthorizedClosure`.
4. **Stale State Closure Rejection:**
   - Attempting to close a finding when underlying action status has changed concurrently fails closed with `ErrStaleVersionAtClosure`.

---

## 8. Dimension 7: Reopening Workflow & Bidirectional Hazard Tracking

Under `ARC-V040-REINSP-001`:

1. **Supervisory Reopening Protocol:**
   - If a closed finding displays recurring defects, un-remediated hazards, or flawed repairs upon post-closure audit, authorized supervisors may trigger `REOPEN`.
   - Reopening mandates an explicit justification reason permanently logged in audit journals.
2. **Automatic Re-engagement & Lineage Tracking:**
   - Reopened findings automatically stage a new child reinspection task (`StatusReopenedPendingReinspection`).
   - Establishes bidirectional traceability linking initial non-conformance, failed closure, and re-opened corrective action.

---

## 9. Dimension 8: Append-Only Audit History & Lineage Reconstruction (`MOD-REC`)

In accordance with `ARC-V040-DOMAIN-001`:

1. **Immutable Audit Entries:** Every finding creation, severity assignment, action assignment, extension approval, evidence submission, reinspection outcome, and closure commits an immutable record in `MOD-REC`.
2. **Audit Envelope Standard:**
   - `record_id`: Strongly-typed UUID (`rec_[0-9a-f]{16}`).
   - `entity_type` & `entity_id`: Target finding/action identifier.
   - `actor_subject` & `actor_role`: Responsible user and role.
   - `from_state` & `to_state`: Exact lifecycle transition endpoints.
   - `audit_digest`: Composite SHA-256 seal ensuring audit log tamper-evidence.
3. **Cryptographic Lineage Reconstruction:** Validates unbroken chronological chain from finding inception to final resolution.

---

## 10. Synthetic Qualification Scenarios Fixture Matrix

The following synthetic YAML fixture specifies the complete qualification scenario catalog:

```yaml
schema_version: "1.0.0"
fixture_id: "fix_syn_capa_qualification_v1"

scenarios:
  - id: "QLF-CAPA-01"
    name: "Missing Action Owner Rejection"
    action_item:
      finding_id: "fnd_syn_scaffold_01"
      assigned_owner_id: ""
    expected_result: "REJECT_ERR_MISSING_ACTION_OWNER"

  - id: "QLF-CAPA-02"
    name: "Invalid Extension Request (Past Window / Retroactive)"
    current_due_date: "2026-09-10T12:00:00Z"
    requested_due_date: "2026-09-08T12:00:00Z"
    expected_result: "REJECT_ERR_INVALID_EXTENSION_REQUEST"

  - id: "QLF-CAPA-03"
    name: "Remediation Evidence Missing"
    action_status: "ACTION_ASSIGNED"
    submitted_evidence_refs: []
    expected_result: "REJECT_ERR_REMEDIATION_EVIDENCE_MISSING"

  - id: "QLF-CAPA-04"
    name: "Self-Review Prohibition (SOD Enforcement)"
    action_owner: "usr_syn_worker_01"
    reviewer_subject: "usr_syn_worker_01"
    action: "APPROVE_REMEDIATION"
    expected_result: "REJECT_ERR_SELF_REVIEW_FORBIDDEN"

  - id: "QLF-CAPA-05"
    name: "Offline Final Closure Denial"
    finding_id: "fnd_syn_guardrail_02"
    client_state: "OFFLINE_DRAFT"
    attempted_transition: "VERIFIED_CLOSED"
    expected_result: "REJECT_ERR_OFFLINE_CLOSURE_FORBIDDEN"

  - id: "QLF-CAPA-06"
    name: "Optimistic Concurrency Stale-State Conflict"
    server_version: 3
    client_base_version: 1
    attempted_action: "UPDATE_REMEDIATION_PLAN"
    expected_result: "REJECT_ERR_STALE_STATE_CONFLICT"

  - id: "QLF-CAPA-07"
    name: "Supervisory Reopening Lineage"
    finding_id: "fnd_syn_harness_03"
    current_status: "VERIFIED_CLOSED"
    reopen_reason: "Latent fraying discovered in secondary lanyard"
    expected_result: "PASS_REOPENED_LINEAGE_PRESERVED"

  - id: "QLF-CAPA-08"
    name: "Immediate Control Note Omission on Critical Hazard"
    severity: "CRITICAL"
    critical_flag: true
    immediate_control_note: ""
    expected_result: "REJECT_ERR_IMMEDIATE_CONTROL_REQUIRED"
```

---

## 11. Governance Boundaries & Non-Claims

In strict adherence to `HDEC-V040-FOUNDATION-054`:

1. **Synthetic Technical Evidence Only:** All finding entities, corrective actions, evidence references, and inspection sessions operate strictly as fictionalized alpha fixtures (`fnd_*`, `act_*`, `evd_*`, `usr_*`). Zero real workforce data or actual plant hazard logs are processed.
2. **Zero Real-Owner Usability Claim:** This technical qualification does **NOT** constitute or substitute for empirical real-user owner evaluation or organizational adoption. Gate `H040-008` remains on strict **`HOLD`**.
3. **Zero Deployment or Release Claim:** Gates `H040-007` through `H040-011` remain on **HOLD**. Zero live database persistence, public routes, or production releases are claimed or authorized.
