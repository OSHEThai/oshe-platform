package localidentity

import (
	"errors"
	"strings"
	"sync"
	"time"
)

var (
	// ErrBlankReason indicates that an audit reason is missing or empty.
	ErrBlankReason = errors.New("revocation reason cannot be blank")
	// ErrInvalidGeneration indicates that a generation number is negative or invalid.
	ErrInvalidGeneration = errors.New("generation must be greater than or equal to zero")
)

// DiagnosticDenialCategory represents stable, explainable, high-level denial categories
// that strictly prevent information leakage regarding roles, memberships, or foreign tenant scopes.
type DiagnosticDenialCategory string

const (
	CategoryNone                 DiagnosticDenialCategory = "NONE"
	CategorySessionRevoked       DiagnosticDenialCategory = "SESSION_REVOKED"
	CategorySessionStale         DiagnosticDenialCategory = "SESSION_STALE"
	CategoryPolicyStale          DiagnosticDenialCategory = "POLICY_GENERATION_STALE"
	CategoryCrossTenantDenied    DiagnosticDenialCategory = "CROSS_TENANT_ACCESS_DENIED"
	CategoryIdentityInactive     DiagnosticDenialCategory = "IDENTITY_INACTIVE"
	CategoryDefaultDeny          DiagnosticDenialCategory = "DEFAULT_DENY"
)

// DecisionDiagnostic encapsulates an explainable access decision that conveys only
// local status and decision generation, without leaking sensitive foreign context.
type DecisionDiagnostic struct {
	Allowed            bool                     `json:"allowed"`
	DenialCategory     DiagnosticDenialCategory `json:"denial_category"`
	DecisionGeneration int64                    `json:"decision_generation"`
	Summary            string                   `json:"summary"`
}

// RevocationAuditEventType defines categories of security revocation events.
type RevocationAuditEventType string

const (
	EventSessionRevoked        RevocationAuditEventType = "SESSION_REVOKED"
	EventSubjectRevoked        RevocationAuditEventType = "SUBJECT_REVOKED"
	EventPolicyGenerationBump  RevocationAuditEventType = "POLICY_GENERATION_BUMPED"
	EventStalenessEvaluated    RevocationAuditEventType = "STALENESS_EVALUATED"
)

// RevocationAuditEvent is an immutable, append-only security journal entry.
type RevocationAuditEvent struct {
	SequenceNumber int64                    `json:"sequence_number"`
	EventType      RevocationAuditEventType `json:"event_type"`
	TenantID       string                   `json:"tenant_id"`
	Subject        string                   `json:"subject,omitempty"`
	TokenDigest    [32]byte                 `json:"token_digest,omitempty"`
	Generation     int64                    `json:"generation"`
	Reason         string                   `json:"reason"`
	Timestamp      time.Time                `json:"timestamp"`
}

type sessionRevocationRecord struct {
	tokenDigest [32]byte
	tenantID    string
	subject     string
	generation  int64
	revokedAt   time.Time
}

type subjectRevocationRecord struct {
	tenantID   string
	subject    string
	generation int64
	revokedAt  time.Time
}

// RevocationRegistry provides thread-safe management of session revocations,
// policy generation tracking, and non-leaking decision diagnostics.
type RevocationRegistry struct {
	mu                 sync.RWMutex
	clock              Clock
	globalGeneration   int64
	revokedSessions    map[[32]byte]sessionRevocationRecord
	subjectRevocations map[string]subjectRevocationRecord // key: tenantID + ":" + subject
	tenantGenerations  map[string]int64                   // key: tenantID
	auditLog           []RevocationAuditEvent
	seqCounter         int64
}

// NewRevocationRegistry constructs an initialized in-memory revocation registry.
func NewRevocationRegistry(clock Clock) *RevocationRegistry {
	if clock == nil {
		clock = time.Now
	}
	return &RevocationRegistry{
		clock:              clock,
		globalGeneration:   1,
		revokedSessions:    make(map[[32]byte]sessionRevocationRecord),
		subjectRevocations: make(map[string]subjectRevocationRecord),
		tenantGenerations:  make(map[string]int64),
		auditLog:           make([]RevocationAuditEvent, 0),
	}
}

// RevokeSessionToken explicitly revokes a specific session by token digest.
func (r *RevocationRegistry) RevokeSessionToken(digest [32]byte, tenantID, subject, reason string) error {
	tTenant := strings.TrimSpace(tenantID)
	if tTenant == "" {
		return ErrBlankTenantID
	}
	tSub := strings.TrimSpace(subject)
	if tSub == "" {
		return ErrBlankSubject
	}
	tReason := strings.TrimSpace(reason)
	if tReason == "" {
		return ErrBlankReason
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.globalGeneration++
	now := r.clock().UTC()

	r.revokedSessions[digest] = sessionRevocationRecord{
		tokenDigest: digest,
		tenantID:    tTenant,
		subject:     tSub,
		generation:  r.globalGeneration,
		revokedAt:   now,
	}

	r.seqCounter++
	r.auditLog = append(r.auditLog, RevocationAuditEvent{
		SequenceNumber: r.seqCounter,
		EventType:      EventSessionRevoked,
		TenantID:       tTenant,
		Subject:        tSub,
		TokenDigest:    digest,
		Generation:     r.globalGeneration,
		Reason:         tReason,
		Timestamp:      now,
	})

	return nil
}

// RevokeSubject revokes all existing sessions for a subject up to the new revocation generation.
func (r *RevocationRegistry) RevokeSubject(tenantID, subject, reason string) error {
	tTenant := strings.TrimSpace(tenantID)
	if tTenant == "" {
		return ErrBlankTenantID
	}
	tSub := strings.TrimSpace(subject)
	if tSub == "" {
		return ErrBlankSubject
	}
	tReason := strings.TrimSpace(reason)
	if tReason == "" {
		return ErrBlankReason
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.globalGeneration++
	now := r.clock().UTC()
	key := tTenant + ":" + tSub

	r.subjectRevocations[key] = subjectRevocationRecord{
		tenantID:   tTenant,
		subject:    tSub,
		generation: r.globalGeneration,
		revokedAt:  now,
	}

	r.seqCounter++
	r.auditLog = append(r.auditLog, RevocationAuditEvent{
		SequenceNumber: r.seqCounter,
		EventType:      EventSubjectRevoked,
		TenantID:       tTenant,
		Subject:        tSub,
		Generation:     r.globalGeneration,
		Reason:         tReason,
		Timestamp:      now,
	})

	return nil
}

// BumpTenantPolicyGeneration increments the policy generation counter for a tenant,
// effectively invalidating any cached decisions or sessions issued under older policy generations.
func (r *RevocationRegistry) BumpTenantPolicyGeneration(tenantID, reason string) (int64, error) {
	tTenant := strings.TrimSpace(tenantID)
	if tTenant == "" {
		return 0, ErrBlankTenantID
	}
	tReason := strings.TrimSpace(reason)
	if tReason == "" {
		return 0, ErrBlankReason
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.globalGeneration++
	currentGen := r.tenantGenerations[tTenant]
	nextGen := currentGen + 1
	r.tenantGenerations[tTenant] = nextGen
	now := r.clock().UTC()

	r.seqCounter++
	r.auditLog = append(r.auditLog, RevocationAuditEvent{
		SequenceNumber: r.seqCounter,
		EventType:      EventPolicyGenerationBump,
		TenantID:       tTenant,
		Generation:     nextGen,
		Reason:         tReason,
		Timestamp:      now,
	})

	return nextGen, nil
}

// EvaluateSession verifies session validity, staleness, and policy-generation alignment.
// Returns an explainable, non-leaking DecisionDiagnostic.
func (r *RevocationRegistry) EvaluateSession(tokenDigest [32]byte, callerTenantID, targetTenantID, subject string, sessionGeneration int64) DecisionDiagnostic {
	tCaller := strings.TrimSpace(callerTenantID)
	tTarget := strings.TrimSpace(targetTenantID)

	r.mu.RLock()
	defer r.mu.RUnlock()

	currentGlobal := r.globalGeneration

	// 1. Cross-tenant check (fails closed immediately without disclosing foreign existence)
	if tCaller == "" || tTarget == "" || tCaller != tTarget {
		return DecisionDiagnostic{
			Allowed:            false,
			DenialCategory:     CategoryCrossTenantDenied,
			DecisionGeneration: currentGlobal,
			Summary:            "tenant context mismatch",
		}
	}

	tSub := strings.TrimSpace(subject)
	if tSub == "" {
		return DecisionDiagnostic{
			Allowed:            false,
			DenialCategory:     CategoryDefaultDeny,
			DecisionGeneration: currentGlobal,
			Summary:            "subject identifier required",
		}
	}

	// 2. Explicit session revocation check
	if _, revoked := r.revokedSessions[tokenDigest]; revoked {
		return DecisionDiagnostic{
			Allowed:            false,
			DenialCategory:     CategorySessionRevoked,
			DecisionGeneration: currentGlobal,
			Summary:            "session token has been explicitly revoked",
		}
	}

	// 3. Subject-level revocation check (staleness)
	subjKey := tCaller + ":" + tSub
	if subRev, exists := r.subjectRevocations[subjKey]; exists {
		if sessionGeneration <= subRev.generation {
			return DecisionDiagnostic{
				Allowed:            false,
				DenialCategory:     CategorySessionStale,
				DecisionGeneration: currentGlobal,
				Summary:            "session invalidated by subject revocation event",
			}
		}
	}

	// 4. Tenant policy-generation check (staleness)
	tenantPolicyGen := r.tenantGenerations[tCaller]
	if tenantPolicyGen > 0 && sessionGeneration < tenantPolicyGen {
		return DecisionDiagnostic{
			Allowed:            false,
			DenialCategory:     CategoryPolicyStale,
			DecisionGeneration: currentGlobal,
			Summary:            "session invalidated by tenant policy generation progression",
		}
	}

	return DecisionDiagnostic{
		Allowed:            true,
		DenialCategory:     CategoryNone,
		DecisionGeneration: currentGlobal,
		Summary:            "session is valid and active",
	}
}

// AuditTrail returns an immutable, tenant-scoped copy of revocation audit events.
// Disallows modification or deletion of any audit records.
func (r *RevocationRegistry) AuditTrail(callerTenantID string) ([]RevocationAuditEvent, error) {
	tCaller := strings.TrimSpace(callerTenantID)
	if tCaller == "" {
		return nil, ErrBlankTenantID
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	var events []RevocationAuditEvent
	for _, entry := range r.auditLog {
		if entry.TenantID == tCaller {
			events = append(events, entry)
		}
	}

	return events, nil
}
