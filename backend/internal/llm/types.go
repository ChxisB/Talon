package llm

// ── Message Types ───────────────────────────────────

type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

type ContentPart struct {
	Type     string       `json:"type"` // "text", "tool_use", "tool_result"
	Text     string       `json:"text,omitempty"`
	ToolUse  *ToolCall    `json:"tool_use,omitempty"`
	ToolResult *ToolResult `json:"tool_result,omitempty"`
}

type Message struct {
	Role       Role         `json:"role"`
	Content    string       `json:"content,omitempty"`
	Parts      []ContentPart `json:"parts,omitempty"`
	ToolCalls  []ToolCall   `json:"tool_calls,omitempty"`
	ToolCallID string       `json:"tool_call_id,omitempty"`
	Name       string       `json:"name,omitempty"`
}

type ToolCall struct {
	ID       string   `json:"id"`
	Type     string   `json:"type"` // "function"
	Function Function `json:"function"`
}

type Function struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON string
}

type ToolResult struct {
	ToolCallID string `json:"tool_call_id"`
	Content    string `json:"content"`
	IsError    bool   `json:"is_error,omitempty"`
}

// ── Tool Definition ─────────────────────────────────

type ToolDefinition struct {
	Type       string       `json:"type"` // "function"
	Function   ToolFunction `json:"function"`
}

type ToolFunction struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  any         `json:"parameters"` // JSON Schema object
}

// ── Completion Request/Response ─────────────────────

type CompletionRequest struct {
	Model       string           `json:"model"`
	Messages    []Message        `json:"messages"`
	System      string           `json:"system,omitempty"`
	MaxTokens   int              `json:"max_tokens,omitempty"`
	Temperature float64          `json:"temperature,omitempty"`
	Tools       []ToolDefinition `json:"tools,omitempty"`
	Stream      bool             `json:"stream,omitempty"`
}

type CompletionResponse struct {
	Content      string     `json:"content"`
	ToolCalls    []ToolCall `json:"tool_calls,omitempty"`
	Usage        *Usage     `json:"usage,omitempty"`
	FinishReason string     `json:"finish_reason,omitempty"`
}

type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// ── Stream Events ───────────────────────────────────

type StreamEvent struct {
	Type        StreamEventType `json:"type"`
	Delta       string          `json:"delta,omitempty"`
	ToolCalls   []ToolCall      `json:"tool_calls,omitempty"`
	Usage       *Usage          `json:"usage,omitempty"`
	FinishReason string         `json:"finish_reason,omitempty"`
	Error       string          `json:"error,omitempty"`
}

type StreamEventType string

const (
	StreamEventText    StreamEventType = "text"
	StreamEventToolCall StreamEventType = "tool_call"
	StreamEventDone    StreamEventType = "done"
	StreamEventError   StreamEventType = "error"
)

// ── Provider Interface ──────────────────────────────

type Provider interface {
	Name() string
	Complete(req CompletionRequest) (*CompletionResponse, error)
	CompleteStream(req CompletionRequest, events chan<- StreamEvent) error
}

// ── Configuration ──────────────────────────────────

type ProviderConfig struct {
	APIKey  string
	BaseURL string
	Model   string
}
