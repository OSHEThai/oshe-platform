# V020-I024 Failure, Quarantine, and Replay Operations Runbook

## 1. Operational Overview & Boundaries

This runbook establishes operational procedures for identifying, diagnosing, and safely resolving event and notification failures within the OSHE Platform Event, Outbox, and Job subsystem (`MOD-EVT`, topic `V020-T05`).

### Operational Invariants
- **Local Synthetic Boundary:** All dispatching, scheduling, outbox, and notification mechanisms operate strictly in-memory or through local sinks.
- **Prohibition of External Delivery:** No external SMTP, SMS, push, or webhook routes may be invoked or configured.
- **Tenant Isolation:** Cross-tenant operations are denied by default (`ErrCrossTenantDispatch`, `ErrCrossTenantReplay`, `ErrCrossTenantNotification`).
- **Non-Destructive Failure Reporting:** Delivery and processing failures must remain observable in diagnostics without altering or corrupting originating business state.

---

## 2. Observable Symptoms & Triage Matrix

| Failure Condition | Observable Symptom / Error | Subsystem State | Operational Action |
| :--- | :--- | :--- | :--- |
| **Poison Event Payload** | `ErrMaxRetriesExceeded`<br>`ErrRetryLimitReached` | `DeliveryRecord.Status = QUARANTINED` | Inspect `LastError`, fix consumer handler bug, execute authorized `Replay`. |
| **Transient Processing Failure** | `dispatch attempt N failed` | `DeliveryRecord.Status = RETRYING` | Monitor automated bounded retry sequence (up to `MaxAttempts = 3`). |
| **Duplicate Dispatch Attempt** | `ErrAlreadyDelivered` | `DeliveryRecord.Status = DELIVERED` | None; handler execution skipped to preserve consumer idempotency. |
| **Schema / Envelope Mismatch** | `ErrIncompatibleSchemaVersion`<br>`ErrUnsupportedEnvelopeVersion` | Staging rejected; 0 events staged | Verify producer payload against canonical version `1.0.0`. |
| **Local Sink Delivery Failure** | `ErrNotificationMaxRetries` | `NotificationRequest.Status = QUARANTINED` | Inspect `Diagnostics`, restore local sink buffer, execute `ReplayNotification`. |
| **Clock Skew Anomaly** | `ErrClockSkewDetected` | Job scheduling rejected | Correct system timer or reschedule with `DueAt >= ScheduledAt`. |
| **Cross-Tenant Violation** | `ErrCrossTenantAssociation`<br>`ErrCrossTenantDispatch` | Transaction or dispatch fails closed | Verify tenant token context; reject unauthorized cross-tenant dispatch. |

---

## 3. Quarantine Handling & Diagnostics

When an event or notification exhausts its configured retry limit (`MaxAttempts`), it transitions into the `QUARANTINED` state to prevent poison-pill loops from blocking queues.

### Diagnostic Inspection Checklist
1. **Retrieve Delivery Record:** Call `GetDeliveryRecord(consumerID, eventID)` to examine:
   - `Attempts`: Confirms attempt count reached `MaxAttempts`.
   - `LastError`: Exact error string reported by consumer handler.
   - `QuarantinedAt`: Timestamp when quarantine was entered.
2. **Retrieve Notification Record:** Call `GetNotification(requestID, tenantID)` to examine:
   - `Diagnostics`: Step-by-step attempt history and failure reasons.
   - `Status`: Must be `QUARANTINED` prior to replay.
3. **Verify Pre-Replay Integrity:**
   - Confirm consumer bug or sink failure is remediated.
   - Confirm target event has not already transitioned to `DELIVERED`.

---

## 4. Safe Replay Procedures

Replay is an explicit, authorized operation that re-introduces a quarantined item for controlled processing.

### Event Replay Procedure (`Dispatcher.Replay`)
1. **Authorization Requirements:**
   - `callerIdentity`: Must be non-blank and belong to authorized operations staff.
   - `callerTenantID`: Must strictly match `record.TenantID`.
2. **Execution Rules:**
   - If event is in `QUARANTINED`, `Replay` re-invokes the registered handler.
   - Upon success: status transitions to `DELIVERED`, `ReplayedCount` increments, and `QuarantinedAt` clears.
   - If replay fails: status remains `QUARANTINED`, `LastError` updates with the new error.
3. **Safety Guardrails:**
   - Replay on a `DELIVERED` event returns `ErrEventNotQuarantined`.
   - Replay with mismatched tenant returns `ErrCrossTenantReplay`.
   - Replay without actor identity returns `ErrUnauthorizedReplay`.

### Notification Replay Procedure (`Scheduler.ReplayNotification`)
1. **Authorization Requirements:**
   - Non-blank `callerIdentity` and matching `callerTenantID`.
2. **Execution Rules:**
   - Validates notification is currently `QUARANTINED`.
   - Delivers strictly to the configured local sink (`LOCAL_MEMORY` or `LOCAL_LOG`).
   - Upon success: status transitions to `DELIVERED` and `Diagnostics` logs the replay actor.

---

## 5. Schema Mismatch Disposition

Schema compatibility is enforced at ingestion time:
- **Rule:** Only `CurrentEnvelopeVersion = "1.0.0"` and `CurrentSchemaVersion = "1.0.0"` are accepted.
- **Fail-Closed Behavior:** Any envelope or schema version mismatch is rejected immediately during transaction staging (`Stage`).
- **No Poison Quarantine:** Mismatched schemas never enter the outbox or quarantine queues; producers receive synchronous rejection errors (`ErrIncompatibleSchemaVersion`).

---

## 6. Non-Runtime & Local Sink Boundary Declarations

Operators and automation must observe these architectural boundaries:
1. **No External Network Sinks:** Under no circumstances should real delivery credentials (SMTP, SendGrid, Twilio, AWS SNS) be configured in this slice.
2. **No Silent State Drops:** Unhandled events and failed deliveries must remain in observable records until explicitly cleared or replayed.
3. **Default-Deny Access:** Unauthenticated requests or cross-tenant lookups must fail closed without revealing record existence.
