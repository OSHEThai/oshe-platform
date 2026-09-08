---
document_id: ARC-V050-PROF-001
title: v0.5.0 Workforce and Incident Supported Alpha Profile Baseline
document_type: architecture_specification
lifecycle_status: DRAFT
status: APPROVED_FOR_SYNTHETIC_FOUNDATION_ONLY
governing_predecessor: "V050-I003 / GitHub Issue #154 (closed planning record)"
governing_decision: HDEC-V050-E003-EXECUTION-ACTIVATION-014
milestone: "v0.5.0 - Workforce and Incident Alpha"
retained_holds: [H050-009, H050-010, H050-011, H050-012, H050-013]
retained_unselected_configuration: [V050-CFG-003, V050-CFG-004, V050-CFG-005, V050-CFG-006, V050-CFG-007, V050-CFG-008]
credit_boundary: PROFILE_SPECIFICATION_ONLY_NO_PARTICIPANT_ACTIVITY_RUNTIME_QUALIFICATION_OR_RELEASE_CREDIT
---

# v0.5.0 Workforce and Incident Supported Alpha Profile Baseline

This materializes planning `V050-I003` as `V050-E003` for synthetic engineering
specification only. It authorizes no participant, employee, contractor,
reporter, witness, customer, production, medical, credential, evidence, or
incident data; no external environment, identity provider, browser/device
service, email, SMS, storage provider, deployment, or public route is activated.

## Supported development envelope

The bounded target is responsive web specification with synthetic fixtures only.
It may describe English (`en-US`) and Thai (`th-TH`) localization,
`Asia/Bangkok` display formatting, keyboard accessibility, and Chromium-family
development observation. It is not a commitment to production browser, device,
locale, availability, or accessibility certification. Unqualified contexts
include public lookup, unmanaged devices, embedded webviews, native mobile,
hardware scanners, sensors, wearables, real identity federation, and external
notifications.

## Data, confidentiality, and recovery

All fixtures require synthetic identifiers and minimum non-attributable fields.
The profile prohibits raw credentials, biometric values, witness narratives,
free-text incident evidence, unrestricted search, public workforce directories,
and cross-tenant exports. `V050-CFG-003` is UNSELECTED.

Any future local recovery must default-deny, retain no raw credential, validate
current authority before protected state, and quarantine conflict rather than
last-write-wins. Recovery, retention, support ownership, and any service-level
target are not established here.

## Non-binding NFR and human boundary

Every quantitative target is `[NON-BINDING_PROPOSED]` or `[TBD]`; none is a
binding SLA, capacity promise, or operational-readiness claim. `V050-CFG-004`
through `V050-CFG-008` are UNSELECTED. H050-009 through H050-013 remain HOLD;
H050-011 controls release/external effect, H050-012 representative-user
activity, and H050-013 outcome/next state.

Human authority is required for protected readiness, access, classification,
immediate controls, causation, evidence acceptance, closure, release, and risk
decisions. AI has no autonomous protected decision authority.

`tests/test_v050_supported_alpha_profile_baseline.py` verifies identity,
synthetic-only boundaries, localization/accessibility, non-binding and TBD
markings, retained configuration/holds, prohibitions, and README registrations.
