# Toolchain Compatibility and Update Policy

Status: `DRAFT_SELECTED_BASELINE_PENDING_IDENTITY_VERIFICATION`

## Selected baseline

V010-I009 records the baseline selected in `HDEC-V010-I009-H010-003`: Go 1.26.5, Node.js 24.20.0, pnpm 11.24.0, Python 3.14.7, Docker Engine 29.7.2, Docker Compose 5.4.0, PostgreSQL 17.11, Meilisearch 1.51.0, Valkey 9.1.1, SeaweedFS 4.29, and NATS JetStream 2.14.5. The full backend and frontend dependency set is recorded in `toolchain.lock.yaml`.

The local host currently proves only Go 1.26.5, Node 24.20.0, and pnpm 11.24.0. This document does not claim a clean install, Windows/Linux parity, container health, package integrity, OCI digest, or runtime qualification.

## Compatibility rules

- A version change needs a successor decision or an explicit HDEC amendment.
- A package lock, OCI digest, action SHA, or acquisition checksum must be recorded before it is treated as immutable.
- `latest`, floating tags, unverified package locks, and inferred compatibility are prohibited.
- Windows and Linux evidence is separate; a successful host command on one platform does not establish the other.
- The source lease currently forbids package installation and network acquisition. Identity verification remains `PENDING_NO_NETWORK` until a separately authorized acquisition step.

## Update procedure

1. Prepare official provenance, version, checksum/digest, compatibility impact, and rollback evidence.
2. Obtain the required bounded route/lease if a network or package operation is needed.
3. Re-run the exact local checks on the immutable candidate.
4. Obtain independent review before a PR or merge. A successful check does not itself authorize merge or release.
