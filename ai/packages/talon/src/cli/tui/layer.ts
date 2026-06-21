import { run as runTui, type TuiInput } from "@talon-ai/tui"
import { Global } from "@talon-ai/core/global"
import { Effect } from "effect"

export function run(input: TuiInput) {
  return runTui(input).pipe(Effect.provide(Global.defaultLayer))
}
