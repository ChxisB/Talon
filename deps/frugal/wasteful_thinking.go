package frugal

// detectWastefulThinking detects sessions where the model spends excessive
// tokens on thinking/reasoning for simple tasks.
func detectWastefulThinking(data *SessionData) []Finding {
	turns := data.Turns
	if len(turns) < 3 {
		return nil
	}

	thinkingTurns := 0
	thinkingTokens := 0

	for _, t := range turns {
		// Check if this looks like a thinking-heavy turn:
		// high output relative to input for simple tools
		if len(t.ToolsUsed) <= 1 && t.OutputTokens > t.InputTokens*4 && t.OutputTokens > 2000 {
			thinkingTurns++
			thinkingTokens += t.OutputTokens - t.InputTokens*2
		}
	}

	if thinkingTurns >= 2 && thinkingTokens > 4000 {
		confidence := 0.45 + float64(thinkingTurns)*0.05
		if confidence > 0.75 {
			confidence = 0.75
		}
		return []Finding{{
			Name:            "wasteful_thinking",
			Confidence:      confidence,
			Evidence:        sprintf("%d turns with excessive thinking output (~%d tokens)", thinkingTurns, thinkingTokens),
			SavingsTokens:   thinkingTokens,
			Suggestion: sprintf(
				"Found %d turns where output tokens far exceeded input for simple operations. "+
					"Consider reducing thinking budget or using a smaller model for routine tasks.",
				thinkingTurns,
			),
			OccurrenceCount: thinkingTurns,
		}}
	}

	return nil
}
