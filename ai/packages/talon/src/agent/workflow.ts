// Multi-step workflow orchestration.
// Decomposes complex tasks into ordered sequences of sub-agent steps.
// Steps run in parallel (no dependency) or sequentially (dependsOn).
// Context flows between steps automatically.
//
// NOTE: The workflow runner below is a stub. For actual multi-step
// orchestration with auto-routing, quality gates, and loop-back,
// use the `pipeline` tool (tool/pipeline.ts) instead.

import { Effect, Context, Layer, Schema } from "effect"
import { Agent } from "./agent"
import type { Info as AgentInfo } from "./agent"

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export const WorkflowStep = Schema.Struct({
  name: Schema.String,
  agent: Schema.String,
  task: Schema.String,
  dependsOn: Schema.optional(Schema.Array(Schema.String)),
})
export type WorkflowStep = Schema.Schema.Type<typeof WorkflowStep>

export const WorkflowPlan = Schema.Struct({
  name: Schema.String,
  description: Schema.optional(Schema.String),
  steps: Schema.Array(WorkflowStep),
})
export type WorkflowPlan = Schema.Schema.Type<typeof WorkflowPlan>

export interface WorkflowResult {
  step: string
  agent: string
  output: string
  error?: string
  duration: number
}

// ---------------------------------------------------------------------------
// Topological sort — respects dependsOn edges
// ---------------------------------------------------------------------------

function sortSteps(steps: readonly WorkflowStep[]): WorkflowStep[][] {
  const byName = new Map(steps.map((s) => [s.name, s]))
  const deps = new Map(steps.map((s) => [s.name, new Set(s.dependsOn ?? [])]))
  const batches: WorkflowStep[][] = []

  let remaining = new Set(steps.map((s) => s.name))

  while (remaining.size > 0) {
    const batch: WorkflowStep[] = []

    for (const name of remaining) {
      const stepDeps = deps.get(name)
      if (!stepDeps || ![...stepDeps].some((dep) => remaining.has(dep))) {
        const step = byName.get(name)
        if (step) batch.push(step)
      }
    }

    if (batch.length === 0) break

    batches.push(batch)
    for (const step of batch) {
      remaining.delete(step.name)
    }
  }

  return batches
}

// ---------------------------------------------------------------------------
// Service interface
// ---------------------------------------------------------------------------

export interface Interface {
  readonly run: (plan: WorkflowPlan) => Effect.Effect<WorkflowResult[]>
  readonly sort: (steps: readonly WorkflowStep[]) => WorkflowStep[][]
}

export class Service extends Context.Service<Service, Interface>()("@talon/Workflow") {}

// ---------------------------------------------------------------------------
// Layer
// ---------------------------------------------------------------------------

export const layer = Layer.effect(
  Service,
  Effect.gen(function* () {
    const agentSvc = yield* Agent.Service

    const run: Interface["run"] = Effect.fn("Workflow.run")(function* (plan: WorkflowPlan) {
      const results: WorkflowResult[] = []
      const outputs = new Map<string, string>()
      const agents = new Map<string, AgentInfo>()

      for (const step of plan.steps) {
        if (!agents.has(step.agent)) {
          agents.set(step.agent, yield* agentSvc.get(step.agent))
        }
      }

      const batches = sortSteps(plan.steps)

      for (const batch of batches) {
        const batchResults = yield* Effect.forEach(
          batch,
          Effect.fnUntraced(function* (step: WorkflowStep) {
            const start = Date.now()

            // Build context from dependency outputs
            const context = (step.dependsOn ?? [])
              .map((dep) => {
                const out = outputs.get(dep)
                return out ? `## Output from ${dep}\n\n${out}` : ""
              })
              .filter(Boolean)

            const fullTask = context.length > 0
              ? `${step.task}\n\nContext from upstream steps:\n\n${context.join("\n\n")}`
              : step.task

            try {
              outputs.set(step.name, fullTask)
              const outcome: WorkflowResult = {
                step: step.name,
                agent: step.agent,
                output: fullTask,
                duration: Date.now() - start,
              }
              return outcome
            } catch (e) {
              const outcome: WorkflowResult = {
                step: step.name,
                agent: step.agent,
                output: "",
                error: String(e),
                duration: Date.now() - start,
              }
              return outcome
            }
          }),
          { concurrency: "unbounded" },
        )

        results.push(...batchResults)
      }

      return results
    })

    const sort: Interface["sort"] = (steps) => sortSteps(steps)

    return Service.of({ run, sort })
  }),
)

export const defaultLayer = layer.pipe(
  Layer.provide(Agent.defaultLayer),
)

export * as Workflow from "./workflow"
