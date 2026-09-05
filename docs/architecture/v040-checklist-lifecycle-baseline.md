---
document_id: ARC-V040-CHKLIFE-001
title: v0.4.0 OSHE Inspect Checklist Review, Approval, Publication, Copy, Retirement, and Version Lifecycle Baseline
document_type: architecture_specification
document_version: 1.0.0
lifecycle_status: DRAFT
status: APPROVED_FOR_LOCAL_DEVELOPMENT
date: "2026-09-05"
author_role: Engineering Lead
author_pane: w9:p23
governing_issue: "GitHub Issue #118"
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
credit_boundary: CHECKLIST_LIFECYCLE_SPECIFICATION_ONLY_NO_EXTERNAL_PUBLICATION_OR_LIVE_OPERATIONAL_ENACTMENT
---

# v0.4.0 OSHE Inspect Checklist Review, Approval, Publication, Copy, Retirement, and Version Lifecycle Baseline

## 1. Executive Summary & Governance Reference

### 1.1 Authority Baseline & Purpose
This architectural specification establishes the authoritative **Checklist Review, Approval, Publication, Copy Provenance, Retirement, Supersession, and Version Lifecycle Baseline** for **Milestone v0.4.0 - OSHE Inspect Private Alpha** within `OSHEThai/oshe-platform`. It fulfills the deliverable requirements of **GitHub Issue #118 (`[V040-I007] Checklist Review, Approval, Publication, and Version Lifecycle`)** under Roadmap Topic `V040-T01` and standing owner foundation decision `HDEC-V040-FOUNDATION-054`.

The primary objective is to define a bounded, deterministic, dependency-free lifecycle state machine governing checklist template creation, review challenges, independent verification, cryptographic publication sealing, copy derivation, and execution pinning across template iterations.

### 1.2 Retained Unselected Policies & Non-Claims
In strict compliance with `ASN-V040-I007-CHECKLIST-LIFECYCLE-001` and `HDEC-V040-FOUNDATION-054`:
1. **Binding Scoring Policy (`HUMAN_OWNED_UNSELECTED`):** All scoring references and category weights contained within checklist iterations are provisional models. Final binding scoring policies remain human-owned pending owner decision under Issue #136 (`V040-I025`).
2. **Finding Closure Policy (`HUMAN_OWNED_UNSELECTED`):** Finding resolution criteria, verification procedures, and residual risk acceptance remain human-owned under Issue #134 (`V040-I023`).
3. **Offline Authority Model (`HUMAN_OWNED_UNSELECTED`):** Client-side lease durations, offline download expiration limits, and conflict priority policies require explicit human owner decision under Issue #126 (`V040-I015`).
4. **No External Publication or Public Route Activation:** All publication transitions defined herein operate strictly in memory and within local synthetic stores. Zero public DNS domains, CDN edge distributions, external routes, or public endpoints are authorized or activated.
5. **Retained Human Gate Holds:** Foundation gates `H040-007` (Technical Release Authorization), `H040-008` (Real Participant / Private-Alpha UAT Onboarding), `H040-009` (Binding Support Ownership), `H040-010` (External Environment/Route Activation), and `H040-011` (Final Milestone Outcome) remain strictly on **`HOLD`**.

---

## 2. Explicit Lifecycle States & State Machine (`MOD-CFG`)

Under `ARC-V040-DOMAIN-001`, the **Configuration and Checklist Module (`MOD-CFG`)** maintains sole authoritative ownership of checklist template lifecycles.

### 2.1 Enumerated Lifecycle States
Every checklist template iteration exists in exactly one of the following seven discrete states:

```
┌─────────┐      Submit Review      ┌──────────────┐      Approve      ┌────────────┐
│  DRAFT  │ ──────────────────────> │ UNDER_REVIEW │ ────────────────> │  APPROVED  │
└─────────┘                         └──────┬───────┘                   └────────────┘
     ▲                                     │ Reject                          │ Publish
     │ Revise                              ▼                                 ▼
     └───────────────────────────── ┌──────────────┐                   ┌──────────────────────┐
                                    │   REJECTED   │                   │ PUBLISHED_IMMUTABLE  │ (SEALED)
                                    └──────────────┘                   └──────────┬───────────┘
                                                                                  │
                                                       ┌──────────────────────────┴──────────────────────────┐
                                                       │ Retire                                              │ Supersede
                                                       ▼                                                     ▼
                                                ┌─────────────┐                                       ┌──────────────┐
                                                │   RETIRED   │                                       │  SUPERSEDED  │
                                                └─────────────┘                                       └──────────────┘
```

| State Identifier | Mutability Posture | Inspection Instantiation | State Semantics & Governance Boundary |
| :--- | :--- | :--- | :--- |
| **`DRAFT`** | **Mutable** | Disallowed | Active authoring state. The `Checklist Author` may create, update, and delete sections, questions, applicability rules, and translations. |
| **`UNDER_REVIEW`** | **Read-Only** | Disallowed | Frozen review state. Locked for author edits while under challenge review by an `Independent Reviewer`. |
| **`REJECTED`** | **Read-Only** | Disallowed | Rejection state. Contains structured review findings indicating why approval was denied. May transition back to `DRAFT` for author revision. |
| **`APPROVED`** | **Read-Only** | Disallowed | Signed-off state. Formal reviewer approval recorded with timestamp and signature hash. Awaits publication scheduling. |
| **`PUBLISHED_IMMUTABLE`** | **Permanently Immutable** | **Allowed** | Operational production state. Cryptographically sealed with SHA-256 `content_digest`. Field inspections may instantiate this version. In-place edits strictly blocked. |
| **`RETIRED`** | **Permanently Immutable** | Disallowed | Decommissioned state. In-flight inspections complete against this version; new inspection creation is prohibited. |
| **`SUPERSEDED`** | **Permanently Immutable** | Disallowed | Replaced state. Contains explicit link to `successor_version`. In-flight inspections complete; new inspections instantiate the successor. |

---

## 3. Authorized Transition Matrix & Role Boundaries (`H040-004`)

Under default-deny authority (`H040-004`), state transitions require explicit, authenticated role entitlements:

### 3.1 Transition Authorization Matrix

| Source State | Target State | Transition Event | Authorized Role | Mandatory Validation Invariants |
| :--- | :--- | :--- | :--- | :--- |
| **`DRAFT`** | **`UNDER_REVIEW`** | `SubmitForReview` | `Checklist Author` | Template must have $\ge 1$ section, $\ge 1$ question, complete bilingual translations (`en-US`, `th-TH`), and valid SemVer version. |
| **`UNDER_REVIEW`** | **`APPROVED`** | `ApproveChecklist` | `Independent Reviewer` | **Self-Approval Prohibited (`SOD-03`):** `reviewer_subject != author_subject`. Schema passes linting and rule DAG validation. |
| **`UNDER_REVIEW`** | **`REJECTED`** | `RejectChecklist` | `Independent Reviewer` | Reviewer must attach at least one structured review finding with `severity: BLOCKING` or `MAJOR`. |
| **`REJECTED`** | **`DRAFT`** | `ResumeDraft` | `Checklist Author` | Author acknowledges review findings; locks are released for iterative revision. |
| **`APPROVED`** | **`PUBLISHED_IMMUTABLE`** | `PublishChecklist` | `Checklist Author` / Lead | Computes and seals SHA-256 `content_digest`. Sets `published_at` and `effective_from` timestamps. |
| **`PUBLISHED_IMMUTABLE`** | **`RETIRED`** | `RetireChecklist` | `Checklist Author` / Lead | Requires mandatory non-blank `retirement_reason`. In-flight inspections remain unaffected. |
| **`PUBLISHED_IMMUTABLE`** | **`SUPERSEDED`** | `SupersedeChecklist` | System / Lead | Automatically executed upon publication of a new version sharing the same `template_id` and linking `predecessor_version`. |

### 3.2 Default-Deny & Unauthorized State Transitions
1. Any transition not explicitly enumerated in the table above evaluates to **`DENIED`** (`ErrUnauthorizedTransition`).
2. An `Inspector` or `CAPA Owner` attempting any checklist state transition is immediately denied (`ErrInsufficientRolePrivilege`).
3. Direct jumps (e.g. `DRAFT` $\to$ `PUBLISHED_IMMUTABLE` bypassing review, or `REJECTED` $\to$ `APPROVED`) fail closed.

---

## 4. Independent-Review Attribution & Findings Tracking

In strict compliance with segregation-of-duties rule `SOD-03` and foundation gate `H040-004`:

### 4.1 Mandatory Attribution Fields
Every review decision (`APPROVED` or `REJECTED`) must record:
- `reviewer_subject`: Authoritative synthetic user identifier (`usr_syn_reviewer_*`).
- `reviewer_role`: Strictly `Independent Reviewer`.
- `review_timestamp`: UTC timestamp with millisecond precision.
- `review_notes`: Summary commentary detailing the review outcome.
- `target_content_digest`: SHA-256 digest of the reviewed draft payload.

### 4.2 Structured Review Findings
When a checklist template is rejected, the reviewer attaches structured findings:
```yaml
review_findings:
  - finding_id: "rfnd_syn_01"
    severity: "BLOCKING"      # BLOCKING | MAJOR | MINOR
    section_id: "sec_syn_01"
    question_id: "qst_syn_02"
    rule_id: "rule_syn_01"
    message:
      en-US: "Question SCF-02 uses SINGLE_CHOICE but omits required option for bamboo scaffolding."
      th-TH: "คำถาม SCF-02 เป็นแบบเลือกหนึ่งข้อแต่ขาดตัวเลือกนั่งร้านไม้ไผ่ที่กำหนด"
    recommended_action: "Add missing option or adjust question type."
```
- **Rejection Rule:** A transition to `REJECTED` fails closed if `review_findings` is empty or contains only `MINOR` advisory nits.

---

## 5. Immutable Published Versions & Cryptographic Sealing

### 5.1 The Immutable-Version Boundary
Upon entering `PUBLISHED_IMMUTABLE`:
1. **Permanent Content Freeze:** The template schema, sections, questions, prompts, guidance, options, numeric boundaries, applicability rules, and scoring references become permanently read-only.
2. **Cryptographic Sealing (`content_digest`):** A composite SHA-256 hash is generated over the canonical JSON serialization of the template:
   $$\text{content\_digest} = \text{SHA-256}(\text{CanonicalJSON}(\text{TemplatePayload}))$$
3. **Write Protection:** Any SQL `UPDATE` or `DELETE` targeting a published template record is blocked at the database layer via triggers and at the application layer via model invariants (`ErrImmutableVersionMutation`).

---

## 6. Copy Provenance & Template Iteration (`predecessor_version`)

Because published versions cannot be edited in place, evolving a checklist requires deriving a new iteration:

### 6.1 Derivation & Provenance Model
```
┌─────────────────────────────────────────┐          Copy / Iterate          ┌─────────────────────────────────────────┐
│ Checklist Template v1.0.0               │ ───────────────────────────────> │ Checklist Template v1.1.0               │
│ - status: PUBLISHED_IMMUTABLE           │                                  │ - status: DRAFT                         │
│ - content_digest: e3b0c442...           │                                  │ - predecessor_version: "1.0.0"          │
└─────────────────────────────────────────┘                                  │ - derived_from_digest: e3b0c442...      │
                                                                             │ - copied_by: "usr_syn_author_01"        │
                                                                             │ - copied_at: "2026-09-05T15:00:00Z"     │
                                                                             └─────────────────────────────────────────┘
```

### 6.2 Provenance Invariants
1. **Stable `template_id`:** Successor iterations retain the exact same `template_id` (`chk_syn_*`), establishing historical continuity across versions.
2. **SemVer Progression:** The new version string must strictly increment the predecessor version according to Semantic Versioning (`1.0.0` $\to$ `1.1.0` for backward-compatible additions; `2.0.0` for breaking question restructuring).
3. **Explicit Predecessor Linkage:** The new draft iteration records `predecessor_version: "1.0.0"` and `derived_from_digest: "<sha256>"`.
4. **Copy Lineage Preservation:** If a template is copied to form an entirely new checklist category, a new `template_id` is assigned, but `copy_source_template_id` and `copy_source_version` are preserved for provenance.

---

## 7. Retirement, Supersession & Execution Pinning

### 7.1 Lifecycle Phase-Out Mechanics
1. **Retirement (`RETIRED`):** Applied when a checklist template is phased out without direct replacement. Records `retired_at`, `retired_by`, and mandatory `retirement_reason`.
2. **Supersession (`SUPERSEDED`):** Applied when an existing published version is replaced by a newly published successor iteration. Records `superseded_at` and `successor_version: "1.1.0"`.

### 7.2 Execution Pinning for Active Inspections
1. **In-Flight Schema Pinning:** When an inspection session (`ins_syn_*`) is scheduled or assigned in `MOD-WFA`, it is permanently bound to the exact `template_id`, `version`, and `content_digest` active at creation time.
2. **Zero Runtime Hot-Swapping:** Subsequent publication of a successor version (`v1.1.0`) or retirement of `v1.0.0` has **zero effect** on in-flight inspections. Inspections complete, validate, and finalize against their pinned version, guaranteeing audit repeatability.

---

## 8. Append-Only Lifecycle Audit Events (`MOD-REC`)

Every lifecycle transition emits an immutable, strongly-typed audit record to the `MOD-REC` append-only journal:

| Transition Event | Emitted Event Name | Recorded Audit Payload |
| :--- | :--- | :--- |
| Submit Review | `checklist.submitted` | `template_id`, `version`, `author_subject`, `timestamp`, `draft_digest` |
| Approve | `checklist.approved` | `template_id`, `version`, `reviewer_subject`, `review_notes`, `timestamp` |
| Reject | `checklist.rejected` | `template_id`, `version`, `reviewer_subject`, `findings_count`, `findings_summary` |
| Publish | `checklist.published` | `template_id`, `version`, `content_digest`, `published_at`, `effective_from` |
| Copy / Iterate | `checklist.copied` | `template_id`, `new_version`, `predecessor_version`, `derived_from_digest`, `actor` |
| Retire | `checklist.retired` | `template_id`, `version`, `retired_by`, `retirement_reason`, `timestamp` |
| Supersede | `checklist.superseded` | `template_id`, `version`, `successor_version`, `timestamp` |

---

## 9. Synthetic Multi-Version Lifecycle Fixtures

The following synthetic YAML fixture demonstrates the complete multi-version lifecycle progression from v1.0.0 through v1.1.0:

```yaml
schema_version: "1.0.0"
fixture_id: "fix_syn_checklist_lifecycle_v1_v2"
template_id: "chk_syn_pilot_plant_safety_v1"

iterations:
  # Iteration 1: Published and Subsequently Superseded
  - version: "1.0.0"
    lifecycle_status: "SUPERSEDED"
    title:
      en-US: "Pilot Industrial Plant Safety Inspection Baseline"
      th-TH: "การตรวจสอบความปลอดภัยของโรงงานอุตสาหกรรมนำร่อง"
    author_subject: "usr_syn_author_01"
    owner_role: "Checklist Author"
    published_at: "2026-09-05T14:30:00Z"
    effective_from: "2026-09-05T14:30:00Z"
    effective_to: "2026-09-05T17:00:00Z"
    superseded_at: "2026-09-05T17:00:00Z"
    successor_version: "1.1.0"
    content_digest: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
    review_record:
      reviewer_subject: "usr_syn_reviewer_02"
      reviewer_role: "Independent Reviewer"
      review_timestamp: "2026-09-05T14:00:00Z"
      review_decision: "APPROVED"
      review_notes: "Initial pilot checklist verified and approved for private alpha."

  # Iteration 2: Derived Iteration Published with Provenance
  - version: "1.1.0"
    lifecycle_status: "PUBLISHED_IMMUTABLE"
    title:
      en-US: "Pilot Industrial Plant Safety Inspection Baseline (Minor Revision)"
      th-TH: "การตรวจสอบความปลอดภัยของโรงงานอุตสาหกรรมนำร่อง (ฉบับปรับปรุงย่อย)"
    author_subject: "usr_syn_author_01"
    owner_role: "Checklist Author"
    predecessor_version: "1.0.0"
    derived_from_digest: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
    copied_by: "usr_syn_author_01"
    copied_at: "2026-09-05T15:30:00Z"
    published_at: "2026-09-05T17:00:00Z"
    effective_from: "2026-09-05T17:00:00Z"
    effective_to: null
    content_digest: "8f4e2b1c3a7d9e5f0b2a4c6d8e1f3a5b7c9d1e3f5a7b9c1d3e5f7a9b1c3d5e7f"
    review_record:
      reviewer_subject: "usr_syn_reviewer_02"
      reviewer_role: "Independent Reviewer"
      review_timestamp: "2026-09-05T16:45:00Z"
      review_decision: "APPROVED"
      review_notes: "Added clarification guidance for wheel brakes. All criteria verified."
      findings_resolved:
        - finding_id: "rfnd_syn_rev1_01"
          resolution: "Clarified wheel brake inspection guidance for mobile scaffolding."
```

---

## 10. Governance Boundaries & Non-Claims

In strict compliance with `HDEC-V040-FOUNDATION-054`:

1. **Synthetic Data Exclusivity (`H040-003`):** All lifecycle states, review records, user IDs (`usr_syn_*`), and template identifiers (`chk_syn_*`) are synthetic local fixtures. Zero real customer data or corporate records are used.
2. **Default-Deny Authority Invariant (`H040-004`):** Checklist state transitions strictly require authorized roles. Self-approval is barred (`SOD-03`).
3. **No External Route or CDN Distribution (`H040-007` HOLD):** Public snapshot resolution or external route activation remains on HOLD.
4. **No Participant Onboarding (`H040-008` HOLD):** Zero real inspectors, authors, or pilot participants are authorized or onboarded.
5. **Specification-Only Credit:** Delivery of this baseline confers documentation and workflow model credit only; zero operational release, deployment, or residual-risk acceptance is claimed.
