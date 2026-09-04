package identifiers_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/oshethai/oshe-platform/packages/identifiers"
)

// TestQualification_IdentifierCollisionAndUniqueness verifies that generating a high volume
// of identifiers concurrently across multiple standard prefixes yields zero collisions,
// correct prefix separation, and strict length invariants.
func TestQualification_IdentifierCollisionAndUniqueness(t *testing.T) {
	const workers = 8
	const idsPerWorker = 1000
	prefixes := []string{"ten", "org", "usr", "ins", "fnd", "act", "evd", "corr"}

	var mu sync.Mutex
	generated := make(map[string]bool)
	var wg sync.WaitGroup

	errCh := make(chan error, workers)

	for w := 0; w < workers; w++ {
		wg.Add(1)
		prefix := prefixes[w%len(prefixes)]
		go func(p string) {
			defer wg.Done()
			localIDs := make([]string, 0, idsPerWorker)
			for i := 0; i < idsPerWorker; i++ {
				id, err := identifiers.Generate(p)
				if err != nil {
					errCh <- fmt.Errorf("worker failed to generate ID for prefix %q: %w", p, err)
					return
				}
				s := id.String()
				if !strings.HasPrefix(s, p+"_") {
					errCh <- fmt.Errorf("ID %q lacks expected prefix %q_", s, p)
					return
				}
				if len(id.Suffix()) != 32 {
					errCh <- fmt.Errorf("ID %q suffix length %d != 32", s, len(id.Suffix()))
					return
				}
				localIDs = append(localIDs, s)
			}

			mu.Lock()
			for _, id := range localIDs {
				if generated[id] {
					mu.Unlock()
					errCh <- fmt.Errorf("CRITICAL: identifier collision detected for ID %q", id)
					return
				}
				generated[id] = true
			}
			mu.Unlock()
		}(prefix)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Fatal(err)
	}

	expectedTotal := workers * idsPerWorker
	if len(generated) != expectedTotal {
		t.Fatalf("expected %d unique identifiers, got %d", expectedTotal, len(generated))
	}
}

// TestQualification_IdentifierEnumerationAndNegativeInputs tests exhaustive negative
// tampering vectors including non-hex characters, uppercase hex, odd payload lengths,
// whitespace injections, and malformed prefixes to ensure fail-closed parse denial.
func TestQualification_IdentifierEnumerationAndNegativeInputs(t *testing.T) {
	cases := []struct {
		name    string
		raw     string
		wantErr error
	}{
		{"empty string", "", identifiers.ErrEmptyID},
		{"whitespace only", "   \t\n", identifiers.ErrEmptyID},
		{"missing underscore separator", "ins0123456789abcdef0123456789abcdef", identifiers.ErrMalformedID},
		{"multiple underscores", "ins_0123_456789abcdef", identifiers.ErrMalformedID},
		{"empty prefix", "_0123456789abcdef0123456789abcdef", identifiers.ErrMalformedID},
		{"single-char prefix", "i_0123456789abcdef0123456789abcdef", identifiers.ErrMalformedID},
		{"17-char prefix", "abcdefghijklmnopq_0123456789abcdef0123456789abcdef", identifiers.ErrMalformedID},
		{"uppercase prefix", "INS_0123456789abcdef0123456789abcdef", identifiers.ErrMalformedID},
		{"prefix with punctuation", "in-s_0123456789abcdef0123456789abcdef", identifiers.ErrMalformedID},
		{"empty suffix payload", "ins_", identifiers.ErrMalformedID},
		{"odd length suffix", "ins_0123456789abcdef0123456789abcde", identifiers.ErrMalformedID},
		{"uppercase hex payload", "ins_0123456789ABCDEF0123456789abcdef", identifiers.ErrMalformedID},
		{"non-hex payload char g", "ins_0123456789abcdef0123456789abcdeg", identifiers.ErrMalformedID},
		{"non-hex punctuation in payload", "ins_0123456789abcdef0123456789abc-ef", identifiers.ErrMalformedID},
		{"spaces in payload", "ins_0123456789abcdef 0123456789abcde", identifiers.ErrMalformedID},
		{"tab in payload", "ins_0123456789abcdef\t0123456789abcde", identifiers.ErrMalformedID},
		{"trailing newline", "ins_0123456789abcdef0123456789abcdef\n", nil}, // TrimSpace trims before split
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			parsed, err := identifiers.Parse(tc.raw)
			if tc.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error %v for input %q, got parsed: %v", tc.wantErr, tc.raw, parsed)
				}
				if !errors.Is(err, tc.wantErr) {
					t.Errorf("expected error %v for input %q, got: %v", tc.wantErr, tc.raw, err)
				}
			} else {
				if err != nil {
					t.Fatalf("expected input %q to parse successfully, got error: %v", tc.raw, err)
				}
			}
		})
	}
}

// TestQualification_PrefixIsolationAndCrossTypeDenial ensures that parsing with
// ParseWithPrefix strictly denies cross-entity prefix misuse.
func TestQualification_PrefixIsolationAndCrossTypeDenial(t *testing.T) {
	validUser := "usr_0123456789abcdef0123456789abcdef"
	validTenant := "ten_0123456789abcdef0123456789abcdef"
	validInspection := "ins_0123456789abcdef0123456789abcdef"
	validFinding := "fnd_0123456789abcdef0123456789abcdef"

	pairs := []struct {
		raw            string
		expectedPrefix string
	}{
		{validUser, "ten"},
		{validTenant, "usr"},
		{validInspection, "fnd"},
		{validFinding, "ins"},
		{validUser, "org"},
		{validTenant, "cmp"},
	}

	for _, p := range pairs {
		t.Run(fmt.Sprintf("%s_as_%s", p.raw[:3], p.expectedPrefix), func(t *testing.T) {
			_, err := identifiers.ParseWithPrefix(p.raw, p.expectedPrefix)
			if err == nil {
				t.Fatalf("expected prefix mismatch error parsing %s with expected prefix %s", p.raw, p.expectedPrefix)
			}
			if !errors.Is(err, identifiers.ErrPrefixMismatch) {
				t.Errorf("expected ErrPrefixMismatch, got: %v", err)
			}
		})
	}
}

// TestQualification_WireSerializationNegativeVectors verifies that DecodeCorrelationEnvelope
// fails closed on duplicate keys, extra/unexpected properties, missing/unsupported versions.
func TestQualification_WireSerializationNegativeVectors(t *testing.T) {
	vectors := []struct {
		name    string
		payload string
		wantErr error
	}{
		{
			name: "duplicate correlation_id key",
			payload: `{"version":"v1","correlation_id":"corr_0123456789abcdef0123456789abcdef",` +
				`"correlation_id":"corr_0123456789abcdef0123456789abcdef","causation_id":"caus_0123456789abcdef0123456789abcdef"}`,
			wantErr: identifiers.ErrDuplicateFields,
		},
		{
			name: "duplicate causation_id key",
			payload: `{"version":"v1","correlation_id":"corr_0123456789abcdef0123456789abcdef",` +
				`"causation_id":"caus_0123456789abcdef0123456789abcdef","causation_id":"caus_0123456789abcdef0123456789abcdef"}`,
			wantErr: identifiers.ErrDuplicateFields,
		},
		{
			name: "duplicate version key",
			payload: `{"version":"v1","version":"v1","correlation_id":"corr_0123456789abcdef0123456789abcdef",` +
				`"causation_id":"caus_0123456789abcdef0123456789abcdef"}`,
			wantErr: identifiers.ErrDuplicateFields,
		},
		{
			name: "extra unknown field injection",
			payload: `{"version":"v1","correlation_id":"corr_0123456789abcdef0123456789abcdef",` +
				`"causation_id":"caus_0123456789abcdef0123456789abcdef","tenant_override":"malicious"}`,
			wantErr: identifiers.ErrExtraFields,
		},
		{
			name: "unsupported version v2",
			payload: `{"version":"v2","correlation_id":"corr_0123456789abcdef0123456789abcdef",` +
				`"causation_id":"caus_0123456789abcdef0123456789abcdef"}`,
			wantErr: identifiers.ErrUnsupportedVersion,
		},
		{
			name: "empty version string",
			payload: `{"version":"","correlation_id":"corr_0123456789abcdef0123456789abcdef",` +
				`"causation_id":"caus_0123456789abcdef0123456789abcdef"}`,
			wantErr: identifiers.ErrEmptyVersion,
		},
		{
			name: "whitespace version string",
			payload: `{"version":"   ","correlation_id":"corr_0123456789abcdef0123456789abcdef",` +
				`"causation_id":"caus_0123456789abcdef0123456789abcdef"}`,
			wantErr: identifiers.ErrEmptyVersion,
		},
		{
			name:    "empty payload bytes",
			payload: ``,
			wantErr: identifiers.ErrMalformedEnvelopeJSON,
		},
		{
			name:    "non-object JSON array",
			payload: `["v1", "corr_0123456789abcdef0123456789abcdef"]`,
			wantErr: identifiers.ErrMalformedEnvelopeJSON,
		},
	}

	for _, v := range vectors {
		t.Run(v.name, func(t *testing.T) {
			_, err := identifiers.DecodeCorrelationEnvelope([]byte(v.payload))
			if err == nil {
				t.Fatalf("expected error %v, got nil", v.wantErr)
			}
			if !errors.Is(err, v.wantErr) {
				t.Errorf("expected error %v, got: %v", v.wantErr, err)
			}
		})
	}
}

// SyntheticMigrationRecord represents an in-memory entity during legacy-to-canonical migration.
type SyntheticMigrationRecord struct {
	ID        string
	TenantID  string
	Title     string
	DigestHex string
}

// TestQualification_IdentifierMigrationAndAtomicRollback simulates a transactional
// batch identifier migration. It proves that clean records migrate deterministically,
// but any malformed record aborts the migration and leaves state untouched (atomic rollback).
func TestQualification_IdentifierMigrationAndAtomicRollback(t *testing.T) {
	// Canonical in-memory state store
	stateStore := make(map[string]SyntheticMigrationRecord)

	// Baseline seed
	seedRecord := SyntheticMigrationRecord{
		ID:        "ins_0123456789abcdef0123456789abcdef",
		TenantID:  "ten_0123456789abcdef0123456789abcdef",
		Title:     "Baseline Inspection",
		DigestHex: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
	}
	stateStore[seedRecord.ID] = seedRecord

	// Migration executor with snapshot / rollback
	migrateBatch := func(legacyInputs []struct {
		rawID    string
		tenantID string
		title    string
	}) error {
		// Take snapshot of existing state
		snapshot := make(map[string]SyntheticMigrationRecord, len(stateStore))
		for k, v := range stateStore {
			snapshot[k] = v
		}

		// Stage candidate migrated records
		staged := make([]SyntheticMigrationRecord, 0, len(legacyInputs))
		for _, in := range legacyInputs {
			// Normalize legacy UUID (strip hyphens) into canonical prefixed format
			cleanHex := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(in.rawID)), "-", "")
			candidateID := fmt.Sprintf("ins_%s", cleanHex)

			parsedID, err := identifiers.ParseWithPrefix(candidateID, "ins")
			if err != nil {
				// Rollback to snapshot immediately
				stateStore = snapshot
				return fmt.Errorf("migration rejected on identifier %q: %w", in.rawID, err)
			}

			// Validate tenant format
			if _, err := identifiers.ParseWithPrefix(in.tenantID, "ten"); err != nil {
				stateStore = snapshot
				return fmt.Errorf("migration rejected on tenant %q: %w", in.tenantID, err)
			}

			// Collision check
			if _, exists := stateStore[parsedID.String()]; exists {
				stateStore = snapshot
				return fmt.Errorf("migration collision on ID %q", parsedID.String())
			}

			payloadBytes := []byte(fmt.Sprintf("%s:%s:%s", parsedID.String(), in.tenantID, in.title))
			digest := sha256.Sum256(payloadBytes)

			staged = append(staged, SyntheticMigrationRecord{
				ID:        parsedID.String(),
				TenantID:  in.tenantID,
				Title:     in.title,
				DigestHex: hex.EncodeToString(digest[:]),
			})
		}

		// Commit staged records
		for _, rec := range staged {
			stateStore[rec.ID] = rec
		}
		return nil
	}

	// 1. Successful migration batch
	validBatch := []struct {
		rawID    string
		tenantID string
		title    string
	}{
		{
			rawID:    "550e8400-e29b-41d4-a716-446655440000",
			tenantID: "ten_0123456789abcdef0123456789abcdef",
			title:    "Scaffold Check 1",
		},
		{
			rawID:    "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
			tenantID: "ten_0123456789abcdef0123456789abcdef",
			title:    "Confined Space Check 1",
		},
	}

	if err := migrateBatch(validBatch); err != nil {
		t.Fatalf("expected valid batch to migrate successfully, got: %v", err)
	}
	if len(stateStore) != 3 {
		t.Fatalf("expected 3 records in state store, got %d", len(stateStore))
	}

	// 2. Failing migration batch (atomic rollback test)
	failingBatch := []struct {
		rawID    string
		tenantID string
		title    string
	}{
		{
			rawID:    "7ca7b810-9dad-11d1-80b4-00c04fd430c9",
			tenantID: "ten_0123456789abcdef0123456789abcdef",
			title:    "Good Record 1",
		},
		{
			rawID:    "MALFORMED_NON_HEX_UUID_VALUE_HERE!!", // triggers ErrMalformedID
			tenantID: "ten_0123456789abcdef0123456789abcdef",
			title:    "Poison Record",
		},
		{
			rawID:    "8ca7b810-9dad-11d1-80b4-00c04fd430c0",
			tenantID: "ten_0123456789abcdef0123456789abcdef",
			title:    "Good Record 2",
		},
	}

	err := migrateBatch(failingBatch)
	if err == nil {
		t.Fatal("expected migration error on failing batch, got nil")
	}

	// Verify rollback preservation: exactly the 3 pre-batch records remain
	if len(stateStore) != 3 {
		t.Fatalf("rollback failure: expected exactly 3 records preserved, got %d", len(stateStore))
	}
	if _, exists := stateStore["ins_7ca7b8109dad11d180b400c04fd430c9"]; exists {
		t.Fatal("rollback failure: partial record 1 leaked into state store")
	}
}

// TestQualification_EntityStateSerializationAndRecovery tests serializing state with
// identifiers and correlation context, calculating SHA-256 digest, simulating memory loss,
// restoring from serialized state, and confirming bit-exact cryptographic recovery.
func TestQualification_EntityStateSerializationAndRecovery(t *testing.T) {
	corrID, _ := identifiers.ParseCorrelationID("corr_0123456789abcdef0123456789abcdef")
	causID, _ := identifiers.ParseCausationID("caus_fedcba9876543210fedcba9876543210")
	corrCtx := &identifiers.CorrelationContext{
		CorrelationID: corrID,
		CausationID:   causID,
	}

	encodedEnv, err := identifiers.EncodeCorrelationContext(corrCtx)
	if err != nil {
		t.Fatalf("failed to encode correlation context: %v", err)
	}

	type EntityState struct {
		EntityID string `json:"entity_id"`
		TenantID string `json:"tenant_id"`
		Status   string `json:"status"`
		Revision int    `json:"revision"`
		Envelope string `json:"envelope"`
	}

	initialState := EntityState{
		EntityID: "ins_550e8400e29b41d4a716446655440000",
		TenantID: "ten_0123456789abcdef0123456789abcdef",
		Status:   "COMPLETED",
		Revision: 3,
		Envelope: string(encodedEnv),
	}

	initialBytes, err := json.Marshal(initialState)
	if err != nil {
		t.Fatalf("failed to marshal initial state: %v", err)
	}
	initialDigest := sha256.Sum256(initialBytes)

	// Simulate total memory corruption
	corruptedBytes := []byte(`{"corrupted":"garbage_memory"}`)
	corruptedDigest := sha256.Sum256(corruptedBytes)
	if initialDigest == corruptedDigest {
		t.Fatal("corrupted digest matches initial digest")
	}

	// Restore from saved serialized snapshot
	var recoveredState EntityState
	if err := json.Unmarshal(initialBytes, &recoveredState); err != nil {
		t.Fatalf("failed to restore entity state: %v", err)
	}

	recoveredEnv, err := identifiers.DecodeCorrelationEnvelope([]byte(recoveredState.Envelope))
	if err != nil {
		t.Fatalf("failed to decode recovered correlation envelope: %v", err)
	}

	if recoveredEnv.CorrelationID.String() != corrID.String() {
		t.Errorf("recovered correlation ID %q != expected %q", recoveredEnv.CorrelationID.String(), corrID.String())
	}
	if recoveredEnv.CausationID.String() != causID.String() {
		t.Errorf("recovered causation ID %q != expected %q", recoveredEnv.CausationID.String(), causID.String())
	}

	recoveredBytes, err := json.Marshal(recoveredState)
	if err != nil {
		t.Fatalf("failed to marshal recovered state: %v", err)
	}
	recoveredDigest := sha256.Sum256(recoveredBytes)

	if initialDigest != recoveredDigest {
		t.Fatalf("cryptographic recovery failure: initial digest %x != recovered digest %x",
			initialDigest, recoveredDigest)
	}
}
