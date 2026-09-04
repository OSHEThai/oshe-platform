package identifiers

import (
	"errors"
	"fmt"
)

var (
	// ErrInvalidCorrelationID indicates that a correlation ID is missing, malformed, or has an unexpected prefix.
	ErrInvalidCorrelationID = errors.New("invalid correlation identifier")
	// ErrInvalidCausationID indicates that a causation ID is missing, malformed, or has an unexpected prefix.
	ErrInvalidCausationID = errors.New("invalid causation identifier")
)

const (
	// PrefixCorrelation is the canonical prefix for correlation identifiers.
	PrefixCorrelation = "corr"
	// PrefixCausation is the canonical prefix for causation identifiers.
	PrefixCausation = "caus"
)

// GenerateCorrelationID generates a new strongly typed correlation identifier with prefix "corr".
func GenerateCorrelationID() (ID, error) {
	return Generate(PrefixCorrelation)
}

// GenerateCausationID generates a new strongly typed causation identifier with prefix "caus".
func GenerateCausationID() (ID, error) {
	return Generate(PrefixCausation)
}

// ParseCorrelationID parses and validates that raw is a valid identifier with prefix "corr".
func ParseCorrelationID(raw string) (ID, error) {
	id, err := ParseWithPrefix(raw, PrefixCorrelation)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidCorrelationID, err)
	}
	return id, nil
}

// ParseCausationID parses and validates that raw is a valid identifier with prefix "caus".
func ParseCausationID(raw string) (ID, error) {
	id, err := ParseWithPrefix(raw, PrefixCausation)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidCausationID, err)
	}
	return id, nil
}

// CorrelationContext captures tracking identifiers for operational causality without external protocol dependencies.
type CorrelationContext struct {
	CorrelationID ID `json:"correlation_id"`
	CausationID   ID `json:"causation_id"`
}

// NewCorrelationContext validates and constructs a CorrelationContext from raw string identifiers.
func NewCorrelationContext(correlationID, causationID string) (*CorrelationContext, error) {
	corr, err := ParseCorrelationID(correlationID)
	if err != nil {
		return nil, err
	}
	caus, err := ParseCausationID(causationID)
	if err != nil {
		return nil, err
	}
	return &CorrelationContext{
		CorrelationID: corr,
		CausationID:   caus,
	}, nil
}

// StartCorrelationContext initializes a new correlation root context with fresh correlation and causation IDs.
func StartCorrelationContext() (*CorrelationContext, error) {
	corr, err := GenerateCorrelationID()
	if err != nil {
		return nil, err
	}
	caus, err := GenerateCausationID()
	if err != nil {
		return nil, err
	}
	return &CorrelationContext{
		CorrelationID: corr,
		CausationID:   caus,
	}, nil
}

// DeriveChild creates a child tracking context that preserves the root CorrelationID and assigns a new CausationID.
func (c *CorrelationContext) DeriveChild() (*CorrelationContext, error) {
	nextCaus, err := GenerateCausationID()
	if err != nil {
		return nil, err
	}
	return &CorrelationContext{
		CorrelationID: c.CorrelationID,
		CausationID:   nextCaus,
	}, nil
}
