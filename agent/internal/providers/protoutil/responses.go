// Package protoutil provides protocol translation utilities for AI agent CLIs.
//
// This package implements the wire format translations between:
//   - Anthropic Messages API  ↔  internal MessagesRequest
//   - OpenAI Responses API    ↔  internal MessagesRequest
//   - Google GenAI API        ↔  internal MessagesRequest
//
// Translations are bidirectional: request parsing converts from the wire format
// to the internal MessagesRequest, and SSE streaming converts from Anthropic-format
// SSE events (returned by all providers) back to the wire format.
package protoutil

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/ChxisB/spectre-proxy/agent/internal/protocol"
	"github.com/google/uuid"
)

// ─── Request Types ─────────────────────────────────────────────────────

// ResponsesRequest is an OpenAI Responses API request body.
type ResponsesRequest struct {
	Model        string          `json:"model"`
	Input        json.RawMessage `json:"input"`
	Instructions string          `json:"instructions,omitempty"`
	Tools        []ResponseTool  `json:"tools,omitempty"`
	Stream       bool            `json:"stream,omitempty"`
	MaxTokens    int             `json:"max_output_tokens,omitempty"`
	Temperature  *float64        `json:"temperature,omitempty"`
	TopP         *float64        `json:"top_p,omitempty"`
}

// ResponseTool is a tool definition in the Responses API.
type ResponseTool struct {
	Type        string          `json:"type"`
	Name        string          `json:"name,omitempty"`
	Description string          `json:"description,omitempty"`
	Function    json.RawMessage `json:"function,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
}

// InputItem is a single item in the input array.
type InputItem struct {
	Type    string          `json:"type"`    // "message", "function_call_output"
	Role    string          `json:"role"`    // "user", "assistant"
	Content json.RawMessage `json:"content"` // array of parts or string
	CallID  string          `json:"call_id,omitempty"`
	Output  string          `json:"output,omitempty"`
}

// ─── Request Parsing ──────────────────────────────────────────────────

// ParseResponsesRequest converts an OpenAI Responses API JSON body into an
// internal MessagesRequest that can be routed through any provider.
func ParseResponsesRequest(raw json.RawMessage, defaultModel string) (*protocol.MessagesRequest, error) {
	var req ResponsesRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		return nil, fmt.Errorf("parse responses request: %w", err)
	}

	mr := &protocol.MessagesRequest{
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		TopP:        req.TopP,
	}

	if req.Model == "" {
		mr.Model = defaultModel
	} else {
		mr.Model = req.Model
	}

	// Instructions → system prompt
	if req.Instructions != "" {
		sysBytes, _ := json.Marshal([]protocol.ContentBlock{{Type: "text", Text: req.Instructions}})
		mr.System = sysBytes
	}

	// Tools
	if len(req.Tools) > 0 {
		tools := make([]protocol.ToolDef, 0, len(req.Tools))
		for _, t := range req.Tools {
			tool := protocol.ToolDef{
				Name:        t.Name,
				Description: t.Description,
				InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
			}
			if t.Function != nil {
				var fn struct {
					Name        string `json:"name"`
					Description string `json:"description"`
					Parameters  any    `json:"parameters"`
				}
				if json.Unmarshal(t.Function, &fn) == nil {
					if fn.Name != "" {
						tool.Name = fn.Name
					}
					if fn.Description != "" {
						tool.Description = fn.Description
					}
					if fn.Parameters != nil {
						tool.InputSchema = fn.Parameters
					}
				}
			} else if t.Parameters != nil {
				var params any
				if json.Unmarshal(t.Parameters, &params) == nil {
					tool.InputSchema = params
				}
			}
			tools = append(tools, tool)
		}
		mr.Tools = tools
	}

	// Input items → messages
	if len(req.Input) == 0 {
		return mr, nil
	}

	var items []InputItem
	if err := json.Unmarshal(req.Input, &items); err != nil {
		// Try as a single string
		var inputStr string
		if err2 := json.Unmarshal(req.Input, &inputStr); err2 == nil && inputStr != "" {
			blocks, _ := json.Marshal([]protocol.ContentBlock{{Type: "text", Text: inputStr}})
			mr.Messages = []protocol.Message{{Role: protocol.RoleUser, Content: blocks}}
			return mr, nil
		}
		return nil, fmt.Errorf("parse responses input: %w", err)
	}

	messages := make([]protocol.Message, 0, len(items))
	for _, item := range items {
		switch item.Type {
		case "message", "":
			role := protocol.RoleUser
			switch item.Role {
			case "assistant", "model":
				role = protocol.RoleAssistant
			case "system":
				role = protocol.RoleSystem
			}
			msg := protocol.Message{Role: role}
			blocks := parseContentParts(item.Content)
			data, _ := json.Marshal(blocks)
			msg.Content = data
			messages = append(messages, msg)

		case "function_call_output":
			msg := protocol.Message{Role: protocol.RoleUser}
			blocks := []protocol.ContentBlock{{
				Type:      "tool_result",
				ToolUseID: item.CallID,
				Content:   item.Output,
			}}
			data, _ := json.Marshal(blocks)
			msg.Content = data
			messages = append(messages, msg)
		}
	}
	mr.Messages = messages
	return mr, nil
}

// parseContentParts converts Responses API content parts to Anthropic content blocks.
func parseContentParts(raw json.RawMessage) []protocol.ContentBlock {
	if len(raw) == 0 {
		return []protocol.ContentBlock{{Type: "text", Text: ""}}
	}

	type part struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	var parts []part
	if json.Unmarshal(raw, &parts) == nil {
		blocks := make([]protocol.ContentBlock, 0, len(parts))
		for _, p := range parts {
			blocks = append(blocks, protocol.ContentBlock{Type: "text", Text: p.Text})
		}
		return blocks
	}

	var text string
	if json.Unmarshal(raw, &text) == nil {
		return []protocol.ContentBlock{{Type: "text", Text: text}}
	}

	return []protocol.ContentBlock{{Type: "text", Text: ""}}
}

// ─── SSE Translation ─────────────────────────────────────────────────

// TranslateAnthropicToResponses converts an Anthropic-format SSE event channel
// into pre-formatted Responses API SSE bytes.
//
// Each []byte in the output channel is a complete SSE event ready to write:
//   data: {"type":"response.output_text.delta","delta":"...","item_id":"..."}\n\n
func TranslateAnthropicToResponses(anthro <-chan protocol.SSEEvent, model string) <-chan []byte {
	out := make(chan []byte, 128)
	go func() {
		defer close(out)

		responseID := "resp_" + uuid.New().String()
		var currentItemID string
		var currentToolID string
		var toolArgsBuf strings.Builder
		var textBuf strings.Builder
		isToolCall := false
		toolName := ""
		hasContent := false

		// Helper: emit a SSE data line
		emit := func(evtType string, extra map[string]any) {
			payload := map[string]any{"type": evtType}
			for k, v := range extra {
				payload[k] = v
			}
			data, _ := json.Marshal(payload)
			out <- []byte(fmt.Sprintf("data: %s\n\n", string(data)))
		}

		// response.created
		emit("response.created", map[string]any{
			"response": map[string]any{
				"id": responseID, "model": model,
				"status": "in_progress", "output": []any{},
			},
		})

		// response.in_progress
		emit("response.in_progress", map[string]any{
			"response": map[string]any{
				"id": responseID, "model": model, "status": "in_progress",
			},
		})

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
				itemID := "item_" + uuid.New().String()
				currentItemID = itemID
				hasContent = true

				if bt == "tool_use" {
					isToolCall = true
					toolName, _ = cb["name"].(string)
					toolArgsBuf.Reset()
					currentToolID = "call_" + uuid.New().String()

					emit("response.output_item.added", map[string]any{
						"item": map[string]any{
							"id": itemID, "type": "function_call",
							"name": toolName, "arguments": "",
							"status": "in_progress", "call_id": currentToolID,
						},
					})

					if input, ok := cb["input"].(map[string]any); ok && len(input) > 0 {
						initArgs, _ := json.Marshal(input)
						toolArgsBuf.WriteString(string(initArgs))
						emit("response.function_call_arguments.delta", map[string]any{
							"delta": string(initArgs), "item_id": itemID,
						})
					}
				} else {
					isToolCall = false
					textBuf.Reset()

					emit("response.output_item.added", map[string]any{
						"item": map[string]any{
							"id": itemID, "type": "message",
							"role": "assistant", "status": "in_progress", "content": []any{},
						},
					})
					emit("response.content_part.added", map[string]any{
						"item_id": itemID,
						"part":    map[string]any{"type": "output_text", "text": ""},
					})
				}

			case "content_block_delta":
				delta, _ := raw["delta"].(map[string]any)
				dt, _ := delta["type"].(string)
				if dt == "text_delta" {
					if text, _ := delta["text"].(string); text != "" {
						textBuf.WriteString(text)
						emit("response.output_text.delta", map[string]any{
							"delta": text, "item_id": currentItemID,
						})
					}
				}

			case "content_block_stop":
				if isToolCall {
					emit("response.function_call_arguments.done", map[string]any{
						"item_id": currentItemID,
					})
					emit("response.output_item.done", map[string]any{
						"item": map[string]any{
							"id": currentItemID, "type": "function_call",
							"status": "completed", "name": toolName,
							"arguments": toolArgsBuf.String(), "call_id": currentToolID,
						},
					})
				} else {
					emit("response.output_text.done", map[string]any{
						"item_id": currentItemID,
					})
					emit("response.output_item.done", map[string]any{
						"item": map[string]any{
							"id": currentItemID, "type": "message",
							"status": "completed", "role": "assistant",
							"content": []map[string]any{
								{"type": "output_text", "text": textBuf.String(), "annotations": []string{}},
							},
						},
					})
				}

			case "message_delta":
				u, _ := raw["usage"].(map[string]any)
				usage := &responsesUsage{}
				if ot, _ := u["output_tokens"].(float64); ot > 0 {
					usage.OutputTokens = int(ot)
				}
				emit("response.completed", map[string]any{
					"response": map[string]any{
						"id": responseID, "model": model,
						"status": "completed", "output": []any{},
						"usage":  usage,
					},
				})

			case "error":
				emit("response.failed", map[string]any{
					"response": map[string]any{
						"id": responseID, "model": model, "status": "failed",
					},
				})
			}
		}

		if !hasContent {
			emit("response.completed", map[string]any{
				"response": map[string]any{
					"id": responseID, "model": model,
					"status": "completed", "output": []any{},
					"usage":  &responsesUsage{},
				},
			})
		}

		// End-of-stream marker
		out <- []byte("data: [DONE]\n\n")
	}()
	return out
}

// BuildResponsesResponse collects all Anthropic SSE events into a Responses API
// JSON response object (for non-streaming requests).
func BuildResponsesResponse(anthro <-chan protocol.SSEEvent, model string) *ResponsesFullResponse {
	resp := &ResponsesFullResponse{
		ID:     "resp_" + uuid.New().String(),
		Object: "response",
		Model:  model,
		Status: "completed",
		Output: []ResponseOutputItem{},
		Usage:  &responsesUsage{},
	}

	var cur *ResponseOutputItem
	var textBuf strings.Builder
	var toolName string
	var toolArgs strings.Builder
	isToolCall := false

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
						resp.Usage.InputTokens = int(it)
					}
				}
			}

		case "content_block_start":
			cb, _ := raw["content_block"].(map[string]any)
			bt, _ := cb["type"].(string)
			iid := "item_" + uuid.New().String()

			if bt == "tool_use" {
				isToolCall = true
				toolName, _ = cb["name"].(string)
				toolArgs.Reset()
				cur = &ResponseOutputItem{
					ID: iid, Type: "function_call",
					Name: toolName, Status: "completed",
					CallID: "call_" + uuid.New().String(),
				}
				if input, ok := cb["input"].(map[string]any); ok && len(input) > 0 {
					a, _ := json.Marshal(input)
					toolArgs.WriteString(string(a))
				}
			} else {
				isToolCall = false
				textBuf.Reset()
				cur = &ResponseOutputItem{
					ID: iid, Type: "message",
					Role: "assistant", Status: "completed",
					Content: []ResponseContentPart{},
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

		case "content_block_stop":
			if cur != nil {
				if isToolCall {
					cur.Arguments = toolArgs.String()
				} else if textBuf.Len() > 0 {
					cur.Content = append(cur.Content, ResponseContentPart{
						Type: "output_text", Text: textBuf.String(), Annotations: []string{},
					})
				}
				resp.Output = append(resp.Output, *cur)
				cur = nil
			}

		case "message_delta":
			if u, ok := raw["usage"].(map[string]any); ok {
				if ot, _ := u["output_tokens"].(float64); ot > 0 {
					resp.Usage.OutputTokens = int(ot)
				}
			}
		}
	}
	return resp
}

// ─── Response Types ───────────────────────────────────────────────────

type responsesUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type ResponsesFullResponse struct {
	ID     string               `json:"id"`
	Object string               `json:"object"`
	Model  string               `json:"model"`
	Status string               `json:"status"`
	Output []ResponseOutputItem `json:"output"`
	Usage  *responsesUsage      `json:"usage,omitempty"`
}

type ResponseOutputItem struct {
	ID        string                `json:"id"`
	Type      string                `json:"type"`
	Role      string                `json:"role,omitempty"`
	Status    string                `json:"status"`
	Content   []ResponseContentPart `json:"content,omitempty"`
	Name      string                `json:"name,omitempty"`
	CallID    string                `json:"call_id,omitempty"`
	Arguments string                `json:"arguments,omitempty"`
}

type ResponseContentPart struct {
	Type        string   `json:"type"`
	Text        string   `json:"text"`
	Annotations []string `json:"annotations,omitempty"`
}

// Ensure uuid usage
var _ = uuid.NewString
var _ = log.Printf
