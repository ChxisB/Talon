import { Effect } from "effect"
import type { DatabaseMigration } from "../migration"

export default {
  id: "20260629123200_account_state_task_id",
  up(tx) {
    return Effect.gen(function* () {
      yield* tx.run(`ALTER TABLE \`account_state\` ADD COLUMN \`active_task_id\` text;`)
    })
  },
} satisfies DatabaseMigration.Migration
