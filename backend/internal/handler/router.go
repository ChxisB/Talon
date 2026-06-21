package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/talon/backend/internal/middleware"
)

func NewRouter() http.Handler {
	mux := http.NewServeMux()

	// Health
	mux.HandleFunc("GET /api/health", HandleHealth)

	// Session routes
	mux.HandleFunc("GET /api/session", HandleSessionList)
	mux.HandleFunc("POST /api/session", HandleSessionCreate)
	mux.HandleFunc("GET /api/session/{sessionID}", HandleSessionGet)

	// Session sub-routes
	mux.HandleFunc("GET /api/session/{sessionID}/message", HandleSessionMessages)
	mux.HandleFunc("POST /api/session/{sessionID}/prompt", HandleSessionPrompt)
	mux.HandleFunc("POST /api/session/{sessionID}/wait", HandleSessionWait)
	mux.HandleFunc("GET /api/session/{sessionID}/context", HandleSessionContext)
	mux.HandleFunc("POST /api/session/{sessionID}/compact", HandleSessionCompact)
	mux.HandleFunc("GET /api/session/{sessionID}/permission", HandleSessionPermissionList)
	mux.HandleFunc("POST /api/session/{sessionID}/permission/{requestID}/reply", HandleSessionPermissionReply)
	mux.HandleFunc("GET /api/session/{sessionID}/question", HandleSessionQuestionList)
	mux.HandleFunc("POST /api/session/{sessionID}/question/{requestID}/reply", HandleSessionQuestionReply)
	mux.HandleFunc("POST /api/session/{sessionID}/question/{requestID}/reject", HandleSessionQuestionReject)

	// Provider routes
	mux.HandleFunc("GET /api/provider", HandleProviderList)
	mux.HandleFunc("GET /api/provider/{providerID}", HandleProviderGet)

	// Model routes
	mux.HandleFunc("GET /api/model", HandleModelList)

	// Agent routes
	mux.HandleFunc("GET /api/agent", HandleAgentList)

	// Command routes
	mux.HandleFunc("GET /api/command", HandleCommandList)

	// Skill routes
	mux.HandleFunc("GET /api/skill", HandleSkillList)

	// Location
	mux.HandleFunc("GET /api/location", HandleLocationGet)

	// Filesystem
	mux.HandleFunc("GET /api/fs/read/", HandleFSRead)
	mux.HandleFunc("GET /api/fs/list", HandleFSList)
	mux.HandleFunc("GET /api/fs/find", HandleFSFind)

	// Credentials
	mux.HandleFunc("PATCH /api/credential/{credentialID}", HandleCredentialUpdate)
	mux.HandleFunc("DELETE /api/credential/{credentialID}", HandleCredentialDelete)

	// Permissions
	mux.HandleFunc("GET /api/permission/request", HandlePermissionRequestList)
	mux.HandleFunc("GET /api/permission/saved", HandlePermissionSavedList)
	mux.HandleFunc("DELETE /api/permission/saved/{id}", HandlePermissionSavedDelete)

	// Questions
	mux.HandleFunc("GET /api/question/request", HandleQuestionRequestList)

	// Events (SSE)
	mux.HandleFunc("GET /api/event", HandleEventSubscribe)

	// Integrations
	mux.HandleFunc("GET /api/integration", HandleIntegrationList)
	mux.HandleFunc("GET /api/integration/{integrationID}", HandleIntegrationGet)

	// References
	mux.HandleFunc("GET /api/reference", HandleReferenceList)

	// PTY
	mux.HandleFunc("GET /api/pty", HandlePTYList)
	mux.HandleFunc("POST /api/pty", HandlePTYCreate)
	mux.HandleFunc("GET /api/pty/{ptyID}", HandlePTYGet)
	mux.HandleFunc("PUT /api/pty/{ptyID}", HandlePTYUpdate)
	mux.HandleFunc("DELETE /api/pty/{ptyID}", HandlePTYRemove)
	mux.HandleFunc("POST /api/pty/{ptyID}/connect-token", HandlePTYConnectToken)
	mux.HandleFunc("GET /api/pty/{ptyID}/connect", HandlePTYConnect)

	// MCP
	mux.HandleFunc("GET /api/mcp/tools", HandleMCPToolsList)
	mux.HandleFunc("POST /api/mcp/call", HandleMCPCallTool)
	mux.HandleFunc("GET /api/mcp/config", HandleMCPGetConfig)
	mux.HandleFunc("POST /api/mcp/refresh", HandleMCPRefresh)

	// Agent / Chat
	mux.HandleFunc("POST /api/session/{sessionID}/chat", HandleSessionChat)
	mux.HandleFunc("POST /api/session/{sessionID}/stream", HandleSessionStream)
	mux.HandleFunc("POST /api/chat/complete", HandleChatComplete)

	// Project copy
	mux.HandleFunc("POST /experimental/project/{projectID}/copy", HandleProjectCopyCreate)
	mux.HandleFunc("DELETE /experimental/project/{projectID}/copy", HandleProjectCopyRemove)
	mux.HandleFunc("POST /experimental/project/{projectID}/copy/refresh", HandleProjectCopyRefresh)

	return middleware.WithCORS(middleware.WithAuth(mux))
}

// ── Helpers ─────────────────────────────────────────

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]any{
		"code":    status,
		"message": msg,
	})
}

func getPathParam(r *http.Request, name string) string {
	v := r.PathValue(name)
	if v == "" {
		v = r.URL.Query().Get(name)
	}
	return v
}

func getQueryParam(r *http.Request, name string) string {
	return r.URL.Query().Get(name)
}

func parseLocation(r *http.Request) string {
	// Location from header or query param
	loc := r.Header.Get("x-talon-directory")
	if loc == "" {
		loc = r.URL.Query().Get("directory")
	}
	if loc == "" {
		loc = r.URL.Query().Get("location[directory]")
	}
	return loc
}

func sanitizePath(path string) string {
	// Prevent directory traversal
	if strings.Contains(path, "..") {
		return ""
	}
	return path
}
