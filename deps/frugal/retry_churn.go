package frugal

// detectRetryChurn detects sessions with excessive retry/API error churn.
func detectRetryChurn(data *SessionData) []Finding {
	turns := data.Turns
	if len(turns) < 3 {
		return nil
	}

	retryCount := 0
	retryTokens := 0
	consecutiveErrors := 0
	maxConsecutiveErrors := 0

	for _, t := range turns {
		isRetry := false
		for _, tool := range t.ToolsUsed {
			if tool == "Bash" || tool == "Fetch" || tool == "Edit" {
				// Check user text for retry indicators
				userLower := toLower(t.UserText)
				if containsAny(userLower, []string{"retry", "try again", "error", "failed", "timeout", "rate limit"}) {
					isRetry = true
					break
				}
			}
		}

		if isRetry {
			retryCount++
			retryTokens += t.InputTokens + t.OutputTokens
			consecutiveErrors++
			if consecutiveErrors > maxConsecutiveErrors {
				maxConsecutiveErrors = consecutiveErrors
			}
		} else {
			consecutiveErrors = 0
		}
	}

	if retryCount >= 2 && retryTokens > 3000 {
		confidence := 0.4 + float64(retryCount)*0.05
		if confidence > 0.8 {
			confidence = 0.8
		}
		return []Finding{{
			Name:            "retry_churn",
			Confidence:      confidence,
			Evidence:        sprintf("%d retry/error turns detected (~%d tokens wasted)", retryCount, retryTokens),
			SavingsTokens:   retryTokens,
			Suggestion: sprintf(
				"Found %d turns with retry/error patterns wasting ~%d tokens. "+
					"Check for: rate limits, timeouts, invalid tool inputs, "+
					"or consider increasing timeout settings.",
				retryCount, retryTokens,
			),
			OccurrenceCount: retryCount,
		}}
	}

	return nil
}

func containsAny(s string, substrs []string) bool {
	for _, sub := range substrs {
		if contains(s, sub) {
			return true
		}
	}
	return false
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && stringInSlice(s, substr)
}

func stringInSlice(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func toLower(s string) string {
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] >= 'A' && s[i] <= 'Z' {
			b[i] = s[i] + 32
		} else {
			b[i] = s[i]
		}
	}
	return string(b)
}
