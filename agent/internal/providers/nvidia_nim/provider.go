// Package nvidia_nim implements the NVIDIA NIM provider.
// Uses OpenAI-compatible API at https://integrate.api.nvidia.com/v1
package nvidia_nim

import (
	"context"
	"encoding/json"

	"github.com/ChxisB/spectre-proxy/agent/internal/config"
	"github.com/ChxisB/spectre-proxy/agent/internal/protocol"
	"github.com/ChxisB/spectre-proxy/agent/internal/providers"
	"github.com/ChxisB/spectre-proxy/agent/internal/providers/openai"
)

const (
	providerName = "nvidia_nim"
	defaultBase  = "https://integrate.api.nvidia.com/v1"
)

type Provider struct {
	transport *openai.Transport
	metadata  providers.ProviderMetadata
}

func metadata() providers.ProviderMetadata {
	return providers.ProviderMetadata{
		ID:              providerName,
		Name:            "NVIDIA NIM",
		Description:     "NVIDIA NIM API - OpenAI-compatible (NVIDIA optimized models)",
		APIType:         "openai",
		BaseURL:         defaultBase,
		RequiresAPIKey:  true,
		Capabilities:    []providers.ProviderCapability{providers.CapabilityStreaming, providers.CapabilityTools, providers.CapabilityVision, providers.CapabilitySystemPrompt, providers.CapabilityModelListing},
		DefaultModels:   []string{"nvidia_nim/nvidia/nemotron-3-ultra", "nvidia_nim/meta/llama-3.1-405b-instruct"},
		ModelPrefix:     "nvidia_nim/",
		SupportsThinking: false,
	}
}

func New(cfg providers.ProviderConfig, _ *config.Settings) (providers.Provider, error) {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = defaultBase
	}
	return &Provider{
		metadata: metadata(),
		transport: openai.NewTransport(openai.Config{
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