package main

import (
	"testing"
)

func TestLocalDispatcher_Lifecycle(t *testing.T) {
	s := newStore(t)
	id := "M-DISP-1"
	if err := s.Create(testMission(id)); err != nil {
		t.Fatal(err)
	}

	d := &LocalDispatcher{Store: s}

	// 1. Dispatch
	if err := d.Dispatch(id); err != nil {
		t.Fatal(err)
	}

	// 2. Duplicate replay
	if err := d.Dispatch(id); err != nil {
		t.Fatal(err)
	}

	// 3. Monitor before complete
	if _, err := d.Monitor(id); err == nil || err.Error() != "pending" {
		t.Fatalf("expected pending, got %v", err)
	}

	// 4. Complete and validate
	if err := d.MockComplete(id); err != nil {
		t.Fatal(err)
	}
	res, err := d.Monitor(id)
	if err != nil {
		t.Fatal(err)
	}
	if res.Disposition != "PASS" {
		t.Fatalf("expected PASS, got %v", res.Disposition)
	}

	// 5. Restart
	if err := d.Restart(id); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Monitor(id); err == nil || err.Error() != "pending" {
		t.Fatal("expected pending after restart")
	}

	// 6. Timeout
	if err := d.Timeout(id); err != nil {
		t.Fatal(err)
	}
	res, _ = d.Monitor(id)
	if res.Disposition != "TIMEOUT" {
		t.Fatalf("expected TIMEOUT, got %v", res.Disposition)
	}

	// 7. Cancel
	d.Restart(id)
	if err := d.Cancel(id); err != nil {
		t.Fatal(err)
	}
	res, _ = d.Monitor(id)
	if res.Disposition != "CANCELLED" {
		t.Fatalf("expected CANCELLED, got %v", res.Disposition)
	}
}

func TestLocalDispatcher_CanonicalValidation(t *testing.T) {
	s := newStore(t)
	id := "M-DISP-2"
	s.Create(testMission(id))
	d := &LocalDispatcher{Store: s}
	d.Dispatch(id)

	// Write non-canonical
	atomicWrite(d.resultFile(id), []byte("{\n  \"mission_id\": \"M-DISP-2\",\n  \"disposition\": \"PASS\"\n}"))
	if _, err := d.Monitor(id); err == nil || err.Error() != "result is not canonical" {
		t.Fatalf("expected canonical error, got %v", err)
	}

	// Write wrong ID
	atomicWrite(d.resultFile(id), []byte(`{"mission_id":"OTHER","disposition":"PASS"}`))
	if _, err := d.Monitor(id); err == nil || err.Error() != "mission id mismatch" {
		t.Fatalf("expected mismatch error, got %v", err)
	}

	// Write unknown field
	atomicWrite(d.resultFile(id), []byte(`{"mission_id":"M-DISP-2","disposition":"PASS","extra":"x"}`))
	if _, err := d.Monitor(id); err == nil {
		t.Fatal("expected invalid result error due to unknown field")
	}
}
