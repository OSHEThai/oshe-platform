---
document_id: ARC-V040-EVD-002
title: v0.4.0 OSHE Inspect Evidence Capture, Metadata, Local Queue, Preview, Caption, Source Context, Record Association, and Denial Baseline
document_type: architecture_specification
document_version: 1.0.0
lifecycle_status: DRAFT
status: APPROVED_FOR_LOCAL_DEVELOPMENT
date: "2026-09-05"
author_role: Security Privacy and Product Safety Lead
author_pane: w9:p13
governing_issue: "GitHub Issue #128"
authority_source: HDEC-V040-FOUNDATION-054
governing_decisions:
  - HDEC-V040-FOUNDATION-054
  - HDEC-AGENT-CONTINUITY-011
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
  - "ARC-V040-EVD-001 (docs/architecture/v040-evidence-capture-prework.md)"
  - "ARC-V040-RECOV-001 (docs/architecture/v040-local-draft-recovery-baseline.md)"
completed_integrations:
  - V040_I014_INTEGRATION_COMPLETED
retained_unselected_policies:
  binding_scoring_policy: HUMAN_OWNED_UNSELECTED
  finding_closure_policy: HUMAN_OWNED_UNSELECTED
  offline_authority: HUMAN_OWNED_UNSELECTED
  evidence_retention_policy: HUMAN_OWNED_UNSELECTED
  external_storage_provider_policy: HUMAN_OWNED_UNSELECTED
credit_boundary: EVIDENCE_CAPTURE_BASELINE_SPECIFICATION_ONLY_NO_EXTERNAL_DELIVERY_OR_LIVE_OPERATIONAL_ENACTMENT
---

# v0.4.0 OSHE Inspect Evidence Capture, Metadata, Local Queue, Preview, Caption, Source Context, Record Association, and Denial Baseline

## 1. Executive Summary & Governance Posture

### 1.1 Authority Baseline & Purpose
This architectural specification establishes the authoritative **Evidence Capture, Metadata Minimization, Local Queue, Preview, Caption, Source Context, Record Association, and Fail-Closed Denial Baseline** for **Milestone v0.4.0 - OSHE Inspect Private Alpha** within `OSHEThai/oshe-platform`. It fulfills the deliverable requirements of **GitHub Issue #128 (`[V040-I017] Implement Evidence Capture, Metadata, Local Queue, Preview, Caption, Source Context, and Record Association`)** under Roadmap Topic `V040-T04`, standing owner foundation decision `HDEC-V040-FOUNDATION-054`, and agent continuity authority `HDEC-AGENT-CONTINUITY-011`.

The primary objective is to define a conservative, fail-closed evidence capture and association model within the **Files and Evidence Module (`MOD-EVD`)**, specifying:
- Strict evidence binding to parent domain entities (checklist response, safety finding, or CAPA action item).
- Privacy-preserving metadata minimization and mandatory EXIF/geotag stripping (`H040-003`).
- Offline-capable local client queueing integrated with the I014 local recovery architecture (`ARC-V040-RECOV-001`).
- Transient, secure preview rendering with role-based access control.
- Bounded supported media formats and strict 10 MiB size thresholds.
- Comprehensive fail-closed denial behavior and audit logging.

### 1.2 Resolution of I014 Dependency: `V040_I014_INTEGRATION_COMPLETED`
This specification formally integrates the local draft, interruption recovery, and synchronization state architecture established under **GitHub Issue #125 / `V040-I014` (`ARC-V040-RECOV-001`)**. The prior prework marker `PENDING_DEPENDENCY_V040_I014` is resolved by incorporating the `oshe_offline_db` persistence model, write-ahead mutation journaling, deterministic synchronization state transitions, and low-storage defensive quotas into the evidence capture lifecycle.

### 1.3 Retained Unselected Policies & Non-Claims
1. **Evidence Retention Policy (`HUMAN_OWNED_UNSELECTED`):** Permanent retention duration, legal archival hold, and automated disposition schedules require explicit owner approval under `V040-T04`.
2. **External Storage Provider Policy (`HUMAN_OWNED_UNSELECTED`):** Direct cloud blob storage (AWS S3, Azure Blob, Cloudflare R2) is strictly unselected; the alpha operates exclusively against synthetic local in-memory storage adapters.
3. **No Real or Production Media:** Live photographs, actual physical plant blueprints, worker biometric scans, or customer documents are strictly forbidden (`H040-003`). All media fixtures must use synthetic or redacted test assets.
4. **Retained Human Gate Holds:** Foundation gates `H040-007` (Technical Release Authorization), `H040-008` (Real Participant / Private-Alpha UAT Onboarding), `H040-009` (Binding Support Ownership), `H040-010` (External Environment/Route Activation), and `H040-011` (Final Milestone Outcome) remain strictly on **`HOLD`**.

---

## 2. Evidence Entity Model & Explicit Association Topology

Under `MOD-EVD`, evidence files cannot exist as detached, orphan blobs. Every evidence record must be tightly bound to an authoritative parent domain entity within `MOD-WFA`.

### 2.1 Entity Model Specification
```
┌────────────────────────────────────────────────────────────────────────┐
│                          EvidenceAttachment                            │
│  - evidence_id: String (evd_syn_[0-9a-f]{16})                          │
│  - tenant_id: String (ten_syn_*)                                       │
│  - association_type: Enum (INSPECTION_RESPONSE, FINDING, CAPA_ACTION)  │
│  - association_id: String (rsp_syn_*, fnd_syn_*, act_syn_*)            │
│  - execution_id: String (ins_syn_*)                                    │
│  - original_filename: String (sanitized basename, no traversal)        │
│  - media_type: Enum (image/jpeg, image/png, image/webp, pdf)           │
│  - size_bytes: Int64 (max 10 MiB)                                      │
│  - sha256_digest: String (64-character lowercase hex digest)           │
│  - caption: LocalizedString (en-US, th-TH, max 250 chars)              │
│  - source_context: Enum (CAMERA_DIRECT, PHOTO_GALLERY, FILE_SYSTEM)    │
│  - capture_metadata: SanitizedMetadata (width, height, timestamp)      │
│  - storage_ref: String (content-addressed key: evd/{tenant}/{digest})   │
│  - state: Enum (DRAFT_LOCAL, QUEUED_FOR_SYNC, SYNCING, COMMITTED)      │
│  - created_at: Timestamp (UTC ISO 8601)                                │
│  - created_by: String (usr_syn_inspector_*)                            │
└────────────────────────────────────────────────────────────────────────┘
```

### 2.2 Association Targets & Validation Invariants
1. **`INSPECTION_RESPONSE`:** Evidence supporting an item response in an active inspection (e.g. photo verifying fire extinguisher pressure gauge).
2. **`SAFETY_FINDING`:** Mandatory evidence documenting an identified non-conformance or hazard (e.g. photo of blocked emergency exit).
3. **`CAPA_ACTION`:** Verification evidence submitted by a CAPA Owner documenting remediation (e.g. photo of cleared exit).
4. **Prohibition of Orphan Evidence:** Any upload or capture request lacking an authenticated, active `association_id` fails closed with `DENIAL_MISSING_ASSOCIATION`.
5. **Tenant Scope Invariant:** The parent entity's `tenant_id` must strictly match the caller's active tenant context (`ErrTenantMismatch` / `DENIAL_WRONG_SCOPE`).

---

## 3. Privacy Impact Minimization & EXIF Scrubbing (`H040-003`)

In strict accordance with `H040-003` (100% Synthetic Data & Privacy Boundary), evidence capture enforces rigorous data minimization:

### 3.1 Client-Side Metadata Scrubbing
Before writing to local storage or calculating the authoritative SHA-256 digest:
1. **Mandatory EXIF Stripping:** All Exchangeable Image File Format (EXIF) metadata is programmatically scrubbed from captured JPEG/PNG/WebP images.
   - Scrubbed fields: GPS latitude/longitude/altitude, camera serial number, user comments, lens metadata, and device hardware UUIDs.
2. **Sanitized Allowed Technical Metadata:** Only structural display attributes are preserved:
   - `pixel_width`: Positive integer.
   - `pixel_height`: Positive integer.
   - `file_format`: Standard MIME string.
   - `sanitized_filename`: Whitelisted alphanumeric string with single extension.
3. **Caption & Context Restrictions:** Captions are limited to 250 characters and strictly filtered to bar credit card numbers, national citizen IDs, or personal telephone numbers.

---

## 4. Supported Media Formats & Size Constraints

In alignment with `modules/files-evidence/file_metadata.go`, the private alpha restricts acceptable attachments to proven, safe web formats:

| Format Category | MIME Type | File Extension | Alpha Status | Maximum Size |
| :--- | :--- | :--- | :--- | :--- |
| **Raster Image** | `image/jpeg` | `.jpg`, `.jpeg` | `SUPPORTED_ALPHA` | 10 MiB |
| **Raster Image** | `image/png` | `.png` | `SUPPORTED_ALPHA` | 10 MiB |
| **Modern Image** | `image/webp` | `.webp` | `SUPPORTED_ALPHA` | 10 MiB |
| **Document** | `application/pdf` | `.pdf` | `SUPPORTED_ALPHA` | 10 MiB |
| **Video / Audio** | `video/*`, `audio/*` | `.mp4`, `.mp3` | **`UNSUPPORTED_PRIVATE_ALPHA`** | 0 B (Rejected) |
| **Executables** | `application/x-sh`, `.exe` | `.sh`, `.exe` | **`PROHIBITED_SECURITY_RISK`** | 0 B (Rejected) |

---

## 5. Client-Side Local Evidence Queue & I014 Synchronization Integration

### 5.1 Local Queueing Model (`IndexedDB` & `ARC-V040-RECOV-001`)
Under intermittent or offline connectivity (`ARC-V040-MOBOFF-001` and `ARC-V040-RECOV-001`):
1. **Atomic Write-Ahead Queue:** When an inspector captures a photo, the scrubbed binary blob is written to `IndexedDB` (`evidence_blobs`), and the associated metadata record is queued in `evidence_queue`.
2. **Immediate SHA-256 Computation:** The client runtime computes the cryptographic digest over the scrubbed bytes immediately upon capture, ensuring immutability before transmission.
3. **Queue Capacity Limits:** A single mobile client may queue at most **10 attachments** or **50 MB** total evidence payload to prevent mobile device memory exhaustion.
4. **Mutation Journal Integration:** Every capture and association event writes an entry to `mutation_journal` with a monotonic sequence number, guaranteeing crash recovery without draft loss.

### 5.2 Synchronization State Machine Integration
Locally queued evidence records transition through the standardized I014 synchronization states:
- `DRAFT_LOCAL`: Captured and stored in local `evidence_blobs` and `evidence_queue`; editable on client.
- `QUEUED_FOR_SYNC`: Locked for dispatch during manual or automated sync batching.
- `SYNCING`: In transit via authenticated HTTP transport.
- `SYNC_ACKNOWLEDGED`: Central server confirmed receipt and committed SHA-256 digest.
- `SYNC_FAILED_RETRYABLE`: Transient failure; scheduled for exponential backoff retry.
- `SYNC_QUARANTINED`: Integrity or association conflict; isolated for supervisory review.

### 5.3 Low-Storage Defenses & Defensive Quotas
1. **Threshold Monitoring:** The client queries `navigator.storage.estimate` before accepting new media attachments.
2. **Critical Ceiling ($\ge 95\%$ quota):** New captures are locked fail-closed (`ErrLocalStorageQuotaExceeded` / `DENIAL_STORAGE_EXCEEDED`).
3. **Zero Silent Eviction:** Under no circumstances does the client prune or delete un-synced evidence blobs or response drafts to free space.

---

## 6. Ephemeral Secure Preview & Access Control

### 6.1 Preview Lifecycle
1. **Blob URL Lifecycle:** Previews in the inspection UI are rendered via transient `URL.createObjectURL(blob)`.
2. **Deterministic Cleanup:** Preview object URLs are revoked immediately upon modal dismiss, item transition, or component unmount to prevent memory leaks.

### 6.2 Access Control
1. **Inspector Session Binding:** Previews are accessible only to the active inspector assigned to the inspection execution.
2. **Fail-Closed Access Denial:** Requests for preview from unauthenticated callers, cross-tenant sessions, or unauthorized roles are denied fail-closed with `DENIAL_PREVIEW_UNAUTHORIZED` (`ErrUnauthorizedPreviewAccess`).

---

## 7. Fail-Closed Denial Reasons Matrix

The evidence capture engine enforces deterministic fail-closed validation across all touchpoints:

| Denial Code | Error Identifier | Validation Trigger |
| :--- | :--- | :--- |
| **`DENIAL_UNSUPPORTED_MEDIA_TYPE`** | `ErrInvalidMediaType` | Uploading unsupported MIME type (e.g. video, executable, audio). |
| **`DENIAL_SIZE_EXCEEDED`** | `ErrInvalidSize` | Payload size exceeds 10 MiB threshold. |
| **`DENIAL_MISSING_ASSOCIATION`** | `ErrMissingAssociationTarget` | Attempting capture without specifying valid response, finding, or action. |
| **`DENIAL_WRONG_SCOPE`** | `ErrTenantMismatch` | Association ID belongs to a different tenant or inaccessible site. |
| **`DENIAL_METADATA_VIOLATION`** | `ErrInvalidFilename` | Filename contains path traversal (`../`), null bytes, or unstripped PII. |
| **`DENIAL_DIGEST_MISMATCH`** | `ErrIntegrityMismatch` | Byte stream SHA-256 hash does not match declared digest. |
| **`DENIAL_PREVIEW_UNAUTHORIZED`** | `ErrUnauthorizedPreviewAccess` | Requesting preview without active session or valid assignment. |
| **`DENIAL_STORAGE_EXCEEDED`** | `ErrLocalStorageQuotaExceeded` | Local device storage exceeds 95% quota ceiling. |

---

## 8. Synthetic Multi-Scenario Fixture Matrix

The following synthetic YAML fixture illustrates valid multi-target evidence attachments and denial scenarios:

```yaml
schema_version: "1.0.0"
fixture_id: "fix_syn_evidence_capture_v1"

attachments:
  # Scenario 1: Valid Checklist Response Evidence
  - evidence_id: "evd_syn_response_photo_01"
    tenant_id: "ten_syn_01"
    execution_id: "ins_syn_clean_01"
    association_type: "INSPECTION_RESPONSE"
    association_id: "rsp_syn_extinguisher_press_01"
    original_filename: "gauge_reading_normal.jpg"
    media_type: "image/jpeg"
    size_bytes: 2048500
    sha256_digest: "a1b2c3d4e5f60718293a4b5c6d7e8f90123456789abcdef0123456789abcdef0"
    caption:
      en-US: "Pressure gauge reads within optimal green zone (150 psi)."
      th-TH: "เกจวัดแรงดันอยู่ในแถบสีเขียวปกติ (150 psi)"
    source_context: "CAMERA_DIRECT"
    state: "COMMITTED"

  # Scenario 2: Safety Finding Non-Conformance Evidence
  - evidence_id: "evd_syn_finding_photo_02"
    tenant_id: "ten_syn_01"
    execution_id: "ins_syn_clean_01"
    association_type: "SAFETY_FINDING"
    association_id: "fnd_syn_blocked_exit_01"
    original_filename: "blocked_emergency_exit_hallway.png"
    media_type: "image/png"
    size_bytes: 4194304
    sha256_digest: "b2c3d4e5f60718293a4b5c6d7e8f90123456789abcdef0123456789abcdef01"
    caption:
      en-US: "Wooden pallets obstructing fire exit corridor door B2."
      th-TH: "พาเลทไม้วางกีดขวางประตูทางออกฉุกเฉิน B2"
    source_context: "CAMERA_DIRECT"
    state: "COMMITTED"

  # Scenario 3: CAPA Remediation Verification Evidence
  - evidence_id: "evd_syn_capa_photo_03"
    tenant_id: "ten_syn_01"
    execution_id: "ins_syn_clean_01"
    association_type: "CAPA_ACTION"
    association_id: "act_syn_clear_pallets_01"
    original_filename: "cleared_corridor_remediation.webp"
    media_type: "image/webp"
    size_bytes: 1572864
    sha256_digest: "c3d4e5f60718293a4b5c6d7e8f90123456789abcdef0123456789abcdef012"
    caption:
      en-US: "Corridor fully cleared; exit route verified unobstructed."
      th-TH: "ทางเดินโล่งเรียบร้อย ตรวจสอบแล้วไม่มีสิ่งกีดขวาง"
    source_context: "PHOTO_GALLERY"
    state: "COMMITTED"

denial_scenarios:
  - scenario_id: "den_evd_01_unsupported_video"
    media_type: "video/mp4"
    denial_code: "DENIAL_UNSUPPORTED_MEDIA_TYPE"
    expected_error: "ErrInvalidMediaType"

  - scenario_id: "den_evd_02_oversized"
    size_bytes: 15728640  # 15 MiB (exceeds 10 MiB)
    denial_code: "DENIAL_SIZE_EXCEEDED"
    expected_error: "ErrInvalidSize"

  - scenario_id: "den_evd_03_missing_association"
    association_id: ""
    denial_code: "DENIAL_MISSING_ASSOCIATION"
    expected_error: "ErrMissingAssociationTarget"

  - scenario_id: "den_evd_04_path_traversal"
    original_filename: "../../etc/passwd.jpg"
    denial_code: "DENIAL_METADATA_VIOLATION"
    expected_error: "ErrInvalidFilename"

  - scenario_id: "den_evd_05_unauthorized_preview"
    caller_role: "UNAUTHORIZED_ANONYMOUS"
    denial_code: "DENIAL_PREVIEW_UNAUTHORIZED"
    expected_error: "ErrUnauthorizedPreviewAccess"

  - scenario_id: "den_evd_06_storage_exhaustion"
    storage_utilization_pct: 96
    denial_code: "DENIAL_STORAGE_EXCEEDED"
    expected_error: "ErrLocalStorageQuotaExceeded"
```

---

## 9. Completion of I014 Integration & Dependency Resolution

1. **Resolution of `PENDING_DEPENDENCY_V040_I014`:** The prior dependency recorded in `ARC-V040-EVD-001` is fulfilled through alignment with `ARC-V040-RECOV-001`. The evidence queue utilizes the local IndexedDB stores (`evidence_queue`, `evidence_blobs`), append-only mutation journal, and retryable synchronization mechanics.
2. **Unselected Human Policies Retained:**
   - `evidence_retention_policy: HUMAN_OWNED_UNSELECTED`
   - `external_storage_provider_policy: HUMAN_OWNED_UNSELECTED`
   - `binding_scoring_policy: HUMAN_OWNED_UNSELECTED`
   - `finding_closure_policy: HUMAN_OWNED_UNSELECTED`
   - `offline_authority: HUMAN_OWNED_UNSELECTED`

---

## 10. Governance Boundaries, Retained Holds & Non-Claims

In strict compliance with `HDEC-V040-FOUNDATION-054`, `HDEC-AGENT-CONTINUITY-011`, and `ASN-V040-I017-EVIDENCE-CAPTURE-INTEGRATION-007`:

1. **100% Synthetic Data Policy (`H040-003`):** All evidence identifiers, captions, and file records are synthetic fixtures. Zero real workforce images or operational documents are permitted.
2. **No External Route or Cloud Bucket Activation (`H040-007` & `H040-010` HOLD):** Zero external S3/Blob endpoints or direct presigned upload URLs are authorized.
3. **No Real Participant Onboarding (`H040-008` HOLD):** Zero real field inspectors or camera operators are onboarded.
4. **Issue Closure Prohibition:** Issue #128 remains open following this draft pull request until formal review and merge completion.
5. **Specification-Only Credit:** Delivery of this baseline confers architectural integration credit only; zero operational release, deployment, or residual-risk acceptance is claimed.
