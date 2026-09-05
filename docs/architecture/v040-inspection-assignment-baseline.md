---
document_id: ARC-V040-ASGN-001
title: v0.4.0 OSHE Inspect Eligibility, Assignment, Reassignment, Cancellation, Downloaded-Work Authority, Audit, and Denial-Reason Baseline
document_type: architecture_specification
document_version: 1.0.0
lifecycle_status: DRAFT
status: APPROVED_FOR_LOCAL_DEVELOPMENT
date: "2026-09-05"
author_role: Security Privacy and Product Safety Lead
author_pane: w9:p13
governing_issue: "GitHub Issue #121"
authority_source: HDEC-V040-FOUNDATION-054
governing_decision: HDEC-V040-FOUNDATION-054
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
credit_boundary: ASSIGNMENT_BASELINE_SPECIFICATION_ONLY_NO_EXTERNAL_DELIVERY_OR_LIVE_OPERATIONAL_ENACTMENT
---

# v0.4.0 OSHE Inspect Eligibility, Assignment, Reassignment, Cancellation, Downloaded-Work Authority, Audit, and Denial-Reason Baseline

## 1. Executive Summary & Governance Posture

### 1.1 Authority Baseline & Purpose
This architectural specification establishes the authoritative **Eligibility, Assignment, Reassignment, Cancellation, Downloaded-Work Authority, Audit, and Denial-Reason Baseline** for **Milestone v0.4.0 - OSHE Inspect Private Alpha** within `OSHEThai/oshe-platform`. It fulfills the deliverable requirements of **GitHub Issue #121 (`[V040-I010] Implement Eligibility, Assignment, Reassignment, Cancellation, and Downloaded-Work Authority Rules`)** under Roadmap Topic `V040-T02` and standing owner foundation decision `HDEC-V040-FOUNDATION-054`.

The primary objective is to define a bounded, deterministic, fail-closed eligibility and assignment engine operating across the **Workflow and Action Module (`MOD-WFA`)** and the **Identity and Authorization Module (`MOD-IAM`)**. The engine governs:
- Strict role, membership, geographic/project scope, session, and device compatibility eligibility checks.
- Deterministic assignment dispatch and lifecycle transitions.
- Robust downloaded-work preservation preventing loss of responsibility during reassignments or cancellations.
- Explicit, auditable denial-reason classifications.
- Append-only audit trail logging for all assignment mutations.

### 1.2 Retained Unselected Policies & Non-Claims
In strict compliance with `ASN-V040-I010-ASSIGNMENT-BASELINE-001` and `HDEC-V040-FOUNDATION-054`:
1. **Binding Scoring Policy (`HUMAN_OWNED_UNSELECTED`):** Assignment configurations associate checklists with scoring references, but final binding passing thresholds and critical-fail triggers remain human-owned pending owner decision under Issue #136 (`V040-I025`).
2. **Finding Closure Policy (`HUMAN_OWNED_UNSELECTED`):** Reassignments do not alter finding verification or closure rules; formal finding closure remains human-owned under Issue #134 (`V040-I023`).
3. **Offline Authority Model (`HUMAN_OWNED_UNSELECTED`):** Downloaded-work expiration windows and reconciliation prioritization remain human-owned under Issue #126 (`V040-I015`).
4. **Zero External Notification or Device Management:** No external email, SMS, push notification (FCM/APNS), or mobile device management (MDM) systems are authorized or integrated. All assignment events operate strictly within local memory and synthetic database queues.
5. **Retained Human Gate Holds:** Foundation gates `H040-007` (Technical Release Authorization), `H040-008` (Real Participant / Private-Alpha UAT Onboarding), `H040-009` (Binding Support Ownership), `H040-010` (External Environment/Route Activation), and `H040-011` (Final Milestone Outcome) remain strictly on **`HOLD`**.

---

## 2. Synthetic Identity, Role, Scope, and Device Eligibility Engine

Under `ARC-V040-DOMAIN-001`, assignment authority evaluates under a strict **default-deny** paradigm (`H040-004`). No user may be assigned an inspection execution unless every eligibility gate resolves affirmatively.

### 2.1 Authorized Role Boundary (`H040-001`)
The private alpha vertical slice recognizes four foundation roles:
1. **Checklist Author:** Drafts and publishes checklists. Ineligible for inspection assignment.
2. **Inspector:** Authorized to download assigned inspections, record responses, capture evidence, and log findings. **Sole role eligible for inspection execution assignment.**
3. **CAPA Owner:** Implements corrective actions. Ineligible for inspection assignment.
4. **Independent Reviewer:** Verifies completed inspections and findings. Ineligible for field inspection execution.

### 2.2 Comprehensive Eligibility Invariants
A user subject (`usr_syn_*`) is eligible to receive an assignment for an `InspectionExecution` if and only if all of the following conditions are met:

```
┌────────────────────────────────────────────────────────────────────────┐
│                        ELIGIBILITY EVALUATION                          │
│                                                                        │
│  [1. Active Membership Gate] ──> Must have active tenant membership    │
│                 │                                                      │
│  [2. Role Authorization Gate] ─> Must hold active 'Inspector' role     │
│                 │                                                      │
│  [3. Scope Confinement Gate] ──> Must have explicit site/area scope    │
│                 │                                                      │
│  [4. Session Validity Gate] ───> Must have active, unrevoked session   │
│                 │                                                      │
│  [5. Supported Device Gate] ───> Client device must match H040-002     │
│                 │                                                      │
│  [6. Non-Duplication Gate] ────> Cannot be assigned to same execution  │
│                 │                                                      │
│                 ▼                                                      │
│     [ELIGIBLE: ASSIGNMENT GRANTED] (Emits assignment.created)          │
└────────────────────────────────────────────────────────────────────────┘
```

1. **Active Tenant Membership:** User record must have `membership_status = "ACTIVE"` and `membership_expires_at > Now` within target `tenant_id`.
2. **Role Verification:** User must possess an active `RoleAssignment` granting `role = "Inspector"` in `MOD-IAM`.
3. **Hierarchical Scope Confinement:** The user's authorized scope must encompass the inspection's `company_id`, `site_id`, and `area_id`. Assigning an inspector to a site outside their authorized geographic or organizational domain is prohibited.
4. **Active Session & Credential State:** User credentials must be unexpired, and the caller session token must not be listed in the session revocation registry (`MOD-IAM`).
5. **Supported Device & Platform Profile (`H040-002`):** The target user's active client registration must match the approved private alpha device profile:
   - Supported: Desktop Google Chrome (v128+), Desktop Microsoft Edge (v128+), Mobile Android Chrome (v128+).
   - Prohibited / Unsupported: Apple iOS (Safari/Chrome WebKit), Mozilla Firefox, legacy browsers, container webviews.
6. **Non-Duplication Invariant:** An inspection cannot be assigned to a user who is already the active assignee of that same inspection execution.

---

## 3. Assignment Lifecycle & State Machine

### 3.1 State Machine Specification
Every `Assignment` entity within `MOD-WFA` progresses through the following deterministic states:

```
                   ┌────────────────┐
                   │   UNASSIGNED   │
                   └───────┬────────┘
                           │ (Assign)
                           ▼
                   ┌────────────────┐
         ┌─────────│    ASSIGNED    │─────────┐
         │         └───────┬────────┘         │
         │                 │ (Download)       │
         │                 ▼                  │
         │         ┌────────────────┐         │
(Cancel) │   ┌─────│   DOWNLOADED   │─────┐   │ (Cancel)
         │   │     └───────┬────────┘     │   │
         │   │             │ (Start)      │   │
         │   │             ▼              │   │
         │   │     ┌────────────────┐     │   │
         │   │     │  IN_PROGRESS   │     │   │
         │   │     └────────────────┘     │   │
         │   │                            │   │
         ▼   ▼ (Reassign w/ Override)     ▼   ▼
  ┌─────────────┐                  ┌─────────────┐
  │ REASSIGNED  │                  │  CANCELLED  │
  └─────────────┘                  └─────────────┘
```

### 3.2 State Definitions
- **`UNASSIGNED`:** Inspection execution exists from schedule dispatch but has no designated inspector.
- **`ASSIGNED`:** Inspection is allocated to an eligible inspector. Package not yet retrieved by client device.
- **`DOWNLOADED`:** Inspector client device has synchronized and cached the inspection package for local execution (`is_downloaded = true`).
- **`IN_PROGRESS`:** Inspector has recorded at least one response item in the checklist.
- **`REASSIGNED`:** Responsibility transferred to a successor inspector. Prior assignment marked `REASSIGNED` with full lineage preserved.
- **`CANCELLED`:** Inspection allocation terminated administratively.

---

## 4. Downloaded-Work Authority & Preservation Invariants (`H040-005`)

When an inspector downloads an inspection package to their local client device (`IndexedDB`), the system enters the **`DOWNLOADED`** operational state. Handling reassignment or cancellation in this state presents critical safety and audit risks regarding orphaned or conflicting offline work.

### 4.1 Prohibition Against Erasing Prior Responsibility
1. **Immutable Historical Provenance:** Under no circumstances may a reassignment or cancellation delete, purge, or overwrite the record of an inspector who previously downloaded or initiated work.
2. **Prior Responsibility Preservation:** The `Assignment` record retains `assignee_subject`, `assigned_at`, `downloaded_at`, `download_digest`, and `client_device_fingerprint`.

### 4.2 Reassignment of Downloaded Work Protocol
If an administrative supervisor attempts to reassign an inspection currently in `DOWNLOADED` or `IN_PROGRESS` state:
1. **Default-Deny Protection (`DENIAL_DOWNLOADED_AMBIGUITY`):** Standard reassignment requests fail closed with `ErrDownloadedWorkConflict`.
2. **Supervisory Override Prerequisite:** Reassignment of downloaded work requires an explicit supervisory override flag (`override_downloaded_work = true`) and a mandatory justification reason (`reassignment_reason`).
3. **Lease Revocation & Sequence Advance:**
   - The server increments the execution's `state_version`.
   - The prior assignment is marked `REASSIGNED` with `revocation_reason = "SUPERVISORY_REASSIGNMENT"`.
   - The prior download token is flagged `REVOKED_SUPERSEDED`.
4. **Offline Reconnect & Conflict Quarantine (`H040-005`):** If the original inspector later comes online and submits cached field work:
   - Server detects that the submission's `base_version` does not match the active state.
   - The original inspector's submission is **not discarded**; it is placed into **`QUARANTINED`** holding for manual supervisory review and synthesis, preventing data loss.

### 4.3 Cancellation of Downloaded Work Protocol
If an inspection in `DOWNLOADED` state is cancelled:
1. Execution status transitions to `CANCELLED`.
2. Any subsequent offline synchronization attempt by the previous assignee is rejected with `ErrExecutionCancelledWhileDownloaded`.
3. An audit record is written preserving the client's cached state digest for retrospective investigation.

---

## 5. Explicit Denial Reasons Register (Fail-Closed Matrix)

When an assignment, reassignment, or cancellation request fails eligibility validation, the engine returns a strongly-typed denial code and records an audit event:

| Denial Code | Error Identifier | Validation Failure Mechanism | Required Remediation |
| :--- | :--- | :--- | :--- |
| **`DENIAL_EXPIRED_MEMBERSHIP`** | `ErrExpiredTenantMembership` | User's organizational membership has expired or is suspended. | Renew user organization contract or tenant access grant. |
| **`DENIAL_WRONG_SCOPE`** | `ErrUnauthorizedAssignmentScope` | User's role grant does not cover the target project, site, or area. | Grant site-specific authorization in `MOD-ORG` / `MOD-IAM`. |
| **`DENIAL_REVOKED_ROLE`** | `ErrRevokedInspectorRole` | User lacks active `Inspector` role (e.g. role revoked, or role is Author/Reviewer/CAPA). | Assign `Inspector` role to user subject in `MOD-IAM`. |
| **`DENIAL_STALE_SESSION`** | `ErrStaleUserSession` | Caller or target session token has expired, been revoked, or failed signature check. | Target user must re-authenticate and establish a fresh session. |
| **`DENIAL_UNSUPPORTED_DEVICE`** | `ErrUnsupportedDevicePlatform` | User's active device registration is an unsupported browser or OS (e.g. iOS Safari, Firefox). | Access application via supported browser (Desktop Chrome/Edge or Android Chrome). |
| **`DENIAL_DUPLICATE_ASSIGNMENT`** | `ErrDuplicateInspectionAssignment` | Target user is already the current active assignee of this execution. | Select a different inspector or retain existing assignment. |
| **`DENIAL_DOWNLOADED_AMBIGUITY`** | `ErrDownloadedWorkConflict` | Attempted reassignment or cancellation of a `DOWNLOADED` inspection without supervisory override. | Provide `override_downloaded_work: true` and explicit supervisory justification. |

---

## 6. Reassignment, Cancellation & Audit Event Catalog (`MOD-REC`)

Every mutation or denial in the assignment lifecycle emits an append-only, tamper-evident audit record to `MOD-REC`.

### 6.1 Audit Event Enumeration
1. **`assignment.created`:** Emitted when an inspection execution is successfully assigned to an eligible inspector.
2. **`assignment.downloaded`:** Emitted when the client device downloads and acknowledges local caching of the inspection package.
3. **`assignment.reassigned`:** Emitted when an active or downloaded inspection is transferred to a successor inspector.
4. **`assignment.cancelled`:** Emitted when an assignment is terminated administratively.
5. **`assignment.denied`:** Emitted whenever an assignment request is rejected by the eligibility engine.

### 6.2 Audit Payload Specification
Audit records contain complete cryptographic and actor attribution:
```yaml
audit_record:
  record_id: "rec_syn_asg_01"
  tenant_id: "ten_syn_01"
  event_name: "assignment.reassigned"
  execution_id: "ins_syn_weekly_rayong_01"
  timestamp: "2026-09-10T04:30:00.000Z"
  actor_subject: "usr_syn_supervisor_01"
  actor_role: "Independent Reviewer"
  previous_assignee: "usr_syn_inspector_01"
  new_assignee: "usr_syn_inspector_02"
  download_state_preserved:
    was_downloaded: true
    downloaded_at: "2026-09-10T02:15:00.000Z"
    download_digest: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
  justification_reason: "Primary inspector reassigned to emergency outage inspection."
  denial_code: null
```

---

## 7. Synthetic Multi-Assignment Fixture Matrix

The following synthetic YAML fixture illustrates standard, reassigned, cancelled, and denied assignments for Milestone v0.4.0 Private Alpha:

```yaml
schema_version: "1.0.0"
fixture_id: "fix_syn_inspection_assignments_v1"

assignments:
  # Scenario 1: Standard Eligible Assignment & Clean Download
  - assignment_id: "asg_syn_clean_01"
    tenant_id: "ten_syn_01"
    execution_id: "ins_syn_clean_01"
    schedule_id: "sch_syn_weekly_plant_01"
    status: "DOWNLOADED"
    assignee_subject: "usr_syn_inspector_01"
    assignee_role: "Inspector"
    assigned_by: "usr_syn_supervisor_01"
    assigned_at: "2026-09-10T01:00:00Z"
    download_state:
      is_downloaded: true
      downloaded_at: "2026-09-10T01:15:00Z"
      client_platform: "Android Chrome Mobile v128"
      download_digest: "a1b2c3d4e5f60718293a4b5c6d7e8f90123456789abcdef0123456789abcdef0"
    scope_verification:
      site_id: "ste_syn_rayong_01"
      area_id: "ara_syn_production_hall_01"
      scope_matched: true

  # Scenario 2: Downloaded Work Reassigned With Supervisory Override
  - assignment_id: "asg_syn_reassigned_02"
    tenant_id: "ten_syn_01"
    execution_id: "ins_syn_reassigned_02"
    schedule_id: "sch_syn_weekly_plant_01"
    status: "REASSIGNED"
    assignee_subject: "usr_syn_inspector_01"
    successor_assignment_id: "asg_syn_successor_02"
    download_state:
      is_downloaded: true
      downloaded_at: "2026-09-10T02:00:00Z"
      download_digest: "b2c3d4e5f60718293a4b5c6d7e8f90123456789abcdef0123456789abcdef01"
    reassignment_record:
      reassigned_by: "usr_syn_supervisor_01"
      reassigned_at: "2026-09-10T04:00:00Z"
      override_downloaded_work: true
      reassignment_reason: "Original inspector taken ill; reassigning to alternate on-site inspector."
      successor_subject: "usr_syn_inspector_02"

  # Scenario 3: Cancelled Assignment Preserving Download History
  - assignment_id: "asg_syn_cancelled_03"
    tenant_id: "ten_syn_01"
    execution_id: "ins_syn_cancelled_03"
    schedule_id: "sch_syn_adhoc_maint_01"
    status: "CANCELLED"
    assignee_subject: "usr_syn_inspector_02"
    download_state:
      is_downloaded: true
      downloaded_at: "2026-09-10T03:00:00Z"
      download_digest: "c3d4e5f60718293a4b5c6d7e8f90123456789abcdef0123456789abcdef012"
    cancellation_record:
      cancelled_by: "usr_syn_supervisor_01"
      cancelled_at: "2026-09-10T05:00:00Z"
      cancellation_reason: "Facility unit taken offline for scheduled boiler overhaul."

denial_scenarios:
  - scenario_id: "den_syn_01_expired"
    target_user: "usr_syn_contractor_expired"
    denial_code: "DENIAL_EXPIRED_MEMBERSHIP"
    expected_error: "ErrExpiredTenantMembership"

  - scenario_id: "den_syn_02_wrong_scope"
    target_user: "usr_syn_inspector_bangkok_only"
    target_site: "ste_syn_rayong_01"
    denial_code: "DENIAL_WRONG_SCOPE"
    expected_error: "ErrUnauthorizedAssignmentScope"

  - scenario_id: "den_syn_03_revoked_role"
    target_user: "usr_syn_author_01"
    target_role: "Checklist Author"
    denial_code: "DENIAL_REVOKED_ROLE"
    expected_error: "ErrRevokedInspectorRole"

  - scenario_id: "den_syn_04_unsupported_device"
    target_user: "usr_syn_inspector_ios"
    client_platform: "Apple iOS Safari v17"
    denial_code: "DENIAL_UNSUPPORTED_DEVICE"
    expected_error: "ErrUnsupportedDevicePlatform"

  - scenario_id: "den_syn_05_downloaded_ambiguity"
    target_execution: "ins_syn_reassigned_02"
    override_downloaded_work: false
    denial_code: "DENIAL_DOWNLOADED_AMBIGUITY"
    expected_error: "ErrDownloadedWorkConflict"
```

---

## 8. Governance Boundaries, Prohibitions & Non-Claims

In strict compliance with `HDEC-V040-FOUNDATION-054` and `ASN-V040-I010-ASSIGNMENT-BASELINE-001`:

1. **100% Synthetic Data Policy (`H040-003`):** All users (`usr_syn_*`), tenants (`ten_syn_*`), sites (`ste_syn_*`), and assignments (`asg_syn_*`) are synthetic entities. Zero real employee, contractor, or workforce data is referenced or stored.
2. **Default-Deny Authority Invariant (`H040-004`):** Assignment modifications, reassignments, and cancellations require authenticated supervisor roles (`Independent Reviewer` or supervisory lead). Inspectors cannot reassign or cancel their own assigned inspections.
3. **No External Route or Notification Activation (`H040-007` & `H040-010` HOLD):** Public assignment endpoints, external email/SMS dispatch, and mobile push notifications remain strictly on HOLD.
4. **No Real Participant Onboarding (`H040-008` HOLD):** Zero real field inspectors, contractors, or pilot users are onboarded or granted system credentials.
5. **Specification-Only Credit:** Delivery of this baseline confers documentation and architectural specification credit only; zero operational release, deployment, or residual-risk acceptance is claimed.
