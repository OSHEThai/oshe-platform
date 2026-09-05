---
document_id: ARC-V040-SCHEDOPS-001
title: v0.4.0 OSHE Inspect Scheduling Operations, Responsibility History, Schedule Diagnostics, Notification Requests, Local Sink, and Visible Failure Baseline
document_type: architecture_specification
document_version: 1.0.0
lifecycle_status: DRAFT
status: APPROVED_FOR_LOCAL_DEVELOPMENT
date: "2026-09-05"
author_role: Engineering Lead
author_pane: w9:p23
governing_issue: "GitHub Issue #122"
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
credit_boundary: SCHEDULING_OPERATIONS_SPECIFICATION_ONLY_NO_EXTERNAL_DELIVERY_OR_LIVE_OPERATIONAL_ENACTMENT
---

# v0.4.0 OSHE Inspect Scheduling Operations, Responsibility History, Schedule Diagnostics, Notification Requests, Local Sink, and Visible Failure Baseline

## 1. Executive Summary & Governance Posture

### 1.1 Authority Baseline & Purpose
This architectural specification establishes the authoritative **Scheduling Operations, Responsibility History, Schedule Diagnostics, Notification Requests, Local Sink, and Visible Failure Baseline** for **Milestone v0.4.0 - OSHE Inspect Private Alpha** within `OSHEThai/oshe-platform`. It fulfills the deliverable requirements of **GitHub Issue #122 (`[V040-I011] Inspection Scheduling Operations, Diagnostics, and Synthetic Notification Dispatch Baseline`)** under Roadmap Topic `V040-T02` and standing owner foundation decision `HDEC-V040-FOUNDATION-054`.

The primary objective is to define the operational, diagnostic, and notification mechanics supporting inspection scheduling (`ARC-V040-SCHED-001`) and assignment authority (`ARC-V040-ASGN-001`), providing:
- Complete, immutable responsibility history preserving inspector attribution across assignments, downloads, reassignments, and cancellations.
- Decoupled, advisory notification mechanics operating strictly via deterministic local sinks (`MOD-EVT`).
- Visible failure diagnostics and operational logging for delayed reminders, duplicate dispatches, schedule alterations, cancellations, and clock skew.
- Deterministic clock abstraction ensuring reproducible simulation and test verification.

### 1.2 Retained Unselected Policies & Non-Claims
In strict compliance with `ASN-V040-I011-SCHEDULING-OPERATIONS-001` and `HDEC-V040-FOUNDATION-054`:
1. **Binding Scoring Policy (`HUMAN_OWNED_UNSELECTED`):** Inspection schedules associate scoring references, but passing thresholds, weights, and ratings remain human-owned under Issue #136 (`V040-I025`).
2. **Finding Closure Policy (`HUMAN_OWNED_UNSELECTED`):** Operational scheduling of reinspections adheres to workflow linking, but finding closure authority remains human-owned under Issue #134 (`V040-I023`).
3. **Offline Authority Model (`HUMAN_OWNED_UNSELECTED`):** Client-side download cache expiration and offline lease limits remain human-owned under Issue #126 (`V040-I015`).
4. **Preservation of Gate `H040-010` (External Activation HOLD):** Zero external email gateways (SMTP/SES), SMS aggregators (Twilio), mobile push notification services (FCM/APNS), external webhook endpoints, or cloud providers are authorized, integrated, or activated. All notification requests operate strictly within local memory sinks and synthetic database queues.
5. **Retained Human Gate Holds:** Foundation gates `H040-007` (Technical Release Authorization), `H040-008` (Real Participant / Private-Alpha UAT Onboarding), `H040-009` (Binding Support Ownership), `H040-010` (External Environment/Route Activation), and `H040-011` (Final Milestone Outcome) remain strictly on **`HOLD`**.

---

## 2. Module Boundaries & Advisory Decoupling (`MOD-WFA` vs `MOD-EVT`)

Under `ARC-V040-DOMAIN-001` and `modules/events-outbox-jobs/README.md`, strict modular decoupling is enforced between business truth and notification delivery:

```
┌─────────────────────────────────────────┐          Emits Events          ┌─────────────────────────────────────────┐
│     Workflow & Action Module (MOD-WFA)   │ ─────────────────────────────> │  Events, Outbox & Jobs Module (MOD-EVT) │
│                                         │                                │                                         │
│  - Authoritative Business Truth         │                                │  - Notification Requests                │
│  - InspectionSchedule, Assignment       │                                │  - Local Sink Dispatch Queue            │
│  - Due Windows, Cancellation Status     │ <───────────────────────────── │  - Retry, Quarantine, Replay            │
│  - Downloader Responsibility Lineage    │         ZERO DIRECT            │  - Simulated Diagnostic Failures        │
│                                         │       BUSINESS MUTATION        │                                         │
└─────────────────────────────────────────┘                                └─────────────────────────────────────────┘
```

### 2.1 The Advisory Decoupling Invariant
1. **Zero Direct Business State Mutation:** Notification delivery success, retry, quarantine, failure, or replay within `MOD-EVT` **cannot directly alter or mutate** any authoritative business state in `MOD-WFA`.
2. **Independent Compliance Evaluation:**
   - If a local notification sink fails or is delayed, the inspection execution remains validly scheduled, assigned, or due according to canonical calendar time.
   - Delivery failure does **not** cancel, postpone, or invalidate an assignment.
   - Successful delivery does **not** automatically mark an inspection as started, in-progress, or completed.
3. **Fault Isolation:** A complete breakdown or crash of the notification subsystem leaves field inspection execution and offline synchronization fully functional.

---

## 3. Append-Only Responsibility History & Audit Trail

In accordance with `ARC-V040-ASGN-001` and foundation gate `H040-004`, responsibility for an inspection execution is tracked through an unbroken, append-only historical trail:

### 3.1 Responsibility State Transition Trail
Whenever an inspection execution is assigned, downloaded, reassigned, or cancelled, an immutable record is appended to the responsibility journal in `MOD-REC`:

| Operational Event | Captured Responsibility Attributes | Preservation Guarantee |
| :--- | :--- | :--- |
| **Initial Assignment** | `assignment_id`, `execution_id`, `assignee_subject`, `assignee_role`, `assigned_by`, `assigned_at`. | Establishes original assigned inspector. |
| **Field Download** | `downloaded_at`, `client_platform`, `download_digest`, `client_device_fingerprint`. | Establishes offline custody and active field responsibility. |
| **Supervisory Reassignment** | `reassigned_by`, `reassigned_at`, `override_downloaded_work: true`, `reassignment_reason`, `prior_assignee`, `successor_assignee`. | Prior assignee is marked `REASSIGNED`, **never erased or overwritten**. Prior cached work digest is preserved. |
| **Administrative Cancellation** | `cancelled_by`, `cancelled_at`, `cancellation_reason`, `was_downloaded`, `cached_digest_at_cancel`. | Preserves download history even when inspection is terminated before submission. |

### 3.2 Audit Log Tamper-Evidence
Every responsibility transition generates an audit entry containing:
- `event_id`: Unique identifier (`rec_syn_*`).
- `tenant_id`: Authoritative tenant scope (`ten_syn_*`).
- `execution_id`: Target inspection (`ins_syn_*`).
- `actor_subject` & `actor_role`: Identity of the user performing the transition.
- `payload_digest`: Cryptographic SHA-256 hash of the state change payload.

---

## 4. Schedule Operational Diagnostics & Visible Failure Model

Operational abnormalities in scheduling and dispatch must be visibly captured, diagnosed, and surfaced to administrative roles without failing silently or disrupting field workflows.

### 4.1 Diagnostic Event Taxonomy
The scheduler emits strongly-typed diagnostic events to the operations log:

```
┌────────────────────────────────────────────────────────────────────────┐
│                   OPERATIONAL DIAGNOSTIC CLASSIFICATION                │
│                                                                        │
│  1. [DIAG_CLOCK_SKEW]            ──> DueAt < ScheduledAt or Backward   │
│  2. [DIAG_DUPLICATE_DISPATCH]    ──> Idempotent duplicate suppressed   │
│  3. [DIAG_NOTIFICATION_FAILED]   ──> Local sink delivery error logged │
│  4. [DIAG_NOTIFICATION_QUARANTINE]─> Max retries exhausted, isolated  │
│  5. [DIAG_SCHEDULE_ALTERED]      ──> Recurrence / due window adjusted  │
│  6. [DIAG_SCHEDULE_CANCELLED]    ──> Pending slots purged safely       │
└────────────────────────────────────────────────────────────────────────┘
```

1. **`DIAG_CLOCK_SKEW` (Temporal Inconsistency):**
   - Emitted when a scheduled job or dispatch request exhibits temporal anomaly (e.g. `DueAt < ScheduledAt` or system clock shifts backwards).
   - Fails closed with `ErrClockSkewDetected`.
2. **`DIAG_DUPLICATE_DISPATCH` (Idempotent Suppression):**
   - Emitted when a recurring schedule worker attempts to generate an execution for an already dispatched slot.
   - Detects existing `idempotency_key`, returns existing execution reference, and logs diagnostic info with zero side-effects.
3. **`DIAG_NOTIFICATION_FAILED` (Local Sink Delivery Failure):**
   - Emitted when the local notification sink returns an injected or temporary error.
   - Increments retry counter and schedules bounded backoff.
4. **`DIAG_NOTIFICATION_QUARANTINED` (Quarantine Isolation):**
   - Emitted when a notification request exceeds `max_attempts` (default: 3).
   - Isolates the request in `StatusQuarantined` state, preventing endless retry storms.
5. **`DIAG_SCHEDULE_ALTERED` (Schedule Configuration Modification):**
   - Emitted when an authorized supervisor alters cadence, shift times, or assigned checklist template version.
   - Logs previous and new parameters with mandatory supervisory justification.
6. **`DIAG_SCHEDULE_CANCELLED` (Schedule Abort):**
   - Emitted when a schedule is cancelled, recording the count of pending executions pruned while preserving in-flight work.

### 4.2 Visible Failure Surface
All diagnostic anomalies are projected into supervisory dashboard read views (`MOD-REP`). Errors are categorized with visible severity badges (`WARNING`, `ERROR`, `CRITICAL`), ensuring operational visibility while strictly isolating frontline inspectors from background processing faults.

---

## 5. Notification Request & Status Lifecycle (`MOD-EVT`)

Within `MOD-EVT`, the notification engine coordinates deterministic job scheduling and local notification sink delivery:

### 5.1 Enumerated Notification Lifecycle States

```
┌─────────────┐       Dispatch to Sink      ┌───────────────┐
│   PENDING   │ ──────────────────────────> │   DELIVERED   │ (Terminal Success)
└──────┬──────┘                             └───────────────┘
       │ Sink Error
       ▼
┌─────────────┐       Retry < Max           ┌───────────────┐
│   FAILED    │ ──────────────────────────> │    PENDING    │
└──────┬──────┘                             └───────────────┘
       │ Retry >= Max
       ▼
┌─────────────┐       Replay by Supervisor  ┌───────────────┐
│ QUARANTINED │ ──────────────────────────> │   DELIVERED   │ (or back to Quarantined)
└─────────────┘                             └───────────────┘
```

| Notification Status | Semantics | Retry Behavior |
| :--- | :--- | :--- |
| **`StatusPending`** | Request queued for local sink dispatch. | Awaiting execution worker. |
| **`StatusDelivered`** | Successfully accepted by local notification sink. | Terminal state. No further processing. |
| **`StatusFailed`** | Sink delivery failed with an error. | Eligible for bounded retry if `attempts < max_attempts`. |
| **`StatusQuarantined`** | Maximum retries exhausted. | Halted. Requires manual supervisory replay. |

### 5.2 Notification Request Structure
```go
type NotificationRequest struct {
    RequestID        string              // ntf_req_syn_*
    TenantID         string              // ten_syn_*
    ExecutionID      string              // ins_syn_*
    RecipientSubject string              // usr_syn_*
    Channel          NotificationChannel // ChannelLocalSink, ChannelInternalLog, ChannelAuditJournal
    Status           NotificationStatus  // StatusPending, StatusDelivered, StatusFailed, StatusQuarantined
    Payload          map[string]string
    Attempts         int
    MaxAttempts      int
    LastError        string
    CreatedAt        time.Time
    UpdatedAt        time.Time
}
```

### 5.3 Controlled Replay Protocol
1. Replaying a quarantined notification requires explicit supervisory action via `ReplayNotification`.
2. Must provide `requestID`, non-blank `callerTenantID`, and authenticated `callerIdentity`.
3. Validates that the request is currently in `StatusQuarantined` and belongs to `callerTenantID`.
4. Increments attempt count and attempts redelivery to the local sink.

---

## 6. Delayed & Duplicate Reminders Handling

In operational field environments, scheduled reminders may experience delivery delays or processing retries:

### 6.1 Deterministic Reminder Idempotency Keys
To prevent spamming inspector client devices with duplicate reminders on network reconnection or background job retries, every reminder request derives an immutable idempotency key:

$$\text{reminder\_key} = \text{SHA-256}(\text{tenant\_id} \mathbin{\Vert} \text{execution\_id} \mathbin{\Vert} \text{reminder\_type} \mathbin{\Vert} \text{target\_due\_date})$$

- If a reminder dispatch job is executed multiple times, subsequent requests matching the same `reminder_key` are rejected as duplicates (`ErrDuplicateNotificationRequest`).

### 6.2 Graceful Degradation for Delayed Reminders
1. **Pre-Due Reminder Window:** Reminders are typically scheduled for dispatch prior to the core due time (e.g. 2 hours before `due_date`).
2. **Stale Reminder Suppression:** If the scheduler worker was delayed (e.g. server restart or node maintenance) and the current evaluation time has already passed `due_date`:
   - The pre-due reminder is **suppressed** as stale (`DIAG_STALE_REMINDER_SUPPRESSED`).
   - The system transitions directly to dispatching the overdue escalation notice, ensuring the inspector does not receive misleading "upcoming due" notices for already overdue tasks.

---

## 7. Clock Skew, Deterministic Clock Fixtures & Local Sinks

### 7.1 Deterministic Clock Abstraction
To guarantee 100% reproducible test execution without relying on unstable wall-clock time:
1. **Clock Interface:** The scheduler accepts a pluggable clock abstraction returning UTC timestamps.
2. **Simulated Clock Advances:** Unit and integration tests drive scheduling intervals by advancing the clock deterministically across anchor dates, recurrence intervals, due windows, and grace periods.

### 7.2 Clock Skew Defense & Temporal Integrity
1. **Temporal Invariant Enforcement:** Scheduling evaluation enforces strict temporal progression. Any job or dispatch proposal where `DueAt < ScheduledAt` is immediately rejected.
2. **Clock Shift Handling:** If system time drifts backwards or shifts unexpectedly, the scheduler refuses to dispatch further executions until reconciled, raising `ErrClockSkewDetected` and emitting `DIAG_CLOCK_SKEW`.

### 7.3 Strict Local Sink Exclusivity
In accordance with foundation gate `H040-010`:
1. **Channel Type Enforcement:** Permitted notification channels are strictly constrained to local sinks:
   - `ChannelLocalSink`: Synthetic in-memory queue.
   - `ChannelInternalLog`: Structured stdout log stream.
   - `ChannelAuditJournal`: Append-only audit table.
2. **Prohibited Network Channels:** Any request specifying external network channels (e.g. `EMAIL_SMTP`, `SMS_TWILIO`, `PUSH_FCM`) fails closed with:
   $$\text{ErrExternalChannelProhibited} = \text{"external notification channels are strictly prohibited under H040-010"}$$

---

## 8. Synthetic Operations Fixture Matrix

The following synthetic YAML fixture illustrates operational scenarios including delivery success, sink failure, quarantine, clock skew, and supervisory replay:

```yaml
schema_version: "1.0.0"
fixture_id: "fix_syn_scheduling_operations_v1"

scenarios:
  # Scenario 1: Clean Local Sink Delivery
  - scenario_id: "scen_syn_clean_delivery_01"
    tenant_id: "ten_syn_01"
    request_id: "ntf_req_syn_01"
    execution_id: "ins_syn_clean_01"
    recipient_subject: "usr_syn_inspector_01"
    channel: "LOCAL_SINK"
    status: "DELIVERED"
    attempts: 1
    max_attempts: 3
    last_error: null
    dispatched_at: "2026-09-10T01:00:00Z"
    business_state_impact: "NONE_ADVISORY_ONLY"

  # Scenario 2: Injected Sink Failure & Quarantine
  - scenario_id: "scen_syn_sink_failure_quarantine_02"
    tenant_id: "ten_syn_01"
    request_id: "ntf_req_syn_02"
    execution_id: "ins_syn_fault_02"
    recipient_subject: "usr_syn_inspector_01"
    channel: "LOCAL_SINK"
    status: "QUARANTINED"
    attempts: 3
    max_attempts: 3
    last_error: "simulated local sink connection reset"
    quarantined_at: "2026-09-10T01:05:00Z"
    business_state_impact: "NONE_INSPECTION_REMAINS_ACTIVE"

  # Scenario 3: Supervisory Replay of Quarantined Notification
  - scenario_id: "scen_syn_supervisory_replay_03"
    tenant_id: "ten_syn_01"
    request_id: "ntf_req_syn_02"
    replayed_by: "usr_syn_supervisor_01"
    replayed_at: "2026-09-10T02:00:00Z"
    status_after_replay: "DELIVERED"
    attempts_after_replay: 4
    business_state_impact: "NONE_ADVISORY_ONLY"

  # Scenario 4: Clock Skew Detection and Safe Failure
  - scenario_id: "scen_syn_clock_skew_04"
    tenant_id: "ten_syn_01"
    job_id: "job_syn_skew_04"
    scheduled_at: "2026-09-10T12:00:00Z"
    due_at: "2026-09-10T10:00:00Z"  # DueAt < ScheduledAt (Inverted)
    evaluation_result: "REJECTED_FAIL_CLOSED"
    diagnostic_code: "DIAG_CLOCK_SKEW"
    expected_error: "ErrClockSkewDetected"

  # Scenario 5: Delayed Reminder Graceful Degradation
  - scenario_id: "scen_syn_delayed_reminder_05"
    tenant_id: "ten_syn_01"
    execution_id: "ins_syn_delayed_05"
    planned_pre_due_reminder: "2026-09-10T06:00:00Z"
    actual_evaluation_time: "2026-09-10T09:00:00Z"
    core_due_time: "2026-09-10T08:00:00Z"
    outcome: "PRE_DUE_SUPPRESSED_TRANSITIONED_TO_OVERDUE"
    diagnostic_code: "DIAG_STALE_REMINDER_SUPPRESSED"
```

---

## 9. Governance Boundaries, Prohibitions & Operational Non-Claims

In strict compliance with `HDEC-V040-FOUNDATION-054` and `ASN-V040-I011-SCHEDULING-OPERATIONS-001`:

1. **100% Synthetic Data Policy (`H040-003`):** All tenant IDs (`ten_syn_*`), user subjects (`usr_syn_*`), execution IDs (`ins_syn_*`), and notification request IDs (`ntf_req_syn_*`) are synthetic local models. Zero customer data or real employee records are referenced.
2. **Preservation of Gate `H040-010` (HOLD):** Zero external notification delivery routes, SMTP servers, SMS gateways, push services, or cloud providers are authorized or deployed.
3. **Decoupled Advisory Posture:** Notification delivery success, failure, or delay carries zero authority to alter inspection assignment, due window, or cancellation status.
4. **Default-Deny Authority Invariant (`H040-004`):** Replaying quarantined notifications or altering schedule parameters requires authenticated supervisory roles.
5. **Specification-Only Credit:** Delivery of this baseline confers documentation and architectural specification credit only; zero operational release, deployment, or residual-risk acceptance is claimed.
