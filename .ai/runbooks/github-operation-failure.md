# GitHub Operation Failure or Divergence

## Trigger

An evidence-gated GitHub operation fails, partially succeeds, affects an unexpected target, produces unexpected post-state, exposes credential material, or diverges from its operation gate.

## Procedure

1. Stop the current and dependent GitHub operations; do not retry with broader permissions or a different target.
2. Preserve the operation gate, record digest, commands or API action class, response metadata, audit event, exact target, and observed before/after state without exposing secrets.
3. Revoke or suspend the credential profile if compromise, scope drift, or secret exposure is possible.
4. Classify the result as no-change, partial-change, unexpected-change, destructive-loss, security incident, or external-effect incident.
5. Apply only the recovery action already authorized by the gate; otherwise create a new recovery assignment and high-impact operation gate.
6. Re-read repository, branch, PR, release, ruleset, credential, workflow, and external-effect state as applicable.
7. Record residual impact and independent review; escalate non-GitHub production, customer-data, legal, safety, ownership, billing, or recovery-owner decisions.

## Stop and Resume Rules

Do not resume while current state, credential integrity, audit evidence, recovery path, or downstream effects are unknown. A successful retry does not erase the failed operation or its evidence.

## Required Record

Gate and assignment IDs, record digest, actor and credential-profile IDs, exact organization/repository/target, attempted action, timestamps, response class, pre/post state, audit references, containment, recovery, verification, independent review, and unresolved decisions.
