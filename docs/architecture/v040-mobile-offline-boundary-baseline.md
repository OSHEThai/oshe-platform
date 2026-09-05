---
document_id: ARC-V040-MOBOFF-001
title: v0.4.0 OSHE Inspect Supported Mobile/Responsive Client Boundary, Download Eligibility, Protected Local Storage, Reference-Data Age, and Online-Only State Baseline
document_type: architecture_specification
document_version: 1.0.0
lifecycle_status: DRAFT
status: APPROVED_FOR_LOCAL_DEVELOPMENT
date: "2026-09-05"
author_role: Security Privacy and Product Safety Lead
author_pane: w9:p13
governing_issue: "GitHub Issue #124"
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
credit_boundary: MOBILE_OFFLINE_BOUNDARY_SPECIFICATION_ONLY_NO_EXTERNAL_DELIVERY_OR_LIVE_OPERATIONAL_ENACTMENT
---

# v0.4.0 OSHE Inspect Supported Mobile/Responsive Client Boundary, Download Eligibility, Protected Local Storage, Reference-Data Age, and Online-Only State Baseline

## 1. Executive Summary & Governance Posture

### 1.1 Authority Baseline & Purpose
This architectural specification establishes the authoritative **Supported Mobile/Responsive Client Boundary, Download Eligibility, Protected Local Storage, Reference-Data Age, and Online-Only State Baseline** for **Milestone v0.4.0 - OSHE Inspect Private Alpha** within `OSHEThai/oshe-platform`. It fulfills the deliverable requirements of **GitHub Issue #124 (`[V040-I013] Implement Supported Mobile/Responsive Client Boundary, Download Eligibility, Protected Local Storage, and Reference-Data Age`)** under Roadmap Topic `V040-T03` and standing owner foundation decision `HDEC-V040-FOUNDATION-054`.

The primary objective is to define a conservative, fail-closed mobile and offline operational boundary governing:
- Strict supported device and responsive viewport constraints (`H040-002`).
- Work-package download eligibility, packaging, and cryptographic sealing.
- Protected client-side local storage with data minimization and session zeroization (`H040-003`).
- Reference-data freshness ceilings, offline lease timeouts, and safe expiration.
- Explicit demarcation between offline-permissible data collection and online-only server-authoritative state transitions (`H040-004`, `H040-005`).

### 1.2 Retained Unselected Policies & Non-Claims
In strict compliance with `ASN-V040-I013-MOBILE-OFFLINE-BOUNDARY-001` and `HDEC-V040-FOUNDATION-054`:
1. **Binding Scoring Policy (`HUMAN_OWNED_UNSELECTED`):** Local offline score calculations are advisory projections only; final binding compliance scoring remains human-owned under Issue #136 (`V040-I025`).
2. **Finding Closure Policy (`HUMAN_OWNED_UNSELECTED`):** Reinspection verification and non-conformance closure are strictly online-only operations and remain human-owned under Issue #134 (`V040-I023`).
3. **Offline Authority Model (`HUMAN_OWNED_UNSELECTED`):** While this specification establishes conservative alpha timeouts (24 hours), long-term operational lease durations and reconciliation prioritization remain human-owned under Issue #126 (`V040-I015`).
4. **No Broad Device Support or Native Mobile App Claim:** Zero native mobile application binaries (APK / IPA), custom Android builds, iOS Safari, or Firefox support are authorized. Client support is strictly limited to responsive web on Chrome and Edge.
5. **Retained Human Gate Holds:** Foundation gates `H040-007` (Technical Release Authorization), `H040-008` (Real Participant / Private-Alpha UAT Onboarding), `H040-009` (Binding Support Ownership), `H040-010` (External Environment/Route Activation), and `H040-011` (Final Milestone Outcome) remain strictly on **`HOLD`**.

---

## 2. Supported Client Platform & Responsive Viewport Boundary (`H040-002`)

Under `ARC-V040-PROF-001` and `H040-002`, client access is strictly gated by browser engine, operating system, and responsive viewport profile.

### 2.1 Supported Platform Matrix
Client applications evaluate browser environment upon initialization. Access is granted **only** to the following three environments:

| Platform Classification | Operating System | Browser Engine | Version Baseline | Status | Input Modality |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **Desktop Web** | Windows 10 / 11 | Google Chrome (Blink) | Current Stable (v128+) | `SUPPORTED_ALPHA` | Keyboard, Mouse |
| **Desktop Web** | Windows 10 / 11 | Microsoft Edge (Blink) | Current Stable (v128+) | `SUPPORTED_ALPHA` | Keyboard, Mouse |
| **Mobile Web** | Android 12+ | Google Chrome Mobile (Blink) | Current Stable (v128+) | `SUPPORTED_ALPHA` | Multi-touch, Virtual Keyboard |

### 2.2 Unsupported Platforms (Fail-Closed Rejection)
The following platforms fail client evaluation with `DENIAL_UNSUPPORTED_DEVICE`:
- **Apple iOS / iPadOS:** All browsers (Safari, Chrome iOS, WebKit-based wrappers) are categorically **`UNSUPPORTED_PRIVATE_ALPHA`**.
- **Mozilla Firefox:** Desktop and mobile Gecko engines.
- **Legacy Browsers:** Internet Explorer, Microsoft Edge Legacy (EdgeHTML).
- **Embedded Webviews:** In-app browsers (e.g. LINE, Facebook, WeChat container webviews).
- **Native Applications:** Zero native Android APK or iOS IPA applications are authorized or built for v0.4.0.

### 2.3 Viewport & Responsive Layout Rules
- **Minimum Mobile Viewport:** $360 \times 640$ CSS pixels (portrait orientation).
- **Minimum Tablet Viewport:** $768 \times 1024$ CSS pixels (portrait or landscape).
- **Desktop Target Viewport:** $1280 \times 720$ up to $1920 \times 1080$ CSS pixels.
- **Display Scaling & Zoom:** Full layout preservation between 80% and 150% browser zoom without text overlap or button clipping.

---

## 3. Work-Package Download Eligibility & Assembly

An inspector cannot download arbitrary inspections. Work-package assembly is gated by the eligibility engine (`ARC-V040-ASGN-001`).

### 3.1 Download Eligibility Gates
Before assembling and transmitting a work package to a client device, the server verifies:
1. **Assignee Match:** Requesting authenticated user (`usr_syn_*`) matches `assigned_to` on the target execution.
2. **Execution State:** Execution status is `ASSIGNED` or `IN_PROGRESS`. Executions in `SCHEDULED`, `COMPLETED`, `CANCELLED`, or `REASSIGNED` cannot be downloaded.
3. **Temporal Due Window:** Current time is within the active due window (`Now \ge window_start` and `Now \le grace_until`).
4. **Active Role & Membership:** Inspector role is active and tenant membership is unexpired (`H040-001`).
5. **Supported Device Header:** Client user-agent and feature profile match `H040-002`.
6. **Checklist Freshness:** Bound checklist template is in `PUBLISHED_IMMUTABLE` state with matching cryptographic content digest.
7. **Reference Data Freshness:** Organizational and asset metadata has not exceeded maximum allowable cache age.

### 3.2 Work-Package Assembly Structure
A downloadable work package is a self-contained, immutable JSON bundle containing:

```
┌────────────────────────────────────────────────────────────────────────┐
│                   InspectionWorkPackage (wpk_syn_*)                    │
├────────────────────────────────────────────────────────────────────────┤
│  1. Package Header:                                                    │
│     - package_id: String (wpk_syn_[0-9a-f]{16})                        │
│     - tenant_id: String (ten_syn_*)                                    │
│     - execution_id: String (ins_syn_*)                                 │
│     - issued_at: Timestamp (UTC ISO 8601)                              │
│     - lease_expires_at: Timestamp (UTC ISO 8601, +24h max)             │
│     - package_digest: String (SHA-256 over payload)                    │
├────────────────────────────────────────────────────────────────────────┤
│  2. Bound Checklist Snapshot (Immutable):                              │
│     - template_id: String (chk_syn_*)                                  │
│     - template_version: String ("1.1.0")                               │
│     - content_digest: String (SHA-256)                                 │
│     - sections & questions (hierarchy, types, validation rules)        │
├────────────────────────────────────────────────────────────────────────┤
│  3. Geographic & Operational Scope:                                    │
│     - company_id, site_id, area_id, equipment_id                       │
│     - localized display names (en-US, th-TH)                           │
├────────────────────────────────────────────────────────────────────────┤
│  4. Localized Reference Data Bundle:                                   │
│     - schema_version: String ("1.0.0")                                 │
│     - reference_data_age: Timestamp (UTC ISO 8601)                     │
│     - translation_bundle: Key-value map (en-US, th-TH)                 │
├────────────────────────────────────────────────────────────────────────┤
│  5. Security & Concurrency Tokens:                                     │
│     - base_version: Integer (Monotonic state version)                  │
│     - assignee_subject: String (usr_syn_inspector_*)                   │
│     - client_lease_token: String (Cryptographic HMAC token)            │
└────────────────────────────────────────────────────────────────────────┘
```

---

## 4. Protected Local Storage Architecture & Data Minimization (`H040-003`)

### 4.1 Storage Partitioning & Technologies
To ensure data minimization and prevent browser storage leaks:
- **`IndexedDB` (`oshe_offline_db`):** Authoritative client-side persistence for active work packages, write-ahead mutation journals (`mutation_queue`), and media attachment drafts.
- **`SessionStorage`:** Transient holding for active session bearer tokens and ephemeral UI view state.
- **`LocalStorage` Prohibited:** `window.localStorage` is **strictly prohibited** from storing inspection records, customer identifiers, or session credentials.

### 4.2 Local Data Protection Rules
1. **Zero Raw Credentials (`H040-003`):** Passwords, API keys, and long-lived private keys are never stored on the client.
2. **Deterministic Session Purge:** Explicit user logout, tenant switch, or session expiration immediately triggers cryptographic zeroization and complete deletion of `oshe_offline_db` object stores.
3. **Storage Ceiling (`NFR-CAP-01`):** Client-side offline cache is limited to a conservative **50 MB** storage ceiling per device, sufficient for up to 5 concurrent offline work packages and compressed evidence references.

---

## 5. Reference-Data Age, Maximum Offline TTL, and Safe Expiry

### 5.1 Temporal Invariants
In the private alpha architecture, offline autonomy is bounded by strict temporal ceilings:
- **Maximum Reference-Data Age:** **24 Hours** (`MAX_REFERENCE_DATA_AGE_HOURS = 24`). Reference data (site names, user rosters, checklist schemas) older than 24 hours cannot be used to initialize new inspections.
- **Maximum Offline Work-Package Lease:** **24 Hours** (`MAX_OFFLINE_LEASE_HOURS = 24`). An inspector may operate offline for a maximum of 24 consecutive hours from package issuance.

### 5.2 Expiry State & Behavior (`OFFLINE_LEASE_EXPIRED`)
If an inspector device remains offline when `Now > lease_expires_at`:
1. **Fail-Closed Edit Lock:** The client application locks the inspection form against new response inputs or modifications.
2. **Visual Warning:** The UI displays a persistent amber banner: `"OFFLINE LEASE EXPIRED: Network reconnection required to validate work package freshness."`
3. **Draft Preservation Invariant:** Locally recorded responses and evidence references are **permanently preserved** in the IndexedDB journal. They are **never deleted or truncated upon expiration**.
4. **Reconnection Handshake:** Upon network restoration, the client transmits an atomic lease verification request to `MOD-WFA` before draining the mutation queue.

---

## 6. Online-Only Protected States Matrix

To protect data integrity, regulatory compliance, and life-safety authority, state machine transitions are strictly partitioned into offline-permissible vs. online-only operations:

| Workflow Operation | Domain Entity | Permissible Offline? | Mandatory Online Prerequisite | Reason & Governance Invariant |
| :--- | :--- | :--- | :--- | :--- |
| **Download Work Package** | `Assignment` | **No (Online Only)** | Central Eligibility Check | Validates active role, device, and fresh template version (`H040-001`). |
| **Start Inspection** | `InspectionExecution` | **Yes (Offline)** | Pre-downloaded Package | Recorded locally in IndexedDB journal; synced on reconnect. |
| **Record Checklist Responses** | `Response` | **Yes (Offline)** | Active Local Lease | Responses (`PASS`, `FAIL`, `UNKNOWN`, `NA`) queued locally. |
| **Capture Photo / Evidence** | `EvidenceRecord` | **Yes (Offline)** | Local SHA-256 Hashing | Image stored in client blob cache; hash computed immediately. |
| **Log Draft Finding** | `Finding` | **Yes (Offline)** | Active Local Lease | Draft finding recorded locally; critical warnings displayed. |
| **Submit / Finalize Inspection** | `InspectionExecution` | **No (Online Only)** | Server Authority Engine | Transition to `COMPLETED` requires server monotonic version commit (`H040-005`). |
| **Independent Review / Verification** | `InspectionExecution` | **No (Online Only)** | Reviewer Role Authority | Segregation of duties; Reviewer cannot operate offline (`SOD-03`). |
| **Finding / CAPA Closure** | `ActionItem` | **No (Online Only)** | Verified Evidence Audit | Finding closure policy is human-owned (`HUMAN_OWNED_UNSELECTED`). |
| **Reassign / Cancel Work** | `Assignment` | **No (Online Only)** | Supervisor Authority | Requires server lease revocation and sequence advance. |

---

## 7. Server Authority, Sync Reconciliation & Conflict Quarantine (`H040-005`)

### 7.1 Server Sole Authority Invariant
Under `H040-005`, the server is the single authoritative source of truth for all persistent state. Client devices submit candidate state mutation proposals accompanied by `(base_version, package_digest)`.

### 7.2 Rejection of Last-Write-Wins (LWW)
Timestamp-based last-write-wins (LWW) is **categorically prohibited**. If two devices submit mutations against the same execution, or if a mutation is submitted after an administrative reassignment:
1. The server detects that `incoming_base_version != current_server_version`.
2. The incoming mutation is rejected from the main state machine.

### 7.3 Conflict Quarantine Protocol (`QUARANTINED`)
1. Stale or conflicting client mutations are automatically partitioned into an immutable holding record (`ConflictRecord`).
2. Execution status is set to `QUARANTINED`.
3. An audit event (`conflict.quarantined`) is emitted to `MOD-REC`.
4. The inspection is locked for manual supervisory reconciliation, ensuring zero silent data overwrite or data loss.

---

## 8. Fail-Closed Denial Reasons Matrix

When an offline request or synchronization violates boundary rules, the engine returns a standardized denial reason:

| Denial Code | Trigger Condition | Error Identifier | Fail-Closed Behavior |
| :--- | :--- | :--- | :--- |
| **`DENIAL_UNSUPPORTED_DEVICE`** | Client browser is iOS Safari, Firefox, or container webview | `ErrUnsupportedDevicePlatform` | Download blocked; UI error modal presented. |
| **`DENIAL_EXPIRED_MEMBERSHIP`** | User tenant membership has lapsed | `ErrExpiredTenantMembership` | Download blocked; local cache zeroized. |
| **`DENIAL_WRONG_SCOPE`** | User scope does not cover inspection site/area | `ErrUnauthorizedAssignmentScope` | Download blocked; security violation logged. |
| **`DENIAL_STALE_CHECKLIST`** | Pinned checklist template is retired or digest mismatch | `ErrStaleChecklistTemplate` | Download blocked; template refresh required. |
| **`DENIAL_STALE_REFERENCE_DATA`** | Reference data age exceeds 24 hours | `ErrStaleReferenceData` | Download blocked until reference cache updated. |
| **`DENIAL_STORAGE_EXPIRY`** | Offline lease age exceeds 24 hours | `ErrOfflineLeaseExpired` | Local editing locked; sync requires server handshake. |
| **`DENIAL_ONLINE_ONLY_STATE`** | Attempted offline finalization or closure | `ErrOnlineOnlyStateTransition` | Mutation rejected; client instructed to reconnect. |

---

## 9. Synthetic Multi-Scenario Fixture Matrix

The following synthetic YAML fixture illustrates mobile/offline package issuance, expiry, and online-only transition boundaries:

```yaml
schema_version: "1.0.0"
fixture_id: "fix_syn_mobile_offline_packages_v1"

packages:
  # Scenario 1: Active Supported Mobile Package (Android Chrome)
  - package_id: "wpk_syn_mobile_active_01"
    tenant_id: "ten_syn_01"
    execution_id: "ins_syn_rayong_01"
    assignee_subject: "usr_syn_inspector_01"
    client_platform: "Android Chrome Mobile v128"
    status: "ACTIVE_OFFLINE"
    issued_at: "2026-09-10T01:00:00Z"
    lease_expires_at: "2026-09-11T01:00:00Z"
    reference_data_age: "2026-09-10T00:30:00Z"
    checklist_binding:
      template_id: "chk_syn_pilot_plant_safety_v1"
      version: "1.1.0"
      content_digest: "8f4e2b1c3a7d9e5f0b2a4c6d8e1f3a5b7c9d1e3f5a7b9c1d3e5f7a9b1c3d5e7f"
    limits:
      max_offline_hours: 24
      max_storage_mb: 50

  # Scenario 2: Expired Offline Lease Package (>24h disconnected)
  - package_id: "wpk_syn_mobile_expired_02"
    tenant_id: "ten_syn_01"
    execution_id: "ins_syn_rayong_02"
    assignee_subject: "usr_syn_inspector_02"
    client_platform: "Android Chrome Mobile v128"
    status: "OFFLINE_LEASE_EXPIRED"
    issued_at: "2026-09-08T01:00:00Z"
    lease_expires_at: "2026-09-09T01:00:00Z"
    evaluation_time: "2026-09-10T02:00:00Z"
    edit_locked: true
    draft_preserved: true

denial_scenarios:
  - scenario_id: "den_mob_01_ios"
    client_platform: "Apple iOS Safari v17"
    denial_code: "DENIAL_UNSUPPORTED_DEVICE"
    expected_error: "ErrUnsupportedDevicePlatform"

  - scenario_id: "den_mob_02_stale_ref"
    reference_data_age: "2026-09-08T00:00:00Z"
    current_time: "2026-09-10T01:00:00Z"
    denial_code: "DENIAL_STALE_REFERENCE_DATA"
    expected_error: "ErrStaleReferenceData"

  - scenario_id: "den_mob_03_online_only_finalize"
    operation: "FINALIZE_INSPECTION"
    is_online: false
    denial_code: "DENIAL_ONLINE_ONLY_STATE"
    expected_error: "ErrOnlineOnlyStateTransition"
```

---

## 10. Governance Boundaries, Prohibitions & Non-Claims

In strict compliance with `HDEC-V040-FOUNDATION-054` and `ASN-V040-I013-MOBILE-OFFLINE-BOUNDARY-001`:

1. **100% Synthetic Data Policy (`H040-003`):** All users, packages, sites, and templates are synthetic entities. Zero real employee records, site photos, or commercial telemetry are used.
2. **Server Authority Invariant (`H040-005`):** Offline devices hold delegated field capture authority only. All final state transitions remain server-authoritative.
3. **No External Route, Notification, or MDM Activation (`H040-007` & `H040-010` HOLD):** Zero external public endpoints, device management APIs, or cloud bucket connections are authorized.
4. **No Real Participant Onboarding (`H040-008` HOLD):** Zero real field inspectors or mobile test cohorts are onboarded.
5. **Specification-Only Credit:** Delivery of this baseline confers documentation and architectural model credit only; zero operational release, deployment, or residual-risk acceptance is claimed.
