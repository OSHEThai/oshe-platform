---
document_id: ARC-V040-SCAN-001
title: v0.4.0 OSHE Inspect Barcode, QR, and Physical Tag Untrusted Scan Resolution, Scope Validation, and Anti-Enumeration Baseline
document_type: architecture_specification
document_version: 1.0.0
lifecycle_status: DRAFT
status: APPROVED_FOR_LOCAL_DEVELOPMENT
date: "2026-09-05"
author_role: Engineering Lead
author_pane: w9:p23
governing_issue: "GitHub Issue #130"
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
credit_boundary: SCAN_RESOLUTION_SPECIFICATION_ONLY_NO_EXTERNAL_DELIVERY_OR_LIVE_OPERATIONAL_ENACTMENT
---

# v0.4.0 OSHE Inspect Barcode, QR, and Physical Tag Untrusted Scan Resolution, Scope Validation, and Anti-Enumeration Baseline

## 1. Executive Summary & Governance Posture

### 1.1 Authority Baseline & Purpose
This architectural specification establishes the authoritative **Barcode, QR Code, and Physical Tag Untrusted Scan Resolution, Multi-Layered Validation, and Anti-Enumeration Baseline** for **Milestone v0.4.0 - OSHE Inspect Private Alpha** within `OSHEThai/oshe-platform`. It fulfills the deliverable requirements of **GitHub Issue #130 (`[V040-I019] Barcode, QR, and Physical Tag Resolution Mechanics`)** under Roadmap Topic `V040-T04` and standing owner foundation decision `HDEC-V040-FOUNDATION-054`.

The primary objective is to define a robust, fail-closed security boundary governing the interpretation of physical tags, equipment barcodes, and QR codes within the **Identity and Authorization Module (`MOD-IAM`)** and **Organization and Tenancy Module (`MOD-ORG`)**.

### 1.2 The Cardinal Security Invariant: A Scan is Never Authority
1. **Untrusted User Input:** An optical scan (QR, Code 128, DataMatrix) or NFC tap payload is strictly **untrusted external input**.
2. **Zero Inherent Authorization:** Merely presenting or decoding a scan payload **never grants authority**, bypasses permission checks, elevates privileges, or discloses resource attributes.
3. **Mandatory Ambient Authorization Prerequisite:** Every scan resolution request evaluates strictly against the ambient, authenticated caller identity (`usr_*`), active tenant scope (`ten_*`), geographic assignment (`ste_*`, `ara_*`), and discrete role permissions (`MOD-IAM`).

### 1.3 Retained Unselected Policies & Non-Claims
In strict compliance with `ASN-V040-I019-SCAN-RESOLUTION-001` and `HDEC-V040-FOUNDATION-054`:
1. **Binding Scoring Policy (`HUMAN_OWNED_UNSELECTED`):** Scanned checklists associate scoring references, but final compliance ratings and weights remain human-owned under Issue #136 (`V040-I025`).
2. **Finding Closure Policy (`HUMAN_OWNED_UNSELECTED`):** Scanning a finding tag does not authorize closure; finding verification remains human-owned under Issue #134 (`V040-I023`).
3. **Offline Authority Model (`HUMAN_OWNED_UNSELECTED`):** Offline tag cache validity and local lookup leases remain human-owned under Issue #126 (`V040-I015`).
4. **No Real Hardware or Device Activation:** Zero physical camera drivers, hardware laser barcode scanners (HID/Bluetooth), NFC readers, or mobile device management (MDM) integrations are authorized or activated. Resolution is modeled strictly via synthetic string fixtures.
5. **Retained Human Gate Holds:** Foundation gates `H040-007` (Technical Release Authorization), `H040-008` (Real Participant / Private-Alpha UAT Onboarding), `H040-009` (Binding Support Ownership), `H040-010` (External Environment/Route Activation), and `H040-011` (Final Milestone Outcome) remain strictly on **`HOLD`**.

---

## 2. Threat Model & Physical Scan Attack Surface

Physical identifiers deployed in industrial plants, construction sites, and offshore platforms operate in hostile physical environments:

```
┌────────────────────────────────────────────────────────────────────────┐
│                        SCAN THREAT VECTORS                             │
│                                                                        │
│  [1. Physical Sticker Swapping] ──> Malicious QR placed over valid tag │
│  [2. Identifier Enumeration]    ──> Guessing sequential asset tags     │
│  [3. Cross-Tenant Probe]        ──> Injecting competitor tenant ID     │
│  [4. Path Traversal Injection]  ──> Injecting ../ or null bytes        │
│  [5. Stale / Expired Tag Replay]──> Scanning decommissioned equipment  │
│  [6. Unauthorized Privilege]    ──> Worker scans executive admin tag   │
└────────────────────────────────────────────────────────────────────────┘
```

1. **Sticker Swapping / Malicious Redirection:** An adversary replaces an equipment barcode sticker with a malicious payload pointing to a different area, a phishing URL, or a different tenant's asset.
2. **Identifier Enumeration & Guessing:** Attackers sequentially probe asset IDs (`eqp_0001`, `eqp_0002`) to harvest asset rosters or infer inspection schedules.
3. **Cross-Tenant Probing:** A multi-tenant client scans a tag from another organization to test tenant isolation.
4. **Path Traversal & Parser Exploits:** Malformed scan strings containing directory traversal (`../`), null bytes (`\x00`), or command delimiters attempting parser evasion.
5. **Stale / Expired Tag Replay:** Attempting to scan tags belonging to decommissioned assets or expired temporary work areas.

---

## 3. Supported Synthetic URI & Tag Schemes

Under `modules/identity-authorization/scan_resolution.go`, the parser accepts exactly three normalized, canonical schemes:

### 3.1 Scheme Specifications

| Format | Scheme Syntax | Example Synthetic Payload | Use Case |
| :--- | :--- | :--- | :--- |
| **URI Scheme** | `oshe://<tenant>/<type>/<id>?token=...&exp=...` | `oshe://ten_syn_01/equipment/eqp_boiler_b1?token=tok123&exp=1789000000` | High-density QR code with temporal expiry. |
| **Compact Tag** | `oshe:<tenant>:<type>:<id>` | `oshe:ten_syn_01:area:ara_boiler_room` | Low-density 1D Barcode (Code 128) or physical metal stamped tag. |
| **Local HTTPS** | `https://app.oshe.local/scan?tenant=...&type=...&id=...` | `https://app.oshe.local/scan?tenant=ten_syn_01&type=site&id=ste_rayong` | NFC tag payload launching localized alpha PWA. |

### 3.2 Permitted Scannable Object Types & Canonical Prefixes
In strict compliance with `packages/identifiers/identifiers.go`:

| Object Type | String Identifier | Canonical Prefix | Example Valid Identifier |
| :--- | :--- | :--- | :--- |
| **Equipment** | `equipment` | `eqp_` | `eqp_boiler_b1` |
| **Site** | `site` | `ste_` | `ste_rayong` |
| **Area** | `area` | `ara_` | `ara_boiler_room` |
| **Checklist** | `checklist` | `chk_` | `chk_daily_safety` |
| **Inspection** | `inspection` | `ins_` | `ins_syn_clean_01` |
| **Finding** | `finding` | `fnd_` | `fnd_syn_blocked_exit_01` |

- **Prefix Invariant:** A declared object type must strictly match its canonical prefix. For example, a payload with `type: equipment` but `id: chk_123` fails closed with `ErrInvalidObjectIdentifier`.

---

## 4. Multi-Layered Validation & Resolution Pipeline

Scan resolution follows a sequential, defense-in-depth pipeline where any failure halts evaluation immediately:

```
[Untrusted Scan Input]
        │
        ▼
[1. Syntax & Scheme Parse] ────────> (Malformed / Traversal / Null Byte) ──> DENIAL_SCAN_INVALID_INPUT
        │
        ▼
[2. Canonical Prefix Check] ───────> (Prefix Mismatch for Type) ──────────> DENIAL_SCAN_INVALID_INPUT
        │
        ▼
[3. Temporal Expiration Check] ────> (Expired: Now > exp) ─────────────────> DENIAL_SCAN_EXPIRED
        │
        ▼
[4. Tenant Boundary Check] ────────> (payload.TenantID != caller.TenantID) ─> DENIAL_SCAN_UNAUTHORIZED
        │
        ▼
[5. Object Existence Check] ───────> (Object Not in Tenant Directory) ────> DENIAL_SCAN_UNAUTHORIZED
        │
        ▼
[6. Lifecycle State Check] ────────> (Lifecycle != ACTIVE) ─────────────────> DENIAL_SCAN_UNAUTHORIZED
        │
        ▼
[7. Scope Confinement Check] ──────> (Caller Site != Target Site) ──────────> DENIAL_SCAN_UNAUTHORIZED
        │
        ▼
[8. Role & Permission Check] ──────> (Role Lacks Required Permission) ─────> DENIAL_SCAN_UNAUTHORIZED
        │
        ▼
[RESOLVED & AUTHORIZED] ───────────> Return Target ScannableObject & Log Audit
```

---

## 5. Anti-Enumeration Protections & Non-Leaking Diagnostics

To eliminate side-channel information leakage:

### 5.1 Constant-Time & Indistinguishable Error Responses
When an unauthorized caller submits a scan payload, the response must **never disclose whether the target resource exists**.
- Probing a **real unauthorized asset** (`eqp_boiler_b1`) vs. probing a **fictitious, guessed asset** (`eqp_guessed_9999`) produces:
  - Identical status: `Allowed = false`
  - Identical denial code: `DenialScanUnauthorized` (`"DENIAL_SCAN_UNAUTHORIZED"`)
  - Identical error message: `"access denied to target resource"`
  - Nil object payload: `ResolvedObject = nil`

### 5.2 Internal vs. External Diagnostic Separation
While external callers receive generic non-enumerating errors, internal audit entries (`MOD-REC`) capture detailed diagnostics (e.g. `"scanned object does not exist in tenant"`, `"cross-tenant access attempt"`, `"role lacks permission"`) to support security investigations and forensics without exposing system state.

---

## 6. Append-Only Audit Logging (`MOD-REC`)

Every scan resolution attempt—successful or denied—is recorded in the append-only in-memory ledger (`ScanResolver.AuditLedger()`):

```yaml
audit_record:
  record_id: "audit_scan_a1b2c3d4"
  tenant_id: "ten_synthetic_alpha"
  actor_subject: "usr_inspector_01"
  caller_role: "INSPECTOR"
  raw_payload_hash: "8f4e2b1c3a7d9e5f0b2a4c6d8e1f3a5b7c9d1e3f5a7b9c1d3e5f7a9b1c3d5e7f"
  parsed_object_type: "equipment"
  parsed_object_id: "eqp_boiler_b1"
  allowed: true
  denial_code: ""
  internal_diagnostic: "scan resolved and caller authorized successfully"
  timestamp: "2026-09-05T17:00:00.000Z"
```

---

## 7. Synthetic Operations Fixture Matrix

The following synthetic YAML fixture illustrates resolution, denial, expiration, and anti-enumeration scenarios:

```yaml
schema_version: "1.0.0"
fixture_id: "fix_syn_scan_resolution_v1"

scenarios:
  # Scenario 1: Clean Authorized Scan Resolution
  - scenario_id: "scen_scan_clean_01"
    caller:
      subject: "usr_inspector_01"
      tenant_id: "ten_synthetic_alpha"
      role: "INSPECTOR"
      active_site: "ste_rayong"
    raw_scan: "oshe://ten_synthetic_alpha/equipment/eqp_boiler_b1"
    expected_allowed: true
    expected_denial: null
    resolved_object_id: "eqp_boiler_b1"

  # Scenario 2: Malformed Syntax & Path Traversal Rejection
  - scenario_id: "scen_scan_traversal_02"
    caller:
      subject: "usr_inspector_01"
      tenant_id: "ten_synthetic_alpha"
      role: "INSPECTOR"
    raw_scan: "oshe://ten_synthetic_alpha/equipment/../../etc/passwd"
    expected_allowed: false
    expected_denial: "DENIAL_SCAN_INVALID_INPUT"

  # Scenario 3: Expired Temporal QR Code
  - scenario_id: "scen_scan_expired_03"
    caller:
      subject: "usr_inspector_01"
      tenant_id: "ten_synthetic_alpha"
      role: "INSPECTOR"
      active_site: "ste_rayong"
    raw_scan: "oshe://ten_synthetic_alpha/equipment/eqp_boiler_b1?exp=1725537600"
    current_time: "2026-09-05T12:00:00Z"
    expected_allowed: false
    expected_denial: "DENIAL_SCAN_EXPIRED"

  # Scenario 4: Cross-Tenant Scan Attack
  - scenario_id: "scen_scan_cross_tenant_04"
    caller:
      subject: "usr_inspector_01"
      tenant_id: "ten_synthetic_alpha"
      role: "INSPECTOR"
    raw_scan: "oshe://ten_foreign_beta/equipment/eqp_boiler_b1"
    expected_allowed: false
    expected_denial: "DENIAL_SCAN_UNAUTHORIZED"

  # Scenario 5: Anti-Enumeration on Guessed Asset ID
  - scenario_id: "scen_scan_guessed_id_05"
    caller:
      subject: "usr_viewer_01"
      tenant_id: "ten_synthetic_alpha"
      role: "VIEWER"
    raw_scan: "oshe://ten_synthetic_alpha/equipment/eqp_guessed_9999"
    expected_allowed: false
    expected_denial: "DENIAL_SCAN_UNAUTHORIZED"
    error_message: "access denied to target resource"
    existence_leaked: false
```

---

## 8. Governance Boundaries, Prohibitions & Non-Claims

In strict compliance with `HDEC-V040-FOUNDATION-054` and `ASN-V040-I019-SCAN-RESOLUTION-001`:

1. **100% Synthetic Data Policy (`H040-003`):** All scan strings, tenant identifiers, and asset tags are synthetic in-memory fixtures. Zero real customer data, serial numbers, or operational barcodes are referenced.
2. **A Scan is Never Authority (`H040-004`):** Presenting an encoded scan string conveys zero authorization or privilege escalation.
3. **No External Route or Device Activation (`H040-007` & `H040-010` HOLD):** Zero camera capture APIs, hardware scanners, external lookup routes, or cloud providers are activated.
4. **No Real Participant Onboarding (`H040-008` HOLD):** Zero real field inspectors or mobile users are onboarded.
5. **Specification-Only Credit:** Delivery of this baseline confers documentation and architectural baseline credit only; zero operational release, deployment, or residual-risk acceptance is claimed.
