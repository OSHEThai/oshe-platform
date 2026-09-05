---
document_id: ARC-V040-CONFLICT-001
title: v0.4.0 OSHE Inspect Offline Conflict Classification, Server Authority, Protected-State Rejection, Quarantine, and Manual Reconciliation Baseline
document_type: architecture_specification
document_version: 1.0.0
lifecycle_status: DRAFT
status: APPROVED_FOR_LOCAL_DEVELOPMENT
date: "2026-09-05"
author_role: Architecture and Data Lead
author_pane: w9:p22
governing_issue: "GitHub Issue #126"
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
credit_boundary: OFFLINE_CONFLICT_SPECIFICATION_ONLY_NO_EXTERNAL_SYNC_OR_DEPLOYMENT_CLAIM
---

# v0.4.0 OSHE Inspect Offline Conflict Classification, Server Authority, Protected-State Rejection, Quarantine, and Manual Reconciliation Baseline

## 1. Executive Summary & Governance Posture

### 1.1 Authority Baseline & Purpose
This architectural specification establishes the authoritative **Offline Conflict Classification, Server Authority, Protected-State Rejection, Quarantine Container, and Manual Reconciliation Baseline** for **Milestone v0.4.0 - OSHE Inspect Private Alpha** in fulfillment of **GitHub Issue #126 (`[V040-I015] Implement Offline Conflict Classification, Server Authority, Protected-State Rejection, Quarantine, and Manual Reconciliation`)** under Roadmap Topic `V040-T03` and standing owner foundation decision `HDEC-V040-FOUNDATION-054`.

The primary objective is to define a conservative, fail-closed conflict resolution architecture for mobile and offline field operations that guarantees:
- Strict **Server Authority (`H040-005`)** over all protected lifecycle states, permissions, and organization boundaries.
- **Categorical Prohibition of Last-Write-Wins (LWW)** timestamp reconciliation.
- Standardized, deterministic classification of synchronization anomalies into six discrete conflict classes (**C0 through C5**).
- Non-destructive **Conflict Quarantine** ensuring incoming evidence, responses, and field observations are never silently discarded.
- Visible, structured **Manual Reconciliation Packets** requiring explicit human attribution for dispute resolution.

### 1.2 Retained Unselected Policies & Non-Claims
In strict compliance with `ASN-V040-I015-OFFLINE-CONFLICT-001` and `HDEC-V040-FOUNDATION-054`:
1. **Offline Authority Model (`HUMAN_OWNED_UNSELECTED`):** While this specification implements conservative alpha conflict rules and 24-hour lease boundaries, long-term statutory lease governance, production sync intervals, and automated priority rules remain human-owned under Issue #126.
2. **Binding Scoring Policy (`HUMAN_OWNED_UNSELECTED`):** Local score calculations and conflict impact projections are advisory only; binding scoring algorithms remain human-owned under Issue #136 (`V040-I025`).
3. **Finding Closure Policy (`HUMAN_OWNED_UNSELECTED`):** Reinspections, finding closures, and safety risk acceptances require explicit human owner authority under Issue #134 (`V040-I023`).
4. **Synthetic-Only Data Boundary (`H040-003`):** All conflict models, simulation fixtures, and test payloads operate exclusively on synthetic alpha data (`usr_*`, `ins_*`, `fnd_*`, `ten_*`). Zero real customer data, real corporate records, or actual employee PII is processed.
5. **Retained Human Gate Holds:** Foundation gates `H040-007` (Technical Release Authorization), `H040-008` (Real Participant / Private-Alpha UAT Onboarding), `H040-009` (Binding Support Ownership), `H040-010` (External Environment/Route Activation), and `H040-011` (Final Milestone Outcome) remain strictly on **`HOLD`**.

---

## 2. Server Authority & Anti-LWW Invariants (`H040-005`)

Under approved foundation gate `H040-005`, synchronization between distributed client devices and central backend services is governed by unyielding server authority:

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                          SERVER AUTHORITY CONSTRAINTS                           │
├────────────────────────────────┬────────────────────────────────────────────────┤
│ Constraint Principle           │ Architectural & Operational Invariant          │
├────────────────────────────────┼────────────────────────────────────────────────┤
│ Sole State Machine Authority   │ Central service is the single source of truth  │
│                                │ for all protected entity states.               │
├────────────────────────────────┼────────────────────────────────────────────────┤
│ Categorical Anti-LWW           │ Timestamp-based Last-Write-Wins is strictly    │
│                                │ prohibited. Client clock values are un-trusted.│
├────────────────────────────────┼────────────────────────────────────────────────┤
│ Optimistic Concurrency Tokens  │ Every client submission must carry expected    │
│                                │ base_version and base_entity_digest.           │
├────────────────────────────────┼────────────────────────────────────────────────┤
│ Protected-State Immutability   │ Completed, finalized, or closed records reject │
│                                │ incoming offline modifications fail-closed.    │
├────────────────────────────────┼────────────────────────────────────────────────┤
│ Non-Destructive Quarantine     │ Conflicting client submissions are captured in │
│                                │ immutable quarantine storage without data loss.│
└────────────────────────────────┴────────────────────────────────────────────────┘
```

### 2.1 Untrusted Client Time
Client device clocks are inherently subject to skew, tampering, time-zone misconfiguration, and battery drainage resets. Central services never use client-supplied timestamps (`client_recorded_at`) to adjudicate state precedence. Precedence is determined exclusively by monotonic server sequence numbers and cryptographic version digests.

## 3. C0–C5 Conflict Classification Catalog

Every synchronization attempt submitted by a client is evaluated against the following mutually exclusive, exhaustive conflict classification hierarchy:

| Class ID | Conflict Classification Name | Trigger Condition / Mismatch Profile | Automatic System Handling | Quarantine Required? | Resulting Client Diagnostic |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **`C0`** | **Idempotent / Benign Duplicate** | Identical submission re-transmitted (same `idempotency_key`, matching `payload_digest`). | Acknowledge with existing server state; zero mutation. | No | `ACK_IDEMPOTENT_DUPLICATE` (HTTP 200) |
| **`C1`** | **Non-Overlapping Additive Merge** | Concurrent updates touch completely disjoint sections/questions with identical base version. | Auto-merged into server state if verified disjoint. | No (auto-resolved) | `ACK_MERGED_ADDITIVE` (HTTP 200) |
| **`C2`** | **Stale Base-Version Concurrent Edit** | Client `base_version < server_current_version` on shared questions or field attributes. | Rejects direct write; preserves client draft in quarantine. | **YES** | `ERR_CONFLICT_STALE_BASE` (HTTP 409) |
| **`C3`** | **Competing Workflow State Transition** | Client attempts to mutate record that server has transitioned to a protected/terminal state. | Rejects client transition; seals client draft into quarantine. | **YES** | `ERR_PROTECTED_STATE_IMMUTABLE` (HTTP 409) |
| **`C4`** | **Revoked / Expired Authority at Sync** | Inspector role, delegation, or assignment was revoked or expired while client was offline. | Fails closed on authorization; quarantines payload. | **YES** | `ERR_REVOKED_AUTHORITY_AT_SYNC` (HTTP 403) |
| **`C5`** | **Cryptographic Integrity / Uncertainty** | Payload SHA-256 digest mismatch, corrupted media attachment, or malformed signature. | Fails closed on security; quarantines for investigation. | **YES** | `ERR_INTEGRITY_UNCERTAINTY` (HTTP 422) |

---

## 4. Conflict Evaluation State Machine

```
                  [Client Sync Submission]
                             │
                             ▼
               {Is Idempotency Key Duplicate?}
                 ├── YES (Payload Match) ────> [C0: Acknowledge Existing State]
                 └── NO
                             │
                             ▼
              {Is Integrity Checksum Valid?}
                 ├── NO ─────────────────────> [C5: Quarantine (Integrity Error)]
                 └── YES
                             │
                             ▼
               {Is Submitter Authorized Now?}
                 ├── NO (Revoked/Expired) ───> [C4: Quarantine (Authority Revoked)]
                 └── YES
                             │
                             ▼
            {Is Server Entity in Protected State?}
                 ├── YES (Completed/Closed) ─> [C3: Quarantine (Protected State)]
                 └── NO
                             │
                             ▼
               {Does Base Version Match Server?}
                 ├── YES ────────────────────> [Apply Server State Mutation]
                 └── NO (Version Stale)
                             │
                             ▼
           {Are Field Changes Strictly Disjoint?}
                 ├── YES ────────────────────> [C1: Auto-Merge Disjoint Fields]
                 └── NO (Collision) ─────────> [C2: Quarantine (Stale Edit)]
```

---

## 5. Conflict Quarantine & Manual Reconciliation Schema

When an incoming sync payload evaluates to **C2, C3, C4, or C5**, the server encapsulates the collision within an immutable `ConflictRecord`:

### 5.1 Conflict Record Entity Definition
```yaml
# Schema: OSHE-Conflict-Record-v1.0
schema_version: "1.0.0"
conflict_id: "cnf_syn_01a2b3c4d5e6"
tenant_id: "ten_syn_alpha"
target_entity_type: "INSPECTION" # "INSPECTION" | "FINDING" | "ACTION"
target_entity_id: "ins_syn_20260905_001"
conflict_class: "C2_STALE_BASE_VERSION"
detected_at: "2026-09-05T16:30:00Z"
submitter_subject: "usr_syn_inspector_01"
submitter_role: "Inspector"
client_device_id: "dev_syn_pixel7_01"

state_divergence:
  server_version: 3
  server_digest: "d1a7a00112233445566778899aabbccddeeff00112233445566778899aabbcc0"
  server_state: "IN_PROGRESS"
  client_base_version: 1
  client_base_digest: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
  client_submission_digest: "a9b8c7d6e5f40123456789abcdef0123456789abcdef0123456789abcdef01"

quarantined_payload:
  responses:
    - question_id: "qst_syn_ground_01"
      response_value: "FAIL"
      inspector_notes: "Cable severed at termination lug"
      evidence_refs: ["evd_syn_photo_01"]

reconciliation_packet:
  status: "PENDING_MANUAL_RECONCILIATION" # PENDING | RESOLVED_SERVER | RESOLVED_CLIENT | RESOLVED_SPLIT | DISCARDED
  assigned_reconciler_role: "Independent Reviewer"
  reconciled_by: null
  reconciled_at: null
  resolution_type: null
  resolution_justification: null
  audit_journal_ref: null
```

### 5.2 Manual Reconciliation Protocol
1. **Quarantine Notification:** Upon conflict detection, `MOD-WFA` stages a high-priority reconciliation item in the Independent Reviewer's work queue.
2. **Side-by-Side Comparison:** The reconciliation interface presents the current server record beside the quarantined client submission.
3. **Four Authorized Resolution Actions:**
   - **`RESOLVE_ACCEPT_SERVER`:** Discards client conflicting changes; confirms server state as authoritative. Quarantined evidence is preserved in audit logs.
   - **`RESOLVE_OVERWRITE_WITH_CLIENT`:** Requires Independent Reviewer explicit sign-off; applies client changes over server state as a new version ($N+2$).
   - **`RESOLVE_MANUAL_MERGE`:** Reviewer selects individual field values from server and client to produce a harmonized composite version.
   - **`RESOLVE_SPLIT_NEW_FINDING`:** Reviewer accepts client observations by spawning a separate, linked child finding or inspection record without corrupting the primary record.
4. **Mandatory Justification:** All resolution actions mandate non-blank justification (`min_length: 10` characters) permanently recorded in `MOD-REC`.

---

## 6. Audit Obligations & Diagnostic Events (`MOD-REC`, `MOD-EVT`)

Every conflict evaluation emits strongly-typed events to ensure complete operational observability:

### 6.1 Diagnostic Events
1. `ConflictDetectedEvent`: Emitted upon identification of C0–C5 conflicts, carrying `conflict_id`, `conflict_class`, and divergence metadata.
2. `ConflictQuarantinedEvent`: Emitted when an incoming submission is stored in quarantine, locking client submission data without data loss.
3. `ConflictReconciledEvent`: Emitted upon human resolution, detailing `resolution_type`, `reconciler_subject`, and resulting entity version.

### 6.2 Audit Journaling (`MOD-REC`)
All conflict evaluations, quarantine admissions, and human reconciliations generate immutable audit entries carrying:
- `record_id`: Strongly-typed UUID (`rec_[0-9a-f]{16}`).
- `entity_id` & `conflict_id`.
- `actor_subject` & `actor_role`.
- `from_state` (`QUARANTINED`) to `to_state` (`RECONCILED`).
- `audit_digest`: Tamper-evident SHA-256 seal.

---

## 7. Synthetic Test Fixtures

The following synthetic YAML fixture models the C0 through C5 conflict catalog for deterministic automated test execution:

```yaml
schema_version: "1.0.0"
fixture_id: "fix_syn_conflict_scenarios_v1"

scenarios:
  - scenario_id: "scen_c0_duplicate"
    conflict_class: "C0"
    description: "Re-transmission of identical inspection payload"
    server_entity:
      id: "ins_syn_001"
      version: 2
      digest: "digest_alpha_01"
      state: "IN_PROGRESS"
    client_submission:
      idempotency_key: "idem_syn_12345"
      base_version: 2
      payload_digest: "digest_alpha_01"
    expected_outcome:
      http_status: 200
      system_action: "ACKNOWLEDGE_EXISTING"
      quarantine: false

  - scenario_id: "scen_c2_stale_edit"
    conflict_class: "C2"
    description: "Inspector submits update against stale base version"
    server_entity:
      id: "ins_syn_002"
      version: 3
      digest: "digest_server_v3"
      state: "IN_PROGRESS"
    client_submission:
      idempotency_key: "idem_syn_67890"
      base_version: 1
      payload_digest: "digest_client_v1"
    expected_outcome:
      http_status: 409
      system_action: "QUARANTINE_STALE_EDIT"
      quarantine: true
      error_code: "ERR_CONFLICT_STALE_BASE"

  - scenario_id: "scen_c3_protected_state"
    conflict_class: "C3"
    description: "Offline update submitted for inspection already marked FINALIZED on server"
    server_entity:
      id: "ins_syn_003"
      version: 4
      digest: "digest_server_v4"
      state: "FINALIZED"
    client_submission:
      idempotency_key: "idem_syn_11223"
      base_version: 3
      payload_digest: "digest_client_attempt"
    expected_outcome:
      http_status: 409
      system_action: "REJECT_PROTECTED_STATE"
      quarantine: true
      error_code: "ERR_PROTECTED_STATE_IMMUTABLE"

  - scenario_id: "scen_c4_revoked_authority"
    conflict_class: "C4"
    description: "Inspector assignment was revoked while device was offline"
    server_entity:
      id: "ins_syn_004"
      version: 2
      state: "IN_PROGRESS"
      assigned_inspector: "usr_syn_inspector_02" # Reassigned away from submitter
    client_submission:
      submitter_subject: "usr_syn_inspector_01" # Former inspector
      base_version: 1
    expected_outcome:
      http_status: 403
      system_action: "QUARANTINE_UNAUTHORIZED"
      quarantine: true
      error_code: "ERR_REVOKED_AUTHORITY_AT_SYNC"

  - scenario_id: "scen_c5_integrity_mismatch"
    conflict_class: "C5"
    description: "Submission payload digest fails verification against media attachment"
    server_entity:
      id: "ins_syn_005"
      version: 1
      state: "IN_PROGRESS"
    client_submission:
      claimed_digest: "sha256_declared_01"
      computed_digest: "sha256_tampered_02"
    expected_outcome:
      http_status: 422
      system_action: "QUARANTINE_INTEGRITY_ERROR"
      quarantine: true
      error_code: "ERR_INTEGRITY_UNCERTAINTY"
```

---

## 8. Governance Boundaries & Non-Claims

In strict compliance with `HDEC-V040-FOUNDATION-054`:

1. **Zero Last-Write-Wins:** Server authority is absolute; client timestamps cannot override server state (`H040-005`).
2. **Evidence Preservation:** Quarantining preserves all field submissions without silent data loss.
3. **Synthetic-Only Data Policy (`H040-003`):** All conflict models, IDs, and payloads operate exclusively on synthetic alpha fixtures. Zero real customer data or PII is processed.
4. **No Deployment or Release Claim:** Gates `H040-007` through `H040-011` remain strictly on `HOLD`. Zero live synchronization services, cloud storage endpoints, or public routes are activated.
