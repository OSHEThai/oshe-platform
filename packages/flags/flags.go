// Package flags implements synthetic-only, default-off feature flag evaluation.
package flags

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
)

var (
	ErrInvalidKey        = errors.New("invalid feature flag key")
	ErrDuplicateKey      = errors.New("duplicate feature flag key")
	ErrInvalidDefinition = errors.New("invalid feature flag definition")
)

var keyPattern = regexp.MustCompile(`^[a-z][a-z0-9.-]{2,63}$`)

// Environment is deliberately limited to the synthetic qualification profile.
type Environment string

const Synthetic Environment = "SYNTHETIC_ONLY"

// Definition is a governed, non-authorizing flag declaration.
type Definition struct {
	Key            string      `json:"key"`
	DefaultEnabled bool        `json:"default_enabled"`
	Stage          Environment `json:"stage"`
	AllowedTenants []string    `json:"allowed_tenants,omitempty"`
	AllowedRoles   []string    `json:"allowed_roles,omitempty"`
}

// Subject is the synthetic caller context. A flag never grants permission.
type Subject struct {
	Tenant string
	Roles  []string
}

// Registry stores immutable copies of validated definitions.
type Registry struct {
	mu   sync.RWMutex
	defs map[string]Definition
}

// NewRegistry accepts only unique, valid, default-off synthetic definitions.
func NewRegistry(definitions []Definition) (*Registry, error) {
	r := &Registry{defs: make(map[string]Definition, len(definitions))}
	for _, definition := range definitions {
		if err := r.Register(definition); err != nil {
			return nil, err
		}
	}
	return r, nil
}

// Register records a definition once. Duplicate keys are denied.
func (r *Registry) Register(definition Definition) error {
	if err := validate(definition); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.defs[definition.Key]; exists {
		return fmt.Errorf("%w: %s", ErrDuplicateKey, definition.Key)
	}
	definition.AllowedTenants = append([]string(nil), definition.AllowedTenants...)
	definition.AllowedRoles = append([]string(nil), definition.AllowedRoles...)
	r.defs[definition.Key] = definition
	return nil
}

// Evaluate fails closed. Missing, malformed, disabled, non-synthetic, or out-of-scope flags are false.
func (r *Registry) Evaluate(key string, environment Environment, subject Subject) bool {
	if !keyPattern.MatchString(key) || environment != Synthetic || strings.TrimSpace(subject.Tenant) == "" {
		return false
	}
	r.mu.RLock()
	definition, found := r.defs[key]
	r.mu.RUnlock()
	if !found || !definition.DefaultEnabled || definition.Stage != Synthetic {
		return false
	}
	if len(definition.AllowedTenants) > 0 && !contains(definition.AllowedTenants, subject.Tenant) {
		return false
	}
	if len(definition.AllowedRoles) > 0 && !intersects(definition.AllowedRoles, subject.Roles) {
		return false
	}
	return true
}

// LoadJSON validates a canonical registry document and returns an independent registry.
func LoadJSON(data []byte) (*Registry, error) {
	var definitions []Definition
	if err := json.Unmarshal(data, &definitions); err != nil {
		return nil, fmt.Errorf("invalid feature flag registry JSON: %w", err)
	}
	return NewRegistry(definitions)
}

func validate(definition Definition) error {
	if !keyPattern.MatchString(definition.Key) {
		return ErrInvalidKey
	}
	if definition.DefaultEnabled || definition.Stage != Synthetic {
		return ErrInvalidDefinition
	}
	for _, tenant := range definition.AllowedTenants {
		if strings.TrimSpace(tenant) == "" {
			return ErrInvalidDefinition
		}
	}
	for _, role := range definition.AllowedRoles {
		if strings.TrimSpace(role) == "" {
			return ErrInvalidDefinition
		}
	}
	return nil
}

func contains(values []string, wanted string) bool {
	return sort.SearchStrings(sorted(values), wanted) < len(values) && sorted(values)[sort.SearchStrings(sorted(values), wanted)] == wanted
}

func intersects(allowed, actual []string) bool {
	for _, role := range actual {
		if contains(allowed, role) {
			return true
		}
	}
	return false
}

func sorted(values []string) []string {
	copyOf := append([]string(nil), values...)
	sort.Strings(copyOf)
	return copyOf
}
