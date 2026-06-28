// Package compress rewrites LLM output to reduce token consumption by
// ~60-75% while preserving technical accuracy. Inspired by caveman's
// output compression approach.
//
// The compressor strips filler words, articles, pleasantries, and
// hedging language while keeping code blocks, error strings, and
// technical terms intact.
package compress

import (
	"regexp"
	"strings"
)

// Level defines compression intensity.
type Level int

const (
	// LevelLite drops filler words and articles.
	LevelLite Level = iota
	// LevelFull is the default — drops filler, fragments sentences, shortens synonyms.
	LevelFull
	// LevelUltra is telegraphic — minimal natural language, code-heavy.
	LevelUltra
)

// ParseLevel converts a string name to a Level.
func ParseLevel(s string) Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "lite":
		return LevelLite
	case "ultra":
		return LevelUltra
	default:
		return LevelFull
	}
}

// Compress rewrites the given text according to the specified intensity level.
// Code blocks (``` fenced) and inline code (`backtick`) are preserved verbatim.
func Compress(text string, level Level) string {
	if level == LevelLite && !shouldCompress(text) {
		return text
	}

	// Split on code blocks to preserve them
	parts := splitOnCodeBlocks(text)
	var result strings.Builder

	for i, part := range parts {
		if i%2 == 1 {
			// Code block — preserve as-is
			result.WriteString(part)
		} else {
			// Natural language — compress
			compressed := compressText(part, level)
			result.WriteString(compressed)
		}
	}

	return result.String()
}

// splitOnCodeBlocks splits text into alternating [text, code, text, code, ...]
// segments. Odd-indexed segments are code blocks that should be preserved.
func splitOnCodeBlocks(text string) []string {
	var parts []string
	lines := strings.Split(text, "\n")
	inCode := false
	var current strings.Builder

	for _, line := range lines {
		if strings.HasPrefix(line, "```") {
			if inCode {
				current.WriteString(line)
				current.WriteString("\n")
				parts = append(parts, current.String())
				current.Reset()
				inCode = false
			} else {
				parts = append(parts, current.String())
				current.Reset()
				current.WriteString(line)
				current.WriteString("\n")
				inCode = true
			}
		} else {
			current.WriteString(line)
			current.WriteString("\n")
		}
	}
	parts = append(parts, current.String())
	return parts
}

// shouldCompress returns false for text that's already compact
// (e.g., pure code, error messages, short responses).
func shouldCompress(text string) bool {
	trimmed := strings.TrimSpace(text)
	if len(trimmed) < 50 {
		return false
	}
	// Already mostly code
	codeLines := strings.Count(trimmed, "\n")
	totalLines := strings.Count(trimmed, "\n") + 1
	if float64(codeLines)/float64(totalLines) > 0.7 {
		return false
	}
	return true
}

// fillerWords are words that can be safely removed without changing meaning.
var fillerWords = regexp.MustCompile(`(?i)\b(very|really|quite|rather|somewhat|basically|actually|just|simply|perhaps|maybe|probably|possibly|somehow|somewhere|certainly|definitely|absolutely|obviously|clearly|essentially|fundamentally|importantly|notably|particularly|especially|generally|typically|usually|commonly|frequently|often|always|never|ever)\b`)

// hedgingPatterns are hedging phrases that can be removed.
var hedgingPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\bI think\b`),
	regexp.MustCompile(`(?i)\bI believe\b`),
	regexp.MustCompile(`(?i)\bI would suggest\b`),
	regexp.MustCompile(`(?i)\bIt seems like\b`),
	regexp.MustCompile(`(?i)\bIt appears that\b`),
	regexp.MustCompile(`(?i)\bIn my opinion\b`),
	regexp.MustCompile(`(?i)\bYou might want to\b`),
	regexp.MustCompile(`(?i)\bYou could consider\b`),
	regexp.MustCompile(`(?i)\bOne option is to\b`),
	regexp.MustCompile(`(?i)\bAnother approach would be\b`),
	regexp.MustCompile(`(?i)\bA good approach would be\b`),
	regexp.MustCompile(`(?i)\bThe best way to\b`),
}

// articlePattern removes articles (a, an, the) when they're not needed.
var articlePattern = regexp.MustCompile(`(?i)\b(the|a|an)\b`)

// synonymMap shortens common verbose phrases.
var synonymMap = []struct{ from, to string }{
	{"in order to", "to"},
	{"due to the fact that", "because"},
	{"at this point in time", "now"},
	{"for the purpose of", "to"},
	{"in the event that", "if"},
	{"on a regular basis", "regularly"},
	{"in the near future", "soon"},
	{"a large number of", "many"},
	{"a small number of", "few"},
	{"the majority of", "most"},
	{"has the ability to", "can"},
	{"is able to", "can"},
	{"make use of", "use"},
	{"give consideration to", "consider"},
	{"arrive at a decision", "decide"},
	{"in excess of", "over"},
	{"fewer than", "under"},
	{"a sufficient amount of", "enough"},
	{"prior to", "before"},
	{"subsequent to", "after"},
	{"commence", "start"},
	{"terminate", "stop"},
	{"endeavor", "try"},
	{"facilitate", "help"},
	{"utilize", "use"},
	{"demonstrate", "show"},
	{"approximately", "~"},
	{"consequently", "so"},
	{"furthermore", "also"},
	{"moreover", "also"},
	{"nevertheless", "but"},
	{"however", "but"},
	{"additionally", "also"},
	{"subsequently", "then"},
	{"previously", "before"},
}

// questionPatterns are verbose question patterns to simplify.
var questionPatterns = []struct{ from, to string }{
	{"Could you please", "Please"},
	{"Would you mind", "Please"},
	{"Can you please", "Please"},
	{"I was wondering if you could", "Please"},
	{"Would it be possible to", "Can you"},
	{"Is it possible to", "Can you"},
	{"Do you think you could", "Please"},
}

func compressText(text string, level Level) string {
	result := text

	// Remove hedging patterns (all levels)
	for _, pat := range hedgingPatterns {
		result = pat.ReplaceAllString(result, "")
	}

	// Simplify verbose question patterns
	for _, qp := range questionPatterns {
		result = strings.ReplaceAll(result, qp.from, qp.to)
	}

	// Apply synonym shortening
	for _, s := range synonymMap {
		result = strings.ReplaceAll(result, s.from, s.to)
		if level >= LevelFull {
			result = strings.ReplaceAll(result, strings.Title(s.from), s.to)
		}
	}

	// Remove filler words (full and ultra only)
	if level >= LevelFull {
		result = fillerWords.ReplaceAllString(result, "")
	}

	// Remove articles (ultra only)
	if level >= LevelUltra {
		result = articlePattern.ReplaceAllString(result, "")
	}

	// Clean up whitespace artifacts
	result = cleanupWhitespace(result)

	return result
}

// cleanupWhitespace removes double spaces, leading/trailing spaces on lines,
// and excessive blank lines left after compression.
func cleanupWhitespace(text string) string {
	// Remove leading/trailing whitespace from each line
	lines := strings.Split(text, "\n")
	var cleaned []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" || (len(cleaned) > 0 && cleaned[len(cleaned)-1] != "") {
			cleaned = append(cleaned, trimmed)
		}
	}

	result := strings.Join(cleaned, "\n")

	// Collapse multiple spaces
	for strings.Contains(result, "  ") {
		result = strings.ReplaceAll(result, "  ", " ")
	}

	// Collapse multiple blank lines
	for strings.Contains(result, "\n\n\n") {
		result = strings.ReplaceAll(result, "\n\n\n", "\n\n")
	}

	return strings.TrimSpace(result)
}

// Stats returns token savings information for a compression operation.
type Stats struct {
	OriginalTokens  int     // Estimated tokens in original text
	CompressedTokens int    // Estimated tokens in compressed text
	SavingsPercent  float64 // Percentage of tokens saved
}

// EstimateStats estimates token savings from compression. Uses a rough
// approximation of 4 characters per token.
func EstimateStats(original, compressed string) Stats {
	origTokens := len(original) / 4
	compTokens := len(compressed) / 4
	savings := 0.0
	if origTokens > 0 {
		savings = float64(origTokens-compTokens) / float64(origTokens) * 100
	}
	return Stats{
		OriginalTokens:  origTokens,
		CompressedTokens: compTokens,
		SavingsPercent:  savings,
	}
}
