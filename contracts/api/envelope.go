// Package api provides provisional static API contract models for v0.2.0.
// This package is an isolated, dependency-free local specification slice
// pending formal Sole Human Owner architecture gate H020-005.
// Zero runtime execution, server binding, or external standard compatibility is claimed.
package api

import (
	"encoding/json"
	"errors"
	"strings"
)

var (
	// ErrEmptyCode indicates that the error code is missing or whitespace-only.
	ErrEmptyCode = errors.New("error code cannot be empty")
	// ErrEmptyMessage indicates that the error message is missing or whitespace-only.
	ErrEmptyMessage = errors.New("error message cannot be empty")
	// ErrEmptyCorrelationID indicates that the correlation ID is missing or whitespace-only.
	ErrEmptyCorrelationID = errors.New("correlation_id cannot be empty")
)

// ErrorDetail represents an individual structured detail entry within an error envelope.
type ErrorDetail struct {
	Field   string `json:"field,omitempty"`
	Reason  string `json:"reason"`
	Message string `json:"message,omitempty"`
}

// ErrorEnvelope represents a canonical, deterministic, serializable API error response envelope.
type ErrorEnvelope struct {
	Code          string        `json:"code"`
	Message       string        `json:"message"`
	CorrelationID string        `json:"correlation_id"`
	Details       []ErrorDetail `json:"details"`
}

// NewErrorEnvelope constructs and validates an ErrorEnvelope.
func NewErrorEnvelope(code, message, correlationID string, details []ErrorDetail) (*ErrorEnvelope, error) {
	env := &ErrorEnvelope{
		Code:          strings.TrimSpace(code),
		Message:       strings.TrimSpace(message),
		CorrelationID: strings.TrimSpace(correlationID),
		Details:       details,
	}
	if env.Details == nil {
		env.Details = []ErrorDetail{}
	}
	if err := env.Validate(); err != nil {
		return nil, err
	}
	return env, nil
}

// Validate ensures all mandatory fields in the error envelope are non-empty.
func (e *ErrorEnvelope) Validate() error {
	if strings.TrimSpace(e.Code) == "" {
		return ErrEmptyCode
	}
	if strings.TrimSpace(e.Message) == "" {
		return ErrEmptyMessage
	}
	if strings.TrimSpace(e.CorrelationID) == "" {
		return ErrEmptyCorrelationID
	}
	return nil
}

// MarshalJSON returns the deterministic JSON encoding of the error envelope.
func (e *ErrorEnvelope) MarshalJSON() ([]byte, error) {
	if err := e.Validate(); err != nil {
		return nil, err
	}
	type Alias ErrorEnvelope
	details := e.Details
	if details == nil {
		details = []ErrorDetail{}
	}
	return json.Marshal(&struct {
		*Alias
		Details []ErrorDetail `json:"details"`
	}{
		Alias:   (*Alias)(e),
		Details: details,
	})
}

// UnmarshalJSON decodes and validates a JSON payload into the ErrorEnvelope.
func (e *ErrorEnvelope) UnmarshalJSON(data []byte) error {
	type Alias ErrorEnvelope
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(e),
	}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}
	if e.Details == nil {
		e.Details = []ErrorDetail{}
	}
	return e.Validate()
}
