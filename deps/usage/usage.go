// Package usage parses JSONL session logs from AI coding agents and
// generates token usage reports with cost calculations.
//
// Reference: ccusage (MIT License)
package usage

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Entry represents a single token usage event.
type Entry struct {
	Timestamp    time.Time `json:"timestamp"`
	Model        string    `json:"model"`
	InputTokens  int       `json:"input_tokens"`
	OutputTokens int       `json:"output_tokens"`
	CacheRead    int       `json:"cache_read_tokens,omitempty"`
	CacheCreate  int       `json:"cache_creation_tokens,omitempty"`
	Cost         float64   `json:"cost,omitempty"`
	SessionID    string    `json:"session_id,omitempty"`
	Project      string    `json:"project,omitempty"`
}

// Session groups entries by session.
type Session struct {
	ID           string
	Entries      []Entry
	StartTime    time.Time
	EndTime      time.Time
	TotalInput   int
	TotalOutput  int
	TotalCost    float64
	Model        string
}

// DailyReport aggregates usage by day.
type DailyReport struct {
	Date         string
	Sessions     int
	TotalInput   int
	TotalOutput  int
	TotalCost    float64
	Models       map[string]int // model -> total tokens
}

// Summary holds overall usage statistics.
type Summary struct {
	Period       string
	TotalInput   int
	TotalOutput  int
	TotalCost    float64
	Sessions     int
	Days         int
	AvgPerDay    float64
	Models       map[string]int
	TopModels    []ModelUsage
}

// ModelUsage holds per-model token counts.
type ModelUsage struct {
	Model  string
	Tokens int
}

// Pricing holds per-model pricing (per 1M tokens, USD).
var Pricing = map[string]struct {
	Input  float64
	Output float64
}{
	// OpenAI
	"gpt-4o":           {2.50, 10.00},
	"gpt-4o-mini":      {0.15, 0.60},
	"o3":               {10.00, 40.00},
	"o4-mini":          {1.10, 4.40},
	// Anthropic
	"claude-sonnet-4":  {3.00, 15.00},
	"claude-sonnet-4.5":{3.00, 15.00},
	"claude-haiku-4":   {0.25, 1.25},
	"claude-opus-4":    {15.00, 75.00},
	// Google
	"gemini-2.0-flash": {0.075, 0.30},
	"gemini-2.5-flash": {0.15, 0.60},
	"gemini-2.5-pro":   {1.25, 10.00},
	// DeepSeek
	"deepseek-chat":    {0.15, 0.60},
	"deepseek-v4-pro":  {3.45, 8.55},
	// Groq
	"llama-3.3-70b":    {0.00, 0.00},
	"mixtral-8x7b":     {0.00, 0.00},
}

// CalculateCost computes the cost for a given model and token counts.
func CalculateCost(model string, input, output int) float64 {
	// Normalize model name
	norm := strings.ToLower(model)
	for name, p := range Pricing {
		if strings.Contains(norm, name) {
			return (float64(input)/1_000_000)*p.Input + (float64(output)/1_000_000)*p.Output
		}
	}
	// Default pricing (moderate)
	return (float64(input)/1_000_000)*3.0 + (float64(output)/1_000_000)*15.0
}

// ParseClaudeLog parses a Claude Code JSONL session log.
func ParseClaudeLog(path string) ([]Entry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var entries []Entry
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var raw map[string]interface{}
		if err := json.Unmarshal(line, &raw); err != nil {
			continue
		}

		entry := Entry{}

		// Parse timestamp
		if ts, ok := raw["timestamp"].(string); ok {
			if t, err := time.Parse(time.RFC3339, ts); err == nil {
				entry.Timestamp = t
			}
		}

		// Parse model
		if m, ok := raw["model"].(string); ok {
			entry.Model = m
		}

		// Parse token usage
		if usage, ok := raw["usage"].(map[string]interface{}); ok {
			if v, ok := usage["input_tokens"].(float64); ok {
				entry.InputTokens = int(v)
			}
			if v, ok := usage["output_tokens"].(float64); ok {
				entry.OutputTokens = int(v)
			}
			if v, ok := usage["cache_read_input_tokens"].(float64); ok {
				entry.CacheRead = int(v)
			}
			if v, ok := usage["cache_creation_input_tokens"].(float64); ok {
				entry.CacheCreate = int(v)
			}
		}

		// Calculate cost
		entry.Cost = CalculateCost(entry.Model, entry.InputTokens, entry.OutputTokens)

		entries = append(entries, entry)
	}

	return entries, scanner.Err()
}

// ParseOpenCodeLog parses an OpenCode session log.
func ParseOpenCodeLog(path string) ([]Entry, error) {
	// OpenCode uses a similar JSONL format
	return ParseClaudeLog(path)
}

// LoadDirectory loads all session logs from a directory.
func LoadDirectory(dir string) ([]Entry, error) {
	var allEntries []Entry

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".jsonl") {
			return nil
		}

		entries, err := ParseClaudeLog(path)
		if err != nil {
			return nil // Skip parse errors
		}
		allEntries = append(allEntries, entries...)
		return nil
	})

	return allEntries, err
}

// GroupByDay groups entries by date.
func GroupByDay(entries []Entry) []DailyReport {
	dayMap := make(map[string]*DailyReport)

	for _, e := range entries {
		date := e.Timestamp.Format("2006-01-02")
		if _, ok := dayMap[date]; !ok {
			dayMap[date] = &DailyReport{
				Date:   date,
				Models: make(map[string]int),
			}
		}
		dr := dayMap[date]
		dr.TotalInput += e.InputTokens
		dr.TotalOutput += e.OutputTokens
		dr.TotalCost += e.Cost
		dr.Models[e.Model] += e.InputTokens + e.OutputTokens
	}

	var reports []DailyReport
	for _, dr := range dayMap {
		reports = append(reports, *dr)
	}

	sort.Slice(reports, func(i, j int) bool {
		return reports[i].Date > reports[j].Date
	})

	return reports
}

// GroupBySession groups entries by session ID.
func GroupBySession(entries []Entry) []Session {
	sessionMap := make(map[string]*Session)

	for _, e := range entries {
		sid := e.SessionID
		if sid == "" {
			sid = e.Timestamp.Format("2006-01-02T15")
		}
		if _, ok := sessionMap[sid]; !ok {
			sessionMap[sid] = &Session{
				ID:    sid,
				Model: e.Model,
			}
		}
		s := sessionMap[sid]
		s.Entries = append(s.Entries, e)
		s.TotalInput += e.InputTokens
		s.TotalOutput += e.OutputTokens
		s.TotalCost += e.Cost
		if e.Timestamp.Before(s.StartTime) || s.StartTime.IsZero() {
			s.StartTime = e.Timestamp
		}
		if e.Timestamp.After(s.EndTime) {
			s.EndTime = e.Timestamp
		}
	}

	var sessions []Session
	for _, s := range sessionMap {
		sessions = append(sessions, *s)
	}

	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].StartTime.After(sessions[j].StartTime)
	})

	return sessions
}

// Summarize creates an overall summary from entries.
func Summarize(entries []Entry, period string) Summary {
	s := Summary{
		Period:  period,
		Models:  make(map[string]int),
	}

	days := make(map[string]bool)
	for _, e := range entries {
		s.TotalInput += e.InputTokens
		s.TotalOutput += e.OutputTokens
		s.TotalCost += e.Cost
		s.Models[e.Model] += e.InputTokens + e.OutputTokens
		days[e.Timestamp.Format("2006-01-02")] = true
	}

	s.Days = len(days)
	s.Sessions = len(GroupBySession(entries))
	if s.Days > 0 {
		s.AvgPerDay = s.TotalCost / float64(s.Days)
	}

	// Build top models
	for model, tokens := range s.Models {
		s.TopModels = append(s.TopModels, ModelUsage{Model: model, Tokens: tokens})
	}
	sort.Slice(s.TopModels, func(i, j int) bool {
		return s.TopModels[i].Tokens > s.TopModels[j].Tokens
	})
	if len(s.TopModels) > 5 {
		s.TopModels = s.TopModels[:5]
	}

	return s
}

// FormatCost formats a cost value as USD.
func FormatCost(cost float64) string {
	if cost < 0.01 {
		return fmt.Sprintf("$%.4f", cost)
	}
	if cost < 1.0 {
		return fmt.Sprintf("$%.3f", cost)
	}
	return fmt.Sprintf("$%.2f", cost)
}

// FormatTokens formats token counts with K/M suffixes.
func FormatTokens(tokens int) string {
	if tokens >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(tokens)/1_000_000)
	}
	if tokens >= 1_000 {
		return fmt.Sprintf("%.1fK", float64(tokens)/1_000)
	}
	return fmt.Sprintf("%d", tokens)
}
