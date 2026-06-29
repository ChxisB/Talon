import { Effect, Schema } from "effect"
import * as Tool from "./tool"
import { getArtifactService } from "./artifact-helper"
import DESCRIPTION from "./artifact-create.txt"

export const Parameters = Schema.Struct({
  type: Schema.Literals(["spec", "ticket", "story", "review", "plan", "note"]).annotate({
    description: "The artifact category",
  }),
  title: Schema.String.annotate({ description: "A short descriptive title" }),
  body: Schema.optional(Schema.String).annotate({
    description: "Optional markdown body content",
  }),
  status: Schema.optional(
    Schema.Literals(["todo", "in_progress", "done", "blocked", "archived"]),
  ).annotate({ description: "Optional initial status (default: todo)" }),
  assignee: Schema.optional(Schema.String).annotate({
    description: "Optional person/agent assigned",
  }),
  parentID: Schema.optional(Schema.String).annotate({
    description: "Optional parent artifact ID for hierarchical organization",
  }),
  taskID: Schema.optional(Schema.String).annotate({
    description: "Optional external task tracking ID",
  }),
})

type Metadata = {
  id: string
  type: string
  title: string
  status: string
}

export const ArtifactCreateTool = Tool.define<typeof Parameters, Metadata, never>(
  "artifact_create",
  Effect.gen(function* () {
    return {
      description: DESCRIPTION,
      parameters: Parameters,
      execute: (params: Schema.Schema.Type<typeof Parameters>, ctx: Tool.Context<Metadata>) =>
        Effect.gen(function* () {
          yield* ctx.ask({
            permission: "artifact_create",
            patterns: ["*"],
            always: ["*"],
            metadata: {},
          })

          const svc = yield* getArtifactService()
          const result = yield* svc.create({
            type: params.type,
            title: params.title,
            body: params.body,
            status: params.status ?? "todo",
            assignee: params.assignee,
            parentID: params.parentID as any,
            taskID: params.taskID,
          })

          return {
            title: `Created ${result.type} artifact: ${result.title}`,
            output: [
              `Created ${result.type} artifact: ${result.title}`,
              `  ID: ${result.id}`,
              `  Status: ${result.status}`,
              `  Path: ${result.path}`,
            ].join("\n"),
            metadata: {
              id: result.id,
              type: result.type,
              title: result.title,
              status: result.status,
            },
          }
        }).pipe(Effect.orDie),
    } satisfies Tool.DefWithoutID<typeof Parameters, Metadata>
  }),
)
