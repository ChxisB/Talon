import { and, asc, eq, inArray } from "drizzle-orm"
import { Context, Effect, Layer } from "effect"
import path from "path"
import type { EffectDrizzleSqlite } from "@talon-ai/effect-drizzle-sqlite"
import { Database } from "../database/database"
import { Global } from "../global"
import { Location } from "../location"
import { FSUtil } from "../fs-util"
import { Hash } from "../util/hash"
import { ProjectV2 } from "../project"
import { WorkspaceV2 } from "../workspace"
import { ArtifactV2, ID } from "../artifact"
import { ArtifactTable } from "./sql"
import { parseFrontmatter, serializeFrontmatter, type Frontmatter } from "./frontmatter"

type DatabaseClient = EffectDrizzleSqlite.EffectSQLiteDatabase
/** A drizzle transaction handle, so callers can run reconcile ops in one tx. */
export type Transaction = Parameters<Parameters<DatabaseClient["transaction"]>[0]>[0]

export type ArtifactType = "spec" | "ticket" | "story" | "review" | "plan" | "note"
export type ArtifactStatus = "todo" | "in_progress" | "done" | "blocked" | "archived"

export type Artifact = {
  readonly id: ArtifactV2.ID
  readonly type: string
  readonly title: string
  readonly status: string
  readonly assignee: string | null
  readonly taskID: string | null
  readonly workspaceID: WorkspaceV2.ID | null
  readonly projectID: ProjectV2.ID
  readonly parentID: ArtifactV2.ID | null
  readonly orderKey: string | null
  readonly path: string
  readonly bodyHash: string | null
  readonly frontmatter: Record<string, unknown> | null
  readonly timeCreated: number
  readonly timeUpdated: number
  readonly timeArchived: number | null
}

export type ArtifactInput = {
  readonly type: ArtifactType
  readonly title: string
  readonly body?: string
  readonly parentID?: ArtifactV2.ID
  readonly taskID?: string
  readonly assignee?: string
  readonly orderKey?: string
  readonly status?: ArtifactStatus
}

export type ArtifactUpdate = Partial<{
  title: string
  status: ArtifactStatus
  assignee: string
  orderKey: string
  taskID: string
  body: string
}>

export type MoveTarget = Partial<{
  parentID: ArtifactV2.ID | null
  taskID: string | null
  orderKey: string | null
}>

export interface Interface {
  /** Read one artifact by id (DB only; does not touch disk). */
  readonly get: (id: ArtifactV2.ID) => Effect.Effect<Artifact | undefined>
  /** List artifacts for the current project, optionally filtered. */
  readonly list: (input?: { type?: string; status?: string; taskID?: string }) => Effect.Effect<Artifact[]>
  /** Create an artifact: writes the markdown file + DB index row. */
  readonly create: (input: ArtifactInput) => Effect.Effect<Artifact>
  /** Update an artifact's title/status/assignee/orderKey/taskID/body (disk + DB). */
  readonly update: (id: ArtifactV2.ID, updates: ArtifactUpdate) => Effect.Effect<void>
  /** Move an artifact under a new parent/task/order (DB only; file path is stable). */
  readonly move: (id: ArtifactV2.ID, target: MoveTarget) => Effect.Effect<void>
  /** Delete an artifact and its descendants (DB rows + files). */
  readonly remove: (id: ArtifactV2.ID) => Effect.Effect<void>
  /** Reconcile the DB index with on-disk markdown files for the current project. */
  readonly reindex: () => Effect.Effect<{ inserted: number; updated: number; removed: number }>
}

export class Service extends Context.Service<Service, Interface>()("@talon/v2/Artifact") {}

type Row = typeof ArtifactTable.$inferSelect

function rowToArtifact(row: Row): Artifact {
  return {
    id: row.id,
    type: row.type,
    title: row.title,
    status: row.status,
    assignee: row.assignee,
    taskID: row.task_id,
    workspaceID: row.workspace_id,
    projectID: row.project_id,
    parentID: row.parent_id,
    orderKey: row.order_key,
    path: row.path,
    bodyHash: row.body_hash,
    frontmatter: row.frontmatter,
    timeCreated: row.time_created,
    timeUpdated: row.time_updated,
    timeArchived: row.time_archived,
  }
}

const ARTIFACT_TYPES: readonly ArtifactType[] = ["spec", "ticket", "story", "review", "plan", "note"]

function isArtifactID(value: unknown): value is string {
  return typeof value === "string" && value.startsWith("art_")
}

export const layer = Layer.effect(
  Service,
  Effect.gen(function* () {
    const { db } = yield* Database.Service
    const location = yield* Location.Service
    const fs = yield* FSUtil.Service

    const projectID = location.project.id
    const workspaceID = location.workspaceID ?? null

    const baseDir = (type: string): string =>
      location.vcs
        ? path.join(location.project.directory, ".talon", "artifacts", type)
        : path.join(Global.Path.data, "artifacts", type)

    const filePath = (type: string, id: ArtifactV2.ID): string => path.join(baseDir(type), `${id}.md`)

    const writeAtomic = (filepath: string, content: string) =>
      Effect.gen(function* () {
        const tempfile = `${filepath}.${process.pid}.${Date.now()}.tmp`
        yield* fs
          .writeWithDirs(tempfile, content)
          .pipe(
            Effect.andThen(fs.rename(tempfile, filepath)),
            Effect.catch((error) =>
              Effect.gen(function* () {
                yield* fs.remove(tempfile, { force: true }).pipe(Effect.ignore)
                return yield* Effect.fail(error)
              }),
            ),
          )
      })

    const get = Effect.fn("Artifact.get")(function* (id: ArtifactV2.ID) {
      const row = yield* db.select().from(ArtifactTable).where(eq(ArtifactTable.id, id)).get().pipe(Effect.orDie)
      return row ? rowToArtifact(row) : undefined
    })

    const list = Effect.fn("Artifact.list")(function* (input?: {
      type?: string
      status?: string
      taskID?: string
    }) {
      const where = and(
        eq(ArtifactTable.project_id, projectID),
        input?.type ? eq(ArtifactTable.type, input.type) : undefined,
        input?.status ? eq(ArtifactTable.status, input.status) : undefined,
        input?.taskID ? eq(ArtifactTable.task_id, input.taskID) : undefined,
      )
      const rows = yield* db
        .select()
        .from(ArtifactTable)
        .where(where)
        .orderBy(asc(ArtifactTable.time_created))
        .all()
        .pipe(Effect.orDie)
      return rows.map(rowToArtifact)
    })

    const create = Effect.fn("Artifact.create")(function* (input: ArtifactInput) {
      const id = ArtifactV2.ID.create()
      const status = input.status ?? "todo"
      const body = input.body ?? ""
      const frontmatter: Frontmatter = { id, type: input.type, title: input.title, status }
      if (input.assignee) frontmatter.assignee = input.assignee
      if (input.parentID) frontmatter.parent = input.parentID
      if (input.taskID) frontmatter.task = input.taskID
      if (input.orderKey) frontmatter.order = input.orderKey
      const filepath = filePath(input.type, id)
      yield* writeAtomic(filepath, serializeFrontmatter(frontmatter, body)).pipe(Effect.orDie)
      const row = {
        id,
        type: input.type,
        title: input.title,
        status,
        assignee: input.assignee ?? null,
        task_id: input.taskID ?? null,
        workspace_id: workspaceID,
        project_id: projectID,
        parent_id: input.parentID ?? null,
        order_key: input.orderKey ?? null,
        path: filepath,
        body_hash: Hash.fast(body),
        frontmatter,
        time_created: Date.now(),
        time_updated: Date.now(),
        time_archived: null,
      }
      yield* db.insert(ArtifactTable).values(row).run().pipe(Effect.orDie)
      return rowToArtifact(row as unknown as Row)
    })

    const update = Effect.fn("Artifact.update")(function* (id: ArtifactV2.ID, updates: ArtifactUpdate) {
      const existing = yield* db.select().from(ArtifactTable).where(eq(ArtifactTable.id, id)).get().pipe(Effect.orDie)
      if (!existing) return
      // Rebuild the file from its current on-disk state + the applied updates so
      // disk and DB stay in sync. If the file is missing, self-heal from the row.
      const disk = yield* fs.readFileStringSafe(existing.path).pipe(Effect.orDie)
      const parsed = disk !== undefined ? parseFrontmatter(disk) : { frontmatter: {}, body: "" }
      const frontmatter: Frontmatter = { ...parsed.frontmatter }
      frontmatter.id = id
      frontmatter.type = existing.type
      if (updates.title !== undefined) frontmatter.title = updates.title
      else if (existing.title) frontmatter.title = existing.title
      if (updates.status !== undefined) frontmatter.status = updates.status
      else frontmatter.status = existing.status
      if (updates.assignee !== undefined) frontmatter.assignee = updates.assignee
      if (updates.orderKey !== undefined) frontmatter.order = updates.orderKey
      if (updates.taskID !== undefined) frontmatter.task = updates.taskID
      if (existing.parent_id) frontmatter.parent = existing.parent_id
      const body = updates.body !== undefined ? updates.body : parsed.body
      yield* writeAtomic(existing.path, serializeFrontmatter(frontmatter, body)).pipe(Effect.orDie)
      const set: Record<string, unknown> = { time_updated: Date.now() }
      if (updates.title !== undefined) set.title = updates.title
      if (updates.status !== undefined) set.status = updates.status
      if (updates.assignee !== undefined) set.assignee = updates.assignee
      if (updates.orderKey !== undefined) set.order_key = updates.orderKey
      if (updates.taskID !== undefined) set.task_id = updates.taskID
      set.body_hash = Hash.fast(body)
      set.frontmatter = frontmatter
      yield* db.update(ArtifactTable).set(set).where(eq(ArtifactTable.id, id)).run().pipe(Effect.orDie)
    })

    const move = Effect.fn("Artifact.move")(function* (id: ArtifactV2.ID, target: MoveTarget) {
      const set: Record<string, unknown> = { time_updated: Date.now() }
      if (target.parentID !== undefined) set.parent_id = target.parentID
      if (target.taskID !== undefined) set.task_id = target.taskID
      if (target.orderKey !== undefined) set.order_key = target.orderKey
      yield* db.update(ArtifactTable).set(set).where(eq(ArtifactTable.id, id)).run().pipe(Effect.orDie)
    })

    const collectDescendants = (rootId: ArtifactV2.ID): Effect.Effect<ArtifactV2.ID[]> =>
      Effect.gen(function* () {
        const collected: ArtifactV2.ID[] = [rootId]
        let frontier: ArtifactV2.ID[] = [rootId]
        while (frontier.length > 0) {
          const children = yield* db
            .select({ id: ArtifactTable.id })
            .from(ArtifactTable)
            .where(inArray(ArtifactTable.parent_id, frontier))
            .all()
            .pipe(Effect.orDie)
          const childIds = children.map((c) => c.id)
          if (childIds.length === 0) break
          collected.push(...childIds)
          frontier = childIds
        }
        return collected
      })

    const remove = Effect.fn("Artifact.remove")(function* (id: ArtifactV2.ID) {
      const ids = yield* collectDescendants(id)
      const rows = yield* db
        .select({ id: ArtifactTable.id, path: ArtifactTable.path })
        .from(ArtifactTable)
        .where(inArray(ArtifactTable.id, ids))
        .all()
        .pipe(Effect.orDie)
      yield* db
        .transaction((tx) => Effect.gen(function* () {
          yield* tx.delete(ArtifactTable).where(inArray(ArtifactTable.id, ids)).run()
        }))
        .pipe(Effect.orDie)
      for (const row of rows) {
        yield* fs.remove(row.path, { force: true }).pipe(Effect.ignore)
      }
    })

    const reindex = Effect.fn("Artifact.reindex")(function* () {
      const seen = new Set<ArtifactV2.ID>()
      let inserted = 0
      let updated = 0
      for (const type of ARTIFACT_TYPES) {
        const dir = baseDir(type)
        const exists = yield* fs.existsSafe(dir)
        if (!exists) continue
        const entries = yield* fs.readDirectoryEntries(dir).pipe(Effect.orDie)
        for (const entry of entries) {
          if (entry.type !== "file" || !entry.name.endsWith(".md")) continue
          const filepath = path.join(dir, entry.name)
          const content = yield* fs.readFileStringSafe(filepath).pipe(Effect.orDie)
          if (content === undefined) continue
          const { frontmatter, body } = parseFrontmatter(content)
          if (!isArtifactID(frontmatter.id)) continue
          const id = ID.make(frontmatter.id) as ArtifactV2.ID
          seen.add(id)
          const bodyHash = Hash.fast(body)
          const existing = yield* db.select().from(ArtifactTable).where(eq(ArtifactTable.id, id)).get().pipe(Effect.orDie)
          const title = typeof frontmatter.title === "string" ? frontmatter.title : entry.name.replace(/\.md$/, "")
          const status = typeof frontmatter.status === "string" ? frontmatter.status : "todo"
          const parent = isArtifactID(frontmatter.parent) ? (ID.make(frontmatter.parent) as ArtifactV2.ID) : null
          if (!existing) {
            yield* db
              .insert(ArtifactTable)
              .values({
                id,
                type,
                title,
                status,
                assignee: typeof frontmatter.assignee === "string" ? frontmatter.assignee : null,
                task_id: typeof frontmatter.task === "string" ? frontmatter.task : null,
                workspace_id: workspaceID,
                project_id: projectID,
                parent_id: parent,
                order_key: typeof frontmatter.order === "string" ? frontmatter.order : null,
                path: filepath,
                body_hash: bodyHash,
                frontmatter,
                time_created: Date.now(),
                time_updated: Date.now(),
                time_archived: null,
              })
              .run()
              .pipe(Effect.orDie)
            inserted++
          } else if (existing.body_hash !== bodyHash || existing.path !== filepath) {
            yield* db
              .update(ArtifactTable)
              .set({ title, status, body_hash: bodyHash, path: filepath, frontmatter, time_updated: Date.now() })
              .where(eq(ArtifactTable.id, id))
              .run()
              .pipe(Effect.orDie)
            updated++
          }
        }
      }
      // Remove DB rows whose backing file no longer exists for this project.
      const allRows = yield* db
        .select({ id: ArtifactTable.id })
        .from(ArtifactTable)
        .where(eq(ArtifactTable.project_id, projectID))
        .all()
        .pipe(Effect.orDie)
      const missing = allRows.filter((r) => !seen.has(r.id))
      let removed = 0
      if (missing.length > 0) {
        yield* db
          .delete(ArtifactTable)
          .where(inArray(ArtifactTable.id, missing.map((r) => r.id)))
          .run()
          .pipe(Effect.orDie)
        removed = missing.length
      }
      return { inserted, updated, removed }
    })

    return Service.of({ get, list, create, update, move, remove, reindex })
  }),
)

export const defaultLayer = layer.pipe(Layer.provide(Database.defaultLayer), Layer.provide(FSUtil.defaultLayer))
