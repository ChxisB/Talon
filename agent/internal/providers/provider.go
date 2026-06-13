// Package providers defines the provider interface and registry for the
// Spectre Proxy proxy.
package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ChxisB/spectre-proxy/agent/internal/config"
	"github.com/ChxisB/spectre-proxy/agent/internal/protocol"
)

// ProviderConfig holds configuration for a provider instance.
type ProviderConfig struct {
	APIKey         string
	BaseURL        string
	RateLimit      int
	RateWindow     time.Duration
	MaxConcurrency int
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	ConnectTimeout time.Duration
	EnableThinking bool
	Proxy          string
}

// DefaultProviderConfig returns sensible defaults.
func DefaultProviderConfig() ProviderConfig {
	return ProviderConfig{
		RateLimit:      0,
		RateWindow:     60 * time.Second,
		MaxConcurrency: 5,
		ReadTimeout:    300 * time.Second,
		WriteTimeout:   10 * time.Second,
		ConnectTimeout: 10 * time.Second,
		EnableThinking: true,
	}
}

// ProviderCapability represents a capability that a provider may support.
type ProviderCapability string

const (
	// CapabilityStreaming indicates the provider supports streaming responses.
	CapabilityStreaming ProviderCapability = "streaming"
	// CapabilityTools indicates the provider supports tool/function calling.
	CapabilityTools ProviderCapability = "tools"
	// CapabilityThinking indicates the provider supports thinking/reasoning blocks.
	CapabilityThinking ProviderCapability = "thinking"
	// CapabilityVision indicates the provider supports image input.
	CapabilityVision ProviderCapability = "vision"
	// CapabilitySystemPrompt indicates the provider supports system prompts.
	CapabilitySystemPrompt ProviderCapability = "system_prompt"
	// CapabilityModelListing indicates the provider supports /models endpoint.
	CapabilityModelListing ProviderCapability = "model_listing"
	// CapabilityStreamingTools indicates the provider supports streaming tool calls.
	CapabilityStreamingTools ProviderCapability = "streaming_tools"
)

// ProviderMetadata holds metadata about a provider's capabilities and configuration.
type ProviderMetadata struct {
	ID              string
	Name            string
	Description     string
	APIType         string // "anthropic" or "openai"
	BaseURL         string
	RequiresAPIKey  bool
	Capabilities    []ProviderCapability
	DefaultModels   []string
	ModelPrefix     string // e.g., "open_router/", "opencode/"
	SupportsThinking bool
}

// ErrProtocolNotSupported is returned when a provider doesn't implement
// a specific protocol (e.g. StreamResponses, StreamGenAI).
type ErrProtocolNotSupported string

func (e ErrProtocolNotSupported) Error() string {
	return fmt.Sprintf("protocol not supported: %s", string(e))
}

// ProtocolSupport describes which CLI protocols this provider implements.
type ProtocolSupport struct {
	Anthropic bool // Claude Code CLI
	Responses bool // OpenAI Codex CLI
	GenAI     bool // Google Gemini CLI
}

// Provider is the interface all providers must implement.
// Each provider can support one or more CLI protocols.
type Provider interface {
	// ID returns the unique provider identifier (e.g. "open_router", "ollama").
	ID() string

	// Metadata returns provider metadata including capabilities.
	Metadata() ProviderMetadata

	// ProtocolSupport returns which CLI protocols this provider implements.
	ProtocolSupport() ProtocolSupport

	// StreamAnthropic sends an Anthropic Messages API request (used by Claude Code).
	// Returns Anthropic-format SSE events.
	StreamAnthropic(ctx context.Context, req *protocol.MessagesRequest, inputTokens int, thinking bool) (<-chan protocol.SSEEvent, error)

	// StreamResponses sends an OpenAI Responses API request (used by Codex CLI).
	// Returns pre-formatted SSE bytes in Responses API format.
	// resolvedModel is the provider-specific model name after routing.
	StreamResponses(ctx context.Context, rawReq json.RawMessage, resolvedModel string) (<-chan []byte, error)

	// StreamGenAI sends a Google GenAI API request (used by Gemini CLI).
	// Returns pre-formatted SSE bytes in GenAI format.
	// resolvedModel is the provider-specific model name after routing.
	StreamGenAI(ctx context.Context, rawReq json.RawMessage, resolvedModel string) (<-chan []byte, error)

	// ListModels returns the available models from this provider.
	ListModels(ctx context.Context) ([]string, error)

	// CheckHealth returns nil if the provider is reachable.
	CheckHealth(ctx context.Context) error
}

// DefaultProvider implements default stubs for optional protocol methods.
// Embed this in a provider struct and override the methods you support.
//
// Usage:
//
//	type MyProvider struct {
//	    providers.DefaultProvider
//	    transport *anthropic.Transport
//	}
//
//	func (p *MyProvider) StreamAnthropic(...) { return p.transport.StreamAnthropic(...) }
//	// StreamResponses and StreamGenAI use the default "not supported" stub.
type DefaultProvider struct{}

func (DefaultProvider) ProtocolSupport() ProtocolSupport {
	return ProtocolSupport{Anthropic: false, Responses: false, GenAI: false}
}

func (DefaultProvider) StreamResponses(ctx context.Context, rawReq json.RawMessage, resolvedModel string) (<-chan []byte, error) {
	return nil, ErrProtocolNotSupported("OpenAI Responses API")
}

func (DefaultProvider) StreamGenAI(ctx context.Context, rawReq json.RawMessage, resolvedModel string) (<-chan []byte, error) {
	return nil, ErrProtocolNotSupported("Google GenAI API")
}

// Registry manages provider instances.
type Registry struct {
	factories map[string]ProviderFactory
	providers map[string]Provider
}

// NewRegistry creates an empty provider registry.
func NewRegistry() *Registry {
	return &Registry{
		factories: make(map[string]ProviderFactory),
		providers: make(map[string]Provider),
	}
}

// Register adds a provider factory.
func (r *Registry) Register(id string, factory ProviderFactory) {
	r.factories[id] = factory
}

// Get returns or creates a provider instance.
func (r *Registry) Get(id string, cfg ProviderConfig, settings *config.Settings) (Provider, error) {
	if p, ok := r.providers[id]; ok {
		return p, nil
	}

	factory, ok := r.factories[id]
	if !ok {
		return nil, fmt.Errorf("unknown provider: %s", id)
	}

	p, err := factory(cfg, settings)
	if err != nil {
		return nil, err
	}

	r.providers[id] = p
	return p, nil
}

// HasProvider returns true if the provider ID is registered.
func (r *Registry) HasProvider(id string) bool {
	_, ok := r.factories[id]
	return ok
}

// GetMetadata returns the metadata for a provider by creating a temporary instance.
// Returns nil if the provider is not registered.
func (r *Registry) GetMetadata(id string) *ProviderMetadata {
	factory, ok := r.factories[id]
	if !ok {
		return nil
	}
	// Create a temporary instance to get metadata
	p, err := factory(DefaultProviderConfig(), nil)
	if err != nil {
		return nil
	}
	meta := p.Metadata()
	return &meta
}

// ListProviderMetadata returns metadata for all registered providers.
func (r *Registry) ListProviderMetadata() []ProviderMetadata {
	var result []ProviderMetadata
	for id := range r.factories {
		if meta := r.GetMetadata(id); meta != nil {
			result = append(result, *meta)
		}
	}
	return result
}
