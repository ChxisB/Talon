package condense

import (
	"encoding/json"
	"strings"
)

// Config controls the compression pipeline.
type Config struct {
	// Crusher config for JSON array compression.
	Crusher CrusherConfig

	// Minimum content length (bytes) to attempt compression.
	MinContentLength int

	// Minimum estimated tokens to attempt compression.
	MinTokens int
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		Crusher:          DefaultCrusherConfig(),
		MinContentLength: 100,
		MinTokens:        50,
	}
}

// CompressContent compresses a single content string based on its type.
// Returns the compressed content and a flag indicating if it was modified.
func CompressContent(content string, cfg Config) (string, bool, string) {
	if len(content) < cfg.MinContentLength {
		return content, false, "too_short"
	}

	detection := DetectContent(content)

	switch detection.ContentType {
	case ContentTypeJSONArray:
		if isDictArray, _ := detection.Metadata["is_dict_array"].(bool); isDictArray {
			result := CrushJSONArray(content, cfg.Crusher)
			if result.WasModified {
				return result.Compressed, true, "json_crush:" + result.Strategy
			}
			return content, false, "json_passthrough"
		}
		// Non-dict arrays: just minify
		minified := minifyJSON(content)
		if minified != content {
			return minified, true, "json_minify"
		}
		return content, false, "json_passthrough"

	case ContentTypeSourceCode:
		compressed := compressCode(content)
		return compressed, compressed != content, "code_compress"

	case ContentTypeSearchResults:
		compressed := compressSearchResults(content)
		return compressed, compressed != content, "search_compress"

	case ContentTypeBuildOutput:
		compressed := compressBuildOutput(content)
		return compressed, compressed != content, "log_compress"

	case ContentTypeGitDiff:
		compressed := compressDiff(content)
		return compressed, compressed != content, "diff_compress"

	case ContentTypeHTML:
		compressed := extractHTMLText(content)
		return compressed, compressed != content, "html_extract"

	default:
		// Plain text: compress for brevity
		compressed := compressText(content)
		return compressed, compressed != content, "text_compress"
	}
}

// CompressMessages compresses tool result content in a list of messages.
// Messages is a generic representation: each message is a map with "role" and "content".
func CompressMessages(messages []map[string]any, cfg Config) []map[string]any {
	modified := false
	result := make([]map[string]any, len(messages))

	for i, msg := range messages {
		msgCopy := make(map[string]any)
		for k, v := range msg {
			msgCopy[k] = v
		}
		result[i] = msgCopy

		role, _ := msgCopy["role"].(string)
		if role != "tool" && role != "user" {
			continue
		}

		content, ok := msgCopy["content"].(string)
		if !ok || len(content) < 100 {
			continue
		}

		compressed, wasModified, _ := CompressContent(content, cfg)
		if wasModified {
			msgCopy["content"] = compressed
			modified = true
		}
	}

	if !modified {
		return messages
	}
	return result
}

// minifyJSON removes whitespace from JSON content.
func minifyJSON(content string) string {
	var parsed any
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		return content
	}
	out, err := json.Marshal(parsed)
	if err != nil {
		return content
	}
	return string(out)
}

// compressCode reduces code verbosity by removing comments and collapsing blank lines.
func compressCode(content string) string {
	lines := strings.Split(content, "\n")
	var result []string
	inComment := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip single-line comments
		if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Handle multi-line comments
		if strings.HasPrefix(trimmed, "/*") {
			inComment = true
			if strings.Contains(trimmed, "*/") {
				inComment = false
			}
			continue
		}
		if inComment {
			if strings.Contains(trimmed, "*/") {
				inComment = false
			}
			continue
		}

		// Collapse multiple blank lines
		if trimmed == "" {
			if len(result) > 0 && result[len(result)-1] == "" {
				continue
			}
		}

		result = append(result, line)
	}

	return strings.Join(result, "\n")
}

// compressSearchResults reduces grep output by deduplicating file paths.
func compressSearchResults(content string) string {
	lines := strings.Split(content, "\n")
	seenFiles := make(map[string]int)
	var result []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		// Extract file prefix
		parts := strings.SplitN(trimmed, ":", 2)
		if len(parts) >= 2 {
			filePath := parts[0]
			seenFiles[filePath]++
			// Only show first 5 matches per file
			if seenFiles[filePath] > 5 {
				continue
			}
		}

		result = append(result, line)
	}

	return strings.Join(result, "\n")
}

// compressBuildOutput reduces verbose build logs.
func compressBuildOutput(content string) string {
	lines := strings.Split(content, "\n")
	var result []string
	consecutive := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip info/debug lines, keep errors and warnings
		if strings.HasPrefix(trimmed, "[INFO]") || strings.HasPrefix(trimmed, "[DEBUG]") || strings.HasPrefix(trimmed, "[TRACE]") {
			consecutive++
			if consecutive > 3 {
				continue
			}
		} else {
			consecutive = 0
		}

		result = append(result, line)
	}

	return strings.Join(result, "\n")
}

// compressDiff reduces large diffs by summarizing unchanged sections.
func compressDiff(content string) string {
	lines := strings.Split(content, "\n")
	var result []string
	skipped := 0

	for _, line := range lines {
		// Skip repeated context lines in large diffs
		if strings.HasPrefix(line, " ") && skipped < 3 {
			skipped++
			continue
		}
		if strings.HasPrefix(line, " ") && skipped >= 3 {
			continue
		}
		skipped = 0
		result = append(result, line)
	}

	return strings.Join(result, "\n")
}

// extractHTMLText strips HTML tags and extracts text content.
func extractHTMLText(content string) string {
	var result strings.Builder
	inTag := false
	inScript := false
	inStyle := false

	// Simple approach: strip tags, keep text
	for i := 0; i < len(content); i++ {
		if content[i] == '<' {
			inTag = true
			// Check for script/style
			if i+6 < len(content) {
				tag := strings.ToLower(content[i : i+7])
				if tag == "<script" || tag == "<SCRIPT" {
					inScript = true
				}
				if i+5 < len(content) {
					tag2 := strings.ToLower(content[i : i+6])
					if tag2 == "<style" || tag2 == "<STYLE" {
						inStyle = true
					}
				}
			}
			continue
		}
		if content[i] == '>' && inTag {
			inTag = false
			if inScript && i > 3 && (content[i-1] == 't' || content[i-1] == 'T') {
				// Check for </script>
				if i >= 8 && strings.Contains(strings.ToLower(content[i-8:i+1]), "/script") {
					inScript = false
				}
			}
			if inStyle && i > 3 && (content[i-1] == 'e' || content[i-1] == 'E') {
				if i >= 6 && strings.Contains(strings.ToLower(content[i-6:i+1]), "/style") {
					inStyle = false
				}
			}
			continue
		}
		if !inTag && !inScript && !inStyle {
			result.WriteByte(content[i])
		}
	}

	return strings.TrimSpace(result.String())
}

// compressText reduces verbosity in plain text.
func compressText(content string) string {
	lines := strings.Split(content, "\n")
	var result []string
	prevBlank := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if prevBlank {
				continue
			}
			prevBlank = true
		} else {
			prevBlank = false
		}
		result = append(result, line)
	}

	return strings.Join(result, "\n")
}
