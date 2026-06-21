package handler

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/talon/backend/internal/db"
	"github.com/talon/backend/internal/model"
	"github.com/talon/backend/internal/service"
)

func HandleSessionList(w http.ResponseWriter, r *http.Request) {
	directory := parseLocation(r)
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, ok := parseInt(l); ok && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	query := `
		SELECT id, project_id, workspace_id, parent_id, slug, directory, path,
			title, version, share_url, summary_additions, summary_deletions,
			summary_files, summary_diffs, metadata, cost, tokens_input, tokens_output,
			tokens_reasoning, tokens_cache_read, tokens_cache_write, revert, permission,
			agent, model, time_created, time_updated, time_compacting, time_archived
		FROM session
		WHERE directory = ?
		ORDER BY time_created DESC
		LIMIT ?
	`
	rows, err := db.DB.Query(query, directory, limit)
	if err != nil {
		writeError(w, 500, "Failed to query sessions: "+err.Error())
		return
	}
	defer rows.Close()

	sessions := []model.Session{}
	for rows.Next() {
		var s model.Session
		var workspaceID, parentID, path, shareURL, summaryDiffs, metadata, revert, permission, agent, modelJSON sql.NullString
		var additions, deletions, files sql.NullInt64
		var timeCompacting, timeArchived sql.NullInt64

		err := rows.Scan(&s.ID, &s.ProjectID, &workspaceID, &parentID, &s.Slug, &s.Directory,
			&path, &s.Title, &s.Version, &shareURL, &additions, &deletions,
			&files, &summaryDiffs, &metadata, &s.Cost, &s.TokensInput, &s.TokensOutput,
			&s.TokensReasoning, &s.TokensCacheRead, &s.TokensCacheWrite, &revert, &permission,
			&agent, &modelJSON, &s.TimeCreated, &s.TimeUpdated, &timeCompacting, &timeArchived)
		if err != nil {
			continue
		}

		if workspaceID.Valid {
			s.WorkspaceID = &workspaceID.String
		}
		if parentID.Valid {
			s.ParentID = &parentID.String
		}
		if path.Valid {
			s.Path = &path.String
		}
		if shareURL.Valid {
			s.ShareURL = &shareURL.String
		}
		if additions.Valid {
			v := int(additions.Int64)
			s.Additions = &v
		}
		if deletions.Valid {
			v := int(deletions.Int64)
			s.Deletions = &v
		}
		if files.Valid {
			v := int(files.Int64)
			s.Files = &v
		}
		if summaryDiffs.Valid && summaryDiffs.String != "" {
			json.Unmarshal([]byte(summaryDiffs.String), &s.Diffs)
		}
		if timeCompacting.Valid {
			s.TimeCompacting = &timeCompacting.Int64
		}
		if timeArchived.Valid {
			s.TimeArchived = &timeArchived.Int64
		}
		if agent.Valid {
			s.Agent = &agent.String
		}
		if modelJSON.Valid && modelJSON.String != "" {
			var m model.ModelRef
			if json.Unmarshal([]byte(modelJSON.String), &m) == nil {
				s.Model = &m
			}
		}

		sessions = append(sessions, s)
	}

	if sessions == nil {
		sessions = []model.Session{}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"sessions": sessions,
	})
}

func HandleSessionGet(w http.ResponseWriter, r *http.Request) {
	sessionID := getPathParam(r, "sessionID")
	if sessionID == "" {
		writeError(w, 400, "sessionID required")
		return
	}

	query := `
		SELECT id, project_id, workspace_id, parent_id, slug, directory, path,
			title, version, share_url, summary_additions, summary_deletions,
			summary_files, summary_diffs, metadata, cost, tokens_input, tokens_output,
			tokens_reasoning, tokens_cache_read, tokens_cache_write, revert, permission,
			agent, model, time_created, time_updated, time_compacting, time_archived
		FROM session WHERE id = ?
	`
	var s model.Session
	var workspaceID, parentID, path, shareURL, summaryDiffs, metadata, revert, permission, agent, modelJSON sql.NullString
	var additions, deletions, files sql.NullInt64
	var timeCompacting, timeArchived sql.NullInt64

	err := db.DB.QueryRow(query, sessionID).Scan(&s.ID, &s.ProjectID, &workspaceID, &parentID,
		&s.Slug, &s.Directory, &path, &s.Title, &s.Version, &shareURL,
		&additions, &deletions, &files, &summaryDiffs, &metadata, &s.Cost,
		&s.TokensInput, &s.TokensOutput, &s.TokensReasoning, &s.TokensCacheRead,
		&s.TokensCacheWrite, &revert, &permission, &agent, &modelJSON,
		&s.TimeCreated, &s.TimeUpdated, &timeCompacting, &timeArchived)

	if err == sql.ErrNoRows {
		writeError(w, 404, "Session not found")
		return
	}
	if err != nil {
		writeError(w, 500, "Failed to get session: "+err.Error())
		return
	}

	if workspaceID.Valid {
		s.WorkspaceID = &workspaceID.String
	}
	if parentID.Valid {
		s.ParentID = &parentID.String
	}
	if path.Valid {
		s.Path = &path.String
	}
	if shareURL.Valid {
		s.ShareURL = &shareURL.String
	}
	if agent.Valid {
		s.Agent = &agent.String
	}
	if modelJSON.Valid && modelJSON.String != "" {
		var m model.ModelRef
		if json.Unmarshal([]byte(modelJSON.String), &m) == nil {
			s.Model = &m
		}
	}

	writeJSON(w, http.StatusOK, s)
}

func HandleSessionCreate(w http.ResponseWriter, r *http.Request) {
	var input struct {
		ID      string       `json:"id"`
		Agent   string       `json:"agent"`
		Model   model.ModelRef `json:"model"`
		Title   string       `json:"title"`
		Directory string     `json:"directory"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, 400, "Invalid request body")
		return
	}

	if input.ID == "" {
		writeError(w, 400, "id is required")
		return
	}
	if input.Directory == "" {
		writeError(w, 400, "directory is required")
		return
	}

	now := model.NowUnix()
	modelJSON, _ := json.Marshal(input.Model)

	_, err := db.DB.Exec(`
		INSERT INTO session (id, project_id, slug, directory, title, version, agent, model, time_created, time_updated)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, input.ID, "default", input.ID, input.Directory, input.Title, "1", input.Agent, string(modelJSON), now, now)

	if err != nil {
		writeError(w, 500, "Failed to create session: "+err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"id": input.ID})
}

func HandleSessionMessages(w http.ResponseWriter, r *http.Request) {
	sessionID := getPathParam(r, "sessionID")
	if sessionID == "" {
		writeError(w, 400, "sessionID required")
		return
	}

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, ok := parseInt(l); ok && parsed > 0 && parsed <= 200 {
			limit = parsed
		}
	}
	order := "ASC"
	if r.URL.Query().Get("order") == "desc" {
		order = "DESC"
	}

	query := `
		SELECT id, session_id, type, seq, time_created, time_updated, data
		FROM session_message
		WHERE session_id = ?
		ORDER BY seq ` + order + `
		LIMIT ?
	`
	rows, err := db.DB.Query(query, sessionID, limit)
	if err != nil {
		writeError(w, 500, "Failed to query messages: "+err.Error())
		return
	}
	defer rows.Close()

	messages := []model.SessionMessage{}
	for rows.Next() {
		var m model.SessionMessage
		if err := rows.Scan(&m.ID, &m.SessionID, &m.Type, &m.Seq, &m.TimeCreated, &m.TimeUpdated, &m.Data); err == nil {
			messages = append(messages, m)
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"messages": messages,
	})
}

func HandleSessionPrompt(w http.ResponseWriter, r *http.Request) {
	sessionID := getPathParam(r, "sessionID")
	if sessionID == "" {
		writeError(w, 400, "sessionID required")
		return
	}

	var input struct {
		Prompt string `json:"prompt"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, 400, "Invalid request body")
		return
	}

	// Verify session exists
	var exists bool
	err := db.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM session WHERE id = ?)", sessionID).Scan(&exists)
	if err != nil || !exists {
		writeError(w, 404, "Session not found")
		return
	}

	// Create session_input for durable prompt admission
	now := model.NowUnix()
	_, err = db.DB.Exec(`
		INSERT INTO session_input (id, session_id, prompt, delivery, admitted_seq, time_created)
		VALUES (?, ?, ?, 'steer', 1, ?)
	`, sessionID+"-input", sessionID, input.Prompt, now)

	if err != nil {
		writeError(w, 500, "Failed to admit prompt: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status": "admitted",
		"id":     sessionID,
	})
}

func HandleSessionWait(w http.ResponseWriter, r *http.Request) {
	sessionID := getPathParam(r, "sessionID")
	if sessionID == "" {
		writeError(w, 400, "sessionID required")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "idle",
		"id":     sessionID,
	})
}

func HandleSessionContext(w http.ResponseWriter, r *http.Request) {
	sessionID := getPathParam(r, "sessionID")
	if sessionID == "" {
		writeError(w, 400, "sessionID required")
		return
	}

	rows, err := db.DB.Query(`
		SELECT id, session_id, type, seq, time_created, time_updated, data
		FROM session_message
		WHERE session_id = ?
		ORDER BY seq ASC
	`, sessionID)
	if err != nil {
		writeError(w, 500, "Failed to get context: "+err.Error())
		return
	}
	defer rows.Close()

	messages := []model.SessionMessage{}
	for rows.Next() {
		var m model.SessionMessage
		if err := rows.Scan(&m.ID, &m.SessionID, &m.Type, &m.Seq, &m.TimeCreated, &m.TimeUpdated, &m.Data); err == nil {
			messages = append(messages, m)
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"context": messages,
	})
}

func HandleSessionCompact(w http.ResponseWriter, r *http.Request) {
	sessionID := getPathParam(r, "sessionID")
	if sessionID == "" {
		writeError(w, 400, "sessionID required")
		return
	}
	now := model.NowUnix()
	db.DB.Exec("UPDATE session SET time_compacting = ? WHERE id = ?", now, sessionID)
	writeJSON(w, http.StatusOK, map[string]string{"status": "compacted"})
}

func HandleSessionPermissionList(w http.ResponseWriter, r *http.Request) {
	sessionID := getPathParam(r, "sessionID")
	requests := service.GlobalPermissions.ListBySession(sessionID)
	writeJSON(w, http.StatusOK, map[string]any{"requests": requests})
}

func HandleSessionPermissionReply(w http.ResponseWriter, r *http.Request) {
	sessionID := getPathParam(r, "sessionID")
	requestID := getPathParam(r, "requestID")

	var input struct {
		Reply   string `json:"reply"`
		Message string `json:"message,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, 400, "Invalid request body")
		return
	}

	req := service.GlobalPermissions.Get(requestID)
	if req == nil || req.SessionID != sessionID {
		writeError(w, 404, "Permission request not found")
		return
	}

	// Handle "always" — save to database
	if input.Reply == "always" && len(req.Resources) > 0 {
		for _, rsrc := range req.Resources {
			service.AddSavedPermission("default", req.Action, rsrc)
		}
	}

	// Cascade: reject all same-session pending requests on rejection
	if input.Reply == "reject" {
		for _, p := range service.GlobalPermissions.ListBySession(sessionID) {
			service.GlobalPermissions.Remove(p.ID)
		}
	}

	service.GlobalPermissions.Remove(requestID)
	writeJSON(w, http.StatusOK, map[string]string{"status": "replied", "reply": input.Reply})
}

func HandleSessionQuestionList(w http.ResponseWriter, r *http.Request) {
	sessionID := getPathParam(r, "sessionID")
	questions := service.GlobalQuestions.ListBySession(sessionID)
	writeJSON(w, http.StatusOK, map[string]any{"requests": questions})
}

func HandleSessionQuestionReply(w http.ResponseWriter, r *http.Request) {
	sessionID := getPathParam(r, "sessionID")
	requestID := getPathParam(r, "requestID")

	var input struct {
		Answers [][]string `json:"answers"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, 400, "Invalid request body")
		return
	}

	q := service.GlobalQuestions.Get(requestID)
	if q == nil || q.SessionID != sessionID {
		writeError(w, 404, "Question request not found")
		return
	}

	service.GlobalQuestions.Remove(requestID)
	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "replied",
		"answers": input.Answers,
	})
}

func HandleSessionQuestionReject(w http.ResponseWriter, r *http.Request) {
	sessionID := getPathParam(r, "sessionID")
	requestID := getPathParam(r, "requestID")

	q := service.GlobalQuestions.Get(requestID)
	if q == nil || q.SessionID != sessionID {
		writeError(w, 404, "Question request not found")
		return
	}

	service.GlobalQuestions.Remove(requestID)
	writeJSON(w, http.StatusOK, map[string]string{"status": "rejected"})
}

func parseInt(s string) (int, bool) {
	var n int
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, false
		}
		n = n*10 + int(c-'0')
	}
	return n, true
}
