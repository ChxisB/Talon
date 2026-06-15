package frugal

import "math"

// Characters per token for non-CJK text (calibrated mean across source code).
const codeCharsPerToken = 3.3

// EstimateTokens estimates the token count for a text string.
// CJK characters at ~1 token each, Latin/code at ~3.3 chars per token.
func EstimateTokens(text string) int {
	if text == "" {
		return 0
	}

	// Fast path: pure ASCII (most source code)
	if isASCII(text) {
		return max(1, int(math.Ceil(float64(len(text))/codeCharsPerToken)))
	}

	cjk := countCJK(text)
	other := len(text) - cjk
	est := int(math.Ceil(float64(other)/codeCharsPerToken)) + cjk
	return max(1, est)
}

// EstimateTokensFromBytes estimates tokens from a byte count.
// Assumes predominantly single-byte content (ASCII/code).
func EstimateTokensFromBytes(n int) int {
	if n <= 0 {
		return 0
	}
	return max(1, int(math.Ceil(float64(n)/codeCharsPerToken)))
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x80 {
			return false
		}
	}
	return true
}

func countCJK(s string) int {
	var n int
	for _, r := range s {
		if isCJK(r) {
			n++
		}
	}
	return n
}

func isCJK(r rune) bool {
	return (0x3040 <= r && r <= 0x30FF) || // Hiragana + Katakana
		(0x3400 <= r && r <= 0x4DBF) || // CJK Ext A
		(0x4E00 <= r && r <= 0x9FFF) || // CJK Unified
		(0xAC00 <= r && r <= 0xD7A3) || // Hangul syllables
		(0xF900 <= r && r <= 0xFAFF) || // CJK compatibility ideographs
		(0xFF00 <= r && r <= 0xFFEF) || // Half/full-width forms
		(0x20000 <= r && r <= 0x2FA1F) // CJK Ext B-F + supplement
}
