import { Effect } from "effect"
import { HttpApiBuilder, HttpApiError } from "effect/unstable/httpapi"
import { ArtifactV2 } from "@talon-ai/core/artifact"
import type { Interface as ArtifactInterface } from "@talon-ai/core/artifact/index"
import { InstanceHttpApi } from "../api"
import { resolveArtifactService } from "@/tool/artifact-helper"
import { InstanceState } from "@/effect/instance-state"
import type {
  CreateArtifactPayloadType,
  ListArtifactQueryType,
  UpdateArtifactPayloadType,
} from "../groups/artifact"

export const artifactHandlers = HttpApiBuilder.group(InstanceHttpApi, "artifact", (handlers) =>
  Effect.gen(function* () {
    // Resolve the ArtifactService once at the group level (avoids per-handler
    // service resolution which would add unwanted R requirements to handler types).
    const ctx = yield* InstanceState.context
    const svc: ArtifactInterface = yield* resolveArtifactService(ctx)

    const list = Effect.fn("ArtifactHttpApi.list")(function* (ctx: {
      query: ListArtifactQueryType
    }) {
      return yield* svc.list({
        type: ctx.query.type,
        status: ctx.query.status,
        taskID: ctx.query.taskID,
      })
    })

    const get = Effect.fn("ArtifactHttpApi.get")(function* (ctx: {
      params: { id: ArtifactV2.ID }
    }) {
      const artifact = yield* svc.get(ctx.params.id)
      if (!artifact) return yield* Effect.fail(new HttpApiError.BadRequest({}))
      return artifact
    })

    const create = Effect.fn("ArtifactHttpApi.create")(function* (ctx: {
      payload: CreateArtifactPayloadType
    }) {
      return yield* svc.create({
        type: ctx.payload.type,
        title: ctx.payload.title,
        body: ctx.payload.body,
        status: ctx.payload.status,
        parentID: ctx.payload.parentID,
        taskID: ctx.payload.taskID,
        assignee: ctx.payload.assignee,
      })
    })

    const update = Effect.fn("ArtifactHttpApi.update")(function* (ctx: {
      params: { id: ArtifactV2.ID }
      payload: UpdateArtifactPayloadType
    }) {
      const existing = yield* svc.get(ctx.params.id)
      if (!existing) return yield* Effect.fail(new HttpApiError.NotFound({}))
      yield* svc.update(ctx.params.id, ctx.payload)
      const updated = yield* svc.get(ctx.params.id)
      if (!updated) return yield* Effect.fail(new HttpApiError.NotFound({}))
      return updated
    })

    const remove = Effect.fn("ArtifactHttpApi.remove")(function* (ctx: {
      params: { id: ArtifactV2.ID }
    }) {
      const existing = yield* svc.get(ctx.params.id)
      if (!existing) return yield* Effect.fail(new HttpApiError.NotFound({}))
      yield* svc.remove(ctx.params.id)
      return true
    })

    return handlers
      .handle("list", list)
      .handle("get", get)
      .handle("create", create)
      .handle("update", update)
      .handle("remove", remove)
  }),
)
