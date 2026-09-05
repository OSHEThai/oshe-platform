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

## Lifecycle State Machine, Move Operations & Historical Preservation (`organization_lifecycle.go`)

- **Terminal Closed Operational Policy:** Enforces `StateClosed` (`CLOSED`) as an operational completion state. Closed entities remain queryable for audit and compliance but reject new operational records (`ErrEntityClosed`). Reopening is prohibited in operational policy (`ErrCannotReopenClosed`).
- **Reversible Local Simulation Harness:** Provides in-memory reversible state simulation (`SimulateReversibleTransition`) for preflight test assurance under Sole Human Owner decision H030-002 without enacting binding external authority.
- **Safe Re-Parenting & Move Operations:** Governs same-tenant site and area move operations (`MoveSiteToProject`, `MoveAreaToSite`). Enforces active source and destination requirements and strictly rejects cross-tenant moves (`ErrCrossTenantMove`) or moves into archived/closed targets (`ErrParentArchived`, `ErrParentClosed`).
- **Historical Attribution Preservation:** Employs `HistoricalScopeRecord` to capture immutable scope snapshots at transition points, ensuring past records and canonical paths remain permanently queryable without physical database deletions.

## External Party Registry & Contractor/Subcontractor Nesting Boundaries (`party.go`)

- **Tenant-Scoped External Party Registry:** Defines `Party` entities for clients, contractors, subcontractors, partners, and auditors (`PartyType`). External parties are strictly bounded to tenant scope and convey zero internal corporate membership or administrative rights.
- **Mandatory Internal Sponsor (`usr_*`):** All contractor and subcontractor project participations mandate an internal sponsor manager (`usr_*`). Registrations with blank or non-user sponsor IDs fail closed (`ErrBlankSponsorID`, `ErrInvalidSponsorID`).
- **Contractor-Subcontractor Nesting Ceiling (`MaxContractorNestingDepth = 1`):** Enforces a strict depth limit permitting only primary contractor (depth 0) to subcontractor (depth 1) nesting. Sub-subcontracting (depth 2+) is strictly prohibited (`ErrNestingDepthExceeded`).
- **Temporal & Scope Containment:** A subcontractor's validity window must be strictly bounded within its parent contractor's window (`ErrValidityWindowExceedsParent`). Subcontractors inherit their parent's project and cannot expand beyond their parent's site bounds (`ErrScopeMismatch`).
- **Parent Status Cascade:** Inactive, archived, or closed parent contractor participations cascade down, preventing subcontractor creation or active operations (`ErrParentClosed`, `ErrParentNotActive`).
- **Strict Non-Elevation & Lateral Denial:** External parties are barred from administrative or internal authority roles (`ErrElevationForbidden`). Lateral sibling contractor and cross-project access attempts are rejected (`ErrSiblingAccessDenied`).
- **Public Contract Redaction (`contracts/api/party_contract.go`):** Redacts internal database keys, bearer tokens, passwords, real PII (national IDs, emails, phone numbers), and authority fields from public views (`PartySummaryView`, `ProjectParticipationView`).
- **H030-002 Provisional Governance:** Confined strictly to in-memory, reversible local simulation and preflight validation fixtures pending successor owner gates.

## Party Lifecycle, Sponsor Reassignment & Historical Attribution (`party_lifecycle.go`)

- **Reversible Deactivation & Archival:** Governs in-memory soft deactivation of parties and project participations (`DeactivateParty`, `DeactivateParticipation`), transitioning active entities to `StateArchived` without hard deletion.
- **Sponsor Reassignment Protocol:** Safely transfers internal supervision of an active participation to a new validated internal manager (`usr_*`) via `ReassignSponsor`. Strictly rejects identical sponsors (`ErrSponsorUnchanged`), non-user IDs (`ErrInvalidSponsorID`), expired participations (`ErrParticipationExpired`), and closed/archived relationships (`ErrParticipationClosed`, `ErrParentNotActive`).
- **Project-Closure Cascade (`CascadeProjectClosure`):** Propagates project closure down to all bounded contractor and subcontractor participations, transitioning active participations to `StateClosed` while preserving existing archived states. Any operational action on a participation bounded to a closed project fails closed (`ErrProjectClosedCascade`).
- **Append-Only Historical Attribution Ledger (`PartyLifecycleLedger`):** Thread-safe in-memory audit ledger recording immutable event histories for party transitions (`HistoricalPartyLifecycleRecord`), participation state changes (`HistoricalParticipationLifecycleRecord`), and sponsor reassignments (`SponsorReassignmentRecord`). Enforces strict tenant boundary isolation across all history queries and guarantees past attribution cannot be mutated or purged.
- **H030-002 Simulation Posture:** All lifecycle transitions and reassignments operate as local, in-memory, reversible simulation fixtures (`SimulateReversiblePartyState`) with zero binding external authority or persistent database side effects.
