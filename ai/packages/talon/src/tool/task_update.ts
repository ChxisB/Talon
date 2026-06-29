import { Effect, Schema } from "effect"
import * as Tool from "./tool"
import { getTaskService } from "./task-helper"
import DESCRIPTION from "./task-update.txt"

export const Parameters = Schema.Struct({
  id: Schema.String.annotate({ description: "The task ID (starts with \"tsk_\")" }),
  title: Schema.optional(Schema.String).annotate({ description: "Optional new title" }),
  status: Schema.optional(
    Schema.Literals(["todo", "in_progress", "done", "blocked", "archived"]),
  ).annotate({ description: "Optional new status" }),
  assignee: Schema.optional(Schema.String).annotate({
    description: "Optional new assignee (empty string to clear)",
  }),
  summary: Schema.optional(Schema.String).annotate({
    description: "Optional new summary",
  }),
})

type Metadata = {
  id: string
  title?: string
  status?: string
}

export const TaskUpdateTool = Tool.define<typeof Parameters, Metadata, never>(
  "task_update",
  Effect.gen(function* () {
    return {
      description: DESCRIPTION,
      parameters: Parameters,
      execute: (params: Schema.Schema.Type<typeof Parameters>, ctx: Tool.Context<Metadata>) =>
        Effect.gen(function* () {
          yield* ctx.ask({
            permission: "task_update",
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
              metadata: { id: params.id },
            }
          }

          const updates: Record<string, unknown> = {}
          if (params.title !== undefined) updates.title = params.title
          if (params.status !== undefined) updates.status = params.status
          if (params.assignee !== undefined) updates.assignee = params.assignee || null
          if (params.summary !== undefined) updates.summary = params.summary

          yield* svc.update(id, updates)

          return {
            title: `Updated task: ${existing.title}`,
            output: `Updated task "${existing.title}" (${params.id})`,
            metadata: {
              id: params.id,
              title: params.title,
              status: params.status,
            },
          }
        }).pipe(Effect.orDie),
    } satisfies Tool.DefWithoutID<typeof Parameters, Metadata>
  }),
)
