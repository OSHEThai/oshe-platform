---
document_id: ARC-V040-RECOV-001
title: v0.4.0 OSHE Inspect Local Draft, Response/Media Queue, Sync State, Retry, Interruption Recovery, and Visible User-Guidance Baseline
document_type: architecture_specification
document_version: 1.0.0
lifecycle_status: DRAFT
status: APPROVED_FOR_LOCAL_DEVELOPMENT
date: "2026-09-05"
author_role: Engineering Lead
author_pane: w9:p23
governing_issue: "GitHub Issue #125"
authority_source: HDEC-V040-FOUNDATION-054
governing_decisions:
  - HDEC-V040-FOUNDATION-054
  - HDEC-V030-ENTRY-AND-POLICY-052
  - ADR-0005
  - ADR-0006
milestone: "v0.4.0 - OSHE Inspect Private Alpha"
consumed_prework_artifacts:
  - "ARC-V040-EVD-001 (docs/architecture/v040-evidence-capture-prework.md)"
pending_integrations:
  - PENDING_INTEGRATION_ISSUE_128_V040_I017
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
  evidence_retention_policy: HUMAN_OWNED_UNSELECTED
  external_storage_provider_policy: HUMAN_OWNED_UNSELECTED
credit_boundary: LOCAL_DRAFT_RECOVERY_SPECIFICATION_ONLY_NO_EXTERNAL_DELIVERY_OR_LIVE_OPERATIONAL_ENACTMENT
---

# v0.4.0 OSHE Inspect Local Draft, Response/Media Queue, Sync State, Retry, Interruption Recovery, and Visible User-Guidance Baseline

## 1. Executive Summary & Governance Posture

### 1.1 Authority Baseline & Purpose
This architectural specification establishes the authoritative **Local Draft, Response and Media Queue, Synchronization State, Retry Mechanics, Interruption Recovery, and Visible User-Guidance Baseline** for **Milestone v0.4.0 - OSHE Inspect Private Alpha** within `OSHEThai/oshe-platform`. It fulfills the deliverable requirements of **GitHub Issue #125 (`[V040-I014] Local Draft, Interruption Recovery, and Synchronization State Baseline`)** under Roadmap Topic `V040-T03` and standing owner foundation decision `HDEC-V040-FOUNDATION-054`.

The primary objective is to define a resilient, client-side persistence and recovery architecture operating within the supported browser envelope (`H040-002`) to ensure:
- Zero data loss across abrupt network drops, browser tab closures, process crashes, and mobile device power cuts (`NFR-REC-01`).
- Atomic, write-ahead journaling for checklist responses, localized notes, and local evidence media blobs.
- Deterministic synchronization state transitions, exponential backoff retries, and cryptographic idempotency.
- Robust low-storage defenses preventing data corruption without silent pruning of draft data.
- Continuous, visible user guidance reflecting sync status, queue depth, and recovery actions in bilingual English and Thai interfaces.

### 1.2 Consumption of I017 Prework & Retention of Issue #128 Open
In strict compliance with assignment directives:
1. **Consumption of I017 Prework (`ARC-V040-EVD-001`):** This baseline consumes the evidence capture interfaces, sanitized metadata models, and association bindings established in `docs/architecture/v040-evidence-capture-prework.md` as non-binding prework.
2. **Pending Integration Status (`PENDING_INTEGRATION_ISSUE_128_V040_I017`):** Downstream end-to-end integration remains open. **GitHub Issue #128 (`[V040-I017] Implement Evidence Capture, Metadata, Local Queue, Preview, Caption, Source Context, and Record Association`) is formally preserved as OPEN** until all dependent pipeline components are qualified and merged.

### 1.3 Retained Unselected Policies & Non-Claims
1. **Binding Scoring Policy (`HUMAN_OWNED_UNSELECTED`):** Local score previews and compliance calculations are advisory only; final binding compliance scoring remains human-owned under Issue #136 (`V040-I025`).
2. **Finding Closure Policy (`HUMAN_OWNED_UNSELECTED`):** Reinspections and non-conformance closures require server-side verification and remain human-owned under Issue #134 (`V040-I023`).
3. **Offline Authority Model (`HUMAN_OWNED_UNSELECTED`):** Operational lease expiration ceilings (24 hours) are private alpha baselines; long-term lease governance remains human-owned under Issue #126 (`V040-I015`).
4. **External Storage Provider Policy (`HUMAN_OWNED_UNSELECTED`):** Cloud blob storage (S3, Azure Blob, Cloudflare R2) is unselected; all media queueing defined herein operates strictly within local browser memory and synthetic client databases.
5. **Retained Human Gate Holds:** Foundation gates `H040-007` (Technical Release Authorization), `H040-008` (Real Participant / Private-Alpha UAT Onboarding), `H040-009` (Binding Support Ownership), `H040-010` (External Environment/Route Activation), and `H040-011` (Final Milestone Outcome) remain strictly on **`HOLD`**.

---

## 2. Local Draft Storage Architecture & Partitioning (`IndexedDB`)

In accordance with `ARC-V040-MOBOFF-001` and data protection gate `H040-003`, client persistence is isolated within a dedicated browser `IndexedDB` database (`oshe_offline_db`):

```
┌────────────────────────────────────────────────────────────────────────┐
│                   oshe_offline_db (IndexedDB Database)                 │
├────────────────────────────────────────────────────────────────────────┤
│  1. work_packages       : Pinned templates, scopes, reference bundles  │
│  2. response_drafts     : Keyed by (execution_id, question_id)         │
│  3. evidence_queue      : Metadata, digest, sync state per attachment  │
│  4. evidence_blobs      : Binary image blobs stripped of EXIF data     │
│  5. mutation_journal    : Append-only write-ahead log (monotonic seq) │
│  6. sync_state_register : Per-execution sync progress & lease tokens   │
└────────────────────────────────────────────────────────────────────────┘
```

### 2.1 Object Store Specifications
1. **`work_packages`:** Stores downloaded immutable work packages (`wpk_syn_*`). Contains pinned checklist schema, question rules, and translation bundles. Read-only during inspection execution.
2. **`response_drafts`:** Stores active responses keyed by composite index `[execution_id + question_id]`.
   - Fields: `execution_id`, `question_id`, `selected_options`, `numeric_value`, `text_notes`, `photos_attached_count`, `updated_at`, `client_draft_version`.
3. **`evidence_queue`:** Stores evidence metadata records conforming to `ARC-V040-EVD-001`.
   - Fields: `evidence_id`, `execution_id`, `association_type`, `association_id`, `sha256_digest`, `media_type`, `size_bytes`, `caption`, `sync_status`, `retry_count`.
4. **`evidence_blobs`:** Content-addressed blob store keyed by `sha256_digest`. Holds clean binary media payloads after EXIF scrubbing.
5. **`mutation_journal`:** Append-only transaction journal capturing every state-changing event with a monotonically increasing integer `seq_num`. Enables atomic write-ahead commit and deterministic crash recovery.

---

## 3. Response & Media Queue Lifecycle and Sync State Machine

Every locally authored item (response, finding, or evidence attachment) transitions through an explicit, deterministic synchronization state machine:

```
┌─────────────┐       User Input       ┌────────────────┐       Online Event     ┌─────────────┐
│ DRAFT_LOCAL │ ─────────────────────> │ QUEUED_FOR_SYNC│ ─────────────────────> │   SYNCING   │
└─────────────┘                        └────────────────┘                        └──────┬──────┘
                                                ▲                                       │
                                                │ Retry Backoff                         │
                                                │                                       ▼
                                       ┌────────────────┐    Server Commit OK    ┌────────────────────┐
                                       │ SYNC_RETRYABLE │ <───────────────────── │ SYNC_ACKNOWLEDGED  │
                                       └────────────────┘ (5xx / Network Drop)   └────────────────────┘
                                                │
                                                │ Version Conflict (409)
                                                ▼
                                       ┌────────────────┐
                                       │   QUARANTINED  │ (Manual Reconciliation)
                                       └────────────────┘
```

### 3.1 Sync State Enumeration & Semantics

| Synchronization State | Client Editing Posture | Network Behavior | State Semantics & Governance Boundary |
| :--- | :--- | :--- | :--- |
| **`DRAFT_LOCAL`** | **Editable** | None | Item is being edited by the inspector. Write-ahead mutations recorded locally; not yet marked for transmission. |
| **`QUEUED_FOR_SYNC`** | Locked for sync | Awaiting dispatch | Inspector transitioned item to ready state or initiated background sync. Added to outbound synchronization batch. |
| **`SYNCING`** | Read-Only | HTTP POST active | Payload is currently in transit to central server endpoints. |
| **`SYNC_ACKNOWLEDGED`** | Read-Only | Committed | Server authority validated and committed monotonic version. Local mutation journal entry marked reconciled. |
| **`SYNC_FAILED_RETRYABLE`** | **Editable** | Scheduled backoff | Network dropped, timed out, or central server returned transient error (HTTP 502/503/504). Scheduled for retry. |
| **`SYNC_QUARANTINED`** | Read-Only | Halted | Concurrency conflict detected (HTTP 409) or supervisory reassignment occurred. Isolated for manual review (`H040-005`). |

---

## 4. Interruption Recovery Engine (Network, App & Device Crashes)

The recovery engine guarantees complete state restoration across three distinct operational disruption domains:

### 4.1 Domain 1: Network Disconnection & Flapping
- **Interruption Vector:** Rapid alternation between cellular (4G), Wi-Fi, and total dead-zones in industrial basements or plant interiors.
- **Recovery Mechanism:**
  1. The client registers standard `navigator.onLine`, `window.addEventListener('online')`, and heartbeats.
  2. If network drops mid-request, in-flight HTTP requests abort cleanly without corrupting local records.
  3. Affected queue items transition from `SYNCING` to `SYNC_FAILED_RETRYABLE`.
  4. Local editing continues uninterrupted against `IndexedDB`.
  5. Upon connection restoration, the queue manager applies an exponential backoff with random jitter before resuming transmission:
     $$T_{\text{backoff}} = \min(T_{\text{max}}, T_{\text{base}} \times 2^{\text{attempt}}) + \text{rand}(0, 1000\text{ms})$$

### 4.2 Domain 2: Browser Tab Closure & Webview Termination
- **Interruption Vector:** Inspector accidentally closes the browser tab, Android OS terminates browser background process under memory pressure, or browser is force-closed.
- **Recovery Mechanism (`NFR-REC-01`):**
  1. Every response keystroke and choice selection commits transactionally to `response_drafts` and `mutation_journal` before UI DOM update acknowledgment.
  2. Upon page reload, the client application inspects `IndexedDB` during initialization.
  3. The recovery engine detects incomplete inspection sessions, reads the `mutation_journal`, and reconstructs the full questionnaire state, current section pointer, answer values, and attachment associations with **100% fidelity**.
  4. A visible notification banner confirms: `"DRAFT RESTORED: Resumed inspection from local journal."`

### 4.3 Domain 3: Device Power Loss & Operating System Crash
- **Interruption Vector:** Mobile device battery exhaustion, sudden thermal shutdown, or kernel panic.
- **Recovery Mechanism:**
  1. `IndexedDB` operations execute within atomic transactions (`readwrite`). A power failure during write either fully commits the journal entry or leaves the previous consistent state intact (ACID compliance).
  2. Upon device reboot and browser restart, the client executes journal integrity validation. Zero orphaned partial records are permitted; uncommitted writes are rolled back to the last atomic sequence.

---

## 5. Idempotent Retries & Duplicate Rejection / Quarantine (`H040-005`)

To prevent duplicate responses, double-counted findings, or multiplied media records during automatic retries:

### 5.1 Deterministic Mutation Idempotency Key
Every local mutation generates an immutable cryptographic deduplication key:

$$\text{mutation\_key} = \text{SHA-256}(\text{tenant\_id} \mathbin{\Vert} \text{execution\_id} \mathbin{\Vert} \text{item\_id} \mathbin{\Vert} \text{client\_seq\_num})$$

- `item_id`: Target `question_id`, `finding_id`, or `evidence_id`.
- `client_seq_num`: Monotonically increasing sequence number from the local `mutation_journal`.

### 5.2 Server-Side Processing Invariants
1. **Idempotent Acknowledgment:** When the central server receives a mutation containing a previously processed `mutation_key`, it does not insert a duplicate record. It returns the existing commit receipt (`HTTP 200 OK`) and current monotonic `entity_version`.
2. **Conflict Quarantine Trigger:** If an incoming mutation's `base_version` does not match the server's current version (e.g. supervisor reassigned the work, or checklist was updated), the server responds with `HTTP 409 Conflict`.
3. **Local Quarantine Isolation:** The client marks the affected record as `SYNC_QUARANTINED`, ceases automatic retries for that item, displays a non-blocking conflict indicator to the inspector, and preserves the local draft for supervisory reconciliation.

---

## 6. Low-Storage Defenses & Defensive Quotas

In mobile field environments, storage exhaustion is a critical risk factor. The system enforces strict capacity monitoring:

### 6.1 Storage Quota Thresholds (`navigator.storage.estimate`)
The client monitors available storage before accepting new media attachments:

| Storage Envelope | Threshold Condition | Client Operational Posture | Frontline UI Indicator |
| :--- | :--- | :--- | :--- |
| **Normal** | Local usage $< 80\%$ quota and $< 40\text{ MB}$ | Full functionality enabled | Standard subtle cloud icon |
| **Warning** | Local usage $\ge 80\%$ quota or $> 40\text{ MB}$ | Capture allowed with caution | Amber warning banner: `"STORAGE WARNING: Synchronize pending drafts to free local space."` |
| **Critical Ceiling** | Local usage $\ge 95\%$ quota or `QuotaExceededError` | **Photo capture locked** | Red error banner: `"STORAGE FULL: Camera disabled. Responses and text notes remain active."` |

### 6.2 Draft Provenance Preservation Invariant
1. **Zero Silent Eviction:** Under no circumstances does the client application prune, truncate, or overwrite active inspection drafts or evidence blobs to reclaim disk space.
2. **Fail-Closed Capture Lock:** When critical storage thresholds are breached, the application refuses new image captures (`ErrLocalStorageQuotaExceeded`) while permitting text responses, choice selections, and synchronization drains.

---

## 7. Visible User-Guidance & UI Synchronization Indicators

Inspectors must never be left in doubt regarding synchronization status, network state, or recovery outcomes:

### 7.1 Global Header Status Banner
The application header displays a persistent, real-time status indicator:

| Network & Sync Condition | Visual Badge | Display Text (English / Thai) | Actionable Guidance |
| :--- | :--- | :--- | :--- |
| **Connected & Synced** | Green Dot | `"All changes saved" / "บันทึกข้อมูลเรียบร้อย"` | Normal operation |
| **Offline (Draft Active)** | Amber Cloud | `"Offline (3 pending)" / "ออฟไลน์ (รอส่ง 3 รายการ)"` | Informational; work is safely preserved |
| **Sync in Progress** | Blue Spinning | `"Syncing 2 of 5..." / "กำลังส่งข้อมูล 2 จาก 5..."` | Background activity indicator |
| **Retry in Progress** | Amber Pulse | `"Retrying in 15s..." / "จะลองส่งใหม่ใน 15 วินาที"` | Network recovery countdown |
| **Sync Quarantined** | Orange Warning | `"Sync Conflict" / "พบข้อขัดแย้งในการส่งข้อมูล"` | Tap for supervisory assistance |

### 7.2 Item-Level Sync Status Markers
Every question card in the inspection form renders a localized micro-badge:
- `SYNCED`: Subtle green checkmark.
- `PENDING_LOCAL`: Amber clock icon denoting uncommitted local draft.
- `ATTACHMENT_QUEUED`: Paperclip icon with badge showing queued photo count and upload progress bar.

---

## 8. Synthetic Operations Fixture Matrix

The following synthetic YAML fixture illustrates local draft recovery, media queue retries, low-storage protection, and conflict quarantine:

```yaml
schema_version: "1.0.0"
fixture_id: "fix_syn_local_draft_recovery_v1"

scenarios:
  # Scenario 1: Tab Crash & Complete Journal Recovery
  - scenario_id: "scen_rec_tab_crash_01"
    tenant_id: "ten_syn_01"
    execution_id: "ins_syn_clean_01"
    interruption_type: "BROWSER_TAB_CLOSE"
    journal_state:
      committed_entries: 14
      uncommitted_memory_entries: 0
    recovery_outcome: "100_PERCENT_RESTORED"
    recovered_items:
      questions_answered: 14
      evidence_photos_queued: 2
    ui_guidance: "DRAFT RESTORED: Resumed inspection from local journal."

  # Scenario 2: Network Interruption & Exponential Backoff Retry
  - scenario_id: "scen_rec_network_drop_02"
    tenant_id: "ten_syn_01"
    execution_id: "ins_syn_clean_01"
    interruption_type: "NETWORK_DROP_DURING_SYNC"
    item_id: "evd_syn_response_photo_01"
    initial_sync_state: "SYNCING"
    post_drop_sync_state: "SYNC_FAILED_RETRYABLE"
    retry_schedule:
      attempt_1_delay_ms: 1000
      attempt_2_delay_ms: 2000
      attempt_3_delay_ms: 4000
    outcome: "SAFE_PRESERVATION_AWAITING_ONLINE"

  # Scenario 3: Low-Storage Threshold & Fail-Closed Photo Lock
  - scenario_id: "scen_rec_low_storage_03"
    tenant_id: "ten_syn_01"
    execution_id: "ins_syn_clean_01"
    storage_state:
      used_bytes: 48500000       # 48.5 MB
      quota_bytes: 50000000      # 50.0 MB (97% utilization)
    attempted_action: "CAPTURE_ATTACHMENT"
    expected_result: "FAIL_CLOSED_LOCK_PHOTO"
    error_code: "DENIAL_STORAGE_QUOTA_EXCEEDED"
    expected_error: "ErrLocalStorageQuotaExceeded"
    text_responses_allowed: true
    drafts_purged: false

  # Scenario 4: Idempotent Duplicate Mutation Submission
  - scenario_id: "scen_rec_idempotent_retry_04"
    tenant_id: "ten_syn_01"
    execution_id: "ins_syn_clean_01"
    mutation_key: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
    submission_count: 2
    server_action: "ACKNOWLEDGE_EXISTING_RECORD"
    duplicate_created: false
    final_sync_state: "SYNC_ACKNOWLEDGED"

  # Scenario 5: Concurrency Conflict Quarantine
  - scenario_id: "scen_rec_conflict_quarantine_05"
    tenant_id: "ten_syn_01"
    execution_id: "ins_syn_reassigned_02"
    submitted_base_version: 1
    current_server_version: 2
    server_response: "HTTP_409_CONFLICT"
    client_sync_state: "SYNC_QUARANTINED"
    data_discarded: false
    diagnostic_event: "conflict.quarantined"
```

---

## 9. Governance Boundaries, Prohibitions & Operational Non-Claims

In strict compliance with `HDEC-V040-FOUNDATION-054` and `ASN-V040-I014-LOCAL-DRAFT-RECOVERY-001`:

1. **Explicit Retention of Issue #128 Open:** This specification implements local draft, queue, and recovery models. Downstream multi-part chunked transfer, background media compression, and final integration remain open under Issue #128 (`V040-I017`). Issue #128 must **NOT** be closed by this delivery.
2. **100% Synthetic Data Policy (`H040-003`):** All test fixtures, IDs, responses, and image metadata are synthetic local artifacts. Zero real inspection photographs, workforce PII, or physical facility data are permitted.
3. **No External Route or Cloud Bucket Activation (`H040-007` & `H040-010` HOLD):** Zero external S3/Blob endpoints, presigned URLs, or mobile device management (MDM) integrations are authorized or activated.
4. **No Real Participant Onboarding (`H040-008` HOLD):** Zero real field inspectors or mobile test cohorts are onboarded.
5. **Specification-Only Credit:** Delivery of this baseline confers documentation and architectural baseline credit only; zero operational release, deployment, or residual-risk acceptance is claimed.
