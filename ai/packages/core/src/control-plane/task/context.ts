export * as TaskContext from "./context"

import { Effect, Layer, Schema } from "effect"
import { SystemContext } from "../../system-context/index"
import { SystemContextRegistry } from "../../system-context/registry"
import { Service as TaskService, defaultLayer as taskServiceDefaultLayer } from "./index"

const TaskSummary = Schema.Struct({
  id: Schema.String,
  title: Schema.String,
  status: Schema.String,
  assignee: Schema.optional(Schema.String),
})
type TaskSummary = typeof TaskSummary.Type

const Summaries = Schema.Array(TaskSummary)
const key = SystemContext.Key.make("talon/tasks")

export const layer = Layer.effectDiscard(
  Effect.gen(function* () {
    const service = yield* TaskService
    const registry = yield* SystemContextRegistry.Service

    const source = (value: ReadonlyArray<TaskSummary> | SystemContext.Unavailable) =>
      SystemContext.make({
        key,
        codec: Schema.toCodecJson(Summaries),
        load: Effect.succeed(value),
        baseline: renderBaseline,
        update: (_previous, current) =>
          `Tasks have been updated.\n\n${renderBaseline(current)}`,
        removed: () => "Previously loaded tasks no longer apply.",
      })

    const observe = Effect.fn("TaskContext.observe")(function* () {
      const tasks = yield* service.list()
      if (tasks.length === 0) return [] as TaskSummary[]
      return tasks.map((t) => ({
        id: t.id,
        title: t.title,
        status: t.status,
        assignee: t.assignee ?? undefined,
      })) as TaskSummary[]
    })

    yield* registry.register({
      key,
      load: observe().pipe(
        Effect.map((summaries) =>
          summaries.length === 0 ? SystemContext.empty : source(summaries),
        ),
        Effect.catch(() => Effect.succeed(source(SystemContext.unavailable))),
        Effect.catchDefect(() => Effect.succeed(source(SystemContext.unavailable))),
      ),
    })
  }),
)

function renderBaseline(summaries: ReadonlyArray<TaskSummary>): string {
  if (summaries.length === 0) return "No active tasks."
  const lines: string[] = ["Active tasks for this project:"]
  for (const s of summaries) {
    const assignee = s.assignee ? ` @${s.assignee}` : ""
    lines.push(`  [${s.status}] ${s.title} — ${s.id}${assignee}`)
  }
  return lines.join("\n")
}

export const withTaskService = layer.pipe(
  Layer.provideMerge(taskServiceDefaultLayer),
)
