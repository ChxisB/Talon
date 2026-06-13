package openai

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/ChxisB/spectre-proxy/agent/internal/protocol"
)

// ─── OpenAI API types ───────────────────────────────────────────────

type ChatRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Stream      bool          `json:"stream"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Temperature *float64      `json:"temperature,omitempty"`
	TopP        *float64      `json:"top_p,omitempty"`
	Stop        []string      `json:"stop,omitempty"`
	Tools       []ChatToolDef `json:"tools,omitempty"`
}

type ChatMessage struct {
	Role             string          `json:"role"`
	Content          json.RawMessage `json:"content"`
	ToolCalls        []ToolCall      `json:"tool_calls,omitempty"`
	ToolCallID       string          `json:"tool_call_id,omitempty"`
	ReasoningContent string          `json:"reasoning_content,omitempty"`
}

type ToolCall struct {
	Index    int              `json:"index"`
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Function ToolCallFunction `json:"function"`
}

type ToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type ChatToolDef struct {
	Type     string          `json:"type"`
	Function ChatFunctionDef `json:"function"`
}

type ChatFunctionDef struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Parameters  any    `json:"parameters"`
	// Strict is OpenAI-specific. Omit for other providers (DeepSeek, etc.)
	Strict *bool `json:"strict,omitempty"`
}

type OpenAIUsage struct {
	InputTokens  int `json:"prompt_tokens"`
	OutputTokens int `json:"completion_tokens"`
}

func (u *OpenAIUsage) ToProtocol() *protocol.Usage {
	if u == nil {
		return nil
	}
	return &protocol.Usage{InputTokens: u.InputTokens, OutputTokens: u.OutputTokens}
}

type ChatStreamChunk struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Model   string         `json:"model"`
	Choices []StreamChoice `json:"choices"`
	Usage   *OpenAIUsage   `json:"usage,omitempty"`
}

type StreamChoice struct {
	Index        int         `json:"index"`
	Delta        StreamDelta `json:"delta"`
	FinishReason string      `json:"finish_reason"`
}

type StreamDelta struct {
	Role             string     `json:"role,omitempty"`
	Content          string     `json:"content,omitempty"`
	ReasoningContent string     `json:"reasoning_content,omitempty"`
	ToolCalls        []ToolCall `json:"tool_calls,omitempty"`
}

// ─── Conversion functions ───────────────────────────────────────────

func anthropicToOpenAI(anthropicReq *protocol.MessagesRequest, systemPrompt string, providerName string) *ChatRequest {
	chatReq := &ChatRequest{
		Model:       anthropicReq.Model,
		Stream:      true,
		Temperature: anthropicReq.Temperature,
		TopP:        anthropicReq.TopP,
	}
	if anthropicReq.MaxTokens > 0 {
		chatReq.MaxTokens = anthropicReq.MaxTokens
	} else {
		chatReq.MaxTokens = 8192
	}
	if chatReq.MaxTokens > 131072 {
		chatReq.MaxTokens = 131072
	}
	if len(anthropicReq.StopSeq) > 0 {
		chatReq.Stop = anthropicReq.StopSeq
	}
	// Convert tools with provider-specific handling
	for _, t := range anthropicReq.Tools {
		params := convertToolSchema(t.InputSchema, providerName)
		// Log each tool's schema for debugging
		if schemaJSON, err := json.Marshal(params); err == nil {
			log.Printf("[tools] provider=%s tool=%s schema=%s", providerName, t.Name, truncate(string(schemaJSON), 300))
		}
		// Only set strict for OpenAI native API
		var strict *bool
		if providerName == "openai" {
			v := true
			strict = &v
		}
		chatReq.Tools = append(chatReq.Tools, ChatToolDef{
			Type: "function",
			Function: ChatFunctionDef{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  params,
				Strict:      strict,
			},
		})
	}
	// Build messages
	var msgs []ChatMessage
	if systemPrompt != "" {
		msgs = append(msgs, ChatMessage{
			Role:    "system",
			Content: mustJSON(systemPrompt),
		})
	}
	for _, msg := range anthropicReq.Messages {
		msgs = append(msgs, convertMessages(msg)...)
	}
	chatReq.Messages = msgs
	return chatReq
}

// convertMessages converts a single Anthropic message into one or more OpenAI messages.
func convertMessages(msg protocol.Message) []ChatMessage {
	var textParts []string
	var thinkingParts []string
	var toolResults []ChatMessage
	var toolCalls []ToolCall

	blocks, _ := msg.ContentBlocks()
	for _, b := range blocks {
		switch b.Type {
		case "text":
			if b.Text != "" {
				textParts = append(textParts, b.Text)
			}
		case "thinking":
			if b.Thinking != "" {
				thinkingParts = append(thinkingParts, b.Thinking)
			}
		case "tool_result":
			contentStr := ""
			if b.Content != nil {
				switch v := b.Content.(type) {
				case string:
					contentStr = v
				default:
					if j, err := json.Marshal(v); err == nil {
						contentStr = string(j)
					}
				}
			}
			toolResults = append(toolResults, ChatMessage{
				Role:       "tool",
				Content:    mustJSON(contentStr),
				ToolCallID: b.ToolUseID,
			})
		case "tool_use":
			args := ""
			if b.Input != nil {
				if j, err := json.Marshal(b.Input); err == nil {
					args = string(j)
				}
			}
			toolCalls = append(toolCalls, ToolCall{
				ID:   b.ID,
				Type: "function",
				Function: ToolCallFunction{
					Name:      b.Name,
					Arguments: args,
				},
			})
		}
	}

	reasoningContent := strings.Join(thinkingParts, "\n")
	role := string(msg.Role)

	var result []ChatMessage

	// User message with tool results
	if len(toolResults) > 0 {
		userText := strings.Join(textParts, "\n")
		if userText != "" {
			result = append(result, ChatMessage{
				Role:    "user",
				Content: mustJSON(userText),
			})
		}
		result = append(result, toolResults...)
		return result
	}

	// Assistant message with tool calls
	if role == "assistant" && len(toolCalls) > 0 {
		chatMsg := ChatMessage{
			Role:      "assistant",
			ToolCalls: toolCalls,
		}
		if reasoningContent != "" {
			chatMsg.ReasoningContent = reasoningContent
		}
		if len(textParts) > 0 {
			chatMsg.Content = mustJSON(strings.Join(textParts, "\n"))
		} else {
			chatMsg.Content = json.RawMessage("null")
		}
		result = append(result, chatMsg)
		return result
	}

	// All other messages
	chatMsg := ChatMessage{
		Role:    role,
		Content: mustJSON(strings.Join(textParts, "\n")),
	}
	if role == "assistant" && reasoningContent != "" {
		chatMsg.ReasoningContent = reasoningContent
	}
	result = append(result, chatMsg)
	return result
}

func openAIToAnthropic(sse *SSEBuilder, chunk *ChatStreamChunk) []protocol.SSEEvent {
	var events []protocol.SSEEvent

	for _, choice := range chunk.Choices {
		delta := choice.Delta

		if delta.Role == "assistant" {
			events = append(events, sse.MessageStart(chunk.Model, chunk.Usage.ToProtocol()))
		}

		combined := delta.ReasoningContent + delta.Content
		if combined != "" {
			if !sse.hasContent {
				events = append(events, sse.ContentBlockStart())
			}
			events = append(events, sse.ContentBlockDelta(combined))
		}

		for _, tc := range delta.ToolCalls {
			events = append(events, sse.ContentBlockStart())
			events = append(events, sse.ContentBlockDelta(tc.Function.Arguments))
			events = append(events, sse.ContentBlockStop())
		}

		if choice.FinishReason != "" {
			stopReason := mapFinishReason(choice.FinishReason)
			events = append(events, sse.MessageDelta(stopReason, chunk.Usage.ToProtocol()))
			events = append(events, sse.MessageStop())
		}
	}

	return events
}

// ─── Helpers ────────────────────────────────────────────────────────

func extractSystemPrompt(req *protocol.MessagesRequest) string {
	if req.System == nil {
		return ""
	}
	var s string
	if err := json.Unmarshal(req.System, &s); err == nil {
		return s
	}
	var blocks []struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(req.System, &blocks); err == nil {
		var parts []string
		for _, b := range blocks {
			parts = append(parts, b.Text)
		}
		return strings.Join(parts, "\n")
	}
	return ""
}

func mapFinishReason(fr string) string {
	switch fr {
	case "stop":
		return "end_turn"
	case "length":
		return "max_tokens"
	case "tool_calls":
		return "tool_use"
	default:
		return fr
	}
}

func mustJSON(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		b, _ = json.Marshal(fmt.Sprintf("%v", v))
	}
	return json.RawMessage(b)
}

// convertToolSchema converts tool schema based on provider requirements
func convertToolSchema(schema any, provider string) any {
	if schema == nil {
		return map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		}
	}

	schemaMap, ok := schema.(map[string]any)
	if !ok {
		return schema
	}

	switch provider {
	case "opencode_go":
		// OpenCode Go: flatten anyOf, add additionalProperties:false, remove null schemas
		return normalizeOpenAISchema(schemaMap)

	case "opencode":
		// OpenCode Zen: same as OpenCode Go
		return normalizeOpenAISchema(schemaMap)

	case "openrouter":
		// OpenRouter: pass through as-is, it handles various models
		return schemaMap

	case "gemini":
		// Gemini: needs additionalProperties:false
		result := make(map[string]any)
		for k, v := range schemaMap {
			result[k] = v
		}
		result["type"] = "object"
		if _, hasAdditional := result["additionalProperties"]; !hasAdditional {
			result["additionalProperties"] = false
		}
		return result

	case "deepseek":
		// DeepSeek direct: similar to OpenCode
		return normalizeOpenAISchema(schemaMap)

	default:
		// Default: ensure type is object, add additionalProperties:false
		result := make(map[string]any)
		for k, v := range schemaMap {
			result[k] = v
		}
		result["type"] = "object"
		if _, hasAdditional := result["additionalProperties"]; !hasAdditional {
			result["additionalProperties"] = false
		}
		return result
	}
}

// normalizeOpenAISchema normalizes schema for OpenAI-compatible providers
// Flattens anyOf variants, adds additionalProperties:false, removes null schemas
func normalizeOpenAISchema(schema map[string]any) map[string]any {
	result := make(map[string]any)

	// Handle anyOf variants - flatten into single object
	if anyOf, ok := schema["anyOf"].([]any); ok {
		mergedProps := make(map[string]any)
		for _, variant := range anyOf {
			if variantMap, ok := variant.(map[string]any); ok {
				if props, ok := variantMap["properties"].(map[string]any); ok {
					for k, v := range props {
						mergedProps[k] = v
					}
				}
			}
		}
		result["type"] = "object"
		result["properties"] = mergedProps
		result["additionalProperties"] = false
		// Keep required from original
		if req, ok := schema["required"]; ok {
			result["required"] = req
		}
		normalized := removeNullSchemas(result)
		if normalizedMap, ok := normalized.(map[string]any); ok {
			return normalizedMap
		}
		return result
	}

	// No anyOf - just ensure type and additionalProperties
	for k, v := range schema {
		if k != "anyOf" {
			result[k] = v
		}
	}
	result["type"] = "object"
	if _, hasAdditional := result["additionalProperties"]; !hasAdditional {
		result["additionalProperties"] = false
	}

	normalized := removeNullSchemas(result)
	if normalizedMap, ok := normalized.(map[string]any); ok {
		return normalizedMap
	}
	return result
}

// removeNullSchemas recursively removes null types from anyOf arrays
func removeNullSchemas(value any) any {
	if arr, ok := value.([]any); ok {
		result := make([]any, 0, len(arr))
		for _, item := range arr {
			result = append(result, removeNullSchemas(item))
		}
		return result
	}
	if m, ok := value.(map[string]any); ok {
		fields := make(map[string]any)
		for k, v := range m {
			if k != "anyOf" {
				fields[k] = removeNullSchemas(v)
			}
		}
		// Handle anyOf - filter out null types
		if anyOf, ok := m["anyOf"].([]any); ok {
			variants := make([]any, 0, len(anyOf))
			for _, v := range anyOf {
				if vm, ok := v.(map[string]any); ok {
					if t, ok := vm["type"].(string); ok && t == "null" {
						continue
					}
				}
				variants = append(variants, removeNullSchemas(v))
			}
			if len(variants) == 1 {
				// Single variant - merge into fields
				if vm, ok := variants[0].(map[string]any); ok {
					for k, v := range vm {
						fields[k] = v
					}
				}
			} else if len(variants) > 1 {
				fields["anyOf"] = variants
			}
		}
		return fields
	}
	return value
}
