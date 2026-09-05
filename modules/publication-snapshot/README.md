# Publication Snapshot

- Module ID: `MOD-PUB`
- Roadmap topic: `V030-T06`
- Governing issue: GitHub Issue #102 (`V030-I029`)
- Implementation state: candidate local synthetic schema and qualification test fixtures

Provides standalone, synthetic immutable publication snapshot models, source entity mapping, approved-field allowlists, deny-by-default redaction, reviewer approval context, version identity, cryptographic integrity verification, and controlled export packaging for OSHE Platform external publication previews and regulatory audit packages.

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

## Negative Controls Catalog

| Control ID | Threat / Scenario | Hostile Input | Expected Failure |
| :--- | :--- | :--- | :--- |
| **`NEG-SNAP-01`** | Unapproved internal or private fields in publication payloads | Unallowlisted fields in strict mode or prohibited keywords in allowlist definition | Fails with `ErrUnapprovedFieldDetected` or `ErrProhibitedFieldInAllowlist` |
| **`NEG-SNAP-02`** | Leakage of credentials, bearer tokens, or sensitive identity numbers | Input payload containing `admin_password`, `bearer_token`, `national_id`, or `oshe_tok_` prefix | Fails with `ErrProhibitedFieldDetected` |
| **`NEG-SNAP-03`** | Cross-tenant source references or missing source provenance | `SourceTenantID != SnapshotTenantID`, blank source ID, or blank digest | Fails with `ErrSourceMismatch` or validation error |
| **`NEG-SNAP-04`** | Undetected payload tampering or digest desynchronization | In-memory payload modification after digest calculation | `VerifyIntegrity()` fails with `ErrIntegrityVerificationFailed` |
| **`NEG-SNAP-05`** | In-place mutation of published snapshots or unapproved publishing | Re-publishing published snapshot, versioning unpublished draft, or unapproved reviewer status | Fails with `ErrSnapshotAlreadyPublished`, `ErrSnapshotImmutable`, or `ErrSnapshotNotApproved` |
| **`NEG-SNAP-06`** | Exporting unvetted draft snapshots or cross-tenant leakage | Invalid classification, unapproved destination scope, or cross-tenant snapshot in export package | Fails with `ErrInvalidClassification`, `ErrUnapprovedDestinationScope`, or `ErrCrossTenantAccessDenied` |

## Governance & Non-Claims Invariant

Operates exclusively in-memory on local synthetic fixtures under Sole Human Owner decisions `H030-003`, `H030-004`, and `H030-005`. Zero external public route activation, zero database persistence, zero customer data, zero production network routes, and zero operational runtime authority or policy activation are claimed or enacted.
