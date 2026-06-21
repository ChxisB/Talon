package handler

import (
	"net/http"

	"github.com/talon/backend/internal/llm"
)

func HandleModelList(w http.ResponseWriter, r *http.Request) {
	models := llm.ListAvailableModels()
	if len(models) == 0 {
		models = []map[string]any{
			{"id": "gpt-4o", "providerID": "openai", "name": "GPT-4o", "enabled": false},
			{"id": "claude-sonnet-4-20250514", "providerID": "anthropic", "name": "Claude Sonnet 4", "enabled": false},
			{"id": "gemini-2.0-flash", "providerID": "google", "name": "Gemini 2.0 Flash", "enabled": false},
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"models": models})
}
