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

## Scoped Directory Visibility & Read Boundaries (`directory_visibility.go`)

- **Privacy & Data Minimization Governance (`H030-005` / Issue #87 / `NFR-V030-PRIV-001`):** Implements local in-memory scoped directory discovery, field minimization, role-bounded read access, and anti-enumeration controls as approved privacy prework under `H030-005`.
- **Exact-Scope Directory Partitioning:** Directory searches are strictly partitioned to the caller's authorized organizational boundary. Project-scoped callers (`RoleInspector`, `RoleProjectManager`, `RoleContractor`) are automatically partitioned to their assigned `ProjectID` and cannot discover workers or entities across project or company boundaries.
- **Anti-Enumeration Defense (`NEG-V030-04`):** When a caller queries a cross-project or unassigned context, the query returns an empty result set (`[]MinimizedDirectoryProfile{}`) with a `nil` error (HTTP 200 equivalent). Direct profile lookups across projects or tenants fail closed with non-leaking `ErrProfileNotFound`, strictly preventing project existence reconnaissance or worker enumeration.
- **Data Minimization & Sanitized Exposure (`MinimizedDirectoryProfile`):** Directory responses expose only operational attributes (`ProfileID`, `Subject`, `TenantID`, `CompanyID`, `ProjectID`, `SiteID`, `DisplayName`, `JobTitle`, `Department`, `AssignedAreas`, `Status`). Strictly excludes personal phone numbers, personal email addresses, national IDs, passwords, session hashes, bearer tokens, and administrative authority grants (`AssertDataMinimization`).
- **Role-Bounded Read Controls:**
  - **Authentication & Permission Enforcement:** Anonymous or unauthenticated callers are rejected (`ErrUnauthenticatedCaller`). Callers must hold active `iam:directory:read` permission (`ErrDirectoryReadPermissionDenied`).
  - **External Contractor Boundary:** External contractors (`RoleContractor`) are strictly bounded to their assigned project and site, and are prohibited from accessing corporate directories or cross-project lists.
  - **Inactive Profile Shielding:** Deactivated profiles (`INACTIVE`) are automatically hidden from operational directory searches; only `RoleTenantAdmin` and `RoleAuditor` with explicit inclusion flags may inspect inactive profiles for compliance purposes.
  - **Anti-Harvesting Query Bounds:** Query pagination is strictly clamped between 1 and a maximum ceiling of 100 entries (`MaxSearchLimit`) to prevent automated bulk directory scraping.
- **Local-Only Boundary Invariant:** All directory visibility mechanics operate strictly in-memory using local synthetic fixtures. Zero external identity providers, external address books, cloud directories, production data, or network routes are connected or claimed.

## Directory Boundaries & Qualification Evidence (`directory_qualification_test.go`)

- **Provisional Qualification Governance (`H030-003`, `H030-004`, `H030-005` / Issue #89 / `V030-I016`):** Establishes integrated end-to-end qualification evidence covering directory privacy, duplicate identity rejection, exact-scope partitioning, safe lifecycle transitions, and simulated migration/recovery lineage.
- **Privacy & Data Minimization Verification:** Proves exposed profiles strictly contain sanitized operational attributes (`AssertDataMinimization`). Confirms absence of sensitive personal PII (personal emails, phone numbers, national IDs) and cryptographic credentials (passwords, bearer tokens, digests).
- **Duplicate Collision & False-Merge Rejection:** Proves uniqueness of profile identifiers within tenant scopes (`ErrDuplicateIdentifierCollision`) and enforces that distinct synthetic subjects (`usr_*`) can never be merged, aliased, or consolidated (`AssertNoFalseMerge`, `AssertDistinctSubjects`, `ErrFalseMergeProhibited`). Structural identifiers remain strictly immutable.
- **Exact-Scope Discovery & Anti-Enumeration Defense (`NEG-V030-04`):** Proves project-scoped queries are partitioned strictly to the caller's assigned project boundary. Hostile or out-of-scope queries return empty results (`[]MinimizedDirectoryProfile{}`) with nil error, preventing project existence or worker enumeration.
- **Multi-Project Subject Isolation:** Confirms a worker active in multiple projects exposes only the profile relevant to the viewer's authorized project context.
- **Safe Lifecycle & Active-by-Default Discovery:** Confirms inactive profiles (`INACTIVE`) are excluded from operational discovery, reject non-structural updates (`ErrProfileInactive`), and are visible only to authorized audit roles (`RoleTenantAdmin`, `RoleAuditor`).
- **Simulated Migration & Recovery Lineage:** Demonstrates end-to-end ingestion, attribute mutation, deactivation, and reactivation. Confirms full historical state can be reconstructed from the append-only ledger (`DirectoryResolutionLedger`) with zero data loss or cross-tenant leakage.
- **Separation of Concerns & Non-Claims:** Confirms directory projections convey zero operational authorization (`AssertNoAuthorizationBypass`). Operates strictly on local in-memory synthetic fixtures; zero real directory migration, external identity provider sync, production persistence, or runtime execution is claimed or enacted.

## Synthetic External User Profiles & Sponsor Controls (`external_user_profile.go`)

- **Approved External User Categories (Issue #94):** Formalizes distinct external user classifications: `TEMPORARY_WORKER`, `SITE_LOCAL_WORKER`, `CONTRACTOR_WORKER`, `CLIENT_INSPECTOR`, `EXTERNAL_AUDITOR`, and `PARTNER_SPECIALIST`.
- **Mandatory Internal Sponsor (`usr_*`):** All external user enrollments require an authoritative internal sponsor manager (`usr_*`). External self-sponsorship and multi-hop chain delegations are strictly rejected (`ErrMissingInternalSponsor`, `ErrInvalidInternalSponsor`).
- **Company Administration Denial:** External users are categorically barred from holding internal Company, Business Unit, or Tenant administrative roles (`RoleTenantAdmin`, `RoleProjectManager`). Attempted escalation fails closed (`ErrCompanyAdminDenied`).
- **Profile Data Minimization:** Restricts profile attributes to sanitized display names and opaque synthetic contact references (`ref_synth_*`). Strictly rejects personal identifiable information (raw emails, phone numbers, national/citizen IDs) via `ErrPIIDetected`.
- **Temporal Validity & Revocation:** Enforces explicit start/end enrollment windows (`ValidFrom`, `ValidTo`). Users past `ValidTo` automatically evaluate as `EXPIRED` (`ErrEnrollmentExpired`). In-memory revocation transitions status to `REVOKED` (`ErrEnrollmentRevoked`) with full audit trail attribution.
- **Append-Only Enrollment Audit Ledger (`ExternalUserLedger`):** An in-memory, thread-safe audit log recording immutable history records (`ExternalUserAuditRecord`) for every external user enrollment and revocation event, enforcing strict tenant boundary isolation and zero hard deletion.
- **Local Synthetic Prework Boundary (H030-004, H030-005):** Confined strictly to in-memory synthetic fixtures without external IdP integration, database persistence, or final user-model binding.

## Scoped Role Assignments, Revocation & Audit Ledger (`scoped_assignment.go`)

- **Explicit Scoped Role Assignments (`ScopedAssignment`):** Binds a discrete security role (`Role`) and explicit organizational hierarchy scope (`ScopeGrant`) to a synthetic subject (`usr_*`) with start/end temporal boundaries (`ValidFrom`, `ValidTo`) and an internal approval source (`usr_*`).
- **Temporal Validity & Expiration:** Evaluates active validity at any timestamp (`IsValidAt`). Assignments past `ValidTo` automatically evaluate as `EXPIRED` (`ErrAssignmentExpired`) and fail closed during access evaluation.
- **Explicit Revocation Mechanics:** Revokes active assignments in memory via `Revoke(revokedBy, reason, at)`, transitioning status to `REVOKED` (`ErrAssignmentRevoked`) and recording immutable audit attribution.
- **Segregation-of-Duties (SOD) Conflict Detection (`CheckRoleConflict`):** Prevents conflicting active role grants on overlapping scopes (e.g. Inspector + Auditor, Project Manager + Auditor, Contractor + Admin/PM, and duplicate active roles).
- **Append-Only Historical Audit Ledger (`AssignmentLedger`):** Captures immutable records (`AssignmentAuditRecord`) for every assignment creation, revocation, and expiration. Guarantees strict tenant isolation and zero hard deletion of past authorization events.
- **End-to-End Scoped Access Evaluation (`EvaluateScopedAccess`):** Integrates scoped role assignments with the `PolicyEvaluator` to dynamically assert active membership, time-valid role grants, and exact scope coverage, failing closed on scope mismatches or unauthorized actions.
- **Provisional Local Boundary (`H030-003` / Issue #91):** Confined strictly to in-memory local fixtures without external identity provider synchronization, persistent database mutation, or final sovereign authority selection.

## Explicit Delegation Controls, Chain Limits & Emergency-Access Boundary (`delegation_control.go`)

- **Direct Delegation Chain Ceiling (`MaxDelegationChainDepth = 1`):** Restricts role delegation strictly to direct 1-hop grants. Re-delegating previously delegated roles or multi-hop chaining is forbidden and fails closed (`ErrUnauthorizedChainDepth`, `ErrMultiHopDelegationForbidden`).
- **Self-Delegation Prohibition:** A delegator cannot delegate authorities to themselves (`ErrSelfDelegationForbidden`).
- **Source Authority & Scope Containment:** A delegator cannot delegate permissions or scopes exceeding their own active entitlements (`ErrExceedsSourceAuthority`, `ErrScopeExceedsSourceAuthority`).
- **Protected Sovereign Authority Non-Delegability:** Sovereign administrative roles (`RoleTenantAdmin`) and protected permissions are categorically non-delegable (`ErrProtectedAuthorityNonDelegable`).
- **Emergency Break-Glass Prohibition:** Unapproved automated emergency escalations or break-glass access bypasses are strictly prohibited in Milestone v0.3.0 and fail closed with default-deny (`ErrEmergencyAccessDenied`, `AssertEmergencyAccessDenied`).
- **Temporal Validity & Revocation:** All delegations enforce explicit start/end windows (`ValidFrom`, `ValidTo`) capped at 30 days (`MaxDelegationDuration`) or role-specific maximums. Active delegations can be explicitly revoked with audit attribution (`Revoke`).
- **Append-Only Delegation Audit Ledger (`DelegationLedger`):** An in-memory, thread-safe audit log recording immutable history records (`DelegationAuditRecord`) for every delegation creation and revocation event, enforcing strict tenant boundary isolation.
- **Provisional Non-Binding Policy Boundary (H030-003 / Issue #92):** Operates purely on local synthetic fixtures in memory without external identity provider synchronization, database persistence, or final policy binding.

## Authorization Boundaries & Qualification Evidence (`authorization_qualification_test.go`)

- **Provisional Authorization Qualification Governance (`H030-003`, `H030-004`, `H030-005` / Issue #93 / `V030-I020`):** Provides complete deterministic qualification evidence across default-deny access evaluation, privilege escalation prevention, scoped role assignment lifecycles, delegation bounding, segregation-of-duties conflict detection, append-only historical audit ledgers, and integrated policy evaluation.
- **Default-Deny Evaluation & Cross-Tenant Boundary Isolation:** Verifies access evaluation fails closed with explicit typed denial reasons for unauthenticated callers (`DenialUnauthenticated`), mismatched tenant IDs (`DenialCrossTenant`), inactive/suspended memberships (`DenialInactiveMembership`), and ungranted roles (`DenialRoleNotGranted`). Raw delegation contexts fail closed (`DenialDelegationNotImplemented`).
- **Least-Privilege Role Bounds & Scope Containment:** Confirms role-based action barriers prevent privilege escalation (e.g. Inspector/Contractor cannot delete/update; Auditor/Viewer cannot create/update/delete). Proves cross-project and cross-site requests are rejected (`DenialScopeMismatch`), and direct-object IDOR attempts fail closed (`DenialDirectObjectMismatch`). Archived records remain strictly immutable (`DenialArchivedRecord`).
- **Scoped Role Assignment Lifecycle & Revocation:** Demonstrates temporal validity enforcement (`ValidFrom`, `ValidTo`) where expired assignments fail closed, explicit revocation (`Revoke`) transitions state to `REVOKED` with audit attribution, and double-revocation is rejected (`ErrAssignmentRevoked`). Scope depth validation rejects invalid scope levels (`ErrInvalidScopeLevel`).
- **Delegation Bounding & Emergency Bypass Denial:** Validates strict 1-hop chain depth ceiling (`ErrUnauthorizedChainDepth`, `ErrMultiHopDelegationForbidden`), self-delegation prohibition (`ErrSelfDelegationForbidden`), inverted temporal window rejection (`ErrInvalidDelegationWindow`), 30-day duration ceiling (`ErrDelegationDurationExceeded`), non-delegable protected sovereign authorities (`ErrProtectedAuthorityNonDelegable`), source authority containment (`ErrExceedsSourceAuthority`), scope containment (`ErrScopeExceedsSourceAuthority`), and emergency break-glass denial (`ErrEmergencyAccessDenied` / `AssertEmergencyAccessDenied`).
- **Segregation of Duties (SOD) Conflict Detection:** Proves conflict detection across concurrent and proposed assignments: `SOD-01` (Inspector vs. Auditor on overlapping scope), `SOD-02` (Contractor vs. TenantAdmin), `SOD-02B` (Contractor vs. ProjectManager), and duplicate active assignments on overlapping scopes (`ErrRoleConflictDetected`).
- **Append-Only Historical Audit Ledgers & Reconstruction Lineage:** Proves assignment and delegation audit ledgers (`AssignmentLedger`, `DelegationLedger`) capture immutable chronological events (`ASSIGNMENT_CREATED`, `ASSIGNMENT_REVOKED`, `DELEGATION_CREATED`, `DELEGATION_REVOKED`), strictly enforce tenant boundary isolation without cross-tenant information leakage, and preserve complete historical audit trails with zero record deletion.
- **Integrated Policy Evaluation:** Proves end-to-end policy evaluation combining active scoped assignments and role grants dynamically enforces role permissions and scope constraints.
- **Local-Only Non-Claims Invariant:** Operates exclusively in-memory on local synthetic fixtures. Zero external identity provider synchronization, zero database persistence, zero customer data, zero production network routes, and zero operational runtime authority or policy activation are claimed or enacted.

## External User Access Conditions, Expiry & Re-authentication Lifecycles (`access_condition.go`)

- **Online/Local Access Only & TrustedDevice Prohibition:** External user access conditions operate strictly in online/local mode. Claims requiring trusted-device verification or offline operation are strictly prohibited and fail closed (`ErrTrustedDeviceProhibited`). Emergency break-glass bypasses fail closed (`ErrEmergencyAccessDenied`).
- **Short Synthetic Validity Ceilings:** Initial external access conditions are capped at a maximum of 14 days (`MaxExternalAccessDuration`). Explicit renewal extensions require active sponsor approval and are capped at 7 days (`MaxRenewalExtension`).
- **Sponsor-Change Protocol:** Internal sponsorship can be transferred to a new internal manager (`ChangeSponsor`). Identical sponsor reassignment is rejected (`ErrSponsorUnchanged`). Non-user sponsors are rejected (`ErrInvalidInternalSponsor`).
- **Generation-Based Stale-Session Invalidation:** Sponsor changes, renewals, and deactivations increment the condition's `Generation` counter. Callers presenting session tokens issued under older generations are immediately denied as stale (`CategorySessionStale`, `ErrStaleSession`).
- **Immediate Local Authority Effects:** Condition suspension or deactivation (`DeactivateAccess`, `SuspendAccess`) takes immediate local effect, failing subsequent access evaluations closed (`CategoryIdentityInactive`).
- **Append-Only Condition Audit Ledger (`AccessConditionLedger`):** An in-memory, thread-safe audit log recording immutable history records (`AccessConditionAuditRecord`) for creation, sponsor changes, renewals, and deactivations with strict tenant boundary isolation and zero hard deletion.
- **Local Synthetic Prework Boundary (H030-004 / Issue #95):** Operates purely on local synthetic fixtures in memory without external device enrollment, identity provider synchronization, database persistence, or final policy binding.

## Multi-Project Participation, Bounded Contractor Administration & Preserved Attribution (`multi_project_participation.go`)

- **Provisional Multi-Project Governance (H030-003, H030-004, H030-005 / Issue #96 / `V030-I023`):** Implements in-memory models for concurrent multi-project participation by synthetic subjects (`usr_*`), bounded contractor administration, auditor read-only invariants, and permanently preserved historical actor attribution following deactivation.
- **Concurrent Multi-Project Participation (`ProjectParticipation`):** Binds a single synthetic subject to multiple projects concurrently with discrete, project-specific roles and scopes (e.g. `RoleInspector` on `prj_alpha` and `RoleContractor` on `prj_beta`). Project boundaries are strictly partitioned; participation in one project conveys zero operational authority or resource visibility in sibling projects (`ErrCrossProjectAccessDenied`).
- **Bounded Contractor Administration:** External contractor participants are categorically barred from holding internal Company, Business Unit, or Tenant administrative roles (`RoleTenantAdmin`, `RoleProjectManager`). Any attempt to assign administrative roles or execute administrative functions (e.g. project management, role assignment, inspection approval, audit export, deletion) fails closed (`ErrContractorAdminProhibited`, `AssertContractorAdminBounds`).
- **Auditor Read-Only Boundary:** Participants holding `RoleAuditor` operate strictly under read-only permissions across all assigned projects. All mutating operations (`ActionCreate`, `ActionUpdate`, `ActionDelete`, inspection creation, finding logging, record archiving) are strictly denied (`ErrAuditorReadOnlyViolation`, `AssertAuditorReadOnly`).
- **Preserved Historical Actor Attribution (`AttributionLedger`):** An in-memory, thread-safe, append-only ledger recording immutable historical records (`HistoricalAttributionRecord`) for all operational actions (inspections, findings, hazard flags, reviews).
- **Post-Deactivation History Preservation Invariant:** When a participant's project binding is deactivated (`DeactivateParticipation`), subsequent operational access fails closed immediately (`DenialInactiveMembership`), but all historical attribution entries remain permanently preserved, intact, and queryable with original subject, display name, role, and timestamps. Tampering with or overwriting historical records is rejected (`ErrAttributionImmutable`).
- **Local Synthetic Prework Boundary (H030-003, H030-004, H030-005):** Operates purely on local synthetic fixtures in memory without external identity provider synchronization, database persistence, customer data, or operational runtime policy activation.

## External User Lifecycles, Conditions, Boundaries & Qualification Evidence (`external_user_qualification_test.go`)

- **Provisional External User Qualification Governance (`H030-003`, `H030-004`, `H030-005` / Issue #97 / `V030-I024`):** Provides end-to-end deterministic qualification evidence across external user temporal validity, sponsor change protocols, access condition renewal limits, generation-based stale-session invalidation, two-project participation isolation, bounded contractor administration, auditor read-only invariants, mandatory online-only access, profile data minimization, and preserved historical actor attribution after deactivation.
- **Temporal Validity, Expiration & Revocation:** Demonstrates rejection of inverted validity windows (`ErrInvalidTimeWindow`), active-by-default access within valid windows, and automatic expiration for timestamps before `ValidFrom` or after `ValidTo`. Proves explicit sponsor revocation immediately transitions profile status to `REVOKED`, deactivates access, and records immutable audit attribution.
- **Access Condition Renewals & Sponsor Change Generation:** Proves access extensions require active internal sponsor approval and fail closed if exceeding the 7-day maximum extension ceiling (`MaxRenewalExtension`, `ErrDurationExceeded`). Confirms sponsor reassignment updates the internal sponsor, rejects self-delegation or external sponsors (`ErrInvalidInternalSponsor`), and increments the condition generation counter (`Generation`). Callers presenting tokens from older generations are immediately denied as stale (`CategorySessionStale`).
- **Two-Project Participation Isolation & Anti-Leakage:** Proves a single synthetic worker active concurrently across two projects (`prj_alpha` as `RoleInspector`, `prj_beta` as `RoleContractor`) exercises role-permitted access within each project while strictly failing closed on unassigned sibling projects (`prj_gamma`, `DenialScopeMismatch`) or foreign tenants (`DenialCrossTenant`), preventing cross-project resource leakage or authority consolidation.
- **Bounded Contractor Administration & Auditor Read-Only Boundaries:** Confirms external contractors are categorically barred from holding administrative roles (`RoleTenantAdmin`, `RoleProjectManager`) in profiles or project participations (`ErrCompanyAdminDenied`, `ErrContractorAdminProhibited`) and cannot execute administrative permissions. Verifies auditors are bounded strictly to read-only capabilities, denying all mutating actions (`ActionCreate`, `ActionUpdate`, `ActionDelete`) and mutating permissions (`ErrAuditorReadOnlyViolation`).
- **Mandatory Online-Only Access & Profile Data Minimization:** Validates that external user access conditions strictly require online/local connectivity, rejecting claims demanding trusted device enrollment or offline capabilities (`ErrTrustedDeviceProhibited`). Confirms profile creation rejects sensitive personal identifiable information (raw email addresses, phone numbers, national/citizen IDs) via `ErrPIIDetected`.
- **Post-Deactivation Historical Attribution Preservation:** Proves participant deactivation immediately terminates operational access (`DenialInactiveMembership`) while preserving all historical operational records in the append-only ledger (`AttributionLedger`). Confirms historical records remain intact, immutable, and queryable with original subject, display name, role, and timestamps; overwrite attempts fail closed (`ErrAttributionImmutable`) and cross-tenant queries return zero records.
- **Local-Only Non-Claims Invariant:** Operates exclusively in-memory on local synthetic fixtures. Zero external identity provider synchronization, zero database persistence, zero customer data, zero production network routes, and zero operational runtime authority or policy activation are claimed or enacted.
