# AI Agent Operating System

This directory is the repository-ready control plane for the OSHE role-agent operating model. It contains role and position records, specialist profiles, skills, contracts, policies, runbooks, provider-route records, examples, and static validation.

## Current Team Layout

The current operating layout has **18 named positions**:

- 1 Project Manager and 1 PM Secretary;
- 9 Leads: Product and Planning, Architecture and Data, Engineering, AI and Automation, Security Privacy and Safety, Quality and Independent Review, Documentation Standards and Legal, DevOps Release and Evidence, and Implementation and Customer Success;
- 7 Free Workers, including `free-mercury`.

Permanent Worker positions are retired. Routine reporting follows `Free Worker -> Lead -> PM Secretary -> Project Manager`. The Project Manager directs Leads; a Lead reviews Free Worker output before the PM Secretary records a Lead-approved completion. A Free Worker receives a named, time-bounded assignment from a Lead and has no standing authority.

The `roles/registry.yaml` catalog and its 12 role cards are a retained **legacy authority catalog**. They are not the current team topology and must not be used to infer current reporting lines, position counts, or agent assignment. Their controlled migration to the 18-position lean topology remains tracked work; until then, the active team layout and reporting model in this README govern operational interpretation.

## Layout

- `roles/` — legacy 12-card authority catalog retained for controlled migration; not the current team layout.
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
