package handler

import (
	"encoding/json"
	"net/http"

	"github.com/talon/backend/internal/service"
)

func HandleAgentList(w http.ResponseWriter, r *http.Request) {
	agents := []map[string]any{
		{"id": "open-code-agent", "name": "Default Agent", "description": "General-purpose coding agent"},
		{"id": "architect", "name": "Architect", "description": "Architecture and design agent"},
	}
	writeJSON(w, http.StatusOK, map[string]any{"agents": agents})
}

// HandleSessionChat — sends a prompt to the agent and streams the response as SSE
// POST /api/session/:sessionID/chat
func HandleSessionChat(w http.ResponseWriter, r *http.Request) {
	sessionID := getPathParam(r, "sessionID")
	if sessionID == "" {
		writeError(w, 400, "sessionID required")
		return
	}

	var input struct {
		Prompt string `json:"prompt"`
		Model  string `json:"model,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, 400, "Invalid request body")
		return
	}

	if input.Prompt == "" {
		writeError(w, 400, "prompt is required")
		return
	}

	// Get or create agent session
	agent := service.GlobalAgents.GetOrCreate(sessionID, input.Model)

	// Set up SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	flusher, canFlush := w.(http.Flusher)

	// Start agent loop in background
	events := agent.Run(input.Prompt)
	if events == nil {
		// Already running
		writeJSON(w, http.StatusConflict, map[string]string{
			"error": "Agent is already running for this session",
		})
		return
	}

	// Stream events
	for event := range events {
		data, _ := json.Marshal(event)
		w.Write([]byte("data: " + string(data) + "\n\n"))
		if canFlush {
			flusher.Flush()
		}

		// Stop if client disconnected
		select {
		case <-r.Context().Done():
			return
		default:
		}
	}

	w.Write([]byte("data: [DONE]\n\n"))
	if canFlush {
		flusher.Flush()
	}
}

// HandleSessionStream — alias for HandleSessionChat
// Used by the AI client to stream agent responses
func HandleSessionStream(w http.ResponseWriter, r *http.Request) {
	HandleSessionChat(w, r)
}
