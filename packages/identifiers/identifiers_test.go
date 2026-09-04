package identifiers_test

import (
	"errors"
	"testing"

	"github.com/oshethai/oshe-platform/packages/identifiers"
)

func TestGenerate_Accepted(t *testing.T) {
	prefixes := []string{"org", "ten", "usr", "chk", "ins", "evd", "act", "dev123"}
	for _, p := range prefixes {
		id, err := identifiers.Generate(p)
		if err != nil {
			t.Fatalf("expected successful generation for prefix %q, got error: %v", p, err)
		}
		if id.String() == "" {
			t.Fatalf("expected non-empty ID for prefix %q", p)
		}
		if id.Prefix() != p {
			t.Errorf("expected prefix %q, got %q", p, id.Prefix())
		}
		if len(id.Suffix()) != 32 {
			t.Errorf("expected 32-char hex suffix for %q, got %d chars: %q", p, len(id.Suffix()), id.Suffix())
		}
	}
}

func TestGenerate_RejectedPrefixes(t *testing.T) {
	invalidPrefixes := []struct {
		name   string
		prefix string
	}{
		{"empty", ""},
		{"whitespace", "   "},
		{"too_short", "a"},
		{"too_long", "thisprefixiswaytoolongexceeding16chars"},
		{"uppercase", "ORG"},
		{"mixed_case", "Tenant"},
		{"symbols_dash", "my-org"},
		{"symbols_underscore", "my_org"},
		{"symbols_dot", "my.org"},
		{"unicode", "องค์กร"},
	}

	for _, tc := range invalidPrefixes {
		t.Run(tc.name, func(t *testing.T) {
			_, err := identifiers.Generate(tc.prefix)
			if err == nil {
				t.Errorf("expected error generating identifier with prefix %q, got nil", tc.prefix)
			}
		})
	}
}

func TestGenerate_Uniqueness(t *testing.T) {
	seen := make(map[string]bool)
	const count = 500
	for i := range count {
		id, err := identifiers.Generate("ten")
		if err != nil {
			t.Fatalf("unexpected generation error on iteration %d: %v", i, err)
		}
		str := id.String()
		if seen[str] {
			t.Fatalf("collision detected on iteration %d: %q", i, str)
		}
		seen[str] = true
	}
}

func TestParse_Accepted(t *testing.T) {
	validIDs := []struct {
		raw            string
		expectedPrefix string
		expectedSuffix string
	}{
		{"org_0123456789abcdef0123456789abcdef", "org", "0123456789abcdef0123456789abcdef"},
		{"ten_abcdef0123456789abcdef0123456789", "ten", "abcdef0123456789abcdef0123456789"},
		{"chk1_aabbccddeeff00112233445566778899", "chk1", "aabbccddeeff00112233445566778899"},
		{"dev_deadbeef", "dev", "deadbeef"},
	}

	for _, tc := range validIDs {
		t.Run(tc.raw, func(t *testing.T) {
			parsed, err := identifiers.Parse(tc.raw)
			if err != nil {
				t.Fatalf("expected valid parse for %q, got error: %v", tc.raw, err)
			}
			if parsed.String() != tc.raw {
				t.Errorf("expected String() == %q, got %q", tc.raw, parsed.String())
			}
			if parsed.Prefix() != tc.expectedPrefix {
				t.Errorf("expected Prefix() == %q, got %q", tc.expectedPrefix, parsed.Prefix())
			}
			if parsed.Suffix() != tc.expectedSuffix {
				t.Errorf("expected Suffix() == %q, got %q", tc.expectedSuffix, parsed.Suffix())
			}
		})
	}
}

func TestParse_RejectedInputs(t *testing.T) {
	invalidInputs := []struct {
		name string
		raw  string
	}{
		{"empty", ""},
		{"whitespace", "   \t\n"},
		{"no_underscore", "org1234567890abcdef"},
		{"multiple_underscores", "org_sub_0123456789abcdef"},
		{"empty_prefix", "_0123456789abcdef"},
		{"empty_suffix", "org_"},
		{"invalid_prefix_short", "o_0123456789abcdef"},
		{"invalid_prefix_uppercase", "ORG_0123456789abcdef"},
		{"invalid_prefix_symbols", "my-org_0123456789abcdef"},
		{"odd_length_hex", "org_abc"},
		{"non_hex_payload", "org_0123456789abcdefghijklmnop"},
		{"uppercase_hex_payload", "org_0123456789ABCDEF0123456789ABCDEF"},
	}

	for _, tc := range invalidInputs {
		t.Run(tc.name, func(t *testing.T) {
			_, err := identifiers.Parse(tc.raw)
			if err == nil {
				t.Errorf("expected parse error for input %q, got nil", tc.raw)
			}
		})
	}
}

func TestParseWithPrefix_AcceptedAndRejected(t *testing.T) {
	raw := "usr_0123456789abcdef0123456789abcdef"

	// Successful match
	id, err := identifiers.ParseWithPrefix(raw, "usr")
	if err != nil {
		t.Fatalf("expected successful match for prefix 'usr', got error: %v", err)
	}
	if id.Prefix() != "usr" {
		t.Errorf("expected prefix 'usr', got %q", id.Prefix())
	}

	// Mismatch prefix
	_, err = identifiers.ParseWithPrefix(raw, "org")
	if err == nil {
		t.Fatalf("expected error for mismatched prefix, got nil")
	}
	if !errors.Is(err, identifiers.ErrPrefixMismatch) {
		t.Errorf("expected ErrPrefixMismatch, got: %v", err)
	}

	// Invalid expected prefix parameter
	_, err = identifiers.ParseWithPrefix(raw, "")
	if err == nil {
		t.Errorf("expected error for empty expected prefix, got nil")
	}
}

func TestID_MalformedAccessors(t *testing.T) {
	// Directly constructed ID without underscore
	badID := identifiers.ID("malformed")
	if badID.Prefix() != "" {
		t.Errorf("expected empty prefix for malformed ID, got %q", badID.Prefix())
	}
	if badID.Suffix() != "" {
		t.Errorf("expected empty suffix for malformed ID, got %q", badID.Suffix())
	}
}
