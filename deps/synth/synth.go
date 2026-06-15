// Package synth provides coding principles for LLM assistants and a prompt
// template engine that injects them into agent prompts automatically.
package synth

import (
	_ "embed"
	"strings"
)

//go:embed principles/think-before-coding.md
var principleThinking string

//go:embed principles/simplicity-first.md
var principleSimplicity string

//go:embed principles/surgical-changes.md
var principleSurgical string

//go:embed principles/goal-driven.md
var principleGoalDriven string

//go:embed principles/skill-header.md
var skillHeader string

// Principle represents one of the four coding guidelines.
type Principle struct {
	ID      string
	Title   string
	Content string
}

// AllPrinciples returns the four coding principles in order.
func AllPrinciples() []Principle {
	return []Principle{
		{ID: "thinking", Title: "Think Before Coding", Content: strings.TrimSpace(principleThinking)},
		{ID: "simplicity", Title: "Simplicity First", Content: strings.TrimSpace(principleSimplicity)},
		{ID: "surgical", Title: "Surgical Changes", Content: strings.TrimSpace(principleSurgical)},
		{ID: "goal", Title: "Goal-Driven Execution", Content: strings.TrimSpace(principleGoalDriven)},
	}
}

// Inject takes a base system prompt and appends the coding principles
// as an additional context block. The principles are formatted as markdown
// sections with clear separators.
func Inject(basePrompt string) string {
	var b strings.Builder
	b.WriteString(basePrompt)
	b.WriteString("\n\n")
	b.WriteString(strings.TrimSpace(skillHeader))
	b.WriteString("\n\n")

	for _, p := range AllPrinciples() {
		b.WriteString("## ")
		b.WriteString(p.Title)
		b.WriteString("\n\n")
		b.WriteString(p.Content)
		b.WriteString("\n\n")
	}

	return b.String()
}

// Compact returns a single-paragraph summary of all four principles,
// suitable for injection into prompts where context length is tight.
func Compact() string {
	return `You are governed by four principles: (1) Think Before Coding — surface assumptions, 
ask clarifying questions, reason through tradeoffs before writing code. (2) Simplicity First — 
write the minimum code that solves the problem, no speculative features. (3) Surgical Changes — 
touch only what is requested, match existing style exactly. (4) Goal-Driven Execution — define 
success criteria upfront, verify with tests before declaring done.`
}

// ShouldActivate returns true if the user message contains signals that
// the coding principles should be activated (e.g., "think before coding",
// "best practices").
func ShouldActivate(msg string) bool {
	lower := strings.ToLower(msg)
	triggers := []string{
		"think before coding", "simplicity first",
		"surgical changes", "goal driven", "best practices",
		"coding principles", "code guidelines",
	}
	for _, t := range triggers {
		if strings.Contains(lower, t) {
			return true
		}
	}
	return false
}
