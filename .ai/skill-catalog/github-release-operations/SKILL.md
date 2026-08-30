---
name: github-release-operations
description: >
  Operate tags, signatures, releases, assets, attestations, and provenance after exact release evidence and independent-review gates pass.
license: Proprietary
metadata:
  oshe-version: "0.1.0"
---

# GitHub Release Operations

## Objective

Publish an exact qualified GitHub release with attributable artifacts and no implied external deployment or compliance claim.

## Required Inputs

- exact commit, tag, version, release notes, artifact digests, SBOM, provenance, signature plan, and rollback;
- passing RELEASE operation gate and independent-review PASS;
- approved signing or credential profile without exposed secret values.

## Procedure

1. Verify version, commit, tag absence or expected state, artifact digests, CI, review, and release evidence.
2. Confirm release publication does not trigger an unauthorized external deployment.
3. Evaluate the exact gate immediately before tag, signature, asset, attestation, draft, or publish action.
4. Execute only recorded actions and capture immutable object IDs, URLs, digests, and timestamps.
5. Read back the release, assets, provenance, signatures, linked issues, and workflow effects.

## Required Output

Tag and release identities, asset digests, signatures, attestations, receipts, post-state validation, limitations, and recovery status.

## Stop Conditions

- artifact, commit, version, signing identity, review, or release evidence is incomplete or inconsistent;
- publication would make an unapproved legal, safety, certification, or compliance claim;
- release action triggers non-GitHub production deployment without separate authority.

## Evaluation Cases

- accept publication of exact independently reviewed artifacts with matching digests;
- reject mutable, unsigned-when-required, stale, or unqualified artifacts.
