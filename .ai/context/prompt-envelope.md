# Prompt Envelope

```text
[PROJECT CONTRACT]
AGENTS.md

[ROLE]
.ai/roles/cards/<role-id>.md

[SPECIALIST PROFILE]
.ai/agents/profiles/<profile-id>.md or NONE

[MISSION]
.ai/missions/<mission-id>/mission.yaml

[TASK]
.ai/missions/<mission-id>/<task-id>.yaml

[ACTIVE SKILLS]
<resolved skill name, version, digest>

[PROVIDER ROUTE]
<approved exact route id or DISABLED>

[TOOL PROFILE]
<profile id>

[WRITE LEASE]
<allowed paths>

[OUTPUT CONTRACT]
.ai/schemas/result.schema.json
```

The assignment must validate against `.ai/schemas/agent-assignment.schema.json`. A specialist profile cannot widen the role, route, data, tool, path, review, quota, timeout, or human-approval envelope.
