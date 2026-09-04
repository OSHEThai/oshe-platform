# Organization and Tenancy

- Module ID: `MOD-ORG`
- Roadmap topic: `V020-T01`
- Implementation state: architecture scaffold only

Owns tenant, company, project, site, area, and stable context references. Candidate public contracts provide organization-context commands, queries, and identifier assertions.

Authoritative identity remains internal to this module. Other modules may use reviewed identifiers and public contracts but may not write its private state or tables.

## Identifier & Tracking Contract (`identifiers.go`)

- **Canonical Prefix Standardization:** Enforces canonical, non-reusable prefixed identifiers (`ten_*`, `cmp_*`, `prj_*`, `ste_*`, `ara_*`, `usr_*`, `corr_*`, `caus_*`, `idem_*`).
- **Idempotency Guarantees:** Tenant-scoped idempotency keys make duplicate resource creations safe by returning recorded resource identifiers upon identical payload digest match while rejecting conflicting payloads (`ErrIdempotencyConflict`).
- **Operational Tracing:** Provides cryptographically generated correlation and causation tracking contexts (`TrackingContext`) for cross-module observability.
- **External Reference Mapping & Anti-Abuse:** Governs bidirectional mapping of external system identifiers to internal canonical IDs. Enforces anti-enumeration bounds (minimum length, character restrictions, duplicate mapping prevention, and trivial pattern rejection) strictly within tenant boundaries.
