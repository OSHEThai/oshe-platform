# Integration Tests

Reserved for cross-component and infrastructure-backed validation.

## Milestone v0.3.0 Walking Skeleton Integration Harness

- Work item: `V030-I036` (GitHub Issue #109)
- Harness script: `tests/test_v030_walking_skeleton_integration_harness.py`
- Test fixtures: `tests/fixtures/integration/v030-walking-skeleton-fixtures.json`
- Implementation state: local in-memory deterministic simulation harness

### Overview & Architecture

The v0.3 Walking Skeleton executes an end-to-end synthetic journey connecting all primary v0.3.0 architectural subsystems (`MOD-ORG`, `MOD-IAM`, `MOD-PUB`, `MOD-PORTAL`, `MOD-REC`):

1. **Organization Hierarchy & Tenancy:** Configures multi-level hierarchy (`Tenant` -> `Company` -> `Business Unit` -> `Project` -> `Site` -> `Area`) with site time zone and locale inheritance.
2. **External Party Bounding:** Enforces mandatory internal user sponsorship (`usr_*`) and contractor-subcontractor nesting depth limits (`MaxContractorNestingDepth = 1`).
3. **Identity & Bearer Tokens:** Issues high-entropy bearer tokens (`oshe_tok_<64-hex>`), storing only SHA-256 digests in memory.
4. **Scoped Directory Projections:** Projects descriptive operational profiles with strict data minimization (zero PII/credentials) and exact-scope project partitioning.
5. **Role Assignments & 1-Hop Delegations:** Enforces least-privilege role bounds, 1-hop delegation ceilings (`MaxDelegationChainDepth = 1`), and non-delegable sovereign administrator authority.
6. **Operational Inspection Records & Evidence:** Binds operational findings to cryptographic SHA-256 evidence digests.
7. **Publication Snapshot Derivation & Redaction:** Filters internal data through approved allowlists, requires reviewer approval with `DecisionHash`, and seals canonical payload digests.
8. **Local Public Portal Resolution:** Emits mandatory HTTP shielding headers (`X-Robots-Tag: noindex, nofollow, noarchive`, `Content-Security-Policy: default-src 'self'`, `Cache-Control: private, no-cache, no-store`) and blocks live operational SQL queries (`ErrLiveQueryProhibited`).
9. **Gated Export Packages:** Enforces cross-tenant homogeneity and approved destination scope validation.
10. **Append-Only Audit Ledger:** Records monotonic, chronological audit entries with SHA-256 integrity digests across every state transition.

### Test Coverage Matrix

| Test Method | Category | Verified Posture |
| :--- | :--- | :--- |
| `test_run_id_and_traceability_format` | Traceability | Validates monotonic `run_*`, `corr_*`, and `caus_*` identifier formats. |
| `test_full_v030_organization_to_publication_journey_success` | End-to-End | Verifies 10-step full journey from tenant setup to export with audit continuity. |
| `test_cross_tenant_isolation_negative_controls` | Isolation | Proves cross-tenant queries fail closed across directory, portal, and export packages. |
| `test_role_and_delegation_negative_controls` | Authorization | Verifies multi-hop delegation denial, non-delegable admin, contractor barriers, and auditor read-only invariants. |
| `test_withdrawal_and_revocation_controls` | Lifecycle / Audit | Verifies emergency publication withdrawal, non-leaking resolution, and generation-based session revocation. |
| `test_migration_and_recovery_lineage` | Continuity | Demonstrates full state serialization and verified reconstruction with zero data loss. |
| `test_held_decisions_and_non_claims_invariant` | Governance | Confirms decisions `H030-003` through `H030-008` remain strictly on HOLD / PROHIBITED. |

### Execution Command

```powershell
python -m unittest discover -s tests -p test_v030_walking_skeleton_integration_harness.py -v
```

### Governance Boundaries & Non-Claims

- **Purely In-Memory Synthetic Harness:** All tests run locally in memory using synthetic fixtures (`usr_*`, `prj_*`, `ten_*`, `snp_*`).
- **Zero Live Services / Runtime Authority:** Zero live HTTP endpoints, Docker/WSL runtime containers, public DNS records, CDN edge caches, or database persistence are deployed or claimed.
- **Held Governance Gates:** Decisions `H030-003` through `H030-008` remain on **HOLD**.
