---
document_id: ADR-0005
title: Sole Human Owner and Herdr Role-Agent Operating Model
document_type: architecture_decision_record
document_version: 1.0.0
lifecycle_status: APPROVED
amended_by:
  - ADR-0006
  - ADR-0007
maturity: BASELINE
implementation_status: IMPLEMENTED
review_status: APPROVED
owner: Sole Human Owner / Human Product and Release Authority
reviewers:
- Project Management Agent
- Architecture Agent
- Security Privacy and Product Safety Agent
- Test and Quality Agent
- Independent Review and Challenge Agent
applicable_releases:
- v0.1.0
- v0.2.0
- v0.3.0
- v0.4.0
effective_date: '2026-08-18'
last_reviewed_date: '2026-08-18'
next_review_trigger: Human ownership or authority changes.; Herdr, provider, model, CLI,
  adapter, or orchestration boundary materially changes.; A production, customer-data,
  legal, medical, certification, or safety-critical gate requires additional human
  accountability.; Agent review independence, continuity, cost, or data-policy evidence
  is insufficient.; Calendar, budget, GitHub, or deployment authority changes materially.
source_of_truth: GOOGLE_DRIVE
classification: INTERNAL
change_risk: R3
related_decisions:
- DEC-008
related_issues:
- PCR-006
- OQ-014
supersedes: []
superseded_by: null
---

# ADR-0005 — Sole Human Owner and Herdr Role-Agent Operating Model

## Status

**Accepted on 2026-08-18 by the Sole Human Owner / Human Product and Release Authority.**

**Amendment notice:** ADR-0006 supersedes clauses 5 and 12 for evidence-gated GitHub operations. ADR-0007 governs pull-request routing, local-first CI, Milestone-only Full CI, branch lifecycle, and workspace cleanup. All other protected boundaries in this ADR remain in force.

This decision approves the operating model for current planning and controlled development preparation. It does not authorize production use, customer-data use, legal or compliance claims, a private alpha, a pilot, a release, provider-route activation, or GitHub execution without the applicable later gate.

## Context

The OSHE project is being created and governed by one human working with Herdr Terminal and multiple AI Services. The documentation baseline previously used role labels such as Project Manager, Product Owner, Architecture Owner, Code Lead, Security Lead, Test Lead, Documentation Reviewer, and Legal Content Owner. Those labels could be misread as a staffed human organization even though the project is operated by one human.

The project needs specialized planning, architecture, engineering, data, security, privacy, product-safety, testing, documentation, research, review, release-evidence, and customer-success capabilities. It also needs separation between change-producing work and independent review, bounded use of different AI Services, traceable data and tool authority, visible cost and quota controls, and final human control over protected decisions.

Treating an AI provider, model alias, autonomous agent, hidden provider-native subagent, service account, or passing workflow as the accountable project owner would weaken authority, evidence, privacy, security, safety, continuity, and residual-risk control. Conversely, requiring a permanently staffed human team would not reflect the actual operating model and would create false planning dependencies.

## Decision

1. **Use one Sole Human Owner as the final accountable authority.** The Sole Human Owner is the Human Product and Release Authority, primary maintainer, owner of project credentials and subscriptions, and final approver for protected decisions.

2. **Use Herdr Terminal role agents for operational project roles.** Planning and delivery role names represent controlled agent roles rather than additional employees or permanently named individuals.

3. **Adopt the baseline role catalog:**
   - Project Management Agent;
   - Product Planning Agent;
   - Architecture Agent;
   - Engineering Agent;
   - Data and Integration Agent;
   - Security Privacy and Product Safety Agent;
   - Test and Quality Agent;
   - Documentation and Configuration Agent;
   - Research and Legal Content Agent;
   - Independent Review and Challenge Agent;
   - Release and Evidence Agent;
   - Implementation and Customer Success Planning Agent.

   Additional specialist roles may be instantiated when an approved mission requires them.

4. **Assign roles dynamically through a governed role registry.** A role name alone does not grant authority. Every active assignment must resolve to a role-registry entry and record the agent or session identity, provider, model or runtime, CLI or adapter, role-card version, skill bundle, allowed data classes, allowed tools, artifact or path scope, read/write mode, task contract, output contract, reviewer, quota or usage boundary, timeout, activation state, and expiry.

5. **Keep protected authority human-only.** Only the Sole Human Owner may approve:
   - project mission, product direction, and protected product scope;
   - residual risk, exceptions, compensating controls, and release decisions;
   - credentials, destructive actions, materially expanded tool or data access, and provider-route activation;
   - production use, customer-data use, private-alpha entry, pilot participation, and public release;
   - legal, regulatory, medical, certification, contractual, or compliance claims;
   - final safety-critical operating decisions;
   - material recurring or high-cost service expansion;
   - calendar commitments and GitHub execution authorization.

6. **Allow agents to prepare and execute bounded work but not to become accountable humans.** Agents may research approved sources, plan, design, implement, test, review, compare evidence, prepare recommendations, maintain controlled registers, and assemble decision packages within their assignment. Agents may not appoint humans, approve themselves, accept residual risk, use unapproved credentials, expand their own authority, make protected OSHE decisions, or declare a release qualified.

7. **Require independent review according to risk.** Material outputs must be reviewed by a separate Independent Review and Challenge Agent or another assigned review role with read-only authority where practical. Higher-risk reviews should use a distinct provider, model family, adapter, prompt/role configuration, or evaluation route when available. Disagreement is preserved as evidence and resolved by the Sole Human Owner.

8. **Prohibit hidden or unregistered delegation.** Every agent, session, process, provider-native subagent, and delegated task must be visible to the control plane or explicitly prohibited. Hidden work must not create authoritative source, write access, approval, or evidence.

9. **Separate AI-service eligibility from role authority.** A provider or model brand is not an approved route. Each enabled route must have a service and route registry entry covering identity, purpose, assigned roles, allowed data classes, tools, endpoint or region where relevant, data-policy status, quota, concurrency, usage evidence, cost class, stop conditions, failover, expiry, and disable/exit procedure.

10. **Use variable usage-based cost governance rather than an invented fixed budget.** No fixed development budget or currency amount is established at this stage. Every mission uses an approved route and bounded quota or usage control. Unknown price and future consumption remain `UNKNOWN`. Material cost expansion requires Sole Human Owner approval.

11. **Assign calendar dates separately after planning completion.** Missing delivery dates are not planning defects. The Sole Human Owner sets dates only after the required planning artifacts, role-agent registry, service-route baseline, work-package decomposition, dependencies, and sufficient mission-throughput evidence exist.

12. **Keep GitHub execution deferred until explicit authorization.** Existing repository and control packages remain preparation evidence only. No agent may create or change the GitHub organization, repositories, rulesets, protected settings, secrets, releases, or hosted workflows until the Sole Human Owner explicitly authorizes the applicable activity.

13. **Use real humans where real-world evidence or professional accountability is required.** Agent simulations may test research instruments, fixtures, scenarios, and hypotheses, but they do not count as customer evidence, user-behavior evidence, usability proof, legal review, medical review, certification, or pilot sponsorship. Representative users and qualified external human specialists are introduced only when the applicable gate requires them.

14. **Apply sole-owner continuity limitations and compensating controls.** AI agents, service accounts, providers, and automation cannot become the human owner, Recovery Owner, or legal successor. Until a later continuity decision adds another accountable human or an accepted recovery arrangement, the project must:
   - keep authoritative planning and evidence exportable;
   - maintain provider-supported account and credential recovery controlled by the Sole Human Owner;
   - avoid undocumented local-only state;
   - preserve backups, revisions, manifests, and reconstruction instructions appropriate to the current risk;
   - fail closed when human authority is unavailable;
   - prohibit production, customer-data, protected operational, or irreversible commitments whose continuity requirements are not satisfied.

15. **Use event-based controlled work cycles.** Work begins from an approved mission and task contract and ends only when outputs, deterministic checks, independent review, evidence, unresolved questions, service usage, and required human decisions are collected. Agent concurrency is limited by non-overlapping authority and write scope, provider quotas, evidence capacity, and the Sole Human Owner's decision bandwidth.

## Alternatives Considered

- Permanently assign all planning and delivery roles to separate named human employees.
- Treat one general-purpose AI agent as autonomous project owner and final approver.
- Allow provider-native hidden subagents or unregistered terminal sessions.
- Bind each project role permanently to one provider or model.
- Use agents only as informal chat assistants while the Sole Human Owner performs every operational task manually.
- Treat a passing automated workflow or aggregate score as sufficient approval.

## Rationale

- Reflects the actual project organization without inventing staff, dates, or budget.
- Preserves one clear human accountability chain for safety, privacy, legal, product, financial, and release decisions.
- Allows specialist work to be performed through replaceable, provider-neutral role contracts.
- Enables independent challenge and review without confusing agent review with human approval.
- Makes data, tool, write, provider, quota, and cost authority explicit for every assignment.
- Supports auditability, reproducibility, fail-closed behavior, and future migration between AI Services.
- Prevents provider branding, model aliases, hidden delegation, or automation from silently becoming project authority.

## Positive Consequences

- Operational roles can scale or change without falsely changing human accountability.
- Multiple AI Services can be selected by mission, data class, skill, cost, and assurance need.
- Agent assignments, outputs, reviews, evidence, and usage can be registered and compared.
- Higher-risk work can receive separate-agent or cross-provider challenge.
- Dates and financial forecasts are added only when evidence supports them.
- Deferred GitHub work remains visible without blocking planning completion.

## Negative Consequences and Trade-offs

- The Sole Human Owner remains a concentration of authority and a potential continuity bottleneck.
- Agent-role registries, assignment schemas, service-route registries, and evidence controls add planning overhead.
- Independent agent review is not equivalent to qualified external human review.
- Real-user, legal, medical, certification, customer, and production gates cannot be completed solely by simulated agents.
- Mission throughput may exceed the Sole Human Owner's review and decision capacity, requiring throttling.
- Variable service pricing creates uncertainty until actual usage data exists.

## Mandatory Constraints

- Every active agent and delegated task is registered, scoped, time-bounded, and attributable.
- Every change-producing assignment has an allowed artifact or path scope and an output contract.
- Every material output has the review required by its risk class.
- No agent approves its own exception, residual risk, protected action, or release qualification.
- No provider route is enabled before its identity, data policy, authority, quota, and stop behavior are approved.
- No live secret is stored in planning documents, role cards, prompts, evidence bundles, or registries.
- No agent may silently widen data class, network, tool, credential, path, or decision authority.
- No hidden provider-native delegation may contribute authoritative work.
- No real-user, legal, medical, certification, or customer claim is based solely on agent simulation.
- No calendar or fixed-budget commitment is inferred from planning detail.
- No GitHub action is inferred from prepared documentation.
- Failed missions, rejected outputs, unresolved findings, and superseded evidence remain retained and linked.

## Required Control Artifacts

Before operational role dispatch beyond planning-only work, the project must establish:

- `herdr-role-registry.yaml`;
- approved role cards for the baseline roles;
- an agent-assignment schema;
- an AI service and route registry;
- populated model/runtime entries for selected routes;
- mission, task, result, review, integration, and handoff contracts;
- an integrated planning control register for assignments, decisions, RAID, gates, findings, evidence, usage, changes, and deferrals.

These artifacts implement this ADR but do not replace Sole Human Owner approval.

## Validation

- The Decision Register records DEC-008 and links this ADR.
- The Open Questions Register resolves OQ-014 through this ADR.
- The Planning Completion and Cross-Document Reconciliation Register marks PCR-006 and P0-A complete.
- The Implementation Readiness Gap Register removes the missing operating-model ADR from the remaining planning-control blockers.
- Current planning documents use Herdr role-agent ownership labels with explicit Sole Human Owner approval points.
- Future agent and service registries validate assignments against this ADR.
- Higher-risk reference missions include a distinct review assignment and human decision evidence.
- Material exceptions require a new or superseding ADR/RFC, impact assessment, compensating controls, and Sole Human Owner approval.

## Review Triggers

- The Sole Human Owner, credential ownership, legal ownership, or protected authority changes.
- A second accountable human, Recovery Owner, custodian, partner, employee, or customer operator is introduced.
- Herdr, provider, model, CLI, adapter, skill, tool, hidden delegation, orchestration, or evidence behavior materially changes.
- A provider route processes a new data class or gains materially broader tool, network, credential, write, or cost authority.
- Production, customer data, legal content release, medical data, certification, pilot, marketplace, or safety-critical operations enter scope.
- Independent agent review cannot provide the required assurance or conflicts of interest cannot be controlled.
- Mission volume exceeds the Sole Human Owner's safe decision and review capacity.
- A continuity, recovery, account-loss, security, privacy, safety, cost, or evidence incident shows that compensating controls are insufficient.
