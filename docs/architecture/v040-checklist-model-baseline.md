---
document_id: ARC-V040-CHKL-001
title: v0.4.0 OSHE Inspect Bounded Checklist Model, Sections, Question Types, Applicability, and Translation Baseline
document_type: architecture_specification
document_version: 1.0.0
lifecycle_status: DRAFT
status: APPROVED_FOR_LOCAL_DEVELOPMENT
date: "2026-09-05"
author_role: Engineering Lead
author_pane: w9:p23
governing_issue: "GitHub Issue #116"
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
credit_boundary: CHECKLIST_MODEL_SPECIFICATION_ONLY_NO_NOCODE_PLATFORM_OR_BINDING_SCORING
---

# v0.4.0 OSHE Inspect Bounded Checklist Model, Sections, Question Types, Applicability, and Translation Baseline

## 1. Executive Summary & Governance Reference

### 1.1 Governance Reference & Purpose
This architectural specification establishes the authoritative **Bounded Checklist Model, Sections, Question Types, Applicability, Scoring Reference, and Translation Baseline** for **Milestone v0.4.0 - OSHE Inspect Private Alpha** within `OSHEThai/oshe-platform`. It fulfills the requirements and deliverable specifications of **GitHub Issue #116 (`[V040-I005] Define Checklist Model, Sections, Question Types, and Applicability`)** under Roadmap Topic `V040-T01` and standing owner foundation decision `HDEC-V040-FOUNDATION-054`.

The primary purpose is to define an integrated, dependency-free, bounded checklist template model across identity, versioning, section hierarchy, supported alpha question types, evidence triggers, localization bindings, and synthetic fixtures for the standalone single-tenant private alpha vertical slice (`H040-001`).

### 1.2 Retained Unselected Policies & Non-Claims
In strict conformance with `ASN-V040-I005-CHECKLIST-MODEL-001` and `HDEC-V040-FOUNDATION-054`:
1. **Binding Scoring Policy (`HUMAN_OWNED_UNSELECTED`):** All scoring parameters, category weights, passing thresholds, and critical-fail triggers defined in this specification are descriptive, provisional reference models only. Final binding scoring policy requires an explicit human owner decision under Issue #136 (`V040-I025`).
2. **Finding Closure Policy (`HUMAN_OWNED_UNSELECTED`):** Finding resolution verification, corrective action sign-off, and residual safety risk acceptance remain human-owned under Issue #134 (`V040-I023`).
3. **Offline Authority Model (`HUMAN_OWNED_UNSELECTED`):** Client-side offline cache limits, write-ahead journal expiry, and manual reconciliation workflows remain human-owned under Issue #126 (`V040-I015`).
4. **No Generalized No-Code Platform:** This specification deliberately rejects and excludes a generalized no-code form builder, Turing-complete expression evaluator, dynamic script execution, or arbitrary widget layout engine.
5. **Retained Human Gate Holds:** Foundation gates `H040-007` (Technical Release Authorization), `H040-008` (Real Participant / Private-Alpha UAT Onboarding), `H040-009` (Binding Support Ownership), `H040-010` (External Environment/Route Activation), and `H040-011` (Final Milestone Outcome) remain strictly on **`HOLD`**.

---

## 2. Checklist Template Model & Stable Identity (`MOD-CFG`)

Under the module data authority baseline (`ARC-V040-DOMAIN-001`), the **Configuration and Checklist Module (`MOD-CFG`)** is the sole source of truth for checklist templates, sections, questions, and publication lifecycles.

### 2.1 Entity Identity & Versioning Model

```
+----------------------------------------------------------------------------------------------------+
| CHECKLIST TEMPLATE ENTITY (chk_syn_*)                                                              |
|                                                                                                    |
|  - template_id: String (chk_syn_[a-z0-9_]{8,32})       - owner_role: "Checklist Author"            |
|  - tenant_id: String (ten_[a-z0-9_]{8,32})             - lifecycle_status: LifecycleStatus         |
|  - version: String (SemVer "major.minor.patch")        - effective_from: ISO-8601 UTC Timestamp    |
|  - predecessor_version: String (nullable)              - effective_to: ISO-8601 UTC (nullable)     |
|  - title: LocalizedStringMap (en-US, th-TH)            - published_at: ISO-8601 UTC (nullable)     |
|  - description: LocalizedStringMap (en-US, th-TH)      - content_digest: SHA-256 Checksum String   |
|  - category: String (e.g. "PLANT_SAFETY", "EHS")       - is_pilot: Boolean (true for Alpha)        |
+----------------------------------------------------------------------------------------------------+
       │ 1
       │
       ▼ *
+----------------------------------------------------------------------------------------------------+
| SECTION ENTITY (sec_syn_*)                                                                         |
|                                                                                                    |
|  - section_id: String (sec_syn_[a-z0-9_]{8,32})        - display_order: Integer (1-based)          |
|  - title: LocalizedStringMap (en-US, th-TH)            - applicability_rules: []ApplicabilityRule  |
|  - description: LocalizedStringMap (en-US, th-TH)      - provisional_weight: Float (0.0 - 100.0)   |
+----------------------------------------------------------------------------------------------------+
       │ 1
       │
       ▼ *
+----------------------------------------------------------------------------------------------------+
| QUESTION ENTITY (qst_syn_*)                                                                        |
|                                                                                                    |
|  - question_id: String (qst_syn_[a-z0-9_]{8,32})       - evidence_policy: EvidencePolicy           |
|  - code: String (e.g. "SAF-01", "PPE-02")              - criticality: CriticalityLevel             |
|  - prompt: LocalizedStringMap (en-US, th-TH)           - scoring_ref: ProvisionalScoringRef        |
|  - guidance: LocalizedStringMap (en-US, th-TH)         - applicability_rules: []ApplicabilityRule  |
|  - question_type: AlphaQuestionType                    - display_order: Integer (1-based)          |
|  - required: Boolean                                   - options: []QuestionOption (for choice)    |
|  - numeric_spec: NumericSpec (for measurement)         - is_archived: Boolean                      |
+----------------------------------------------------------------------------------------------------+
```

### 2.2 Template Lifecycle State Machine
Checklist templates progress through a deterministic, irreversible-when-published lifecycle:

```
┌─────────┐      Submit Review      ┌──────────────┐      Approve      ┌────────────┐
│  DRAFT  │ ──────────────────────> │ UNDER_REVIEW │ ────────────────> │  APPROVED  │
└─────────┘                         └──────────────┘                   └────────────┘
     ▲                                     │ Reject                          │ Publish
     │ Revise                              ▼                                 ▼
     └───────────────────────────── [ REJECTED ]                       ┌───────────┐
                                                                       │ PUBLISHED │ (IMMUTABLE)
                                                                       └─────┬─────┘
                                                                             │ Supersede
                                                                             ▼
                                                                       ┌───────────┐
                                                                       │  RETIRED  │
                                                                       └───────────┘
```

#### Lifecycle State Invariants
1. **`DRAFT`:** Authoring state. The `Checklist Author` can add, remove, and mutate sections, questions, applicability rules, and translation maps.
2. **`UNDER_REVIEW`:** Content freeze. The template is locked for modification while under review by an authorized reviewer.
3. **`APPROVED`:** Formal sign-off granted by reviewer. The template is validated against schema constraints and awaits publication scheduling.
4. **`PUBLISHED`:** **Immutable published-version boundary.** Upon publication, the checklist template is cryptographically sealed (SHA-256 `content_digest`). It can **never** be edited, deleted, or altered in place. Operational inspections instantiate immutable bindings to this published version.
5. **`RETIRED` / `SUPERSEDED`:** Historical reference state. Existing in-flight inspections complete against this version; new inspections cannot instantiate retired templates.

---

## 3. Supported Alpha Question Types (`H040-006`)

In accordance with `H040-006` and private-alpha boundary constraints, the question model is restricted exclusively to **six supported alpha question types**. Any question definition outside this catalog is rejected during validation:

| Type Identifier | Description & Behavioral Semantics | Permitted Response Values | Supported Triggers / Flags |
| :--- | :--- | :--- | :--- |
| **`PASS_FAIL_NA_UNKNOWN`** | Core compliance evaluation. Evaluates safety condition with explicit `NA` (Not Applicable) and `UNKNOWN` support per `H040-006`. | `PASS`, `FAIL`, `NA`, `UNKNOWN` | Mandatory finding creation on `FAIL`; mandatory justification on `NA` / `UNKNOWN`. |
| **`SINGLE_CHOICE`** | Single selection from a closed, predefined list of localized enum options. | Exactly one selected `option_id` string | Option-level finding triggers; optional comment trigger. |
| **`MULTI_CHOICE`** | Multiple selection from a closed, predefined list of localized enum options. | Array of selected `option_id` strings | Option-level finding triggers; minimum/maximum selection bounds. |
| **`NUMERIC_MEASUREMENT`** | Quantitative sensor, meter, or physical measurement with explicit unit and reference bounds. | IEEE 754 float number | Warning/critical boundary range check; mandatory unit label (`dBA`, `ppm`, `Celsius`, `psi`). |
| **`TEXT_NOTE`** | Qualitative text observation or field notes constrained to safe lengths. | String (1 to 1000 characters UTF-8) | Whitespace trimming; control-character stripping. |
| **`EVIDENCE_ATTACHMENT`** | Direct photographic or document evidence requirement independent of finding generation. | Array of evidence references (`evd_syn_*`) | Minimum attachment count; supported MIME types (`image/jpeg`, `image/png`, `application/pdf`). |

### 3.1 Non-Compliance and `UNKNOWN` / `NA` Handling (`H040-006`)
In strict adherence to `H040-006`:
1. **Explicit `UNKNOWN` Option:** If an inspector cannot verify a physical condition due to accessibility, obstruction, or safety hazards, they must record `UNKNOWN`. An `UNKNOWN` response requires a mandatory explanatory note and triggers supervisory follow-up without creating a false-pass.
2. **Explicit `NA` (Not Applicable) Option:** If a physical condition is irrelevant to the inspected area (e.g. electrical substations in an outdoor yard), the inspector records `NA`. An `NA` response requires mandatory justification and is excluded from non-compliance score denominators.
3. **Automatic Finding Trigger:** Any question response resolving to non-compliance (`FAIL`, out-of-range measurement, or critical option selection) immediately generates a draft `Finding` entity in `MOD-WFA`.

---

## 4. Unsupported Generalized Form Builder Declarations

To protect milestone velocity, system stability, and security boundaries, the following capabilities are explicitly declared **`UNSUPPORTED_PRIVATE_ALPHA`**:

1. **No Generalized No-Code Form Builder Platform:**
   - Zero visual drag-and-drop schema canvas.
   - Zero arbitrary layout widgets, custom HTML components, or embedded CSS styling.
   - Zero dynamic database table generators or end-user schema alter commands.
2. **No Dynamic Client-Side Code Execution:**
   - Zero JavaScript `eval()`, WebAssembly, or sandboxed script execution inside checklist questions.
   - Zero custom calculated fields using arbitrary arithmetic expressions.
3. **No Unconstrained Recursive Sub-Forms:**
   - Section hierarchy is strictly 1-level deep (`Checklist` $\rightarrow$ `Sections` $\rightarrow$ `Questions`). Multi-tiered nested matrices, recursive grids, or nested sub-checklists are prohibited.
4. **No External Live API Lookup Questions:**
   - Questions cannot execute runtime REST/GraphQL queries to external third-party endpoints or enterprise ERPs. All options must be declared statically within the template schema.
5. **No Biometric or Sensitive Worker Identity Capture:**
   - Questions cannot capture employee national IDs, medical histories, facial biometrics, or personal health records (`H040-003`).

---

## 5. Applicability & Bounded Conditional Logic

Checklists, sections, and questions evaluate applicability against deterministic operational metadata to present only relevant evaluation points:

### 5.1 Applicability Entity Schema
```yaml
applicability_rule:
  rule_id: "app_syn_01"
  predicate_type: "METADATA_MATCH"  # METADATA_MATCH | PRIOR_ANSWER_MATCH
  target_field: "facility_type"     # Bounded metadata attribute
  operator: "EQUALS"                # EQUALS | NOT_EQUALS | IN | NOT_IN
  comparison_values:
    - "PETROCHEMICAL_REFINERY"
    - "HEAVY_FABRICATION"
  action: "SHOW"                    # SHOW | HIDE
```

### 5.2 Deterministic Conditional Evaluation Rules
1. **Bounded Preconditions:** Rules can only evaluate static context metadata (e.g. `Site.facility_type`, `Area.classification`, `Inspection.work_type`) or previous discrete single-choice question responses.
2. **Acyclic Dependency DAG:** Circular question dependencies (Question A depends on Question B, which depends on Question A) are strictly prohibited and rejected during template validation.
3. **Fail-Closed Visibility:** If an applicability condition cannot be evaluated due to missing context metadata, the rule defaults to **`SHOW`** (ensuring safety questions are never silently hidden due to configuration omission).

---

## 6. Evidence Policies & Provisional Scoring References

### 6.1 Evidence Attachment Policies
Each question declares an evidence policy governing attachment obligations:

| Evidence Policy | Behavioral Semantics & Validation Invariants |
| :--- | :--- |
| **`OPTIONAL`** | Inspector may voluntarily attach supporting photos or notes. |
| **`MANDATORY_ON_FAIL`** | If response evaluates to `FAIL` or non-compliant, at least one photo or document attachment is required before inspection submission. |
| **`MANDATORY_ALWAYS`** | Inspection submission is blocked until at least one photo or document attachment is uploaded, regardless of compliance outcome. |

### 6.2 Provisional Scoring References (`HUMAN_OWNED_UNSELECTED`)
In accordance with `HDEC-V040-FOUNDATION-054`, scoring weights and thresholds are strictly descriptive reference models:

```yaml
provisional_scoring_ref:
  scoring_model_reference: "PROVISIONAL_WEIGHTED_SCORE_V1"
  governance_status: "HUMAN_OWNED_UNSELECTED"  # Must not be used for binding certification
  question_weight: 1.0                         # Descriptive weight reference (0.1 - 10.0)
  criticality_level: "NORMAL"                  # NORMAL | CRITICAL_FAIL_TRIGGER
  provisional_finding_severity: "MEDIUM"       # LOW | MEDIUM | HIGH | CRITICAL
```

- **`CRITICAL_FAIL_TRIGGER` Reference:** Designates high-hazard questions where a non-compliant response indicates an immediate life-safety hazard, generating a priority review flag without claiming binding legal non-compliance.

---

## 7. Bilingual Localization & Language Mapping (`H040-002`)

In accordance with `H040-002`, every human-readable string in the checklist template model enforces dual-language mapping across English (`en-US`) and Thai (`th-TH`):

### 7.1 LocalizedStringMap Specification
```yaml
prompt:
  en-US: "Are all mobile scaffold wheels locked with functional wheel brakes?"
  th-TH: "ล้อของนั่งร้านแบบเคลื่อนที่ได้ทุกล้อถูกล็อคด้วยระบบเบรกล้อที่ใช้งานได้หรือไม่?"
guidance:
  en-US: "Inspect wheel brake levers on all four casters. Check for cracked housing or missing pins."
  th-TH: "ตรวจสอบคันโยกเบรกล้อทั้งสี่ด้าน ตรวจหารอยแตกร้าวหรือสลักล็อคที่สูญหาย"
```

### 7.2 Translation Validation Invariants
1. **Zero Monolingual Content:** A checklist template cannot be published if any prompt, title, guidance, or option label is missing either `en-US` or `th-TH`.
2. **Whitespace Trimming:** All localized string values are stripped of leading and trailing whitespace.
3. **No Automatic Machine Translation at Runtime:** Real-time unreviewed machine translation is prohibited. All translations must be declared and reviewed statically in the template schema.

---

## 8. Pilot Plant Synthetic Checklist Fixture (`H040-006`)

The following synthetic non-regulatory pilot checklist fixture demonstrates the complete, bounded checklist model:

```yaml
schema_version: "1.0.0"
checklist_id: "chk_syn_pilot_plant_safety_v1"
tenant_id: "ten_syn_safety_corp"
version: "1.0.0"
lifecycle_status: "PUBLISHED_IMMUTABLE"
title:
  en-US: "Pilot Industrial Plant Safety Inspection Baseline"
  th-TH: "การตรวจสอบความปลอดภัยของโรงงานอุตสาหกรรมนำร่อง"
description:
  en-US: "Standard non-regulatory private alpha pilot checklist covering general plant safety, mobile scaffolding, and electrical hazards."
  th-TH: "แบบตรวจสอบความปลอดภัยโรงงานอุตสาหกรรมนำร่องสำหรับไพรเวทแอลฟา ครอบคลุมความปลอดภัยทั่วไป นั่งร้านเคลื่อนที่ และอันตรายจากไฟฟ้า"
owner_role: "Checklist Author"
category: "PLANT_SAFETY"
is_pilot: true
effective_from: "2026-09-05T00:00:00Z"
effective_to: "2027-09-05T00:00:00Z"
published_at: "2026-09-05T14:30:00Z"
content_digest: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

sections:
  - section_id: "sec_syn_scaffolding_01"
    display_order: 1
    provisional_weight: 40.0
    title:
      en-US: "Mobile Scaffolding & Working at Heights"
      th-TH: "นั่งร้านแบบเคลื่อนที่และการทำงานบนที่สูง"
    description:
      en-US: "Verification of mobile scaffold stability, wheel brakes, guardrails, and access ladders."
      th-TH: "การตรวจสอบความมั่นคงของนั่งร้านเคลื่อนที่ เบรกล้อ ราวกันตก และบันไดทางขึ้น"
    applicability_rules: []
    questions:
      - question_id: "qst_syn_scaffold_brakes_01"
        code: "SCF-01"
        display_order: 1
        required: true
        question_type: "PASS_FAIL_NA_UNKNOWN"
        prompt:
          en-US: "Are all mobile scaffold wheels locked with functional brakes while workers are on the platform?"
          th-TH: "ล้อของนั่งร้านเคลื่อนที่ทุกล้อถูกล็อคด้วยเบรกที่ใช้งานได้ขณะที่มีผู้ปฏิบัติงานอยู่บนนั่งร้านหรือไม่?"
        guidance:
          en-US: "Inspect all caster wheel locks. Ensure brake levers are engaged and wheels do not rotate under lateral pressure."
          th-TH: "ตรวจสอบตัวล็อคล้อทุกล้อ ตรวจดูว่าคันโยกเบรกทำงานและล้อไม่หมุนเมื่อมีแรงผลักดันด้านข้าง"
        evidence_policy: "MANDATORY_ON_FAIL"
        criticality: "CRITICAL_FAIL_TRIGGER"
        provisional_scoring_ref:
          scoring_model_reference: "PROVISIONAL_WEIGHTED_SCORE_V1"
          governance_status: "HUMAN_OWNED_UNSELECTED"
          question_weight: 2.5
          criticality_level: "CRITICAL_FAIL_TRIGGER"
          provisional_finding_severity: "HIGH"

      - question_id: "qst_syn_scaffold_type_02"
        code: "SCF-02"
        display_order: 2
        required: true
        question_type: "SINGLE_CHOICE"
        prompt:
          en-US: "Select the primary construction material of the deployed mobile scaffold."
          th-TH: "เลือกประเภทวัสดุก่อสร้างหลักของนั่งร้านเคลื่อนที่ที่ติดตั้งใช้งาน"
        guidance:
          en-US: "Identify scaffold frame material according to manufacturer tag."
          th-TH: "ระบุประเภทวัสดุโครงสร้างนั่งร้านตามป้ายกำกับของผู้ผลิต"
        options:
          - option_id: "opt_scf_aluminum"
            label:
              en-US: "Lightweight Aluminum Frame"
              th-TH: "โครงสร้างอะลูมิเนียมน้ำหนักเบา"
            triggers_finding: false
          - option_id: "opt_scf_steel"
            label:
              en-US: "Heavy Tubular Steel"
              th-TH: "โครงสร้างเหล็กกล้าท่อหนา"
            triggers_finding: false
          - option_id: "opt_scf_timber"
            label:
              en-US: "Wood / Bamboo Improvised Structure"
              th-TH: "โครงสร้างไม้หรือไม้ไผ่ดัดแปลง"
            triggers_finding: true
        evidence_policy: "OPTIONAL"
        criticality: "NORMAL"
        provisional_scoring_ref:
          scoring_model_reference: "PROVISIONAL_WEIGHTED_SCORE_V1"
          governance_status: "HUMAN_OWNED_UNSELECTED"
          question_weight: 1.0
          criticality_level: "NORMAL"
          provisional_finding_severity: "MEDIUM"

  - section_id: "sec_syn_electrical_02"
    display_order: 2
    provisional_weight: 35.0
    title:
      en-US: "Electrical Enclosures and Grounding"
      th-TH: "ตู้ควบคุมไฟฟ้าและระบบต่อลงดิน"
    description:
      en-US: "Inspection of high-voltage enclosures, grounding resistance, and cable insulation integrity."
      th-TH: "การตรวจสอบตู้ไฟฟ้าแรงสูง ความต้านทานการต่อลงดิน และความสมบูรณ์ของฉนวนสายไฟ"
    applicability_rules: []
    questions:
      - question_id: "qst_syn_ground_resistance_01"
        code: "ELE-01"
        display_order: 1
        required: true
        question_type: "NUMERIC_MEASUREMENT"
        prompt:
          en-US: "Record measured ground loop resistance at main distribution panel."
          th-TH: "บันทึกค่าความต้านทานการต่อลงดินที่วัดได้ ณ ตู้เมนสวิตช์บอร์ด"
        guidance:
          en-US: "Use calibrated earth ground resistance clamp tester. Normal baseline is <= 5.0 ohms."
          th-TH: "ใช้เครื่องวัดความต้านทานดินที่สอบเทียบแล้ว ค่าปกติควรน้อยกว่าหรือเท่ากับ 5.0 โอห์ม"
        numeric_spec:
          unit: "ohms"
          min_value: 0.0
          max_value: 100.0
          warning_threshold_high: 5.0
          critical_threshold_high: 10.0
        evidence_policy: "MANDATORY_ALWAYS"
        criticality: "CRITICAL_FAIL_TRIGGER"
        provisional_scoring_ref:
          scoring_model_reference: "PROVISIONAL_WEIGHTED_SCORE_V1"
          governance_status: "HUMAN_OWNED_UNSELECTED"
          question_weight: 2.0
          criticality_level: "CRITICAL_FAIL_TRIGGER"
          provisional_finding_severity: "HIGH"

      - question_id: "qst_syn_elec_ppe_02"
        code: "ELE-02"
        display_order: 2
        required: true
        question_type: "MULTI_CHOICE"
        prompt:
          en-US: "Verify all mandatory electrical personal protective equipment (PPE) present on site."
          th-TH: "ตรวจสอบอุปกรณ์คุ้มครองความปลอดภัยส่วนบุคคล (PPE) ด้านไฟฟ้าที่จำเป็นทั้งหมดในพื้นที่"
        guidance:
          en-US: "Check for unexpired dielectric gloves, arc flash shield, and insulated shoes."
          th-TH: "ตรวจสอบถุงมือฉนวนไฟฟ้าที่ยังไม่หมดอายุ กระบังหน้าป้องกันประกายไฟ และรองเท้าฉนวนไฟฟ้า"
        options:
          - option_id: "opt_ppe_dielectric_gloves"
            label:
              en-US: "Class 0 Dielectric Insulated Gloves"
              th-TH: "ถุงมือฉนวนไฟฟ้า Class 0"
            triggers_finding: false
          - option_id: "opt_ppe_arc_shield"
            label:
              en-US: "Arc Flash Face Shield (ATPV >= 12 cal/cm²)"
              th-TH: "กระบังหน้ากันประกายไฟอาร์ค"
            triggers_finding: false
          - option_id: "opt_ppe_safety_shoes"
            label:
              en-US: "Dielectric Safety Shoes"
              th-TH: "รองเท้านิรภัยฉนวนไฟฟ้า"
            triggers_finding: false
        evidence_policy: "OPTIONAL"
        criticality: "NORMAL"
        provisional_scoring_ref:
          scoring_model_reference: "PROVISIONAL_WEIGHTED_SCORE_V1"
          governance_status: "HUMAN_OWNED_UNSELECTED"
          question_weight: 1.0
          criticality_level: "NORMAL"
          provisional_finding_severity: "LOW"

  - section_id: "sec_syn_observations_03"
    display_order: 3
    provisional_weight: 25.0
    title:
      en-US: "Field Observations & Overview Photo"
      th-TH: "ข้อสังเกตหน้างานและภาพถ่ายภาพรวม"
    description:
      en-US: "General inspector notes and mandatory wide-angle photographic evidence of inspected area."
      th-TH: "บันทึกข้อสังเกตทั่วไปของผู้ตรวจสอบและภาพถ่ายมุมกว้างของพื้นที่ตรวจสอบที่จำเป็น"
    applicability_rules: []
    questions:
      - question_id: "qst_syn_site_photo_01"
        code: "OBS-01"
        display_order: 1
        required: true
        question_type: "EVIDENCE_ATTACHMENT"
        prompt:
          en-US: "Attach at least one wide-angle overview photograph of the inspected plant area."
          th-TH: "แนบภาพถ่ายมุมกว้างอย่างน้อยหนึ่งภาพเพื่อแสดงภาพรวมของพื้นที่โรงงานที่ได้รับการตรวจสอบ"
        guidance:
          en-US: "Capture clear lighting showing overall working conditions and housekeeping status."
          th-TH: "ถ่ายภาพในสภาพแสงที่ชัดเจนเพื่อแสดงสภาพแวดล้อมการทำงานและความเป็นระเบียบเรียบร้อย"
        evidence_policy: "MANDATORY_ALWAYS"
        criticality: "NORMAL"
        provisional_scoring_ref:
          scoring_model_reference: "PROVISIONAL_WEIGHTED_SCORE_V1"
          governance_status: "HUMAN_OWNED_UNSELECTED"
          question_weight: 1.0
          criticality_level: "NORMAL"
          provisional_finding_severity: "LOW"

      - question_id: "qst_syn_general_notes_02"
        code: "OBS-02"
        display_order: 2
        required: false
        question_type: "TEXT_NOTE"
        prompt:
          en-US: "Record additional safety observations, housekeeping notes, or weather conditions."
          th-TH: "บันทึกข้อสังเกตด้านความปลอดภัยเพิ่มเติม ความเป็นระเบียบเรียบร้อย หรือสภาพอากาศ"
        guidance:
          en-US: "Maximum 1000 characters. Omit employee personal names or private identifiers."
          th-TH: "ความยาวไม่เกิน 1,000 ตัวอักษร ห้ามระบุชื่อบุคคลหรือข้อมูลส่วนบุคคลของพนักงาน"
        evidence_policy: "OPTIONAL"
        criticality: "NORMAL"
        provisional_scoring_ref:
          scoring_model_reference: "PROVISIONAL_WEIGHTED_SCORE_V1"
          governance_status: "HUMAN_OWNED_UNSELECTED"
          question_weight: 0.5
          criticality_level: "NORMAL"
          provisional_finding_severity: "LOW"
```

---

## 9. Governance Boundaries & Non-Claims

In strict compliance with `HDEC-V040-FOUNDATION-054`:

1. **Synthetic-Only Data Policy (`H040-003`):** All checklist IDs (`chk_syn_*`), questions, sections, and synthetic test payloads are fictionalized alpha fixtures. Zero customer data, real corporate records, or actual employee PII is included.
2. **Default-Deny Change Authority (`H040-004`):** Checklist authoring, editing, and approval require explicit role entitlement (`Checklist Author`, `Independent Reviewer`).
3. **No Generalized No-Code Platform:** As declared in Section 4, this model rejects dynamic form builder engines and unconstrained script execution.
4. **Non-Binding Scoring References:** All weights, thresholds, and severity ratings are descriptive and provisional. Final scoring policy remains `HUMAN_OWNED_UNSELECTED` pending owner decision.
5. **No Deployment or Release Claim:** Gates `H040-007` through `H040-011` remain on `HOLD`. Zero public routes, internet endpoints, CDN activations, or production software deployments are claimed or authorized.
