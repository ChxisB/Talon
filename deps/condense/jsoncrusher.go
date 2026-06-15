package condense

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"unicode"
)

// CrusherConfig controls JSON array compression behavior.
// Mirrors key settings from SmartCrusherConfig.
type CrusherConfig struct {
	Enabled              bool    // Master switch
	MinItemsToAnalyze    int     // Don't analyze arrays smaller than this (default 5)
	MinTokensToCrush     int     // Only crush content with more than this many tokens (default 200)
	MaxItemsAfterCrush   int     // Target maximum items in output (default 15)
	DedupIdenticalItems  bool    // Drop content-identical items (default true)
	FirstFraction        float64 // Fraction of slots for start of array (default 0.3)
	LastFraction         float64 // Fraction of slots for end of array (default 0.15)
	RelevanceThreshold   float64 // Score threshold for pinning items (default 0.3)
	PreserveChangePoints bool    // Keep structural change points (default true)
	UniquenessThreshold  float64 // Below this ratio, a field is nearly constant (default 0.1)
}

// DefaultCrusherConfig returns sensible defaults for JSON crushing.
func DefaultCrusherConfig() CrusherConfig {
	return CrusherConfig{
		Enabled:              true,
		MinItemsToAnalyze:    5,
		MinTokensToCrush:     200,
		MaxItemsAfterCrush:   15,
		DedupIdenticalItems:  true,
		FirstFraction:        0.3,
		LastFraction:         0.15,
		RelevanceThreshold:   0.3,
		PreserveChangePoints: true,
		UniquenessThreshold:  0.1,
	}
}

// CrushResult holds the output of crushing a JSON array.
type CrushResult struct {
	Compressed   string  // The compressed JSON
	Original     string  // Original JSON
	WasModified  bool    // Whether compression actually changed the content
	Strategy     string  // Strategy used: "passthrough", "dedup", "top_n", "smart_sample"
	ItemsBefore  int     // Original item count
	ItemsAfter   int     // Compressed item count
	SavingsRatio float64 // 0.0 = none, 1.0 = all removed
}

// CrushJSONArray compresses a JSON array of objects.
// It deduplicates, analyzes field structure, and selectively keeps important items.
func CrushJSONArray(content string, cfg CrusherConfig) CrushResult {
	result := CrushResult{
		Original:    content,
		Strategy:    "passthrough",
		ItemsBefore: 0,
		ItemsAfter:  0,
	}

	if !cfg.Enabled {
		result.Compressed = content
		return result
	}

	// Parse the JSON array
	var items []any
	if err := json.Unmarshal([]byte(content), &items); err != nil {
		result.Compressed = content
		return result
	}

	result.ItemsBefore = len(items)

	// Skip if below thresholds
	if len(items) < cfg.MinItemsToAnalyze {
		result.Compressed = content
		result.ItemsAfter = len(items)
		return result
	}

	// Check token count estimate (rough: ~4 chars per token)
	if len(content) < cfg.MinTokensToCrush*4 {
		result.Compressed = content
		result.ItemsAfter = len(items)
		return result
	}

	// Only crush arrays of objects
	if len(items) == 0 {
		result.Compressed = content
		result.ItemsAfter = 0
		return result
	}

	allObjects := true
	for _, item := range items {
		if _, ok := item.(map[string]any); !ok {
			allObjects = false
			break
		}
	}

	if !allObjects {
		result.Compressed = content
		result.ItemsAfter = len(items)
		return result
	}

	// Analyze structure to find important fields
	objects := make([]map[string]any, len(items))
	for i, item := range items {
		objects[i] = item.(map[string]any)
	}

	fieldInfo := analyzeFields(objects)

	// Find score field (if any) for relevance-based selection
	scoreField := findScoreField(fieldInfo)

	// Dedup identical items
	if cfg.DedupIdenticalItems && len(objects) > 1 {
		objects = dedupObjects(objects)
	}

	// If we're already small enough, return as-is
	if len(objects) <= cfg.MaxItemsAfterCrush {
		out, _ := json.Marshal(objects)
		result.Compressed = string(out)
		result.ItemsAfter = len(objects)
		result.WasModified = len(objects) < len(items)
		if result.WasModified {
			result.Strategy = "dedup"
			result.SavingsRatio = 1.0 - float64(len(objects))/float64(len(items))
		}
		return result
	}

	// Select items to keep
	kept := selectItems(objects, cfg, fieldInfo, scoreField)

	// Compress field names in the output
	kept = compressFieldNames(kept)

	out, _ := json.Marshal(kept)
	result.Compressed = string(out)
	result.ItemsAfter = len(kept)
	result.WasModified = true

	// Determine strategy
	if scoreField != "" {
		result.Strategy = "top_n"
	} else {
		result.Strategy = "smart_sample"
	}
	result.SavingsRatio = 1.0 - float64(len(kept))/float64(len(items))

	return result
}

// FieldInfo describes statistics for a single field across all objects.
type fieldInfo struct {
	Name           string
	IsNumeric      bool
	IsString       bool
	UniqueRatio    float64 // ratio of unique values (0-1)
	HasScore       bool    // field looks like a score (0-1 range)
	HasError       bool    // field looks like error status
	ConstantValue  any     // if all identical, the constant value
}

func analyzeFields(objects []map[string]any) []fieldInfo {
	if len(objects) == 0 {
		return nil
	}

	// Collect all field names, preserving order
	fieldNames := make([]string, 0)
	seen := make(map[string]bool)
	for _, obj := range objects {
		for k := range obj {
			if !seen[k] {
				seen[k] = true
				fieldNames = append(fieldNames, k)
			}
		}
	}

	var fields []fieldInfo
	for _, name := range fieldNames {
		fi := fieldInfo{Name: name}
		values := make([]any, 0, len(objects))
		uniqueSet := make(map[string]bool)
		isNumeric := true
		isString := true
		allSame := true
		var firstVal any
		allInZeroOne := true
		hasExplicitScoreName := name == "score" || name == "relevance" || name == "confidence" || name == "similarity"

		for i, obj := range objects {
			val := obj[name]
			values = append(values, val)

			// Track unique values
			valStr := fmt.Sprintf("%v", val)
			uniqueSet[valStr] = true

			// Type detection
			switch v := val.(type) {
			case float64:
				isString = false
				if !(v >= 0 && v <= 1) {
					allInZeroOne = false
				}
				// Fields named "score" etc. are always treated as score fields
				if hasExplicitScoreName {
					fi.HasScore = true
				}
			case string:
				isNumeric = false
			case bool:
				isNumeric = false
				isString = false
			case nil:
				// nulls don't affect type
			default:
				_ = v
				isNumeric = false
				isString = false
			}

			// Check for error indicators in string values
			if s, ok := val.(string); ok {
				lower := strings.ToLower(s)
				if strings.Contains(lower, "error") || strings.Contains(lower, "fail") || strings.Contains(lower, "fatal") {
					fi.HasError = true
				}
			}

			// Check if all same
			if i == 0 {
				firstVal = val
			} else {
				if val != firstVal {
					allSame = false
				}
			}
		}

		// Only set HasScore for non-explicitly-named fields if ALL values are in 0-1 range
		if !hasExplicitScoreName && allInZeroOne && isNumeric && name != "id" && name != "ID" {
			fi.HasScore = true
		}

		fi.IsNumeric = isNumeric
		fi.IsString = isString
		fi.UniqueRatio = float64(len(uniqueSet)) / float64(len(objects))
		if allSame {
			fi.ConstantValue = firstVal
		}

		fields = append(fields, fi)
	}

	return fields
}

// findScoreField looks for a numeric field with values in 0-1 range (relevance score).
func findScoreField(fields []fieldInfo) string {
	// First pass: look for fields named "score", "relevance", "confidence"
	for _, f := range fields {
		lower := strings.ToLower(f.Name)
		if (lower == "score" || lower == "relevance" || lower == "confidence" || lower == "similarity") && f.HasScore {
			return f.Name
		}
	}
	// Second pass: any numeric field with score characteristics
	for _, f := range fields {
		if f.HasScore && f.IsNumeric && f.UniqueRatio > 0.1 {
			return f.Name
		}
	}
	return ""
}

// dedupObjects removes duplicate objects from the slice.
func dedupObjects(objects []map[string]any) []map[string]any {
	seen := make(map[string]bool)
	result := make([]map[string]any, 0, len(objects))

	for _, obj := range objects {
		data, err := json.Marshal(obj)
		if err != nil {
			result = append(result, obj)
			continue
		}
		key := string(data)
		if !seen[key] {
			seen[key] = true
			result = append(result, obj)
		}
	}

	return result
}

// selectItems picks the most important items to keep within the budget.
func selectItems(objects []map[string]any, cfg CrusherConfig, fields []fieldInfo, scoreField string) []map[string]any {
	n := len(objects)
	effectiveMax := cfg.MaxItemsAfterCrush
	if effectiveMax > n {
		effectiveMax = n
	}

	// Allocate budget: first items, last items, then score-based
	firstCount := int(math.Round(float64(effectiveMax) * cfg.FirstFraction))
	lastCount := int(math.Round(float64(effectiveMax) * cfg.LastFraction))
	midBudget := effectiveMax - firstCount - lastCount
	if midBudget < 0 {
		midBudget = 0
	}

	// Calculate importance scores for each item
	scores := make([]float64, n)
	for i, obj := range objects {
		score := 0.0

		// Boost items with error indicators
		for _, f := range fields {
			if f.HasError {
				if val, ok := obj[f.Name]; ok {
					if s, ok := val.(string); ok {
						lower := strings.ToLower(s)
						if strings.Contains(lower, "error") || strings.Contains(lower, "fail") || strings.Contains(lower, "fatal") {
							score += 5.0
						}
					}
				}
			}
		}

		// Use explicit score field if available
		if scoreField != "" {
			if val, ok := obj[scoreField]; ok {
				if v, ok := val.(float64); ok && v >= cfg.RelevanceThreshold {
					// Only auto-keep items with very high scores (>0.8)
					if v > 0.8 {
						score += 20.0
					} else {
						score += v * 5.0
					}
				}
			}
		}

		scores[i] = score
	}

	// Build kept set
	keptSet := make(map[int]bool)

	// Always keep first items
	for i := 0; i < firstCount && i < n; i++ {
		keptSet[i] = true
	}

	// Always keep last items
	for i := n - lastCount; i < n; i++ {
		if i >= 0 {
			keptSet[i] = true
		}
	}

	// Fill mid-budget with highest-scoring items
	type scoredIdx struct {
		idx   int
		score float64
	}
	var midCandidates []scoredIdx
	for i := firstCount; i < n-lastCount; i++ {
		if !keptSet[i] {
			midCandidates = append(midCandidates, scoredIdx{idx: i, score: scores[i]})
		}
	}

	// Keep all high-score items regardless of budget
	var highScoreItems []scoredIdx
	for _, si := range midCandidates {
		if si.score >= 5.0 {
			keptSet[si.idx] = true
		} else {
			highScoreItems = append(highScoreItems, si)
		}
	}

	// Fill remaining slots from sorted candidates
	sort.Slice(highScoreItems, func(i, j int) bool {
		return highScoreItems[i].score > highScoreItems[j].score
	})

	remainingBudget := effectiveMax - len(keptSet)
	for i := 0; i < remainingBudget && i < len(highScoreItems); i++ {
		keptSet[highScoreItems[i].idx] = true
	}

	// If we still have budget, fill with evenly-spaced items
	if len(keptSet) < effectiveMax {
		fillEvenlySpaced(objects, keptSet, effectiveMax)
	}

	// Build result in original order
	result := make([]map[string]any, 0, len(keptSet))
	for i := 0; i < n; i++ {
		if keptSet[i] {
			result = append(result, objects[i])
		}
	}

	return result
}

// fillEvenlySpaced adds items at regular intervals to fill remaining budget.
func fillEvenlySpaced(objects []map[string]any, keptSet map[int]bool, target int) {
	n := len(objects)
	if n == 0 {
		return
	}

	current := len(keptSet)
	needed := target - current
	if needed <= 0 {
		return
	}

	// Collect eligible indices (not already kept)
	var eligible []int
	for i := 0; i < n; i++ {
		if !keptSet[i] {
			eligible = append(eligible, i)
		}
	}

	if len(eligible) == 0 {
		return
	}

	// Sample evenly from eligible
	step := float64(len(eligible)) / float64(needed+1)
	for i := 0; i < needed; i++ {
		idx := int(math.Round(float64(i+1) * step))
		if idx >= len(eligible) {
			idx = len(eligible) - 1
		}
		keptSet[eligible[idx]] = true
	}
}

// compressFieldNames shortens long field names in the output.
func compressFieldNames(objects []map[string]any) []map[string]any {
	if len(objects) == 0 {
		return objects
	}

	// Build field name mapping (only shorten names > 8 chars)
	fieldMap := make(map[string]string)
	for k := range objects[0] {
		if len(k) > 8 {
			short := shortenName(k)
			// Only use shortened name if it doesn't collide
			if !hasCollision(objects, short, k) {
				fieldMap[k] = short
			}
		}
	}

	if len(fieldMap) == 0 {
		return objects
	}

	// Apply mapping, preserving key order
	result := make([]map[string]any, len(objects))
	for i, obj := range objects {
		renamed := make(map[string]any)
		for k, v := range obj {
			if short, ok := fieldMap[k]; ok {
				renamed[short] = v
			} else {
				renamed[k] = v
			}
		}
		result[i] = renamed
	}

	return result
}

// shortenName creates an abbreviated field name.
func shortenName(name string) string {
	// Try common abbreviation patterns
	if strings.HasSuffix(name, "_count") {
		return strings.TrimSuffix(name, "_count") + "_c"
	}
	if strings.HasSuffix(name, "_name") {
		return strings.TrimSuffix(name, "_name") + "_n"
	}
	if strings.HasSuffix(name, "_id") {
		return strings.TrimSuffix(name, "_id") + "_id"
	}
	if strings.HasSuffix(name, "_type") {
		return strings.TrimSuffix(name, "_type") + "_t"
	}

	// Acronym: take first letter of each word
	parts := splitCamelOrSnake(name)
	if len(parts) >= 3 {
		var acronym strings.Builder
		for _, p := range parts {
			if len(p) > 0 {
				acronym.WriteRune(rune(p[0]))
			}
		}
		if acronym.Len() >= 2 && acronym.Len() < len(name) {
			return acronym.String()
		}
	}

	// Truncate to first 4 chars + last 2
	if len(name) > 10 {
		return name[:4] + name[len(name)-2:]
	}

	return name
}

// splitCamelOrSnake splits a name into words.
func splitCamelOrSnake(name string) []string {
	// Snake case
	if strings.Contains(name, "_") {
		return strings.Split(name, "_")
	}

	// CamelCase
	var parts []string
	var current strings.Builder
	for _, r := range name {
		if unicode.IsUpper(r) && current.Len() > 0 {
			parts = append(parts, strings.ToLower(current.String()))
			current.Reset()
		}
		current.WriteRune(r)
	}
	if current.Len() > 0 {
		parts = append(parts, strings.ToLower(current.String()))
	}
	return parts
}

// hasCollision checks if shortening a field name would collide with another field.
func hasCollision(objects []map[string]any, short, original string) bool {
	for _, obj := range objects {
		for k := range obj {
			if k != original && k == short {
				return true
			}
		}
	}
	return false
}
