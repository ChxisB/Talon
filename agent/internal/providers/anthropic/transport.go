// Package anthropic provides a shared Anthropic Messages API transport.
// Any provider with a native /v1/messages endpoint (OpenRouter, DeepSeek,
// Kimi, Wafer, Fireworks, Z.ai, etc.) can use this transport.
package anthropic

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

	"github.com/ChxisB/spectre-proxy/agent/internal/protocol"
)

func safePrefix(s string) string {
	if len(s) >= 12 {
		return s[:12]
	}
	return s
}

// Config holds configuration for an Anthropic Messages-compatible provider.
type Config struct {
	Name    string
	BaseURL string
	APIKey  string
}

// Transport implements the Provider interface for Anthropic Messages APIs.
type Transport struct {
	cfg    Config
	client *http.Client
}

// NewTransport creates a new Anthropic Messages transport.
func NewTransport(cfg Config) *Transport {
	return &Transport{
		cfg: cfg,
		client: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}
}

// ID returns the provider identifier.
func (t *Transport) ID() string { return t.cfg.Name }

// StreamResponse sends a messages request and returns SSE events.
func (t *Transport) StreamResponse(ctx context.Context, req *protocol.MessagesRequest, inputTokens int, thinking bool) (<-chan protocol.SSEEvent, error) {
	ch := make(chan protocol.SSEEvent, 128)

	bodyMap := map[string]any{
		"model":      req.Model,
		"stream":     true,
		"max_tokens": 8192,
	}

	if req.System != nil {
		bodyMap["system"] = json.RawMessage(req.System)
	}
	if len(req.Messages) > 0 {
		bodyMap["messages"] = req.Messages
	}
	if req.Temperature != nil {
		bodyMap["temperature"] = *req.Temperature
	}
	if req.TopP != nil {
		bodyMap["top_p"] = *req.TopP
	}
	if req.MaxTokens > 0 {
		bodyMap["max_tokens"] = req.MaxTokens
	}
	if len(req.Tools) > 0 {
		// Enforce strict schema compliance — prevents models from generating
		// tool calls with missing required fields (e.g. Bash without command).
		toolsWithStrict := make([]map[string]any, len(req.Tools))
		for i, tool := range req.Tools {
			toolsWithStrict[i] = map[string]any{
				"name":        tool.Name,
				"description": tool.Description,
				"input_schema": tool.InputSchema,
				"strict":      true,
			}
		}
		bodyMap["tools"] = toolsWithStrict
	}

	body, err := json.Marshal(bodyMap)
	if err != nil {
		close(ch)
		return ch, fmt.Errorf("%s: marshal: %w", t.cfg.Name, err)
	}

	log.Printf("[debug] %s request body: %s", t.cfg.Name, string(body))

	// Base URL may or may not include /v1. Check to avoid double path.
	messagesPath := "/v1/messages"
	if strings.HasSuffix(strings.TrimRight(t.cfg.BaseURL, "/"), "/v1") {
		messagesPath = "/messages"
	}
	apiURL := strings.TrimRight(t.cfg.BaseURL, "/") + messagesPath
	httpReq, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(body))
	if err != nil {
		close(ch)
		return ch, fmt.Errorf("%s: create request: %w", t.cfg.Name, err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	if t.cfg.APIKey != "" {
		httpReq.Header.Set("x-api-key", t.cfg.APIKey)
	}

	resp, err := t.client.Do(httpReq)
	if err != nil {
		close(ch)
		return ch, fmt.Errorf("%s: request failed: %w", t.cfg.Name, err)
	}

	if resp.StatusCode != 200 {
		// Retry once on transient errors (502/503/504)
		if resp.StatusCode == 502 || resp.StatusCode == 503 || resp.StatusCode == 504 {
			resp.Body.Close()
			time.Sleep(1 * time.Second)
			// Re-create the request since the body was consumed by the first Do
			httpReq, _ = http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(body))
			httpReq.Header.Set("Content-Type", "application/json")
			httpReq.Header.Set("anthropic-version", "2023-06-01")
			if t.cfg.APIKey != "" {
				httpReq.Header.Set("x-api-key", t.cfg.APIKey)
			}
			resp, err = t.client.Do(httpReq)
			if err != nil {
				close(ch)
				return ch, fmt.Errorf("%s: retry request failed: %w", t.cfg.Name, err)
			}
		}
	}

	log.Printf("[debug] %s request to %s: status=%d, key_prefix=%q", t.cfg.Name, apiURL, resp.StatusCode, safePrefix(t.cfg.APIKey))

	if resp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		close(ch)
		return ch, fmt.Errorf("%s: status %d: %s", t.cfg.Name, resp.StatusCode, strings.TrimSpace(string(bodyBytes)))
	}

	go func() {
		defer resp.Body.Close()
		defer close(ch)

		scanner := bufio.NewScanner(resp.Body)
		scanner.Buffer(make([]byte, 0, 64*1024), 512*1024)

		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "event: ") {
				eventType := strings.TrimPrefix(line, "event: ")
				if !scanner.Scan() {
					break
				}
				dataLine := scanner.Text()
				if strings.HasPrefix(dataLine, "data: ") {
					data := strings.TrimPrefix(dataLine, "data: ")
					select {
					case ch <- protocol.SSEEvent{Type: eventType, Data: []byte(data)}:
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}()

	return ch, nil
}

// ListModels returns available models (delegates to provider's model endpoint).
func (t *Transport) ListModels(ctx context.Context) ([]string, error) {
	apiURL := strings.TrimRight(t.cfg.BaseURL, "/") + "/models"
	httpReq, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	if t.cfg.APIKey != "" {
		httpReq.Header.Set("x-api-key", t.cfg.APIKey)
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
		httpReq.Header.Set("x-api-key", t.cfg.APIKey)
	}

	resp, err := t.client.Do(httpReq)
	if err != nil {
		return err
	}
	resp.Body.Close()

	if resp.StatusCode >= 500 {
		return fmt.Errorf("provider returned status %d", resp.StatusCode)
	}
	return nil
}
