import { sqliteTable, text, integer, index } from "drizzle-orm/sqlite-core"
import { ProjectTable } from "../project/sql"
import { ProjectV2 } from "../project"
import { WorkspaceTable } from "../control-plane/workspace.sql"
import { WorkspaceV2 } from "../workspace"
import { ArtifactV2 } from "../artifact"
import { Timestamps } from "../database/schema.sql"

export const ArtifactTable = sqliteTable(
  "artifact",
  {
    id: text().$type<ArtifactV2.ID>().primaryKey(),
    // "spec" | "ticket" | "story" | "review" | "plan" | "note"
    type: text().notNull(),
    title: text().notNull(),
    // "todo" | "in_progress" | "done" | "blocked" | "archived"
    // (todo/in_progress/done apply to ticket+story; spec/review use "draft"/"approved"/"archived")
    status: text().notNull().default("todo"),
    assignee: text(),
    // FK -> task.id is deferred to Feature D (task table does not exist yet); plain text for now.
    task_id: text(),
    workspace_id: text()
      .$type<WorkspaceV2.ID>()
      .references(() => WorkspaceTable.id, { onDelete: "set null" }),
    project_id: text()
      .$type<ProjectV2.ID>()
      .notNull()
      .references(() => ProjectTable.id, { onDelete: "cascade" }),
    // Self-referential hierarchy. Deliberately NOT a Drizzle `.references()` FK
    // (matches SessionTable.parent_id convention) to avoid a circular type
    // inference error. Cascade-delete of children is handled in the service layer.
    parent_id: text().$type<ArtifactV2.ID>(),
    order_key: text(),
    // Absolute path to the .md file on disk (source of truth for body content).
    path: text().notNull(),
    // sha1 of file body at last index (drift detection for reindex).
    body_hash: text(),
    frontmatter: text({ mode: "json" }).$type<Record<string, unknown>>(),
    ...Timestamps,
    time_archived: integer(),
  },
  (table) => [
    index("artifact_project_type_idx").on(table.project_id, table.type),
    index("artifact_task_idx").on(table.task_id),
    index("artifact_workspace_idx").on(table.workspace_id),
    index("artifact_parent_idx").on(table.parent_id),
  ],
)
