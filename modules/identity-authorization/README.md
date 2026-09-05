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

## Directory Resolution, Profile Lifecycle & Append-Only History (`directory_resolution.go`)

- **Duplicate Identifier Collision Rejection:** Enforces uniqueness of directory profile identifiers (`ProfileID`) within tenant boundaries, rejecting duplicate registration attempts with `ErrDuplicateIdentifierCollision`.
- **Explicit False-Merge Prohibition:** Distinct synthetic subjects (`usr_*`) can NEVER be merged, aliased, or consolidated, even if they share identical display names or job designations (`AssertNoFalseMerge`, `AssertDistinctSubjects`, `ErrFalseMergeProhibited`). The `Subject` of a directory profile is permanently immutable.
- **Structural Identity Immutability:** Structural identifiers (`ProfileID`, `Subject`, `TenantID`, `CompanyID`, `ProjectID`, `SiteID`) are strictly immutable (`ErrStructuralIdentityImmutable`). Only sanitized non-structural attributes (`DisplayName`, `JobTitle`, `Department`, `AssignedAreas`) may be updated via `UpdateProfileAttributes`.
- **Safe Profile Inactivation:** Inactivation transitions active profiles to `INACTIVE` (`InactivateProfile`), automatically excluding them from default directory discovery while preserving full historical identity. Inactive profiles reject attribute updates (`ErrProfileInactive`).
- **Append-Only Context History Ledger (`DirectoryResolutionLedger`):** An in-memory, thread-safe audit trail capturing immutable historical records (`DirectoryProfileHistoryRecord`) for all initial registrations, non-structural attribute mutations, and inactivation events.
- **Tenant Boundary Isolation:** Profile and subject audit trails (`GetProfileAuditTrail`, `GetSubjectAuditTrail`) strictly enforce caller tenant equality, preventing cross-tenant information leakage and guaranteeing zero physical record deletion.

## Provisional Scoped Authorization Matrix & Delegation Engine (`authorization_matrix.go`)

- **Provisional Prework Governance (`H030-003` / Issue #90):** Implements a provisional, local-only authorization matrix, least-privilege role/permission catalog, separation-of-duty (SOD) conflict engine, source-authority verification, and delegation bounds as an AI prework candidate. Invariant: It does NOT select or bind a final authority model or protected authority owner. All bindings remain strictly provisional, in-memory, and local-only.
- **Hierarchical Scope Levels:** Formalizes explicit scope levels (`ScopeLevelTenant` > `ScopeLevelCompany` > `ScopeLevelProject` > `ScopeLevelSite` > `ScopeLevelArea`) and validates that role assignments and delegation requests remain strictly within permitted scope depths.
- **Protected Sovereign Authorities:** Categorizes sensitive sovereign permissions (`PermOrgTenantManage`, `PermIdentityUserManage`, `PermIdentityRoleAssign`, `PermIdentitySessionRevoke`, `PermAuditExport`, `PermLegalHoldManage`, `PermPortalSnapshotPublish`, `PermPortalSnapshotWithdraw`, `PermDelegationGrant`) and protected roles (`RoleTenantAdmin`) which can never be delegated (`ErrProtectedAuthorityNonDelegable`).
- **Least-Privilege Role Catalog:**
  - `RoleTenantAdmin`: Sovereign tenant administration; bounded strictly to `TENANT` scope; non-delegable (`MaxDelegationDays = 0`).
  - `RoleProjectManager`: Operational inspection and site leadership (`PROJECT`, `SITE` scopes); max delegation 14 days.
  - `RoleInspector`: Field inspection execution and finding creation (`PROJECT`, `SITE`, `AREA` scopes); cannot approve inspections; max delegation 7 days.
  - `RoleAuditor`: Independent oversight; strictly read-only inspection and audit export access (`TENANT`, `COMPANY`, `PROJECT`, `SITE` scopes); zero operational create/update rights; max delegation 14 days.
  - `RoleViewer`: Passive read-only observer across hierarchy scopes; max delegation 30 days.
  - `RoleContractor`: External bounded partner (`PROJECT`, `SITE`, `AREA` scopes); submit inspection responses and remediate assigned findings only; max delegation 7 days.
  - `RoleSupport`: Technical diagnostic review (`PROJECT`, `SITE` scopes); max delegation 3 days.
- **Separation-of-Duties (SOD) Conflict Engine:**
  - `SOD-01`: Inspector vs. Auditor separation across overlapping project/company scopes (`ErrSODConflict`).
  - `SOD-02`: External Contractor vs. Tenant Administrator prohibition (scope-insensitive; `ErrSODConflict`).
  - `SOD-02B`: External Contractor vs. Project Manager conflict on overlapping scopes (`ErrSODConflict`).
  - `SOD-03`: Submitter vs. Formal Approver conflict (`ErrSODConflict`).
  - `SOD-05`: Self-delegation prohibition (`ErrSelfDelegationForbidden`).
- **Source-Authority Verification & Delegation Bounding:**
  - **No Multi-Hop Delegation:** Re-delegating previously delegated authorities is strictly forbidden (`ErrMultiHopDelegationForbidden`).
  - **No Self-Delegation:** Delegators cannot assign delegated roles to themselves (`ErrSelfDelegationForbidden`).
  - **Duration Ceiling:** All delegations are capped at a strict maximum of 30 days (`MaxDelegationDuration`) or the delegator role's configured maximum (`ErrDelegationDurationExceeded`).
  - **Permission Containment:** Delegator must possess every permission granted to the delegatee; escalation beyond source authority is denied (`ErrExceedsSourceAuthority`).
  - **Scope Containment:** Delegated scope must be contained within or equal to delegator scope; cross-tenant or sibling-project escalation is denied (`ErrScopeExceedsSourceAuthority`).
