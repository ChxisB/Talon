// ── Quality Gates ────────────────────────────────────────────────────────
// Gates check implementation quality between pipeline stages.

import type { Gate, GateResult, StageType } from "./types"
import { stageToAgent, stagePrompt, stageDescription } from "../router"

/**
 * Build the quality gates for a given mode and stage.
 * Returns gates that should run after the given stage completes.
 */
export function gatesForStage(stage: StageType, mode: "fix" | "feature", goal: string): Gate[] {
  const gates: Gate[] = []

  switch (stage) {
    case "fix":
    case "implement":
      gates.push({
        name: "test",
        agent: stageToAgent("test", mode),
        task: stagePrompt("test", goal),
      })
      gates.push({
        name: "review",
        agent: stageToAgent("review", mode),
        task: stagePrompt("review", goal),
      })
      break
  }

  return gates
}

/**
 * Parse a gate result to determine if it passed.
 * Looks for keywords in the gate output.
 */
export function evaluateGateResult(gate: Gate, output: string): GateResult {
  const lower = output.toLowerCase()

  let passed: boolean
  let details: string | undefined

  switch (gate.name) {
    case "test": {
      // Tests pass if no failure indicators found
      const hasFailure = /\b(fail|failed|failure|error)\b/i.test(output)
      const hasPass = /\b(pass|passed|success|ok)\b/i.test(output)
      passed = !hasFailure || (hasPass && !hasFailure)
      if (!passed) {
        // Extract specific failure messages
        const failures = output.match(/(?:\d+)\s*(?:fail|error)(?:ed|s)?/gi)
        details = failures?.join(", ") ?? "Tests failed"
      }
      break
    }
    case "review": {
      // Review passes if output contains APPROVED
      passed = /\bapproved\b/i.test(output)
      if (!passed) {
        if (/\brejected\b/i.test(output)) {
          details = "Review rejected"
        } else if (/\bchanges_requested\b/i.test(output)) {
          details = "Changes requested"
        } else {
          details = "Review did not explicitly approve"
        }
      }
      break
    }
    default:
      passed = true
  }

  return {
    gate: gate.name,
    passed,
    output,
    details,
  }
}
