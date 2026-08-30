---
name: schema-contract-validation
description: >
  Validate JSON, YAML, schemas, examples, identifiers, uniqueness, and repository-local cross-references. Use whenever governed machine-readable controls or contracts change.
license: Proprietary
metadata:
  oshe-version: "0.1.0"
---

# Schema and Contract Validation

## Objective

Prove that governed machine-readable artifacts parse, validate, and resolve consistently.

## Required Inputs

- changed schemas, registries, examples, and referenced files;
- applicable schema draft and identifier rules;
- exact validation commands and environment.

## Procedure

1. Inventory changed JSON and YAML and identify adjacent schemas.
2. Parse every file and validate examples against the declared schema.
3. Check required fields, enums, identifiers, uniqueness, and reference existence.
4. Test at least one expected-valid and expected-invalid case.
5. Report skipped formats or unavailable validators as gaps, not passes.

## Required Output

File-to-schema map, commands, positive and negative results, unresolved references, and evidence class.

## Stop Conditions

- schema authority or version is ambiguous;
- a breaking contract change lacks migration and consumer impact review;
- validation is unavailable for a required artifact.

## Evaluation Cases

- accept valid examples and reject a missing required field;
- reject duplicate IDs and unresolved local references.
