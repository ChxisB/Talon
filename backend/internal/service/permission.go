package service

import (
	"database/sql"
	"encoding/json"
	"sync"
	"time"

	"github.com/talon/backend/internal/db"
	"github.com/google/uuid"
)

// ── Types ───────────────────────────────────────────

type PermissionRequest struct {
	ID        string   `json:"id"`
	SessionID string   `json:"sessionID"`
	Action    string   `json:"action"`
	Resources []string `json:"resources"`
	Save      []string `json:"save,omitempty"`
	Source    *Source  `json:"source,omitempty"`
	TimeCreated int64  `json:"timeCreated"`
}

type Source struct {
	Type      string `json:"type"`
	MessageID string `json:"messageID"`
	CallID    string `json:"callID"`
}

type PermissionReply struct {
	RequestID string `json:"requestID"`
	Reply     string `json:"reply"` // "once", "always", "reject"
	Message   string `json:"message,omitempty"`
}

type PermissionSaved struct {
	ID        string `json:"id"`
	ProjectID string `json:"projectID"`
	Action    string `json:"action"`
	Resource  string `json:"resource"`
	TimeCreated int64 `json:"timeCreated"`
}

// ── Store ───────────────────────────────────────────

type PermissionStore struct {
	mu      sync.RWMutex
	pending map[string]*PermissionRequest
}

var GlobalPermissions = &PermissionStore{
	pending: make(map[string]*PermissionRequest),
}

func (s *PermissionStore) List(location string) []*PermissionRequest {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []*PermissionRequest
	for _, p := range s.pending {
		result = append(result, p)
	}
	return result
}

func (s *PermissionStore) ListBySession(sessionID string) []*PermissionRequest {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []*PermissionRequest
	for _, p := range s.pending {
		if p.SessionID == sessionID {
			result = append(result, p)
		}
	}
	return result
}

func (s *PermissionStore) Add(req *PermissionRequest) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pending[req.ID] = req
}

func (s *PermissionStore) Get(id string) *PermissionRequest {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.pending[id]
}

func (s *PermissionStore) Remove(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.pending, id)
}

// ── Saved Permissions (SQLite) ──────────────────────

func ListSavedPermissions(projectID string) ([]PermissionSaved, error) {
	var rows *sql.Rows
	var err error
	if projectID != "" {
		rows, err = db.DB.Query("SELECT id, project_id, action, resource, time_created FROM permission WHERE project_id = ?", projectID)
	} else {
		rows, err = db.DB.Query("SELECT id, project_id, action, resource, time_created FROM permission")
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []PermissionSaved
	for rows.Next() {
		var p PermissionSaved
		if err := rows.Scan(&p.ID, &p.ProjectID, &p.Action, &p.Resource, &p.TimeCreated); err == nil {
			result = append(result, p)
		}
	}
	return result, nil
}

func AddSavedPermission(projectID, action, resource string) (*PermissionSaved, error) {
	id := "psv_" + uuid.New().String()[:12]
	now := time.Now().UnixMilli()
	_, err := db.DB.Exec(
		"INSERT OR IGNORE INTO permission (id, project_id, action, resource, time_created, time_updated) VALUES (?, ?, ?, ?, ?, ?)",
		id, projectID, action, resource, now, now,
	)
	if err != nil {
		return nil, err
	}
	return &PermissionSaved{ID: id, ProjectID: projectID, Action: action, Resource: resource, TimeCreated: now}, nil
}

func RemoveSavedPermission(id string) error {
	_, err := db.DB.Exec("DELETE FROM permission WHERE id = ?", id)
	return err
}

// ── Helper ──────────────────────────────────────────

func NewPermissionID() string {
	return "per_" + uuid.New().String()[:12]
}

func MarshalJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
