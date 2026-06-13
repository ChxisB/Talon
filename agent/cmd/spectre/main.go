// Spectre Proxy — Unified CLI
//
//	spectre                        Launch the AI coding agent
//	spectre "prompt"               Single prompt, print the reply
//	spectre --model <m> <prompt>   Use a specific model
//	spectre serve                  Start the proxy server only
//	spectre status                 Check proxy health
//	spectre models                 List available models
//	spectre configure              Open .env for editing
//
// Binary layout:
//
//	spectre           → this CLI (user-facing)
//	spectre-server    → the Go proxy server (daemon)
//	spectre-dashboard → the web UI (Docker)
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
)

const proxyURL = "http://localhost:8082"

var Version = "0.1.0"

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `Spectre Proxy v%s

Usage:
  spectre                        Launch AI agent (selects backend interactively)
  spectre --cli claude           Launch Claude Code
  spectre --cli codex            Launch OpenAI Codex
  spectre --cli gemini           Launch Google Gemini
  spectre <prompt>               Single prompt, print and exit
  spectre --model <m> <prompt>   Use a specific model
  spectre serve                  Start the proxy server only
  spectre status                 Check proxy health
  spectre models                 List available models from proxy
  spectre configure              Open ~/.spectre-proxy/.env for editing
  spectre mcp-server             Run MCP tool server (used internally)

Tools provided to agent:
  vault_search / vault_read     Obsidian vault access
  vault_graph                   Knowledge graph of your vault
  task_create / task_list       Task and goal management
  sub_agent_chat                Expert sub-agents (auto-detect)
  code_review                   Code review and analysis

Sub-agents (auto-selected by task):
  Flutter Developer · Frontend Developer (React/Next.js)
  Backend Developer (Go/Python/Rust/Zig)
  DevOps (GCP/AWS/Azure/Docker/K8s)
  Data Engineer (PostgreSQL/analytics)

Examples:
  spectre                                 # prompt for backend, launch agent
  spectre --cli codex                     # launch Codex directly
  spectre "write a python script"          # single prompt
  spectre --model openai/gpt-4o "hello"    # with model override
  spectre --cli gemini --model deepseek    # Gemini with model override
  spectre status                           # check proxy status
  spectre models                          # list models

Flags:
`, Version)
		flag.PrintDefaults()
	}
}

func main() {
	cliFlag := flag.String("cli", "", "CLI backend: claude, codex, or gemini")
	modelFlag := flag.String("model", "", "Model override (e.g. openrouter/anthropic/claude-sonnet-4)")
	dirFlag := flag.String("dir", "", "Working directory for the session")
	versionFlag := flag.Bool("version", false, "Print version")
	helpFlag := flag.Bool("help", false, "Show this help")
	flag.Parse()

	// Ensure default config files exist
	writeDefaultAgents()
	initTaskStore()

	// Check for incomplete tasks on startup
	resumeIncompleteTasks()

	if *versionFlag {
		fmt.Printf("spectre v%s\n", Version)
		return
	}
	if *helpFlag {
		flag.Usage()
		return
	}

	// Read config from .env
	if *dirFlag == "" {
		if dw := readEnvVar("DEFAULT_WORKSPACE"); dw != "" {
			*dirFlag = dw
		}
	}
	if *modelFlag == "" {
		if m := readEnvVar("MODEL"); m != "" {
			*modelFlag = m
		}
	}
	agentName := readEnvVar("AGENT_NAME")
	if agentName == "" {
		agentName = "Spectre"
	}

	args := flag.Args()

	if len(args) == 0 {
		backend := selectBackend(*cliFlag)
		launchAgent(backend, agentName, *modelFlag, *dirFlag)
		return
	}

	switch args[0] {
	case "serve":
		startServer()
	case "status":
		status()
	case "models":
		models()
	case "configure":
		configure()
	case "mcp-server", "mcp":
		runMCPServer()
	case "run":
		runPendingTasks()
	case "tasks":
		fmt.Println(taskListJSON())
	case "cron":
		if len(args) > 1 && args[1] == "run" {
			cronRun()
		} else if len(args) > 1 && args[1] == "add" && len(args) >= 5 {
			cronAdd(args[2], args[3], args[4], "")
		} else {
			cronList()
		}
	case "dream":
		dreamCycle()
	case "chat", "agent":
		backend := selectBackend(*cliFlag)
		launchAgent(backend, agentName, *modelFlag, *dirFlag)
	default:
		prompt(strings.Join(args, " "), *modelFlag)
	}
}

// ─── Subcommands ──────────────────────────────────────────────────────

func status() {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(proxyURL + "/health")
	if err != nil {
		fmt.Println("Proxy: Offline")
		fmt.Printf("  %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	var data map[string]string
	json.NewDecoder(resp.Body).Decode(&data)
	fmt.Printf("Proxy: Online (%s)\n", data["status"])
}

func models() {
	ensureProxy()
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(proxyURL + "/v1/models")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching models: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding models: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Available models (%d):\n", len(result.Data))
	for _, m := range result.Data {
		fmt.Printf("  %s\n", m.ID)
	}
}

func configure() {
	home, _ := os.UserHomeDir()
	envPath := filepath.Join(home, ".spectre-proxy", ".env")

	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		os.MkdirAll(filepath.Dir(envPath), 0755)
		os.WriteFile(envPath, []byte(`# Spectre Proxy Configuration
AGENT_NAME=Spectre
# CLI_BACKEND=claude      # Uncomment to skip the agent picker prompt
# CLI_BACKEND=codex
# CLI_BACKEND=gemini
OPENROUTER_API_KEY=
OPENAI_API_KEY=
MODEL=openrouter/anthropic/claude-sonnet-4
`), 0644)
	}

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "nano"
		if runtime.GOOS == "darwin" {
			editor = "open"
		}
	}

	cmd := exec.Command(editor, envPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open editor: %v\n", err)
		os.Exit(1)
	}
}

func startServer() {
	fmt.Println("Starting Spectre proxy server...")
	cmd := exec.Command(findServerBin())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Server exited: %v\n", err)
		os.Exit(1)
	}
}

// ─── Agent Launcher ───────────────────────────────────────────────────

// launchAgent ensures the proxy is running, syncs backend config,
// and launches the selected AI coding agent CLI as a child process.
func launchAgent(backend CLIBackend, agentName, model, workDir string) {
	ensureProxy()

	// Sync backend-specific config (MCP servers, model providers, etc.)
	backend.SyncConfig(proxyURL)

	// Show Spectre-branded welcome
	fmt.Printf("\n  \033[36m%s\033[0m — AI coding agent\n", agentName)
	fmt.Printf("  \033[90mBackend: %s\033[0m\n", backend.Name())
	fmt.Printf("  %s\n", backend.WelcomeMessage(agentName))
	if model != "" {
		fmt.Printf("  \033[90mModel: %s\033[0m\n", model)
	}
	if workDir != "" {
		fmt.Printf("  \033[90mWorking dir: %s\033[0m\n", workDir)
	}
	fmt.Println()

	// Find the CLI binary
	binPath, err := exec.LookPath(backend.Binary())
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s CLI not found. Install it:\n", backend.Name())
		fmt.Fprintf(os.Stderr, "%s\n", backend.InstallHint())
		os.Exit(1)
	}

	// ── Write default sub-agent files if missing ──
	writeDefaultAgents()

	// ── Build environment ──
	env := os.Environ()
	env = append(env, backend.EnvVars(proxyURL)...)

	// ── Build arguments ──
	remaining := flag.Args()
	if len(remaining) > 0 {
		remaining = remaining[1:]
	}
	cmdArgs := backend.Args(model, remaining)

	// ── Spawn the CLI ──
	cmd := exec.Command(binPath, cmdArgs...)
	if workDir != "" {
		cmd.Dir = workDir
	}
	cmd.Env = env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Handle Ctrl+C cleanup
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		if cmd.Process != nil {
			cmd.Process.Signal(syscall.SIGTERM)
		}
	}()

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintf(os.Stderr, "Failed to launch agent: %v\n", err)
		os.Exit(1)
	}
}

// ─── Prompt Mode ──────────────────────────────────────────────────────

func prompt(text string, model string) {
	ensureProxy()
	if model == "" {
		if m := readEnvVar("MODEL"); m != "" {
			model = m
		}
	}

	workDir, _ := os.Getwd()
	// Prepend an implicit instruction to explore if the prompt sounds like a question about the project
	fullText := text
	if !strings.Contains(strings.ToLower(text), "use") && !strings.Contains(strings.ToLower(text), "explore") && !strings.Contains(strings.ToLower(text), "list") {
		fullText = fmt.Sprintf("I'm working in the directory: %s\n\n%s", workDir, text)
	}

	history := []map[string]string{{"role": "user", "text": fullText}}
	reply, err := sendMessages(history, model)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(reply)
}

// ─── Shared API ───────────────────────────────────────────────────────

// ─── Tool Definitions ────────────────────────────────────────────────

var agentTools = []map[string]any{
	{
		"name":        "Bash",
		"description": "Execute shell commands on the local machine. Use this to explore directories, read files, run commands, and interact with the system.",
		"input_schema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"command":     map[string]any{"type": "string", "description": "The shell command to execute"},
				"description": map[string]any{"type": "string", "description": "A brief description of what this command does"},
			},
			"required": []string{"command"},
		},
	},
	{
		"name":        "Read",
		"description": "Read the contents of a file. Use this to view source code, configuration files, and other text files.",
		"input_schema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"filePath": map[string]any{"type": "string", "description": "The absolute path to the file to read"},
			},
			"required": []string{"filePath"},
		},
	},
	{
		"name":        "Glob",
		"description": "Search for files matching a glob pattern. Use this to find files by name or extension.",
		"input_schema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"pattern": map[string]any{"type": "string", "description": "The glob pattern to search for (e.g. **/*.go, **/package.json)"},
				"path":    map[string]any{"type": "string", "description": "Optional directory to search in"},
			},
			"required": []string{"pattern"},
		},
	},
	{
		"name":        "Grep",
		"description": "Search file contents using regex patterns. Use this to find specific code, configuration, or text across files.",
		"input_schema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"pattern": map[string]any{"type": "string", "description": "The regex pattern to search for"},
				"include": map[string]any{"type": "string", "description": "Optional file pattern to filter by (e.g. *.go, *.json)"},
			},
			"required": []string{"pattern"},
		},
	},
}

const agentSystemPrompt = `You are an AI coding assistant with access to local development tools.
You have Bash, Read, Glob, and Grep tools available.
Use Bash to explore directories and run commands.
Use Read to view file contents.
Use Glob to find files by name pattern.
Use Grep to search inside files.
Think step by step about what tools to use, but always use the appropriate tools rather than just guessing.
When you run Bash commands, always include the 'command' parameter with the actual shell command to execute.
Stop using tools once you have enough information to answer the user's question.`

// agentLoop sends messages to the model, executes any tool calls, and loops
// until the model returns a final text response. Returns the accumulated text.
func agentLoop(messages []map[string]any, model string) (string, error) {
	client := &http.Client{Timeout: 120 * time.Second}
	var fullText strings.Builder
	maxTurns := 10

	for turn := 0; turn < maxTurns; turn++ {
		body := map[string]any{
			"messages":   messages,
			"max_tokens": 8192,
			"stream":     true,
			"tools":      agentTools,
			"system":     agentSystemPrompt,
		}
		if model != "" {
			body["model"] = model
		}

		payload, _ := json.Marshal(body)

		resp, err := client.Post(proxyURL+"/v1/messages", "application/json", bytes.NewReader(payload))
		if err != nil {
			return "", fmt.Errorf("connection failed: %w", err)
		}

		if resp.StatusCode != 200 {
			errBody, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return "", fmt.Errorf("proxy error (%d): %s", resp.StatusCode, string(errBody))
		}

		// Parse the SSE stream, collecting text and tool calls
		var textBuf strings.Builder
		type pendingTool struct {
			ID        string
			Name      string
			Arguments map[string]any
		}
		var toolCalls []pendingTool
		scanner := bufio.NewScanner(resp.Body)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}

			var raw map[string]any
			if err := json.Unmarshal([]byte(line[6:]), &raw); err != nil {
				continue
			}

			eventType, _ := raw["type"].(string)
			switch eventType {
			case "content_block_delta":
				if delta, ok := raw["delta"].(map[string]any); ok {
					if dt, _ := delta["type"].(string); dt == "text_delta" {
						if text, _ := delta["text"].(string); text != "" {
							textBuf.WriteString(text)
							fmt.Print(text)
						}
					}
				}
			case "content_block_start":
				if block, ok := raw["content_block"].(map[string]any); ok {
					if bt, _ := block["type"].(string); bt == "tool_use" {
						tc := pendingTool{
							ID:   getString(block, "id"),
							Name: getString(block, "name"),
						}
						if input, ok := block["input"].(map[string]any); ok {
							tc.Arguments = input
						}
						if tc.ID != "" && tc.Name != "" {
							toolCalls = append(toolCalls, tc)
						}
					}
				}
			case "message_stop":
				// End of this turn's response
			}
		}
		resp.Body.Close()

		if len(toolCalls) == 0 {
			// No more tool calls — we're done
			return textBuf.String(), nil
		}

		// Filter out tool calls with empty arguments — DeepSeek sometimes
		// generates the tool call structure without filling in parameters.
		var validCalls []pendingTool
		for _, tc := range toolCalls {
			if len(tc.Arguments) == 0 {
				fmt.Print("\n[Tool call skipped: no arguments provided. Please include required parameters.]\n")
				continue
			}
			validCalls = append(validCalls, tc)
		}
		if len(validCalls) == 0 {
			// No valid tool calls — send feedback and continue the loop
			messages = append(messages, map[string]any{
				"role": "user",
				"content": []map[string]string{
					{"type": "text", "text": "Your tool calls are missing the required arguments. Please include the command parameter when using Bash."},
				},
			})
			continue
		}

		// Execute tool calls and add results to the conversation history
		for _, tc := range validCalls {
			result := executeTool(tc.Name, tc.Arguments)
			messages = append(messages, map[string]any{
				"role": "assistant",
				"content": []map[string]any{
					{"type": "tool_use", "id": tc.ID, "name": tc.Name, "input": tc.Arguments},
				},
			})
			messages = append(messages, map[string]any{
				"role": "user",
				"content": []map[string]any{
					{"type": "tool_result", "tool_use_id": tc.ID, "content": result},
				},
			})
		}
	}

	return fullText.String(), fmt.Errorf("agent loop: reached max turns (%d)", maxTurns)
}

// executeTool runs a tool locally and returns its output as a string.
func executeTool(name string, args map[string]any) string {
	switch name {
	case "Bash":
		cmdStr, _ := args["command"].(string)
		if cmdStr == "" {
			return "Error: command parameter is required"
		}
		cmd := exec.Command("sh", "-c", cmdStr)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Sprintf("Exit code: %d\nOutput:\n%s\nError: %v", cmd.ProcessState.ExitCode(), string(out), err)
		}
		return string(out)

	case "Read":
		filePath, _ := args["filePath"].(string)
		if filePath == "" {
			return "Error: filePath parameter is required"
		}
		data, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Sprintf("Error reading file: %v", err)
		}
		return string(data)

	case "Glob":
		pattern, _ := args["pattern"].(string)
		if pattern == "" {
			return "Error: pattern parameter is required"
		}
		searchPath, _ := args["path"].(string)
		if searchPath == "" {
			searchPath = "."
		}
		matches, err := filepath.Glob(filepath.Join(searchPath, pattern))
		if err != nil {
			// Try without joining to path
			matches, err = filepath.Glob(pattern)
			if err != nil {
				return fmt.Sprintf("Error globbing: %v", err)
			}
		}
		if len(matches) == 0 {
			return "No files found matching: " + pattern
		}
		return strings.Join(matches, "\n")

	case "Grep":
		pattern, _ := args["pattern"].(string)
		if pattern == "" {
			return "Error: pattern parameter is required"
		}
		// Simple grep-like search in current directory
		// os.ReadDir only reads one level, use Bash for recursive
		return "Use Bash with grep command for recursive searches"

	default:
		return fmt.Sprintf("Unknown tool: %s", name)
	}
}

func getString(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func sendMessages(history []map[string]string, model string) (string, error) {
	var messages []map[string]any
	for _, msg := range history {
		messages = append(messages, map[string]any{
			"role": msg["role"],
			"content": []map[string]string{
				{"type": "text", "text": msg["text"]},
			},
		})
	}

	return agentLoop(messages, model)
}

// ─── Helpers ──────────────────────────────────────────────────────────

func ensureProxy() {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(proxyURL + "/health")
	if err == nil {
		resp.Body.Close()
		return
	}

	fmt.Fprintf(os.Stderr, "Starting proxy server on %s...\n", proxyURL)
	cmd := exec.Command(findServerBin())
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start proxy: %v\n", err)
		os.Exit(1)
	}

	for i := 0; i < 15; i++ {
		time.Sleep(500 * time.Millisecond)
		resp, err := client.Get(proxyURL + "/health")
		if err == nil {
			resp.Body.Close()
			fmt.Fprintf(os.Stderr, "Proxy started.\n")
			return
		}
	}
	fmt.Fprintf(os.Stderr, "Proxy failed to start within timeout.\n")
	os.Exit(1)
}

func readEnvVar(key string) string {
	home, _ := os.UserHomeDir()
	data, err := os.ReadFile(filepath.Join(home, ".spectre-proxy", ".env"))
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, key+"=") {
			return strings.TrimPrefix(trimmed, key+"=")
		}
	}
	return ""
}

func findServerBin() string {
	exe, _ := os.Executable()
	dir := filepath.Dir(exe)
	candidate := filepath.Join(dir, "spectre-server")
	if _, err := os.Stat(candidate); err == nil {
		return candidate
	}
	for _, p := range []string{
		filepath.Join(os.Getenv("HOME"), ".spectre-proxy", "bin", "spectre-server"),
		"/usr/local/bin/spectre-server",
	} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	if bin, err := exec.LookPath("spectre-server"); err == nil {
		return bin
	}
	return "spectre-server"
}
