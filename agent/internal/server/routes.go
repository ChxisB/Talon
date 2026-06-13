package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ChxisB/spectre-proxy/agent/internal/config"
	"github.com/ChxisB/spectre-proxy/agent/internal/logger"
	"github.com/ChxisB/spectre-proxy/agent/internal/protocol"
	"github.com/ChxisB/spectre-proxy/agent/internal/providers"
	"github.com/ChxisB/spectre-proxy/agent/internal/router"
	"github.com/ChxisB/spectre-proxy/agent/internal/tools"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// getEnvOrDotenv returns the value of an env var. Priority order:
//  1. ~/.spectre-proxy/.env (dashboard saves take immediate effect)
//  2. os.Getenv (Docker env vars / host environment)
// This ensures API keys saved by the dashboard are picked up
// without restarting the proxy process, while still allowing
// Docker env vars to serve as a fallback.
func getEnvOrDotenv(key string) string {
	// 1. Check .env file first (dashboard saves are live)
	home, err := os.UserHomeDir()
	if err == nil {
		data, err := os.ReadFile(filepath.Join(home, ".spectre-proxy", ".env"))
		if err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				trimmed := strings.TrimSpace(line)
				if strings.HasPrefix(trimmed, key+"=") {
					return strings.TrimPrefix(trimmed, key+"=")
				}
			}
		}
	}
	// 2. Fall back to process environment (Docker / host)
	return os.Getenv(key)
}

func (s *Server) setupRoutes() {
	log.Println("[debug] setupRoutes called")

	// Provider metadata
	s.router.HandleFunc("/v1/providers", s.handleListProviders).Methods("GET", "HEAD", "OPTIONS")
	s.router.HandleFunc("/v1/providers/{id}", s.handleGetProvider).Methods("GET", "HEAD", "OPTIONS")

	// Health
	s.router.HandleFunc("/health", s.handleHealth).Methods("GET", "HEAD", "OPTIONS")

	// Root
	s.router.HandleFunc("/", s.handleRoot).Methods("GET", "HEAD", "OPTIONS")

	// Models — returns both Anthropic and OpenAI-compatible model lists
	// Codex reads the "data" array with "id" fields, same as Anthropic format.
	// Gemini uses a separate endpoint at /v1beta/models.
	s.router.HandleFunc("/v1/models", s.handleListModels).Methods("GET", "HEAD", "OPTIONS")

	// Messages (Anthropic API — used by Claude Code)
	s.router.HandleFunc("/v1/messages", s.handleCreateMessage).Methods("POST")
	s.router.HandleFunc("/v1/messages", s.handleProbe).Methods("HEAD", "OPTIONS")

	// Token count (Anthropic API)
	s.router.HandleFunc("/v1/messages/count_tokens", s.handleCountTokens).Methods("POST")
	s.router.HandleFunc("/v1/messages/count_tokens", s.handleProbe).Methods("HEAD", "OPTIONS")

	// Responses API (OpenAI — used by Codex CLI)
	s.router.HandleFunc("/v1/responses", s.handleResponsesAPI).Methods("POST")

	// Google GenAI API (used by Gemini CLI)
	// Model listing
	s.router.HandleFunc("/{version:[^/]+}/models", s.handleGenAIModels).Methods("GET", "HEAD", "OPTIONS")
	s.router.HandleFunc("/models", s.handleGenAIModels).Methods("GET", "HEAD", "OPTIONS")
	// Inference — model name is everything before the colon
	s.router.HandleFunc("/{version:[^/]+}/models/{model:[^:]+}:generateContent", s.handleGenAI).Methods("POST")
	s.router.HandleFunc("/{version:[^/]+}/models/{model:[^:]+}:streamGenerateContent", s.handleGenAIStream).Methods("POST")
	// Also handle without version prefix (just in case)
	s.router.HandleFunc("/models/{model:[^:]+}:generateContent", s.handleGenAI).Methods("POST")
	s.router.HandleFunc("/models/{model:[^:]+}:streamGenerateContent", s.handleGenAIStream).Methods("POST")

	// OpenAI format model list (compatible with both Codex and general OpenAI clients)
	s.router.HandleFunc("/v1/responses/models", s.handleOpenAIModels).Methods("GET", "HEAD", "OPTIONS")

	// Admin API
	s.router.HandleFunc("/admin/api/config", s.handleAdminGetConfig).Methods("GET")
	s.router.HandleFunc("/admin/api/config/validate", s.handleAdminValidateConfig).Methods("POST")
	s.router.HandleFunc("/admin/api/config/apply", s.handleAdminApplyConfig).Methods("POST")
	s.router.HandleFunc("/admin/api/status", s.handleAdminStatus).Methods("GET")
}

// Middleware: require API key
func (s *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.config.APIKey != "" {
			key := r.Header.Get("x-api-key")
			if key == "" {
				key = r.Header.Get("authorization")
				if strings.HasPrefix(key, "Bearer ") {
					key = strings.TrimPrefix(key, "Bearer ")
				}
			}
			if key != s.config.APIKey {
				http.Error(w, `{"type":"error","error":{"type":"authentication_error","message":"Invalid API key"}}`, http.StatusUnauthorized)
				return
			}
		}
		next(w, r)
	}
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status":   "ok",
		"provider": s.config.ProviderType,
		"model":    s.config.Model,
	})
}

func (s *Server) handleProbe(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

// Hardcoded models for providers that don't support model listing
var hardcodedProviderModels = map[string][]string{
	"opencode": {
		"opencode/glm-5.1",
		"opencode/glm-5",
		"opencode/kimi-k2.5",
		"opencode/kimi-k2.6",
		"opencode/mimo-v2.5-free",
		"opencode/mimo-v2.5-pro",
		"opencode/minimax-m3",
		"opencode/minimax-m2.7",
		"opencode/minimax-m2.5",
		"opencode/qwen3.7-max",
		"opencode/qwen3.7-plus",
		"opencode/qwen3.6-plus",
		"opencode/qwen3.5-plus",
		"opencode/deepseek-v4-pro",
		"opencode/deepseek-v4-flash",
		"opencode/deepseek-v4-flash-free",
		"opencode/nemotron-3-ultra-free",
		"opencode/north-mini-code-free",
		"opencode/big-pickle",
		"opencode/grok-build-0.1",
	},
	"opencode_go": {
		"opencode_go/gpt-5.5",
		"opencode_go/gpt-5.5-pro",
		"opencode_go/gpt-5.4",
		"opencode_go/gpt-5.4-pro",
		"opencode_go/gpt-5.4-mini",
		"opencode_go/gpt-5.4-nano",
		"opencode_go/gpt-5.3-codex",
		"opencode_go/gpt-5.3-codex-spark",
		"opencode_go/gpt-5.2",
		"opencode_go/gpt-5.2-codex",
		"opencode_go/gpt-5.1",
		"opencode_go/gpt-5.1-codex",
		"opencode_go/gpt-5.1-codex-max",
		"opencode_go/gpt-5.1-codex-mini",
		"opencode_go/gpt-5",
		"opencode_go/gpt-5-codex",
		"opencode_go/gpt-5-nano",
		"opencode_go/claude-fable-5",
		"opencode_go/claude-opus-4.8",
		"opencode_go/claude-opus-4.7",
		"opencode_go/claude-opus-4.6",
		"opencode_go/claude-opus-4.5",
		"opencode_go/claude-opus-4.1",
		"opencode_go/claude-sonnet-4.6",
		"opencode_go/claude-sonnet-4.5",
		"opencode_go/claude-sonnet-4",
		"opencode_go/claude-haiku-4.5",
		"opencode_go/claude-3-5-haiku",
		"opencode_go/gemini-3.5-flash",
		"opencode_go/gemini-3.1-pro",
		"opencode_go/gemini-3-flash",
		"opencode_go/qwen3.7-max",
		"opencode_go/qwen3.7-plus",
		"opencode_go/qwen3.6-plus",
		"opencode_go/qwen3.5-plus",
		"opencode_go/deepseek-v4-pro",
		"opencode_go/deepseek-v4-flash",
		"opencode_go/minimax-m2.7",
		"opencode_go/minimax-m2.5",
		"opencode_go/glm-5.1",
		"opencode_go/glm-5",
		"opencode_go/kimi-k2.5",
		"opencode_go/kimi-k2.6",
		"opencode_go/grok-build-0.1",
		"opencode_go/big-pickle",
		"opencode_go/mimo-v2.5-free",
		"opencode_go/north-mini-code-free",
		"opencode_go/nemotron-3-ultra-free",
		"opencode_go/deepseek-v4-flash-free",
	},
}

// providerEnvKey maps provider IDs to their API key env vars.
var providerEnvKey = map[string]string{
	"open_router": "OPENROUTER_API_KEY",
	"nvidia_nim":  "NVIDIA_NIM_API_KEY",
	"gemini":      "GEMINI_API_KEY",
	"deepseek":    "DEEPSEEK_API_KEY",
	"mistral":     "MISTRAL_API_KEY",
	"codestral":   "CODESTRAL_API_KEY",
	"opencode":    "OPENCODE_API_KEY",
	"opencode_go": "OPENCODE_API_KEY",
	"wafer":       "WAFER_API_KEY",
	"kimi":        "KIMI_API_KEY",
	"cerebras":    "CEREBRAS_API_KEY",
	"groq":        "GROQ_API_KEY",
	"fireworks":   "FIREWORKS_API_KEY",
	"zai":         "ZAI_API_KEY",
	"ollama":      "",
	"lmstudio":    "",
	"llamacpp":    "",
}

func (s *Server) handleListModels(w http.ResponseWriter, r *http.Request) {
	var models []protocol.ModelResponse
	seen := make(map[string]bool)

	// Add the currently configured model first so it appears at the top
	if s.config.Model != "" && !seen[s.config.Model] {
		seen[s.config.Model] = true
		models = append(models, protocol.ModelResponse{
			ID:          s.config.Model,
			DisplayName: s.config.Model + " (current)",
			CreatedAt:   "",
		})
	}

	// Try every registered provider that has an API key set
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	for _, desc := range config.ProviderCatalog() {
		envKey := providerEnvKey[desc.ID]
		if envKey == "" {
			// Local-only providers (Ollama, LM Studio, llama.cpp) — try anyway
		} else if getEnvOrDotenv(envKey) == "" {
			continue // skip providers without API keys
		}

		// Build a ProviderConfig with the correct API key
		cfg := providers.DefaultProviderConfig()
		if envKey != "" {
			cfg.APIKey = getEnvOrDotenv(envKey)
		}

		provider, err := s.Registry.Get(desc.ID, cfg, s.config)
		if err != nil {
			continue
		}

		providerModels, err := provider.ListModels(ctx)
		if err != nil {
			// If listing fails but the provider has an API key, use hardcoded models
			if hardcoded, ok := hardcodedProviderModels[desc.ID]; ok && len(hardcoded) > 0 {
				for _, id := range hardcoded {
					if !seen[id] {
						seen[id] = true
						models = append(models, protocol.ModelResponse{ID: id, DisplayName: id})
					}
				}
			}
			continue
		}

		for _, id := range providerModels {
			fullID := id
			// If the model ID doesn't include a provider prefix, add one
			if !strings.Contains(id, "/") && desc.ID != "" {
				fullID = desc.ID + "/" + id
			}
			// Normalise provider prefix aliases (e.g. openrouter/ → open_router/)
			// so the router can match them when the user selects a model.
			if slash := strings.IndexByte(fullID, '/'); slash > 0 {
				prefix := fullID[:slash]
				if canonical := router.ResolveProviderID(prefix, nil); canonical != "" && canonical != prefix {
					fullID = canonical + fullID[slash:]
				}
			}
			if !seen[fullID] {
				seen[fullID] = true
				models = append(models, protocol.ModelResponse{
					ID:          fullID,
					DisplayName: fullID,
					CreatedAt:   "",
				})
			}
		}
	}

	// Fallback: static Claude models (always included)
	for _, m := range staticClaudeModels() {
		if !seen[m.ID] {
			seen[m.ID] = true
			models = append(models, m)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(protocol.ModelsListResponse{
		Data:    models,
		HasMore: false,
	})
}

// collectModelIDs returns a deduplicated list of model IDs from all available providers.
func (s *Server) collectModelIDs() []string {
	seen := make(map[string]bool)
	var ids []string

	// Add configured default model first
	if s.config.Model != "" && !seen[s.config.Model] {
		seen[s.config.Model] = true
		ids = append(ids, s.config.Model)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for _, desc := range config.ProviderCatalog() {
		envKey := providerEnvKey[desc.ID]
		if envKey == "" {
			// Local-only — try anyway
		} else if getEnvOrDotenv(envKey) == "" {
			continue
		}
		cfg := providers.DefaultProviderConfig()
		if envKey != "" {
			cfg.APIKey = getEnvOrDotenv(envKey)
		}
		provider, err := s.Registry.Get(desc.ID, cfg, s.config)
		if err != nil {
			continue
		}
		providerModels, err := provider.ListModels(ctx)
		if err != nil {
			if hardcoded, ok := hardcodedProviderModels[desc.ID]; ok {
				for _, id := range hardcoded {
					if !seen[id] {
						seen[id] = true
						ids = append(ids, id)
					}
				}
			}
			continue
		}
		for _, id := range providerModels {
			if !strings.Contains(id, "/") && desc.ID != "" {
				id = desc.ID + "/" + id
			}
			if !seen[id] {
				seen[id] = true
				ids = append(ids, id)
			}
		}
	}

	// Static Claude fallbacks
	for _, m := range staticClaudeModels() {
		if !seen[m.ID] {
			seen[m.ID] = true
			ids = append(ids, m.ID)
		}
	}

	return ids
}

// handleOpenAIModels returns models in OpenAI format (compatible with Codex CLI).
// GET /v1/responses/models
func (s *Server) handleOpenAIModels(w http.ResponseWriter, r *http.Request) {
	modelIDs := s.collectModelIDs()
	type openAIModel struct {
		ID      string `json:"id"`
		Object  string `json:"object"`
		Created int64  `json:"created"`
		OwnedBy string `json:"owned_by"`
	}
	data := make([]openAIModel, 0, len(modelIDs))
	for _, id := range modelIDs {
		data = append(data, openAIModel{
			ID:      id,
			Object:  "model",
			Created: 0,
			OwnedBy: "spectre",
		})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"object": "list",
		"data":   data,
	})
}

// handleGenAIModels returns models in Google GenAI format (compatible with Gemini CLI).
// GET /v1beta/models
func (s *Server) handleGenAIModels(w http.ResponseWriter, r *http.Request) {
	modelIDs := s.collectModelIDs()
	type genAIModel struct {
		Name                       string   `json:"name"`
		DisplayName                string   `json:"displayName"`
		SupportedGenerationMethods []string `json:"supportedGenerationMethods"`
	}
	models := make([]genAIModel, 0, len(modelIDs))
	for _, id := range modelIDs {
		displayName := id
		// Strip provider prefix for display
		if idx := strings.IndexByte(id, '/'); idx >= 0 {
			displayName = id[idx+1:]
		}
		models = append(models, genAIModel{
			Name:                       "models/" + displayName,
			DisplayName:                displayName,
			SupportedGenerationMethods: []string{"generateContent", "streamGenerateContent"},
		})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"models": models,
	})
}

func (s *Server) handleCreateMessage(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Model       string             `json:"model"`
		Messages    []protocol.Message `json:"messages"`
		System      json.RawMessage    `json:"system,omitempty"`
		MaxTokens   int                `json:"max_tokens,omitempty"`
		Stream      bool               `json:"stream"`
		Temperature *float64           `json:"temperature,omitempty"`
		TopP        *float64           `json:"top_p,omitempty"`
		Tools       []protocol.ToolDef `json:"tools,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", fmt.Sprintf("Invalid request: %v", err))
		return
	}

	if len(req.Messages) == 0 {
		writeError(w, http.StatusBadRequest, "invalid_request", "messages cannot be empty")
		return
	}

	// Build the protocol request
	msgReq := &protocol.MessagesRequest{
		Model:       req.Model,
		Messages:    req.Messages,
		System:      req.System,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		TopP:        req.TopP,
		Tools:       req.Tools,
	}

	// Log incoming request
	logger.Log("info", "", req.Model, "inbound",
		fmt.Sprintf("model=%s messages=%d tools=%d stream=%v",
			req.Model, len(req.Messages), len(req.Tools), req.Stream), "", map[string]any{
			"model":    req.Model,
			"messages": len(req.Messages),
			"tools":    len(req.Tools),
			"stream":   req.Stream,
		})
	for _, t := range req.Tools {
		toolJSON, _ := json.Marshal(t)
		logger.Log("info", "", req.Model, "tool", string(toolJSON), "", nil)
	}

	// Resolve model to provider
	resolved := s.modelRoute.Resolve(req.Model)

	// Check optimization handlers
	if resp := tryOptimizations(msgReq, resolved, s.config); resp != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Check for local web tool (web_search / web_fetch) handling
	if s.config.EnableWebServerTools {
		if resp := handleWebToolRequest(r.Context(), msgReq); resp != nil {
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")
			w.Write([]byte(resp))
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
			return
		}
	}

	// Get provider with its API key
	providerCfg := s.providerCfg
	if envKey := providerEnvKey[resolved.ProviderID]; envKey != "" {
		if key := getEnvOrDotenv(envKey); key != "" {
			providerCfg.APIKey = key
			log.Printf("[debug] using key for %s: %s... (len=%d)", resolved.ProviderID, key[:12], len(key))
		} else {
			log.Printf("[debug] no key found for %s (env=%s)", resolved.ProviderID, envKey)
		}
	} else {
		log.Printf("[debug] no env key mapping for %s", resolved.ProviderID)
	}
	provider, err := s.Registry.Get(resolved.ProviderID, providerCfg, s.config)
	if err != nil {
		writeError(w, http.StatusBadGateway, "provider_error", fmt.Sprintf("Provider %s not available: %v", resolved.ProviderID, err))
		return
	}

	msgReq.Model = resolved.ProviderModel

	// Stream response
	ctx := r.Context()
	events, err := provider.StreamAnthropic(ctx, msgReq, 0, resolved.ThinkingEnabled)
	if err != nil {
		writeError(w, http.StatusBadGateway, "provider_error", fmt.Sprintf("Provider error: %v", err))
		return
	}

	if !req.Stream {
		// Non-streaming: collect all events and build a single JSON response
		w.Header().Set("Content-Type", "application/json")
		resp := buildNonStreamingResponse(events, req.Model)
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Streaming: send SSE events
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		log.Println("Warning: ResponseWriter does not support flushing")
	}

	// Send at least one event so claude doesn't get an empty 200 response
	sentEvent := false
	for event := range events {
		// Skip events with empty data — they cause "JSON Parse error" in claude
		if len(event.Data) == 0 {
			continue
		}
		if !sentEvent {
			sentEvent = true
		}
		writeSSE(w, event.Type, json.RawMessage(event.Data))
		if flusher != nil {
			flusher.Flush()
		}
	}

	// If the provider returned an empty channel, send an error event
	if !sentEvent {
		log.Printf("[warn] provider %s returned empty stream for model %s", resolved.ProviderID, req.Model)
		errData, _ := json.Marshal(map[string]any{
			"type": "error",
			"error": map[string]any{
				"type":    "empty_response",
				"message": fmt.Sprintf("Provider %s returned an empty response", resolved.ProviderID),
			},
		})
		writeSSE(w, "error", json.RawMessage(errData))
		writeSSE(w, "message_stop", map[string]string{"type": "message_stop"})
		flusher.Flush()
	}
}

func (s *Server) handleCountTokens(w http.ResponseWriter, r *http.Request) {
	var req protocol.TokenCountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", fmt.Sprintf("Invalid request: %v", err))
		return
	}

	// Estimate tokens (simple heuristic: ~4 chars per token)
	inputTokens := 0
	for _, msg := range req.Messages {
		if blocks, err := msg.ContentBlocks(); err == nil {
			for _, block := range blocks {
				inputTokens += len(block.Text) / 4
				if inputTokens < 1 {
					inputTokens = 1
				}
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(protocol.TokenCountResponse{InputTokens: inputTokens})
}

// ─── Admin API handlers ─────────────────────────────────────────────

func (s *Server) handleAdminGetConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"fields": []map[string]any{
			{"key": "API_KEY", "label": "API Key", "type": "secret", "value": maskSecret(s.config.APIKey), "source": "env"},
			{"key": "MODEL", "label": "Default Model", "type": "text", "value": s.config.Model, "source": "env"},
			{"key": "HOST", "label": "Host", "type": "text", "value": s.config.Host, "source": "env"},
			{"key": "PORT", "label": "Port", "type": "number", "value": fmt.Sprintf("%d", s.config.Port), "source": "env"},
		},
	})
}

func (s *Server) handleAdminValidateConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"valid": true})
}

func (s *Server) handleAdminApplyConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"status": "applied", "restart_required": false})
}

func (s *Server) handleAdminStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status":   "running",
		"host":     s.config.Host,
		"port":     s.config.Port,
		"model":    s.config.Model,
		"provider": s.config.ProviderType,
	})
}

// handleListProviders returns metadata for all registered providers.
func (s *Server) handleListProviders(w http.ResponseWriter, r *http.Request) {
	log.Printf("[debug] handleListProviders called for %s", r.URL.Path)
	providers := s.Registry.ListProviderMetadata()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"providers": providers,
		"count":     len(providers),
	})
}

// handleGetProvider returns metadata for a specific provider.
func (s *Server) handleGetProvider(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	meta := s.Registry.GetMetadata(id)
	if meta == nil {
		http.Error(w, `{"type":"error","error":{"type":"not_found","message":"Provider not found"}}`, http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(meta)
}

// ─── Helpers ────────────────────────────────────────────────────────

// buildNonStreamingResponse collects all SSE events and assembles a single
// Anthropic Messages API JSON response for non-streaming requests.
func buildNonStreamingResponse(events <-chan protocol.SSEEvent, model string) *protocol.MessagesResponse {
	var content []protocol.ContentBlock
	var stopReason string
	var usage protocol.Usage
	var currentBlock *protocol.ContentBlock
	var inputTokens int

	for event := range events {
		if len(event.Data) == 0 {
			continue
		}

		switch event.Type {
		case "message_start":
			var msg struct {
				Message struct {
					Usage struct {
						InputTokens  int `json:"input_tokens"`
						OutputTokens int `json:"output_tokens"`
					} `json:"usage"`
				} `json:"message"`
				Type string `json:"type"`
			}
			json.Unmarshal(event.Data, &msg)
			inputTokens = msg.Message.Usage.InputTokens

		case "content_block_start":
			var cb struct {
				ContentBlock struct {
					Type  string         `json:"type"`
					ID    string         `json:"id,omitempty"`
					Name  string         `json:"name,omitempty"`
					Input map[string]any `json:"input,omitempty"`
				} `json:"content_block"`
			}
			if err := json.Unmarshal(event.Data, &cb); err != nil {
				continue
			}
			block := protocol.ContentBlock{Type: cb.ContentBlock.Type}
			if cb.ContentBlock.Type == "tool_use" {
				block.ID = cb.ContentBlock.ID
				block.Name = cb.ContentBlock.Name
				block.Input = cb.ContentBlock.Input
			}
			content = append(content, block)
			currentBlock = &content[len(content)-1]

		case "content_block_delta":
			var delta struct {
				Delta struct {
					Type string `json:"type"`
					Text string `json:"text,omitempty"`
				} `json:"delta"`
			}
			if err := json.Unmarshal(event.Data, &delta); err != nil {
				continue
			}
			if currentBlock != nil && delta.Delta.Type == "text_delta" {
				currentBlock.Text += delta.Delta.Text
			}

		case "content_block_stop":
			currentBlock = nil

		case "message_delta":
			var md struct {
				Delta struct {
					StopReason string `json:"stop_reason"`
				} `json:"delta"`
				Usage struct {
					OutputTokens int `json:"output_tokens"`
				} `json:"usage"`
			}
			json.Unmarshal(event.Data, &md)
			stopReason = md.Delta.StopReason
			usage.OutputTokens = md.Usage.OutputTokens
		}
	}

	if stopReason == "" {
		stopReason = "end_turn"
	}
	usage.InputTokens = inputTokens

	return &protocol.MessagesResponse{
		ID:         fmt.Sprintf("msg_%s", uuid.New().String()),
		Model:      model,
		Role:       "assistant",
		Content:    content,
		StopReason: stopReason,
		Usage:      usage,
		Type:       "message",
	}
}

func writeError(w http.ResponseWriter, status int, errType, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{
		"type": "error",
		"error": map[string]any{
			"type":    errType,
			"message": msg,
		},
	})
}

func writeSSE(w http.ResponseWriter, eventType string, data any) {
	var jsonData []byte
	switch d := data.(type) {
	case []byte:
		jsonData = d
	case json.RawMessage:
		jsonData = []byte(d)
	default:
		jsonData, _ = json.Marshal(data)
	}
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventType, string(jsonData))
}

func maskSecret(s string) string {
	if len(s) <= 8 {
		return "********"
	}
	return s[:4] + "****" + s[len(s)-4:]
}

// handleWebToolRequest checks if the request is for web_search/web_fetch
// and handles it locally, returning SSE-encoded response bytes or nil.
func handleWebToolRequest(ctx context.Context, req *protocol.MessagesRequest) []byte {
	lastMsg := lastUserMessage(req.Messages)
	if lastMsg == "" {
		return nil
	}

	// Detect web_search tool request
	if hasTool(req.Tools, "web_search") && isWebSearchRequest(lastMsg) {
		query := extractSearchQuery(lastMsg)
		if query == "" {
			return nil
		}

		results, err := tools.WebSearch(ctx, query)
		if err != nil {
			return buildWebToolSSEError("web_search", err.Error())
		}

		return buildWebToolSSEResult("web_search", query, results)
	}

	// Detect web_fetch tool request
	if hasTool(req.Tools, "web_fetch") && isWebFetchRequest(lastMsg) {
		fetchURL := extractFetchURL(lastMsg)
		if fetchURL == "" {
			return nil
		}

		egress := tools.DefaultEgressPolicy()
		result, err := tools.WebFetch(ctx, fetchURL, egress)
		if err != nil {
			return buildWebToolSSEError("web_fetch", err.Error())
		}

		return buildWebFetchSSEResult(result)
	}

	return nil
}

func hasTool(tools []protocol.ToolDef, name string) bool {
	for _, t := range tools {
		if t.Name == name {
			return true
		}
	}
	return false
}

func isWebSearchRequest(text string) bool {
	lower := strings.ToLower(text)
	return strings.Contains(lower, "search") || strings.Contains(lower, "look up") || strings.Contains(lower, "find")
}

func isWebFetchRequest(text string) bool {
	return strings.HasPrefix(text, "http://") || strings.HasPrefix(text, "https://")
}

func extractSearchQuery(text string) string {
	// Simple heuristic: take the last line or use the whole text
	lines := strings.Split(text, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line != "" && !strings.HasPrefix(line, "http") {
			return line
		}
	}
	return strings.TrimSpace(text)
}

func extractFetchURL(text string) string {
	for _, word := range strings.Fields(text) {
		if strings.HasPrefix(word, "http://") || strings.HasPrefix(word, "https://") {
			return strings.TrimRight(word, ".,;:!?")
		}
	}
	return ""
}

func buildWebToolSSEEvent(eventType string, data map[string]any) []byte {
	msgID := "msg_" + uuid.New().String()
	toolID := "tool_" + uuid.New().String()
	data["message_id"] = msgID
	data["tool_id"] = toolID

	content, _ := json.Marshal(data)

	var buf strings.Builder
	buf.WriteString(fmt.Sprintf("event: %s\ndata: %s\n\n", eventType, string(content)))
	return []byte(buf.String())
}

func buildWebToolSSEResult(toolName, query string, results []tools.SearchResult) []byte {
	var buf strings.Builder

	// message_start
	msgID := "msg_" + uuid.New().String()
	buf.WriteString(fmt.Sprintf("event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"id\":%q,\"type\":\"message\",\"role\":\"assistant\",\"model\":\"web_search\",\"content\":[]}}\n\n", msgID))

	// content_block_start (tool_use)
	buf.WriteString(fmt.Sprintf("event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"tool_use\",\"id\":\"tool_%s\",\"name\":%q,\"input\":{\"query\":%q}}}\n\n", uuid.New().String(), toolName, query))

	// content_block_stop
	buf.WriteString("event: content_block_stop\ndata: {\"type\":\"content_block_stop\",\"index\":0}\n\n")

	// content_block_start (tool_result)
	summary := webSearchSummary(query, results)
	buf.WriteString(fmt.Sprintf("event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":1,\"content_block\":{\"type\":\"text\",\"text\":%q}}\n\n", summary))

	// content_block_stop
	buf.WriteString("event: content_block_stop\ndata: {\"type\":\"content_block_stop\",\"index\":1}\n\n")

	// message_delta
	buf.WriteString("event: message_delta\ndata: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\",\"stop_sequence\":null},\"usage\":{\"input_tokens\":10,\"output_tokens\":50}}\n\n")

	// message_stop
	buf.WriteString("event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n")

	return []byte(buf.String())
}

func buildWebFetchSSEResult(result *tools.FetchResult) []byte {
	var buf strings.Builder

	msgID := "msg_" + uuid.New().String()
	buf.WriteString(fmt.Sprintf("event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"id\":%q,\"type\":\"message\",\"role\":\"assistant\",\"model\":\"web_fetch\",\"content\":[]}}\n\n", msgID))

	summary := fmt.Sprintf("Title: %s\nURL: %s\n\n%s", result.Title, result.URL, result.Data)

	buf.WriteString(fmt.Sprintf("event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\",\"text\":%q}}\n\n", summary))
	buf.WriteString("event: content_block_stop\ndata: {\"type\":\"content_block_stop\",\"index\":0}\n\n")
	buf.WriteString("event: message_delta\ndata: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\",\"stop_sequence\":null},\"usage\":{\"input_tokens\":10,\"output_tokens\":50}}\n\n")
	buf.WriteString("event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n")

	return []byte(buf.String())
}

func buildWebToolSSEError(toolName, errMsg string) []byte {
	var buf strings.Builder
	msgID := "msg_" + uuid.New().String()

	buf.WriteString(fmt.Sprintf("event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"id\":%q,\"type\":\"message\",\"role\":\"assistant\",\"model\":%q,\"content\":[]}}\n\n", msgID, toolName))
	buf.WriteString(fmt.Sprintf("event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\",\"text\":%q}}\n\n", errMsg))
	buf.WriteString("event: content_block_stop\ndata: {\"type\":\"content_block_stop\",\"index\":0}\n\n")
	buf.WriteString("event: message_delta\ndata: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"error\",\"stop_sequence\":null},\"usage\":{\"input_tokens\":5,\"output_tokens\":5}}\n\n")
	buf.WriteString("event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n")

	return []byte(buf.String())
}

func webSearchSummary(query string, results []tools.SearchResult) string {
	if len(results) == 0 {
		return fmt.Sprintf("No web search results found for: %s", query)
	}
	var lines []string
	lines = append(lines, fmt.Sprintf("Search results for: %s", query))
	for i, r := range results {
		lines = append(lines, fmt.Sprintf("%d. %s\n   %s", i+1, r.Title, r.URL))
	}
	return strings.Join(lines, "\n\n")
}

func staticClaudeModels() []protocol.ModelResponse {
	return []protocol.ModelResponse{
		{ID: "claude-opus-4-20250514", DisplayName: "Claude Opus 4", CreatedAt: "2025-05-14T00:00:00Z"},
		{ID: "claude-sonnet-4-20250514", DisplayName: "Claude Sonnet 4", CreatedAt: "2025-05-14T00:00:00Z"},
		{ID: "claude-haiku-4-20250514", DisplayName: "Claude Haiku 4", CreatedAt: "2025-05-14T00:00:00Z"},
	}
}
