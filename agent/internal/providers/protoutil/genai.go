// Package protoutil — Google GenAI API translation utilities.
//
// Translates between Google GenAI API wire format and the internal
// MessagesRequest format, and between Anthropic SSE events and GenAI SSE chunks.
package protoutil

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ChxisB/spectre-proxy/agent/internal/protocol"
	"github.com/google/uuid"
)

// ─── Request Types ─────────────────────────────────────────────────────

// GenAIRequest is a Google GenAI API generateContent request.
type GenAIRequest struct {
	Contents         []GenAIContent      `json:"contents"`
	SystemInstruction *GenAIPartContainer `json:"systemInstruction,omitempty"`
	Tools            []GenAITool         `json:"tools,omitempty"`
	GenerationConfig *GenAIConfig        `json:"generationConfig,omitempty"`
}

type GenAIContent struct {
	Role  string     `json:"role"`
	Parts []GenAIPart `json:"parts"`
}

type GenAIPartContainer struct {
	Parts []GenAIPart `json:"parts"`
}

type GenAIPart struct {
	Text             string             `json:"text,omitempty"`
	FunctionCall     *GenAIFunctionCall  `json:"functionCall,omitempty"`
	FunctionResponse *GenAIFuncResponse  `json:"functionResponse,omitempty"`
}

type GenAIFunctionCall struct {
	Name string         `json:"name"`
	Args map[string]any `json:"args"`
}

type GenAIFuncResponse struct {
	Name     string `json:"name"`
	Response any    `json:"response"`
}

type GenAITool struct {
	FunctionDeclarations []GenAIFunctionDecl `json:"functionDeclarations,omitempty"`
}

type GenAIFunctionDecl struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Parameters  any    `json:"parameters,omitempty"`
}

type GenAIConfig struct {
	Temperature    *float64 `json:"temperature,omitempty"`
	TopP           *float64 `json:"topP,omitempty"`
	TopK           *int     `json:"topK,omitempty"`
	MaxOutputTokens int     `json:"maxOutputTokens,omitempty"`
}

// ─── Request Parsing ──────────────────────────────────────────────────

// ParseGenAIRequest converts a Google GenAI API JSON body into an internal
// MessagesRequest. The model name is passed separately (extracted from URL).
func ParseGenAIRequest(raw json.RawMessage, model string) (*protocol.MessagesRequest, error) {
	var req GenAIRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		return nil, fmt.Errorf("parse genai request: %w", err)
	}

	mr := &protocol.MessagesRequest{Model: model}

	if req.GenerationConfig != nil {
		mr.MaxTokens = req.GenerationConfig.MaxOutputTokens
		mr.Temperature = req.GenerationConfig.Temperature
		mr.TopP = req.GenerationConfig.TopP
		if req.GenerationConfig.TopK != nil {
			mr.TopK = req.GenerationConfig.TopK
		}
	}

	// System instruction
	if req.SystemInstruction != nil && len(req.SystemInstruction.Parts) > 0 {
		var sysText strings.Builder
		for _, p := range req.SystemInstruction.Parts {
			sysText.WriteString(p.Text)
		}
		if sysText.Len() > 0 {
			sysBytes, _ := json.Marshal([]protocol.ContentBlock{{Type: "text", Text: sysText.String()}})
			mr.System = sysBytes
		}
	}

	// Tools → functionDeclarations
	if len(req.Tools) > 0 {
		tools := make([]protocol.ToolDef, 0)
		for _, tool := range req.Tools {
			for _, fd := range tool.FunctionDeclarations {
				schema := fd.Parameters
				if schema == nil {
					schema = map[string]any{"type": "object", "properties": map[string]any{}}
				}
				tools = append(tools, protocol.ToolDef{
					Name:        fd.Name,
					Description: fd.Description,
					InputSchema: schema,
				})
			}
		}
		mr.Tools = tools
	}

	// Contents → messages
	if len(req.Contents) > 0 {
		messages := make([]protocol.Message, 0, len(req.Contents))
		for _, c := range req.Contents {
			role := protocol.RoleUser
			switch c.Role {
			case "model":
				role = protocol.RoleAssistant
			case "function":
				role = protocol.RoleUser
			}

			msg := protocol.Message{Role: role}
			blocks := make([]protocol.ContentBlock, 0, len(c.Parts))

			for _, part := range c.Parts {
				if part.Text != "" {
					blocks = append(blocks, protocol.ContentBlock{Type: "text", Text: part.Text})
				} else if part.FunctionCall != nil {
					input := part.FunctionCall.Args
					if input == nil {
						input = map[string]any{}
					}
					blocks = append(blocks, protocol.ContentBlock{
						Type:  "tool_use",
						ID:    "call_" + uuid.New().String(),
						Name:  part.FunctionCall.Name,
						Input: input,
					})
				} else if part.FunctionResponse != nil {
					respContent := ""
					if part.FunctionResponse.Response != nil {
						if s, ok := part.FunctionResponse.Response.(string); ok {
							respContent = s
						} else {
							b, _ := json.Marshal(part.FunctionResponse.Response)
							respContent = string(b)
						}
					}
					blocks = append(blocks, protocol.ContentBlock{
						Type:      "tool_result",
						ToolUseID: "call_" + part.FunctionResponse.Name,
						Content:   respContent,
					})
				}
			}

			if len(blocks) > 0 {
				data, _ := json.Marshal(blocks)
				msg.Content = data
				messages = append(messages, msg)
			}
		}
		mr.Messages = messages
	}

	return mr, nil
}

// ─── SSE Translation ─────────────────────────────────────────────────

// TranslateAnthropicToGenAI converts an Anthropic-format SSE event channel into
// Google GenAI-format SSE byte chunks.
//
// Each []byte in the output channel is a complete SSE chunk:
//   data: {"candidates":[{"content":{"parts":[{"text":"..."}]}}]}\n\n
func TranslateAnthropicToGenAI(anthro <-chan protocol.SSEEvent) <-chan []byte {
	out := make(chan []byte, 128)
	go func() {
		defer close(out)

		var textPending strings.Builder
		var toolParts []GenAIPart
		hasContent := false

		for event := range anthro {
			if len(event.Data) == 0 {
				continue
			}
			var raw map[string]any
			if json.Unmarshal(event.Data, &raw) != nil {
				continue
			}

			switch event.Type {
			case "content_block_start":
				cb, _ := raw["content_block"].(map[string]any)
				bt, _ := cb["type"].(string)
				if bt == "tool_use" {
					name, _ := cb["name"].(string)
					args := map[string]any{}
					if input, ok := cb["input"].(map[string]any); ok {
						args = input
					}
					toolParts = append(toolParts, GenAIPart{
						FunctionCall: &GenAIFunctionCall{Name: name, Args: args},
					})
				}

			case "content_block_delta":
				delta, _ := raw["delta"].(map[string]any)
				dt, _ := delta["type"].(string)
				if dt == "text_delta" {
					if text, _ := delta["text"].(string); text != "" {
						textPending.WriteString(text)
					}
				}

			case "content_block_stop":
				// Flush accumulated text + tool calls as a chunk
				if textPending.Len() > 0 || len(toolParts) > 0 || !hasContent {
					parts := []GenAIPart{}
					if textPending.Len() > 0 {
						parts = append(parts, GenAIPart{Text: textPending.String()})
					}
					parts = append(parts, toolParts...)

					chunk := genAIChunk{
						Candidates: []genAICandidate{{
							Index:   0,
							Content: GenAIContent{Role: "model", Parts: parts},
						}},
					}
					data, _ := json.Marshal(chunk)
					out <- []byte(fmt.Sprintf("data: %s\n\n", string(data)))
					hasContent = true
					textPending.Reset()
					toolParts = nil
				}

			case "message_delta":
				// Flush remaining text
				if textPending.Len() > 0 {
					chunk := genAIChunk{
						Candidates: []genAICandidate{{
							Index:   0,
							Content: GenAIContent{Role: "model", Parts: []GenAIPart{{Text: textPending.String()}}},
						}},
					}
					data, _ := json.Marshal(chunk)
					out <- []byte(fmt.Sprintf("data: %s\n\n", string(data)))
					hasContent = true
					textPending.Reset()
				}

				// Emit usage if present
				if u, ok := raw["usage"].(map[string]any); ok {
					usage := map[string]any{}
					if ot, _ := u["output_tokens"].(float64); ot > 0 {
						usage["candidatesTokenCount"] = int(ot)
					}
					if len(usage) > 0 {
						chunk := map[string]any{
							"candidates":    []any{},
							"usageMetadata": usage,
						}
						data, _ := json.Marshal(chunk)
						out <- []byte(fmt.Sprintf("data: %s\n\n", string(data)))
					}
				}
			}
		}

		if !hasContent {
			empty := genAIChunk{
				Candidates: []genAICandidate{{
					Index:   0,
					Content: GenAIContent{Role: "model", Parts: []GenAIPart{{Text: ""}}},
				}},
			}
			data, _ := json.Marshal(empty)
			out <- []byte(fmt.Sprintf("data: %s\n\n", string(data)))
		}

		out <- []byte("data: [DONE]\n\n")
	}()
	return out
}

// BuildGenAIResponse collects all Anthropic SSE events into a GenAI response object.
func BuildGenAIResponse(anthro <-chan protocol.SSEEvent) *GenAIResponse {
	resp := &GenAIResponse{
		Candidates:    []genAICandidate{},
		UsageMetadata: &GenAIUsage{},
	}

	var textBuf strings.Builder
	var toolParts []GenAIPart

	for event := range anthro {
		if len(event.Data) == 0 {
			continue
		}
		var raw map[string]any
		if json.Unmarshal(event.Data, &raw) != nil {
			continue
		}

		switch event.Type {
		case "message_start":
			if msg, ok := raw["message"].(map[string]any); ok {
				if u, ok := msg["usage"].(map[string]any); ok {
					if it, _ := u["input_tokens"].(float64); it > 0 {
						resp.UsageMetadata.PromptTokenCount = int(it)
					}
				}
			}

		case "content_block_start":
			if cb, ok := raw["content_block"].(map[string]any); ok {
				if bt, _ := cb["type"].(string); bt == "tool_use" {
					name, _ := cb["name"].(string)
					args := map[string]any{}
					if input, ok := cb["input"].(map[string]any); ok {
						args = input
					}
					toolParts = append(toolParts, GenAIPart{
						FunctionCall: &GenAIFunctionCall{Name: name, Args: args},
					})
				}
			}

		case "content_block_delta":
			if d, ok := raw["delta"].(map[string]any); ok {
				if dt, _ := d["type"].(string); dt == "text_delta" {
					if t, _ := d["text"].(string); t != "" {
						textBuf.WriteString(t)
					}
				}
			}

		case "message_delta":
			if u, ok := raw["usage"].(map[string]any); ok {
				if ot, _ := u["output_tokens"].(float64); ot > 0 {
					resp.UsageMetadata.CandidatesTokenCount = int(ot)
				}
			}
		}
	}

	parts := []GenAIPart{}
	if textBuf.Len() > 0 {
		parts = append(parts, GenAIPart{Text: textBuf.String()})
	}
	parts = append(parts, toolParts...)
	if len(parts) == 0 {
		parts = []GenAIPart{{Text: ""}}
	}

	resp.Candidates = append(resp.Candidates, genAICandidate{
		Index: 0,
		Content: GenAIContent{Role: "model", Parts: parts},
	})
	return resp
}

// ─── Internal Types for Serialization ─────────────────────────────────

type genAIChunk struct {
	Candidates []genAICandidate `json:"candidates"`
}

type genAICandidate struct {
	Index        int          `json:"index"`
	Content      GenAIContent `json:"content"`
	FinishReason string       `json:"finishReason,omitempty"`
}

type GenAIResponse struct {
	Candidates    []genAICandidate `json:"candidates"`
	UsageMetadata *GenAIUsage      `json:"usageMetadata,omitempty"`
}

type GenAIUsage struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
}
