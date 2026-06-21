// Declaration-build shim: tsconfig.build maps @tui/react here so keymap
// can emit d.ts for its React entrypoint without importing framework sources.
import type { CliRenderer } from "@tui/core"

export function useRenderer(): CliRenderer
