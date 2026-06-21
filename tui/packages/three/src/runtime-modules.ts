import type { RuntimeModuleEntry } from "@tui/core/runtime-plugin"
import * as threeRuntime from "@tui/three"

export const runtimeModules = {
  "@tui/three": threeRuntime,
} satisfies Record<string, RuntimeModuleEntry>
