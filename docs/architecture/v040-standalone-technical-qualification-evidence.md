---
document_id: EV-V040-TECH-001
title: v0.4.0 Standalone Technical Qualification Evidence
governing_issue: 143
assignment_id: ASN-V040-I032-TECHNICAL-QUALIFICATION-002
lease_id: LEASE-V040-I032-TECHNICAL-QUALIFICATION-002
status: APPROVED
lifecycle: APPROVED
target_milestone: v0.4.0
synthetic_scenario_id: fix_syn_technical_qualification_evidence_v1
governing_decisions:
  - HDEC-V040-FOUNDATION-054
  - HDEC-V040-SCORING-058
retained_holds:
  - H040-007
  - H040-008
  - H040-009
  - H040-010
  - H040-011
---

# v0.4.0 Standalone Technical Qualification Evidence

## 1. Executive Summary & Objective

This document records the source-grounded technical qualification evidence for Milestone `v0.4.0 OSHE Inspect Private Alpha` under Issue #143 (`V040-I032`) and Assignment `ASN-V040-I032-TECHNICAL-QUALIFICATION-002`.

Five standalone Go module test suites within the platform repository were executed deterministically in-process without network, database, or external infrastructure dependencies. Every suite executed successfully with zero failures (`PASS`), establishing algorithmic correctness, state-machine invariants, role containment, deterministic scoring arithmetic, audit trail immutability, and localization contracts.

---

## 2. Explicit Non-Claims & Boundary Declarations

To maintain strict compliance with governance directives, the following boundary declarations and non-claims are formally recorded:

- **User Evidence Remains Unproven:** Zero tests in this qualification suite involve human users, user feedback, usability assessments, or field trials. All user identities (`usr_*`) are synthetic stubs.
- **Device Evidence Remains Unproven:** Zero tests involve physical mobile devices, field sensors, or hardware interfaces.
- **Browser Evidence Remains Unproven:** Zero tests execute within a browser runtime, web engine, DOM environment, or mobile WebView.
- **Runtime Evidence Remains Unproven:** Zero tests execute against live staging or production infrastructure, live PostgreSQL databases, live NATS message brokers, live Meilisearch instances, or live Valkey caches. All execution is isolated, in-memory Go unit and qualification test execution.
- **H040-008 Remains Unproven:** Real participant, private-alpha, and UAT engagement (`H040-008`) is **unproven** and remains strictly on **`HOLD`**.
- **Synthetic Isolation Boundary:** All operational fixtures utilize exclusively synthetic tenant identifiers (`ten_*`), project identifiers (`prj_*`), user identifiers (`usr_*`), and record keys (`rec_*`). Zero customer data or production records exist in this test harness.

---

## 3. Retained Foundation Holds

Under **HDEC-V040-FOUNDATION-054**, all foundation operational holds remain in active, unlifted **`HOLD`** status. Zero authority is granted to alter, bypass, or lift any hold:

| Hold ID | Governance Subject | Status | Operational Restriction |
|---|---|---|---|
| `H040-007` | Technical release authorization | **HOLD** | No technical release authorization is granted. |
| `H040-008` | Real participant, private-alpha, and UAT engagement | **HOLD** | Real participant, private-alpha, and UAT engagement remains unproven; no engagement authorization is granted. |
| `H040-009` | Binding support and manual-fallback operational ownership | **HOLD** | No binding support or manual-fallback operational ownership authorization is granted. |
| `H040-010` | External environment, device, account, route, storage, and notification activation | **HOLD** | No external environment, device, account, route, storage, or notification activation is granted. |
| `H040-011` | Final outcome, residual-risk acceptance, and v0.5.0 entry decision | **HOLD** | No final outcome, residual-risk acceptance, or v0.5.0 entry decision authorization is granted. |

Zero authority is granted to alter, bypass, or lift any hold.

---

## 4. Five Module Suite Execution Results

The five standalone module test suites were executed sequentially under `LEASE-V040-I032-TECHNICAL-QUALIFICATION-002`. All suites passed completely:

| # | Module Path | Package Identity | Command Line | Execution Time | Result |
|---|---|---|---|---|---|
| 1 | `modules/internal-portal` | `oshe/internal-portal` | `go test ./...` | 0.589s | **PASS** |
| 2 | `modules/identity-authorization` | `github.com/oshethai/oshe-platform/modules/identity-authorization` | `go test ./...` | 0.370s | **PASS** |
| 3 | `modules/reporting-localization` | `github.com/oshethai/oshe-platform/modules/reporting-localization` | `go test ./...` | 0.361s | **PASS** |
| 4 | `modules/records-audit` | `github.com/oshethai/oshe-platform/modules/records-audit` | `go test ./...` | 0.387s | **PASS** |
| 5 | `modules/workflow-action` | `github.com/oshethai/oshe-platform/modules/workflow-action` | `go test ./...` | 0.343s | **PASS** |

**Summary:** 5 of 5 suites passed. 0 suites failed. Total execution time: ~2.05s.

---

## 5. Source-to-Test Mapping & Invariant Coverage

### 5.1 `modules/internal-portal`
- **Source Files:**
  - `portal.go`: Portal viewer validation, role-based navigation resolution, and route authorization gates.
  - `inspect_composition.go`: Composite inspection model aggregation, multi-domain summary building, and project containment.
- **Test Files:**
  - `portal_test.go`: Viewer validation (`ErrUnauthenticatedViewer`, `ErrBlankSubject`, `ErrBlankTenantID`, `ErrInvalidRole`) and role-based navigation containment across PM, Auditor, Inspector, and Contractor.
  - `inspect_composition_test.go`: Composition aggregation, inspection finding counts, and tenant project scoping.
  - `negative_controls_test.go`: Rejection of cross-tenant access, unauthorized portal actions, and malformed requests.
- **Key Invariants Qualified:**
  - Strict tenant and subject identity validation before any portal view is resolved.
  - Navigation menus strictly filtered to authorized capabilities per role (e.g. contractors cannot view audit or tenant admin routes).
  - Cross-tenant data leakage prevented at the portal presentation boundary.

### 5.2 `modules/identity-authorization`
- **Source Files:**
  - `access_policy.go`, `access_condition.go`: Core ABAC/RBAC policy engine and contextual condition evaluators.
  - `authorization_matrix.go`: Canonical role-to-permission mapping and permission inheritance.
  - `delegation_control.go`: Bounded delegation mechanics, 1-hop limit enforcement, and temporal windows.
  - `scoped_assignment.go`: Scoped role assignment lifecycle, project/site bounds, and activation rules.
  - `session_revocation.go`: Immediate and tenant-wide session revocation controls.
  - `directory_profile.go`, `directory_resolution.go`, `directory_visibility.go`: Directory scoping and profile access controls.
  - `external_user_profile.go`, `multi_project_participation.go`: External contractor multi-project boundary containment.
  - `local_identity.go`, `scan_resolution.go`: Identity verification and QR/scan credential validation.
- **Test Files:**
  - `authorization_qualification_test.go`: Comprehensive qualification scenarios: default-deny matrix, unauthenticated rejection, privilege escalation prevention, temporal validity, 1-hop delegation ceilings, non-self delegation, segregation-of-duties (SOD) conflict detection, and audit lineage reconstruction.
  - `directory_qualification_test.go`: Directory scoping and tenant visibility boundaries.
  - `external_user_qualification_test.go`: Multi-project contractor isolation and least-privilege role bounds.
  - `access_policy_test.go`, `access_condition_test.go`, `authorization_matrix_test.go`: Core unit tests.
  - `delegation_control_test.go`, `scoped_assignment_test.go`, `session_revocation_test.go`: Lifecycle unit tests.
  - `negative_controls_test.go`, `multi_project_negative_controls_test.go`: Negative security control coverage.
- **Key Invariants Qualified:**
  - Default-deny evaluation fails closed for all unauthenticated or unassigned requests.
  - Cross-tenant boundaries are strictly enforced across all policy evaluation points.
  - Delegation is bounded to a 1-hop maximum; non-delegable protected roles cannot be transferred; emergency break-glass cannot be bypassed.
  - Segregation-of-duties (SOD) conflicts prevent toxic role combinations across concurrent assignments.

### 5.3 `modules/reporting-localization`
- **Source Files:**
  - `reporting.go`: Metric definition catalog, freshness bounds, reader authorization, and metric calculation.
  - `report_renderer.go`: Structured report generation with mandatory `DERIVED_OUTPUT_NON_AUTHORITY` disclaimer and SHA-256 digest binding.
  - `localization.go`: BCP-47 locale bundle resolution (`en-US`, `th-TH`), visible fallback formatting, Buddhist Era calendar arithmetic, and metric unit localization.
  - `feature_flags.go`: Governed feature toggle engine with mandatory `DefaultOff: true` invariant and temporal/cohort rollout bounds.
- **Test Files:**
  - `reporting_qualification_test.go`: Scoped filter validation, stale metric freshness rejection, empty/large record handling, context preservation, complete export packaging, digest verification, and non-authority attestation.
  - `localization_qualification_test.go`: Exact match locale resolution, visible fallback for unpopulated Thai keys, visible `[MISSING: <key>]` indicators, Buddhist Era year formatting ($\text{BE} = \text{CE} + 543$), localized units (`°C`, `ม.`, `กก.`, `เดซิเบล`), text expansion tolerances, `DefaultOff` registration failure (`ErrMustDefaultOff`), and authorization separation.
  - `reporting_test.go`, `report_renderer_test.go`: Unit tests for metric catalog and report formatting.
  - `localization_test.go`, `feature_flags_test.go`: Unit tests for language bundles and feature evaluation.
- **Key Invariants Qualified:**
  - All analytical reports and metric evaluations enforce non-authority: `DERIVED_OUTPUT_NON_AUTHORITY`.
  - Missing translations fall back visibly to `en-US` or output `[MISSING: <key>]`; silent substitution is prohibited.
  - Date/time formatting for `th-TH` in `Asia/Bangkok` deterministically computes Buddhist Era years without epoch distortion.
  - Governed feature flags default closed; `DefaultOff: false` registration fails closed with `ErrMustDefaultOff`; flags cannot bypass authorization controls.

### 5.4 `modules/records-audit`
- **Source Files:**
  - `record_audit.go`: Append-only record store, audit entry declaration, and tenant-scoped query engine.
  - `immutable_objects.go`: Immutability validation, SHA-256 content digest computation, correlation IDs, and causation tracking.
  - `retention_export.go`: Audit archive creation, retention schedule evaluation, and cryptographic export packaging.
- **Test Files:**
  - `record_audit_test.go`: Record declaration lifecycle, tenant-isolated queries, causation/correlation ID binding, and invalid state rejection.
  - `immutable_objects_test.go`: Immutability guarantees, tamper detection, digest verification, and prevention of in-place modifications.
  - `retention_export_test.go`: Audit archive generation, checksum validation, and retention lifecycle compliance.
- **Key Invariants Qualified:**
  - Records once declared cannot be updated or deleted; modifications require new immutable version records.
  - Audit trail entries maintain unbroken cryptographic digest linkages, correlation IDs, and causation IDs.
  - Cross-tenant record queries are strictly rejected.

### 5.5 `modules/workflow-action`
- **Source Files:**
  - `scoring.go`: Deterministic scoring engine, rule matrices, basis points conversion, and `R1_ROUND_HALF_UP` arithmetic.
  - `fail_closed_governance.go`: State transition governor, transition catalog, and fail-closed safety intercepts.
  - `action_lifecycle.go`, `action_governance.go`: CAPA action creation, status transitions, and role authorization.
  - `finding_lifecycle.go`: Inspection finding classification, severity scoring, and verification states.
  - `reinspection_lifecycle.go`: Reinspection scheduling, boundary conditions, and signoff workflows.
  - `business_rules.go`, `workflow.go`: Workflow validation, prerequisite checking, and outcome evaluation.
- **Test Files:**
  - `scoring_qualification_test.go`: Exact basis points rounding at the 80.00% (8000 bps) threshold, handling of missing/UNKNOWN items, excluded/NA item filtering, rule version pinning, critical item failure override (immediate non-compliance), and AI score recommendation denial (AI advice cannot override deterministic scoring).
  - `fail_closed_governance_test.go`: Fail-closed evaluation, invalid transition rejection, and security state locks.
  - `action_lifecycle_test.go`, `action_governance_test.go`: Action lifecycle state transitions and access controls.
  - `finding_lifecycle_test.go`, `reinspection_lifecycle_test.go`: Finding resolution and reinspection workflows.
  - `scoring_test.go`, `workflow_test.go`, `business_rules_test.go`: Core unit tests.
- **Key Invariants Qualified:**
  - Inspection scores compute deterministically with exact basis point arithmetic and explicit `R1_ROUND_HALF_UP` rounding.
  - A single critical failure immediately forces a non-compliant outcome regardless of total earned points.
  - AI score suggestions and advisory outputs cannot override or bypass deterministic scoring rules.
  - State machine transitions fail closed when prerequisite conditions or authorization credentials are unmet.

---

## 6. Traceability Matrix

| Requirement / Invariant Domain | Governing Module | Test File Verification | Status |
|---|---|---|---|
| Role-based Navigation & Portal Containment | `modules/internal-portal` | `portal_test.go`, `negative_controls_test.go` | **QUALIFIED** |
| Multi-Domain Inspection Composition | `modules/internal-portal` | `inspect_composition_test.go` | **QUALIFIED** |
| Default-Deny Authorization & Tenant Isolation | `modules/identity-authorization` | `authorization_qualification_test.go` | **QUALIFIED** |
| Bounded 1-Hop Delegation & Temporal Limits | `modules/identity-authorization` | `authorization_qualification_test.go` | **QUALIFIED** |
| Segregation-of-duties (SOD) Conflict Detection | `modules/identity-authorization` | `authorization_qualification_test.go` | **QUALIFIED** |
| External Contractor Scoping & Visibility | `modules/identity-authorization` | `external_user_qualification_test.go` | **QUALIFIED** |
| Deterministic Reporting & Freshness Bounds | `modules/reporting-localization` | `reporting_qualification_test.go` | **QUALIFIED** |
| Derived Output Non-Authority Disclaimer | `modules/reporting-localization` | `reporting_qualification_test.go` | **QUALIFIED** |
| BCP-47 Localization & Visible Fallback | `modules/reporting-localization` | `localization_qualification_test.go` | **QUALIFIED** |
| Asia/Bangkok Buddhist Era Time-Zone Formatting | `modules/reporting-localization` | `localization_qualification_test.go` | **QUALIFIED** |
| Default-Off Feature Flags & Auth Separation | `modules/reporting-localization` | `localization_qualification_test.go` | **QUALIFIED** |
| Append-Only Audit Records & Digest Integrity | `modules/records-audit` | `record_audit_test.go`, `immutable_objects_test.go` | **QUALIFIED** |
| Audit Lineage Reconstruction & Export Retention | `modules/records-audit` | `retention_export_test.go` | **QUALIFIED** |
| Deterministic Scoring & 8000 bps Boundary Rounding | `modules/workflow-action` | `scoring_qualification_test.go` | **QUALIFIED** |
| Critical Item Override & AI Suggestion Denial | `modules/workflow-action` | `scoring_qualification_test.go` | **QUALIFIED** |
| Fail-Closed Workflow State Governance | `modules/workflow-action` | `fail_closed_governance_test.go` | **QUALIFIED** |
