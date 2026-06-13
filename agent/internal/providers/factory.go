// Package providers defines the provider interface and registry for the
// Spectre Proxy proxy.
package providers

import (
	"github.com/ChxisB/spectre-proxy/agent/internal/config"
)

// ProviderFactory creates a provider from config.
type ProviderFactory func(cfg ProviderConfig, settings *config.Settings) (Provider, error)

// Registry holds the mapping of provider IDs to their factories.
var ProviderFactories = map[string]ProviderFactory{}

// RegisterProvider registers a provider factory.
func RegisterProvider(id string, factory ProviderFactory) {
	ProviderFactories[id] = factory
}

// GetFactory returns the factory for the given provider ID.
func GetFactory(id string) (ProviderFactory, bool) {
	factory, ok := ProviderFactories[id]
	return factory, ok
}

// CreateProvider creates a provider instance using the registered factory.
func CreateProvider(id string, cfg ProviderConfig, settings *config.Settings) (Provider, error) {
	factory, ok := ProviderFactories[id]
	if !ok {
		return nil, ErrUnknownProvider(id)
	}
	return factory(cfg, settings)
}

// ErrUnknownProvider is returned when a provider ID is not registered.
type ErrUnknownProvider string

func (e ErrUnknownProvider) Error() string {
	return "unknown provider: " + string(e)
}