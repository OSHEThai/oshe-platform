# Organization and Tenancy

- Module ID: `MOD-ORG`
- Roadmap topic: `V020-T01`
- Implementation state: architecture scaffold only

Owns tenant, company, project, site, area, and stable context references. Candidate public contracts provide organization-context commands, queries, and identifier assertions.

Authoritative identity remains internal to this module. Other modules may use reviewed identifiers and public contracts but may not write its private state or tables.

## Identifier & Tracking Contract (`identifiers.go`)

- **Canonical Prefix Standardization:** Enforces canonical, non-reusable prefixed identifiers (`ten_*`, `cmp_*`, `bnu_*`, `prj_*`, `ste_*`, `ara_*`, `prt_*`, `usr_*`, `corr_*`, `caus_*`, `idem_*`).
- **Idempotency Guarantees:** Tenant-scoped idempotency keys make duplicate resource creations safe by returning recorded resource identifiers upon identical payload digest match while rejecting conflicting payloads (`ErrIdempotencyConflict`).
- **Operational Tracing:** Provides cryptographically generated correlation and causation tracking contexts (`TrackingContext`) for cross-module observability.
- **External Reference Mapping & Anti-Abuse:** Governs bidirectional mapping of external system identifiers to internal canonical IDs. Enforces anti-enumeration bounds (minimum length, character restrictions, duplicate mapping prevention, and trivial pattern rejection) strictly within tenant boundaries.

## Hierarchy Constraints, Scope Propagation & Localization Rules (`organization_hierarchy.go`)

- **Six-Level Organizational Hierarchy:** Governs the canonical hierarchy structure (`Tenant` $\rightarrow$ `Company` $\rightarrow$ `Business Unit` $\rightarrow$ `Project` $\rightarrow$ `Site` $\rightarrow$ `Area`) alongside sponsored third-party contractor relationships (`SponsoredParty`).
- **Parent-Child & Project-Site Relationship Invariants:** Enforces strict parent consistency checks (`ErrParentMismatch`, `ErrProjectSiteMismatch`) and freezes child attachment under archived parents (`ErrParentArchived`).
- **Identifier Validation Behavior:** Enforces strict canonical prefix format (`<prefix>_<token>` with min 8 chars) when canonical prefixes are presented, while preserving backward-compatible non-blank synthetic slugs.
- **Site Time Zone & Locale Ownership:** Site entities own operational IANA time zones (default `Asia/Bangkok`) and BCP 47 locales (default `th-TH`, fallback `en-US`). Unspecified values on child `Area` entities automatically inherit their parent Site's configuration.
- **Non-Authoritative Scope Propagation:** `ResolvedScope` produces descriptive, immutable canonical path projections embedding `DERIVED_OUTPUT_NON_AUTHORITY`. Scope resolution validates tenant equality only and conveys zero lateral, upward, or implicit operational authority.
