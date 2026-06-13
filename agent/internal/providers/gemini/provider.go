// Package gemini implements the Google Gemini provider.
// Uses OpenAI-compatible API at https://generativelanguage.googleapis.com/v1beta/openai
package gemini

import (
	"context"
	"encoding/json"

	"github.com/ChxisB/spectre-proxy/agent/internal/config"
	"github.com/ChxisB/spectre-proxy/agent/internal/protocol"
	"github.com/ChxisB/spectre-proxy/agent/internal/providers"
	"github.com/ChxisB/spectre-proxy/agent/internal/providers/openai"
)

const (
	providerName = "gemini"
	defaultBase  = "https://generativelanguage.googleapis.com/v1beta/openai"
)

// Provider wraps the OpenAI transport for Gemini.
type Provider struct {
	transport *openai.Transport
	metadata  providers.ProviderMetadata
}

func metadata() providers.ProviderMetadata {
	return providers.ProviderMetadata{
		ID:              providerName,
		Name:            "Google Gemini",
		Description:     "Google Gemini API via OpenAI-compatible endpoint",
		APIType:         "openai",
		BaseURL:         defaultBase,
		RequiresAPIKey:  true,
		Capabilities:    []providers.ProviderCapability{providers.CapabilityStreaming, providers.CapabilityTools, providers.CapabilityThinking, providers.CapabilityVision, providers.CapabilitySystemPrompt, providers.CapabilityModelListing},
		DefaultModels:   []string{"gemini/gemini-2.5-pro", "gemini/gemini-2.5-flash", "gemini/gemini-1.5-pro"},
		ModelPrefix:     "gemini/",
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