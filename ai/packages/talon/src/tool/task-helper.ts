/**
 * Helper to get TaskV2.Service within tool execution and non-session contexts.
 *
 * ProjectV2.Service is provided on-demand (NOT from AppLayer) to avoid
 * the Worker thread inadvertently initializing the database connection
 * (which causes SQLite misuse crashes in compiled binaries).
 */
import { Context, Effect, Layer } from "effect"
import { Service as TaskService, defaultLayer as taskDefaultLayer } from "@talon-ai/core/control-plane/task/index"
import type { Interface as TaskInterface } from "@talon-ai/core/control-plane/task/index"
import { LocationServiceMap } from "@talon-ai/core/location-layer"
import { Location } from "@talon-ai/core/location"
import { AbsolutePath } from "@talon-ai/core/schema"
import { InstanceState } from "@/effect/instance-state"
import { ProjectV2 } from "@talon-ai/core/project"

export const getTaskService = Effect.fn("TaskHelper.getService")(function* () {
  const instance = yield* InstanceState.context
  const fullCtx = yield* Effect.context<never>()
  const locations = Context.getUnsafe(fullCtx, LocationServiceMap)

  const locationLayer = locations.get(
    Location.Ref.make({ directory: AbsolutePath.make(instance.directory) }),
  )

  return yield* Effect.provide(
    Effect.provide(
      Effect.gen(function* () {
        return yield* TaskService
      }),
      taskDefaultLayer,
    ),
    locationLayer,
  )
})

export const resolveTaskService = (
  instance: { directory: string },
): Effect.Effect<TaskInterface, never, never> =>
  Effect.gen(function* () {
    const ref = Location.Ref.make({ directory: AbsolutePath.make(instance.directory) })

    // Provide ProjectV2 on-demand (merged with taskDefaultLayer).
    const combinedLayer = taskDefaultLayer.pipe(
      Layer.provideMerge(ProjectV2.defaultLayer),
    )

    return yield* Effect.provide(
      Effect.gen(function* () {
        const projectSvc = yield* ProjectV2.Service
        const resolved = yield* projectSvc.resolve(ref.directory).pipe(Effect.orDie)

        const location = Location.Service.of({
          directory: ref.directory,
          workspaceID: ref.workspaceID,
          project: { id: resolved.id, directory: resolved.directory },
          vcs: resolved.vcs,
        })

        return yield* Effect.provide(
          Effect.gen(function* () {
            return yield* TaskService
          }),
          Layer.succeed(Location.Service, location),
        )
      }),
      combinedLayer,
    )
  }) as Effect.Effect<TaskInterface, never, never>
