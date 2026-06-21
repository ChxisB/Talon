package llm

import (
	"fmt"
	"os"
	"strings"
)

func GetProvider(modelID string) (Provider, *ProviderConfig, error) {
	// Determine provider from model ID
	switch {
	case strings.HasPrefix(modelID, "gpt-"), strings.HasPrefix(modelID, "o1"), strings.HasPrefix(modelID, "o3-"):
		apiKey := os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			return nil, nil, fmt.Errorf("OPENAI_API_KEY not set")
		}
		config := ProviderConfig{
			APIKey:  apiKey,
			Model:   modelID,
			BaseURL: os.Getenv("OPENAI_BASE_URL"),
		}
		return NewOpenAIProvider(config), &config, nil

	case strings.HasPrefix(modelID, "claude-"):
		apiKey := os.Getenv("ANTHROPIC_API_KEY")
		if apiKey == "" {
			return nil, nil, fmt.Errorf("ANTHROPIC_API_KEY not set")
		}
		config := ProviderConfig{
			APIKey:  apiKey,
			Model:   modelID,
			BaseURL: os.Getenv("ANTHROPIC_BASE_URL"),
		}
		return NewAnthropicProvider(config), &config, nil

	case strings.HasPrefix(modelID, "gemini-"):
		apiKey := os.Getenv("GOOGLE_API_KEY")
		if apiKey == "" {
			return nil, nil, fmt.Errorf("GOOGLE_API_KEY not set")
		}
		// For now, route Gemini through OpenAI-compatible format
		config := ProviderConfig{
			APIKey:  apiKey,
			Model:   modelID,
			BaseURL: "https://generativelanguage.googleapis.com/v1beta/openai",
		}
		return NewOpenAIProvider(config), &config, nil

	default:
		// Try OpenAI-compatible (for OpenRouter, Together, etc.)
		apiKey := os.Getenv("OPENAI_API_KEY")
		baseURL := os.Getenv("OPENAI_BASE_URL")
		if apiKey == "" {
			return nil, nil, fmt.Errorf("no API key found for model: %s", modelID)
		}
		config := ProviderConfig{
			APIKey:  apiKey,
			Model:   modelID,
			BaseURL: baseURL,
		}
		return NewOpenAIProvider(config), &config, nil
	}
}

func ListAvailableProviders() []map[string]any {
	providers := []map[string]any{}
	if key := os.Getenv("OPENAI_API_KEY"); key != "" {
		providers = append(providers, map[string]any{
			"id": "openai", "name": "OpenAI", "enabled": true,
			"description": "OpenAI models (GPT-4, GPT-4o, o1, etc.)",
		})
	}
	if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
		providers = append(providers, map[string]any{
			"id": "anthropic", "name": "Anthropic", "enabled": true,
			"description": "Anthropic models (Claude 3.5 Sonnet, Opus, etc.)",
		})
	}
	if key := os.Getenv("GOOGLE_API_KEY"); key != "" {
		providers = append(providers, map[string]any{
			"id": "google", "name": "Google", "enabled": true,
			"description": "Google models (Gemini 1.5, Gemini 2.0, etc.)",
		})
	}
	return providers
}

func ListAvailableModels() []map[string]any {
	models := []map[string]any{}
	if os.Getenv("OPENAI_API_KEY") != "" {
		models = append(models,
			map[string]any{"id": "gpt-4o", "providerID": "openai", "name": "GPT-4o", "enabled": true},
			map[string]any{"id": "gpt-4o-mini", "providerID": "openai", "name": "GPT-4o Mini", "enabled": true},
			map[string]any{"id": "o1", "providerID": "openai", "name": "O1", "enabled": true},
			map[string]any{"id": "o3-mini", "providerID": "openai", "name": "o3-mini", "enabled": true},
		)
	}
	if os.Getenv("ANTHROPIC_API_KEY") != "" {
		models = append(models,
			map[string]any{"id": "claude-sonnet-4-20250514", "providerID": "anthropic", "name": "Claude Sonnet 4", "enabled": true},
			map[string]any{"id": "claude-opus-4-20250514", "providerID": "anthropic", "name": "Claude Opus 4", "enabled": true},
			map[string]any{"id": "claude-haiku-3-5-20241022", "providerID": "anthropic", "name": "Claude Haiku 3.5", "enabled": true},
		)
	}
	if os.Getenv("GOOGLE_API_KEY") != "" {
		models = append(models,
			map[string]any{"id": "gemini-2.0-flash", "providerID": "google", "name": "Gemini 2.0 Flash", "enabled": true},
			map[string]any{"id": "gemini-1.5-pro", "providerID": "google", "name": "Gemini 1.5 Pro", "enabled": true},
		)
	}
	return models
}
