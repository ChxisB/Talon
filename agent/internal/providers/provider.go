// Package providers defines the provider interface and registry for the
// Spectre Proxy proxy. Ported from free-claude-code/providers/base.py and registry.py.
package providers

import (
	"context"
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

// Provider is the interface all providers must implement.
type Provider interface {
	// ID returns the unique provider identifier (e.g. "open_router", "ollama").
	ID() string

	// StreamResponse sends a messages request and returns an iterator of SSE events.
	StreamResponse(ctx context.Context, req *protocol.MessagesRequest, inputTokens int, thinking bool) (<-chan protocol.SSEEvent, error)

	// ListModels returns the available models from this provider.
	ListModels(ctx context.Context) ([]string, error)

	// CheckHealth returns nil if the provider is reachable.
	CheckHealth(ctx context.Context) error
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
