package handler

import (
	"net/http"

	"github.com/talon/backend/internal/llm"
)

func HandleProviderList(w http.ResponseWriter, r *http.Request) {
	providers := llm.ListAvailableProviders()
	// Always include generic entries
	if len(providers) == 0 {
		providers = []map[string]any{
			{"id": "openai", "name": "OpenAI", "description": "Set OPENAI_API_KEY", "enabled": false},
			{"id": "anthropic", "name": "Anthropic", "description": "Set ANTHROPIC_API_KEY", "enabled": false},
			{"id": "google", "name": "Google", "description": "Set GOOGLE_API_KEY", "enabled": false},
			{"id": "openai-compatible", "name": "OpenAI Compatible", "description": "Set OPENAI_API_KEY + OPENAI_BASE_URL", "enabled": false},
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"providers": providers})
}

func HandleProviderGet(w http.ResponseWriter, r *http.Request) {
	providerID := getPathParam(r, "providerID")
	for _, p := range llm.ListAvailableProviders() {
		if p["id"] == providerID {
			writeJSON(w, http.StatusOK, p)
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id": providerID, "name": providerID,
		"description": "Provider not configured. Set the appropriate API key environment variable.",
		"enabled":     false,
	})
}
