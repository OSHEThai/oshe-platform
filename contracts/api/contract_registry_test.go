package api_test

import (
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/oshethai/oshe-platform/contracts/api"
)

func validTestDescriptor(id string) api.ContractDescriptor {
	return api.ContractDescriptor{
		ContractID:       id,
		Name:             "Inspections API Contract",
		Version:          api.CurrentContractVersion,
		Description:      "Standard inspection workflow API contract specification",
		PathPattern:      "/api/v1/inspections",
		Protocol:         api.DefaultAPIProtocol,
		ErrorEnvelopeRef: api.CanonicalErrorEnvelopeRef,
		SupportedMethods: []string{"GET", "POST"},
		RequiredHeaders:  []string{"X-Correlation-ID"},
		ProvisionalState: api.ProvisionalStatus,
		Metadata: map[string]string{
			"owner": "safety-engineering",
		},
	}
}

func TestContractRegistry_SuccessfulRegistrationAndRetrieval(t *testing.T) {
	registry := api.NewContractRegistry()

	desc := validTestDescriptor("CTR-API-INSPECT-001")
	if err := registry.RegisterContract(desc); err != nil {
		t.Fatalf("expected successful registration, got: %v", err)
	}

	if !registry.HasContract("CTR-API-INSPECT-001") {
		t.Fatalf("expected registry to have contract 'CTR-API-INSPECT-001'")
	}

	retrieved, err := registry.GetContract("CTR-API-INSPECT-001")
	if err != nil {
		t.Fatalf("expected successful retrieval, got: %v", err)
	}

	if retrieved.ContractID != "CTR-API-INSPECT-001" {
		t.Errorf("expected ContractID 'CTR-API-INSPECT-001', got %q", retrieved.ContractID)
	}
	if retrieved.Version != api.CurrentContractVersion {
		t.Errorf("expected Version %q, got %q", api.CurrentContractVersion, retrieved.Version)
	}
	if retrieved.ErrorEnvelopeRef != api.CanonicalErrorEnvelopeRef {
		t.Errorf("expected ErrorEnvelopeRef %q, got %q", api.CanonicalErrorEnvelopeRef, retrieved.ErrorEnvelopeRef)
	}
	if retrieved.ProvisionalState != api.ProvisionalStatus {
		t.Errorf("expected ProvisionalState %q, got %q", api.ProvisionalStatus, retrieved.ProvisionalState)
	}
	if registry.Count() != 1 {
		t.Errorf("expected count 1, got %d", registry.Count())
	}
}

func TestContractRegistry_IdempotentReRegistration(t *testing.T) {
	registry := api.NewContractRegistry()

	desc := validTestDescriptor("CTR-API-IDEM-001")
	if err := registry.RegisterContract(desc); err != nil {
		t.Fatalf("initial registration failed: %v", err)
	}

	// Identical re-registration must succeed cleanly
	if err := registry.RegisterContract(desc); err != nil {
		t.Fatalf("idempotent re-registration failed: %v", err)
	}

	if registry.Count() != 1 {
		t.Errorf("expected count to remain 1 after identical re-registration, got %d", registry.Count())
	}
}

func TestContractRegistry_ConflictDetection(t *testing.T) {
	registry := api.NewContractRegistry()

	desc1 := validTestDescriptor("CTR-API-CONF-001")
	if err := registry.RegisterContract(desc1); err != nil {
		t.Fatalf("initial registration failed: %v", err)
	}

	// Candidate with same ContractID but different Name
	descDiffName := validTestDescriptor("CTR-API-CONF-001")
	descDiffName.Name = "Differing Name Specification"

	err := registry.RegisterContract(descDiffName)
	if err == nil {
		t.Fatalf("expected error on conflicting registration, got nil")
	}
	if !errors.Is(err, api.ErrDuplicateContractID) {
		t.Errorf("expected ErrDuplicateContractID, got: %v", err)
	}

	// Candidate with same ContractID but different PathPattern
	descDiffPath := validTestDescriptor("CTR-API-CONF-001")
	descDiffPath.PathPattern = "/api/v1/different-path"

	err = registry.RegisterContract(descDiffPath)
	if err == nil || !errors.Is(err, api.ErrDuplicateContractID) {
		t.Errorf("expected ErrDuplicateContractID for path conflict, got: %v", err)
	}

	// Candidate with same ContractID but different Methods
	descDiffMethods := validTestDescriptor("CTR-API-CONF-001")
	descDiffMethods.SupportedMethods = []string{"GET", "PUT"}

	err = registry.RegisterContract(descDiffMethods)
	if err == nil || !errors.Is(err, api.ErrDuplicateContractID) {
		t.Errorf("expected ErrDuplicateContractID for method conflict, got: %v", err)
	}
}

func TestContractRegistry_VersionEnforcement(t *testing.T) {
	registry := api.NewContractRegistry()

	// Unsupported version v2
	descV2 := validTestDescriptor("CTR-API-VER-002")
	descV2.Version = "v2"
	err := registry.RegisterContract(descV2)
	if err == nil || !errors.Is(err, api.ErrUnsupportedVersion) {
		t.Errorf("expected ErrUnsupportedVersion for v2, got: %v", err)
	}

	// Blank version
	descBlankVer := validTestDescriptor("CTR-API-VER-BLANK")
	descBlankVer.Version = "   "
	err = registry.RegisterContract(descBlankVer)
	if err == nil || !errors.Is(err, api.ErrUnsupportedVersion) {
		t.Errorf("expected ErrUnsupportedVersion for blank version, got: %v", err)
	}
}

func TestContractRegistry_ValidationFailures(t *testing.T) {
	tests := []struct {
		name        string
		modify      func(d *api.ContractDescriptor)
		expectedErr error
	}{
		{
			name: "blank contract ID",
			modify: func(d *api.ContractDescriptor) {
				d.ContractID = "   "
			},
			expectedErr: api.ErrBlankContractID,
		},
		{
			name: "blank contract name",
			modify: func(d *api.ContractDescriptor) {
				d.Name = ""
			},
			expectedErr: api.ErrBlankContractName,
		},
		{
			name: "blank path pattern",
			modify: func(d *api.ContractDescriptor) {
				d.PathPattern = "  "
			},
			expectedErr: api.ErrBlankPathPattern,
		},
		{
			name: "blank error envelope ref",
			modify: func(d *api.ContractDescriptor) {
				d.ErrorEnvelopeRef = ""
			},
			expectedErr: api.ErrBlankErrorEnvelopeRef,
		},
		{
			name: "no supported methods",
			modify: func(d *api.ContractDescriptor) {
				d.SupportedMethods = []string{}
			},
			expectedErr: api.ErrNoSupportedMethods,
		},
		{
			name: "invalid HTTP method",
			modify: func(d *api.ContractDescriptor) {
				d.SupportedMethods = []string{"GET", "INVALID_METHOD"}
			},
			expectedErr: api.ErrInvalidMethod,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg := api.NewContractRegistry()
			desc := validTestDescriptor("CTR-API-VAL-TEST")
			tt.modify(&desc)

			err := reg.RegisterContract(desc)
			if err == nil {
				t.Fatalf("expected error %v, got nil", tt.expectedErr)
			}
			if !errors.Is(err, tt.expectedErr) {
				t.Errorf("expected %v, got: %v", tt.expectedErr, err)
			}
		})
	}
}

func TestContractRegistry_RetrievalEdgeCases(t *testing.T) {
	registry := api.NewContractRegistry()

	// Blank ID retrieval
	_, err := registry.GetContract("   ")
	if !errors.Is(err, api.ErrBlankContractID) {
		t.Errorf("expected ErrBlankContractID, got: %v", err)
	}

	// Missing ID retrieval
	_, err = registry.GetContract("NON-EXISTENT-CTR")
	if !errors.Is(err, api.ErrContractNotFound) {
		t.Errorf("expected ErrContractNotFound, got: %v", err)
	}

	// HasContract with blank ID
	if registry.HasContract(" ") {
		t.Errorf("HasContract should return false for blank input")
	}

	// HasContract with missing ID
	if registry.HasContract("MISSING") {
		t.Errorf("HasContract should return false for missing input")
	}
}

func TestContractRegistry_DeterministicListOrdering(t *testing.T) {
	registry := api.NewContractRegistry()

	ids := []string{
		"CTR-API-003-REPORTS",
		"CTR-API-001-TEMPLATES",
		"CTR-API-004-EXPORTS",
		"CTR-API-002-INSTANCES",
	}

	for _, id := range ids {
		desc := validTestDescriptor(id)
		if err := registry.RegisterContract(desc); err != nil {
			t.Fatalf("failed to register %s: %v", id, err)
		}
	}

	if registry.Count() != 4 {
		t.Fatalf("expected count 4, got %d", registry.Count())
	}

	list := registry.ListContracts()
	if len(list) != 4 {
		t.Fatalf("expected list length 4, got %d", len(list))
	}

	expectedOrder := []string{
		"CTR-API-001-TEMPLATES",
		"CTR-API-002-INSTANCES",
		"CTR-API-003-REPORTS",
		"CTR-API-004-EXPORTS",
	}

	for i, expectedID := range expectedOrder {
		if list[i].ContractID != expectedID {
			t.Errorf("list[%d]: expected %q, got %q", i, expectedID, list[i].ContractID)
		}
	}
}

func TestContractRegistry_ConcurrentOperations(t *testing.T) {
	registry := api.NewContractRegistry()
	numGoroutines := 20
	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			id := fmt.Sprintf("CTR-CONC-%03d", idx)
			desc := validTestDescriptor(id)
			_ = registry.RegisterContract(desc)

			// Concurrently read and check
			_ = registry.HasContract(id)
			_, _ = registry.GetContract(id)
			_ = registry.ListContracts()
			_ = registry.Count()
		}(i)
	}

	wg.Wait()

	if registry.Count() != numGoroutines {
		t.Errorf("expected count %d, got %d", numGoroutines, registry.Count())
	}
}
