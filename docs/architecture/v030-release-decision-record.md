---
document_id: ARC-V030-DECREC-001
document_type: architecture_decision_record
lifecycle_status: APPROVED
authority_source: HDEC-V030-RELEASE-053
governing_issue: GitHub Issue #111
governing_gate: H030-008
decided_by: Sole Human Owner
decided_at: 2026-09-05T13:51:37Z
selected_option: OPTION_1_FULL_APPROVAL_AND_MILESTONE_CLOSURE
retained_holds:
  - H030-007
credit_boundary: NO_RELEASE_OR_TAGGING_CREDIT
---

# v0.3.0 Sole Human Owner Release Decision Record

| Metadata Field | Value |
| :--- | :--- |
| **Document ID** | `ARC-V030-DECREC-001` |
| **Document Type** | Architecture Decision Record (Milestone Release Authority) |
| **Lifecycle Status** | `APPROVED` |
| **Authority Source** | `HDEC-V030-RELEASE-053` |
| **Governing Issue** | GitHub Issue #111 (`V030-I038`) |
| **Governing Gate** | `H030-008` (Release Decision Gate - Milestone v0.3.0) |
| **Decided By** | Sole Human Owner |
| **Decided At** | `2026-09-05T13:51:37Z` |
| **Selected Option** | `OPTION_1_FULL_APPROVAL_AND_MILESTONE_CLOSURE` |
| **Retained Holds** | `H030-007` (`HOLD_UNTIL_SUPERSEDED`) |
| **Credit Boundary** | Decision record capture only (`NO_RELEASE_OR_TAGGING_CREDIT`) |

---

## 1. Executive Summary & Context

This document constitutes the authoritative Architecture Decision Record executing Sole Human Owner decision `HDEC-V030-RELEASE-053` in response to the formal evaluation presented in the [v0.3.0 Sole Human Owner Release Decision Packet](v030-human-release-decision-packet.md) (`ARC-V030-DECPKT-001`).

On 2026-09-05 at 13:51:37 UTC, the Sole Human Owner formally selected **Option 1 (`OPTION_1_FULL_APPROVAL_AND_MILESTONE_CLOSURE`)**, approving the completion and closure of Milestone v0.3.0 (*Organization Identity and Portal Alpha*).

---

## 2. Approved Authorizations (Option 1 Execution)

Under `HDEC-V030-RELEASE-053`, the Sole Human Owner has authorized the following operational actions:

1. **Milestone v0.3.0 Closure:** Milestone v0.3.0 achieves formal release status upon the clean merge and independent review readback of this decision record.
2. **Annotated Git Tag Creation:** Creation of an annotated git tag `v0.3.0` pointing to the exact commit where this decision record (`ARC-V030-DECREC-001`) is merged into `main`.
3. **GitHub Release Publication:** Creation and publication of a formal GitHub Release for `v0.3.0` documenting delivery of the milestone's synthetic prework architecture.
4. **Milestone v0.4.0 Planning:** Formal transition of engineering focus to Milestone v0.4.0 (*OSHE Inspect Private Alpha*) planning, scoping, and foundational design.

---

## 3. Retained Holds & Invariant Non-Claims

While Milestone v0.3.0 closure is approved, the Sole Human Owner explicitly reaffirmed key boundary constraints and retained governance holds:

### Retained Gate Hold: H030-007
- **Status:** **`HOLD_UNTIL_SUPERSEDED`**
- **Boundary:** Gate `H030-007` (Public Network Routes, DNS & CDN Distribution) remains on strict **HOLD** until explicitly superseded by a future sovereign human decision.

### Strict Prohibitions & Non-Claims
Neither this decision record nor `HDEC-V030-RELEASE-053` authorizes any of the following actions:
- **Zero External Public Routes:** No public internet DNS records, reverse proxies, HTTP/HTTPS ingress routes, or public endpoints may be activated.
- **Zero CDN Edge Deployment:** No Cloudflare workers, Fastly, AWS CloudFront, or intermediate proxy caching distribution networks may be provisioned.
- **Zero Production Database Deployment:** No production relational database schemas, tables, or persistence engines may be deployed. State remains local and in-memory.
- **Zero Real Customer or Personal Data:** All data models remain local synthetic fixtures (`usr_*`, `prj_*`, `ten_*`, `snp_*`). Real PII, workforce personal details, and biometric records remain categorically excluded.
- **Zero Provider, Credential, or Account Mutations:** No external identity providers (IdP), SAML/OIDC integrations, production cryptographic keys, or payment provider credentials may be provisioned or mutated.
- **Mandatory Local-Synthetic Boundary Statement:** All release notes and metadata associated with `v0.3.0` must prominently state the local-synthetic boundary and retained `H030-007` hold.

---

## 4. Evidence Basis & Precondition Verification

This approval is supported by the verified evidence recorded in the repository baseline:

1. **Full Engineering Coverage:** All 26 planned engineering issues (`V030-I011` through `V030-I036`) are implemented, tested, reviewed, and merged on `origin/main`.
2. **Deterministic Qualification Test Suites:** 100% of required local unit, integration, and qualification test suites pass cleanly across identity, authorization, publication snapshots, public portal shielding, and the walking skeleton harness.
3. **Static Governance Conformance:** `python -B .ai/tools/validate_agent_os.py` passes cleanly across all 130 governed policy, role, profile, and schema definitions with exactly 0 provider routes enabled.
4. **Whitespace & Git Hygiene:** `git diff --check` passes cleanly with zero formatting defects across all commits.
5. **Exception Dispositions:** All three milestone exceptions (`EXC-V030-001`, `EXC-V030-002`, `EXC-V030-003`) remain properly handled through non-adoption boundaries and clean successor lineage.

---

## 5. Execution Sequence & Operational Posture

The standing protocol for finalizing Milestone v0.3.0 proceeds strictly in the following sequence:

1. **Decision Record Ingestion:** Author and commit `ARC-V030-DECREC-001` (this document) on a short-lived feature branch with dedicated qualification tests.
2. **Draft PR & CI Verification:** Open a non-closing draft pull request, verify passing hosted CI, and complete independent quality review.
3. **Merge & Readback:** Non-force merge this decision record to `main` and execute formal readback to the PM Secretary (`w9:p21`).
4. **Subsequent Authorized Actions:** Execution of the annotated git tag `v0.3.0` and GitHub Release creation shall occur under dedicated, authorized successor leases, ensuring full separation of authority.
