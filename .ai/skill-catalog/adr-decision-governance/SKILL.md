---
name: adr-decision-governance
description: >
  Create or update ADRs and decision records with authority, alternatives, consequences, review triggers, and supersession controls. Use for material architecture, security, safety, legal, permission, or source-of-truth decisions.
license: Proprietary
metadata:
  oshe-version: "0.1.0"
---

# ADR and Decision Governance

## Objective

Produce an attributable decision record without silently converting a proposal into an approved decision.

## Required Inputs

- decision question, scope, owner, and approval authority;
- related requirements, risks, evidence, prior ADRs, and affected repositories;
- lifecycle and metadata standards.

## Procedure

1. Search for an existing decision and supersession chain.
2. State context, options, trade-offs, constraints, and protected decisions.
3. Record the selected or proposed outcome, consequences, migration, rollback, and assurance impact.
4. Link affected requirements, risks, schemas, issues, and evidence.
5. Set lifecycle honestly and obtain required independent and human review.

## Required Output

ADR or decision update, link inventory, validation results, unresolved disagreements, and approval needed.

## Stop Conditions

- decision authority is unclear;
- a protected decision lacks Sole Human Owner approval;
- the change conflicts with an active ADR without an explicit supersession path.

## Evaluation Cases

- accept a complete DRAFT with alternatives and review triggers;
- reject self-approved or silently superseding decisions.
