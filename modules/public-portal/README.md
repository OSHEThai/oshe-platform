# Public Portal Snapshot Resolution and Safety Shielding

- Module ID: `MOD-PUB`
- Roadmap topic: `V030-T06` (GitHub Issue #99 / V030-I026)
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
