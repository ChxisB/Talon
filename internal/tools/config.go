// Package tools manages the configuration and orchestration of Talon.s
// built-in tools (synth, compress, reducer, viz, filter, usage, graph).
// Tools can be enabled/disabled via the dashboard and are auto-run
// when appropriate.
package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// ToolID identifies a built-in tool.
type ToolID string

const (
	// ─── Standalone tools (not part of reducer) ─────────────────────
	ToolViz           ToolID = "viz"           // Diagram generation
	ToolUsage         ToolID = "usage"         // Token usage analytics
	ToolGraph         ToolID = "graph"         // Knowledge graph
	ToolCVE           ToolID = "cve"              // CVE vulnerability checker

	// ─── Token-reduction tools (all controlled by ToolReducer) ────
	ToolSynth         ToolID = "synth"         // Coding principles injection
	ToolFilter        ToolID = "filter"        // Command output filtering
	ToolCompress      ToolID = "compress"      // Output compression
	ToolCondense      ToolID = "condense"      // Input context compression (tool outputs, JSON)
	ToolMemoryTree    ToolID = "memory-tree"   // glade: hierarchical memory tree (input savings)
	ToolTokenOptimizer ToolID = "token-optimizer" // frugal: token/context optimization
	ToolResponseCache ToolID = "response-cache"   // SQLite response cache
	ToolPromptCache   ToolID = "promptcache"   // Prompt-response semantic cache

	// ─── Master switch for all token reduction ────────────────────
	ToolReducer       ToolID = "reducer"
)

// AllTools returns all available tool IDs.
func AllTools() []ToolID {
	return []ToolID{ToolSynth, ToolViz, ToolFilter, ToolUsage, ToolGraph, ToolCVE, ToolCompress, ToolCondense, ToolMemoryTree, ToolTokenOptimizer, ToolResponseCache, ToolPromptCache, ToolReducer}
}

// ReducerSubTools returns the tool IDs that the reducer controls.
func ReducerSubTools() []ToolID {
	return []ToolID{ToolSynth, ToolFilter, ToolCompress, ToolCondense, ToolMemoryTree, ToolTokenOptimizer, ToolResponseCache, ToolPromptCache}
}

// ToolConfig holds the configuration for a single tool.
type ToolConfig struct {
	Enabled bool   `json:"enabled"`
	Level   string `json:"level,omitempty"` // For compress: "lite"/"full"/"ultra"
}

// Config holds the configuration for all tools.
type Config struct {
	Tools map[ToolID]*ToolConfig `json:"tools"`
	mu    sync.RWMutex
}

// DefaultConfig returns a config with all tools enabled at default settings.
// The reducer is enabled with "recommended" level, which applies sensible
// defaults to all token-reduction sub-tools.
func DefaultConfig() *Config {
	c := &Config{
		Tools: make(map[ToolID]*ToolConfig),
	}
	for _, id := range AllTools() {
		c.Tools[id] = &ToolConfig{
			Enabled: true,
		}
	}
	// Reducer defaults to "recommended" — the one-knob setting
	c.Tools[ToolReducer].Level = "recommended"
	// Output compression defaults to "full" mode
	c.Tools[ToolCompress].Level = "full"
	// Condense defaults to "moderate"
	c.Tools[ToolCondense].Level = "moderate"
	return c
}

// configPath returns the path to the tools config file.
func configPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".talon/tools.json"
	}
	return filepath.Join(home, ".talon", "tools.json")
}

// Load reads the config from disk, or returns defaults if not found.
func Load() *Config {
	path := configPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return DefaultConfig()
	}

	c := &Config{}
	if err := json.Unmarshal(data, c); err != nil {
		return DefaultConfig()
	}

	// Fill in missing tools with defaults
	def := DefaultConfig()
	if c.Tools == nil {
		c.Tools = make(map[ToolID]*ToolConfig)
	}
	for _, id := range AllTools() {
		if _, ok := c.Tools[id]; !ok {
			c.Tools[id] = def.Tools[id]
		}
	}

	return c
}

// Save writes the config to disk.
func (c *Config) Save() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	path := configPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o644)
}

// IsEnabled returns whether a tool is enabled.
// For reducer sub-tools, this checks the master reducer toggle only —
// individual sub-tool settings are overridden by the reducer's level.
func (c *Config) IsEnabled(id ToolID) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// If the master reducer is disabled, all sub-tools are disabled
	reducer, hasReducer := c.Tools[ToolReducer]
	if hasReducer && !reducer.Enabled && id != ToolReducer && c.isSubTool(id) {
		return false
	}

	// If the master reducer is enabled, sub-tools are always on
	// (their behavior is controlled by the level, not individual toggles)
	if hasReducer && reducer.Enabled && id != ToolReducer && c.isSubTool(id) {
		return true
	}

	if tc, ok := c.Tools[id]; ok {
		return tc.Enabled
	}
	return false
}

// isSubTool checks if a tool ID is controlled by the reducer.
func (c *Config) isSubTool(id ToolID) bool {
	for _, sub := range ReducerSubTools() {
		if id == sub {
			return true
		}
	}
	return false
}

// SetEnabled sets a tool's enabled state.
func (c *Config) SetEnabled(id ToolID, enabled bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.Tools == nil {
		c.Tools = make(map[ToolID]*ToolConfig)
	}
	if tc, ok := c.Tools[id]; ok {
		tc.Enabled = enabled
	} else {
		c.Tools[id] = &ToolConfig{Enabled: enabled}
	}
}

// GetLevel returns a tool's level setting.
func (c *Config) GetLevel(id ToolID) string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if tc, ok := c.Tools[id]; ok {
		return tc.Level
	}
	return ""
}

// SetLevel sets a tool's level setting.
func (c *Config) SetLevel(id ToolID, level string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.Tools == nil {
		c.Tools = make(map[ToolID]*ToolConfig)
	}
	if tc, ok := c.Tools[id]; ok {
		tc.Level = level
	} else {
		c.Tools[id] = &ToolConfig{Level: level}
	}
}

// Status returns the status of all tools.
func (c *Config) Status() map[ToolID]ToolStatus {
	c.mu.RLock()
	defer c.mu.RUnlock()

	status := make(map[ToolID]ToolStatus)
	for _, id := range AllTools() {
		tc := c.Tools[id]
		status[id] = ToolStatus{
			ID:      id,
			Name:    toolName(id),
			Enabled: tc != nil && tc.Enabled,
			Level:   tc.Level,
		}
	}
	return status
}

// ToolStatus holds the status of a tool for display.
type ToolStatus struct {
	ID      ToolID `json:"id"`
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
	Level   string `json:"level,omitempty"`
}

func toolName(id ToolID) string {
	switch id {
	case ToolSynth:
		return "Synth (Coding Principles)"
	case ToolFilter:
		return "Filter (Command Output)"
	case ToolCompress:
		return "Compress (Output)"
	case ToolCondense:
		return "Condense (Input)"
	case ToolMemoryTree:
		return "Memory Tree"
	case ToolTokenOptimizer:
		return "Token Optimizer"
	case ToolResponseCache:
		return "Response Cache"
	case ToolPromptCache:
		return "Prompt Cache"
	case ToolViz:
		return "Viz (Diagram Generator)"
	case ToolUsage:
		return "Usage (Token Analytics)"
	case ToolGraph:
		return "Graph (Knowledge Graph)"
	case ToolCVE:
		return "CVE Vulnerability Checker"
	case ToolReducer:
		return "Token Reducer"
	default:
		return string(id)
	}
}

// MarshalJSON implements json.Marshaler for ToolID map keys.
func (c *Config) MarshalJSON() ([]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	m := make(map[string]*ToolConfig)
	for k, v := range c.Tools {
		m[string(k)] = v
	}
	return json.Marshal(m)
}

// UnmarshalJSON implements json.Unmarshaler for ToolID map keys.
func (c *Config) UnmarshalJSON(data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var m map[string]*ToolConfig
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}

	c.Tools = make(map[ToolID]*ToolConfig)
	for k, v := range m {
		c.Tools[ToolID(k)] = v
	}
	return nil
}

// String returns a human-readable summary.
func (c *Config) String() string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	enabled := 0
	for _, tc := range c.Tools {
		if tc.Enabled {
			enabled++
		}
	}
	return fmt.Sprintf("Tools: %d/%d enabled", enabled, len(AllTools()))
}
