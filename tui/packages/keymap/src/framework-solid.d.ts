// Declaration-build shim: tsconfig.build maps @tui/solid here so keymap
// can emit d.ts for its Solid entrypoint without importing framework sources.
import type { CliRenderer } from "@tui/core"

export function useRenderer(): CliRenderer
