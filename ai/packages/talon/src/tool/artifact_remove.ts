import { Effect, Schema } from "effect"
import * as Tool from "./tool"
import { getArtifactService } from "./artifact-helper"
import DESCRIPTION from "./artifact-remove.txt"

export const Parameters = Schema.Struct({
  id: Schema.String.annotate({ description: "The artifact ID to remove (starts with \"art_\")" }),
})

type Metadata = {
  id: string
  removed: boolean
}

export const ArtifactRemoveTool = Tool.define<typeof Parameters, Metadata, never>(
  "artifact_remove",
  Effect.gen(function* () {
    return {
      description: DESCRIPTION,
      parameters: Parameters,
      execute: (params: Schema.Schema.Type<typeof Parameters>, ctx: Tool.Context<Metadata>) =>
        Effect.gen(function* () {
          yield* ctx.ask({
            permission: "artifact_remove",
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
              metadata: { id: params.id, removed: false },
            }
          }

          yield* svc.remove(id)

          return {
            title: `Removed artifact: ${existing.title}`,
            output: `Removed artifact "${existing.title}" (${params.id}) and its descendants.`,
            metadata: { id: params.id, removed: true },
          }
        }).pipe(Effect.orDie),
    } satisfies Tool.DefWithoutID<typeof Parameters, Metadata>
  }),
)
