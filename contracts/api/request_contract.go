// Package api provides provisional static API contract models for v0.2.0.
//
// PROVISIONAL GOVERNANCE DECLARATION:
// The request-contract fixtures in this file are provisional, local-only
// specification models pending formal Sole Human Owner architecture gate H020-005.
// Zero HTTP server binding, endpoint registration, database persistence,
// compatibility certification, or runtime behavior is claimed or granted.
package api

import (
	"errors"
	"fmt"
	"strings"
)

const (
	// CurrentContractVersion is the canonical version identifier for v0.2.0 request contracts.
	CurrentContractVersion = "v1"

	// ProvisionalStatus declares the governance state pending H020-005.
	ProvisionalStatus = "PROVISIONAL_PENDING_H020_005"

	// DefaultPaginationLimit is the default result count when unspecified.
	DefaultPaginationLimit = 20
	// MaxPaginationLimit is the strict upper bound on page size.
	MaxPaginationLimit = 100
)

var (
	// ErrBlankTenantScope indicates that the tenant identifier is empty or whitespace-only.
	ErrBlankTenantScope = errors.New("tenant scope cannot be blank")
	// ErrInvalidTenantScope indicates that the tenant identifier lacks the canonical ten_ prefix.
	ErrInvalidTenantScope = errors.New("tenant scope must have valid prefix ten_")
	// ErrBlankCorrelationID indicates that the correlation ID is empty or whitespace-only.
	ErrBlankCorrelationID = errors.New("correlation_id cannot be blank")
	// ErrInvalidCorrelationID indicates that the correlation ID lacks the canonical corr_ prefix.
	ErrInvalidCorrelationID = errors.New("correlation_id must have valid prefix corr_")
	// ErrUnsupportedVersion indicates an unrecognized or mismatched contract version.
	ErrUnsupportedVersion = errors.New("unsupported contract version: must be v1")
	// ErrInvalidPagination indicates that pagination parameters violate boundary constraints.
	ErrInvalidPagination = errors.New("malformed pagination parameters")
	// ErrStaleConcurrencyToken indicates that the provided revision does not match the current resource revision.
	ErrStaleConcurrencyToken = errors.New("optimistic concurrency conflict: stale concurrency token")
	// ErrDuplicateIdempotencyKey indicates a conflicting request payload reusing an idempotency key.
	ErrDuplicateIdempotencyKey = errors.New("idempotency key conflict: payload mismatch")
	// ErrContradictoryMetadata indicates conflicting scope, tenant, or request headers.
	ErrContradictoryMetadata = errors.New("contradictory request metadata: caller and target scope conflict")
)

// PaginationParams encapsulates deterministic cursor-based pagination parameters.
type PaginationParams struct {
	Limit  int    `json:"limit"`
	Cursor string `json:"cursor,omitempty"`
}

// Validate ensures pagination limits remain within stable boundaries [1, MaxPaginationLimit].
func (p *PaginationParams) Validate() error {
	if p.Limit <= 0 || p.Limit > MaxPaginationLimit {
		return fmt.Errorf("%w: limit %d must be between 1 and %d", ErrInvalidPagination, p.Limit, MaxPaginationLimit)
	}
	if strings.ContainsAny(p.Cursor, " \t\r\n") {
		return fmt.Errorf("%w: cursor contains illegal whitespace characters", ErrInvalidPagination)
	}
	return nil
}

// ConcurrencyToken encapsulates optimistic concurrency control tokens.
type ConcurrencyToken struct {
	ETag             string `json:"etag,omitempty"`
	ExpectedRevision int    `json:"expected_revision"`
}

// Validate ensures the concurrency token has a non-negative revision.
func (c *ConcurrencyToken) Validate() error {
	if c.ExpectedRevision <= 0 {
		return fmt.Errorf("%w: expected revision must be greater than zero", ErrStaleConcurrencyToken)
	}
	return nil
}

// IdempotencyParams captures caller idempotency constraints.
type IdempotencyParams struct {
	Key           string `json:"key"`
	PayloadDigest string `json:"payload_digest"`
}

// Validate ensures idempotency key and digest are non-empty and well-formed.
func (i *IdempotencyParams) Validate() error {
	trimmedKey := strings.TrimSpace(i.Key)
	if trimmedKey == "" {
		return errors.New("idempotency key cannot be blank")
	}
	trimmedDigest := strings.TrimSpace(i.PayloadDigest)
	if len(trimmedDigest) != 64 {
		return errors.New("payload digest must be a 64-character SHA-256 hex string")
	}
	return nil
}

// RequestEnvelope defines the canonical versioned wire structure for incoming API commands and queries.
type RequestEnvelope struct {
	Version       string             `json:"version"`
	TenantID      string             `json:"tenant_id"`
	CorrelationID string             `json:"correlation_id"`
	CausationID   string             `json:"causation_id,omitempty"`
	Pagination    *PaginationParams  `json:"pagination,omitempty"`
	Concurrency   *ConcurrencyToken  `json:"concurrency,omitempty"`
	Idempotency   *IdempotencyParams `json:"idempotency,omitempty"`
}

// NewRequestEnvelope constructs and validates a base RequestEnvelope.
func NewRequestEnvelope(tenantID, correlationID string) (*RequestEnvelope, error) {
	env := &RequestEnvelope{
		Version:       CurrentContractVersion,
		TenantID:      strings.TrimSpace(tenantID),
		CorrelationID: strings.TrimSpace(correlationID),
	}
	if err := env.Validate(); err != nil {
		return nil, err
	}
	return env, nil
}

// Validate ensures all mandatory envelope fields conform to canonical constraints.
func (e *RequestEnvelope) Validate() error {
	if strings.TrimSpace(e.Version) != CurrentContractVersion {
		return fmt.Errorf("%w: expected %q, got %q", ErrUnsupportedVersion, CurrentContractVersion, e.Version)
	}

	trimmedTenant := strings.TrimSpace(e.TenantID)
	if trimmedTenant == "" {
		return ErrBlankTenantScope
	}
	if !strings.HasPrefix(trimmedTenant, "ten_") {
		return fmt.Errorf("%w: got %q", ErrInvalidTenantScope, trimmedTenant)
	}

	trimmedCorr := strings.TrimSpace(e.CorrelationID)
	if trimmedCorr == "" {
		return ErrBlankCorrelationID
	}
	if !strings.HasPrefix(trimmedCorr, "corr_") {
		return fmt.Errorf("%w: got %q", ErrInvalidCorrelationID, trimmedCorr)
	}

	if e.CausationID != "" {
		trimmedCaus := strings.TrimSpace(e.CausationID)
		if !strings.HasPrefix(trimmedCaus, "caus_") {
			return fmt.Errorf("causation_id must have valid prefix caus_: got %q", trimmedCaus)
		}
	}

	if e.Pagination != nil {
		if err := e.Pagination.Validate(); err != nil {
			return err
		}
	}

	if e.Concurrency != nil {
		if err := e.Concurrency.Validate(); err != nil {
			return err
		}
	}

	if e.Idempotency != nil {
		if err := e.Idempotency.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// EvaluateConcurrency compares an incoming token against the current resource revision.
// Fails closed if token is nil, revision is stale, or mismatch is detected.
func EvaluateConcurrency(token *ConcurrencyToken, currentRevision int) error {
	if token == nil {
		return errors.New("concurrency token is required for protected state modifications")
	}
	if err := token.Validate(); err != nil {
		return err
	}
	if token.ExpectedRevision != currentRevision {
		return fmt.Errorf("%w: expected revision %d, current resource revision is %d",
			ErrStaleConcurrencyToken, token.ExpectedRevision, currentRevision)
	}
	return nil
}

// EvaluateIdempotencyReplay determines whether a candidate request is an identical replay or a conflicting collision.
func EvaluateIdempotencyReplay(existingKey, existingDigest, candidateKey, candidateDigest string) (isReplay bool, err error) {
	tExistKey := strings.TrimSpace(existingKey)
	tCandKey := strings.TrimSpace(candidateKey)
	if tExistKey == "" || tCandKey == "" || tExistKey != tCandKey {
		return false, errors.New("mismatched idempotency keys")
	}

	tExistDigest := strings.TrimSpace(existingDigest)
	tCandDigest := strings.TrimSpace(candidateDigest)

	if tExistDigest == tCandDigest {
		return true, nil
	}
	return false, fmt.Errorf("%w: existing digest %s, candidate digest %s",
		ErrDuplicateIdempotencyKey, tExistDigest, tCandDigest)
}

// ValidateScopeMatch enforces that caller tenant scope strictly matches target resource tenant.
func ValidateScopeMatch(callerTenantID, targetTenantID string) error {
	tCaller := strings.TrimSpace(callerTenantID)
	tTarget := strings.TrimSpace(targetTenantID)
	if tCaller == "" || tTarget == "" || tCaller != tTarget {
		return fmt.Errorf("%w: caller %q cannot access resource in tenant %q",
			ErrContradictoryMetadata, tCaller, tTarget)
	}
	return nil
}
