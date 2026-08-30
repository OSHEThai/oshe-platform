---
name: security-review
description: >
  Review threat surface, authorization, isolation, input handling, secrets, dependencies, and exfiltration. Use this skill when the assigned task explicitly involves this capability.
license: Proprietary
metadata:
  oshe-version: "0.1.0"
---

# Security Review

## Objective

Review threat surface, authorization, isolation, input handling, secrets, dependencies, and exfiltration.

## Required Inputs

- active role;
- mission and task packets;
- allowed paths and tool profile;
- relevant repository evidence.

## Procedure

1. Validate inputs and authority.
2. Inspect only the relevant scope.
3. Execute the capability-specific work.
4. Run required deterministic checks.
5. Produce the required structured output.
6. Stop and escalate on ambiguity, unsafe state, or out-of-scope access.

## Required Output

Use the output schema specified by the task packet. Include assumptions, risks, tests, and human decisions needed.

## Stop Conditions

- Assignment, authority, route, data class, or required input cannot be validated.
- Work would exceed the tool profile, write lease, repository boundary, or approved scope.
- A protected decision, unsafe action, missing rollback, or unresolved critical finding requires human authority.

## Evaluation Cases

- Accept a bounded result with reproducible commands, evidence class, assumptions, risks, and decisions needed.
- Reject self-approval, hidden delegation, out-of-scope writes, unapproved routes, or skipped checks reported as passing.
