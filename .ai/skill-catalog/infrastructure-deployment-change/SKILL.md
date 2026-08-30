---
name: infrastructure-deployment-change
description: >
  Prepare infrastructure and deployment-profile changes with environment separation, least privilege, rollback, and verification. Use for deploy definitions or platform configuration.
license: Proprietary
metadata:
  oshe-version: "0.1.0"
---

# Infrastructure and Deployment Change

## Objective

Prepare reproducible deployment controls without granting production authority or embedding secrets.

## Required Inputs

- target profile and environment, architecture, resources, identities, network, and data classes;
- configuration source, secret references, rollback, and recovery requirements;
- exact validation and deployment authority boundaries.

## Procedure

1. Confirm environment separation and source-of-truth configuration.
2. Apply least privilege to identities, network, storage, and runtime permissions.
3. Use secret references only; never commit secret values.
4. Define health, migration ordering, rollback, backup, and failure behavior.
5. Run static or non-production validation and label evidence class accurately.

## Required Output

Configuration diff, dependency and permission analysis, validation, rollback, residual risks, and human deployment decision.

## Stop Conditions

- production credentials or customer data are required;
- rollback or recovery is undefined for a material change;
- the assignment is asked to deploy or alter protected settings without approval.

## Evaluation Cases

- accept a scoped non-production configuration with rollback;
- reject embedded secrets or unbounded production permissions.
