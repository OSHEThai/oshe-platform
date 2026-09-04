# Contributing

All changes enter through a pull request. Link an existing governing Issue when one exists. Work directly authorized by the Sole Human Owner outside the prepared Issue set uses the pull request as its primary audit record and does not require a synthetic Issue first.

## Minimum Flow

1. Link the governing Issue, or state the direct out-of-Issue authorization in the pull request.
2. Use a short-lived branch or controlled Herdr worktree. `main` is permanent. At each release candidate cut, retain `release/v<major>.<minor>.<patch>` as the long-lived backport branch for security and critical fixes to that released version.
3. Respect allowed paths and module ownership.
4. Run `python tools/run_local_ci.py --mode incremental`; it runs all applicable checks without fail-fast behavior and checkpoints unchanged passes.
5. Fix the complete failure set together, rerun locally, and open or update the pull request only after the applicable local batch passes.
6. Complete the pull-request template and let GitHub CI verify the exact head.
7. Resolve review findings and required checks. Use Full CI only for Milestone closure, locally first and then on GitHub.
8. Merge only through the ADR-0006 evidence gate. Delete only the merged short-lived head branch, then verify its absence. Never delete `main` or a `release/v*` branch through ordinary cleanup; version-branch deletion requires a Sole Human Owner decision with release and recovery review.

## Developer Certificate of Origin

Contributions are inbound-equals-outbound under the file's declared license.
Contributors must certify [DCO 1.1](DCO-1.1.txt) with a `Signed-off-by:` line
in each commit. Do not submit third-party material unless its provenance and
rights metadata are recorded and compatible.

## Prohibited

- production credentials or customer data;
- direct push to protected `main`;
- force push to shared branches;
- GitHub CI before the applicable local CI pass;
- Full CI outside Milestone closure;
- checkpoint reuse after command, toolchain, repository input, or base commit changes;
- direct cross-module database access;
- undocumented destructive migration;
- bypass of safety, security, privacy, or legal gates.
