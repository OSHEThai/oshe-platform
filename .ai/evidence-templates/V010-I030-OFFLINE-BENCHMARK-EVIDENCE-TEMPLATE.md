# V010-I030 Offline Benchmark Evidence Template

## Scope

Candidate 2 — Standalone JSON Validation CLI in `scratch/json_validator/`, evaluated only with synthetic fixtures and deterministic local controls.

## Required evidence

- Base commit SHA and fixture corpus SHA-256.
- Exact commands and exit codes for `python -m unittest tests.test_reference_benchmark -v` and `python tools/reference_benchmark.py --verify --fixture-file tests/fixtures/reference_benchmark/synthetic_cases.json`.
- Scorecard SHA-256, fixture count, completed/failed/total measures, and each failure-injection result.
- Proof that `provider_routes_enabled` is `0`, `provider_mode` is `SYNTHETIC_OFFLINE`, and `route_selection` is `NONE`.
- Any failure, drift, overwrite-rejection, or inconclusive result recorded without upgrade to PASS.

## Credit boundary

This template supports offline synthetic prework only. It does not authorize or evidence provider/model selection, live provider use, route activation, Candidate 2 CLI implementation, live qualification, mission execution, Issue closure, release, or deployment.
