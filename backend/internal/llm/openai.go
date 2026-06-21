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

type OpenAIProvider struct {
	config ProviderConfig
	client *http.Client
}

func NewOpenAIProvider(config ProviderConfig) *OpenAIProvider {
	if config.BaseURL == "" {
		config.BaseURL = "https://api.openai.com/v1"
	}
	return &OpenAIProvider{
		config: config,
		client: &http.Client{},
	}
}

func (p *OpenAIProvider) Name() string {
	return "openai"
}

func (p *OpenAIProvider) Complete(req CompletionRequest) (*CompletionResponse, error) {
	openAIReq := p.toOpenAIRequest(req)
	body, err := json.Marshal(openAIReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", p.config.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.config.APIKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	var openAIResp struct {
		Choices []struct {
			Message struct {
				Role         string     `json:"role"`
				Content      string     `json:"content"`
				ToolCalls    []ToolCall `json:"tool_calls"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Usage *Usage `json:"usage"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&openAIResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	result := &CompletionResponse{
		FinishReason: openAIResp.Choices[0].FinishReason,
		Usage:        openAIResp.Usage,
	}

	if len(openAIResp.Choices) > 0 {
		result.Content = openAIResp.Choices[0].Message.Content
		result.ToolCalls = openAIResp.Choices[0].Message.ToolCalls
	}

	return result, nil
}

func (p *OpenAIProvider) CompleteStream(req CompletionRequest, events chan<- StreamEvent) error {
	defer close(events)

	openAIReq := p.toOpenAIRequest(req)
	openAIReq["stream"] = true
	body, err := json.Marshal(openAIReq)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", p.config.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.config.APIKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
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

		var chunk struct {
			Choices []struct {
				Delta struct {
					Role      string     `json:"role"`
					Content   string     `json:"content"`
					ToolCalls []ToolCall `json:"tool_calls"`
				} `json:"delta"`
				FinishReason string `json:"finish_reason"`
			} `json:"choices"`
			Usage *Usage `json:"usage"`
		}

		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		if len(chunk.Choices) == 0 {
			continue
		}

		delta := chunk.Choices[0].Delta
		if delta.Content != "" {
			events <- StreamEvent{Type: StreamEventText, Delta: delta.Content}
		}

		if len(delta.ToolCalls) > 0 {
			events <- StreamEvent{Type: StreamEventToolCall, ToolCalls: delta.ToolCalls}
		}

		if chunk.Choices[0].FinishReason != "" {
			events <- StreamEvent{
				Type:         StreamEventDone,
				FinishReason: chunk.Choices[0].FinishReason,
				Usage:        chunk.Usage,
			}
		}
	}

	return scanner.Err()
}

func (p *OpenAIProvider) toOpenAIRequest(req CompletionRequest) map[string]any {
	messages := make([]map[string]any, 0, len(req.Messages))
	for _, msg := range req.Messages {
		m := map[string]any{
			"role": string(msg.Role),
		}
		if msg.Content != "" {
			m["content"] = msg.Content
		}
		if len(msg.ToolCalls) > 0 {
			m["tool_calls"] = msg.ToolCalls
		}
		if msg.ToolCallID != "" {
			m["tool_call_id"] = msg.ToolCallID
		}
		messages = append(messages, m)
	}

	result := map[string]any{
		"model":    req.Model,
		"messages": messages,
	}

	if req.MaxTokens > 0 {
		result["max_tokens"] = req.MaxTokens
	}
	if req.Temperature > 0 {
		result["temperature"] = req.Temperature
	}
	if len(req.Tools) > 0 {
		result["tools"] = req.Tools
	}

	return result
}
