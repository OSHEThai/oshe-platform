---
document_id: ARC-V050-DOMAIN-001
title: v0.5.0 Workforce and Incident Domain, Data Authority, and Protected-State Baseline
document_type: architecture_specification
document_version: 1.0.0
lifecycle_status: DRAFT
status: APPROVED_FOR_SYNTHETIC_FOUNDATION_ONLY
date: "2026-09-07"
author_role: Engineering Agent
governing_predecessor: "V050-I002 / GitHub Issue #153 (closed planning record)"
governing_decision: HDEC-V050-E002-EXECUTION-ACTIVATION-008
milestone: "v0.5.0 - Workforce and Incident Alpha"
approved_foundation_gates:
  - H050-001
  - H050-002
  - H050-003
  - H050-004
  - H050-005
  - H050-006
  - H050-007
  - H050-008
retained_holds:
  - H050-009
  - H050-010
  - H050-011
  - H050-012
  - H050-013
retained_unselected_configuration:
  - V050-CFG-003
  - V050-CFG-004
  - V050-CFG-005
  - V050-CFG-006
  - V050-CFG-007
  - V050-CFG-008
credit_boundary: ARCHITECTURE_BASELINE_ONLY_NO_MODULE_CONTRACT_MIGRATION_RUNTIME_OR_RELEASE_CREDIT
---

# v0.5.0 Workforce and Incident Domain, Data Authority, and Protected-State Baseline

## Purpose and non-claims

This source-governed baseline materializes planning `V050-I002` as execution
successor `V050-E002`. It is a synthetic, testable architecture record only.
It does not create an application module, public contract, database schema,
migration, user account, external route, or operational Workforce or Incident
capability.

No real workforce, employment, reporter, witness, medical, credential,
incident, evidence, or customer data is permitted. Identifiers in examples
must be synthetic and non-attributable. Binding values for `V050-CFG-003`
through `V050-CFG-008` are intentionally unselected.

## Authority and ownership boundary

The future **Workforce** domain is the sole authority for synthetic person
profiles, trusted-identity references, employment or contractor relationships,
project/site assignments, sponsorship, appointments, validity windows,
transfers, overlaps, lifecycle, and historical context. A person record links
to a trusted identity *by reference*; it must not duplicate an authentication
identity.

The future **Incident** domain is the sole authority for synthetic incident
intake, classification proposals, investigation relationships, evidence-link
references, CAPA-link references, lessons, and alerts. Incident may consume a
controlled Workforce reference but may not write Workforce state. Workforce may
not write Incident state.

Shared services retain authority for identity and authorization, files,
append-only records/audit, events, reporting, export, and recovery. OSHE
Inspect remains the authority for its own records. Cross-domain direct writes
are forbidden; future integrations must use a versioned public contract only
after a separately leased contract change.

## Default-deny and tenant isolation

Every read, create, update, transition, and reference resolution is
**default-deny**. The actor, tenant scope, ownership scope, active authority,
and current-state precondition must be validated before a proposal is accepted.
An absent, expired, ambiguous, cross-tenant, or cross-domain grant is denied.

Protected state has no last-write-wins behavior. A stale version or conflicting
proposal must be rejected or quarantined for authorized human reconciliation;
timestamps do not confer authority. The record must preserve an append-only
historical context with actor reference, reason, prior state, proposed state,
and synthetic correlation reference.

## Proposed synthetic lifecycle vocabulary

This vocabulary constrains future design; it does not activate a workflow:

| Area | Ordinary states | Protected human-only transition |
|---|---|---|
| Workforce relationship | `DRAFT`, `PROPOSED`, `ACTIVE`, `SUSPENDED`, `EXPIRED`, `SUPERSEDED` | approval, revocation, transfer conflict reconciliation, historical correction |
| Assignment validity | `DRAFT`, `PROPOSED`, `ACTIVE`, `EXPIRED`, `REVOKED` | overlap decision, revocation, exception approval |
| Incident intake | `UNCLASSIFIED`, `TRIAGE_PROPOSED`, `ROUTED`, `UNDER_REVIEW`, `SUPERSEDED` | classification, reportability, immediate-control authorization, closure |
| Investigation/CAPA reference | `DRAFT`, `LINK_PROPOSED`, `UNDER_REVIEW`, `REJECTED`, `SUPERSEDED` | causation, evidence acceptance, effectiveness, closure, reopening |

AI may summarize or identify a synthetic proposal, but **AI has no autonomous
protected decision authority**. It cannot approve a protected transition,
classification, causation, immediate control, closure, residual risk, or
release outcome.

## Configuration and human-gate retention

`V050-CFG-003` governs the detailed data field policy; `V050-CFG-004` and
`V050-CFG-005` govern readiness and authorization rules; `V050-CFG-006` and
`V050-CFG-007` govern incident and investigation rules; and `V050-CFG-008`
governs lesson/alert eligibility. This baseline selects none of them.

H050-009 through H050-013 remain HOLD. H050-011 is required before technical
alpha release or external effect, H050-012 before representative-user activity,
and H050-013 before milestone outcome or next state. No irreversible migration
may proceed until a separate migration and reversibility packet is reviewed.

## Deterministic verification boundary

`tests/test_v050_domain_data_authority_baseline.py` verifies identity,
synthetic-only scope, ownership separation, default-deny authority,
no-last-write-wins, protected human decisions, unselected configuration,
retained holds, and README registrations. Passing it is architecture-baseline
evidence only, not runtime, qualification, merge, release, or operational
evidence.
