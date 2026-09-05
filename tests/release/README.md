# Release Tests

This area contains release-gate, artifact-integrity, provenance, compatibility, migration, rollback, and evidence-binding tests. Skipped, unknown, quarantined, or inconclusive checks are not passing evidence.

## Milestone v0.3.0 Release Verification

- `tests/test_v030_release_evidence_bundle.py`: Deterministic completeness test verifying requirements/test/review lineage, defect/predecessor dispositions, fail-closed negative controls, revocation/withdrawal invariants, and deferred human gate preservation (`H030-007` and `H030-008` HOLD).
- Governed by: [v0.3.0 Deterministic Release Evidence Bundle](docs/architecture/v030-release-evidence-bundle.md) (`ARC-V030-RELEVD-001` / `V030-I037`).
