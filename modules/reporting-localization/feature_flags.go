package reporting

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// FlagStage represents the lifecycle maturity stage of a feature flag.
type FlagStage string

const (
	StageExperimental FlagStage = "EXPERIMENTAL"
	StageAlpha        FlagStage = "ALPHA"
	StageBeta         FlagStage = "BETA"
	StageGA           FlagStage = "GA"
	StageDeprecated   FlagStage = "DEPRECATED"
)

const (
	// DefaultFlagAuthorityNotice is the standard required disclaimer that flags never grant authority.
	DefaultFlagAuthorityNotice = "FEATURE_FLAG_NON_AUTHORITY: Feature flags control client exposure and operational gates only. Flags never grant security authority or bypass authorization controls."
)

var (
	ErrBlankFlagID               = errors.New("flag ID cannot be blank")
	ErrDuplicateFlagID           = errors.New("duplicate feature flag ID")
	ErrFlagNotFound              = errors.New("feature flag not found in registry")
	ErrMustDefaultOff            = errors.New("governed feature flags must explicitly declare DefaultOff as true")
	ErrAuthorityBypassDenied     = errors.New("feature flags cannot grant authority or bypass authorization controls")
	ErrAccessibilityNonCompliant = errors.New("feature flag fails accessibility qualification standards")
	ErrInvalidRolloutPercentage  = errors.New("rollout percentage must be between 0 and 100 inclusive")
)

// RolloutMetadata governs targeted tenant, role, and temporal exposure.
type RolloutMetadata struct {
	Percentage     int       `json:"percentage"`
	AllowedTenants []string  `json:"allowed_tenants,omitempty"`
	AllowedRoles   []string  `json:"allowed_roles,omitempty"`
	EffectiveFrom  time.Time `json:"effective_from,omitempty"`
	EffectiveTo    time.Time `json:"effective_to,omitempty"`
}

// AccessibilityMetadata captures qualification fixtures for UI feature flags.
type AccessibilityMetadata struct {
	KeyboardNavigable bool   `json:"keyboard_navigable"`
	ScreenReaderLabel string `json:"screen_reader_label"`
	AriaRole          string `json:"aria_role"`
	ContrastCertified bool   `json:"contrast_certified"`
}

// FeatureFlag defines the governed contract of a feature toggle.
type FeatureFlag struct {
	FlagID        string                `json:"flag_id"`
	Title         string                `json:"title"`
	Description   string                `json:"description"`
	DefaultOff    bool                  `json:"default_off"`
	Enabled       bool                  `json:"enabled"`
	Stage         FlagStage             `json:"stage"`
	Owner         string                `json:"owner"`
	Rollout       RolloutMetadata       `json:"rollout"`
	Accessibility AccessibilityMetadata `json:"accessibility"`
	CreatedAt     time.Time             `json:"created_at"`
	UpdatedAt     time.Time             `json:"updated_at"`
}

// EvaluationContext contains caller environmental attributes for evaluating flag exposure.
type EvaluationContext struct {
	TenantID       string    `json:"tenant_id"`
	SubjectID      string    `json:"subject_id"`
	CallerRoles    []string  `json:"caller_roles,omitempty"`
	IsAuthorized   bool      `json:"is_authorized"`
	EvaluationTime time.Time `json:"evaluation_time"`
}

// EvaluationResult captures the deterministic exposure decision and safety notices.
type EvaluationResult struct {
	FlagID        string                `json:"flag_id"`
	Exposed       bool                  `json:"exposed"`
	Reason        string                `json:"reason"`
	Accessibility AccessibilityMetadata `json:"accessibility"`
	AuthorityNote string                `json:"authority_note"`
}

// FeatureFlagRegistry coordinates thread-safe in-memory feature flag governance.
type FeatureFlagRegistry struct {
	mu    sync.RWMutex
	flags map[string]FeatureFlag
	clock func() time.Time
}

// NewFeatureFlagRegistry constructs a new FeatureFlagRegistry.
func NewFeatureFlagRegistry(clock func() time.Time) *FeatureFlagRegistry {
	if clock == nil {
		clock = time.Now
	}
	return &FeatureFlagRegistry{
		flags: make(map[string]FeatureFlag),
		clock: clock,
	}
}

// RegisterFlag registers a new feature flag.
// Enforces:
// - FlagID must not be blank
// - Duplicate FlagID is rejected
// - DefaultOff must be true (default-off invariant)
func (r *FeatureFlagRegistry) RegisterFlag(flag FeatureFlag) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	id := strings.TrimSpace(flag.FlagID)
	if id == "" {
		return ErrBlankFlagID
	}
	if _, exists := r.flags[id]; exists {
		return ErrDuplicateFlagID
	}

	if !flag.DefaultOff {
		return ErrMustDefaultOff
	}

	if flag.Rollout.Percentage < 0 || flag.Rollout.Percentage > 100 {
		return ErrInvalidRolloutPercentage
	}
	tenantsCopy := make([]string, len(flag.Rollout.AllowedTenants))
	copy(tenantsCopy, flag.Rollout.AllowedTenants)

	rolesCopy := make([]string, len(flag.Rollout.AllowedRoles))
	copy(rolesCopy, flag.Rollout.AllowedRoles)

	now := r.clock().UTC()
	r.flags[id] = FeatureFlag{
		FlagID:      id,
		Title:       strings.TrimSpace(flag.Title),
		Description: strings.TrimSpace(flag.Description),
		DefaultOff:  true,
		Enabled:     flag.Enabled,
		Stage:       flag.Stage,
		Owner:       strings.TrimSpace(flag.Owner),
		Rollout: RolloutMetadata{
			Percentage:     flag.Rollout.Percentage,
			AllowedTenants: tenantsCopy,
			AllowedRoles:   rolesCopy,
			EffectiveFrom:  flag.Rollout.EffectiveFrom,
			EffectiveTo:    flag.Rollout.EffectiveTo,
		},
		Accessibility: flag.Accessibility,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	return nil
}

// SetFlagState updates the active toggle state of a flag.
func (r *FeatureFlagRegistry) SetFlagState(flagID string, enabled bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	id := strings.TrimSpace(flagID)
	f, exists := r.flags[id]
	if !exists {
		return ErrFlagNotFound
	}

	f.Enabled = enabled
	f.UpdatedAt = r.clock().UTC()
	r.flags[id] = f
	return nil
}

// SetRolloutPercentage updates the rollout percentage for an existing flag.
func (r *FeatureFlagRegistry) SetRolloutPercentage(flagID string, percentage int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	id := strings.TrimSpace(flagID)
	f, exists := r.flags[id]
	if !exists {
		return ErrFlagNotFound
	}

	if percentage < 0 || percentage > 100 {
		return ErrInvalidRolloutPercentage
	}

	f.Rollout.Percentage = percentage
	f.UpdatedAt = r.clock().UTC()
	r.flags[id] = f
	return nil
}

// Evaluate evaluates whether a feature flag should be exposed to a caller context.
// Invariant Guarantees:
// 1. If caller is not authorized (ctx.IsAuthorized == false), exposure is ALWAYS denied.
// 2. If flag is not found, exposure is ALWAYS denied (default off).
// 3. If flag is disabled, exposure is ALWAYS denied (safe fallback).
// 4. If current time is outside effective time window, exposure is denied (stale config).
// 5. If tenant is not in allowed tenants list, exposure is denied.
// 6. If caller roles do not intersect with allowed roles, exposure is denied.
// 7. If rollout percentage is 0%, exposure is denied.
// 8. If fractional rollout (<100%), caller subject must be deterministically assigned to a qualifying cohort bucket.
func (r *FeatureFlagRegistry) Evaluate(flagID string, ctx EvaluationContext) EvaluationResult {
	r.mu.RLock()
	defer r.mu.RUnlock()

	id := strings.TrimSpace(flagID)
	f, exists := r.flags[id]
	if !exists {
		return EvaluationResult{
			FlagID:        id,
			Exposed:       false,
			Reason:        "flag not found in registry (default off)",
			AuthorityNote: DefaultFlagAuthorityNotice,
		}
	}

	// 1. Critical Invariant: Flags NEVER grant authority or bypass authorization
	if !ctx.IsAuthorized {
		return EvaluationResult{
			FlagID:        id,
			Exposed:       false,
			Reason:        "underlying authorization denied: flags cannot bypass security controls",
			Accessibility: f.Accessibility,
			AuthorityNote: DefaultFlagAuthorityNotice,
		}
	}

	// 2. Safe Disablement: If disabled, clean fallback with zero exposed state
	if !f.Enabled {
		return EvaluationResult{
			FlagID:        id,
			Exposed:       false,
			Reason:        "flag disabled (safe fallback active)",
			Accessibility: f.Accessibility,
			AuthorityNote: DefaultFlagAuthorityNotice,
		}
	}

	// 3. Temporal Rollout Boundary (Stale configuration check)
	evalTime := ctx.EvaluationTime
	if evalTime.IsZero() {
		evalTime = r.clock().UTC()
	}

	if !f.Rollout.EffectiveFrom.IsZero() && evalTime.Before(f.Rollout.EffectiveFrom) {
		return EvaluationResult{
			FlagID:        id,
			Exposed:       false,
			Reason:        "evaluation time precedes rollout effective window",
			Accessibility: f.Accessibility,
			AuthorityNote: DefaultFlagAuthorityNotice,
		}
	}

	if !f.Rollout.EffectiveTo.IsZero() && evalTime.After(f.Rollout.EffectiveTo) {
		return EvaluationResult{
			FlagID:        id,
			Exposed:       false,
			Reason:        "rollout effective window expired (stale configuration)",
			Accessibility: f.Accessibility,
			AuthorityNote: DefaultFlagAuthorityNotice,
		}
	}

	// 4. Tenant Scope Allowlist
	if len(f.Rollout.AllowedTenants) > 0 {
		tenantMatch := false
		trimmedTenant := strings.TrimSpace(ctx.TenantID)
		for _, allowed := range f.Rollout.AllowedTenants {
			if allowed == trimmedTenant {
				tenantMatch = true
				break
			}
		}
		if !tenantMatch {
			return EvaluationResult{
				FlagID:        id,
				Exposed:       false,
				Reason:        fmt.Sprintf("tenant %q not included in rollout allowlist", trimmedTenant),
				Accessibility: f.Accessibility,
				AuthorityNote: DefaultFlagAuthorityNotice,
			}
		}
	}

	// 5. Role Scope Allowlist
	if len(f.Rollout.AllowedRoles) > 0 {
		roleMatch := false
		for _, allowedRole := range f.Rollout.AllowedRoles {
			for _, callerRole := range ctx.CallerRoles {
				if allowedRole == strings.TrimSpace(callerRole) {
					roleMatch = true
					break
				}
			}
			if roleMatch {
				break
			}
		}
		if !roleMatch {
			return EvaluationResult{
				FlagID:        id,
				Exposed:       false,
				Reason:        "caller roles do not match required rollout exposure roles",
				Accessibility: f.Accessibility,
				AuthorityNote: DefaultFlagAuthorityNotice,
			}
		}
	}
	// 6. Rollout Percentage & Deterministic Cohort Evaluation
	if f.Rollout.Percentage <= 0 {
		return EvaluationResult{
			FlagID:        id,
			Exposed:       false,
			Reason:        "subject excluded: rollout percentage is 0%",
			Accessibility: f.Accessibility,
			AuthorityNote: DefaultFlagAuthorityNotice,
		}
	}

	if f.Rollout.Percentage < 100 {
		trimmedSubject := strings.TrimSpace(ctx.SubjectID)
		if trimmedSubject == "" {
			return EvaluationResult{
				FlagID:        id,
				Exposed:       false,
				Reason:        "subject identifier required for fractional rollout cohort evaluation",
				Accessibility: f.Accessibility,
				AuthorityNote: DefaultFlagAuthorityNotice,
			}
		}

		bucket := ComputeCohortBucket(id, trimmedSubject)
		if bucket >= f.Rollout.Percentage {
			return EvaluationResult{
				FlagID:        id,
				Exposed:       false,
				Reason:        fmt.Sprintf("subject %q assigned to cohort bucket %d outside rollout percentage %d%%", trimmedSubject, bucket, f.Rollout.Percentage),
				Accessibility: f.Accessibility,
				AuthorityNote: DefaultFlagAuthorityNotice,
			}
		}
	}

	return EvaluationResult{
		FlagID:        id,
		Exposed:       true,
		Reason:        "all rollout and governance criteria satisfied",
		Accessibility: f.Accessibility,
		AuthorityNote: DefaultFlagAuthorityNotice,
	}
}

// ComputeCohortBucket deterministically maps a flag ID and subject ID to a bucket in [0, 99].
// It uses a SHA-256 hash to ensure uniform, reproducible cohort assignment across synthetic subjects.
func ComputeCohortBucket(flagID, subjectID string) int {
	cleanFlag := strings.TrimSpace(flagID)
	cleanSubject := strings.TrimSpace(subjectID)
	h := sha256.Sum256([]byte(cleanFlag + ":" + cleanSubject))
	val := binary.BigEndian.Uint64(h[:8])
	return int(val % 100)
}
