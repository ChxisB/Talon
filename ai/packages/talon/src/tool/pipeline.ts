// ── Pipeline Tool ────────────────────────────────────────────────────────
// Auto-orchestrates complex tasks through fix or feature pipelines with
// quality gates, loop-back, and sign-off.

import { Effect, Schema } from "effect"
import * as Tool from "./tool"
import type { TaskPromptOps } from "./task"
import { runPipeline } from "../agent/pipeline/engine"
import type { PipelineMode } from "../agent/pipeline/types"

// ── Schema ──────────────────────────────────────────────────────────────

const PipelineModeSchema = Schema.Literals(["auto", "fix", "feature"])

export const Parameters = Schema.Struct({
  goal: Schema.String.annotate({
    description: "The goal or task to accomplish. Be specific about what you want done.",
  }),
  mode: Schema.optional(PipelineModeSchema).annotate({
    description:
      'Pipeline mode: "auto" (default, auto-detect), "fix" (bugs/issues), "feature" (new features)',
  }),
})

// ── Tool Definition ─────────────────────────────────────────────────────

const id = "pipeline"

// Build via Tool.define with explicit cast to bypass complex type inference
const initEffect = Effect.gen(function* () {
  const def = {
    description: [
      "Automated pipeline execution for complex tasks. ",
      "Decomposes your goal into stages, delegates to specialist subagents, ",
      "runs quality gates (test + review) after implementation, ",
      "and loops back on failures until resolved or signed off.",
      "",
      "Use this INSTEAD of manually calling task() for each step. ",
      "The pipeline handles agent selection, stage ordering, and verification automatically.",
      "",
      "Modes:",
      '- "auto" (default): Classifies the request as fix or feature',
      '- "fix":        explore → fix → test → review → signoff',
      '- "feature":    research → plan → implement → test → review → signoff',
      "",
      "Example:",
      '  pipeline(goal="Fix the login button not working", mode="fix")',
      '  pipeline(goal="Add dark mode support", mode="feature")',
    ].join("\n"),
    parameters: Parameters,
    execute: (params: Schema.Schema.Type<typeof Parameters>, ctx: Tool.Context) =>
      Effect.gen(function* () {
        const ops = ctx.extra?.promptOps as TaskPromptOps | undefined
        if (!ops) {
          return yield* Effect.fail(new Error("Pipeline tool requires promptOps in ctx.extra"))
        }

        const mode = params.mode === "auto" || !params.mode ? undefined : (params.mode as PipelineMode | undefined)
        const result = yield* runPipeline(params.goal, ops, mode)

        const lines: string[] = [
          `# Pipeline Result: ${result.passed ? "PASSED" : "FAILED"}`,
          ``,
          `**Goal**: ${params.goal}`,
          `**Mode**: ${result.mode}`,
          `**Summary**: ${result.summary}`,
          `**Loopbacks**: ${result.loopbacks}`,
          ``,
        ]

        if (result.stages.length > 0) {
          lines.push(`## Stages`)
          for (const stage of result.stages) {
            const icon = stage.passed ? "PASS" : "FAIL"
            lines.push(`### ${icon} ${stage.stage} (${stage.agent})`)
            lines.push(`  Duration: ${stage.duration}ms`)
            if (stage.error) lines.push(`  Error: ${stage.error}`)
            lines.push("")
          }
        }

        if (result.gates.length > 0) {
          lines.push(`## Quality Gates`)
          for (const gate of result.gates) {
            const icon = gate.passed ? "PASS" : "FAIL"
            lines.push(`### ${icon} ${gate.gate}`)
            if (gate.details) lines.push(`  Details: ${gate.details}`)
            lines.push("")
          }
        }

        lines.push(`---`)
        lines.push(
          result.passed
            ? `Pipeline completed successfully with ${result.loopbacks} loopback(s). Ready for PR.`
            : `Pipeline failed. Review the errors above and try again.`,
        )

        return {
          title: `pipeline: ${params.goal.slice(0, 60)}`,
          metadata: {
            passed: result.passed,
            mode: result.mode,
            stages: result.stages.length,
            gates: result.gates.length,
            loopbacks: result.loopbacks,
            signedOff: result.signedOff,
          },
          output: lines.join("\n").trim(),
        }
      }).pipe(Effect.orDie),
  }
  return def
})

export const PipelineTool: unknown = Tool.define(
  id,
  initEffect as any,
)
