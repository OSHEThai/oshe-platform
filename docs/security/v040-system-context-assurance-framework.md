---
document_id: SEC-V040-ASR-001
title: v0.4.0 System Context, Trust Boundary, Data Flow, Tenant Isolation, Offline, Evidence, and Safety Assurance Framework
document_type: security_and_safety_assurance_framework
document_version: 1.0.0
lifecycle_status: DRAFT
status: APPROVED_FOR_LOCAL_ALPHA_DEVELOPMENT
date: "2026-09-05"
author_role: Security Privacy and Product Safety Lead
author_pane: w9:p13
governing_issue: "GitHub Issue #144"
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
unresolved_dependencies:
  - V040-I013
  - V040-I018
  - V040-I021
  - V040-I027
  - V040-I029
pending_dependency_status: PENDING_DEPENDENCY
credit_boundary: ASSURANCE_FRAMEWORK_ONLY_NO_OPERATIONAL_OR_RELEASE_CLAIM
---

# v0.4.0 System Context, Trust Boundary, Data Flow, Tenant Isolation, Offline, Evidence, and Safety Assurance Framework

## 1. Executive Summary & Governance Reference

### 1.1 Authority Baseline & Purpose
This framework establishes the authoritative **System Context, Trust Boundary, Data Flow, Tenant Isolation, Offline, Evidence, and Safety Assurance Case Framework** for **Milestone v0.4.0 - OSHE Inspect Private Alpha** within `OSHEThai/oshe-platform`. It fulfills the requirements and deliverable specifications of **GitHub Issue #144 (`[V040-I033] Create v0.4.0 System Context, Trust Boundary, Data Flow, Offline, Evidence, Tenant-Isolation, and Safety Assurance Case`)** under the governing authority of Sole Human Owner decision `HDEC-V040-FOUNDATION-054`.

The primary objective is to link each client, device, identity, organization, API, module, database, object, background, report, export, support, and offline boundary to explicit preventive controls, detective controls, negative tests, verification evidence, accountable owner placeholders, and release gates.

### 1.2 Unresolved Feature Dependencies (`PENDING_DEPENDENCY`)
In strict adherence to assignment directives, this framework models the complete architectural assurance topology while explicitly preserving all unfinished feature-dependent coverage as **`PENDING_DEPENDENCY`**:
- **`V040-I013` (Checklist Authoring & Versioning Engine):** `PENDING_DEPENDENCY`
- **`V040-I018` (Mobile/Offline Inspection Execution Engine):** `PENDING_DEPENDENCY`
- **`V040-I021` (Evidence Capture, Media Hash & Manifest Service):** `PENDING_DEPENDENCY`
- **`V040-I027` (Finding Generation, Critical Stop-Work & CAPA Lifecycle):** `PENDING_DEPENDENCY`
- **`V040-I029` (Report Synthesis, Summary Export & Audit Journal Preservation):** `PENDING_DEPENDENCY`

Issue #144 remains open following this delivery until all named feature dependencies deliver their respective implementation and qualification evidence.

### 1.3 Deferred Human Authority & Operational Non-Claims
In strict compliance with `HDEC-V040-FOUNDATION-054`, the following governance gates remain permanently on **`HOLD`**:
- **Gate `H040-007` (Technical Release Authorization):** `HOLD`. No technical release approval is granted.
- **Gate `H040-008` (Real Participant / Private Alpha UAT Onboarding):** `HOLD`. No real users, external participants, or customer pilots are authorized.
- **Gate `H040-009` (Binding Support and Manual-Fallback Ownership):** `HOLD`. No operational helpdesk staffing, binding response times, or live on-call rotations are enacted.
- **Gate `H040-010` (External Environment, Route, Account, or Effect Activation):** `HOLD`. Zero cloud deployment, public routing, or external provider mutation is authorized.
- **Gate `H040-011` (Final Milestone v0.4.0 Outcome and Residual-Risk Acceptance):** `HOLD`. Residual operational and safety risk acceptance is strictly reserved for the Sole Human Owner.

**Operational Non-Claim:** No technical assurance claim in this framework is based on simulated user success or automated happiness metrics. Assurance is grounded strictly in deterministic verification, negative test validation, and tamper-evident cryptographic controls.

---

## 2. V0.4 System Context & Component Architecture

### 2.1 Component Topology & Boundaries
The Milestone v0.4.0 architecture operates as a narrow, standalone, single-tenant private-alpha vertical slice (`H040-001`). The system is partitioned into the following physical and logical components:

```
┌─────────────────────────────────────────────────────────────────────────────────────────────────┐
│ CLIENT ZONE (Browser Sandbox - Desktop Chrome/Edge, Android Chrome Mobile)                      │
│                                                                                                 │
│  ┌──────────────────────┐   ┌──────────────────────┐   ┌─────────────────────────────────────┐  │
│  │ UI Presentation Layer│   │ Local State Manager  │   │ Offline Persistence Storage         │  │
│  │ (Bilingual en/th)    │ ──│ (Optimistic Journal) │ ──│ (IndexedDB / SessionStorage)       │  │
│  └──────────────────────┘   └──────────────────────┘   └─────────────────────────────────────┘  │
└──────────────────────────────────────────────┬──────────────────────────────────────────────────┘
                                               │
                                       [TRUST BOUNDARY TB-01]
                                 (Client Webview / Host Sandbox)
                                               │
                                       [TRUST BOUNDARY TB-02]
                               (Local Network / Loopback Transport)
                                               │
┌──────────────────────────────────────────────▼──────────────────────────────────────────────────┐
│ SERVER ZONE (Central Application Service - Single-Tenant Private Alpha)                         │
│                                                                                                 │
│  ┌───────────────────────────────────────────────────────────────────────────────────────────┐  │
│  │ Gateway & Reverse Proxy: TLS Termination, Host Header Check, Content-Type Whitelisting    │  │
│  └───────────────────────────────────────────┬───────────────────────────────────────────────┘  │
│                                              │                                                  │
│                                      [TRUST BOUNDARY TB-03]                                     │
│                             (API Ingress & Default-Deny RBAC)                                   │
│                                              │                                                  │
│  ┌───────────────────────────────────────────▼───────────────────────────────────────────────┐  │
│  │ Core Domain Modules (Zero Direct Cross-Module DB Access):                                 │  │
│  │  - MOD-CFG: Checklist Configuration & Versioning (Owns ChecklistTemplate)                 │  │
│  │  - MOD-WFA: Workflow, Inspection & CAPA (Owns InspectionSession, Finding, ActionItem)     │  │
│  │  - MOD-EVD: Evidence & Attachment Registry (Owns EvidenceRecord, SHA-256 Manifest)        │  │
│  │  - MOD-REP: Reporting, Projections & Localization (Owns InspectionReport, Views)          │  │
│  │  - MOD-REC: Append-Only Audit & Transition Journal (Owns LifecycleAuditRecord)            │  │
│  │  - MOD-IAM: Identity, Session & Role Authority (Owns Subject, RoleGrant, Revocation)      │  │
│  │  - MOD-ORG: Single-Tenant Organization Hierarchy (Owns Tenant, Site, Area)               │  │
│  └───────────────────────────────────────────┬───────────────────────────────────────────────┘  │
│                                              │                                                  │
│                                      [TRUST BOUNDARY TB-04]                                     │
│                            (Tenant & Module Isolation Boundary)                                 │
│                                              │                                                  │
│  ┌───────────────────────────────────────────▼───────────────────────────────────────────────┐  │
│  │ Persistence & Media Store:                                                               │  │
│  │  - Relational Database: Scoped by tenant_id foreign keys, monotonic state versioning      │  │
│  │  - Evidence Blob Store: Content-addressed storage keyed by SHA-256 digest                │  │
│  │  - Audit Journal: Append-only chronological event ledger with cryptographic hash seals    │  │
│  └───────────────────────────────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────────────────────────────┘
```

### 2.2 System Context Boundaries Definition

| Boundary Identifier | Boundary Name | Component Interfaces Separated | Inherent Risk & Primary Hazard | Baseline Preventive Control |
| :--- | :--- | :--- | :--- | :--- |
| **`TB-01`** | **Client Sandbox Boundary** | Browser DOM / JS Runtime vs Client Operating System & Local Storage | DOM injection, malicious script execution, token extraction from storage. | Strict Content Security Policy (CSP), zero credentials in localStorage, ephemeral IndexedDB drafts zeroized on logout. |
| **`TB-02`** | **Transport Boundary** | Client Browser Runtime vs Central Application Gateway | Man-in-the-Middle (MitM) inspection, cleartext snooping, packet replay. | TLS 1.3 / HTTPS only, strict transport security headers, monotonic sequence counters against replay attacks. |
| **`TB-03`** | **API Ingress & RBAC Boundary** | Untrusted Client Request vs Protected Application Domain Modules | Unauthorized state transition, parameter tampering, privilege escalation. | Server-enforced Default-Deny RBAC (`H040-004`), strong schema validation, rejection of unauthorized caller roles. |
| **`TB-04`** | **Tenant & Module Isolation Boundary** | Tenant Data Boundaries & Inter-Module Domain Encapsulation | Cross-tenant data leakage (`THR-01`), cross-module direct database contamination. | Strict single-tenant scoping, tenant_id verification on every SQL predicate, inter-module communication exclusively via versioned contracts. |
| **`TB-05`** | **Evidence & Media Storage Boundary** | Application Logic vs File/Media Storage Subsystem | Path traversal (`../`), malicious payload execution, evidence substitution (`THR-03`, `THR-07`). | SHA-256 content-addressing, strict MIME-type allowlist, file extension validation, media immutability. |
| **`TB-06`** | **Audit & State Integrity Boundary** | State Machine Transitions vs Append-Only Audit Journal | Tampering with finding status, retroactive score alteration, silent finding deletion. | Append-only audit logging (`MOD-REC`), composite SHA-256 transition digests, server authority for protected state (`H040-005`). |

---

## 3. Data Flow Architecture & Boundaries

### 3.1 Primary Operational Data Flows
The following lifecycle data flows span client, gateway, domain modules, and persistence stores:

```
[Inspector Device] ──(1. Download Checklist)──> [MOD-CFG: Checklist Service]
       │
       ├──(2. Execute Offline Inspection)──> [Local IndexedDB Journal]
       │
       ├──(3. Capture Evidence & Compute SHA-256)──> [Local Media Cache]
       │
       ├──(4. Submit Inspection & Findings Batch)──> [Gateway / TB-03]
                                                             │
                                                             ▼
                                                [MOD-WFA: State Machine]
                                                             │
                              ┌──────────────────────────────┼──────────────────────────────┐
                              │ (Version Match)              │ (Version Conflict)           │ (Critical S0/S1 Finding)
                              ▼                              ▼                              ▼
                 [Commit Monotonic State]       [Quarantine Conflict]          [Trigger Halt & Alert]
                              │                   (Lock for Supervisor)          (Force UNSAFE_FAILED)
                              ▼                              │                              │
                 [MOD-REC: Audit Journal]                    ▼                              ▼
                 (Append Immutable Seal)        [Emit Conflict Event]          [Emergency Stop Banner]
```

### 3.2 Detailed Data Flow Specification

1. **Checklist Distribution Flow (`DF-01`):**
   - *Source:* `MOD-CFG` (Authoritative Checklist Template).
   - *Destination:* Inspector client cache (IndexedDB).
   - *Security Controls:* Pinned `checklist_version_id`, cryptographic payload digest, read-only cache protection.
   - *Status:* `PENDING_DEPENDENCY` (`V040-I013`).

2. **Field Execution & Response Recording Flow (`DF-02`):**
   - *Source:* Inspector user input (touch/keyboard).
   - *Destination:* Client IndexedDB transactional journal.
   - *Security Controls:* Atomic write-ahead journal before UI confirmation, explicit answer state enum (`PASS`, `FAIL`, `UNKNOWN`, `NOT_APPLICABLE`).
   - *Status:* `PENDING_DEPENDENCY` (`V040-I018`).

3. **Evidence Capture & Hashing Flow (`DF-03`):**
   - *Source:* Client camera / file picker.
   - *Destination:* `MOD-EVD` Evidence Registry and Object Store.
   - *Security Controls:* Client-side SHA-256 digest computation immediately upon capture, sanitization of metadata, server-side re-hashing verification.
   - *Status:* `PENDING_DEPENDENCY` (`V040-I021`).

4. **Mutation Synchronization & Conflict Resolution Flow (`DF-04`):**
   - *Source:* Inspector client mutation queue.
   - *Destination:* `MOD-WFA` Workflow Engine.
   - *Security Controls:* Optimistic concurrency check (`base_version == current_version`), zero last-write-wins (`H040-005`), automatic routing of mismatched submissions to `QUARANTINED` container.
   - *Status:* `PENDING_DEPENDENCY` (`V040-I018`).

5. **Critical Finding & Stop-Work Escalation Flow (`DF-05`):**
   - *Source:* Inspector logs critical non-conformance (`S0` or `S1`).
   - *Destination:* `MOD-WFA` Finding Engine & Client UI Banner.
   - *Security Controls:* Immediate scoring override to `UNSAFE_FAILED`, presentation of emergency physical stop-work instructions, unsuppressible finding record.
   - *Status:* `PENDING_DEPENDENCY` (`V040-I027`).

6. **Review Verification & Finding Closure Flow (`DF-06`):**
   - *Source:* `Independent Reviewer` role submission.
   - *Destination:* `MOD-WFA` CAPA Engine & `MOD-REC` Audit Log.
   - *Security Controls:* Segregation of duties (`SOD-CAPA-01`: Inspector/CAPA Owner cannot self-close), verified evidence attachment requirement, cryptographic audit seal.
   - *Status:* `PENDING_DEPENDENCY` (`V040-I027`).

7. **Report Synthesis & Historical Export Flow (`DF-07`):**
   - *Source:* `MOD-REP` Reporting Engine.
   - *Destination:* Reviewer/Auditor presentation view.
   - *Security Controls:* Read-only projection from immutable audit records, watermark metadata, synthetic data verification.
   - *Status:* `PENDING_DEPENDENCY` (`V040-I029`).

---

## 4. Tenant, Project, Contractor, Offline, and Evidence Control Matrix

The following control matrix defines the specific technical and procedural boundaries governing tenancy, hierarchy, contractor access, offline resilience, and evidence custody:

```
┌─────────────────────────┬───────────────────────────────────────────────────┬────────────────────────────────────────────────────────┐
│ Dimension               │ Preventive Control                                │ Detective / Verification Control                       │
├─────────────────────────┼───────────────────────────────────────────────────┼────────────────────────────────────────────────────────┤
│ Tenant Isolation        │ Strict single-tenant tenancy scoping (H040-001).  │ SQL assertion: all queries enforce WHERE tenant_id = ? │
│                         │ Zero multi-tenant routing in private alpha.       │ Synthetic ID prefix validation (usr_syn_, ins_syn_).   │
├─────────────────────────┼───────────────────────────────────────────────────┼────────────────────────────────────────────────────────┤
│ Organization Hierarchy  │ 6-level hierarchy scoping: Tenant -> Company ->   │ Foreign key tree validation on inspection creation;    │
│                         │ BusinessUnit -> Project -> Site -> Area.          │ area containment assertions.                           │
├─────────────────────────┼───────────────────────────────────────────────────┼────────────────────────────────────────────────────────┤
│ Contractor Boundaries   │ Sponsored party assignments restricted to narrow │ Negative test: contractor cannot query unassigned      │
│                         │ assigned site/checklist scope. Zero author rights.│ sites, edit templates, or access financial data.       │
├─────────────────────────┼───────────────────────────────────────────────────┼────────────────────────────────────────────────────────┤
│ Offline Integrity       │ Write-ahead local IndexedDB journaling; zero LWW; │ Version token comparison; collision quarantine alert   │
│                         │ 24-hr maximum offline window [NON-BINDING_PROP].  │ emitted upon out-of-order sequence detection.          │
├─────────────────────────┼───────────────────────────────────────────────────┼────────────────────────────────────────────────────────┤
│ Evidence Custody        │ Deterministic SHA-256 calculation upon capture;   │ Server-side hash verification on upload; payload       │
│                         │ immutable blob binding; strict MIME validation.   │ manifest count matching client recorded items.         │
└─────────────────────────┴───────────────────────────────────────────────────┴────────────────────────────────────────────────────────┘
```

---

## 5. Safety Claims & Assurance Argument (Goal Structuring Notation)

To establish a defensible safety and security assurance case, system claims are structured under explicit top-level goals, backed by architectural arguments and verification evidence:

### Top-Level Safety & Integrity Goals

#### Goal G1: Fail-Closed Protection Against Masked Hazards
- **Claim G1:** *The OSHE Inspect application prevents masked life-safety hazards and enforces fail-closed state on critical failures (S0/S1).*
- **Argument A1.1:** Any finding categorized as S0 (Catastrophic) or S1 (Critical) programmatically overrides weighted numerical scores, forcing the overall inspection outcome to `UNSAFE_FAILED`.
- **Argument A1.2:** Uncompleted critical safety items block inspection finalization; an inspector cannot bypass mandatory safety questions.
- **Evidence E1.1:** Negative test suite asserting score override behavior when S0/S1 finding is present (`test_s0_s1_critical_override_forces_failed`).
- **Gating Status:** `PENDING_DEPENDENCY` (`V040-I027`).

#### Goal G2: Server Authority & Zero Data Loss Across Offline States
- **Claim G2:** *The system guarantees server authority and eliminates data loss, race conditions, or silent corruption across offline and intermittent network states.*
- **Argument A2.1:** All state mutations are validated against monotonic entity versions; blind last-write-wins (LWW) is categorically prohibited (`H040-005`).
- **Argument A2.2:** Mismatched or conflicting concurrent submissions automatically move into `QUARANTINED` status for supervisory human resolution.
- **Evidence E2.1:** Concurrency collision test asserting rejection of stale version and creation of `ConflictRecord` (`test_concurrency_conflict_routes_to_quarantine`).
- **Gating Status:** `PENDING_DEPENDENCY` (`V040-I018`).

#### Goal G3: Enforced Segregation of Duties & Default-Deny Authority
- **Claim G3:** *The system enforces default-deny role separation, preventing unauthorized finding closure, template alteration, or review bypass.*
- **Argument A3.1:** Checklist Author, Inspector, CAPA Owner, and Independent Reviewer roles possess strictly non-overlapping transition authority (`H040-004`).
- **Argument A3.2:** Corrective action closure requires independent review verification (`SOD-CAPA-01`); self-closure by CAPA Owner is rejected.
- **Evidence E3.1:** RBAC matrix negative tests asserting rejection of unauthorized transitions (`test_capa_owner_cannot_verify_closure`).
- **Gating Status:** `PENDING_DEPENDENCY` (`V040-I002`, `V040-I027`).

#### Goal G4: Synthetic-Only Privacy Boundary & Storage Minimization
- **Claim G4:** *The system strictly maintains synthetic-only privacy boundaries, zero customer data leakage, and local storage minimization.*
- **Argument A4.1:** All entities conform to synthetic prefixes (`usr_syn_*`, `ins_syn_*`); production data and real PII are strictly barred (`H040-003`).
- **Argument A4.2:** Browser storage never persists credentials or private keys; cached inspection drafts are zeroized on session termination.
- **Evidence E4.1:** Synthetic dataset pattern validator asserting zero real PII across test fixtures (`test_synthetic_data_sanitization`).
- **Gating Status:** Verified under `ARC-V040-PROF-001` and `SEC-V040-BASE-001`.

#### Goal G5: Zero AI Autonomous Safety Authority
- **Claim G5:** *Artificial intelligence models possess zero autonomous safety decision authority.*
- **Argument A5.1:** AI components are restricted to advisory text summarization and schema assistance.
- **Argument A5.2:** No safety score, non-conformance severity, or CAPA closure transition can be authorized without explicit human affirmative input.
- **Evidence E5.1:** Architecture and policy invariant test asserting AI autonomy lock (`test_zero_ai_safety_autonomy_enforced`).
- **Gating Status:** Verified under `HDEC-V040-FOUNDATION-054` and `ARC-V040-DOMAIN-001`.

---

## 6. Negative-Test Map & Boundary Stress Matrix

The following catalog maps threats and boundary hazards to explicit negative test specifications:

| Negative Test ID | Target Threat / Hazard | Targeted Boundary | Injected Fault / Malicious Action | Expected Fail-Closed Behavior | Status |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **`NEG-TEST-01`** | `THR-01` (Tenant Leak) | `TB-04` (Tenant) | Inspector attempts to query inspection using cross-tenant ID | Request rejected with generic 404 Not Found; zero cross-tenant disclosure. | `PENDING_DEPENDENCY` (`V040-I018`) |
| **`NEG-TEST-02`** | `THR-02` (Escalation) | `TB-03` (RBAC) | Inspector submits `VERIFIED_CLOSED` mutation for own finding | Transition rejected (`ErrUnauthorizedTransitionRole`); audit violation logged. | `PENDING_DEPENDENCY` (`V040-I027`) |
| **`NEG-TEST-03`** | `THR-03` (Bad QR/Media) | `TB-05` (Media) | Uploading executable or file with directory traversal (`../../mal.sh`) | Upload rejected by MIME/extension whitelist; path stripped; 400 Bad Request. | `PENDING_DEPENDENCY` (`V040-I021`) |
| **`NEG-TEST-04`** | `THR-04` (Replay/Race) | `TB-03` (API) | Submitting mutation batch with stale `base_version` (v1 vs server v2) | Submission rejected; record moved to `QUARANTINED`; conflict event emitted. | `PENDING_DEPENDENCY` (`V040-I018`) |
| **`NEG-TEST-05`** | `THR-05` (Stale Token) | `TB-03` (API) | Executing inspection transition using revoked session token | Server rejects token against revocation registry; 401 Unauthorized. | `PENDING_DEPENDENCY` (`V040-I018`) |
| **`NEG-TEST-06`** | `THR-07` (Tampered Evd) | `TB-05` (Media) | Uploading media whose byte hash does not match client SHA-256 | Upload rejected (`ErrEvidenceDigestMismatch`); transaction aborted. | `PENDING_DEPENDENCY` (`V040-I021`) |
| **`NEG-TEST-07`** | `THR-08` (Masked Pass) | `MOD-WFA` (Logic) | 99% score with a single S0 Catastrophic safety finding | Overall status forced to `UNSAFE_FAILED`; score calculation suppressed. | `PENDING_DEPENDENCY` (`V040-I027`) |
| **`NEG-TEST-08`** | `THR-09` (Lost Finding) | `TB-03` (API) | Client payload omits finding item present in local journal | Server detects manifest count discrepancy; sync batch aborted. | `PENDING_DEPENDENCY` (`V040-I018`) |
| **`NEG-TEST-09`** | `THR-10` (Self-Closure)| `TB-03` (RBAC) | CAPA Owner attempts to close finding without Reviewer sign-off | Rejection with `ErrSegregationOfDutiesViolation`; audit alert generated. | `PENDING_DEPENDENCY` (`V040-I027`) |
| **`NEG-TEST-10`** | `H040-003` (PII Leak) | `TB-01` (Client) | Storing unredacted personal telephone number or citizen ID | Client validator blocks input; logs redaction violation. | `PENDING_DEPENDENCY` (`V040-I018`) |

---

## 7. Control & Evidence Matrix with Accountable Owner Placeholders

Every critical system boundary is assigned explicit controls, verification sources, and accountable role placeholders:

| Boundary ID | Target Threat / Hazard | Preventive Control | Detective Control | Negative Test Harness | Verification Evidence Source | Accountable Owner Placeholder | Release Gate |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **`TB-01`** | Client XSS / Storage Tamper | CSP headers; zero raw tokens in storage; session purge | Content security violation logging; session watchdog | `NEG-TEST-10` | Browser storage unit suite | `[OWNER-CLIENT-LEAD]` | `H040-002` |
| **`TB-02`** | Transport Snooping / Replay | TLS 1.3 only; monotonic client sequence counters | Network ingress inspection; replay detector | Automated replay test | Network qualification report | `[OWNER-INFRA-HOLD]` | `H040-010` (HOLD) |
| **`TB-03`** | Privilege Escalation / Tamper | Server-side default-deny RBAC; strong parameter validation | State machine assertion logging; audit trail | `NEG-TEST-02`, `NEG-TEST-09` | RBAC regression test suite | `[OWNER-SEC-LEAD]` | `H040-004` |
| **`TB-04`** | Cross-Tenant Leakage | Scoped tenant_id foreign keys; query predicate enforcement | Cross-tenant query monitor; audit anomaly alert | `NEG-TEST-01` | Multi-tenant isolation test | `[OWNER-ARCH-LEAD]` | `H040-001` |
| **`TB-05`** | Media Tampering / Traversal | SHA-256 content addressing; MIME allowlist; path sanitize | Server-side hash re-verification; digest monitor | `NEG-TEST-03`, `NEG-TEST-06` | Evidence integrity test | `[OWNER-DATA-LEAD]` | `H040-003` |
| **`TB-06`** | Audit Tampering / State Drift | Append-only audit journal; SHA-256 seal; zero LWW | Cryptographic ledger validation; quarantine monitor | `NEG-TEST-04`, `NEG-TEST-08` | Audit journal proof suite | `[OWNER-QA-LEAD]` | `H040-005` |

---

## 8. Unresolved Feature Dependencies Register (`PENDING_DEPENDENCY`)

The following register details the five unresolved feature dependencies whose implementation and test suites are required before full assurance case closure can be claimed:

```
┌───────────────┬──────────────────────────────────────────┬────────────────────────────────────────────────────────┐
│ Issue Number  │ Feature Domain & Required Deliverable    │ Status & Gating Condition                              │
├───────────────┼──────────────────────────────────────────┼────────────────────────────────────────────────────────┤
│ V040-I013     │ Checklist Authoring, Versioning & Schema │ PENDING_DEPENDENCY: Template immutability, scoring     │
│               │ Definition Engine                        │ rules, and question hierarchy evidence required.       │
├───────────────┼──────────────────────────────────────────┼────────────────────────────────────────────────────────┤
│ V040-I018     │ Mobile/Offline Field Inspection Engine & │ PENDING_DEPENDENCY: Offline IndexedDB journaling, sync │
│               │ State Synchronization Service            │ batching, and conflict quarantine evidence required.   │
├───────────────┼──────────────────────────────────────────┼────────────────────────────────────────────────────────┤
│ V040-I021     │ Evidence Capture, SHA-256 Manifest, and  │ PENDING_DEPENDENCY: Client hashing, attachment binding,│
│               │ Media Registry Service                   │ and MIME sanitization evidence required.               │
├───────────────┼──────────────────────────────────────────┼────────────────────────────────────────────────────────┤
│ V040-I027     │ Finding Generation, S0/S1 Stop-Work,     │ PENDING_DEPENDENCY: Score override, stop banner, and   │
│               │ and Corrective Action (CAPA) Engine      │ segregation of duties closure evidence required.       │
├───────────────┼──────────────────────────────────────────┼────────────────────────────────────────────────────────┤
│ V040-I029     │ Inspection Report Synthesis, Summary     │ PENDING_DEPENDENCY: Bilingual report generation and    │
│               │ Export & Audit Preservation Service      │ immutable audit record preservation evidence required. │
└───────────────┴──────────────────────────────────────────┴────────────────────────────────────────────────────────┘
```

### Detailed Dependency Gap Analysis:

1. **`V040-I013` (Checklist Authoring Engine):**
   - *Assurance Gap:* Authoring UI, JSON schema validator, and published template sealing are not yet implemented.
   - *Assurance Requirement:* Must prove that published checklists are permanently immutable and that modifications generate versioned successor records.
   - *State:* **`PENDING_DEPENDENCY`**.

2. **`V040-I018` (Inspection Execution & Sync Engine):**
   - *Assurance Gap:* Mobile responsive inspection runner and offline synchronization queue are not yet implemented.
   - *Assurance Requirement:* Must prove zero last-write-wins (LWW), deterministic conflict quarantine, and zero local data loss upon network restoration.
   - *State:* **`PENDING_DEPENDENCY`**.

3. **`V040-I021` (Evidence Registry & Media Service):**
   - *Assurance Gap:* Media capture pipeline, client-side SHA-256 generation, and attachment binding are not yet implemented.
   - *Assurance Requirement:* Must prove that media cannot be detached, replaced, or tampered with once bound to a non-conformance record.
   - *State:* **`PENDING_DEPENDENCY`**.

4. **`V040-I027` (Finding & CAPA Lifecycle Engine):**
   - *Assurance Gap:* Non-conformance logging, S0/S1 score override triggers, and reviewer verification workflows are not yet implemented.
   - *Assurance Requirement:* Must prove that a single catastrophic/critical finding overrides numerical scores to `UNSAFE_FAILED` and that CAPA Owners cannot self-close actions.
   - *State:* **`PENDING_DEPENDENCY`**.

5. **`V040-I029` (Report Synthesis & Audit Journal Service):**
   - *Assurance Gap:* Bilingual report compiler and tamper-evident audit projection views are not yet implemented.
   - *Assurance Requirement:* Must prove historical reproducibility of completed inspection records from append-only audit entries.
   - *State:* **`PENDING_DEPENDENCY`**.

---

## 9. Retained Human Gates, Operational Prohibitions & Non-Claims

### 9.1 Retained Human Gate Register
In strict accordance with Sole Human Owner decision `HDEC-V040-FOUNDATION-054`, the following governance gates remain strictly on **`HOLD`**:

| Gate ID | Gate Title | Current Status | Prerequisite for Human Action |
| :--- | :--- | :--- | :--- |
| **`H040-007`** | **Technical Release Authorization** | **`HOLD`** | Completion of all feature dependencies (`I013`, `I018`, `I021`, `I027`, `I029`) and verification evidence suites. |
| **`H040-008`** | **Real Participant / Private-Alpha UAT** | **`HOLD`** | Explicit human selection, ethics review, and onboarding authorization of real test participants. |
| **`H040-009`** | **Binding Support & Fallback Ownership** | **`HOLD`** | Formal operational support staffing agreements, helpdesk escalation paths, and manual fallback runbook ownership. |
| **`H040-010`** | **External Environment Activation** | **`HOLD`** | Cloud infrastructure provisioning, public domain registration, external IdP linkage, and live storage bucket activation. |
| **`H040-011`** | **Final v0.4 Outcome & v0.5 Entry** | **`HOLD`** | Milestone completion review, residual risk acceptance, and transition approval by the Sole Human Owner. |

### 9.2 Strict Operational Prohibitions
1. **Zero Public Route or External Exposure:** No public internet routes, DNS hostnames, ingress proxies, or cloud CDN endpoints are authorized.
2. **Zero Production Data:** Real corporate records, employee personal identifiable information (PII), or physical facility blueprints remain strictly forbidden (`H040-003`).
3. **No Automated Issue Closure:** Pull requests delivering this assurance framework must **not** contain automatic Issue-closing keywords (e.g. "Closes #144", "Fixes #144"). Issue #144 must remain open until all dependent feature evidence is integrated.
4. **Zero Simulated-User Assurance Claims:** No claim of safety, reliability, or compliance may be asserted based on simulated user journeys or mock satisfaction scores.
5. **Zero Autonomous AI Decision Authority:** AI models remain strictly supporting tools; all safety judgments require explicit human affirmation (`H040-004`).
