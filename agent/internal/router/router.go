// Package router resolves Claude model names to provider/model pairs.
package router

import (
	"strings"

	"github.com/ChxisB/spectre-proxy/agent/internal/config"
)

// ProviderAliases maps non-canonical provider prefixes (as used by some
// upstream APIs like OpenRouter's model listing) to the canonical provider
// IDs registered in the provider catalog.
var ProviderAliases = map[string]string{
	"openrouter": "open_router",
}

// ResolveProviderID maps a user-supplied provider prefix to a canonical
// provider ID, handling aliases like openrouter → open_router.
func ResolveProviderID(prefix string, supported map[string]bool) string {
	if supported[prefix] {
		return prefix
	}
	if alias, ok := ProviderAliases[prefix]; ok {
		return alias
	}
	return ""
}

// ResolvedModel holds the result of model resolution.
type ResolvedModel struct {
	OriginalModel    string
	ProviderID       string
	ProviderModel    string
	ProviderModelRef string
	ThinkingEnabled  bool
}

// ModelRouter resolves incoming Claude model names to configured provider/model pairs.
type ModelRouter struct {
	settings *config.Settings
}

// NewModelRouter creates a new model router.
func NewModelRouter(settings *config.Settings) *ModelRouter {
	return &ModelRouter{settings: settings}
}

// Resolve resolves a Claude model name to a provider/model pair.
func (r *ModelRouter) Resolve(claudeModel string) *ResolvedModel {
	supported := config.SupportedProviderIDs()

	// Try direct "provider/model" format
	parts := strings.SplitN(claudeModel, "/", 2)
	if len(parts) == 2 {
		providerID := ResolveProviderID(parts[0], supported)
		if providerID != "" {
			return &ResolvedModel{
				OriginalModel:    claudeModel,
				ProviderID:       providerID,
				ProviderModel:    parts[1],
				ProviderModelRef: claudeModel,
				ThinkingEnabled:  r.settings.ResolveThinking(parts[1]),
			}
		}
	}

	// Try env-configured model mapping
	modelRef := r.settings.ResolveModel(claudeModel)
	if modelRef != "" && modelRef != claudeModel {
		providerID := config.ParseProviderType(modelRef)
		providerModel := config.ParseModelName(modelRef)
		return &ResolvedModel{
			OriginalModel:    claudeModel,
			ProviderID:       providerID,
			ProviderModel:    providerModel,
			ProviderModelRef: modelRef,
			ThinkingEnabled:  r.settings.ResolveThinking(providerModel),
		}
	}

	// Fallback: default model
	defaultRef := r.settings.ResolveModel("")
	if defaultRef == "" {
		defaultRef = "openrouter/anthropic/claude-sonnet-4"
	}
	providerID := config.ParseProviderType(defaultRef)
	providerModel := config.ParseModelName(defaultRef)

	return &ResolvedModel{
		OriginalModel:    claudeModel,
		ProviderID:       providerID,
		ProviderModel:    providerModel,
		ProviderModelRef: defaultRef,
		ThinkingEnabled:  r.settings.ResolveThinking(providerModel),
	}
}
