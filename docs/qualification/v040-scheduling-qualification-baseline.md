---
document_id: QLF-V040-SCHED-001
title: v0.4.0 OSHE Inspect Scheduling, Assignment Authority, Recurrence, Time Zone, Reassignment, Cancellation, and Notification Failure Qualification Baseline
document_type: qualification_specification
document_version: 1.0.0
lifecycle_status: DRAFT
status: APPROVED_FOR_LOCAL_DEVELOPMENT
date: "2026-09-05"
author_role: Test and Quality Lead
author_pane: w9:p14
governing_issue: "GitHub Issue #123"
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
credit_boundary: TECHNICAL_QUALIFICATION_ONLY_NO_USER_EVIDENCE_OR_RELEASE_CREDIT
---

# v0.4.0 OSHE Inspect Scheduling, Assignment Authority, Recurrence, Time Zone, Reassignment, Cancellation, and Notification Failure Qualification Baseline

## 1. Executive Summary & Governance Authority

### 1.1 Authority Baseline & Purpose
This qualification specification establishes the authoritative, deterministic **Inspection Scheduling and Assignment Technical Qualification Baseline** for **Milestone v0.4.0 - OSHE Inspect Private Alpha** within `OSHEThai/oshe-platform`. It fulfills the qualification scope and acceptance criteria of **GitHub Issue #123 (`[V040-I012] Qualify Scheduling, Assignment, Eligibility, Time Zone, Recurrence, Reassignment, Cancellation, and Notification Failure`)** under Roadmap Topic `V040-T02` and standing owner foundation decision `HDEC-V040-FOUNDATION-054`.

The primary purpose is to define an integrated, dependency-free, deterministic verification harness covering one-time and recurring inspection generation, default-deny assignment eligibility, Asia/Bangkok time-zone resolution, generation idempotency and deduplication, supervisory reassignment after field download, safe cancellation, append-only responsibility history in `MOD-REC`, decoupled advisory notification mechanics, and visible failure diagnostics.

### 1.2 Non-Substitution Invariant: Technical & Synthetic Scope Only
A foundational invariant governing this baseline is the categorical separation between **synthetic technical evidence** and **empirical real-user evidence**:
- **Synthetic Technical Qualification:** Verifies scheduling logic, recurrence mathematics, eligibility gates, temporal boundaries, cryptographic idempotency keys, and fail-closed state machines using deterministic local synthetic fixtures and simulated clocks.
- **Empirical Real-User Evidence:** Evaluates real human inspector availability, shift handovers, cognitive workload, and organizational adoption.
- **Non-Substitution Invariant:** Under no circumstances may simulated agent runs, automated scheduler test scripts, or synthetic worker payloads substitute for, replace, or claim the status of empirical real-user evidence or actual participant availability. Gate `H040-008` (Real Participant / Private-Alpha UAT Onboarding) remains strictly on **`HOLD`** pending separate owner authorization.

### 1.3 Retained Governance Holds & Explicit Prohibitions
In strict accordance with `HDEC-V040-FOUNDATION-054`, the following governance holds remain in effect and cannot be enacted, bypassed, or scheduled by this specification:
- **`H040-007` (Technical Release Authorization):** HOLD pending completed qualification bundles and owner sign-off.
- **`H040-008` (Real Participant / Private-Alpha / UAT Authorization):** HOLD pending separate owner screening and onboarding decision.
- **`H040-009` (Binding Support & Manual-Fallback Ownership):** HOLD pending formal organizational staffing and handover.
- **`H040-010` (External Environment & Route Activation):** HOLD pending infrastructure security review.
- **`H040-011` (Final v0.4 Outcome & v0.5 Entry Decision):** HOLD reserved exclusively to the Sole Human Owner.

#### Explicit Prohibitions & Anti-Scope Invariants:
- **Zero External Public Routes:** No public internet routes, DNS records, or external web endpoints may be activated.
- **Zero CDN Edge Deployment:** No static assets, templates, or scripts may be deployed to content delivery networks.
- **Zero Production Database Deployment:** Qualification is confined exclusively to local, ephemeral, or isolated in-memory/sqlite instances.
- **Zero Real Customer or Personal Data:** Zero real employee PII, customer records, or production workplace data may be ingested (`H040-003`).
- **Zero Provider, Credential, or Account Mutations:** Zero cloud provider accounts, authentication secrets, or external API keys may be provisioned or mutated.
- **Zero External Notification Delivery:** Zero SMTP/SES email, Twilio SMS, or FCM/APNS push gateways may be provisioned or contacted; all notifications operate strictly via local in-memory/synthetic queues.

---

## 2. Qualification Architecture & Test Domain Matrix

The technical qualification suite systematically verifies the architectural guarantees defined in `ARC-V040-SCHED-001` (Scheduling Baseline), `ARC-V040-ASGN-001` (Assignment Baseline), and `ARC-V040-SCHEDOPS-001` (Scheduling Operations Baseline):

| Domain ID | Qualification Domain | Architectural Reference | Governing Invariant | Key Verification Objective |
| :--- | :--- | :--- | :--- | :--- |
| **QSCHED-01** | Assignment Eligibility & Default-Deny | `ARC-V040-ASGN-001` Sec 2 | `H040-004` | Asserts fail-closed denial for expired membership, wrong scope, revoked role, unsupported device, and duplicate assignment. |
| **QSCHED-02** | Temporal Recurrence & Cadence | `ARC-V040-SCHED-001` Sec 5 | Deterministic Clock | Validates `DAILY`, `WEEKLY`, `MONTHLY`, and `ONCE` recurrence in `Asia/Bangkok` without clock drift. |
| **QSCHED-03** | Due Windows & Grace Period | `ARC-V040-SCHED-001` Sec 6 | Due Progression | Validates temporal anatomy (`window_start`, `due_date`, `grace_until`) and compliance progression (`ON_TIME` $\to$ `IN_GRACE_PERIOD` $\to$ `OVERDUE`). |
| **QSCHED-04** | Cryptographic Idempotency | `ARC-V040-SCHED-001` Sec 8 | Zero Duplicate Dispatch | Asserts SHA-256 deduplication key composition and duplicate execution suppression without side-effects. |
| **QSCHED-05** | Downloaded-Work Custody & Reassignment | `ARC-V040-ASGN-001` Sec 4 | Work Preservation | Verifies supervisory reassignment requires explicit override; prior custody and cached digest preserved. |
| **QSCHED-06** | Safe Cancellation & Pruning | `ARC-V040-SCHED-001` Sec 9 | In-Flight Safety | Asserts schedule cancellation prunes unassigned pending slots while preserving in-flight and completed field work. |
| **QSCHED-07** | Append-Only Responsibility History | `ARC-V040-SCHEDOPS-001` Sec 3 | `MOD-REC` Lineage | Asserts unbroken audit trail across assignment, download, reassignment, and cancellation. |
| **QSCHED-08** | Advisory Decoupling & Local Sinks | `ARC-V040-SCHEDOPS-001` Sec 2 | Fault Isolation | Confirms notification delivery failure never alters or invalidates authoritative business state in `MOD-WFA`. |
| **QSCHED-09** | Visible Diagnostics & Failure Logging | `ARC-V040-SCHEDOPS-001` Sec 4 | Visible Failure | Validates diagnostic emissions (`DIAG_CLOCK_SKEW`, `DIAG_DUPLICATE_DISPATCH`, `DIAG_NOTIFICATION_FAILED`, `DIAG_NOTIFICATION_QUARANTINED`, `DIAG_SCHEDULE_ALTERED`, `DIAG_SCHEDULE_CANCELLED`). |
| **QSCHED-10** | Historical Retention & Execution Pinning | `ARC-V040-SCHED-001` Sec 4 | Audit Repeatability | Verifies active executions remain permanently bound to historical template version/digest; retired schedules preserved. |

---

## 3. Assignment Eligibility & Default-Deny Verification (`QSCHED-01`)

In accordance with `ARC-V040-ASGN-001` and foundation gate `H040-004`, assignment eligibility enforces strict default-deny:
1. **Role Eligibility:** The `Inspector` role is the sole role eligible for inspection execution assignment. `Checklist Author`, `CAPA Owner`, and `Independent Reviewer` are barred.
2. **Device Qualification:** Assignments require verified client device platform support (`Desktop Chrome`, `Desktop Edge`, `Android Chrome Mobile`). Unsupported platforms (`iOS`, `Firefox`, webviews) fail closed.
3. **Fail-Closed Denial Register:** The qualification harness exercises seven explicit denial reasons:
   - `DENIAL_EXPIRED_MEMBERSHIP` (`ErrExpiredTenantMembership`)
   - `DENIAL_WRONG_SCOPE` (`ErrUnauthorizedAssignmentScope`)
   - `DENIAL_REVOKED_ROLE` (`ErrRevokedInspectorRole`)
   - `DENIAL_STALE_SESSION` (`ErrStaleUserSession`)
   - `DENIAL_UNSUPPORTED_DEVICE` (`ErrUnsupportedDevicePlatform`)
   - `DENIAL_DUPLICATE_ASSIGNMENT` (`ErrDuplicateInspectionAssignment`)
   - `DENIAL_DOWNLOADED_AMBIGUITY` (`ErrDownloadedWorkConflict`)

---

## 4. Recurrence Mathematics & Time-Zone Canonicalization (`QSCHED-02`)

In adherence to `ARC-V040-SCHED-001` and `ARC-V040-PROF-001`:
1. **Canonical Indochina Time (`Asia/Bangkok`):** All calendar recurrence rules, shift schedules, and day boundaries evaluate exclusively in `Asia/Bangkok` (UTC+07:00).
2. **Zero Daylight Saving Time (DST) Ambiguity:** `Asia/Bangkok` maintains a fixed $+07:00$ offset, guaranteeing 24-hour day arithmetic without missing or duplicated clock hours.
3. **Deterministic Recurrence Evaluation:**
   - `ONCE`: Dispatches once and transitions schedule to `COMPLETED`.
   - `DAILY`: Advances by `interval` days.
   - `WEEKLY`: Advances to next matching day in `days_of_week`.
   - `MONTHLY`: Advances by `interval` months to `day_of_month`, clamping to month-end (e.g. Feb 28/29).
4. **Anchor-Based Calculation:** Recurrence calculation uses target scheduled dates as anchors, preventing wall-clock drift over operational months.

---

## 5. Due Windows & Grace Period Compliance Progression (`QSCHED-03`)

Each generated execution defines three temporal thresholds:
- `window_start`: Earliest moment inspection can be downloaded.
- `due_date`: Target operational completion deadline.
- `grace_until`: Hard cutoff calculated as $\text{due\_date} + (\text{grace\_period\_hours} \times 3600\text{s})$.

**Compliance Progression Invariant:**
- $\text{CurrentTime} \le \text{due\_date} \implies \mathbf{ON\_TIME}$
- $\text{due\_date} < \text{CurrentTime} \le \text{grace\_until} \implies \mathbf{IN\_GRACE\_PERIOD}$
- $\text{CurrentTime} > \text{grace\_until} \implies \mathbf{OVERDUE}$

---

## 6. Cryptographic Generation Idempotency & Deduplication (`QSCHED-04`)

To prevent duplicate execution generation during distributed retries or offline synchronization:
1. **Idempotency Key Formulation:**
   $$\text{idempotency\_key} = \text{SHA-256}(\text{tenant\_id} \mathbin{\Vert} \text{schedule\_id} \mathbin{\Vert} \text{site\_id} \mathbin{\Vert} \text{area\_id} \mathbin{\Vert} \text{checklist\_version} \mathbin{\Vert} \text{scheduled\_date})$$
2. **Deduplication Invariant:** If the scheduler dispatches twice for the same calendar slot, the existing `idempotency_key` is detected, the existing `execution_id` is returned, and duplicate creation is safely suppressed with a `DIAG_DUPLICATE_DISPATCH` diagnostic log.

---

## 7. Downloaded-Work Custody, Reassignment & Cancellation (`QSCHED-05`, `QSCHED-06`)

In accordance with `ARC-V040-ASGN-001` and `ARC-V040-SCHEDOPS-001`:
1. **Prohibition Against Erasing Prior Responsibility:**
   - When an inspection in `DOWNLOADED` state is reassigned, the prior assignee's custody record is marked `REASSIGNED`, **never deleted or overwritten**.
   - Reassignment of downloaded work strictly requires explicit supervisory override (`override_downloaded_work: true`) and a mandatory justification reason.
2. **Quarantine on Uncoordinated Sync:** If the original inspector attempts to submit work after supervisory reassignment, the conflicting submission is quarantined as `QUARANTINED_CONFLICT` for administrative triage.
3. **Safe Schedule Cancellation:** Cancelling a schedule terminates future recurring dispatches and prunes unassigned pending slots, but leaves in-flight (`IN_PROGRESS`) and completed (`COMPLETED`) executions intact.

---

## 8. Advisory Decoupling & Local Notification Sinks (`QSCHED-08`)

Under the modular contract between `MOD-WFA` and `MOD-EVT`:
1. **Advisory Decoupling Invariant:** Notification delivery success, retry, quarantine, or failure in `MOD-EVT` **never alters or mutates** authoritative business state in `MOD-WFA`.
2. **Fault Isolation:** If the local notification sink fails or exhausts retries, the inspection execution remains validly scheduled, assigned, or overdue according to canonical time.
3. **Synthetic Local Sink Only:** Notifications operate exclusively through local in-memory queues and synthetic event records (`ntf_syn_*`); zero external gateways are invoked.

---

## 9. Diagnostic Event Taxonomy & Visible Failure Surface (`QSCHED-09`)

The qualification suite exercises the emission and administrative capture of six strongly-typed diagnostic events:
- `DIAG_CLOCK_SKEW`: Triggered when `due_date < scheduled_start` or system clock steps backward (`ErrClockSkewDetected`).
- `DIAG_DUPLICATE_DISPATCH`: Emitted upon idempotent deduplication of recurring slots.
- `DIAG_NOTIFICATION_FAILED`: Emitted upon temporary local sink dispatch failure.
- `DIAG_NOTIFICATION_QUARANTINED`: Emitted when notification retries exceed `max_attempts` (default: 3).
- `DIAG_SCHEDULE_ALTERED`: Emitted upon supervisor modification of recurrence or template version.
- `DIAG_SCHEDULE_CANCELLED`: Emitted upon schedule cancellation with pruned slot metrics.

---

## 10. Synthetic Scheduling Qualification Fixtures (`FIX-QUAL-V040-SCHED`)

```yaml
schema_version: "1.0.0"
fixture_id: "fix_syn_scheduling_qualification_v1"

organization:
  tenant_id: "ten_syn_safety_corp"
  site_id: "ste_syn_rayong_01"
  area_id: "ara_syn_boiler_01"

roles:
  eligible_inspector:
    subject_id: "usr_syn_inspector_01"
    role: "Inspector"
    membership_status: "ACTIVE"
    device_platform: "Android Chrome Mobile v128"
  ineligible_author:
    subject_id: "usr_syn_author_01"
    role: "Checklist Author"
    membership_status: "ACTIVE"
  expired_inspector:
    subject_id: "usr_syn_inspector_expired"
    role: "Inspector"
    membership_status: "EXPIRED"

schedules:
  weekly_active:
    schedule_id: "sch_syn_weekly_qual_01"
    status: "ACTIVE"
    time_zone: "Asia/Bangkok"
    trigger_type: "CALENDAR_RECURRING"
    recurrence:
      frequency: "WEEKLY"
      interval: 1
      days_of_week: ["MON"]
    checklist_binding:
      template_id: "chk_syn_pilot_plant_safety_v1"
      version: "1.1.0"
      content_digest: "8f4e2b1c3a7d9e5f0b2a4c6d8e1f3a5b7c9d1e3f5a7b9c1d3e5f7a9b1c3d5e7f"
    due_window:
      window_hours: 12
      grace_period_hours: 12
```

---

## 11. Deterministic Negative Controls & Anti-Regressions

The qualification harness enforces explicit negative test assertions:
- **NSCHED-01 (Expired Membership Denial):** Dispatch or assignment to an expired tenant member fails closed (`ErrExpiredTenantMembership`).
- **NSCHED-02 (Wrong Scope Denial):** Assignment outside inspector's authorized organizational scope fails closed (`ErrUnauthorizedAssignmentScope`).
- **NSCHED-03 (Revoked Role Denial):** Assignment to a user without active `Inspector` role fails closed (`ErrRevokedInspectorRole`).
- **NSCHED-04 (Unsupported Device Denial):** Assignment requiring unsupported client platform (e.g. iOS or Firefox) fails closed (`ErrUnsupportedDevicePlatform`).
- **NSCHED-05 (Duplicate Assignment Denial):** Assigning an execution already in `IN_PROGRESS` without reassignment workflow fails closed (`ErrDuplicateInspectionAssignment`).
- **NSCHED-06 (Reassignment without Override Denial):** Reassigning downloaded work without `override_downloaded_work: true` fails closed (`ErrDownloadedWorkConflict`).
- **NSCHED-07 (Temporal Inconsistency / Clock Skew):** Schedule configuration where `due_date < scheduled_start` fails closed (`ErrClockSkewDetected`).
- **NSCHED-08 (Unpublished Checklist Binding Denial):** Binding a schedule to an unapproved or draft checklist template fails closed (`ErrInvalidChecklistVersionState`).

---

## 12. Governance Boundaries & Operational Non-Claims

In strict conformance with `HDEC-V040-FOUNDATION-054`:
1. **Planning & Qualification Specification Only:** This baseline defines the deterministic verification suite for scheduling. It does not authorize live software release, customer deployment, or residual-risk acceptance.
2. **Synthetic Data Exclusivity (`H040-003`):** 100% synthetic fixtures (`ten_syn_*`, `sch_syn_*`, `ins_syn_*`); zero live customer data or real worker identities.
3. **Default-Deny Authority (`H040-004`):** Assignment and schedule changes require authorized roles.
4. **Server Authority (`H040-005`):** Server sole authority over schedule state and conflict quarantine.
5. **Retained Holds (`H040-007` - `H040-011`):** All release, real-user UAT, external routing, support ownership, and milestone completion gates remain strictly on **`HOLD`**.
