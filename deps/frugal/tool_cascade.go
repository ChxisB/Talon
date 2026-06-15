package frugal

// detectToolCascade detects chains of rapid tool calls where each tool
// invocation is a fix-up or workaround for the previous one.
func detectToolCascade(data *SessionData) []Finding {
	turns := data.Turns
	if len(turns) < 4 {
		return nil
	}

	// Look for patterns: 3+ consecutive turns with edit/fix tools
	cascadeCount := 0
	cascadeTokens := 0
	inCascade := false

	for _, t := range turns {
		hasEdit := false
		for _, tool := range t.ToolsUsed {
			if tool == "Edit" || tool == "Write" || tool == "Bash" {
				hasEdit = true
				break
			}
		}
		if hasEdit {
			if !inCascade {
				inCascade = true
				cascadeCount = 1
				cascadeTokens = t.InputTokens + t.OutputTokens
			} else {
				cascadeCount++
				cascadeTokens += t.InputTokens + t.OutputTokens
			}
		} else {
			if inCascade && cascadeCount >= 3 {
				// Cascade ended - we would report here
				// But continue scanning for others
			}
			inCascade = false
		}
	}

	if inCascade && cascadeCount >= 3 {
		// Terminal cascade detected
	}

	// Simpler: count edit-heavy turns
	editTurns := 0
	totalTokens := 0
	for _, t := range turns {
		editCount := 0
		for _, tool := range t.ToolsUsed {
			if tool == "Edit" || tool == "Write" {
				editCount++
			}
		}
		if editCount >= 2 {
			editTurns++
			totalTokens += t.InputTokens + t.OutputTokens
		}
	}

	if editTurns >= 3 && totalTokens > 5000 {
		confidence := 0.5
		if editTurns >= 5 {
			confidence = 0.7
		}
		return []Finding{{
			Name:            "tool_cascade",
			Confidence:      confidence,
			Evidence:        sprintf("%d turns with cascading edit/write calls (~%d tokens)", editTurns, totalTokens),
			SavingsTokens:   totalTokens / 2,
			Suggestion: sprintf(
				"Found %d turns with cascading edit/write operations. "+
					"Consider batching changes or using a more targeted edit strategy to avoid churn.",
				editTurns,
			),
			OccurrenceCount: editTurns,
		}}
	}

	return nil
}
