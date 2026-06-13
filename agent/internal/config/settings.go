// Package config implements the Spectre Proxy proxy configuration system.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Settings holds all application configuration, loaded from environment variables.
type Settings struct {
	// Server
	Host string
	Port int

	// Auth
	APIKey string

	// Model routing
	Model       string // Default model
	ModelOpus   string
	ModelSonnet string
	ModelHaiku  string

	// Provider config
	ProviderType string

	// Thinking
	EnableOpusThinking   bool
	EnableSonnetThinking bool
	EnableHaikuThinking  bool

	// Web tools
	EnableWebServerTools         bool
	WebFetchAllowPrivateNetworks bool
	WebFetchAllowedSchemes       []string

	// Optimizations
	FastPrefixDetection       bool
	EnableNetworkProbeMock    bool
	EnableTitleGenerationSkip bool
	EnableSuggestionModeSkip  bool

	// Logging
	LogRawAPIPayloads     bool
	LogAPIErrorTracebacks bool

	// Messaging
	DiscordBotToken  string
	TelegramBotToken string

	// Provider-specific
	NIM NimSettings

	// File paths
	ConfigDir string
}

// NimSettings holds NVIDIA NIM-specific configuration.
type NimSettings struct {
	Temperature float64
	TopP        float64
	TopK        int
}

// DefaultSettings returns a Settings with sensible defaults.
func DefaultSettings() *Settings {
	return &Settings{
		Host:                      "127.0.0.1",
		Port:                      8082,
		Model:                     "claude-sonnet-4-20250514",
		EnableSonnetThinking:      true,
		EnableOpusThinking:        true,
		FastPrefixDetection:       true,
		EnableNetworkProbeMock:    true,
		EnableTitleGenerationSkip: true,
		EnableSuggestionModeSkip:  true,
		WebFetchAllowedSchemes:    []string{"https", "http"},
		ConfigDir:                 defaultConfigDir(),
		NIM: NimSettings{
			Temperature: 0.6,
			TopP:        0.9,
			TopK:        0,
		},
	}
}

// LoadFromEnv loads settings from environment variables, overriding defaults.
func (s *Settings) LoadFromEnv() {
	if v := os.Getenv("HOST"); v != "" {
		s.Host = v
	}
	if v := os.Getenv("PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			s.Port = p
		}
	}
	if v := os.Getenv("API_KEY"); v != "" {
		s.APIKey = v
	}
	if v := os.Getenv("ANTHROPIC_API_KEY"); v != "" {
		s.APIKey = v
	}
	if v := os.Getenv("MODEL"); v != "" {
		s.Model = v
	}
	if v := os.Getenv("MODEL_OPUS"); v != "" {
		s.ModelOpus = v
	}
	if v := os.Getenv("MODEL_SONNET"); v != "" {
		s.ModelSonnet = v
	}
	if v := os.Getenv("MODEL_HAIKU"); v != "" {
		s.ModelHaiku = v
	}
	if v := os.Getenv("PROVIDER_TYPE"); v != "" {
		s.ProviderType = v
	}
	if v := os.Getenv("ENABLE_WEB_SERVER_TOOLS"); v != "" {
		s.EnableWebServerTools = v == "true" || v == "1"
	}
	if v := os.Getenv("FAST_PREFIX_DETECTION"); v != "" {
		s.FastPrefixDetection = v == "true" || v == "1"
	}
	if v := os.Getenv("ENABLE_NETWORK_PROBE_MOCK"); v != "" {
		s.EnableNetworkProbeMock = v == "true" || v == "1"
	}
	if v := os.Getenv("ENABLE_TITLE_GENERATION_SKIP"); v != "" {
		s.EnableTitleGenerationSkip = v == "true" || v == "1"
	}
	if v := os.Getenv("DISCORD_BOT_TOKEN"); v != "" {
		s.DiscordBotToken = v
	}
	if v := os.Getenv("TELEGRAM_BOT_TOKEN"); v != "" {
		s.TelegramBotToken = v
	}
}

// ResolveModel resolves a Claude model name to a provider/model reference.
// Returns the model reference from env if set, otherwise the default model.
func (s *Settings) ResolveModel(claudeModel string) string {
	switch claudeModel {
	case "claude-opus-4-20250514", "claude-3-opus-20240229":
		if s.ModelOpus != "" {
			return s.ModelOpus
		}
	case "claude-sonnet-4-20250514", "claude-3-5-sonnet-20241022":
		if s.ModelSonnet != "" {
			return s.ModelSonnet
		}
	case "claude-haiku-4-20250514", "claude-3-5-haiku-20241022", "claude-3-haiku-20240307":
		if s.ModelHaiku != "" {
			return s.ModelHaiku
		}
	}
	return s.Model
}

// ResolveThinking returns whether thinking is enabled for the given model.
func (s *Settings) ResolveThinking(modelRef string) bool {
	// Provider-prefixed models: "openrouter/anthropic/claude-3.5-sonnet"
	// Check configured model tiers
	if strings.Contains(modelRef, "opus") {
		return s.EnableOpusThinking
	}
	if strings.Contains(modelRef, "sonnet") {
		return s.EnableSonnetThinking
	}
	if strings.Contains(modelRef, "haiku") {
		return s.EnableHaikuThinking
	}
	return true
}

// ParseProviderType extracts the provider ID from a "provider/model" reference.
func ParseProviderType(modelRef string) string {
	parts := strings.SplitN(modelRef, "/", 2)
	if len(parts) < 2 {
		return ""
	}
	return parts[0]
}

// ParseModelName extracts the model name from a "provider/model" reference.
func ParseModelName(modelRef string) string {
	parts := strings.SplitN(modelRef, "/", 2)
	if len(parts) < 2 {
		return modelRef
	}
	return parts[1]
}

func defaultConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".spectre-proxy"
	}
	return fmt.Sprintf("%s/.spectre-proxy", home)
}
