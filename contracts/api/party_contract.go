// Package api provides provisional static API contract models for v0.3.0.
// This package is an isolated, dependency-free local specification slice
// pending formal Sole Human Owner architecture gate H030-008.
// Zero runtime execution, server binding, or external standard compatibility is claimed.
package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	// ErrEmptyPartyID indicates missing party identifier.
	ErrEmptyPartyID = errors.New("party_id cannot be empty")
	// ErrEmptyDisplayName indicates missing display name.
	// ErrEmptyTenantID indicates missing tenant identifier.
	ErrEmptyTenantID = errors.New("tenant_id cannot be empty")
	ErrEmptyDisplayName = errors.New("display_name cannot be empty")
	// ErrInvalidContractPartyType indicates an invalid party type string.
	ErrInvalidContractPartyType = errors.New("party_type must be CLIENT, CONTRACTOR, SUBCONTRACTOR, PARTNER, or AUDITOR")
	// ErrInvalidContractStatus indicates an invalid lifecycle status string.
	ErrInvalidContractStatus = errors.New("status must be ACTIVE or ARCHIVED")
	// ErrEmptyParticipationID indicates missing participation identifier.
	ErrEmptyParticipationID = errors.New("participation_id cannot be empty")
	// ErrEmptyProjectID indicates missing project identifier.
	ErrEmptyProjectID = errors.New("project_id cannot be empty")
	// ErrInvalidContractRole indicates an unapproved participation role string.
	ErrInvalidContractRole = errors.New("role must be CONTRACTOR_WORKER, SITE_SAFETY_LEAD, CLIENT_AUDITOR, CONSULTANT, or SUBCONTRACTOR_LEAD")
	// ErrInvalidDateFormat indicates timestamp is not RFC3339.
	ErrInvalidDateFormat = errors.New("timestamp must conform to RFC3339 format")
	// ErrRedactionViolation indicates forbidden internal, sensitive, or authority fields detected in public payload.
	ErrRedactionViolation = errors.New("contract payload contains forbidden internal, PII, or authority field")
)

// ProhibitedFieldPatterns lists JSON keys that must NEVER be present in public contract representations.
var ProhibitedFieldPatterns = []string{
	"password",
	"secret",
	"token",
	"national_id",
	"ssn",
	"tax_id",
	"email",
	"phone",
	"is_admin",
	"admin",
	"permissions",
	"database_id",
	"raw_id",
}

// PartySummaryView represents the sanitized, public-facing representation of an external party.
// Invariant: Omits database autoincrements, sensitive contact details, and internal credentials.
type PartySummaryView struct {
	PartyID     string `json:"party_id"`
	TenantID    string `json:"tenant_id"`
	DisplayName string `json:"display_name"`
	PartyType   string `json:"party_type"`
	Status      string `json:"status"`
}

// Validate checks all mandatory invariants for PartySummaryView.
func (v *PartySummaryView) Validate() error {
	if strings.TrimSpace(v.PartyID) == "" {
		return ErrEmptyPartyID
	}
	if strings.TrimSpace(v.TenantID) == "" {
		return ErrEmptyTenantID
	}
	if strings.TrimSpace(v.DisplayName) == "" {
		return ErrEmptyDisplayName
	}

	switch v.PartyType {
	case "CLIENT", "CONTRACTOR", "SUBCONTRACTOR", "PARTNER", "AUDITOR":
	default:
		return ErrInvalidContractPartyType
	}

	switch v.Status {
	case "ACTIVE", "ARCHIVED":
	default:
		return ErrInvalidContractStatus
	}

	return nil
}

// ProjectParticipationView represents the public-facing bounded participation of a party in a project.
// Invariant: Strict scope boundary; omits internal sponsor private details and permission bitmasks.
type ProjectParticipationView struct {
	ParticipationID string `json:"participation_id"`
	TenantID        string `json:"tenant_id"`
	PartyID         string `json:"party_id"`
	ProjectID       string `json:"project_id"`
	SiteID          string `json:"site_id,omitempty"`
	Role            string `json:"role"`
	ValidFrom       string `json:"valid_from"`
	ValidTo         string `json:"valid_to"`
	Status          string `json:"status"`
}

// Validate checks all mandatory invariants for ProjectParticipationView.
func (v *ProjectParticipationView) Validate() error {
	if strings.TrimSpace(v.ParticipationID) == "" {
		return ErrEmptyParticipationID
	}
	if strings.TrimSpace(v.TenantID) == "" {
		return ErrEmptyTenantID
	}
	if strings.TrimSpace(v.PartyID) == "" {
		return ErrEmptyPartyID
	}
	if strings.TrimSpace(v.ProjectID) == "" {
		return ErrEmptyProjectID
	}

	switch v.Role {
	case "CONTRACTOR_WORKER", "SITE_SAFETY_LEAD", "CLIENT_AUDITOR", "CONSULTANT", "SUBCONTRACTOR_LEAD":
	default:
		return ErrInvalidContractRole
	}

	tFrom, err := time.Parse(time.RFC3339, v.ValidFrom)
	if err != nil {
		return fmt.Errorf("%w: valid_from: %v", ErrInvalidDateFormat, err)
	}
	tTo, err := time.Parse(time.RFC3339, v.ValidTo)
	if err != nil {
		return fmt.Errorf("%w: valid_to: %v", ErrInvalidDateFormat, err)
	}
	if tTo.Before(tFrom) || tTo.Equal(tFrom) {
		return errors.New("valid_to must be strictly after valid_from")
	}

	switch v.Status {
	case "ACTIVE", "ARCHIVED":
	default:
		return ErrInvalidContractStatus
	}

	return nil
}

// AssertRedactedPartyContract asserts that a serialized JSON payload does not contain
// any forbidden internal identifiers, PII keys, or authority escalation fields.
func AssertRedactedPartyContract(payload []byte) error {
	var parsed map[string]any
	if err := json.Unmarshal(payload, &parsed); err != nil {
		return fmt.Errorf("invalid json payload: %w", err)
	}

	for k := range parsed {
		lowerK := strings.ToLower(k)
		for _, pattern := range ProhibitedFieldPatterns {
			if strings.Contains(lowerK, pattern) {
				return fmt.Errorf("%w: key %q matches prohibited pattern %q", ErrRedactionViolation, k, pattern)
			}
		}
	}

	return nil
}
