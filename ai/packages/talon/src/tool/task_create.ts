import { Effect, Schema } from "effect"
import * as Tool from "./tool"
import { getTaskService } from "./task-helper"
import DESCRIPTION from "./task-create.txt"

export const Parameters = Schema.Struct({
  title: Schema.String.annotate({ description: "A short descriptive title for the task" }),
  status: Schema.optional(
    Schema.Literals(["todo", "in_progress", "done", "blocked", "archived"]),
  ).annotate({ description: "Optional initial status (default: todo)" }),
  assignee: Schema.optional(Schema.String).annotate({
    description: "Optional person/agent assigned",
  }),
  parentID: Schema.optional(Schema.String).annotate({
    description: "Optional parent task ID for hierarchical organization",
  }),
  summary: Schema.optional(Schema.String).annotate({
    description: "Optional longer summary or description of the task",
  }),
})

type Metadata = {
  id: string
  title: string
  status: string
}

export const TaskCreateTool = Tool.define<typeof Parameters, Metadata, never>(
  "task_create",
  Effect.gen(function* () {
    return {
      description: DESCRIPTION,
      parameters: Parameters,
      execute: (params: Schema.Schema.Type<typeof Parameters>, ctx: Tool.Context<Metadata>) =>
        Effect.gen(function* () {
          yield* ctx.ask({
            permission: "task_create",
            patterns: ["*"],
            always: ["*"],
            metadata: {},
          })

          const svc = yield* getTaskService()
          const result = yield* svc.create({
            title: params.title,
            status: params.status,
            assignee: params.assignee,
            parentID: params.parentID as any,
            summary: params.summary,
          })

          return {
            title: `Created task: ${result.title}`,
            output: [
              `Created task: ${result.title}`,
              `  ID: ${result.id}`,
              `  Status: ${result.status}`,
              ...(result.assignee ? [`  Assignee: ${result.assignee}`] : []),
            ].join("\n"),
            metadata: {
              id: result.id,
              title: result.title,
              status: result.status,
            },
          }
        }).pipe(Effect.orDie),
    } satisfies Tool.DefWithoutID<typeof Parameters, Metadata>
  }),
)
