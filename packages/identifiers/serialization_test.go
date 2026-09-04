package identifiers_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/oshethai/oshe-platform/packages/identifiers"
)

const (
	sampleCorrID = "corr_0123456789abcdef0123456789abcdef"
	sampleCausID = "caus_fedcba9876543210fedcba9876543210"
)

func mustContext(t *testing.T) *identifiers.CorrelationContext {
	t.Helper()
	ctx, err := identifiers.NewCorrelationContext(sampleCorrID, sampleCausID)
	if err != nil {
		t.Fatalf("failed to create fixture context: %v", err)
	}
	return ctx
}

func TestEncodeCorrelationContext_Valid(t *testing.T) {
	ctx := mustContext(t)
	data, err := identifiers.EncodeCorrelationContext(ctx)
	if err != nil {
		t.Fatalf("failed to encode correlation context: %v", err)
	}

	expectedJSON := `{"version":"v1","correlation_id":"corr_0123456789abcdef0123456789abcdef","causation_id":"caus_fedcba9876543210fedcba9876543210"}`
	if string(data) != expectedJSON {
		t.Errorf("encoded JSON mismatch:\nexpected: %s\ngot:      %s", expectedJSON, string(data))
	}
}

func TestEncodeCorrelationContext_NilAndInvalidContext(t *testing.T) {
	if _, err := identifiers.EncodeCorrelationContext(nil); !errors.Is(err, identifiers.ErrNilContext) {
		t.Errorf("expected ErrNilContext for nil context, got: %v", err)
	}

	invalidContext := &identifiers.CorrelationContext{
		CorrelationID: identifiers.ID("bad_id"),
		CausationID:   identifiers.ID(sampleCausID),
	}
	if _, err := identifiers.EncodeCorrelationContext(invalidContext); err == nil {
		t.Error("expected error encoding context with invalid correlation ID prefix")
	}
}

func TestCorrelationEnvelope_RoundTrip(t *testing.T) {
	ctx := mustContext(t)
	encoded, err := identifiers.EncodeCorrelationContext(ctx)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	decoded, err := identifiers.DecodeCorrelationEnvelope(encoded)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if decoded.CorrelationID != ctx.CorrelationID {
		t.Errorf("expected CorrelationID %q, got %q", ctx.CorrelationID, decoded.CorrelationID)
	}
	if decoded.CausationID != ctx.CausationID {
		t.Errorf("expected CausationID %q, got %q", ctx.CausationID, decoded.CausationID)
	}
}

func TestCorrelationEnvelope_ByteForByteReproducibility(t *testing.T) {
	ctx := mustContext(t)
	first, err := identifiers.EncodeCorrelationContext(ctx)
	if err != nil {
		t.Fatalf("initial encode failed: %v", err)
	}

	const iterations = 100
	for range iterations {
		subsequent, err := identifiers.EncodeCorrelationContext(ctx)
		if err != nil {
			t.Fatalf("subsequent encode failed: %v", err)
		}
		if !bytes.Equal(first, subsequent) {
			t.Fatalf("byte-for-byte non-reproducible output:\nfirst:      %s\nsubsequent: %s", string(first), string(subsequent))
		}
	}
}

func TestDecodeCorrelationEnvelope_MissingOrUnknownVersion(t *testing.T) {
	missingVersion := `{"correlation_id":"corr_0123456789abcdef0123456789abcdef","causation_id":"caus_fedcba9876543210fedcba9876543210"}`
	_, err := identifiers.DecodeCorrelationEnvelope([]byte(missingVersion))
	if !errors.Is(err, identifiers.ErrEmptyVersion) {
		t.Errorf("expected ErrEmptyVersion for missing version, got: %v", err)
	}

	blankVersion := `{"version":"   ","correlation_id":"corr_0123456789abcdef0123456789abcdef","causation_id":"caus_fedcba9876543210fedcba9876543210"}`
	_, err = identifiers.DecodeCorrelationEnvelope([]byte(blankVersion))
	if !errors.Is(err, identifiers.ErrEmptyVersion) {
		t.Errorf("expected ErrEmptyVersion for blank version, got: %v", err)
	}

	unsupportedVersion := `{"version":"v2","correlation_id":"corr_0123456789abcdef0123456789abcdef","causation_id":"caus_fedcba9876543210fedcba9876543210"}`
	_, err = identifiers.DecodeCorrelationEnvelope([]byte(unsupportedVersion))
	if !errors.Is(err, identifiers.ErrUnsupportedVersion) {
		t.Errorf("expected ErrUnsupportedVersion for version v2, got: %v", err)
	}
}

func TestDecodeCorrelationEnvelope_MalformedOrWrongPrefixIDs(t *testing.T) {
	cases := []struct {
		name string
		json string
	}{
		{
			name: "wrong_prefix_correlation",
			json: `{"version":"v1","correlation_id":"ten_0123456789abcdef0123456789abcdef","causation_id":"caus_fedcba9876543210fedcba9876543210"}`,
		},
		{
			name: "wrong_prefix_causation",
			json: `{"version":"v1","correlation_id":"corr_0123456789abcdef0123456789abcdef","causation_id":"corr_fedcba9876543210fedcba9876543210"}`,
		},
		{
			name: "malformed_correlation_hex",
			json: `{"version":"v1","correlation_id":"corr_badhex123","causation_id":"caus_fedcba9876543210fedcba9876543210"}`,
		},
		{
			name: "malformed_causation_hex",
			json: `{"version":"v1","correlation_id":"corr_0123456789abcdef0123456789abcdef","causation_id":"caus_zzzzzzzzzzzzzz"}`,
		},
		{
			name: "empty_correlation_id",
			json: `{"version":"v1","correlation_id":"","causation_id":"caus_fedcba9876543210fedcba9876543210"}`,
		},
		{
			name: "empty_causation_id",
			json: `{"version":"v1","correlation_id":"corr_0123456789abcdef0123456789abcdef","causation_id":""}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := identifiers.DecodeCorrelationEnvelope([]byte(tc.json))
			if err == nil {
				t.Errorf("expected decoding error for %s, got nil", tc.name)
			}
		})
	}
}

func TestDecodeCorrelationEnvelope_ExtraFields(t *testing.T) {
	extraFieldsJSON := `{"version":"v1","correlation_id":"corr_0123456789abcdef0123456789abcdef","causation_id":"caus_fedcba9876543210fedcba9876543210","unexpected_field":"value"}`
	_, err := identifiers.DecodeCorrelationEnvelope([]byte(extraFieldsJSON))
	if !errors.Is(err, identifiers.ErrExtraFields) {
		t.Errorf("expected ErrExtraFields for unexpected property, got: %v", err)
	}
}

func TestDecodeCorrelationEnvelope_DuplicateFields(t *testing.T) {
	duplicateVersionJSON := `{"version":"v1","version":"v1","correlation_id":"corr_0123456789abcdef0123456789abcdef","causation_id":"caus_fedcba9876543210fedcba9876543210"}`
	_, err := identifiers.DecodeCorrelationEnvelope([]byte(duplicateVersionJSON))
	if !errors.Is(err, identifiers.ErrDuplicateFields) {
		t.Errorf("expected ErrDuplicateFields for duplicate version key, got: %v", err)
	}

	duplicateCorrelationJSON := `{"version":"v1","correlation_id":"corr_0123456789abcdef0123456789abcdef","correlation_id":"corr_0123456789abcdef0123456789abcdef","causation_id":"caus_fedcba9876543210fedcba9876543210"}`
	_, err = identifiers.DecodeCorrelationEnvelope([]byte(duplicateCorrelationJSON))
	if !errors.Is(err, identifiers.ErrDuplicateFields) {
		t.Errorf("expected ErrDuplicateFields for duplicate correlation_id key, got: %v", err)
	}
}

func TestDecodeCorrelationEnvelope_MalformedSyntax(t *testing.T) {
	malformedCases := []struct {
		name string
		raw  string
	}{
		{"empty", ""},
		{"whitespace", "   \t\n"},
		{"array_not_object", `["v1", "corr_1", "caus_2"]`},
		{"primitive_string", `"hello"`},
		{"broken_json", `{"version":"v1",`},
	}

	for _, tc := range malformedCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := identifiers.DecodeCorrelationEnvelope([]byte(tc.raw))
			if !errors.Is(err, identifiers.ErrMalformedEnvelopeJSON) {
				t.Errorf("expected ErrMalformedEnvelopeJSON for %s, got: %v", tc.name, err)
			}
		})
	}
}
