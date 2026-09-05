---
document_id: QLF-V040-EVDSCAN-001
title: v0.4.0 OSHE Inspect Evidence Capture, Integrity Verification, QR/Barcode Security, and Export Qualification Baseline
document_type: qualification_specification
document_version: 1.0.0
lifecycle_status: DRAFT
status: APPROVED_FOR_LOCAL_DEVELOPMENT
date: "2026-09-05"
author_role: Architecture and Data Lead
author_pane: w9:p22
governing_issue: "GitHub Issue #131"
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
  evidence_retention_policy: HUMAN_OWNED_UNSELECTED
  external_storage_provider_policy: HUMAN_OWNED_UNSELECTED
credit_boundary: TECHNICAL_QUALIFICATION_ONLY_NO_USER_OR_DEVICE_EVIDENCE
---

# v0.4.0 OSHE Inspect Evidence Capture, Integrity Verification, QR/Barcode Security, and Export Qualification Baseline

## 1. Executive Summary & Governance Authority

### 1.1 Authority Baseline & Purpose
This qualification specification establishes the authoritative, deterministic **Evidence Capture, Metadata Minimization, Immutable Originals, Chain of Custody, Interruption Recovery, QR/Barcode Untrusted Resolution, and Export Qualification Baseline** for **Milestone v0.4.0 - OSHE Inspect Private Alpha** within `OSHEThai/oshe-platform`. It fulfills the qualification scope and acceptance criteria of **GitHub Issue #131 (`[V040-I020] Qualify Evidence Capture, Association, Metadata, Integrity, Upload, QR Abuse, Interruption, Restore, and Export`)** under Roadmap Topic `V040-T04` and standing owner foundation decision `HDEC-V040-FOUNDATION-054`.

The primary objective is to define an integrated, dependency-free technical verification harness qualifying the security, privacy, integrity, and operational boundaries established across:
- **`ARC-V040-EVD-002` (Issue #128):** Evidence capture, metadata minimization, local queueing, and record association.
- **`ARC-V040-EVD-003` (Issue #129):** Immutable originals, derived objects, integrity verification, chunked transfer, and chain of custody.
- **`ARC-V040-SCAN-001` (Issue #130):** Barcode, QR, and physical tag untrusted resolution, anti-enumeration, and abuse diagnostics.

### 1.2 Non-Substitution Invariant: Technical & Synthetic Scope Only
In strict compliance with `ASN-V040-I020-EVIDENCE-SCAN-QUALIFICATION-001` and `HDEC-V040-FOUNDATION-054`:
- **Synthetic Technical Qualification Only:** This baseline evaluates deterministic data models, SHA-256 cryptographic verification, EXIF stripping algorithms, state machines, and optical scan parsing using local synthetic fixtures (`evd_*`, `usr_*`, `ins_*`, `scan_*`) and mock adapters.
- **Non-Substitution Invariant:** Automated test harnesses, simulated scripts, and synthetic data **cannot substitute for, replace, or claim the status of real participant, physical camera device, or UAT evidence**. Gate `H040-008` (Real Participant / Private-Alpha UAT Authorization) remains strictly on **`HOLD`** pending explicit owner screening and authorization.

### 1.3 Retained Unselected Policies & Non-Claims
1. **Binding Scoring Policy (`HUMAN_OWNED_UNSELECTED`):** Evidence associations satisfy question rules, but final binding compliance scoring remains human-owned under Issue #136 (`V040-I025`).
2. **Finding Closure Policy (`HUMAN_OWNED_UNSELECTED`):** Evidence submission on findings does not authorize automated closure; finding verification remains human-owned under Issue #134 (`V040-I023`).
3. **Offline Authority Model (`HUMAN_OWNED_UNSELECTED`):** Offline lease parameters remain human-owned under Issue #126 (`V040-I015`).
4. **Evidence Retention & Storage Policies (`HUMAN_OWNED_UNSELECTED`):** Cloud storage provider selection (S3, R2, Azure Blob) and statutory retention periods remain unselected and human-owned.
5. **No Camera or Scanner Device Activation:** Zero physical camera drivers, hardware laser scanners, or NFC readers are activated. Resolution is qualified purely on synthetic scan payloads.
6. **Retained Human Gate Holds:** Foundation gates `H040-007` (Technical Release Authorization), `H040-008` (Real Participant UAT), `H040-009` (Binding Support Ownership), `H040-010` (External Environment Activation), and `H040-011` (Final Milestone Outcome) remain strictly on **`HOLD`**.

---

## 2. Dimension 1: Evidence Association & Parent Entity Scoping

In accordance with `ARC-V040-EVD-002`:

1. **Explicit Parent Entity Binding:** Every evidence item (`evd_syn_*`) must be strictly bound to exactly one authorized parent domain entity:
   - `CHECKLIST_RESPONSE`: Bound to a specific question response within an active inspection session.
   - `FINDING`: Bound to a specific non-conformance finding (`fnd_syn_*`).
   - `ACTION_ITEM`: Bound to a specific corrective action remediation item (`act_syn_*`).
2. **Orphan Prohibition (Fail-Closed):** Evidence items submitted without valid parent entity associations are rejected with `ERR_ORPHAN_EVIDENCE_PROHIBITED`.
3. **Cross-Inspection Scoping Defense:** Attempting to associate evidence to an entity belonging to a different inspection or project is rejected with `ERR_PARENT_CONTEXT_MISMATCH`.

---

## 3. Dimension 2: Privacy Metadata Minimization & EXIF Stripping (`H040-003`)

Under data protection gate `H040-003`:

1. **Mandatory EXIF & Geotag Stripping:**
   - Client upload pipeline strips all sensitive metadata tags from media binaries prior to persistence:
     - GPS Latitude, Longitude, Altitude, Timestamp
     - Device serial number, hardware IMEI, camera serial
     - Camera manufacturer, model, software version
     - Personal owner attribution or author fields
2. **Sanitized Metadata Whitelist:** The persistent metadata model retains only operational attributes:
   - `content_type`: Whitelisted MIME type (`image/jpeg`, `image/png`, `application/pdf`).
   - `file_size_bytes`: Bounded size (strictly <= 10 MiB).
   - `width_px` & `height_px`: Image pixel dimensions.
   - `sha256_digest`: Cryptographic content checksum.
   - `captured_at`: Operational UTC timestamp.
   - `caption`: Optional sanitized text description (max 255 chars; PII patterns rejected).

---

## 4. Dimension 3: Immutable Originals vs. Derived Objects

In accordance with `ARC-V040-EVD-003`:

1. **Write-Once Originals (`ORIGINAL_ACCEPTED`):**
   - Once uploaded and verified against its SHA-256 checksum, an original evidence binary is permanently sealed and immutable.
   - Any attempt to overwrite, truncate, or update an accepted original fails closed with `ERR_ORIGINAL_IMMUTABLE`.
2. **Classification of Derived Objects (`DERIVED_OBJECT`):**
   - Auxiliary artifacts (thumbnails, downscaled previews, watermarked representations, redacted views) are explicitly classified as `DERIVED_OBJECT`.
   - Every derived object stores an authoritative reference to its parent: `original_evidence_id` and `original_content_digest`.
3. **Anti-Masquerading Invariant:** Derived objects can **never** replace, overwrite, or masquerade as primary evidence in audit logs or regulatory exports (`ERR_DERIVED_CANNOT_REPLACE_ORIGINAL`).

---

## 5. Dimension 4: Cryptographic Integrity, Tamper Detection & Duplicate Handling

Under `ARC-V040-EVD-003`:

1. **Continuous Digest Verification:** Every evidence artifact computes and seals a canonical SHA-256 content digest at initial client capture.
2. **Tamper Detection (Fail-Closed):**
   - If an evidence file's byte sequence is modified in storage, recalculation of the SHA-256 digest fails to match the sealed `sha256_digest`.
   - Verification fails closed immediately with `ERR_DIGEST_TAMPER_DETECTED`.
3. **Idempotent Duplicate Upload Handling:**
   - If a client re-transmits an evidence upload with matching `idempotency_key` and matching `sha256_digest`, the server returns HTTP 200 with existing metadata (`ACK_IDEMPOTENT_DUPLICATE`) without storing redundant binary duplicates.
   - If the same idempotency key is submitted with a different file digest, the request fails closed with `ERR_IDEMPOTENCY_CONFLICT`.

---

## 6. Dimension 5: Transfer Interruption, Resumption & Export Manifests

In accordance with `ARC-V040-EVD-003`:

1. **Chunked Upload Interruption Resilience:**
   - Client transfers media in deterministic 1 MiB chunks.
   - If network connectivity drops mid-upload, the server preserves verified chunk offsets.
   - On resume, the client queries uploaded chunk status and resumes from the last confirmed byte offset without re-transmitting verified chunks.
2. **Tamper-Evident Export Package & Manifest Verification:**
   - Bundled exports assemble original evidence files, parent associations, metadata ledgers, and a cryptographic `manifest.json`.
   - The export manifest computes a Merkle root digest over all constituent evidence digests (`pkg_manifest_digest`).
   - Any missing, corrupted, or altered file within the export package invalidates the manifest root and causes export verification to fail closed (`ERR_EXPORT_MANIFEST_INVALID`).

---

## 7. Dimension 6: Barcode & QR Untrusted Scan Resolution Security

Under `ARC-V040-SCAN-001`:

### 7.1 The Cardinal Security Invariant: A Scan is Never Authority
- **Untrusted External Input:** Optical scans (QR, Code 128, DataMatrix) and NFC payloads are strictly untrusted external inputs.
- **Zero Inherent Authorization:** Decoding a QR code or barcode **never grants permissions**, bypasses role checks, or elevates privilege.
- **Mandatory Ambient Authorization Prerequisite:** Access to the resolved resource is gated entirely by the caller's ambient, authenticated session (`usr_*`), tenant membership (`ten_*`), and project role (`MOD-IAM`).

### 7.2 Malicious Payload & Injection Defenses
Incoming scan strings are subjected to multi-layered sanitization:
1. **SQL Injection & Script Tag Rejection:** Payloads containing SQL syntax (`SELECT`, `UNION`, `DROP`), shell metacharacters, or HTML `<script>` tags are rejected with `ERR_SCAN_PAYLOAD_MALICIOUS`.
2. **Path Traversal Defense:** Payloads containing directory traversal sequences (`../`, `..\`) are rejected with `ERR_SCAN_PAYLOAD_MALICIOUS`.
3. **Length & Protocol Bounding:** Payloads exceeding 512 bytes or attempting unapproved URI schemes (`javascript:`, `data:`, `file:`) are rejected with `ERR_SCAN_PAYLOAD_INVALID`.

### 7.3 Guessing, Brute-Force & Cross-Tenant Non-Enumeration
1. **Cross-Tenant Scanning Rejection:** An inspector in Tenant A scanning a legitimate equipment QR code belonging to Tenant B is rejected with a generic `ERR_SCAN_RESOURCE_NOT_FOUND`.
2. **Anti-Enumeration Uniformity:** Randomly guessed, malformed, or cross-tenant scan payloads return an identical HTTP 404 generic error structure with uniform response latency ($\pm 5\text{ms}$ simulated delay) to prevent timing side-channels and entity existence enumeration.

---

## 8. Synthetic Qualification Scenarios Fixture Matrix

The following synthetic YAML fixture specifies the complete qualification scenario catalog:

```yaml
schema_version: "1.0.0"
fixture_id: "fix_syn_evidence_scan_qualification_v1"

scenarios:
  - id: "QLF-EVD-01"
    name: "Valid Evidence Association"
    parent_type: "CHECKLIST_RESPONSE"
    parent_id: "qst_syn_ground_01"
    media_type: "image/jpeg"
    file_size_bytes: 2048576
    expected_result: "PASS_ASSOCIATED"

  - id: "QLF-EVD-02"
    name: "Orphan Evidence Rejection"
    parent_type: "NONE"
    parent_id: ""
    expected_result: "REJECT_ERR_ORPHAN_EVIDENCE_PROHIBITED"

  - id: "QLF-EVD-03"
    name: "EXIF Stripping Verification"
    input_exif:
      GPSLatitude: "13.7563 N"
      GPSLongitude: "100.5018 E"
      CameraModel: "Pixel 7 Pro"
    expected_output_exif: {}
    expected_result: "PASS_METADATA_MINIMIZED"

  - id: "QLF-EVD-04"
    name: "Cryptographic Digest Tamper Detection"
    sealed_digest: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
    simulated_byte_tamper: true
    expected_result: "FAIL_ERR_DIGEST_TAMPER_DETECTED"

  - id: "QLF-EVD-05"
    name: "Idempotent Duplicate Upload"
    idempotency_key: "idem_syn_evd_001"
    action: "REPEAT_UPLOAD_SAME_DIGEST"
    expected_result: "ACK_IDEMPOTENT_DUPLICATE_200"

  - id: "QLF-EVD-06"
    name: "Malicious QR Payload Rejection"
    scan_payload: "oshe://inspect/equip?id=1;DROP TABLE equipments;--"
    expected_result: "REJECT_ERR_SCAN_PAYLOAD_MALICIOUS"

  - id: "QLF-EVD-07"
    name: "Cross-Tenant Scan Non-Enumeration"
    caller_tenant: "ten_syn_alpha"
    target_tenant: "ten_syn_bravo"
    scan_payload: "oshe://inspect/equip/eq_bravo_001"
    expected_result: "REJECT_ERR_SCAN_RESOURCE_NOT_FOUND"

  - id: "QLF-EVD-08"
    name: "Export Package Manifest Root Verification"
    package_id: "pkg_syn_export_001"
    constituent_evidence_count: 5
    expected_result: "PASS_MANIFEST_ROOT_VERIFIED"
```

---

## 9. Governance Boundaries & Non-Claims

In strict adherence to `HDEC-V040-FOUNDATION-054`:

1. **Synthetic-Only Data Policy (`H040-003`):** All evidence files, EXIF tags, scan strings, and parent entities operate strictly as fictionalized alpha fixtures. Zero real customer data, live photos, or physical equipment QR codes are processed.
2. **Zero Real-User or Hardware Claim:** Automated tests cannot substitute for empirical human inspector trials or physical optical scanner hardware. Gate `H040-008` remains on strict **`HOLD`**.
3. **Zero Deployment or Release Claim:** Gates `H040-007` through `H040-011` remain on **`HOLD`**. Zero live cloud storage buckets, public DNS routes, or production releases are claimed or authorized.
