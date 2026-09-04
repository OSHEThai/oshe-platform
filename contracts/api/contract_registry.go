// Package api provides provisional static API contract models for v0.2.0.
//
// PROVISIONAL GOVERNANCE DECLARATION:
// The contract registry in this file is an in-memory, local-only specification
// registry for versioned API contract descriptors pending formal Sole Human Owner
// architecture gate H020-005.
// Zero HTTP server binding, runtime daemon attachment, database persistence,
// ownership-policy determination, or external standard compatibility is claimed or granted.
package api

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
)

const (
	// CanonicalErrorEnvelopeRef is the standard reference identifier for v0.2.0 error envelopes.
	CanonicalErrorEnvelopeRef = "OSHE_CANONICAL_ERROR_ENVELOPE_V1"

	// DefaultAPIProtocol defines the standard protocol descriptor for REST APIs.
	DefaultAPIProtocol = "REST_JSON"
)

var (
	// ErrBlankContractID indicates that the contract ID is empty or whitespace-only.
	ErrBlankContractID = errors.New("contract ID cannot be blank")
	// ErrBlankContractName indicates that the contract name is empty or whitespace-only.
	ErrBlankContractName = errors.New("contract name cannot be blank")
	// ErrBlankPathPattern indicates that the path pattern is empty or whitespace-only.
	ErrBlankPathPattern = errors.New("path pattern cannot be blank")
	// ErrBlankErrorEnvelopeRef indicates that the error envelope reference is empty or whitespace-only.
	ErrBlankErrorEnvelopeRef = errors.New("error envelope reference cannot be blank")
	// ErrContractNotFound indicates that the requested contract ID is not registered.
	ErrContractNotFound = errors.New("contract descriptor not found in registry")
	// ErrDuplicateContractID indicates a collision with an existing, differing contract descriptor.
	ErrDuplicateContractID = errors.New("duplicate contract registration: contract ID already registered with differing specification")
	// ErrContractConflict indicates a structural conflict during contract registration.
	ErrContractConflict = errors.New("contract specification conflict detected")
	// ErrNoSupportedMethods indicates that no HTTP methods were declared for the contract.
	ErrNoSupportedMethods = errors.New("contract must specify at least one supported HTTP method")
	// ErrInvalidMethod indicates an unrecognized HTTP verb.
	ErrInvalidMethod = errors.New("unrecognized HTTP method specified")
)

var validHTTPMethods = map[string]bool{
	"GET":     true,
	"POST":    true,
	"PUT":     true,
	"PATCH":   true,
	"DELETE":  true,
	"HEAD":    true,
	"OPTIONS": true,
}

// ContractDescriptor encapsulates a versioned, declarative API contract specification.
type ContractDescriptor struct {
	ContractID       string            `json:"contract_id"`
	Name             string            `json:"name"`
	Version          string            `json:"version"`
	Description      string            `json:"description,omitempty"`
	PathPattern      string            `json:"path_pattern"`
	Protocol         string            `json:"protocol"`
	ErrorEnvelopeRef string            `json:"error_envelope_ref"`
	SupportedMethods []string          `json:"supported_methods"`
	RequiredHeaders  []string          `json:"required_headers,omitempty"`
	ProvisionalState string            `json:"provisional_state"`
	Metadata         map[string]string `json:"metadata,omitempty"`
}

// Validate ensures that a ContractDescriptor contains well-formed, complete metadata.
func (d *ContractDescriptor) Validate() error {
	trimmedID := strings.TrimSpace(d.ContractID)
	if trimmedID == "" {
		return ErrBlankContractID
	}

	trimmedName := strings.TrimSpace(d.Name)
	if trimmedName == "" {
		return ErrBlankContractName
	}

	trimmedVersion := strings.TrimSpace(d.Version)
	if trimmedVersion == "" {
		return ErrUnsupportedVersion
	}
	if trimmedVersion != CurrentContractVersion {
		return fmt.Errorf("%w: expected %q, got %q", ErrUnsupportedVersion, CurrentContractVersion, trimmedVersion)
	}

	trimmedPath := strings.TrimSpace(d.PathPattern)
	if trimmedPath == "" {
		return ErrBlankPathPattern
	}

	trimmedEnvRef := strings.TrimSpace(d.ErrorEnvelopeRef)
	if trimmedEnvRef == "" {
		return ErrBlankErrorEnvelopeRef
	}

	if len(d.SupportedMethods) == 0 {
		return ErrNoSupportedMethods
	}

	for _, m := range d.SupportedMethods {
		upper := strings.ToUpper(strings.TrimSpace(m))
		if !validHTTPMethods[upper] {
			return fmt.Errorf("%w: %q", ErrInvalidMethod, m)
		}
	}

	return nil
}

// Equals evaluates whether two contract descriptors are structurally equivalent.
func (d *ContractDescriptor) Equals(other *ContractDescriptor) bool {
	if other == nil {
		return false
	}
	if strings.TrimSpace(d.ContractID) != strings.TrimSpace(other.ContractID) ||
		strings.TrimSpace(d.Name) != strings.TrimSpace(other.Name) ||
		strings.TrimSpace(d.Version) != strings.TrimSpace(other.Version) ||
		strings.TrimSpace(d.Description) != strings.TrimSpace(other.Description) ||
		strings.TrimSpace(d.PathPattern) != strings.TrimSpace(other.PathPattern) ||
		strings.TrimSpace(d.Protocol) != strings.TrimSpace(other.Protocol) ||
		strings.TrimSpace(d.ErrorEnvelopeRef) != strings.TrimSpace(other.ErrorEnvelopeRef) ||
		strings.TrimSpace(d.ProvisionalState) != strings.TrimSpace(other.ProvisionalState) {
		return false
	}

	if len(d.SupportedMethods) != len(other.SupportedMethods) {
		return false
	}
	methodsA := append([]string(nil), d.SupportedMethods...)
	methodsB := append([]string(nil), other.SupportedMethods...)
	sort.Strings(methodsA)
	sort.Strings(methodsB)
	for i := range methodsA {
		if strings.ToUpper(strings.TrimSpace(methodsA[i])) != strings.ToUpper(strings.TrimSpace(methodsB[i])) {
			return false
		}
	}

	if len(d.RequiredHeaders) != len(other.RequiredHeaders) {
		return false
	}
	headersA := append([]string(nil), d.RequiredHeaders...)
	headersB := append([]string(nil), other.RequiredHeaders...)
	sort.Strings(headersA)
	sort.Strings(headersB)
	for i := range headersA {
		if strings.TrimSpace(headersA[i]) != strings.TrimSpace(headersB[i]) {
			return false
		}
	}

	if len(d.Metadata) != len(other.Metadata) {
		return false
	}
	for k, v := range d.Metadata {
		if other.Metadata[k] != v {
			return false
		}
	}

	return true
}

// ContractRegistry provides a thread-safe, in-memory repository for versioned API contract descriptors.
type ContractRegistry struct {
	mu        sync.RWMutex
	contracts map[string]ContractDescriptor
}

// NewContractRegistry initializes a new empty in-memory ContractRegistry.
func NewContractRegistry() *ContractRegistry {
	return &ContractRegistry{
		contracts: make(map[string]ContractDescriptor),
	}
}

// RegisterContract validates and records a versioned API contract descriptor.
// Invariants:
// - Re-registering an identical descriptor is idempotent and succeeds cleanly.
// - Registering a differing descriptor with an existing ContractID returns ErrDuplicateContractID.
// - Malformed or blank descriptors fail closed with descriptive validation errors.
func (r *ContractRegistry) RegisterContract(desc ContractDescriptor) error {
	if err := desc.Validate(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	key := strings.TrimSpace(desc.ContractID)
	if existing, exists := r.contracts[key]; exists {
		if existing.Equals(&desc) {
			// Idempotent re-registration of identical descriptor
			return nil
		}
		return fmt.Errorf("%w: existing name %q differs from candidate %q", ErrDuplicateContractID, existing.Name, desc.Name)
	}

	// Normalize methods to uppercase
	normalizedMethods := make([]string, len(desc.SupportedMethods))
	for i, m := range desc.SupportedMethods {
		normalizedMethods[i] = strings.ToUpper(strings.TrimSpace(m))
	}
	sort.Strings(normalizedMethods)

	// Ensure provisional state is set
	provisionalState := desc.ProvisionalState
	if strings.TrimSpace(provisionalState) == "" {
		provisionalState = ProvisionalStatus
	}

	protocol := desc.Protocol
	if strings.TrimSpace(protocol) == "" {
		protocol = DefaultAPIProtocol
	}

	normalized := ContractDescriptor{
		ContractID:       key,
		Name:             strings.TrimSpace(desc.Name),
		Version:          strings.TrimSpace(desc.Version),
		Description:      strings.TrimSpace(desc.Description),
		PathPattern:      strings.TrimSpace(desc.PathPattern),
		Protocol:         protocol,
		ErrorEnvelopeRef: strings.TrimSpace(desc.ErrorEnvelopeRef),
		SupportedMethods: normalizedMethods,
		RequiredHeaders:  append([]string(nil), desc.RequiredHeaders...),
		ProvisionalState: provisionalState,
		Metadata:         make(map[string]string),
	}
	for k, v := range desc.Metadata {
		normalized.Metadata[k] = v
	}

	r.contracts[key] = normalized
	return nil
}

// GetContract retrieves a registered contract descriptor by its unique ID.
func (r *ContractRegistry) GetContract(contractID string) (ContractDescriptor, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	key := strings.TrimSpace(contractID)
	if key == "" {
		return ContractDescriptor{}, ErrBlankContractID
	}

	desc, exists := r.contracts[key]
	if !exists {
		return ContractDescriptor{}, fmt.Errorf("%w: %q", ErrContractNotFound, key)
	}

	return desc, nil
}

// HasContract checks whether a given contract ID is registered.
func (r *ContractRegistry) HasContract(contractID string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	key := strings.TrimSpace(contractID)
	if key == "" {
		return false
	}
	_, exists := r.contracts[key]
	return exists
}

// ListContracts returns all registered contract descriptors sorted deterministically by ContractID.
func (r *ContractRegistry) ListContracts() []ContractDescriptor {
	r.mu.RLock()
	defer r.mu.RUnlock()

	keys := make([]string, 0, len(r.contracts))
	for k := range r.contracts {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	results := make([]ContractDescriptor, len(keys))
	for i, k := range keys {
		results[i] = r.contracts[k]
	}
	return results
}

// Count returns the number of registered contracts.
func (r *ContractRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.contracts)
}
