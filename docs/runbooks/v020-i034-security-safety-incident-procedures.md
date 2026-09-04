# V020-I034 Security, Privacy, and Product-Safety Incident Procedures Runbook

## 1. Operational Overview & Authority

This runbook establishes standard operational incident response procedures for managing security breaches, privacy violations, product-safety hazard emergencies, evidence integrity failures, and offline state conflicts within the OSHE Platform (`topic V020-T08`, issue #69).

### Governance & Decision Baseline
- **Governing Issue:** GitHub Issue #69 (`[V020-I034] Create Threat Model, Privacy Assessment, Product-Safety Hazard Log, and Incident Procedures`)
- **Governing Decision:** `HDEC-V020-GATE-B-APPROVAL-051`
- **Governing Lease:** `LEASE-V020-I034-THREAT-PRIVACY-SAFETY-PREWORK-288`
- **Operational Status:** Prework Draft & Simulation Runbook.
- **Safety Invariant:** Human life safety and evidentiary integrity take precedence over system availability. Under safety-critical ambiguity, the platform **must fail closed**.

---

## 2. Incident Severity Classification and Escalation SLA

| Severity Level | Operational Criteria | Escalation Target | Response SLA | System Action |
| :--- | :--- | :--- | :--- | :--- |
| **Severity 0 (`S0`)** | Cross-tenant data leak, evidence tampering/digest mismatch, false-negative safety evaluation, unauthorized finding closure, or compromised signing credentials. | **Sole Human Owner** & Security Lead | **Immediate (< 15 min)** | **Halt / Lockdown:** Revoke affected sessions, isolate tenant schema, freeze evidence storage, enforce fail-closed state. |
| **Severity 1 (`S1`)** | Outbox notification queue deadlock, stale checklist omitting mandatory checks, unverified action transition, or partial database migration failure. | Engineering Lead & Security Lead | **< 1 Hour** | **Quarantine / Fallback:** Quarantine failing queue, redirect queries to authoritative database, halt template instantiation. |
| **Severity 2 (`S2`)** | Derived search index delay, non-authoritative projection lag, or transient memory storage saturation. | Test & Quality Lead | **< 4 Hours** | **Rebuild / Resync:** Display staleness warning badges, trigger background projection resynchronization. |
| **Severity 3 (`S3`)** | Non-misleading UI label errors, non-safety translation typos. | Engineering Backlog | Next sprint cycle | Log ticket in backlog. |

---

## 3. Emergency Incident Response Procedures

### Procedure 1: Cross-Tenant Isolation Breach Response (`PROC-INC-01`)
*Trigger: Audit log alerts `ErrTenantMismatch`, `ErrTenantOverrideForbidden`, or unauthorized cross-tenant object access.*

1. **Immediate Containment:**
   - Call `RevocationManager.RevokeSubject(compromisedSubjectID)` to invalidate all active session tokens immediately.
   - If breach affects an entire API client or compromised integration, revoke tenant API credentials.
2. **Schema & Data Isolation:**
   - Lock down affected tenant PostgreSQL schema (`org`, `workflow`, `evidence`) into read-only mode to prevent data tampering.
3. **Forensic Audit Collection:**
   - Extract append-only audit trail from `MOD-REC` for the incident window:
     ```
     SELECT * FROM audit.events WHERE tenant_id = :affected_tenant AND created_at >= :breach_start ORDER BY monotonic_sequence ASC;
     ```
   - Identify all accessed entity IDs, queried records, and origin IP addresses.
4. **Tenant & Owner Notification:**
   - Prepare incident notification packet citing affected data fields, breach timeline, and containment confirmation.
   - Escalate to Sole Human Owner for regulatory disclosure assessment (PDPA Section 37(4)).

---

### Procedure 2: Evidence Tampering and Digest Mismatch Response (`PROC-INC-02`)
*Trigger: Storage retrieval or background integrity scanner reports SHA-256 mismatch against PostgreSQL metadata.*

1. **Asset Quarantine:**
   - Immediately set file metadata status to `QUARANTINED` in `MOD-EVD`:
     ```
     UPDATE evidence.files SET status = 'QUARANTINED' WHERE file_id = :corrupted_file_id;
     ```
   - Scoped storage adapter blocks downloads and returns `ErrFileCorruptedOrQuarantined`.
2. **Downstream Report Suppression:**
   - Invalidate any generated PDF or inspection reports in `MOD-REP` referencing `:corrupted_file_id`.
   - Prevent export of affected inspection packages.
3. **Provenance Reconstruction:**
   - Retrieve original upload transaction from `MOD-EVT` outbox history and `MOD-REC` audit logs.
   - Verify if mismatch was caused by storage bit-rot, transmission error, or unauthorized S3 write.
4. **Restoration / Re-Ingestion:**
   - If original file exists in local pre-upload cache with verified SHA-256 digest, re-upload via authorized admin command.
   - Record re-ingestion audit record linked to the original incident ticket.

---

### Procedure 3: Safety-Critical State Corruption & Unsafe Closure (`PROC-INC-03`)
*Trigger: Inspection finding prematurely marked "RESOLVED" or "CLOSED" without required evidence or independent review.*

1. **Fail-Closed Reversion:**
   - Immediately transition finding state back to `IN_REVIEW` or `OPEN` in `MOD-WFA`.
   - Lock finding from subsequent modifications until supervisor sign-off:
     ```
     UPDATE workflow.actions SET state = 'REOPENED', locked = true WHERE action_id = :action_id;
     ```
2. **Emergency Field Notification:**
   - Emit high-priority emergency alert to site safety manager via local notification sink.
   - Dispatch immediate workplace advisory: physical hazard may remain active and unmitigated.
3. **Transition Verification:**
   - Inspect authorization log to determine which identity authorized the invalid state transition.
   - If caused by business rule qualification failure, confirm rollback invariant was preserved (`TestQualification_RollbackAndNonMutationOnDenial`).
4. **Physical Remediation Verification:**
   - Require new photographic evidence with fresh timestamp and verified SHA-256 hash before permitting re-evaluation.

---

### Procedure 4: Credential Exposure or Token Leak Containment (`PROC-INC-04`)
*Trigger: Leaked Bearer token, session key, or development signing key discovered in public repository or logs.*

1. **Global Session Revocation:**
   - Execute emergency revocation across all active sessions issued under the compromised key:
     ```
     RevocationManager.RevokeAllSessionsBefore(revocationTimestamp)
     ```
2. **Scrub Diagnostics and Logs:**
   - Run `tools/validate_repository.py` to confirm zero secrets remain in repository files.
   - Flush transient telemetry buffers in local OpenTelemetry collectors.
3. **Synthetic Key Regeneration:**
   - Generate new synthetic development keys and update local `.env` and `.devcontainer` configurations.
   - Re-run local CI and test suites to confirm clean execution under new keys.

---

### Procedure 5: Offline Synchronization Conflict & Causal Desync (`PROC-INC-05`)
*Trigger: Edge node submits inspection findings that contradict existing server findings without causal sequence.*

1. **Prohibit Blind Last-Write-Wins (LWW):**
   - The platform kernel strictly rejects blind overwrites of safety states.
2. **Fork Preservation:**
   - Store offline submission as a concurrent `CONFLICT_CANDIDATE` branch in `MOD-WFA`, preserving both Inspector A and Inspector B findings.
3. **Audit Recording:**
   - Record both client-asserted timestamps and server receipt timestamps in `MOD-REC`.
4. **Supervisor Dispute Resolution:**
   - Flag inspection record as `REQUIRES_RECONCILIATION`.
   - Present side-by-side diff of conflicting responses to an authorized Site Safety Supervisor.
   - Once resolved, record supervisor decision, reasoning, and reconciliation signature.

---

### Procedure 6: Transactional Outbox Poison Queue Recovery (`PROC-INC-06`)
*Trigger: Outbox dispatcher marks event `QUARANTINED` after exhausting bounded retry attempts.*

1. **Inspect Quarantined Event:**
   - Call `GetDeliveryRecord(consumerID, eventID)` to examine `LastError` and attempt history per `v020-i024-failure-replay-operations.md`.
2. **Verify Schema Conformance:**
   - Validate event envelope against canonical schema version in `MOD-CTR`.
3. **Fix Consumer / Schema Defect:**
   - If caused by consumer parsing bug, deploy software fix to consumer handler.
4. **Execute Authorized Replay:**
   - Call `Dispatcher.Replay(callerIdentity, callerTenantID, eventID)`.
   - Verify event transitions from `QUARANTINED` to `DELIVERED` without duplicate side effects.

---

## 4. Post-Incident Review and Evidence Archival

Following containment of any S0 or S1 incident:
1. **Root-Cause Analysis (RCA):** The Security Lead must compile an RCA document identifying the failure vector, affected boundaries (`TB-01`–`TB-08`), and preventive control gaps.
2. **Negative Regression Test Creation:** An automated negative test reproducing the incident scenario must be added to the appropriate module test suite (e.g., `negative_controls_test.go`).
3. **Evidence Bundle Update:** Incident logs, forensic snapshots, and RCA reports must be bundled and presented to the Sole Human Owner prior to release qualification.
