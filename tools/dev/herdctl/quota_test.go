package main

import (
	"testing"
)

func TestQuotaController(t *testing.T) {
	q := NewQuotaController(100, 2)

	// Test concurrency
	if err := q.Acquire(10); err != nil {
		t.Fatal(err)
	}
	if err := q.Acquire(10); err != nil {
		t.Fatal(err)
	}
	if err := q.Acquire(10); err != ErrConcurrencyLimit {
		t.Fatalf("expected concurrency limit, got %v", err)
	}

	q.Release()
	if err := q.Acquire(10); err != nil {
		t.Fatal(err)
	}
	q.Release()
	q.Release()

	// Test quota limit
	if err := q.Acquire(80); err != ErrQuotaExceeded {
		t.Fatalf("expected quota exceeded, got %v", err)
	}

	// Test stop behavior
	q.SetStopBehavior(true)
	if err := q.Acquire(1); err == nil || err.Error() != "system stopped" {
		t.Fatalf("expected system stopped, got %v", err)
	}
}
