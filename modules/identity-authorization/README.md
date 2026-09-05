# Identity and Authorization

- Module ID: `MOD-IAM`
- Roadmap topic: `V020-T02`
- Implementation state: architecture scaffold only

Owns identity references, memberships, role assignments, scoped authorization evaluation, and revocation evidence. It may depend on stable organization identifiers from `MOD-ORG`.

Entitlement does not replace authorization. Clients, integrations, extensions, and AI agents cannot supply or bypass authoritative tenant and scope decisions.

## Local Identity & Bearer Token Management (`local_identity.go`, `session_revocation.go`)

- **Singular Synthetic Identity:** Manages local synthetic user subjects (`usr_*`) with active/disabled lifecycle states (`IdentityActive`, `IdentityDisabled`).
- **Cryptographic Bearer Tokens:** Issues high-entropy session tokens (`oshe_tok_<64-hex>`) stored exclusively as SHA-256 digests in memory. Raw token secrets are never persisted.
- **Session Revocation Registry:** Real-time token revocation registry supporting targeted session revocation, subject-level session invalidation, and policy generation tracking with an append-only audit trail.

## Scoped Authorization & Access Policy (`access_policy.go`)

- **Default-Deny Access Evaluation:** `AccessEvaluator` asserts valid authenticated claims, active tenant membership, exact role-to-action permissions, and project/site scope matches before granting access.
- **Non-Leaking Diagnostics:** All authorization failures return stable, typed denial reasons (`DenialReasonCrossTenant`, `DenialReasonScopeMismatch`, `DenialReasonArchivedRecord`) that prevent cross-tenant or sibling-project object existence discovery.

## Synthetic Scoped Directory Profiles (`directory_profile.go`)

- **Singular Identity Truth:** Maps one trusted synthetic subject (`usr_*`) to multiple operational company and project contexts (`DirectoryProfile`). A single worker may hold distinct operational designations across projects while maintaining a single identity record.
- **Strict Profile-to-Authorization Separation:** Directory profiles are descriptive directory projections only. They NEVER store, duplicate, or manage credentials, passwords, session tokens, role grants, or permissions. Possessing a directory profile conveys ZERO operational authority (`AssertNoAuthorizationBypass`).
- **Data Minimization & Sanitization:** Exposes only sanitized display attributes (`DisplayName`, `JobTitle`, `Department`, `AssignedAreas`). Strictly omits personal contact details (phone numbers, personal emails, national IDs), internal database keys, and security credentials.
- **Active Exact-Scope Directory Discovery (`DirectoryRegistry`):** Thread-safe in-memory directory registry supporting project and site boundary filtering. Cross-scope or non-matching queries return non-leaking empty results (`[]DirectoryProfile{}`), preventing project reconnaissance or peer contractor discovery (`H030-005`).
- **Provisional Governance:** Implemented strictly as local, in-memory, synthetic-only simulation fixtures under approved human decisions `H030-003`, `H030-004`, and `H030-005`.
