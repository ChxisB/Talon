package llm

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type AnthropicProvider struct {
	config ProviderConfig
	client *http.Client
}

func NewAnthropicProvider(config ProviderConfig) *AnthropicProvider {
	if config.BaseURL == "" {
		config.BaseURL = "https://api.anthropic.com/v1"
	}
	return &AnthropicProvider{
		config: config,
		client: &http.Client{},
	}
}

func (p *AnthropicProvider) Name() string {
	return "anthropic"
}

func (p *AnthropicProvider) Complete(req CompletionRequest) (*CompletionResponse, error) {
	anthropicReq := p.toAnthropicRequest(req)
	body, err := json.Marshal(anthropicReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", p.config.BaseURL+"/messages", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.config.APIKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Anthropic API error %d: %s", resp.StatusCode, string(respBody))
	}

	var anthropicResp struct {
		Content []struct {
			Type  string `json:"type"`
			Text  string `json:"text"`
			ID    string `json:"id"`
			Name  string `json:"name"`
			Input json.RawMessage `json:"input"`
		} `json:"content"`
		StopReason string `json:"stop_reason"`
		Usage      struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&anthropicResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	result := &CompletionResponse{
		FinishReason: anthropicResp.StopReason,
		Usage: &Usage{
			InputTokens:  anthropicResp.Usage.InputTokens,
			OutputTokens: anthropicResp.Usage.OutputTokens,
		},
	}

	for _, block := range anthropicResp.Content {
		switch block.Type {
		case "text":
			result.Content += block.Text
		case "tool_use":
			inputStr := string(block.Input)
			result.ToolCalls = append(result.ToolCalls, ToolCall{
				ID:   block.ID,
				Type: "function",
				Function: Function{
					Name:      block.Name,
					Arguments: inputStr,
				},
			})
		}
	}

	return result, nil
}

func (p *AnthropicProvider) CompleteStream(req CompletionRequest, events chan<- StreamEvent) error {
	defer close(events)

	anthropicReq := p.toAnthropicRequest(req)
	anthropicReq["stream"] = true
	body, err := json.Marshal(anthropicReq)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", p.config.BaseURL+"/messages", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.config.APIKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Anthropic API error %d: %s", resp.StatusCode, string(respBody))
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			events <- StreamEvent{Type: StreamEventDone}
			return nil
		}

		var event struct {
			Type string `json:"type"`
			Delta *struct {
				Text string `json:"text"`
				StopReason string `json:"stop_reason"`
			} `json:"delta,omitempty"`
			ContentBlock *struct {
				Type string `json:"type"`
				ID   string `json:"id"`
				Name string `json:"name"`
				Input json.RawMessage `json:"input"`
			} `json:"content_block,omitempty"`
			Message *struct {
				Usage *Usage `json:"usage,omitempty"`
				StopReason string `json:"stop_reason"`
			} `json:"message,omitempty"`
		}

		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}

		switch event.Type {
		case "content_block_delta":
			if event.Delta != nil && event.Delta.Text != "" {
				events <- StreamEvent{Type: StreamEventText, Delta: event.Delta.Text}
			}
		case "content_block_start":
			if event.ContentBlock != nil && event.ContentBlock.Type == "tool_use" {
				inputStr := string(event.ContentBlock.Input)
				if inputStr == "" {
					inputStr = "{}"
				}
				events <- StreamEvent{
					Type: StreamEventToolCall,
					ToolCalls: []ToolCall{{
						ID:   event.ContentBlock.ID,
						Type: "function",
						Function: Function{
							Name:      event.ContentBlock.Name,
							Arguments: inputStr,
						},
					}},
				}
			}
		case "message_delta":
			if event.Message != nil && event.Message.StopReason != "" {
				events <- StreamEvent{
					Type:         StreamEventDone,
					FinishReason: event.Message.StopReason,
				}
			}
		case "message_stop":
			events <- StreamEvent{Type: StreamEventDone}
		}
	}

	return scanner.Err()
}

func (p *AnthropicProvider) toAnthropicRequest(req CompletionRequest) map[string]any {
	messages := make([]map[string]any, 0, len(req.Messages))
	system := ""

	for _, msg := range req.Messages {
		if msg.Role == RoleSystem {
			system += msg.Content + "\n"
			continue
		}

		m := map[string]any{
			"role": string(msg.Role),
		}

		// Handle tool results
		if msg.Role == RoleTool {
			m["content"] = []map[string]any{
				{
					"type":        "tool_result",
					"tool_use_id": msg.ToolCallID,
					"content":     msg.Content,
				},
			}
			messages = append(messages, m)
			continue
		}

		m["content"] = msg.Content

		messages = append(messages, m)
	}

	result := map[string]any{
		"model":    req.Model,
		"messages": messages,
		maxTokensKey(): req.MaxTokens,
	}

	if system != "" {
		result["system"] = strings.TrimSpace(system)
	}
	if len(req.Tools) > 0 {
		result["tools"] = req.Tools
	}

	return result
}

// Anthropic uses "max_tokens" (not "max_tokens")
func maxTokensKey() string {
	return "max_tokens"
}
