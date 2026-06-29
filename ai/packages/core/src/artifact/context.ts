/**
 * SystemContext source `talon/artifacts` — injects the project's artifact index
 * (specs, tickets, stories, reviews, plans) into the agent's system prompt.
 *
 * Mirrors `InstructionContext` (`instruction-context.ts`): same
 * `registry.register({ key, load })` pattern, same `SystemContext.make` source
 * shape with `baseline`/`update`/`removed` renderers, same `unavailable` +
 * `empty` handling. The only difference is the value type: `ArtifactSummary[]`
 * instead of `File[]`, and the `load` reads from the `ArtifactV2.Service` DB
 * query + on-disk bodies for in-progress artifacts.
 *
 * Token discipline: the baseline renders a compact one-line index for ALL
 * artifacts, and full bodies only for in_progress artifacts (the ones the agent
 * is actively working on). The update renderer emits only the delta (added,
 * changed, removed) so context refreshes don't re-dump the whole index every
 * turn. When there are zero artifacts, the source returns `SystemContext.empty`
 * (no artifact context injected at all — same as InstructionContext with no
 * AGENTS.md files).
 */

export * as ArtifactContext from "./context"

import { Effect, Layer, Schema } from "effect"
import { SystemContext } from "../system-context/index"
import { SystemContextRegistry } from "../system-context/registry"
import { FSUtil } from "../fs-util"
import { Service as ArtifactService, defaultLayer as artifactServiceDefaultLayer } from "./index"

const ArtifactSummary = Schema.Struct({
  id: Schema.String,
  type: Schema.String,
  title: Schema.String,
  status: Schema.String,
  assignee: Schema.optional(Schema.String),
  path: Schema.String,
  body: Schema.optional(Schema.String),
})
type ArtifactSummary = typeof ArtifactSummary.Type

const Summaries = Schema.Array(ArtifactSummary)
const key = SystemContext.Key.make("talon/artifacts")

export const layer = Layer.effectDiscard(
  Effect.gen(function* () {
    const service = yield* ArtifactService
    const fs = yield* FSUtil.Service
    const registry = yield* SystemContextRegistry.Service

    const source = (value: ReadonlyArray<ArtifactSummary> | SystemContext.Unavailable) =>
      SystemContext.make({
        key,
        codec: Schema.toCodecJson(Summaries),
        load: Effect.succeed(value),
        baseline: renderBaseline,
        update: (_previous, current) =>
          `Artifacts have been updated.\n\n${renderBaseline(current)}`,
        removed: () => "Previously loaded artifacts no longer apply.",
      })

    const observe = Effect.fn("ArtifactContext.observe")(function* () {
      const artifacts = yield* service.list()
      if (artifacts.length === 0) return [] as ArtifactSummary[]
      const summaries = yield* Effect.forEach(
        artifacts,
        (a) =>
          a.status === "in_progress"
            ? fs.readFileStringSafe(a.path).pipe(
                Effect.orDie,
                Effect.map((body) => ({
                  id: a.id,
                  type: a.type,
                  title: a.title,
                  status: a.status,
                  assignee: a.assignee ?? undefined,
                  path: a.path,
                  body: body ?? undefined,
                })),
              )
            : Effect.succeed({
                id: a.id,
                type: a.type,
                title: a.title,
                status: a.status,
                assignee: a.assignee ?? undefined,
                path: a.path,
              }),
        { concurrency: "unbounded" },
      )
      return summaries
    })

    yield* registry.register({
      key,
      load: observe().pipe(
        Effect.map((summaries) =>
          summaries.length === 0 ? SystemContext.empty : source(summaries),
        ),
        Effect.catch(() => Effect.succeed(source(SystemContext.unavailable))),
        Effect.catchDefect(() => Effect.succeed(source(SystemContext.unavailable))),
      ),
    })
  }),
)

function renderBaseline(summaries: ReadonlyArray<ArtifactSummary>): string {
  const lines: string[] = ["Active artifacts for this project:"]
  const grouped = new Map<string, ArtifactSummary[]>()
  for (const s of summaries) {
    const list = grouped.get(s.type) ?? []
    list.push(s)
    grouped.set(s.type, list)
  }
  const types = ["spec", "ticket", "story", "review", "plan", "note"]
  const ordered = [...grouped.keys()].sort((a, b) => types.indexOf(a) - types.indexOf(b))
  for (const type of ordered) {
    const items = grouped.get(type)!
    lines.push(`\n${type}:`)
    for (const item of items) {
      const assignee = item.assignee ? ` @${item.assignee}` : ""
      lines.push(`  [${item.status}] ${item.title} — ${item.id}${assignee}`)
      if (item.body) {
        const indented = item.body
          .split("\n")
          .map((l) => `    ${l}`)
          .join("\n")
        lines.push(indented)
      }
    }
  }
  return lines.join("\n")
}

/**
 * Self-contained layer that bundles its own `ArtifactService` (with `Database`
 * + `FSUtil` deps). Use this in `SystemContextBuiltIns` so the builtins layer
 * doesn't gain a `Database.Service` requirement. `ArtifactService.Service` is
 * in the output (via `provideMerge`) so downstream consumers (tools, CLI) can
 * access it.
 */
export const withArtifactService = layer.pipe(
  Layer.provideMerge(artifactServiceDefaultLayer),
)
