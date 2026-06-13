// Package server — OpenAI Responses API handler (/v1/responses)
//
// Thin HTTP handler that routes to the provider's StreamResponses method.
// Protocol translation is handled by the transport layer.
package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

// handleResponsesAPI serves the OpenAI Responses API endpoint for Codex CLI.
// It reads the raw request, resolves the model, delegates to the provider's
// StreamResponses method (which returns pre-formatted SSE bytes), and streams
// them directly to the HTTP response.
func (s *Server) handleResponsesAPI(w http.ResponseWriter, r *http.Request) {
	// Read raw body — we need it untouched for the provider to parse
	body, err := io.ReadAll(r.Body)
	r.Body.Close()
	if err != nil {
		writeResponsesError(w, http.StatusBadRequest, "invalid_request", "Failed to read request body")
		return
	}

	// Parse minimally to extract model for routing
	var modelOnly struct {
		Model string `json:"model"`
	}
	json.Unmarshal(body, &modelOnly)
	if modelOnly.Model == "" {
		modelOnly.Model = s.config.Model
	}

	// Resolve model to provider
	resolved := s.modelRoute.Resolve(modelOnly.Model)

	log.Printf("[responses] model=%s provider=%s provider_model=%s", modelOnly.Model, resolved.ProviderID, resolved.ProviderModel)

	// Get provider
	providerCfg := s.providerCfg
	if envKey := providerEnvKey[resolved.ProviderID]; envKey != "" {
		if key := getEnvOrDotenv(envKey); key != "" {
			providerCfg.APIKey = key
		}
	}
	provider, err := s.Registry.Get(resolved.ProviderID, providerCfg, s.config)
	if err != nil {
		writeResponsesError(w, http.StatusBadGateway, "provider_error",
			fmt.Sprintf("Provider %s not available: %v", resolved.ProviderID, err))
		return
	}

	// Call provider's Responses method — it handles parsing + translation
	events, err := provider.StreamResponses(r.Context(), body, resolved.ProviderModel)
	if err != nil {
		writeResponsesError(w, http.StatusBadGateway, "provider_error",
			fmt.Sprintf("Provider error: %v", err))
		return
	}

	// Stream the pre-formatted SSE events directly
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, _ := w.(http.Flusher)
	sentAny := false

	for ev := range events {
		if len(ev) > 0 {
			w.Write(ev)
			sentAny = true
			if flusher != nil {
				flusher.Flush()
			}
		}
	}

	// Ensure at least a close marker
	if !sentAny {
		fmt.Fprintf(w, "data: [DONE]\n\n")
		if flusher != nil {
			flusher.Flush()
		}
	}
}

// writeResponsesError writes an error response in Responses API format.
func writeResponsesError(w http.ResponseWriter, status int, errType, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]any{
			"type":    errType,
			"message": msg,
		},
	})
}
