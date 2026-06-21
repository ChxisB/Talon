package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/talon/backend/internal/model"
	"github.com/talon/backend/internal/service"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for PTY WebSocket
	},
}

// ── Location ────────────────────────────────────────

func HandleLocationGet(w http.ResponseWriter, r *http.Request) {
	dir := parseLocation(r)
	if dir == "" {
		cwd, _ := os.Getwd()
		dir = cwd
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"directory": dir,
		"workspace": r.Header.Get("x-talon-workspace"),
	})
}

// ── Command ─────────────────────────────────────────

func HandleCommandList(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"commands": []any{}})
}

// ── Skill ───────────────────────────────────────────

func HandleSkillList(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"skills": []any{}})
}

// ── Filesystem ──────────────────────────────────────

func HandleFSRead(w http.ResponseWriter, r *http.Request) {
	dir := parseLocation(r)
	filePath := strings.TrimPrefix(r.URL.Path, "/api/fs/read/")
	filePath = sanitizePath(filePath)

	fullPath := filepath.Join(dir, filePath)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		writeError(w, 404, "File not found")
		return
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		writeError(w, 500, "Failed to read file: "+err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(data)
}

func HandleFSList(w http.ResponseWriter, r *http.Request) {
	dir := parseLocation(r)
	prefix := r.URL.Query().Get("prefix")
	if prefix == "" {
		prefix = "."
	}
	fullPath := filepath.Join(dir, prefix)

	entries, err := os.ReadDir(fullPath)
	if err != nil {
		writeError(w, 404, "Directory not found")
		return
	}

	files := []map[string]any{}
	for _, e := range entries {
		info, _ := e.Info()
		files = append(files, map[string]any{
			"name":  e.Name(),
			"isDir": e.IsDir(),
			"size":  info.Size(),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"files": files})
}

func HandleFSFind(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("query")
	if query == "" {
		writeJSON(w, http.StatusOK, map[string]any{"files": []any{}})
		return
	}

	dir := parseDir(r)
	var results []string
	filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if strings.Contains(strings.ToLower(d.Name()), strings.ToLower(query)) {
			results = append(results, path)
		}
		return nil
	})

	if len(results) > 50 {
		results = results[:50]
	}
	writeJSON(w, http.StatusOK, map[string]any{"files": results})
}

// ── Location helper (overloads the one from router.go) ──
func parseDir(r *http.Request) string {
	loc := r.Header.Get("x-talon-directory")
	if loc == "" {
		loc = r.URL.Query().Get("directory")
	}
	if loc == "" {
		loc = r.URL.Query().Get("location[directory]")
	}
	if loc == "" {
		cwd, _ := os.Getwd()
		loc = cwd
	}
	return loc
}

// ── Credential ──────────────────────────────────────

func HandleCredentialUpdate(w http.ResponseWriter, r *http.Request) {
	credentialID := getPathParam(r, "credentialID")
	var input struct {
		Label string `json:"label"`
	}
	json.NewDecoder(r.Body).Decode(&input)
	writeJSON(w, http.StatusOK, map[string]string{
		"id":    credentialID,
		"label": input.Label,
	})
}

func HandleCredentialDelete(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

// ── Permission ──────────────────────────────────────

func HandlePermissionRequestList(w http.ResponseWriter, r *http.Request) {
	requests := service.GlobalPermissions.List(parseDir(r))
	writeJSON(w, http.StatusOK, map[string]any{"requests": requests})
}

func HandlePermissionSavedList(w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("project")
	permissions, err := service.ListSavedPermissions(projectID)
	if err != nil {
		writeError(w, 500, "Failed to list saved permissions")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"permissions": permissions})
}

func HandlePermissionSavedDelete(w http.ResponseWriter, r *http.Request) {
	id := getPathParam(r, "id")
	if err := service.RemoveSavedPermission(id); err != nil {
		writeError(w, 500, "Failed to delete permission")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── Question ────────────────────────────────────────

func HandleQuestionRequestList(w http.ResponseWriter, r *http.Request) {
	requests := service.GlobalQuestions.List(parseDir(r))
	writeJSON(w, http.StatusOK, map[string]any{"requests": requests})
}

// ── Event (SSE) ─────────────────────────────────────

func HandleEventSubscribe(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	// Send initial keepalive
	w.Write([]byte(": keepalive\n\n"))
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

// ── Integration ─────────────────────────────────────

func HandleIntegrationList(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"integrations": []any{}})
}

func HandleIntegrationGet(w http.ResponseWriter, r *http.Request) {
	integrationID := getPathParam(r, "integrationID")
	writeJSON(w, http.StatusOK, map[string]any{
		"id":   integrationID,
		"name": integrationID,
	})
}

// ── Reference ───────────────────────────────────────

func HandleReferenceList(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"references": []any{}})
}

// ── PTY ─────────────────────────────────────────────

func HandlePTYList(w http.ResponseWriter, r *http.Request) {
	ptys := service.GlobalPTYs.List()
	writeJSON(w, http.StatusOK, map[string]any{"ptys": ptys})
}

func HandlePTYCreate(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Title  string         `json:"title"`
		Args   []string       `json:"args"`
		Cwd    string         `json:"cwd"`
		Env    map[string]string `json:"env"`
		Size   *service.Size  `json:"size"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, 400, "Invalid request body")
		return
	}

	if input.Cwd == "" {
		cwd, _ := os.Getwd()
		input.Cwd = cwd
	}

	sess, err := service.GlobalPTYs.Create(input.Cwd, input.Title, input.Args, input.Size)
	if err != nil {
		writeError(w, 500, "Failed to create PTY: "+err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, sess.Info)
}

func HandlePTYGet(w http.ResponseWriter, r *http.Request) {
	ptyID := getPathParam(r, "ptyID")
	sess := service.GlobalPTYs.Get(ptyID)
	if sess == nil {
		writeError(w, 404, "PTY not found")
		return
	}
	writeJSON(w, http.StatusOK, sess.Info)
}

func HandlePTYUpdate(w http.ResponseWriter, r *http.Request) {
	ptyID := getPathParam(r, "ptyID")
	sess := service.GlobalPTYs.Get(ptyID)
	if sess == nil {
		writeError(w, 404, "PTY not found")
		return
	}

	var input struct {
		Title string       `json:"title"`
		Size  *service.Size `json:"size"`
	}
	json.NewDecoder(r.Body).Decode(&input)

	if input.Title != "" {
		sess.Info.Title = input.Title
	}
	if input.Size != nil {
		sess.Resize(input.Size)
	}
	sess.Info.TimeUpdated = model.NowUnix()

	writeJSON(w, http.StatusOK, sess.Info)
}

func HandlePTYRemove(w http.ResponseWriter, r *http.Request) {
	ptyID := getPathParam(r, "ptyID")
	service.GlobalPTYs.Remove(ptyID)
	w.WriteHeader(http.StatusNoContent)
}

func HandlePTYConnectToken(w http.ResponseWriter, r *http.Request) {
	ptyID := getPathParam(r, "ptyID")
	sess := service.GlobalPTYs.Get(ptyID)
	if sess == nil {
		writeError(w, 404, "PTY not found")
		return
	}

	token := service.PTYConnectToken(ptyID)
	writeJSON(w, http.StatusOK, map[string]any{
		"token": token,
		"url":   "ws://localhost:8090/api/pty/" + ptyID + "/connect",
	})
}

func HandlePTYConnect(w http.ResponseWriter, r *http.Request) {
	ptyID := getPathParam(r, "ptyID")
	sess := service.GlobalPTYs.Get(ptyID)
	if sess == nil {
		writeError(w, 404, "PTY not found")
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("PTY WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close()
	defer conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))

	// Subscribe to PTY output
	outputCh := sess.Subscribe()
	errCh := make(chan error, 1)

	// Write PTY output to WebSocket
	go func() {
		for data := range outputCh {
			if err := conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
				errCh <- err
				return
			}
		}
	}()

	// Read WebSocket messages -> write to PTY
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			break
		}

		// Handle resize control messages
		msgStr := string(message)
		if strings.HasPrefix(msgStr, "resize:") {
			var size service.Size
			json.Unmarshal([]byte(msgStr[7:]), &size)
			sess.Resize(&size)
			continue
		}

		// Write input to PTY
		sess.Write(message)
	}
}

// ── Project Copy ────────────────────────────────────

func HandleProjectCopyCreate(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusCreated, map[string]string{"status": "copying"})
}

func HandleProjectCopyRemove(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

func HandleProjectCopyRefresh(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "refreshed"})
}
