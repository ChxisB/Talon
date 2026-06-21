package handler

import (
	"encoding/json"
	"net/http"

	"github.com/talon/backend/internal/service"
)

func HandleMCPToolsList(w http.ResponseWriter, r *http.Request) {
	tools := service.GlobalMCPServers.GetAllTools()
	writeJSON(w, http.StatusOK, map[string]any{"tools": tools})
}

func HandleMCPCallTool(w http.ResponseWriter, r *http.Request) {
	var input service.MCPCallRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, 400, "Invalid request body")
		return
	}

	result, err := service.GlobalMCPServers.CallTool(input)
	if err != nil {
		writeError(w, 500, "MCP call failed: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"result": string(result),
	})
}

// HandleMCPGetConfig — returns the merged MCP tool list for the agent
func HandleMCPGetConfig(w http.ResponseWriter, r *http.Request) {
	dir := parseDir(r)
	config := service.FindMCPConfig(dir)
	if config == nil {
		writeJSON(w, http.StatusOK, map[string]any{"servers": map[string]any{}})
		return
	}

	// Start MCP servers if not already running
	service.GlobalMCPServers.StartAll(config, dir)

	writeJSON(w, http.StatusOK, config)
}

// HandleMCPRefresh — reload MCP config and restart servers
func HandleMCPRefresh(w http.ResponseWriter, r *http.Request) {
	service.GlobalMCPServers.StopAll()
	dir := parseDir(r)
	config := service.FindMCPConfig(dir)
	service.GlobalMCPServers.StartAll(config, dir)

	tools := service.GlobalMCPServers.GetAllTools()
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "refreshed",
		"tools":  tools,
	})
}
