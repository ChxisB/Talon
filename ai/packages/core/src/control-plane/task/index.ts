import { and, asc, eq, inArray } from "drizzle-orm"
import { Context, Effect, Layer } from "effect"
import type { EffectDrizzleSqlite } from "@talon-ai/effect-drizzle-sqlite"
import { Database } from "../../database/database"
import { Location } from "../../location"
import { FSUtil } from "../../fs-util"
import { Git } from "../../git"
import { ProjectV2 } from "../../project"
import { WorkspaceV2 } from "../../workspace"
import { TaskV2 } from "../../task"
import { TaskTable } from "../task.sql"
import { SessionTable } from "../../session/sql"
import { ArtifactTable } from "../../artifact/sql"
import { AccountStateTable } from "../../account/sql"
import { AbsolutePath } from "../../schema"

type DatabaseClient = EffectDrizzleSqlite.EffectSQLiteDatabase
export type Transaction = Parameters<Parameters<DatabaseClient["transaction"]>[0]>[0]

export type TaskStatus = "todo" | "in_progress" | "done" | "blocked" | "archived"

export type Task = {
  readonly id: TaskV2.ID
  readonly title: string
  readonly status: string
  readonly assignee: string | null
  // workspace_id is a deferred FK (task table column has no FK constraint); typed as
  // plain string | null to match the column type, not a branded ID.
  readonly workspaceID: string | null
  readonly projectID: ProjectV2.ID
  readonly parentID: TaskV2.ID | null
  readonly orderKey: string | null
  readonly summary: string | null
  readonly metadata: Record<string, unknown> | null
  readonly timeCreated: number
  readonly timeUpdated: number
  readonly timeArchived: number | null
}

export type TaskInput = {
  readonly title: string
  readonly status?: TaskStatus
  readonly assignee?: string
  readonly parentID?: TaskV2.ID
  readonly orderKey?: string
  readonly summary?: string
  readonly workspaceID?: WorkspaceV2.ID
}

export type TaskUpdate = Partial<{
  title: string
  status: TaskStatus
  assignee: string
  orderKey: string
  summary: string
  metadata: Record<string, unknown>
  workspaceID: WorkspaceV2.ID
}>

export type TaskAggregate = {
  readonly task: Task
  readonly sessions: Array<{ id: string; title: string; timeCreated: number }>
  readonly artifacts: Array<{
    id: string
    type: string
    title: string
    status: string
    parentID: string | null
  }>
  readonly diff: string | null
}

export interface Interface {
  readonly get: (id: TaskV2.ID) => Effect.Effect<Task | undefined>
  readonly list: (input?: { status?: string; projectID?: ProjectV2.ID }) => Effect.Effect<Task[]>
  readonly create: (input: TaskInput) => Effect.Effect<Task>
  readonly update: (id: TaskV2.ID, updates: TaskUpdate) => Effect.Effect<void>
  readonly remove: (id: TaskV2.ID) => Effect.Effect<void>
  readonly focus: (id: TaskV2.ID) => Effect.Effect<void>
  readonly focused: () => Effect.Effect<TaskV2.ID | undefined>
  readonly aggregate: (id: TaskV2.ID) => Effect.Effect<TaskAggregate>
}

export class Service extends Context.Service<Service, Interface>()("@talon/v2/Task") {}

type Row = typeof TaskTable.$inferSelect

function rowToTask(row: Row): Task {
  return {
    id: row.id,
    title: row.title,
    status: row.status,
    assignee: row.assignee,
    workspaceID: row.workspace_id,
    projectID: row.project_id,
    parentID: row.parent_id,
    orderKey: row.order_key,
    summary: row.summary,
    metadata: row.metadata,
    timeCreated: row.time_created,
    timeUpdated: row.time_updated,
    timeArchived: row.time_archived,
  }
}

export const layer = Layer.effect(
  Service,
  Effect.gen(function* () {
    const { db } = yield* Database.Service
    const location = yield* Location.Service
    const gitOpt = yield* Effect.serviceOption(Git.Service)

    const projectID = location.project.id

    const get = Effect.fn("Task.get")(function* (id: TaskV2.ID) {
      const row = yield* db.select().from(TaskTable).where(eq(TaskTable.id, id)).get().pipe(Effect.orDie)
      return row ? rowToTask(row) : undefined
    })

    const list = Effect.fn("Task.list")(function* (input?: { status?: string; projectID?: ProjectV2.ID }) {
      const where = and(
        eq(TaskTable.project_id, input?.projectID ?? projectID),
        input?.status ? eq(TaskTable.status, input.status) : undefined,
      )
      const rows = yield* db
        .select()
        .from(TaskTable)
        .where(where)
        .orderBy(asc(TaskTable.time_created))
        .all()
        .pipe(Effect.orDie)
      return rows.map(rowToTask)
    })

    const create = Effect.fn("Task.create")(function* (input: TaskInput) {
      const id = TaskV2.ID.create()
      const workspaceID = input.workspaceID ?? location.workspaceID ?? null
      const row = {
        id,
        title: input.title,
        status: input.status ?? "todo",
        assignee: input.assignee ?? null,
        workspace_id: workspaceID,
        project_id: projectID,
        parent_id: input.parentID ?? null,
        order_key: input.orderKey ?? null,
        summary: input.summary ?? null,
        metadata: null,
        time_created: Date.now(),
        time_updated: Date.now(),
        time_archived: null,
      }
      yield* db.insert(TaskTable).values(row).run().pipe(Effect.orDie)
      return rowToTask(row as unknown as Row)
    })

    const update = Effect.fn("Task.update")(function* (id: TaskV2.ID, updates: TaskUpdate) {
      const set: Record<string, unknown> = { time_updated: Date.now() }
      if (updates.title !== undefined) set.title = updates.title
      if (updates.status !== undefined) set.status = updates.status
      if (updates.assignee !== undefined) set.assignee = updates.assignee
      if (updates.orderKey !== undefined) set.order_key = updates.orderKey
      if (updates.summary !== undefined) set.summary = updates.summary
      if (updates.metadata !== undefined) set.metadata = updates.metadata
      if (updates.workspaceID !== undefined) set.workspace_id = updates.workspaceID
      yield* db.update(TaskTable).set(set).where(eq(TaskTable.id, id)).run().pipe(Effect.orDie)
    })

    const collectDescendants = (rootId: TaskV2.ID): Effect.Effect<TaskV2.ID[]> =>
      Effect.gen(function* () {
        const collected: TaskV2.ID[] = [rootId]
        let frontier: TaskV2.ID[] = [rootId]
        while (frontier.length > 0) {
          const children = yield* db
            .select({ id: TaskTable.id })
            .from(TaskTable)
            .where(inArray(TaskTable.parent_id, frontier))
            .all()
            .pipe(Effect.orDie)
          const childIds = children.map((c) => c.id) as TaskV2.ID[]
          if (childIds.length === 0) break
          collected.push(...childIds)
          frontier = childIds
        }
        return collected
      })

    const remove = Effect.fn("Task.remove")(function* (id: TaskV2.ID) {
      const ids = yield* collectDescendants(id)
      yield* db
        .transaction((tx) =>
          Effect.gen(function* () {
            // Unlink artifacts and sessions from the task before deleting
            yield* tx.update(ArtifactTable).set({ task_id: null }).where(inArray(ArtifactTable.task_id, ids)).run()
            yield* tx.update(SessionTable).set({ task_id: null }).where(inArray(SessionTable.task_id, ids)).run()
            yield* tx.delete(TaskTable).where(inArray(TaskTable.id, ids)).run()
          }),
        )
        .pipe(Effect.orDie)
    })

    const focus = Effect.fn("Task.focus")(function* (id: TaskV2.ID) {
      yield* db
        .update(AccountStateTable)
        .set({ active_task_id: id })
        .where(eq(AccountStateTable.id, 1))
        .run()
        .pipe(Effect.orDie)
    })

    const focused = Effect.fn("Task.focused")(function* () {
      const row = yield* db
        .select({ active_task_id: AccountStateTable.active_task_id })
        .from(AccountStateTable)
        .where(eq(AccountStateTable.id, 1))
        .get()
        .pipe(Effect.orDie)
      if (!row?.active_task_id) return undefined
      return TaskV2.ID.ascending(row.active_task_id)
    })

    const aggregate = Effect.fn("Task.aggregate")(function* (id: TaskV2.ID) {
      const task = yield* get(id)
      if (!task) return yield* Effect.die(new Error(`Task not found: ${id}`))

      const sessions = yield* db
        .select({ id: SessionTable.id, title: SessionTable.title, timeCreated: SessionTable.time_created })
        .from(SessionTable)
        .where(eq(SessionTable.task_id, id))
        .all()
        .pipe(Effect.orDie)

      const artifacts = yield* db
        .select({
          id: ArtifactTable.id,
          type: ArtifactTable.type,
          title: ArtifactTable.title,
          status: ArtifactTable.status,
          parentID: ArtifactTable.parent_id,
        })
        .from(ArtifactTable)
        .where(eq(ArtifactTable.task_id, id))
        .all()
        .pipe(Effect.orDie)

      // Attempt to get git diff from the location directory
      const diff: string | null = yield* Effect.gen(function* () {
        const dir = location.directory
        if (!dir) return null
        if (gitOpt._tag === "None") return null
        const diffResult = yield* gitOpt.value.patch(AbsolutePath.make(dir)).pipe(
          Effect.catch(() => Effect.succeed("")),
        )
        return diffResult || null
      }).pipe(
        Effect.catch(() => Effect.succeed(null)),
      )

      return {
        task,
        sessions: sessions.map((s) => ({ id: s.id, title: s.title, timeCreated: s.timeCreated })),
        artifacts: artifacts.map((a) => ({
          id: a.id,
          type: a.type,
          title: a.title,
          status: a.status,
          parentID: a.parentID,
        })),
        diff,
      } satisfies TaskAggregate
    })

    return Service.of({ get, list, create, update, remove, focus, focused, aggregate })
  }),
)

export const defaultLayer = layer.pipe(Layer.provide(Database.defaultLayer), Layer.provide(FSUtil.defaultLayer))
