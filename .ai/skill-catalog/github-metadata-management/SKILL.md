---
name: github-metadata-management
description: >
  Operate GitHub Issues, Discussions, labels, milestones, Projects, assignments, comments, and relationships after the exact ADR-0006 evidence gate passes.
license: Proprietary
metadata:
  oshe-version: "0.1.0"
---

# GitHub Metadata Management

## Objective

Maintain complete and traceable GitHub planning metadata without acting outside the exact assignment and passing operation gate.

## Required Inputs

- release-evidence-agent assignment with github-manager profile;
- exact organization, repository, object IDs, intended changes, and expected pre/post state;
- valid METADATA operation-gate record and approved credential profile.

## Procedure

1. Re-read the target objects and compare current state with the recorded pre-state.
2. Run `python .ai/tools/evaluate_github_operation.py <gate-record>` immediately before execution.
3. Apply only the listed metadata mutations using non-interactive API or CLI commands.
4. Read back every changed object and record URLs, IDs, timestamps, and before/after values.
5. Stop and invoke recovery if any mutation is partial, unexpected, or broader than the gate.

## Required Output

Gate digest, exact mutations, operation receipts, post-state validation, failures, assumptions, and unresolved findings.

## Stop Conditions

- target state drifted after gate evaluation;
- an object is outside the allowlisted organization, repository, issue, milestone, or Project scope;
- the mutation would hide unresolved evidence or misrepresent completion.

## Evaluation Cases

- accept exact issue/label/milestone updates with readback evidence;
- reject an unlisted bulk close, deletion, or cross-repository mutation.
