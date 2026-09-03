# OSHE Context Digest (OSHE-CONTEXT-v1)

Standalone governed context-digest contract (V010-I013 semantic 1B). A context
digest proves only the declared input-byte binding at a bound commit; it proves
no semantic equivalence, authority, freshness, acceptance, or runtime execution.

## Carrier fields

| Field | Rule |
| --- | --- |
| `schema_version` | required string, const `1.0.0` |
| `digest_type` | required string, const `oshe-context-digest` |
| `algorithm` | required string, const `sha256` |
| `canonicalization` | required string, const `OSHE-CONTEXT-v1` |
| `repository_commit` | required string, full lowercase 40- or 64-hex commit identity |
| `inputs` | required array 1..* of closed `{path, sha256}` objects, unique and sorted by Unicode code-point order of `path` |
| `digest` | required string, `sha256:` followed by 64 lowercase hex characters |

## Canonicalization

`OSHE-CONTEXT-v1` hashes raw regular-file blob bytes at the bound commit. The
declared `repository_commit` MUST resolve to a real commit object in the local
repository. Each declared input path is resolved through Git tree and object
reads only, never through worktree files or links; a symlink, submodule, or
directory entry is rejected. Each input is one UTF-8 leaf record in the exact
form `<path>\0<lowercase-sha256>\n`, where the NUL and LF are literal bytes.
Leaf records are concatenated in sorted path order and hashed with SHA-256;
`digest` is `sha256:` followed by that lowercase hex.

Input paths MUST be a single canonical repository-relative POSIX path. Drive
qualification, colon/alias forms, leading root slash, backslash, empty
segments, `.` and `..` segments, and traversal forms are rejected. Generated
timestamps, absolute paths, provider names, workspace paths, and execution
order MUST NOT enter the digest.

## Fail-closed rules

Validation MUST fail closed for an unknown commit, absent path, non-regular or
symlink entry, byte mismatch against the bound-commit blob, duplicate path,
unsorted path list, non-canonical path form, digest mismatch, malformed commit
identity, unsupported algorithm or canonicalization, or any unknown field.
Validation never substitutes worktree or external bytes for bound-commit blob
bytes.

## Boundaries

- No numeric adapter budget is selected or inferred (`DEFER` per
  HDEC-V010-I013-I015-PROTECTED-SEMANTICS-SELECTION-037).
- Root `CODEX.md` and `DEEPSEEK.md` are reserved for their dedicated CLI
  conventions and are not required artifacts (`ROOT_CODEX_DEEPSEEK_RESERVED`).
- This contract does not authorize provider routes, credentials, runtime
  dispatch, or any active-behavior claim. Provider notes remain static
  fail-closed default-deny.
