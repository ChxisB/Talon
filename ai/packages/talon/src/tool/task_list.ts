import { Effect, Schema } from "effect"
import * as Tool from "./tool"
import { getTaskService } from "./task-helper"
import DESCRIPTION from "./task-list.txt"

export const Parameters = Schema.Struct({
  status: Schema.optional(
    Schema.Literals(["todo", "in_progress", "done", "blocked", "archived"]),
  ).annotate({ description: "Optional filter by status" }),
})

type Metadata = {
  count: number
  statuses: string[]
}

export const TaskListTool = Tool.define<typeof Parameters, Metadata, never>(
  "task_list",
  Effect.gen(function* () {
    return {
      description: DESCRIPTION,
      parameters: Parameters,
      execute: (params: Schema.Schema.Type<typeof Parameters>, ctx: Tool.Context<Metadata>) =>
        Effect.gen(function* () {
          yield* ctx.ask({
            permission: "task_list",
            patterns: ["*"],
            always: ["*"],
            metadata: {},
          })

          const svc = yield* getTaskService()
          const tasks = yield* svc.list({
            status: params.status,
          })

          if (tasks.length === 0) {
            const msg = params.status
              ? `No tasks found (status=${params.status})`
              : "No tasks found"
            return {
              title: msg,
              output: msg,
              metadata: { count: 0, statuses: [] },
            }
          }

          const lines: string[] = []
          for (const t of tasks) {
            lines.push(`  ${t.id}  ${t.status.padEnd(12)}  ${t.assignee?.padEnd(12) ?? "-".padEnd(12)}  ${t.title}`)
          }

          const statuses = [...new Set(tasks.map((t) => t.status))]

          return {
            title: `${tasks.length} task(s)`,
            output: lines.join("\n").trim(),
            metadata: { count: tasks.length, statuses },
          }
        }).pipe(Effect.orDie),
    } satisfies Tool.DefWithoutID<typeof Parameters, Metadata>
  }),
)
