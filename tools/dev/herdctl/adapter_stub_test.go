package main

import (
	"testing"
)

func TestLocalStubAdapter(t *testing.T) {
	a := &LocalStubAdapter{}
	if a.Name() != "local-stub" {
		t.Fatal("wrong name")
	}
	if a.Version() != "1.0.0" {
		t.Fatal("wrong version")
	}
	if _, err := a.Dispatch("fail"); err == nil {
		t.Fatal("expected failure")
	}
	if res, err := a.Dispatch("ok"); err != nil || res != "stub-success" {
		t.Fatal("expected success")
	}
}

func TestOfflineMockAdapter(t *testing.T) {
	a := &OfflineMockAdapter{}
	if a.Name() != "offline-mock" {
		t.Fatal("wrong name")
	}
	if a.Version() != "v2" {
		t.Fatal("wrong version")
	}
	if _, err := a.Dispatch("error"); err == nil {
		t.Fatal("expected failure")
	}
	if res, err := a.Dispatch("ok"); err != nil || res != "mock-success" {
		t.Fatal("expected success")
	}
}
