---
document_id: QLF-V040-OFFLINE-001
title: v0.4.0 OSHE Inspect Offline, Synchronization, Interruption, Device, Storage, Authorization, Conflict, and Recovery Qualification Baseline
document_type: qualification_specification
document_version: 1.0.0
lifecycle_status: DRAFT
status: APPROVED_FOR_LOCAL_DEVELOPMENT
date: "2026-09-05"
author_role: Architecture and Data Lead
author_pane: w9:p22
governing_issue: "GitHub Issue #127"
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

# v0.4.0 OSHE Inspect Offline, Synchronization, Interruption, Device, Storage, Authorization, Conflict, and Recovery Qualification Baseline

## 1. Executive Summary & Governance Authority

### 1.1 Authority Baseline & Purpose
This qualification specification establishes the authoritative, deterministic **Offline, Synchronization, Interruption, Device, Storage, Authorization, Conflict, and Recovery Technical Qualification Baseline** for **Milestone v0.4.0 - OSHE Inspect Private Alpha** within `OSHEThai/oshe-platform`. It fulfills the qualification scope and acceptance criteria of **GitHub Issue #127 (`[V040-I016] Qualify Offline, Synchronization, Interruption, Device, Storage, Authorization, Conflict, and Recovery Behavior`)** under Roadmap Topic `V040-T03` and standing owner foundation decision `HDEC-V040-FOUNDATION-054`.

The primary objective is to define an integrated, dependency-free, deterministic verification harness covering eight essential offline operational dimensions:
1. Supported client device, browser engine, and responsive viewport boundaries (`H040-002`).
2. Reference-data freshness ceilings, cryptographic packages, and 24-hour offline lease expiration.
3. Offline scope containment and default-deny authorization enforcement (`H040-004`).
4. Protected local storage partitioning (`IndexedDB`), atomic write-ahead journaling, and low-storage defenses (`H040-003`).
5. Abrupt interruption, process crash, and device restart recovery (`NFR-REC-01`).
6. Synchronization state machine transitions, exponential backoff retries, and cryptographic idempotency.
7. Conservative C0–C5 conflict classification, server authority, and non-destructive quarantine (`H040-005`).
8. Visible human reconciliation workflows and bilingual English/Thai diagnostic guidance.

### 1.2 Non-Substitution Invariant: Technical & Synthetic Scope Only
In strict compliance with `ASN-V040-I016-OFFLINE-QUALIFICATION-002` and `HDEC-V040-FOUNDATION-054`:
- **Synthetic Technical Qualification Only:** This baseline evaluates deterministic client state machines, encryption boundaries, local storage journaling, and conflict handling using local synthetic fixtures (`usr_*`, `ins_*`, `dev_*`, `fix_*`) and simulated clocks.
- **Non-Substitution Invariant:** Automated agent simulations, synthetic tests, and local test harnesses **cannot substitute for, replace, or claim the status of empirical real-user evidence or UAT**. Gate `H040-008` (Real Participant / Private-Alpha UAT Authorization) remains strictly on **`HOLD`** pending explicit owner screening and authorization.

### 1.3 Retained Unselected Policies & Non-Claims
1. **Binding Scoring Policy (`HUMAN_OWNED_UNSELECTED`):** Local offline score projections are advisory only; binding scoring policies remain human-owned under Issue #136 (`V040-I025`).
2. **Finding Closure Policy (`HUMAN_OWNED_UNSELECTED`):** Reinspections and finding closures are strictly server-authoritative operations and remain human-owned under Issue #134 (`V040-I023`).
3. **Offline Authority Model (`HUMAN_OWNED_UNSELECTED`):** Long-term statutory lease durations and automated precedence rules remain human-owned under Issue #126 (`V040-I015`).
4. **Retained Human Gate Holds:** Foundation gates `H040-007` (Technical Release Authorization), `H040-008` (Real Participant UAT), `H040-009` (Binding Support Ownership), `H040-010` (External Environment Activation), and `H040-011` (Final Milestone Outcome) remain strictly on **`HOLD`**.

---

## 2. Dimension 1: Supported Device & Viewport Boundary (`H040-002`)

Under `ARC-V040-MOBOFF-001`, client runtime evaluates device, browser, and viewport constraints:

### 2.1 Supported Platform Matrix
- **Desktop Web:** Google Chrome v128+ and Microsoft Edge v128+ on Windows 10/11.
- **Mobile Web:** Google Chrome Mobile v128+ on Android 12+.

### 2.2 Unsupported Platform Rejection (Fail-Closed)
The following environments are strictly rejected upon initialization with `DENIAL_UNSUPPORTED_DEVICE`:
- Apple iOS / iPadOS (Safari, Chrome iOS, WebKit wrappers).
- Mozilla Firefox (desktop and mobile Gecko).
- Embedded container webviews (LINE, Facebook, WeChat in-app browsers).
- Native application binaries (zero APK or IPA builds exist).

### 2.3 Viewport Rules
- Desktop Viewport: minimum $1280 \times 720\text{ px}$.
- Mobile Viewport: minimum $360 \times 640\text{ px}$.
- Displays below minimum dimensions engage visible fallback banners advising display adjustment.

---

## 3. Dimension 2: Reference Data Freshness & 24-Hour Lease Expiration

In accordance with `ARC-V040-MOBOFF-001`:

1. **Cryptographic Work Packages:** Downloaded inspection packages contain immutable template definitions, question applicability rules, and package signature digests (`pkg_digest`).
2. **24-Hour Maximum Offline Lease Ceiling:**
   - Every package embeds `downloaded_at` and `lease_expires_at` (strictly $\le 24\text{ hours}$ from download).
3. **Lease Expiration Enforcement (Fail-Closed):**
   - If client elapsed time exceeds `lease_expires_at`:
     - Active editing is immediately locked.
     - New response input is rejected with `ERR_OFFLINE_LEASE_EXPIRED`.
     - Inspector is prompted with bilingual instructions to re-authenticate and revalidate the assignment online.

---

## 4. Dimension 3: Offline Scope & Authorization Enforcement (`H040-004`)

Under default-deny authority gate `H040-004`:

1. **Strict Assignment Scoping:** Offline inspectors are authorized strictly for specifically assigned inspection IDs (`ins_syn_*`) and their exact bounded site/area context.
2. **Sibling Project Isolation:** Attempting to query or create records for unassigned sites, areas, or sister projects fails closed.
3. **Revoked Authority at Synchronization (C4):** If an inspector's role or project assignment is revoked or modified on the server while the device is offline, incoming submissions are rejected with `ERR_REVOKED_AUTHORITY_AT_SYNC` and quarantined.

---

## 5. Dimension 4: Protected Local Storage Partitioning (`IndexedDB`) & Storage Defenses

Under `ARC-V040-RECOV-001` and data privacy gate `H040-003`:

1. **Partitioned Local Storage (`oshe_offline_db`):**
   - `draft_responses`: Keyed by `inspection_id + ":" + question_id`.
   - `media_queue`: Stores binary media blobs, MIME types, and SHA-256 digests.
   - `sync_journal`: Write-ahead transaction journal tracking dirty state.
2. **Low-Storage Defenses & Quota Protection:**
   - Client checks `navigator.storage.estimate()` prior to capturing media.
   - If remaining browser quota is below 15%, media capture is suspended with `ERR_STORAGE_QUOTA_CRITICAL`.
   - Existing draft text responses are never pruned or purged automatically.

---

## 6. Dimension 5: Abrupt Interruption & Crash Recovery (`NFR-REC-01`)

In accordance with `NFR-REC-01`:

1. **Atomic Write-Ahead Journaling:** Every question response, note edit, and media attachment is committed to local `IndexedDB` before UI transition acknowledgement.
2. **Crash & Power-Loss Resilience:**
   - Abrupt browser process kill, device battery depletion, or tab termination leaves local storage in a consistent state.
   - On application restart, the sync journal re-hydrates un-synced draft responses and indicates pending queue depth.
3. **Zero In-Flight Data Loss:** All captured field responses remain preserved locally until affirmative server synchronization acknowledgement.

---

## 7. Dimension 6: Synchronization State Machine & Exponential Backoff Retries

Client synchronization operates as a deterministic finite state machine:

```
[DRAFT_LOCAL] ──(Network Restored)──> [SYNC_QUEUED] ──(Dispatch)──> [SYNC_IN_PROGRESS]
                                                                           │
               ┌───────────────────────┬───────────────────────────────────┤
               ▼                       ▼                                   ▼
      [SYNC_ACKNOWLEDGED]   [SYNC_FAILED_RETRYING]                [SYNC_QUARANTINED]
         (HTTP 200 OK)      (Network / 5xx Error)                 (HTTP 409 / Conflict)
```

1. **Cryptographic Idempotency Keys:** Every sync batch carries an idempotency key (`idem_syn_*`) derived from transaction content. Re-transmitted batches are recognized and acknowledged without duplicate execution (C0).
2. **Exponential Backoff with Jitter:** On transient network failures (e.g. timeout, connection reset, HTTP 503), client retries with bounded exponential intervals: $1\text{s}, 2\text{s}, 4\text{s}, 8\text{s}, 16\text{s}, \text{max } 60\text{s}$, combined with pseudorandom jitter ($\pm 20\%$) to avoid server thundering herds.

---

## 8. Dimension 7: C0–C5 Conflict Classification & Quarantine Verification (`H040-005`)

In strict conformance with `ARC-V040-CONFLICT-001`:

1. **Server Authority Invariant:** The central server is the absolute source of truth for all protected states.
2. **Zero Last-Write-Wins (LWW):** Timestamp-based client overwrites are categorically prohibited.
3. **Deterministic Conflict Classes:**
   - **`C0` (Idempotent Duplicate):** Acknowledges existing state (`ACK_IDEMPOTENT_DUPLICATE`, HTTP 200).
   - **`C1` (Additive Disjoint Merge):** Auto-merges disjoint question responses (`ACK_MERGED_ADDITIVE`, HTTP 200).
   - **`C2` (Stale Base-Version Edit):** Rejects direct overwrite; quarantines client draft (`ERR_CONFLICT_STALE_BASE`, HTTP 409).
   - **`C3` (Competing Workflow State Transition):** Rejects client edits against completed/closed records (`ERR_PROTECTED_STATE_IMMUTABLE`, HTTP 409).
   - **`C4` (Revoked Authority at Sync):** Rejects submissions from former inspectors (`ERR_REVOKED_AUTHORITY_AT_SYNC`, HTTP 403).
   - **`C5` (Cryptographic Integrity Uncertainty):** Rejects corrupted or mismatched digests (`ERR_INTEGRITY_UNCERTAINTY`, HTTP 422).
4. **Non-Destructive Quarantine:** Conflicting client payloads are preserved in `ConflictRecord` without data loss.

---

## 9. Dimension 8: Visible Human Reconciliation & User Guidance

Under `H040-005` and `ARC-V040-CONFLICT-001`:

1. **Independent Reviewer Work Queue:** Quarantined conflicts stage a prioritized reconciliation packet in the Independent Reviewer's interface.
2. **Four Authorized Resolution Actions:**
   - `RESOLVE_ACCEPT_SERVER`: Confirms server record; client draft preserved in audit logs.
   - `RESOLVE_OVERWRITE_WITH_CLIENT`: Reviewer explicitly adopts client observations as new server version ($N+2$).
   - `RESOLVE_MANUAL_MERGE`: Reviewer harmonizes individual fields.
   - `RESOLVE_SPLIT_NEW_FINDING`: Reviewer accepts observations by creating linked child finding.
3. **Bilingual Diagnostic Guidance (`th-TH`, `en-US`):**
   - Offline, syncing, conflict, and quarantine statuses display localized, user-friendly status banners in Thai and English.

---

## 10. Synthetic Qualification Scenarios Fixture

The following synthetic YAML fixture specifies the complete qualification scenario matrix:

```yaml
schema_version: "1.0.0"
fixture_id: "fix_syn_offline_qualification_v1"

scenarios:
  - id: "QLF-OFF-01"
    name: "Supported Device & Viewport Gate"
    client:
      os: "Android 13"
      browser: "Chrome Mobile 128"
      viewport: "412x915"
    expected_result: "PASS_SUPPORTED"

  - id: "QLF-OFF-02"
    name: "Unsupported Platform Rejection"
    client:
      os: "iOS 17"
      browser: "Safari 17"
    expected_result: "FAIL_DENIAL_UNSUPPORTED_DEVICE"

  - id: "QLF-OFF-03"
    name: "24-Hour Lease Expiration Enforcement"
    lease_duration_hours: 24
    simulated_elapsed_hours: 25
    expected_result: "LOCKED_ERR_OFFLINE_LEASE_EXPIRED"

  - id: "QLF-OFF-04"
    name: "Low-Storage Defense"
    remaining_storage_percent: 10
    action: "CAPTURE_MEDIA"
    expected_result: "REJECT_ERR_STORAGE_QUOTA_CRITICAL"

  - id: "QLF-OFF-05"
    name: "Crash Recovery"
    action: "PROCESS_KILL_DURING_EDIT"
    expected_result: "RECOVERED_FROM_JOURNAL"

  - id: "QLF-OFF-06"
    name: "Idempotent Re-transmission (C0)"
    action: "RESEND_IDENTICAL_BATCH"
    expected_result: "ACK_IDEMPOTENT_DUPLICATE_200"

  - id: "QLF-OFF-07"
    name: "Stale Version Quarantine (C2)"
    server_version: 3
    client_base_version: 1
    expected_result: "QUARANTINED_ERR_CONFLICT_STALE_BASE_409"

  - id: "QLF-OFF-08"
    name: "Protected State Rejection (C3)"
    server_state: "FINALIZED"
    client_action: "SUBMIT_RESPONSES"
    expected_result: "QUARANTINED_ERR_PROTECTED_STATE_IMMUTABLE_409"
```

---

## 11. Governance Boundaries & Non-Claims

In strict adherence to `HDEC-V040-FOUNDATION-054`:

1. **Synthetic Technical Evidence Only:** All qualification scenarios operate strictly against synthetic fixtures (`usr_*`, `ins_*`, `dev_*`). Zero real workforce data or actual customer site records are processed.
2. **Zero Real-User UAT Claim:** This technical baseline does **NOT** constitute or substitute for empirical real-user evidence. Gate `H040-008` remains on strict **HOLD**.
3. **Zero Deployment or Release Claim:** Gates `H040-007` through `H040-011` remain on **HOLD**. Zero live synchronization gateways, cloud storage endpoints, or public routes are authorized or deployed.
