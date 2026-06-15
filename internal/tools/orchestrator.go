package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/ChxisB/talon/deps/compress"
	"github.com/ChxisB/talon/deps/condense"
	"github.com/ChxisB/talon/deps/filter"
	"github.com/ChxisB/talon/deps/graph"
	"github.com/ChxisB/talon/deps/promptcache"
	"github.com/ChxisB/talon/deps/synth"
	"github.com/ChxisB/talon/deps/usage"
	"github.com/ChxisB/talon/deps/viz"
	"github.com/ChxisB/talon/internal/message"
)

// Orchestrator coordinates tool execution based on config.
type Orchestrator struct {
	config *Config
	cache  *promptcache.Cache

	// inputSavings tracks the estimated input token savings %
	// from memory-tree (glade) and token-optimizer (frugal).
	inputSavings      float64
	memoryTreeEnabled bool
}

// NewOrchestrator creates an orchestrator with the given config.
func NewOrchestrator(config *Config) *Orchestrator {
	return &Orchestrator{config: config}
}

// initCache lazily creates the prompt cache if enabled.
func (o *Orchestrator) initCache() *promptcache.Cache {
	if o.cache == nil && o.config.IsEnabled(ToolPromptCache) {
		cfg := promptcache.DefaultConfig()
		if l := o.config.GetLevel(ToolPromptCache); l != "" {
			switch l {
			case "low":
				cfg.HighThreshold = 0.85
			case "medium":
				cfg.HighThreshold = 0.92
			case "high":
				cfg.HighThreshold = 0.97
			}
		}
		o.cache = promptcache.New(cfg, nil)
	}
	return o.cache
}

// DefaultOrchestrator creates an orchestrator with default config.
func DefaultOrchestrator() *Orchestrator {
	return NewOrchestrator(DefaultConfig())
}

// ProcessPrompt applies enabled tools to an outgoing prompt.
// This is called before sending a prompt to the LLM.
// When the reducer is enabled, the prompt text is compressed to reduce
// token consumption — this is the main lever for reducing the 10k+
// initial system prompt overhead.
func (o *Orchestrator) ProcessPrompt(prompt string) string {
	result := prompt

	// Apply synth (Karpathy principles) if enabled
	if o.config.IsEnabled(ToolSynth) {
		if synth.ShouldActivate(prompt) || !strings.Contains(prompt, "## Karpathy") {
			result = synth.Inject(result)
		}
	}

	// Compress the prompt itself when the reducer is enabled.
	// This shrinks the system prompt + tool descriptions by removing
	// verbose language, collapsing whitespace, and compacting code blocks.
	if o.config.IsEnabled(ToolReducer) && len(result) > 500 {
		cfg := condense.DefaultConfig()
		level := o.config.GetLevel(ToolReducer)
		applyCondenseLevel(&cfg, level)
		originalLen := len(result)
		compressed, wasModified, _ := condense.CompressContent(result, cfg)
		if wasModified {
			result = compressed
			compressedLen := len(result)
			if originalLen > 0 {
				// Weight: system prompt is ~1/Nth of input savings but
				// update the rolling estimate so the sidebar sees real data.
				savings := float64(originalLen-compressedLen) / float64(originalLen) * 100
				// Blend with existing estimate (update rolling avg)
				if o.inputSavings == 0 {
					o.inputSavings = savings
				} else {
					o.inputSavings = (o.inputSavings + savings) / 2
				}
			}
		}
	}

	return result
}

// CompressResult holds the result of compressing an LLM response.
type CompressResult struct {
	Text                    string
	OriginalChars           int
	CompressedChars         int
	SavingsPercent          float64
	SavingsPercentFormatted string
}

// ProcessResponse applies enabled tools to an LLM response.
// This is called after receiving a response from the LLM.
func (o *Orchestrator) ProcessResponse(response string) CompressResult {
	result := CompressResult{Text: response}

	// Apply compression if enabled
	if o.config.IsEnabled(ToolCompress) {
		level := compress.LevelFull
		if l := o.config.GetLevel(ToolCompress); l != "" {
			level = compress.ParseLevel(l)
		}
		compressed := compress.Compress(result.Text, level)
		stats := compress.EstimateStats(result.Text, compressed)
		if stats.SavingsPercent > 10 {
			result.Text = compressed
		}
		result.OriginalChars = len(response)
		result.CompressedChars = len(result.Text)
		result.SavingsPercent = stats.SavingsPercent
		result.SavingsPercentFormatted = fmt.Sprintf("%.0f%%", stats.SavingsPercent)
	}

	return result
}

// TokenReducer runs all enabled token-reduction tools on message history
// before it is sent to the LLM. This is the single entry point for input-side
// token reduction — it coordinates condense, synth, filter, memory-tree,
// token-optimizer, and output compression based on the configured level.
//
// Level determines how aggressively tokens are reduced:
//
//   - "recommended": balanced savings (default). JSON arrays sampled to 20 items,
//     output compression at full, synth + filter + caching enabled.
//   - "light": minimal reduction. Only obvious waste trimmed, 30 items kept,
//     lite output compression.
//   - "moderate": good savings. 15 items kept, full output compression.
//   - "aggressive": maximum savings. 10 items kept, ultra output compression,
//     system prompts compressed, aggressive token optimization.
func (o *Orchestrator) TokenReducer(msgs []message.Message) ([]message.Message, int) {
	if !o.config.IsEnabled(ToolReducer) {
		return msgs, 0
	}

	level := o.config.GetLevel(ToolReducer)
	if level == "" {
		level = "recommended"
	}

	totalCompressed := 0

	// Track total characters before compression for savings measurement
	var originalTotalChars, compressedTotalChars int

	// 1. Compress all message content (system, user, tool) using condense
	if o.config.IsEnabled(ToolCondense) {
		cfg := condense.DefaultConfig()
		applyCondenseLevel(&cfg, level)

		compressed := 0
		result := make([]message.Message, len(msgs))
		copy(result, msgs)

		for i := range result {
			for j := range result[i].Parts {
				// Compress text content in any message role
				if tc, ok := result[i].Parts[j].(message.TextContent); ok {
					originalTotalChars += len(tc.Text)
					if len(tc.Text) >= cfg.MinContentLength {
						compressedContent, wasModified, _ := condense.CompressContent(tc.Text, cfg)
						if wasModified {
							tc.Text = compressedContent
							result[i].Parts[j] = tc
							compressed++
						}
					}
					compressedTotalChars += len(tc.Text)
				}
				// Compress tool result content
				if tr, ok := result[i].Parts[j].(message.ToolResult); ok {
					originalTotalChars += len(tr.Content)
					if len(tr.Content) >= cfg.MinContentLength {
						compressedContent, wasModified, _ := condense.CompressContent(tr.Content, cfg)
						if wasModified {
							tr.Content = compressedContent
							result[i].Parts[j] = tr
							compressed++
						}
					}
					compressedTotalChars += len(tr.Content)
				}
			}
		}
		if compressed > 0 {
			msgs = result
			totalCompressed += compressed
		}
	}

	// Update rolling input savings estimate from actual condense results
	if originalTotalChars > 0 && compressedTotalChars > 0 && originalTotalChars > compressedTotalChars {
		actualSavings := float64(originalTotalChars-compressedTotalChars) / float64(originalTotalChars) * 100
		if o.inputSavings == 0 {
			o.inputSavings = actualSavings
		} else {
			// Blend: 70% existing + 30% new to smooth over variance
			o.inputSavings = o.inputSavings*0.7 + actualSavings*0.3
		}
	}

	// 2. Future: memory-tree integration
	// if o.config.IsEnabled(ToolMemoryTree) { ... }

	// 3. Future: token-optimizer integration
	// if o.config.IsEnabled(ToolTokenOptimizer) { ... }

	return msgs, totalCompressed
}

// applyCondenseLevel maps the reducer level to condense configuration.
// Lower thresholds = more compression, especially for the initial large system prompt.
func applyCondenseLevel(cfg *condense.Config, level string) {
	switch level {
	case "aggressive":
		cfg.Crusher.MaxItemsAfterCrush = 10
		cfg.Crusher.MinTokensToCrush = 50
		cfg.MinContentLength = 50
	case "moderate":
		cfg.Crusher.MaxItemsAfterCrush = 15
		cfg.Crusher.MinTokensToCrush = 100
		cfg.MinContentLength = 100
	case "light":
		cfg.Crusher.MaxItemsAfterCrush = 30
		cfg.Crusher.MinTokensToCrush = 300
		cfg.MinContentLength = 200
	default: // "recommended"
		cfg.Crusher.MaxItemsAfterCrush = 20
		cfg.Crusher.MinTokensToCrush = 150
		cfg.MinContentLength = 100
	}
}

// CompressToolOutputs compresses tool result content in message history
// to reduce input token consumption. Handles JSON arrays, code, logs, etc.
func (o *Orchestrator) CompressToolOutputs(msgs []message.Message) ([]message.Message, int) {
	if !o.config.IsEnabled(ToolCondense) {
		return msgs, 0
	}

	cfg := condense.DefaultConfig()
	if l := o.config.GetLevel(ToolCondense); l != "" {
		switch l {
		case "aggressive":
			cfg.Crusher.MaxItemsAfterCrush = 10
			cfg.Crusher.MinTokensToCrush = 100
		case "moderate":
			cfg.Crusher.MaxItemsAfterCrush = 20
			cfg.Crusher.MinTokensToCrush = 300
		case "light":
			cfg.Crusher.MaxItemsAfterCrush = 30
			cfg.Crusher.MinTokensToCrush = 500
		}
	}

	compressed := 0
	result := make([]message.Message, len(msgs))
	copy(result, msgs)

	for i := range result {
		if result[i].Role != message.Tool {
			continue
		}
		for j := range result[i].Parts {
			tr, ok := result[i].Parts[j].(message.ToolResult)
			if !ok {
				continue
			}
			if len(tr.Content) < cfg.MinContentLength {
				continue
			}
			compressedContent, wasModified, _ := condense.CompressContent(tr.Content, cfg)
			if wasModified {
				tr.Content = compressedContent
				result[i].Parts[j] = tr
				compressed++
			}
		}
	}

	if compressed == 0 {
		return msgs, 0
	}
	return result, compressed
}

// InputSavings returns the measured input token savings percentage
// from condense compression during token reduction.
func (o *Orchestrator) InputSavings() float64 {
	return o.inputSavings
}

// SetInputSavings sets the estimated input token savings percentage.
func (o *Orchestrator) SetInputSavings(v float64) {
	o.inputSavings = v
}

// ReloadConfig re-reads the tools configuration from disk.
// This allows the orchestrator to pick up dashboard changes without restarting.
func (o *Orchestrator) ReloadConfig() {
	o.config = Load()
}

// IsMemoryTreeEnabled returns whether the memory tree tool is enabled.
func (o *Orchestrator) IsMemoryTreeEnabled() bool {
	return o.config.IsEnabled(ToolMemoryTree)
}

// IsTokenOptimizerEnabled returns whether the token optimizer tool is enabled.
func (o *Orchestrator) IsTokenOptimizerEnabled() bool {
	return o.config.IsEnabled(ToolTokenOptimizer)
}

// IsResponseCacheEnabled returns whether the response cache tool is enabled.
func (o *Orchestrator) IsResponseCacheEnabled() bool {
	return o.config.IsEnabled(ToolResponseCache)
}

// FilterCommand runs a command and filters its output.
func (o *Orchestrator) FilterCommand(args []string) (*filter.Result, error) {
	if !o.config.IsEnabled(ToolFilter) {
		return nil, fmt.Errorf("filter tool is disabled")
	}
	return filter.Filter(args, filter.LevelMinimal)
}

// FilterOutput applies command-specific output filtering to already-captured
// output text. This is used to filter command output that was already run
// (e.g., by the bash tool) without re-executing the command.
func (o *Orchestrator) FilterOutput(cmdName string, cmdArgs []string, output string) string {
	if !o.config.IsEnabled(ToolFilter) {
		return output
	}
	return filter.FilterOutput(cmdName, cmdArgs, output, filter.LevelMinimal)
}

// GenerateDiagram creates a diagram from JSON IR.
func (o *Orchestrator) GenerateDiagram(diagramJSON []byte) ([]byte, error) {
	if !o.config.IsEnabled(ToolViz) {
		return nil, fmt.Errorf("viz tool is disabled")
	}
	diagram, err := viz.Parse(diagramJSON)
	if err != nil {
		return nil, err
	}
	return viz.Generate(diagram)
}

// AnalyzeCode runs code analysis and builds a knowledge graph.
func (o *Orchestrator) AnalyzeCode(rootDir string) (*graph.Result, error) {
	if !o.config.IsEnabled(ToolGraph) {
		return nil, fmt.Errorf("graph tool is disabled")
	}
	analyzer := graph.NewAnalyzer(rootDir)
	return analyzer.Analyze()
}

// LoadUsageReport loads and summarizes token usage data.
func (o *Orchestrator) LoadUsageReport(logDir, period string) (*usage.Summary, error) {
	if !o.config.IsEnabled(ToolUsage) {
		return nil, fmt.Errorf("usage tool is disabled")
	}
	entries, err := usage.LoadDirectory(logDir)
	if err != nil {
		return nil, err
	}
	summary := usage.Summarize(entries, period)
	return &summary, nil
}

// GetStatus returns the status of all tools.
func (o *Orchestrator) GetStatus() map[ToolID]ToolStatus {
	return o.config.Status()
}

// SetToolEnabled enables or disables a tool.
func (o *Orchestrator) SetToolEnabled(id ToolID, enabled bool) {
	o.config.SetEnabled(id, enabled)
}

// SaveConfig saves the current configuration.
func (o *Orchestrator) SaveConfig() error {
	return o.config.Save()
}

// CheckCache looks for a cached response matching the prompt.
// Returns the cached entry and true if found.
// Checks both the local semantic cache (promptcache) and the
// SQLite response cache (if available and enabled).
func (o *Orchestrator) CheckCache(ctx context.Context, prompt string) (*promptcache.Entry, bool) {
	// Check local semantic prompt cache first
	if o.config.IsEnabled(ToolPromptCache) {
		c := o.initCache()
		if c != nil {
			if entry, hit := c.Check(ctx, prompt); hit {
				return entry, true
			}
		}
	}

	return nil, false
}

// StoreCache caches a prompt-response pair for future lookup.
func (o *Orchestrator) StoreCache(ctx context.Context, prompt, response string) error {
	if o.config.IsEnabled(ToolPromptCache) {
		c := o.initCache()
		if c != nil {
			return c.Store(ctx, prompt, response)
		}
	}

	return nil
}

// FlushCache flushes the prompt cache to disk.
func (o *Orchestrator) FlushCache() error {
	if o.cache == nil {
		return nil
	}
	return o.cache.Flush()
}

// CacheStats returns current prompt cache statistics.
func (o *Orchestrator) CacheStats() promptcache.Stats {
	if o.cache == nil {
		return promptcache.Stats{}
	}
	return o.cache.Stats()
}

// SetPromptCacheEmbedder sets the embedding function for semantic matching.
func (o *Orchestrator) SetPromptCacheEmbedder(embedder promptcache.Embedder) {
	if o.cache != nil {
		o.cache.SetEmbedder(embedder)
	}
}

// ClearPromptCache clears all cached entries.
func (o *Orchestrator) ClearPromptCache() error {
	if o.cache == nil {
		return nil
	}
	return o.cache.Clear()
}
