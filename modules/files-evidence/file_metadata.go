// Package evidence provides provider-neutral file metadata and integrity
// validation models for v0.2.0 files and evidence under milestone topic V020-T03.
// This package operates in-memory and executes zero storage I/O.
package evidence

import (
	"encoding/hex"
	"errors"
	"fmt"
	"mime"
	"path/filepath"
	"strings"
	"unicode"
)

var (
	// ErrEmptyFileID indicates that the file identifier is empty or whitespace-only.
	ErrEmptyFileID = errors.New("file_id cannot be empty")
	// ErrEmptyTenantID indicates that the tenant identifier is empty or whitespace-only.
	ErrEmptyTenantID = errors.New("tenant_id cannot be empty")
	// ErrEmptyStorageRef indicates that the storage reference is empty or whitespace-only.
	ErrEmptyStorageRef = errors.New("storage_ref cannot be empty")
	// ErrInvalidFilename indicates that the original filename contains path separators, control chars, or traversal elements.
	ErrInvalidFilename = errors.New("original filename contains illegal characters, path separators, or traversal elements")
	// ErrInvalidMediaType indicates that the media type is unsupported, empty, or malformed.
	ErrInvalidMediaType = errors.New("unsupported or malformed media type")
	// ErrInvalidSize indicates that the declared size is non-positive or exceeds the maximum allowed limit.
	ErrInvalidSize = errors.New("file size must be positive and within maximum allowed limit")
	// ErrInvalidDigest indicates that the content digest is not a valid 64-character lowercase SHA-256 hex string.
	ErrInvalidDigest = errors.New("content digest must be a 64-character lowercase SHA-256 hex string")
	// ErrTenantMismatch indicates that the caller tenant does not match the resource tenant scope.
	ErrTenantMismatch = errors.New("tenant mismatch: caller is not authorized to access this storage reference")
	// ErrInvalidLifecycleTransition indicates an unauthorized forward or backward lifecycle state transition.
	ErrInvalidLifecycleTransition = errors.New("invalid upload lifecycle state transition")
	// ErrCompletedStateImmutable indicates an attempt to modify or transition an already completed upload.
	ErrCompletedStateImmutable = errors.New("completed evidence upload state is immutable and cannot be replaced or transitioned")
)

const (
	// MaxFileSizeBytes sets a strict upper bound on acceptable single-file upload size (50 MiB).
	MaxFileSizeBytes int64 = 50 * 1024 * 1024
)

// UploadLifecycleState defines the deterministic linear progression of an evidence file upload.
type UploadLifecycleState string

const (
	// StateInitialized indicates upload registration has occurred but payload transfer has not begun.
	StateInitialized UploadLifecycleState = "INITIALIZED"
	// StateTransferring indicates payload transfer is actively in progress.
	StateTransferring UploadLifecycleState = "TRANSFERRING"
	// StateCompleted indicates payload transfer and SHA-256 integrity verification are completed. This state is immutable.
	StateCompleted UploadLifecycleState = "COMPLETED"
	// StateFailed indicates payload transfer or digest verification failed.
	StateFailed UploadLifecycleState = "FAILED"
	// StateAborted indicates upload was explicitly canceled before completion.
	StateAborted UploadLifecycleState = "ABORTED"
)

// SupportedMediaTypes enumerates the provider-neutral supported MIME types for evidence attachments.
var SupportedMediaTypes = map[string]bool{
	"image/jpeg":       true,
	"image/png":        true,
	"image/webp":       true,
	"application/pdf":  true,
	"text/plain":       true,
	"application/json": true,
}

// FileMetadata represents provider-neutral metadata for an immutable evidence file.
type FileMetadata struct {
	FileID       string               `json:"file_id"`
	TenantID     string               `json:"tenant_id"`
	OriginalName string               `json:"original_name"`
	StorageRef   string               `json:"storage_ref"`
	MediaType    string               `json:"media_type"`
	SizeBytes    int64                `json:"size_bytes"`
	SHA256Digest string               `json:"sha256_digest"`
	State        UploadLifecycleState `json:"state"`
}

// ValidateFilename fails closed for path separators, directory traversal, null bytes, or control characters.
func ValidateFilename(name string) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return ErrInvalidFilename
	}
	if strings.ContainsAny(trimmed, "/\\") || strings.Contains(trimmed, "..") || strings.ContainsRune(trimmed, 0) {
		return ErrInvalidFilename
	}
	if filepath.Base(trimmed) != trimmed {
		return ErrInvalidFilename
	}
	for _, r := range trimmed {
		if unicode.IsControl(r) {
			return ErrInvalidFilename
		}
	}
	return nil
}

// NormalizeAndValidateMediaType validates that mediaType is well-formed, lowercase, and in the approved allowlist.
func NormalizeAndValidateMediaType(mediaType string) (string, error) {
	trimmed := strings.ToLower(strings.TrimSpace(mediaType))
	if trimmed == "" {
		return "", ErrInvalidMediaType
	}
	parsed, _, err := mime.ParseMediaType(trimmed)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidMediaType, err)
	}
	if !SupportedMediaTypes[parsed] {
		return "", fmt.Errorf("%w: type %q is not in the approved evidence media type allowlist", ErrInvalidMediaType, parsed)
	}
	return parsed, nil
}

// ValidateDigest ensures the digest is a 64-character lowercase SHA-256 hexadecimal string.
func ValidateDigest(digest string) error {
	trimmed := strings.TrimSpace(digest)
	if len(trimmed) != 64 {
		return ErrInvalidDigest
	}
	if strings.ToLower(trimmed) != trimmed {
		return ErrInvalidDigest
	}
	raw, err := hex.DecodeString(trimmed)
	if err != nil || len(raw) != 32 {
		return ErrInvalidDigest
	}
	return nil
}

// NewFileMetadata constructs and validates a new FileMetadata record in the INITIALIZED state.
func NewFileMetadata(fileID, tenantID, originalName, storageRef, mediaType string, sizeBytes int64, sha256Digest string) (*FileMetadata, error) {
	tFileID := strings.TrimSpace(fileID)
	if tFileID == "" {
		return nil, ErrEmptyFileID
	}
	tTenantID := strings.TrimSpace(tenantID)
	if tTenantID == "" {
		return nil, ErrEmptyTenantID
	}
	tStorageRef := strings.TrimSpace(storageRef)
	if tStorageRef == "" {
		return nil, ErrEmptyStorageRef
	}
	if err := ValidateFilename(originalName); err != nil {
		return nil, err
	}
	normMediaType, err := NormalizeAndValidateMediaType(mediaType)
	if err != nil {
		return nil, err
	}
	if sizeBytes <= 0 || sizeBytes > MaxFileSizeBytes {
		return nil, fmt.Errorf("%w: declared size %d bytes (limit 1 to %d bytes)", ErrInvalidSize, sizeBytes, MaxFileSizeBytes)
	}
	if err := ValidateDigest(sha256Digest); err != nil {
		return nil, err
	}

	return &FileMetadata{
		FileID:       tFileID,
		TenantID:     tTenantID,
		OriginalName: strings.TrimSpace(originalName),
		StorageRef:   tStorageRef,
		MediaType:    normMediaType,
		SizeBytes:    sizeBytes,
		SHA256Digest: strings.TrimSpace(sha256Digest),
		State:        StateInitialized,
	}, nil
}

// AuthorizeStorageReference evaluates tenant scope.
// It fails closed for empty or mismatched caller tenant IDs and only returns the storage reference upon strict match.
func (m *FileMetadata) AuthorizeStorageReference(callerTenantID string) (string, error) {
	trimmedCaller := strings.TrimSpace(callerTenantID)
	if trimmedCaller == "" {
		return "", ErrEmptyTenantID
	}
	if m.TenantID != trimmedCaller {
		return "", fmt.Errorf("%w: resource tenant %q does not match caller %q", ErrTenantMismatch, m.TenantID, trimmedCaller)
	}
	return m.StorageRef, nil
}

// TransitionState applies a deterministic state transition to the upload lifecycle.
// Rejects any transition if current state is COMPLETED (immutable), or if the transition is illegal.
func (m *FileMetadata) TransitionState(target UploadLifecycleState) error {
	if m.State == StateCompleted {
		return ErrCompletedStateImmutable
	}

	switch m.State {
	case StateInitialized:
		if target == StateTransferring || target == StateAborted || target == StateFailed {
			m.State = target
			return nil
		}
	case StateTransferring:
		if target == StateCompleted || target == StateFailed || target == StateAborted {
			m.State = target
			return nil
		}
	case StateFailed, StateAborted:
		return fmt.Errorf("%w: cannot transition from terminal state %q to %q", ErrInvalidLifecycleTransition, m.State, target)
	}

	return fmt.Errorf("%w: transition from %q to %q is not permitted", ErrInvalidLifecycleTransition, m.State, target)
}
