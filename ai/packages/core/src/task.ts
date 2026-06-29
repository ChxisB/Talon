export * as TaskV2 from "./task"

import { Schema } from "effect"
import { withStatics } from "./schema"
import { Identifier } from "./util/identifier"

export const ID = Schema.String.check(Schema.isStartsWith("tsk")).pipe(
  Schema.brand("TaskV2.ID"),
  withStatics((schema) => ({
    ascending: (id?: string) => {
      if (!id) return schema.make("tsk_" + Identifier.ascending())
      if (!id.startsWith("tsk")) throw new Error(`ID ${id} does not start with tsk`)
      return schema.make(id)
    },
    create: () => schema.make("tsk_" + Identifier.ascending()),
  })),
)
export type ID = typeof ID.Type
