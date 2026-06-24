# Talon Coding Conventions

## General

- Use Bun for TypeScript/JavaScript tasks (not Node.js)
- Zig only in `native/` directory for the rendering core
- Default to `bun` over `npm`/`yarn`/`pnpm`

## TypeScript (TUI + Packages)

- Runtime: Bun with TypeScript
- JSX: `@tui/react` (set `"jsxImportSource": "@tui/react"` in tsconfig)
- Imports: Group standard library, external deps, internal modules
- Naming: camelCase for variables/functions, PascalCase for components/types
- Async: Prefer async/await, handle errors explicitly
- Formatting: Consistent indentation, no semicolons
- No `any` type — use proper types

## Zig (Native Core)

- Naming of exported FFI functions: Verb + Noun, no prefix
  - e.g., `createScreen`, `paintText`, `composeFrame`, `moveCursor`
- Internal Zig code follows original OpenTUI conventions
- Build with `zig build` from `native/` directory
- Output: `libtalon.dylib` (macOS), `libtalon.so` (Linux), `talon.dll` (Windows)

## FFI Boundary

- Zig exports a C ABI via `export fn` declarations
- TypeScript loads the library via platform-specific `dlopen`
- Structs shared across FFI are defined in both Zig (`extern struct`) and TypeScript
- Use explicit-width types (u32, u64, f32, f64) at the boundary
