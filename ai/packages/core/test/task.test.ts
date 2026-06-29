import { describe, expect, it } from "bun:test"
import { Effect, Layer } from "effect"
import { eq, sql } from "drizzle-orm"
import { Database } from "@talon-ai/core/database/database"
import { Location } from "@talon-ai/core/location"
import { ProjectV2 } from "@talon-ai/core/project"
import { ProjectTable } from "@talon-ai/core/project/sql"
import { FSUtil } from "@talon-ai/core/fs-util"
import { AbsolutePath } from "@talon-ai/core/schema"
import { Service as TaskService, layer as taskLayer } from "@talon-ai/core/control-plane/task/index"
import { TaskTable } from "@talon-ai/core/control-plane/task.sql"
import { SessionTable } from "@talon-ai/core/session/sql"
import { ArtifactTable } from "@talon-ai/core/artifact/sql"
import { AccountStateTable } from "@talon-ai/core/account/sql"
import { tmpdir } from "./fixture/tmpdir"
import { location } from "./fixture/location"

const databaseLayer = Database.layerFromPath(":memory:")

const locationLayerFor = (dir: string) =>
  Layer.succeed(
    Location.Service,
    Location.Service.of(
      location(new Location.Ref({ directory: AbsolutePath.make(dir) }), {
        projectDirectory: AbsolutePath.make(dir),
        vcs: undefined,
      }),
    ),
  )

const provideFor = (dir: string) =>
  Layer.mergeAll(
    taskLayer.pipe(
      Layer.provide(databaseLayer),
      Layer.provide(FSUtil.defaultLayer),
      Layer.provide(locationLayerFor(dir)),
    ),
    databaseLayer,
  )

const run = <A, R>(dir: string, effect: Effect.Effect<A, never, R>) =>
  Effect.runPromise(
    Effect.gen(function* () {
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
      // Seed account_state row (single row with id=1)
      yield* db
        .insert(AccountStateTable)
        .values({ id: 1 })
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
    await Bun.$`rm -rf ${tmp.path}`.quiet()
  }
}

describe("Task", () => {
  it("create + get returns the task", async () => {
    await withTmpDir(async (dir) => {
      await run(dir, Effect.gen(function* () {
        const svc = yield* TaskService
        const created = yield* svc.create({ title: "Fix login bug" })
        expect(created.title).toBe("Fix login bug")
        expect(created.status).toBe("todo")
        expect(created.id).toStartWith("tsk_")
        const fetched = yield* svc.get(created.id)
        expect(fetched).toBeDefined()
        expect(fetched!.title).toBe("Fix login bug")
      }))
    })
  })

  it("create with explicit status and assignee", async () => {
    await withTmpDir(async (dir) => {
      await run(dir, Effect.gen(function* () {
        const svc = yield* TaskService
        const created = yield* svc.create({ title: "Important", status: "in_progress", assignee: "build" })
        expect(created.status).toBe("in_progress")
        expect(created.assignee).toBe("build")
      }))
    })
  })

  it("list returns all tasks for the project", async () => {
    await withTmpDir(async (dir) => {
      await run(dir, Effect.gen(function* () {
        const svc = yield* TaskService
        yield* svc.create({ title: "Task A" })
        yield* svc.create({ title: "Task B" })
        const tasks = yield* svc.list()
        expect(tasks).toHaveLength(2)
      }))
    })
  })

  it("list filters by status", async () => {
    await withTmpDir(async (dir) => {
      await run(dir, Effect.gen(function* () {
        const svc = yield* TaskService
        yield* svc.create({ title: "A", status: "todo" })
        yield* svc.create({ title: "B", status: "done" })
        const doneTasks = yield* svc.list({ status: "done" })
        expect(doneTasks).toHaveLength(1)
        expect(doneTasks[0].title).toBe("B")
      }))
    })
  })

  it("update modifies task fields", async () => {
    await withTmpDir(async (dir) => {
      await run(dir, Effect.gen(function* () {
        const svc = yield* TaskService
        const created = yield* svc.create({ title: "Original" })
        yield* svc.update(created.id, { title: "Updated", status: "in_progress" })
        const fetched = yield* svc.get(created.id)
        expect(fetched!.title).toBe("Updated")
        expect(fetched!.status).toBe("in_progress")
      }))
    })
  })

  it("remove deletes the task", async () => {
    await withTmpDir(async (dir) => {
      await run(dir, Effect.gen(function* () {
        const svc = yield* TaskService
        const created = yield* svc.create({ title: "Delete me" })
        yield* svc.remove(created.id)
        const fetched = yield* svc.get(created.id)
        expect(fetched).toBeUndefined()
      }))
    })
  })

  it("focus/focused round-trips", async () => {
    await withTmpDir(async (dir) => {
      await run(dir, Effect.gen(function* () {
        const svc = yield* TaskService
        const initial = yield* svc.focused()
        expect(initial).toBeUndefined()
        const created = yield* svc.create({ title: "Focus target" })
        yield* svc.focus(created.id)
        const focused = yield* svc.focused()
        expect(focused).toBe(created.id)
      }))
    })
  })

  it("aggregate returns task with sessions and artifacts", async () => {
    await withTmpDir(async (dir) => {
      await run(dir, Effect.gen(function* () {
        const svc = yield* TaskService
        const { db } = yield* Database.Service
        const created = yield* svc.create({ title: "Aggregate test" })
        // Insert a session linked to the task via raw SQL to avoid Drizzle type issues
        const now = Date.now()
        yield* db.run(
          sql`INSERT INTO session (id, project_id, task_id, slug, directory, title, version, cost, tokens_input, tokens_output, tokens_reasoning, tokens_cache_read, tokens_cache_write, time_created, time_updated) VALUES (${"ses_test_session"}, ${ProjectV2.ID.global}, ${created.id}, ${"test-session"}, ${dir}, ${"Test session"}, ${"1.0"}, 0, 0, 0, 0, 0, 0, ${now}, ${now})`,
        ).pipe(Effect.orDie)
        // Insert an artifact linked to the task via raw SQL
        yield* db.run(
          sql`INSERT INTO artifact (id, type, title, status, task_id, project_id, path, time_created, time_updated) VALUES (${"art_test_artifact"}, ${"ticket"}, ${"Test ticket"}, ${"todo"}, ${created.id}, ${ProjectV2.ID.global}, ${`${dir}/test.md`}, ${now}, ${now})`,
        ).pipe(Effect.orDie)
        const agg = yield* svc.aggregate(created.id)
        expect(agg.task.id).toBe(created.id)
        expect(agg.sessions).toHaveLength(1)
        expect(agg.sessions[0].title).toBe("Test session")
        expect(agg.artifacts).toHaveLength(1)
        expect(agg.artifacts[0].type).toBe("ticket")
      }))
    })
  })

  it("list returns empty array when no tasks exist", async () => {
    await withTmpDir(async (dir) => {
      await run(dir, Effect.gen(function* () {
        const svc = yield* TaskService
        const tasks = yield* svc.list()
        expect(tasks).toEqual([])
      }))
    })
  })

  it("get on non-existent id returns undefined", async () => {
    await withTmpDir(async (dir) => {
      await run(dir, Effect.gen(function* () {
        const svc = yield* TaskService
        const result = yield* svc.get("tsk_nonexistent" as any)
        expect(result).toBeUndefined()
      }))
    })
  })
})
