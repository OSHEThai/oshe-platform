# Organization and Tenancy

- Module ID: `MOD-ORG`
- Roadmap topic: `V020-T01`
- Implementation state: architecture scaffold only

Owns tenant, company, project, site, area, and stable context references. Candidate public contracts provide organization-context commands, queries, and identifier assertions.

Authoritative identity remains internal to this module. Other modules may use reviewed identifiers and public contracts but may not write its private state or tables.
