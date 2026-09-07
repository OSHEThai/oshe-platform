---
document_id: ARC-V050-DECREC-001
title: v0.5.0 Workforce and Incident Alpha Foundation Boundaries
document_type: architecture_decision_record
document_version: 1.0.0
lifecycle_status: APPROVED
status: APPROVED_FOR_SYNTHETIC_FOUNDATION_MATERIALIZATION
authority_source: HDEC-V050-WAVE0-EXECUTION-ACTIVATION-001
decided_at: '2026-09-07T10:40:15Z'
decided_by: Sole Human Owner
governing_execution_successor: V050-E001
planning_predecessor: GitHub Issue #152 / V050-I001
milestone: v0.5.0 - Workforce and Incident Alpha
approved_gates:
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
credit_boundary: FOUNDATION_BOUNDARY_MATERIALIZATION_ONLY
---

# v0.5.0 Workforce and Incident Alpha Foundation Boundaries

## Purpose and Credit Boundary

This record materializes the owner-approved v0.5.0 planning conditions as
source-governed constraints for synthetic foundation work. It is not evidence
that a Workforce or Incident capability is implemented, qualified, merged,
released, externally activated, or ready for operational use.

The planning predecessor is closed GitHub Issue #152 (`V050-I001`). This
execution record neither reopens nor closes that predecessor. Any later
execution successor must retain the boundary in this document and earn its own
implementation, validation, independent-review, merge, and closure evidence.

## Approved Foundation Conditions

### H050-001 - Scope and alpha gates

The supported scope is a narrow Workforce and Incident alpha vertical slice.
Workforce and Incident may consume controlled public contracts from OSHE
Inspect, but they must not write OSHE Inspect records directly. Gate A is
technical qualification; Gate B requires attributable representative-user
evidence only after the separately authorized human gates. Production, pilot
operation, unrestricted public lookup, and automated protected decisions are
excluded.

### H050-002 - Workforce authority

One person record links to trusted identity by reference and does not duplicate
authentication identity. Employment, contractor-worker relationships,
project/site assignments, sponsorship, appointment, validity, ownership,
transfer, overlap, lifecycle, and historical context are explicit,
tenant-scoped, and default-deny. A destructive or irreversible migration needs
a separate approved migration and recovery packet.

### H050-003 - Data and confidentiality

Only synthetic or redacted fixtures are permitted. Real workforce, participant,
witness, incident, credential, or evidence data is prohibited unless a
separate field allowlist, purpose, classification, owner, audience, retention,
correction, export, and restriction policy is approved. Medical records,
production secrets, unrestricted worker lookup, and legal-completeness claims
remain excluded. Confidentiality defaults to restricted search and minimum
necessary access.

### H050-004 and H050-005 - Readiness and worker-ID boundary

Unknown, expired, disputed, and exceptional readiness states must remain
explicit; competency must not be inferred. A readiness view is communicative
only and never grants an operational, legal, or physical-access authorization.
An internal QR is an opaque untrusted identifier whose authenticated resolver
checks current scope and status. Protected data is never carried in the QR
value, and cached or offline information cannot establish binding access
status.

### H050-006 through H050-008 - Incident, investigation, and learning

Hazard, near-miss, and incident intake starts unclassified and requires human
triage. AI may prepare forms, fixtures, and rule options but may not make legal,
reportability, or protected safety classifications. Investigation, root cause,
and CAPA require explicit authority, independent review, evidence provenance,
and authorized human closure; none may be inferred or closed automatically.
Lessons and alerts are internal-only, anonymized, deny-by-default, mapped to a
source, field allowlist, named audience, review, approval, expiry, withdrawal,
and supersession control.

## Cross-Cutting Invariants

- Default-deny authorization and tenant, company, project, site, contractor,
  worker, reporter, witness, investigation, CAPA, and audience isolation apply
  to every future protected workflow.
- Current authority must be checked at every protected transition. No report,
  QR, offline client, dashboard, notification, derived metric, or AI output is
  operational authority.
- Protected state never uses last-write-wins; conflicting or stale state is
  rejected or quarantined for authorized human reconciliation.
- AI has zero autonomous safety, legal, reportability, permit, access,
  investigation-closure, CAPA-closure, or residual-risk authority.

## Retained Holds and Non-Claims

H050-009 through H050-013 remain held or not due. They govern participant and
protocol authorization, alpha operating commitments, technical alpha release
or external effect, representative-user validation, and outcome/residual-risk/
v0.6.0 direction respectively.

This foundation authorizes no real data, participant activity, external
service/device/account/route activation, physical or operational safety action,
deployment, protected merge, release, production, or residual-risk acceptance.
