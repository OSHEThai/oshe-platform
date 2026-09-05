---
document_id: ARC-V040-PROF-001
title: v0.4.0 Private-Alpha Supported Profile, Device, Environment, Language, Data, Offline, NFR, and Compatibility Baseline
document_type: architecture_specification
document_version: 1.0.0
lifecycle_status: DRAFT
status: APPROVED_FOR_LOCAL_DEVELOPMENT
date: "2026-09-05"
author_role: Security Privacy and Product Safety Lead
author_pane: w9:p13
governing_issue: "GitHub Issue #114"
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
credit_boundary: PROFILE_BASELINE_SPECIFICATION_ONLY_NO_PARTICIPANT_ONBOARDING_OR_BINDING_SLA
---

# v0.4.0 Private-Alpha Supported Profile, Device, Environment, Language, Data, Offline, NFR, and Compatibility Baseline

## 1. Executive Summary & Governance Reference

### 1.1 Governance Reference
This architectural specification establishes the authoritative **Supported Private-Alpha Device, Environment, Language, Data, Offline, NFR, and Compatibility Profile Baseline** for **Milestone v0.4.0 - OSHE Inspect Private Alpha** within `OSHEThai/oshe-platform`. It fulfills the requirements and deliverable specifications of **GitHub Issue #114 (`[V040-I003] Freeze Supported Private-Alpha Device, Environment, Language, Data, Offline, NFR, and Compatibility Profile`)** under the governing authority of Sole Human Owner decision `HDEC-V040-FOUNDATION-054`.

### 1.2 Boundary & Purpose
The purpose of this document is to freeze a controlled, bounded development profile for the standalone single-tenant OSHE Inspect vertical slice (`H040-001`). All technical targets, environmental boundaries, and capability matrices defined herein serve to guide engineering and qualification during the private-alpha phase.

### 1.3 Strict Non-Binding and TBD Invariant
In strict adherence to assignment directives:
- **Every performance threshold, latency objective, capacity ceiling, and sync window proposed in this specification is explicitly designated as `[NON-BINDING_PROPOSED]` or `[TBD]`.**
- **Zero performance claims, binding service level agreements (SLAs), or uptime guarantees are made.**
- All thresholds remain non-binding proposals until subjected to empirical testing and formal Sole Human Owner sign-off.

### 1.4 Deferred Human Authority & Operational Non-Claims
In compliance with `HDEC-V040-FOUNDATION-054`, the following gates remain permanently on **`HOLD`**:
- **Gate `H040-007` (Technical Release Authorization):** `HOLD`.
- **Gate `H040-008` (Real Participant / Private Alpha UAT Onboarding):** `HOLD`. Zero real-user recruitment, testing, or onboarding is authorized.
- **Gate `H040-009` (Binding Support and Manual-Fallback Ownership):** `HOLD`. No binding support commitments, operational runbook staffing, or helpdesk SLAs are enacted.
- **Gate `H040-010` (External Environment, Account, Route, or Effect Activation):** `HOLD`.
- **Gate `H040-011` (Final Milestone v0.4.0 Outcome and Residual-Risk Acceptance):** `HOLD`.

---

## 2. Supported Device, OS, and Browser Matrix (`H040-002`)

### 2.1 Supported Client Platforms
In accordance with `H040-002`, client support is constrained to modern responsive web environments:

| Platform Category | Operating System | Browser Engine | Version Baseline | Status | Input Modality |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **Desktop Web** | Windows 10 / 11 | Google Chrome (Blink) | Current Stable (v128+) | `SUPPORTED_ALPHA` | Mouse, Keyboard |
| **Desktop Web** | Windows 10 / 11 | Microsoft Edge (Blink) | Current Stable (v128+) | `SUPPORTED_ALPHA` | Mouse, Keyboard |
| **Mobile Web** | Android 12+ | Google Chrome Mobile (Blink) | Current Stable (v128+) | `SUPPORTED_ALPHA` | Multi-touch, Virtual Keyboard |
| **Desktop Web (Dev)** | macOS 14+ | Google Chrome (Blink) | Current Stable (v128+) | `SUPPORTED_DEV_ONLY` | Mouse, Keyboard |

### 2.2 Screen Viewport & Responsive Breakpoints
- **Mobile Viewport:** Minimum resolution `360 x 640` CSS pixels (portrait).
- **Tablet Viewport:** Minimum resolution `768 x 1024` CSS pixels (portrait/landscape).
- **Desktop Viewport:** Target baseline `1280 x 720` CSS pixels up to `1920 x 1080`.
- **Display Scaling:** Supports browser zoom levels from 80% to 150% without functional clipping.

### 2.3 Unsupported Platforms Declaration
The following client environments are explicitly declared **`UNSUPPORTED_PRIVATE_ALPHA`**:
- Apple iOS (Safari, Chrome for iOS, WebKit-based browsers).
- Mozilla Firefox (Desktop and Mobile Gecko engines).
- Legacy Microsoft browsers (Internet Explorer, Edge Legacy EdgeHTML).
- Embedded webviews within third-party container applications (e.g. LINE in-app browser, Facebook in-app browser).
- Custom Android distributions lacking Google Mobile Services or modern Chromium webviews.
- Native mobile applications (zero native APK or iOS IPA builds are planned or authorized for v0.4.0).

---

## 3. Localization, Language, and Temporal Baseline (`H040-002`)

### 3.1 Supported Languages
The user interface, validation messaging, and checklist schemas support two locales:
1. **English (`en-US`):** Primary administrative and technical schema language.
2. **Thai (`th-TH`):** Primary field inspection and frontline workforce localization.

### 3.2 Canonical Operational Time Zone
- **Time Zone:** Canonical time zone is strictly **`Asia/Bangkok` (UTC+7, Indochina Time)**.
- **Internal Storage:** All timestamps in server storage and client write-ahead journals must be formatted as UTC ISO 8601 strings (e.g. `2026-09-05T14:25:00Z`).
- **UI Representation:** Display conversion to `Asia/Bangkok` with dual Gregorian and Buddhist Era (`BE`) display formatting for Thai locale.

---

## 4. Data Governance, Minimization, and Synthetic Scope (`H040-003`)

### 4.1 Synthetic Data Mandate
During Milestone v0.4.0, all data across databases, caches, test fixtures, and mock services must adhere to the **100% Synthetic Data Policy (`H040-003`)**:
- **Prohibition:** Zero real customer data, real corporate records, real site blueprints, real safety incidents, real employee contact details, or biometric records.
- **Controlled Synthetic Prefixes:**
  - Synthetic Users: `usr_syn_<role>_<id>` (e.g., `usr_syn_inspector_01`)
  - Synthetic Inspections: `ins_syn_<id>`
  - Synthetic Sites/Areas: `ste_syn_<id>`, `ara_syn_<id>`
  - Synthetic Findings: `fnd_syn_<id>`
  - Synthetic Checklists: `chk_syn_<id>`

### 4.2 Local Browser Storage Security & Minimization
- **Storage Technologies:** Browser local storage is limited to `IndexedDB` for offline inspection state and `SessionStorage` for active session state.
- **Zero Raw Credentials:** Passwords, unhashed tokens, and long-lived private keys are strictly forbidden from browser persistence.
- **Session Purge:** Termination of inspection session or user logout triggers immediate cryptographic zeroization and deletion of cached inspection working state.

---

## 5. Offline, Online, and Network State Envelope (`H040-005`)

### 5.1 Operating Network Modes

```
┌─────────────────────────┐         Network Detected         ┌─────────────────────────┐
│   ONLINE_CONNECTED      │ ───────────────────────────────> │  INTERMITTENT_SYNCING   │
│ - Direct server comms   │                                  │ - Batch queue drain     │
│ - Instant commit ack    │ <─────────────────────────────── │ - Conflict validation   │
└─────────────────────────┘          Sync Completed          └────────────┬────────────┘
             │                                                            │
       Network Lost                                                 Collision Detected
             │                                                            │
             ▼                                                            ▼
┌─────────────────────────┐                                  ┌─────────────────────────┐
│  OFFLINE_DISCONNECTED   │                                  │  QUARANTINED_COLLISION  │
│ - IndexedDB journaling  │                                  │ - Version lock          │
│ - Optimistic UI state   │                                  │ - Supervisory reconcile │
└─────────────────────────┘                                  └─────────────────────────┘
```

### 5.2 Server Authority & Concurrency Architecture (`H040-005`)
- **Server Sole Authority:** The central server is the single authoritative source of truth. Client devices submit candidate mutations.
- **Zero Last-Write-Wins (LWW):** Blind overwriting based on client-side timestamps is strictly prohibited.
- **Conflict Quarantine:** Any mutation submitted against a stale state version triggers **`QUARANTINED`** status. The inspection record is locked for manual supervisory reconciliation.
- **Conservative Offline Age:** `[NON-BINDING_PROPOSED]` Maximum offline duration before mandatory server re-check is proposed as **24 hours** `[TBD]`.

---

## 6. Non-Functional Requirements (NFR) Catalog (Non-Binding Targets)

All performance, capacity, and recovery targets below are non-binding engineering guidelines for private-alpha development. They carry zero contractual SLA or production readiness commitment:

| NFR ID | Capability Domain | Proposed Metric / Parameter | Proposed Alpha Target | Status / Gating |
| :--- | :--- | :--- | :--- | :--- |
| **`NFR-PERF-01`** | **UI Responsiveness** | Input latency on checklist item response | `< 100 ms` | `[NON-BINDING_PROPOSED]` |
| **`NFR-PERF-02`** | **Initial Page Load** | P95 load time on broadband (Chrome desktop) | `< 2.5 s` | `[NON-BINDING_PROPOSED]` |
| **`NFR-PERF-03`** | **Mobile Initial Load** | P95 load time on 4G LTE (Android Chrome) | `< 4.0 s` | `[NON-BINDING_PROPOSED]` |
| **`NFR-SYNC-01`** | **Sync Throughput** | Batch sync time for 100-item inspection | `< 3.0 s` | `[NON-BINDING_PROPOSED]` |
| **`NFR-CAP-01`** | **Local Cache Capacity** | Maximum local draft storage per device | `50 MB` | `[TBD]` |
| **`NFR-CAP-02`** | **Checklist Complexity** | Maximum items per single checklist template | `250 items` | `[NON-BINDING_PROPOSED]` |
| **`NFR-A11Y-01`** | **Accessibility** | WCAG 2.1 Level AA color contrast compliance | `>= 4.5:1 text` | `[NON-BINDING_PROPOSED]` |
| **`NFR-A11Y-02`** | **Keyboard Navigation** | Visible focus ring & logical tab order | `100% interactive` | `[NON-BINDING_PROPOSED]` |
| **`NFR-REC-01`** | **Crash Resilience** | Unsynced draft recovery after unexpected tab close | `100% recovery` | `[NON-BINDING_PROPOSED]` |
| **`NFR-REL-01`** | **Availability Target** | Local development container uptime | `TBD` | `[TBD]` |

---

## 7. Compatibility, Interoperability, and Unsupported Envelope

### 7.1 External Systems & Integration Boundary
Milestone v0.4.0 is strictly self-contained. The following enterprise integration capabilities are out of scope:
- **ERP / Enterprise Asset Management:** Zero integration with SAP, Oracle, Maximo, or IBM.
- **Enterprise Identity Providers:** Zero SAML 2.0, OpenID Connect, or Microsoft Entra ID enterprise federations. Authentication uses synthetic local session tokens.
- **External File Storage / CDN:** Zero direct S3, Azure Blob, or Cloudflare R2 uploads from client browsers.
- **Notification Services:** Zero SMS, external email gateways, or push notification provider integrations (e.g. Firebase Cloud Messaging).

### 7.2 Hardware Peripherals
- **Barcode & QR Scanners:** Mobile camera video stream scanning via standard WebRTC MediaDevices API only. Dedicated external hardware scanners (HID/Bluetooth) are `UNSUPPORTED_PRIVATE_ALPHA`.
- **Sensors & Wearables:** Zero IoT, environmental sensor, or wearable biometric device integrations.

---

## 8. Governance Invariants & Non-Claims

In strict compliance with `HDEC-V040-FOUNDATION-054` and `ASN-V040-I003-PROFILE-BASELINE-001`:

1. **Zero Participant Authorization:** Gate `H040-008` remains on `HOLD`. This document does **NOT** authorize real participants, private-alpha test cohorts, or customer pilots.
2. **Zero Deployment Authority:** Gate `H040-007` and Gate `H040-010` remain on `HOLD`. No external domains, public routing, or cloud infrastructure provisioning is authorized.
3. **Zero Support or SLA Commitment:** Gate `H040-009` remains on `HOLD`. No operational helpdesk, binding response time, or manual fallback ownership is established.
4. **Zero Residual-Risk Acceptance:** Gate `H040-011` remains on `HOLD`. Residual operational and safety risk acceptance is strictly reserved for the Sole Human Owner.
5. **Specification-Only Credit:** Delivery of this specification confers documentation baseline credit only; it does not confer qualification, release, or production authority.
