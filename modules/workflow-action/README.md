# Workflow and Action

- Module ID: `MOD-WFA`
- Roadmap topic: `V020-T04`
- Implementation state: architecture scaffold only

Owns workflow instances, transitions, findings, actions, due dates, reviews, reopen decisions, and closure state. It may reference approved configuration versions, authorization decisions, and evidence links.

Protected transitions and closure cannot use last-write-wins or be inferred from generated reports, notifications, or AI output.
