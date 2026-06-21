package service

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/talon/backend/internal/llm"
)

// ── Tool Registry ───────────────────────────────────

type ToolFunc func(args json.RawMessage) (string, error)

type ToolRegistry struct {
	tools map[string]ToolDef
}

type ToolDef struct {
	Name        string
	Description string
	Parameters  any
	Execute     ToolFunc
}

func NewToolRegistry() *ToolRegistry {
	r := &ToolRegistry{tools: make(map[string]ToolDef)}
	r.registerBuiltins()
	return r
}

func (r *ToolRegistry) Register(t ToolDef) {
	r.tools[t.Name] = t
}

func (r *ToolRegistry) Definitions() []llm.ToolDefinition {
	var defs []llm.ToolDefinition
	for _, t := range r.tools {
		defs = append(defs, llm.ToolDefinition{
			Type: "function",
			Function: llm.ToolFunction{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.Parameters,
			},
		})
	}
	return defs
}

func (r *ToolRegistry) Execute(name string, argsJSON string) (string, error) {
	t, ok := r.tools[name]
	if !ok {
		return "", fmt.Errorf("unknown tool: %s", name)
	}
	return t.Execute(json.RawMessage(argsJSON))
}

// ── Built-in Tools ──────────────────────────────────

func (r *ToolRegistry) registerBuiltins() {
	r.Register(ToolDef{
		Name:        "read_file",
		Description: "Read the contents of a file at the given path",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "The absolute path to the file to read",
				},
			},
			"required": []string{"path"},
		},
		Execute: func(args json.RawMessage) (string, error) {
			var params struct{ Path string `json:"path"` }
			if err := json.Unmarshal(args, &params); err != nil {
				return "", fmt.Errorf("invalid args: %w", err)
			}
			data, err := os.ReadFile(params.Path)
			if err != nil {
				return "", fmt.Errorf("read file: %w", err)
			}
			return string(data), nil
		},
	})

	r.Register(ToolDef{
		Name:        "write_file",
		Description: "Write content to a file at the given path",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "The absolute path to the file to write",
				},
				"content": map[string]any{
					"type":        "string",
					"description": "The content to write to the file",
				},
			},
			"required": []string{"path", "content"},
		},
		Execute: func(args json.RawMessage) (string, error) {
			var params struct {
				Path    string `json:"path"`
				Content string `json:"content"`
			}
			if err := json.Unmarshal(args, &params); err != nil {
				return "", fmt.Errorf("invalid args: %w", err)
			}
			dir := filepath.Dir(params.Path)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return "", fmt.Errorf("create dir: %w", err)
			}
			if err := os.WriteFile(params.Path, []byte(params.Content), 0644); err != nil {
				return "", fmt.Errorf("write file: %w", err)
			}
			return fmt.Sprintf("Written %d bytes to %s", len(params.Content), params.Path), nil
		},
	})

	r.Register(ToolDef{
		Name:        "bash",
		Description: "Run a bash command and get the output",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"command": map[string]any{
					"type":        "string",
					"description": "The bash command to run",
				},
				"workdir": map[string]any{
					"type":        "string",
					"description": "The working directory to run the command in (optional)",
				},
			},
			"required": []string{"command"},
		},
		Execute: func(args json.RawMessage) (string, error) {
			var params struct {
				Command string `json:"command"`
				Workdir string `json:"workdir,omitempty"`
			}
			if err := json.Unmarshal(args, &params); err != nil {
				return "", fmt.Errorf("invalid args: %w", err)
			}

			cmd := exec.Command("bash", "-c", params.Command)
			if params.Workdir != "" {
				cmd.Dir = params.Workdir
			}

			output, err := cmd.CombinedOutput()
			if err != nil {
				return fmt.Sprintf("Exit error: %v\nOutput: %s", err, string(output)), nil
			}
			return string(output), nil
		},
	})

	r.Register(ToolDef{
		Name:        "search_code",
		Description: "Search for files and code patterns in a directory",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"pattern": map[string]any{
					"type":        "string",
					"description": "The search pattern (glob or filename)",
				},
				"path": map[string]any{
					"type":        "string",
					"description": "The directory to search in",
				},
			},
			"required": []string{"pattern", "path"},
		},
		Execute: func(args json.RawMessage) (string, error) {
			var params struct {
				Pattern string `json:"pattern"`
				Path    string `json:"path"`
			}
			if err := json.Unmarshal(args, &params); err != nil {
				return "", fmt.Errorf("invalid args: %w", err)
			}

			var results []string
			filepath.WalkDir(params.Path, func(path string, d os.DirEntry, err error) error {
				if err != nil || d.IsDir() {
					return nil
				}
				if strings.Contains(strings.ToLower(d.Name()), strings.ToLower(params.Pattern)) {
					results = append(results, path)
				}
				return nil
			})

			if len(results) > 50 {
				results = results[:50]
			}

			if len(results) == 0 {
				return "No matching files found", nil
			}
			return strings.Join(results, "\n"), nil
		},
	})

	r.Register(ToolDef{
		Name:        "list_dir",
		Description: "List files and directories in a path",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "The absolute path to list",
				},
			},
			"required": []string{"path"},
		},
		Execute: func(args json.RawMessage) (string, error) {
			var params struct{ Path string `json:"path"` }
			if err := json.Unmarshal(args, &params); err != nil {
				return "", fmt.Errorf("invalid args: %w", err)
			}

			entries, err := os.ReadDir(params.Path)
			if err != nil {
				return "", fmt.Errorf("read dir: %w", err)
			}

			var lines []string
			for _, e := range entries {
				info, _ := e.Info()
				size := ""
				if !e.IsDir() {
					size = fmt.Sprintf(" (%d bytes)", info.Size())
				}
				prefix := "📄"
				if e.IsDir() {
					prefix = "📁"
				}
				lines = append(lines, fmt.Sprintf("%s %s%s", prefix, e.Name(), size))
			}
			return strings.Join(lines, "\n"), nil
		},
	})
}
