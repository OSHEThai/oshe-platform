# Validation or Schema Failure

## Trigger

A governed JSON, YAML, schema, example, ID, or cross-reference fails parsing or validation.

## Procedure

1. Stop integration and preserve the exact failing file, command, environment, and output.
2. Determine whether the artifact, schema, validator, or version selection is wrong.
3. Repair the narrowest authoritative source; do not weaken a schema merely to obtain a pass.
4. Run positive and negative validation cases and all affected repository checks.
5. Obtain independent review for a material contract or compatibility change.

## Resume Criteria

All applicable artifacts parse, examples validate, references resolve, and breaking changes have an approved migration path.

## Required Record

Failure reproduction, cause, changed paths, schema version, positive and negative tests, compatibility impact, and unresolved decisions.
