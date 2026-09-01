---
name: github-repository-administration
description: >
  Administer repositories, rulesets, protections, integrations, collaborators, Apps, credentials, visibility, and transfer after high-impact ADR-0006 gates pass. Repository deletion is always denied under the Sole Human Owner's 2026-09-01 prohibition.
license: Proprietary
metadata:
  oshe-version: "0.1.0"
---

# GitHub Repository Administration

## Objective

Perform exact allowed GitHub administrative or destructive operations with independent review, recovery evidence, secret safety, and post-state verification. Repository deletion is never an allowed operation.

## Required Inputs

- exact organization, repository, administrative target, current settings export, desired state, impact, and rollback or reconstruction plan;
- passing REPOSITORY_ADMIN, CREDENTIAL, SECURITY, DESTRUCTIVE, or DEPLOYMENT_TRIGGER operation gate;
- independent-review PASS and approved credential profile.

## Procedure

1. Export or record the safe current configuration and resolve the exact target immediately before execution.
2. Verify owners, affected repositories, protections, workflows, integrations, credentials, data, and external effects.
3. Evaluate the high-impact gate and confirm recovery evidence and validity window.
4. Apply only the exact change; never print or retrieve secret values and never weaken controls to bypass a failed gate.
5. Read back settings, permissions, rules, audit events, and downstream effects; invoke recovery on divergence.

## Required Output

Redacted pre/post configuration, target identity, operation receipt, independent review, recovery status, audit references, and unresolved impact.

## Stop Conditions

- repository deletion is requested, regardless of gate, review, credential, backup, reconstruction, or recovery evidence;
- exact target, ownership, impact, backup, recovery, credential, or independent review is missing;
- state changes after gate evaluation;
- an external production, customer-data, billing, ownership, legal, or safety boundary lacks separate authority.

## Evaluation Cases

- accept an exact independently reviewed ruleset or credential rotation with redacted evidence;
- reject repository deletion unconditionally and reject any operation that exposes a secret.
