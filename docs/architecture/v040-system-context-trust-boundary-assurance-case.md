# v0.4.0 System Context, Trust Boundaries, and Safety Assurance Case

| Metadata Field | Value |
| :--- | :--- |
| **Document ID** | `DOC-ASSURE-V040-001` |
| **Work Item** | `V040-I033` (GitHub Issue #144) |
| **Lifecycle State** | `DRAFT_STATIC_ASSURANCE_CASE` |
| **Planning Status** | `PLANNING_ONLY` |
| **Governing Decisions** | `HDEC-V040-FOUNDATION-054`, `HDEC-V040-SCORING-058`, `H040-001`..`H040-006` Approved / `H040-007`..`H040-011` HOLD |
| **Target Milestone** | `v0.4.0 - OSHE Inspect Private Alpha` |
| **Security Risk Classification** | Proposed Architecture Planning & Boundary Specification |

---

## 1. Executive Summary & Purpose

This document establishes the bounded, static system-context, trust-boundary, and safety-assurance planning case for Milestone `v0.4.0 OSHE Inspect Private Alpha` under Issue #144 (`V040-I033`).

The primary objective is to define the proposed architectural boundaries, planned data-flow directions, conceptual trust domains, and control matrices governing operational inspections, hazard identification, corrective and preventive actions (CAPA), evidence integrity, mobile offline synchronization, and derived reporting.

### Explicit Planning-Only, Non-Claims, and Non-Activation Declaration
In strict adherence to approved Sole Human Owner foundation decisions **HDEC-V040-FOUNDATION-054** and **HDEC-V040-SCORING-058**:
- **Planning-Only Status:** This document is an architectural planning specification and static design baseline only. It operates strictly under lifecycle state `DRAFT_STATIC_ASSURANCE_CASE`.
- **Proposed and Static Concepts Only:** Every system, subsystem, module, storage tier, encryption concept, API endpoint, UI/browser interface, offline mechanism, test requirement, role assignment, and control mechanism described in this document is explicitly **proposed, planned, and static**. Nothing herein represents an implemented, validated, or running capability.
- **Zero Runtime Claims:** This static document makes no claims of live runtime deployment, system execution, operational availability, throughput, or performance behavior.
- **Zero Security Enforcement Claims:** No active security enforcement, access control implementation, or real-time barrier operation is asserted as functioning.
- **Zero Certification Claims:** No external certification (such as ISO, SOC, or regulatory safety accreditations) is asserted, granted, or implied.
- **Zero Data Retention Claims:** No operational data retention, long-term archival guarantee, or compliance storage behavior is enacted.
- **No Schema, Service, Container, Event Contract, or Activation:** This static record creates **no database schema, migration, microservice, container configuration, broker event contract, external route, or infrastructure activation**.
- **Retained Foundation Holds:** Foundation holds `H040-007` through `H040-011` remain strictly on `HOLD`. Zero activation, release, or operational authorization is granted.

---

## 2. Proposed System Context & Architecture Overview

The proposed OSHE Platform v0.4.0 Private Alpha architecture establishes a modular design model for safety inspection workflows, structured findings tracking, evidence custody, offline-first field capture, and non-authoritative reporting projections.

### Proposed Architectural Subsystems and Modules (Planning Models)

1. **Proposed `MOD-WFA` (Workflow & Inspection Action Subsystem Planning Model):**
   - Planned lifecycle states (`DRAFT` $\to$ `IN_PROGRESS` $\to$ `UNDER_REVIEW` $\to$ `APPROVED` $\to$ `CLOSED`).
   - Planned deterministic compliance scoring direction under **HDEC-V040-SCORING-058** (`MODEL_2_WEIGHTED`, `U1_QUARANTINE_DENOMINATOR`, `R1_ROUND_HALF_UP`, `CF1_PRIORITY_FLAG`, 80.00% / 8000 bps threshold).
   - Planned fail-closed priority model (`critical > UNKNOWN > score`), deferred override boundary under Gate `H040-004`, and autonomous AI decision prohibition.
2. **Proposed `MOD-EVD` (Evidence Integrity & Capture Subsystem Planning Model):**
   - Planned content-addressable reference design using SHA-256 digests for inspection media and documentation.
   - Proposed tamper-evident bundle binding linking evidence references to checklist items and findings.
3. **Proposed `MOD-CAPA` (Finding & Corrective Action Subsystem Planning Model):**
   - Planned hazard triage, critical severity tagging, remediation tracking, and re-inspection verification workflows.
4. **Proposed `MOD-OFF` (Mobile Offline Synchronization Planning Model):**
   - Planned local encrypted sandbox storage model for provisional field inspection drafts.
   - Proposed three-way conflict detection model with final reconciliation deferred to server-side ingestion.
5. **Proposed `MOD-REP` (Controlled Reporting & Localization Subsystem Planning Model):**
   - Planned reporting catalogs, dashboard aggregations, and export generation renderers.
   - Proposed mandatory inclusion of the `DERIVED_OUTPUT_NON_AUTHORITY` notice across all reporting projections.
6. **Proposed `MOD-ORG` & `MOD-IAM` (Hierarchy & Access Control Planning Model):**
   - Planned multi-tenant isolation, organizational hierarchy levels (`Tenant` $\to$ `Company` $\to$ `Project` $\to$ `Site` $\to$ `Area`), and role-based access specifications.
7. **Proposed `MOD-PORTAL` (Standalone Inspect Composition Planning Model):**
   - Planned user-facing navigation and inspection views, structured under client-side untrusted assumptions.

### Proposed System Context & Boundary Architecture Diagram

```mermaid
flowchart TD
    subgraph Untrusted_Clients["Untrusted Client Domain (Proposed Boundary TB-01)"]
        MobileClient["Proposed Mobile Inspection App\n(Planned Local Encrypted Sandbox)"]
        WebPortal["Proposed Web Inspection Portal\n(Planned Browser Interface)"]
    end

    subgraph API_Edge["Platform API Boundary (Proposed Boundary TB-02)"]
        Gateway["Planned API Boundary & Middleware\n(Proposed Session Validation & Scope Derivation)"]
    end

    subgraph Authoritative_Core["Selected Persistence Direction (Proposed Boundary TB-03)"]
        PG[("Authoritative PostgreSQL Store\n(Selected Planning Direction: Single Source of Truth)")]
        Outbox[("Transactional Outbox Pattern\n(Selected Planning Direction: Atomic State & Event)")]
    end

    subgraph Event_Transport["Selected Event Transport Direction (Proposed Boundary TB-04)"]
        NATS["NATS JetStream\n(Selected Planning Direction: Event Coordination Only)")]
    end

    subgraph Projection_Plane["Selected Derived Projection Direction (Proposed Boundary TB-05)"]
        Meili[("Meilisearch\n(Selected Planning Direction: Projection-Only Search)")]
        Valkey[("Valkey\n(Selected Planning Direction: Cache & Rate Coordination Only)")]
    end

    subgraph Derived_Reporting["Selected Reporting Direction (Proposed Boundary TB-06)"]
        ReportCatalog["MOD-REP Catalog & Renderer\n(Planned DERIVED_OUTPUT_NON_AUTHORITY)"]
    end

    MobileClient -->|"Planned HTTPS / REST (Untrusted Scopes)"| Gateway
    WebPortal -->|"Planned HTTPS / REST (Untrusted Scopes)"| Gateway
    Gateway -->|"Planned Validated Commands"| PG
    PG ---|"Planned Transactional Write"| Outbox
    Outbox -->|"Planned Outbox Relay"| NATS
    NATS -->|"Planned Projection Consumer"| Meili
    NATS -->|"Planned Invalidation Events"| Valkey
    NATS -->|"Planned Metrics Ingestion"| ReportCatalog
    Gateway -.->|"Planned Scoped Search Read"| Meili
    Gateway -.->|"Planned Cached Read"| Valkey
    Gateway -.->|"Planned Export Queries"| ReportCatalog
```

*Note: All architectural entities, data stores, and transport paths in the diagram represent the selected architecture planning direction from `data-projection-boundaries.md` (ADR-0006). They do not represent deployed, active, or running components.*

---

## 3. Data-Flow and Trust Boundaries (Selected Planning Direction)

The platform architecture adopts the selected data-projection and storage direction documented in `data-projection-boundaries.md` (ADR-0006). These specifications represent architectural design targets, not active infrastructure:

1. **Authoritative PostgreSQL (Selected Planning Direction):**
   - PostgreSQL is selected as the planned sole authoritative persistent store for operational data.
   - All domain state changes are planned to execute via transactional operations with ACID guarantees.
   - This document provisions no database tables, schemas, or migrations.

2. **Transactional Outbox Pattern (Selected Planning Direction):**
   - Domain mutations and event payloads are planned to commit atomically within the PostgreSQL transaction boundary.
   - This architectural pattern is planned to eliminate dual-write inconsistencies.

3. **NATS JetStream (Selected Planning Direction):**
   - NATS JetStream is selected as the planned event streaming backbone for asynchronous messaging.
   - Under this design, JetStream is strictly a transport and coordination mechanism, never an operational source of truth.

4. **Meilisearch Projection-Only (Selected Planning Direction):**
   - Meilisearch is selected strictly as a search projection store derived from transactional outbox events.
   - Meilisearch is planned to be non-authoritative and read-only to clients.
   - If search index drift or corruption occurs, the planned recovery mechanism is full re-indexing from PostgreSQL via outbox replay.

5. **Valkey Cache-Only (Selected Planning Direction):**
   - Valkey is selected strictly for caching, rate-limit state, and transient coordination.
   - Valkey is planned to never store authoritative business state. Cache evictions are designed to fall back safely to PostgreSQL.

6. **Failed Projection Rebuilding:**
   - Failures or desynchronization in projection or cache tiers are designed to leave PostgreSQL unaffected.
   - Planned restoration relies solely on replaying outbox event logs from PostgreSQL.

---

## 4. Untrusted Client Scopes & Default-Deny Concept

### Proposed Handling of Untrusted Client Scopes
The architectural baseline specifies that all client-supplied parameters are treated as untrusted:
- **`tenant_id` Scopes:** Planned to be derived server-side from verified authentication tokens rather than accepted from client headers or payloads.
- **`project_id` & `site_id` Scopes:** Planned to require server-side verification against active user assignment records in PostgreSQL.
- **Role Claims:** Proposed design ignores client-asserted roles; permissions are planned to be evaluated server-side per request.
- **Client Timestamps:** Proposed design treats client timestamps as unverified hints, using monotonic server time for authoritative ordering.

### Proposed Default-Deny Policy
- The architectural design specifies a default-deny stance: requests without valid credentials and explicit authorization are planned to be rejected.
- Unrecognized or malformed scope claims are planned to result in immediate fail-closed denial with stable error codes.

---

## 5. Discrete Multi-Dimensional Isolation Boundaries (Proposed Design)

| Boundary Domain | Boundary Identifier | Proposed Isolation Mechanism & Planning Rules |
| :--- | :--- | :--- |
| **Tenant Boundary** | `TB-TENANT` | Planned relational isolation by `tenant_id`. Database queries are planned with mandatory tenant filtering; cross-tenant references fail closed. |
| **Project Boundary** | `TB-PROJECT` | Planned hierarchical containment (`Tenant` $\to$ `Company` $\to$ `Project`). Inspection access is planned to require active project assignment. |
| **Site / Area Boundary** | `TB-SITE` | Planned physical site and area constraints pinning checklists and findings to designated locations (`SiteID`, `AreaID`). |
| **Contractor Boundary** | `TB-CONTRACTOR` | Planned third-party contractor access restricted to assigned corrective actions, with directory browsing disabled. |
| **Evidence Boundary** | `TB-EVIDENCE` | Planned content-addressable reference model with SHA-256 cryptographic hashes bound immutably to inspection records. |
| **Offline Boundary** | `TB-OFFLINE` | Planned local encrypted sandbox storage on mobile devices with provisional status; authoritative status requires server ingestion. |
| **Report Boundary** | `TB-REPORT` | Planned derived reporting projections carrying the mandatory `DERIVED_OUTPUT_NON_AUTHORITY` notice. |
| **Export Boundary** | `TB-EXPORT` | Planned complete-record export generation with cryptographic manifests containing source and rendered SHA-256 digests. |

---

## 6. Prevention, Detection, Evidence, and Owner (PDEO) Planning Matrix

The following matrix defines the proposed architectural threat vectors, planned prevention mechanisms, proposed detection strategies, planned evidence verification criteria, and designated governance roles:

| Domain | Threat / Risk Vector | Planned Prevention Mechanism | Proposed Detection Strategy | Planned Evidence Artifact / Verification Criteria | Proposed Governed Owner Role |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **Tenant Isolation** | Cross-tenant data leakage via forged request parameters | Planned server-side tenant derivation from session token; mandatory tenant query filtering | Proposed runtime query inspection and automated cross-tenant isolation test suite | Planned tenant isolation verification test specification | Architecture & Data Lead |
| **Data Flow** | Unauthorized write to PostgreSQL from projection tiers | Planned network segmentation; projection credentials have zero write privileges to PostgreSQL | Proposed database connection permission audit and role monitoring | Planned database privilege audit and boundary specification | Platform Security Lead |
| **Projection Integrity** | Search index desynchronization or cache pollution | Planned unidirectional data flow via PostgreSQL outbox and NATS JetStream; direct writes rejected | Proposed periodic hash reconciliation between PostgreSQL source records and search index | Planned outbox replay test harness specification | Infrastructure & Core Lead |
| **Client Scope** | Privilege escalation via forged `project_id` or role | Planned server-side validation of user participation; client scope claims discarded | Proposed audit logging of scope mismatch rejections | Planned authorization boundary test specification | Authorization Lead |
| **Evidence Custody** | Post-inspection alteration or replacement of hazard media | Planned SHA-256 digest computation at capture, stored immutably in PostgreSQL | Proposed hash verification on retrieval and digest validation in export manifests | Planned cryptographic digest verification test harness | Evidence & Assurance Lead |
| **Offline Sync** | Stale offline mobile draft overwriting concurrent online edits | Planned three-way conflict detection and server-side reconciliation in `MOD-OFF` | Proposed conflict audit logging and rejected sequence tracking | Planned offline synchronization test specification | Mobile Client Lead |
| **Scoring Safety** | High numerical score masking active critical hazard | Planned `CF1_PRIORITY_FLAG` locking outcome to `NON_COMPLIANT_CRITICAL` while reporting score unmasked | Planned mandatory three-predicate conjunction check before compliance approval | Planned scoring safety qualification test specification | Test & Quality Lead |
| **Override Safety** | Unauthorized managerial bypass of critical inspection failures | Planned unconditional override denial under deferred Gate `H040-004` | Proposed append-only audit logging of all override attempts | Planned override denial audit log verification specification | Human Reviewer / PM Secretary |
| **Autonomous AI** | Autonomous AI agent clearing critical finding or authorizing state change | Planned strict role checking; unconditional fail-closed denial for AI roles | Proposed immediate security alert on AI transition authorization attempt | Planned autonomous AI boundary test specification | Sole Human Owner |
| **Report Authority** | User treating derived report metric as binding operational authority | Planned mandatory `DERIVED_OUTPUT_NON_AUTHORITY` header on all reports and exports | Proposed schema validation checking non-authority designation on metric definitions | Planned non-authority notice verification specification | Reporting Lead |

---

## 7. Retained Foundation Holds & Non-Authority Affirmation

Under **HDEC-V040-FOUNDATION-054**, the following foundation holds remain in active **`HOLD`** status:

| Hold ID | Governance Subject | Status | Enforcement Constraint |
| :--- | :--- | :--- | :--- |
| `H040-007` | Technical release authorization | **HOLD** | HOLD: No technical release authorization is granted. |
| `H040-008` | Real participant, private-alpha, and UAT engagement | **HOLD** | HOLD: No real participant, private-alpha, or UAT engagement authorization is granted. |
| `H040-009` | Binding support and manual-fallback operational ownership | **HOLD** | HOLD: No binding support or manual-fallback operational ownership authorization is granted. |
| `H040-010` | External environment, device, account, route, storage, and notification activation | **HOLD** | HOLD: No external environment, device, account, route, storage, or notification activation is granted. |
| `H040-011` | Final outcome, residual-risk acceptance, and v0.5.0 entry decision | **HOLD** | HOLD: No final outcome, residual-risk acceptance, or v0.5.0 entry decision authorization is granted. |

Every hold row remains in active **HOLD** status. No activation, authorization, release, operational deployment, or hold lift is granted.
