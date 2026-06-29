import { Effect } from "effect"
import { HttpApiBuilder, HttpApiError } from "effect/unstable/httpapi"
import { TaskV2 } from "@talon-ai/core/task"
import type { Interface as TaskInterface } from "@talon-ai/core/control-plane/task/index"
import { InstanceHttpApi } from "../api"
import { resolveTaskService } from "@/tool/task-helper"
import { InstanceState } from "@/effect/instance-state"
import type {
  CreateTaskPayloadType,
  ListTaskQueryType,
  UpdateTaskPayloadType,
} from "../groups/task"

export const taskHandlers = HttpApiBuilder.group(InstanceHttpApi, "task", (handlers) =>
  Effect.gen(function* () {
    const ctx = yield* InstanceState.context
    const svc: TaskInterface = yield* resolveTaskService(ctx)

    const list = Effect.fn("TaskHttpApi.list")(function* (ctx: {
      query: ListTaskQueryType
    }) {
      return yield* svc.list({
        status: ctx.query.status,
      })
    })

    const get = Effect.fn("TaskHttpApi.get")(function* (ctx: {
      params: { id: TaskV2.ID }
    }) {
      const task = yield* svc.get(ctx.params.id)
      if (!task) return yield* Effect.fail(new HttpApiError.BadRequest({}))
      return task
    })

    const create = Effect.fn("TaskHttpApi.create")(function* (ctx: {
      payload: CreateTaskPayloadType
    }) {
      return yield* svc.create({
        title: ctx.payload.title,
        status: ctx.payload.status,
        assignee: ctx.payload.assignee,
        parentID: ctx.payload.parentID,
        summary: ctx.payload.summary,
      })
    })

    const update = Effect.fn("TaskHttpApi.update")(function* (ctx: {
      params: { id: TaskV2.ID }
      payload: UpdateTaskPayloadType
    }) {
      const existing = yield* svc.get(ctx.params.id)
      if (!existing) return yield* Effect.fail(new HttpApiError.NotFound({}))
      yield* svc.update(ctx.params.id, ctx.payload)
      const updated = yield* svc.get(ctx.params.id)
      if (!updated) return yield* Effect.fail(new HttpApiError.NotFound({}))
      return updated
    })

    const remove = Effect.fn("TaskHttpApi.remove")(function* (ctx: {
      params: { id: TaskV2.ID }
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
