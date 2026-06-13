package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ChxisB/spectre-proxy/agent/internal/config"
)

func TestProviderRoutes(t *testing.T) {
	cfg := config.DefaultSettings()
	s, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Test /v1/providers
	req := httptest.NewRequest("GET", "/v1/providers", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("/v1/providers returned %d, expected %d", w.Code, http.StatusOK)
	}

	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	t.Logf("Response: %+v", resp)

	if resp["providers"] == nil {
		t.Error("Expected 'providers' field in response")
	}

	// Test /health
	req = httptest.NewRequest("GET", "/health", nil)
	w = httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("/health returned %d, expected %d", w.Code, http.StatusOK)
	}
}

func TestHealthRoute(t *testing.T) {
	cfg := config.DefaultSettings()
	s, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("/health returned %d, expected %d", w.Code, http.StatusOK)
	}
}
