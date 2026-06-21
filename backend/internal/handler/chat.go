package handler

import (
	"encoding/json"
	"net/http"

	"github.com/talon/backend/internal/llm"
)

func HandleChatComplete(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Model    string          `json:"model"`
		Messages []llm.Message   `json:"messages"`
		Tools    []llm.ToolDefinition `json:"tools,omitempty"`
		Stream   bool            `json:"stream,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, 400, "Invalid request body")
		return
	}

	if input.Model == "" {
		writeError(w, 400, "model is required")
		return
	}

	provider, _, err := llm.GetProvider(input.Model)
	if err != nil {
		writeError(w, 400, "Failed to get provider: "+err.Error())
		return
	}

	req := llm.CompletionRequest{
		Model:    input.Model,
		Messages: input.Messages,
		Tools:    input.Tools,
		Stream:   input.Stream,
	}

	if input.Stream {
		// Streaming response
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.WriteHeader(http.StatusOK)

		events := make(chan llm.StreamEvent, 64)
		go func() {
			provider.CompleteStream(req, events)
		}()

		flusher, canFlush := w.(http.Flusher)
		for event := range events {
			data, _ := json.Marshal(event)
			w.Write([]byte("data: " + string(data) + "\n\n"))
			if canFlush {
				flusher.Flush()
			}
		}
		w.Write([]byte("data: [DONE]\n\n"))
		if canFlush {
			flusher.Flush()
		}
		return
	}

	// Non-streaming response
	resp, err := provider.Complete(req)
	if err != nil {
		writeError(w, 500, "Completion failed: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, resp)
}
