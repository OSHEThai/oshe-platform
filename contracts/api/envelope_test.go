package api_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/oshethai/oshe-platform/contracts/api"
)

func TestNewErrorEnvelope_Valid(t *testing.T) {
	details := []api.ErrorDetail{
		{Field: "email", Reason: "invalid_format", Message: "must be a valid email"},
	}
	env, err := api.NewErrorEnvelope("VALIDATION_FAILED", "request payload invalid", "corr_123456789", details)
	if err != nil {
		t.Fatalf("unexpected construction error: %v", err)
	}
	if env.Code != "VALIDATION_FAILED" {
		t.Errorf("expected code 'VALIDATION_FAILED', got %q", env.Code)
	}
	if env.Message != "request payload invalid" {
		t.Errorf("expected message 'request payload invalid', got %q", env.Message)
	}
	if env.CorrelationID != "corr_123456789" {
		t.Errorf("expected correlation_id 'corr_123456789', got %q", env.CorrelationID)
	}
	if len(env.Details) != 1 {
		t.Fatalf("expected 1 detail entry, got %d", len(env.Details))
	}
}

func TestNewErrorEnvelope_NilDetailsDefaultsToEmptySlice(t *testing.T) {
	env, err := api.NewErrorEnvelope("INTERNAL_ERROR", "an unexpected error occurred", "corr_987654321", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env.Details == nil {
		t.Fatal("expected details to be non-nil empty slice")
	}
	if len(env.Details) != 0 {
		t.Errorf("expected 0 details, got %d", len(env.Details))
	}
}

func TestErrorEnvelope_ValidationRejections(t *testing.T) {
	cases := []struct {
		name          string
		code          string
		message       string
		correlationID string
		expectedErr   error
	}{
		{"empty_code", "", "message", "corr_1", api.ErrEmptyCode},
		{"whitespace_code", "   ", "message", "corr_1", api.ErrEmptyCode},
		{"empty_message", "CODE", "", "corr_1", api.ErrEmptyMessage},
		{"whitespace_message", "CODE", "   ", "corr_1", api.ErrEmptyMessage},
		{"empty_correlation_id", "CODE", "message", "", api.ErrEmptyCorrelationID},
		{"whitespace_correlation_id", "CODE", "message", "   ", api.ErrEmptyCorrelationID},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := api.NewErrorEnvelope(tc.code, tc.message, tc.correlationID, nil)
			if err == nil {
				t.Fatalf("expected validation error, got nil")
			}
			if !errors.Is(err, tc.expectedErr) {
				t.Errorf("expected error %v, got %v", tc.expectedErr, err)
			}
		})
	}
}

func TestErrorEnvelope_JSONRoundTrip(t *testing.T) {
	original, err := api.NewErrorEnvelope(
		"NOT_FOUND",
		"requested resource does not exist",
		"corr_abc123def456",
		[]api.ErrorDetail{
			{Field: "resource_id", Reason: "entity_absent", Message: "ID not found in tenant scope"},
		},
	)
	if err != nil {
		t.Fatalf("failed to create original envelope: %v", err)
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal envelope: %v", err)
	}

	var decoded api.ErrorEnvelope
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal envelope: %v", err)
	}

	if decoded.Code != original.Code {
		t.Errorf("expected code %q, got %q", original.Code, decoded.Code)
	}
	if decoded.Message != original.Message {
		t.Errorf("expected message %q, got %q", original.Message, decoded.Message)
	}
	if decoded.CorrelationID != original.CorrelationID {
		t.Errorf("expected correlation_id %q, got %q", original.CorrelationID, decoded.CorrelationID)
	}
	if len(decoded.Details) != len(original.Details) {
		t.Fatalf("expected %d details, got %d", len(original.Details), len(decoded.Details))
	}
	if decoded.Details[0].Field != original.Details[0].Field {
		t.Errorf("expected detail field %q, got %q", original.Details[0].Field, decoded.Details[0].Field)
	}
}

func TestErrorEnvelope_UnmarshalInvalidJSON(t *testing.T) {
	invalidJSON := `{"code":"","message":"bad","correlation_id":"corr_1"}`
	var env api.ErrorEnvelope
	if err := json.Unmarshal([]byte(invalidJSON), &env); err == nil {
		t.Errorf("expected unmarshal error for empty code, got nil")
	}
}
