package main

import (
	"errors"
)

// ProviderAdapter defines the local/stub contract for cross-CLI adapters.
type ProviderAdapter interface {
	Name() string
	Version() string
	Dispatch(payload string) (string, error)
}

// LocalStubAdapter is a candidate stub.
type LocalStubAdapter struct{}

func (a *LocalStubAdapter) Name() string {
	return "local-stub"
}

func (a *LocalStubAdapter) Version() string {
	return "1.0.0"
}

func (a *LocalStubAdapter) Dispatch(payload string) (string, error) {
	if payload == "fail" {
		return "", errors.New("stub injected failure")
	}
	return "stub-success", nil
}

// OfflineMockAdapter is another candidate stub.
type OfflineMockAdapter struct{}

func (a *OfflineMockAdapter) Name() string {
	return "offline-mock"
}

func (a *OfflineMockAdapter) Version() string {
	return "v2"
}

func (a *OfflineMockAdapter) Dispatch(payload string) (string, error) {
	if payload == "error" {
		return "", errors.New("mock injected error")
	}
	return "mock-success", nil
}
