/**
 * Helper to get ArtifactV2.Service within tool execution context.
 *
 * ArtifactV2.Service depends on Location.Service (per-project), available
 * only via LocationServiceMap. Since tools are built globally but run
 * per-location, resolve the service lazily at execution time.
 *
 * ProjectV2.Service is provided on-demand (NOT from AppLayer) to avoid
 * the Worker thread inadvertently initializing the database connection
 * (which causes SQLite misuse crashes in compiled binaries).
 */
import { Context, Effect, Layer } from "effect"
import { Service as ArtifactService, defaultLayer as artifactDefaultLayer } from "@talon-ai/core/artifact/index"
import type { Interface as ArtifactInterface } from "@talon-ai/core/artifact/index"
import { LocationServiceMap } from "@talon-ai/core/location-layer"
import { Location } from "@talon-ai/core/location"
import { AbsolutePath } from "@talon-ai/core/schema"
import { InstanceState } from "@/effect/instance-state"
import { ProjectV2 } from "@talon-ai/core/project"

export const getArtifactService = Effect.fn("ArtifactHelper.getService")(function* () {
  const instance = yield* InstanceState.context

  // Use Context.getUnsafe to extract LocationServiceMap from the runtime context
  // WITHOUT adding a type-level requirement (R=never). The tool's DefWithoutID
  // type-check expects execute to return Effect<ExecuteResult<M>, never, never> —
  // but LocationServiceMap is only provided by the session layer at execution
  // time, not at init. Effect.context<never>() returns the full runtime context
  // with R=never, and Context.getUnsafe has no I extends Services constraint so
  // it can freely extract any service from any context at runtime.
  const fullCtx = yield* Effect.context<never>()
  const locations = Context.getUnsafe(fullCtx, LocationServiceMap)

  const locationLayer = locations.get(
    Location.Ref.make({ directory: AbsolutePath.make(instance.directory) }),
  )

  // Build ArtifactV2.Service within the location context:
  //   Effect.provide(effect, layer) — satisfy effect's requirements with layer's outputs
  return yield* Effect.provide(
    Effect.provide(
      Effect.gen(function* () {
        return yield* ArtifactService
      }),
      artifactDefaultLayer,
    ),
    locationLayer,
  )
})

/**
 * Resolve ArtifactV2.Service for non-session contexts (CLI commands, server handlers).
 *
 * Unlike `getArtifactService` (which extracts LocationServiceMap from the runtime
 * context at tool-execution time), this helper builds the service from scratch
 * using a directory path. It provides ProjectV2 on-demand to avoid initializing
 * the database in the Worker thread (which causes SQLite misuse crashes in
 * compiled binaries).
 *
 * Usage from an effectCmd handler:
 * ```
 * const ctx = yield* InstanceState.context
 * const svc = yield* resolveArtifactService(ctx)
 * ```
 */
export const resolveArtifactService = (
  instance: { directory: string },
): Effect.Effect<ArtifactInterface, never, never> =>
  Effect.gen(function* () {
    const ref = Location.Ref.make({ directory: AbsolutePath.make(instance.directory) })

    // Provide ProjectV2 on-demand (merged with artifactDefaultLayer so their
    // shared dependencies like FSUtil are resolved once).
    const combinedLayer = artifactDefaultLayer.pipe(
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
            return yield* ArtifactService
          }),
          Layer.succeed(Location.Service, location),
        )
      }),
      combinedLayer,
    )
  }) as Effect.Effect<ArtifactInterface, never, never>
