import { Effect, Schema } from "effect"
import * as Tool from "./tool"
import { getArtifactService } from "./artifact-helper"
import DESCRIPTION from "./artifact-update.txt"

export const Parameters = Schema.Struct({
  id: Schema.String.annotate({ description: "The artifact ID (starts with \"art_\")" }),
  title: Schema.optional(Schema.String).annotate({ description: "Optional new title" }),
  status: Schema.optional(
    Schema.Literals(["todo", "in_progress", "done", "blocked", "archived"]),
  ).annotate({ description: "Optional new status" }),
  assignee: Schema.optional(Schema.String).annotate({
    description: "Optional new assignee (empty string to clear)",
  }),
  body: Schema.optional(Schema.String).annotate({
    description: "Optional new body content (replaces existing body)",
  }),
  taskID: Schema.optional(Schema.String).annotate({
    description: "Optional new task tracking ID (empty string to clear)",
  }),
})

type Metadata = {
  id: string
  title?: string
  status?: string
}

export const ArtifactUpdateTool = Tool.define<typeof Parameters, Metadata, never>(
  "artifact_update",
  Effect.gen(function* () {
    return {
      description: DESCRIPTION,
      parameters: Parameters,
      execute: (params: Schema.Schema.Type<typeof Parameters>, ctx: Tool.Context<Metadata>) =>
        Effect.gen(function* () {
          yield* ctx.ask({
            permission: "artifact_update",
            patterns: ["*"],
            always: ["*"],
            metadata: {},
          })

          const svc = yield* getArtifactService()
          const id = params.id as any
          const existing = yield* svc.get(id)

          if (!existing) {
            return {
              title: `Artifact not found: ${params.id}`,
              output: `Artifact not found: ${params.id}`,
              metadata: { id: params.id },
            }
          }

          const updates: Record<string, unknown> = {}
          if (params.title !== undefined) updates.title = params.title
          if (params.status !== undefined) updates.status = params.status
          if (params.assignee !== undefined) updates.assignee = params.assignee || null
          if (params.body !== undefined) updates.body = params.body
          if (params.taskID !== undefined) updates.taskID = params.taskID || null

          yield* svc.update(id, updates)

          const updated = yield* svc.get(id)

          return {
            title: `Updated artifact: ${updated?.title ?? params.id}`,
            output: [
              `Updated artifact: ${updated?.title ?? params.id}`,
              `  ID: ${params.id}`,
              ...(params.title ? [`  Title: ${params.title}`] : []),
              ...(params.status ? [`  Status: ${params.status}`] : []),
            ].join("\n"),
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
