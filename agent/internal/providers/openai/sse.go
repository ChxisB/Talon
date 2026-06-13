package openai

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ChxisB/spectre-proxy/agent/internal/protocol"
)

// SSEBuilder constructs Anthropic SSE events for a streaming response.
type SSEBuilder struct {
	hasContent     bool
	knownToolNames map[string]bool // valid tool names from the request
	openToolBlock  bool            // true when a tool_use block is open
	blockIndex     int             // incrementing index for content blocks
}

func NewSSEBuilder(knownToolNames ...string) *SSEBuilder {
	names := make(map[string]bool)
	for _, n := range knownToolNames {
		names[n] = true
	}
	return &SSEBuilder{knownToolNames: names}
}

func (s *SSEBuilder) MessageStart(model string, usage *protocol.Usage) protocol.SSEEvent {
	data, _ := json.Marshal(map[string]any{
		"type": "message_start",
		"message": map[string]any{
			"id":      fmt.Sprintf("msg_%d", timeNowNano()),
			"type":    "message",
			"role":    "assistant",
			"model":   model,
			"content": []any{},
			"usage":   usage,
		},
	})
	return protocol.SSEEvent{Type: "message_start", Data: data}
}

func (s *SSEBuilder) ContentBlockStart() protocol.SSEEvent {
	s.hasContent = true
	idx := s.blockIndex
	s.blockIndex++
	data, err := json.Marshal(map[string]any{
		"type":          "content_block_start",
		"index":         idx,
		"content_block": map[string]string{"type": "text", "text": ""},
	})
	if err != nil || len(data) == 0 {
		data = []byte(fmt.Sprintf(`{"type":"content_block_start","index":%d,"content_block":{"type":"text","text":""}}`, idx))
	}
	return protocol.SSEEvent{Type: "content_block_start", Data: data}
}

func (s *SSEBuilder) ContentBlockDelta(text string) protocol.SSEEvent {
	data, _ := json.Marshal(map[string]any{
		"type":  "content_block_delta",
		"index": s.blockIndex - 1,
		"delta": map[string]string{"type": "text_delta", "text": text},
	})
	return protocol.SSEEvent{Type: "content_block_delta", Data: data}
}

func (s *SSEBuilder) ContentBlockStop() protocol.SSEEvent {
	data, _ := json.Marshal(map[string]any{"type": "content_block_stop", "index": s.blockIndex - 1})
	return protocol.SSEEvent{Type: "content_block_stop", Data: data}
}

func (s *SSEBuilder) MessageDelta(stopReason string, usage *protocol.Usage) protocol.SSEEvent {
	data, _ := json.Marshal(map[string]any{
		"type": "message_delta",
		"delta": map[string]any{
			"stop_reason":   stopReason,
			"stop_sequence": nil,
		},
		"usage": usage,
	})
	return protocol.SSEEvent{Type: "message_delta", Data: data}
}

func (s *SSEBuilder) MessageStop() protocol.SSEEvent {
	data, _ := json.Marshal(map[string]string{"type": "message_stop"})
	return protocol.SSEEvent{Type: "message_stop", Data: data}
}

// ─── Thinking Block Support ──────────────────────────────────────────

func (s *SSEBuilder) ThinkingBlockStart() protocol.SSEEvent {
	s.hasContent = true
	idx := s.blockIndex
	s.blockIndex++
	data, _ := json.Marshal(map[string]any{
		"type":  "content_block_start",
		"index": idx,
		"content_block": map[string]any{
			"type": "thinking",
			// Note: thinking content comes in delta events, not here
		},
	})
	return protocol.SSEEvent{Type: "content_block_start", Data: data}
}

func (s *SSEBuilder) ThinkingBlockDelta(thinking string) protocol.SSEEvent {
	data, _ := json.Marshal(map[string]any{
		"type":  "content_block_delta",
		"index": s.blockIndex - 1,
		"delta": map[string]string{"type": "thinking_delta", "thinking": thinking},
	})
	return protocol.SSEEvent{Type: "content_block_delta", Data: data}
}

func (s *SSEBuilder) ThinkingBlockStop() protocol.SSEEvent {
	s.hasContent = false
	data, _ := json.Marshal(map[string]any{"type": "content_block_stop", "index": s.blockIndex - 1})
	return protocol.SSEEvent{Type: "content_block_stop", Data: data}
}

// ─── Tool Block Helpers ───────────────────────────────────────────────

// StartToolBlock begins a tool_use content block with name guard.
// Returns nil if the tool name is empty or not in the known list (phantom suppression).
func (s *SSEBuilder) StartToolBlock(name, id, args string) *protocol.SSEEvent {
	if name == "" || id == "" {
		return nil
	}
	// Phantom tool name suppression
	if len(s.knownToolNames) > 0 && !s.knownToolNames[name] {
		return nil
	}

	s.openToolBlock = true
	s.hasContent = true
	idx := s.blockIndex
	s.blockIndex++

	// Validate and prepare args for the input field.
	// Empty args → empty object. Invalid JSON → wrap as raw string to avoid
	// "Tool use input must be a string or object" client errors.
	inputJSON := json.RawMessage("{}")
	if len(args) > 0 {
		if json.Valid([]byte(args)) {
			inputJSON = json.RawMessage(args)
		} else {
			// Args are not valid JSON — wrap them so the client gets a valid object
			inputJSON = json.RawMessage(fmt.Sprintf(`{"_raw_arguments":%q}`, args))
		}
	}

	data, _ := json.Marshal(map[string]any{
		"type":  "content_block_start",
		"index": idx,
		"content_block": map[string]any{
			"type":  "tool_use",
			"id":    id,
			"name":  name,
			"input": inputJSON,
		},
	})
	evt := protocol.SSEEvent{Type: "content_block_start", Data: data}
	return &evt
}

// EndToolBlock closes an open tool block.
func (s *SSEBuilder) EndToolBlock() *protocol.SSEEvent {
	if !s.openToolBlock {
		return nil
	}
	s.openToolBlock = false
	evt := s.ContentBlockStop()
	return &evt
}

// CloseIncompleteToolBlock validates and repairs tool block arguments at stream end.
func (s *SSEBuilder) CloseIncompleteToolBlock(acc string) (string, *protocol.SSEEvent) {
	if !s.openToolBlock {
		return acc, nil
	}

	repaired := rescuePartialJSON(acc)
	stop := s.ContentBlockStop()
	s.openToolBlock = false
	return repaired, &stop
}

// ─── JSON Rescue ───────────────────────────────────────────────────────

// rescuePartialJSON completes partial JSON when a stream disconnects mid-tool-call.
func rescuePartialJSON(text string) string {
	if text == "" {
		return `{"_interrupted":true}`
	}

	trimmed := strings.TrimRight(text, " \n\r\t,")

	// Handle unterminated string
	if strings.Count(trimmed, "\"")%2 != 0 {
		trimmed += "\""
	}

	// Count open/close braces
	openBraces := strings.Count(trimmed, "{")
	closeBraces := strings.Count(trimmed, "}")
	openBrackets := strings.Count(trimmed, "[")
	closeBrackets := strings.Count(trimmed, "]")

	for openBraces > closeBraces {
		trimmed += "}"
		closeBraces++
	}
	for openBrackets > closeBrackets {
		trimmed += "]"
		closeBrackets++
	}

	if !json.Valid([]byte(trimmed)) {
		return `{"_interrupted":true}`
	}

	return trimmed
}

// timeNowNano provides a unique timestamp for message IDs.
func timeNowNano() int64 {
	return time.Now().UnixNano()
}
