// Package wafer implements the Wafer provider.
// Uses Anthropic Messages API at https://api.wafer.ai/v1
package wafer

import (
	"context"
	"encoding/json"

	"github.com/ChxisB/spectre-proxy/agent/internal/config"
	"github.com/ChxisB/spectre-proxy/agent/internal/protocol"
	"github.com/ChxisB/spectre-proxy/agent/internal/providers"
	"github.com/ChxisB/spectre-proxy/agent/internal/providers/anthropic"
)

const (
	providerName = "wafer"
	defaultBase  = "https://api.wafer.ai/v1"
)

type Provider struct {
	transport *anthropic.Transport
	metadata  providers.ProviderMetadata
}

func metadata() providers.ProviderMetadata {
	return providers.ProviderMetadata{
		ID:              providerName,
		Name:            "Wafer",
		Description:     "Wafer API - Anthropic Messages API compatible",
		APIType:         "anthropic",
		BaseURL:         defaultBase,
		RequiresAPIKey:  true,
		Capabilities:    []providers.ProviderCapability{providers.CapabilityStreaming, providers.CapabilityTools, providers.CapabilityThinking, providers.CapabilitySystemPrompt, providers.CapabilityModelListing},
		DefaultModels:   []string{"wafer/wafer-1", "wafer/wafer-2"},
		ModelPrefix:     "wafer/",
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