---
document_id: ARC-V030-EXTROUTE-001
title: v0.3.0 External Route Activation, Cache, Indexing, and Withdrawal Controls Decision Packet (H030-007 HOLD)
document_type: architecture_decision_packet
document_version: 1.0.0
lifecycle_status: DRAFT
status: HELD_PENDING_HUMAN_GATE_H030_007
date: "2026-09-05"
author_role: Architecture and Data Lead
author_pane: w9:p22
governing_issue: "GitHub Issue #100"
governing_decision: HDEC-V030-ENTRY-AND-POLICY-052
milestone: "v0.3.0 - Organization Identity and Portal Alpha"
deferred_human_gates:
  - H030-007
credit_boundary: EVIDENCE_AND_DECISION_PACKET_ONLY_NO_ROUTE_ACTIVATION
---

# v0.3.0 External Route Activation, Cache, Indexing, and Withdrawal Controls Decision Packet (H030-007 HOLD)

## 1. Executive Summary & Governance Declaration

### 1.1 Gate H030-007 HOLD Declaration
In accordance with Sole Human Owner decision `HDEC-V030-ENTRY-AND-POLICY-052` and GitHub Issue #100 (`V030-I027`), **Human Gate `H030-007` remains strictly on `HOLD`**.

- **Zero External Route Activation:** This document does **NOT** activate, provision, bind, or configure any live public DNS record, domain registration, CDN distribution, public IP route, ingress controller, web service listener, or search engine submission.
- **Zero Credential or Provider Action:** Zero cloud provider accounts, API tokens, TLS private keys, payment profiles, or edge provider configurations are instantiated or modified.
- **Decision Packet Purpose:** This document provides the authoritative architectural options, local configuration contract, cache/anti-indexing controls, emergency withdrawal mechanisms, dry-run verification harness, and rollback runbooks for review by the **Sole Human Owner**. Any subsequent activation requires explicit human sign-off transitioning `H030-007` from `HOLD`.

---

## 2. Architectural Delivery Options & Tradeoffs

To deliver sanitized, read-only publication snapshots to external parties (e.g., public inspection verification, regulatory audit packages) without compromising internal infrastructure, three delivery architectures are evaluated:

| Criterion | Option A: Static Pre-Rendered Export (CDN Edge Preview) | Option B: Reverse Proxy Gateway with Scoped Edge Caching | Option C: Bounded Local Staging Route (Internal Loopback Tunnel) |
| :--- | :--- | :--- | :--- |
| **Architecture** | Pre-rendered JSON/HTML bundles pushed to isolated edge bucket behind CDN. | Edge proxy (e.g., Cloudflare / AWS CloudFront / Nginx) routing to internal portal service. | Private loopback / mTLS staging listener without public DNS. |
| **Attack Surface** | **Minimal:** Zero origin exposure; purely static objects served from edge storage. | **Moderate:** Origin server exposed to edge proxy; requires ingress security and WAF. | **Zero Public Surface:** Accessible only via authenticated local/internal network. |
| **Cache Invalidation Latency** | **Fast:** Purge by object path or surrogate key (< 15 seconds). | **Very Fast:** API-driven surrogate tag purge (< 5 seconds) or header revalidation. | **Immediate:** In-memory registry invalidation (0 latency). |
| **Deny-by-Default Isolation** | **Complete:** Only allowlisted and exported snapshots exist on edge storage. | **High:** Ingress gateway enforces route-level authentication and snapshot status checks. | **Complete:** No external connectivity exists. |
| **Operational Complexity** | Low; file upload and edge bucket lifecycle policies. | Moderate; requires ingress TLS, WAF rules, and origin security configuration. | Low for engineering validation; unviable for external non-VPN stakeholders. |
| **Recommended Staging** | **Target Production Architecture (Post-H030-007)** | Secondary / Evaluation Alternative | **Recommended Pre-Activation Testing Baseline** |

### 2.1 Architectural Recommendation
- **Current Alpha Phase (`H030-007 HOLD`):** Maintain **Option C (Bounded Local In-Memory Testing)** as formalized in `modules/publication-snapshot`.
- **Upon Human Authorization of `H030-007`:** Implement **Option A (Static Pre-Rendered Export)** as the primary public route architecture, ensuring physical separation between internal transactional databases and public inspection verification endpoints.

---

## 3. Local Configuration Contract

The following schema formalizes the configuration contract for external publication routes. It is maintained strictly as a local declarative specification pending human approval.

```yaml
# Schema: OSHE-External-Route-Contract-v1.0
schema_version: 1.0.0
contract_id: CTR-EXTROUTE-V030-001
governing_gate: H030-007
gate_status: HOLD
route_profile:
  route_id: route-public-portal-preview-001
  route_name: OSHE Public Inspection Verification Preview
  environment: LOCAL_SYNTHETIC_STAGING
  activation_status: HELD_PENDING_HUMAN_APPROVAL

network_binding:
  allowed_origins:
    - "127.0.0.1:8443" # Local loopback only
  public_hostname_template: "preview-{tenant_id}.oshe.internal" # Synthetic internal only
  canonical_public_domain: TBD_PENDING_H030_007_HUMAN_SELECTION
  enforce_https: true
  min_tls_version: "TLSv1.3"
  hsts_policy:
    enabled: true
    max_age_seconds: 31536000 # 1 year
    include_subdomains: true
    preload: true

security_headers:
  Content-Security-Policy: "default-src 'none'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'; script-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'none';"
  X-Frame-Options: "DENY"
  X-Content-Type-Options: "nosniff"
  Referrer-Policy: "strict-origin-when-cross-origin"
  Permissions-Policy: "camera=(), microphone=(), geolocation=(), payment=()"
  Cross-Origin-Embedder-Policy: "require-corp"
  Cross-Origin-Opener-Policy: "same-origin"
  Cross-Origin-Resource-Policy: "same-origin"

cache_controls:
  default_cache_control: "public, max-age=0, s-maxage=300, must-revalidate, proxy-revalidate, no-transform"
  stale_while_revalidate_seconds: 60
  stale_if_error_seconds: 0 # Never serve stale snapshot on error; fail closed
  surrogate_key_header: "Surrogate-Key"
  surrogate_key_format: "snap_{snapshot_id} ten_{tenant_id} ver_{version}"

indexing_controls:
  robots_tag: "noindex, nofollow, noarchive, nosnippet, noimageindex, notranslate"
  robots_txt_policy: "DISALLOW_ALL"
  search_engine_ping_enabled: false

emergency_controls:
  kill_switch_enabled: false
  kill_switch_response_code: 503
  kill_switch_message: "OSHE Public Portal Preview is temporarily unavailable for administrative maintenance."
```

---

## 4. Cache & Anti-Indexing Controls

### 4.1 Anti-Indexing Guarantee (Search Engine & Scraping Protection)
Public portal previews contain sanitized inspection excerpts and compliance certificates. Under no circumstances should search engine crawlers, web indexers, or AI web scrapers index or cache these pages prematurely.

1. **HTTP Response Header Enforcement:**
   Every HTTP response emitted by the route proxy **must** include:
   ```http
   X-Robots-Tag: noindex, nofollow, noarchive, nosnippet, noimageindex, notranslate
   ```
   - `noindex`: Prevents search engines from indexing the document.
   - `nofollow`: Prevents crawling outbound links.
   - `noarchive`: Prevents caching search engine cache copies.
   - `nosnippet`: Prevents search engine result snippets.
   - `notranslate`: Prevents automated third-party translation proxies.

2. **Root `robots.txt` Disallow-All Rule:**
   The root route must serve an immutable `robots.txt`:
   ```txt
   User-agent: *
   Disallow: /
   ```

### 4.2 Edge & Browser Cache Directives
To balance edge scalability against rapid retraction requirements, cache directives are parameterized as follows:

1. **Short Shared Cache TTL (`s-maxage=300`):**
   - Edge CDN caches snapshot payloads for at most 300 seconds (5 minutes).
   - Browser client caches are restricted to `max-age=0, must-revalidate`, forcing immediate revalidation with the edge proxy on page reload.
2. **Surrogate-Key / Cache-Tag Invalidation:**
   - Every response tags its headers with:
     ```http
     Surrogate-Key: snap_insp_001 ten_alpha ver_1
     ```
   - Enables instant targeted invalidation of specific snapshots or tenant scopes across all edge nodes upon withdrawal or supersession.

---

## 5. Withdrawal & Emergency Retraction Protocol

### 5.1 Snapshot Withdrawal Flow
When an authorized reviewer (`AUDITOR` or `TENANT_ADMIN`) withdraws a snapshot via `modules/publication-snapshot` (`WithdrawSnapshot` / `Withdraw`):

```
[Operational State Transition: WITHDRAWN]
                │
                ▼
[Emit: PublicationWithdrawnEvent]
  - SnapshotID: snap_insp_001
  - TenantID: ten_alpha
  - Reason: "Measurement correction"
                │
                ▼
[Execute Targeted Edge Purge]
  - Command: PurgeKey("snap_insp_001")
  - Propagation SLA: < 15 seconds
                │
                ▼
[Edge Status Transition]
  - Edge Cache: Cleared
  - Subsequent Requests: Return HTTP 410 (Gone) with:
    - X-OSHE-Snapshot-Status: WITHDRAWN
    - X-OSHE-Withdrawal-Reason: "Measurement correction"
```

### 5.2 Emergency Global Kill-Switch Runbook
If an unapproved data exposure or security anomaly is detected:
1. **Trigger:** Set `kill_switch_enabled: true` in route configuration.
2. **Propagation:** Edge CDN configuration updates within 30 seconds.
3. **Behavior:** All incoming requests immediately terminate at the edge with HTTP 503 (Service Unavailable) without forwarding traffic to origin storage.
4. **Purge:** Simultaneously execute global edge purge (`PurgeAll()`) to wipe all cached preview fragments.

---

## 6. Dry-Run Evidence & Verification Plan

Before human authorization of `H030-007`, the following verification scenarios must execute in a local synthetic test harness:

1. **Scenario DRY-01: Header Verification**
   - Send HTTP GET to local preview endpoint.
   - Assert presence and exact values of `X-Robots-Tag`, `Content-Security-Policy`, `X-Frame-Options`, and `Cache-Control`.
2. **Scenario DRY-02: Surrogate Tag Extraction**
   - Send HTTP GET for snapshot `snap_test_01` version 1.
   - Assert `Surrogate-Key` header equals `snap_test_01 ten_test ver_1`.
3. **Scenario DRY-03: Withdrawal Invalidation Verification**
   - Transition snapshot to `WITHDRAWN`.
   - Issue purge request for surrogate key.
   - Immediate repeat GET request returns HTTP 410 Gone (or HTTP 404).
4. **Scenario DRY-04: Kill-Switch Verification**
   - Engage kill-switch.
   - All subsequent requests return HTTP 503.

---

## 7. Rollback Procedures & Reversion Runbook

In the event that an external route is provisioned post-`H030-007` and must be immediately de-provisioned:

```
Step 1: Sever Public DNS Routing
  - Update public DNS records: Delete or point to 127.0.0.1 loopback sinkhole.
  - TTL countdown: Monitor DNS propagation until TTL expires.

Step 2: Disable Edge Distribution
  - CDN Distribution status: Set to DISABLED.
  - Edge Routing: Flush edge caches via global PURGE ALL command.

Step 3: Revoke Origin Ingress
  - Origin Storage Bucket / Gateway: Set bucket policy to PRIVATE (deny all public read).
  - Firewall / Security Group: Remove ingress rules permitting CDN edge IPs.

Step 4: Audit & Lineage Recording
  - Append rollback event into MOD-REC audit journal.
  - Record timestamp, operator, and post-rollback verification proofs.
```

---

## 8. Post-Human-Action Verification Checklist

Prior to and immediately following any future human action to transition Gate `H030-007` from `HOLD` to `APPROVED`:

- [ ] **Authority Verification:** Sole Human Owner explicit written decision recorded in repository decisions (`HDEC-V030-*`).
- [ ] **DNS Ownership:** Public domain registration confirmed in corporate asset inventory; WHOIS privacy enabled.
- [ ] **TLS Certificate Validation:** Valid, un-expired TLS 1.3 certificate issued by trusted public CA; auto-renewal configured.
- [ ] **Header Inspection:** Automated curl test confirms `X-Robots-Tag: noindex...` and `robots.txt: Disallow: /` on production edge.
- [ ] **Cache TTL Audit:** Browser cache confirmed at `max-age=0, must-revalidate`; edge cache confirmed at `s-maxage <= 300`.
- [ ] **Purge API Health:** Automated test confirms edge cache invalidation API credentials function with < 15s latency.
- [ ] **Kill-Switch Test:** Emergency 503 kill-switch verified in staging environment prior to live traffic routing.
- [ ] **No Internal PII / Secret Leakage:** Static snapshot payloads verified against `PublicationFieldAllowlist` with 0 unredacted fields.

---

## 9. Governance Non-Claims Invariant

This document operates strictly as an architectural decision packet and prework artifact under Sole Human Owner decisions `H030-003`, `H030-004`, `H030-005`, and `H030-007`.

- **Zero live public routes, CDN distributions, DNS records, or edge listeners are enacted.**
- **Zero production database schema or SQL mutations are enacted.**
- **Zero customer credentials, billing accounts, or cloud infrastructure settings are altered.**
- **Gate `H030-007` remains permanently on `HOLD` until explicitly transitioned by the Sole Human Owner.**
