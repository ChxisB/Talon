// Package main — CLI backend abstraction for Spectre Proxy
//
// Defines the CLIBackend interface and implementations for each supported
// AI coding agent CLI: Claude Code (Anthropic), Codex (OpenAI), Gemini (Google).
//
// Each backend knows how to:
//   - Identify its binary and install instructions
//   - Set environment variables to route through the proxy
//   - Write any config files needed for proxy integration
//   - Build the correct command-line arguments
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CLIBackend defines the interface for an AI coding agent CLI.
// Each implementation handles the specific binary, env vars, config files,
// and launch arguments needed for that agent.
type CLIBackend interface {
	// Name returns a human-readable display name (e.g. "Claude Code").
	Name() string

	// Binary returns the CLI binary name (e.g. "claude", "codex", "gemini").
	Binary() string

	// InstallHint returns installation instructions if the binary isn't found.
	InstallHint() string

	// EnvVars returns environment variables to set when spawning the CLI.
	// These route the CLI's API calls through the Spectre Proxy.
	EnvVars(proxyURL string) []string

	// Args returns the command-line arguments for the CLI.
	// extraArgs are any remaining CLI arguments passed after flags.
	Args(model string, extraArgs []string) []string

	// SyncConfig writes any config files needed for this backend to work
	// with Spectre Proxy (e.g. MCP server config, custom model providers).
	SyncConfig(proxyURL string) error

	// WelcomeMessage returns the welcome line shown to the user on launch.
	WelcomeMessage(agentName string) string
}

// ─── Backend Implementations ─────────────────────────────────────────

// ── Claude Backend ──────────────────────────────────────────────────

type claudeBackend struct{}

func (claudeBackend) Name() string { return "Claude Code" }

func (claudeBackend) Binary() string { return "claude" }

func (claudeBackend) InstallHint() string {
	return "  npm install -g @anthropic-ai/claude-code"
}

func (claudeBackend) EnvVars(proxyURL string) []string {
	token := readEnvVar("ANTHROPIC_AUTH_TOKEN")
	if token == "" {
		token = "spectre-proxy"
	}
	home, _ := os.UserHomeDir()
	return []string{
		"ANTHROPIC_BASE_URL=" + proxyURL,
		"ANTHROPIC_AUTH_TOKEN=" + token,
		"CLAUDE_CODE_ENABLE_GATEWAY_MODEL_DISCOVERY=1",
		"VAULT_PATH=" + home + "/Spectre Proxy/agent-vault",
	}
}

func (claudeBackend) Args(model string, extraArgs []string) []string {
	args := []string{"--allowedTools", "Bash,Read,Edit,Write"}
	if model != "" {
		args = append(args, "--model", model)
	}
	args = append(args, extraArgs...)
	return args
}

func (b claudeBackend) SyncConfig(proxyURL string) error {
	syncMCPToClaude()
	return nil
}

func (claudeBackend) WelcomeMessage(agentName string) string {
	return fmt.Sprintf("  \033[90mPowered by Claude via Spectre Proxy\033[0m")
}

// ── Codex Backend ───────────────────────────────────────────────────

type codexBackend struct{}

func (codexBackend) Name() string { return "Codex" }

func (codexBackend) Binary() string { return "codex" }

func (codexBackend) InstallHint() string {
	return "  npm install -g @openai/codex\n  or: curl -fsSL https://chatgpt.com/codex/install.sh | sh"
}

func (codexBackend) EnvVars(proxyURL string) []string {
	return []string{
		"OPENAI_API_KEY=spectre-proxy",
	}
}

func (codexBackend) Args(model string, extraArgs []string) []string {
	var args []string
	if model != "" {
		args = append(args, "--model", model)
	}
	args = append(args, extraArgs...)
	return args
}

// syncCodexConfig writes ~/.codex/config.toml with a custom model provider
// that routes API calls through the Spectre Proxy.
//
// Codex does not support a simple OPENAI_BASE_URL env var, so we define
// a custom provider in its TOML config that points at the proxy.
func syncCodexConfig(proxyURL string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	codexDir := filepath.Join(home, ".codex")
	if err := os.MkdirAll(codexDir, 0755); err != nil {
		return err
	}

	config := fmt.Sprintf(`# Spectre Proxy — Codex config (auto-generated)
# This configures Codex to route API calls through the Spectre Proxy.
# When Phase B (proxy protocol expansion) is complete, the proxy will
# serve the OpenAI Responses API at /v1/responses.
# Until then, Codex connects directly — set CODEX_DIRECT=1 to bypass.

[model_providers.spectre]
name = "Spectre Proxy"
base_url = "%[1]s/v1"
env_key = "OPENAI_API_KEY"
wire_api = "responses"
requires_openai_auth = false
`, proxyURL)

	return os.WriteFile(filepath.Join(codexDir, "config.toml"), []byte(config), 0644)
}

func (b codexBackend) SyncConfig(proxyURL string) error {
	return syncCodexConfig(proxyURL)
}

func (codexBackend) WelcomeMessage(agentName string) string {
	return fmt.Sprintf("  \033[90mPowered by Codex via Spectre Proxy\033[0m")
}

// ── Gemini Backend ──────────────────────────────────────────────────

type geminiBackend struct{}

func (geminiBackend) Name() string { return "Gemini" }

func (geminiBackend) Binary() string { return "gemini" }

func (geminiBackend) InstallHint() string {
	return "  npm install -g @google/gemini-cli\n  or: brew install gemini-cli"
}

func (geminiBackend) EnvVars(proxyURL string) []string {
	return []string{
		"GOOGLE_GEMINI_BASE_URL=" + proxyURL,
		"GEMINI_API_KEY=spectre-proxy",
	}
}

func (geminiBackend) Args(model string, extraArgs []string) []string {
	var args []string
	if model != "" {
		args = append(args, "--model", model)
	}
	args = append(args, extraArgs...)
	return args
}

func (geminiBackend) SyncConfig(proxyURL string) error {
	// Gemini uses the GOOGLE_GEMINI_BASE_URL env var for routing.
	// No config file changes needed — the env var is sufficient.
	return nil
}

func (geminiBackend) WelcomeMessage(agentName string) string {
	return fmt.Sprintf("  \033[90mPowered by Gemini via Spectre Proxy\033[0m")
}

// ─── Backend Registry & Selection ───────────────────────────────────

// availableBackends maps backend IDs to their implementations.
var availableBackends = map[string]CLIBackend{
	"claude": claudeBackend{},
	"codex":  codexBackend{},
	"gemini": geminiBackend{},
}

// getBackend returns the backend for the given ID and whether it was found.
func getBackend(id string) (CLIBackend, bool) {
	b, ok := availableBackends[id]
	return b, ok
}

// selectBackend determines which CLI backend to use. Resolution order:
//  1. --cli flag value (handled by caller, passed via preferredID)
//  2. CLI_BACKEND env var in ~/.spectre-proxy/.env (persisted from a previous run)
//  3. Interactive prompt — choice is saved to .env for future runs
//  4. Default to claude
func selectBackend(preferredID string) CLIBackend {
	// 1. Explicit preference from --cli flag
	if preferredID != "" {
		if b, ok := getBackend(preferredID); ok {
			return b
		}
		fmt.Fprintf(os.Stderr, "  Unknown backend %q, falling back to selection.\n", preferredID)
	}

	// 2. CLI_BACKEND env var (persisted from a previous interactive selection)
	if id := readEnvVar("CLI_BACKEND"); id != "" {
		if b, ok := getBackend(id); ok {
			return b
		}
	}

	// 3. Interactive prompt
	fmt.Println()
	fmt.Println("  \033[36mSelect AI coding agent:\033[0m")
	fmt.Println("    \033[1m1)\033[0m  Claude Code \033[90m(Anthropic)\033[0m")
	fmt.Println("    \033[1m2)\033[0m  Codex \033[90m(OpenAI)\033[0m")
	fmt.Println("    \033[1m3)\033[0m  Gemini \033[90m(Google)\033[0m")
	fmt.Print("  \033[36mChoice [1-3]:\033[0m ")

	var input string
	fmt.Scanln(&input)

	var chosen CLIBackend
	switch input {
	case "1", "claude", "Claude":
		chosen = claudeBackend{}
	case "2", "codex", "Codex":
		chosen = codexBackend{}
	case "3", "gemini", "Gemini":
		chosen = geminiBackend{}
	default:
		fmt.Println("  \033[33mInvalid choice, defaulting to Claude Code.\033[0m")
		chosen = claudeBackend{}
	}

	// Save the choice to .env so the prompt is skipped next time
	saveCLIBackend(chosen.Binary())

	return chosen
}

// saveCLIBackend writes the CLI_BACKEND setting to ~/.spectre-proxy/.env
// so the user's selection persists across runs.
func saveCLIBackend(id string) {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	envPath := filepath.Join(home, ".spectre-proxy", ".env")
	data, err := os.ReadFile(envPath)
	if err != nil {
		return
	}

	lines := strings.Split(string(data), "\n")
	found := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "CLI_BACKEND=") {
			lines[i] = "CLI_BACKEND=" + id
			found = true
			break
		}
	}
	if !found {
		lines = append(lines, "CLI_BACKEND="+id)
	}
	os.WriteFile(envPath, []byte(strings.Join(lines, "\n")), 0644)
}
