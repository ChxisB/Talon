# Talon

A terminal-based AI coding assistant built with TypeScript/Effect, the OpenTUI terminal rendering framework, and a native Zig rendering core.

## Stack

- **Application**: TypeScript/Effect (via `ai/packages/talon`)
- **TUI Rendering**: OpenTUI (`@tui/react`, `@tui/core`) forked in `tui/`
- **Native Rendering**: Zig core (forked from OpenTUI, `libopentui.dylib`)
- **HTTP API**: Effect HttpApi (via `@talon-ai/server` library package)

## Project Structure

```
talon/
├── ai/               # Application monorepo (main Talon app)
│   └── packages/
│       ├── talon/    # Main Talon application
│       ├── server/   # HTTP API library (Effect HttpApi)
│       ├── core/     # Core data layer
│       └── cli/      # CLI commands
├── tui/              # OpenTUI terminal rendering framework (fork)
├── native/           # Zig native rendering core
├── scripts/          # Build and install scripts
└── .claude/          # Project configuration
```

## Key Decisions

- Zig native core is forked from OpenTUI and rebranded with talon-specific naming
- Bun is the runtime for the TUI (OpenTUI requires Bun >= 1.3.0)
- The application embed its own HTTP server via Effect HttpApi (no separate Go backend)
- The React reconciler (`@tui/react`) is used as-is from upstream
