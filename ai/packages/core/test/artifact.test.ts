import { describe, expect, it } from "bun:test"
import path from "path"
import { Effect, Layer } from "effect"
import { eq } from "drizzle-orm"
import { Database } from "@talon-ai/core/database/database"
import { Location } from "@talon-ai/core/location"
import { ProjectV2 } from "@talon-ai/core/project"
import { ProjectTable } from "@talon-ai/core/project/sql"
import { FSUtil } from "@talon-ai/core/fs-util"
import { AbsolutePath } from "@talon-ai/core/schema"
import { ArtifactV2 } from "@talon-ai/core/artifact"
import { Service as ArtifactService, layer as artifactLayer } from "@talon-ai/core/artifact/index"
import { ArtifactTable } from "@talon-ai/core/artifact/sql"
import { parseFrontmatter } from "@talon-ai/core/artifact/frontmatter"
import { tmpdir } from "./fixture/tmpdir"
import { location } from "./fixture/location"

const databaseLayer = Database.layerFromPath(":memory:")

const locationLayerFor = (dir: string) =>
  Layer.succeed(
    Location.Service,
    Location.Service.of(
      location(new Location.Ref({ directory: AbsolutePath.make(dir) }), {
        projectDirectory: AbsolutePath.make(dir),
        vcs: { type: "git", store: AbsolutePath.make(dir) } as ProjectV2.Vcs,
      }),
    ),
  )

/** Provide the artifact service + a shared in-memory DB for assertions. */
const provideFor = (dir: string) =>
  Layer.mergeAll(
    artifactLayer.pipe(Layer.provide(databaseLayer), Layer.provide(FSUtil.defaultLayer), Layer.provide(locationLayerFor(dir))),
    databaseLayer,
  )

const run = <A, E>(dir: string, effect: Effect.Effect<A, E, ArtifactService | Database.Service>) =>
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

async function withTmpDir<A>(fn: (dir: string) => Promise<A>): Promise<A> {
  const tmp = await tmpdir()
  try {
    return await fn(tmp.path)
  } finally {
    await tmp[Symbol.asyncDispose]()
  }
}

const dbRow = (id: ArtifactV2.ID) =>
  Database.Service.use(({ db }) =>
    db.select().from(ArtifactTable).where(eq(ArtifactTable.id, id)).get().pipe(Effect.orDie),
  )

const fileText = (filepath: string) => Effect.promise(() => Bun.file(filepath).text())
const fileExists = (filepath: string) => Effect.promise(() => Bun.file(filepath).exists())

describe("Artifact", () => {
  it("round-trips create → get on disk and DB (VCS path)", async () => {
    await withTmpDir(async (dir) => {
      await run(dir, Effect.gen(function* () {
        const service = yield* ArtifactService
        const created = yield* service.create({ type: "ticket", title: "Implement login throttle", body: "Do the thing", assignee: "build" })

        expect(created.type).toBe("ticket")
        expect(created.title).toBe("Implement login throttle")
        expect(created.status).toBe("todo")
        expect(created.projectID).toBe(ProjectV2.ID.global)
        expect(created.path).toBe(path.join(dir, ".talon", "artifacts", "ticket", `${created.id}.md`))

        // DB row
        const row = yield* dbRow(created.id)
        expect(row?.title).toBe("Implement login throttle")
        expect(row?.body_hash).toBeTruthy()

        // File on disk with frontmatter
        const exists = yield* fileExists(created.path)
        expect(exists).toBe(true)
        const { frontmatter, body } = parseFrontmatter(yield* fileText(created.path))
        expect(frontmatter.id).toBe(created.id)
        expect(frontmatter.type).toBe("ticket")
        expect(frontmatter.title).toBe("Implement login throttle")
        expect(frontmatter.status).toBe("todo")
        expect(frontmatter.assignee).toBe("build")
        expect(body).toBe("Do the thing")

        // get returns the same
        const got = yield* service.get(created.id)
        expect(got?.id).toBe(created.id)
      }))
    })
  })

  it("update title/status/body syncs disk and DB", async () => {
    await withTmpDir(async (dir) => {
      await run(dir, Effect.gen(function* () {
        const service = yield* ArtifactService
        const created = yield* service.create({ type: "spec", title: "Draft", body: "old body" })
        yield* service.update(created.id, { title: "Final spec", status: "approved" as never, body: "new body" })

        const row = yield* dbRow(created.id)
        expect(row?.title).toBe("Final spec")
        expect(row?.status).toBe("approved")

        const { frontmatter, body } = parseFrontmatter(yield* fileText(created.path))
        expect(frontmatter.title).toBe("Final spec")
        expect(frontmatter.status).toBe("approved")
        expect(body).toBe("new body")
      }))
    })
  })

  it("move reparents and reassigns without touching the file", async () => {
    await withTmpDir(async (dir) => {
      await run(dir, Effect.gen(function* () {
        const service = yield* ArtifactService
        const parent = yield* service.create({ type: "story", title: "Epic" })
        const child = yield* service.create({ type: "ticket", title: "Child" })
        const before = yield* fileText(child.path)

        yield* service.move(child.id, { parentID: parent.id, taskID: "tsk_1", orderKey: "001" })

        const row = yield* dbRow(child.id)
        expect(row?.parent_id).toBe(parent.id)
        expect(row?.task_id).toBe("tsk_1")
        expect(row?.order_key).toBe("001")
        // File body unchanged (path is stable on move)
        const after = yield* fileText(child.path)
        expect(after).toBe(before)
      }))
    })
  })

  it("remove deletes the artifact and its descendants", async () => {
    await withTmpDir(async (dir) => {
      await run(dir, Effect.gen(function* () {
        const service = yield* ArtifactService
        const parent = yield* service.create({ type: "story", title: "Epic" })
        const child = yield* service.create({ type: "ticket", title: "Child", parentID: parent.id })
        const grandchild = yield* service.create({ type: "ticket", title: "Grandchild", parentID: child.id })

        yield* service.remove(parent.id)

        expect(yield* dbRow(parent.id)).toBeUndefined()
        expect(yield* dbRow(child.id)).toBeUndefined()
        expect(yield* dbRow(grandchild.id)).toBeUndefined()
        expect(yield* fileExists(parent.path)).toBe(false)
        expect(yield* fileExists(child.path)).toBe(false)
        expect(yield* fileExists(grandchild.path)).toBe(false)
      }))
    })
  })

  it("list filters by type/status/taskID", async () => {
    await withTmpDir(async (dir) => {
      await run(dir, Effect.gen(function* () {
        const service = yield* ArtifactService
        yield* service.create({ type: "ticket", title: "t1", status: "todo" })
        yield* service.create({ type: "ticket", title: "t2", status: "in_progress", taskID: "tsk_x" })
        yield* service.create({ type: "spec", title: "s1" })

        expect((yield* service.list({ type: "ticket" })).length).toBe(2)
        expect((yield* service.list({ type: "spec" })).length).toBe(1)
        expect((yield* service.list({ status: "in_progress" })).length).toBe(1)
        expect((yield* service.list({ taskID: "tsk_x" })).length).toBe(1)
        expect((yield* service.list()).length).toBe(3)
      }))
    })
  })

  it("reindex inserts new files, updates drifted bodies, removes orphaned rows", async () => {
    await withTmpDir(async (dir) => {
      await run(dir, Effect.gen(function* () {
        const service = yield* ArtifactService
        const keep = yield* service.create({ type: "ticket", title: "keep", body: "original" })
        const drift = yield* service.create({ type: "ticket", title: "drift", body: "original" })
        const orphan = yield* service.create({ type: "ticket", title: "orphan" })

        // Externally edit `drift`'s body (disk drifts from DB).
        yield* Effect.promise(() => Bun.write(drift.path, `---\nid: ${drift.id}\ntype: ticket\ntitle: drift\nstatus: todo\n---\n\nexternally edited\n`))
        // Externally delete `orphan`'s file.
        yield* Effect.promise(() => Bun.$`rm -f ${orphan.path}`)
        // Externally add a brand-new artifact file with a fresh id.
        const newID = ArtifactV2.ID.create()
        const newPath = path.join(dir, ".talon", "artifacts", "spec", `${newID}.md`)
        yield* Effect.promise(() => Bun.write(newPath, `---\nid: ${newID}\ntype: spec\ntitle: new from disk\nstatus: todo\n---\n\nfresh\n`))

        const result = yield* service.reindex()
        expect(result.inserted).toBe(1) // the new spec file
        expect(result.updated).toBe(1) // drift's body changed
        expect(result.removed).toBe(1) // orphan's file deleted

        // keep unchanged in DB
        const keepRow = yield* dbRow(keep.id)
        expect(keepRow?.body_hash).toBe(keep.bodyHash)
        // drift body_hash updated
        const driftRow = yield* dbRow(drift.id)
        expect(driftRow?.body_hash).not.toBe(drift.bodyHash)
        // orphan removed
        expect(yield* dbRow(orphan.id)).toBeUndefined()
        // new file indexed
        const newRow = yield* dbRow(newID)
        expect(newRow?.title).toBe("new from disk")
      }))
    })
  })
})
