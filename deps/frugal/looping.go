package frugal

// detectLooping detects sessions where consecutive user messages have high
// word overlap, indicating the model is stuck in a loop.
func detectLooping(data *SessionData) []Finding {
	turns := data.Turns
	if len(turns) < 4 {
		return nil
	}

	userTexts := make([]string, 0, len(turns))
	for _, t := range turns {
		if len(t.UserText) > 10 {
			text := t.UserText
			if len(text) > 500 {
				text = text[:500]
			}
			userTexts = append(userTexts, text)
		}
	}

	if len(userTexts) < 4 {
		return nil
	}

	wordSets := make([]map[string]bool, len(userTexts))
	for i, t := range userTexts {
		wordSets[i] = wordSet(t)
	}

	streak := 1
	maxStreak := 1
	for i := 1; i < len(wordSets); i++ {
		sim := jaccardSets(wordSets[i], wordSets[i-1])
		if sim > 0.75 {
			streak++
			if streak > maxStreak {
				maxStreak = streak
			}
		} else {
			streak = 1
		}
	}

	if maxStreak >= 4 {
		estTokens := maxStreak * 5000
		return []Finding{{
			Name:            "looping",
			Confidence:      0.6,
			Evidence:        sprintf("%d similar consecutive user messages detected", maxStreak),
			SavingsTokens:   estTokens,
			Suggestion: sprintf(
				"You sent %d similar messages in a row, suggesting the model was stuck. "+
					"Try: restate the problem differently, provide a concrete example, or start fresh.",
				maxStreak,
			),
			OccurrenceCount: maxStreak,
		}}
	}

	return nil
}

func jaccardSets(a, b map[string]bool) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	intersection := 0
	for w := range a {
		if b[w] {
			intersection++
		}
	}
	union := len(a) + len(b) - intersection
	if union == 0 {
		return 0
	}
	return float64(intersection) / float64(union)
}
