package frugal

const (
	outputRatioThreshold     = 3.0
	verboseResponseTokens    = 2000
	similarityThreshold      = 0.6
	minTurnsForOutputWaste   = 5
	minSavingsTokensOutput   = 5000
)

// detectOutputWaste detects sessions with excessive output relative to task complexity.
func detectOutputWaste(data *SessionData) []Finding {
	turns := data.Turns
	if len(turns) < minTurnsForOutputWaste {
		return nil
	}

	totalOutput := data.TotalOutputTokens
	if totalOutput == 0 {
		return nil
	}

	var findings []Finding

	// Signal 1: High output/input ratio on simple turns
	simpleOutput := 0
	simpleInput := 0
	simpleCount := 0
	for _, t := range turns {
		if isSimpleToolTurn(t) {
			simpleOutput += t.OutputTokens
			simpleInput += max(1, t.InputTokens)
			simpleCount++
		}
	}

	if simpleCount >= 3 && simpleInput > 0 {
		ratio := float64(simpleOutput) / float64(simpleInput)
		if ratio > outputRatioThreshold {
			excess := simpleOutput - int(float64(simpleInput)*1.5)
			if excess > minSavingsTokensOutput {
				confidence := 0.5 + (ratio-outputRatioThreshold)*0.1
				if confidence > 0.85 {
					confidence = 0.85
				}
				findings = append(findings, Finding{
					Name:      "output_waste",
					Confidence: confidence,
					Evidence: sprintf(
						"%.1fx output/input ratio across %d simple turns (%d output vs %d input tokens)",
						ratio, simpleCount, simpleOutput, simpleInput,
					),
					SavingsTokens:   excess,
					Suggestion: sprintf(
						"Output tokens are %.1fx higher than input on simple file operations. "+
							"This session could save ~%d output tokens by requesting concise responses. "+
							"Add 'Be concise' or 'No explanations' for routine tasks.",
						ratio, excess,
					),
					OccurrenceCount: simpleCount,
				})
			}
		}
	}

	// Signal 2: Verbose responses after simple operations
	verboseCount := 0
	verboseWaste := 0
	for _, t := range turns {
		if isSimpleToolTurn(t) && t.OutputTokens > verboseResponseTokens {
			verboseCount++
			verboseWaste += t.OutputTokens - verboseResponseTokens
		}
	}
	if verboseCount >= 3 && verboseWaste > minSavingsTokensOutput {
		findings = append(findings, Finding{
			Name:      "output_waste",
			Confidence: 0.65,
			Evidence: sprintf(
				"%d turns had >2K output tokens after simple file operations (~%d excess output tokens)",
				verboseCount, verboseWaste,
			),
			SavingsTokens:   verboseWaste,
			Suggestion: sprintf(
				"Found %d cases of verbose responses to simple edits/reads. "+
					"~%d output tokens could be saved with more targeted instructions.",
				verboseCount, verboseWaste,
			),
			OccurrenceCount: verboseCount,
		})
	}

	// Signal 3: Repeated similar explanations
	assistantTexts := make([]string, 0, len(turns))
	for _, t := range turns {
		if len(t.AssistantText) > 200 {
			assistantTexts = append(assistantTexts, t.AssistantText)
			if len(assistantTexts) >= 100 {
				break
			}
		}
	}

	repeatedPairs := 0
	repeatedWaste := 0
	for i := 0; i < len(assistantTexts); i++ {
		maxJ := i + 5
		if maxJ > len(assistantTexts) {
			maxJ = len(assistantTexts)
		}
		for j := i + 1; j < maxJ; j++ {
			if JaccardSimilarity(assistantTexts[i], assistantTexts[j]) > similarityThreshold {
				repeatedPairs++
				repeatedWaste += len(assistantTexts[j]) / 4
			}
		}
	}

	if repeatedPairs >= 2 && repeatedWaste > minSavingsTokensOutput {
		findings = append(findings, Finding{
			Name:      "output_waste",
			Confidence: 0.55,
			Evidence: sprintf(
				"%d pairs of similar assistant responses detected (>%.0f%% word overlap, ~%d repeated tokens)",
				repeatedPairs, similarityThreshold*100, repeatedWaste,
			),
			SavingsTokens:   repeatedWaste,
			Suggestion: sprintf(
				"The model repeated similar explanations %d times. "+
					"~%d tokens could be saved. "+
					"Consider adding 'Don't repeat previous explanations' to instructions.",
				repeatedPairs, repeatedWaste,
			),
			OccurrenceCount: repeatedPairs,
		})
	}

	return findings
}

func isSimpleToolTurn(t TurnData) bool {
	if len(t.ToolsUsed) == 0 {
		return false
	}
	simple := map[string]bool{
		"Read": true, "Glob": true, "Grep": true, "Edit": true, "Write": true,
		"View": true, "LS": true, "List": true, "Bash": true,
	}
	for _, tool := range t.ToolsUsed {
		if !simple[tool] {
			return false
		}
	}
	return len(t.ToolsUsed) <= 2
}
