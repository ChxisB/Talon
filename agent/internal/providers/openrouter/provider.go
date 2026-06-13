// Package openrouter implements the OpenRouter provider.
// Uses Anthropic Messages API at https://openrouter.ai/api/v1
package openrouter

import (
	"context"
	"encoding/json"

	"github.com/ChxisB/spectre-proxy/agent/internal/config"
	"github.com/ChxisB/spectre-proxy/agent/internal/protocol"
	"github.com/ChxisB/spectre-proxy/agent/internal/providers"
	"github.com/ChxisB/spectre-proxy/agent/internal/providers/anthropic"
)

const (
	providerName = "open_router"
	defaultBase  = "https://openrouter.ai/api/v1"
)

type Provider struct {
	transport *anthropic.Transport
	metadata  providers.ProviderMetadata
}

func metadata() providers.ProviderMetadata {
	return providers.ProviderMetadata{
		ID:              providerName,
		Name:            "OpenRouter",
		Description:     "OpenRouter - Unified API for 100+ LLMs via Anthropic Messages API",
		APIType:         "anthropic",
		BaseURL:         defaultBase,
		RequiresAPIKey:  true,
		Capabilities:    []providers.ProviderCapability{providers.CapabilityStreaming, providers.CapabilityTools, providers.CapabilityThinking, providers.CapabilityVision, providers.CapabilitySystemPrompt, providers.CapabilityModelListing, providers.CapabilityStreamingTools},
		DefaultModels:   []string{"open_router/anthropic/claude-sonnet-4", "open_router/anthropic/claude-opus-4", "open_router/google/gemini-2.5-pro"},
		ModelPrefix:     "open_router/",
		SupportsThinking: true,
	}
}

func New(cfg providers.ProviderConfig, _ *config.Settings) (providers.Provider, error) {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = defaultBase
	}
	return &Provider{
		metadata: metadata(),
		transport: anthropic.NewTransport(anthropic.Config{
			Name:    providerName,
			BaseURL: baseURL,
			APIKey:  cfg.APIKey,
		}),
	}, nil
}

func (p *Provider) ID() string { return providerName }

func (p *Provider) Metadata() providers.ProviderMetadata { return p.metadata }

func (p *Provider) ProtocolSupport() providers.ProtocolSupport {
	return providers.ProtocolSupport{Anthropic: true, Responses: true, GenAI: true}
}

func (p *Provider) StreamResponses(ctx context.Context, rawReq json.RawMessage, resolvedModel string) (<-chan []byte, error) {
	return p.transport.StreamResponses(ctx, rawReq, resolvedModel)
}

func (p *Provider) StreamGenAI(ctx context.Context, rawReq json.RawMessage, resolvedModel string) (<-chan []byte, error) {
	return p.transport.StreamGenAI(ctx, rawReq, resolvedModel)
}

func (p *Provider) StreamAnthropic(ctx context.Context, req *protocol.MessagesRequest, inputTokens int, thinking bool) (<-chan protocol.SSEEvent, error) {
	return p.transport.StreamAnthropic(ctx, req, inputTokens, thinking)
}

func (p *Provider) ListModels(ctx context.Context) ([]string, error) {
	return p.transport.ListModels(ctx)
}

func (p *Provider) CheckHealth(ctx context.Context) error {
	return p.transport.CheckHealth(ctx)
}
