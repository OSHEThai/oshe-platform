---
document_id: SEC-V040-BASE-001
title: v0.4.0 Threat Model, Privacy Impact Assessment, Safety Hazard Log, and Critical-Function Baseline
document_type: security_and_safety_baseline
document_version: 1.0.0
lifecycle_status: DRAFT
status: APPROVED_FOR_LOCAL_ALPHA_DEVELOPMENT
date: "2026-09-05"
author_role: Security Privacy and Product Safety Lead
author_pane: w9:p13
governing_issue: "GitHub Issue #145"
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
credit_boundary: BASELINE_SPECIFICATION_ONLY_NO_RELEASE_FALLBACK_OWNERSHIP_OR_RESIDUAL_RISK_ACCEPTANCE
---

# v0.4.0 Threat Model, Privacy Impact Assessment, Safety Hazard Log, and Critical-Function Baseline

## 1. Executive Summary & Governance Reference

### 1.1 Governance Reference
This document establishes the authoritative **Threat Model, Privacy Impact Assessment (PIA), Product-Safety Hazard Log, Critical-Function Register, Incident, Stop, and Manual-Fallback Baseline** for **Milestone v0.4.0 - OSHE Inspect Private Alpha** within `OSHEThai/oshe-platform`. It fulfills the requirements and deliverable specifications of **GitHub Issue #145 (`[V040-I034] Create Threat Model, Privacy Assessment, Product-Safety Hazard Log, Critical-Function Register, Incident, Stop, and Manual-Fallback Procedures`)** under the governing authority of Sole Human Owner decision `HDEC-V040-FOUNDATION-054`.

### 1.2 Boundary and Scope
Milestone v0.4.0 operates as a narrow, standalone, single-tenant private-alpha vertical slice of the OSHE Inspect application (`H040-001`). All implementation and qualification prework is conducted strictly against local synthetic data fixtures and mock environments.

### 1.3 Deferred Human Authority and Invariant Non-Claims
In strict accordance with `HDEC-V040-FOUNDATION-054`, key operational and release authorities remain deferred to the **Sole Human Owner**:
1. **Gate `H040-009` (Binding Support and Manual-Fallback Ownership):** Remains on **`HOLD`**. This document specifies technical fallback architectures and procedures, but establishes **zero binding operational support ownership or staffing SLAs**.
2. **Gate `H040-011` (Final Milestone v0.4.0 Outcome and Residual-Risk Acceptance):** Remains on **`HOLD`**. This document provides residual-risk assessments and recommendations only; it enacts **zero residual-risk acceptance**.
3. **Gate `H040-007` (Technical Release Authorization):** Remains on **`HOLD`**.
4. **Gate `H040-008` (Real Participant / Private Alpha UAT Onboarding):** Remains on **`HOLD`**.
5. **Gate `H040-010` (External Environment, Route, Account, or Effect Activation):** Remains on **`HOLD`**.

---

## 2. V0.4 System Context & Role Architecture

### 2.1 Single-Tenant Vertical Slice Scope (`H040-001`)
The private alpha vertical slice implements end-to-end inspection lifecycle operations for a single synthetic organization, strictly excluding:
- Public internet endpoints or unauthenticated public routes.
- External enterprise integrations (ERP, HRMS, external IdP).
- Production or live customer data.
- Autonomous or closed-loop AI safety decision-making.

### 2.2 Authorized Role Matrix
Authorization adheres to a strict default-deny model (`H040-004`). Four distinct functional roles are defined:
1. **Checklist Author:** Creates, versions, and publishes structured inspection checklist templates with defined scoring rules.
2. **Inspector:** Executes assigned field inspections, records item responses (`PASS`, `FAIL`, `UNKNOWN`, `NOT_APPLICABLE`), captures evidence references, and logs safety findings.
3. **CAPA Owner:** Formulates, implements, and documents Corrective and Preventive Actions (CAPA) resolving non-conformances.
4. **Independent Reviewer:** Performs independent verification, review of findings, and formal closure of CAPAs and completed inspection records.

### 2.3 Environmental & Localization Parameters (`H040-002`)
- **Supported Browsers:** Current versions of desktop Google Chrome, Microsoft Edge, and Android Chrome.
- **Languages:** Bilingual support for English (`en-US`) and Thai (`th-TH`).
- **Time Zone:** Canonical operational time zone is `Asia/Bangkok` (UTC+7).

### 2.4 AI Non-Authority Invariant (`H040-004`)
Artificial intelligence models, agents, and automated algorithms possess **zero autonomous safety authority**. AI components may assist in summarizing draft findings or suggesting checklist mappings, but every state mutation, score calculation, non-conformance classification, and CAPA resolution requires explicit, logged human review and affirmation by an authorized role.

---

## 3. Threat Model (STRIDE & Abuse Scenarios)

The following threat catalog analyzes the primary threat scenarios identified in GitHub Issue #145 across the v0.4.0 inspection architecture:

| Threat ID | Threat Category | Threat / Attack Scenario | Attacker Vector & Mechanism | Prevention & Fail-Closed Mitigation | Governed Gate |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **`THR-01`** | **Tenant Leakage** | Cross-tenant data inspection or finding disclosure | Multi-tenant identifier tampering in synthetic object references | Strict single-tenant tenancy scoping; server-side enforcement of tenant foreign key constraints; cross-tenant references return generic non-leaking not-found errors. | `H040-001` |
| **`THR-02`** | **Privilege Escalation** | Unauthorized role assertion or lateral elevation | An Inspector attempts to self-verify findings or close assigned CAPAs | Server-validated RBAC; state transition engine rejects requests where caller role does not match required transition authority (`ErrUnauthorizedTransitionRole`). | `H040-004` |
| **`THR-03`** | **Malicious Evidence / QR** | Malicious payload injection via QR scanner or evidence upload | Malicious QR strings injected into asset scan; path traversal (`../`) in evidence attachment metadata | Strict payload schema validation; sanitization of scanned barcode/QR strings; whitelist MIME type enforcement; cryptographic content hashing. | `H040-003` |
| **`THR-04`** | **Sync Replay & Race** | Replay of superseded inspection mutations | Attacker or faulty client replays stale offline mutation batches to overwrite newer server state | Monotonic state versioning; server-enforced sequence numbers; stale mutations rejected with version mismatch (`ErrStateVersionConflict`). | `H040-005` |
| **`THR-05`** | **Stale Authority** | Exploitation of revoked session or expired role assignment | Client executes mutations using cached JWT/bearer tokens after administrative role revocation | Short-lived bearer token grants; generation-based session invalidation; server-side revocation registry queried on every mutating transition. | `H040-004` |
| **`THR-06`** | **Data Loss** | Client-side data loss during intermittent field connectivity | Device battery exhaustion or network drop mid-inspection before sync | Local IndexedDB transactional write-ahead journaling; local persistence before UI acknowledgement; atomic mutation batches. | `H040-005` |
| **`THR-07`** | **Evidence Tampering** | Post-inspection modification of photo or load-test evidence | Tampering with stored image bytes or falsifying certificate hashes | Deterministic SHA-256 calculation computed immediately upon capture; immutable evidence digests bound to inspection finding record. | `H040-003` |
| **`THR-08`** | **Misleading Pass** | Masking critical safety hazards beneath aggregate numeric scores | High non-critical pass rate mathematically obscuring an S0/S1 life-safety failure | Override rule: Any S0 or S1 critical finding automatically forces an overall inspection state of `UNSAFE_FAILED` regardless of weighted score. | `H040-006` |
| **`THR-09`** | **Lost Finding** | Silent omission or drop of non-conformance records | Network truncation or payload serialization error dropping negative items | End-to-end item count assertions; cryptographic payload manifest matching client-recorded findings to server-ingested findings. | `H040-005` |
| **`THR-10`** | **Unauthorized Closure** | Self-closure of corrective action by CAPA Owner | CAPA Owner marks corrective action closed without independent quality verification | Enforced segregation of duties (`SOD-CAPA-01`): Closure requires distinct `Independent Reviewer` role approval and signed verification evidence. | `H040-004` |
| **`THR-11`** | **Support / Fallback Failure** | Inability to transition to manual inspection during outage | Field inspector unable to operate due to software freeze or sync crash | Standardized physical paper inspection checklist format; offline local storage; explicit procedural fallback runbooks (ownership held under `H040-009`). | `H040-009` (HOLD) |

---

## 4. Privacy Impact Assessment (PIA) & Data Minimization

### 4.1 Synthetic Data Mandate (`H040-003`)
During Milestone v0.4.0 development and private-alpha testing, all data must remain **100% synthetic or redacted**:
- **Prohibited Data Classes:** Real employee personal data, national identification numbers, real contact details, real commercial facilities, real disciplinary records, and biometric data are strictly forbidden.
- **Synthetic Entity Patterns:** All test entities must use designated synthetic prefixes:
  - Users: `usr_syn_<role>_<id>`
  - Inspections: `ins_syn_<id>`
  - Assets / Locations: `ast_syn_<id>`, `loc_syn_<id>`
  - Findings: `fnd_syn_<id>`
  - Evidence: `evd_syn_<id>`

### 4.2 Local Device Storage Privacy & Security
- **Data Minimization in Browser Storage:** Local storage (IndexedDB / LocalStorage) is restricted to active inspection draft cache and checklist schemas.
- **Zero Raw Credentials:** Passwords, private keys, and long-lived refresh tokens must never be written to browser local storage.
- **Session Purge:** Termination of an inspection session or user logout must trigger deterministic clearing of sensitive cached inspection drafts.

---

## 5. Product-Safety Hazard Log (Severity S0 / S1 / S2 / S3)

To ensure inspection integrity and workforce safety, hazards arising from system operation or failure are classified and governed according to the following severity matrix:

```
┌─────────┬──────────────────┬─────────────────────────────────────────────────────────────┐
│ Severity│ Classification   │ Impact Definition                                           │
├─────────┼──────────────────┼─────────────────────────────────────────────────────────────┤
│   S0    │ Catastrophic     │ Fatal harm, acute life-threatening condition, or structural │
│         │                  │ collapse resulting from unflagged critical hazard.          │
├─────────┼──────────────────┼─────────────────────────────────────────────────────────────┤
│   S1    │ Critical         │ Severe injury, lost-time incident, or masked high-severity  │
│         │                  │ non-conformance in active industrial operating zone.        │
├─────────┼──────────────────┼─────────────────────────────────────────────────────────────┤
│   S2    │ Moderate         │ Minor injury, inaccurate scoring, delayed finding reporting,│
│         │                  │ or localized compliance defect without immediate danger.    │
├─────────┼──────────────────┼─────────────────────────────────────────────────────────────┤
│   S3    │ Minor            │ Cosmetic UI error, translation typo, or non-safety labeling │
│         │                  │ inconsistency carrying zero physical or compliance risk.    │
└─────────┴──────────────────┴─────────────────────────────────────────────────────────────┘
```

### 5.1 S0 & S1 Critical Hazard Controls

For all hazards classified as **S0** or **S1**, the system mandates the following multi-layer defense framework:

1. **Prevention:** 
   - Checklists must include explicit mandatory critical-item flags (`is_critical_safety_item = true`).
   - Inspectors cannot submit an inspection with uncompleted critical items.
2. **Detection:**
   - Immediate real-time client-side warning and severity tagging upon recording a failed critical item.
3. **Fail-Safe & Stop Behavior:**
   - **Automatic Inspection Fail-Closed:** A single S0/S1 finding overrides all scoring formulas and marks the inspection `UNSAFE_FAILED`.
   - **Stop-Work Escalation:** Triggers the immediate presentation of a "STOP WORK & REPORT" banner on the inspector device.
4. **Manual Fallback:**
   - If the electronic application crashes or hangs during an active critical hazard detection, the inspector is procedurally instructed to halt physical operations and notify the on-site Safety Officer directly.
5. **Release-Blocking Treatment:**
   - Any unresolved defect or regression in S0/S1 detection, scoring override, or finding logging is an absolute blocker for private-alpha qualification.

---

## 6. Critical-Function Register & Server Authority

### 6.1 Critical-Function Register
The following capabilities represent critical system functions whose failure directly impacts inspection validity:

| Function ID | Function Name | Failure Mode | Resilience & Integrity Mechanism |
| :--- | :--- | :--- | :--- |
| **`FN-01`** | **Checklist Definition & Versioning** | Inconsistent schema or altered questions mid-inspection | Checklists are immutable once published; inspections pin to exact `checklist_version_id` and content digest. |
| **`FN-02`** | **Response Capture & Scoring** | Score miscalculation or lost answer | Deterministic scoring algorithm; `UNKNOWN` and `NOT_APPLICABLE` handled explicitly; double-precision verification. |
| **`FN-03`** | **Finding & Severity Binding** | Severity downgrade or dropped finding | Server rejects finding creation without explicit severity (`S0`..`S3`); finding state immutable once submitted. |
| **`FN-04`** | **CAPA State Transition** | Unauthorized closure without verification | Multi-step state machine (`OPEN` -> `IN_PROGRESS` -> `RESOLVED` -> `VERIFIED_CLOSED`); separate role verification required. |
| **`FN-05`** | **Evidence Digest Binding** | Evidence detachment or hash substitution | Cryptographic binding of SHA-256 evidence digests directly into finding record payload. |
| **`FN-06`** | **Audit Trail Logging** | Tampering or missing audit records | Append-only chronological audit log entries with monotonic sequence numbers and caller attribution. |

### 6.2 Server Authority & Conflict Quarantine (`H040-005`)
To ensure state integrity in offline-capable field environments:
- **Server Authority:** The server remains the single authoritative source of truth for all persistent state. Client devices propose state mutations; the server validates and commits them.
- **Prohibition of Last-Write-Wins (LWW):** Blind last-write-wins resolution is strictly prohibited.
- **Conflict Quarantine:** When concurrent or out-of-sequence mutations are detected (e.g. concurrent edits from multiple inspectors), the server moves the affected inspection entity into a **`QUARANTINED`** state. Quarantined records are locked against further automated modification until manual reconciliation is executed by an authorized supervisory role.

---

## 7. Incident, Stop, and Manual-Fallback Baseline

### 7.1 Stop-Inspection Protocol
1. **Trigger:** Identification of an S0/S1 critical hazard or acute physical safety risk during an active inspection.
2. **Action:**
   - Inspector taps "LOG CRITICAL HAZARD & STOP INSPECTION".
   - Application marks inspection state as `HALTED_CRITICAL_HAZARD`.
   - Application displays immediate emergency contact and physical site evacuation guidance.
   - Digital findings are queued with high-priority synchronization flag.

### 7.2 Quarantine Protocol
1. **Trigger:** Mutation version mismatch, checksum divergence, or corrupted payload transmission.
2. **Action:**
   - Server rejects automated commit and marks record `STATUS_QUARANTINED`.
   - Audit alert logged: `AUDIT_SYNC_CONFLICT_QUARANTINED`.
   - Inspector receives non-destructive alert indicating inspection has been preserved for administrative review.
   - Client cache remains intact to prevent local data loss.

### 7.3 Manual Fallback Architecture
In the event of device failure, power outage, or total software unresponsiveness:
1. **Physical Standardized Fallback:** Standardized physical paper checklists matching the versioned digital schema (`DOC-SYN-FALLBACK-001`) must be maintained on-site.
2. **Post-Restoration Reconciliation:** Once digital services are restored, manual records are transcribed into the system by an authorized Inspector and verified by an Independent Reviewer with explicit provenance tagging (`source: MANUAL_PAPER_FALLBACK`).
3. **Governance Reservation (`H040-009` HOLD):** Operational support agreements, SLA response times, staffing schedules, and physical logistics for manual fallback remain explicitly on **`HOLD` under Gate `H040-009`**.

---

## 8. Residual-Risk Assessment & Governance Non-Claims

### 8.1 Residual Risk Recommendations
The following technical and operational risks remain inherent in the Milestone v0.4.0 alpha architecture:
1. **Local-Only Offline Simulation:** Current qualification relies on in-memory and synthetic mocks. Real-world network drops and physical device hardware quirks remain unvalidated pending future staging spikes.
2. **Manual Quarantine Overhead:** Rejection of LWW in favor of quarantine prevents data loss, but requires human administrator overhead to resolve high-frequency concurrent edits.
3. **Unexercised Operational Support:** Until Gate `H040-009` is decided by the Sole Human Owner, no formal helpdesk or on-call rotation exists.

### 8.2 Operational Non-Claims & Retained Gate Declaration
In compliance with `HDEC-V040-FOUNDATION-054`:
- **Zero Release Recommendation or Approval:** This baseline does **NOT** approve or authorize a release of OSHE Inspect.
- **Zero Real-User Authorization:** Gate `H040-008` remains on `HOLD`; no real users may be recruited, onboarded, or granted system access.
- **Zero Residual-Risk Acceptance:** In accordance with Gate `H040-011` (`HOLD`), acceptance of residual operational or safety risk is strictly reserved for the Sole Human Owner.
- **Zero Operational Support Ownership:** In accordance with Gate `H040-009` (`HOLD`), no binding operational support ownership or SLA commitments are enacted by this document.
