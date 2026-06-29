import { describe, expect, it } from "bun:test"
import path from "path"
import { Effect, Layer } from "effect"
import { Database } from "@talon-ai/core/database/database"
import { Location } from "@talon-ai/core/location"
import { ProjectV2 } from "@talon-ai/core/project"
import { ProjectTable } from "@talon-ai/core/project/sql"
import { FSUtil } from "@talon-ai/core/fs-util"
import { AbsolutePath } from "@talon-ai/core/schema"
import { SystemContext } from "@talon-ai/core/system-context"
import { SystemContextRegistry } from "@talon-ai/core/system-context/registry"
import * as ArtifactService from "@talon-ai/core/artifact/index"
import { ArtifactContext } from "@talon-ai/core/artifact/context"
import { tmpdir } from "./fixture/tmpdir"
import { location } from "./fixture/location"

async function withTmpDir<A>(fn: (dir: string) => Promise<A>): Promise<A> {
  const tmp = await tmpdir()
  try {
    return await fn(tmp.path)
  } finally {
    await tmp[Symbol.asyncDispose]()
  }
}

const databaseLayer = Database.layerFromPath(":memory:")

const provideFor = (dir: string) => {
  const locationLayer = Layer.succeed(
    Location.Service,
    Location.Service.of(
      location(new Location.Ref({ directory: AbsolutePath.make(dir) }), {
        projectDirectory: AbsolutePath.make(dir),
        vcs: { type: "git", store: AbsolutePath.make(dir) } as ProjectV2.Vcs,
      }),
    ),
  )
  // Build ArtifactService with all its deps self-contained
  const artifactWithDeps = ArtifactService.layer.pipe(
    Layer.provide(databaseLayer),
    Layer.provide(FSUtil.defaultLayer),
    Layer.provide(locationLayer),
  )
  // Build ArtifactContext with registry + ArtifactService + FSUtil self-contained,
  // but keep ArtifactService.Service and SystemContextRegistry.Service in the output
  const contextWithDeps = ArtifactContext.layer.pipe(
    Layer.provideMerge(SystemContextRegistry.layer),
    Layer.provideMerge(artifactWithDeps),
    Layer.provide(FSUtil.defaultLayer),
  )
  // Merge with databaseLayer so the test body can access Database.Service for seeding
  return Layer.mergeAll(contextWithDeps, databaseLayer)
}

const run = <A, E>(
  dir: string,
  effect: Effect.Effect<A, E, ArtifactService.Service | SystemContextRegistry.Service>,
) =>
  Effect.runPromise(
    Effect.gen(function* () {
      // Seed the project row so the artifact.project_id FK is satisfied.
      const { db } = yield* Database.Service
      yield* db
        .insert(ProjectTable)
        .values({
          id: ProjectV2.ID.global,
          worktree: AbsolutePath.make(dir),
          sandboxes: [],
          time_created: Date.now(),
          time_updated: Date.now(),
        })
        .run()
        .pipe(Effect.orDie)
      yield* effect
    }).pipe(Effect.provide(provideFor(dir)), Effect.scoped),
  )

describe("ArtifactContext", () => {
  it("baseline renders the artifact index with in_progress bodies", async () => {
    await withTmpDir(async (dir) => {
      await run(dir, Effect.gen(function* () {
        const service = yield* ArtifactService.Service
        yield* service.create({ type: "spec", title: "Architecture spec", body: "Use Effect-TS" })
        yield* service.create({ type: "ticket", title: "Implement throttle", body: "Add rate limiter", status: "in_progress" })
        yield* service.create({ type: "ticket", title: "Write tests", status: "todo" })

        const registry = yield* SystemContextRegistry.Service
        const context = yield* registry.load()
        const initialized = yield* SystemContext.initialize(context)

        expect(initialized.baseline).toContain("Active artifacts for this project:")
        expect(initialized.baseline).toContain("spec:")
        expect(initialized.baseline).toContain("Architecture spec")
        expect(initialized.baseline).toContain("ticket:")
        expect(initialized.baseline).toContain("Implement throttle")
        // in_progress body is included
        expect(initialized.baseline).toContain("Add rate limiter")
        // todo body is NOT included (only in_progress gets bodies)
        expect(initialized.baseline).not.toContain("Use Effect-TS")
        expect(initialized.baseline).toContain("Write tests")
      }))
    })
  })

  it("update emits changed artifacts on reconcile", async () => {
    await withTmpDir(async (dir) => {
      await run(dir, Effect.gen(function* () {
        const service = yield* ArtifactService.Service
        const ticket = yield* service.create({ type: "ticket", title: "Implement throttle", body: "Do it", status: "todo" })

        const registry = yield* SystemContextRegistry.Service
        const context = yield* registry.load()
        const initialized = yield* SystemContext.initialize(context)

        // Change the ticket status
        yield* service.update(ticket.id, { status: "in_progress" })

        const result = yield* SystemContext.reconcile(yield* registry.load(), initialized.snapshot)
        expect(result._tag).toBe("Updated")
        if (result._tag === "Updated") {
          expect(result.text).toContain("Artifacts have been updated.")
          expect(result.text).toContain("Implement throttle")
          expect(result.text).toContain("in_progress")
        }
      }))
    })
  })

  it("removed emits removal text when all artifacts are deleted", async () => {
    await withTmpDir(async (dir) => {
      await run(dir, Effect.gen(function* () {
        const service = yield* ArtifactService.Service
        const ticket = yield* service.create({ type: "ticket", title: "Implement throttle", body: "Do it" })

        const registry = yield* SystemContextRegistry.Service
        const context = yield* registry.load()
        const initialized = yield* SystemContext.initialize(context)

        // Delete all artifacts
        yield* service.remove(ticket.id)

        const result = yield* SystemContext.reconcile(yield* registry.load(), initialized.snapshot)
        expect(result._tag).toBe("Updated")
        if (result._tag === "Updated") {
          expect(result.text).toContain("Previously loaded artifacts no longer apply.")
        }
      }))
    })
  })

  it("empty artifacts produce no artifact context (SystemContext.empty)", async () => {
    await withTmpDir(async (dir) => {
      await run(dir, Effect.gen(function* () {
        const registry = yield* SystemContextRegistry.Service
        const context = yield* registry.load()
        const initialized = yield* SystemContext.initialize(context)

        // With no artifacts, the baseline should NOT contain artifact text.
        // Other sources (environment, date) are still present.
        expect(initialized.baseline).not.toContain("Active artifacts for this project:")
      }))
    })
  })
})
