# Talon Architecture

## System Topology

```
┌──────────────────────────────────────────┐
│  Terminal                                │
│  ┌────────────────────────────────────┐  │
│  │  Talon Application (Bun runtime)    │  │
│  │  ┌────────────────────────────┐    │  │
│  │  │  ai/packages/talon          │    │  │
│  │  │  ┌──────────────────────┐  │    │  │
│  │  │  │  Effect HttpApi       │──│───│──│── HTTP API
│  │  │  │  (built-in server)    │  │    │  │
│  │  │  ├──────────────────────┤  │    │  │
│  │  │  │  Solid/TUI components │  │    │  │
│  │  │  │  (@tui/solid)         │  │    │  │
│  │  │  ├──────────────────────┤  │    │  │
│  │  │  │  @tui/core (TS)      │  │    │  │
│  │  │  ├──────────────────────┤  │    │  │
│  │  │  │  libopentui.dylib    │  │    │  │
│  │  │  │  (Zig renderer)      │  │    │  │
│  │  │  └──────────────────────┘  │    │  │
│  │  └────────────────────────────┘    │  │
│  └────────────────────────────────────┘  │
└──────────────────────────────────────────┘
```

## Data Flow

1. Talon starts via `ai/packages/talon/src/index.ts`
2. Application renders TUI via OpenTUI (`@tui/react`/`@tui/core`) which loads `libopentui.dylib` (Zig) via FFI
3. Built-in HTTP server (Effect HttpApi via `@talon-ai/server`) handles API requests
4. LLM providers, sessions, tools, filesystem access all handled within the application process

## Directory Layout

- `ai/packages/talon/src/index.ts` — Application entry point
- `ai/packages/server/` — HTTP API library (reusable handlers/routes)
- `ai/packages/core/` — Core data models and database layer
- `tui/packages/core/src/zig/lib.zig` — Zig FFI exports (rendering core)
- `tui/packages/core/src/ffi.ts` — TypeScript FFI bindings to libopentui
