import { Effect } from "effect"
import type { DatabaseMigration } from "../migration"

export default {
  id: "20260629123000_task",
  up(tx) {
    return Effect.gen(function* () {
      yield* tx.run(`
        CREATE TABLE \`task\` (
          \`id\` text PRIMARY KEY,
          \`title\` text NOT NULL,
          \`status\` text DEFAULT 'todo' NOT NULL,
          \`assignee\` text,
          \`workspace_id\` text,
          \`project_id\` text NOT NULL,
          \`parent_id\` text,
          \`order_key\` text,
          \`summary\` text,
          \`metadata\` text,
          \`time_created\` integer NOT NULL,
          \`time_updated\` integer NOT NULL,
          \`time_archived\` integer,
          CONSTRAINT \`fk_task_project_id_project_id_fk\` FOREIGN KEY (\`project_id\`) REFERENCES \`project\`(\`id\`) ON DELETE CASCADE
        );
      `)
      yield* tx.run(`CREATE INDEX \`task_project_idx\` ON \`task\` (\`project_id\`);`)
      yield* tx.run(`CREATE INDEX \`task_workspace_idx\` ON \`task\` (\`workspace_id\`);`)
      yield* tx.run(`CREATE INDEX \`task_parent_idx\` ON \`task\` (\`parent_id\`);`)
    })
  },
} satisfies DatabaseMigration.Migration
