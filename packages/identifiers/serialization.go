package identifiers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

const (
	// CurrentCorrelationEnvelopeVersion defines the canonical version identifier for the wire envelope.
	CurrentCorrelationEnvelopeVersion = "v1"
)

var (
	// ErrNilContext indicates that a nil CorrelationContext was provided for encoding.
	ErrNilContext = errors.New("correlation context cannot be nil")
	// ErrEmptyVersion indicates that the version field is missing or whitespace-only.
	ErrEmptyVersion = errors.New("envelope version cannot be empty")
	// ErrUnsupportedVersion indicates that the envelope version is not recognized.
	ErrUnsupportedVersion = errors.New("unsupported envelope version: expected v1")
	// ErrExtraFields indicates that unexpected or extra properties were present in the JSON envelope.
	ErrExtraFields = errors.New("envelope contains unexpected or extra fields")
	// ErrDuplicateFields indicates that duplicate or ambiguous fields were detected in the JSON envelope.
	ErrDuplicateFields = errors.New("envelope contains duplicate or ambiguous fields")
	// ErrMalformedEnvelopeJSON indicates that the JSON payload could not be parsed into an object.
	ErrMalformedEnvelopeJSON = errors.New("malformed envelope JSON")
)

var allowedCorrelationEnvelopeFields = map[string]bool{
	"version":        true,
	"correlation_id": true,
	"causation_id":   true,
}

// CorrelationEnvelope defines the canonical, versioned wire structure for serializing CorrelationContext.
type CorrelationEnvelope struct {
	Version       string `json:"version"`
	CorrelationID string `json:"correlation_id"`
	CausationID   string `json:"causation_id"`
}

// EncodeCorrelationContext serializes a validated CorrelationContext into a deterministic,
// byte-for-byte reproducible JSON payload with fixed envelope version "v1".
func EncodeCorrelationContext(ctx *CorrelationContext) ([]byte, error) {
	if ctx == nil {
		return nil, ErrNilContext
	}

	corrID, err := ParseCorrelationID(ctx.CorrelationID.String())
	if err != nil {
		return nil, err
	}
	causID, err := ParseCausationID(ctx.CausationID.String())
	if err != nil {
		return nil, err
	}

	env := CorrelationEnvelope{
		Version:       CurrentCorrelationEnvelopeVersion,
		CorrelationID: corrID.String(),
		CausationID:   causID.String(),
	}

	return json.Marshal(env)
}

func checkEnvelopeSyntaxAndKeys(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return fmt.Errorf("%w: payload is empty", ErrMalformedEnvelopeJSON)
	}

	dec := json.NewDecoder(bytes.NewReader(trimmed))
	t, err := dec.Token()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrMalformedEnvelopeJSON, err)
	}
	delim, ok := t.(json.Delim)
	if !ok || delim != '{' {
		return fmt.Errorf("%w: top-level element must be a JSON object", ErrMalformedEnvelopeJSON)
	}

	seenKeys := make(map[string]bool)
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return fmt.Errorf("%w: %v", ErrMalformedEnvelopeJSON, err)
		}
		key, ok := tok.(string)
		if !ok {
			return fmt.Errorf("%w: object key must be a string", ErrMalformedEnvelopeJSON)
		}

		if !allowedCorrelationEnvelopeFields[key] {
			return fmt.Errorf("%w: field %q is not permitted", ErrExtraFields, key)
		}
		if seenKeys[key] {
			return fmt.Errorf("%w: field %q occurs multiple times", ErrDuplicateFields, key)
		}
		seenKeys[key] = true

		var discard any
		if err := dec.Decode(&discard); err != nil {
			return fmt.Errorf("%w: failed to parse value for %q: %v", ErrMalformedEnvelopeJSON, key, err)
		}
	}

	return nil
}

// DecodeCorrelationEnvelope parses a JSON byte slice into a CorrelationContext.
// It fails closed if the envelope is empty, contains unknown fields, duplicate keys,
// wrong version, or invalid correlation/causation identifier formats.
func DecodeCorrelationEnvelope(data []byte) (*CorrelationContext, error) {
	if err := checkEnvelopeSyntaxAndKeys(data); err != nil {
		return nil, err
	}

	var env CorrelationEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrMalformedEnvelopeJSON, err)
	}

	v := strings.TrimSpace(env.Version)
	if v == "" {
		return nil, ErrEmptyVersion
	}
	if v != CurrentCorrelationEnvelopeVersion {
		return nil, fmt.Errorf("%w: got %q", ErrUnsupportedVersion, v)
	}

	corr, err := ParseCorrelationID(env.CorrelationID)
	if err != nil {
		return nil, err
	}

	caus, err := ParseCausationID(env.CausationID)
	if err != nil {
		return nil, err
	}

	return &CorrelationContext{
		CorrelationID: corr,
		CausationID:   caus,
	}, nil
}
