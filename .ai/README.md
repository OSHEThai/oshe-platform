# AI Agent Operating System

This directory is the repository-ready control plane for the OSHE role-agent operating model approved by ADR-0005. It contains the canonical role catalog, specialist profiles, skills, contracts, policies, runbooks, provider-route records, examples, and static validation.

## Layout

- `roles/` — 12 canonical authority roles and their machine-readable registry.
- `agents/` — narrower specialist profiles; profiles never grant authority independently of a canonical role assignment.
- `bundles/` and `skill-catalog/` — role-to-skill mappings and reusable capability procedures.
- `schemas/` and `examples/` — mission, task, assignment, result, review, integration, handoff, and provider-control contracts.
- `policies/` — authority, data, risk, tool, write, budget, evidence, retention, repository, incident, model, and routing controls.
- `provider-routes/` — candidate route records and policy-review evidence. All routes remain disabled until qualified and approved.
- `policies/github-operations.yaml` — ADR-0006 standing conditional full GitHub authority and action-specific evidence gates.
- `policies/repository-workflow-and-ci.yaml` — ADR-0007 local-first CI, PR routing, checkpoint, branch-lifecycle, and cleanup controls.
- `runbooks/` — bounded operating and recovery procedures.
- `tools/validate_agent_os.py` — static cross-reference, structure, YAML, JSON, and fail-closed checks.
- `preparation-handoff.md` — review and four-PR publication split for V010-I013 through V010-I016.

## Current Status

The files are prepared for controlled GitHub transfer and review. ADR-0006 authorizes full GitHub operations after an exact gate PASS, but static validation does not prove Herdr adapter enforcement, approved provider routes, approved credential profiles, safe runtime dispatch, independent-provider review, or release qualification. Runtime state, transcripts, credentials, tokens, and customer data must never be committed.

Install validator dependencies with `python -m pip install --disable-pip-version-check -r .ai/requirements-validation.txt`. Then run `python tools/run_local_ci.py --mode incremental` for the non-fail-fast checkpointed local batch, `python tools/validate_repository.py --repo-kind platform` for the integrated repository check, or `python .ai/tools/validate_agent_os.py` for the AI controls alone.
