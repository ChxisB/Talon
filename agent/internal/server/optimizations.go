package server

import (
	"github.com/ChxisB/spectre-proxy/agent/internal/config"
	"github.com/ChxisB/spectre-proxy/agent/internal/protocol"
	"github.com/ChxisB/spectre-proxy/agent/internal/router"
	"github.com/google/uuid"
)

// tryOptimizations checks for fast-path responses before forwarding to a provider.
func tryOptimizations(req *protocol.MessagesRequest, resolved *router.ResolvedModel, cfg *config.Settings) *protocol.MessagesResponse {
	lastMsg := lastUserMessage(req.Messages)
	if lastMsg == "" {
		return nil
	}

	// Network probe mock
	if cfg.EnableNetworkProbeMock && isQuotaCheck(lastMsg) {
		return textResponse(req.Model, "Quota check passed.", 10, 5)
	}

	// Fast prefix detection
	if cfg.FastPrefixDetection && isPrefixRequest(lastMsg) {
		return textResponse(req.Model, extractCommandPrefix(lastMsg), 100, 5)
	}

	// Title generation skip
	if cfg.EnableTitleGenerationSkip && isTitleRequest(lastMsg) {
		return textResponse(req.Model, "Conversation", 100, 5)
	}

	// Suggestion mode skip
	if cfg.EnableSuggestionModeSkip && isSuggestionRequest(lastMsg) {
		return textResponse(req.Model, "", 100, 5)
	}

	return nil
}

func textResponse(model, text string, inputTokens, outputTokens int) *protocol.MessagesResponse {
	return &protocol.MessagesResponse{
		ID:    "msg_" + uuid.New().String(),
		Model: model,
		Content: []protocol.ContentBlock{
			{Type: "text", Text: text},
		},
		StopReason: "end_turn",
		Usage: protocol.Usage{
			InputTokens:  inputTokens,
			OutputTokens: outputTokens,
		},
	}
}

func lastUserMessage(messages []protocol.Message) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			blocks, _ := messages[i].ContentBlocks()
			for _, block := range blocks {
				if block.Type == "text" && block.Text != "" {
					return block.Text
				}
			}
		}
	}
	return ""
}

func isQuotaCheck(text string) bool {
	return containsAny(text, []string{"quota", "/quota", "billing", "usage"})
}

func isPrefixRequest(text string) bool {
	return containsAny(text, []string{"command_prefix", "find_prefix"})
}

func isTitleRequest(text string) bool {
	return containsAny(text, []string{
		`"type":"conversation"`, `"type": "conversation"`,
		`summary`,
		"generate a title",
		"conversation title",
	})
}

func isSuggestionRequest(text string) bool {
	return containsAny(text, []string{"suggestion_mode", "suggestion mode"})
}

func extractCommandPrefix(text string) string {
	// Simple heuristic: grab the first line or first word
	for _, line := range splitLines(text) {
		line = trimSpace(line)
		if line != "" {
			return line
		}
	}
	return ""
}

func containsAny(s string, substrs []string) bool {
	for _, sub := range substrs {
		if stringsContains(s, sub) {
			return true
		}
	}
	return false
}
