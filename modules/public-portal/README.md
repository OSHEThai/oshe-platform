# Public Portal Snapshot Resolution and Safety Shielding

- Module ID: `MOD-PUB`
- Roadmap topic: `V030-T06` (GitHub Issue #99 / V030-I026)
- Qualification suite: `V030-I028` (GitHub Issue #101)
- Implementation state: local standalone synthetic specification module

Owns public snapshot resolution, publication state verification, safety shielding, and non-leaking fail-closed access controls for external/public consumers.

## Architectural Boundaries & Non-Claims

- **Provisional In-Memory Model Only:** This module operates as a dependency-free, standalone local Go package (`oshe/public-portal`). It does NOT deploy a live web server, public DNS route, reverse proxy, or cloud CDN caching layer.
- **Sanitized Presentation Projections Only:** Serves exclusively sanitized, approved read-model snapshots derived from internal modules (`MOD-REP`). Zero operational database queries or live transactional joins are permitted.
- **Held Governance Gates:** Decision `H030-006` (Portal Staging & Publication), `H030-007` (Public Routes & CDN), and `H030-008` (Release & Deployment) remain on **HOLD**.

## Core Capabilities (`public_snapshot_resolver.go`)

- **Publication Lifecycle Verification:** Resolves snapshots strictly when in `PUBLISHED_IMMUTABLE` status. Unapproved drafts, staged versions, un-published approvals, withdrawn records, and superseded versions fail closed with non-leaking `NOT_FOUND` (`NEG-SNAP-04`).
- **Operational Query Prohibition (`NEG-SNAP-01`):** Any attempt to bypass snapshot resolution or execute live queries against operational tables fails closed immediately with `OPERATIONAL_QUERY_BLOCKED` (`ErrLiveQueryProhibited`).
- **Guessed-Identifier & Cross-Tenant Defense (`NEG-SNAP-02`, `NEG-SNAP-03`):** Probing non-existent or foreign tenant snapshot IDs returns generic, non-leaking `NOT_FOUND`, preventing resource existence discovery or organizational reconnaissance.
- **Temporal Effectiveness Bounding:** Public snapshots are served strictly within their `[EffectiveFrom, EffectiveTo]` window. Expired snapshots return `EXPIRED` (HTTP 410 Gone equivalent).
- **Search Engine & Cache Shielding (`NEG-SNAP-05`):** All public snapshot resolution responses mandate strict security headers:
  - `X-Robots-Tag: noindex, nofollow, noarchive`
  - `Content-Security-Policy: default-src 'self'`
  - `Cache-Control: private, no-cache, no-store`
- **Data Minimization & Mandatory Disclaimer:** Snapshots omit internal database autoincrements, credentials, and personal PII, and embed the mandatory `DERIVED_OUTPUT_NON_AUTHORITY` notice.

## Qualification Test Suite (`public_portal_qualification_test.go`)

The portal qualification test suite enforces deterministic local boundaries across all isolation, lifecycle, cache, and presentation requirements:

| Qualification Area | Test Function | Verified Boundary | Status |
| :--- | :--- | :--- | :--- |
| **Internal/Public Separation** | `TestQualification_Portal_InternalPublicSeparation_OperationalQueryBlocked` | Intercepts `IsOperationalQuery: true` with `DenialOperationalQueryBlocked` (`ErrLiveQueryProhibited`). | PASS |
| **Guessed Identifier Defense** | `TestQualification_Portal_GuessedIdentifier_AntiEnumeration` | Generic non-leaking `NOT_FOUND` on un-registered or probe IDs; anti-reconnaissance. | PASS |
| **Tenant Boundary Isolation** | `TestQualification_Portal_TenantBoundary_Isolation` | Cross-tenant snapshot queries fail closed with `NOT_FOUND`; zero cross-tenant leakage. | PASS |
| **Lifecycle State Verification** | `TestQualification_Portal_SnapshotLifecycleStates_OnlyPublishedImmutable` | Only `PUBLISHED_IMMUTABLE` resolves; `DRAFT`, `APPROVED`, `WITHDRAWN`, `SUPERSEDED` fail closed. | PASS |
| **Temporal Window Bounding** | `TestQualification_Portal_TemporalValidity_EffectiveAndExpiryWindows` | Rejects pre-effective queries (`NOT_FOUND`) and expired queries (`EXPIRED`). | PASS |
| **Stale Cache Prevention** | `TestQualification_Portal_StaleCache_CacheControlShielding` | Injects `Cache-Control: private, no-cache, no-store` on all success/error resolution responses. | PASS |
| **Link Leakage & Minimization** | `TestQualification_Portal_LinkLeakage_And_DataMinimization` | Mandatory `DERIVED_OUTPUT_NON_AUTHORITY` notice; absence of internal URLs, credentials, tokens. | PASS |
| **Search Engine Shielding** | `TestQualification_Portal_IndexingShielding_RobotsAndCSP` | Mandates `X-Robots-Tag: noindex, nofollow, noarchive` and `Content-Security-Policy: default-src 'self'`. | PASS |
| **Accessible Presentation** | `TestQualification_Portal_Accessibility_ReadModelPresentation` | Semantic, human-legible title/summary structure without unrendered control characters. | PASS |
| **Held Gate Invariant** | `TestQualification_Portal_LocalSyntheticNonClaims_H030_007_Hold` | Local in-memory execution only; Decision `H030-007` (Public Routes & CDN) remains on **HOLD**. | PASS |
