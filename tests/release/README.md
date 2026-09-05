# Release Tests

This area contains release-gate, artifact-integrity, provenance, compatibility, migration, rollback, and evidence-binding tests. Skipped, unknown, quarantined, or inconclusive checks are not passing evidence.

## Release Evidence Bundle Verification

- `tests/test_v030_release_evidence_bundle.py`: Deterministic completeness, traceability, artifact-integrity, and non-claims verification for Milestone v0.3.0 (`REL-V030-EVD-001` / Issue #110).
- `tests/test_v030_release_decision_packet.py`: Deterministic verification of the v0.3.0 Sole Human Owner decision packet, 4-option tradeoff analysis, prerequisite checklist, residual risk assessment, and unfilled decision template (`ARC-V030-DECPKT-001` / Issue #111).
- `tests/test_v030_release_decision_record.py`: Deterministic verification of the Sole Human Owner release decision record, Option 1 approval status, granted tagging/release/planning authorities, and retained H030-007 HOLD (`ARC-V030-DECREC-001` / `HDEC-V030-RELEASE-053`).
