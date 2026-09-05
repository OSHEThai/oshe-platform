---
document_id: OPS-V040-ONBOARD-001
title: v0.4.0 OSHE Inspect Private Alpha Onboarding, Training, Support, Maintenance, Incident Communication, Manual Fallback, and Session-Operations Pack (PLANNING_PREWORK_ONLY)
document_type: operations_specification
document_version: 1.0.0
lifecycle_status: DRAFT
status: HOLD_PENDING_OWNER_GATE_H040_009
date: "2026-09-05"
author_role: Architecture and Data Lead
author_pane: w9:p22
governing_issue: "GitHub Issue #147"
authority_source: HDEC-V040-FOUNDATION-054
governing_decisions:
  - HDEC-V040-FOUNDATION-054
  - HDEC-V030-ENTRY-AND-POLICY-052
  - ADR-0005
  - ADR-0006
  - ADR-0007
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
retained_unselected_operations:
  support_owner: HOLD_NOT_SELECTED
  support_operating_hours: HOLD_NOT_SELECTED
  support_response_expectations: HOLD_NOT_SELECTED
  support_channel_route: HOLD_NOT_SELECTED
  participant_communication: HOLD_NOT_SELECTED
  actual_training_commitment: HOLD_NOT_SELECTED
  client_device_baseline: NOT_SELECTED
  evidence_privacy_handling: NOT_SELECTED
  attachment_size_limit: NOT_SELECTED
  evidence_immutability_control: NOT_SELECTED
credit_boundary: PLANNING_PREWORK_ONLY_NO_OPERATIONAL_SUPPORT_OR_PARTICIPANT_COMMITMENT
---

# v0.4.0 OSHE Inspect Private Alpha Onboarding, Training, Support, Maintenance, Incident Communication, Manual Fallback, and Session-Operations Pack (PLANNING_PREWORK_ONLY)

## 1. Executive Summary & Governance Posture

### 1.1 Authority Baseline & Purpose
This operational specification establishes the authoritative **Private-Alpha Onboarding, Training, Support, Maintenance, Incident Communication, Manual Fallback, and Session-Operations Pack** for **Milestone v0.4.0 - OSHE Inspect Private Alpha** within `OSHEThai/oshe-platform`. It fulfills the deliverable requirements of **GitHub Issue #147 (`[V040-I036] Create Private-Alpha Onboarding, Training, Support, Maintenance, Incident Communication, Manual Fallback, and Session-Operations Pack`)** under Roadmap Topic `V040-T08` and standing owner foundation decision `HDEC-V040-FOUNDATION-054`.

The primary objective is to define a planning-only operational prework package that outlines:
- Role-specific synthetic onboarding for the four approved alpha roles (`H040-001`).
- Clear scope and limitation disclaimers regarding private alpha boundaries.
- High-level privacy and evidence handling planning controls (`H040-003`).
- Synthetic training-assessment exercises and scenario validation fixtures.
- Incident severity classification and manual fallback tabletop runbooks.
- Generic session-operation checklists for simulated testing sessions.

### 1.2 Categorical Distinction: Synthetic Rehearsal vs. Real-User Validation
A cardinal governance boundary governing this operations pack is the strict demarcation between **synthetic rehearsal** and **empirical real-user validation**:
1. **Synthetic Rehearsal Scope (Authorized Prework):** All workflows, tabletop exercises, onboarding guides, and session checklists detailed herein operate exclusively against synthetic test fixtures (`usr_*`, `ins_*`, `chk_*`, `ten_*`) in local simulated environments.
2. **Real-User Operational Validation (Strictly Held):** Under no circumstances does this document authorize real participant recruitment, live user communication, real device issuance, or actual field training. Gate `H040-008` (Real Participant / Private-Alpha UAT Authorization) remains strictly on **`HOLD`** pending explicit owner screening and authorization.

### 1.3 Retained Unselected Operations & Support Attributes (`H040-009 HOLD`)
In strict adherence to `ASN-V040-I036-ONBOARDING-SUPPORT-PREWORK-001` and `HDEC-V040-FOUNDATION-054`, Human Gate `H040-009` (Binding Support and Manual-Fallback Ownership) remains on **`HOLD`**. The following operational support parameters are formally designated as **`HOLD_NOT_SELECTED`**:
- **Support Owner:** `HOLD_NOT_SELECTED` (No individual, team, or vendor is bound to support obligations).
- **Operating Hours:** `HOLD_NOT_SELECTED` (No operational SLA or support coverage window is active).
- **Response Expectations:** `HOLD_NOT_SELECTED` (No binding response or resolution time guarantees).
- **Support Channel Route:** `HOLD_NOT_SELECTED` (No ticketing system, hotline, helpdesk email, or chat channel is deployed).
- **Participant Communication:** `HOLD_NOT_SELECTED` (No outbound user communication or notification routing is active).
- **Actual Training Commitment:** `HOLD_NOT_SELECTED` (No live instructional delivery or classroom commitment is enacted).

---

## 2. Role-Specific Synthetic Onboarding Architecture

Under foundation decision `H040-001`, Milestone v0.4.0 operates as a standalone, single-tenant private alpha vertical slice structured around four discrete functional roles:

```
┌────────────────────────────────────────────────────────────────────────────────────────┐
│                          PRIVATE-ALPHA ROLE ONBOARDING MATRIX                          │
├────────────────────┬─────────────────────────────┬─────────────────────────────────────┤
│ Approved Role      │ Core Alpha Responsibilities │ System Boundaries & Denials         │
├────────────────────┼─────────────────────────────┼─────────────────────────────────────┤
│ Checklist Author   │ Author, version, and edit   │ Cannot execute inspections, log     │
│                    │ synthetic checklist models. │ findings, or review own checklists. │
├────────────────────┼─────────────────────────────┼─────────────────────────────────────┤
│ Inspector          │ Download work packages,     │ Cannot alter checklist templates,   │
│                    │ capture responses/evidence. │ review own findings, or close CAPA. │
├────────────────────┼─────────────────────────────┼─────────────────────────────────────┤
│ CAPA Owner         │ Receive findings, submit    │ Cannot verify, review, or self-close│
│                    │ remediation evidence proof. │ assigned corrective action items.   │
├────────────────────┼─────────────────────────────┼─────────────────────────────────────┤
│ Independent        │ Verify inspections, audit   │ Cannot author checklists or execute │
│ Reviewer           │ evidence, resolve conflicts.│ primary inspections on same scope.  │
└────────────────────┴─────────────────────────────┴─────────────────────────────────────┘
```

### 2.1 Synthetic Account Provisioning
All test operators are assigned synthetic accounts adhering to the singular identity model (`MOD-IAM`):
- Format: `usr_syn_<role>_<id>` (e.g. `usr_syn_inspector_01`, `usr_syn_reviewer_01`).
- Cryptographic Bearer Tokens: High-entropy session tokens (`oshe_tok_<64-hex>`) stored exclusively as SHA-256 digests in browser session memory. Raw credentials are never persisted to disk.

### 2.2 Client Device & Browser Setup Baseline (NOT_SELECTED / Generic Planning Control)
- **Specific Browser Builds and Hardware Baselines:** **`NOT_SELECTED`**. Specific browser version baselines, physical hardware device lists, and strict pixel viewport bounds are not selected or asserted in this prework pack.
- **Generic Planning Control:** In accordance with foundation principle `H040-002`, client testing is planned around responsive web environments on Chrome/Edge and Android Chrome without asserting specific browser build versions, native client installers, or physical hardware requirements.

---

## 3. Scope & Alpha Limitation Notice

All onboarding documentation must prominently communicate the following architectural boundaries:

1. **Standalone Single-Tenant Slice:** The platform operates strictly in a single-tenant configuration. Cross-tenant federation and enterprise multi-tenancy are deferred to v0.5.0.
2. **Synthetic Non-Regulatory Checklist:** Inspection checklists (e.g. `chk_syn_pilot_plant_safety_v1`) are fictionalized synthetic models. Zero legal safety compliance certification or regulatory reporting is claimed.
3. **Synthetic Test Data Only (`H040-003`):** No real plant hazards, actual site measurements, or live customer records may be entered into the system.
4. **Untrusted Optical Scans:** Scanning an asset code conveys zero operational authority and bypasses zero permission checks (`H040-004`).
5. **Server Authority & Conflict Quarantine (`H040-005`):** The central server is the sole state authority. Offline edits never overwrite server state via last-write-wins; conflicting edits are quarantined for manual review.

---

## 4. Privacy, Security & Evidence Handling Planning Boundaries

### 4.1 Evidence Privacy & Metadata Handling (NOT_SELECTED / Generic Planning Control)
- **Specific Metadata Stripping Mechanics:** **`NOT_SELECTED`**. Detailed EXIF tag parsing algorithms, geotag stripping implementations, and specific binary size ceilings are not selected or asserted in this operational prework pack.
- **Generic Planning Control:** In accordance with `H040-003`, evidence handling operates under strict privacy minimization principles. Any captured media in simulated trials must exclude real employee PII or sensitive operational details without asserting implementation-level EXIF parsing.

### 4.2 Evidence Immutability & Storage (NOT_SELECTED / Generic Planning Control)
- **Specific Immutability Flags & Attachment Limits:** **`NOT_SELECTED`**. Specific binary size thresholds (e.g. 10 MiB) and object status codes are not selected in this pack.
- **Generic Planning Control:** Evidence collected during synthetic trials is treated as read-only test artifacts without making live cloud storage or binary persistence claims.

### 4.3 Session Boundaries (Generic Planning Control)
- **Specific Lease Timeouts:** **`NOT_SELECTED`**.
- **Generic Planning Control:** Test operators operate within bounded synthetic sessions without asserting client-side lease durations or token zeroization implementations.

---

## 5. Synthetic Training-Assessment Fixture & Scenarios

To validate operator readiness during simulated trials, the following four competency exercises are established as planning templates:

### Exercise 1: Not Applicable (`NA`) Response Justification
- **Scenario:** Inspector encounters an inspection question not relevant to the current synthetic area.
- **Competency Requirement:** Inspector selects `NA` and provides a non-blank explanatory justification note.
- **System Verification:** Verifies that the question is removed from the active denominator without negative compliance distortion (`H040-006`).

### Exercise 2: Unknown (`UNKNOWN`) Response Protocol
- **Scenario:** Inspector encounters an un-evaluable condition due to a simulated physical barrier.
- **Competency Requirement:** Inspector selects `UNKNOWN` and enters a written justification.
- **System Verification:** Verifies that an automatic supervisory verification task is staged and the inspection is flagged provisional.

### Exercise 3: Immediate Controls on Critical Hazards (`H040-004`)
- **Scenario:** Inspector observes an imminent danger condition during simulated inspection.
- **Competency Requirement:** Inspector logs a `CRITICAL` finding and enters mandatory temporary immediate controls.
- **System Verification:** Confirms that omitting immediate controls fails closed with an explicit validation error.

### Exercise 4: Offline Interruption Recovery & Conflict Quarantine (`H040-005`)
- **Scenario:** Simulated browser interruption during response capture; concurrent edit exists on server.
- **Competency Requirement:** Inspector verifies draft recovery from local storage and observes conflict quarantine notification.

---

## 6. Incident Severity & Manual Fallback Tabletop Runbooks

### 6.1 Severity Classification Matrix

| Severity Level | Operational Impact Profile | System Response & Technical Escalation |
| :--- | :--- | :--- |
| **`SEV-1` (Critical)** | Data integrity violation, unauthorized privilege escalation, PII leakage, or total sync failure. | Immediate engagement of kill-switch; halt test session; alert Engineering Lead. |
| **`SEV-2` (Major)** | Core inspection workflow blocked, upload failure, or false conflict quarantine loop. | Engage manual paper fallback; export local diagnostics; notify Reviewer. |
| **`SEV-3` (Moderate)** | Non-blocking UI glitch, localized translation defect, or slow response rendering. | Log diagnostic issue ticket; proceed with test session using desktop workaround. |
| **`SEV-4` (Minor)** | Cosmetic layout misalignment, minor wording typo, or non-functional styling quirk. | Record in post-session feedback log; no operational interruption. |

### 6.2 Manual Fallback Tabletop Runbooks

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    MANUAL FALLBACK TABLETOP WORKFLOW                    │
├─────────────────────────────────────────────────────────────────────────┤
│ 1. Extended Network Outage / Synchronization Block:                     │
│    - Inspector continues inspection in offline mode.                    │
│    - If session becomes locked, transition to physical paper checklist. │
│                                                                         │
│ 2. Local Storage Pressure Warning (Generic Planning Control):           │
│    - System suspends media capture upon storage warning.                │
│    - Text responses continue to commit to local storage.                │
│    - Operator records manual reference notes on paper log.              │
│                                                                         │
│ 3. Client Device Malfunction:                                           │
│    - Operator switches to secondary supported test device.              │
│    - Re-authenticates; server restores last synced state.               │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 7. Session-Operation Checklist & Protocol

### 7.1 Pre-Session Protocol
- [ ] Verify test environment base commit matches authoritative release branch.
- [ ] Confirm `provider_routes_enabled = 0` via static policy validation.
- [ ] Provision synthetic test accounts (`usr_syn_*`) for each operator role.
- [ ] Initialize local client storage in a clean state.

### 7.2 In-Session Protocol
- [ ] Monitor active sync queue depth and local draft commitments.
- [ ] Verify non-blank justification on any `NA` or `UNKNOWN` selections.
- [ ] Confirm immediate control notes entered for all critical findings.
- [ ] Inspect conflict holding container if concurrency collision occurs.

### 7.3 Post-Session Protocol
- [ ] Validate that all client-queued responses have synced to central server.
- [ ] Verify append-only audit trail captures all state changes.
- [ ] Execute session logout and clear local test session state.
- [ ] Compile diagnostic export packages and error logs for engineering review.
- [ ] Reset synthetic test database to clean baseline.

---

## 8. Governance Boundaries & Operational Prohibitions

In strict compliance with `HDEC-V040-FOUNDATION-054`:

1. **Zero Real-User Recruitment:** No real inspectors, contractors, or site operators may be contacted, recruited, or onboarded under this specification (`H040-008` remains on HOLD).
2. **Zero Operational Support Commitment:** Operating hours, response SLAs, support channels, and support staffing remain strictly unselected (`H040-009` remains on HOLD).
3. **Zero Production Infrastructure:** No cloud infrastructure, live DNS routes, or production databases may be activated (`H040-007`, `H040-010` remain on HOLD).
4. **No Automatic Issue Closure:** Pull requests delivering this prework pack must not contain automatic Issue-closing keywords for Issue #147.
