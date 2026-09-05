---
document_id: ARC-V040-COMP-001
title: v0.4.0 OSHE Inspect Standalone Product Composition, Manifest, Default-Deny Role Navigation, Work Queues, and Entitlement Baseline
document_type: architecture_specification
document_version: 1.0.0
lifecycle_status: DRAFT
status: APPROVED_FOR_LOCAL_DEVELOPMENT
date: "2026-09-05"
author_role: Engineering Lead
author_pane: w9:p23
governing_issue: "GitHub Issue #140"
authority_source: HDEC-V040-FOUNDATION-054
governing_decisions:
  - HDEC-V040-FOUNDATION-054
  - HDEC-V030-ENTRY-AND-POLICY-052
  - ADR-0005
  - ADR-0006
milestone: "v0.4.0 - OSHE Inspect Private Alpha"
approved_foundation_gates:
  - H040-001
  - H040-002
  - H040-003
  - H040-004
  - H040-005
  - H040-006
retained_holds:
  - H040-007
  - H040-008
  - H040-009
  - H040-010
  - H040-011
retained_unselected_policies:
  binding_scoring_policy: HUMAN_OWNED_UNSELECTED
  finding_closure_policy: HUMAN_OWNED_UNSELECTED
  offline_authority: HUMAN_OWNED_UNSELECTED
credit_boundary: STANDALONE_COMPOSITION_SPECIFICATION_ONLY_NO_EXTERNAL_DELIVERY_OR_LIVE_OPERATIONAL_ENACTMENT
---

# v0.4.0 OSHE Inspect Standalone Product Composition, Manifest, Default-Deny Role Navigation, Work Queues, and Entitlement Baseline

## 1. Executive Summary & Governance Posture

### 1.1 Authority Baseline & Purpose
This architectural specification establishes the authoritative **Standalone Product Composition, Product Manifest, Default-Deny Role Navigation, Bounded Work Queues, and Entitlement Separation Baseline** for **Milestone v0.4.0 - OSHE Inspect Private Alpha** within `OSHEThai/oshe-platform`. It fulfills the deliverable requirements of **GitHub Issue #140 (`[V040-I029] Standalone OSHE Inspect Product Composition, Navigation, and Work Queues`)** under Roadmap Topic `V040-T07` and standing owner foundation decision `HDEC-V040-FOUNDATION-054`.

The primary objective is to synthesize the discrete private-alpha functional modules (checklists, scheduling, assignments, offline sync, evidence capture, findings, and CAPA governance) into a unified, standalone OSHE Inspect product composition operating within the **Internal Portal Module (`MOD-POR` / `modules/internal-portal`)**.

### 1.2 The Cardinal Security Invariant: Entitlement Never Substitutes for Record Authorization
1. **Entitlement Boundary (`FeatureEntitlement`):** An entitlement represents a commercial feature or license grant at the tenant or user level (e.g. `InspectEnabled: true`). It defines whether the product surface is unlocked for the tenant.
2. **Authorization Boundary (`TargetRecord`):** Record authorization represents object-level security truth evaluated against tenant isolation, geographic/project scope, assigned custody, and role-based permissions (`MOD-IAM`).
3. **Strict Invariant:** Possessing an Inspect product entitlement is a **necessary prerequisite** for accessing the module, but **NEVER suffices to grant access to individual records, work queues, or assets**. An entitled user attempting to access a cross-tenant, cross-project, or role-unauthorized record is immediately denied (`ErrRecordAccessDenied`).

### 1.3 Retained Unselected Policies & Non-Claims
In strict compliance with `ASN-V040-I029-STANDALONE-COMPOSITION-001` and `HDEC-V040-FOUNDATION-054`:
1. **Binding Scoring Policy (`HUMAN_OWNED_UNSELECTED`):** Compliance scoring models presented in product views are advisory projections under Issue #136 (`V040-I025`).
2. **Finding Closure Policy (`HUMAN_OWNED_UNSELECTED`):** CAPA closure and safety risk sign-off rules remain human-owned under Issue #134 (`V040-I023`).
3. **Offline Authority Model (`HUMAN_OWNED_UNSELECTED`):** Operational lease expiration limits and offline queue prioritization remain human-owned under Issue #126 (`V040-I015`).
4. **No Real Public Routes or Cloud Hosting:** Zero public DNS endpoints, CDN edge distributions, external user accounts, or production databases are connected or deployed. All composition models operate on in-memory synthetic fixtures.
5. **Retained Human Gate Holds:** Foundation gates `H040-007` (Technical Release Authorization), `H040-008` (Real Participant / Private-Alpha UAT Onboarding), `H040-009` (Binding Support Ownership), `H040-010` (External Environment/Route Activation), and `H040-011` (Final Milestone Outcome) remain strictly on **`HOLD`**.

---

## 2. Standalone Inspect Product Identity & Manifest

Under `modules/internal-portal/inspect_composition.go`, OSHE Inspect declares an explicit product identity distinct from generic administrative portals:

### 2.1 Manifest Specification (`InspectProductManifest`)

```
┌────────────────────────────────────────────────────────────────────────┐
│                        InspectProductManifest                          │
├────────────────────────────────────────────────────────────────────────┤
│  product_id        : "oshe-inspect"                                    │
│  product_name      : "OSHE Inspect"                                    │
│  product_version   : "v0.4.0-alpha"                                    │
│  supported_modes   : ["STANDALONE_WEB", "OFFLINE_RESPONSIVE"]          │
│  supported_roles   : [INSPECTOR, CHECKLIST_AUTHOR, CAPA_OWNER,         │
│                       INDEPENDENT_REVIEWER, SUPERVISOR, TENANT_ADMIN,  │
│                       PROJECT_MANAGER]                                 │
│  core_capabilities : ["CHECKLIST_EXECUTION", "OFFLINE_LOCAL_SYNC",     │
│                       "EVIDENCE_CAPTURE_PREVIEW",                      │
│                       "FINDING_IDENTIFICATION", "CAPA_GOVERNANCE",     │
│                       "AUDIT_PRESERVATION"]                            │
│  notice_watermark  : "SYNTHETIC_STANDALONE_INSPECT_ALPHA"              │
└────────────────────────────────────────────────────────────────────────┘
```

- **Standalone Operation:** The product surface operates independently without requiring unrelated enterprise ERP, CRM, or external IdP modules.
- **Notice Watermark:** All presented views embed `SYNTHETIC_STANDALONE_INSPECT_ALPHA` to ensure strict non-production audit traceability.

---

## 3. Strict Separation of Entitlements vs. Record Authorization

A foundational vulnerability in multi-tenant SaaS architectures occurs when system designers treat product license entitlements as security authorization. OSHE Inspect strictly decouples these two concepts:

```
[User Request: View Record]
           │
           ▼
[1. Entitlement Pre-Check] ───────> (InspectEnabled == false) ──────────> DENIED (ErrEntitlementRequired)
           │
           ▼ (Entitlement OK)
[2. Record Authorization Check]
     ├─ 2a. Tenant Isolation ────> (record.TenantID != viewer.TenantID) ─> DENIED (ErrRecordAccessDenied)
     ├─ 2b. Project Scope ───────> (record.ProjectID != viewer.ProjectID) > DENIED (ErrRecordAccessDenied)
     ├─ 2c. Role Permission ─────> (Role not in AllowedRoles) ───────────> DENIED (ErrRecordAccessDenied)
     └─ 2d. Assigned Custody ────> (Assigned to someone else) ───────────> DENIED (ErrRecordAccessDenied)
           │
           ▼ (All Checks Pass)
[ACCESS GRANTED TO RECORD]
```

### 3.1 Entitlement-Authorization Invariant Table

| Scenario | Entitlement Status | Record Scope Match? | Role Permitted? | Final Evaluation | Governance Rationale |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **Normal Authorized Access** | Enabled (`true`) | Matches tenant & project | Permitted | **ALLOWED** | Valid entitlement and valid record authorization. |
| **Unentitled Tenant** | Disabled (`false`) | Matches tenant & project | Permitted | **DENIED** (`ErrEntitlementRequired`) | Feature disabled at product/license level. |
| **Cross-Tenant Attack** | Enabled (`true`) | Mismatched tenant | Permitted | **DENIED** (`ErrRecordAccessDenied`) | Entitlement never grants cross-tenant access. |
| **Cross-Project Probing** | Enabled (`true`) | Mismatched project | Permitted | **DENIED** (`ErrRecordAccessDenied`) | Entitlement never bypasses project scope bounding. |
| **Privilege Escalation** | Enabled (`true`) | Matches tenant & project | Unpermitted role | **DENIED** (`ErrRecordAccessDenied`) | Entitlement never bypasses role boundaries. |
| **Unassigned Custody** | Enabled (`true`) | Matches tenant & project | Permitted role, but assigned to another user | **DENIED** (`ErrRecordAccessDenied`) | Entitlement never bypasses assigned custody. |

---

## 4. Default-Deny Role Navigation Catalog (`InspectNavigationCatalog`)

Navigation menus are resolved dynamically per user session under strict default-deny rules:

### 4.1 Role-Specific Navigation Journeys

1. **Inspector Journey (`RoleInspector`):**
   - `inspect-nav-assigned` (`/inspect/assigned`): Active field inspections assigned to the caller.
   - `inspect-nav-offline` (`/inspect/offline`): Downloaded offline packages and local synchronization queue.
   - **Isolation:** Inspectors cannot discover template authoring, CAPA actions, or supervisory schedules.
2. **Checklist Author Journey (`RoleChecklistAuthor`):**
   - `inspect-nav-templates` (`/inspect/templates`): Template library, versions, and publication status.
   - `inspect-nav-questions` (`/inspect/questions`): Question bank, rules, and condition branches.
   - **Isolation:** Authors cannot access field execution queues or approve findings.
3. **CAPA Owner Journey (`RoleCAPAOwner`):**
   - `inspect-nav-capa` (`/inspect/actions`): Safety corrective action items assigned for remediation.
   - `inspect-nav-evidence` (`/inspect/evidence-submit`): Verification evidence upload queue.
   - **Isolation:** CAPA owners cannot review or approve their own evidence submissions.
4. **Independent Reviewer Journey (`RoleIndependentReviewer`):**
   - `inspect-nav-review-queue` (`/inspect/reviews`): Verification queue for completed inspections.
   - `inspect-nav-evidence-verify` (`/inspect/evidence-verify`): Evidence verification and sign-off queue.
   - **Isolation:** Reviewers cannot author checklists or execute field inspections.
5. **Supervisor / Admin Journey (`RoleSupervisor`, `RoleTenantAdmin`):**
   - `inspect-nav-schedules` (`/inspect/schedules`): Recurrence rules and inspection dispatch.
   - `inspect-nav-diagnostics` (`/inspect/diagnostics`): Sync conflicts and operational diagnostics.

### 4.2 Accessibility Standards Compliance (WCAG 2.1 AA)
Every navigation item mandates:
- Non-empty, descriptive `Label`.
- Explicit `AriaLabel` articulating the exact functional intent for assistive technologies.
- Validated programmatically via `item.ValidateAccessibility()`.

---

## 5. Bounded Work Queues & Non-Leaking States (`InspectWorkQueue`)

Operational task queues (`InspectWorkQueue`) aggregate actionable items filtered strictly to the caller's authorized context:

### 5.1 Partitioning & Isolation Rules
1. **Tenant & Project Partitioning:** Items belonging to another tenant or project are completely filtered out.
2. **Role Partitioning:** Tasks are visible only to the designated operational role.
3. **Subject Custody:** Tasks assigned to a specific user subject are visible only to that assignee (or supervisory admins).

### 5.2 Non-Leaking Empty and Denied States
1. **Sanitized Empty State:** When an authorized user has zero pending tasks, the queue returns:
   - `IsEmpty = true`
   - `EmptyStateCode = "NO_ACTIVE_TASKS"`
   - `EmptyMessage = "No pending tasks in your work queue for this project."`
2. **Anti-Enumeration Denial:** When an unauthorized caller queries a work queue or resource outside their project scope, the system fails closed uniformly with `ErrRecordAccessDenied` without revealing whether the project or records exist.

---

## 6. Synthetic Operations Fixture Matrix

The following synthetic YAML fixture illustrates standalone composition, navigation resolution, and entitlement separation:

```yaml
schema_version: "1.0.0"
fixture_id: "fix_syn_standalone_composition_v1"

product_identity:
  product_id: "oshe-inspect"
  product_name: "OSHE Inspect"
  version: "v0.4.0-alpha"
  supported_modes:
    - "STANDALONE_WEB"
    - "OFFLINE_RESPONSIVE"

scenarios:
  # Scenario 1: Inspector Role Navigation Resolution
  - scenario_id: "scen_comp_nav_inspector_01"
    viewer:
      subject: "usr_syn_inspector_01"
      tenant_id: "ten_synthetic_alpha"
      project_id: "prj_plant_safety_01"
      role: "INSPECTOR"
    entitlement:
      inspect_enabled: true
    expected_nav_items_count: 2
    expected_routes:
      - "/inspect/assigned"
      - "/inspect/offline"
    leaked_admin_routes: false

  # Scenario 2: Entitlement Separation from Record Authorization
  - scenario_id: "scen_comp_entitlement_separation_02"
    viewer:
      subject: "usr_syn_inspector_01"
      tenant_id: "ten_synthetic_alpha"
      project_id: "prj_plant_safety_01"
      role: "INSPECTOR"
    entitlement:
      inspect_enabled: true # Entitlement active
    target_record:
      record_id: "ins_syn_clean_01"
      tenant_id: "ten_foreign_beta" # CROSS-TENANT TARGET
      project_id: "prj_plant_safety_01"
    expected_access: false
    expected_error: "ErrRecordAccessDenied"
    invariant_proven: "ENTITLEMENT_NEVER_SUBSTITUTES_FOR_RECORD_AUTHORIZATION"

  # Scenario 3: Sanitized Empty Work Queue
  - scenario_id: "scen_comp_empty_queue_03"
    viewer:
      subject: "usr_syn_inspector_01"
      tenant_id: "ten_synthetic_alpha"
      project_id: "prj_empty_project"
      role: "INSPECTOR"
    entitlement:
      inspect_enabled: true
    expected_total_count: 0
    expected_is_empty: true
    empty_state_code: "NO_ACTIVE_TASKS"
```

---

## 7. Governance Boundaries, Prohibitions & Operational Non-Claims

In strict compliance with `HDEC-V040-FOUNDATION-054` and `ASN-V040-I029-STANDALONE-COMPOSITION-001`:

1. **100% Synthetic Scope Policy (`H040-003`):** All users, roles, manifests, and queues are synthetic in-memory models. Zero real customer data or live operational records are referenced.
2. **Default-Deny Navigation Invariant (`H040-004`):** Access to navigation links and work items evaluates strictly on default-deny.
3. **No External Route or Cloud Activation (`H040-007` & `H040-010` HOLD):** Zero public routes, internet gateways, or cloud environments are activated.
4. **No Participant Onboarding (`H040-008` HOLD):** Zero real field personnel or pilot users are recruited.
5. **Specification-Only Credit:** Delivery of this baseline confers documentation and architectural baseline credit only; zero operational release, deployment, or residual-risk acceptance is claimed.
