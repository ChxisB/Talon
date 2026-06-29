import { Effect, Schema } from "effect"
import * as Tool from "./tool"
import { getArtifactService } from "./artifact-helper"
import DESCRIPTION from "./artifact-list.txt"

export const Parameters = Schema.Struct({
  type: Schema.optional(
    Schema.Literals(["spec", "ticket", "story", "review", "plan", "note"]),
  ).annotate({ description: "Optional filter by artifact type" }),
  status: Schema.optional(
    Schema.Literals(["todo", "in_progress", "done", "blocked", "archived"]),
  ).annotate({ description: "Optional filter by status" }),
  taskID: Schema.optional(Schema.String).annotate({
    description: "Optional filter by task tracking ID",
  }),
})

type Metadata = {
  count: number
  types: string[]
}

export const ArtifactListTool = Tool.define<typeof Parameters, Metadata, never>(
  "artifact_list",
  Effect.gen(function* () {
    return {
      description: DESCRIPTION,
      parameters: Parameters,
      execute: (params: Schema.Schema.Type<typeof Parameters>, ctx: Tool.Context<Metadata>) =>
        Effect.gen(function* () {
          yield* ctx.ask({
            permission: "artifact_list",
            patterns: ["*"],
            always: ["*"],
            metadata: {},
          })

          const svc = yield* getArtifactService()
          const artifacts = yield* svc.list({
            type: params.type,
            status: params.status,
            taskID: params.taskID,
          })

          if (artifacts.length === 0) {
            const filter = [
              params.type ? `type=${params.type}` : "",
              params.status ? `status=${params.status}` : "",
              params.taskID ? `task=${params.taskID}` : "",
            ]
              .filter(Boolean)
              .join(", ")
            const msg = filter ? `No artifacts found (${filter})` : "No artifacts found"
            return {
              title: msg,
              output: msg,
              metadata: { count: 0, types: [] },
            }
          }

          const lines: string[] = []
          let currentType = ""
          for (const a of artifacts) {
            if (a.type !== currentType) {
              currentType = a.type
              lines.push(`\n${currentType.toUpperCase()}:`)
            }
            lines.push(`  ${a.id}  ${a.status.padEnd(12)}  ${a.title}`)
          }

          const types = [...new Set(artifacts.map((a) => a.type))]

          return {
            title: `${artifacts.length} artifact(s)`,
            output: lines.join("\n").trim(),
            metadata: { count: artifacts.length, types },
          }
        }).pipe(Effect.orDie),
    } satisfies Tool.DefWithoutID<typeof Parameters, Metadata>
  }),
)
