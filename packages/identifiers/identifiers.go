package identifiers

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"unicode"
)

var (
	// ErrEmptyPrefix indicates that a prefix is empty or whitespace-only.
	ErrEmptyPrefix = errors.New("identifier prefix cannot be empty")
	// ErrInvalidPrefix indicates that a prefix contains non-alphanumeric lowercase characters or invalid length.
	ErrInvalidPrefix = errors.New("identifier prefix must contain only lowercase ASCII letters and digits (2-16 chars)")
	// ErrEmptyID indicates that an identifier string is empty or whitespace-only.
	ErrEmptyID = errors.New("identifier cannot be empty")
	// ErrMalformedID indicates that an identifier does not conform to the expected prefix_payload format.
	ErrMalformedID = errors.New("identifier is malformed")
	// ErrPrefixMismatch indicates that an identifier does not match the expected prefix.
	ErrPrefixMismatch = errors.New("identifier prefix mismatch")
)

// ID represents a canonical strongly-typed prefixed identifier.
type ID string

// String returns the serialized string representation of the identifier.
func (id ID) String() string {
	return string(id)
}

// Prefix returns the prefix component before the first underscore, or empty string if malformed.
func (id ID) Prefix() string {
	parts := strings.SplitN(string(id), "_", 2)
	if len(parts) == 2 {
		return parts[0]
	}
	return ""
}

// Suffix returns the payload component after the first underscore, or empty string if malformed.
func (id ID) Suffix() string {
	parts := strings.SplitN(string(id), "_", 2)
	if len(parts) == 2 {
		return parts[1]
	}
	return ""
}

// ValidatePrefix ensures that a prefix is non-empty, contains only lowercase ASCII letters/digits,
// and has a bounded length between 2 and 16 characters.
func ValidatePrefix(prefix string) error {
	trimmed := strings.TrimSpace(prefix)
	if trimmed == "" {
		return ErrEmptyPrefix
	}
	if len(trimmed) < 2 || len(trimmed) > 16 {
		return ErrInvalidPrefix
	}
	for _, r := range trimmed {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')) {
			return ErrInvalidPrefix
		}
	}
	return nil
}

// Generate creates a new non-empty prefixed identifier using crypto/rand entropy.
// The resulting serialized format is "<prefix>_<hex-encoded-random-bytes>".
func Generate(prefix string) (ID, error) {
	if err := ValidatePrefix(prefix); err != nil {
		return "", err
	}
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("failed to read random entropy: %w", err)
	}
	payload := hex.EncodeToString(raw[:])
	return ID(fmt.Sprintf("%s_%s", prefix, payload)), nil
}

// Parse validates that raw conforms to the required "<prefix>_<hex-payload>" structure.
func Parse(raw string) (ID, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", ErrEmptyID
	}
	parts := strings.Split(trimmed, "_")
	if len(parts) != 2 {
		return "", ErrMalformedID
	}
	prefix, payload := parts[0], parts[1]
	if err := ValidatePrefix(prefix); err != nil {
		return "", fmt.Errorf("%w: %v", ErrMalformedID, err)
	}
	if len(payload) == 0 || len(payload)%2 != 0 {
		return "", ErrMalformedID
	}
	for _, r := range payload {
		if !unicode.Is(unicode.Hex_Digit, r) || unicode.IsUpper(r) {
			return "", ErrMalformedID
		}
	}
	return ID(trimmed), nil
}

// ParseWithPrefix validates that raw conforms to canonical structure and matches expectedPrefix.
func ParseWithPrefix(raw string, expectedPrefix string) (ID, error) {
	if err := ValidatePrefix(expectedPrefix); err != nil {
		return "", fmt.Errorf("invalid expected prefix: %w", err)
	}
	id, err := Parse(raw)
	if err != nil {
		return "", err
	}
	if id.Prefix() != expectedPrefix {
		return "", fmt.Errorf("%w: expected prefix %q, got %q", ErrPrefixMismatch, expectedPrefix, id.Prefix())
	}
	return id, nil
}
