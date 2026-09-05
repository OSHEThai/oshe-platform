---
document_id: REL-V040-ACTPKT-001
title: v0.4.0 OSHE Inspect Private Alpha Controlled Activation Prework Packet (H040-010 HOLD)
document_type: release_activation_packet
document_version: 1.0.0
lifecycle_status: DRAFT
status: HELD_PENDING_SOLE_HUMAN_OWNER_ACTIVATION_H040_010
date: "2026-09-05"
author_role: Release and Evidence Lead
author_pane: w9:p16
governing_issue: "GitHub Issue #146"
authority_source: HDEC-V040-FOUNDATION-054
governing_decisions:
  - HDEC-V040-FOUNDATION-054
  - HDEC-V030-ENTRY-AND-POLICY-052
  - ADR-0005
  - ADR-0006
  - ADR-0007
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
retained_unselected_choices:
  target_cloud_environment: NOT_SELECTED
  external_dns_domain: NOT_SELECTED
  cdn_edge_distribution: NOT_SELECTED
  cloud_storage_bucket: NOT_SELECTED
  notification_provider: NOT_SELECTED
  real_test_devices: NOT_COLLECTED
  pilot_user_accounts: NOT_COLLECTED
  budget_and_cost_ceiling: NOT_SELECTED
  production_seed_data: NOT_COLLECTED
credit_boundary: PLANNING_ONLY_ACTIVATION_PREWORK_NO_EXTERNAL_ACTION_OR_LIVE_DEPLOYMENT
---

# v0.4.0 OSHE Inspect Private Alpha Controlled Activation Prework Packet (H040-010 HOLD)

## 1. Executive Summary & Governance Posture

### 1.1 Authority Baseline & Purpose
This document establishes the authoritative, planning-only **Controlled Private Alpha Environment Activation Prework Packet** for **Milestone v0.4.0 - OSHE Inspect Private Alpha** within `OSHEThai/oshe-platform`. It fulfills the deliverable specifications of **GitHub Issue #146 (`[V040-I035] Private Alpha Environment, Test Devices, Accounts, Routes, Monitoring, and Maintenance Activation Packet`)** under Roadmap Topic `V040-T08` and standing owner foundation decision `HDEC-V040-FOUNDATION-054`.

The primary objective is to define a comprehensive, fail-closed prework baseline covering:
- Declarative synthetic-only environment and resource inventory schemas.
- Least-privilege configuration checklists for compute, storage, and networking.
- Rigorous route-disabled verification protocols asserting zero public internet exposure.
- Health, conflict, and quarantine monitoring architectures.
- Tabletop disaster recovery, backup, and restore verification runbooks.
- Safe post-alpha rollback, state purge, and infrastructure cleanup procedures.
- A structured Sole Human Owner activation decision checklist.

### 1.2 Gate H040-010 HOLD Invariant
In strict accordance with `HDEC-V040-FOUNDATION-054` and `ASN-V040-I035-ALPHA-ACTIVATION-PREWORK-001`, **Human Gate `H040-010` (External Environment, Account, Route, Storage, Notification, or External-Effect Activation) remains strictly on `HOLD`**.

- **Planning-Only Boundary:** This packet defines architecture, schemas, verification procedures, and risk models. It carries **zero authority to execute, activate, provision, or mutate** any external system, cloud service, domain record, or credential.
- **Explicit Unselected Placeholders:** Wherever an operational choice requires real cloud infrastructure, external provider selection, device procurement, real participant onboarding, or commercial cost commitment, this packet explicitly records `HOLD`, `NOT_SELECTED`, or `NOT_COLLECTED`.
- **Prohibitions Strictly Observed:**
  1. Zero external public routes, DNS records, or ingress controllers are activated.
  2. Zero cloud provider accounts, API keys, credentials, or payment profiles are created or modified.
  3. Zero real participant, contractor, or workforce personal data is collected or ingested (`H040-003` / `H040-008` HOLD).
  4. Zero software release tagging, commercial distribution, or cryptographic signing is performed (`H040-007` HOLD).
  5. Zero residual risk acceptance or Milestone v0.5.0 entry authorization is claimed (`H040-011` HOLD).

---

## 2. Synthetic-Only Resource & Environment Inventory Schema

The private alpha operates strictly on local loopback and in-memory synthetic adapters. The declarative inventory schema below defines all required infrastructure parameters, with all real external choices explicitly designated as `HOLD`, `NOT_SELECTED`, or `NOT_COLLECTED`:

```yaml
# Schema: OSHE-Inspect-Alpha-Environment-Inventory-v1.0
schema_version: "1.0.0"
inventory_id: "inv_syn_alpha_environment_v1"
governing_gate: "H040-010"
gate_status: "HOLD"

environment_profile:
  environment_tier: "LOCAL_SYNTHETIC_ALPHA"
  hosting_provider: "NOT_SELECTED"          # HOLD: Requires Sole Human Owner decision
  cloud_region: "NOT_SELECTED"              # HOLD: Default evaluation anchor Asia/Bangkok
  target_cluster_id: "NOT_SELECTED"         # HOLD: Zero cloud clusters provisioned
  compute_architecture: "x86_64 / arm64 local development hosts only"

network_and_routing:
  public_dns_zone: "NOT_SELECTED"           # HOLD: Zero public domains registered
  canonical_hostname: "NOT_SELECTED"        # HOLD: No external hostname configured
  ingress_controller: "NOT_SELECTED"        # HOLD: Ingress routes disabled
  tls_certificate_source: "NOT_SELECTED"    # HOLD: Zero public CA certificates requested
  bind_address: "127.0.0.1"                 # Loopback only for local alpha verification
  bind_port: 8080                           # Private local development port

persistence_and_storage:
  database_engine: "LOCAL_SQLITE_SYNTHETIC" # In-memory or ephemeral local sqlite
  database_host: "localhost"
  database_encryption_at_rest: "NOT_SELECTED" # Production KMS unselected
  evidence_blob_storage: "LOCAL_SYNTHETIC_MEMORY" # MemoryStorageAdapter (modules/files-evidence)
  cloud_storage_bucket: "NOT_SELECTED"      # HOLD: AWS S3 / R2 / GCS unselected
  backup_storage_target: "LOCAL_EPHEMERAL_PATH" # Target directory under .local-ci/

identities_and_accounts:
  authentication_mode: "LOCAL_SYNTHETIC_IAM" # Modular in-memory bearer token hashing
  external_idp_provider: "NOT_SELECTED"    # HOLD: Zero Okta / Entra / Auth0 integrations
  pilot_user_accounts: "NOT_COLLECTED"      # HOLD: Zero real users onboarded
  synthetic_test_subjects:
    - subject_id: "usr_syn_inspector_01"
      role: "Inspector"
    - subject_id: "usr_syn_author_01"
      role: "Checklist Author"
    - subject_id: "usr_syn_capa_01"
      role: "CAPA Owner"
    - subject_id: "usr_syn_reviewer_01"
      role: "Independent Reviewer"

client_devices:
  procurement_status: "NOT_COLLECTED"       # HOLD: No physical hardware procured
  supported_platforms:
    - platform: "Desktop Chrome / Edge"
      test_mode: "Local Headless & DevTools Emulation"
    - platform: "Android Chrome Mobile"
      test_mode: "Local Viewport Emulation (360x640)"

notifications_and_messaging:
  delivery_channel: "LOCAL_SYNTHETIC_QUEUE" # In-app SQLite queue (ChannelLocalSink)
  smtp_relay_host: "NOT_SELECTED"           # HOLD: Outbound email disabled
  sms_gateway_provider: "NOT_SELECTED"      # HOLD: Twilio / SMS disabled
  push_notification_service: "NOT_SELECTED" # HOLD: FCM / APNS disabled

financial_and_budget:
  cost_center: "NOT_SELECTED"               # HOLD: Zero commercial billing accounts
  monthly_spend_ceiling_usd: "0.00"         # Strict zero-cost invariant for local alpha
  authorized_payment_method: "NOT_SELECTED" # HOLD: Zero credit cards or invoices linked
```

---

## 3. Least-Privilege Configuration Checklist

To guarantee defense-in-depth and prevent inadvertent exposure, any potential activation of a private-alpha environment must enforce the following technical controls:

| Domain | Control Specification | Verification Method | Pre-Activation Status |
| :--- | :--- | :--- | :--- |
| **Network Ingress** | Bind listeners strictly to localhost (`127.0.0.1`) or private VPC subnet; zero listening on `0.0.0.0`. | `netstat` / port socket assertion | **VERIFIED_LOCAL_ONLY** |
| **Public Egress** | Restrict outbound traffic; block SMTP (ports 25, 465, 587) and unwhitelisted external HTTP/S. | Firewall egress rule check | **VERIFIED_LOCAL_ONLY** |
| **IAM Authority** | Enforce Default-Deny RBAC (`H040-004`); reject unauthenticated calls with 401; reject cross-tenant IDOR. | Automated security regression suite | **VERIFIED_LOCAL_ONLY** |
| **Storage Access** | File system storage paths jailed to application working directory; reject path traversal (`../`). | Negative test suite `NEG-TEST-03` | **VERIFIED_LOCAL_ONLY** |
| **Credential Hygiene** | Zero plaintext secrets or passwords in configuration files; API tokens generated with 32-byte entropy. | `.ai/tools/validate_agent_os.py` scan | **VERIFIED_LOCAL_ONLY** |
| **Database Scoping** | Every database transaction scoped by `tenant_id` foreign key filter; zero cross-tenant joins. | Database query predicate auditor | **VERIFIED_LOCAL_ONLY** |
| **Audit Immutability** | Append-only audit logging for all protected state transitions; monotonic state versioning (`H040-005`). | Concurrency & quarantine test harness | **VERIFIED_LOCAL_ONLY** |

---

## 4. Route-Disabled Verification Protocol

Prior to connecting any network interface, the system must execute the following automated route-disabled verification protocol:

```
┌────────────────────────────────────────────────────────────────────────┐
│                   ROUTE-DISABLED VERIFICATION PIPELINE                 │
│                                                                        │
│  [Step 1: DNS Resolution Check]                                        │
│  Assert query for alpha hostname returns NXDOMAIN (No public record).  │
│                                                                        │
│  [Step 2: Socket Listener Audit]                                       │
│  Assert zero listeners on 0.0.0.0 or public interface IPs.             │
│                                                                        │
│  [Step 3: Ingress Proxy Inspection]                                    │
│  Assert external ingress routes return HTTP 404 / Connection Refused.  │
│                                                                        │
│  [Step 4: Anti-Indexing Header Verification]                           │
│  Assert X-Robots-Tag: noindex, nofollow, noarchive on all responses.   │
│                                                                        │
│  [Step 5: External Notification Block Audit]                           │
│  Assert ChannelLocalSink exclusivity; external gateways return error.  │
└────────────────────────────────────────────────────────────────────────┘
```

- **Pass Criteria:** 100% of checks must confirm that the application is unreachable from the public internet. If any check detects a live public listener or routable DNS record, the environment must immediately halt.

---

## 5. Operational Monitoring & Observability Baseline

The monitoring architecture provides real-time diagnostic visibility across operational health, concurrency conflicts, and failure modes:

### 5.1 Monitored Diagnostic Telemetry
1. **Application Health (`/healthz`):** Ephemeral loopback check reporting process uptime, memory consumption, and SQLite connection pool status.
2. **Conflict Quarantine Rate (Gate `H040-005`):** Monitored count of concurrent update collisions routed to `QUARANTINED_CONFLICT`. A spike indicates offline synchronization contention or client clock drift.
3. **Evidence Ingestion & Digest Verification:** Tracks evidence upload volumes, average byte size, and digest verification latency. Alerts emitted on `ErrTamperDetected` or `ErrInvalidMediaType`.
4. **Local Notification Sink Queue:** Tracks queued, delivered, and failed synthetic in-app notifications. Alerts emitted on `DIAG_NOTIFICATION_QUARANTINED`.
5. **Session Revocation Volume:** Monitors active bearer token invalidations and security logouts.

### 5.2 Alert Thresholds & Escalation
- **Warning:** Concurrency conflict rate $> 5\%$ of sync batches.
- **Critical:** Any occurrence of `ErrTamperDetected`, `ErrCrossTenantLinkage`, or `ErrProhibitedFieldDetected`.
- **Action:** Critical alerts trigger an immediate administrative hold on affected tenant sessions.

---

## 6. Backup, Restore, and Disaster Recovery Tabletop Runbook

To validate operational resilience without risking customer data, the following tabletop procedure defines disaster recovery verification:

### 6.1 Tabletop Backup Execution
1. **Snapshot Creation:** Freeze in-memory/sqlite write transactions.
2. **Deterministic State Export:** Export application graph (tenants, checklists, inspections, findings, evidence metadata, audit log) to canonical JSON artifact.
3. **Cryptographic Sealing:** Compute SHA-256 root digest across the exported backup payload.
4. **Storage Target:** Write backup archive to encrypted local staging directory (`.local-ci/backups/`).

### 6.2 Tabletop Restore & Verification
1. **Clean Slate Initialization:** Instantiate a fresh, empty application runtime in an isolated scratch worktree.
2. **Integrity Pre-Check:** Recompute SHA-256 digest of the backup archive; assert exact match against recorded manifest digest.
3. **State Ingestion:** Hydrate relational tables and in-memory caches from the backup archive.
4. **Lineage & Audit Validation:** Verify that 100% of historical audit records, state versions, and evidence digests reconstruct with zero variance.
5. **Pass Criterion:** The restored instance must pass the complete regression test suite (`test_v040_walking_skeleton_integration_harness.py`) with zero errors.

---

## 7. Rollback, Purge & Post-Alpha Cleanup Procedure

Following the conclusion of any private-alpha testing cycle, the environment must be decommissioned cleanly:

### 7.1 Rollback Protocol
1. **Session Termination:** Invalidate all active synthetic bearer tokens in `session_revocation_registry`.
2. **Listener Shutdown:** Terminate local application processes and release network sockets.
3. **Storage Purge:**
   - Execute secure wipe of local SQLite database files (`oshe_alpha.db*`).
   - Remove ephemeral IndexedDB stores and browser session storage from test devices.
   - Delete temporary evidence blobs from local storage staging directories.
4. **Worktree & Branch Cleanup:** Remove isolated git worktrees and delete short-lived feature branches.

### 7.2 Post-Alpha Artifact Preservation
- Historical audit journals and qualification test reports are archived to permanent repository governance records.
- Zero residual unencrypted credentials, test caches, or temporary logs remain on development workstations.

---

## 8. Sole Human Owner Activation Decision Checklist (UNFILLED / HOLD)

> **MANDATORY GOVERNANCE NOTICE:** This section is strictly reserved for the Sole Human Owner. Automated agents, subagents, and scripts possess zero authority to populate, check, or execute this decision.

Prior to authorizing any potential transition of Gate **`H040-010` from `HOLD`**, the Sole Human Owner must review and sign:

```yaml
# Decision Checklist: HDEC-V040-ACTIVATION-058 (UNFILLED / PENDING)
schema_version: "1.0.0"
governing_gate: "H040-010"
activation_status: "HOLD_PENDING_SOLE_HUMAN_OWNER"

prerequisite_review_checklist:
  - requirement: "Independent security review of private-alpha architecture completed"
    verified: false
  - requirement: "Route-disabled verification protocol executed with 100% pass rate"
    verified: false
  - requirement: "Tabletop backup and restore exercise verified with zero data loss"
    verified: false
  - requirement: "Real participant selection and consent policy approved under H040-008"
    verified: false
  - requirement: "Operational support and manual-fallback ownership staffed under H040-009"
    verified: false
  - requirement: "Commercial spend ceiling and cloud provider selection confirmed"
    verified: false

# Sole Human Owner Execution Block:
decision_selection: "HOLD"              # Options: [ HOLD | AUTHORIZE_CONTROLLED_ACTIVATION | REJECT ]
decided_by: "UNFILLED"                  # Must be Sole Human Owner
decided_at: "UNFILLED"                  # ISO 8601 UTC Timestamp
signature_or_auth_ref: "UNFILLED"
```
