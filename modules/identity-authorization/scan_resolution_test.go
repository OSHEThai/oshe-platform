package localidentity_test

import (
	"testing"
	"time"

	localidentity "github.com/oshethai/oshe-platform/modules/identity-authorization"
)

func setupScanResolver() (*localidentity.ScanResolver, string) {
	matrix := localidentity.NewProvisionalAuthorizationMatrix()
	evaluator := localidentity.NewPolicyEvaluator()
	tenantID := "ten_synthetic_alpha"

	resolver := localidentity.NewScanResolver(matrix, evaluator)

	// Register test objects
	_ = resolver.RegisterObject(localidentity.ScannableObject{
		TenantID:           tenantID,
		ObjectType:         localidentity.ScannableEquipment,
		ObjectID:           "eqp_boiler_b1",
		ProjectID:          "prj_alpha",
		SiteID:             "ste_rayong",
		AreaID:             "ara_boiler_room",
		LifecycleState:     localidentity.ResourceActive,
		RequiredPermission: localidentity.PermInspectionCreate,
		Metadata:           map[string]string{"name": "Boiler B1"},
	})

	_ = resolver.RegisterObject(localidentity.ScannableObject{
		TenantID:           tenantID,
		ObjectType:         localidentity.ScannableSite,
		ObjectID:           "ste_rayong",
		ProjectID:          "prj_alpha",
		SiteID:             "ste_rayong",
		LifecycleState:     localidentity.ResourceActive,
		RequiredPermission: localidentity.PermOrgSiteRead,
	})

	_ = resolver.RegisterObject(localidentity.ScannableObject{
		TenantID:           tenantID,
		ObjectType:         localidentity.ScannableEquipment,
		ObjectID:           "eqp_decommissioned_turbine",
		ProjectID:          "prj_alpha",
		SiteID:             "ste_rayong",
		LifecycleState:     localidentity.ResourceArchived,
		RequiredPermission: localidentity.PermInspectionCreate,
	})

	return resolver, tenantID
}

func TestScanResolution_ParsingSupportedSchemes(t *testing.T) {
	resolver, tenantID := setupScanResolver()

	cases := []struct {
		name       string
		raw        string
		wantType   localidentity.ScannableObjectType
		wantID     string
		wantTenant string
	}{
		{
			name:       "uri_scheme_equipment",
			raw:        "oshe://ten_synthetic_alpha/equipment/eqp_boiler_b1",
			wantType:   localidentity.ScannableEquipment,
			wantID:     "eqp_boiler_b1",
			wantTenant: tenantID,
		},
		{
			name:       "uri_scheme_with_token_and_exp",
			raw:        "oshe://ten_synthetic_alpha/site/ste_rayong?token=tok123&exp=1789000000",
			wantType:   localidentity.ScannableSite,
			wantID:     "ste_rayong",
			wantTenant: tenantID,
		},
		{
			name:       "compact_scheme_area",
			raw:        "oshe:ten_synthetic_alpha:area:ara_boiler_room",
			wantType:   localidentity.ScannableArea,
			wantID:     "ara_boiler_room",
			wantTenant: tenantID,
		},
		{
			name:       "https_scheme_checklist",
			raw:        "https://app.oshe.local/scan?tenant=ten_synthetic_alpha&type=checklist&id=chk_daily_safety",
			wantType:   localidentity.ScannableChecklist,
			wantID:     "chk_daily_safety",
			wantTenant: tenantID,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			payload, err := resolver.ParseScan(tc.raw)
			if err != nil {
				t.Fatalf("unexpected error parsing %s: %v", tc.raw, err)
			}
			if payload.ObjectType != tc.wantType {
				t.Errorf("expected object type %s, got %s", tc.wantType, payload.ObjectType)
			}
			if payload.ObjectID != tc.wantID {
				t.Errorf("expected object ID %s, got %s", tc.wantID, payload.ObjectID)
			}
			if payload.TenantID != tc.wantTenant {
				t.Errorf("expected tenant ID %s, got %s", tc.wantTenant, payload.TenantID)
			}
		})
	}
}

func TestScanResolution_MalformedInput(t *testing.T) {
	resolver, _ := setupScanResolver()

	badInputs := []struct {
		name string
		raw  string
	}{
		{"empty_string", ""},
		{"unsupported_scheme_ftp", "ftp://ten_synthetic_alpha/equipment/eqp_01"},
		{"unsupported_scheme_qr", "qr://equipment/eqp_01"},
		{"path_traversal_relative", "oshe://ten_synthetic_alpha/equipment/../../etc/passwd"},
		{"null_bytes", "oshe://ten_synthetic_alpha\x00/equipment/eqp_01"},
		{"missing_object_id", "oshe://ten_synthetic_alpha/equipment/"},
		{"prefix_mismatch_type", "oshe://ten_synthetic_alpha/equipment/chk_wrong_prefix"},
	}

	for _, tc := range badInputs {
		t.Run(tc.name, func(t *testing.T) {
			_, err := resolver.ParseScan(tc.raw)
			if err == nil {
				t.Errorf("expected parsing error for input %q, got nil", tc.raw)
			}

			// Also test Resolve() fails closed
			res := resolver.Resolve(localidentity.ScanResolutionContext{
				Identity: localidentity.SubjectIdentity{
					Subject:         "usr_inspector_01",
					TenantID:        "ten_synthetic_alpha",
					IsAuthenticated: true,
				},
				CallerRole: localidentity.RoleInspector,
				RawScan:    tc.raw,
			})
			if res.Allowed {
				t.Errorf("expected Resolve() to fail closed for malformed input %q", tc.raw)
			}
			if res.DenialCode != localidentity.DenialScanInvalidInput {
				t.Errorf("expected DenialScanInvalidInput, got %s", res.DenialCode)
			}
		})
	}
}

func TestScanResolution_TemporalExpiration(t *testing.T) {
	resolver, tenantID := setupScanResolver()
	now := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)

	// Expired payload: exp is 1 hour before now (in the past)
	expiredScan := "oshe://" + tenantID + "/equipment/eqp_boiler_b1?exp=1725537600"
	res := resolver.Resolve(localidentity.ScanResolutionContext{
		Identity: localidentity.SubjectIdentity{
			Subject:         "usr_inspector_01",
			TenantID:        tenantID,
			IsAuthenticated: true,
		},
		CallerRole: localidentity.RoleInspector,
		ActiveSite: "ste_rayong",
		RawScan:    expiredScan,
		At:         now,
	})

	if res.Allowed {
		t.Fatalf("expected expired scan to be denied")
	}
	if res.DenialCode != localidentity.DenialScanExpired {
		t.Errorf("expected DenialScanExpired, got %s", res.DenialCode)
	}

	// Unexpired payload
	resValid := resolver.Resolve(localidentity.ScanResolutionContext{
		Identity: localidentity.SubjectIdentity{
			Subject:         "usr_inspector_01",
			TenantID:        tenantID,
			IsAuthenticated: true,
		},
		CallerRole: localidentity.RoleInspector,
		ActiveSite: "ste_rayong",
		RawScan:    "oshe://" + tenantID + "/equipment/eqp_boiler_b1?exp=1893456000", // future timestamp
		At:         now,
	})

	if !resValid.Allowed {
		t.Fatalf("expected unexpired valid scan to be allowed, got error: %s (denial: %s)", resValid.ErrorMessage, resValid.DenialCode)
	}
}

func TestScanResolution_CrossTenantIsolation(t *testing.T) {
	resolver, tenantID := setupScanResolver()

	// Caller belongs to tenant_A, but scan payload targets tenant_B
	res := resolver.Resolve(localidentity.ScanResolutionContext{
		Identity: localidentity.SubjectIdentity{
			Subject:         "usr_inspector_01",
			TenantID:        tenantID,
			IsAuthenticated: true,
		},
		CallerRole: localidentity.RoleInspector,
		RawScan:    "oshe://ten_foreign_beta/equipment/eqp_boiler_b1",
	})

	if res.Allowed {
		t.Fatalf("expected cross-tenant scan resolution to be denied")
	}
	// Anti-enumeration: must return generic DenialScanUnauthorized
	if res.DenialCode != localidentity.DenialScanUnauthorized {
		t.Errorf("expected DenialScanUnauthorized, got %s", res.DenialCode)
	}
}

func TestScanResolution_ScopeConfinement(t *testing.T) {
	resolver, tenantID := setupScanResolver()

	// Caller is active at ste_bangkok, but target eqp_boiler_b1 is at ste_rayong
	res := resolver.Resolve(localidentity.ScanResolutionContext{
		Identity: localidentity.SubjectIdentity{
			Subject:         "usr_inspector_01",
			TenantID:        tenantID,
			IsAuthenticated: true,
		},
		CallerRole: localidentity.RoleInspector,
		ActiveSite: "ste_bangkok",
		RawScan:    "oshe://" + tenantID + "/equipment/eqp_boiler_b1",
	})

	if res.Allowed {
		t.Fatalf("expected out-of-scope scan to be denied")
	}
	if res.DenialCode != localidentity.DenialScanUnauthorized {
		t.Errorf("expected DenialScanUnauthorized, got %s", res.DenialCode)
	}
}

func TestScanResolution_ObjectLifecycleState(t *testing.T) {
	resolver, tenantID := setupScanResolver()

	// Attempting to resolve a decommissioned turbine
	res := resolver.Resolve(localidentity.ScanResolutionContext{
		Identity: localidentity.SubjectIdentity{
			Subject:         "usr_inspector_01",
			TenantID:        tenantID,
			IsAuthenticated: true,
		},
		CallerRole: localidentity.RoleInspector,
		ActiveSite: "ste_rayong",
		RawScan:    "oshe://" + tenantID + "/equipment/eqp_decommissioned_turbine",
	})

	if res.Allowed {
		t.Fatalf("expected decommissioned object scan to be denied")
	}
	if res.DenialCode != localidentity.DenialScanUnauthorized {
		t.Errorf("expected DenialScanUnauthorized, got %s", res.DenialCode)
	}
}

func TestScanResolution_PermissionCheck_ScanIsNotAuthority(t *testing.T) {
	resolver, tenantID := setupScanResolver()

	// RoleViewer does NOT possess PermissionInspectionExecute
	resViewer := resolver.Resolve(localidentity.ScanResolutionContext{
		Identity: localidentity.SubjectIdentity{
			Subject:         "usr_viewer_01",
			TenantID:        tenantID,
			IsAuthenticated: true,
		},
		CallerRole: localidentity.RoleViewer,
		ActiveSite: "ste_rayong",
		RawScan:    "oshe://" + tenantID + "/equipment/eqp_boiler_b1",
	})

	if resViewer.Allowed {
		t.Fatalf("expected scan without required permission to fail: scan is never authority")
	}
	if resViewer.DenialCode != localidentity.DenialScanUnauthorized {
		t.Errorf("expected DenialScanUnauthorized, got %s", resViewer.DenialCode)
	}

	// RoleInspector DOES possess PermissionInspectionExecute
	resInspector := resolver.Resolve(localidentity.ScanResolutionContext{
		Identity: localidentity.SubjectIdentity{
			Subject:         "usr_inspector_01",
			TenantID:        tenantID,
			IsAuthenticated: true,
		},
		CallerRole: localidentity.RoleInspector,
		ActiveSite: "ste_rayong",
		RawScan:    "oshe://" + tenantID + "/equipment/eqp_boiler_b1",
	})

	if !resInspector.Allowed {
		t.Fatalf("expected inspector with valid permission to resolve scan successfully, got error: %s", resInspector.ErrorMessage)
	}
	if resInspector.ResolvedObject == nil || resInspector.ResolvedObject.ObjectID != "eqp_boiler_b1" {
		t.Errorf("expected resolved object eqp_boiler_b1, got %v", resInspector.ResolvedObject)
	}
}

func TestScanResolution_AntiEnumeration_IndistinguishableDenials(t *testing.T) {
	resolver, tenantID := setupScanResolver()

	// 1. Unauthorized real object
	resReal := resolver.Resolve(localidentity.ScanResolutionContext{
		Identity: localidentity.SubjectIdentity{
			Subject:         "usr_viewer_01",
			TenantID:        tenantID,
			IsAuthenticated: true,
		},
		CallerRole: localidentity.RoleViewer,
		ActiveSite: "ste_rayong",
		RawScan:    "oshe://" + tenantID + "/equipment/eqp_boiler_b1", // real ID
	})

	// 2. Guessed, non-existent object
	resGuessed := resolver.Resolve(localidentity.ScanResolutionContext{
		Identity: localidentity.SubjectIdentity{
			Subject:         "usr_viewer_01",
			TenantID:        tenantID,
			IsAuthenticated: true,
		},
		CallerRole: localidentity.RoleViewer,
		ActiveSite: "ste_rayong",
		RawScan:    "oshe://" + tenantID + "/equipment/eqp_guessed_9999", // fake ID
	})

	if resReal.Allowed || resGuessed.Allowed {
		t.Fatalf("both real unauthorized and guessed objects must be denied")
	}

	// Anti-enumeration requirement: DenialCode and ErrorMessage must be IDENTICAL
	if resReal.DenialCode != resGuessed.DenialCode {
		t.Errorf("enumeration leak: denial codes differ! real=%s, guessed=%s", resReal.DenialCode, resGuessed.DenialCode)
	}
	if resReal.ErrorMessage != resGuessed.ErrorMessage {
		t.Errorf("enumeration leak: error messages differ! real=%q, guessed=%q", resReal.ErrorMessage, resGuessed.ErrorMessage)
	}
	if resReal.ResolvedObject != nil || resGuessed.ResolvedObject != nil {
		t.Errorf("security leak: resolved object returned on denied scan")
	}
}

func TestScanResolution_AuditLedger(t *testing.T) {
	resolver, tenantID := setupScanResolver()

	// Perform a scan attempt
	resolver.Resolve(localidentity.ScanResolutionContext{
		Identity: localidentity.SubjectIdentity{
			Subject:         "usr_auditor_01",
			TenantID:        tenantID,
			IsAuthenticated: true,
		},
		CallerRole: localidentity.RoleInspector,
		ActiveSite: "ste_rayong",
		RawScan:    "oshe://" + tenantID + "/equipment/eqp_boiler_b1",
	})

	records := resolver.AuditLedger()
	if len(records) == 0 {
		t.Fatalf("expected audit records to be recorded in ledger")
	}

	last := records[len(records)-1]
	if last.ActorSubject != "usr_auditor_01" {
		t.Errorf("expected actor usr_auditor_01, got %s", last.ActorSubject)
	}
	if last.RawPayloadHash == "" {
		t.Errorf("expected non-empty raw payload hash")
	}
	if !last.Allowed {
		t.Errorf("expected allowed audit record")
	}
}
