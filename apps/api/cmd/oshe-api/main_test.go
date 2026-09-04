package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthHandler_Methods(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		target         string
		expectedStatus int
		expectBody     bool
	}{
		{
			name:           "GET /health returns 200 OK with body",
			method:         http.MethodGet,
			target:         "/health",
			expectedStatus: http.StatusOK,
			expectBody:     true,
		},
		{
			name:           "HEAD /health returns 200 OK without body",
			method:         http.MethodHead,
			target:         "/health",
			expectedStatus: http.StatusOK,
			expectBody:     false,
		},
		{
			name:           "GET /healthz returns 200 OK with body",
			method:         http.MethodGet,
			target:         "/healthz",
			expectedStatus: http.StatusOK,
			expectBody:     true,
		},
		{
			name:           "HEAD /healthz returns 200 OK without body",
			method:         http.MethodHead,
			target:         "/healthz",
			expectedStatus: http.StatusOK,
			expectBody:     false,
		},
		{
			name:           "POST /health returns 405 Method Not Allowed",
			method:         http.MethodPost,
			target:         "/health",
			expectedStatus: http.StatusMethodNotAllowed,
			expectBody:     true,
		},
		{
			name:           "PUT /health returns 405 Method Not Allowed",
			method:         http.MethodPut,
			target:         "/health",
			expectedStatus: http.StatusMethodNotAllowed,
			expectBody:     true,
		},
		{
			name:           "DELETE /health returns 405 Method Not Allowed",
			method:         http.MethodDelete,
			target:         "/health",
			expectedStatus: http.StatusMethodNotAllowed,
			expectBody:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.target, nil)
			rec := httptest.NewRecorder()

			healthHandler(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			contentType := rec.Header().Get("Content-Type")
			if tt.method == http.MethodGet || tt.method == http.MethodHead {
				if contentType != "application/json" {
					t.Errorf("expected Content-Type application/json, got %q", contentType)
				}
			}

			body := rec.Body.Bytes()
			if tt.expectBody && (tt.method == http.MethodGet) {
				if len(body) == 0 {
					t.Errorf("expected non-empty body for GET")
				}
				var payload map[string]any
				if err := json.Unmarshal(body, &payload); err != nil {
					t.Fatalf("failed to parse JSON response: %v", err)
				}
				if payload["status"] != "ok" {
					t.Errorf("expected status 'ok', got %v", payload["status"])
				}
				if payload["service"] != "oshe-api" {
					t.Errorf("expected service 'oshe-api', got %v", payload["service"])
				}
			} else if !tt.expectBody && (tt.method == http.MethodHead) {
				if len(body) != 0 {
					t.Errorf("expected empty body for HEAD, got %d bytes: %q", len(body), string(body))
				}
			}
		})
	}
}
