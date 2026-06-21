import { plugin as registerPlugin } from "bun"
import type { RuntimeModuleEntry } from "@tui/core/runtime-plugin"
import * as keymapRuntime from "@tui/keymap"
import * as keymapAddonsRuntime from "@tui/keymap/addons"
import * as keymapExtrasRuntime from "@tui/keymap/extras"
import * as keymapSolidRuntime from "@tui/keymap/solid"
import { ensureRuntimePluginSupport } from "@tui/solid/runtime-plugin-support/configure"
import * as threeRuntime from "../../three/src/index.js"
import { resetSolidTransformPluginState } from "../scripts/solid-plugin.js"

type FixtureState = typeof globalThis & {
  __solidRuntimeHost__?: {
    keymap: Record<string, unknown>
    keymapAddons: Record<string, unknown>
    keymapExtras: Record<string, unknown>
    keymapSolid: Record<string, unknown>
    three: Record<string, unknown>
  }
}

const state = globalThis as FixtureState
state.__solidRuntimeHost__ = {
  keymap: keymapRuntime as Record<string, unknown>,
  keymapAddons: keymapAddonsRuntime as Record<string, unknown>,
  keymapExtras: keymapExtrasRuntime as Record<string, unknown>,
  keymapSolid: keymapSolidRuntime as Record<string, unknown>,
  three: threeRuntime as Record<string, unknown>,
}

registerPlugin.clearAll()
resetSolidTransformPluginState()

try {
  const additional = {
    "@tui/keymap": keymapRuntime,
    "@tui/keymap/addons": keymapAddonsRuntime,
    "@tui/keymap/extras": keymapExtrasRuntime,
    "@tui/keymap/solid": keymapSolidRuntime,
    "@tui/three": threeRuntime,
  } satisfies Record<string, RuntimeModuleEntry>
  const first = ensureRuntimePluginSupport({ additional })
  const second = ensureRuntimePluginSupport({ additional })
  console.log(`first=${first};second=${second}`)
  await import("./runtime-plugin-support-configure-entry.fixture.tsx")
} finally {
  registerPlugin.clearAll()
  delete state.__solidRuntimeHost__
}
