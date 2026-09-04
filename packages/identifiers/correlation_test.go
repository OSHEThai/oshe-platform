package identifiers_test

import (
	"errors"
	"testing"

	"github.com/oshethai/oshe-platform/packages/identifiers"
)

func TestGenerateCorrelationID(t *testing.T) {
	id, err := identifiers.GenerateCorrelationID()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id.Prefix() != identifiers.PrefixCorrelation {
		t.Errorf("expected prefix %q, got %q", identifiers.PrefixCorrelation, id.Prefix())
	}
	if len(id.Suffix()) != 32 {
		t.Errorf("expected 32-char hex suffix, got %d", len(id.Suffix()))
	}
}

func TestGenerateCausationID(t *testing.T) {
	id, err := identifiers.GenerateCausationID()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id.Prefix() != identifiers.PrefixCausation {
		t.Errorf("expected prefix %q, got %q", identifiers.PrefixCausation, id.Prefix())
	}
	if len(id.Suffix()) != 32 {
		t.Errorf("expected 32-char hex suffix, got %d", len(id.Suffix()))
	}
}

func TestParseCorrelationID_ValidAndInvalid(t *testing.T) {
	valid := "corr_0123456789abcdef0123456789abcdef"
	id, err := identifiers.ParseCorrelationID(valid)
	if err != nil {
		t.Fatalf("expected valid parse, got %v", err)
	}
	if id.String() != valid {
		t.Errorf("expected %q, got %q", valid, id.String())
	}

	invalidCases := []string{
		"",
		"   ",
		"caus_0123456789abcdef0123456789abcdef",
		"ten_0123456789abcdef0123456789abcdef",
		"corr_invalidhexcharacterszzzzzzzzzzzzzz",
	}
	for _, inv := range invalidCases {
		_, err := identifiers.ParseCorrelationID(inv)
		if err == nil {
			t.Errorf("expected error for invalid correlation ID %q, got nil", inv)
		}
		if !errors.Is(err, identifiers.ErrInvalidCorrelationID) {
			t.Errorf("expected ErrInvalidCorrelationID, got: %v", err)
		}
	}
}

func TestParseCausationID_ValidAndInvalid(t *testing.T) {
	valid := "caus_0123456789abcdef0123456789abcdef"
	id, err := identifiers.ParseCausationID(valid)
	if err != nil {
		t.Fatalf("expected valid parse, got %v", err)
	}
	if id.String() != valid {
		t.Errorf("expected %q, got %q", valid, id.String())
	}

	invalidCases := []string{
		"",
		"   ",
		"corr_0123456789abcdef0123456789abcdef",
		"caus_badhex",
	}
	for _, inv := range invalidCases {
		_, err := identifiers.ParseCausationID(inv)
		if err == nil {
			t.Errorf("expected error for invalid causation ID %q, got nil", inv)
		}
		if !errors.Is(err, identifiers.ErrInvalidCausationID) {
			t.Errorf("expected ErrInvalidCausationID, got: %v", err)
		}
	}
}

func TestCorrelationContext_LifecycleAndDerivation(t *testing.T) {
	ctx, err := identifiers.StartCorrelationContext()
	if err != nil {
		t.Fatalf("failed to start correlation context: %v", err)
	}

	if ctx.CorrelationID.Prefix() != identifiers.PrefixCorrelation {
		t.Errorf("expected root correlation prefix %q, got %q", identifiers.PrefixCorrelation, ctx.CorrelationID.Prefix())
	}
	if ctx.CausationID.Prefix() != identifiers.PrefixCausation {
		t.Errorf("expected root causation prefix %q, got %q", identifiers.PrefixCausation, ctx.CausationID.Prefix())
	}

	// Derive child
	child, err := ctx.DeriveChild()
	if err != nil {
		t.Fatalf("failed to derive child: %v", err)
	}

	// Correlation ID must be preserved
	if child.CorrelationID != ctx.CorrelationID {
		t.Errorf("expected child to preserve CorrelationID %s, got %s", ctx.CorrelationID, child.CorrelationID)
	}
	// Causation ID must be a newly generated ID
	if child.CausationID == ctx.CausationID {
		t.Errorf("expected child to have new CausationID, got identical: %s", child.CausationID)
	}
	if child.CausationID.Prefix() != identifiers.PrefixCausation {
		t.Errorf("expected child causation prefix %q, got %q", identifiers.PrefixCausation, child.CausationID.Prefix())
	}
}

func TestNewCorrelationContext_ValidAndInvalid(t *testing.T) {
	corr := "corr_0123456789abcdef0123456789abcdef"
	caus := "caus_0123456789abcdef0123456789abcdef"

	ctx, err := identifiers.NewCorrelationContext(corr, caus)
	if err != nil {
		t.Fatalf("expected successful construction, got: %v", err)
	}
	if ctx.CorrelationID.String() != corr {
		t.Errorf("expected %q, got %q", corr, ctx.CorrelationID)
	}
	if ctx.CausationID.String() != caus {
		t.Errorf("expected %q, got %q", caus, ctx.CausationID)
	}

	// Invalid inputs
	if _, err := identifiers.NewCorrelationContext("", caus); err == nil {
		t.Error("expected error for empty correlation ID")
	}
	if _, err := identifiers.NewCorrelationContext(corr, ""); err == nil {
		t.Error("expected error for empty causation ID")
	}
	if _, err := identifiers.NewCorrelationContext("bad_123", caus); err == nil {
		t.Error("expected error for non-corr correlation ID")
	}
}
