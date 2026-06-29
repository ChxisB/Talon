import { Effect } from "effect"
import type { DatabaseMigration } from "../migration"

export default {
  id: "20260629122805_artifact",
  up(tx) {
    return Effect.gen(function* () {
      yield* tx.run(`
        CREATE TABLE \`artifact\` (
          \`id\` text PRIMARY KEY,
          \`type\` text NOT NULL,
          \`title\` text NOT NULL,
          \`status\` text DEFAULT 'todo' NOT NULL,
          \`assignee\` text,
          \`task_id\` text,
          \`workspace_id\` text,
          \`project_id\` text NOT NULL,
          \`parent_id\` text,
          \`order_key\` text,
          \`path\` text NOT NULL,
          \`body_hash\` text,
          \`frontmatter\` text,
          \`time_created\` integer NOT NULL,
          \`time_updated\` integer NOT NULL,
          \`time_archived\` integer,
          CONSTRAINT \`fk_artifact_workspace_id_workspace_id_fk\` FOREIGN KEY (\`workspace_id\`) REFERENCES \`workspace\`(\`id\`) ON DELETE SET NULL,
          CONSTRAINT \`fk_artifact_project_id_project_id_fk\` FOREIGN KEY (\`project_id\`) REFERENCES \`project\`(\`id\`) ON DELETE CASCADE
        );
      `)
      yield* tx.run(`CREATE INDEX \`artifact_project_type_idx\` ON \`artifact\` (\`project_id\`,\`type\`);`)
      yield* tx.run(`CREATE INDEX \`artifact_task_idx\` ON \`artifact\` (\`task_id\`);`)
      yield* tx.run(`CREATE INDEX \`artifact_workspace_idx\` ON \`artifact\` (\`workspace_id\`);`)
      yield* tx.run(`CREATE INDEX \`artifact_parent_idx\` ON \`artifact\` (\`parent_id\`);`)
    })
  },
} satisfies DatabaseMigration.Migration
