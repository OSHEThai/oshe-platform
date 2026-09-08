---
document_id: RFC-20260908-001
title: 'Tenant-Scoped Store Boundaries: Hardening In-Memory Registries Against Cross-Tenant Collisions and State Leaks'
document_type: request_for_comments
document_version: 1.0.0
lifecycle_status: DRAFT
maturity: PROPOSED
implementation_status: IN_PROGRESS
review_status: REVIEW_REQUIRED
owner: Architecture and Data Lead
reviewers:
- Security Privacy and Product Safety Lead
- Test and Quality Lead
applicable_releases:
- v0.3.0
- v0.4.0
- v0.5.0
effective_date: '2026-09-08'
last_reviewed_date: '2026-09-08'
source_of_truth: LOCAL_GIT
classification: INTERNAL
change_risk: R2
related_decisions:
- HDEC-GITHUB-AUTHORITY-AND-REVIEW-EFFICIENCY-030
- HDEC-SPEND-ALLOWED-HERDR-DISPATCH-045
- HDEC-V040-FOUNDATION-054
---

# RFC: Tenant-Scoped Store Boundaries

## 1. Executive Summary & Problem Statement

During the whole-project architecture and security audit (`AUDIT-20260908-ARCH`), critical multi-tenant isolation defects were identified in several in-memory operational and evidentiary registries within `OSHEThai/oshe-platform`:

1. **`workflowaction.ActionManager` (`modules/workflow-action/action_lifecycle.go`)**:
   - Stores actions in `actions map[string]*action` keyed globally by `actionID`.
   - `CreateAction` rejects duplicates globally across all tenants (`ErrDuplicateActionID`), enabling cross-tenant ID squatting / denial-of-service.
   - Core operational methods (`GetAction`, `StartWork`, `SubmitForReview`, `RejectReview`, `CloseAction`, `ReopenAction`, `CheckOverdue`) accept only `actionID` and caller subject strings, lacking any `tenantID` scoping. An actor in Tenant B knowing or guessing an action ID from Tenant A can inspect its full history or execute lifecycle transitions.
2. **`workflowaction.ActionGovernanceEngine` (`modules/workflow-action/action_governance.go`)**:
   - Stores actions in `actions map[string]*GovernedAction` keyed globally by `actionID`.
   - Methods (`GetAction`, `ReassignOwner`, `RevokeOwner`, `RequestExtension`, `ReviewExtension`, `RequestEscalation`, `AcknowledgeEscalation`, `SubmitEvidence`, `ReviewEvidence`) lack tenant scoping.
3. **`recordsaudit.IntegrityRegistry` (`modules/records-audit/immutable_objects.go`)**:
   - Stores original and derived evidence objects in `originals map[string]OriginalRecord` and `derived map[string]DerivedRecord` keyed solely by `objectID`.
   - Rejects duplicate registrations globally (`ErrDuplicateObjectID`), preventing legitimate same-ID usage across distinct tenants.
   - Lifecycle state changes (`AcceptOriginal`, `ArchiveOriginal`, `AcceptDerived`) take only `objectID` without verifying tenant ownership, permitting cross-tenant lifecycle mutation.
4. **`recordsaudit.RecordStore` (`modules/records-audit/record_audit.go`)**:
   - Queries (`GetAuditTrail`, `GetSnapshots`, `GetRecord`) iterate through all tenants when a record is missing, returning `ErrCrossTenantAccess` if found elsewhere, creating a cross-tenant existence enumeration oracle.

## 2. Hardening Principles (Zero New Business Rules)

In accordance with PM task instructions, this RFC establishes architectural and data-boundary hardening without introducing unapproved business rules:

1. **Mandatory Tenant Scoping on All Reads and Mutations:**
   - Every lookup, query, state transition, and attachment requires an explicit trusted `tenantID`.
   - Lookups default-deny with non-leaking errors (`ErrActionNotFound`, `ErrRecordNotFound`, `ErrUnknownParent`) when an entity does not exist within the caller's tenant scope.
2. **Collision-Safe Multi-Tenant Storage:**
   - In-memory registries must partition state by `tenantID` either using structured composite keys (`type actionKey struct { tenantID, actionID string }`) or nested maps (`map[string]map[string]*action`).
   - Two tenants may legitimately register identical local identifiers (e.g. `act_001` or `orig_001`) without collision, rejection, or cross-tenant interference.
3. **Preservation of Immutability and Segregation of Duties:**
   - All existing lifecycle states (`ASSIGNED`, `IN_PROGRESS`, `IN_REVIEW`, `REJECTED`, `OVERDUE`, `CLOSED`, `REOPENED`), optimistic concurrency (`StateVersion`), append-only audit histories, and Segregation of Duties (SOD) self-approval blocks remain strictly preserved.
4. **Caller Boundary Isolation:**
   - Callers outside the leased modules (`modules/workflow-action/` and `modules/records-audit/`) are surveyed. `apps/api/walking_skeleton.go` (Lane A) maintains an isolated ad-hoc store and is not modified in Lane B.

## 3. Detailed Interface Changes

### 3.1 `modules/workflow-action/action_lifecycle.go`
- Refactor internal storage:
  ```go
  type actionKey struct {
      tenantID string
      actionID string
  }
  type ActionManager struct {
      mu      sync.RWMutex
      actions map[actionKey]*action
      clock   Clock
  }
  ```
- Method Signatures Updated:
  ```go
  func (m *ActionManager) GetAction(tenantID, id string) (ActionSnapshot, error)
  func (m *ActionManager) StartWork(tenantID, actionID, callerIdentity string) error
  func (m *ActionManager) AttachEvidence(tenantID, actionID, callerIdentity string, ev EvidenceAttachment) error
  func (m *ActionManager) SubmitForReview(tenantID, actionID, callerIdentity, notes string) error
  func (m *ActionManager) RejectReview(tenantID, actionID, callerIdentity, reason string) error
  func (m *ActionManager) CloseAction(tenantID, actionID, callerIdentity, notes string) error
  func (m *ActionManager) ReopenAction(tenantID, actionID, callerIdentity, reason string) error
  func (m *ActionManager) CheckOverdue(tenantID, actionID string) (bool, error)
  ```

### 3.2 `modules/workflow-action/action_governance.go`
- Refactor internal storage:
  ```go
  type ActionGovernanceEngine struct {
      mu      sync.RWMutex
      clock   Clock
      actions map[actionKey]*GovernedAction
  }
  ```
- Method Signatures Updated:
  ```go
  func (e *ActionGovernanceEngine) GetAction(tenantID, actionID string) (GovernedAction, error)
  func (e *ActionGovernanceEngine) ReassignOwner(tenantID, actionID, newOwner, newRole, callerSubject, reason string, expectedVersion int64) (GovernedAction, error)
  func (e *ActionGovernanceEngine) RevokeOwner(tenantID, actionID, callerSubject, reason string, expectedVersion int64) (GovernedAction, error)
  func (e *ActionGovernanceEngine) RequestExtension(tenantID string, req ExtensionRequest, expectedVersion int64) (GovernedAction, error)
  func (e *ActionGovernanceEngine) ReviewExtension(tenantID, actionID, requestID, reviewerSubject string, approve bool, notes string, expectedVersion int64) (GovernedAction, error)
  func (e *ActionGovernanceEngine) RequestEscalation(tenantID string, req EscalationRequest, expectedVersion int64) (GovernedAction, error)
  func (e *ActionGovernanceEngine) AcknowledgeEscalation(tenantID, actionID, requestID, reviewerSubject, notes string, expectedVersion int64) (GovernedAction, error)
  func (e *ActionGovernanceEngine) SubmitEvidence(tenantID, actionID string, ev GovernedEvidence, expectedVersion int64) (GovernedAction, error)
  func (e *ActionGovernanceEngine) ReviewEvidence(tenantID, actionID, evidenceID, reviewerSubject string, accept bool, notes string, expectedVersion int64) (GovernedAction, error)
  ```

### 3.3 `modules/records-audit/immutable_objects.go`
- Refactor internal storage:
  ```go
  type objectKey struct {
      tenantID string
      objectID string
  }
  type IntegrityRegistry struct {
      mu        sync.RWMutex
      originals map[objectKey]OriginalRecord
      derived   map[objectKey]DerivedRecord
  }
  ```
- Method Signatures Updated:
  ```go
  func (reg *IntegrityRegistry) RegisterOriginal(rec OriginalRecord) error
  func (reg *IntegrityRegistry) AcceptOriginal(tenantID, objectID string) error
  func (reg *IntegrityRegistry) ArchiveOriginal(tenantID, objectID string) error
  func (reg *IntegrityRegistry) RegisterDerived(rec DerivedRecord) error
  func (reg *IntegrityRegistry) AcceptDerived(tenantID, objectID string) error
  func (reg *IntegrityRegistry) VerifyIntegrityLinkage(derivedID, callerTenantID, expectedDigest string) (IntegrityLinkage, error)
  ```

### 3.4 `modules/records-audit/record_audit.go`
- Eliminate Cross-Tenant Existence Oracle:
  - Remove `for _, r := range s.records { if r.RecordID == tRecordID && r.TenantID != tCaller ... }` loops in `GetAuditTrail`, `GetSnapshots`, and `GetRecord`.
  - Return `ErrRecordNotFound` immediately when key `makeRecordKey(tCaller, tRecordID)` does not exist.

## 4. Cross-Lane Impact & External Callers
- Callers surveyed: `reinspection_lifecycle.go` (inside `modules/workflow-action/`) is adapted to pass `order.TenantID` to `e.actionEngine.GetAction`.
- `apps/api/walking_skeleton.go` (Lane A) does not consume these modules and requires zero changes from Lane B.

## 5. Security & Verification Plan
- Unit test suite adaptation in `action_lifecycle_test.go`, `action_governance_test.go`, `reinspection_lifecycle_test.go`, `immutable_objects_test.go`, and `record_audit_test.go`.
- Added multi-tenant collision tests asserting that Tenant A and Tenant B can concurrently manage entities with identical local IDs without cross-tenant leakage or mutation.
