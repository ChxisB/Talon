// Package openai provides a shared OpenAI Chat Completions transport.
// Any provider with an OpenAI-compatible /chat/completions endpoint
// (Gemini, Mistral, Groq, Cerebras, Codestral, NVIDIA NIM, OpenCode, etc.) can use this.
package openai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/ChxisB/spectre-proxy/agent/internal/logger"
	"github.com/ChxisB/spectre-proxy/agent/internal/protocol"
	"github.com/ChxisB/spectre-proxy/agent/internal/providers"
	"github.com/ChxisB/spectre-proxy/agent/internal/providers/protoutil"
)

// Config holds configuration for an OpenAI-compatible provider.
type Config struct {
	Name     string
	BaseURL  string
	APIKey   string
	ModelMap func(string) string
	// SkipToolValidation disables required-field validation on tool calls.
	// Some models (e.g. DeepSeek on opencode_go) generate tool calls with
	// inconsistent schemas that fail strict validation. Set this to true
	// to let those tool calls through and let the client handle them instead.
	SkipToolValidation bool
}

// Transport implements the Provider interface for OpenAI-compatible APIs.
type Transport struct {
	cfg    Config
	client *http.Client
}

// NewTransport creates a new OpenAI Chat transport.
func NewTransport(cfg Config) *Transport {
	return &Transport{
		cfg:    cfg,
		client: &http.Client{Timeout: 5 * time.Minute},
	}
}

// ID returns the provider identifier.
func (t *Transport) ID() string { return t.cfg.Name }

// ProtocolSupport returns which CLI protocols this transport supports.
func (t *Transport) ProtocolSupport() providers.ProtocolSupport {
	return providers.ProtocolSupport{
		Anthropic: true,  // Via OpenAI Chat → Anthropic SSE translation
		Responses: true,  // Via protocol translation
		GenAI:     true,  // Via protocol translation
	}
}

// StreamAnthropic translates an Anthropic Messages API request to OpenAI Chat
// Completions format, sends it, and translates the response SSE back to
// Anthropic format.
func (t *Transport) StreamAnthropic(ctx context.Context, req *protocol.MessagesRequest, inputTokens int, thinking bool) (<-chan protocol.SSEEvent, error) {
	ch := make(chan protocol.SSEEvent, 128)

	sysPrompt := extractSystemPrompt(req)
	chatReq := anthropicToOpenAI(req, sysPrompt, t.cfg.Name)

	if t.cfg.ModelMap != nil {
		chatReq.Model = t.cfg.ModelMap(chatReq.Model)
	}

	body, err := json.Marshal(chatReq)
	if err != nil {
		close(ch)
		return ch, fmt.Errorf("%s: marshal: %w", t.cfg.Name, err)
	}

	// Log the full request
	logger.LogRequest(t.cfg.Name, chatReq.Model, map[string]any{
		"url":        chatReq.Model,
		"stream":     true,
		"messages":   len(chatReq.Messages),
		"tools":      len(chatReq.Tools),
		"max_tokens": chatReq.MaxTokens,
		"body":       string(body),
	})

	apiURL := strings.TrimRight(t.cfg.BaseURL, "/") + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(body))
	if err != nil {
		close(ch)
		return ch, fmt.Errorf("%s: create request: %w", t.cfg.Name, err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if t.cfg.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+t.cfg.APIKey)
	}

	resp, err := t.client.Do(httpReq)
	if err != nil {
		close(ch)
		logger.LogError(t.cfg.Name, chatReq.Model, err.Error(), nil)
		return ch, fmt.Errorf("%s: request failed: %w", t.cfg.Name, err)
	}

	if resp.StatusCode != 200 {
		// Retry once on transient errors (502/503/504)
		if resp.StatusCode == 502 || resp.StatusCode == 503 || resp.StatusCode == 504 {
			resp.Body.Close()
			logger.LogDebug(t.cfg.Name, chatReq.Model, fmt.Sprintf("got status %d, retrying...", resp.StatusCode))
			time.Sleep(1 * time.Second)
			httpReq, _ = http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(body))
			httpReq.Header.Set("Content-Type", "application/json")
			if t.cfg.APIKey != "" {
				httpReq.Header.Set("Authorization", "Bearer "+t.cfg.APIKey)
			}
			resp, err = t.client.Do(httpReq)
			if err != nil {
				close(ch)
				logger.LogError(t.cfg.Name, chatReq.Model, fmt.Sprintf("retry failed: %v", err), nil)
				return ch, fmt.Errorf("%s: retry request failed: %w", t.cfg.Name, err)
			}
		}
	}

	if resp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		errMsg := strings.TrimSpace(string(bodyBytes))
		logger.LogError(t.cfg.Name, chatReq.Model, fmt.Sprintf("status %d", resp.StatusCode), map[string]any{
			"response": truncate(errMsg, 1000),
		})
		close(ch)
		return ch, fmt.Errorf("%s: status %d: %s", t.cfg.Name, resp.StatusCode, errMsg)
	}

	contentType := resp.Header.Get("Content-Type")
	logger.LogDebug(t.cfg.Name, chatReq.Model, fmt.Sprintf("response status=200, content-type=%s", contentType))

	go func() {
		defer resp.Body.Close()
		defer close(ch)

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			errMsg := fmt.Sprintf("failed to read response: %v", err.Error())
			logger.LogError(t.cfg.Name, chatReq.Model, errMsg, nil)
			ch <- protocol.SSEEvent{Type: "error", Data: []byte(fmt.Sprintf(`{"type":"error","error":{"type":"transport_error","message":%q}}`, errMsg))}
			return
		}

		logger.LogDebug(t.cfg.Name, chatReq.Model, fmt.Sprintf("received %d bytes", len(respBody)))
		logger.LogResponse(t.cfg.Name, chatReq.Model, map[string]any{
			"bytes":        len(respBody),
			"content_type": contentType,
			"body":         string(respBody),
		})

		trimmedBody := bytes.TrimSpace(respBody)

		// Check if this is a non-streaming JSON response (single JSON object, not SSE)
		if len(trimmedBody) > 0 && trimmedBody[0] == '{' && !bytes.Contains(trimmedBody, []byte("\ndata: ")) {
			logger.LogDebug(t.cfg.Name, chatReq.Model, "detected non-streaming JSON response")
			var nsResp NonStreamingResponse
			if err := json.Unmarshal(trimmedBody, &nsResp); err == nil && len(nsResp.Choices) > 0 {
				logger.LogDebug(t.cfg.Name, chatReq.Model, fmt.Sprintf("parsed non-streaming: model=%s, choices=%d", nsResp.Model, len(nsResp.Choices)))
				emitNonStreamingResponse(ch, &nsResp, req.Tools, t.cfg.SkipToolValidation)
				return
			}
			logger.LogDebug(t.cfg.Name, chatReq.Model, fmt.Sprintf("failed to parse as non-streaming: %v", err))
		}

		// Parse as SSE stream
		logger.LogDebug(t.cfg.Name, chatReq.Model, "parsing as SSE stream")
		parseSSEStream(ch, respBody, req, t.cfg.SkipToolValidation)
	}()

	return ch, nil
}

// NonStreamingResponse holds a parsed non-streaming OpenAI response.
type NonStreamingResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int    `json:"index"`
		FinishReason string `json:"finish_reason"`
		Message      struct {
			Role             string     `json:"role"`
			Content          string     `json:"content"`
			ReasoningContent string     `json:"reasoning_content,omitempty"`
			ToolCalls        []ToolCall `json:"tool_calls,omitempty"`
		} `json:"message"`
	} `json:"choices"`
	Usage *OpenAIUsage `json:"usage,omitempty"`
}

// emitNonStreamingResponse converts a non-streaming OpenAI response to Anthropic SSE events.
// Uses a single SSEBuilder to ensure correct, incrementing block indices.
func emitNonStreamingResponse(ch chan<- protocol.SSEEvent, fullResp *NonStreamingResponse, tools []protocol.ToolDef, skipToolValidation bool) {
	sse := NewSSEBuilder()
	ch <- sse.MessageStart(fullResp.Model, fullResp.Usage.ToProtocol())

	for _, choice := range fullResp.Choices {
		msg := choice.Message

		if msg.Content != "" {
			ch <- sse.ContentBlockStart()
			ch <- sse.ContentBlockDelta(msg.Content)
			ch <- sse.ContentBlockStop()
		}

		for _, tc := range msg.ToolCalls {
			if tc.ID != "" && tc.Function.Name != "" {
				// Validate tool call arguments against required fields (unless skipped)
				if !skipToolValidation {
					if validationErr := validateToolCall(tc.Function.Name, tc.Function.Arguments, tools); validationErr != "" {
						emitToolValidationError(ch, sse, tc.Function.Name, validationErr)
						continue
					}
				}
				if start := sse.StartToolBlock(tc.Function.Name, tc.ID, tc.Function.Arguments); start != nil {
					ch <- *start
					ch <- sse.ContentBlockStop()
				}
			}
		}

		stopReason := mapFinishReason(choice.FinishReason)
		ch <- sse.MessageDelta(stopReason, fullResp.Usage.ToProtocol())
	}

	ch <- sse.MessageStop()
}

// parseSSEStream parses an SSE response body and emits Anthropic SSE events.
func parseSSEStream(ch chan<- protocol.SSEEvent, respBody []byte, req *protocol.MessagesRequest, skipToolValidation bool) {
	scanner := bufio.NewScanner(bytes.NewReader(respBody))
	scanner.Buffer(make([]byte, 0, 64*1024), 512*1024)

	var toolNames []string
	for _, t := range req.Tools {
		toolNames = append(toolNames, t.Name)
	}
	sse := NewSSEBuilder(toolNames...)

	toolCallAccum := map[int]*ToolCall{}
	var textBuf strings.Builder
	var flushedText string
	var allChunks []ChatStreamChunk

	// Send message_start immediately
	ch <- sse.MessageStart(req.Model, &protocol.Usage{InputTokens: 0, OutputTokens: 0})

	pingTicker := time.NewTicker(5 * time.Second)
	defer pingTicker.Stop()
	go func() {
		for range pingTicker.C {
			select {
			case ch <- protocol.SSEEvent{Type: "ping", Data: []byte(`{"type":"ping"}`)}:
			default:
			}
		}
	}()

	lineCount := 0
	for scanner.Scan() {
		line := scanner.Text()
		lineCount++

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			if textBuf.Len() > 0 {
				if !sse.hasContent {
					ch <- sse.ContentBlockStart()
				}
				chunkText := textBuf.String()
				ch <- sse.ContentBlockDelta(chunkText)
				flushedText += chunkText
				textBuf.Reset()
			}
			break
		}

		var chunk ChatStreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		allChunks = append(allChunks, chunk)

		for _, choice := range chunk.Choices {
			delta := choice.Delta

			for _, tc := range delta.ToolCalls {
				existing, ok := toolCallAccum[tc.Index]
				if !ok {
					toolCallAccum[tc.Index] = &ToolCall{
						Index:    tc.Index,
						ID:       tc.ID,
						Type:     tc.Type,
						Function: tc.Function,
					}
				} else {
					if tc.ID != "" {
						existing.ID = tc.ID
					}
					if tc.Type != "" {
						existing.Type = tc.Type
					}
					if tc.Function.Name != "" {
						existing.Function.Name = tc.Function.Name
					}
					if tc.Function.Arguments != "" {
						existing.Function.Arguments += tc.Function.Arguments
					}
				}
			}

			// Emit reasoning content as a separate thinking block so DeepSeek
			// can recognize it when the conversation continues. Some providers
			// (DeepSeek) require `reasoning_content` to be passed back or they
			// return a 400 error.
			if delta.ReasoningContent != "" {
				if !sse.hasContent {
					ch <- sse.ThinkingBlockStart()
				}
				ch <- sse.ThinkingBlockDelta(delta.ReasoningContent)
			}
			if delta.Content != "" {
				textBuf.WriteString(delta.Content)
			}
			if textBuf.Len() >= 200 || choice.FinishReason != "" {
				if textBuf.Len() > 0 {
					if !sse.hasContent {
						ch <- sse.ContentBlockStart()
					}
					chunkText := textBuf.String()
					ch <- sse.ContentBlockDelta(chunkText)
					flushedText += chunkText
					textBuf.Reset()
				}
			}
		}
	}

	log.Printf("[debug] %s: stream complete: lines=%d, chunks=%d, toolCalls=%d, textLen=%d	",
		"sse", lineCount, len(allChunks), len(toolCallAccum), len(flushedText))

	if sse.hasContent {
		ch <- sse.ContentBlockStop()
		sse.hasContent = false
	}

	toolCreated := false
	for _, tc := range toolCallAccum {
		if tc.ID != "" && tc.Function.Name != "" && tc.Function.Arguments != "" && !sse.openToolBlock {
			// Validate tool call arguments against required fields (unless skipped)
			if !skipToolValidation {
				if validationErr := validateToolCall(tc.Function.Name, tc.Function.Arguments, req.Tools); validationErr != "" {
					// Emit validation error instead of the tool call
					emitToolValidationError(ch, sse, tc.Function.Name, validationErr)
					continue
				}
			}
			if start := sse.StartToolBlock(tc.Function.Name, tc.ID, tc.Function.Arguments); start != nil {
				ch <- *start
				ch <- sse.ContentBlockStop()
				toolCreated = true
			}
		}
	}

	stopReason := "end_turn"
	if toolCreated {
		stopReason = "tool_use"
	}

	var usage *protocol.Usage
	if len(allChunks) > 0 {
		usage = allChunks[len(allChunks)-1].Usage.ToProtocol()
	}
	if usage == nil {
		usage = &protocol.Usage{}
	}

	ch <- sse.MessageDelta(stopReason, usage)
	ch <- sse.MessageStop()
}

// ListModels returns available models.
func (t *Transport) ListModels(ctx context.Context) ([]string, error) {
	apiURL := strings.TrimRight(t.cfg.BaseURL, "/") + "/models"
	httpReq, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	if t.cfg.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+t.cfg.APIKey)
	}

	resp, err := t.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	models := make([]string, len(result.Data))
	for i, m := range result.Data {
		models[i] = m.ID
	}
	return models, nil
}

// CheckHealth verifies the provider is reachable.
func (t *Transport) CheckHealth(ctx context.Context) error {
	apiURL := strings.TrimRight(t.cfg.BaseURL, "/") + "/models"
	httpReq, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return err
	}
	if t.cfg.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+t.cfg.APIKey)
	}
	resp, err := t.client.Do(httpReq)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode >= 500 {
		return fmt.Errorf("status %d", resp.StatusCode)
	}
	return nil
}

// StreamResponses translates an OpenAI Responses API request to OpenAI Chat
// format via the internal MessagesRequest, then translates back to Responses SSE.
func (t *Transport) StreamResponses(ctx context.Context, rawReq json.RawMessage, resolvedModel string) (<-chan []byte, error) {
	msgReq, err := protoutil.ParseResponsesRequest(rawReq, resolvedModel)
	if err != nil {
		return nil, fmt.Errorf("%s: parse responses request: %w", t.cfg.Name, err)
	}
	msgReq.Model = resolvedModel

	anthroEvents, err := t.StreamAnthropic(ctx, msgReq, 0, false)
	if err != nil {
		return nil, fmt.Errorf("%s: stream responses: %w", t.cfg.Name, err)
	}

	return protoutil.TranslateAnthropicToResponses(anthroEvents, resolvedModel), nil
}

// StreamGenAI translates a Google GenAI API request to OpenAI Chat format,
// then translates the response back to GenAI streaming format.
func (t *Transport) StreamGenAI(ctx context.Context, rawReq json.RawMessage, resolvedModel string) (<-chan []byte, error) {
	msgReq, err := protoutil.ParseGenAIRequest(rawReq, resolvedModel)
	if err != nil {
		return nil, fmt.Errorf("%s: parse genai request: %w", t.cfg.Name, err)
	}
	msgReq.Model = resolvedModel

	anthroEvents, err := t.StreamAnthropic(ctx, msgReq, 0, false)
	if err != nil {
		return nil, fmt.Errorf("%s: stream genai: %w", t.cfg.Name, err)
	}

	return protoutil.TranslateAnthropicToGenAI(anthroEvents), nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "...[truncated]"
}

// validateToolCall checks that a tool call's arguments contain all required
// fields defined in the tool's input schema. Returns an error message if
// validation fails, or empty string if valid.
func validateToolCall(toolName string, args string, tools []protocol.ToolDef) string {
	// Find the tool definition
	var toolDef *protocol.ToolDef
	for i := range tools {
		if tools[i].Name == toolName {
			toolDef = &tools[i]
			break
		}
	}
	if toolDef == nil {
		return "" // unknown tool, let it through
	}

	// Parse the arguments
	if args == "" || args == "{}" {
		args = "{}"
	}
	var argsMap map[string]any
	if err := json.Unmarshal([]byte(args), &argsMap); err != nil {
		return fmt.Sprintf("Tool %q received invalid JSON arguments: %v", toolName, err)
	}

	// Extract required fields from the schema
	schema, ok := toolDef.InputSchema.(map[string]any)
	if !ok {
		return "" // no schema to validate against
	}

	required, ok := schema["required"].([]any)
	if !ok {
		// Try []string (some schemas use this)
		if reqStr, ok := schema["required"].([]string); ok {
			for _, field := range reqStr {
				if _, exists := argsMap[field]; !exists {
					return fmt.Sprintf("Tool %q is missing required field: %q", toolName, field)
				}
			}
			return ""
		}
		return "" // no required fields
	}

	for _, field := range required {
		fieldName, ok := field.(string)
		if !ok {
			continue
		}
		if _, exists := argsMap[fieldName]; !exists {
			return fmt.Sprintf("Tool %q is missing required field: %q", toolName, fieldName)
		}
	}
	return ""
}

// emitToolValidationError sends an error message back to the model when a
// tool call has invalid or missing required arguments, preventing infinite loops.
func emitToolValidationError(ch chan<- protocol.SSEEvent, sse *SSEBuilder, toolName, errMsg string) {
	// Send as a text response so the model can self-correct
	if !sse.hasContent {
		ch <- sse.ContentBlockStart()
	}
	ch <- sse.ContentBlockDelta(fmt.Sprintf("[Tool validation error] %s", errMsg))
	ch <- sse.ContentBlockStop()
	log.Printf("[tool-validation] %s: %s", toolName, errMsg)
}
