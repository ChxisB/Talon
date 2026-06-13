// Package protocol implements the Anthropic Messages API types and SSE streaming
// used by the Spectre Proxy proxy.
package protocol

import (
	"encoding/json"
	"fmt"
)

// ─── Roles ──────────────────────────────────────────────────────────

type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleSystem    Role = "system"
)

// ─── Content Blocks ─────────────────────────────────────────────────

type ContentBlock struct {
	Type       string         `json:"type"`
	Text       string         `json:"text,omitempty"`
	ID         string         `json:"id,omitempty"`
	Name       string         `json:"name,omitempty"`
	Input      map[string]any `json:"input,omitempty"`
	ToolUseID  string         `json:"tool_use_id,omitempty"`
	Content    any            `json:"content,omitempty"`
	Thinking   string         `json:"thinking,omitempty"`
	Signature  string         `json:"signature,omitempty"`
	Source     map[string]any `json:"source,omitempty"`
	Data       string         `json:"data,omitempty"`
}

// ─── Messages ───────────────────────────────────────────────────────

type Message struct {
	Role    Role            `json:"role"`
	Content json.RawMessage `json:"content"`
}

// ContentBlocks parses the Content field which can be either a string
// or an array of content blocks (Anthropic API compatibility).
func (m *Message) ContentBlocks() ([]ContentBlock, error) {
	if m.Content == nil {
		return nil, nil
	}
	// Try array first
	var blocks []ContentBlock
	if err := json.Unmarshal(m.Content, &blocks); err == nil {
		return blocks, nil
	}
	// Fall back to string
	var text string
	if err := json.Unmarshal(m.Content, &text); err != nil {
		return nil, fmt.Errorf("content must be string or array, got %s", string(m.Content))
	}
	return []ContentBlock{{Type: "text", Text: text}}, nil
}

// ─── Request ────────────────────────────────────────────────────────

type MessagesRequest struct {
	Model       string          `json:"model"`
	Messages    []Message       `json:"messages"`
	System      json.RawMessage `json:"system,omitempty"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Stream      bool            `json:"stream"`
	Temperature *float64        `json:"temperature,omitempty"`
	TopP        *float64        `json:"top_p,omitempty"`
	TopK        *int            `json:"top_k,omitempty"`
	StopSeq     []string        `json:"stop_sequences,omitempty"`
	Tools       []ToolDef       `json:"tools,omitempty"`
	Thinking    *ThinkingConfig `json:"thinking,omitempty"`
	Metadata    map[string]any  `json:"metadata,omitempty"`
}

type ThinkingConfig struct {
	Type         string `json:"type"`
	BudgetTokens int    `json:"budget_tokens,omitempty"`
}

type ToolDef struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema any         `json:"input_schema"`
}

// ─── Response ───────────────────────────────────────────────────────

type MessagesResponse struct {
	ID           string           `json:"id"`
	Model        string           `json:"model"`
	Role         string           `json:"role"`
	Content      []ContentBlock   `json:"content"`
	StopReason   string           `json:"stop_reason"`
	StopSequence string           `json:"stop_sequence,omitempty"`
	Usage        Usage            `json:"usage"`
	Type         string           `json:"type"`
}

type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type TokenCountRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	System   any       `json:"system,omitempty"`
	Tools    []ToolDef `json:"tools,omitempty"`
}

type TokenCountResponse struct {
	InputTokens int `json:"input_tokens"`
}

// ─── Model Listing ──────────────────────────────────────────────────

type ModelResponse struct {
	ID           string `json:"id"`
	DisplayName  string `json:"display_name"`
	CreatedAt    string `json:"created_at"`
}

type ModelsListResponse struct {
	Data     []ModelResponse `json:"data"`
	FirstID  string          `json:"first_id,omitempty"`
	LastID   string          `json:"last_id,omitempty"`
	HasMore  bool            `json:"has_more"`
}

// ─── SSE Events ─────────────────────────────────────────────────────

type SSEEvent struct {
	Type string
	Data []byte
}

const (
	SSEMessageStart        = "message_start"
	SSEContentBlockStart   = "content_block_start"
	SSEPing                = "ping"
	SSEContentBlockDelta   = "content_block_delta"
	SSEContentBlockStop    = "content_block_stop"
	SSEMessageDelta        = "message_delta"
	SSEMessageStop         = "message_stop"
	SSEError               = "error"
)

type ContentBlockDelta struct {
	Index int `json:"index"`
	Delta struct {
		Type        string `json:"type"`
		Text        string `json:"text,omitempty"`
		Thinking    string `json:"thinking,omitempty"`
		PartialJSON string `json:"partial_json,omitempty"`
	} `json:"delta"`
}

type MessageDelta struct {
	Delta struct {
		StopReason   string `json:"stop_reason"`
		StopSequence string `json:"stop_sequence,omitempty"`
	} `json:"delta"`
	Usage struct {
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}
