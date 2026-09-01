# Data and Projection Boundaries

Status: `DRAFT_STATIC_BOUNDARY_RECORD`

The selected ADR-0006 baseline keeps PostgreSQL as the authoritative store. A transactional outbox publishes committed changes to NATS JetStream for search indexing, cache invalidation, notifications, and later event consumers.

- Meilisearch stores locale-aware search projections only; it is not an authoritative record store.
- Valkey is a cache, rate-limit, and short-lived coordination service only; it is never a source of truth.
- A failed search or cache projection is rebuilt from PostgreSQL through the transactional-outbox path. It does not authorize direct alteration of authoritative records.
- API responses apply tenant, site, classification, and authorization checks before returning projected search results.

This static record creates no schema, migration, service, container, event contract, or runtime claim. Those changes remain separately leased and reviewed.
