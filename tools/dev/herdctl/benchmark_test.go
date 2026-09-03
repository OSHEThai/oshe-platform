package main

import (
	"testing"
)

func TestBenchmarkController_Lifecycle(t *testing.T) {
	s := newStore(t)
	id := "M-BENCH-1"
	if err := s.Create(testMission(id)); err != nil {
		t.Fatal(err)
	}

	c := &BenchmarkController{Store: s}

	// 1. Initial State
	sc, err := c.Scorecard(id)
	if err != nil {
		t.Fatal(err)
	}
	if sc.Status != "PENDING" {
		t.Fatalf("expected PENDING, got %s", sc.Status)
	}

	// 2. Set Fixture
	if err := c.SetFixture(id, "FIXTURE-1"); err != nil {
		t.Fatal(err)
	}

	// 3. Set Failure Injection
	if err := c.SetFailureInjection(id, "FAIL-NONE"); err != nil {
		t.Fatal(err)
	}

	// 4. Record Measure
	if err := c.RecordMeasure(id, "latency", "100ms"); err != nil {
		t.Fatal(err)
	}

	// 5. Complete
	if err := c.Complete(id); err != nil {
		t.Fatal(err)
	}

	// 6. Record Measure after complete (should fail)
	if err := c.RecordMeasure(id, "extra", "1"); err == nil {
		t.Fatal("expected failure recording measure after completion")
	}

	// 7. Verify
	sc, err = c.Scorecard(id)
	if err != nil {
		t.Fatal(err)
	}
	if sc.Status != "COMPLETED" || sc.Fixture != "FIXTURE-1" || sc.FailureInjection != "FAIL-NONE" || sc.Measures["latency"] != "100ms" {
		t.Fatalf("unexpected state: %+v", sc)
	}
}

func TestBenchmarkController_Fail(t *testing.T) {
	s := newStore(t)
	id := "M-BENCH-2"
	if err := s.Create(testMission(id)); err != nil {
		t.Fatal(err)
	}

	c := &BenchmarkController{Store: s}

	if err := c.Fail(id); err != nil {
		t.Fatal(err)
	}

	sc, err := c.Scorecard(id)
	if err != nil {
		t.Fatal(err)
	}
	if sc.Status != "FAILED" {
		t.Fatalf("expected FAILED, got %s", sc.Status)
	}
}
