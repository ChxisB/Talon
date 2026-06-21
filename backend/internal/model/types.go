package model

import (
	"time"
	"github.com/google/uuid"
)

// ── Location ────────────────────────────────────────

type Location struct {
	Directory string `json:"directory"`
	Workspace string `json:"workspace,omitempty"`
	ProjectID string `json:"projectID,omitempty"`
}

// ── Session ─────────────────────────────────────────

type ModelRef struct {
	ID         string `json:"id"`
	ProviderID string `json:"providerID"`
	Variant    string `json:"variant,omitempty"`
}

type SummaryDiff struct {
	Path    string `json:"path"`
	Added   int    `json:"added"`
	Removed int    `json:"removed"`
}

type Session struct {
	ID               string       `json:"id"`
	ProjectID        string       `json:"projectID"`
	WorkspaceID      *string      `json:"workspaceID,omitempty"`
	ParentID         *string      `json:"parentID,omitempty"`
	Slug             string       `json:"slug"`
	Directory        string       `json:"directory"`
	Path             *string      `json:"path,omitempty"`
	Title            string       `json:"title"`
	Version          string       `json:"version"`
	ShareURL         *string      `json:"shareURL,omitempty"`
	Additions        *int         `json:"additions,omitempty"`
	Deletions        *int         `json:"deletions,omitempty"`
	Files            *int         `json:"files,omitempty"`
	Diffs            []SummaryDiff `json:"diffs,omitempty"`
	Metadata         *string      `json:"metadata,omitempty"`
	Cost             float64      `json:"cost"`
	TokensInput      int64        `json:"tokensInput"`
	TokensOutput     int64        `json:"tokensOutput"`
	TokensReasoning  int64        `json:"tokensReasoning"`
	TokensCacheRead  int64        `json:"tokensCacheRead"`
	TokensCacheWrite int64        `json:"tokensCacheWrite"`
	Revert           *string      `json:"revert,omitempty"`
	Permission       *string      `json:"permission,omitempty"`
	Agent            *string      `json:"agent,omitempty"`
	Model            *ModelRef    `json:"model,omitempty"`
	TimeCreated      int64        `json:"timeCreated"`
	TimeUpdated      int64        `json:"timeUpdated"`
	TimeCompacting   *int64       `json:"timeCompacting,omitempty"`
	TimeArchived     *int64       `json:"timeArchived,omitempty"`
}

type SessionListParams struct {
	Cursor    string `json:"cursor,omitempty"`
	Limit     int    `json:"limit,omitempty"`
	Direction string `json:"direction,omitempty"`
	Search    string `json:"search,omitempty"`
}

// ── Session Message (V2) ────────────────────────────

type SessionMessage struct {
	ID          string `json:"id"`
	SessionID   string `json:"sessionID"`
	Type        string `json:"type"`
	Seq         int    `json:"seq"`
	TimeCreated int64  `json:"timeCreated"`
	TimeUpdated int64  `json:"timeUpdated"`
	Data        string `json:"data"`
}

// ── Provider ────────────────────────────────────────

type Provider struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Enabled     bool   `json:"enabled"`
}

// ── Model ───────────────────────────────────────────

type AIModel struct {
	ID         string `json:"id"`
	ProviderID string `json:"providerID"`
	Name       string `json:"name"`
	Enabled    bool   `json:"enabled"`
}

// ── Credential ──────────────────────────────────────

type Credential struct {
	ID            string `json:"id"`
	IntegrationID string `json:"integrationID,omitempty"`
	Label         string `json:"label"`
	Active        bool   `json:"active"`
	TimeCreated   int64  `json:"timeCreated"`
	TimeUpdated   int64  `json:"timeUpdated"`
}

// ── Permission ──────────────────────────────────────

type Permission struct {
	ID          string `json:"id"`
	ProjectID   string `json:"projectID"`
	Action      string `json:"action"`
	Resource    string `json:"resource"`
	TimeCreated int64  `json:"timeCreated"`
	TimeUpdated int64  `json:"timeUpdated"`
}

// ── Agent ───────────────────────────────────────────

type Agent struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// ── Command ─────────────────────────────────────────

type Command struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// ── Skill ───────────────────────────────────────────

type Skill struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// ── Error ───────────────────────────────────────────

type APIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

func (e APIError) Error() string {
	return e.Message
}

func NewAPIError(code int, msg string) APIError {
	return APIError{Code: code, Message: msg}
}

func NewNotFoundError(resource string) APIError {
	return APIError{Code: 404, Message: resource + " not found"}
}

func NewInvalidInputError(msg string) APIError {
	return APIError{Code: 400, Message: msg}
}

// ── Helpers ─────────────────────────────────────────

func NowUnix() int64 {
	return time.Now().UnixMilli()
}

func NewUUID() string {
	return uuid.New().String()
}
