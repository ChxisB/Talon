import { sqliteTable, text, integer, index } from "drizzle-orm/sqlite-core"
import { ProjectTable } from "../project/sql"
import { ProjectV2 } from "../project"
import { TaskV2 } from "../task"
import { Timestamps } from "../database/schema.sql"

export const TaskTable = sqliteTable(
  "task",
  {
    id: text().$type<TaskV2.ID>().primaryKey(),
    title: text().notNull(),
    // "todo" | "in_progress" | "done" | "blocked" | "archived"
    status: text().notNull().default("todo"),
    assignee: text(),
    // FK -> workspace.id is deferred; plain text for now.
    workspace_id: text(),
    project_id: text()
      .$type<ProjectV2.ID>()
      .notNull()
      .references(() => ProjectTable.id, { onDelete: "cascade" }),
    // Self-referential hierarchy. Deliberately NOT a Drizzle `.references()` FK
    // (matches SessionTable.parent_id and ArtifactTable.parent_id convention)
    // to avoid a circular type inference error. Cascade-delete of children is
    // handled in the service layer.
    parent_id: text().$type<TaskV2.ID>(),
    order_key: text(),
    summary: text(),
    metadata: text({ mode: "json" }).$type<Record<string, unknown>>(),
    ...Timestamps,
    time_archived: integer(),
  },
  (table) => [
    index("task_project_idx").on(table.project_id),
    index("task_workspace_idx").on(table.workspace_id),
    index("task_parent_idx").on(table.parent_id),
  ],
)
