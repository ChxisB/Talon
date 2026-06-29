import { Effect } from "effect"
import type { DatabaseMigration } from "../migration"

export default {
  id: "20260629123100_session_task_id",
  up(tx) {
    return Effect.gen(function* () {
      yield* tx.run(`ALTER TABLE \`session\` ADD COLUMN \`task_id\` text;`)
      yield* tx.run(`CREATE INDEX \`session_task_idx\` ON \`session\` (\`task_id\`);`)
    })
  },
} satisfies DatabaseMigration.Migration
