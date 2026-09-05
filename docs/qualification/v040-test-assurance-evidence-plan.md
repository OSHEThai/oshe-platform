---
document_id: DOC-PLAN-V040-001
document_type: quality_assurance_plan
lifecycle_status: DRAFT
governing_issue: GitHub Issue #115
authority_source: HDEC-V040-FOUNDATION-054
release_target: v0.4.0 - OSHE Inspect Private Alpha
governing_gates_approved:
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
credit_boundary: PLANNING_ONLY_NO_EXECUTION_OR_RELEASE_CREDIT
---

# v0.4.0 Test, Assurance, UAT, Support, Recovery, and Evidence Plan

| Metadata Field | Value |
| :--- | :--- |
| **Document ID** | `DOC-PLAN-V040-001` |
| **Document Type** | Comprehensive Test, Assurance, UAT, and Evidence Plan |
| **Lifecycle Status** | `DRAFT` |
| **Governing Issue** | GitHub Issue #115 (`V040-I004`) |
| **Authority Source** | `HDEC-V040-FOUNDATION-054` |
| **Release Target** | Milestone v0.4.0 (*OSHE Inspect Private Alpha*) |
| **Approved Foundation Gates** | `H040-001`, `H040-002`, `H040-003`, `H040-004`, `H040-005`, `H040-006` |
| **Retained Governance Holds** | `H040-007`, `H040-008`, `H040-009`, `H040-010`, `H040-011` |
| **Credit Boundary** | Planning and framework materialization only (`PLANNING_ONLY_NO_EXECUTION_OR_RELEASE_CREDIT`) |

---

## 1. Executive Summary & Purpose

This plan establishes the authoritative, deterministic test, assurance, user acceptance testing (UAT), operational support, disaster recovery, defect triage, and evidence mapping framework for **Milestone v0.4.0 (*OSHE Inspect Private Alpha*)**.

The core objective is to define unambiguous criteria, verification harnesses, and evidence contracts that govern the narrow, single-tenant private-alpha vertical slice authorized under Sole Human Owner decision `HDEC-V040-FOUNDATION-054`.

### Foundational Invariant: Strict Separation of Synthetic and Real-User Evidence
A cornerstone requirement of this assurance plan is the categorical separation between **synthetic technical evidence** and **empirical real-user evidence**:
- **Synthetic Technical Evidence:** Validates software correctness, authorization gates, concurrency handling, and fail-closed invariants using deterministic local test fixtures.
- **Empirical Real-User Evidence:** Evaluates genuine field usability, cognitive burden, and ergonomics through structured observation of human participants.
- **Non-Substitution Invariant:** Under no circumstances may simulated agent runs, automated scripts, or mock data substitute for, replace, or claim the status of empirical real-user evidence.

---

## 2. Governance Gates & Operational Boundaries

This plan aligns strictly with Sole Human Owner decision `HDEC-V040-FOUNDATION-054`, distinguishing between approved foundation parameters and retained governance holds.

### Approved Foundation Parameters (`H040-001` - `H040-006`)
1. **`H040-001` (Narrow Single-Tenant Vertical Slice):** Covers four core roles: Checklist Author, Inspector, CAPA Owner, and Independent Reviewer. Excludes public access, external integrations, production customer data, and autonomous AI decision-making.
2. **`H040-002` (Client Environment Baseline):** Responsive web support for current desktop Chrome/Edge and mobile Android Chrome; dual-language UI in English (`en-US`) and Thai (`th-TH`); operational time zone anchored to `Asia/Bangkok`.
3. **`H040-003` (Synthetic Data Boundary):** Purely synthetic or sanitized mock data during engineering; zero real PII or live customer evidence prior to explicit separate approval.
4. **`H040-004` (Default-Deny Authority):** Default-deny access policy; protected state transitions require named responsible roles; overrides mandate justification and immutable audit logging; AI has zero autonomous safety decision authority.
5. **`H040-005` (Server Authority & Conflict Resolution):** Server authority for all protected state; rejection of last-write-wins (LWW); concurrent update conflicts quarantine immediately for manual reconciliation.
6. **`H040-006` (Synthetic Pilot Checklist):** Single synthetic non-regulatory pilot checklist with versioned scoring, explicit `Unknown` and `Not Applicable` response handling, and zero legal or real-world safety-threshold claims.

### Retained Governance Holds (`H040-007` - `H040-011`)
The following gates remain on strict **`HOLD`** and cannot be enacted, bypassed, or scheduled by this plan:
- **`H040-007` (Technical Release Authorization):** HOLD pending verified completion of all technical evidence gates.
- **`H040-008` (Real Participant / Private-Alpha / UAT Authorization):** HOLD pending separate owner selection, screening, and onboarding authorization.
- **`H040-009` (Binding Support & Manual-Fallback Ownership):** HOLD pending formal organizational staffing and operational handover.
- **`H040-010` (External Environment & Route Activation):** HOLD pending infrastructure security review; zero public internet routes, cloud storage buckets, or external notifications may be provisioned.
- **`H040-011` (Final v0.4 Outcome & v0.5 Entry Decision):** HOLD reserved exclusively to the Sole Human Owner.

### Explicit Prohibitions & Anti-Scope Invariants
In accordance with `HDEC-V040-FOUNDATION-054`, the following prohibitions are strictly enforced during this phase:
- **Zero External Public Routes:** No public domains, DNS records, or external HTTP routes may be exposed.
- **Zero CDN Edge Deployment:** No static assets or endpoints may be published to edge networks or CDNs.
- **Zero Production Database Deployment:** Testing is confined exclusively to local, ephemeral, or isolated test instances.
- **Zero Real Customer or Personal Data:** No live customer records, real identifiable credentials, or field pilot personal data may be ingested.
- **Zero Provider, Credential, or Account Mutations:** No third-party API accounts, production cloud services, or authentication providers may be provisioned or mutated.

---

## 3. Technical Testing Framework

The technical testing architecture provides multi-layered deterministic verification of the v0.4 vertical slice.

### 3.1 Role & Workflow Journey Matrix
The test suite validates four distinct operational roles across their core lifecycle paths:
- **Checklist Author (`RoleChecklistAuthor`):** Template drafting, version incrementing, scoring weight assignment, and template archival.
- **Field Inspector (`RoleInspector`):** Inspection instantiation from active template, question response evaluation (`Pass`, `Fail`, `Unknown`, `Not Applicable`), finding logging, and evidence attachment.
- **CAPA Owner (`RoleCAPAOwner`):** Finding assignment, corrective action drafting, remedial evidence attachment, and completion submission.
- **Independent Reviewer (`RoleIndependentReviewer`):** Inspection and action verification, formal closeout approval, and review audit sealing.

### 3.2 Client Environment & Platform Testing
- **Browsers:** Automated headless and responsive layout testing on Chrome, Edge, and Android Chrome viewports (360x640 to 1920x1080).
- **Localization:** 100% string coverage across Thai (`th-TH`) and English (`en-US`) translation dictionaries without unrendered keys.
- **Temporal Handling:** Deterministic time injection asserting timestamp rendering strictly in `Asia/Bangkok` (ICT, UTC+7).

### 3.3 Concurrency & State Authority (Gate `H040-005`)
- **Server Authority:** State transitions are mediated exclusively by backend domain logic; client state is treated as unverified proposal.
- **No Last-Write-Wins (LWW):** Conflicting concurrent submissions against the same inspection or finding do not silently overwrite.
- **Quarantine on Conflict:** Conflicting state transitions transition into `QUARANTINED_CONFLICT`, emitting audit alerts for manual administrative triage.

### 3.4 Deterministic Negative Controls
The suite enforces strict negative boundary checks:
- **Cross-Role Escalation:** Inspectors cannot approve their own inspections; CAPA Owners cannot close findings without Independent Reviewer verification.
- **Unauthorized Bypass:** Unauthenticated callers fail closed (`401 Unauthorized`); cross-tenant probing fails closed (`404 Not Found`).
- **AI Boundary Enforcement:** AI recommendation engines cannot transition findings, approve checklists, or alter risk classifications.

---

## 4. Independent Review & Challenge Framework

Technical evidence is supplemented by formal peer and challenge reviews:
- **Reviewer Independence:** Reviewers must be functionally distinct from authors and implementers.
- **Adversarial Challenge Review:** Specialized challenge reviewers conduct structured IDOR, boundary fuzzing, and permission escalation probing.
- **Audit Verification:** Review decisions must be cryptographically sealed with decision digests and logged to append-only ledgers.

---

## 5. Real-User UAT & Private-Alpha Protocol (Gated under `H040-008` HOLD)

Empirical validation with human participants is strictly deferred until the Sole Human Owner authorizes Gate `H040-008`.

### 5.1 Standby Protocol Architecture
When authorized, the private-alpha protocol will evaluate:
- **Task Completion Rate:** Proportion of checklist questions and finding remediations successfully completed without operator assistance.
- **Cognitive Clarity & Language Usability:** Usability of Thai and English terminology in high-stress inspection scenarios.
- **Observation Logging:** Structured observer notes capturing confusion, hesitation, or navigation friction.

### 5.2 Participant Gating Pre-Conditions
Before human onboarding can occur, the following prerequisites must be met:
1. Formal lifting of Gate `H040-008` by the Sole Human Owner.
2. Participant consent agreements adhering to PDPA/GDPR minimization.
3. Verification that testing occurs strictly in an isolated staging sandbox without real workplace legal liability.

---

## 6. Support, Failure Modes, and Operational Recovery (Gated under `H040-009` HOLD)

Operational resilience ensures continuity during synthetic testing and prepares for eventual deployment.

### 6.1 Support Observation & Metrics
- **Triage Dashboard:** Real-time visibility of active sessions, quarantined conflicts, and failed validations.
- **Support Boundaries:** Support personnel hold read-only diagnostic visibility; data mutation is barred.

### 6.2 Manual Fallback Pathways
- **Quarantine Resolution:** Structured workflow allowing authorized managers to manually reconcile quarantined conflicts with full audit logging.
- **Paper / Offline Fallback:** Exportable standardized PDF/print representations of pilot checklists for field continuity during system downtime.

### 6.3 Disaster Recovery & Lineage Rebuild
- **State Serialization:** Periodic deterministic snapshots of the domain graph.
- **Reconstruction Verification:** Validates that entire system state and audit logs can be rebuilt from append-only transaction ledgers with zero data loss.

---

## 7. Defect Severity Policy & Triage Rules

Defects identified during qualification testing are classified according to strict operational criteria:

| Severity Level | Definition | Release Gating Rule |
| :--- | :--- | :--- |
| **P1 - Critical / Blocker** | Security breach, data loss, cross-tenant leak, incorrect safety scoring, server crash, or role bypass. | **Zero P1 defects permitted.** Blocks any release or milestone completion. |
| **P2 - High** | Major functional defect without workaround, localization missing critical safety terms, or failed state conflict quarantine. | Must be resolved or explicitly waived by Sole Human Owner with documented mitigation. |
| **P3 - Medium** | Minor functional defect with clear workaround, non-critical UI layout flaw, or cosmetic translation error. | Triage for resolution; does not inherently block private alpha. |
| **P4 - Low** | Minor cosmetic styling inconsistency, documentation typo, or minor enhancement suggestion. | Backlog for future milestone refinement. |

---

## 8. Comprehensive Evidence Mapping & Assurance Scorecard

The evidence map links requirements, risks, journeys, and validation mechanisms to their authoritative evidence artifacts:

| ID | Focus Domain | Primary Gate | Technical Verification Harness | Review / Human Evidence Requirement | Release Blocker? |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **EVD-01** | Role-Based Access Control | `H040-001`, `H040-004` | `test_rbac_matrix_and_escalation_denial` | Independent Security Review | **YES (P1)** |
| **EVD-02** | Client Platform & Viewports | `H040-002` | `test_responsive_viewport_and_localization` | Usability Review | **YES (P2)** |
| **EVD-03** | Synthetic Data Containment | `H040-003` | `test_pii_detection_and_sanitization` | Privacy Lead Review | **YES (P1)** |
| **EVD-04** | Server Authority & No-LWW | `H040-005` | `test_conflict_quarantine_no_lww` | Systems Architecture Review | **YES (P1)** |
| **EVD-05** | Scoring & Non-Regulatory Pilot | `H040-006` | `test_checklist_scoring_and_unknown_handling` | Domain Safety Review | **YES (P2)** |
| **EVD-06** | Real-User Field Usability | `H040-008` (HOLD) | Simulated journey preflight only | Empirical UAT observation (Gated under `H040-008`) | **HOLD** |
| **EVD-07** | Support & Conflict Recovery | `H040-009` (HOLD) | `test_quarantine_manual_resolution_rebuild` | Support Operations Review (Gated under `H040-009`) | **HOLD** |
| **EVD-08** | External Isolation | `H040-010` (HOLD) | `test_zero_external_route_listeners` | Infrastructure Security Review | **YES (P1)** |
| **EVD-09** | Release Decision Packaging | `H040-007`, `H040-011` | Automated evidence bundle completeness check | Sole Human Owner Gating Decision | **HOLD** |

---

## 9. Gating Decision Checklist & Non-Claims Conclusion

Prior to seeking authorization for subsequent milestone gates, the project must verify:
- [x] Test and assurance planning baseline completed and reviewed (`V040-I004`).
- [ ] Core vertical slice implementation completed and verified by technical tests.
- [ ] Zero P1 defects open in the tracking register.
- [ ] Retained holds `H040-007` through `H040-011` respected and maintained without premature activation.

### Final Non-Claims Notice
This document is a **planning artifact only**. It does not authorize software deployment, production routing, real-user onboarding, commercial pilot release, or residual-risk acceptance. All authority remains reserved to the Sole Human Owner.
