import type { RuntimeModuleEntry, RuntimeModuleExports } from "@tui/core/runtime-plugin"
import * as keymap from "@tui/keymap"
import * as keymapExtras from "@tui/keymap/extras"
import * as keymapGraphExtra from "@tui/keymap/extras/graph"
import * as keymapAddons from "@tui/keymap/addons"
import * as keymapOpenTuiAddons from "@tui/keymap/addons/opentui"
import * as keymapHtml from "@tui/keymap/html"
import * as keymapOpenTui from "@tui/keymap/opentui"

const loadKeymapReact = async (): Promise<RuntimeModuleExports> => {
  return (await import("@tui/keymap/react")) as RuntimeModuleExports
}

const loadKeymapSolid = async (): Promise<RuntimeModuleExports> => {
  return (await import("@tui/keymap/solid")) as RuntimeModuleExports
}

export const runtimeModules = {
  "@tui/keymap": keymap,
  "@tui/keymap/extras": keymapExtras,
  "@tui/keymap/extras/graph": keymapGraphExtra,
  "@tui/keymap/addons": keymapAddons,
  "@tui/keymap/addons/opentui": keymapOpenTuiAddons,
  "@tui/keymap/html": keymapHtml,
  "@tui/keymap/opentui": keymapOpenTui,
  "@tui/keymap/react": loadKeymapReact,
  "@tui/keymap/solid": loadKeymapSolid,
} satisfies Record<string, RuntimeModuleEntry>
