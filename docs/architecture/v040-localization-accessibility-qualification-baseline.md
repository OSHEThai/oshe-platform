---
document_id: QLF-V040-LOC-A11Y-001
title: v0.4.0 Localization, Time-Zone Formatting, and Feature Accessibility Qualification Baseline
governing_issue: 142
assignment_id: ASN-V040-I031-LOCALIZATION-QUALIFICATION-003
lease_id: LEASE-V040-I031-LOCALIZATION-QUALIFICATION-003
status: APPROVED
lifecycle: APPROVED
target_milestone: v0.4.0
synthetic_scenario_id: fix_syn_localization_accessibility_qualification_v1
governing_decisions:
  - HDEC-V040-FOUNDATION-054
  - HDEC-V040-SCORING-058
deferred_decisions:
  - H040-002
  - H040-006
retained_holds:
  - H040-007
  - H040-008
  - H040-009
  - H040-010
  - H040-011
---

# v0.4.0 Localization, Time-Zone Formatting, and Feature Accessibility Qualification Baseline

## 1. Executive Summary & Purpose

This document establishes the governed qualification baseline for internationalization, localized date/time presentation, unit representation, text expansion tolerances, default-off feature toggling, and accessibility metadata contracts for Milestone `v0.4.0 OSHE Inspect Private Alpha` under Issue #142 (`V040-I031`).

### Explicit Non-Claims & Deferred Decisions Declaration
- **No Browser, UI, or Mobile Device Runtime Claim:** This qualification baseline establishes synthetic and algorithmic contract verification only. It makes no browser, UI, or mobile device runtime claim.
- **Deferred Human Decisions:** Final sovereign selections for client language/locale coverage (`H040-002`) and comprehensive accessibility/UI design selections (`H040-006`) remain **deferred human-owned decisions** under Sole Human Owner governance.
- **Zero Production or Customer Data:** All qualification fixtures operate strictly with synthetic identifiers (`usr_*`, `ten_*`, `flag_*`).
- **Retained Foundation Holds:** Foundation holds `H040-007` through `H040-011` remain strictly on `HOLD`. Zero authority is granted to alter or lift these holds.

---

## 2. Core Qualification Invariants

### Invariant 1: Supported BCP-47 Locales (`en-US` and `th-TH`)
- The platform defines two supported synthetic test locales: `en-US` (American English, default fallback) and `th-TH` (Central Thai).
- Translations resolved directly from the target locale bundle return `DispositionExactMatch` with `FallbackUsed = false`.

### Invariant 2: Visible Fallback for Missing Translations
- When a translation key is requested in `th-TH` but is unpopulated, the catalog falls open to `en-US` with full visibility:
  - `Disposition = DispositionFallback`
  - `FallbackUsed = true`
  - `MissingNotice` explicitly details the key and fallback path (e.g. `key "..." not found in th-TH; fallen back to en-US`).
- Silent fallback, unlogged substitution, or unexpected empty strings are prohibited.

### Invariant 3: Visible Missing Indicator
- When a translation key is missing from both the target locale and the fallback locale:
  - `Disposition = DispositionMissing`
  - `Text = "[MISSING: <key>]"`
  - `MissingNotice` details the missing key across both locales.
- The synthetic qualification contract explicitly exposes missing translations via visible placeholders rather than masking them, with no runtime UI or API implementation implied.

### Invariant 4: Dual Time-Zone and Buddhist Era Formatting
- Timestamps are formatted deterministically for UTC and `Asia/Bangkok` (UTC+7).
- For `th-TH` in `Asia/Bangkok`, dates use the Buddhist Era (BE) standard:
  $$\text{Year}_{\text{BE}} = \text{Year}_{\text{CE}} + 543$$
- For `en-US` in UTC, dates use standard Gregorian CE notation.
- Unrecognized or malformed time zone identifiers reject with `ErrInvalidTimeZone`.

### Invariant 5: Localized Units and Text Expansion Tolerances
- Metric and physical units format deterministically:
  - Celsius: `°C`
  - Length: `m` (`en-US`) vs `ม.` (`th-TH`)
  - Weight: `kg` (`en-US`) vs `กก.` (`th-TH`)
  - Sound pressure: `dB` (`en-US`) vs `เดซิเบล` (`th-TH`)
- Long technical observations verify that Thai script expansion (ratio typically between $1.0$ and $1.5\times$) is accommodated without string truncation or byte corruption.

### Invariant 6: Default-Off Feature Toggle Invariant
- All governed feature flags must explicitly declare `DefaultOff: true`. Registering a flag with `DefaultOff: false` fails closed with `ErrMustDefaultOff`.
- A disabled flag is not exposed (evaluates to `Exposed = false` with safe fallback active), and registration with `DefaultOff = false` fails closed with `ErrMustDefaultOff`.

### Invariant 7: Strict Authorization Separation
- Feature flags govern client qualification exposure only (with no runtime UI or API implementation implied) and **never grant security authority or bypass authorization**:
  $$\text{ctx.IsAuthorized} = \text{false} \implies \text{res.Exposed} = \text{false}$$
- If a caller lacks authorization, the flag evaluation returns `Exposed = false` with reason `"underlying authorization denied: flags cannot bypass security controls"`.
- Every flag evaluation result includes the mandatory disclaimer:
  `FEATURE_FLAG_NON_AUTHORITY: Feature flags control client exposure and operational gates only. Flags never grant security authority or bypass authorization controls.`

### Invariant 8: Accessibility Metadata Attestation
- Feature flag definitions require structured accessibility metadata:
  - `KeyboardNavigable` (boolean)
  - `ScreenReaderLabel` (non-empty descriptive label)
  - `AriaRole` (valid ARIA widget/landmark role)
  - `ContrastCertified` (boolean certification placeholder)
- Evaluation results propagate accessibility metadata alongside exposure decisions.

---

## 3. Synthetic Qualification Scenario Matrix (`fix_syn_localization_accessibility_qualification_v1`)

| Scenario ID | Test Domain | Scenario Description | Expected Outcome | Governed Invariant |
|---|---|---|---|---|
| `SYN-LOC-01` | Localization | Exact match resolution in `en-US` and `th-TH` | `DispositionExactMatch`, `FallbackUsed = false` | Invariant 1 |
| `SYN-LOC-02` | Fallback | Key present in `en-US` but missing in `th-TH` | `DispositionFallback`, `FallbackUsed = true`, visible notice | Invariant 2 |
| `SYN-LOC-03` | Missing Key | Key missing in both `en-US` and `th-TH` | `DispositionMissing`, `[MISSING: key]`, visible notice | Invariant 3 |
| `SYN-LOC-04` | Time-Zone & BE | Date formatting in `Asia/Bangkok` for `th-TH` | Buddhist Era year ($2026 + 543 = 2569$), UTC+7 offset | Invariant 4 |
| `SYN-LOC-05` | Units & Numbers | Number formatting and localized unit labels (`ม.`, `กก.`, `เดซิเบล`) | Formatted comma separators and correct Thai abbreviations | Invariant 5 |
| `SYN-LOC-06` | Text Expansion | Long technical text expansion ratio between EN and TH | Preserved without truncation, ratio within bounds ($0.5$–$2.5$) | Invariant 5 |
| `SYN-LOC-07` | Feature Flags | Attempt registration with `DefaultOff = false` | Rejected with `ErrMustDefaultOff` | Invariant 6 |
| `SYN-LOC-08` | Auth Separation | Flag enabled but `ctx.IsAuthorized = false` | `Exposed = false`, reason: authorization denied | Invariant 7 |
| `SYN-LOC-09` | Accessibility | Flag evaluation with accessibility metadata | Complete `AccessibilityMetadata` propagated in result | Invariant 8 |
| `SYN-LOC-10` | Rollout Boundaries | Temporal bounds and fractional cohort hashing | Correct exclusion before `EffectiveFrom` and after `EffectiveTo` | Invariant 6, 7 |

---

## 4. Retained Foundation Holds & Deferred Human Decisions

### Deferred Human Decisions
- **`H040-002` (Locale and Language Final Selection):** Language pack finalization, legal review of translations, and localized font assets remain deferred human decisions.
- **`H040-006` (Accessibility and UI Design Final Selection):** WCAG 2.1 AA binding compliance audit, screen-reader device certification, and high-contrast color scheme signoff remain deferred human decisions.

### Retained Foundation Holds
Under **HDEC-V040-FOUNDATION-054**, the following foundation holds remain in active **`HOLD`** status:

| Hold ID | Governance Subject | Status | Enforcement Constraint |
|---|---|---|---|
| `H040-007` | Technical release authorization | **HOLD** | No technical release authorization is granted. |
| `H040-008` | Real participant, private-alpha, and UAT engagement | **HOLD** | No real participant, private-alpha, or UAT engagement authorization is granted. |
| `H040-009` | Binding support and manual-fallback operational ownership | **HOLD** | No binding support or manual-fallback operational ownership authorization is granted. |
| `H040-010` | External environment, device, account, route, storage, and notification activation | **HOLD** | No external environment, device, account, route, storage, or notification activation is granted. |
| `H040-011` | Final outcome, residual-risk acceptance, and v0.5.0 entry decision | **HOLD** | No final outcome, residual-risk acceptance, or v0.5.0 entry decision authorization is granted. |

Zero authority is granted to lift any hold. No claim is made beyond the listed synthetic localization and accessibility fixtures; this qualification suite does not assert audit-record immutability.
