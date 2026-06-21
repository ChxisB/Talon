# Talon

A terminal UI application built with React + OpenTUI on Bun, a Go backend, and a native Zig rendering core.

## Stack

- **TUI Frontend**: React (via `@tui/react`) running on Bun
- **Backend**: Go HTTP/WebSocket server
- **Native Rendering**: Zig core (forked from OpenTUI, rebranded as `libtalon`)

## Project Structure

```
talon/
├── tui/              # React TUI application
├── backend/          # Go API server
├── native/           # Zig native rendering core
├── packages/core/    # TypeScript FFI bindings to native core
└── .claude/          # Project configuration
```

## Key Decisions

- Zig native core is forked from OpenTUI and rebranded with talon-specific naming
- Bun is the runtime for the TUI (OpenTUI requires Bun >= 1.3.0)
- Go backend communicates with TUI via HTTP/WebSocket
- The React reconciler (`@tui/react`) is used as-is from upstream
