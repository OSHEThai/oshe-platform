---
name: ci-workflow-change
description: >
  Change CI workflows with least privilege, pinned dependencies, deterministic checks, and safe pull-request behavior. Use for hosted automation or status-check changes.
license: Proprietary
metadata:
  oshe-version: "0.1.0"
---

# CI Workflow Change

## Objective

Prepare deterministic, least-privilege CI without creating an unsafe secret or protected-branch path.

## Required Inputs

- workflow scope, required check name, event model, and repository rules;
- dependency versions, permissions, secrets policy, and expected artifacts;
- local or synthetic validation method.

## Procedure

1. Inspect existing workflows, rulesets, and required check contracts.
2. Minimize triggers, token permissions, network access, and artifact retention.
3. Pin actions and dependencies according to supply-chain policy.
4. Prevent untrusted pull-request code from receiving privileged secrets.
5. Validate syntax locally and obtain hosted CI evidence on the exact commit before release credit.

## Required Output

Workflow diff, permission analysis, pins, commands, evidence class, expected check name, and rollback.

## Stop Conditions

- a secret or write permission is broader than justified;
- required hosted behavior cannot be tested safely;
- the change would bypass review or branch protection.

## Evaluation Cases

- accept a read-only pull-request workflow with pinned actions;
- reject mutable action tags or secret exposure to untrusted code.
