import { mkdtempSync, rmSync, writeFileSync } from "node:fs"
import { tmpdir } from "node:os"
import { join } from "node:path"
import { plugin as registerPlugin } from "bun"
import * as coreRuntime from "@tui/core"
import * as reactRuntime from "react"
import * as reactJsxRuntime from "react/jsx-runtime"
import * as reactJsxDevRuntime from "react/jsx-dev-runtime"
import * as opentuiReactRuntime from "../src/index.js"

type FixtureState = typeof globalThis & {
  __reactRuntimeHost__?: {
    core: Record<string, unknown>
    coreTesting: Record<string, unknown>
    opentuiReact: Record<string, unknown>
    opentuiReactJsx: Record<string, unknown>
    opentuiReactJsxDev: Record<string, unknown>
    react: Record<string, unknown>
    reactJsx: Record<string, unknown>
    reactJsxDev: Record<string, unknown>
  }
}

const tempRoot = mkdtempSync(join(tmpdir(), "react-runtime-plugin-support-fixture-"))
const entryPath = join(tempRoot, "entry.ts")

const source = [
  'import * as core from "@tui/core"',
  'import * as coreTesting from "@tui/core/testing"',
  'import * as opentuiReact from "@tui/react"',
  'import * as opentuiReactJsx from "@tui/react/jsx-runtime"',
  'import * as opentuiReactJsxDev from "@tui/react/jsx-dev-runtime"',
  'import * as react from "react"',
  'import * as reactJsx from "react/jsx-runtime"',
  'import * as reactJsxDev from "react/jsx-dev-runtime"',
  "const state = globalThis as { __reactRuntimeHost__?: { core: Record<string, unknown>; coreTesting: Record<string, unknown>; opentuiReact: Record<string, unknown>; opentuiReactJsx: Record<string, unknown>; opentuiReactJsxDev: Record<string, unknown>; react: Record<string, unknown>; reactJsx: Record<string, unknown>; reactJsxDev: Record<string, unknown> } }",
  "const host = state.__reactRuntimeHost__",
  "const checks = [",
  "  `core=${core.engine === host?.core.engine}`,",
  "  `coreTesting=${coreTesting.createTestRenderer === host?.coreTesting.createTestRenderer}`,",
  "  `opentuiReact=${opentuiReact.render === host?.opentuiReact.render}`,",
  "  `opentuiReactJsx=${opentuiReactJsx.jsx === host?.opentuiReactJsx.jsx}`,",
  "  `opentuiReactJsxDev=${opentuiReactJsxDev.jsxDEV === host?.opentuiReactJsxDev.jsxDEV}`,",
  "  `react=${react.useState === host?.react.useState}`,",
  "  `reactJsx=${reactJsx.jsx === host?.reactJsx.jsx}`,",
  "  `reactJsxDev=${reactJsxDev.jsxDEV === host?.reactJsxDev.jsxDEV}`,",
  "]",
  "console.log(checks.join(';'))",
  "export const noop = 1",
].join("\n")

writeFileSync(entryPath, source)

const state = globalThis as FixtureState
state.__reactRuntimeHost__ = {
  core: coreRuntime as Record<string, unknown>,
  coreTesting: (await import("@tui/core/testing")) as Record<string, unknown>,
  opentuiReact: opentuiReactRuntime as Record<string, unknown>,
  opentuiReactJsx: (await import("../jsx-runtime.js")) as Record<string, unknown>,
  opentuiReactJsxDev: (await import("../jsx-dev-runtime.js")) as Record<string, unknown>,
  react: reactRuntime as Record<string, unknown>,
  reactJsx: reactJsxRuntime as Record<string, unknown>,
  reactJsxDev: reactJsxDevRuntime as Record<string, unknown>,
}

registerPlugin.clearAll()

try {
  await import("../scripts/runtime-plugin-support.js")
  await import(entryPath)
} finally {
  registerPlugin.clearAll()
  delete state.__reactRuntimeHost__
  rmSync(tempRoot, { recursive: true, force: true })
}
