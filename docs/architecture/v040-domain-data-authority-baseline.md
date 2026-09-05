---
document_id: ARC-V040-DOMAIN-001
title: v0.4.0 OSHE Inspect Domain, Data Authority, Workflow State, and Protected Business-Rule Baseline
document_type: architecture_specification
document_version: 1.0.0
lifecycle_status: DRAFT
status: APPROVED_FOR_LOCAL_ALPHA_DEVELOPMENT
date: "2026-09-05"
author_role: Architecture and Data Lead
author_pane: w9:p22
governing_issue: "GitHub Issue #113"
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
deferred_human_gates:
  - H040-007
  - H040-008
  - H040-009
  - H040-010
  - H040-011
retained_unselected_policies:
  binding_scoring_policy: HUMAN_OWNED_UNSELECTED
  finding_closure_policy: HUMAN_OWNED_UNSELECTED
  offline_authority: HUMAN_OWNED_UNSELECTED
credit_boundary: ARCHITECTURE_SPECIFICATION_ONLY_NO_IMPLEMENTATION_OR_RUNTIME_CREDIT
---

# v0.4.0 OSHE Inspect Domain, Data Authority, Workflow State, and Protected Business-Rule Baseline

## 1. Executive Summary & Governance Posture

### 1.1 Authority Baseline & Purpose
This architectural specification establishes the authoritative **Domain, Data Authority, Workflow State, and Protected Business-Rule Baseline** for **Milestone v0.4.0 - OSHE Inspect Private Alpha** in accordance with GitHub Issue #113 (`V040-I002`) and standing owner foundation decision `HDEC-V040-FOUNDATION-054`.

The primary objective is to define an integrated, dependency-free, bounded domain model across checklist configuration, inspection scheduling, assignment dispatch, evidence capture, findings, corrective actions (CAPA), scoring calculations, and reporting for a standalone, single-tenant private alpha vertical slice.

### 1.2 Retained Human-Owned Policies (Explicitly Unselected)
In strict compliance with `ASN-V040-I002-DOMAIN-AUTHORITY-001` and `HDEC-V040-FOUNDATION-054`, the following policy areas are formally designated as **`HUMAN_OWNED_UNSELECTED`** and are not resolved or enacted by this engineering specification:
1. **Binding Scoring Policy (`HUMAN_OWNED_UNSELECTED`):** Final scoring algorithms, section weights, passing thresholds, critical-fail triggers, and regulatory compliance ratings require explicit human owner decision under Issue #136 (`V040-I025`).
2. **Finding Closure Policy (`HUMAN_OWNED_UNSELECTED`):** Regulatory sign-off criteria, residual safety risk acceptance, and formal finding closure authorization rules require explicit human owner decision under Issue #134 (`V040-I023`).
3. **Offline Authority Model (`HUMAN_OWNED_UNSELECTED`):** Client-side lease durations, offline download expiration limits, and conflict priority policies require explicit human owner decision under Issue #126 (`V040-I015`).

---

## 2. Explicit Module Ownership & Data Authority

To maintain strict modular isolation, zero cross-module direct database access is permitted. Each business module acts as the sole source of truth for its assigned domain entities:

| Module Identifier | Module Name | Authoritative Domain Entities | State & Data Ownership Boundaries |
| :--- | :--- | :--- | :--- |
| **`MOD-CFG`** | Configuration & Checklist | `ChecklistTemplate`, `Section`, `Question`, `RuleVersion`, `ScoringRef` | Owns checklist authoring, versioning, question definitions, conditional branch references, and publication states. |
| **`MOD-WFA`** | Workflow & Action | `InspectionSchedule`, `Assignment`, `InspectionExecution`, `Response`, `Finding`, `ActionItem`, `Reinspection` | Owns operational inspection lifecycle, scheduling triggers, inspector assignment, finding logging, corrective action tracking, and state transitions. |
| **`MOD-EVD`** | Files & Evidence | `EvidenceRecord`, `Attachment`, `MediaQueue`, `ChecksumManifest` | Owns media capture metadata, binary storage references, cryptographic checksums, and chain of custody tracking. |
| **`MOD-REP`** | Reporting & Localization | `InspectionReport`, `FindingSummary`, `MetricView`, `LocalizationBundle` | Owns synthesized report models, metric calculation views, dashboard read projections, and language resources. |
| **`MOD-REC`** | Records & Audit | `LifecycleAuditRecord`, `TransitionEvent`, `HistoricalPreservation` | Owns append-only audit journals, cryptographic transition signatures, and historical lineage reconstruction. |
| **`MOD-IAM`** | Identity & Authorization | `UserSubject`, `RoleAssignment`, `DelegationGrant`, `SessionRecord` | Owns authentication truth, role-to-permission mapping, temporal delegation ceilings, and session revocation. |
| **`MOD-ORG`** | Organization & Tenancy | `Tenant`, `Company`, `BusinessUnit`, `Project`, `Site`, `Area`, `Party` | Owns 6-level organization hierarchy, sponsored party relationships, and tenant boundary isolation. |

### 2.1 Default-Deny Authority Invariant (`H040-004`)
In accordance with `H040-004`, all operations evaluate under strict **default-deny**:
- No state transition, creation, modification, or query is permitted without an explicit, active role grant.
- Access is restricted to the four approved foundation roles:
  1. **Checklist Author:** Can draft, edit, and propose checklist templates in `MOD-CFG`. Cannot execute inspections or close findings.
  2. **Inspector:** Can download assigned checklists, capture responses, attach evidence, and log findings in `MOD-WFA`. Cannot alter checklist templates or review own findings.
  3. **CAPA Owner:** Can submit action evidence, propose remediation dates, and record immediate controls. Cannot verify or close findings.
  4. **Independent Reviewer:** Can verify inspection submissions, review corrective actions, and authorize reinspections.
- **AI Boundary:** AI has zero autonomous safety decision authority. AI outputs are strictly advisory, supporting artifacts requiring human verification.

## 3. Deterministic Workflow State Machines

### 3.1 Checklist Template Lifecycle (`MOD-CFG`)
```
[DRAFT] ──(Submit)──> [UNDER_REVIEW] ──(Approve)──> [APPROVED] ──(Publish)──> [PUBLISHED] ──(Retire/Supersede)──> [RETIRED / SUPERSEDED]
```
- **Invariants:** Published checklists are permanently sealed and immutable. Any modification generates a new template iteration linking to its predecessor.

### 3.2 Inspection Execution Lifecycle (`MOD-WFA`)
```
[SCHEDULED] ──(Assign)──> [ASSIGNED] ──(Start)──> [IN_PROGRESS] ──(Submit)──> [COMPLETED] ──(Review)──> [FINALIZED / REJECTED]
```
- **Protected Transitions:** `FINALIZED` and `REJECTED` are protected transitions requiring an authorized `Independent Reviewer`.

### 3.3 Finding & Corrective Action (CAPA) Lifecycle (`MOD-WFA`)
```
[IDENTIFIED] ──(Assign Action)──> [ACTION_ASSIGNED] ──(Submit Evidence)──> [EVIDENCE_SUBMITTED] ──(Review)──> [VERIFIED_CLOSED / REOPENED]
```
- **Protected Transitions:** `VERIFIED_CLOSED` and `REOPENED` are protected transitions requiring explicit reviewer attribution. Reopened findings trigger automatic reinspection workflows.

---

## 4. State Synchronization, Server Authority & Conflict Resolution (`H040-005`)

Under approved foundation gate `H040-005`, state synchronization between client devices and central services is governed by strict server authority:

1. **Server Authority for Protected State:**
   - The central service maintains the authoritative state machine for all protected entities (completed inspections, finalized scores, closed findings).
   - Client applications submit state transition proposals accompanied by optimistic concurrency version tokens (`base_version`, `entity_digest`).
2. **Zero Last-Write-Wins (LWW):**
   - Server authority enforces zero last-write-wins; timestamp-based last-write-wins reconciliation is **categorically prohibited**.
   - If an incoming client submission's `base_version` does not match the server's current `entity_version`, the submission is rejected.
3. **Conflict Quarantine & Manual Reconciliation:**
   - Conflicting submissions are automatically partitioned into a `QUARANTINED` holding container (`ConflictRecord`).
   - An immutable conflict diagnostic event (`ConflictQuarantinedEvent`) is emitted, requiring explicit manual reconciliation by an authorized human role.
---

## 5. Public-Contract Map & Domain Events

Inter-module communication operates exclusively via versioned public contracts (`contracts/api/`) and asynchronous domain events (`MOD-EVT`):

### 5.1 Public Contract Registry
- `contracts/api/checklist/v1`: Defines `ChecklistTemplateView`, `QuestionDTO`, `SectionDTO`, `ApplicabilityRuleDTO`.
- `contracts/api/inspection/v1`: Defines `InspectionSessionDTO`, `ResponseItemDTO`, `InspectionSummaryView`.
- `contracts/api/finding/v1`: Defines `FindingDetailView`, `SeverityClassificationDTO`, `ImmediateControlDTO`.
- `contracts/api/action/v1`: Defines `CAPAActionDTO`, `RemediationEvidenceDTO`, `ExtensionRequestDTO`.
- `contracts/api/evidence/v1`: Defines `EvidenceMetadataView`, `MediaChecksumDTO`, `AttachmentRefDTO`.

### 5.2 Domain Event Catalog
Every critical state change emits a strongly-typed domain event to the append-only outbox:
1. `ChecklistPublishedEvent`: Emitted by `MOD-CFG` when a checklist enters `PUBLISHED` state.
2. `InspectionAssignedEvent`: Emitted by `MOD-WFA` upon inspector assignment.
3. `InspectionCompletedEvent`: Emitted by `MOD-WFA` upon field submission.
4. `FindingIdentifiedEvent`: Emitted by `MOD-WFA` when a critical or non-compliant finding is logged.
5. `ConflictQuarantinedEvent`: Emitted by synchronization handlers when concurrency conflict occurs.

---

## 6. Append-Only Audit Obligations (`MOD-REC`)

Every protected state transition, security check failure, or manual override must generate an immutable audit record containing:
- `record_id`: Strongly-typed unique identifier (`rec_[0-9a-f]{16}`).
- `tenant_id`: Authoritative tenant identifier (`ten_*`).
- `entity_type` & `entity_id`: Target entity identifier (`chk_*`, `ins_*`, `fnd_*`, `act_*`).
- `version`: Monotonically increasing entity version number.
- `from_state` & `to_state`: Exact transition endpoints.
- `actor_subject` & `actor_role`: Responsible synthetic user and role.
- `timestamp`: UTC timestamp with millisecond precision.
- `reason`: Mandatory justification for overrides, rejections, or closures.
- `payload_digest`: Cryptographic SHA-256 hash of the modified entity payload.
- `audit_digest`: Composite SHA-256 seal ensuring audit log tamper-evidence.

---

## 7. Retained Human Gates, Prohibitions & Non-Claims

### 7.1 Retained Human Gate Register
The following governance gates remain strictly on **`HOLD`**:
- **`H040-007` (Technical Release Authorization):** `HOLD` pending full integration evidence.
- **`H040-008` (Real Participant / Private-Alpha / UAT Authorization):** `HOLD` pending human participant authorization.
- **`H040-009` (Binding Support & Manual-Fallback Ownership):** `HOLD` pending operational agreement.
- **`H040-010` (External Environment & Service Activation):** `HOLD` pending cloud provisioning authorization.
- **`H040-011` (Final v0.4 Outcome & v0.5 Entry Decision):** `HOLD` pending milestone completion evaluation.

### 7.2 Strict Operational Prohibitions
- Zero public routes, internet endpoints, DNS records, or CDN distributions are authorized.
- Zero live cloud deployments, package signings, or database persistence mutations are authorized.
- Zero customer data, real workforce PII, or physical site measurements are permitted.
- Zero automated Issue-closing keywords may be used in pull requests for Issue #113.
