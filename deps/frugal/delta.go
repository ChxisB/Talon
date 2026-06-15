package frugal

import (
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"strings"
)

const (
	maxDeltaChars   = 1500
	maxDeltaLines   = 2000
	maxCacheBytes   = 50 * 1024 // 50KB
)

// Code file extensions eligible for delta mode.
var codeExtensions = map[string]bool{
	".py": true, ".js": true, ".ts": true, ".jsx": true, ".tsx": true,
	".rb": true, ".rs": true, ".go": true, ".java": true, ".kt": true,
	".swift": true, ".c": true, ".cpp": true, ".h": true, ".hpp": true,
	".cs": true, ".php": true, ".sh": true, ".bash": true, ".zsh": true,
	".yaml": true, ".yml": true, ".toml": true, ".json": true, ".xml": true,
	".html": true, ".css": true, ".scss": true, ".less": true, ".sql": true,
	".md": true, ".txt": true, ".cfg": true, ".ini": true, ".vue": true,
	".svelte": true, ".astro": true, ".ex": true, ".exs": true, ".erl": true,
	".hs": true, ".lua": true, ".r": true, ".jl": true, ".dart": true,
	".scala": true, ".clj": true, ".tf": true, ".hcl": true,
}

var specialFiles = map[string]bool{
	"makefile": true, "dockerfile": true, "gemfile": true,
	"rakefile": true, "procfile": true, "jenkinsfile": true,
	".gitignore": true, ".dockerignore": true,
}

// DeltaStats holds delta computation statistics.
type DeltaStats struct {
	Added        int `json:"added"`
	Removed      int `json:"removed"`
	ChangedLines int `json:"changed_lines"`
}

// IsDeltaEligible checks if a file is eligible for delta mode.
func IsDeltaEligible(filePath string) bool {
	name := strings.ToLower(filepath.Base(filePath))
	if strings.HasPrefix(name, ".env") {
		return false
	}
	if specialFiles[name] {
		return true
	}
	ext := strings.ToLower(filepath.Ext(filePath))
	return codeExtensions[ext]
}

// ContentHash computes SHA-256 hash of text content.
func ContentHash(text string) string {
	h := sha256.Sum256([]byte(text))
	return fmt.Sprintf("%x", h)
}

// ComputeDelta computes a compact unified diff between old and new content.
// Returns (deltaText, stats, ok). ok=false if delta is not viable.
func ComputeDelta(oldContent, newContent, filename string) (string, *DeltaStats, bool) {
	if oldContent == newContent {
		return "", nil, false
	}

	oldLines := strings.SplitAfter(oldContent, "\n")
	newLines := strings.SplitAfter(newContent, "\n")

	if len(oldLines) > maxDeltaLines || len(newLines) > maxDeltaLines {
		return "", nil, false
	}

	// Simple diff: find first and last difference
	// For a proper unified diff we'd use a library, but this is a
	// simplified implementation suitable for the frugal package.
	firstDiff := -1
	lastDiffOld := len(oldLines)
	lastDiffNew := len(newLines)

	for i := 0; i < len(oldLines) && i < len(newLines); i++ {
		if oldLines[i] != newLines[i] {
			firstDiff = i
			break
		}
	}
	if firstDiff == -1 {
		if len(oldLines) != len(newLines) {
			firstDiff = min(len(oldLines), len(newLines))
		} else {
			return "", nil, false
		}
	}

	// Find end differences
	for i, j := len(oldLines)-1, len(newLines)-1; i >= 0 && j >= 0; i, j = i-1, j-1 {
		if oldLines[i] != newLines[j] {
			lastDiffOld = i + 1
			lastDiffNew = j + 1
			break
		}
	}

	added := 0
	removed := 0

	var b strings.Builder
	b.WriteString(sprintf("%s: lines changed", filename))
	b.WriteString("\n")

	// Build delta output
	for i := 0; i < len(oldLines) || i < len(newLines); i++ {
		if i < len(oldLines) && i >= firstDiff && i < lastDiffOld {
			if i < len(newLines) && i < lastDiffNew && oldLines[i] != newLines[i] {
				removed++
				b.WriteString("-" + oldLines[i])
				added++
				b.WriteString("+" + newLines[i])
			} else if i >= len(newLines) || i >= lastDiffNew {
				removed++
				b.WriteString("-" + oldLines[i])
			}
		} else if i >= len(oldLines) || i >= lastDiffOld {
			if i < len(newLines) && i < lastDiffNew {
				added++
				b.WriteString("+" + newLines[i])
			}
		}
	}

	if b.Len() > maxDeltaChars {
		return "", nil, false
	}

	stats := &DeltaStats{
		Added:        added,
		Removed:      removed,
		ChangedLines: added + removed,
	}

	return b.String(), stats, true
}
