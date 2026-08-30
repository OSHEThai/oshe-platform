# Identity and Authorization

- Module ID: `MOD-IAM`
- Roadmap topic: `V020-T02`
- Implementation state: architecture scaffold only

Owns identity references, memberships, role assignments, scoped authorization evaluation, and revocation evidence. It may depend on stable organization identifiers from `MOD-ORG`.

Entitlement does not replace authorization. Clients, integrations, extensions, and AI agents cannot supply or bypass authoritative tenant and scope decisions.
