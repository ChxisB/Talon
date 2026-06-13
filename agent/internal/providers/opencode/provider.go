// Package opencode implements the OpenCode provider.
// OpenCode provides two API endpoints:
//   - Zen: https://opencode.ai/zen/v1 (standard OpenAI-compatible)
//   - Go:  https://opencode.ai/zen/go/v1 (OpenAI-compatible, skip tool validation for DeepSeek models)
package opencode

import (
	"context"
	"encoding/json"

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
	metadata  providers.ProviderMetadata
}

// zenMetadata returns metadata for OpenCode Zen.
func zenMetadata() providers.ProviderMetadata {
	return providers.ProviderMetadata{
		ID:              providerNameZen,
		Name:            "OpenCode Zen",
		Description:     "OpenCode Zen API - OpenAI-compatible endpoint with curated models",
		APIType:         "openai",
		BaseURL:         defaultBaseZen,
		RequiresAPIKey:  true,
		Capabilities:    []providers.ProviderCapability{providers.CapabilityStreaming, providers.CapabilityTools, providers.CapabilityThinking, providers.CapabilityVision, providers.CapabilitySystemPrompt, providers.CapabilityModelListing, providers.CapabilityStreamingTools},
		DefaultModels:   []string{"opencode/gpt-5", "opencode/claude-sonnet-4", "opencode/gemini-2.5-pro"},
		ModelPrefix:     "opencode/",
		SupportsThinking: true,
	}
}

// goMetadata returns metadata for OpenCode Go.
func goMetadata() providers.ProviderMetadata {
	return providers.ProviderMetadata{
		ID:              providerNameGo,
		Name:            "OpenCode Go",
		Description:     "OpenCode Go API - OpenAI-compatible endpoint with skip tool validation for DeepSeek models",
		APIType:         "openai",
		BaseURL:         defaultBaseGo,
		RequiresAPIKey:  true,
		Capabilities:    []providers.ProviderCapability{providers.CapabilityStreaming, providers.CapabilityTools, providers.CapabilityThinking, providers.CapabilityVision, providers.CapabilitySystemPrompt, providers.CapabilityModelListing, providers.CapabilityStreamingTools},
		DefaultModels:   []string{"opencode_go/gpt-5", "opencode_go/deepseek-v4-pro", "opencode_go/claude-sonnet-4"},
		ModelPrefix:     "opencode_go/",
		SupportsThinking: true,
	}
}

// NewZen creates a new OpenCode Zen provider.
func NewZen(cfg providers.ProviderConfig, _ *config.Settings) (providers.Provider, error) {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = defaultBaseZen
	}
	return &Provider{
		name: providerNameZen,
		metadata: zenMetadata(),
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
		metadata: goMetadata(),
		transport: openai.NewTransport(openai.Config{
			Name:               providerNameGo,
			BaseURL:            baseURL,
			APIKey:             cfg.APIKey,
			SkipToolValidation: true,
		}),
	}, nil
}

func (p *Provider) ID() string { return p.name }

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