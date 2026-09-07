# Release Tests

This area contains release-gate, artifact-integrity, provenance, compatibility, migration, rollback, and evidence-binding tests. Skipped, unknown, quarantined, or inconclusive checks are not passing evidence.

## Release Evidence Bundle Verification

- `tests/test_v030_release_evidence_bundle.py`: Deterministic completeness, traceability, artifact-integrity, and non-claims verification for Milestone v0.3.0 (`REL-V030-EVD-001` / Issue #110).
- `tests/test_v030_release_decision_packet.py`: Deterministic verification of the v0.3.0 Sole Human Owner decision packet, 4-option tradeoff analysis, prerequisite checklist, residual risk assessment, and unfilled decision template (`ARC-V030-DECPKT-001` / Issue #111).
- `tests/test_v030_release_decision_record.py`: Deterministic verification of the approved v0.3.0 Sole Human Owner release decision record, Option 1 authorizations, retained H030-007 hold, and governance boundaries (`ARC-V030-DECREC-001` / `HDEC-V030-RELEASE-053` / Issue #111).
- `tests/test_v040_foundation_decision.py`: Deterministic verification of the approved v0.4.0 OSHE Inspect Private Alpha foundation decisions, approved gates H040-001 through H040-006, retained holds H040-007 through H040-011, and non-production operational boundaries (`ARC-V040-DECREC-001` / `HDEC-V040-FOUNDATION-054` / Issue #112).
- `tests/test_v050_foundation_decision.py`: Deterministic verification of the v0.5.0 Workforce and Incident Alpha foundation boundary, approved gates H050-001 through H050-008, retained holds H050-009 through H050-013, and synthetic-only non-operational constraints (`ARC-V050-DECREC-001` / `HDEC-V050-WAVE0-EXECUTION-ACTIVATION-001` / `V050-E001`).
- `tests/test_v050_domain_data_authority_baseline.py`: Deterministic verification of the v0.5.0 synthetic data-authority baseline, ownership isolation, protected-state boundaries, retained holds, and unselected configuration values (`ARC-V050-DOMAIN-001` / `V050-E002`).
