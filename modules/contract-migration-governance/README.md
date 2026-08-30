# Contract and Migration Governance

- Module ID: `MOD-CTR`
- Roadmap topic: `V020-T07`
- Implementation state: architecture scaffold only

Owns API, event, schema, migration-registry metadata, compatibility decisions, deprecation, and supersession records. It references public contracts from every module without owning their business state.

Contract and migration changes require deterministic compatibility evidence and cannot silently rewrite completed records or accepted evidence.
