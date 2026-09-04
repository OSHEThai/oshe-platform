package identifiers_test

import (
	"errors"
	"sync"
	"testing"

	"github.com/oshethai/oshe-platform/packages/identifiers"
)

func TestCheckOrRecord_FirstUse(t *testing.T) {
	ledger := identifiers.NewIdempotencyLedger()
	payload := []byte(`{"action":"create_inspection","tenant_id":"ten_001"}`)

	status, rec, err := ledger.CheckOrRecord("key-101", payload)
	if err != nil {
		t.Fatalf("unexpected error on first use: %v", err)
	}
	if status != identifiers.StatusFirstUse {
		t.Errorf("expected StatusFirstUse, got %q", status)
	}
	if rec == nil || rec.Key != "key-101" {
		t.Fatalf("expected valid record for key-101, got %+v", rec)
	}
	if rec.PayloadHash != identifiers.HashPayload(payload) {
		t.Errorf("expected payload hash %s, got %s", identifiers.HashPayload(payload), rec.PayloadHash)
	}
	if ledger.Count() != 1 {
		t.Errorf("expected ledger count 1, got %d", ledger.Count())
	}
}

func TestCheckOrRecord_Replay(t *testing.T) {
	ledger := identifiers.NewIdempotencyLedger()
	payload := []byte(`{"action":"record_finding","finding_id":"fnd_001"}`)

	// First use
	_, _, err := ledger.CheckOrRecord("idemp-key-202", payload)
	if err != nil {
		t.Fatalf("first registration failed: %v", err)
	}

	// Immediate replay with exact same payload
	status, rec, err := ledger.CheckOrRecord("idemp-key-202", payload)
	if err != nil {
		t.Fatalf("replay should not return error, got: %v", err)
	}
	if status != identifiers.StatusReplay {
		t.Errorf("expected StatusReplay, got %q", status)
	}
	if rec.Key != "idemp-key-202" {
		t.Errorf("expected key 'idemp-key-202', got %q", rec.Key)
	}
	if ledger.Count() != 1 {
		t.Errorf("expected count to remain 1 after replay, got %d", ledger.Count())
	}
}

func TestCheckOrRecord_Conflict(t *testing.T) {
	ledger := identifiers.NewIdempotencyLedger()
	payloadA := []byte(`{"amount":100}`)
	payloadB := []byte(`{"amount":200}`)

	// Initial registration
	_, _, err := ledger.CheckOrRecord("trans-999", payloadA)
	if err != nil {
		t.Fatalf("initial registration failed: %v", err)
	}

	// Conflicting reuse with different payload
	status, _, err := ledger.CheckOrRecord("trans-999", payloadB)
	if err == nil {
		t.Fatal("expected error on conflicting payload, got nil")
	}
	if !errors.Is(err, identifiers.ErrIdempotencyConflict) {
		t.Errorf("expected ErrIdempotencyConflict, got %v", err)
	}
	if status != identifiers.StatusConflict {
		t.Errorf("expected StatusConflict, got %q", status)
	}
}

func TestCheckOrRecord_EmptyKey(t *testing.T) {
	ledger := identifiers.NewIdempotencyLedger()
	for _, k := range []string{"", "   ", "\t\n"} {
		_, _, err := ledger.CheckOrRecord(k, []byte("data"))
		if !errors.Is(err, identifiers.ErrEmptyIdempotencyKey) {
			t.Errorf("expected ErrEmptyIdempotencyKey for key %q, got %v", k, err)
		}
	}
}

func TestCheckOrRecord_ConcurrentDuplicateRegistration(t *testing.T) {
	ledger := identifiers.NewIdempotencyLedger()
	key := "concurrent-race-key"
	payload := []byte(`{"concurrent":true,"data":"immutable"}`)

	const goroutines = 50
	var wg sync.WaitGroup
	firstUseCount := 0
	replayCount := 0
	var mu sync.Mutex

	for range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			status, _, err := ledger.CheckOrRecord(key, payload)
			if err != nil {
				t.Errorf("unexpected concurrent check error: %v", err)
				return
			}
			mu.Lock()
			defer mu.Unlock()
			if status == identifiers.StatusFirstUse {
				firstUseCount++
			} else if status == identifiers.StatusReplay {
				replayCount++
			}
		}()
	}

	wg.Wait()

	if firstUseCount != 1 {
		t.Errorf("expected exactly 1 StatusFirstUse among concurrent calls, got %d", firstUseCount)
	}
	if replayCount != goroutines-1 {
		t.Errorf("expected %d StatusReplay calls, got %d", goroutines-1, replayCount)
	}
	if ledger.Count() != 1 {
		t.Errorf("expected ledger count 1, got %d", ledger.Count())
	}
}
