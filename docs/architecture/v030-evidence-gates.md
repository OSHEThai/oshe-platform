---
document_id: ARC-V030-EVGATES-001
title: v0.3.0 Deterministic Local Evidence Gates and Traceable Required-Check Matrix
document_type: architecture_specification
document_version: 1.0.0
lifecycle_status: DRAFT
status: APPROVED_FOR_LOCAL_ALPHA_DEVELOPMENT
date: "2026-09-05"
author_role: Architecture and Data Lead
author_pane: w9:p22
governing_issue: "GitHub Issue #108"
governing_decision: HDEC-V030-ENTRY-AND-POLICY-052
milestone: "v0.3.0 - Organization Identity and Portal Alpha"
deferred_human_gates:
  - H030-007
  - H030-008
credit_boundary: DOCUMENTATION_AND_EVIDENCE_GATES_ONLY_NO_RELEASE_OR_HOSTED_ACTIVATION
---

# v0.3.0 Deterministic Local Evidence Gates and Traceable Required-Check Matrix

## 1. Governance Reference, Purpose, and Status Boundary

### 1.1 Governance Reference
This architectural specification establishes the authoritative **Deterministic Local Evidence Gates and Traceable Required-Check Matrix** for **Milestone v0.3.0 - Organization Identity and Portal Alpha** within `OSHEThai/oshe-platform`. It fulfills the requirements and deliverable specifications of **GitHub Issue #108 (`[V030-I035] Define Deterministic Local Evidence Gates and Traceable Required-Check Matrix`)** under the governing authority of Sole Human Owner decision `HDEC-V030-ENTRY-AND-POLICY-052`, `DOC-V030-REL-BOUNDARY-001`, `ARC-V030-AUTHMODEL-001`, and `ARC-V030-PROFILE-NFR-001`.

### 1.2 Purpose and Scope
The purpose of this document is to establish an unyielding, deterministic, reproducible evidence-gate framework that bridges local engineering artifacts, qualification test suites, and operational policies. It establishes unambiguous acceptance criteria across eight core functional domains:
1. **Organization and Tenancy (`MOD-ORG`):** Hierarchy isolation, contractor nesting limits, sponsored participation temporal bounds, and audit preservation.
2. **Identity and Bearer Security (`MOD-IAM`):** Singular synthetic identity truth, high-entropy cryptographic bearer tokens, in-memory SHA-256 digest validation, and session revocation.
3. **Scoped Authorization and Delegation (`MOD-IAM`):** Default-deny policy evaluation, role-based action barriers, direct-object IDOR protection, 1-hop delegation bounding, and segregation-of-duties (SOD) conflict engines.
4. **Internal Portal and Directory Discovery (`MOD-PRT`):** Scoped project directory visibility, contractor discovery boundaries, anti-enumeration empty returns, and data minimization.
5. **Publication Snapshot and Redaction (`MOD-PUB`):** Deny-by-default allowlist redaction, prohibited credential/PII purging, reviewer decision gates, source-change drift isolation, sealed immutability, and export packaging.
6. **Contract Migration and Schema Governance (`MOD-CTR`):** Non-destructive migrations, backward-compatible schema transforms, and referential integrity assertions.
7. **Accessibility and UI Baseline (`A11Y`):** WCAG 2.1 AA contrast compliance, semantic HTML5 landmarks, keyboard navigation focus indicators, and screen reader label bindings.
8. **Release Assurance and Governance (`MOD-REL` / Agent OS):** Agent OS policy conformance, zero unauthorized provider routes, git diff hygiene, and verifiable local CI reproducibility.

### 1.3 Boundary and Non-Claims Invariant
1. **Local-Synthetic Boundary:** All gates evaluate deterministic commands against local synthetic data fixtures and in-memory test harnesses.
2. **Zero Hosted / External Route Activation:** This specification does **NOT** activate live public DNS routes, CDN distributions, hosted GitHub Actions workflows, cloud provider services, or external identity providers.
3. **Deferred Human Authority Reservation:** Human gates **`H030-007` (External Route Activation)** and **`H030-008` (Milestone v0.3.0 Final Release and Tagging)** remain permanently on **`HOLD`**. No claim of a completed release, production deployment, or residual-risk acceptance is made.

---

## 2. Deterministic Evidence Gate Architecture & Lifecycle

### 2.1 Evidence Gate State Machine
Each evidence gate operates as a formal, deterministic state machine with verifiable transition conditions:

```
    ┌──────────────────────┐
    │     UNSATISFIED      │
    └──────────┬───────────┘
               │ Local test/check command passes (100% exit code 0)
               ▼
    ┌──────────────────────┐
    │   QUALIFIED_LOCAL    │
    └──────────┬───────────┘
               │ PR branch clean, ancestry verified, zero diff errors
               ▼
    ┌──────────────────────┐
    │  APPROVED_FOR_MERGE  │
    └──────────┬───────────┘
               │ Requires explicit Sole Human Owner decision (H030-007 / H030-008)
               ▼
    ┌──────────────────────┐
    │   DEFERRED_HUMAN     │
    │   GATE (HOLD/SEALED) │
    └──────────────────────┘
```

### 2.2 Gate Evaluation Invariants
1. **Deterministic Execution:** Every check command must execute locally and produce deterministic results independent of network connectivity or wall-clock jitter (using injectable mock clocks where temporal evaluation is required).
2. **Binary Acceptance:** A gate is either `PASS` (100% assertions satisfied, exit code 0) or `FAIL` (exit code != 0). Zero credit is granted for skipped, quarantined, partially passed, or unverified checks.
3. **Upstream Gate Dependency:** Downstream gates cannot evaluate to `QUALIFIED_LOCAL` if any prerequisite upstream gate is in an `UNSATISFIED` state.
4. **Audit Immutability:** Execution evidence (command logs, test outputs, validation digests) must be preserved in immutable artifacts linked to exact git commit SHAs.

---

## 3. Traceable Required-Check Matrix (The 8 Core Capability Domains)

| Gate ID | Capability Domain | Governed Capabilities | Required Validation Commands | Target Test Fixtures & Code Paths | Acceptance Criteria & Failure Assertions | Governing Decisions & Human Gates |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **`GATE-ORG-01`** | **Organization & Tenancy** (`MOD-ORG`) | 6-level hierarchy, bounded contractor relationships, sponsored participation, and audit preservation | `cd modules/organization-tenancy; go test -v ./...` | `party.go`, `party_test.go`, `organization_lifecycle.go`, `organization_hierarchy.go` | - 100% tests PASS.<br>- Reversible transitions verify memory safety.<br>- Contractor nesting ceiling enforced (`MaxContractorNestingDepth = 1`).<br>- Cross-tenant moves and un-sponsored contractor additions fail closed (`ErrCrossTenantLinkage`). | `H030-002`, `H030-008` (HOLD) |
| **`GATE-ID-01`** | **Identity & Bearer Security** (`MOD-IAM`) | Singular synthetic identity truth, high-entropy bearer tokens, SHA-256 digest validation, session revocation | `cd modules/identity-authorization; go test -v -run "TestIdentity|TestToken|TestValidateSession|TestRevocationRegistry" ./...` | `local_identity.go`, `local_identity_test.go`, `session_revocation.go` | - 100% tests PASS.<br>- Raw tokens formatted `oshe_tok_<64-hex>` with 32-byte entropy.<br>- Stored exclusively as SHA-256 digests; raw tokens never persisted.<br>- Expired or revoked sessions fail closed (`ErrTokenExpired`, `ErrTokenRevoked`).<br>- Subject-level revocation invalidates all active sessions. | `H030-004`, `H030-008` (HOLD) |
| **`GATE-AUTH-01`** | **Scoped Authorization & Delegation** (`MOD-IAM`) | Default-deny policy evaluation, least-privilege role bounds, IDOR defense, 1-hop delegation bounds, SOD conflict engine | `cd modules/identity-authorization; go test -v -run "TestPolicy|TestAuthorizationMatrix|TestScopedAssignment|TestDelegation|TestQualification" ./...` | `access_policy.go`, `authorization_matrix.go`, `authorization_qualification_test.go`, `negative_controls_test.go` | - 100% tests PASS.<br>- Unauthenticated calls fail (`DenialUnauthenticated`).<br>- Sibling project access rejected (`DenialScopeMismatch`).<br>- 1-hop delegation ceiling strictly enforced (`ErrMultiHopDelegationForbidden`).<br>- SOD conflicts detected (`SOD-01`, `SOD-02`, `SOD-02B`).<br>- Emergency break-glass override fails closed (`ErrEmergencyAccessDenied`). | `H030-003`, `H030-008` (HOLD) |
| **`GATE-PORTAL-01`**| **Internal Portal & Directory Discovery** (`MOD-PRT`) | Scoped directory visibility, contractor boundaries, anti-enumeration empty returns, profile data minimization | `cd modules/internal-portal; go test -v ./...` | `portal.go`, `portal_test.go`, `negative_controls_test.go` | - 100% tests PASS.<br>- Directory queries partition strictly to caller project.<br>- Cross-project searches return empty slices (`[]`), preventing enumeration.<br>- Contractor access restricted strictly to assigned projects.<br>- Data minimization asserts 0 passwords, 0 bearer tokens, 0 national IDs. | `H030-005`, `H030-008` (HOLD) |
| **`GATE-PUB-01`** | **Publication Snapshot & Redaction** (`MOD-PUB`) | Deny-by-default allowlist redaction, prohibited data purge, reviewer gates, source drift isolation, sealed immutability | `cd modules/publication-snapshot; go test -v ./...` | `publication_snapshot.go`, `publication_lifecycle.go`, `immutable_publication.go`, `publication_qualification_test.go` | - 100% tests PASS (43/43).<br>- Unallowlisted fields stripped in permissive, rejected in strict mode.<br>- Prohibited credentials/PII fail closed (`ErrProhibitedFieldDetected`).<br>- Stale approvals (> 7 days) rejected (`ErrStaleApproval`).<br>- Published snapshots permanently immutable (`ErrPublicationVersionImmutable`).<br>- Direct source mutation rejected (`ErrDirectSourceMutationForbidden`).<br>- Audit reconstruction verifies unbroken chain (`StatusVerifiedIntact`). | `H030-003`, `H030-007` (HOLD), `H030-008` (HOLD) |
| **`GATE-CTR-01`** | **Contract Migration & Schema Governance** (`MOD-CTR`) | Non-destructive schema evolution, backward-compatible API models, referential integrity | `cd contracts/api; go test -v ./...` | `party_contract.go`, `party_contract_test.go` | - 100% tests PASS.<br>- API contracts assert redacted party representations.<br>- Prohibited field patterns rejected in serialized payloads.<br>- Non-destructive data serialization verified. | `H030-002`, `H030-008` (HOLD) |
| **`GATE-A11Y-01`** | **Accessibility & UI Engineering Baseline** (`A11Y`) | WCAG 2.1 AA color contrast (>= 4.5:1 text, >= 3:1 UI), semantic HTML landmarks, keyboard focus rings, ARIA labels | Visual/DevTools audit protocol and contract tests | `docs/architecture/v0.3.0-supported-identity-and-portal-profile.md` | - Text contrast >= 4.5:1; interactive components >= 3:1.<br>- Distinct visible focus ring on all focusable elements.<br>- Zero auto-playing media, zero flashing content.<br>- All form inputs associated with explicit `<label>` tags.<br>- Screen reader accessibility verified across simulated tree. | `H030-008` (HOLD) |
| **`GATE-REL-01`** | **Release Assurance & Governance Baseline** (`MOD-REL` / Agent OS) | Agent OS policy conformance, zero live provider routes, clean git whitespace, reproducible local CI | `python -B .ai/tools/validate_agent_os.py` & `git diff --check` | `.ai/tools/validate_agent_os.py`, `.ai/policies/budgets.yaml`, `.ai/provider-routes/` | - Agent OS validation PASSED (130 files parsed/checked).<br>- Exactly 0 provider routes enabled (`provider_routes_enabled = 0`).<br>- `git diff --check` clean (0 whitespace errors).<br>- All changed files confined strictly to authorized lease scope. | `H030-007` (HOLD), `H030-008` (HOLD) |

---

## 4. Negative Controls & Anti-Bypass Catalog

To prevent regression, privilege escalation, or unauthorized data exposure, the following negative controls must be maintained and verified programmatically:

| Negative Control ID | Threat / Attack Scenario | Hostile Input / Malicious Action | Fail-Closed Mechanism & Error Code | Governed Module |
| :--- | :--- | :--- | :--- | :--- |
| **`NEG-ORG-01`** | Contractor nesting hierarchy bypass | Subcontractor attempts to sponsor tier-3 contractor (`depth >= 2`) | Rejected with `ErrNestingDepthExceeded` | `modules/organization-tenancy` |
| **`NEG-ORG-02`** | Cross-tenant entity attachment | Entity in Tenant A attempts to link project or party in Tenant B | Rejected with `ErrCrossTenantLinkage` | `modules/organization-tenancy` |
| **`NEG-IAM-01`** | Credential / bearer token leakage | Serializing raw session tokens or storing passwords in directory | Memory-only token hashing; raw token persistence forbidden | `modules/identity-authorization` |
| **`NEG-IAM-02`** | IDOR direct-object scope mismatch | Authorized subject attempts to access object in sibling project | Evaluator fails with `DenialScopeMismatch` / `DenialDirectObjectMismatch` | `modules/identity-authorization` |
| **`NEG-AUTH-01`** | Multi-hop delegation escalation | Delegatee attempts to re-delegate authority to third party | Matrix rejects sub-delegation with `ErrMultiHopDelegationForbidden` | `modules/identity-authorization` |
| **`NEG-AUTH-02`** | Sovereign role delegation | Tenant administrator attempts to delegate `RoleTenantAdmin` | Matrix rejects with `ErrProtectedAuthorityNonDelegable` | `modules/identity-authorization` |
| **`NEG-AUTH-03`** | Emergency break-glass bypass | Subject asserts emergency override or break-glass privilege | Evaluator fails with `ErrEmergencyAccessDenied` | `modules/identity-authorization` |
| **`NEG-AUTH-04`** | Segregation of Duties (SOD) conflict | Assigning concurrent conflicting roles (Inspector + Auditor) | Matrix rejects with `ErrRoleConflictDetected` | `modules/identity-authorization` |
| **`NEG-PRT-01`** | Directory existence enumeration | Caller queries project roster outside active authorized scope | Registry returns empty slice (`[]`), preventing existence oracle | `modules/internal-portal` |
| **`NEG-PRT-02`** | PII leakage in directory profile | Injecting national ID or personal phone number into profile | Sanitizer and allowlist strip unapproved fields | `modules/internal-portal` |
| **`NEG-PUB-01`** | Sensitive data in published snapshot | Injecting passwords, tokens, or national IDs into publication | Redactor fails closed with `ErrProhibitedFieldDetected` | `modules/publication-snapshot` |
| **`NEG-PUB-02`** | Stale / tampered reviewer approval | Publishing snapshot with approval > 7 days old or altered digest | Lifecycle controller fails with `ErrStaleApproval` / `ErrApprovalDigestMismatch` | `modules/publication-snapshot` |
| **`NEG-PUB-03`** | Operational source-drift mutation | Updating operational database to alter published snapshot | Published record sealed; direct mutation rejected with `ErrDirectSourceMutationForbidden` | `modules/publication-snapshot` |
| **`NEG-PUB-04`** | Lineage tampering in version chain | Manipulating predecessor hash in multi-version chain | Audit reconstruction reports `StatusTamperDetected` | `modules/publication-snapshot` |
| **`NEG-EXT-01`** | Search engine indexing of preview | Web crawlers indexing public snapshot previews | Mandatory `X-Robots-Tag: noindex...` and `robots.txt: Disallow: /` | `docs/architecture` / Route Packet |

---

## 5. Local Verification Automation & Local CI Contract

### 5.1 Local CI Configuration (`.ci/local-ci.json`)
Local continuous integration is configured to execute deterministic test suites prior to any branch push or pull request submission:
```json
{
  "schema_version": "1.0.0",
  "repository_kind": "platform",
  "checks": [
    {
      "id": "agent-os-regression",
      "command": [
        "python",
        "-m",
        "unittest",
        "tests.test_validate_agent_os",
        "tests.test_permission_delegation",
        "tests.test_local_ci",
        "tests.test_supply_chain",
        "tests.test_evidence_bundle",
        "tests.test_reference_benchmark",
        "tests.test_mission_rehearsal",
        "tests.test_api_contracts",
        "tests.test_recovery_qualification"
      ]
    },
    {
      "id": "foundation-validation",
      "command": [
        "python",
        "tools/validate_repository.py",
        "--repo-kind",
        "platform"
      ]
    },
    {
      "id": "supply-chain-verification",
      "command": [
        "python",
        "tools/supply_chain.py",
        "--verify"
      ]
    }
  ]
}
```

### 5.2 Deterministic Execution Runbook
For every task and release candidate, the following local verification runbook must execute in sequence:
1. **Module Unit Tests:**
   ```powershell
   cd modules/organization-tenancy; go test -v ./...
   cd modules/identity-authorization; go test -v ./...
   cd modules/internal-portal; go test -v ./...
   cd modules/publication-snapshot; go test -v ./...
   ```
2. **Agent OS Governance Validation:**
   ```powershell
   python -B .ai/tools/validate_agent_os.py
   ```
3. **Repository Validation & Local CI:**
   ```powershell
   python tools/validate_repository.py --repo-kind platform
   ```
4. **Diff Hygiene & Path Confinement:**
   ```powershell
   git diff --check <base-sha> HEAD
   git diff --name-only <base-sha> HEAD
   ```

---

## 6. Deferred Human Gates & Sovereign Authority (`H030-007`, `H030-008`)

The evidence gates established in this document provide complete local qualification. However, final operational deployment, public route activation, and release authority remain exclusively reserved for the **Sole Human Owner**:

### 6.1 Human Gate H030-007 (External Route Activation)
- **Status:** **`HOLD`**
- **Reserved Human Actions:**
  - Selection and binding of production public domains and DNS zones.
  - Deployment of CDN edge proxies and production TLS certificates.
  - Activation of internet-accessible publication endpoints.
- **AI Agent Prohibition:** AI agents are strictly prohibited from provisioning cloud DNS records, purchasing domains, configuring live CDN edge routes, or creating external network listeners.

### 6.2 Human Gate H030-008 (Milestone v0.3.0 Release Approval)
- **Status:** **`HOLD`**
- **Reserved Human Actions:**
  - Final milestone acceptance and sign-off.
  - Creation of production release git tags (`v0.3.0`).
  - Formal publication of release notes and deployment packages.
- **AI Agent Prohibition:** AI agents are strictly prohibited from approving releases, tagging releases, or accepting residual operational risk.

---

## 7. Governance Non-Claims Invariant

This specification operates strictly as an architectural specification and evidence-gate definition under Sole Human Owner decisions `HDEC-V030-ENTRY-AND-POLICY-052`, `H030-002`, `H030-003`, `H030-004`, and `H030-005`.

- **Zero release recommendation is made.**
- **Zero release approval is granted.**
- **Zero live public routes or CDN endpoints are provisioned.**
- **Zero production database persistence or customer data mutations are enacted.**
- **All human gates (`H030-007`, `H030-008`) remain strictly on `HOLD`.**
