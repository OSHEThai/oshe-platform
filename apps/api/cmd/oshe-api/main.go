package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/oshethai/oshe-platform/apps/api"
	orgtenancy "github.com/oshethai/oshe-platform/modules/organization-tenancy"
)

func createClaimsResolver(authMode string) api.ClaimsResolver {
	isSynthetic := strings.ToLower(strings.TrimSpace(authMode)) == "synthetic"
	return func(r *http.Request) (*orgtenancy.TrustedClaims, error) {
		if !isSynthetic {
			// Fail-closed default: non-synthetic mode has no production identity provider wired
			return nil, errors.New("identity provider unconfigured: non-synthetic mode fails closed (default-deny)")
		}

		authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
		if authHeader == "" {
			return nil, errors.New("missing bearer authorization header")
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
			return nil, errors.New("invalid authorization scheme: expected Bearer <token>")
		}

		token := strings.TrimSpace(parts[1])

		// Synthetic token format: "synthetic:<tenant_id>:<subject_id>"
		if strings.HasPrefix(token, "synthetic:") {
			tokenParts := strings.Split(token, ":")
			if len(tokenParts) >= 3 && tokenParts[1] != "" && tokenParts[2] != "" {
				return &orgtenancy.TrustedClaims{
					Subject:         tokenParts[2],
					TenantID:        tokenParts[1],
					IsAuthenticated: true,
				}, nil
			}
		}

		// Built-in synthetic test tokens
		switch token {
		case "token_inspector_alpha":
			return &orgtenancy.TrustedClaims{
				Subject:         "sub_inspector_somchai",
				TenantID:        "ten_safety_corp",
				IsAuthenticated: true,
			}, nil
		case "token_officer_alpha":
			return &orgtenancy.TrustedClaims{
				Subject:         "sub_officer_alice",
				TenantID:        "ten_safety_corp",
				IsAuthenticated: true,
			}, nil
		case "token_lead_alpha":
			return &orgtenancy.TrustedClaims{
				Subject:         "sub_lead_bob",
				TenantID:        "ten_safety_corp",
				IsAuthenticated: true,
			}, nil
		}

		return nil, errors.New("unrecognized synthetic bearer token")
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	authMode := os.Getenv("OSHE_AUTH_MODE")
	if authMode == "" {
		authMode = "fail_closed"
	}

	resp := map[string]any{
		"status":      "ok",
		"service":     "oshe-api",
		"auth_mode":   authMode,
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
		"environment": "local-synthetic",
		"disclaimer":  "LOCAL_SYNTHETIC_ONLY: Zero production connectivity, customer data, or live credentials.",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	authMode := os.Getenv("OSHE_AUTH_MODE")
	resolver := createClaimsResolver(authMode)

	store := api.NewWalkingSkeletonStore(nil)
	wsServer := api.NewWalkingSkeletonServer(store, resolver)
	wsHandler := wsServer.Handler()

	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/healthz", healthHandler)
	mux.Handle("/api/", wsHandler)

	addr := fmt.Sprintf("0.0.0.0:%s", port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Server shutdown channel
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("Starting OSHE Local API Container service on %s (auth_mode=%s)", addr, authMode)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server listen error: %v", err)
		}
	}()

	<-stop
	log.Println("Shutting down OSHE Local API service...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
	log.Println("Service stopped cleanly.")
}
