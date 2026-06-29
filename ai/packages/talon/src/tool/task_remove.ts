import { Effect, Schema } from "effect"
import * as Tool from "./tool"
import { getTaskService } from "./task-helper"
import DESCRIPTION from "./task-remove.txt"

export const Parameters = Schema.Struct({
  id: Schema.String.annotate({ description: "The task ID to remove (starts with \"tsk_\")" }),
})

type Metadata = {
  id: string
  removed: boolean
}

export const TaskRemoveTool = Tool.define<typeof Parameters, Metadata, never>(
  "task_remove",
  Effect.gen(function* () {
    return {
      description: DESCRIPTION,
      parameters: Parameters,
      execute: (params: Schema.Schema.Type<typeof Parameters>, ctx: Tool.Context<Metadata>) =>
        Effect.gen(function* () {
          yield* ctx.ask({
            permission: "task_remove",
            patterns: ["*"],
            always: ["*"],
            metadata: {},
          })

          const svc = yield* getTaskService()
          const id = params.id as any
          const existing = yield* svc.get(id)

          if (!existing) {
            return {
              title: `Task not found: ${params.id}`,
              output: `Task not found: ${params.id}`,
              metadata: { id: params.id, removed: false },
            }
          }

          yield* svc.remove(id)

          return {
            title: `Removed task: ${existing.title}`,
            output: `Removed task "${existing.title}" (${params.id}) and its descendants.`,
            metadata: { id: params.id, removed: true },
          }
        }).pipe(Effect.orDie),
    } satisfies Tool.DefWithoutID<typeof Parameters, Metadata>
  }),
)
