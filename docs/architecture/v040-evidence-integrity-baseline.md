---
document_id: ARC-V040-EVD-003
title: v0.4.0 OSHE Inspect Immutable Originals, Derived Objects, Integrity Verification, Transfer, Upload, Export, and Chain of Custody Baseline
document_type: architecture_specification
document_version: 1.0.0
lifecycle_status: DRAFT
status: APPROVED_FOR_LOCAL_DEVELOPMENT
date: "2026-09-05"
author_role: Security Privacy and Product Safety Lead
author_pane: w9:p13
governing_issue: "GitHub Issue #129"
authority_source: HDEC-V040-FOUNDATION-054
governing_decisions:
  - HDEC-V040-FOUNDATION-054
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
  - "ARC-V040-EVD-002 (docs/architecture/v040-evidence-capture-baseline.md)"
  - "ARC-V040-RECOV-001 (docs/architecture/v040-local-draft-recovery-baseline.md)"
retained_unselected_policies:
  binding_scoring_policy: HUMAN_OWNED_UNSELECTED
  finding_closure_policy: HUMAN_OWNED_UNSELECTED
  offline_authority: HUMAN_OWNED_UNSELECTED
  evidence_retention_policy: HUMAN_OWNED_UNSELECTED
  external_storage_provider_policy: HUMAN_OWNED_UNSELECTED
credit_boundary: EVIDENCE_INTEGRITY_BASELINE_SPECIFICATION_ONLY_NO_EXTERNAL_DELIVERY_OR_LIVE_OPERATIONAL_ENACTMENT
---

# v0.4.0 OSHE Inspect Immutable Originals, Derived Objects, Integrity Verification, Transfer, Upload, Export, and Chain of Custody Baseline

## 1. Executive Summary & Governance Posture

### 1.1 Authority Baseline & Purpose
This architectural specification establishes the authoritative **Immutable Originals, Derived Objects, Integrity Verification, Transfer Interruption, Duplicate Handling, Export Manifest, and Chain of Custody Baseline** for **Milestone v0.4.0 - OSHE Inspect Private Alpha** within `OSHEThai/oshe-platform`. It fulfills the deliverable requirements of **GitHub Issue #129 (`[V040-I018] Implement Immutable Originals, Derived Objects, Integrity Verification, Transfer, Upload, Export, and Chain of Custody`)** under Roadmap Topic `V040-T04` and standing owner foundation decision `HDEC-V040-FOUNDATION-054`.

The primary objective is to define a mathematically rigorous, fail-closed evidence integrity architecture within the **Files and Evidence Module (`MOD-EVD`)**, guaranteeing:
- **Permanent Immutability of Accepted Originals:** Accepted original evidence files are write-once, tamper-evident, and cannot be modified, overwritten, or silently replaced.
- **Traceable Derived Objects:** Every derived artifact (thumbnail, compressed preview, watermark, redacted export) is strictly classified, linked to an authoritative original parent, and prevented from masquerading as primary evidence.
- **Continuous Cryptographic Integrity Verification:** End-to-end SHA-256 hashing across client-side capture, local IndexedDB queueing, chunked transfer, server receipt, storage persistence, and export packaging.
- **Resilient Interruption & Duplicate Handling:** Robust handling of network interruptions, resumed uploads, idempotent duplicate submissions, and conflict rejection.
- **Cryptographic Chain of Custody:** Monotonic, hash-chained audit logging capturing all custody transitions from local capture to export packaging.
- **Tamper-Evident Export Manifests:** Deterministic root digest generation and verification across bundled package exports.

### 1.2 Retained Unselected Policies & Non-Claims
1. **Evidence Retention Policy (`HUMAN_OWNED_UNSELECTED`):** Final statutory archival retention, legal hold periods, and disposition schedules remain human-owned.
2. **External Storage Provider Policy (`HUMAN_OWNED_UNSELECTED`):** Direct cloud object store configurations (AWS S3, Azure Blob, Cloudflare R2) remain unselected; all evidence persistence executes via synthetic in-memory storage adapters.
3. **No Real or Production Media (`H040-003`):** 100% synthetic media test fixtures only; zero live customer, worker, or operational imagery.
4. **Retained Foundation Gate Holds:** Foundation gates `H040-007` (Technical Release Authorization), `H040-008` (Real Participant / Private-Alpha UAT Onboarding), `H040-009` (Binding Support Ownership), `H040-010` (External Environment/Route Activation), and `H040-011` (Final Milestone Outcome) remain strictly on **`HOLD`**.

---

## 2. Topological Entity Model: Originals vs. Derived Objects

In `MOD-EVD`, evidence records maintain a strict ontological dichotomy between primary authoritative evidence and downstream derived artifacts:

```
┌────────────────────────────────────────────────────────────────────────┐
│                   Original Evidence (Authoritative)                    │
│  - ObjectType: ORIGINAL                                                │
│  - EvidenceID: evd_syn_orig_01                                         │
│  - ParentEvidenceID: None                                              │
│  - DerivationType: NONE                                                │
│  - Immutability: WRITE-ONCE (Permanent)                                │
└──────────────────────────────────┬─────────────────────────────────────┘
                                   │ 1:N Linkage
                                   ▼
┌────────────────────────────────────────────────────────────────────────┐
│                      Derived Evidence Objects                          │
├──────────────────────────────────┬─────────────────────────────────────┤
│ Thumbnail Preview                │ Compressed Rendition                │
│ - ObjectType: DERIVED            │ - ObjectType: DERIVED               │
│ - DerivationType: THUMBNAIL_PREVIEW│ - DerivationType: COMPRESSED_RENDITION│
│ - ParentEvidenceID: evd_syn_orig_01│ - ParentEvidenceID: evd_syn_orig_01 │
└──────────────────────────────────┴─────────────────────────────────────┘
```

### 2.1 Entity Invariants & Protections
1. **Prohibition of Confusion (`ErrDerivedOriginalConfusion`):** A derived artifact cannot be registered with `ObjectTypeOriginal`, nor can an existing original be converted to a derived artifact.
2. **Parent Linkage Invariant (`ErrMissingParentEvidence`):** Every derived artifact must reference an existing, committed original parent within the identical tenant scope.
3. **Prohibition of Nested Derivations (`ErrNestedDerivationProhibited`):** Derived objects cannot derive from other derived objects; all derivations must stem directly from an authoritative original parent.
4. **Parent Commitment Prerequisite (`ErrRecordNotCommitted`):** Derivation requests for uncommitted or in-flight originals are rejected fail-closed.

---

## 3. Cryptographic Digest Verification & Immutability Guarantees

### 3.1 Content Digest Verification
1. **Client Capture Hashing:** The client computes the SHA-256 digest immediately following EXIF stripping before persistence to local `IndexedDB` (`evidence_blobs`).
2. **Commit Verification:** Upon transfer completion, the receiver recomputes the SHA-256 digest over the entire payload byte stream. If the actual digest differs from the declared hash, the operation fails closed with `ErrTamperDetected`, and an audit event is logged.
3. **Permanent State Immutability:** Once committed (`StateCompleted` / `Committed: true`), the record's payload, digest, and metadata are frozen. Subsequent write attempts fail closed with `ErrOriginalImmutable`.

---

## 4. Transfer Interruption, Resumption & Duplicate Handling Mechanics

### 4.1 Transfer Interruption & Resumption
During intermittent field connectivity:
1. **Interruption Event:** When a network drop or timeout occurs mid-transfer, the system records `EventTransferInterrupted`, sets state to `StateFailed`, and records bytes transferred.
2. **Resumption Event:** When connectivity restores, the transfer resumes, recording `EventTransferResumed` and transitioning state back to `StateTransferring`.
3. **Final Commit:** Upon complete payload receipt and digest verification, the original transitions to `StateCompleted`.

### 4.2 Duplicate Handling Mechanics
1. **Idempotent Duplicate Uploads:** If an upload submission carries an `evidence_id` that is already committed, and the payload's computed SHA-256 matches the committed record's hash, the system recognizes the submission as an idempotent retry. The original payload remains unmodified, and a custody entry `INTEGRITY_VERIFIED` is logged (`"idempotent duplicate upload recognized; original preserved unchanged"`).
2. **Conflicting Duplicate Uploads:** If an upload submission carries an `evidence_id` that is already committed, but the payload's computed SHA-256 differs from the committed record's hash, the submission is rejected fail-closed with `ErrDuplicateEvidenceConflict`.

---

## 5. Tamper Detection & Cryptographic Chain of Custody

### 5.1 Chain of Custody Log
Every evidence object maintains an append-only, ordered ledger of lifecycle custody events:

```
Event 1 (CAPTURE_LOCAL)  -->  Event 2 (INTEGRITY_VERIFIED)  -->  Event 3 (ORIGINAL_COMMITTED)
  Hash: H_1                    Hash: H_2 = SHA256(H_1 + ...)       Hash: H_3 = SHA256(H_2 + ...)
```

Each `CustodyEvent` encapsulates:
- `event_id`: Unique monotonic identifier.
- `evidence_id`: Target evidence identifier.
- `event_type`: Categorical event code (`CAPTURE_LOCAL`, `QUEUE_LOCAL`, `TRANSFER_START`, `TRANSFER_INTERRUPTED`, `TRANSFER_RESUMED`, `INTEGRITY_VERIFIED`, `ORIGINAL_COMMITTED`, `DERIVED_GENERATED`, `PREVIEW_RENDERED`, `EXPORT_PACKAGED`, `TAMPER_DETECTED`).
- `payload_digest`: 64-character SHA-256 digest at time of event.
- `previous_event_digest`: Cryptographic pointer linking to preceding event, forming an immutable hash chain.
- `event_digest`: Cryptographic digest over current entry and previous digest.

### 5.2 Tamper Detection Protocol
When `VerifyTamper` evaluates a candidate payload against a committed record:
1. If $\text{SHA-256}(\text{candidate}) == \text{record.SHA256Digest}$: returns `(true, nil)`.
2. If $\text{SHA-256}(\text{candidate}) \ne \text{record.SHA256Digest}$: appends `EventTamperDetected` to the chain of custody and returns `(false, ErrTamperDetected)`.

---

## 6. Export Manifest Specification & Package Verification

When audit, legal, or compliance packages are generated:

### 6.1 Manifest Structure (`ExportManifest`)
An export manifest bundles all selected evidence items with complete metadata:
- `export_id`: Unique package identifier.
- `tenant_id`: Authoritative tenant context.
- `items`: Ordered list of `ExportManifestItem` containing `evidence_id`, `object_type`, `parent_evidence_id`, `derivation_type`, `media_type`, `size_bytes`, and `sha256_digest`.
- `root_digest`: Cryptographic package root digest computed across sorted items:
  $$\text{root\_digest} = \text{SHA-256}\left(\sum_{\text{sorted } i} \text{EvidenceID}_i \mathbin{\Vert} \text{ObjectType}_i \mathbin{\Vert} \text{ParentEvidenceID}_i \mathbin{\Vert} \text{SHA256Digest}_i \mathbin{\Vert} \text{SizeBytes}_i\right)$$

### 6.2 Package Verification Protocol
The `VerifyExportManifest` function enforces dual-layer validation:
1. **Root Digest Check:** Recomputes the root digest across manifest items. If mismatch, fails closed with `ErrExportTampered`.
2. **Payload Integrity Check:** Verifies each package payload against its manifest `sha256_digest` and `size_bytes`. If any item is missing, truncated, or tampered, fails closed with `ErrExportTampered`.

---

## 7. Synthetic Multi-Scenario Fixtures & Verification Matrix

```yaml
schema_version: "1.0.0"
fixture_id: "fix_syn_evidence_integrity_v1"

originals:
  - evidence_id: "evd_syn_orig_extinguisher_01"
    tenant_id: "ten_syn_01"
    object_type: "ORIGINAL"
    association_type: "INSPECTION_RESPONSE"
    association_id: "rsp_syn_extinguisher_press_01"
    original_name: "extinguisher_gauge.jpg"
    media_type: "image/jpeg"
    size_bytes: 2048500
    sha256_digest: "a1b2c3d4e5f60718293a4b5c6d7e8f90123456789abcdef0123456789abcdef0"
    state: "COMPLETED"
    committed: true

derived_objects:
  - evidence_id: "evd_syn_deriv_thumb_01"
    tenant_id: "ten_syn_01"
    object_type: "DERIVED"
    parent_evidence_id: "evd_syn_orig_extinguisher_01"
    derivation_type: "THUMBNAIL_PREVIEW"
    original_name: "thumb_extinguisher_gauge.jpg"
    media_type: "image/jpeg"
    size_bytes: 32768
    sha256_digest: "d4e5f60718293a4b5c6d7e8f90123456789abcdef0123456789abcdef0123456"
    state: "COMPLETED"
    committed: true

denial_and_fault_scenarios:
  - scenario_id: "fault_tamper_detection"
    action: "VERIFY_TAMPER"
    candidate_digest: "0000000000000000000000000000000000000000000000000000000000000000"
    expected_error: "ErrTamperDetected"
    custody_event: "TAMPER_DETECTED"

  - scenario_id: "fault_derived_confusion"
    action: "REGISTER_DERIVED_AS_ORIGINAL"
    expected_error: "ErrDerivedOriginalConfusion"

  - scenario_id: "fault_nested_derivation"
    action: "DERIVE_FROM_DERIVED"
    parent_id: "evd_syn_deriv_thumb_01"
    expected_error: "ErrNestedDerivationProhibited"

  - scenario_id: "fault_duplicate_conflict"
    action: "DUPLICATE_WITH_CONFLICTING_HASH"
    expected_error: "ErrDuplicateEvidenceConflict"

  - scenario_id: "fault_export_tampered_payload"
    action: "VERIFY_EXPORT_PACKAGE"
    corrupted_item: "evd_syn_orig_extinguisher_01"
    expected_error: "ErrExportTampered"
```

---

## 8. Governance Boundaries, Retained Holds & Non-Claims

In strict compliance with `HDEC-V040-FOUNDATION-054` and `ASN-V040-I018-EVIDENCE-INTEGRITY-001`:

1. **100% Synthetic Data Policy (`H040-003`):** All evidence identifiers, hashes, and binary payloads are synthetic fixtures. Zero real workforce images or customer files are permitted.
2. **No External Route or Cloud Bucket Activation (`H040-007` & `H040-010` HOLD):** Zero external S3/Blob endpoints or direct presigned upload URLs are authorized.
3. **No Real Participant Onboarding (`H040-008` HOLD):** Zero real field inspectors or camera operators are onboarded.
4. **Issue Closure Prohibition:** Issue #129 remains open following this draft pull request until formal review and merge completion.
5. **Specification-Only Credit:** Delivery of this baseline confers architectural integration and test credit only; zero operational release, deployment, or residual-risk acceptance is claimed.
