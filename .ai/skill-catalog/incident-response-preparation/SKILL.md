---
name: incident-response-preparation
description: >
  Prepare detection, containment, evidence preservation, recovery, communication, and resumption decision procedures. Use for security or control incident readiness.
license: Proprietary
metadata:
  oshe-version: "0.1.0"
---

# Incident Response Preparation

## Objective

Create a safe, attributable incident path that stops harm and preserves the minimum necessary evidence.

## Required Inputs

- incident classes, owners, escalation contacts, data classes, systems, and provider routes;
- containment, credential-revocation, evidence, recovery, and communication constraints;
- applicable runbooks and human approval gates.

## Procedure

1. Define triggers, severity, authority, and immediate stop behavior.
2. Map containment actions that do not destroy required evidence.
3. Define secret revocation, data exposure assessment, recovery, and validation.
4. Prepare internal communication and decision packets without publishing unverified claims.
5. Exercise a safe scenario and retain findings and unresolved gaps.

## Required Output

Runbook, responsibility map, exercise evidence, gaps, recovery criteria, and human resumption decision.

## Stop Conditions

- active exposure requires immediate human-controlled containment;
- evidence collection would reveal or spread a live secret;
- an agent is asked to approve resumption or incident closure.

## Evaluation Cases

- accept a fail-closed exercised response with evidence preservation;
- reject procedures that delete evidence or expose credentials.
