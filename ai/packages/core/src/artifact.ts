export * as ArtifactV2 from "./artifact"

import { Schema } from "effect"
import { withStatics } from "./schema"
import { Identifier } from "./util/identifier"

export const ID = Schema.String.check(Schema.isStartsWith("art")).pipe(
  Schema.brand("ArtifactV2.ID"),
  withStatics((schema) => ({
    ascending: (id?: string) => {
      if (!id) return schema.make("art_" + Identifier.ascending())
      if (!id.startsWith("art")) throw new Error(`ID ${id} does not start with art`)
      return schema.make(id)
    },
    create: () => schema.make("art_" + Identifier.ascending()),
  })),
)
export type ID = typeof ID.Type
