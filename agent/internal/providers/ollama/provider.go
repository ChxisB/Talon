// Package ollama implements the Ollama provider.
// Uses Anthropic Messages API at http://localhost:11434
package ollama

import (
	"context"

	"github.com/ChxisB/spectre-proxy/agent/internal/config"
	"github.com/ChxisB/spectre-proxy/agent/internal/protocol"
	"github.com/ChxisB/spectre-proxy/agent/internal/providers"
	"github.com/ChxisB/spectre-proxy/agent/internal/providers/anthropic"
)

const (
	providerName = "ollama"
	defaultBase  = "http://localhost:11434"
)

// Provider wraps the Anthropic transport for Ollama.
type Provider struct {
	transport *anthropic.Transport
}

func New(cfg providers.ProviderConfig, _ *config.Settings) (providers.Provider, error) {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = defaultBase
	}
	return &Provider{
		transport: anthropic.NewTransport(anthropic.Config{
			Name:    providerName,
			BaseURL: baseURL,
			APIKey:  cfg.APIKey, // Ollama doesn't require API key, but accept if provided
		}),
	}, nil
}

func (p *Provider) ID() string { return providerName }

func (p *Provider) StreamResponse(ctx context.Context, req *protocol.MessagesRequest, inputTokens int, thinking bool) (<-chan protocol.SSEEvent, error) {
	return p.transport.StreamResponse(ctx, req, inputTokens, thinking)
}

func (p *Provider) ListModels(ctx context.Context) ([]string, error) {
	return p.transport.ListModels(ctx)
}

func (p *Provider) CheckHealth(ctx context.Context) error {
	return p.transport.CheckHealth(ctx)
}