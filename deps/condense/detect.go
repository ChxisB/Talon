// Package condense detects content types and selects compression strategies
// to reduce token consumption for LLM prompts. Ported concepts from context
// compression research.
package condense

import (
	"encoding/json"
	"math"
	"regexp"
	"strings"
)

// ContentType represents the type of content to compress.
type ContentType int

const (
	ContentTypePlainText    ContentType = iota // Generic text (fallback)
	ContentTypeJSONArray                       // JSON array of objects
	ContentTypeSourceCode                      // Source code
	ContentTypeSearchResults                   // grep/ripgrep output
	ContentTypeBuildOutput                     // Build/test/log output
	ContentTypeGitDiff                         // Unified diff format
	ContentTypeHTML                            // HTML content
)

func (c ContentType) String() string {
	switch c {
	case ContentTypeJSONArray:
		return "json_array"
	case ContentTypeSourceCode:
		return "source_code"
	case ContentTypeSearchResults:
		return "search_results"
	case ContentTypeBuildOutput:
		return "build_output"
	case ContentTypeGitDiff:
		return "git_diff"
	case ContentTypeHTML:
		return "html"
	default:
		return "plain_text"
	}
}

// DetectionResult holds the result of content type detection.
type DetectionResult struct {
	ContentType ContentType
	Confidence  float64
	Metadata    map[string]any
}

// Content language patterns for code detection.
var codePatterns = map[string][]*regexp.Regexp{
	"python": {
		regexp.MustCompile(`^\s*(def|class|import|from|async def)\s+\w+`),
		regexp.MustCompile(`^\s*@\w+`),
		regexp.MustCompile(`^\s*"""`),
		regexp.MustCompile(`^\s*if __name__\s*==`),
	},
	"javascript": {
		regexp.MustCompile(`^\s*(function|const|let|var|class|import|export)\s+`),
		regexp.MustCompile(`^\s*(async\s+function|=>\s*\{)`),
		regexp.MustCompile(`^\s*module\.exports`),
	},
	"typescript": {
		regexp.MustCompile(`^\s*(interface|type|enum|namespace)\s+\w+`),
		regexp.MustCompile(`:\s*(string|number|boolean|any|void)\b`),
	},
	"go": {
		regexp.MustCompile(`^\s*(func|type|package|import)\s+`),
		regexp.MustCompile(`^\s*func\s+\([^)]+\)\s+\w+`),
	},
	"rust": {
		regexp.MustCompile(`^\s*(fn|struct|enum|impl|mod|use|pub)\s+`),
		regexp.MustCompile(`^\s*#\[`),
	},
	"java": {
		regexp.MustCompile(`^\s*(public|private|protected)\s+(class|interface|enum)`),
		regexp.MustCompile(`^\s*@\w+`),
		regexp.MustCompile(`^\s*package\s+[\w.]+;`),
	},
}

// Patterns for search results (grep -n style).
var searchResultPattern = regexp.MustCompile(`^[^\s:]+:\d+:`)

// Patterns for git diff detection.
var diffHeaderPattern = regexp.MustCompile(`^(diff --git|diff --combined |diff --cc |--- a/|@@\s+-\d+,\d+\s+\+\d+,\d+\s+@@|@@@+\s+-\d+(?:,\d+)?\s+(?:-\d+(?:,\d+)?\s+)+\+\d+(?:,\d+)?\s+@@@+)`)

var diffChangePattern = regexp.MustCompile(`^[+-][^+-]`)

// HTML detection patterns.
var htmlDoctypePattern = regexp.MustCompile(`(?i)^\s*<!doctype\s+html`)
var htmlTagPattern = regexp.MustCompile(`(?i)<html[\s>]`)
var htmlHeadPattern = regexp.MustCompile(`(?i)<head[\s>]`)
var htmlBodyPattern = regexp.MustCompile(`(?i)<body[\s>]`)
var htmlStructuralTags = regexp.MustCompile(`(?i)<(div|span|script|style|link|meta|nav|header|footer|aside|article|section|main)[\s>]`)

// Patterns for log/build output.
var logPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\b(ERROR|FAIL|FAILED|FATAL|CRITICAL)\b`),
	regexp.MustCompile(`(?i)\b(WARN|WARNING)\b`),
	regexp.MustCompile(`(?i)\b(INFO|DEBUG|TRACE)\b`),
	regexp.MustCompile(`^\s*\d{4}-\d{2}-\d{2}`),
	regexp.MustCompile(`^\s*\[\d{2}:\d{2}:\d{2}\]`),
	regexp.MustCompile(`^={3,}|^-{3,}`),
	regexp.MustCompile(`^\s*PASSED|^\s*FAILED|^\s*SKIPPED`),
	regexp.MustCompile(`^npm ERR!|^yarn error|^cargo error`),
	regexp.MustCompile(`Traceback \(most recent call last\)`),
	regexp.MustCompile(`^\w*(Error|Exception):`),
	regexp.MustCompile(`^\s*at\s+[\w.$]+\(`),
}

// DetectContent detects the type of content for appropriate compression.
func DetectContent(content string) DetectionResult {
	if strings.TrimSpace(content) == "" {
		return DetectionResult{ContentType: ContentTypePlainText, Confidence: 0.0, Metadata: map[string]any{}}
	}

	// 1. Try JSON first (highest priority)
	if r := tryDetectJSON(content); r != nil {
		return *r
	}

	// 2. Check for diff
	if r := tryDetectDiff(content); r != nil && r.Confidence >= 0.7 {
		return *r
	}

	// 3. Check for HTML
	if r := tryDetectHTML(content); r != nil && r.Confidence >= 0.7 {
		return *r
	}

	// 4. Check for search results
	if r := tryDetectSearch(content); r != nil && r.Confidence >= 0.6 {
		return *r
	}

	// 5. Check for build/log output
	if r := tryDetectLog(content); r != nil && r.Confidence >= 0.5 {
		return *r
	}

	// 6. Check for source code
	if r := tryDetectCode(content); r != nil && r.Confidence >= 0.5 {
		return *r
	}

	return DetectionResult{ContentType: ContentTypePlainText, Confidence: 0.5, Metadata: map[string]any{}}
}

func tryDetectJSON(content string) *DetectionResult {
	content = strings.TrimSpace(content)
	if !strings.HasPrefix(content, "[") {
		return nil
	}

	var parsed any
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		return nil
	}

	if arr, ok := parsed.([]any); ok {
		result := DetectionResult{
			ContentType: ContentTypeJSONArray,
			Confidence:  0.8,
			Metadata:    map[string]any{"item_count": len(arr), "is_dict_array": false},
		}
		// Check if all items are objects
		if len(arr) > 0 {
			allDicts := true
			for _, item := range arr {
				if _, ok := item.(map[string]any); !ok {
					allDicts = false
					break
				}
			}
			if allDicts {
				result.Confidence = 1.0
				result.Metadata["is_dict_array"] = true
			}
		}
		return &result
	}

	return nil
}

func tryDetectDiff(content string) *DetectionResult {
	lines := strings.SplitN(content, "\n", 500)
	headerMatches := 0
	changeMatches := 0

	for _, line := range lines {
		if diffHeaderPattern.MatchString(line) {
			headerMatches++
		}
		if diffChangePattern.MatchString(line) {
			changeMatches++
		}
	}

	if headerMatches == 0 {
		return nil
	}

	confidence := math.Min(1.0, 0.5+float64(headerMatches)*0.2+float64(changeMatches)*0.05)
	return &DetectionResult{
		ContentType: ContentTypeGitDiff,
		Confidence:  confidence,
		Metadata:    map[string]any{"header_matches": headerMatches, "change_lines": changeMatches},
	}
}

func tryDetectHTML(content string) *DetectionResult {
	sample := content
	if len(sample) > 3000 {
		sample = sample[:3000]
	}

	hasDoctype := htmlDoctypePattern.MatchString(sample)
	hasHTMLTag := htmlTagPattern.MatchString(sample)
	hasHead := htmlHeadPattern.MatchString(sample)
	hasBody := htmlBodyPattern.MatchString(sample)
	structuralMatches := len(htmlStructuralTags.FindAllString(sample, -1))

	if !hasDoctype && !hasHTMLTag && structuralMatches < 3 {
		return nil
	}

	confidence := 0.0
	if hasDoctype {
		confidence += 0.5
	}
	if hasHTMLTag {
		confidence += 0.3
	}
	if hasHead {
		confidence += 0.1
	}
	if hasBody {
		confidence += 0.1
	}
	confidence += math.Min(0.3, float64(structuralMatches)*0.03)

	if confidence < 0.5 {
		return nil
	}

	return &DetectionResult{
		ContentType: ContentTypeHTML,
		Confidence:  math.Min(1.0, confidence),
		Metadata: map[string]any{
			"has_doctype":       hasDoctype,
			"has_html_tag":      hasHTMLTag,
			"structural_tags":   structuralMatches,
		},
	}
}

func tryDetectSearch(content string) *DetectionResult {
	lines := strings.SplitN(content, "\n", 100)
	if len(lines) == 0 {
		return nil
	}

	matchingLines := 0
	nonEmpty := 0
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		nonEmpty++
		if searchResultPattern.MatchString(line) {
			matchingLines++
		}
	}

	if matchingLines == 0 || nonEmpty == 0 {
		return nil
	}

	ratio := float64(matchingLines) / float64(nonEmpty)
	if ratio < 0.3 {
		return nil
	}

	return &DetectionResult{
		ContentType: ContentTypeSearchResults,
		Confidence:  math.Min(1.0, 0.4+ratio*0.6),
		Metadata:    map[string]any{"matching_lines": matchingLines, "total_lines": nonEmpty},
	}
}

func tryDetectLog(content string) *DetectionResult {
	lines := strings.SplitN(content, "\n", 200)
	if len(lines) == 0 {
		return nil
	}

	patternMatches := 0
	errorMatches := 0

	for _, line := range lines {
		for i, pat := range logPatterns {
			if pat.MatchString(line) {
				patternMatches++
				if i < 2 {
					errorMatches++
				}
				break
			}
		}
	}

	if patternMatches == 0 {
		return nil
	}

	nonEmpty := 0
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			nonEmpty++
		}
	}
	if nonEmpty == 0 {
		return nil
	}

	ratio := float64(patternMatches) / float64(nonEmpty)
	if ratio < 0.1 {
		return nil
	}

	return &DetectionResult{
		ContentType: ContentTypeBuildOutput,
		Confidence:  math.Min(1.0, 0.3+ratio*0.5+float64(errorMatches)*0.05),
		Metadata:    map[string]any{"pattern_matches": patternMatches, "error_matches": errorMatches},
	}
}

func tryDetectCode(content string) *DetectionResult {
	lines := strings.SplitN(content, "\n", 100)
	if len(lines) == 0 {
		return nil
	}

	languageScores := make(map[string]int)
	for _, line := range lines {
		for lang, patterns := range codePatterns {
			for _, pat := range patterns {
				if pat.MatchString(line) {
					languageScores[lang]++
					break
				}
			}
		}
	}

	if len(languageScores) == 0 {
		return nil
	}

	bestLang := ""
	bestScore := 0
	for lang, score := range languageScores {
		if score > bestScore {
			bestScore = score
			bestLang = lang
		}
	}

	if bestScore < 2 {
		return nil
	}

	nonEmpty := 0
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			nonEmpty++
		}
	}

	ratio := float64(bestScore) / math.Max(float64(nonEmpty), 1)
	confidence := math.Min(1.0, 0.4+ratio*0.4+float64(bestScore)*0.02)

	return &DetectionResult{
		ContentType: ContentTypeSourceCode,
		Confidence:  confidence,
		Metadata:    map[string]any{"language": bestLang, "pattern_matches": bestScore},
	}
}

// IsJSONArrayOfDicts is a quick check if content is a JSON array of dicts.
func IsJSONArrayOfDicts(content string) bool {
	r := DetectContent(content)
	return r.ContentType == ContentTypeJSONArray && r.Metadata["is_dict_array"] == true
}
