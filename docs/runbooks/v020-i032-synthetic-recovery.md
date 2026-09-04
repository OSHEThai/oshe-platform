# V020-I032 Synthetic Migration, Recovery, and Rollback Operations Runbook

## 1. Operational Overview & Authority

This runbook defines the operational procedures and safety boundaries for database schema migration, partial-failure recovery, backfilling, rollback, forward-fix, and backup/restore qualification within the OSHE Platform (`MOD-CTR`, topic `V020-T07`, issue #67).

### Governance & Decision Baseline
- **Governing Issue:** GitHub Issue #67 (`[V020-I032] Implement Non-Destructive Migration and Rollback Qualification`)
- **Governing Decision:** `HDEC-V020-GATE-B-APPROVAL-051`
- **Human Gate Deferral:** `H020-005` (Database Migration Execution) remains explicitly **DEFERRED**.
- **Execution Authority:** Synthetic in-memory prework and qualification testing only. No live database migration is authorized.

---

## 2. Safety Invariants & Prohibited Actions

The following constraints are strictly enforced across all operational and engineering workflows:

| Constraint / Rule | Enforcement Level | Operational Meaning |
| :--- | :--- | :--- |
| **No Live SQL / Database Execution** | **MANDATORY** | No connections to PostgreSQL, CockroachDB, SQLite, or external database engines. |
| **No Container Execution** | **MANDATORY** | Docker, Podman, and containerized test fixtures are prohibited in this slice. |
| **No Customer / Production Data** | **MANDATORY** | Only synthetic, dependency-free mock fixtures may be processed. |
| **No Destructive Drops (Phase M3)** | **MANDATORY** | `DROP TABLE`, `DROP COLUMN`, or data purge operations are prohibited without a separate owner decision. |
| **Cross-Module Direct Writes** | **PROHIBITED** | Modules may only write to their own module-owned schemas and tables (`table_prefix`). |
| **Shared Tables / Foreign Joins** | **PROHIBITED** | Direct SQL joins across private module schemas are prohibited. Integration occurs via contracts/events. |

---

## 3. Migration & Recovery Procedures

All migration activities follow the topological module dependency DAG defined in `database/migration-manifest.yaml` and `database/schema-ordering.json`:
`MOD-ORG` $\to$ `MOD-IAM` $\to$ `MOD-REC` $\to$ `MOD-EVD` $\to$ `MOD-CFG` $\to$ `MOD-EVT` $\to$ `MOD-WFA` $\to$ `MOD-REP` $\to$ `MOD-CTR`.

### Procedure 1: Forward Upgrade (`UPGRADE_SUCCESS`)
1. Verify pre-upgrade database baseline in phase `M0`.
2. Apply migrations sequentially in strict topological DAG order.
3. For each module step, verify schema creation and seed verification.
4. Record disposition `UPGRADE_SUCCESS` upon all 9 modules reaching target version `0.2.0`.

### Procedure 2: Partial-Failure Isolation & Preservation (`PARTIAL_FAILURE_PRESERVED`)
1. Prior to executing each module step, take an atomic transaction savepoint or state snapshot.
2. If a migration step fails (e.g. syntax error or constraint conflict):
   - Immediately abort the failed step.
   - Revert state to the pre-step snapshot.
   - Preserve all previously committed module migrations.
   - Record disposition `PARTIAL_FAILURE_PRESERVED`.
3. Prohibit speculative continuation or silent partial execution.

### Procedure 3: Non-Destructive Data Backfill (`BACKFILL_VERIFIED`)
1. For schema updates introducing new mandatory fields (e.g. SHA-256 digests or revision markers):
   - Execute an in-place backfill script iterating over existing records.
   - Compute values deterministically without mutating historical primary keys or immutable timestamps.
   - Verify all rows satisfy non-null and format constraints.
2. Record disposition `BACKFILL_VERIFIED`.

### Procedure 4: Non-Destructive Rollback (`ROLLBACK_SUCCESS`)
1. When a migration set is rejected prior to promotion:
   - Rollback transaction to the clean baseline checkpoint.
   - Ensure zero uncommitted or speculative tables remain.
   - Verify baseline schema integrity.
2. Record disposition `ROLLBACK_SUCCESS`.

### Procedure 5: Non-Destructive Forward-Fix (`FORWARD_FIX_SUCCESS`)
1. When a partial failure occurs during phase `M2`:
   - Never issue `DROP TABLE` or destructive rollbacks.
   - Author an explicit corrective migration script (`0.2.0-patch1`) addressing the constraint or column definition.
   - Apply the patch to the failed module step.
   - Resume remaining topological migrations.
2. Record disposition `FORWARD_FIX_SUCCESS`.

### Procedure 6 & 7: Synthetic Backup & Restore (`BACKUP_COMPLETED` / `RESTORE_VERIFIED`)
1. **Backup:**
   - Extract schema metadata, table records, and migration phase states.
   - Generate canonical JSON snapshot and compute SHA-256 digest.
   - Record disposition `BACKUP_COMPLETED`.
2. **Restore:**
   - Initialize clean target state.
   - Ingest snapshot and apply recorded schemas and tables.
   - Verify SHA-256 digest matches the pre-backup manifest.
   - Record disposition `RESTORE_VERIFIED`.

---

## 4. Dispositions Reference Matrix

| Disposition Key | Qualification Phase | Verification Check |
| :--- | :--- | :--- |
| `UPGRADE_SUCCESS` | M1 Forward Upgrade | All 9 modules migrated in topological order to v0.2.0. |
| `PARTIAL_FAILURE_PRESERVED` | M2 Failure Triage | Prior committed modules retained; failed step cleanly aborted. |
| `BACKFILL_VERIFIED` | M1/M2 Data Enrichment | All historical synthetic rows populated with valid digests. |
| `ROLLBACK_SUCCESS` | M2 Rollback | Dirty tables discarded; baseline snapshot cleanly restored. |
| `FORWARD_FIX_SUCCESS` | M2 Remediation | Corrective patch applied forward; all 9 modules completed. |
| `BACKUP_COMPLETED` | M0-M2 Backup | Deterministic snapshot created with 64-char SHA-256 digest. |
| `RESTORE_VERIFIED` | M0-M2 Restore | State restored from backup with matching SHA-256 digest. |

---

## 5. Human Gate H020-005 Deferral Status

In strict accordance with release management protocol:
- **Status:** **DEFERRED**
- **Rationale:** Live database migrations require approved staging infrastructure, live backup verification, and explicit human sign-off.
- **Evidence Ready:** This synthetic qualification suite confirms all migration paths, failure recoveries, and topological dependencies are fully verified in-memory prior to gate unblocking.
