package frugal

import "sort"

// Registry holds all registered waste detectors.
type Registry struct {
	detectors []Detector
}

// NewRegistry creates a registry with all built-in detectors.
func NewRegistry() *Registry {
	r := &Registry{}
	r.Register(DetectorFunc(detectLooping))
	r.Register(DetectorFunc(detectOutputWaste))
	r.Register(DetectorFunc(detectRetryChurn))
	r.Register(DetectorFunc(detectToolCascade))
	r.Register(DetectorFunc(detectWastefulThinking))
	return r
}

// Register adds a detector to the registry.
func (r *Registry) Register(d Detector) {
	r.detectors = append(r.detectors, d)
}

// RunAll runs all registered detectors and returns sorted findings.
// Findings with confidence <= 0.3 are filtered out.
func (r *Registry) RunAll(data *SessionData) []Finding {
	var findings []Finding
	for _, d := range r.detectors {
		results := d.Detect(data)
		for _, f := range results {
			if f.Confidence > 0.3 {
				findings = append(findings, f)
			}
		}
	}
	sort.Slice(findings, func(i, j int) bool {
		return findings[i].Confidence > findings[j].Confidence
	})
	return findings
}

// Triage returns findings with savings_tokens above minTokens.
func Triage(findings []Finding, minTokens int) []Finding {
	if minTokens <= 0 {
		minTokens = 5000
	}
	var triaged []Finding
	for _, f := range findings {
		if f.SavingsTokens >= minTokens {
			triaged = append(triaged, f)
		}
	}
	return triaged
}

// JaccardSimilarity computes word overlap between two strings.
func JaccardSimilarity(a, b string) float64 {
	wa := wordSet(a)
	wb := wordSet(b)
	if len(wa) == 0 || len(wb) == 0 {
		return 0
	}
	intersection := 0
	for w := range wa {
		if wb[w] {
			intersection++
		}
	}
	union := len(wa) + len(wb) - intersection
	if union == 0 {
		return 0
	}
	return float64(intersection) / float64(union)
}

func wordSet(s string) map[string]bool {
	words := make(map[string]bool)
	var word []byte
	for i := 0; i < len(s); i++ {
		if s[i] == ' ' || s[i] == '\n' || s[i] == '\t' {
			if len(word) > 0 {
				// simple lowercase (ASCII only for word set)
				for j := range word {
					if word[j] >= 'A' && word[j] <= 'Z' {
						word[j] += 32
					}
				}
				words[string(word)] = true
				word = word[:0]
			}
		} else {
			word = append(word, s[i])
		}
	}
	if len(word) > 0 {
		for j := range word {
			if word[j] >= 'A' && word[j] <= 'Z' {
				word[j] += 32
			}
		}
		words[string(word)] = true
	}
	return words
}
