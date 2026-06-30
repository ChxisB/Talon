// ── Agent Router ─────────────────────────────────────────────────────────
// Classifies user requests into pipeline modes and selects appropriate agents.

import type { PipelineMode, IntentClassification, StageType } from "./pipeline/types"

/**
 * Intent keywords that signal a "fix" mode request.
 */
const FIX_KEYWORDS = [
  "fix", "bug", "broken", "error", "issue", "crash", "failing",
  "doesn't work", "not working", "repair", "problem", "wrong",
  "incorrect", "corrupt", "corrupted", "defect", "defective",
  "regression", "glitch", "malfunction", "typo", "mistake",
]

/**
 * Intent keywords that signal a "feature" mode request.
 */
const FEATURE_KEYWORDS = [
  "add", "feature", "implement", "build", "create", "new",
  "design", "architecture", "plan", "research", "explore",
  "investigate", "support", "integrate", "migrate", "refactor",
  "redesign", "restructure", "upgrade", "enhance",
]

/**
 * Classify a user's request into a pipeline mode.
 * Uses keyword matching with confidence scoring.
 */
export function classifyIntent(request: string): IntentClassification {
  const lower = request.toLowerCase()
  let fixScore = 0
  let featureScore = 0
  const findings: string[] = []

  for (const kw of FIX_KEYWORDS) {
    if (lower.includes(kw)) {
      fixScore++
      if (fixScore <= 3) findings.push(`fix keyword: "${kw}"`)
    }
  }

  for (const kw of FEATURE_KEYWORDS) {
    if (lower.includes(kw)) {
      featureScore++
      if (featureScore <= 3) findings.push(`feature keyword: "${kw}"`)
    }
  }

  // Determine mode
  let mode: PipelineMode
  let confidence: number
  let summary: string

  if (fixScore > featureScore) {
    mode = "fix"
    confidence = Math.min(0.5 + (fixScore - featureScore) * 0.15, 0.95)
    summary = `Classified as fix (score: ${fixScore} fix vs ${featureScore} feature)`
  } else if (featureScore > fixScore) {
    mode = "feature"
    confidence = Math.min(0.5 + (featureScore - fixScore) * 0.15, 0.95)
    summary = `Classified as feature (score: ${featureScore} feature vs ${fixScore} fix)`
  } else {
    // Tie or no keywords — default to feature for safety
    mode = "feature"
    confidence = 0.5
    summary = "No clear intent detected, defaulting to feature pipeline"
  }

  return { mode, confidence, summary, keyFindings: findings }
}

/**
 * Map a pipeline stage to the correct subagent type.
 */
export function stageToAgent(stage: StageType, mode: PipelineMode): string {
  switch (stage) {
    case "explore":
      return "explore"
    case "research":
      return "librarian"
    case "plan":
      return "planner"
    case "implement":
    case "fix":
      return mode === "fix" ? "general" : "general"
    case "test":
      return "general"
    case "review":
      return "reviewer"
    case "signoff":
      return "general"
  }
}

/**
 * Generate the task prompt for a pipeline stage.
 */
export function stagePrompt(
  stage: StageType,
  goal: string,
  context?: string,
): string {
  const contextBlock = context
    ? `\n\nContext from previous stages:\n\n${context}`
    : ""

  switch (stage) {
    case "explore":
      return [
        `You are a codebase explorer. Your task is to investigate and understand this issue:`,
        ``,
        `Goal: ${goal}`,
        ``,
        `Search the codebase to find:`,
        `1. Where the problem is occurring (file paths, line numbers)`,
        `2. What's causing the issue (root cause analysis)`,
        `3. Any related code that might be affected`,
        `4. Existing tests related to this area`,
        ``,
        `Return a detailed report with file paths, line numbers, and your analysis.`,
        contextBlock,
      ].join("\n")

    case "research":
      return [
        `You are a codebase researcher. Your task is to investigate and understand the codebase for this goal:`,
        ``,
        `Goal: ${goal}`,
        ``,
        `Research the codebase thoroughly:`,
        `1. Find relevant files, patterns, and existing implementations`,
        `2. Understand the architecture and data flow`,
        `3. Identify key modules, functions, and their relationships`,
        `4. Find existing tests and their coverage`,
        `5. Look for similar features already implemented`,
        ``,
        `Return a comprehensive research report with file paths, patterns found, and recommendations.`,
        contextBlock,
      ].join("\n")

    case "plan":
      return [
        `You are a technical planner. Your task is to create a detailed implementation plan for:`,
        ``,
        `Goal: ${goal}`,
        ``,
        `Create a plan that includes:`,
        `1. Architecture/design decisions`,
        `2. Files to create or modify`,
        `3. Step-by-step implementation order`,
        `4. Dependencies between steps`,
        `5. Testing strategy`,
        `6. Potential risks and mitigations`,
        ``,
        `Return a structured plan.`,
        contextBlock,
      ].join("\n")

    case "implement":
      return [
        `You are a senior engineer. Implement the solution for:`,
        ``,
        `Goal: ${goal}`,
        ``,
        `Follow these requirements:`,
        `1. Write production-quality code`,
        `2. Follow existing code patterns and conventions`,
        `3. Write tests for your changes`,
        `4. Ensure the build compiles`,
        `5. Handle edge cases and errors`,
        ``,
        `DO NOT use 'any' type.`,
        `DO write failing tests first (RED → GREEN).`,
        `ALWAYS verify your work (build, test, lint).`,
        contextBlock,
      ].join("\n")

    case "fix":
      return [
        `You are a senior engineer. Fix the issue described below:`,
        ``,
        `Goal: ${goal}`,
        ``,
        `Requirements:`,
        `1. Understand the root cause before fixing`,
        `2. Make minimal, targeted changes`,
        `3. Don't break existing tests`,
        `4. Add tests for the fix`,
        `5. Verify the build compiles`,
        ``,
        `DO NOT use 'any' type.`,
        `ALWAYS verify your work (build, test).`,
        contextBlock,
      ].join("\n")

    case "test":
      return [
        `You are a test engineer. Your task is to verify the implementation for:`,
        ``,
        `Goal: ${goal}`,
        ``,
        `Run tests and verify:`,
        `1. All existing tests still pass`,
        `2. New tests pass (if any were added)`,
        `3. The build compiles without errors`,
        `4. Type checking passes`,
        ``,
        `If tests fail, report which tests failed and why.`,
        contextBlock,
      ].join("\n")

    case "review":
      return [
        `You are a code reviewer. Your task is to review the implementation for:`,
        ``,
        `Goal: ${goal}`,
        ``,
        `Review for:`,
        `1. Correctness — does it solve the problem?`,
        `2. Security — any vulnerabilities?`,
        `3. Performance — any obvious issues?`,
        `4. Code quality — follows project patterns?`,
        `5. Test coverage — adequate tests?`,
        `6. Edge cases — all handled?`,
        ``,
        `For each issue found, provide: file path, line number, description, and severity.`,
        `Rate the overall quality as: APPROVED, CHANGES_REQUESTED, or REJECTED.`,
        contextBlock,
      ].join("\n")

    case "signoff":
      return [
        `You are a release manager. Review the pipeline results for:`,
        ``,
        `Goal: ${goal}`,
        ``,
        `Check:`,
        `1. All stages completed successfully`,
        `2. All quality gates passed`,
        `3. Tests are passing`,
        `4. Code review approved`,
        ``,
        `Decide: SIGNED_OFF or REJECTED with reasons.`,
        contextBlock,
      ].join("\n")
  }
}

/**
 * Build the task description (short summary) for a pipeline stage.
 */
export function stageDescription(stage: StageType, goal: string): string {
  switch (stage) {
    case "explore":
      return `Explore: ${goal.slice(0, 60)}`
    case "research":
      return `Research: ${goal.slice(0, 60)}`
    case "plan":
      return `Plan: ${goal.slice(0, 60)}`
    case "implement":
      return `Implement: ${goal.slice(0, 60)}`
    case "fix":
      return `Fix: ${goal.slice(0, 60)}`
    case "test":
      return `Test: ${goal.slice(0, 60)}`
    case "review":
      return `Review: ${goal.slice(0, 60)}`
    case "signoff":
      return `Sign-off: ${goal.slice(0, 60)}`
  }
}
