# Database Architecture and Migration Ordering

## 1. Governance Reference and H020-005 Boundary

This directory houses the module-owned persistence schemas, migration ordering definitions, and baseline installation specifications for **`v0.2.0 — Shared Platform Kernel`**.

- **Governing Issue**: [OSHEThai/oshe-platform Issue #64](https://github.com/OSHEThai/oshe-platform/issues/64) (`[V020-I029] Implement Module-Owned Database Schemas and Migration Ordering`).
- **Governing Human Decision**: `HDEC-V020-GATE-B-APPROVAL-051` (*v0.2.0 Gate B Entry and Local-Synthetic Implementation Baseline*).
- **Sole Human Owner Gate**: **`H020-005`** (`DEFERRED_TO_ARCHITECTURE_GATE`).
  - *Blocking Rule*: AI implements provisional schemas and migration tooling; final ownership boundary is human-approved once per release.
  - *Provisional State*: All manifests and ordering records herein are **provisional static specifications** pending formal H020-005 release authorization. Zero runtime execution or clean-install qualification is claimed.

---

## 2. Module Persistence Ownership Invariants

Per `HDEC-V020-GATE-B-APPROVAL-051` (decision `V020-D04`) and `docs/architecture/v0.2.0-system-context-and-module-baseline.md`:

1. **Strict Module Data Authority**: Each of the nine approved kernel modules owns its schema namespace, tables, state, and domain invariants.
2. **Zero Shared Tables**: Direct sharing of relational tables across modules is strictly `PROHIBITED`.
3. **No Direct Cross-Module Writes**: No module may directly insert, update, or delete records in another module's tables.
4. **No Private Table Joins**: Cross-module queries must not bypass service/contract boundaries with private SQL joins.
5. **Transactional Outbox Coordination**: Cross-module communication is conducted via versioned in-memory Go contracts or transactional outbox events dispatched to NATS JetStream.

---

## 3. Schema Namespaces and Table Prefixes

The nine approved modules are allocated distinct, isolated table prefixes:

| Module ID | Module Name | Namespace | Table Prefix | Dependency Predecessors |
|---|---|---|---|---|
| `MOD-ORG` | Organization and Tenancy | `org` | `org_` | *(Root context)* |
| `MOD-IAM` | Identity and Authorization | `iam` | `iam_` | `MOD-ORG` |
| `MOD-REC` | Records and Audit | `rec` | `rec_` | `MOD-ORG`, `MOD-IAM` |
| `MOD-EVD` | Files and Evidence | `evd` | `evd_` | `MOD-ORG`, `MOD-IAM` |
| `MOD-CFG` | Configuration and Checklist | `cfg` | `cfg_` | `MOD-ORG`, `MOD-IAM`, `MOD-REC` |
| `MOD-EVT` | Events, Outbox and Jobs | `evt` | `evt_` | `MOD-ORG`, `MOD-IAM` |
| `MOD-WFA` | Workflow and Action | `wfa` | `wfa_` | `MOD-CFG`, `MOD-IAM`, `MOD-EVD`, `MOD-REC`, `MOD-EVT` |
| `MOD-REP` | Reporting and Localization | `rep` | `rep_` | `MOD-WFA`, `MOD-REC`, `MOD-EVD` |
| `MOD-CTR` | Contract and Migration Governance | `ctr` | `ctr_` | *(Validates all public schemas)* |

---

## 4. Migration Lifecycle Phases (`V020-D08`)

- **Phase M0 (Clean Install)**: Establishes base schemas on an empty database in strict topological order.
- **Phase M1 (Forward Upgrade)**: Non-destructive schema and data migrations applied incrementally.
- **Phase M2 (Partial Recovery & Forward-Fix)**: Non-destructive forward-fix and recovery migrations under simulated partial failure.
- **Phase M3 (Destructive Migration)**: Destructive schema changes are strictly `PROHIBITED` without a separate explicit Sole Human Owner decision.

---

## 5. Directory Layout & Artifacts

- `database/migration-manifest.yaml`: Machine-readable declaration of module schemas, table prefixes, and dependencies.
- `database/schema-ordering.json`: Topological execution sequence and phase metadata.
- `database/migrations/<module>/`: Future Goose migration SQL scripts (governed under subsequent execution leases).
