// Package frugal provides token optimization, waste detection, and
// context management for LLM interactions. Ported from the original
// token-optimizer project.
package frugal

// Finding represents a detected waste or optimization opportunity.
type Finding struct {
	Name           string  `json:"name"`
	Confidence     float64 `json:"confidence"`
	Evidence       string  `json:"evidence"`
	SavingsTokens  int     `json:"savings_tokens"`
	Suggestion     string  `json:"suggestion"`
	OccurrenceCount int    `json:"occurrence_count"`
}

// SessionData represents a session for waste detection analysis.
type SessionData struct {
	Turns             []TurnData `json:"turns"`
	TotalOutputTokens int        `json:"total_output_tokens"`
	TotalInputTokens  int        `json:"total_input_tokens"`
	JSONLPath         string     `json:"jsonl_path,omitempty"`
}

// TurnData represents a single user-assistant turn in a session.
type TurnData struct {
	UserText      string   `json:"user_text"`
	AssistantText string   `json:"assistant_text"`
	InputTokens   int      `json:"input_tokens"`
	OutputTokens  int      `json:"output_tokens"`
	ToolsUsed     []string `json:"tools_used"`
}

// Detector detects waste patterns in session data.
type Detector interface {
	// Name returns the detector's unique name.
	Name() string
	// Detect runs detection on session data and returns findings.
	Detect(data *SessionData) []Finding
}

// DetectorFunc is a convenience type for single-function detectors.
type DetectorFunc func(data *SessionData) []Finding

func (f DetectorFunc) Name() string { return "" }
func (f DetectorFunc) Detect(data *SessionData) []Finding { return f(data) }
