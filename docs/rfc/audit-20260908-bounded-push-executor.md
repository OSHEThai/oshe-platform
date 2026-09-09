# Bounded audit branch publication

Status: UNDER_REVIEW

## Authority and scope

The owner's 2026-09-08 approval, recorded in the external wave control record
`owner-push-extension-011.md`, authorizes this narrow implementation. H3 is the
producer; Security supplies the independent final exact-commit review. Neither
implementation nor this RFC grants Release authority. Only the existing Release
role/profile may execute a valid operation gate. The existing direct-gh route is
unchanged. This is not permission to use arbitrary Git commands or GitHub APIs.

The policy adds one expiring `DIRECT_GIT_PUSH` route for
`https://github.com/OSHEThai/oshe-platform.git`. Source and destination branches
must be one of the six enumerated `fix/audit-20260908-*` branches in the policy
and validator. There is no wildcard matching, main/version/release publication,
force, force-with-lease, deletion, mirror, tag, recursive submodule, custom
receive-pack, push-option, or user-supplied configuration capability.

## Gate and execution

Each operation binds the canonical worktree, source branch and 40-character
commit, exact destination ref and URL, installed Git path and binary hash, and
expected remote OID (or null for an absent branch). The command must exactly
equal the validator's fixed argv. The generic schema, role, credential profile,
lease, route expiry, evidence flags, command digest and independent-review gates
remain mandatory.

The `execution.push.review_evidence` and `test_evidence` objects each contain an
absolute canonical JSON file path and its raw-file SHA-256. Both files require
`disposition: PASS` and `target_commit` equal to the candidate. Review evidence
also requires `reviewer_assignment_id` matching the gate and a distinct
`producer_assignment_id`. Test evidence requires integer `passed > 0`,
`failed = 0`, and `skipped = 0`; the accompanying report must retain actual
commands and results. These records are attestations by trusted evidence
authors, not cryptographic proof of reviewer identity or test execution.

Release constructs the expected prestate with the read-only `snapshot(record)`
and canonical `digest(...)` functions after independently checking authority and
evidence. Construction is not approval. The executor revalidates the generic
gate using actual time, verifies a clean exact-HEAD worktree, checks configuration
and evidence hashes, reads the remote ref, and proves ancestry for an existing
branch. It compares the complete snapshot digest with the gate. Dry-run performs
these read-only checks but does not publish or consume the gate.

An actual attempt exclusively creates `<common-git-dir>/audit-push-gates/<gate-id>.used`.
The marker consumes the gate even if subsequent checks or transport fail. A
second full snapshot and current-time gate evaluation precede the single normal
non-force push. A successful transport must be followed by exact remote OID
readback. Receipts expose exit status and output hashes, not transport output.
Missing objects, ambiguous state, timeout, changed evidence, stale state, and
readback mismatch are failures, never publication success.

Do not remove markers or automatically retry failed/uncertain attempts. Release
must reconcile live remote state and preserve the failed evidence before any
newly authorized gate. A new gate ID alone is not renewed authorization.

## Environment and trust boundary

The validator rejects Git environment overrides, supported proxy/loader override
variables, URL rewriting, includes, custom hooks, alternate object stores,
replacement refs, custom credential helpers, and transport-affecting configuration
covered by its deny checks. Fixed command configuration disables filesystem
monitoring, LFS filters, HTTP redirects and push signing; pre-push hooks and
submodule recursion cannot run. The exact origin must be present once and match
the approved HTTPS URL. Only existing Git Credential Manager helpers are accepted;
no credentials, provider settings, routes or account provisioning are changed.

Following Security finding H-P1-001, `core.askPass` and every `credential.*`
configuration key except the unscoped `credential.helper=manager` or
`manager-core` are rejected, including URL-scoped helpers and identity/store
selectors, subject only to the exact H-P1-002 compatibility exception below.
All ambient `GCM_*` variables and GitHub token override variables are
rejected before Git diagnostics. The executor alone sets its fixed noninteractive
GCM value after this check. Configuration discovered through HOME or XDG is subject
to the same credential-key rejection; changing its source does not exempt it.

H-P1-002 adds one exact compatibility exception for the native Git-for-Windows
system setting `credential.https://dev.azure.com.useHttpPath=true`. Its literal
HTTPS host is unrelated to the fixed GitHub remote; the setting is not a helper
or identity selector. No wildcard, GitHub-scoped setting, Azure helper/username,
or other credential key is exempted. Tests reproduce this native setting beside
the approved manager helper; an actual native read-only config preflight is also
required in candidate evidence. No host configuration is modified by the fix.

This is a procedural control on a trusted workstation, not an OS sandbox or a
complete defense against a hostile local user. Installed Git, its child helpers,
OS trust store, credential manager, environment outside the explicitly checked
overrides, and trusted evidence authors remain dependencies. Release must verify
the existing credential profile's effective repository identity; a gh identity
probe alone does not prove which identity Git Credential Manager will use.

Preflight plus ordinary non-force push is **not atomic compare-and-swap** against
the recorded remote OID. A concurrent change between the final read and push may
still be accepted if Git regards the resulting update as a fast-forward. The
source commit/ref remain fixed, non-fast-forward updates fail, and final readback
must match. Local configuration/evidence can likewise race after their last
check. The bounded workflow requires exclusive trusted ownership and no competing
publisher; it does not claim to enforce that against hostile concurrent actors.

## Verification and bootstrap

`tests.test_git_push_gate` uses disposable repositories, a local bare remote and
mocked transport selection. It must never write to GitHub. Tests cover permitted
creation/fast-forward, argument and scope rejection, unsafe configuration and
environment, stale state, evidence binding, replay, expiry and readback failure.
Existing agent-OS tests and the configured local CI must also pass at the final
candidate commit. Test outcomes belong in the exact-HEAD evidence, not this RFC.

The owner approved using the Security-reviewed local executor for the first
bounded publication; remote availability of the executor is not a prerequisite.
This bootstrap still requires the independent final PASS, actual tests, current
Release authority, unexpired route and an exact per-operation gate. No remote
write or release/deployment acceptance is established by merging this code.
