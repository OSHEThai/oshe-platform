---
document_id: QLF-V040-CHKL-001
title: v0.4.0 OSHE Inspect Checklist Authoring, Lifecycle, Concurrency, Provenance, Localization, and Historical Preservation Qualification Baseline
document_type: qualification_specification
document_version: 1.0.0
lifecycle_status: DRAFT
status: APPROVED_FOR_LOCAL_DEVELOPMENT
date: "2026-09-05"
author_role: Test and Quality Lead
author_pane: w9:p14
governing_issue: "GitHub Issue #119"
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
credit_boundary: TECHNICAL_QUALIFICATION_ONLY_NO_USER_EVIDENCE_OR_RELEASE_CREDIT
---

# v0.4.0 OSHE Inspect Checklist Authoring, Lifecycle, Concurrency, Provenance, Localization, and Historical Preservation Qualification Baseline

## 1. Executive Summary & Governance Authority

### 1.1 Authority Baseline & Purpose
This qualification specification establishes the authoritative, deterministic **Checklist Technical Qualification Baseline** for **Milestone v0.4.0 - OSHE Inspect Private Alpha** within `OSHEThai/oshe-platform`. It fulfills the qualification scope and acceptance criteria of **GitHub Issue #119 (`[V040-I008] Qualify Checklist Authoring, Versioning, Conditions, Translation References, and Historical Preservation`)** under Roadmap Topic `V040-T01` and standing owner foundation decision `HDEC-V040-FOUNDATION-054`.

The primary purpose is to define an integrated, dependency-free, deterministic verification harness and qualification suite covering checklist template authoring, lifecycle state machine progression, Segregation of Duties (`SOD-03`), multi-client concurrency conflicts, copy provenance derivation, dual-language localization references, execution pinning, and report traceability for the standalone single-tenant private alpha vertical slice (`H040-001`).

### 1.2 Non-Substitution Invariant: Technical & Synthetic Scope Only
A core invariant governing this baseline is the categorical separation between **synthetic technical evidence** and **empirical real-user evidence**:
- **Synthetic Technical Qualification:** Verifies software logic, schema constraints, role-based authorization, concurrency quarantine, cryptographic digests, and deterministic fail-closed state machines using local synthetic fixtures and automated test suites.
- **Empirical Real-User Evidence:** Evaluates real-world field ergonomics, inspector cognitive load, and human usability under authentic workplace conditions.
- **Non-Substitution Invariant:** Under no circumstances may simulated agent runs, automated test executions, or synthetic test payloads substitute for, replace, or claim the status of empirical real-user evidence. Gate `H040-008` (Real Participant / Private-Alpha UAT Onboarding) remains strictly on **`HOLD`** pending separate owner authorization.

### 1.3 Retained Governance Holds & Explicit Prohibitions
In strict accordance with `HDEC-V040-FOUNDATION-054`, the following governance holds remain in effect and cannot be enacted, bypassed, or scheduled by this specification:
- **`H040-007` (Technical Release Authorization):** HOLD pending completed qualification bundles and owner sign-off.
- **`H040-008` (Real Participant / Private-Alpha / UAT Authorization):** HOLD pending separate owner screening and onboarding decision.
- **`H040-009` (Binding Support & Manual-Fallback Ownership):** HOLD pending formal organizational staffing and handover.
- **`H040-010` (External Environment & Route Activation):** HOLD pending infrastructure security review.
- **`H040-011` (Final v0.4 Outcome & v0.5 Entry Decision):** HOLD reserved exclusively to the Sole Human Owner.

#### Explicit Prohibitions & Anti-Scope Invariants:
- **Zero External Public Routes:** No public internet routes, DNS records, or external web endpoints may be activated.
- **Zero CDN Edge Deployment:** No static assets, templates, or scripts may be deployed to content delivery networks.
- **Zero Production Database Deployment:** Qualification is confined exclusively to local, ephemeral, or isolated in-memory/sqlite instances.
- **Zero Real Customer or Personal Data:** Zero real employee PII, customer records, or production workplace data may be ingested (`H040-003`).
- **Zero Provider, Credential, or Account Mutations:** Zero cloud provider accounts, authentication secrets, or external API keys may be provisioned or mutated.

---

## 2. Qualification Architecture & Test Domain Matrix

The technical qualification suite systematically verifies the architectural guarantees defined in `ARC-V040-CHKL-001` (Checklist Model), `ARC-V040-CHKRULE-001` (Checklist Rules), and `ARC-V040-CHKLIFE-001` (Checklist Lifecycle):

| Test Domain ID | Focus Domain | Architectural Source | Governing Invariant | Key Verification Objective |
| :--- | :--- | :--- | :--- | :--- |
| **QDOM-01** | Question Types & Validation | `ARC-V040-CHKL-001` Sec 3 | `H040-006` | Validates all 6 alpha question types; enforces mandatory unit labels and bounds. |
| **QDOM-02** | Bilingual Localization | `ARC-V040-CHKL-001` Sec 7 | `H040-002` | Asserts 100% complete `en-US` and `th-TH` string mappings; zero monolingual templates. |
| **QDOM-03** | Conditional Rules & DAG | `ARC-V040-CHKRULE-001` Sec 2 | Acyclic Evaluation | Enforces strict forward-only topological ordering; rejects circular rule dependencies. |
| **QDOM-04** | Exclusion Scoring Isolation | `ARC-V040-CHKRULE-001` Sec 5 | Zero Scoring Bias | Confirms excluded questions (`EXCLUDED_BY_RULE`) are removed from scoring denominators. |
| **QDOM-05** | Lifecycle State Machine | `ARC-V040-CHKLIFE-001` Sec 2 | Irreversible Publication | Validates full 7-state lifecycle progression; rejects unauthorized jumps. |
| **QDOM-06** | Segregation of Duties | `ARC-V040-CHKLIFE-001` Sec 3 | `SOD-03` / `H040-004` | Enforces `reviewer != author`; blocks self-approval attempts. |
| **QDOM-07** | Cryptographic Sealing | `ARC-V040-CHKLIFE-001` Sec 5 | SHA-256 Immutability | Verifies canonical `content_digest` computation; blocks in-place mutation of published templates. |
| **QDOM-08** | Provenance & Derivation | `ARC-V040-CHKLIFE-001` Sec 6 | SemVer Progression | Asserts stable `template_id`, valid `predecessor_version`, and `derived_from_digest` lineage. |
| **QDOM-09** | Concurrency & Server Authority | `ARC-V040-DOMAIN-001` | `H040-005` (No LWW) | Simulates concurrent updates; confirms conflicting transitions quarantine immediately. |
| **QDOM-10** | Execution Pinning | `ARC-V040-CHKLIFE-001` Sec 7 | Audit Repeatability | Verifies active inspections remain bound to template version/digest; zero runtime hot-swapping. |
| **QDOM-11** | Traceability & Safe Failures | `ARC-V040-CHKRULE-001` Sec 7 | Fail-Closed Default | Confirms `UNKNOWN` and `NA` require non-blank justifications and trigger audit follow-up. |

---

## 3. Supported Alpha Question Types & Input Qualification (`QDOM-01`)

The qualification suite asserts that the template engine accepts only the **six authorized alpha question types** and enforces their specific boundary constraints:

1. **`PASS_FAIL_NA_UNKNOWN` (`H040-006`):**
   - Valid responses: `PASS`, `FAIL`, `NA`, `UNKNOWN`.
   - Mandatory non-blank justification note on `NA` and `UNKNOWN` (minimum 5 characters).
   - Automatic finding generation on `FAIL`.
2. **`SINGLE_CHOICE`:**
   - Requires exactly one valid `option_id` string from the declared options list.
   - Option-level finding trigger flags evaluated deterministically.
3. **`MULTI_CHOICE`:**
   - Array of unique valid `option_id` strings.
   - Enforces selection bounds (`min_selections <= count <= max_selections`); duplicates rejected.
4. **`NUMERIC_MEASUREMENT`:**
   - IEEE 754 float number.
   - Mandatory unit specification (`ohms`, `dBA`, `Celsius`, `psi`).
   - Range validation (`min_value <= val <= max_value`); warning and critical threshold evaluation.
5. **`TEXT_NOTE`:**
   - UTF-8 string constrained between 1 and 1000 characters.
   - Automatic trimming of leading/trailing whitespace; control-character sanitization.
6. **`EVIDENCE_ATTACHMENT`:**
   - Array of valid evidence reference identifiers (`evd_syn_*`).
   - MIME type restriction to `image/jpeg`, `image/png`, `application/pdf`.

---

## 4. Bilingual Localization & String Completeness (`QDOM-02`)

In adherence to `H040-002`, the qualification suite verifies that every user-facing checklist component provides valid, non-blank translations in both English (`en-US`) and Thai (`th-TH`):
- Template Title and Description.
- Section Title and Description.
- Question Prompt and Guidance.
- Option Labels for Single-Choice and Multi-Choice questions.
- Rule descriptions and finding summary templates.

- **Zero Monolingual Content:** A checklist template cannot be published if any prompt, title, guidance, or option label is missing either `en-US` or `th-TH`.

**Rejection Rule:** Any template containing missing translation keys, empty string values, or whitespace-only localized entries is rejected during schema validation.

---

## 5. Lifecycle State Transitions & Segregation of Duties (`QDOM-05`, `QDOM-06`, `QDOM-07`)

### 5.1 Seven-State Transition Engine
The qualification suite exercises the complete lifecycle state graph:
```
DRAFT -> UNDER_REVIEW -> APPROVED -> PUBLISHED_IMMUTABLE -> RETIRED / SUPERSEDED
  ^            |
  |-- REJECTED <
```

### 5.2 Segregation of Duties (`SOD-03`)
- **Self-Approval Denial:** If `submitter_id == reviewer_id` on an `ApproveChecklist` transition, the operation fails closed with `ErrSelfApprovalProhibited`.
- **Role Enforcement:** Transitions submitted by unprivileged roles (`Inspector`, `CAPA Owner`) fail closed with `ErrInsufficientRolePrivilege`.

### 5.3 Cryptographic Sealing & Immutability Boundary
- Upon transition to `PUBLISHED_IMMUTABLE`, the system computes:
  $$\text{content\_digest} = \text{SHA-256}(\text{CanonicalJSON}(\text{TemplatePayload}))$$
- Any subsequent attempt to modify or delete a published template fails closed with `ErrImmutableVersionMutation`.

---

## 6. Multi-Version Provenance & Derivation Lineage (`QDOM-08`)

Because published versions are immutable, updates require deriving a new iteration:
1. **Stable Identifier:** Derived iterations retain the identical `template_id` (`chk_syn_*`).
2. **Predecessor Tracking:** The new draft iteration explicitly records:
   - `predecessor_version`: Exact version string of the parent (e.g. `"1.0.0"`).
   - `derived_from_digest`: SHA-256 digest of the parent version.
   - `copied_by`: Synthetic user ID of the author initiating derivation.
   - `copied_at`: UTC timestamp of derivation.
3. **SemVer Progression:** Versions must increment monotonically (`1.0.0` $\to$ `1.1.0` or `2.0.0`). Non-incremental or duplicate version tags are rejected.
4. **Supersession Cascade:** When the successor version transitions to `PUBLISHED_IMMUTABLE`, the predecessor version automatically transitions from `PUBLISHED_IMMUTABLE` to `SUPERSEDED`, recording `successor_version`.

---

## 7. Concurrency, Server Authority & Conflict Quarantine (`QDOM-09`)

In strict conformance with foundation gate `H040-005`:
1. **Server Sole Authority:** All state transitions and question response commits are mediated exclusively by backend domain logic.
2. **Rejection of Last-Write-Wins (LWW):** When two inspectors or synchronization clients attempt concurrent writes against the same question or inspection session from different base versions, the second write is **never** permitted to silently overwrite the first.
3. **Quarantine on Conflict:** The conflicting update is trapped and transitioned into `QUARANTINED_CONFLICT`, emitting a structured audit alert for manual administrative triage.

---

## 8. Execution Pinning & Historical Retention (`QDOM-10`)

The qualification suite confirms that in-flight operational inspections are permanently protected from template lifecycle changes:
1. **Immutable Binding:** Instantiated inspections record `template_id`, `template_version`, and `content_digest` at creation time.
2. **Zero Runtime Hot-Swapping:** If a template is superseded by v1.1.0 or retired while an inspection is in progress, the active inspection continues to evaluate exclusively against the pinned v1.0.0 schema.
3. **Historical Audit Repeatability:** Superseded and retired checklist versions remain permanently stored and queryable in read-only mode to support retrospective safety and regulatory audits.

---

## 9. Deterministic Synthetic Fixtures & Lineage

The qualification suite provides a deterministic multi-version fixture set (`FIX-QUAL-V040-CHKL`):

```yaml
schema_version: "1.0.0"
fixture_id: "fix_syn_qualification_multi_version_v1"
template_id: "chk_syn_qual_plant_safety_01"

iterations:
  # Base Published Version
  - version: "1.0.0"
    lifecycle_status: "SUPERSEDED"
    title:
      en-US: "Plant Safety Qualification Baseline v1.0"
      th-TH: "เกณฑ์มาตรฐานการตรวจสอบความปลอดภัยโรงงาน v1.0"
    content_digest: "a1b2c3d4e5f60718293a4b5c6d7e8f90a1b2c3d4e5f60718293a4b5c6d7e8f90"
    successor_version: "1.1.0"

  # Derived Successor Version
  - version: "1.1.0"
    lifecycle_status: "PUBLISHED_IMMUTABLE"
    title:
      en-US: "Plant Safety Qualification Baseline v1.1"
      th-TH: "เกณฑ์มาตรฐานการตรวจสอบความปลอดภัยโรงงาน v1.1"
    predecessor_version: "1.0.0"
    derived_from_digest: "a1b2c3d4e5f60718293a4b5c6d7e8f90a1b2c3d4e5f60718293a4b5c6d7e8f90"
    content_digest: "b2c3d4e5f6a10718293a4b5c6d7e8f90a1b2c3d4e5f60718293a4b5c6d7e8f91"

pinned_inspection:
  inspection_id: "ins_syn_pinned_audit_01"
  pinned_template_id: "chk_syn_qual_plant_safety_01"
  pinned_version: "1.0.0"
  pinned_digest: "a1b2c3d4e5f60718293a4b5c6d7e8f90a1b2c3d4e5f60718293a4b5c6d7e8f90"
  status: "IN_PROGRESS"
```

---

## 10. Deterministic Negative Controls & Anti-Regressions

The qualification harness enforces explicit negative test assertions:
- **NEG-01 (Direct Publish Bypass):** Attempting `DRAFT` $\to$ `PUBLISHED_IMMUTABLE` without intermediate `UNDER_REVIEW` and `APPROVED` fails closed.
- **NEG-02 (Self-Approval Violation):** Review approval where `author_id == reviewer_id` fails closed (`SOD-03`).
- **NEG-03 (Cyclic Rule Dependency):** Templates with circular question rule dependencies fail closed during linting.
- **NEG-04 (Forward Rule Dependency):** Rule depending on a downstream question (`display_order_target < display_order_dependency`) fails closed.
- **NEG-05 (In-Place Mutation of Published Version):** Modification of any attribute on a `PUBLISHED_IMMUTABLE` template fails closed.
- **NEG-06 (Unregistered Question Type):** Question schemas utilizing non-alpha types fail validation.
- **NEG-07 (Monolingual Template Rejection):** Missing either `en-US` or `th-TH` localized strings fails validation.
- **NEG-08 (Unjustified NA / UNKNOWN Response):** Submission of `NA` or `UNKNOWN` without minimum 5-character justification is rejected.
- **NEG-09 (Missing Mandatory Evidence):** Submission of non-compliant response without mandatory attachment fails closed.
- **NEG-10 (Concurrent Overwrite Rejection):** Simultaneous writes against identical question state reject Last-Write-Wins and quarantine conflict.

---

## 11. Rerun Lineage & Reproducibility Verification

To guarantee audit repeatability:
- All qualification test runs execute hermetically against local synthetic fixtures.
- Test runs produce deterministic exit codes and structured JSON test execution logs.
- Rerun lineage records:
  - Base Commit OID: `42eaabc0822c47ef2950ce3945ed91ae6546f109`
  - Synthetic Fixture Digest: Verified via SHA-256.
  - Test Suite Digest: Verified via git commit tree.

---

## 12. Governance Boundaries & Non-Claims

In strict conformance with `HDEC-V040-FOUNDATION-054`:
1. **Planning & Qualification Specification Only:** This document establishes the technical qualification baseline and verification rules. It does not confer operational release, deployment, or residual-risk acceptance authority.
2. **Synthetic Data Mandate (`H040-003`):** 100% synthetic fixtures; zero real customer or personal data.
3. **Default-Deny Authority (`H040-004`):** Role-based authorization enforced across all lifecycle transitions.
4. **Server Authority & Conflict Quarantine (`H040-005`):** Server authority enforced; zero Last-Write-Wins.
5. **Pilot Non-Regulatory Checklist (`H040-006`):** Explicit `NA` and `UNKNOWN` handling without legal safety claims.
6. **Retained Holds (`H040-007` - `H040-011`):** All release, participant UAT, external routing, support ownership, and final milestone decisions remain strictly on **`HOLD`**.
