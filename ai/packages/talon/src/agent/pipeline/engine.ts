// ── Pipeline Engine ─────────────────────────────────────────────────────
// Orchestrates pipeline stages, quality gates, and loop-back.

import { Effect, Exit } from "effect"
import { Agent } from "../agent"
import { SessionID, MessageID } from "../../session/schema"
import { Session } from "../../session/session"
import type { TaskPromptOps } from "../../tool/task"
import {
  classifyIntent,
  stageToAgent,
  stagePrompt,
  stageDescription,
} from "../router"
import { gatesForStage, evaluateGateResult } from "./gates"
import type {
  PipelineMode,
  StageType,
  Stage,
  StageResult,
  GateResult,
  PipelineResult,
  PipelineConfig,
} from "./types"
import { defaultPipelineConfig } from "./types"

// ── Stage Plans ─────────────────────────────────────────────────────────

function stagesForMode(mode: PipelineMode, goal: string): Stage[] {
  if (mode === "fix") {
    return [
      {
        name: "explore",
        agent: stageToAgent("explore", mode),
        task: stagePrompt("explore", goal),
        sequential: true,
      },
      {
        name: "fix",
        agent: stageToAgent("fix", mode),
        task: stagePrompt("fix", goal),
        sequential: true,
      },
      {
        name: "signoff",
        agent: stageToAgent("signoff", mode),
        task: stagePrompt("signoff", goal),
        sequential: true,
      },
    ]
  }

  return [
    {
      name: "research",
      agent: stageToAgent("research", mode),
      task: stagePrompt("research", goal),
      sequential: true,
    },
    {
      name: "plan",
      agent: stageToAgent("plan", mode),
      task: stagePrompt("plan", goal),
      sequential: true,
    },
    {
      name: "implement",
      agent: stageToAgent("implement", mode),
      task: stagePrompt("implement", goal),
      sequential: true,
    },
    {
      name: "signoff",
      agent: stageToAgent("signoff", mode),
      task: stagePrompt("signoff", goal),
      sequential: true,
    },
  ]
}

// ── Stage Execution ─────────────────────────────────────────────────────

/**
 * Execute a single pipeline stage via subagent session.
 * Requires Agent.Service and Session.Service in the Effect context.
 */
const executeStage = (stage: Stage, goal: string, ops: TaskPromptOps, contextOutput?: string) =>
  Effect.gen(function* () {
  const agentSvc = yield* Agent.Service
    const sessions = yield* Session.Service
    const start = Date.now()

    const agentInfo = yield* agentSvc.get(stage.agent)
    if (!agentInfo) {
      return {
        stage: stage.name,
        agent: stage.agent,
        output: "",
        error: `Unknown agent: ${stage.agent}`,
        duration: 0,
        passed: false,
      } as StageResult
    }

    const prompt = contextOutput
      ? `${stage.task}\n\n## Context from previous stage\n\n${contextOutput}`
      : stage.task

    const description = stageDescription(stage.name, goal)

    const childSession = yield* sessions.create({
      title: `${description} (@${agentInfo.name} subagent)`,
      agent: agentInfo.name,
    })

    const parts = yield* ops.resolvePromptParts(prompt)

    const exit = yield* Effect.exit(
      ops.prompt({
        messageID: MessageID.ascending(),
        sessionID: childSession.id,
        agent: agentInfo.name,
        parts,
      }),
    )

    const duration = Date.now() - start

    if (Exit.isFailure(exit)) {
      const causeStr = String(exit.cause)
      return {
        stage: stage.name,
        agent: stage.agent,
        output: "",
        error: causeStr,
        duration,
        passed: false,
      } as StageResult
    }

    const success = exit.value
    const lastTextPart = [...(success.parts ?? [])].reverse().find(
      (p: { type: string; text?: string }) => p.type === "text",
    )
    const text = lastTextPart && "text" in lastTextPart ? (lastTextPart as { text: string }).text : ""

    return {
      stage: stage.name,
      agent: stage.agent,
      output: text,
      duration,
      passed: true,
    } as StageResult
  })

// ── Pipeline Runner ─────────────────────────────────────────────────────

/**
 * Run the full pipeline for a given goal and mode.
 * Handles stage execution, quality gates, and loop-back.
 */
const runPipelineImpl = (
  goal: string,
  ops: TaskPromptOps,
  mode?: PipelineMode,
  config: PipelineConfig = defaultPipelineConfig,
) =>
  Effect.gen(function* () {
  const classification = mode
      ? {
          mode,
          confidence: 1,
          summary: "Explicitly specified",
          keyFindings: [] as string[],
        }
      : classifyIntent(goal)

    const pipelineMode = classification.mode
    const stages = stagesForMode(pipelineMode, goal)

    const stageResults: StageResult[] = []
    const gateResults: GateResult[] = []
    let loopbacks = 0
    let contextOutput: string | undefined

    for (let i = 0; i < stages.length; i++) {
      const stage = stages[i]

      const result = yield* executeStage(stage, goal, ops, contextOutput)
      stageResults.push(result)

      if (!result.passed) {
        return {
          goal,
          mode: pipelineMode,
          stages: stageResults,
          gates: gateResults,
          loopbacks,
          signedOff: false,
          passed: false,
          summary: `Pipeline failed at stage "${stage.name}": ${result.error ?? "Unknown error"}`,
        } as PipelineResult
      }

      contextOutput = result.output

      // Run quality gates after fix/implement stages
      if (stage.name === "fix" || stage.name === "implement") {
        const gates = gatesForStage(stage.name, pipelineMode, goal)

        for (const gate of gates) {
          const gateStage: Stage = {
            name: gate.name as StageType,
            agent: gate.agent,
            task: gate.task,
            sequential: true,
          }

          const gateResult = yield* executeStage(gateStage, goal, ops, contextOutput)
          const evaluated = evaluateGateResult(gate, gateResult.output)
          gateResults.push(evaluated)

          if (!evaluated.passed) {
            loopbacks++
            if (loopbacks >= config.maxLoopbacks) {
              return {
                goal,
                mode: pipelineMode,
                stages: stageResults,
                gates: gateResults,
                loopbacks,
                signedOff: false,
                passed: false,
                summary: `Pipeline failed: ${evaluated.gate} gate failed after ${loopbacks} retries. ${evaluated.details ?? ""}`,
              } as PipelineResult
            }

            // Loop back: re-run stage with gate feedback
            contextOutput = [
              `Previous attempt output:\n${result.output}`,
              ``,
              `${evaluated.gate} gate: FAILED`,
              evaluated.details ? `Details: ${evaluated.details}` : "",
              ``,
              `Please fix the issues identified above.`,
            ].join("\n")

            const retryResult = yield* executeStage(stage, goal, ops, contextOutput)
            stageResults.push(retryResult)

            if (!retryResult.passed) {
              return {
                goal,
                mode: pipelineMode,
                stages: stageResults,
                gates: gateResults,
                loopbacks,
                signedOff: false,
                passed: false,
                summary: `Pipeline failed: retry of "${stage.name}" also failed.`,
              } as PipelineResult
            }

            contextOutput = retryResult.output

            // Re-run the gate
            const retryGateResult = yield* executeStage(
              gateStage,
              goal,
              ops,
              contextOutput,
            )
            const retryEvaluated = evaluateGateResult(gate, retryGateResult.output)
            gateResults.push(retryEvaluated)

            if (!retryEvaluated.passed) {
              return {
                goal,
                mode: pipelineMode,
                stages: stageResults,
                gates: gateResults,
                loopbacks,
                signedOff: false,
                passed: false,
                summary: `Pipeline failed: ${evaluated.gate} gate still failing after retry. ${retryEvaluated.details ?? ""}`,
              } as PipelineResult
            }
          }
        }
      }
    }

    const signedOff =
      stageResults.every((s) => s.passed) && gateResults.every((g) => g.passed)

    const summary = signedOff
      ? `Pipeline completed successfully after ${stageResults.length} stages and ${gateResults.length} gates with ${loopbacks} loopback(s).`
      : `Pipeline completed but sign-off failed.`

    return {
      goal,
      mode: pipelineMode,
      stages: stageResults,
      gates: gateResults,
      loopbacks,
      signedOff,
      passed: signedOff,
      summary,
    } as PipelineResult
})

export const runPipeline = runPipelineImpl
