# Internal Portal Scopes, Role Navigation, and Task Views

- Module ID: `MOD-PORTAL`
- Roadmap topic: `V030-T06` (GitHub Issue #98 / V030-I025)
- Implementation state: local standalone synthetic specification module

Owns synthetic internal portal audience resolution, role-scoped navigation menus, bounded work queues, and authorized content presentation for internal operational users.

## Architectural Boundaries & Non-Claims

- **Provisional In-Memory Model Only:** This module operates as a dependency-free, standalone local Go package (`oshe/internal-portal`). It does NOT deploy a web server, HTTP router, frontend application, or public network service.
- **Synthetic Roles & Contexts Only:** Operates strictly on synthetic role/context fixtures (`PortalViewer`). Zero real user/customer data or external identity providers are connected.
- **Zero Authority Enactment:** Portal views are descriptive presentation projections only. They do not grant, evaluate, or mutate security policies, permissions, or session tokens.
- **Held Governance Gates:** Decision `H030-006` (Portal Staging & Publication), `H030-007` (Public Routes & CDN), and `H030-008` (Release & Deployment) remain strictly on **HOLD**.

## Core Capabilities (`portal.go`)

- **Role-Scoped Navigation Menus (`ResolveNavigation`):** Dynamically resolves navigation items strictly filtered to the viewer's active role (`RoleTenantAdmin`, `RoleProjectManager`, `RoleInspector`, `RoleAuditor`, `RoleViewer`, `RoleContractor`). Prevents lower-privilege roles from discovering administrative or peer contractor surfaces (`NEG-PORTAL-01`).
- **Exact-Scope Work Queue Bounding (`ResolveWorkQueue`):** Partitions operational work queue tasks strictly to the caller's authoritative tenant and project scope. Sibling project tasks are filtered out (`NEG-PORTAL-02`), cross-tenant leakage is prevented (`NEG-PORTAL-03`), and unassigned scopes return a clean, non-leaking empty state (`EmptyStateNotice`).
- **Authorized Content Presentation (`ViewContent`):** Evaluates content retrieval against tenant, project, site, and role requirements. Cross-tenant, cross-project, or unauthorized role queries fail closed with generic `ErrAccessDenied`, strictly preventing object existence confirmation (`NEG-PORTAL-04`).
- **Accessibility & Usability Enforcement:** Mandates non-empty human-readable text labels and descriptive ARIA labels (`ValidateAccessibility`, `ErrEmptyLabel`, `ErrEmptyAriaLabel`) for all navigation items (`NEG-PORTAL-05`).
