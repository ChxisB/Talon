// Package opencode implements the OpenCode provider.
// OpenCode provides two API endpoints:
//   - Zen: https://opencode.ai/zen/v1 (standard OpenAI-compatible)
//   - Go:  https://opencode.ai/zen/go/v1 (OpenAI-compatible, skip tool validation for DeepSeek models)
package opencode

import (
	"context"

	"github.com/ChxisB/spectre-proxy/agent/internal/config"
	"github.com/ChxisB/spectre-proxy/agent/internal/protocol"
	"github.com/ChxisB/spectre-proxy/agent/internal/providers"
	"github.com/ChxisB/spectre-proxy/agent/internal/providers/openai"
)

const (
	providerNameZen = "opencode"
	providerNameGo  = "opencode_go"
	defaultBaseZen  = "https://opencode.ai/zen/v1"
	defaultBaseGo   = "https://opencode.ai/zen/go/v1"
)

// Provider wraps the OpenAI transport for OpenCode.
type Provider struct {
	transport *openai.Transport
	name      string
}

// NewZen creates a new OpenCode Zen provider.
func NewZen(cfg providers.ProviderConfig, _ *config.Settings) (providers.Provider, error) {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = defaultBaseZen
	}
	return &Provider{
		name: providerNameZen,
		transport: openai.NewTransport(openai.Config{
			Name:               providerNameZen,
			BaseURL:            baseURL,
			APIKey:             cfg.APIKey,
			SkipToolValidation: false,
		}),
	}, nil
}

// NewGo creates a new OpenCode Go provider.
// DeepSeek models on opencode_go generate tool calls with inconsistent schemas
// that fail strict required-field checking. SkipToolValidation lets those
// tool calls through so the agent can handle them.
func NewGo(cfg providers.ProviderConfig, _ *config.Settings) (providers.Provider, error) {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = defaultBaseGo
	}
	return &Provider{
		name: providerNameGo,
		transport: openai.NewTransport(openai.Config{
			Name:               providerNameGo,
			BaseURL:            baseURL,
			APIKey:             cfg.APIKey,
			SkipToolValidation: true,
		}),
	}, nil
}

func (p *Provider) ID() string { return p.name }

func (p *Provider) StreamResponse(ctx context.Context, req *protocol.MessagesRequest, inputTokens int, thinking bool) (<-chan protocol.SSEEvent, error) {
	return p.transport.StreamResponse(ctx, req, inputTokens, thinking)
}

func (p *Provider) ListModels(ctx context.Context) ([]string, error) {
	return p.transport.ListModels(ctx)
}

func (p *Provider) CheckHealth(ctx context.Context) error {
	return p.transport.CheckHealth(ctx)
}