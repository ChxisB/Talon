# Talon Architecture

## System Topology

```
┌──────────────────────────────────────────────────────┐
│  Terminal                                              │
│  ┌─────────────────────┐    ┌──────────────────────┐  │
│  │  TUI (Bun runtime)   │    │  Go Backend          │  │
│  │                      │    │                       │  │
│  │  React App           │    │  HTTP/WS Server       │  │
│  │    │                 │    │  Port 8090            │  │
│  │  @tui/react      │    │                       │  │
│  │    │                 │◄──►│  /health              │  │
│  │  @tui/core (TS)    │    │  /api/v1/...          │  │
│  │    │                 │    │                       │  │
│  │  libtalon.dylib (Zig)│    │  Go routines          │  │
│  └─────────────────────┘    └──────────────────────┘  │
└──────────────────────────────────────────────────────┘
```

## Data Flow

1. TUI starts → Bun runs React app → `@tui/react` reconciler renders components
2. React components call into `@tui/core` TypeScript API
3. `@tui/core` loads `libtalon.dylib` (Zig) via FFI for terminal rendering + input
4. TUI fetches data from Go backend via HTTP at `localhost:8090`
5. Go backend processes requests, returns JSON
6. React components re-render with new data

## Directory Layout

- `tui/src/index.ts` — TUI entry point, creates renderer and root
- `tui/src/App.tsx` — Main application component
- `tui/src/api/client.ts` — HTTP client for Go backend
- `backend/cmd/server/main.go` — Go server entry point
- `backend/internal/handler/` — HTTP handlers
- `native/src/lib.zig` — Zig FFI exports (rebranded)
- `packages/core/src/ffi.ts` — TypeScript FFI bindings to libtalon
