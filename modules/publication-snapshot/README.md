# Publication Snapshot

- Module ID: `MOD-PUB`
- Roadmap topic: `V030-T06`
- Governing issues: GitHub Issue #102 (`V030-I029`), GitHub Issue #103 (`V030-I030`)
- Implementation state: candidate local synthetic schema, deterministic lifecycle controls, and qualification test fixtures

Provides standalone, synthetic immutable publication snapshot models, source entity mapping, approved-field allowlists, deny-by-default redaction, reviewer approval context, version identity, cryptographic integrity verification, controlled export packaging, and deterministic publication lifecycle state machines for OSHE Platform external publication previews and regulatory audit packages.

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

## Governance & Non-Claims Invariant

Operates exclusively in-memory on local synthetic fixtures under Sole Human Owner decisions `H030-003`, `H030-004`, and `H030-005`. Zero external public route activation, zero database persistence, zero customer data, zero production network routes, and zero operational runtime authority or policy activation are claimed or enacted.
