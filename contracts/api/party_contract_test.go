package api

import (
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestPartySummaryView_ValidSerializationAndValidation(t *testing.T) {
	view := &PartySummaryView{
		PartyID:     "prt_01j9876543210zyxwvutsrqpon",
		TenantID:    "ten_01j9876543210zyxwvutsrqpon",
		DisplayName: "Siam Industrial Safety Consultants",
		PartyType:   "CONTRACTOR",
		Status:      "ACTIVE",
	}

	if err := view.Validate(); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}

	b, err := json.Marshal(view)
	if err != nil {
		t.Fatalf("failed to marshal view: %v", err)
	}

	if err := AssertRedactedPartyContract(b); err != nil {
		t.Errorf("expected clean public contract, got %v", err)
	}

	var roundtrip PartySummaryView
	if err := json.Unmarshal(b, &roundtrip); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if roundtrip.PartyID != view.PartyID || roundtrip.DisplayName != view.DisplayName {
		t.Errorf("roundtrip mismatch: %+v", roundtrip)
	}
}

func TestPartySummaryView_ValidationRejections(t *testing.T) {
	cases := []struct {
		name    string
		view    PartySummaryView
		wantErr error
	}{
		{
			name:    "empty party_id",
			view:    PartySummaryView{TenantID: "ten_01", DisplayName: "Apex", PartyType: "CLIENT", Status: "ACTIVE"},
			wantErr: ErrEmptyPartyID,
		},
		{
			name:    "empty tenant_id",
			view:    PartySummaryView{PartyID: "prt_01", DisplayName: "Apex", PartyType: "CLIENT", Status: "ACTIVE"},
			wantErr: ErrEmptyTenantID,
		},
		{
			name:    "empty display_name",
			view:    PartySummaryView{PartyID: "prt_01", TenantID: "ten_01", PartyType: "CLIENT", Status: "ACTIVE"},
			wantErr: ErrEmptyDisplayName,
		},
		{
			name:    "invalid party_type",
			view:    PartySummaryView{PartyID: "prt_01", TenantID: "ten_01", DisplayName: "Apex", PartyType: "INVALID", Status: "ACTIVE"},
			wantErr: ErrInvalidContractPartyType,
		},
		{
			name:    "invalid status",
			view:    PartySummaryView{PartyID: "prt_01", TenantID: "ten_01", DisplayName: "Apex", PartyType: "AUDITOR", Status: "PENDING"},
			wantErr: ErrInvalidContractStatus,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.view.Validate()
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("expected error %v, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestProjectParticipationView_ValidSerializationAndValidation(t *testing.T) {
	validFrom := time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC).Format(time.RFC3339)
	validTo := time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC).Format(time.RFC3339)

	view := &ProjectParticipationView{
		ParticipationID: "ptp_01j9876543210zyxwvutsrqpon",
		TenantID:        "ten_01j9876543210zyxwvutsrqpon",
		PartyID:         "prt_01j9876543210zyxwvutsrqpon",
		ProjectID:       "prj_01j9876543210zyxwvutsrqpon",
		SiteID:          "ste_01j9876543210zyxwvutsrqpon",
		Role:            "SITE_SAFETY_LEAD",
		ValidFrom:       validFrom,
		ValidTo:         validTo,
		Status:          "ACTIVE",
		NestingDepth:    0,
	}

	if err := view.Validate(); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}

	b, err := json.Marshal(view)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	if err := AssertRedactedPartyContract(b); err != nil {
		t.Errorf("expected clean redacted contract, got %v", err)
	}
}

func TestProjectParticipationView_NestedSubcontractorValidation(t *testing.T) {
	validFrom := time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC).Format(time.RFC3339)
	validTo := time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC).Format(time.RFC3339)

	// Valid subcontractor view at depth 1
	subView := &ProjectParticipationView{
		ParticipationID:       "ptp_sub_01",
		TenantID:              "ten_01",
		PartyID:               "prt_sub_01",
		ProjectID:             "prj_01",
		SiteID:                "ste_01",
		Role:                  "SUBCONTRACTOR_LEAD",
		ValidFrom:             validFrom,
		ValidTo:               validTo,
		Status:                "ACTIVE",
		ParentParticipationID: "ptp_prime_01",
		NestingDepth:          1,
	}

	if err := subView.Validate(); err != nil {
		t.Fatalf("unexpected validation error for valid subcontractor: %v", err)
	}

	b, err := json.Marshal(subView)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}
	if err := AssertRedactedPartyContract(b); err != nil {
		t.Errorf("expected clean subcontractor payload, got: %v", err)
	}
}

func TestProjectParticipationView_InvalidNestingRejections(t *testing.T) {
	from := time.Now().UTC().Format(time.RFC3339)
	to := time.Now().Add(48 * time.Hour).UTC().Format(time.RFC3339)

	// Depth 2 (sub-subcontractor) rejected
	v := ProjectParticipationView{
		ParticipationID:       "ptp_sub2",
		TenantID:              "ten_01",
		PartyID:               "prt_sub2",
		ProjectID:             "prj_01",
		Role:                  "CONTRACTOR_WORKER",
		ValidFrom:             from,
		ValidTo:               to,
		Status:                "ACTIVE",
		ParentParticipationID: "ptp_sub1",
		NestingDepth:          2,
	}
	if err := v.Validate(); !errors.Is(err, ErrInvalidNestingDepth) {
		t.Errorf("expected ErrInvalidNestingDepth for depth 2, got %v", err)
	}

	// Depth 1 with missing parent ID rejected
	v = ProjectParticipationView{
		ParticipationID: "ptp_sub1",
		TenantID:        "ten_01",
		PartyID:         "prt_sub1",
		ProjectID:       "prj_01",
		Role:            "SUBCONTRACTOR_LEAD",
		ValidFrom:       from,
		ValidTo:         to,
		Status:          "ACTIVE",
		NestingDepth:    1,
	}
	if err := v.Validate(); err == nil {
		t.Errorf("expected error for depth 1 without parent ID")
	}

	// Depth 0 with non-empty parent ID rejected
	v = ProjectParticipationView{
		ParticipationID:       "ptp_prime1",
		TenantID:              "ten_01",
		PartyID:               "prt_prime1",
		ProjectID:             "prj_01",
		Role:                  "SITE_SAFETY_LEAD",
		ValidFrom:             from,
		ValidTo:               to,
		Status:                "ACTIVE",
		ParentParticipationID: "ptp_other",
		NestingDepth:          0,
	}
	if err := v.Validate(); err == nil {
		t.Errorf("expected error for depth 0 with parent ID")
	}
}

func TestProjectParticipationView_ValidationRejections(t *testing.T) {
	from := time.Now().UTC().Format(time.RFC3339)
	to := time.Now().Add(48 * time.Hour).UTC().Format(time.RFC3339)

	// Missing participation ID
	v := ProjectParticipationView{TenantID: "ten_01", PartyID: "prt_01", ProjectID: "prj_01", Role: "CONSULTANT", ValidFrom: from, ValidTo: to, Status: "ACTIVE"}
	if err := v.Validate(); !errors.Is(err, ErrEmptyParticipationID) {
		t.Errorf("expected ErrEmptyParticipationID, got %v", err)
	}

	// Missing project ID
	v = ProjectParticipationView{ParticipationID: "ptp_01", TenantID: "ten_01", PartyID: "prt_01", Role: "CONSULTANT", ValidFrom: from, ValidTo: to, Status: "ACTIVE"}
	if err := v.Validate(); !errors.Is(err, ErrEmptyProjectID) {
		t.Errorf("expected ErrEmptyProjectID, got %v", err)
	}

	// Invalid role
	v = ProjectParticipationView{ParticipationID: "ptp_01", TenantID: "ten_01", PartyID: "prt_01", ProjectID: "prj_01", Role: "ROOT_ADMIN", ValidFrom: from, ValidTo: to, Status: "ACTIVE"}
	if err := v.Validate(); !errors.Is(err, ErrInvalidContractRole) {
		t.Errorf("expected ErrInvalidContractRole, got %v", err)
	}

	// Malformed date
	v = ProjectParticipationView{ParticipationID: "ptp_01", TenantID: "ten_01", PartyID: "prt_01", ProjectID: "prj_01", Role: "CONSULTANT", ValidFrom: "not-a-date", ValidTo: to, Status: "ACTIVE"}
	if err := v.Validate(); !errors.Is(err, ErrInvalidDateFormat) {
		t.Errorf("expected ErrInvalidDateFormat, got %v", err)
	}

	// Inverted time window
	v = ProjectParticipationView{ParticipationID: "ptp_01", TenantID: "ten_01", PartyID: "prt_01", ProjectID: "prj_01", Role: "CONSULTANT", ValidFrom: to, ValidTo: from, Status: "ACTIVE"}
	if err := v.Validate(); err == nil {
		t.Errorf("expected error for inverted time window")
	}
}

func TestAssertRedactedPartyContract_BoundaryEnforcement(t *testing.T) {
	// Clean payload passes
	clean := []byte(`{"party_id":"prt_01","tenant_id":"ten_01","display_name":"Acme","party_type":"CONTRACTOR","status":"ACTIVE"}`)
	if err := AssertRedactedPartyContract(clean); err != nil {
		t.Errorf("expected clean payload to pass, got %v", err)
	}

	// Payloads leaking PII, secrets, or internal/authority fields must be rejected
	leakCases := []struct {
		name    string
		payload string
	}{
		{"internal database_id leak", `{"party_id":"prt_01","database_id":1042}`},
		{"password hash leak", `{"party_id":"prt_01","password_hash":"$2a$12$..."}`},
		{"bearer token leak", `{"party_id":"prt_01","token":"oshe_tok_123"}`},
		{"national_id PII leak", `{"party_id":"prt_01","national_id":"1234567890123"}`},
		{"email leak", `{"party_id":"prt_01","email":"contractor@example.com"}`},
		{"phone leak", `{"party_id":"prt_01","phone":"+66812345678"}`},
		{"admin privilege leak", `{"party_id":"prt_01","is_admin":true}`},
		{"permissions bitmask leak", `{"party_id":"prt_01","permissions":["*"]}`},
		{"ssn leak", `{"party_id":"prt_01","ssn":"000-00-0000"}`},
		{"sponsor private key leak", `{"party_id":"prt_01","sponsor_private_key":"sk_live_123"}`},
		{"internal authority escalation leak", `{"party_id":"prt_01","internal_authority":"COMPANY_ADMIN"}`},
	}

	for _, tc := range leakCases {
		t.Run(tc.name, func(t *testing.T) {
			err := AssertRedactedPartyContract([]byte(tc.payload))
			if !errors.Is(err, ErrRedactionViolation) {
				t.Errorf("expected ErrRedactionViolation for %s, got %v", tc.name, err)
			}
		})
	}
}
