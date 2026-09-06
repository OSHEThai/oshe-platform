---
document_id: QLF-V040-REPORTING-001
title: V0.4.0 Controlled Reporting, Export, and Audit Reconstruction Qualification Baseline
governing_issue: 141
assignment_id: ASN-V040-I030-CONTROLLED-REPORTING-008
lease_id: LEASE-V040-I030-CONTROLLED-REPORTING-008
status: APPROVED
lifecycle: APPROVED
author: Test and Quality Lead
version: 1.0.0
date: 2026-09-06
target_milestone: v0.4.0
synthetic_scenario_id: fix_syn_controlled_reporting_qualification_v1
governing_decisions:
  - HDEC-V040-FOUNDATION-054
  - HDEC-V040-SCORING-058
  - HDEC-V040-I030-BOUNDED-REPOSITORY-DISCOVERY-001
retained_holds:
  - H040-007
  - H040-008
  - H040-009
  - H040-010
  - H040-011
---

# V0.4.0 Controlled Reporting, Export, and Audit Reconstruction Qualification Baseline

## 1. Executive Summary & Purpose

This specification establishes the authoritative qualification baseline for controlled operational reporting, metric aggregations, scoped operational views, complete-record exports, and tamper-evident audit reconstruction in Milestone `v0.4.0 OSHE Inspect Private Alpha` under Issue #141 (`V040-I030`).

Under approved Sole Human Owner decisions **HDEC-V040-FOUNDATION-054**, **HDEC-V040-SCORING-058**, and **HDEC-V040-I030-BOUNDED-REPOSITORY-DISCOVERY-001**, this qualification suite defines the synthetic verification test matrix and automated invariants governing:
1. **Derived Reporting Non-Authority (`DERIVED_OUTPUT_NON_AUTHORITY`)**: Reporting projections, metrics, and rendered exports are strictly read-only derivative artifacts. They do not alter operational truth, cannot mutate entity states, and cannot grant compliance, sign-off, or certification authority.
2. **Strict Scope Isolation**: Queries, report generation, and exports enforce tenant and project boundaries. Queries attempting to leak across unauthorized tenant or project boundaries fail closed (`ErrUnauthorizedReader`, `ErrCrossTenantRecord`).
3. **Metric Freshness Governance**: Projections calculate explicit freshness dispositions (`FRESH`, `STALE`, `NOT_FRESH`) based on declared `FreshnessBound` thresholds and source update timestamps.
4. **Extreme Boundary Robustness**: Deterministic execution handles empty inspection exports (0 records) without panic, large exports (100+ items) with complete record preservation and deterministic sorting, and long textual notes (>5,000 characters) without truncation or memory degradation.
5. **Version Pinning & Provenance**: Rendered manifests immutably capture template ID, version ID, generation timestamp, requesting actor, source data digest, and rendered output digest.
6. **Complete-Record Export Completeness**: The export engine produces full-record packages encompassing all domain records, source digests, rendered digests, and manifest records.
7. **Tamper Detection**: Cryptographic SHA-256 verification (`VerifyReportIntegrity`) asserts that any post-generation modification to rendered output immediately fails closed (`ErrReportTampered`).

All operations execute strictly on in-memory synthetic fixtures (`fix_syn_controlled_reporting_qualification_v1`). Zero customer data, external networks, or production credentials are used. Foundation holds `H040-007` through `H040-011` remain strictly on `HOLD`.

---

## 2. Core Qualification Invariants

### Invariant 1: Derived Non-Authoritative Status (`DERIVED_OUTPUT_NON_AUTHORITY`)
Every generated query result and rendered generation manifest must include the mandatory non-authority disclaimer:
> *"DERIVED_OUTPUT_NON_AUTHORITY: Reports, metrics, and analytics are derived outputs and never constitute operational authority or replace authoritative records."*

Reports cannot be used to bypass gates, override business rules, or clear findings. Operational truth resides exclusively in domain modules. Metrics lacking non-authority declaration fail registration closed (`ErrMissingNonAuthority`).

### Invariant 2: Tenant Scope Boundary Enforcement & Leak Prevention
Every reporting operation requires valid, non-blank `TenantID` and authorized reader identity. Any request attempting to access or combine records from differing tenants is immediately rejected with `ErrUnauthorizedReader` or `ErrCrossTenantRecord`. Requests containing undeclared query filters fail closed with `ErrUnsupportedFilter`.

### Invariant 3: Freshness Boundary Evaluation (`FreshnessDisposition`)
Metrics enforce temporal freshness (`FreshnessDisposition`):
- `FRESH`: When elapsed time from `SourceLastUpdatedAt` to `now` is $\le$ `FreshnessBound`.
- `STALE`: When elapsed time from `SourceLastUpdatedAt` to `now` exceeds `FreshnessBound`.
- `NOT_FRESH`: When `SourceLastUpdatedAt` is zero or unrecorded.

### Invariant 4: Extreme Boundary Resilience
- **Empty Records**: An export request with 0 records renders safely, producing a valid manifest with `RecordCount: 0` and format-appropriate empty indicators.
- **Large Records**: Exports with $>100$ items across arbitrary sections preserve all records in deterministic sort order without loss.
- **Long Text Integrity**: Free-form comments and notes exceeding standard lengths (e.g. $>5000$ characters) are preserved verbatim without truncation or corruption.

### Invariant 5: Complete Export & Tamper-Evident Manifest
The export facility produces a `GenerationManifest` containing:
1. `TemplateID` and `TemplateVersionID`.
2. `GeneratedAt` timestamp and `GeneratedBy` actor identity.
3. `RecordCount`.
4. `SourceDataDigest`: SHA-256 digest of input records.
5. `RenderedDigest`: SHA-256 digest of rendered payload.
6. `NonAuthorityNotice`: Canonical disclaimer.
7. `Limitations`: Pinned template limitations.

### Invariant 6: Tamper Detection
`VerifyReportIntegrity` recomputes the SHA-256 digest of the rendered output and compares it against the manifest digest. Any discrepancy immediately fails closed with `ErrReportTampered`.

---

## 3. Synthetic Scenario Matrix

| Scenario ID | Test Case | Target Invariant | Expected Outcome |
| :--- | :--- | :--- | :--- |
| **SYN-REP-01** | Metric Query Calculation | Formula & Calculation | `CalculatedValue` matches fixture math, `SampleCount` correct |
| **SYN-REP-02** | Scoped Filter Application | Scope & Isolation | Filters isolate matching records; unauthorized filters rejected |
| **SYN-REP-03** | Freshness Boundaries | Freshness Disposition | `FRESH` within bound; `STALE` when expired; `NOT_FRESH` on zero |
| **SYN-REP-04** | Cross-Tenant Reader Denial | Multi-Tenant Isolation | Fails closed with `ErrUnauthorizedReader` |
| **SYN-REP-05** | Empty Record Export | Boundary Robustness | Valid output, `RecordCount: 0`, no arithmetic panics |
| **SYN-REP-06** | Large Record Export (100+) | Scalability & Completeness | All 100+ records preserved in deterministic sort order |
| **SYN-REP-07** | Long Text Content (>5KB) | Data Integrity | Full text preserved without truncation, digest verified |
| **SYN-REP-08** | Template Version Pinning | Version Provenance | Pinned `TemplateID` and `VersionID` recorded in manifest |
| **SYN-REP-09** | Export Manifest Hashing | Cryptographic Verification | `SourceDataDigest` and `RenderedDigest` match SHA-256 |
| **SYN-REP-10** | Tamper Detection | Tamper Protection | Mutated rendered string rejected with `ErrReportTampered` |
| **SYN-REP-11** | Cross-Tenant Record Rejection | Data Hygiene | Export with mismatched tenant record fails closed (`ErrCrossTenantRecord`) |
| **SYN-REP-12** | Non-Authority Declaration | Legal & Gate Protection | Mandatory `DERIVED_OUTPUT_NON_AUTHORITY` notice present on all outputs |

---

## 4. Retained Foundation Holds

| Hold ID | Area | Status | Enforcement Description |
| :--- | :--- | :--- | :--- |
| **H040-007** | External Production Deployment | **HOLD** | Purely local in-memory execution; no external hosting or cloud deployments. |
| **H040-008** | Live Third-Party Integrations | **HOLD** | Zero external API calls, webhook dispatches, or third-party storage adapters. |
| **H040-009** | Commercial Licensing & Payments | **HOLD** | No commercial or payment gateway capabilities implemented. |
| **H040-010** | Automated Destructive Maintenance | **HOLD** | Reporting engine is strictly read-only; no deletion or mutation logic. |
| **H040-011** | Autonomous Human Delegation | **HOLD** | Reports do not grant authority; sign-off and closures remain human-owned. |
