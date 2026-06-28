package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(setupCmd)
}

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Run the first-time setup wizard",
	Long: `Guides you through initial configuration of talon.
Asks about your preferred runtime (Docker or native), API keys, and default model.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSetup()
	},
}

func runSetup() error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println()
	fmt.Println("╭──────────────────────────────────────────────╮")
	fmt.Println("│       talon Setup Wizard             │")
	fmt.Println("╰──────────────────────────────────────────────╯")
	fmt.Println()

	// Step 1: Runtime selection
	fmt.Println("Step 1: How would you like to run talon?")
	fmt.Println()
	dockerAvailable := isDockerAvailable()

	runOptions := []string{
		"Native — run directly on this machine (recommended for development)",
	}
	if dockerAvailable {
		runOptions = append(runOptions, "Docker — run in a container (recommended for deployment)")
	} else {
		fmt.Println("  (Docker not detected on this system)")
	}
	runOptions = append(runOptions, "Skip — I'll configure later")

	for i, opt := range runOptions {
		fmt.Printf("  %d) %s\n", i+1, opt)
	}
	fmt.Println()

	runChoice := promptChoice(reader, "Select option", 1, len(runOptions))

	fmt.Println()

	// Step 2: API Key setup
	fmt.Println("Step 2: Set up at least one API key")
	fmt.Println()
	fmt.Println("  You'll need an API key from a supported provider to use AI models.")
	fmt.Println("  Common options:")
	fmt.Println("    • Anthropic:    ANTHROPIC_API_KEY")
	fmt.Println("    • OpenAI:       OPENAI_API_KEY")
	fmt.Println("    • OpenRouter:   OPENROUTER_API_KEY")
	fmt.Println("    • Google Gemini: GEMINI_API_KEY")
	fmt.Println("    • Groq:         GROQ_API_KEY")
	fmt.Println()

	homeDir, _ := os.UserHomeDir()
	envFile := filepath.Join(homeDir, ".talon", ".env")
	envDir := filepath.Join(homeDir, ".talon")

	fmt.Printf("  API keys will be stored in: %s\n", envFile)
	fmt.Println()

	keys := map[string]string{
		"ANTHROPIC_API_KEY":    "Anthropic",
		"OPENAI_API_KEY":       "OpenAI",
		"OPENROUTER_API_KEY":   "OpenRouter",
		"GEMINI_API_KEY":       "Google Gemini",
		"DEEPSEEK_API_KEY":     "DeepSeek",
		"GROQ_API_KEY":         "Groq",
		"MISTRAL_API_KEY":      "Mistral",
		"CODESTRAL_API_KEY":    "Codestral",
		"FIREWORKS_API_KEY":    "Fireworks AI",
	}

	for envVar, name := range keys {
		fmt.Printf("  Enter your %s API key (or press Enter to skip): ", name)
		val, _ := reader.ReadString('\n')
		val = strings.TrimSpace(val)
		if val != "" {
			// Ensure env dir exists
			if err := os.MkdirAll(envDir, 0o755); err != nil {
				return fmt.Errorf("failed to create config directory: %w", err)
			}
			// Append to .env file
			f, err := os.OpenFile(envFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
			if err != nil {
				return fmt.Errorf("failed to open env file: %w", err)
			}
			fmt.Fprintf(f, "%s=%s\n", envVar, val)
			f.Close()
			fmt.Printf("    ✓ %s set\n", name)
		}
	}

	fmt.Println()

	// Step 3: Summary
	fmt.Println("Step 3: Summary")
	fmt.Println()

	switch runChoice {
	case 1:
		fmt.Println("  Runtime:   Native")
		fmt.Println("  To start:  go build -o talon . && ./talon")
	case 2:
		fmt.Println("  Runtime:   Docker")
		fmt.Println("  To start:  cd docker && docker compose up -d")
	case 3:
		fmt.Println("  Runtime:   Not configured (you can set up later)")
	}

	if envFileExists(envFile) {
		fmt.Println("  API keys:  Configured ✓")
	} else {
		fmt.Println("  API keys:  Not configured (set in ~/.talon/.env)")
	}

	fmt.Println()
	fmt.Println("╭──────────────────────────────────────────────╮")
	fmt.Println("│       Setup complete!                        │")
	fmt.Println("╰──────────────────────────────────────────────╯")
	fmt.Println()

	return nil
}

func isDockerAvailable() bool {
	if runtime.GOOS == "windows" {
		cmd := exec.Command("where", "docker")
		return cmd.Run() == nil
	}
	cmd := exec.Command("which", "docker")
	return cmd.Run() == nil
}

func envFileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func promptChoice(reader *bufio.Reader, prompt string, min, max int) int {
	for {
		fmt.Printf("  %s [%d-%d]: ", prompt, min, max)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		var choice int
		if _, err := fmt.Sscanf(input, "%d", &choice); err != nil {
			fmt.Println("  Please enter a number.")
			continue
		}
		if choice < min || choice > max {
			fmt.Printf("  Please enter a number between %d and %d.\n", min, max)
			continue
		}
		return choice
	}
}
