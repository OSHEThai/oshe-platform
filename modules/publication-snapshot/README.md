# Publication Snapshot

- Module ID: `MOD-PUB`
- Roadmap topic: `V030-T06`
- Governing issues: GitHub Issue #102 (`V030-I029`), GitHub Issue #103 (`V030-I030`), GitHub Issue #104 (`V030-I031`)
- Implementation state: candidate local synthetic schema, deterministic lifecycle controls, immutable storage, and qualification test fixtures

Provides standalone, synthetic immutable publication snapshot models, source entity mapping, approved-field allowlists, deny-by-default redaction, reviewer approval context, version identity, cryptographic integrity verification, controlled export packaging, deterministic publication lifecycle state machines, sealed published-version storage, source-change isolation, and tamper-evident audit reconstruction for OSHE Platform external publication previews and regulatory audit packages.

## Architectural Components

### 1. Synthetic Immutable Publication Snapshots (`publication_snapshot.go`)

- **Permanent Immutability:** Once transitioned to `PUBLISHED`, a snapshot's payload, source mapping, and integrity digests become permanently immutable. In-place modification of published records is strictly prohibited (`ErrSnapshotImmutable`).
- **Version Lifecycle:** Updates are enacted exclusively by generating a new version with an incremented version number (`CreateNewVersion`), which automatically supersedes the preceding version (`StatusSuperseded`).
- **Controlled Withdrawal:** Snapshots can be retracted from publication via formal withdrawal (`WithdrawSnapshot`), transitioning state to `StatusWithdrawn` and severing active discovery links.

### 2. Source Entity Mapping & Provenance (`SourceEntityRef`)

- **Internal Provenance Anchoring:** Every snapshot explicitly references its underlying source entity (`SourceType`, `SourceID`, `SourceTenantID`, `SourceVersion`).
- **Source Integrity Check:** Captures the source content's SHA-256 digest (`SourceContentDigest`) to detect out-of-band drift or tampering.
- **Tenant Scope Enforcement:** Strict tenant match is enforced between the source entity and the snapshot envelope; cross-tenant source references fail closed (`ErrSourceMismatch`).

### 3. Approved-Field Allowlist & Deny-by-Default Redaction (`RedactionEngine`)

- **Deny-by-Default Transformation:** Payloads are sanitized against an explicit `PublicationFieldAllowlist`. Unapproved fields are systematically stripped from published outputs.
- **Strict Mode Enforcement:** In strict validation mode, the presence of unapproved fields causes immediate rejection (`ErrUnapprovedFieldDetected`).
- **Prohibited Sensitive Data Purge:** Prohibited keywords (passwords, session tokens, bearer strings, national citizen IDs, SSNs, private keys) are rejected at allowlist definition (`ErrProhibitedFieldInAllowlist`) and trigger immediate failure if detected in input payloads (`ErrProhibitedFieldDetected`).

### 4. Reviewer Context & Decision Attribution (`ReviewerContext`)

- **Human Attribution Context:** Publication requires explicit reviewer attribution (`ReviewerSubject`, `ReviewerRole`, `ReviewedAt`, `ReviewNotes`).
- **Formal Approval Gate:** Snapshots cannot be published without formal reviewer approval (`ReviewApproved`), failing closed with `ErrSnapshotNotApproved` if pending or incomplete.
- **Deterministic Decision Digest:** Generates a cryptographic SHA-256 hash over the reviewer identity, role, timestamp, and notes (`ReviewDecisionHash`) bound into the snapshot envelope.

### 5. Cryptographic Integrity Verification (`IntegrityMetadata`)

- **Canonical Payload Digest:** Computes a deterministic SHA-256 hash over canonical, key-sorted JSON (`PayloadDigest`).
- **Composite Envelope Signature:** Computes a signature digest over tenant ID, snapshot ID, version, source reference, payload digest, and reviewer decision hash (`SignatureDigest`).
- **Self-Verifying Seal:** Snapshots provide `VerifyIntegrity()` asserting that current payload contents exactly match sealed digests.

### 6. Controlled Export Packaging (`ExportPackage`)

- **Validated Export Artifacts:** Bundles published snapshots into formalized export structures (`NewExportPackage`) with classification tagging (`PUBLIC_SANITIZED`, `EXTERNAL_CONTROLLED`).
- **Destination Scope Validation:** Restricts exports to recognized scopes (`PUBLIC_PORTAL_PREVIEW`, `EXTERNAL_AUDITOR_PACKAGE`, `REGULATORY_SUBMISSION`), rejecting unapproved destinations (`ErrUnapprovedDestinationScope`).
- **Unpublished Record Exclusion:** Ensures only active `PUBLISHED` snapshots can be exported, rejecting draft or superseded records (`ErrUnpublishedSnapshotInExport`).
- **Cross-Tenant Isolation:** Enforces tenant consistency across all bundled snapshots (`ErrCrossTenantAccessDenied`).

### 7. Deterministic Publication Lifecycle Controls (`publication_lifecycle.go`)

- **Explicit State Machine:** Formalizes states `DRAFT` -> `UNDER_REVIEW` -> `APPROVED` -> `PUBLISHED` -> `EXPIRED`, `WITHDRAWN`, `REPLACED`, `SUPERSEDED`. Arbitrary state leaps and transitions from terminal states are rejected (`ErrIllegalStateTransition`, `ErrDuplicateTransition`).
- **Authorized Review & Approval Evidence (`ApprovalEvidence`):** Publication requires explicit approval by an authorized role (`AUDITOR` or `TENANT_ADMIN`). Approvals include cryptographic decision hashes, content digest verification (`ErrApprovalDigestMismatch`), and freshness bounds. Stale approvals exceeding maximum validity (default 7 days) are rejected (`ErrStaleApproval`).
- **Effective & Expiry Window Enforcement (`EffectiveWindow`):** Defines temporal validity windows (`EffectiveFrom`, `ExpiresAt`). Inverted dates, zero-duration, or windows exceeding 365 days are rejected (`ErrInvalidPublicationWindow`). Snapshots before `EffectiveFrom` return `ErrNotYetEffective`. Snapshots past `ExpiresAt` automatically evaluate to `EXPIRED` and fail closed (`ErrSnapshotExpired`).
- **Withdrawal Attribution:** Snapshots can be retracted with mandatory justification (`Withdraw`), permanently recording the authorized withdrawer, timestamp, and justification (`ErrMissingWithdrawalReason`). Withdrawn snapshots reject reactivation (`ErrCannotReactivateWithdrawn`).
- **Replacement & Supersession:** Replaces published snapshots with linked successor IDs (`Replace`), recording justification and predecessor linkage. Supersedes prior versions upon publishing a new iteration (`Supersede`).
- **Append-Only Historical Audit Ledger (`LifecycleAuditLedger`):** An in-memory, append-only ledger capturing immutable chronological records (`LifecycleAuditRecord`) with tamper-evident SHA-256 audit digests for every state change. Strictly isolates tenant boundaries (`GetHistory`) and guarantees zero hard deletion.

### 8. Immutable Published-Version Storage, Source Isolation & Audit Reconstruction (`immutable_publication.go`)

- **Sealed Version Store (`ImmutablePublicationStore`):** Sealed in-memory store indexing published versions by `(TenantID, SnapshotID, Version)`. Once stored, records cannot be modified or overwritten (`ErrPublicationVersionImmutable`). Defensive deep-copying on all reads and writes prevents caller mutations.
- **Source-Change Isolation:** Published snapshots capture an immutable snapshot of the operational source at publication time (`SourceEntityRef`). Operational updates to the underlying entity do not mutate the sealed snapshot. Source drift is detected via `CheckSourceDrift`, and direct mutation attempts are rejected (`ErrDirectSourceMutationForbidden`).
- **Replacement & Successor Lineage:** Successor versions explicitly link predecessor versions (`PredecessorVersion`, `PredecessorDigest`) and update predecessor successor pointers (`SuccessorVersion`, `SuccessorDigest`, `ReplacementType`, `ReplacementReason`).
- **Tamper-Evident Audit Reconstruction:** `ReconstructPublicationAuditTrail` verifies full cryptographic continuity across version chains (validating canonical payload digests, signature digests, predecessor-successor digest chaining, and monotonic version numbering). Returns `VERIFIED_INTACT` with unbroken chain digests or detects tamper (`StatusTamperDetected`).

## Negative Controls Catalog

| Control ID | Threat / Scenario | Hostile Input | Expected Failure |
| :--- | :--- | :--- | :--- |
| **`NEG-SNAP-01`** | Unapproved internal or private fields in publication payloads | Unallowlisted fields in strict mode or prohibited keywords in allowlist definition | Fails with `ErrUnapprovedFieldDetected` or `ErrProhibitedFieldInAllowlist` |
| **`NEG-SNAP-02`** | Leakage of credentials, bearer tokens, or sensitive identity numbers | Input payload containing `admin_password`, `bearer_token`, `national_id`, or `oshe_tok_` prefix | Fails with `ErrProhibitedFieldDetected` |
| **`NEG-SNAP-03`** | Cross-tenant source references or missing source provenance | `SourceTenantID != SnapshotTenantID`, blank source ID, or blank digest | Fails with `ErrSourceMismatch` or validation error |
| **`NEG-SNAP-04`** | Undetected payload tampering or digest desynchronization | In-memory payload modification after digest calculation | `VerifyIntegrity()` fails with `ErrIntegrityVerificationFailed` |
| **`NEG-SNAP-05`** | In-place mutation of published snapshots or unapproved publishing | Re-publishing published snapshot, versioning unpublished draft, or unapproved reviewer status | Fails with `ErrSnapshotAlreadyPublished`, `ErrSnapshotImmutable`, or `ErrSnapshotNotApproved` |
| **`NEG-SNAP-06`** | Exporting unvetted draft snapshots or cross-tenant leakage | Invalid classification, unapproved destination scope, or cross-tenant snapshot in export package | Fails with `ErrInvalidClassification`, `ErrUnapprovedDestinationScope`, or `ErrCrossTenantAccessDenied` |
| **`NEG-LIFE-01`** | Unauthorized approval or publication attempts | Non-auditor/non-admin roles approving/publishing or publishing unapproved drafts | Fails with `ErrUnauthorizedReviewer`, `ErrUnauthorizedPublish`, or `ErrIllegalStateTransition` |
| **`NEG-LIFE-02`** | Stale approvals or content digest drift prior to publication | Approval older than 7 days, content modified after approval, or approval expiring before publish | Fails with `ErrStaleApproval` or `ErrApprovalDigestMismatch` |
| **`NEG-LIFE-03`** | Inverted, excessively long, or past publication windows | `ExpiresAt <= EffectiveFrom`, window > 365 days, or publishing into expired window | Fails with `ErrInvalidPublicationWindow` or `ErrSnapshotExpired` |
| **`NEG-LIFE-04`** | Unattributed withdrawal, unauthorized withdrawer, or invalid replacement | Blank reason, unauthorized withdrawer role, duplicate withdrawal, or replacing withdrawn record | Fails with `ErrMissingWithdrawalReason`, `ErrUnauthorizedReviewer`, or `ErrDuplicateTransition` |
| **`NEG-LIFE-05`** | Duplicate transitions or invalid state leaps | Re-submitting under-review draft or re-publishing published snapshot | Fails with `ErrDuplicateTransition` or `ErrIllegalStateTransition` |
| **`NEG-LIFE-06`** | False claims of live external route activation or persistence | Testing in-memory synthetic fixture assertions against non-live invariants | Strictly operates in-memory on local fixtures; zero live external routes or database persistence |
| **`NEG-IMM-01`** | Overwriting existing sealed published versions | Storing duplicate version with altered payload or signature | Fails with `ErrPublicationVersionImmutable` |
| **`NEG-IMM-02`** | Creating successor versions with invalid predecessor links | Non-contiguous version, wrong predecessor version, or mismatched digest | Fails with `ErrInvalidPredecessor` or `ErrBrokenLineageChain` |
| **`NEG-IMM-03`** | Undetected payload tampering or signature corruption | In-memory corruption of stored record verified via audit reconstruction | `ReconstructPublicationAuditTrail` reports `StatusTamperDetected` |
| **`NEG-IMM-04`** | Direct mutation of published snapshots from operational source | Attempting direct write into sealed published record upon source change | Fails with `ErrDirectSourceMutationForbidden` |
| **`NEG-IMM-05`** | Blank replacement justification, missing successor, or non-existent snapshot | Registering replacement with blank reason, blank successor ID, or invalid version | Fails with `ErrBlankReplacementReason`, `ErrBlankSuccessorID`, or `ErrSnapshotNotFound` |
| **`NEG-IMM-06`** | False claims of live external publication routes or persistence | Asserting non-live synthetic invariants for immutable store | Strictly operates in-memory on local fixtures; zero live routes or persistent DB |

## Publication Snapshot Boundaries & Qualification Evidence (`publication_qualification_test.go`)

- **Provisional Qualification Governance (`H030-003`, `H030-004`, `H030-005` / Issue #105 / `V030-I032`):** Establishes integrated end-to-end qualification evidence covering deny-by-default redaction, sensitive keyword rejection, reviewer decision authorization gates, source-change drift and mutation isolation, temporal window validity and expiration, withdrawal attribution, supersession and lineage chaining, cryptographic integrity, tamper detection, and export scope boundaries.
- **Redaction & Prohibited Data Minimization:** Validates allowlist enforcement stripping unapproved fields in permissive mode and rejecting unapproved fields in strict mode (`ErrUnapprovedFieldDetected`). Prohibits sensitive field names in allowlist definitions (`ErrProhibitedFieldInAllowlist`) and immediately detects and rejects credentials, bearer tokens, and government IDs (`ErrProhibitedFieldDetected`).
- **Unauthorized Publication & Reviewer Decision Gate:** Proves draft snapshots cannot be published without formal approval (`ErrIllegalStateTransition`), unauthorized reviewer roles (`INSPECTOR`, `CONTRACTOR`, `VIEWER`, `PROJECT_MANAGER`) cannot approve (`ErrUnauthorizedReviewer`), stale approvals exceeding 7 days fail closed (`ErrStaleApproval`), and content digest mismatches are rejected (`ErrApprovalDigestMismatch`).
- **Source-Change Drift & Mutation Isolation:** Proves source entity mapping enforces tenant match (`ErrSourceMismatch`). Defensive copying ensures caller payload modifications do not mutate stored snapshots. Source drift detection (`CheckSourceDrift`) accurately identifies operational source evolution, and direct mutation attempts against sealed records fail closed (`ErrDirectSourceMutationForbidden`).
- **Temporal Validity & Automatic Expiration:** Proves inverted windows and windows exceeding 365 days are rejected (`ErrInvalidPublicationWindow`). Publishing into expired windows fails closed (`ErrSnapshotExpired`). Active access checks fail before `EffectiveFrom` and automatically transition to `EXPIRED` past `ExpiresAt`.
- **Withdrawal & Inactivation Attribution:** Validates that formal withdrawal requires non-blank justification (`ErrMissingWithdrawalReason`) and authorized roles (`ErrUnauthorizedReviewer`). Withdrawn snapshots transition to terminal `WITHDRAWN` state, reject reactivation (`ErrCannotReactivateWithdrawn`), and record permanent audit attribution.
- **Supersession & Lineage Chaining:** Proves monotonic versioning, predecessor digest chaining (`PredecessorVersion`, `PredecessorDigest`), and successor pointers. Rejects invalid predecessor references (`ErrInvalidPredecessor`), broken lineage chains (`ErrBrokenLineageChain`), and blank replacement metadata (`ErrBlankReplacementReason`, `ErrBlankSuccessorID`).
- **Cryptographic Integrity & Tamper Detection:** Confirms deterministic canonical payload hashing across key iterations (`ComputeCanonicalPayloadDigest`), composite envelope signatures (`ComputeSignatureDigest`), and self-verifying seals (`VerifyIntegrity`). Audit reconstruction across multi-version publication chains validates unbroken cryptographic continuity (`StatusVerifiedIntact`) and detects tampered records (`StatusTamperDetected`).
- **Export Denial & Cross-Tenant Scope Boundaries:** Restricts export packages to approved destination scopes (`ErrUnapprovedDestinationScope`), valid classifications (`ErrInvalidClassification`), published-only snapshots (`ErrUnpublishedSnapshotInExport`), and single-tenant consistency (`ErrCrossTenantAccessDenied`). Cross-tenant lookups and audit queries on the immutable store fail closed with non-leaking `ErrSnapshotNotFound`.
- **Local-Only Non-Claims Invariant:** Operates exclusively in-memory on local synthetic fixtures. Zero external public route activation, zero database persistence, zero customer data, zero production network routes, and zero operational runtime authority or policy activation are claimed or enacted.

## Governance & Non-Claims Invariant

Operates exclusively in-memory on local synthetic fixtures under Sole Human Owner decisions `H030-003`, `H030-004`, and `H030-005`. Zero external public route activation, zero database persistence, zero customer data, zero production network routes, and zero operational runtime authority or policy activation are claimed or enacted.
