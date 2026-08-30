---
name: dependency-supply-chain-review
description: >
  Review dependencies, provenance, licenses, advisories, lockfiles, checksums, and build-chain risk. Use for dependency, action, image, package, or tool changes.
license: Proprietary
metadata:
  oshe-version: "0.1.0"
---

# Dependency and Supply-Chain Review

## Objective

Bound dependency risk and preserve reproducible provenance for every introduced component.

## Required Inputs

- manifests, lockfiles, image references, workflow actions, and tool versions;
- license policy and applicable advisories;
- package source, version, checksum or digest where supported.

## Procedure

1. Inventory direct and material transitive additions or updates.
2. Verify official source, exact version, license compatibility, maintenance, and advisories.
3. Require lockfile, immutable digest, checksum, or documented exception as applicable.
4. Assess install/build scripts, permissions, network behavior, and rollback.
5. Record unresolved provenance or advisory status as a blocker or risk.

## Required Output

Dependency inventory, provenance, license and advisory findings, pins, exceptions, and decision needs.

## Stop Conditions

- source or license is unknown;
- a critical unresolved advisory affects the intended use;
- an unpinned executable dependency enters a protected workflow without approval.

## Evaluation Cases

- accept a verified pinned dependency with compatible license;
- reject an ambiguous package source or mutable production image tag.
