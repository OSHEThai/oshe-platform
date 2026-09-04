package evidence_test

import (
	"errors"
	"strings"
	"testing"

	evidence "github.com/oshethai/oshe-platform/modules/files-evidence"
)

const (
	validDigest = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
)

func TestNewFileMetadata_Valid(t *testing.T) {
	meta, err := evidence.NewFileMetadata(
		"evd_0123456789abcdef",
		"ten_alpha",
		"site_inspection_photo.png",
		"s3://oshe-evidence/ten_alpha/evd_0123456789abcdef.png",
		"image/png",
		1024*1024,
		validDigest,
	)
	if err != nil {
		t.Fatalf("unexpected construction error: %v", err)
	}

	if meta.FileID != "evd_0123456789abcdef" {
		t.Errorf("expected file_id 'evd_0123456789abcdef', got %q", meta.FileID)
	}
	if meta.TenantID != "ten_alpha" {
		t.Errorf("expected tenant_id 'ten_alpha', got %q", meta.TenantID)
	}
	if meta.OriginalName != "site_inspection_photo.png" {
		t.Errorf("expected original_name 'site_inspection_photo.png', got %q", meta.OriginalName)
	}
	if meta.MediaType != "image/png" {
		t.Errorf("expected media_type 'image/png', got %q", meta.MediaType)
	}
	if meta.SizeBytes != 1024*1024 {
		t.Errorf("expected size 1048576, got %d", meta.SizeBytes)
	}
	if meta.SHA256Digest != validDigest {
		t.Errorf("expected digest %s, got %s", validDigest, meta.SHA256Digest)
	}
	if meta.State != evidence.StateInitialized {
		t.Errorf("expected initial state 'INITIALIZED', got %q", meta.State)
	}
}

func TestValidateFilename_MaliciousAndInvalidInputs(t *testing.T) {
	maliciousCases := []struct {
		name     string
		filename string
	}{
		{"empty", ""},
		{"whitespace", "   "},
		{"unix_traversal", "../../../etc/passwd"},
		{"windows_traversal", "..\\..\\windows\\system32\\calc.exe"},
		{"embedded_traversal", "safe/../../sensitive.pdf"},
		{"slash_separator", "nested/folder/photo.png"},
		{"backslash_separator", "nested\\folder\\photo.png"},
		{"null_byte", "safe.png\x00.exe"},
		{"newline_injection", "photo\nmalicious.jpg"},
		{"carriage_return", "photo\rmalicious.jpg"},
		{"bell_control_char", "photo\a.png"},
	}

	for _, tc := range maliciousCases {
		t.Run(tc.name, func(t *testing.T) {
			err := evidence.ValidateFilename(tc.filename)
			if err == nil {
				t.Fatalf("expected error for malicious filename %q, got nil", tc.filename)
			}
			if !errors.Is(err, evidence.ErrInvalidFilename) {
				t.Errorf("expected ErrInvalidFilename, got: %v", err)
			}
		})
	}
}

func TestNormalizeAndValidateMediaType_ValidAndInvalid(t *testing.T) {
	validCases := []struct {
		input    string
		expected string
	}{
		{"image/png", "image/png"},
		{"IMAGE/PNG", "image/png"},
		{"image/jpeg", "image/jpeg"},
		{"image/webp", "image/webp"},
		{"application/pdf", "application/pdf"},
		{"text/plain; charset=utf-8", "text/plain"},
		{"application/json", "application/json"},
	}

	for _, tc := range validCases {
		t.Run(tc.input, func(t *testing.T) {
			norm, err := evidence.NormalizeAndValidateMediaType(tc.input)
			if err != nil {
				t.Fatalf("expected valid media type for %q, got: %v", tc.input, err)
			}
			if norm != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, norm)
			}
		})
	}

	invalidCases := []struct {
		name      string
		mediaType string
	}{
		{"empty", ""},
		{"whitespace", "   "},
		{"unsupported_exe", "application/x-msdownload"},
		{"unsupported_sh", "application/x-sh"},
		{"unsupported_html", "text/html"},
		{"unsupported_video", "video/mp4"},
		{"malformed_syntax", "invalid-no-slash"},
	}

	for _, tc := range invalidCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := evidence.NormalizeAndValidateMediaType(tc.mediaType)
			if err == nil {
				t.Fatalf("expected error for media type %q, got nil", tc.mediaType)
			}
			if !errors.Is(err, evidence.ErrInvalidMediaType) {
				t.Errorf("expected ErrInvalidMediaType, got: %v", err)
			}
		})
	}
}

func TestSizeBoundaries(t *testing.T) {
	cases := []struct {
		name      string
		size      int64
		expectErr bool
	}{
		{"negative", -1, true},
		{"zero", 0, true},
		{"one_byte", 1, false},
		{"typical_image", 4 * 1024 * 1024, false},
		{"max_allowed", evidence.MaxFileSizeBytes, false},
		{"exceeds_max", evidence.MaxFileSizeBytes + 1, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := evidence.NewFileMetadata(
				"evd_001",
				"ten_1",
				"file.png",
				"ref://storage/file.png",
				"image/png",
				tc.size,
				validDigest,
			)
			if tc.expectErr && err == nil {
				t.Fatalf("expected error for size %d, got nil", tc.size)
			}
			if !tc.expectErr && err != nil {
				t.Fatalf("unexpected error for size %d: %v", tc.size, err)
			}
			if tc.expectErr && !errors.Is(err, evidence.ErrInvalidSize) {
				t.Errorf("expected ErrInvalidSize, got: %v", err)
			}
		})
	}
}

func TestValidateDigest_ValidAndInvalid(t *testing.T) {
	if err := evidence.ValidateDigest(validDigest); err != nil {
		t.Fatalf("expected valid digest to pass, got: %v", err)
	}

	invalidDigests := []struct {
		name   string
		digest string
	}{
		{"empty", ""},
		{"too_short", "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b85"},
		{"too_long", validDigest + "0"},
		{"uppercase_hex", strings.ToUpper(validDigest)},
		{"non_hex_characters", "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz"},
	}

	for _, tc := range invalidDigests {
		t.Run(tc.name, func(t *testing.T) {
			err := evidence.ValidateDigest(tc.digest)
			if err == nil {
				t.Fatalf("expected error for digest %q, got nil", tc.digest)
			}
			if !errors.Is(err, evidence.ErrInvalidDigest) {
				t.Errorf("expected ErrInvalidDigest, got: %v", err)
			}
		})
	}
}

func TestAuthorizeStorageReference_TenantIsolation(t *testing.T) {
	meta, err := evidence.NewFileMetadata(
		"evd_scope_test",
		"ten_hospital_a",
		"evidence.pdf",
		"s3://bucket/ten_hospital_a/evd_scope_test.pdf",
		"application/pdf",
		2048,
		validDigest,
	)
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	// Authorized matching tenant receives storage reference
	ref, err := meta.AuthorizeStorageReference("ten_hospital_a")
	if err != nil {
		t.Fatalf("expected authorized access for matching tenant, got error: %v", err)
	}
	if ref != meta.StorageRef {
		t.Errorf("expected storage ref %q, got %q", meta.StorageRef, ref)
	}

	// Foreign tenant access fails closed
	_, err = meta.AuthorizeStorageReference("ten_hospital_b")
	if err == nil {
		t.Fatal("security violation: foreign tenant accessed storage reference")
	}
	if !errors.Is(err, evidence.ErrTenantMismatch) {
		t.Errorf("expected ErrTenantMismatch, got: %v", err)
	}

	// Empty caller tenant fails closed
	_, err = meta.AuthorizeStorageReference("")
	if !errors.Is(err, evidence.ErrEmptyTenantID) {
		t.Errorf("expected ErrEmptyTenantID, got: %v", err)
	}
}

func TestUploadLifecycleState_TransitionsAndImmutability(t *testing.T) {
	meta, err := evidence.NewFileMetadata(
		"evd_lifecycle_test",
		"ten_1",
		"photo.png",
		"s3://bucket/ten_1/photo.png",
		"image/png",
		512,
		validDigest,
	)
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	// Initialized -> Transferring (valid)
	if err := meta.TransitionState(evidence.StateTransferring); err != nil {
		t.Fatalf("valid transition to transferring failed: %v", err)
	}
	if meta.State != evidence.StateTransferring {
		t.Errorf("expected StateTransferring, got %q", meta.State)
	}

	// Transferring -> Completed (valid)
	if err := meta.TransitionState(evidence.StateCompleted); err != nil {
		t.Fatalf("valid transition to completed failed: %v", err)
	}
	if meta.State != evidence.StateCompleted {
		t.Errorf("expected StateCompleted, got %q", meta.State)
	}

	// Completed state is strictly immutable: all subsequent transitions MUST fail
	forbiddenTargets := []evidence.UploadLifecycleState{
		evidence.StateInitialized,
		evidence.StateTransferring,
		evidence.StateCompleted,
		evidence.StateFailed,
		evidence.StateAborted,
	}

	for _, target := range forbiddenTargets {
		err := meta.TransitionState(target)
		if err == nil {
			t.Fatalf("expected error attempting to transition completed upload to %q, got nil", target)
		}
		if !errors.Is(err, evidence.ErrCompletedStateImmutable) {
			t.Errorf("expected ErrCompletedStateImmutable for target %q, got: %v", target, err)
		}
	}
}

func TestUploadLifecycleState_InvalidTransitions(t *testing.T) {
	// Direct Initialized -> Completed is invalid (must transfer first)
	meta, _ := evidence.NewFileMetadata("evd_1", "ten_1", "file.png", "s3://b/k", "image/png", 100, validDigest)
	err := meta.TransitionState(evidence.StateCompleted)
	if !errors.Is(err, evidence.ErrInvalidLifecycleTransition) {
		t.Errorf("expected ErrInvalidLifecycleTransition for direct Initialized->Completed, got: %v", err)
	}

	// Aborted terminal state cannot transition to transferring or completed
	metaAborted, _ := evidence.NewFileMetadata("evd_2", "ten_1", "file.png", "s3://b/k", "image/png", 100, validDigest)
	_ = metaAborted.TransitionState(evidence.StateAborted)
	err = metaAborted.TransitionState(evidence.StateTransferring)
	if !errors.Is(err, evidence.ErrInvalidLifecycleTransition) {
		t.Errorf("expected ErrInvalidLifecycleTransition for Aborted->Transferring, got: %v", err)
	}
}
