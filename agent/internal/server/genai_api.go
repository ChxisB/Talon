// Package server — Google GenAI API handlers for Gemini CLI
//
// Thin HTTP handlers that route to the provider's StreamGenAI method.
// Protocol translation is handled by the transport layer.
package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

// extractGenAIModel extracts the model name from a URL path like:
// /v1beta/models/gemini-2.5-pro:generateContent
func extractGenAIModel(path string) string {
	if idx := strings.Index(path, "/models/"); idx >= 0 {
		rest := path[idx+len("/models/"):]
		if colon := strings.IndexByte(rest, ':'); colon >= 0 {
			return rest[:colon]
		}
		return rest
	}
	return ""
}

// handleGenAIStream serves the streaming GenAI endpoint (:streamGenerateContent).
func (s *Server) handleGenAIStream(w http.ResponseWriter, r *http.Request) {
	model := extractGenAIModel(r.URL.Path)
	if model == "" {
		writeGenAIError(w, http.StatusBadRequest, "Could not extract model from path")
		return
	}

	body, err := io.ReadAll(r.Body)
	r.Body.Close()
	if err != nil {
		writeGenAIError(w, http.StatusBadRequest, "Failed to read request body")
		return
	}

	// Resolve model
	resolved := s.modelRoute.Resolve(model)
	log.Printf("[genai] streaming model=%s provider=%s provider_model=%s", model, resolved.ProviderID, resolved.ProviderModel)

	// Get provider
	providerCfg := s.providerCfg
	if envKey := providerEnvKey[resolved.ProviderID]; envKey != "" {
		if key := getEnvOrDotenv(envKey); key != "" {
			providerCfg.APIKey = key
		}
	}
	provider, err := s.Registry.Get(resolved.ProviderID, providerCfg, s.config)
	if err != nil {
		writeGenAIError(w, http.StatusBadGateway, fmt.Sprintf("Provider not available: %v", err))
		return
	}

	// Call provider's GenAI method — it handles parsing + translation
	events, err := provider.StreamGenAI(r.Context(), body, resolved.ProviderModel)
	if err != nil {
		writeGenAIError(w, http.StatusBadGateway, fmt.Sprintf("Provider error: %v", err))
		return
	}

	// Stream the pre-formatted SSE chunks directly
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

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

	if !sentAny {
		fmt.Fprintf(w, "data: [DONE]\n\n")
		if flusher != nil {
			flusher.Flush()
		}
	}
}

// handleGenAI serves the non-streaming GenAI endpoint (:generateContent).
func (s *Server) handleGenAI(w http.ResponseWriter, r *http.Request) {
	model := extractGenAIModel(r.URL.Path)
	if model == "" {
		writeGenAIError(w, http.StatusBadRequest, "Could not extract model from path")
		return
	}

	body, err := io.ReadAll(r.Body)
	r.Body.Close()
	if err != nil {
		writeGenAIError(w, http.StatusBadRequest, "Failed to read request body")
		return
	}

	resolved := s.modelRoute.Resolve(model)
	log.Printf("[genai] non-streaming model=%s provider=%s provider_model=%s", model, resolved.ProviderID, resolved.ProviderModel)

	providerCfg := s.providerCfg
	if envKey := providerEnvKey[resolved.ProviderID]; envKey != "" {
		if key := getEnvOrDotenv(envKey); key != "" {
			providerCfg.APIKey = key
		}
	}
	provider, err := s.Registry.Get(resolved.ProviderID, providerCfg, s.config)
	if err != nil {
		writeGenAIError(w, http.StatusBadGateway, fmt.Sprintf("Provider not available: %v", err))
		return
	}

	// For non-streaming, call StreamGenAI and collect all events into a response
	events, err := provider.StreamGenAI(r.Context(), body, resolved.ProviderModel)
	if err != nil {
		writeGenAIError(w, http.StatusBadGateway, fmt.Sprintf("Provider error: %v", err))
		return
	}

	// Collect all SSE chunks into a single GenAI response object
	var textParts []string
	for ev := range events {
		if len(ev) > 0 {
			// Parse the SSE data: "data: {...}\n\n"
			line := string(ev)
			if strings.HasPrefix(line, "data: ") {
				data := strings.TrimPrefix(line, "data: ")
				data = strings.TrimRight(data, "\n")
				if data == "[DONE]" {
					break
				}
				// Extract text from GenAI candidate chunks
				var chunk struct {
					Candidates []struct {
						Content struct {
							Parts []struct {
								Text string `json:"text"`
							} `json:"parts"`
						} `json:"content"`
					} `json:"candidates"`
				}
				if err := json.Unmarshal([]byte(data), &chunk); err == nil {
					for _, c := range chunk.Candidates {
						for _, p := range c.Content.Parts {
							if p.Text != "" {
								textParts = append(textParts, p.Text)
							}
						}
					}
				}
			}
		}
	}

	// Build final GenAI response
	fullText := strings.Join(textParts, "")
	response := map[string]any{
		"candidates": []map[string]any{
			{
				"index": 0,
				"content": map[string]any{
					"role": "model",
					"parts": []map[string]any{
						{"text": fullText},
					},
				},
				"finishReason": "STOP",
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// writeGenAIError writes an error response in Google GenAI format.
func writeGenAIError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]any{
			"code":    status,
			"message": message,
			"status":  http.StatusText(status),
		},
	})
}
