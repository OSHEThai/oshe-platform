---
document_id: ARC-V040-SCHED-001
title: v0.4.0 OSHE Inspect Inspection Scheduling, Recurrence, Scope, Checklist Binding, Due-Window, Time-Zone, and Idempotency Baseline
document_type: architecture_specification
document_version: 1.0.0
lifecycle_status: DRAFT
status: APPROVED_FOR_LOCAL_DEVELOPMENT
date: "2026-09-05"
author_role: Engineering Lead
author_pane: w9:p23
governing_issue: "GitHub Issue #120"
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
credit_boundary: INSPECTION_SCHEDULING_SPECIFICATION_ONLY_NO_EXTERNAL_DELIVERY_OR_LIVE_OPERATIONAL_ENACTMENT
---

# v0.4.0 OSHE Inspect Inspection Scheduling, Recurrence, Scope, Checklist Binding, Due-Window, Time-Zone, and Idempotency Baseline

## 1. Executive Summary & Governance Posture

### 1.1 Authority Baseline & Purpose
This architectural specification establishes the authoritative **Inspection Scheduling, Recurrence, Scope, Checklist Binding, Due-Window, Time-Zone, and Idempotency Baseline** for **Milestone v0.4.0 - OSHE Inspect Private Alpha** within `OSHEThai/oshe-platform`. It fulfills the deliverable requirements of **GitHub Issue #120 (`[V040-I009] Inspection Scheduling, Recurrence, and Assignment Dispatch Baseline`)** under Roadmap Topic `V040-T02` and standing owner foundation decision `HDEC-V040-FOUNDATION-054`.

The primary objective is to define a bounded, deterministic, dependency-free scheduling engine within the **Workflow and Action Module (`MOD-WFA`)** governing one-time and recurring inspection generation, explicit organizational and geographic scope binding, immutable checklist template version pinning, strict Asia/Bangkok time-zone resolution, due-window tracking, and cryptographic generation idempotency.

### 1.2 Retained Unselected Policies & Non-Claims
In strict compliance with `ASN-V040-I009-SCHEDULING-BASELINE-001` and `HDEC-V040-FOUNDATION-054`:
1. **Binding Scoring Policy (`HUMAN_OWNED_UNSELECTED`):** Schedule definitions may associate scoring reference metadata, but all scoring models remain provisional. Final binding scoring criteria remain human-owned pending owner decision under Issue #136 (`V040-I025`).
2. **Finding Closure Policy (`HUMAN_OWNED_UNSELECTED`):** Reinspection schedules spawned from safety defects adhere to workflow linking rules, but defect closure and risk sign-off policies remain human-owned under Issue #134 (`V040-I023`).
3. **Offline Authority Model (`HUMAN_OWNED_UNSELECTED`):** Client-side inspection cache validity and offline schedule prefetch limits remain human-owned under Issue #126 (`V040-I015`).
4. **No External Notification or Message Delivery:** Zero external email gateways (SMTP/SES), SMS aggregators (Twilio), or mobile push notification services (FCM/APNS) are authorized, integrated, or activated. All scheduling dispatches and notifications operate strictly within synthetic in-app queues and local memory structures.
5. **Retained Human Gate Holds:** Foundation gates `H040-007` (Technical Release Authorization), `H040-008` (Real Participant / Private-Alpha UAT Onboarding), `H040-009` (Binding Support Ownership), `H040-010` (External Environment/Route Activation), and `H040-011` (Final Milestone Outcome) remain strictly on **`HOLD`**.

---

## 2. Schedule Entity Model & Lifecycle States (`MOD-WFA`)

Under `ARC-V040-DOMAIN-001`, the **Workflow and Action Module (`MOD-WFA`)** maintains sole authoritative ownership of `InspectionSchedule` and `Assignment` entities.

### 2.1 Entity Model Specification
```
┌────────────────────────────────────────────────────────────────────────┐
│                          InspectionSchedule                            │
│  - schedule_id: String (sch_syn_*)                                     │
│  - tenant_id: String (ten_syn_*)                                       │
│  - title: LocalizedString (en-US, th-TH)                               │
│  - trigger_type: Enum (CALENDAR_RECURRING, MANUAL_ADHOC, ...)         │
│  - status: Enum (ACTIVE, PAUSED, COMPLETED, CANCELLED, RETIRED)        │
│  - scope: ScopeBinding (site_id, area_id, equipment_id)                │
│  - checklist_binding: ChecklistVersionBinding (template_id, version)   │
│  - recurrence: RecurrenceRule (frequency, interval, days_of_week)     │
│  - due_window: DueWindowConfig (window_hours, grace_period_hours)      │
│  - assignment: AssignmentRule (assignee_type, assignee_subject)        │
│  - time_zone: String ("Asia/Bangkok")                                  │
│  - next_run_at: Timestamp (UTC ISO 8601)                              │
└──────────────────────────────────┬─────────────────────────────────────┘
                                   │ Spawns (Deterministic Dispatch)
                                   ▼
┌────────────────────────────────────────────────────────────────────────┐
│                          InspectionExecution                           │
│  - execution_id: String (ins_syn_*)                                    │
│  - schedule_id: String (sch_syn_*)                                     │
│  - idempotency_key: String (SHA-256 Digest)                            │
│  - status: Enum (SCHEDULED, ASSIGNED, IN_PROGRESS, COMPLETED, ...)     │
│  - scheduled_date: Date ("YYYY-MM-DD")                                 │
│  - window_start: Timestamp (UTC ISO 8601)                              │
│  - due_date: Timestamp (UTC ISO 8601)                                  │
│  - grace_until: Timestamp (UTC ISO 8601)                               │
│  - pinned_template_version: String ("1.1.0")                           │
│  - pinned_content_digest: String (SHA-256 Digest)                      │
└────────────────────────────────────────────────────────────────────────┘
```

### 2.2 Enumerated Schedule Lifecycle States
Every `InspectionSchedule` exists in exactly one of the following five discrete states:

| Schedule State | Execution Generation | Modification Allowed | State Semantics & Boundary |
| :--- | :--- | :--- | :--- |
| **`ACTIVE`** | **Enabled** | Bounded edits | Operational schedule. The dispatch engine evaluates recurrence rules and spawns new inspection executions. |
| **`PAUSED`** | **Suspended** | Yes | Temporarily halted schedule (e.g. during facility maintenance or seasonal shutdown). No new executions are generated. |
| **`COMPLETED`** | Disabled | Read-only | Terminal state for finite schedules (e.g. `ONCE` or recurrence count exhausted). |
| **`CANCELLED`** | Disabled | Read-only | Explicitly aborted schedule. Pending unstarted executions are marked cancelled; in-flight executions remain pinned. |
| **`RETIRED`** | Disabled | Read-only | Decommissioned schedule superseded by an updated program. Preserved permanently for historical audit compliance. |

---

## 3. Schedule Triggers & Dispatch Classification

Inspection schedules originate from one of four mutually exclusive trigger classes:

1. **`CALENDAR_RECURRING` (Periodic Scheduled Inspection):**
   - Automatically scheduled by temporal recurrence engine (e.g. daily shift inspection, weekly safety walk, monthly audit).
   - Driven by calendar date evaluation in canonical `Asia/Bangkok` time.
2. **`MANUAL_ADHOC` (On-Demand Targeted Inspection):**
   - Created manually by an authorized safety officer, site supervisor, or operations lead.
   - Non-recurring single execution (`frequency: ONCE`) targeting a specific area, task, or contractor.
3. **`EVENT_TRIGGERED` (Operational Incident or Process Change):**
   - Spawned in response to an operational trigger (e.g. severe weather alert, new chemical delivery, post-incident safety stand-down).
   - Generates an immediate inspection window with priority escalation flags.
4. **`REINSPECTION_DEFECT` (Corrective Action Verification):**
   - Spawned automatically by `MOD-WFA` when a corrective action item (CAPA) is submitted or when an initial finding is reopened upon review.
   - Explicitly links to `source_finding_id` and `source_execution_id` to establish end-to-end audit traceability.

---

## 4. Scope Definition & Checklist Version Binding

### 4.1 Hierarchical Scope Binding
Every schedule must bind explicitly to the organizational and physical asset hierarchy:
- `tenant_id`: Authoritative tenant identifier (`ten_syn_*`).
- `company_id`: Specific subsidiary or operational operating company.
- `site_id`: Geographic facility or plant location (`ste_syn_*`).
- `area_id`: Sub-facility area, zone, or building floor (`ara_syn_*`).
- `equipment_id`: (Optional) Specific asset or machinery tag (`eqp_syn_*`).

### 4.2 Immutable Checklist Version Binding Invariants
In accordance with `ARC-V040-CHKLIFE-001` and `ARC-V040-DOMAIN-001`:
1. **Binding to Published Immutable Versions Only:** A schedule may bind **only** to a checklist template iteration currently in `PUBLISHED_IMMUTABLE` state. Binding to `DRAFT`, `UNDER_REVIEW`, `REJECTED`, or `RETIRED` versions fails closed with `ErrInvalidChecklistVersionState`.
2. **Cryptographic Content Digest Verification:** The schedule records `template_id`, `published_version`, and `content_digest`. At execution instantiation time, the dispatch engine verifies that `content_digest` strictly matches the published schema record in `MOD-CFG`.
3. **Strict Execution Pinning:** Once an `InspectionExecution` is generated from a schedule, its checklist schema, version, and content digest are **permanently pinned**. If the parent schedule is updated to point to a newer checklist iteration (e.g. `v1.2.0`), all existing in-flight and completed executions remain bound to their historical version (`v1.1.0`).

---

## 5. Deterministic Recurrence Rules & Evaluation Model

### 5.1 Recurrence Pattern Enumeration
The recurrence engine supports four deterministic frequency patterns:

| Frequency | Supported Intervals | Day Constraints | Deterministic Next Run Rule |
| :--- | :--- | :--- | :--- |
| **`ONCE`** | `interval = 1` | None | Executes once at `start_date`. Transitions schedule to `COMPLETED` upon dispatch. |
| **`DAILY`** | `interval \ge 1` | Optional `weekdays_only` | Next occurrence = $\text{CurrentDate} + (\text{interval} \times \text{Days})$. |
| **`WEEKLY`** | `interval \ge 1` | `days_of_week` (`MON`..`SUN`) | Advances to the next matching day in `days_of_week`. If all days in the current week elapsed, jumps $\text{interval}$ weeks. |
| **`MONTHLY`** | `interval \ge 1` | `day_of_month` (1..31) | Advances by $\text{interval}$ months to specified calendar day. If day exceeds month length (e.g. Feb 31), clamps deterministically to the last valid day of the month (e.g. Feb 28/29). |

### 5.2 Deterministic Next-Occurrence Algorithm
The calculation of `next_run_at` is a pure mathematical function of:
$$\text{NextOccurrence}(\text{RecurrenceRule}, \text{AnchorDate}, \text{TimeZone})$$
- Zero system clock drift dependency: Next occurrence evaluation uses the scheduled target date as the base anchor, not the current wall-clock time, preventing schedule slip over long operational cycles.

---

## 6. Due Windows, Grace Periods, and Overdue Progression

### 6.1 Due-Window Temporal Anatomy
Each generated inspection execution defines three explicit temporal boundaries:

```
┌─────────────────────────┬─────────────────────────┬─────────────────────────┐
│     ADVANCE WINDOW      │       CORE WINDOW       │      GRACE PERIOD       │
│ (Inspection Available)  │ (Planned Execution)     │ (Buffer Before Overdue) │
└─────────────────────────┴─────────────────────────┴─────────────────────────┘
▲                         ▲                         ▲                         ▲
window_start              scheduled_start           due_date                  grace_until
```

1. **`window_start`:** The earliest moment the inspection can be downloaded and initiated on a client device (e.g. 06:00:00 Asia/Bangkok on the scheduled date).
2. **`due_date`:** The target completion deadline (e.g. 18:00:00 Asia/Bangkok on the scheduled date).
3. **`grace_period_hours`:** Controlled operational buffer (e.g. 12 or 24 hours) permitting delayed submission without immediate compliance penalty.
4. **`grace_until`:** Hard boundary:
   $$\text{grace\_until} = \text{due\_date} + (\text{grace\_period\_hours} \times 3600\text{ seconds})$$

### 6.2 Deterministic Compliance Status Progression
Based on the current evaluation time relative to the due window:

| Condition | Compliance Status | Operational Action & Visibility |
| :--- | :--- | :--- |
| $\text{Now} \le \text{due\_date}$ | **`ON_TIME`** | Normal execution. Unhighlighted in dashboard views. |
| $\text{due\_date} < \text{Now} \le \text{grace\_until}$ | **`IN_GRACE_PERIOD`** | Warning state. Amber badge displayed on client device and supervisor dashboard. |
| $\text{Now} > \text{grace\_until}$ | **`OVERDUE`** | Escalated non-compliance state. Red indicator emitted to supervisory reporting views. |

---

## 7. Canonical Time-Zone Handling (`Asia/Bangkok`)

In strict compliance with `ARC-V040-PROF-001`:

1. **Sole Canonical Time Zone:** All schedule configurations, calendar recurrence calculations, due window boundaries, and operational shift rules evaluate exclusively within **`Asia/Bangkok` (Indochina Time, UTC+07:00)**.
2. **UTC Storage Invariant:** While evaluated in `Asia/Bangkok`, all stored database timestamps and wire-format DTOs must use **UTC ISO 8601 formatting with millisecond precision** (e.g. `2026-09-10T01:00:00.000Z`).
3. **Zero Daylight Saving Time (DST) Ambiguity:** The `Asia/Bangkok` time zone observes a constant UTC offset of $+07:00$ with zero Daylight Saving Time transitions, ensuring 100% deterministic day-boundary arithmetic without missing or duplicated clock hours.

---

## 8. Idempotency, Deduplication & Generation Keys

To guarantee zero duplicate inspection generation in distributed, retrying, or offline-reconnecting environments:

### 8.1 Deterministic Idempotency Key Composition
Every generated `InspectionExecution` derives an immutable cryptographic deduplication key:

$$\text{idempotency\_key} = \text{SHA-256}(\text{tenant\_id} \mathbin{\Vert} \text{schedule\_id} \mathbin{\Vert} \text{site\_id} \mathbin{\Vert} \text{area\_id} \mathbin{\Vert} \text{checklist\_version} \mathbin{\Vert} \text{scheduled\_date})$$

Example Key Input:
`ten_syn_01|sch_syn_weekly_plant_01|ste_syn_rayong_01|ara_syn_boiler_01|1.1.0|2026-09-14`

### 8.2 Dispatch Deduplication Invariant
1. **Unique Database Constraint:** A database-level unique constraint is enforced on `(tenant_id, idempotency_key)`.
2. **Idempotent Dispatch Execution:** If a schedule dispatch worker runs multiple times for the same calendar slot (e.g. due to process restart or network retry), the second execution detects the existing `idempotency_key`, returns the existing `execution_id`, and safely discards the duplicate operation without error or data mutation.

---

## 9. Safe Cancellation, Pause & Retirement Behaviors

### 9.1 Schedule State Transition Invariants
- **Pausing a Schedule (`ACTIVE` $\to$ `PAUSED`):**
  - Halts generation of future inspection executions.
  - Leaves all existing scheduled, assigned, and in-progress inspection executions completely active and unaffected.
- **Resuming a Schedule (`PAUSED` $\to$ `ACTIVE`):**
  - Recalculates `next_run_at` starting from the next valid calendar slot following the resumption timestamp. Missed occurrences during the paused period are **not** backfilled unless explicitly authorized via a separate ad-hoc trigger.
- **Cancelling a Schedule (`ACTIVE` / `PAUSED` $\to$ `CANCELLED`):**
  - Marks the schedule permanently inactive.
  - Future dispatch generation is permanently terminated.
  - Automatically cancels pending executions in `SCHEDULED` status that have not yet been assigned or started.
  - Any execution in `IN_PROGRESS` or `COMPLETED` remains intact to preserve historical field work.
- **Retiring a Schedule (`ACTIVE` / `PAUSED` $\to$ `RETIRED`):**
  - Used when an inspection schedule is decommissioned or replaced by an updated program.
  - Requires a mandatory non-empty `retirement_reason`.
  - Preserved in read-only state for longitudinal safety reporting.

---

## 10. Synthetic Notification & In-App Dispatch Boundary

In strict compliance with `ASN-V040-I009-SCHEDULING-BASELINE-001` and `ARC-V040-PROF-001`:

1. **Zero External Notification Claim:**
   - External notification channels (email, SMS, WhatsApp, Telegram, push notifications) are strictly **`UNSUPPORTED_PRIVATE_ALPHA`**.
   - No SMTP, SMS API, or external messaging libraries are permitted or integrated.
2. **Local In-App Notification Queue:**
   - Schedule dispatch events generate local synthetic notification records in the `MOD-WFA` in-app notification store:
```yaml
notification_record:
  notification_id: "ntf_syn_01"
  recipient_subject: "usr_syn_inspector_01"
  notification_type: "INSPECTION_ASSIGNED"
  title: "New Scheduled Inspection Assigned"
  body: "Weekly Plant Safety Inspection assigned for Site Rayong Plant A."
  execution_id: "ins_syn_01"
  dispatched_at: "2026-09-10T01:00:00Z"
  read_status: "UNREAD"
```
   - These records serve exclusively to support local UI badge rendering and unit test verification.

---

## 11. Synthetic Multi-Schedule Fixture Matrix

The following synthetic YAML fixture illustrates the complete scheduling configuration for Milestone v0.4.0 Private Alpha:

```yaml
schema_version: "1.0.0"
fixture_id: "fix_syn_inspection_schedules_v1"

schedules:
  # Schedule 1: Weekly Recurrent Site Safety Walk
  - schedule_id: "sch_syn_weekly_plant_01"
    tenant_id: "ten_syn_01"
    title:
      en-US: "Weekly Rayong Plant General Safety Walk"
      th-TH: "การตรวจความปลอดภัยทั่วไปประจำสัปดาห์ โรงงานระยอง"
    trigger_type: "CALENDAR_RECURRING"
    status: "ACTIVE"
    time_zone: "Asia/Bangkok"
    scope:
      company_id: "cmp_syn_manufacturing_01"
      site_id: "ste_syn_rayong_01"
      area_id: "ara_syn_production_hall_01"
      equipment_id: null
    checklist_binding:
      template_id: "chk_syn_pilot_plant_safety_v1"
      published_version: "1.1.0"
      content_digest: "8f4e2b1c3a7d9e5f0b2a4c6d8e1f3a5b7c9d1e3f5a7b9c1d3e5f7a9b1c3d5e7f"
    recurrence:
      frequency: "WEEKLY"
      interval: 1
      days_of_week: ["MON"]
      start_date: "2026-09-07"
      end_condition: "NEVER"
    due_window:
      window_start_time: "08:00:00"
      due_time: "17:00:00"
      grace_period_hours: 24
    assignment:
      assignee_type: "NAMED_USER"
      assignee_subject: "usr_syn_inspector_01"
      fallback_role: "Safety Supervisor"

  # Schedule 2: Monthly Confined Space Safety Inspection
  - schedule_id: "sch_syn_monthly_confined_01"
    tenant_id: "ten_syn_01"
    title:
      en-US: "Monthly Boiler Confined Space Safety Inspection"
      th-TH: "การตรวจความปลอดภัยพื้นที่อับอากาศประจำเดือน หม้อน้ำอุตสาหกรรม"
    trigger_type: "CALENDAR_RECURRING"
    status: "ACTIVE"
    time_zone: "Asia/Bangkok"
    scope:
      company_id: "cmp_syn_manufacturing_01"
      site_id: "ste_syn_rayong_01"
      area_id: "ara_syn_boiler_room_01"
      equipment_id: "eqp_syn_boiler_b1"
    checklist_binding:
      template_id: "chk_syn_confined_space_v1"
      published_version: "1.0.0"
      content_digest: "d41d8cd98f00b204e9800998ecf8427e"
    recurrence:
      frequency: "MONTHLY"
      interval: 1
      day_of_month: 15
      start_date: "2026-09-15"
      end_condition: "NEVER"
    due_window:
      window_start_time: "07:30:00"
      due_time: "16:30:00"
      grace_period_hours: 48
    assignment:
      assignee_type: "ROLE_BASED"
      assignee_role: "Confined Space Competent Person"
      fallback_role: "Lead Inspector"

  # Schedule 3: Manual Ad-Hoc Post-Maintenance Inspection
  - schedule_id: "sch_syn_adhoc_maint_01"
    tenant_id: "ten_syn_01"
    title:
      en-US: "Ad-Hoc Post-Maintenance Safety Clearance"
      th-TH: "การตรวจประเมินความปลอดภัยเฉพาะกิจหลังงานซ่อมบำรุง"
    trigger_type: "MANUAL_ADHOC"
    status: "COMPLETED"
    time_zone: "Asia/Bangkok"
    scope:
      company_id: "cmp_syn_manufacturing_01"
      site_id: "ste_syn_rayong_01"
      area_id: "ara_syn_turbine_hall_01"
      equipment_id: "eqp_syn_turbine_t1"
    checklist_binding:
      template_id: "chk_syn_pilot_plant_safety_v1"
      published_version: "1.1.0"
      content_digest: "8f4e2b1c3a7d9e5f0b2a4c6d8e1f3a5b7c9d1e3f5a7b9c1d3e5f7a9b1c3d5e7f"
    recurrence:
      frequency: "ONCE"
      interval: 1
      start_date: "2026-09-05"
      end_condition: "COUNT"
      max_occurrences: 1
    due_window:
      window_start_time: "13:00:00"
      due_time: "17:00:00"
      grace_period_hours: 8
    assignment:
      assignee_type: "NAMED_USER"
      assignee_subject: "usr_syn_inspector_02"
      fallback_role: "Shift Supervisor"
```

---

## 12. Governance Boundaries, Prohibitions & Non-Claims

In strict compliance with `HDEC-V040-FOUNDATION-054` and `ASN-V040-I009-SCHEDULING-BASELINE-001`:

1. **100% Synthetic Data Policy (`H040-003`):** All schedule IDs (`sch_syn_*`), tenant IDs (`ten_syn_*`), user IDs (`usr_syn_*`), site IDs (`ste_syn_*`), and asset IDs (`eqp_syn_*`) are synthetic local models. Zero customer data, real factory records, or employee personal data are referenced.
2. **Default-Deny Authority Invariant (`H040-004`):** Modifying, pausing, or retiring inspection schedules requires authenticated administrative or supervisory roles. Inspectors have read-only visibility into assigned dispatches.
3. **No External Route or Message Activation (`H040-007` & `H040-010` HOLD):** Public scheduling endpoints, webhook integrations, and external email/SMS notification dispatches remain strictly on HOLD.
4. **No Real Participant Onboarding (`H040-008` HOLD):** Zero real field inspectors or plant operators are recruited or scheduled.
5. **Specification-Only Credit:** Delivery of this baseline confers documentation and architectural model credit only; zero operational release, deployment, or residual-risk acceptance is claimed.
